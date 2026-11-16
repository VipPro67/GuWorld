package uihub

import (
	"encoding/json"

	"github.com/google/uuid"
)

// TargetedEvent is a Redis envelope for a client event that should reach only
// the listed players on every backend instance.
type TargetedEvent struct {
	Targets []uuid.UUID     `json:"targets"`
	Payload json.RawMessage `json:"payload"`
}
