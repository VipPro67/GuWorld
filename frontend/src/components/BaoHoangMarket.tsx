"use client";

import { useEffect, useState } from "react";
import { GuInstance, Player, GuSpec, ItemSpec, useGameStore } from "@/store/useGameStore";
import {
  AlertCircle,
  ShoppingCart,
  Gavel,
  PlusCircle,
  Search,
  Coins,
  Clock,
  Trash2,
  Tag,
  BookOpen,
  Sparkles,
  RefreshCw,
  AlertTriangle,
} from "lucide-react";

interface RecipeData {
  recipe_id: string;
  completion_rate: number;
}

interface Listing {
  id: string;
  seller_id: string;
  item_type: "Gu" | "Material" | "Recipe";
  item_data: string | RecipeData;
  quantity: number;
  currency: "primeval" | "immortal";
  start_price: number;
  current_price: number;
  highest_bidder_id: string;
  end_time: string;
  listing_type: "MARKET" | "AUCTION" | "TRADE";
  target_buyer_id?: string;
  target_buyer_username?: string;
}

interface Props {
  player: Player;
  guInventory: GuInstance[];
  materials: Record<string, number>;
  actionLoading: boolean;
  listings: Listing[];
  onRefresh: () => Promise<void>;
  onCreate: (payload: {
    itemType: "Gu" | "Material" | "Recipe";
    guInstanceId?: string;
    guInstanceIds?: string[];
    itemId?: string;
    quantity: number;
    price: number;
    durationHours: number;
    currency: "primeval" | "immortal";
    listingType: "MARKET" | "AUCTION" | "TRADE";
    targetBuyerUsername?: string;
  }) => Promise<boolean>;
  onBid: (listingId: string, amount: number) => Promise<boolean>;
  onCancel: (listingId: string) => Promise<boolean>;
  onBuy: (listingId: string, quantity?: number) => Promise<boolean>;
  guDictionary: Record<string, GuSpec>;
  itemDictionary: Record<string, ItemSpec>;
}

