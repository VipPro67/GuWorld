package monitor

import (
	"sync"
	"testing"
)

func TestAdminOperationTrackerRecordsAndResets(t *testing.T) {
	tracker := NewAdminOperationTracker()
	tracker.RecordForbidden("players.update")
	tracker.RecordForbidden("players.update")
	tracker.RecordFailedMutation("admin.update_player")
	tracker.RecordDestructiveAction("admin.delete_player_pet")
	tracker.RecordImportFailure("admin.import_table")
	tracker.RecordAuditWriteFailure()
	tracker.RecordLegacyAdminFallback()
	tracker.RecordPermissionResolveError()

	snapshot := tracker.Snapshot()
	if snapshot.ForbiddenByPermission["players.update"] != 2 ||
		snapshot.FailedMutationsByAction["admin.update_player"] != 1 ||
		snapshot.DestructiveByAction["admin.delete_player_pet"] != 1 ||
		snapshot.ImportFailuresByAction["admin.import_table"] != 1 ||
		snapshot.AuditWriteFailures != 1 || snapshot.LegacyAdminFallbacks != 1 ||
		snapshot.PermissionResolveErrors != 1 {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}

	// A caller cannot mutate tracker state through a returned map.
	snapshot.ForbiddenByPermission["players.update"] = 99
	if got := tracker.Snapshot().ForbiddenByPermission["players.update"]; got != 2 {
		t.Fatalf("snapshot map aliases tracker state: got %d", got)
	}

	tracker.Reset()
	reset := tracker.Snapshot()
	if len(reset.ForbiddenByPermission) != 0 || reset.AuditWriteFailures != 0 {
		t.Fatalf("reset left counters behind: %#v", reset)
	}
}

func TestAdminOperationTrackerConcurrentRecording(t *testing.T) {
	tracker := NewAdminOperationTracker()
	const workers = 20
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			tracker.RecordForbidden("audit.read")
			tracker.RecordAuditWriteFailure()
		}()
	}
	group.Wait()
	snapshot := tracker.Snapshot()
	if snapshot.ForbiddenByPermission["audit.read"] != workers || snapshot.AuditWriteFailures != workers {
		t.Fatalf("concurrent counters lost updates: %#v", snapshot)
	}
}
