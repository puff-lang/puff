package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

type BundleOptions struct {
	Target     string
	AllTargets bool
	Output     string
}

func NewBundleCommand() *cobra.Command {
	bundleOpts := &BundleOptions{}

	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Compile a Puff project into a Minecraft datapack",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			fmt.Fprint(out, "bundle")

			if bundleOpts.Target != "" {
				fmt.Fprintf(out, " --target %s", bundleOpts.Target)
			}

			if bundleOpts.AllTargets {
				fmt.Fprint(out, " --all-targets")
			}

			if bundleOpts.Output != "" {
				fmt.Fprintf(out, " --output %s", bundleOpts.Output)
			}

			fmt.Fprintln(out)

			return nil
		},
	}

	cmd.Flags().StringVar(&bundleOpts.Target, "target", "", "Minecraft target version")
	cmd.Flags().BoolVar(&bundleOpts.AllTargets, "all-targets", false, "Compile all supported Minecraft targets")
	cmd.Flags().StringVarP(&bundleOpts.Output, "output", "o", "", "Output directory")

	return cmd
}
