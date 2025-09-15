package images

import (
	"bytes"
	"errors"
	"fmt"
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
		loading: make(types.HashSet[string]),
		mu:      sync.RWMutex{},
	}
}

func Get(key string) (*ebiten.Image, bool) {
	sys := resources.GetSystem(systemType).(*ImageResourceSystem)
	return sys.GetImage(key)
}

func (rs *ImageResourceSystem) ResourceTypes() []string {
	return []string{"png", "jpg", "jpeg", "bmp"}
}

func (rs *ImageResourceSystem) Type() resources.ResourceSystemType {
	return systemType
}

func (rs *ImageResourceSystem) IsLoaded(key string) bool {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	_, exists := rs.images[key]
	return exists
}

func (rs *ImageResourceSystem) Load(ctx finch.Context, key string, metadata *resources.Metadata) error {
	if err := rs.try_load(key); err != nil {
		return fmt.Errorf("image resource is already loading or loaded: %s", key)
	}

	defer func() {
		rs.mu.Lock()
		rs.loading.Remove(key)
		rs.mu.Unlock()
	}()

	data, err := resources.LoadData(ctx, key, metadata)
	if err != nil {
		return err
	}

	img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	rs.mu.Lock()
	rs.images[key] = img
	rs.mu.Unlock()
	return nil
}

func (rs *ImageResourceSystem) Unload(ctx finch.Context, key string) error {
	return errors.New("not implemented")
}

func (rs *ImageResourceSystem) GenerateMetadata(ctx finch.Context, key string, metadata *resources.Metadata) error {
	return nil
}

func (rs *ImageResourceSystem) GetDependencies(ctx finch.Context, key string, metadata *resources.Metadata) []string {
	return nil
}

func (rs *ImageResourceSystem) GetImage(key string) (*ebiten.Image, bool) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	img, exists := rs.images[key]
	return img, exists
}

func (rs *ImageResourceSystem) try_load(key string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if _, exists := rs.images[key]; exists {
		return fmt.Errorf("image resource is already loaded: %s", key)
	}
	if rs.loading.Contains(key) {
		return fmt.Errorf("image resource is already loading: %s", key)
	}

	rs.loading.Add(key)
	return nil
}
