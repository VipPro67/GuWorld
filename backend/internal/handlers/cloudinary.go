package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

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

	// Parse multipart form
	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy file ảnh!"})
		return
	}
	defer file.Close()

	// Validate file size
	if header.Size > maxAvatarSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ảnh quá lớn! Tối đa 2MB."})
		return
	}

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if !allowedImageTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ định dạng JPEG, PNG hoặc WebP!"})
		return
	}

	// Read file contents
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc file"})
		return
	}

	// Upload to Cloudinary
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cloudinary chưa được cấu hình!"})
		return
	}

	uploadURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", cloudName)

	// Build multipart request for Cloudinary
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("api_key", apiKey)
	_ = writer.WriteField("upload_preset", "ml_default")
	_ = writer.WriteField("folder", "guworld/avatars")
	_ = writer.WriteField("public_id", fmt.Sprintf("avatar_%s", playerID.String()))
	_ = writer.WriteField("overwrite", "true")
	_ = writer.WriteField("transformation", "c_limit,w_256,h_256")

	part, err := writer.CreateFormFile("file", header.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo request upload"})
		return
	}
	_, _ = part.Write(fileBytes)
	writer.Close()

	req, err := http.NewRequest("POST", uploadURL, &body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tạo request"})
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kết nối Cloudinary: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var cloudResp struct {
		SecureURL string `json:"secure_url"`
		Error     struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cloudResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi phân tích response từ Cloudinary"})
		return
	}

	if cloudResp.Error.Message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cloudinary error: " + cloudResp.Error.Message})
		return
	}

	if cloudResp.SecureURL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload thành công nhưng không nhận được URL ảnh"})
		return
	}

	// Update avatar_url in database
	_, err = db.DB.Exec(`UPDATE players SET avatar_url = $1 WHERE id = $2`, cloudResp.SecureURL, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu avatar vào database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"avatar_url": cloudResp.SecureURL,
		"message":    "Cập nhật ảnh đại diện thành công!",
	})
}
