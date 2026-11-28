package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// loadMartialCombatant loads one combatant's player data for its match.
func loadMartialCombatant(q db.Queryer, combatantID string) (*teamMemberCombatant, error) {
	var m teamMemberCombatant
	err := q.QueryRow(`
		SELECT mc.player_id, p.hp, p.base_hp, p.current_mana, p.max_mana,
		       p.username, p.rank, p.sub_rank, p.hp, p.base_hp, p.current_mana, p.max_mana,
		       COALESCE(p.race, ''), COALESCE(p.constitution, ''),
		       `+db.GuInventoryAggregateEquipped("p.id")+`,
		       COALESCE(ia.dao_marks, '{}'::jsonb),
		       COALESCE(p.path_masteries, '{}'::jsonb)
		FROM martial_combatants mc
		JOIN players p ON mc.player_id = p.id
		LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
		WHERE mc.id = $1
	`, combatantID).Scan(
		&m.PlayerID, &m.CurrentHP, &m.MaxHP, &m.CurrentMana, &m.MaxMana,
		&m.Username, &m.Rank, &m.SubRank, &m.PlayerHP, &m.BaseHP, &m.PlayerMana, &m.MaxManaPlayer,
		&m.Race, &m.Constitution, &m.GuInventory, &m.DaoMarks, &m.PathMasteries,
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

// awardMartialChampionRewards writes claimable rewards for the top 3 placements.
// Gu rewards are granted at claim time so inventory capacity never blocks settlement.
func awardMartialChampionRewards(tx *sql.Tx, tournamentID string, finalRound int) ([]string, error) {
	placements, err := getMartialPlacements(tx, tournamentID, finalRound)
	if err != nil {
		return nil, err
	}
	var rewardedPlayers []string
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
			return nil, fmt.Errorf("insert martial arena placement %d reward for player %s: %w", place, playerID, err)
		}
		rewardedPlayers = append(rewardedPlayers, playerID)

		if _, err := tx.Exec(`UPDATE martial_combatants SET final_placement = $1 WHERE id = $2`, place, combatantID); err != nil {
			return nil, fmt.Errorf("save martial arena placement %d for combatant %s: %w", place, combatantID, err)
		}
	}
	return rewardedPlayers, nil
}

// getMartialPlacements returns combatant IDs in placement order (1st, 2nd, 3rd).
func getMartialPlacements(tx *sql.Tx, tournamentID string, finalRound int) ([]string, error) {
	return resolveTournamentPlacements(tx, placementQueryConfig{
		TournamentTable:  "martial_tournaments",
		ChampionColumn:   "champion_player_id",
		MatchTable:       "martial_matches",
		SideAColumn:      "combatant_a_id",
		SideBColumn:      "combatant_b_id",
		WinnerSideColumn: "winner_combatant_id",
	}, tournamentID, "", finalRound)
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
		httperr.Respond(c, httperr.Internal("Failed to fetch rewards", err))
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

// ClaimMartialReward claims a placement reward (stones, immortal stones, title, Gu, items).
func ClaimMartialReward(c *gin.Context) {
	claimTournamentReward(c, martialRewardClaimSpec)
}
