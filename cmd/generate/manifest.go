package generate

import (
	"path/filepath"

	"github.com/adm87/finch-core/fsys"
	"github.com/adm87/finch-core/types"
	"github.com/adm87/finch-resources/manifest"
	"github.com/spf13/cobra"
)

var (
	RootPath     = "."
	ManifestName = "manifest.json"
	Indent       = false
)

var ManifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Generate a resource manifest for the specified root path",
	RunE: func(cmd *cobra.Command, args []string) error {
		abs, err := filepath.Abs(RootPath)
		if err != nil {
			return err
		}
		m, err := manifest.GenerateManifest(abs, ManifestName, types.MakeSetFrom(
			".go", ".mod", ".sum", ".gitignore", ".git", ".DS_Store", ".vscode", ".idea",
		))
		if err != nil {
			return err
		}
		if Indent {
			return fsys.WriteJsonIndent(filepath.Join(RootPath, ManifestName), m)
		}
		return fsys.WriteJson(filepath.Join(RootPath, ManifestName), m)
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	ManifestCmd.Flags().StringVar(&RootPath, "root", RootPath, "The root path to generate a resource manifest for.")
	ManifestCmd.Flags().StringVar(&ManifestName, "name", ManifestName, "The name of the generated manifest file.")
	ManifestCmd.Flags().BoolVar(&Indent, "indent", Indent, "Whether to indent the generated manifest file for readability.")
}
