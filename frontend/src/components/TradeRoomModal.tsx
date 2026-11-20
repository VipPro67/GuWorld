"use client";

import React, { useEffect, useState } from "react";
import { useGameStore, GuInstance, formatRecipeName, guSpecFor } from "@/store/useGameStore";
import {
  X, Coins, Sparkles, Lock, Unlock, CheckCircle2,
  Trash2, Plus, Minus, RefreshCw, User, ShieldAlert, ScrollText
} from "lucide-react";
import { GameModal, GameTabs, GameButton, GameCard, GameBadge, GameInput } from "@/components/ui";
import { getGuIconUrl } from "@/lib/gu-icons";
import { getItemIconUrl } from "@/lib/item-icons";
// UI migrated to shared components

export const TradeRoomModal: React.FC = () => {
  const {
    player,
    activeTrade,
    cancelTrade,
    updateTradeOffer,
    lockTrade,
    unlockTrade,
    confirmTrade,
    guDictionary,
    itemDictionary,
    recipes,
    fetchRecipes,
    showConfirm,
  } = useGameStore();

  useEffect(() => {
    fetchRecipes();
  }, [fetchRecipes]);

  const [primevalInput, setPrimevalInput] = useState<number>(0);
  const [immortalInput, setImmortalInput] = useState<number>(0);
  const [selectedMat, setSelectedMat] = useState<string>("");
  const [selectedMatQty, setSelectedMatQty] = useState<number>(1);
  const [selectedRecipe, setSelectedRecipe] = useState<string>("");
  const [activeTab, setActiveTab] = useState<"stones" | "gus" | "materials" | "recipes">("gus");
  const [assetPickerOpen, setAssetPickerOpen] = useState(false);
  const [expandedGus, setExpandedGus] = useState<Record<string, boolean>>({});
  const [tradeFilterRank, setTradeFilterRank] = useState<number | "all">("all");
  const [searchTerm, setSearchTerm] = useState<string>("");
  const toggleGuGroup = (guId: string) => {
    setExpandedGus((prev) => ({ ...prev, [guId]: !prev[guId] }));
  };

  const isSender = player ? player.id === activeTrade?.sender_id : false;
  const myOffer = activeTrade ? (isSender ? activeTrade.sender_offer : activeTrade.receiver_offer) : null;

  // Initialize stone inputs from database offer
  useEffect(() => {
    if (myOffer) {

      setPrimevalInput(myOffer.primeval_stones || 0);
      setImmortalInput(myOffer.immortal_stones || 0);
    }
  }, [activeTrade, myOffer]);

  // Safety checks
  if (!player || !activeTrade) return null;

  // Destructure statuses and offers
  const myLocked = isSender ? activeTrade.sender_locked : activeTrade.receiver_locked;
  const myConfirmed = isSender ? activeTrade.sender_confirmed : activeTrade.receiver_confirmed;

  const oppOffer = isSender ? activeTrade.receiver_offer : activeTrade.sender_offer;
  const oppLocked = isSender ? activeTrade.receiver_locked : activeTrade.sender_locked;
  const oppConfirmed = isSender ? activeTrade.receiver_confirmed : activeTrade.sender_confirmed;
  const oppUsername = isSender ? activeTrade.receiver_username : activeTrade.sender_username;

  // Sync Stone offers to backend
  const syncStones = () => {
    if (myLocked) return;

    // Clamp values
    const clampedPrimeval = Math.max(0, Math.min(player.primeval_stones, primevalInput));
    const clampedImmortal = Math.max(0, Math.min(player.immortal_stones, immortalInput));

    setPrimevalInput(clampedPrimeval);
    setImmortalInput(clampedImmortal);

    const updatedOffer = {
      primeval_stones: clampedPrimeval,
      immortal_stones: clampedImmortal,
      gus: myOffer?.gus || [],
      materials: myOffer?.materials || {},
      recipes: myOffer?.recipes || {},
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
  };

  // Sync Gu selections
  const toggleGu = (guInstanceId: string) => {
    if (myLocked) return;

    const currentGus = myOffer?.gus || [];
    let updatedGus: string[];

    if (currentGus.includes(guInstanceId)) {
      updatedGus = currentGus.filter((id: string) => id !== guInstanceId);
    } else {
      updatedGus = [...currentGus, guInstanceId];
    }

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: updatedGus,
      materials: myOffer?.materials || {},
      recipes: myOffer?.recipes || {},
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
  };

  // Sync Material offers
  const addMaterial = () => {
    if (myLocked || !selectedMat) return;

    const currentMats = { ...(myOffer?.materials || {}) };
    const maxQty = player.materials_inventory[selectedMat] || 0;
    const currentQty = currentMats[selectedMat] || 0;
    const targetQty = Math.max(1, Math.min(maxQty, currentQty + selectedMatQty));

    currentMats[selectedMat] = targetQty;

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: myOffer?.gus || [],
      materials: currentMats,
      recipes: myOffer?.recipes || {},
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
    setSelectedMat("");
    setSelectedMatQty(1);
  };

  const removeMaterial = (materialId: string) => {
    if (myLocked) return;

    const currentMats = { ...(myOffer?.materials || {}) };
    delete currentMats[materialId];

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: myOffer?.gus || [],
      materials: currentMats,
      recipes: myOffer?.recipes || {},
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
  };

  const adjustMaterialQty = (materialId: string, delta: number) => {
    if (myLocked) return;

    const currentMats = { ...(myOffer?.materials || {}) };
    const currentQty = currentMats[materialId] || 0;
    const maxQty = player.materials_inventory[materialId] || 0;
    const newQty = currentQty + delta;

    if (newQty <= 0) {
      delete currentMats[materialId];
    } else {
      currentMats[materialId] = Math.min(maxQty, newQty);
    }

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: myOffer?.gus || [],
      materials: currentMats,
      recipes: myOffer?.recipes || {},
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
  };

  // Recipe helpers
  const getRecipeName = (recipeId: string) => {
    return formatRecipeName(recipeId);
  };

  const addRecipe = () => {
    if (myLocked || !selectedRecipe) return;

    const currentRecipes = { ...(myOffer?.recipes || {}) };
    const recipe = recipes.find((r) => r.id === selectedRecipe);
    if (!recipe) return;
    currentRecipes[selectedRecipe] = recipe.completion_rate ?? 100;

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: myOffer?.gus || [],
      materials: myOffer?.materials || {},
      recipes: currentRecipes,
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
    setSelectedRecipe("");
  };

  const removeRecipe = (recipeId: string) => {
    if (myLocked) return;

    const currentRecipes = { ...(myOffer?.recipes || {}) };
    delete currentRecipes[recipeId];

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: myOffer?.gus || [],
      materials: myOffer?.materials || {},
      recipes: currentRecipes,
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
  };

  // Helpers
  const getGuName = (identifier: string) => {
    // If it is an instance ID found in gu_details
    const guIdFromDetails = activeTrade.gu_details?.[identifier];
    if (guIdFromDetails) {
      return guDictionary[guIdFromDetails]?.name || guIdFromDetails.replace(/^GU_/, "").replace(/_/g, " ");
    }
    // If it is a spec ID directly
    if (guDictionary[identifier]) {
      return guDictionary[identifier].name;
    }
    // If it is an instance ID from local inventory
    const localGu = (player.gu_inventory || []).find((g) => g.instance_id === identifier);
    if (localGu) {
      return guSpecFor(localGu.gu_id, localGu.rank)?.name || localGu.gu_id.replace(/^GU_/, "").replace(/_/g, " ");
    }
    // Fallback
    return identifier.replace(/^GU_/, "").replace(/_/g, " ");
  };

  const getGuRank = (identifier: string) => {
    const rankFromDetails = activeTrade.gu_ranks?.[identifier];
    if (typeof rankFromDetails === "number" && rankFromDetails > 0) {
      return rankFromDetails;
    }

    const guIdFromDetails = activeTrade.gu_details?.[identifier];
    if (guIdFromDetails) {
      return guDictionary[guIdFromDetails]?.rank || 1;
    }
    // If it is a spec ID directly
    if (guDictionary[identifier]) {
      return guDictionary[identifier].rank || 1;
    }
    // If it is an instance ID from local inventory
    const localGu = (player.gu_inventory || []).find((g) => g.instance_id === identifier);
    if (localGu) {
      return localGu.rank || guDictionary[localGu.gu_id]?.rank || 1;
    }
    return 1;
  };

  const getMatName = (matId: string) => {
    return itemDictionary[matId]?.name || matId;
  };

  // Filter list of materials in inventory that are not fully added yet
  const availableMaterials = Object.keys(player.materials_inventory || {}).filter(
    (matId) => {
      const playerQty = player.materials_inventory[matId] || 0;
      const offeredQty = myOffer?.materials?.[matId] || 0;
      if (playerQty <= offeredQty) return false;
      if (tradeFilterRank !== "all") {
        const rank = itemDictionary[matId]?.rank || 0;
        if (rank !== tradeFilterRank) return false;
      }
      return true;
    }
  );

  // Filter player Gus (Exclude Vital Gus, Equipped Gus and Dead Gus)
  const tradeableGus = (player.gu_inventory || []).filter(
    (g) => {
      if (g.is_vital || g.equipped || g.status === "DEAD") return false;
      if (tradeFilterRank !== "all") {
        const rank = g.rank || guDictionary[g.gu_id]?.rank || 0;
        if (rank !== tradeFilterRank) return false;
      }
      return true;
    }
  );

  // Search-filtered lists for each tab
  const searchLower = searchTerm.trim().toLowerCase();
  const filteredGus = searchLower
    ? tradeableGus.filter(g => {
        const name = guSpecFor(g.gu_id, g.rank)?.name || g.gu_id;
        return name.toLowerCase().includes(searchLower);
      })
    : tradeableGus;

  const filteredMaterials = searchLower
    ? availableMaterials.filter(matId => getMatName(matId).toLowerCase().includes(searchLower))
    : availableMaterials;

  const filteredRecipes = searchLower
    ? recipes.filter(r => !myOffer?.recipes?.[r.id]).filter(r => formatRecipeName(r.id).toLowerCase().includes(searchLower))
    : recipes.filter(r => !myOffer?.recipes?.[r.id]);

  return (
    <GameModal
      open={!!activeTrade && !!player}
      onClose={async () => {
        if (await showConfirm("Đạo hữu có chắc chắn muốn hủy kết nối giao dịch này không?")) {
          cancelTrade(activeTrade.id);
        }
      }}
      title="Không Gian Giao Dịch Thông Thiên"
      description="Được liên kết an toàn qua tần số Thông Thiên Cổ"
      artwork="/assets/world/cultivator.webp"
      size="6xl"
      showCloseButton={true}
    >
      <div className="w-full max-w-screen-xl bg-zinc-950 border border-zinc-900 rounded-3xl overflow-hidden shadow-2xl flex flex-col max-h-[95vh]">

        {/* Main Workspace split */}
        <div className="flex-1 overflow-y-auto grid grid-cols-1 md:grid-cols-2 divide-y md:divide-y-0 md:divide-x divide-zinc-900">

          {/* LEFT SIDE: Player (You) */}
          <div className="p-6 flex flex-col gap-5 bg-zinc-950/40">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-serif font-bold text-zinc-300 flex items-center gap-1.5">
                <User className="w-4 h-4 text-emerald-400" />
                Đạo Hữu
              </h4>
              <div className="flex items-center gap-1.5">
                {myLocked ? (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-emerald-950/50 text-emerald-400 border border-emerald-500/30 font-mono flex items-center gap-1 animate-pulse">
                    <Lock className="w-3 h-3" /> Đã Khóa
                  </span>
                ) : (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-zinc-900 text-zinc-400 border border-zinc-800 font-mono flex items-center gap-1">
                    <Unlock className="w-3 h-3" /> Đang Biên Soạn
                  </span>
                )}
                {myConfirmed && (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-blue-950/50 text-blue-400 border border-blue-500/30 font-mono flex items-center gap-1">
                    <CheckCircle2 className="w-3 h-3" /> Xác Nhận
                  </span>
                )}
              </div>
            </div>

            {/* Current Offer Display Box */}
            <div className="bg-zinc-950 border border-zinc-900/80 rounded-2xl p-4 space-y-3 min-h-[120px] shadow-inner">
              <span className="text-[9px] uppercase tracking-wider text-zinc-500 font-mono block">Ưu đãi hiện tại của bạn:</span>

              {/* Stones offered */}
              <div className="flex flex-wrap gap-2 text-xs font-mono">
                {((myOffer?.primeval_stones || 0) > 0) && (
                  <span className="px-2.5 py-1.5 rounded-xl bg-yellow-950/35 border border-yellow-500/20 text-yellow-500 flex items-center gap-1">
                    <Coins className="w-3.5 h-3.5" />
                    {(myOffer.primeval_stones).toLocaleString()} Nguyên Thạch
                  </span>
                )}
                {((myOffer?.immortal_stones || 0) > 0) && (
                  <span className="px-2.5 py-1.5 rounded-xl bg-purple-950/35 border border-purple-500/20 text-purple-400 flex items-center gap-1">
                    <Sparkles className="w-3.5 h-3.5" />
                    {(myOffer.immortal_stones).toFixed(1)} Tiên Nguyên
                  </span>
                )}
              </div>

              {/* Gus offered */}
              {myOffer?.gus && myOffer.gus.length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Cổ trùng ({myOffer.gus.length}):</span>
                  <div className="flex flex-wrap gap-1.5">
                    {(myOffer.gus as string[]).map((instId: string) => {
                      const guName = getGuName(instId);
                      const guRank = getGuRank(instId);
                      const guId = activeTrade.gu_details?.[instId] || player.gu_inventory?.find((g) => g.instance_id === instId)?.gu_id || instId;
                      return (
                        <span key={instId} className="text-xs px-2.5 py-1 rounded-xl bg-emerald-950/30 border border-emerald-900/40 text-emerald-400 flex items-center gap-1.5">
                          <img src={getGuIconUrl(guId, guName, guDictionary[guId]?.image_url)} alt="" className="h-6 w-6 rounded-md bg-zinc-950 object-contain p-0.5" />
                          <span className="max-w-full truncate">{guName}</span>
                          <span className="text-[9px] font-mono opacity-80 px-1 bg-emerald-900/20 border border-emerald-800/35 rounded text-emerald-300">
                            Rank {guRank}
                          </span>
                          {!myLocked && (
                            <button type="button" aria-label={`Bỏ ${guName} khỏi đề nghị`} onClick={() => toggleGu(instId)} className="ml-1 flex min-h-11 min-w-11 items-center justify-center hover:text-red-400 cursor-pointer">
                              <X className="w-3 h-3" />
                            </button>
                          )}
                        </span>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Materials offered */}
              {myOffer?.materials && Object.keys(myOffer.materials).length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Nguyên liệu:</span>
                  <div className="flex flex-wrap gap-1.5">
                    {Object.entries(myOffer.materials as Record<string, number>).map(([matId, qty]) => (
                      <span key={matId} className="text-xs px-2.5 py-1 rounded-xl bg-zinc-900 border border-zinc-800 text-zinc-300 flex items-center gap-1.5 max-w-full truncate">
                        <img src={getItemIconUrl(itemDictionary[matId] ?? { id: matId, name: getMatName(matId), item_type: "Material" })} alt="" className="h-5 w-5 shrink-0 rounded object-contain" />
                        {getMatName(matId)} (x{qty})
                        {!myLocked && (
                          <div className="flex items-center gap-1 border-l border-zinc-800 pl-1.5 ml-1">
                            <button type="button" aria-label={`Giảm số lượng ${getMatName(matId)}`} onClick={() => adjustMaterialQty(matId, -1)} className="flex min-h-11 min-w-11 items-center justify-center hover:text-amber-500 cursor-pointer">
                              <Minus className="w-2.5 h-2.5" />
                            </button>
                            <button type="button" aria-label={`Tăng số lượng ${getMatName(matId)}`} onClick={() => adjustMaterialQty(matId, 1)} className="flex min-h-11 min-w-11 items-center justify-center hover:text-amber-500 cursor-pointer">
                              <Plus className="w-2.5 h-2.5" />
                            </button>
                            <button type="button" aria-label={`Bỏ ${getMatName(matId)} khỏi đề nghị`} onClick={() => removeMaterial(matId)} className="flex min-h-11 min-w-11 items-center justify-center hover:text-red-400 cursor-pointer pl-1 text-red-500/80">
                              <Trash2 className="w-2.5 h-2.5" />
                            </button>
                          </div>
                        )}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Recipes offered */}
              {myOffer?.recipes && Object.keys(myOffer.recipes).length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Bí phương:</span>
                  <div className="flex flex-wrap gap-1.5">
                    {Object.entries(myOffer.recipes as Record<string, number>).map(([recipeId, rate]) => (
                      <span key={recipeId} className="text-xs px-2.5 py-1 rounded-xl bg-violet-950/30 border border-violet-500/30 text-violet-300 flex items-center gap-1.5 max-w-full truncate">
                        <ScrollText className="h-4 w-4 shrink-0" />
                        {getRecipeName(recipeId)} ({rate.toFixed(0)}%)
                        {!myLocked && (
                          <button type="button" aria-label={`Bỏ ${getRecipeName(recipeId)} khỏi đề nghị`} onClick={() => removeRecipe(recipeId)} className="ml-1 flex min-h-11 min-w-11 items-center justify-center hover:text-red-400 cursor-pointer">
                            <X className="w-3 h-3" />
                          </button>
                        )}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {!(myOffer?.primeval_stones || 0) && !(myOffer?.immortal_stones || 0) && (!myOffer?.gus || myOffer.gus.length === 0) && (!myOffer?.materials || Object.keys(myOffer.materials).length === 0) && (!myOffer?.recipes || Object.keys(myOffer.recipes).length === 0) && (
                <p className="text-xs text-zinc-600 italic">Chưa chọn vật phẩm nào để giao dịch</p>
              )}
            </div>

            {!myLocked && (
              <button
                type="button"
                onClick={() => setAssetPickerOpen(true)}
                className="flex w-full items-center justify-between gap-3 rounded-2xl border border-emerald-500/25 bg-emerald-950/15 p-4 text-left transition-colors hover:border-emerald-400/50 hover:bg-emerald-950/25"
              >
                <span className="flex min-w-0 items-center gap-3">
                  <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-300"><Plus className="h-5 w-5" /></span>
                  <span className="min-w-0"><span className="block text-sm font-semibold text-emerald-200">Thêm tài sản vào giao dịch</span><span className="block truncate text-xs text-zinc-500">Chọn Cổ trùng, vật phẩm, bí phương hoặc tiền tệ</span></span>
                </span>
                <span className="shrink-0 rounded-lg border border-emerald-500/20 px-2.5 py-1 text-[10px] font-mono text-emerald-300">{(myOffer?.gus?.length || 0) + Object.keys(myOffer?.materials || {}).length + Object.keys(myOffer?.recipes || {}).length} tài sản</span>
              </button>
            )}

            {/* Asset picker dialog */}
            {assetPickerOpen && (
              <GameModal
                open={assetPickerOpen}
                onClose={() => setAssetPickerOpen(false)}
                title="Chọn tài sản giao dịch"
                description="Chọn từng cá thể Cổ trùng, đặt số lượng vật phẩm hoặc chọn bí phương. Thay đổi được áp dụng ngay vào đề nghị; Hoàn Tất chỉ đóng bộ chọn."
                size="5xl"
                variant="success"
                footer={<GameButton variant="primary" onClick={() => setAssetPickerOpen(false)}>Hoàn Tất</GameButton>}
              >
            {activeTab !== "stones" && (
              <input
                type="text"
                placeholder="Tìm kiếm..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full p-2 mb-2 bg-zinc-950 border border-zinc-900 rounded-xl text-xs text-zinc-300 focus:outline-none"
              />
            )}
              {/* Selectors tabs */}
              <div className="flex border-b border-zinc-900 text-xs font-mono bg-zinc-950">
                <button
                  onClick={() => { setSearchTerm(""); setActiveTab("stones"); }}
                  className={`flex-1 py-2.5 border-b-2 font-bold cursor-pointer transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] ${
                    activeTab === "stones"
                      ? "text-amber-400 border-amber-500/40 bg-amber-950/5"
                      : "text-zinc-500 border-transparent hover:text-zinc-300"
                  }`}
                >
                  Linh Thạch
                </button>
                <button
                  onClick={() => { setSearchTerm(""); setActiveTab("gus"); }}
                  className={`flex-1 py-2.5 border-b-2 font-bold cursor-pointer transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] ${
                    activeTab === "gus"
                      ? "text-emerald-400 border-emerald-500/40 bg-emerald-950/5"
                      : "text-zinc-500 border-transparent hover:text-zinc-300"
                  }`}
                >
                  Cổ Trùng
                </button>
                <button
                  onClick={() => { setSearchTerm(""); setActiveTab("materials"); }}
                  className={`flex-1 py-2.5 border-b-2 font-bold cursor-pointer transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] ${
                    activeTab === "materials"
                      ? "text-violet-400 border-violet-500/40 bg-violet-950/5"
                      : "text-zinc-500 border-transparent hover:text-zinc-300"
                  }`}
                >
                  Nguyên Liệu
                </button>
                <button
                  onClick={() => { setSearchTerm(""); setActiveTab("recipes"); }}
                  className={`flex-1 py-2.5 border-b-2 font-bold cursor-pointer transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] ${
                    activeTab === "recipes"
                      ? "text-pink-400 border-pink-500/40 bg-pink-950/5"
                      : "text-zinc-500 border-transparent hover:text-zinc-300"
                  }`}
                >
                  Bí Phương
                </button>
              </div>

              {/* Selector Content Area */}
              <div className="flex-1 p-4 overflow-y-auto max-h-[55dvh]">
                {myLocked ? (
                  <div className="h-full flex items-center justify-center text-center flex-col gap-1 text-zinc-500 font-serif">
                    <Lock className="w-5 h-5 text-zinc-650" />
                    <p className="text-xs">Giao dịch đã được khóa</p>
                    <p className="text-[10px] text-zinc-660 font-mono">Vui lòng hủy khóa nếu muốn sửa ưu đãi</p>
                  </div>
                ) : (
                  <>
                    {/* Rank Filter if in gus or materials tab */}
                    {(activeTab === "gus" || activeTab === "materials") && (
                      <div className="flex items-center gap-2 mb-3 bg-zinc-950/60 p-2.5 rounded-xl border border-zinc-900">
                        <span className="text-[10px] text-zinc-400 font-serif font-bold uppercase tracking-wider">Lọc theo Rank:</span>
                        <select
                          value={tradeFilterRank}
                          onChange={(e) => setTradeFilterRank(e.target.value === "all" ? "all" : Number(e.target.value))}
                          className="bg-zinc-950 border border-zinc-900 rounded-lg text-[10px] px-2.5 py-1 focus:outline-none font-mono text-zinc-300 cursor-pointer"
                        >
                          <option value="all">Tất cả Rank</option>
                          {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((r) => (
                            <option key={r} value={r}>Rank {r}</option>
                          ))}
                        </select>
                      </div>
                    )}

                    {/* Stones selectors */}
                    {activeTab === "stones" && (
                      <div className="space-y-4">
                        {/* Primeval Stones */}
                        <div className="space-y-1.5">
                          <div className="flex justify-between items-center text-[10px] font-mono text-zinc-500">
                            <span className="flex items-center gap-1 text-yellow-500/95"><Coins className="w-3 h-3" /> Nguyên Thạch</span>
                            <div className="flex items-center gap-2">
                              <span>Sở hữu: {(player.primeval_stones).toLocaleString()}</span>
                              <button
                                type="button"
                                onClick={() => {
                                  setPrimevalInput(player.primeval_stones);
                                  const updatedOffer = {
                                    primeval_stones: player.primeval_stones,
                                    immortal_stones: myOffer?.immortal_stones || 0,
                                    gus: myOffer?.gus || [],
                                    materials: myOffer?.materials || {},
                                    recipes: myOffer?.recipes || {},
                                  };
                                  updateTradeOffer(activeTrade.id, updatedOffer);
                                }}
                                className="text-[9px] font-mono font-bold text-amber-400 hover:text-amber-300 px-1.5 py-0.5 rounded bg-amber-950/40 border border-amber-800/40 cursor-pointer"
                              >
                                Tối đa
                              </button>
                            </div>
                          </div>
                          <input
                            type="number"
                            value={primevalInput}
                            step="0.001"
                            onChange={(e) => setPrimevalInput(parseFloat(e.target.value) || 0)}
                            onBlur={syncStones}
                            onKeyDown={(e) => e.key === "Enter" && syncStones()}
                            min={0}
                            max={player.primeval_stones}
                            className="w-full bg-zinc-950 border border-zinc-900 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:border-zinc-800 font-mono"
                          />
                        </div>

                        {/* Immortal Stones (TIEN NGUYEN THACH - for Rank >= 6) */}
                        {player.rank >= 6 && (
                          <div className="space-y-1.5">
                            <div className="flex justify-between items-center text-[10px] font-mono text-zinc-500">
                              <span className="flex items-center gap-1 text-purple-400"><Sparkles className="w-3 h-3" /> Tiên Nguyên Thạch</span>
                              <div className="flex items-center gap-2">
                                <span>Sở hữu: {player.immortal_stones.toLocaleString("vi-VN", { maximumFractionDigits: 3 })}</span>
                                <button
                                  type="button"
                                  onClick={() => {
                                    setImmortalInput(player.immortal_stones);
                                    const updatedOffer = {
                                      primeval_stones: myOffer?.primeval_stones || 0,
                                      immortal_stones: player.immortal_stones,
                                      gus: myOffer?.gus || [],
                                      materials: myOffer?.materials || {},
                                      recipes: myOffer?.recipes || {},
                                    };
                                    updateTradeOffer(activeTrade.id, updatedOffer);
                                  }}
                                  className="text-[9px] font-mono font-bold text-amber-400 hover:text-amber-300 px-1.5 py-0.5 rounded bg-amber-950/40 border border-amber-800/40 cursor-pointer"
                                >
                                  Tối đa
                                </button>
                              </div>
                            </div>
                            <input
                              type="number"
                              step="0.001"
                              value={immortalInput}
                              onChange={(e) => setImmortalInput(parseFloat(e.target.value) || 0)}
                              onBlur={syncStones}
                              onKeyDown={(e) => e.key === "Enter" && syncStones()}
                              min={0}
                              max={player.immortal_stones}
                              className="w-full bg-zinc-950 border border-zinc-900 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:border-zinc-800 font-mono"
                            />
                          </div>
                        )}
                      </div>
                    )}

                    {/* Gus selectors */}
                    {activeTab === "gus" && (
                      <div className="space-y-1.5">
                        {/* Quick select bar */}
                        {!myLocked && tradeableGus.length > 0 && (
                          <div className="flex items-center gap-1.5 flex-wrap bg-zinc-950/40 border border-zinc-900 rounded-xl p-2">
                            <span className="text-[9px] font-mono text-zinc-500">Chọn nhanh:</span>
                            {Array.from(new Set(tradeableGus.map(g => g.rank || guDictionary[g.gu_id]?.rank || 1))).sort((a,b)=>a-b).map(rank => (
                              <button
                                key={rank}
                                type="button"
                                onClick={() => {
                                  const rankGus = tradeableGus.filter(g => (g.rank || guDictionary[g.gu_id]?.rank || 1) === rank).map(g => g.instance_id);
                                  const currentGus = [...(myOffer?.gus || [])];
                                  const updatedGus = Array.from(new Set([...currentGus, ...rankGus]));
                                  updateTradeOffer(activeTrade.id, {
                                    primeval_stones: myOffer?.primeval_stones || 0,
                                    immortal_stones: myOffer?.immortal_stones || 0,
                                    gus: updatedGus,
                                    materials: myOffer?.materials || {},
                                    recipes: myOffer?.recipes || {},
                                  });
                                }}
                                className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 border border-zinc-800 transition-colors cursor-pointer"
                              >
                                Rank {rank}
                              </button>
                            ))}
                            <button
                              type="button"
                              onClick={() => {
                                const allInstanceIds = tradeableGus.map(g => g.instance_id);
                                const updatedOffer = {
                                  primeval_stones: myOffer?.primeval_stones || 0,
                                  immortal_stones: myOffer?.immortal_stones || 0,
                                  gus: allInstanceIds,
                                  materials: myOffer?.materials || {},
                                  recipes: myOffer?.recipes || {},
                                };
                                updateTradeOffer(activeTrade.id, updatedOffer);
                              }}
                              className="px-2 py-0.5 text-[9px] font-mono rounded bg-emerald-950/20 hover:bg-emerald-950/30 text-emerald-400 border border-emerald-900/30 transition-colors cursor-pointer"
                            >
                              Tất cả Cổ trùng
                            </button>
                            <button
                              type="button"
                              onClick={() => {
                                const updatedOffer = {
                                  primeval_stones: myOffer?.primeval_stones || 0,
                                  immortal_stones: myOffer?.immortal_stones || 0,
                                  gus: [],
                                  materials: myOffer?.materials || {},
                                  recipes: myOffer?.recipes || {},
                                };
                                updateTradeOffer(activeTrade.id, updatedOffer);
                              }}
                              className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-zinc-300 border border-zinc-800 transition-colors cursor-pointer"
                            >
                              Bỏ chọn hết
                            </button>
                          </div>
                        )}
                        {tradeableGus.length > 0 ? (
                          (() => {
                            const grouped: Record<string, GuInstance[]> = {};
                            filteredGus.forEach((gu) => {
                              const rank = gu.rank || guDictionary[gu.gu_id]?.rank || 1;
                              const groupKey = `${gu.gu_id}:${rank}`;
                              if (!grouped[groupKey]) grouped[groupKey] = [];
                              grouped[groupKey].push(gu);
                            });

                            const entries = Object.entries(grouped);
                            if (entries.length === 0 && searchLower) {
                              return <p className="text-xs text-zinc-600 italic text-center py-4">Không tìm thấy Cổ trùng nào phù hợp</p>;
                            }

                            return entries.map(([groupKey, list]) => {
                              const guId = list[0].gu_id;
                              const rank = list[0].rank || guDictionary[guId]?.rank || 1;
                              const name = guSpecFor(guId, rank)?.name || guId;
                              const isExpanded = !!expandedGus[groupKey];

                              // Count how many instances in this group are already selected in myOffer
                              const selectedCountInGroup = list.filter(g => (myOffer?.gus || []).includes(g.instance_id)).length;

                              return (
                                <div key={guId} className="border border-zinc-900 bg-zinc-950/40 rounded-xl overflow-hidden">
                                  {/* Group Header Row */}
                                  <div
                                    onClick={() => toggleGuGroup(groupKey)}
                                    className="p-3 flex justify-between items-center cursor-pointer hover:bg-zinc-950/80 transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] select-none font-serif text-xs"
                                  >
                                    <div className="flex items-center gap-2 flex-wrap">
                                      <img src={getGuIconUrl(guId, name, guDictionary[guId]?.image_url)} alt="" className="h-9 w-9 rounded-lg border border-emerald-500/15 bg-zinc-900 object-contain p-1" />
                                      <span className="font-bold text-zinc-200">{name}</span>
                                      <span className="text-[10px] font-mono px-1.5 py-0.2 rounded-full bg-zinc-900 border border-zinc-850 text-zinc-400">
                                        x{list.length}
                                      </span>
                                      <span className="text-[8px] font-mono border border-amber-500/20 px-1 py-0.2 rounded text-amber-400 uppercase">
                                        Rank {rank}
                                      </span>
                                      {selectedCountInGroup > 0 && (
                                        <span className="text-[8px] font-mono font-bold text-emerald-400 bg-emerald-950/20 border border-emerald-500/25 px-1 py-0.2 rounded uppercase animate-pulse">
                                          Đã chọn {selectedCountInGroup}
                                        </span>
                                      )}
                                    </div>
                                    <span className="text-[10px] font-mono text-zinc-500">
                                      {isExpanded ? "Thu gọn" : "Xem cá thể"}
                                    </span>
                                  </div>

                                  {/* Expandable list of instances */}
                                  {isExpanded && (
                                    <div className="bg-zinc-950 border-t border-zinc-900 p-2 space-y-1.5">
                                      {list.map((gu) => {
                                        const selected = (myOffer?.gus || []).includes(gu.instance_id);
                                        const isSleeping = gu.status === "SLEEPING";
                                        const isSealed = gu.status === "SEALED";

                                        return (
                                          <button
                                            type="button"
                                            key={gu.instance_id}
                                            onClick={() => toggleGu(gu.instance_id)}
                                            className={`w-full text-left p-2.5 rounded-lg border text-xs font-mono flex justify-between items-center transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] cursor-pointer ${
                                              selected
                                                ? "bg-emerald-950/20 border-emerald-500/40 text-emerald-400 shadow-md shadow-emerald-950/15"
                                                : "bg-zinc-950 border-zinc-900 hover:border-zinc-850 text-zinc-300"
                                            }`}
                                          >
                                            <div className="flex items-center gap-2">
                                              <span className="font-bold">#{gu.instance_id.slice(0, 6)}</span>
                                              <span className="text-[10px] text-zinc-550">Độ no: {gu.hunger}/100</span>
                                              {isSleeping && <span className="text-[8px] font-bold text-amber-500 uppercase bg-amber-955/20 px-1 rounded border border-amber-900/10">Ngủ</span>}
                                              {isSealed && <span className="text-[8px] font-bold text-cyan-400 uppercase bg-cyan-955/20 px-1 rounded border border-cyan-900/10">Ấn</span>}
                                            </div>
                                            <span className="text-[9px] font-bold uppercase">
                                              {selected ? "Hủy chọn" : "Tích chọn"}
                                            </span>
                                          </button>
                                        );
                                      })}
                                    </div>
                                  )}
                                </div>
                              );
                            });
                          })()
                        ) : (
                          <p className="text-xs text-zinc-650 italic text-center py-4">Không có Cổ trùng nào có thể giao dịch</p>
                        )}
                      </div>
                    )}

                    {/* Materials selectors */}
                    {activeTab === "materials" && (
                      <div className="space-y-3">
                        {/* Quick select bar */}
                        {!myLocked && availableMaterials.length > 0 && (
                          <div className="flex items-center gap-1.5 flex-wrap bg-zinc-950/40 border border-zinc-900 rounded-xl p-2 mb-2">
                            <span className="text-[9px] font-mono text-zinc-500">Chọn nhanh:</span>
                            {Array.from(new Set(availableMaterials.map(m => itemDictionary[m]?.rank || 1))).sort((a,b)=>a-b).map(rank => (
                              <button
                                key={rank}
                                type="button"
                                onClick={() => {
                                  const currentMats = { ...(myOffer?.materials || {}) };
                                  availableMaterials.forEach(m => {
                                    if ((itemDictionary[m]?.rank || 1) === rank) {
                                      currentMats[m] = player.materials_inventory[m];
                                    }
                                  });
                                  updateTradeOffer(activeTrade.id, {
                                    primeval_stones: myOffer?.primeval_stones || 0,
                                    immortal_stones: myOffer?.immortal_stones || 0,
                                    gus: myOffer?.gus || [],
                                    materials: currentMats,
                                    recipes: myOffer?.recipes || {},
                                  });
                                }}
                                className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 border border-zinc-800 transition-colors cursor-pointer"
                              >
                                Rank {rank}
                              </button>
                            ))}
                            <button
                              type="button"
                              onClick={() => {
                                const currentMats = { ...(myOffer?.materials || {}) };
                                availableMaterials.forEach(m => {
                                  currentMats[m] = player.materials_inventory[m];
                                });
                                updateTradeOffer(activeTrade.id, {
                                  primeval_stones: myOffer?.primeval_stones || 0,
                                  immortal_stones: myOffer?.immortal_stones || 0,
                                  gus: myOffer?.gus || [],
                                  materials: currentMats,
                                  recipes: myOffer?.recipes || {},
                                });
                              }}
                              className="px-2 py-0.5 text-[9px] font-mono rounded bg-emerald-950/20 hover:bg-emerald-950/30 text-emerald-400 border border-emerald-900/30 transition-colors cursor-pointer"
                            >
                              Tất cả Vật Phẩm
                            </button>
                            <button
                              type="button"
                              onClick={() => {
                                updateTradeOffer(activeTrade.id, {
                                  primeval_stones: myOffer?.primeval_stones || 0,
                                  immortal_stones: myOffer?.immortal_stones || 0,
                                  gus: myOffer?.gus || [],
                                  materials: {},
                                  recipes: myOffer?.recipes || {},
                                });
                              }}
                              className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-zinc-300 border border-zinc-800 transition-colors cursor-pointer"
                            >
                              Bỏ chọn hết
                            </button>
                          </div>
                        )}
                        <div className="grid gap-3 lg:grid-cols-[minmax(0,1.4fr)_minmax(15rem,1fr)]">
                          <div className="max-h-[34dvh] space-y-1.5 overflow-y-auto pr-1">
                            {filteredMaterials.map((matId) => {
                              const item = itemDictionary[matId];
                              const available = player.materials_inventory[matId] - (myOffer?.materials?.[matId] || 0);
                              return (
                                <button
                                  key={matId}
                                  type="button"
                                  onClick={() => { setSelectedMat(matId); setSelectedMatQty(1); }}
                                  className={`flex w-full items-center gap-3 rounded-xl border p-2.5 text-left transition-colors ${selectedMat === matId ? "border-violet-400/50 bg-violet-500/10" : "border-zinc-900 bg-zinc-950/50 hover:border-zinc-700"}`}
                                >
                                  <img src={getItemIconUrl(item ?? { id: matId, name: getMatName(matId), item_type: "Material" })} alt="" className="h-10 w-10 shrink-0 rounded-lg border border-zinc-800 bg-zinc-900 object-contain p-1" />
                                  <span className="min-w-0 flex-1"><span className="block truncate text-xs font-semibold text-zinc-200">{getMatName(matId)}</span><span className="block truncate text-[10px] font-mono text-zinc-500">{matId}</span></span>
                                  <span className="shrink-0 text-right"><span className="block text-xs font-mono text-violet-300">Còn {available}</span><span className="block text-[10px] text-zinc-600">/ {player.materials_inventory[matId]} trong túi</span></span>
                                </button>
                              );
                            })}
                          </div>

                          {selectedMat ? (() => {
                            const item = itemDictionary[selectedMat];
                            const available = player.materials_inventory[selectedMat] - (myOffer?.materials?.[selectedMat] || 0);
                            return (
                              <div className="flex flex-col justify-between gap-4 rounded-2xl border border-violet-500/20 bg-violet-950/10 p-4">
                                <div className="flex items-center gap-3">
                                  <img src={getItemIconUrl(item ?? { id: selectedMat, name: getMatName(selectedMat), item_type: "Material" })} alt="" className="h-14 w-14 rounded-xl border border-violet-500/20 bg-zinc-950 object-contain p-2" />
                                  <div className="min-w-0"><p className="truncate text-sm font-semibold text-zinc-100">{getMatName(selectedMat)}</p><p className="text-xs text-zinc-500">Khả dụng: {available} / {player.materials_inventory[selectedMat]}</p></div>
                                </div>
                                <div className="space-y-2">
                                  <label htmlFor="trade-material-quantity" className="text-xs font-medium text-zinc-300">Số lượng muốn đưa</label>
                                  <div className="flex items-center gap-2">
                                    <button type="button" aria-label="Giảm số lượng vật phẩm" disabled={selectedMatQty <= 1} onClick={() => setSelectedMatQty((qty) => Math.max(1, qty - 1))} className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-300 disabled:opacity-40"><Minus className="h-4 w-4" /></button>
                                    <input id="trade-material-quantity" type="number" min={1} max={available} value={selectedMatQty} onChange={(e) => setSelectedMatQty(Math.max(1, Math.min(available, parseInt(e.target.value) || 1)))} className="h-10 min-w-0 flex-1 rounded-lg border border-zinc-800 bg-zinc-950 px-2 text-center text-sm font-mono text-zinc-100" />
                                    <button type="button" aria-label="Tăng số lượng vật phẩm" disabled={selectedMatQty >= available} onClick={() => setSelectedMatQty((qty) => Math.min(available, qty + 1))} className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-300 disabled:opacity-40"><Plus className="h-4 w-4" /></button>
                                    <button type="button" onClick={() => setSelectedMatQty(available)} className="h-10 shrink-0 rounded-lg border border-amber-500/20 bg-amber-950/20 px-3 text-xs font-bold text-amber-300">Tối đa</button>
                                  </div>
                                </div>
                                <GameButton variant="primary" onClick={addMaterial} disabled={available <= 0}>Thêm vào giao dịch</GameButton>
                              </div>
                            );
                          })() : (
                            <div className="flex min-h-40 items-center justify-center rounded-2xl border border-dashed border-zinc-800 p-4 text-center text-xs text-zinc-500">Chọn vật phẩm để xem ảnh, số lượng khả dụng và điều chỉnh số lượng trao đổi.</div>
                          )}
                        </div>

                        {availableMaterials.length === 0 && (
                          <p className="text-xs text-zinc-600 italic text-center py-2">Hành lý trống hoặc toàn bộ nguyên liệu đã được chọn</p>
                        )}
                      </div>
                    )}

                    {/* Recipes selectors */}
                    {activeTab === "recipes" && (
                      <div className="space-y-3">
                        {/* Quick select bar */}
                        {!myLocked && recipes.filter((r) => !myOffer?.recipes?.[r.id]).length > 0 && (
                          <div className="flex items-center gap-1.5 flex-wrap bg-zinc-950/40 border border-zinc-900 rounded-xl p-2 mb-2">
                            <span className="text-[9px] font-mono text-zinc-500">Chọn nhanh:</span>
                            {Array.from(new Set(recipes.filter((r) => !myOffer?.recipes?.[r.id]).map(r => r.target_rank || 1))).sort((a,b)=>a-b).map(rank => (
                              <button
                                key={rank}
                                type="button"
                                onClick={() => {
                                  const currentRecipes = { ...(myOffer?.recipes || {}) };
                                  recipes.filter((r) => !myOffer?.recipes?.[r.id]).forEach(r => {
                                    if ((r.target_rank || 1) === rank) {
                                      currentRecipes[r.id] = r.completion_rate ?? 100;
                                    }
                                  });
                                  updateTradeOffer(activeTrade.id, {
                                    primeval_stones: myOffer?.primeval_stones || 0,
                                    immortal_stones: myOffer?.immortal_stones || 0,
                                    gus: myOffer?.gus || [],
                                    materials: myOffer?.materials || {},
                                    recipes: currentRecipes,
                                  });
                                }}
                                className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-400 border border-zinc-800 transition-colors cursor-pointer"
                              >
                                Rank {rank}
                              </button>
                            ))}
                            <button
                              type="button"
                              onClick={() => {
                                const currentRecipes = { ...(myOffer?.recipes || {}) };
                                recipes.filter((r) => !myOffer?.recipes?.[r.id]).forEach(r => {
                                  currentRecipes[r.id] = r.completion_rate ?? 100;
                                });
                                updateTradeOffer(activeTrade.id, {
                                  primeval_stones: myOffer?.primeval_stones || 0,
                                  immortal_stones: myOffer?.immortal_stones || 0,
                                  gus: myOffer?.gus || [],
                                  materials: myOffer?.materials || {},
                                  recipes: currentRecipes,
                                });
                              }}
                              className="px-2 py-0.5 text-[9px] font-mono rounded bg-emerald-950/20 hover:bg-emerald-950/30 text-emerald-400 border border-emerald-900/30 transition-colors cursor-pointer"
                            >
                              Tất cả Bí Phương
                            </button>
                            <button
                              type="button"
                              onClick={() => {
                                updateTradeOffer(activeTrade.id, {
                                  primeval_stones: myOffer?.primeval_stones || 0,
                                  immortal_stones: myOffer?.immortal_stones || 0,
                                  gus: myOffer?.gus || [],
                                  materials: myOffer?.materials || {},
                                  recipes: {},
                                });
                              }}
                              className="px-2 py-0.5 text-[9px] font-mono rounded bg-zinc-900 hover:bg-zinc-800 text-zinc-500 hover:text-zinc-300 border border-zinc-800 transition-colors cursor-pointer"
                            >
                              Bỏ chọn hết
                            </button>
                          </div>
                        )}
                        <div className="max-h-[38dvh] space-y-2 overflow-y-auto pr-1">
                          {filteredRecipes.map((recipe) => {
                            const selected = selectedRecipe === recipe.id;
                            const targetGu = recipe.target_gu_id ? guSpecFor(recipe.target_gu_id, recipe.target_rank || 1)?.name : "";
                            return (
                              <div key={recipe.id} className={`flex items-center gap-3 rounded-xl border p-3 transition-colors ${selected ? "border-pink-400/40 bg-pink-500/10" : "border-zinc-900 bg-zinc-950/50"}`}>
                                <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-pink-500/15 bg-pink-500/5 text-pink-300"><ScrollText className="h-5 w-5" /></span>
                                <button type="button" onClick={() => setSelectedRecipe(recipe.id)} className="min-w-0 flex-1 text-left"><span className="block truncate text-xs font-semibold text-zinc-100">{getRecipeName(recipe.id)}</span><span className="block truncate text-[10px] text-zinc-500">{targetGu ? `Cổ trùng đích: ${targetGu} · ` : ""}{recipe.id}</span><span className="mt-1 inline-flex rounded-md bg-pink-950/40 px-1.5 py-0.5 text-[10px] font-mono text-pink-300">Hoàn thiện {recipe.completion_rate ?? 100}%</span></button>
                                <GameButton size="xs" variant="secondary" disabled={!selected} onClick={addRecipe}>Thêm</GameButton>
                              </div>
                            );
                          })}
                        </div>

                        {filteredRecipes.length === 0 && (
                          <p className="text-xs text-zinc-600 italic text-center py-2">Không có Bí phương nào có thể giao dịch</p>
                        )}
                      </div>
                    )}
                  </>
                )}
              </div>
              </GameModal>
            )}
          </div>

          {/* RIGHT SIDE: Opponent (Partner) */}
          <div className="p-6 flex flex-col gap-5 bg-zinc-950/20">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-serif font-bold text-zinc-300 flex items-center gap-1.5">
                <User className="w-4 h-4 text-violet-400 animate-pulse" />
                Đối Phương ({oppUsername})
              </h4>
              <div className="flex items-center gap-1.5">
                {oppLocked ? (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-emerald-950/50 text-emerald-400 border border-emerald-500/30 font-mono flex items-center gap-1">
                    <Lock className="w-3 h-3 text-emerald-400" /> Đã Khóa
                  </span>
                ) : (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-rose-950/20 text-rose-400 border border-rose-900/10 font-mono flex items-center gap-1 animate-pulse">
                    <Unlock className="w-3 h-3 text-rose-400" /> Đang Biên Soạn
                  </span>
                )}
                {oppConfirmed && (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-blue-950/50 text-blue-400 border border-blue-500/30 font-mono flex items-center gap-1">
                    <CheckCircle2 className="w-3 h-3" /> Xác Nhận
                  </span>
                )}
              </div>
            </div>

            {/* Opponent's Offer Display Box */}
            <div className="bg-zinc-950 border border-zinc-900 rounded-2xl p-5 space-y-4 flex-1 shadow-md">
              <span className="text-[9px] uppercase tracking-wider text-zinc-500 font-mono block">Ưu đãi đề xuất của đối phương:</span>

              {/* Stones offered */}
              <div className="flex flex-wrap gap-2 text-xs font-mono">
                {((oppOffer?.primeval_stones || 0) > 0) && (
                  <span className="px-2.5 py-1.5 rounded-xl bg-yellow-950/20 border border-yellow-500/10 text-yellow-500 flex items-center gap-1 select-none">
                    <Coins className="w-3.5 h-3.5" />
                    {(oppOffer.primeval_stones).toLocaleString()} Nguyên Thạch
                  </span>
                )}
                {((oppOffer?.immortal_stones || 0) > 0) && (
                  <span className="px-2.5 py-1.5 rounded-xl bg-purple-950/20 border border-purple-500/10 text-purple-400 flex items-center gap-1 select-none">
                    <Sparkles className="w-3.5 h-3.5" />
                    {(oppOffer.immortal_stones).toFixed(1)} Tiên Nguyên
                  </span>
                )}
              </div>

              {/* Gus offered */}
              {oppOffer?.gus && oppOffer.gus.length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Cổ trùng ({oppOffer.gus.length}):</span>
                  <div className="flex flex-wrap gap-1.5">
                    {(oppOffer.gus as string[]).map((instId: string, index: number) => {
                      const guName = getGuName(instId);
                      const guRank = getGuRank(instId);
                      const guId = activeTrade.gu_details?.[instId] || instId;
                      return (
                        <span key={index} className="text-xs px-2.5 py-1 rounded-xl bg-emerald-950/20 border border-emerald-900/20 text-emerald-400 select-none flex items-center gap-1.5">
                          <img src={getGuIconUrl(guId, guName, guDictionary[guId]?.image_url)} alt="" className="h-6 w-6 rounded-md bg-zinc-950 object-contain p-0.5" />
                          <span>{guName}</span>
                          <span className="text-[9px] font-mono opacity-80 px-1 bg-emerald-900/20 border border-emerald-800/35 rounded text-emerald-300">
                            Rank {guRank}
                          </span>
                        </span>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Materials offered */}
              {oppOffer?.materials && Object.keys(oppOffer.materials).length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Nguyên liệu:</span>
                  <div className="flex flex-wrap gap-1.5">
                    {Object.entries(oppOffer.materials as Record<string, number>).map(([matId, qty]) => (
                      <span key={matId} className="text-xs px-2.5 py-1 rounded-xl bg-zinc-900 border border-zinc-850 text-zinc-300 select-none">
                        <img src={getItemIconUrl(itemDictionary[matId] ?? { id: matId, name: getMatName(matId), item_type: "Material" })} alt="" className="mr-1 inline h-5 w-5 rounded object-contain align-middle" />
                        {getMatName(matId)} (x{qty})
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Recipes offered */}
              {oppOffer?.recipes && Object.keys(oppOffer.recipes).length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Bí phương:</span>
                  <div className="flex flex-wrap gap-1.5">
                    {Object.entries(oppOffer.recipes as Record<string, number>).map(([recipeId, rate]) => (
                      <span key={recipeId} className="text-xs px-2.5 py-1 rounded-xl bg-violet-950/20 border border-violet-500/20 text-violet-300 select-none">
                        <ScrollText className="mr-1 inline h-4 w-4 align-middle" />
                        {getRecipeName(recipeId)} ({rate.toFixed(0)}%)
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {!(oppOffer?.primeval_stones || 0) && !(oppOffer?.immortal_stones || 0) && (!oppOffer?.gus || oppOffer.gus.length === 0) && (!oppOffer?.materials || Object.keys(oppOffer.materials).length === 0) && (!oppOffer?.recipes || Object.keys(oppOffer.recipes).length === 0) && (
                <p className="text-xs text-zinc-650 italic py-6 select-none">Chưa đề xuất vật phẩm nào</p>
              )}
            </div>

            {/* Guidance panel */}
            <div className="px-4 py-3 bg-zinc-900/10 border-l border-zinc-800 text-[10px] text-zinc-500 font-mono rounded-r-xl flex items-start gap-2 leading-relaxed">
              <ShieldAlert className="w-4 h-4 text-zinc-600 shrink-0" />
              <div>
                <p className="font-bold text-zinc-400">Luật Bảo Mật:</p>
                <p>Mỗi khi một bên chỉnh sửa ưu đãi, toàn bộ khóa và xác nhận sẽ được đặt lại để đảm bảo an toàn, phòng ngừa lừa đảo.</p>
              </div>
            </div>

          </div>
        </div>

        {/* Footer actions bar */}
        <div className="px-6 py-5 border-t border-zinc-900 bg-zinc-950 flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4">
          <div className="text-[10px] font-mono text-zinc-500 leading-tight">
            <span>Thực trạng liên kết: </span>
            {myLocked && oppLocked ? (
              <span className="text-emerald-400">Kết nối ổn định. Đã khóa cả hai bên. Sẵn sàng xác nhận.</span>
            ) : (
              <span className="text-amber-500">Đang đồng bộ hóa ưu đãi... Cần cả hai bên cùng khóa.</span>
            )}
          </div>

          <div className="flex items-center gap-3">
            {/* Lock / Unlock Offer Button */}
            {!myLocked ? (
              <button
                onClick={() => {
                  syncStones();
                  lockTrade(activeTrade.id);
                }}
                className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors flex items-center gap-1.5 cursor-pointer shadow-lg shadow-emerald-950/25 border border-emerald-500/20"
              >
                <Lock className="w-3.5 h-3.5" /> Khóa Giao Dịch
              </button>
            ) : (
              <button
                onClick={() => unlockTrade(activeTrade.id)}
                className="px-4 py-2.5 rounded-xl text-xs font-serif font-bold bg-amber-950/50 hover:bg-amber-900/60 text-amber-300 border border-amber-800/40 flex items-center gap-1.5 cursor-pointer transition-colors shadow-lg shadow-amber-950/25"
                title="Bấm để mở khóa và chỉnh sửa lại vật phẩm"
              >
                <Unlock className="w-3.5 h-3.5" /> Mở Khóa Chỉnh Sửa
              </button>
            )}

            {/* Confirm Trade Button (Enabled only when both locked and you haven't confirmed yet) */}
            {myLocked && oppLocked && !myConfirmed ? (
              <button
                onClick={() => confirmTrade(activeTrade.id)}
                className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-gradient-to-r from-blue-600 to-indigo-500 hover:from-blue-500 hover:to-indigo-400 text-zinc-100 transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] cursor-pointer shadow-lg shadow-blue-950/25 border border-blue-500/20"
              >
                <CheckCircle2 className="w-3.5 h-3.5" /> Xác Nhận Giao Dịch
              </button>
            ) : myConfirmed ? (
              <span className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-indigo-950/40 text-indigo-400 border border-indigo-900/35 flex items-center gap-1.5 animate-pulse select-none">
                <RefreshCw className="w-3 h-3 animate-spin" /> Đang Chờ Đối Phương...
              </span>
            ) : (
              <button
                disabled
                className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-zinc-900 text-zinc-650 border border-zinc-950 flex items-center gap-1.5 select-none"
              >
                Xác Nhận
              </button>
            )}
          </div>
        </div>

      </div>
    </GameModal>
  );
};
