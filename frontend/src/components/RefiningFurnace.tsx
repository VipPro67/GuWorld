"use client";

import React, { useState, useEffect } from "react";
import { Player, GuSpec, ItemSpec, RefiningRecipe, GuInstance, useGameStore } from "@/store/useGameStore";
import { Flame, Sparkles, AlertCircle, Check, X, RefreshCw, Coins, Zap, Trophy, HelpCircle } from "lucide-react";

interface RefiningFurnaceProps {
  player: Player;
  guDictionary: Record<string, GuSpec>;
  itemDictionary: Record<string, ItemSpec>;
  recipes: RefiningRecipe[];
  actionLoading: boolean;
  onRefine: (recipeId: string, selectedInstanceIds: string[]) => Promise<{ status: "success" | "failed"; message: string; new_gu?: GuInstance } | null>;
}

export const RefiningFurnace: React.FC<RefiningFurnaceProps> = ({
  player,
  guDictionary,
  itemDictionary,
  recipes,
  actionLoading,
  onRefine,
}) => {
  const [selectedRecipeId, setSelectedRecipeId] = useState<string | null>(null);
  const [slottedGus, setSlottedGus] = useState<Record<number, string>>({}); // slotIndex -> guInstanceId
  const [isRefining, setIsRefining] = useState(false);
  const [furnaceStage, setFurnaceStage] = useState("Đang khởi động lò luyện...");
  const [refineResult, setRefineResult] = useState<{ status: "success" | "failed"; new_gu?: GuInstance; message: string } | null>(null);
  const [searchKeyword, setSearchKeyword] = useState("");
  const [customRecipes, setCustomRecipes] = useState<RefiningRecipe[]>([]);

  // Auto-select first recipe if none selected
  useEffect(() => {
    if (recipes.length > 0 && !selectedRecipeId) {
      setSelectedRecipeId(recipes[0].id);
    }
  }, [recipes, selectedRecipeId]);

  // Clear slotted Gus when changing recipe
  useEffect(() => {
    setSlottedGus({});
  }, [selectedRecipeId]);

  const allRecipes = [...recipes, ...customRecipes];
  const filteredRecipes = allRecipes.filter((r) => {
    const targetName = guDictionary[r.target_gu_id]?.name || r.target_gu_id;
    return targetName.toLowerCase().includes(searchKeyword.toLowerCase()) || r.id.toLowerCase().includes(searchKeyword.toLowerCase());
  });
  const selectedRecipe = allRecipes.find((r) => r.id === selectedRecipeId);

  // Flatten the required Gus into an ordered list of slots
  const getRequiredSlots = () => {
    if (!selectedRecipe) return [];
    const slots: { index: number; guId: string }[] = [];
    let idx = 0;
    Object.entries(selectedRecipe.required_gus).forEach(([guId, qty]) => {
      for (let i = 0; i < qty; i++) {
        slots.push({ index: idx++, guId });
      }
    });
    return slots;
  };

  const requiredSlots = getRequiredSlots();

  // Find all Gus currently in player inventory that are NOT dead and NOT slotted
  const slottedIds = Object.values(slottedGus);
  const availableInventoryGus = (player.gu_inventory || []).filter(
    (gu) => gu.status !== "DEAD" && !slottedIds.includes(gu.instance_id)
  );

  // Handle slotting a Gu by instanceId into a target slot index
  const slotGu = (instanceId: string, targetSlotIndex: number) => {
    const guInstance = player.gu_inventory.find((g) => g.instance_id === instanceId);
    if (!guInstance) return;

    const targetSlotRequirement = requiredSlots.find((s) => s.index === targetSlotIndex);
    if (!targetSlotRequirement) return;

    // Verify it matches the requirement
    if (guInstance.gu_id !== targetSlotRequirement.guId) return;

    setSlottedGus((prev) => ({
      ...prev,
      [targetSlotIndex]: instanceId,
    }));
  };

  // Auto-slot Gu by clicking it
  const handleInventoryClick = (gu: GuInstance) => {
    // Find the first empty slot that requires this Gu type
    const emptySlot = requiredSlots.find(
      (slot) => slot.guId === gu.gu_id && !slottedGus[slot.index]
    );

    if (emptySlot !== undefined) {
      slotGu(gu.instance_id, emptySlot.index);
    }
  };

  // Remove Gu from slot
  const removeGuFromSlot = (slotIndex: number) => {
    setSlottedGus((prev) => {
      const copy = { ...prev };
      delete copy[slotIndex];
      return copy;
    });
  };

  // Check if all slots are filled
  const allSlotsFilled = requiredSlots.every((slot) => !!slottedGus[slot.index]);

  const hasSlottedVital = Object.values(slottedGus).some(instanceId => {
    const gu = player.gu_inventory.find(g => g.instance_id === instanceId);
    return gu ? gu.is_vital : false;
  });

  // Check if player has enough materials
  const hasRequiredMaterials = () => {
    if (!selectedRecipe) return false;
    return Object.entries(selectedRecipe.required_materials).every(([matId, qty]) => {
      const playerQty = player.materials_inventory?.[matId] || 0;
      return playerQty >= qty;
    });
  };

  // Check if player has enough stones
  const hasEnoughStones = selectedRecipe ? player.primeval_stones >= selectedRecipe.primeval_stones_cost : false;

  const canRefine = allSlotsFilled && hasRequiredMaterials() && hasEnoughStones && !actionLoading && !isRefining;

  // Run Refining Logic
  const handleRefine = async () => {
    if (!canRefine || !selectedRecipe) return;

    setIsRefining(true);
    setRefineResult(null);

    // Gacha Suspense Artificial Delay Loop
    const stages = [
      "Đang khơi mào chân hỏa...",
      "Đang dung hợp linh tính...",
      "Khống chế hỏa hầu cực độ...",
      "Ngưng tụ ý chí nguyên sinh..."
    ];

    let currentStage = 0;
    setFurnaceStage(stages[0]);

    const stageInterval = setInterval(() => {
      currentStage++;
      if (currentStage < stages.length) {
        setFurnaceStage(stages[currentStage]);
      }
    }, 600);

    // Wait 2.5 seconds total for suspense
    setTimeout(async () => {
      clearInterval(stageInterval);
      const selectedIds = requiredSlots.map((slot) => slottedGus[slot.index]);
      
      try {
        const response = await onRefine(selectedRecipe.id, selectedIds);
        if (response) {
          setRefineResult(response);
          if (response.status === "success" && response.new_gu) {
            const discovered: RefiningRecipe = {
              id: `user_${Date.now()}`,
              target_gu_id: response.new_gu.gu_id,
              required_gus: requiredSlots.reduce((acc, slot) => { acc[slot.guId] = (acc[slot.guId] || 0) + 1; return acc; }, {} as Record<string, number>),
              required_materials: selectedRecipe.required_materials,
              success_rate: selectedRecipe.success_rate,
              primeval_stones_cost: selectedRecipe.primeval_stones_cost,
            };
            setCustomRecipes((prev) => [discovered, ...prev]);
          }
        } else {
          setRefineResult({
            status: "failed",
            message: "Hợp luyện thất bại bí ẩn. Lò hỏa đã tắt ngúm.",
          });
        }
      } catch (err: any) {
        setRefineResult({
          status: "failed",
          message: err.message || "Lỗi giao dịch hợp luyện.",
        });
      } finally {
        setIsRefining(false);
      }
    }, 2500);
  };

  // Drag and Drop helpers
  const handleDragStart = (e: React.DragEvent, instanceId: string) => {
    e.dataTransfer.setData("instanceId", instanceId);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleDrop = (e: React.DragEvent, targetSlotIndex: number) => {
    e.preventDefault();
    const instanceId = e.dataTransfer.getData("instanceId");
    slotGu(instanceId, targetSlotIndex);
  };

  // Helper for material styling & colors
  const getMaterialDetails = (matId: string) => {
    const spec = itemDictionary[matId];
    const name = spec ? spec.name : matId;
    const desc = spec ? spec.description : "";
    
    // Assign specific icons/colors based on ID
    let icon = <HelpCircle className="w-4 h-4" />;
    let colorClass = "text-zinc-400";
    let bgClass = "bg-zinc-950/70 border-zinc-900";

    if (matId.includes("wine")) {
      colorClass = "text-amber-400";
      if (matId === "sour_wine") colorClass = "text-lime-400";
      else if (matId === "sweet_wine") colorClass = "text-rose-400";
      else if (matId === "bitter_wine") colorClass = "text-purple-400";
      else if (matId === "spicy_wine") colorClass = "text-orange-400";
    } else if (matId === "wild_boar_tusk") {
      icon = <Trophy className="w-4 h-4" />;
      colorClass = "text-stone-300";
    } else if (matId === "boar_meat") {
      icon = <Flame className="w-4 h-4 text-orange-400" />;
      colorClass = "text-orange-300";
    }

    return { name, desc, icon, colorClass, bgClass };
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 w-full max-w-5xl mx-auto relative">
      {/* 1. LEFT PANEL: Inventory List */}
      <div className="lg:col-span-5 glass-panel-gold rounded-3xl p-5 flex flex-col h-[580px] overflow-hidden">
        <div className="flex items-center gap-2 border-b border-zinc-800/80 pb-3 mb-4 shrink-0">
          <Sparkles className="w-5 h-5 text-amber-500 animate-pulse" />
          <h3 className="text-md font-bold text-zinc-100 font-serif">
            Linh Khí Túi (Gu Bugs For Refining)
          </h3>
        </div>

        <span className="text-[10px] text-zinc-500 font-mono mb-3 uppercase tracking-wider block shrink-0">
          * Kéo thả hoặc click vào Cổ trùng để đặt vào Lò luyện.
        </span>

        {/* Scrollable list */}
        <div className="flex-1 overflow-y-auto custom-scrollbar space-y-2.5 pr-1">
          {availableInventoryGus.length === 0 ? (
            <div className="h-full flex flex-col items-center justify-center text-center p-6 border border-dashed border-zinc-800/60 rounded-2xl text-xs text-zinc-600 italic">
              Không có Cổ trùng phù hợp hoặc toàn bộ Cổ trùng đã được xếp vào lò.
            </div>
          ) : (
            availableInventoryGus.map((gu) => {
              const spec = guDictionary[gu.gu_id] || {
                id: gu.gu_id,
                name: gu.gu_id,
                rank: 1,
                path: "Unknown Path",
                gu_type: "Phàm Cổ",
                origin_region: "",
                base_mana_cost: 0,
                effects: {},
                hunger_rate: 0,
                food_type: "unknown",
                max_hunger: 100,
                description: "Cổ trùng chưa được giải nghĩa.",
              };
              const isSleeping = gu.status === "SLEEPING";

              return (
                <div
                  key={gu.instance_id}
                  draggable
                  onDragStart={(e) => handleDragStart(e, gu.instance_id)}
                  onClick={() => handleInventoryClick(gu)}
                  className={`p-3.5 rounded-xl border border-zinc-900 bg-zinc-950/40 hover:bg-zinc-950/80 hover:border-amber-500/30 cursor-grab active:cursor-grabbing transition-all select-none group text-left relative flex justify-between items-center`}
                >
                  <div className="space-y-1 max-w-[80%]">
                    <div className="flex items-center gap-1.5">
                      <span className="text-sm font-bold text-zinc-200 group-hover:text-amber-400 transition-colors font-serif">
                        {spec.name}
                      </span>
                      <span className="text-[8px] font-mono border border-amber-500/20 px-1 py-0.2 rounded text-amber-400 uppercase">
                        Rank {spec.rank}
                      </span>
                    </div>
                    <span className="text-[9px] text-zinc-500 font-mono block">
                      {spec.gu_type}
                    </span>
                    <p className="text-[10px] text-zinc-400 line-clamp-1 leading-normal">
                      {spec.description}
                    </p>
                  </div>

                  <div className="flex flex-col items-end gap-2 shrink-0">
                    <span className="text-[9px] text-zinc-500 font-mono">
                      No: {gu.hunger}/100
                    </span>
                    {isSleeping ? (
                      <span className="text-[8px] text-amber-500 font-bold bg-amber-950/20 border border-amber-500/20 px-1 rounded animate-pulse uppercase">
                        Ngủ
                      </span>
                    ) : (
                      <span className="text-[8px] text-emerald-400 font-bold bg-emerald-950/20 border border-emerald-500/20 px-1 rounded uppercase">
                        Tỉnh
                      </span>
                    )}
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* 2. RIGHT PANEL: Furnace Crucible */}
      <div
        className={`lg:col-span-7 glass-panel rounded-3xl p-5 flex flex-col h-[580px] justify-between relative overflow-hidden transition-all duration-300 ${
          isRefining ? "animate-shake border-rose-500/30" : "border-zinc-800/80"
        }`}
      >
        {/* Decorative intense fire background during refining */}
        {isRefining && (
          <>
            <div className="absolute inset-0 bg-radial-gradient from-red-600/35 via-amber-500/10 to-transparent animate-pulse pointer-events-none z-0" />
            <div className="absolute -bottom-10 left-1/2 -translate-x-1/2 w-80 h-80 bg-orange-600/20 rounded-full blur-[80px] animate-burn z-0" />
            {/* Smoke elements */}
            <div className="absolute bottom-20 left-1/3 w-8 h-8 rounded-full bg-zinc-700/10 animate-smoke z-0" />
            <div className="absolute bottom-24 left-2/3 w-10 h-10 rounded-full bg-zinc-700/15 animate-smoke z-0" />
            <div className="absolute bottom-16 left-1/2 w-6 h-6 rounded-full bg-zinc-800/10 animate-smoke z-0" />
          </>
        )}

        <div className="z-10 flex flex-col h-full justify-between">
          {/* Header & Tabs */}
          <div className="shrink-0 space-y-4">
            <div className="flex justify-between items-center border-b border-zinc-900 pb-3">
              <h3 className="text-md font-bold text-zinc-100 font-serif flex items-center gap-1.5">
                <Flame className={`w-5 h-5 text-rose-500 ${isRefining ? "animate-bounce" : ""}`} />
                Lò Luyện Cổ (Refinement Crucible)
              </h3>
              <span className="text-xs text-zinc-500 font-mono">
                {selectedRecipe?.success_rate ? `Tỷ lệ: ${selectedRecipe.success_rate * 100}%` : ""}
              </span>
            </div>

            {/* Recipes Tabs */}
            <div className="flex gap-2">
              {filteredRecipes.map((recipe) => {
                const targetGu = guDictionary[recipe.target_gu_id] || { name: recipe.target_gu_id };
                const isActive = recipe.id === selectedRecipeId;
                return (
                  <button
                    key={recipe.id}
                    disabled={isRefining}
                    onClick={() => setSelectedRecipeId(recipe.id)}
                    className={`flex-1 py-2 px-3 text-xs font-semibold rounded-xl border text-center transition-all cursor-pointer ${
                      isActive
                        ? "bg-amber-500/15 text-amber-400 border-amber-500/50 shadow-md shadow-amber-950/20"
                        : "bg-zinc-950/40 text-zinc-400 border-zinc-900 hover:text-zinc-200 hover:border-zinc-800"
                    }`}
                  >
                    {targetGu.name}
                  </button>
                );
              })}
            </div>
          </div>

          {/* Central Furnace Slots Area */}
          <div className="flex-1 flex flex-col justify-center items-center py-6 relative z-10">
            {selectedRecipe ? (
              <div className="w-full space-y-6">
                {/* Target Gu Header */}
                <div className="text-center space-y-1">
                  <span className="text-[10px] text-amber-500 uppercase tracking-widest font-mono font-bold block">
                    Sản Vật Hợp Thành (Target Product)
                  </span>
                  <h4 className="text-lg font-bold text-zinc-100 font-serif">
                    {guDictionary[selectedRecipe.target_gu_id]?.name || selectedRecipe.target_gu_id}
                  </h4>
                  <p className="text-xs text-zinc-400 max-w-md mx-auto line-clamp-1 italic leading-relaxed">
                    {guDictionary[selectedRecipe.target_gu_id]?.description || ""}
                  </p>
                </div>

                {/* Slots Crucible Row */}
                <div className="flex justify-center items-center gap-6 py-2">
                  {requiredSlots.map((slot) => {
                    const reqGuSpec = guDictionary[slot.guId] || { name: slot.guId };
                    const slottedInstanceId = slottedGus[slot.index];
                    const slottedGu = player.gu_inventory.find(
                      (g) => g.instance_id === slottedInstanceId
                    );

                    return (
                      <div
                        key={slot.index}
                        onDragOver={handleDragOver}
                        onDrop={(e) => handleDrop(e, slot.index)}
                        onClick={() => slottedInstanceId && removeGuFromSlot(slot.index)}
                        className={`w-32 h-36 rounded-2xl border transition-all flex flex-col items-center justify-center p-3 relative cursor-pointer group ${
                          slottedGu
                            ? "bg-zinc-950/90 border-amber-500/40 shadow-lg shadow-amber-900/5 hover:border-rose-500/50"
                            : "bg-zinc-950/40 border-dashed border-zinc-800 hover:border-zinc-700"
                        }`}
                      >
                        {slottedGu ? (
                          <div className="text-center space-y-1.5 w-full">
                            <span className="text-[9px] font-mono border border-amber-500/20 px-1 py-0.2 rounded text-amber-400 uppercase inline-block mx-auto">
                              Rank {guDictionary[slottedGu.gu_id]?.rank || 1}
                            </span>
                            <div className="text-xs font-bold text-zinc-200 truncate font-serif">
                              {guDictionary[slottedGu.gu_id]?.name || slottedGu.gu_id}
                            </div>
                            <div className="text-[9px] text-zinc-500">
                              Độ no: {slottedGu.hunger}/100
                            </div>
                            <span className="text-[8px] text-rose-400 opacity-0 group-hover:opacity-100 transition-opacity block font-bold font-mono uppercase tracking-wider pt-1 border-t border-zinc-900">
                              Click để gỡ
                            </span>
                          </div>
                        ) : (
                          <div className="text-center space-y-2">
                            <Zap className="w-5 h-5 text-zinc-700 mx-auto animate-pulse" />
                            <div className="text-[10px] text-zinc-500 font-semibold max-w-[80px] leading-tight">
                              Xếp vào:<br />
                              <span className="text-amber-500/80 font-bold">{reqGuSpec.name}</span>
                            </div>
                          </div>
                        )}
                        </div>
                    );
                  })}
                </div>
                {hasSlottedVital && (
                  <div className="text-center text-[10px] text-amber-400 font-mono bg-amber-950/20 border border-amber-900/30 px-3 py-1.5 rounded-lg max-w-md mx-auto animate-pulse flex items-center gap-1.5 justify-center">
                    <Sparkles className="w-3.5 h-3.5" />
                    <span>Đặc quyền Bản Mệnh: Cổ Bản Mệnh sẽ không bị tiêu biến nếu hợp luyện thất bại!</span>
                  </div>
                )}
              </div>
            ) : (
              <div className="text-xs text-zinc-500 italic">Đang phân tích dược phẩm và công thức hợp luyện...</div>
            )}
          </div>

          {/* Footer Details: Costs, Materials, Action Button */}
          <div className="shrink-0 space-y-4 pt-4 border-t border-zinc-900/60 z-10">
            {selectedRecipe && (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {/* Requirements check list */}
                <div className="space-y-2">
                  <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono block">
                    Phụ Gia Vật Phẩm (Required Materials)
                  </span>

                  {Object.keys(selectedRecipe.required_materials).length === 0 ? (
                    <div className="text-[11px] text-zinc-500 italic">Không yêu cầu phụ phẩm.</div>
                  ) : (
                    <div className="space-y-1.5">
                      {Object.entries(selectedRecipe.required_materials).map(([matId, qty]) => {
                        const playerQty = player.materials_inventory?.[matId] || 0;
                        const hasEnough = playerQty >= qty;
                        const matDetails = getMaterialDetails(matId);

                        return (
                          <div
                            key={matId}
                            className="flex items-center justify-between text-xs bg-zinc-950/50 border border-zinc-900/60 px-3 py-1.5 rounded-lg"
                          >
                            <div className="flex items-center gap-1.5">
                              <span className={matDetails.colorClass}>{matDetails.name}</span>
                            </div>
                            <div className="flex items-center gap-1.5 font-mono">
                              <span className={hasEnough ? "text-zinc-300" : "text-rose-400 font-bold"}>
                                {playerQty}/{qty}
                              </span>
                              {hasEnough ? (
                                <Check className="w-3.5 h-3.5 text-emerald-400" />
                              ) : (
                                <X className="w-3.5 h-3.5 text-rose-500" />
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>

                {/* Stones cost and Execution button */}
                <div className="flex flex-col justify-end space-y-3">
                  <div className="flex justify-between items-center text-xs px-1">
                    <span className="text-zinc-500 uppercase tracking-widest font-mono text-[10px] flex items-center gap-1">
                      <Coins className="w-3.5 h-3.5 text-amber-500" />
                      Nguyên thạch tiêu hao:
                    </span>
                    <span className={`font-bold font-mono ${hasEnoughStones ? "text-amber-400" : "text-rose-400 font-bold text-sm animate-pulse"}`}>
                      {player.primeval_stones}/{selectedRecipe.primeval_stones_cost} viên
                    </span>
                  </div>

                  <button
                    onClick={handleRefine}
                    disabled={!canRefine}
                    className={`w-full py-3 px-4 rounded-xl text-sm font-bold tracking-wide font-serif transition-all flex items-center justify-center gap-2 cursor-pointer ${
                      canRefine
                        ? "bg-gradient-to-r from-amber-500 to-orange-500 hover:from-amber-400 hover:to-orange-400 text-zinc-950 shadow-lg shadow-amber-950/20 active:scale-[0.98]"
                        : "bg-zinc-900 border border-zinc-800 text-zinc-500 cursor-not-allowed"
                    }`}
                  >
                    {isRefining ? (
                      <>
                        <RefreshCw className="w-4 h-4 animate-spin text-zinc-950" />
                        <span>{furnaceStage}</span>
                      </>
                    ) : (
                      <>
                        <Flame className="w-4 h-4 text-zinc-950 animate-pulse" />
                        <span>HỢP LUYỆN CỔ TRÙNG</span>
                      </>
                    )}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* 3. FULL SCREEN RESULT MODAL OVERLAY (GACHA SHOWCASE) */}
        {refineResult && (
          <div className="absolute inset-0 bg-zinc-950/95 backdrop-filter backdrop-blur-md flex flex-col justify-center items-center z-50 p-6 animate-fade-in text-center">
            {/* Flash Effect on start */}
            <div className="absolute inset-0 bg-white opacity-0 pointer-events-none animate-flash z-50" />

            {refineResult.status === "success" ? (
              <div className="space-y-6 max-w-sm w-full animate-gold-glow p-6 rounded-3xl border border-amber-500/40 bg-zinc-950 relative overflow-hidden">
                <div className="absolute inset-0 bg-radial-gradient from-amber-500/10 to-transparent pointer-events-none" />

                {/* Celebration header */}
                <div className="space-y-1">
                  <div className="w-14 h-14 rounded-full bg-amber-500/20 flex items-center justify-center mx-auto mb-2 shadow-[0_0_15px_rgba(234,179,8,0.2)]">
                    <Sparkles className="w-7 h-7 text-amber-400 animate-spin" />
                  </div>
                  <h3 className="text-xl font-extrabold text-amber-400 font-serif tracking-wider">
                    HỢP LUYỆN THÀNH CÔNG!
                  </h3>
                  <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono block">
                    Đã Khai Sinh Tân Cổ Trùng
                  </span>
                </div>

                {/* Show created Gu bug specs */}
                {refineResult.new_gu && (
                  <div className="p-4 rounded-2xl bg-zinc-950 border border-zinc-900 space-y-2.5">
                    <div className="flex justify-between items-center">
                      <span className="text-base font-bold text-zinc-100 font-serif">
                        {guDictionary[refineResult.new_gu.gu_id]?.name || refineResult.new_gu.gu_id}
                      </span>
                      <span className="text-[9px] font-mono border border-amber-500/30 px-1.5 py-0.2 rounded text-amber-400 font-bold uppercase">
                        Rank {guDictionary[refineResult.new_gu.gu_id]?.rank || 1}
                      </span>
                    </div>
                    <div className="text-[9px] text-left text-zinc-500 font-mono">
                      Loại: {guDictionary[refineResult.new_gu.gu_id]?.gu_type || "Thượng Cổ"}
                    </div>
                    <p className="text-[11px] text-left text-zinc-400 leading-relaxed italic border-t border-zinc-900 pt-2">
                      {guDictionary[refineResult.new_gu.gu_id]?.description || ""}
                    </p>
                  </div>
                )}

                <p className="text-xs text-emerald-400 font-semibold px-2">
                  {refineResult.message}
                </p>

                <button
                  onClick={() => {
                    setRefineResult(null);
                    setSlottedGus({});
                  }}
                  className="w-full py-2.5 px-4 rounded-xl text-xs font-bold bg-amber-400 hover:bg-amber-300 text-zinc-950 transition-all font-serif cursor-pointer tracking-wider"
                >
                  XÁC NHẬN THU HỒI
                </button>
              </div>
            ) : (
              <div className="space-y-6 max-w-sm w-full p-6 rounded-3xl border border-rose-500/20 bg-zinc-950 relative overflow-hidden">
                <div className="space-y-2">
                  <div className="w-14 h-14 rounded-full bg-rose-500/10 flex items-center justify-center mx-auto mb-2">
                    <AlertCircle className="w-7 h-7 text-rose-500" />
                  </div>
                  <h3 className="text-lg font-bold text-rose-500 font-serif tracking-wider">
                    HỢP LUYỆN THẤT BẠI!
                  </h3>
                  <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono block">
                    Khói đen nghi ngút, hỏa lò tro tàn
                  </span>
                </div>

                <p className="text-xs text-zinc-400 max-w-xs mx-auto leading-relaxed border border-zinc-900 p-4 rounded-2xl italic bg-zinc-950/60">
                  {refineResult.message}
                </p>

                <button
                  onClick={() => {
                    setRefineResult(null);
                    setSlottedGus({});
                  }}
                  className="w-full py-2.5 px-4 rounded-xl text-xs font-bold bg-zinc-900 hover:bg-zinc-850 text-zinc-300 transition-all font-serif cursor-pointer tracking-wider border border-zinc-800"
                >
                  RÚT KINH NGHIỆM
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
