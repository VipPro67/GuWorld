package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// RealmRules defines the rule engine constraints on a spatial realm or inheritance ground.
type RealmRules struct {
	DurationMinutes      int    `json:"duration_minutes"`
	PvPMode              string `json:"pvp_mode"` // "SAFE", "FREE_PK", "FACTION_PK"
	MaxAllowedRank       int    `json:"max_allowed_rank"`
	ApplyRankSuppression bool   `json:"apply_rank_suppression"`
	DeathPenalty         string `json:"death_penalty"` // "EVICT", "SAFE_RESPAWN"
	AllowTeleport        bool   `json:"allow_teleport"`
	EntryItem            string `json:"entry_item"`
	EntryItemCount       int    `json:"entry_item_count"`
	MinMembers           int    `json:"min_members"`
	MaxMembers           int    `json:"max_members"`
}

func parseRealmRules(raw []byte, defaultDuration int) RealmRules {
	r := RealmRules{
		DurationMinutes: defaultDuration,
		PvPMode:         "SAFE",
		DeathPenalty:    "EVICT",
		AllowTeleport:   false,
		EntryItemCount:  1,
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &r)
	}
	if r.DurationMinutes <= 0 {
		r.DurationMinutes = defaultDuration
	}
	if r.DurationMinutes <= 0 {
		r.DurationMinutes = 120
	}
	if r.PvPMode == "" {
		r.PvPMode = "SAFE"
	}
	if r.DeathPenalty == "" {
		r.DeathPenalty = "EVICT"
	}
	if r.EntryItemCount <= 0 {
		r.EntryItemCount = 1
	}
	return r
}

type spatialRealm struct {
	TemplateID      string     `json:"template_id"`
	Name            string     `json:"name"`
	Type            string     `json:"type"`
	Scope           string     `json:"scope"` // "SHARED", "PARTY", "SOLO"
	Description     string     `json:"description"`
	RequiredRank    int        `json:"required_rank"`
	RegionID        string     `json:"region_id"`
	EntryRegionID   string     `json:"entry_region_id"`
	EntryX          int        `json:"entry_x"`
	EntryY          int        `json:"entry_y"`
	DurationMinutes int        `json:"duration_minutes"`
	InstanceID      string     `json:"instance_id,omitempty"`
	OpenedAt        time.Time  `json:"opened_at,omitempty"`
	ExpiresAt       time.Time  `json:"expires_at,omitempty"`
	PlayerCount     int        `json:"player_count"`
	Rules           RealmRules `json:"rules"`
}

