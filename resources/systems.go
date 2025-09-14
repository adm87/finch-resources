package resources

import (
	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-core/utils"
	"github.com/adm87/finch-resources/manifest"
)

var (
	byResourceType = make(map[string]ResourceSystem)
	bySystemType   = make(map[ResourceSystemType]ResourceSystem)
)

type ResourceSystem interface {
	ResourceTypes() []string
	Type() ResourceSystemType

	Load(ctx finch.Context, key string, metadata manifest.Metadata) error
	Unload(ctx finch.Context, key string) error
}

type ResourceSystemType uint64

func NewResourceSystemKey[T ResourceSystem]() ResourceSystemType {
	return ResourceSystemType(utils.GetHashFromType[T]())
}

func RegisterSystem(system ResourceSystem) {
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
}

func SystemForType(rt string) ResourceSystem {
	if sys, exists := byResourceType[rt]; exists {
		return sys
	}
	return nil
}

func GetSystem(st ResourceSystemType) ResourceSystem {
	if sys, exists := bySystemType[st]; exists {
		return sys
	}
	return nil
}
