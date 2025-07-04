package diskq

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// logging stuff copied from github.com/nsqio/nsq/internal/lg

type LogLevel int

const (
	DEBUG = LogLevel(1)
	INFO  = LogLevel(2)
	WARN  = LogLevel(3)
	ERROR = LogLevel(4)
	FATAL = LogLevel(5)
)

type AppLogFunc func(lvl LogLevel, f string, args ...interface{})

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARNING"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	}
	panic("invalid LogLevel")
}

type Interface interface {
	Put([]byte) error
	ReadChan() <-chan []byte // this is expected to be an *unbuffered* channel
	PeekChan() <-chan []byte // this is expected to be an *unbuffered* channel
	Close() error
	Delete() error
	Depth() int64
	Empty() error
}

// DiskQueue implements a filesystem backed FIFO queue
type DiskQueue struct {
	// 64bit atomic vars need to be first for proper alignment on 32bit platforms

	// run-time state (also persisted to disk)
	readPos      int64
	writePos     int64
	readFileNum  int64
	writeFileNum int64
	depth        int64

	sync.RWMutex

	// instantiation time metadata
	name                string
	dataDIR             string
	maxBytesPerFile     int64 // cannot change once created
	maxBytesPerFileRead int64
	minMsgSize          int32
	maxMsgSize          int32
	syncEvery           int64         // number of writes per fsync
	syncTimeout         time.Duration // duration of time per fsync
	exitFlag            int32
	needSync            bool

	// keeps track of the position where we have read
	// (but not yet sent over readChan)
	nextReadPos     int64
	nextReadFileNum int64

	readFile  *os.File
	writeFile *os.File
	reader    *bufio.Reader
	writeBuf  bytes.Buffer

	// exposed via ReadChan()
	readChan chan []byte

	// exposed via PeekChan()
	peekChan chan []byte

	// internal channels
	depthChan         chan int64
	writeChan         chan []byte
	writeResponseChan chan error
	emptyChan         chan int
	emptyResponseChan chan error
	exitChan          chan int
	exitSyncChan      chan int

	logf AppLogFunc
}

type options struct {
	name            string
	dataDIR         string
	maxBytesPerFile int64
	minMsgSize      int32
	maxMsgSize      int32
	syncEvery       int64
	syncTimeout     time.Duration
	logf            AppLogFunc
}

func defaultOptions() *options {
	wd, _ := os.Getwd()
	return &options{
		name:            "diskqueue",
		dataDIR:         filepath.Join(wd, ".diskq"),
		maxBytesPerFile: 104857600, // 100MB
		minMsgSize:      0,
		maxMsgSize:      10 * 1024 * 1024, // 10MB
		syncEvery:       2500,             // number of writes
		syncTimeout:     2 * time.Second,  // duration of time
		logf: func(lvl LogLevel, f string, args ...interface{}) {
			builder := strings.Builder{}
			builder.WriteString(fmt.Sprintf("[%s] ", lvl))
			builder.WriteString(fmt.Sprintf(f, args...))
			fmt.Println(builder.String())
		},
	}
}

type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(opt *options) {
	f(opt)
}

func WithName(name string) Option {
	return optionFunc(func(opt *options) {
		opt.name = name
	})
}

func WithDataDIR(dataDIR string) Option {
	return optionFunc(func(opt *options) {
		opt.dataDIR = dataDIR
	})
}

func WithMaxBytesPerFile(maxBytesPerFile int64) Option {
	return optionFunc(func(opt *options) {
		opt.maxBytesPerFile = maxBytesPerFile
	})
}

func WithMinMsgSize(minMsgSize int32) Option {
	return optionFunc(func(opt *options) {
		opt.minMsgSize = minMsgSize
	})
}

func WithMaxMsgSize(maxMsgSize int32) Option {
	return optionFunc(func(opt *options) {
		opt.maxMsgSize = maxMsgSize
	})
}

func WithSyncEvery(syncEvery int64) Option {
	return optionFunc(func(opt *options) {
		opt.syncEvery = syncEvery
	})
}

func WithSyncTimeout(syncTimeout time.Duration) Option {
	return optionFunc(func(opt *options) {
		opt.syncTimeout = syncTimeout
	})
}

func WithLogf(logf AppLogFunc) Option {
	return optionFunc(func(opt *options) {
		opt.logf = logf
	})
}

