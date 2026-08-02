package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// adminPlayerSearchClause builds the WHERE fragment + args for the admin player list search.
// Supported searchBy values: all, id, username, email, item, gu.
// When searchBy is "all", a valid UUID input is auto-detected as an id search.
func adminPlayerSearchClause(searchBy, search string) (string, []interface{}) {
	if search == "" {
		return "", nil
	}

	by := searchBy
	if by == "all" {
		if _, err := uuid.Parse(search); err == nil {
			by = "id"
		}
	}

	like := "%" + search + "%"
	switch by {
	case "id":
		return `players.id::text ILIKE $1`, []interface{}{like}
	case "username":
		return `username ILIKE $1`, []interface{}{like}
	case "email":
		return `COALESCE(email, '') ILIKE $1`, []interface{}{like}
	case "item":
		itemIDs := models.ItemIDsByName(search)
		if len(itemIDs) > 0 {
			return `EXISTS (SELECT 1 FROM player_inventory_items i WHERE i.player_id = players.id AND (i.item_id ILIKE $1 OR i.item_id = ANY($2)))`, []interface{}{like, itemIDs}
		}
		return `EXISTS (SELECT 1 FROM player_inventory_items i WHERE i.player_id = players.id AND i.item_id ILIKE $1)`, []interface{}{like}
	case "gu":
		guIDs := models.GuIDsByName(search)
		if len(guIDs) > 0 {
			return `EXISTS (SELECT 1 FROM player_gu pg WHERE pg.player_id = players.id AND (pg.gu_id ILIKE $1 OR pg.gu_id = ANY($2)))`, []interface{}{like, guIDs}
		}
		return `EXISTS (SELECT 1 FROM player_gu pg WHERE pg.player_id = players.id AND pg.gu_id ILIKE $1)`, []interface{}{like}
	default: // all
		clauses := []string{`username ILIKE $1`, `COALESCE(email, '') ILIKE $1`}
		args := []interface{}{like}
		argIdx := 2

		itemIDs := models.ItemIDsByName(search)
		if len(itemIDs) > 0 {
			clauses = append(clauses, fmt.Sprintf(`EXISTS (SELECT 1 FROM player_inventory_items i WHERE i.player_id = players.id AND (i.item_id ILIKE $1 OR i.item_id = ANY($%d)))`, argIdx))
			args = append(args, itemIDs)
			argIdx++
		} else {
			clauses = append(clauses, `EXISTS (SELECT 1 FROM player_inventory_items i WHERE i.player_id = players.id AND i.item_id ILIKE $1)`)
		}

		guIDs := models.GuIDsByName(search)
		if len(guIDs) > 0 {
			clauses = append(clauses, fmt.Sprintf(`EXISTS (SELECT 1 FROM player_gu pg WHERE pg.player_id = players.id AND (pg.gu_id ILIKE $1 OR pg.gu_id = ANY($%d)))`, argIdx))
			args = append(args, guIDs)
			argIdx++
		} else {
			clauses = append(clauses, `EXISTS (SELECT 1 FROM player_gu pg WHERE pg.player_id = players.id AND pg.gu_id ILIKE $1)`)
		}

		return "(" + strings.Join(clauses, " OR ") + ")", args
	}
}

