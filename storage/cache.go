package storage

import "github.com/hajimehoshi/ebiten/v2"

// =================================================================
// Data storage
// =================================================================

var dataStore = NewStore[[]byte]()

type DataHandle string

func (h DataHandle) Get() ([]byte, error) {
	data, err := dataStore.Get(string(h))
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (h DataHandle) IsValid() bool {
	_, exists := dataStore.items[string(h)]
	return exists
}

// =================================================================
// Image storage
// =================================================================

var imageStore = NewStore[*ebiten.Image]()

type ImageHandle string

func (h ImageHandle) Get() (*ebiten.Image, error) {
	img, err := imageStore.Get(string(h))
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (h ImageHandle) IsValid() bool {
	_, exists := imageStore.items[string(h)]
	return exists
}
