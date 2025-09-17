package enums

import (
	"embed"
	"encoding/json"
	"log/slog"

	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-core/tmpl"
	"github.com/adm87/finch-resources/resources"
)

//go:embed template.tpl
var tmplFS embed.FS

func Generate(ctx finch.Context, manifest resources.Manifest, pkg string, dest string) []byte {
	ctx.Logger().Info("generating resource enums", slog.String("dest", dest))

	data, err := tmplFS.ReadFile("template.tpl")
	if err != nil {
		ctx.Logger().Error("failed to read template", slog.String("error", err.Error()))
		return nil
	}

	wrapper := struct {
		Pkg      string
		Manifest map[string]any
	}{
		Pkg:      pkg,
		Manifest: manifest_map(manifest),
	}

	return tmpl.Render("enums.go", data, wrapper)
}

func manifest_map(manifest resources.Manifest) map[string]any {
	result := make(map[string]any)

	data, err := json.Marshal(manifest)
	if err != nil {
		return result
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return result
	}

	return result
}
