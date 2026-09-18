package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Standard Teleport Templates
var teleportTemplates = map[string]models.TeleportTemplate{
	models.TeleportTier1: {
		Tier:               models.TeleportTier1,
		Name:               "Tiểu Hình Truyền Tống Trận (Vi Trận)",
		Description:        "Trận pháp sơ cấp, dịch chuyển trong phạm vi ngắn (cự ly tối đa 1.000 ô trong cùng khu vực).",
		MinRank:            2,
		MaxRange:           1000,
		IsCrossRegion:      false,
		MaxDurability:      50,
		CostPrimevalStones: 2000,
		CostImmortalStones: 0,
		RequiredGu:         []string{},
		RequiredItems:      map[string]int{"void_shattering_crystal": 20},
	},
	models.TeleportTier2: {
		Tier:               models.TeleportTier2,
		Name:               "Đại Vực Truyền Tống Trận",
		Description:        "Trận pháp trung cấp, dịch chuyển toàn bộ trong cùng một Đại Vực (cự ly 10.000 ô).",
		MinRank:            4,
		MaxRange:           10000,
		IsCrossRegion:      false,
		MaxDurability:      100,
		CostPrimevalStones: 10000,
		CostImmortalStones: 0,
		RequiredGu:         []string{},
		RequiredItems:      map[string]int{"boundless_space_domain_stone": 10, "void_shattering_crystal": 20},
	},
	models.TeleportTier3: {
		Tier:               models.TeleportTier3,
		Name:               "Ngũ Vực Tiên Cấp Truyền Tống Trận",
		Description:        "Tiên cấp trận pháp kỳ quan, cho phép dịch chuyển tự do xuyên qua các Đại Vực trong Ngũ Vực.",
		MinRank:            6,
		MaxRange:           999999,
		IsCrossRegion:      true,
		MaxDurability:      200,
		CostPrimevalStones: 50000,
		CostImmortalStones: 20,
		RequiredGu:         []string{},
		RequiredItems:      map[string]int{"void_shattering_crystal": 50, "boundless_space_domain_stone": 20, "void_rocks": 5},
	},
}

// Helper to query player's sect ID
func getPlayerSectID(tx *sql.Tx, playerID uuid.UUID) *string {
	var sID string
	var err error
	if tx != nil {
		err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&sID)
	} else {
		err = db.DB.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&sID)
	}
	if err == nil && sID != "" {
		return &sID
	}
	return nil
}

// GetTeleportTemplates returns all available teleport formation templates
func GetTeleportTemplates(c *gin.Context) {
	templates := []models.TeleportTemplate{
		teleportTemplates[models.TeleportTier1],
		teleportTemplates[models.TeleportTier2],
		teleportTemplates[models.TeleportTier3],
	}
	c.JSON(http.StatusOK, templates)
}

// GetMyTeleportArrays returns all teleport arrays created by the player or their sect
func GetMyTeleportArrays(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	sectIDPtr := getPlayerSectID(nil, authID)

	query := `
		SELECT t.id, t.name, t.owner_id, p.username, t.sect_id, s.name,
		       t.region_id, r.name, t.x, t.y, t.tier, t.max_range, t.is_cross_region,
		       t.direction_type, t.access_policy, t.fee_enabled, t.fee_currency,
		       t.fee_amount, t.vault_balance, t.current_durability, t.max_durability,
		       t.is_active, t.created_at, t.updated_at
		FROM teleport_arrays t
		JOIN players p ON p.id = t.owner_id
		LEFT JOIN sects s ON s.id = t.sect_id
		JOIN regions r ON r.id = t.region_id
		WHERE t.owner_id = $1
	`
	args := []interface{}{authID}

	if sectIDPtr != nil && *sectIDPtr != "" {
		query += " OR (t.sect_id = $2 AND t.access_policy = 'SECT')"
		args = append(args, *sectIDPtr)
	}
	query += " ORDER BY t.created_at DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		logger.S.Errorf("GetMyTeleportArrays query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	arrays := []models.TeleportArray{}
	for rows.Next() {
		var a models.TeleportArray
		var sectID, sectName sql.NullString
		if err := rows.Scan(
			&a.ID, &a.Name, &a.OwnerID, &a.OwnerName, &sectID, &sectName,
			&a.RegionID, &a.RegionName, &a.X, &a.Y, &a.Tier, &a.MaxRange, &a.IsCrossRegion,
			&a.DirectionType, &a.AccessPolicy, &a.FeeEnabled, &a.FeeCurrency,
			&a.FeeAmount, &a.VaultBalance, &a.CurrentDurability, &a.MaxDurability,
			&a.IsActive, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			logger.S.Errorf("GetMyTeleportArrays scan failed: %v", err)
			continue
		}
		if sectID.Valid {
			a.SectID = &sectID.String
		}
		if sectName.Valid {
			a.SectName = &sectName.String
		}
		arrays = append(arrays, a)
	}

	c.JSON(http.StatusOK, arrays)
}

