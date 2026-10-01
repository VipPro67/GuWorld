import React, { useEffect, useState, useCallback } from "react";
import { authFetch } from "@/lib/auth";
import { Search, RefreshCw, History, Shield, Sparkles, User, ArrowRight, Copy, Check, Filter } from "lucide-react";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

interface GuHistoryItem {
  id: string;
  instance_id: string;
  gu_id: string;
  rank: number;
  event_type: string;
  actor_id?: string;
  actor_name?: string;
  target_id?: string;
  target_name?: string;
  summary: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

const EVENT_TYPE_BADGES: Record<string, { label: string; bg: string; text: string; border: string }> = {
  CREATED_REFINE: { label: "Hợp Luyện Kế Thừa", bg: "bg-amber-950/80", text: "text-amber-400", border: "border-amber-500/40" },
  CREATED_LOOT: { label: "Bóc Tách Khai Sinh", bg: "bg-yellow-950/80", text: "text-yellow-300", border: "border-yellow-500/40" },
  MARKET_LISTED: { label: "Niêm Yết Bảo Hoàng Thiên", bg: "bg-blue-950/80", text: "text-blue-400", border: "border-blue-500/40" },
  MARKET_SOLD: { label: "Giao Dịch Thành Công", bg: "bg-emerald-950/80", text: "text-emerald-400", border: "border-emerald-500/40" },
  MARKET_CANCELLED: { label: "Hủy Niêm Yết", bg: "bg-zinc-800", text: "text-zinc-400", border: "border-zinc-700" },
  SECT_VAULT_DEPOSITED: { label: "Quyên Góp Tông Môn", bg: "bg-purple-950/80", text: "text-purple-400", border: "border-purple-500/40" },
  SECT_VAULT_WITHDRAWN: { label: "Rút Bảo Khố Tông Môn", bg: "bg-indigo-950/80", text: "text-indigo-400", border: "border-indigo-500/40" },
  DEATH_DROPPED: { label: "Tử Đạo Rơi Rớt", bg: "bg-red-950/80", text: "text-red-400", border: "border-red-500/40" },
  DEATH_LOOTED: { label: "Nhặt Đồ Tử Đạo", bg: "bg-orange-950/80", text: "text-orange-400", border: "border-orange-500/40" },
  STARVED_DEATH: { label: "Chết Đói Tiêu Hủy", bg: "bg-rose-950/90", text: "text-rose-400", border: "border-rose-600/50" },
};

export const GuHistoryAdminTab: React.FC<{
  addToast: (msg: string, type: "success" | "error" | "info") => void;
}> = ({ addToast }) => {
  const [logs, setLogs] = useState<GuHistoryItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [limit] = useState(20);
  const [search, setSearch] = useState("");
  const [searchDebounce, setSearchDebounce] = useState("");
  const [eventTypeFilter, setEventTypeFilter] = useState("");
  const [copiedId, setCopiedId] = useState<string | null>(null);

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearch(searchDebounce);
      setPage(1);
    }, 400);
    return () => clearTimeout(timer);
  }, [searchDebounce]);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString(),
      });
      if (search) params.append("search", search);
      if (eventTypeFilter) params.append("event_type", eventTypeFilter);

      const res = await authFetch(`${BACKEND_URL}/api/admin/gu-history?${params.toString()}`);
      if (res.ok) {
        const body = await res.json();
        setLogs(body.logs || []);
        setTotal(body.total || 0);
      } else {
        addToast("Không thể tải nhật ký lịch sử Cổ trùng", "error");
      }
    } catch {
      addToast("Lỗi kết nối máy chủ", "error");
    } finally {
      setLoading(false);
    }
  }, [page, limit, search, eventTypeFilter, addToast]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(text);
    addToast("Đã chép ID vào clipboard", "info");
    setTimeout(() => setCopiedId(null), 2000);
  };

  const totalPages = Math.max(1, Math.ceil(total / limit));

  return (
    <div className="space-y-4 font-sans text-xs">
      <div className="bg-zinc-900/60 border border-zinc-800/80 rounded-xl p-4 shadow-xl">
        <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-amber-500/10 border border-amber-500/30 flex items-center justify-center">
              <History className="w-4 h-4 text-amber-400" />
            </div>
            <div>
              <h3 className="text-sm font-serif font-bold text-zinc-100 flex items-center gap-2">
                Sổ Phả Hệ & Lịch Sử Tiên Cổ
                <span className="text-[10px] px-2 py-0.5 rounded bg-amber-500/10 border border-amber-500/30 text-amber-400 font-mono">
                  Rank 6+ / Tracked
                </span>
              </h3>
              <p className="text-[11px] text-zinc-400">
                Theo dõi dòng thời gian chi tiết của từng Cổ trùng: Khai sinh, Mua bán Bảo Hoàng Thiên, Kho Tông môn, Tử đạo...
              </p>
            </div>
          </div>
          <button
            onClick={fetchLogs}
            disabled={loading}
            className="px-3 py-1.5 bg-zinc-800 hover:bg-zinc-700 text-zinc-200 border border-zinc-700 rounded-lg flex items-center gap-1.5 text-xs font-semibold cursor-pointer transition-colors"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
            Làm mới
          </button>
        </div>

        {/* Filters */}
        <div className="flex flex-wrap items-center gap-3 mb-4">
          <div className="flex-1 min-w-[240px] relative">
            <Search className="w-3.5 h-3.5 text-zinc-500 absolute left-3 top-2.5" />
            <input
              type="text"
              value={searchDebounce}
              onChange={(e) => setSearchDebounce(e.target.value)}
              placeholder="Tìm kiếm theo Instance ID, Gu ID, Tên Cổ Sư hoặc Mô tả..."
              className="w-full pl-9 pr-3 py-1.5 bg-zinc-950/80 border border-zinc-800 rounded-lg text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-amber-500/50"
            />
          </div>

          <div className="flex items-center gap-1.5">
            <Filter className="w-3.5 h-3.5 text-zinc-500" />
            <select
              value={eventTypeFilter}
              onChange={(e) => {
                setEventTypeFilter(e.target.value);
                setPage(1);
              }}
              className="px-2.5 py-1.5 bg-zinc-950/80 border border-zinc-800 rounded-lg text-xs text-zinc-300 focus:outline-none focus:border-amber-500/50"
            >
              <option value="">-- Tất cả sự kiện --</option>
              <option value="CREATED_REFINE">Hợp Luyện Thành Công</option>
              <option value="MARKET_LISTED">Niêm Yết Bảo Hoàng Thiên</option>
              <option value="MARKET_SOLD">Giao Dịch Thành Công</option>
              <option value="SECT_VAULT_DEPOSITED">Quyên Góp Tông Môn</option>
              <option value="SECT_VAULT_WITHDRAWN">Rút Kho Tông Môn</option>
              <option value="DEATH_DROPPED">Tử Đạo Rơi Rớt</option>
              <option value="STARVED_DEATH">Tiêu Hủy Do Chết Đói</option>
            </select>
          </div>
        </div>

        {/* Log Table */}
        <div className="overflow-x-auto border border-zinc-800 rounded-xl bg-zinc-950/40">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-zinc-800/80 bg-zinc-900/60 text-[11px] font-mono text-zinc-400 uppercase tracking-wider">
                <th className="px-3 py-2.5">Thời Gian</th>
                <th className="px-3 py-2.5">Sự Kiện</th>
                <th className="px-3 py-2.5">Instance ID / Cổ Trùng</th>
                <th className="px-3 py-2.5">Cổ Sư Thực Hiện / Nhận</th>
                <th className="px-3 py-2.5">Mô Tả Lịch Sử</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-800/50">
              {loading && logs.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-zinc-500">
                    Đang tải nhật ký phả hệ Cổ trùng...
                  </td>
                </tr>
              ) : logs.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-zinc-500">
                    Chưa tìm thấy dữ liệu nhật ký lịch sử nào.
                  </td>
                </tr>
              ) : (
                logs.map((log) => {
                  const badge = EVENT_TYPE_BADGES[log.event_type] || {
                    label: log.event_type,
                    bg: "bg-zinc-800",
                    text: "text-zinc-300",
                    border: "border-zinc-700",
                  };
                  return (
                    <tr key={log.id} className="hover:bg-zinc-900/40 transition-colors">
                      <td className="px-3 py-2.5 font-mono text-[11px] text-zinc-400 whitespace-nowrap">
                        {new Date(log.created_at).toLocaleString("vi-VN")}
                      </td>
                      <td className="px-3 py-2.5 whitespace-nowrap">
                        <span className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-mono font-bold border ${badge.bg} ${badge.text} ${badge.border}`}>
                          {badge.label}
                        </span>
                      </td>
                      <td className="px-3 py-2.5">
                        <div className="flex flex-col gap-0.5">
                          <div className="flex items-center gap-1">
                            <span className="font-semibold text-zinc-200">{log.gu_id}</span>
                            <span className="text-[10px] font-mono font-bold px-1 rounded bg-amber-500/20 text-amber-400 border border-amber-500/30">
                              R{log.rank}
                            </span>
                          </div>
                          <button
                            onClick={() => copyToClipboard(log.instance_id)}
                            className="font-mono text-[10px] text-zinc-500 hover:text-amber-400 flex items-center gap-1 cursor-pointer transition-colors text-left"
                            title="Click để copy Instance ID"
                          >
                            {copiedId === log.instance_id ? (
                              <Check className="w-3 h-3 text-emerald-400" />
                            ) : (
                              <Copy className="w-3 h-3 text-zinc-600" />
                            )}
                            <span className="truncate max-w-[160px]">{log.instance_id}</span>
                          </button>
                        </div>
                      </td>
                      <td className="px-3 py-2.5 whitespace-nowrap">
                        <div className="flex items-center gap-1.5">
                          {log.actor_name ? (
                            <span className="font-semibold text-amber-400/90">{log.actor_name}</span>
                          ) : (
                            <span className="text-zinc-600">--</span>
                          )}
                          {log.target_name && (
                            <>
                              <ArrowRight className="w-3 h-3 text-zinc-600" />
                              <span className="font-semibold text-emerald-400/90">{log.target_name}</span>
                            </>
                          )}
                        </div>
                      </td>
                      <td className="px-3 py-2.5 text-zinc-300">
                        <div>{log.summary}</div>
                        {log.metadata && Object.keys(log.metadata).length > 0 && (
                          <div className="text-[10px] font-mono text-zinc-500 mt-0.5 truncate max-w-[320px]">
                            {JSON.stringify(log.metadata)}
                          </div>
                        )}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="flex items-center justify-between mt-4 text-xs text-zinc-400">
          <div>
            Hiển thị <span className="text-zinc-200 font-bold">{logs.length}</span> /{" "}
            <span className="text-zinc-200 font-bold">{total}</span> nhật ký
          </div>
          <div className="flex items-center gap-2">
            <span>Trang {page} / {totalPages}</span>
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="px-2.5 py-1 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-300 disabled:opacity-40 hover:bg-zinc-700 cursor-pointer"
            >
              Trước
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="px-2.5 py-1 bg-zinc-800 border border-zinc-700 rounded-lg text-zinc-300 disabled:opacity-40 hover:bg-zinc-700 cursor-pointer"
            >
              Sau
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
