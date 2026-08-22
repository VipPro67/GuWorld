package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
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

// processHungerDecay depletes Gu hunger for every player. Work is done per
// player in small transactions rather than one giant set-based batch: a single
// UPDATE over the whole player_gu table exceeds the statement_timeout configured
// in db.Connect and deadlocks against concurrent gameplay writes on large data
// sets. Only the rare ON_TICK effects (Thien Nguyen Bao Lien series,
// GU_CREATION_ORIGIN) are handled in Go, and only for the small subset of
// players that actually own an equipped ACTIVE Gu with such effects.
func processHungerDecay(db *sql.DB) {
	// 1. Find players who own an equipped ACTIVE Gu carrying ON_TICK effects.
	subset := findOnTickPlayers(db)

	// 2. Apply ON_TICK effects (HP / stones / immortal stones) BEFORE the decay
	//    pass so a Gu that starves this tick still produces its ON_TICK.
	for _, pID := range subset {
		if err := applyOnTickEffects(db, pID); err != nil {
			logger.S.Errorf("Hunger Worker Error applying ON_TICK for player %s: %v", pID, err)
		}
	}

	// 3. Per-player decay + starvation (sleep/death) + HP backlash + audit logs
	//    + equipped_gus cleanup.
	if err := runHungerDecayBatch(db); err != nil {
		logger.S.Errorf("Hunger Worker Error running decay batch: %v", err)
	}

	// 4. Sect vault (kho tông môn) decay: Gu stored in a vault still get hungry
	//    at the ACTIVE rate and starve to death in storage.
	processVaultHungerDecay(db)
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

// applyOnTickEffects handles the rare ON_TICK effects (GENERATE_STONES_TICK,
// GENERATE_IMMORTAL_STONES_TICK) for a single player's equipped ACTIVE Gus. It
// only updates player counters (stones), never hunger or Gu state.
// MANA_REGEN_TICK is handled separately as an hourly rate (see
// models.ManaRegenTickPerHour), so it is not applied here. HP regeneration is
// handled by the lazy projected-status service, so HOT_TICK no longer heals.
func applyOnTickEffects(db *sql.DB, playerID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var immortalStones float64
	var primevalStones float64
	err = tx.QueryRow(`
		SELECT primeval_stones, immortal_stones
		FROM players WHERE id = $1`, playerID).Scan(
		&primevalStones, &immortalStones)
	if err != nil {
		return err
	}

	tickStones := 0.0
	tickImmortalStones := 0.0

	rows, err := tx.Query(`
		SELECT gd.effects->'other'
		FROM player_gu pg
		JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank
		WHERE pg.player_id = $1 AND pg.equipped AND pg.state = 'ACTIVE'`, playerID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var otherRaw []byte
		if err := rows.Scan(&otherRaw); err != nil {
			return err
		}
		var other []models.SpecialMechanic
		if len(otherRaw) > 0 {
			if err := json.Unmarshal(otherRaw, &other); err != nil {
				continue
			}
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
			case "GENERATE_STONES_TICK":
				val, _ := params["value"].(float64)
				if val <= 0 {
					val = 1.0
				}
				tickStones += val
			case "GENERATE_IMMORTAL_STONES_TICK":
				val, _ := params["value"].(float64)
				if val <= 0 {
					val = 1.0
				}
				tickImmortalStones += val
			}
		}
	}

	if tickStones > 0 {
		primevalStones += tickStones
	}
	if tickImmortalStones > 0 {
		immortalStones += tickImmortalStones
	}

	if tickStones == 0 && tickImmortalStones == 0 {
		return tx.Rollback()
	}

	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, immortal_stones = $2 WHERE id = $3`,
		primevalStones, immortalStones, playerID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// runHungerDecayBatch applies hunger decay for every player that owns a Gu in a
// decayable state. Each player is processed in its own small transaction so lock
// hold times stay short (no deadlock with concurrent gameplay) and each statement
// stays well under the statement_timeout configured in db.Connect.
func runHungerDecayBatch(db *sql.DB) error {
	playerIDs, err := findDecayPlayers(db)
	if err != nil {
		return err
	}

	logger.S.Infof("Hunger Worker: processing hunger decay for %d players", len(playerIDs))

	for _, pID := range playerIDs {
		if err := processPlayerHungerDecay(db, pID); err != nil {
			// One player failing must not abort the rest of the batch.
			logger.S.Errorf("Hunger Worker Error updating player %s: %v", pID, err)
		}
	}
	return nil
}

// findDecayPlayers returns the ids of all players owning at least one Gu in a
// decayable state (ACTIVE / SLEEPING / SEALED).
func findDecayPlayers(db *sql.DB) ([]string, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// The DISTINCT scan over the full player_gu table can exceed the default
	// 15s statement timeout; relax it for this transaction only.
	if _, err := tx.Exec(`SET LOCAL statement_timeout = '600000'`); err != nil {
		return nil, err
	}

	rows, err := tx.Query(`SELECT DISTINCT player_id::text FROM player_gu WHERE state IN ('ACTIVE','SLEEPING','SEALED')`)
	if err != nil {
		return nil, err
	}

	var playerIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		playerIDs = append(playerIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	return playerIDs, tx.Commit()
}

// processPlayerHungerDecay applies decay + starvation for one player, retrying a
// few times when the small per-player transaction hits a deadlock or lock
// timeout against concurrent gameplay.
func processPlayerHungerDecay(db *sql.DB, playerID string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = processPlayerHungerDecayOnce(db, playerID)
		if lastErr == nil {
			return nil
		}
		if !isDeadlockOrLockTimeout(lastErr) {
			return lastErr
		}
		logger.S.Warnf("Hunger Worker: deadlock/lock timeout on player %s (attempt %d/3): %v", playerID, attempt+1, lastErr)
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return fmt.Errorf("hunger decay failed for player %s after 3 attempts: %w", playerID, lastErr)
}

// processPlayerHungerDecayOnce decays hunger for one player's Gus, transitions
// starved Gus to DEAD, applies the vital HP backlash, writes audit logs, rebuilds
// equipped_gus and deletes the dead instances. All steps share one transaction so
// the outcome is atomic per player.
func processPlayerHungerDecayOnce(db *sql.DB, playerID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Relax the default 15s statement / 10s lock timeout for this transaction;
	// a player with a very large inventory legitimately needs more time.
	if _, err := tx.Exec(`SET LOCAL statement_timeout = '600000'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`SET LOCAL lock_timeout = '60000'`); err != nil {
		return err
	}

	// Apply decay + capture every Gu that starved this tick in one pass.
	rows, err := tx.Query(`
		WITH decayed AS (
			SELECT pg.id AS gu_instance_id, pg.gu_id, pg.is_vital, pg.equipped,
				CASE WHEN pg.state='ACTIVE' THEN pg.hunger - gd.hunger_rate
				     WHEN pg.state='SLEEPING' THEN pg.hunger - 0.25 * gd.hunger_rate
				     WHEN pg.state='SEALED' THEN pg.hunger - 0.1 * gd.hunger_rate
				     ELSE pg.hunger END AS new_hunger,
				gd.name AS gu_name
			FROM player_gu pg
			JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank
			WHERE pg.player_id = $1 AND pg.state IN ('ACTIVE','SLEEPING','SEALED')
		),
		upd AS (
			UPDATE player_gu pg
			SET hunger = GREATEST(d.new_hunger, 0),
			    state = CASE WHEN d.new_hunger <= 0 THEN 'DEAD' ELSE pg.state END,
			    updated_at = NOW()
			FROM decayed d
			WHERE pg.id = d.gu_instance_id
			RETURNING pg.id, pg.gu_id, pg.state, pg.is_vital, pg.equipped, d.gu_name
		)
		SELECT id, gu_id, state, is_vital, equipped, gu_name FROM upd`, playerID)
	if err != nil {
		return err
	}

	type starvedGu struct {
		id, guID, state, guName string
		isVital, equipped       bool
	}
	var dead []starvedGu
	var deadIDs []string
	vitalDeaths := 0
	equippedDeath := false

	for rows.Next() {
		var s starvedGu
		if err := rows.Scan(&s.id, &s.guID, &s.state, &s.isVital, &s.equipped, &s.guName); err != nil {
			rows.Close()
			return err
		}
		if s.state == "DEAD" {
			dead = append(dead, s)
			deadIDs = append(deadIDs, s.id)
			if s.isVital {
				vitalDeaths++
			}
			if s.equipped {
				equippedDeath = true
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if len(dead) == 0 {
		return tx.Commit()
	}

	// Vital Gu death → 80 HP backlash per death; drop active_combat on collapse.
	if vitalDeaths > 0 {
		backlash := 80.0 * float64(vitalDeaths)
		if _, err := tx.Exec(`
			UPDATE players p
			SET hp = GREATEST(p.hp - $1, 0),
			    active_combat = CASE WHEN p.hp - $1 <= 0 THEN NULL ELSE p.active_combat END
			WHERE p.id = $2`, backlash, playerID); err != nil {
			return err
		}
	}

	// Audit log for every Gu that starved this tick (vital vs regular death).
	for _, s := range dead {
		action := "gu.starve_death"
		summary := s.guName + " | Cổ chết đói"
		if s.isVital {
			action = "gu.starve_vital_death"
			summary = s.guName + " | bản mệnh Cổ chết đói"
		}
		if _, err := tx.Exec(`
			INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
			VALUES ($1, $2, 'gu_instance', $3, $4, '{}'::jsonb, '', false, 'Chết đói', NOW())`,
			playerID, action, s.guID, summary); err != nil {
			return err
		}
	}

	// Rebuild equipped_gus when an equipped Gu died: only ACTIVE equipped Gus
	// stay in the equipped_gus array.
	if equippedDeath {
		if _, err := tx.Exec(`
			UPDATE players p
			SET equipped_gus = COALESCE(
				(SELECT jsonb_agg(pg.id ORDER BY pg.id)
				 FROM player_gu pg
				 WHERE pg.player_id = p.id AND pg.equipped AND pg.state = 'ACTIVE'),
				'[]'::jsonb)
			WHERE p.id = $1`, playerID); err != nil {
			return err
		}
	}

	// Remove starved (now DEAD) Gu instances.
	if _, err := tx.Exec(`DELETE FROM player_gu WHERE id = ANY($1::text[])`, pq.Array(deadIDs)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// Send notifications after successful commit (best-effort).
	if parsedPID, err := uuid.Parse(playerID); err == nil {
		for _, s := range dead {
			ntype := "gu_death"
			title := "Cổ trùng chết đói"
			content := fmt.Sprintf("%s đã chết vì đói!", s.guName)
			if s.isVital {
				ntype = "vital_gu_death"
				title = "Bản mệnh cổ chết đói"
				content = fmt.Sprintf("Bản mệnh cổ %s đã chết vì đói! Bạn nhận 80 sát thương.", s.guName)
			}
			notify.Send(db, parsedPID, ntype, title, content)
		}
	}

	return nil
}

// ---- Sect vault (kho tông môn) hunger decay ---------------------------------

// vaultGuItem is the JSONB shape of a single entry in sects.vault_gu.
type vaultGuItem struct {
	ID       string  `json:"id"`
	Rank     int     `json:"rank,omitempty"`
	Quantity int     `json:"quantity"`
	Hunger   float64 `json:"hunger,omitempty"`
}

// vaultGuSpec returns (maxHunger, hungerRate) for a vault Gu group, with safe
// defaults when the dictionary entry is missing.
func vaultGuSpec(id string, rank int) (float64, float64) {
	if spec, ok := models.GetGu(id, rank); ok {
		return float64(spec.MaxHunger), float64(spec.HungerRate)
	}
	if spec, ok := models.GetGuAny(id); ok {
		return float64(spec.MaxHunger), float64(spec.HungerRate)
	}
	return 100, 0
}

// processVaultHungerDecay decays hunger for Gu stored in every sect vault at the
// same ACTIVE rate as equipped Gu, removing groups that starve to 0. Each sect
// is processed in its own short transaction.
func processVaultHungerDecay(db *sql.DB) {
	sectIDs, err := findSectIDsWithVaultGu(db)
	if err != nil {
		logger.S.Errorf("Hunger Worker Error finding sects with vault Gu: %v", err)
		return
	}
	for _, id := range sectIDs {
		if err := processSectVaultHungerDecay(db, id); err != nil {
			logger.S.Errorf("Hunger Worker Error decaying sect %s vault: %v", id, err)
		}
	}
}

func findSectIDsWithVaultGu(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT id FROM sects WHERE vault_gu <> '[]'::jsonb`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// processSectVaultHungerDecay retries a few times on transient deadlocks, mirroring
// the per-player decay loop.
func processSectVaultHungerDecay(db *sql.DB, sectID string) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		lastErr = processSectVaultHungerDecayOnce(db, sectID)
		if lastErr == nil {
			return nil
		}
		if !isDeadlockOrLockTimeout(lastErr) {
			return lastErr
		}
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}
	return lastErr
}

// processSectVaultHungerDecayOnce decays hunger for one sect vault and removes
// groups that starve, writing an audit log per removed group.
func processSectVaultHungerDecayOnce(db *sql.DB, sectID string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`SET LOCAL statement_timeout = '600000'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`SET LOCAL lock_timeout = '60000'`); err != nil {
		return err
	}

	var raw []byte
	if err := tx.QueryRow(`SELECT vault_gu FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&raw); err != nil {
		return err
	}

	var items []vaultGuItem
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &items); err != nil {
			return err
		}
	}

	type starved struct {
		name string
	}
	var starvedGroups []starved
	changed := false

	for i := range items {
		maxH, rate := vaultGuSpec(items[i].ID, items[i].Rank)
		if items[i].Hunger <= 0 {
			// Legacy/backfilled entry with no stored hunger: treat as full.
			items[i].Hunger = maxH
		}
		items[i].Hunger -= rate
		if items[i].Hunger < 0 {
			items[i].Hunger = 0
		}
		changed = true

		name := items[i].ID
		if spec, ok := models.GetGu(items[i].ID, items[i].Rank); ok {
			name = spec.Name
		}
		if items[i].Hunger <= 0 {
			starvedGroups = append(starvedGroups, starved{name: name})
		}
	}

	if len(starvedGroups) > 0 {
		kept := items[:0]
		for _, it := range items {
			if it.Hunger > 0 {
				kept = append(kept, it)
			}
		}
		items = kept
	}

	if changed {
		newRaw, err := json.Marshal(items)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE sects SET vault_gu = $1 WHERE id = $2`, newRaw, sectID); err != nil {
			return err
		}
	}

	for _, s := range starvedGroups {
		if _, err := tx.Exec(`
			INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
			VALUES (NULL, 'sect.vault_gu_starve', 'sect', $1, $2, '{}'::jsonb, '', false, 'Chết đói trong Bảo khố', NOW())`,
			sectID, s.name+" | Cổ trong Bảo khố chết đói"); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// isDeadlockOrLockTimeout reports whether err is a Postgres deadlock (40P01) or
// lock-not-available (55P03) error, i.e. a transient condition worth retrying.
func isDeadlockOrLockTimeout(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "40P01", "55P03":
			return true
		}
	}
	return false
}
