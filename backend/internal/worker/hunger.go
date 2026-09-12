package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

var (
	hungerWorkerStarted bool
	hungerWorkerMutex   sync.Mutex

	// PlayerEventPublisher is an optional callback to broadcast player updates over WebSocket.
	PlayerEventPublisher func(playerID uuid.UUID, eventType string)
)

// StartHungerWorker begins the background process for depleting Gu bug hunger.
func StartHungerWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	hungerWorkerMutex.Lock()
	if hungerWorkerStarted {
		hungerWorkerMutex.Unlock()
		return
	}
	hungerWorkerStarted = true
	hungerWorkerMutex.Unlock()

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
				processHungerDecay(ctx, db)
			}
		}
	}()
}

// processHungerDecay handles background tasks for Gu ticks.
// Note: Gu hunger decay for players is handled lazily on-demand
// (EvaluatePlayerHungerTx). Sect vault Gu hunger decay has been eliminated
// to save CPU/DB overhead; withdrawn Gus start at hunger 1 instead.
// This worker only processes lightweight ON_TICK effects.
func processHungerDecay(ctx context.Context, db *sql.DB) {
	// 1. Batch ON_TICK effects (generating stones from special Gus)
	batchApplyOnTickEffects(db)
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

	for pid := range stoneMap {
		ps := tickStones[pid]
		is := tickImmortal[pid]
		if ps == 0 && is == 0 {
			continue
		}
		updateIDs = append(updateIDs, pid)
		updatePrimeval = append(updatePrimeval, ps)
		updateImmortal = append(updateImmortal, is)
	}

	if len(updateIDs) == 0 {
		return
	}

	// 5. Batch UPDATE
	_, err = db.Exec(`
		UPDATE players AS p SET
			primeval_stones = p.primeval_stones + v.ps,
			immortal_stones = p.immortal_stones + v.is2
		FROM (SELECT unnest($1::uuid[]) AS id,
		             unnest($2::float8[]) AS ps,
		             unnest($3::float8[]) AS is2) v
		WHERE p.id = v.id
	`, pq.Array(updateIDs), pq.Array(updatePrimeval), pq.Array(updateImmortal))
	if err != nil {
		logger.S.Errorf("Hunger Worker Error batch updating stones: %v", err)
	}
}

// ── Gu Hunger Starvation Notifications ──────────────────────────────────────

// StarvedGuInfo holds information about a Gu that died of starvation.
type StarvedGuInfo struct {
	ID, GuID, State, GuName string
	IsVital, Equipped       bool
}

// FormatStarvedGuNotification formats a grouped notification for normal Gus that starved to death.
func FormatStarvedGuNotification(normalDead []StarvedGuInfo) (title string, content string) {
	if len(normalDead) == 0 {
		return "", ""
	}
	title = "Cổ trùng chết đói"
	if len(normalDead) == 1 {
		content = fmt.Sprintf("%s đã chết vì đói!", normalDead[0].GuName)
		return
	}

	nameCounts := make(map[string]int)
	var uniqueNames []string
	for _, s := range normalDead {
		if nameCounts[s.GuName] == 0 {
			uniqueNames = append(uniqueNames, s.GuName)
		}
		nameCounts[s.GuName]++
	}

	var parts []string
	for _, name := range uniqueNames {
		if cnt := nameCounts[name]; cnt > 1 {
			parts = append(parts, fmt.Sprintf("%s x%d", name, cnt))
		} else {
			parts = append(parts, name)
		}
	}

	var detail string
	if len(parts) <= 5 {
		detail = strings.Join(parts, ", ")
	} else {
		detail = fmt.Sprintf("%s và %d loại khác", strings.Join(parts[:5], ", "), len(parts)-5)
	}
	content = fmt.Sprintf("%d cổ trùng đã chết vì đói: %s.", len(normalDead), detail)
	return
}

// FormatVitalGuNotification formats a grouped notification for vital Gus that starved to death.
func FormatVitalGuNotification(vitalDead []StarvedGuInfo) (title string, content string) {
	if len(vitalDead) == 0 {
		return "", ""
	}
	title = "Bản mệnh cổ chết đói"
	if len(vitalDead) == 1 {
		content = fmt.Sprintf("Bản mệnh cổ %s đã chết vì đói! Bạn nhận 80 sát thương.", vitalDead[0].GuName)
		return
	}

	var vNames []string
	for _, s := range vitalDead {
		vNames = append(vNames, s.GuName)
	}
	backlash := 80.0 * float64(len(vitalDead))
	content = fmt.Sprintf("%d bản mệnh cổ đã chết vì đói: %s! Bạn nhận %.0f sát thương phản phệ.", len(vitalDead), strings.Join(vNames, ", "), backlash)
	return
}

