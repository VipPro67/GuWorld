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
  importGameDataset: { permission: "game_data.import", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  importAllGameData: { permission: "game_data.import_all", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
  flushRedis: { permission: "cache.flush", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: true },
	deleteData: { permission: "data.delete", riskLevel: "destructive", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
	adjustPlayerResources: { permission: "players.resources.adjust", riskLevel: "high", confirmationRequired: true, reasonRequired: true, reauthenticationRequired: false },
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
