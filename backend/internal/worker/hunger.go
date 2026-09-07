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

// processHungerDecay depletes Gu hunger for every player.
func processHungerDecay(db *sql.DB) {
	// 1. Batch ON_TICK effects (all players in one pass)
	batchApplyOnTickEffects(db)

	// 2. Per-player decay + starvation (sleep/death) + HP backlash + audit logs
	//    + equipped_gus cleanup.  Per-player transactions are intentional:
	//    a single UPDATE over the whole player_gu table exceeds the
	//    statement_timeout and deadlocks against concurrent gameplay writes.
	if err := runHungerDecayBatch(db); err != nil {
		logger.S.Errorf("Hunger Worker Error running decay batch: %v", err)
	}

	// 3. Sect vault hunger decay — batch across all sects.
	batchVaultHungerDecay(db)
}

// ── ON_TICK batch ──────────────────────────────────────────────────────────

// findOnTickPlayers returns the player ids that own at least one equipped
// ACTIVE Gu whose dictionary effects.other contains an ON_TICK trigger.
func findOnTickPlayers(db *sql.DB) ([]string, error) {
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
		return nil, err
	}
	defer rows.Close()
	var playerIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			playerIDs = append(playerIDs, id)
		}
	}
	return playerIDs, rows.Err()
}

// batchApplyOnTickEffects replaces the old per-player applyOnTickEffects loop.
// It batch-reads all ON_TICK players' stones and equipped Gu effects in 2
// queries, computes stone generation in Go, then batch-writes in 1 UPDATE.
func batchApplyOnTickEffects(db *sql.DB) {
	playerIDs, err := findOnTickPlayers(db)
	if err != nil {
		logger.S.Errorf("Hunger Worker Error: failed to query ON_TICK players: %v", err)
		return
	}
	if len(playerIDs) == 0 {
		return
	}

	pqIDs := pq.Array(playerIDs)

	// 1. Batch-read stones
	type stones struct {
		primeval float64
		immortal float64
	}
	stoneMap := make(map[string]*stones, len(playerIDs))
	sRows, err := db.Query(`SELECT id, primeval_stones, immortal_stones FROM players WHERE id = ANY($1::uuid[])`, pqIDs)
	if err != nil {
		logger.S.Errorf("Hunger Worker Error reading stones: %v", err)
		return
	}
	for sRows.Next() {
		var pid string
		var s stones
		if sRows.Scan(&pid, &s.primeval, &s.immortal) == nil {
			stoneMap[pid] = &s
		}
	}
	sRows.Close()

	// 2. Batch-read ON_TICK effects for all equipped ACTIVE Gus
	type effectRow struct {
		playerID string
		otherRaw []byte
	}
	var effectRows []effectRow
	eRows, err := db.Query(`
		SELECT pg.player_id::text, gd.effects->'other'
		FROM player_gu pg
		JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank
		WHERE pg.player_id = ANY($1::uuid[]) AND pg.equipped AND pg.state = 'ACTIVE'`, pqIDs)
	if err != nil {
		logger.S.Errorf("Hunger Worker Error reading ON_TICK effects: %v", err)
		return
	}
	for eRows.Next() {
		var er effectRow
		if eRows.Scan(&er.playerID, &er.otherRaw) == nil {
			effectRows = append(effectRows, er)
		}
	}
	eRows.Close()

	// 3. Compute per-player stone generation
	tickStones := make(map[string]float64)
	tickImmortal := make(map[string]float64)

	for _, er := range effectRows {
		var other []models.SpecialMechanic
		if len(er.otherRaw) > 0 {
			if err := json.Unmarshal(er.otherRaw, &other); err != nil {
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
				tickStones[er.playerID] += val
			case "GENERATE_IMMORTAL_STONES_TICK":
				val, _ := params["value"].(float64)
				if val <= 0 {
					val = 1.0
				}
				tickImmortal[er.playerID] += val
			}
		}
	}

	// 4. Collect players that actually need an update
	var updateIDs []string
	var updatePrimeval []float64
	var updateImmortal []float64

	for pid, s := range stoneMap {
		ps := tickStones[pid]
		is := tickImmortal[pid]
		if ps == 0 && is == 0 {
			continue
		}
		updateIDs = append(updateIDs, pid)
		updatePrimeval = append(updatePrimeval, s.primeval+ps)
		updateImmortal = append(updateImmortal, s.immortal+is)
	}

	if len(updateIDs) == 0 {
		return
	}

	// 5. Batch UPDATE
	_, err = db.Exec(`
		UPDATE players AS p SET
			primeval_stones = v.ps,
			immortal_stones = v.is
		FROM (SELECT unnest($1::uuid[]) AS id,
		             unnest($2::float8[]) AS ps,
		             unnest($3::float8[]) AS is2) v
		WHERE p.id = v.id
	`, pq.Array(updateIDs), pq.Array(updatePrimeval), pq.Array(updateImmortal))
	if err != nil {
		logger.S.Errorf("Hunger Worker Error batch updating stones: %v", err)
	}
}

