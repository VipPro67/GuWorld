package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// GetAperturePortal returns the active world portal for the host's aperture.
func GetAperturePortal(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var portal models.AperturePortal
	err := db.DB.QueryRowContext(c.Request.Context(), `
		SELECT p.id, p.aperture_id, p.owner_id, p.region_id, p.x, p.y, p.name, p.access_policy,
		       p.concealment_power, p.is_active, p.created_at, p.updated_at
		FROM aperture_portals p
		JOIN immortal_apertures a ON a.id = p.aperture_id
		WHERE a.player_id = $1 AND p.is_active = true
	`, playerID).Scan(&portal.ID, &portal.ApertureID, &portal.OwnerID, &portal.RegionID, &portal.X, &portal.Y,
		&portal.Name, &portal.AccessPolicy, &portal.ConcealmentPower, &portal.IsActive, &portal.CreatedAt, &portal.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"portal": nil, "message": "Chưa hạ khiếu hoặc môn hộ chưa mở"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn Môn Hộ: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"portal": portal})
}

// ManageAperturePortal updates the policy and name of the host's world portal.
func ManageAperturePortal(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Name         string `json:"name"`
		AccessPolicy string `json:"access_policy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	validPolicies := map[string]bool{"OPEN": true, "PARTY": true, "SECT": true, "KEY": true, "APPROVAL": true, "HIDDEN": true}
	if !validPolicies[req.AccessPolicy] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chính sách ra vào không hợp lệ"})
		return
	}

	res, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE aperture_portals p
		SET name = COALESCE(NULLIF($1, ''), p.name),
		    access_policy = $2,
		    updated_at = NOW()
		FROM immortal_apertures a
		WHERE p.aperture_id = a.id AND a.player_id = $3 AND p.is_active = true
	`, req.Name, req.AccessPolicy, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Môn Hộ: " + err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy Môn Hộ đang hoạt động"})
		return
	}

	var regionID string
	var px, py float64
	var apID uuid.UUID
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT p.region_id, p.x, p.y, p.aperture_id
		FROM aperture_portals p
		JOIN immortal_apertures a ON a.id = p.aperture_id
		WHERE a.player_id = $1 AND p.is_active = true
	`, playerID).Scan(&regionID, &px, &py, &apID)
	if regionID != "" {
		_ = cache.PublishEventQuick("world:cell", map[string]any{
			"type":       "portal_update",
			"region_id":  regionID,
			"x":          int(px),
			"y":          int(py),
			"cell_type":  "AperturePortal",
			"content_id": apID.String(),
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật Môn Hộ thành công", "access_policy": req.AccessPolicy})
}

// EnterAperturePortal allows a guest to enter an active world gateway into the host's aperture.
func EnterAperturePortal(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	portalID := c.Param("portalId")
	portalUUID, err := uuid.Parse(portalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã Môn Hộ không hợp lệ"})
		return
	}

	var inCombatSession bool
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM players
			WHERE id = $1 AND active_combat IS NOT NULL
		)
	`, playerID).Scan(&inCombatSession)
	if inCombatSession {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể bước vào Môn Hộ khi đang trong trận chiến thời gian thực."})
		return
	}

	handedOff, handoffErr := middleware.BeginRealtimeHandoff(playerID)
	if handoffErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Không thể tạm dừng điều khiển thời gian thực, vui lòng thử lại."})
		return
	}
	var realtimeFinished bool
	if handedOff {
		defer func() {
			if !realtimeFinished {
				middleware.AbortRealtimeHandoff(playerID)
			}
		}()
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch Guest Player
	var guest models.Player
	var guestSectID sql.NullString
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT p.id, p.rank, p.username, COALESCE(p.current_region, ''), COALESCE(p.x, 500), COALESCE(p.y, 500), sm.sect_id
		FROM players p
		LEFT JOIN sect_members sm ON sm.player_id = p.id
		WHERE p.id = $1 FOR UPDATE OF p
	`, playerID).Scan(&guest.ID, &guest.Rank, &guest.Username, &guest.CurrentRegion, &guest.X, &guest.Y, &guestSectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cổ sư không tồn tại"})
		return
	}

	if liveRegion, liveX, liveY, ok := middleware.RealtimePlayerLocation(playerID); ok && liveRegion != "" {
		guest.CurrentRegion = liveRegion
		guest.X = liveX
		guest.Y = liveY
	}

	if guest.CurrentRegion == "aperture_interior" || strings.HasPrefix(guest.CurrentRegion, "aperture:") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã đang ở bên trong một Tiên Khiếu."})
		return
	}

	// 2. Fetch Portal & Host Aperture
	var portal models.AperturePortal
	var hostRank int
	var hostPartyID sql.NullString
	var hostSectID sql.NullString
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT p.id, p.aperture_id, p.owner_id, p.region_id, p.x, p.y, p.name, p.access_policy, p.is_active,
		       h.rank, h.party_id, sm.sect_id
		FROM aperture_portals p
		JOIN players h ON h.id = p.owner_id
		LEFT JOIN sect_members sm ON sm.player_id = h.id
		WHERE p.id = $1 AND p.is_active = true
	`, portalUUID).Scan(&portal.ID, &portal.ApertureID, &portal.OwnerID, &portal.RegionID, &portal.X, &portal.Y,
		&portal.Name, &portal.AccessPolicy, &portal.IsActive, &hostRank, &hostPartyID, &hostSectID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Môn Hộ không tồn tại hoặc đã khép lại"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn Môn Hộ: " + err.Error()})
		return
	}

	if portal.OwnerID == guest.ID.String() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chủ Khiếu không thể bước vào Môn Hộ dưới tư cách khách."})
		return
	}

	// 3. Proximity check: Guest must be in the same region and within 3 cells of the portal
	if guest.CurrentRegion != portal.RegionID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không ở cùng khu vực với Môn Hộ."})
		return
	}
	dist := math.Hypot(float64(guest.X)-portal.X, float64(guest.Y)-portal.Y)
	if dist > 3.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu ở quá xa Môn Hộ (cần đứng trong phạm vi 3 bước)."})
		return
	}

	// 4. Access Policy Enforcement
	switch portal.AccessPolicy {
	case "PARTY":
		if !hostPartyID.Valid || hostPartyID.String == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Chủ Khiếu không trong tổ đội nào"})
			return
		}
		var guestPartyID sql.NullString
		_ = tx.QueryRowContext(c.Request.Context(), `SELECT party_id FROM players WHERE id = $1`, playerID).Scan(&guestPartyID)
		if !guestPartyID.Valid || guestPartyID.String != hostPartyID.String {
			c.JSON(http.StatusForbidden, gin.H{"error": "Môn Hộ chỉ mở cho thành viên Tổ Đội của Chủ Khiếu"})
			return
		}
	case "SECT":
		if !hostSectID.Valid || hostSectID.String == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Chủ Khiếu không có Tông Môn"})
			return
		}
		if !guestSectID.Valid || guestSectID.String != hostSectID.String {
			c.JSON(http.StatusForbidden, gin.H{"error": "Môn Hộ chỉ mở cho đồng môn cùng Tông Môn"})
			return
		}
	}

	// 5. Add to aperture_guest_roster
	_, err = tx.ExecContext(c.Request.Context(), `
		INSERT INTO aperture_guest_roster (aperture_id, player_id, role, granted_by, entered_at)
		VALUES ($1, $2, 'VISITOR', $3, NOW())
		ON CONFLICT (aperture_id, player_id) DO UPDATE
		SET role = 'VISITOR', entered_at = NOW()
	`, portal.ApertureID, guest.ID, portal.OwnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi ghi nhận danh sách khách: " + err.Error()})
		return
	}

	// 6. Transition guest into the aperture instance
	apertureInstanceID := fmt.Sprintf("aperture:%s", portal.ApertureID)
	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET current_region = 'aperture_interior', map_instance_id = $1, x = 10, y = 10,
		    world_zone_id = 'aperture_interior', world_instance_id = $2,
		    world_x = 10, world_y = 10, world_safe_x = 10, world_safe_y = 10,
		    world_location_revision = world_location_revision + 1, version = version + 1
		WHERE id = $3
	`, portal.ApertureID, apertureInstanceID, guest.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi chuyển dịch vào Tiên Khiếu: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hoàn tất bước vào Môn Hộ"})
		return
	}

	if handedOff {
		middleware.FinishRealtimeHandoff(playerID)
		realtimeFinished = true
	}
	portalApUUID, _ := uuid.Parse(portal.ApertureID)
	syncSpatialPlayer(playerID, "aperture_interior", 10, 10, portalApUUID)

	c.JSON(http.StatusOK, gin.H{
		"message":           fmt.Sprintf("Đạo hữu đã bước qua Môn Hộ tiến vào %s an toàn.", portal.Name),
		"aperture_region":   "aperture_interior",
		"aperture_instance": apertureInstanceID,
		"x":                 10,
		"y":                 10,
		"role":              "VISITOR",
	})
}

// LeaveAperture allows a guest to exit the aperture back to the world gateway location.
// Hosts cannot leave while anchored; they must invoke UnanchorAperture instead.
func LeaveAperture(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var inCombatSession bool
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM players
			WHERE id = $1 AND active_combat IS NOT NULL
		)
	`, playerID).Scan(&inCombatSession)
	if inCombatSession {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể rời khỏi Tiên Khiếu khi đang trong trận chiến thời gian thực."})
		return
	}

	handedOff, handoffErr := middleware.BeginRealtimeHandoff(playerID)
	if handoffErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Không thể tạm dừng điều khiển thời gian thực, vui lòng thử lại."})
		return
	}
	var realtimeFinished bool
	if handedOff {
		defer func() {
			if !realtimeFinished {
				middleware.AbortRealtimeHandoff(playerID)
			}
		}()
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var currentRegion string
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT COALESCE(current_region, '')
		FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&currentRegion)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cổ sư không tồn tại"})
		return
	}

	if liveRegion, _, _, ok := middleware.RealtimePlayerLocation(playerID); ok && liveRegion != "" {
		currentRegion = liveRegion
	}

	if currentRegion != "aperture_interior" && !strings.HasPrefix(currentRegion, "aperture:") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu hiện không ở bên trong Tiên Khiếu."})
		return
	}

	var apertureUUID uuid.UUID
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT aperture_id FROM aperture_guest_roster WHERE player_id = $1
	`, playerID).Scan(&apertureUUID)
	if err == sql.ErrNoRows {
		var mapInst sql.NullString
		_ = tx.QueryRowContext(c.Request.Context(), `SELECT map_instance_id::text FROM players WHERE id = $1`, playerID).Scan(&mapInst)
		if mapInst.Valid && mapInst.String != "" {
			apertureUUID, _ = uuid.Parse(mapInst.String)
		} else if strings.HasPrefix(currentRegion, "aperture:") {
			apertureUUID, _ = uuid.Parse(strings.TrimPrefix(currentRegion, "aperture:"))
		}
	}
	if apertureUUID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không xác định được Tiên Khiếu đạo hữu đang ở."})
		return
	}

	// Check if caller is Host
	var hostPlayerID uuid.UUID
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT player_id FROM immortal_apertures WHERE id = $1
	`, apertureUUID).Scan(&hostPlayerID)
	if err == nil && hostPlayerID == playerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chủ Khiếu không thể rời Tiên Khiếu bằng Môn Hộ thông thường; đạo hữu cần thực hiện Thu Khiếu (Unanchor) để thu hồi tiểu thế giới."})
		return
	}

	// Fetch outer world gateway location
	var returnRegion string
	var returnX, returnY float64
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT region_id, x, y
		FROM aperture_portals
		WHERE aperture_id = $1
	`, apertureUUID).Scan(&returnRegion, &returnX, &returnY)
	if err != nil {
		returnRegion = "Southern_Border"
		returnX = 500
		returnY = 500
	}

	// Remove from guest roster
	_, _ = tx.ExecContext(c.Request.Context(), `
		DELETE FROM aperture_guest_roster WHERE aperture_id = $1 AND player_id = $2
	`, apertureUUID, playerID)

	// Move guest back to outer world
	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET current_region = $1, map_instance_id = NULL, x = ($2)::int, y = ($3)::int, world_zone_id = $1, world_instance_id = '',
		    world_x = ($2)::double precision, world_y = ($3)::double precision, world_safe_x = ($2)::double precision, world_safe_y = ($3)::double precision,
		    world_location_revision = world_location_revision + 1, version = version + 1
		WHERE id = $4
	`, returnRegion, int(returnX), int(returnY), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đưa nhân vật trở lại thế giới: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu dữ liệu rời khiếu"})
		return
	}

	if handedOff {
		middleware.FinishRealtimeHandoff(playerID)
		realtimeFinished = true
	}
	syncSpatialPlayer(playerID, returnRegion, int(returnX), int(returnY), uuid.Nil)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Đạo hữu đã bước qua Môn Hộ trở lại thế giới thực.",
		"current_region": returnRegion,
		"x":              int(returnX),
		"y":              int(returnY),
	})
}