// BuildTeleportArray creates a new teleportation array at player's current location
func BuildTeleportArray(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.BuildTeleportArrayReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	tpl, ok := teleportTemplates[req.Tier]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phẩm cấp Trận Pháp không tồn tại"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = tpl.Name
	}
	if len(req.Name) > 50 {
		req.Name = req.Name[:50]
	}

	if req.DirectionType == "" {
		req.DirectionType = models.TeleportDirectionTwoWay
	}
	if req.AccessPolicy == "" {
		req.AccessPolicy = models.TeleportAccessPublic
	}
	if req.FeeCurrency == "" {
		req.FeeCurrency = "primeval_stones"
	}
	if req.FeeAmount < 0 {
		req.FeeAmount = 0
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi giao dịch cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin nhân vật"})
		return
	}

	// 1. Check player Rank requirement
	if player.Rank < tpl.MinRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần đạt tu vi tối thiểu Chuyển %d để thiết lập trận pháp này", tpl.MinRank)})
		return
	}

	// 2. Check cell validity: Cannot place on existing teleport array
	var existingCount int
	err = tx.QueryRow(`SELECT COUNT(*) FROM teleport_arrays WHERE region_id = $1 AND x = $2 AND y = $3`, player.CurrentRegion, player.X, player.Y).Scan(&existingCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra vị trí trận pháp"})
		return
	}
	if existingCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ này đã có một Truyền Tống Trận khác được thiết lập"})
		return
	}

	// 3. Check Primeval Stones and Immortal Stones cost
	if player.PrimevalStones < tpl.CostPrimevalStones {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch! Cần %.0f Nguyên Thạch", tpl.CostPrimevalStones)})
		return
	}
	if player.ImmortalStones < tpl.CostImmortalStones {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch! Cần %.0f Tiên Nguyên Thạch", tpl.CostImmortalStones)})
		return
	}

	// 4. Check & consume required items if any
	if InventorySvc != nil {
		for itemID, reqQty := range tpl.RequiredItems {
			qty, cat, _ := InventorySvc.GetQuantityInAnyCategory(tx, player.ID.String(), itemID, "materials", "consumables", "other")
			if qty < int64(reqQty) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ vật liệu xây trận! Cần %d x %s (hiện có: %d)", reqQty, itemID, qty)})
				return
			}
			if err := InventorySvc.RemoveItem(tx, player.ID.String(), itemID, cat, int64(reqQty)); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tiêu hao vật liệu xây trận"})
				return
			}
		}
	}

	// Deduct currencies
	player.PrimevalStones -= tpl.CostPrimevalStones
	player.ImmortalStones -= tpl.CostImmortalStones

	sectIDPtr := getPlayerSectID(tx, player.ID)

	newID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO teleport_arrays (
			id, name, owner_id, sect_id, region_id, x, y, tier, max_range,
			is_cross_region, direction_type, access_policy, fee_enabled, fee_currency,
			fee_amount, vault_balance, current_durability, max_durability, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, 0, $16, $16, TRUE
		)
	`, newID, req.Name, player.ID, sectIDPtr, player.CurrentRegion, player.X, player.Y,
		tpl.Tier, tpl.MaxRange, tpl.IsCrossRegion, req.DirectionType, req.AccessPolicy,
		req.FeeEnabled, req.FeeCurrency, req.FeeAmount, tpl.MaxDurability)
	if err != nil {
		logger.S.Errorf("BuildTeleportArray insert error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu Trận Pháp vào cơ sở dữ liệu"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, immortal_stones = $2 WHERE id = $3`,
		player.PrimevalStones, player.ImmortalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật tài sản nhân vật"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận giao dịch"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("Bố trí thành công [%s] tại tọa độ (%d, %d)!", req.Name, player.X, player.Y),
		"array_id": newID.String(),
		"player":   player,
	})
}

