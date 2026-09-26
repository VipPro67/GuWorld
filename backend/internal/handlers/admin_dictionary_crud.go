package handlers

import (
	"encoding/json"
	"fmt"
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
		COALESCE(is_immortal,false), COALESCE(primary_role,''), COALESCE(combat_roles,'[]'::jsonb),
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
		IsImmortal   bool            `json:"is_immortal"`
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
			&r.Description, &r.TargetType, &r.Rarity, &r.IsImmortal, &r.PrimaryRole,
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
	IsImmortal   bool            `json:"is_immortal"`
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
		req.Effects = json.RawMessage(`{"schema_version":2,"blocks":[]}`)
	}
	if !validateAndNormalizeAdminGuEffects(c, req.ID, &req.Effects) {
		return
	}

	_, err := db.DB.Exec(`INSERT INTO gu_dictionary (id,name,rank,path,gu_type,base_mana_cost,effects,food_types,hunger_rate,max_hunger,description,target_type,rarity,is_immortal,primary_role,combat_roles,combat_tags,support_tags,base_price,image_url) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		req.ID, req.Name, req.Rank, marshalArray(req.Path), req.GuType, req.BaseManaCost, req.Effects, marshalArray(req.FoodTypes),
		req.HungerRate, req.MaxHunger, req.Description, req.TargetType, req.Rarity, req.IsImmortal, req.PrimaryRole,
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
		req.Effects = json.RawMessage(`{"schema_version":2,"blocks":[]}`)
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
		sets = append(sets, "is_immortal=$"+strconv.Itoa(idx))
		args = append(args, req.IsImmortal)
		idx++
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
		_, err := db.DB.Exec(`UPDATE gu_dictionary SET name=$1,rank=$2,path=$3,gu_type=$4,base_mana_cost=$5,effects=$6,food_types=$7,hunger_rate=$8,max_hunger=$9,description=$10,target_type=$11,rarity=$12,is_immortal=$13,primary_role=$14,combat_roles=$15,combat_tags=$16,support_tags=$17,base_price=$18,image_url=$19 WHERE id=$20 AND rank=$21`,
			req.Name, req.Rank, marshalArray(req.Path), req.GuType, req.BaseManaCost, req.Effects, marshalArray(req.FoodTypes),
			req.HungerRate, req.MaxHunger, req.Description, req.TargetType, req.Rarity, req.IsImmortal, req.PrimaryRole,
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
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, name, item_type, rank, effects, COALESCE(description,''), COALESCE(base_price,0), COALESCE(image_url,''), COALESCE(rarity,0) FROM item_dictionary ORDER BY name`)
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
	}
	items := []ItemRow{}
	for rows.Next() {
		var r ItemRow
		if err := rows.Scan(&r.ID, &r.Name, &r.ItemType, &r.Rank, &r.Effects, &r.Description, &r.BasePrice, &r.ImageURL, &r.Rarity); err != nil {
			continue
		}
		items = append(items, r)
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
}

func AdminCreateItemItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var req AdminItemInput
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" || req.Name == "" || req.ItemType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ (id, name, item_type là bắt buộc)"})
		return
	}
	if req.Rank < 1 {
		req.Rank = 1
	}
	if len(req.Effects) == 0 {
		req.Effects = json.RawMessage(`{"schema_version":2,"blocks":[]}`)
	}
	if !validateAdminEffects(c, "item", req.ID, req.Effects) {
		return
	}
	_, err := db.DB.Exec(`INSERT INTO item_dictionary(id,name,item_type,rank,effects,description,base_price,image_url,rarity) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		req.ID, req.Name, req.ItemType, req.Rank, req.Effects, req.Description, req.BasePrice, req.ImageURL, req.Rarity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	models.LoadItemRegistry(db.DB)
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
	sets := []string{}
	args := []interface{}{}
	idx := 1
	if req.Name != "" {
		sets = append(sets, "name=$"+strconv.Itoa(idx))
		args = append(args, req.Name)
		idx++
	}
	if req.ItemType != "" {
		sets = append(sets, "item_type=$"+strconv.Itoa(idx))
		args = append(args, req.ItemType)
		idx++
	}
	if req.Rank > 0 {
		sets = append(sets, "rank=$"+strconv.Itoa(idx))
		args = append(args, req.Rank)
		idx++
	}
	if req.Description != "" || true {
		sets = append(sets, "description=$"+strconv.Itoa(idx))
		args = append(args, req.Description)
		idx++
	}
	if req.BasePrice > 0 || true {
		sets = append(sets, "base_price=$"+strconv.Itoa(idx))
		args = append(args, req.BasePrice)
		idx++
	}
	if req.Rarity >= 0 || true {
		sets = append(sets, "rarity=$"+strconv.Itoa(idx))
		args = append(args, req.Rarity)
		idx++
	}
	if req.ImageURL != "" {
		sets = append(sets, "image_url=$"+strconv.Itoa(idx))
		args = append(args, req.ImageURL)
		idx++
	}
	if len(req.Effects) > 0 {
		if !validateAdminEffects(c, "item", itemID, req.Effects) {
			return
		}
		sets = append(sets, "effects=$"+strconv.Itoa(idx))
		args = append(args, req.Effects)
		idx++
	}
	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu"})
		return
	}
	args = append(args, itemID)
	_, err := db.DB.Exec(`UPDATE item_dictionary SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(idx), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	models.LoadItemRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.update_item", "id=%s", itemID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật vật phẩm"})
}

func AdminDeleteItemItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	itemID := c.Param("id")
	_, err := db.DB.Exec(`DELETE FROM item_dictionary WHERE id=$1`, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	models.LoadItemRegistry(db.DB)
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
			continue
		}
		r.TargetGuID = targetGuID
		r.ID = fmt.Sprintf("recipe_%s_%d_%d", targetGuID, targetRank, variant)
		items = append(items, r)
	}
	c.JSON(http.StatusOK, items)
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
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetGuID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	targetRank := 1
	if req.TargetRank > 0 {
		targetRank = req.TargetRank
	}
	var exists bool
	db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM gu_dictionary WHERE id=$1 AND rank=$2)`, req.TargetGuID, targetRank).Scan(&exists)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_gu_id không tồn tại"})
		return
	}
	if req.SuccessRate <= 0 {
		req.SuccessRate = 0.5
	}
	if len(req.RequiredGus) == 0 {
		req.RequiredGus = json.RawMessage(`{}`)
	}
	if len(req.RequiredMaterials) == 0 {
		req.RequiredMaterials = json.RawMessage(`{}`)
	}

	variant := 1
	if tg, tr, v, err := ParseRecipeID(req.ID); err == nil && tg == req.TargetGuID && tr == targetRank {
		variant = v
	} else {
		_ = db.DB.QueryRow(`SELECT COALESCE(MAX(variant), 0) + 1 FROM refining_recipes WHERE target_gu_id = $1 AND target_rank = $2`, req.TargetGuID, targetRank).Scan(&variant)
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
	sets := []string{}
	args := []interface{}{}
	idx := 1
	if req.TargetGuID != "" {
		sets = append(sets, "target_gu_id=$"+strconv.Itoa(idx))
		args = append(args, req.TargetGuID)
		idx++
	}
	if req.TargetRank > 0 {
		sets = append(sets, "target_rank=$"+strconv.Itoa(idx))
		args = append(args, req.TargetRank)
		idx++
	}
	if req.SuccessRate > 0 {
		sets = append(sets, "success_rate=$"+strconv.Itoa(idx))
		args = append(args, req.SuccessRate)
		idx++
	}
	if req.PrimevalStonesCost > 0 {
		sets = append(sets, "primeval_stones_cost=$"+strconv.Itoa(idx))
		args = append(args, req.PrimevalStonesCost)
		idx++
	}
	if req.ImmortalEssenceStonesCost > 0 {
		sets = append(sets, "immortal_essence_stones_cost=$"+strconv.Itoa(idx))
		args = append(args, req.ImmortalEssenceStonesCost)
		idx++
	}
	if len(req.RequiredGus) > 0 {
		sets = append(sets, "required_gus=$"+strconv.Itoa(idx))
		args = append(args, req.RequiredGus)
		idx++
	}
	if len(req.RequiredMaterials) > 0 {
		sets = append(sets, "required_materials=$"+strconv.Itoa(idx))
		args = append(args, req.RequiredMaterials)
		idx++
	}
	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu"})
		return
	}

	args = append(args, targetGuID, targetRank, variant)
	whereClause := fmt.Sprintf(" WHERE target_gu_id=$%d AND target_rank=$%d AND variant=$%d", idx, idx+1, idx+2)
	_, err = db.DB.Exec(`UPDATE refining_recipes SET `+strings.Join(sets, ", ")+whereClause, args...)
	if err != nil {
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
	_, err = db.DB.Exec(`DELETE FROM refining_recipes WHERE target_gu_id=$1 AND target_rank=$2 AND variant=$3`, targetGuID, targetRank, variant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	revision := invalidateRecipeCatalog()
	auditlog.Logf(db.DB, &adminID, "admin.delete_recipe", "id=%s", recipeID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa cổ phương", "catalog_revision": revision})
}
