package handlers

import (
	"database/sql"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type statusExecRecorder struct {
	query string
	args  []any
}

func (r *statusExecRecorder) Exec(query string, args ...any) (sql.Result, error) {
	r.query = query
	r.args = append([]any(nil), args...)
	return nil, nil
}

func TestApplyPlayerBattleStartStatusOperationRevokesRegisteredStatus(t *testing.T) {
	const statusID = "regional_wall_suppression"
	player := &models.Player{}
	player.Statuses.Add(statusID, "test", 0, -0.2)
	if err := applyPlayerBattleStartStatusOperation(player, gameplayeffects.CompiledOperation{
		Type: gameplayeffects.OpRevokeStatus, Trigger: gameplayeffects.TriggerOnBattleStart, StatusID: statusID,
	}); err != nil {
		t.Fatalf("apply status revoke: %v", err)
	}
	if player.Statuses.Has(statusID) {
		t.Fatal("REVOKE_STATUS did not remove the registered persistent status")
	}
}

func TestApplyPlayerBattleStartStatusOperationRejectsUnknownStatus(t *testing.T) {
	err := applyPlayerBattleStartStatusOperation(&models.Player{}, gameplayeffects.CompiledOperation{
		Type: gameplayeffects.OpRevokeStatus, Trigger: gameplayeffects.TriggerOnBattleStart, StatusID: "missing_status",
	})
	if err == nil {
		t.Fatal("unknown persistent status was silently accepted")
	}
}

func TestPersistBattleStartStatusesWritesRevokedStatusList(t *testing.T) {
	recorder := &statusExecRecorder{}
	player := &models.Player{}
	player.Statuses.Add("regional_wall_suppression", "test", 0, -0.2)
	player.Statuses.Remove("regional_wall_suppression")
	if err := persistBattleStartStatuses(recorder, player); err != nil {
		t.Fatalf("persist battle-start statuses: %v", err)
	}
	if recorder.query != `UPDATE players SET statuses = $1 WHERE id = $2` || len(recorder.args) != 2 {
		t.Fatalf("unexpected status persistence query: %q args=%v", recorder.query, recorder.args)
	}
	encoded, ok := recorder.args[0].(string)
	if !ok || encoded != "[]" {
		t.Fatalf("persisted statuses = %#v, want empty status array", recorder.args[0])
	}
}
