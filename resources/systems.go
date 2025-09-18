package resources

import (
	"sync"

	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-core/linq"
	"github.com/adm87/finch-core/utils"
)

var (
	byResourceType = make(map[string]ResourceSystem)
	bySystemType   = make(map[ResourceSystemType]ResourceSystem)

	mu sync.RWMutex
)

// ResourceHandle provides an accessor for information about a specific resource,
// including its key, metadata, and loaded state.
//
// ResourceHandles are used throughout the resource management framework to reference and manage individual resources.
// They encapsulate the resource key and provide methods to access associated metadata and check if the resource is currently loaded.
type ResourceHandle string

func (r ResourceHandle) Key() string {
	return string(r)
}

func (r ResourceHandle) Metadata() (*Metadata, bool) {
	if loadedManifest == nil {
		return nil, false
	}

	if metadata, exists := loadedManifest[r.Key()]; exists {
		return metadata, true
	}

	return nil, false
}

func (r ResourceHandle) IsLoaded() bool {
	metadata, exists := r.Metadata()
	if !exists {
		return false
	}

	if sys := SystemForType(metadata.Type); sys != nil {
		return sys.IsLoaded(r)
	}

	return false
}

// ResourceSystem defines a system that can load and manage resources of specific types.
//
// Each ResourceSystem must implement this interface to be registered and used within the resource management framework.
// ResourceSystems are responsible for loading, unloading, and providing metadata properties for resources of the types they support.
// The resource management framework loads large numbers of resources in batches, so ResourceSystems should be designed to handle concurrent load requests efficiently.
type ResourceSystem interface {
	ResourceTypes() []string
	Type() ResourceSystemType

	Load(ctx finch.Context, handle ResourceHandle) error
	Unload(ctx finch.Context, handle ResourceHandle) error
	GetDependencies(ctx finch.Context, handle ResourceHandle) []ResourceHandle

	GenerateMetadata(ctx finch.Context, key string, metadata *Metadata) error
	IsLoaded(handle ResourceHandle) bool
}

type ResourceSystemType uint64

func NewResourceSystemKey[T ResourceSystem]() ResourceSystemType {
	return ResourceSystemType(utils.GetHashFromType[T]())
}

func RegisterSystem(system ResourceSystem) {
	mu.Lock()
	defer mu.Unlock()

	st := system.Type()
	if _, exists := bySystemType[st]; exists {
		panic("resource system of type already registered")
	}

	rts := system.ResourceTypes()
	for _, rt := range rts {
		if _, exists := byResourceType[rt]; exists {
			panic("resource system for type " + rt + " already registered")
		}
		byResourceType[rt] = system
	}

	bySystemType[st] = system
}

func SystemForType(rt string) ResourceSystem {
	mu.RLock()
	defer mu.RUnlock()

	if sys, exists := byResourceType[rt]; exists {
		return sys
	}
	return nil
}

func GetSystem(st ResourceSystemType) ResourceSystem {
	mu.RLock()
	defer mu.RUnlock()

	if sys, exists := bySystemType[st]; exists {
		return sys
	}
	return nil
}

func GetSupportedResourceTypes() []string {
	mu.RLock()
	defer mu.RUnlock()

	return linq.Keys(byResourceType)
}
