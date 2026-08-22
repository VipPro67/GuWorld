package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type BeastTemplate struct {
	Name  string  `json:"name"`
	MaxHP float64 `json:"max_hp"`
	Atk   float64 `json:"atk"`
}

var beastTemplates = []BeastTemplate{
	{"Hỏa Lân Sư", 1500, 120},
	{"Băng Hồn Lang", 1200, 150},
	{"Huyết Giáp Tê", 2000, 90},
	{"U Minh Hổ", 1400, 130},
	{"Lôi Vân Ưng", 1000, 180},
	{"Kim Cương Thử", 1100, 160},
	{"Mộc Linh Xà", 1300, 140},
	{"Cửu Thiên Hạc", 1250, 145},
	{"Hám Địa Thú", 2500, 80},
}

type ActiveBeast struct {
	Name  string  `json:"name"`
	HP    float64 `json:"hp"`
	MaxHP float64 `json:"max_hp"`
	Atk   float64 `json:"atk"`
}

type BettingRound struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"` // ACTIVE, LOCKED, FINISHED
	Winner     string     `json:"winner"` // "", team_a, team_b, draw
	TeamAName  string     `json:"team_a_name"`
	TeamBName  string     `json:"team_b_name"`
	TotalBetA  float64    `json:"total_bet_a"`
	TotalBetB  float64    `json:"total_bet_b"`
	BattleLog  string     `json:"battle_log"`
	CreatedAt  time.Time  `json:"created_at"`
	LockedAt   *time.Time `json:"locked_at"`
	ResolvedAt *time.Time `json:"resolved_at"`
}

type PlaceBetRequest struct {
	Team   string  `json:"team" binding:"required"` // team_a or team_b
	Amount float64 `json:"amount" binding:"required"`
}

var (
	workerStarted bool
	workerMutex   sync.Mutex
)

