package worker

import "sync/atomic"

type BatchMetrics struct {
	TotalFlushes     int64
	SuccessfulFlushes int64
	FailedFlushes    int64
	ItemsProcessed   int64
	ItemsRetried     int64
	ItemsDropped     int64
	QueueDepth       int64
}

var Metrics = &BatchMetrics{}

func (m *BatchMetrics) RecordFlush(entityType string, count int, success bool) {
	atomic.AddInt64(&m.TotalFlushes, 1)
	atomic.AddInt64(&m.ItemsProcessed, int64(count))
	if success {
		atomic.AddInt64(&m.SuccessfulFlushes, 1)
	} else {
		atomic.AddInt64(&m.FailedFlushes, 1)
	}
}

func (m *BatchMetrics) RecordRetry(count int) {
	atomic.AddInt64(&m.ItemsRetried, int64(count))
}

func (m *BatchMetrics) RecordDrop(count int) {
	atomic.AddInt64(&m.ItemsDropped, int64(count))
}
