import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import {
  History,
  Filter,
  CheckCircle,
  XCircle,
  ChevronLeft,
  ChevronRight,
  RefreshCw,
  BookOpen,
  Maximize2,
  Minimize2,
  Calendar,
  Search
} from "lucide-react";

export const AuditLogsView: React.FC = () => {
  const { auditLogs, auditLogsLoading, fetchAuditLogs, guDictionary, itemDictionary, monsterDictionary } = useGameStore();

  const formatRequestSummary = (msg: string): string => {
    if (!msg) return "";
    let formatted = msg;
    Object.keys(guDictionary || {}).forEach((id) => {
      const name = guDictionary[id]?.name;
      if (name) formatted = formatted.split(id).join(name);
    });
    Object.keys(itemDictionary || {}).forEach((id) => {
      const name = itemDictionary[id]?.name;
      if (name) formatted = formatted.split(id).join(name);
    });
    Object.keys(monsterDictionary || {}).forEach((id) => {
      const name = monsterDictionary[id]?.name;
      if (name) formatted = formatted.split(id).join(name);
    });
    return formatted;
  };

  const [selectedAction, setSelectedAction] = useState<string>("");
  const [selectedSuccess, setSelectedSuccess] = useState<string>("");
  const [searchDebounce, setSearchDebounce] = useState<string>("");
  const [searchTerm, setSearchTerm] = useState<string>("");
  const [page, setPage] = useState(1);
  const [expandedLogId, setExpandedLogId] = useState<string | null>(null);

  const limit = 20;
  const offset = (page - 1) * limit;

  useEffect(() => {
    const timer = setTimeout(() => {
      setSearchTerm(searchDebounce);
      setPage(1);
    }, 400);
    return () => clearTimeout(timer);
  }, [searchDebounce]);

  const loadLogs = () => {
    const filters: Record<string, unknown> = {
      limit,
      offset,
    };
    if (selectedAction) filters.action = selectedAction;
    if (selectedSuccess !== "") {
      filters.success = selectedSuccess === "true";
    }
    if (searchTerm) {
      filters.search = searchTerm;
    }
    fetchAuditLogs(filters);
  };

  useEffect(() => {
    loadLogs();

  }, [selectedAction, selectedSuccess, searchTerm, page]);

  const handleRefresh = () => {
    loadLogs();
  };

  const getActionBadge = (action: string) => {
    switch (action) {
      case "gu.feed":
      case "gu.feed_group":
      case "gu.feed_all":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-950/80 border border-emerald-500/30 text-emerald-400 font-mono">
            🌱 NUÔI DƯỠNG
          </span>
        );
      case "refine.gu":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-purple-950/80 border border-purple-500/30 text-purple-400 font-mono">
            🔥 HỢP LUYỆN
          </span>
        );
      case "gu.set_vital":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-amber-950/80 border border-amber-500/30 text-amber-400 font-mono">
            ✨ BẢN MỆNH
          </span>
        );
      case "gu.starve_death":
      case "gu.starve_vital_death":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-rose-950/80 border border-rose-500/30 text-rose-400 font-mono">
            💀 TỊCH DIỆT
          </span>
        );
      case "gu.drop":
      case "gu.drop_batch":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-zinc-800 border border-zinc-600/40 text-zinc-300 font-mono">
            🗑️ VỨT BỎ
          </span>
        );
      case "gu.use_consumable":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-cyan-950/80 border border-cyan-500/30 text-cyan-400 font-mono">
            💊 DÙNG CỔ
          </span>
        );
      case "gu.seal":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-sky-950/80 border border-sky-500/30 text-sky-400 font-mono">
            🧊 PHONG ẤN
          </span>
        );
      case "item.use":
      case "item.use_batch":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-teal-950/80 border border-teal-500/30 text-teal-400 font-mono">
            🧪 DÙNG ĐỒ
          </span>
        );
      case "item.use_relic":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-yellow-950/80 border border-yellow-500/30 text-yellow-400 font-mono">
            🔮 RELIC
          </span>
        );
      case "shop.buy_item":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-amber-950/80 border border-amber-500/30 text-amber-300 font-mono">
            🛒 MUA TIỆM ĐỒ
          </span>
        );
      case "shop.sell_item":
      case "shop.sell_gu":
      case "shop.sell_bulk":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-amber-950/80 border border-amber-600/30 text-amber-300 font-mono">
            🪙 BÁN ĐỒ
          </span>
        );
      case "cultivate.breakthrough":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-violet-950/80 border border-violet-500/30 text-violet-300 font-mono">
            ⚡ TRÙNG KÍCH
          </span>
        );
      case "gift_code.redeem":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-pink-950/80 border border-pink-500/30 text-pink-300 font-mono">
            🎁 MÃ QUÀ TẶNG
          </span>
        );
      case "sect.donate_stones":
      case "sect.mission_complete":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-blue-950/80 border border-blue-500/30 text-blue-400 font-mono">
            📜 TÔNG MÔN
          </span>
        );
      case "aperture.build_node":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-indigo-950/80 border border-indigo-500/30 text-indigo-400 font-mono">
            🏯 TIÊN KHIẾU
          </span>
        );
      case "combat.death_penalty":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-red-950/80 border border-red-500/30 text-red-400 font-mono">
            ⚔️ TỬ ĐẠO
          </span>
        );
      case "recipe.deduce":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-fuchsia-950/80 border border-fuchsia-500/30 text-fuchsia-400 font-mono">
            📖 THÔI DIỄN
          </span>
        );
      case "pet.feed":
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-lime-950/80 border border-lime-500/30 text-lime-400 font-mono">
            🐾 LINH THÚ
          </span>
        );
      default:
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-bold bg-slate-800 border border-slate-700 text-slate-300 font-mono">
            {action.toUpperCase()}
          </span>
        );
    }
  };

  const formatLogTime = (timeStr: string) => {
    if (!timeStr) return "";
    const d = new Date(timeStr);
    return d.toLocaleString("vi-VN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  };

  return (
    <div className="w-full mx-auto bg-slate-950/70 border border-slate-800/80 rounded-2xl p-4 sm:p-6 backdrop-blur-md space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center border-b border-slate-800 pb-4 gap-4">
        <div>
          <h2 className="text-xl sm:text-2xl font-serif font-bold text-slate-100 flex items-center gap-2">
            <History className="w-6 h-6 text-indigo-400" />
            Ký Sự Tu Hành
          </h2>
          <p className="text-xs text-slate-400 mt-1 font-mono">
            Nhật ký ghi chép nhân quả tu luyện, hợp luyện, giao dịch và nuôi dưỡng linh trùng.
          </p>
        </div>
        <button
          onClick={handleRefresh}
          disabled={auditLogsLoading}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-slate-900 border border-slate-700/60 rounded-lg text-xs font-semibold text-slate-200 hover:bg-slate-850 hover:text-white transition-all disabled:opacity-50"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${auditLogsLoading ? "animate-spin" : ""}`} />
          Làm mới
        </button>
      </div>

      {/* Filter & Search Options */}
      <div className="bg-slate-900/40 p-4 rounded-xl border border-slate-800/60 flex flex-col md:flex-row gap-4 items-end">
        {/* Search Bar */}
        <div className="w-full md:w-1/3 space-y-1.5">
          <label className="text-[11px] font-bold text-slate-400 uppercase tracking-wider font-mono flex items-center gap-1">
            <Search className="w-3 h-3 text-indigo-400" />
            Tìm kiếm
          </label>
          <div className="relative">
            <input
              type="text"
              value={searchDebounce}
              onChange={(e) => setSearchDebounce(e.target.value)}
              placeholder="Tìm Cổ trùng, vật phẩm, hành động..."
              className="w-full bg-slate-950 border border-slate-800 rounded-lg pl-8 pr-3 py-2 text-xs text-slate-200 placeholder-slate-600 focus:outline-none focus:border-indigo-500 transition-colors"
            />
            <Search className="w-3.5 h-3.5 text-slate-500 absolute left-2.5 top-2.5" />
          </div>
        </div>

        {/* Action Filter */}
        <div className="w-full md:w-1/3 space-y-1.5">
          <label className="text-[11px] font-bold text-slate-400 uppercase tracking-wider font-mono flex items-center gap-1">
            <Filter className="w-3 h-3 text-indigo-400" />
            Lọc theo hành động
          </label>
          <select
            value={selectedAction}
            onChange={(e) => {
              setSelectedAction(e.target.value);
              setPage(1);
            }}
            className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors"
          >
            <option value="">-- Tất cả hành động --</option>
            <option value="gu.feed">🌱 Cho ăn Cổ trùng</option>
            <option value="gu.feed_group">🌱 Cho ăn hàng loạt</option>
            <option value="gu.feed_all">🌱 Cho ăn toàn bộ</option>
            <option value="refine.gu">🔥 Hợp luyện Cổ</option>
            <option value="gu.set_vital">✨ Khóa Bản Mệnh Cổ</option>
            <option value="gu.starve_death">💀 Cổ trùng chết đói</option>
            <option value="gu.starve_vital_death">💀 Bản mệnh Cổ chết đói</option>
            <option value="gu.drop">🗑️ Vứt bỏ Cổ trùng</option>
            <option value="gu.drop_batch">🗑️ Vứt bỏ Cổ hàng loạt</option>
            <option value="gu.use_consumable">💊 Dùng Cổ tiêu hao</option>
            <option value="gu.seal">🧊 Phong ấn Cổ</option>
            <option value="item.use">🧪 Sử dụng vật phẩm</option>
            <option value="item.use_batch">🧪 Sử dụng hàng loạt</option>
            <option value="item.use_relic">🔮 Dùng Relic Cổ đột phá</option>
            <option value="cultivate.breakthrough">⚡ Trùng kích vách khiếu</option>
            <option value="shop.buy_item">🛒 Mua tiệm đồ</option>
            <option value="shop.sell_item">🪙 Bán vật phẩm</option>
            <option value="shop.sell_gu">🪙 Bán Cổ trùng</option>
            <option value="shop.sell_bulk">🪙 Bán sỉ cho Tiệm Buôn</option>
            <option value="sect.donate_stones">📜 Quyên góp Tông môn</option>
            <option value="sect.mission_complete">📜 Nộp nhiệm vụ Tông môn</option>
            <option value="gift_code.redeem">🎁 Nhập mã quà tặng</option>
            <option value="aperture.build_node">🏯 Xây dựng Tiên Khiếu</option>
            <option value="combat.death_penalty">⚔️ Tử đạo rơi tài sản</option>
            <option value="recipe.deduce">📖 Thôi diễn bí phương</option>
            <option value="pet.feed">🐾 Cho linh thú ăn</option>
            <option value="trade.confirm">🤝 Giao dịch người chơi</option>
            <option value="marketplace.create_listing">🏷️ Treo bán Bảo Hoàng Thiên</option>
            <option value="marketplace.buy_listing">🏷️ Mua đồ Bảo Hoàng Thiên</option>
          </select>
        </div>

        {/* Result Filter */}
        <div className="w-full md:w-1/3 space-y-1.5">
          <label className="text-[11px] font-bold text-slate-400 uppercase tracking-wider font-mono flex items-center gap-1">
            <CheckCircle className="w-3 h-3 text-indigo-400" />
            Lọc theo kết quả
          </label>
          <select
            value={selectedSuccess}
            onChange={(e) => {
              setSelectedSuccess(e.target.value);
              setPage(1);
            }}
            className="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors"
          >
            <option value="">-- Tất cả kết quả --</option>
            <option value="true">Thành công</option>
            <option value="false">Thất bại</option>
          </select>
        </div>
      </div>

      {/* Main List */}
      {auditLogsLoading ? (
        <div className="flex flex-col items-center justify-center py-20 space-y-3">
          <RefreshCw className="w-8 h-8 text-indigo-400 animate-spin" />
          <span className="text-xs text-slate-400 font-mono">Đang truy xuất ký sự tu hành...</span>
        </div>
      ) : auditLogs.length === 0 ? (
        <div className="text-center py-16 bg-slate-900/10 border border-dashed border-slate-800/80 rounded-xl space-y-3">
          <BookOpen className="w-10 h-10 text-slate-600 mx-auto" />
          <p className="text-sm text-slate-400 font-medium">Chưa ghi nhận bất kỳ nhân quả nào trong lịch sử.</p>
          <p className="text-xs text-slate-500 font-mono">Hãy tiến hành cho ăn, luyện cổ hoặc khóa bản mệnh cổ để ghi chép.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {auditLogs.map((log) => {
            const isExpanded = expandedLogId === log.id;
            return (
              <div
                key={log.id}
                className={`bg-slate-900/30 border rounded-xl hover:border-slate-700/50 transition-all ${
                  isExpanded ? "border-slate-700 bg-slate-900/50" : "border-slate-800/80"
                }`}
              >
                {/* Simple row */}
                <div
                  onClick={() => setExpandedLogId(isExpanded ? null : log.id)}
                  className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 cursor-pointer select-none"
                >
                  <div className="flex items-start sm:items-center gap-3">
                    {/* Status Icon */}
                    {log.success ? (
                      <CheckCircle className="w-5 h-5 text-emerald-500 shrink-0 mt-0.5 sm:mt-0" />
                    ) : (
                      <XCircle className="w-5 h-5 text-rose-500 shrink-0 mt-0.5 sm:mt-0" />
                    )}

                    {/* Action Badge & Summary */}
                    <div className="space-y-1">
                      <div className="flex flex-wrap items-center gap-2">
                        {getActionBadge(log.action)}
                        <span className="text-slate-500 text-[11px] font-mono flex items-center gap-1">
                          <Calendar className="w-3 h-3" />
                          {formatLogTime(log.created_at)}
                        </span>
                      </div>
                      <p className="text-xs sm:text-sm font-semibold text-slate-200">
                        {formatRequestSummary(log.request_summary) || "Hành động tu luyện không xác định"}
                      </p>
                    </div>
                  </div>

                  {/* Toggle inspect details */}
                  <div className="flex items-center justify-end text-xs text-slate-400 font-semibold gap-1 font-mono hover:text-indigo-400 transition-colors shrink-0">
                    {log.metadata && Object.keys(log.metadata).length > 0 && (
                      <>
                        {isExpanded ? "Đóng chi tiết" : "Xem chi tiết"}
                        {isExpanded ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
                      </>
                    )}
                  </div>
                </div>

                {/* Expanded Details */}
                {isExpanded && log.metadata && Object.keys(log.metadata).length > 0 && (
                  <div className="px-4 pb-4 pt-1 border-t border-slate-800/80 space-y-3">
                    <div className="text-[11px] font-bold text-slate-400 uppercase tracking-wider font-mono">
                      📋 Thuộc tính chi tiết
                    </div>
                    <div className="bg-slate-950 p-3 rounded-lg border border-slate-850 font-mono text-[11px] text-slate-350 grid grid-cols-1 sm:grid-cols-2 gap-2 text-left">
                      {Object.entries(log.metadata).map(([key, val]) => {
                        let displayVal = typeof val === "object" ? JSON.stringify(val) : String(val);
                        if (key === "item_id" || key === "material_id") {
                          displayVal = itemDictionary[displayVal]?.name || displayVal;
                        } else if (key === "gu_id" || key === "gu_instance_id") {
                          displayVal = guDictionary[displayVal]?.name || displayVal;
                        } else if (key === "monster_id") {
                          displayVal = monsterDictionary[displayVal]?.name || displayVal;
                        }
                        return (
                          <div key={key} className="flex justify-between border-b border-slate-900/60 pb-1">
                            <span className="text-slate-500 font-semibold">{key}:</span>
                            <span className="text-slate-200 text-right max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap" title={displayVal}>
                              {displayVal}
                            </span>
                          </div>
                        );
                      })}
                    </div>
                    {log.error_message && (
                      <div className="bg-rose-950/20 border border-rose-900/40 p-3 rounded-lg text-xs text-rose-350 font-mono">
                        ⚠️ **Chi tiết lỗi:** {log.error_message}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Pagination Controls */}
      <div className="flex items-center justify-between border-t border-slate-800 pt-4 text-xs font-semibold text-slate-400">
        <div>Trang {page}</div>
        <div className="flex gap-2">
          <button
            onClick={() => setPage(Math.max(1, page - 1))}
            disabled={page === 1 || auditLogsLoading}
            className="flex items-center gap-1 px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg hover:bg-slate-850 disabled:opacity-40 transition-colors"
          >
            <ChevronLeft className="w-4 h-4" />
            Trước
          </button>
          <button
            onClick={() => setPage(page + 1)}
            disabled={auditLogs.length < limit || auditLogsLoading}
            className="flex items-center gap-1 px-3 py-1.5 bg-slate-900 border border-slate-800 rounded-lg hover:bg-slate-850 disabled:opacity-40 transition-colors"
          >
            Sau
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
};
