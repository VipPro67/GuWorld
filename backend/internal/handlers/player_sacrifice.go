package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type SacrificeNPCRequest struct {
	NPCID string `json:"npc_id" binding:"required"`
}

// SacrificeNPC handles sacrificing clan/sect NPCs using Huyết Sọ Cổ to raise player aptitude
func SacrificeNPC(c *gin.Context) {
	var req SacrificeNPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "npc_id là bắt buộc"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player status"})
		}
		return
	}

	// 1. Resolve the sacrifice ability and its tuning from the equipped Gu effects.
	aptitudeGain, aptitudeCap := 0, 0
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		if gain := models.GuMechanicFloat(gu, "SACRIFICE_NPC", "aptitude_gain"); gain > 0 {
			aptitudeGain = int(gain)
			aptitudeCap = int(models.GuMechanicFloat(gu, "SACRIFICE_NPC", "aptitude_cap"))
			break
		}
	}
	if aptitudeGain <= 0 || aptitudeCap <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu cần phải trang bị Huyết Sọ Cổ (GU_BLOOD_SKULL) ở trạng thái ACTIVE để thực hiện hiến tế!"})
		return
	}

	// 2. Check if NPC exists
	var npcName string
	var npcAptitude int
	err = tx.QueryRow("SELECT name, aptitude_percent FROM npcs WHERE id = $1 FOR UPDATE", req.NPCID).Scan(&npcName, &npcAptitude)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy NPC đệ tử này trong bang phái"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query NPC"})
		}
		return
	}

	// 3. Delete NPC permanently
	_, err = tx.Exec("DELETE FROM npcs WHERE id = $1", req.NPCID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete NPC"})
		return
	}

	// 4. Boost Player Aptitude using the catalog mechanic.
	oldAptitude := player.AptitudePercent
	player.AptitudePercent += aptitudeGain
	if player.AptitudePercent > aptitudeCap {
		player.AptitudePercent = aptitudeCap
	}

	// Dynamically update Aptitude Type
	if player.AptitudePercent >= 80 {
		player.AptitudeType = "Giáp"
	} else if player.AptitudePercent >= 60 {
		player.AptitudeType = "Ất"
	} else if player.AptitudePercent >= 40 {
		player.AptitudeType = "Bính"
	} else {
		player.AptitudeType = "Đinh"
	}

	// 5. Increase Aperture Stress (+40)
	player.ApertureStress += 40.0

	qiGained := 0.0

	// 6. Save back to DB
	_, err = tx.Exec(`
		UPDATE players
		SET aptitude_percent = $1, aptitude_type = $2, aperture_stress = $3, human_qi = $4
		WHERE id = $5
	`, player.AptitudePercent, player.AptitudeType, player.ApertureStress, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player stats"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         fmt.Sprintf("Hiến tế thành công %s! Huyết Sọ Cổ đã nghiền nát cốt nhục đệ tử, chuyển hóa chân huyết nâng cao tư chất Không khiếu của đạo hữu (+5%% tư chất). Nhân Khí +%.0f.", npcName, qiGained),
		"old_aptitude":    oldAptitude,
		"new_aptitude":    player.AptitudePercent,
		"aptitude_type":   player.AptitudeType,
		"aperture_stress": player.ApertureStress,
	})
}
