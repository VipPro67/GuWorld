package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type SetTimeRateRequest struct {
	TimeRate float64 `json:"time_rate" binding:"required"`
}

func SetApertureTimeRate(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SetTimeRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	if req.TimeRate < 1.0 || req.TimeRate > 100.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tốc độ dòng thời gian chỉ được điều chỉnh từ 1.0x đến 100.0x"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi giao dịch", err))
		return
	}
	defer tx.Rollback()

	// Project internal_years_passed forward using the OLD rate before changing it.
	// This prevents the frontend from miscalculating the arrival time when
	// updated_at is set to NOW() but internal_years_passed is stale.
	now := time.Now()
	res, err := tx.Exec(`
		UPDATE immortal_apertures ia
		SET internal_years_passed = ia.internal_years_passed + GREATEST(0, EXTRACT(EPOCH FROM (NOW() - ia.updated_at))) * ia.time_rate * (CASE WHEN p.race = 'cuong_thi' THEN 1.0 ELSE (p.time_speed_multiplier + COALESCE((SELECT SUM((gd.effects->'passive'->>'time_speed_multiplier')::double precision) FROM player_gu pg JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank WHERE pg.player_id = p.id AND pg.equipped = true AND pg.state = 'ACTIVE'), 0)) END) / 86400.0,
			time_rate = $1, updated_at = $2
		FROM players p
		WHERE ia.player_id = p.id AND ia.player_id = $3
	`, req.TimeRate, now, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật dòng thời gian", err))
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi ghi dữ liệu", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Điều chỉnh dòng thời gian Tiên Khiếu thành công: %.1fx", req.TimeRate)})
}

type SetLaborRequest struct {
	Race           string `json:"race" binding:"required"`
	LaborIntensity string `json:"labor_intensity" binding:"required"`
}

func SetApertureLabor(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SetLaborRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	intensity := strings.ToUpper(req.LaborIntensity)
	if intensity != "LOW" && intensity != "NORMAL" && intensity != "HIGH" && intensity != "EXPLOIT" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cường độ lao động không hợp lệ (LOW, NORMAL, HIGH, EXPLOIT)"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	var apID string
	err = tx.QueryRow(`SELECT id FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	}

	req.Race = normalizeVariantEnum(req.Race)
	_, err = tx.Exec(`
		UPDATE aperture_variant_humans
		SET labor_intensity = $1, updated_at = NOW()
		WHERE aperture_id = $2 AND race = $3
	`, intensity, apID, req.Race)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật cường độ lao động", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi lưu cường độ lao động", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Cập nhật cường độ lao động của %s thành %s thành công!", req.Race, intensity)})
}

func RepairApertureNode(c *gin.Context) {
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
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	// Fetch node
	var n models.ApertureResourceNode
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.name, n.hp, n.max_hp, n.is_destroyed
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2 AND n.is_destroyed = false
		FOR UPDATE OF n
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.Name, &n.HP, &n.MaxHP, &n.IsDestroyed)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi truy vấn điểm tài nguyên", err))
		return
	}

	cost := int(math.Ceil(float64(n.MaxHP-n.HP) / 50.0))
	if cost <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ tài nguyên đang ở trạng thái nguyên vẹn tốt nhất."})
		return
	}

	// Check player stones
	var stones int
	err = tx.QueryRow(`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1`, playerID).Scan(&stones)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi lấy số dư tài khoản", err))
		return
	}

	if stones < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Nguyên Thạch không đủ để sửa chữa mỏ này (Yêu cầu: %d, Hiện có: %d).", cost, stones)})
		return
	}

	// Deduct stones
	var newStones float64
	err = tx.QueryRow(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2 RETURNING immortal_stones`, cost, playerID).Scan(&newStones)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi khấu trừ Tiên Nguyên Thạch", err))
		return
	}

	// Repair node
	res, err := tx.Exec(`
		UPDATE aperture_resource_nodes
		SET hp = max_hp, updated_at = NOW()
		WHERE id = $1 AND is_destroyed = false
	`, nodeID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật trạng thái sửa chữa", err))
		return
	}
	if repaired, rowsErr := res.RowsAffected(); rowsErr != nil || repaired != 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "Mỏ đã bị phá hủy trong lúc sửa chữa; giao dịch đã được hoàn lại"})
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi lưu trạng thái sửa chữa", err))
		return
	}

	repairMsg := fmt.Sprintf("Sửa chữa thành công %s! Tiêu tốn %d Tiên Nguyên Thạch.", n.Name, cost)
	notify.SendBestEffort(db.DB, playerID, "aperture_repair", "Sửa Chữa Tiên Khiếu", repairMsg, c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"message":         repairMsg,
		"immortal_stones": newStones,
	})
}

