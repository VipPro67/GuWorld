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
  Bug,
  ScrollText,
  Plus,
  Minus,
  X,
} from "lucide-react";
import { GamePanel, GameTabs, GameButton, GameCard, GameInput, GameBadge } from "@/components/ui";
import { TradeAssetPickerModal, type AssetTab } from "./TradeAssetPickerModal";
import { buildBuyOrderExecution, reconcileGuSelection, toggleGuSelection } from "./marketplaceGuSelection";

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
  guDictionary,
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
  
  const recipes = useGameStore((s) => s.recipes);
  const fetchRecipes = useGameStore((s) => s.fetchRecipes);

  // Buyback Order Creation Form State
  const [bbItemType, setBbItemType] = useState<"Gu" | "Material" | "Recipe">("Material");
  const [bbTargetMat, setBbTargetMat] = useState("");
  const [bbTargetGu, setBbTargetGu] = useState("");
  const [bbTargetRank, setBbTargetRank] = useState(1);
  const [bbTargetRecipe, setBbTargetRecipe] = useState("");
  const [bbUnitPrice, setBbUnitPrice] = useState(10);
  const [bbQuantity, setBbQuantity] = useState(1);
  const [bbCurrency, setBbCurrency] = useState<"primeval" | "immortal">("primeval");
  const [bbDuration, setBbDuration] = useState(72);

  // Sell Order Creation Form State
  const [sellItemType, setSellItemType] = useState<"Gu" | "Material" | "Recipe">("Material");
  const [sellMatID, setSellMatID] = useState("");
  const [sellGuInstIDs, setSellGuInstIDs] = useState<string[]>([]);
  const [sellGuTarget, setSellGuTarget] = useState("");
  const [sellRecipeID, setSellRecipeID] = useState("");
  const [sellUnitPrice, setSellUnitPrice] = useState(10);
  const [sellQuantity, setSellQuantity] = useState(1);
  const [sellCurrency, setSellCurrency] = useState<"primeval" | "immortal">("primeval");
  const [sellDuration, setSellDuration] = useState(72);

  // Modal state for Buy/Sell Interaction
  const [selectedBuybackOrder, setSelectedBuybackOrder] = useState<BuybackOrder | null>(null);
  const [sellQtyInput, setSellQtyInput] = useState(1);
  const [selectedBuybackGuIDs, setSelectedBuybackGuIDs] = useState<string[]>([]);

  // Form state for Async Trade Offer Creation
  const [offeredStones, setOfferedStones] = useState(0);
  const [offeredImmortal, setOfferedImmortal] = useState(0);
  const [offeredGuIDs, setOfferedGuIDs] = useState<string[]>([]);
  const [offeredMats, setOfferedMats] = useState<Record<string, number>>({});
  const [offeredRecipes, setOfferedRecipes] = useState<Record<string, number>>({});

  const [requestedStones, setRequestedStones] = useState(0);
  const [requestedImmortal, setRequestedImmortal] = useState(0);
  const [requestedGuIDs, setRequestedGuIDs] = useState<string[]>([]);
  const [requestedMats, setRequestedMats] = useState<Record<string, number>>({});
  const [requestedRecipes, setRequestedRecipes] = useState<Record<string, number>>({});
  const [tradeOfferDuration, setTradeOfferDuration] = useState(72);
  const [tradeOfferTargetUser, setTradeOfferTargetUser] = useState("");
  const [tradeOfferFilter, setTradeOfferFilter] = useState<"all" | "mine" | "targeted">("all");

  // Visual Asset Picker Modal State
  const [pickerOpen, setPickerOpen] = useState(false);
  const [pickerMode, setPickerMode] = useState<"offer" | "request">("offer");
  const [pickerTab, setPickerTab] = useState<AssetTab>("material");

  const openAssetPicker = (mode: "offer" | "request", tab: AssetTab) => {
    setPickerMode(mode);
    setPickerTab(tab);
    setPickerOpen(true);
  };

  const handleAddMaterialFromPicker = (matId: string, qty: number) => {
    if (pickerMode === "offer") {
      const owned = player.materials_inventory?.[matId] || 0;
      const current = offeredMats[matId] || 0;
      const newQty = Math.min(owned, current + qty);
      setOfferedMats((prev) => ({ ...prev, [matId]: newQty }));
    } else {
      setRequestedMats((prev) => ({ ...prev, [matId]: (prev[matId] || 0) + qty }));
    }
  };

  const handleAddGuFromPicker = (guIdentifier: string) => {
    if (pickerMode === "offer") {
      if (!offeredGuIDs.includes(guIdentifier)) {
        setOfferedGuIDs((prev) => [...prev, guIdentifier]);
      }
    } else {
      if (!requestedGuIDs.includes(guIdentifier)) {
        setRequestedGuIDs((prev) => [...prev, guIdentifier]);
      }
    }
  };

  const handleAddRecipeFromPicker = (recipeId: string, completionRate: number = 100) => {
    if (pickerMode === "offer") {
      setOfferedRecipes((prev) => ({ ...prev, [recipeId]: completionRate }));
    } else {
      setRequestedRecipes((prev) => ({ ...prev, [recipeId]: 100 }));
    }
  };

  // Auction form state
  const [aucItemType] = useState<"Gu" | "Material" | "Recipe">("Material");
  const [aucPrice, setAucPrice] = useState(10);
  const [aucDuration] = useState(24);
  const [aucCurrency] = useState<"primeval" | "immortal">("primeval");
  const [selectedAucGu, setSelectedAucGu] = useState("");
  const [selectedAucMat] = useState("");

  const [searchQuery, setSearchQuery] = useState("");

  useEffect(() => {
    fetchRecipes();
  }, [fetchRecipes]);

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
    } else if (bbItemType === "Recipe") {
      if (!bbTargetRecipe) return;
      targetPayload = bbTargetRecipe;
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
      if (sellGuInstIDs.length === 0) return;
      const g = guInventory.find((inst) => inst.instance_id === sellGuInstIDs[0]);
      if (!g) return;
      targetPayload = { gu_id: g.gu_id, rank: g.rank };
      guInstIDs = sellGuInstIDs;
    } else if (sellItemType === "Recipe") {
      if (!sellRecipeID) return;
      targetPayload = sellRecipeID;
    }
    const quantityRequested = sellItemType === "Gu" ? sellGuInstIDs.length : sellItemType === "Recipe" ? 1 : sellQuantity;
    await createBuybackOrder({
      orderType: "SELL",
      itemType: sellItemType,
      itemTarget: targetPayload,
      guInstanceIds: guInstIDs,
      unitPrice: sellUnitPrice,
      quantityRequested,
      currency: sellCurrency,
      durationHours: sellDuration,
    });
    setTab("orderBook");
  };

  const handleCreateTradeOfferSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const success = await createTradeOffer({
      offered: {
        primeval_stones: offeredStones,
        immortal_stones: offeredImmortal,
        gus: offeredGuIDs,
        materials: offeredMats,
        recipes: offeredRecipes,
      },
      requested: {
        primeval_stones: requestedStones,
        immortal_stones: requestedImmortal,
        gus: requestedGuIDs,
        materials: requestedMats,
        recipes: requestedRecipes,
      },
      durationHours: tradeOfferDuration,
      targetUsername: tradeOfferTargetUser.trim() || undefined,
    });
    if (success) {
      setOfferedStones(0);
      setOfferedImmortal(0);
      setOfferedGuIDs([]);
      setOfferedMats({});
      setOfferedRecipes({});
      setRequestedStones(0);
      setRequestedImmortal(0);
      setRequestedGuIDs([]);
      setRequestedMats({});
      setRequestedRecipes({});
      setTradeOfferTargetUser("");
      setTab("asyncTrade");
    }
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

  // Auction custom bid modal state
  const [biddingListing, setBiddingListing] = useState<Listing | null>(null);
  const [customBidPrice, setCustomBidPrice] = useState(0);

  const activeAuctionListings = listings.filter((l) => {
    if (l.listing_type !== "AUCTION") return false;
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    const details = getListingDetails(l);
    const seller = (l.target_buyer_username || l.seller_id).toLowerCase();
    return details.title.toLowerCase().includes(q) || (details.subtitle && details.subtitle.toLowerCase().includes(q)) || seller.includes(q);
  });

  const filteredOrderBook = buybackOrders.filter((o) => {
    const orderType = o.order_type || "BUY";
    if (orderBookFilter !== "all" && orderType !== orderBookFilter) return false;
    if (!searchQuery.trim()) return true;

    const q = searchQuery.toLowerCase();
    let targetName = "Vật phẩm";
    if (o.item_type === "Material" && typeof o.item_target === "string") {
      targetName = itemDictionary[o.item_target]?.name || o.item_target;
    } else if (o.item_type === "Gu" && typeof o.item_target === "object" && o.item_target !== null) {
      const t = o.item_target as { gu_id?: string; rank?: number };
      if (t.gu_id) {
        targetName = guSpecFor(t.gu_id, t.rank || 1)?.name || t.gu_id;
      }
    } else if (o.item_type === "Recipe") {
      const recId = typeof o.item_target === "string" ? o.item_target : String(o.item_target);
      const rSpec = recipes.find((r) => r.id === recId);
      const targetGuName = rSpec?.target_gu_id ? guSpecFor(rSpec.target_gu_id, rSpec.target_rank || 1)?.name : "";
      targetName = targetGuName ? `Bí phương ${targetGuName}` : `Bí phương ${recId}`;
    }
    const buyerName = (o.buyer_username || o.buyer_id).toLowerCase();
    return targetName.toLowerCase().includes(q) || o.item_type.toLowerCase().includes(q) || buyerName.includes(q);
  });

  const filteredTradeOffers = tradeOffers.filter((offer) => {
    if (tradeOfferFilter === "mine" && offer.creator_id !== player.id) return false;
    if (tradeOfferFilter === "targeted" && offer.target_id !== player.id && offer.target_username !== player.username) return false;

    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    const creator = (offer.creator_username || offer.creator_id).toLowerCase();
    const target = (offer.target_username || offer.target_id || "").toLowerCase();
    const offeredMatNames = Object.keys(offer.offered.materials || {}).map((k) => (itemDictionary[k]?.name || k).toLowerCase());
    const requestedMatNames = Object.keys(offer.requested.materials || {}).map((k) => (itemDictionary[k]?.name || k).toLowerCase());
    const offeredGuNames = (offer.offered_gu_details && offer.offered_gu_details.length > 0)
      ? offer.offered_gu_details.map(g => g.gu_name.toLowerCase())
      : (offer.offered.gus || []).map((gid) => {
          const g = guInventory.find((inst) => inst.instance_id === gid);
          return g ? (guSpecFor(g.gu_id, g.rank)?.name || g.gu_id).toLowerCase() : gid.toLowerCase();
        });
    const requestedGuNames = (offer.requested.gus || []).map((gid) => {
      const spec = guSpecFor(gid, 1) || guDictionary?.[gid];
      return spec ? spec.name.toLowerCase() : gid.toLowerCase();
    });
    const offeredRecNames = Object.keys(offer.offered.recipes || {}).map((r) => r.toLowerCase());
    const requestedRecNames = Object.keys(offer.requested.recipes || {}).map((r) => r.toLowerCase());

    return (
      creator.includes(q) ||
      target.includes(q) ||
      offeredMatNames.some((n) => n.includes(q)) ||
      requestedMatNames.some((n) => n.includes(q)) ||
      offeredGuNames.some((n) => n.includes(q)) ||
      requestedGuNames.some((n) => n.includes(q)) ||
      offeredRecNames.some((n) => n.includes(q)) ||
      requestedRecNames.some((n) => n.includes(q))
    );
  });

  const sellableGus = guInventory.filter((g) => !g.is_vital && !g.equipped && !g.is_locked && g.status !== "DEAD");
  const sellGuGroupMap = new Map<string, { key: string; guId: string; rank: number; gus: GuInstance[] }>();
  for (const g of sellableGus) {
    const key = `${g.gu_id}::${g.rank}`;
    const group = sellGuGroupMap.get(key);
    if (group) group.gus.push(g);
    else sellGuGroupMap.set(key, { key, guId: g.gu_id, rank: g.rank, gus: [g] });
  }
  const sellGuGroups = Array.from(sellGuGroupMap.values());
  const selectedSellGroup = sellGuGroups.find((group) => group.key === sellGuTarget);
  const sellGuGroup = selectedSellGroup?.gus || [];
  const effectiveSellQuantity = sellItemType === "Recipe" ? 1 : sellQuantity;

  const eligibleGusForBuybackOrder = (order: BuybackOrder) => {
    if (order.order_type === "SELL" || order.item_type !== "Gu") return [];
    const target = order.item_target as { gu_id?: string; rank?: number };
    return sellableGus.filter((g) => g.gu_id === target.gu_id && g.rank === Number(target.rank));
  };

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
                  } else if (order.item_type === "Recipe") {
                    const recId = typeof order.item_target === "string" ? order.item_target : String(order.item_target);
                    const rSpec = recipes.find((r) => r.id === recId);
                    const targetGuName = rSpec?.target_gu_id ? guSpecFor(rSpec.target_gu_id, rSpec.target_rank || 1)?.name : "";
                    targetName = targetGuName ? `Bí phương ${targetGuName}` : `Bí phương ${recId}`;
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
                              const initialGuIDs = reconcileGuSelection(eligibleGusForBuybackOrder(order).map((g) => g.instance_id), [], 1);
                              setSellQtyInput(order.item_type === "Gu" && order.order_type !== "SELL" ? initialGuIDs.length : 1);
                              setSelectedBuybackGuIDs(initialGuIDs);
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
                              const initialGuIDs = reconcileGuSelection(eligibleGusForBuybackOrder(order).map((g) => g.instance_id), [], 1);
                              setSellQtyInput(order.item_type === "Gu" && order.order_type !== "SELL" ? initialGuIDs.length : 1);
                              setSelectedBuybackGuIDs(initialGuIDs);
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
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex gap-2">
                <GameButton
                  size="xs"
                  variant={tradeOfferFilter === "all" ? "primary" : "ghost"}
                  onClick={() => setTradeOfferFilter("all")}
                >
                  Tất Cả ({tradeOffers.length})
                </GameButton>
                <GameButton
                  size="xs"
                  variant={tradeOfferFilter === "mine" ? "primary" : "ghost"}
                  onClick={() => setTradeOfferFilter("mine")}
                >
                  Của Tôi ({tradeOffers.filter(o => o.creator_id === player.id).length})
                </GameButton>
                <GameButton
                  size="xs"
                  variant={tradeOfferFilter === "targeted" ? "primary" : "ghost"}
                  onClick={() => setTradeOfferFilter("targeted")}
                >
                  Chỉ Định Cho Tôi ({tradeOffers.filter(o => o.target_id === player.id || o.target_username === player.username).length})
                </GameButton>
              </div>

              <div className="relative w-64">
                <Search className="w-4 h-4 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2" />
                <GameInput
                  placeholder="Tìm kiếm đơn trao đổi..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9 text-xs"
                />
              </div>
            </div>

            {filteredTradeOffers.length === 0 ? (
              <div className="text-center py-16 bg-zinc-950/40 rounded-2xl border border-zinc-900 text-zinc-500 font-serif">
                Chưa có đơn trao đổi tự động nào khớp với tìm kiếm.
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {filteredTradeOffers.map((offer) => {
                  const isMine = offer.creator_id === player.id;
                  const isTargetedToOther = !!offer.target_username && offer.target_username !== player.username && !isMine;

                  const hasPrimeval = (player.primeval_stones || 0) >= (offer.requested.primeval_stones || 0);
                  const hasImmortal = offer.requested.immortal_stones > 0 ? (player.rank >= 6 && (player.immortal_stones || 0) >= offer.requested.immortal_stones) : true;
                  const hasMats = Object.entries(offer.requested.materials || {}).every(([matId, qty]) => (player.materials_inventory?.[matId] || 0) >= qty);
                  const hasGus = (offer.requested.gus || []).every(gid => guInventory.some(g => (g.gu_id === gid || g.instance_id === gid) && !g.is_vital && !g.equipped && g.status !== "DEAD"));
                  const hasRecipes = Object.keys(offer.requested.recipes || {}).every(recId => (player.unlocked_recipes || []).some(r => r.recipe_id === recId));
                  const canAccept = hasPrimeval && hasImmortal && hasMats && hasGus && hasRecipes && !isTargetedToOther;

                  return (
                    <GameCard key={offer.id} variant="vital" className="p-4 space-y-4">
                      <div className="flex items-center justify-between border-b border-zinc-800 pb-2">
                        <div className="flex items-center gap-2 flex-wrap">
                          <ArrowRightLeft className="w-4 h-4 text-cyan-400" />
                          <span className="text-sm font-bold font-serif text-zinc-200">{offer.creator_username}</span>
                          {offer.target_username && (
                            <span className="text-[10px] bg-amber-500/20 text-amber-300 border border-amber-500/30 px-2 py-0.5 rounded-full font-sans">
                              🎯 Chỉ định: {offer.target_username}
                            </span>
                          )}
                          {isMine && (
                            <span className="text-[10px] bg-cyan-500/20 text-cyan-300 border border-cyan-500/30 px-2 py-0.5 rounded-full font-sans">
                              Đơn của tôi
                            </span>
                          )}
                        </div>
                        <span className="text-xs text-zinc-500 font-mono">
                          Hạn: {new Date(offer.end_time).toLocaleDateString()}
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
                          {offer.offered_gu_details && offer.offered_gu_details.length > 0 ? (
                            offer.offered_gu_details.map((g) => (
                              <div key={g.instance_id}>• Cổ: {g.gu_name} (Hạng {g.rank})</div>
                            ))
                          ) : (
                            (offer.offered.gus || []).map((instId) => {
                              const g = guInventory.find((i) => i.instance_id === instId);
                              const name = g ? (guSpecFor(g.gu_id, g.rank)?.name || g.gu_id) : instId;
                              return <div key={instId}>• Cổ: {name}</div>;
                            })
                          )}
                          {Object.entries(offer.offered.recipes || {}).map(([recId, rate]) => {
                            const rSpec = recipes.find((r) => r.id === recId);
                            const targetGu = rSpec?.target_gu_id ? guSpecFor(rSpec.target_gu_id, rSpec.target_rank || 1)?.name : "";
                            const name = targetGu ? `Bí phương ${targetGu}` : `Bí phương ${recId}`;
                            return <div key={recId}>• {name} ({rate}%)</div>;
                          })}
                        </div>

                        <div className="bg-amber-950/20 p-3 rounded-xl border border-amber-500/20 space-y-1">
                          <span className="text-amber-400 font-bold text-[11px] block border-b border-amber-500/20 pb-1 mb-1">
                            VẬT YÊU CẦU (REQUESTED)
                          </span>
                          {offer.requested.primeval_stones > 0 && (
                            <div className="flex items-center justify-between">
                              <span>• {offer.requested.primeval_stones} Nguyên Thạch</span>
                              {!isMine && (
                                <span className={hasPrimeval ? "text-emerald-400 font-bold text-[10px]" : "text-red-400 font-bold text-[10px]"}>
                                  {hasPrimeval ? "✓" : `✗`}
                                </span>
                              )}
                            </div>
                          )}
                          {offer.requested.immortal_stones > 0 && (
                            <div className="flex items-center justify-between">
                              <span>• {offer.requested.immortal_stones} Tiên Nguyên</span>
                              {!isMine && (
                                <span className={hasImmortal ? "text-emerald-400 font-bold text-[10px]" : "text-red-400 font-bold text-[10px]"}>
                                  {hasImmortal ? "✓" : `✗`}
                                </span>
                              )}
                            </div>
                          )}
                          {Object.entries(offer.requested.materials || {}).map(([k, v]) => {
                            const owned = player.materials_inventory?.[k] || 0;
                            const ok = owned >= v;
                            return (
                              <div key={k} className="flex items-center justify-between">
                                <span>• {v}x {itemDictionary[k]?.name || k}</span>
                                {!isMine && (
                                  <span className={ok ? "text-emerald-400 font-bold text-[10px]" : "text-red-400 font-bold text-[10px]"}>
                                    {ok ? `✓ (${owned})` : `✗ (${owned}/${v})`}
                                  </span>
                                )}
                              </div>
                            );
                          })}
                          {(offer.requested.gus || []).map((gid) => {
                            const spec = guSpecFor(gid, 1) || guDictionary?.[gid];
                            const ok = guInventory.some(g => (g.gu_id === gid || g.instance_id === gid) && !g.is_vital && !g.equipped && g.status !== "DEAD");
                            return (
                              <div key={gid} className="flex items-center justify-between">
                                <span>• Cổ: {spec?.name || gid}</span>
                                {!isMine && (
                                  <span className={ok ? "text-emerald-400 font-bold text-[10px]" : "text-red-400 font-bold text-[10px]"}>
                                    {ok ? "✓ Có" : "✗ Thiếu"}
                                  </span>
                                )}
                              </div>
                            );
                          })}
                          {Object.entries(offer.requested.recipes || {}).map(([recId]) => {
                            const rSpec = recipes.find((r) => r.id === recId);
                            const targetGu = rSpec?.target_gu_id ? guSpecFor(rSpec.target_gu_id, rSpec.target_rank || 1)?.name : "";
                            const name = targetGu ? `Bí phương ${targetGu}` : `Bí phương ${recId}`;
                            const ok = (player.unlocked_recipes || []).some(r => r.recipe_id === recId);
                            return (
                              <div key={recId} className="flex items-center justify-between">
                                <span>• {name}</span>
                                {!isMine && (
                                  <span className={ok ? "text-emerald-400 font-bold text-[10px]" : "text-red-400 font-bold text-[10px]"}>
                                    {ok ? "✓ Có" : "✗ Thiếu"}
                                  </span>
                                )}
                              </div>
                            );
                          })}
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
                        ) : isTargetedToOther ? (
                          <span className="text-xs text-zinc-500 italic">
                            Chỉ dành riêng cho {offer.target_username}
                          </span>
                        ) : (
                          <GameButton
                            size="xs"
                            variant={canAccept ? "primary" : "ghost"}
                            disabled={!canAccept}
                            onClick={() => acceptTradeOffer(offer.id)}
                            loading={actionLoading}
                            leftIcon={canAccept ? <CheckCircle className="w-3.5 h-3.5" /> : undefined}
                          >
                            {canAccept ? "Chấp Nhận Trao Đổi (Đủ Điều Kiện)" : "Chưa Đủ Điều Kiện Trao Đổi"}
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
            <div className="flex justify-end">
              <div className="relative w-64">
                <Search className="w-4 h-4 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2" />
                <GameInput
                  placeholder="Tìm kiếm bảo vật đấu giá..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9 text-xs"
                />
              </div>
            </div>

            {activeAuctionListings.length === 0 ? (
              <div className="text-center py-16 bg-zinc-950/40 rounded-2xl border border-zinc-900 text-zinc-500 font-serif">
                Hiện không có bảo vật nào đang được đấu giá khớp tìm kiếm.
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
                        <div className="flex items-center gap-1.5">
                          <GameButton
                            size="xs"
                            variant="ghost"
                            onClick={() => onBid(listing.id, listing.current_price + 10)}
                            loading={actionLoading}
                          >
                            +10
                          </GameButton>
                          <GameButton
                            size="xs"
                            variant="primary"
                            onClick={() => {
                              setBiddingListing(listing);
                              setCustomBidPrice(listing.current_price + 10);
                            }}
                            loading={actionLoading}
                            leftIcon={<Gavel className="w-3 h-3" />}
                          >
                            Đặt Giá
                          </GameButton>
                        </div>
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
                      <option value="Recipe">Bí Phương Phối Chế</option>
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

                {bbItemType === "Recipe" && (
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Chọn Bí Phương Cần Thu Mua</label>
                    <select
                      value={bbTargetRecipe}
                      onChange={(e) => setBbTargetRecipe(e.target.value)}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="">-- Chọn Bí Phương --</option>
                      {recipes.map((r) => {
                        const targetGu = r.target_gu_id ? guSpecFor(r.target_gu_id, r.target_rank || 1)?.name : "";
                        return (
                          <option key={r.id} value={r.id}>
                            {targetGu ? `Bí phương ${targetGu} (${r.id})` : r.id}
                          </option>
                        );
                      })}
                    </select>
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
                      onChange={(e) => {
                        setSellItemType(e.target.value as "Gu" | "Material" | "Recipe");
                        setSellGuInstIDs([]);
                        setSellGuTarget("");
                        setSellQuantity(1);
                      }}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="Material">Nguyên Liệu / Tiên Tài</option>
                      <option value="Gu">Cổ Trùng Trong Túi</option>
                      <option value="Recipe">Bí Phương Đã Khai Mở</option>
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
                ) : sellItemType === "Gu" ? (
                  <div className="space-y-2">
                    <label className="text-xs text-zinc-400 block mb-1">Chọn loại Cổ Trùng Đăng Bán</label>
                    <select
                      value={sellGuTarget}
                      onChange={(e) => {
                        const group = sellGuGroups.find((candidate) => candidate.key === e.target.value);
                        const next = group ? group.gus.slice(0, 1).map((g) => g.instance_id) : [];
                        setSellGuTarget(e.target.value);
                        setSellGuInstIDs(next);
                        setSellQuantity(next.length);
                      }}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="">-- Chọn Cổ Trùng --</option>
                      {sellGuGroups.map((group) => {
                        const spec = guSpecFor(group.guId, group.rank);
                        return (
                          <option key={group.key} value={group.key}>
                            {spec?.name || group.guId} (Hạng {group.rank}) — {group.gus.length} con
                          </option>
                        );
                      })}
                    </select>
                    {selectedSellGroup && (
                      <div className="max-h-40 overflow-y-auto space-y-1 rounded-xl border border-zinc-800 bg-zinc-950 p-2">
                        {sellGuGroup.map((g) => {
                          const checked = sellGuInstIDs.includes(g.instance_id);
                          return (
                            <label key={g.instance_id} className="flex cursor-pointer items-center gap-2 rounded-lg p-2 text-xs text-zinc-300 hover:bg-zinc-900">
                              <input type="checkbox" checked={checked} onChange={() => {
                                const next = toggleGuSelection(sellGuInstIDs, g.instance_id);
                                setSellGuInstIDs(next);
                                setSellQuantity(next.length);
                              }} />
                              <span>{guSpecFor(g.gu_id, g.rank)?.name || g.gu_id} (Hạng {g.rank})</span>
                              <span className="ml-auto font-mono text-zinc-600">{g.instance_id.slice(0, 8)}</span>
                            </label>
                          );
                        })}
                      </div>
                    )}
                  </div>
                ) : (
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Chọn Bí Phương Đăng Bán</label>
                    <select
                      value={sellRecipeID}
                      onChange={(e) => setSellRecipeID(e.target.value)}
                      className="w-full bg-zinc-950 border border-zinc-800 rounded-xl p-2.5 text-xs text-zinc-200"
                    >
                      <option value="">-- Chọn Bí Phương Trong Kho --</option>
                      {(player.unlocked_recipes || []).map((ur) => {
                        const rSpec = recipes.find((r) => r.id === ur.recipe_id);
                        const targetGu = rSpec?.target_gu_id ? guSpecFor(rSpec.target_gu_id, rSpec.target_rank || 1)?.name : "";
                        return (
                          <option key={ur.recipe_id} value={ur.recipe_id}>
                            {targetGu ? `Bí phương ${targetGu} (${ur.completion_rate}%)` : `${ur.recipe_id} (${ur.completion_rate}%)`}
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
                    <div className="flex items-center justify-between mb-1">
                      <label className="text-xs text-zinc-400">Số Lượng Bán</label>
                      {sellItemType === "Gu" && selectedSellGroup && (
                        <button type="button" className="text-[10px] font-mono text-amber-400" onClick={() => {
                          const ids = sellGuGroup.map((g) => g.instance_id);
                          setSellGuInstIDs(ids);
                          setSellQuantity(ids.length);
                        }}>Tối Đa</button>
                      )}
                    </div>
                    <GameInput
                      type="number"
                      min={sellItemType === "Gu" ? 0 : 1}
                      max={sellItemType === "Gu" ? sellGuGroup.length : undefined}
                      disabled={sellItemType === "Recipe" || (sellItemType === "Gu" && !selectedSellGroup)}
                      value={sellItemType === "Recipe" ? 1 : sellQuantity}
                      onChange={(e) => {
                        const quantity = Math.max(0, Number(e.target.value) || 0);
                        if (sellItemType === "Gu") {
                          const next = reconcileGuSelection(sellGuGroup.map((g) => g.instance_id), sellGuInstIDs, quantity);
                          setSellGuInstIDs(next);
                          setSellQuantity(next.length);
                        } else {
                          setSellQuantity(quantity);
                        }
                      }}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-zinc-400 block mb-1">Thời Hạn (Giờ)</label>
                    <GameInput type="number" min={1} max={168} value={sellDuration} onChange={(e) => setSellDuration(Number(e.target.value))} />
                  </div>
                </div>

                <div className="bg-zinc-950 p-3 rounded-xl border border-zinc-800 text-xs font-mono text-zinc-400 space-y-1">
                  <div>• Tổng giá trị rao bán: <strong className="text-amber-400">{sellUnitPrice * effectiveSellQuantity} {sellCurrency}</strong></div>
                  <div>• Phí niêm yết (4% nộp trước): <strong className="text-amber-400">{(sellUnitPrice * effectiveSellQuantity * 0.04).toFixed(3)} {sellCurrency}</strong></div>
                  <div>• Phí hoa hồng (6% khi bán thành công): <strong className="text-red-400">{(sellUnitPrice * effectiveSellQuantity * 0.06).toFixed(3)} {sellCurrency}</strong></div>
                  <div>• Thực nhận dự kiến: <strong className="text-emerald-400 font-bold">{(sellUnitPrice * effectiveSellQuantity * 0.94).toFixed(3)} {sellCurrency}</strong></div>
                </div>

                <GameButton type="submit" variant="primary" className="w-full" loading={actionLoading} disabled={sellItemType === "Gu" && sellGuInstIDs.length === 0}>
                  Đăng Đơn Rao Bán
                </GameButton>
              </form>
            )}

            {/* FORM 3: ASYNC TRADE OFFER */}
            {createMode === "tradeOffer" && (
              <form onSubmit={handleCreateTradeOfferSubmit} className="space-y-4">

                {/* ===== PANEL: ĐỒ ĐEM TRAO ĐỔI ===== */}
                <div className="rounded-xl border border-emerald-500/30 overflow-hidden bg-zinc-950/60">
                  {/* Header */}
                  <div className="bg-emerald-950/60 px-4 py-2.5 flex items-center justify-between border-b border-emerald-500/20">
                    <div className="flex items-center gap-2">
                      <span className="text-emerald-400 text-base">📤</span>
                      <span className="text-sm font-bold text-emerald-300">Đồ Đem Trao Đổi</span>
                      <span className="text-[10px] text-zinc-500 font-normal">(tài sản trong túi bạn sẽ bị khóa)</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <GameButton
                        size="xs"
                        variant="secondary"
                        type="button"
                        onClick={() => openAssetPicker("offer", "material")}
                        className="text-xs"
                      >
                        + Vật Phẩm
                      </GameButton>
                      <GameButton
                        size="xs"
                        variant="secondary"
                        type="button"
                        onClick={() => openAssetPicker("offer", "gu")}
                        className="text-xs"
                      >
                        + Cổ Trùng
                      </GameButton>
                      <GameButton
                        size="xs"
                        variant="secondary"
                        type="button"
                        onClick={() => openAssetPicker("offer", "recipe")}
                        className="text-xs"
                      >
                        + Bí Phương
                      </GameButton>
                    </div>
                  </div>

                  <div className="p-4 space-y-4">
                    {/* Currency */}
                    <div>
                      <p className="text-[11px] text-zinc-500 font-semibold uppercase mb-2">💰 Tiền Tệ</p>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <div className="flex justify-between items-center mb-1">
                            <label className="text-[11px] text-zinc-400">Nguyên Thạch</label>
                            <button
                              type="button"
                              onClick={() => setOfferedStones(Math.floor(player.primeval_stones || 0))}
                              className="text-[10px] text-emerald-400 hover:text-emerald-300 font-medium"
                            >
                              Tối đa: {(player.primeval_stones || 0).toLocaleString()}
                            </button>
                          </div>
                          <GameInput
                            type="number"
                            min={0}
                            value={offeredStones}
                            onChange={(e) => setOfferedStones(Number(e.target.value))}
                          />
                        </div>
                        <div>
                          <div className="flex justify-between items-center mb-1">
                            <label className="text-[11px] text-zinc-400">Tiên Nguyên</label>
                            {player.rank >= 6 && (
                              <button
                                type="button"
                                onClick={() => setOfferedImmortal(Math.floor(player.immortal_stones || 0))}
                                className="text-[10px] text-emerald-400 hover:text-emerald-300 font-medium"
                              >
                                Tối đa: {(player.immortal_stones || 0).toLocaleString()}
                              </button>
                            )}
                          </div>
                          <GameInput
                            type="number"
                            min={0}
                            value={offeredImmortal}
                            onChange={(e) => setOfferedImmortal(Number(e.target.value))}
                            disabled={player.rank < 6}
                          />
                        </div>
                      </div>
                    </div>

                    {/* Visual Selections Display */}
                    <div className="border-t border-zinc-800/80 pt-3 space-y-3">
                      {Object.keys(offeredMats).length === 0 && offeredGuIDs.length === 0 && Object.keys(offeredRecipes).length === 0 ? (
                        <div className="border border-dashed border-zinc-800 rounded-xl p-4 text-center">
                          <p className="text-xs text-zinc-500">Chưa có vật phẩm, cổ trùng hay bí phương nào được đưa vào đơn.</p>
                          <div className="flex justify-center gap-3 mt-2 text-xs">
                            <button
                              type="button"
                              onClick={() => openAssetPicker("offer", "material")}
                              className="text-emerald-400 hover:underline flex items-center gap-1 font-medium"
                            >
                              <Package className="w-3.5 h-3.5" /> + Chọn Vật Phẩm
                            </button>
                            <span className="text-zinc-700">•</span>
                            <button
                              type="button"
                              onClick={() => openAssetPicker("offer", "gu")}
                              className="text-emerald-400 hover:underline flex items-center gap-1 font-medium"
                            >
                              <Bug className="w-3.5 h-3.5" /> + Chọn Cổ Trùng
                            </button>
                            <span className="text-zinc-700">•</span>
                            <button
                              type="button"
                              onClick={() => openAssetPicker("offer", "recipe")}
                              className="text-emerald-400 hover:underline flex items-center gap-1 font-medium"
                            >
                              <ScrollText className="w-3.5 h-3.5" /> + Chọn Bí Phương
                            </button>
                          </div>
                        </div>
                      ) : (
                        <>
                          {/* Materials */}
                          {Object.keys(offeredMats).length > 0 && (
                            <div>
                              <div className="flex items-center justify-between mb-1.5">
                                <span className="text-[11px] font-semibold text-zinc-400 uppercase flex items-center gap-1">
                                  <Package className="w-3 h-3 text-emerald-400" /> Vật Phẩm ({Object.keys(offeredMats).length})
                                </span>
                                <button
                                  type="button"
                                  onClick={() => openAssetPicker("offer", "material")}
                                  className="text-[10px] text-emerald-400 hover:text-emerald-300 font-medium"
                                >
                                  + Thêm / Sửa
                                </button>
                              </div>
                              <div className="flex flex-wrap gap-2">
                                {Object.entries(offeredMats).map(([matId, qty]) => {
                                  const total = player.materials_inventory?.[matId] || 0;
                                  const avail = total - qty;
                                  return (
                                    <div
                                      key={matId}
                                      className="inline-flex items-center gap-2 bg-zinc-900 border border-emerald-500/40 rounded-xl px-2.5 py-1 text-xs text-emerald-200"
                                    >
                                      <span className="font-semibold text-white">{qty}×</span>
                                      <span className="truncate max-w-[140px]">{itemDictionary[matId]?.name || matId}</span>
                                      <div className="flex items-center gap-1 border-l border-zinc-800 pl-1.5 ml-0.5">
                                        <button
                                          type="button"
                                          title="Giảm 1"
                                          onClick={() => {
                                            if (qty <= 1) {
                                              const c = { ...offeredMats };
                                              delete c[matId];
                                              setOfferedMats(c);
                                            } else {
                                              setOfferedMats((prev) => ({ ...prev, [matId]: qty - 1 }));
                                            }
                                          }}
                                          className="w-4 h-4 rounded bg-zinc-800 hover:bg-zinc-700 flex items-center justify-center text-zinc-300 text-xs"
                                        >
                                          -
                                        </button>
                                        <button
                                          type="button"
                                          title="Tăng 1"
                                          disabled={avail <= 0}
                                          onClick={() => {
                                            if (avail > 0) {
                                              setOfferedMats((prev) => ({ ...prev, [matId]: qty + 1 }));
                                            }
                                          }}
                                          className="w-4 h-4 rounded bg-zinc-800 hover:bg-zinc-700 disabled:opacity-30 disabled:hover:bg-zinc-800 flex items-center justify-center text-zinc-300 text-xs"
                                        >
                                          +
                                        </button>
                                        <button
                                          type="button"
                                          title="Xóa"
                                          onClick={() => {
                                            const c = { ...offeredMats };
                                            delete c[matId];
                                            setOfferedMats(c);
                                          }}
                                          className="text-red-400 hover:text-red-300 ml-1"
                                        >
                                          <X className="w-3.5 h-3.5" />
                                        </button>
                                      </div>
                                    </div>
                                  );
                                })}
                              </div>
                            </div>
                          )}

                          {/* Gus */}
                          {offeredGuIDs.length > 0 && (
                            <div>
                              <div className="flex items-center justify-between mb-1.5">
                                <span className="text-[11px] font-semibold text-zinc-400 uppercase flex items-center gap-1">
                                  <Bug className="w-3 h-3 text-emerald-400" /> Cổ Trùng ({offeredGuIDs.length})
                                </span>
                                <button
                                  type="button"
                                  onClick={() => openAssetPicker("offer", "gu")}
                                  className="text-[10px] text-emerald-400 hover:text-emerald-300 font-medium"
                                >
                                  + Thêm / Sửa
                                </button>
                              </div>
                              <div className="flex flex-wrap gap-2">
                                {offeredGuIDs.map((instId) => {
                                  const g = guInventory.find((i) => i.instance_id === instId);
                                  const spec = g ? guSpecFor(g.gu_id, g.rank) : undefined;
                                  const name = spec?.name || g?.gu_id || instId;
                                  const rank = g?.rank || 1;
                                  return (
                                    <div
                                      key={instId}
                                      className="inline-flex items-center gap-2 bg-zinc-900 border border-emerald-500/40 rounded-xl px-2.5 py-1 text-xs text-emerald-200"
                                    >
                                      <GameBadge variant="rank" rank={rank} size="sm">
                                        Hạng {rank}
                                      </GameBadge>
                                      <span className="truncate max-w-[140px] font-medium">{name}</span>
                                      <button
                                        type="button"
                                        title="Xóa"
                                        onClick={() => setOfferedGuIDs((prev) => prev.filter((id) => id !== instId))}
                                        className="text-red-400 hover:text-red-300 ml-1"
                                      >
                                        <X className="w-3.5 h-3.5" />
                                      </button>
                                    </div>
                                  );
                                })}
                              </div>
                            </div>
                          )}

                          {/* Recipes */}
                          {Object.keys(offeredRecipes).length > 0 && (
                            <div>
                              <div className="flex items-center justify-between mb-1.5">
                                <span className="text-[11px] font-semibold text-zinc-400 uppercase flex items-center gap-1">
                                  <ScrollText className="w-3 h-3 text-emerald-400" /> Bí Phương ({Object.keys(offeredRecipes).length})
                                </span>
                                <button
                                  type="button"
                                  onClick={() => openAssetPicker("offer", "recipe")}
                                  className="text-[10px] text-emerald-400 hover:text-emerald-300 font-medium"
                                >
                                  + Thêm / Sửa
                                </button>
                              </div>
                              <div className="flex flex-wrap gap-2">
                                {Object.entries(offeredRecipes).map(([recId, rate]) => {
                                  const rSpec = recipes.find((r) => r.id === recId);
                                  const targetGu = rSpec?.target_gu_id ? guSpecFor(rSpec.target_gu_id, rSpec.target_rank || 1)?.name : "";
                                  const name = targetGu ? `Bí phương ${targetGu}` : recId;
                                  return (
                                    <div
                                      key={recId}
                                      className="inline-flex items-center gap-2 bg-zinc-900 border border-emerald-500/40 rounded-xl px-2.5 py-1 text-xs text-emerald-200"
                                    >
                                      <span className="truncate max-w-[140px] font-medium">{name}</span>
                                      <span className="text-[10px] text-emerald-400 font-mono">({rate}%)</span>
                                      <button
                                        type="button"
                                        title="Xóa"
                                        onClick={() => {
                                          const c = { ...offeredRecipes };
                                          delete c[recId];
                                          setOfferedRecipes(c);
                                        }}
                                        className="text-red-400 hover:text-red-300 ml-1"
                                      >
                                        <X className="w-3.5 h-3.5" />
                                      </button>
                                    </div>
                                  );
                                })}
                              </div>
                            </div>
                          )}
                        </>
                      )}
                    </div>
                  </div>
                </div>

                {/* Arrow / divider */}
                <div className="flex items-center gap-3 text-zinc-600">
                  <div className="flex-1 border-t border-dashed border-zinc-700" />
                  <span className="text-xl">⇅</span>
                  <div className="flex-1 border-t border-dashed border-zinc-700" />
                </div>

                {/* ===== PANEL: ĐỒ MUỐN NHẬN ===== */}
                <div className="rounded-xl border border-amber-500/30 overflow-hidden bg-zinc-950/60">
                  {/* Header */}
                  <div className="bg-amber-950/50 px-4 py-2.5 flex items-center justify-between border-b border-amber-500/20">
                    <div className="flex items-center gap-2">
                      <span className="text-amber-400 text-base">📥</span>
                      <span className="text-sm font-bold text-amber-300">Đồ Muốn Nhận</span>
                      <span className="text-[10px] text-zinc-500 font-normal">(người đổi phải cung cấp)</span>
                    </div>
                    <div className="flex items-center gap-1.5">
                      <GameButton
                        size="xs"
                        variant="secondary"
                        type="button"
                        onClick={() => openAssetPicker("request", "material")}
                        className="text-xs"
                      >
                        + Vật Phẩm
                      </GameButton>
                      <GameButton
                        size="xs"
                        variant="secondary"
                        type="button"
                        onClick={() => openAssetPicker("request", "gu")}
                        className="text-xs"
                      >
                        + Cổ Trùng
                      </GameButton>
                      <GameButton
                        size="xs"
                        variant="secondary"
                        type="button"
                        onClick={() => openAssetPicker("request", "recipe")}
                        className="text-xs"
                      >
                        + Bí Phương
                      </GameButton>
                    </div>
                  </div>

                  <div className="p-4 space-y-4">
                    {/* Currency */}
                    <div>
                      <p className="text-[11px] text-zinc-500 font-semibold uppercase mb-2">💰 Tiền Tệ</p>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="text-[11px] text-zinc-400 block mb-1">Nguyên Thạch</label>
                          <GameInput
                            type="number"
                            min={0}
                            value={requestedStones}
                            onChange={(e) => setRequestedStones(Number(e.target.value))}
                          />
                        </div>
                        <div>
                          <label className="text-[11px] text-zinc-400 block mb-1">Tiên Nguyên</label>
                          <GameInput
                            type="number"
                            min={0}
                            value={requestedImmortal}
                            onChange={(e) => setRequestedImmortal(Number(e.target.value))}
                          />
                        </div>
                      </div>
                    </div>

                    {/* Visual Selections Display */}
                    <div className="border-t border-zinc-800/80 pt-3 space-y-3">
                      {Object.keys(requestedMats).length === 0 && requestedGuIDs.length === 0 && Object.keys(requestedRecipes).length === 0 ? (
                        <div className="border border-dashed border-zinc-800 rounded-xl p-4 text-center">
                          <p className="text-xs text-zinc-500">Chưa có vật phẩm, cổ trùng hay bí phương nào được yêu cầu.</p>
                          <div className="flex justify-center gap-3 mt-2 text-xs">
                            <button
                              type="button"
                              onClick={() => openAssetPicker("request", "material")}
                              className="text-amber-400 hover:underline flex items-center gap-1 font-medium"
                            >
                              <Package className="w-3.5 h-3.5" /> + Yêu Cầu Vật Phẩm
                            </button>
                            <span className="text-zinc-700">•</span>
                            <button
                              type="button"
                              onClick={() => openAssetPicker("request", "gu")}
                              className="text-amber-400 hover:underline flex items-center gap-1 font-medium"
                            >
                              <Bug className="w-3.5 h-3.5" /> + Yêu Cầu Cổ Trùng
                            </button>
                            <span className="text-zinc-700">•</span>
                            <button
                              type="button"
                              onClick={() => openAssetPicker("request", "recipe")}
                              className="text-amber-400 hover:underline flex items-center gap-1 font-medium"
                            >
                              <ScrollText className="w-3.5 h-3.5" /> + Yêu Cầu Bí Phương
                            </button>
                          </div>
                        </div>
                      ) : (
                        <>
                          {/* Materials */}
                          {Object.keys(requestedMats).length > 0 && (
                            <div>
                              <div className="flex items-center justify-between mb-1.5">
                                <span className="text-[11px] font-semibold text-zinc-400 uppercase flex items-center gap-1">
                                  <Package className="w-3 h-3 text-amber-400" /> Vật Phẩm ({Object.keys(requestedMats).length})
                                </span>
                                <button
                                  type="button"
                                  onClick={() => openAssetPicker("request", "material")}
                                  className="text-[10px] text-amber-400 hover:text-amber-300 font-medium"
                                >
                                  + Thêm / Sửa
                                </button>
                              </div>
                              <div className="flex flex-wrap gap-2">
                                {Object.entries(requestedMats).map(([matId, qty]) => (
                                  <div
                                    key={matId}
                                    className="inline-flex items-center gap-2 bg-zinc-900 border border-amber-500/40 rounded-xl px-2.5 py-1 text-xs text-amber-200"
                                  >
                                    <span className="font-semibold text-white">{qty}×</span>
                                    <span className="truncate max-w-[140px]">{itemDictionary[matId]?.name || matId}</span>
                                    <div className="flex items-center gap-1 border-l border-zinc-800 pl-1.5 ml-0.5">
                                      <button
                                        type="button"
                                        title="Giảm 1"
                                        onClick={() => {
                                          if (qty <= 1) {
                                            const c = { ...requestedMats };
                                            delete c[matId];
                                            setRequestedMats(c);
                                          } else {
                                            setRequestedMats((prev) => ({ ...prev, [matId]: qty - 1 }));
                                          }
                                        }}
                                        className="w-4 h-4 rounded bg-zinc-800 hover:bg-zinc-700 flex items-center justify-center text-zinc-300 text-xs"
                                      >
                                        -
                                      </button>
                                      <button
                                        type="button"
                                        title="Tăng 1"
                                        onClick={() => {
                                          setRequestedMats((prev) => ({ ...prev, [matId]: qty + 1 }));
                                        }}
                                        className="w-4 h-4 rounded bg-zinc-800 hover:bg-zinc-700 flex items-center justify-center text-zinc-300 text-xs"
                                      >
                                        +
                                      </button>
                                      <button
                                        type="button"
                                        title="Xóa"
                                        onClick={() => {
                                          const c = { ...requestedMats };
                                          delete c[matId];
                                          setRequestedMats(c);
                                        }}
                                        className="text-red-400 hover:text-red-300 ml-1"
                                      >
                                        <X className="w-3.5 h-3.5" />
                                      </button>
                                    </div>
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}

                          {/* Gus */}
                          {requestedGuIDs.length > 0 && (
                            <div>
                              <div className="flex items-center justify-between mb-1.5">
                                <span className="text-[11px] font-semibold text-zinc-400 uppercase flex items-center gap-1">
                                  <Bug className="w-3 h-3 text-amber-400" /> Cổ Trùng ({requestedGuIDs.length})
                                </span>
                                <button
                                  type="button"
                                  onClick={() => openAssetPicker("request", "gu")}
                                  className="text-[10px] text-amber-400 hover:text-amber-300 font-medium"
                                >
                                  + Thêm / Sửa
                                </button>
                              </div>
                              <div className="flex flex-wrap gap-2">
                                {requestedGuIDs.map((gid) => {
                                  const spec = guSpecFor(gid, 1) || guDictionary?.[gid];
                                  const name = spec?.name || gid;
                                  const rank = spec?.rank || 1;
                                  return (
                                    <div
                                      key={gid}
                                      className="inline-flex items-center gap-2 bg-zinc-900 border border-amber-500/40 rounded-xl px-2.5 py-1 text-xs text-amber-200"
                                    >
                                      <GameBadge variant="rank" rank={rank} size="sm">
                                        Hạng {rank}
                                      </GameBadge>
                                      <span className="truncate max-w-[140px] font-medium">{name}</span>
                                      <button
                                        type="button"
                                        title="Xóa"
                                        onClick={() => setRequestedGuIDs((prev) => prev.filter((id) => id !== gid))}
                                        className="text-red-400 hover:text-red-300 ml-1"
                                      >
                                        <X className="w-3.5 h-3.5" />
                                      </button>
                                    </div>
                                  );
                                })}
                              </div>
                            </div>
                          )}

                          {/* Recipes */}
                          {Object.keys(requestedRecipes).length > 0 && (
                            <div>
                              <div className="flex items-center justify-between mb-1.5">
                                <span className="text-[11px] font-semibold text-zinc-400 uppercase flex items-center gap-1">
                                  <ScrollText className="w-3 h-3 text-amber-400" /> Bí Phương ({Object.keys(requestedRecipes).length})
                                </span>
                                <button
                                  type="button"
                                  onClick={() => openAssetPicker("request", "recipe")}
                                  className="text-[10px] text-amber-400 hover:text-amber-300 font-medium"
                                >
                                  + Thêm / Sửa
                                </button>
                              </div>
                              <div className="flex flex-wrap gap-2">
                                {Object.keys(requestedRecipes).map((recId) => {
                                  const rSpec = recipes.find((r) => r.id === recId);
                                  const targetGu = rSpec?.target_gu_id ? guSpecFor(rSpec.target_gu_id, rSpec.target_rank || 1)?.name : "";
                                  const name = targetGu ? `Bí phương ${targetGu}` : recId;
                                  return (
                                    <div
                                      key={recId}
                                      className="inline-flex items-center gap-2 bg-zinc-900 border border-amber-500/40 rounded-xl px-2.5 py-1 text-xs text-amber-200"
                                    >
                                      <span className="truncate max-w-[140px] font-medium">{name}</span>
                                      <button
                                        type="button"
                                        title="Xóa"
                                        onClick={() => {
                                          const c = { ...requestedRecipes };
                                          delete c[recId];
                                          setRequestedRecipes(c);
                                        }}
                                        className="text-red-400 hover:text-red-300 ml-1"
                                      >
                                        <X className="w-3.5 h-3.5" />
                                      </button>
                                    </div>
                                  );
                                })}
                              </div>
                            </div>
                          )}
                        </>
                      )}
                    </div>
                  </div>
                </div>

                {/* Target Player (Optional) */}
                {/* <div className="bg-zinc-900/40 rounded-xl p-3 border border-zinc-800 space-y-1">
                  <label className="text-xs text-zinc-400 block font-medium">🎯 Chỉ Định Đạo Hữu (Tùy chọn)</label>
                  <GameInput
                    placeholder="Nhập tên đạo hữu (bỏ trống để đăng công khai cho toàn máy chủ)..."
                    value={tradeOfferTargetUser}
                    onChange={(e) => setTradeOfferTargetUser(e.target.value)}
                    className="text-xs"
                  />
                  <span className="text-[10px] text-zinc-500 block">
                    Nếu điền tên, chỉ đạo hữu đó mới nhìn thấy và chấp nhận được đơn trao đổi này.
                  </span>
                </div> */}

                {/* Footer */}
                <div className="flex items-center gap-3 bg-zinc-900/60 rounded-lg px-4 py-2.5">
                  <span className="text-xs text-zinc-400">⏱ Thời Hạn:</span>
                  <GameInput
                    type="number"
                    min={1}
                    max={168}
                    value={tradeOfferDuration}
                    onChange={(e) => setTradeOfferDuration(Number(e.target.value))}
                    className="w-20 text-xs"
                  />
                  <span className="text-xs text-zinc-500">giờ (tối đa 168 giờ)</span>
                </div>

                <GameButton type="submit" variant="primary" className="w-full" loading={actionLoading}>
                  Đăng Đơn Trao Đổi Tự Động
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
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-start sm:items-center justify-center overflow-y-auto p-4">
          <GameCard variant="vital" className="p-6 max-w-md w-full max-h-[calc(100dvh-2rem)] overflow-y-auto space-y-4 relative">
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
              <div className="flex items-center justify-between mb-1">
                <label className="text-xs text-zinc-400">Số lượng giao dịch</label>
                <button
                  type="button"
                  onClick={() => {
                    const maxOrder = selectedBuybackOrder.quantity_requested - selectedBuybackOrder.quantity_filled;
                    if (selectedBuybackOrder.order_type === "SELL") {
                      const affordable = selectedBuybackOrder.currency === "immortal"
                        ? Math.floor(player.immortal_stones / selectedBuybackOrder.unit_price)
                        : Math.floor(player.primeval_stones / selectedBuybackOrder.unit_price);
                      setSellQtyInput(Math.max(0, Math.min(maxOrder, affordable)));
                    } else {
                      let owned = 0;
                      if (selectedBuybackOrder.item_type === "Gu") {
                        const eligibleIDs = eligibleGusForBuybackOrder(selectedBuybackOrder).map((g) => g.instance_id);
                        const next = reconcileGuSelection(eligibleIDs, selectedBuybackGuIDs, Math.min(maxOrder, eligibleIDs.length));
                        setSelectedBuybackGuIDs(next);
                        setSellQtyInput(next.length);
                        return;
                      } else if (selectedBuybackOrder.item_type === "Material") {
                        const matId = typeof selectedBuybackOrder.item_target === "string" ? selectedBuybackOrder.item_target : String(selectedBuybackOrder.item_target);
                        owned = player.materials_inventory?.[matId] || 0;
                      }
                      setSellQtyInput(Math.max(1, Math.min(maxOrder, owned > 0 ? owned : maxOrder)));
                    }
                  }}
                  className="text-[10px] font-mono text-amber-400 hover:text-amber-300 font-bold px-2 py-0.5 rounded bg-amber-950/40 border border-amber-800/40 hover:bg-amber-900/40 cursor-pointer transition-colors"
                >
                  Tối Đa
                </button>
              </div>
              <GameInput
                type="number"
                min={0}
                max={selectedBuybackOrder.quantity_requested - selectedBuybackOrder.quantity_filled}
                value={sellQtyInput}
                onChange={(e) => {
                  const maxOrder = selectedBuybackOrder.quantity_requested - selectedBuybackOrder.quantity_filled;
                  const quantity = Math.max(0, Math.min(maxOrder, Number(e.target.value) || 0));
                  if (selectedBuybackOrder.order_type !== "SELL" && selectedBuybackOrder.item_type === "Gu") {
                    const eligibleIDs = eligibleGusForBuybackOrder(selectedBuybackOrder).map((g) => g.instance_id);
                    const next = reconcileGuSelection(eligibleIDs, selectedBuybackGuIDs, quantity);
                    setSelectedBuybackGuIDs(next);
                    setSellQtyInput(next.length);
                  } else setSellQtyInput(quantity);
                }}
              />
            </div>

            {selectedBuybackOrder.order_type !== "SELL" && selectedBuybackOrder.item_type === "Gu" && (() => {
              const eligibleGus = eligibleGusForBuybackOrder(selectedBuybackOrder);
              return (
                <div className="space-y-2">
                  <label className="text-xs text-zinc-400">
                    Chọn đúng {sellQtyInput} Cổ trùng ({selectedBuybackGuIDs.length}/{sellQtyInput})
                  </label>
                  <div className="max-h-40 overflow-y-auto space-y-1 rounded-xl border border-zinc-800 bg-zinc-950 p-2">
                    {eligibleGus.length === 0 ? (
                      <p className="text-xs text-red-400">Không có Cổ trùng đúng loại và hạng để bán.</p>
                    ) : eligibleGus.map((g) => {
                      const checked = selectedBuybackGuIDs.includes(g.instance_id);
                      return (
                        <label key={g.instance_id} className="flex cursor-pointer items-center gap-2 rounded-lg p-2 text-xs text-zinc-300 hover:bg-zinc-900">
                          <input type="checkbox" checked={checked} onChange={() => {
                            const maxOrder = selectedBuybackOrder.quantity_requested - selectedBuybackOrder.quantity_filled;
                            const toggled = toggleGuSelection(selectedBuybackGuIDs, g.instance_id);
                            const next = reconcileGuSelection(
                              eligibleGus.map((candidate) => candidate.instance_id),
                              toggled,
                              Math.min(toggled.length, maxOrder),
                            );
                            setSelectedBuybackGuIDs(next);
                            setSellQtyInput(next.length);
                          }} disabled={!checked && selectedBuybackGuIDs.length >= selectedBuybackOrder.quantity_requested - selectedBuybackOrder.quantity_filled} />
                          <span>{guSpecFor(g.gu_id, g.rank)?.name || g.gu_id} (Hạng {g.rank})</span>
                          <span className="ml-auto font-mono text-zinc-600">{g.instance_id.slice(0, 8)}</span>
                        </label>
                      );
                    })}
                  </div>
                </div>
              );
            })()}

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
                onClick={() => {
                  setSelectedBuybackOrder(null);
                  setSelectedBuybackGuIDs([]);
                }}
              >
                Hủy Thao Tác
              </GameButton>
              <GameButton
                variant="primary"
                className="flex-1"
                loading={actionLoading}
                disabled={sellQtyInput < 1 || (selectedBuybackOrder.order_type !== "SELL" && selectedBuybackOrder.item_type === "Gu" && selectedBuybackGuIDs.length !== sellQtyInput)}
                onClick={async () => {
                  const payload = buildBuyOrderExecution(
                    selectedBuybackOrder.id,
                    sellQtyInput,
                    selectedBuybackOrder.item_type,
                    selectedBuybackGuIDs,
                    selectedBuybackOrder.order_type || "BUY",
                  );
                  if (await sellToBuybackOrder(payload)) {
                    setSelectedBuybackOrder(null);
                    setSelectedBuybackGuIDs([]);
                  }
                }}
              >
                {selectedBuybackOrder.order_type === "SELL" ? "Xác Nhận Mua" : "Xác Nhận Bán"}
              </GameButton>
            </div>
          </GameCard>
        </div>
      )}
      {/* MODAL: CUSTOM BID FOR AUCTION LISTING */}
      {biddingListing && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
          <GameCard variant="vital" className="p-6 max-w-md w-full space-y-4 relative">
            <h3 className="text-lg font-bold font-serif text-amber-200">
              Đặt Giá Đấu Giá Bảo Vật
            </h3>
            <p className="text-xs text-zinc-400">
              {getListingDetails(biddingListing).title}
            </p>
            <div className="bg-zinc-950 p-3 rounded-xl border border-zinc-800 text-xs font-mono space-y-1">
              <div className="text-zinc-400">
                • Giá hiện tại: <span className="text-amber-400 font-bold">{biddingListing.current_price} {biddingListing.currency === "immortal" ? "Tiên Nguyên" : "Nguyên Thạch"}</span>
              </div>
              <div className="text-zinc-400">
                • Yêu cầu mức giá tối thiểu: <span className="text-emerald-400 font-bold">{biddingListing.current_price + 1}</span>
              </div>
            </div>

            <div>
              <label className="text-xs text-zinc-400 block mb-1">Mức giá đặt mới</label>
              <GameInput
                type="number"
                min={biddingListing.current_price + 1}
                value={customBidPrice}
                onChange={(e) => setCustomBidPrice(Number(e.target.value))}
              />
            </div>

            <div className="flex items-center gap-3 pt-2">
              <GameButton
                variant="ghost"
                className="flex-1"
                onClick={() => setBiddingListing(null)}
              >
                Hủy
              </GameButton>
              <GameButton
                variant="primary"
                className="flex-1"
                loading={actionLoading}
                disabled={customBidPrice <= biddingListing.current_price}
                onClick={async () => {
                  if (customBidPrice > biddingListing.current_price) {
                    await onBid(biddingListing.id, customBidPrice);
                    setBiddingListing(null);
                  }
                }}
              >
                Xác Nhận Đặt Giá
              </GameButton>
            </div>
          </GameCard>
        </div>
      )}

      {/* MODAL: VISUAL ASSET PICKER FOR TRADE OFFERS */}
      <TradeAssetPickerModal
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        mode={pickerMode}
        initialTab={pickerTab}
        player={player}
        guInventory={guInventory}
        recipes={recipes}
        itemDictionary={itemDictionary}
        guDictionary={guDictionary}
        offeredMats={offeredMats}
        offeredGuIDs={offeredGuIDs}
        offeredRecipes={offeredRecipes}
        requestedMats={requestedMats}
        requestedGuIDs={requestedGuIDs}
        requestedRecipes={requestedRecipes}
        onAddMaterial={handleAddMaterialFromPicker}
        onAddGu={handleAddGuFromPicker}
        onAddRecipe={handleAddRecipeFromPicker}
      />
    </GamePanel>
  );
}
