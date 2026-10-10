package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// -------- Admin: Gu Dictionary CRUD --------

func AdminListGuItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, name, rank, path, gu_type,
		base_mana_cost, effects, food_types, hunger_rate, max_hunger,
		COALESCE(description,''), COALESCE(target_type,'SINGLE'), COALESCE(rarity,0),
		COALESCE(primary_role,''), COALESCE(combat_roles,'[]'::jsonb),
		COALESCE(combat_tags,'[]'::jsonb), COALESCE(support_tags,'[]'::jsonb), COALESCE(base_price,0),
		COALESCE(image_url,'')
		FROM gu_dictionary ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type GuRow struct {
		ID           string          `json:"id"`
		Name         string          `json:"name"`
		Rank         int             `json:"rank"`
		Path         json.RawMessage `json:"path"`
		GuType       string          `json:"gu_type"`
		BaseManaCost float64         `json:"base_mana_cost"`
		Effects      json.RawMessage `json:"effects"`
		FoodTypes    json.RawMessage `json:"food_types"`
		HungerRate   int             `json:"hunger_rate"`
		MaxHunger    int             `json:"max_hunger"`
		Description  string          `json:"description"`
		TargetType   string          `json:"target_type"`
		Rarity       int             `json:"rarity"`
		PrimaryRole  string          `json:"primary_role"`
		CombatRoles  json.RawMessage `json:"combat_roles"`
		CombatTags   json.RawMessage `json:"combat_tags"`
		SupportTags  json.RawMessage `json:"support_tags"`
		BasePrice    float64         `json:"base_price"`
		ImageURL     string          `json:"image_url"`
	}

	items := []GuRow{}
	for rows.Next() {
		var r GuRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Rank, &r.Path, &r.GuType,
			&r.BaseManaCost, &r.Effects, &r.FoodTypes, &r.HungerRate, &r.MaxHunger,
			&r.Description, &r.TargetType, &r.Rarity, &r.PrimaryRole,
			&r.CombatRoles, &r.CombatTags, &r.SupportTags, &r.BasePrice, &r.ImageURL); err != nil {
			continue
		}
		items = append(items, r)
	}
	c.JSON(http.StatusOK, items)
}

type AdminGuInput struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Rank         int             `json:"rank"`
	Path         []string        `json:"path"`
	GuType       string          `json:"gu_type"`
	BaseManaCost float64         `json:"base_mana_cost"`
	Effects      json.RawMessage `json:"effects"`
	FoodTypes    []string        `json:"food_types"`
	HungerRate   int             `json:"hunger_rate"`
	MaxHunger    int             `json:"max_hunger"`
	Description  string          `json:"description"`
	TargetType   string          `json:"target_type"`
	Rarity       int             `json:"rarity"`
	PrimaryRole  string          `json:"primary_role"`
	CombatRoles  []string        `json:"combat_roles"`
	CombatTags   []string        `json:"combat_tags"`
	SupportTags  []string        `json:"support_tags"`
	BasePrice    float64         `json:"base_price"`
	ImageURL     string          `json:"image_url"`
}

func guValidate(req *AdminGuInput) (bool, string) {
	if req.ID == "" {
		return false, "ID là bắt buộc"
	}
	if req.Name == "" {
		return false, "Tên là bắt buộc"
	}
	if req.Rank < 1 {
		return false, "Rank phải >= 1"
	}
	validTypes := map[string]bool{"Active": true, "Buff": true, "Consumable": true, "Control": true, "Defense": true, "Healing": true, "Investigation": true, "Movement": true, "Passive": true, "Refinement": true, "Special": true, "Storage": true, "Summon": true, "Support": true, "Transformation": true, "Vital": true}
	if !validTypes[req.GuType] {
		return false, "gu_type không hợp lệ"
	}
	return true, ""
}

func marshalArray(arr []string) json.RawMessage {
	if arr == nil {
		arr = []string{}
	}
	b, _ := json.Marshal(arr)
	return b
}

func validateAndNormalizeAdminGuEffects(c *gin.Context, key string, raw *json.RawMessage) bool {
	issues := effectcatalog.ValidateBlockV2(*raw, effectcatalog.Location{Source: "gu", Key: key})
	if len(issues) != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Effect Block v2 không hợp lệ",
			"issues": issues,
		})
		return false
	}
	return true
}

