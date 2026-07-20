"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Gem, ShoppingBag, QrCode, RefreshCw, ShieldCheck, Gift, Lock } from "lucide-react";
import { PremiumWallet, WalletBalance } from "./PremiumWallet";
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

interface Reward {
  currency?: string;
  amount?: number;
  itemId?: string;
  itemName?: string;
  quantity?: number;
  note?: string;
}

interface ShopPackage {
  id: string;
  name: string;
  priceJade: number;
  rewards: Reward[];
  requirements?: { minRank?: number; minSubRank?: number };
  limit: { count: number; period: string };
  description: string;
  remainingPurchaseCount?: number;
}

interface PremiumStoreViewProps {
  defaultTab?: "topup" | "packages";
  successMode?: boolean;
}

export const PremiumStoreView: React.FC<PremiumStoreViewProps> = ({
  defaultTab = "packages",
  successMode = false,
}) => {
  const [activeTab, setActiveTab] = useState<"topup" | "packages">(defaultTab);
  const [wallet, setWallet] = useState<WalletBalance | null>(null);

  // Topup State
  const [topupPackages, setTopupPackages] = useState<TopupPackage[]>([]);
  const [order, setOrder] = useState<TopupOrder | null>(null);
  const [topupMessage, setTopupMessage] = useState<string | null>(null);
  const [loadingPackage, setLoadingPackage] = useState<string | null>(null);

  // Premium Shop State
  const [shopPackages, setShopPackages] = useState<ShopPackage[]>([]);
  const [shopMessage, setShopMessage] = useState<string | null>(null);
  const [buyingPackage, setBuyingPackage] = useState<string | null>(null);



  const orderCodeFromUrl = useMemo(
    () => (typeof window !== "undefined" ? new URLSearchParams(window.location.search).get("orderCode") : null),
    []
  );

  // Fetch Topup Packages
  useEffect(() => {
    fetch(`${BACKEND_URL}/api/topup/packages`, { credentials: "include" })
      .then((res) => res.json())
      .then(setTopupPackages)
      .catch(() => setTopupMessage("Không tải được danh sách gói nạp."));
  }, []);

  // Fetch Shop Packages
  const fetchShopPackages = useCallback(async () => {
    const response = await fetch(`${BACKEND_URL}/api/shop/packages`, { credentials: "include" });
    if (response.ok) setShopPackages(await response.json());
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void fetchShopPackages();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [fetchShopPackages]);

  // Refresh Order Status
  const refreshOrder = useCallback(
    async (code = order?.orderCode || orderCodeFromUrl) => {
      if (!code) return;
      const response = await fetch(`${BACKEND_URL}/api/topup/orders/${code}`, { credentials: "include" });
      if (response.ok) {
        setOrder(await response.json());
        setTopupMessage(null);
      } else {
        setTopupMessage("Không tìm thấy đơn nạp hoặc đạo hữu không sở hữu đơn này.");
      }
    },
    [order?.orderCode, orderCodeFromUrl]
  );

  useEffect(() => {
    if (!successMode || !orderCodeFromUrl) return;
    setActiveTab("topup");
    const timer = window.setTimeout(() => {
      void refreshOrder(orderCodeFromUrl);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [successMode, orderCodeFromUrl, refreshOrder]);

  // Create payOS order
  const createOrder = async (packageId: string) => {
    setLoadingPackage(packageId);
    setTopupMessage(null);
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
      setTopupMessage(err instanceof Error ? err.message : "Tạo đơn thất bại");
    } finally {
      setLoadingPackage(null);
    }
  };

  // Buy Premium Package
  const buyPackage = async (packageId: string) => {
    setBuyingPackage(packageId);
    setShopMessage(null);
    try {
      const response = await fetch(`${BACKEND_URL}/api/shop/buy-package`, {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ packageId }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Mua gói thất bại");
      setWallet(data.wallet);
      setShopMessage("Mua gói thành công, phần thưởng đã nhập kho.");
      await fetchShopPackages();
    } catch (err) {
      setShopMessage(err instanceof Error ? err.message : "Mua gói thất bại");
    } finally {
      setBuyingPackage(null);
    }
  };

  const rewardText = (reward: Reward) => {
    if (reward.currency) {
      const label =
        reward.currency === "IMMORTAL_JADE"
          ? "Tiên Ngọc"
          : reward.currency === "IMMORTAL_ESSENCE_STONE"
          ? "Tiên Nguyên Thạch"
          : "Nguyên Thạch";
      return `+${Number(reward.amount || 0).toLocaleString()} ${label}`;
    }
    return `${reward.itemName || reward.itemId} x${reward.quantity || 1}`;
  };

  return (
    <div className="mx-auto w-full max-w-6xl space-y-6 rounded-3xl border border-zinc-800 bg-zinc-950/70 p-6 shadow-2xl backdrop-blur-md">
      {/* Header & Wallet */}
      <div className="flex flex-col gap-4 border-b border-zinc-850 pb-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="font-serif text-2xl font-bold text-zinc-100 flex items-center gap-2">
            🏛️ Cát Bảo Các
          </h2>
          <p className="mt-1 text-xs text-zinc-400 font-mono">
            Tiên nhân bảo các — Nạp Tiên Ngọc qua VietQR và đổi các gói phúc lợi trợ lực tu hành.
          </p>
        </div>
        <PremiumWallet compact onLoaded={setWallet} />
      </div>

      {/* Tabs Switcher */}
      <div className="flex border-b border-zinc-900 gap-2 p-1 bg-zinc-900/30 rounded-xl w-fit">
        <button
          onClick={() => {
            setActiveTab("packages");
            setShopMessage(null);
          }}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-mono font-bold transition-all cursor-pointer ${
            activeTab === "packages"
              ? "bg-violet-950/80 border border-violet-500/30 text-violet-200"
              : "text-zinc-400 hover:text-zinc-200"
          }`}
        >
          <ShoppingBag className="h-4 w-4" />
          Gói Tu Luyện
        </button>
        <button
          onClick={() => {
            setActiveTab("topup");
            setTopupMessage(null);
          }}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-mono font-bold transition-all cursor-pointer ${
            activeTab === "topup"
              ? "bg-emerald-950/80 border border-emerald-500/30 text-emerald-200"
              : "text-zinc-400 hover:text-zinc-200"
          }`}
        >
          <Gem className="h-4 w-4" />
          Nạp Tiên Ngọc
        </button>
      </div>

      {/* Tab 1: Gói Tu Luyện */}
      {activeTab === "packages" && (
        <div className="space-y-6">
          {shopMessage && (
            <div className="rounded-2xl border border-violet-700/40 bg-violet-950/30 p-3 text-sm text-violet-100 font-mono">
              {shopMessage}
            </div>
          )}

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {shopPackages.map((pkg) => {
              const remaining = pkg.remainingPurchaseCount ?? pkg.limit.count;
              const canAfford = (wallet?.immortalJade ?? 0) >= pkg.priceJade;
              const blocked = remaining <= 0 || !canAfford;
              return (
                <div
                  key={pkg.id}
                  className="rounded-2xl border border-zinc-900 bg-zinc-950/60 p-5 hover:border-violet-600/30 transition-all text-left flex flex-col justify-between"
                >
                  <div>
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <h3 className="font-serif text-lg font-bold text-zinc-100">{pkg.name}</h3>
                        <p className="mt-1 text-xs text-zinc-500">{pkg.description}</p>
                      </div>
                      <div className="flex items-center gap-1 rounded-xl border border-emerald-700/40 bg-emerald-950/30 px-3 py-1.5 text-sm font-bold text-emerald-300">
                        <Gem className="h-4 w-4" /> {pkg.priceJade}
                      </div>
                    </div>

                    <div className="mt-4 space-y-2">
                      {pkg.rewards.map((reward, index) => (
                        <div
                          key={`${pkg.id}-${index}`}
                          className="flex items-center gap-2 rounded-xl bg-zinc-900/50 px-3 py-2 text-xs text-zinc-300 font-mono"
                        >
                          <Gift className="h-4 w-4 text-amber-300" /> {rewardText(reward)}
                        </div>
                      ))}
                    </div>
                  </div>

                  <div>
                    <div className="mt-4 flex flex-wrap gap-2 text-[10px] text-zinc-500 font-mono">
                      <span className="rounded-lg border border-zinc-900 px-2 py-0.5">
                        Giới hạn: {pkg.limit.count}/{pkg.limit.period}
                      </span>
                      <span className="rounded-lg border border-zinc-900 px-2 py-0.5">Còn: {remaining} lượt</span>
                    </div>

                    <button
                      disabled={blocked || buyingPackage === pkg.id}
                      onClick={() => buyPackage(pkg.id)}
                      className="mt-4 flex w-full items-center justify-center gap-2 rounded-xl bg-violet-650 hover:bg-violet-650 px-4 py-2 text-xs font-mono font-bold text-white disabled:bg-zinc-900 disabled:text-zinc-650 transition-colors"
                    >
                      {blocked ? <Lock className="h-3.5 w-3.5" /> : <ShoppingBag className="h-3.5 w-3.5" />}
                      {remaining <= 0
                        ? "Đã hết lượt mua"
                        : !canAfford
                        ? "Không đủ Tiên Ngọc"
                        : buyingPackage === pkg.id
                        ? "Đang mua..."
                        : "Mua gói phúc lợi"}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Tab 2: Nạp Tiên Ngọc */}
      {activeTab === "topup" && (
        <div className="space-y-6">
          {topupMessage && (
            <div className="rounded-2xl border border-red-700/40 bg-red-950/30 p-3 text-sm text-red-200 font-mono">
              {topupMessage}
            </div>
          )}

          {successMode && (
            <div className="rounded-2xl border border-amber-700/40 bg-amber-950/20 p-4 text-xs text-amber-100 font-mono space-y-2">
              <div className="flex items-center gap-2 font-bold text-sm">
                <ShieldCheck className="h-4 w-4 text-amber-400" /> Đang xác minh thanh toán
              </div>
              <p className="text-zinc-400 leading-relaxed">
                Sau khi chuyển khoản thành công, payOS cần vài giây để đồng bộ trạng thái qua webhook. Hãy nhấn làm mới để cập nhật số dư.
              </p>
              <button
                onClick={() => refreshOrder()}
                className="flex items-center gap-2 rounded-xl border border-amber-600/40 px-3 py-2 text-xs text-amber-200 hover:bg-amber-900/30"
              >
                <RefreshCw className="h-3.5 w-3.5" /> Làm mới trạng thái đơn nạp
              </button>
            </div>
          )}

          {order && (
            <div className="grid gap-3 rounded-2xl border border-emerald-700/30 bg-emerald-950/20 p-4 text-xs font-mono text-left md:grid-cols-2">
              <div>
                <b>Mã đơn:</b> {order.orderCode}
              </div>
              <div>
                <b>Trạng thái đơn:</b>{" "}
                <span className={order.status === "PAID" ? "text-emerald-300 font-bold" : "text-amber-300"}>
                  {order.status === "PAID" ? "ĐÃ THANH TOÁN" : "CHỜ THANH TOÁN"}
                </span>
              </div>
              <div>
                <b>Số tiền VND:</b> {order.amountVnd.toLocaleString()} VND
              </div>
              <div>
                <b>Tiên Ngọc nhận được:</b> +{order.jadeAmount.toLocaleString()}
              </div>
              {order.checkoutUrl && (
                <a className="text-emerald-300 underline font-bold" href={order.checkoutUrl} target="_blank" rel="noreferrer">
                  Mở cổng thanh toán payOS VietQR
                </a>
              )}
              {order.qrCode && (
                <div className="flex items-center gap-2 text-zinc-400">
                  <QrCode className="h-4 w-4" /> QR payload đã sẵn sàng trên trang checkout payOS.
                </div>
              )}
            </div>
          )}

          {!successMode && (
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
              {topupPackages?.map((pkg) => (
                <div
                  key={pkg.id}
                  className="rounded-2xl border border-zinc-900 bg-zinc-950/60 p-5 hover:border-emerald-600/30 transition-all text-left flex flex-col justify-between"
                >
                  <div>
                    <div className="flex items-center justify-between">
                      <h3 className="font-serif text-lg font-bold text-zinc-100">{pkg.name}</h3>
                      <Gem className="h-5 w-5 text-emerald-400" />
                    </div>
                    <div className="mt-4 text-3xl font-black text-emerald-300 font-mono">
                      {pkg.jadeAmount.toLocaleString()}
                    </div>
                    <div className="text-[10px] text-zinc-500 font-mono uppercase tracking-wider">Tiên Ngọc</div>
                    <div className="mt-3 font-mono text-amber-300 text-sm font-bold">
                      {pkg.amountVnd.toLocaleString()} VND
                    </div>
                  </div>
                  <button
                    disabled={loadingPackage === pkg.id}
                    onClick={() => createOrder(pkg.id)}
                    className="mt-5 w-full rounded-xl bg-emerald-650 hover:bg-emerald-650 px-4 py-2 text-xs font-mono font-bold text-white disabled:opacity-60 transition-colors"
                  >
                    {loadingPackage === pkg.id ? "Đang tạo đơn..." : "Nạp ngay"}
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
};
