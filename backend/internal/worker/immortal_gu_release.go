package worker

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

var (
	immortalGuReleaseStarted bool
	immortalGuReleaseMutex   sync.Mutex
)

const (
	systemSenderUUID    = "11111111-1111-1111-1111-111111111111"
	globalBroadcastUUID = "00000000-0000-0000-0000-000000000000"
)

// StartImmortalGuReleaseWorker begins the periodic scan to release starved or abandoned Immortal Gus.
func StartImmortalGuReleaseWorker(ctx context.Context, dbConn *sql.DB, interval time.Duration) {
	immortalGuReleaseMutex.Lock()
	if immortalGuReleaseStarted {
		immortalGuReleaseMutex.Unlock()
		return
	}
	immortalGuReleaseStarted = true
	immortalGuReleaseMutex.Unlock()

	RunEvery(ctx, interval, "Immortal Gu Release Worker", func(ctx context.Context) {
		ProcessImmortalGuReleaseTick(ctx, dbConn)
	})
}

// ProcessImmortalGuReleaseTick runs both player and sect vault immortal Gu release evaluations.
func ProcessImmortalGuReleaseTick(ctx context.Context, dbConn *sql.DB) {
	if err := releaseStarvedOrAbandonedPlayerImmortalGus(ctx, dbConn); err != nil {
		logger.S.Errorf("ImmortalGuReleaseWorker: failed releasing player immortal gus: %v", err)
	}
	if err := releaseAbandonedSectImmortalGus(ctx, dbConn); err != nil {
		logger.S.Errorf("ImmortalGuReleaseWorker: failed releasing sect immortal gus: %v", err)
	}
}

type releasedPlayerGuInfo struct {
	InstanceID string
	PlayerID   uuid.UUID
	GuID       string
	Rank       int
	GuName     string
	Username   string
	PlayerRank int
	Reason     string
}

