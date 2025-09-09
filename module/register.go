package module

import (
	"github.com/adm87/finch-resources/resources"
)

func RegisterModule() error {
	if err := resources.RegisterHandler(
		resources.Images(),
	); err != nil {
		return err
	}
	return nil
}
