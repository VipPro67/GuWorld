// Package admincatalog owns the compile-time allowlist of administrative
// resources. It deliberately contains no database discovery or table names.
package admincatalog

// Resource describes one administrative API domain. Permission lists are
// split so callers can distinguish observation from mutation without deriving
// either from a client-supplied resource name.
type Resource struct {
	Key               string   `json:"key"`
	Domain            string   `json:"domain"`
	Label             string   `json:"label"`
	Capabilities      []string `json:"capabilities"`
	ReadPermissions   []string `json:"read_permissions"`
	WritePermissions  []string `json:"write_permissions"`
	RiskLevel         string   `json:"risk_level"`
	CacheInvalidation []string `json:"cache_invalidation"`
	RegistryReload    []string `json:"registry_reload"`
	RestartRequired   bool     `json:"restart_required"`
	AuditActionPrefix string   `json:"audit_action_prefix"`
	FrontendRoute     string   `json:"frontend_route"`
}

// resources is the complete allowlist. Additions must be reviewed alongside
// adminRouteSpecs; resources are never inferred from information_schema.
var resources = []Resource{
	resource("monitoring", "system", "Giám sát", "/admin/monitoring", []string{"read", "reset"}, []string{"monitoring.read"}, []string{"monitoring.reset"}, "medium", nil, nil, "admin."),
	resource("admin_roles", "identity", "Vai trò quản trị", "/admin/players", []string{"read", "assign"}, []string{"admin_roles.read"}, []string{"admin_roles.assign"}, "critical", nil, nil, "admin."),
	resource("billing", "economy", "Nạp tiền và ví", "/admin/topup_wallet", []string{"read"}, []string{"billing.read"}, nil, "high", nil, nil, "admin."),
	resource("players", "game", "Người chơi", "/admin/players", []string{"read", "update", "manage_inventory"}, []string{"players.read"}, []string{"players.update", "players.inventory.grant", "players.resources.adjust", "players.delete"}, "critical", []string{"player profile and inventory caches"}, nil, "admin."),
	resource("apertures", "game", "Tiên khiếu", "/admin/apertures", []string{"read", "create", "update", "delete"}, []string{"apertures.read"}, []string{"apertures.update", "apertures.delete"}, "high", nil, nil, "admin."),
	resource("gus", "game", "Lịch sử Tiên Cổ", "/admin/gu_history", []string{"read"}, []string{"gus.read"}, nil, "medium", nil, nil, "admin."),
	resource("sects", "game", "Tông môn", "/admin/sects", []string{"read", "create", "update", "delete"}, []string{"sects.read"}, []string{"sects.update", "sects.delete"}, "critical", []string{"sects_list"}, nil, "admin."),
	resource("game_data", "content", "Nhập/xuất dữ liệu", "/admin/data", []string{"read", "import", "export"}, []string{"game_data.read"}, []string{"game_data.import", "game_data.import_all", "game_data.export"}, "critical", []string{"dictionary and gameplay caches"}, []string{"affected gameplay registries"}, "admin."),
	resource("cache", "system", "Bộ nhớ đệm", "/admin/data", []string{"reload", "flush"}, nil, []string{"cache.reload", "cache.flush"}, "critical", []string{"application and Redis caches"}, []string{"all gameplay registries"}, "admin."),
	resource("audit", "system", "Nhật ký", "/admin/logs", []string{"read"}, []string{"audit.read"}, nil, "medium", nil, nil, "admin."),
	resource("regions", "world", "Khu vực", "/admin/data", crud(), []string{"regions.read"}, []string{"regions.update", "regions.delete"}, "high", []string{"region dictionary"}, []string{"zone registry"}, "admin."),
	resource("map_cells", "world", "Ô bản đồ", "/admin/data", crud(), []string{"map_cells.read"}, []string{"map_cells.update", "map_cells.delete"}, "high", []string{"world map caches"}, []string{"zone registry"}, "admin."),
	resource("shop", "economy", "Cửa hàng", "/admin/shop", crud(), []string{"shop.read"}, []string{"shop.update", "shop.delete"}, "high", []string{"premium_shop_packages"}, nil, "admin."),
	resource("arena_shop", "economy", "Cửa hàng đấu trường", "/admin/data", crud(), []string{"arena_shop.read"}, []string{"arena_shop.update", "arena_shop.delete"}, "high", []string{"arena_shop_items"}, nil, "admin."),
	resource("mine_slots", "world", "Mỏ", "/admin/mines", []string{"read", "update", "reset"}, []string{"mine_slots.read"}, []string{"mine_slots.update"}, "high", []string{"mine_slots_list"}, nil, "admin."),
	resource("gu_dictionary", "content", "Từ điển Cổ", "/admin/data", append(crud(), "validate", "audit"), []string{"gu_dictionary.read"}, []string{"gu_dictionary.update", "gu_dictionary.delete"}, "critical", []string{"game dictionary", "gacha pools"}, []string{"Gu registry"}, "admin."),
	resource("item_dictionary", "content", "Từ điển vật phẩm", "/admin/data", crud(), []string{"item_dictionary.read"}, []string{"item_dictionary.update", "item_dictionary.delete"}, "high", []string{"game dictionary", "gacha pools"}, []string{"item registry"}, "admin."),
	resource("recipes", "content", "Cổ phương", "/admin/data", crud(), []string{"recipes.read"}, []string{"recipes.update", "recipes.delete"}, "high", []string{"recipe catalog", "gacha pools"}, []string{"recipe catalog"}, "admin."),
	resource("monsters", "content", "Quái vật", "/admin/data", crud(), []string{"monsters.read"}, []string{"monsters.update", "monsters.delete"}, "high", []string{"monster dictionary"}, []string{"monster registry"}, "admin."),
	resource("loot_tables", "content", "Bảng chiến lợi phẩm", "/admin/data", crud(), []string{"loot_tables.read"}, []string{"loot_tables.update", "loot_tables.delete"}, "high", []string{"loot table dictionary"}, []string{"monster registry"}, "admin."),
	resource("gift_codes", "economy", "Mã quà", "/admin/gift_codes", []string{"read", "create"}, []string{"gift_codes.read"}, []string{"gift_codes.update"}, "high", nil, nil, "admin."),
	resource("images", "content", "Hình ảnh", "/admin/data", []string{"upload"}, nil, []string{"images.update"}, "medium", nil, nil, "admin."),
	resource("constitution_dictionary", "content", "Thể chất", "/admin/data", crud(), []string{"constitution_dictionary.read"}, []string{"constitution_dictionary.update", "constitution_dictionary.delete"}, "high", []string{"constitution dictionary"}, []string{"constitution registry"}, "admin."),
	resource("race_dictionary", "content", "Chủng tộc", "/admin/data", crud(), []string{"race_dictionary.read"}, []string{"race_dictionary.update", "race_dictionary.delete"}, "high", []string{"race dictionary"}, []string{"race registry"}, "admin."),
	resource("status_dictionary", "content", "Trạng thái", "/admin/data", crud(), []string{"status_dictionary.read"}, []string{"status_dictionary.update", "status_dictionary.delete"}, "high", []string{"status dictionary"}, []string{"status registry"}, "admin."),
	resource("monster_spawns", "world", "Điểm sinh quái", "/admin/data", crud(), []string{"monster_spawns.read"}, []string{"monster_spawns.update", "monster_spawns.delete"}, "high", []string{"monster spawn dictionary"}, []string{"zone and monster registries"}, "admin."),
	resource("quests", "content", "Nhiệm vụ", "/admin/data", crud(), []string{"quests.read"}, []string{"quests.update", "quests.delete"}, "high", []string{"game_quests_list"}, []string{"quest registry"}, "admin."),
	resource("dungeons", "world", "Phó bản", "/admin/data", crud(), []string{"dungeons.read"}, []string{"dungeons.update", "dungeons.delete"}, "high", nil, []string{"instance cache"}, "admin."),
	resource("sect_mines", "world", "Lãnh địa tông môn", "/admin/data", []string{"read", "update"}, []string{"sect_mines.read"}, []string{"sect_mines.update"}, "high", nil, nil, "admin."),
}

