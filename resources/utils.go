package resources

import (
	"embed"
	"errors"
	"io"
	"path"
	"strings"

	"github.com/adm87/finch-core/finch"
)

func LoadData(ctx finch.Context, key string, metadata *Metadata) ([]byte, error) {
	sys := GetFilesystem(metadata.Root)
	if sys == nil {
		return nil, errors.New("no filesystem found for root: " + metadata.Root)
	}

	// Note: if the filesystem is an embed.FS, we need to join the root and path

	fpath := path.Join(strings.Trim(metadata.Path, "/"), key) + "." + metadata.Type
	if _, ok := sys.(embed.FS); ok {
		fpath = path.Join(metadata.Root, fpath)
	}

	file, err := sys.Open(fpath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func KeyFromPath(p string) string {
	ext := path.Ext(p)
	base := path.Base(p)
	return strings.TrimSuffix(base, ext)
}
