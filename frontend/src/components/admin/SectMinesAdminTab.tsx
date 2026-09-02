import React, { useEffect, useState, useCallback } from "react";
import { Shield, RefreshCw, Edit3, X } from "lucide-react";

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

export const SectMinesAdminTab: React.FC<{
  addToast: (msg: string, type: "success" | "error" | "info") => void;
}> = ({ addToast }) => {
  const [mines, setMines] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [editMine, setEditMine] = useState<any | null>(null);

  const fetchSectMines = useCallback(async () => {
    setLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/sect-mines`);
      if (res.ok) {
        setMines(await res.json());
      } else {
        addToast("Không thể tải danh sách mỏ tông môn", "error");
      }
    } catch {
      addToast("Lỗi kết nối mỏ tông môn", "error");
    } finally {
      setLoading(false);
    }
  }, [addToast]);

  useEffect(() => {
    fetchSectMines();
  }, [fetchSectMines]);

  const handleSave = async () => {
    if (!editMine || !editMine.id) return;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/sect-mines/${editMine.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          occupier_sect_id: editMine.occupier_sect_id,
          rarity: editMine.rarity,
          pending_funds: Number(editMine.pending_funds || 0),
        }),
      });
      if (res.ok) {
        addToast("Cập nhật mỏ tông môn thành công", "success");
        setEditMine(null);
        fetchSectMines();
      } else {
        addToast("Cập nhật thất bại", "error");
      }
    } catch {
      addToast("Lỗi kết nối server", "error");
    }
  };

  return (
    <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-5 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
          <Shield className="w-4 h-4 text-purple-400" />
          Quản Lý Mỏ Tông Môn (Sect Mines - 5 Mỏ Lớn)
        </h3>
        <button
          onClick={fetchSectMines}
          className="p-1.5 hover:bg-zinc-800 rounded-lg text-zinc-400 hover:text-zinc-200 transition-colors cursor-pointer"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
        </button>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse text-xs">
          <thead>
            <tr className="border-b border-zinc-800 text-zinc-400 font-mono text-[11px]">
              <th className="py-2.5 px-3">Mã mỏ</th>
              <th className="py-2.5 px-3">Tên mỏ</th>
              <th className="py-2.5 px-3">Phẩm chất (Rarity)</th>
              <th className="py-2.5 px-3">Tông môn chiếm giữ</th>
              <th className="py-2.5 px-3">Thời gian bảo vệ Khiên</th>
              <th className="py-2.5 px-3">Quỹ linh thạch tích lũy</th>
              <th className="py-2.5 px-3 text-right">Thao tác</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-800/50 text-zinc-300">
            {mines.length === 0 ? (
              <tr>
                <td colSpan={7} className="text-center py-6 text-zinc-500 italic">
                  {loading ? "Đang tải dữ liệu..." : "Chưa có mỏ tông môn"}
                </td>
              </tr>
            ) : (
              mines.map((m) => (
                <tr key={m.id} className="hover:bg-zinc-900/50">
                  <td className="py-2.5 px-3 font-mono text-amber-400 font-bold">{m.id}</td>
                  <td className="py-2.5 px-3 font-semibold text-zinc-200">{m.name}</td>
                  <td className="py-2.5 px-3 uppercase text-purple-400 font-bold font-mono">{m.rarity}</td>
                  <td className="py-2.5 px-3 text-emerald-400 font-medium">
                    {m.occupier_sect_name ? `${m.occupier_sect_name} (${m.occupier_sect_id})` : <span className="text-zinc-500 italic">Chưa ai chiếm</span>}
                  </td>
                  <td className="py-2.5 px-3 font-mono text-zinc-400">{m.shield_until || "Không có"}</td>
                  <td className="py-2.5 px-3 font-mono text-sky-400">+{Number(m.pending_funds).toLocaleString()}</td>
                  <td className="py-2.5 px-3 text-right">
                    <button
                      onClick={() => setEditMine({ ...m })}
                      className="p-1 hover:bg-zinc-800 rounded text-zinc-400 hover:text-amber-400 cursor-pointer"
                    >
                      <Edit3 className="w-3.5 h-3.5" />
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {editMine && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm">
          <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-2xl p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-bold text-zinc-100 font-serif">Sửa mỏ tông môn: {editMine.name}</h3>
              <button onClick={() => setEditMine(null)} className="p-1 hover:bg-zinc-900 rounded-lg cursor-pointer">
                <X className="w-4 h-4 text-zinc-500" />
              </button>
            </div>

            <div className="space-y-3 text-xs">
              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Sect ID chiếm giữ (Để trống = Hoang)</label>
                <input
                  type="text"
                  value={editMine.occupier_sect_id || ""}
                  onChange={(e) => setEditMine({ ...editMine, occupier_sect_id: e.target.value })}
                  placeholder="sect_uuid..."
                  className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                />
              </div>

              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Phẩm chất (Rarity)</label>
                <input
                  type="text"
                  value={editMine.rarity || ""}
                  onChange={(e) => setEditMine({ ...editMine, rarity: e.target.value })}
                  placeholder="legendary, epic, rare..."
                  className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                />
              </div>

              <div>
                <label className="text-[11px] font-bold text-zinc-500 font-mono">Quỹ tích lũy (Pending Funds)</label>
                <input
                  type="number"
                  value={editMine.pending_funds || 0}
                  onChange={(e) => setEditMine({ ...editMine, pending_funds: e.target.value })}
                  className="w-full bg-zinc-900 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 mt-1 focus:outline-none focus:border-amber-500/50"
                />
              </div>
            </div>

            <div className="flex gap-3 justify-end pt-2">
              <button
                onClick={() => setEditMine(null)}
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