func resource(key, domain, label, route string, capabilities, reads, writes []string, risk string, invalidation, reload []string, auditPrefix string) Resource {
	return Resource{Key: key, Domain: domain, Label: label, Capabilities: capabilities, ReadPermissions: reads, WritePermissions: writes, RiskLevel: risk, CacheInvalidation: nonNil(invalidation), RegistryReload: nonNil(reload), AuditActionPrefix: auditPrefix, FrontendRoute: route}
}

func crud() []string { return []string{"read", "create", "update", "delete"} }

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// All returns a defensive copy of the complete allowlist.
func All() []Resource {
	return clone(resources)
}

func clone(entries []Resource) []Resource {
	result := make([]Resource, 0, len(entries))
	for _, entry := range entries {
		entry.Capabilities = append([]string{}, entry.Capabilities...)
		entry.ReadPermissions = append([]string{}, entry.ReadPermissions...)
		entry.WritePermissions = append([]string{}, entry.WritePermissions...)
		entry.CacheInvalidation = append([]string{}, entry.CacheInvalidation...)
		entry.RegistryReload = append([]string{}, entry.RegistryReload...)
		result = append(result, entry)
	}
	return result
}

// Visible returns defensive copies of resources for which can grants at least
// one declared permission. The callback can implement exact grants or wildcard
// semantics without coupling this package to authentication middleware.
func Visible(can func(string) bool) []Resource {
	result := make([]Resource, 0, len(resources))
	for _, entry := range resources {
		visible := false
		for _, permission := range append(append([]string{}, entry.ReadPermissions...), entry.WritePermissions...) {
			if can(permission) {
				visible = true
				break
			}
		}
		if !visible {
			continue
		}
		entry.Capabilities = append([]string{}, entry.Capabilities...)
		entry.ReadPermissions = append([]string{}, entry.ReadPermissions...)
		entry.WritePermissions = append([]string{}, entry.WritePermissions...)
		entry.CacheInvalidation = append([]string{}, entry.CacheInvalidation...)
		entry.RegistryReload = append([]string{}, entry.RegistryReload...)
		result = append(result, entry)
	}
	return result
}
