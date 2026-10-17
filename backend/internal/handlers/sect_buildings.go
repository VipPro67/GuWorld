package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type UpgradeBuildingRequest struct {
	BuildingName string `json:"building_name"`
}

func getBuildingCost(buildingName string, targetLevel int) int64 {
	// Cost scales with level
	baseFunds := int64(1000)

	switch buildingName {
	case "main_hall":
		baseFunds = 5000
	case "defensive_array":
		baseFunds = 2000
	case "spirit_vein":
		baseFunds = 3000
	case "treasury":
		baseFunds = 1500
	case "training_ground":
		baseFunds = 1500
	default:
		return 0
	}

	mult := int64(targetLevel * targetLevel)
	return baseFunds * mult
}

func UpgradeSectBuilding(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpgradeBuildingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	validBuildings := map[string]bool{
		"main_hall": true, "defensive_array": true, "spirit_vein": true,
		"treasury": true, "training_ground": true,
	}
	if !validBuildings[req.BuildingName] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kiến trúc không hợp lệ!"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có quản lý mới được nâng cấp kiến trúc!"})
		return
	}

	var level int
	var funds int64
	var buildingsJSON []byte
	var sectRegion string
	var mapX, mapY int
	err = tx.QueryRow(`SELECT level, funds, buildings, region, map_x, map_y FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&level, &funds, &buildingsJSON, &sectRegion, &mapX, &mapY)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tông môn!"})
		return
	}

	var buildings map[string]int
	if err := json.Unmarshal(buildingsJSON, &buildings); err != nil {
		buildings = make(map[string]int)
	}

	currentLevel := buildings[req.BuildingName]
	if currentLevel == 0 {
		currentLevel = 1
	}

	// Main hall constraints
	if req.BuildingName != "main_hall" {
		mainHallLvl := buildings["main_hall"]
		if mainHallLvl == 0 {
			mainHallLvl = 1
		}
		if currentLevel >= mainHallLvl {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cấp kiến trúc không được vượt quá cấp Chính Điện!"})
			return
		}
	}

	targetLevel := currentLevel + 1

	// When upgrading main_hall, verify expanded territory does not overlap neighboring sects
	if req.BuildingName == "main_hall" {
		newRadius := 5 + targetLevel
		hasOverlap, conflictingSect, errOverlap := CheckTerritoryOverlap(tx, sectRegion, mapX, mapY, newRadius, sectID)
		if errOverlap != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kiểm tra mở rộng lãnh địa thất bại: " + errOverlap.Error()})
			return
		}
		if hasOverlap {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Lãnh địa sau khi nâng cấp Chính Điện (bán kính %d ô) sẽ xâm lấn tông môn \"%s\", không thể mở rộng!", newRadius, conflictingSect)})
			return
		}
	}

	reqFunds := getBuildingCost(req.BuildingName, targetLevel)

	if funds < reqFunds {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ (cần %d)!", reqFunds)})
		return
	}

	buildings[req.BuildingName] = targetLevel

	newBuildings, _ := json.Marshal(buildings)

	// Sync sect level with main_hall level
	var newLevel int
	if req.BuildingName == "main_hall" {
		newLevel = targetLevel
	} else {
		newLevel = buildings["main_hall"]
		if newLevel == 0 {
			newLevel = 1
		}
	}

	_, err = tx.Exec(`UPDATE sects SET level = $1, funds = funds - $2, buildings = $3 WHERE id = $4`, newLevel, reqFunds, newBuildings, sectID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	announceSectFundsSpend(sectID, playerID, fmt.Sprintf("nâng cấp %s lên cấp %d", req.BuildingName, targetLevel), reqFunds, funds-reqFunds)

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Đã nâng cấp %s lên cấp %d!", req.BuildingName, targetLevel)})
}
