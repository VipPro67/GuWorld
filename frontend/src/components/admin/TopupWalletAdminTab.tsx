import React, { useEffect, useState, useCallback } from "react";
import { authFetch } from "@/lib/auth";
import { Coins, RefreshCw, Search, CheckCircle, Clock, XCircle, ArrowUpRight, ArrowDownRight } from "lucide-react";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

export const TopupWalletAdminTab: React.FC<{ addToast: (msg: string, type: "success" | "error" | "info") => void }> = ({ addToast }) => {
  const [subTab, setSubTab] = useState<"orders" | "transactions">("orders");
  const [orders, setOrders] = useState<ApiResponseData[]>([]);
  const [ordersLoading, setOrdersLoading] = useState(false);

  const [transactions, setTransactions] = useState<ApiResponseData[]>([]);
  const [txLoading, setTxLoading] = useState(false);
  const [userFilter, setUserFilter] = useState("");

  const fetchOrders = useCallback(async () => {
    setOrdersLoading(true);
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/topup-orders`);
      if (res.ok) {
        setOrders(await res.json());
      } else {
        addToast("Không thể tải đơn nạp tiền", "error");
      }
    } catch (caught: unknown) {
      const e = caught instanceof Error ? caught : new Error(String(caught));
      addToast("Lỗi kết nối đơn nạp tiền", "error");
    } finally {
      setOrdersLoading(false);
    }
  }, [addToast]);

  const fetchTransactions = useCallback(async () => {
    setTxLoading(true);
    try {
      const url = userFilter ? `${BACKEND_URL}/api/admin/wallet-transactions?userId=${encodeURIComponent(userFilter)}` : `${BACKEND_URL}/api/admin/wallet-transactions`;
      const res = await authFetch(url);
      if (res.ok) {
        setTransactions(await res.json());
      } else {
        addToast("Không thể tải lịch sử ví", "error");
      }
    } catch (caught: unknown) {
      const e = caught instanceof Error ? caught : new Error(String(caught));
      addToast("Lỗi kết nối lịch sử ví", "error");
    } finally {
      setTxLoading(false);
    }
  }, [userFilter, addToast]);

  useEffect(() => {
    if (subTab === "orders") fetchOrders();
    else fetchTransactions();
  }, [subTab, fetchOrders, fetchTransactions]);

  return (
    <div className="space-y-4">
      {/* Sub-tabs nav */}
      <div className="border-b border-zinc-800/80 flex gap-0 mb-4 overflow-x-auto scrollbar-hide whitespace-nowrap">
        <button
          onClick={() => setSubTab("orders")}
          className={`px-4 py-2.5 text-xs font-semibold font-mono border-b-2 transition-colors cursor-pointer ${
            subTab === "orders" ? "text-amber-400 border-amber-500" : "text-zinc-500 border-transparent hover:text-zinc-300"
          }`}
        >
          <Coins className="w-3.5 h-3.5 inline mr-1.5" />Đơn Nạp Tiền
        </button>
        <button
          onClick={() => setSubTab("transactions")}
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
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="border-b border-zinc-800 text-zinc-400 font-mono text-[11px]">
                  <th className="py-2.5 px-3">Mã đơn</th>
                  <th className="py-2.5 px-3">User ID</th>
                  <th className="py-2.5 px-3">Số tiền (VND)</th>
                  <th className="py-2.5 px-3">Tiên Ngọc nhận</th>
                  <th className="py-2.5 px-3">Cổng thanh toán</th>
                  <th className="py-2.5 px-3">Trạng thái</th>
                  <th className="py-2.5 px-3">Ngày tạo</th>
                  <th className="py-2.5 px-3">Ngày thanh toán</th>
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
                      <td className="py-2.5 px-3 font-mono text-amber-400/90">{o.orderCode}</td>
                      <td className="py-2.5 px-3 font-mono text-zinc-400">{o.userId}</td>
                      <td className="py-2.5 px-3 font-mono text-emerald-400">
                        {Number(o.amountVnd).toLocaleString("vi-VN")} đ
                      </td>
                      <td className="py-2.5 px-3 font-mono text-sky-400 font-bold">
                        +{Number(o.jadeAmount).toLocaleString()}
                      </td>
                      <td className="py-2.5 px-3 uppercase text-zinc-400">{o.provider || "payos"}</td>
                      <td className="py-2.5 px-3">
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
                      <td className="py-2.5 px-3 text-zinc-400 font-mono">
                        {o.createdAt ? new Date(o.createdAt).toLocaleString("vi-VN") : "-"}
                      </td>
                      <td className="py-2.5 px-3 text-zinc-400 font-mono">
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
                  onChange={(e) => setUserFilter(e.target.value)}
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
            <table className="w-full text-left border-collapse text-xs">
              <thead>
                <tr className="border-b border-zinc-800 text-zinc-400 font-mono text-[11px]">
                  <th className="py-2.5 px-3">Loại GD</th>
                  <th className="py-2.5 px-3">User ID</th>
                  <th className="py-2.5 px-3">Loại tiền</th>
                  <th className="py-2.5 px-3">Số lượng</th>
                  <th className="py-2.5 px-3">Số dư Trước &rarr; Sau</th>
                  <th className="py-2.5 px-3">Ghi chú</th>
                  <th className="py-2.5 px-3">Thời gian</th>
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
                        <td className="py-2.5 px-3 font-mono font-bold">
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
                        <td className="py-2.5 px-3 font-mono text-zinc-400">{t.userId}</td>
                        <td className="py-2.5 px-3 uppercase text-amber-400 font-mono">{t.currency}</td>
                        <td className={`py-2.5 px-3 font-mono font-bold ${isPositive ? "text-emerald-400" : "text-rose-400"}`}>
                          {isPositive ? "+" : ""}{Number(t.amount).toLocaleString()}
                        </td>
                        <td className="py-2.5 px-3 font-mono text-zinc-400">
                          {Number(t.balanceBefore).toLocaleString()} → {Number(t.balanceAfter).toLocaleString()}
                        </td>
                        <td className="py-2.5 px-3 text-zinc-300 max-w-xs truncate">{t.note || "-"}</td>
                        <td className="py-2.5 px-3 text-zinc-400 font-mono">
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
    </div>
  );
};
