package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartHungerWorker begins the background process for depleting Gu bug hunger.
func StartHungerWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Hunger Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Hunger Worker stopped")
				return
			case <-ticker.C:
				processHungerDecay(db)
			}
		}
	}()
}

// processHungerDecay depletes Gu hunger for every player. The heavy lifting is
// done with set-based SQL over the normalized player_gu table (single pass,
// no per-player transactions). Only the rare ON_TICK effects (Thien Nguyen Bao
// Lien series, GU_CREATION_ORIGIN) are handled in Go, and only for the small
// subset of players that actually own an equipped ACTIVE Gu with such effects.
func processHungerDecay(db *sql.DB) {
	// 1. Find players who own an equipped ACTIVE Gu carrying ON_TICK effects.
	subset := findOnTickPlayers(db)

	// 2. Apply ON_TICK effects (HP / stones / immortal stones) BEFORE the SQL
	//    decay pass so a Gu that starves this tick still produces its ON_TICK.
	for _, pID := range subset {
		if err := applyOnTickEffects(db, pID); err != nil {
			logger.S.Errorf("Hunger Worker Error applying ON_TICK for player %s: %v", pID, err)
		}
	}

	// 3. Set-based decay + starvation + HP backlash + audit log + equipped_gus.
	if err := runHungerDecayBatch(db); err != nil {
		logger.S.Errorf("Hunger Worker Error running decay batch: %v", err)
	}
}

// findOnTickPlayers returns the player ids that own at least one equipped
// ACTIVE Gu whose dictionary effects.other contains an ON_TICK trigger.
func findOnTickPlayers(db *sql.DB) []string {
	rows, err := db.Query(`
		SELECT DISTINCT pg.player_id::text
		FROM player_gu pg
		JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank
		WHERE pg.equipped AND pg.state = 'ACTIVE'
		  AND EXISTS (
			SELECT 1 FROM jsonb_array_elements(COALESCE(gd.effects->'other', '[]'::jsonb)) m
			WHERE m->>'event_trigger' = 'ON_TICK'
		  )`)
	if err != nil {
		logger.S.Errorf("Hunger Worker Error: failed to query ON_TICK players: %v", err)
		return nil
	}
	defer rows.Close()

	var playerIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			playerIDs = append(playerIDs, id)
		}
	}
	return playerIDs
}

