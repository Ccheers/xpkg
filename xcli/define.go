package xcli

import (
	"context"
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ICommandList []ICommand

type ICommand interface {
	Use() string
	Short() string
	Long() string
	Run(ctx context.Context, args []string) error
	Flags() *flag.FlagSet
}

func BuildCobraCommand(icmd ICommand) *cobra.Command {
	c := &cobra.Command{
		Use:   icmd.Use(),
		Short: icmd.Short(),
		Long:  icmd.Long(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return icmd.Run(cmd.Context(), args)
		},
	}
	ConvFlag2Pflag(icmd.Flags(), c.Flags())
	return c
}

// ============================================== flag value ==============================================

type pflagValueAdapter struct {
	value flag.Value
}

func newPflagValueAdapter(value flag.Value) *pflagValueAdapter {
	return &pflagValueAdapter{value: value}
}

func (x *pflagValueAdapter) String() string {
	return x.value.String()
}

func (x *pflagValueAdapter) Set(s string) error {
	return x.value.Set(s)
}

func (x *pflagValueAdapter) Type() string {
	return "string"
}

func ConvFlag2Pflag(src *flag.FlagSet, dst *pflag.FlagSet) {
	src.VisitAll(func(f *flag.Flag) {
		dst.Var(newPflagValueAdapter(f.Value), f.Name, f.Usage)
	})
}
