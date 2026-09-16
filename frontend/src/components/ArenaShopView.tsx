import React, { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { useGameStore } from "@/store/useGameStore";
import { ShoppingCart, AlertCircle, Coins, Sparkles } from "lucide-react";

export const ArenaShopView: React.FC = () => {
  const { arenaShopItems, fetchArenaShop, buyArenaItem, actionLoading, player } = useGameStore();
  const [searchTerm, setSearchTerm] = useState("");
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

  return (
    <div className="space-y-4">
      <input
        type="text"
        placeholder="Tìm kiếm vật phẩm..."
        value={searchTerm}
        onChange={(e) => setSearchTerm(e.target.value)}
        className="w-full p-2.5 border border-zinc-800 rounded-xl bg-zinc-900 text-zinc-300 placeholder-zinc-650 focus:outline-none focus:border-amber-500/40 font-mono text-xs transition-colors"
      />
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
        {filteredItems.map((item: ApiResponseData) => (
          <div
            key={item.id}
            className="bg-zinc-950/60 border border-zinc-800 rounded-2xl p-4 flex flex-col justify-between items-start hover:border-amber-900/50 transition-all group"
          >
            <div className="space-y-1">
              <h4 className="text-amber-500 font-bold font-serif text-sm group-hover:text-amber-400 transition-colors">
                {item.name}
              </h4>
              <p className="text-xs text-zinc-500 font-mono">
                Số lượng: {item.qty}
              </p>
            </div>

            <div className="mt-4 w-full flex items-center justify-between border-t border-zinc-900/60 pt-3">
              <span className="text-xs font-mono font-bold text-amber-500 flex items-center gap-1">
                {item.price}{" "}
                <span className="text-[10px] text-zinc-500 uppercase tracking-widest">
                  Xu
                </span>
              </span>
              <button
                type="button"
                disabled={actionLoading}
                onClick={() => handleOpenBuyModal(item)}
                className="bg-zinc-900 hover:bg-amber-955/60 border border-zinc-800 hover:border-amber-900 text-zinc-300 hover:text-amber-500 px-3 py-1.5 rounded-xl text-xs font-bold font-mono transition-all flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
              >
                <ShoppingCart className="w-3.5 h-3.5" />
                Đổi
              </button>
            </div>
          </div>
        ))}
        {arenaShopItems.length === 0 && (
          <div className="col-span-full text-center py-8 text-zinc-500 font-mono text-xs italic border border-zinc-900/50 rounded-2xl bg-zinc-950/20">
            Chưa có vật phẩm nào trong shop...
          </div>
        )}
      </div>

      {/* Buy Quantity Modal */}
      {selectedBuyItem &&
        createPortal(
          <div
            className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50 animate-fade-in"
            onClick={() => setSelectedBuyItem(null)}
          >
            <div
              className="bg-zinc-950 border border-zinc-850 rounded-3xl p-6 max-w-sm w-full space-y-4"
              onClick={(e) => e.stopPropagation()}
            >
              <div>
                <h4 className="text-lg font-bold text-zinc-100 font-serif">
                  Quyết Định Đổi Vật Phẩm
                </h4>
                <p className="text-xs text-zinc-400 mt-1">
                  Đổi {selectedBuyItem.name} từ Cửa hàng Võ Đài.
                </p>
              </div>

              <div className="bg-zinc-900/40 border border-zinc-850 rounded-xl p-3 space-y-1.5 text-xs font-mono">
                <div className="flex justify-between">
                  <span className="text-zinc-400">Đơn giá:</span>
                  <span className="font-bold flex items-center gap-1 text-amber-400">
                    <Coins className="w-3.5 h-3.5 text-amber-500" />
                    {selectedBuyItem.price} Xu
                  </span>
                </div>
                <div className="flex justify-between border-t border-zinc-800/40 pt-1.5">
                  <span className="text-zinc-400">Bạn có:</span>
                  <span className="text-emerald-550 font-bold flex items-center gap-1">
                    <Coins className="w-3.5 h-3.5 text-emerald-500" />
                    {(player?.materials_inventory?.arena_coin || 0).toLocaleString()} Xu
                  </span>
                </div>
              </div>

              <div className="space-y-2">
                <div className="flex justify-between items-center text-[10px] font-mono text-zinc-500 uppercase">
                  <span>Số lượng cần đổi:</span>
                  <span>
                    {selectedBuyItem.price > 0 &&
                      `(Có thể đổi: ${Math.floor(
                        (player?.materials_inventory?.arena_coin || 0) /
                          selectedBuyItem.price
                      ).toLocaleString()})`}
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
                      className="flex-1 py-1 text-[11px] font-mono text-zinc-400 bg-zinc-900/50 hover:bg-zinc-900 border border-zinc-850 hover:border-zinc-700 rounded-lg transition-all cursor-pointer"
                    >
                      +{step}
                    </button>
                  ))}
                  <button
                    type="button"
                    onClick={() => {
                      const affordable =
                        selectedBuyItem.price > 0
                          ? Math.floor(
                              (player?.materials_inventory?.arena_coin || 0) /
                                selectedBuyItem.price
                            )
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
              <div className="flex justify-between items-center pt-2 font-mono border-t border-dashed border-zinc-900 text-xs">
                <span className="text-zinc-400">Tổng thanh toán:</span>
                <span className="font-bold flex items-center gap-1 text-sm text-amber-400">
                  <Coins className="w-4 h-4 text-amber-500" />
                  {(selectedBuyItem.price * (buyQuantity || 0)).toLocaleString()} Xu Đấu Trường
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
