package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// RespawnKind describes why a respawn destination was chosen.
type RespawnKind int

const (
	RespawnHub RespawnKind = iota
	RespawnStay
)

// RespawnResult is the destination and reason produced by the death respawn
// resolver. Message renders the player-facing log line for the outcome.
type RespawnResult struct {
	Region string
	X      int
	Y      int
	Kind   RespawnKind
}

// ResolveRespawnPosition chooses a same-map HUB first, then a SAFE_ZONE, then the current position. It never moves the player to another map layer.
//
// This is the voluntary-retreat rule. Deaths use ResolveDeathRespawn so the
// respawn destination rules never drift.
func ResolveRespawnPosition(_ *sql.Tx, _ uuid.UUID, currentRegion string, currentX, currentY int) (string, int, int) {
	mapID := worldMapIDForRegion(currentRegion)
	if region, x, y, found := NearestRespawnPointOnMap(currentX, currentY, mapID); found {
		return region, x, y
	}
	// Do not strand the player on another map if no same-layer KHU_VUC exists.
	return currentRegion, currentX, currentY
}

// ResolveDeathRespawn picks a death respawn destination: the nearest HUB/SAFE
// zone on the death map, falling back to the death spot when none exists.
func ResolveDeathRespawn(deathRegion string, x, y int) RespawnResult {
	mapID := worldMapIDForRegion(deathRegion)
	if region, rx, ry, found := NearestRespawnPointOnMap(x, y, mapID); found {
		return RespawnResult{Region: region, X: rx, Y: ry, Kind: RespawnHub}
	}
	return RespawnResult{Region: deathRegion, X: x, Y: y, Kind: RespawnStay}
}

// Message renders the player-facing respawn log line.
func (r RespawnResult) Message() string {
	if r.Kind == RespawnStay {
		return "Không tìm thấy HUB hoặc SAFE zone hợp lệ trên tầng này; đạo hữu hồi sinh tại vị trí tử vong."
	}
	name := r.Region
	if zone := ZoneByID(r.Region); zone != nil && zone.Name != "" {
		name = zone.Name
	}
	return fmt.Sprintf("Hồi sinh tại %s (%d, %d).", name, r.X, r.Y)
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
