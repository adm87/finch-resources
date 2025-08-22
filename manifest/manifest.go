package manifest

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