// ListSpatialRealms exposes walkable realms (both SHARED and PARTY/SOLO spatial templates).
func ListSpatialRealms(c *gin.Context) {
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT t.id, t.name, t.type, COALESCE(t.scope, 'SHARED'), t.description, t.required_rank, t.spatial_region_id,
		       COALESCE(gate.region_id,''), COALESCE(gate.x,0), COALESCE(gate.y,0),
		       t.duration_minutes, COALESCE(t.rules, '{}'::jsonb),
		       COALESCE(s.id::text,''), s.opened_at, s.expires_at,
		       (SELECT COUNT(*) FROM shared_spatial_instance_members m WHERE m.instance_id=s.id AND m.exited_at IS NULL)
		FROM instance_templates t
		LEFT JOIN map_cells gate ON gate.cell_type='ENTRY_PORTAL' AND gate.content_id=t.id AND gate.is_active=TRUE
		LEFT JOIN shared_spatial_instances s ON s.template_id=t.id AND s.status='ACTIVE' AND s.expires_at>NOW()
		WHERE t.type IN ('SPATIAL_REALM', 'SPATIAL_INHERITANCE') AND t.spatial_region_id IS NOT NULL
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
		var rawRules []byte
		if err := rows.Scan(&item.TemplateID, &item.Name, &item.Type, &item.Scope, &item.Description, &item.RequiredRank, &item.RegionID,
			&item.EntryRegionID, &item.EntryX, &item.EntryY, &item.DurationMinutes, &rawRules, &item.InstanceID, &opened, &expires, &item.PlayerCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc danh sách bí cảnh: " + err.Error()})
			return
		}
		item.Rules = parseRealmRules(rawRules, item.DurationMinutes)
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

// GetActiveSpatialRealm returns the player's current spatial realm, if any.
func GetActiveSpatialRealm(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var item spatialRealm
	var rawRules []byte
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT t.id, t.name, t.type, COALESCE(t.scope, 'SHARED'), t.description, t.required_rank, t.spatial_region_id, t.duration_minutes,
		       COALESCE(t.rules, '{}'::jsonb),
		       s.id::text, s.opened_at, s.expires_at,
		       (SELECT COUNT(*) FROM shared_spatial_instance_members a WHERE a.instance_id=s.id AND a.exited_at IS NULL)
		FROM shared_spatial_instance_members m
		JOIN shared_spatial_instances s ON s.id=m.instance_id AND s.status='ACTIVE'
		JOIN instance_templates t ON t.id=s.template_id
		WHERE m.player_id=$1 AND m.exited_at IS NULL AND s.expires_at>NOW()`, playerID).
		Scan(&item.TemplateID, &item.Name, &item.Type, &item.Scope, &item.Description, &item.RequiredRank, &item.RegionID, &item.DurationMinutes, &rawRules, &item.InstanceID, &item.OpenedAt, &item.ExpiresAt, &item.PlayerCount)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"realm": nil})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải bí cảnh hiện tại"})
		return
	}
	item.Rules = parseRealmRules(rawRules, item.DurationMinutes)
	c.JSON(http.StatusOK, gin.H{"realm": item})
}

type enterSpatialRealmRequest struct {
	TemplateID string `json:"template_id" binding:"required"`
}

// EnterSpatialRealm joins or creates a spatial realm with Hybrid scope (SHARED, PARTY, SOLO)
// and validates all rules via the Rule Engine.
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

	var oldRegion, entryItem, regionID, scope string
	var oldX, oldY, requiredRank, entryX, entryY, duration int
	var rawRules []byte
	var activeCombat sql.NullString
	var oldInstance sql.NullString
	err = tx.QueryRowContext(c.Request.Context(), `SELECT current_region,x,y,rank,active_combat::text,map_instance_id::text FROM players WHERE id=$1 FOR UPDATE`, playerID).
		Scan(&oldRegion, &oldX, &oldY, &requiredRank, &activeCombat, &oldInstance)
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
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu đang ở một mộng cảnh hoặc phó bản khác"})
		return
	}

	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT t.entry_item, t.required_rank, t.spatial_region_id, t.duration_minutes,
		       COALESCE(t.scope, 'SHARED'), COALESCE(t.rules, '{}'::jsonb),
		       r.entry_x, r.entry_y
		FROM instance_templates t JOIN regions r ON r.id=t.spatial_region_id
		WHERE t.id=$1 AND t.type IN ('SPATIAL_REALM', 'SPATIAL_INHERITANCE') AND t.spatial_region_id IS NOT NULL AND r.is_active=TRUE
		FOR UPDATE OF t`, req.TemplateID).
		Scan(&entryItem, &requiredRank, &regionID, &duration, &scope, &rawRules, &entryX, &entryY)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bí cảnh bản đồ này chưa được cấu hình"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải cấu hình bí cảnh"})
		return
	}

	// Parse Realm Rules
	rules := parseRealmRules(rawRules, duration)
	if rules.DurationMinutes > 0 {
		duration = rules.DurationMinutes
	}

	var atEntryPortal bool
	if err := tx.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM map_cells
			WHERE region_id=$1 AND x=$2 AND y=$3 AND is_active=TRUE
			  AND UPPER(cell_type)='ENTRY_PORTAL' AND content_id=$4
			  AND (UPPER(COALESCE(metadata->>'action','')) IN ('ENTER_SPATIAL_REALM', 'ENTER_INHERITANCE') OR UPPER(COALESCE(content_id,''))=$4)
		)`, oldRegion, oldX, oldY, req.TemplateID).Scan(&atEntryPortal); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra cổng vào bí cảnh"})
		return
	}
	if !atEntryPortal {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần đến đúng cổng trên bản đồ thế giới để vào bí cảnh"})
		return
	}

	// Check rank requirements
	var playerRank int
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT rank FROM players WHERE id=$1`, playerID).Scan(&playerRank); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra cảnh giới"})
		return
	}
	if playerRank < requiredRank {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Cảnh giới chưa đạt yêu cầu vào bí cảnh (cần Bậc %d)", requiredRank)})
		return
	}
	if rules.MaxAllowedRank > 0 && playerRank > rules.MaxAllowedRank && !rules.ApplyRankSuppression {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Cảnh giới vượt quá giới hạn tối đa cho phép vào bí cảnh (tối đa Bậc %d)", rules.MaxAllowedRank)})
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

	// Check entry item consumption (prioritize rules.EntryItem if defined, fallback to template)
	neededItem := rules.EntryItem
	if neededItem == "" {
		neededItem = entryItem
	}
	neededCount := rules.EntryItemCount
	if neededCount <= 0 {
		neededCount = 1
	}

	if neededItem != "" && neededItem != "NONE" {
		qty, err := InventorySvc.GetQuantity(tx, playerID.String(), neededItem)
		if err != nil || qty < int64(neededCount) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Thiếu vật phẩm vào bí cảnh: %dx %s", neededCount, neededItem)})
			return
		}
		if err := InventorySvc.RemoveItem(tx, playerID.String(), neededItem, "materials", int64(neededCount)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tiêu hao vật phẩm vào bí cảnh: " + err.Error()})
			return
		}
	}

	// Expire stale session rows for this template
	var expiredIDs []uuid.UUID
	rows, err := tx.QueryContext(c.Request.Context(), `SELECT id FROM shared_spatial_instances WHERE template_id=$1 AND status='ACTIVE' AND expires_at<=NOW() FOR UPDATE`, req.TemplateID)
	if err == nil {
		for rows.Next() {
			var id uuid.UUID
			if rows.Scan(&id) == nil {
				expiredIDs = append(expiredIDs, id)
			}
		}
		rows.Close()
		for _, id := range expiredIDs {
			_ = closeSpatialRealmTx(c.Request.Context(), tx, id, "TIMEOUT")
		}
	}

	// Hybrid scope allocation: SHARED, PARTY, or SOLO
	var instanceID uuid.UUID
	if scope == "PARTY" {
		var partyID uuid.UUID
		err := tx.QueryRowContext(c.Request.Context(), `SELECT party_id FROM party_members WHERE player_id=$1`, playerID).Scan(&partyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Khu vực truyền thừa này yêu cầu phải tham gia vào một tổ đội!"})
			return
		}

		var memberCount int
		_ = tx.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM party_members WHERE party_id=$1`, partyID).Scan(&memberCount)
		if rules.MinMembers > 0 && memberCount < rules.MinMembers {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tổ đội cần tối thiểu %d thành viên để vào truyền thừa!", rules.MinMembers)})
			return
		}
		if rules.MaxMembers > 0 && memberCount > rules.MaxMembers {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tổ đội vượt quá tối đa %d thành viên cho phép!", rules.MaxMembers)})
			return
		}

		err = tx.QueryRowContext(c.Request.Context(), `
			SELECT id FROM shared_spatial_instances 
			WHERE template_id=$1 AND party_id=$2 AND status='ACTIVE' AND expires_at>NOW() FOR UPDATE
		`, req.TemplateID, partyID).Scan(&instanceID)
		if err == sql.ErrNoRows {
			err = tx.QueryRowContext(c.Request.Context(), `
				INSERT INTO shared_spatial_instances(template_id, region_id, scope, party_id, expires_at)
				VALUES($1, $2, 'PARTY', $3, NOW() + ($4 * INTERVAL '1 minute'))
				RETURNING id
			`, req.TemplateID, regionID, partyID, duration).Scan(&instanceID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo phiên truyền thừa tổ đội: " + err.Error()})
			return
		}
	} else if scope == "SOLO" {
		err = tx.QueryRowContext(c.Request.Context(), `
			SELECT id FROM shared_spatial_instances 
			WHERE template_id=$1 AND owner_id=$2 AND status='ACTIVE' AND expires_at>NOW() FOR UPDATE
		`, req.TemplateID, playerID).Scan(&instanceID)
		if err == sql.ErrNoRows {
			err = tx.QueryRowContext(c.Request.Context(), `
				INSERT INTO shared_spatial_instances(template_id, region_id, scope, owner_id, expires_at)
				VALUES($1, $2, 'SOLO', $3, NOW() + ($4 * INTERVAL '1 minute'))
				RETURNING id
			`, req.TemplateID, regionID, playerID, duration).Scan(&instanceID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo phiên truyền thừa cá nhân: " + err.Error()})
			return
		}
	} else {
		// SHARED scope
		err = tx.QueryRowContext(c.Request.Context(), `
			SELECT id FROM shared_spatial_instances 
			WHERE template_id=$1 AND scope='SHARED' AND status='ACTIVE' AND expires_at>NOW() FOR UPDATE
		`, req.TemplateID).Scan(&instanceID)
		if err == sql.ErrNoRows {
			err = tx.QueryRowContext(c.Request.Context(), `
				INSERT INTO shared_spatial_instances(template_id, region_id, scope, expires_at)
				VALUES($1, $2, 'SHARED', NOW() + ($3 * INTERVAL '1 minute'))
				RETURNING id
			`, req.TemplateID, regionID, duration).Scan(&instanceID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở phiên bí cảnh chung: " + err.Error()})
			return
		}
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
	if neededItem != "" && neededItem != "NONE" {
		auditlog.Log(db.DB, auditlog.Entry{PlayerID: &playerID, Action: "spatial_realm.enter", EntityType: "instance_template", EntityID: req.TemplateID, RequestSummary: fmt.Sprintf("Vào bí cảnh bản đồ (%s): %dx %s", scope, neededCount, neededItem), IPAddress: c.ClientIP(), Success: true})
	}
	c.JSON(http.StatusOK, gin.H{"instance_id": instanceID, "region_id": regionID, "x": entryX, "y": entryY, "scope": scope, "rules": rules})
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
	if _, err := tx.ExecContext(ctx, `DELETE FROM map_ground_drops WHERE region_id=$1 AND (map_instance_id=$2 OR map_instance_id IS NULL)`, regionID, instanceID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE shared_spatial_instances SET status='CLOSED',closed_at=NOW(),close_reason=$1 WHERE id=$2 AND status='ACTIVE'`, reason, instanceID)
	return err
}

