package resources

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/adm87/finch-core/finch"
	"github.com/adm87/finch-core/linq"
	"github.com/adm87/finch-core/types"
)

const BatchSize = 100

var batchIdCounter = 0

func next_batch_id() int {
	batchIdCounter++
	return batchIdCounter
}

func Load(ctx finch.Context, handles ...ResourceHandle) {
	if loadedManifest == nil {
		panic("resource manifest not loaded, cannot load resources")
	}

	if len(handles) == 0 {
		ctx.Logger().Warn("no resource provided to load")
		return
	}

	ctx.Logger().Info("resource load requested:", slog.Int("count", len(handles)))

	requests := make(types.HashSet[ResourceHandle])
	build_requests(ctx, requests, handles)

	if len(requests) == 0 {
		ctx.Logger().Warn("no valid resource requests built, nothing to load")
		return
	}

	load_batches(ctx, linq.Batch(requests.ToSlice(), BatchSize))
}

func build_requests(ctx finch.Context, requests types.HashSet[ResourceHandle], handles []ResourceHandle) {
	for _, handle := range handles {
		if _, exists := requests[handle]; exists {
			continue
		}

		requests.Add(handle)

		if dependencies := fetch_request_dependencies(ctx, handle); len(dependencies) > 0 {
			ctx.Logger().Info("resource dependencies found:", slog.String("key", handle.Key()), slog.Any("count", len(dependencies)))

			build_requests(ctx, requests, dependencies)
		}
	}
}

func fetch_request_dependencies(ctx finch.Context, handle ResourceHandle) []ResourceHandle {
	metadata, exists := handle.Metadata()
	if !exists {
		return nil
	}

	sys := SystemForType(metadata.Type)

	if sys == nil {
		ctx.Logger().Warn("cannot find resource system for type:", slog.String("type", metadata.Type))
		return nil
	}

	return sys.GetDependencies(ctx, handle)
}

func load_batches(ctx finch.Context, batches [][]ResourceHandle) {
	if len(batches) == 1 {
		load_batch(ctx, next_batch_id(), batches[0])
		return
	}

	panicCh := make(chan error, len(batches))
	wg := sync.WaitGroup{}

	wg.Add(len(batches))
	for _, batch := range batches {
		go func(c finch.Context, id int, requests []ResourceHandle) {
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

func load_batch(ctx finch.Context, id int, requests []ResourceHandle) {
	if len(requests) == 0 {
		return
	}

	ctx.Logger().Info("resource loading begin:", slog.Int("batch", id), slog.Int("count", len(requests)))

	success := 0
	skipped := 0
	failed := 0

	for _, req := range requests {
		metadata, exists := req.Metadata()
		if !exists {
			ctx.Logger().Warn("cannot find metadata in manifest:", slog.String("key", req.Key()), slog.Int("batch", id))
			skipped++
			continue
		}

		sys := SystemForType(metadata.Type)

		if sys == nil {
			ctx.Logger().Warn("cannot find resource system for type:", slog.String("type", metadata.Type), slog.String("key", req.Key()), slog.Int("batch", id))
			skipped++
			continue
		}

		if err := sys.Load(ctx, req); err != nil {
			ctx.Logger().Error("error loading resource:", slog.String("type", metadata.Type), slog.String("key", req.Key()), slog.Int("batch", id), slog.String("error", err.Error()))
			failed++
			continue
		}

		success++
	}

	ctx.Logger().Info("resource loading finished:", slog.Int("batch", id), slog.Int("count", len(requests)), slog.Int("success", success), slog.Int("skipped", skipped), slog.Int("failed", failed))
}
