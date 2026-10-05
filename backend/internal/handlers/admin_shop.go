package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

// -------- Admin: Shop Management --------

type AdminShopItem struct {
	ID        string  `json:"id"`
	Rank      int     `json:"rank"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	IsDefault bool    `json:"is_default"`
	SortOrder int     `json:"sort_order"`
	Name      string  `json:"name,omitempty"`
	ItemType  string  `json:"item_type,omitempty"`
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ danh sách cửa hàng"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách cửa hàng"}); return }
	if err := rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách cửa hàng"}); return }

	c.JSON(http.StatusOK, items)
}

type AdminShopItemRequest struct {
	ID        string  `json:"id"`
	Rank      int     `json:"rank"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	IsDefault bool    `json:"is_default"`
	SortOrder int     `json:"sort_order"`
}

func AdminCreateShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	var req AdminShopItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if strings.TrimSpace(req.ID) == "" || len(req.ID) > 50 || req.Rank < 0 || req.Price < 0.001 || !pricing.IsValidAmount(req.Price) || req.Stock < -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Verify item exists in item_dictionary, gu_dictionary, or refining_recipes
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM item_dictionary WHERE id=$1 UNION SELECT 1 FROM gu_dictionary WHERE id=$1 UNION SELECT 1 FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1)`, req.ID).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh mặt hàng"})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong item_dictionary, gu_dictionary hoặc công thức"})
		return
	}
	// A pinned rank must exist for the target Gu (rank = 0 means auto).
	if req.Rank > 0 {
		var guRankExists bool
		if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM gu_dictionary WHERE id=$1 AND rank=$2)`, req.ID, req.Rank).Scan(&guRankExists); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh bậc cổ trùng"})
			return
		}
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

	auditlog.Logf(db.DB, &adminID, "admin.create_shop_item", "id=%s rank=%d price=%.3f stock=%d", req.ID, req.Rank, req.Price, req.Stock)
	c.JSON(http.StatusOK, gin.H{"message": "Đã thêm mặt hàng vào shop"})
}

func AdminUpdateShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	itemID := c.Param("id")
	rank, err := strconv.Atoi(c.Param("rank"))
	if err != nil || rank < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bậc cửa hàng không hợp lệ"})
		return
	}
	var req AdminShopItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if req.ID != "" && req.ID != itemID || req.Rank != 0 && req.Rank != rank || req.Price < 0.001 || !pricing.IsValidAmount(req.Price) || req.Stock < -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE caravan_shop_items SET price=$1, stock=$2, is_default=$3, sort_order=$4, updated_at=NOW() WHERE id=$5 AND rank=$6`,
		req.Price, req.Stock, req.IsDefault, req.SortOrder, itemID, rank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
		return
	}
	updated, err := result.RowsAffected(); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận mặt hàng đã cập nhật"}); return }; if updated == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mặt hàng cửa hàng"}); return }

	auditlog.Logf(db.DB, &adminID, "admin.update_shop_item", "id=%s rank=%d price=%.3f stock=%d is_default=%v", itemID, rank, req.Price, req.Stock, req.IsDefault)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật mặt hàng"})
}

func AdminDeleteShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	itemID := c.Param("id")
	rank, err := strconv.Atoi(c.Param("rank"))
	if err != nil || rank < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bậc cửa hàng không hợp lệ"})
		return
	}

	result, err := db.DB.Exec(`DELETE FROM caravan_shop_items WHERE id=$1 AND rank=$2`, itemID, rank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}
	deleted, err := result.RowsAffected(); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận mặt hàng đã xóa"}); return }; if deleted == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mặt hàng cửa hàng"}); return }

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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc đầy đủ danh sách Arena Shop"})
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách Arena Shop"}); return }

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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	if strings.TrimSpace(req.ID) == "" || len(req.ID) > 50 || req.Price <= 0 || req.Qty < 1 || req.Stock < -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Verify item exists in item_dictionary, gu_dictionary, or refining_recipes
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM item_dictionary WHERE id=$1 UNION SELECT 1 FROM gu_dictionary WHERE id=$1 UNION SELECT 1 FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1)`, req.ID).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh mặt hàng"})
		return
	}
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
	if req.ID != "" && req.ID != itemID || req.Price <= 0 || req.Qty < 1 || req.Stock < -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Giá, số lượng hoặc tồn kho không hợp lệ"})
		return
	}

	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE arena_shop_items SET price=$1, qty=$2, stock=$3, is_default=$4, sort_order=$5, updated_at=NOW() WHERE id=$6`,
		req.Price, req.Qty, req.Stock, req.IsDefault, req.SortOrder, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật: " + err.Error()})
		return
	}
	updated, err := result.RowsAffected(); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận mặt hàng Arena Shop đã cập nhật"}); return }; if updated == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mặt hàng Arena Shop"}); return }

	auditlog.Logf(db.DB, &adminID, "admin.update_arena_shop_item", "id=%s price=%d qty=%d stock=%d is_default=%v", itemID, req.Price, req.Qty, req.Stock, req.IsDefault)
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật mặt hàng Arena Shop"})
}

func AdminDeleteArenaShopItem(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	itemID := c.Param("id")

	result, err := db.DB.Exec(`DELETE FROM arena_shop_items WHERE id=$1`, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa: " + err.Error()})
		return
	}
	deleted, err := result.RowsAffected(); if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận mặt hàng Arena Shop đã xóa"}); return }; if deleted == 0 { c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy mặt hàng Arena Shop"}); return }

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
	MineType       string     `json:"mine_type"`
	OccupiedAt     *time.Time `json:"occupied_at"`
	LastClaimedAt  *time.Time `json:"last_claimed_at"`
	AbandonedAt    *time.Time `json:"abandoned_at"`
}

func AdminListMineSlots(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT m.id, m.rarity, m.occupier_id, COALESCE(p.username, ''), m.monster_id, m.monster_rank, m.monster_sub_rank, m.mine_type,
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
		if err := rows.Scan(&s.ID, &s.Rarity, &s.OccupierID, &occupierName, &s.MonsterID, &s.MonsterRank, &s.MonsterSubRank, &s.MineType,
			&s.OccupiedAt, &s.LastClaimedAt, &s.AbandonedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách mỏ"})
			return
		}
		if occupierName != "" {
			s.OccupierName = &occupierName
		}
		slots = append(slots, s)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách mỏ"})
		return
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
		if *req.MonsterRank < 1 || *req.MonsterRank > 9 { c.JSON(http.StatusBadRequest, gin.H{"error":"Cấp quái phải từ 1 đến 9"}); return }
		sets = append(sets, "monster_rank=$"+strconv.Itoa(idx))
		args = append(args, *req.MonsterRank)
		idx++
	}
	if req.MonsterSubRank != nil {
		if *req.MonsterSubRank < 1 || *req.MonsterSubRank > 10 { c.JSON(http.StatusBadRequest, gin.H{"error":"Cấp phụ quái phải từ 1 đến 10"}); return }
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

	result, err := db.DB.ExecContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật mỏ: " + err.Error()})
		return
	}
	if n, _ := result.RowsAffected(); n == 0 { c.JSON(http.StatusNotFound, gin.H{"error":"Không tìm thấy mỏ"}); return }

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

	result, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE mine_slots
		SET occupier_id = NULL, monster_id = NULL, occupied_at = NULL, last_claimed_at = NULL, abandoned_at = NOW()
		WHERE id = $1
	`, slotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể reset mỏ: " + err.Error()})
		return
	}
	if n, _ := result.RowsAffected(); n == 0 { c.JSON(http.StatusNotFound, gin.H{"error":"Không tìm thấy mỏ"}); return }

	auditlog.Logf(db.DB, &adminID, "admin.reset_mine_slot", "slot=%d", slotID)
	c.JSON(http.StatusOK, gin.H{"message": "Đã reset mỏ thành công"})
}
