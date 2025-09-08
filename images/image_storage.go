package images

import (
	"github.com/adm87/finch-core/errors"
	"github.com/hajimehoshi/ebiten/v2"
)

// =================================================================
// Image storage
// =================================================================

var resource_types = []string{"png", "jpg", "jpeg"}
var handlerInstance = &ImageResourceHandler{
	fallbackKey: "",
}

func Resources() *ImageResourceHandler {
	return handlerInstance
}

type ImageResourceHandler struct {
	fallbackKey string
}

func (handler *ImageResourceHandler) Get(key string) (*ebiten.Image, error) {
	return nil, nil
}

func (handler *ImageResourceHandler) StoreData(key string, data []byte) error {
	return nil
}

func (handler *ImageResourceHandler) ClearData(key string) error {
	return nil
}

func (handler *ImageResourceHandler) Fallback() string {
	return handler.fallbackKey
}

func (handler *ImageResourceHandler) SetFallback(key string) error {
	if key == "" {
		return errors.NewInvalidArgumentError("fallback key cannot be empty")
	}
	// TODO - validate key is loaded
	handler.fallbackKey = key
	return nil
}

func (handler *ImageResourceHandler) ResourceTypes() []string {
	return resource_types
}
