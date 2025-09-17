package resources

import "io/fs"

var registry = make(map[string]fs.FS)

func AddFilesystem(name string, filesystem fs.FS) {
	if _, exists := registry[name]; exists {
		panic("filesystem already registered: " + name)
	}
	registry[name] = filesystem
}

func GetFilesystem(name string) fs.FS {
	if fsys, exists := registry[name]; exists {
		return fsys
	}
	return nil
}
