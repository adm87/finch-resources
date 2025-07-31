package manifest

import (
	"github.com/adm87/finch-core/errors"
	"github.com/adm87/finch-core/hash"
)

func ValidateManifest(m ResourceManifest) error {
	nameChecked := hash.HashSet[string]{}
	pathChecked := hash.HashSet[string]{}
	for key, metadata := range m {
		if key == "" {
			return errors.NewInvalidArgumentError("resource key must not be empty")
		}
		if metadata.Root == "" {
			return errors.NewInvalidArgumentError("resource root must not be empty")
		}
		if metadata.Path == "" {
			return errors.NewInvalidArgumentError("resource path must not be empty")
		}
		if metadata.Size < 0 {
			return errors.NewInvalidArgumentError("resource size must not be negative")
		}
		if metadata.Root == metadata.Path {
			return errors.NewInvalidArgumentError("resource root and path must not be the same")
		}

		if nameChecked.Contains(key) {
			return errors.NewDuplicateError("resource key must be unique: " + key)
		}
		nameChecked.Add(key)

		if pathChecked.Contains(metadata.Path) {
			return errors.NewDuplicateError("resource path must be unique: " + metadata.Path)
		}
		pathChecked.Add(metadata.Path)
	}
	return nil
}
