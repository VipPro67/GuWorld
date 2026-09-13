package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// -------- Admin: Shop Management --------

type AdminShopItem struct {
	ID        string `json:"id"`
	Rank      int    `json:"rank"`
	Price     int64  `json:"price"`
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
		SELECT c.id, c.rank, c.price, c.stock, c.is_default, c.sort_order,
			COALESCE(i.name, g.name, 'Công thức ' || rg.name, c.id) AS name,
			CASE WHEN r.target_gu_id IS NOT NULL THEN 'Recipe' ELSE COALESCE(i.item_type, 'Gu') END AS item_type
		FROM caravan_shop_items c
		LEFT JOIN item_dictionary i ON c.id = i.id
		LEFT JOIN gu_dictionary g ON c.id = g.id
		LEFT JOIN refining_recipes r ON c.id = ('recipe_' || r.target_gu_id || '_' || r.target_rank::text || '_' || r.variant::text)
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		ORDER BY c.sort_order, c.id, c.rank
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách shop: " + err.Error()})
		return
	}
	defer rows.Close()

	items := []AdminShopItem{}
	for rows.Next() {
		var item AdminShopItem
		if err := rows.Scan(&item.ID, &item.Rank, &item.Price, &item.Stock, &item.IsDefault, &item.SortOrder, &item.Name, &item.ItemType); err != nil {
			continue
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

type AdminShopItemRequest struct {
	ID        string `json:"id"`
	Rank      int    `json:"rank"`
	Price     int64  `json:"price"`
	Stock     int    `json:"stock"`
	IsDefault bool   `json:"is_default"`
	SortOrder int    `json:"sort_order"`
}

func AdminCreateShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req AdminShopItemRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" || req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Verify item exists in item_dictionary, gu_dictionary, or refining_recipes
	var exists bool
	_ = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM item_dictionary WHERE id=$1 UNION SELECT 1 FROM gu_dictionary WHERE id=$1 UNION SELECT 1 FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1)`, req.ID).Scan(&exists)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong item_dictionary, gu_dictionary hoặc công thức"})
		return
	}
	// A pinned rank must exist for the target Gu (rank = 0 means auto).
	if req.Rank > 0 {
		var guRankExists bool
		_ = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM gu_dictionary WHERE id=$1 AND rank=$2)`, req.ID, req.Rank).Scan(&guRankExists)
		if !guRankExists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng không có rank này trong gu_dictionary"})
			return
		}
	}

	_, err := db.DB.Exec(`INSERT INTO caravan_shop_items (id, rank, price, stock, is_default, sort_order) VALUES ($1,$2,$3,$4,$5,$6)`,
		req.ID, req.Rank, req.Price, req.Stock, req.IsDefault, req.SortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.create_shop_item", "id=%s rank=%d price=%d stock=%d", req.ID, req.Rank, req.Price, req.Stock)
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm mặt hàng vào shop"})
}

func AdminUpdateShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	itemID := c.Param("id")
	rank := 0
	if raw := c.Param("rank"); raw != "" {
		fmt.Sscanf(raw, "%d", &rank)
	}
	var req AdminShopItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	_, err := db.DB.Exec(`UPDATE caravan_shop_items SET price=$1, stock=$2, is_default=$3, sort_order=$4, updated_at=NOW() WHERE id=$5 AND rank=$6`,
		req.Price, req.Stock, req.IsDefault, req.SortOrder, itemID, rank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.update_shop_item", "id=%s rank=%d price=%d stock=%d is_default=%v", itemID, rank, req.Price, req.Stock, req.IsDefault)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật mặt hàng"})
}

func AdminDeleteShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	itemID := c.Param("id")
	rank := 0
	if raw := c.Param("rank"); raw != "" {
		fmt.Sscanf(raw, "%d", &rank)
	}

	_, err := db.DB.Exec(`DELETE FROM caravan_shop_items WHERE id=$1 AND rank=$2`, itemID, rank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}

	auditlog.Logf(db.DB, &adminID, "admin.delete_shop_item", "id=%s rank=%d", itemID, rank)
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa mặt hàng khỏi shop"})
}

// -------- Admin: Arena Shop Management --------

type AdminArenaShopItem struct {
	ID        string `json:"id"`
	Price     int64  `json:"price"`
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
		LEFT JOIN refining_recipes r ON a.id = ('recipe_' || r.target_gu_id || '_' || r.target_rank::text || '_' || r.variant::text)
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
	Price     int64  `json:"price"`
	Qty       int    `json:"qty"`
	Stock     int    `json:"stock"`
	IsDefault bool   `json:"is_default"`
	SortOrder int    `json:"sort_order"`
}

func AdminCreateArenaShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
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
	_ = db.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM item_dictionary WHERE id=$1 UNION SELECT 1 FROM gu_dictionary WHERE id=$1 UNION SELECT 1 FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1)`, req.ID).Scan(&exists)
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
	adminID, ok := requireAdmin(c)
	if !ok {
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
	adminID, ok := requireAdmin(c)
	if !ok {
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
	ID             int        `json:"id"`
	Rarity         string     `json:"rarity"`
	OccupierID     *string    `json:"occupier_id"`
	OccupierName   *string    `json:"occupier_name"`
	MonsterID      *string    `json:"monster_id"`
	MonsterRank    int        `json:"monster_rank"`
	MonsterSubRank int        `json:"monster_sub_rank"`
	OccupiedAt     *time.Time `json:"occupied_at"`
	LastClaimedAt  *time.Time `json:"last_claimed_at"`
	AbandonedAt    *time.Time `json:"abandoned_at"`
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
	adminID, ok := requireAdmin(c)
	if !ok {
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
		validRarities := map[string]bool{"Siêu Cấp": true, "Thường": true, "Hiếm": true, "Sử Thi": true, "Truyền Thuyết": true}
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
	adminID, ok := requireAdmin(c)
	if !ok {
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