const MaxNodeLevel = 5

// UpgradeApertureNode upgrades a resource/defense node to the next level
func UpgradeApertureNode(c *gin.Context) {
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
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	// Fetch node
	var n models.ApertureResourceNode
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.name, n.node_type, n.level, n.max_hp, n.is_destroyed,
			COALESCE(n.node_category, 'RESOURCE'), COALESCE(n.defense_power, 0), COALESCE(n.attack_range, 0),
			COALESCE(n.hp_bonus, 0), COALESCE(n.damage_reduction, 0), COALESCE(n.production_amount, 0)
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.Name, &n.NodeType, &n.Level, &n.MaxHP, &n.IsDestroyed,
		&n.NodeCategory, &n.DefensePower, &n.AttackRange,
		&n.HPBonus, &n.DamageReduction, &n.ProductionAmount)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi truy vấn điểm tài nguyên", err))
		return
	}

	if n.IsDestroyed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể nâng cấp công trình đã bị phá hủy!"})
		return
	}

	if n.Level >= MaxNodeLevel {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đã đạt cấp tối đa (%d). Không thể nâng cấp thêm.", MaxNodeLevel)})
		return
	}

	// Calculate upgrade cost: base cost * level * 1.5
	template, ok := GetResourceNodeTemplate(n.NodeType)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại node không hợp lệ"})
		return
	}
	upgradeCost := int(float64(template.CostStones) * float64(n.Level) * 1.5)
	if upgradeCost < 5 {
		upgradeCost = 5
	}

	// Check player stones
	var stones int
	err = tx.QueryRow(`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1`, playerID).Scan(&stones)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi lấy số dư tài khoản", err))
		return
	}

	if stones < upgradeCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Nguyên Thạch không đủ để nâng cấp (Yêu cầu: %d, Hiện có: %d).", upgradeCost, stones)})
		return
	}

	// Deduct stones
	var newStones float64
	err = tx.QueryRow(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2 RETURNING immortal_stones`, upgradeCost, playerID).Scan(&newStones)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi khấu trừ Tiên Nguyên Thạch", err))
		return
	}

	// Calculate new stats
	newLevel := n.Level + 1
	newMaxHP := n.MaxHP + 500
	hpGain := newMaxHP - n.MaxHP // HP increased by this amount

	newProductionAmount := n.ProductionAmount
	newDefensePower := n.DefensePower
	newHPBonus := n.HPBonus

	// Resource nodes gain +50% yield per level through the harvest formula
	// (gamefx.ApertureYieldLevelStep), so production_amount is left untouched to
	// avoid applying the level bonus twice. Defense nodes still scale in place.
	if n.NodeCategory != "RESOURCE" {
		if n.DefensePower > 0 {
			newDefensePower = int(float64(n.DefensePower) * 1.25)
			if newDefensePower == n.DefensePower {
				newDefensePower = n.DefensePower + 1
			}
		}
		if n.HPBonus > 0 {
			newHPBonus = int(float64(n.HPBonus) * 1.25)
			if newHPBonus == n.HPBonus {
				newHPBonus = n.HPBonus + 1
			}
		}
	}

	// Apply upgrade
	_, err = tx.Exec(`
		UPDATE aperture_resource_nodes
		SET level = $1, max_hp = $2, hp = hp + $3,
			production_amount = $4, defense_power = $5, hp_bonus = $6,
			updated_at = NOW()
		WHERE id = $7
	`, newLevel, newMaxHP, hpGain, newProductionAmount, newDefensePower, newHPBonus, nodeID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi nâng cấp: "+err.Error(), err))
		return
	}

	_ = tx.Commit()

	upgradeMsg := fmt.Sprintf("Nâng cấp %s lên cấp %d thành công! Tiêu tốn %d Tiên Nguyên Thạch.", n.Name, newLevel, upgradeCost)
	notify.SendBestEffort(db.DB, playerID, "aperture_upgrade", "Nâng Cấp Tiên Khiếu", upgradeMsg, c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"message":         upgradeMsg,
		"new_level":       newLevel,
		"new_max_hp":      newMaxHP,
		"cost":            upgradeCost,
		"immortal_stones": newStones,
	})
}

func GetApertureVariantHumans(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT v.id, v.aperture_id, v.race, v.population, v.happiness, v.oppression, v.rebellion_rate, v.labor_intensity,
		       v.prosperity, v.gu_master_count, v.gu_immortal_count, v.food_stored, COALESCE(v.starving, false), COALESCE(v.auto_essence_relief, false),
		       v.created_at, v.updated_at
		FROM aperture_variant_humans v
		JOIN immortal_apertures a ON v.aperture_id = a.id
		WHERE a.player_id = $1
	`, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi lấy thông tin dị nhân", err))
		return
	}
	defer rows.Close()

	list := []models.ApertureVariantHuman{}
	for rows.Next() {
		var v models.ApertureVariantHuman
		err = rows.Scan(&v.ID, &v.ApertureID, &v.Race, &v.Population, &v.Happiness, &v.Oppression, &v.RebellionRate, &v.LaborIntensity,
			&v.Prosperity, &v.GuMasterCount, &v.GuImmortalCount, &v.FoodStored, &v.Starving, &v.AutoEssenceRelief,
			&v.CreatedAt, &v.UpdatedAt)
		if err == nil {
			list = append(list, v)
		}
	}

	c.JSON(http.StatusOK, list)
}