func AdminListPlayers(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")
	searchBy := c.DefaultQuery("search_by", "all")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	where, args := adminPlayerSearchClause(searchBy, search)

	var total int
	countQuery := `SELECT COUNT(*) FROM players`
	if where != "" {
		countQuery += ` WHERE ` + where
	}
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm số lượng người chơi"})
		return
	}

	query := `SELECT id, username, COALESCE(email, ''), rank, sub_rank, current_mana, max_mana, hp, base_hp, primeval_stones, immortal_stones, COALESCE(is_admin, false), created_at, COALESCE(current_region, ''), COALESCE(aptitude_type, ''), wall_progress, human_qi, lifespan FROM players`
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)

	dataArgs := append(append([]interface{}{}, args...), limit, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách người chơi"})
		return
	}
	defer rows.Close()

	players := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var username, email, currentRegion, aptitudeType string
		var rank, subRank, primevalStones int
		var currentMana, maxMana, hp, baseHP, immortalStones, wallProgress, humanQi, lifespan float64
		var isAdmin bool
		var createdAt time.Time

		if err := rows.Scan(&id, &username, &email, &rank, &subRank, &currentMana, &maxMana, &hp, &baseHP, &primevalStones, &immortalStones, &isAdmin, &createdAt, &currentRegion, &aptitudeType, &wallProgress, &humanQi, &lifespan); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc dữ liệu người chơi"})
			return
		}
		players = append(players, gin.H{
			"id":              id,
			"username":        username,
			"email":           email,
			"rank":            rank,
			"sub_rank":        subRank,
			"current_mana":    currentMana,
			"max_mana":        maxMana,
			"hp":              hp,
			"base_hp":         baseHP,
			"primeval_stones": primevalStones,
			"immortal_stones": immortalStones,
			"is_admin":        isAdmin,
			"created_at":      createdAt,
			"current_region":  currentRegion,
			"aptitude_type":   aptitudeType,
			"wall_progress":   wallProgress,
			"human_qi":        humanQi,
			"lifespan":        lifespan,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  players,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func AdminGetPlayerDetail(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	player, err := loadPlayerForStatus(db.DB, playerID, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	var guCount int
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM player_gu WHERE player_id=$1`, playerID).Scan(&guCount)
	var itemCount int
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM player_inventory_items WHERE player_id=$1`, playerID).Scan(&itemCount)

	c.JSON(http.StatusOK, gin.H{
		"id":                   player.ID,
		"username":             player.Username,
		"email":                player.Email,
		"rank":                 player.Rank,
		"sub_rank":             player.SubRank,
		"aptitude_type":        player.AptitudeType,
		"aptitude_percent":     player.AptitudePercent,
		"aptitude_grade":       player.AptitudeGrade,
		"current_mana":         player.CurrentMana,
		"max_mana":             player.MaxMana,
		"primeval_stones":      player.PrimevalStones,
		"immortal_stones":      player.ImmortalStones,
		"hp":                   player.HP,
		"base_hp":              player.BaseHP,
		"wall_progress":        player.WallProgress,
		"human_qi":             player.HumanQi,
		"current_region":       player.CurrentRegion,
		"is_admin":             player.IsAdmin,
		"created_at":           player.CreatedAt,
		"lifespan":             player.Lifespan,
		"soul_hp":              player.SoulHP,
		"max_soul":             player.MaxSoul,
		"soul_tier":            player.SoulTier,
		"explore_count":        player.ExploreCount,
		"aperture_stress":      player.ApertureStress,
		"avatar_url":           player.AvatarURL,
		"bio":                  player.Bio,
		"gu_count":             guCount,
		"item_count":           itemCount,
	})
}

type AdminItemRequest struct {
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

// adminItemKind returns "item" if the ID is a material item, "recipe" if it is
// a refining recipe (Cổ phương), or "" if the ID matches neither.
func adminItemKind(itemID string) string {
	if _, ok := models.ItemRegistry[itemID]; ok {
		return "item"
	}
	var exists bool
	if err := db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM refining_recipes WHERE id = $1)`, itemID).Scan(&exists); err == nil && exists {
		return "recipe"
	}
	return ""
}

func AdminAddPlayerItems(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
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
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.add_items",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: fmt.Sprintf("%s=%s qty=%d", kind, req.ItemID, req.Quantity),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm thành công", "kind": kind, "item_id": req.ItemID, "quantity": req.Quantity})
}

func AdminRemovePlayerItems(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
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
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.remove_items",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: fmt.Sprintf("%s=%s qty=%d", kind, req.ItemID, req.Quantity),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thành công", "kind": kind, "item_id": req.ItemID, "quantity": req.Quantity})
}

type AdminAddGuRequest struct {
	GuID     string `json:"gu_id"`
	Quantity int    `json:"quantity"`
}

func AdminAddPlayerGus(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req AdminAddGuRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.GuID == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if _, exists := models.GetGuAny(req.GuID); !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng không tồn tại"})
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

	guSpec, _ := models.GetGuAny(req.GuID)
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

	if err := db.SavePlayerGus(tx, playerID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cổ trùng"})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.add_gus",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: fmt.Sprintf("gu=%s qty=%d", req.GuID, req.Quantity),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm cổ trùng thành công", "gu_id": req.GuID, "quantity": req.Quantity})
}

func AdminRemovePlayerGu(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
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

	found := false
	newInv := models.GuInventory{}
	for _, gu := range player.GuInventory {
		if gu.InstanceID == instanceID {
			found = true
			continue
		}
		newInv = append(newInv, gu)
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy cổ trùng này"})
		return
	}

	if err := db.SavePlayerGus(tx, playerID.String(), newInv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật cổ trùng"})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.remove_gu",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: fmt.Sprintf("instance=%s", instanceID),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa cổ trùng thành công"})
}

func AdminChangePlayerStones(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req struct {
		PrimevalStones *int     `json:"primeval_stones"`
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
	summary := ""
	if req.PrimevalStones != nil {
		summary += fmt.Sprintf("primeval_stones=%d ", *req.PrimevalStones)
	}
	if req.ImmortalStones != nil {
		summary += fmt.Sprintf("immortal_stones=%.0f ", *req.ImmortalStones)
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.change_stones",
		EntityType: "player", EntityID: playerID.String(),
		RequestSummary: summary,
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật tài nguyên thành công"})
}

// AdminListPlayerItems returns a paginated, lazily-loaded list of a player's inventory items.
func AdminListPlayerItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM players WHERE id=$1)`, playerID).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	where := `player_id = $1`
	args := []interface{}{playerID}
	if search != "" {
		like := "%" + search + "%"
		itemIDs := models.ItemIDsByName(search)
		if len(itemIDs) > 0 {
			where += ` AND (item_id ILIKE $2 OR item_id = ANY($3))`
			args = append(args, like, itemIDs)
		} else {
			where += ` AND item_id ILIKE $2`
			args = append(args, like)
		}
	}

	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM player_inventory_items WHERE `+where, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm vật phẩm"})
		return
	}

	query := `SELECT item_id, category, quantity FROM player_inventory_items WHERE ` + where + ` ORDER BY item_id ASC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	dataArgs := append(append([]interface{}{}, args...), limit, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải vật phẩm"})
		return
	}
	defer rows.Close()

	data := []gin.H{}
	for rows.Next() {
		var itemID, category string
		var quantity int
		if err := rows.Scan(&itemID, &category, &quantity); err != nil {
			continue
		}
		spec, ok := models.ItemRegistry[itemID]
		name, itemType := itemID, ""
		rank := 0
		if ok {
			name = spec.Name
			itemType = spec.Type
			rank = spec.Rank
		}
		data = append(data, gin.H{
			"item_id":   itemID,
			"name":      name,
			"item_type": itemType,
			"rank":      rank,
			"category":  category,
			"quantity":  quantity,
		})
	}

	// Include unlocked recipes (Cổ phương) so admins can view/remove them too.
	recipeCount := 0
	if recipeRows, err := db.DB.QueryContext(c.Request.Context(), `SELECT recipe_id FROM player_recipe_progress WHERE player_id = $1 AND is_unlocked = true ORDER BY recipe_id`, playerID); err == nil {
		defer recipeRows.Close()
		for recipeRows.Next() {
			var recipeID string
			if err := recipeRows.Scan(&recipeID); err != nil {
				continue
			}
			name := recipeID
			var targetGuID string
			_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(target_gu_id,'') FROM refining_recipes WHERE id = $1`, recipeID).Scan(&targetGuID)
			if spec, ok := models.GetGuAny(targetGuID); ok && spec.Name != "" {
				name = spec.Name
			}
			data = append(data, gin.H{
				"item_id":   recipeID,
				"name":      name,
				"item_type": "recipe",
				"rank":      0,
				"category":  "recipe",
				"quantity":  1,
			})
			recipeCount++
		}
	}
	total += recipeCount

	c.JSON(http.StatusOK, gin.H{"data": data, "total": total})
}

// AdminListPlayerGus returns a paginated, lazily-loaded list of a player's Gu instances.
func AdminListPlayerGus(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	inv, err := db.LoadPlayerGus(db.DB, playerID.String(), false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	if search != "" {
		lower := strings.ToLower(search)
		guIDs := models.GuIDsByName(search)
		filtered := models.GuInventory{}
		for _, gu := range inv {
			if strings.Contains(strings.ToLower(gu.GuID), lower) || strings.Contains(strings.ToLower(gu.InstanceID), lower) {
				filtered = append(filtered, gu)
				continue
			}
			for _, gid := range guIDs {
				if gu.GuID == gid {
					filtered = append(filtered, gu)
					break
				}
			}
		}
		inv = filtered
	}

	total := len(inv)
	start := offset
	if start > total {
		start = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	data := []gin.H{}
	for _, gu := range inv[start:end] {
		name := gu.GuID
		if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
			name = spec.Name
		} else if spec, ok := models.GetGuAny(gu.GuID); ok {
			name = spec.Name
		}
		data = append(data, gin.H{
			"instance_id": gu.InstanceID,
			"gu_id":       gu.GuID,
			"name":        name,
			"rank":        gu.Rank,
			"hunger":      gu.Hunger,
			"status":      gu.Status,
			"equipped":    gu.Equipped,
			"is_vital":    gu.IsVital,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": data, "total": total})
}

type AdminUpdatePlayerRequest struct {
	Rank            *int     `json:"rank"`
	SubRank         *int     `json:"sub_rank"`
	AptitudeType    *string  `json:"aptitude_type"`
	AptitudePercent *float64 `json:"aptitude_percent"`
	AptitudeGrade   *string  `json:"aptitude_grade"`
	CurrentMana     *float64 `json:"current_mana"`
	MaxMana         *float64 `json:"max_mana"`
	HP              *float64 `json:"hp"`
	BaseHP          *float64 `json:"base_hp"`
	WallProgress    *float64 `json:"wall_progress"`
	HumanQi         *float64 `json:"human_qi"`
	Lifespan        *float64 `json:"lifespan"`
	CurrentRegion   *string  `json:"current_region"`
	IsAdmin         *bool    `json:"is_admin"`
	SoulHP          *float64 `json:"soul_hp"`
	MaxSoul         *float64 `json:"max_soul"`
	SoulTier        *string  `json:"soul_tier"`
	ExploreCount    *int     `json:"explore_count"`
	ApertureStress  *float64 `json:"aperture_stress"`
	Bio             *string  `json:"bio"`
	AvatarURL       *string  `json:"avatar_url"`
}

// AdminUpdatePlayer allows admins to edit a wide range of player attributes in one call.
func AdminUpdatePlayer(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	var req AdminUpdatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
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

	sets := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Rank != nil {
		rank := *req.Rank
		if rank < 1 {
			rank = 1
		} else if rank > 9 {
			rank = 9
		}
		sets = append(sets, "rank=$"+strconv.Itoa(argIdx))
		args = append(args, rank)
		argIdx++
	}
	if req.SubRank != nil {
		subRank := *req.SubRank
		if subRank < 1 {
			subRank = 1
		} else if subRank > 4 {
			subRank = 4
		}
		sets = append(sets, "sub_rank=$"+strconv.Itoa(argIdx))
		args = append(args, subRank)
		argIdx++
	}
	if req.AptitudeType != nil {
		sets = append(sets, "aptitude_type=$"+strconv.Itoa(argIdx))
		args = append(args, *req.AptitudeType)
		argIdx++
	}
	if req.AptitudePercent != nil {
		pct := *req.AptitudePercent
		if pct < 0 {
			pct = 0
		} else if pct > 100 {
			pct = 100
		}
		sets = append(sets, "aptitude_percent=$"+strconv.Itoa(argIdx))
		args = append(args, pct)
		argIdx++
	}
	if req.AptitudeGrade != nil {
		sets = append(sets, "aptitude_grade=$"+strconv.Itoa(argIdx))
		args = append(args, *req.AptitudeGrade)
		argIdx++
	}
	if req.CurrentMana != nil {
		val := math.Max(0, *req.CurrentMana)
		sets = append(sets, "current_mana=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.MaxMana != nil {
		val := math.Max(0, *req.MaxMana)
		sets = append(sets, "max_mana=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.HP != nil {
		val := math.Max(0, *req.HP)
		sets = append(sets, "hp=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.BaseHP != nil {
		val := math.Max(0, *req.BaseHP)
		sets = append(sets, "base_hp=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.WallProgress != nil {
		val := math.Max(0, *req.WallProgress)
		sets = append(sets, "wall_progress=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.HumanQi != nil {
		val := math.Max(0, *req.HumanQi)
		sets = append(sets, "human_qi=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.Lifespan != nil {
		val := math.Max(0, *req.Lifespan)
		sets = append(sets, "lifespan=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.CurrentRegion != nil {
		sets = append(sets, "current_region=$"+strconv.Itoa(argIdx))
		args = append(args, *req.CurrentRegion)
		argIdx++
	}
	if req.IsAdmin != nil {
		sets = append(sets, "is_admin=$"+strconv.Itoa(argIdx))
		args = append(args, *req.IsAdmin)
		argIdx++
	}
	if req.SoulHP != nil {
		val := math.Max(0, *req.SoulHP)
		sets = append(sets, "soul_hp=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.MaxSoul != nil {
		val := math.Max(0, *req.MaxSoul)
		sets = append(sets, "max_soul=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.SoulTier != nil {
		sets = append(sets, "soul_tier=$"+strconv.Itoa(argIdx))
		args = append(args, *req.SoulTier)
		argIdx++
	}
	if req.ExploreCount != nil {
		val := *req.ExploreCount
		if val < 0 {
			val = 0
		}
		sets = append(sets, "explore_count=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.ApertureStress != nil {
		val := math.Max(0, *req.ApertureStress)
		sets = append(sets, "aperture_stress=$"+strconv.Itoa(argIdx))
		args = append(args, val)
		argIdx++
	}
	if req.Bio != nil {
		sets = append(sets, "bio=$"+strconv.Itoa(argIdx))
		args = append(args, *req.Bio)
		argIdx++
	}
	if req.AvatarURL != nil {
		sets = append(sets, "avatar_url=$"+strconv.Itoa(argIdx))
		args = append(args, *req.AvatarURL)
		argIdx++
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có trường nào để cập nhật"})
		return
	}
	args = append(args, playerID)
	if _, err := tx.Exec(`UPDATE players SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(argIdx), args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật thông tin người chơi"})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		Action:         "admin.update_player",
		EntityType:     "player",
		EntityID:       playerID.String(),
		RequestSummary: strings.Join(sets, ", "),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật thông tin người chơi"})
}

// AdminSearchGus searches for Gu instances across player inventories, auctions and shops.
func AdminSearchGus(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	guSearch := c.Query("gu_id")
	instSearch := c.Query("instance_id")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	if guSearch == "" && instSearch == "" {
		c.JSON(http.StatusOK, gin.H{"data": []gin.H{}, "total": 0})
		return
	}

	guWhere := ""
	invWhere := ""
	shopWhere := ""
	args := []interface{}{}
	argIdx := 1
	if guSearch != "" {
		like := "%" + guSearch + "%"
		guIDs := models.GuIDsByName(guSearch)
		if len(guIDs) > 0 {
			guWhere = fmt.Sprintf("(g->>'gu_id' ILIKE $%d OR g->>'gu_id' = ANY($%d))", argIdx, argIdx+1)
			invWhere = fmt.Sprintf("(pg.gu_id ILIKE $%d OR pg.gu_id = ANY($%d))", argIdx, argIdx+1)
			shopWhere = fmt.Sprintf("(s.id ILIKE $%d OR s.id = ANY($%d))", argIdx, argIdx+1)
			args = append(args, like, guIDs)
			argIdx += 2
		} else {
			guWhere = fmt.Sprintf("g->>'gu_id' ILIKE $%d", argIdx)
			invWhere = fmt.Sprintf("pg.gu_id ILIKE $%d", argIdx)
			shopWhere = fmt.Sprintf("s.id ILIKE $%d", argIdx)
			args = append(args, like)
			argIdx++
		}
	}
	if instSearch != "" {
		instClause := fmt.Sprintf("g->>'instance_id' ILIKE $%d", argIdx)
		invInstClause := fmt.Sprintf("pg.id::text ILIKE $%d", argIdx)
		args = append(args, "%"+instSearch+"%")
		argIdx++
		if guWhere == "" {
			guWhere = instClause
		} else {
			guWhere = "(" + guWhere + " OR " + instClause + ")"
		}
		if invWhere == "" {
			invWhere = invInstClause
		} else {
			invWhere = "(" + invWhere + " OR " + invInstClause + ")"
		}
	}

	inventoryBranch := `SELECT 'inventory' AS location,
		p.id AS owner_id, p.username AS owner_username,
		NULL::text AS listing_id,
		pg.id::text AS instance_id, pg.gu_id AS gu_id,
		pg.gu_rank AS rank,
		COALESCE(pg.hunger, 0) AS hunger,
		COALESCE(pg.state, '') AS status,
		COALESCE(pg.equipped, false) AS equipped,
		COALESCE(pg.is_vital, false) AS is_vital,
		'' AS listing_status, NULL::text AS shop_name
	FROM players p
	JOIN player_gu pg ON pg.player_id = p.id
	WHERE ` + invWhere

	auctionBranch := `SELECT 'auction' AS location,
		a.seller_id AS owner_id, pl.username AS owner_username,
		a.id::text AS listing_id,
		g->>'instance_id' AS instance_id, g->>'gu_id' AS gu_id,
		(g->>'rank')::int AS rank,
		COALESCE((g->>'hunger')::int, 0) AS hunger,
		COALESCE(g->>'status', '') AS status,
		COALESCE((g->>'equipped')::boolean, false) AS equipped,
		COALESCE((g->>'is_vital')::boolean, false) AS is_vital,
		COALESCE(a.status, '') AS listing_status, NULL::text AS shop_name
	FROM auction a
	CROSS JOIN LATERAL jsonb_array_elements(a.item_data) g
	JOIN players pl ON pl.id = a.seller_id
	WHERE a.item_type = 'Gu' AND ` + guWhere

	guDictSub := `(SELECT id, MAX(rank) AS rank, MIN(name) AS name FROM gu_dictionary GROUP BY id)`

	shopBranch := `SELECT 'shop' AS location,
		NULL::uuid AS owner_id, NULL::text AS owner_username,
		s.id AS listing_id,
		NULL::text AS instance_id, s.id AS gu_id,
		d.rank AS rank,
		NULL::int AS hunger,
		'SHOP' AS status,
		false AS equipped,
		false AS is_vital,
		'SHOP' AS listing_status, d.name AS shop_name
	FROM caravan_shop_items s
	JOIN ` + guDictSub + ` d ON d.id = s.id
	WHERE ` + shopWhere

	arenaShopBranch := `SELECT 'shop' AS location,
		NULL::uuid AS owner_id, NULL::text AS owner_username,
		s.id AS listing_id,
		NULL::text AS instance_id, s.id AS gu_id,
		d.rank AS rank,
		NULL::int AS hunger,
		'SHOP' AS status,
		false AS equipped,
		false AS is_vital,
		'SHOP' AS listing_status, d.name AS shop_name
	FROM arena_shop_items s
	JOIN ` + guDictSub + ` d ON d.id = s.id
	WHERE ` + shopWhere

	branches := []string{inventoryBranch, auctionBranch}
	if shopWhere != "" {
		branches = append(branches, shopBranch, arenaShopBranch)
	}
	union := strings.Join(branches, " UNION ALL ")

	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM (`+union+`) AS t`, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tìm cổ trùng"})
		return
	}

	query := `SELECT location, owner_id::text, owner_username, listing_id, instance_id, gu_id, rank, hunger, status, equipped, is_vital, listing_status, shop_name FROM (` + union + `) AS t ORDER BY t.location, t.owner_username, t.gu_id LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	dataArgs := append(append([]interface{}{}, args...), limit, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tìm cổ trùng"})
		return
	}
	defer rows.Close()

	data := []gin.H{}
	for rows.Next() {
		var location, ownerID, ownerUsername, listingID, instanceID, guID, status, listingStatus, shopName string
		var rank, hunger int
		var equipped, isVital bool
		if err := rows.Scan(&location, &ownerID, &ownerUsername, &listingID, &instanceID, &guID, &rank, &hunger, &status, &equipped, &isVital, &listingStatus, &shopName); err != nil {
			continue
		}
		name := guID
		if spec, ok := models.GetGu(guID, rank); ok {
			name = spec.Name
		} else if spec, ok := models.GetGuAny(guID); ok {
			name = spec.Name
		}
		data = append(data, gin.H{
			"location":       location,
			"owner_id":       ownerID,
			"owner_username": ownerUsername,
			"listing_id":     listingID,
			"instance_id":    instanceID,
			"gu_id":          guID,
			"name":           name,
			"rank":           rank,
			"hunger":         hunger,
			"status":         status,
			"equipped":       equipped,
			"is_vital":       isVital,
			"listing_status": listingStatus,
			"shop_name":      shopName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": data, "total": total})
}

// -------- Admin: Sect Management --------

func AdminListSects(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT s.id, s.name, s.description, s.region, s.level, s.funds, s.require_approval,
			COALESCE(m.member_count, 0) AS member_count,
			l.leader_name
		FROM sects s
		LEFT JOIN (
			SELECT sect_id, COUNT(*) AS member_count FROM sect_members GROUP BY sect_id
		) m ON m.sect_id = s.id
		LEFT JOIN (
			SELECT sm.sect_id, p.username AS leader_name
			FROM sect_members sm
			JOIN players p ON p.id = sm.player_id
			WHERE sm.member_rank = 'CHUONG_MON'
		) l ON l.sect_id = s.id
		ORDER BY s.name
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách tông môn"})
		return
	}
	defer rows.Close()

	type SectInfo struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Description     string `json:"description"`
		Region          string `json:"region"`
		Level           int    `json:"level"`
		Funds           int    `json:"funds"`
		RequireApproval bool   `json:"require_approval"`
		MemberCount     int    `json:"member_count"`
		LeaderName      string `json:"leader_name,omitempty"`
	}

	var sects []SectInfo
	for rows.Next() {
		var s SectInfo
		var leaderName *string
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.Level, &s.Funds, &s.RequireApproval, &s.MemberCount, &leaderName); err != nil {
			continue
		}
		if leaderName != nil {
			s.LeaderName = *leaderName
		}
		sects = append(sects, s)
	}

	c.JSON(http.StatusOK, sects)
}

func AdminGetSectDetail(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	sectID := c.Param("id")

	type sectVault struct {
		ID              string          `json:"id"`
		Name            string          `json:"name"`
		Description     string          `json:"description"`
		Region          string          `json:"region"`
		Level           int             `json:"level"`
		Funds           int             `json:"funds"`
		RequireApproval bool            `json:"require_approval"`
		VaultGu         json.RawMessage `json:"vault_gu"`
		VaultMaterials  json.RawMessage `json:"vault_materials"`
		VaultRecipes    json.RawMessage `json:"vault_recipes"`
	}
	var s sectVault
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT id, name, description, region, level, funds, require_approval, vault_gu, vault_materials, vault_recipes FROM sects WHERE id = $1`, sectID).Scan(&s.ID, &s.Name, &s.Description, &s.Region, &s.Level, &s.Funds, &s.RequireApproval, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT sm.player_id, p.username, sm.member_rank, sm.contribution_points, sm.joined_at, p.rank
		FROM sect_members sm
		JOIN players p ON p.id = sm.player_id
		WHERE sm.sect_id = $1
		ORDER BY
			CASE sm.member_rank
				WHEN 'CHUONG_MON' THEN 1
				WHEN 'TRUONG_LAO' THEN 2
				WHEN 'CHAN_TRUYEN' THEN 3
				WHEN 'TINH_ANH' THEN 4
				WHEN 'NOI_MON' THEN 5
				ELSE 6
			END, sm.contribution_points DESC
	`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thành viên"})
		return
	}
	defer rows.Close()

	type MemberInfo struct {
		PlayerID     uuid.UUID `json:"player_id"`
		Username     string    `json:"username"`
		MemberRank   string    `json:"member_rank"`
		Contribution int       `json:"contribution_points"`
		JoinedAt     time.Time `json:"joined_at"`
		PlayerRank   int       `json:"player_rank"`
	}

	var members []MemberInfo
	for rows.Next() {
		var m MemberInfo
		if err := rows.Scan(&m.PlayerID, &m.Username, &m.MemberRank, &m.Contribution, &m.JoinedAt, &m.PlayerRank); err == nil {
			members = append(members, m)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"sect":    s,
		"members": members,
	})
}

type AdminSectUpdateRequest struct {
	Name            *string `json:"name"`
	Description     *string `json:"description"`
	Region          *string `json:"region"`
	Level           *int    `json:"level"`
	Funds           *int    `json:"funds"`
	RequireApproval *bool   `json:"require_approval"`
}

func AdminUpdateSect(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	sectID := c.Param("id")

	var req AdminSectUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	sets := []string{}
	args := []interface{}{}
	idx := 1

	if req.Name != nil {
		sets = append(sets, "name=$"+strconv.Itoa(idx))
		args = append(args, *req.Name)
		idx++
	}
	if req.Description != nil {
		sets = append(sets, "description=$"+strconv.Itoa(idx))
		args = append(args, *req.Description)
		idx++
	}
	if req.Region != nil {
		sets = append(sets, "region=$"+strconv.Itoa(idx))
		args = append(args, *req.Region)
		idx++
	}
	if req.Level != nil {
		sets = append(sets, "level=$"+strconv.Itoa(idx))
		args = append(args, *req.Level)
		idx++
	}
	if req.Funds != nil {
		sets = append(sets, "funds=$"+strconv.Itoa(idx))
		args = append(args, *req.Funds)
		idx++
	}
	if req.RequireApproval != nil {
		sets = append(sets, "require_approval=$"+strconv.Itoa(idx))
		args = append(args, *req.RequireApproval)
		idx++
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu cập nhật"})
		return
	}

	args = append(args, sectID)
	query := `UPDATE sects SET ` + strings.Join(sets, ", ") + ` WHERE id=$` + strconv.Itoa(idx)

	if _, err := db.DB.Exec(query, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tông môn"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.update_sect",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("sets=%d", len(sets)),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật tông môn thành công"})
}

func AdminSetSectLeader(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	sectID := c.Param("id")
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID thành viên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	// Verify member belongs to the sect
	var currentRank string
	err = tx.QueryRow(`SELECT member_rank FROM sect_members WHERE player_id = $1 AND sect_id = $2`, memberID, sectID).Scan(&currentRank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thành viên không thuộc tông môn này"})
		return
	}

	// Demote all existing leaders
	_, err = tx.Exec(`UPDATE sect_members SET member_rank = 'TRUONG_LAO' WHERE sect_id = $1 AND member_rank = 'CHUONG_MON'`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thay đổi chức vụ"})
		return
	}

	// Set new leader
	_, err = tx.Exec(`UPDATE sect_members SET member_rank = 'CHUONG_MON' WHERE player_id = $1 AND sect_id = $2`, memberID, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bổ nhiệm tân Chưởng môn"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.set_sect_leader",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("memberId=%s", memberIDStr),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã bổ nhiệm tân Chưởng môn thành công"})
}

func AdminKickSectMember(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	sectID := c.Param("id")
	memberIDStr := c.Param("memberId")
	memberID, err := uuid.Parse(memberIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID thành viên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	var currentRank string
	err = tx.QueryRow(`SELECT member_rank FROM sect_members WHERE player_id = $1 AND sect_id = $2`, memberID, sectID).Scan(&currentRank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thành viên không thuộc tông môn này"})
		return
	}

	// Delete active missions for this member
	_, _ = tx.Exec(`DELETE FROM player_sect_missions WHERE player_id = $1 AND mission_id IN (SELECT id FROM sect_missions WHERE sect_id = $2)`, memberID, sectID)

	// Delete member
	_, err = tx.Exec(`DELETE FROM sect_members WHERE player_id = $1 AND sect_id = $2`, memberID, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa thành viên"})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.kick_sect_member",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("memberId=%s", memberIDStr),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa thành viên khỏi tông môn"})
}

type AdminSectAddMemberRequest struct {
	PlayerID   string `json:"player_id"`
	MemberRank string `json:"member_rank"`
}

func AdminAddToSect(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	sectID := c.Param("id")

	var req AdminSectAddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlayerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	if req.MemberRank == "" {
		req.MemberRank = "NGOAI_MON"
	}
	validRanks := map[string]bool{"NGOAI_MON": true, "NOI_MON": true, "TINH_ANH": true, "CHAN_TRUYEN": true, "TRUONG_LAO": true, "CHUONG_MON": true}
	if !validRanks[req.MemberRank] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chức vụ không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	// Check player exists
	var username string
	err = tx.QueryRow(`SELECT username FROM players WHERE id = $1`, playerUUID).Scan(&username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	// Check if already in a sect
	var existingSectID string
	err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerUUID).Scan(&existingSectID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người chơi đã ở trong tông môn " + existingSectID})
		return
	}

	// Verify sect exists
	var sectExists bool
	err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM sects WHERE id = $1)`, sectID).Scan(&sectExists)
	if err != nil || !sectExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	// If being added as leader, demote existing leader first
	if req.MemberRank == "CHUONG_MON" {
		_, _ = tx.Exec(`UPDATE sect_members SET member_rank = 'TRUONG_LAO' WHERE sect_id = $1 AND member_rank = 'CHUONG_MON'`, sectID)
	}

	_, err = tx.Exec(`INSERT INTO sect_members (sect_id, player_id, member_rank, contribution_points) VALUES ($1, $2, $3, 0)`, sectID, playerUUID, req.MemberRank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm thành viên"})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.add_to_sect",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("player=%s rank=%s", req.PlayerID, req.MemberRank),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm " + username + " vào tông môn thành công"})
}

// -------- Admin: Sect Vault --------

type AdminStockVaultRequest struct {
	ItemType string `json:"item_type"` // gu, material, recipe
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

func AdminStockSectVault(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	sectID := c.Param("id")
	var req AdminStockVaultRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemID == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	var vaultGuJSON, vaultMatsJSON, vaultRecipesJSON []byte
	err = tx.QueryRow(`SELECT vault_gu, vault_materials, vault_recipes FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&vaultGuJSON, &vaultMatsJSON, &vaultRecipesJSON)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	switch req.ItemType {
	case "gu":
		var guItems []SectVaultGuItem
		json.Unmarshal(vaultGuJSON, &guItems)
		if guItems == nil {
			guItems = []SectVaultGuItem{}
		}
		found := false
		for i, item := range guItems {
			if item.ID == req.ItemID {
				guItems[i].Quantity += req.Quantity
				found = true
				break
			}
		}
		if !found {
			guItems = append(guItems, SectVaultGuItem{ID: req.ItemID, Cost: 0, Quantity: req.Quantity})
		}
		updated, _ := json.Marshal(guItems)
		_, err = tx.Exec(`UPDATE sects SET vault_gu=$1 WHERE id=$2`, updated, sectID)

	case "material":
		var mats map[string]int
		json.Unmarshal(vaultMatsJSON, &mats)
		if mats == nil {
			mats = map[string]int{}
		}
		mats[req.ItemID] += req.Quantity
		updated, _ := json.Marshal(mats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials=$1 WHERE id=$2`, updated, sectID)

	case "recipe":
		var recipes []string
		json.Unmarshal(vaultRecipesJSON, &recipes)
		if recipes == nil {
			recipes = []string{}
		}
		recipes = append(recipes, req.ItemID)
		updated, _ := json.Marshal(recipes)
		_, err = tx.Exec(`UPDATE sects SET vault_recipes=$1 WHERE id=$2`, updated, sectID)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm không hợp lệ (gu, material, recipe)"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố"})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.stock_sect_vault",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("type=%s item=%s qty=%d", req.ItemType, req.ItemID, req.Quantity),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm vào bảo khố thành công"})
}

func AdminRemoveSectVaultItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	sectID := c.Param("id")
	var req AdminStockVaultRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ItemID == "" || req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	var vaultGuJSON, vaultMatsJSON, vaultRecipesJSON []byte
	err = tx.QueryRow(`SELECT vault_gu, vault_materials, vault_recipes FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&vaultGuJSON, &vaultMatsJSON, &vaultRecipesJSON)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	switch req.ItemType {
	case "gu":
		var guItems []SectVaultGuItem
		json.Unmarshal(vaultGuJSON, &guItems)
		found := false
		for i, item := range guItems {
			if item.ID == req.ItemID {
				if item.Quantity < req.Quantity {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ số lượng trong bảo khố"})
					return
				}
				guItems[i].Quantity -= req.Quantity
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy vật phẩm trong bảo khố"})
			return
		}
		updated, _ := json.Marshal(guItems)
		_, err = tx.Exec(`UPDATE sects SET vault_gu=$1 WHERE id=$2`, updated, sectID)

	case "material":
		var mats map[string]int
		json.Unmarshal(vaultMatsJSON, &mats)
		if mats[req.ItemID] < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ số lượng trong bảo khố"})
			return
		}
		mats[req.ItemID] -= req.Quantity
		if mats[req.ItemID] <= 0 {
			delete(mats, req.ItemID)
		}
		updated, _ := json.Marshal(mats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials=$1 WHERE id=$2`, updated, sectID)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm không hợp lệ"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bảo khố"})
		return
	}

	tx.Commit()
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.remove_sect_vault",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("type=%s item=%s qty=%d", req.ItemType, req.ItemID, req.Quantity),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa khỏi bảo khố thành công"})
}

// -------- Admin: Sect Missions --------

func AdminListSectMissions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	sectID := c.Param("id")

	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank FROM sect_missions WHERE sect_id = $1 ORDER BY required_member_rank, title`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhiệm vụ"})
		return
	}
	defer rows.Close()

	type Mission struct {
		ID                  string `json:"id"`
		SectID              string `json:"sect_id"`
		Title               string `json:"title"`
		Description         string `json:"description"`
		MissionType         string `json:"mission_type"`
		TargetID            string `json:"target_id"`
		TargetAmount        int    `json:"target_amount"`
		RewardContribution  int    `json:"reward_contribution"`
		RequiredMemberRank  string `json:"required_member_rank"`
	}

	var missions []Mission
	for rows.Next() {
		var m Mission
		if err := rows.Scan(&m.ID, &m.SectID, &m.Title, &m.Description, &m.MissionType, &m.TargetID, &m.TargetAmount, &m.RewardContribution, &m.RequiredMemberRank); err == nil {
			missions = append(missions, m)
		}
	}

	c.JSON(http.StatusOK, missions)
}

type AdminCreateMissionRequest struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	MissionType        string `json:"mission_type"` // GATHER, DONATE, COMBAT, REFINE
	TargetID           string `json:"target_id"`
	TargetAmount       int    `json:"target_amount"`
	RewardContribution int    `json:"reward_contribution"`
	RequiredMemberRank string `json:"required_member_rank"`
}

func AdminCreateSectMission(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	sectID := c.Param("id")

	var req AdminCreateMissionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Title == "" || req.MissionType == "" || req.TargetID == "" || req.TargetAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	if req.RequiredMemberRank == "" {
		req.RequiredMemberRank = "NGOAI_MON"
	}

	_, err := db.DB.Exec(`INSERT INTO sect_missions (sect_id, title, description, mission_type, target_id, target_amount, reward_contribution, required_member_rank) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		sectID, req.Title, req.Description, req.MissionType, req.TargetID, req.TargetAmount, req.RewardContribution, req.RequiredMemberRank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo nhiệm vụ"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.create_sect_mission",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("title=%s type=%s target=%s qty=%d", req.Title, req.MissionType, req.TargetID, req.TargetAmount),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo nhiệm vụ thành công"})
}

func AdminDeleteSectMission(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	missionID := c.Param("missionId")

	_, err := db.DB.Exec(`DELETE FROM sect_missions WHERE id = $1`, missionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa nhiệm vụ"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.delete_sect_mission",
		EntityType: "sect_mission", EntityID: missionID,
		RequestSummary: "",
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa nhiệm vụ thành công"})
}

func AdminDeleteSect(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	sectID := c.Param("id")

	// Verify sect exists before deleting
	var name string
	err := db.DB.QueryRow(`SELECT name FROM sects WHERE id = $1`, sectID).Scan(&name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn"})
		return
	}

	// CASCADE will delete sect_members, sect_missions, player_sect_missions, etc
	if _, err := db.DB.Exec(`DELETE FROM sects WHERE id = $1`, sectID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa tông môn"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.delete_sect",
		EntityType: "sect", EntityID: sectID,
		RequestSummary: fmt.Sprintf("name=%s", name),
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã giải tán tông môn " + name})
}

// AdminImportData seeds all dictionary tables from CSV files.
// POST /api/admin/data/import
func AdminImportData(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	if err := db.SeedFromCSV(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Import thất bại: " + err.Error()})
		return
	}

	// Reload registries
	if err := models.LoadGuRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Gu registry after import: %v", err)
	}
	if err := models.LoadItemRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Item registry after import: %v", err)
	}
	if err := models.LoadMonsterRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Monster registry after import: %v", err)
	}
	if err := LoadQuestRegistry(); err != nil {
		logger.S.Warnf("failed to reload Quest registry after import: %v", err)
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.import_data",
		EntityType: "system", EntityID: "",
		RequestSummary: "data import from CSV",
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Import dữ liệu từ CSV thành công"})
}

// AdminExportData exports all dictionary tables to CSV files.
// POST /api/admin/data/export
func AdminExportData(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	if err := db.ExportToCSV(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Export thất bại: " + err.Error()})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &adminID, Action: "admin.export_data",
		EntityType: "system", EntityID: "",
		RequestSummary: "data export to CSV",
		IPAddress: c.ClientIP(), Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Export dữ liệu ra CSV thành công"})
}

// AdminClearCache reloads all in-memory registries from the database and clears Redis caches.
// POST /api/admin/clear-cache
func AdminClearCache(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	if err := models.LoadGuRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Gu registry: %v", err)
	}
	if err := models.LoadItemRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Item registry: %v", err)
	}
	if err := models.LoadMonsterRegistry(db.DB); err != nil {
		logger.S.Warnf("failed to reload Monster registry: %v", err)
	}
	if err := LoadQuestRegistry(); err != nil {
		logger.S.Warnf("failed to reload Quest registry: %v", err)
	}

	// Clear all known Redis cache keys
	cache.Delete(
		"game_dictionary_unified",
		"game_quests_list",
		"caravan_shop_catalog",
		"master_refining_recipes",
		"market_listings_public",
	)
	// Also clear profile page cache keys (pattern-based, use a wildcard approach)
	if cache.Client != nil {
		iter := cache.Client.Scan(cache.Ctx, 0, "profile_*", 100).Iterator()
		for iter.Next(cache.Ctx) {
			cache.Client.Del(cache.Ctx, iter.Val())
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa và tải lại toàn bộ cache dữ liệu thành công!"})
}

// AdminGetAuditLogs returns all audit logs (admin only) with optional filtering.
// GET /api/admin/audit-logs
func AdminGetAuditLogs(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	action := c.Query("action")
	entityType := c.Query("entity_type")
	playerIDStr := c.Query("player_id")
	successStr := c.Query("success")
	search := c.Query("search")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	query := `SELECT id, player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at FROM audit_logs WHERE 1=1`
	args := []interface{}{}
	argCount := 0

	if action != "" {
		argCount++
		query += " AND action = $" + strconv.Itoa(argCount)
		args = append(args, action)
	}
	if entityType != "" {
		argCount++
		query += " AND entity_type = $" + strconv.Itoa(argCount)
		args = append(args, entityType)
	}
	if playerIDStr != "" {
		pid, err := uuid.Parse(playerIDStr)
		if err == nil {
			argCount++
			query += " AND player_id = $" + strconv.Itoa(argCount)
			args = append(args, pid)
		}
	}
	if successStr != "" {
		success, err := strconv.ParseBool(successStr)
		if err == nil {
			argCount++
			query += " AND success = $" + strconv.Itoa(argCount)
			args = append(args, success)
		}
	}
	if search != "" {
		argCount++
		query += " AND (action ILIKE $" + strconv.Itoa(argCount) + " OR request_summary ILIKE $" + strconv.Itoa(argCount) + " OR entity_id ILIKE $" + strconv.Itoa(argCount) + " OR error_message ILIKE $" + strconv.Itoa(argCount) + ")"
		args = append(args, "%"+search+"%")
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)
	args = append(args, limit, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể truy vấn nhật ký: " + err.Error()})
		return
	}
	defer rows.Close()

	logs := []AuditLogResponse{}
	for rows.Next() {
		var logRow AuditLogResponse
		var metaJSON []byte
		err := rows.Scan(
			&logRow.ID, &logRow.PlayerID, &logRow.Action, &logRow.EntityType,
			&logRow.EntityID, &logRow.RequestSummary, &metaJSON, &logRow.IPAddress,
			&logRow.Success, &logRow.ErrorMessage, &logRow.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc nhật ký: " + err.Error()})
			return
		}
		if len(metaJSON) > 0 {
			_ = json.Unmarshal(metaJSON, &logRow.Metadata)
		}
		logs = append(logs, logRow)
	}

	// Get total count for pagination
	countQuery := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`
	countArgs := make([]interface{}, 0)
	ci := 0
	if action != "" {
		ci++
		countQuery += " AND action = $" + strconv.Itoa(ci)
		countArgs = append(countArgs, action)
	}
	if entityType != "" {
		ci++
		countQuery += " AND entity_type = $" + strconv.Itoa(ci)
		countArgs = append(countArgs, entityType)
	}
	if playerIDStr != "" {
		pid, err := uuid.Parse(playerIDStr)
		if err == nil {
			ci++
			countQuery += " AND player_id = $" + strconv.Itoa(ci)
			countArgs = append(countArgs, pid)
		}
	}
	if successStr != "" {
		success, err := strconv.ParseBool(successStr)
		if err == nil {
			ci++
			countQuery += " AND success = $" + strconv.Itoa(ci)
			countArgs = append(countArgs, success)
		}
	}
	if search != "" {
		ci++
		countQuery += " AND (action ILIKE $" + strconv.Itoa(ci) + " OR request_summary ILIKE $" + strconv.Itoa(ci) + " OR entity_id ILIKE $" + strconv.Itoa(ci) + " OR error_message ILIKE $" + strconv.Itoa(ci) + ")"
		countArgs = append(countArgs, "%"+search+"%")
	}

	var total int
	_ = db.DB.QueryRowContext(c.Request.Context(), countQuery, countArgs...).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// -------- Admin: Shop Management --------

type AdminShopItem struct {
	ID        string `json:"id"`
	Price     int    `json:"price"`
	Stock     int    `json:"stock"`
	IsDefault bool   `json:"is_default"`
	SortOrder int    `json:"sort_order"`
	Name      string `json:"name,omitempty"`
	ItemType  string `json:"item_type,omitempty"`
}

func AdminListShopItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT c.id, c.price, c.stock, c.is_default, c.sort_order,
			COALESCE(i.name, g.name, 'Công thức ' || rg.name, c.id) AS name,
			CASE WHEN r.id IS NOT NULL THEN 'Recipe' ELSE COALESCE(i.item_type, 'Gu') END AS item_type
		FROM caravan_shop_items c
		LEFT JOIN item_dictionary i ON c.id = i.id
		LEFT JOIN gu_dictionary g ON c.id = g.id
		LEFT JOIN refining_recipes r ON c.id = r.id
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		ORDER BY c.sort_order, c.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách shop: " + err.Error()})
		return
	}
	defer rows.Close()

	items := []AdminShopItem{}
	for rows.Next() {
		var item AdminShopItem
		if err := rows.Scan(&item.ID, &item.Price, &item.Stock, &item.IsDefault, &item.SortOrder, &item.Name, &item.ItemType); err != nil {
			continue
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

type AdminShopItemRequest struct {
	ID        string `json:"id"`
	Price     int    `json:"price"`
	Stock     int    `json:"stock"`
	IsDefault bool   `json:"is_default"`
	SortOrder int    `json:"sort_order"`
}

func AdminCreateShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	var req AdminShopItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" || req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Verify item exists in item_dictionary, gu_dictionary, or refining_recipes
	var exists bool
	_ = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM item_dictionary WHERE id=$1 UNION SELECT 1 FROM gu_dictionary WHERE id=$1 UNION SELECT 1 FROM refining_recipes WHERE id=$1)`, req.ID).Scan(&exists)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong item_dictionary, gu_dictionary hoặc công thức"})
		return
	}

	_, err := db.DB.Exec(`INSERT INTO caravan_shop_items (id, price, stock, is_default, sort_order) VALUES ($1,$2,$3,$4,$5)`,
		req.ID, req.Price, req.Stock, req.IsDefault, req.SortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.create_shop_item", "id=%s price=%d stock=%d", req.ID, req.Price, req.Stock)
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm mặt hàng vào shop"})
}

func AdminUpdateShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	itemID := c.Param("id")
	var req AdminShopItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	_, err := db.DB.Exec(`UPDATE caravan_shop_items SET price=$1, stock=$2, is_default=$3, sort_order=$4, updated_at=NOW() WHERE id=$5`,
		req.Price, req.Stock, req.IsDefault, req.SortOrder, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.update_shop_item", "id=%s price=%d stock=%d is_default=%v", itemID, req.Price, req.Stock, req.IsDefault)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật mặt hàng"})
}

func AdminDeleteShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	itemID := c.Param("id")

	_, err := db.DB.Exec(`DELETE FROM caravan_shop_items WHERE id=$1`, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.delete_shop_item", "id=%s", itemID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa mặt hàng khỏi shop"})
}

// -------- Admin: Arena Shop Management --------

type AdminArenaShopItem struct {
	ID        string `json:"id"`
	Price     int    `json:"price"`
	Qty       int    `json:"qty"`
	Stock     int    `json:"stock"`
	IsDefault bool   `json:"is_default"`
	SortOrder int    `json:"sort_order"`
	Name      string `json:"name,omitempty"`
}

func AdminListArenaShopItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT a.id, a.price, a.qty, a.stock, a.is_default, a.sort_order,
			COALESCE(i.name, g.name, 'Công thức ' || rg.name, a.id) AS name
		FROM arena_shop_items a
		LEFT JOIN item_dictionary i ON a.id = i.id
		LEFT JOIN gu_dictionary g ON a.id = g.id
		LEFT JOIN refining_recipes r ON a.id = r.id
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		ORDER BY a.sort_order, a.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách arena shop: " + err.Error()})
		return
	}
	defer rows.Close()

	items := []AdminArenaShopItem{}
	for rows.Next() {
		var item AdminArenaShopItem
		if err := rows.Scan(&item.ID, &item.Price, &item.Qty, &item.Stock, &item.IsDefault, &item.SortOrder, &item.Name); err != nil {
			continue
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

type AdminArenaShopItemRequest struct {
	ID        string `json:"id"`
	Price     int    `json:"price"`
	Qty       int    `json:"qty"`
	Stock     int    `json:"stock"`
	IsDefault bool   `json:"is_default"`
	SortOrder int    `json:"sort_order"`
}

func AdminCreateArenaShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	var req AdminArenaShopItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" || req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.Qty <= 0 {
		req.Qty = 1
	}

	// Verify item exists in item_dictionary, gu_dictionary, or refining_recipes
	var exists bool
	_ = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM item_dictionary WHERE id=$1 UNION SELECT 1 FROM gu_dictionary WHERE id=$1 UNION SELECT 1 FROM refining_recipes WHERE id=$1)`, req.ID).Scan(&exists)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong item_dictionary, gu_dictionary hoặc công thức"})
		return
	}

	_, err := db.DB.Exec(`INSERT INTO arena_shop_items (id, price, qty, stock, is_default, sort_order) VALUES ($1,$2,$3,$4,$5,$6)`,
		req.ID, req.Price, req.Qty, req.Stock, req.IsDefault, req.SortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.create_arena_shop_item", "id=%s price=%d qty=%d stock=%d", req.ID, req.Price, req.Qty, req.Stock)
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm mặt hàng vào Arena Shop"})
}

func AdminUpdateArenaShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	itemID := c.Param("id")
	var req AdminArenaShopItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.Qty <= 0 {
		req.Qty = 1
	}

	_, err := db.DB.Exec(`UPDATE arena_shop_items SET price=$1, qty=$2, stock=$3, is_default=$4, sort_order=$5, updated_at=NOW() WHERE id=$6`,
		req.Price, req.Qty, req.Stock, req.IsDefault, req.SortOrder, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.update_arena_shop_item", "id=%s price=%d qty=%d stock=%d is_default=%v", itemID, req.Price, req.Qty, req.Stock, req.IsDefault)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật mặt hàng Arena Shop"})
}

func AdminDeleteArenaShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	itemID := c.Param("id")

	_, err := db.DB.Exec(`DELETE FROM arena_shop_items WHERE id=$1`, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.delete_arena_shop_item", "id=%s", itemID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa mặt hàng khỏi Arena Shop"})
}

// -------- Admin: Mine Slot Management --------

type AdminMineSlot struct {
	ID              int        `json:"id"`
	Rarity          string     `json:"rarity"`
	OccupierID      *string    `json:"occupier_id"`
	OccupierName    *string    `json:"occupier_name"`
	MonsterID       *string    `json:"monster_id"`
	MonsterRank     int        `json:"monster_rank"`
	MonsterSubRank  int        `json:"monster_sub_rank"`
	OccupiedAt      *time.Time `json:"occupied_at"`
	LastClaimedAt   *time.Time `json:"last_claimed_at"`
	AbandonedAt     *time.Time `json:"abandoned_at"`
}

func AdminListMineSlots(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT m.id, m.rarity, m.occupier_id, COALESCE(p.username, ''), m.monster_id, m.monster_rank, m.monster_sub_rank,
			m.occupied_at, m.last_claimed_at, m.abandoned_at
		FROM mine_slots m
		LEFT JOIN players p ON m.occupier_id = p.id
		ORDER BY m.id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách mỏ: " + err.Error()})
		return
	}
	defer rows.Close()

	slots := []AdminMineSlot{}
	for rows.Next() {
		var s AdminMineSlot
		var occupierName string
		if err := rows.Scan(&s.ID, &s.Rarity, &s.OccupierID, &occupierName, &s.MonsterID, &s.MonsterRank, &s.MonsterSubRank,
			&s.OccupiedAt, &s.LastClaimedAt, &s.AbandonedAt); err != nil {
			continue
		}
		if occupierName != "" {
			s.OccupierName = &occupierName
		}
		slots = append(slots, s)
	}

	c.JSON(http.StatusOK, slots)
}

