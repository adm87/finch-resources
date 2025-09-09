package module

import (
	"github.com/adm87/finch-resources/images"
	"github.com/adm87/finch-resources/resources"
)

func RegisterModule() error {
	if err := resources.RegisterHandler(
		images.Resources(),
	); err != nil {
		return err
	}
	return nil
}
