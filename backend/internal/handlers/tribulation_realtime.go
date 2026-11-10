package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RealtimeResolveTribulationVictory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Turn-based mode active"})
}

func RealtimeEscapeTribulation(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Turn-based mode active"})
}

func RealtimeExitTribulation(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Turn-based mode active"})
}
