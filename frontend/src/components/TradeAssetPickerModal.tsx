"use client";

import React, { useState, useMemo } from "react";
import {
  Search,
  X,
  Plus,
  Minus,
  Check,
  Package,
  Bug,
  ScrollText,
  Filter,
} from "lucide-react";
import { GameModal } from "@/components/ui/GameModal";
import { GameButton } from "@/components/ui/GameButton";
import { GameBadge } from "@/components/ui/GameBadge";
import type { Player, GuInstance, GuSpec, ItemSpec, RefiningRecipe } from "@/store/useGameStore";
import { guSpecFor } from "@/store/useGameStore";

export type AssetTab = "material" | "gu" | "recipe";

interface TradeAssetPickerModalProps {
  open: boolean;
  onClose: () => void;
  mode: "offer" | "request";
  initialTab?: AssetTab;
  player: Player;
  guInventory: GuInstance[];
  recipes: RefiningRecipe[];
  itemDictionary: Record<string, ItemSpec>;
  guDictionary: Record<string, GuSpec>;

  // Current selections
  offeredMats: Record<string, number>;
  offeredGuIDs: string[];
  offeredRecipes: Record<string, number>;

  requestedMats: Record<string, number>;
  requestedGuIDs: string[];
  requestedRecipes: Record<string, number>;

  // Callback additions
  onAddMaterial: (matId: string, quantity: number) => void;
  onAddGu: (guIdentifier: string) => void;
  onAddRecipe: (recipeId: string, completionRate?: number) => void;
}

