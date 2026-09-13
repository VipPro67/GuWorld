package handlers

import (
	"encoding/json"
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
		COALESCE(gu_inventory,'[]'::jsonb), COALESCE(item_inventory,'{}'::jsonb),
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
			&r.GuInventory, &r.ItemInventory, &r.DaoMarks, &r.PathMasteries,
			&r.AIProfile, &r.LootTableID, &r.Description, &r.IsActive,
			&r.IsCapturable, &r.CaptureChance, &r.PetHungerPerHour); err != nil {
			continue
		}
		items = append(items, r)
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
	if req.ID == "" || req.Name == "" || req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id, name, type là bắt buộc"})
		return
	}
	if req.MaxHP <= 0 {
		req.MaxHP = 100
	}
	if req.AIProfile == "" {
		req.AIProfile = "beast_basic"
	}
	if len(req.GuInventory) == 0 {
		req.GuInventory = json.RawMessage(`[]`)
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
	if req.CaptureChance <= 0 {
		req.CaptureChance = 0.20
	}
	if req.PetHungerPerHour <= 0 {
		req.PetHungerPerHour = 2.0
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isCapturable := false
	if req.IsCapturable != nil {
		isCapturable = *req.IsCapturable
	}
	// loot_table_id FK references loot_tables(id); normalize '' to NULL so
	// monsters without a loot table remain insertable.
	var lootTableID interface{}
	if strings.TrimSpace(req.LootTableID) != "" {
		lootTableID = req.LootTableID
	}

	_, err := db.DB.Exec(`INSERT INTO monster_templates(id,name,type,max_hp,base_attack,base_defense,gu_inventory,item_inventory,dao_marks,path_masteries,ai_profile,loot_table_id,description,is_active,is_capturable,capture_chance,pet_hunger_per_hour) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		req.ID, req.Name, req.Type, req.MaxHP, req.BaseAttack, req.BaseDefense,
		req.GuInventory, req.ItemInventory, req.DaoMarks, req.PathMasteries,
		req.AIProfile, lootTableID, req.Description, isActive, isCapturable,
		req.CaptureChance, req.PetHungerPerHour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo: " + err.Error()})
		return
	}
	models.LoadMonsterRegistry(db.DB)
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
	sets := []string{}
	args := []interface{}{}
	idx := 1
	if req.Name != "" {
		sets = append(sets, "name=$"+strconv.Itoa(idx))
		args = append(args, req.Name)
		idx++
	}
	if req.Type != "" {
		sets = append(sets, "type=$"+strconv.Itoa(idx))
		args = append(args, req.Type)
		idx++
	}
	if req.MaxHP > 0 {
		sets = append(sets, "max_hp=$"+strconv.Itoa(idx))
		args = append(args, req.MaxHP)
		idx++
	}
	if req.BaseAttack > 0 {
		sets = append(sets, "base_attack=$"+strconv.Itoa(idx))
		args = append(args, req.BaseAttack)
		idx++
	}
	if req.BaseDefense > 0 {
		sets = append(sets, "base_defense=$"+strconv.Itoa(idx))
		args = append(args, req.BaseDefense)
		idx++
	}
	if req.AIProfile != "" {
		sets = append(sets, "ai_profile=$"+strconv.Itoa(idx))
		args = append(args, req.AIProfile)
		idx++
	}
	if req.LootTableID != "" || true {
		sets = append(sets, "loot_table_id=$"+strconv.Itoa(idx))
		args = append(args, req.LootTableID)
		idx++
	}
	if req.Description != "" || true {
		sets = append(sets, "description=$"+strconv.Itoa(idx))
		args = append(args, req.Description)
		idx++
	}
	if req.GuInventory != nil {
		sets = append(sets, "gu_inventory=$"+strconv.Itoa(idx))
		args = append(args, req.GuInventory)
		idx++
	}
	if req.ItemInventory != nil {
		sets = append(sets, "item_inventory=$"+strconv.Itoa(idx))
		args = append(args, req.ItemInventory)
		idx++
	}
	if req.DaoMarks != nil {
		sets = append(sets, "dao_marks=$"+strconv.Itoa(idx))
		args = append(args, req.DaoMarks)
		idx++
	}
	if req.PathMasteries != nil {
		sets = append(sets, "path_masteries=$"+strconv.Itoa(idx))
		args = append(args, req.PathMasteries)
		idx++
	}
	if req.IsActive != nil {
		sets = append(sets, "is_active=$"+strconv.Itoa(idx))
		args = append(args, *req.IsActive)
		idx++
	}
	if req.IsCapturable != nil {
		sets = append(sets, "is_capturable=$"+strconv.Itoa(idx))
		args = append(args, *req.IsCapturable)
		idx++
	}
	if req.CaptureChance > 0 {
		sets = append(sets, "capture_chance=$"+strconv.Itoa(idx))
		args = append(args, req.CaptureChance)
		idx++
	}
	if req.PetHungerPerHour > 0 {
		sets = append(sets, "pet_hunger_per_hour=$"+strconv.Itoa(idx))
		args = append(args, req.PetHungerPerHour)
		idx++
	}
	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu"})
		return
	}
	sets = append(sets, "updated_at=NOW()")
	args = append(args, monsterID)
	_, err := db.DB.Exec(`UPDATE monster_templates SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(idx), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
		return
	}
	models.LoadMonsterRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.update_monster", "id=%s", monsterID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật quái vật"})
}

func AdminDeleteMonster(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	monsterID := c.Param("id")
	_, err := db.DB.Exec(`DELETE FROM monster_templates WHERE id=$1`, monsterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}
	models.LoadMonsterRegistry(db.DB)
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
			continue
		}
		items = append(items, r)
	}
	c.JSON(http.StatusOK, items)
}

type AdminLootTableInput struct {
	ID      string          `json:"id"`
	Name    string          `json:"name"`
	Entries json.RawMessage `json:"entries"`
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
	if req.ID == "" || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id và name là bắt buộc"})
		return
	}
	if len(req.Entries) == 0 {
		req.Entries = json.RawMessage(`[]`)
	}
	req.Entries = json.RawMessage(db.FilterImmortalLootEntries(string(req.Entries)))

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
	sets := []string{}
	args := []interface{}{}
	idx := 1
	if req.Name != "" {
		sets = append(sets, "name=$"+strconv.Itoa(idx))
		args = append(args, req.Name)
		idx++
	}
	if req.Entries != nil {
		req.Entries = json.RawMessage(db.FilterImmortalLootEntries(string(req.Entries)))
		sets = append(sets, "entries=$"+strconv.Itoa(idx)+"::jsonb")
		args = append(args, req.Entries)
		idx++
	}
	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu"})
		return
	}
	sets = append(sets, "updated_at=NOW()")
	args = append(args, tableID)
	_, err := db.DB.Exec(`UPDATE loot_tables SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(idx), args...)
	if err != nil {
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
	_, err := db.DB.Exec(`DELETE FROM loot_tables WHERE id=$1`, tableID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_loot_table", "id=%s", tableID)
	cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa bảng loot"})
}
