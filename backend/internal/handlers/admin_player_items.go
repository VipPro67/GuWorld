package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminItemRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int64  `json:"quantity"`
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

	kind := adminItemKind(req.ItemID)
	if kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm hoặc công thức không tồn tại"})
		return
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
	} else if err := InventorySvc.AddItem(tx, playerID.String(), req.ItemID, "materials", req.Quantity); err != nil {
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
		RequestSummary: fmt.Sprintf("%s=%s qty=%d", kind, req.ItemID, req.Quantity),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm thành công", "kind": kind, "item_id": req.ItemID, "quantity": req.Quantity})
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

	kind := adminItemKind(req.ItemID)
	if kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm hoặc công thức không tồn tại"})
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
	} else if err := InventorySvc.RemoveItem(tx, playerID.String(), req.ItemID, "materials", req.Quantity); err != nil {
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
		RequestSummary: fmt.Sprintf("%s=%s qty=%d", kind, req.ItemID, req.Quantity),
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thành công", "kind": kind, "item_id": req.ItemID, "quantity": req.Quantity})
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

	if err := db.RemovePlayerGuForPlayer(tx, playerID.String(), instanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cổ trùng"})
		return
	}

	tx.Commit()

	guName := removedGuID
	if spec, ok := models.GetGuAny(removedGuID); ok {
		guName = spec.Name
	}
	_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Admin đã thu hồi Cổ trùng khỏi đạo hữu %s: %s", player.Username, guName))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.remove_gu",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: fmt.Sprintf("instance=%s", instanceID),
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
		args = append(args, math.Max(0, *req.Hunger))
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
		args = append(args, math.Max(0, *req.IntegrationTime))
		argIdx++
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu thay đổi"})
		return
	}

	sets = append(sets, "updated_at=NOW()")
	args = append(args, instanceID, playerID.String())

	query := fmt.Sprintf(`UPDATE player_gu SET %s WHERE id=$%d AND player_id=$%d`, strings.Join(sets, ", "), argIdx, argIdx+1)
	res, err := db.DB.ExecContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Gu: " + err.Error()})
		return
	}

	if rows, _ := res.RowsAffected(); rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Gu instance này"})
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
		PrimevalStones *float64 `json:"primeval_stones"`
		ImmortalStones *float64 `json:"immortal_stones"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.PrimevalStones == nil && req.ImmortalStones == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phải nhập ít nhất một giá trị"})
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

	if _, err := tx.Exec(`UPDATE players SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(argIdx), args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tài nguyên"})
		return
	}

	tx.Commit()

	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
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

	summary := ""
	if req.PrimevalStones != nil {
		summary += fmt.Sprintf("primeval_stones=%.0f ", *req.PrimevalStones)
	}
	if req.ImmortalStones != nil {
		summary += fmt.Sprintf("immortal_stones=%.0f ", *req.ImmortalStones)
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.change_stones",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: summary,
		IPAddress:      c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật tài nguyên thành công"})
}
