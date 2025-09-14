package manifest

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/adm87/finch-core/types"
)

const JsonName = "resource.manifest"

// TASK : Check for resource systems to get type specific metadata properties. Note, packages will need to be rearranged to avoid import cycles.

func Generate(root string, ignore types.HashSet[string]) Model {
	m := make(Model)

	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		file := d.Name()
		if file == JsonName {
			return nil
		}

		ext := filepath.Ext(file)
		name := strings.TrimSuffix(file, ext)

		if _, exists := m[name]; exists {
			panic("Duplicate resource name: " + name)
		}

		ext = strings.TrimPrefix(ext, ".")
		if ignore.Contains(ext) {
			return nil
		}

		relPath, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, "/")

		relPath = strings.TrimPrefix(relPath, parts[0]+"/")
		relPath = strings.TrimSuffix(relPath, file)

		m[name] = Metadata{
			Root: parts[0],
			Type: ext,
			Path: relPath,
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	return m
}
