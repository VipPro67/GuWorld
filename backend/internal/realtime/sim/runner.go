package sim

import (
	"context"

	"github.com/vippro67/GuWorld/backend/internal/worker"
)

// Run starts one bounded 20Hz driver and returns a channel closed on shutdown.
// A zone's World must be owned by exactly one runner. A full outbound channel
// drops a snapshot without blocking zone ticks; the manager fans available
// snapshots out through bounded/coalescing feeds.
// The channel must remain open until ctx has stopped the runner.

func Run(ctx context.Context, world *World, snapshots chan<- Snapshot) <-chan struct{} {
	return worker.RunEvery(ctx, TickDuration, "Realtime World Zone", func(context.Context) {
		snapshot := world.Step()
		if snapshots != nil {
			select {
			case snapshots <- snapshot:
			default:
			}
		}
	})
}