// GetTeleportDirectory lists available destination nodes from current position
func GetTeleportDirectory(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	player, err := loadPlayerForStatus(db.DB, authID, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player data"})
		return
	}

	playerSectID := getPlayerSectID(nil, authID)

	// 1. Find the teleport array at current player position
	var source models.TeleportArray
	var sectID, sectName sql.NullString
	err = db.DB.QueryRow(`
		SELECT t.id, t.name, t.owner_id, p.username, t.sect_id, s.name,
		       t.region_id, r.name, t.x, t.y, t.tier, t.max_range, t.is_cross_region,
		       t.direction_type, t.access_policy, t.fee_enabled, t.fee_currency,
		       t.fee_amount, t.vault_balance, t.current_durability, t.max_durability,
		       t.is_active, t.created_at, t.updated_at
		FROM teleport_arrays t
		JOIN players p ON p.id = t.owner_id
		LEFT JOIN sects s ON s.id = t.sect_id
		JOIN regions r ON r.id = t.region_id
		WHERE t.region_id = $1 AND t.x = $2 AND t.y = $3
	`, player.CurrentRegion, player.X, player.Y).Scan(
		&source.ID, &source.Name, &source.OwnerID, &source.OwnerName, &sectID, &sectName,
		&source.RegionID, &source.RegionName, &source.X, &source.Y, &source.Tier, &source.MaxRange, &source.IsCrossRegion,
		&source.DirectionType, &source.AccessPolicy, &source.FeeEnabled, &source.FeeCurrency,
		&source.FeeAmount, &source.VaultBalance, &source.CurrentDurability, &source.MaxDurability,
		&source.IsActive, &source.CreatedAt, &source.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{
				"is_standing_on_array": false,
				"source_array":         nil,
				"destinations":         []models.TeleportDirectoryItem{},
			})
			return
		}
		logger.S.Errorf("GetTeleportDirectory error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if sectID.Valid {
		source.SectID = &sectID.String
	}
	if sectName.Valid {
		source.SectName = &sectName.String
	}

	// 2. Query all candidate destination arrays
	rows, err := db.DB.Query(`
		SELECT t.id, t.name, t.owner_id, p.username, t.sect_id, s.name,
		       t.region_id, r.name, t.x, t.y, t.tier, t.max_range, t.is_cross_region,
		       t.direction_type, t.access_policy, t.fee_enabled, t.fee_currency,
		       t.fee_amount, t.vault_balance, t.current_durability, t.max_durability,
		       t.is_active, t.created_at, t.updated_at
		FROM teleport_arrays t
		JOIN players p ON p.id = t.owner_id
		LEFT JOIN sects s ON s.id = t.sect_id
		JOIN regions r ON r.id = t.region_id
		WHERE t.id != $1
		ORDER BY t.created_at DESC
	`, source.ID)
	if err != nil {
		logger.S.Errorf("GetTeleportDirectory candidates query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	destinations := []models.TeleportDirectoryItem{}
	for rows.Next() {
		var dest models.TeleportArray
		var dSectID, dSectName sql.NullString
		if err := rows.Scan(
			&dest.ID, &dest.Name, &dest.OwnerID, &dest.OwnerName, &dSectID, &dSectName,
			&dest.RegionID, &dest.RegionName, &dest.X, &dest.Y, &dest.Tier, &dest.MaxRange, &dest.IsCrossRegion,
			&dest.DirectionType, &dest.AccessPolicy, &dest.FeeEnabled, &dest.FeeCurrency,
			&dest.FeeAmount, &dest.VaultBalance, &dest.CurrentDurability, &dest.MaxDurability,
			&dest.IsActive, &dest.CreatedAt, &dest.UpdatedAt,
		); err != nil {
			continue
		}
		if dSectID.Valid {
			dest.SectID = &dSectID.String
		}
		if dSectName.Valid {
			dest.SectName = &dSectName.String
		}

		item := evaluateTeleportLink(&player, playerSectID, &source, &dest)
		destinations = append(destinations, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"is_standing_on_array": true,
		"source_array":         source,
		"destinations":         destinations,
	})
}

// evaluateTeleportLink validates connectivity, range, access policy and fee between 2 arrays
func evaluateTeleportLink(player *models.Player, playerSectID *string, source *models.TeleportArray, dest *models.TeleportArray) models.TeleportDirectoryItem {
	item := models.TeleportDirectoryItem{
		Array:             *dest,
		CanUse:            true,
		FeeCurrency:       dest.FeeCurrency,
		IsCrossRegionJump: source.RegionID != dest.RegionID,
	}

	// 1. Check Source status
	if !source.IsActive || source.CurrentDurability <= 0 {
		item.CanUse = false
		item.Reason = "Trận pháp hiện tại đã cạn kiệt năng lượng/bị vô hiệu hóa"
		return item
	}
	if source.DirectionType == models.TeleportDirectionOneWayReceive {
		item.CanUse = false
		item.Reason = "Trận pháp hiện tại là Cổng Nhận 1 chiều, không thể truyền tống đi"
		return item
	}

	// 2. Check Destination status & direction
	if !dest.IsActive || dest.CurrentDurability <= 0 {
		item.CanUse = false
		item.Reason = "Trận pháp đích đã cạn kiệt năng lượng"
		return item
	}
	if dest.DirectionType == models.TeleportDirectionOneWaySend {
		item.CanUse = false
		item.Reason = "Trận pháp đích chỉ là Cổng Gửi, không thể tiếp nhận truyền tống đến"
		return item
	}

	// 3. Access Policy check
	isOwner := dest.OwnerID == player.ID.String()
	isSectMember := false
	if dest.SectID != nil && playerSectID != nil && *dest.SectID == *playerSectID {
		isSectMember = true
	}

	switch dest.AccessPolicy {
	case models.TeleportAccessPersonal:
		if !isOwner {
			item.CanUse = false
			item.Reason = "Trận pháp thuộc sở hữu riêng của đạo hữu khác"
			return item
		}
	case models.TeleportAccessSect:
		if !isOwner && !isSectMember {
			item.CanUse = false
			item.Reason = "Chỉ thành viên cùng Tông Môn mới có quyền sử dụng"
			return item
		}
	}

	// 4. Cross-Region & Range validation
	if source.RegionID != dest.RegionID {
		if !source.IsCrossRegion {
			item.CanUse = false
			item.Reason = "Trận pháp hiện tại không đủ phẩm cấp để truyền tống xuyên Đại Vực"
			return item
		}
		item.Distance = -1 // Cross-region distance
	} else {
		dx := math.Abs(float64(source.X - dest.X))
		dy := math.Abs(float64(source.Y - dest.Y))
		distance := int(math.Max(dx, dy))
		item.Distance = distance

		if distance > source.MaxRange {
			item.CanUse = false
			item.Reason = fmt.Sprintf("Vượt quá tầm truyền tống tối đa (%d > %d ô)", distance, source.MaxRange)
			return item
		}
	}

	// 5. Fee Calculation
	if dest.FeeEnabled && dest.FeeAmount > 0 && !isOwner && !isSectMember {
		item.FeeToPay = dest.FeeAmount
	} else {
		item.FeeToPay = 0
	}

	return item
}

// TravelTeleport executes the teleportation jump to the selected destination
func TravelTeleport(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.TravelTeleportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	targetUUID, err := uuid.Parse(req.TargetArrayID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã trận pháp đích không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi giao dịch cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin nhân vật"})
		return
	}

	playerSectID := getPlayerSectID(tx, authID)

	// 1. Load Source Array
	var source models.TeleportArray
	var sSectID sql.NullString
	err = tx.QueryRow(`
		SELECT id, name, owner_id, sect_id, region_id, x, y, tier, max_range,
		       is_cross_region, direction_type, access_policy, fee_enabled, fee_currency,
		       fee_amount, vault_balance, current_durability, max_durability, is_active
		FROM teleport_arrays
		WHERE region_id = $1 AND x = $2 AND y = $3
		FOR UPDATE
	`, player.CurrentRegion, player.X, player.Y).Scan(
		&source.ID, &source.Name, &source.OwnerID, &sSectID, &source.RegionID, &source.X, &source.Y,
		&source.Tier, &source.MaxRange, &source.IsCrossRegion, &source.DirectionType, &source.AccessPolicy,
		&source.FeeEnabled, &source.FeeCurrency, &source.FeeAmount, &source.VaultBalance,
		&source.CurrentDurability, &source.MaxDurability, &source.IsActive,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu cần đứng trực tiếp trên một Truyền Tống Trận để kích hoạt!"})
		return
	}
	if sSectID.Valid {
		source.SectID = &sSectID.String
	}

	// 2. Load Destination Array
	var dest models.TeleportArray
	var dSectID sql.NullString
	err = tx.QueryRow(`
		SELECT id, name, owner_id, sect_id, region_id, x, y, tier, max_range,
		       is_cross_region, direction_type, access_policy, fee_enabled, fee_currency,
		       fee_amount, vault_balance, current_durability, max_durability, is_active
		FROM teleport_arrays
		WHERE id = $1
		FOR UPDATE
	`, targetUUID).Scan(
		&dest.ID, &dest.Name, &dest.OwnerID, &dSectID, &dest.RegionID, &dest.X, &dest.Y,
		&dest.Tier, &dest.MaxRange, &dest.IsCrossRegion, &dest.DirectionType, &dest.AccessPolicy,
		&dest.FeeEnabled, &dest.FeeCurrency, &dest.FeeAmount, &dest.VaultBalance,
		&dest.CurrentDurability, &dest.MaxDurability, &dest.IsActive,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trận pháp đích không tồn tại hoặc đã bị phá hủy"})
		return
	}
	if dSectID.Valid {
		dest.SectID = &dSectID.String
	}

	// 3. Evaluate Link Validity
	evaluation := evaluateTeleportLink(&player, playerSectID, &source, &dest)
	if !evaluation.CanUse {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể truyền tống: " + evaluation.Reason})
		return
	}

	// 4. Charge Fee
	feePaid := evaluation.FeeToPay
	if feePaid > 0 {
		if evaluation.FeeCurrency == "immortal_stones" {
			if player.ImmortalStones < float64(feePaid) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch để trả phí! Cần %d Tiên Nguyên Thạch", feePaid)})
				return
			}
			player.ImmortalStones -= float64(feePaid)
		} else {
			if player.PrimevalStones < float64(feePaid) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch để trả phí! Cần %d Nguyên Thạch", feePaid)})
				return
			}
			player.PrimevalStones -= float64(feePaid)
		}

		// 95% goes to Array Vault, 5% is destroyed as heaven/earth toll
		vaultShare := int64(float64(feePaid) * 0.95)
		_, err = tx.Exec(`UPDATE teleport_arrays SET vault_balance = vault_balance + $1 WHERE id = $2`, vaultShare, dest.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi chuyển phí vào Két Trận Pháp"})
			return
		}
	}

	// 5. Deduct Durability
	source.CurrentDurability--
	sourceActive := source.CurrentDurability > 0
	_, err = tx.Exec(`UPDATE teleport_arrays SET current_durability = $1, is_active = $2 WHERE id = $3`,
		source.CurrentDurability, sourceActive, source.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật độ bền trận pháp"})
		return
	}

	// 6. Record Log
	_, _ = tx.Exec(`
		INSERT INTO teleport_array_logs (
			array_id, user_id, from_region, from_x, from_y, to_region, to_x, to_y, fee_paid, fee_currency
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`, dest.ID, player.ID, source.RegionID, source.X, source.Y, dest.RegionID, dest.X, dest.Y, feePaid, evaluation.FeeCurrency)

	// 7. Update Player Position & Check Regional Wall Debuff
	oldX := player.X
	oldY := player.Y
	oldRegion := player.CurrentRegion

	player.CurrentRegion = dest.RegionID
	player.X = dest.X
	player.Y = dest.Y

	logMsg := fmt.Sprintf("🌀 Không gian vặn xoắn! Đạo hữu vừa truyền tống từ [%s] đến [%s] (%d, %d).", source.Name, dest.Name, dest.X, dest.Y)
	if oldRegion != dest.RegionID {
		if debuffMsg := CheckAndApplyRegionalWallDebuff(&player, oldX, oldY, dest.X, dest.Y); debuffMsg != "" {
			logMsg += " " + debuffMsg
		}
	}

	_, err = tx.Exec(`
		UPDATE players
		SET current_region = $1, x = $2, y = $3, primeval_stones = $4, immortal_stones = $5, statuses = $6
		WHERE id = $7
	`, player.CurrentRegion, player.X, player.Y, player.PrimevalStones, player.ImmortalStones, player.Statuses, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu vị trí mới của nhân vật"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận truyền tống"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": logMsg,
		"player":  player,
	})
}

