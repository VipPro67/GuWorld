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

const AdminAccessContext = createContext<AdminAccess | null>(null);

export const AdminAccessProvider = AdminAccessContext.Provider;

export function useAdminAccess(): AdminAccess {
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
