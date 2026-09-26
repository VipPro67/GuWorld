package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func validateDictionaryEffects(c *gin.Context, source, key string, raw json.RawMessage) bool {
	issues := effectcatalog.ValidateBlockV2(raw, effectcatalog.Location{Source: source, Key: key})
	if len(issues) == 0 {
		return true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "Effect Block v2 không hợp lệ", "issues": issues})
	return false
}

var emptyBlockEffects = json.RawMessage(`{"schema_version":2,"blocks":[]}`)

// --- Constitution Dictionary Handlers ---

type ConstitutionRow struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Tier        int             `json:"tier"`
	Description string          `json:"description"`
	Effects     json.RawMessage `json:"effects"`
}

func AdminListConstitutions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT key, COALESCE(name,''), COALESCE(tier,1), COALESCE(description,''), COALESCE(effects,'{}') FROM constitution_dictionary WHERE key != '' ORDER BY tier ASC, key ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []ConstitutionRow{}
	for rows.Next() {
		var r ConstitutionRow
		if err := rows.Scan(&r.Key, &r.Name, &r.Tier, &r.Description, &r.Effects); err != nil {
			continue
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateConstitution(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req ConstitutionRow
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key và Tên thể chất không được để trống"})
		return
	}
	if len(req.Effects) == 0 {
		req.Effects = append(json.RawMessage(nil), emptyBlockEffects...)
	}
	if !validateDictionaryEffects(c, "constitution", req.Key, req.Effects) {
		return
	}
	if req.Tier < 1 {
		req.Tier = 1
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `INSERT INTO constitution_dictionary (key, name, tier, description, effects) VALUES ($1, $2, $3, $4, $5)`,
		req.Key, req.Name, req.Tier, req.Description, req.Effects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = gamefx.LoadConstitutionRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.create_constitution", "key=%s", req.Key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo thể chất mới"})
}

func AdminUpdateConstitution(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	key := c.Param("key")
	var req ConstitutionRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validateDictionaryEffects(c, "constitution", key, req.Effects) {
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `UPDATE constitution_dictionary SET name = COALESCE(NULLIF($1,''), name), tier = $2, description = $3, effects = $4 WHERE key = $5`,
		req.Name, req.Tier, req.Description, req.Effects, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = gamefx.LoadConstitutionRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.update_constitution", "key=%s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật thể chất"})
}

func AdminDeleteConstitution(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	key := c.Param("key")
	_, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM constitution_dictionary WHERE key = $1`, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = gamefx.LoadConstitutionRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.delete_constitution", "key=%s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thể chất"})
}

// --- Race Dictionary Handlers ---

type RaceRow struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Effects     json.RawMessage `json:"effects"`
}

func AdminListRaces(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT key, COALESCE(name,''), COALESCE(description,''), COALESCE(effects,'{}') FROM race_dictionary WHERE key != '' ORDER BY key ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []RaceRow{}
	for rows.Next() {
		var r RaceRow
		if err := rows.Scan(&r.Key, &r.Name, &r.Description, &r.Effects); err != nil {
			continue
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateRace(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req RaceRow
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key và Tên chủng tộc không được để trống"})
		return
	}
	if len(req.Effects) == 0 {
		req.Effects = append(json.RawMessage(nil), emptyBlockEffects...)
	}
	if !validateDictionaryEffects(c, "race", req.Key, req.Effects) {
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `INSERT INTO race_dictionary (key, name, description, effects) VALUES ($1, $2, $3, $4)`,
		req.Key, req.Name, req.Description, req.Effects)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = gamefx.LoadRaceRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.create_race", "key=%s", req.Key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo chủng tộc mới"})
}

func AdminUpdateRace(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	key := c.Param("key")
	var req RaceRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validateDictionaryEffects(c, "race", key, req.Effects) {
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `UPDATE race_dictionary SET name = COALESCE(NULLIF($1,''), name), description = $2, effects = $3 WHERE key = $4`,
		req.Name, req.Description, req.Effects, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = gamefx.LoadRaceRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.update_race", "key=%s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật chủng tộc"})
}

func AdminDeleteRace(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	key := c.Param("key")
	_, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM race_dictionary WHERE key = $1`, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = gamefx.LoadRaceRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.delete_race", "key=%s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa chủng tộc"})
}

// --- Status Dictionary Handlers ---

type StatusDictRow struct {
	Key                    string          `json:"key"`
	Name                   string          `json:"name"`
	Description            string          `json:"description"`
	Effects                json.RawMessage `json:"effects"`
	DefaultDurationMinutes int             `json:"default_duration_minutes"`
	BlockedActions         json.RawMessage `json:"blocked_actions"`
}

func AdminListStatuses(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT key, COALESCE(name,''), COALESCE(description,''), COALESCE(effects,'{}'), COALESCE(default_duration_minutes,0), COALESCE(blocked_actions,'[]') FROM status_dictionary ORDER BY key ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	list := []StatusDictRow{}
	for rows.Next() {
		var r StatusDictRow
		if err := rows.Scan(&r.Key, &r.Name, &r.Description, &r.Effects, &r.DefaultDurationMinutes, &r.BlockedActions); err != nil {
			continue
		}
		list = append(list, r)
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateStatus(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req StatusDictRow
	if err := c.ShouldBindJSON(&req); err != nil || req.Key == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Key và Tên trạng thái không được để trống"})
		return
	}
	if len(req.Effects) == 0 {
		req.Effects = append(json.RawMessage(nil), emptyBlockEffects...)
	}
	if !validateDictionaryEffects(c, "status", req.Key, req.Effects) {
		return
	}
	if len(req.BlockedActions) == 0 {
		req.BlockedActions = json.RawMessage(`[]`)
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `INSERT INTO status_dictionary (key, name, description, effects, default_duration_minutes, blocked_actions) VALUES ($1, $2, $3, $4, $5, $6)`,
		req.Key, req.Name, req.Description, req.Effects, req.DefaultDurationMinutes, req.BlockedActions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_status", "key=%s", req.Key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo trạng thái mới"})
}

func AdminUpdateStatus(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	key := c.Param("key")
	var req StatusDictRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !validateDictionaryEffects(c, "status", key, req.Effects) {
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `UPDATE status_dictionary SET name = COALESCE(NULLIF($1,''), name), description = $2, effects = $3, default_duration_minutes = $4, blocked_actions = $5 WHERE key = $6`,
		req.Name, req.Description, req.Effects, req.DefaultDurationMinutes, req.BlockedActions, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_status", "key=%s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật trạng thái"})
}

func AdminDeleteStatus(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	key := c.Param("key")
	_, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM status_dictionary WHERE key = $1`, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_status", "key=%s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa trạng thái"})
}