// New instantiates an instance of DiskQueue, retrieving metadata
// from the filesystem and starting the read ahead goroutine
func New(opts ...Option) Interface {
	_options := defaultOptions()
	for _, o := range opts {
		o.apply(_options)
	}

	return newDiskq(_options.name, _options.dataDIR, _options.maxBytesPerFile,
		_options.minMsgSize, _options.maxMsgSize,
		_options.syncEvery, _options.syncTimeout, _options.logf)
}

// New instantiates an instance of DiskQueue, retrieving metadata
// from the filesystem and starting the read ahead goroutine
func newDiskq(name string, dataDIR string, maxBytesPerFile int64,
	minMsgSize int32, maxMsgSize int32,
	syncEvery int64, syncTimeout time.Duration, logf AppLogFunc,
) Interface {
	d := &DiskQueue{
		name:            name,
		dataDIR:         dataDIR,
		maxBytesPerFile: maxBytesPerFile,
		minMsgSize:      minMsgSize,
		maxMsgSize:      maxMsgSize,
		syncEvery:       syncEvery,
		syncTimeout:     syncTimeout,
		logf:            logf,

		readChan:          make(chan []byte),
		peekChan:          make(chan []byte),
		depthChan:         make(chan int64),
		writeChan:         make(chan []byte),
		writeResponseChan: make(chan error),
		emptyChan:         make(chan int),
		emptyResponseChan: make(chan error),
		exitChan:          make(chan int),
		exitSyncChan:      make(chan int),
	}

	// no need to lock here, nothing else could possibly be touching this instance
	err := d.retrieveMetaData()
	if err != nil && !os.IsNotExist(err) {
		d.logf(ERROR, "DISKQUEUE(%s) failed to retrieveMetaData - %s", d.name, err)
	}

	go d.ioLoop()
	return d
}

// Depth returns the depth of the queue
func (d *DiskQueue) Depth() int64 {
	depth, ok := <-d.depthChan
	if !ok {
		// ioLoop exited
		depth = d.depth
	}
	return depth
}

// ReadChan returns the receive-only []byte channel for reading data
func (d *DiskQueue) ReadChan() <-chan []byte {
	return d.readChan
}

func (d *DiskQueue) PeekChan() <-chan []byte {
	return d.peekChan
}

// Put writes a []byte to the queue
func (d *DiskQueue) Put(data []byte) error {
	d.RLock()
	defer d.RUnlock()

	if d.exitFlag == 1 {
		return errors.New("exiting")
	}

	d.writeChan <- data
	return <-d.writeResponseChan
}

// Close cleans up the queue and persists metadata
func (d *DiskQueue) Close() error {
	err := d.exit(false)
	if err != nil {
		return err
	}
	return d.sync()
}

func (d *DiskQueue) Delete() error {
	return d.exit(true)
}

func (d *DiskQueue) exit(deleted bool) error {
	d.Lock()
	defer d.Unlock()

	d.exitFlag = 1

	if deleted {
		d.logf(INFO, "DISKQUEUE(%s): deleting", d.name)
	} else {
		d.logf(INFO, "DISKQUEUE(%s): closing", d.name)
	}

	close(d.exitChan)
	// ensure that ioLoop has exited
	<-d.exitSyncChan

	close(d.depthChan)

	if d.readFile != nil {
		d.readFile.Close()
		d.readFile = nil
	}

	if d.writeFile != nil {
		d.writeFile.Close()
		d.writeFile = nil
	}

	return nil
}

// Empty destructively clears out any pending data in the queue
// by fast forwarding read positions and removing intermediate files
func (d *DiskQueue) Empty() error {
	d.RLock()
	defer d.RUnlock()

	if d.exitFlag == 1 {
		return errors.New("exiting")
	}

	d.logf(INFO, "DISKQUEUE(%s): emptying", d.name)

	d.emptyChan <- 1
	return <-d.emptyResponseChan
}

func (d *DiskQueue) deleteAllFiles() error {
	err := d.skipToNextRWFile()

	innerErr := os.Remove(d.metaDataFileName())
	if innerErr != nil && !os.IsNotExist(innerErr) {
		d.logf(ERROR, "DISKQUEUE(%s) failed to remove metadata file - %s", d.name, innerErr)
		return innerErr
	}

	return err
}

