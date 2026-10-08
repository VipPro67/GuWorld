package authz

// BuiltinRole describes the immutable access policy shipped with the server.
// The UI consumes permissions, never role names; these definitions are kept in
// one place so schema seeding and authorization tests cannot drift apart.
type BuiltinRole struct {
	Key         string
	Name        string
	Description string
	Level       int
	Permissions []string
}

var dashboard = []string{AdminAccessPermission}

var readOnly = []string{
	"monitoring.read", "players.read", "apertures.read", "gus.read", "sects.read",
	"game_data.read", "audit.read", "regions.read", "map_cells.read", "shop.read",
	"combat_history.read", "map_drops.read", "arena_rankings.read",
	"arena_shop.read", "mine_slots.read", "gu_dictionary.read", "item_dictionary.read",
	"recipes.read", "monsters.read", "loot_tables.read", "gift_codes.read",
	"constitution_dictionary.read", "race_dictionary.read", "status_dictionary.read",
	"monster_spawns.read", "quests.read", "dungeons.read", "sect_mines.read",
}

// BuiltinRoles returns fresh copies of the built-in policies.
func BuiltinRoles() []BuiltinRole {
	roles := []BuiltinRole{
		{Key: "observer", Name: "Observer", Description: "Read-only operational access", Level: 10, Permissions: append(copyPermissions(dashboard), readOnly...)},
		{Key: "finance_viewer", Name: "Finance Viewer", Description: "Read-only billing access", Level: 20, Permissions: append(copyPermissions(dashboard), "billing.read")},
		{Key: "support_admin", Name: "Support Admin", Description: "Player support operations", Level: 30, Permissions: append(copyPermissions(dashboard), "players.read", "players.update", "players.inventory.grant", "players.resources.adjust", "gus.read", "audit.read")},
		{Key: "content_admin", Name: "Content Admin", Description: "Game content authoring and publishing", Level: 60, Permissions: append(copyPermissions(dashboard),
			"game_data.read", "game_data.import", "game_data.import_all", "game_data.export", "cache.reload",
			"gu_dictionary.read", "gu_dictionary.update", "gu_dictionary.delete", "item_dictionary.read", "item_dictionary.update", "item_dictionary.delete",
			"recipes.read", "recipes.update", "recipes.delete", "monsters.read", "monsters.update", "monsters.delete",
			"loot_tables.read", "loot_tables.update", "loot_tables.delete", "constitution_dictionary.read", "constitution_dictionary.update", "constitution_dictionary.delete",
			"race_dictionary.read", "race_dictionary.update", "race_dictionary.delete", "status_dictionary.read", "status_dictionary.update", "status_dictionary.delete",
			"quests.read", "quests.update", "quests.delete", "images.update")},
		{Key: "game_admin", Name: "Game Admin", Description: "Live game and world operations", Level: 70, Permissions: append(copyPermissions(dashboard),
			"monitoring.read", "monitoring.reset", "players.read", "players.update", "players.inventory.grant", "players.resources.adjust", "players.delete", "players.ban",
			"apertures.read", "apertures.update", "apertures.delete", "gus.read", "sects.read", "sects.update", "sects.delete", "audit.read",
			"regions.read", "regions.update", "regions.delete", "map_cells.read", "map_cells.update", "map_cells.delete",
			"combat_history.read", "map_drops.read", "map_drops.update", "map_drops.delete", "arena_rankings.read", "arena_rankings.update",
			"shop.read", "shop.update", "shop.delete", "arena_shop.read", "arena_shop.update", "arena_shop.delete",
			"mine_slots.read", "mine_slots.update", "gift_codes.read", "gift_codes.update", "monster_spawns.read", "monster_spawns.update", "monster_spawns.delete",
			"dungeons.read", "dungeons.update", "dungeons.delete", "sect_mines.read", "sect_mines.update")},
		{Key: SuperAdminRole, Name: "Super Admin", Description: "Unrestricted administrative access", Level: 100, Permissions: []string{AdminAccessPermission, "admin.*"}},
	}
	for i := range roles {
		roles[i].Permissions = copyPermissions(roles[i].Permissions)
	}
	return roles
}

func copyPermissions(source []string) []string { return append([]string{}, source...) }

func PermissionsForRole(key string) []string {
	for _, role := range BuiltinRoles() {
		if role.Key == key {
			return role.Permissions
		}
	}
	return []string{}
}
