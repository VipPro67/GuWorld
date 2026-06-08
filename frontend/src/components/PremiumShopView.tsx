"use client";

import React, { useCallback, useEffect, useState } from "react";
import { Gem, Gift, Lock, ShoppingBag } from "lucide-react";
import { PremiumWallet, WalletBalance } from "./PremiumWallet";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

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

export const PremiumShopView: React.FC = () => {
  const [packages, setPackages] = useState<ShopPackage[]>([]);
  const [wallet, setWallet] = useState<WalletBalance | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [buying, setBuying] = useState<string | null>(null);

  const fetchPackages = useCallback(async () => {
    const response = await fetch(`${BACKEND_URL}/api/shop/packages`, { credentials: "include" });
    if (response.ok) setPackages(await response.json());
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => { void fetchPackages(); }, 0);
    return () => window.clearTimeout(timer);
  }, [fetchPackages]);

  const buyPackage = async (packageId: string) => {
    setBuying(packageId);
    setMessage(null);
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
      setMessage("Mua gói thành công, phần thưởng đã nhập kho.");
      await fetchPackages();
    } catch (err) {
      setMessage(err instanceof Error ? err.message : "Mua gói thất bại");
    } finally {
      setBuying(null);
    }
  };

  const rewardText = (reward: Reward) => {
    if (reward.currency) {
      const label = reward.currency === "IMMORTAL_JADE" ? "Tiên Ngọc" : reward.currency === "IMMORTAL_ESSENCE_STONE" ? "Tiên Nguyên Thạch" : "Nguyên Thạch";
      return `+${Number(reward.amount || 0).toLocaleString()} ${label}`;
    }
    return `${reward.itemName || reward.itemId} x${reward.quantity || 1}`;
  };

  return (
    <div className="mx-auto w-full max-w-6xl space-y-6 rounded-3xl border border-violet-900/40 bg-zinc-950/70 p-6 shadow-2xl">
      <div className="flex flex-col gap-4 border-b border-zinc-800 pb-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="flex items-center gap-2 font-serif text-2xl font-bold text-violet-100"><ShoppingBag className="h-6 w-6 text-violet-300" /> Shop Gói Tu Luyện</h2>
          <p className="mt-1 text-sm text-zinc-400">Dùng Tiên Ngọc để mua gói hỗ trợ tu luyện. Không có chiều đổi ngược từ tài nguyên game sang Tiên Ngọc.</p>
        </div>
        {wallet ? <PremiumWallet compact onLoaded={setWallet} /> : <PremiumWallet compact onLoaded={setWallet} />}
      </div>

      {message && <div className="rounded-2xl border border-violet-700/40 bg-violet-950/30 p-3 text-sm text-violet-100">{message}</div>}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {packages.map((pkg) => {
          const remaining = pkg.remainingPurchaseCount ?? pkg.limit.count;
          const canAfford = (wallet?.immortalJade || 0) >= pkg.priceJade;
          const blocked = remaining <= 0 || !canAfford;
          return (
            <div key={pkg.id} className="rounded-2xl border border-zinc-800 bg-zinc-950/60 p-5 transition hover:border-violet-600/50">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h3 className="font-serif text-lg font-bold text-zinc-100">{pkg.name}</h3>
                  <p className="mt-1 text-xs text-zinc-500">{pkg.description}</p>
                </div>
                <div className="flex items-center gap-1 rounded-xl border border-emerald-700/40 bg-emerald-950/30 px-3 py-1.5 text-sm font-bold text-emerald-300"><Gem className="h-4 w-4" /> {pkg.priceJade}</div>
              </div>

              <div className="mt-4 space-y-2">
                {pkg.rewards.map((reward, index) => (
                  <div key={`${pkg.id}-${index}`} className="flex items-center gap-2 rounded-xl bg-zinc-900/50 px-3 py-2 text-sm text-zinc-300"><Gift className="h-4 w-4 text-amber-300" /> {rewardText(reward)}</div>
                ))}
              </div>

              <div className="mt-4 flex flex-wrap gap-2 text-xs text-zinc-400">
                <span className="rounded-lg border border-zinc-800 px-2 py-1">Limit: {pkg.limit.count}/{pkg.limit.period}</span>
                <span className="rounded-lg border border-zinc-800 px-2 py-1">Còn: {remaining}</span>
                {pkg.requirements?.minRank && <span className="rounded-lg border border-zinc-800 px-2 py-1">Yêu cầu Rank {pkg.requirements.minRank}+</span>}
              </div>

              <button disabled={blocked || buying === pkg.id} onClick={() => buyPackage(pkg.id)} className="mt-4 flex w-full items-center justify-center gap-2 rounded-xl bg-violet-600 px-4 py-2 text-sm font-bold text-white hover:bg-violet-500 disabled:bg-zinc-800 disabled:text-zinc-500">
                {blocked ? <Lock className="h-4 w-4" /> : <ShoppingBag className="h-4 w-4" />}
                {remaining <= 0 ? "Đã hết lượt" : !canAfford ? "Không đủ Tiên Ngọc" : buying === pkg.id ? "Đang mua..." : "Mua gói"}
              </button>
            </div>
          );
        })}
      </div>
    </div>
  );
};
