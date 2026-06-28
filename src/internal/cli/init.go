package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init [name]",
		Short: "Create a new Puff project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "puff_project"

			if len(args) == 1 {
				name = args[0]
			}

			fmt.Fprintf(cmd.OutOrStdout(), "init %s\n", name)

			return nil
		},
	}
}
