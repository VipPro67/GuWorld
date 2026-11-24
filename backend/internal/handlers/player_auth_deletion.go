package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"golang.org/x/crypto/bcrypt"
)

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required"`
	Reason   string `json:"reason"`
}

// RequestAccountDeletion handles user right to be forgotten (Nghị định 13/2023/NĐ-CP).
// It verifies password, soft-deletes the account and anonymizes personal identifiers (email, IPs, fingerprints).
func RequestAccountDeletion(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Phiên đăng nhập không hợp lệ!"})
		return
	}

	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập mật khẩu xác nhận chính chủ!"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể mở giao dịch", err))
		return
	}
	defer tx.Rollback()

	var passwordHash string
	var email string
	var isDeleted bool

	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT password_hash, email, COALESCE(is_deleted, false)
		FROM players
		WHERE id = $1
		FOR UPDATE
	`, playerID).Scan(&passwordHash, &email, &isDeleted)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thông tin tài khoản"})
		} else {
			httperr.Respond(c, httperr.Internal("Lỗi truy vấn tài khoản", err))
		}
		return
	}

	if isDeleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tài khoản này đã bị xóa/vô danh hóa trước đó!"})
		return
	}

	if passwordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Mật khẩu xác nhận không chính xác!"})
			return
		}
	}

	anonymizedEmail := fmt.Sprintf("deleted_%s@guworld.local", playerID.String()[:8])
	anonymizedIP := "0.0.0.0"
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Người dùng yêu cầu xóa tài khoản theo Nghị định 13/2023/NĐ-CP"
	}

	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET is_deleted = true,
			deleted_at = NOW(),
			deletion_reason = $1,
			email = $2,
			password_hash = '',
			register_ip = $3,
			last_login_ip = $3,
			register_fingerprint = '',
			last_login_fingerprint = '',
			active_session_id = ''
		WHERE id = $4
	`, reason, anonymizedEmail, anonymizedIP, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật xóa dữ liệu cá nhân", err))
		return
	}

	auditlog.LogContext(c.Request.Context(), tx, auditlog.Entry{
		PlayerID:  &playerID,
		Action:    "player.account_deleted_gdpr",
		Reason:    reason,
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi xác nhận xóa tài khoản", err))
		return
	}

	middleware.InvalidateSessionCache(playerID)
	clearAuthCookie(c)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Tài khoản và toàn bộ dữ liệu định danh cá nhân đã được vô danh hóa thành công theo Nghị định 13/2023/NĐ-CP.",
	})
}

type ReconsentTermsRequest struct {
	AgreeTerms bool `json:"agree_terms" binding:"required"`
}

// ReconsentTerms updates player consent when terms/privacy policy versions are updated.
func ReconsentTerms(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Phiên đăng nhập không hợp lệ!"})
		return
	}

	var req ReconsentTermsRequest
	if err := c.ShouldBindJSON(&req); err != nil || !req.AgreeTerms {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu cần đồng ý với Điều Khoản Dịch Vụ và Chính Sách Bảo Mật mới!"})
		return
	}

	_, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE players
		SET accepted_terms_version = $1,
			accepted_privacy_version = $2,
			terms_accepted_at = NOW(),
			terms_accepted_ip = $3
		WHERE id = $4
	`, CurrentTermsVersion, CurrentPrivacyVersion, c.ClientIP(), playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật điều khoản", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Đã cập nhật chấp thuận điều khoản thành công.",
	})
}
