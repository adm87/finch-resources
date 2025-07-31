package manifest

import "github.com/adm87/finch-core/fsys"

func LoadManifest(path string) (ResourceManifest, error) {
	manifest := ResourceManifest{}

	if err := fsys.ReadJson(path, &manifest); err != nil {
		return ResourceManifest{}, err
	}

	return manifest, ValidateManifest(manifest)
}
