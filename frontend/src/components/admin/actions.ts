export type AdminRiskLevel = "low" | "medium" | "high" | "destructive";

export type AdminActionPolicy = {
  permission: string;
  riskLevel: AdminRiskLevel;
  confirmationRequired: boolean;
  reasonRequired: boolean;
  reauthenticationRequired: boolean;
};

/** Application-owned safety declarations for every supported admin action. */
export const ADMIN_ACTIONS = {
  viewPlayer: { permission: "players.read", riskLevel: "low", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  updatePlayer: { permission: "players.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  updatePlayerRecipe: { permission: "players.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  updatePlayerGu: { permission: "players.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deletePlayerGu: { permission: "players.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  deletePlayerPet: { permission: "players.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updatePlayerInventoryItem: { permission: "players.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deletePlayerInventoryItem: { permission: "players.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updatePlayerQuest: { permission: "players.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deletePlayerQuest: { permission: "players.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updatePlayerInstanceSession: { permission: "players.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  updatePlayerSectMission: { permission: "players.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  updatePlayerTrackingMark: { permission: "players.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deletePlayerTrackingMark: { permission: "players.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updatePlayerAchievement: { permission: "players.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  grantPlayerAchievement: { permission: "players.update", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  deletePlayerAchievement: { permission: "players.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateResidualTrace: { permission: "regions.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deleteResidualTrace: { permission: "regions.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateMapDrop: { permission: "map_drops.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deleteMapDrop: { permission: "map_drops.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateArenaRanking: { permission: "arena_rankings.update", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateApertureChild: { permission: "apertures.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deleteApertureChild: { permission: "apertures.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateVariantNPC: { permission: "apertures.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  dismissVariantNPC: { permission: "apertures.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  removeSectDefender: { permission: "sects.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  cancelSectRequest: { permission: "sects.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  deletePlayerInstanceSession: { permission: "players.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  grantPlayerInventory: { permission: "players.inventory.grant", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  importGameDataset: { permission: "game_data.import", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  importAllGameData: { permission: "game_data.import_all", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateItemDictionary: { permission: "item_dictionary.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deleteItemDictionary: { permission: "item_dictionary.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateRecipe: { permission: "recipes.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deleteRecipe: { permission: "recipes.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateMonster: { permission: "monsters.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deleteMonster: { permission: "monsters.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateLootTable: { permission: "loot_tables.update", riskLevel: "medium", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  deleteLootTable: { permission: "loot_tables.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateMonsterSpawn: { permission: "monster_spawns.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteMonsterSpawn: { permission: "monster_spawns.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateWorldBoss: { permission: "world_bosses.update", riskLevel: "high", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  spawnWorldBoss: { permission: "world_bosses.update", riskLevel: "high", confirmationRequired: true, reasonRequired: false, reauthenticationRequired: false },
  purgeWorldBosses: { permission: "world_bosses.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateWorldQuest: { permission: "quests.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteWorldQuest: { permission: "quests.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateDungeon: { permission: "dungeons.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteDungeon: { permission: "dungeons.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateConstitution: { permission: "constitution_dictionary.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteConstitution: { permission: "constitution_dictionary.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateRaceDictionary: { permission: "race_dictionary.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteRaceDictionary: { permission: "race_dictionary.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateStatusDictionary: { permission: "status_dictionary.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteStatusDictionary: { permission: "status_dictionary.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateRegion: { permission: "regions.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteRegion: { permission: "regions.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateMapCell: { permission: "map_cells.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteMapCell: { permission: "map_cells.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateShop: { permission: "shop.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteShop: { permission: "shop.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  updateArenaShop: { permission: "arena_shop.update", riskLevel: "medium", confirmationRequired: false, reasonRequired: false, reauthenticationRequired: false },
  deleteArenaShop: { permission: "arena_shop.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  flushRedis: { permission: "cache.flush", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: true },
	deleteData: { permission: "data.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
	adjustPlayerResources: { permission: "players.resources.adjust", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
	banPlayer: { permission: "players.ban", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
	unbanPlayer: { permission: "players.ban", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
	mutePlayer: { permission: "players.ban", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
	unmutePlayer: { permission: "players.ban", riskLevel: "medium", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
	assignPlayerRoles: { permission: "admin_roles.assign", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: true },
	mutatePayment: { permission: "billing.mutate", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: true },
} as const satisfies Record<string, AdminActionPolicy>;

export type AdminActionKey = keyof typeof ADMIN_ACTIONS;

export function canPerformAdminAction(permissions: readonly string[], action: AdminActionKey): boolean {
  return permissions.includes("admin.*") || permissions.includes(ADMIN_ACTIONS[action].permission);
}

export type PreparedAdminAction = { reason?: string; confirmation_target?: string; reauth_password?: string; idempotencyKey: string };

/** Enforces the declared client-side safety ceremony immediately before execution. */
export function prepareAdminAction(actionKey: AdminActionKey, label: string, target: string): PreparedAdminAction | null {
  const policy = ADMIN_ACTIONS[actionKey];
  let reason: string | undefined;
  if (policy.reasonRequired) {
    reason = window.prompt(`Lý do ${label}:`)?.trim();
    if (!reason) return null;
  }
  if (policy.reauthenticationRequired) {
	const reauthPassword = window.prompt("Nhập lại mật khẩu để xác thực thao tác:");
	if (!reauthPassword) return null;
	if (!window.confirm(`Xác nhận ${label} trên target “${target}”?`)) return null;
	return { reason, confirmation_target: target, reauth_password: reauthPassword, idempotencyKey: crypto.randomUUID() };
  }
  if (policy.confirmationRequired) {
    const warning = policy.riskLevel === "destructive" ? " Đây là thao tác phá hủy và có thể không hoàn tác được." : "";
	if (!window.confirm(`Xác nhận ${label} trên target “${target}”?${warning}`)) return null;
  }
	return { reason, confirmation_target: target, idempotencyKey: crypto.randomUUID() };
}
