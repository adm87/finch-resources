package images

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-resources/manifest"
	"github.com/adm87/finch-resources/resources"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var systemType = resources.NewResourceSystemKey[*ImageResourceSystem]()

type ImageResourceSystem struct {
	images map[string]*ebiten.Image
	mu     sync.RWMutex
}

func NewImageResourceSystem() *ImageResourceSystem {
	return &ImageResourceSystem{
		images: make(map[string]*ebiten.Image),
	}
}

func (irs *ImageResourceSystem) ResourceTypes() []string {
	return []string{"png", "jpg", "jpeg", "bmp"}
}

func (irs *ImageResourceSystem) Type() resources.ResourceSystemType {
	return systemType
}

func (irs *ImageResourceSystem) GetImage(key string) (*ebiten.Image, bool) {
	irs.mu.RLock()
	defer irs.mu.RUnlock()

	img, exists := irs.images[key]
	return img, exists
}

func (irs *ImageResourceSystem) Load(ctx finch.Context, key string, metadata manifest.Metadata) error {
	if _, exists := irs.images[key]; exists {
		return fmt.Errorf("image '%s' already loaded", key)
	}

	data, err := resources.LoadData(ctx, key, metadata)
	if err != nil {
		return err
	}

	img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	irs.mu.Lock()
	defer irs.mu.Unlock()

	irs.images[key] = img
	return nil
}

func (irs *ImageResourceSystem) Unload(ctx finch.Context, key string) error {
	return errors.New("not implemented")
}

func Get(key string) (*ebiten.Image, bool) {
	sys := resources.GetSystem(systemType).(*ImageResourceSystem)
	return sys.GetImage(key)
}
