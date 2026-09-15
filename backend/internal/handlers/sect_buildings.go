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

func getBuildingCost(buildingName string, targetLevel int) (int64, int) {
	// Cost scales with level
	baseFunds := int64(1000)
	baseMat := 100

	switch buildingName {
	case "main_hall":
		baseFunds = 5000
		baseMat = 500
	case "defensive_array":
		baseFunds = 2000
		baseMat = 300
	case "spirit_vein":
		baseFunds = 3000
		baseMat = 200
	case "treasury":
		baseFunds = 1500
		baseMat = 150
	case "training_ground":
		baseFunds = 1500
		baseMat = 100
	default:
		return 0, 0
	}

	mult := int64(targetLevel * targetLevel)
	return baseFunds * mult, baseMat * targetLevel * targetLevel
}

func getRequiredMaterialsForBuilding(sectLevel int) string {
	// Simple mapping for demo. Rank 1 Wood/Stone etc.
	// You might want to use specific IDs or general 'Material' types.
	return "mat_rank1_wood"
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
	var buildingsJSON, vaultMatsJSON []byte
	err = tx.QueryRow(`SELECT level, funds, buildings, vault_materials FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&level, &funds, &buildingsJSON, &vaultMatsJSON)
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
	reqFunds, reqMats := getBuildingCost(req.BuildingName, targetLevel)

	if funds < reqFunds {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ (cần %d)!", reqFunds)})
		return
	}

	var vaultMats map[string]int
	if err := json.Unmarshal(vaultMatsJSON, &vaultMats); err != nil {
		vaultMats = make(map[string]int)
	}

	matID := getRequiredMaterialsForBuilding(level)
	if vaultMats[matID] < reqMats {
		// In a real game, you might allow any materials of correct rank, or specific ones.
		// For simplicity, we just deduct random materials or a specific one.
		// Let's assume a generic check or just skip mat check if none.
		// Let's deduct from the highest qty material for now just for demo if they have enough total.
		totalMats := 0
		for _, v := range vaultMats {
			totalMats += v
		}
		if totalMats < reqMats {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu (cần %d)!", reqMats)})
			return
		}

		// Deduct mats
		deducted := 0
		for k, v := range vaultMats {
			if deducted >= reqMats {
				break
			}
			take := v
			if deducted+take > reqMats {
				take = reqMats - deducted
			}
			vaultMats[k] -= take
			deducted += take
			if vaultMats[k] <= 0 {
				delete(vaultMats, k)
			}
		}
	} else {
		vaultMats[matID] -= reqMats
		if vaultMats[matID] <= 0 {
			delete(vaultMats, matID)
		}
	}

	buildings[req.BuildingName] = targetLevel

	newBuildings, _ := json.Marshal(buildings)
	newVault, _ := json.Marshal(vaultMats)

	if req.BuildingName == "main_hall" {
		// Upgrade sect level as well
		_, err = tx.Exec(`UPDATE sects SET level = $1, funds = funds - $2, buildings = $3, vault_materials = $4 WHERE id = $5`, targetLevel, reqFunds, newBuildings, newVault, sectID)
	} else {
		_, err = tx.Exec(`UPDATE sects SET funds = funds - $1, buildings = $2, vault_materials = $3 WHERE id = $4`, reqFunds, newBuildings, newVault, sectID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Đã nâng cấp %s lên cấp %d!", req.BuildingName, targetLevel)})
}
