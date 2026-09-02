package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// ClaimReferralRequest represents the body of a claim request
type ClaimReferralRequest struct {
	RefereeID string `json:"referee_id" binding:"required"`
	Milestone int    `json:"milestone" binding:"required"`
}

// ClaimReferralReward handles the manual claiming of referral milestone rewards
func ClaimReferralReward(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ClaimReferralRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	refereeUUID, err := uuid.Parse(req.RefereeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid referee ID"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Check if a referral relationship exists
	var refID uuid.UUID
	var hasDeposited bool
	var depositAmountStones int
	var milestone1Rewarded, milestone2Rewarded, milestone3Rewarded bool

	err = tx.QueryRow(`
		SELECT id, has_deposited, deposit_amount_stones, milestone1_rewarded, milestone2_rewarded, milestone3_rewarded
		FROM referrals
		WHERE referee_id = $1 AND referrer_id = $2
	`, refereeUUID, playerID).Scan(&refID, &hasDeposited, &depositAmountStones, &milestone1Rewarded, &milestone2Rewarded, &milestone3Rewarded)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Referral relationship not found or not your referee"})
		return
	}

	// 2. Fetch referee rank and verify verification
	var refereeRank int
	err = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, refereeUUID).Scan(&refereeRank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referee stats"})
		return
	}

	var completedCount int
	err = tx.QueryRow(`
		SELECT COUNT(*)
		FROM player_quest_progress
		WHERE player_id = $1 AND status = 'COMPLETED' AND quest_id IN ('quest_1_awaken', 'quest_2_vital_gu')
	`, refereeUUID).Scan(&completedCount)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check referee quests"})
		return
	}
	verified := refereeRank >= 2 || completedCount >= 2
	if !verified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử này chưa hoàn thành nhiệm vụ tân thủ nên chưa được xác thực."})
		return
	}

	// 3. Process the milestone
	var referrerUsername string
	var referrerStones int
	err = tx.QueryRow(`
		SELECT username, FLOOR(primeval_stones)::BIGINT
		FROM players
		WHERE id = $1
		FOR UPDATE
	`, playerID).Scan(&referrerUsername, &referrerStones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referrer details"})
		return
	}

	message := ""
	var stonesAdded int
	switch req.Milestone {
	case 1:
		if milestone1Rewarded {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đã nhận mốc này rồi."})
			return
		}
		if refereeRank < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử chưa đạt Nhị Chuyển."})
			return
		}
		stonesAdded = 500
		milestone1Rewarded = true
		message = "Nhận thành công 500 Nguyên Thạch!"
	case 2:
		if milestone2Rewarded {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đã nhận mốc này rồi."})
			return
		}
		if refereeRank < 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử chưa đạt Tam Chuyển."})
			return
		}
		var randomGuID string
		err = tx.QueryRow(`SELECT id FROM gu_dictionary WHERE rank = 1 ORDER BY RANDOM() LIMIT 1`).Scan(&randomGuID)
		if err != nil {
			randomGuID = "GU_MOONLIGHT"
		}
		guSpec, _ := models.GetGuAny(randomGuID)
		newGu := models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       randomGuID,
			Rank:       guSpec.Rank,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
			IsVital:    false,
		}
		milestone2Rewarded = true
		message = fmt.Sprintf("Nhận thành công 1 %s!", guSpec.Name)
		// Save the granted Gu immediately instead of rewriting the whole inventory
		if err := db.AddPlayerGu(tx, playerID.String(), newGu); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory"})
			return
		}
	case 3:
		if milestone3Rewarded {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đã nhận mốc này rồi."})
			return
		}
		if !hasDeposited {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đệ tử chưa nạp tiền."})
			return
		}
		commission := int(float64(depositAmountStones) * 0.10)
		if commission <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Hoa hồng không hợp lệ (số tiền nạp quá nhỏ)."})
			return
		}
		stonesAdded = commission
		milestone3Rewarded = true
		message = fmt.Sprintf("Nhận thành công %d Nguyên Thạch tiền hoa hồng!", commission)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mốc nhận thưởng không hợp lệ."})
		return
	}

	// Save updates
	if stonesAdded > 0 {
		_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, stonesAdded, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update resources"})
			return
		}
	}
	_, err = tx.Exec(`
		UPDATE referrals 
		SET milestone1_rewarded = $1, milestone2_rewarded = $2, milestone3_rewarded = $3 
		WHERE id = $4
	`, milestone1Rewarded, milestone2Rewarded, milestone3Rewarded, refID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update milestone flags"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "referral.claim", EntityType: "referral",
		RequestSummary: fmt.Sprintf("nhận thưởng sư đồ mốc %d từ %s", req.Milestone, refereeUUID.String()),
		IPAddress: c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": message,
	})
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
		SELECT FLOOR(primeval_stones)::BIGINT 
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

	// 3. (Old logic for checking milestones was removed since it's now manually claimed)

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
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT ref_code 
		FROM players 
		WHERE id = $1
	`, playerID).Scan(&refCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch referral code"})
		return
	}

	// Fetch referrals stats
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT r.id, p.username, p.rank, r.is_suspicious, r.has_deposited, 
		       r.milestone1_rewarded, r.milestone2_rewarded, r.milestone3_rewarded, r.created_at,
		       (
		           SELECT COUNT(*) >= 2 
		           FROM player_quest_progress pqp 
		           WHERE pqp.player_id = p.id AND pqp.status = 'COMPLETED' AND pqp.quest_id IN ('quest_1_awaken', 'quest_2_vital_gu')
		       ) as completed_quests
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
		IsSuspicious       bool      `json:"is_suspicious"`
		HasDeposited       bool      `json:"has_deposited"`
		Milestone1Rewarded bool      `json:"milestone1_rewarded"`
		Milestone2Rewarded bool      `json:"milestone2_rewarded"`
		Milestone3Rewarded bool      `json:"milestone3_rewarded"`
		IsVerified         bool      `json:"is_verified"`
		CreatedAt          time.Time `json:"created_at"`
	}

	referees := []RefereeInfo{}
	for rows.Next() {
		var r RefereeInfo
		var completedQuests bool
		err := rows.Scan(
			&r.ID, &r.Username, &r.Rank, &r.IsSuspicious, &r.HasDeposited,
			&r.Milestone1Rewarded, &r.Milestone2Rewarded, &r.Milestone3Rewarded, &r.CreatedAt, &completedQuests,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse referee details"})
			return
		}
		
		r.IsVerified = r.Rank >= 2 || completedQuests
		referees = append(referees, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"ref_code":  refCode,
		"referees":  referees,
		"total_ref": len(referees),
	})
}
