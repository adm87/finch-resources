package storage

import (
	"io/fs"

	"github.com/adm87/finch-core/hash"
	"github.com/adm87/finch-resources/manifest"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	ResourceTypeImage = "image"
	ResourceTypeData  = "data"
)

var (
	ImageTypes = hash.MakeSetFrom(
		".png",
		".jpg",
		".jpeg",
		".gif",
		".bmp",
		".webp",
	)
	DataTypes = hash.MakeSetFrom(
		".json",
		".txt",
		".xml",
		".csv",
	)
)

// ResourceCache is a cache for resources like images and data.
//
// It provides a manifest driven interface to load resources by their filename. A manifest much be loaded before resources can be loaded.
// Resources loaded into cache are stored and can be retrieved by their names. The cache will panic if a resource hasn't been loaded before requesting it,
// of if a resource is requested that does not exist in the manifest.
//
// Loaded resources must be unloaded after they are no longer needed to free up memory. Unloading a resource will remove it from the cache and make it unusable.
type ResourceCache struct {
	manifest    manifest.ResourceManifest
	filesystems map[string]fs.FS

	imageStore Store[ebiten.Image]
	dataStore  Store[[]byte]
}

// NewResourceCache creates a new ResourceCache instance.
func NewResourceCache() *ResourceCache {
	return &ResourceCache{
		filesystems: make(map[string]fs.FS),
		imageStore: Store[ebiten.Image]{
			items: make(map[string]*ebiten.Image),
		},
		dataStore: Store[[]byte]{
			items: make(map[string]*[]byte),
		},
	}
}

// Images returns the image store of the cache.
//
// This store contains loaded images. Call LoadImage to load images into this store.
func (c *ResourceCache) Images() *Store[ebiten.Image] {
	return &c.imageStore
}

// Data returns the data store of the cache.
//
// This store contains loaded data. Call LoadData to load data into this store.
func (c *ResourceCache) Data() *Store[[]byte] {
	return &c.dataStore
}

// LoadManifest loads a resource manifest from the specified path.
func (c *ResourceCache) LoadManifest(path string) (*manifest.ResourceManifest, error) {
	return &c.manifest, nil
}

// Load loads resources by their names into the cache.
//
// Each name must correspond to a file within the manifest. Calls will panic if a name is empty,
// does not exist in the manifest, or if the manifest is not loaded.
//
// If a resource is already loaded, it will be skipped.
func (c *ResourceCache) Load(names ...string) error {
	return nil
}
