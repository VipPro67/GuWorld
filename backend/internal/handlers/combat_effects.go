package handlers

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func getActorPathMasteryMultiplier(actor *models.Combatant, path string) float64 {
	masteryMultiplier := 1.0
	if actor.PathMasteries != nil {
		cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
		if mLevel, ok := actor.PathMasteries[cleanPath]; ok {
			switch mLevel {
			case "Ordinary":
				masteryMultiplier = 1.0
			case "Master":
				masteryMultiplier = 1.1
			case "Grandmaster":
				masteryMultiplier = 1.25
			case "Great Grandmaster":
				masteryMultiplier = 1.4
			case "Supreme Grandmaster":
				masteryMultiplier = 1.5
			}
		}
	}
	return masteryMultiplier
}

func CalculateGuEffectForCombatant(actor *models.Combatant, guSpec *models.GuDictionaryItem) (effectiveDamage float64, effectiveDefense float64, manaCost float64, err error) {
	if actor.Rank < guSpec.Rank {
		backlashDmg := 50.0
		if float64(actor.HP) <= backlashDmg {
			actor.HP = 0
		} else {
			actor.HP -= backlashDmg
		}
		return 0, 0, 0, fmt.Errorf("Backlash: Cổ trùng Rank %d phản phệ Không Khiếu Rank %d của %s! (Trừ 50 HP)", guSpec.Rank, actor.Rank, actor.Name)
	}

	powerScale := 1.0

	activeEffects := models.GetGuEffectSection(*guSpec, "active")
	var baseDmg float64
	if dmg, ok := activeEffects["damage"]; ok {
		baseDmg = dmg
	} else if dmg, ok := activeEffects["add_base_attack"]; ok {
		baseDmg = dmg
	} else if dmg, ok := activeEffects["base_damage_bonus"]; ok {
		baseDmg = dmg
	}

	var baseDef float64
	if def, ok := activeEffects["defense"]; ok {
		baseDef = def
	} else if def, ok := activeEffects["add_base_defense"]; ok {
		baseDef = def
	} else if def, ok := activeEffects["base_defense_bonus"]; ok {
		baseDef = def
	}

	cleanPath := strings.TrimSuffix(strings.ToLower(guSpec.Path), " path")

	attainmentLevel := 0
	if actor.PathMasteries != nil {
		if mLevel, ok := actor.PathMasteries[cleanPath]; ok {
			switch mLevel {
			case "Ordinary":
				attainmentLevel = 0
			case "Master":
				attainmentLevel = 1
			case "Grandmaster":
				attainmentLevel = 2
			case "Great Grandmaster":
				attainmentLevel = 3
			case "Supreme Grandmaster":
				attainmentLevel = 4
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
		switch strings.ToLower(currentRegion) {
		case "eastern_sea":
			isResonant = (cleanPath == "water")
		case "western_desert":
			isResonant = (cleanPath == "fire")
		case "northern_plains":
			isResonant = (cleanPath == "wind")
		case "southern_border":
			isResonant = (cleanPath == "earth")
		case "central_continent":
			isResonant = (cleanPath == "qi")
		}
		if isResonant {
			powerScale *= 1.50
		}
	}

	masteryMultiplier := getActorPathMasteryMultiplier(actor, guSpec.Path)

	daoMultiplier := 1.0
	if actor.DaoMarks != nil {
		if amt, ok := actor.DaoMarks[cleanPath]; ok && amt > 0 {
			if amt <= 10000 {
				daoMultiplier = 1.0 + float64(amt)/100.0
			} else {
				daoMultiplier = 1.0 + 100.0 + float64(amt-10000)/500.0
			}
		}
	}

	// Gu skill damage scales with player's base attack so it stays relevant vs basic attack
	effectiveDamage = (baseDmg + actor.BaseAttack) * powerScale * masteryMultiplier * daoMultiplier
	effectiveDefense = baseDef * powerScale * masteryMultiplier * daoMultiplier
	if baseDef > 0 {
		effectiveDefense += actor.BaseDefense * powerScale * masteryMultiplier * daoMultiplier
	}

	cost := calculateEnergyCostForCombatant(actor, guSpec)

	return effectiveDamage, effectiveDefense, cost, nil
}

func CalculateGuEffect(player *models.Player, guSpec *models.GuDictionaryItem) (effectiveDamage float64, effectiveDefense float64, manaCost float64, err error) {
	actor := models.CombatantFromPlayer(player, CalculatePlayerStats(player))
	dmg, def, cost, err := CalculateGuEffectForCombatant(&actor, guSpec)
	player.HP = actor.HP
	return dmg, def, cost, err
}