// SimulateBeastFight runs a step-by-step combat simulation between two identical teams of 9 beasts
func SimulateBeastFight() (string, string) {
	teamA := make([]*ActiveBeast, len(beastTemplates))
	teamB := make([]*ActiveBeast, len(beastTemplates))
	for i, t := range beastTemplates {
		teamA[i] = &ActiveBeast{Name: t.Name, HP: t.MaxHP, MaxHP: t.MaxHP, Atk: t.Atk}
		teamB[i] = &ActiveBeast{Name: t.Name, HP: t.MaxHP, MaxHP: t.MaxHP, Atk: t.Atk}
	}

	var logs []string
	logs = append(logs, "⚔️ Trận chiến bắt đầu giữa **Tả Thanh Long** (Phe Đỏ) và **Hữu Bạch Hổ** (Phe Xanh)!")
	logs = append(logs, "Cả hai bên đều xuất quân với 9 linh thú dũng mãnh giống nhau:")
	for _, t := range beastTemplates {
		logs = append(logs, fmt.Sprintf("- %s (HP: %.0f, Atk: %.0f)", t.Name, t.MaxHP, t.Atk))
	}
	logs = append(logs, "\n--- CHIẾN ĐẤU BẮT ĐẦU ---\n")

	step := 1
	for step < 1000 {
		var aliveA []*ActiveBeast
		for _, b := range teamA {
			if b.HP > 0 {
				aliveA = append(aliveA, b)
			}
		}

		var aliveB []*ActiveBeast
		for _, b := range teamB {
			if b.HP > 0 {
				aliveB = append(aliveB, b)
			}
		}

		if len(aliveA) == 0 && len(aliveB) == 0 {
			logs = append(logs, "\n💥 Cả hai bên lưỡng bại câu thương, tất cả linh thú đã gục ngã!")
			return "draw", strings.Join(logs, "\n")
		}
		if len(aliveA) == 0 {
			logs = append(logs, "\n🏆 **Hữu Bạch Hổ** (Phe Xanh) giành chiến thắng vẻ vang!")
			return "team_b", strings.Join(logs, "\n")
		}
		if len(aliveB) == 0 {
			logs = append(logs, "\n🏆 **Tả Thanh Long** (Phe Đỏ) giành chiến thắng vẻ vang!")
			return "team_a", strings.Join(logs, "\n")
		}

		// Pick random combatants
		beastA := aliveA[rand.Intn(len(aliveA))]
		beastB := aliveB[rand.Intn(len(aliveB))]

		// Coin flip for turn order
		if rand.Float64() < 0.5 {
			// A attacks B first
			dmg := beastA.Atk * (0.8 + rand.Float64()*0.4)
			beastB.HP -= dmg
			if beastB.HP < 0 {
				beastB.HP = 0
			}
			logs = append(logs, fmt.Sprintf("Hiệp %d: 🔴 [%s] (Phe Đỏ) tấn công 🔵 [%s] (Phe Xanh) gây %.0f sát thương. [%s] còn %.0f/%.0f HP.", step, beastA.Name, beastB.Name, dmg, beastB.Name, beastB.HP, beastB.MaxHP))
			if beastB.HP <= 0 {
				logs = append(logs, fmt.Sprintf("☠️ [%s] của Phe Xanh đã gục ngã!", beastB.Name))
			} else {
				// B counter attacks A
				dmgB := beastB.Atk * (0.8 + rand.Float64()*0.4)
				beastA.HP -= dmgB
				if beastA.HP < 0 {
					beastA.HP = 0
				}
				logs = append(logs, fmt.Sprintf("Hiệp %d: 🔵 [%s] (Phe Xanh) phản kích 🔴 [%s] (Phe Đỏ) gây %.0f sát thương. [%s] còn %.0f/%.0f HP.", step, beastB.Name, beastA.Name, dmgB, beastA.Name, beastA.HP, beastA.MaxHP))
				if beastA.HP <= 0 {
					logs = append(logs, fmt.Sprintf("☠️ [%s] của Phe Đỏ đã gục ngã!", beastA.Name))
				}
			}
		} else {
			// B attacks A first
			dmg := beastB.Atk * (0.8 + rand.Float64()*0.4)
			beastA.HP -= dmg
			if beastA.HP < 0 {
				beastA.HP = 0
			}
			logs = append(logs, fmt.Sprintf("Hiệp %d: 🔵 [%s] (Phe Xanh) tấn công 🔴 [%s] (Phe Đỏ) gây %.0f sát thương. [%s] còn %.0f/%.0f HP.", step, beastB.Name, beastA.Name, dmg, beastA.Name, beastA.HP, beastA.MaxHP))
			if beastA.HP <= 0 {
				logs = append(logs, fmt.Sprintf("☠️ [%s] của Phe Đỏ đã gục ngã!", beastA.Name))
			} else {
				// A counter attacks B
				dmgA := beastA.Atk * (0.8 + rand.Float64()*0.4)
				beastB.HP -= dmgA
				if beastB.HP < 0 {
					beastB.HP = 0
				}
				logs = append(logs, fmt.Sprintf("Hiệp %d: 🔴 [%s] (Phe Đỏ) phản kích 🔵 [%s] (Phe Xanh) gây %.0f sát thương. [%s] còn %.0f/%.0f HP.", step, beastA.Name, beastB.Name, dmgA, beastB.Name, beastB.HP, beastB.MaxHP))
				if beastB.HP <= 0 {
					logs = append(logs, fmt.Sprintf("☠️ [%s] của Phe Xanh đã gục ngã!", beastB.Name))
				}
			}
		}
		step++
	}

	return "draw", strings.Join(logs, "\n")
}

