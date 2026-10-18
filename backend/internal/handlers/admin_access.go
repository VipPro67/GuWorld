package handlers

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/admincatalog"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// Bump this marker whenever the legacy permission grant or resource catalog
// changes so clients can invalidate their cached access snapshot.
const legacyAdminPermissionVersion = "admin-catalog-v6"

// AdminGetMe returns the effective access snapshot for the authenticated admin.
// Until persisted RBAC is introduced, is_admin actors use the explicit legacy
// role and permission set below.
func AdminGetMe(c *gin.Context) {
	actorID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var username string
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT username FROM players WHERE id=$1`, actorID).Scan(&username); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin admin"})
		return
	}

	principal, ok := middleware.GetAdminPrincipal(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	permissions := make([]string, 0, len(principal.Permissions))
	for permission := range principal.Permissions {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)

	c.JSON(http.StatusOK, gin.H{
		"actor_id":              actorID,
		"username":              username,
		"roles":                 principal.Roles,
		"effective_permissions": permissions,
		"legacy_fallback":       principal.LegacyAdminFallback,
		"permission_version":    legacyAdminPermissionVersion,
	})
}

// AdminGetResources returns only application module metadata visible to the
// actor. The current legacy admin fallback grants every catalog permission.
func AdminGetResources(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	principal, ok := middleware.GetAdminPrincipal(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"resources":          admincatalog.Visible(principal.HasPermission),
		"permission_version": legacyAdminPermissionVersion,
	})
}
