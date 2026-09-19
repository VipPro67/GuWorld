"use client";

import { useEffect, useState } from "react";
import { GuInstance, Player, GuSpec, ItemSpec, useGameStore, guSpecFor, BuybackOrder } from "@/store/useGameStore";
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
  Sparkles,
  RefreshCw,
  ArrowRightLeft,
  CheckCircle,
  XCircle,
  Package,
} from "lucide-react";
import { GamePanel, GameTabs, GameButton, GameCard, GameInput } from "@/components/ui";

interface RecipeData {
  recipe_id: string;
  completion_rate: number;
}

interface Listing {
  id: string;
  seller_id: string;
  item_type: "Gu" | "Material" | "Recipe";
  item_data: string | RecipeData | ApiResponseData[];
  quantity: number;
  currency: "primeval" | "immortal";
  start_price: number;
  current_price: number;
  highest_bidder_id: string;
  end_time: string;
  listing_type: "MARKET" | "AUCTION" | "TRADE";
  target_buyer_id?: string;
  target_buyer_username?: string;
  created_at?: string;
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
  actionLoading,
  listings,
  onRefresh,
  onCreate,
  onBid,
  itemDictionary,
}: Props) {
  const [tab, setTab] = useState<"orderBook" | "asyncTrade" | "auction" | "create">("orderBook");
  const [orderBookFilter, setOrderBookFilter] = useState<"all" | "BUY" | "SELL">("all");

  const buybackOrders = useGameStore((s) => s.buybackOrders);
  const tradeOffers = useGameStore((s) => s.tradeOffers);
  const fetchBuybackOrders = useGameStore((s) => s.fetchBuybackOrders);
  const createBuybackOrder = useGameStore((s) => s.createBuybackOrder);
  const sellToBuybackOrder = useGameStore((s) => s.sellToBuybackOrder);
  const cancelBuybackOrder = useGameStore((s) => s.cancelBuybackOrder);

  const fetchTradeOffers = useGameStore((s) => s.fetchTradeOffers);
  const createTradeOffer = useGameStore((s) => s.createTradeOffer);
  const acceptTradeOffer = useGameStore((s) => s.acceptTradeOffer);
  const cancelTradeOffer = useGameStore((s) => s.cancelTradeOffer);

  // Form states for Order Creation
  const [createMode, setCreateMode] = useState<"buyback" | "sellOrder" | "tradeOffer" | "auction">("buyback");
  
  // Buyback Order Creation Form State
  const [bbItemType, setBbItemType] = useState<"Gu" | "Material" | "Recipe">("Material");
  const [bbTargetMat, setBbTargetMat] = useState("");
  const [bbTargetGu, setBbTargetGu] = useState("");
  const [bbTargetRank, setBbTargetRank] = useState(1);
  const [bbUnitPrice, setBbUnitPrice] = useState(10);
  const [bbQuantity, setBbQuantity] = useState(1);
  const [bbCurrency, setBbCurrency] = useState<"primeval" | "immortal">("primeval");
  const [bbDuration, setBbDuration] = useState(72);

  // Sell Order Creation Form State
  const [sellItemType, setSellItemType] = useState<"Gu" | "Material">("Material");
  const [sellMatID, setSellMatID] = useState("");
  const [sellGuInstID, setSellGuInstID] = useState("");
  const [sellUnitPrice, setSellUnitPrice] = useState(10);
  const [sellQuantity, setSellQuantity] = useState(1);
  const [sellCurrency, setSellCurrency] = useState<"primeval" | "immortal">("primeval");
  const [sellDuration, setSellDuration] = useState(72);

  // Modal state for Buy/Sell Interaction
  const [selectedBuybackOrder, setSelectedBuybackOrder] = useState<BuybackOrder | null>(null);
  const [sellQtyInput, setSellQtyInput] = useState(1);

  // Form state for Async Trade Offer Creation
  const [offeredStones, setOfferedStones] = useState(0);
  const [offeredImmortal, setOfferedImmortal] = useState(0);
  const [offeredGuIDs] = useState<string[]>([]);
  const [offeredMats] = useState<Record<string, number>>({});

  const [requestedStones, setRequestedStones] = useState(0);
  const [requestedImmortal, setRequestedImmortal] = useState(0);
  const [requestedGuIDs] = useState<string[]>([]);
  const [requestedMats] = useState<Record<string, number>>({});
  const [tradeOfferDuration] = useState(72);

  // Auction form state
  const [aucItemType] = useState<"Gu" | "Material" | "Recipe">("Material");
  const [aucPrice, setAucPrice] = useState(10);
  const [aucDuration] = useState(24);
  const [aucCurrency] = useState<"primeval" | "immortal">("primeval");
  const [selectedAucGu, setSelectedAucGu] = useState("");
  const [selectedAucMat] = useState("");

  const [searchQuery, setSearchQuery] = useState("");

  useEffect(() => {
    fetchBuybackOrders();
    fetchTradeOffers();
    onRefresh();
  }, [fetchBuybackOrders, fetchTradeOffers, onRefresh]);

  const hasConnectingHeaven = Array.isArray(guInventory) && guInventory.some(
    (g) => g.gu_id === "GU_CONNECTING_HEAVEN" && g.status !== "DEAD"
  );
  const hasDivineSense = Array.isArray(guInventory) && guInventory.some(
    (g) => g.gu_id === "GU_DIVINE_SENSE" && g.status !== "DEAD"
  );

  if (!hasConnectingHeaven || !hasDivineSense) {
    return (
      <div className="w-full mx-auto glass-panel-gold p-6 rounded-3xl border border-amber-500/20 shadow-2xl relative overflow-hidden">
        <div className="absolute top-0 right-0 w-64 h-64 bg-amber-500/5 rounded-full blur-[100px] pointer-events-none" />
        <div className="absolute bottom-0 left-0 w-80 h-80 bg-cyan-500/5 rounded-full blur-[120px] pointer-events-none" />
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <AlertCircle className="w-16 h-16 text-amber-500 mb-6 animate-pulse" />
          <h3 className="text-xl font-serif font-bold text-amber-200 mb-2">Chưa Thể Kết Nối Bảo Hoàng Thiên</h3>
          <p className="text-sm text-zinc-400 font-serif max-w-md leading-relaxed">
            Đạo hữu cần sở hữu đồng thời <strong className="text-amber-300">Thông Thiên Cổ (`GU_CONNECTING_HEAVEN`)</strong> VÀ <strong className="text-amber-300">Thần Niệm Cổ (`GU_DIVINE_SENSE`)</strong> còn sống trong túi đồ để mở kết nối Thần Niệm với Bảo Hoàng Thiên.
          </p>
        </div>
      </div>
    );
  }

  const handleCreateBuybackSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    let targetPayload: string | { gu_id: string; rank: number } = "";
    if (bbItemType === "Material") {
      if (!bbTargetMat) return;
      targetPayload = bbTargetMat;
    } else if (bbItemType === "Gu") {
      if (!bbTargetGu) return;
      targetPayload = { gu_id: bbTargetGu, rank: bbTargetRank };
    }
    await createBuybackOrder({
      orderType: "BUY",
      itemType: bbItemType,
      itemTarget: targetPayload,
      unitPrice: bbUnitPrice,
      quantityRequested: bbQuantity,
      currency: bbCurrency,
      durationHours: bbDuration,
    });
    setTab("orderBook");
  };

  const handleCreateSellOrderSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    let targetPayload: string | { gu_id: string; rank: number } = "";
    let guInstIDs: string[] = [];
    if (sellItemType === "Material") {
      if (!sellMatID) return;
      targetPayload = sellMatID;
    } else if (sellItemType === "Gu") {
      if (!sellGuInstID) return;
      const g = guInventory.find((inst) => inst.instance_id === sellGuInstID);
      if (!g) return;
      targetPayload = { gu_id: g.gu_id, rank: g.rank };
      guInstIDs = [sellGuInstID];
    }
    await createBuybackOrder({
      orderType: "SELL",
      itemType: sellItemType,
      itemTarget: targetPayload,
      guInstanceIds: guInstIDs,
      unitPrice: sellUnitPrice,
      quantityRequested: sellQuantity,
      currency: sellCurrency,
      durationHours: sellDuration,
    });
    setTab("orderBook");
  };

  const handleCreateTradeOfferSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await createTradeOffer({
      offered: {
        primeval_stones: offeredStones,
        immortal_stones: offeredImmortal,
        gus: offeredGuIDs,
        materials: offeredMats,
        recipes: {},
      },
      requested: {
        primeval_stones: requestedStones,
        immortal_stones: requestedImmortal,
        gus: requestedGuIDs,
        materials: requestedMats,
        recipes: {},
      },
      durationHours: tradeOfferDuration,
    });
    setTab("asyncTrade");
  };

  const handleCreateAuctionSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (aucItemType === "Gu" && !selectedAucGu) return;
    if (aucItemType === "Material" && !selectedAucMat) return;
    await onCreate({
      itemType: aucItemType,
      guInstanceId: selectedAucGu,
      itemId: selectedAucMat,
      quantity: 1,
      price: aucPrice,
      durationHours: aucDuration,
      currency: aucCurrency,
      listingType: "AUCTION",
    });
    setTab("auction");
  };

  const activeAuctionListings = listings.filter((l) => l.listing_type === "AUCTION");

  const filteredOrderBook = buybackOrders.filter((o) => {
    const orderType = o.order_type || "BUY";
    if (orderBookFilter !== "all" && orderType !== orderBookFilter) return false;
    return true;
  });

  const getListingDetails = (listing: Listing): { title: string; subtitle?: string } => {
    let parsedData: Record<string, unknown> | string | null = listing.item_data as Record<string, unknown> | string | null;
    if (typeof parsedData === "string") {
      try {
        parsedData = JSON.parse(parsedData);
      } catch {
        // raw material id
      }
    }

    if (listing.item_type === "Gu") {
      const gus: GuInstance[] = Array.isArray(parsedData) ? parsedData : [];
      if (gus.length > 0) {
        const g = gus[0];
        const spec = guSpecFor(g.gu_id, g.rank);
        const name = spec?.name || g.gu_id;
        return {
          title: `${name} (Hạng ${g.rank})`,
          subtitle: `Độ no: ${g.hunger}%`,
        };
      }
      return { title: "Cổ Trùng" };
    } else if (listing.item_type === "Material") {
      const matId = typeof parsedData === "string" ? parsedData : String(parsedData);
      const spec = itemDictionary[matId];
      const name = spec?.name || matId;
      return {
        title: name,
        subtitle: `Số lượng: ${listing.quantity}`,
      };
    } else if (listing.item_type === "Recipe") {
      const recObj = (parsedData && typeof parsedData === "object") ? (parsedData as Record<string, unknown>) : null;
      const recId = (typeof recObj?.recipe_id === "string" ? recObj.recipe_id : "") || (typeof parsedData === "string" ? parsedData : "");
      const completionRate = typeof recObj?.completion_rate === "number" || typeof recObj?.completion_rate === "string" ? recObj.completion_rate : null;
      return {
        title: `Bí phương ${recId}`,
        subtitle: completionRate !== null ? `Hoàn thiện: ${completionRate}%` : undefined,
      };
    }
    return { title: "Bảo Vật" };
  };

  return (
    <GamePanel variant="gold" className="rounded-3xl p-3 sm:p-6 border border-amber-500/20 shadow-2xl relative overflow-hidden">
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4 border-b border-amber-500/10 pb-5 mb-6">
        <div>
          <h2 className="text-2xl font-bold font-serif text-transparent bg-clip-text bg-gradient-to-r from-amber-200 via-amber-300 to-amber-500 tracking-wider flex items-center gap-2">
            <Sparkles className="w-6 h-6 text-amber-400 animate-pulse" />
            BẢO HOÀNG THIÊN
          </h2>
          <p className="text-xs text-zinc-500 font-mono mt-1">
            Chợ Giao Thương Cổ Tiên • Đơn Rao Bán - Đơn Thu Mua - Trao Đổi Auto - Đấu Giá
          </p>
        </div>

        <div className="flex items-center gap-3 bg-zinc-950/60 px-4 py-2.5 rounded-2xl border border-zinc-800/80 font-mono text-xs">
          <div className="flex items-center gap-1.5 pr-3 border-r border-zinc-850">
            <Coins className="w-4 h-4 text-amber-500" />
            <span className="text-zinc-400">Nguyên Thạch:</span>
            <span className="text-amber-400 font-bold text-sm">{player.primeval_stones}</span>
          </div>
          <div className="flex items-center gap-1.5">
            <Sparkles className="w-4 h-4 text-cyan-400" />
            <span className="text-zinc-400">Tiên Nguyên:</span>
            <span className="text-cyan-400 font-bold text-sm">{player.immortal_stones}</span>
          </div>
          <GameButton
            size="xs"
            variant="ghost"
            onClick={() => {
              fetchBuybackOrders();
              fetchTradeOffers();
              onRefresh();
            }}
            loading={actionLoading}
            leftIcon={<RefreshCw className="w-3.5 h-3.5" />}
          >
            Làm Mới
          </GameButton>
        </div>
      </div>

      <GameTabs
        activeId={tab}
        onChange={(id) => setTab(id as "orderBook" | "asyncTrade" | "auction" | "create")}
        items={[
          { id: "orderBook", label: "Chợ Lệnh (Order Book)", icon: <ShoppingCart className="w-4 h-4" /> },
          { id: "asyncTrade", label: "Trao Đổi Auto", icon: <ArrowRightLeft className="w-4 h-4" /> },
          { id: "auction", label: "Đấu Giá Bảo Vật", icon: <Gavel className="w-4 h-4" /> },
          { id: "create", label: "Đăng Đơn Mới", icon: <PlusCircle className="w-4 h-4" /> },
        ]}
      />

      <div className="mt-6">
        {/* TAB 1: CHỢ LỆNH (ORDER BOOK - RAO BÁN & THU MUA) */}
        {tab === "orderBook" && (
          <div className="space-y-4">
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-2">
                <GameButton
                  size="xs"
                  variant={orderBookFilter === "all" ? "primary" : "ghost"}
                  onClick={() => setOrderBookFilter("all")}
                >
                  Tất Cả ({buybackOrders.length})
                </GameButton>
                <GameButton
                  size="xs"
                  variant={orderBookFilter === "BUY" ? "primary" : "ghost"}
                  onClick={() => setOrderBookFilter("BUY")}
                >
                  Đơn Thu Mua ({buybackOrders.filter((o) => (o.order_type || "BUY") === "BUY").length})
                </GameButton>
                <GameButton
                  size="xs"
                  variant={orderBookFilter === "SELL" ? "primary" : "ghost"}
                  onClick={() => setOrderBookFilter("SELL")}
                >
                  Đơn Rao Bán ({buybackOrders.filter((o) => o.order_type === "SELL").length})
                </GameButton>
              </div>

              <div className="relative w-64">
                <Search className="w-4 h-4 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2" />
                <GameInput
                  placeholder="Tìm kiếm trong chợ lệnh..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9 text-xs"
                />
              </div>
            </div>

            {filteredOrderBook.length === 0 ? (
              <div className="text-center py-16 bg-zinc-950/40 rounded-2xl border border-zinc-900 text-zinc-500 font-serif">
                Chưa có đơn rao bán hay thu mua nào khớp với bộ lọc. Hãy tạo đơn mới!
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {filteredOrderBook.map((order) => {
                  const isSellOrder = order.order_type === "SELL";
                  let targetName = "Vật phẩm";
                  if (order.item_type === "Material" && typeof order.item_target === "string") {
                    targetName = itemDictionary[order.item_target]?.name || order.item_target;
                  } else if (order.item_type === "Gu" && typeof order.item_target === "object" && order.item_target !== null) {
                    const t = order.item_target as { gu_id?: string; rank?: number };
                    if (t.gu_id) {
                      targetName = guSpecFor(t.gu_id, t.rank || 1)?.name || t.gu_id;
                    }
                  }
                  const isMine = order.buyer_id === player.id;
                  const remaining = order.quantity_requested - order.quantity_filled;

                  return (
                    <GameCard key={order.id} variant="vital" className="p-4 space-y-3 relative overflow-hidden">
                      <div className="flex items-start justify-between">
                        <div>
                          <span
                            className={`text-[10px] font-mono uppercase px-2 py-0.5 rounded border ${
                              isSellOrder
                                ? "bg-cyan-500/10 text-cyan-400 border-cyan-500/20"
                                : "bg-amber-500/10 text-amber-400 border-amber-500/20"
                            }`}
                          >
                            {isSellOrder ? "Rao Bán" : "Thu Mua"}: {order.item_type}
                          </span>
                          <h4 className="text-base font-bold font-serif text-amber-100 mt-1">{targetName}</h4>
                          <p className="text-xs text-zinc-500 font-mono">
                            Chủ đơn: {order.buyer_username}
                          </p>
                        </div>
                        <div className="text-right">
                          <div className="text-sm font-bold text-amber-400 font-mono">
                            {order.unit_price} {order.currency === "immortal" ? "Tiên Nguyên" : "Nguyên Thạch"}
                          </div>
                          <span className="text-[10px] text-zinc-500">/ 1 món</span>
                        </div>
                      </div>

                      <div className="bg-zinc-950/60 p-2.5 rounded-xl border border-zinc-800/60 space-y-1 font-mono text-xs">
                        <div className="flex justify-between text-zinc-400">
                          <span>{isSellOrder ? "Đã bán:" : "Tiến độ thu mua:"}</span>
                          <span className="text-amber-300 font-bold">
                            {order.quantity_filled} / {order.quantity_requested} (Còn {remaining})
                          </span>
                        </div>
                        <div className="w-full bg-zinc-900 rounded-full h-1.5 overflow-hidden">
                          <div
                            className={`h-full rounded-full transition-all ${isSellOrder ? "bg-cyan-400" : "bg-amber-400"}`}
                            style={{ width: `${Math.min(100, (order.quantity_filled / order.quantity_requested) * 100)}%` }}
                          />
                        </div>
                      </div>

                      <div className="flex items-center justify-between pt-2">
                        <span className="text-[10px] text-zinc-500 flex items-center gap-1 font-mono">
                          <Clock className="w-3 h-3 text-zinc-400" />
                          Hết hạn: {new Date(order.end_time).toLocaleDateString()}
                        </span>
                        {isMine ? (
                          <GameButton
                            size="xs"
                            variant="danger"
                            onClick={() => cancelBuybackOrder(order.id)}
                            loading={actionLoading}
                            leftIcon={<Trash2 className="w-3.5 h-3.5" />}
                          >
                            Hủy Đơn
                          </GameButton>
                        ) : isSellOrder ? (
                          <GameButton
                            size="xs"
                            variant="primary"
                            onClick={() => {
                              setSelectedBuybackOrder(order);
                              setSellQtyInput(1);
                            }}
                            loading={actionLoading}
                            leftIcon={<ShoppingCart className="w-3.5 h-3.5" />}
                          >
                            Mua Ngay
                          </GameButton>
                        ) : (
                          <GameButton
                            size="xs"
                            variant="primary"
                            onClick={() => {
                              setSelectedBuybackOrder(order);
                              setSellQtyInput(1);
                            }}
                            loading={actionLoading}
                            leftIcon={<Tag className="w-3.5 h-3.5" />}
                          >
                            Bán Ngay
                          </GameButton>
                        )}
                      </div>
                    </GameCard>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* TAB 2: TRAO ĐỔI AUTO (ASYNC TRADE OFFERS) */}
        {tab === "asyncTrade" && (
          <div className="space-y-4">
            {tradeOffers.length === 0 ? (
              <div className="text-center py-16 bg-zinc-950/40 rounded-2xl border border-zinc-900 text-zinc-500 font-serif">
                Chưa có đơn trao đổi tự động nào. Hãy đăng đơn trao đổi mới!
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {tradeOffers.map((offer) => {
                  const isMine = offer.creator_id === player.id;
                  return (
                    <GameCard key={offer.id} variant="vital" className="p-4 space-y-4">
                      <div className="flex items-center justify-between border-b border-zinc-800 pb-2">
                        <div className="flex items-center gap-2">
                          <ArrowRightLeft className="w-4 h-4 text-cyan-400" />
                          <span className="text-sm font-bold font-serif text-zinc-200">{offer.creator_username}</span>
                        </div>
                        <span className="text-xs text-zinc-500 font-mono">
                          Thời hạn: {new Date(offer.end_time).toLocaleDateString()}
                        </span>
                      </div>

                      <div className="grid grid-cols-2 gap-3 text-xs font-mono">
                        <div className="bg-emerald-950/20 p-3 rounded-xl border border-emerald-500/20 space-y-1">
                          <span className="text-emerald-400 font-bold text-[11px] block border-b border-emerald-500/20 pb-1 mb-1">
                            VẬT ĐƯA ĐI (OFFERED)
                          </span>
                          {offer.offered.primeval_stones > 0 && <div>• {offer.offered.primeval_stones} Nguyên Thạch</div>}
                          {offer.offered.immortal_stones > 0 && <div>• {offer.offered.immortal_stones} Tiên Nguyên</div>}
                          {Object.entries(offer.offered.materials || {}).map(([k, v]) => (
                            <div key={k}>• {v}x {itemDictionary[k]?.name || k}</div>
                          ))}
                        </div>

                        <div className="bg-amber-950/20 p-3 rounded-xl border border-amber-500/20 space-y-1">
                          <span className="text-amber-400 font-bold text-[11px] block border-b border-amber-500/20 pb-1 mb-1">
                            VẬT YÊU CẦU (REQUESTED)
                          </span>
                          {offer.requested.primeval_stones > 0 && <div>• {offer.requested.primeval_stones} Nguyên Thạch</div>}
                          {offer.requested.immortal_stones > 0 && <div>• {offer.requested.immortal_stones} Tiên Nguyên</div>}
                          {Object.entries(offer.requested.materials || {}).map(([k, v]) => (
                            <div key={k}>• {v}x {itemDictionary[k]?.name || k}</div>
                          ))}
                        </div>
                      </div>

                      <div className="flex justify-end pt-1">
                        {isMine ? (
                          <GameButton
                            size="xs"
                            variant="danger"
                            onClick={() => cancelTradeOffer(offer.id)}
                            loading={actionLoading}
                            leftIcon={<XCircle className="w-3.5 h-3.5" />}
                          >
                            Hủy Trao Đổi
                          </GameButton>
                        ) : (
                          <GameButton
                            size="xs"
                            variant="primary"
                            onClick={() => acceptTradeOffer(offer.id)}
                            loading={actionLoading}
                            leftIcon={<CheckCircle className="w-3.5 h-3.5" />}
                          >
                            Chấp Nhận Trao Đổi (Exact Match)
                          </GameButton>
                        )}
                      </div>
                    </GameCard>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* TAB 3: ĐẤU GIÁ (AUCTION) */}
        {tab === "auction" && (
          <div className="space-y-4">
            {activeAuctionListings.length === 0 ? (
              <div className="text-center py-16 bg-zinc-950/40 rounded-2xl border border-zinc-900 text-zinc-500 font-serif">
                Hiện không có bảo vật nào đang được đấu giá.
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {activeAuctionListings.map((listing) => {
                  const details = getListingDetails(listing);
                  return (
                    <GameCard key={listing.id} variant="vital" className="p-4 space-y-3">
                      <div className="flex items-start justify-between">
                        <div>
                          <span className="text-[10px] font-mono uppercase px-2 py-0.5 rounded bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
                            Đấu Giá: {listing.item_type}
                          </span>
                          <h4 className="text-base font-bold font-serif text-amber-100 mt-1">{details.title}</h4>
                          {details.subtitle && (
                            <p className="text-xs text-zinc-400 font-mono mt-0.5">{details.subtitle}</p>
                          )}
                          <p className="text-[11px] text-zinc-500 font-mono mt-1">
                            Người bán: {listing.target_buyer_username || listing.seller_id.slice(0, 8)}
                          </p>
                        </div>
                        <div className="text-right">
                          <div className="text-sm font-bold text-amber-400 font-mono">
                            {listing.current_price} {listing.currency === "immortal" ? "Tiên Nguyên" : "Nguyên Thạch"}
                          </div>
                          <span className="text-[10px] text-zinc-500">Giá hiện tại</span>
                        </div>
                      </div>

                      <div className="flex items-center justify-between pt-2">
                        <span className="text-[10px] text-zinc-500 flex items-center gap-1 font-mono">
                          <Clock className="w-3 h-3 text-zinc-400" />
                          Hết hạn: {new Date(listing.end_time).toLocaleDateString()}
                        </span>
                        <GameButton
                          size="xs"
                          variant="primary"
                          onClick={() => onBid(listing.id, listing.current_price + 10)}
                          loading={actionLoading}
                        >
                          Đặt Giá (+10)
                        </GameButton>
                      </div>
                    </GameCard>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {/* TAB 4: ĐĂNG ĐƠN MỚI */}
        {tab === "create" && (
          <GameCard variant="vital" className="p-6 max-w-2xl mx-auto space-y-6">
            <div className="flex items-center gap-2 border-b border-zinc-800 pb-4 overflow-x-auto">
              <GameButton
                size="sm"
                variant={createMode === "buyback" ? "primary" : "ghost"}
                onClick={() => setCreateMode("buyback")}
              >
                Tạo Đơn Thu Mua
              </GameButton>
              <GameButton
                size="sm"
                variant={createMode === "sellOrder" ? "primary" : "ghost"}
                onClick={() => setCreateMode("sellOrder")}
              >
                Đăng Đơn Bán
              </GameButton>
              <GameButton
                size="sm"
                variant={createMode === "tradeOffer" ? "primary" : "ghost"}
                onClick={() => setCreateMode("tradeOffer")}
              >
                Đăng Trao Đổi Auto
              </GameButton>
              <GameButton
                size="sm"
                variant={createMode === "auction" ? "primary" : "ghost"}
                onClick={() => setCreateMode("auction")}
              >
                Đăng Đấu Giá
              </GameButton>
            </div>

            {/* FORM 1: BUYBACK ORDER (THU MUA) */}
            {createMode === "buyback" && (
              <form onSubmit={handleCreateBuybackSubmit} className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Loại Vật Phẩm Cần Mua</label>
                    <select
                      value={bbItemType}
                      onChange={(e) => setBbItemType(e.target.value as "Gu" | "Material" | "Recipe")}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="Material">Nguyên Liệu / Tiên Tài</option>
                      <option value="Gu">Cổ Trùng</option>
                    </select>
                  </div>
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Đơn Vị Tiền Trả</label>
                    <select
                      value={bbCurrency}
                      onChange={(e) => setBbCurrency(e.target.value as "primeval" | "immortal")}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="primeval">Nguyên Thạch</option>
                      <option value="immortal">Tiên Nguyên Thạch</option>
                    </select>
                  </div>
                </div>

                {bbItemType === "Material" && (
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Chọn Nguyên Liệu Cần Mua</label>
                    <select
                      value={bbTargetMat}
                      onChange={(e) => setBbTargetMat(e.target.value)}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="">-- Chọn Nguyên Liệu --</option>
                      {Object.entries(itemDictionary).map(([k, v]) => (
                        <option key={k} value={k}>{v.name} ({k})</option>
                      ))}
                    </select>
                  </div>
                )}

                {bbItemType === "Gu" && (
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="text-xs text-zinc-400 block mb-1">Mã Cổ Trùng (Gu ID)</label>
                      <GameInput
                        placeholder="VD: GU_DIVINE_SENSE"
                        value={bbTargetGu}
                        onChange={(e) => setBbTargetGu(e.target.value)}
                      />
                    </div>
                    <div>
                      <label className="text-xs text-zinc-400 block mb-1">Chuyển (Hạng)</label>
                      <GameInput
                        type="number"
                        min={1}
                        max={9}
                        value={bbTargetRank}
                        onChange={(e) => setBbTargetRank(Number(e.target.value))}
                      />
                    </div>
                  </div>
                )}

                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Đơn Giá Mua</label>
                    <GameInput type="number" min={1} value={bbUnitPrice} onChange={(e) => setBbUnitPrice(Number(e.target.value))} />
                  </div>
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Số Lượng Cần Mua</label>
                    <GameInput type="number" min={1} value={bbQuantity} onChange={(e) => setBbQuantity(Number(e.target.value))} />
                  </div>
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Thời Hạn (Giờ)</label>
                    <GameInput type="number" min={1} max={168} value={bbDuration} onChange={(e) => setBbDuration(Number(e.target.value))} />
                  </div>
                </div>

                <div className="bg-zinc-950 p-3 rounded-xl border border-zinc-800 text-xs font-mono text-zinc-400 space-y-1">
                  <div>• Tổng giá trị đơn: <strong className="text-amber-400">{bbUnitPrice * bbQuantity} {bbCurrency}</strong></div>
                  <div>• Phí niêm yết (4% trả trước): <strong className="text-amber-400">{(bbUnitPrice * bbQuantity * 0.04).toFixed(3)} {bbCurrency}</strong></div>
                  <div>• Tổng tiền cọc bị khóa: <strong className="text-amber-300">{(bbUnitPrice * bbQuantity * 1.04).toFixed(3)} {bbCurrency}</strong></div>
                </div>

                <GameButton type="submit" variant="primary" className="w-full" loading={actionLoading}>
                  Đăng Đơn Thu Mua
                </GameButton>
              </form>
            )}

            {/* FORM 2: SELL ORDER (ĐĂNG ĐƠN BÁN) */}
            {createMode === "sellOrder" && (
              <form onSubmit={handleCreateSellOrderSubmit} className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Loại Vật Phẩm Bán</label>
                    <select
                      value={sellItemType}
                      onChange={(e) => setSellItemType(e.target.value as "Gu" | "Material")}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="Material">Nguyên Liệu / Tiên Tài</option>
                      <option value="Gu">Cổ Trùng Trong Túi</option>
                    </select>
                  </div>
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Đơn Vị Tiền Thu Về</label>
                    <select
                      value={sellCurrency}
                      onChange={(e) => setSellCurrency(e.target.value as "primeval" | "immortal")}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="primeval">Nguyên Thạch</option>
                      <option value="immortal">Tiên Nguyên Thạch</option>
                    </select>
                  </div>
                </div>

                {sellItemType === "Material" ? (
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Chọn Nguyên Liệu Trong Túi</label>
                    <select
                      value={sellMatID}
                      onChange={(e) => setSellMatID(e.target.value)}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="">-- Chọn Nguyên Liệu --</option>
                      {Object.entries(itemDictionary).map(([k, v]) => (
                        <option key={k} value={k}>{v.name} ({k})</option>
                      ))}
                    </select>
                  </div>
                ) : (
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Chọn Cổ Trùng Đăng Bán</label>
                    <select
                      value={sellGuInstID}
                      onChange={(e) => setSellGuInstID(e.target.value)}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="">-- Chọn Cổ Trùng --</option>
                      {guInventory.filter(g => !g.is_vital && g.status !== "DEAD").map((g) => {
                        const spec = guSpecFor(g.gu_id, g.rank);
                        return (
                          <option key={g.instance_id} value={g.instance_id}>
                            {spec?.name || g.gu_id} (Hạng {g.rank})
                          </option>
                        );
                      })}
                    </select>
                  </div>
                )}

                <div className="grid grid-cols-3 gap-4">
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Đơn Giá Bán</label>
                    <GameInput type="number" min={1} value={sellUnitPrice} onChange={(e) => setSellUnitPrice(Number(e.target.value))} />
                  </div>
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Số Lượng Bán</label>
                    <GameInput type="number" min={1} value={sellQuantity} onChange={(e) => setSellQuantity(Number(e.target.value))} />
                  </div>
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Thời Hạn (Giờ)</label>
                    <GameInput type="number" min={1} max={168} value={sellDuration} onChange={(e) => setSellDuration(Number(e.target.value))} />
                  </div>
                </div>

                <div className="bg-zinc-950 p-3 rounded-xl border border-zinc-800 text-xs font-mono text-zinc-400 space-y-1">
                  <div>• Tổng giá trị rao bán: <strong className="text-amber-400">{sellUnitPrice * sellQuantity} {sellCurrency}</strong></div>
                  <div>• Phí niêm yết (4% nộp trước): <strong className="text-amber-400">{(sellUnitPrice * sellQuantity * 0.04).toFixed(3)} {sellCurrency}</strong></div>
                  <div>• Phí hoa hồng (6% khi bán thành công): <strong className="text-red-400">{(sellUnitPrice * sellQuantity * 0.06).toFixed(3)} {sellCurrency}</strong></div>
                  <div>• Thực nhận dự kiến: <strong className="text-emerald-400 font-bold">{(sellUnitPrice * sellQuantity * 0.94).toFixed(3)} {sellCurrency}</strong></div>
                </div>

                <GameButton type="submit" variant="primary" className="w-full" loading={actionLoading}>
                  Đăng Đơn Rao Bán
                </GameButton>
              </form>
            )}

            {/* FORM 3: ASYNC TRADE OFFER */}
            {createMode === "tradeOffer" && (
              <form onSubmit={handleCreateTradeOfferSubmit} className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="bg-zinc-950/60 p-4 rounded-xl border border-zinc-800 space-y-3">
                    <h5 className="text-xs font-bold text-emerald-400 uppercase">Đồ Đem Trao Đổi (Offered)</h5>
                    <div>
                      <label className="text-[11px] text-zinc-400 block mb-1">Nguyên Thạch Đưa Đi</label>
                      <GameInput type="number" min={0} value={offeredStones} onChange={(e) => setOfferedStones(Number(e.target.value))} />
                    </div>
                    <div>
                      <label className="text-[11px] text-zinc-400 block mb-1">Tiên Nguyên Đưa Đi</label>
                      <GameInput type="number" min={0} value={offeredImmortal} onChange={(e) => setOfferedImmortal(Number(e.target.value))} />
                    </div>
                  </div>

                  <div className="bg-zinc-950/60 p-4 rounded-xl border border-zinc-800 space-y-3">
                    <h5 className="text-xs font-bold text-amber-400 uppercase">Đồ Muốn Nhận (Requested)</h5>
                    <div>
                      <label className="text-[11px] text-zinc-400 block mb-1">Nguyên Thạch Yêu Cầu</label>
                      <GameInput type="number" min={0} value={requestedStones} onChange={(e) => setRequestedStones(Number(e.target.value))} />
                    </div>
                    <div>
                      <label className="text-[11px] text-zinc-400 block mb-1">Tiên Nguyên Yêu Cầu</label>
                      <GameInput type="number" min={0} value={requestedImmortal} onChange={(e) => setRequestedImmortal(Number(e.target.value))} />
                    </div>
                  </div>
                </div>

                <GameButton type="submit" variant="primary" className="w-full" loading={actionLoading}>
                  Đăng Đơn Trao Đổi Auto
                </GameButton>
              </form>
            )}

            {/* FORM 4: AUCTION */}
            {createMode === "auction" && (
              <form onSubmit={handleCreateAuctionSubmit} className="space-y-4">
                <div>
                  <label className="text-xs text-zinc-400 block mb-1">Chọn Cổ Trùng Đưa Đấu Giá</label>
                  <select
                    value={selectedAucGu}
                    onChange={(e) => setSelectedAucGu(e.target.value)}
                    className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                  >
                    <option value="">-- Chọn Cổ Trùng --</option>
                    {guInventory.map((g) => {
                      const spec = guSpecFor(g.gu_id, g.rank);
                      return (
                        <option key={g.instance_id} value={g.instance_id}>
                          {spec?.name || g.gu_id} (Hạng {g.rank})
                        </option>
                      );
                    })}
                  </select>
                </div>
                <div>
                  <label className="text-xs text-zinc-400 block mb-1">Giá Khởi Điểm</label>
                  <GameInput type="number" min={1} value={aucPrice} onChange={(e) => setAucPrice(Number(e.target.value))} />
                </div>
                <GameButton type="submit" variant="primary" className="w-full" loading={actionLoading}>
                  Đưa Lên Đấu Giá
                </GameButton>
              </form>
            )}
          </GameCard>
        )}
      </div>

      {/* MODAL: BUY/SELL INTERACTION FOR ORDER BOOK */}
      {selectedBuybackOrder && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
          <GameCard variant="vital" className="p-6 max-w-md w-full space-y-4 relative">
            <h3 className="text-lg font-bold font-serif text-amber-200">
              {selectedBuybackOrder.order_type === "SELL" ? "Mua Vật Phẩm Từ Đơn Rao Bán" : "Bán Vật Phẩm Vào Đơn Thu Mua"}
            </h3>
            <p className="text-xs text-zinc-400">
              Đơn {selectedBuybackOrder.order_type === "SELL" ? "rao bán" : "thu mua"} yêu cầu:{" "}
              <strong className="text-amber-300">
                {selectedBuybackOrder.quantity_requested - selectedBuybackOrder.quantity_filled} món
              </strong>{" "}
              với đơn giá <strong className="text-amber-300">{selectedBuybackOrder.unit_price} {selectedBuybackOrder.currency}</strong>.
            </p>

            <div>
              <label className="text-xs text-zinc-400 block mb-1">Số lượng giao dịch</label>
              <GameInput
                type="number"
                min={1}
                max={selectedBuybackOrder.quantity_requested - selectedBuybackOrder.quantity_filled}
                value={sellQtyInput}
                onChange={(e) => setSellQtyInput(Number(e.target.value))}
              />
            </div>

            <div className="bg-zinc-950 p-3 rounded-xl border border-zinc-800 text-xs font-mono space-y-1">
              {selectedBuybackOrder.order_type === "SELL" ? (
                <>
                  <div className="text-zinc-400">• Tổng thanh toán: <span className="text-amber-400 font-bold">{sellQtyInput * selectedBuybackOrder.unit_price} {selectedBuybackOrder.currency}</span></div>
                </>
              ) : (
                <>
                  <div className="text-zinc-400">• Tổng nhận thô: <span className="text-amber-400">{sellQtyInput * selectedBuybackOrder.unit_price} {selectedBuybackOrder.currency}</span></div>
                  <div className="text-zinc-400">• Phí hoa hồng (6%): <span className="text-red-400">-{(sellQtyInput * selectedBuybackOrder.unit_price * 0.06).toFixed(3)} {selectedBuybackOrder.currency}</span></div>
                  <div className="text-amber-200 font-bold">• Thực nhận: {(sellQtyInput * selectedBuybackOrder.unit_price * 0.94).toFixed(3)} {selectedBuybackOrder.currency}</div>
                </>
              )}
            </div>

            <div className="flex items-center gap-3 pt-2">
              <GameButton
                variant="ghost"
                className="flex-1"
                onClick={() => setSelectedBuybackOrder(null)}
              >
                Hủy Thao Tác
              </GameButton>
              <GameButton
                variant="primary"
                className="flex-1"
                loading={actionLoading}
                onClick={async () => {
                  await sellToBuybackOrder({
                    orderId: selectedBuybackOrder.id,
                    quantity: sellQtyInput,
                  });
                  setSelectedBuybackOrder(null);
                }}
              >
                {selectedBuybackOrder.order_type === "SELL" ? "Xác Nhận Mua" : "Xác Nhận Bán"}
              </GameButton>
            </div>
          </GameCard>
        </div>
      )}
    </GamePanel>
  );
}
