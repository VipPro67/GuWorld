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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi giao dịch"})
		return
	}
	defer tx.Rollback()

	// Project internal_years_passed forward using the OLD rate before changing it.
	// This prevents the frontend from miscalculating the arrival time when
	// updated_at is set to NOW() but internal_years_passed is stale.
	now := time.Now()
	res, err := tx.Exec(`
		UPDATE immortal_apertures ia
		SET internal_years_passed = ia.internal_years_passed + GREATEST(0, EXTRACT(EPOCH FROM (NOW() - ia.updated_at))) * ia.time_rate * (CASE WHEN p.race = 'cuong_thi' THEN 1.0 ELSE (p.time_speed_multiplier + COALESCE((SELECT SUM((gd.effects->'passive'->>'time_speed_multiplier')::double precision) FROM player_gu pg JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank WHERE pg.player_id = p.id AND pg.equipped = true AND pg.state = 'ACTIVE'), 0)) END) / 36000.0,
			time_rate = $1, updated_at = $2
		FROM players p
		WHERE ia.player_id = p.id AND ia.player_id = $3
	`, req.TimeRate, now, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật dòng thời gian"})
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi ghi dữ liệu"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var apID string
	err = tx.QueryRow(`SELECT id FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	}

	_, err = tx.Exec(`
		UPDATE aperture_variant_humans
		SET labor_intensity = $1, updated_at = NOW()
		WHERE aperture_id = $2 AND race = $3
	`, intensity, apID, req.Race)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật cường độ lao động"})
		return
	}

	_ = tx.Commit()

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Fetch node
	var n models.ApertureResourceNode
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.name, n.hp, n.max_hp, n.is_destroyed
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.Name, &n.HP, &n.MaxHP, &n.IsDestroyed)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}

	cost := int(math.Ceil(float64(n.MaxHP-n.HP) / 50.0))
	if n.IsDestroyed {
		cost = int(math.Ceil(float64(n.MaxHP*2) / 50.0))
	}

	if cost <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ tài nguyên đang ở trạng thái nguyên vẹn tốt nhất."})
		return
	}

	// Check player stones
	var stones int
	err = tx.QueryRow(`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1`, playerID).Scan(&stones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên Thạch"})
		return
	}

	// Repair node
	_, err = tx.Exec(`
		UPDATE aperture_resource_nodes
		SET hp = max_hp, is_destroyed = false, updated_at = NOW()
		WHERE id = $1
	`, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái sửa chữa"})
		return
	}

	_ = tx.Commit()

	repairMsg := fmt.Sprintf("Sửa chữa thành công %s! Tiêu tốn %d Tiên Nguyên Thạch.", n.Name, cost)
	notify.Send(db.DB, playerID, "aperture_repair", "Sửa Chữa Tiên Khiếu", repairMsg)

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên Thạch"})
		return
	}

	// Calculate new stats
	newLevel := n.Level + 1
	newMaxHP := n.MaxHP + 500
	hpGain := newMaxHP - n.MaxHP // HP increased by this amount

	newProductionAmount := n.ProductionAmount
	newDefensePower := n.DefensePower
	newHPBonus := n.HPBonus

	if n.NodeCategory == "RESOURCE" {
		// Resource nodes: +50% production per level
		newProductionAmount = int64(float64(n.ProductionAmount) * 1.5)
		if newProductionAmount == n.ProductionAmount {
			newProductionAmount = n.ProductionAmount + 1
		}
	} else {
		// Defense nodes: +25% defense stats per level
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi nâng cấp: " + err.Error()})
		return
	}

	_ = tx.Commit()

	upgradeMsg := fmt.Sprintf("Nâng cấp %s lên cấp %d thành công! Tiêu tốn %d Tiên Nguyên Thạch.", n.Name, newLevel, upgradeCost)
	notify.Send(db.DB, playerID, "aperture_upgrade", "Nâng Cấp Tiên Khiếu", upgradeMsg)

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
		SELECT v.id, v.aperture_id, v.race, v.population, v.happiness, v.oppression, v.rebellion_rate, v.labor_intensity, v.created_at, v.updated_at
		FROM aperture_variant_humans v
		JOIN immortal_apertures a ON v.aperture_id = a.id
		WHERE a.player_id = $1
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin dị nhân"})
		return
	}
	defer rows.Close()

	list := []models.ApertureVariantHuman{}
	for rows.Next() {
		var v models.ApertureVariantHuman
		err = rows.Scan(&v.ID, &v.ApertureID, &v.Race, &v.Population, &v.Happiness, &v.Oppression, &v.RebellionRate, &v.LaborIntensity, &v.CreatedAt, &v.UpdatedAt)
		if err == nil {
			list = append(list, v)
		}
	}

	c.JSON(http.StatusOK, list)
}