// GetApertureGuests returns the list of guests currently present inside the host's aperture.
func GetApertureGuests(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT g.aperture_id, g.player_id, p.username, p.rank, g.role, g.granted_by, g.entered_at
		FROM aperture_guest_roster g
		JOIN immortal_apertures a ON a.id = g.aperture_id
		JOIN players p ON p.id = g.player_id
		WHERE a.player_id = $1
		ORDER BY g.entered_at ASC
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn danh sách khách: " + err.Error()})
		return
	}
	defer rows.Close()

	var guests []models.ApertureGuest
	for rows.Next() {
		var g models.ApertureGuest
		if err := rows.Scan(&g.ApertureID, &g.PlayerID, &g.Username, &g.Rank, &g.Role, &g.GrantedBy, &g.EnteredAt); err == nil {
			guests = append(guests, g)
		}
	}

	c.JSON(http.StatusOK, gin.H{"guests": guests})
}

// SetGuestRole updates the role of a guest (VISITOR, ASSISTANT, PRISONER).
func SetGuestRole(c *gin.Context) {
	hostID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	guestIDStr := c.Param("guestId")
	guestUUID, err := uuid.Parse(guestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã khách không hợp lệ"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	validRoles := map[string]bool{"VISITOR": true, "ASSISTANT": true, "PRISONER": true}
	if !validRoles[req.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vai trò không hợp lệ"})
		return
	}

	res, err := db.DB.ExecContext(c.Request.Context(), `
		UPDATE aperture_guest_roster g
		SET role = $1
		FROM immortal_apertures a
		WHERE g.aperture_id = a.id AND a.player_id = $2 AND g.player_id = $3
	`, req.Role, hostID, guestUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi phân quyền khách: " + err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy khách trong Tiên Khiếu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã điều chỉnh vai trò thành công", "role": req.Role})
}

