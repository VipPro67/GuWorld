package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// GetDictionary returns the unified Gu and Item master catalogs
func GetDictionary(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"gu":    models.GuRegistry,
		"items": models.ItemRegistry,
	})
}