func validateAdminEffects(c *gin.Context, source, key string, raw json.RawMessage) bool {
	issues := effectcatalog.ValidateBlockV2(raw, effectcatalog.Location{Source: source, Key: key})
	if len(issues) == 0 {
		return true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "Effect Block v2 không hợp lệ", "issues": issues})
	return false
}

func guEffectWarnings(c *gin.Context) []effectcatalog.Issue {
	value, ok := c.Get("gu_effect_warnings")
	if !ok {
		return []effectcatalog.Issue{}
	}
	warnings, ok := value.([]effectcatalog.Issue)
	if !ok {
		return []effectcatalog.Issue{}
	}
	return warnings
}

func AdminCreateGuItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req AdminGuInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if valid, msg := guValidate(&req); !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if req.MaxHunger <= 0 {
		req.MaxHunger = 100
	}
	if req.TargetType == "" {
		req.TargetType = "SINGLE"
	}
	if len(req.Effects) == 0 {
		req.Effects = json.RawMessage(`[]`)
	}
	if !validateAndNormalizeAdminGuEffects(c, req.ID, &req.Effects) {
		return
	}

	_, err := db.DB.Exec(`INSERT INTO gu_dictionary (id,name,rank,path,gu_type,base_mana_cost,effects,food_types,hunger_rate,max_hunger,description,target_type,rarity,primary_role,combat_roles,combat_tags,support_tags,base_price,image_url) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		req.ID, req.Name, req.Rank, marshalArray(req.Path), req.GuType, req.BaseManaCost, req.Effects, marshalArray(req.FoodTypes),
		req.HungerRate, req.MaxHunger, req.Description, req.TargetType, req.Rarity, req.PrimaryRole,
		marshalArray(req.CombatRoles), marshalArray(req.CombatTags), marshalArray(req.SupportTags), req.BasePrice, req.ImageURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo: " + err.Error()})
		return
	}
	if err := models.LoadGuRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã ghi dữ liệu nhưng không thể tải lại catalog Cổ: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_gu", "id=%s name=%s", req.ID, req.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo cổ trùng", "issues": guEffectWarnings(c)})
}

func AdminUpdateGuItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	guID := c.Param("id")
	var req AdminGuInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	oldRankStr := c.Query("old_rank")
	oldRank, _ := strconv.Atoi(oldRankStr)
	if oldRank == 0 {
		oldRank = req.Rank
	}
	// Use existing ID from URL if not in body
	if req.ID == "" {
		req.ID = guID
	}
	if req.Name != "" && len(req.Effects) == 0 {
		req.Effects = json.RawMessage(`[]`)
	}
	if len(req.Effects) > 0 && !validateAndNormalizeAdminGuEffects(c, req.ID, &req.Effects) {
		return
	}

	if req.Name == "" {
		// Partial update — only update provided fields
		sets := []string{}
		args := []interface{}{}
		idx := 1
		if req.Rank > 0 {
			sets = append(sets, "rank=$"+strconv.Itoa(idx))
			args = append(args, req.Rank)
			idx++
		}
		if req.GuType != "" {
			sets = append(sets, "gu_type=$"+strconv.Itoa(idx))
			args = append(args, req.GuType)
			idx++
		}
		if req.BaseManaCost > 0 || true {
			sets = append(sets, "base_mana_cost=$"+strconv.Itoa(idx))
			args = append(args, req.BaseManaCost)
			idx++
		}
		if req.HungerRate > 0 {
			sets = append(sets, "hunger_rate=$"+strconv.Itoa(idx))
			args = append(args, req.HungerRate)
			idx++
		}
		if req.MaxHunger > 0 {
			sets = append(sets, "max_hunger=$"+strconv.Itoa(idx))
			args = append(args, req.MaxHunger)
			idx++
		}
		if req.Path != nil {
			sets = append(sets, "path=$"+strconv.Itoa(idx))
			args = append(args, marshalArray(req.Path))
			idx++
		}
		if req.FoodTypes != nil {
			sets = append(sets, "food_types=$"+strconv.Itoa(idx))
			args = append(args, marshalArray(req.FoodTypes))
			idx++
		}
		if req.CombatRoles != nil {
			sets = append(sets, "combat_roles=$"+strconv.Itoa(idx))
			args = append(args, marshalArray(req.CombatRoles))
			idx++
		}
		if req.CombatTags != nil {
			sets = append(sets, "combat_tags=$"+strconv.Itoa(idx))
			args = append(args, marshalArray(req.CombatTags))
			idx++
		}
		if req.SupportTags != nil {
			sets = append(sets, "support_tags=$"+strconv.Itoa(idx))
			args = append(args, marshalArray(req.SupportTags))
			idx++
		}
		if req.Description != "" || true {
			sets = append(sets, "description=$"+strconv.Itoa(idx))
			args = append(args, req.Description)
			idx++
		}
		if req.TargetType != "" {
			sets = append(sets, "target_type=$"+strconv.Itoa(idx))
			args = append(args, req.TargetType)
			idx++
		}
		if req.Rarity >= 0 || true {
			sets = append(sets, "rarity=$"+strconv.Itoa(idx))
			args = append(args, req.Rarity)
			idx++
		}
		if req.PrimaryRole != "" || true {
			sets = append(sets, "primary_role=$"+strconv.Itoa(idx))
			args = append(args, req.PrimaryRole)
			idx++
		}
		if req.BasePrice > 0 || true {
			sets = append(sets, "base_price=$"+strconv.Itoa(idx))
			args = append(args, req.BasePrice)
			idx++
		}
		if req.ImageURL != "" {
			sets = append(sets, "image_url=$"+strconv.Itoa(idx))
			args = append(args, req.ImageURL)
			idx++
		}
		if len(req.Effects) > 0 {
			sets = append(sets, "effects=$"+strconv.Itoa(idx))
			args = append(args, req.Effects)
			idx++
		}

		if len(sets) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu"})
			return
		}
		args = append(args, guID, oldRank)
		_, err := db.DB.Exec(`UPDATE gu_dictionary SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(idx)+` AND rank=$`+strconv.Itoa(idx+1), args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
			return
		}
	} else {
		if valid, msg := guValidate(&req); !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}
		if req.MaxHunger <= 0 {
			req.MaxHunger = 100
		}
		if req.TargetType == "" {
			req.TargetType = "SINGLE"
		}
		_, err := db.DB.Exec(`UPDATE gu_dictionary SET name=$1,rank=$2,path=$3,gu_type=$4,base_mana_cost=$5,effects=$6,food_types=$7,hunger_rate=$8,max_hunger=$9,description=$10,target_type=$11,rarity=$12,primary_role=$13,combat_roles=$14,combat_tags=$15,support_tags=$16,base_price=$17,image_url=$18 WHERE id=$19 AND rank=$20`,
			req.Name, req.Rank, marshalArray(req.Path), req.GuType, req.BaseManaCost, req.Effects, marshalArray(req.FoodTypes),
			req.HungerRate, req.MaxHunger, req.Description, req.TargetType, req.Rarity, req.PrimaryRole,
			marshalArray(req.CombatRoles), marshalArray(req.CombatTags), marshalArray(req.SupportTags), req.BasePrice, req.ImageURL, guID, oldRank)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
			return
		}
	}
	if err := models.LoadGuRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã ghi dữ liệu nhưng không thể tải lại catalog Cổ: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_gu", "id=%s", guID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật cổ trùng", "issues": guEffectWarnings(c)})
}

func AdminDeleteGuItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	guID := c.Param("id")

	var ownedCount int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE gu_id = $1`, guID).Scan(&ownedCount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra sở hữu: " + err.Error()})
		return
	}
	if ownedCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể xóa: có %d Cổ trùng loại này đang được người chơi sở hữu", ownedCount)})
		return
	}

	_, err := db.DB.Exec(`DELETE FROM gu_dictionary WHERE id=$1`, guID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}
	if err := models.LoadGuRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã xóa dữ liệu nhưng không thể tải lại catalog Cổ: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_gu", "id=%s", guID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa cổ trùng"})
}

// -------- Admin: Item Dictionary CRUD --------

func AdminListItemItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, name, item_type, rank, effects, COALESCE(description,''), COALESCE(base_price,0), COALESCE(image_url,''), COALESCE(rarity,0), COALESCE(path,'') FROM item_dictionary ORDER BY name`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type ItemRow struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		ItemType    string          `json:"item_type"`
		Rank        int             `json:"rank"`
		Effects     json.RawMessage `json:"effects"`
		Description string          `json:"description"`
		BasePrice   float64         `json:"base_price"`
		ImageURL    string          `json:"image_url"`
		Rarity      int             `json:"rarity"`
		Path        string          `json:"path"`
	}
	items := []ItemRow{}
	for rows.Next() {
		var r ItemRow
		if err := rows.Scan(&r.ID, &r.Name, &r.ItemType, &r.Rank, &r.Effects, &r.Description, &r.BasePrice, &r.ImageURL, &r.Rarity, &r.Path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ từ điển vật phẩm"})
			return
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc từ điển vật phẩm"})
		return
	}
	c.JSON(http.StatusOK, items)
}

