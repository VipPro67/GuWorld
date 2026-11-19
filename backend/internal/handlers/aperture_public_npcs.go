package handlers

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

const publicApertureNPCCellPrefix = "public_aperture_npc:"

var errPublicApertureNPCBusy = errors.New("public aperture NPC is already in combat")

func publicApertureNPCAt(tx *sql.Tx, apertureID string, x, y int) (string, error) {
	var actorID string
	err := tx.QueryRow(`
		SELECT id::text FROM non_player_actors
		WHERE aperture_id = $1 AND owner_player_id IS NULL
		  AND actor_kind = 'VARIANT_NPC' AND lifecycle_state = 'ACTIVE'
		  AND execution_state IN ('GUARDING','COMBAT')
		  AND world_x = $2 AND world_y = $3
	`, apertureID, x, y).Scan(&actorID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("finding public aperture NPC at (%d,%d): %w", x, y, err)
	}
	return actorID, nil
}

func publicApertureNPCCell(tx *sql.Tx, apertureID string, cell models.Cell) (models.Cell, error) {
	actorID, err := publicApertureNPCAt(tx, apertureID, cell.X, cell.Y)
	if err != nil || actorID == "" {
		return cell, err
	}
	return models.Cell{
		X: cell.X, Y: cell.Y, Type: "Monster",
		Content: publicApertureNPCCellPrefix + actorID,
	}, nil
}

func startPublicApertureNPCCombat(tx *sql.Tx, apertureID, actorID string) (models.CombatState, error) {
	var combat models.CombatState
	var path, aiProfile string
	err := tx.QueryRow(`
		SELECT name, template_id, rank, sub_rank, hp, max_hp, base_attack, base_defense, speed, path, ai_profile
		FROM non_player_actors
		WHERE id = $1 AND aperture_id = $2 AND owner_player_id IS NULL
		  AND actor_kind = 'VARIANT_NPC' AND lifecycle_state = 'ACTIVE'
		  AND execution_state = 'GUARDING'
		FOR UPDATE
	`, actorID, apertureID).Scan(
		&combat.EnemyName, &combat.EnemyTemplateID, &combat.EnemyRank, &combat.EnemySubRank,
		&combat.EnemyHP, &combat.EnemyMaxHP, &combat.EnemyDamage, &combat.EnemyDefense,
		&combat.EnemySpeed, &path, &aiProfile,
	)
	if errors.Is(err, sql.ErrNoRows) {
		var state string
		stateErr := tx.QueryRow(`SELECT execution_state FROM non_player_actors WHERE id = $1 AND aperture_id = $2`, actorID, apertureID).Scan(&state)
		if stateErr == nil && state == "COMBAT" {
			return models.CombatState{}, errPublicApertureNPCBusy
		}
		return models.CombatState{}, sql.ErrNoRows
	}
	if err != nil {
		return models.CombatState{}, fmt.Errorf("loading public aperture NPC: %w", err)
	}
	if _, err := tx.Exec(`
		UPDATE non_player_actors SET execution_state = 'COMBAT', updated_at = NOW(), version = version + 1
		WHERE id = $1 AND execution_state = 'GUARDING'
	`, actorID); err != nil {
		return models.CombatState{}, fmt.Errorf("reserving public aperture NPC: %w", err)
	}
	combat.EnemyType = "MONSTER"
	combat.CombatMode = "real"
	combat.AIProfile = aiProfile
	combat.PublicApertureNPCID = actorID
	if combat.EnemyHP <= 0 {
		combat.EnemyHP = combat.EnemyMaxHP
	}
	if combat.EnemyMaxHP <= 0 {
		return models.CombatState{}, errors.New("NPC hộ vệ có sinh lực không hợp lệ")
	}
	if combat.EnemyRank <= 0 {
		combat.EnemyRank = 1
	}
	if combat.EnemySubRank <= 0 {
		combat.EnemySubRank = 1
	}
	if path != "" {
		combat.EnemyPathMasteries = map[string]int{path: combat.EnemyRank * 100}
	}
	return combat, nil
}

func settlePublicApertureNPC(tx *sql.Tx, actorID string, defeated bool) error {
	if actorID == "" {
		return nil
	}
	if defeated {
		_, err := tx.Exec(`
			UPDATE non_player_actors
			SET hp = 0, lifecycle_state = 'DEAD', execution_state = 'DEAD',
			    world_x = NULL, world_y = NULL, updated_at = NOW(), version = version + 1
			WHERE id = $1 AND actor_kind = 'VARIANT_NPC' AND execution_state = 'COMBAT'
		`, actorID)
		return err
	}
	_, err := tx.Exec(`
		UPDATE non_player_actors
		SET execution_state = 'GUARDING', updated_at = NOW(), version = version + 1
		WHERE id = $1 AND actor_kind = 'VARIANT_NPC' AND execution_state = 'COMBAT'
	`, actorID)
	return err
}
