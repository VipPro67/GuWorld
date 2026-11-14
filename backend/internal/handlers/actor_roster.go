package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// ListActors exposes the canonical non-player actor roster.
func ListActors(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	actors, err := models.LoadOwnedNonPlayerActors(c.Request.Context(), db.DB, playerID.String())
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to load actors", err))
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
	actor, err := models.LoadOwnedNonPlayerActor(c.Request.Context(), db.DB, actorID, playerID.String())
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Actor not found"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to load actor", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"actor": actor})
}
