package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type spatialRealm struct {
	TemplateID      string    `json:"template_id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	RequiredRank    int       `json:"required_rank"`
	RegionID        string    `json:"region_id"`
	EntryRegionID   string    `json:"entry_region_id"`
	EntryX          int       `json:"entry_x"`
	EntryY          int       `json:"entry_y"`
	DurationMinutes int       `json:"duration_minutes"`
	InstanceID      string    `json:"instance_id,omitempty"`
	OpenedAt        time.Time `json:"opened_at,omitempty"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	PlayerCount     int       `json:"player_count"`
}

// ListSpatialRealms exposes walkable shared realms independently from
// choice-based DREAM_REALM and legacy stage-based SECRET_REALM templates.
func ListSpatialRealms(c *gin.Context) {
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT t.id, t.name, t.description, t.required_rank, t.spatial_region_id,
		       COALESCE(gate.region_id,''), COALESCE(gate.x,0), COALESCE(gate.y,0),
		       t.duration_minutes, COALESCE(s.id::text,''), s.opened_at, s.expires_at,
	       (SELECT COUNT(*) FROM shared_spatial_instance_members m WHERE m.instance_id=s.id AND m.exited_at IS NULL)
		FROM instance_templates t
		LEFT JOIN map_cells gate ON gate.cell_type='ENTRY_PORTAL' AND gate.content_id=t.id AND gate.is_active=TRUE
		LEFT JOIN shared_spatial_instances s ON s.template_id=t.id AND s.status='ACTIVE' AND s.expires_at>NOW()
		WHERE t.type='SPATIAL_REALM' AND t.spatial_region_id IS NOT NULL
		ORDER BY t.required_rank, t.id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách bí cảnh"})
		return
	}
	defer rows.Close()
	result := make([]spatialRealm, 0)
	for rows.Next() {
		var item spatialRealm
		var opened, expires sql.NullTime
		if err := rows.Scan(&item.TemplateID, &item.Name, &item.Description, &item.RequiredRank, &item.RegionID,
			&item.EntryRegionID, &item.EntryX, &item.EntryY, &item.DurationMinutes, &item.InstanceID, &opened, &expires, &item.PlayerCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách bí cảnh"})
			return
		}
		if opened.Valid {
			item.OpenedAt = opened.Time
		}
		if expires.Valid {
			item.ExpiresAt = expires.Time
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách bí cảnh"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"realms": result})
}

// GetActiveSpatialRealm returns the player's current shared spatial realm, if any.
func GetActiveSpatialRealm(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var item spatialRealm
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT t.id,t.name,t.description,t.required_rank,t.spatial_region_id,t.duration_minutes,
		       s.id::text,s.opened_at,s.expires_at,
		       (SELECT COUNT(*) FROM shared_spatial_instance_members a WHERE a.instance_id=s.id AND a.exited_at IS NULL)
		FROM shared_spatial_instance_members m
		JOIN shared_spatial_instances s ON s.id=m.instance_id AND s.status='ACTIVE'
		JOIN instance_templates t ON t.id=s.template_id
		WHERE m.player_id=$1 AND m.exited_at IS NULL AND s.expires_at>NOW()`, playerID).
		Scan(&item.TemplateID, &item.Name, &item.Description, &item.RequiredRank, &item.RegionID, &item.DurationMinutes, &item.InstanceID, &item.OpenedAt, &item.ExpiresAt, &item.PlayerCount)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"realm": nil})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải bí cảnh hiện tại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"realm": item})
}

type enterSpatialRealmRequest struct {
	TemplateID string `json:"template_id" binding:"required"`
}

// EnterSpatialRealm joins the server-wide active copy, creating it atomically
// when the first player enters. Every member records a personal return point.
func EnterSpatialRealm(c *gin.Context) {
	var req enterSpatialRealmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template_id is required"})
		return
	}
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu giao dịch"})
		return
	}
	defer tx.Rollback()

	var oldRegion, entryItem, regionID string
	var oldX, oldY, rank, entryX, entryY, duration int
	var activeCombat sql.NullString
	var oldInstance sql.NullString
	err = tx.QueryRowContext(c.Request.Context(), `SELECT current_region,x,y,rank,active_combat::text,map_instance_id::text FROM players WHERE id=$1 FOR UPDATE`, playerID).
		Scan(&oldRegion, &oldX, &oldY, &rank, &activeCombat, &oldInstance)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhân vật"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhân vật"})
		return
	}
	if oldInstance.Valid || (activeCombat.Valid && activeCombat.String != "null" && activeCombat.String != "") {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu đang ở bí cảnh khác hoặc đang chiến đấu"})
		return
	}
	var hasStageSession bool
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM player_instance_sessions WHERE player_id=$1 AND status='ACTIVE')`, playerID).Scan(&hasStageSession); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra phiên phó bản hiện tại"})
		return
	}
	if hasStageSession {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu đang ở một phó bản khác"})
		return
	}

	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT t.entry_item,t.required_rank,t.spatial_region_id,t.duration_minutes,r.entry_x,r.entry_y
		FROM instance_templates t JOIN regions r ON r.id=t.spatial_region_id
		WHERE t.id=$1 AND t.type='SPATIAL_REALM' AND t.spatial_region_id IS NOT NULL AND r.is_active=TRUE AND UPPER(r.zone_type)='SECRET_REALM'
		FOR UPDATE OF t`, req.TemplateID).
		Scan(&entryItem, &rank, &regionID, &duration, &entryX, &entryY)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bí cảnh bản đồ này chưa được cấu hình"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải cấu hình bí cảnh"})
		return
	}
	var atEntryPortal bool
	if err := tx.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM map_cells
			WHERE region_id=$1 AND x=$2 AND y=$3 AND is_active=TRUE
			  AND UPPER(cell_type)='ENTRY_PORTAL' AND content_id=$4
			  AND UPPER(COALESCE(metadata->>'action',''))='ENTER_SPATIAL_REALM'
		)`, oldRegion, oldX, oldY, req.TemplateID).Scan(&atEntryPortal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra cổng vào bí cảnh"})
		return
	}
	if !atEntryPortal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần đến đúng cổng trên bản đồ thế giới để vào bí cảnh"})
		return
	}
	// Keep the character's rank separate from the template requirement.
	var playerRank int
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT rank FROM players WHERE id=$1`, playerID).Scan(&playerRank); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra cảnh giới"})
		return
	}
	if playerRank < rank {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cảnh giới chưa đạt yêu cầu vào bí cảnh"})
		return
	}
	inventory, err := db.LoadPlayerGus(tx, playerID.String(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra Cổ trùng đang trang bị"})
		return
	}
	if err := checkRegionEntry(regionID, playerRank, inventory); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if duration < 1 {
		duration = 120
	}

	if entryItem != "" && entryItem != "NONE" {
		qty, err := InventorySvc.GetQuantity(tx, playerID.String(), entryItem)
		if err != nil || qty < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu vật phẩm vào bí cảnh: " + entryItem})
			return
		}
		if err := InventorySvc.RemoveItem(tx, playerID.String(), entryItem, "materials", 1); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tiêu hao vật phẩm vào bí cảnh"})
			return
		}
	}

	// Expire stale session rows before opening the next shared run.
	var expiredIDs []uuid.UUID
	rows, err := tx.QueryContext(c.Request.Context(), `SELECT id FROM shared_spatial_instances WHERE template_id=$1 AND status='ACTIVE' AND expires_at<=NOW() FOR UPDATE`, req.TemplateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra phiên bí cảnh"})
		return
	}
	for rows.Next() {
		var id uuid.UUID
		if rows.Scan(&id) == nil {
			expiredIDs = append(expiredIDs, id)
		}
	}
	rows.Close()
	for _, id := range expiredIDs {
		if err := closeSpatialRealmTx(c.Request.Context(), tx, id, "TIMEOUT"); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đóng phiên bí cảnh hết hạn"})
			return
		}
	}

	var instanceID uuid.UUID
	err = tx.QueryRowContext(c.Request.Context(), `SELECT id FROM shared_spatial_instances WHERE template_id=$1 AND status='ACTIVE' AND expires_at>NOW() FOR UPDATE`, req.TemplateID).Scan(&instanceID)
	if err == sql.ErrNoRows {
		err = tx.QueryRowContext(c.Request.Context(), `INSERT INTO shared_spatial_instances(template_id,region_id,expires_at) VALUES($1,$2,NOW()+($3 * INTERVAL '1 minute')) RETURNING id`, req.TemplateID, regionID, duration).Scan(&instanceID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở phiên bí cảnh chung"})
		return
	}
	_, err = tx.ExecContext(c.Request.Context(), `INSERT INTO shared_spatial_instance_members(instance_id,player_id,return_region_id,return_x,return_y) VALUES($1,$2,$3,$4,$5)`, instanceID, playerID, oldRegion, oldX, oldY)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu đã ở trong một bí cảnh"})
		return
	}
	_, err = tx.ExecContext(c.Request.Context(), `UPDATE players SET current_region=$1,x=$2,y=$3,map_instance_id=$4,last_mana_update=NOW(),version=version+1 WHERE id=$5`, regionID, entryX, entryY, instanceID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuyển nhân vật vào bí cảnh"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu trạng thái vào bí cảnh"})
		return
	}
	for _, expiredID := range expiredIDs {
		syncExitedSpatialInstance(expiredID)
	}
	syncSpatialPlayer(playerID, regionID, entryX, entryY, instanceID)
	if entryItem != "" && entryItem != "NONE" {
		auditlog.Log(db.DB, auditlog.Entry{PlayerID: &playerID, Action: "spatial_realm.enter", EntityType: "instance_template", EntityID: req.TemplateID, RequestSummary: "Vào bí cảnh bản đồ chung", IPAddress: c.ClientIP(), Success: true})
	}
	c.JSON(http.StatusOK, gin.H{"instance_id": instanceID, "region_id": regionID, "x": entryX, "y": entryY})
}

// ExitSpatialRealm is the idempotent common exit path used by the exit portal,
// the timeout worker, and future Cổ Trùng effects.
func ExitSpatialRealm(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể bắt đầu giao dịch"})
		return
	}
	defer tx.Rollback()
	var instanceID uuid.UUID
	var region string
	var x, y, currentX, currentY int
	var currentRegion string
	err = tx.QueryRowContext(c.Request.Context(), `SELECT m.instance_id,m.return_region_id,m.return_x,m.return_y,p.current_region,p.x,p.y FROM shared_spatial_instance_members m JOIN players p ON p.id=m.player_id WHERE m.player_id=$1 AND m.exited_at IS NULL FOR UPDATE OF m,p`, playerID).Scan(&instanceID, &region, &x, &y, &currentRegion, &currentX, &currentY)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"exited": false})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải trạng thái bí cảnh"})
		return
	}
	var atExitPortal bool
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM map_cells WHERE region_id=$1 AND x=$2 AND y=$3 AND (UPPER(cell_type) IN ('EXIT_PORTAL','PORTAL_EXIT') OR UPPER(COALESCE(content_id,''))='EXIT_SPATIAL_REALM' OR UPPER(COALESCE(metadata->>'action',''))='EXIT_SPATIAL_REALM'))`, currentRegion, currentX, currentY).Scan(&atExitPortal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra cổng thoát"})
		return
	}
	if !atExitPortal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần đứng tại cổng thoát của bí cảnh"})
		return
	}
	result, err := tx.ExecContext(c.Request.Context(), `UPDATE players SET current_region=$1,x=$2,y=$3,map_instance_id=NULL,active_combat=NULL,last_mana_update=NOW(),version=version+1 WHERE id=$4 AND map_instance_id=$5`, region, x, y, playerID, instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đưa nhân vật về vị trí cũ"})
		return
	}
	updated, err := result.RowsAffected()
	if err != nil || updated != 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "Vị trí nhân vật không khớp với phiên bí cảnh"})
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE shared_spatial_instance_members SET exited_at=NOW(),exit_reason='PORTAL' WHERE player_id=$1 AND exited_at IS NULL`, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất thoát bí cảnh"})
		return
	}
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu trạng thái thoát"})
		return
	}
	syncSpatialPlayer(playerID, region, x, y, uuid.Nil)
	c.JSON(http.StatusOK, gin.H{"exited": true, "region_id": region, "x": x, "y": y})
}

func closeSpatialRealmTx(ctx context.Context, tx *sql.Tx, instanceID uuid.UUID, reason string) error {
	rows, err := tx.QueryContext(ctx, `SELECT player_id,return_region_id,return_x,return_y FROM shared_spatial_instance_members WHERE instance_id=$1 AND exited_at IS NULL FOR UPDATE`, instanceID)
	if err != nil {
		return err
	}
	type member struct {
		id     uuid.UUID
		region string
		x, y   int
	}
	var members []member
	for rows.Next() {
		var m member
		if err := rows.Scan(&m.id, &m.region, &m.x, &m.y); err != nil {
			rows.Close()
			return err
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, m := range members {
		if _, err := tx.ExecContext(ctx, `UPDATE players SET current_region=$1,x=$2,y=$3,map_instance_id=NULL,active_combat=NULL,last_mana_update=NOW(),version=version+1 WHERE id=$4 AND map_instance_id=$5`, m.region, m.x, m.y, m.id, instanceID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE shared_spatial_instance_members SET exited_at=NOW(),exit_reason=$1 WHERE player_id=$2 AND instance_id=$3 AND exited_at IS NULL`, reason, m.id, instanceID); err != nil {
			return err
		}
	}
	// A realm run is a finite shared map copy; its ground loot expires with it.
	var regionID string
	if err := tx.QueryRowContext(ctx, `SELECT region_id FROM shared_spatial_instances WHERE id=$1`, instanceID).Scan(&regionID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM map_ground_drops WHERE region_id=$1`, regionID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE shared_spatial_instances SET status='CLOSED',closed_at=NOW(),close_reason=$1 WHERE id=$2 AND status='ACTIVE'`, reason, instanceID)
	return err
}

