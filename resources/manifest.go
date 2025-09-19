package resources

import (
	"encoding/json"
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

const (
	MetadataRoot = "root"
	MetadataPath = "path"
	MetadataType = "type"
)

var loadedManifest Manifest

var propertySetters = map[string]func(m *Metadata, v any){
	MetadataRoot: func(m *Metadata, v any) { m.Root = v.(string) },
	MetadataPath: func(m *Metadata, v any) { m.Path = v.(string) },
	MetadataType: func(m *Metadata, v any) { m.Type = v.(string) },
}

type (
	Manifest map[string]*Metadata
	Metadata struct {
		Root       string
		Path       string
		Type       string
		Properties map[string]any
	}
)

func (m Metadata) MarshalJSON() ([]byte, error) {
	kvp := map[string]any{
		MetadataRoot: m.Root,
		MetadataType: m.Type,
	}
	if m.Path != "" {
		kvp[MetadataPath] = m.Path
	}
	for k, v := range m.Properties {
		kvp[k] = v
	}
	return json.Marshal(kvp)
}

func (m *Metadata) UnmarshalJSON(data []byte) error {
	kvp := make(map[string]any)
	if err := json.Unmarshal(data, &kvp); err != nil {
		return err
	}
	m.Properties = make(map[string]any)
	for k, v := range kvp {
		if setter, exists := propertySetters[k]; exists {
			setter(m, v)
			continue
		}
		m.Properties[k] = v
	}
	return nil
}

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
		name := KeyFromPath(file)

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
