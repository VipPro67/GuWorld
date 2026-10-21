package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// UpgradeSect upgrades the sect's level
func UpgradeSect(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Fetch caller's sect and rank
	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền tăng cấp tông môn!"})
		return
	}

	// Atomic level upgrade and funds deduction with guard clause
	var newLevel int
	var newFunds int64
	err = tx.QueryRow(`
		UPDATE sects
		SET level = level + 1,
		    funds = funds - (level * level * 5000)
		WHERE id = $1 AND funds >= (level * level * 5000)
		RETURNING level, funds
	`, sectID).Scan(&newLevel, &newFunds)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Quỹ tông môn không đủ để nâng cấp!"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade sect"})
		return
	}
	level := newLevel
	funds := newFunds

	upgradeSpend := int64(newLevel-1) * int64(newLevel-1) * 5000
	auditlog.Log(tx, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.upgrade", EntityType: "sect",
		EntityID:       sectID,
		RequestSummary: fmt.Sprintf("Nâng cấp tông môn lên Cấp %d, tiêu %d Quỹ (còn %d)", newLevel, upgradeSpend, newFunds),
		Metadata: map[string]interface{}{
			"sect_id":      sectID,
			"new_level":    newLevel,
			"cost_funds":   upgradeSpend,
			"funds_change": -upgradeSpend,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	// The UPDATE priced the upgrade with the pre-increment level, so recover the
	// cost from the new level for the chat notice.
	upgradeCost := int64(newLevel-1) * int64(newLevel-1) * 5000
	announceSectFundsSpend(sectID, playerID, fmt.Sprintf("nâng cấp tông môn lên Cấp %d", newLevel), upgradeCost, newFunds)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Nâng cấp tông môn thành công!",
		"new_level": level,
		"new_funds": funds,
	})
}

type ToggleSectApprovalRequest struct {
	RequireApproval bool `json:"require_approval"`
}

// ToggleSectApproval toggles the require_approval setting for a sect
func ToggleSectApproval(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ToggleSectApprovalRequest
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

	var sectID, callerRank string
	var sectLevel int
	err = tx.QueryRow(`SELECT sect_id, member_rank, (SELECT level FROM sects WHERE id = sect_members.sect_id) FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank, &sectLevel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền thay đổi thiết lập này!"})
		return
	}

	if req.RequireApproval && sectLevel < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tông môn phải đạt cấp 10 mới có thể bật duyệt thành viên!"})
		return
	}

	_, err = tx.Exec(`UPDATE sects SET require_approval = $1 WHERE id = $2`, req.RequireApproval, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect setting"})
		return
	}

	toggleWord := "bật"
	if !req.RequireApproval {
		toggleWord = "tắt"
	}
	auditlog.Log(tx, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.toggle_approval", EntityType: "sect",
		EntityID:       sectID,
		RequestSummary: fmt.Sprintf("%s duyệt thành viên gia nhập", toggleWord),
		Metadata:       map[string]interface{}{"sect_id": sectID, "require_approval": req.RequireApproval},
		IPAddress:      c.ClientIP(), Success: true,
	})

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	msg := "Đã bật duyệt thành viên gia nhập!"
	if !req.RequireApproval {
		msg = "Đã tắt duyệt thành viên, cho phép gia nhập tự do!"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          msg,
		"require_approval": req.RequireApproval,
	})
}

type ToggleSectPromotionApprovalRequest struct {
	RequirePromotionApproval bool `json:"require_promotion_approval"`
}

// ToggleSectPromotionApproval toggles whether all rank-ups require leadership approval.
func ToggleSectPromotionApproval(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ToggleSectPromotionApprovalRequest
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

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền thay đổi thiết lập này!"})
		return
	}

	_, err = tx.Exec(`UPDATE sects SET require_promotion_approval = $1 WHERE id = $2`, req.RequirePromotionApproval, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect setting"})
		return
	}

	toggleWord := "bật"
	if !req.RequirePromotionApproval {
		toggleWord = "tắt"
	}
	auditlog.Log(tx, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.toggle_promotion_approval", EntityType: "sect",
		EntityID:       sectID,
		RequestSummary: fmt.Sprintf("%s duyệt lên cấp cho đệ tử", toggleWord),
		Metadata:       map[string]interface{}{"sect_id": sectID, "require_promotion_approval": req.RequirePromotionApproval},
		IPAddress:      c.ClientIP(), Success: true,
	})

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	msg := "Đã bật duyệt lên cấp: mọi thăng cấp đều cần Trưởng Lão/Chưởng Môn duyệt!"
	if !req.RequirePromotionApproval {
		msg = "Đã tắt duyệt lên cấp: đệ tử tự thăng cấp theo tiêu chuẩn (trừ Chân Truyền/Trưởng Lão)."
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                    msg,
		"require_promotion_approval": req.RequirePromotionApproval,
	})
}
