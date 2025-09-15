package resources

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-core/linq"
)

const BatchSize = 100

type LoadRequest struct {
	key      string
	metadata *Metadata
}

func Load(ctx finch.Context, keys ...string) {
	if loadedManifest == nil {
		panic("resource manifest not loaded, cannot load resources")
	}

	if len(keys) == 0 {
		ctx.Logger().Warn("no resource keys provided to load")
		return
	}

	requests := make([]LoadRequest, 0)
	for _, key := range linq.Distinct(keys) {
		metadata, exists := loadedManifest[key]

		if !exists {
			ctx.Logger().Warn("cannot find metadata in manifest for", slog.String("key", key))
			continue
		}

		requests = append(requests, LoadRequest{
			key:      key,
			metadata: &metadata,
		})
	}

	if len(requests) == 0 {
		ctx.Logger().Warn("no valid resource keys provided to load")
		return
	}

	load_batches(ctx, linq.Batch(requests, BatchSize))
}

func load_batches(ctx finch.Context, batches [][]LoadRequest) {
	if len(batches) == 1 {
		load_batch(ctx, 1, batches[0])
		return
	}

	panicCh := make(chan error, len(batches))
	wg := sync.WaitGroup{}

	wg.Add(len(batches))
	for i, batch := range batches {
		go func(c finch.Context, id int, requests []LoadRequest) {
			defer wg.Done()

			defer func() {
				if r := recover(); r != nil {
					panicCh <- fmt.Errorf("panic in resource load batch %d: %v", id, r)
				}
			}()

			load_batch(c, id, requests)
		}(ctx, i+1, batch)
	}
	wg.Wait()

	close(panicCh)

	panics := make([]error, 0)
	for err := range panicCh {
		if err != nil {
			panics = append(panics, err)
		}
	}

	if len(panics) > 0 {
		panic(errors.Join(panics...))
	}
}

func load_batch(ctx finch.Context, id int, requests []LoadRequest) {
	if len(requests) == 0 {
		return
	}

	ctx.Logger().Info("loading resources", slog.Int("batch", id), slog.Int("size", len(requests)))

	for _, req := range requests {
		rt := req.metadata.Type

		sys := SystemForType(rt)
		if sys == nil {
			ctx.Logger().Warn("no resource system found for type, skipping", slog.Int("batch", id), slog.String("key", req.key), slog.String("type", rt))
			continue
		}

		if err := sys.Load(ctx, req.key, *req.metadata); err != nil {
			ctx.Logger().Error("error loading resource", slog.Int("batch", id), slog.String("key", req.key), slog.String("type", rt), slog.String("error", err.Error()))
			continue
		}
	}

	ctx.Logger().Info("finished loading resources", slog.Int("batch", id), slog.Int("size", len(requests)))
}
