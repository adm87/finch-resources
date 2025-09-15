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

var batchIdCounter = 0

func next_batch_id() int {
	batchIdCounter++
	return batchIdCounter
}

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

	ctx.Logger().Info("resource load requested:", slog.Int("count", len(keys)))

	requests := make(map[string]LoadRequest)
	build_requests(ctx, requests, linq.Distinct(keys))

	r := linq.Values(requests)

	if len(r) == 0 {
		ctx.Logger().Warn("no valid resource keys provided to load")
		return
	}

	load_batches(ctx, linq.Batch(r, BatchSize))
}

func build_requests(ctx finch.Context, requests map[string]LoadRequest, key []string) {
	for _, k := range key {
		if _, exists := requests[k]; exists {
			continue
		}

		metadata, exists := loadedManifest[k]

		if !exists {
			ctx.Logger().Warn("cannot find metadata in manifest:", slog.String("key", k))
			continue
		}

		requests[k] = LoadRequest{
			key:      k,
			metadata: metadata,
		}

		if dependencies := fetch_request_dependencies(ctx, k, metadata); len(dependencies) > 0 {
			ctx.Logger().Info("resource dependencies found:", slog.String("key", k), slog.Any("dependencies", dependencies))

			build_requests(ctx, requests, dependencies)
		}
	}
}

func fetch_request_dependencies(ctx finch.Context, key string, metadata *Metadata) []string {
	if metadata == nil {
		return nil
	}

	sys := SystemForType(metadata.Type)

	if sys == nil {
		ctx.Logger().Warn("cannot find resource system for type:", slog.String("type", metadata.Type))
		return nil
	}

	return sys.GetDependencies(ctx, key, metadata)
}

func load_batches(ctx finch.Context, batches [][]LoadRequest) {
	if len(batches) == 1 {
		load_batch(ctx, next_batch_id(), batches[0])
		return
	}

	panicCh := make(chan error, len(batches))
	wg := sync.WaitGroup{}

	wg.Add(len(batches))
	for _, batch := range batches {
		go func(c finch.Context, id int, requests []LoadRequest) {
			defer wg.Done()

			defer func() {
				if r := recover(); r != nil {
					panicCh <- fmt.Errorf("panic in resource load batch %d: %v", id, r)
				}
			}()

			load_batch(c, id, requests)
		}(ctx, next_batch_id(), batch)
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

	ctx.Logger().Info("resource loading begin:", slog.Int("batch", id), slog.Int("count", len(requests)))

	success := 0
	skipped := 0
	failed := 0

	for _, req := range requests {
		rt := req.metadata.Type

		sys := SystemForType(rt)
		if sys == nil {
			ctx.Logger().Warn("cannot find resource system for type:", slog.String("type", rt), slog.String("key", req.key), slog.Int("batch", id))
			skipped++
			continue
		}

		if err := sys.Load(ctx, req.key, req.metadata); err != nil {
			ctx.Logger().Error("error loading resource:", slog.String("type", rt), slog.String("key", req.key), slog.Int("batch", id), slog.String("error", err.Error()))
			failed++
			continue
		}

		success++
	}

	ctx.Logger().Info("resource loading finished:", slog.Int("batch", id), slog.Int("count", len(requests)), slog.Int("success", success), slog.Int("skipped", skipped), slog.Int("failed", failed))
}
