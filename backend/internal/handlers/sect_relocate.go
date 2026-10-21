package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/sectmechanics"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

const SectRelocationCooldown = 72 * time.Hour

type SectRelocationCostBreakdown struct {
	BaseCost         int64 `json:"base_cost"`
	MemberCount      int   `json:"member_count"`
	MemberCost       int64 `json:"member_cost"`
	MaterialsCount   int64 `json:"materials_count"`
	MaterialsCost    int64 `json:"materials_cost"`
	MortalGuCount    int64 `json:"mortal_gu_count"`
	MortalGuCost     int64 `json:"mortal_gu_cost"`
	ImmortalGuCount  int64 `json:"immortal_gu_count"`
	ImmortalGuCost   int64 `json:"immortal_gu_cost"`
	RecipesCount     int   `json:"recipes_count"`
	RecipesCost      int64 `json:"recipes_cost"`
	TreasuryLevel    int   `json:"treasury_level"`
	TreasuryDiscount int64 `json:"treasury_discount"`
	TotalCost        int64 `json:"total_cost"`
}

// CalculateSectRelocationCost computes the full cost breakdown for relocating a sect base:
// Base: 10,000 + (members * 500) + Vault Fees:
// - Materials: sum(Rank * 10 * Qty)
// - Mortal Gu (Rank 1-5): sum(Rank * 20 * Qty)
// - Immortal Gu (Rank 6-9): sum(Rank * 500 * Qty)
// - Recipes: sum(TargetRank * 100 * Qty)
// Bảo Khố giảm 50%-90% riêng tổng phí vận chuyển tài sản trong kho.
func CalculateSectRelocationCost(q dbQueryRunner, sectID string) (int64, SectRelocationCostBreakdown, error) {
	if q == nil {
		q = db.DB
	}

	var breakdown SectRelocationCostBreakdown
	breakdown.BaseCost = 10000
	var treasuryLevel int
	if err := q.QueryRow(`SELECT COALESCE((buildings->>'treasury')::int, 1) FROM sects WHERE id = $1`, sectID).Scan(&treasuryLevel); err != nil {
		return 0, breakdown, fmt.Errorf("failed to load treasury level: %w", err)
	}
	breakdown.TreasuryLevel = treasuryLevel

	// 1. Members
	err := q.QueryRow(`SELECT COUNT(*) FROM sect_members WHERE sect_id = $1`, sectID).Scan(&breakdown.MemberCount)
	if err != nil && err != sql.ErrNoRows {
		return 0, breakdown, fmt.Errorf("failed to count members: %w", err)
	}
	breakdown.MemberCost = int64(breakdown.MemberCount) * 500

	// 2. Materials
	matRows, err := q.Query(`SELECT item_id, quantity FROM sect_vault_materials WHERE sect_id = $1 AND quantity > 0`, sectID)
	if err == nil {
		for matRows.Next() {
			var itemID string
			var qty int64
			if err := matRows.Scan(&itemID, &qty); err == nil {
				rank := 1
				if spec, ok := models.ItemRegistry[itemID]; ok && spec.Rank > 0 {
					rank = spec.Rank
				}
				breakdown.MaterialsCost += int64(rank) * 10 * qty
				breakdown.MaterialsCount += qty
			}
		}
		matRows.Close()
	}

	// 3. Gu
	guRows, err := q.Query(`SELECT rank, quantity FROM sect_vault_gu WHERE sect_id = $1 AND quantity > 0`, sectID)
	if err == nil {
		for guRows.Next() {
			var rank int
			var qty int64
			if err := guRows.Scan(&rank, &qty); err == nil {
				if rank <= 0 {
					rank = 1
				}
				if rank < 6 {
					breakdown.MortalGuCost += int64(rank) * 20 * qty
					breakdown.MortalGuCount += qty
				} else {
					breakdown.ImmortalGuCost += int64(rank) * 500 * qty
					breakdown.ImmortalGuCount += qty
				}
			}
		}
		guRows.Close()
	}

	// 4. Recipes
	recRows, err := q.Query(`SELECT recipe_id FROM sect_vault_recipes WHERE sect_id = $1`, sectID)
	if err == nil {
		for recRows.Next() {
			var recID string
			if err := recRows.Scan(&recID); err == nil {
				targetRank := 1
				if _, tr, _, parseErr := ParseRecipeID(recID); parseErr == nil && tr > 0 {
					targetRank = tr
				} else if guDef, ok := models.GetGuAny(recID); ok && guDef.Rank > 0 {
					targetRank = guDef.Rank
				}
				breakdown.RecipesCost += int64(targetRank) * 100
				breakdown.RecipesCount++
			}
		}
		recRows.Close()
	}

	var saved int64
	breakdown.MaterialsCost, saved = sectmechanics.DiscountVaultFee(breakdown.MaterialsCost, treasuryLevel)
	breakdown.TreasuryDiscount += saved
	breakdown.MortalGuCost, saved = sectmechanics.DiscountVaultFee(breakdown.MortalGuCost, treasuryLevel)
	breakdown.TreasuryDiscount += saved
	breakdown.ImmortalGuCost, saved = sectmechanics.DiscountVaultFee(breakdown.ImmortalGuCost, treasuryLevel)
	breakdown.TreasuryDiscount += saved
	breakdown.RecipesCost, saved = sectmechanics.DiscountVaultFee(breakdown.RecipesCost, treasuryLevel)
	breakdown.TreasuryDiscount += saved
	breakdown.TotalCost = breakdown.BaseCost + breakdown.MemberCost + breakdown.MaterialsCost + breakdown.MortalGuCost + breakdown.ImmortalGuCost + breakdown.RecipesCost
	return breakdown.TotalCost, breakdown, nil
}

