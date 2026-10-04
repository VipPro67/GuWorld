package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type PaySectSalaryRequest struct {
	Confirm bool `json:"confirm" binding:"required"`
}

type SectMemberSalaryInfo struct {
	PlayerID        uuid.UUID `json:"player_id"`
	Username        string    `json:"username"`
	MemberRank      string    `json:"member_rank"`
	ContributionPts int64     `json:"contribution_points"`
	Salary          int64     `json:"salary"`
}

type PaySectSalaryResponse struct {
	Message     string                 `json:"message"`
	TotalSalary int64                  `json:"total_salary"`
	SectFunds   int64                  `json:"sect_funds"`
	PaidMembers []SectMemberSalaryInfo `json:"paid_members"`
}

var rankSalaryBonus = map[string]int64{
	"NGOAI_MON":   0,
	"NOI_MON":     50,
	"TINH_ANH":    150,
	"CHAN_TRUYEN": 400,
	"TRUONG_LAO":  1000,
	"HO_PHAP":     2000,
	"CHUONG_MON":  5000,
}

func calculateSalary(sectLevel int, memberRank string) int64 {
	baseSalary := int64(sectLevel) * 100
	bonus := rankSalaryBonus[memberRank]
	return baseSalary + bonus
}

func PaySectSalary(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req PaySectSalaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần xác nhận để phát lương"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	var sectLevel int
	var sectFunds int64
	var lastSalaryPaidAt *time.Time
	err = tx.QueryRow(`
		SELECT sm.sect_id, sm.member_rank, s.level, s.funds, s.last_salary_paid_at
		FROM sect_members sm
		JOIN sects s ON s.id = sm.sect_id
		WHERE sm.player_id = $1
	`, playerID).Scan(&sectID, &callerRank, &sectLevel, &sectFunds, &lastSalaryPaidAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão, Hộ Pháp hoặc Chưởng Môn mới có quyền phát lương!"})
		return
	}

	if lastSalaryPaidAt != nil {
		now := time.Now()
		if now.Sub(*lastSalaryPaidAt) < 24*time.Hour {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tông môn đã phát lương hôm nay. Vui lòng chờ 24h."})
			return
		}
	}

	rows, err := tx.Query(`
		SELECT sm.player_id, p.username, sm.member_rank, sm.contribution_points
		FROM sect_members sm
		JOIN players p ON p.id = sm.player_id
		WHERE sm.sect_id = $1
	`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect members"})
		return
	}
	defer rows.Close()

	var members []SectMemberSalaryInfo
	var totalSalary int64
	for rows.Next() {
		var m SectMemberSalaryInfo
		if err := rows.Scan(&m.PlayerID, &m.Username, &m.MemberRank, &m.ContributionPts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan member"})
			return
		}
		m.Salary = calculateSalary(sectLevel, m.MemberRank)
		totalSalary += m.Salary
		members = append(members, m)
	}

	if totalSalary == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có thành viên nào để phát lương"})
		return
	}

	if sectFunds < totalSalary {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ! Cần: %d, Hiện có: %d", totalSalary, sectFunds)})
		return
	}

	newFunds := sectFunds - totalSalary
	_, err = tx.Exec(`UPDATE sects SET funds = $1, last_salary_paid_at = now() WHERE id = $2`, newFunds, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	for _, m := range members {
		_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, m.Salary, m.PlayerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player stones"})
			return
		}
	}

	var pid uuid.UUID
	for _, m := range members {
		pid = m.PlayerID
		auditlog.Log(tx, auditlog.Entry{
			PlayerID:       &pid,
			Action:         "SECT_SALARY_RECEIVED",
			EntityType:     "sect",
			EntityID:       sectID,
			RequestSummary: fmt.Sprintf("Nhận lương tông môn: %d linh thạch (cấp tông %d, chức vụ %s)", m.Salary, sectLevel, m.MemberRank),
			Metadata: map[string]interface{}{
				"sect_id":          sectID,
				"sect_level":       sectLevel,
				"member_rank":      m.MemberRank,
				"salary":           m.Salary,
				"base_salary":      int64(sectLevel) * 100,
				"rank_bonus":       rankSalaryBonus[m.MemberRank],
				"contribution_pts": m.ContributionPts,
			},
			IPAddress: c.ClientIP(),
			Success:   true,
		})
	}

	pid = playerID
	auditlog.Log(tx, auditlog.Entry{
		PlayerID:       &pid,
		Action:         "SECT_SALARY_PAID",
		EntityType:     "sect",
		EntityID:       sectID,
		RequestSummary: fmt.Sprintf("Đã phát lương cho %d thành viên, tổng: %d linh thạch", len(members), totalSalary),
		Metadata: map[string]interface{}{
			"sect_id":           sectID,
			"sect_level":        sectLevel,
			"total_salary":      totalSalary,
			"sect_funds_before": sectFunds,
			"sect_funds_after":  newFunds,
			"member_count":      len(members),
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, PaySectSalaryResponse{
		Message:     fmt.Sprintf("Đã phát lương cho %d thành viên, tổng %d linh thạch", len(members), totalSalary),
		TotalSalary: totalSalary,
		SectFunds:   newFunds,
		PaidMembers: members,
	})
}