// releaseStarvedOrAbandonedPlayerImmortalGus finds Immortal Gus (Rank >= 6) in player_gu that
// are starved to death, or whose mortal owners have abandoned them / died of old age, and releases them.
func releaseStarvedOrAbandonedPlayerImmortalGus(ctx context.Context, dbConn *sql.DB) error {
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Query candidate immortal gus that need to be released
	rows, err := tx.QueryContext(ctx, `
		SELECT pg.id, pg.player_id, pg.gu_id, COALESCE(pg.gu_rank, pg.rank, 6),
		       COALESCE(gd.name, pg.gu_id), p.username, p.rank,
		       CASE
		           WHEN pg.hunger <= 0 OR pg.state = 'DEAD' THEN 'HUNGER_DEPLETED'
		           WHEN p.rank < 6 AND (COALESCE(p.last_login_time, p.last_activity, p.created_at) < NOW() - INTERVAL '14 days') THEN 'MORTAL_INACTIVE_14D'
		           WHEN p.rank < 6 AND (p.lifespan <= 0 OR (p.lifespan - (EXTRACT(EPOCH FROM (NOW() - COALESCE(p.last_mana_update, p.created_at, NOW()))) / 86400.0) <= 0)) THEN 'MORTAL_LIFESPAN_EXPIRED'
		           ELSE 'CALCULATED_STARVATION'
		       END AS release_reason
		FROM player_gu pg
		JOIN players p ON p.id = pg.player_id
		LEFT JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = COALESCE(pg.gu_rank, pg.rank, 6)
		WHERE (pg.gu_rank >= 6 OR pg.rank >= 6)
		  AND (
		    -- 1. Explicitly dead or zero hunger
		    pg.hunger <= 0 OR pg.state = 'DEAD'
		    -- 2. Hunger decayed to <= 0 based on elapsed time since last_hunger_tick_at
		    OR (
		      pg.hunger - (
		        FLOOR(EXTRACT(EPOCH FROM (NOW() - COALESCE(p.last_hunger_tick_at, p.created_at, NOW()))) / 10800.0)
		        * COALESCE(gd.hunger_rate, 1)
		        * (CASE WHEN pg.state = 'SLEEPING' THEN 0.25 WHEN pg.state = 'SEALED' THEN 0.1 ELSE 1.0 END)
		      ) <= 0
		      AND (
		        p.rank < 6
		        OR COALESCE(p.last_login_time, p.last_activity, p.created_at) < NOW() - INTERVAL '30 days'
		      )
		    )
		    -- 3. Mortal player offline > 14 days
		    OR (
		      p.rank < 6
		      AND COALESCE(p.last_login_time, p.last_activity, p.created_at) < NOW() - INTERVAL '14 days'
		    )
		    -- 4. Mortal player whose lifespan is expired (dead state) and offline > 7 days
		    OR (
		      p.rank < 6
		      AND (
		        p.lifespan <= 0
		        OR (p.lifespan - (EXTRACT(EPOCH FROM (NOW() - COALESCE(p.last_mana_update, p.created_at, NOW()))) / 86400.0) <= 0)
		      )
		      AND COALESCE(p.last_login_time, p.last_activity, p.created_at) < NOW() - INTERVAL '7 days'
		    )
		  )
		FOR UPDATE OF pg
		LIMIT 100
	`)
	if err != nil {
		return fmt.Errorf("querying candidates: %w", err)
	}
	defer rows.Close()

	var toRelease []releasedPlayerGuInfo
	var instanceIDs []string
	playerIDMap := make(map[uuid.UUID]bool)

	for rows.Next() {
		var item releasedPlayerGuInfo
		if err := rows.Scan(
			&item.InstanceID, &item.PlayerID, &item.GuID, &item.Rank,
			&item.GuName, &item.Username, &item.PlayerRank, &item.Reason,
		); err != nil {
			return fmt.Errorf("scanning candidate: %w", err)
		}
		toRelease = append(toRelease, item)
		instanceIDs = append(instanceIDs, item.InstanceID)
		playerIDMap[item.PlayerID] = true
	}
	rows.Close()

	if len(toRelease) == 0 {
		return nil
	}

	var playerIDs []uuid.UUID
	for pid := range playerIDMap {
		playerIDs = append(playerIDs, pid)
	}

	// 1. Clean up killer move reservations
	if _, err := tx.ExecContext(ctx, `
		UPDATE player_killer_moves AS move SET is_equipped = false, updated_at = NOW()
		WHERE move.is_equipped = true AND EXISTS (
			SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(move.components) = 'array' THEN move.components ELSE '[]'::jsonb END) AS component(value)
			WHERE component.value->>'instance_id' = ANY($1::text[])
		)
	`, pq.Array(instanceIDs)); err != nil {
		return fmt.Errorf("retiring player killer moves: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_killer_move_gu_reservations WHERE gu_instance_id = ANY($1::text[])`, pq.Array(instanceIDs)); err != nil {
		return fmt.Errorf("deleting killer move reservations: %w", err)
	}

	// 2. Clean up gu house reservations
	if _, err := tx.ExecContext(ctx, `
		UPDATE player_gu_houses AS house SET is_equipped = false, updated_at = NOW()
		WHERE house.is_equipped = true AND EXISTS (
			SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(house.core_gu_instances) = 'array' THEN house.core_gu_instances ELSE '[]'::jsonb END) AS component(value)
			WHERE component.value #>> '{}' = ANY($1::text[])
		)
	`, pq.Array(instanceIDs)); err != nil {
		return fmt.Errorf("retiring player gu houses: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_gu_house_core_reservations WHERE gu_instance_id = ANY($1::text[])`, pq.Array(instanceIDs)); err != nil {
		return fmt.Errorf("deleting gu house core reservations: %w", err)
	}

	// 3. Clean up teleport array reservations
	if _, err := tx.ExecContext(ctx, `DELETE FROM teleport_array_gu_reservations WHERE gu_instance_id = ANY($1::text[])`, pq.Array(instanceIDs)); err != nil {
		return fmt.Errorf("deleting teleport reservations: %w", err)
	}

	// 4. Prune from equipped_gus
	if _, err := tx.ExecContext(ctx, `
		UPDATE players p
		SET equipped_gus = (
			SELECT COALESCE(jsonb_agg(elem), '[]'::jsonb)
			FROM jsonb_array_elements_text(CASE WHEN jsonb_typeof(p.equipped_gus) = 'array' THEN p.equipped_gus ELSE '[]'::jsonb END) AS elem
			WHERE elem NOT IN (SELECT unnest($1::text[]))
		)
		WHERE p.id = ANY($2::uuid[])
	`, pq.Array(instanceIDs), pq.Array(playerIDs)); err != nil {
		return fmt.Errorf("pruning equipped_gus: %w", err)
	}

	// 5. Delete from player_gu (Releasing the Unique constraint and CheckImmortalGuExists lock)
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_gu WHERE id = ANY($1::text[])`, pq.Array(instanceIDs)); err != nil {
		return fmt.Errorf("deleting released player_gu rows: %w", err)
	}

	// 6. Post-delete notifications, history, and chat announcements
	for _, item := range toRelease {
		meta := map[string]interface{}{
			"instance_id": item.InstanceID,
			"gu_id":       item.GuID,
			"gu_rank":     item.Rank,
			"reason":      item.Reason,
			"owner_rank":  item.PlayerRank,
			"released_at": time.Now().UTC().Format(time.RFC3339),
		}

		_ = db.LogGuHistory(tx, item.InstanceID, item.GuID, item.Rank, "IMMORTAL_ESCAPED", &item.PlayerID, nil, "Tiên Cổ phá không bay về thiên địa do không được nuôi dưỡng", meta)

		auditlog.Log(tx, auditlog.Entry{
			PlayerID:         &item.PlayerID,
			AffectedPlayerID: &item.PlayerID,
			Action:           "gu.immortal_escaped_hunger",
			RequestID:        "immortal-escape:" + item.InstanceID,
			EntityType:       "gu_instance",
			EntityID:         item.InstanceID,
			RequestSummary:   fmt.Sprintf("Tiên Cổ %s (Chuyển %d) phá vỡ phong ấn thoát khốn do chủ nhân không chu cấp nuôi dưỡng", item.GuName, item.Rank),
			Metadata:         meta,
			Success:          true,
		})

		// Notify player
		notifContent := fmt.Sprintf("Do không được cung phụng nuôi dưỡng kịp thời (lý do: %s), Tiên Cổ [%s] (Chuyển %d) đã cắn nát phong ấn, phá không bay về thiên địa!", item.Reason, item.GuName, item.Rank)
		notify.SendBestEffort(tx, item.PlayerID, "IMMORTAL_GU_ESCAPED", "Tiên Cổ Thoát Khốn!", notifContent)

		// Private system message
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO messages (sender_id, receiver_id, content, channel)
			VALUES ($1, $2, $3, 'private')
		`, systemSenderUUID, item.PlayerID, notifContent)

		// World announcement
		worldContent := fmt.Sprintf("【THIÊN ĐỊA BIẾN ĐỘNG】 Tiên Cổ [%s] (Chuyển %d) do mất đi chủ nhân chu cấp nuôi dưỡng, đã phá vỡ gông cùm thoát khốn, quy hồi thiên địa! Thiên hạ Cổ sư nay đã có thể tìm kiếm hoặc luyện chế lại!", item.GuName, item.Rank)
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO messages (sender_id, receiver_id, content, channel)
			VALUES ($1, $2, $3, 'world')
		`, systemSenderUUID, globalBroadcastUUID, worldContent)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing release transaction: %w", err)
	}

	_ = cache.PublishEventQuick("chat:world", map[string]any{"type": "new_message"})
	logger.S.Infof("ImmortalGuReleaseWorker: successfully released %d immortal Gus from players", len(toRelease))
	return nil
}

type releasedSectGuInfo struct {
	ID       string
	SectID   string
	GuID     string
	Rank     int
	Quantity int64
	SectName string
	GuName   string
}

// releaseAbandonedSectImmortalGus releases Immortal Gus stuck in sect vaults whose sects are abandoned
// (no members, or all members offline for > 14 days).
func releaseAbandonedSectImmortalGus(ctx context.Context, dbConn *sql.DB) error {
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT vg.id, vg.sect_id, vg.gu_id, vg.rank, vg.quantity,
		       s.name AS sect_name, COALESCE(gd.name, vg.gu_id) AS gu_name
		FROM sect_vault_gu vg
		JOIN sects s ON s.id = vg.sect_id
		LEFT JOIN gu_dictionary gd ON gd.id = vg.gu_id AND gd.rank = vg.rank
		WHERE vg.rank >= 6 AND vg.quantity > 0
		  AND (
		    -- 1. Sect has no members at all
		    NOT EXISTS (
		      SELECT 1 FROM sect_members sm WHERE sm.sect_id = s.id
		    )
		    -- 2. OR all members of the sect have been offline / inactive for > 14 days
		    OR NOT EXISTS (
		      SELECT 1 FROM sect_members sm
		      JOIN players p ON p.id = sm.player_id
		      WHERE sm.sect_id = s.id
		        AND COALESCE(p.last_login_time, p.last_activity, p.created_at) > NOW() - INTERVAL '14 days'
		    )
		  )
		FOR UPDATE OF vg
		LIMIT 50
	`)
	if err != nil {
		return fmt.Errorf("querying abandoned sect immortal gus: %w", err)
	}
	defer rows.Close()

	var toRelease []releasedSectGuInfo
	var vaultIDs []string

	for rows.Next() {
		var item releasedSectGuInfo
		if err := rows.Scan(&item.ID, &item.SectID, &item.GuID, &item.Rank, &item.Quantity, &item.SectName, &item.GuName); err != nil {
			return fmt.Errorf("scanning sect vault candidate: %w", err)
		}
		toRelease = append(toRelease, item)
		vaultIDs = append(vaultIDs, item.ID)
	}
	rows.Close()

	if len(toRelease) == 0 {
		return nil
	}

	// Delete from sect_vault_gu to release the unique constraint and CheckImmortalGuExists lock
	if _, err := tx.ExecContext(ctx, `DELETE FROM sect_vault_gu WHERE id = ANY($1::text[])`, pq.Array(vaultIDs)); err != nil {
		return fmt.Errorf("deleting released sect_vault_gu rows: %w", err)
	}

	for _, item := range toRelease {
		meta := map[string]interface{}{
			"sect_id":     item.SectID,
			"sect_name":   item.SectName,
			"gu_id":       item.GuID,
			"gu_rank":     item.Rank,
			"quantity":    item.Quantity,
			"released_at": time.Now().UTC().Format(time.RFC3339),
		}

		_ = db.LogGuHistory(tx, item.ID, item.GuID, item.Rank, "IMMORTAL_VAULT_ESCAPED", nil, nil, fmt.Sprintf("Tiên Cổ bảo khố tông môn %s phá không bay về thiên địa do sơn môn hoang phế", item.SectName), meta)

		auditlog.Log(tx, auditlog.Entry{
			Action:         "sect.immortal_gu_escaped_abandoned",
			RequestID:      "sect-immortal-escape:" + item.ID,
			EntityType:     "sect_vault_gu",
			EntityID:       item.ID,
			RequestSummary: fmt.Sprintf("Tiên Cổ %s (Chuyển %d) thoát khỏi kho tông môn %s do sơn môn hoang phế", item.GuName, item.Rank, item.SectName),
			Metadata:       meta,
			Success:        true,
		})

		worldContent := fmt.Sprintf("【THIÊN ĐỊA BIẾN ĐỘNG】 Tông môn [%s] không còn đệ tử hương hỏa, sơn môn hoang phế đã lâu. Tiên Cổ [%s] (Chuyển %d) trấn thủ bảo khố đã phá không bay đi, quy hồi thiên địa! Thiên hạ Cổ sư nay đã có thể tìm kiếm hoặc luyện chế lại!", item.SectName, item.GuName, item.Rank)
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO messages (sender_id, receiver_id, content, channel)
			VALUES ($1, $2, $3, 'world')
		`, systemSenderUUID, globalBroadcastUUID, worldContent)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing sect release transaction: %w", err)
	}

	_ = cache.PublishEventQuick("chat:world", map[string]any{"type": "new_message"})
	logger.S.Infof("ImmortalGuReleaseWorker: successfully released %d immortal Gus from abandoned sects", len(toRelease))
	return nil
}
