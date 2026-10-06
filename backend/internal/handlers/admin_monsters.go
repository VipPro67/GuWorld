package handlers

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type OnlinePlayerRow struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Rank         int        `json:"rank"`
	SubRank      int        `json:"sub_rank"`
	IsOnline     bool       `json:"is_online"`
	LastActivity *time.Time `json:"last_activity"`
}

func AdminListOnlinePlayers(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	minutesStr := c.DefaultQuery("minutes", "5")
	minutes, err := strconv.Atoi(minutesStr)
	if err != nil || minutes < 1 {
		minutes = 5
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, username, rank, sub_rank, last_activity
		FROM players
		WHERE last_activity IS NOT NULL
		ORDER BY last_activity DESC
		LIMIT 100
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	threshold := time.Now().Add(-time.Duration(minutes) * time.Minute)
	result := []OnlinePlayerRow{}
	for rows.Next() {
		var r OnlinePlayerRow
		if err := rows.Scan(&r.ID, &r.Username, &r.Rank, &r.SubRank, &r.LastActivity); err != nil {
			continue
		}
		if r.LastActivity != nil && r.LastActivity.After(threshold) {
			r.IsOnline = true
		}
		result = append(result, r)
	}

	c.JSON(http.StatusOK, result)
}

// -------- Admin: Monster Template CRUD --------

func AdminListMonsters(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, name, type, max_hp, base_attack, base_defense,
		COALESCE(gu_inventory,'[]'::jsonb), COALESCE(gu_pool,'[]'::jsonb), COALESCE(item_inventory,'{}'::jsonb),
		COALESCE(dao_marks,'{}'::jsonb), COALESCE(path_masteries,'{}'::jsonb),
		COALESCE(ai_profile,'beast_basic'), COALESCE(loot_table_id,''),
		COALESCE(description,''), COALESCE(is_active,true),
		COALESCE(is_capturable,false), COALESCE(capture_chance,0.20), COALESCE(pet_hunger_per_hour,2.0)
		FROM monster_templates ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type MonsterRow struct {
		ID               string          `json:"id"`
		Name             string          `json:"name"`
		Type             string          `json:"type"`
		MaxHP            float64         `json:"max_hp"`
		BaseAttack       float64         `json:"base_attack"`
		BaseDefense      float64         `json:"base_defense"`
		GuInventory      json.RawMessage `json:"gu_inventory"`
		GuPool           json.RawMessage `json:"gu_pool"`
		ItemInventory    json.RawMessage `json:"item_inventory"`
		DaoMarks         json.RawMessage `json:"dao_marks"`
		PathMasteries    json.RawMessage `json:"path_masteries"`
		AIProfile        string          `json:"ai_profile"`
		LootTableID      string          `json:"loot_table_id"`
		Description      string          `json:"description"`
		IsActive         bool            `json:"is_active"`
		IsCapturable     bool            `json:"is_capturable"`
		CaptureChance    float64         `json:"capture_chance"`
		PetHungerPerHour float64         `json:"pet_hunger_per_hour"`
	}
	items := []MonsterRow{}
	for rows.Next() {
		var r MonsterRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.MaxHP, &r.BaseAttack, &r.BaseDefense,
			&r.GuInventory, &r.GuPool, &r.ItemInventory, &r.DaoMarks, &r.PathMasteries,
			&r.AIProfile, &r.LootTableID, &r.Description, &r.IsActive,
			&r.IsCapturable, &r.CaptureChance, &r.PetHungerPerHour); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ dữ liệu quái vật"})
			return
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách quái vật"})
		return
	}
	c.JSON(http.StatusOK, items)
}

type AdminMonsterInput struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Type             string          `json:"type"`
	MaxHP            float64         `json:"max_hp"`
	BaseAttack       float64         `json:"base_attack"`
	BaseDefense      float64         `json:"base_defense"`
	GuInventory      json.RawMessage `json:"gu_inventory"`
	GuPool           json.RawMessage `json:"gu_pool"`
	ItemInventory    json.RawMessage `json:"item_inventory"`
	DaoMarks         json.RawMessage `json:"dao_marks"`
	PathMasteries    json.RawMessage `json:"path_masteries"`
	AIProfile        string          `json:"ai_profile"`
	LootTableID      string          `json:"loot_table_id"`
	Description      string          `json:"description"`
	IsActive         *bool           `json:"is_active"`
	IsCapturable     *bool           `json:"is_capturable"`
	CaptureChance    float64         `json:"capture_chance"`
	PetHungerPerHour float64         `json:"pet_hunger_per_hour"`
}

