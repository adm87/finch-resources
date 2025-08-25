package storage

import "github.com/adm87/finch-core/errors"

// Store is a generic store for items of type T, identified by a string name.
type Store[T any] struct {
	items    map[string]T
	fallback string
}

func NewStore[T any]() *Store[T] {
	return &Store[T]{
		items: make(map[string]T),
	}
}

func (s *Store[T]) SetDefault(fallback string) {
	s.fallback = fallback
}

// Add adds an item to the store with the given name.
// Returns an error if the name is empty, the item is nil, or an item with the same name already exists.
func (s *Store[T]) Add(name string, item T) error {
	hasItem, err := s.Has(name)
	if err != nil {
		return err
	}
	if hasItem {
		return errors.NewDuplicateError("item already exists: " + name)
	}

	s.items[name] = item
	return nil
}

// Remove removes the item with the given name from the store.
// Returns an error if the name is empty or if the item does not exist.
func (s *Store[T]) Remove(name string) error {
	if name == "" {
		return errors.InvalidArgumentError("item name cannot be empty")
	}
	if _, exists := s.items[name]; !exists {
		return errors.NewNotFoundError("item not found: " + name)
	}

	delete(s.items, name)
	return nil
}

// Get retrieves the item with the given name from the store.
// Returns an error if the name is empty or if the item does not exist.
func (s *Store[T]) Get(name string) (T, error) {
	if name == "" {
		var zero T
		return zero, errors.InvalidArgumentError("item name cannot be empty")
	}
	item, exists := s.items[name]
	if exists {
		return item, nil
	}
	if s.fallback != "" {
		return s.Get(s.fallback)
	}
	var zero T
	return zero, errors.NewNotFoundError("item not found: " + name)
}

// Has checks if an item with the given name exists in the store.
// Returns an error if the name is empty.
func (s *Store[T]) Has(name string) (bool, error) {
	if name == "" {
		return false, errors.InvalidArgumentError("item name cannot be empty")
	}
	_, exists := s.items[name]
	return exists, nil
}
