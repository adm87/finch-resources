package storage

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"sync"

	stderr "errors"

	"github.com/adm87/finch-core/errors"
	"github.com/adm87/finch-core/linq"
	"github.com/adm87/finch-core/types"
	"github.com/adm87/finch-resources/manifest"
)

// StorageHandler defines the interface for a resource storage management system.
//
// Loaded resources are cached within the storage handler, and can be written to disk or deallocated as needed.
type StorageHandler interface {
	PutValue(key string, data any) error    // Store a value for a specific key.
	Allocate(key string, data []byte) error // Allocate data for a specific key.
	Deallocate(key string) error            // Deallocate data for a specific key.
	AssetTypes() types.HashSet[string]      // Return the asset types (file extensions) supported by the storage.
	SetDefault(key string) error            // Set the data for a specific key as the default data. This is returned if a key is not found.
	Has(key string) bool                    // Check if a specific key exists in the storage and return its value.
}

var (
	storageByAssetType = make(map[string]StorageHandler)
	storageByKey       = make(map[string]StorageHandler)
	filesystems        = make(map[string]fs.FS)
	storageManifest    = make(manifest.ResourceManifest)
)

// =================================================================
// Manifest
// =================================================================

// SetManifest sets and validates the resource manifest for the storage framework.
//
// If the manifest is already set, replacing it will not unload resources from it or remove registered filesystems.
func SetManifest(m manifest.ResourceManifest) error {
	storageManifest = m
	return manifest.ValidateManifest(m)
}

// Manifest returns the resource manifest for the storage framework.
func Manifest() manifest.ResourceManifest {
	return storageManifest
}

// GetSubManifest retrieves a sub-manifest from the main manifest based on the root path.
func GetSubManifest(root string) (manifest.ResourceManifest, error) {
	if storageManifest == nil {
		return nil, errors.NewInvalidArgumentError("manifest cannot be nil")
	}
	names := linq.SelectKeys(storageManifest, func(key string, value manifest.ResourceMetadata) bool {
		return value.Root == root
	})
	results := make(manifest.ResourceManifest, len(names))
	for _, name := range names {
		results[name] = storageManifest[name]
	}
	return results, nil
}

// =================================================================
// Registration
// =================================================================

// RegisterStorageSystems registers a new storage for a collection of asset types.
func RegisterStorageSystems(store ...StorageHandler) error {
	for _, c := range store {
		for assetType := range c.AssetTypes() {
			if _, exists := storageByAssetType[assetType]; exists {
				return errors.NewDuplicateError("storage already exists for asset type: " + assetType)
			}
			storageByAssetType[assetType] = c
		}
	}
	return nil
}

// RegisterFilesystem registers a new storage filesystem.
func RegisterFileSystems(fsystems map[string]fs.FS) error {
	for name, filesystem := range fsystems {
		if filesystem == nil {
			return errors.NewNilError("nil filesystem")
		}
		if _, exists := filesystems[name]; exists {
			return errors.NewDuplicateError("filesystem already exists: " + name)
		}
		filesystems[name] = filesystem
	}
	return nil
}

// =================================================================
// Loading/Unloading
// =================================================================

func Load(keys ...string) error {
	if len(keys) == 0 {
		return errors.NewInvalidArgumentError("no resource keys provided")
	}

	requests := make([]types.Pair[string, manifest.ResourceMetadata], 0)
	for _, key := range linq.Distinct(keys) {
		meta, exists := storageManifest[key]
		if !exists {
			return errors.NewNotFoundError("resource not found in manifest: " + key)
		}
		requests = append(requests, types.Pair[string, manifest.ResourceMetadata]{
			First: key, Second: meta,
		})
	}

	err := load(linq.Batch(requests, 100))
	if err != nil {
		return err
	}

	return nil
}

func load(batches [][]types.Pair[string, manifest.ResourceMetadata]) error {
	if len(batches) == 1 {
		return load_batch(batches[0])
	}

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

			err := load_batch(requests)
			if err != nil {
				batchErrCh <- err
				return
			}
		}(batch)
	}
	wg.Wait()

	close(batchErrCh)

	errs := make([]error, 0)
	for err := range batchErrCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return stderr.Join(errs...)
	}

	return nil
}

func load_batch(batch []types.Pair[string, manifest.ResourceMetadata]) error {
	if len(batch) == 0 {
		return nil
	}

	for _, request := range batch {
		key := request.First
		requestRoot := request.Second.Root
		requestPath := request.Second.Path
		requestSize := request.Second.Size

		filesys, exists := filesystems[requestRoot]
		if !exists {
			return errors.NewNotFoundError("unknown filesystem: " + requestRoot)
		}

		// Note: Loading from an embedded filesystem requires the root of the filesystem to be included in the path.

		filesysPrefix := path.Join(requestRoot)
		if _, ok := filesys.(embed.FS); ok && !strings.HasPrefix(requestPath, filesysPrefix) {
			requestPath = path.Join(filesysPrefix, requestPath)
		}

		file, err := filesys.Open(requestPath)
		if err != nil {
			return err
		}
		defer file.Close()

		raw, err := io.ReadAll(file)
		if err != nil {
			return err
		}

		if requestSize != int64(len(raw)) {
			println(errors.NewConflictError(fmt.Sprintf("resource size mismatch. Expected %d, got %d", requestSize, len(raw))))
		}

		ext := strings.ToLower(filepath.Ext(requestPath))

		store, exists := storageByAssetType[ext]
		if !exists {
			return errors.NewNotFoundError("no storage found for asset type: " + ext)
		}

		storageByKey[key] = store
		if err := store.Allocate(key, raw); err != nil {
			delete(storageByKey, key)
			return err
		}
	}

	return nil
}

// Unload removes and deallocates data for a specific key.
//
// Unloaded data should be considered no longer valid, and could result in unintended behavior.
func Unload(key string) error {
	store := storageByKey[key]
	delete(storageByKey, key)

	if store == nil {
		return nil
	}
	return store.Deallocate(key)
}

// =================================================================
// Utility
// =================================================================

func SetDefault(key string) error {
	store := storageByKey[key]
	if store == nil {
		return errors.NewNotFoundError("no storage found for key: " + key)
	}
	return store.SetDefault(key)
}
