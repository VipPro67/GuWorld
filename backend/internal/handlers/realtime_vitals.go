package handlers

import (
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// seedRealtimeHP makes the live MMO zone HP the base for an out-of-combat item
// or Gu action. Without this the handler would heal the stale legacy
// players.hp value while the HUD and checkpoint keep the zone value.
//
// It returns the zone HP used as the action baseline and whether the player is
// currently owned by a realtime zone. Dead zone entities are ignored so an item
// cannot silently revive a character through the legacy path.
func seedRealtimeHP(playerID uuid.UUID, player *models.Player) (float64, bool) {
	if player == nil {
		return 0, false
	}
	hp, _, ok := middleware.RealtimePlayerHP(playerID)
	if !ok {
		return 0, false
	}
	player.HP = hp
	return hp, true
}

// applyRealtimeHPDelta pushes the HP change produced by an item/Gu action back
// to the live zone entity after the database transaction commits.
func applyRealtimeHPDelta(playerID uuid.UUID, baseHP float64, active bool, player *models.Player) {
	if !active || player == nil {
		return
	}
	delta := player.HP - baseHP
	if delta == 0 {
		return
	}
	middleware.ApplyRealtimeHPDelta(playerID, delta)
}
