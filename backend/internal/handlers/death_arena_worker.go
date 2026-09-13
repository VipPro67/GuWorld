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
	deathArenaTimer         *time.Timer
	deathArenaTimerMutex    sync.Mutex
)

// StartDeathArenaWorker starts the event-driven scheduler for death arena tournaments (Beast Betting style)
func StartDeathArenaWorker(ctx context.Context) {
	deathArenaWorkerMutex.Lock()
	if deathArenaWorkerStarted {
		deathArenaWorkerMutex.Unlock()
		return
	}
	deathArenaWorkerStarted = true
	deathArenaWorkerMutex.Unlock()

	logger.S.Info("Started Death Arena worker (event-driven timer schedule)")

	// 1. Recover state on startup (handles server restarts during tournaments)
	recoverDeathArenaOnStartup()

	// 2. Schedule next event
	scheduleNextDeathArenaEvent(ctx)

	// 3. Listen for context shutdown
	go func() {
		<-ctx.Done()
		deathArenaTimerMutex.Lock()
		if deathArenaTimer != nil {
			deathArenaTimer.Stop()
		}
		deathArenaTimerMutex.Unlock()
		logger.S.Info("Death Arena worker stopped")
	}()
}

// recoverDeathArenaOnStartup checks if any tournament was interrupted by a restart
func recoverDeathArenaOnStartup() {
	// A. Resume active IN_PROGRESS tournament
	var inProgressID string
	err := db.DB.QueryRow(`
		SELECT id FROM death_arena_tournaments 
		WHERE status = 'IN_PROGRESS' 
		ORDER BY scheduled_at DESC LIMIT 1
	`).Scan(&inProgressID)
	if err == nil && inProgressID != "" {
		logger.S.Infof("Death Arena: Recovered active tournament %s on startup", inProgressID)
		if !IsProcessingMatches() {
			go processPendingMatches(inProgressID)
		}
		return
	}

	// B. Close and start tournament if REGISTRATION is already past closed time
	_ = closeRegistrationAndStart()

	// C. If Saturday between 16:00 and 17:00 VN time, ensure registration is created
	vnLoc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	nowVN := time.Now().In(vnLoc)
	if nowVN.Weekday() == time.Saturday && nowVN.Hour() >= 16 && nowVN.Hour() < 17 {
		_ = createTournamentIfNeeded()
	}
}

// scheduleNextDeathArenaEvent calculates delay to the next target event and arms a timer
func scheduleNextDeathArenaEvent(ctx context.Context) {
	deathArenaTimerMutex.Lock()
	defer deathArenaTimerMutex.Unlock()

	if deathArenaTimer != nil {
		deathArenaTimer.Stop()
	}

	targetTime, eventType := getNextDeathArenaTarget()
	delay := time.Until(targetTime)

	if delay <= 0 {
		logger.S.Infof("Death Arena: Target event %s is due immediately", eventType)
		go handleScheduledEvent(eventType, ctx)
		return
	}

	logger.S.Infof("Death Arena: Next event (%s) scheduled at %s (in %v)",
		eventType, targetTime.Format("2006-01-02 15:04:05 MST"), delay.Round(time.Second))

	deathArenaTimer = time.AfterFunc(delay, func() {
		handleScheduledEvent(eventType, ctx)
	})
}

// handleScheduledEvent executes the scheduled event and schedules the subsequent event
func handleScheduledEvent(eventType string, ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	logger.S.Infof("Death Arena: Executing scheduled event %s", eventType)

	switch eventType {
	case "OPEN_REGISTRATION":
		if err := createTournamentIfNeeded(); err != nil {
			logger.S.Errorf("Death Arena: Failed to create tournament: %v", err)
		}
	case "START_TOURNAMENT":
		if err := closeRegistrationAndStart(); err != nil {
			logger.S.Errorf("Death Arena: Failed to start tournament: %v", err)
		}
	}

	// Schedule the next event
	scheduleNextDeathArenaEvent(ctx)
}

