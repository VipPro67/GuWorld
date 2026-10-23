package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

const (
	// HighRankRespawnThreshold is the cultivation rank (Ngũ Chuyển) from which
	// players get the random/sect respawn rules instead of always waking up at
	// the nearest HUB/SAFE_ZONE.
	HighRankRespawnThreshold = 5
	// SectRespawnChance is the chance for a high-rank sect member to respawn at
	// their sect hall instead of a random point in the death KHU_VUC.
	SectRespawnChance = 0.5
)

// RespawnKind describes why a respawn destination was chosen.
type RespawnKind int

const (
	RespawnHub RespawnKind = iota
	RespawnRandom
	RespawnSect
	RespawnStay
)

// RespawnResult is the destination and reason produced by the death respawn
// resolver. Message renders the player-facing log line for the outcome.
type RespawnResult struct {
	Region   string
	X        int
	Y        int
	Kind     RespawnKind
	SectName string
}

type respawnProfile struct {
	Rank       int
	HasSect    bool
	SectRegion string
	SectName   string
	SectX      int
	SectY      int
}

// ResolveRespawnPosition chooses a same-map HUB first, then a SAFE_ZONE, then a
// random point inside the KHU_VUC containing the death position. It never moves
// the player to another map layer.
//
// This is the voluntary-retreat rule. Deaths use ResolveDeathRespawn so the
// high-rank random/sect rules never leak into a normal retreat.
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

// ResolveDeathRespawn picks a death respawn destination:
//   - Rank < 5: nearest HUB/SAFE_ZONE (same map), then random KHU_VUC, then death spot.
//   - Rank >= 5 without a sect: random KHU_VUC, then nearest HUB/SAFE_ZONE, then death spot.
//   - Rank >= 5 with a sect: 50% sect hall (may be in another region), otherwise
//     the rank >= 5 random rule.
func ResolveDeathRespawn(tx *sql.Tx, playerID uuid.UUID, deathRegion string, x, y int) RespawnResult {
	return resolveDeathRespawn(fetchRespawnProfile(tx, playerID), rand.Float64(), deathRegion, x, y)
}

// fetchRespawnProfile loads the player's cultivation rank and sect coordinates
// in one query. A missing player/sect, or a nil test transaction, yields the
// zero profile and therefore the legacy low-rank behaviour.
func fetchRespawnProfile(tx *sql.Tx, playerID uuid.UUID) respawnProfile {
	if tx == nil || playerID == uuid.Nil {
		return respawnProfile{}
	}

	var profile respawnProfile
	var sectRegion, sectName sql.NullString
	var sectX, sectY sql.NullInt64
	err := tx.QueryRow(`
		SELECT COALESCE(p.rank, 0), s.region, s.name, s.map_x, s.map_y
		FROM players p
		LEFT JOIN sect_members sm ON sm.player_id = p.id
		LEFT JOIN sects s ON s.id = sm.sect_id
		WHERE p.id = $1
	`, playerID).Scan(&profile.Rank, &sectRegion, &sectName, &sectX, &sectY)
	if err != nil {
		return respawnProfile{}
	}

	if sectRegion.Valid && sectRegion.String != "" && sectName.Valid {
		profile.HasSect = true
		profile.SectRegion = sectRegion.String
		profile.SectName = sectName.String
		profile.SectX = int(sectX.Int64)
		profile.SectY = int(sectY.Int64)
	}
	return profile
}

// resolveDeathRespawn is the deterministic core (roll is injected for tests).
func resolveDeathRespawn(profile respawnProfile, roll float64, deathRegion string, x, y int) RespawnResult {
	mapID := worldMapIDForRegion(deathRegion)

	if profile.Rank >= HighRankRespawnThreshold {
		if sectRespawnAvailable(profile) && roll < SectRespawnChance {
			return RespawnResult{
				Region:   profile.SectRegion,
				X:        profile.SectX,
				Y:        profile.SectY,
				Kind:     RespawnSect,
				SectName: profile.SectName,
			}
		}
		if region, rx, ry, found := RandomRespawnPointInKhuVuc(deathRegion, x, y, mapID); found {
			return RespawnResult{Region: region, X: rx, Y: ry, Kind: RespawnRandom}
		}
		if region, rx, ry, found := NearestRespawnPointOnMap(x, y, mapID); found {
			return RespawnResult{Region: region, X: rx, Y: ry, Kind: RespawnHub}
		}
		return RespawnResult{Region: deathRegion, X: x, Y: y, Kind: RespawnStay}
	}

	if region, rx, ry, found := NearestRespawnPointOnMap(x, y, mapID); found {
		return RespawnResult{Region: region, X: rx, Y: ry, Kind: RespawnHub}
	}
	if region, rx, ry, found := RandomRespawnPointInKhuVuc(deathRegion, x, y, mapID); found {
		return RespawnResult{Region: region, X: rx, Y: ry, Kind: RespawnRandom}
	}
	// Do not strand the player on another map if no same-layer KHU_VUC exists.
	return RespawnResult{Region: deathRegion, X: x, Y: y, Kind: RespawnStay}
}

// sectRespawnAvailable rejects stale sect coordinates whose region no longer
// exists, since players.current_region has an FK and the respawn UPDATE would
// otherwise fail.
func sectRespawnAvailable(profile respawnProfile) bool {
	return profile.HasSect && profile.SectRegion != "" && ZoneByID(profile.SectRegion) != nil
}

// Message renders the player-facing respawn log line.
func (r RespawnResult) Message() string {
	name := r.Region
	if zone := ZoneByID(r.Region); zone != nil && zone.Name != "" {
		name = zone.Name
	}

	switch r.Kind {
	case RespawnSect:
		sect := r.SectName
		if sect == "" {
			sect = "Tông Môn"
		}
		return fmt.Sprintf("Hồi sinh tại sơn môn Tông Môn [%s] (%d, %d).", sect, r.X, r.Y)
	case RespawnRandom:
		return fmt.Sprintf("Hồi sinh ngẫu nhiên tại %s (%d, %d).", name, r.X, r.Y)
	case RespawnStay:
		return "Không tìm thấy HUB, SAFE zone hoặc KHU_VUC hợp lệ trên tầng này; đạo hữu hồi sinh tại vị trí tử vong."
	default:
		return fmt.Sprintf("Hồi sinh tại %s (%d, %d).", name, r.X, r.Y)
	}
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
