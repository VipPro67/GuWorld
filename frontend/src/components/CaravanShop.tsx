"use client";

import React, { useState, useEffect, useMemo, useCallback } from "react";
import { createPortal } from "react-dom";
import { GuInstance, Player, GuSpec, ItemSpec, useGameStore, formatRecipeName, guSpecFor } from "@/store/useGameStore";
import { ShoppingBag, Coins, Sparkles, AlertCircle, ShoppingCart, ArrowLeftRight, Trash2, Search, Check } from "lucide-react";
import { translatePath } from "@/lib/translate";
import { GamePanel, GameButton, GameTabs, GameCard, GameInput, GameSelect } from "@/components/ui";
import { EffectEnvelopeView } from "@/components/EffectBlockView";
// UI migrated to shared components

interface ShopItem {
  id: string;
  name: string;
  description: string;
  type: string;
  price: number;
  rank?: number;
  currency?: string; // "primeval" (rank < 6) | "immortal" (rank >= 6)
  stock: number;
}

type StoneTier = "primeval" | "immortal";

const stoneTier = (rank: number): StoneTier => (rank >= 6 ? "immortal" : "primeval");

const currencyLabel = (tier: StoneTier): string =>
  tier === "immortal" ? "Tiên Nguyên Thạch" : "Nguyên Thạch";

interface CaravanShopProps {
  player: Player;
  actionLoading: boolean;
  onBuy: (itemId: string, quantity: number, rank?: number) => Promise<boolean>;
  onSell?: (itemId: string, quantity: number, itemType: "Gu" | "Material" | "Recipe", rank?: number) => Promise<boolean>;
  onSellBulk: (items: { item_id: string; quantity: number; item_type: "Gu" | "Material" | "Recipe"; rank?: number }[]) => Promise<boolean>;
}