type AdminUpdateMineSlotRequest struct {
	Rarity         *string `json:"rarity"`
	MonsterRank    *int    `json:"monster_rank"`
	MonsterSubRank *int    `json:"monster_sub_rank"`
}

func AdminUpdateMineSlot(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	slotID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID mỏ không hợp lệ"})
		return
	}

	var req AdminUpdateMineSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	sets := []string{}
	args := []interface{}{}
	idx := 1

	if req.Rarity != nil {
		validRarities := map[string]bool{"Thường": true, "Hiếm": true, "Sử Thi": true, "Truyền Thuyết": true}
		if !validRarities[*req.Rarity] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Phẩm chất không hợp lệ"})
			return
		}
		sets = append(sets, "rarity=$"+strconv.Itoa(idx))
		args = append(args, *req.Rarity)
		idx++
	}
	if req.MonsterRank != nil {
		sets = append(sets, "monster_rank=$"+strconv.Itoa(idx))
		args = append(args, *req.MonsterRank)
		idx++
	}
	if req.MonsterSubRank != nil {
		sets = append(sets, "monster_sub_rank=$"+strconv.Itoa(idx))
		args = append(args, *req.MonsterSubRank)
		idx++
	}

	if len(sets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu cập nhật"})
		return
	}

	args = append(args, slotID)
	query := `UPDATE mine_slots SET ` + strings.Join(sets, ", ") + ` WHERE id=$` + strconv.Itoa(idx)

	if _, err := db.DB.Exec(query, args...); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật mỏ: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.update_mine_slot", "slot=%d rarity=%v rank=%v subRank=%v", slotID, req.Rarity, req.MonsterRank, req.MonsterSubRank)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật mỏ thành công"})
}

