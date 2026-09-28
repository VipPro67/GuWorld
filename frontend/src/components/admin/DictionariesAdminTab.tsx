import React, { useEffect, useState, useCallback } from "react";
import { Plus, RefreshCw, X, Shield, Activity, Users } from "lucide-react";
import { AdminDataGrid, AdminGridColumn } from "./AdminDataGrid";
import { GuEffectsEditor, validateAllEffectEditors } from "./GuEffectsEditor";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const parseEffectsForEditor = (value: ApiResponseData) => {
  if (typeof value !== "string") return value || { schema_version: 2, blocks: [] };
  try { return JSON.parse(value); } catch { return { schema_version: 2, blocks: [] }; }
};

function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(^| )${name}=([^;]+)`));
  return match ? decodeURIComponent(match[2]) : null;
}

function authFetch(input: RequestInfo | URL, init: RequestInit = {}) {
  const csrfToken = getCookie("gu_csrf_token");
  const headers: Record<string, string> = {};
  if (init.headers) {
    if (Array.isArray(init.headers)) {
      for (const [k, v] of init.headers) headers[k] = v;
    } else if (init.headers instanceof Headers) {
      init.headers.forEach((v, k) => { headers[k] = v; });
    } else {
      Object.assign(headers, init.headers);
    }
  }
  if (csrfToken && init.method && init.method !== "GET" && init.method !== "HEAD") {
    headers["X-CSRF-Token"] = csrfToken;
  }
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("gu_auth_token");
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
  }
  return fetch(input, { ...init, headers, credentials: "include" });
}

export const DictionariesAdminTab: React.FC<{
  type: "constitution" | "race" | "status";
  addToast: (msg: string, type: "success" | "error" | "info") => void;
}> = ({ type, addToast }) => {
  const [data, setData] = useState<ApiResponseData[]>([]);
  const [loading, setLoading] = useState(false);
  const [editItem, setEditItem] = useState<ApiResponseData | null>(null);
  const [isNew, setIsNew] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const endpoint =
        type === "constitution"
          ? "/api/admin/constitution-dictionary"
          : type === "race"
          ? "/api/admin/race-dictionary"
          : "/api/admin/status-dictionary";
      const res = await authFetch(`${BACKEND_URL}${endpoint}`);
      if (res.ok) {
        setData(await res.json());
      } else {
        addToast("Không thể tải danh sách", "error");
      }
    } catch {
      addToast("Lỗi kết nối server", "error");
    } finally {
      setLoading(false);
    }
  }, [type, addToast]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleSave = async () => {
    if (!editItem || !editItem.key) {
      addToast("Key không được để trống", "error");
      return;
    }
    if (!await validateAllEffectEditors()) {
      addToast("Effect Block chưa hợp lệ", "error");
      return;
    }
    try {
      const endpoint =
        type === "constitution"
          ? "/api/admin/constitution-dictionary"
          : type === "race"
          ? "/api/admin/race-dictionary"
          : "/api/admin/status-dictionary";
      const url = isNew ? `${BACKEND_URL}${endpoint}` : `${BACKEND_URL}${endpoint}/${editItem.key}`;
      const method = isNew ? "POST" : "PUT";
      const body = { ...editItem };
      try {
        if (typeof body.effects === "string") body.effects = JSON.parse(body.effects);
        if (type === "status" && typeof body.blocked_actions === "string") body.blocked_actions = JSON.parse(body.blocked_actions);
      } catch {
        addToast("Effects hoặc danh sách hành động bị chặn không phải JSON hợp lệ", "error");
        return;
      }
      const res = await authFetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (res.ok) {
        addToast("Lưu thành công", "success");
        setEditItem(null);
        fetchData();
      } else {
        const errData = await res.json().catch(() => ({}));
        addToast(`Lưu thất bại: ${errData.error || "Lỗi server"}`, "error");
      }
    } catch (caught: unknown) {
      const e = caught instanceof Error ? caught : new Error(String(caught));
      addToast(`Lỗi: ${e.message}`, "error");
    }
  };

  const handleDelete = async (key: string) => {
    if (!confirm(`Bạn có chắc muốn xóa '${key}'?`)) return;
    try {
      const endpoint =
        type === "constitution"
          ? "/api/admin/constitution-dictionary"
          : type === "race"
          ? "/api/admin/race-dictionary"
          : "/api/admin/status-dictionary";
      const res = await authFetch(`${BACKEND_URL}${endpoint}/${key}`, { method: "DELETE" });
      if (res.ok) {
        addToast("Xóa thành công", "success");
        fetchData();
      } else {
        addToast("Không thể xóa", "error");
      }
    } catch {
      addToast("Lỗi kết nối khi xóa", "error");
    }
  };

  const title =
    type === "constitution" ? "Từ điển Thể chất" : type === "race" ? "Từ điển Chủng tộc" : "Từ điển Trạng thái";

  const columns: AdminGridColumn<ApiResponseData>[] = [
    { key: "key", label: "Key", value: (item) => item.key, className: "font-mono font-bold text-amber-400" },
    { key: "name", label: "Tên", value: (item) => item.name, className: "font-semibold text-zinc-200" },
    ...(type === "constitution" ? [{ key: "tier", label: "Tầng", value: (item: ApiResponseData) => item.tier, className: "font-mono text-emerald-400" }] : []),
    ...(type === "status" ? [{ key: "duration", label: "Thời gian", value: (item: ApiResponseData) => item.default_duration_minutes, render: (item: ApiResponseData) => `${item.default_duration_minutes ?? 0} phút` }] : []),
    { key: "description", label: "Mô tả", value: (item) => item.description, className: "max-w-xs truncate text-zinc-400" },
    { key: "effects", label: "Effects JSON", value: (item) => item.effects, render: (item) => <span className="block max-w-xs truncate font-mono text-[10px] text-zinc-500">{JSON.stringify(item.effects)}</span> },
  ];

  const copyItem = (item: ApiResponseData) => {
    setIsNew(true);
    setEditItem({ ...structuredClone(item), key: "" });
  };

  return (
    <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-5 space-y-4">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <h3 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
          {type === "constitution" ? (
            <Shield className="w-4 h-4 text-amber-400" />
          ) : type === "race" ? (
            <Users className="w-4 h-4 text-emerald-400" />
          ) : (
            <Activity className="w-4 h-4 text-sky-400" />
          )}
          {title} ({data.length})
        </h3>
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              setIsNew(true);
              setEditItem(
                type === "constitution"
                  ? { key: "", name: "", tier: 1, description: "", effects: { schema_version: 2, blocks: [] } }
                  : type === "race"
                  ? { key: "", name: "", description: "", effects: { schema_version: 2, blocks: [] } }
                  : { key: "", name: "", description: "", effects: { schema_version: 2, blocks: [] }, default_duration_minutes: 30, blocked_actions: [] }
              );
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-amber-600 hover:bg-amber-500 text-xs font-bold text-white transition-colors cursor-pointer"
          >
            <Plus className="w-3.5 h-3.5" /> Thêm mới
          </button>
          <button
            onClick={fetchData}
            className="p-1.5 hover:bg-zinc-800 rounded-lg text-zinc-400 hover:text-zinc-200 transition-colors cursor-pointer"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
          </button>
        </div>
      </div>

      <AdminDataGrid rows={data} columns={columns} rowKey={(item) => item.key} loading={loading}
        onEdit={(item) => { setIsNew(false); setEditItem(structuredClone(item)); }}
        onCopy={copyItem} onDelete={(item) => handleDelete(item.key)} />

      {/* Edit / Create Modal */}
      {editItem && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
          <div className="w-full max-w-lg bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold text-zinc-100 font-serif">
                {isNew ? "Thêm mới" : "Chỉnh sửa"} {title}
              </h3>
              <button onClick={() => setEditItem(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer">
                <X className="w-4 h-4 text-zinc-500" />
              </button>
            </div>

            <div className="space-y-3 text-xs">
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Key (Mã định danh)</label>
                <input
                  type="text"
                  disabled={!isNew}
                  value={editItem.key || ""}
                  onChange={(e) => setEditItem({ ...editItem, key: e.target.value })}
                  placeholder="solar_sun_moon_body"
                  className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50 disabled:opacity-60"
                />
              </div>

              {type === "status" && (
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Hành động bị chặn (JSON array)</label>
                  <textarea value={typeof editItem.blocked_actions === "string" ? editItem.blocked_actions : JSON.stringify(editItem.blocked_actions ?? [], null, 2)} onChange={(e) => setEditItem({ ...editItem, blocked_actions: e.target.value })} rows={3} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 font-mono text-[11px] mt-1 focus:outline-none focus:border-amber-500/50" />
                </div>
              )}

              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Tên hiển thị</label>
                <input
                  type="text"
                  value={editItem.name || ""}
                  onChange={(e) => setEditItem({ ...editItem, name: e.target.value })}
                  placeholder="Nhật Nguyệt Thể"
                  className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                />
              </div>

              {type === "constitution" && (
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Tầng Phẩm Cấp (Tier)</label>
                  <input
                    type="number"
                    min={1}
                    value={editItem.tier || 1}
                    onChange={(e) => setEditItem({ ...editItem, tier: Number(e.target.value) })}
                    className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                  />
                </div>
              )}

              {type === "status" && (
                <div>
                  <label className="text-[11px] font-bold text-zinc-500 font-mono">Thời gian mặc định (Phút)</label>
                  <input
                    type="number"
                    min={0}
                    value={editItem.default_duration_minutes || 0}
                    onChange={(e) => setEditItem({ ...editItem, default_duration_minutes: Number(e.target.value) })}
                    className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                  />
                </div>
              )}

              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Mô tả</label>
                <textarea
                  value={editItem.description || ""}
                  onChange={(e) => setEditItem({ ...editItem, description: e.target.value })}
                  rows={2}
                  className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                />
              </div>

              <GuEffectsEditor
                effects={parseEffectsForEditor(editItem.effects)}
                source={type}
                effectKey={editItem.key || "new"}
                onChange={(effects) => setEditItem({ ...editItem, effects })}
              />
            </div>

            <div className="flex gap-3 justify-end pt-2">
              <button
                onClick={() => setEditItem(null)}
                className="px-4 py-2 rounded-lg text-xs font-bold bg-zinc-900 text-zinc-400 hover:text-white border border-zinc-800 transition-colors cursor-pointer"
              >
                Hủy
              </button>
              <button
                onClick={handleSave}
                className="px-4 py-2 rounded-lg text-xs font-bold bg-amber-600 hover:bg-amber-500 text-zinc-100 transition-colors cursor-pointer"
              >
                Lưu
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
