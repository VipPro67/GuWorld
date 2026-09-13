package handlers

import (
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// MaxHumanQi caps the amount of Nhân Khí a player can hold.
const MaxHumanQi = 500000.0

// AddHumanQi increases a player's Nhân Khí and returns how much was actually gained.
func AddHumanQi(player *models.Player, amount float64) float64 {
	before := player.HumanQi
	player.HumanQi += amount
	if player.HumanQi > MaxHumanQi {
		player.HumanQi = MaxHumanQi
	}
	return player.HumanQi - before
}
