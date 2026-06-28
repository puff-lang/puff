package cli

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "puff",
		Short:         "The Puff programming language",
		Long:          "Puff is a programming language and toolchain for Minecraft datapacks.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(NewVersionCommand())
	cmd.AddCommand(NewInitCommand())
	cmd.AddCommand(NewCheckCommand())
	cmd.AddCommand(NewBundleCommand())

	return cmd
}
