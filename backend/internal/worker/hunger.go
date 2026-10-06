package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

var (
	hungerWorkerStarted bool
	hungerWorkerMutex   sync.Mutex

	// PlayerEventPublisher is an optional callback to broadcast player updates over WebSocket.
	PlayerEventPublisher func(playerID uuid.UUID, eventType string, changedDomains ...string)
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
				processScheduledTicks(ctx, db)
			}
		}
	}()
}

// processScheduledTicks handles background SCHEDULED-timing effects.
// Note: Gu hunger decay for players is handled lazily on-demand
// (EvaluatePlayerHungerTx). Sect vault Gu hunger decay has been eliminated
// to save CPU/DB overhead; withdrawn Gus start at hunger 1 instead.
// This worker only processes lightweight ON_TICK effects.
func processScheduledTicks(ctx context.Context, db *sql.DB) {
	// 1. Batch ON_TICK effects (generating stones from special Gus)
	batchApplyOnTickEffects(db)
}

// ── ON_TICK batch ──────────────────────────────────────────────────────────

// findOnTickPlayers returns the player ids that own at least one equipped
// ACTIVE Gu whose compiled definition contains an ON_TICK trigger.
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
	Rank                    int
	IsVital, Equipped       bool
	// StateBefore là trạng thái (ACTIVE/SLEEPING/SEALED) trước khi chết, quyết
	// định tốc độ hao hụt. HungerBefore là độ no ngay trước kỳ decay kết liễu.
	// LastFedAt là mốc lần cuối được cho ăn (cột player_gu.last_fed_at).
	StateBefore  string
	HungerBefore float64
	LastFedAt    *time.Time
}

