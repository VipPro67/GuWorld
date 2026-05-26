"use client";

import React, { useState, useEffect } from "react";
import { Player, useGameStore } from "@/store/useGameStore";
import { ShoppingBag, Coins, Sparkles, AlertCircle, ShoppingCart } from "lucide-react";

interface ShopItem {
  id: string;
  name: string;
  description: string;
  type: string;
  price: number;
  stock: number;
}

interface CaravanShopProps {
  player: Player;
  actionLoading: boolean;
  onBuy: (itemId: string, quantity: number) => Promise<boolean>;
}

export const CaravanShop: React.FC<CaravanShopProps> = ({
  player,
  actionLoading,
  onBuy,
}) => {
  const [shopItems, setShopItems] = useState<ShopItem[]>([]);
  const [fetchingList, setFetchingList] = useState(false);
  const [selectedItem, setSelectedItem] = useState<ShopItem | null>(null);
  const [buyQuantity, setBuyQuantity] = useState(1);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

  // Fetch shop items on mount
  useEffect(() => {
    const fetchShopList = async () => {
      setFetchingList(true);
      try {
        const response = await fetch(`${BACKEND_URL}/api/shop/list`);
        if (response.ok) {
          const data = await response.json();
          setShopItems(data);
        }
      } catch (err) {
        console.error("Failed to fetch shop list:", err);
      } finally {
        setFetchingList(false);
      }
    };
    fetchShopList();
  }, [BACKEND_URL]);

  const handleOpenBuyModal = (item: ShopItem) => {
    setSelectedItem(item);
    setBuyQuantity(1);
    setErrorMessage(null);
    setSuccessMessage(null);
  };

  const handleConfirmPurchase = async () => {
    if (!selectedItem) return;
    setErrorMessage(null);
    setSuccessMessage(null);

    const totalCost = selectedItem.price * buyQuantity;
    if (player.primeval_stones < totalCost) {
      setErrorMessage("Không đủ Nguyên Thạch để thực hiện giao dịch này.");
      return;
    }

    const success = await onBuy(selectedItem.id, buyQuantity);
    if (success) {
      setSuccessMessage("Giao dịch thành công! Đã thêm vào túi đồ.");
      // Close modal after a short delay
      setTimeout(() => {
        setSelectedItem(null);
      }, 1000);
    } else {
      setErrorMessage("Giao dịch thất bại. Hãy thử lại.");
    }
  };

  return (
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-4xl mx-auto space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
        <div className="flex items-center gap-2">
          <ShoppingBag className="w-5 h-5 text-amber-500 animate-pulse" />
          <h3 className="text-lg font-bold text-zinc-100 font-serif">
            Giả Gia Thương Đội (Caravan Beginner Shop)
          </h3>
        </div>
        <div className="flex items-center gap-1.5 text-xs text-amber-400 font-mono font-bold bg-zinc-950/40 border border-amber-500/20 px-3 py-1.5 rounded-xl">
          <Coins className="w-4 h-4 text-amber-500" />
          <span>{player.primeval_stones} Nguyên Thạch</span>
        </div>
      </div>

      <div className="text-xs text-zinc-400 max-w-xl">
        * Thương đội của Giả gia đi ngang qua Thanh Mao Sơn thôn trại, bán các nhu yếu phẩm cơ bản cho tân thủ cổ sư sinh tồn và tu luyện.
      </div>

      {fetchingList ? (
        <div className="py-12 flex flex-col items-center justify-center gap-3">
          <div className="w-8 h-8 border-4 border-amber-500 border-t-transparent rounded-full animate-spin" />
          <span className="text-xs font-mono text-zinc-500">Đang kiểm kê hàng hóa...</span>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {shopItems.map((item) => {
            const totalCost = item.price;
            const canAfford = player.primeval_stones >= totalCost;

            return (
              <div
                key={item.id}
                className="bg-zinc-950/50 border border-zinc-900 rounded-2xl p-4 flex flex-col justify-between hover:border-zinc-850 hover:bg-zinc-950/80 transition-all group"
              >
                <div className="space-y-2">
                  <div className="flex justify-between items-start">
                    <span className="text-[9px] font-mono border border-zinc-800 px-2 py-0.5 rounded text-zinc-500 uppercase tracking-wider">
                      {item.type === "Gu" ? "Cổ Trùng" : "Nguyên Liệu"}
                    </span>
                    <span className="text-xs font-mono font-bold text-amber-400 flex items-center gap-1">
                      <Coins className="w-3.5 h-3.5 text-amber-500" />
                      {item.price}
                    </span>
                  </div>

                  <h4 className="text-base font-bold text-zinc-200 font-serif group-hover:text-amber-400 transition-colors">
                    {item.name}
                  </h4>

                  <p className="text-xs text-zinc-405 leading-normal">
                    {item.description}
                  </p>
                </div>

                <div className="mt-4 pt-3 border-t border-zinc-900/60 flex items-center justify-between">
                  <span className="text-[10px] text-zinc-600 font-mono">
                    Hiện có: {item.stock > 0 ? `${item.stock} cái` : "Vô hạn"}
                  </span>
                  <button
                    type="button"
                    disabled={actionLoading}
                    onClick={() => handleOpenBuyModal(item)}
                    className={`py-1.5 px-4 text-xs font-bold rounded-xl flex items-center gap-1 transition-all cursor-pointer ${
                      canAfford
                        ? "bg-amber-600 hover:bg-amber-500 text-zinc-950 active:scale-[0.98]"
                        : "bg-zinc-900 border border-zinc-850 text-zinc-500 hover:text-zinc-400"
                    }`}
                  >
                    <ShoppingCart className="w-3.5 h-3.5" />
                    <span>Mua</span>
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Buy Quantity Modal */}
      {selectedItem && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 animate-fade-in">
          <div className="bg-zinc-950 border border-zinc-850 rounded-3xl p-6 max-w-sm w-full space-y-4">
            <div>
              <h4 className="text-lg font-bold text-zinc-100 font-serif">
                Thực Hiện Giao Dịch
              </h4>
              <p className="text-xs text-zinc-400 mt-1">
                Mua {selectedItem.name} từ Giả gia thương đội.
              </p>
            </div>

            <div className="bg-zinc-900/40 border border-zinc-850 rounded-xl p-3 flex justify-between text-xs font-mono">
              <span className="text-zinc-400">Đơn giá:</span>
              <span className="text-amber-400 font-bold flex items-center gap-1">
                <Coins className="w-3.5 h-3.5" /> {selectedItem.price} Nguyên Thạch
              </span>
            </div>

            {selectedItem.type === "Material" ? (
              <div className="space-y-1.5">
                <label className="text-[10px] font-mono text-zinc-500 uppercase">
                  Số lượng cần mua:
                </label>
                <div className="flex items-center justify-between border border-zinc-850 rounded-xl bg-zinc-900/20 overflow-hidden">
                  <button
                    type="button"
                    disabled={buyQuantity <= 1}
                    onClick={() => setBuyQuantity((prev) => prev - 1)}
                    className="w-10 h-10 flex items-center justify-center font-bold text-zinc-400 hover:bg-zinc-900 active:bg-zinc-950 disabled:opacity-30"
                  >
                    -
                  </button>
                  <span className="font-bold text-zinc-100 font-mono text-sm">
                    {buyQuantity}
                  </span>
                  <button
                    type="button"
                    onClick={() => setBuyQuantity((prev) => prev + 1)}
                    className="w-10 h-10 flex items-center justify-center font-bold text-zinc-400 hover:bg-zinc-900 active:bg-zinc-950"
                  >
                    +
                  </button>
                </div>
              </div>
            ) : (
              <div className="text-xs text-zinc-500 italic py-1">
                * Cổ trùng quý hiếm, chỉ giao dịch từng con một để đảm bảo sinh khí.
              </div>
            )}

            {/* Total Cost Display */}
            <div className="flex justify-between items-center pt-2 font-mono border-t border-dashed border-zinc-900 text-xs">
              <span className="text-zinc-400">Tổng thanh toán:</span>
              <span className="text-amber-400 font-bold flex items-center gap-1 text-sm">
                <Coins className="w-4 h-4 text-amber-500" />
                {selectedItem.price * buyQuantity} Nguyên Thạch
              </span>
            </div>

            {/* Messages */}
            {errorMessage && (
              <div className="bg-rose-950/20 border border-rose-900/40 rounded-xl p-3 flex items-start gap-2 text-[11px] text-rose-450 leading-normal font-mono animate-shake">
                <AlertCircle className="w-4 h-4 text-rose-500 shrink-0" />
                <span>{errorMessage}</span>
              </div>
            )}

            {successMessage && (
              <div className="bg-emerald-950/20 border border-emerald-900/40 rounded-xl p-3 flex items-start gap-2 text-[11px] text-emerald-450 leading-normal font-mono">
                <Sparkles className="w-4 h-4 text-emerald-500 shrink-0 animate-bounce" />
                <span>{successMessage}</span>
              </div>
            )}

            {/* Action buttons */}
            <div className="grid grid-cols-2 gap-3 pt-2">
              <button
                type="button"
                disabled={actionLoading}
                onClick={() => setSelectedItem(null)}
                className="py-2.5 rounded-xl border border-zinc-800 text-zinc-400 hover:bg-zinc-900 font-semibold text-xs active:scale-[0.98] transition-all cursor-pointer"
              >
                Hủy Bỏ
              </button>
              <button
                type="button"
                disabled={actionLoading || !!successMessage}
                onClick={handleConfirmPurchase}
                className="py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-zinc-950 font-bold text-xs active:scale-[0.98] transition-all cursor-pointer"
              >
                Xác Nhận Mua
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
