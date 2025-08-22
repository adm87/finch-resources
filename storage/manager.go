package storage

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sync"

	stderr "errors"

	"github.com/adm87/finch-core/errors"
	"github.com/adm87/finch-core/linq"
	"github.com/adm87/finch-core/types"
	"github.com/adm87/finch-resources/manifest"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type ResourceType string

const (
	ImageResourceType ResourceType = "image"
	DataResourceType  ResourceType = "data"
)

const (
	RequestBatchSize = 100
)

var (
	resourceTypeIdentifiers = map[ResourceType]types.HashSet[string]{
		ImageResourceType: types.MakeSetFrom(
			".png",
			".jpg",
			".jpeg",
			".bmp",
		),
		DataResourceType: types.MakeSetFrom(
			".json",
			".csv",
			".xml",
		),
	}
)

var (
	resourceManifest = make(manifest.ResourceManifest)
	fileSystems      = make(map[string]fs.FS)
)

var (
	ErrDuplicateFileSystems       = errors.NewDuplicateError("filesystem already registered")
	ErrEmptyLoadRequest           = errors.NewInvalidArgumentError("no resources to load")
	ErrInvalidRoot                = errors.NewInvalidArgumentError("invalid filesystem root")
	ErrResourceSizeMismatch       = errors.NewInvalidArgumentError("resource size mismatch")
	ErrNilFileSystem              = errors.NewNilError("nil filesystem")
	ErrMetadataNotFound           = errors.NewNotFoundError("resource metadata not found")
	ErrResourceFileSystemNotFound = errors.NewNotFoundError("resource filesystem not found")
	ErrUnknownResourceType        = errors.NewNotFoundError("unknown resource type")
)

// =================================================================
// Manifest
// =================================================================

// SetManifest sets and validates the resource manifest for the storage framework.
//
// If the manifest is already set, replacing it will not unload resources from it or remove registered filesystems.
func SetManifest(m manifest.ResourceManifest) error {
	resourceManifest = m
	return manifest.ValidateManifest(m)
}

// Manifest returns the resource manifest for the storage framework.
func Manifest() manifest.ResourceManifest {
	return resourceManifest
}

// GetSubManifest retrieves a sub-manifest from the main manifest based on the root path.
func GetSubManifest(root string) (manifest.ResourceManifest, error) {
	if resourceManifest == nil {
		return nil, errors.NewInvalidArgumentError("manifest cannot be nil")
	}
	names := linq.SelectKeys(resourceManifest, func(key string, value manifest.ResourceMetadata) bool {
		return value.Root == root
	})
	results := make(manifest.ResourceManifest, len(names))
	for _, name := range names {
		results[name] = resourceManifest[name]
	}
	return results, nil
}

// =================================================================
// File Systems
// =================================================================

// RegisterFileSystems registers a batch of filesystems to the storage framework.
//
// These filesystems will be used to load resources into their respected caches based off their manifest metadata.
func RegisterFileSystems(fsystems map[string]fs.FS) error {
	for root, sys := range fsystems {
		if root == "" {
			return ErrInvalidRoot
		}

		if sys == nil {
			return ErrNilFileSystem
		}

		if _, exists := fileSystems[root]; exists {
			return ErrDuplicateFileSystems
		}

		fileSystems[root] = sys
	}
	return nil
}

// RemoveFileSystems removes a batch of filesystems from the storage framework.
//
// Resources loaded from these filesystems will remain usable until explicitly released.
func RemoveFileSystems(roots ...string) {
	for _, root := range roots {
		delete(fileSystems, root)
	}
}

// =================================================================
// Loading
// =================================================================

// Load attempts to load resources using the provided names. The method will check the set manifest for associated resource metadata and prepare requests to load the resources.
// Requests are batched if the number of requests exceeds the storage.RequestBatchSize. Each batch will be processed concurrently, and any errors will be returned.
func Load(names ...string) error {
	if len(names) == 0 {
		return ErrEmptyLoadRequest
	}

	requests := make([]types.Pair[string, manifest.ResourceMetadata], 0)
	for _, name := range linq.Distinct(names) {
		meta, exists := resourceManifest[name]
		if !exists {
			return ErrMetadataNotFound
		}
		requests = append(requests, types.Pair[string, manifest.ResourceMetadata]{
			First: name, Second: meta,
		})
	}

	results, err := load(linq.Batch(requests, RequestBatchSize))
	if err != nil {
		return err
	}

	for name, data := range results {
		resourceType, err := GetResourceType(name)
		if err != nil {
			return err
		}

		switch resourceType {
		case ImageResourceType:
			img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
			if err != nil {
				return err
			}
			imageStore.Add(name, img)
		case DataResourceType:
			dataStore.Add(name, data)
		}
	}

	return nil
}

func load(batches [][]types.Pair[string, manifest.ResourceMetadata]) (map[string][]byte, error) {
	if len(batches) == 1 {
		return load_batch(batches[0])
	}

	results := make(map[string][]byte)

	batchResultCh := make(chan map[string][]byte, len(batches))
	batchErrCh := make(chan error, len(batches))
	wg := sync.WaitGroup{}

	wg.Add(len(batches))
	for _, batch := range batches {
		go func(requests []types.Pair[string, manifest.ResourceMetadata]) {
			defer wg.Done()

			defer func() {
				if r := recover(); r != nil {
					batchErrCh <- fmt.Errorf("panic occurred: %v", r)
				}
			}()

			result, err := load_batch(requests)
			if err != nil {
				batchErrCh <- err
				return
			}

			batchResultCh <- result
		}(batch)
	}
	wg.Wait()

	close(batchResultCh)
	close(batchErrCh)

	errs := make([]error, 0)
	for err := range batchErrCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, stderr.Join(errs...)
	}

	for result := range batchResultCh {
		for key, value := range result {
			results[key] = value
		}
	}

	return results, nil
}

func load_batch(batch []types.Pair[string, manifest.ResourceMetadata]) (map[string][]byte, error) {
	results := make(map[string][]byte)

	if len(batch) == 0 {
		return results, nil
	}

	for _, request := range batch {
		key := request.First
		root := request.Second.Root
		path := request.Second.Path
		size := request.Second.Size

		filesys, exists := fileSystems[root]
		if !exists {
			return nil, ErrResourceFileSystemNotFound
		}

		// Embedded file systems need the root as part of the path
		if _, ok := filesys.(embed.FS); ok {
			path = filepath.Join(root, path)
		}

		file, err := filesys.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		raw, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		if size != int64(len(raw)) {
			return nil, ErrResourceSizeMismatch
		}

		results[key] = raw
	}

	return results, nil
}

// =================================================================
// Utils
// =================================================================

func GetResourceType(name string) (ResourceType, error) {
	meta, exists := resourceManifest[name]
	if !exists {
		return "", ErrMetadataNotFound
	}

	for resourceType, extensions := range resourceTypeIdentifiers {
		if extensions.Contains(filepath.Ext(meta.Path)) {
			return resourceType, nil
		}
	}

	return "", ErrUnknownResourceType
}

func SetFallback(name string) error {
	resourceType, err := GetResourceType(name)
	if err != nil {
		return err
	}

	switch resourceType {
	case ImageResourceType:
		imageStore.SetFallback(name)
	case DataResourceType:
		dataStore.SetFallback(name)
	}

	return nil
}
