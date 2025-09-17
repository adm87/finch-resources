package enums

import (
	"embed"
	"encoding/json"
	"log/slog"
	"os"
	"path"

	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-core/tmpl"
	"github.com/adm87/finch-resources/resources"
)

//go:embed template.tpl
var tmplFS embed.FS

func Generate(ctx finch.Context, manifest resources.Manifest, pkg string, dest string) {
	ctx.Logger().Info("generating resource enums", slog.String("dest", dest))

	data, err := tmplFS.ReadFile("template.tpl")
	if err != nil {
		ctx.Logger().Error("failed to read template", slog.String("error", err.Error()))
		return
	}

	wrapper := struct {
		Pkg      string
		Manifest map[string]any
	}{
		Pkg:      pkg,
		Manifest: manifest_map(manifest),
	}

	content := tmpl.Render("enums.go", data, wrapper)

	if err := os.WriteFile(path.Join(dest, "enums.go"), content, 0644); err != nil {
		ctx.Logger().Error("failed to write enums file", slog.String("error", err.Error()))
		return
	}
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
