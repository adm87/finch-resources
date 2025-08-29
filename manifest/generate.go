package manifest

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/adm87/finch-core/errors"
	"github.com/adm87/finch-core/types"
)

// defaultIgnoredExtensions is a set of file extensions that are ignored by default if not specified in the command line options.
var defaultIgnoredExtensions = types.MakeSetFrom(".go")

// GenerateManifest walks through the specified root directory and generates metadata for each resource in it. That metadata is then returned as a ResourceManifest.
func GenerateManifest(root string, manifestName string, ignoredExtensions types.HashSet[string]) (ResourceManifest, error) {
	if ignoredExtensions == nil {
		ignoredExtensions = defaultIgnoredExtensions
	}

	manifest := ResourceManifest{}

	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the manifest file itself and directories
		if info.IsDir() || info.Name() == manifestName {
			return nil
		}

		ext := filepath.Ext(p)
		if ignoredExtensions.Contains(ext) {
			return nil
		}

		relativePath, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)

		parts := strings.Split(relativePath, "/")
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
			Path: path.Join(parts[1:]...),
			Size: info.Size(),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return manifest, nil
}
