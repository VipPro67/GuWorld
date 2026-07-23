package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"os"
	"strings"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// GenerateOTP generates a random 6-digit numeric string
func GenerateOTP() (string, error) {
	var otp string
	for i := 0; i < 6; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		otp += fmt.Sprintf("%d", num)
	}
	return otp, nil
}

// SendEmailOTP sends the generated OTP via SMTP configured in environment
func SendEmailOTP(toEmail string, otp string) error {
	smtpHost := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	smtpPort := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	smtpUsername := strings.TrimSpace(os.Getenv("SMTP_USERNAME"))
	smtpPassword := strings.TrimSpace(os.Getenv("SMTP_PASSWORD"))
	mailFrom := strings.TrimSpace(os.Getenv("MAIL_FROM"))

	if smtpHost == "" || smtpPort == "" || smtpUsername == "" || smtpPassword == "" || mailFrom == "" {
		return fmt.Errorf("missing SMTP configuration in environment variables")
	}

	subject := "Subject: [Cổ Giới] Mã Xác Thực Đăng Ký (OTP)\r\n"
	contentType := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; background-color: #1a1a1a; color: #ffffff; padding: 20px;">
			<div style="max-width: 600px; margin: 0 auto; background-color: #2c2c2c; border: 1px solid #444; border-radius: 8px; padding: 30px; box-shadow: 0 4px 10px rgba(0,0,0,0.3);">
				<h2 style="color: #ffd700; text-align: center;">CỔ GIỚI - XÁC THỰC OTP</h2>
				<p style="font-size: 16px; line-height: 1.6;">Chào đạo hữu,</p>
				<p style="font-size: 16px; line-height: 1.6;">Đạo hữu đang thực hiện đăng ký tài khoản trong <strong>Cổ Giới</strong>. Vui lòng sử dụng mã xác thực (OTP) dưới đây để hoàn tất đăng ký:</p>
				<div style="text-align: center; margin: 30px 0;">
					<span style="font-size: 32px; font-weight: bold; color: #ffd700; background-color: #111; padding: 15px 30px; border-radius: 5px; border: 1px solid #ffd700; letter-spacing: 5px;">%s</span>
				</div>
				<p style="font-size: 14px; color: #aaa;">* Mã OTP này có hiệu lực trong vòng <strong>5 phút</strong>. Vui lòng không chia sẻ mã này cho bất kỳ ai.</p>
				<hr style="border: 0; border-top: 1px solid #444; margin: 20px 0;">
				<p style="font-size: 12px; color: #777; text-align: center;">© Cổ Giới - Thiên Địa Cổ Sư</p>
			</div>
		</body>
		</html>
	`, otp)

	msg := []byte("To: " + toEmail + "\r\n" +
		"From: " + mailFrom + "\r\n" +
		subject +
		contentType +
		body)

	auth := smtp.PlainAuth("", smtpUsername, smtpPassword, smtpHost)
	addr := smtpHost + ":" + smtpPort

	err := smtp.SendMail(addr, auth, mailFrom, []string{toEmail}, msg)
	if err != nil {
		logger.S.Errorf("Failed to send SMTP email to %s: %v", toEmail, err)
		return err
	}

	return nil
}
