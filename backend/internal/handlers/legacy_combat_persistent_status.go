package handlers

import (
	"database/sql"
	"fmt"
	"sort"
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

// persistentHitGrant là một persistent status được grant sau combat.
type persistentHitGrant struct {
	StatusKey     string
	GuID          string
	Tier          int
	Value         float64
	DurationHours float64
}

// persistentHitStatusKeys maps GuID phe ta (SimAction.SourceID) sang
// persistent status key khai trên Cổ (UNIQUE PERSIST_STATUS_ON_HIT).
func persistentHitStatusKeys(simState *simulator.CombatState) map[string]models.PersistentHitStatusSpec {
	out := map[string]models.PersistentHitStatusSpec{}
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
				out[act.SourceID] = models.PersistentHitStatusSpec{
					StatusKey: act.PersistentStatusKey, PvPOnly: act.PersistentPvPOnly,
					Configured: act.PersistentStatusConfigured, MaxStacks: act.PersistentStatusMaxStacks,
					DurationHours: act.PersistentStatusDurationHours, DispelRule: act.PersistentStatusDispelRule,
					TierScaling: append([]gameplayeffects.StackingStatusTier(nil), act.PersistentStatusTierScaling...),
				}
			}
		}
	}
	return out
}

// maybeGrantPersistentHitStatuses grant các persistent status data-driven
// cho defender PvP khi đòn của Cổ mang PERSIST_STATUS_ON_HIT đánh trúng
// (log HitLanded, gồm cả hit bị khiên hấp thụ; miss/evade/miễn nhiễm
// không grant). Mỗi status grant tối đa 1 lần mỗi trận. Chỉ PvP thật
// (bỏ arena virtual/dummy). Lỗi cấu hình hoặc ghi status được trả về để transaction
// không xác nhận một trận đấu mà effect CSV đã bị bỏ qua.
func maybeGrantPersistentHitStatuses(tx *sql.Tx, player *models.Player, simState *simulator.CombatState, logs []simulator.CombatLog) ([]persistentHitGrant, error) {
	if player == nil || player.ActiveCombat == nil || simState == nil {
		return nil, nil
	}
	if player.ActiveCombat.EnemyType != "PVP" {
		return nil, nil
	}
	defenderID := player.ActiveCombat.TargetPlayerID
	if defenderID == "" || defenderID == "dummy" {
		return nil, nil
	}
	statusByGu := persistentHitStatusKeys(simState)
	if len(statusByGu) == 0 {
		return nil, nil
	}
	allyIDs := make(map[string]bool, len(simState.Allies))
	for _, a := range simState.Allies {
		if a != nil {
			allyIDs[a.ID] = true
		}
	}
	type hitSource struct {
		guID string
		spec models.PersistentHitStatusSpec
	}
	hitKeys := make(map[string]hitSource, len(statusByGu))
	for _, l := range logs {
		if !l.HitLanded || !allyIDs[l.ActorId] {
			continue
		}
		spec, hasGu := statusByGu[l.SourceGuID]
		if !hasGu || spec.StatusKey == "" {
			continue
		}
		if _, dup := hitKeys[spec.StatusKey]; !dup {
			hitKeys[spec.StatusKey] = hitSource{guID: l.SourceGuID, spec: spec}
		}
	}
	if len(hitKeys) == 0 {
		return nil, nil
	}
	defenderUUID, err := uuid.Parse(defenderID)
	if err != nil || defenderUUID == player.ID {
		return nil, err
	}
	keys := make([]string, 0, len(hitKeys))
	for key := range hitKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var grants []persistentHitGrant
	for _, key := range keys {
		hit := hitKeys[key]
		if !gamefx.ValidStatuses()[key] {
			return nil, fmt.Errorf("persistent hit status %q from Gu %q is not registered", key, hit.guID)
		}
		if !hit.spec.Configured && !strings.EqualFold(key, gamefx.StatusSinhKhiTan) {
			if ok, err := GrantPlayerStatus(tx, tx, defenderUUID, key, hit.guID, -1, 0); err != nil {
				return nil, fmt.Errorf("grant persistent status %q: %w", key, err)
			} else if !ok {
				return nil, fmt.Errorf("grant persistent status %q returned no result", key)
			}
			grants = append(grants, persistentHitGrant{StatusKey: key, GuID: hit.guID})
			continue
		}
		grantSpec := stackingStatusConfig(hit.spec)
		tier, value, ok, err := GrantPersistentStackingStatus(tx, tx, defenderUUID, grantSpec)
		if err != nil {
			return nil, fmt.Errorf("grant stacking status %q: %w", key, err)
		}
		if !ok {
			return nil, fmt.Errorf("grant stacking status %q returned no result", key)
		}
		grants = append(grants, persistentHitGrant{
			StatusKey: key, GuID: hit.guID, Tier: tier, Value: value,
			DurationHours: grantSpec.DurationHours,
		})
	}
	return grants, nil
}

