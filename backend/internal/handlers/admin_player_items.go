package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

type AdminItemRequest struct {
	ItemID             string `json:"item_id"`
	Quantity           int64  `json:"quantity"`
	Category           string `json:"category"`
	Reason             string `json:"reason"`
	ConfirmationTarget string `json:"confirmation_target"`
}

type AdminUpdateInventoryItemRequest struct {
	Quantity int64  `json:"quantity"`
	Category string `json:"category"`
}

func AdminUpdatePlayerInventoryItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	itemID := strings.TrimSpace(c.Param("item_id"))
	oldCategory := adminInventoryCategory(c.Param("category"))
	if itemID == "" || oldCategory == "" || oldCategory == "recipe" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Định danh vật phẩm không hợp lệ"})
		return
	}
	if _, exists := models.ItemRegistry[itemID]; !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong từ điển"})
		return
	}
	var req AdminUpdateInventoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Quantity < 0 || req.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng phải không âm và nhóm vật phẩm phải hợp lệ"})
		return
	}
	newCategory := adminInventoryCategory(req.Category)
	if newCategory == "" || newCategory == "recipe" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nhóm vật phẩm không hợp lệ"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()
	var current int64
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT quantity FROM player_inventory_items WHERE player_id=$1 AND item_id=$2 AND category=$3 FOR UPDATE`, playerID, itemID, oldCategory).Scan(&current); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy dòng vật phẩm"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc vật phẩm"})
		}
		return
	}
	if req.Quantity == 0 {
		if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM player_inventory_items WHERE player_id=$1 AND item_id=$2 AND category=$3`, playerID, itemID, oldCategory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm"})
			return
		}
	} else if newCategory == oldCategory {
		if _, err := tx.ExecContext(c.Request.Context(), `UPDATE player_inventory_items SET quantity=$4, updated_at=NOW() WHERE player_id=$1 AND item_id=$2 AND category=$3`, playerID, itemID, oldCategory, req.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật vật phẩm"})
			return
		}
	} else {
		if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM player_inventory_items WHERE player_id=$1 AND item_id=$2 AND category=$3`, playerID, itemID, oldCategory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuyển nhóm vật phẩm"})
			return
		}
		if _, err := tx.ExecContext(c.Request.Context(), `INSERT INTO player_inventory_items(player_id,item_id,category,quantity) VALUES($1,$2,$3,$4) ON CONFLICT(player_id,item_id,category) DO UPDATE SET quantity=player_inventory_items.quantity+EXCLUDED.quantity,updated_at=NOW()`, playerID, itemID, newCategory, req.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuyển nhóm vật phẩm"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu thay đổi"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.update_player_inventory_item", EntityType: "player_inventory_items", EntityID: playerID.String() + ":" + itemID + ":" + oldCategory, RequestSummary: fmt.Sprintf("quantity=%d category=%s->%s", req.Quantity, oldCategory, newCategory), IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật vật phẩm", "quantity": req.Quantity, "category": newCategory})
}

type AdminRecipeProgressRequest struct {
	CompletionRate float64 `json:"completion_rate"`
	IsUnlocked     bool    `json:"is_unlocked"`
	DeduceCount    int     `json:"deduce_count"`
	Favorite       bool    `json:"favorite"`
}

func adminInventoryCategory(value string) string {
	category := strings.ToLower(strings.TrimSpace(value))
	if category == "" {
		return "materials"
	}
	if category == "materials" || category == "consumables" || category == "equipment" || category == "quest" || category == "misc" || category == "recipe" {
		return category
	}
	return ""
}

// adminItemKind returns "item" if the ID is a material item, "recipe" if it is
// a refining recipe (Cổ phương), or "" if the ID matches neither.
func adminItemKind(itemID string) string {
	if _, ok := models.ItemRegistry[itemID]; ok {
		return "item"
	}
	var exists bool
	if targetGuID, targetRank, variant, err := ParseRecipeID(itemID); err == nil {
		if err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM refining_recipes WHERE target_gu_id = $1 AND target_rank = $2 AND variant = $3)`, targetGuID, targetRank, variant).Scan(&exists); err == nil && exists {
			return "recipe"
		}
	}
	return ""
}

