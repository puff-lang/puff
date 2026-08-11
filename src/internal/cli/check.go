package cli

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/puff-lang/puff/internal/compiler"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/spf13/cobra"
)

var errCheckFailed = errors.New("check failed")

func NewCheckCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check a Puff project without generating output",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result := compiler.Check(cmd.Context(), compiler.CheckOptions{StartDir: "."})

			if jsonOutput {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(result.Diagnostics); err != nil {
					return err
				}
			} else {
				sources := make(map[string]string, len(result.Files))
				for _, file := range result.Files {
					sources[file.RelPath] = file.Text
				}

				for _, issues := range [][]diagnostic.Diagnostic{result.Diagnostics.Errors, result.Diagnostics.Warnings} {
					for _, issue := range issues {
						if _, err := io.WriteString(cmd.ErrOrStderr(), diagnostic.FormatDiagnostic(issue, sources[issue.File])); err != nil {
							return err
						}
					}
				}
			}

			if !result.Diagnostics.OK {
				return errCheckFailed
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print diagnostics as JSON")

	return cmd
}
