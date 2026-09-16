package handlers

import (
	"math"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func CalculatePlayerStats(player *models.Player) map[string]float64 {
	return GetPassiveStats(player).Stats
}

// ── Passive Stats Breakdown ──────────────────────────────────────────────────

// PassiveStatSource describes one source contributing to a passive stat.
type PassiveStatSource struct {
	Source string  `json:"source"` // e.g. "base", "growth", "gu:GU_JADE_SKIN"
	Name   string  `json:"name"`   // human-readable label
	Value  float64 `json:"value"`  // raw contribution before multipliers
	Mult   float64 `json:"mult"`   // path mastery multiplier applied
	Dao    float64 `json:"dao"`    // dao mark multiplier applied
	Final  float64 `json:"final"`  // actual contribution = value * mult * dao
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
	strengthMarks := int64(0)
	if player.DaoMarks != nil {
		strengthMarks = player.DaoMarks["strength"]
	}

	// ── Base stats from Rank / SubRank ──
	baseAttack := 10.0 + float64(player.Rank-1)*10.0 + float64(player.SubRank-1)*2.0 + float64(strengthMarks)/10.0
	baseDefense := 5.0 + float64(player.Rank-1)*5.0 + float64(player.SubRank-1)*1.0
	baseSpeed := 10.0 + float64(player.Rank-1)*2.0 + float64(player.SubRank-1)*0.5

	// Accumulator: key -> []PassiveStatSource
	acc := map[string][]PassiveStatSource{}
	addSrc := func(key, source, name string, value, mult, dao float64) {
		final := value * mult * dao
		if final == 0 && value == 0 {
			return
		}
		acc[key] = append(acc[key], PassiveStatSource{
			Source: source, Name: name, Value: value, Mult: mult, Dao: dao, Final: final,
		})
	}

	// Base sources
	addSrc("attack", "base", "Cơ bản (Cấp bậc)", baseAttack, 1, 1)
	addSrc("defense", "base", "Cơ bản (Cấp bậc)", baseDefense, 1, 1)
	addSrc("speed", "base", "Cơ bản (Cấp bậc)", baseSpeed, 1, 1)
	addSrc("movement", "base", "Cơ bản", 1.0, 1, 1)
	addSrc("investigation", "base", "Cơ bản", 1.0, 1, 1)
	addSrc("storage", "base", "Cơ bản", 100.0, 1, 1)

	// Growth boosts
	if player.GrowthData.Boosts.Attack > 0 {
		addSrc("attack", "growth", "Trưởng thành Cổ", player.GrowthData.Boosts.Attack, 1, 1)
	}
	if player.GrowthData.Boosts.Defense > 0 {
		addSrc("defense", "growth", "Trưởng thành Cổ", player.GrowthData.Boosts.Defense, 1, 1)
	}
	if player.GrowthData.Boosts.Speed > 0 {
		addSrc("speed", "growth", "Trưởng thành Cổ", player.GrowthData.Boosts.Speed, 1, 1)
	}

	// Equipped active Gu passives — each GuID contributes only its highest rank,
	// applied exactly once (duplicates never stack).
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
			if !ok || spec.Effects.Passive == nil {
				continue
			}
			passive := spec.Effects.Passive
			mult := getPathMasteryMultiplier(player, spec.Path)

			daoMult := 1.0
			if player.DaoMarks != nil {
				maxDaoMult := 1.0
				for _, path := range spec.Path {
					cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
					if amt, ok := player.DaoMarks[cleanPath]; ok && amt > 0 {
						pm := 1.0 + float64(amt)/100.0
						if pm > maxDaoMult {
							maxDaoMult = pm
						}
					}
				}
				daoMult = maxDaoMult
			}

			srcID := "gu:" + gu.GuID
			if passive.StatBaseAttack > 0 {
				addSrc("attack", srcID, spec.Name, passive.StatBaseAttack, mult, daoMult)
			}
			if passive.StatBaseDefense > 0 {
				addSrc("defense", srcID, spec.Name, passive.StatBaseDefense, mult, daoMult)
			}
			if passive.StatBaseSpeed > 0 {
				addSrc("speed", srcID, spec.Name, passive.StatBaseSpeed, mult, 1)
			}
			if passive.StatBaseHP > 0 {
				addSrc("hp", srcID, spec.Name, passive.StatBaseHP, mult, daoMult)
			}
			if passive.DamageMultiplierOnHit > 0 && player.BaseHP > 0 {
				missingHpRate := 1.0 - player.HP/player.BaseHP
				addSrc("damage_multiplier", srcID, spec.Name, missingHpRate*passive.DamageMultiplierOnHit, 1, 1)
			}
			if passive.SoulRefinement > 0 {
				addSrc("soul", srcID, spec.Name, passive.SoulRefinement, 1, 1)
			}
			if passive.StorageCapacity > 0 {
				addSrc("storage", srcID, spec.Name, passive.StorageCapacity, 1, 1)
			}
			if passive.Movement > 0 {
				addSrc("movement", srcID, spec.Name, passive.Movement, 1, 1)
			}
			if passive.Investigation > 0 {
				addSrc("investigation", srcID, spec.Name, passive.Investigation, 1, 1)
			}
			if passive.Hearing > 0 {
				addSrc("hearing", srcID, spec.Name, passive.Hearing, 1, 1)
			}
			if passive.Evasion > 0 {
				addSrc("evasion", srcID, spec.Name, passive.Evasion, 1, 1)
			}
			if passive.Accuracy > 0 {
				addSrc("accuracy", srcID, spec.Name, passive.Accuracy, 1, 1)
			}
			if passive.CleansingPower > 0 {
				addSrc("cleansing_power", srcID, spec.Name, passive.CleansingPower, 1, 1)
			}
			if passive.Luck > 0 {
				addSrc("luck", srcID, spec.Name, passive.Luck, 1, 1)
			}
		}
	}

	// Active statuses (World Map buffs)
	for _, st := range player.Statuses.Active() {
		if st.ID == "buff_atk" {
			addSrc("attack", "status:"+st.ID, st.Source, st.Value, 1, 1)
		}
		if st.ID == "buff_def" {
			addSrc("defense", "status:"+st.ID, st.Source, st.Value, 1, 1)
		}
		if st.ID == "buff_speed" {
			addSrc("speed", "status:"+st.ID, st.Source, st.Value, 1, 1)
		}
	}

	// Sect Territory AoE Buff
	if player.ID.String() != "" {
		territory := CheckPlayerSectTerritory(db.DB, player.ID, player.CurrentRegion, player.X, player.Y)
		if territory.InTerritory {
			// Player is inside their sect's territory!
			// Give +10% base attack/defense buff, plus 2% per main hall level
			buffMult := 0.10 + float64(territory.MainHallLvl)*0.02
			addSrc("attack", "sect_territory", "Hộ Sơn", baseAttack*buffMult, 1, 1)
			addSrc("defense", "sect_territory", "Hộ Sơn", baseDefense*buffMult, 1, 1)
		}
	}

	// Build totals and details
	stats := map[string]float64{
		"attack":        0.0,
		"defense":       0.0,
		"speed":         0.0,
		"movement":      0.0,
		"investigation": 0.0,
		"storage":       0.0,
		"accuracy":      0.0,
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
					Source: "dynamic", Name: "Lực tấn công (×0.5)", Value: storageBonus, Mult: 1, Dao: 1, Final: storageBonus,
				})
				details[i].Total += storageBonus
				break
			}
		}
	}

	// Apply speed_boost_flat from equipped Gu active effects
	{
		totalSpeedBoost := 0.0
		for _, gu := range player.GuInventory {
			if gu.Equipped && gu.Status == "ACTIVE" {
				spec, ok := models.GetGu(gu.GuID, gu.Rank)
				if !ok || spec.Effects.Active == nil {
					continue
				}
				if spec.Effects.Active.SpeedBoostFlat > 0 {
					totalSpeedBoost += spec.Effects.Active.SpeedBoostFlat
				}
			}
		}
		if totalSpeedBoost > 0 {
			stats["speed"] += totalSpeedBoost
			details = append(details, PassiveStatDetail{Key: "speed", Total: totalSpeedBoost, Sources: []PassiveStatSource{{
				Source: "gu:active", Name: "Speed Boost", Value: totalSpeedBoost, Mult: 1, Dao: 1, Final: totalSpeedBoost,
			}}})
		}
	}

	// Apply all_stats_buff_percent from equipped Gu active effects
	{
		totalBuffPct := 0.0
		for _, gu := range player.GuInventory {
			if gu.Equipped && gu.Status == "ACTIVE" {
				spec, ok := models.GetGu(gu.GuID, gu.Rank)
				if !ok || spec.Effects.Active == nil {
					continue
				}
				if spec.Effects.Active.AllStatsBuffPct > 0 {
					totalBuffPct += spec.Effects.Active.AllStatsBuffPct
				}
			}
		}
		if totalBuffPct > 0 {
			for key := range stats {
				if key == "storage" || key == "accuracy" {
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
								Source: "gu:active", Name: "All Stats Buff", Value: delta, Mult: 1, Dao: 1, Final: delta,
							})
							details[i].Total += delta
							found = true
							break
						}
					}
					if !found {
						details = append(details, PassiveStatDetail{Key: key, Total: delta, Sources: []PassiveStatSource{{
							Source: "gu:active", Name: "All Stats Buff", Value: delta, Mult: 1, Dao: 1, Final: delta,
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
						Source: sourceID, Name: sourceName, Value: delta, Mult: 1, Dao: 1, Final: delta,
					})
					details[i].Total += delta
					found = true
					break
				}
			}
			if !found {
				details = append(details, PassiveStatDetail{Key: key, Total: stats[key], Sources: []PassiveStatSource{{
					Source: sourceID, Name: sourceName, Value: delta, Mult: 1, Dao: 1, Final: delta,
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
				statMods := def.Effects.StatMods
				if st.Value < 0 && len(statMods) > 0 {
					statMods = map[string]float64{}
					for k := range def.Effects.StatMods {
						statMods[k] = st.Value
					}
				}
				applyCharEffect("status:"+def.Key, "Trạng thái: "+def.Name, statMods)
				break
			}
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
			Source: "constitution:" + player.Constitution, Name: constitutionName, Value: delta, Mult: 1, Dao: 1, Final: delta,
		}}})
	}
	if charFX.MentalResistanceBonus > 0 {
		delta := charFX.MentalResistanceBonus * 100.0
		stats["mental_resistance"] += delta
		details = append(details, PassiveStatDetail{Key: "mental_resistance", Total: delta, Sources: []PassiveStatSource{{
			Source: "constitution:" + player.Constitution, Name: constitutionName, Value: delta, Mult: 1, Dao: 1, Final: delta,
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
			Source: "race:" + player.Race, Name: raceName, Value: delta, Mult: 1, Dao: 1, Final: delta,
		}}})
	}
	if charFX.LuckBonus > 0 {
		stats["luck"] += charFX.LuckBonus
		details = append(details, PassiveStatDetail{Key: "luck", Total: charFX.LuckBonus, Sources: []PassiveStatSource{{
			Source: "race:" + player.Race, Name: raceName, Value: charFX.LuckBonus, Mult: 1, Dao: 1, Final: charFX.LuckBonus,
		}}})
	}

	// Sort details by key for stable output
	sort.Slice(details, func(i, j int) bool { return details[i].Key < details[j].Key })

	return PassiveStatsResult{Stats: stats, Details: details}
}

// GetPassiveStatsHandler returns a detailed breakdown of all passive stats for the authenticated player.
