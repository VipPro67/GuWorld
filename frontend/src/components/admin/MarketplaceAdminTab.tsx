"use client";

import React, { useEffect, useState, useCallback, useTransition } from "react";
import {
  Scale,
  Search,
  RefreshCw,
  Ban,
  Clock,
  ArrowRightLeft,
  Gavel,
  ShieldAlert,
  Coins,
  Bug,
  Package,
} from "lucide-react";
import { authFetch } from "@/lib/auth";
import { prepareAdminAction, canPerformAdminAction } from "./actions";
import { useAdminAccess } from "./AdminAccessContext";
import { formatNumber } from "@/lib/utils";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

type AuctionItem = {
  id: string;
  seller_id: string;
  seller_name: string;
  item_type: string;
  item_data: Record<string, unknown>;
  quantity: number;
  currency: string;
  start_price: number;
  current_price: number;
  highest_bidder_id: string | null;
  highest_bidder_name: string;
  end_time: string;
  status: "active" | "sold" | "cancelled" | "expired";
  listing_type: string;
  created_at: string;
  duration_hours: number;
};

type TradeItem = {
  id: string;
  sender_id: string;
  sender_name: string;
  receiver_id: string;
  receiver_name: string;
  sender_offer: {
    primeval_stones?: number;
    immortal_stones?: number;
    gus?: Array<Record<string, unknown>>;
    materials?: Record<string, number>;
  };
  receiver_offer: {
    primeval_stones?: number;
    immortal_stones?: number;
    gus?: Array<Record<string, unknown>>;
    materials?: Record<string, number>;
  };
  status: string;
  created_at: string;
  updated_at: string;
};

