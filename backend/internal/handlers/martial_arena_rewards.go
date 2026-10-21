package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// loadMartialCombatant loads one combatant's player data into the shared
// teamMemberCombatant shape so buildTeamCombatant can be reused unchanged.
func loadMartialCombatant(q db.Queryer, combatantID string) (*teamMemberCombatant, error) {
	var m teamMemberCombatant
	err := q.QueryRow(`
		SELECT mc.player_id, p.hp, p.base_hp, p.current_mana, p.max_mana,
		       p.username, p.rank, p.sub_rank, p.hp, p.base_hp, p.current_mana, p.max_mana,
		       COALESCE(p.race, ''), COALESCE(p.constitution, ''),
		       `+db.GuInventoryAggregateEquipped("p.id")+`,
		       COALESCE(ia.dao_marks, '{}'::jsonb),
		       COALESCE(p.path_masteries, '{}'::jsonb),
		       COALESCE(p.combat_settings, '{}'::jsonb)
		FROM martial_combatants mc
		JOIN players p ON mc.player_id = p.id
		LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
		WHERE mc.id = $1
	`, combatantID).Scan(
		&m.PlayerID, &m.CurrentHP, &m.MaxHP, &m.CurrentMana, &m.MaxMana,
		&m.Username, &m.Rank, &m.SubRank, &m.PlayerHP, &m.BaseHP, &m.PlayerMana, &m.MaxManaPlayer,
		&m.Race, &m.Constitution, &m.GuInventory, &m.DaoMarks, &m.PathMasteries, &m.CombatSettings,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// martialReward holds the tunable top-3 placement rewards.
// ponytail: guID/items left empty by default — fill valid IDs when the reward
// catalog is decided; the claim/award machinery already handles them.
type martialReward struct {
	title          string
	stones         int
	immortalStones int
	guID           string
	guRank         int
}

var martialPlacementRewards = map[int]martialReward{
	1: {title: "Võ Tôn Đại Hội", stones: 5000, immortalStones: 500},
	2: {title: "Võ Vương Đại Hội", stones: 3000, immortalStones: 300},
	3: {title: "Võ Tướng Đại Hội", stones: 1500, immortalStones: 150},
}

// awardMartialChampionRewards writes claimable rewards for the top 3 placements and
// grants Gu rewards directly (mirrors death_arena).
func awardMartialChampionRewards(tx *sql.Tx, tournamentID string) error {
	placements := getMartialPlacements(tx, tournamentID)
	for i, combatantID := range placements {
		if i >= 3 {
			break
		}
		place := i + 1
		r, ok := martialPlacementRewards[place]
		if !ok {
			continue
		}

		var playerID string
		if err := tx.QueryRow(`SELECT player_id FROM martial_combatants WHERE id = $1`, combatantID).Scan(&playerID); err != nil || playerID == "" {
			continue
		}

		var guIDArg, guRankArg interface{}
		if r.guID != "" {
			guIDArg = r.guID
			guRankArg = r.guRank
		}

		if _, err := tx.Exec(`
			INSERT INTO martial_champion_rewards
			(tournament_id, player_id, placement, combatant_id, title, reward_stones, reward_immortal_stones, reward_gu_id, reward_gu_rank)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, tournamentID, playerID, place, combatantID, r.title, r.stones, r.immortalStones, guIDArg, guRankArg); err != nil {
			logger.S.Errorf("Martial Arena: failed to insert champion reward: %v", err)
		}

		if r.guID != "" {
			gu := models.GuInstance{
				InstanceID: fmt.Sprintf("martial_%s_%d_%s", tournamentID, place, playerID[:8]),
				GuID:       r.guID,
				Rank:       r.guRank,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   false,
				IsVital:    false,
			}
			if err := db.AddPlayerGu(tx, playerID, gu); err != nil {
				logger.S.Errorf("Martial Arena: failed to award Gu %s to %s: %v", r.guID, playerID, err)
			}
		}

		_, _ = tx.Exec(`UPDATE martial_combatants SET final_placement = $1 WHERE id = $2`, place, combatantID)
	}
	return nil
}

// getMartialPlacements returns combatant IDs in placement order (1st, 2nd, 3rd).
func getMartialPlacements(tx *sql.Tx, tournamentID string) []string {
	var championID string
	if err := tx.QueryRow(`
		SELECT COALESCE(winner_combatant_id, combatant_a_id) FROM martial_matches
		WHERE tournament_id = $1 AND round_number = (SELECT total_rounds FROM martial_tournaments WHERE id = $1)
		  AND status IN ('FINISHED','BYE')
		ORDER BY CASE WHEN status = 'FINISHED' THEN 1 ELSE 2 END
		LIMIT 1
	`, tournamentID).Scan(&championID); err != nil || championID == "" {
		return nil
	}

	result := []string{championID}

	var runnerUp string
	if err := tx.QueryRow(`
		SELECT CASE WHEN winner_combatant_id = combatant_a_id THEN combatant_b_id ELSE combatant_a_id END
		FROM martial_matches
		WHERE tournament_id = $1 AND round_number = (SELECT total_rounds FROM martial_tournaments WHERE id = $1)
		  AND status = 'FINISHED' AND combatant_b_id IS NOT NULL
		LIMIT 1
	`, tournamentID).Scan(&runnerUp); err == nil && runnerUp != "" {
		result = append(result, runnerUp)
	}

	if runnerUp != "" {
		var third string
		if err := tx.QueryRow(`
			SELECT CASE WHEN winner_combatant_id = combatant_a_id THEN combatant_b_id ELSE combatant_a_id END
			FROM martial_matches
			WHERE tournament_id = $1 AND round_number = (SELECT total_rounds - 1 FROM martial_tournaments WHERE id = $1)
			  AND status = 'FINISHED' AND combatant_b_id IS NOT NULL
			  AND (CASE WHEN winner_combatant_id = combatant_a_id THEN combatant_b_id ELSE combatant_a_id END) NOT IN ($2, $3)
			LIMIT 1
		`, tournamentID, championID, runnerUp).Scan(&third); err == nil && third != "" {
			result = append(result, third)
		}
	}

	return result
}

// GetMyMartialRewards returns the player's claimable martial arena rewards.
func GetMyMartialRewards(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT mcr.id, mcr.tournament_id, mcr.player_id, p.username, mcr.placement,
		       mcr.combatant_id, mcr.title, mcr.reward_stones, mcr.reward_immortal_stones,
		       mcr.reward_gu_id, mcr.reward_gu_rank, COALESCE(mcr.reward_items, '[]'::jsonb), mcr.claimed, mcr.created_at
		FROM martial_champion_rewards mcr
		JOIN players p ON mcr.player_id = p.id
		WHERE mcr.player_id = $1
		ORDER BY mcr.created_at DESC
	`, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rewards"})
		return
	}
	defer rows.Close()

	rewards := []MartialChampionReward{}
	for rows.Next() {
		var r MartialChampionReward
		if errScan := rows.Scan(&r.ID, &r.TournamentID, &r.PlayerID, &r.Username, &r.Placement,
			&r.CombatantID, &r.Title, &r.RewardStones, &r.RewardImmortalStones,
			&r.RewardGuID, &r.RewardGuRank, &r.RewardItems, &r.Claimed, &r.CreatedAt); errScan == nil {
			rewards = append(rewards, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{"rewards": rewards})
}

// ClaimMartialReward claims a placement reward (stones, immortal stones, title, items).
func ClaimMartialReward(c *gin.Context) {
	var req struct {
		RewardID string `json:"reward_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reward ID is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rewardUUID, err := uuid.Parse(req.RewardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reward ID"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var r MartialChampionReward
	err = tx.QueryRow(`
		SELECT id, player_id, placement, reward_stones, reward_immortal_stones, title, claimed
		FROM martial_champion_rewards
		WHERE id = $1 AND player_id = $2
	`, rewardUUID, authID).Scan(&r.ID, &r.PlayerID, &r.Placement, &r.RewardStones, &r.RewardImmortalStones, &r.Title, &r.Claimed)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phần thưởng"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if r.Claimed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phần thưởng đã được nhận rồi"})
		return
	}

	if r.RewardStones > 0 {
		if _, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, r.RewardStones, authID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to award stones"})
			return
		}
	}
	if r.RewardImmortalStones > 0 {
		if _, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, r.RewardImmortalStones, authID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to award immortal stones"})
			return
		}
	}
	if r.Title != "" {
		if _, err = tx.Exec(`UPDATE players SET martial_title = $1 WHERE id = $2`, r.Title, authID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to award title"})
			return
		}
	}

	// Grant item rewards (reward_items JSONB) if any.
	if InventorySvc != nil {
		var itemsRaw []byte
		_ = tx.QueryRow(`SELECT reward_items FROM martial_champion_rewards WHERE id = $1`, rewardUUID).Scan(&itemsRaw)
		var items []struct {
			ItemID   string `json:"item_id"`
			Category string `json:"category"`
			Quantity int64  `json:"quantity"`
		}
		if len(itemsRaw) > 0 && json.Unmarshal(itemsRaw, &items) == nil {
			for _, it := range items {
				if it.ItemID == "" {
					continue
				}
				if it.Quantity <= 0 {
					it.Quantity = 1
				}
				if err := InventorySvc.AddItem(tx, authID.String(), it.ItemID, it.Category, it.Quantity); err != nil {
					logger.S.Errorf("Martial Arena: failed to grant item %s to %s: %v", it.ItemID, authID, err)
				}
			}
		}
	}

	if _, err = tx.Exec(`UPDATE martial_champion_rewards SET claimed = true, claimed_at = NOW() WHERE id = $1`, rewardUUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as claimed"})
		return
	}

	if err := auditlog.LogChecked(tx, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "arena.martial_claim_reward",
		EntityType:     "martial_arena_reward",
		EntityID:       rewardUUID.String(),
		RequestSummary: fmt.Sprintf("Nhận thưởng Võ Đấu Trường hạng %d: %d Nguyên Thạch, %d Tiên Thạch, danh hiệu '%s'", r.Placement, r.RewardStones, r.RewardImmortalStones, r.Title),
		Metadata: map[string]interface{}{
			"reward_id":       rewardUUID.String(),
			"placement":       r.Placement,
			"stones":          r.RewardStones,
			"immortal_stones": r.RewardImmortalStones,
			"title":           r.Title,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi audit"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Nhận phần thưởng thành công!",
		"placement":       r.Placement,
		"title":           r.Title,
		"stones":          r.RewardStones,
		"immortal_stones": r.RewardImmortalStones,
	})
}
