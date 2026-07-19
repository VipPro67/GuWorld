package handlers

import (
	"bytes"
	"io"
	"net/http"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

const maxAvatarSize = 2 << 20 // 2 MB

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// UploadAvatar handles avatar image upload to Cloudinary
func UploadAvatar(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy file ảnh!"})
		return
	}
	defer file.Close()

	if header.Size > maxAvatarSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ảnh quá lớn! Tối đa 2MB."})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedImageTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ định dạng JPEG, PNG hoặc WebP!"})
		return
	}

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc file"})
		return
	}

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cloudinary chưa được cấu hình!"})
		return
	}

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo Cloudinary instance"})
		return
	}

	resp, err := cld.Upload.Upload(c, bytes.NewReader(fileBytes), uploader.UploadParams{
		PublicID:  "avatar_" + playerID.String(),
		Overwrite: api.Bool(true),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cloudinary error: " + err.Error()})
		return
	}

	if resp.SecureURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload thành công nhưng không nhận được URL ảnh"})
		return
	}

	_, err = db.DB.Exec(`UPDATE players SET avatar_url = $1 WHERE id = $2`, resp.SecureURL, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu avatar vào database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"avatar_url": resp.SecureURL,
		"message":    "Cập nhật ảnh đại diện thành công!",
	})
}
