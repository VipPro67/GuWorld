package ws

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
)

func TestWSActionMetricsTrackReceivedAndFinalOutcomes(t *testing.T) {
	client := &Client{}

	accepted := WSMessage{Type: protocol.TypeCast}
	client.beginWSAction(&accepted)
	client.completeWSAction(accepted, http.StatusAccepted, time.Millisecond, nil)

	rejected := WSMessage{Type: protocol.TypeAttack}
	client.beginWSAction(&rejected)
	client.completeWSAction(rejected, http.StatusTooManyRequests, time.Millisecond, json.RawMessage(`{"code":"realtime_input_rate_limited"}`))

	failed := WSMessage{Type: "command", Channel: "chat.send"}
	client.beginWSAction(&failed)
	client.completeWSAction(failed, http.StatusInternalServerError, 2*time.Millisecond, json.RawMessage(`{"code":"storage_unavailable"}`))

	stats := client.snapshotWSActions()
	if len(stats) != 3 {
		t.Fatalf("action metric count=%d, want 3: %+v", len(stats), stats)
	}
	byAction := make(map[string]struct {
		received, accepted, rejected, errors int64
		lastErrorCode                        string
	})
	for _, stat := range stats {
		byAction[stat.Action] = struct {
			received, accepted, rejected, errors int64
			lastErrorCode                        string
		}{stat.Received, stat.Accepted, stat.Rejected, stat.Errors, stat.LastErrorCode}
	}
	if got := byAction["world.cast"]; got.received != 1 || got.accepted != 1 {
		t.Fatalf("accepted action was not counted: %+v", got)
	}
	if got := byAction["world.attack"]; got.received != 1 || got.rejected != 1 || got.lastErrorCode != "realtime_input_rate_limited" {
		t.Fatalf("rejected action was not counted: %+v", got)
	}
	if got := byAction["command:chat.send"]; got.received != 1 || got.errors != 1 || got.lastErrorCode != "storage_unavailable" {
		t.Fatalf("failed command was not counted: %+v", got)
	}
}
