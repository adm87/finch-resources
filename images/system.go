package images

import (
	"bytes"
	"errors"
	"sync"

	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-core/types"
	"github.com/adm87/finch-resources/resources"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

var systemType = resources.NewResourceSystemKey[*ImageResourceSystem]()

type ImageResourceSystem struct {
	images  map[string]*ebiten.Image
	loading types.HashSet[string]
	mu      sync.RWMutex
}

func NewImageResourceSystem() *ImageResourceSystem {
	return &ImageResourceSystem{
		images:  make(map[string]*ebiten.Image),
		loading: types.NewHashSet[string](),
		mu:      sync.RWMutex{},
	}
}

func Get(key string) (*ebiten.Image, bool) {
	sys := resources.GetSystem(systemType).(*ImageResourceSystem)
	return sys.GetImage(key)
}

func (irs *ImageResourceSystem) ResourceTypes() []string {
	return []string{"png", "jpg", "jpeg", "bmp"}
}

func (irs *ImageResourceSystem) Type() resources.ResourceSystemType {
	return systemType
}

func (irs *ImageResourceSystem) Load(ctx finch.Context, key string, metadata resources.Metadata) error {
	if !irs.try_load(key) {
		return nil
	}

	defer func() {
		irs.mu.Lock()
		irs.loading.Remove(key)
		irs.mu.Unlock()
	}()

	data, err := resources.LoadData(ctx, key, metadata)
	if err != nil {
		return err
	}

	img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	irs.mu.Lock()
	irs.images[key] = img
	irs.mu.Unlock()
	return nil
}

func (irs *ImageResourceSystem) Unload(ctx finch.Context, key string) error {
	return errors.New("not implemented")
}

func (irs *ImageResourceSystem) GetProperties(resourceType string) (map[string]any, error) {
	return nil, nil
}

func (irs *ImageResourceSystem) GetImage(key string) (*ebiten.Image, bool) {
	irs.mu.RLock()
	defer irs.mu.RUnlock()

	img, exists := irs.images[key]
	return img, exists
}

func (irs *ImageResourceSystem) IsLoaded(key string) bool {
	irs.mu.RLock()
	defer irs.mu.RUnlock()

	_, exists := irs.images[key]
	return exists
}

func (irs *ImageResourceSystem) try_load(key string) bool {
	irs.mu.Lock()
	defer irs.mu.Unlock()

	if _, exists := irs.images[key]; exists {
		return false
	}
	if irs.loading.Contains(key) {
		return false
	}

	irs.loading.Add(key)
	return true
}