export const CaravanShop: React.FC<CaravanShopProps> = ({
  player,
  actionLoading,
  onBuy,
  onSellBulk,
}) => {
  const { guDictionary, itemDictionary, addToast } = useGameStore();
  const [shopItems, setShopItems] = useState<ShopItem[]>([]);
  const [fetchingList, setFetchingList] = useState(false);
  const [mode, setMode] = useState<"buy" | "sell">("buy");
  const [shopTab, setShopTab] = useState<"Gu" | "Material" | "Consumable">("Gu");
  const [sellTab, setSellTab] = useState<"All" | "Gu" | "Material" | "Recipe">("All");
  const [sellSearch, setSellSearch] = useState("");
  const [sellRankFilter, setSellRankFilter] = useState<string>("All");
  const [sellPathFilter, setSellPathFilter] = useState<string>("All");

  // Selection states for Modals
  const [selectedBuyItem, setSelectedBuyItem] = useState<ShopItem | null>(null);
  const [buyQuantity, setBuyQuantity] = useState(1);



  // Bulk selection states
  const [selectedSellItems, setSelectedSellItems] = useState<Record<string, {
    id: string;
    name: string;
    type: "Gu" | "Material" | "Recipe";
    owned: number;
    price: number;
    quantityToSell: number;
    rank: number;
  }>>({});
  const [showReviewDrawer, setShowReviewDrawer] = useState(false);

  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

  const loadShopItems = useCallback(async () => {
    setFetchingList(true);
    try {
      const response = await fetch(`${BACKEND_URL}/api/shop/list`);
      if (response.ok) setShopItems(await response.json());
    } catch (err) {
      console.error("Failed to fetch shop list:", err);
    } finally {
      setFetchingList(false);
    }
  }, [BACKEND_URL]);

  useEffect(() => {
    void loadShopItems();
  }, [loadShopItems]);

  // Reset filters when changing tab or mode
  useEffect(() => {
    setSellRankFilter("All");
    setSellPathFilter("All");
  }, [sellTab, mode]);

  const sellBackRatio = 0.10;

  const getBalance = (tier: StoneTier): number =>
    tier === "immortal" ? (player.immortal_stones || 0) : player.primeval_stones;

  const buyTier: StoneTier = selectedBuyItem?.currency === "immortal" ? "immortal" : "primeval";

  const guRankBasePrice = [0, 200, 500, 1200, 2500, 5000, 200, 500, 1200, 2500];
  const itemRankBasePrice = [0, 1, 3, 6, 13, 25, 10, 25, 60, 150];

  const getGuBasePrice = (guId: string, rank?: number): number => {
    const spec = guSpecFor(guId, rank) || guDictionary[guId];
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

  // SellPrice = basePrice * 0.10, rounded to the 0.001 currency unit.
  const getSellPrice = (basePrice: number): number => Math.max(0.001, Math.round(basePrice * sellBackRatio * 1000) / 1000);

  // Helper to calculate exact sell price matching pricing.go
  const getItemSellPrice = (itemId: string, type: "Gu" | "Material", rank?: number): number => {
    const basePrice = type === "Gu" ? getGuBasePrice(itemId, rank) : getItemBasePrice(itemId);
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
    .filter(([, qty]) => qty > 0)
    .map(([itemId, qty]) => {
      const spec = itemDictionary[itemId];
      const rank = spec?.rank || 1;
      return {
        id: itemId,
        name: spec?.name || itemId,
        description: spec?.description || "Vật phẩm tu sĩ.",
        type: "Material" as const,
        owned: qty,
        price: getItemSellPrice(itemId, "Material"),
        rank,
        currency: stoneTier(rank),
        paths: spec?.path ? [spec.path] : [],
      };
    });

  const sellableGus = useMemo(() => {
    return Object.entries(
      (player.gu_inventory || [])
        .filter((gu) => !gu.is_vital && !gu.equipped && gu.status !== "DEAD")
        .reduce((acc, gu) => {
          const key = `${gu.gu_id}@${gu.rank}`;
          acc[key] = (acc[key] || 0) + 1;
          return acc;
        }, {} as Record<string, number>)
    ).map(([key, count]) => {
      const guId = key.split("@")[0];
      const rank = Number(key.split("@")[1]) || 1;
      const spec = guSpecFor(guId, rank) || guDictionary[guId];
      return {
        id: guId,
        name: spec?.name || guId,
        description: spec?.description || "Cổ trùng bản thân nuôi dưỡng.",
        type: "Gu" as const,
        owned: count,
        price: getItemSellPrice(guId, "Gu", rank),
        rank,
        currency: stoneTier(rank),
        paths: spec?.path || [],
      };
    });
  }, [player.gu_inventory, guDictionary]);

  const sellableRecipes = (player.unlocked_recipes || []).map((recipe) => {
    const targetGuID = recipe.recipe_id.replace("recipe_", "").replace(/_\d+_\d+$/, "").toUpperCase();
    const targetGuRank = Number(recipe.recipe_id.match(/_(\d+)_\d+$/)?.[1]);
    return {
      id: recipe.recipe_id,
      name: formatRecipeName(recipe.recipe_id),
      description: `Bí phương luyện chế Cổ trùng. Độ hoàn thiện: ${recipe.completion_rate}%`,
      type: "Recipe" as const,
      owned: 1,
      price: getSellPrice(getGuBasePrice(targetGuID, targetGuRank)),
      rank: targetGuRank,
      currency: stoneTier(targetGuRank),
      paths: [] as string[],
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

    if (!buyQuantity || buyQuantity <= 0) {
      setErrorMessage("Số lượng mua phải lớn hơn 0.");
      return;
    }

    const totalCost = selectedBuyItem.price * buyQuantity;
    const balance = buyTier === "immortal" ? (player.immortal_stones || 0) : player.primeval_stones;
    if (balance < totalCost) {
      setErrorMessage(`Không đủ ${currencyLabel(buyTier)} để thực hiện giao dịch này.`);
      return;
    }

    const success = await onBuy(selectedBuyItem.id, buyQuantity, selectedBuyItem.rank ?? 0);
    if (success) {
      await loadShopItems();
      setSuccessMessage("Giao dịch thành công! Đã thêm vào túi đồ.");
      setTimeout(() => {
        setSelectedBuyItem(null);
      }, 1000);
    }
  };

  const normalizeText = (text: string) => text.toLowerCase().trim();

  const matchSellItem = (item: { id: string; name: string; description: string }, query: string): boolean => {
    if (!query) return true;
    const q = normalizeText(query);
    return normalizeText(item.name).includes(q) || normalizeText(item.id).includes(q);
  };

  const sellableAll = [...sellableGus, ...sellableMaterials, ...sellableRecipes];

  // Base list for current active sell tab (before filters)
  const currentTabBaseList = sellTab === "All" ? sellableAll : (sellTab === "Gu" ? sellableGus : (sellTab === "Material" ? sellableMaterials : sellableRecipes));

  // Dynamic filter lists
  const availableRanks = Array.from(
    new Set(currentTabBaseList.map((item) => item.rank))
  ).filter(Boolean).sort((a: number, b: number) => a - b);

  const availablePaths = Array.from(
    new Set(currentTabBaseList.flatMap((item) => item.paths || []))
  ).filter(Boolean).sort();

  const filteredSellGus = sellableGus.filter((item) => {
    const matchesSearch = matchSellItem(item, sellSearch);
    const matchesRank = sellRankFilter === "All" || item.rank.toString() === sellRankFilter;
    const matchesPath = sellPathFilter === "All" || item.paths.includes(sellPathFilter);
    return matchesSearch && matchesRank && matchesPath;
  });

  const filteredSellMaterials = sellableMaterials.filter((item) => {
    const matchesSearch = matchSellItem(item, sellSearch);
    const matchesRank = sellRankFilter === "All" || item.rank.toString() === sellRankFilter;
    const matchesPath = sellPathFilter === "All" || item.paths.includes(sellPathFilter);
    return matchesSearch && matchesRank && matchesPath;
  });

  const filteredSellRecipes = sellableRecipes.filter((item) => {
    const matchesSearch = matchSellItem(item, sellSearch);
    const matchesRank = sellRankFilter === "All" || item.rank.toString() === sellRankFilter;
    const matchesPath = sellPathFilter === "All" || item.paths.includes(sellPathFilter);
    return matchesSearch && matchesRank && matchesPath;
  });

  const filteredSellAll = [...filteredSellGus, ...filteredSellMaterials, ...filteredSellRecipes];

  const activeSellList = sellTab === "All" ? filteredSellAll : (sellTab === "Gu" ? filteredSellGus : (sellTab === "Material" ? filteredSellMaterials : filteredSellRecipes));

  const toggleSelectItem = (item: {
    id: string;
    name: string;
    type: "Gu" | "Material" | "Recipe";
    owned: number;
    price: number;
    rank: number;
  }) => {
    setSelectedSellItems((prev) => {
      const key = `${item.type}-${item.id}-${item.rank}`;
      const next = { ...prev };
      if (next[key]) {
        delete next[key];
      } else {
        if (Object.keys(prev).length >= 50) {
          setErrorMessage("Mỗi lần thanh lý tối đa 50 loại vật phẩm!");
          setTimeout(() => setErrorMessage(null), 4000);
          return prev;
        }
        next[key] = {
          ...item,
          quantityToSell: item.owned,
        };
      }
      return next;
    });
  };

  const updateQuantity = (key: string, qty: number) => {
    setSelectedSellItems((prev) => {
      if (!prev[key]) return prev;
      const item = prev[key];
      let newQty = qty;
      if (isNaN(qty) || qty <= 0) {
        newQty = 1;
      } else if (qty > item.owned) {
        newQty = item.owned;
      }
      return {
        ...prev,
        [key]: {
          ...item,
          quantityToSell: newQty,
        },
      };
    });
  };

  const handleQuickSelectByRank = (rankNum: number) => {
    const list = sellTab === "All" ? sellableAll : (sellTab === "Gu" ? sellableGus : (sellTab === "Material" ? sellableMaterials : sellableRecipes));
    const itemsToSelect = list.filter(item => item.rank === rankNum);
    setSelectedSellItems((prev) => {
      const next = { ...prev };
      let addedCount = Object.keys(prev).length;
      for (const item of itemsToSelect) {
        const key = `${item.type}-${item.id}-${item.rank}`;
        if (!next[key]) {
          if (addedCount >= 50) {
            setErrorMessage("Mỗi lần thanh lý tối đa 50 loại vật phẩm! Đã tự động giới hạn ở 50.");
            setTimeout(() => setErrorMessage(null), 4500);
            break;
          }
          next[key] = {
            ...item,
            quantityToSell: item.owned,
          };
          addedCount++;
        } else {
          next[key] = {
            ...next[key],
            quantityToSell: item.owned,
          };
        }
      }
      return next;
    });
  };

  const handleSelectAllCurrent = () => {
    const list = sellTab === "All" ? sellableAll : (sellTab === "Gu" ? sellableGus : (sellTab === "Material" ? sellableMaterials : sellableRecipes));
    const filtered = list.filter((item) => {
        const matchesSearch = matchSellItem(item, sellSearch);
        const matchesRank = sellRankFilter === "All" || item.rank.toString() === sellRankFilter;
        const matchesPath = sellPathFilter === "All" || item.paths.includes(sellPathFilter);
        return matchesSearch && matchesRank && matchesPath;
    });
    setSelectedSellItems((prev) => {
      const next = { ...prev };
      let addedCount = Object.keys(prev).length;
      for (const item of filtered) {
        const key = `${item.type}-${item.id}-${item.rank}`;
        if (!next[key]) {
          if (addedCount >= 50) {
            setErrorMessage("Mỗi lần thanh lý tối đa 50 loại vật phẩm! Đã tự động giới hạn ở 50.");
            setTimeout(() => setErrorMessage(null), 4500);
            break;
          }
          next[key] = {
            ...item,
            quantityToSell: item.owned,
          };
          addedCount++;
        } else {
          next[key] = {
            ...next[key],
            quantityToSell: item.owned,
          };
        }
      }
      return next;
    });
  };

  const bulkGainByCurrency = Object.values(selectedSellItems).reduce<{ primeval: number; immortal: number }>(
    (acc, item) => {
      const tier = stoneTier(item.rank);
      acc[tier] += item.price * item.quantityToSell;
      return acc;
    },
    { primeval: 0, immortal: 0 }
  );

  const bulkGainSummary = bulkGainByCurrency.immortal > 0
    ? (bulkGainByCurrency.primeval > 0
        ? `+${bulkGainByCurrency.primeval.toLocaleString()} Nguyên Thạch và +${bulkGainByCurrency.immortal.toLocaleString()} Tiên Nguyên Thạch`
        : `+${bulkGainByCurrency.immortal.toLocaleString()} Tiên Nguyên Thạch`)
    : `+${bulkGainByCurrency.primeval.toLocaleString()} Nguyên Thạch`;

  const handleConfirmBulkSell = async () => {
    const items = Object.values(selectedSellItems).map(item => ({
      item_id: item.id,
      quantity: item.quantityToSell,
      item_type: item.type,
      rank: item.rank,
    }));
    if (items.length === 0) return;

    setErrorMessage(null);
    setSuccessMessage(null);

    const success = await onSellBulk(items);
    if (success) {
      setSuccessMessage(`Đã thanh lý thành công ${items.length} loại vật phẩm!`);
      if (addToast) {
        addToast(`Đã thanh lý thành công ${items.length} loại vật phẩm! Nhận về ${bulkGainSummary}.`, "success");
      }
      setSelectedSellItems({});
      setShowReviewDrawer(false);
      setTimeout(() => setSuccessMessage(null), 4000);
    } else {
      setErrorMessage("Thanh lý hàng loạt thất bại. Vui lòng kiểm tra lại túi đồ.");
      if (addToast) {
        addToast("Thanh lý hàng loạt thất bại. Vui lòng kiểm tra lại túi đồ.", "error");
      }
      setTimeout(() => setErrorMessage(null), 4000);
    }
  };



  return (
    <GamePanel variant="gold" className="rounded-3xl p-3 sm:p-6 w-full mx-auto space-y-6 animate-fade-in">
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
        <GameTabs
          items={[
            { id: "buy", label: "Mua Hàng", icon: <ShoppingCart className="w-3.5 h-3.5" />, disabled: false },
            { id: "sell", label: "Thu Lại (Bán)", icon: <ArrowLeftRight className="w-3.5 h-3.5" />, disabled: false },
          ]}
          activeId={mode}
          onChange={(id) => {
            setMode(id as ApiResponseData);
            setErrorMessage(null);
            setSuccessMessage(null);
          }}
          variant="pills"
          size="sm"
          scrollable={true}
          fullWidth={false}
        >
        </GameTabs>
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
          {(["All", "Gu", "Material", "Recipe"] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setSellTab(tab)}
              className={`flex-1 text-xs py-2 rounded-lg font-mono font-medium transition-all cursor-pointer ${
                sellTab === tab
                  ? "bg-rose-500/10 text-rose-400 border border-rose-500/20"
                  : "text-zinc-500 hover:text-zinc-300"
              }`}
            >
              {tab === "All" ? "Tất Cả" : tab === "Gu" ? "Cổ Trùng" : tab === "Material" ? "Vật Phẩm" : "Bí Phương"}
            </button>
          ))}
        </div>
      )}

      {/* Sell Search Bar & Filters */}
      {mode === "sell" && (
        <div className="flex flex-col sm:flex-row gap-2">
          {/* Search Input */}
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-500" />
            <input
              type="text"
              value={sellSearch}
              onChange={(e) => setSellSearch(e.target.value)}
              placeholder="Tìm kiếm vật phẩm..."
              className="w-full bg-zinc-950/60 border border-zinc-900 rounded-xl pl-9 pr-4 py-2.5 text-xs font-mono text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-rose-500/40 transition-colors"
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

          {/* Filters Selectors */}
          <div className="flex gap-2 w-full sm:w-auto">
            {/* Rank Selector */}
            <select
              value={sellRankFilter}
              onChange={(e) => setSellRankFilter(e.target.value)}
              className="flex-1 sm:flex-none bg-zinc-950/60 border border-zinc-900 rounded-xl px-3 py-2.5 text-xs font-mono text-zinc-300 focus:outline-none focus:border-rose-500/40 cursor-pointer min-w-[100px] outline-none"
            >
              <option value="All" className="bg-zinc-950 text-zinc-300">Tất cả Hạng</option>
              {availableRanks.map((rank) => (
                <option key={rank} value={rank.toString()} className="bg-zinc-950 text-zinc-300">
                  Hạng {rank}
                </option>
              ))}
            </select>

            {/* Path Selector */}
            <select
              value={sellPathFilter}
              onChange={(e) => setSellPathFilter(e.target.value)}
              className="flex-1 sm:flex-none bg-zinc-950/60 border border-zinc-900 rounded-xl px-3 py-2.5 text-xs font-mono text-zinc-300 focus:outline-none focus:border-rose-500/40 cursor-pointer min-w-[120px] outline-none"
            >
              <option value="All" className="bg-zinc-950 text-zinc-300">Tất cả Phái</option>
              {availablePaths.map((path) => (
                <option key={path} value={path} className="bg-zinc-950 text-zinc-300">
                  {translatePath(path)}
                </option>
              ))}
            </select>
          </div>
        </div>
      )}
      {/* Bulk action bar (Inline below search bar) */}
      {mode === "sell" && Object.keys(selectedSellItems).length > 0 && (
        <div className="bg-zinc-950/80 border border-rose-500/30 rounded-2xl p-4 shadow-xl flex flex-col gap-3 animate-in fade-in duration-300">
          <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3 border-b border-zinc-900 pb-2">
            <div className="flex items-center gap-2">
              <div className="bg-rose-950/20 border border-rose-500/30 p-1.5 rounded-lg">
                <ArrowLeftRight className="w-4 h-4 text-rose-450" />
              </div>
              <div>
                <h4 className="text-xs font-bold text-zinc-200 font-serif">Thanh Lý Hàng Loạt</h4>
                <p className="text-[10px] text-zinc-500">
                  Đã chọn <span className={Object.keys(selectedSellItems).length >= 50 ? "text-rose-400 font-bold" : "text-zinc-300 font-mono"}>{Object.keys(selectedSellItems).length}</span>/50 loại
                </p>
              </div>
            </div>

            {/* Quick select buttons */}
            <div className="flex items-center gap-1.5 flex-wrap">
              <span className="text-[9px] font-mono text-zinc-500">Chọn nhanh:</span>
              <button
                type="button"
                onClick={() => handleQuickSelectByRank(1)}
                className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 border border-zinc-800 transition-colors"
              >
                Rank 1
              </button>
              <button
                type="button"
                onClick={() => handleQuickSelectByRank(2)}
                className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 border border-zinc-800 transition-colors"
              >
                Rank 2
              </button>
              <button
                type="button"
                onClick={handleSelectAllCurrent}
                className="px-2 py-0.5 text-[9px] font-mono rounded bg-rose-950/10 hover:bg-rose-950/20 text-rose-450 border border-rose-950/20 transition-colors"
              >
                Tất cả tab này
              </button>
              <button
                type="button"
                onClick={() => setSelectedSellItems({})}
                className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-zinc-300 border border-zinc-800 transition-colors"
              >
                Bỏ chọn hết
              </button>
            </div>
          </div>

          {/* Notification messages inside bulk sell bar */}
          {errorMessage && (
            <div className="bg-rose-950/20 border border-rose-900/40 rounded-xl p-2.5 flex items-start gap-2 text-[11px] text-rose-400 leading-normal font-mono animate-shake">
              <AlertCircle className="w-4 h-4 text-rose-500 shrink-0" />
              <span>{errorMessage}</span>
            </div>
          )}

          {successMessage && (
            <div className="bg-emerald-950/20 border border-emerald-900/40 rounded-xl p-2.5 flex items-start gap-2 text-[11px] text-emerald-400 leading-normal font-mono">
              <Sparkles className="w-4 h-4 text-emerald-500 shrink-0 animate-bounce" />
              <span>{successMessage}</span>
            </div>
          )}

          {/* Collapsible item list for review */}
          {showReviewDrawer && (
            <div className="max-h-32 overflow-y-auto border-b border-zinc-900/60 pb-2 space-y-1.5 pr-2 scrollbar-thin">
              {Object.entries(selectedSellItems).map(([itemKey, selectedItem]) => (
                <div key={itemKey} className="flex justify-between items-center bg-zinc-900/30 border border-zinc-900/50 rounded-lg p-2 text-[11px] font-mono">
                  <div className="flex items-center gap-1.5">
                    <span className={`text-[8px] px-1.5 py-0.2 rounded ${selectedItem.type === "Gu" ? "bg-violet-950/20 text-violet-400 border border-violet-900/30" : "bg-cyan-950/20 text-cyan-400 border border-cyan-900/30"}`}>
                      {selectedItem.type === "Gu" ? "Cổ" : "Linh tài"}
                    </span>
                    <span className="text-zinc-300 font-bold">{selectedItem.name}</span>
                    <span className="text-zinc-500">x{selectedItem.quantityToSell}</span>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className="text-emerald-400 font-bold flex items-center gap-0.5">
                      +{selectedItem.price * selectedItem.quantityToSell}
                      <span className="text-[9px] text-zinc-500 font-normal">
                        {stoneTier(selectedItem.rank) === "immortal" ? "TNT" : "NT"}
                      </span>
                    </span>
                    <button
                      type="button"
                      onClick={() => setSelectedSellItems(prev => {
                        const next = { ...prev };
                        delete next[itemKey];
                        return next;
                      })}
                      className="text-zinc-600 hover:text-rose-455 transition-colors cursor-pointer"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Bottom row: Total gain & actions */}
          <div className="flex justify-between items-center gap-4">
            <button
              type="button"
              onClick={() => setShowReviewDrawer(!showReviewDrawer)}
              className="flex items-center gap-1 text-[10px] text-zinc-450 hover:text-zinc-200 transition-colors font-mono cursor-pointer"
            >
              {showReviewDrawer ? "Thu gọn chi tiết" : "Xem chi tiết đã chọn"}
              <span className="text-[9px]">{showReviewDrawer ? "▲" : "▼"}</span>
            </button>

            <div className="flex items-center gap-3">
              <div className="text-right">
                <span className="text-[10px] text-zinc-500 font-mono block">Tổng thu hồi</span>
                <span className="text-emerald-450 font-bold flex flex-col items-end gap-1 text-sm font-mono">
                  {bulkGainByCurrency.primeval > 0 && (
                    <span className="flex items-center gap-1">
                      <Coins className="w-4 h-4 text-emerald-500" />
                      +{bulkGainByCurrency.primeval.toLocaleString()} NT
                    </span>
                  )}
                  {bulkGainByCurrency.immortal > 0 && (
                    <span className="flex items-center gap-1">
                      <Sparkles className="w-4 h-4 text-violet-400" />
                      +{bulkGainByCurrency.immortal.toLocaleString()} TNT
                    </span>
                  )}
                </span>
              </div>

              <button
                type="button"
                onClick={handleConfirmBulkSell}
                disabled={actionLoading || Object.keys(selectedSellItems).length === 0}
                className="px-5 py-2 text-xs font-mono font-bold rounded-xl bg-rose-600 hover:bg-rose-500 text-zinc-950 cursor-pointer shadow-lg active:scale-[0.98] transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
              >
                {actionLoading ? "Đang xử lý..." : "Xác Nhận Bán"}
              </button>
            </div>
          </div>
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
              const guSpec = item.type === "Gu" ? guSpecFor(item.id, item.rank) : undefined;
              const itemTier: StoneTier = item.currency === "immortal" ? "immortal" : "primeval";
              const canAfford = itemTier === "immortal"
                ? (player.immortal_stones || 0) >= item.price
                : player.primeval_stones >= item.price;
              const ownedQty = item.type === "Gu"
                ? (player.gu_inventory?.filter((g) => g.gu_id === item.id && g.status !== "DEAD").length || 0)
                : (player.materials_inventory?.[item.id] || 0);

              return (
                <div
                  key={`${item.id}-${item.rank || 0}`}
                  className="bg-zinc-950/50 border border-zinc-900 rounded-2xl p-4 flex flex-col justify-between hover:border-zinc-800 hover:bg-zinc-950/80 transition-all group"
                >
                  <div className="space-y-2 mb-3">
                    <div className="flex justify-between items-start">
                      <span className="text-[9px] font-mono border border-zinc-800 px-2 py-0.5 rounded text-zinc-500 uppercase tracking-wider">
                        {item.type === "Gu" ? "Cổ Trùng" : item.type === "Material" ? "Nguyên Liệu" : "Tiêu Hao"}
                      </span>
                      {item.rank ? (
                        <span className="text-[9px] font-mono border border-zinc-800 px-2 py-0.5 rounded text-violet-400">
                          Hạng {item.rank}
                        </span>
                      ) : null}
                      <span className={`text-xs font-mono font-bold flex items-center gap-1 ${itemTier === "immortal" ? "text-violet-400" : "text-amber-400"}`}>
                        {itemTier === "immortal"
                          ? <Sparkles className="w-3.5 h-3.5 text-violet-400" />
                          : <Coins className="w-3.5 h-3.5 text-amber-500" />}
                        {item.price}
                        <span className="text-[9px] text-zinc-500 font-normal">{itemTier === "immortal" ? "Tiên Nguyên Thạch" : ""}</span>
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
                      {item.type === "Gu" && (
                        <div className="mt-3">
                          <h5 className="mb-1 text-[10px] font-mono font-bold uppercase tracking-wide text-amber-500">Tác dụng</h5>
                          {guSpec?.effects?.blocks?.length ? (
                            <EffectEnvelopeView effects={guSpec.effects} />
                          ) : (
                            <p className="text-[10px] text-zinc-500">Cổ trùng này không có hiệu ứng được ghi nhận.</p>
                          )}
                        </div>
                      )}
                    </details>
                  </div>
                  <div className="border-t border-zinc-900/60 pt-3 flex items-center justify-between">
                    <div className="flex flex-col gap-0.5">
                      <span className="text-[10px] text-zinc-600 font-mono">
                        Kho hàng: {item.stock > 0 ? `${item.stock} cái` : "Vô hạn"}
                      </span>
                      <span className="text-[10px] text-zinc-500 font-mono">
                        Bạn có: <span className="text-emerald-500 font-bold">x{ownedQty}</span>
                      </span>
                    </div>
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
        activeSellList.length === 0 ? (
          <div className="py-12 flex flex-col items-center justify-center gap-2 text-zinc-600">
            <Trash2 className="w-8 h-8 text-zinc-700" />
            <span className="text-xs font-mono">
              {sellSearch ? "Không tìm thấy vật phẩm phù hợp." : "Bạn không có vật phẩm thích hợp để bán thu lại."}
            </span>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pb-6">
            {activeSellList.map((item) => {
              const key = `${item.type}-${item.id}-${item.rank}`;
              const isSelected = !!selectedSellItems[key];
              const selectedItem = selectedSellItems[key];
              return (
                <div
                  key={key}
                  onClick={() => toggleSelectItem(item)}
                  className={`bg-zinc-950/30 border rounded-2xl p-4 flex flex-col justify-between hover:bg-zinc-950/60 transition-all group relative cursor-pointer ${
                    isSelected
                      ? "border-rose-500/50 bg-rose-500/5 shadow-md shadow-rose-950/20"
                      : "border-zinc-900/80 hover:border-rose-950/50"
                  }`}
                >
                  <div className="space-y-2 mb-3">
                    <div className="flex justify-between items-start pr-8">
                      <span className="text-[9px] font-mono border border-zinc-900 px-2 py-0.5 rounded text-rose-600 dark:text-rose-400 bg-rose-950/10">
                        Sở hữu: {item.owned}x
                      </span>
                      <span className="text-xs font-mono font-bold text-emerald-400 flex items-center gap-1">
                        {item.currency === "immortal"
                          ? <Sparkles className="w-3.5 h-3.5 text-violet-400" />
                          : <Coins className="w-3.5 h-3.5 text-emerald-500" />}
                        +{item.price} <span className="text-[10px] text-zinc-500 font-normal">/ cái {item.currency === "immortal" ? "TNT" : "NT"}</span>
                      </span>
                    </div>

                    {/* Selection Checkbox */}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        toggleSelectItem(item);
                      }}
                      className={`absolute top-4 right-4 w-5 h-5 rounded-lg border flex items-center justify-center transition-all ${
                        isSelected
                          ? "bg-rose-600 border-rose-500 text-zinc-950"
                          : "bg-zinc-900 border-zinc-800 hover:border-rose-500/30 text-transparent"
                      }`}
                    >
                      <Check className="w-3.5 h-3.5 text-zinc-950 font-bold" />
                    </button>

                    <h4 className="text-base font-bold text-zinc-200 font-serif group-hover:text-rose-400 transition-colors">
                      {item.name}
                    </h4>
                    <details className="mt-1" onClick={(e) => e.stopPropagation()}>
                      <summary className="text-[10px] text-zinc-500 hover:text-rose-405 cursor-pointer select-none list-none font-mono outline-none inline-flex items-center gap-1 transition-colors">
                        [+] Xem mô tả
                      </summary>
                      <p className="text-xs text-zinc-405 leading-normal mt-2 border-l-2 border-zinc-850 pl-2">
                        {item.description}
                      </p>
                    </details>
                  </div>
                  <div className="border-t border-zinc-900/60 pt-3 flex items-center justify-between">
                    <span className="text-[10px] text-zinc-650 font-mono">
                      {isSelected
                        ? `Trị giá chọn: ${item.price * selectedItem.quantityToSell} ${currencyLabel(stoneTier(item.rank))}`
                        : `Tổng trị giá: ${item.price * item.owned} ${currencyLabel(stoneTier(item.rank))}`}
                    </span>

                    {/* Quantity controls or select button */}
                    {isSelected ? (
                      <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                        <button
                          type="button"
                          onClick={() => updateQuantity(key, selectedItem.quantityToSell - 1)}
                          className="w-7 h-7 flex items-center justify-center bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 rounded-lg text-zinc-400 font-bold transition-all text-xs"
                        >
                          -
                        </button>
                        <input
                          type="number"
                          min={1}
                          max={item.owned}
                          value={selectedItem.quantityToSell}
                          onChange={(e) => updateQuantity(key, parseInt(e.target.value, 10))}
                          className="w-10 text-center bg-zinc-900/80 border border-zinc-800 rounded-lg py-1 text-xs font-mono font-bold text-zinc-200 focus:outline-none focus:border-rose-500/40"
                        />
                        <button
                          type="button"
                          onClick={() => updateQuantity(key, selectedItem.quantityToSell + 1)}
                          className="w-7 h-7 flex items-center justify-center bg-zinc-900 hover:bg-zinc-850 border border-zinc-800 rounded-lg text-zinc-400 font-bold transition-all text-xs"
                        >
                          +
                        </button>
                        <button
                          type="button"
                          onClick={() => updateQuantity(key, item.owned)}
                          className="px-2 py-1 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/20 rounded-lg text-[10px] font-mono text-rose-400 transition-all font-semibold"
                        >
                          Max
                        </button>
                      </div>
                    ) : (
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleSelectItem(item);
                        }}
                        className="py-1.5 px-4 text-xs font-bold rounded-xl flex items-center gap-1 transition-all cursor-pointer bg-rose-900/20 hover:bg-rose-900/40 border border-rose-900/30 text-rose-300 active:scale-[0.98]"
                      >
                        <ArrowLeftRight className="w-3.5 h-3.5" />
                        <span>Chọn bán</span>
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
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

            <div className="bg-zinc-900/40 border border-zinc-850 rounded-xl p-3 space-y-1.5 text-xs font-mono">
              <div className="flex justify-between">
                <span className="text-zinc-400">Đơn giá:</span>
                <span className={`font-bold flex items-center gap-1 ${buyTier === "immortal" ? "text-violet-400" : "text-amber-400"}`}>
                  {buyTier === "immortal"
                    ? <Sparkles className="w-3.5 h-3.5 text-violet-400" />
                    : <Coins className="w-3.5 h-3.5 text-amber-500" />}
                  {selectedBuyItem.price} {currencyLabel(buyTier)}
                </span>
              </div>
              <div className="flex justify-between border-t border-zinc-800/40 pt-1.5">
                <span className="text-zinc-400">Bạn có:</span>
                <span className="text-emerald-500 font-bold">
                  {selectedBuyItem.type === "Gu"
                    ? `x${player.gu_inventory?.filter((g) => g.gu_id === selectedBuyItem.id && g.status !== "DEAD").length || 0}`
                    : `x${player.materials_inventory?.[selectedBuyItem.id] || 0}`
                  }
                </span>
              </div>
            </div>

            <div className="space-y-2">
                <div className="flex justify-between items-center text-[10px] font-mono text-zinc-500 uppercase">
                  <span>Số lượng cần mua:</span>
                  <span>
                    {selectedBuyItem.price > 0 && `(Có thể mua: ${Math.floor(getBalance(buyTier) / selectedBuyItem.price).toLocaleString()})`}
                  </span>
                </div>
                <div className="flex items-center justify-between border border-zinc-850 rounded-xl bg-zinc-900/20 overflow-hidden px-1">
                  <button
                    type="button"
                    disabled={buyQuantity <= 1}
                    onClick={() => setBuyQuantity((prev) => Math.max(1, (prev || 0) - 1))}
                    className="w-10 h-10 flex items-center justify-center font-bold text-zinc-400 hover:bg-zinc-900 active:bg-zinc-950 disabled:opacity-30 rounded-lg cursor-pointer transition-colors"
                  >
                    -
                  </button>
                  <input
                    type="number"
                    min={1}
                    max={selectedBuyItem.stock > 0 ? selectedBuyItem.stock : undefined}
                    value={buyQuantity || ""}
                    onChange={(e) => {
                      const val = parseInt(e.target.value, 10);
                      if (isNaN(val) || val <= 0) {
                        setBuyQuantity(0);
                      } else if (selectedBuyItem.stock > 0 && val > selectedBuyItem.stock) {
                        setBuyQuantity(selectedBuyItem.stock);
                      } else {
                        setBuyQuantity(val);
                      }
                    }}
                    onBlur={() => {
                      if (!buyQuantity || buyQuantity <= 0) setBuyQuantity(1);
                    }}
                    placeholder="1"
                    className="w-24 text-center bg-transparent font-bold text-zinc-100 font-mono text-sm focus:outline-none focus:bg-zinc-900/50 rounded-lg py-1 transition-colors [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                  />
                  <button
                    type="button"
                    disabled={selectedBuyItem.stock > 0 && buyQuantity >= selectedBuyItem.stock}
                    onClick={() => setBuyQuantity((prev) => {
                      const next = (prev || 0) + 1;
                      return selectedBuyItem.stock > 0 ? Math.min(next, selectedBuyItem.stock) : next;
                    })}
                    className="w-10 h-10 flex items-center justify-center font-bold text-zinc-400 hover:bg-zinc-900 active:bg-zinc-950 disabled:opacity-30 rounded-lg cursor-pointer transition-colors"
                  >
                    +
                  </button>
                </div>
                <div className="flex gap-1.5 pt-0.5">
                  {[10, 50, 100].map((step) => (
                    <button
                      key={step}
                      type="button"
                      onClick={() => {
                        setBuyQuantity((prev) => {
                          const next = (prev || 0) + step;
                          return selectedBuyItem.stock > 0 ? Math.min(next, selectedBuyItem.stock) : next;
                        });
                      }}
                      className="flex-1 py-1 text-[11px] font-mono text-zinc-400 bg-zinc-900/50 hover:bg-zinc-900 border border-zinc-850 hover:border-zinc-700 rounded-lg transition-all cursor-pointer"
                    >
                      +{step}
                    </button>
                  ))}
                  <button
                    type="button"
                    onClick={() => {
                      const affordable = selectedBuyItem.price > 0 ? Math.floor(getBalance(buyTier) / selectedBuyItem.price) : 999;
                      const max = selectedBuyItem.stock > 0 ? Math.min(affordable, selectedBuyItem.stock) : affordable;
                      setBuyQuantity(Math.max(1, max));
                    }}
                    className="flex-1 py-1 text-[11px] font-mono text-amber-400 bg-amber-500/10 hover:bg-amber-500/20 border border-amber-500/20 hover:border-amber-500/30 rounded-lg transition-all cursor-pointer font-semibold"
                  >
                    Tối đa
                  </button>
                </div>
              </div>

            {/* Total Cost Display */}
            <div className="flex justify-between items-center pt-2 font-mono border-t border-dashed border-zinc-900 text-xs">
              <span className="text-zinc-400">Tổng thanh toán:</span>
              <span className={`font-bold flex items-center gap-1 text-sm ${buyTier === "immortal" ? "text-violet-400" : "text-amber-400"}`}>
                {buyTier === "immortal"
                  ? <Sparkles className="w-4 h-4 text-violet-400" />
                  : <Coins className="w-4 h-4 text-amber-500" />}
                {(selectedBuyItem.price * (buyQuantity || 0)).toLocaleString()} {currencyLabel(buyTier)}
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




    </GamePanel>
  );
};