func getOrCreateCurrentRound(tx *sql.Tx, now time.Time) (*BettingRound, error) {
	currentHour := now.Truncate(time.Hour)
	var r BettingRound
	var lockedAt, resolvedAt sql.NullTime
	err := tx.QueryRow(`
		SELECT id, status, winner, team_a_name, team_b_name, total_bet_a, total_bet_b, battle_log, created_at, locked_at, resolved_at
		FROM beast_betting_rounds
		WHERE created_at = $1
	`, currentHour).Scan(&r.ID, &r.Status, &r.Winner, &r.TeamAName, &r.TeamBName, &r.TotalBetA, &r.TotalBetB, &r.BattleLog, &r.CreatedAt, &lockedAt, &resolvedAt)

	if err == sql.ErrNoRows {
		status := "ACTIVE"
		if now.Minute() >= 55 {
			status = "LOCKED"
		}
		id := uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO beast_betting_rounds (id, status, winner, team_a_name, team_b_name, total_bet_a, total_bet_b, battle_log, created_at)
			VALUES ($1, $2, '', 'Tả Thanh Long', 'Hữu Bạch Hổ', 0, 0, '', $3)
			ON CONFLICT (created_at) DO NOTHING
		`, id, status, currentHour)
		if err != nil {
			return nil, err
		}

		// Query again in case another concurrent transaction inserted it
		err = tx.QueryRow(`
			SELECT id, status, winner, team_a_name, team_b_name, total_bet_a, total_bet_b, battle_log, created_at, locked_at, resolved_at
			FROM beast_betting_rounds
			WHERE created_at = $1
		`, currentHour).Scan(&r.ID, &r.Status, &r.Winner, &r.TeamAName, &r.TeamBName, &r.TotalBetA, &r.TotalBetB, &r.BattleLog, &r.CreatedAt, &lockedAt, &resolvedAt)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	if lockedAt.Valid {
		r.LockedAt = &lockedAt.Time
	}
	if resolvedAt.Valid {
		r.ResolvedAt = &resolvedAt.Time
	}
	return &r, nil
}

func StartBeastBettingWorker(ctx context.Context) {
	workerMutex.Lock()
	if workerStarted {
		workerMutex.Unlock()
		return
	}
	workerStarted = true
	workerMutex.Unlock()

	logger.S.Info("Started Beast Betting background worker (hourly pool resolve)")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.S.Info("Shutting down Beast Betting worker")
			return
		case <-ticker.C:
			if err := runWorkerTick(ctx); err != nil {
				logger.S.Errorf("Error in beast betting worker tick: %v", err)
			}
		}
	}
}

func runWorkerTick(ctx context.Context) error {
	now := time.Now()
	currentHour := now.Truncate(time.Hour)

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Find all past unfinished rounds
	rows, err := tx.QueryContext(ctx, `
		SELECT id, status, winner, team_a_name, team_b_name, total_bet_a, total_bet_b, battle_log, created_at, locked_at, resolved_at
		FROM beast_betting_rounds
		WHERE created_at < $1 AND status != 'FINISHED'
		ORDER BY created_at ASC
	`, currentHour)
	if err != nil {
		return err
	}

	var pastRounds []BettingRound
	for rows.Next() {
		var r BettingRound
		var lockedAt, resolvedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.Status, &r.Winner, &r.TeamAName, &r.TeamBName, &r.TotalBetA, &r.TotalBetB, &r.BattleLog, &r.CreatedAt, &lockedAt, &resolvedAt); err != nil {
			rows.Close()
			return err
		}
		if lockedAt.Valid {
			r.LockedAt = &lockedAt.Time
		}
		if resolvedAt.Valid {
			r.ResolvedAt = &resolvedAt.Time
		}
		pastRounds = append(pastRounds, r)
	}
	rows.Close()

	// Resolve all of them
	resolvedAny := false
	for _, r := range pastRounds {
		if err := resolveRound(tx, &r); err != nil {
			return err
		}
		resolvedAny = true
	}

	// Get or create current hour's round
	r, err := getOrCreateCurrentRound(tx, now)
	if err != nil {
		return err
	}

	// Lock if it is past minute 55 of current hour
	if r.Status == "ACTIVE" && now.Minute() >= 55 {
		_, err = tx.ExecContext(ctx, `
			UPDATE beast_betting_rounds
			SET status = 'LOCKED', locked_at = $1
			WHERE id = $2
		`, now, r.ID)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if resolvedAny {
		notifyBaoHoangMessages("world", sql.NullString{})
	}

	return nil
}

func resolveRound(tx *sql.Tx, r *BettingRound) error {
	winner, battleLog := SimulateBeastFight()
	now := time.Now()

	_, err := tx.Exec(`
		UPDATE beast_betting_rounds
		SET status = 'FINISHED', winner = $1, battle_log = $2, resolved_at = $3
		WHERE id = $4
	`, winner, battleLog, now, r.ID)
	if err != nil {
		return err
	}

	rows, err := tx.Query(`
		SELECT id, player_id, team, amount
		FROM beast_bets
		WHERE round_id = $1
	`, r.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type Bet struct {
		ID       string
		PlayerID string
		Team     string
		Amount   float64
	}
	var bets []Bet
	var totalA, totalB float64
	for rows.Next() {
		var b Bet
		if err := rows.Scan(&b.ID, &b.PlayerID, &b.Team, &b.Amount); err != nil {
			return err
		}
		bets = append(bets, b)
		if b.Team == "team_a" {
			totalA += b.Amount
		} else {
			totalB += b.Amount
		}
	}
	rows.Close()

	isRefund := winner == "draw" || totalA == 0 || totalB == 0

	// New payout logic: 1:1 matching. Matched = min(totalA, totalB).
	// Winners receive their original stake back plus profit equal to their share of the matched pool.
	// Losers receive back their share of the unmatched pool (if any).
	matched := math.Min(totalA, totalB)
	var unmatchedA, unmatchedB float64
	if totalA > totalB {
		unmatchedA = totalA - matched
	} else {
		unmatchedB = totalB - matched
	}

	for _, b := range bets {
		pID, err := uuid.Parse(b.PlayerID)
		if err != nil {
			continue
		}

		var payout float64
		var note string
		var txType string

		if isRefund {
			payout = b.Amount
			note = fmt.Sprintf("Hoàn trả cược Đấu Thú vòng %s (một bên không có cược)", r.CreatedAt.Format("15:00 02/01"))
			txType = "REFUND"
		} else if b.Team == winner {
			// Calculate matched portion of this bet (share of the matched pool)
			var totalWinning float64
			if winner == "team_a" {
				totalWinning = totalA
			} else {
				totalWinning = totalB
			}

			var matchedPortion float64
			if totalWinning > 0 {
				matchedPortion = b.Amount * (matched / totalWinning)
			} else {
				matchedPortion = 0
			}

			profit := math.Round(matchedPortion) // round profit to nearest integer
			payout = b.Amount + profit
			note = fmt.Sprintf("Thưởng thắng cược Đấu Thú vòng %s", r.CreatedAt.Format("15:00 02/01"))
			txType = "EXCHANGE"
		} else {
			// Losing bet: refund unmatched portion of losing side
			var totalLosing float64
			var unmatched float64
			if b.Team == "team_a" {
				totalLosing = totalA
				unmatched = unmatchedA
			} else {
				totalLosing = totalB
				unmatched = unmatchedB
			}

			var refundPortion float64
			if totalLosing > 0 {
				refundPortion = b.Amount * (unmatched / totalLosing)
			} else {
				refundPortion = 0
			}

			payout = math.Round(refundPortion)
			note = fmt.Sprintf("Hoàn trả phần dư cược Đấu Thú vòng %s (phe thua)", r.CreatedAt.Format("15:00 02/01"))
			txType = "REFUND"
		}

		if payout > 0 {
			var before float64
			err = tx.QueryRow("SELECT primeval_stones FROM players WHERE id = $1 FOR UPDATE", pID).Scan(&before)
			if err != nil {
				return err
			}
			after := before + payout
			_, err = tx.Exec("UPDATE players SET primeval_stones = $1 WHERE id = $2", after, pID)
			if err != nil {
				return err
			}

			if err := insertWalletTx(tx, pID, txType, "PRIMEVAL_STONE", int64(payout), int64(before), int64(after), nil, nil, note); err != nil {
				return err
			}
		}

		_, err = tx.Exec("UPDATE beast_bets SET payout = $1 WHERE id = $2", payout, b.ID)
		if err != nil {
			return err
		}

		// Send win/lose/refund notification to player
		if isRefund {
			notify.Send(tx, pID, "BEAST_BETTING", "Hoàn trả cược Đấu Thú", fmt.Sprintf("Vòng đấu %s đã được hoàn trả %.0f Nguyên Thạch do một bên không có cược hoặc trận hòa.", r.CreatedAt.Format("15:00 02/01"), b.Amount))
		} else if b.Team == winner {
			notify.Send(tx, pID, "BEAST_BETTING", "Thắng cược Đấu Thú!", fmt.Sprintf("Chúc mừng đạo hữu đã đoán trúng kết quả vòng %s! Nhận được %.0f Nguyên Thạch.", r.CreatedAt.Format("15:00 02/01"), payout))
		} else {
			notify.Send(tx, pID, "BEAST_BETTING", "Thua cược Đấu Thú", fmt.Sprintf("Rất tiếc, phe cược của đạo hữu đã thất bại ở vòng đấu %s. Hoàn trả phần dư: %.0f Nguyên Thạch.", r.CreatedAt.Format("15:00 02/01"), payout))
		}
	}

	// Broadcast round results to global chat
	var chatContent string
	systemID := "11111111-1111-1111-1111-111111111111"
	globalUUID := "00000000-0000-0000-0000-000000000000"

	if isRefund {
		if winner == "draw" {
			chatContent = fmt.Sprintf("[Đấu Thú] Trận đấu kết thúc với kết quả Hòa! Tổng cược Đỏ: %.0f, Xanh: %.0f Nguyên Thạch. Đã hoàn trả toàn bộ Nguyên Thạch cược.", totalA, totalB)
		} else {
			chatContent = fmt.Sprintf("[Đấu Thú] Vòng đấu kết thúc nhưng một bên không có cược! Tổng cược Đỏ: %.0f, Xanh: %.0f Nguyên Thạch. Đã hoàn trả toàn bộ Nguyên Thạch cược.", totalA, totalB)
		}
	} else if winner == "team_a" {
		chatContent = fmt.Sprintf("[Đấu Thú] Trận đấu kịch tính kết thúc! Phe Đỏ (Tả Thanh Long) giành chiến thắng. Tổng cược Đỏ: %.0f, Xanh: %.0f Nguyên Thạch.", totalA, totalB)
	} else {
		chatContent = fmt.Sprintf("[Đấu Thú] Trận đấu kịch tính kết thúc! Phe Xanh (Hữu Bạch Hổ) giành chiến thắng. Tổng cược Đỏ: %.0f, Xanh: %.0f Nguyên Thạch.", totalA, totalB)
	}

	_, err = tx.Exec(`
		INSERT INTO messages (sender_id, receiver_id, content, channel)
		VALUES ($1, $2, $3, 'world')
	`, systemID, globalUUID, chatContent)
	if err != nil {
		logger.S.Errorf("beast_betting: failed to insert global chat announcement: %v", err)
	}

	return nil
}

func GetCurrentBettingRound(c *gin.Context) {
	playerIDVal, ok := c.Get("player_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := playerIDVal.(uuid.UUID)

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	r, err := getOrCreateCurrentRound(tx, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var myBetTeam string
	var myBetAmount float64
	err = db.DB.QueryRow(`
		SELECT team, amount FROM beast_bets
		WHERE round_id = $1 AND player_id = $2
	`, r.ID, userID).Scan(&myBetTeam, &myBetAmount)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"round":          r,
		"my_bet_team":    myBetTeam,
		"my_bet_amount":  myBetAmount,
		"current_minute": time.Now().Minute(),
	})
}

func GetBettingHistory(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT id, status, winner, team_a_name, team_b_name, total_bet_a, total_bet_b, battle_log, created_at, resolved_at
		FROM beast_betting_rounds
		WHERE status = 'FINISHED'
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var rounds []BettingRound
	for rows.Next() {
		var r BettingRound
		var resolvedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.Status, &r.Winner, &r.TeamAName, &r.TeamBName, &r.TotalBetA, &r.TotalBetB, &r.BattleLog, &r.CreatedAt, &resolvedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if resolvedAt.Valid {
			r.ResolvedAt = &resolvedAt.Time
		}
		rounds = append(rounds, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"history": rounds,
	})
}

func PlaceBeastBet(c *gin.Context) {
	playerIDVal, ok := c.Get("player_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := playerIDVal.(uuid.UUID)

	var req PlaceBetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Team != "team_a" && req.Team != "team_b" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Team phải là 'team_a' hoặc 'team_b'"})
		return
	}

	req.Amount = math.Round(req.Amount)
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng cược phải lớn hơn 0"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	r, err := getOrCreateCurrentRound(tx, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if r.Status != "ACTIVE" || time.Now().Minute() >= 55 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời gian đặt cược cho vòng này đã kết thúc (khóa cược sau 55 phút)"})
		return
	}

	var existingBetID string
	var existingTeam string
	var existingAmount float64
	err = tx.QueryRow(`
		SELECT id, team, amount FROM beast_bets
		WHERE round_id = $1 AND player_id = $2
	`, r.ID, userID).Scan(&existingBetID, &existingTeam, &existingAmount)

	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err == nil && existingTeam != req.Team {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã cược cho phe đối lập rồi, không thể đổi phe!"})
		return
	}

	var balance float64
	err = tx.QueryRow("SELECT primeval_stones FROM players WHERE id = $1 FOR UPDATE", userID).Scan(&balance)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải số dư ví"})
		return
	}

	if balance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch. Đạo hữu chỉ có %.0f Nguyên Thạch", balance)})
		return
	}

	newBalance := balance - req.Amount
	_, err = tx.Exec("UPDATE players SET primeval_stones = $1 WHERE id = $2", newBalance, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := insertWalletTx(tx, userID, "SPEND", "PRIMEVAL_STONE", -int64(req.Amount), int64(balance), int64(newBalance), nil, nil, fmt.Sprintf("Đặt cược Đấu Thú: %s", req.Team)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if existingBetID != "" {
		_, err = tx.Exec(`
			UPDATE beast_bets
			SET amount = amount + $1
			WHERE id = $2
		`, req.Amount, existingBetID)
	} else {
		_, err = tx.Exec(`
			INSERT INTO beast_bets (round_id, player_id, team, amount)
			VALUES ($1, $2, $3, $4)
		`, r.ID, userID, req.Team, req.Amount)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Team == "team_a" {
		_, err = tx.Exec("UPDATE beast_betting_rounds SET total_bet_a = total_bet_a + $1 WHERE id = $2", req.Amount, r.ID)
	} else {
		_, err = tx.Exec("UPDATE beast_betting_rounds SET total_bet_b = total_bet_b + $1 WHERE id = $2", req.Amount, r.ID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Đặt cược thành công!",
		"team":    req.Team,
		"amount":  req.Amount + existingAmount,
	})
}
