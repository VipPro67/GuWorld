package handlers

import (
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// MaxHumanQi caps the amount of Nhân Khí a player can hold.
const MaxHumanQi = 500000.0

// maxCombatQiRank caps the enemy rank used for combat Nhân Khí rewards: Nhân Khí
// after ascension no longer matters to the ascension economy, so rank > 5 clamps.
const maxCombatQiRank = 5

// combatHumanQiReward returns the Nhân Khí granted for a combat victory, scaled
// by the defeated enemy's cultivation rank: R1=3, R2=6, R3=15, R4=60, R5=300.
func combatHumanQiReward(enemyRank int) float64 {
	if enemyRank < 1 {
		enemyRank = 1
	}
	if enemyRank > maxCombatQiRank {
		enemyRank = maxCombatQiRank
	}
	return 3.0 * gamefx.RankScale(enemyRank)
}

// AddHumanQi increases a player's Nhân Khí and returns how much was actually gained.
func AddHumanQi(player *models.Player, amount float64) float64 {
	before := player.HumanQi
	player.HumanQi += amount
	if player.HumanQi > MaxHumanQi {
		player.HumanQi = MaxHumanQi
	}
	return player.HumanQi - before
}
