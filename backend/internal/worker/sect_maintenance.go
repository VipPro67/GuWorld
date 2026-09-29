package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	maintenanceFeePerLevel = int64(5000)
	maxSectLevel           = 1
)

func StartSectMaintenanceWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Sect Maintenance Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Sect Maintenance Worker stopped")
				return
			case <-ticker.C:
				processSectMaintenanceTick(ctx, db)
			}
		}
	}()
}

func processSectMaintenanceTick(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, level, funds, last_maintenance_at
		FROM sects
	`)
	if err != nil {
		logger.S.Errorf("SectMaintenanceWorker: failed to query sects: %v", err)
		return
	}
	defer rows.Close()

	type paidUpdate struct {
		sectID   string
		newFunds int64
	}
	var paidUpdates []paidUpdate

	type downgradeUpdate struct {
		sectID   string
		oldLevel int
		newLevel int
		funds    int64
		level    int
	}
	var downgrades []downgradeUpdate

	for rows.Next() {
		var sectID string
		var level int
		var funds int64
		var lastMaintenance sql.NullTime
		if err := rows.Scan(&sectID, &level, &funds, &lastMaintenance); err != nil {
			logger.S.Errorf("SectMaintenanceWorker: failed to scan sect: %v", err)
			continue
		}

		if lastMaintenance.Valid && lastMaintenance.Time.After(time.Now().Add(-24*time.Hour)) {
			continue
		}

		fee := calculateMaintenanceFee(int64(level))

		if fee == 0 {
			// No fee needed - just update timestamp
			_, _ = db.ExecContext(ctx, `UPDATE sects SET last_maintenance_at = now() WHERE id = $1`, sectID)
			continue
		}

		if funds >= fee {
			newFunds := funds - fee
			paidUpdates = append(paidUpdates, paidUpdate{sectID: sectID, newFunds: newFunds})
		} else {
			newLevel := level - 1
			if newLevel < maxSectLevel {
				newLevel = maxSectLevel
			}
			downgrades = append(downgrades, downgradeUpdate{sectID: sectID, oldLevel: level, newLevel: newLevel, funds: funds, level: level})
		}
	}

	// Batch UPDATE for paid maintenance
	if len(paidUpdates) > 0 {
		ids := make([]string, len(paidUpdates))
		fundsArr := make([]int64, len(paidUpdates))
		now := time.Now()
		timestamps := make([]time.Time, len(paidUpdates))
		for i, u := range paidUpdates {
			ids[i] = u.sectID
			fundsArr[i] = u.newFunds
			timestamps[i] = now
		}

		_, err := db.ExecContext(ctx, `
			WITH updates(id, funds, last_maintenance_at) AS (
				SELECT unnest($1::text[]), unnest($2::int8[]), unnest($3::timestamptz[])
			)
			UPDATE sects s SET funds = u.funds, last_maintenance_at = u.last_maintenance_at
			FROM updates u WHERE s.id = u.id
		`, pq.Array(ids), pq.Array(fundsArr), pq.Array(timestamps))
		if err != nil {
			logger.S.Errorf("SectMaintenanceWorker: failed batch update for paid maintenance: %v", err)
		} else {
			logger.S.Infof("SectMaintenanceWorker: %d sect(s) paid maintenance", len(paidUpdates))
		}
	}

	// Batch UPDATE for downgrades
	if len(downgrades) > 0 {
		ids := make([]string, len(downgrades))
		levels := make([]int, len(downgrades))
		now := time.Now()
		timestamps := make([]time.Time, len(downgrades))
		for i, u := range downgrades {
			ids[i] = u.sectID
			levels[i] = u.newLevel
			timestamps[i] = now
		}

		_, err := db.ExecContext(ctx, `
			WITH updates(id, level, last_maintenance_at) AS (
				SELECT unnest($1::text[]), unnest($2::int[]), unnest($3::timestamptz[])
			)
			UPDATE sects s SET level = u.level, last_maintenance_at = u.last_maintenance_at
			FROM updates u WHERE s.id = u.id
		`, pq.Array(ids), pq.Array(levels), pq.Array(timestamps))
		if err != nil {
			logger.S.Errorf("SectMaintenanceWorker: failed batch update for downgrades: %v", err)
		} else {
			logger.S.Infof("SectMaintenanceWorker: %d sect(s) downgraded", len(downgrades))
			for _, u := range downgrades {
				notifySectLeader(ctx, db, u.sectID, u.oldLevel, u.newLevel, u.funds, maintenanceFeePerLevel*int64(u.level))
			}
		}
	}
}

func calculateMaintenanceFee(level int64) int64 {
	return level * maintenanceFeePerLevel
}

func updateLastMaintenance(ctx context.Context, db *sql.DB, sectID string) error {
	_, err := db.ExecContext(ctx, `UPDATE sects SET last_maintenance_at = now() WHERE id = $1`, sectID)
	return err
}

func notifySectLeader(ctx context.Context, db *sql.DB, sectID string, oldLevel, newLevel int, funds, fee int64) {
	var leaderID sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT player_id FROM sect_members
		WHERE sect_id = $1 AND member_rank = 'CHUONG_MON'
		LIMIT 1
	`).Scan(&leaderID)
	if err != nil {
		logger.S.Debugf("SectMaintenanceWorker: no leader found for sect %s: %v", sectID, err)
		return
	}

	if !leaderID.Valid {
		return
	}

	message := "Tông môn bị trừ cấp do không đủ quỹ duy trì!"
	_, err = db.ExecContext(ctx, `
		INSERT INTO system_messages (player_id, type, title, content)
		VALUES ($1, 'SECT_MAINTENANCE', 'CẢNH BÁO TỔNG MÔN', $2)
	`, leaderID.String, message)
	if err != nil {
		logger.S.Errorf("SectMaintenanceWorker: failed to notify sect leader %s: %v", leaderID.String, err)
	}
}