func AdminResetMineSlot(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok {
		return
	}

	slotID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID mỏ không hợp lệ"})
		return
	}

	_, err = db.DB.Exec(`
		UPDATE mine_slots
		SET occupier_id = NULL, monster_id = NULL, occupied_at = NULL, last_claimed_at = NULL, abandoned_at = NOW()
		WHERE id = $1
	`, slotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể reset mỏ: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.reset_mine_slot", "slot=%d", slotID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã reset mỏ thành công"})
}

// -------- Admin: Gu Dictionary CRUD --------

func AdminListGuItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, name, rank, path, gu_type,
		base_mana_cost, effects, food_types, hunger_rate, max_hunger,
		COALESCE(description,''), COALESCE(target_type,'SINGLE'), COALESCE(rarity,''),
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
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		Rank        int             `json:"rank"`
		Path        json.RawMessage `json:"path"`
		GuType      string          `json:"gu_type"`
		BaseManaCost float64        `json:"base_mana_cost"`
		Effects     json.RawMessage `json:"effects"`
		FoodTypes   json.RawMessage `json:"food_types"`
		HungerRate  int             `json:"hunger_rate"`
		MaxHunger   int             `json:"max_hunger"`
		Description string          `json:"description"`
		TargetType  string          `json:"target_type"`
		Rarity      string          `json:"rarity"`
		IsImmortal  bool            `json:"is_immortal"`
		PrimaryRole string          `json:"primary_role"`
		CombatRoles json.RawMessage `json:"combat_roles"`
		CombatTags  json.RawMessage `json:"combat_tags"`
		SupportTags json.RawMessage `json:"support_tags"`
		BasePrice   int             `json:"base_price"`
		ImageURL    string          `json:"image_url"`
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
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Rank        int             `json:"rank"`
	Path        []string        `json:"path"`
	GuType      string          `json:"gu_type"`
	BaseManaCost float64        `json:"base_mana_cost"`
	Effects     json.RawMessage `json:"effects"`
	FoodTypes   []string        `json:"food_types"`
	HungerRate  int             `json:"hunger_rate"`
	MaxHunger   int             `json:"max_hunger"`
	Description string          `json:"description"`
	TargetType  string          `json:"target_type"`
	Rarity      string          `json:"rarity"`
	IsImmortal  bool            `json:"is_immortal"`
	PrimaryRole string          `json:"primary_role"`
	CombatRoles []string        `json:"combat_roles"`
	CombatTags  []string        `json:"combat_tags"`
	SupportTags []string        `json:"support_tags"`
	BasePrice   int             `json:"base_price"`
	ImageURL    string          `json:"image_url"`
}

