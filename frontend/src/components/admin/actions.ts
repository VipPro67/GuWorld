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
  grantPlayerInventory: { permission: "players.inventory.grant", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  importAllGameData: { permission: "game_data.import", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  flushRedis: { permission: "cache.flush", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
} as const satisfies Record<string, AdminActionPolicy>;

export type AdminActionKey = keyof typeof ADMIN_ACTIONS;

export function canPerformAdminAction(permissions: readonly string[], action: AdminActionKey): boolean {
  return permissions.includes("admin.*") || permissions.includes(ADMIN_ACTIONS[action].permission);
}

export type PreparedAdminAction = { reason?: string };

/** Enforces the declared client-side safety ceremony immediately before execution. */
export function prepareAdminAction(actionKey: AdminActionKey, label: string): PreparedAdminAction | null {
  const policy = ADMIN_ACTIONS[actionKey];
  let reason: string | undefined;
  if (policy.reasonRequired) {
    reason = window.prompt(`Lý do ${label}:`)?.trim();
    if (!reason) return null;
  }
  if (policy.reauthenticationRequired) {
    throw new Error(`Action ${actionKey} requires a re-authentication provider`);
  }
  if (policy.confirmationRequired) {
    const warning = policy.riskLevel === "destructive" ? " Đây là thao tác phá hủy và có thể không hoàn tác được." : "";
    if (!window.confirm(`Xác nhận ${label}?${warning}`)) return null;
  }
  return reason ? { reason } : {};
}
