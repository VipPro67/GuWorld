package handlers

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func getActorPathMasteryMultiplier(actor *models.Combatant, paths []string) float64 {
	return gamefx.PathMasteryMultiplier(actor.PathMasteries, paths)
}

func CalculateGuEffectForCombatant(actor *models.Combatant, guSpec *models.GuDictionaryItem, applyDao bool, bonuses ...map[string]float64) (effectiveDamage float64, effectiveDefense float64, manaCost float64, err error) {
	// Mortal users cannot use immortal Gu that cost mana.
	if !gamefx.CanUseGuWithMana(actor.Rank, guSpec) {
		return 0, 0, 0, fmt.Errorf("Tu vi %s chưa đủ để sử dụng Tiên Cổ %s (Rank %d) có hao phí!", models.GetRankName(actor.Rank), guSpec.Name, guSpec.Rank)
	}

	powerScale := 1.0

	var baseDmg float64
	var baseDef float64
	if guSpec.Effects.Active != nil {
		baseDmg = guSpec.Effects.Active.DamageFlat
		if baseDmg == 0 && guSpec.Effects.Active.DamageMult > 0 {
			baseDmg = actor.BaseAttack * guSpec.Effects.Active.DamageMult
		} else if baseDmg > 0 && guSpec.Effects.Active.DamageMult > 1.0 {
			baseDmg *= guSpec.Effects.Active.DamageMult
		}
		baseDef = guSpec.Effects.Active.ShieldFlat
	}

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
	if attainmentLevel < 2 {
		powerScale *= 0.70
	}

	if attainmentLevel >= 3 && actor.Kind == "Player" {
		currentRegion := ""
		playerUUID, parseErr := uuid.Parse(actor.ID)
		if parseErr == nil {
			_ = db.DB.QueryRow(`SELECT current_region FROM players WHERE id = $1`, playerUUID).Scan(&currentRegion)
		}
		isResonant := false
		for _, path := range guSpec.Path {
			cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
			switch strings.ToLower(currentRegion) {
			case "eastern_sea":
				if cleanPath == "water" {
					isResonant = true
				}
			case "western_desert":
				if cleanPath == "fire" {
					isResonant = true
				}
			case "northern_plains":
				if cleanPath == "wind" {
					isResonant = true
				}
			case "southern_border":
				if cleanPath == "earth" {
					isResonant = true
				}
			case "central_continent":
				if cleanPath == "qi" {
					isResonant = true
				}
			}
		}
		if isResonant {
			powerScale *= 1.50
		}
	}

	masteryMultiplier := getActorPathMasteryMultiplier(actor, guSpec.Path)

	daoMultiplier := 1.0
	var daoBonus, pathDamage map[string]float64
	if len(bonuses) > 0 {
		daoBonus = bonuses[0]
	}
	if len(bonuses) > 1 {
		pathDamage = bonuses[1]
	}
	if applyDao {
		daoMultiplier = gamefx.DaoMultiplierForPaths(actor.DaoMarks, guSpec.Path, daoBonus)
	}

	// Gu skill damage scales with player's base attack so it stays relevant vs basic attack
	if baseDmg > 0 {
		effectiveDamage = (baseDmg + actor.BaseAttack) * powerScale * masteryMultiplier * daoMultiplier
	} else {
		effectiveDamage = 0
	}
	effectiveDamage *= gamefx.PathDamageMultiplierForPaths(guSpec.Path, pathDamage)
	if baseDef > 0 {
		effectiveDefense += actor.BaseDefense * powerScale * masteryMultiplier * daoMultiplier
	}
	cost := gamefx.EnergyCostForCombatant(actor, guSpec)

	return effectiveDamage, effectiveDefense, cost, nil
}

func CalculateGuEffect(player *models.Player, guSpec *models.GuDictionaryItem) (effectiveDamage float64, effectiveDefense float64, manaCost float64, err error) {
	actor := models.CombatantFromPlayer(player, CalculatePlayerStats(player))
	dmg, def, cost, err := CalculateGuEffectForCombatant(&actor, guSpec, true, gamefx.Get(player).DaoNganBonus, gamefx.Get(player).PathDamageBonus)
	player.HP = actor.HP
	return dmg, def, cost, err
}
