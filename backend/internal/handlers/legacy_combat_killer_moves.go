package handlers

import (
	"database/sql"
	"encoding/json"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func loadCombatKillerMoves(tx *sql.Tx, player *models.Player, loadout *simulator.CombatLoadout) ([]storedKillerMove, error) {
	if tx == nil || player == nil {
		return nil, nil
	}

	var targetIDs []string
	if loadout != nil && loadout.KillerMoveIds != nil {
		if len(loadout.KillerMoveIds) == 0 {
			return nil, nil
		}
		targetIDs = loadout.KillerMoveIds
	} else {
		kmTactics := player.CombatSettings.KillerMovePriorities
		if len(kmTactics) == 0 {
			kmTactics = player.CombatSettings.KmPriorities
		}
		if len(kmTactics) == 0 {
			return nil, nil
		}
		for _, t := range kmTactics {
			if t.GuID != "" {
				targetIDs = append(targetIDs, t.GuID)
			}
		}
		if len(targetIDs) == 0 {
			return nil, nil
		}
	}

	rows, err := tx.Query(`SELECT id::text,name,main_gu_instance_id,main_gu_id,main_gu_rank,components,composition_graph,selected_effects,compiled_effects,path,creation_path_mastery_level,power_budget,power_load,harmony,complexity,success_chance,mastery_xp,COALESCE(combat_profile,'{}'::jsonb) FROM player_killer_moves WHERE player_id=$1 AND id = ANY($2)`, player.ID, targetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	owned := make(map[string]models.GuInstance, len(player.GuInventory))
	for _, g := range player.GuInventory {
		owned[g.InstanceID] = g
	}
	byID := make(map[string]storedKillerMove)
	for rows.Next() {
		var m storedKillerMove
		var comps, graph, selected, compiled, combatProf []byte
		if err := rows.Scan(&m.ID, &m.Name, &m.MainGuInstance, &m.MainGuID, &m.MainGuRank, &comps, &graph, &selected, &compiled, &m.Path, &m.CreationPathMasteryLevel, &m.PowerBudget, &m.PowerLoad, &m.Harmony, &m.Complexity, &m.SuccessChance, &m.MasteryXP, &combatProf); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(comps, &m.Components)
		_ = json.Unmarshal(graph, &m.CompositionGraph)
		_ = json.Unmarshal(selected, &m.SelectedEffects)
		blocks, decodeErr := gameplayeffects.DecodeBlockSet(compiled)
		if decodeErr != nil {
			continue
		}
		m.CompiledEffects = []gameplayeffects.Block(blocks)
		_ = json.Unmarshal(combatProf, &m.CombatProfile)
		if m.CombatProfile.CooldownTurns == 0 {
			m.CombatProfile.CooldownTurns = 1 + (len(m.Components)-1)/2
			m.CombatProfile.TargetType = "SINGLE"
		}
		valid := true
		for _, component := range m.Components {
			g, ok := owned[component.InstanceID]
			if !ok || !killerMoveComponentMatches(component, g) || g.Hunger <= 0 {
				valid = false
				break
			}
		}
		if valid && validateKillerMoveCompositionSnapshotMode(m.CompositionGraph, m.SelectedEffects, m.Components, m.CompiledEffects, false) != nil {
			valid = false
		}
		hasCombatEffect := false
		for _, block := range m.CompiledEffects {
			if block.Trigger == gameplayeffects.TriggerOnAction && len(block.Operations) > 0 {
				hasCombatEffect = true
				break
			}
		}
		if valid && hasCombatEffect {
			byID[m.ID] = m
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []storedKillerMove
	seen := make(map[string]bool, len(targetIDs))
	for _, id := range targetIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		if m, ok := byID[id]; ok {
			result = append(result, m)
		}
	}
	return result, nil
}

// Load equipped moves for combat construction.
func loadEquippedKillerMoves(tx *sql.Tx, player *models.Player) ([]storedKillerMove, error) {
	return loadCombatKillerMoves(tx, player, nil)
}

// ComponentInstanceIDs exposes the Gu IDs whose fullness is consumed by a
// killer move action.
func (m storedKillerMove) ComponentInstanceIDs() []string {
	result := make([]string, 0, len(m.Components))
	for _, c := range m.Components {
		result = append(result, c.InstanceID)
	}
	return result
}
