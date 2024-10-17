package xcli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ICommandList []ICommand

type ICommand interface {
	Use() string
	Short() string
	Long() string
	Run(ctx context.Context, args []string) error
	Flags() *pflag.FlagSet
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

func ConvFlag2Pflag(src *pflag.FlagSet, dst *pflag.FlagSet) {
	src.VisitAll(func(f *pflag.Flag) {
		dst.AddFlag(f)
	})
}
