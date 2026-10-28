package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var (
	martialWorkerStarted bool
	martialWorkerMutex   sync.Mutex
	martialTimer         *time.Timer
	martialTimerMutex    sync.Mutex
)

// StartMartialArenaWorker starts the event-driven scheduler for the weekly
// martial arts tournament (every Tuesday 12:00 VN). No registration phase —
// players are auto-snapshotted at event time.
func StartMartialArenaWorker(ctx context.Context) {
	martialWorkerMutex.Lock()
	if martialWorkerStarted {
		martialWorkerMutex.Unlock()
		return
	}
	martialWorkerStarted = true
	martialWorkerMutex.Unlock()

	logger.S.Info("Started Martial Arena worker (weekly Tuesday 12:00 VN)")

	recoverMartialArenaOnStartup()
	scheduleNextMartialArenaEvent(ctx)

	go func() {
		<-ctx.Done()
		martialTimerMutex.Lock()
		if martialTimer != nil {
			martialTimer.Stop()
		}
		martialTimerMutex.Unlock()
		logger.S.Info("Martial Arena worker stopped")
	}()
}

// recoverMartialArenaOnStartup resumes/finishes every stuck IN_PROGRESS bracket and
// starts every stuck LOCKED one after a restart (multiple rank brackets run concurrently).
func recoverMartialArenaOnStartup() {
	rows, err := db.DB.Query(`SELECT id FROM martial_tournaments WHERE status = 'IN_PROGRESS' ORDER BY scheduled_at DESC`)
	if err == nil {
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				reconcileMartialTournament(id)
			}
		}
		rows.Close()
	}

	rows2, err := db.DB.Query(`SELECT id FROM martial_tournaments WHERE status = 'LOCKED' ORDER BY scheduled_at DESC`)
	if err == nil {
		for rows2.Next() {
			var id string
			if rows2.Scan(&id) == nil {
				logger.S.Infof("Martial Arena: starting stuck LOCKED tournament %s", id)
				if err := StartMartialTournament(id); err != nil {
					logger.S.Errorf("Martial Arena: failed to start LOCKED tournament %s: %v", id, err)
				}
			}
		}
		rows2.Close()
	}
}

func reconcileMartialTournament(tournamentID string) {
	_, _ = db.DB.Exec(`UPDATE martial_matches m SET status='PENDING'
		WHERE m.tournament_id=$1 AND m.status='IN_PROGRESS' AND NOT EXISTS (
			SELECT 1 FROM realtime_combat_sessions s WHERE s.mode='tournament' AND s.status='active'
			AND s.payload->>'tournament_id'=$1 AND s.payload->>'match_id'=m.id::text)`, tournamentID)

	var pending int
	_ = db.DB.QueryRow(`SELECT COUNT(*) FROM martial_matches WHERE tournament_id = $1 AND status = 'PENDING'`, tournamentID).Scan(&pending)
	if pending > 0 {
		logger.S.Infof("Martial Arena: resumed tournament %s (%d pending matches)", tournamentID, pending)
		go processPendingMartialMatches(tournamentID)
		return
	}

	advanceMartialRound(tournamentID)
	var status string
	_ = db.DB.QueryRow(`SELECT status FROM martial_tournaments WHERE id = $1`, tournamentID).Scan(&status)
	if status == "IN_PROGRESS" {
		go processPendingMartialMatches(tournamentID)
	}
}

func scheduleNextMartialArenaEvent(ctx context.Context) {
	martialTimerMutex.Lock()
	defer martialTimerMutex.Unlock()

	if martialTimer != nil {
		martialTimer.Stop()
	}

	target := nextMartialArenaTarget()
	delay := time.Until(target)
	if delay <= 0 {
		go handleMartialArenaEvent(ctx)
		return
	}

	logger.S.Infof("Martial Arena: next event at %s (in %v)", target.Format("2006-01-02 15:04:05 MST"), delay.Round(time.Second))
	martialTimer = time.AfterFunc(delay, func() {
		handleMartialArenaEvent(ctx)
	})
}

func handleMartialArenaEvent(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	runMartialArenaEvent()
	scheduleNextMartialArenaEvent(ctx)
}

// nextMartialArenaTarget returns the next Tuesday 12:00 in VN time.
func nextMartialArenaTarget() time.Time {
	vnLoc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	now := time.Now().In(vnLoc)

	daysUntil := (int(time.Tuesday) - int(now.Weekday()) + 7) % 7
	if daysUntil == 0 && now.Hour() >= 12 {
		daysUntil = 7
	}

	return time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, vnLoc).AddDate(0, 0, daysUntil)
}
