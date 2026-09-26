package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

func sortedEffectKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// AdminEffectCatalogMetadata exposes the validator's catalogs so every admin
// editor authors exactly the same vocabulary accepted by the backend.
func AdminEffectCatalogMetadata(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	operations := make([]gin.H, 0, len(effectcatalog.OperationCatalog))
	for _, name := range sortedEffectKeys(effectcatalog.OperationCatalog) {
		definition := effectcatalog.OperationCatalog[name]
		operations = append(operations, gin.H{
			"name": name, "sub_types": sortedEffectKeys(definition.SubTypes),
			"triggers": sortedEffectKeys(definition.Triggers), "targets": sortedEffectKeys(definition.Targets),
			"allowed_fields": sortedEffectKeys(definition.Allowed), "required_fields": definition.Required,
		})
	}
	nativeCodes := sortedEffectKeys(effectcatalog.NativeHooks)
	c.JSON(http.StatusOK, gin.H{
		"schema_version": effectcatalog.BlockSchemaVersion,
		"triggers":       sortedEffectKeys(effectcatalog.TriggerCatalog), "targets": sortedEffectKeys(effectcatalog.TargetCatalog),
		"operations": operations, "stats": sortedEffectKeys(effectcatalog.StatCatalog),
		"statuses": sortedEffectKeys(effectcatalog.StatusCatalog), "resources": sortedEffectKeys(effectcatalog.ResourceCatalog),
		"units": sortedEffectKeys(effectcatalog.UnitCatalog), "stack_policies": sortedEffectKeys(effectcatalog.StackPolicyCatalog),
		"condition_fields": sortedEffectKeys(effectcatalog.ConditionFields), "condition_operators": sortedEffectKeys(effectcatalog.ConditionOperators),
		"native_codes": nativeCodes,
	})
}

const (
	maxGuEffectPreviewBytes = 256 << 10
	maxGuCatalogEntries     = 100_000
	maxGuCatalogEffectBytes = 32 << 20
)

type effectPreviewRequest struct {
	Source  string          `json:"source"`
	Key     string          `json:"key"`
	Effects json.RawMessage `json:"effects" binding:"required"`
}

func AdminValidateGuEffects(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxGuEffectPreviewBytes)
	var req effectPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Payload preview vượt giới hạn 256 KiB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload preview không hợp lệ: " + err.Error()})
		return
	}

	issues := effectcatalog.ValidateBlockV2(req.Effects, effectcatalog.Location{Source: req.Source, Key: req.Key})
	c.JSON(http.StatusOK, gin.H{"valid": len(issues) == 0, "schema_version": effectcatalog.BlockSchemaVersion, "issues": issues})
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

// AdminAuditEffectCatalog validates every effect-bearing dictionary through
// the same block-v2 contract used by write and CSV boundaries.
func AdminAuditEffectCatalog(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	type query struct{ source, sql string }
	queries := []query{
		{"gu", `SELECT id || ':' || rank::text, effects FROM gu_dictionary ORDER BY id, rank`},
		{"item", `SELECT id, effects FROM item_dictionary ORDER BY id`},
		{"race", `SELECT key, effects FROM race_dictionary WHERE key != '' ORDER BY key`},
		{"constitution", `SELECT key, effects FROM constitution_dictionary WHERE key != '' ORDER BY key`},
		{"status", `SELECT key, effects FROM status_dictionary ORDER BY key`},
	}
	issues := make([]effectcatalog.ValidationIssue, 0)
	entries := 0
	for _, q := range queries {
		rows, err := db.DB.QueryContext(c.Request.Context(), q.sql)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc effect catalog: " + err.Error()})
			return
		}
		row := 0
		for rows.Next() {
			row++
			var key string
			var raw []byte
			if err := rows.Scan(&key, &raw); err != nil {
				rows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			entries++
			issues = append(issues, effectcatalog.ValidateBlockV2(raw, effectcatalog.Location{Source: q.source, Row: row, Key: key})...)
			if entries > maxGuCatalogEntries || len(issues) > maxGuCatalogEntries {
				rows.Close()
				c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Effect catalog vượt giới hạn audit an toàn"})
				return
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rows.Close()
	}
	c.JSON(http.StatusOK, gin.H{"schema_version": effectcatalog.BlockSchemaVersion, "source": "database", "generated_at": time.Now().UTC(), "valid": len(issues) == 0, "entries": entries, "issues": issues})
}