// applyOnTickEffects handles the rare ON_TICK effects (HOT_TICK,
// GENERATE_STONES_TICK, GENERATE_IMMORTAL_STONES_TICK) for a single player's
// equipped ACTIVE Gus. It only updates player counters (hp / stones), never
// hunger or Gu state.
func applyOnTickEffects(db *sql.DB, playerID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentMana, maxMana, immortalStones float64
	var primevalStones float64
	var playerRank int
	var hp, baseHP float64
	err = tx.QueryRow(`
		SELECT primeval_stones, current_mana, max_mana, immortal_stones, rank, hp, base_hp
		FROM players WHERE id = $1`, playerID).Scan(
		&primevalStones, &currentMana, &maxMana, &immortalStones, &playerRank, &hp, &baseHP)
	if err != nil {
		return err
	}

	tickStones := 0.0
	tickHP := 0.0
	tickImmortalStones := 0.0

	rows, err := tx.Query(`
		SELECT pg.gu_id, pg.gu_rank, gd.effects->'other'
		FROM player_gu pg
		JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank
		WHERE pg.player_id = $1 AND pg.equipped AND pg.state = 'ACTIVE'`, playerID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var guID string
		var guRank int
		var otherRaw []byte
		if err := rows.Scan(&guID, &guRank, &otherRaw); err != nil {
			return err
		}
		var other []models.SpecialMechanic
		if len(otherRaw) > 0 {
			if err := json.Unmarshal(otherRaw, &other); err != nil {
				continue
			}
		}

		tnMultiplier := 1.0
		if strings.HasPrefix(guID, "GU_HEAVENLY_ESSENCE_LOTUS") {
			tnMultiplier = getThienNguyenMultiplier(playerRank, guRank)
		}
		for _, mech := range other {
			if mech.EventTrigger != "ON_TICK" {
				continue
			}
			var params map[string]interface{}
			if len(mech.Params) > 0 {
				json.Unmarshal(mech.Params, &params)
			}
			switch mech.ActionCode {
			case "HOT_TICK":
				val, _ := params["value"].(float64)
				if val <= 0 {
					val = 5.0
				}
				tickHP += val * tnMultiplier
			case "GENERATE_STONES_TICK":
				val, _ := params["value"].(float64)
				if val <= 0 {
					val = 1.0
				}
				tickStones += val * tnMultiplier
			case "GENERATE_IMMORTAL_STONES_TICK":
				val, _ := params["value"].(float64)
				if val <= 0 {
					val = 1.0
				}
				tickImmortalStones += val * tnMultiplier
			}
		}
	}

	if tickHP > 0 {
		hp += tickHP
		if hp > baseHP {
			hp = baseHP
		}
	}
	if tickStones > 0 {
		primevalStones += tickStones
	}
	if tickImmortalStones > 0 {
		immortalStones += tickImmortalStones
	}

	if tickHP == 0 && tickStones == 0 && tickImmortalStones == 0 {
		return tx.Rollback()
	}

	_, err = tx.Exec(`UPDATE players SET hp = $1, primeval_stones = $2, immortal_stones = $3 WHERE id = $4`,
		hp, primevalStones, immortalStones, playerID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// runHungerDecayBatch applies hunger decay, starvation (sleep/death),
// vital HP backlash, death audit logs and equipped_gus cleanup in a single
// set-based transaction across all players.
func runHungerDecayBatch(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Materialize the per-Gu decay result into a temp table so subsequent
	// statements (state update, backlash, audit, cleanup) all agree on the
	// exact same random rolls and computed values.
	_, err = tx.Exec(`
		CREATE TEMP TABLE _hunger_tick ON COMMIT DROP AS
		WITH decayed AS (
			SELECT
				pg.id AS gu_instance_id,
				pg.player_id,
				pg.gu_id,
				pg.gu_rank,
				pg.is_vital,
				pg.equipped,
				pg.state,
				CASE
					WHEN pg.state = 'ACTIVE' THEN pg.hunger - gd.hunger_rate
					WHEN pg.state = 'SLEEPING' THEN pg.hunger - 0.25 * gd.hunger_rate
					WHEN pg.state = 'SEALED' THEN pg.hunger - 0.1 * gd.hunger_rate
					ELSE pg.hunger
				END AS new_hunger,
				gd.name AS gu_name
			FROM player_gu pg
			JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank
			WHERE pg.state IN ('ACTIVE', 'SLEEPING', 'SEALED')
		)
		SELECT
			d.gu_instance_id,
			d.player_id,
			d.gu_id,
			d.gu_rank,
			d.is_vital,
			d.equipped,
			d.gu_name,
			GREATEST(d.new_hunger, 0) AS new_hunger,
			CASE
				WHEN d.new_hunger <= 0 THEN 'DEAD'
				ELSE d.state
			END AS new_state
		FROM decayed d`)
	if err != nil {
		return err
	}

	// Apply decay + state transitions.
	_, err = tx.Exec(`
		UPDATE player_gu pg
		SET hunger = t.new_hunger,
		    state = t.new_state,
		    updated_at = NOW()
		FROM _hunger_tick t
		WHERE pg.id = t.gu_instance_id`)
	if err != nil {
		return err
	}

	// Vital Gu death → 80 HP backlash per death; drop active_combat on collapse.
	_, err = tx.Exec(`
		UPDATE players p
		SET hp = GREATEST(p.hp - 80.0 * COALESCE(v.cnt, 0), 0),
		    active_combat = CASE WHEN p.hp - 80.0 * COALESCE(v.cnt, 0) <= 0 THEN NULL ELSE p.active_combat END
		FROM (
			SELECT player_id, COUNT(*) AS cnt
			FROM _hunger_tick
			WHERE is_vital AND new_state = 'DEAD'
			GROUP BY player_id
		) v
		WHERE p.id = v.player_id`)
	if err != nil {
		return err
	}

	// Audit log for every Gu that starved this tick (vital vs regular death).
	_, err = tx.Exec(`
		INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
		SELECT
			t.player_id,
			CASE WHEN t.is_vital THEN 'gu.starve_vital_death' ELSE 'gu.starve_death' END,
			'gu_instance',
			t.gu_id,
			CASE WHEN t.is_vital THEN t.gu_name || ' | bản mệnh Cổ chết đói' ELSE t.gu_name || ' | Cổ chết đói' END,
			'{}'::jsonb,
			'',
			false,
			'Chết đói',
			NOW()
		FROM _hunger_tick t
		WHERE t.new_state = 'DEAD'`)
	if err != nil {
		return err
	}

	// Rebuild equipped_gus for players whose Gu just died or went dormant:
	// only ACTIVE equipped Gus stay in the equipped_gus array.
	_, err = tx.Exec(`
		UPDATE players p
		SET equipped_gus = COALESCE(
			(SELECT jsonb_agg(pg.id ORDER BY pg.id)
			 FROM player_gu pg
			 WHERE pg.player_id = p.id AND pg.equipped AND pg.state = 'ACTIVE'),
			'[]'::jsonb)
		FROM _hunger_tick t
		WHERE p.id = t.player_id AND t.new_state IN ('DEAD', 'SLEEPING', 'SEALED')`)
	if err != nil {
		return err
	}

	// Remove starved (now DEAD) Gu instances.
	_, err = tx.Exec(`DELETE FROM player_gu pg USING _hunger_tick t WHERE pg.id = t.gu_instance_id AND t.new_state = 'DEAD'`)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func getThienNguyenMultiplier(playerRank, guRank int) float64 {
	if playerRank == guRank {
		return 2.0
	}
	immortalCost := func(diff int) float64 {
		return math.Pow(100, float64(diff))
	}
	mortalCost := map[int]float64{1: 1, 2: 2, 3: 5, 4: 20, 5: 100}
	if playerRank >= 6 && guRank >= 6 {
		diff := playerRank - guRank
		if diff > 0 {
			return 2.0 / immortalCost(diff)
		}
		return 2.0 * immortalCost(-diff)
	}
	if playerRank >= 6 {
		return 2.0 / immortalCost(playerRank-guRank)
	}
	if guRank >= 6 {
		return 0
	}
	playerCost := mortalCost[playerRank]
	guCost := mortalCost[guRank]
	if playerCost == 0 || guCost == 0 {
		return 0
	}
	return 2.0 * guCost / playerCost
}
