package worker

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
)

var workerEventRecorder struct {
	sync.RWMutex
	fn func(workerName, workerRunID string, err error, duration time.Duration, stacktrace string)
}

// SetWorkerEventRecorder wires an external telemetry collector (e.g. monitor.RecordWorkerEvent).
func SetWorkerEventRecorder(fn func(workerName, workerRunID string, err error, duration time.Duration, stacktrace string)) {
	workerEventRecorder.Lock()
	workerEventRecorder.fn = fn
	workerEventRecorder.Unlock()
}

func recordWorkerEvent(name, runID string, err error, duration time.Duration, stacktrace string) {
	workerEventRecorder.RLock()
	recorder := workerEventRecorder.fn
	workerEventRecorder.RUnlock()
	if recorder != nil {
		recorder(name, runID, err, duration, stacktrace)
	}
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
	runID := uuid.NewString()
	var runErr error
	var stacktrace string
	defer func() {
		dur := time.Since(start)
		if p := recover(); p != nil {
			runErr = fmt.Errorf("panic: %v", p)
			stacktrace = string(debug.Stack())
		}
		if runErr != nil {
			fields := []zap.Field{
				zap.String("worker", name),
				zap.String("worker_run_id", runID),
				zap.Float64("duration_ms", float64(dur.Microseconds())/1000),
			}
			if stacktrace != "" {
				fields = append(fields, zap.String("stacktrace", stacktrace))
			}
			logger.LogError(ctx, "background_worker_failed", runErr, fields...)
			recordWorkerEvent(name, runID, runErr, dur, stacktrace)
		}
	}()
	runErr = fn(ctx)
}
