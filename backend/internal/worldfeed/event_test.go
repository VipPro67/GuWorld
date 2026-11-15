package worldfeed

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateRequiresCoordinateScope(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		wantErr bool
	}{
		{
			name:  "valid public event",
			event: Event{EventID: "event-1", EntityType: "map_cell", Action: "cleared", Scope: Scope{RegionID: "north", WorldMapID: "world_map"}},
		},
		{
			name:    "missing region",
			event:   Event{EventID: "event-1", EntityType: "map_cell", Action: "cleared", Scope: Scope{WorldMapID: "world_map"}},
			wantErr: true,
		},
		{
			name:    "missing map",
			event:   Event{EventID: "event-1", EntityType: "map_cell", Action: "cleared", Scope: Scope{RegionID: "north"}},
			wantErr: true,
		},
		{
			name:    "aperture requires instance",
			event:   Event{EventID: "event-1", EntityType: "map_cell", Action: "cleared", Scope: Scope{RegionID: "aperture_interior", WorldMapID: "world_map"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validate(tt.event); (err != nil) != tt.wantErr {
				t.Fatalf("validate() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeEnvelopeKeepsTargetsScoped(t *testing.T) {
	playerID := uuid.MustParse("7b7bb3a0-0f7e-4f15-993a-b22fe79f0cc2")
	payload := []byte(`{"deliveries":[{"event":{"event_id":"event-1","entity_type":"map_cell","action":"cleared","scope":{"region_id":"north","world_map_id":"world_map"},"x":4,"y":8},"targets":["` + playerID.String() + `"]}]}`)

	deliveries, err := DecodeEnvelope(payload)
	if err != nil {
		t.Fatalf("DecodeEnvelope() error = %v", err)
	}
	if len(deliveries) != 1 {
		t.Fatalf("DecodeEnvelope() deliveries = %d, want 1", len(deliveries))
	}
	delivery := deliveries[0]
	event, targets := delivery.Event, delivery.Targets
	if event.Scope.RegionID != "north" || event.X != 4 || event.Y != 8 {
		t.Fatalf("DecodeEnvelope() event = %+v", event)
	}
	if len(targets) != 1 || targets[0] != playerID {
		t.Fatalf("DecodeEnvelope() targets = %+v", targets)
	}
}
