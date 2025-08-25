package module

import (
	"github.com/adm87/finch-resources/images"
	"github.com/adm87/finch-resources/storage"
)

func RegisterModule() error {
	if err := storage.RegisterCache(images.Cache()); err != nil {
		return err
	}
	return nil
}