func guValidate(req *AdminGuInput) (bool, string) {
	if req.ID == "" { return false, "ID là bắt buộc" }
	if req.Name == "" { return false, "Tên là bắt buộc" }
	if req.Rank < 1 { return false, "Rank phải >= 1" }
	validTypes := map[string]bool{"Active": true, "Buff": true, "Consumable": true, "Control": true, "Defense": true, "Healing": true, "Investigation": true, "Movement": true, "Passive": true, "Refinement": true, "Special": true, "Storage": true, "Summon": true, "Support": true, "Transformation": true, "Vital": true}
	if !validTypes[req.GuType] { return false, "gu_type không hợp lệ" }
	if req.HungerRate < 1 { return false, "hunger_rate phải >= 1" }
	return true, ""
}

func marshalArray(arr []string) json.RawMessage {
	if arr == nil { arr = []string{} }
	b, _ := json.Marshal(arr)
	return b
}

func AdminCreateGuItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	var req AdminGuInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if valid, msg := guValidate(&req); !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if req.MaxHunger <= 0 { req.MaxHunger = 100 }
	if req.TargetType == "" { req.TargetType = "SINGLE" }
	if len(req.Effects) == 0 { req.Effects = json.RawMessage(`{"active":{},"passive":{},"house":{}}`) }

	_, err := db.DB.Exec(`INSERT INTO gu_dictionary (id,name,rank,path,gu_type,base_mana_cost,effects,food_types,hunger_rate,max_hunger,description,target_type,rarity,is_immortal,primary_role,combat_roles,combat_tags,support_tags,base_price,image_url) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		req.ID, req.Name, req.Rank, marshalArray(req.Path), req.GuType, req.BaseManaCost, req.Effects, marshalArray(req.FoodTypes),
		req.HungerRate, req.MaxHunger, req.Description, req.TargetType, req.Rarity, req.IsImmortal, req.PrimaryRole,
		marshalArray(req.CombatRoles), marshalArray(req.CombatTags), marshalArray(req.SupportTags), req.BasePrice, req.ImageURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo: " + err.Error()})
		return
	}
	models.LoadGuRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.create_gu", "id=%s name=%s", req.ID, req.Name)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo cổ trùng"})
}

func AdminUpdateGuItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	guID := c.Param("id")
	var req AdminGuInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	// Use existing ID from URL if not in body
	if req.ID == "" { req.ID = guID }

	if req.Name == "" {
		// Partial update — only update provided fields
		sets := []string{}
		args := []interface{}{}
		idx := 1
		if req.Rank > 0 { sets = append(sets, "rank=$"+strconv.Itoa(idx)); args = append(args, req.Rank); idx++ }
		if req.GuType != "" { sets = append(sets, "gu_type=$"+strconv.Itoa(idx)); args = append(args, req.GuType); idx++ }
		if req.BaseManaCost > 0 || true { sets = append(sets, "base_mana_cost=$"+strconv.Itoa(idx)); args = append(args, req.BaseManaCost); idx++ }
		if req.HungerRate > 0 { sets = append(sets, "hunger_rate=$"+strconv.Itoa(idx)); args = append(args, req.HungerRate); idx++ }
		if req.MaxHunger > 0 { sets = append(sets, "max_hunger=$"+strconv.Itoa(idx)); args = append(args, req.MaxHunger); idx++ }
		if req.Path != nil { sets = append(sets, "path=$"+strconv.Itoa(idx)); args = append(args, marshalArray(req.Path)); idx++ }
		if req.FoodTypes != nil { sets = append(sets, "food_types=$"+strconv.Itoa(idx)); args = append(args, marshalArray(req.FoodTypes)); idx++ }
		if req.CombatRoles != nil { sets = append(sets, "combat_roles=$"+strconv.Itoa(idx)); args = append(args, marshalArray(req.CombatRoles)); idx++ }
		if req.CombatTags != nil { sets = append(sets, "combat_tags=$"+strconv.Itoa(idx)); args = append(args, marshalArray(req.CombatTags)); idx++ }
		if req.SupportTags != nil { sets = append(sets, "support_tags=$"+strconv.Itoa(idx)); args = append(args, marshalArray(req.SupportTags)); idx++ }
		if req.Description != "" || true { sets = append(sets, "description=$"+strconv.Itoa(idx)); args = append(args, req.Description); idx++ }
		if req.TargetType != "" { sets = append(sets, "target_type=$"+strconv.Itoa(idx)); args = append(args, req.TargetType); idx++ }
		if req.Rarity != "" || true { sets = append(sets, "rarity=$"+strconv.Itoa(idx)); args = append(args, req.Rarity); idx++ }
		if req.PrimaryRole != "" || true { sets = append(sets, "primary_role=$"+strconv.Itoa(idx)); args = append(args, req.PrimaryRole); idx++ }
		sets = append(sets, "is_immortal=$"+strconv.Itoa(idx)); args = append(args, req.IsImmortal); idx++
		if req.BasePrice > 0 || true { sets = append(sets, "base_price=$"+strconv.Itoa(idx)); args = append(args, req.BasePrice); idx++ }
		if req.ImageURL != "" { sets = append(sets, "image_url=$"+strconv.Itoa(idx)); args = append(args, req.ImageURL); idx++ }
		if len(req.Effects) > 0 { sets = append(sets, "effects=$"+strconv.Itoa(idx)); args = append(args, req.Effects); idx++ }

		if len(sets) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu"})
			return
		}
		args = append(args, guID)
		_, err := db.DB.Exec(`UPDATE gu_dictionary SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(idx), args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
			return
		}
	} else {
		if valid, msg := guValidate(&req); !valid { c.JSON(http.StatusBadRequest, gin.H{"error": msg}); return }
		if req.MaxHunger <= 0 { req.MaxHunger = 100 }
		if req.TargetType == "" { req.TargetType = "SINGLE" }
		if len(req.Effects) == 0 { req.Effects = json.RawMessage(`{"active":{},"passive":{},"house":{}}`) }
		_, err := db.DB.Exec(`UPDATE gu_dictionary SET name=$1,rank=$2,path=$3,gu_type=$4,base_mana_cost=$5,effects=$6,food_types=$7,hunger_rate=$8,max_hunger=$9,description=$10,target_type=$11,rarity=$12,is_immortal=$13,primary_role=$14,combat_roles=$15,combat_tags=$16,support_tags=$17,base_price=$18,image_url=$19 WHERE id=$20`,
			req.Name, req.Rank, marshalArray(req.Path), req.GuType, req.BaseManaCost, req.Effects, marshalArray(req.FoodTypes),
			req.HungerRate, req.MaxHunger, req.Description, req.TargetType, req.Rarity, req.IsImmortal, req.PrimaryRole,
			marshalArray(req.CombatRoles), marshalArray(req.CombatTags), marshalArray(req.SupportTags), req.BasePrice, req.ImageURL, guID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
			return
		}
	}
	models.LoadGuRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.update_gu", "id=%s", guID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật cổ trùng"})
}

func AdminDeleteGuItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	guID := c.Param("id")
	_, err := db.DB.Exec(`DELETE FROM gu_dictionary WHERE id=$1`, guID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}
	models.LoadGuRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.delete_gu", "id=%s", guID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa cổ trùng"})
}

// -------- Admin: Item Dictionary CRUD --------

func AdminListItemItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok { return }
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, name, item_type, rank, effects, COALESCE(description,''), COALESCE(base_price,0), COALESCE(image_url,'') FROM item_dictionary ORDER BY name`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()

	type ItemRow struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		ItemType    string          `json:"item_type"`
		Rank        int             `json:"rank"`
		Effects     json.RawMessage `json:"effects"`
		Description string          `json:"description"`
		BasePrice   int             `json:"base_price"`
		ImageURL    string          `json:"image_url"`
	}
	items := []ItemRow{}
	for rows.Next() {
		var r ItemRow
		if err := rows.Scan(&r.ID, &r.Name, &r.ItemType, &r.Rank, &r.Effects, &r.Description, &r.BasePrice, &r.ImageURL); err != nil { continue }
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
	BasePrice   int             `json:"base_price"`
	ImageURL    string          `json:"image_url"`
}

func AdminCreateItemItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	var req AdminItemInput
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" || req.Name == "" || req.ItemType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ (id, name, item_type là bắt buộc)"})
		return
	}
	if req.Rank < 1 { req.Rank = 1 }
	if len(req.Effects) == 0 { req.Effects = json.RawMessage(`{}`) }
	_, err := db.DB.Exec(`INSERT INTO item_dictionary(id,name,item_type,rank,effects,description,base_price,image_url) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		req.ID, req.Name, req.ItemType, req.Rank, req.Effects, req.Description, req.BasePrice, req.ImageURL)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	models.LoadItemRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.create_item", "id=%s", req.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo vật phẩm"})
}

func AdminUpdateItemItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	itemID := c.Param("id")
	var req AdminItemInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	sets := []string{}; args := []interface{}{}; idx := 1
	if req.Name != "" { sets = append(sets, "name=$"+strconv.Itoa(idx)); args = append(args, req.Name); idx++ }
	if req.ItemType != "" { sets = append(sets, "item_type=$"+strconv.Itoa(idx)); args = append(args, req.ItemType); idx++ }
	if req.Rank > 0 { sets = append(sets, "rank=$"+strconv.Itoa(idx)); args = append(args, req.Rank); idx++ }
	if req.Description != "" || true { sets = append(sets, "description=$"+strconv.Itoa(idx)); args = append(args, req.Description); idx++ }
	if req.BasePrice > 0 || true { sets = append(sets, "base_price=$"+strconv.Itoa(idx)); args = append(args, req.BasePrice); idx++ }
	if req.ImageURL != "" { sets = append(sets, "image_url=$"+strconv.Itoa(idx)); args = append(args, req.ImageURL); idx++ }
	if len(req.Effects) > 0 { sets = append(sets, "effects=$"+strconv.Itoa(idx)); args = append(args, req.Effects); idx++ }
	if len(sets) == 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu"}); return }
	args = append(args, itemID)
	_, err := db.DB.Exec(`UPDATE item_dictionary SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(idx), args...)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	models.LoadItemRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.update_item", "id=%s", itemID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật vật phẩm"})
}

func AdminDeleteItemItem(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	itemID := c.Param("id")
	_, err := db.DB.Exec(`DELETE FROM item_dictionary WHERE id=$1`, itemID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	models.LoadItemRegistry(db.DB)
	auditlog.Logf(db.DB, &adminID, "admin.delete_item", "id=%s", itemID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa vật phẩm"})
}

// -------- Admin: Recipe CRUD --------

func AdminListRecipes(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok { return }
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT r.id, r.target_gu_id, COALESCE(g.name, ''), r.required_gus, r.required_materials, r.success_rate, r.primeval_stones_cost, COALESCE(r.immortal_essence_stones_cost,0) FROM refining_recipes r LEFT JOIN gu_dictionary g ON r.target_gu_id = g.id AND r.target_rank = g.rank ORDER BY r.id`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()

	type RecipeRow struct {
		ID                        string          `json:"id"`
		TargetGuID                string          `json:"target_gu_id"`
		TargetGuName              string          `json:"target_gu_name,omitempty"`
		RequiredGus               json.RawMessage `json:"required_gus"`
		RequiredMaterials         json.RawMessage `json:"required_materials"`
		SuccessRate               float64         `json:"success_rate"`
		PrimevalStonesCost        int             `json:"primeval_stones_cost"`
		ImmortalEssenceStonesCost float64         `json:"immortal_essence_stones_cost"`
	}
	items := []RecipeRow{}
	for rows.Next() {
		var r RecipeRow
		if err := rows.Scan(&r.ID, &r.TargetGuID, &r.TargetGuName, &r.RequiredGus, &r.RequiredMaterials, &r.SuccessRate, &r.PrimevalStonesCost, &r.ImmortalEssenceStonesCost); err != nil { continue }
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
	PrimevalStonesCost        int             `json:"primeval_stones_cost"`
	ImmortalEssenceStonesCost float64         `json:"immortal_essence_stones_cost"`
}

func AdminCreateRecipe(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	var req AdminRecipeInput
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" || req.TargetGuID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	targetRank := 1
	if req.TargetRank > 0 { targetRank = req.TargetRank }
	var exists bool
	db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM gu_dictionary WHERE id=$1 AND rank=$2)`, req.TargetGuID, targetRank).Scan(&exists)
	if !exists { c.JSON(http.StatusBadRequest, gin.H{"error": "target_gu_id không tồn tại"}); return }
	if req.SuccessRate <= 0 { req.SuccessRate = 0.5 }
	if len(req.RequiredGus) == 0 { req.RequiredGus = json.RawMessage(`{}`) }
	if len(req.RequiredMaterials) == 0 { req.RequiredMaterials = json.RawMessage(`{}`) }
	_, err := db.DB.Exec(`INSERT INTO refining_recipes(id,target_gu_id,target_rank,required_gus,required_materials,success_rate,primeval_stones_cost,immortal_essence_stones_cost) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
		req.ID, req.TargetGuID, targetRank, req.RequiredGus, req.RequiredMaterials, req.SuccessRate, req.PrimevalStonesCost, req.ImmortalEssenceStonesCost)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	auditlog.Logf(db.DB, &adminID, "admin.create_recipe", "id=%s target=%s", req.ID, req.TargetGuID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã tạo cổ phương"})
}