func expireSpatialRealm(ctx context.Context, instanceID uuid.UUID) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM shared_spatial_instances WHERE id=$1 AND expires_at<=NOW() FOR UPDATE`, instanceID).Scan(&status)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if status == "ACTIVE" {
		if err := closeSpatialRealmTx(ctx, tx, instanceID, "TIMEOUT"); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	syncExitedSpatialInstance(instanceID)
	return nil
}

func syncExitedSpatialInstance(instanceID uuid.UUID) {
	rows, err := db.DB.Query(`SELECT player_id,return_region_id,return_x,return_y FROM shared_spatial_instance_members WHERE instance_id=$1 AND exit_reason='TIMEOUT' AND exited_at>NOW()-INTERVAL '2 minutes'`, instanceID)
	if err != nil {
		return
	}
	type moved struct {
		player uuid.UUID
		region string
		x, y   int
	}
	var members []moved
	for rows.Next() {
		var m moved
		if rows.Scan(&m.player, &m.region, &m.x, &m.y) == nil {
			members = append(members, m)
		}
	}
	rows.Close()
	for _, m := range members {
		syncSpatialPlayer(m.player, m.region, m.x, m.y, uuid.Nil)
	}
}

func StartSpatialRealmExpiryWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rows, err := db.DB.QueryContext(ctx, `SELECT id FROM shared_spatial_instances WHERE status='ACTIVE' AND expires_at<=NOW() ORDER BY expires_at LIMIT 100`)
				if err != nil {
					continue
				}
				ids := make([]uuid.UUID, 0, 100)
				for rows.Next() {
					var id uuid.UUID
					if rows.Scan(&id) == nil {
						ids = append(ids, id)
					}
				}
				rows.Close()
				for _, id := range ids {
					_ = expireSpatialRealm(ctx, id)
				}
			}
		}
	}()
}

func syncSpatialPlayer(playerID uuid.UUID, region string, x, y int, instanceID uuid.UUID) {
	var username, avatar string
	if err := db.DB.QueryRow(`SELECT username,COALESCE(avatar_url,'') FROM players WHERE id=$1`, playerID).Scan(&username, &avatar); err != nil {
		return
	}
	inst := ""
	if instanceID != uuid.Nil {
		inst = instanceID.String()
	}
	broadcastWorldPos(WorldPosEvent{ID: playerID.String(), Username: username, AvatarURL: avatar, X: x, Y: y, Region: region, MapID: worldMapIDForRegion(region), InstanceID: inst, TS: time.Now().UnixMilli(), ScopeResolved: true})
}
