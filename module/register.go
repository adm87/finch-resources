package module

import (
	"github.com/adm87/finch-resources/images"
	"github.com/adm87/finch-resources/storage"
)

func RegisterModule() error {
	if err := storage.RegisterStorageSystems(
		images.Storage(),
	); err != nil {
		return err
	}
	return nil
}
