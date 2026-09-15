package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

const (
	maxGuEffectPreviewBytes = 256 << 10
	maxGuCatalogEntries     = 100_000
	maxGuCatalogEffectBytes = 32 << 20
)

type guEffectPreviewRequest struct {
	Effects json.RawMessage `json:"effects" binding:"required"`
}

func AdminValidateGuEffects(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxGuEffectPreviewBytes)
	var req guEffectPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Payload preview vượt giới hạn 256 KiB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload preview không hợp lệ: " + err.Error()})
		return
	}

	normalized, report, err := effectcatalog.NormalizeAndValidate(req.Effects)
	response := gin.H{"valid": err == nil, "report": report}
	if err == nil {
		response["normalized_effects"] = json.RawMessage(normalized)
	}
	c.JSON(http.StatusOK, response)
}

func AdminAuditGuEffects(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, rank, effects FROM gu_dictionary ORDER BY id, rank`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc catalog Cổ: " + err.Error()})
		return
	}
	defer rows.Close()

	entries := make([]effectcatalog.GuEntry, 0)
	totalBytes := 0
	for rows.Next() {
		var entry effectcatalog.GuEntry
		var effects []byte
		if err := rows.Scan(&entry.ID, &entry.Rank, &effects); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc effect Cổ: " + err.Error()})
			return
		}
		if len(entries) >= maxGuCatalogEntries || totalBytes+len(effects) > maxGuCatalogEffectBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Catalog Cổ vượt giới hạn audit an toàn"})
			return
		}
		entry.Effects = append(json.RawMessage(nil), effects...)
		entries = append(entries, entry)
		totalBytes += len(effects)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi duyệt catalog Cổ: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"source":       "database",
		"generated_at": time.Now().UTC(),
		"report":       effectcatalog.ValidateGuCatalog(entries),
	})
}
