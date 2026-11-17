package worker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunEveryRecoversFromPanic(t *testing.T) {
	var ticks int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	RunEvery(ctx, 5*time.Millisecond, "test", func(context.Context) {
		atomic.AddInt32(&ticks, 1)
		panic("boom")
	})

	time.Sleep(80 * time.Millisecond)
	if got := atomic.LoadInt32(&ticks); got < 3 {
		t.Fatalf("worker stopped after panic: only %d ticks", got)
	}
}

func TestRunEveryStopsOnContextCancel(t *testing.T) {
	var ticks int32
	ctx, cancel := context.WithCancel(context.Background())

	RunEvery(ctx, 5*time.Millisecond, "test", func(context.Context) {
		atomic.AddInt32(&ticks, 1)
	})
	time.Sleep(40 * time.Millisecond)
	cancel()
	time.Sleep(30 * time.Millisecond)
	after := atomic.LoadInt32(&ticks)
	time.Sleep(40 * time.Millisecond)
	if got := atomic.LoadInt32(&ticks); got != after {
		t.Fatalf("worker kept ticking after cancel: %d -> %d", after, got)
	}
}

func TestSafeRunRecovers(t *testing.T) {
	called := false
	SafeRun("test", context.Background(), func(context.Context) {
		called = true
		panic("boom")
	})
	if !called {
		t.Fatal("SafeRun did not call fn")
	}
}

func TestSafeRunEAndEventRecorder(t *testing.T) {
	var recordedName string
	var recordedRunID string
	var recordedErr error
	var recordedStack string
	SetWorkerEventRecorder(func(workerName, workerRunID string, err error, duration time.Duration, stacktrace string) {
		recordedName = workerName
		recordedRunID = workerRunID
		recordedErr = err
		recordedStack = stacktrace
	})
	defer SetWorkerEventRecorder(nil)

	// Test error recording
	SafeRunE("test-worker-err", context.Background(), func(context.Context) error {
		return errors.New("something went wrong")
	})
	if recordedName != "test-worker-err" || recordedRunID == "" || recordedErr == nil || recordedErr.Error() != "something went wrong" {
		t.Fatalf("Expected recorded error, got name=%s run_id=%s err=%v", recordedName, recordedRunID, recordedErr)
	}

	// Test panic recording
	SafeRunE("test-worker-panic", context.Background(), func(context.Context) error {
		panic("disaster")
	})
	if recordedName != "test-worker-panic" || recordedErr == nil || recordedErr.Error() != "panic: disaster" {
		t.Fatalf("Expected recorded panic, got name=%s err=%v", recordedName, recordedErr)
	}
	if recordedStack == "" {
		t.Fatal("expected panic stacktrace to be recorded")
	}
}
