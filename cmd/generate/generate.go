package generate

import (
	"resources/cmd/generate/manifest"

	"github.com/spf13/cobra"
)

func Generate() *cobra.Command {
	cmd := &cobra.Command{
		Use: "generate",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(manifest.ManifestCmd())

	return cmd
}
