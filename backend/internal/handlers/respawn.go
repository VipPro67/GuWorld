package handlers

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// ResolveRespawnPosition returns the nearest safe hub after death. The legacy
// arguments remain because multiple defeat flows share this helper.
func ResolveRespawnPosition(_ *sql.Tx, _ uuid.UUID, _ string, currentX, currentY int) (string, int, int) {
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
