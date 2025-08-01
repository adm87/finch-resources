package storage

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sync"

	stderrs "errors"

	"github.com/adm87/finch-core/errors"
	"github.com/adm87/finch-core/hash"
	"github.com/adm87/finch-core/linq"
	"github.com/adm87/finch-resources/manifest"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	ResourceTypeImage = "image"
	ResourceTypeData  = "data"
)

const (
	// ResourceBatchThreshold is the number of resources that can be loaded before parallel loading is used.
	//
	// If the number of resources isn't fully divisible, then the request is split into batches of the nearest divisible size.
	ResourceBatchThreshold = 100
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

	imageStore Store[*ebiten.Image]
	dataStore  Store[*[]byte]
}

// NewResourceCache creates a new ResourceCache instance.
func NewResourceCache() *ResourceCache {
	return &ResourceCache{
		manifest:    manifest.ResourceManifest{},
		filesystems: make(map[string]fs.FS),
		imageStore:  make(Store[*ebiten.Image]),
		dataStore:   make(Store[*[]byte]),
	}
}

func (c *ResourceCache) Manifest() manifest.ResourceManifest {
	return c.manifest
}

func (c *ResourceCache) SetManifest(m manifest.ResourceManifest) error {
	if err := manifest.ValidateManifest(m); err != nil {
		return err
	}
	c.manifest = m
	return nil
}

func (c *ResourceCache) AddFilesystem(root string, fsys fs.FS) error {
	if root == "" {
		return errors.InvalidArgumentError("root must not be empty")
	}
	if fsys == nil {
		return errors.NewNilError("filesystem cannot be nil")
	}

	if _, exists := c.filesystems[root]; exists {
		return errors.NewDuplicateError(fmt.Sprintf("filesystem for root '%s' already exists", root))
	}

	c.filesystems[root] = fsys
	return nil
}

func (c *ResourceCache) RemoveFilesystem(root string) error {
	if root == "" {
		return errors.InvalidArgumentError("root must not be empty")
	}

	if _, exists := c.filesystems[root]; !exists {
		return nil
	}

	// TODO:  Consider deallocating resources associated with this filesystem

	delete(c.filesystems, root)
	return nil
}

// Images returns the image store of the cache.
//
// This store contains loaded images. Call LoadImage to load images into this store.
func (c *ResourceCache) Images() *Store[*ebiten.Image] {
	return &c.imageStore
}

// Data returns the data store of the cache.
//
// This store contains loaded data. Call LoadData to load data into this store.
func (c *ResourceCache) Data() *Store[*[]byte] {
	return &c.dataStore
}

// ClearCache clears the cache of all loaded resources.
//
// Resources are deallocated and removed from the cache, making them unusable.
func (c *ResourceCache) ClearCache() {
	for name := range c.imageStore {
		c.imageStore[name].Deallocate()
	}
	for name := range c.dataStore {
		c.dataStore[name] = nil
	}
	c.imageStore = make(Store[*ebiten.Image])
	c.dataStore = make(Store[*[]byte])
}

// Load loads resources by their names into the cache.
func (c *ResourceCache) Load(names ...string) error {
	if len(names) == 0 {
		return nil
	}

	requests := []linq.Pair[string, manifest.ResourceMetadata]{}
	for _, name := range linq.Distinct(names) {
		metadata, ok := c.manifest[name]
		if !ok {
			return errors.NewNotFoundError(fmt.Sprintf("resource '%s' not found in manifest", name))
		}
		requests = append(requests, linq.Pair[string, manifest.ResourceMetadata]{
			First:  name,
			Second: metadata,
		})
	}

	batches := linq.Batch(requests, ResourceBatchThreshold)
	if len(batches) == 0 {
		return nil
	}

	results, err := c.internal_load_batches(batches)
	if err != nil {
		return err
	}

	for name, data := range results {
		rtype, err := c.internal_get_resource_type(c.manifest[name])
		if err != nil {
			return err
		}
		switch rtype {
		case ResourceTypeImage:
			img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
			if err != nil {
				return err
			}
			c.imageStore.Add(name, img)
		case ResourceTypeData:
			c.dataStore.Add(name, &data)
		}
	}

	return nil
}