// GetSectRelocationEstimate checks relocation eligibility and estimates the cost for the current player
func GetSectRelocationEstimate(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var player models.Player
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id, username, current_region, x, y
		FROM players WHERE id = $1
	`, playerID).Scan(&player.ID, &player.Username, &player.CurrentRegion, &player.X, &player.Y)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thông tin nhân vật"})
		return
	}

	var m SectMember
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1
	`, playerID).Scan(&m.SectID, &m.MemberRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	var sectName, sectRegion string
	var sectX, sectY int
	var sectFunds int64
	var lastRelocatedAt sql.NullTime
	var buildingsJSON []byte
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT name, region, map_x, map_y, funds, last_relocated_at, buildings
		FROM sects WHERE id = $1
	`, m.SectID).Scan(&sectName, &sectRegion, &sectX, &sectY, &sectFunds, &lastRelocatedAt, &buildingsJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin tông môn"})
		return
	}

	mainHallLvl := getBuildingLvl(buildingsJSON, "main_hall")
	radius := 5 + mainHallLvl

	totalCost, breakdown, err := CalculateSectRelocationCost(db.DB, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tính toán chi phí di dời: " + err.Error()})
		return
	}

	canRelocate := true
	var reason string

	// Check leadership
	if m.MemberRank != "CHUONG_MON" {
		canRelocate = false
		reason = "Chỉ có Chưởng Môn mới có quyền kích hoạt đại điển di dời sơn môn!"
	}

	// Check cooldown (72h)
	cooldownRemainingSeconds := int64(0)
	if lastRelocatedAt.Valid {
		elapsed := time.Since(lastRelocatedAt.Time)
		if elapsed < SectRelocationCooldown {
			remaining := SectRelocationCooldown - elapsed
			cooldownRemainingSeconds = int64(remaining.Seconds())
			if canRelocate {
				canRelocate = false
				hours := int(remaining.Hours())
				minutes := int(remaining.Minutes()) % 60
				reason = fmt.Sprintf("Sơn môn vừa di dời cách đây không lâu! Cần chờ thêm %d giờ %d phút để tiếp tục di dời.", hours, minutes)
			}
		}
	}

	// Check safe zone / hub
	var zoneType string
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT COALESCE(zone_type, 'KHU_VUC') FROM regions WHERE id = $1
	`, player.CurrentRegion).Scan(&zoneType)
	if err == nil {
		if zoneType == "HUB" || zoneType == "SAFE_ZONE" || player.CurrentRegion == "hub_nam_cuong" || player.CurrentRegion == "Lobby" {
			if canRelocate {
				canRelocate = false
				reason = "Không thể di dời sơn môn vào An Toàn Khu / Trấn (HUB/SAFE_ZONE)! Đạo hữu hãy di chuyển ra dã ngoại."
			}
		}
	}

	// Check if identical position
	if player.CurrentRegion == sectRegion && player.X == sectX && player.Y == sectY {
		if canRelocate {
			canRelocate = false
			reason = "Đạo hữu hiện đang ở ngay vị trí sơn môn hiện tại."
		}
	}

	// Check territory overlap
	hasOverlap, conflictingSect, errOverlap := CheckTerritoryOverlap(db.DB, player.CurrentRegion, player.X, player.Y, radius, m.SectID)
	if errOverlap == nil && hasOverlap {
		if canRelocate {
			canRelocate = false
			reason = fmt.Sprintf("Lãnh địa tại vị trí này (%d, %d) bị đè/trùng lấn với sơn môn \"%s\"! Vui lòng chọn vị trí khác.", player.X, player.Y, conflictingSect)
		}
	}

	// Check funds
	if sectFunds < totalCost {
		if canRelocate {
			canRelocate = false
			reason = fmt.Sprintf("Quỹ tông môn không đủ! Cần %d quỹ nhưng hiện chỉ có %d quỹ (thiếu %d).", totalCost, sectFunds, totalCost-sectFunds)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"can_relocate":               canRelocate,
		"reason":                     reason,
		"is_leader":                  m.MemberRank == "CHUONG_MON",
		"cooldown_remaining_seconds": cooldownRemainingSeconds,
		"cooldown_total_seconds":     int64(SectRelocationCooldown.Seconds()),
		"current_funds":              sectFunds,
		"funds_needed":               totalCost,
		"cost_breakdown":             breakdown,
		"current_sect_location": gin.H{
			"region": sectRegion,
			"x":      sectX,
			"y":      sectY,
		},
		"target_location": gin.H{
			"region": player.CurrentRegion,
			"x":      player.X,
			"y":      player.Y,
			"radius": radius,
		},
	})
}

