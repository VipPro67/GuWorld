package handlers

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func loadTeamMembersForCombat(q db.Queryer, teamID string) ([]*teamMemberCombatant, error) {
	rows, err := q.Query(`
		SELECT dtm.player_id, dtm.current_hp, dtm.max_hp, dtm.current_mana, dtm.max_mana,
		       p.username, p.rank, p.sub_rank, p.hp, p.base_hp, p.current_mana, p.max_mana,
		       COALESCE(p.race, ''), COALESCE(p.constitution, ''),
		       `+db.GuInventoryAggregateEquipped("p.id")+`,
		       COALESCE(ia.dao_marks, '{}'::jsonb),
		       COALESCE(p.path_masteries, '{}'::jsonb),
		       COALESCE(p.combat_settings, '{}'::jsonb)
		FROM death_arena_team_members dtm
		JOIN players p ON dtm.player_id = p.id
		LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
		WHERE dtm.team_id = $1 AND dtm.eliminated = false AND dtm.current_hp > 0
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
			&m.GuInventory, &m.DaoMarks, &m.PathMasteries, &m.CombatSettings,
		); errScan != nil {
			logger.S.Errorf("Death Arena: Failed to scan team member %s: %v", m.PlayerID, errScan)
			return nil, errScan
		}
		members = append(members, &m)
	}

	return members, nil
}

type teamMemberCombatant struct {
	PlayerID       string
	CurrentHP      float64
	MaxHP          float64
	CurrentMana    float64
	MaxMana        float64
	Username       string
	Rank           int
	SubRank        int
	PlayerHP       float64
	BaseHP         float64
	PlayerMana     float64
	MaxManaPlayer  float64
	Race           string
	Constitution   string
	GuInventory    models.GuInventory
	DaoMarks       models.DaoMarksMap
	PathMasteries  models.PathMasteriesMap
	CombatSettings models.AutoCombatConfig
}

// buildTeamCombatant builds a combatant from team member data
func buildTeamCombatant(member *teamMemberCombatant) (*simulator.CombatantAuto, error) {
	// Calculate stats
	player := &models.Player{}
	player.ID = uuid.MustParse(member.PlayerID)
	player.Username = member.Username
	player.Rank = member.Rank
	player.SubRank = member.SubRank
	player.HP = member.CurrentHP
	player.BaseHP = member.BaseHP
	player.CurrentMana = member.CurrentMana
	player.MaxMana = member.MaxMana
	player.Race = member.Race
	player.Constitution = member.Constitution
	player.GuInventory = member.GuInventory
	player.DaoMarks = member.DaoMarks
	player.PathMasteries = member.PathMasteries
	player.CombatSettings = member.CombatSettings

	playerStats := CalculatePlayerStats(player)
	actor := models.CombatantFromPlayer(player, playerStats)

	// Build actions from equipped Gus
	var allActions []simulator.SimAction
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" && gu.Hunger > 0 {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				if hasCombatActiveEffects(&spec) {
					act, err := buildSimActionFromGu(&actor, &spec, "GU_SKILL")
					if err != nil {
						continue
					}
					act.SourceID = gu.GuID
					allActions = append(allActions, act)
				}
			}
		}
	}

	// Basic attack fallback
	allActions = append(allActions, simulator.SimAction{
		ActionType: "BASIC_ATTACK",
		Name:       "Đánh thường",
		Damage:     playerStats["attack"],
		ManaCost:   0,
	})

	// Build DaoMarks
	daoMarks := make(simulator.DaoMarks)
	for k, v := range member.DaoMarks {
		daoMarks[k] = v
	}

	// Calculate max HP
	baseHP := member.BaseHP + playerStats["hp"]
	maxHP := baseHP
	if maxHP <= 0 {
		maxHP = 100
	}

	// Use current HP from tournament state
	simHP := member.CurrentHP
	if simHP > maxHP {
		simHP = maxHP
	}
	if simHP <= 0 {
		simHP = maxHP
	}

	simMana := member.CurrentMana
	if simMana > member.MaxMana {
		simMana = member.MaxMana
	}

	charFX := gamefx.Get(player)

	simCombatant := &simulator.CombatantAuto{
		ID:                   member.PlayerID,
		Name:                 member.Username,
		Kind:                 "Player",
		Rank:                 member.Rank,
		SubRank:              member.SubRank,
		MaxHP:                maxHP,
		HP:                   simHP,
		MaxMana:              member.MaxMana,
		Mana:                 simMana,
		BaseDmg:              playerStats["attack"],
		BaseDef:              playerStats["defense"],
		Speed:                playerStats["speed"],
		DaoMarks:             daoMarks,
		PathMasteries:        member.PathMasteries,
		PathMasteryGainLimit: gamefx.PathMasteryBattleGainLimit(member.Rank),
		DaoBonus:             charFX.DaoNganBonus,
		PathDamageBonus:      charFX.PathDamageBonus,
		PathResistance:       charFX.PathResistanceBonus,
		HealingReceived:      charFX.HealingReceivedBonus,
		MentalResistance:     charFX.MentalResistanceBonus,
		Actions:              allActions,
		AutoHealHP:           false,
		HealThreshold:        0.3,
		AutoRestoreMana:      true,
		ManaThreshold:        0.2,
		HealCharges:          0,
		HealPotency:          0,
		PrimevalStones:       999999,
		ImmortalStones:       999999,
		RefineManaTier:       gamefx.RefineManaTier(member.Rank, member.GuInventory),
		Loadout: &simulator.CombatLoadout{
			GuIds: func() []string {
				var ids []string
				for _, gu := range player.GuInventory {
					if gu.Equipped && gu.Status == "ACTIVE" && gu.Hunger > 0 {
						ids = append(ids, gu.GuID)
					}
				}
				return ids
			}(),
		},
	}

	return simCombatant, nil
}

// awardChampionRewards awards rewards to top 3 teams
func awardChampionRewards(tx *sql.Tx, tournamentID, championTeamID string, placement int) error {
	// Get all teams ordered by placement
	teams := getTeamPlacements(tx, tournamentID)

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

	// Gu rewards for top 3 (guID, rank)
	type guReward struct {
		guID string
		rank int
	}
	placementGuRewards := map[int]guReward{
		1: {guID: "GU_LIFESPAN_500", rank: 5},
		2: {guID: "GU_LIFESPAN_100", rank: 5},
		3: {guID: "GU_LIFESPAN_50", rank: 4},
	}

	for i, teamID := range teams {
		if i >= 3 {
			break
		}
		place := i + 1
		title := placementTitles[place]
		stones := placementStones[place]
		immortalStones := placementImmortalStones[place]
		gu := placementGuRewards[place]

		// Get team members
		members := getTeamMembers(tx, teamID)
		for _, memberID := range members {
			_, err := tx.Exec(`
				INSERT INTO death_arena_champion_rewards 
				(tournament_id, player_id, placement, team_id, title, reward_stones, reward_immortal_stones, reward_gu_id, reward_gu_rank)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`, tournamentID, memberID, place, teamID, title, stones, immortalStones, gu.guID, gu.rank)
			if err != nil {
				logger.S.Errorf("Failed to insert champion reward: %v", err)
			}

			// Award Gu directly to player
			newGu := models.GuInstance{
				InstanceID: fmt.Sprintf("death_arena_%s_%d_%s", tournamentID, place, memberID[:8]),
				GuID:       gu.guID,
				Rank:       gu.rank,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   false,
				IsVital:    false,
			}
			if err := db.AddPlayerGu(tx, memberID, newGu); err != nil {
				logger.S.Errorf("Failed to award Gu %s rank %d to player %s: %v", gu.guID, gu.rank, memberID, err)
			}
		}

		// Update team final placement
		_, _ = tx.Exec(`UPDATE death_arena_teams SET final_placement = $1 WHERE id = $2`, place, teamID)
	}

	return nil
}

// getTeamPlacements returns team IDs ordered by placement
func getTeamPlacements(tx *sql.Tx, tournamentID string) []string {
	// First get champion
	var championID string
	err := tx.QueryRow(`
		SELECT champion_team_id FROM death_arena_tournaments WHERE id = $1
	`, tournamentID).Scan(&championID)
	if err != nil || championID == "" {
		return nil
	}

	result := []string{championID}

	// Get runner-up from final match (the loser of the final match)
	var runnerUp *string
	err = tx.QueryRow(`
		SELECT CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END
		FROM death_arena_matches
		WHERE tournament_id = $1 AND round_number = (
			SELECT total_rounds FROM death_arena_tournaments WHERE id = $1
		) AND status = 'FINISHED'
		LIMIT 1
	`, tournamentID).Scan(&runnerUp)
	if err == nil && runnerUp != nil && *runnerUp != "" {
		result = append(result, *runnerUp)
	}

	// Get 3rd place from semi-final losers
	runnerUpID := ""
	if runnerUp != nil {
		runnerUpID = *runnerUp
	}
	rows, err := tx.Query(`
		SELECT DISTINCT 
			CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END AS loser_id
		FROM death_arena_matches
		WHERE tournament_id = $1 
		AND round_number = (SELECT total_rounds - 1 FROM death_arena_tournaments WHERE id = $1)
		AND status = 'FINISHED'
		AND (CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END) IS NOT NULL
		AND (CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END) != $2
		AND (CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END) != $3
		LIMIT 1
	`, tournamentID, championID, runnerUpID)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			var third string
			if errScan := rows.Scan(&third); errScan == nil && third != "" {
				result = append(result, third)
			}
		}
	}

	return result
}

// getTeamMembers returns player IDs for a team
func getTeamMembers(tx *sql.Tx, teamID string) []string {
	rows, err := tx.Query(`SELECT player_id FROM death_arena_team_members WHERE team_id = $1`, teamID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var id string
		if errScan := rows.Scan(&id); errScan == nil {
			members = append(members, id)
		}
	}
	return members
}
