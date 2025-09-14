package module

import (
	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-resources/images"
	"github.com/adm87/finch-resources/resources"
)

func Register(ctx finch.Context) {
	resources.RegisterSystem(images.NewImageResourceSystem())

	ctx.Logger().Info("resource module registered")
}
