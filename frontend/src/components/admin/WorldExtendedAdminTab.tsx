import React, { useEffect, useState, useCallback } from "react";
import { Plus, Trash2, Edit3, RefreshCw, X, MapPin, ListTodo, Shield } from "lucide-react";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

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

export const WorldExtendedAdminTab: React.FC<{
  type: "spawns" | "quests" | "dungeons";
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
        type === "spawns"
          ? "/api/admin/monster-spawns"
          : type === "quests"
          ? "/api/admin/quests"
          : "/api/admin/dungeons";
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
    if (!editItem || !editItem.id) {
      addToast("ID không được để trống", "error");
      return;
    }
    try {
      const endpoint =
        type === "spawns"
          ? "/api/admin/monster-spawns"
          : type === "quests"
          ? "/api/admin/quests"
          : "/api/admin/dungeons";
      const url = isNew ? `${BACKEND_URL}${endpoint}` : `${BACKEND_URL}${endpoint}/${editItem.id}`;
      const method = isNew ? "POST" : "PUT";
      const body = { ...editItem };

      if (type === "quests" && typeof body.rewards === "string") {
        try { body.rewards = JSON.parse(body.rewards); } catch {}
      }
      if (type === "dungeons") {
        if (typeof body.rewards === "string") { try { body.rewards = JSON.parse(body.rewards); } catch {} }
        if (typeof body.map_data === "string") { try { body.map_data = JSON.parse(body.map_data); } catch {} }
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

  const handleDelete = async (id: string) => {
    if (!confirm(`Bạn có chắc muốn xóa '${id}'?`)) return;
    try {
      const endpoint =
        type === "spawns"
          ? "/api/admin/monster-spawns"
          : type === "quests"
          ? "/api/admin/quests"
          : "/api/admin/dungeons";
      const res = await authFetch(`${BACKEND_URL}${endpoint}/${id}`, { method: "DELETE" });
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
    type === "spawns" ? "Điểm Xuất Hiện Quái" : type === "quests" ? "Nhiệm Vụ Thế Giới" : "Phụ Bản / Dungeons";

  return (
    <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-5 space-y-4">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <h3 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
          {type === "spawns" ? (
            <MapPin className="w-4 h-4 text-amber-400" />
          ) : type === "quests" ? (
            <ListTodo className="w-4 h-4 text-emerald-400" />
          ) : (
            <Shield className="w-4 h-4 text-sky-400" />
          )}
          {title} ({data.length})
        </h3>
        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              setIsNew(true);
              setEditItem(
                type === "spawns"
                  ? { id: "", region_id: "", monster_id: "", x: 0, y: 0, spawn_mode: "normal", chance: 1.0, respawn_seconds: 300, min_rank: 1, max_rank: 9, is_boss_spawn: false, is_active: true }
                  : type === "quests"
                  ? { id: "", title: "", description: "", req_desc: "", message: "", rewards: {} }
                  : { id: "", name: "", type: "normal", required_rank: 1, entry_item: "", description: "", rewards: {}, map_data: {}, cooldown_hours: 24 }
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

      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse text-xs">
          <thead>
            <tr className="border-b border-zinc-800 text-zinc-400 font-mono text-[11px]">
              <th className="py-2.5 px-3">ID</th>
              {type === "spawns" && <th className="py-2.5 px-3">Khu Vực</th>}
              {type === "spawns" && <th className="py-2.5 px-3">Mẫu Quái</th>}
              {type === "spawns" && <th className="py-2.5 px-3">Tọa độ (X, Y)</th>}
              {type !== "spawns" && <th className="py-2.5 px-3">Tên / Tiêu đề</th>}
              {type === "dungeons" && <th className="py-2.5 px-3">Cấp Yêu Cầu</th>}
              <th className="py-2.5 px-3">Mô tả / Thưởng</th>
              <th className="py-2.5 px-3 text-right">Thao tác</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-800/50 text-zinc-300">
            {data.length === 0 ? (
              <tr>
                <td colSpan={6} className="text-center py-6 text-zinc-500 italic">
                  {loading ? "Đang tải dữ liệu..." : "Chưa có dữ liệu"}
                </td>
              </tr>
            ) : (
              data.map((item) => (
                <tr key={item.id} className="hover:bg-zinc-900/50">
                  <td className="py-2.5 px-3 font-mono font-bold text-amber-400">{item.id}</td>
                  {type === "spawns" && <td className="py-2.5 px-3 font-mono text-zinc-300">{item.region_id}</td>}
                  {type === "spawns" && <td className="py-2.5 px-3 font-mono text-emerald-400">{item.monster_id}</td>}
                  {type === "spawns" && (
                    <td className="py-2.5 px-3 font-mono text-sky-400">
                      ({item.x}, {item.y})
                    </td>
                  )}
                  {type !== "spawns" && <td className="py-2.5 px-3 font-semibold text-zinc-200">{item.title || item.name}</td>}
                  {type === "dungeons" && <td className="py-2.5 px-3 font-mono text-emerald-400">Chuyển {item.required_rank}</td>}
                  <td className="py-2.5 px-3 text-zinc-400 max-w-xs truncate">
                    {type === "spawns"
                      ? `Mode: ${item.spawn_mode}, Respawn: ${item.respawn_seconds}s`
                      : item.description || JSON.stringify(item.rewards)}
                  </td>
                  <td className="py-2.5 px-3 text-right">
                    <div className="flex items-center justify-end gap-1.5">
                      <button
                        onClick={() => {
                          setIsNew(false);
                          setEditItem({ ...item });
                        }}
                        className="p-1 hover:bg-zinc-800 rounded text-zinc-400 hover:text-amber-400 cursor-pointer"
                      >
                        <Edit3 className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => handleDelete(item.id)}
                        className="p-1 hover:bg-zinc-800 rounded text-zinc-400 hover:text-rose-400 cursor-pointer"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Edit Modal */}
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
                <label className="text-[11px] font-bold text-zinc-500 font-mono">ID</label>
                <input
                  type="text"
                  disabled={!isNew}
                  value={editItem.id || ""}
                  onChange={(e) => setEditItem({ ...editItem, id: e.target.value })}
                  placeholder="ID duy nhất"
                  className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50 disabled:opacity-60"
                />
              </div>

              {type === "spawns" && (
                <>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-[11px] font-bold text-zinc-500 font-mono">Region ID</label>
                      <input
                        type="text"
                        value={editItem.region_id || ""}
                        onChange={(e) => setEditItem({ ...editItem, region_id: e.target.value })}
                        className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                      />
                    </div>
                    <div>
                      <label className="text-[11px] font-bold text-zinc-500 font-mono">Monster Template ID</label>
                      <input
                        type="text"
                        value={editItem.monster_id || ""}
                        onChange={(e) => setEditItem({ ...editItem, monster_id: e.target.value })}
                        className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-[11px] font-bold text-zinc-500 font-mono">Tọa độ X</label>
                      <input
                        type="number"
                        value={editItem.x || 0}
                        onChange={(e) => setEditItem({ ...editItem, x: Number(e.target.value) })}
                        className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                      />
                    </div>
                    <div>
                      <label className="text-[11px] font-bold text-zinc-500 font-mono">Tọa độ Y</label>
                      <input
                        type="number"
                        value={editItem.y || 0}
                        onChange={(e) => setEditItem({ ...editItem, y: Number(e.target.value) })}
                        className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-[11px] font-bold text-zinc-500 font-mono">Tỉ lệ xuất hiện (0.0 - 1.0)</label>
                      <input
                        type="number"
                        step="0.1"
                        value={editItem.chance || 1.0}
                        onChange={(e) => setEditItem({ ...editItem, chance: Number(e.target.value) })}
                        className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                      />
                    </div>
                    <div>
                      <label className="text-[11px] font-bold text-zinc-500 font-mono">Thời gian hồi (Giây)</label>
                      <input
                        type="number"
                        value={editItem.respawn_seconds || 300}
                        onChange={(e) => setEditItem({ ...editItem, respawn_seconds: Number(e.target.value) })}
                        className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                      />
                    </div>
                  </div>
                </>
              )}

              {type === "quests" && (
                <>
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Tên Nhiệm Vụ</label>
                    <input
                      type="text"
                      value={editItem.title || ""}
                      onChange={(e) => setEditItem({ ...editItem, title: e.target.value })}
                      className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                    />
                  </div>
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Mô tả</label>
                    <textarea
                      value={editItem.description || ""}
                      onChange={(e) => setEditItem({ ...editItem, description: e.target.value })}
                      rows={2}
                      className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                    />
                  </div>
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Phần Thưởng (JSON)</label>
                    <textarea
                      value={typeof editItem.rewards === "string" ? editItem.rewards : JSON.stringify(editItem.rewards, null, 2)}
                      onChange={(e) => setEditItem({ ...editItem, rewards: e.target.value })}
                      rows={3}
                      className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 font-mono text-[11px] mt-1 focus:outline-none focus:border-amber-500/50"
                    />
                  </div>
                </>
              )}

              {type === "dungeons" && (
                <>
                  <div>
                    <label className="text-[11px] font-bold text-zinc-500 font-mono">Tên Phụ Bản</label>
                    <input
                      type="text"
                      value={editItem.name || ""}
                      onChange={(e) => setEditItem({ ...editItem, name: e.target.value })}
                      className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                    />
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-[11px] font-bold text-zinc-500 font-mono">Cấp Yêu Cầu (Rank)</label>
                      <input
                        type="number"
                        min={1}
                        max={9}
                        value={editItem.required_rank || 1}
                        onChange={(e) => setEditItem({ ...editItem, required_rank: Number(e.target.value) })}
                        className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                      />
                    </div>
                    <div>
                      <label className="text-[11px] font-bold text-zinc-500 font-mono">Hồi Chiêu (Giờ)</label>
                      <input
                        type="number"
                        value={editItem.cooldown_hours || 24}
                        onChange={(e) => setEditItem({ ...editItem, cooldown_hours: Number(e.target.value) })}
                        className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                      />
                    </div>
                  </div>
                </>
              )}
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
