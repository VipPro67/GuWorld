"use client";

import { createContext, useContext } from "react";

export type AdminAccess = {
  actor_id: string;
  username: string;
  roles: string[];
  effective_permissions: string[];
  legacy_fallback: boolean;
  permission_version: string;
};

export type AdminAccessContextValue = AdminAccess & { refreshAccess: () => Promise<void> };

const AdminAccessContext = createContext<AdminAccessContextValue | null>(null);

export const AdminAccessProvider = AdminAccessContext.Provider;

export function useAdminAccess(): AdminAccessContextValue {
  const access = useContext(AdminAccessContext);
  if (!access) {
    throw new Error("useAdminAccess must be used inside the admin layout");
  }
  return access;
}

export function useAdminPermission(permission: string): boolean {
  const { effective_permissions: permissions } = useAdminAccess();
  return permissions.includes("admin.*") || permissions.includes(permission);
}

export function canAccessAdmin(access: AdminAccess, permission: string): boolean {
  return access.effective_permissions.includes("admin.*") || access.effective_permissions.includes(permission);
}

export function useCanAdmin(): (permission: string) => boolean {
  const access = useAdminAccess();
  return (permission) => canAccessAdmin(access, permission);
}
