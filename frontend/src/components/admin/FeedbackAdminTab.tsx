"use client";

import React, { useEffect, useState, useCallback, useTransition } from "react";
import {
  MessageSquare,
  Search,
  RefreshCw,
  CheckCircle,
  Clock,
  AlertTriangle,
  XCircle,
  ThumbsUp,
  MessageCircle,
  Award,
  Trash2,
  Edit3,
  X,
  Send,
} from "lucide-react";
import { authFetch } from "@/lib/auth";
import { prepareAdminAction, canPerformAdminAction } from "./actions";
import { useAdminAccess } from "./AdminAccessContext";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

type FeedbackItem = {
  id: string;
  user_id: string;
  username: string;
  title: string;
  description: string;
  category: string;
  status: "pending" | "active" | "in_progress" | "completed" | "rejected";
  total_votes: number;
  comment_count: number;
  developer_response: string;
  rewarded: boolean;
  created_at: string;
};

const CATEGORIES = [
  { id: "ALL", label: "Tất cả danh mục" },
  { id: "BUG", label: "Báo lỗi (Bug)" },
  { id: "GU", label: "Cổ Trùng" },
  { id: "ITEM", label: "Vật phẩm" },
  { id: "RECIPE", label: "Cổ phương" },
  { id: "BALANCE", label: "Cân bằng" },
  { id: "FEATURE", label: "Đề xuất tính năng" },
  { id: "OTHER", label: "Khác" },
];

const STATUSES = [
  { id: "ALL", label: "Tất cả trạng thái" },
  { id: "pending", label: "Chờ duyệt (Pending)" },
  { id: "active", label: "Đã duyệt (Active)" },
  { id: "in_progress", label: "Đang xử lý" },
  { id: "completed", label: "Hoàn tất" },
  { id: "rejected", label: "Từ chối" },
];

