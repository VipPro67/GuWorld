package handlers

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func CalculatePlayerStats(player *models.Player) map[string]float64 {
	return GetPassiveStats(player).Stats
}

func calculatePlayerStatsWithTerritory(player *models.Player, territory SectTerritoryInfo) map[string]float64 {
	return getPassiveStats(player, &territory).Stats
}

// StatusFlying is the world-map status granted by flying Gu. While active the
// player ignores ground-monster encounters (grid cells, ambient ambush and
// world-monster aggro). Grant/refresh it with a positive duration; using the
// same skill again removes it (early landing).
const StatusFlying = "phi_hanh"

// CapIgnoreMonsterEncounter is the passive capability with the same effect as
// StatusFlying, for permanently flying Gu.
const CapIgnoreMonsterEncounter = "IGNORE_MONSTER_ENCOUNTER"

// PlayerIgnoresMonsters reports whether a player currently bypasses monster
// encounters while moving: either a passive flying Gu (capability) or the
// active "phi_hanh" (flying) status.
func PlayerIgnoresMonsters(player *models.Player) bool {
	if player == nil {
		return false
	}
	if models.InventoryHasGuCapability(player.GuInventory, CapIgnoreMonsterEncounter, true) {
		return true
	}
	for _, st := range player.Statuses.Active() {
		if st.ID == StatusFlying {
			return true
		}
	}
	return false
}

// toggleFlyingStatus grants or removes the phi_hanh flying status. Duration
// defaults to the Gu's rank in minutes (rank 1..9), or turns when set in the
// CSV. Returns the log line and whether the player is now flying.
func toggleFlyingStatus(player *models.Player, guName string, guRank, turns int) (string, bool) {
	flyDuration := time.Duration(turns) * time.Minute
	if turns <= 0 {
		flyDuration = time.Duration(guRank) * time.Minute
	}
	if flyDuration <= 0 {
		flyDuration = time.Minute
	}
	for _, st := range player.Statuses.Active() {
		if st.ID == StatusFlying {
			player.Statuses.Remove(StatusFlying)
			return fmt.Sprintf("%s thu hồi pháp lực, đạo hữu hạ cánh xuống mặt đất.", guName), false
		}
	}
	player.Statuses.Add(StatusFlying, guName, flyDuration, 0)
	return fmt.Sprintf("%s thôi động, đạo hữu phi thân lên không trung, miễn nhiễm quái thú dưới đất trong %.0f phút.", guName, flyDuration.Minutes()), true
}

// PlayerDisguiseLevel is the total concealment score of a player: rank plus
// DISGUISE stats from equipped active Gu plus wisdom/dark path mastery bonus.
// Used both by scouting/profile and by world-map monster ambush rolls.
func PlayerDisguiseLevel(player *models.Player) float64 {
	disguiseLevel := float64(player.Rank)
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			disguiseLevel += models.GuStatValue(spec, "DISGUISE")
		}
	}
	for path, points := range player.PathMasteries {
		cleanPath := strings.ToLower(path)
		if strings.Contains(cleanPath, "wisdom") || strings.Contains(cleanPath, "dark") {
			switch gamefx.PathMasteryLevel(points) {
			case 1:
				disguiseLevel += 1.0
			case 2:
				disguiseLevel += 2.0
			case 3:
				disguiseLevel += 3.0
			case 4:
				disguiseLevel += 5.0
			}
		}
	}
	return disguiseLevel
}

// ── Passive Stats Breakdown ──────────────────────────────────────────────────

// PassiveStatSource describes one source contributing to a passive stat.
type PassiveStatSource struct {
	Source string  `json:"source"` // e.g. "base", "growth", "gu:GU_JADE_SKIN"
	Name   string  `json:"name"`   // human-readable label
	Value  float64 `json:"value"`  // contribution added to the stat total
	Final  float64 `json:"final"`  // actual contribution (equal to Value)
}

// PassiveStatDetail holds the breakdown for one stat key.
type PassiveStatDetail struct {
	Key     string              `json:"key"`
	Total   float64             `json:"total"`
	Sources []PassiveStatSource `json:"sources"`
}

// PassiveStatsResult is the full passive stats response.
type PassiveStatsResult struct {
	Stats   map[string]float64  `json:"stats"`   // flat totals (same as CalculatePlayerStats)
	Details []PassiveStatDetail `json:"details"` // per-stat breakdown
}