func AdminAddPlayerItems(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req AdminItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemID == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	category := adminInventoryCategory(req.Category)
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category vật phẩm không hợp lệ"})
		return
	}

	kind := adminItemKind(req.ItemID)
	if kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm hoặc công thức không tồn tại"})
		return
	}
	if kind == "recipe" {
		category = "recipe"
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	_, err = loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	if kind == "recipe" {
		if err := RecipeSvc.Upsert(tx, &services.RecipeProgress{
			PlayerID:       playerID.String(),
			RecipeID:       req.ItemID,
			CompletionRate: 100,
			IsUnlocked:     true,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật công thức"})
			return
		}
	} else if err := InventorySvc.AddItem(tx, playerID.String(), req.ItemID, category, req.Quantity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật vật phẩm"})
		return
	}

	tx.Commit()

	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	itemName := req.ItemID
	if spec, ok := models.ItemRegistry[req.ItemID]; ok {
		itemName = spec.Name
	} else if kind == "recipe" {
		if targetGuID, _, _, err := ParseRecipeID(req.ItemID); err == nil {
			if guSpec, ok := models.GetGuAny(targetGuID); ok {
				itemName = "Công thức luyện " + guSpec.Name
			}
		}
	}
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã thêm vào túi đồ của đạo hữu %s: %s (SL: %d)", username, itemName, req.Quantity))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.add_items",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: fmt.Sprintf("%s=%s category=%s qty=%d", kind, req.ItemID, category, req.Quantity),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm thành công", "kind": kind, "item_id": req.ItemID, "category": category, "quantity": req.Quantity})
}

func AdminRemovePlayerItems(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req AdminItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemID == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	category := adminInventoryCategory(req.Category)
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category vật phẩm không hợp lệ"})
		return
	}

	kind := adminItemKind(req.ItemID)
	if kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm hoặc công thức không tồn tại"})
		return
	}
	if kind == "recipe" {
		category = "recipe"
	}
	if strings.TrimSpace(req.Reason) == "" || strings.TrimSpace(req.ConfirmationTarget) != c.Request.URL.Path {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần lý do và xác nhận đúng vật phẩm cần thu hồi"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	if kind == "recipe" {
		if err := RecipeSvc.Delete(tx, playerID.String(), req.ItemID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa công thức"})
			return
		}
	} else if err := InventorySvc.RemoveItem(tx, playerID.String(), req.ItemID, category, req.Quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người chơi không có đủ vật phẩm"})
		return
	}

	tx.Commit()

	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	itemName := req.ItemID
	if spec, ok := models.ItemRegistry[req.ItemID]; ok {
		itemName = spec.Name
	} else if kind == "recipe" {
		if targetGuID, _, _, err := ParseRecipeID(req.ItemID); err == nil {
			if guSpec, ok := models.GetGuAny(targetGuID); ok {
				itemName = "Công thức luyện " + guSpec.Name
			}
		}
	}
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã thu hồi từ túi đồ của đạo hữu %s: %s (SL: %d)", username, itemName, req.Quantity))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.remove_items",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: fmt.Sprintf("%s=%s category=%s qty=%d reason=%s", kind, req.ItemID, category, req.Quantity, req.Reason),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thành công", "kind": kind, "item_id": req.ItemID, "category": category, "quantity": req.Quantity})
}

