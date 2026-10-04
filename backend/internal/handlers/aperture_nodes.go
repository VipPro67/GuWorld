package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type PlaceNodeRequest struct {
	NodeType string `json:"node_type" binding:"required"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Path     string `json:"path"`
	ProdKey  string `json:"prod_key"`
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

	if req.X < 0 || req.X >= 20 || req.Y < 0 || req.Y >= 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ nằm ngoài ranh giới Tiên Khiếu (0-19)."})
		return
	}

	template, valid := GetResourceNodeTemplate(req.NodeType)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại tài nguyên xây dựng không tồn tại"})
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

	// Limit maximum resource nodes count based on area_size
	var currentNodesCount int
	err = tx.QueryRow(`
		SELECT COUNT(1) FROM aperture_resource_nodes 
		WHERE aperture_id = $1 AND is_destroyed = false
	`, ap.ID).Scan(&currentNodesCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra số lượng mỏ hiện tại"})
		return
	}

	maxNodes := int(ap.AreaSize / 100000)
	if maxNodes > 200 {
		maxNodes = 200
	}
	if currentNodesCount >= maxNodes {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quy mô Tiên Khiếu không đủ rộng lớn. Số lượng điểm tài nguyên tối đa là %d mỏ (Hiện tại: %d).", maxNodes, currentNodesCount)})
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

	// 2. Check collision
	var collision bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM aperture_resource_nodes 
			WHERE aperture_id = $1 AND x = $2 AND y = $3 AND is_destroyed = false
		)
	`, ap.ID, req.X, req.Y).Scan(&collision)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra trùng tọa độ"})
		return
	}

	if collision {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ này đã có điểm tài nguyên khác xây dựng rồi."})
		return
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
	if hasItem {
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

// ArePathsConflicting returns true if two paths conflict
