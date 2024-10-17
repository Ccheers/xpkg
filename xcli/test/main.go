package main

import (
	"context"
	"log"

	"github.com/ccheers/xpkg/xcli"
	"github.com/spf13/pflag"
)

func main() {
	err := xcli.NewXCli(
		"xtest",
		xcli.WithCommandList(xcli.ICommandList{NewT()}),
		xcli.WithHandleUnknownCommand(func(ctx context.Context, args []string) error {
			log.Println("fallback", args)
			return nil
		}),
	).Run(context.Background())
	if err != nil {
		panic(err)
	}
}

type T struct {
	fs *pflag.FlagSet

	boolVar   bool
	stringVar string
}

func NewT() *T {
	x := &T{
		fs: nil,
	}

	fs := pflag.NewFlagSet("foo", pflag.ContinueOnError)
	fs.BoolVarP(&x.boolVar, "bool", "b", false, "bool var")
	fs.StringVarP(&x.stringVar, "string", "s", "default", "string var")
	x.fs = fs
	return x
}

func (x *T) Use() string {
	return "foo"
}

func (x *T) Short() string {
	return "foo-short"
}

func (x *T) Long() string {
	return "foo-long"
}

func (x *T) Run(ctx context.Context, args []string) error {
	log.Println("foo run")
	log.Println("x.boolVar", x.boolVar)
	log.Println("x.stringVar", x.stringVar)
	return nil
}

func (x *T) Flags() *pflag.FlagSet {
	return x.fs
}
