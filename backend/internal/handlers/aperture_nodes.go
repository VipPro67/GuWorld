package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type PlaceNodeRequest struct {
	NodeType string                    `json:"node_type" binding:"required"`
	X        int                       `json:"x"`
	Y        int                       `json:"y"`
	Path     string                    `json:"path"`
	ProdKey  string                    `json:"prod_key"`
	Cells    []models.ApertureGridCell `json:"cells"`
}

type ManageApertureNodesBatchRequest struct {
	Action  string   `json:"action" binding:"required"`
	NodeIDs []string `json:"node_ids" binding:"required,min=1,max=400"`
}

func ManageApertureNodesBatch(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req ManageApertureNodesBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách node không hợp lệ"})
		return
	}
	req.Action = strings.ToLower(strings.TrimSpace(req.Action))
	if req.Action != "repair" && req.Action != "upgrade" && req.Action != "delete" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thao tác không hợp lệ"})
		return
	}
	nodeIDs := make([]uuid.UUID, 0, len(req.NodeIDs))
	seen := make(map[uuid.UUID]struct{}, len(req.NodeIDs))
	for _, rawID := range req.NodeIDs {
		id, err := uuid.Parse(rawID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mã node không hợp lệ"})
			return
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		nodeIDs = append(nodeIDs, id)
	}
	if len(nodeIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa chọn node nào"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	type batchNode struct {
		ID, ApertureID, Name, Type, Category string
		Level, HP, MaxHP                     int
		Destroyed                            bool
		ProductionAmount                     int64
		DefensePower, HPBonus                int
	}
	rows, err := tx.Query(`
		SELECT n.id, n.aperture_id, n.name, n.node_type, COALESCE(n.node_category, 'RESOURCE'),
			n.level, n.hp, n.max_hp, n.is_destroyed, COALESCE(n.production_amount, 0),
			COALESCE(n.defense_power, 0), COALESCE(n.hp_bonus, 0)
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON a.id = n.aperture_id
		WHERE n.id = ANY($1::uuid[]) AND a.player_id = $2
		ORDER BY n.id FOR UPDATE OF n
	`, pq.Array(nodeIDs), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải danh sách node"})
		return
	}
	nodes := make([]batchNode, 0, len(nodeIDs))
	for rows.Next() {
		var n batchNode
		if err := rows.Scan(&n.ID, &n.ApertureID, &n.Name, &n.Type, &n.Category, &n.Level, &n.HP, &n.MaxHP, &n.Destroyed, &n.ProductionAmount, &n.DefensePower, &n.HPBonus); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc thông tin node"})
			return
		}
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc danh sách node"})
		return
	}
	rows.Close()
	if len(nodes) != len(nodeIDs) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Một hoặc nhiều node không tồn tại hoặc không thuộc Tiên Khiếu của đạo hữu"})
		return
	}

	totalCost := 0
	if req.Action != "delete" {
		for _, n := range nodes {
			if n.Destroyed {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể sửa hoặc nâng cấp node đã bị phá hủy"})
				return
			}
			if req.Action == "repair" {
				cost := int(math.Ceil(float64(n.MaxHP-n.HP) / 50.0))
				if cost <= 0 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách không có node nào cần sửa chữa"})
					return
				}
				totalCost += cost
				continue
			}
			if n.Level >= MaxNodeLevel {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s đã đạt cấp tối đa", n.Name)})
				return
			}
			template, ok := GetResourceNodeTemplate(n.Type)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Loại node không hợp lệ: " + n.Type})
				return
			}
			cost := int(float64(template.CostStones) * float64(n.Level) * 1.5)
			if cost < 5 {
				cost = 5
			}
			totalCost += cost
		}
		var stones int
		if err := tx.QueryRow(`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1 FOR UPDATE`, playerID).Scan(&stones); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư Tiên Nguyên Thạch"})
			return
		}
		if stones < totalCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần %d Tiên Nguyên Thạch, hiện có %d", totalCost, stones)})
			return
		}
	}

	var newStones float64
	if totalCost > 0 {
		if err := tx.QueryRow(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2 RETURNING immortal_stones`, totalCost, playerID).Scan(&newStones); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên Thạch"})
			return
		}
	}
	for _, n := range nodes {
		switch req.Action {
		case "repair":
			if _, err := tx.Exec(`UPDATE aperture_resource_nodes SET hp = max_hp, updated_at = NOW() WHERE id = $1`, n.ID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi sửa chữa node"})
				return
			}
		case "upgrade":
			newLevel := n.Level + 1
			newMaxHP := n.MaxHP + 500
			newProduction := n.ProductionAmount
			newDefense, newHPBonus := n.DefensePower, n.HPBonus
			if n.Category == "RESOURCE" {
				newProduction = int64(float64(n.ProductionAmount) * 1.5)
				if newProduction == n.ProductionAmount {
					newProduction++
				}
			} else {
				if newDefense > 0 {
					newDefense = int(float64(newDefense) * 1.25)
					if newDefense == n.DefensePower {
						newDefense++
					}
				}
				if newHPBonus > 0 {
					newHPBonus = int(float64(newHPBonus) * 1.25)
					if newHPBonus == n.HPBonus {
						newHPBonus++
					}
				}
			}
			if _, err := tx.Exec(`UPDATE aperture_resource_nodes SET level=$1, max_hp=$2, hp=hp+500, production_amount=$3, defense_power=$4, hp_bonus=$5, updated_at=NOW() WHERE id=$6`, newLevel, newMaxHP, newProduction, newDefense, newHPBonus, n.ID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi nâng cấp node"})
				return
			}
		case "delete":
			if err := updateVariantHumanOnNodeChange(tx, n.ApertureID, n.Type, -50); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật dân số dị nhân"})
				return
			}
		}
	}
	if req.Action == "delete" {
		if _, err := tx.Exec(`DELETE FROM aperture_resource_nodes WHERE id = ANY($1::uuid[])`, pq.Array(nodeIDs)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi san phẳng node đã chọn"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu thao tác hàng loạt"})
		return
	}
	actionLabel := map[string]string{"repair": "Sửa chữa", "upgrade": "Nâng cấp", "delete": "San phẳng"}[req.Action]
	response := gin.H{"message": fmt.Sprintf("%s %d node thành công.", actionLabel, len(nodes)), "count": len(nodes), "cost": totalCost}
	if totalCost > 0 {
		response["immortal_stones"] = newStones
	}
	c.JSON(http.StatusOK, response)
}

func loadApertureNodeCells(tx *sql.Tx, apertureID any) (map[string][]models.ApertureGridCell, error) {
	rows, err := tx.Query(`SELECT node_id, x, y FROM aperture_resource_node_cells WHERE aperture_id = $1 ORDER BY node_id, y, x`, apertureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cells := make(map[string][]models.ApertureGridCell)
	for rows.Next() {
		var nodeID string
		var cell models.ApertureGridCell
		if err := rows.Scan(&nodeID, &cell.X, &cell.Y); err != nil {
			return nil, err
		}
		cells[nodeID] = append(cells[nodeID], cell)
	}
	return cells, rows.Err()
}

func apertureCellsAreConnected(cells []models.ApertureGridCell) bool {
	if len(cells) == 0 {
		return false
	}
	remaining := make(map[models.ApertureGridCell]bool, len(cells))
	for _, cell := range cells {
		if cell.X < 0 || cell.X >= 20 || cell.Y < 0 || cell.Y >= 20 || remaining[cell] {
			return false
		}
		remaining[cell] = true
	}
	queue := []models.ApertureGridCell{cells[0]}
	delete(remaining, cells[0])
	for len(queue) > 0 {
		cell := queue[0]
		queue = queue[1:]
		for _, next := range []models.ApertureGridCell{{X: cell.X - 1, Y: cell.Y}, {X: cell.X + 1, Y: cell.Y}, {X: cell.X, Y: cell.Y - 1}, {X: cell.X, Y: cell.Y + 1}} {
			if remaining[next] {
				delete(remaining, next)
				queue = append(queue, next)
			}
		}
	}
	return len(remaining) == 0
}

func PlaceApertureNode(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req PlaceNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	req.NodeType = strings.TrimSpace(req.NodeType)
	req.ProdKey = strings.TrimSpace(req.ProdKey)
	if len(req.Cells) == 0 {
		req.Cells = []models.ApertureGridCell{{X: req.X, Y: req.Y}}
	}

	if len(req.Cells) > 400 || !apertureCellsAreConnected(req.Cells) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Khu tài nguyên phải gồm các ô liền nhau trong lưới Tiên Khiếu."})
		return
	}
	anchorFound := false
	for _, cell := range req.Cells {
		if cell.X == req.X && cell.Y == req.Y {
			anchorFound = true
			break
		}
	}
	if !anchorFound {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ node phải nằm trong khu đã chọn."})
		return
	}

	template, valid := GetResourceNodeTemplate(req.NodeType)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại tài nguyên xây dựng không tồn tại"})
		return
	}
	if len(req.Cells) > 1 && req.NodeType != "FARMING_ZONE" && req.NodeType != "BREEDING_ZONE" && req.NodeType != "MINING_ZONE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ khu trồng trọt, chăn nuôi và khai thác mới chiếm nhiều ô."})
		return
	}

	if req.ProdKey != "" {
		if itemSpec, exists := models.ItemRegistry[req.ProdKey]; exists {
			if itemSpec.Type == "Currency" || itemSpec.Type == "Consumable" || itemSpec.ItemType == "Currency" || itemSpec.ItemType == "Consumable" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể sản xuất vật phẩm loại Currency hoặc Consumable trong mỏ tài nguyên"})
				return
			}
		}
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch player & aperture
	var ap models.ImmortalAperture
	err = tx.QueryRow(`
		SELECT id, state, area_size FROM immortal_apertures WHERE player_id = $1
	`, playerID).Scan(&ap.ID, &ap.State, &ap.AreaSize)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra Tiên Khiếu"})
		return
	}

	if ap.State == "TRIBULATION_ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Khiếu đang có tai kiếp càn quét, không thể xây dựng!"})
		return
	}

	var immortalStones int64
	err = tx.QueryRow(`
		SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&immortalStones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
		return
	}

	// Every grid cell can belong to only one active node. Legacy nodes without
	// footprint rows continue to occupy their original anchor cell.
	for _, cell := range req.Cells {
		var collision bool
		err = tx.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM aperture_resource_node_cells WHERE aperture_id = $1 AND x = $2 AND y = $3
				UNION ALL
				SELECT 1 FROM aperture_resource_nodes n
				WHERE n.aperture_id = $1 AND n.x = $2 AND n.y = $3 AND n.is_destroyed = false
				  AND NOT EXISTS (SELECT 1 FROM aperture_resource_node_cells c WHERE c.node_id = n.id)
			)
		`, ap.ID, cell.X, cell.Y).Scan(&collision)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra ô đất đang sử dụng"})
			return
		}
		if collision {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Ô (%d, %d) đã thuộc về một khu khác.", cell.X, cell.Y)})
			return
		}
	}

	// 2b. Limit unique defense structures (Barracks: max 1)
	if req.NodeType == "BARRACKS" {
		var barrackCount int
		err = tx.QueryRow(`
			SELECT COUNT(1) FROM aperture_resource_nodes
			WHERE aperture_id = $1 AND node_type = 'BARRACKS' AND is_destroyed = false
		`, ap.ID).Scan(&barrackCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra Doanh Trại"})
			return
		}
		if barrackCount >= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi Tiên Khiếu chỉ được xây dựng tối đa 1 Doanh Trại."})
			return
		}
	}

	// 3. Deduct cost or item
	nodeCategory := template.NodeCategory
	if nodeCategory == "" {
		nodeCategory = "RESOURCE"
	}
	nodeHPMax := template.HPMax
	if nodeHPMax <= 0 {
		nodeHPMax = 1000
	}

	nodeItemID := "node_" + strings.ToLower(req.NodeType)
	qty, err := InventorySvc.GetQuantity(tx, playerID.String(), nodeItemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn túi đồ"})
		return
	}
	hasItem := qty > 0

	var newStones float64
	if hasItem {
		if err := InventorySvc.RemoveItem(tx, playerID.String(), nodeItemID, "materials", 1); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tiêu hao vật phẩm mỏ"})
			return
		}
		_ = tx.QueryRow(`SELECT immortal_stones FROM players WHERE id = $1`, playerID).Scan(&newStones)
	} else {
		if immortalStones < int64(template.CostStones) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Nguyên Thạch không đủ để xây dựng mỏ này (Yêu cầu: %d, Hiện có: %d).", template.CostStones, immortalStones)})
			return
		}
		err = tx.QueryRow(`
			UPDATE players 
			SET immortal_stones = immortal_stones - $1
			WHERE id = $2
			RETURNING immortal_stones
		`, template.CostStones, playerID).Scan(&newStones)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ chi phí"})
			return
		}
	}

	// 3b. Consume required Gu items for defense structures
	if len(template.CostGu) > 0 {
		for _, requiredGuID := range template.CostGu {
			var guInstanceID string
			var guRank int
			err = tx.QueryRow(`
				SELECT id, gu_rank FROM player_gu 
				WHERE player_id = $1 AND gu_id = $2 AND gu_rank >= $3 AND state = 'ACTIVE' AND is_vital = false AND equipped = false
				LIMIT 1
			`, playerID, requiredGuID, template.CostGuRank).Scan(&guInstanceID, &guRank)
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần sở hữu Cổ trùng %s (rank %d+, không phải Bản Mệnh/đang trang bị) để xây dựng %s", requiredGuID, template.CostGuRank, template.Name)})
				return
			} else if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn cổ trùng"})
				return
			}
			// Consume the Gu (destroy it)
			_, err = tx.Exec(`DELETE FROM player_gu WHERE id = $1 AND is_vital = false`, guInstanceID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tiêu hao cổ trùng"})
				return
			}
			// Audit log for the consumed Gu
			consumedGuName := requiredGuID
			if spec, ok := models.GetGu(requiredGuID, guRank); ok {
				consumedGuName = spec.Name
			}
			auditlog.Log(tx, auditlog.Entry{
				PlayerID:       &playerID,
				Action:         "aperture.consume_gu",
				EntityType:     "gu_instance",
				EntityID:       guInstanceID,
				RequestSummary: fmt.Sprintf("Tiêu hao Cổ trùng %s (rank %d) để xây dựng %s", consumedGuName, guRank, template.Name),
				Metadata: map[string]interface{}{
					"gu_id":     requiredGuID,
					"gu_name":   consumedGuName,
					"gu_rank":   guRank,
					"node_name": template.Name,
					"node_type": template.NodeType,
				},
				IPAddress: c.ClientIP(),
				Success:   true,
			})
		}
	}

	nodePath := template.Path
	if req.Path != "" {
		nodePath = req.Path
	}

	nodeProdKey := template.ProdKey
	if req.ProdKey != "" {
		nodeProdKey = req.ProdKey
	}

	nodeName := template.Name
	if nodeProdKey != "" {
		if itemSpec, exists := models.ItemRegistry[nodeProdKey]; exists {
			nodeName = template.Name + " (" + itemSpec.Name + ")"
		}
	}

	// 3c. Gieo giống: khu tài nguyên tiêu hao một lượng chính vật phẩm
	// định sản xuất (prod_key) làm hạt giống trước khi node hoạt động.
	seedsConsumed := int64(0)
	seedName := ""
	if nodeCategory == "RESOURCE" {
		if nodeProdKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Phải chọn vật phẩm sản xuất cho khu tài nguyên."})
			return
		}
		if template.SeedAmount > 0 {
			seedQty := int64(template.SeedAmount)
			seedName = nodeProdKey
			if itemSpec, exists := models.ItemRegistry[nodeProdKey]; exists {
				seedName = itemSpec.Name
			}
			have, err := InventorySvc.GetQuantity(tx, playerID.String(), nodeProdKey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn túi đồ giống"})
				return
			}
			if have < seedQty {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Thiếu giống để gieo trồng: cần %d %s (đang có %d).", seedQty, seedName, have)})
				return
			}
			if err := InventorySvc.RemoveItem(tx, playerID.String(), nodeProdKey, "materials", seedQty); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tiêu hao giống gieo trồng"})
				return
			}
			seedsConsumed = seedQty
		}
	}

	// 4. Insert node
	var node models.ApertureResourceNode
	var createdAt time.Time
	err = tx.QueryRow(`
		INSERT INTO aperture_resource_nodes (aperture_id, node_type, name, path, level, hp, max_hp, x, y,
			production_type, production_amount, production_interval_hours, last_collected_at,
			node_category, defense_power, attack_range, hp_bonus, damage_reduction, blocks_movement,
			required_gu_id, required_gu_rank, zone_type, food_path, item_tier, production_key)
		VALUES ($1, $2, $3, $4, 1, $5, $5, $6, $7, $8, $9, $10, NOW(),
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id, created_at
	`, ap.ID, template.NodeType, nodeName, nodePath, nodeHPMax, req.X, req.Y,
		template.ProdType, template.ProdAmount, template.ProdIntervalHours,
		nodeCategory, template.DefensePower, template.AttackRange, template.HPBonus,
		template.DamageReduction, template.BlocksMovement,
		strings.Join(template.CostGu, ","), template.CostGuRank,
		template.ZoneType, template.FoodPath, template.ItemTier, nodeProdKey).Scan(&node.ID, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu điểm tài nguyên: " + err.Error()})
		return
	}
	for _, cell := range req.Cells {
		if _, err = tx.Exec(`INSERT INTO aperture_resource_node_cells (node_id, aperture_id, x, y) VALUES ($1, $2, $3, $4)`, node.ID, ap.ID, cell.X, cell.Y); err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				c.JSON(http.StatusConflict, gin.H{"error": "Một ô trong khu vừa được sử dụng. Hãy chọn vùng khác."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu diện tích khu tài nguyên"})
			return
		}
	}

	err = updateVariantHumanOnNodeChange(tx, ap.ID, template.NodeType, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật số lượng dị nhân: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit"})
		return
	}

	node.ApertureID = ap.ID
	node.NodeType = template.NodeType
	node.Name = template.Name
	node.Path = template.Path
	node.Level = 1
	node.HP = nodeHPMax
	node.MaxHP = nodeHPMax
	node.X = req.X
	node.Y = req.Y
	node.ProductionType = template.ProdType
	node.ProductionAmount = template.ProdAmount
	node.ProductionIntervalHours = template.ProdIntervalHours
	node.NodeCategory = nodeCategory
	node.DefensePower = template.DefensePower
	node.AttackRange = template.AttackRange
	node.HPBonus = template.HPBonus
	node.DamageReduction = template.DamageReduction
	node.BlocksMovement = template.BlocksMovement
	node.RequiredGuID = strings.Join(template.CostGu, ",")
	node.RequiredGuRank = template.CostGuRank
	node.ZoneType = template.ZoneType
	node.FoodPath = template.FoodPath
	node.ItemTier = template.ItemTier
	node.CreatedAt = createdAt
	node.UpdatedAt = createdAt

	var costParts []string
	if hasItem {
		costParts = append(costParts, fmt.Sprintf("1x %s", nodeItemID))
	} else {
		costParts = append(costParts, fmt.Sprintf("%d Tiên Nguyên Thạch", template.CostStones))
	}
	if len(template.CostGu) > 0 {
		costParts = append(costParts, fmt.Sprintf("Cổ trùng: %s", strings.Join(template.CostGu, ", ")))
	}
	costDesc := strings.Join(costParts, ", ")

	buildMsg := fmt.Sprintf("Xây dựng thành công %s! Tiêu hao: %s.", template.Name, costDesc)
	if seedsConsumed > 0 {
		buildMsg += fmt.Sprintf(" Gieo giống: %dx %s.", seedsConsumed, seedName)
	}
	notify.Send(db.DB, playerID, "aperture_build", "Xây Dựng Tiên Khiếu", buildMsg)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "aperture.build_node",
		EntityType:     "immortal_aperture_node",
		EntityID:       node.ID,
		RequestSummary: fmt.Sprintf("Xây dựng kiến trúc Tiên Khiếu: %s | Tiêu hao: %s", template.Name, costDesc),
		Metadata: map[string]interface{}{
			"node_id":   node.ID,
			"node_type": req.NodeType,
			"node_name": template.Name,
			"has_item":  hasItem,
			"cost_desc": costDesc,
			"cost_gus":  template.CostGu,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	resp := gin.H{
		"node":            node,
		"message":         buildMsg,
		"immortal_stones": newStones,
	}
	if hasItem || seedsConsumed > 0 {
		resp["materials_inventory"] = loadMaterialsInventory(playerID.String())
	}
	c.JSON(http.StatusOK, resp)
}

// UninstallApertureNode uninstalls a node, deleting it and refunding 1x node_<type> to materials_inventory
func UninstallApertureNode(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	nodeIDStr := c.Param("id")
	nodeID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã tài nguyên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch node details
	var n models.ApertureResourceNode
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.is_destroyed
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.IsDestroyed)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}

	if n.IsDestroyed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ tài nguyên đã bị phá hủy, không thể tháo dỡ thu hồi!"})
		return
	}

	// 2. Add refund item to materials_inventory
	nodeItemID := "node_" + strings.ToLower(n.NodeType)
	err = InventorySvc.AddItem(tx, playerID.String(), nodeItemID, "materials", 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hoàn trả vật phẩm khi thu hồi mỏ"})
		return
	}

	// 3. Delete node from database completely
	_, err = tx.Exec(`DELETE FROM aperture_resource_nodes WHERE id = $1`, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xóa điểm tài nguyên trong cơ sở dữ liệu"})
		return
	}

	err = updateVariantHumanOnNodeChange(tx, n.ApertureID, n.NodeType, -50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật số lượng dị nhân: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to uninstall node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             fmt.Sprintf("Tháo dỡ thành công %s, đã thu hồi lại mô hình mỏ vào túi đồ.", n.Name),
		"node_id":             nodeIDStr,
		"materials_inventory": loadMaterialsInventory(playerID.String()),
	})
}

// DeleteApertureNode completely deletes a node (even if destroyed) without refunding the item
func DeleteApertureNode(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	nodeIDStr := c.Param("id")
	nodeID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã tài nguyên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch node details to verify ownership
	var n models.ApertureResourceNode
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.name, n.node_type
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.Name, &n.NodeType)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}

	// 2. Delete node from database
	_, err = tx.Exec(`DELETE FROM aperture_resource_nodes WHERE id = $1`, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xóa điểm tài nguyên trong cơ sở dữ liệu"})
		return
	}

	err = updateVariantHumanOnNodeChange(tx, n.ApertureID, n.NodeType, -50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật số lượng dị nhân: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("San phẳng thành công %s. Vị trí đã được giải phóng.", n.Name),
		"node_id": nodeIDStr,
	})
}
