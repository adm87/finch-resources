package resources

import (
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-core/fsys"
	"github.com/adm87/finch-core/types"
)

const JsonName = "resource.manifest"

type Manifest map[string]*Metadata

type Metadata struct {
	Root   string         `json:"root"`
	Path   string         `json:"path,omitempty"`
	Type   string         `json:"type"`
	Extras map[string]any `json:"extras,omitempty"`
}

var loadedManifest Manifest

func GetManifest() Manifest {
	return loadedManifest
}

func LoadManifest(ctx finch.Context, p string) {
	file := path.Join(p, JsonName)
	if err := fsys.ReadJson(file, &loadedManifest); err != nil {
		if os.IsNotExist(err) {
			ctx.Logger().Warn("no resource manifest found")
			return
		}
		panic(err)
	}
}

func GenerateManifest(ctx finch.Context, root string) Manifest {
	supportedTypes := types.NewHashSetFromSlice(GetSupportedResourceTypes())

	m := make(Manifest)

	ctx.Logger().Info("generating resource manifest", slog.String("path", root))
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		file := d.Name()
		if file == JsonName {
			return nil
		}

		relPath, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}

		ext := filepath.Ext(file)
		name := strings.TrimSuffix(file, ext)

		if _, exists := m[name]; exists {
			ctx.Logger().Warn("skipping duplicate resource name", slog.String("name", name), slog.String("path", relPath))
			return nil
		}

		ext = strings.TrimPrefix(ext, ".")
		if !supportedTypes.Contains(ext) {
			ctx.Logger().Warn("skipping unsupported resource type", slog.String("path", relPath))
			return nil
		}

		ctx.Logger().Info("processing", slog.String("path", relPath))

		parts := strings.Split(relPath, "/")

		resPath := strings.TrimPrefix(relPath, parts[0]+"/")
		resPath = strings.TrimSuffix(resPath, file)

		metadata := &Metadata{
			Root: parts[0],
			Type: ext,
			Path: resPath,
		}

		if err := SystemForType(ext).GenerateMetadata(ctx, name, metadata); err != nil {
			ctx.Logger().Error("failed to generate metadata for resource", slog.String("path", relPath), slog.Any("error", err))
			return nil
		}

		m[name] = metadata
		return nil
	})

	if err != nil {
		panic(err)
	}

	return m
}
