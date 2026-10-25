package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/match"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// demoKitGuIDs is the fixed low-rank kit the practice match offers. It covers
// every mechanic the realtime mode supports (point target, skillshot, delayed
// ground AoE, cone, blink, shield) so the mode is demonstrable on a fresh
// account that owns none of them.
var demoKitGuIDs = []match.GuRef{
	{GuID: "GU_BLADE_AURA_1", Rank: 1},
	{GuID: "GU_WATER_ARROW", Rank: 1},
	{GuID: "GU_STAR_FALL_1", Rank: 1},
	{GuID: "GU_SOUNDWAVE", Rank: 3},
	{GuID: "GU_SPACE_DOOR_1", Rank: 1},
	{GuID: "GU_LAW_SHIELD_2", Rank: 2},
}

const maxMatchKitSize = 6

// StartPracticeMatch creates a single-player realtime match against a training
// dummy and returns the static setup (arena, roster, kits). Live per-tick state
// arrives on the match:<id> WebSocket channel.
func StartPracticeMatch(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if existing, ok := match.Global.LiveMatchFor(authID); ok {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu đang trong một trận đấu!", "match_id": existing.ID.String()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTxEquippedReadonly(tx, authID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đạo hữu"})
		return
	}

	stats := CalculatePlayerStats(&player)
	maxHP := gamefx.EffectiveMaxHP(&player)
	if maxHP <= 0 {
		maxHP = 100
	}

	arena := match.NewMatch(uuid.New())
	arena.AddParticipant(&match.Participant{
		ID: authID, Name: player.Username,
		X: 6, Y: match.ArenaHeight / 2,
		HP: maxHP, MaxHP: maxHP,
		Mana: match.MaxMana, MaxMana: match.MaxMana,
		Attack: stats["attack"], Defense: stats["defense"], Speed: stats["speed"],
		Kit: append([]match.Skill{match.BasicAttackSkill(stats["attack"])}, buildPlayerKit(&player)...),
	})
	arena.AddParticipant(match.NewDummy(uuid.New(), match.ArenaWidth-6, match.ArenaHeight/2))

	match.Global.Add(arena)
	c.JSON(http.StatusOK, arena.MatchSetup())
}

// buildPlayerKit always includes the demo kit so the mode works on a fresh
// account, then appends any equipped Gu the player owns that already has a
// realtime profile.
//
// ponytail: fixed demo kit + owned extras. Switch to owned-only once enough of
// Gu.csv has GuMoba.csv rows.
func buildPlayerKit(player *models.Player) []match.Skill {
	refs := make([]match.GuRef, 0, len(demoKitGuIDs))
	seen := map[string]bool{}
	for _, ref := range demoKitGuIDs {
		refs = append(refs, ref)
		seen[ref.GuID] = true
	}
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		if seen[gu.GuID] {
			continue
		}
		if _, ok := models.GetMobaProfile(gu.GuID, gu.Rank); !ok {
			continue
		}
		refs = append(refs, match.GuRef{GuID: gu.GuID, Rank: gu.Rank})
		seen[gu.GuID] = true
	}
	if len(refs) > maxMatchKitSize {
		refs = refs[:maxMatchKitSize]
	}
	return match.BuildKit(refs)
}
