package handlers

import (
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func getActorPathMasteryMultiplier(actor *models.Combatant, paths []string) float64 {
	return gamefx.PathMasteryMultiplier(actor.PathMasteries, paths)
}

// GetGuAttainmentScale calculates the path attainment multiplier and reason for a Gu action.
// Level 0 = 1.0 (Ordinary / Phổ thông).
// Level 1-4 scale up progressively (1.1x, 1.25x, 1.4x, 1.5x).
func GetGuAttainmentScale(actor *models.Combatant, guSpec *models.GuDictionaryItem) (float64, string) {
	if guSpec == nil || len(guSpec.Path) == 0 {
		return 1.0, ""
	}
	masteryMultiplier := getActorPathMasteryMultiplier(actor, guSpec.Path)
	if masteryMultiplier <= 0 {
		masteryMultiplier = 1.0
	}
	reason := ""
	if masteryMultiplier > 1.0 {
		attainmentLevel := 0
		if actor.PathMasteries != nil {
			for _, path := range guSpec.Path {
				cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
				if points, ok := actor.PathMasteries[cleanPath]; ok {
					lvl := gamefx.PathMasteryLevel(points)
					if lvl > attainmentLevel {
						attainmentLevel = lvl
					}
				}
			}
		}
		levelName := gamefx.PathMasteryLevelName(attainmentLevel)
		reason = fmt.Sprintf("Cảnh giới lưu phái %s (x%.2f)", levelName, masteryMultiplier)
	}
	return masteryMultiplier, reason
}

func CalculateGuEffectForCombatant(actor *models.Combatant, guSpec *models.GuDictionaryItem, applyDao bool, bonuses ...map[string]float64) (effectiveDamage float64, effectiveDefense float64, manaCost float64, err error) {
	// Mortal users cannot use immortal Gu that cost mana.
	if !gamefx.CanUseGuWithMana(actor.Rank, guSpec) {
		return 0, 0, 0, fmt.Errorf("Tu vi %s chưa đủ để sử dụng Tiên Cổ %s (Rank %d) có hao phí!", models.GetRankName(actor.Rank), guSpec.Name, guSpec.Rank)
	}

	var baseDmg float64
	var baseDef float64
	if guSpec.Effects.Active != nil {
		baseDmg = guSpec.Effects.Active.DamageFlat
		baseDef = guSpec.Effects.Active.ShieldFlat
	} else {
		for _, block := range guSpec.Effects.Blocks {
			if block.Trigger == gameplayeffects.TriggerOnAction {
				for _, op := range block.Operations {
					if op.Op == gameplayeffects.OpDamage && (strings.EqualFold(op.SubType, "FLAT") || op.SubType == "") {
						baseDmg += op.Value
					}
					if op.Op == gameplayeffects.OpShield && (strings.EqualFold(op.SubType, "FLAT") || op.SubType == "") {
						baseDef += op.Value
					}
				}
			}
		}
	}

	if !applyDao {
		// Building action for simulator: preserve exact Gu flat values so breakdown
		// displays the authentic base stats. Scaling and reasons are applied via Action multipliers.
		cost := gamefx.EnergyCostForCombatant(actor, guSpec)
		return baseDmg, baseDef, cost, nil
	}

	powerScale, _ := GetGuAttainmentScale(actor, guSpec)
	masteryMultiplier := getActorPathMasteryMultiplier(actor, guSpec.Path)

	daoMultiplier := 1.0
	var daoBonus, pathDamage map[string]float64
	if len(bonuses) > 0 {
		daoBonus = bonuses[0]
	}
	if len(bonuses) > 1 {
		pathDamage = bonuses[1]
	}
	pathKey := "strength"
	if len(guSpec.Path) > 0 {
		pathKey = guSpec.Path[0]
	}
	conflictReductionPct := actor.DaoConflictReductionPct
	if conflictReductionPct == 0 && !actor.HasNonConflictingDao {
		conflictReductionPct = combatantDaoConflictReduction(actor.GuInventory)
	}
	profile := gamefx.BuildDaoCombatProfileWithReduction(actor.DaoMarks, actor.PathMasteries, actor.HasNonConflictingDao, conflictReductionPct, daoBonus)
	daoAmp, realmEff, conflictMult := profile.GetOffensiveFactors(pathKey)
	daoMultiplier = (1.0 + (daoAmp-1.0)*realmEff) * conflictMult
	masteryMultiplier = 1.0 // integrated in DaoCombatProfile

	if baseDmg > 0 {
		effectiveDamage = baseDmg * powerScale * masteryMultiplier * daoMultiplier
	} else {
		effectiveDamage = 0
	}
	effectiveDamage *= gamefx.PathDamageMultiplierForPaths(guSpec.Path, pathDamage)
	if baseDef > 0 {
		effectiveDefense = baseDef * powerScale * masteryMultiplier * daoMultiplier
	}
	cost := gamefx.EnergyCostForCombatant(actor, guSpec)

	return effectiveDamage, effectiveDefense, cost, nil
}

// combatantDaoConflictReduction projects equipped Gu effects for combatants
// assembled outside the full player-stat pipeline (for example map actions).
func combatantDaoConflictReduction(inventory models.GuInventory) float64 {
	bestRank := make(map[string]int)
	for _, gu := range inventory {
		if gu.Equipped && gu.Status == "ACTIVE" && gu.Rank > bestRank[gu.GuID] {
			bestRank[gu.GuID] = gu.Rank
		}
	}
	reduction := 0.0
	for guID, rank := range bestRank {
		spec, ok := models.GetGu(guID, rank)
		if !ok {
			continue
		}
		reduction = gamefx.CombineDaoConflictReduction(reduction, models.GuStatValue(spec, "PATH_CONFLICT_REDUCTION"))
	}
	return reduction
}

func CalculateGuEffect(player *models.Player, guSpec *models.GuDictionaryItem) (effectiveDamage float64, effectiveDefense float64, manaCost float64, err error) {
	actor := models.CombatantFromPlayer(player, CalculatePlayerStats(player))
	charFX := gamefx.Get(player)
	actor.HasNonConflictingDao = charFX.NonConflictingDaoMarks
	actor.DaoConflictReductionPct = charFX.DaoConflictReductionPct
	dmg, def, cost, err := CalculateGuEffectForCombatant(&actor, guSpec, true, charFX.DaoNganBonus, charFX.PathDamageBonus)
	player.HP = actor.HP
	return dmg, def, cost, err
}
