package resources

import (
	"bytes"

	"github.com/adm87/finch-core/errors"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// =================================================================
// Image storage
// =================================================================

var resource_types = []string{"png", "jpg", "jpeg"}
var handlerInstance = &ImageResourceHandler{
	fallbackKey: "",
	store:       NewStore[*ebiten.Image](),
}

func Images() *ImageResourceHandler {
	return handlerInstance
}

type ImageResourceHandler struct {
	fallbackKey string
	store       *Store[*ebiten.Image]
}

func (handler *ImageResourceHandler) Get(key string) (*ebiten.Image, error) {
	img, err := handler.store.Get(key)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (handler *ImageResourceHandler) StoreData(key string, data []byte, resourceType string) error {
	if key == "" {
		return errors.NewInvalidArgumentError("resource key cannot be empty")
	}
	if data == nil {
		return errors.NewInvalidArgumentError("resource data cannot be nil")
	}

	has, err := handler.store.Has(key)
	if err != nil {
		return err
	}
	if has {
		return errors.NewDuplicateError("resource already exists: " + key)
	}

	img, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	return handler.store.Set(key, img)
}

func (handler *ImageResourceHandler) ClearData(key string) error {
	has, err := handler.store.Has(key)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}

	img, err := handler.store.Get(key)
	if err != nil {
		return err
	}
	img.Deallocate()

	return handler.store.Remove(key)
}

func (handler *ImageResourceHandler) Fallback() string {
	return handler.fallbackKey
}

func (handler *ImageResourceHandler) SetFallback(key string) error {
	if key == "" {
		return errors.NewInvalidArgumentError("fallback key cannot be empty")
	}
	handler.fallbackKey = key
	return nil
}

func (handler *ImageResourceHandler) ResourceTypes() []string {
	return resource_types
}

func (handler *ImageResourceHandler) IsLoaded(key string) bool {
	has, _ := handler.store.Has(key)
	return has
}
