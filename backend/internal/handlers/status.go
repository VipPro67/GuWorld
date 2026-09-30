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

// SinhKhiTanGuID là Cổ trùng gây ra Sinh Khí Tàn khi đánh trúng ở PvP.
const SinhKhiTanGuID = "GU_KILL_ENERGY"

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

// maybeGrantSinhKhiTan cộng 1 tầng Sinh Khí Tàn cho defender PvP khi attacker
// mang Sát Sinh Khí Cổ trong loadout và có ít nhất 1 hit gây damage của phe ta.
// Chỉ áp dụng PvP thật (bỏ arena virtual/dummy). Trả granted + tầng mới.
// Lỗi grant (defender không tồn tại...) được bỏ qua để không chặn kết toán combat.
func maybeGrantSinhKhiTan(tx *sql.Tx, player *models.Player, loadout *simulator.CombatLoadout, simState *simulator.CombatState, logs []simulator.CombatLog) (granted bool, tier int) {
	if player == nil || player.ActiveCombat == nil || loadout == nil || simState == nil {
		return false, 0
	}
	if player.ActiveCombat.EnemyType != "PVP" {
		return false, 0
	}
	defenderID := player.ActiveCombat.TargetPlayerID
	if defenderID == "" || defenderID == "dummy" {
		return false, 0
	}
	// Loadout settings-based (AutoResolveCombat) không có GuIds: quét action
	// phe ta tìm SourceID của Sát Sinh Khí Cổ, đồng thời lấy tên đòn để match
	// chính xác hit gây damage (tránh cộng khi damage từ Cổ khác).
	bringsKillEnergy, hitLanded := sinhKhiTanHitLanded(loadout, simState, logs)
	if !bringsKillEnergy || !hitLanded {
		return false, 0
	}
	defenderUUID, err := uuid.Parse(defenderID)
	if err != nil || defenderUUID == player.ID {
		return false, 0
	}
	tier, ok, err := GrantSinhKhiTanHit(tx, tx, defenderUUID)
	if err != nil || !ok {
		return false, 0
	}
	return true, tier
}

// sinhKhiTanHitLanded pure: attacker có mang Sát Sinh Khí Cổ (theo loadout
// hoặc SourceID action phe ta) và có hit damage của phe ta từ đúng đòn đó.
// Trả (bringsGu, hitLanded).
func sinhKhiTanHitLanded(loadout *simulator.CombatLoadout, simState *simulator.CombatState, logs []simulator.CombatLog) (bool, bool) {
	if simState == nil {
		return false, false
	}
	bringsKillEnergy := false
	if loadout != nil {
		for _, guID := range loadout.GuIds {
			if guID == SinhKhiTanGuID {
				bringsKillEnergy = true
				break
			}
		}
	}
	var killEnergyNames []string
	allyIDs := make(map[string]bool, len(simState.Allies))
	for _, a := range simState.Allies {
		if a == nil {
			continue
		}
		allyIDs[a.ID] = true
		for _, act := range a.Actions {
			if act.SourceID == SinhKhiTanGuID {
				bringsKillEnergy = true
				if act.Name != "" {
					killEnergyNames = append(killEnergyNames, "["+act.Name+"]")
				}
			}
		}
	}
	if !bringsKillEnergy {
		return false, false
	}
	for _, l := range logs {
		if l.Damage <= 0 || !allyIDs[l.ActorId] {
			continue
		}
		if len(killEnergyNames) == 0 {
			return true, true // loadout có Cổ nhưng không resolve được tên đòn: nới lỏng
		}
		for _, n := range killEnergyNames {
			if strings.Contains(l.Detail, n) {
				return true, true
			}
		}
	}
	return true, false
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
	if p != nil && p.IsDead {
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