// ManageTeleportArray updates settings (fees, access policy, name)
func ManageTeleportArray(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.ManageTeleportArrayReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	arrayUUID, err := uuid.Parse(req.ArrayID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã trận pháp không hợp lệ"})
		return
	}

	var ownerID string
	var currentName, currentDirection, currentPolicy, currentCurrency string
	var currentFeeEnabled bool
	var currentFeeAmount int64

	err = db.DB.QueryRow(`
		SELECT owner_id, name, direction_type, access_policy, fee_enabled, fee_currency, fee_amount
		FROM teleport_arrays WHERE id = $1
	`, arrayUUID).Scan(&ownerID, &currentName, &currentDirection, &currentPolicy, &currentFeeEnabled, &currentCurrency, &currentFeeAmount)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trận pháp không tồn tại"})
		return
	}

	if ownerID != authID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ chủ sở hữu mới có quyền quản lý trận pháp này"})
		return
	}

	if req.Name != nil && strings.TrimSpace(*req.Name) != "" {
		currentName = strings.TrimSpace(*req.Name)
	}
	if req.DirectionType != nil && *req.DirectionType != "" {
		currentDirection = *req.DirectionType
	}
	if req.AccessPolicy != nil && *req.AccessPolicy != "" {
		currentPolicy = *req.AccessPolicy
	}
	if req.FeeEnabled != nil {
		currentFeeEnabled = *req.FeeEnabled
	}
	if req.FeeCurrency != nil && *req.FeeCurrency != "" {
		currentCurrency = *req.FeeCurrency
	}
	if req.FeeAmount != nil && *req.FeeAmount >= 0 {
		currentFeeAmount = *req.FeeAmount
	}

	_, err = db.DB.Exec(`
		UPDATE teleport_arrays
		SET name = $1, direction_type = $2, access_policy = $3, fee_enabled = $4,
		    fee_currency = $5, fee_amount = $6, updated_at = NOW()
		WHERE id = $7
	`, currentName, currentDirection, currentPolicy, currentFeeEnabled, currentCurrency, currentFeeAmount, arrayUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật cấu hình trận pháp"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật cấu hình trận pháp thành công!"})
}

