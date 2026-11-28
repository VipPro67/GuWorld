package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type placementQueryConfig struct {
	TournamentTable  string
	ChampionColumn   string
	MatchTable       string
	SideAColumn      string
	SideBColumn      string
	WinnerSideColumn string
}

// resolveTournamentPlacements returns ordered participant/team IDs (1st, 2nd, 3rd)
// using the actual finalRound so early-finished brackets still resolve placements.
func resolveTournamentPlacements(tx *sql.Tx, cfg placementQueryConfig, tournamentID, championID string, finalRound int) ([]string, error) {
	if finalRound <= 0 {
		if err := tx.QueryRow(
			fmt.Sprintf(`SELECT COALESCE(NULLIF(current_round, 0), total_rounds) FROM %s WHERE id = $1`, cfg.TournamentTable),
			tournamentID,
		).Scan(&finalRound); err != nil {
			return nil, err
		}
	}

	if championID == "" {
		var champ sql.NullString
		err := tx.QueryRow(
			fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1`, cfg.ChampionColumn, cfg.TournamentTable),
			tournamentID,
		).Scan(&champ)
		if err != nil {
			return nil, err
		}
		if champ.Valid {
			championID = champ.String
		}
	}
	if championID == "" {
		return nil, fmt.Errorf("tournament %s has no champion", tournamentID)
	}

	result := []string{championID}

	var runnerUp sql.NullString
	err := tx.QueryRow(fmt.Sprintf(`
		SELECT CASE WHEN %s = %s THEN %s ELSE %s END
		FROM %s
		WHERE tournament_id = $1 AND round_number = $2 AND status = 'FINISHED' AND %s IS NOT NULL
		LIMIT 1
	`, cfg.WinnerSideColumn, cfg.SideAColumn, cfg.SideBColumn, cfg.SideAColumn, cfg.MatchTable, cfg.SideBColumn),
		tournamentID, finalRound,
	).Scan(&runnerUp)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if runnerUp.Valid && runnerUp.String != "" && runnerUp.String != championID {
		result = append(result, runnerUp.String)
	}

	if finalRound > 1 && runnerUp.Valid && runnerUp.String != "" {
		var third sql.NullString
		err = tx.QueryRow(fmt.Sprintf(`
			SELECT CASE WHEN %s = %s THEN %s ELSE %s END
			FROM %s
			WHERE tournament_id = $1
			  AND round_number = $2
			  AND status = 'FINISHED'
			  AND %s IS NOT NULL
			  AND (CASE WHEN %s = %s THEN %s ELSE %s END) NOT IN ($3, $4)
			LIMIT 1
		`, cfg.WinnerSideColumn, cfg.SideAColumn, cfg.SideBColumn, cfg.SideAColumn, cfg.MatchTable, cfg.SideBColumn,
			cfg.WinnerSideColumn, cfg.SideAColumn, cfg.SideBColumn, cfg.SideAColumn),
			tournamentID, finalRound-1, championID, runnerUp.String,
		).Scan(&third)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if third.Valid && third.String != "" {
			result = append(result, third.String)
		}
	}

	return result, nil
}

type tournamentRewardClaimSpec struct {
	TableName      string
	TitleColumn    string
	GuInstancePref string
	AuditAction    string
	AuditEntity    string
	AuditLabel     string
}

var (
	deathArenaRewardClaimSpec = tournamentRewardClaimSpec{
		TableName:      "death_arena_champion_rewards",
		TitleColumn:    "death_arena_title",
		GuInstancePref: "death_arena",
		AuditAction:    "tournament.claim_reward",
		AuditEntity:    "death_arena_reward",
		AuditLabel:     "Tử Chiến Đấu Trường",
	}
	martialRewardClaimSpec = tournamentRewardClaimSpec{
		TableName:      "martial_champion_rewards",
		TitleColumn:    "martial_title",
		GuInstancePref: "martial",
		AuditAction:    "arena.martial_claim_reward",
		AuditEntity:    "martial_arena_reward",
		AuditLabel:     "Đại Hội Võ Thuật",
	}
)

// claimTournamentReward settles stones, title, Gu, and items for a single reward row.
// Gu inventory checks happen here at claim time so a full aperture never blocks tournament finish.
func claimTournamentReward(c *gin.Context, spec tournamentRewardClaimSpec) {
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
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	var (
		tournamentID   string
		placement      int
		title          string
		stones         int
		immortalStones int
		guID           string
		guRank         int
		itemsRaw       []byte
		claimed        bool
	)
	err = tx.QueryRow(fmt.Sprintf(`
		SELECT tournament_id, placement, COALESCE(title, ''), reward_stones, reward_immortal_stones,
		       COALESCE(reward_gu_id, ''), COALESCE(reward_gu_rank, 0), COALESCE(reward_items, '[]'::jsonb), claimed
		FROM %s
		WHERE id = $1 AND player_id = $2
		FOR UPDATE
	`, spec.TableName), rewardUUID, authID).Scan(
		&tournamentID, &placement, &title, &stones, &immortalStones,
		&guID, &guRank, &itemsRaw, &claimed,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phần thưởng"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	if claimed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phần thưởng đã được nhận rồi"})
		return
	}

	if stones > 0 {
		if _, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, stones, authID); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to award stones", err))
			return
		}
	}

	if immortalStones > 0 {
		if _, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`, immortalStones, authID); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to award immortal stones", err))
			return
		}
	}

	if title != "" {
		if _, err = tx.Exec(fmt.Sprintf(`UPDATE players SET %s = $1 WHERE id = $2`, spec.TitleColumn), title, authID); err != nil {
			httperr.Respond(c, httperr.Internal("Failed to award title", err))
			return
		}
	}

	if guID != "" {
		rank := guRank
		if rank <= 0 {
			rank = 1
		}
		playerPrefix := authID.String()
		if len(playerPrefix) > 8 {
			playerPrefix = playerPrefix[:8]
		}
		newGu := models.GuInstance{
			InstanceID: fmt.Sprintf("%s_%s_%d_%s", spec.GuInstancePref, tournamentID, placement, playerPrefix),
			GuID:       guID,
			Rank:       rank,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
			IsVital:    false,
		}
		if err := db.AddPlayerGu(tx, authID.String(), newGu); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Không thể nhận Cổ Trùng thưởng (%s): %v", guID, err),
			})
			return
		}
	}

	if InventorySvc != nil && len(itemsRaw) > 0 {
		var items []struct {
			ItemID   string `json:"item_id"`
			Category string `json:"category"`
			Quantity int64  `json:"quantity"`
		}
		if json.Unmarshal(itemsRaw, &items) == nil {
			for _, it := range items {
				if it.ItemID == "" {
					continue
				}
				if it.Quantity <= 0 {
					it.Quantity = 1
				}
				if err := InventorySvc.AddItem(tx, authID.String(), it.ItemID, it.Category, it.Quantity); err != nil {
					logger.S.Errorf("%s: failed to grant item %s to %s: %v", spec.AuditLabel, it.ItemID, authID, err)
					httperr.Respond(c, httperr.Internal("Không thể trao vật phẩm thưởng", err))
					return
				}
			}
		}
	}

	if _, err = tx.Exec(fmt.Sprintf(`UPDATE %s SET claimed = true, claimed_at = NOW() WHERE id = $1`, spec.TableName), rewardUUID); err != nil {
		httperr.Respond(c, httperr.Internal("Failed to mark as claimed", err))
		return
	}

	if err := auditlog.LogCheckedContext(c.Request.Context(), tx, auditlog.Entry{
		PlayerID:       &authID,
		Action:         spec.AuditAction,
		EntityType:     spec.AuditEntity,
		EntityID:       rewardUUID.String(),
		RequestSummary: fmt.Sprintf("Nhận thưởng %s hạng %d: %d Nguyên Thạch, %d Tiên Thạch, danh hiệu '%s'", spec.AuditLabel, placement, stones, immortalStones, title),
		Metadata: map[string]interface{}{
			"reward_id":       rewardUUID.String(),
			"placement":       placement,
			"stones":          stones,
			"immortal_stones": immortalStones,
			"title":           title,
			"reward_gu_id":    guID,
			"reward_gu_rank":  guRank,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	}); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể ghi audit", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Database commit failed", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Nhận phần thưởng thành công!",
		"placement":       placement,
		"title":           title,
		"stones":          stones,
		"immortal_stones": immortalStones,
		"reward_gu_id":    guID,
		"reward_gu_rank":  guRank,
	})
}