func (d *DiskQueue) skipToNextRWFile() error {
	var err error

	if d.readFile != nil {
		d.readFile.Close()
		d.readFile = nil
	}

	if d.writeFile != nil {
		d.writeFile.Close()
		d.writeFile = nil
	}

	for i := d.readFileNum; i <= d.writeFileNum; i++ {
		fn := d.fileName(i)
		innerErr := os.Remove(fn)
		if innerErr != nil && !os.IsNotExist(innerErr) {
			d.logf(ERROR, "DISKQUEUE(%s) failed to remove data file - %s", d.name, innerErr)
			err = innerErr
		}
	}

	d.writeFileNum++
	d.writePos = 0
	d.readFileNum = d.writeFileNum
	d.readPos = 0
	d.nextReadFileNum = d.writeFileNum
	d.nextReadPos = 0
	d.depth = 0

	return err
}

// readOne performs a low level filesystem read for a single []byte
// while advancing read positions and rolling files, if necessary
func (d *DiskQueue) readOne() ([]byte, error) {
	var err error
	var msgSize int32

	if d.readFile == nil {
		curFileName := d.fileName(d.readFileNum)
		d.readFile, err = os.OpenFile(curFileName, os.O_RDONLY, 0o600)
		if err != nil {
			return nil, err
		}

		d.logf(INFO, "DISKQUEUE(%s): readOne() opened %s", d.name, curFileName)

		if d.readPos > 0 {
			_, err = d.readFile.Seek(d.readPos, 0)
			if err != nil {
				d.readFile.Close()
				d.readFile = nil
				return nil, err
			}
		}

		// for "complete" files (i.e. not the "current" file), maxBytesPerFileRead
		// should be initialized to the file's size, or default to maxBytesPerFile
		d.maxBytesPerFileRead = d.maxBytesPerFile
		if d.readFileNum < d.writeFileNum {
			stat, err := d.readFile.Stat()
			if err == nil {
				d.maxBytesPerFileRead = stat.Size()
			}
		}

		d.reader = bufio.NewReader(d.readFile)
	}

	err = binary.Read(d.reader, binary.BigEndian, &msgSize)
	if err != nil {
		d.readFile.Close()
		d.readFile = nil
		return nil, err
	}

	if msgSize < d.minMsgSize || msgSize > d.maxMsgSize {
		// this file is corrupt and we have no reasonable guarantee on
		// where a new message should begin
		d.readFile.Close()
		d.readFile = nil
		return nil, fmt.Errorf("invalid message read size (%d)", msgSize)
	}

	readBuf := make([]byte, msgSize)
	_, err = io.ReadFull(d.reader, readBuf)
	if err != nil {
		d.readFile.Close()
		d.readFile = nil
		return nil, err
	}

	totalBytes := int64(4 + msgSize)

	// we only advance next* because we have not yet sent this to consumers
	// (where readFileNum, readPos will actually be advanced)
	d.nextReadPos = d.readPos + totalBytes
	d.nextReadFileNum = d.readFileNum

	// we only consider rotating if we're reading a "complete" file
	// and since we cannot know the size at which it was rotated, we
	// rely on maxBytesPerFileRead rather than maxBytesPerFile
	if d.readFileNum < d.writeFileNum && d.nextReadPos >= d.maxBytesPerFileRead {
		if d.readFile != nil {
			d.readFile.Close()
			d.readFile = nil
		}

		d.nextReadFileNum++
		d.nextReadPos = 0
	}

	return readBuf, nil
}

// writeOne performs a low level filesystem write for a single []byte
// while advancing write positions and rolling files, if necessary
func (d *DiskQueue) writeOne(data []byte) error {
	var err error

	dataLen := int32(len(data))
	totalBytes := int64(4 + dataLen)

	if dataLen < d.minMsgSize || dataLen > d.maxMsgSize {
		return fmt.Errorf("invalid message write size (%d) minMsgSize=%d maxMsgSize=%d", dataLen, d.minMsgSize, d.maxMsgSize)
	}

	// will not wrap-around if maxBytesPerFile + maxMsgSize < Int64Max
	if d.writePos > 0 && d.writePos+totalBytes > d.maxBytesPerFile {
		if d.readFileNum == d.writeFileNum {
			d.maxBytesPerFileRead = d.writePos
		}

		d.writeFileNum++
		d.writePos = 0

		// sync every time we start writing to a new file
		err = d.sync()
		if err != nil {
			d.logf(ERROR, "DISKQUEUE(%s) failed to sync - %s", d.name, err)
		}

		if d.writeFile != nil {
			d.writeFile.Close()
			d.writeFile = nil
		}
	}
	if d.writeFile == nil {
		curFileName := d.fileName(d.writeFileNum)
		d.writeFile, err = os.OpenFile(curFileName, os.O_RDWR|os.O_CREATE, 0o600)
		if err != nil {
			return err
		}

		d.logf(INFO, "DISKQUEUE(%s): writeOne() opened %s", d.name, curFileName)

		if d.writePos > 0 {
			_, err = d.writeFile.Seek(d.writePos, 0)
			if err != nil {
				d.writeFile.Close()
				d.writeFile = nil
				return err
			}
		}
	}

	d.writeBuf.Reset()
	err = binary.Write(&d.writeBuf, binary.BigEndian, dataLen)
	if err != nil {
		return err
	}

	_, err = d.writeBuf.Write(data)
	if err != nil {
		return err
	}

	// only write to the file once
	_, err = d.writeFile.Write(d.writeBuf.Bytes())
	if err != nil {
		d.writeFile.Close()
		d.writeFile = nil
		return err
	}

	d.writePos += totalBytes
	d.depth += 1

	return err
}

