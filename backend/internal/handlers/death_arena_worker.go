package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var (
	deathArenaWorkerStarted bool
	deathArenaWorkerMutex   sync.Mutex
)

// StartDeathArenaWorker starts the background worker for death arena tournaments
func StartDeathArenaWorker(ctx context.Context) {
	deathArenaWorkerMutex.Lock()
	if deathArenaWorkerStarted {
		deathArenaWorkerMutex.Unlock()
		return
	}
	deathArenaWorkerStarted = true
	deathArenaWorkerMutex.Unlock()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	logger.S.Info("Death Arena worker started (checking every 30s)")

	for {
		select {
		case <-ctx.Done():
			logger.S.Info("Death Arena worker stopped")
			return
		case <-ticker.C:
			if err := runDeathArenaTick(); err != nil {
				logger.S.Errorf("Death Arena worker tick error: %v", err)
			}
		}
	}
}

// runDeathArenaTick processes one tick of the death arena worker
func runDeathArenaTick() error {
	now := time.Now()
	vnLoc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	nowVN := now.In(vnLoc)

	// Check if it's Friday (5 = Friday in Go's time.Weekday)
	if nowVN.Weekday() != time.Friday {
		return nil
	}

	hour := nowVN.Hour()
	minute := nowVN.Minute()

	// Tournament schedule (all times in VN timezone):
	// 20:00 - Create tournament (REGISTRATION)
	// 21:00 - Close registration, generate bracket (LOCKED)
	// 21:01+ - Process matches (IN_PROGRESS)

	switch {
	case hour == 20 && minute == 0:
		// Create new tournament for registration
		return createTournamentIfNeeded()
	case hour == 21 && minute == 0:
		// Close registration and start tournament
		return closeRegistrationAndStart()
	case hour >= 21:
		// Process pending matches (if any)
		return processActiveTournament()
	}

	return nil
}

// createTournamentIfNeeded creates a new tournament if none exists
func createTournamentIfNeeded() error {
	// Check if there's already a non-FINISHED tournament
	var count int
	err := db.DB.QueryRow(`
		SELECT COUNT(*) FROM death_arena_tournaments 
		WHERE status != 'FINISHED'
	`).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // Tournament already exists
	}

	// Check if we already created one for today (dedup)
	var todayCount int
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM death_arena_tournaments 
		WHERE DATE(scheduled_at AT TIME ZONE 'Asia/Ho_Chi_Minh') = CURRENT_DATE
	`).Scan(&todayCount)
	if err != nil {
		return err
	}

	if todayCount > 0 {
		return nil // Already created today
	}

	// Create new tournament
	vnLoc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	now := time.Now().In(vnLoc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, vnLoc)
	regStart := today.Add(20 * time.Hour)  // 20:00
	regEnd := today.Add(21 * time.Hour)    // 21:00

	_, err = db.DB.Exec(`
		INSERT INTO death_arena_tournaments (status, scheduled_at, registration_started_at, registration_closed_at)
		VALUES ('REGISTRATION', $1, $2, $3)
	`, today.Add(21*time.Hour), regStart, regEnd)

	if err != nil {
		return err
	}

	logger.S.Info("Death Arena: New tournament created for registration")

	// Broadcast to world chat
	_, _ = db.DB.Exec(`
		INSERT INTO messages (sender_id, content, channel)
		VALUES ('00000000-0000-0000-0000-000000000000', $1, 'world')
	`, "[Đấu Trường Sinh Tử] Giải đấu Sinh Tử hàng tuần đã mở đăng ký! Thời gian đăng ký: 20:00-21:00 Thứ 6. Gõ /death-arena để tham gia!")

	return nil
}

// closeRegistrationAndStart closes registration and starts the tournament
func closeRegistrationAndStart() error {
	// Find REGISTRATION tournament
	var tournamentID string
	err := db.DB.QueryRow(`
		SELECT id FROM death_arena_tournaments 
		WHERE status = 'REGISTRATION'
		ORDER BY scheduled_at DESC
		LIMIT 1
	`).Scan(&tournamentID)
	if err != nil {
		return nil // No tournament to start
	}

	// Atomically transition status from REGISTRATION to LOCKED to prevent duplicate starts during minute 00
	res, err := db.DB.Exec(`
		UPDATE death_arena_tournaments 
		SET status = 'LOCKED'
		WHERE id = $1 AND status = 'REGISTRATION'
	`, tournamentID)
	if err != nil {
		return err
	}
	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return nil // Already processed by another tick
	}

	// Generate bracket
	if err := GenerateBracket(tournamentID); err != nil {
		logger.S.Errorf("Death Arena: Failed to generate bracket: %v", err)
		return err
	}

	// Start tournament
	if err := StartTournament(tournamentID); err != nil {
		logger.S.Errorf("Death Arena: Failed to start tournament: %v", err)
		return err
	}

	logger.S.Infof("Death Arena: Tournament %s started", tournamentID)

	// Broadcast
	_, _ = db.DB.Exec(`
		INSERT INTO messages (sender_id, content, channel)
		VALUES ('00000000-0000-0000-0000-000000000000', $1, 'world')
	`, "[Đấu Trường Sinh Tử] Giải đấu đã bắt đầu! Các trận chiến đang diễn ra...")

	return nil
}

// processActiveTournament processes any active tournament's pending matches
func processActiveTournament() error {
	if IsProcessingMatches() {
		return nil // Already actively processing matches
	}

	// Find IN_PROGRESS tournament
	var tournamentID string
	err := db.DB.QueryRow(`
		SELECT id FROM death_arena_tournaments 
		WHERE status = 'IN_PROGRESS'
		ORDER BY scheduled_at DESC
		LIMIT 1
	`).Scan(&tournamentID)
	if err != nil {
		return nil // No active tournament
	}

	// Process tournament matches, advance rounds, or finalize tournament
	if !IsProcessingMatches() {
		go processPendingMatches(tournamentID)
	}

	return nil
}