// EvictGuest forcibly removes a guest from the aperture back to the world gateway.
func EvictGuest(c *gin.Context) {
	hostID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	guestIDStr := c.Param("guestId")
	guestUUID, err := uuid.Parse(guestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã khách không hợp lệ"})
		return
	}

	var inCombatSession bool
	_ = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM players
			WHERE id = $1 AND active_combat IS NOT NULL
		)
	`, guestUUID).Scan(&inCombatSession)
	if inCombatSession {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể trục xuất khách khi họ đang trong trận chiến thời gian thực."})
		return
	}

	handedOff, handoffErr := middleware.BeginRealtimeHandoff(guestUUID)
	if handoffErr != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Không thể tạm dừng điều khiển thời gian thực của khách, vui lòng thử lại."})
		return
	}
	var realtimeFinished bool
	if handedOff {
		defer func() {
			if !realtimeFinished {
				middleware.AbortRealtimeHandoff(guestUUID)
			}
		}()
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	// Verify aperture and portal
	var apertureID uuid.UUID
	var returnRegion string
	var returnX, returnY float64
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT a.id, p.region_id, p.x, p.y
		FROM immortal_apertures a
		JOIN aperture_portals p ON p.aperture_id = a.id
		WHERE a.player_id = $1 AND p.is_active = true
	`, hostID).Scan(&apertureID, &returnRegion, &returnX, &returnY)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Khiếu không ở trạng thái Hạ Khiếu"})
		return
	}

	// Delete from guest roster
	res, err := tx.ExecContext(c.Request.Context(), `
		DELETE FROM aperture_guest_roster WHERE aperture_id = $1 AND player_id = $2
	`, apertureID, guestUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xóa danh sách khách"})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Khách không có mặt trong Tiên Khiếu"})
		return
	}

	// Teleport guest outside
	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET current_region = $1, map_instance_id = NULL, x = ($2)::int, y = ($3)::int, world_zone_id = $1, world_instance_id = '',
		    world_x = ($2)::double precision, world_y = ($3)::double precision, world_safe_x = ($2)::double precision, world_safe_y = ($3)::double precision,
		    world_location_revision = world_location_revision + 1, version = version + 1
		WHERE id = $4
	`, returnRegion, int(returnX), int(returnY), guestUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi dịch chuyển khách ra ngoài: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu dữ liệu trục xuất"})
		return
	}

	if handedOff {
		middleware.FinishRealtimeHandoff(guestUUID)
		realtimeFinished = true
	}
	syncSpatialPlayer(guestUUID, returnRegion, int(returnX), int(returnY), uuid.Nil)

	c.JSON(http.StatusOK, gin.H{"message": "Đã trục xuất khách ra ngoài Môn Hộ thế giới thành công."})
}

// SignAllianceContract creates or updates an Information-path agreement between host and guest.
func SignAllianceContract(c *gin.Context) {
	hostID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	guestIDStr := c.Param("guestId")
	guestUUID, err := uuid.Parse(guestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã khách không hợp lệ"})
		return
	}

	var req struct {
		ContractTier string `json:"contract_tier"` // MINOR, MAJOR, ABSOLUTE
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ContractTier == "" {
		req.ContractTier = "MAJOR"
	}

	var apertureID uuid.UUID
	err = db.DB.QueryRowContext(c.Request.Context(), `
		SELECT id FROM immortal_apertures WHERE player_id = $1
	`, hostID).Scan(&apertureID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chưa khai mở Tiên Khiếu"})
		return
	}

	_, err = db.DB.ExecContext(c.Request.Context(), `
		INSERT INTO aperture_alliance_contracts (aperture_id, host_id, guest_id, contract_tier, is_active, created_at)
		VALUES ($1, $2, $3, $4, true, NOW())
	`, apertureID, hostID, guestUUID, req.ContractTier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lập Minh Ước: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã thiết lập Minh Ước Tín Đạo Hộ Đạo thành công", "contract_tier": req.ContractTier})
}