// RelocateSect handles the relocation of a sect base to Chưởng Môn's current location
func RelocateSect(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khởi tạo giao dịch"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin nhân vật"})
		return
	}
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã tử vong, không thể thực hiện hành động này!"})
		return
	}

	var m SectMember
	err = tx.QueryRow(`
		SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1
	`, playerID).Scan(&m.SectID, &m.MemberRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if m.MemberRank != "CHUONG_MON" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Chưởng Môn mới có quyền kích hoạt đại điển di dời sơn môn!"})
		return
	}

	var sectName, sectRegion string
	var sectX, sectY int
	var sectFunds int64
	var lastRelocatedAt sql.NullTime
	var buildingsJSON []byte
	err = tx.QueryRow(`
		SELECT name, region, map_x, map_y, funds, last_relocated_at, buildings
		FROM sects WHERE id = $1 FOR UPDATE
	`, m.SectID).Scan(&sectName, &sectRegion, &sectX, &sectY, &sectFunds, &lastRelocatedAt, &buildingsJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tìm thấy tông môn"})
		return
	}

	// 1. Check cooldown
	if lastRelocatedAt.Valid {
		elapsed := time.Since(lastRelocatedAt.Time)
		if elapsed < SectRelocationCooldown {
			remaining := SectRelocationCooldown - elapsed
			hours := int(remaining.Hours())
			minutes := int(remaining.Minutes()) % 60
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Sơn môn vừa di dời cách đây không lâu! Cần chờ thêm %d giờ %d phút để tiếp tục di dời.", hours, minutes),
			})
			return
		}
	}

	// 2. Check position is outdoor and not HUB / SAFE_ZONE
	var zoneType string
	err = tx.QueryRow(`
		SELECT COALESCE(zone_type, 'KHU_VUC') FROM regions WHERE id = $1
	`, player.CurrentRegion).Scan(&zoneType)
	if err == nil {
		if zoneType == "HUB" || zoneType == "SAFE_ZONE" || player.CurrentRegion == "hub_nam_cuong" || player.CurrentRegion == "Lobby" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể di dời sơn môn vào An Toàn Khu / Trấn (HUB/SAFE_ZONE)! Đạo hữu hãy di chuyển ra dã ngoại."})
			return
		}
	}

	if player.CurrentRegion == sectRegion && player.X == sectX && player.Y == sectY {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu hiện đang ở ngay vị trí sơn môn hiện tại, không cần di dời!"})
		return
	}

	// 3. Check territory overlap
	mainHallLvl := getBuildingLvl(buildingsJSON, "main_hall")
	radius := 5 + mainHallLvl

	hasOverlap, conflictingSect, errOverlap := CheckTerritoryOverlap(tx, player.CurrentRegion, player.X, player.Y, radius, m.SectID)
	if errOverlap != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kiểm tra lãnh địa thất bại: " + errOverlap.Error()})
		return
	}
	if hasOverlap {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Lãnh địa tại vị trí này (%d, %d) bị đè/trùng lấn với sơn môn \"%s\"! Vui lòng chọn vị trí khác.", player.X, player.Y, conflictingSect),
		})
		return
	}

	// 4. Calculate cost
	totalCost, breakdown, err := CalculateSectRelocationCost(tx, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tính toán chi phí di dời: " + err.Error()})
		return
	}

	if sectFunds < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Quỹ tông môn không đủ để tiến hành di dời! Yêu cầu %d Quỹ nhưng hiện chỉ có %d Quỹ (thiếu %d).", totalCost, sectFunds, totalCost-sectFunds),
		})
		return
	}

	// 5. Update sect: deduct funds, set new location, update last_relocated_at
	res, err := tx.Exec(`
		UPDATE sects
		SET region = $1, map_x = $2, map_y = $3, funds = funds - $4, last_relocated_at = NOW()
		WHERE id = $5 AND funds >= $4
	`, player.CurrentRegion, player.X, player.Y, totalCost, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cập nhật vị trí sơn môn thất bại: " + err.Error()})
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quỹ tông môn không đủ để khấu trừ"})
		return
	}

	auditlog.Log(tx, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.relocate", EntityType: "sect",
		EntityID:       m.SectID,
		RequestSummary: fmt.Sprintf("Di dời sơn môn từ %s (%d, %d) đến %s (%d, %d), tiêu %d Quỹ", sectRegion, sectX, sectY, player.CurrentRegion, player.X, player.Y, totalCost),
		Metadata: map[string]interface{}{
			"sect_id":      m.SectID,
			"from_region":  sectRegion,
			"from_x":       sectX,
			"from_y":       sectY,
			"to_region":    player.CurrentRegion,
			"to_x":         player.X,
			"to_y":         player.Y,
			"cost_funds":   totalCost,
			"funds_change": -totalCost,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hoàn tất giao dịch"})
		return
	}

	_ = cache.Delete(sectsListCacheKey)

	// 6. Broadcast notification to sect chat
	chatMsg := fmt.Sprintf("[Sơn Môn] Chưởng Môn %s đã chủ trì đại điển dời căn cứ tông môn đến tọa độ (%d, %d) thuộc %s! Tiêu hao %d Quỹ Tông Môn.",
		player.Username, player.X, player.Y, player.CurrentRegion, totalCost)
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, chatMsg)

	c.JSON(http.StatusOK, gin.H{
		"message":         "Di dời sơn môn thành công!",
		"new_region":      player.CurrentRegion,
		"new_x":           player.X,
		"new_y":           player.Y,
		"cost":            totalCost,
		"remaining_funds": sectFunds - totalCost,
		"cost_breakdown":  breakdown,
	})
}