func normalizeAdminMonsterInput(c *gin.Context, req *AdminMonsterInput, creating bool) bool {
	req.ID, req.Name, req.Type = strings.TrimSpace(req.ID), strings.TrimSpace(req.Name), strings.TrimSpace(req.Type)
	if req.ID == "" || len(req.ID) > 50 || req.Name == "" || len(req.Name) > 100 || req.Type == "" || len(req.Type) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID (1-50), tên (1-100) và loại quái vật (1-50) là bắt buộc"})
		return false
	}
	if math.IsNaN(req.MaxHP) || math.IsInf(req.MaxHP, 0) || req.MaxHP <= 0 ||
		math.IsNaN(req.BaseAttack) || math.IsInf(req.BaseAttack, 0) || req.BaseAttack < 0 ||
		math.IsNaN(req.BaseDefense) || math.IsInf(req.BaseDefense, 0) || req.BaseDefense < 0 ||
		math.IsNaN(req.CaptureChance) || math.IsInf(req.CaptureChance, 0) || req.CaptureChance < 0 || req.CaptureChance > 1 ||
		math.IsNaN(req.PetHungerPerHour) || math.IsInf(req.PetHungerPerHour, 0) || req.PetHungerPerHour < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ số HP, tấn công, phòng thủ, tỉ lệ bắt hoặc mức đói không hợp lệ"})
		return false
	}
	if req.AIProfile == "" {
		req.AIProfile = "beast_basic"
	}
	if creating {
		if req.IsActive == nil {
			value := true
			req.IsActive = &value
		}
		if req.IsCapturable == nil {
			value := false
			req.IsCapturable = &value
		}
	} else if req.IsActive == nil || req.IsCapturable == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu trạng thái hoạt động hoặc trạng thái có thể bắt"})
		return false
	}
	if len(req.GuInventory) == 0 {
		req.GuInventory = json.RawMessage(`[]`)
	}
	if len(req.GuPool) == 0 {
		req.GuPool = json.RawMessage(`[]`)
	}
	if len(req.ItemInventory) == 0 {
		req.ItemInventory = json.RawMessage(`{}`)
	}
	if len(req.DaoMarks) == 0 {
		req.DaoMarks = json.RawMessage(`{}`)
	}
	if len(req.PathMasteries) == 0 {
		req.PathMasteries = json.RawMessage(`{}`)
	}
	var guInventory, guPool []json.RawMessage
	var itemInventory, daoMarks, pathMasteries map[string]json.RawMessage
	if json.Unmarshal(req.GuInventory, &guInventory) != nil || guInventory == nil ||
		json.Unmarshal(req.GuPool, &guPool) != nil || guPool == nil ||
		json.Unmarshal(req.ItemInventory, &itemInventory) != nil || itemInventory == nil ||
		json.Unmarshal(req.DaoMarks, &daoMarks) != nil || daoMarks == nil ||
		json.Unmarshal(req.PathMasteries, &pathMasteries) != nil || pathMasteries == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gu inventory và gu pool phải là mảng JSON; item inventory, dao marks và path masteries phải là đối tượng JSON"})
		return false
	}
	return true
}

