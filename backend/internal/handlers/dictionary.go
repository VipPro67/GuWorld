package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type DictionaryResponse struct {
	Gu    interface{} `json:"gu"`
	Items interface{} `json:"items"`
}

// GetDictionary returns the unified Gu and Item master catalogs, cached in Redis
func GetDictionary(c *gin.Context) {
	var resp DictionaryResponse
	cacheKey := "game_dictionary_unified"

	if err := cache.GetJSON(cacheKey, &resp); err == nil && resp.Gu != nil {
		c.JSON(http.StatusOK, resp)
		return
	}

	resp = DictionaryResponse{
		Gu:    models.GuRegistry,
		Items: models.ItemRegistry,
	}

	_ = cache.SetJSON(cacheKey, resp, 24*time.Hour)
	c.JSON(http.StatusOK, resp)
}
