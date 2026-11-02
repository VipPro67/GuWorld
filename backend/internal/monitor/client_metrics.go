package monitor

import (
	"errors"
	"math"
	"sort"
	"sync"
	"time"
)

const (
	ClientMetricsMaxPlayers = 512
	ClientMetricsMaxWindows = 8
	ClientMetricsMaxEvents  = 24
	ClientMetricsMaxBuckets = 16
	clientMetricsMaxText    = 96
)

// RealtimeClientWindow is one aggregation interval uploaded by the Phaser
// client. Every field is optional on the wire; missing values decode as zero.
type RealtimeClientWindow struct {
	WindowStartedAt  int64   `json:"window_started_at"`
	WindowDurationMs int64   `json:"window_duration_ms"`
	Zone             string  `json:"zone"`
	Status           string  `json:"status"`
	TickRate         int     `json:"tick_rate"`
	Entities         int     `json:"entities"`
	Snapshots        int     `json:"snapshots"`
	SnapshotRateHz   float64 `json:"snapshot_rate_hz"`
	SnapshotGapAvgMs float64 `json:"snapshot_gap_avg_ms"`
	SnapshotGapP95Ms float64 `json:"snapshot_gap_p95_ms"`
	SnapshotGapP99Ms float64 `json:"snapshot_gap_p99_ms"`
	SnapshotGapMaxMs float64 `json:"snapshot_gap_max_ms"`
	SnapshotGapHist  []int64 `json:"snapshot_gap_hist"`
	RenderDelayAvgMs float64 `json:"render_delay_avg_ms"`
	HardSnaps        int     `json:"hard_snaps"`
	BlendCount       int     `json:"blend_count"`
	LatencyAvgMs     float64 `json:"latency_avg_ms"`
	LatencyMaxMs     float64 `json:"latency_max_ms"`
	LatencySamples   int     `json:"latency_samples"`
	LatencyClamped   int     `json:"latency_clamped"`
	JitterMs         float64 `json:"jitter_ms"`
	EstimatedLost    int     `json:"estimated_lost"`
	Stalls           int     `json:"stalls"`
	MessagesIn       int     `json:"messages_in"`
	BytesIn          int64   `json:"bytes_in"`
	BytesOut         int64   `json:"bytes_out"`
	InputsSent       int     `json:"inputs_sent"`
	Resyncs          int     `json:"resyncs"`
	Reconnects       int     `json:"reconnects"`
	StaleReconnects  int     `json:"stale_reconnects"`
}

// RealtimeClientEvent records why a connection lifecycle event happened, so a
// reconnect spike can be attributed without a second log pipeline.
type RealtimeClientEvent struct {
	Kind string `json:"kind"`
	Code string `json:"code,omitempty"`
	At   int64  `json:"at"`
}

