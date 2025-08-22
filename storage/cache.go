package storage

import "github.com/hajimehoshi/ebiten/v2"

// =================================================================
// Data storage
// =================================================================

var dataStore = NewStore[[]byte]()

type DataHandle StorageHandle[[]byte]

func (h DataHandle) Get() ([]byte, error) {
	data, err := dataStore.Get(h.String())
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (h DataHandle) IsValid() bool {
	_, exists := dataStore.items[h.String()]
	return exists
}

func (h DataHandle) String() string {
	return string(h)
}

// =================================================================
// Image storage
// =================================================================

var imageStore = NewStore[*ebiten.Image]()

type ImageHandle StorageHandle[*ebiten.Image]

func (h ImageHandle) Get() (*ebiten.Image, error) {
	img, err := imageStore.Get(h.String())
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (h ImageHandle) IsValid() bool {
	_, exists := imageStore.items[h.String()]
	return exists
}

func (h ImageHandle) String() string {
	return string(h)
}
