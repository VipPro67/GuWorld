package handlers

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// rowQuerier matches the QueryRow surface shared by *sql.DB and *sql.Tx so
// status helpers work inside and outside transactions.
type rowQuerier interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

// execQuerier matches the Exec surface shared by *sql.DB and *sql.Tx.
type execQuerier interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// loadPlayerStatuses reads the raw statuses array for a player.
func loadPlayerStatuses(q rowQuerier, playerID uuid.UUID) (models.StatusList, error) {
	var statuses models.StatusList
	err := q.QueryRow(`SELECT statuses FROM players WHERE id = $1`, playerID).Scan(&statuses)
	if err != nil {
		return nil, err
	}
	return statuses, nil
}

// GrantPlayerStatus adds (or refreshes) a status on a player inside the given
// transaction. Duration semantics: d < 0 => registry default (permanent when
// the definition has none), d == 0 => forced permanent, d > 0 => explicit.
// It returns false when the status key is unknown to the registry.
func GrantPlayerStatus(q execQuerier, rq rowQuerier, playerID uuid.UUID, statusID, source string, d time.Duration, value float64) (bool, error) {
	if !gamefx.ValidStatuses()[statusID] {
		return false, fmt.Errorf("unknown status %q", statusID)
	}
	statuses, err := loadPlayerStatuses(rq, playerID)
	if err != nil {
		return false, err
	}
	if d < 0 {
		d = time.Duration(gamefx.StatusDefaultDuration(statusID)) * time.Minute
	}
	statuses.Add(statusID, source, d, value)
	raw, err := statuses.Value()
	if err != nil {
		return false, err
	}
	if _, err := q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, raw, playerID); err != nil {
		return false, err
	}
	return true, nil
}

// SinhKhiTanGuID giữ để tương thích: Cổ đầu tiên mang persistent
// Sinh Khí Tàn. Detection mới đọc spec Cổ (PERSIST_STATUS_ON_HIT),
// không còn so GuID ở đây.
const SinhKhiTanGuID = "GU_KILL_ENERGY"

// playerHasBattleStartMechanic reports whether any equipped, ACTIVE, fed Gu
// authored with the given ON_BATTLE_START UNIQUE native-hook code is in the
// player's loadout. Dùng chung cho CLEANSE_REGIONAL_WALL và FAKE_REGIONAL_WALL.
func playerHasBattleStartMechanic(player *models.Player, code string) bool {
	if player == nil {
		return false
	}
	for _, gu := range player.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" || gu.Hunger <= 0 {
			continue
		}
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok {
			if spec, ok = models.GetGuAny(gu.GuID); !ok {
				continue
			}
		}
		for _, m := range models.CompiledGuEffects(spec).NativeMechanics() {
			if m.Trigger == gameplayeffects.TriggerOnBattleStart && m.Code == code {
				return true
			}
		}
	}
	return false
}

// maybeCleanseRegionalWall gỡ trạng thái Áp Chế Giới Vực khỏi người chơi khi
// bắt đầu trận (bất kỳ Cổ nào trong loadout mang CLEANSE_REGIONAL_WALL).
// Gọi trước khi tính CalculatePlayerStats.
func maybeCleanseRegionalWall(player *models.Player) {
	if player == nil || !playerHasBattleStartMechanic(player, "CLEANSE_REGIONAL_WALL") {
		return
	}
	if player.Statuses.Has("regional_wall_suppression") {
		player.Statuses.Remove("regional_wall_suppression")
		logger.S.Infof("Cleared regional_wall_suppression via CLEANSE_REGIONAL_WALL")
	}
}

// ApplySinhKhiTanHitToList tăng StatusList thêm 1 tầng Sinh Khí Tàn (cap V),
// refresh thời hạn SinhKhiTanDurationMinutes. Trả về tầng mới + cường độ.
// Pure trên memory, caller tự persist.
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

// CleanseSinhKhiTanOneTier thanh tẩy 1 tầng Sinh Khí Tàn của player.
// Trả tier còn lại (0 = đã sạch), found=false khi không có trạng thái.
func CleanseSinhKhiTanOneTier(q execQuerier, rq rowQuerier, playerID uuid.UUID) (tier int, found bool, err error) {
	statuses, err := loadPlayerStatuses(rq, playerID)
	if err != nil {
		return 0, false, err
	}
	tier, found = CleanseSinhKhiTanOneTierFromList(&statuses)
	if !found {
		return 0, false, nil
	}
	raw, err := statuses.Value()
	if err != nil {
		return 0, false, err
	}
	if _, err := q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, raw, playerID); err != nil {
		return 0, false, err
	}
	return tier, true, nil
}

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

// RevokePlayerStatus removes every instance of a status id. Returns whether
// the status was present.
func RevokePlayerStatus(q execQuerier, rq rowQuerier, playerID uuid.UUID, statusID string) (bool, error) {
	statuses, err := loadPlayerStatuses(rq, playerID)
	if err != nil {
		return false, err
	}
	if !statuses.Has(statusID) {
		return false, nil
	}
	statuses.Remove(statusID)
	raw, err := statuses.Value()
	if err != nil {
		return false, err
	}
	if _, err := q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, raw, playerID); err != nil {
		return false, err
	}
	return true, nil
}

// CleanExpiredPlayerStatuses drops expired entries and persists when anything
// changed. Safe to call on hot paths before gameplay checks.
func CleanExpiredPlayerStatuses(q execQuerier, rq rowQuerier, playerID uuid.UUID) error {
	statuses, err := loadPlayerStatuses(rq, playerID)
	if err != nil {
		return err
	}
	if !statuses.Expire(time.Now()) {
		return nil
	}
	raw, err := statuses.Value()
	if err != nil {
		return err
	}
	_, err = q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, raw, playerID)
	return err
}

// statusBlockedMessage builds a Vietnamese error listing the statuses that
// block the given action.
func statusBlockedMessage(p *models.Player, action string) string {
	if gamefx.IsDeadState(p) {
		return "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể thực hiện hành động này!"
	}
	defs := gamefx.BlockingStatuses(p, action)
	if len(defs) == 0 {
		return "Hành động hiện bị chặn bởi trạng thái."
	}
	names := ""
	for i, def := range defs {
		if i > 0 {
			names += ", "
		}
		names += def.Name
	}
	return fmt.Sprintf("Không thể thực hiện do đang trong trạng thái: %s", names)
}
