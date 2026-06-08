"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Gem, QrCode, RefreshCw, ShieldCheck } from "lucide-react";
import { PremiumWallet } from "./PremiumWallet";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

interface TopupPackage {
  id: string;
  name: string;
  amountVnd: number;
  jadeAmount: number;
}

interface TopupOrder {
  orderCode: number;
  amountVnd: number;
  jadeAmount: number;
  checkoutUrl: string;
  qrCode: string;
  status: string;
}

interface TopupViewProps {
  successMode?: boolean;
}

export const TopupView: React.FC<TopupViewProps> = ({ successMode = false }) => {
  const [packages, setPackages] = useState<TopupPackage[]>([]);
  const [order, setOrder] = useState<TopupOrder | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [loadingPackage, setLoadingPackage] = useState<string | null>(null);

  const orderCodeFromUrl = useMemo(() => (typeof window !== "undefined" ? new URLSearchParams(window.location.search).get("orderCode") : null), []);

  useEffect(() => {
    fetch(`${BACKEND_URL}/api/topup/packages`, { credentials: "include" })
      .then((res) => res.json())
      .then(setPackages)
      .catch(() => setMessage("Không tải được danh sách gói nạp."));
  }, []);

  const refreshOrder = useCallback(async (code = order?.orderCode || orderCodeFromUrl) => {
    if (!code) return;
    const response = await fetch(`${BACKEND_URL}/api/topup/orders/${code}`, { credentials: "include" });
    if (response.ok) {
      setOrder(await response.json());
      setMessage(null);
    } else {
      setMessage("Không tìm thấy đơn nạp hoặc đạo hữu không sở hữu đơn này.");
    }
  }, [order?.orderCode, orderCodeFromUrl]);

  useEffect(() => {
    if (!successMode || !orderCodeFromUrl) return;
    const timer = window.setTimeout(() => { void refreshOrder(orderCodeFromUrl); }, 0);
    return () => window.clearTimeout(timer);
  }, [successMode, orderCodeFromUrl, refreshOrder]);

  const createOrder = async (packageId: string) => {
    setLoadingPackage(packageId);
    setMessage(null);
    try {
      const response = await fetch(`${BACKEND_URL}/api/topup/orders`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ packageId }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Tạo đơn thất bại");
      setOrder(data);
      if (data.checkoutUrl) window.location.assign(data.checkoutUrl);
    } catch (err) {
      setMessage(err instanceof Error ? err.message : "Tạo đơn thất bại");
    } finally {
      setLoadingPackage(null);
    }
  };

  return (
    <div className="mx-auto w-full max-w-5xl space-y-6 rounded-3xl border border-emerald-900/40 bg-zinc-950/70 p-6 shadow-2xl">
      <div className="flex flex-col gap-3 border-b border-zinc-800 pb-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="font-serif text-2xl font-bold text-emerald-100">Nạp Tiên Ngọc</h2>
          <p className="mt-1 text-sm text-zinc-400">Thanh toán VND qua payOS/VietQR. Return URL chỉ để hiển thị, ví chỉ cộng sau webhook hợp lệ.</p>
        </div>
        <PremiumWallet compact />
      </div>

      {message && <div className="rounded-2xl border border-red-700/40 bg-red-950/30 p-3 text-sm text-red-200">{message}</div>}

      {successMode && (
        <div className="rounded-2xl border border-amber-700/40 bg-amber-950/20 p-4 text-sm text-amber-100">
          <div className="flex items-center gap-2 font-semibold"><ShieldCheck className="h-4 w-4" /> Đang xác nhận thanh toán</div>
          <p className="mt-1 text-zinc-400">Nếu webhook payOS chưa về, trạng thái có thể vẫn là PENDING. Bấm làm mới, hệ thống không cộng tiền từ trang này.</p>
          <button onClick={() => refreshOrder()} className="mt-3 flex items-center gap-2 rounded-xl border border-amber-600/40 px-3 py-2 text-xs text-amber-200 hover:bg-amber-900/30"><RefreshCw className="h-3.5 w-3.5" /> Làm mới trạng thái</button>
        </div>
      )}

      {order && (
        <div className="grid gap-3 rounded-2xl border border-emerald-700/30 bg-emerald-950/20 p-4 text-sm md:grid-cols-2">
          <div><b>Mã đơn:</b> {order.orderCode}</div>
          <div><b>Trạng thái:</b> <span className={order.status === "PAID" ? "text-emerald-300" : "text-amber-300"}>{order.status}</span></div>
          <div><b>Số tiền:</b> {order.amountVnd.toLocaleString()} VND</div>
          <div><b>Tiên Ngọc:</b> +{order.jadeAmount.toLocaleString()}</div>
          {order.checkoutUrl && <a className="text-emerald-300 underline" href={order.checkoutUrl}>Mở trang thanh toán payOS</a>}
          {order.qrCode && <div className="flex items-center gap-2 text-zinc-400"><QrCode className="h-4 w-4" /> QR payload đã sẵn sàng trong checkout payOS.</div>}
        </div>
      )}

      {!successMode && (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          {packages.map((pkg) => (
            <div key={pkg.id} className="rounded-2xl border border-zinc-800 bg-zinc-950/60 p-5 transition hover:border-emerald-600/50">
              <div className="flex items-center justify-between">
                <h3 className="font-serif text-lg font-bold text-zinc-100">{pkg.name}</h3>
                <Gem className="h-5 w-5 text-emerald-300" />
              </div>
              <div className="mt-4 text-3xl font-black text-emerald-300">{pkg.jadeAmount}</div>
              <div className="text-xs text-zinc-500">Tiên Ngọc</div>
              <div className="mt-3 font-mono text-amber-300">{pkg.amountVnd.toLocaleString()} VND</div>
              <button disabled={loadingPackage === pkg.id} onClick={() => createOrder(pkg.id)} className="mt-4 w-full rounded-xl bg-emerald-600 px-4 py-2 text-sm font-bold text-white hover:bg-emerald-500 disabled:opacity-60">
                {loadingPackage === pkg.id ? "Đang tạo QR..." : "Nạp"}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
