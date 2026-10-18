package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// filterGuLoadout restricts a combatant's Gu inventory to their saved combat
// loadout (gu_priorities). It first keeps only equipped+ACTIVE Gus with
// combat-relevant active effects (mirroring InitializeCombatState's PVP filter),
// then, if a loadout is configured, restricts to the loadout's gu_ids in order.
// Falls back to all equipped+ACTIVE combat Gus when the loadout is empty or none
// of its gu_ids resolve to a valid instance.
func filterGuLoadout(inv models.GuInventory, cs models.AutoCombatConfig) models.GuInventory {
	combatGu := make(models.GuInventory, 0, len(inv))
	byID := make(map[string]models.GuInstance, len(inv))
	for _, g := range inv {
		if !g.Equipped || g.Status != "ACTIVE" {
			continue
		}
		spec, ok := models.GetGu(g.GuID, g.Rank)
		if !ok {
			spec, ok = models.GetGuAny(g.GuID)
		}
		if !ok || !hasCombatEffects(&spec) {
			continue
		}
		combatGu = append(combatGu, g)
		if best, exists := byID[g.GuID]; !exists || g.Rank > best.Rank {
			byID[g.GuID] = g
		}
	}

	if len(cs.GuPriorities) == 0 {
		return combatGu
	}

	// Preserve loadout order, skipping duplicate gu_ids.
	seen := make(map[string]bool, len(cs.GuPriorities))
	result := make(models.GuInventory, 0, len(cs.GuPriorities))
	for _, t := range cs.GuPriorities {
		if seen[t.GuID] {
			continue
		}
		seen[t.GuID] = true
		if g, ok := byID[t.GuID]; ok {
			result = append(result, g)
		}
	}
	if len(result) == 0 {
		return combatGu
	}
	return result
}

// filterCombatGuInventory restricts a player's Gu inventory for combat display & history.
//  1. When explicit selectedGuIDs are provided (len > 0), only display those selected Gus.
//  2. When no explicit selectedGuIDs are provided, display the Gus configured in combat settings (cs.GuPriorities).
//     If settings is also empty, fall back to all equipped active combat Gus.
func filterCombatGuInventory(inv models.GuInventory, selectedGuIDs []string, cs models.AutoCombatConfig) models.GuInventory {
	if len(inv) == 0 {
		return inv
	}
	if len(selectedGuIDs) > 0 {
		byID := make(map[string]models.GuInstance, len(inv))
		byInstID := make(map[string]models.GuInstance, len(inv))
		for _, g := range inv {
			if !g.Equipped || g.Status != "ACTIVE" {
				continue
			}
			byInstID[g.InstanceID] = g
			if best, exists := byID[g.GuID]; !exists || g.Rank > best.Rank {
				byID[g.GuID] = g
			}
		}
		seen := make(map[string]bool, len(selectedGuIDs))
		result := make(models.GuInventory, 0, len(selectedGuIDs))
		for _, id := range selectedGuIDs {
			if seen[id] {
				continue
			}
			seen[id] = true
			if g, ok := byInstID[id]; ok {
				result = append(result, g)
			} else if g, ok := byID[id]; ok {
				result = append(result, g)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return filterGuLoadout(inv, cs)
}

// GetCombatSettings returns the player's persistent auto-combat settings.
func GetCombatSettings(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var settings models.AutoCombatConfig
	if err := db.DB.QueryRowContext(c.Request.Context(),
		`SELECT combat_settings FROM players WHERE id = $1`, authID,
	).Scan(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"combat_settings": settings})
}

// UpdateCombatSettings saves the player's auto-combat settings.
func UpdateCombatSettings(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		CombatSettings models.AutoCombatConfig `json:"combat_settings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTxEquipped(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	// Sanitize gu_priorities: only keep Gu the player owns, is equipped and ACTIVE.
	ownedActive := make(map[string]bool)
	for _, g := range player.GuInventory {
		if g.Equipped && g.Status == "ACTIVE" {
			ownedActive[g.GuID] = true
		}
	}
	seen := make(map[string]bool)
	validGu := req.CombatSettings.GuPriorities[:0]
	for _, t := range req.CombatSettings.GuPriorities {
		if !ownedActive[t.GuID] || seen[t.GuID] {
			continue
		}
		seen[t.GuID] = true
		if t.Priority < 1 {
			t.Priority = 1
		}
		validGu = append(validGu, t)
	}
	req.CombatSettings.GuPriorities = validGu

	// Sanitize killer_move_priorities: only keep killer moves owned by the player.
	kmRows, err := tx.Query(`SELECT id::text FROM player_killer_moves WHERE player_id = $1`, authID)
	if err == nil {
		ownedKm := make(map[string]bool)
		for kmRows.Next() {
			var kmID string
			if scanErr := kmRows.Scan(&kmID); scanErr == nil {
				ownedKm[kmID] = true
			}
		}
		kmRows.Close()

		seenKm := make(map[string]bool)
		var validKm []models.GuTactic
		kmList := req.CombatSettings.KillerMovePriorities
		if len(kmList) == 0 {
			kmList = req.CombatSettings.KmPriorities
		}
		for _, t := range kmList {
			if !ownedKm[t.GuID] || seenKm[t.GuID] {
				continue
			}
			seenKm[t.GuID] = true
			if t.Priority < 1 {
				t.Priority = 1
			}
			validKm = append(validKm, t)
		}
		req.CombatSettings.KillerMovePriorities = validKm
		req.CombatSettings.KmPriorities = validKm

		// Synchronize is_equipped in player_killer_moves
		if len(validKm) > 0 {
			var activeIDs []string
			for _, t := range validKm {
				activeIDs = append(activeIDs, t.GuID)
			}
			_, _ = tx.Exec(`UPDATE player_killer_moves SET is_equipped = (id = ANY($1)) WHERE player_id = $2`, activeIDs, authID)
		} else {
			_, _ = tx.Exec(`UPDATE player_killer_moves SET is_equipped = false WHERE player_id = $1`, authID)
		}
	}

	// Sanitize category_priorities: ensure complete permutation of shield, heal, damage, other.
	validCats := map[string]bool{"shield": true, "heal": true, "damage": true, "other": true}
	var cleanCats []string
	seenCat := make(map[string]bool)
	for _, cat := range req.CombatSettings.CategoryPriorities {
		if validCats[cat] && !seenCat[cat] {
			seenCat[cat] = true
			cleanCats = append(cleanCats, cat)
		}
	}
	for _, cat := range []string{"other", "shield", "heal", "damage"} {
		if !seenCat[cat] {
			cleanCats = append(cleanCats, cat)
		}
	}
	req.CombatSettings.CategoryPriorities = cleanCats

	settingsJSON, _ := json.Marshal(req.CombatSettings)
	_, err = tx.Exec(`UPDATE players SET combat_settings = $1 WHERE id = $2`, settingsJSON, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save combat settings"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Cập nhật cài đặt chiến đấu thành công!",
		"combat_settings": req.CombatSettings,
	})
}
