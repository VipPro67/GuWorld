package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

type tournamentSideResult struct {
	combatants []*simulator.CombatantAuto
	teamID     string
}

// simulateTournamentMatch resolves an isolated tournament match without
// moving or requiring either participant to be online. The seed is stable for
// the match so a retry after a process restart produces the same combat log.
func simulateTournamentMatch(tx *sql.Tx, tournamentID, matchID string, sideA, sideB tournamentSideResult) ([]simulator.CombatLog, string, error) {
	if len(sideA.combatants) == 0 || len(sideB.combatants) == 0 {
		return nil, "", fmt.Errorf("tournament match requires combatants on both sides")
	}
	seedBytes := sha256.Sum256([]byte(tournamentID + ":" + matchID))
	state := &simulator.CombatState{
		Seed:                 int64(binary.LittleEndian.Uint64(seedBytes[:8])),
		MaxGuActionsPerTurn:  maxCombatGuActionsPerTurn(),
		Allies:               sideA.combatants,
		Enemies:              sideB.combatants,
		CaptureTurnSnapshots: true,
	}
	logs := simulator.Simulate(state)
	winningSide := tournamentWinningSide(state.Allies, state.Enemies)
	return logs, winningSide, nil
}

func tournamentWinningSide(sideA, sideB []*simulator.CombatantAuto) string {
	countAlive := func(combatants []*simulator.CombatantAuto) int {
		alive := 0
		for _, combatant := range combatants {
			if combatant != nil && combatant.HP > 0 {
				alive++
			}
		}
		return alive
	}
	remaining := func(combatants []*simulator.CombatantAuto) float64 {
		total := 0.0
		for _, combatant := range combatants {
			if combatant == nil || combatant.HP <= 0 {
				continue
			}
			total += combatant.HP / math.Max(1, combatant.MaxHP)
		}
		return total
	}
	aliveA, aliveB := countAlive(sideA), countAlive(sideB)
	if aliveA != aliveB {
		if aliveA > aliveB {
			return "a"
		}
		return "b"
	}
	if remaining(sideA) >= remaining(sideB) {
		return "a"
	}
	return "b"
}

func buildTournamentCombatant(tx *sql.Tx, playerID string, startingHP, maxHP, startingMana, maxMana float64, gridX, gridY int) (*simulator.CombatantAuto, error) {
	id, err := uuid.Parse(playerID)
	if err != nil {
		return nil, fmt.Errorf("invalid tournament player id %q: %w", playerID, err)
	}
	player, err := evaluateManaTxEquippedReadonly(tx, id)
	if err != nil {
		return nil, fmt.Errorf("load tournament player %s: %w", playerID, err)
	}
	// Tournament vitals belong to the tournament snapshot. They must not be
	// replaced by the player's current live-world vitals when a later round is
	// simulated.
	player.HP = startingHP
	player.BaseHP = maxHP
	player.CurrentMana = startingMana
	player.MaxMana = maxMana
	combatant, err := buildSimCombatant(tx, &player, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("build tournament combatant %s: %w", playerID, err)
	}
	if maxHP > 0 {
		combatant.MaxHP = maxHP
	}
	combatant.HP = math.Max(0, math.Min(startingHP, combatant.MaxHP))
	combatant.MaxMana = math.Max(0, maxMana)
	combatant.Mana = math.Max(0, math.Min(startingMana, combatant.MaxMana))
	combatant.GridX = gridX
	combatant.GridY = gridY
	// Tournament resolution must not silently spend a player's live inventory;
	// these matches only persist tournament-side HP/mana snapshots and rewards.
	combatant.AutoHealHP = false
	combatant.AutoRestoreMana = false
	combatant.HealCharges = 0
	combatant.PrimevalStones = 0
	combatant.ImmortalStones = 0
	return combatant, nil
}

func buildDeathArenaSide(tx *sql.Tx, teamID string) (tournamentSideResult, error) {
	members, err := loadTeamMembersForCombat(tx, teamID)
	if err != nil {
		return tournamentSideResult{}, err
	}
	side := tournamentSideResult{teamID: teamID}
	for i, member := range members {
		combatant, err := buildTournamentCombatant(tx, member.PlayerID, member.CurrentHP, member.MaxHP, member.CurrentMana, member.MaxMana, 1+(i/3), 1+(i%3))
		if err != nil {
			return tournamentSideResult{}, err
		}
		side.combatants = append(side.combatants, combatant)
	}
	return side, nil
}

func buildMartialArenaSide(tx *sql.Tx, combatantID string) (tournamentSideResult, error) {
	member, err := loadMartialCombatant(tx, combatantID)
	if err != nil {
		return tournamentSideResult{}, err
	}
	maxHP := member.MaxHP
	if maxHP <= 0 {
		maxHP = member.BaseHP
	}
	// Martial Arena is an individual virtual duel. Its realtime incarnation
	// started each fighter at full HP and full mana, independent of world HP.
	combatant, err := buildTournamentCombatant(tx, member.PlayerID, maxHP, maxHP, member.MaxMana, member.MaxMana, 1, 2)
	if err != nil {
		return tournamentSideResult{}, err
	}
	return tournamentSideResult{teamID: combatantID, combatants: []*simulator.CombatantAuto{combatant}}, nil
}

func saveTournamentCombatLog(logs []simulator.CombatLog) ([]byte, error) {
	raw, err := json.Marshal(logs)
	if err != nil {
		return nil, fmt.Errorf("encode tournament combat history: %w", err)
	}
	return raw, nil
}

