package ws

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDashAcceptancePayloadPreservesZeroDirectionAxis(t *testing.T) {
	x, y := 1.0, 0.0
	encoded, err := json.Marshal(worldEventPayload{DashDirX: &x, DashDirY: &y, DashDurationTicks: 6})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"dash_dir_x":1`, `"dash_dir_y":0`, `"dash_duration_ticks":6`} {
		if !strings.Contains(string(encoded), field) {
			t.Fatalf("missing %s in %s", field, encoded)
		}
	}
}