export const FeedbackAdminTab: React.FC<{
  addToast: (msg: string, type: "success" | "error" | "info") => void;
}> = ({ addToast }) => {
  const adminAccess = useAdminAccess();
  const canWrite = canPerformAdminAction(adminAccess.effective_permissions, "updateFeedbackStatus");
  const canDelete = canPerformAdminAction(adminAccess.effective_permissions, "deleteFeedback");

  const [feedbacks, setFeedbacks] = useState<FeedbackItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(15);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [category, setCategory] = useState("ALL");
  const [status, setStatus] = useState("ALL");
  const [loading, setLoading] = useState(false);
  const [, startTransition] = useTransition();

  // Status update modal
  const [selectedItem, setSelectedItem] = useState<FeedbackItem | null>(null);
  const [editStatus, setEditStatus] = useState<FeedbackItem["status"]>("active");
  const [devResponse, setDevResponse] = useState("");
  const [rewardStones, setRewardStones] = useState<number>(0);
  const [saving, setSaving] = useState(false);

  // Debounce search
  useEffect(() => {
    const handler = setTimeout(() => {
      startTransition(() => {
        setDebouncedSearch(search);
        setPage(1);
      });
    }, 400);
    return () => clearTimeout(handler);
  }, [search]);

  const fetchFeedbacks = useCallback(
    async (p: number, q: string, cat: string, st: string) => {
      setLoading(true);
      try {
        const params = new URLSearchParams({
          page: String(p),
          page_size: String(pageSize),
        });
        if (q) params.set("search", q);
        if (cat && cat !== "ALL") params.set("category", cat);
        if (st && st !== "ALL") params.set("status", st);

        const res = await authFetch(`${BACKEND_URL}/api/admin/feedback?${params.toString()}`);
        if (!res.ok) throw new Error("Không thể tải danh sách phản hồi");
        const json = await res.json();
        setFeedbacks(json.data || []);
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
    fetchFeedbacks(page, debouncedSearch, category, status);
  }, [page, debouncedSearch, category, status, fetchFeedbacks]);

  const handleOpenStatusModal = (item: FeedbackItem) => {
    setSelectedItem(item);
    setEditStatus(item.status);
    setDevResponse(item.developer_response || "");
    setRewardStones(0);
  };

  const handleSaveStatus = async () => {
    if (!selectedItem) return;
    setSaving(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/feedback/${selectedItem.id}/status`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          status: editStatus,
          developer_response: devResponse,
          reward_stones: rewardStones,
        }),
      });
      if (!res.ok) {
        const errJson = await res.json().catch(() => ({}));
        throw new Error(errJson.error || "Không thể cập nhật phản hồi");
      }
      addToast("Cập nhật phản hồi thành công", "success");
      setSelectedItem(null);
      fetchFeedbacks(page, debouncedSearch, category, status);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      addToast(msg, "error");
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteFeedback = async (item: FeedbackItem) => {
    const prepared = prepareAdminAction("deleteFeedback", "xóa phản hồi", item.title);
    if (!prepared) return;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/feedback/${item.id}`, {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
          "X-Admin-Reason": prepared.reason || "",
          "Idempotency-Key": prepared.idempotencyKey,
        },
        body: JSON.stringify(prepared),
      });
      if (!res.ok) throw new Error("Không thể xóa phản hồi");
      addToast("Đã xóa phản hồi thành công", "success");
      fetchFeedbacks(page, debouncedSearch, category, status);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      addToast(msg, "error");
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="space-y-4">
      {/* Header controls */}
      <div className="bg-zinc-900/60 border border-zinc-800/80 rounded-xl p-4 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <h2 className="text-base font-serif font-bold text-zinc-100 flex items-center gap-2">
            <MessageSquare className="w-5 h-5 text-amber-400" />
            Quản Lý Ý Kiến & Báo Cáo Lỗi
          </h2>
          <p className="text-xs text-zinc-400 mt-0.5">
            Xem, phê duyệt, phản hồi đóng góp và khen thưởng Tiên Thạch cho người chơi.
          </p>
        </div>
        <button
          onClick={() => fetchFeedbacks(page, debouncedSearch, category, status)}
          disabled={loading}
          className="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-50 text-zinc-300 rounded-lg text-xs font-mono flex items-center gap-1.5 self-start md:self-auto cursor-pointer"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin text-amber-400" : ""}`} />
          Làm mới
        </button>
      </div>

      {/* Filters & Search */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
          <input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Tìm theo tiêu đề, nội dung, người gửi..."
            className="w-full bg-zinc-900/80 border border-zinc-800 rounded-lg pl-9 pr-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50"
          />
        </div>
        <div>
          <select
            value={category}
            onChange={(e) => {
              setCategory(e.target.value);
              setPage(1);
            }}
            className="w-full bg-zinc-900/80 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-300 focus:outline-none focus:border-amber-500/50"
          >
            {CATEGORIES.map((c) => (
              <option key={c.id} value={c.id}>
                {c.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
            className="w-full bg-zinc-900/80 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-300 focus:outline-none focus:border-amber-500/50"
          >
            {STATUSES.map((s) => (
              <option key={s.id} value={s.id}>
                {s.label}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Table view */}
      <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-xl overflow-hidden">
        {loading && feedbacks.length === 0 ? (
          <div className="py-16 text-center text-xs text-zinc-500 flex flex-col items-center gap-2">
            <RefreshCw className="w-5 h-5 animate-spin text-amber-400" />
            Đang tải danh sách phản hồi...
          </div>
        ) : feedbacks.length === 0 ? (
          <div className="py-16 text-center text-xs text-zinc-500">
            Không tìm thấy ý kiến hoặc báo lỗi nào phù hợp.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="bg-zinc-900/90 border-b border-zinc-800 text-zinc-400 font-mono text-[11px] uppercase">
                  <th className="px-4 py-3">Phản hồi</th>
                  <th className="px-3 py-3">Danh mục</th>
                  <th className="px-3 py-3">Trạng thái</th>
                  <th className="px-3 py-3">Người gửi</th>
                  <th className="px-3 py-3 text-center">Bình chọn</th>
                  <th className="px-3 py-3">Ngày gửi</th>
                  <th className="px-4 py-3 text-right">Thao tác</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/50">
                {feedbacks.map((f) => {
                  const statusColors: Record<FeedbackItem["status"], string> = {
                    pending: "text-amber-400 bg-amber-500/10 border-amber-500/30",
                    active: "text-emerald-400 bg-emerald-500/10 border-emerald-500/30",
                    in_progress: "text-cyan-400 bg-cyan-500/10 border-cyan-500/30",
                    completed: "text-blue-400 bg-blue-500/10 border-blue-500/30",
                    rejected: "text-rose-400 bg-rose-500/10 border-rose-500/30",
                  };
                  return (
                    <tr key={f.id} className="hover:bg-zinc-800/30 transition-colors">
                      <td className="px-4 py-3 max-w-xs">
                        <div className="font-semibold text-zinc-200 truncate">{f.title}</div>
                        <div className="text-[11px] text-zinc-400 line-clamp-1 mt-0.5">{f.description}</div>
                        {f.developer_response && (
                          <div className="text-[10px] text-amber-400/90 mt-1 italic line-clamp-1">
                            Phản hồi: {f.developer_response}
                          </div>
                        )}
                      </td>
                      <td className="px-3 py-3 font-mono text-zinc-400 text-[11px]">
                        <span className="px-2 py-0.5 rounded bg-zinc-800 border border-zinc-700/60">
                          {f.category}
                        </span>
                      </td>
                      <td className="px-3 py-3">
                        <span className={`px-2 py-0.5 rounded-full text-[10px] font-semibold border ${statusColors[f.status] || ""}`}>
                          {f.status}
                        </span>
                        {f.rewarded && (
                          <span className="ml-1 inline-flex items-center text-[10px] text-yellow-400" title="Đã trao thưởng">
                            <Award className="w-3 h-3 inline" />
                          </span>
                        )}
                      </td>
                      <td className="px-3 py-3 font-mono text-zinc-300">
                        {f.username}
                        <span className="block text-[10px] text-zinc-600 truncate max-w-[100px]">{f.user_id}</span>
                      </td>
                      <td className="px-3 py-3 text-center text-zinc-400 font-mono">
                        <div className="flex items-center justify-center gap-2">
                          <span className="inline-flex items-center gap-0.5 text-zinc-300">
                            <ThumbsUp className="w-3 h-3 text-zinc-500" />
                            {f.total_votes}
                          </span>
                          <span className="inline-flex items-center gap-0.5 text-zinc-500">
                            <MessageCircle className="w-3 h-3" />
                            {f.comment_count}
                          </span>
                        </div>
                      </td>
                      <td className="px-3 py-3 text-zinc-500 text-[11px]">
                        {new Date(f.created_at).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" })}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          {canWrite && (
                            <button
                              onClick={() => handleOpenStatusModal(f)}
                              className="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 hover:text-amber-400 rounded text-xs font-mono transition-colors cursor-pointer flex items-center gap-1"
                              title="Cập nhật trạng thái / Phản hồi"
                            >
                              <Edit3 className="w-3 h-3" />
                              Xử lý
                            </button>
                          )}
                          {canDelete && (
                            <button
                              onClick={() => handleDeleteFeedback(f)}
                              className="p-1 text-zinc-500 hover:text-red-400 transition-colors cursor-pointer rounded"
                              title="Xóa phản hồi"
                            >
                              <Trash2 className="w-3.5 h-3.5" />
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        <div className="px-4 py-3 border-t border-zinc-800/80 flex items-center justify-between text-xs text-zinc-500">
          <div>
            Tổng số: <span className="font-mono text-zinc-300">{total}</span> phản hồi
          </div>
          <div className="flex items-center gap-2">
            <span>
              Trang {page} / {totalPages}
            </span>
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 text-zinc-300 rounded cursor-pointer"
            >
              Trước
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 text-zinc-300 rounded cursor-pointer"
            >
              Sau
            </button>
          </div>
        </div>
      </div>

      {/* Detail / Status Modal */}
      {selectedItem && (
        <div className="fixed inset-0 z-50 bg-black/70 flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl max-w-lg w-full p-5 space-y-4 shadow-2xl">
            <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
              <h3 className="text-sm font-bold text-zinc-200 flex items-center gap-2">
                <Edit3 className="w-4 h-4 text-amber-400" />
                Xử Lý Phản Hồi: {selectedItem.title}
              </h3>
              <button onClick={() => setSelectedItem(null)} className="text-zinc-500 hover:text-zinc-300">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="bg-zinc-950 p-3 rounded-lg border border-zinc-800/80 space-y-1.5 text-xs">
              <div className="text-zinc-400">
                <span className="font-semibold text-zinc-300">Tác giả:</span> {selectedItem.username} (ID: {selectedItem.user_id})
              </div>
              <div className="text-zinc-400">
                <span className="font-semibold text-zinc-300">Danh mục:</span> {selectedItem.category}
              </div>
              <div className="text-zinc-300 font-mono text-[11px] whitespace-pre-wrap bg-zinc-900/60 p-2 rounded">
                {selectedItem.description}
              </div>
            </div>

            <div className="space-y-3">
              <div>
                <label className="text-[11px] font-bold text-zinc-400 block mb-1">Trạng thái mới</label>
                <select
                  value={editStatus}
                  onChange={(e) => setEditStatus(e.target.value as FeedbackItem["status"])}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200"
                >
                  <option value="pending">Chờ duyệt (Pending)</option>
                  <option value="active">Đã duyệt hiển thị (Active)</option>
                  <option value="in_progress">Đang sửa / xử lý (In Progress)</option>
                  <option value="completed">Đã hoàn tất / Đã giải quyết (Completed)</option>
                  <option value="rejected">Từ chối / Không hợp lệ (Rejected)</option>
                </select>
              </div>

              <div>
                <label className="text-[11px] font-bold text-zinc-400 block mb-1">Phản hồi của Nhà Phát Hành</label>
                <textarea
                  value={devResponse}
                  onChange={(e) => setDevResponse(e.target.value)}
                  placeholder="Nhập thông điệp gửi tới người chơi..."
                  rows={3}
                  className="w-full bg-zinc-950 border border-zinc-800 rounded-lg p-2.5 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50"
                />
              </div>

              {!selectedItem.rewarded && (
                <div>
                  <label className="text-[11px] font-bold text-zinc-400 block mb-1 flex items-center gap-1.5">
                    <Award className="w-3.5 h-3.5 text-yellow-400" />
                    Thưởng Nguyên Thạch (Tùy chọn)
                  </label>
                  <input
                    type="number"
                    min={0}
                    value={rewardStones}
                    onChange={(e) => setRewardStones(Math.max(0, parseInt(e.target.value) || 0))}
                    placeholder="Số lượng nguyên thạch..."
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-200 font-mono"
                  />
                  <span className="text-[10px] text-zinc-500 block mt-1">
                    Cộng thẳng vào tài khoản của người báo lỗi khi lưu.
                  </span>
                </div>
              )}
            </div>

            <div className="flex justify-end gap-2 pt-2 border-t border-zinc-800">
              <button
                onClick={() => setSelectedItem(null)}
                className="px-4 py-2 bg-zinc-800 hover:bg-zinc-700 text-zinc-300 rounded-lg text-xs font-semibold cursor-pointer"
              >
                Hủy
              </button>
              <button
                onClick={handleSaveStatus}
                disabled={saving}
                className="px-4 py-2 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white rounded-lg text-xs font-bold cursor-pointer flex items-center gap-1.5"
              >
                <Send className="w-3.5 h-3.5" />
                {saving ? "Đang lưu..." : "Lưu Thay Đổi"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