export function BaoHoangMarket({
  player,
  guInventory,
  materials,
  actionLoading,
  listings,
  onRefresh,
  onCreate,
  onBid,
  onCancel,
  onBuy,
  guDictionary,
  itemDictionary,
}: Props) {
  const [tab, setTab] = useState<"market" | "auction" | "sell">("market");
  const [listingType, setListingType] = useState<"MARKET" | "AUCTION">("MARKET");
  const [targetBuyerUsername, setTargetBuyerUsername] = useState("");
  const [itemType, setItemType] = useState<"Gu" | "Material" | "Recipe">("Material");
  
  // Sell form inputs
  const [price, setPrice] = useState(10);
  const [durationHours, setDurationHours] = useState(24);
  const [selectedGuType, setSelectedGuType] = useState("");
  const [selectedGus, setSelectedGus] = useState<string[]>([]);
  const [selectedMaterial, setSelectedMaterial] = useState("");
  const [selectedRecipe, setSelectedRecipe] = useState("");
  const [quantity, setQuantity] = useState(1);
  const [currency, setCurrency] = useState<"primeval" | "immortal">("primeval");
  
  // Search & Filters
  const [searchQuery, setSearchQuery] = useState("");
  const [filterType, setFilterType] = useState<"all" | "Gu" | "Material" | "Recipe">("all");
  
  const [bidAmounts, setBidAmounts] = useState<Record<string, number>>({});
  const [activePurchaseListing, setActivePurchaseListing] = useState<Listing | null>(null);
  const [purchaseQuantity, setPurchaseQuantity] = useState<number>(1);
  const [activeBidListing, setActiveBidListing] = useState<Listing | null>(null);
  const [bidAmount, setBidAmount] = useState<number>(0);
  const [message, setMessage] = useState<{ type: "error" | "success"; text: string } | null>(null);

  useEffect(() => {
    onRefresh();
  }, [onRefresh]);

  useEffect(() => {
    if (!message) return;
    const timer = setTimeout(() => setMessage(null), 5000);
    return () => clearTimeout(timer);
  }, [message]);

  const materialKeys = Object.keys(materials);

  const getDisplayName = (listing: Listing): string => {
    if (listing.item_type === "Recipe") {
      const rData = typeof listing.item_data === "object" ? listing.item_data : null;
      if (rData) {
        const targetGuName = guDictionary[rData.recipe_id.replace("recipe_", "GU_").toUpperCase()]?.name || 
                           guDictionary[rData.recipe_id.replace("recipe_", "GU")]?.name ||
                           rData.recipe_id.replace("recipe_", "").replace(/_/g, " ");
        return `Bí phương: ${targetGuName} (${rData.completion_rate}%)`;
      }
      return `Bí phương: ${String(listing.item_data)}`;
    }
    
    const dataStr = typeof listing.item_data === "string" ? listing.item_data : "";
    if (listing.item_type === "Gu") {
      const spec = guDictionary[dataStr];
      return spec?.name || dataStr.replace(/^GU_/, "").replace(/_/g, " ").replace(/\b\w/g, c => c.toUpperCase());
    }
    
    const spec = itemDictionary[dataStr];
    return spec?.name || dataStr.replace(/_/g, " ").replace(/\b\w/g, c => c.toUpperCase());
  };

  const getRecipeDisplayName = (recipeId: string): string => {
    const targetGuID = recipeId.replace("recipe_", "GU_").toUpperCase();
    const targetGuIDAlt = recipeId.replace("recipe_", "GU").toUpperCase();
    const name = guDictionary[targetGuID]?.name || guDictionary[targetGuIDAlt]?.name || recipeId;
    return `Bí phương: ${name}`;
  };

  // Fees calculation
  const getUpfrontFee = () => {
    return listingType === "MARKET" ? 5 : 20;
  };
  const getCommissionRateText = () => {
    return listingType === "MARKET" ? "2%" : "5%";
  };

  const createListing = async () => {
    if (actionLoading) return;

    if (itemType === "Gu" && selectedGus.length === 0) {
      setMessage({ type: "error", text: "Vui lòng chọn ít nhất một Cổ trùng cần bán." });
      return;
    }
    if (itemType === "Material" && !selectedMaterial) {
      setMessage({ type: "error", text: "Vui lòng chọn nguyên liệu cần bán." });
      return;
    }
    if (itemType === "Recipe" && !selectedRecipe) {
      setMessage({ type: "error", text: "Vui lòng chọn Bí phương cần bán." });
      return;
    }
    if (quantity < 1) {
      setMessage({ type: "error", text: "Số lượng phải tối thiểu là 1." });
      return;
    }
    if (price <= 0) {
      setMessage({ type: "error", text: "Đơn giá/Giá khởi điểm phải lớn hơn 0." });
      return;
    }
    if (durationHours < 1) {
      setMessage({ type: "error", text: "Thời gian đăng bán phải từ 1 giờ." });
      return;
    }

    const upfrontFee = getUpfrontFee();
    if (player.primeval_stones < upfrontFee) {
      setMessage({ type: "error", text: `Đạo hữu không đủ Nguyên Thạch đóng phí cơ bản (cần ${upfrontFee} Nguyên Thạch).` });
      return;
    }

    if (itemType === "Gu") {
      const ok = await onCreate({
        itemType,
        guInstanceIds: selectedGus,
        quantity: selectedGus.length,
        price,
        durationHours,
        currency: player.rank >= 6 ? currency : "primeval",
        listingType,
        targetBuyerUsername: undefined,
      } as any);

      if (ok) {
        await onRefresh();
        setSelectedGuType("");
        setSelectedGus([]);
        setTab(listingType === "MARKET" ? "market" : "auction");
        setMessage({ type: "success", text: `Đã đăng bán thành công ${selectedGus.length} Cổ trùng!` });
        return;
      }
      setMessage({ type: "error", text: "Đăng bán thất bại. Hãy kiểm tra số dư Nguyên Thạch của bạn." });
    } else {
      const ok = await onCreate({
        itemType,
        itemId: itemType === "Material" ? selectedMaterial : selectedRecipe,
        recipeId: itemType === "Recipe" ? selectedRecipe : undefined,
        itemIdRecipe: itemType === "Recipe" ? selectedRecipe : undefined,
        quantity: itemType === "Material" ? quantity : 1,
        price,
        durationHours,
        currency: player.rank >= 6 ? currency : "primeval",
        listingType,
        targetBuyerUsername: undefined,
      } as any);

      if (ok) {
        await onRefresh();
        setSelectedMaterial("");
        setSelectedRecipe("");
        setQuantity(1);
        setTab(listingType === "MARKET" ? "market" : "auction");
        setMessage({ type: "success", text: "Đăng bán thành công!" });
        return;
      }
      setMessage({ type: "error", text: "Đăng bán thất bại. Hãy kiểm tra số dư Nguyên Thạch của bạn." });
    }
  };

  const handleBuy = async (
    listingId: string,
    purchaseQty: number,
    unitPrice: number,
    currType: string,
    displayName: string,
    skipConfirm = false
  ) => {
    if (actionLoading) return;

    const listing = listings.find((l) => l.id === listingId);
    if (listing && listing.listing_type === "TRADE") {
      const hasConnectingHeaven = guInventory.some(
        (g) => g.gu_id === "GU_CONNECTING_HEAVEN" && g.status !== "DEAD"
      );
      if (!hasConnectingHeaven) {
        setMessage({ type: "error", text: "Để thực hiện giao dịch trực tiếp, bạn cần có Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN)!" });
        return;
      }
    }

    const totalPrice = purchaseQty * unitPrice;
    if (currType === "primeval" && player.primeval_stones < totalPrice) {
      setMessage({ type: "error", text: "Đạo hữu không đủ Nguyên Thạch." });
      return;
    }
    if (currType === "immortal" && player.immortal_stones < totalPrice) {
      setMessage({ type: "error", text: "Đạo hữu không đủ Tiên Nguyên Thạch." });
      return;
    }

    if (!skipConfirm) {
      const currencyText = currType === "primeval" ? "Nguyên Thạch" : "Tiên Nguyên Thạch";
      const confirmMsg = `Đạo hữu có chắc chắn muốn dùng ${totalPrice} ${currencyText} để mua ngay "${displayName}" không?`;
      if (!await useGameStore.getState().showConfirm(confirmMsg)) {
        return;
      }
    }

    const ok = await onBuy(listingId, purchaseQty);
    if (ok) {
      await onRefresh();
      setMessage({ type: "success", text: "Mua hàng thành công! Linh vật đã được gửi vào túi đồ." });
      return;
    }
    setMessage({ type: "error", text: "Giao dịch thất bại. Hãy thử lại." });
  };

  const handleBid = async (
    listingId: string,
    amount: number,
    currentPrice: number,
    displayName: string,
    skipConfirm = false
  ) => {
    if (actionLoading) return;

    if (amount <= currentPrice) {
      setMessage({ type: "error", text: "Giá đặt phải cao hơn giá hiện tại của phiên đấu giá." });
      return;
    }
    if (player.primeval_stones < amount) {
      setMessage({ type: "error", text: "Đạo hữu không đủ Nguyên Thạch để đặt giá này." });
      return;
    }

    if (!skipConfirm) {
      const confirmMsg = `Đạo hữu có chắc chắn muốn đặt giá thầu ${amount} Nguyên Thạch cho "${displayName}" không?`;
      if (!await useGameStore.getState().showConfirm(confirmMsg)) {
        return;
      }
    }

    const ok = await onBid(listingId, amount);
    if (ok) {
      await onRefresh();
      setMessage({ type: "success", text: "Đặt giá đấu thầu thành công." });
      return;
    }
    setMessage({ type: "error", text: "Đặt giá thất bại. Có thể đã có người thầu giá cao hơn." });
  };

  const handleCancel = async (listingId: string) => {
    if (actionLoading) return;

    const ok = await onCancel(listingId);
    if (ok) {
      await onRefresh();
      setMessage({ type: "success", text: "Đã hủy đăng bán và hoàn trả linh vật vào hành lý." });
      return;
    }
    setMessage({ type: "error", text: "Không thể hủy phiên giao dịch lúc này." });
  };

  // Filter listings
  const filteredListings = listings.filter((l) => {
    if (tab === "market" && l.listing_type !== "MARKET") return false;
    if (tab === "auction" && l.listing_type !== "AUCTION") return false;
    
    if (filterType !== "all" && l.item_type !== filterType) return false;
    
    if (searchQuery) {
      const name = getDisplayName(l).toLowerCase();
      if (!name.includes(searchQuery.toLowerCase())) return false;
    }
    return true;
  });

  if (!Array.isArray(guInventory) || !materials || typeof materials !== 'object') {
    return (
      <div className="w-full max-w-6xl mx-auto glass-panel-gold p-6 rounded-3xl border border-amber-500/20 shadow-2xl relative overflow-hidden">
        <div className="absolute top-0 right-0 w-64 h-64 bg-amber-500/5 rounded-full blur-[100px] pointer-events-none" />
        <div className="absolute bottom-0 left-0 w-80 h-80 bg-cyan-500/5 rounded-full blur-[120px] pointer-events-none" />
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <AlertCircle className="w-16 h-16 text-zinc-700 mb-6" />
          <h3 className="text-lg font-serif font-bold text-zinc-500 mb-2">Chưa thể kết nối Bảo Hoàng Thiên</h3>
          <p className="text-sm text-zinc-600 font-serif max-w-md">
            Đạo hữu chưa đủ điều kiện để tham gia giao dịch tại Bảo Hoàng Thiên. 
            Hãy tu luyện và quay lại sau.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full max-w-6xl mx-auto glass-panel-gold p-6 rounded-3xl border border-amber-500/20 shadow-2xl relative overflow-hidden">
      {/* Background decoration */}
      <div className="absolute top-0 right-0 w-64 h-64 bg-amber-500/5 rounded-full blur-[100px] pointer-events-none" />
      <div className="absolute bottom-0 left-0 w-80 h-80 bg-cyan-500/5 rounded-full blur-[120px] pointer-events-none" />

      {/* Header section */}
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4 border-b border-amber-500/10 pb-5 mb-6">
        <div>
          <h2 className="text-2xl font-bold font-serif text-transparent bg-clip-text bg-gradient-to-r from-amber-200 via-amber-300 to-amber-500 tracking-wider flex items-center gap-2">
            <Sparkles className="w-6 h-6 text-amber-400 animate-pulse" />
            BẢO HOÀNG THIÊN
          </h2>
          <p className="text-xs text-zinc-500 font-mono mt-1">
            Sàn giao dịch tối cao xuyên Không gian & Thời gian • Miễn phí giao thương cho mọi Cổ Sư
          </p>
        </div>

        {/* Player Assets display */}
        <div className="flex items-center gap-3 bg-zinc-950/60 px-4 py-2.5 rounded-2xl border border-zinc-800/80 font-mono text-xs">
          <div className="flex items-center gap-1.5 pr-3 border-r border-zinc-850">
            <Coins className="w-4 h-4 text-amber-500" />
            <span className="text-zinc-400">Nguyên Thạch:</span>
            <span className="text-amber-400 font-bold text-sm">{player.primeval_stones}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Sparkles className="w-4 h-4 text-violet-400" />
            <span className="text-zinc-400">Tiên Nguyên:</span>
            <span className="text-violet-300 font-bold text-sm">{player.immortal_stones.toFixed(2)}</span>
          </div>
        </div>
      </div>

      {/* Navigation tabs & Actions */}
      <div className="flex flex-col items-stretch gap-3 mb-6">
        <div className="flex bg-zinc-950/80 p-1.5 rounded-2xl border border-zinc-900 gap-1.5 flex-wrap">
          <button
            onClick={() => setTab("market")}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold font-serif transition-all cursor-pointer ${
              tab === "market"
                ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <ShoppingCart className="w-4 h-4" />
            Chợ Giao Dịch
          </button>
          <button
            onClick={() => setTab("auction")}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold font-serif transition-all cursor-pointer ${
              tab === "auction"
                ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <Gavel className="w-4 h-4" />
            Sảnh Đấu Giá
          </button>
          <button
            onClick={() => setTab("sell")}
            className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold font-serif transition-all cursor-pointer ${
              tab === "sell"
                ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                : "text-zinc-400 hover:text-zinc-200"
            }`}
          >
            <PlusCircle className="w-4 h-4" />
            Đăng Bán
          </button>
        </div>

        {tab !== "sell" && (
          <div className="flex items-center gap-2 flex-wrap">
            <button
              onClick={onRefresh}
              disabled={actionLoading}
              className="bg-zinc-950/80 border border-zinc-800 hover:border-zinc-700 hover:text-zinc-100 p-2 rounded-xl text-zinc-400 transition-all cursor-pointer shrink-0"
              title="Làm mới"
            >
              <RefreshCw className={`w-4 h-4 ${actionLoading ? "animate-spin" : ""}`} />
            </button>
            
            {/* Search Input */}
            <div className="relative flex-1 min-w-[140px]">
              <Search className="w-4 h-4 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                placeholder="Tìm tên bảo vật..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full bg-zinc-950/80 border border-zinc-850 hover:border-zinc-700 focus:border-amber-500/30 text-xs text-zinc-200 pl-9 pr-4 py-2 rounded-xl focus:outline-none focus:ring-0 transition-all placeholder:text-zinc-600 font-serif"
              />
            </div>

            {/* Type Filter */}
            <select
              value={filterType}
              onChange={(e) => setFilterType(e.target.value as any)}
              className="bg-zinc-950/80 border border-zinc-850 hover:border-zinc-700 text-xs text-zinc-400 px-3 py-2 rounded-xl focus:outline-none focus:ring-0 font-serif shrink-0"
            >
              <option value="all">Tất cả loại</option>
              <option value="Gu">Cổ Trùng</option>
              <option value="Material">Vật Phẩm</option>
              <option value="Recipe">Bí Phương</option>
            </select>
          </div>
        )}
      </div>

      {/* Message Notifications */}
      {message && (
        <div
          className={`flex items-start gap-2.5 p-4 mb-6 rounded-2xl border ${
            message.type === "error"
              ? "border-rose-500/30 bg-rose-500/5 text-rose-200"
              : "border-emerald-500/30 bg-emerald-500/5 text-emerald-250"
          } animate-in fade-in duration-300`}
        >
          <AlertCircle className="w-5 h-5 mt-0.5 shrink-0" />
          <div className="text-xs leading-normal">{message.text}</div>
        </div>
      )}

      {/* Tab Contents */}
      {tab === "sell" ? (
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
          {/* Sell Form */}
          <div className="lg:col-span-8 space-y-5 bg-zinc-950/40 border border-zinc-900 p-6 rounded-2xl">
            <h3 className="text-sm font-serif font-bold text-amber-300 border-b border-zinc-900 pb-2 mb-4 uppercase tracking-wider">
              Thiết lập giao dịch mới
            </h3>

            {/* Step 1: Listing Mode */}
            <div className="space-y-3">
              <span className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider block">Chế độ giao dịch</span>
              <div className="flex gap-2 bg-zinc-950 p-1.5 rounded-xl border border-zinc-900 flex-wrap sm:flex-nowrap">
                <button
                  type="button"
                  onClick={() => setListingType("MARKET")}
                  className={`flex-1 py-2 text-xs font-serif font-bold rounded-lg transition-all cursor-pointer min-w-[100px] ${
                    listingType === "MARKET"
                      ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                      : "text-zinc-500 hover:text-zinc-300"
                  }`}
                >
                  Chợ (Mở)
                </button>
                <button
                  type="button"
                  onClick={() => setListingType("AUCTION")}
                  className={`flex-1 py-2 text-xs font-serif font-bold rounded-lg transition-all cursor-pointer min-w-[100px] ${
                    listingType === "AUCTION"
                      ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                      : "text-zinc-500 hover:text-zinc-300"
                  }`}
                >
                  Đấu Giá
                </button>
              </div>
            </div>

            {/* Step 2: Item category & specifics */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <label className="space-y-2">
                <span className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider block">Phân loại hàng hóa</span>
                <select
                  value={itemType}
                  onChange={(e) => {
                    setItemType(e.target.value as any);
                    setSelectedGuType("");
                    setSelectedGus([]);
                    setSelectedMaterial("");
                    setSelectedRecipe("");
                    setQuantity(1);
                  }}
                  className="w-full bg-zinc-950 border border-zinc-900 hover:border-zinc-800 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:ring-0 font-serif"
                >
                  <option value="Material">Nguyên Liệu / Vật Phẩm</option>
                  <option value="Gu">Cổ Trùng</option>
                  <option value="Recipe">Bí Phương</option>
                </select>
              </label>

              {itemType === "Gu" && (
                <div className="space-y-2 block sm:col-span-2">
                  <span className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider block">Chọn Cổ trùng đăng bán</span>
                  
                  {/* Select Gu Type */}
                  <select
                    value={selectedGuType}
                    onChange={(e) => {
                      setSelectedGuType(e.target.value);
                      setSelectedGus([]);
                    }}
                    className="w-full bg-zinc-950 border border-zinc-900 hover:border-zinc-800 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:ring-0 font-serif"
                  >
                    <option value="">-- Chọn loại Cổ trùng cần bán --</option>
                    {(() => {
                      const tradeable = guInventory.filter((g) => !g.is_vital && g.status !== "DEAD");
                      const typesMap: Record<string, number> = {};
                      tradeable.forEach((g) => {
                        typesMap[g.gu_id] = (typesMap[g.gu_id] || 0) + 1;
                      });
                      return Object.entries(typesMap).map(([guId, count]) => {
                        const name = guDictionary[guId]?.name || guId;
                        const rank = guDictionary[guId]?.rank || 1;
                        return (
                          <option key={guId} value={guId}>
                            {name} [Rank {rank}] ({count} con khả dụng)
                          </option>
                        );
                      });
                    })()}
                  </select>

                  {/* If a Gu Type is selected, show the instances list with checkboxes */}
                  {selectedGuType && (
                    <div className="border border-zinc-900 rounded-xl p-3 bg-zinc-950/40 space-y-2">
                      <div className="flex justify-between items-center pb-2 border-b border-zinc-900/60">
                        <span className="text-[10px] text-zinc-400 font-mono">
                          Đã chọn {selectedGus.length} con
                        </span>
                        <div className="flex gap-2">
                          <button
                            type="button"
                            onClick={() => {
                              const insts = guInventory
                                .filter((g) => g.gu_id === selectedGuType && !g.is_vital && g.status !== "DEAD")
                                .map((g) => g.instance_id);
                              setSelectedGus(insts);
                            }}
                            className="text-[9px] text-amber-400 hover:underline font-mono cursor-pointer"
                          >
                            Chọn tất cả
                          </button>
                          <span className="text-zinc-755 font-mono">|</span>
                          <button
                            type="button"
                            onClick={() => setSelectedGus([])}
                            className="text-[9px] text-zinc-500 hover:underline font-mono cursor-pointer"
                          >
                            Bỏ chọn
                          </button>
                        </div>
                      </div>

                      <div className="max-h-40 overflow-y-auto space-y-1.5 pr-1">
                        {guInventory
                          .filter((g) => g.gu_id === selectedGuType && !g.is_vital && g.status !== "DEAD")
                          .map((g) => {
                            const selected = selectedGus.includes(g.instance_id);
                            const hungerText = `Độ no: ${g.hunger}/100`;
                            const statusText = g.equipped ? "Xuất Trận" : g.status === "SLEEPING" ? "Ngủ Đông" : g.status === "SEALED" ? "Phong Ấn" : "Chờ Lệnh";
                            
                            return (
                              <label
                                key={g.instance_id}
                                className={`flex items-center justify-between p-2.5 rounded-lg border text-xs font-mono transition-all cursor-pointer select-none ${
                                  selected
                                    ? "bg-amber-500/5 border-amber-500/30 text-amber-400"
                                    : "bg-zinc-950/20 border-zinc-900/60 hover:border-zinc-800 text-zinc-400"
                                }`}
                              >
                                <div className="flex items-center gap-2">
                                  <input
                                    type="checkbox"
                                    checked={selected}
                                    onChange={(e) => {
                                      if (e.target.checked) {
                                        setSelectedGus((prev) => [...prev, g.instance_id]);
                                      } else {
                                        setSelectedGus((prev) => prev.filter((id) => id !== g.instance_id));
                                      }
                                    }}
                                    className="rounded border-zinc-800 bg-zinc-950 text-amber-500 focus:ring-0 focus:ring-offset-0 cursor-pointer"
                                  />
                                  <span className="font-bold">#{g.instance_id.slice(0, 6)}</span>
                                </div>
                                <div className="flex items-center gap-2 text-[10px]">
                                  <span className="text-zinc-500">{hungerText}</span>
                                  <span className="text-zinc-650">•</span>
                                  <span className={g.equipped ? "text-emerald-500 font-bold" : g.status === "SLEEPING" ? "text-amber-500/80" : g.status === "SEALED" ? "text-cyan-400" : "text-zinc-405"}>
                                    {statusText}
                                  </span>
                                </div>
                              </label>
                            );
                          })}
                      </div>
                    </div>
                  )}
                </div>
              )}

              {itemType === "Recipe" && (
                <label className="space-y-2 block">
                  <span className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider block">Chọn Bí Phương cần bán</span>
                  <select
                    value={selectedRecipe}
                    onChange={(e) => setSelectedRecipe(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-900 hover:border-zinc-800 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:ring-0 font-serif"
                  >
                    <option value="">-- Chọn công thức ({player.unlocked_recipes?.length || 0} khả dụng) --</option>
                    {player.unlocked_recipes?.map((ur) => (
                      <option key={ur.recipe_id} value={ur.recipe_id}>
                        {getRecipeDisplayName(ur.recipe_id)} ({ur.completion_rate}%)
                      </option>
                    ))}
                  </select>
                </label>
              )}

              {itemType === "Material" && (
                <label className="space-y-2 block">
                  <span className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider block">Chọn Vật phẩm</span>
                  <select
                    value={selectedMaterial}
                    onChange={(e) => setSelectedMaterial(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-900 hover:border-zinc-800 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:ring-0 font-serif"
                  >
                    <option value="">-- Chọn vật phẩm ({materialKeys.length} khả dụng) --</option>
                    {materialKeys.map((k) => (
                      <option key={k} value={k}>
                        {itemDictionary[k]?.name || k} (Đang có x{materials[k]})
                      </option>
                    ))}
                  </select>
                </label>
              )}
            </div>

            {/* Step 3: Quantity & Price */}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <label className="space-y-2">
                <span className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider block">Số lượng</span>
                <input
                  type="number"
                  min={1}
                  disabled={itemType !== "Material"}
                  value={quantity}
                  onChange={(e) => setQuantity(Number(e.target.value))}
                  className="w-full bg-zinc-950 border border-zinc-900 hover:border-zinc-800 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:ring-0 font-mono disabled:opacity-50 disabled:cursor-not-allowed"
                />
              </label>

              <label className="space-y-2">
                <span className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider block">
                  {listingType === "MARKET" ? "Đơn giá bán" : "Giá khởi điểm"}
                </span>
                <input
                  type="number"
                  min={1}
                  value={price}
                  onChange={(e) => setPrice(Number(e.target.value))}
                  className="w-full bg-zinc-950 border border-zinc-900 hover:border-zinc-800 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:ring-0 font-mono"
                  placeholder="100"
                />
              </label>

              <label className="space-y-2">
                <span className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider block">Thời hạn thầu</span>
                <select
                  value={durationHours}
                  onChange={(e) => setDurationHours(Number(e.target.value))}
                  className="w-full bg-zinc-950 border border-zinc-900 hover:border-zinc-800 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:ring-0 font-serif"
                >
                  <option value={12}>12 Giờ</option>
                  <option value={24}>24 Giờ</option>
                  <option value={48}>48 Giờ</option>
                  <option value={72}>72 Giờ</option>
                </select>
              </label>
            </div>

            {/* Currency Option */}
            <div className="space-y-2">
              <span className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider block">Loại tiền tệ thụ hưởng</span>
              {player.rank < 6 ? (
                <div className="flex items-center gap-2 bg-zinc-950/60 p-3 rounded-xl border border-zinc-900 text-xs text-zinc-500 font-serif">
                  <AlertTriangle className="w-4 h-4 text-amber-500" />
                  <span>Tu vi phàm nhân (dưới 6 Chuyển) chỉ giao dịch bằng <strong>Nguyên Thạch</strong>.</span>
                </div>
              ) : (
                <div className="flex gap-2 bg-zinc-950 p-1.5 rounded-xl border border-zinc-900">
                  <button
                    type="button"
                    onClick={() => setCurrency("primeval")}
                    className={`flex-1 py-2 text-xs font-serif font-bold rounded-lg transition-all cursor-pointer ${
                      currency === "primeval"
                        ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                        : "text-zinc-500 hover:text-zinc-300"
                    }`}
                  >
                    Nguyên Thạch
                  </button>
                  <button
                    type="button"
                    onClick={() => setCurrency("immortal")}
                    className={`flex-1 py-2 text-xs font-serif font-bold rounded-lg transition-all cursor-pointer ${
                      currency === "immortal"
                        ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                        : "text-zinc-500 hover:text-zinc-300"
                    }`}
                  >
                    Tiên Nguyên Thạch
                  </button>
                </div>
              )}
            </div>
          </div>

          {/* Fee details & submit card */}
          <div className="lg:col-span-4 space-y-4">
            {/* Fee Box */}
            <div className="bg-zinc-950/60 border border-amber-500/15 p-5 rounded-2xl space-y-4 font-mono text-xs relative">
              <div className="absolute top-0 right-0 w-24 h-24 bg-amber-500/5 rounded-full blur-2xl pointer-events-none" />
              <h4 className="font-serif font-bold text-amber-300 border-b border-zinc-900 pb-2 uppercase text-center">
                Bảng Ước Tính Lệ Phí
              </h4>

              <div className="space-y-2.5">
                <div className="flex justify-between items-center text-zinc-400">
                  <span>Phí Đăng Upfront:</span>
                  <span className="text-zinc-200 font-bold">{getUpfrontFee()} Nguyên Thạch</span>
                </div>
                <div className="flex justify-between items-center text-zinc-500 text-[10px] leading-tight pl-2">
                  <span>* Thanh toán tức thì bằng Nguyên Thạch và không được hoàn lại kể cả khi hủy giao dịch.</span>
                </div>
                
                <div className="h-px bg-zinc-900 my-2" />

                <div className="flex justify-between items-center text-zinc-400">
                  <span>Phí Hoa Hồng (sau bán):</span>
                  <span className="text-amber-400 font-bold">{getCommissionRateText()} doanh thu</span>
                </div>
                <div className="flex justify-between items-center text-zinc-500 text-[10px] leading-tight pl-2">
                  <span>* Chỉ khấu trừ khi hàng hóa bán thành công.</span>
                </div>
              </div>

              <div className="bg-zinc-950 p-3.5 rounded-xl border border-zinc-900 space-y-1.5">
                <div className="text-[10px] text-zinc-500 uppercase tracking-wider block">Giao Dịch Dự Kiến</div>
                <div className="text-zinc-300 font-serif">
                  {itemType === "Material" ? `${quantity}x ` : itemType === "Gu" ? `${selectedGus.length}x ` : "1x "} 
                  <span className="font-bold text-zinc-100">
                    {itemType === "Gu" && selectedGuType ? guDictionary[selectedGuType]?.name || selectedGuType : ""}
                    {itemType === "Material" && selectedMaterial ? itemDictionary[selectedMaterial]?.name || selectedMaterial : ""}
                    {itemType === "Recipe" && selectedRecipe ? getRecipeDisplayName(selectedRecipe) : ""}
                  </span>
                </div>
                <div className="text-[11px] text-amber-400 font-bold">
                  Tổng thu thầu: {itemType === "Material" ? quantity * price : itemType === "Gu" ? selectedGus.length * price : price} {currency === "primeval" ? "Nguyên Thạch" : "Tiên Nguyên"}
                </div>
              </div>
            </div>

            <button
              disabled={actionLoading || player.primeval_stones < getUpfrontFee()}
              onClick={createListing}
              className={`w-full py-3.5 px-4 rounded-xl font-serif font-bold text-xs tracking-wider transition-all flex items-center justify-center gap-2 cursor-pointer shadow-lg ${
                player.primeval_stones >= getUpfrontFee() && !actionLoading
                  ? "bg-gradient-to-r from-amber-500 via-amber-600 to-orange-500 hover:from-amber-400 hover:to-orange-400 text-zinc-950 shadow-amber-950/10 active:scale-[0.98]"
                  : "bg-zinc-900 border border-zinc-850 text-zinc-650 cursor-not-allowed"
              }`}
            >
              <Sparkles className="w-4 h-4" />
              ĐỒNG Ý ĐĂNG BÁN
            </button>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          {/* Active Listings Display */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {filteredListings.map((listing) => {
              const isSelf = listing.seller_id === player.id;
              const unitPrice = listing.current_price;
              const totalPrice = listing.quantity * unitPrice;
              const currencyName = listing.currency === "primeval" ? "Nguyên Thạch" : "Tiên Nguyên Thạch";
              const currencyBadge = listing.currency === "primeval" 
                ? "text-amber-400 border-amber-500/20 bg-amber-950/10" 
                : "text-purple-400 border-purple-500/20 bg-purple-950/10";
              
              let typeBadge = "text-amber-400 border-amber-500/10 bg-amber-950/5";
              if (listing.item_type === "Material") typeBadge = "text-emerald-400 border-emerald-500/10 bg-emerald-950/5";
              if (listing.item_type === "Recipe") typeBadge = "text-violet-400 border-violet-500/10 bg-violet-950/5";

              return (
                <div
                  key={listing.id}
                  className="bg-zinc-950/55 border border-zinc-900 hover:border-zinc-850 rounded-2xl p-4 flex flex-col justify-between gap-4 transition-all relative overflow-hidden group shadow-md"
                >
                  {/* Decorative corner light */}
                  <div className="absolute top-0 right-0 w-20 h-20 bg-amber-500/2 rounded-full blur-xl group-hover:bg-amber-500/5 transition-all pointer-events-none" />

                  {/* Top info row */}
                  <div className="flex justify-between items-start gap-2 relative z-10">
                    <div className="space-y-1">
                      <div className="flex items-center flex-wrap gap-1">
                        <span className={`text-[9px] font-mono border px-1.5 py-0.2 rounded uppercase ${typeBadge}`}>
                          {listing.item_type === "Gu" ? "Cổ Trùng" : listing.item_type === "Material" ? "Vật Phẩm" : "Bí Phương"}
                        </span>
                        {listing.listing_type === "TRADE" && (
                          <span className="text-[9px] font-mono border border-amber-500/30 bg-amber-950/20 text-amber-400 px-1.5 py-0.2 rounded uppercase">
                            Giao Dịch Riêng
                          </span>
                        )}
                        {isSelf && (
                          <span className="text-[9px] font-mono border border-cyan-500/20 bg-cyan-950/15 text-cyan-400 px-1.5 py-0.2 rounded uppercase">
                            Của Bạn
                          </span>
                        )}
                      </div>
                      <h4 className="text-zinc-100 font-serif font-bold text-sm leading-snug group-hover:text-amber-300 transition-colors pt-0.5">
                        {getDisplayName(listing)}
                      </h4>
                      {listing.listing_type === "TRADE" && listing.target_buyer_username && (
                        <p className="text-[10px] text-amber-400/90 font-mono">
                          Người nhận: {listing.target_buyer_username}
                        </p>
                      )}
                      {listing.item_type === "Gu" && (
                        <p className="text-[10px] text-zinc-500 line-clamp-1 italic font-serif">
                          {guDictionary[typeof listing.item_data === "string" ? listing.item_data : ""]?.description || ""}
                        </p>
                      )}
                    </div>
                  </div>

                  {/* Price & quantities details */}
                  <div className="bg-zinc-950/80 p-3 rounded-xl border border-zinc-900/60 flex justify-between items-center font-mono text-[11px] leading-tight relative z-10 font-serif">
                    <div className="space-y-1">
                      <div>Số lượng: <span className="text-zinc-200 font-bold font-mono">{listing.quantity}</span></div>
                      <div>
                        Đơn giá:{" "}
                        <span className={`${listing.currency === "primeval" ? "text-amber-400" : "text-purple-400"} font-bold font-mono`}>
                          {unitPrice} {listing.currency === "primeval" ? "Nguyên Thạch" : "Tiên Nguyên"}
                        </span>
                      </div>
                    </div>
                    <div className="text-right space-y-1">
                      <div className="text-zinc-500">
                        {listing.listing_type === "MARKET" ? "Tổng tiền mua:" : listing.listing_type === "TRADE" ? "Giao dịch riêng:" : "Giá thầu hiện tại:"}
                      </div>
                      <div className={`${listing.currency === "primeval" ? "text-amber-400" : "text-purple-400"} font-bold text-xs font-mono`}>
                        {listing.listing_type === "MARKET" || listing.listing_type === "TRADE" ? totalPrice : listing.current_price} {listing.currency === "primeval" ? "Nguyên Thạch" : "Tiên Nguyên"}
                      </div>
                    </div>
                  </div>

                  {/* Expiry & Actions */}
                  <div className="flex items-center justify-between text-[10px] text-zinc-500 pt-1 border-t border-zinc-900/40 relative z-10 font-serif">
                    <div className="flex items-center gap-1 font-mono">
                      <Clock className="w-3.5 h-3.5 text-zinc-650" />
                      <span>
                        Hết hạn: {new Date(listing.end_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} {new Date(listing.end_time).toLocaleDateString([], { month: '2-digit', day: '2-digit' })}
                      </span>
                    </div>

                    <div className="flex gap-2 items-center">
                      {isSelf ? (
                        <button
                          disabled={actionLoading}
                          onClick={() => handleCancel(listing.id)}
                          className="px-3.5 py-1.5 rounded-xl bg-rose-950/20 border border-rose-900/30 hover:border-rose-500/50 hover:bg-rose-950/40 text-rose-300 font-bold font-serif text-[10px] cursor-pointer transition-all flex items-center gap-1 active:scale-95 disabled:opacity-50"
                        >
                          <Trash2 className="w-3 h-3" />
                          Hủy Bán
                        </button>
                      ) : (
                        <>
                          {listing.listing_type === "MARKET" || listing.listing_type === "TRADE" ? (
                            <button
                              disabled={actionLoading}
                              onClick={() => {
                                if (listing.quantity > 1) {
                                  setActivePurchaseListing(listing);
                                  setPurchaseQuantity(1);
                                } else {
                                  handleBuy(listing.id, 1, listing.current_price, listing.currency, getDisplayName(listing));
                                }
                              }}
                              className="px-4 py-1.5 rounded-xl bg-cyan-900/30 border border-cyan-500/30 hover:bg-cyan-900/60 hover:border-cyan-400 text-cyan-300 font-bold font-serif text-[10px] cursor-pointer transition-all active:scale-95"
                            >
                              Mua Ngay
                            </button>
                          ) : (
                            <button
                              disabled={actionLoading}
                              onClick={() => {
                                setActiveBidListing(listing);
                                setBidAmount(listing.current_price + 1);
                              }}
                              className="px-4 py-1.5 rounded-xl bg-amber-500/10 border border-amber-500/20 hover:bg-amber-500/20 hover:border-amber-400 text-amber-400 font-bold font-serif text-[10px] cursor-pointer transition-all active:scale-95"
                            >
                              Đấu Thầu
                            </button>
                          )}
                        </>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {filteredListings.length === 0 && (
            <div className="py-16 text-center border border-dashed border-zinc-900 rounded-3xl space-y-2 bg-zinc-950/20">
              <AlertCircle className="w-8 h-8 text-zinc-650 mx-auto" />
              <p className="text-zinc-500 text-sm font-serif italic">
                {tab === "market"
                  ? "Sàn chợ chưa có linh bảo nào đăng bán trực tiếp."
                  : tab === "auction"
                  ? "Chưa có bảo thạch nào được mang đấu thầu lúc này."
                  : "Đạo hữu không có giao dịch riêng tư nào lúc này."}
              </p>
            </div>
          )}
        </div>
      )}

      {/* Retail Buyout Modal */}
      {activePurchaseListing && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 animate-in fade-in duration-200">
          <div className="bg-zinc-950/95 border border-zinc-800 rounded-3xl p-6 max-w-sm w-full space-y-5 relative shadow-2xl">
            {/* Close button */}
            <button
              onClick={() => setActivePurchaseListing(null)}
              className="absolute -top-3 -right-3 w-8 h-8 rounded-full bg-violet-600 hover:bg-violet-500 text-white flex items-center justify-center shadow-lg transition-transform active:scale-90 cursor-pointer"
            >
              <span className="font-sans font-bold text-sm">✕</span>
            </button>

            {/* Header / Item Visual */}
            <div className="text-center space-y-3">
              <div className="w-20 h-20 mx-auto flex items-center justify-center rounded-2xl bg-gradient-to-br from-amber-500/10 to-orange-500/10 border border-amber-500/20 relative shadow-inner">
                {activePurchaseListing.item_type === "Gu" ? (
                  <Sparkles className="w-10 h-10 text-amber-400 animate-pulse" />
                ) : activePurchaseListing.item_type === "Recipe" ? (
                  <BookOpen className="w-10 h-10 text-violet-400" />
                ) : (
                  <Tag className="w-10 h-10 text-emerald-400" />
                )}
              </div>
              <h4 className="text-sm font-serif font-bold text-zinc-100 uppercase tracking-wide">
                {getDisplayName(activePurchaseListing)}
              </h4>
            </div>

            <div className="h-px bg-zinc-900" />

            {/* Form Fields */}
            <div className="space-y-4 text-xs font-serif">
              {/* Quantity Selector */}
              <div className="flex items-center justify-between">
                <span className="text-zinc-400">Số lượng</span>
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    disabled={purchaseQuantity <= 1}
                    onClick={() => setPurchaseQuantity(prev => Math.max(1, prev - 1))}
                    className="w-8 h-8 rounded-lg bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed flex items-center justify-center font-bold text-zinc-200 cursor-pointer transition-colors"
                  >
                    -
                  </button>
                  <input
                    type="number"
                    min={1}
                    max={activePurchaseListing.quantity}
                    value={purchaseQuantity}
                    onChange={(e) => {
                      const val = Math.max(1, Math.min(activePurchaseListing.quantity, Number(e.target.value)));
                      setPurchaseQuantity(val);
                    }}
                    className="w-14 bg-zinc-900/40 border border-zinc-850 rounded-lg p-1.5 font-mono text-xs text-zinc-200 text-center focus:outline-none focus:ring-0"
                  />
                  <button
                    type="button"
                    disabled={purchaseQuantity >= activePurchaseListing.quantity}
                    onClick={() => setPurchaseQuantity(prev => Math.min(activePurchaseListing.quantity, prev + 1))}
                    className="w-8 h-8 rounded-lg bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed flex items-center justify-center font-bold text-zinc-200 cursor-pointer transition-colors"
                  >
                    +
                  </button>
                </div>
              </div>

              {/* Unit Price */}
              <div className="flex justify-between items-center">
                <span className="text-zinc-400">Đơn giá</span>
                <div className="flex items-center gap-1.5 font-mono font-bold text-zinc-200">
                  <span>{activePurchaseListing.current_price}</span>
                  {activePurchaseListing.currency === "primeval" ? (
                    <Coins className="w-4 h-4 text-amber-500" />
                  ) : (
                    <Sparkles className="w-4 h-4 text-violet-400" />
                  )}
                </div>
              </div>

              {/* Total Price */}
              <div className="flex justify-between items-center">
                <span className="text-zinc-400">Tổng phí</span>
                <div className="flex items-center gap-1.5 font-mono font-bold text-zinc-200 text-sm">
                  <span>{(purchaseQuantity * activePurchaseListing.current_price)}</span>
                  {activePurchaseListing.currency === "primeval" ? (
                    <Coins className="w-4.5 h-4.5 text-amber-500" />
                  ) : (
                    <Sparkles className="w-4.5 h-4.5 text-violet-400" />
                  )}
                </div>
              </div>

              {/* Owned Balance */}
              <div className="flex justify-between items-center">
                <span className="text-zinc-400">Đang có</span>
                <div className="flex items-center gap-1.5 font-mono font-bold text-zinc-350">
                  <span>
                    {activePurchaseListing.currency === "primeval"
                      ? player.primeval_stones
                      : player.immortal_stones.toFixed(2)}
                  </span>
                  {activePurchaseListing.currency === "primeval" ? (
                    <Coins className="w-4 h-4 text-amber-500" />
                  ) : (
                    <Sparkles className="w-4 h-4 text-violet-400" />
                  )}
                </div>
              </div>
            </div>

            {/* Purchase Action Button */}
            <button
              onClick={async () => {
                const listing = activePurchaseListing;
                setActivePurchaseListing(null);
                await handleBuy(
                  listing.id,
                  purchaseQuantity,
                  listing.current_price,
                  listing.currency,
                  getDisplayName(listing),
                  true
                );
              }}
              className="w-full py-3 rounded-2xl bg-indigo-600 hover:bg-indigo-500 font-serif font-bold text-xs tracking-wider transition-all text-white shadow-lg cursor-pointer active:scale-[0.98]"
            >
              Mua
            </button>
          </div>
        </div>
      )}

      {/* Auction Bid Modal */}
      {activeBidListing && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 animate-in fade-in duration-200">
          <div className="bg-zinc-950/95 border border-zinc-800 rounded-3xl p-6 max-w-sm w-full space-y-5 relative shadow-2xl">
            {/* Close button */}
            <button
              onClick={() => setActiveBidListing(null)}
              className="absolute -top-3 -right-3 w-8 h-8 rounded-full bg-violet-600 hover:bg-violet-500 text-white flex items-center justify-center shadow-lg transition-transform active:scale-90 cursor-pointer"
            >
              <span className="font-sans font-bold text-sm">✕</span>
            </button>

            {/* Header / Item Visual */}
            <div className="text-center space-y-3">
              <div className="w-20 h-20 mx-auto flex items-center justify-center rounded-2xl bg-gradient-to-br from-amber-500/10 to-orange-500/10 border border-amber-500/20 relative shadow-inner">
                {activeBidListing.item_type === "Gu" ? (
                  <Sparkles className="w-10 h-10 text-amber-400 animate-pulse" />
                ) : activeBidListing.item_type === "Recipe" ? (
                  <BookOpen className="w-10 h-10 text-violet-400" />
                ) : (
                  <Tag className="w-10 h-10 text-emerald-400" />
                )}
              </div>
              <h4 className="text-sm font-serif font-bold text-zinc-100 uppercase tracking-wide">
                {getDisplayName(activeBidListing)}
              </h4>
            </div>

            <div className="h-px bg-zinc-900" />

            {/* Form Fields */}
            <div className="space-y-4 text-xs font-serif">
              {/* Current Price */}
              <div className="flex justify-between items-center">
                <span className="text-zinc-400">Giá hiện tại</span>
                <div className="flex items-center gap-1.5 font-mono font-bold text-zinc-200">
                  <span>{activeBidListing.current_price}</span>
                  {activeBidListing.currency === "primeval" ? (
                    <Coins className="w-4 h-4 text-amber-500" />
                  ) : (
                    <Sparkles className="w-4 h-4 text-violet-400" />
                  )}
                </div>
              </div>

              {/* Bid Amount Input */}
              <div className="flex items-center justify-between">
                <span className="text-zinc-400">Giá đấu</span>
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    disabled={bidAmount <= activeBidListing.current_price + 1}
                    onClick={() => setBidAmount(prev => Math.max(activeBidListing.current_price + 1, prev - 1))}
                    className="w-8 h-8 rounded-lg bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed flex items-center justify-center font-bold text-zinc-200 cursor-pointer transition-colors"
                  >
                    -
                  </button>
                  <input
                    type="number"
                    min={activeBidListing.current_price + 1}
                    value={bidAmount}
                    onChange={(e) => {
                      const val = Math.max(activeBidListing.current_price + 1, Number(e.target.value));
                      setBidAmount(val);
                    }}
                    className="w-20 bg-zinc-900/40 border border-zinc-850 rounded-lg p-1.5 font-mono text-xs text-zinc-200 text-center focus:outline-none focus:ring-0"
                  />
                  <button
                    type="button"
                    onClick={() => setBidAmount(prev => prev + 1)}
                    className="w-8 h-8 rounded-lg bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 flex items-center justify-center font-bold text-zinc-200 cursor-pointer transition-colors"
                  >
                    +
                  </button>
                </div>
              </div>

              {/* Owned Balance */}
              <div className="flex justify-between items-center">
                <span className="text-zinc-400">Đang có</span>
                <div className="flex items-center gap-1.5 font-mono font-bold text-zinc-350">
                  <span>
                    {activeBidListing.currency === "primeval"
                      ? player.primeval_stones
                      : player.immortal_stones.toFixed(2)}
                  </span>
                  {activeBidListing.currency === "primeval" ? (
                    <Coins className="w-4 h-4 text-amber-500" />
                  ) : (
                    <Sparkles className="w-4 h-4 text-violet-400" />
                  )}
                </div>
              </div>
            </div>

            {/* Bid Action Button */}
            <button
              onClick={async () => {
                const listing = activeBidListing;
                setActiveBidListing(null);
                await handleBid(
                  listing.id,
                  bidAmount,
                  listing.current_price,
                  getDisplayName(listing),
                  true
                );
              }}
              className="w-full py-3 rounded-2xl bg-amber-500 hover:bg-amber-400 font-serif font-bold text-xs tracking-wider transition-all text-zinc-950 shadow-lg cursor-pointer active:scale-[0.98]"
            >
              Đấu Thầu
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
