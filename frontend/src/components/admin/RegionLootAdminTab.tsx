"use client";

import React, { useEffect, useState, useCallback, useTransition } from "react";
import {
  Gift,
  Search,
  Plus,
  RefreshCw,
  Edit3,
  Trash2,
  X,
  Check,
  Globe,
} from "lucide-react";
import { authFetch } from "@/lib/auth";
import { prepareAdminAction, canPerformAdminAction } from "./actions";
import { useAdminAccess } from "./AdminAccessContext";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

type RegionLootItem = {
  region_id: string;
  region_name: string;
  loot_table_id: string;
  loot_table_name: string;
  monster_type: string;
  min_rank: number;
  max_rank: number;
  chance: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export const RegionLootAdminTab: React.FC<{
  addToast: (msg: string, type: "success" | "error" | "info") => void;
}> = ({ addToast }) => {
  const adminAccess = useAdminAccess();
  const canWrite = canPerformAdminAction(adminAccess.effective_permissions, "updateRegionLoot");
  const canDelete = canPerformAdminAction(adminAccess.effective_permissions, "deleteRegionLoot");

  const [items, setItems] = useState<RegionLootItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(25);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [loading, setLoading] = useState(false);

  // Edit / Create modal
  const [editItem, setEditItem] = useState<{
    region_id: string;
    loot_table_id: string;
    monster_type: string;
    min_rank: number;
    max_rank: number;
    chance: number;
    is_active: boolean;
    _isNew?: boolean;
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
    async (p: number, q: string) => {
      setLoading(true);
      try {
        const params = new URLSearchParams({
          page: String(p),
          page_size: String(pageSize),
        });
        if (q) params.set("search", q);

        const res = await authFetch(`${BACKEND_URL}/api/admin/region-loot-tables?${params.toString()}`);
        if (!res.ok) throw new Error("Không thể tải danh sách bảng rơi đồ");
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
    fetchItems(page, debouncedSearch);
  }, [page, debouncedSearch, fetchItems]);

  const handleSave = async () => {
    if (!editItem) return;
    if (!editItem.region_id || !editItem.loot_table_id || !editItem.monster_type) {
      addToast("Vui lòng điền đầy đủ Khu vực, Bảng loot và Loại quái", "error");
      return;
    }
    setSaving(true);
    try {
      const url = editItem._isNew
        ? `${BACKEND_URL}/api/admin/region-loot-tables`
        : `${BACKEND_URL}/api/admin/region-loot-tables/${encodeURIComponent(editItem.region_id)}/${encodeURIComponent(editItem.loot_table_id)}/${encodeURIComponent(editItem.monster_type)}`;
      const method = editItem._isNew ? "POST" : "PUT";

      const res = await authFetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(editItem),
      });

      if (!res.ok) {
        const errJson = await res.json().catch(() => ({}));
        throw new Error(errJson.error || "Không thể lưu cấu hình");
      }

      addToast("Đã lưu cấu hình loot khu vực thành công", "success");
      setEditItem(null);
      fetchItems(page, debouncedSearch);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      addToast(msg, "error");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (item: RegionLootItem) => {
    const targetLabel = `${item.region_id} · ${item.monster_type} · ${item.loot_table_id}`;
    const prepared = prepareAdminAction("deleteRegionLoot", "xóa cấu hình rơi đồ", targetLabel);
    if (!prepared) return;
    try {
      const res = await authFetch(
        `${BACKEND_URL}/api/admin/region-loot-tables/${encodeURIComponent(item.region_id)}/${encodeURIComponent(item.loot_table_id)}/${encodeURIComponent(item.monster_type)}`,
        {
          method: "DELETE",
          headers: {
            "Content-Type": "application/json",
            "X-Admin-Reason": prepared.reason || "",
            "Idempotency-Key": prepared.idempotencyKey,
          },
          body: JSON.stringify(prepared),
        }
      );
      if (!res.ok) throw new Error("Không thể xóa cấu hình rơi đồ");
      addToast("Đã xóa cấu hình thành công", "success");
      fetchItems(page, debouncedSearch);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      addToast(msg, "error");
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4 space-y-4">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div>
          <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2">
            <Gift className="w-4 h-4 text-emerald-400" />
            Cấu Hình Rơi Đồ Khu Vực (Region Loot Tables)
          </h3>
          <p className="text-xs text-zinc-500 mt-0.5">
            Quản lý bảng liên kết giữa khu vực, chủng loại quái và bảng chiến lợi phẩm (Loot Table).
          </p>
        </div>
        <div className="flex items-center gap-2">
          {canWrite && (
            <button
              onClick={() =>
                setEditItem({
                  region_id: "",
                  loot_table_id: "",
                  monster_type: "Normal",
                  min_rank: 1,
                  max_rank: 5,
                  chance: 1.0,
                  is_active: true,
                  _isNew: true,
                })
              }
              className="flex items-center justify-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-xs font-semibold cursor-pointer"
            >
              <Plus className="w-3.5 h-3.5" />
              Thêm Cấu Hình Loot
            </button>
          )}
          <button
            onClick={() => fetchItems(page, debouncedSearch)}
            disabled={loading}
            className="p-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg cursor-pointer"
            title="Làm mới"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin text-amber-400" : ""}`} />
          </button>
        </div>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
        <input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Tìm theo khu vực, loại quái, bảng loot..."
          className="w-full pl-9 pr-3 py-2 bg-zinc-800/80 border border-zinc-700/80 rounded-lg text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-emerald-500/50"
        />
      </div>

      {/* Table */}
      <div className="rounded-xl border border-zinc-800/80 overflow-hidden">
        {loading && items.length === 0 ? (
          <div className="py-12 text-center text-xs text-zinc-500 flex flex-col items-center gap-2">
            <RefreshCw className="w-5 h-5 animate-spin text-amber-400" />
            Đang tải dữ liệu rơi đồ...
          </div>
        ) : items.length === 0 ? (
          <div className="py-12 text-center text-xs text-zinc-500">
            Không tìm thấy cấu hình rơi đồ nào.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="bg-zinc-900 text-zinc-400 font-mono text-[10px] uppercase border-b border-zinc-800">
                  <th className="px-3 py-2.5">Khu vực</th>
                  <th className="px-3 py-2.5">Loại quái</th>
                  <th className="px-3 py-2.5">Bảng Loot</th>
                  <th className="px-3 py-2.5">Cảnh giới</th>
                  <th className="px-3 py-2.5 text-right">Tỷ lệ</th>
                  <th className="px-3 py-2.5 text-center">Trạng thái</th>
                  <th className="px-3 py-2.5 text-right">Thao tác</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/40">
                {items.map((row) => (
                  <tr key={`${row.region_id}:${row.loot_table_id}:${row.monster_type}`} className="hover:bg-zinc-800/30">
                    <td className="px-3 py-2.5">
                      <div className="font-semibold text-zinc-200">{row.region_name}</div>
                      <div className="font-mono text-[10px] text-zinc-500">{row.region_id}</div>
                    </td>
                    <td className="px-3 py-2.5 font-mono text-amber-400 text-[11px]">
                      {row.monster_type}
                    </td>
                    <td className="px-3 py-2.5">
                      <div className="font-semibold text-zinc-300">{row.loot_table_name}</div>
                      <div className="font-mono text-[10px] text-zinc-500">{row.loot_table_id}</div>
                    </td>
                    <td className="px-3 py-2.5 font-mono text-zinc-400">
                      C{row.min_rank} - C{row.max_rank}
                    </td>
                    <td className="px-3 py-2.5 text-right font-mono text-emerald-400 font-semibold">
                      {(row.chance * 100).toFixed(0)}%
                    </td>
                    <td className="px-3 py-2.5 text-center">
                      <span className={`px-2 py-0.5 rounded text-[10px] font-semibold ${row.is_active ? "text-emerald-400 bg-emerald-500/10" : "text-zinc-500 bg-zinc-800"}`}>
                        {row.is_active ? "BẬT" : "TẮT"}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 text-right">
                      <div className="flex items-center justify-end gap-1">
                        {canWrite && (
                          <button
                            onClick={() => setEditItem({ ...row, _isNew: false })}
                            className="p-1 text-zinc-400 hover:text-amber-400 cursor-pointer rounded"
                            title="Chỉnh sửa"
                          >
                            <Edit3 className="w-3.5 h-3.5" />
                          </button>
                        )}
                        {canDelete && (
                          <button
                            onClick={() => handleDelete(row)}
                            className="p-1 text-zinc-400 hover:text-red-400 cursor-pointer rounded"
                            title="Xóa"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
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
            Tổng số: <span className="text-zinc-300 font-mono">{total}</span> cấu hình
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
                <Gift className="w-4 h-4 text-emerald-400" />
                {editItem._isNew ? "Thêm Cấu Hình Rơi Đồ" : "Sửa Cấu Hình Rơi Đồ"}
              </h3>
              <button onClick={() => setEditItem(null)} className="text-zinc-500 hover:text-zinc-300">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-3 text-xs">
              <div>
                <label className="text-[11px] font-bold text-zinc-400 block mb-1">Mã Khu Vực (Region ID)</label>
                <input
                  disabled={!editItem._isNew}
                  value={editItem.region_id}
                  onChange={(e) => setEditItem({ ...editItem, region_id: e.target.value })}
                  placeholder="Ví dụ: Southern_Border, trung_chau..."
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 disabled:opacity-60"
                />
              </div>

              <div>
                <label className="text-[11px] font-bold text-zinc-400 block mb-1">Bảng Loot (Loot Table ID)</label>
                <input
                  disabled={!editItem._isNew}
                  value={editItem.loot_table_id}
                  onChange={(e) => setEditItem({ ...editItem, loot_table_id: e.target.value })}
                  placeholder="Ví dụ: loot_monster_nam_cuong_normal..."
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 disabled:opacity-60"
                />
              </div>

              <div>
                <label className="text-[11px] font-bold text-zinc-400 block mb-1">Loại Quái (Monster Type)</label>
                <input
                  disabled={!editItem._isNew}
                  value={editItem.monster_type}
                  onChange={(e) => setEditItem({ ...editItem, monster_type: e.target.value })}
                  placeholder="Ví dụ: Normal, Elite, Boss, Beast..."
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 disabled:opacity-60"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[11px] font-bold text-zinc-400 block mb-1">Rank tối thiểu</label>
                  <input
                    type="number"
                    min={1}
                    max={9}
                    value={editItem.min_rank}
                    onChange={(e) => setEditItem({ ...editItem, min_rank: parseInt(e.target.value) || 1 })}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 font-mono"
                  />
                </div>
                <div>
                  <label className="text-[11px] font-bold text-zinc-400 block mb-1">Rank tối đa</label>
                  <input
                    type="number"
                    min={1}
                    max={9}
                    value={editItem.max_rank}
                    onChange={(e) => setEditItem({ ...editItem, max_rank: parseInt(e.target.value) || 9 })}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 font-mono"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[11px] font-bold text-zinc-400 block mb-1">Tỷ lệ rơi (0.0 - 1.0)</label>
                  <input
                    type="number"
                    step="0.05"
                    min={0}
                    max={1}
                    value={editItem.chance}
                    onChange={(e) => setEditItem({ ...editItem, chance: parseFloat(e.target.value) || 0 })}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-zinc-200 font-mono"
                  />
                </div>
                <div className="flex items-center pt-5">
                  <label className="flex items-center gap-2 cursor-pointer text-zinc-300">
                    <input
                      type="checkbox"
                      checked={editItem.is_active}
                      onChange={(e) => setEditItem({ ...editItem, is_active: e.target.checked })}
                      className="rounded bg-zinc-950 border-zinc-800 text-emerald-500"
                    />
                    Kích hoạt hiệu lực
                  </label>
                </div>
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
                className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white rounded-lg text-xs font-bold cursor-pointer"
              >
                {saving ? "Đang lưu..." : "Lưu Cấu Hình"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