func AdminUpdateRecipe(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	recipeID := c.Param("id")
	var req AdminRecipeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	sets := []string{}; args := []interface{}{}; idx := 1
	if req.TargetGuID != "" { sets = append(sets, "target_gu_id=$"+strconv.Itoa(idx)); args = append(args, req.TargetGuID); idx++ }
	if req.TargetRank > 0 { sets = append(sets, "target_rank=$"+strconv.Itoa(idx)); args = append(args, req.TargetRank); idx++ }
	if req.SuccessRate > 0 { sets = append(sets, "success_rate=$"+strconv.Itoa(idx)); args = append(args, req.SuccessRate); idx++ }
	if req.PrimevalStonesCost > 0 { sets = append(sets, "primeval_stones_cost=$"+strconv.Itoa(idx)); args = append(args, req.PrimevalStonesCost); idx++ }
	if req.ImmortalEssenceStonesCost > 0 { sets = append(sets, "immortal_essence_stones_cost=$"+strconv.Itoa(idx)); args = append(args, req.ImmortalEssenceStonesCost); idx++ }
	if len(req.RequiredGus) > 0 { sets = append(sets, "required_gus=$"+strconv.Itoa(idx)); args = append(args, req.RequiredGus); idx++ }
	if len(req.RequiredMaterials) > 0 { sets = append(sets, "required_materials=$"+strconv.Itoa(idx)); args = append(args, req.RequiredMaterials); idx++ }
	if len(sets) == 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "Không có dữ liệu"}); return }
	args = append(args, recipeID)
	_, err := db.DB.Exec(`UPDATE refining_recipes SET `+strings.Join(sets, ", ")+` WHERE id=$`+strconv.Itoa(idx), args...)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	auditlog.Logf(db.DB, &adminID, "admin.update_recipe", "id=%s", recipeID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật cổ phương"})
}

func AdminDeleteRecipe(c *gin.Context) {
	adminID, ok := requireAdmin(c); if !ok { return }
	recipeID := c.Param("id")
	_, err := db.DB.Exec(`DELETE FROM refining_recipes WHERE id=$1`, recipeID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	auditlog.Logf(db.DB, &adminID, "admin.delete_recipe", "id=%s", recipeID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa cổ phương"})
}

type OnlinePlayerRow struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	Rank         int        `json:"rank"`
	SubRank      int        `json:"sub_rank"`
	IsOnline     bool       `json:"is_online"`
	LastActivity *time.Time `json:"last_activity"`
}

func AdminListOnlinePlayers(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok { return }

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