type AdminItemInput struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	ItemType    string          `json:"item_type"`
	Rank        int             `json:"rank"`
	Effects     json.RawMessage `json:"effects"`
	Description string          `json:"description"`
	BasePrice   float64         `json:"base_price"`
	ImageURL    string          `json:"image_url"`
	Rarity      int             `json:"rarity"`
	Path        string          `json:"path"`
}

func validateAdminItemInput(c *gin.Context, req *AdminItemInput) bool {
	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	req.ItemType = strings.TrimSpace(req.ItemType)
	if req.ID == "" || len(req.ID) > 50 || req.Name == "" || len(req.Name) > 100 || req.ItemType == "" || len(req.ItemType) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID (1-50), tên (1-100) và loại vật phẩm (1-50) là bắt buộc"})
		return false
	}
	if req.Rank < 1 || req.Rarity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bậc phải từ 1 trở lên và độ hiếm không được âm"})
		return false
	}
	if len(req.Path) > 50 || math.IsNaN(req.BasePrice) || math.IsInf(req.BasePrice, 0) || req.BasePrice < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đường tu tối đa 50 ký tự; giá phải là số không âm hợp lệ"})
		return false
	}
	return true
}

func AdminCreateItemItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req AdminItemInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if !validateAdminItemInput(c, &req) {
		return
	}
	if len(req.Effects) == 0 {
		req.Effects = json.RawMessage(`{"schema_version":2,"blocks":[]}`)
	}
	if !validateAdminEffects(c, "item", req.ID, req.Effects) {
		return
	}
	_, err := db.DB.Exec(`INSERT INTO item_dictionary(id,name,item_type,rank,effects,description,base_price,image_url,rarity,path) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		req.ID, req.Name, req.ItemType, req.Rank, req.Effects, req.Description, req.BasePrice, req.ImageURL, req.Rarity, req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := models.LoadItemRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu vật phẩm nhưng không thể tải lại catalog: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.create_item", "id=%s", req.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo vật phẩm"})
}

func AdminUpdateItemItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	itemID := c.Param("id")
	var req AdminItemInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.ID != "" && req.ID != itemID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi ID vật phẩm; hãy tạo vật phẩm mới"})
		return
	}
	req.ID = itemID
	if !validateAdminItemInput(c, &req) {
		return
	}
	if len(req.Effects) == 0 {
		req.Effects = json.RawMessage(`{"schema_version":2,"blocks":[]}`)
	}
	if !validateAdminEffects(c, "item", itemID, req.Effects) {
		return
	}
	result, err := db.DB.Exec(`UPDATE item_dictionary SET name=$1,item_type=$2,rank=$3,effects=$4,description=$5,base_price=$6,image_url=$7,rarity=$8,path=$9 WHERE id=$10`,
		req.Name, req.ItemType, req.Rank, req.Effects, req.Description, req.BasePrice, req.ImageURL, req.Rarity, req.Path, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận vật phẩm đã được cập nhật"})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy vật phẩm"})
		return
	}
	if err := models.LoadItemRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã lưu vật phẩm nhưng không thể tải lại catalog: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.update_item", "id=%s", itemID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật vật phẩm"})
}

func AdminDeleteItemItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	itemID := c.Param("id")
	var owned int64
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM player_inventory_items WHERE item_id=$1`, itemID).Scan(&owned); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra túi đồ người chơi"})
		return
	}
	if owned > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Không thể xóa: vật phẩm đang có trong túi đồ của %d dòng dữ liệu người chơi", owned)})
		return
	}
	var shopEntries int64
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM caravan_shop_items WHERE id=$1`, itemID).Scan(&shopEntries); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra vật phẩm trong cửa hàng"})
		return
	}
	if shopEntries > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Không thể xóa: vật phẩm đang có trong %d mục cửa hàng; hãy gỡ khỏi cửa hàng trước", shopEntries)})
		return
	}
	result, err := db.DB.Exec(`DELETE FROM item_dictionary WHERE id=$1`, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận vật phẩm đã được xóa"})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy vật phẩm"})
		return
	}
	if err := models.LoadItemRegistry(db.DB); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Đã xóa vật phẩm nhưng không thể tải lại catalog: " + err.Error()})
		return
	}
	auditlog.Logf(db.DB, &adminID, "admin.delete_item", "id=%s", itemID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa vật phẩm"})
}

// -------- Admin: Recipe CRUD --------

func AdminListRecipes(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT r.target_gu_id, r.target_rank, r.variant, COALESCE(g.name, ''), r.required_gus, r.required_materials, r.success_rate, r.primeval_stones_cost, COALESCE(r.immortal_essence_stones_cost,0) FROM refining_recipes r LEFT JOIN gu_dictionary g ON r.target_gu_id = g.id AND r.target_rank = g.rank ORDER BY r.target_gu_id, r.target_rank, r.variant`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type RecipeRow struct {
		ID                        string          `json:"id"`
		TargetGuID                string          `json:"target_gu_id"`
		TargetRank                int             `json:"target_rank"`
		TargetGuName              string          `json:"target_gu_name,omitempty"`
		RequiredGus               json.RawMessage `json:"required_gus"`
		RequiredMaterials         json.RawMessage `json:"required_materials"`
		SuccessRate               float64         `json:"success_rate"`
		PrimevalStonesCost        float64         `json:"primeval_stones_cost"`
		ImmortalEssenceStonesCost float64         `json:"immortal_essence_stones_cost"`
	}
	items := []RecipeRow{}
	for rows.Next() {
		var r RecipeRow
		var targetGuID string
		var targetRank, variant int
		if err := rows.Scan(&targetGuID, &targetRank, &variant, &r.TargetGuName, &r.RequiredGus, &r.RequiredMaterials, &r.SuccessRate, &r.PrimevalStonesCost, &r.ImmortalEssenceStonesCost); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ danh sách cổ phương"})
			return
		}
		r.TargetGuID = targetGuID
		r.TargetRank = targetRank
		r.ID = fmt.Sprintf("recipe_%s_%d_%d", targetGuID, targetRank, variant)
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ danh sách cổ phương"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func validateAdminRecipeInput(c *gin.Context, req *AdminRecipeInput) bool {
	if strings.TrimSpace(req.TargetGuID) == "" || len(req.TargetGuID) > 50 || req.TargetRank < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đích và bậc đích hợp lệ là bắt buộc"})
		return false
	}
	if math.IsNaN(req.SuccessRate) || math.IsInf(req.SuccessRate, 0) || req.SuccessRate < 0 || req.SuccessRate > 1 ||
		math.IsNaN(req.PrimevalStonesCost) || math.IsInf(req.PrimevalStonesCost, 0) || req.PrimevalStonesCost < 0 ||
		math.IsNaN(req.ImmortalEssenceStonesCost) || math.IsInf(req.ImmortalEssenceStonesCost, 0) || req.ImmortalEssenceStonesCost < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tỉ lệ thành công phải từ 0 đến 1; chi phí không được âm"})
		return false
	}
	if len(req.RequiredGus) == 0 {
		req.RequiredGus = json.RawMessage(`{}`)
	}
	if len(req.RequiredMaterials) == 0 {
		req.RequiredMaterials = json.RawMessage(`{}`)
	}
	var requiredGus, requiredMaterials map[string]json.RawMessage
	if json.Unmarshal(req.RequiredGus, &requiredGus) != nil || requiredGus == nil ||
		json.Unmarshal(req.RequiredMaterials, &requiredMaterials) != nil || requiredMaterials == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng và nguyên liệu yêu cầu phải là đối tượng JSON hợp lệ"})
		return false
	}
	return true
}

