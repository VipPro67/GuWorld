package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

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

func validateDictionaryIdentity(c *gin.Context, key, name string) bool {
	key, name = strings.TrimSpace(key), strings.TrimSpace(name)
	if key == "" || len(key) > 100 || name == "" || len(name) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã định danh (1-100) và tên (1-200) là bắt buộc"})
		return false
	}
	return true
}

func validateBlockedActions(c *gin.Context, raw *json.RawMessage) bool {
	if len(*raw) == 0 {
		*raw = json.RawMessage(`[]`)
	}
	var actions []string
	if json.Unmarshal(*raw, &actions) != nil || actions == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hành động bị chặn phải là mảng chuỗi JSON"})
		return false
	}
	return true
}

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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ từ điển thể chất"})
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc từ điển thể chất"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateConstitution(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req ConstitutionRow
	if err := c.ShouldBindJSON(&req); err != nil || !validateDictionaryIdentity(c, req.Key, req.Name) {
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu thể chất không hợp lệ"})
		}
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
	if err := gamefx.LoadConstitutionRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu nhưng không thể tải lại từ điển thể chất: " + err.Error()})
		return
	}
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
	if req.Key != "" && req.Key != key {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi key thể chất"})
		return
	}
	if !validateDictionaryIdentity(c, key, req.Name) {
		return
	}
	if len(req.Effects) == 0 {
		req.Effects = append(json.RawMessage(nil), emptyBlockEffects...)
	}
	if req.Tier < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tier phải từ 1 trở lên"})
		return
	}
	if !validateDictionaryEffects(c, "constitution", key, req.Effects) {
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE constitution_dictionary SET name = $1, tier = $2, description = $3, effects = $4 WHERE key = $5`,
		req.Name, req.Tier, req.Description, req.Effects, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận thể chất đã cập nhật"})
		return
	}
	if updated == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thể chất"})
		return
	}
	if err := gamefx.LoadConstitutionRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu nhưng không thể tải lại từ điển thể chất: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_constitution", "key=%s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật thể chất"})
}

func AdminDeleteConstitution(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	key := c.Param("key")
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM constitution_dictionary WHERE key = $1`, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận thể chất đã xóa"})
		return
	}
	if deleted == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thể chất"})
		return
	}
	if err := gamefx.LoadConstitutionRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã xóa nhưng không thể tải lại từ điển thể chất: " + err.Error()})
		return
	}
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ từ điển chủng tộc"})
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc từ điển chủng tộc"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateRace(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req RaceRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu chủng tộc không hợp lệ"})
		return
	}
	if !validateDictionaryIdentity(c, req.Key, req.Name) {
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
	if err := gamefx.LoadRaceRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu nhưng không thể tải lại từ điển chủng tộc: " + err.Error()})
		return
	}
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
	if req.Key != "" && req.Key != key {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi key chủng tộc"})
		return
	}
	if !validateDictionaryIdentity(c, key, req.Name) {
		return
	}
	if len(req.Effects) == 0 {
		req.Effects = append(json.RawMessage(nil), emptyBlockEffects...)
	}
	if !validateDictionaryEffects(c, "race", key, req.Effects) {
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE race_dictionary SET name = $1, description = $2, effects = $3 WHERE key = $4`,
		req.Name, req.Description, req.Effects, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận chủng tộc đã cập nhật"})
		return
	}
	if updated == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy chủng tộc"})
		return
	}
	if err := gamefx.LoadRaceRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu nhưng không thể tải lại từ điển chủng tộc: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_race", "key=%s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật chủng tộc"})
}

func AdminDeleteRace(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	key := c.Param("key")
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM race_dictionary WHERE key = $1`, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận chủng tộc đã xóa"})
		return
	}
	if deleted == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy chủng tộc"})
		return
	}
	if err := gamefx.LoadRaceRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã xóa nhưng không thể tải lại từ điển chủng tộc: " + err.Error()})
		return
	}
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ từ điển trạng thái"})
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc từ điển trạng thái"})
		return
	}
	c.JSON(http.StatusOK, list)
}

func AdminCreateStatus(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req StatusDictRow
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu trạng thái không hợp lệ"})
		return
	}
	if !validateDictionaryIdentity(c, req.Key, req.Name) {
		return
	}
	if req.DefaultDurationMinutes < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời gian trạng thái không được âm"})
		return
	}
	if len(req.Effects) == 0 {
		req.Effects = append(json.RawMessage(nil), emptyBlockEffects...)
	}
	if !validateDictionaryEffects(c, "status", req.Key, req.Effects) {
		return
	}
	if !validateBlockedActions(c, &req.BlockedActions) {
		return
	}
	_, err := db.DB.ExecContext(c.Request.Context(), `INSERT INTO status_dictionary (key, name, description, effects, default_duration_minutes, blocked_actions) VALUES ($1, $2, $3, $4, $5, $6)`,
		req.Key, req.Name, req.Description, req.Effects, req.DefaultDurationMinutes, req.BlockedActions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := gamefx.LoadStatusRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu nhưng không thể tải lại từ điển trạng thái: " + err.Error()})
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
	if req.Key != "" && req.Key != key {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi key trạng thái"})
		return
	}
	if !validateDictionaryIdentity(c, key, req.Name) {
		return
	}
	if req.DefaultDurationMinutes < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời gian trạng thái không được âm"})
		return
	}
	if len(req.Effects) == 0 {
		req.Effects = append(json.RawMessage(nil), emptyBlockEffects...)
	}
	if !validateDictionaryEffects(c, "status", key, req.Effects) {
		return
	}
	if !validateBlockedActions(c, &req.BlockedActions) {
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE status_dictionary SET name = $1, description = $2, effects = $3, default_duration_minutes = $4, blocked_actions = $5 WHERE key = $6`,
		req.Name, req.Description, req.Effects, req.DefaultDurationMinutes, req.BlockedActions, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận trạng thái đã cập nhật"})
		return
	}
	if updated == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy trạng thái"})
		return
	}
	if err := gamefx.LoadStatusRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu nhưng không thể tải lại từ điển trạng thái: " + err.Error()})
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
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM status_dictionary WHERE key = $1`, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận trạng thái đã xóa"})
		return
	}
	if deleted == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy trạng thái"})
		return
	}
	if err := gamefx.LoadStatusRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã xóa nhưng không thể tải lại từ điển trạng thái: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_status", "key=%s", key)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa trạng thái"})
}
