import React, { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { BookOpen, RefreshCw } from "lucide-react";
import { useAdminUrlState } from "../useAdminUrlState";

const BACKEND_URL =
  process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const AUDIT_DEFAULTS = { page: "1", page_size: "50", search: "", action: "", player_id: "", success: "", sort: "created_at", direction: "desc" };
const AUDIT_KEYS = Object.keys(AUDIT_DEFAULTS);
function formatAdminLogTime(timeStr: string): string {
  if (!timeStr) return "—";
  const d = new Date(timeStr);
  if (isNaN(d.getTime())) return timeStr;
  const pad = (n: number) => (n < 10 ? "0" + n : n);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}

const AUDIT_ACTIONS = [
  { action: "gu.feed", label: "Cho ăn Cổ trùng", icon: "🌱" },
  { action: "gu.feed_group", label: "Cho ăn hàng loạt", icon: "🌱" },
  { action: "gu.equip", label: "Trang bị Cổ trùng", icon: "⚔️" },
  { action: "gu.unequip", label: "Tháo Cổ trùng", icon: "🛡️" },
  { action: "gu.set_vital", label: "Khóa Bản Mệnh Cổ", icon: "✨" },
  { action: "gu.drop", label: "Vứt Cổ trùng", icon: "🗑️" },
  { action: "gu.drop_batch", label: "Vứt Cổ trùng hàng loạt", icon: "🗑️" },
  { action: "gu.use_consumable", label: "Sử dụng Cổ trùng", icon: "🧪" },
  { action: "gu.starve_death", label: "Cổ trùng chết đói", icon: "💀" },
  {
    action: "gu.starve_vital_death",
    label: "Bản mệnh Cổ chết đói",
    icon: "💀",
  },
  { action: "refine.gu", label: "Hợp luyện Cổ", icon: "🔥" },
  { action: "refine.deduce_recipe", label: "Suy luận công thức", icon: "📜" },
  { action: "combat.started", label: "Bắt đầu chiến đấu", icon: "⚔️" },
  { action: "combat.ended", label: "Kết thúc chiến đấu", icon: "⚔️" },
  { action: "combat.finish", label: "Hoàn thành chiến đấu", icon: "🏆" },
  { action: "combat.flee", label: "Bỏ chạy chiến đấu", icon: "🏃" },
  { action: "combat.retreat", label: "Rút lui chiến đấu", icon: "🏃" },
  { action: "cultivate.breakthrough", label: "Đột phá tu vi", icon: "🌟" },
  { action: "cultivate.break_wall", label: "Phá vách", icon: "🧱" },
  { action: "cultivate.wall_cracked", label: "Vách nứt", icon: "🧱" },
  { action: "soul.fortify", label: "Luyện hồn", icon: "👻" },
  { action: "soul.refine", label: "Tinh luyện hồn", icon: "👻" },
  { action: "gacha.open", label: "Mở đá", icon: "💎" },
  { action: "gacha.open_bulk", label: "Mở đá hàng loạt", icon: "💎" },
  { action: "shop.buy", label: "Mua vật phẩm", icon: "🛒" },
  { action: "shop.buy_package", label: "Mua gói vật phẩm", icon: "🛒" },
  { action: "shop.sell", label: "Bán vật phẩm", icon: "💰" },
  { action: "marketplace.create_listing", label: "Đăng bán chợ", icon: "🏪" },
  { action: "marketplace.buy", label: "Mua từ chợ", icon: "🏪" },
  { action: "marketplace.sold", label: "Chợ bán thành công", icon: "🏪" },
  { action: "marketplace.cancel_listing", label: "Hủy đăng bán", icon: "🏪" },
  { action: "marketplace.bid", label: "Đặt giá đấu giá", icon: "🏷️" },
  {
    action: "marketplace.settle_auction_win",
    label: "Kết toán đấu giá thắng",
    icon: "🏷️",
  },
  {
    action: "marketplace.settle_auction_expired",
    label: "Kết toán đấu giá hết hạn",
    icon: "🏷️",
  },
  { action: "trade.initiate", label: "Mở giao dịch", icon: "🤝" },
  { action: "trade.confirm", label: "Xác nhận giao dịch", icon: "🤝" },
  { action: "trade.accept", label: "Chấp nhận giao dịch", icon: "🤝" },
  { action: "trade.complete", label: "Hoàn thành giao dịch", icon: "🤝" },
  { action: "trade.cancel", label: "Hủy giao dịch", icon: "🤝" },
  { action: "trade.decline", label: "Từ chối giao dịch", icon: "🤝" },
  { action: "trade.lock", label: "Khóa giao dịch", icon: "🔒" },
  { action: "trade.updated", label: "Cập nhật giao dịch", icon: "🤝" },
  { action: "quest.complete", label: "Hoàn thành nhiệm vụ", icon: "📋" },
  { action: "achievement.claim", label: "Nhận thành tựu", icon: "🏅" },
  { action: "sect.join", label: "Vào tông môn", icon: "🏯" },
  { action: "sect.leave", label: "Rời tông môn", icon: "🏯" },
  { action: "sect.donate_stones", label: "Quyên góp đá tông môn", icon: "🏯" },
  {
    action: "sect.donate_materials",
    label: "Quyên góp nguyên liệu tông môn",
    icon: "🏯",
  },
  {
    action: "sect.donate_gu",
    label: "Quyên góp Cổ trùng tông môn",
    icon: "🏯",
  },
  { action: "sect.buy_vault", label: "Mua kho tông môn", icon: "🏯" },
  { action: "player.register", label: "Đăng ký tài khoản", icon: "👤" },
  { action: "player.login", label: "Đăng nhập", icon: "🔑" },
  { action: "player.logout", label: "Đăng xuất", icon: "🔑" },
  { action: "player.updated", label: "Cập nhật nhân vật", icon: "👤" },
  { action: "player.heartbeat", label: "Heartbeat", icon: "💓" },
  { action: "gift_code.redeem", label: "Đổi mã quà", icon: "🎁" },
  { action: "topup.create_order", label: "Tạo đơn nạp tiền", icon: "💳" },
  { action: "topup.webhook_applied", label: "Nạp tiền thành công", icon: "💳" },
  { action: "topup.webhook_failed", label: "Nạp tiền thất bại", icon: "💳" },
  { action: "deposit.simulate", label: "Mô phỏng nạp tiền", icon: "💳" },
  { action: "admin.create_gu", label: "Admin tạo Cổ trùng", icon: "🔧" },
  { action: "admin.update_gu", label: "Admin sửa Cổ trùng", icon: "🔧" },
  { action: "admin.delete_gu", label: "Admin xóa Cổ trùng", icon: "🔧" },
  { action: "admin.add_gus", label: "Admin thêm Cổ trùng", icon: "🔧" },
  { action: "admin.remove_gu", label: "Admin xóa Cổ trùng NL", icon: "🔧" },
  { action: "admin.create_item", label: "Admin tạo vật phẩm", icon: "🔧" },
  { action: "admin.update_item", label: "Admin sửa vật phẩm", icon: "🔧" },
  { action: "admin.delete_item", label: "Admin xóa vật phẩm", icon: "🔧" },
  { action: "admin.add_items", label: "Admin thêm vật phẩm", icon: "🔧" },
  { action: "admin.remove_items", label: "Admin xóa vật phẩm NL", icon: "🔧" },
  { action: "admin.create_recipe", label: "Admin tạo công thức", icon: "🔧" },
  { action: "admin.update_recipe", label: "Admin sửa công thức", icon: "🔧" },
  { action: "admin.delete_recipe", label: "Admin xóa công thức", icon: "🔧" },
  { action: "admin.change_stones", label: "Admin đổi đá", icon: "🔧" },
  { action: "admin.update_player", label: "Admin sửa nhân vật", icon: "🔧" },
  { action: "admin.create_shop_item", label: "Admin tạo cửa hàng", icon: "🔧" },
  { action: "admin.update_shop_item", label: "Admin sửa cửa hàng", icon: "🔧" },
  { action: "admin.delete_shop_item", label: "Admin xóa cửa hàng", icon: "🔧" },
  {
    action: "admin.create_arena_shop_item",
    label: "Admin tạo cửa hàng ĐT",
    icon: "🔧",
  },
  {
    action: "admin.update_arena_shop_item",
    label: "Admin sửa cửa hàng ĐT",
    icon: "🔧",
  },
  {
    action: "admin.delete_arena_shop_item",
    label: "Admin xóa cửa hàng ĐT",
    icon: "🔧",
  },
  { action: "admin.create_gift_code", label: "Admin tạo mã quà", icon: "🔧" },
  {
    action: "admin.create_sect_mission",
    label: "Admin tạo nhiệm vụ TM",
    icon: "🔧",
  },
  {
    action: "admin.delete_sect_mission",
    label: "Admin xóa nhiệm vụ TM",
    icon: "🔧",
  },
  { action: "admin.update_sect", label: "Admin sửa tông môn", icon: "🔧" },
  { action: "admin.delete_sect", label: "Admin xóa tông môn", icon: "🔧" },
  {
    action: "admin.set_sect_leader",
    label: "Admin đặt trưởng lão TM",
    icon: "🔧",
  },
  {
    action: "admin.kick_sect_member",
    label: "Admin kick thành viên TM",
    icon: "🔧",
  },
  {
    action: "admin.stock_sect_vault",
    label: "Admin nạp kho tông môn",
    icon: "🔧",
  },
  {
    action: "admin.remove_sect_vault",
    label: "Admin rút kho tông môn",
    icon: "🔧",
  },
  { action: "admin.add_to_sect", label: "Admin thêm vào tông môn", icon: "🔧" },
  { action: "admin.import_data", label: "Admin nhập dữ liệu", icon: "📥" },
  { action: "admin.export_data", label: "Admin xuất dữ liệu", icon: "📤" },
  { action: "mana.snapshot", label: "Chụp snapshot mana", icon: "📸" },
];

export function AuditLogsAdminTab() {
  const [auditLogs, setAuditLogs] = useState<ApiResponseData[]>([]);
  const [auditLogTotal, setAuditLogTotal] = useState(0);
  const [auditLogLoading, setAuditLogLoading] = useState(false);
  const [query, updateQuery] = useAdminUrlState(AUDIT_DEFAULTS, AUDIT_KEYS);
  const auditLogPage = Math.max(1, Number(query.page) || 1);
  const auditLogFilterAction = query.action;
  const auditLogFilterPlayer = query.player_id;
  const auditLogFilterSuccess = query.success;
  const auditLogFilterSearch = query.search;
  const [auditLogSearchDebounce, setAuditLogSearchDebounce] = useState(query.search);
  const setAuditLogPage = (value: number | ((page: number) => number)) => updateQuery({ page: String(typeof value === "function" ? value(auditLogPage) : value) });
  const [auditLogExpandId, setAuditLogExpandId] = useState<string | null>(null);
  const [auditLogActionOpen, setAuditLogActionOpen] = useState(false);
  const [auditLogActionSearch, setAuditLogActionSearch] = useState("");

  const fetchAuditLogs = useCallback(async () => {
    setAuditLogLoading(true);
    try {
      const params = new URLSearchParams({
        page_size: query.page_size,
        page: String(auditLogPage), sort: query.sort, direction: query.direction,
      });
      if (auditLogFilterAction) params.set("action", auditLogFilterAction);
      if (auditLogFilterPlayer) params.set("player_id", auditLogFilterPlayer);
      if (auditLogFilterSuccess) params.set("success", auditLogFilterSuccess);
      if (auditLogFilterSearch) params.set("search", auditLogFilterSearch);
      const response = await authFetch(
        `${BACKEND_URL}/api/admin/audit-logs?${params}`,
      );
      if (response.ok) {
        const data = await response.json();
        setAuditLogs(data.data || []);
        setAuditLogTotal(data.total || 0);
      }
    } catch {
      // Preserve the existing silent failure behavior for audit-log loading.
    } finally {
      setAuditLogLoading(false);
    }
  }, [
    auditLogPage,
    auditLogFilterAction,
    auditLogFilterPlayer,
    auditLogFilterSuccess,
    auditLogFilterSearch, query.direction, query.page_size, query.sort,
  ]);

  useEffect(() => {
    void fetchAuditLogs();
  }, [fetchAuditLogs]);

  useEffect(() => {
    const timer = window.setTimeout(
      () => updateQuery({ search: auditLogSearchDebounce, page: "1" }, true),
      400,
    );
    return () => window.clearTimeout(timer);
  }, [auditLogSearchDebounce, updateQuery]);

  useEffect(() => setAuditLogSearchDebounce(query.search), [query.search]);

  useEffect(() => {
    if (!auditLogActionOpen) return;
    const handler = (event: MouseEvent) => {
      const target = event.target as HTMLElement;
      if (!target.closest("[data-audit-action-combobox]")) {
        setAuditLogActionOpen(false);
        setAuditLogActionSearch("");
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [auditLogActionOpen]);

  return (
    <div className="space-y-4">
      <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-4">
        <h3 className="text-sm font-serif font-bold text-zinc-200 flex items-center gap-2 mb-3">
          <BookOpen className="w-4 h-4 text-amber-400" />
          Nhật Ký Hệ Thống
        </h3>
        {/* Filters */}
        <div className="flex flex-wrap gap-3 mb-4">
          <div className="flex-1 min-w-[200px]">
            <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
              Tìm kiếm
            </label>
            <input
              value={auditLogSearchDebounce}
              onChange={(e) => setAuditLogSearchDebounce(e.target.value)}
              placeholder="Tìm trong action, mô tả, entity..."
              className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-amber-500/50"
            />
          </div>
          <div
            className="flex-1 min-w-[180px] relative"
            data-audit-action-combobox
          >
            <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
              Hành động
            </label>
            <div className="relative">
              <button
                onClick={() => setAuditLogActionOpen(!auditLogActionOpen)}
                className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 text-left flex items-center justify-between focus:outline-none focus:border-amber-500/50 cursor-pointer"
              >
                <span className={auditLogFilterAction ? "" : "text-zinc-600"}>
                  {auditLogFilterAction
                    ? AUDIT_ACTIONS.find(
                        (a) => a.action === auditLogFilterAction,
                      )?.label || auditLogFilterAction
                    : "-- Tất cả --"}
                </span>
                <svg
                  className={`w-3 h-3 text-zinc-500 transition-transform ${auditLogActionOpen ? "rotate-180" : ""}`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 9l-7 7-7-7"
                  />
                </svg>
              </button>
              {auditLogActionOpen && (
                <div
                  data-audit-action-combobox
                  className="absolute z-50 mt-1 w-full bg-zinc-900 border border-zinc-700 rounded-lg shadow-xl max-h-[280px] overflow-hidden flex flex-col"
                >
                  <div className="p-1.5 border-b border-zinc-800">
                    <input
                      autoFocus
                      value={auditLogActionSearch}
                      onChange={(e) => setAuditLogActionSearch(e.target.value)}
                      placeholder="Tìm action..."
                      className="w-full px-2 py-1 bg-zinc-800 border border-zinc-700 rounded text-xs text-zinc-200 placeholder-zinc-600 focus:outline-none"
                    />
                  </div>
                  <div className="overflow-y-auto max-h-[230px]">
                    <button
                      onClick={() => {
                        updateQuery({ action: "", page: "1" });
                        setAuditLogActionOpen(false);
                        setAuditLogActionSearch("");
                      }}
                      className="w-full text-left px-3 py-1.5 text-xs hover:bg-amber-600/20 text-zinc-400 hover:text-amber-400 cursor-pointer"
                    >
                      -- Tất cả --
                    </button>
                    {AUDIT_ACTIONS.filter(
                      (a) =>
                        !auditLogActionSearch ||
                        a.action.includes(auditLogActionSearch.toLowerCase()) ||
                        a.label
                          .toLowerCase()
                          .includes(auditLogActionSearch.toLowerCase()),
                    ).map((a) => (
                      <button
                        key={a.action}
                        onClick={() => {
                          updateQuery({ action: a.action, page: "1" });
                          setAuditLogActionOpen(false);
                          setAuditLogActionSearch("");
                        }}
                        className={`w-full text-left px-3 py-1.5 text-xs hover:bg-amber-600/20 cursor-pointer flex items-center gap-2 ${auditLogFilterAction === a.action ? "text-amber-400 bg-amber-600/10" : "text-zinc-300"}`}
                      >
                        <span>{a.icon}</span>
                        <span className="truncate">{a.label}</span>
                        <span className="ml-auto text-[10px] text-zinc-600 font-mono shrink-0">
                          {a.action}
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
          <div className="flex-1 min-w-[120px]">
            <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
              Player ID
            </label>
            <input
              value={auditLogFilterPlayer}
              onChange={(e) => updateQuery({ player_id: e.target.value, page: "1" })}
              placeholder="UUID..."
              className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-amber-500/50"
            />
          </div>
          <div className="w-[100px]">
            <label className="text-[10px] font-bold text-zinc-500 uppercase tracking-wider block mb-1">
              Kết quả
            </label>
            <select
              value={auditLogFilterSuccess}
              onChange={(e) => {
                updateQuery({ success: e.target.value, page: "1" });
              }}
              className="w-full px-2.5 py-1.5 bg-zinc-800/80 border border-zinc-700/60 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50"
            >
              <option value="">Tất cả</option>
              <option value="true">Thành công</option>
              <option value="false">Thất bại</option>
            </select>
          </div>
          <div className="flex items-end">
            <button
              onClick={fetchAuditLogs}
              disabled={auditLogLoading}
              className="px-3 py-1.5 bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white rounded-lg text-xs font-semibold cursor-pointer"
            >
              {auditLogLoading ? "..." : "Tìm"}
            </button>
          </div>
        </div>
        {/* Log list */}
        {auditLogLoading ? (
          <div className="flex justify-center py-10">
            <RefreshCw className="w-6 h-6 text-amber-400 animate-spin" />
          </div>
        ) : auditLogs.length === 0 ? (
          <div className="text-center py-10 bg-zinc-900/10 border border-dashed border-zinc-800 rounded-xl">
            <BookOpen className="w-8 h-8 text-zinc-700 mx-auto mb-2" />
            <p className="text-xs text-zinc-500">Không có nhật ký nào.</p>
          </div>
        ) : (
          <>
            <div className="space-y-1 max-h-[500px] overflow-y-auto">
              {auditLogs.map((log: ApiResponseData) => (
                <div key={log.id}>
                  <div
                    onClick={() =>
                      setAuditLogExpandId(
                        auditLogExpandId === log.id ? null : log.id,
                      )
                    }
                    className="flex items-center gap-3 px-3 py-2 bg-zinc-800/40 hover:bg-zinc-800/60 rounded-lg cursor-pointer transition-colors"
                  >
                    <div
                      className={`w-2 h-2 rounded-full shrink-0 ${log.success ? "bg-emerald-500" : "bg-red-500"}`}
                    />
                    <span
                      className="text-xs font-mono text-zinc-400 w-[145px] shrink-0 overflow-hidden text-ellipsis whitespace-nowrap"
                      title={formatAdminLogTime(log.created_at)}
                    >
                      {formatAdminLogTime(log.created_at)}
                    </span>
                    <span
                      className="text-xs font-semibold text-amber-400 min-w-[150px] max-w-[200px] shrink-0 truncate"
                      title={log.action}
                    >
                      {log.action}
                    </span>
                    <span
                      className="text-xs text-zinc-400 truncate flex-1"
                      title={log.request_summary}
                    >
                      {log.request_summary}
                    </span>
                    <span
                      className="text-[10px] text-zinc-600 w-[45px] text-right font-mono shrink-0"
                      title={log.player_id}
                    >
                      {log.player_id ? log.player_id.substring(0, 8) : "—"}
                    </span>
                  </div>
                  {auditLogExpandId === log.id && (
                    <div className="mx-3 mb-1 px-4 py-2 bg-zinc-900/60 border border-zinc-800/60 rounded-lg text-[11px] text-zinc-400 font-mono space-y-1">
                      {log.player_id && <div>player_id: {log.player_id}</div>}
                      {log.entity_type && (
                        <div>entity_type: {log.entity_type}</div>
                      )}
                      {log.entity_id && <div>entity_id: {log.entity_id}</div>}
                      {log.error_message && (
                        <div className="text-red-400">
                          error: {log.error_message}
                        </div>
                      )}
                      {log.ip_address && <div>ip: {log.ip_address}</div>}
                      {log.metadata && Object.keys(log.metadata).length > 0 && (
                        <div>
                          metadata: {JSON.stringify(log.metadata, null, 1)}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
            {/* Pagination */}
            <div className="flex items-center justify-between mt-4 pt-3 border-t border-zinc-800/60">
              <span className="text-[11px] text-zinc-500 font-mono">
                {auditLogTotal} bản ghi
              </span>
              <div className="flex gap-2">
                <button
                  disabled={auditLogPage <= 1}
                  onClick={() => setAuditLogPage((p) => Math.max(1, p - 1))}
                  className="px-3 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-30 text-zinc-300 rounded-lg text-[11px] font-semibold cursor-pointer"
                >
                  Trước
                </button>
                <button
                  disabled={auditLogPage * Number(query.page_size) >= auditLogTotal}
                  onClick={() => setAuditLogPage((p) => p + 1)}
                  className="px-3 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-30 text-zinc-300 rounded-lg text-[11px] font-semibold cursor-pointer"
                >
                  Sau
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
