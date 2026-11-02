package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
)

const maxClientMetricsBodyBytes = 64 << 10

// SubmitRealtimeClientMetrics records one baseline window uploaded by the
// Phaser client. Identity comes from the auth middleware and headers, never
// from the request body, so a client cannot overwrite another player's report.
func SubmitRealtimeClientMetrics(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxClientMetricsBodyBytes)
	var report monitor.RealtimeClientReport
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid realtime metrics payload"})
		return
	}

	report.PlayerID = playerID.String()
	report.TabID = c.GetHeader("X-Player-Tab-ID")
	report.IP = c.ClientIP()
	if err := monitor.GlobalClientMetrics.Record(report); err != nil {
		monitor.GlobalClientMetrics.RecordRejected()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// AdminGetRealtimeClientMetrics exposes the newest per-player baseline report
// next to the existing per-zone realtime stats endpoint.
func AdminGetRealtimeClientMetrics(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	c.JSON(http.StatusOK, monitor.GlobalClientMetrics.Snapshot())
}
