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

// ResourceSystem defines a system that can load and manage resources of specific types.
//
// Each ResourceSystem must implement this interface to be registered and used within the resource management framework.
// ResourceSystems are responsible for loading, unloading, and providing metadata properties for resources of the types they support.
// The resource management framework loads large numbers of resources in batches, so ResourceSystems should be designed to handle concurrent load requests efficiently.
type ResourceSystem interface {
	ResourceTypes() []string
	Type() ResourceSystemType

	IsLoaded(key string) bool

	Load(ctx finch.Context, key string, metadata *Metadata) error
	Unload(ctx finch.Context, key string) error

	GenerateMetadata(key string, metadata *Metadata) error
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