// AdminUpdatePlayerRecipe updates every administrator-controlled field of a player's recipe progress.
func AdminUpdatePlayerRecipe(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	recipeID := strings.TrimSpace(c.Param("recipe_id"))
	if recipeID == "" || adminItemKind(recipeID) != "recipe" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Công thức không tồn tại"})
		return
	}
	var req AdminRecipeProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil || math.IsNaN(req.CompletionRate) || math.IsInf(req.CompletionRate, 0) || req.CompletionRate < 0 || req.CompletionRate > 100 || req.DeduceCount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiến độ công thức không hợp lệ"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()
	if _, err := loadPlayerForStatus(tx, playerID, true); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}
	var lastDeduceAt *time.Time
	if existing, getErr := RecipeSvc.Get(tx, playerID.String(), recipeID); getErr == nil {
		lastDeduceAt = existing.LastDeduceAt
	}
	if err := RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: playerID.String(), RecipeID: recipeID, CompletionRate: req.CompletionRate, IsUnlocked: req.IsUnlocked, DeduceCount: req.DeduceCount, LastDeduceAt: lastDeduceAt, Favorite: req.Favorite}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tiến độ công thức"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu tiến độ công thức"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.update_recipe_progress", EntityType: "player_recipe_progress", EntityID: playerID.String() + ":" + recipeID, RequestSummary: fmt.Sprintf("recipe=%s completion=%.2f unlocked=%t deduce=%d favorite=%t", recipeID, req.CompletionRate, req.IsUnlocked, req.DeduceCount, req.Favorite), IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật tiến độ công thức", "recipe_id": recipeID, "completion_rate": req.CompletionRate, "is_unlocked": req.IsUnlocked, "deduce_count": req.DeduceCount, "favorite": req.Favorite})
}

type AdminAddGuRequest struct {
	GuID     string `json:"gu_id"`
	Rank     int    `json:"rank"`
	Quantity int    `json:"quantity"`
}

func AdminAddPlayerGus(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req AdminAddGuRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.GuID == "" || req.Quantity <= 0 || req.Rank < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	var guSpec models.GuDictionaryItem
	if req.Rank > 0 {
		var ok bool
		guSpec, ok = models.GetGu(req.GuID, req.Rank)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cổ trùng %s bậc %d không tồn tại", req.GuID, req.Rank)})
			return
		}
	} else {
		var ok bool
		guSpec, ok = models.GetGuAny(req.GuID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng không tồn tại"})
			return
		}
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	player, err := loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	for i := 0; i < req.Quantity; i++ {
		player.GuInventory = append(player.GuInventory, models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       req.GuID,
			Rank:       guSpec.Rank,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
		})
	}

	if err := db.SyncPlayerGus(tx, playerID.String(), before, player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cổ trùng"})
		return
	}

	tx.Commit()

	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	guName := req.GuID
	if spec, ok := models.GetGu(req.GuID, guSpec.Rank); ok {
		guName = spec.Name
	}
	rankDesc := ""
	if guSpec.Rank > 0 {
		rankDesc = fmt.Sprintf(" (Bậc %d)", guSpec.Rank)
	}
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã ban tặng Cổ trùng cho đạo hữu %s: %s%s (SL: %d)", username, guName, rankDesc, req.Quantity))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.add_gus",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: fmt.Sprintf("gu=%s rank=%d qty=%d", req.GuID, guSpec.Rank, req.Quantity),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm cổ trùng thành công", "gu_id": req.GuID, "quantity": req.Quantity})
}

func AdminRemovePlayerGu(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	instanceID := c.Param("instance_id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance ID không hợp lệ"})
		return
	}
	var adminAction struct {
		Reason             string `json:"reason"`
		ConfirmationTarget string `json:"confirmation_target"`
	}
	if err := c.ShouldBindJSON(&adminAction); err != nil || strings.TrimSpace(adminAction.Reason) == "" || strings.TrimSpace(adminAction.ConfirmationTarget) != c.Request.URL.Path {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần lý do và xác nhận mục tiêu để thu hồi Cổ trùng"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	player, err := loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	gu, _, found := models.FindGuByInstanceID(player.GuInventory, instanceID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cổ trùng này"})
		return
	}
	removedGuID := gu.GuID

	deleteResult, err := tx.Exec(`DELETE FROM player_gu WHERE player_id=$1 AND id=$2 AND NOT EXISTS (SELECT 1 FROM actor_gu_equipment WHERE player_gu_id=$2)`, playerID.String(), instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cổ trùng"})
		return
	}
	if affected, _ := deleteResult.RowsAffected(); affected != 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cổ trùng không thuộc túi người chơi hoặc đang được nhân vật khác sử dụng"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu thay đổi"})
		return
	}

	guName := removedGuID
	if spec, ok := models.GetGuAny(removedGuID); ok {
		guName = spec.Name
	}
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã thu hồi Cổ trùng khỏi đạo hữu %s: %s", player.Username, guName))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.remove_gu",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: fmt.Sprintf("instance=%s reason=%s", instanceID, adminAction.Reason),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa cổ trùng thành công"})
}