// RealtimeClientReport is the latest upload for one player. The server fills
// identity fields; the client only supplies measurements.
type RealtimeClientReport struct {
	PlayerID   string                 `json:"player_id"`
	TabID      string                 `json:"tab_id,omitempty"`
	IP         string                 `json:"ip,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Network    string                 `json:"network,omitempty"`
	Zone       string                 `json:"zone"`
	ReceivedAt time.Time              `json:"received_at"`
	Windows    []RealtimeClientWindow `json:"windows"`
	Events     []RealtimeClientEvent  `json:"events"`
}

// ClientMetricsSnapshot is the admin read model.
type ClientMetricsSnapshot struct {
	Players      int                    `json:"players"`
	TotalReports int64                  `json:"total_reports"`
	Rejected     int64                  `json:"rejected_reports"`
	LastReceived *time.Time             `json:"last_received_at,omitempty"`
	Reports      []RealtimeClientReport `json:"reports"`
}

// ClientMetricsTracker keeps the newest report per player in a bounded map.
// Keys are player IDs resolved by auth middleware, never client-supplied text.
type ClientMetricsTracker struct {
	mu           sync.RWMutex
	reports      map[string]RealtimeClientReport
	order        []string
	totalReports int64
	rejected     int64
	lastReceived time.Time
}

var GlobalClientMetrics = NewClientMetricsTracker()

func NewClientMetricsTracker() *ClientMetricsTracker {
	return &ClientMetricsTracker{reports: make(map[string]RealtimeClientReport)}
}

// Record validates, clamps and stores one report. It replaces the previous
// report for the same player so memory stays bounded by player count.
func (t *ClientMetricsTracker) Record(report RealtimeClientReport) error {
	if report.PlayerID == "" {
		return errors.New("player id is required")
	}
	if err := sanitizeClientReport(&report); err != nil {
		return err
	}
	now := time.Now().UTC()
	if report.ReceivedAt.IsZero() || report.ReceivedAt.After(now.Add(time.Minute)) {
		report.ReceivedAt = now
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	key := report.PlayerID
	if _, exists := t.reports[key]; !exists {
		if len(t.reports) >= ClientMetricsMaxPlayers {
			t.evictOldestLocked()
		}
		t.order = append(t.order, key)
	}
	t.reports[key] = report
	t.totalReports++
	t.lastReceived = report.ReceivedAt
	return nil
}

func (t *ClientMetricsTracker) evictOldestLocked() {
	for len(t.order) > 0 {
		oldest := t.order[0]
		t.order = t.order[1:]
		if _, ok := t.reports[oldest]; ok {
			delete(t.reports, oldest)
			return
		}
	}
	// Fallback: map grew without order bookkeeping (should not happen).
	for key := range t.reports {
		delete(t.reports, key)
		return
	}
}

func (t *ClientMetricsTracker) RecordRejected() {
	t.mu.Lock()
	t.rejected++
	t.mu.Unlock()
}

// Snapshot returns reports newest-first for the admin monitor.
func (t *ClientMetricsTracker) Snapshot() ClientMetricsSnapshot {
	t.mu.RLock()
	reports := make([]RealtimeClientReport, 0, len(t.reports))
	for _, report := range t.reports {
		reports = append(reports, report)
	}
	total := t.totalReports
	rejected := t.rejected
	var last *time.Time
	if !t.lastReceived.IsZero() {
		value := t.lastReceived
		last = &value
	}
	t.mu.RUnlock()

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].ReceivedAt.After(reports[j].ReceivedAt)
	})
	return ClientMetricsSnapshot{
		Players:      len(reports),
		TotalReports: total,
		Rejected:     rejected,
		LastReceived: last,
		Reports:      reports,
	}
}

func sanitizeClientReport(report *RealtimeClientReport) error {
	report.TabID = clampText(report.TabID)
	report.IP = clampText(report.IP)
	report.UserAgent = clampText(report.UserAgent)
	report.Network = clampText(report.Network)
	report.Zone = clampText(report.Zone)

	if len(report.Windows) > ClientMetricsMaxWindows {
		report.Windows = report.Windows[:ClientMetricsMaxWindows]
	}
	for i := range report.Windows {
		window := &report.Windows[i]
		window.Zone = clampText(window.Zone)
		window.Status = clampText(window.Status)
		window.WindowStartedAt = maxInt64(window.WindowStartedAt, 0)
		window.WindowDurationMs = maxInt64(window.WindowDurationMs, 0)
		window.Snapshots = maxInt(window.Snapshots, 0)
		window.Entities = maxInt(window.Entities, 0)
		window.TickRate = minInt(maxInt(window.TickRate, 0), 240)
		window.HardSnaps = maxInt(window.HardSnaps, 0)
		window.BlendCount = maxInt(window.BlendCount, 0)
		window.LatencySamples = maxInt(window.LatencySamples, 0)
		window.LatencyClamped = maxInt(window.LatencyClamped, 0)
		window.EstimatedLost = maxInt(window.EstimatedLost, 0)
		window.Stalls = maxInt(window.Stalls, 0)
		window.MessagesIn = maxInt(window.MessagesIn, 0)
		window.InputsSent = maxInt(window.InputsSent, 0)
		window.Resyncs = maxInt(window.Resyncs, 0)
		window.Reconnects = maxInt(window.Reconnects, 0)
		window.StaleReconnects = maxInt(window.StaleReconnects, 0)
		window.BytesIn = maxInt64(window.BytesIn, 0)
		window.BytesOut = maxInt64(window.BytesOut, 0)
		window.SnapshotRateHz = sanitizeFloat(window.SnapshotRateHz)
		window.SnapshotGapAvgMs = sanitizeFloat(window.SnapshotGapAvgMs)
		window.SnapshotGapP95Ms = sanitizeFloat(window.SnapshotGapP95Ms)
		window.SnapshotGapP99Ms = sanitizeFloat(window.SnapshotGapP99Ms)
		window.SnapshotGapMaxMs = sanitizeFloat(window.SnapshotGapMaxMs)
		window.RenderDelayAvgMs = sanitizeFloat(window.RenderDelayAvgMs)
		window.LatencyAvgMs = sanitizeFloat(window.LatencyAvgMs)
		window.LatencyMaxMs = sanitizeFloat(window.LatencyMaxMs)
		window.JitterMs = sanitizeFloat(window.JitterMs)
		if len(window.SnapshotGapHist) > ClientMetricsMaxBuckets {
			window.SnapshotGapHist = window.SnapshotGapHist[:ClientMetricsMaxBuckets]
		}
		for j := range window.SnapshotGapHist {
			window.SnapshotGapHist[j] = maxInt64(window.SnapshotGapHist[j], 0)
		}
	}
	if len(report.Events) > ClientMetricsMaxEvents {
		report.Events = report.Events[len(report.Events)-ClientMetricsMaxEvents:]
	}
	for i := range report.Events {
		report.Events[i].Kind = clampText(report.Events[i].Kind)
		report.Events[i].Code = clampText(report.Events[i].Code)
		report.Events[i].At = maxInt64(report.Events[i].At, 0)
	}
	return nil
}

func sanitizeFloat(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	return value
}

func clampText(value string) string {
	if len(value) > clientMetricsMaxText {
		return value[:clientMetricsMaxText]
	}
	return value
}

func maxInt(value, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

func minInt(value, ceiling int) int {
	if value > ceiling {
		return ceiling
	}
	return value
}

func maxInt64(value, floor int64) int64 {
	if value < floor {
		return floor
	}
	return value
}
