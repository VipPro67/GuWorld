package monitor

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestClientMetricsTrackerReplacesReportPerPlayer(t *testing.T) {
	tracker := NewClientMetricsTracker()
	first := RealtimeClientReport{PlayerID: "p1", Zone: "Southern_Border", Windows: []RealtimeClientWindow{{Snapshots: 10}}}
	if err := tracker.Record(first); err != nil {
		t.Fatalf("record first: %v", err)
	}
	second := RealtimeClientReport{PlayerID: "p1", Zone: "Southern_Border", Windows: []RealtimeClientWindow{{Snapshots: 20}}}
	if err := tracker.Record(second); err != nil {
		t.Fatalf("record second: %v", err)
	}

	snapshot := tracker.Snapshot()
	if snapshot.Players != 1 {
		t.Fatalf("players = %d, want 1", snapshot.Players)
	}
	if snapshot.TotalReports != 2 {
		t.Fatalf("total reports = %d, want 2", snapshot.TotalReports)
	}
	if got := snapshot.Reports[0].Windows[0].Snapshots; got != 20 {
		t.Fatalf("latest window snapshots = %d, want 20", got)
	}
}

func TestClientMetricsTrackerEvictsOldestPlayer(t *testing.T) {
	tracker := NewClientMetricsTracker()
	now := time.Now().UTC()
	for i := 0; i < ClientMetricsMaxPlayers+1; i++ {
		report := RealtimeClientReport{
			PlayerID:   fmt.Sprintf("p%03d", i),
			ReceivedAt: now.Add(time.Duration(i) * time.Millisecond),
		}
		if err := tracker.Record(report); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}

	snapshot := tracker.Snapshot()
	if snapshot.Players != ClientMetricsMaxPlayers {
		t.Fatalf("players = %d, want %d", snapshot.Players, ClientMetricsMaxPlayers)
	}
	for _, report := range snapshot.Reports {
		if report.PlayerID == "p000" {
			t.Fatalf("oldest player p000 should have been evicted")
		}
	}
}

func TestClientMetricsTrackerSanitizesNegativeAndNonFiniteValues(t *testing.T) {
	tracker := NewClientMetricsTracker()
	report := RealtimeClientReport{
		PlayerID: "p1",
		Windows: []RealtimeClientWindow{{
			Snapshots:       -3,
			BytesIn:         -10,
			TickRate:        10_000,
			LatencyAvgMs:    math.NaN(),
			SnapshotGapHist: []int64{-1, 5, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 9},
		}},
	}
	if err := tracker.Record(report); err != nil {
		t.Fatalf("record: %v", err)
	}

	got := tracker.Snapshot().Reports[0].Windows[0]
	if got.Snapshots != 0 || got.BytesIn != 0 {
		t.Fatalf("negative counters not clamped: %+v", got)
	}
	if got.TickRate != 240 {
		t.Fatalf("tick rate = %d, want 240", got.TickRate)
	}
	if got.LatencyAvgMs != 0 {
		t.Fatalf("NaN latency = %v, want 0", got.LatencyAvgMs)
	}
	if len(got.SnapshotGapHist) != ClientMetricsMaxBuckets {
		t.Fatalf("histogram len = %d, want %d", len(got.SnapshotGapHist), ClientMetricsMaxBuckets)
	}
	if got.SnapshotGapHist[0] != 0 {
		t.Fatalf("negative histogram bucket = %d, want 0", got.SnapshotGapHist[0])
	}
}

func TestClientMetricsTrackerRejectsMissingPlayer(t *testing.T) {
	tracker := NewClientMetricsTracker()
	if err := tracker.Record(RealtimeClientReport{}); err == nil {
		t.Fatal("expected error for missing player id")
	}
}
