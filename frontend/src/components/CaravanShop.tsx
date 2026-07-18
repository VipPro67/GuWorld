"use client";

import React, { useState, useEffect } from "react";
import { createPortal } from "react-dom";
import { Player, useGameStore } from "@/store/useGameStore";
import { ShoppingBag, Coins, Sparkles, AlertCircle, ShoppingCart, ArrowLeftRight, Trash2, Search } from "lucide-react";

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
  onSell: (itemId: string, quantity: number, itemType: "Gu" | "Material") => Promise<boolean>;
}

export const CaravanShop: React.FC<CaravanShopProps> = ({
  player,
  actionLoading,
  onBuy,
  onSell,
}) => {
  const { guDictionary, itemDictionary } = useGameStore();
  const [shopItems, setShopItems] = useState<ShopItem[]>([]);
  const [fetchingList, setFetchingList] = useState(false);
  const [mode, setMode] = useState<"buy" | "sell">("buy");
  const [shopTab, setShopTab] = useState<"Gu" | "Material" | "Consumable">("Gu");
  const [sellTab, setSellTab] = useState<"Gu" | "Material">("Gu");
  const [sellSearch, setSellSearch] = useState("");

  // Selection states for Modals
  const [selectedBuyItem, setSelectedBuyItem] = useState<ShopItem | null>(null);
  const [buyQuantity, setBuyQuantity] = useState(1);

  const [selectedSellItem, setSelectedSellItem] = useState<{
    id: string;
    name: string;
    owned: number;
    price: number;
    type: "Gu" | "Material";
  } | null>(null);
  const [sellQuantity, setSellQuantity] = useState(1);

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

  const sellBackRatio = 0.10;

  const guRankBasePrice = [0, 200, 500, 1200, 2500, 5000, 10000, 25000, 60000, 150000];
  const itemRankBasePrice = [0, 5, 15, 30, 60, 120, 250, 500, 1000, 2500];

  const getGuBasePrice = (guId: string): number => {
    const spec = guDictionary[guId];
    if (!spec) return 200;
    if (spec.base_price && spec.base_price > 0) return spec.base_price;
    return guRankBasePrice[spec.rank || 1] || 200;
  };

  const getItemBasePrice = (itemId: string): number => {
    const spec = itemDictionary[itemId];
    if (!spec) return 5;
    if (spec.base_price && spec.base_price > 0) return spec.base_price;
    return itemRankBasePrice[spec.rank || 1] || 5;
  };

  // SellPrice = basePrice * 0.10, minimum 1 (matches backend pricing.SellPrice)
  const getSellPrice = (basePrice: number): number => Math.max(1, Math.floor(basePrice * sellBackRatio));

  // Helper to calculate exact sell price matching pricing.go
  const getItemSellPrice = (itemId: string, type: "Gu" | "Material"): number => {
    const basePrice = type === "Gu" ? getGuBasePrice(itemId) : getItemBasePrice(itemId);
    return getSellPrice(basePrice);
  };

  // Filtered lists for Buy Mode
  const filteredBuyItems = shopItems.filter((item) => {
    if (shopTab === "Material") {
      return item.type === "Material" || item.type === "Food" || item.type === "food";
    }
    return item.type === shopTab;
  });

  // Filtered lists for Sell Mode
  const sellableMaterials = Object.entries(player.materials_inventory || {})
    .filter(([_, qty]) => qty > 0)
    .map(([itemId, qty]) => {
      const spec = itemDictionary[itemId];
      return {
        id: itemId,
        name: spec?.name || itemId,
        description: spec?.description || "Vật phẩm tu sĩ.",
        type: "Material" as const,
        owned: qty,
        price: getItemSellPrice(itemId, "Material"),
      };
    });

  const sellableGus = Object.entries(
    (player.gu_inventory || [])
      .filter((gu) => !gu.is_vital && gu.status === "ACTIVE" && !gu.equipped)
      .reduce((acc, gu) => {
        acc[gu.gu_id] = (acc[gu.gu_id] || 0) + 1;
        return acc;
      }, {} as Record<string, number>)
  ).map(([guId, count]) => {
    const spec = guDictionary[guId];
    return {
      id: guId,
      name: spec?.name || guId,
      description: spec?.description || "Cổ trùng bản thân nuôi dưỡng.",
      type: "Gu" as const,
      owned: count,
      price: getItemSellPrice(guId, "Gu"),
    };
  });

  const handleOpenBuyModal = (item: ShopItem) => {
    setSelectedBuyItem(item);
    setBuyQuantity(1);
    setErrorMessage(null);
    setSuccessMessage(null);
  };

  const handleConfirmPurchase = async () => {
    if (!selectedBuyItem) return;
    setErrorMessage(null);
    setSuccessMessage(null);

    const totalCost = selectedBuyItem.price * buyQuantity;
    if (player.primeval_stones < totalCost) {
      setErrorMessage("Không đủ Nguyên Thạch để thực hiện giao dịch này.");
      return;
    }

    const success = await onBuy(selectedBuyItem.id, buyQuantity);
    if (success) {
      setSuccessMessage("Giao dịch thành công! Đã thêm vào túi đồ.");
      setTimeout(() => {
        setSelectedBuyItem(null);
      }, 1000);
    } else {
      setErrorMessage("Giao dịch thất bại. Bách Bảo Nang có thể đã đầy hoặc lỗi hệ thống.");
    }
  };

  const normalizeText = (text: string) => text.toLowerCase().trim();

  const matchSellItem = (item: { id: string; name: string; description: string }, query: string): boolean => {
    if (!query) return true;
    const q = normalizeText(query);
    return normalizeText(item.name).includes(q) || normalizeText(item.id).includes(q);
  };

  const filteredSellGus = sellableGus.filter((item) => matchSellItem(item, sellSearch));
  const filteredSellMaterials = sellableMaterials.filter((item) => matchSellItem(item, sellSearch));

  const handleOpenSellModal = (item: typeof sellableGus[0] | typeof sellableMaterials[0]) => {
    setSelectedSellItem(item);
    setSellQuantity(1);
    setErrorMessage(null);
    setSuccessMessage(null);
  };

  const handleConfirmSell = async () => {
    if (!selectedSellItem) return;
    setErrorMessage(null);
    setSuccessMessage(null);

    if (sellQuantity > selectedSellItem.owned) {
      setErrorMessage("Số lượng bán vượt quá số lượng hiện có.");
      return;
    }

    const success = await onSell(selectedSellItem.id, sellQuantity, selectedSellItem.type);
    if (success) {
      setSuccessMessage(`Đã bán thành công! Nhận về +${selectedSellItem.price * sellQuantity} Nguyên Thạch.`);
      setTimeout(() => {
        setSelectedSellItem(null);
      }, 1000);
    } else {
      setErrorMessage("Giao dịch thất bại. Hãy thử lại sau.");
    }
  };

  return (
    <>
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-4xl mx-auto space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between border-b border-zinc-800 pb-4 gap-4">
        <div className="flex items-center gap-2">
          <ShoppingBag className="w-5 h-5 text-amber-500 animate-pulse" />
          <h3 className="text-lg font-bold text-zinc-100 font-serif">
            Giả Gia Thương Đội
          </h3>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5 text-xs text-amber-400 font-mono font-bold bg-zinc-950/40 border border-amber-500/20 px-3 py-1.5 rounded-xl">
            <Coins className="w-4 h-4 text-amber-500" />
            <span>{player.primeval_stones} Nguyên Thạch</span>
          </div>
          <div className="flex items-center gap-1.5 text-xs text-cyan-300 font-mono font-bold bg-zinc-950/40 border border-cyan-500/20 px-3 py-1.5 rounded-xl">
            <Sparkles className="w-4 h-4 text-violet-400" />
            <span>{(player.immortal_stones || 0).toLocaleString()} Tiên Nguyên Thạch</span>
          </div>
        </div>
      </div>

      {/* Mode Switcher: Buy / Sell */}
      <div className="flex justify-center border-b border-zinc-900 pb-2">
        <div className="flex bg-zinc-950/80 p-1 rounded-2xl border border-zinc-900 w-full sm:w-80">
          <button
            onClick={() => {
              setMode("buy");
              setErrorMessage(null);
              setSuccessMessage(null);
            }}
            className={`flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-mono font-bold rounded-xl transition-all cursor-pointer ${
              mode === "buy"
                ? "bg-amber-600 text-zinc-950 shadow-lg"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <ShoppingCart className="w-3.5 h-3.5" />
            Mua Hàng
          </button>
          <button
            onClick={() => {
              setMode("sell");
              setErrorMessage(null);
              setSuccessMessage(null);
            }}
            className={`flex-1 flex items-center justify-center gap-1.5 py-2 text-xs font-mono font-bold rounded-xl transition-all cursor-pointer ${
              mode === "sell"
                ? "bg-rose-600/10 dark:bg-rose-600/20 text-rose-600 dark:text-rose-400 border border-rose-500/30 shadow-lg"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <ArrowLeftRight className="w-3.5 h-3.5" />
            Thu Lại (Bán)
          </button>
        </div>
      </div>

      <div className="text-xs text-zinc-400 max-w-xl">
        {mode === "buy"
          ? "* Thương đội của Giả gia mang tới Thanh Mao Sơn rất nhiều Cổ trùng, linh tài và bí dược bổ ích phục vụ các Cổ sư."
          : "* Giả gia đi khắp ngũ vực thu mua cổ trùng hoang dại và nguyên liệu tu hành với giá chiết khấu ưu đãi."}
      </div>

      {/* TABS filter */}
      {mode === "buy" ? (
        <div className="flex gap-2 bg-zinc-950/85 p-1.5 rounded-xl border border-zinc-900">
          {(["Gu", "Material", "Consumable"] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setShopTab(tab)}
              className={`flex-1 text-xs py-2 rounded-lg font-mono font-medium transition-all cursor-pointer ${
                shopTab === tab
                  ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                  : "text-zinc-500 hover:text-zinc-300"
              }`}
            >
              {tab === "Gu" ? "Cổ Trùng" : tab === "Material" ? "Nguyên Liệu" : "Tiêu Hao"}
            </button>
          ))}
        </div>
      ) : (
        <div className="flex gap-2 bg-zinc-950/85 p-1.5 rounded-xl border border-zinc-900">
          {(["Gu", "Material"] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setSellTab(tab)}
              className={`flex-1 text-xs py-2 rounded-lg font-mono font-medium transition-all cursor-pointer ${
                sellTab === tab
                  ? "bg-rose-500/10 text-rose-400 border border-rose-500/20"
                  : "text-zinc-500 hover:text-zinc-300"
              }`}
            >
              {tab === "Gu" ? "Cổ Trùng Bản Thân" : "Nguyên Liệu & Tiêu Hao"}
            </button>
          ))}
        </div>
      )}

      {/* Sell Search Bar */}
      {mode === "sell" && (
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
          <input
            type="text"
            value={sellSearch}
            onChange={(e) => setSellSearch(e.target.value)}
            placeholder="Tìm kiếm vật phẩm..."
            className="w-full bg-zinc-950/60 border border-zinc-900 rounded-xl pl-9 pr-4 py-2 text-xs font-mono text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-rose-500/40 transition-colors"
          />
          {sellSearch && (
            <button
              onClick={() => setSellSearch("")}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-zinc-300 transition-colors"
            >
              <Trash2 className="w-3 h-3" />
            </button>
          )}
        </div>
      )}

      {/* Main List display */}
      {fetchingList ? (
        <div className="py-12 flex flex-col items-center justify-center gap-3">
          <div className="w-8 h-8 border-4 border-amber-500 border-t-transparent rounded-full animate-spin" />
          <span className="text-xs font-mono text-zinc-500">Đang kiểm kê hàng hóa...</span>
        </div>
      ) : mode === "buy" ? (
        filteredBuyItems.length === 0 ? (
          <div className="py-12 flex flex-col items-center justify-center gap-2 text-zinc-650">
            <ShoppingBag className="w-8 h-8 text-zinc-700" />
            <span className="text-xs font-mono">Thương Đội không có bán loại này.</span>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {filteredBuyItems.map((item) => {
              const canAfford = player.primeval_stones >= item.price;
              return (
                <div
                  key={item.id}
                  className="bg-zinc-950/50 border border-zinc-900 rounded-2xl p-4 flex flex-col justify-between hover:border-zinc-800 hover:bg-zinc-950/80 transition-all group"
                >
                  <div className="space-y-2 mb-3">
                    <div className="flex justify-between items-start">
                      <span className="text-[9px] font-mono border border-zinc-800 px-2 py-0.5 rounded text-zinc-500 uppercase tracking-wider">
                        {item.type === "Gu" ? "Cổ Trùng" : item.type === "Material" ? "Nguyên Liệu" : "Tiêu Hao"}
                      </span>
                      <span className="text-xs font-mono font-bold text-amber-400 flex items-center gap-1">
                        <Coins className="w-3.5 h-3.5 text-amber-500" />
                        {item.price}
                      </span>
                    </div>
                    <h4 className="text-base font-bold text-zinc-200 font-serif group-hover:text-amber-400 transition-colors">
                      {item.name}
                    </h4>
                    <details className="mt-1">
                      <summary className="text-[10px] text-zinc-500 hover:text-amber-500 cursor-pointer select-none list-none font-mono outline-none inline-flex items-center gap-1 transition-colors">
                        [+] Xem mô tả
                      </summary>
                      <p className="text-xs text-zinc-405 leading-normal mt-2 border-l-2 border-zinc-800 pl-2">
                        {item.description}
                      </p>
                    </details>
                  </div>
                  <div className="border-t border-zinc-900/60 pt-3 flex items-center justify-between">
                    <span className="text-[10px] text-zinc-600 font-mono">
                      Kho hàng: {item.stock > 0 ? `${item.stock} cái` : "Vô hạn"}
                    </span>
                    <button
                      type="button"
                      disabled={actionLoading}
                      onClick={() => handleOpenBuyModal(item)}
                      className={`py-1.5 px-4 text-xs font-bold rounded-xl flex items-center gap-1 transition-all cursor-pointer ${
                        canAfford
                          ? "bg-amber-600 hover:bg-amber-500 text-zinc-950 active:scale-[0.98]"
                          : "bg-zinc-900 border border-zinc-850 text-zinc-500"
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
        )
      ) : (
        // Sell Mode List
        (sellTab === "Gu" ? filteredSellGus : filteredSellMaterials).length === 0 ? (
          <div className="py-12 flex flex-col items-center justify-center gap-2 text-zinc-600">
            <Trash2 className="w-8 h-8 text-zinc-700" />
            <span className="text-xs font-mono">
              {sellSearch ? "Không tìm thấy vật phẩm phù hợp." : "Bạn không có vật phẩm thích hợp để bán thu lại."}
            </span>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {(sellTab === "Gu" ? filteredSellGus : filteredSellMaterials).map((item) => (
              <div
                key={item.id}
                className="bg-zinc-950/30 border border-zinc-900/80 rounded-2xl p-4 flex flex-col justify-between hover:border-rose-950/50 hover:bg-zinc-950/60 transition-all group"
              >
                <div className="space-y-2 mb-3">
                  <div className="flex justify-between items-start">
                    <span className="text-[9px] font-mono border border-zinc-900 px-2 py-0.5 rounded text-rose-600 dark:text-rose-400 bg-rose-950/10">
                      Sở hữu: {item.owned}x
                    </span>
                    <span className="text-xs font-mono font-bold text-emerald-400 flex items-center gap-1">
                      <Coins className="w-3.5 h-3.5 text-emerald-500" />
                      +{item.price} <span className="text-[10px] text-zinc-500 font-normal">/ cái</span>
                    </span>
                  </div>
                  <h4 className="text-base font-bold text-zinc-200 font-serif group-hover:text-rose-405 transition-colors">
                    {item.name}
                  </h4>
                  <details className="mt-1">
                    <summary className="text-[10px] text-zinc-500 hover:text-rose-400 cursor-pointer select-none list-none font-mono outline-none inline-flex items-center gap-1 transition-colors">
                      [+] Xem mô tả
                    </summary>
                    <p className="text-xs text-zinc-405 leading-normal mt-2 border-l-2 border-zinc-800 pl-2">
                      {item.description}
                    </p>
                  </details>
                </div>
                <div className="border-t border-zinc-900/60 pt-3 flex items-center justify-between">
                  <span className="text-[10px] text-zinc-600 font-mono">
                    Trị giá bán: {item.price * item.owned} Nguyên Thạch
                  </span>
                  <button
                    type="button"
                    disabled={actionLoading}
                    onClick={() => handleOpenSellModal(item)}
                    className="py-1.5 px-4 text-xs font-bold rounded-xl flex items-center gap-1 transition-all cursor-pointer bg-rose-900/20 hover:bg-rose-900/40 border border-rose-900/30 text-rose-300 active:scale-[0.98]"
                  >
                    <ArrowLeftRight className="w-3.5 h-3.5" />
                    <span>Bán Lại</span>
                  </button>
                </div>
              </div>
            ))}
          </div>
        )
      )}

      {/* Buy Quantity Modal */}
      {selectedBuyItem && createPortal(
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 animate-fade-in" onClick={() => setSelectedBuyItem(null)}>
          <div className="bg-zinc-950 border border-zinc-850 rounded-3xl p-6 max-w-sm w-full space-y-4" onClick={(e) => e.stopPropagation()}>
            <div>
              <h4 className="text-lg font-bold text-zinc-100 font-serif">
                Thực Hiện Giao Dịch
              </h4>
              <p className="text-xs text-zinc-400 mt-1">
                Mua {selectedBuyItem.name} từ Giả gia thương đội.
              </p>
            </div>

            <div className="bg-zinc-900/40 border border-zinc-850 rounded-xl p-3 flex justify-between text-xs font-mono">
              <span className="text-zinc-400">Đơn giá:</span>
              <span className="text-amber-400 font-bold flex items-center gap-1">
                <Coins className="w-3.5 h-3.5" /> {selectedBuyItem.price} Nguyên Thạch
              </span>
            </div>

            {selectedBuyItem.type !== "Gu" ? (
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
                {selectedBuyItem.price * buyQuantity} Nguyên Thạch
              </span>
            </div>

            {/* Messages */}
            {errorMessage && (
              <div className="bg-rose-950/20 border border-rose-900/40 rounded-xl p-3 flex items-start gap-2 text-[11px] text-rose-400 leading-normal font-mono animate-shake">
                <AlertCircle className="w-4 h-4 text-rose-500 shrink-0" />
                <span>{errorMessage}</span>
              </div>
            )}

            {successMessage && (
              <div className="bg-emerald-950/20 border border-emerald-900/40 rounded-xl p-3 flex items-start gap-2 text-[11px] text-emerald-400 leading-normal font-mono">
                <Sparkles className="w-4 h-4 text-emerald-500 shrink-0 animate-bounce" />
                <span>{successMessage}</span>
              </div>
            )}

            {/* Action buttons */}
            <div className="grid grid-cols-2 gap-3 pt-2">
              <button
                type="button"
                disabled={actionLoading}
                onClick={() => setSelectedBuyItem(null)}
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
        </div>,
        document.body
      )}

      {/* Sell Quantity Modal */}
      {selectedSellItem && createPortal(
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 animate-fade-in" onClick={() => setSelectedSellItem(null)}>
          <div className="bg-zinc-950 border border-zinc-850 rounded-3xl p-6 max-w-sm w-full space-y-4" onClick={(e) => e.stopPropagation()}>
            <div>
              <h4 className="text-lg font-bold text-zinc-100 font-serif">
                Xác Nhận Bán Lại
              </h4>
              <p className="text-xs text-zinc-400 mt-1">
                Bán lại {selectedSellItem.name} cho Giả gia thương đội.
              </p>
            </div>

            <div className="bg-zinc-900/40 border border-zinc-850 rounded-xl p-3 flex justify-between text-xs font-mono">
              <span className="text-zinc-400">Giá thu hồi:</span>
              <span className="text-emerald-400 font-bold flex items-center gap-1">
                <Coins className="w-3.5 h-3.5" /> +{selectedSellItem.price} Nguyên Thạch
              </span>
            </div>

            {selectedSellItem.type !== "Gu" ? (
              <div className="space-y-1.5">
                <div className="flex justify-between items-center text-[10px] font-mono text-zinc-500 uppercase">
                  <span>Số lượng cần bán:</span>
                  <span>(Tối đa: {selectedSellItem.owned})</span>
                </div>
                <div className="flex items-center justify-between border border-zinc-850 rounded-xl bg-zinc-900/20 overflow-hidden">
                  <button
                    type="button"
                    disabled={sellQuantity <= 1}
                    onClick={() => setSellQuantity((prev) => prev - 1)}
                    className="w-10 h-10 flex items-center justify-center font-bold text-zinc-400 hover:bg-zinc-900 active:bg-zinc-950 disabled:opacity-30"
                  >
                    -
                  </button>
                  <span className="font-bold text-zinc-100 font-mono text-sm">
                    {sellQuantity}
                  </span>
                  <button
                    type="button"
                    disabled={sellQuantity >= selectedSellItem.owned}
                    onClick={() => setSellQuantity((prev) => prev + 1)}
                    className="w-10 h-10 flex items-center justify-center font-bold text-zinc-400 hover:bg-zinc-900 active:bg-zinc-950 disabled:opacity-30"
                  >
                    +
                  </button>
                </div>
              </div>
            ) : (
              <div className="text-xs text-zinc-500 italic py-1">
                * Cổ trùng là sinh linh độc bản, chỉ có thể thanh lý từng con một.
              </div>
            )}

            {/* Total Gain Display */}
            <div className="flex justify-between items-center pt-2 font-mono border-t border-dashed border-zinc-900 text-xs">
              <span className="text-zinc-400">Nguyên Thạch nhận về:</span>
              <span className="text-emerald-400 font-bold flex items-center gap-1 text-sm">
                <Coins className="w-4 h-4 text-emerald-500" />
                +{selectedSellItem.price * sellQuantity} Nguyên Thạch
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
                <Sparkles className="w-4 h-4 text-emerald-500 shrink-0" />
                <span>{successMessage}</span>
              </div>
            )}

            {/* Action buttons */}
            <div className="grid grid-cols-2 gap-3 pt-2">
              <button
                type="button"
                disabled={actionLoading}
                onClick={() => setSelectedSellItem(null)}
                className="py-2.5 rounded-xl border border-zinc-800 text-zinc-400 hover:bg-zinc-900 font-semibold text-xs active:scale-[0.98] transition-all cursor-pointer"
              >
                Hủy Bỏ
              </button>
              <button
                type="button"
                disabled={actionLoading || !!successMessage}
                onClick={handleConfirmSell}
                className="py-2.5 rounded-xl bg-rose-600/35 hover:bg-rose-600/50 border border-rose-500/40 text-rose-100 font-bold text-xs active:scale-[0.98] transition-all cursor-pointer"
              >
                Xác Nhận Bán
              </button>
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
    </>
  );
};
