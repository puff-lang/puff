package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Check a Puff project without generating output",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "check")

			return nil
		},
	}
}
