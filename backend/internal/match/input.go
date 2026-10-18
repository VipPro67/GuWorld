package match

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// MoveInput is a movement intent: a direction vector, not a destination. The
// server owns the speed and the collision response.
type MoveInput struct {
	DX float64 `json:"dx"`
	DY float64 `json:"dy"`
}

// CastInput is a cast intent. Which fields matter depends on the skill's
// CastMode: TARGET uses TargetID, DIRECTION uses AX/AY as a vector, GROUND uses
// AX/AY as a point, SELF ignores both.
type CastInput struct {
	GuID     string              `json:"gu_id"`
	Rank     int                 `json:"rank"`
	Mode     models.MobaCastMode `json:"mode"`
	AX       float64             `json:"ax"`
	AY       float64             `json:"ay"`
	TargetID string              `json:"target_id"`
}

// Input is one client frame. Casts are capped so a single frame cannot spam.
type Input struct {
	Seq   int         `json:"seq"`
	Move  *MoveInput  `json:"move,omitempty"`
	Casts []CastInput `json:"casts,omitempty"`
}

const maxCastsPerInput = 2

func ParseInput(payload json.RawMessage) (Input, error) {
	var input Input
	if err := json.Unmarshal(payload, &input); err != nil {
		return Input{}, fmt.Errorf("invalid match input: %w", err)
	}
	if len(input.Casts) > maxCastsPerInput {
		input.Casts = input.Casts[:maxCastsPerInput]
	}
	return input, nil
}

// SubmitInput buffers a client frame for the next tick. It never mutates
// authoritative state directly, so an out-of-tick burst cannot skip validation.
// Casts and move inputs arriving within the same tick are merged so high-frequency
// movement reports cannot wipe out an unconsumed cast.
func (m *Match) SubmitInput(playerID uuid.UUID, input Input) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Ended || m.find(playerID) == nil {
		return false
	}
	if prev, exists := m.pending[playerID]; exists {
		if len(input.Casts) == 0 && len(prev.Casts) > 0 {
			input.Casts = prev.Casts
		} else if len(input.Casts) > 0 && len(prev.Casts) > 0 {
			combined := append(prev.Casts, input.Casts...)
			if len(combined) > maxCastsPerInput {
				combined = combined[:maxCastsPerInput]
			}
			input.Casts = combined
		}
		if input.Move == nil && prev.Move != nil {
			input.Move = prev.Move
		}
	}
	m.pending[playerID] = input
	return true
}

// PlayerIDs returns the human participants, used to address broadcasts.
func (m *Match) PlayerIDs() []uuid.UUID {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]uuid.UUID, 0, len(m.Participants))
	for _, p := range m.Participants {
		if !p.IsDummy {
			ids = append(ids, p.ID)
		}
	}
	return ids
}