export function TradeAssetPickerModal({
  open,
  onClose,
  mode,
  initialTab = "material",
  player,
  guInventory,
  recipes,
  itemDictionary,
  guDictionary,
  offeredMats,
  offeredGuIDs,
  offeredRecipes,
  requestedMats,
  requestedGuIDs,
  requestedRecipes,
  onAddMaterial,
  onAddGu,
  onAddRecipe,
}: TradeAssetPickerModalProps) {
  const [activeTab, setActiveTab] = useState<AssetTab>(initialTab);
  const [searchQuery, setSearchQuery] = useState("");
  const [rankFilter, setRankFilter] = useState<number | "all">("all");

  // Selection state within the picker
  const [selectedMatId, setSelectedMatId] = useState<string | null>(null);
  const [matQuantity, setMatQuantity] = useState<number>(1);
  const [actionFeedback, setActionFeedback] = useState<string | null>(null);

  // Sync initial tab when opening
  React.useEffect(() => {
    if (open) {
      setActiveTab(initialTab);
      setSearchQuery("");
      setRankFilter("all");
      setSelectedMatId(null);
      setMatQuantity(1);
      setActionFeedback(null);
    }
  }, [open, initialTab]);

  const showFeedback = (msg: string) => {
    setActionFeedback(msg);
    setTimeout(() => {
      setActionFeedback(null);
    }, 2500);
  };

  // 1. Materials List
  const filteredMaterials = useMemo(() => {
    const q = searchQuery.toLowerCase().trim();
    if (mode === "offer") {
      // From player's materials_inventory
      const owned = Object.entries(player.materials_inventory || {})
        .filter(([id, qty]) => qty > 0 && id !== "primeval_stone" && id !== "immortal_stone")
        .map(([id, qty]) => {
          const spec = itemDictionary[id];
          const name = spec?.name || id;
          const inOffer = offeredMats[id] || 0;
          const avail = Math.max(0, qty - inOffer);
          return {
            id,
            name,
            totalOwned: qty,
            inOffer,
            avail,
            description: spec?.description || "",
            type: spec?.type || "Material",
          };
        });

      return owned.filter(
        (m) =>
          !q ||
          m.name.toLowerCase().includes(q) ||
          m.id.toLowerCase().includes(q)
      );
    } else {
      // From encyclopedia / itemDictionary
      const allItems = Object.entries(itemDictionary)
        .filter(([id]) => id !== "primeval_stone" && id !== "immortal_stone")
        .map(([id, spec]) => ({
          id,
          name: spec.name || id,
          description: spec.description || "",
          type: spec.type || "Material",
          inRequested: requestedMats[id] || 0,
        }));

      return allItems.filter(
        (m) =>
          !q ||
          m.name.toLowerCase().includes(q) ||
          m.id.toLowerCase().includes(q)
      );
    }
  }, [mode, player.materials_inventory, itemDictionary, offeredMats, requestedMats, searchQuery]);

  // 2. Gus List
  const filteredGus = useMemo(() => {
    const q = searchQuery.toLowerCase().trim();
    if (mode === "offer") {
      // From player's guInventory
      return guInventory
        .filter((g) => !g.is_vital && !g.equipped && g.status !== "DEAD")
        .map((g) => {
          const spec = guSpecFor(g.gu_id, g.rank);
          const name = spec?.name || g.gu_id;
          const isAdded = offeredGuIDs.includes(g.instance_id);
          return {
            instanceId: g.instance_id,
            guId: g.gu_id,
            name,
            rank: g.rank,
            isAdded,
          };
        })
        .filter((g) => {
          if (rankFilter !== "all" && g.rank !== rankFilter) return false;
          if (q && !g.name.toLowerCase().includes(q) && !g.guId.toLowerCase().includes(q)) return false;
          return true;
        });
    } else {
      // From encyclopedia / guDictionary
      return Object.entries(guDictionary || {})
        .map(([gid, spec]) => {
          const name = (spec as GuSpec)?.name || gid;
          const rank = (spec as GuSpec)?.rank || 1;
          const isRequested = requestedGuIDs.includes(gid);
          return {
            guId: gid,
            name,
            rank,
            isRequested,
            description: (spec as GuSpec)?.description || "",
          };
        })
        .filter((g) => {
          if (rankFilter !== "all" && g.rank !== rankFilter) return false;
          if (q && !g.name.toLowerCase().includes(q) && !g.guId.toLowerCase().includes(q)) return false;
          return true;
        })
        .sort((a, b) => a.name.localeCompare(b.name));
    }
  }, [mode, guInventory, guDictionary, offeredGuIDs, requestedGuIDs, rankFilter, searchQuery]);

  // 3. Recipes List
  const filteredRecipes = useMemo(() => {
    const q = searchQuery.toLowerCase().trim();
    if (mode === "offer") {
      // From player.unlocked_recipes
      return (player.unlocked_recipes || [])
        .map((ur) => {
          const rSpec = recipes.find((r) => r.id === ur.recipe_id);
          const targetGu = rSpec?.target_gu_id
            ? guSpecFor(rSpec.target_gu_id, rSpec.target_rank || 1)?.name
            : "";
          const name = targetGu ? `Bí phương ${targetGu}` : ur.recipe_id;
          const isAdded = Boolean(offeredRecipes[ur.recipe_id]);
          return {
            recipeId: ur.recipe_id,
            name,
            targetRank: rSpec?.target_rank || 1,
            completionRate: ur.completion_rate,
            isAdded,
          };
        })
        .filter((r) => {
          if (!q) return true;
          return (
            r.name.toLowerCase().includes(q) ||
            r.recipeId.toLowerCase().includes(q)
          );
        });
    } else {
      // From all recipes
      return recipes
        .map((r) => {
          const targetGu = r.target_gu_id
            ? guSpecFor(r.target_gu_id, r.target_rank || 1)?.name
            : "";
          const name = targetGu ? `Bí phương ${targetGu}` : r.id;
          const isRequested = Boolean(requestedRecipes[r.id]);
          return {
            recipeId: r.id,
            name,
            targetRank: r.target_rank || 1,
            isRequested,
          };
        })
        .filter((r) => {
          if (!q) return true;
          return (
            r.name.toLowerCase().includes(q) ||
            r.recipeId.toLowerCase().includes(q)
          );
        });
    }
  }, [mode, player.unlocked_recipes, recipes, offeredRecipes, requestedRecipes, searchQuery]);

  // Selected Material helper
  const activeMat = useMemo(() => {
    if (!selectedMatId) return null;
    return filteredMaterials.find((m) => m.id === selectedMatId) || null;
  }, [selectedMatId, filteredMaterials]);

  // Add material handler
  const handleAddMaterial = () => {
    if (!selectedMatId || matQuantity <= 0) return;
    if (mode === "offer" && activeMat && "avail" in activeMat) {
      const avail = (activeMat as { avail: number }).avail;
      if (avail <= 0) return;
      const toAdd = Math.min(matQuantity, avail);
      onAddMaterial(selectedMatId, toAdd);
      showFeedback(`✓ Đã thêm ${toAdd}× ${activeMat.name}`);
    } else if (activeMat) {
      onAddMaterial(selectedMatId, matQuantity);
      showFeedback(`✓ Đã yêu cầu ${matQuantity}× ${activeMat.name}`);
    }
    setMatQuantity(1);
  };

  const isOffer = mode === "offer";
  const themeText = isOffer ? "text-emerald-400" : "text-amber-400";
  const themeBorder = isOffer ? "border-emerald-500" : "border-amber-500";
  const themeBg = isOffer ? "bg-emerald-950/20" : "bg-amber-950/20";
  const themeRing = isOffer ? "ring-emerald-500/40" : "ring-amber-500/40";
  const themeBtnBg = isOffer ? "bg-emerald-600" : "bg-amber-600";

  return (
    <GameModal
      open={open}
      onClose={onClose}
      size="2xl"
      title={
        isOffer
          ? "📤 Chọn Tài Sản Đem Trao Đổi (Trong Túi)"
          : "📥 Chọn Tài Sản Yêu Cầu Nhận (Bách Khoa)"
      }
      description={
        isOffer
          ? "Chọn vật phẩm, cổ trùng hoặc bí phương từ túi đồ của bạn để đưa vào đơn trao đổi."
          : "Tìm kiếm vật phẩm, cổ trùng hoặc bí phương mà bạn muốn người khác cung cấp."
      }
      variant={isOffer ? "success" : "warning"}
      footer={
        <div className="flex items-center justify-between w-full">
          <div className="text-xs">
            {actionFeedback && (
              <span className={`font-semibold ${themeText} animate-pulse`}>
                {actionFeedback}
              </span>
            )}
          </div>
          <GameButton variant="primary" onClick={onClose}>
            Hoàn Tất
          </GameButton>
        </div>
      }
    >
      <div className="space-y-4">
        {/* Navigation Tabs */}
        <div className="flex border-b border-zinc-800">
          <button
            type="button"
            onClick={() => {
              setActiveTab("material");
              setSelectedMatId(null);
            }}
            className={`flex-1 py-2.5 text-xs font-bold flex items-center justify-center gap-1.5 border-b-2 transition-all cursor-pointer ${
              activeTab === "material"
                ? `${themeText} ${themeBorder} ${themeBg}`
                : "text-zinc-500 border-transparent hover:text-zinc-300"
            }`}
          >
            <Package className="w-4 h-4" />
            <span>Vật Phẩm</span>
            <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-zinc-800 text-zinc-400">
              {filteredMaterials.length}
            </span>
          </button>

          <button
            type="button"
            onClick={() => {
              setActiveTab("gu");
              setSelectedMatId(null);
            }}
            className={`flex-1 py-2.5 text-xs font-bold flex items-center justify-center gap-1.5 border-b-2 transition-all cursor-pointer ${
              activeTab === "gu"
                ? `${themeText} ${themeBorder} ${themeBg}`
                : "text-zinc-500 border-transparent hover:text-zinc-300"
            }`}
          >
            <Bug className="w-4 h-4" />
            <span>Cổ Trùng</span>
            <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-zinc-800 text-zinc-400">
              {filteredGus.length}
            </span>
          </button>

          <button
            type="button"
            onClick={() => {
              setActiveTab("recipe");
              setSelectedMatId(null);
            }}
            className={`flex-1 py-2.5 text-xs font-bold flex items-center justify-center gap-1.5 border-b-2 transition-all cursor-pointer ${
              activeTab === "recipe"
                ? `${themeText} ${themeBorder} ${themeBg}`
                : "text-zinc-500 border-transparent hover:text-zinc-300"
            }`}
          >
            <ScrollText className="w-4 h-4" />
            <span>Bí Phương</span>
            <span className="text-[10px] px-1.5 py-0.2 rounded-full bg-zinc-800 text-zinc-400">
              {filteredRecipes.length}
            </span>
          </button>
        </div>

        {/* Search & Filter Header */}
        <div className="space-y-2">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-zinc-500" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={
                activeTab === "material"
                  ? "Tìm kiếm tên vật phẩm, tiên tài..."
                  : activeTab === "gu"
                  ? "Tìm kiếm tên hoặc mã cổ trùng..."
                  : "Tìm kiếm bí phương..."
              }
              className="w-full bg-zinc-900 border border-zinc-800 rounded-xl pl-9 pr-8 py-2 text-xs text-zinc-200 placeholder:text-zinc-600 focus:outline-none focus:border-zinc-700"
            />
            {searchQuery && (
              <button
                type="button"
                onClick={() => setSearchQuery("")}
                className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-zinc-300"
              >
                <X className="w-3.5 h-3.5" />
              </button>
            )}
          </div>

          {/* Gu Rank Pills */}
          {activeTab === "gu" && (
            <div className="flex items-center gap-1 overflow-x-auto pb-1 text-[11px]">
              <span className="text-zinc-500 flex items-center gap-1 text-[10px] font-mono mr-1">
                <Filter className="w-3 h-3" /> Hạng:
              </span>
              <button
                type="button"
                onClick={() => setRankFilter("all")}
                className={`px-2 py-0.5 rounded-lg text-xs font-mono transition-colors ${
                  rankFilter === "all"
                    ? `${themeBtnBg} text-white font-bold`
                    : "bg-zinc-900 text-zinc-400 hover:bg-zinc-800"
                }`}
              >
                Tất cả
              </button>
              {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((r) => (
                <button
                  key={r}
                  type="button"
                  onClick={() => setRankFilter(r)}
                  className={`px-2 py-0.5 rounded-lg text-xs font-mono transition-colors ${
                    rankFilter === r
                      ? `${themeBtnBg} text-white font-bold`
                      : "bg-zinc-900 text-zinc-400 hover:bg-zinc-800"
                  }`}
                >
                  H{r}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* List Content */}
        <div className="max-h-[300px] overflow-y-auto space-y-1.5 pr-1 min-h-[160px]">
          {/* TAB 1: MATERIALS */}
          {activeTab === "material" && (
            <>
              {filteredMaterials.length === 0 ? (
                <div className="text-center py-8 text-zinc-500 text-xs italic">
                  Không tìm thấy vật phẩm nào phù hợp.
                </div>
              ) : (
                filteredMaterials.map((mat) => {
                  const isSelected = selectedMatId === mat.id;
                  const isOfferMode = isOffer && "avail" in mat;
                  const avail = isOfferMode ? (mat as { avail: number }).avail : 999999;
                  const isDisabled = isOfferMode && avail <= 0;

                  return (
                    <div
                      key={mat.id}
                      onClick={() => {
                        if (!isDisabled) {
                          setSelectedMatId(mat.id);
                          setMatQuantity(1);
                        }
                      }}
                      className={`flex items-center justify-between p-2.5 rounded-xl border transition-all cursor-pointer ${
                        isSelected
                          ? `${themeBg} border-${themeBorder} ring-1 ${themeRing}`
                          : isDisabled
                          ? "bg-zinc-950/40 border-zinc-900 opacity-50 cursor-not-allowed"
                          : "bg-zinc-900/60 border-zinc-800/80 hover:bg-zinc-850 hover:border-zinc-700"
                      }`}
                    >
                      <div className="flex items-center gap-2.5 min-w-0">
                        <div className="w-8 h-8 rounded-lg bg-zinc-800 flex items-center justify-center shrink-0 text-zinc-300">
                          <Package className="w-4 h-4 text-zinc-400" />
                        </div>
                        <div className="min-w-0">
                          <p className="text-xs font-semibold text-zinc-200 truncate">
                            {mat.name}
                          </p>
                          <p className="text-[10px] text-zinc-500 font-mono truncate">
                            {mat.id}
                          </p>
                        </div>
                      </div>

                      <div className="text-right shrink-0">
                        {isOfferMode ? (
                          <div>
                            <span
                              className={`text-xs font-mono font-medium ${
                                avail > 0 ? themeText : "text-zinc-600"
                              }`}
                            >
                              Còn: {avail} / {(mat as { totalOwned: number }).totalOwned}
                            </span>
                            {isDisabled && (
                              <p className="text-[9px] text-zinc-600">Đã chọn hết</p>
                            )}
                          </div>
                        ) : (
                          <div>
                            {"inRequested" in mat && (mat as { inRequested: number }).inRequested > 0 && (
                              <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-amber-950/40 border border-amber-800/40 text-amber-400">
                                Đang cần: {(mat as { inRequested: number }).inRequested}
                              </span>
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })
              )}
            </>
          )}

          {/* TAB 2: GU */}
          {activeTab === "gu" && (
            <>
              {filteredGus.length === 0 ? (
                <div className="text-center py-8 text-zinc-500 text-xs italic">
                  Không tìm thấy Cổ Trùng nào phù hợp.
                </div>
              ) : (
                filteredGus.map((g) => {
                  const isOfferMode = isOffer && "instanceId" in g;
                  const isAdded = isOfferMode
                    ? (g as { isAdded: boolean }).isAdded
                    : (g as { isRequested: boolean }).isRequested;

                  return (
                    <div
                      key={isOfferMode ? (g as { instanceId: string }).instanceId : g.guId}
                      className={`flex items-center justify-between p-2.5 rounded-xl border transition-all ${
                        isAdded
                          ? "bg-zinc-950/50 border-zinc-900 opacity-60"
                          : "bg-zinc-900/60 border-zinc-800/80 hover:bg-zinc-850 hover:border-zinc-700"
                      }`}
                    >
                      <div className="flex items-center gap-2.5 min-w-0">
                        <div className="w-8 h-8 rounded-lg bg-zinc-800 flex items-center justify-center shrink-0">
                          <Bug className="w-4 h-4 text-emerald-400" />
                        </div>
                        <div className="min-w-0">
                          <div className="flex items-center gap-1.5">
                            <GameBadge variant="rank" rank={g.rank} size="sm">
                              Hạng {g.rank}
                            </GameBadge>
                            <span className="text-xs font-semibold text-zinc-200 truncate">
                              {g.name}
                            </span>
                          </div>
                          <p className="text-[10px] text-zinc-500 font-mono truncate">
                            {g.guId}
                          </p>
                        </div>
                      </div>

                      <div className="shrink-0">
                        {isAdded ? (
                          <span className="text-[11px] font-mono text-zinc-500 flex items-center gap-1">
                            <Check className="w-3.5 h-3.5 text-zinc-500" /> Đã chọn
                          </span>
                        ) : (
                          <GameButton
                            size="xs"
                            variant={isOffer ? "secondary" : "warning"}
                            type="button"
                            onClick={() => {
                              if (isOfferMode) {
                                onAddGu((g as { instanceId: string }).instanceId);
                                showFeedback(`✓ Đã thêm ${g.name}`);
                              } else {
                                onAddGu(g.guId);
                                showFeedback(`✓ Đã yêu cầu ${g.name}`);
                              }
                            }}
                          >
                            + Thêm
                          </GameButton>
                        )}
                      </div>
                    </div>
                  );
                })
              )}
            </>
          )}

          {/* TAB 3: RECIPES */}
          {activeTab === "recipe" && (
            <>
              {filteredRecipes.length === 0 ? (
                <div className="text-center py-8 text-zinc-500 text-xs italic">
                  Không tìm thấy Bí Phương nào phù hợp.
                </div>
              ) : (
                filteredRecipes.map((r) => {
                  const isOfferMode = isOffer && "completionRate" in r;
                  const isAdded = isOfferMode
                    ? (r as { isAdded: boolean }).isAdded
                    : (r as { isRequested: boolean }).isRequested;

                  return (
                    <div
                      key={r.recipeId}
                      className={`flex items-center justify-between p-2.5 rounded-xl border transition-all ${
                        isAdded
                          ? "bg-zinc-950/50 border-zinc-900 opacity-60"
                          : "bg-zinc-900/60 border-zinc-800/80 hover:bg-zinc-850 hover:border-zinc-700"
                      }`}
                    >
                      <div className="flex items-center gap-2.5 min-w-0">
                        <div className="w-8 h-8 rounded-lg bg-zinc-800 flex items-center justify-center shrink-0">
                          <ScrollText className="w-4 h-4 text-violet-400" />
                        </div>
                        <div className="min-w-0">
                          <p className="text-xs font-semibold text-zinc-200 truncate">
                            {r.name}
                          </p>
                          <p className="text-[10px] text-zinc-500 font-mono truncate">
                            {r.recipeId} {isOfferMode && `• Độ hoàn thiện: ${(r as { completionRate: number }).completionRate}%`}
                          </p>
                        </div>
                      </div>

                      <div className="shrink-0">
                        {isAdded ? (
                          <span className="text-[11px] font-mono text-zinc-500 flex items-center gap-1">
                            <Check className="w-3.5 h-3.5 text-zinc-500" /> Đã chọn
                          </span>
                        ) : (
                          <GameButton
                            size="xs"
                            variant={isOffer ? "secondary" : "warning"}
                            type="button"
                            onClick={() => {
                              if (isOfferMode) {
                                onAddRecipe(r.recipeId, (r as { completionRate: number }).completionRate);
                                showFeedback(`✓ Đã thêm ${r.name}`);
                              } else {
                                onAddRecipe(r.recipeId, 100);
                                showFeedback(`✓ Đã yêu cầu ${r.name}`);
                              }
                            }}
                          >
                            + Thêm
                          </GameButton>
                        )}
                      </div>
                    </div>
                  );
                })
              )}
            </>
          )}
        </div>

        {/* Selected Material Quantity Stepper (Active when an item in Material tab is selected) */}
        {activeTab === "material" && activeMat && (
          <div className="bg-zinc-950 p-3 rounded-xl border border-zinc-800 space-y-2.5">
            <div className="flex items-center justify-between">
              <span className="text-xs text-zinc-300 font-medium">
                Số lượng: <strong className="text-white">{activeMat.name}</strong>
              </span>
              {isOffer && "avail" in activeMat && (
                <span className="text-[11px] text-zinc-500 font-mono">
                  Tối đa khả dụng: {(activeMat as { avail: number }).avail}
                </span>
              )}
            </div>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setMatQuantity((q) => Math.max(1, q - 1))}
                className="w-8 h-8 rounded-lg bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 flex items-center justify-center text-zinc-300"
              >
                <Minus className="w-3.5 h-3.5" />
              </button>
              <input
                type="number"
                min={1}
                max={isOffer && "avail" in activeMat ? (activeMat as { avail: number }).avail : undefined}
                value={matQuantity}
                onChange={(e) => setMatQuantity(Math.max(1, Number(e.target.value) || 1))}
                className="w-20 bg-zinc-900 border border-zinc-800 rounded-lg py-1 px-2 text-center text-xs font-mono text-zinc-200"
              />
              <button
                type="button"
                onClick={() => {
                  const maxAvail = isOffer && "avail" in activeMat ? (activeMat as { avail: number }).avail : 999999;
                  setMatQuantity((q) => Math.min(maxAvail, q + 1));
                }}
                className="w-8 h-8 rounded-lg bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 flex items-center justify-center text-zinc-300"
              >
                <Plus className="w-3.5 h-3.5" />
              </button>

              <div className="flex items-center gap-1">
                {[5, 10, 50].map((step) => (
                  <button
                    key={step}
                    type="button"
                    onClick={() => {
                      const maxAvail = isOffer && "avail" in activeMat ? (activeMat as { avail: number }).avail : 999999;
                      setMatQuantity((q) => Math.min(maxAvail, q + step));
                    }}
                    className="text-[10px] font-mono px-2 py-1 rounded bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 text-zinc-400"
                  >
                    +{step}
                  </button>
                ))}
                {isOffer && "avail" in activeMat && (
                  <button
                    type="button"
                    onClick={() => setMatQuantity((activeMat as { avail: number }).avail)}
                    className="text-[10px] font-mono px-2 py-1 rounded bg-emerald-950/40 hover:bg-emerald-900/60 border border-emerald-800/50 text-emerald-400 font-bold"
                  >
                    Tối Đa
                  </button>
                )}
              </div>

              <div className="flex-1 text-right">
                <GameButton
                  size="sm"
                  variant={isOffer ? "primary" : "warning"}
                  type="button"
                  onClick={handleAddMaterial}
                >
                  {isOffer ? `Thêm Vào Đơn (+${matQuantity})` : `Yêu Cầu (+${matQuantity})`}
                </GameButton>
              </div>
            </div>
          </div>
        )}
      </div>
    </GameModal>
  );
}