var raceRecruitCostPerPerson = map[string]float64{
	"HAIRY_MAN":   2,
	"ROCKMAN":     2,
	"FEATHERMAN":  2,
	"MINIMAN":     2,
	"MUSHROOMMAN": 2,
	"MERMAN":      4,
	"SNOWMAN":     4,
	"INKMAN":      5,
	"DRAGONMAN":   8,
}

type RecruitVariantRequest struct {
	Race  string `json:"race" binding:"required"`
	Count int    `json:"count" binding:"required,min=10,max=10000"`
}

// RecruitVariantHumans settles recruited/purchased variant humans into an existing settlement.
func RecruitVariantHumans(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req RecruitVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng chiêu mộ phải từ 10 đến 10,000"})
		return
	}
	req.Race = normalizeVariantEnum(req.Race)
	costPer, ok := raceRecruitCostPerPerson[req.Race]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chủng tộc dị nhân không hợp lệ"})
		return
	}
	totalCost := costPer * float64(req.Count)

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	var apertureID string
	var capacity, currentPop int
	var stones float64
	err = tx.QueryRow(`
		SELECT a.id, s.population_capacity, COALESCE(v.population, 0), p.immortal_stones
		FROM immortal_apertures a
		JOIN players p ON p.id = a.player_id
		JOIN aperture_variant_settlements s ON s.aperture_id = a.id AND s.race = $2
		LEFT JOIN aperture_variant_humans v ON v.aperture_id = a.id AND v.race = $2
		WHERE a.player_id = $1
		FOR UPDATE OF p
	`, playerID, req.Race).Scan(&apertureID, &capacity, &currentPop, &stones)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần xây Thành Trì cho chủng tộc này trước khi chiêu mộ dân"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	if currentPop+req.Count > capacity {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vượt quá sức chứa Thành Trì (%d/%d)", currentPop+req.Count, capacity)})
		return
	}
	if stones < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần %.0f Tiên Nguyên Thạch để chiêu mộ %d %s", totalCost, req.Count, req.Race)})
		return
	}

	if _, err := tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, totalCost, playerID); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi khấu trừ Tiên Nguyên Thạch", err))
		return
	}
	_, err = tx.Exec(`
		INSERT INTO aperture_variant_humans (aperture_id, race, population, happiness, oppression, labor_intensity, prosperity, food_stored)
		VALUES ($1, $2, $3, 70.0, 0.0, 'NORMAL', 50.0, 0)
		ON CONFLICT (aperture_id, race)
		DO UPDATE SET population = aperture_variant_humans.population + $3, updated_at = NOW()
	`, apertureID, req.Race, req.Count)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật dân số", err))
		return
	}
	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Đã chiêu mộ %d %s vào Thành Trì (tốn %.0f Tiên Nguyên Thạch)", req.Count, req.Race, totalCost),
	})
}