export const MarketplaceAdminTab: React.FC<{
  addToast: (msg: string, type: "success" | "error" | "info") => void;
}> = ({ addToast }) => {
  const adminAccess = useAdminAccess();
  const canCancel = canPerformAdminAction(adminAccess.effective_permissions, "cancelAuction");

  const [tab, setTab] = useState<"auctions" | "trades">("auctions");

  // Auctions state
  const [auctions, setAuctions] = useState<AuctionItem[]>([]);
  const [auctionsTotal, setAuctionsTotal] = useState(0);
  const [auctionsPage, setAuctionsPage] = useState(1);
  const [auctionSearch, setAuctionSearch] = useState("");
  const [debouncedAuctionSearch, setDebouncedAuctionSearch] = useState("");
  const [auctionStatus, setAuctionStatus] = useState("ALL");
  const [auctionItemType, setAuctionItemType] = useState("ALL");
  const [auctionsLoading, setAuctionsLoading] = useState(false);

  // Trades state
  const [trades, setTrades] = useState<TradeItem[]>([]);
  const [tradesTotal, setTradesTotal] = useState(0);
  const [tradesPage, setTradesPage] = useState(1);
  const [tradeSearch, setTradeSearch] = useState("");
  const [debouncedTradeSearch, setDebouncedTradeSearch] = useState("");
  const [tradeStatus, setTradeStatus] = useState("ALL");
  const [tradesLoading, setTradesLoading] = useState(false);

  const [, startTransition] = useTransition();

  // Debounce auction search
  useEffect(() => {
    const handler = setTimeout(() => {
      startTransition(() => {
        setDebouncedAuctionSearch(auctionSearch);
        setAuctionsPage(1);
      });
    }, 400);
    return () => clearTimeout(handler);
  }, [auctionSearch]);

  // Debounce trade search
  useEffect(() => {
    const handler = setTimeout(() => {
      startTransition(() => {
        setDebouncedTradeSearch(tradeSearch);
        setTradesPage(1);
      });
    }, 400);
    return () => clearTimeout(handler);
  }, [tradeSearch]);

  const fetchAuctions = useCallback(
    async (p: number, q: string, st: string, it: string) => {
      setAuctionsLoading(true);
      try {
        const params = new URLSearchParams({
          page: String(p),
          page_size: "20",
        });
        if (q) params.set("search", q);
        if (st && st !== "ALL") params.set("status", st);
        if (it && it !== "ALL") params.set("item_type", it);

        const res = await authFetch(`${BACKEND_URL}/api/admin/marketplace/auctions?${params.toString()}`);
        if (!res.ok) throw new Error("Không thể tải danh sách đấu giá");
        const json = await res.json();
        setAuctions(json.data || []);
        setAuctionsTotal(json.total || 0);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        addToast(msg, "error");
      } finally {
        setAuctionsLoading(false);
      }
    },
    [addToast]
  );

  const fetchTrades = useCallback(
    async (p: number, q: string, st: string) => {
      setTradesLoading(true);
      try {
        const params = new URLSearchParams({
          page: String(p),
          page_size: "20",
        });
        if (q) params.set("search", q);
        if (st && st !== "ALL") params.set("status", st);

        const res = await authFetch(`${BACKEND_URL}/api/admin/marketplace/trades?${params.toString()}`);
        if (!res.ok) throw new Error("Không thể tải lịch sử giao dịch");
        const json = await res.json();
        setTrades(json.data || []);
        setTradesTotal(json.total || 0);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : String(err);
        addToast(msg, "error");
      } finally {
        setTradesLoading(false);
      }
    },
    [addToast]
  );

  useEffect(() => {
    if (tab === "auctions") {
      fetchAuctions(auctionsPage, debouncedAuctionSearch, auctionStatus, auctionItemType);
    } else {
      fetchTrades(tradesPage, debouncedTradeSearch, tradeStatus);
    }
  }, [tab, auctionsPage, debouncedAuctionSearch, auctionStatus, auctionItemType, tradesPage, debouncedTradeSearch, tradeStatus, fetchAuctions, fetchTrades]);

  const handleCancelAuction = async (item: AuctionItem) => {
    const prepared = prepareAdminAction("cancelAuction", "hủy phiên đấu giá", item.id);
    if (!prepared) return;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/marketplace/auctions/${item.id}`, {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
          "X-Admin-Reason": prepared.reason || "",
          "Idempotency-Key": prepared.idempotencyKey,
        },
        body: JSON.stringify(prepared),
      });
      if (!res.ok) {
        const errJson = await res.json().catch(() => ({}));
        throw new Error(errJson.error || "Không thể hủy phiên đấu giá");
      }
      addToast("Đã hủy phiên đấu giá thành công", "success");
      fetchAuctions(auctionsPage, debouncedAuctionSearch, auctionStatus, auctionItemType);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err);
      addToast(msg, "error");
    }
  };

  const renderOfferSummary = (offer: TradeItem["sender_offer"]) => {
    if (!offer) return <span className="text-zinc-600">—</span>;
    const parts = [];
    if (offer.primeval_stones && offer.primeval_stones > 0) {
      parts.push(`${formatNumber(offer.primeval_stones, 0)} NT`);
    }
    if (offer.immortal_stones && offer.immortal_stones > 0) {
      parts.push(`${formatNumber(offer.immortal_stones, 1)} TN`);
    }
    if (offer.gus && offer.gus.length > 0) {
      parts.push(`${offer.gus.length} Cổ`);
    }
    if (offer.materials && Object.keys(offer.materials).length > 0) {
      const matCount = Object.values(offer.materials).reduce((a, b) => a + b, 0);
      parts.push(`${matCount} Nguyên liệu`);
    }
    return parts.length > 0 ? (
      <span className="font-mono text-zinc-300 text-[11px]">{parts.join(", ")}</span>
    ) : (
      <span className="text-zinc-600 text-[11px]">Trống</span>
    );
  };

  return (
    <div className="space-y-4">
      {/* Subtab Navigation */}
      <div className="border-b border-zinc-800/80 flex gap-0 mb-4 overflow-x-auto scrollbar-hide whitespace-nowrap [&>button]:shrink-0">
        <button
          onClick={() => setTab("auctions")}
          className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex items-center gap-1.5 ${
            tab === "auctions" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"
          }`}
        >
          <Gavel className="w-3.5 h-3.5" />
          Sàn Đấu Giá (Auctions)
        </button>
        <button
          onClick={() => setTab("trades")}
          className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer flex items-center gap-1.5 ${
            tab === "trades" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"
          }`}
        >
          <ArrowRightLeft className="w-3.5 h-3.5" />
          Nhật Ký Giao Dịch Tự Do (Trades)
        </button>
      </div>

      {tab === "auctions" && (
        <div className="space-y-4">
          {/* Controls */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
            <div className="relative md:col-span-2">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
              <input
                value={auctionSearch}
                onChange={(e) => setAuctionSearch(e.target.value)}
                placeholder="Tìm theo người bán, người đấu, tên vật phẩm..."
                className="w-full bg-zinc-900/80 border border-zinc-800 rounded-lg pl-9 pr-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50"
              />
            </div>
            <div>
              <select
                value={auctionStatus}
                onChange={(e) => {
                  setAuctionStatus(e.target.value);
                  setAuctionsPage(1);
                }}
                className="w-full bg-zinc-900/80 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-300 focus:outline-none focus:border-amber-500/50"
              >
                <option value="ALL">Tất cả trạng thái</option>
                <option value="active">Đang đấu giá (Active)</option>
                <option value="sold">Đã bán thành công (Sold)</option>
                <option value="cancelled">Đã hủy (Cancelled)</option>
                <option value="expired">Hết hạn (Expired)</option>
              </select>
            </div>
            <div>
              <select
                value={auctionItemType}
                onChange={(e) => {
                  setAuctionItemType(e.target.value);
                  setAuctionsPage(1);
                }}
                className="w-full bg-zinc-900/80 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-300 focus:outline-none focus:border-amber-500/50"
              >
                <option value="ALL">Tất cả loại hàng</option>
                <option value="GU">Cổ Trùng (Gu)</option>
                <option value="MATERIAL">Nguyên liệu (Material)</option>
                <option value="RECIPE">Cổ phương (Recipe)</option>
              </select>
            </div>
          </div>

          {/* Table */}
          <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-xl overflow-hidden">
            {auctionsLoading && auctions.length === 0 ? (
              <div className="py-16 text-center text-xs text-zinc-500 flex flex-col items-center gap-2">
                <RefreshCw className="w-5 h-5 animate-spin text-amber-400" />
                Đang tải danh sách đấu giá...
              </div>
            ) : auctions.length === 0 ? (
              <div className="py-16 text-center text-xs text-zinc-500">
                Không có phiên đấu giá nào phù hợp.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="bg-zinc-900/90 border-b border-zinc-800 text-zinc-400 font-mono text-[11px] uppercase">
                      <th className="px-4 py-3">Vật phẩm</th>
                      <th className="px-3 py-3">Loại</th>
                      <th className="px-3 py-3">Người bán</th>
                      <th className="px-3 py-3 text-right">Khởi điểm</th>
                      <th className="px-3 py-3 text-right">Hiện tại</th>
                      <th className="px-3 py-3">Đang dẫn đầu</th>
                      <th className="px-3 py-3">Hết hạn</th>
                      <th className="px-3 py-3">Trạng thái</th>
                      <th className="px-4 py-3 text-right">Thao tác</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/50">
                    {auctions.map((a) => {
                      const itemName = String(
                        a.item_data?.name || a.item_data?.target_gu_name || a.item_data?.id || a.item_type || "Vật phẩm"
                      );
                      const statusBadge =
                        a.status === "active"
                          ? "text-emerald-400 bg-emerald-500/10 border-emerald-500/30"
                          : a.status === "sold"
                          ? "text-blue-400 bg-blue-500/10 border-blue-500/30"
                          : a.status === "cancelled"
                          ? "text-rose-400 bg-rose-500/10 border-rose-500/30"
                          : "text-zinc-500 bg-zinc-800 border-zinc-700";

                      return (
                        <tr key={a.id} className="hover:bg-zinc-800/30 transition-colors">
                          <td className="px-4 py-3">
                            <div className="font-semibold text-zinc-200 flex items-center gap-1.5">
                              {a.item_type === "GU" ? (
                                <Bug className="w-3.5 h-3.5 text-amber-400" />
                              ) : (
                                <Package className="w-3.5 h-3.5 text-emerald-400" />
                              )}
                              <span>{itemName}</span>
                              <span className="text-[10px] text-zinc-500 font-mono">x{a.quantity}</span>
                            </div>
                            <span className="text-[10px] text-zinc-600 font-mono">{a.id}</span>
                          </td>
                          <td className="px-3 py-3 font-mono text-[11px] text-zinc-400">
                            {a.listing_type}
                          </td>
                          <td className="px-3 py-3 text-zinc-300 font-mono">
                            {a.seller_name}
                          </td>
                          <td className="px-3 py-3 text-right font-mono text-zinc-400">
                            {formatNumber(Number(a.start_price), 1)}
                          </td>
                          <td className="px-3 py-3 text-right font-mono text-yellow-400 font-semibold">
                            {formatNumber(Number(a.current_price), 1)}
                          </td>
                          <td className="px-3 py-3 text-zinc-300 font-mono">
                            {a.highest_bidder_name || <span className="text-zinc-600">Chưa có</span>}
                          </td>
                          <td className="px-3 py-3 text-zinc-500 text-[11px]">
                            {new Date(a.end_time).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" })}
                          </td>
                          <td className="px-3 py-3">
                            <span className={`px-2 py-0.5 rounded-full text-[10px] font-semibold border ${statusBadge}`}>
                              {a.status}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-right">
                            {canCancel && a.status === "active" && (
                              <button
                                onClick={() => handleCancelAuction(a)}
                                className="px-2.5 py-1 bg-red-950/60 hover:bg-red-900 border border-red-800/50 text-red-300 rounded text-xs font-mono transition-colors cursor-pointer flex items-center gap-1 ml-auto"
                                title="Hủy phiên đấu giá và hoàn trả tài sản"
                              >
                                <Ban className="w-3 h-3" />
                                Hủy phiên
                              </button>
                            )}
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
                Tổng số: <span className="font-mono text-zinc-300">{auctionsTotal}</span> phiên đấu giá
              </div>
              <div className="flex items-center gap-2">
                <span>
                  Trang {auctionsPage} / {Math.max(1, Math.ceil(auctionsTotal / 20))}
                </span>
                <button
                  onClick={() => setAuctionsPage((p) => Math.max(1, p - 1))}
                  disabled={auctionsPage <= 1}
                  className="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 text-zinc-300 rounded cursor-pointer"
                >
                  Trước
                </button>
                <button
                  onClick={() => setAuctionsPage((p) => Math.min(Math.ceil(auctionsTotal / 20), p + 1))}
                  disabled={auctionsPage >= Math.ceil(auctionsTotal / 20)}
                  className="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 text-zinc-300 rounded cursor-pointer"
                >
                  Sau
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {tab === "trades" && (
        <div className="space-y-4">
          {/* Controls */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <div className="relative md:col-span-2">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
              <input
                value={tradeSearch}
                onChange={(e) => setTradeSearch(e.target.value)}
                placeholder="Tìm theo người gửi, người nhận..."
                className="w-full bg-zinc-900/80 border border-zinc-800 rounded-lg pl-9 pr-3 py-2 text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50"
              />
            </div>
            <div>
              <select
                value={tradeStatus}
                onChange={(e) => {
                  setTradeStatus(e.target.value);
                  setTradesPage(1);
                }}
                className="w-full bg-zinc-900/80 border border-zinc-800 rounded-lg px-3 py-2 text-xs text-zinc-300 focus:outline-none focus:border-amber-500/50"
              >
                <option value="ALL">Tất cả trạng thái</option>
                <option value="PENDING">Đang chờ (Pending)</option>
                <option value="COMPLETED">Đã hoàn tất (Completed)</option>
                <option value="CANCELLED">Đã hủy (Cancelled)</option>
              </select>
            </div>
          </div>

          {/* Table */}
          <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-xl overflow-hidden">
            {tradesLoading && trades.length === 0 ? (
              <div className="py-16 text-center text-xs text-zinc-500 flex flex-col items-center gap-2">
                <RefreshCw className="w-5 h-5 animate-spin text-amber-400" />
                Đang tải lịch sử giao dịch...
              </div>
            ) : trades.length === 0 ? (
              <div className="py-16 text-center text-xs text-zinc-500">
                Không có giao dịch nào phù hợp.
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="bg-zinc-900/90 border-b border-zinc-800 text-zinc-400 font-mono text-[11px] uppercase">
                      <th className="px-4 py-3">Mã GD</th>
                      <th className="px-3 py-3">Người gửi</th>
                      <th className="px-3 py-3">Vật phẩm gửi</th>
                      <th className="px-3 py-3">Người nhận</th>
                      <th className="px-3 py-3">Vật phẩm đối ứng</th>
                      <th className="px-3 py-3">Trạng thái</th>
                      <th className="px-4 py-3">Thời gian</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-zinc-800/50">
                    {trades.map((t) => {
                      const statusBadge =
                        t.status === "COMPLETED"
                          ? "text-emerald-400 bg-emerald-500/10 border-emerald-500/30"
                          : t.status === "PENDING"
                          ? "text-amber-400 bg-amber-500/10 border-amber-500/30"
                          : "text-rose-400 bg-rose-500/10 border-rose-500/30";

                      return (
                        <tr key={t.id} className="hover:bg-zinc-800/30 transition-colors">
                          <td className="px-4 py-3 font-mono text-zinc-500 text-[10px]">
                            {t.id.slice(0, 8)}...
                          </td>
                          <td className="px-3 py-3 font-mono text-zinc-200 font-semibold">
                            {t.sender_name}
                          </td>
                          <td className="px-3 py-3">
                            {renderOfferSummary(t.sender_offer)}
                          </td>
                          <td className="px-3 py-3 font-mono text-zinc-200 font-semibold">
                            {t.receiver_name}
                          </td>
                          <td className="px-3 py-3">
                            {renderOfferSummary(t.receiver_offer)}
                          </td>
                          <td className="px-3 py-3">
                            <span className={`px-2 py-0.5 rounded-full text-[10px] font-semibold border ${statusBadge}`}>
                              {t.status}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-zinc-500 text-[11px]">
                            {new Date(t.created_at).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" })}
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
                Tổng số: <span className="font-mono text-zinc-300">{tradesTotal}</span> giao dịch
              </div>
              <div className="flex items-center gap-2">
                <span>
                  Trang {tradesPage} / {Math.max(1, Math.ceil(tradesTotal / 20))}
                </span>
                <button
                  onClick={() => setTradesPage((p) => Math.max(1, p - 1))}
                  disabled={tradesPage <= 1}
                  className="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 text-zinc-300 rounded cursor-pointer"
                >
                  Trước
                </button>
                <button
                  onClick={() => setTradesPage((p) => Math.min(Math.ceil(tradesTotal / 20), p + 1))}
                  disabled={tradesPage >= Math.ceil(tradesTotal / 20)}
                  className="px-2.5 py-1 bg-zinc-800 hover:bg-zinc-700 disabled:opacity-40 text-zinc-300 rounded cursor-pointer"
                >
                  Sau
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
