package storage

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// ResourceCache is a cache for resources like images and data.
type ResourceCache struct {
	images Store[ebiten.Image]
	data   Store[[]byte]
}

// NewResourceCache creates a new ResourceCache instance.
func NewResourceCache() *ResourceCache {
	return &ResourceCache{
		images: Store[ebiten.Image]{
			items: make(map[string]*ebiten.Image),
		},
		data: Store[[]byte]{
			items: make(map[string]*[]byte),
		},
	}
}
