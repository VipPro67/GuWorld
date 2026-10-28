package handlers

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// GetRegionalWallDebuff calculates the debuff percentage (negative float64) and duration
// based on player rank and sub-rank when crossing between different NGU_VUC regions.
// Higher rank and sub-rank result in harsher stat suppression ("rank càng cao giảm càng nhiều").
func GetRegionalWallDebuff(rank int, subRank int) (float64, time.Duration) {
	return gamefx.RegionalWallDebuff(rank, subRank)
}

// CheckAndApplyRegionalWallDebuff checks if a player moved from one NGU_VUC to a different NGU_VUC.
// If so, it applies the "regional_wall_suppression" debuff status to the player's Statuses list.
func CheckAndApplyRegionalWallDebuff(player *models.Player, fromX, fromY, toX, toY int, q ...notify.Querier) string {
	if player == nil || (fromX == 0 && fromY == 0) {
		return ""
	}
	return applyRegionalWallDebuff(player, FindLargeZoneAt(fromX, fromY), FindLargeZoneAt(toX, toY), q...)
}

// CheckAndApplyRegionalWallDebuffBetween resolves the containing large zones
// from region ancestry. Use it when callers know both region IDs: stacked maps
// can reuse coordinates, so coordinate-only lookup may miss or misidentify a
// transition between world-map layers.
func CheckAndApplyRegionalWallDebuffBetween(player *models.Player, fromRegion, toRegion string, q ...notify.Querier) string {
	if player == nil || fromRegion == "" || toRegion == "" || fromRegion == toRegion {
		return ""
	}
	return applyRegionalWallDebuff(player, largeZoneForRegion(fromRegion), largeZoneForRegion(toRegion), q...)
}

func largeZoneForRegion(regionID string) *models.Region {
	if regionID == "" {
		return nil
	}
	for _, ancestorID := range regionScopeIDs(regionID) {
		if zone := ZoneByID(ancestorID); zone != nil && zone.ZoneType == ZoneTypeLarge {
			return zone
		}
	}
	zone := ZoneByID(regionID)
	if zone == nil {
		return nil
	}
	if zone.ZoneType == ZoneTypeLarge {
		return zone
	}
	if strings.EqualFold(zone.ZoneType, ZoneTypeSecret) || strings.EqualFold(zone.ZoneType, ZoneTypeDungeon) {
		return nil
	}
	refX, refY := zone.EntryX, zone.EntryY
	if refX == 0 && refY == 0 {
		refX = (zone.MinX + zone.MaxX) / 2
		refY = (zone.MinY + zone.MaxY) / 2
	}
	if refX != 0 || refY != 0 {
		ensureZonesLoaded()
		zoneReg.mu.RLock()
		defer zoneReg.mu.RUnlock()
		mapID := zone.WorldMapID
		for i := range zoneReg.zones {
			z := &zoneReg.zones[i]
			if z.ZoneType == ZoneTypeLarge && (mapID == "" || z.WorldMapID == "" || z.WorldMapID == mapID) && zoneContains(z, refX, refY) {
				return z
			}
		}
	}
	return nil
}

// RegionalWallSuppressionEnabled reports whether crossing a Great Domain (NGU_VUC)
// boundary applies the regional_wall_suppression penalty. Defaults to enabled;
// set REGIONAL_WALL_SUPPRESSION_ENABLED to false/0/off/no to disable it.
func RegionalWallSuppressionEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("REGIONAL_WALL_SUPPRESSION_ENABLED"))) {
	case "false", "0", "off", "no":
		return false
	}
	return true
}

func applyRegionalWallDebuff(player *models.Player, oldZone, newZone *models.Region, q ...notify.Querier) string {
	if !RegionalWallSuppressionEnabled() {
		return ""
	}
	if oldZone == nil || newZone == nil {
		return ""
	}
	if oldZone.ID == newZone.ID {
		return ""
	}
	if oldZone.ZoneType != ZoneTypeLarge || newZone.ZoneType != ZoneTypeLarge {
		return ""
	}

	debuffPct, duration := GetRegionalWallDebuff(player.Rank, player.SubRank)
	sourceLabel := fmt.Sprintf("Giới Vực Bích (%s ➔ %s)", oldZone.Name, newZone.Name)
	player.Statuses.Add("regional_wall_suppression", sourceLabel, duration, debuffPct)

	debuffInt := int(math.Abs(debuffPct * 100))
	durMinutes := int(duration.Minutes())
	var msg string
	if durMinutes >= 60 {
		msg = fmt.Sprintf("⚠️ Đạo hữu vừa vượt qua Giới Vực Bích từ %s sang %s! Quy tắc địa giới phản phệ gây ra [Áp Chế Giới Vực] (Công/Thủ/Tốc giảm %d%% trong %d giờ).",
			oldZone.Name, newZone.Name, debuffInt, durMinutes/60)
	} else {
		msg = fmt.Sprintf("⚠️ Đạo hữu vừa vượt qua Giới Vực Bích từ %s sang %s! Quy tắc địa giới phản phệ gây ra [Áp Chế Giới Vực] (Công/Thủ/Tốc giảm %d%% trong %d phút).",
			oldZone.Name, newZone.Name, debuffInt, durMinutes)
	}

	logger.S.Warnf("[Áp Chế Giới Vực] Người chơi %s (%s) bị áp chế khi vượt Giới Vực Bích từ %s sang %s (giảm %d%%, thời gian %v)",
		player.Username, player.ID, oldZone.Name, newZone.Name, debuffInt, duration)

	var querier notify.Querier
	if len(q) > 0 && q[0] != nil {
		querier = q[0]
	} else if db.DB != nil {
		querier = db.DB
	}
	if querier != nil && player.ID != uuid.Nil {
		_ = notify.Send(querier, player.ID, "regional_wall_suppression", "Áp Chế Giới Vực", msg)
	}

	return msg
}

// returns the log message plus whether a fight was started. attackTargetID is
// only used to explicitly attack another player standing on the destination.