// getNextDeathArenaTarget calculates the next milestone time and event type
func getNextDeathArenaTarget() (time.Time, string) {
	vnLoc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	nowVN := time.Now().In(vnLoc)

	// 1. If there's an open REGISTRATION tournament, target its registration_closed_at
	var regClosedAt time.Time
	err := db.DB.QueryRow(`
		SELECT registration_closed_at FROM death_arena_tournaments 
		WHERE status = 'REGISTRATION'
		ORDER BY scheduled_at DESC LIMIT 1
	`).Scan(&regClosedAt)
	if err == nil {
		return regClosedAt.In(vnLoc), "START_TOURNAMENT"
	}

	// 2. Otherwise calculate next Saturday 16:00 (or today 16:00 / 17:00 if Saturday)
	daysUntilSaturday := (int(time.Saturday) - int(nowVN.Weekday()) + 7) % 7

	if daysUntilSaturday == 0 {
		// Today is Saturday
		if nowVN.Hour() < 16 {
			// Before 16:00 -> Open registration at 16:00 today
			return time.Date(nowVN.Year(), nowVN.Month(), nowVN.Day(), 16, 0, 0, 0, vnLoc), "OPEN_REGISTRATION"
		} else if nowVN.Hour() == 16 {
			// Between 16:00 and 17:00 -> Start tournament at 17:00 today
			return time.Date(nowVN.Year(), nowVN.Month(), nowVN.Day(), 17, 0, 0, 0, vnLoc), "START_TOURNAMENT"
		} else {
			// After 17:00 -> Next Saturday at 16:00
			return time.Date(nowVN.Year(), nowVN.Month(), nowVN.Day()+7, 16, 0, 0, 0, vnLoc), "OPEN_REGISTRATION"
		}
	}

	// Another day of the week -> target upcoming Saturday at 16:00
	return time.Date(nowVN.Year(), nowVN.Month(), nowVN.Day()+daysUntilSaturday, 16, 0, 0, 0, vnLoc), "OPEN_REGISTRATION"
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
	regStart := today.Add(16 * time.Hour) // 16:00 (Chiều Thứ 7)
	regEnd := today.Add(17 * time.Hour)   // 17:00 (Chiều 5h Thứ 7 khai chiến)

	_, err = db.DB.Exec(`
		INSERT INTO death_arena_tournaments (status, scheduled_at, registration_started_at, registration_closed_at)
		VALUES ('REGISTRATION', $1, $2, $3)
	`, today.Add(17*time.Hour), regStart, regEnd)

	if err != nil {
		return err
	}

	logger.S.Info("Death Arena: New tournament created for registration")

	// Broadcast to world chat
	_, _ = db.DB.Exec(`
		INSERT INTO messages (sender_id, content, channel)
		VALUES ('00000000-0000-0000-0000-000000000000', $1, 'world')
	`, "[Đấu Trường Sinh Tử] Giải đấu Sinh Tử hàng tuần đã mở đăng ký! Thời gian đăng ký: 16:00-17:00 Thứ 7 (Khai chiến lúc 17:00 chiều Thứ 7). Gõ /death-arena để tham gia!")

	return nil
}

// closeRegistrationAndStart closes registration and starts the tournament
func closeRegistrationAndStart() error {
	// Find REGISTRATION tournament
	var tournamentID string
	err := db.DB.QueryRow(`
		SELECT id FROM death_arena_tournaments 
		WHERE status = 'REGISTRATION' AND registration_closed_at <= NOW()
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

	// Check if tournament was already finalized (e.g. 0 or 1 team registered)
	var currentStatus string
	_ = db.DB.QueryRow(`SELECT status FROM death_arena_tournaments WHERE id = $1`, tournamentID).Scan(&currentStatus)
	if currentStatus == "FINISHED" {
		logger.S.Infof("Death Arena: Tournament %s auto-finished during bracket generation (not enough teams)", tournamentID)
		return nil
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
