import React, { useEffect, useState, useCallback } from "react";
import { authFetch } from "@/lib/auth";
import { Coins, RefreshCw, Search, CheckCircle, Clock, XCircle, ArrowUpRight, ArrowDownRight } from "lucide-react";
import { AdminServerPagination } from "./AdminServerPagination";
import { useAdminUrlState } from "./useAdminUrlState";
import { AdminServerSort } from "./AdminServerSort";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const BILLING_DEFAULTS = { tab: "orders", page: "1", page_size: "25", search: "", status: "", user_id: "", sort: "created_at", direction: "desc" };
const BILLING_KEYS = Object.keys(BILLING_DEFAULTS);

export const TopupWalletAdminTab: React.FC<{ addToast: (msg: string, type: "success" | "error" | "info") => void }> = ({ addToast }) => {
  const [query, updateQuery] = useAdminUrlState(BILLING_DEFAULTS, BILLING_KEYS);
  const subTab = query.tab === "transactions" ? "transactions" : "orders";
  const page = Math.max(1, Number(query.page) || 1);
  const pageSize = Math.max(1, Number(query.page_size) || 25);
  const [total, setTotal] = useState(0);
  const [orders, setOrders] = useState<ApiResponseData[]>([]);
  const [ordersLoading, setOrdersLoading] = useState(false);

  const [transactions, setTransactions] = useState<ApiResponseData[]>([]);
  const [txLoading, setTxLoading] = useState(false);
  const userFilter = query.user_id;

  const fetchOrders = useCallback(async () => {
    setOrdersLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: String(pageSize), search: query.search, sort: query.sort, direction: query.direction });
      if (query.status) params.set("status", query.status);
      const res = await authFetch(`${BACKEND_URL}/api/admin/topup-orders?${params}`);
      if (res.ok) {
        const body = await res.json(); setOrders(body.data || []); setTotal(body.total || 0);
      } else {
        addToast("Không thể tải đơn nạp tiền", "error");
      }
    } catch (caught: unknown) {
      const e = caught instanceof Error ? caught : new Error(String(caught));
      addToast("Lỗi kết nối đơn nạp tiền", "error");
    } finally {
      setOrdersLoading(false);
    }
  }, [addToast, page, pageSize, query.direction, query.search, query.sort, query.status]);

  const fetchTransactions = useCallback(async () => {
    setTxLoading(true);
    try {
      const params = new URLSearchParams({ page: String(page), page_size: String(pageSize), search: query.search, sort: query.sort, direction: query.direction });
      if (userFilter) params.set("user_id", userFilter);
      const url = `${BACKEND_URL}/api/admin/wallet-transactions?${params}`;
      const res = await authFetch(url);
      if (res.ok) {
        const body = await res.json(); setTransactions(body.data || []); setTotal(body.total || 0);
      } else {
        addToast("Không thể tải lịch sử ví", "error");
      }
    } catch (caught: unknown) {
      const e = caught instanceof Error ? caught : new Error(String(caught));
      addToast("Lỗi kết nối lịch sử ví", "error");
    } finally {
      setTxLoading(false);
    }
  }, [userFilter, addToast, page, pageSize, query.direction, query.search, query.sort]);

  useEffect(() => {
    if (subTab === "orders") fetchOrders();
    else fetchTransactions();
  }, [subTab, fetchOrders, fetchTransactions]);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative flex-1 min-w-56"><Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-zinc-500" /><input value={query.search} onChange={(event) => updateQuery({ search: event.target.value, page: "1" }, true)} placeholder="Tìm mã đơn, người chơi, ghi chú..." className="w-full rounded-lg border border-zinc-800 bg-zinc-950 py-2 pl-8 pr-3 text-xs" /></div>
        <AdminServerSort sort={query.sort} direction={query.direction} options={subTab === "orders" ? [{ value: "created_at", label: "Ngày tạo" }, { value: "amount_vnd", label: "Số tiền" }, { value: "status", label: "Trạng thái" }] : [{ value: "created_at", label: "Thời gian" }, { value: "amount", label: "Số lượng" }, { value: "type", label: "Loại" }]} onChange={(sort, direction) => updateQuery({ sort, direction, page: "1" })} />
      </div>
      {/* Sub-tabs nav */}
      <div className="border-b border-zinc-800/80 flex gap-0 mb-4 overflow-x-auto scrollbar-hide whitespace-nowrap">
        <button
          onClick={() => updateQuery({ tab: "orders", page: "1" })}
          className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${
            subTab === "orders" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"
          }`}
        >
          <Coins className="w-3.5 h-3.5 inline mr-1.5" />Đơn Nạp Tiền
        </button>
        <button
          onClick={() => updateQuery({ tab: "transactions", page: "1" })}
          className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${
            subTab === "transactions" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"
          }`}
        >
          <RefreshCw className="w-3.5 h-3.5 inline mr-1.5" />Nhật Ký Giao Dịch Ví
        </button>
      </div>

      {subTab === "orders" && (
        <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-5 space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
              <Coins className="w-4 h-4 text-amber-400" />
              Danh Sách Đơn Nạp Tiền (Topup Orders)
            </h3>
            <button
              onClick={fetchOrders}
              className="p-1.5 hover:bg-zinc-800 rounded-lg text-zinc-400 hover:text-zinc-200 transition-colors cursor-pointer"
            >
              <RefreshCw className={`w-4 h-4 ${ordersLoading ? "animate-spin" : ""}`} />
            </button>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-xs min-w-max">
              <thead>
                <tr className="border-b border-zinc-800 text-zinc-400 font-mono text-[11px]">
                  <th className="py-2.5 px-3 whitespace-nowrap">Mã đơn</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">User ID</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Số tiền (VND)</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Tiên Ngọc nhận</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Cổng thanh toán</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Trạng thái</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Ngày tạo</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Ngày thanh toán</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/50 text-zinc-300">
                {orders.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="text-center py-6 text-zinc-500 italic">
                      {ordersLoading ? "Đang tải dữ liệu..." : "Chưa có đơn nạp nào"}
                    </td>
                  </tr>
                ) : (
                  orders.map((o, idx) => (
                    <tr key={idx} className="hover:bg-zinc-900/50">
                      <td className="py-2.5 px-3 font-mono text-amber-400/90 whitespace-nowrap">{o.orderCode}</td>
                      <td className="py-2.5 px-3 font-mono text-zinc-400 whitespace-nowrap">{o.userId}</td>
                      <td className="py-2.5 px-3 font-mono text-emerald-400 whitespace-nowrap">
                        {Number(o.amountVnd).toLocaleString("vi-VN")} đ
                      </td>
                      <td className="py-2.5 px-3 font-mono text-sky-400 font-bold whitespace-nowrap">
                        +{Number(o.jadeAmount).toLocaleString()}
                      </td>
                      <td className="py-2.5 px-3 uppercase text-zinc-400 whitespace-nowrap">{o.provider || "payos"}</td>
                      <td className="py-2.5 px-3 whitespace-nowrap">
                        {o.status === "PAID" ? (
                          <span className="inline-flex items-center gap-1 text-emerald-400 bg-emerald-950/60 border border-emerald-800/60 px-2 py-0.5 rounded text-[10px] font-bold">
                            <CheckCircle className="w-3 h-3" /> Đã TT
                          </span>
                        ) : o.status === "CANCELLED" ? (
                          <span className="inline-flex items-center gap-1 text-rose-400 bg-rose-950/60 border border-rose-800/60 px-2 py-0.5 rounded text-[10px] font-bold">
                            <XCircle className="w-3 h-3" /> Đã hủy
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1 text-amber-400 bg-amber-950/60 border border-amber-800/60 px-2 py-0.5 rounded text-[10px] font-bold">
                            <Clock className="w-3 h-3" /> Chờ TT
                          </span>
                        )}
                      </td>
                      <td className="py-2.5 px-3 text-zinc-400 font-mono whitespace-nowrap">
                        {o.createdAt ? new Date(o.createdAt).toLocaleString("vi-VN") : "-"}
                      </td>
                      <td className="py-2.5 px-3 text-zinc-400 font-mono whitespace-nowrap">
                        {o.paidAt ? new Date(o.paidAt).toLocaleString("vi-VN") : "-"}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {subTab === "transactions" && (
        <div className="bg-zinc-900/40 border border-zinc-800/60 rounded-xl p-5 space-y-4">
          <div className="flex items-center justify-between flex-wrap gap-3">
            <h3 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
              <RefreshCw className="w-4 h-4 text-sky-400" />
              Nhật Ký Giao Dịch Ví (Wallet Transactions)
            </h3>
            <div className="flex items-center gap-2">
              <div className="relative">
                <Search className="w-3.5 h-3.5 absolute left-3 top-2.5 text-zinc-500" />
                <input
                  type="text"
                  placeholder="Lọc theo Player UUID..."
                  value={userFilter}
                  onChange={(e) => updateQuery({ user_id: e.target.value, page: "1" }, true)}
                  className="pl-8 pr-3 py-1.5 bg-zinc-900 border border-zinc-800 rounded-lg text-xs text-zinc-200 focus:outline-none focus:border-amber-500/50 w-64"
                />
              </div>
              <button
                onClick={fetchTransactions}
                className="p-1.5 hover:bg-zinc-800 rounded-lg text-zinc-400 hover:text-zinc-200 transition-colors cursor-pointer"
              >
                <RefreshCw className={`w-4 h-4 ${txLoading ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse text-xs min-w-max">
              <thead>
                <tr className="border-b border-zinc-800 text-zinc-400 font-mono text-[11px]">
                  <th className="py-2.5 px-3 whitespace-nowrap">Loại GD</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">User ID</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Loại tiền</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Số lượng</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Số dư Trước &rarr; Sau</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Ghi chú</th>
                  <th className="py-2.5 px-3 whitespace-nowrap">Thời gian</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-800/50 text-zinc-300">
                {transactions.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="text-center py-6 text-zinc-500 italic">
                      {txLoading ? "Đang tải dữ liệu..." : "Chưa có giao dịch nào"}
                    </td>
                  </tr>
                ) : (
                  transactions.map((t, idx) => {
                    const isPositive = Number(t.amount) > 0;
                    return (
                      <tr key={idx} className="hover:bg-zinc-900/50">
                        <td className="py-2.5 px-3 font-mono font-bold whitespace-nowrap">
                          <span
                            className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] ${
                              isPositive
                                ? "bg-emerald-950/60 text-emerald-400 border border-emerald-800/60"
                                : "bg-rose-950/60 text-rose-400 border border-rose-800/60"
                            }`}
                          >
                            {isPositive ? <ArrowUpRight className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
                            {t.type}
                          </span>
                        </td>
                        <td className="py-2.5 px-3 font-mono text-zinc-400 whitespace-nowrap">{t.userId}</td>
                        <td className="py-2.5 px-3 uppercase text-amber-400 font-mono whitespace-nowrap">{t.currency}</td>
                        <td className={`py-2.5 px-3 font-mono font-bold whitespace-nowrap ${isPositive ? "text-emerald-400" : "text-rose-400"}`}>
                          {isPositive ? "+" : ""}{Number(t.amount).toLocaleString()}
                        </td>
                        <td className="py-2.5 px-3 font-mono text-zinc-400 whitespace-nowrap">
                          {Number(t.balanceBefore).toLocaleString()} → {Number(t.balanceAfter).toLocaleString()}
                        </td>
                        <td className="py-2.5 px-3 text-zinc-300 max-w-xs truncate whitespace-nowrap" title={t.note || ""}>{t.note || "-"}</td>
                        <td className="py-2.5 px-3 text-zinc-400 font-mono whitespace-nowrap">
                          {t.createdAt ? new Date(t.createdAt).toLocaleString("vi-VN") : "-"}
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
      <AdminServerPagination page={page} pageSize={pageSize} total={total} loading={ordersLoading || txLoading}
        onPageChange={(next) => updateQuery({ page: String(next) })}
        onPageSizeChange={(next) => updateQuery({ page: "1", page_size: String(next) })} />
    </div>
  );
};
