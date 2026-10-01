package authz

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
)

const (
	SuperAdminRole        = "super_admin"
	AdminAccessPermission = "admin.dashboard.access"
)

// AdminPrincipal is the database-resolved authorization identity for an
// authenticated player. Roles and permissions are intentionally not sourced
// from JWT claims so authorization changes take effect on the next request.
type AdminPrincipal struct {
	PlayerID            uuid.UUID
	Roles               []string
	Permissions         map[string]struct{}
	LegacyAdminFallback bool
}

func (p AdminPrincipal) HasPermission(permissionKey string) bool {
	if _, ok := p.Permissions["admin.*"]; ok {
		return true
	}
	_, ok := p.Permissions[permissionKey]
	return ok
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// Resolver resolves assigned roles and their effective permissions from the
// database. A legacy is_admin account receives a synthetic super_admin role
// only when no role assignment exists.
type Resolver struct {
	db queryer
}

func NewResolver(database *sql.DB) *Resolver {
	if database == nil {
		return &Resolver{}
	}
	return &Resolver{db: database}
}

func (r *Resolver) Resolve(ctx context.Context, playerID uuid.UUID) (AdminPrincipal, error) {
	principal := AdminPrincipal{
		PlayerID:    playerID,
		Permissions: make(map[string]struct{}),
	}
	if r == nil || r.db == nil {
		return principal, errors.New("authorization database unavailable")
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT COALESCE(p.is_admin, false),
		       COALESCE(par.role_key, ''),
		       COALESCE(arp.permission_key, '')
		FROM players p
		LEFT JOIN player_admin_roles par ON par.player_id = p.id
		LEFT JOIN admin_role_permissions arp ON arp.role_key = par.role_key
		WHERE p.id = $1
		ORDER BY par.role_key, arp.permission_key`, playerID)
	if err != nil {
		return principal, err
	}
	defer rows.Close()

	roles := make(map[string]struct{})
	isAdmin := false
	foundPlayer := false
	for rows.Next() {
		foundPlayer = true
		var roleKey, permissionKey string
		if err := rows.Scan(&isAdmin, &roleKey, &permissionKey); err != nil {
			return principal, err
		}
		if roleKey != "" {
			if _, exists := roles[roleKey]; !exists {
				roles[roleKey] = struct{}{}
				principal.Roles = append(principal.Roles, roleKey)
			}
		}
		if permissionKey != "" {
			principal.Permissions[permissionKey] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return principal, err
	}

	if applyLegacyAdminFallback(&principal, foundPlayer, isAdmin) {
		monitor.GlobalAdminOperations.RecordLegacyAdminFallback()
		logger.L.Warn("authorization_legacy_is_admin_fallback", zap.String("player_id", playerID.String()), zap.String("remediation", "backfill super_admin role"))
	}

	return principal, nil
}

func applyLegacyAdminFallback(principal *AdminPrincipal, foundPlayer, isAdmin bool) bool {
	if !foundPlayer || !isAdmin || len(principal.Roles) != 0 {
		return false
	}
	principal.Roles = []string{SuperAdminRole}
	principal.Permissions[AdminAccessPermission] = struct{}{}
	principal.Permissions["admin.*"] = struct{}{}
	principal.LegacyAdminFallback = true
	return true
}
