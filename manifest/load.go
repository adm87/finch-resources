package manifest

import (
	"log/slog"
	"os"
	"path"

	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-core/fsys"
)

func Load(ctx finch.Context, p string) Model {
	var model Model

	file := path.Join(p, JsonName)
	if err := fsys.ReadJson(file, &model); err != nil {
		if os.IsNotExist(err) {
			ctx.Logger().Warn("resource manifest not found, continuing without", slog.String("path", file))
			return model
		}
		panic(err)
	}

	return model
}
