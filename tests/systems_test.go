package main

import (
	"errors"
	"fmt"
	"resources"
	"resources/tests/data"
	"testing"
)

type DataManager struct {
	cache map[resources.Asset][]byte
}

func NewDataManager() *DataManager {
	return &DataManager{
		cache: make(map[resources.Asset][]byte),
	}
}

func (m *DataManager) Load(asset resources.Asset) resources.ResourceLoader {
	return func(ctx *resources.LoaderContext) error {
		lock := ctx.Lock(asset)
		defer lock.Release()

		if _, exists := m.cache[asset]; exists {
			return resources.ErrSkipped
		}

		data, err := ctx.ReadBytes(asset)
		if err != nil {
			return err
		}
		m.cache[asset] = data

		return nil
	}
}

func assertPanicIs(t *testing.T, expected error, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic %v, got none", expected)
		}
		var err error
		switch v := r.(type) {
		case error:
			err = v
		default:
			err = fmt.Errorf("%v", v)
		}
		if !errors.Is(err, expected) {
			t.Fatalf("expected panic %v, got %v", expected, err)
		}
	}()
	fn()
}

func TestLoadOne(t *testing.T) {
	manager := NewDataManager()

	if err := data.Kenney.Load(manager.Load(data.Tile0000)); err != nil {
		t.Fatalf("failed to load asset: %v", err)
	}

	if len(manager.cache) != 1 {
		t.Fatalf("expected 1 asset loaded, got %d", len(manager.cache))
	}

	if _, exists := manager.cache[data.Tile0000]; !exists {
		t.Fatalf("expected asset Tile0000 to be loaded")
	}
}

func TestLoadTen(t *testing.T) {
	manager := NewDataManager()

	assets := make([]resources.Asset, 0, 10)
	for key := range data.KenneyManifest {
		assets = append(assets, key)
		if len(assets) >= 10 {
			break
		}
	}

	loaders := make([]resources.ResourceLoader, 0, len(assets))
	for _, asset := range assets {
		loaders = append(loaders, manager.Load(asset))
	}

	if err := data.Kenney.Load(loaders...); err != nil {
		t.Fatalf("failed to load assets: %v", err)
	}

	if len(manager.cache) != 10 {
		t.Fatalf("expected 10 assets loaded, got %d", len(manager.cache))
	}

	for _, asset := range assets {
		if _, exists := manager.cache[asset]; !exists {
			t.Fatalf("expected asset %v to be loaded", asset)
		}
	}
}

func TestLoadHundred(t *testing.T) {
	manager := NewDataManager()

	assets := make([]resources.Asset, 0, 100)
	for key := range data.KenneyManifest {
		assets = append(assets, key)
		if len(assets) >= 100 {
			break
		}
	}

	loaders := make([]resources.ResourceLoader, 0, len(assets))
	for _, asset := range assets {
		loaders = append(loaders, manager.Load(asset))
	}

	if err := data.Kenney.Load(loaders...); err != nil {
		t.Fatalf("failed to load assets: %v", err)
	}

	if len(manager.cache) != len(assets) {
		t.Fatalf("expected %d assets loaded, got %d", len(assets), len(manager.cache))
	}

	for _, asset := range assets {
		if _, exists := manager.cache[asset]; !exists {
			t.Fatalf("expected asset %v to be loaded", asset)
		}
	}
}

func TestLoadAll(t *testing.T) {
	manager := NewDataManager()

	loaders := make([]resources.ResourceLoader, 0, len(data.KenneyManifest))
	for asset := range data.KenneyManifest {
		loaders = append(loaders, manager.Load(asset))
	}

	if err := data.Kenney.Load(loaders...); err != nil {
		t.Fatalf("failed to load assets: %v", err)
	}

	if len(manager.cache) != len(data.KenneyManifest) {
		t.Fatalf("expected %d assets loaded, got %d", len(data.KenneyManifest), len(manager.cache))
	}

	for asset := range data.KenneyManifest {
		if _, exists := manager.cache[asset]; !exists {
			t.Fatalf("expected asset %v to be loaded", asset)
		}
	}
}

func TestMissingFilesystem(t *testing.T) {
	kenneyFS := data.Kenney.Filesystem()
	t.Cleanup(func() {
		data.Kenney.UseFilesystem(kenneyFS)
	})

	// Set to nil for the test
	data.Kenney.UseFilesystem(nil)

	manager := NewDataManager()

	err := data.Kenney.Load(manager.Load(data.Tile0000))

	if err == nil {
		t.Fatalf("expected error when loading with nil filesystem, got nil")
	}

	if !errors.Is(err, resources.ErrMissingFilesystem) {
		t.Fatalf("expected %v, got %v", resources.ErrMissingFilesystem, err)
	}
}

func TestAssetNotInFilesystem(t *testing.T) {
	manager := NewDataManager()

	nilAsset := resources.Asset(^uint64(0))

	err := data.Kenney.Load(manager.Load(nilAsset))

	if err == nil {
		t.Fatalf("expected error when loading non-existent asset, got nil")
	}

	if !errors.Is(err, resources.ErrNotInFilesystem) {
		t.Fatalf("expected %v, got %v", resources.ErrNotInFilesystem, err)
	}
}

func TestNestedLockPanics(t *testing.T) {
	assertPanicIs(t, resources.ErrBatchAlreadyLocked, func() {
		data.Kenney.Load(func(ctx *resources.LoaderContext) error {
			lock1 := ctx.Lock(data.Tile0000)
			defer lock1.Release()

			_ = ctx.Lock(data.Tile0001) // must panic
			return nil
		})
	})
}

func TestDoubleReleasePanics(t *testing.T) {
	assertPanicIs(t, resources.ErrDoubleRelease, func() {
		data.Kenney.Load(func(ctx *resources.LoaderContext) error {
			lock := ctx.Lock(data.Tile0000)
			lock.Release()

			lock.Release() // must panic
			return nil
		})
	})
}
