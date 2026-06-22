package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// CheckReferralMilestones evaluates and awards rewards to the referrer if the referee is verified
func CheckReferralMilestones(tx *sql.Tx, refereeID uuid.UUID) error {
	// 1. Check if a referral relationship exists
	var refID uuid.UUID
	var referrerID uuid.UUID
	var hasDeposited bool
	var depositAmountStones int
	var milestone1Rewarded bool
	var milestone2Rewarded bool
	var milestone3Rewarded bool

	err := tx.QueryRow(`
		SELECT id, referrer_id, has_deposited, deposit_amount_stones, milestone1_rewarded, milestone2_rewarded, milestone3_rewarded
		FROM referrals
		WHERE referee_id = $1
	`, refereeID).Scan(&refID, &referrerID, &hasDeposited, &depositAmountStones, &milestone1Rewarded, &milestone2Rewarded, &milestone3Rewarded)

	if err != nil {
		if err == sql.ErrNoRows {
			// No referrer, do nothing
			return nil
		}
		return fmt.Errorf("failed to fetch referral record: %w", err)
	}

	// 2. Fetch referee details (Rank and Completed Quests)
	var refereeRank int
	var completedQuestsBytes []byte
	err = tx.QueryRow(`
		SELECT rank, completed_quests
		FROM players
		WHERE id = $1
	`, refereeID).Scan(&refereeRank, &completedQuestsBytes)
	if err != nil {
		return fmt.Errorf("failed to fetch referee stats: %w", err)
	}

	var completedQuests []string
	if len(completedQuestsBytes) > 0 {
		_ = json.Unmarshal(completedQuestsBytes, &completedQuests)
	}

	// 3. Determine if the referee is verified (anti-cheat verification)
	// Referee is verified if:
	// - reached Rank 2
	// - OR completed both novice quests: "quest_1_awaken" and "quest_2_vital_gu"
	hasQuest1 := false
	hasQuest2 := false
	for _, q := range completedQuests {
		if q == "quest_1_awaken" {
			hasQuest1 = true
		}
		if q == "quest_2_vital_gu" {
			hasQuest2 = true
		}
	}
	verified := refereeRank >= 2 || (hasQuest1 && hasQuest2)

	if !verified {
		// Not verified yet, delay rewards
		return nil
	}

	// 4. Fetch referrer details for updates (with FOR UPDATE locks to prevent race conditions)
	var referrerUsername string
	var referrerStones int
	var referrerGuInvBytes []byte
	err = tx.QueryRow(`
		SELECT username, primeval_stones, gu_inventory
		FROM players
		WHERE id = $1
		FOR UPDATE
	`, referrerID).Scan(&referrerUsername, &referrerStones, &referrerGuInvBytes)
	if err != nil {
		return fmt.Errorf("failed to fetch referrer details: %w", err)
	}

	var referrerGuInv models.GuInventory
	if len(referrerGuInvBytes) > 0 {
		_ = json.Unmarshal(referrerGuInvBytes, &referrerGuInv)
	}

	rewardsAwarded := false
	var notifications []string

	// Milestone 1: Referee reaches Rank 2 -> Referrer gets 500 Primeval Stones
	if refereeRank >= 2 && !milestone1Rewarded {
		referrerStones += 500
		milestone1Rewarded = true
		rewardsAwarded = true
		notifications = append(notifications, "Đệ tử của bạn đã đột phá Nhất Chuyển lên Nhị Chuyển! Nhận ngay 500 Nguyên Thạch.")
	}

	// Milestone 2: Referee reaches Rank 3 -> Referrer gets a random Gu bug
	if refereeRank >= 3 && !milestone2Rewarded {
		// Fetch a random Rank 1 Gu bug ID from the dictionary
		var randomGuID string
		err = tx.QueryRow(`
			SELECT id 
			FROM gu_dictionary 
			WHERE rank = 1 
			ORDER BY RANDOM() 
			LIMIT 1
		`).Scan(&randomGuID)
		if err != nil {
			log.Printf("Warning: failed to roll random Gu bug from dictionary: %v. Defaulting to Moonlight Gu.", err)
			randomGuID = "GU_MOONLIGHT"
		}

		newGu := models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       randomGuID,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
			IsVital:    false,
		}
		referrerGuInv = append(referrerGuInv, newGu)
		milestone2Rewarded = true
		rewardsAwarded = true
		notifications = append(notifications, fmt.Sprintf("Đệ tử của bạn đã đột phá lên Tam Chuyển! Nhận ngay một Cổ trùng ngẫu nhiên: %s.", randomGuID))
	}

	// Milestone 3: Referee has deposited -> Referrer gets 10% of the deposit amount in Primeval Stones
	if hasDeposited && !milestone3Rewarded {
		commission := int(float64(depositAmountStones) * 0.10)
		if commission > 0 {
			referrerStones += commission
			milestone3Rewarded = true
			rewardsAwarded = true
			notifications = append(notifications, fmt.Sprintf("Đệ tử của bạn đã nạp tiền lần đầu! Nhận ngay 10%% hoa hồng trị giá %d Nguyên Thạch.", commission))
		}
	}

	if rewardsAwarded {
		// Save referrer updates
		referrerGuInvBytes, _ = json.Marshal(referrerGuInv)
		_, err = tx.Exec(`
			UPDATE players 
			SET primeval_stones = $1, gu_inventory = $2 
			WHERE id = $3
		`, referrerStones, referrerGuInvBytes, referrerID)
		if err != nil {
			return fmt.Errorf("failed to update referrer rewards: %w", err)
		}

		// Save referral milestone flags
		_, err = tx.Exec(`
			UPDATE referrals 
			SET milestone1_rewarded = $1, milestone2_rewarded = $2, milestone3_rewarded = $3 
			WHERE id = $4
		`, milestone1Rewarded, milestone2Rewarded, milestone3Rewarded, refID)
		if err != nil {
			return fmt.Errorf("failed to update referral milestone flags: %w", err)
		}

		// Send system notification messages to the referrer
		for _, msg := range notifications {
			_, _ = tx.Exec(`
				INSERT INTO messages (sender_id, receiver_id, content, is_read, channel)
				VALUES ('00000000-0000-0000-0000-000000000000', $1, $2, false, 'private')
			`, referrerID, msg)
		}
	}

	return nil
}

