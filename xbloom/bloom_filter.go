// Package xbloom 提供了一个高效的布隆过滤器实现
// 布隆过滤器是一种空间高效的概率数据结构，用于测试元素是否在集合中
// 它可能产生假阳性（说存在但实际不存在），但不会产生假阴性（说不存在但实际存在）
package xbloom

import (
	"bytes"
	"encoding/gob"
	"errors"
	"hash/fnv"
	"math"
)

// BloomFilter 布隆过滤器结构体
// 使用字节数组存储位信息，相比布尔数组节省87.5%内存
type BloomFilter struct {
	bits    []byte // 位数组，每个字节包含8个位
	size    uint64 // 位数组的总位数
	hashNum uint64 // 哈希函数的数量
}

// hash1 第一个哈希函数，使用FNV-1a算法
// 参数 data: 要哈希的字节数据
// 返回值: 64位哈希值
func hash1(data []byte) uint64 {
	h := fnv.New64a() // 创建FNV-1a哈希器
	h.Write(data)     // 写入数据
	return h.Sum64()  // 返回64位哈希值
}

// hash2 第二个哈希函数，使用FNV-1算法
// 参数 data: 要哈希的字节数据
// 返回值: 64位哈希值
func hash2(data []byte) uint64 {
	h := fnv.New64() // 创建FNV-1哈希器
	h.Write(data)    // 写入数据
	return h.Sum64() // 返回64位哈希值
}

// New 创建一个新的布隆过滤器
// 参数 expectedItems: 预期要插入的元素数量
// 参数 falsePositiveRate: 期望的假阳性率（0.0-1.0之间）
// 返回值: 布隆过滤器指针
func New(expectedItems uint64, falsePositiveRate float64) *BloomFilter {
	// 根据预期元素数量和假阳性率计算最优位数组大小
	size := optimalSize(expectedItems, falsePositiveRate)
	// 计算最优哈希函数数量
	hashNum := optimalHashFunctions(size, expectedItems)
	// 计算需要的字节数（每8位需要1字节，向上取整）
	byteSize := (size + 7) / 8

	return &BloomFilter{
		bits:    make([]byte, byteSize), // 初始化字节数组
		size:    size,                   // 设置位数组大小
		hashNum: hashNum,                // 设置哈希函数数量
	}
}

// optimalSize 根据预期元素数量和假阳性率计算最优位数组大小
// 使用公式: m = -n * ln(p) / (ln(2))^2
// 参数 n: 预期元素数量
// 参数 p: 期望假阳性率
// 返回值: 最优位数组大小
func optimalSize(n uint64, p float64) uint64 {
	return uint64(math.Ceil(-1 * float64(n) * math.Log(p) / math.Pow(math.Log(2), 2)))
}

// optimalHashFunctions 根据位数组大小和预期元素数量计算最优哈希函数数量
// 使用公式: k = (m/n) * ln(2)
// 参数 m: 位数组大小
// 参数 n: 预期元素数量
// 返回值: 最优哈希函数数量
func optimalHashFunctions(m, n uint64) uint64 {
	return uint64(math.Ceil(float64(m) / float64(n) * math.Log(2)))
}

// Add 向布隆过滤器中添加元素
// 参数 data: 要添加的元素数据
func (bf *BloomFilter) Add(data []byte) {
	// 计算两个基础哈希值
	h1 := hash1(data)
	h2 := hash2(data)

	// 使用双哈希技术生成k个不同的哈希值
	for i := uint64(0); i < bf.hashNum; i++ {
		// 组合哈希: hash = h1 + i*h2
		hash := (h1 + i*h2) % bf.size
		// 计算字节索引（哪个字节）
		byteIndex := hash / 8
		// 计算位索引（字节内的哪一位）
		bitIndex := hash % 8
		// 使用按位或操作设置对应位为1
		bf.bits[byteIndex] |= 1 << bitIndex
	}
}

// Contains 检查元素是否可能存在于布隆过滤器中
// 返回 false 表示元素绝对不存在，返回 true 表示元素可能存在
// 参数 data: 要检查的元素数据
// 返回值: 元素是否可能存在
func (bf *BloomFilter) Contains(data []byte) bool {
	// 计算两个基础哈希值
	h1 := hash1(data)
	h2 := hash2(data)

	// 检查所有k个哈希位置
	for i := uint64(0); i < bf.hashNum; i++ {
		// 组合哈希: hash = h1 + i*h2
		hash := (h1 + i*h2) % bf.size
		// 计算字节索引
		byteIndex := hash / 8
		// 计算位索引
		bitIndex := hash % 8
		// 使用按位与操作检查对应位是否为1
		if (bf.bits[byteIndex] & (1 << bitIndex)) == 0 {
			// 只要有一个位为0，元素绝对不存在
			return false
		}
	}
	// 所有位都为1，元素可能存在
	return true
}

// Serialize 将布隆过滤器序列化为字节数组
// 返回值: 序列化后的字节数据和错误信息
func (bf *BloomFilter) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	// 创建 gob 编码器
	encoder := gob.NewEncoder(&buf)

	// 创建临时结构体包含所有需要序列化的字段
	data := struct {
		Bits    []byte // 位数组
		Size    uint64 // 位数组大小
		HashNum uint64 // 哈希函数数量
	}{
		Bits:    bf.bits,
		Size:    bf.size,
		HashNum: bf.hashNum,
	}

	// 编码数据
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Deserialize 从字节数组中反序列化布隆过滤器
// 参数 data: 序列化的字节数据
// 返回值: 布隆过滤器指针和错误信息
func Deserialize(data []byte) (*BloomFilter, error) {
	// 检查输入数据是否为空
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}

	// 创建缓冲区和解码器
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)

	// 定义解码目标结构体
	var decoded struct {
		Bits    []byte // 位数组
		Size    uint64 // 位数组大小
		HashNum uint64 // 哈希函数数量
	}

	// 解码数据
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}

	// 创建并返回布隆过滤器实例
	return &BloomFilter{
		bits:    decoded.Bits,    // 恢复位数组
		size:    decoded.Size,    // 恢复大小
		hashNum: decoded.HashNum, // 恢复哈希函数数量
	}, nil
}

// Clear 清空布隆过滤器，将所有位设置为0
func (bf *BloomFilter) Clear() {
	// 遍历所有字节，将其设置为0
	for i := range bf.bits {
		bf.bits[i] = 0
	}
}

// Size 返回布隆过滤器的位数组大小
// 返回值: 位数组的总位数
func (bf *BloomFilter) Size() uint64 {
	return bf.size
}

// HashFunctions 返回布隆过滤器使用的哈希函数数量
// 返回值: 哈希函数数量
func (bf *BloomFilter) HashFunctions() uint64 {
	return bf.hashNum
}

// EstimatedFalsePositiveRate 估算当前的假阳性率
// 根据当前已设置的位数计算假阳性率
// 返回值: 估算的假阳性率 (0.0-1.0)
func (bf *BloomFilter) EstimatedFalsePositiveRate() float64 {
	setBits := uint64(0)
	// 统计已设置的位数
	for _, b := range bf.bits {
		// 检查每个字节的每一位
		for i := uint(0); i < 8; i++ {
			if (b & (1 << i)) != 0 {
				setBits++ // 统计设置为1的位数
			}
		}
	}
	// 计算位设置比例
	ratio := float64(setBits) / float64(bf.size)
	// 使用公式计算假阳性率: (ratio)^k
	return math.Pow(ratio, float64(bf.hashNum))
}