// RepairTeleportArray restores durability by consuming Primeval Stones
func RepairTeleportArray(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.RepairTeleportArrayReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	arrayUUID, err := uuid.Parse(req.ArrayID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã trận pháp không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi giao dịch cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin nhân vật"})
		return
	}

	var ownerID, tier string
	var curDur, maxDur int
	err = tx.QueryRow(`
		SELECT owner_id, tier, current_durability, max_durability
		FROM teleport_arrays WHERE id = $1 FOR UPDATE
	`, arrayUUID).Scan(&ownerID, &tier, &curDur, &maxDur)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trận pháp không tồn tại"})
		return
	}

	if curDur >= maxDur {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trận pháp đã đạt độ bền tối đa"})
		return
	}

	missing := maxDur - curDur
	toRepair := req.Amount
	if toRepair <= 0 || toRepair > missing {
		toRepair = missing
	}

	costPerPoint := 50.0
	if tier == models.TeleportTier3 {
		costPerPoint = 200.0
	}
	totalCost := float64(toRepair) * costPerPoint

	if player.PrimevalStones < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch sửa chữa! Cần %.0f Nguyên Thạch", totalCost)})
		return
	}

	player.PrimevalStones -= totalCost
	curDur += toRepair

	_, err = tx.Exec(`
		UPDATE teleport_arrays
		SET current_durability = $1, is_active = TRUE, updated_at = NOW()
		WHERE id = $2
	`, curDur, arrayUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật độ bền"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1 WHERE id = $2`, player.PrimevalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi trừ nguyên thạch"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận giao dịch"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            fmt.Sprintf("Sửa chữa thành công +%d độ bền! Tiêu tốn %.0f Nguyên Thạch.", toRepair, totalCost),
		"current_durability": curDur,
		"player":             player,
	})
}

