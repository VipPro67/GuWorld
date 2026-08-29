package worker

import (
	"context"
	"database/sql"
	"time"

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
				processSectMaintenanceTick(db)
			}
		}
	}()
}

func processSectMaintenanceTick(db *sql.DB) {
	ctx := context.Background()

	rows, err := db.QueryContext(ctx, `
		SELECT id, level, funds, last_maintenance_at
		FROM sects
	`)
	if err != nil {
		logger.S.Errorf("SectMaintenanceWorker: failed to query sects: %v", err)
		return
	}
	defer rows.Close()

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

		fee := calculateMaintenanceFee(level)

		if fee == 0 {
			if err := updateLastMaintenance(ctx, db, sectID); err != nil {
				logger.S.Errorf("SectMaintenanceWorker: failed to update last_maintenance for sect %s: %v", sectID, err)
			}
			continue
		}

		if funds >= fee {
			newFunds := funds - fee
			_, err := db.ExecContext(ctx, `UPDATE sects SET funds = $1, last_maintenance_at = now() WHERE id = $2`, newFunds, sectID)
			if err != nil {
				logger.S.Errorf("SectMaintenanceWorker: failed to deduct maintenance fee for sect %s: %v", sectID, err)
			} else {
				logger.S.Infof("SectMaintenanceWorker: sect %s paid maintenance fee %d, remaining funds %d", sectID, fee, newFunds)
			}
		} else {
			newLevel := level - 1
			if newLevel < maxSectLevel {
				newLevel = maxSectLevel
			}

			_, err := db.ExecContext(ctx, `UPDATE sects SET level = $1, last_maintenance_at = now() WHERE id = $2`, newLevel, sectID)
			if err != nil {
				logger.S.Errorf("SectMaintenanceWorker: failed to downgrade sect %s: %v", sectID, err)
			} else {
				logger.S.Warnf("SectMaintenanceWorker: sect %s downgraded from level %d to %d due to insufficient funds (had %d, needed %d)", sectID, level, newLevel, funds, fee)
				notifySectLeader(ctx, db, sectID, level, newLevel, funds, fee)
			}
		}
	}

	if err := rows.Err(); err != nil {
		logger.S.Errorf("SectMaintenanceWorker: rows error: %v", err)
	}
}

func calculateMaintenanceFee(level int) int64 {
	return int64(level) * maintenanceFeePerLevel
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
	`, sectID).Scan(&leaderID)
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