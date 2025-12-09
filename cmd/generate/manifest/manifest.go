package manifest

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	input       string   = "."
	packageName string   = "assets"
	embed       []string = []string{}
)

func ManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Generates manifest script with resource.Assets constants",
		RunE: func(cmd *cobra.Command, args []string) error {
			absPath, err := filepath.Abs(input)
			if err != nil {
				return err
			}
			for i := range embed {
				embed[i], err = filepath.Abs(embed[i])
				if err != nil {
					return err
				}
				embed[i], err = filepath.Rel(absPath, embed[i])
				if err != nil {
					return err
				}
			}
			return Generate(absPath, packageName, embed)
		},
	}

	cmd.Flags().StringVarP(&input, "input", "i", input, "Path to the input directory")
	cmd.Flags().StringVarP(&packageName, "package", "p", packageName, "Package name for the generated manifest")
	cmd.Flags().StringArrayVarP(&embed, "embed", "e", embed, "Paths to embed into the binary")

	return cmd
}