// WithdrawTeleportVault withdraws accumulated usage fees into owner's pocket
func WithdrawTeleportVault(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.WithdrawTeleportVaultReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	arrayUUID, err := uuid.Parse(req.ArrayID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã trận pháp không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi giao dịch cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin nhân vật"})
		return
	}

	var ownerID, currency string
	var vaultBalance int64
	err = tx.QueryRow(`
		SELECT owner_id, fee_currency, vault_balance
		FROM teleport_arrays WHERE id = $1 FOR UPDATE
	`, arrayUUID).Scan(&ownerID, &currency, &vaultBalance)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trận pháp không tồn tại"})
		return
	}

	if ownerID != authID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ chủ sở hữu mới có quyền rút doanh thu Két Trận Pháp"})
		return
	}

	if vaultBalance <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Két sắt Trận Pháp hiện không có số dư"})
		return
	}

	withdrawAmt := req.Amount
	if withdrawAmt <= 0 || withdrawAmt > vaultBalance {
		withdrawAmt = vaultBalance
	}

	if currency == "immortal_stones" {
		player.ImmortalStones += float64(withdrawAmt)
	} else {
		player.PrimevalStones += float64(withdrawAmt)
	}

	_, err = tx.Exec(`UPDATE teleport_arrays SET vault_balance = vault_balance - $1 WHERE id = $2`, withdrawAmt, arrayUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi trừ số dư Két Sắt"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, immortal_stones = $2 WHERE id = $3`,
		player.PrimevalStones, player.ImmortalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng tiền cho đạo hữu"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận giao dịch"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       fmt.Sprintf("Rút thành công %d %s từ Két Trận Pháp về túi đồ!", withdrawAmt, currency),
		"vault_balance": vaultBalance - withdrawAmt,
		"player":        player,
	})
}

