package module

import (
	"github.com/adm87/finch-resources/storage"
)

func RegisterModule() error {
	if err := storage.RegisterResourceHandler(); err != nil {
		return err
	}
	return nil
}
