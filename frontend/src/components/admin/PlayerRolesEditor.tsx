"use client";

import { useCallback, useEffect, useState } from "react";
import { Crown, RefreshCw, ShieldAlert, X } from "lucide-react";
import { authFetch } from "@/lib/auth";
import { prepareAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

export type AdminRoleOption = {
  key: string;
  name: string;
  description: string;
  level: number;
};

type PlayerRolesEditorProps = {
  playerId: string;
  username: string;
  canAssign: boolean;
  addToast: (message: string, type: "success" | "error" | "info") => void;
  onClose: () => void;
  onSaved: () => void;
};

export function PlayerRolesEditor({ playerId, username, canAssign, addToast, onClose, onSaved }: PlayerRolesEditorProps) {
  const [catalog, setCatalog] = useState<AdminRoleOption[]>([]);
  const [selected, setSelected] = useState<string[]>([]);
  const [baseline, setBaseline] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadError, setLoadError] = useState("");

  const fetchRoles = useCallback(async () => {
    setLoading(true);
    setLoadError("");
    try {
      const [catalogRes, playerRes] = await Promise.all([
        authFetch(`${BACKEND_URL}/api/admin/roles`),
        authFetch(`${BACKEND_URL}/api/admin/players/${playerId}/roles`),
      ]);
      if (catalogRes.status === 403 || playerRes.status === 403) {
        throw new Error("Tài khoản của bạn không có quyền xem vai trò (cần admin_roles.read).");
      }
      if (!catalogRes.ok || !playerRes.ok) {
        const data = await (catalogRes.ok ? playerRes : catalogRes).json().catch(() => null);
        throw new Error((data as { error?: string } | null)?.error || "Không thể tải vai trò");
      }
      const catalogData = (await catalogRes.json()) as { roles?: AdminRoleOption[] };
      const playerData = (await playerRes.json()) as { roles?: { key: string }[] };
      setCatalog(Array.isArray(catalogData.roles) ? catalogData.roles : []);
      const keys = Array.isArray(playerData.roles) ? playerData.roles.map((role) => role.key) : [];
      setSelected(keys);
      setBaseline(keys);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Không thể tải vai trò";
      setLoadError(message);
      addToast(message, "error");
    } finally {
      setLoading(false);
    }
  }, [playerId, addToast]);

  useEffect(() => {
    void fetchRoles();
  }, [fetchRoles]);

  const toggle = (key: string) => {
    if (!canAssign || saving) return;
    setSelected((current) => (current.includes(key) ? current.filter((item) => item !== key) : [...current, key]));
  };

  const dirty = JSON.stringify([...selected].sort()) !== JSON.stringify([...baseline].sort());

  const save = async () => {
    if (!dirty || saving) return;
    const prepared = prepareAdminAction("assignPlayerRoles", `gán vai trò cho ${username}`, playerId);
    if (!prepared) return;
    setSaving(true);
    try {
      const response = await authFetch(`${BACKEND_URL}/api/admin/players/${playerId}/roles`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "Idempotency-Key": prepared.idempotencyKey },
        body: JSON.stringify({
          roles: selected,
          reason: prepared.reason,
          confirmation_target: prepared.confirmation_target,
          reauth_password: prepared.reauth_password,
        }),
      });
      const data = await response.json().catch(() => null);
      if (!response.ok) {
        throw new Error((data as { error?: string } | null)?.error || "Cập nhật vai trò thất bại");
      }
      const after = (data as { roles?: { key: string }[] } | null)?.roles;
      const keys = Array.isArray(after) ? after.map((role) => role.key) : selected;
      setSelected(keys);
      setBaseline(keys);
      addToast("Đã cập nhật vai trò quản trị.", "success");
      onSaved();
      onClose();
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Cập nhật vai trò thất bại", "error");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
      <div className="w-full max-w-lg bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
            <Crown className="w-4 h-4 text-amber-400" />
            Vai trò quản trị — {username}
          </h3>
          <button onClick={onClose} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer" aria-label="Đóng">
            <X className="w-4 h-4 text-zinc-500" />
          </button>
        </div>

        <p className="text-[11px] text-zinc-500 font-mono break-all">{playerId}</p>

        {loading ? (
          <div className="flex flex-col items-center py-10">
            <RefreshCw className="w-6 h-6 text-amber-400 animate-spin" />
            <span className="text-xs text-zinc-500 font-mono mt-2">Đang tải vai trò...</span>
          </div>
        ) : loadError ? (
          <div className="text-center py-8 space-y-3">
            <p className="text-xs text-red-400">{loadError}</p>
            <button onClick={() => void fetchRoles()} className="text-xs text-amber-400 underline cursor-pointer">Thử lại</button>
          </div>
        ) : (
          <>
            <div className="space-y-2">
              {catalog.map((role) => {
                const checked = selected.includes(role.key);
                return (
                  <label
                    key={role.key}
                    className={`flex items-start gap-3 rounded-xl border p-3 transition-colors ${!canAssign ? "opacity-70" : "cursor-pointer"} ${checked ? "border-amber-500/50 bg-amber-500/5" : "border-zinc-800 bg-zinc-900/40 hover:border-zinc-700"}`}
                  >
                    <input
                      type="checkbox"
                      checked={checked}
                      disabled={!canAssign || saving}
                      onChange={() => toggle(role.key)}
                      className="mt-0.5 accent-amber-500 w-4 h-4"
                    />
                    <span className="min-w-0">
                      <span className="flex items-center gap-2 text-xs font-bold text-zinc-100">
                        {role.key === "super_admin" && <ShieldAlert className="w-3.5 h-3.5 text-red-400" />}
                        {role.name || role.key}
                        <span className="rounded bg-zinc-800 px-1.5 py-0.5 text-[10px] font-mono text-zinc-400">lv {role.level}</span>
                      </span>
                      <span className="block truncate text-[11px] font-mono text-zinc-500" title={role.key}>{role.key}</span>
                      {role.description && <span className="mt-0.5 block text-[11px] text-zinc-400">{role.description}</span>}
                    </span>
                  </label>
                );
              })}
              {catalog.length === 0 && <p className="text-xs text-zinc-500 text-center py-4">Chưa có vai trò nào trong hệ thống.</p>}
            </div>

            {!canAssign && (
              <p className="text-xs text-red-400">Bạn chỉ có quyền xem (cần admin_roles.assign để thay đổi).</p>
            )}
            <p className="text-[11px] text-zinc-500">
              Lưu ý: không thể tự nâng quyền, cấp vai trò cao hơn quyền của bạn (trừ super admin), hoặc xóa super admin cuối cùng. Thao tác yêu cầu lý do + xác thực lại mật khẩu và được audit.
            </p>

            <div className="flex justify-end gap-2">
              <button onClick={onClose} className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 border border-zinc-800 text-zinc-300 hover:text-white transition-colors cursor-pointer">
                {canAssign ? "Hủy" : "Đóng"}
              </button>
              {canAssign && (
                <button onClick={() => void save()} disabled={!dirty || saving} className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 transition-colors cursor-pointer disabled:opacity-50">
                  {saving ? "Đang lưu..." : "Lưu vai trò"}
                </button>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
