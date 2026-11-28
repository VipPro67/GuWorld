package handlers

import (
	"database/sql"
	"fmt"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func loadTeamMembersForCombat(q db.Queryer, teamID string) ([]*teamMemberCombatant, error) {
	rows, err := q.Query(`
		SELECT dtm.player_id, dtm.current_hp, dtm.max_hp, dtm.current_mana, dtm.max_mana,
		       p.username, p.rank, p.sub_rank, p.hp, p.base_hp, p.current_mana, p.max_mana,
		       COALESCE(p.race, ''), COALESCE(p.constitution, ''),
		       `+db.GuInventoryAggregateEquipped("p.id")+`,
		       COALESCE(ia.dao_marks, '{}'::jsonb),
		       COALESCE(p.path_masteries, '{}'::jsonb)
		FROM death_arena_team_members dtm
		JOIN players p ON dtm.player_id = p.id
		LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
		WHERE dtm.team_id = $1 AND dtm.eliminated = false AND dtm.current_hp > 0
		ORDER BY dtm.player_id
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*teamMemberCombatant
	for rows.Next() {
		var m teamMemberCombatant
		if errScan := rows.Scan(
			&m.PlayerID, &m.CurrentHP, &m.MaxHP, &m.CurrentMana, &m.MaxMana,
			&m.Username, &m.Rank, &m.SubRank, &m.PlayerHP, &m.BaseHP, &m.PlayerMana, &m.MaxManaPlayer,
			&m.Race, &m.Constitution,
			&m.GuInventory, &m.DaoMarks, &m.PathMasteries,
		); errScan != nil {
			logger.S.Errorf("Death Arena: Failed to scan team member %s: %v", m.PlayerID, errScan)
			return nil, errScan
		}
		members = append(members, &m)
	}

	return members, nil
}

type teamMemberCombatant struct {
	PlayerID      string
	CurrentHP     float64
	MaxHP         float64
	CurrentMana   float64
	MaxMana       float64
	Username      string
	Rank          int
	SubRank       int
	PlayerHP      float64
	BaseHP        float64
	PlayerMana    float64
	MaxManaPlayer float64
	Race          string
	Constitution  string
	GuInventory   models.GuInventory
	DaoMarks      models.DaoMarksMap
	PathMasteries models.PathMasteriesMap
}

// awardChampionRewards writes claimable rewards for the top 3 teams.
// Gu rewards are recorded on the claim row and granted when the player claims
// so a full aperture never rolls back tournament settlement.
func awardChampionRewards(tx *sql.Tx, tournamentID, championTeamID string, finalRound int) ([]string, error) {
	teams, err := getTeamPlacements(tx, tournamentID, championTeamID, finalRound)
	if err != nil {
		return nil, err
	}

	placementTitles := map[int]string{
		1: "Quán Quân Đấu Trường Sinh Tử",
		2: "Á Quân Đấu Trường Sinh Tử",
		3: "Quý Quân Đấu Trường Sinh Tử",
	}

	placementStones := map[int]int{
		1: 5000,
		2: 3000,
		3: 1500,
	}

	placementImmortalStones := map[int]int{
		1: 500,
		2: 300,
		3: 150,
	}

	type guReward struct {
		guID string
		rank int
	}
	placementGuRewards := map[int]guReward{
		1: {guID: "GU_LIFESPAN_500", rank: 5},
		2: {guID: "GU_LIFESPAN_100", rank: 5},
		3: {guID: "GU_LIFESPAN_50", rank: 4},
	}

	var rewardedPlayers []string
	for i, teamID := range teams {
		if i >= 3 {
			break
		}
		place := i + 1
		title := placementTitles[place]
		stones := placementStones[place]
		immortalStones := placementImmortalStones[place]
		gu := placementGuRewards[place]

		members, err := getTeamMembers(tx, teamID)
		if err != nil {
			return nil, err
		}
		for _, memberID := range members {
			_, err := tx.Exec(`
				INSERT INTO death_arena_champion_rewards 
				(tournament_id, player_id, placement, team_id, title, reward_stones, reward_immortal_stones, reward_gu_id, reward_gu_rank)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`, tournamentID, memberID, place, teamID, title, stones, immortalStones, gu.guID, gu.rank)
			if err != nil {
				return nil, fmt.Errorf("insert death arena placement %d reward for player %s: %w", place, memberID, err)
			}
			rewardedPlayers = append(rewardedPlayers, memberID)
		}

		if _, err := tx.Exec(`UPDATE death_arena_teams SET final_placement = $1 WHERE id = $2`, place, teamID); err != nil {
			return nil, fmt.Errorf("save death arena placement %d for team %s: %w", place, teamID, err)
		}
	}

	return rewardedPlayers, nil
}

// getTeamPlacements returns team IDs ordered by placement.
func getTeamPlacements(tx *sql.Tx, tournamentID, championTeamID string, finalRound int) ([]string, error) {
	return resolveTournamentPlacements(tx, placementQueryConfig{
		TournamentTable:  "death_arena_tournaments",
		ChampionColumn:   "champion_team_id",
		MatchTable:       "death_arena_matches",
		SideAColumn:      "team_a_id",
		SideBColumn:      "team_b_id",
		WinnerSideColumn: "winner_team_id",
	}, tournamentID, championTeamID, finalRound)
}

// getTeamMembers returns player IDs for a team
func getTeamMembers(tx *sql.Tx, teamID string) ([]string, error) {
	rows, err := tx.Query(`SELECT player_id FROM death_arena_team_members WHERE team_id = $1`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var id string
		if errScan := rows.Scan(&id); errScan != nil {
			return nil, errScan
		}
		members = append(members, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return members, nil
}
