package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
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

	// Fetch level and funds
	var level int
	var funds int64
	err = tx.QueryRow(`SELECT level, funds FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&level, &funds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details"})
		return
	}

	cost := int64(level) * int64(level) * 5000
	if funds < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quỹ tông môn không đủ để nâng cấp!"})
		return
	}

	funds -= cost
	level++

	_, err = tx.Exec(`UPDATE sects SET level = $1, funds = $2 WHERE id = $3`, level, funds, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade sect"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

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
