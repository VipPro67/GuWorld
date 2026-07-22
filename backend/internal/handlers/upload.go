package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const maxImageSize = 2 << 20

var allowedUploadTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

var uploadEntityTables = map[string]string{
	"item": "item_dictionary",
	"gu":   "gu_dictionary",
}

func AdminUploadImage(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	entityType := c.PostForm("entity_type")
	entityID := c.PostForm("entity_id")

	if entityType == "" || entityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu entity_type hoặc entity_id"})
		return
	}

	tableName, ok := uploadEntityTables[entityType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type phải là 'item' hoặc 'gu'"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy file ảnh!"})
		return
	}
	defer file.Close()

	if header.Size > maxImageSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ảnh quá lớn! Tối đa 2MB."})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedUploadTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ hỗ trợ định dạng JPEG, PNG hoặc WebP!"})
		return
	}

	ext := ".png"
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	}

	uploadDir := filepath.Join("storage", "images", entityType)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo thư mục lưu ảnh"})
		return
	}

	filename := entityID + ext
	filePath := filepath.Join(uploadDir, filename)

	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu file ảnh"})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi ghi file ảnh"})
		return
	}

	imageURL := fmt.Sprintf("/images/%s/%s", entityType, filename)

	_, err = db.DB.Exec(
		fmt.Sprintf(`UPDATE %s SET image_url = $1 WHERE id = $2`, tableName),
		imageURL, entityID,
	)
	if err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật database: " + err.Error()})
		return
	}

	if entityType == "item" {
		models.LoadItemRegistry(db.DB)
	} else {
		models.LoadGuRegistry(db.DB)
	}

	c.JSON(http.StatusOK, gin.H{
		"image_url": imageURL,
		"message":   "Tải ảnh lên thành công!",
	})
}
