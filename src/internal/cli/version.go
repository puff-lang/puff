package cli

import (
	"fmt"

	"github.com/puff-lang/puff/internal/version"
	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			out := cmd.OutOrStdout()

			fmt.Fprintf(out, "puff %s\n", version.Version)
			fmt.Fprintf(out, "commit: %s\n", version.Commit)
			fmt.Fprintf(out, "date: %s\n", version.Date)
		},
	}
}