// formatUnfedDuration renders how long a Gu has gone without feeding, in
// Vietnamese. Returns "không rõ" when the last-fed timestamp is missing.
func formatUnfedDuration(lastFed *time.Time, now time.Time) string {
	if lastFed == nil {
		return "không rõ"
	}
	d := now.Sub(*lastFed)
	if d < 0 {
		d = 0
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if days := hours / 24; days > 0 {
		return fmt.Sprintf("%d ngày %d giờ", days, hours%24)
	}
	if hours > 0 {
		return fmt.Sprintf("%d giờ %d phút", hours, minutes)
	}
	return fmt.Sprintf("%d phút", minutes)
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

// playerLiveGuIDs returns the ids of a player's ACTIVE/SLEEPING/SEALED Gu,
// i.e. every instance EvaluatePlayerHungerTx needs to decay.
func playerLiveGuIDs(tx *sql.Tx, playerID uuid.UUID) ([]string, error) {
	rows, err := tx.Query(`
		SELECT id FROM player_gu
		WHERE player_id = $1 AND state IN ('ACTIVE','SLEEPING','SEALED')`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// NeedsHungerEvaluation performs a lock-free check to determine if a player's Gus
// require hunger decay evaluation (i.e. >= 3 hours since last_hunger_tick_at).
// Read paths can use this to skip acquiring a write transaction and FOR UPDATE lock entirely.
func NeedsHungerEvaluation(q interface{ QueryRow(query string, args ...any) *sql.Row }, playerID uuid.UUID) bool {
	var lastTick time.Time
	err := q.QueryRow(`
		SELECT COALESCE(last_hunger_tick_at, created_at, NOW())
		FROM players
		WHERE id = $1
	`, playerID).Scan(&lastTick)
	if err != nil {
		return false
	}
	return time.Since(lastTick) >= 3*time.Hour
}

// EvaluatePlayerHungerTx performs lazy on-demand decay of a player's Gus.
// It checks the player's last_hunger_tick_at timestamp, and if one or more
// 3-hour intervals have passed, decays the player's Gus according to their state.
func EvaluatePlayerHungerTx(tx *sql.Tx, playerID uuid.UUID) error {
	const tickInterval = 3 * time.Hour

	// Fast path: hunger only advances every 3h. Read without the players row
	// lock first so hot polling/mutation paths don't serialize on it; the lock
	// is acquired only when a tick is actually due.
	var lastTick time.Time
	err := tx.QueryRow(`
		SELECT COALESCE(last_hunger_tick_at, created_at, NOW())
		FROM players
		WHERE id = $1
	`, playerID).Scan(&lastTick)
	if err != nil {
		return err
	}

	now := time.Now()
	if now.Before(lastTick) || int(now.Sub(lastTick)/tickInterval) <= 0 {
		return nil
	}

	// A tick is due: lock the row and re-read so concurrent callers cannot
	// decay the same interval twice.
	err = tx.QueryRow(`
		SELECT COALESCE(last_hunger_tick_at, created_at, NOW())
		FROM players
		WHERE id = $1
		FOR UPDATE
	`, playerID).Scan(&lastTick)
	if err != nil {
		return err
	}
	if now.Before(lastTick) {
		return nil
	}

	ticks := int(now.Sub(lastTick) / tickInterval)
	if ticks <= 0 {
		return nil
	}

	// Apply decay + capture every Gu that starved. Processed in bounded chunks
	// so a large inventory (10k+ Gu) cannot blow the statement timeout; the
	// whole loop runs inside the caller's tx, so any failure rolls back
	// atomically and the tick is not advanced.
	const decayChunkSize = 500
	liveIDs, err := playerLiveGuIDs(tx, playerID)
	if err != nil {
		return err
	}

	var dead []StarvedGuInfo
	var deadIDs []string
	vitalDeaths := 0
	equippedDeath := false

	for start := 0; start < len(liveIDs); start += decayChunkSize {
		end := start + decayChunkSize
		if end > len(liveIDs) {
			end = len(liveIDs)
		}
		rows, err := tx.Query(`
			WITH decayed AS (
				SELECT pg.id AS gu_instance_id, pg.gu_id, pg.is_vital, pg.equipped,
					pg.hunger AS hunger_before, pg.state AS state_before,
					CASE WHEN pg.state='ACTIVE' THEN pg.hunger - ($1 * gd.hunger_rate)
					     WHEN pg.state='SLEEPING' THEN pg.hunger - ($1 * 0.25 * gd.hunger_rate)
					     WHEN pg.state='SEALED' THEN pg.hunger - ($1 * 0.1 * gd.hunger_rate)
					     ELSE pg.hunger END AS new_hunger,
					gd.name AS gu_name
				FROM player_gu pg
				JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank
				WHERE pg.id = ANY($2::text[])
			),
			upd AS (
				UPDATE player_gu pg
				SET hunger = GREATEST(d.new_hunger, 0),
				    state = CASE WHEN d.new_hunger <= 0 THEN 'DEAD' ELSE pg.state END
				FROM decayed d
				WHERE pg.id = d.gu_instance_id
				RETURNING pg.id, pg.gu_id, pg.gu_rank, pg.state, pg.is_vital, pg.equipped, d.gu_name,
					d.state_before, d.hunger_before, pg.last_fed_at
			)
			SELECT id, gu_id, gu_rank, state, is_vital, equipped, gu_name,
				state_before, hunger_before, last_fed_at FROM upd WHERE state = 'DEAD'`, ticks, pq.Array(liveIDs[start:end]))
		if err != nil {
			if IsHungerTransientError(err) {
				// PostgreSQL aborts the transaction after these errors. Return
				// the original code so the transaction owner can roll back and
				// retry the complete operation; swallowing it would only defer
				// failure until the next statement or commit.
				logger.S.Warnf("Hunger decay skipped for player %s (transient, retry later): %v", playerID, err)
			}
			return err
		}
		for rows.Next() {
			var s StarvedGuInfo
			if err := rows.Scan(&s.ID, &s.GuID, &s.Rank, &s.State, &s.IsVital, &s.Equipped, &s.GuName,
				&s.StateBefore, &s.HungerBefore, &s.LastFedAt); err != nil {
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
		unfed := formatUnfedDuration(s.LastFedAt, now)
		content := fmt.Sprintf("%s đã chết vì đói! Độ no trước khi chết: %.0f. Thời gian không được cho ăn: %s.", s.GuName, s.HungerBefore, unfed)
		if s.IsVital {
			action = "gu.starve_vital_death"
			content = fmt.Sprintf("Bản mệnh cổ %s đã chết vì đói! Độ no trước khi chết: %.0f. Thời gian không được cho ăn: %s. Bạn nhận 80 sát thương.", s.GuName, s.HungerBefore, unfed)
		}
		deathMeta := map[string]interface{}{
			"gu_instance_id": s.ID,
			"gu_id":          s.GuID,
			"gu_name":        s.GuName,
			"is_vital":       s.IsVital,
			"state_before":   s.StateBefore,
			"hunger_before":  s.HungerBefore,
		}
		if s.LastFedAt != nil {
			unfedHours := now.Sub(*s.LastFedAt).Hours()
			if unfedHours < 0 {
				unfedHours = 0
			}
			deathMeta["last_fed_at"] = s.LastFedAt.UTC().Format(time.RFC3339)
			deathMeta["unfed_hours"] = math.Round(unfedHours*10) / 10
		} else {
			deathMeta["last_fed_at"] = nil
		}
		_ = db.LogGuHistory(tx, s.ID, s.GuID, s.Rank, "STARVED_DEATH", &playerID, nil, content, deathMeta)
		auditlog.Log(tx, auditlog.Entry{
			PlayerID:       &playerID,
			Action:         action,
			EntityType:     "gu_instance",
			EntityID:       s.ID,
			RequestSummary: content,
			Metadata:       deathMeta,
			Success:        true,
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
	if err := tx.Commit(); err != nil {
		return err
	}
	if PlayerEventPublisher != nil {
		PlayerEventPublisher(parsedPID, "player.updated", "player", "gu_inventory", "combat_stats")
	}
	return nil
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

// IsHungerTransientError reports whether err is a transient Postgres condition
// (statement timeout 57014, deadlock 40P01, lock timeout 55P03) that the
// hunger path or its transaction owner may retry.
func IsHungerTransientError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "57014", "40P01", "55P03":
			return true
		}
	}
	return false
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
