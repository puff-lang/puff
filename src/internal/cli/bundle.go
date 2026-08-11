package cli

import (
	"errors"
	"io"

	"github.com/puff-lang/puff/internal/compiler"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/spf13/cobra"
)

var errBundleFailed = errors.New("bundle failed")

type BundleOptions struct {
	Target string
	Output string
}

func NewBundleCommand() *cobra.Command {
	bundleOpts := &BundleOptions{}

	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Compile a Puff project into a Minecraft datapack",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result := compiler.Bundle(cmd.Context(), compiler.BundleOptions{
				StartDir: ".",
				Target:   bundleOpts.Target,
				Output:   bundleOpts.Output,
			})

			for _, issues := range [][]diagnostic.Diagnostic{result.Diagnostics.Errors, result.Diagnostics.Warnings} {
				for _, issue := range issues {
					if _, err := io.WriteString(cmd.ErrOrStderr(), diagnostic.FormatDiagnostic(issue, "")); err != nil {
						return err
					}
				}
			}

			if !result.Diagnostics.OK {
				return errBundleFailed
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&bundleOpts.Target, "target", "", "Minecraft target version")
	cmd.Flags().StringVarP(&bundleOpts.Output, "output", "o", "", "Output directory")

	return cmd
}