// GetPassiveStats computes a detailed breakdown of all passive stats, attributing
// each contribution to its source (base cultivation, growth boosts, equipped Gu).
func GetPassiveStats(player *models.Player) PassiveStatsResult {
	return getPassiveStats(player, nil)
}

func getPassiveStats(player *models.Player, territoryOverride *SectTerritoryInfo) PassiveStatsResult {
	// ── Base stats from Rank / SubRank ──
	baseAttack := 10.0 + float64(player.Rank-1)*10.0 + float64(player.SubRank-1)*2.0
	baseDefense := 5.0 + float64(player.Rank-1)*5.0 + float64(player.SubRank-1)*1.0
	baseSpeed := 10.0 + float64(player.Rank-1)*2.0 + float64(player.SubRank-1)*0.5

	// Accumulator: key -> []PassiveStatSource
	acc := map[string][]PassiveStatSource{}
	addSrc := func(key, source, name string, value float64) {
		if value == 0 {
			return
		}
		acc[key] = append(acc[key], PassiveStatSource{
			Source: source, Name: name, Value: value, Final: value,
		})
	}

	// Base sources
	addSrc("attack", "base", "Cơ bản (Cấp bậc)", baseAttack)
	addSrc("defense", "base", "Cơ bản (Cấp bậc)", baseDefense)
	addSrc("speed", "base", "Cơ bản (Cấp bậc)", baseSpeed)
	addSrc("movement", "base", "Cơ bản", 1.0)
	addSrc("investigation", "base", "Cơ bản", 1.0)
	addSrc("storage", "base", "Cơ bản", 100.0)

	// Growth boosts
	if player.GrowthData.Boosts.Attack > 0 {
		addSrc("attack", "growth", "Trưởng thành Cổ", player.GrowthData.Boosts.Attack)
	}
	if player.GrowthData.Boosts.Defense > 0 {
		addSrc("defense", "growth", "Trưởng thành Cổ", player.GrowthData.Boosts.Defense)
	}
	if player.GrowthData.Boosts.Speed > 0 {
		addSrc("speed", "growth", "Trưởng thành Cổ", player.GrowthData.Boosts.Speed)
	}

	// Equipped active Gu passives — each GuID contributes only its highest rank,
	// applied exactly once (duplicates never stack). Additive Gu stats are raw
	// values: path mastery and dao marks never scale passive stats.
	guBestRank := map[string]int{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if r, ok := guBestRank[gu.GuID]; !ok || gu.Rank > r {
				guBestRank[gu.GuID] = gu.Rank
			}
		}
	}
	guApplied := map[string]bool{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" && gu.Rank == guBestRank[gu.GuID] && !guApplied[gu.GuID] {
			guApplied[gu.GuID] = true
			spec, ok := models.GetGu(gu.GuID, gu.Rank)
			if !ok {
				continue
			}

			srcID := "gu:" + gu.GuID
			if value := models.GuStatValue(spec, "ATTACK"); value != 0 {
				addSrc("attack", srcID, spec.Name, value)
			}
			if value := models.GuStatValue(spec, "DEFENSE"); value != 0 {
				addSrc("defense", srcID, spec.Name, value)
			}
			if value := models.GuStatValue(spec, "SPEED"); value != 0 {
				addSrc("speed", srcID, spec.Name, value)
			}
			if value := models.GuStatValue(spec, "HP"); value != 0 {
				addSrc("hp", srcID, spec.Name, value)
			}
			if value := models.GuStatValueBySubtype(spec, "DAMAGE_DEALT", "MULTIPLY_PERCENT"); value > 0 && player.BaseHP > 0 {
				missingHpRate := 1.0 - player.HP/player.BaseHP
				addSrc("damage_multiplier", srcID, spec.Name, missingHpRate*value)
			}
			for _, entry := range []struct{ stat, key string }{{"SOUL_REFINEMENT", "soul"}, {"STORAGE", "storage"}, {"MOVEMENT", "movement"}, {"INVESTIGATION", "investigation"}, {"HEARING", "hearing"}, {"CLEANSING_POWER", "cleansing_power"}, {"LUCK", "luck"}} {
				if value := models.GuStatValue(spec, entry.stat); value != 0 {
					addSrc(entry.key, srcID, spec.Name, value)
				}
			}
		}
	}

	// Active statuses (World Map buffs)
	for _, st := range player.Statuses.Active() {
		if st.ID == "buff_atk" {
			addSrc("attack", "status:"+st.ID, st.Source, st.Value)
		}
		if st.ID == "buff_def" {
			addSrc("defense", "status:"+st.ID, st.Source, st.Value)
		}
		if st.ID == "buff_speed" {
			addSrc("speed", "status:"+st.ID, st.Source, st.Value)
		}
	}

	// Sect Territory AoE Buff
	territory := SectTerritoryInfo{}
	if territoryOverride != nil {
		territory = *territoryOverride
	} else if db.DB != nil && player.ID.String() != "" {
		territory = CheckPlayerSectTerritory(db.DB, player.ID, player.CurrentRegion, player.X, player.Y)
	}
	if territory.InTerritory {
		// Player is inside their sect's territory!
		// Give +10% base attack/defense buff, plus 2% per main hall level.
		buffMult := 0.10 + float64(territory.MainHallLvl)*0.02
		addSrc("attack", "sect_territory", "Hộ Sơn", baseAttack*buffMult)
		addSrc("defense", "sect_territory", "Hộ Sơn", baseDefense*buffMult)
	}

	// Build totals and details
	stats := map[string]float64{
		"attack":        0.0,
		"defense":       0.0,
		"speed":         0.0,
		"movement":      0.0,
		"investigation": 0.0,
		"storage":       0.0,
	}
	var details []PassiveStatDetail
	for key, srcs := range acc {
		total := 0.0
		for _, s := range srcs {
			total += s.Final
		}
		stats[key] = total
		details = append(details, PassiveStatDetail{Key: key, Total: total, Sources: srcs})
	}

	// Dynamic storage bonus
	// Only calculated if player rank < 6, as higher ranks have infinite storage
	if player.Rank < 6 {
		storageBonus := math.Floor(stats["attack"] * 0.5)
		stats["storage"] += storageBonus
		for i := range details {
			if details[i].Key == "storage" {
				details[i].Sources = append(details[i].Sources, PassiveStatSource{
					Source: "dynamic", Name: "Lực tấn công (×0.5)", Value: storageBonus, Final: storageBonus,
				})
				details[i].Total += storageBonus
				break
			}
		}
	}

	// Apply all_stats_buff_percent from equipped Gu active effects
	{
		totalBuffPct := 0.0
		seen := map[string]bool{}
		for _, gu := range player.GuInventory {
			if gu.Equipped && gu.Status == "ACTIVE" && gu.Rank == guBestRank[gu.GuID] && !seen[gu.GuID] {
				seen[gu.GuID] = true
				spec, ok := models.GetGu(gu.GuID, gu.Rank)
				if !ok {
					continue
				}
				if compiled := models.CompiledGuEffects(spec); compiled != nil {
					totalBuffPct += compiled.StatValue(gameplayeffects.TriggerOnAction, "ALL", "MULTIPLY_PERCENT")
				}
			}
		}
		if totalBuffPct > 0 {
			for key := range stats {
				if key == "storage" {
					continue
				}
				cur := stats[key]
				if cur > 0 {
					delta := cur * totalBuffPct
					stats[key] += delta
					found := false
					for i := range details {
						if details[i].Key == key {
							details[i].Sources = append(details[i].Sources, PassiveStatSource{
								Source: "gu:active", Name: "All Stats Buff", Value: delta, Final: delta,
							})
							details[i].Total += delta
							found = true
							break
						}
					}
					if !found {
						details = append(details, PassiveStatDetail{Key: key, Total: delta, Sources: []PassiveStatSource{{
							Source: "gu:active", Name: "All Stats Buff", Value: delta, Final: delta,
						}}})
					}
				}
			}
		}
	}

	// Character effects from race / constitution / statuses (percentage modifiers)
	applyCharEffect := func(sourceID, sourceName string, statMods map[string]float64) {
		if statMods == nil {
			return
		}
		for key, pct := range statMods {
			cur, ok := stats[key]
			if !ok || cur == 0 {
				continue
			}
			delta := cur * pct
			if delta == 0 {
				continue
			}
			stats[key] += delta

			found := false
			for i := range details {
				if details[i].Key == key {
					details[i].Sources = append(details[i].Sources, PassiveStatSource{
						Source: sourceID, Name: sourceName, Value: delta, Final: delta,
					})
					details[i].Total += delta
					found = true
					break
				}
			}
			if !found {
				details = append(details, PassiveStatDetail{Key: key, Total: stats[key], Sources: []PassiveStatSource{{
					Source: sourceID, Name: sourceName, Value: delta, Final: delta,
				}}})
			}
		}
	}

	for _, r := range gamefx.AllRaces() {
		if r.Key == player.Race {
			applyCharEffect("race:"+r.Key, "Chủng tộc: "+r.Name, r.Effects.StatMods)
			break
		}
	}

	for _, c := range gamefx.AllConstitutions() {
		if c.Key == player.Constitution {
			applyCharEffect("constitution:"+c.Key, "Thể chất: "+c.Name, c.Effects.StatMods)
			break
		}
	}

	for _, st := range player.Statuses.Active() {
		for _, def := range gamefx.AllStatuses() {
			if def.Key == st.ID {
				eff := def.EffectsForValue(st.Value)
				applyCharEffect("status:"+def.Key, "Trạng thái: "+def.Name, eff.StatMods)
				break
			}
		}
	}

	// Floor the physical stats at zero: heavy status suppression (stat_mods)
	// must not drive them negative — a negative defense would amplify
	// incoming damage and a negative speed would break turn order.
	for _, key := range []string{"attack", "defense", "speed", "hp"} {
		cur, ok := stats[key]
		if !ok || cur >= 0 {
			continue
		}
		delta := -cur
		stats[key] = 0
		clampSrc := PassiveStatSource{Source: "floor", Name: "Giới hạn dưới (0)", Value: -delta, Final: -delta}
		found := false
		for i := range details {
			if details[i].Key == key {
				details[i].Sources = append(details[i].Sources, clampSrc)
				details[i].Total = 0
				found = true
				break
			}
		}
		if !found {
			details = append(details, PassiveStatDetail{Key: key, Total: 0, Sources: []PassiveStatSource{clampSrc}})
		}
	}

	charFX := gamefx.Get(player)

	// Derived defensive bonuses from constitution (soul defense, mental resistance)
	constitutionName := "Thể chất"
	if player.Constitution != "" {
		constitutionName = gamefx.ConstitutionDisplayName(player.Constitution)
	}
	if charFX.SoulDefenseBonus > 0 {
		delta := charFX.SoulDefenseBonus * 100.0
		stats["soul_defense"] += delta
		details = append(details, PassiveStatDetail{Key: "soul_defense", Total: delta, Sources: []PassiveStatSource{{
			Source: "constitution:" + player.Constitution, Name: constitutionName, Value: delta, Final: delta,
		}}})
	}
	if charFX.MentalResistanceBonus > 0 {
		delta := charFX.MentalResistanceBonus * 100.0
		stats["mental_resistance"] += delta
		details = append(details, PassiveStatDetail{Key: "mental_resistance", Total: delta, Sources: []PassiveStatSource{{
			Source: "constitution:" + player.Constitution, Name: constitutionName, Value: delta, Final: delta,
		}}})
	}

	// Race-derived bonuses (path resistance, luck)
	raceName := "Chủng tộc"
	if player.Race != "" {
		raceName = gamefx.RaceDisplayName(player.Race)
	}
	for path, pct := range charFX.PathResistanceBonus {
		delta := pct * 100.0
		key := path + "_resistance"
		stats[key] += delta
		details = append(details, PassiveStatDetail{Key: key, Total: delta, Sources: []PassiveStatSource{{
			Source: "race:" + player.Race, Name: raceName, Value: delta, Final: delta,
		}}})
	}
	if charFX.LuckBonus > 0 {
		stats["luck"] += charFX.LuckBonus
		details = append(details, PassiveStatDetail{Key: "luck", Total: charFX.LuckBonus, Sources: []PassiveStatSource{{
			Source: "race:" + player.Race, Name: raceName, Value: charFX.LuckBonus, Final: charFX.LuckBonus,
		}}})
	}

	// Sort details by key for stable output
	sort.Slice(details, func(i, j int) bool { return details[i].Key < details[j].Key })

	return PassiveStatsResult{Stats: stats, Details: details}
}

// GetPassiveStatsHandler returns a detailed breakdown of all passive stats for the authenticated player.