type FeedVariantRequest struct {
	Race     string `json:"race" binding:"required"`
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity" binding:"required,min=1,max=1000"`
}

// FeedVariantHumans converts food items from player_inventory into settlement food_stored.
func FeedVariantHumans(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req FeedVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu nạp lương thực không hợp lệ"})
		return
	}
	req.Race = normalizeVariantEnum(req.Race)
	itemID := strings.TrimSpace(req.ItemID)
	if itemID == "" {
		itemID = "food_essence_rank1"
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	var invQty int
	err = tx.QueryRow(`
		SELECT quantity FROM player_inventory
		WHERE player_id = $1 AND item_id = $2 FOR UPDATE
	`, playerID, itemID).Scan(&invQty)
	if err == sql.ErrNoRows || invQty < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ vật phẩm lương thực (%s)", itemID)})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	if invQty == req.Quantity {
		_, err = tx.Exec(`DELETE FROM player_inventory WHERE player_id = $1 AND item_id = $2`, playerID, itemID)
	} else {
		_, err = tx.Exec(`UPDATE player_inventory SET quantity = quantity - $1 WHERE player_id = $2 AND item_id = $3`, req.Quantity, playerID, itemID)
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi trừ vật phẩm túi đồ", err))
		return
	}

	foodGain := float64(req.Quantity) * 500.0
	var newFood float64
	err = tx.QueryRow(`
		UPDATE aperture_variant_humans v
		SET food_stored = LEAST(
			COALESCE((SELECT s.food_storage FROM aperture_variant_settlements s WHERE s.aperture_id = v.aperture_id AND s.race = v.race), 5000)::double precision,
			v.food_stored + $3
		),
		starving = false,
		updated_at = NOW()
		FROM immortal_apertures a
		WHERE v.aperture_id = a.id AND a.player_id = $1 AND v.race = $2
		RETURNING v.food_stored
	`, playerID, req.Race, foodGain).Scan(&newFood)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy bộ tộc trong Tiên Khiếu"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật kho lương thực", err))
		return
	}
	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     fmt.Sprintf("Đã nạp %.0f lương thực cho %s", foodGain, req.Race),
		"food_stored": newFood,
	})
}

type ToggleReliefRequest struct {
	Race    string `json:"race" binding:"required"`
	Enabled bool   `json:"enabled"`
}

// ToggleEssenceRelief toggles auto_essence_relief for a variant race.
func ToggleEssenceRelief(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req ToggleReliefRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	req.Race = normalizeVariantEnum(req.Race)
	res, err := db.DB.Exec(`
		UPDATE aperture_variant_humans v
		SET auto_essence_relief = $3, updated_at = NOW()
		FROM immortal_apertures a
		WHERE v.aperture_id = a.id AND a.player_id = $1 AND v.race = $2
	`, playerID, req.Race, req.Enabled)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy bộ tộc trong Tiên Khiếu"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"auto_essence_relief": req.Enabled})
}