// EvaluatePlayerHungerTx performs lazy on-demand decay of a player's Gus.
// It checks the player's last_hunger_tick_at timestamp, and if one or more
// 3-hour intervals have passed, decays the player's Gus according to their state.
func EvaluatePlayerHungerTx(tx *sql.Tx, playerID uuid.UUID) error {
	var lastTick time.Time
	err := tx.QueryRow(`
		SELECT COALESCE(last_hunger_tick_at, created_at, NOW())
		FROM players
		WHERE id = $1
		FOR UPDATE
	`, playerID).Scan(&lastTick)
	if err != nil {
		return err
	}

	now := time.Now()
	const tickInterval = 3 * time.Hour
	if now.Before(lastTick) {
		return nil
	}

	ticks := int(now.Sub(lastTick) / tickInterval)
	if ticks <= 0 {
		return nil
	}

	// Apply decay + capture every Gu that starved in one pass.
	rows, err := tx.Query(`
		WITH decayed AS (
			SELECT pg.id AS gu_instance_id, pg.gu_id, pg.is_vital, pg.equipped,
				CASE WHEN pg.state='ACTIVE' THEN pg.hunger - ($2 * gd.hunger_rate)
				     WHEN pg.state='SLEEPING' THEN pg.hunger - ($2 * 0.25 * gd.hunger_rate)
				     WHEN pg.state='SEALED' THEN pg.hunger - ($2 * 0.1 * gd.hunger_rate)
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
		SELECT id, gu_id, state, is_vital, equipped, gu_name FROM upd`, playerID.String(), ticks)
	if err != nil {
		return err
	}

	var dead []StarvedGuInfo
	var deadIDs []string
	vitalDeaths := 0
	equippedDeath := false

	for rows.Next() {
		var s StarvedGuInfo
		if err := rows.Scan(&s.ID, &s.GuID, &s.State, &s.IsVital, &s.Equipped, &s.GuName); err != nil {
			rows.Close()
			return err
		}
		if s.State == "DEAD" {
			dead = append(dead, s)
			deadIDs = append(deadIDs, s.ID)
			if s.IsVital {
				vitalDeaths++
			}
			if s.Equipped {
				equippedDeath = true
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// Advance last_hunger_tick_at preserving remainder
	newTick := lastTick.Add(time.Duration(ticks) * tickInterval)
	if _, err := tx.Exec(`UPDATE players SET last_hunger_tick_at = $1 WHERE id = $2`, newTick, playerID); err != nil {
		return err
	}

	if len(dead) == 0 {
		return nil
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

	// Send notifications and audit logs (using tx).
	var normalDead []StarvedGuInfo
	var vitalDead []StarvedGuInfo

	for _, s := range dead {
		if s.IsVital {
			vitalDead = append(vitalDead, s)
		} else {
			normalDead = append(normalDead, s)
		}

		action := "gu.starve_death"
		content := fmt.Sprintf("%s đã chết vì đói!", s.GuName)
		if s.IsVital {
			action = "gu.starve_vital_death"
			content = fmt.Sprintf("Bản mệnh cổ %s đã chết vì đói! Bạn nhận 80 sát thương.", s.GuName)
		}
		auditlog.Log(tx, auditlog.Entry{
			PlayerID:       &playerID,
			Action:         action,
			EntityType:     "gu_instance",
			EntityID:       s.ID,
			RequestSummary: content,
			Metadata: map[string]interface{}{
				"gu_instance_id": s.ID,
				"gu_id":          s.GuID,
				"gu_name":        s.GuName,
				"is_vital":       s.IsVital,
			},
			Success: true,
		})
	}

	// Grouped notification for normal starved Gus
	if title, content := FormatStarvedGuNotification(normalDead); title != "" {
		notify.Send(tx, playerID, "gu_death", title, content)
	}

	// Grouped notification for vital starved Gus
	if title, content := FormatVitalGuNotification(vitalDead); title != "" {
		notify.Send(tx, playerID, "vital_gu_death", title, content)
	}

	if len(dead) > 0 && PlayerEventPublisher != nil {
		PlayerEventPublisher(playerID, "player.updated")
	}

	return nil
}

// processPlayerHungerDecay applies decay + starvation for one player.
func processPlayerHungerDecay(db *sql.DB, playerID string) error {
	parsedPID, err := uuid.Parse(playerID)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := EvaluatePlayerHungerTx(tx, parsedPID); err != nil {
		return err
	}
	return tx.Commit()
}

// ── Sect vault hunger decay (batch) ────────────────────────────────────────

// vaultGuItem is the JSONB shape of a single entry in sects.vault_gu (used in tests).
type vaultGuItem struct {
	ID       string  `json:"id"`
	Rank     int     `json:"rank,omitempty"`
	Quantity int     `json:"quantity"`
	Hunger   float64 `json:"hunger,omitempty"`
}

// EvaluateSectVaultHungerTx is a no-op because hunger decay for sect vaults
// has been disabled to eliminate background/on-demand CPU overhead and row locking.
// Gu in the sect vault does not decay or starve. When withdrawn, its hunger is set to 1.
func EvaluateSectVaultHungerTx(tx *sql.Tx, sectID string) error {
	return nil
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
