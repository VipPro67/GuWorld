package worker

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// RunEvery starts a goroutine that calls fn on every tick until ctx is
// cancelled. A panic inside fn is logged and the loop keeps running instead of
// taking the whole process down with it.
func RunEvery(ctx context.Context, interval time.Duration, name string, fn func(context.Context)) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		logger.S.Infof("%s started (every %v)", name, interval)
		for {
			select {
			case <-ctx.Done():
				logger.S.Infof("%s stopped", name)
				return
			case <-ticker.C:
				SafeRun(name, ctx, fn)
			}
		}
	}()
}

// SafeRun calls fn, turning a panic into a logged error. Use it for work that
// runs outside RunEvery: startup pre-runs, and loops that must flush on
// shutdown and therefore cannot use RunEvery.
func SafeRun(name string, ctx context.Context, fn func(context.Context)) {
	defer func() {
		if p := recover(); p != nil {
			logger.S.Errorf("%s panicked: %v\n%s", name, p, debug.Stack())
		}
	}()
	fn(ctx)
}
