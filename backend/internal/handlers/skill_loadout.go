package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetCombatSkillLoadout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"candidates": []any{}, "configured": false})
}

func UpdateCombatSkillLoadout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true})
}