// SimulateDepositRequest represents the body of simulated deposit request
type SimulateDepositRequest struct {
	Stones int `json:"stones" binding:"required,min=100"`
}

// SimulateDeposit simulates referee purchasing Primeval Stones
func SimulateDeposit(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SimulateDepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Add stones to the player making the deposit
	var currentStones int
	err = tx.QueryRow(`
		SELECT primeval_stones 
		FROM players 
		WHERE id = $1 
		FOR UPDATE
	`, playerID).Scan(&currentStones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stones"})
		return
	}

	currentStones += req.Stones
	_, err = tx.Exec(`
		UPDATE players 
		SET primeval_stones = $1 
		WHERE id = $2
	`, currentStones, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player stones"})
		return
	}

	// 2. Track deposit in referrals table if they were referred
	var refID uuid.UUID
	var hasDeposited bool
	var depositAmountStones int

	err = tx.QueryRow(`
		SELECT id, has_deposited, deposit_amount_stones 
		FROM referrals 
		WHERE referee_id = $1
	`, playerID).Scan(&refID, &hasDeposited, &depositAmountStones)

	if err == nil {
		// Update referral deposit stats
		hasDeposited = true
		depositAmountStones += req.Stones

		_, err = tx.Exec(`
			UPDATE referrals 
			SET has_deposited = $1, deposit_amount_stones = $2 
			WHERE id = $3
		`, hasDeposited, depositAmountStones, refID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record referral deposit"})
			return
		}
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking referral status"})
		return
	}

	// 3. Trigger check milestones (will only award to referrer if referee is verified)
	err = CheckReferralMilestones(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to evaluate referral milestones: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit deposit transaction"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "deposit.simulate", EntityType: "player",
		RequestSummary: fmt.Sprintf("nạp %d Nguyên Thạch (admin simulated)", req.Stones),
		Metadata: map[string]interface{}{"stones": req.Stones, "new_stones": currentStones},
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message":         "Nạp Nguyên Thạch thành công (Simulated)!",
		"deposited_stones": req.Stones,
		"new_stones":      currentStones,
	})
}

// GetReferralStats returns the referral code and details of invited referees
func GetReferralStats(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Fetch player's ref code
	var refCode string
	err := db.DB.QueryRow(`
		SELECT ref_code 
		FROM players 
		WHERE id = $1
	`, playerID).Scan(&refCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referral code"})
		return
	}

	// Fetch referrals stats
	rows, err := db.DB.Query(`
		SELECT r.id, p.username, p.rank, r.referee_ip, r.is_suspicious, r.has_deposited, 
		       r.milestone1_rewarded, r.milestone2_rewarded, r.milestone3_rewarded, r.created_at
		FROM referrals r
		JOIN players p ON r.referee_id = p.id
		WHERE r.referrer_id = $1
		ORDER BY r.created_at DESC
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referrals stats"})
		return
	}
	defer rows.Close()

	type RefereeInfo struct {
		ID                 uuid.UUID `json:"id"`
		Username           string    `json:"username"`
		Rank               int       `json:"rank"`
		IP                 string    `json:"ip"`
		IsSuspicious       bool      `json:"is_suspicious"`
		HasDeposited       bool      `json:"has_deposited"`
		Milestone1Rewarded bool      `json:"milestone1_rewarded"`
		Milestone2Rewarded bool      `json:"milestone2_rewarded"`
		Milestone3Rewarded bool      `json:"milestone3_rewarded"`
		CreatedAt          time.Time `json:"created_at"`
	}

	referees := []RefereeInfo{}
	for rows.Next() {
		var r RefereeInfo
		err := rows.Scan(
			&r.ID, &r.Username, &r.Rank, &r.IP, &r.IsSuspicious, &r.HasDeposited,
			&r.Milestone1Rewarded, &r.Milestone2Rewarded, &r.Milestone3Rewarded, &r.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse referee details"})
			return
		}
		referees = append(referees, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"ref_code":  refCode,
		"referees":  referees,
		"total_ref": len(referees),
	})
}