// DismantleTeleportArray dismantles the formation and refunds 50% materials according to durability
func DismantleTeleportArray(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.DismantleTeleportArrayReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	arrayUUID, err := uuid.Parse(req.ArrayID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã trận pháp không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi giao dịch cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin nhân vật"})
		return
	}

	var ownerID, tier string
	var curDur, maxDur int
	var vaultBalance int64
	var currency string

	err = tx.QueryRow(`
		SELECT owner_id, tier, current_durability, max_durability, vault_balance, fee_currency
		FROM teleport_arrays WHERE id = $1 FOR UPDATE
	`, arrayUUID).Scan(&ownerID, &tier, &curDur, &maxDur, &vaultBalance, &currency)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Trận pháp không tồn tại"})
		return
	}

	if ownerID != authID.String() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ chủ sở hữu mới có quyền thu hồi trận pháp"})
		return
	}

	tpl := teleportTemplates[tier]

	// 50% refund modified by remaining durability ratio
	durRatio := float64(curDur) / float64(maxDur)
	if durRatio < 0.1 {
		durRatio = 0.1
	}

	refundPrimeval := (tpl.CostPrimevalStones * 0.5) * durRatio
	refundImmortal := (tpl.CostImmortalStones * 0.5) * durRatio

	player.PrimevalStones += refundPrimeval
	player.ImmortalStones += refundImmortal

	// Auto withdraw remaining vault balance
	if vaultBalance > 0 {
		if currency == "immortal_stones" {
			player.ImmortalStones += float64(vaultBalance)
		} else {
			player.PrimevalStones += float64(vaultBalance)
		}
	}

	_, err = tx.Exec(`DELETE FROM teleport_arrays WHERE id = $1`, arrayUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xóa Trận Pháp"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, immortal_stones = $2 WHERE id = $3`,
		player.PrimevalStones, player.ImmortalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật tài sản nhân vật"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận giao dịch"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Thu hồi trận pháp thành công! Hoàn trả %.0f Nguyên Thạch, %.0f Tiên Nguyên và số dư Két Sắt.", refundPrimeval, refundImmortal),
		"player":  player,
	})
}