// ── Hunger decay (per-player, intentional for deadlock safety) ─────────────

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

	// Batch audit logs for all dead Gus in one multi-row INSERT.
	if len(dead) > 0 {
		actions := make([]string, 0, len(dead))
		summaries := make([]string, 0, len(dead))
		guIDs := make([]string, 0, len(dead))
		for _, s := range dead {
			if s.isVital {
				actions = append(actions, "gu.starve_vital_death")
				summaries = append(summaries, s.guName+" | bản mệnh Cổ chết đói")
			} else {
				actions = append(actions, "gu.starve_death")
				summaries = append(summaries, s.guName+" | Cổ chết đói")
			}
			guIDs = append(guIDs, s.guID)
		}
		if _, err := tx.Exec(`
			INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
			SELECT $1::uuid, v.action, 'gu_instance', v.entity_id, v.summary, '{}'::jsonb, '', false, 'Chết đói', NOW()
			FROM (SELECT unnest($2::text[]) AS action,
			             unnest($3::text[]) AS entity_id,
			             unnest($4::text[]) AS summary) v
		`, playerID, pq.Array(actions), pq.Array(guIDs), pq.Array(summaries)); err != nil {
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

// ── Sect vault hunger decay (batch) ────────────────────────────────────────

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

// batchVaultHungerDecay replaces the old per-sect transaction loop. It
// batch-reads all sect vaults in 1 query, decays hunger in Go, then batch-writes
// all modified vaults + audit logs in a single transaction.
func batchVaultHungerDecay(db *sql.DB) {
	// 1. Batch-read all sects with non-empty vault_gu
	type sectVault struct {
		id   string
		raw  []byte
	}
	var vaults []sectVault
	vRows, err := db.Query(`SELECT id, vault_gu FROM sects WHERE vault_gu <> '[]'::jsonb`)
	if err != nil {
		logger.S.Errorf("Hunger Worker Error reading vault Gus: %v", err)
		return
	}
	for vRows.Next() {
		var sv sectVault
		if vRows.Scan(&sv.id, &sv.raw) == nil {
			vaults = append(vaults, sv)
		}
	}
	vRows.Close()

	if len(vaults) == 0 {
		return
	}

	// 2. Decay hunger in Go for each vault
	type auditEntry struct {
		sectID  string
		summary string
	}
	var modifiedIDs []string
	var modifiedRaw []string
	var auditLogs []auditEntry

	for _, sv := range vaults {
		var items []vaultGuItem
		if len(sv.raw) > 0 {
			if err := json.Unmarshal(sv.raw, &items); err != nil {
				continue
			}
		}

		changed := false
		for i := range items {
			maxH, rate := vaultGuSpec(items[i].ID, items[i].Rank)
			if items[i].Hunger <= 0 {
				items[i].Hunger = maxH
			}
			items[i].Hunger -= rate
			if items[i].Hunger < 0 {
				items[i].Hunger = 0
			}
			changed = true
		}

		// Remove starved groups
		kept := items[:0]
		for _, it := range items {
			if it.Hunger > 0 {
				kept = append(kept, it)
			} else {
				name := it.ID
				if spec, ok := models.GetGu(it.ID, it.Rank); ok {
					name = spec.Name
				}
				auditLogs = append(auditLogs, auditEntry{
					sectID:  sv.id,
					summary: name + " | Cổ trong Bảo khố chết đói",
				})
			}
		}
		items = kept

		if changed {
			newRaw, err := json.Marshal(items)
			if err != nil {
				continue
			}
			modifiedIDs = append(modifiedIDs, sv.id)
			modifiedRaw = append(modifiedRaw, string(newRaw))
		}
	}

	// 3. Batch-write modified vaults + audit logs in one transaction
	if len(modifiedIDs) == 0 && len(auditLogs) == 0 {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		logger.S.Errorf("Hunger Worker Error starting vault batch tx: %v", err)
		return
	}
	defer tx.Rollback()

	if len(modifiedIDs) > 0 {
		if _, err := tx.Exec(`
			UPDATE sects AS s SET vault_gu = v.vault
			FROM (SELECT unnest($1::text[]) AS id, unnest($2::jsonb[]) AS vault) v
			WHERE s.id = v.id
		`, pq.Array(modifiedIDs), pq.Array(modifiedRaw)); err != nil {
			logger.S.Errorf("Hunger Worker Error batch updating vaults: %v", err)
			return
		}
	}

	if len(auditLogs) > 0 {
		auditIDs := make([]string, 0, len(auditLogs))
		auditSummaries := make([]string, 0, len(auditLogs))
		for _, al := range auditLogs {
			auditIDs = append(auditIDs, al.sectID)
			auditSummaries = append(auditSummaries, al.summary)
		}
		if _, err := tx.Exec(`
			INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at)
			SELECT NULL, 'sect.vault_gu_starve', 'sect', v.entity_id, v.summary, '{}'::jsonb, '', false, 'Chết đói trong Bảo khố', NOW()
			FROM (SELECT unnest($1::text[]) AS entity_id,
			             unnest($2::text[]) AS summary) v
		`, pq.Array(auditIDs), pq.Array(auditSummaries)); err != nil {
			logger.S.Errorf("Hunger Worker Error batch inserting vault audit logs: %v", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Hunger Worker Error committing vault batch: %v", err)
	}
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
