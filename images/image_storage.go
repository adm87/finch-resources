package images

import (
	"bytes"
	"sync"

	"github.com/adm87/finch-core/errors"
	"github.com/adm87/finch-core/types"
	"github.com/adm87/finch-resources/storage"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// =================================================================
// Image storage
// =================================================================

var (
	assetTypes      = types.MakeSetFrom(".png", ".jpg", ".jpeg")
	storageInstance = &ImageStorage{
		mu:    sync.RWMutex{},
		store: storage.NewStore[*ebiten.Image](),
	}
)

type ImageStorage struct {
	mu    sync.RWMutex
	store *storage.Store[*ebiten.Image]
}

func Storage() *ImageStorage {
	return storageInstance
}

func (c *ImageStorage) Get(key string) (*ebiten.Image, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	img, err := c.store.Get(key)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func (c *ImageStorage) Allocate(key string, data []byte) error {
	img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.store.Add(key, img); err != nil {
		img.Deallocate()
		return err
	}

	return nil
}

func (c *ImageStorage) PutValue(key string, value any) error {
	img, ok := value.(*ebiten.Image)
	if !ok {
		return errors.NewInvalidArgumentError("value must be of type *ebiten.Image")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.store.Add(key, img); err != nil {
		img.Deallocate()
		return err
	}

	return nil
}

func (c *ImageStorage) Deallocate(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	img, err := c.store.Get(key)
	if err != nil {
		return err
	}

	img.Deallocate()

	if err := c.store.Remove(key); err != nil {
		return err
	}

	return nil
}

func (c *ImageStorage) AssetTypes() types.HashSet[string] {
	return assetTypes
}

func (c *ImageStorage) SetDefault(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	has, err := c.store.Has(key)
	if err != nil {
		return err
	}

	if !has {
		return errors.NewNotFoundError("default image not found in storage: " + key)
	}

	c.store.SetDefault(key)
	return nil
}