// sync fsyncs the current writeFile and persists metadata
func (d *DiskQueue) sync() error {
	if d.writeFile != nil {
		err := d.writeFile.Sync()
		if err != nil {
			d.writeFile.Close()
			d.writeFile = nil
			return err
		}
	}

	err := d.persistMetaData()
	if err != nil {
		return err
	}

	d.needSync = false
	return nil
}

// retrieveMetaData initializes state from the filesystem
func (d *DiskQueue) retrieveMetaData() error {
	var f *os.File
	var err error

	fileName := d.metaDataFileName()
	f, err = os.OpenFile(fileName, os.O_RDONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	var depth int64
	_, err = fmt.Fscanf(f, "%d\n%d,%d\n%d,%d\n",
		&depth,
		&d.readFileNum, &d.readPos,
		&d.writeFileNum, &d.writePos)
	if err != nil {
		return err
	}
	d.depth = depth
	d.nextReadFileNum = d.readFileNum
	d.nextReadPos = d.readPos

	// if the metadata was not sync'd at the last shutdown of nsqd
	// then the actual file size might actually be larger than the writePos,
	// in which case the safest thing to do is skip to the next file for
	// writes, and let the reader salvage what it can from the messages in the
	// diskqueue beyond the metadata's likely also stale readPos
	fileName = d.fileName(d.writeFileNum)
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		return err
	}
	fileSize := fileInfo.Size()
	if d.writePos < fileSize {
		d.logf(WARN,
			"DISKQUEUE(%s) %s metadata writePos %d < file size of %d, skipping to new file",
			d.name, fileName, d.writePos, fileSize)
		d.writeFileNum += 1
		d.writePos = 0
		if d.writeFile != nil {
			d.writeFile.Close()
			d.writeFile = nil
		}
	}

	return nil
}

