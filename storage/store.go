package storage

import "github.com/adm87/finch-core/errors"

// Store is a generic store for items of type T, identified by a string name.
type Store[T any] struct {
	items map[string]*T
}

// Add adds an item to the store.
//
// Panics if the name is empty, if the item already exists, or if the item is nil.
func (s *Store[T]) Add(name string, item *T) {
	if s.Has(name) {
		panic(errors.NewDuplicateError("item already exists: " + name))
	}
	if item == nil {
		panic(errors.NewNilError("item cannot be nil"))
	}
	s.items[name] = item
}

// Remove removes an item from the store by its name.
//
// Panics if the name is empty.
func (s *Store[T]) Remove(name string) {
	if s.Has(name) {
		delete(s.items, name)
	}
}

// Get retrieves an item from the store by its name.
//
// Panics if the name is empty or if the item does not exist.
func (s *Store[T]) Get(name string) (*T, error) {
	if s.Has(name) {
		return s.items[name], nil
	}
	return nil, errors.NewNotFoundError("item not found: " + name)
}

// Has checks if an item exists in the store by its name.
//
// Panics if the name is empty.
func (s *Store[T]) Has(name string) bool {
	if name == "" {
		panic(errors.InvalidArgumentError("item name cannot be empty"))
	}
	_, exists := s.items[name]
	return exists
}
