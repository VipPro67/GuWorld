package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// ListActors exposes the canonical, read-only roster during the foundation
// release. Legacy pet and named-NPC endpoints remain authoritative for writes.
func ListActors(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if err := db.SyncLegacyActorProjections(c.Request.Context(), playerID.String()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh actors"})
		return
	}
	actors, err := models.LoadOwnedNonPlayerActors(c.Request.Context(), db.DB, playerID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load actors"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"actors": actors})
}

// GetActor never reveals whether an actor owned by another player exists.
func GetActor(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	actorID := c.Param("id")
	if _, err := uuid.Parse(actorID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid actor ID"})
		return
	}
	if err := db.SyncLegacyActorProjections(c.Request.Context(), playerID.String()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refresh actor"})
		return
	}
	actor, err := models.LoadOwnedNonPlayerActor(c.Request.Context(), db.DB, actorID, playerID.String())
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Actor not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load actor"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"actor": actor})
}