func AdminCreateMonster(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req AdminMonsterInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if !normalizeAdminMonsterInput(c, &req, true) {
		return
	}
	// loot_table_id FK references loot_tables(id); normalize '' to NULL so
	// monsters without a loot table remain insertable.
	var lootTableID interface{}
	if strings.TrimSpace(req.LootTableID) != "" {
		lootTableID = req.LootTableID
	}

	_, err := db.DB.Exec(`INSERT INTO monster_templates(id,name,type,max_hp,base_attack,base_defense,gu_inventory,gu_pool,item_inventory,dao_marks,path_masteries,ai_profile,loot_table_id,description,is_active,is_capturable,capture_chance,pet_hunger_per_hour) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		req.ID, req.Name, req.Type, req.MaxHP, req.BaseAttack, req.BaseDefense,
		req.GuInventory, req.GuPool, req.ItemInventory, req.DaoMarks, req.PathMasteries,
		req.AIProfile, lootTableID, req.Description, *req.IsActive, *req.IsCapturable,
		req.CaptureChance, req.PetHungerPerHour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo: " + err.Error()})
		return
	}
	if err := models.LoadMonsterRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu quái vật nhưng không thể tải lại catalog: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_monster", "id=%s name=%s", req.ID, req.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo quái vật"})
}

func AdminUpdateMonster(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	monsterID := c.Param("id")
	var req AdminMonsterInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.ID != "" && req.ID != monsterID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi ID quái vật; hãy tạo quái vật mới"})
		return
	}
	req.ID = monsterID
	if !normalizeAdminMonsterInput(c, &req, false) {
		return
	}
	var lootTableID interface{}
	if strings.TrimSpace(req.LootTableID) != "" {
		lootTableID = req.LootTableID
	}
	result, err := db.DB.Exec(`UPDATE monster_templates SET name=$1,type=$2,max_hp=$3,base_attack=$4,base_defense=$5,gu_inventory=$6,gu_pool=$7,item_inventory=$8,dao_marks=$9,path_masteries=$10,ai_profile=$11,loot_table_id=$12,description=$13,is_active=$14,is_capturable=$15,capture_chance=$16,pet_hunger_per_hour=$17,updated_at=NOW() WHERE id=$18`,
		req.Name, req.Type, req.MaxHP, req.BaseAttack, req.BaseDefense, req.GuInventory, req.GuPool, req.ItemInventory, req.DaoMarks, req.PathMasteries, req.AIProfile, lootTableID, req.Description, *req.IsActive, *req.IsCapturable, req.CaptureChance, req.PetHungerPerHour, monsterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
		return
	}
	updated, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận quái vật đã được cập nhật"})
		return
	}
	if updated == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy quái vật"})
		return
	}
	if err := models.LoadMonsterRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu quái vật nhưng không thể tải lại catalog: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_monster", "id=%s", monsterID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật quái vật"})
}

func AdminDeleteMonster(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	monsterID := c.Param("id")
	result, err := db.DB.Exec(`DELETE FROM monster_templates WHERE id=$1`, monsterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận quái vật đã được xóa"})
		return
	}
	if deleted == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy quái vật"})
		return
	}
	if err := models.LoadMonsterRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã xóa quái vật nhưng không thể tải lại catalog: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_monster", "id=%s", monsterID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa quái vật"})
}

// -------- Admin: Loot Table CRUD --------

func AdminListLootTables(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, name, COALESCE(entries,'[]'::jsonb) FROM loot_tables ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type LootTableRow struct {
		ID      string          `json:"id"`
		Name    string          `json:"name"`
		Entries json.RawMessage `json:"entries"`
	}
	items := []LootTableRow{}
	for rows.Next() {
		var r LootTableRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Entries); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ danh sách bảng loot"})
			return
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách bảng loot"})
		return
	}
	c.JSON(http.StatusOK, items)
}

type AdminLootTableInput struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Entries json.RawMessage `json:"entries"`
}

func validateAdminLootTableInput(c *gin.Context, req *AdminLootTableInput) bool {
	req.ID, req.Name = strings.TrimSpace(req.ID), strings.TrimSpace(req.Name)
	if req.ID == "" || len(req.ID) > 50 || req.Name == "" || len(req.Name) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID (1-50) và tên bảng loot (1-100) là bắt buộc"})
		return false
	}
	if len(req.Entries) == 0 {
		req.Entries = json.RawMessage(`[]`)
	}
	var entries []models.LootEntry
	if json.Unmarshal(req.Entries, &entries) != nil || entries == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách loot phải là mảng JSON hợp lệ"})
		return false
	}
	req.Entries = json.RawMessage(db.FilterImmortalLootEntries(string(req.Entries)))
	return true
}

func AdminCreateLootTable(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req AdminLootTableInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if !validateAdminLootTableInput(c, &req) {
		return
	}

	_, err := db.DB.Exec(`INSERT INTO loot_tables(id,name,entries) VALUES($1,$2,$3::jsonb)`,
		req.ID, req.Name, req.Entries)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_loot_table", "id=%s name=%s", req.ID, req.Name)
	cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo bảng loot"})
}

func AdminUpdateLootTable(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	tableID := c.Param("id")
	var req AdminLootTableInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.ID != "" && req.ID != tableID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi ID bảng loot; hãy tạo bảng mới"})
		return
	}
	req.ID = tableID
	if !validateAdminLootTableInput(c, &req) {
		return
	}
	var updatedID string
	err := db.DB.QueryRowContext(c.Request.Context(), `UPDATE loot_tables SET name=$1,entries=$2::jsonb,updated_at=NOW() WHERE id=$3 RETURNING id`, req.Name, req.Entries, tableID).Scan(&updatedID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy bảng loot"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_loot_table", "id=%s", tableID)
	cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật bảng loot"})
}

func AdminDeleteLootTable(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	tableID := c.Param("id")
	result, err := db.DB.Exec(`DELETE FROM loot_tables WHERE id=$1`, tableID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận bảng loot đã được xóa"})
		return
	}
	if deleted == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy bảng loot"})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_loot_table", "id=%s", tableID)
	cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa bảng loot"})
}