type AdminUpdateGuInstanceRequest struct {
	Rank            *int     `json:"rank"`
	Level           *int     `json:"level"`
	Hunger          *float64 `json:"hunger"`
	State           *string  `json:"state"`
	Equipped        *bool    `json:"equipped"`
	IsVital         *bool    `json:"is_vital"`
	IsLocked        *bool    `json:"is_locked"`
	IntegrationTime *float64 `json:"integration_time"`
}

func AdminUpdatePlayerGu(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	instanceID := c.Param("instance_id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance ID không hợp lệ"})
		return
	}

	var req AdminUpdateGuInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.Rank != nil && (*req.Rank < 1 || *req.Rank > 9) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bậc Cổ trùng phải từ 1 đến 9"})
		return
	}
	if req.Rank != nil {
		var guID string
		if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT gu_id FROM player_gu WHERE id=$1 AND player_id=$2`, instanceID, playerID.String()).Scan(&guID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Gu instance này"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc Gu instance"})
			}
			return
		}
		var rankExists bool
		if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM gu_dictionary WHERE id=$1 AND rank=$2)`, guID, *req.Rank).Scan(&rankExists); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh bậc Cổ trùng"})
			return
		}
		if !rankExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bậc này không tồn tại trong từ điển Cổ trùng"})
			return
		}
	}
	if req.Level != nil && *req.Level < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cấp Cổ trùng phải lớn hơn 0"})
		return
	}
	if req.Hunger != nil && (math.IsNaN(*req.Hunger) || math.IsInf(*req.Hunger, 0) || *req.Hunger < 0 || *req.Hunger > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Độ đói phải nằm trong khoảng 0 đến 100"})
		return
	}
	if req.IntegrationTime != nil && (math.IsNaN(*req.IntegrationTime) || math.IsInf(*req.IntegrationTime, 0) || *req.IntegrationTime < 0) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thời gian dung hợp phải là số không âm hợp lệ"})
		return
	}
	if req.State != nil {
		state := strings.ToUpper(strings.TrimSpace(*req.State))
		if state != "ACTIVE" && state != "SLEEPING" && state != "SEALED" && state != "DEAD" && state != "EQUIPPING" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái Cổ trùng không hợp lệ"})
			return
		}
		req.State = &state
	}

	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Rank != nil {
		sets = append(sets, "rank=$"+strconv.Itoa(argIdx), "gu_rank=$"+strconv.Itoa(argIdx))
		args = append(args, *req.Rank)
		argIdx++
	}
	if req.Level != nil {
		sets = append(sets, "level=$"+strconv.Itoa(argIdx))
		args = append(args, *req.Level)
		argIdx++
	}
	if req.Hunger != nil {
		sets = append(sets, "hunger=$"+strconv.Itoa(argIdx))
		args = append(args, *req.Hunger)
		argIdx++
	}
	if req.State != nil {
		sets = append(sets, "state=$"+strconv.Itoa(argIdx))
		args = append(args, *req.State)
		argIdx++
	}
	if req.Equipped != nil {
		sets = append(sets, "equipped=$"+strconv.Itoa(argIdx))
		args = append(args, *req.Equipped)
		argIdx++
	}
	if req.IsVital != nil {
		sets = append(sets, "is_vital=$"+strconv.Itoa(argIdx))
		args = append(args, *req.IsVital)
		argIdx++
	}
	if req.IsLocked != nil {
		sets = append(sets, "is_locked=$"+strconv.Itoa(argIdx))
		args = append(args, *req.IsLocked)
		argIdx++
	}
	if req.IntegrationTime != nil {
		sets = append(sets, "integration_time=$"+strconv.Itoa(argIdx))
		args = append(args, *req.IntegrationTime)
		argIdx++
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu thay đổi"})
		return
	}

	sets = append(sets, "updated_at=NOW()")
	args = append(args, instanceID, playerID.String())

	// Keep vital Gu semantics consistent with the player-facing equip flow and
	// refuse mutations when this instance is currently assigned to an actor.
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()
	var location string
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT location FROM player_gu WHERE id=$1 AND player_id=$2 FOR UPDATE`, instanceID, playerID.String()).Scan(&location); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Gu instance này"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc Gu instance"})
		}
		return
	}
	var actorOwned bool
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM actor_gu_equipment WHERE player_gu_id=$1)`, instanceID).Scan(&actorOwned); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác định chủ sở hữu Cổ trùng"})
		return
	}
	if location == "ACTOR" || actorOwned {
		c.JSON(http.StatusConflict, gin.H{"error": "Cổ trùng đang thuộc trang bị của nhân vật khác, không thể sửa tại hồ sơ người chơi"})
		return
	}
	if req.IsVital != nil && *req.IsVital {
		sets = append(sets, "state='ACTIVE'", "equipped=true")
		if _, err := tx.ExecContext(c.Request.Context(), `UPDATE player_gu SET is_vital=false WHERE player_id=$1 AND id<>$2 AND location <> 'ACTOR'`, playerID.String(), instanceID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật Cổ trùng bản mệnh"})
			return
		}
	}
	query := fmt.Sprintf(`UPDATE player_gu SET %s WHERE id=$%d AND player_id=$%d`, strings.Join(sets, ", "), argIdx, argIdx+1)
	res, err := tx.ExecContext(c.Request.Context(), query, args...)
	if err != nil {
		if strings.Contains(err.Error(), "uq_player_gu_immortal") {
			c.JSON(http.StatusConflict, gin.H{"error": "Cổ trùng Tiên cấp này đã tồn tại trong hệ thống"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Gu: " + err.Error()})
		return
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Gu instance này"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu thay đổi"})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.update_gu_instance", "instance_id=%s, player_id=%s", instanceID, playerID.String())
	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật Gu instance thành công"})
}

func AdminChangePlayerStones(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req struct {
		PrimevalStones     *float64 `json:"primeval_stones"`
		ImmortalStones     *float64 `json:"immortal_stones"`
		Reason             string   `json:"reason"`
		ConfirmationTarget string   `json:"confirmation_target"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.PrimevalStones == nil && req.ImmortalStones == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phải nhập ít nhất một giá trị"})
		return
	}
	if (req.PrimevalStones != nil && (*req.PrimevalStones < 0 || !pricing.IsValidAmount(*req.PrimevalStones))) ||
		(req.ImmortalStones != nil && (*req.ImmortalStones < 0 || !pricing.IsValidAmount(*req.ImmortalStones))) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số dư phải không âm và là bội số của 0.001"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.PrimevalStones != nil {
		sets = append(sets, "primeval_stones=$"+strconv.Itoa(argIdx))
		args = append(args, *req.PrimevalStones)
		argIdx++
	}
	if req.ImmortalStones != nil {
		sets = append(sets, "immortal_stones=$"+strconv.Itoa(argIdx))
		args = append(args, *req.ImmortalStones)
		argIdx++
	}
	args = append(args, playerID)

	var username string
	if err := tx.QueryRow(`UPDATE players SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(argIdx)+` RETURNING username`, args...).Scan(&username); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tài nguyên"})
		return
	}
	if err := auditlog.LogChecked(tx, auditlog.Entry{PlayerID: &adminID, Action: "admin.change_stones", EntityType: "player", EntityID: playerID.String(), Reason: strings.TrimSpace(req.Reason), RequestSummary: "absolute balance adjustment", IPAddress: c.ClientIP(), Success: true}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi audit; transaction đã rollback"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất cập nhật tài nguyên"})
		return
	}
	msgContent := fmt.Sprintf("Admin đã điều chỉnh tài nguyên của đạo hữu %s: ", username)
	msgParts := []string{}
	if req.PrimevalStones != nil {
		msgParts = append(msgParts, fmt.Sprintf("%.0f Nguyên Thạch", *req.PrimevalStones))
	}
	if req.ImmortalStones != nil {
		msgParts = append(msgParts, fmt.Sprintf("%.0f Tiên Nguyên Thạch", *req.ImmortalStones))
	}
	msgContent += strings.Join(msgParts, ", ")
	_ = SendSystemMessage("world", sql.NullString{}, msgContent)

	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật tài nguyên thành công"})
}
