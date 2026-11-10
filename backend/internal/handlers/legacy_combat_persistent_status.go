package handlers

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// persistentHitGrant là một persistent status được grant sau combat.
type persistentHitGrant struct {
	StatusKey string
	GuID      string
	Tier      int // tier sau grant cho sinh_khi_tan, các status khác = 0
}

// persistentHitStatusKeys maps GuID phe ta (SimAction.SourceID) sang
// persistent status key khai trên Cổ (UNIQUE PERSIST_STATUS_ON_HIT).
func persistentHitStatusKeys(simState *simulator.CombatState) map[string]string {
	out := map[string]string{}
	if simState == nil {
		return out
	}
	for _, a := range simState.Allies {
		if a == nil {
			continue
		}
		for _, act := range a.Actions {
			if act.PersistentStatusKey == "" || act.SourceID == "" {
				continue
			}
			if _, dup := out[act.SourceID]; !dup {
				out[act.SourceID] = act.PersistentStatusKey
			}
		}
	}
	return out
}

// maybeGrantPersistentHitStatuses grant các persistent status data-driven
// cho defender PvP khi đòn của Cổ mang PERSIST_STATUS_ON_HIT đánh trúng
// (log HitLanded, gồm cả hit bị khiên hấp thụ; miss/evade/miễn nhiễm
// không grant). Mỗi status grant tối đa 1 lần mỗi trận. Chỉ PvP thật
// (bỏ arena virtual/dummy). Lỗi grant được bỏ qua để không chặn kết toán.
func maybeGrantPersistentHitStatuses(tx *sql.Tx, player *models.Player, simState *simulator.CombatState, logs []simulator.CombatLog) []persistentHitGrant {
	if player == nil || player.ActiveCombat == nil || simState == nil {
		return nil
	}
	if player.ActiveCombat.EnemyType != "PVP" {
		return nil
	}
	defenderID := player.ActiveCombat.TargetPlayerID
	if defenderID == "" || defenderID == "dummy" {
		return nil
	}
	statusByGu := persistentHitStatusKeys(simState)
	if len(statusByGu) == 0 {
		return nil
	}
	allyIDs := make(map[string]bool, len(simState.Allies))
	for _, a := range simState.Allies {
		if a != nil {
			allyIDs[a.ID] = true
		}
	}
	hitKeys := make(map[string]string, len(statusByGu))
	for _, l := range logs {
		if !l.HitLanded || !allyIDs[l.ActorId] {
			continue
		}
		key, hasGu := statusByGu[l.SourceGuID]
		if !hasGu || key == "" {
			continue
		}
		if _, dup := hitKeys[key]; !dup {
			hitKeys[key] = l.SourceGuID
		}
	}
	if len(hitKeys) == 0 {
		return nil
	}
	defenderUUID, err := uuid.Parse(defenderID)
	if err != nil || defenderUUID == player.ID {
		return nil
	}
	var grants []persistentHitGrant
	for key, guID := range hitKeys {
		if strings.EqualFold(key, gamefx.StatusSinhKhiTan) {
			tier, ok, err := GrantSinhKhiTanHit(tx, tx, defenderUUID)
			if err != nil || !ok {
				continue
			}
			grants = append(grants, persistentHitGrant{StatusKey: key, GuID: guID, Tier: tier})
			continue
		}
		if !gamefx.ValidStatuses()[key] {
			continue
		}
		if ok, err := GrantPlayerStatus(tx, tx, defenderUUID, key, guID, -1, 0); err != nil || !ok {
			continue
		}
		grants = append(grants, persistentHitGrant{StatusKey: key, GuID: guID})
	}
	return grants
}

