package manifest

import (
	"github.com/adm87/finch-core/errors"
	"github.com/adm87/finch-core/linq"
)

// ResourceManifest is a map of resource names to their metadata.
//
// Resource names must be unique within the manifest. This enforces uniqueness across all resources within an application.
type ResourceManifest map[string]ResourceMetadata

// ResourceMetadata contains metadata about a resource in a filesystem.
type ResourceMetadata struct {
	// Root is the root path of the filesystem where the resource is located.
	Root string `json:"root"`

	// Path is the path to the resource relative to the root.
	//
	// Path does not include the root.
	Path string `json:"path"`

	// Size is the size of the resource in bytes.
	Size int64 `json:"size"`
}

// GetSubManifest retrieves a sub-manifest from the main manifest based on the root path.
func GetSubManifest(manifest ResourceManifest, root string) (ResourceManifest, error) {
	if manifest == nil {
		return nil, errors.NewInvalidArgumentError("manifest cannot be nil")
	}
	names := linq.SelectKeys(manifest, func(key string, value ResourceMetadata) bool {
		return value.Root == root
	})
	results := make(ResourceManifest, len(names))
	for _, name := range names {
		results[name] = manifest[name]
	}
	return results, nil
}