type AdminRecipeInput struct {
	ID                        string          `json:"id"`
	TargetGuID                string          `json:"target_gu_id"`
	TargetRank                int             `json:"target_rank"`
	RequiredGus               json.RawMessage `json:"required_gus"`
	RequiredMaterials         json.RawMessage `json:"required_materials"`
	SuccessRate               float64         `json:"success_rate"`
	PrimevalStonesCost        float64         `json:"primeval_stones_cost"`
	ImmortalEssenceStonesCost float64         `json:"immortal_essence_stones_cost"`
}

func AdminCreateRecipe(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req AdminRecipeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if !validateAdminRecipeInput(c, &req) {
		return
	}
	targetRank := req.TargetRank
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM gu_dictionary WHERE id=$1 AND rank=$2)`, req.TargetGuID, targetRank).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác thực cổ trùng đích"})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_gu_id không tồn tại"})
		return
	}
	variant := 1
	if tg, tr, v, err := ParseRecipeID(req.ID); err == nil && tg == req.TargetGuID && tr == targetRank {
		variant = v
	} else {
		if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(MAX(variant), 0) + 1 FROM refining_recipes WHERE target_gu_id = $1 AND target_rank = $2`, req.TargetGuID, targetRank).Scan(&variant); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác định biến thể cổ phương"})
			return
		}
	}

	_, err := db.DB.Exec(`INSERT INTO refining_recipes(target_gu_id,target_rank,variant,required_gus,required_materials,success_rate,primeval_stones_cost,immortal_essence_stones_cost) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		req.TargetGuID, targetRank, variant, req.RequiredGus, req.RequiredMaterials, req.SuccessRate, req.PrimevalStonesCost, req.ImmortalEssenceStonesCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	generatedID := fmt.Sprintf("recipe_%s_%d_%d", req.TargetGuID, targetRank, variant)
	revision := invalidateRecipeCatalog()
	auditlog.Logf(db.DB, &adminID, "admin.create_recipe", "id=%s target=%s", generatedID, req.TargetGuID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo cổ phương", "recipe_id": generatedID, "catalog_revision": revision})
}

func AdminUpdateRecipe(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	recipeID := c.Param("id")
	targetGuID, targetRank, variant, err := ParseRecipeID(recipeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID cổ phương không hợp lệ"})
		return
	}
	var req AdminRecipeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.TargetGuID != "" && req.TargetGuID != targetGuID || req.TargetRank != 0 && req.TargetRank != targetRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể đổi cổ trùng hoặc bậc đích sau khi tạo; hãy tạo cổ phương mới"})
		return
	}
	req.TargetGuID, req.TargetRank = targetGuID, targetRank
	if !validateAdminRecipeInput(c, &req) {
		return
	}
	var updatedID string
	err = db.DB.QueryRowContext(c.Request.Context(), `UPDATE refining_recipes SET required_gus=$1,required_materials=$2,success_rate=$3,primeval_stones_cost=$4,immortal_essence_stones_cost=$5 WHERE target_gu_id=$6 AND target_rank=$7 AND variant=$8 RETURNING target_gu_id`,
		req.RequiredGus, req.RequiredMaterials, req.SuccessRate, req.PrimevalStonesCost, req.ImmortalEssenceStonesCost, targetGuID, targetRank, variant).Scan(&updatedID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cổ phương"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	revision := invalidateRecipeCatalog()
	auditlog.Logf(db.DB, &adminID, "admin.update_recipe", "id=%s", recipeID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật cổ phương", "catalog_revision": revision})
}

func AdminDeleteRecipe(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	recipeID := c.Param("id")
	targetGuID, targetRank, variant, err := ParseRecipeID(recipeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID cổ phương không hợp lệ"})
		return
	}
	result, err := db.DB.Exec(`DELETE FROM refining_recipes WHERE target_gu_id=$1 AND target_rank=$2 AND variant=$3`, targetGuID, targetRank, variant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận cổ phương đã được xóa"})
		return
	}
	if deleted == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cổ phương"})
		return
	}
	revision := invalidateRecipeCatalog()
	auditlog.Logf(db.DB, &adminID, "admin.delete_recipe", "id=%s", recipeID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa cổ phương", "catalog_revision": revision})
}
