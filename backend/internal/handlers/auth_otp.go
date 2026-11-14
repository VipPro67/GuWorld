package handlers

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
)

type SendOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// EmailVerificationRequired is opt-in; email verification is disabled by default.
func EmailVerificationRequired() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("REQUIRE_EMAIL_VERIFICATION")), "true")
}

func GetAuthConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"email_verification_required": EmailVerificationRequired()})
}

// SendOTP handles generating and sending an OTP to the user's email
func SendOTP(c *gin.Context) {
	if !EmailVerificationRequired() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Xác thực email hiện đang được tắt"})
		return
	}
	var req SendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng Email không hợp lệ!"})
		return
	}

	req.Email = normalizeEmail(req.Email)
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng Email không hợp lệ!"})
		return
	}
	if isDisposableEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email tạm thời không được chấp nhận. Vui lòng dùng email thật!"})
		return
	}

	// 1. Check if email already registered
	var emailExists bool
	err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE email = $1)", req.Email).Scan(&emailExists)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kết nối cơ sở dữ liệu", err))
		return
	}
	if emailExists {
		c.JSON(http.StatusConflict, gin.H{"error": "Email này đã được đăng ký trong Cổ Giới!"})
		return
	}

	// 1.5. Prevent email bombing: Check if OTP was sent to this email less than 1 minute ago
	var lastSentAt time.Time
	err = db.DB.QueryRow("SELECT created_at FROM email_otps WHERE email = $1", req.Email).Scan(&lastSentAt)
	if err == nil && time.Since(lastSentAt) < time.Minute {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Yêu cầu OTP quá nhanh. Vui lòng đợi 1 phút trước khi yêu cầu gửi lại!"})
		return
	}

	// 2. Generate OTP
	otp, err := auth.GenerateOTP()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tạo mã xác thực", err))
		return
	}

	expiresAt := time.Now().Add(5 * time.Minute)

	// 3. Save OTP to DB
	_, err = db.DB.Exec(`
		INSERT INTO email_otps (email, otp, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE 
		SET otp = EXCLUDED.otp, expires_at = EXCLUDED.expires_at, created_at = NOW()
	`, req.Email, otp, expiresAt)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi lưu mã xác thực", err))
		return
	}

	// 4. Send email synchronously so the client knows if Brevo is down
	err = auth.SendEmailOTP(req.Email, otp)
	if err != nil {
		// Clean up the generated OTP on failure so we don't hold bad state
		_, _ = db.DB.Exec("DELETE FROM email_otps WHERE email = $1", req.Email)
		httperr.Respond(c, httperr.Internal("Gửi email xác thực thất bại. Vui lòng kiểm tra lại địa chỉ email hoặc liên hệ Admin.", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mã xác thực OTP đã được gửi đến email của đạo hữu!"})
}