// persistMetaData atomically writes state to the filesystem
func (d *DiskQueue) persistMetaData() error {
	var f *os.File
	var err error

	fileName := d.metaDataFileName()
	tmpFileName := fmt.Sprintf("%s.%d.tmp", fileName, rand.Int())

	// write to tmp file
	f, err = os.OpenFile(tmpFileName, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(f, "%d\n%d,%d\n%d,%d\n",
		d.depth,
		d.readFileNum, d.readPos,
		d.writeFileNum, d.writePos)
	if err != nil {
		f.Close()
		return err
	}
	f.Sync()
	f.Close()

	// atomically rename
	return os.Rename(tmpFileName, fileName)
}

func (d *DiskQueue) metaDataFileName() string {
	return fmt.Sprintf(path.Join(d.dataDIR, "%s.diskqueue.meta.dat"), d.name)
}

func (d *DiskQueue) fileName(fileNum int64) string {
	return fmt.Sprintf(path.Join(d.dataDIR, "%s.diskqueue.%06d.dat"), d.name, fileNum)
}

func (d *DiskQueue) checkTailCorruption(depth int64) {
	if d.readFileNum < d.writeFileNum || d.readPos < d.writePos {
		return
	}

	// we've reached the end of the diskqueue
	// if depth isn't 0 something went wrong
	if depth != 0 {
		if depth < 0 {
			d.logf(ERROR,
				"DISKQUEUE(%s) negative depth at tail (%d), metadata corruption, resetting 0...",
				d.name, depth)
		} else if depth > 0 {
			d.logf(ERROR,
				"DISKQUEUE(%s) positive depth at tail (%d), data loss, resetting 0...",
				d.name, depth)
		}
		// force set depth 0
		d.depth = 0
		d.needSync = true
	}

	if d.readFileNum != d.writeFileNum || d.readPos != d.writePos {
		if d.readFileNum > d.writeFileNum {
			d.logf(ERROR,
				"DISKQUEUE(%s) readFileNum > writeFileNum (%d > %d), corruption, skipping to next writeFileNum and resetting 0...",
				d.name, d.readFileNum, d.writeFileNum)
		}

		if d.readPos > d.writePos {
			d.logf(ERROR,
				"DISKQUEUE(%s) readPos > writePos (%d > %d), corruption, skipping to next writeFileNum and resetting 0...",
				d.name, d.readPos, d.writePos)
		}

		d.skipToNextRWFile()
		d.needSync = true
	}
}

func (d *DiskQueue) moveForward() {
	oldReadFileNum := d.readFileNum
	d.readFileNum = d.nextReadFileNum
	d.readPos = d.nextReadPos
	d.depth -= 1

	// see if we need to clean up the old file
	if oldReadFileNum != d.nextReadFileNum {
		// sync every time we start reading from a new file
		d.needSync = true

		fn := d.fileName(oldReadFileNum)
		err := os.Remove(fn)
		if err != nil {
			d.logf(ERROR, "DISKQUEUE(%s) failed to Remove(%s) - %s", d.name, fn, err)
		}
	}

	d.checkTailCorruption(d.depth)
}

func (d *DiskQueue) handleReadError() {
	// jump to the next read file and rename the current (bad) file
	if d.readFileNum == d.writeFileNum {
		// if you can't properly read from the current write file it's safe to
		// assume that something is fucked and we should skip the current file too
		if d.writeFile != nil {
			d.writeFile.Close()
			d.writeFile = nil
		}
		d.writeFileNum++
		d.writePos = 0
	}

	badFn := d.fileName(d.readFileNum)
	badRenameFn := badFn + ".bad"

	d.logf(WARN,
		"DISKQUEUE(%s) jump to next file and saving bad file as %s",
		d.name, badRenameFn)

	err := os.Rename(badFn, badRenameFn)
	if err != nil {
		d.logf(ERROR,
			"DISKQUEUE(%s) failed to rename bad diskqueue file %s to %s",
			d.name, badFn, badRenameFn)
	}

	d.readFileNum++
	d.readPos = 0
	d.nextReadFileNum = d.readFileNum
	d.nextReadPos = 0

	// significant state change, schedule a sync on the next iteration
	d.needSync = true

	d.checkTailCorruption(d.depth)
}

// ioLoop provides the backend for exposing a go channel (via ReadChan())
// in support of multiple concurrent queue consumers
//
// it works by looping and branching based on whether or not the queue has data
// to read and blocking until data is either read or written over the appropriate
// go channels
//
// conveniently this also means that we're asynchronously reading from the filesystem
func (d *DiskQueue) ioLoop() {
	var dataRead []byte
	var err error
	var count int64
	var r chan []byte
	var p chan []byte

	syncTicker := time.NewTicker(d.syncTimeout)

	for {
		// dont sync all the time :)
		if count == d.syncEvery {
			d.needSync = true
		}

		if d.needSync {
			err = d.sync()
			if err != nil {
				d.logf(ERROR, "DISKQUEUE(%s) failed to sync - %s", d.name, err)
			}
			count = 0
		}

		if (d.readFileNum < d.writeFileNum) || (d.readPos < d.writePos) {
			if d.nextReadPos == d.readPos {
				dataRead, err = d.readOne()
				if err != nil {
					d.logf(ERROR, "DISKQUEUE(%s) reading at %d of %s - %s",
						d.name, d.readPos, d.fileName(d.readFileNum), err)
					d.handleReadError()
					continue
				}
			}
			r = d.readChan
			p = d.peekChan
		} else {
			r = nil
			p = nil
		}

		select {
		// the Go channel spec dictates that nil channel operations (read or write)
		// in a select are skipped, we set r to d.readChan only when there is data to read
		case p <- dataRead:
		case r <- dataRead:
			count++
			// moveForward sets needSync flag if a file is removed
			d.moveForward()
		case d.depthChan <- d.depth:
		case <-d.emptyChan:
			d.emptyResponseChan <- d.deleteAllFiles()
			count = 0
		case dataWrite := <-d.writeChan:
			count++
			d.writeResponseChan <- d.writeOne(dataWrite)
		case <-syncTicker.C:
			if count == 0 {
				// avoid sync when there's no activity
				continue
			}
			d.needSync = true
		case <-d.exitChan:
			goto exit
		}
	}

exit:
	d.logf(INFO, "DISKQUEUE(%s): closing ... ioLoop", d.name)
	syncTicker.Stop()
	d.exitSyncChan <- 1
}
