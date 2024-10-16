package xcli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ccheers/xpkg/sync/errgroup"
	"github.com/spf13/cobra"
)

type XCli struct {
	rootCmd *cobra.Command
}

type XCliOptions struct {
	short   string
	long    string
	cmdList ICommandList
}

func defaultXCliOptions() *XCliOptions {
	return &XCliOptions{}
}

type XCliOption interface {
	apply(opt *XCliOptions)
}

type XCliOptionFunc func(opt *XCliOptions)

func (f XCliOptionFunc) apply(opt *XCliOptions) {
	f(opt)
}

func WithShort(short string) XCliOption {
	return XCliOptionFunc(func(opt *XCliOptions) {
		opt.short = short
	})
}

func WithLong(long string) XCliOption {
	return XCliOptionFunc(func(opt *XCliOptions) {
		opt.long = long
	})
}

func WithCommandList(cmdList ICommandList) XCliOption {
	return XCliOptionFunc(func(opt *XCliOptions) {
		opt.cmdList = cmdList
	})
}

func NewXCli(name string, opts ...XCliOption) *XCli {
	options := defaultXCliOptions()
	for _, o := range opts {
		o.apply(options)
	}

	rootCmd := &cobra.Command{
		Use:   name,
		Short: options.short,
		Long:  options.long,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	for _, cmd := range options.cmdList {
		rootCmd.AddCommand(BuildCobraCommand(cmd))
	}

	return &XCli{
		rootCmd: rootCmd,
	}
}

func (x *XCli) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	eg := errgroup.WithCancel(ctx)
	eg.Go(func(ctx context.Context) error {
		return x.rootCmd.ExecuteContext(ctx)
	})
	eg.Go(func(ctx context.Context) error {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			cancel()
			<-c
			os.Exit(1) // second signal. Exit directly.
		}()
		return nil
	})
	return eg.Wait()
}
