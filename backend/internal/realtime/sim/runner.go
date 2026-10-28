package sim

import (
	"context"

	"github.com/vippro67/GuWorld/backend/internal/worker"
)

// Run starts one bounded 20Hz driver and returns a channel closed on shutdown.
// A zone's World must be owned by exactly one runner. Ordinary full outbound
// channels may coalesce snapshots; death snapshots apply backpressure so
// settlement events are not lost.
// The channel must remain open until ctx has stopped the runner.

func Run(ctx context.Context, world *World, snapshots chan<- Snapshot) <-chan struct{} {
	return worker.RunEvery(ctx, TickDuration, "Realtime World Zone", func(context.Context) {
		snapshot := world.Step()
		if snapshots != nil {
			hasDeath := false
			for _, event := range snapshot.Events {
				if event.Type == "ENTITY_DIED" {
					hasDeath = true
					break
				}
			}
			if hasDeath {
				// Deaths drive durable rewards and session settlement. Backpressure
				// the zone tick rather than silently dropping their snapshot.
				snapshots <- snapshot
			} else {
				select {
				case snapshots <- snapshot:
				default:
				}
			}
		}
	})
}
