package handlers

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// Bump this marker whenever the permission model or resource catalog
// changes so clients can invalidate their cached access snapshot.
const legacyAdminPermissionVersion = "admin-resources-v3"

type adminResource struct {
	Key                   string   `json:"key"`
	Label                 string   `json:"label"`
	Category              string   `json:"category"`
	FrontendRoute         string   `json:"frontend_route"`
	Capabilities          []string `json:"capabilities"`
	RequiredPermissions   []string `json:"required_permissions"`
	RiskLevel             string   `json:"risk_level"`
	ManagementMode        string   `json:"management_mode"`
	capabilityPermissions map[string][]string
}

// adminResourceCatalog is application-owned navigation metadata. It must not
// be populated from database introspection: callers should never learn table,
// column, schema, or SQL details through this endpoint.
var adminResourceCatalog = []adminResource{
	{Key: "players", Label: "Người chơi", Category: "game", FrontendRoute: "/admin/players", Capabilities: []string{"read", "domain_commands"}, RequiredPermissions: []string{"players.read"}, RiskLevel: "high", ManagementMode: "domain_command_only", capabilityPermissions: map[string][]string{"domain_commands": {"players.update"}}},
	{Key: "sects", Label: "Tông môn", Category: "game", FrontendRoute: "/admin/sects", Capabilities: []string{"read", "domain_commands"}, RequiredPermissions: []string{"sects.read"}, RiskLevel: "high", ManagementMode: "domain_command_only", capabilityPermissions: map[string][]string{"domain_commands": {"sects.update"}}},
	{Key: "apertures", Label: "Tiên khiếu", Category: "game", FrontendRoute: "/admin/apertures", Capabilities: []string{"read", "domain_commands"}, RequiredPermissions: []string{"apertures.read"}, RiskLevel: "high", ManagementMode: "domain_command_only", capabilityPermissions: map[string][]string{"domain_commands": {"apertures.update"}}},
	{Key: "gu_history", Label: "Lịch sử Tiên Cổ", Category: "game", FrontendRoute: "/admin/gu_history", Capabilities: []string{"read"}, RequiredPermissions: []string{"gus.read"}, RiskLevel: "medium", ManagementMode: "read_only"},
	{Key: "data", Label: "Dữ liệu", Category: "content", FrontendRoute: "/admin/data", Capabilities: []string{"read", "import", "export"}, RequiredPermissions: []string{"game_data.read"}, RiskLevel: "critical", ManagementMode: "import_export_only", capabilityPermissions: map[string][]string{"import": {"game_data.import"}, "export": {"game_data.export"}}},
	{Key: "shop", Label: "Cửa hàng", Category: "economy", FrontendRoute: "/admin/shop", Capabilities: []string{"read", "create", "update", "delete"}, RequiredPermissions: []string{"shop.read"}, RiskLevel: "high", ManagementMode: "full_crud", capabilityPermissions: map[string][]string{"create": {"shop.update"}, "update": {"shop.update"}, "delete": {"shop.update"}}},
	{Key: "mines", Label: "Mỏ", Category: "economy", FrontendRoute: "/admin/mines", Capabilities: []string{"read", "update", "reset"}, RequiredPermissions: []string{"mine_slots.read"}, RiskLevel: "high", ManagementMode: "domain_command_only", capabilityPermissions: map[string][]string{"update": {"mine_slots.update"}, "reset": {"mine_slots.update"}}},
	{Key: "gift_codes", Label: "Mã quà", Category: "economy", FrontendRoute: "/admin/gift_codes", Capabilities: []string{"read", "create"}, RequiredPermissions: []string{"gift_codes.read"}, RiskLevel: "high", ManagementMode: "domain_command_only", capabilityPermissions: map[string][]string{"create": {"gift_codes.update"}}},
	{Key: "topup_wallet", Label: "Nạp tiền và ví", Category: "economy", FrontendRoute: "/admin/topup_wallet", Capabilities: []string{"read"}, RequiredPermissions: []string{"billing.read"}, RiskLevel: "critical", ManagementMode: "read_only"},
	{Key: "logs", Label: "Nhật ký", Category: "system", FrontendRoute: "/admin/logs", Capabilities: []string{"read"}, RequiredPermissions: []string{"audit.read"}, RiskLevel: "medium", ManagementMode: "read_only"},
	{Key: "monitoring", Label: "Giám sát", Category: "system", FrontendRoute: "/admin/monitoring", Capabilities: []string{"read", "reset"}, RequiredPermissions: []string{"monitoring.read"}, RiskLevel: "medium", ManagementMode: "domain_command_only", capabilityPermissions: map[string][]string{"reset": {"monitoring.reset"}}},
}

func catalogPermissions() []string {
	permissions := make([]string, 0, len(adminResourceCatalog))
	for _, resource := range adminResourceCatalog {
		permissions = append(permissions, resource.RequiredPermissions...)
		for _, required := range resource.capabilityPermissions {
			permissions = append(permissions, required...)
		}
	}
	return permissions
}

func visibleAdminResources(permissions []string) []adminResource {
	granted := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		granted[permission] = struct{}{}
	}
	_, hasWildcard := granted["admin.*"]

	resources := make([]adminResource, 0, len(adminResourceCatalog))
	for _, resource := range adminResourceCatalog {
		visible := true
		for _, required := range resource.RequiredPermissions {
			if hasWildcard {
				continue
			}
			if _, ok := granted[required]; !ok {
				visible = false
				break
			}
		}
		if visible {
			capabilities := make([]string, 0, len(resource.Capabilities))
			for _, capability := range resource.Capabilities {
				required := resource.capabilityPermissions[capability]
				allowed := hasWildcard || len(required) == 0
				for _, permission := range required {
					if _, ok := granted[permission]; ok {
						allowed = true
						break
					}
				}
				if allowed {
					capabilities = append(capabilities, capability)
				}
			}
			resource.Capabilities = capabilities
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

	principal, ok := middleware.GetAdminPrincipal(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	permissions := make([]string, 0, len(principal.Permissions))
	for permission := range principal.Permissions {
		permissions = append(permissions, permission)
	}

	c.JSON(http.StatusOK, gin.H{
		"resources":          visibleAdminResources(permissions),
		"permission_version": legacyAdminPermissionVersion,
	})
}
