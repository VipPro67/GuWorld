package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type DictionaryResponse struct {
	Gu            interface{}              `json:"gu"`
	Items         interface{}              `json:"items"`
	Monsters      interface{}              `json:"monsters"`
	Constitutions []gamefx.ConstitutionDef `json:"constitutions"`
	Races         []gamefx.RaceDef         `json:"races"`
}

// GetDictionary returns the unified Gu and Item master catalogs, cached in Redis
func GetDictionary(c *gin.Context) {
	var resp DictionaryResponse
	cacheKey := "game_dictionary_unified"

	if err := cache.GetJSON(cacheKey, &resp); err == nil && resp.Gu != nil {
		c.JSON(http.StatusOK, resp)
		return
	}

	flatGu := make([]models.GuDictionaryItem, 0)
	for _, ranks := range models.GuRegistry {
		for _, item := range ranks {
			flatGu = append(flatGu, item)
		}
	}

	resp = DictionaryResponse{
		Gu:            flatGu,
		Items:         models.ItemRegistry,
		Monsters:      models.MonsterRegistry,
		Constitutions: gamefx.AllConstitutions(),
		Races:         gamefx.AllRaces(),
	}

	_ = cache.SetJSON(cacheKey, resp, 24*time.Hour)
	c.JSON(http.StatusOK, resp)
}
