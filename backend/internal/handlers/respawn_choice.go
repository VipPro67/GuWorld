package handlers

import (
	"database/sql"
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải lựa chọn hồi sinh"})
		return
	}
	var options []respawnChoice
	if err := json.Unmarshal(raw, &options); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu hồi sinh không hợp lệ"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu hồi sinh"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận lựa chọn hồi sinh"})
		return
	}
	var options []respawnChoice
	if err := json.Unmarshal(raw, &options); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu hồi sinh không hợp lệ"})
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
	if _, err := tx.ExecContext(c.Request.Context(), `
		UPDATE players SET hp=1,current_region=$1,world_zone_id=$1,world_instance_id='',map_instance_id=NULL,
			x=$2,y=$3,world_x=$2,world_y=$3,world_safe_x=$2,world_safe_y=$3,
			world_location_revision=world_location_revision+1,version=version+1
		WHERE id=$4
	`, chosen.Region, chosen.X, chosen.Y, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu vị trí hồi sinh"})
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `
		INSERT INTO player_world_state_v2 (player_id,hp,shield,map_version,content_version,updated_at)
		VALUES ($1,1,0,'','',NOW()) ON CONFLICT (player_id)
		DO UPDATE SET hp=1,shield=0,updated_at=NOW()
	`, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái hồi sinh"})
		return
	}
	if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM player_pending_respawns WHERE player_id=$1`, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất hồi sinh"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận hồi sinh"})
		return
	}
	middleware.FinishRealtimeHandoff(playerID)
	finished = true
	var username, avatarURL string
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT username,COALESCE(avatar_url,'') FROM players WHERE id=$1`, playerID).Scan(&username, &avatarURL); err == nil {
		mapID := "world_map"
		if zone := ZoneByID(chosen.Region); zone != nil && zone.WorldMapID != "" {
			mapID = zone.WorldMapID
		}
		worldPosIndex.Set(WorldPosEntry{ID: playerID.String(), Username: username, AvatarURL: avatarURL, X: chosen.X, Y: chosen.Y, Region: chosen.Region, MapID: mapID, TS: time.Now().UnixMilli()})
	}
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
