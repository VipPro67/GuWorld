package worker

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var workerEventRecorder func(workerName string, err error, duration time.Duration)

// SetWorkerEventRecorder wires an external telemetry collector (e.g. monitor.RecordWorkerEvent).
func SetWorkerEventRecorder(fn func(workerName string, err error, duration time.Duration)) {
	workerEventRecorder = fn
}

// RunEvery starts a goroutine that calls fn on every tick until ctx is
// cancelled. A panic inside fn is logged and the loop keeps running instead of
// taking the whole process down with it.
func RunEvery(ctx context.Context, interval time.Duration, name string, fn func(context.Context)) <-chan struct{} {
	return RunEveryE(ctx, interval, name, func(c context.Context) error {
		fn(c)
		return nil
	})
}

// RunEveryE is like RunEvery but fn returns an error. Non-nil errors and panics
// are logged and dispatched to the worker event recorder.
func RunEveryE(ctx context.Context, interval time.Duration, name string, fn func(context.Context) error) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		logger.S.Infof("%s started (every %v)", name, interval)
		for {
			select {
			case <-ctx.Done():
				logger.S.Infof("%s stopped", name)
				return
			case <-ticker.C:
				SafeRunE(name, ctx, fn)
			}
		}
	}()
	return done
}

// SafeRun calls fn, turning a panic into a logged error. Use it for work that
// runs outside RunEvery: startup pre-runs, and loops that must flush on
// shutdown and therefore cannot use RunEvery.
func SafeRun(name string, ctx context.Context, fn func(context.Context)) {
	SafeRunE(name, ctx, func(c context.Context) error {
		fn(c)
		return nil
	})
}

// SafeRunE calls fn with error return. Panics and returned errors are logged
// and dispatched to the worker event recorder.
func SafeRunE(name string, ctx context.Context, fn func(context.Context) error) {
	start := time.Now()
	var runErr error
	defer func() {
		dur := time.Since(start)
		if p := recover(); p != nil {
			logger.S.Errorf("%s panicked: %v\n%s", name, p, debug.Stack())
			if workerEventRecorder != nil {
				workerEventRecorder(name, fmt.Errorf("panic: %v", p), dur)
			}
		} else if runErr != nil {
			logger.S.Errorf("%s failed: %v", name, runErr)
			if workerEventRecorder != nil {
				workerEventRecorder(name, runErr, dur)
			}
		}
	}()
	runErr = fn(ctx)
}