// Unload unloads resources by their names from the cache.
//
// Unloading a resource will remove it from the cache and deallocate it, making it unusable.
func (c *ResourceCache) Unload(names ...string) error {
	if len(names) == 0 {
		return nil
	}

	for _, name := range linq.Distinct(names) {
		if image, ok := c.imageStore[name]; ok {
			image.Deallocate()
			c.imageStore.Remove(name)
			continue
		}
		if _, ok := c.dataStore[name]; ok {
			c.dataStore.Remove(name)
			continue
		}
	}

	return nil
}

func (c *ResourceCache) internal_load_batches(batches [][]linq.Pair[string, manifest.ResourceMetadata]) (map[string][]byte, error) {
	if len(batches) == 1 {
		return c.internal_load_batch(batches[0])
	}

	results := make(map[string][]byte)
	batchResultCh := make(chan map[string][]byte, len(batches))
	batchErrCh := make(chan error, len(batches))

	wg := sync.WaitGroup{}

	wg.Add(len(batches))
	for _, batch := range batches {
		go func(b []linq.Pair[string, manifest.ResourceMetadata]) {
			defer wg.Done()

			// If a panic occurs, we'll catch it and send an error to the channel
			defer func() {
				if r := recover(); r != nil {
					batchErrCh <- errors.NewParallelError(fmt.Sprintf("panic while loading batch: %v", r))
				}
			}()

			// Load the batch of resources
			result, err := c.internal_load_batch(b)
			if err != nil {
				batchErrCh <- err
				return
			}

			// Send the result to the channel
			batchResultCh <- result
		}(batch)
	}
	wg.Wait()

	close(batchResultCh)
	close(batchErrCh)

	// Collect errors from the batchErrCh
	errs := make([]error, 0)
	for err := range batchErrCh {
		errs = append(errs, err)
	}

	// If there are any errors, return them
	if len(errs) > 0 {
		return nil, stderrs.Join(errs...)
	}

	// Collect results from the batchResultCh
	for result := range batchResultCh {
		for name, data := range result {
			results[name] = data
		}
	}

	return results, nil
}

func (c *ResourceCache) internal_load_batch(batch []linq.Pair[string, manifest.ResourceMetadata]) (map[string][]byte, error) {
	results := make(map[string][]byte)

	if len(batch) == 0 {
		return results, nil
	}

	for _, pair := range batch {
		filesys := c.filesystems[pair.Second.Root]
		if filesys == nil {
			return nil, errors.NewNotFoundError(fmt.Sprintf("filesystem for root '%s' not found", pair.Second.Root))
		}

		path := pair.Second.Path

		if _, ok := filesys.(embed.FS); ok {
			path = filepath.Join(pair.Second.Root, pair.Second.Path)
		}

		file, err := filesys.Open(path)
		if err != nil {
			return nil, errors.NewIOError(fmt.Sprintf("failed to open resource '%s' at path '%s': %v", pair.First, path, err))
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return nil, errors.NewIOError(fmt.Sprintf("failed to read resource '%s' at path '%s': %v", pair.First, path, err))
		}

		if err := file.Close(); err != nil {
			return nil, errors.NewIOError(fmt.Sprintf("failed to close resource '%s' at path '%s': %v", pair.First, path, err))
		}

		dataSize := int64(len(data))
		if dataSize != pair.Second.Size {
			return nil, errors.NewInvalidArgumentError(fmt.Sprintf("resource '%s' at path '%s' has size %d, expected %d", pair.First, path, dataSize, pair.Second.Size))
		}

		results[pair.First] = data
	}

	return results, nil
}

func (c *ResourceCache) internal_get_resource_type(m manifest.ResourceMetadata) (string, error) {
	if ImageTypes.Contains(filepath.Ext(m.Path)) {
		return ResourceTypeImage, nil
	}
	if DataTypes.Contains(filepath.Ext(m.Path)) {
		return ResourceTypeData, nil
	}
	return "", errors.NewInvalidArgumentError(fmt.Sprintf("unknown resource type for path '%s'", m.Path))
}
