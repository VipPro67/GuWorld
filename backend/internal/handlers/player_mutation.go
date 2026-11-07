package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
)

// MutationError wraps domain business errors with an explicit HTTP status code.
type MutationError struct {
	StatusCode int
	Message    string
}

func (e MutationError) Error() string {
	return e.Message
}

// NewMutationError creates a typed domain error for MutatePlayer.
func NewMutationError(code int, msg string) error {
	return MutationError{StatusCode: code, Message: msg}
}

// HandleMutationError inspects an error returned by MutatePlayer and sends the proper JSON response.
func HandleMutationError(c *gin.Context, err error) {
	if mErr, ok := err.(MutationError); ok {
		c.JSON(mErr.StatusCode, gin.H{"error": mErr.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// PlayerMutationOptions configures the player mutation pipeline.
type PlayerMutationOptions struct {
	// Scope selects which Gu inventory records to hydrate with row locking.
	// Defaults to playerstate.GuFull for complete inventory safety.
	Scope playerstate.GuScope
	// EventDomains specifies which domains changed (e.g. PlayerDomain, GuInventoryDomain, CombatStatsDomain).
	// Always includes PlayerDomain by default.
	EventDomains []PlayerEventDomain
	// SkipManaEvaluation skips passive mana regeneration projection before mutation.
	SkipManaEvaluation bool
	// SkipDurabilityHealing skips passive Gu durability healing evaluation.
	SkipDurabilityHealing bool
	// SkipSnapshotPublish skips post-commit player snapshot publication (rare).
	SkipSnapshotPublish bool
	// SkipPlayerRowUpdate skips updating the players table columns if only child collections were modified.
	SkipPlayerRowUpdate bool
	// IncludeCollections is kept for backwards compatibility; full canonical update is default.
	IncludeCollections bool
}

func marshalJSONOrDefault(v any, defaultJSON string) []byte {
	if v == nil {
		return []byte(defaultJSON)
	}
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return []byte(defaultJSON)
	}
	return b
}

// MutatePlayer executes fn within a dedicated database transaction.
// It guarantees:
// 1. Transaction opening & deferred rollback.
// 2. Row locking (FOR UPDATE) and state projection.
// 3. Passive evaluation (mana and Gu durability healing).
// 4. Execution of the caller's pure domain mutation closure.
// 5. Full canonical persistence of all modified player fields, with version & revision monotonic increment.
// 6. Child domain updates (immortal apertures dao marks).
// 7. Atomic transaction commit.
// 8. Post-commit Redis/SSE event publication and pending queue clearing.
func MutatePlayer(playerID uuid.UUID, opts PlayerMutationOptions, fn func(tx *sql.Tx, player *models.Player) error) (*models.Player, error) {
	if db.DB == nil {
		return nil, fmt.Errorf("database connection unavailable")
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Passive durability healing
	if !opts.SkipDurabilityHealing {
		_ = EvaluateGuDurabilityHealing(tx, playerID.String())
	}

	scope := opts.Scope
	if scope == playerstate.GuNone {
		scope = playerstate.GuFull
	}

	// 2. Load player with FOR UPDATE lock and mana evaluation
	var player models.Player
	if !opts.SkipManaEvaluation {
		if scope == playerstate.GuFull {
			player, err = evaluateManaTx(tx, playerID)
		} else {
			player, err = evaluateManaTxEquipped(tx, playerID)
		}
	} else {
		if scope == playerstate.GuFull {
			player, err = PlayerSvc.LoadForUpdate(tx, playerID)
		} else {
			player, err = PlayerSvc.LoadForUpdateEquipped(tx, playerID)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("load player for mutation: %w", err)
	}

	// 3. Execute domain mutation closure
	if err := fn(tx, &player); err != nil {
		return nil, err
	}

	// 4. Persist updated player row
	if !opts.SkipPlayerRowUpdate {
		equippedGusJSON := marshalJSONOrDefault(player.EquippedGus, "[]")
		statusesJSON := marshalJSONOrDefault(player.Statuses, "[]")
		bodyModsJSON := marshalJSONOrDefault(player.BodyMods, "[]")
		hotbarJSON := marshalJSONOrDefault(player.HotbarGus, "[]")
		clearedCellsJSON := marshalJSONOrDefault(player.ClearedCells, "{}")
		visitedSubzonesJSON := marshalJSONOrDefault(player.VisitedSubZones, "[]")
		pathMasteriesJSON := marshalJSONOrDefault(player.PathMasteries, "{}")
		growthDataJSON := marshalJSONOrDefault(player.GrowthData, `{"boosts":{"attack":0,"defense":0,"hp":0,"speed":0},"fully_grown":[]}`)
		immortalStateJSON := marshalJSONOrDefault(player.ImmortalState, "{}")
		combatSettingsJSON := marshalJSONOrDefault(player.CombatSettings, "{}")

		_, err = tx.Exec(`
			UPDATE players
			SET rank = $1, sub_rank = $2, wall_progress = $3, human_qi = $4,
			    current_mana = $5, max_mana = $6, last_mana_update = $7,
			    primeval_stones = $8, immortal_stones = $9,
			    hp = $10, base_hp = $11, soul_hp = $12, max_soul = $13, soul_tier = $14, needs_soul_refine = $15,
			    current_thought = $16, max_thought = $17,
			    lifespan = $18, aperture_stress = $19, last_cultivate_time = $20, time_speed_multiplier = $21,
			    aptitude_type = $22, aptitude_percent = $23, aptitude_grade = $24,
			    equipped_gus = $25, statuses = $26, body_mods = $27, realtime_hotbar = $28,
			    cleared_cells = $29, visited_subzones = $30, path_masteries = $31,
			    growth_data = $32, immortal_state = $33, combat_settings = $34,
			    explore_count = $35, last_explore_reset = $36,
			    bio = $37, fake_rank = $38, fake_sub_rank = $39, fake_vital_gu = $40, is_disguised = $41, avatar_url = $42,
			    version = version + 1,
			    state_revision = COALESCE(state_revision, 0) + 1
			WHERE id = $43
		`,
			player.Rank, player.SubRank, player.WallProgress, player.HumanQi,
			player.CurrentMana, player.MaxMana, player.LastManaUpdate,
			player.PrimevalStones, player.ImmortalStones,
			player.HP, player.BaseHP, player.SoulHP, player.MaxSoul, player.SoulTier, player.NeedsSoulRefine,
			player.CurrentThought, player.MaxThought,
			player.Lifespan, player.ApertureStress, player.LastCultivateTime, player.TimeSpeedMultiplier,
			player.AptitudeType, player.AptitudePercent, player.AptitudeGrade,
			equippedGusJSON, statusesJSON, bodyModsJSON, hotbarJSON,
			clearedCellsJSON, visitedSubzonesJSON, pathMasteriesJSON,
			growthDataJSON, immortalStateJSON, combatSettingsJSON,
			player.ExploreCount, player.LastExploreReset,
			player.Bio, player.FakeRank, player.FakeSubRank, player.FakeVitalGu, player.IsDisguised, player.AvatarURL,
			player.ID,
		)
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

	// 5. Update Dao Marks for Immortals (Rank >= 6) if present
	if player.Rank >= 6 && player.DaoMarks != nil {
		daoMarksJSON := marshalJSONOrDefault(player.DaoMarks, "{}")
		_, _ = tx.Exec(`
			UPDATE immortal_apertures
			SET dao_marks = $1
			WHERE player_id = $2
		`, daoMarksJSON, player.ID)
	}

	// 6. Atomic transaction commit
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit player mutation: %w", err)
	}

	// 7. Post-commit event publications & queue clearing
	ClearPendingPlayerQueues(player.ID.String())
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