// maybeGrantFakeRegionalWallStatus grant trạng thái Áp Chế Giới Vực (regional_wall_suppression)
// cho defender PvP khi attacker mang FAKE_REGIONAL_WALL (Ngũ Hành Thiên Chướng Trận).
// Cường độ debuff giảm 1 nửa so với Giới Vực Bích tự nhiên, thời gian ngắn hơn (15 phút + 5 phút mỗi rank).
func maybeGrantFakeRegionalWallStatus(tx *sql.Tx, player *models.Player) *simulator.CombatLog {
	if player == nil || player.ActiveCombat == nil {
		return nil
	}
	if player.ActiveCombat.EnemyType != "PVP" || !playerHasBattleStartMechanic(player, "FAKE_REGIONAL_WALL") {
		return nil
	}
	defenderID := player.ActiveCombat.TargetPlayerID
	if defenderID == "" || defenderID == "dummy" {
		return nil
	}
	defenderUUID, err := uuid.Parse(defenderID)
	if err != nil || defenderUUID == player.ID {
		return nil
	}
	defenderRank := player.ActiveCombat.EnemyRank
	defenderSubRank := player.ActiveCombat.EnemySubRank
	if defenderRank < 1 && len(player.ActiveCombat.Enemies) > 0 {
		defenderRank = player.ActiveCombat.Enemies[0].Rank
		defenderSubRank = player.ActiveCombat.Enemies[0].SubRank
	}
	if defenderRank < 1 {
		defenderRank = 1
	}
	if defenderSubRank < 1 {
		defenderSubRank = 1
	}
	pct, _ := gamefx.RegionalWallDebuff(defenderRank, defenderSubRank)
	debuffPct := pct * 0.5 // Debuff giảm 1 nửa
	durationMinutes := 15 + (defenderRank-1)*5
	if durationMinutes < 15 {
		durationMinutes = 15
	}
	duration := time.Duration(durationMinutes) * time.Minute

	ok, err := GrantPlayerStatus(tx, tx, defenderUUID, "regional_wall_suppression", "Ngũ Hành Thiên Chướng Trận", duration, debuffPct)
	if err != nil || !ok {
		return nil
	}
	formationMsg := fmt.Sprintf("⚠️ Đạo hữu bị đối phương kích hoạt Ngũ Hành Thiên Chướng Trận phản chiếu, gánh chịu [Áp Chế Giới Vực] (Công/Thủ/Tốc giảm %.1f%% trong %d phút).", -debuffPct*100, durationMinutes)
	logger.S.Warnf("[Áp Chế Giới Vực] Người chơi %s bị áp chế bởi Ngũ Hành Thiên Chướng Trận (giảm %.1f%% trong %d phút)", defenderUUID, -debuffPct*100, durationMinutes)
	_ = notify.Send(tx, defenderUUID, "regional_wall_suppression", "Áp Chế Giới Vực (Trận Pháp)", formationMsg)
	return &simulator.CombatLog{
		Action: "REGIONAL_WALL_SUPPRESSION_APPLIED",
		Detail: fmt.Sprintf("Ngũ Hành Thiên Chướng Trận phản chiếu! Kẻ địch nhận [Áp Chế Giới Vực] (công/thủ/tốc giảm %.1f%% trong %d phút).", -debuffPct*100, durationMinutes),
	}
}
func ApplySinhKhiTanHitToList(list *models.StatusList) (tier int, value float64) {
	now := time.Now()
	var cur float64
	for _, st := range *list {
		if st.ID == gamefx.StatusSinhKhiTan {
			if st.ExpiresAt != nil && !st.ExpiresAt.After(now) {
				continue // entry hết hạn coi như chưa có
			}
			cur = st.Value
			break
		}
	}
	tier = gamefx.SinhKhiTanTierFromValue(cur) + 1
	if tier > gamefx.SinhKhiTanMaxTier {
		tier = gamefx.SinhKhiTanMaxTier
	}
	if tier < 1 {
		tier = 1
	}
	value = gamefx.SinhKhiTanValueForTier(tier)
	list.Add(gamefx.StatusSinhKhiTan, "Sát Sinh Khí Cổ",
		time.Duration(gamefx.SinhKhiTanDurationMinutes)*time.Minute, value)
	return tier, value
}

// CleanseSinhKhiTanOneTierFromList tụt StatusList 1 tầng Sinh Khí Tàn.
// Hết tầng I thì xóa hẳn. Trả về tầng còn lại (0 = đã sạch) + found.
func CleanseSinhKhiTanOneTierFromList(list *models.StatusList) (tier int, found bool) {
	now := time.Now()
	var cur float64
	found = false
	for _, st := range *list {
		if st.ID == gamefx.StatusSinhKhiTan {
			if st.ExpiresAt != nil && !st.ExpiresAt.After(now) {
				list.Remove(gamefx.StatusSinhKhiTan)
				return 0, true // hết hạn sẵn, dọn luôn coi như đã sạch
			}
			cur = st.Value
			found = true
			break
		}
	}
	if !found {
		return 0, false
	}
	tier = gamefx.SinhKhiTanTierFromValue(cur) - 1
	if tier <= 0 {
		list.Remove(gamefx.StatusSinhKhiTan)
		return 0, true
	}
	list.Add(gamefx.StatusSinhKhiTan, "thanh tẩy",
		time.Duration(gamefx.SinhKhiTanDurationMinutes)*time.Minute,
		gamefx.SinhKhiTanValueForTier(tier))
	return tier, true
}

// GrantSinhKhiTanHit cộng 1 tầng Sinh Khí Tàn cho victim (PvP đánh trúng),
// refresh 5h. Vẫn cộng/refresh ở tầng V. Trả ok=false khi key chưa có registry.
func GrantSinhKhiTanHit(q execQuerier, rq rowQuerier, victimID uuid.UUID) (tier int, ok bool, err error) {
	if !gamefx.ValidStatuses()[gamefx.StatusSinhKhiTan] {
		return 0, false, fmt.Errorf("unknown status %q", gamefx.StatusSinhKhiTan)
	}
	statuses, err := loadPlayerStatuses(rq, victimID)
	if err != nil {
		return 0, false, err
	}
	tier, _ = ApplySinhKhiTanHitToList(&statuses)
	raw, err := statuses.Value()
	if err != nil {
		return 0, false, err
	}
	if _, err := q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, raw, victimID); err != nil {
		return 0, false, err
	}
	return tier, true, nil
}
