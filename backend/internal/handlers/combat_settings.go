package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// GetCombatSettings returns the saved combat tactics and world controls.
func GetCombatSettings(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var settings models.AutoCombatConfig
	if err := db.DB.QueryRowContext(c.Request.Context(),
		`SELECT combat_settings FROM players WHERE id = $1`, authID,
	).Scan(&settings); err != nil {
		httperr.Respond(c, httperr.Internal("Failed to fetch player", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"combat_settings": settings})
}

// UpdateCombatSettings persists turn-based tactics together with world controls.
func UpdateCombatSettings(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		CombatSettings models.AutoCombatConfig `json:"combat_settings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.CombatSettings.Normalize()
	settingsJSON, err := json.Marshal(req.CombatSettings)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid combat settings"})
		return
	}
	if _, err = db.DB.ExecContext(c.Request.Context(),
		`UPDATE players SET combat_settings = $1 WHERE id = $2`, settingsJSON, authID,
	); err != nil {
		httperr.Respond(c, httperr.Internal("Failed to save combat settings", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Cập nhật cài đặt chiến đấu thành công!",
		"combat_settings": req.CombatSettings,
	})
}