// ListVariantEvents returns unresolved and recent variant human events for the player's aperture.
func ListVariantEvents(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	rows, err := db.DB.Query(`
		SELECT e.id, e.aperture_id, e.event_type, COALESCE(e.race, ''), e.severity, COALESCE(e.description, ''),
		       COALESCE(e.effects::text, '{}'), e.resolved, e.created_at, e.resolved_at
		FROM aperture_variant_events e
		JOIN immortal_apertures a ON a.id = e.aperture_id
		WHERE a.player_id = $1
		ORDER BY e.resolved ASC, e.created_at DESC
		LIMIT 30
	`, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer rows.Close()

	list := []models.ApertureVariantEvent{}
	for rows.Next() {
		var ev models.ApertureVariantEvent
		if rows.Scan(&ev.ID, &ev.ApertureID, &ev.EventType, &ev.Race, &ev.Severity, &ev.Description, &ev.Effects, &ev.Resolved, &ev.CreatedAt, &ev.ResolvedAt) == nil {
			list = append(list, ev)
		}
	}
	c.JSON(http.StatusOK, gin.H{"events": list})
}

type ResolveVariantEventRequest struct {
	Mode string `json:"mode" binding:"required,oneof=PACIFY SUPPRESS"`
}

// ResolveVariantEvent resolves an active STRIKE/SABOTAGE/REBELLION event via PACIFY or SUPPRESS.
func ResolveVariantEvent(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	eventID := c.Param("event_id")
	var req ResolveVariantEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chế độ xử lý phải là PACIFY hoặc SUPPRESS"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	var apertureID, race, eventType string
	var resolved bool
	err = tx.QueryRow(`
		SELECT e.aperture_id, COALESCE(e.race, ''), e.event_type, e.resolved
		FROM aperture_variant_events e
		JOIN immortal_apertures a ON a.id = e.aperture_id
		WHERE e.id = $1 AND a.player_id = $2
		FOR UPDATE OF e
	`, eventID, playerID).Scan(&apertureID, &race, &eventType, &resolved)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy sự kiện"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	if resolved {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sự kiện đã được xử lý"})
		return
	}

	if req.Mode == "PACIFY" {
		const stoneCost = 300.0
		var stones float64
		if err := tx.QueryRow(`SELECT immortal_stones FROM players WHERE id = $1 FOR UPDATE`, playerID).Scan(&stones); err != nil {
			httperr.Respond(c, httperr.Internal("Database error", err))
			return
		}
		if stones < stoneCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("An dân cần %.0f Tiên Nguyên Thạch", stoneCost)})
			return
		}
		_, _ = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, stoneCost, playerID)
		_, _ = tx.Exec(`
			UPDATE aperture_variant_humans
			SET labor_intensity = CASE WHEN labor_intensity IN ('HIGH', 'EXPLOIT') THEN 'NORMAL' ELSE labor_intensity END,
			    happiness = LEAST(100, happiness + 15),
			    oppression = GREATEST(0, oppression - 20),
			    rebellion_rate = GREATEST(0, rebellion_rate - 35),
			    updated_at = NOW()
			WHERE aperture_id = $1 AND race = $2
		`, apertureID, race)
	} else {
		const essenceCost = 20.0
		var essence float64
		if err := tx.QueryRow(`SELECT COALESCE(immortal_essence, 0) FROM immortal_apertures WHERE id = $1 FOR UPDATE`, apertureID).Scan(&essence); err != nil {
			httperr.Respond(c, httperr.Internal("Database error", err))
			return
		}
		if essence < essenceCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Trấn áp cần %.0f Tiên Nguyên", essenceCost)})
			return
		}
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = immortal_essence - $1 WHERE id = $2`, essenceCost, apertureID)
		_, _ = tx.Exec(`
			UPDATE aperture_variant_humans
			SET population = GREATEST(0, FLOOR(population * 0.95)::int),
			    happiness = GREATEST(0, happiness - 8),
			    oppression = LEAST(100, oppression + 15),
			    rebellion_rate = GREATEST(0, rebellion_rate - 40),
			    updated_at = NOW()
			WHERE aperture_id = $1 AND race = $2
		`, apertureID, race)
	}

	_, err = tx.Exec(`UPDATE aperture_variant_events SET resolved = true, resolved_at = NOW() WHERE id = $1`, eventID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Đã xử lý sự kiện %s bằng phương án %s", eventType, req.Mode)})
}
