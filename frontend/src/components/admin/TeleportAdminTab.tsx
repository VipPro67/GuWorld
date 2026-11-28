"use client";

import React, { useEffect, useState, useCallback, useTransition } from "react";
import {
  Compass,
  Search,
  RefreshCw,
  Edit3,
  X,
  MapPin,
  Coins,
  Shield,
  Building2,
} from "lucide-react";
import { authFetch } from "@/lib/auth";
import { prepareAdminAction, canPerformAdminAction } from "./actions";
import { useAdminAccess } from "./AdminAccessContext";
import { formatNumber } from "@/lib/utils";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

type TeleportItem = {
  id: string;
  name: string;
  owner_id: string;
  owner_name: string;
  sect_id: string;
  sect_name: string;
  region_id: string;
  region_name: string;
  x: number;
  y: number;
  tier: string;
  max_range: number;
  is_cross_region: boolean;
  direction_type: string;
  access_policy: string;
  fee_enabled: boolean;
  fee_currency: string;
  fee_amount: number;
  vault_balance: number;
  current_durability: number;
  max_durability: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export const TeleportAdminTab: React.FC<{
  addToast: (msg: string, type: "success" | "error" | "info") => void;
}> = ({ addToast }) => {
  const adminAccess = useAdminAccess();
  const canWrite = canPerformAdminAction(adminAccess.effective_permissions, "updateTeleportArray");

  const [items, setItems] = useState<TeleportItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [regionFilter, setRegionFilter] = useState("ALL");
  const [loading, setLoading] = useState(false);

  // Edit modal
  const [editItem, setEditItem] = useState<{
    id: string;
    name: string;
    fee_amount: number;
    fee_enabled: boolean;
    is_active: boolean;
    max_range: number;
    access_policy: string;
  } | null>(null);
  const [saving, setSaving] = useState(false);

  const [, startTransition] = useTransition();

  useEffect(() => {
    const handler = setTimeout(() => {
      startTransition(() => {
        setDebouncedSearch(search);
        setPage(1);
      });
    }, 400);
    return () => clearTimeout(handler);
  }, [search]);

  const fetchItems = useCallback(
    async (p: number, q: string, reg: string) => {
      setLoading(true);
      try {
        const params = new URLSearchParams({
          page: String(p),
          page_size: String(pageSize),
        });
        if (q) params.set("search", q);
        if (reg && reg !== "ALL") params.set("region_id", reg);

        const res = await authFetch(`${BACKEND_URL}/api/admin/teleport-arrays?${params.toString()}`);
        if (!res.ok) throw new Error("Không thể tải danh sách truyền tống trận");
        const json = await res.json();
        setItems(json.data || []);
        setTotal(json.total || 0);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        addToast(msg, "error");
      } finally {
        setLoading(false);
      }
    },
    [pageSize, addToast]
  );

  useEffect(() => {
    fetchItems(page, debouncedSearch, regionFilter);
  }, [page, debouncedSearch, regionFilter, fetchItems]);

  const handleSave = async () => {
    if (!editItem) return;
    setSaving(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/teleport-arrays/${editItem.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(editItem),
      });

      if (!res.ok) {
        const errJson = await res.json().catch(() => ({}));
        throw new Error(errJson.error || "Không thể cập nhật truyền tống trận");
      }

      addToast("Cập nhật truyền tống trận thành công", "success");
      setEditItem(null);
      fetchItems(page, debouncedSearch, regionFilter);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      addToast(msg, "error");
    } finally {
      setSaving(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4 space-y-4">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div>
          <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
            <Compass className="w-4 h-4 text-cyan-400" />
            Mạng Lưới Truyền Tống Trận (Teleport Arrays)
          </h3>
          <p className="text-xs text-zinc-500 mt-0.5">
            Quản lý các pháp trận dịch chuyển thế giới, thiết lập phí di chuyển và kiểm soát bảo trì.
          </p>
        </div>
        <button
          onClick={() => fetchItems(page, debouncedSearch, regionFilter)}
          disabled={loading}
          className="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 text-zinc-300 rounded-lg text-xs font-mono flex items-center gap-1.5 self-start sm:self-auto cursor-pointer"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin text-amber-400" : ""}`} />
          Làm mới
        </button>
      </div>

      {/* Search & Filter */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div className="relative sm:col-span-2">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Tìm theo tên trận pháp, khu vực, người sở hữu, tông môn..."
            className="w-full pl-9 pr-3 py-2 bg-zinc-800/80 border border-zinc-700/80 rounded-lg text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-cyan-500/50"
          />
        </div>
        <div>
          <input
            value={regionFilter === "ALL" ? "" : regionFilter}
            onChange={(e) => {
              setRegionFilter(e.target.value || "ALL");
              setPage(1);
            }}
            placeholder="Lọc ID khu vực (ví dụ: Southern_Border)..."
            className="w-full px-3 py-2 bg-zinc-800/80 border border-zinc-700/80 rounded-lg text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-cyan-500/50"
          />
        </div>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-zinc-800/80 overflow-hidden">
        {loading && items.length === 0 ? (
          <div className="py-12 text-center text-xs text-zinc-500 flex flex-col items-center gap-2">
            <RefreshCw className="w-5 h-5 animate-spin text-amber-400" />
            Đang tải danh sách truyền tống trận...
          </div>
        ) : items.length === 0 ? (
          <div className="py-12 text-center text-xs text-zinc-500">
            Không tìm thấy truyền tống trận nào.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="bg-zinc-900 text-zinc-400 font-mono text-[10px] uppercase border-b border-zinc-800">
                  <th className="px-3 py-2.5">Trận pháp</th>
                  <th className="px-3 py-2.5">Khu vực / Tọa độ</th>
                  <th className="px-3 py-2.5">Chủ sở hữu / Tông môn</th>
                  <th className="px-3 py-2.5">Phân cấp & Tầm</th>
                  <th className="px-3 py-2.5 text-right">Phí dịch chuyển</th>
                  <th className="px-3 py-2.5 text-center">Độ bền</th>
                  <th className="px-3 py-2.5 text-center">Trạng thái</th>
                  <th className="px-3 py-2.5 text-right">Thao tác</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/40">
                {items.map((row) => (
                  <tr key={row.id} className="hover:bg-zinc-800/30">
                    <td className="px-3 py-2.5">
                      <div className="font-semibold text-zinc-200 flex items-center gap-1.5">
                        <Compass className="w-3.5 h-3.5 text-cyan-400" />
                        <span>{row.name}</span>
                      </div>
                      <span className="text-[10px] text-zinc-600 font-mono">{row.id}</span>
                    </td>
                    <td className="px-3 py-2.5">
                      <div className="text-zinc-300 font-medium">{row.region_name}</div>
                      <div className="font-mono text-[10px] text-amber-400">
                        ({row.x}, {row.y})
                      </div>
                    </td>
                    <td className="px-3 py-2.5">
                      <div className="text-zinc-300">{row.owner_name}</div>
                      {row.sect_name && (
                        <div className="text-[10px] text-zinc-500 flex items-center gap-1">
                          <Building2 className="w-3 h-3" />
                          {row.sect_name}
                        </div>
                      )}
                    </td>
                    <td className="px-3 py-2.5 font-mono text-zinc-400 text-[11px]">
                      {row.tier} · {row.max_range} ô
                      {row.is_cross_region && (
                        <span className="block text-[10px] text-purple-400">Xuyên Vực</span>
                      )}
                    </td>
                    <td className="px-3 py-2.5 text-right font-mono">
                      {row.fee_enabled ? (
                        <span className="text-yellow-400 font-semibold">
                          {formatNumber(Number(row.fee_amount), 0)} NT
                        </span>
                      ) : (
                        <span className="text-zinc-500">Miễn phí</span>
                      )}
                    </td>
                    <td className="px-3 py-2.5 text-center font-mono text-[11px] text-zinc-300">
                      {row.current_durability} / {row.max_durability}
                    </td>
                    <td className="px-3 py-2.5 text-center">
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                          row.is_active ? "text-emerald-400 bg-emerald-500/10" : "text-rose-400 bg-rose-500/10"
                        }`}
                      >
                        {row.is_active ? "HOẠT ĐỘNG" : "BẢO TRÌ"}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-right">
                      {canWrite && (
                        <button
                          onClick={() =>
                            setEditItem({
                              id: row.id,
                              name: row.name,
                              fee_amount: row.fee_amount,
                              fee_enabled: row.fee_enabled,
                              is_active: row.is_active,
                              max_range: row.max_range,
                              access_policy: row.access_policy,
                            })
                          }
                          className="p-1 text-zinc-400 hover:text-cyan-400 cursor-pointer rounded"
                          title="Cấu hình trận pháp"
                        >
                          <Edit3 className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        <div className="px-3 py-2.5 border-t border-zinc-800 flex items-center justify-between text-xs text-zinc-500">
          <div>
            Tổng số: <span className="text-zinc-300 font-mono">{total}</span> truyền tống trận
          </div>
          <div className="flex items-center gap-2">
            <span>
              Trang {page}/{totalPages}
            </span>
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-2 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 text-zinc-300 rounded cursor-pointer"
            >
              Trước
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-2 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 text-zinc-300 rounded cursor-pointer"
            >
              Sau
            </button>
          </div>
        </div>
      </div>

      {/* Edit Modal */}
      {editItem && (
        <div className="fixed inset-0 z-50 bg-black/70 flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl max-w-md w-full p-5 space-y-4 shadow-2xl">
            <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
              <h3 className="text-sm font-bold text-zinc-200 flex items-center gap-2">
                <Compass className="w-4 h-4 text-cyan-400" />
                Cấu Hình Truyền Tống Trận
              </h3>
              <button onClick={() => setEditItem(null)} className="text-zinc-500 hover:text-zinc-300">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-3 text-xs">
              <div>
                <label className="text-[11px] font-bold text-zinc-400 block mb-1">Tên trận pháp</label>
                <input
                  value={editItem.name}
                  onChange={(e) => setEditItem({ ...editItem, name: e.target.value })}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200"
                />
              </div>

              <div>
                <label className="text-[11px] font-bold text-zinc-400 block mb-1">Chính sách truy cập</label>
                <select
                  value={editItem.access_policy}
                  onChange={(e) => setEditItem({ ...editItem, access_policy: e.target.value })}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200"
                >
                  <option value="PUBLIC">Công khai (Tất cả người chơi)</option>
                  <option value="SECT_ONLY">Nội bộ Tông Môn</option>
                  <option value="PRIVATE">Riêng tư (Chỉ chủ nhân)</option>
                </select>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[11px] font-bold text-zinc-400 block mb-1">Phí Tiên Thạch</label>
                  <input
                    type="number"
                    min={0}
                    value={editItem.fee_amount}
                    onChange={(e) => setEditItem({ ...editItem, fee_amount: parseInt(e.target.value) || 0 })}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 font-mono"
                  />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-400 block mb-1">Tầm phát xạ (ô)</label>
                  <input
                    type="number"
                    min={10}
                    value={editItem.max_range}
                    onChange={(e) => setEditItem({ ...editItem, max_range: parseInt(e.target.value) || 1000 })}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 font-mono"
                  />
                </div>
              </div>

              <div className="flex gap-4 pt-2">
                <label className="flex items-center gap-2 cursor-pointer text-zinc-300">
                  <input
                    type="checkbox"
                    checked={editItem.fee_enabled}
                    onChange={(e) => setEditItem({ ...editItem, fee_enabled: e.target.checked })}
                    className="rounded bg-zinc-950 border-zinc-800 text-cyan-500"
                  />
                  Thu phí dịch chuyển
                </label>
                <label className="flex items-center gap-2 cursor-pointer text-zinc-300">
                  <input
                    type="checkbox"
                    checked={editItem.is_active}
                    onChange={(e) => setEditItem({ ...editItem, is_active: e.target.checked })}
                    className="rounded bg-zinc-950 border-zinc-800 text-cyan-500"
                  />
                  Đang hoạt động (Active)
                </label>
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2 border-t border-zinc-800">
              <button
                onClick={() => setEditItem(null)}
                className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs font-semibold cursor-pointer"
              >
                Hủy
              </button>
              <button
                onClick={handleSave}
                disabled={saving}
                className="px-4 py-2 bg-cyan-600 hover:bg-cyan-500 disabled:opacity-50 text-white rounded-lg text-xs font-bold cursor-pointer"
              >
                {saving ? "Đang lưu..." : "Lưu Cập Nhật"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
