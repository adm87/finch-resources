package resources

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"sync"
)

// noCopy is a Go trick to prevent copying of structs that embed it.
//
// See https://golang.org/issues/8005#issuecomment-190753527 for details.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// =============== Asset Types ===============

// Asset represents a unique identifier for an asset within a resource system.
type Asset uint64

// AssetPath represents the file path of an asset within a resource system.
type AssetPath string

// AssetManifest maps assets to their corresponding file paths.
type AssetManifest map[Asset]AssetPath

// AssetLock represents a lock for a specific asset within a resource system.
//
// It encapsulates the mutex used to synchronize access to the asset,
// along with references to the resource system and the loader context.
type AssetLock struct {
	_ noCopy

	lock    *sync.Mutex
	context *LoaderContext
}

// Release releases the lock held by the AssetGuard.
//
// Once released, the guard should be considered invalid and should not be used again.
// Subsequent calls to Release will result in a panic
func (al *AssetLock) Release() {
	if al.lock == nil {
		panic(ErrDoubleRelease)
	}

	al.lock.Unlock()
	al.context.system.release(al)

	al.lock = nil
}

// =============== Resource Loader ===============

// LoaderContext provides context for resource loading operations.
//
// It includes a reference to the resource system and the batch index
// for the current loading operation.
type LoaderContext struct {
	context.Context
	system *ResourceSystem
	batch  uint
}

// Lock acquires an exclusive lock for the specified asset.
//
// The returned AssetLock must be released by calling its Release method
// once the asset is no longer needed by the loader.
//
// Panics if the batch already holds an acquired asset lock.
func (lc *LoaderContext) Lock(asset Asset) *AssetLock {
	return lc.system.lockAsset(lc, asset)
}

// ReadBytes reads the raw byte data of the specified asset from the resource system.
func (lc *LoaderContext) ReadBytes(asset Asset) ([]byte, error) {
	return lc.system.ReadBytes(asset)
}

type ResourceLoader func(ctx *LoaderContext) error

// NoopResourceLoader is a ResourceLoader that performs no operation. Use this to skip loading or unloading.
var NoopResourceLoader ResourceLoader = func(ctx *LoaderContext) error {
	return nil
}

// =============== Errors ===============

type ResourceError string

func (re ResourceError) Error() string {
	return string(re)
}

var (
	ErrSkipped            = ResourceError("resource loading/unloading skipped")
	ErrMissingFilesystem  = ResourceError("missing filesystem for resource system")
	ErrNotInFilesystem    = ResourceError("asset not found in linked filesystem")
	ErrBatchAlreadyLocked = ResourceError("batch already has an acquired asset lock")
	ErrDoubleRelease      = ResourceError("attempted to release an already released AssetLock")
)

// =============== Resource System Options ===============

type ResourceSystemOptions struct {
	// TrimRoot indicates whether to trim the root directory from asset paths when reading from the filesystem.
	//
	// Some filesystems may include the root directory in asset paths, which can lead to mismatches with the manifest.
	// Enabling this option ensures that asset paths are consistent with the manifest by removing the root directory prefix.
	TrimRoot bool

	// BatchSize defines the number of assets to process in a single batch during loading or unloading.
	// A value of 0 indicates that all assets should be processed in a single batch.
	//
	// Batches are loaded in parallel, be sure to consider thread-safety when using this option.
	// Resource systems provide locking mechanisms for individual assets to help with thread-safety.
	//
	// Default is 0.
	BatchSize uint
}

// =============== Resource System ===============

// ResourceSystem manages the loading and unloading of assets from a linked filesystem.
//
// It provides mechanisms for asset locking to ensure thread-safe access during loading operations.
// Each resource system has its own manifest mapping assets to file paths.
// Multiple resource systems can coexist, each managing its own set of assets and filesystem.
type ResourceSystem struct {
	locks      map[Asset]*sync.Mutex
	batchLocks map[uint]*AssetLock
	mu         sync.Mutex

	name       string
	options    ResourceSystemOptions
	manifest   AssetManifest
	filesystem fs.FS
}

func NewResourceSystem(name string, manifest AssetManifest, options ResourceSystemOptions) *ResourceSystem {
	return &ResourceSystem{
		locks:      make(map[Asset]*sync.Mutex, len(manifest)),
		batchLocks: make(map[uint]*AssetLock),
		manifest:   manifest,
		name:       name,
		options:    options,
	}
}

func (rs *ResourceSystem) release(lock *AssetLock) {
	rs.mu.Lock()
	delete(rs.batchLocks, lock.context.batch)
	rs.mu.Unlock()
}

// lockAsset acquires an exclusive lock for the given asset.
//
// Each asset has its own mutex, allowing different assets to be loaded in
// parallel while ensuring that only one batch loads a specific asset at
// a time.
//
// This function enforces a "one asset lock at a time per batch" rule.
// A batch must release its current asset lock before acquiring another.
//
// Violations result in a panic.
func (rs *ResourceSystem) lockAsset(ctx *LoaderContext, asset Asset) *AssetLock {
	// Note:
	// We first lock the resource system to safely access the locks map.
	// Then we lock the specific asset's mutex.
	// This ordering prevents potential deadlocks.

	rs.mu.Lock()
	lock, exists := rs.locks[asset]
	if !exists {
		lock = &sync.Mutex{}
		rs.locks[asset] = lock
	}
	rs.mu.Unlock()

	lock.Lock()

	rs.mu.Lock()
	// Enforce one asset lock at a time per batch
	if _, exists := rs.batchLocks[ctx.batch]; exists {
		rs.mu.Unlock()
		lock.Unlock()
		panic(ErrBatchAlreadyLocked)
	}

	assetLock := &AssetLock{
		lock:    lock,
		context: ctx,
	}
	rs.batchLocks[ctx.batch] = assetLock
	rs.mu.Unlock()

	return assetLock
}

// Name returns the name of the resource system.
func (rs *ResourceSystem) Name() string {
	return rs.name
}

// Contains checks if the asset exists in the resource system's manifest.
func (rs *ResourceSystem) Contains(asset Asset) bool {
	_, exists := rs.manifest[asset]
	return exists
}

// ReadBytes reads the raw byte data of the specified asset from the resource system.
func (rs *ResourceSystem) ReadBytes(asset Asset) ([]byte, error) {
	if !rs.Contains(asset) {
		return nil, ErrNotInFilesystem
	}

	fsys := rs.Filesystem()
	if fsys == nil {
		return nil, ErrMissingFilesystem
	}

	path := string(rs.manifest[asset])
	if rs.options.TrimRoot {
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 {
			path = parts[1]
		}
	}

	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Load loads the specified assets into the resource system.
func (rs *ResourceSystem) Load(loaders ...ResourceLoader) error {
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, loader := range loaders {
		ctx := &LoaderContext{
			Context: cancelCtx,
			system:  rs,
			batch:   0, // TASK: 0 is placeholder, will be replaced with actual batch index in the future
		}
		if err := loader(ctx); err != nil && !errors.Is(err, ErrSkipped) {
			return err
		}
	}

	return nil
}

// UseFilesystem links a filesystem to the resource system for asset loading.
func (rs *ResourceSystem) UseFilesystem(filesystem fs.FS) {
	rs.filesystem = filesystem
}

// Filesystem returns the linked filesystem of the resource system.
func (rs *ResourceSystem) Filesystem() fs.FS {
	return rs.filesystem
}
