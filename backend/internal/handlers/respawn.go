package handlers

import (
	"database/sql"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// SectRespawnChance is the probability that a sect member respawns at their sect base.
// The remaining chance falls back to random coordinates in the current region.
const SectRespawnChance = 0.50

// ResolveRespawnPosition resolves the respawn location for a deceased player.
// Priority:
// 1. Player's sect base location (probabilistic: 50% chance if player belongs to a sect).
// 2. Random valid coordinates within the player's current region (region stays unchanged).
// 3. Fallback to NearestHub if region geometry is unavailable.
func ResolveRespawnPosition(tx *sql.Tx, playerID uuid.UUID, currentRegion string, currentX, currentY int) (string, int, int) {
	// 1. Check if player belongs to a sect with SectRespawnChance
	if tx != nil && rand.Float64() < SectRespawnChance {
		var sectRegion string
		var sectX, sectY int
		err := tx.QueryRow(`
			SELECT s.region, s.map_x, s.map_y
			FROM sect_members sm
			JOIN sects s ON sm.sect_id = s.id
			WHERE sm.player_id = $1
		`, playerID).Scan(&sectRegion, &sectX, &sectY)

		if err == nil && sectRegion != "" && sectX > 0 && sectY > 0 {
			return sectRegion, sectX, sectY
		}
	}

	// 2. Tán Tu / No Sect: Spawn randomly within the current region
	if currentRegion != "" {
		if z := ZoneByID(currentRegion); z != nil && z.MaxX > z.MinX && z.MaxY > z.MinY {
			rx := z.MinX + rand.Intn(z.MaxX-z.MinX+1)
			ry := z.MinY + rand.Intn(z.MaxY-z.MinY+1)
			if rx < 0 {
				rx = 0
			}
			if rx >= WorldWidth {
				rx = WorldWidth - 1
			}
			if ry < 0 {
				ry = 0
			}
			if ry >= WorldHeight {
				ry = WorldHeight - 1
			}
			return currentRegion, rx, ry
		}

		// If current coordinates are inside world, jitter within +/- 50 cells
		if inWorld(currentX, currentY) {
			rx := currentX + rand.Intn(101) - 50
			ry := currentY + rand.Intn(101) - 50
			if rx < 0 {
				rx = 0
			}
			if rx >= WorldWidth {
				rx = WorldWidth - 1
			}
			if ry < 0 {
				ry = 0
			}
			if ry >= WorldHeight {
				ry = WorldHeight - 1
			}
			return currentRegion, rx, ry
		}
	}

	// 3. Absolute fallback to nearest hub
	return NearestHub(currentX, currentY)
}

// GrantPvpImmunity grants the pvp_immunity status to a player upon respawning.
func GrantPvpImmunity(tx *sql.Tx, playerID uuid.UUID, duration time.Duration) {
	if duration <= 0 {
		duration = 10 * time.Minute
	}

	var rawStatuses []byte
	err := tx.QueryRow(`SELECT statuses FROM players WHERE id = $1 FOR UPDATE`, playerID).Scan(&rawStatuses)
	if err != nil {
		return
	}

	var sl models.StatusList
	if len(rawStatuses) > 0 {
		_ = json.Unmarshal(rawStatuses, &sl)
	}

	sl.Add("pvp_immunity", "respawn", duration, 1.0)
	newBytes, _ := json.Marshal(sl)
	_, _ = tx.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, newBytes, playerID)

	notify.Send(tx, playerID, "pvp_immunity", "Thiên Đạo Hộ Thể!",
		"Bạn vừa hồi sinh và được Thiên Đạo ban cho trạng thái Miễn Chiến PvP trong 10 phút. Nếu chủ động tấn công người khác, trạng thái này sẽ bị phá vỡ!")
}

// RemovePvpImmunity removes pvp_immunity when a player actively initiates PvP.
func RemovePvpImmunity(tx *sql.Tx, playerID uuid.UUID) {
	var rawStatuses []byte
	err := tx.QueryRow(`SELECT statuses FROM players WHERE id = $1 FOR UPDATE`, playerID).Scan(&rawStatuses)
	if err != nil || len(rawStatuses) == 0 {
		return
	}

	var sl models.StatusList
	if err := json.Unmarshal(rawStatuses, &sl); err != nil {
		return
	}

	if sl.Has("pvp_immunity") {
		sl.Remove("pvp_immunity")
		newBytes, _ := json.Marshal(sl)
		_, _ = tx.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, newBytes, playerID)
		notify.Send(tx, playerID, "pvp_immunity_broken", "Phá vỡ Miễn Chiến!",
			"Đạo hữu vừa chủ động khai chiến, trạng thái Thiên Đạo Miễn Chiến đã bị tiêu tán!")
	}
}
