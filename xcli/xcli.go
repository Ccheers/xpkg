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
	rootCmd  *cobra.Command
	namedCmd map[string]struct{}
	options  *XCliOptions
}

type XCliOptions struct {
	short                string
	long                 string
	cmdList              ICommandList
	handleUnknownCommand func(ctx context.Context, args []string) error
}

func defaultXCliOptions() *XCliOptions {
	return &XCliOptions{
		short:                "xcli is a command line sdk",
		long:                 "xcli is a command line sdk",
		cmdList:              nil,
		handleUnknownCommand: nil,
	}
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

func WithHandleUnknownCommand(handleUnknownCommand func(ctx context.Context, args []string) error) XCliOption {
	return XCliOptionFunc(func(opt *XCliOptions) {
		opt.handleUnknownCommand = handleUnknownCommand
	})
}

func NewXCli(name string, opts ...XCliOption) *XCli {
	options := defaultXCliOptions()
	for _, o := range opts {
		o.apply(options)
	}

	root := &cobra.Command{
		Use:   name,
		Short: options.short,
		Long:  options.long,
		RunE: func(cmd *cobra.Command, args []string) error {
			if options.handleUnknownCommand != nil && len(args) > 0 {
				return options.handleUnknownCommand(cmd.Context(), args)
			}
			return cmd.Help()
		},
	}

	namedCmd := make(map[string]struct{}, len(options.cmdList))
	for _, cmd := range options.cmdList {
		root.AddCommand(BuildCobraCommand(cmd))
		namedCmd[cmd.Use()] = struct{}{}
	}

	return &XCli{
		rootCmd:  root,
		namedCmd: namedCmd,
		options:  options,
	}
}

func (x *XCli) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	eg := errgroup.WithCancel(ctx)
	eg.Go(func(ctx context.Context) error {
		args := os.Args[1:]
		if x.options.handleUnknownCommand != nil && len(args) > 0 {
			if _, ok := x.namedCmd[args[0]]; !ok {
				return x.options.handleUnknownCommand(ctx, args)
			}
		}
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
