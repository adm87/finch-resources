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
	Root       string         `json:"root"`
	Path       string         `json:"path,omitempty"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
}

var loadedManifest Manifest

func GetManifest() Manifest {
	return loadedManifest
}

func UseManifest(m Manifest) {
	loadedManifest = m
}

func LoadManifest(ctx finch.Context, p string) Manifest {
	var manifest Manifest

	file := path.Join(p, JsonName)
	if err := fsys.ReadJson(file, &manifest); err != nil {
		if os.IsNotExist(err) {
			ctx.Logger().Warn("resource manifest not found, continuing without", slog.String("path", file))
			return manifest
		}
		panic(err)
	}

	return manifest
}

func GenerateManifest(ctx finch.Context, root string) Manifest {
	supportedTypes := types.NewHashSetFromSlice(GetSupportedResourceTypes())

	m := make(Manifest)

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

		ext := filepath.Ext(file)
		name := strings.TrimSuffix(file, ext)

		if _, exists := m[name]; exists {
			panic("Duplicate resource name: " + name)
		}

		ext = strings.TrimPrefix(ext, ".")
		if !supportedTypes.Contains(ext) {
			ctx.Logger().Warn("skipping unsupported resource type", slog.String("path", p))
			return nil
		}

		relPath, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}

		parts := strings.Split(relPath, "/")

		relPath = strings.TrimPrefix(relPath, parts[0]+"/")
		relPath = strings.TrimSuffix(relPath, file)

		metadata := &Metadata{
			Root: parts[0],
			Type: ext,
			Path: relPath,
		}

		if err := SystemForType(ext).GenerateMetadata(name, metadata); err != nil {
			ctx.Logger().Error("failed to generate metadata for resource", slog.String("path", p), slog.Any("error", err))
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
