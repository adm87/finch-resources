package manifest

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/adm87/finch-core/errors"
	"github.com/adm87/finch-core/hash"
)

// defaultIgnoredExtensions is a set of file extensions that are ignored by default if not specified in the command line options.
var defaultIgnoredExtensions = hash.HashSet[string]{
	".go": hash.SetEntry,
}

// GenerateManifest walks through the specified root directory and generates metadata for each resource in it. That metadata is then returned as a ResourceManifest.
func GenerateManifest(root string, manifestName string, ignoredExtensions hash.HashSet[string]) (ResourceManifest, error) {
	if ignoredExtensions == nil {
		ignoredExtensions = defaultIgnoredExtensions
	}

	manifest := ResourceManifest{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the manifest file itself and directories
		if info.IsDir() || info.Name() == manifestName {
			return nil
		}

		ext := filepath.Ext(path)
		if ignoredExtensions.Contains(ext) {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		parts := strings.Split(relativePath, string(filepath.Separator))
		if len(parts) == 0 {
			return errors.NewInvalidArgumentError("path must contain at least one part")
		}

		filename := strings.TrimSuffix(parts[len(parts)-1], ext)
		if filename == "" {
			return errors.NewInvalidArgumentError("filename must not be empty")
		}

		if _, exists := manifest[filename]; exists {
			return errors.NewDuplicateError(filename)
		}

		manifest[filename] = ResourceMetadata{
			Root: parts[0],
			Path: strings.Join(parts[1:], string(filepath.Separator)),
			Size: info.Size(),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return manifest, nil
}
