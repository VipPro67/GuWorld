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

// ResolveRespawnPosition chooses a same-map HUB first, then a SAFE_ZONE, then a
// random point inside the KHU_VUC containing the death position. It never moves
// the player to another map layer.
func ResolveRespawnPosition(_ *sql.Tx, _ uuid.UUID, currentRegion string, currentX, currentY int) (string, int, int) {
	mapID := worldMapIDForRegion(currentRegion)
	if region, x, y, found := NearestRespawnPointOnMap(currentX, currentY, mapID); found {
		return region, x, y
	}
	if region, x, y, found := RandomRespawnPointInKhuVuc(currentRegion, currentX, currentY, mapID); found {
		return region, x, y
	}
	// Do not strand the player on another map if no same-layer KHU_VUC exists.
	return currentRegion, currentX, currentY
}

func RespawnLocationMessage(fromRegion, toRegion string, x, y int) string {
	name := toRegion
	if zone := ZoneByID(toRegion); zone != nil && zone.Name != "" {
		name = zone.Name
	}
	message := fmt.Sprintf("Hồi sinh tại %s (%d, %d).", name, x, y)
	mapID := worldMapIDForRegion(fromRegion)
	if _, _, _, found := NearestRespawnPointOnMap(x, y, mapID); !found {
		if zone := ZoneByID(toRegion); zone != nil && zone.ZoneType == ZoneTypeNamed {
			message += " Không có HUB hoặc SAFE zone trên tầng này; đạo hữu được hồi sinh ngẫu nhiên trong KHU_VUC."
		} else {
			message += " Không tìm thấy HUB, SAFE zone hoặc KHU_VUC hợp lệ trên tầng này; đạo hữu hồi sinh tại vị trí tử vong."
		}
	}
	return message
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
