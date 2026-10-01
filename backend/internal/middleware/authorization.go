package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/authz"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

const adminPrincipalKey = "admin_principal"

type PrincipalResolver interface {
	Resolve(context.Context, uuid.UUID) (authz.AdminPrincipal, error)
}

func RequirePermission(permissionKey string) gin.HandlerFunc {
	return requirePermission(authz.NewResolver(db.DB), permissionKey)
}

func requirePermission(resolver PrincipalResolver, permissionKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		playerID, ok := GetPlayerID(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}
		if resolver == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Authorization service unavailable"})
			return
		}

		principal, err := resolver.Resolve(c.Request.Context(), playerID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "Authorization service unavailable"})
			return
		}
		if !principal.HasPermission(permissionKey) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}

		c.Set(adminPrincipalKey, principal)
		c.Request = c.Request.WithContext(authz.WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}

// GetAdminPrincipal reads the principal attached by RequirePermission.
func GetAdminPrincipal(c *gin.Context) (authz.AdminPrincipal, bool) {
	if value, exists := c.Get(adminPrincipalKey); exists {
		if principal, ok := value.(authz.AdminPrincipal); ok {
			return principal, true
		}
	}
	return authz.PrincipalFromContext(c.Request.Context())
}