// resolveDeathArenaTurnBasedMatch simulates and commits match outcome,
// survivor snapshots, rewards, and combat history atomically. The match row
// lock makes a worker retry harmless after another worker has already settled.
func resolveDeathArenaTurnBasedMatch(tournamentID, matchID, teamAID, teamBID string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRow(`SELECT status FROM death_arena_matches WHERE id=$1 FOR UPDATE`, matchID).Scan(&status); err != nil {
		return err
	}
	if status != "IN_PROGRESS" {
		return nil
	}
	sideA, err := buildDeathArenaSide(tx, teamAID)
	if err != nil {
		return err
	}
	sideB, err := buildDeathArenaSide(tx, teamBID)
	if err != nil {
		return err
	}
	logs, winnerSide, err := simulateTournamentMatch(tx, tournamentID, matchID, sideA, sideB)
	if err != nil {
		return err
	}
	winnerID, loserID, winner, loser := sideA.teamID, sideB.teamID, sideA, sideB
	if winnerSide == "b" {
		winnerID, loserID, winner, loser = sideB.teamID, sideA.teamID, sideB, sideA
	}
	for _, side := range []struct {
		result   tournamentSideResult
		isWinner bool
	}{{winner, true}, {loser, false}} {
		for _, combatant := range side.result.combatants {
			alive := side.isWinner && combatant.HP > 0
			eliminated := !alive
			hp := combatant.HP
			if eliminated {
				hp = 0
			}
			query := `UPDATE death_arena_team_members SET current_hp=$1,current_mana=$2,eliminated=$3,
				eliminated_at_round=CASE WHEN $3 THEN (SELECT current_round FROM death_arena_tournaments WHERE id=$4) ELSE NULL END
				WHERE team_id=$5 AND player_id=$6`
			if _, err := tx.Exec(query, hp, combatant.Mana, eliminated, tournamentID, side.result.teamID, combatant.ID); err != nil {
				return fmt.Errorf("persist death arena fighter %s: %w", combatant.ID, err)
			}
		}
	}
	logJSON, err := saveTournamentCombatLog(logs)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE death_arena_matches SET status='FINISHED',winner_team_id=$1,combat_log=$2,finished_at=NOW() WHERE id=$3 AND status='IN_PROGRESS'`, winnerID, logJSON, matchID); err != nil {
		return fmt.Errorf("finish death arena match: %w", err)
	}
	if _, err := tx.Exec(`UPDATE death_arena_teams SET eliminated=true,eliminated_at_round=(SELECT current_round FROM death_arena_tournaments WHERE id=$1) WHERE id=$2`, tournamentID, loserID); err != nil {
		return fmt.Errorf("eliminate death arena team: %w", err)
	}
	var reward int
	if err := tx.QueryRow(`SELECT reward_stones_per_member FROM death_arena_matches WHERE id=$1`, matchID).Scan(&reward); err != nil {
		return fmt.Errorf("load death arena round reward: %w", err)
	}
	if reward > 0 {
		if _, err := tx.Exec(`UPDATE players SET primeval_stones=primeval_stones+$1 WHERE id IN (SELECT player_id FROM death_arena_team_members WHERE team_id=$2 AND eliminated=false)`, reward, winnerID); err != nil {
			return fmt.Errorf("award death arena round reward: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	invalidateTournamentCache(tournamentID)
	BroadcastDeathArenaEvent("match_updated", tournamentID, map[string]interface{}{"match_id": matchID, "winner_team_id": winnerID, "status": "FINISHED"})
	return nil
}

func resolveMartialArenaTurnBasedMatch(tournamentID, matchID, combatantAID, combatantBID string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err := tx.QueryRow(`SELECT status FROM martial_matches WHERE id=$1 FOR UPDATE`, matchID).Scan(&status); err != nil {
		return err
	}
	if status != "IN_PROGRESS" {
		return nil
	}
	sideA, err := buildMartialArenaSide(tx, combatantAID)
	if err != nil {
		return err
	}
	sideB, err := buildMartialArenaSide(tx, combatantBID)
	if err != nil {
		return err
	}
	logs, winnerSide, err := simulateTournamentMatch(tx, tournamentID, matchID, sideA, sideB)
	if err != nil {
		return err
	}
	winnerID, loserID := combatantAID, combatantBID
	if winnerSide == "b" {
		winnerID, loserID = combatantBID, combatantAID
	}
	logJSON, err := saveTournamentCombatLog(logs)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE martial_combatants SET eliminated=true,eliminated_at_round=(SELECT current_round FROM martial_tournaments WHERE id=$1) WHERE id=$2`, tournamentID, loserID); err != nil {
		return fmt.Errorf("eliminate martial arena combatant: %w", err)
	}
	if _, err := tx.Exec(`UPDATE martial_matches SET status='FINISHED',winner_combatant_id=$1,combat_log=$2,finished_at=NOW() WHERE id=$3 AND status='IN_PROGRESS'`, winnerID, logJSON, matchID); err != nil {
		return fmt.Errorf("finish martial arena match: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	invalidateMartialCache(tournamentID)
	BroadcastMartialArenaEvent("match_updated", tournamentID, map[string]interface{}{"match_id": matchID, "winner_combatant_id": winnerID, "status": "FINISHED"})
	return nil
}
