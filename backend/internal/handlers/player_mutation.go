package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
)

// PlayerMutationOptions configures the player mutation pipeline.
type PlayerMutationOptions struct {
	// Scope selects which Gu inventory records to hydrate with row locking.
	// Defaults to playerstate.GuEquipped for performance.
	Scope playerstate.GuScope
	// EventDomains specifies which domains changed (e.g. PlayerDomain, GuInventoryDomain).
	// Always includes PlayerDomain by default.
	EventDomains []PlayerEventDomain
	// SkipManaEvaluation skips passive mana regeneration projection before mutation.
	SkipManaEvaluation bool
	// SkipSnapshotPublish skips post-commit player snapshot publication (rare).
	SkipSnapshotPublish bool
	// SkipPlayerRowUpdate skips updating the players table columns if only child collections were modified.
	SkipPlayerRowUpdate bool
	// IncludeCollections updates JSONB collections (equipped_gus, statuses, body_mods) on the player row.
	IncludeCollections bool
}

// MutatePlayer executes fn within a dedicated database transaction.
// It guarantees:
// 1. Transaction opening & deferred rollback.
// 2. Row locking (FOR UPDATE) and state projection.
// 3. Execution of the caller's pure domain mutation closure.
// 4. Persistence of modified player fields, with version & revision monotonic increment.
// 5. Atomic transaction commit.
// 6. Post-commit Redis/SSE event publication to eliminate client/realtime state desync.
func MutatePlayer(playerID uuid.UUID, opts PlayerMutationOptions, fn func(tx *sql.Tx, player *models.Player) error) (*models.Player, error) {
	if db.DB == nil {
		return nil, fmt.Errorf("database connection unavailable")
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var player models.Player
	if !opts.SkipManaEvaluation {
		if opts.Scope == playerstate.GuFull {
			player, err = evaluateManaTx(tx, playerID)
		} else {
			player, err = evaluateManaTxEquipped(tx, playerID)
		}
	} else {
		if opts.Scope == playerstate.GuFull {
			player, err = PlayerSvc.LoadForUpdate(tx, playerID)
		} else {
			player, err = PlayerSvc.LoadForUpdateEquipped(tx, playerID)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("load player for mutation: %w", err)
	}

	// Execute domain mutation
	if err := fn(tx, &player); err != nil {
		return nil, err
	}

	// Persist updated player row
	if !opts.SkipPlayerRowUpdate {
		if opts.IncludeCollections {
			equippedGusJSON, _ := json.Marshal(player.EquippedGus)
			statusesJSON, _ := json.Marshal(player.Statuses)
			bodyModsJSON, _ := json.Marshal(player.BodyMods)
			_, err = tx.Exec(`
				UPDATE players
				SET current_mana = $1, max_mana = $2, last_mana_update = $3,
				    hp = $4, base_hp = $5, soul_hp = $6,
				    rank = $7, sub_rank = $8, wall_progress = $9, human_qi = $10,
				    primeval_stones = $11, immortal_stones = $12,
				    aperture_stress = $13, last_cultivate_time = $14, lifespan = $15,
				    time_speed_multiplier = $16, current_thought = $17,
				    equipped_gus = $18, statuses = $19, body_mods = $20,
				    version = version + 1,
				    state_revision = COALESCE(state_revision, 0) + 1
				WHERE id = $21
			`, player.CurrentMana, player.MaxMana, player.LastManaUpdate,
				player.HP, player.BaseHP, player.SoulHP,
				player.Rank, player.SubRank, player.WallProgress, player.HumanQi,
				player.PrimevalStones, player.ImmortalStones,
				player.ApertureStress, player.LastCultivateTime, player.Lifespan,
				player.TimeSpeedMultiplier, player.CurrentThought,
				equippedGusJSON, statusesJSON, bodyModsJSON,
				player.ID)
		} else {
			_, err = tx.Exec(`
				UPDATE players
				SET current_mana = $1, max_mana = $2, last_mana_update = $3,
				    hp = $4, base_hp = $5, soul_hp = $6,
				    rank = $7, sub_rank = $8, wall_progress = $9, human_qi = $10,
				    primeval_stones = $11, immortal_stones = $12,
				    aperture_stress = $13, last_cultivate_time = $14, lifespan = $15,
				    time_speed_multiplier = $16, current_thought = $17,
				    version = version + 1,
				    state_revision = COALESCE(state_revision, 0) + 1
				WHERE id = $18
			`, player.CurrentMana, player.MaxMana, player.LastManaUpdate,
				player.HP, player.BaseHP, player.SoulHP,
				player.Rank, player.SubRank, player.WallProgress, player.HumanQi,
				player.PrimevalStones, player.ImmortalStones,
				player.ApertureStress, player.LastCultivateTime, player.Lifespan,
				player.TimeSpeedMultiplier, player.CurrentThought,
				player.ID)
		}
		if err != nil {
			return nil, fmt.Errorf("persist player mutation: %w", err)
		}
		player.Version++
		player.StateRevision++
	} else {
		// When player row update is skipped (e.g. child collection only), still bump state revision for cache invalidation
		_, err = tx.Exec(`
			UPDATE players
			SET state_revision = COALESCE(state_revision, 0) + 1
			WHERE id = $1
		`, player.ID)
		if err != nil {
			return nil, fmt.Errorf("bump state revision: %w", err)
		}
		player.StateRevision++
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit player mutation: %w", err)
	}

	// Post-commit publication: ensures readers reload freshly committed data.
	if !opts.SkipSnapshotPublish {
		domains := opts.EventDomains
		if len(domains) == 0 {
			domains = []PlayerEventDomain{PlayerDomain}
		}
		PublishPlayerSnapshot(&player, PlayerEventUpdated, domains...)
		PublishPlayerManaDelta(player.ID, player.CurrentMana, player.MaxMana)
	}

	return &player, nil
}
