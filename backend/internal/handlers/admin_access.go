package handlers

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// Bump this marker whenever the legacy permission grant or resource catalog
// changes so clients can invalidate their cached access snapshot.
const legacyAdminPermissionVersion = "legacy-admin-v1"

type adminResource struct {
	Key                 string   `json:"key"`
	Label               string   `json:"label"`
	Category            string   `json:"category"`
	FrontendRoute       string   `json:"frontend_route"`
	Capabilities        []string `json:"capabilities"`
	RequiredPermissions []string `json:"required_permissions"`
	RiskLevel           string   `json:"risk_level"`
}

// adminResourceCatalog is application-owned navigation metadata. It must not
// be populated from database introspection: callers should never learn table,
// column, schema, or SQL details through this endpoint.
var adminResourceCatalog = []adminResource{
	{Key: "players", Label: "Người chơi", Category: "game", FrontendRoute: "/", Capabilities: []string{"read", "update", "manage_inventory"}, RequiredPermissions: []string{"admin.players"}, RiskLevel: "high"},
	{Key: "sects", Label: "Tông môn", Category: "game", FrontendRoute: "/", Capabilities: []string{"read", "create", "update", "delete"}, RequiredPermissions: []string{"admin.sects"}, RiskLevel: "high"},
	{Key: "data", Label: "Dữ liệu", Category: "content", FrontendRoute: "/", Capabilities: []string{"read", "import", "export", "clear_cache"}, RequiredPermissions: []string{"admin.data"}, RiskLevel: "critical"},
	{Key: "logs", Label: "Nhật ký", Category: "system", FrontendRoute: "/", Capabilities: []string{"read"}, RequiredPermissions: []string{"admin.audit_logs"}, RiskLevel: "medium"},
	{Key: "monitoring", Label: "Giám sát", Category: "system", FrontendRoute: "/", Capabilities: []string{"read", "reset"}, RequiredPermissions: []string{"admin.monitoring"}, RiskLevel: "medium"},
	{Key: "shop", Label: "Cửa hàng", Category: "economy", FrontendRoute: "/", Capabilities: []string{"read", "create", "update", "delete"}, RequiredPermissions: []string{"admin.shop"}, RiskLevel: "high"},
	{Key: "mines", Label: "Mỏ", Category: "world", FrontendRoute: "/", Capabilities: []string{"read", "update", "reset"}, RequiredPermissions: []string{"admin.mines"}, RiskLevel: "high"},
	{Key: "gift_codes", Label: "Mã quà", Category: "economy", FrontendRoute: "/", Capabilities: []string{"read", "create"}, RequiredPermissions: []string{"admin.gift_codes"}, RiskLevel: "high"},
	{Key: "apertures", Label: "Tiên khiếu", Category: "game", FrontendRoute: "/", Capabilities: []string{"read", "create", "update", "delete"}, RequiredPermissions: []string{"admin.apertures"}, RiskLevel: "high"},
	{Key: "topup_wallet", Label: "Nạp tiền và ví", Category: "economy", FrontendRoute: "/", Capabilities: []string{"read"}, RequiredPermissions: []string{"admin.topup_wallet"}, RiskLevel: "critical"},
	{Key: "gu_history", Label: "Lịch sử Tiên Cổ", Category: "game", FrontendRoute: "/", Capabilities: []string{"read"}, RequiredPermissions: []string{"admin.gu_history"}, RiskLevel: "medium"},
}

func legacyAdminPermissions() []string {
	permissions := make([]string, 0, len(adminResourceCatalog))
	for _, resource := range adminResourceCatalog {
		permissions = append(permissions, resource.RequiredPermissions...)
	}
	return permissions
}

func visibleAdminResources(permissions []string) []adminResource {
	granted := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		granted[permission] = struct{}{}
	}

	resources := make([]adminResource, 0, len(adminResourceCatalog))
	for _, resource := range adminResourceCatalog {
		visible := true
		for _, required := range resource.RequiredPermissions {
			if _, ok := granted[required]; !ok {
				visible = false
				break
			}
		}
		if visible {
			resource.Capabilities = append([]string(nil), resource.Capabilities...)
			resource.RequiredPermissions = append([]string(nil), resource.RequiredPermissions...)
			resources = append(resources, resource)
		}
	}
	return resources
}

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

	c.JSON(http.StatusOK, gin.H{
		"resources":          visibleAdminResources(legacyAdminPermissions()),
		"permission_version": legacyAdminPermissionVersion,
	})
}