// GetSpatialRealmRules returns the active rule set for a given spatial instance.
func GetSpatialRealmRules(instanceID uuid.UUID) (*RealmRules, error) {
	var rulesJSON []byte
	var duration int
	err := db.DB.QueryRow(`
		SELECT COALESCE(t.rules, '{}'::jsonb), t.duration_minutes
		FROM shared_spatial_instances s
		JOIN instance_templates t ON t.id = s.template_id
		WHERE s.id = $1
	`, instanceID).Scan(&rulesJSON, &duration)
	if err != nil {
		return nil, err
	}
	rules := parseRealmRules(rulesJSON, duration)
	return &rules, nil
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
	worker.RunEvery(ctx, 15*time.Second, "Spatial Realm Expiry", func(ctx context.Context) {
		rows, err := db.DB.QueryContext(ctx, `SELECT id FROM shared_spatial_instances WHERE status='ACTIVE' AND expires_at<=NOW() ORDER BY expires_at LIMIT 100`)
		if err != nil {
			return
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
	})
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

// ClaimAltarRequest specifies the altar being claimed by the player.
type ClaimAltarRequest struct {
	AltarID string `json:"altar_id" binding:"required"`
	X       *int   `json:"x,omitempty"`
	Y       *int   `json:"y,omitempty"`
}

// AltarMetadata describes the trial rules, requirements, and legacy rewards of an INHERITANCE_ALTAR cell.
type AltarMetadata struct {
	AltarName         string           `json:"altar_name"`
	Path              string           `json:"path"`
	Tier              string           `json:"tier"` // "MORTAL" or "IMMORTAL"
	GuardianSpawnID   string           `json:"guardian_spawn_id,omitempty"`
	GuardianCoord     *AltarCoord      `json:"guardian_coord,omitempty"`
	ReqRank           int              `json:"req_rank,omitempty"`
	ReqDaoMarks       map[string]int64 `json:"req_dao_marks,omitempty"`
	RequiredItem      string           `json:"required_item,omitempty"`
	RequiredItemCount int64            `json:"required_item_count,omitempty"`
	CooldownHours     int              `json:"cooldown_hours,omitempty"`
	Rewards           AltarRewards     `json:"rewards"`
}

// AltarCoord represents grid coordinates of a guardian or altar tile.
type AltarCoord struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// AltarRewards defines the legacy items, recipes, Gus, stones, and narrative text granted upon completing the trial.
type AltarRewards struct {
	Gus            []string         `json:"gus,omitempty"`
	Recipes        []string         `json:"recipes,omitempty"`
	PrimevalStones int64            `json:"primeval_stones,omitempty"`
	Materials      map[string]int64 `json:"materials,omitempty"`
	LegacyText     string           `json:"legacy_text,omitempty"`
}

// GetInheritanceAltarInfo returns detailed status of an altar for previewing in UI modal
func GetInheritanceAltarInfo(c *gin.Context) {
	playerIDRaw, ok := c.Get("player_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa xác thực"})
		return
	}
	playerID, ok := playerIDRaw.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Mã người chơi không hợp lệ"})
		return
	}
	altarID := strings.TrimSpace(c.Query("altar_id"))
	if altarID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu altar_id"})
		return
	}

	var currentRegion string
	var px, py int
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT current_region, x, y FROM players WHERE id = $1`, playerID).Scan(&currentRegion, &px, &py); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	var cellX, cellY int
	var metaRaw string
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT x, y, COALESCE(metadata::text, '{}')
		FROM map_cells
		WHERE region_id = $1 AND UPPER(cell_type) = 'INHERITANCE_ALTAR' AND content_id = $2 AND is_active = TRUE
		LIMIT 1
	`, currentRegion, altarID).Scan(&cellX, &cellY, &metaRaw)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Đài Truyền Thừa trong khu vực này"})
		return
	}

	var meta AltarMetadata
	_ = json.Unmarshal([]byte(metaRaw), &meta)
	if meta.Tier == "" {
		meta.Tier = "MORTAL"
	}
	meta.Tier = strings.ToUpper(meta.Tier)

	// Check guardian
	guardianDefeated := true
	if meta.GuardianCoord != nil {
		cleared, _, _ := cache.IsCellCleared(currentRegion, meta.GuardianCoord.X, meta.GuardianCoord.Y)
		if !cleared {
			var lastCleared sql.NullTime
			var respawnSec int
			if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT last_cleared_at, COALESCE(respawn_seconds, 300) FROM map_cells WHERE region_id = $1 AND x = $2 AND y = $3`, currentRegion, meta.GuardianCoord.X, meta.GuardianCoord.Y).Scan(&lastCleared, &respawnSec); err == nil && lastCleared.Valid {
				if time.Since(lastCleared.Time) >= time.Duration(respawnSec)*time.Second {
					guardianDefeated = false
				}
			} else {
				guardianDefeated = false
			}
		}
	} else if meta.GuardianSpawnID != "" {
		var gx, gy int
		var lastCleared sql.NullTime
		var respawnSec int
		if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT x, y, last_cleared_at, COALESCE(respawn_seconds, 300) FROM map_cells WHERE region_id = $1 AND (content_id = $2 OR metadata->>'spawn_id' = $2) LIMIT 1`, currentRegion, meta.GuardianSpawnID).Scan(&gx, &gy, &lastCleared, &respawnSec); err == nil {
			cleared, _, _ := cache.IsCellCleared(currentRegion, gx, gy)
			if !cleared {
				if !lastCleared.Valid || time.Since(lastCleared.Time) >= time.Duration(respawnSec)*time.Second {
					guardianDefeated = false
				}
			}
		}
	}

	// Check claim history
	var alreadyClaimed bool
	var lastClaimedAt time.Time
	var claimTier string
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT claimed_at, tier FROM player_inheritance_claims WHERE player_id = $1 AND altar_id = $2 ORDER BY claimed_at DESC LIMIT 1`, playerID, altarID).Scan(&lastClaimedAt, &claimTier); err == nil {
		if meta.Tier == "IMMORTAL" || strings.ToUpper(claimTier) == "IMMORTAL" || meta.CooldownHours <= 0 {
			alreadyClaimed = true
		} else {
			if time.Since(lastClaimedAt) < time.Duration(meta.CooldownHours)*time.Hour {
				alreadyClaimed = true
			}
		}
	}

	dx := px - cellX
	if dx < 0 {
		dx = -dx
	}
	dy := py - cellY
	if dy < 0 {
		dy = -dy
	}
	inRange := dx <= 2 && dy <= 2

	c.JSON(http.StatusOK, gin.H{
		"altar_id":            altarID,
		"altar_name":          meta.AltarName,
		"path":                meta.Path,
		"tier":                meta.Tier,
		"x":                   cellX,
		"y":                   cellY,
		"in_range":            inRange,
		"req_rank":            meta.ReqRank,
		"req_dao_marks":       meta.ReqDaoMarks,
		"required_item":       meta.RequiredItem,
		"required_item_count": meta.RequiredItemCount,
		"guardian_defeated":   guardianDefeated,
		"already_claimed":     alreadyClaimed,
		"legacy_text":         meta.Rewards.LegacyText,
		"rewards": gin.H{
			"gus":             meta.Rewards.Gus,
			"recipes":         meta.Rewards.Recipes,
			"primeval_stones": meta.Rewards.PrimevalStones,
			"materials":       meta.Rewards.Materials,
		},
	})
}

// ClaimInheritanceAltar handles step 3 of the inheritance trial: validating conditions and granting rewards.
func ClaimInheritanceAltar(c *gin.Context) {
	playerIDRaw, ok := c.Get("player_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa xác thực"})
		return
	}
	playerID, ok := playerIDRaw.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Mã người chơi không hợp lệ"})
		return
	}

	var req ClaimAltarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}
	req.AltarID = strings.TrimSpace(req.AltarID)
	if req.AltarID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã Đài Truyền Thừa (altar_id)"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khởi tạo giao dịch"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải dữ liệu người chơi"})
		return
	}

	var cellX, cellY int
	var metaRaw string
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT x, y, COALESCE(metadata::text, '{}')
		FROM map_cells
		WHERE region_id = $1 AND UPPER(cell_type) = 'INHERITANCE_ALTAR' AND content_id = $2 AND is_active = TRUE
		LIMIT 1
	`, player.CurrentRegion, req.AltarID).Scan(&cellX, &cellY, &metaRaw)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Đài Truyền Thừa trong khu vực hiện tại"})
		return
	}

	dx := player.X - cellX
	if dx < 0 {
		dx = -dx
	}
	dy := player.Y - cellY
	if dy < 0 {
		dy = -dy
	}
	if dx > 2 || dy > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu cần đứng gần Đài Truyền Thừa để tương tác"})
		return
	}

	var meta AltarMetadata
	if err := json.Unmarshal([]byte(metaRaw), &meta); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cấu hình Đài Truyền Thừa không hợp lệ"})
		return
	}
	if meta.Tier == "" {
		meta.Tier = "MORTAL"
	}
	meta.Tier = strings.ToUpper(meta.Tier)

	// 1. Guardian check
	if meta.GuardianCoord != nil {
		cleared, _, _ := cache.IsCellCleared(player.CurrentRegion, meta.GuardianCoord.X, meta.GuardianCoord.Y)
		if !cleared {
			var lastCleared sql.NullTime
			var respawnSec int
			if err := tx.QueryRowContext(c.Request.Context(), `SELECT last_cleared_at, COALESCE(respawn_seconds, 300) FROM map_cells WHERE region_id = $1 AND x = $2 AND y = $3`, player.CurrentRegion, meta.GuardianCoord.X, meta.GuardianCoord.Y).Scan(&lastCleared, &respawnSec); err != nil || !lastCleared.Valid || time.Since(lastCleared.Time) >= time.Duration(respawnSec)*time.Second {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Thủ hộ giả của truyền thừa vẫn còn hiện diện, cơ quan truyền thừa chưa thể mở!"})
				return
			}
		}
	} else if meta.GuardianSpawnID != "" {
		var gx, gy int
		var lastCleared sql.NullTime
		var respawnSec int
		if err := tx.QueryRowContext(c.Request.Context(), `SELECT x, y, last_cleared_at, COALESCE(respawn_seconds, 300) FROM map_cells WHERE region_id = $1 AND (content_id = $2 OR metadata->>'spawn_id' = $2) LIMIT 1`, player.CurrentRegion, meta.GuardianSpawnID).Scan(&gx, &gy, &lastCleared, &respawnSec); err == nil {
			cleared, _, _ := cache.IsCellCleared(player.CurrentRegion, gx, gy)
			if !cleared {
				if !lastCleared.Valid || time.Since(lastCleared.Time) >= time.Duration(respawnSec)*time.Second {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Thủ hộ giả của truyền thừa vẫn còn hiện diện, cơ quan truyền thừa chưa thể mở!"})
					return
				}
			}
		}
	}

	// 2. Rank requirement check
	if meta.ReqRank > 0 && player.Rank < meta.ReqRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cảnh giới chưa đủ (yêu cầu chuyển %d), không thể tiếp nhận khảo nghiệm!", meta.ReqRank)})
		return
	}

	// 3. Dao Marks requirement check
	if len(meta.ReqDaoMarks) > 0 {
		for pName, minMarks := range meta.ReqDaoMarks {
			var currentMarks int64
			if player.DaoMarks != nil {
				for k, v := range player.DaoMarks {
					if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(pName)) {
						currentMarks = v
						break
					}
				}
			}
			if currentMarks < minMarks {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đạo Ngân lưu phái [%s] không đủ (yêu cầu %d, hiện có %d), không được truyền thừa thừa nhận!", pName, minMarks, currentMarks)})
				return
			}
		}
	}

	// 4. Token / Required item check & deduct
	if meta.RequiredItem != "" {
		reqCount := meta.RequiredItemCount
		if reqCount <= 0 {
			reqCount = 1
		}
		if InventorySvc != nil {
			qty, cat, err := InventorySvc.GetQuantityInAnyCategory(tx, playerID.String(), meta.RequiredItem, "materials", "consumables", "treasures", "other")
			if err != nil || qty < reqCount {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Thiếu tín vật truyền thừa: %s (cần %d)", meta.RequiredItem, reqCount)})
				return
			}
			if err := InventorySvc.RemoveItem(tx, playerID.String(), meta.RequiredItem, cat, reqCount); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trừ tín vật truyền thừa"})
				return
			}
		}
	}

	// 5. Claim history & immortal unique check
	var lastClaimedAt time.Time
	var lastTier string
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT claimed_at, tier
		FROM player_inheritance_claims
		WHERE player_id = $1 AND altar_id = $2
		ORDER BY claimed_at DESC
		LIMIT 1
	`, playerID, req.AltarID).Scan(&lastClaimedAt, &lastTier)

	if err == nil {
		if meta.Tier == "IMMORTAL" || strings.ToUpper(lastTier) == "IMMORTAL" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã lĩnh nhận toàn bộ chân truyền tại nơi này, không thể nhận thêm!"})
			return
		}
		if meta.CooldownHours > 0 {
			cd := time.Duration(meta.CooldownHours) * time.Hour
			if time.Since(lastClaimedAt) < cd {
				rem := cd - time.Since(lastClaimedAt)
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đài truyền thừa đang trong thời gian tích lũy linh khí, vui lòng quay lại sau %s", formatCooldownDuration(rem))})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã hoàn thành khảo nghiệm tại nơi này!"})
			return
		}
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra lịch sử truyền thừa"})
		return
	}

	// 6. Grant Rewards
	// 6a. Gus
	beforeGus := append(models.GuInventory(nil), player.GuInventory...)
	grantedGus := make([]string, 0, len(meta.Rewards.Gus))
	for _, guID := range meta.Rewards.Gus {
		guID = strings.TrimSpace(guID)
		if guID == "" {
			continue
		}
		spec, ok := models.GetGuAny(guID)
		if !ok {
			logger.S.Warnf("ClaimInheritanceAltar: gu %s not found in registry", guID)
			continue
		}
		player.GuInventory = append(player.GuInventory, models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       spec.ID,
			Rank:       spec.Rank,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
		})
		grantedGus = append(grantedGus, spec.Name)
	}
	if len(grantedGus) > 0 {
		if err := db.SyncPlayerGus(tx, playerID.String(), beforeGus, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cấp phát Cổ trùng truyền thừa"})
			return
		}
	}

	// 6b. Recipes
	if RecipeSvc != nil {
		for _, recipeID := range meta.Rewards.Recipes {
			recipeID = strings.TrimSpace(recipeID)
			if recipeID == "" {
				continue
			}
			_ = RecipeSvc.Upsert(tx, &services.RecipeProgress{
				PlayerID:       playerID.String(),
				RecipeID:       recipeID,
				CompletionRate: 100,
				IsUnlocked:     true,
			})
		}
	}

	// 6c. Primeval Stones
	if meta.Rewards.PrimevalStones > 0 {
		player.PrimevalStones += float64(meta.Rewards.PrimevalStones)
		if _, err := tx.ExecContext(c.Request.Context(), `UPDATE players SET primeval_stones = $1, version = version + 1 WHERE id = $2`, player.PrimevalStones, playerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cấp phát nguyên thạch truyền thừa"})
			return
		}
	}

	// 6d. Materials
	if InventorySvc != nil {
		for matID, count := range meta.Rewards.Materials {
			if count <= 0 {
				continue
			}
			cat := "materials"
			if itemSpec, ok := models.ItemRegistry[matID]; ok && itemSpec.ItemType != "" {
				cat = strings.ToLower(itemSpec.ItemType)
			}
			if err := InventorySvc.AddItem(tx, playerID.String(), matID, cat, count); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cấp phát nguyên liệu truyền thừa"})
				return
			}
		}
	}

	// 6e. Record claim in player_inheritance_claims
	var mapInstanceID sql.NullString
	_ = tx.QueryRowContext(c.Request.Context(), `SELECT map_instance_id::text FROM players WHERE id = $1`, playerID).Scan(&mapInstanceID)

	templateID := ""
	if mapInstanceID.Valid && mapInstanceID.String != "" {
		_ = tx.QueryRowContext(c.Request.Context(), `SELECT template_id FROM shared_spatial_instances WHERE id = $1`, mapInstanceID.String).Scan(&templateID)
	}
	if templateID == "" {
		_ = tx.QueryRowContext(c.Request.Context(), `SELECT id FROM instance_templates WHERE spatial_region_id = $1 LIMIT 1`, player.CurrentRegion).Scan(&templateID)
	}
	rewardsJSON, _ := json.Marshal(meta.Rewards)
	if _, err := tx.ExecContext(c.Request.Context(), `
		INSERT INTO player_inheritance_claims (player_id, altar_id, template_id, claimed_at, tier, rewards_snapshot)
		VALUES ($1, $2, $3, NOW(), $4, $5)
	`, playerID, req.AltarID, templateID, meta.Tier, rewardsJSON); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu hồ sơ khảo nghiệm truyền thừa"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất tiếp nhận truyền thừa"})
		return
	}

	if meta.Tier == "IMMORTAL" {
		_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Chúc mừng đạo hữu %s đã thông qua khảo nghiệm, đắc được Chân Truyền [%s]!", player.Username, meta.AltarName))
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:   &playerID,
		Action:     "inheritance.claim_altar",
		EntityType: "altar",
		EntityID:   req.AltarID,
		Metadata: map[string]interface{}{
			"altar_name": meta.AltarName,
			"tier":       meta.Tier,
			"gus":        meta.Rewards.Gus,
			"recipes":    meta.Rewards.Recipes,
			"stones":     meta.Rewards.PrimevalStones,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"claimed":     true,
		"altar_id":    req.AltarID,
		"altar_name":  meta.AltarName,
		"tier":        meta.Tier,
		"legacy_text": meta.Rewards.LegacyText,
		"rewards": gin.H{
			"gus":             meta.Rewards.Gus,
			"recipes":         meta.Rewards.Recipes,
			"primeval_stones": meta.Rewards.PrimevalStones,
			"materials":       meta.Rewards.Materials,
		},
	})
}

func formatCooldownDuration(d time.Duration) string {
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%d giờ %d phút", hours, mins)
	}
	return fmt.Sprintf("%d phút", mins)
}