// maybeGrantFakeRegionalWallStatus grant trạng thái Áp Chế Giới Vực (regional_wall_suppression)
// cho defender PvP khi attacker mang FAKE_REGIONAL_WALL (Ngũ Hành Thiên Chướng Trận).
// Cường độ debuff giảm 1 nửa so với Giới Vực Bích tự nhiên, thời gian ngắn hơn (15 phút + 5 phút mỗi rank).
func maybeGrantFakeRegionalWallStatus(tx *sql.Tx, player *models.Player) *simulator.CombatLog {
	if player == nil || player.ActiveCombat == nil {
		return nil
	}
	rankDebuff, configured := playerRankScaledDebuff(player)
	if player.ActiveCombat.EnemyType != "PVP" || !configured {
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
	debuffPct, duration := rankScaledDebuffValues(rankDebuff, defenderRank, defenderSubRank)
	durationMinutes := int(duration / time.Minute)

	ok, err := GrantPlayerStatus(tx, tx, defenderUUID, rankDebuff.StatusID, rankDebuff.SourceName, duration, debuffPct)
	if err != nil || !ok {
		return nil
	}
	formationMsg := fmt.Sprintf("⚠️ Đạo hữu bị đối phương kích hoạt Ngũ Hành Thiên Chướng Trận phản chiếu, gánh chịu [Áp Chế Giới Vực] (Công/Thủ/Tốc giảm %.1f%% trong %d phút).", -debuffPct*100, durationMinutes)
	logger.S.Warnf("[Áp Chế Giới Vực] Người chơi %s bị áp chế bởi Ngũ Hành Thiên Chướng Trận (giảm %.1f%% trong %d phút)", defenderUUID, -debuffPct*100, durationMinutes)
	notify.SendBestEffort(tx, defenderUUID, "regional_wall_suppression", "Áp Chế Giới Vực (Trận Pháp)", formationMsg)
	return &simulator.CombatLog{
		Action: "REGIONAL_WALL_SUPPRESSION_APPLIED",
		Detail: fmt.Sprintf("Ngũ Hành Thiên Chướng Trận phản chiếu! Kẻ địch nhận [Áp Chế Giới Vực] (công/thủ/tốc giảm %.1f%% trong %d phút).", -debuffPct*100, durationMinutes),
	}
}
func defaultSinhKhiTanStackingSpec() models.PersistentHitStatusSpec {
	tiers := make([]gameplayeffects.StackingStatusTier, 0, gamefx.SinhKhiTanMaxTier)
	for tier := 1; tier <= gamefx.SinhKhiTanMaxTier; tier++ {
		tiers = append(tiers, gameplayeffects.StackingStatusTier{
			Tier: tier, StatPercent: gamefx.SinhKhiTanTierValues[tier] * 100,
		})
	}
	return models.PersistentHitStatusSpec{
		StatusKey: gamefx.StatusSinhKhiTan, PvPOnly: true,
		MaxStacks:     gamefx.SinhKhiTanMaxTier,
		DurationHours: float64(gamefx.SinhKhiTanDurationMinutes) / 60,
		DispelRule:    "DECREMENT_ONE", TierScaling: tiers, Configured: true,
	}
}

func stackingStatusConfig(spec models.PersistentHitStatusSpec) models.PersistentHitStatusSpec {
	if !spec.Configured && strings.EqualFold(spec.StatusKey, gamefx.StatusSinhKhiTan) {
		return defaultSinhKhiTanStackingSpec()
	}
	return spec
}

func stackingStatusTierValue(spec models.PersistentHitStatusSpec, tier int) (float64, bool) {
	for _, scaling := range spec.TierScaling {
		if scaling.Tier == tier {
			return scaling.StatPercent / 100, true
		}
	}
	return 0, false
}

func currentStackingStatusTier(status models.StatusInstance, spec models.PersistentHitStatusSpec) int {
	if status.Stacking != nil && status.Stacking.CurrentTier > 0 {
		return status.Stacking.CurrentTier
	}
	for _, scaling := range spec.TierScaling {
		if absFloat(status.Value-scaling.StatPercent/100) < 1e-9 {
			return scaling.Tier
		}
	}
	if strings.EqualFold(spec.StatusKey, gamefx.StatusSinhKhiTan) {
		return gamefx.SinhKhiTanTierFromValue(status.Value)
	}
	return 0
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

// ApplyPersistentStackingStatusToList increments a configured tier and stores
// its policy with the player status for future reapplications and cleansing.
func ApplyPersistentStackingStatusToList(list *models.StatusList, authored models.PersistentHitStatusSpec, source string) (tier int, value float64, ok bool) {
	spec := stackingStatusConfig(authored)
	if list == nil || spec.StatusKey == "" || spec.MaxStacks < 1 || len(spec.TierScaling) == 0 {
		return 0, 0, false
	}
	current := 0
	now := time.Now()
	for _, status := range *list {
		if status.ID != spec.StatusKey || (status.ExpiresAt != nil && !status.ExpiresAt.After(now)) {
			continue
		}
		current = currentStackingStatusTier(status, spec)
		break
	}
	tier = current + 1
	if tier > spec.MaxStacks {
		tier = spec.MaxStacks
	}
	value, ok = stackingStatusTierValue(spec, tier)
	if !ok {
		return 0, 0, false
	}
	duration := time.Duration(spec.DurationHours * float64(time.Hour))
	if duration <= 0 {
		return 0, 0, false
	}
	metadataTiers := make([]models.StatusStackTier, len(spec.TierScaling))
	for i, scaling := range spec.TierScaling {
		metadataTiers[i] = models.StatusStackTier{Tier: scaling.Tier, StatPercent: scaling.StatPercent}
	}
	list.AddStacking(spec.StatusKey, source, duration, value, models.StatusStackingMetadata{
		CurrentTier: tier, MaxStacks: spec.MaxStacks, DurationHours: spec.DurationHours,
		DispelRule: strings.ToUpper(spec.DispelRule), TierScaling: metadataTiers,
	})
	return tier, value, true
}

func ApplySinhKhiTanHitToList(list *models.StatusList) (tier int, value float64) {
	tier, value, _ = ApplyPersistentStackingStatusToList(list, defaultSinhKhiTanStackingSpec(), "Sát Sinh Khí Cổ")
	return tier, value
}

// CleansePersistentStackingStatusOneTierFromList applies the stored cleanse
// policy. Legacy Sinh Khí Tàn entries without metadata retain decrement-one.
func CleansePersistentStackingStatusOneTierFromList(list *models.StatusList, statusKey string) (tier int, found bool) {
	if list == nil {
		return 0, false
	}
	now := time.Now()
	for i := range *list {
		status := &(*list)[i]
		if status.ID != statusKey {
			continue
		}
		if status.ExpiresAt != nil && !status.ExpiresAt.After(now) {
			list.Remove(statusKey)
			return 0, true
		}
		spec := models.PersistentHitStatusSpec{StatusKey: statusKey}
		metadata := status.Stacking
		if metadata == nil {
			if !strings.EqualFold(statusKey, gamefx.StatusSinhKhiTan) {
				return 0, false
			}
			spec = defaultSinhKhiTanStackingSpec()
		} else {
			spec.MaxStacks = metadata.MaxStacks
			spec.DurationHours = metadata.DurationHours
			spec.DispelRule = metadata.DispelRule
			spec.Configured = true
			for _, scaling := range metadata.TierScaling {
				spec.TierScaling = append(spec.TierScaling, gameplayeffects.StackingStatusTier{Tier: scaling.Tier, StatPercent: scaling.StatPercent})
			}
		}
		if strings.EqualFold(spec.DispelRule, "CLEAR_ALL") {
			list.Remove(statusKey)
			return 0, true
		}
		current := currentStackingStatusTier(*status, spec)
		if current <= 1 {
			list.Remove(statusKey)
			return 0, true
		}
		tier = current - 1
		value, ok := stackingStatusTierValue(spec, tier)
		if !ok {
			list.Remove(statusKey)
			return 0, true
		}
		duration := time.Duration(spec.DurationHours * float64(time.Hour))
		if duration <= 0 {
			duration = time.Duration(gamefx.SinhKhiTanDurationMinutes) * time.Minute
		}
		metadataTiers := make([]models.StatusStackTier, len(spec.TierScaling))
		for j, scaling := range spec.TierScaling {
			metadataTiers[j] = models.StatusStackTier{Tier: scaling.Tier, StatPercent: scaling.StatPercent}
		}
		list.AddStacking(statusKey, "thanh tẩy", duration, value, models.StatusStackingMetadata{
			CurrentTier: tier, MaxStacks: spec.MaxStacks, DurationHours: spec.DurationHours,
			DispelRule: spec.DispelRule, TierScaling: metadataTiers,
		})
		return tier, true
	}
	return 0, false
}

// CleanseSinhKhiTanOneTierFromList preserves the legacy helper API.
func CleanseSinhKhiTanOneTierFromList(list *models.StatusList) (tier int, found bool) {
	return CleansePersistentStackingStatusOneTierFromList(list, gamefx.StatusSinhKhiTan)
}

// GrantSinhKhiTanHit cộng 1 tầng Sinh Khí Tàn cho victim (PvP đánh trúng),
// refresh 5h. Vẫn cộng/refresh ở tầng V. Trả ok=false khi key chưa có registry.
func GrantSinhKhiTanHit(q execQuerier, rq rowQuerier, victimID uuid.UUID) (tier int, ok bool, err error) {
	if !gamefx.ValidStatuses()[gamefx.StatusSinhKhiTan] {
		return 0, false, fmt.Errorf("unknown status %q", gamefx.StatusSinhKhiTan)
	}
	tier, _, ok, err = GrantPersistentStackingStatus(q, rq, victimID, defaultSinhKhiTanStackingSpec())
	return tier, ok, err
}

func GrantPersistentStackingStatus(q execQuerier, rq rowQuerier, victimID uuid.UUID, spec models.PersistentHitStatusSpec) (tier int, value float64, ok bool, err error) {
	if !gamefx.ValidStatuses()[spec.StatusKey] {
		return 0, 0, false, fmt.Errorf("unknown status %q", spec.StatusKey)
	}
	statuses, err := loadPlayerStatuses(rq, victimID)
	if err != nil {
		return 0, 0, false, err
	}
	tier, value, ok = ApplyPersistentStackingStatusToList(&statuses, spec, spec.StatusKey)
	if !ok {
		return 0, 0, false, fmt.Errorf("invalid stacking status configuration for %q", spec.StatusKey)
	}
	raw, err := statuses.Value()
	if err != nil {
		return 0, 0, false, err
	}
	if _, err := q.Exec(`UPDATE players SET statuses = $1 WHERE id = $2`, raw, victimID); err != nil {
		return 0, 0, false, err
	}
	return tier, value, true, nil
}
