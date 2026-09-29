package util

import "github.com/vippro67/GuWorld/backend/internal/models"

// ImmortalRankThreshold defines the rank at which a cultivator is considered immortal.
const ImmortalRankThreshold = 6

// IsImmortalRank returns true if the given rank meets or exceeds the immortal threshold.
func IsImmortalRank(rank int) bool {
	return rank >= ImmortalRankThreshold
}

// IsImmortalPlayer returns true if the player is an immortal cultivator based on rank.
func IsImmortalPlayer(p *models.Player) bool {
	if p == nil {
		return false
	}
	return IsImmortalRank(p.Rank)
}
