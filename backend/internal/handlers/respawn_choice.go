package handlers

import (
	"database/sql"
	"encoding/json"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type respawnChoice struct {
	Kind   string `json:"kind"`
	Label  string `json:"label"`
	Region string `json:"region"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

// GetPendingRespawn returns the one-time destination choices after an ordinary
// world death. Other death modes do not create a pending choice.
func GetPendingRespawn(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var raw []byte
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT options FROM player_pending_respawns
		WHERE player_id=$1 AND expires_at>NOW()
	`, playerID).Scan(&raw)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"pending": false})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải lựa chọn hồi sinh", err))
		return
	}
	var options []respawnChoice
	if err := json.Unmarshal(raw, &options); err != nil {
		httperr.Respond(c, httperr.Internal("Dữ liệu hồi sinh không hợp lệ", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"pending": true, "options": options})
}

// ChooseRespawn consumes the pending choice and moves the player to one of the
// destinations captured at death. The server never accepts client coordinates.
func ChooseRespawn(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var request struct {
		Kind string `json:"kind"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.Kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lựa chọn hồi sinh không hợp lệ"})
		return
	}
	if _, err := middleware.BeginRealtimeHandoff(playerID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Nhân vật đang đồng bộ trạng thái, hãy thử lại"})
		return
	}
	finished := false
	defer func() {
		if !finished {
			middleware.AbortRealtimeHandoff(playerID)
		}
	}()

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể bắt đầu hồi sinh", err))
		return
	}
	defer func() { _ = tx.Rollback() }()
	var raw []byte
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT options FROM player_pending_respawns
		WHERE player_id=$1 AND expires_at>NOW() FOR UPDATE
	`, playerID).Scan(&raw)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusConflict, gin.H{"error": "Lựa chọn hồi sinh đã hết hạn hoặc đã được sử dụng"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể xác nhận lựa chọn hồi sinh", err))
		return
	}
	var options []respawnChoice
	if err := json.Unmarshal(raw, &options); err != nil {
		httperr.Respond(c, httperr.Internal("Dữ liệu hồi sinh không hợp lệ", err))
		return
	}
	var chosen *respawnChoice
	for i := range options {
		if options[i].Kind == request.Kind {
			chosen = &options[i]
			break
		}
	}
	if chosen == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm hồi sinh không khả dụng"})
		return
	}
	var regionExists bool
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM regions WHERE id=$1 AND is_active=TRUE)`, chosen.Region).Scan(&regionExists); err != nil {
		logger.S.Errorf("respawn choice: validate region player=%s region=%q: %v", playerID, chosen.Region, err)
		httperr.Respond(c, httperr.Internal("Không thể xác thực khu vực hồi sinh", err))
		return
	}
	if !regionExists {
		c.JSON(http.StatusConflict, gin.H{"error": "Khu vực hồi sinh không còn khả dụng; hãy chọn điểm khác"})
		return
	}
	if err := RelocatePlayerRespawn(c.Request.Context(), tx, playerID, chosen.Region, chosen.X, chosen.Y); err != nil {
		logger.S.Errorf("respawn choice: relocate player=%s region=%q x=%d y=%d: %v", playerID, chosen.Region, chosen.X, chosen.Y, err)
		httperr.Respond(c, httperr.Internal("Không thể lưu vị trí hồi sinh", err))
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM player_pending_respawns WHERE player_id=$1`, playerID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể hoàn tất hồi sinh", err))
		return
	}
	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể xác nhận hồi sinh", err))
		return
	}
	middleware.FinishRealtimeHandoff(playerID)
	finished = true
	PublishRespawnWorldPosition(c.Request.Context(), playerID, chosen.Region, chosen.X, chosen.Y)
	PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated)
	c.JSON(http.StatusOK, gin.H{"success": true, "region": chosen.Region, "x": chosen.X, "y": chosen.Y})
}

func deathRespawnOptions(playerID uuid.UUID, deathRegion string, deathX, deathY int) []respawnChoice {
	options := make([]respawnChoice, 0, 3)
	mapID := worldMapIDForRegion(deathRegion)
	if region, x, y, found := NearestRespawnPointOnMap(deathX, deathY, mapID); found {
		options = append(options, respawnChoice{Kind: "safe", Label: "Khu an toàn gần nhất", Region: region, X: x, Y: y})
	} else {
		options = append(options, respawnChoice{Kind: "safe", Label: "Khu an toàn (vị trí hiện tại)", Region: deathRegion, X: deathX, Y: deathY})
	}
	var sectRegion string
	var sectX, sectY int
	if err := db.DB.QueryRow(`
		SELECT s.region,s.map_x,s.map_y FROM sect_members sm JOIN sects s ON s.id=sm.sect_id
		JOIN regions r ON r.id=s.region AND r.is_active=TRUE
		WHERE sm.player_id=$1
	`, playerID).Scan(&sectRegion, &sectX, &sectY); err == nil {
		options = append(options, respawnChoice{Kind: "sect", Label: "Tông môn", Region: sectRegion, X: sectX, Y: sectY})
	}
	if zone := ZoneByID(deathRegion); zone != nil {
		minX, maxX, minY, maxY := zone.MinX, zone.MaxX, zone.MinY, zone.MaxY
		if maxX <= minX || maxY <= minY {
			minX, maxX = deathX-200, deathX+200
			minY, maxY = deathY-200, deathY+200
		}
		options = append(options, respawnChoice{Kind: "random", Label: "Vị trí ngẫu nhiên trong khu hiện tại", Region: deathRegion,
			X: minX + rand.Intn(maxX-minX+1), Y: minY + rand.Intn(maxY-minY+1)})
	}
	return options
}
