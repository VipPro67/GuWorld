import React, { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { useGameStore } from "@/store/useGameStore";
import { ShoppingCart, AlertCircle, Coins, Sparkles, LayoutGrid, List, Search, X, Info } from "lucide-react";
import { getItemIconUrl } from "@/lib/item-icons";
import { getGuIconUrl } from "@/lib/gu-icons";
import { getGuRankColor } from "@/lib/gu-frames";

interface ItemMeta {
  type: string;
  typeName: string;
  name: string;
  rank: number;
  iconUrl: string;
  description: string;
  rankColor: ReturnType<typeof getGuRankColor>;
}

export const ArenaShopView: React.FC = () => {
  const { arenaShopItems, fetchArenaShop, buyArenaItem, actionLoading, player, guDictionary, itemDictionary } = useGameStore();
  const [searchTerm, setSearchTerm] = useState("");
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [selectedBuyItem, setSelectedBuyItem] = useState<ApiResponseData | null>(null);
  const [buyQuantity, setBuyQuantity] = useState(1);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const filteredItems = arenaShopItems.filter((item: ApiResponseData) =>
    item.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  useEffect(() => {
    fetchArenaShop();
  }, []);

  const resolveMeta = (item: ApiResponseData): ItemMeta => {
    const guSpec = guDictionary?.[item.id];
    const itemSpec = itemDictionary?.[item.id];

    if (guSpec) {
      const rank = guSpec.rank || 1;
      return {
        type: "Gu",
        typeName: "Cổ Trùng",
        name: guSpec.name || item.name,
        rank,
        iconUrl: getGuIconUrl(guSpec.id, guSpec.name),
        description: guSpec.description || "Cổ trùng kỳ dị trong thiên địa.",
        rankColor: getGuRankColor(rank),
      };
    }

    if (itemSpec) {
      const rank = itemSpec.rank || 1;
      return {
        type: itemSpec.item_type || "Material",
        typeName: itemSpec.item_type === "Food" ? "Thức Ăn" : itemSpec.item_type === "Consumable" ? "Tiêu Hao" : "Linh Tài",
        name: itemSpec.name || item.name,
        rank,
        iconUrl: getItemIconUrl(itemSpec),
        description: itemSpec.description || "Nguyên liệu tu tiên trân quý.",
        rankColor: getGuRankColor(rank),
      };
    }

    if (item.id.startsWith("recipe_")) {
      return {
        type: "Recipe",
        typeName: "Bí Phương",
        name: item.name,
        rank: 3,
        iconUrl: getItemIconUrl({ id: "recipe", name: "Bí Phương", item_type: "Recipe" }),
        description: "Bí phương phối luyện Cổ trùng thần bí.",
        rankColor: getGuRankColor(3),
      };
    }

    return {
      type: "Item",
      typeName: "Vật Phẩm",
      name: item.name,
      rank: 1,
      iconUrl: getItemIconUrl({ id: item.id, name: item.name }),
      description: "Vật phẩm quy đổi từ Đấu Trường.",
      rankColor: getGuRankColor(1),
    };
  };

  const handleOpenBuyModal = (item: ApiResponseData) => {
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
      setErrorMessage("Số lượng đổi phải lớn hơn 0.");
      return;
    }

    const totalCost = selectedBuyItem.price * buyQuantity;
    const balance = player?.materials_inventory?.arena_coin || 0;
    if (balance < totalCost) {
      setErrorMessage("Không đủ Xu Đấu Trường để thực hiện đổi.");
      return;
    }

    const success = await buyArenaItem(selectedBuyItem.id, buyQuantity);
    if (success) {
      setSuccessMessage("Đổi thành công! Đã thêm vào túi đồ.");
      setTimeout(() => {
        setSelectedBuyItem(null);
      }, 1000);
    } else {
      setErrorMessage("Đổi thất bại. Vui lòng thử lại.");
    }
  };

  const selectedMeta = selectedBuyItem ? resolveMeta(selectedBuyItem) : null;
  const arenaCoinBalance = player?.materials_inventory?.arena_coin || 0;

  return (
    <div className="space-y-4">
      {/* Control bar */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-zinc-950/60 p-3 rounded-2xl border border-zinc-800">
        <div className="relative flex-1">
          <Search className="w-3.5 h-3.5 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Tìm kiếm vật phẩm đổi..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-9 pr-3 py-2 border border-zinc-800 rounded-xl bg-zinc-900 text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-amber-500/40 font-mono text-xs transition-colors"
          />
        </div>

        <div className="flex items-center justify-between sm:justify-end gap-3">
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-400 font-mono font-bold text-xs">
            <Coins className="w-4 h-4 text-amber-500" />
            <span>{arenaCoinBalance.toLocaleString()} Xu</span>
          </div>

          <div className="flex items-center bg-zinc-900 border border-zinc-800 rounded-xl p-0.5">
            <button
              type="button"
              onClick={() => setViewMode("grid")}
              className={`p-1.5 rounded-lg transition-colors cursor-pointer ${
                viewMode === "grid" ? "bg-amber-500/20 text-amber-400 font-bold" : "text-zinc-500 hover:text-zinc-300"
              }`}
              title="Dạng Lưới RPG"
            >
              <LayoutGrid className="w-4 h-4" />
            </button>
            <button
              type="button"
              onClick={() => setViewMode("list")}
              className={`p-1.5 rounded-lg transition-colors cursor-pointer ${
                viewMode === "list" ? "bg-amber-500/20 text-amber-400 font-bold" : "text-zinc-500 hover:text-zinc-300"
              }`}
              title="Dạng Danh Sách"
            >
              <List className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Grid or List View */}
      {filteredItems.length === 0 ? (
        <div className="text-center py-12 text-zinc-500 font-mono text-xs italic border border-zinc-900/50 rounded-2xl bg-zinc-950/20">
          Chưa có vật phẩm nào phù hợp trong cửa hàng...
        </div>
      ) : viewMode === "grid" ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
          {filteredItems.map((item: ApiResponseData) => {
            const meta = resolveMeta(item);
            const canAfford = arenaCoinBalance >= item.price;
            return (
              <div
                key={item.id}
                onClick={() => handleOpenBuyModal(item)}
                className={`group relative rounded-2xl border p-3 flex flex-col justify-between transition-all duration-200 cursor-pointer hover:-translate-y-1 ${meta.rankColor.border} ${meta.rankColor.bgGradient} hover:shadow-lg ${meta.rankColor.glow}`}
              >
                {/* Art & Badges */}
                <div className="relative aspect-square w-full rounded-xl bg-black/40 border border-zinc-800/60 overflow-hidden flex items-center justify-center p-2 mb-2">
                  <span className={`absolute top-1 left-1 px-1.5 py-0.5 rounded text-[9px] font-mono font-bold uppercase border ${meta.rankColor.badge}`}>
                    R{meta.rank}
                  </span>
                  <span className="absolute top-1 right-1 px-1.5 py-0.5 rounded text-[8px] font-mono text-zinc-400 bg-zinc-950/80 border border-zinc-800">
                    {meta.typeName}
                  </span>
                  <img
                    src={meta.iconUrl}
                    alt={meta.name}
                    loading="lazy"
                    decoding="async"
                    className="w-12 h-12 sm:w-14 sm:h-14 object-contain transition-transform group-hover:scale-110 duration-200 drop-shadow-md"
                  />
                  {item.qty > 1 && (
                    <span className="absolute bottom-1 right-1.5 px-1 rounded text-[10px] font-mono font-black text-amber-300 bg-zinc-950/90 border border-amber-500/30">
                      ×{item.qty}
                    </span>
                  )}
                </div>

                {/* Info & Price */}
                <div className="space-y-1">
                  <h4 className={`text-xs font-serif font-bold truncate group-hover:text-amber-300 transition-colors ${meta.rankColor.text}`}>
                    {meta.name}
                  </h4>
                  <div className="flex items-center justify-between text-xs font-mono pt-1 border-t border-zinc-800/40">
                    <span className="flex items-center gap-1 font-bold text-amber-400">
                      <Coins className="w-3.5 h-3.5 text-amber-500 shrink-0" />
                      {item.price}
                    </span>
                    <button
                      type="button"
                      disabled={actionLoading}
                      onClick={(e) => {
                        e.stopPropagation();
                        handleOpenBuyModal(item);
                      }}
                      className={`px-2 py-0.5 rounded-lg text-[10px] font-mono font-bold transition-all ${
                        canAfford
                          ? "bg-amber-500/20 text-amber-300 hover:bg-amber-500 hover:text-black border border-amber-500/30"
                          : "bg-zinc-900 text-zinc-500 border border-zinc-800"
                      }`}
                    >
                      Đổi
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <div className="space-y-2">
          {filteredItems.map((item: ApiResponseData) => {
            const meta = resolveMeta(item);
            const canAfford = arenaCoinBalance >= item.price;
            return (
              <div
                key={item.id}
                onClick={() => handleOpenBuyModal(item)}
                className={`bg-zinc-950/60 border rounded-2xl p-3 flex items-center justify-between gap-3 hover:border-amber-900/50 transition-all group cursor-pointer ${meta.rankColor.border}`}
              >
                <div className="flex items-center gap-3 min-w-0">
                  <div className="w-12 h-12 rounded-xl bg-black/40 border border-zinc-800 flex items-center justify-center p-1.5 shrink-0 relative">
                    <img
                      src={meta.iconUrl}
                      alt={meta.name}
                      loading="lazy"
                      decoding="async"
                      className="w-9 h-9 object-contain"
                    />
                    {item.qty > 1 && (
                      <span className="absolute bottom-0.5 right-1 text-[9px] font-mono font-black text-amber-300">
                        ×{item.qty}
                      </span>
                    )}
                  </div>
                  <div className="min-w-0 space-y-0.5">
                    <div className="flex items-center gap-2">
                      <h4 className="text-sm font-serif font-bold text-zinc-200 group-hover:text-amber-400 truncate">
                        {meta.name}
                      </h4>
                      <span className={`px-1.5 py-0.2 rounded text-[9px] font-mono font-semibold border ${meta.rankColor.badge}`}>
                        R{meta.rank}
                      </span>
                      <span className="text-[9px] text-zinc-500 font-mono">
                        {meta.typeName}
                      </span>
                    </div>
                    <p className="text-xs text-zinc-500 font-mono truncate max-w-md">
                      {meta.description}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-3 shrink-0">
                  <span className="text-xs font-mono font-bold text-amber-400 flex items-center gap-1">
                    <Coins className="w-3.5 h-3.5 text-amber-500" />
                    {item.price}{" "}
                    <span className="text-[10px] text-zinc-500 uppercase tracking-widest">
                      Xu
                    </span>
                  </span>
                  <button
                    type="button"
                    disabled={actionLoading}
                    onClick={(e) => {
                      e.stopPropagation();
                      handleOpenBuyModal(item);
                    }}
                    className={`px-3 py-1.5 rounded-xl text-xs font-bold font-mono transition-all flex items-center gap-1.5 cursor-pointer ${
                      canAfford
                        ? "bg-amber-600 hover:bg-amber-500 text-zinc-950 shadow-md"
                        : "bg-zinc-900 border border-zinc-800 text-zinc-500"
                    }`}
                  >
                    <ShoppingCart className="w-3.5 h-3.5" />
                    Đổi
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Buy & Inspector Modal */}
      {selectedBuyItem && selectedMeta &&
        createPortal(
          <div
            className="mmo-paper-theme fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 animate-fade-in"
            onClick={() => setSelectedBuyItem(null)}
          >
            <div
              className={`bg-zinc-950 border ${selectedMeta.rankColor.border} rounded-3xl p-6 max-w-sm w-full space-y-4 shadow-2xl relative ${selectedMeta.rankColor.glow}`}
              onClick={(e) => e.stopPropagation()}
            >
              <button
                type="button"
                onClick={() => setSelectedBuyItem(null)}
                className="absolute top-4 right-4 p-1.5 text-zinc-500 hover:text-zinc-200 rounded-full hover:bg-zinc-900 transition-colors cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>

              {/* Inspector Header with Art */}
              <div className="flex items-center gap-3 pt-1">
                <div className={`w-16 h-16 rounded-2xl bg-black/60 border ${selectedMeta.rankColor.border} flex items-center justify-center p-2 shrink-0 relative shadow-inner`}>
                  <img
                    src={selectedMeta.iconUrl}
                    alt={selectedMeta.name}
                    className="w-12 h-12 object-contain"
                  />
                  {selectedBuyItem.qty > 1 && (
                    <span className="absolute bottom-1 right-1 text-[10px] font-mono font-black text-amber-300 bg-zinc-950/80 px-1 rounded border border-amber-500/20">
                      ×{selectedBuyItem.qty}
                    </span>
                  )}
                </div>
                <div className="min-w-0">
                  <h4 className="text-base font-bold text-zinc-100 font-serif truncate">
                    {selectedMeta.name}
                  </h4>
                  <div className="flex items-center gap-1.5 mt-1">
                    <span className={`px-2 py-0.5 rounded text-[10px] font-mono font-bold uppercase border ${selectedMeta.rankColor.badge}`}>
                      Hạng {selectedMeta.rank}
                    </span>
                    <span className="px-2 py-0.5 rounded text-[10px] font-mono text-zinc-400 bg-zinc-900 border border-zinc-800">
                      {selectedMeta.typeName}
                    </span>
                  </div>
                </div>
              </div>

              {/* Lore / Description */}
              <div className="bg-zinc-900/40 border border-zinc-800/80 rounded-xl p-3 text-xs text-zinc-400 font-mono leading-relaxed max-h-24 overflow-y-auto">
                <div className="flex items-start gap-1.5 mb-1 text-zinc-500 text-[10px] uppercase font-bold tracking-wider">
                  <Info className="w-3 h-3 text-amber-500/70 shrink-0 mt-0.5" />
                  Mô tả vật phẩm
                </div>
                {selectedMeta.description}
              </div>

              {/* Price & Balance */}
              <div className="bg-zinc-900/60 border border-zinc-800 rounded-xl p-3 space-y-1.5 text-xs font-mono">
                <div className="flex justify-between">
                  <span className="text-zinc-400">Đơn giá:</span>
                  <span className="font-bold flex items-center gap-1 text-amber-400">
                    <Coins className="w-3.5 h-3.5 text-amber-500" />
                    {selectedBuyItem.price} Xu
                  </span>
                </div>
                <div className="flex justify-between border-t border-zinc-800/60 pt-1.5">
                  <span className="text-zinc-400">Hiện có:</span>
                  <span className="text-emerald-400 font-bold flex items-center gap-1">
                    <Coins className="w-3.5 h-3.5 text-emerald-500" />
                    {arenaCoinBalance.toLocaleString()} Xu
                  </span>
                </div>
              </div>

              {/* Quantity Stepper */}
              <div className="space-y-2">
                <div className="flex justify-between items-center text-[10px] font-mono text-zinc-500 uppercase">
                  <span>Số lượng cần đổi:</span>
                  <span>
                    {selectedBuyItem.price > 0 &&
                      `(Tối đa: ${Math.floor(arenaCoinBalance / selectedBuyItem.price).toLocaleString()})`}
                  </span>
                </div>
                <div className="flex items-center justify-between border border-zinc-800 rounded-xl bg-zinc-900/40 overflow-hidden px-1">
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
                    value={buyQuantity || ""}
                    onChange={(e) => {
                      const val = parseInt(e.target.value, 10);
                      if (isNaN(val) || val <= 0) {
                        setBuyQuantity(0);
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
                    onClick={() => setBuyQuantity((prev) => (prev || 0) + 1)}
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
                      onClick={() => setBuyQuantity((prev) => (prev || 0) + step)}
                      className="flex-1 py-1 text-[11px] font-mono text-zinc-400 bg-zinc-900/50 hover:bg-zinc-900 border border-zinc-800 hover:border-zinc-700 rounded-lg transition-all cursor-pointer"
                    >
                      +{step}
                    </button>
                  ))}
                  <button
                    type="button"
                    onClick={() => {
                      const affordable =
                        selectedBuyItem.price > 0
                          ? Math.floor(arenaCoinBalance / selectedBuyItem.price)
                          : 999;
                      setBuyQuantity(Math.max(1, affordable));
                    }}
                    className="flex-1 py-1 text-[11px] font-mono text-amber-400 bg-amber-500/10 hover:bg-amber-500/20 border border-amber-500/20 hover:border-amber-500/30 rounded-lg transition-all cursor-pointer font-semibold"
                  >
                    Tối đa
                  </button>
                </div>
              </div>

              {/* Total Cost Display */}
              <div className="flex justify-between items-center pt-2 font-mono border-t border-dashed border-zinc-800 text-xs">
                <span className="text-zinc-400">Tổng thanh toán:</span>
                <span className="font-bold flex items-center gap-1 text-sm text-amber-400">
                  <Coins className="w-4 h-4 text-amber-500" />
                  {(selectedBuyItem.price * (buyQuantity || 0)).toLocaleString()} Xu
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
                  className="py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 text-zinc-950 font-bold text-xs active:scale-[0.98] transition-all cursor-pointer shadow-lg"
                >
                  Xác Nhận Đổi
                </button>
              </div>
            </div>
          </div>,
          document.body
        )}
    </div>
  );
};
