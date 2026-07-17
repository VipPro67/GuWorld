package handlers

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func getActorPathMasteryMultiplier(actor *models.Combatant, paths []string) float64 {
	masteryMultiplier := 1.0
	if actor.PathMasteries != nil {
		for _, path := range paths {
			cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
			if mLevel, ok := actor.PathMasteries[cleanPath]; ok {
				mult := 1.0
				switch mLevel {
				case "Ordinary":
					mult = 1.0
				case "Master":
					mult = 1.1
				case "Grandmaster":
					mult = 1.25
				case "Great Grandmaster":
					mult = 1.4
				case "Supreme Grandmaster":
					mult = 1.5
				}
				if mult > masteryMultiplier {
					masteryMultiplier = mult
				}
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

	var baseDmg float64
	var baseDef float64
	if guSpec.Effects.Active != nil {
		baseDmg = guSpec.Effects.Active.DamageFlat
		baseDef = guSpec.Effects.Active.ShieldFlat
	}

	attainmentLevel := 0
	if actor.PathMasteries != nil {
		for _, path := range guSpec.Path {
			cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
			if mLevel, ok := actor.PathMasteries[cleanPath]; ok {
				lvl := 0
				switch mLevel {
				case "Ordinary":
					lvl = 0
				case "Master":
					lvl = 1
				case "Grandmaster":
					lvl = 2
				case "Great Grandmaster":
					lvl = 3
				case "Supreme Grandmaster":
					lvl = 4
				}
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
	if actor.DaoMarks != nil {
		maxDaoMultiplier := 1.0
		for _, path := range guSpec.Path {
			cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
			if amt, ok := actor.DaoMarks[cleanPath]; ok && amt > 0 {
				var mult float64
				if amt <= 10000 {
					mult = 1.0 + float64(amt)/100.0
				} else {
					mult = 1.0 + 100.0 + float64(amt-10000)/500.0
				}
				if mult > maxDaoMultiplier {
					maxDaoMultiplier = mult
				}
			}
		}
		daoMultiplier = maxDaoMultiplier
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
