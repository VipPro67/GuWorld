package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

type adminRole struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Level       int    `json:"level"`
}

type adminUpdateRolesRequest struct {
	Roles *[]string `json:"roles" binding:"required"`
}

var errInvalidAdminRoleRequest = errors.New("invalid admin role request")

func AdminListRoles(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	roles, err := queryAdminRoles(db.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách vai trò"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

func AdminGetPlayerRoles(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM players WHERE id=$1)`, playerID).Scan(&exists); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}
	roles, err := queryPlayerAdminRoles(db.DB, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải vai trò người chơi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"player_id": playerID, "roles": roles})
}

func AdminUpdatePlayerRoles(c *gin.Context) {
	actorID, ok := requireAdmin(c)
	if !ok {
		return
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	var req adminUpdateRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu vai trò không hợp lệ"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	// Serialize all role writes. This makes the last-super-admin count safe
	// even when two different super admins are demoted concurrently.
	if _, err = tx.ExecContext(c.Request.Context(), `SELECT pg_advisory_xact_lock(782014621)`); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khóa cập nhật vai trò"})
		return
	}
	if err = lockRolePlayers(tx, actorID, targetID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		}
		return
	}

	catalog, err := queryAdminRoles(tx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách vai trò"})
		return
	}
	actorRoles, err := queryPlayerAdminRoles(tx, actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra quyền quản trị"})
		return
	}
	before, err := queryPlayerAdminRoles(tx, targetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải vai trò người chơi"})
		return
	}
	after, validationErr := validateAdminRoleChange(actorID, targetID, actorRoles, before, *req.Roles, catalog)
	if validationErr != nil {
		status := http.StatusForbidden
		if errors.Is(validationErr, errInvalidAdminRoleRequest) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": validationErr.Error()})
		return
	}

	if containsRole(before, "super_admin") && !containsRole(after, "super_admin") {
		var count int
		if err := tx.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM player_admin_roles WHERE role_key='super_admin'`).Scan(&count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra super admin"})
			return
		}
		if wouldRemoveLastSuperAdmin(before, after, count) {
			c.JSON(http.StatusConflict, gin.H{"error": "Không thể xóa super admin cuối cùng"})
			return
		}
	}

	if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM player_admin_roles WHERE player_id=$1`, targetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật vai trò"})
		return
	}
	for _, role := range after {
		if _, err := tx.ExecContext(c.Request.Context(), `INSERT INTO player_admin_roles (player_id, role_key, assigned_by) VALUES ($1,$2,$3)`, targetID, role.Key, actorID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật vai trò"})
			return
		}
	}
	if _, err := tx.ExecContext(c.Request.Context(), `UPDATE players SET is_admin=$2 WHERE id=$1`, targetID, len(after) > 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đồng bộ quyền quản trị"})
		return
	}
	if err := insertRoleAudit(tx, actorID, targetID, roleKeys(before), roleKeys(after), c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi audit vai trò"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất cập nhật vai trò"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"player_id": targetID, "roles": after})
}

func wouldRemoveLastSuperAdmin(before, after []adminRole, currentSuperAdmins int) bool {
	return containsRole(before, "super_admin") && !containsRole(after, "super_admin") && currentSuperAdmins <= 1
}

type roleQuerier interface {
	Query(string, ...interface{}) (*sql.Rows, error)
}

func queryAdminRoles(q roleQuerier) ([]adminRole, error) {
	rows, err := q.Query(`SELECT key, name, description, level FROM admin_roles ORDER BY level, key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := []adminRole{}
	for rows.Next() {
		var role adminRole
		if err := rows.Scan(&role.Key, &role.Name, &role.Description, &role.Level); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func queryPlayerAdminRoles(q roleQuerier, playerID uuid.UUID) ([]adminRole, error) {
	rows, err := q.Query(`SELECT r.key, r.name, r.description, r.level FROM player_admin_roles pr JOIN admin_roles r ON r.key=pr.role_key WHERE pr.player_id=$1 ORDER BY r.level, r.key`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := []adminRole{}
	for rows.Next() {
		var role adminRole
		if err := rows.Scan(&role.Key, &role.Name, &role.Description, &role.Level); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func lockRolePlayers(tx *sql.Tx, actorID, targetID uuid.UUID) error {
	ids := []string{actorID.String(), targetID.String()}
	sort.Strings(ids)
	rows, err := tx.Query(`SELECT id::text FROM players WHERE id::text = ANY($1) ORDER BY id FOR UPDATE`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		seen[id] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !seen[actorID.String()] || !seen[targetID.String()] {
		return sql.ErrNoRows
	}
	return nil
}

func validateAdminRoleChange(actorID, targetID uuid.UUID, actorRoles, before []adminRole, requested []string, catalog []adminRole) ([]adminRole, error) {
	byKey := make(map[string]adminRole, len(catalog))
	for _, role := range catalog {
		byKey[role.Key] = role
	}
	seen := make(map[string]bool, len(requested))
	after := make([]adminRole, 0, len(requested))
	for _, key := range requested {
		key = strings.TrimSpace(key)
		role, ok := byKey[key]
		if !ok {
			return nil, fmt.Errorf("%w: vai trò không tồn tại: %s", errInvalidAdminRoleRequest, key)
		}
		if seen[key] {
			return nil, fmt.Errorf("%w: vai trò bị trùng: %s", errInvalidAdminRoleRequest, key)
		}
		seen[key] = true
		after = append(after, role)
	}
	sort.Slice(after, func(i, j int) bool {
		if after[i].Level == after[j].Level {
			return after[i].Key < after[j].Key
		}
		return after[i].Level < after[j].Level
	})

	actorMax, actorSuper := 0, false
	for _, role := range actorRoles {
		if role.Level > actorMax {
			actorMax = role.Level
		}
		actorSuper = actorSuper || role.Key == "super_admin"
	}
	beforeSet := map[string]bool{}
	for _, role := range before {
		beforeSet[role.Key] = true
	}
	if actorID == targetID {
		for _, role := range after {
			if !beforeSet[role.Key] {
				return nil, errors.New("Không thể tự nâng quyền")
			}
		}
	}
	if !actorSuper {
		for _, role := range after {
			if role.Level > actorMax {
				return nil, errors.New("Không thể cấp vai trò cao hơn quyền của bạn")
			}
		}
		// A lower-level admin must not be able to demote a higher-level account.
		for _, role := range before {
			if role.Level > actorMax && !seen[role.Key] {
				return nil, errors.New("Không thể thay đổi vai trò cao hơn quyền của bạn")
			}
		}
	}
	return after, nil
}

func containsRole(roles []adminRole, key string) bool {
	for _, role := range roles {
		if role.Key == key {
			return true
		}
	}
	return false
}
func roleKeys(roles []adminRole) []string {
	keys := make([]string, len(roles))
	for i, role := range roles {
		keys[i] = role.Key
	}
	return keys
}

func insertRoleAudit(tx *sql.Tx, actorID, targetID uuid.UUID, before, after []string, ip string) error {
	metadata, err := json.Marshal(gin.H{"before": before, "after": after})
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO audit_logs (player_id, action, entity_type, entity_id, request_summary, metadata, ip_address, success, error_message, created_at) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10)`, actorID, "admin.update_player_roles", "player", targetID.String(), "roles: "+strings.Join(before, ",")+" -> "+strings.Join(after, ","), string(metadata), ip, true, "", time.Now().UTC())
	return err
}
