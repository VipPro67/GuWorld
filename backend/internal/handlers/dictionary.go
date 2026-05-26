package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// ItemSpec represents static item specs returned to the client
type ItemSpec struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"item_type"`
	Description string `json:"description"`
}

// GetDictionary returns the unified Gu and Item master catalogs
func GetDictionary(c *gin.Context) {
	// Query Item Dictionary
	rows, err := db.DB.Query("SELECT id, name, item_type, description FROM item_dictionary")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load item catalog"})
		return
	}
	defer rows.Close()

	items := make(map[string]ItemSpec)
	for rows.Next() {
		var item ItemSpec
		err = rows.Scan(&item.ID, &item.Name, &item.Type, &item.Description)
		if err != nil {
			continue
		}
		items[item.ID] = item
	}

	c.JSON(http.StatusOK, gin.H{
		"gu":    models.GuRegistry,
		"items": items,
	})
}
