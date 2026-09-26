"use client";

import React, { useState, useEffect, useRef, useMemo } from "react";
import { hasRefinementMechanic } from "@/lib/effect-blocks";
import { GamePanel, GameCard } from "@/components/ui";
// UI migrated to shared components
import { Player, GuSpec, ItemSpec, RefiningRecipe, GuInstance, useGameStore, guSpecFor, RefineRateBreakdown } from "@/store/useGameStore";
import { reqGuQty, reqGuRank } from "@/lib/gu-dict";
import { Flame, Sparkles, AlertCircle, Check, X, RefreshCw, Coins, Zap, Trophy, HelpCircle, ChevronDown, Search, Brain, ShieldCheck, Clock, TrendingUp, Award } from "lucide-react";

interface RefiningFurnaceProps {
  player: Player;
  guDictionary: Record<string, GuSpec>;
  itemDictionary: Record<string, ItemSpec>;
  actionLoading: boolean;
  onRefine: (recipeId: string, selectedInstanceIds: string[]) => Promise<{ status: "success" | "failed"; message: string; new_gu?: GuInstance; rate_breakdown?: RefineRateBreakdown } | null>;
}

export const RefiningFurnace: React.FC<RefiningFurnaceProps> = ({
  player,
  guDictionary,
  itemDictionary,
  actionLoading,
  onRefine,
}) => {
  const fetchRecipes = useGameStore((state) => state.fetchRecipes);
  const recipes = useGameStore((state) => state.recipes);

  useEffect(() => {
    fetchRecipes();
  }, [fetchRecipes]);
  const [selectedRecipeId, setSelectedRecipeId] = useState<string | null>(null);
  const [slottedGus, setSlottedGus] = useState<Record<number, string>>({}); // slotIndex -> guInstanceId
  const [isRefining, setIsRefining] = useState(false);
  const [furnaceStage, setFurnaceStage] = useState("Đang khởi động lò luyện...");
  const [refineResult, setRefineResult] = useState<{ status: "success" | "failed"; new_gu?: GuInstance; message: string; rate_breakdown?: RefineRateBreakdown } | null>(null);
  const [searchKeyword, setSearchKeyword] = useState("");
  const [customRecipes, setCustomRecipes] = useState<RefiningRecipe[]>([]);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [expandedGroups, setExpandedGroups] = useState<Record<string, boolean>>({});
  const [statusTab, setStatusTab] = useState<"all" | "complete" | "incomplete">("all");
  const [isDeducing, setIsDeducing] = useState(false);
  const [showFullDesc, setShowFullDesc] = useState(false);

  const consumeGu = useGameStore((state) => state.useGu);
  const [isUsingRestoreGu, setIsUsingRestoreGu] = useState(false);

  const restoreMentalGus = (player.gu_inventory || []).filter((gu) => {
    if (gu.status === "DEAD" || gu.status === "SEALED") return false;
    if (gu.gu_id === "GU_THOUGHT_RECOVERY") return true;
    const spec = guSpecFor(gu.gu_id, gu.rank) || guDictionary[gu.gu_id];
    if (!spec) return false;
    return hasRefinementMechanic(spec.effects, "RESTORE_MENTAL");
  });

  const handleQuickUseRestoreMental = async () => {
    if (restoreMentalGus.length === 0 || isUsingRestoreGu || actionLoading) return;
    setIsUsingRestoreGu(true);
    try {
      const targetGu = restoreMentalGus[0];
      await consumeGu(targetGu.instance_id);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Lỗi khi sử dụng Cổ trùng hồi Niệm Lực";
      addToast(msg, "error");
    } finally {
      setIsUsingRestoreGu(false);
    }
  };

  const toggleGroupExpand = (guId: string, event?: React.MouseEvent) => {
    if (event) event.stopPropagation();
    setExpandedGroups(prev => ({ ...prev, [guId]: !prev[guId] }));
  };
  const deduceRecipe = useGameStore((state) => state.deduceRecipe);
  const addToast = useGameStore((state) => state.addToast);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

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
    const targetName = guSpecFor(r.target_gu_id, r.target_rank)?.name || r.target_gu_id;
    const matchesSearch = targetName.toLowerCase().includes(searchKeyword.toLowerCase()) || r.id.toLowerCase().includes(searchKeyword.toLowerCase());
    const isComplete = r.completion_rate === undefined || r.completion_rate >= 100.0;

    if (statusTab === "complete") {
      return matchesSearch && isComplete;
    }
    if (statusTab === "incomplete") {
      return matchesSearch && !isComplete;
    }
    return matchesSearch;
  });
  const selectedRecipe = allRecipes.find((r) => r.id === selectedRecipeId);

  // Get masked status for current recipe
  const getMaskedStatus = (recipe?: RefiningRecipe) => {
    if (!recipe) return { maskedMats: new Set<string>(), maskedGus: new Set<string>() };
    const rate = recipe.completion_rate !== undefined ? recipe.completion_rate : 100;
    const isComplete = rate >= 100;

    // Get all materials
    const materials = Object.entries(recipe.required_materials).sort((a, b) => a[0].localeCompare(b[0]));
    const totalMats = materials.length;
    const numMaskedMats = isComplete ? 0 : Math.ceil(((100 - rate) / 100) * totalMats);
    const maskedMats = new Set<string>();
    for (let i = totalMats - numMaskedMats; i < totalMats; i++) {
      if (materials[i]) {
        maskedMats.add(materials[i][0]);
      }
    }

    // Get all required Gus
    const gus = Object.entries(recipe.required_gus).sort((a, b) => a[0].localeCompare(b[0]));
    const totalGus = gus.length;
    const numMaskedGus = isComplete ? 0 : Math.ceil(((100 - rate) / 100) * totalGus);
    const maskedGus = new Set<string>();
    for (let i = totalGus - numMaskedGus; i < totalGus; i++) {
      if (gus[i]) {
        maskedGus.add(gus[i][0]);
      }
    }

    return { maskedMats, maskedGus };
  };

  const { maskedMats, maskedGus } = getMaskedStatus(selectedRecipe);

  // Flatten the required Gus into an ordered list of slots
  const getRequiredSlots = () => {
    if (!selectedRecipe) return [];
    const slots: { index: number; guId: string; rank: number }[] = [];
    let idx = 0;
    Object.entries(selectedRecipe.required_gus).forEach(([guId, value]) => {
      const qty = reqGuQty(value);
      const rank = reqGuRank(value);
      for (let i = 0; i < qty; i++) {
        slots.push({ index: idx++, guId, rank });
      }
    });
    return slots;
  };

  const requiredSlots = getRequiredSlots();

  // Find all Gus currently in player inventory that are NOT dead and NOT slotted
  const slottedIds = useMemo(() => Object.values(slottedGus), [slottedGus]);
  const availableInventoryGus = useMemo(() => {
    return (player.gu_inventory || []).filter(
      (gu) => gu.status !== "DEAD" && !slottedIds.includes(gu.instance_id)
    );
  }, [player.gu_inventory, slottedIds]);

  const getBestCandidateForRefine = (guId: string, rank: number): GuInstance | undefined => {
    const candidates = availableInventoryGus.filter(g => g.gu_id === guId && g.rank === rank);
    if (candidates.length === 0) return undefined;

    // Sort: non-vital & non-equipped first, then lowest hunger, then sleeping
    candidates.sort((a, b) => {
      if (a.is_vital !== b.is_vital) return a.is_vital ? 1 : -1;
      if (a.equipped !== b.equipped) return a.equipped ? 1 : -1;
      return a.hunger - b.hunger;
    });
    return candidates[0];
  };

  const handleInventoryGroupClick = (guId: string, rank: number) => {
    const candidate = getBestCandidateForRefine(guId, rank);
    if (candidate) {
      handleInventoryClick(candidate);
    }
  };

  // Handle slotting a Gu by instanceId into a target slot index.
  // Any Gu may be placed (drag-drop experiment); the recipe match is validated at refine time.
  const slotGu = (instanceId: string, targetSlotIndex: number) => {
    const guInstance = player.gu_inventory.find((g) => g.instance_id === instanceId);
    if (!guInstance) return;

    const targetSlotRequirement = requiredSlots.find((s) => s.index === targetSlotIndex);
    if (!targetSlotRequirement) return;

    setSlottedGus((prev) => {
      const copy = { ...prev };
      // Remove from old slot if it was already slotted
      for (const [key, val] of Object.entries(copy)) {
        if (val === instanceId) {
          delete copy[Number(key)];
        }
      }
      copy[targetSlotIndex] = instanceId;
      return copy;
    });
  };

  // Auto-slot Gu by clicking it
  const handleInventoryClick = (gu: GuInstance) => {
    // Find the first empty slot that requires this Gu type (and rank when specified).
    // Masked slots (Cổ Trùng Ẩn) are excluded — no auto-slot to keep the mystery.
    const emptySlot = requiredSlots.find(
      (slot) => slot.guId === gu.gu_id && !slottedGus[slot.index] && (slot.rank === 0 || slot.rank === gu.rank) && !maskedGus.has(slot.guId)
    );

    if (emptySlot !== undefined) {
      slotGu(gu.instance_id, emptySlot.index);
    }
  };

  const handleSlotClick = (slot: { index: number; guId: string; rank: number }) => {
    const slottedInstanceId = slottedGus[slot.index];
    if (slottedInstanceId) {
      removeGuFromSlot(slot.index);
    } else if (maskedGus.has(slot.guId)) {
      addToast("Cổ Trùng Ẩn — hãy kéo thả Cổ trùng vào ô này để luyện thử!", "info");
    } else {
      const matchingGu = availableInventoryGus.find(
        (g) => g.gu_id === slot.guId && (slot.rank === 0 || g.rank === slot.rank)
      );
      if (matchingGu) {
        slotGu(matchingGu.instance_id, slot.index);
      } else {
        addToast("Không có cổ trùng phù hợp trong túi đồ!", "error");
      }
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
  const hasEnoughImmortalStones = selectedRecipe
    ? (selectedRecipe.immortal_essence_stones_cost <= 0 || player.immortal_stones >= selectedRecipe.immortal_essence_stones_cost)
    : false;

  const canRefine = allSlotsFilled && hasRequiredMaterials() && hasEnoughStones && hasEnoughImmortalStones && !actionLoading && !isRefining;

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
              target_rank: response.new_gu.rank,
              required_gus: requiredSlots.reduce((acc, slot) => {
                const prev = acc[slot.guId];
                acc[slot.guId] = { rank: slot.rank, qty: (prev ? reqGuQty(prev) : 0) + 1 };
                return acc;
              }, {} as Record<string, { rank: number; qty: number }>),
              required_materials: selectedRecipe.required_materials,
              success_rate: selectedRecipe.success_rate,
              primeval_stones_cost: selectedRecipe.primeval_stones_cost,
              immortal_essence_stones_cost: selectedRecipe.immortal_essence_stones_cost,
            };
            setCustomRecipes((prev) => [discovered, ...prev]);
          }
        } else {
          setRefineResult({
            status: "failed",
            message: "Hợp luyện thất bại bí ẩn. Lò hỏa đã tắt ngúm.",
          });
        }
      } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
        setRefineResult({
          status: "failed",
          message: err.message || "Lỗi giao dịch hợp luyện.",
        });
      } finally {
        setIsRefining(false);
      }
    }, 2500);
  };

  const hasEnoughThoughtForDeduction = (player.current_thought ?? 0) >= 15;
  const hasEnoughStonesForDeduction = player.primeval_stones >= 10;
  const canDeduce = hasEnoughThoughtForDeduction && hasEnoughStonesForDeduction && !actionLoading && !isDeducing;

  const handleDeduce = async () => {
    if (!canDeduce || !selectedRecipe) return;

    setIsDeducing(true);
    try {
      await deduceRecipe(selectedRecipe.id);
    } catch (caught: unknown) {
      const err = caught instanceof Error ? caught : new Error(String(caught));
      console.error("Lỗi khi thôi diễn: ", err);
    } finally {
      setIsDeducing(false);
    }
  };

  // Drag and Drop helpers
  const handleDragStart = (e: React.DragEvent, instanceId: string) => {
    e.dataTransfer.setData("text/plain", instanceId);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleDrop = (e: React.DragEvent, targetSlotIndex: number) => {
    e.preventDefault();
    const instanceId = e.dataTransfer.getData("text/plain");
    if (instanceId) {
      slotGu(instanceId, targetSlotIndex);
    }
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
      colorClass = "text-zinc-300";
    } else if (matId === "boar_meat") {
      icon = <Flame className="w-4 h-4 text-orange-400" />;
      colorClass = "text-orange-300";
    }

    return { name, desc, icon, colorClass, bgClass };
  };

  const parseRefineMessage = (rawMessage: string, isSuccess: boolean) => {
    let text = rawMessage || "";

    let successRateStr: string | undefined;
    const rateMatch = text.match(/\(Tỉ lệ:\s*([\d.]+%)\)/i);
    if (rateMatch) {
      successRateStr = rateMatch[1];
      text = text.replace(rateMatch[0], "");
    }

    let backfireText: string | undefined;
    const backfireMatch = text.match(/(Luyện cổ CHÁY NỔ.*?\.|Ép Luyện thất bại!.*?\.)/i);
    if (backfireMatch) {
      backfireText = backfireMatch[0].trim();
      text = text.replace(backfireMatch[0], "");
    }

    let vitalSurvivedText: string | undefined;
    const vitalMatch = text.match(/(Riêng Cổ Bản Mệnh vẫn sống sót.*?!)/i);
    if (vitalMatch) {
      vitalSurvivedText = vitalMatch[0].trim();
      text = text.replace(vitalMatch[0], "");
    }

    let statusDebuffText: string | undefined;
    const statusMatch = text.match(/(\[(?:Tâm Thần Mệt Mỏi|Luyện Cổ Phản Phệ)\].*?\.|Trạng thái \[\w+.*?\.)/i);
    if (statusMatch) {
      statusDebuffText = statusMatch[0].trim();
      text = text.replace(statusMatch[0], "");
    }

    let gainsText: string | undefined;
    const gainsParts: string[] = [];
    const daoMatch = text.match(/(Ngộ Đạo tăng tiến!.*?\.)/i);
    if (daoMatch) {
      gainsParts.push(daoMatch[0].trim());
      text = text.replace(daoMatch[0], "");
    }
    const qiMatch = text.match(/(Nhân Khí \+[\d\.]+\.?)/i);
    if (qiMatch) {
      if (qiMatch[0].trim() !== "Nhân Khí +0.") {
        gainsParts.push(qiMatch[0].trim());
      }
      text = text.replace(qiMatch[0], "");
    }
    if (gainsParts.length > 0) {
      gainsText = gainsParts.join(" ");
    }

    const isBackfire = rawMessage.includes("Backfire") || rawMessage.includes("CHÁY NỔ") || rawMessage.includes("Phản phệ dữ dội");
    const isFailsafe = rawMessage.includes("bảo toàn Cổ nguyên liệu");

    let mainText = text
      .replace(/^Hợp luyện thất bại!\s*/i, "")
      .replace(/^Luyện cổ thất bại nhưng\s*/i, "")
      .replace(/\s+/g, " ")
      .trim();

    if (!mainText) {
      if (isSuccess) {
        mainText = "Hợp luyện thành công tân Cổ trùng!";
      } else if (isFailsafe) {
        mainText = "Cảnh Giới thâm hậu giúp bảo toàn Cổ nguyên liệu phụ!";
      } else {
        mainText = "Các nguyên liệu thường đã hóa thành tro bụi.";
      }
    }

    return {
      mainText,
      successRateStr,
      backfireText,
      vitalSurvivedText,
      statusDebuffText,
      gainsText,
      isBackfire,
      isFailsafe,
    };
  };

  const renderRateBreakdown = (breakdown?: RefineRateBreakdown) => {
    if (!breakdown) return null;

    return (
      <div className="text-left text-xs bg-zinc-900/80 border border-zinc-800/80 p-3.5 rounded-2xl space-y-1.5 mt-4 font-sans select-none max-w-full">
        <div className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono font-bold mb-2 pb-1 border-b border-zinc-900 flex justify-between">
          <span>Thành Phần Tỷ Lệ</span>
          <span>Gia Thành</span>
        </div>
        <div className="space-y-1">
          <div className="flex justify-between items-center text-zinc-400">
            <span className="flex items-center gap-1">📜 Tỷ lệ cơ bản:</span>
            <span className="font-mono text-zinc-300">{(breakdown.base_rate * 100).toFixed(1)}%</span>
          </div>
          {breakdown.material_quality > 0 && (
            <div className="flex justify-between items-center text-zinc-400">
              <span className="flex items-center gap-1">💎 Phẩm chất nguyên liệu:</span>
              <span className="font-mono text-emerald-400">+{(breakdown.material_quality * 100).toFixed(1)}%</span>
            </div>
          )}
          {breakdown.refinement_dao_marks > 0 && (
            <div className="flex justify-between items-center text-zinc-400">
              <span className="flex items-center gap-1">🌀 Dao văn Luyện Đạo:</span>
              <span className="font-mono text-emerald-400">+{(breakdown.refinement_dao_marks * 100).toFixed(1)}%</span>
            </div>
          )}
          {breakdown.refinement_mastery > 0 && (
            <div className="flex justify-between items-center text-zinc-400">
              <span className="flex items-center gap-1">🎓 Cảnh giới Luyện Đạo:</span>
              <span className="font-mono text-emerald-400">+{(breakdown.refinement_mastery * 100).toFixed(1)}%</span>
            </div>
          )}
          {breakdown.target_path_mastery > 0 && (
            <div className="flex justify-between items-center text-zinc-400">
              <span className="flex items-center gap-1">⚔️ Cảnh giới lưu phái:</span>
              <span className="font-mono text-emerald-400">+{(breakdown.target_path_mastery * 100).toFixed(1)}%</span>
            </div>
          )}
          {breakdown.luck_bonus > 0 && (
            <div className="flex justify-between items-center text-zinc-400">
              <span className="flex items-center gap-1">🍀 Vận khí gia thành:</span>
              <span className="font-mono text-emerald-400">+{(breakdown.luck_bonus * 100).toFixed(1)}%</span>
            </div>
          )}
          {breakdown.support_bonus > 0 && (
            <div className="flex justify-between items-center text-zinc-400">
              <span className="flex items-center gap-1">🛡️ Cổ trùng phụ trợ:</span>
              <span className="font-mono text-emerald-400">+{(breakdown.support_bonus * 100).toFixed(1)}%</span>
            </div>
          )}
          {breakdown.env_bonus > 0 && (
            <div className="flex justify-between items-center text-zinc-400">
              <span className="flex items-center gap-1">🗺️ Địa lợi (Môi trường):</span>
              <span className="font-mono text-emerald-400">+{(breakdown.env_bonus * 100).toFixed(1)}%</span>
            </div>
          )}
          {breakdown.status_penalty && breakdown.status_penalty > 0 ? (
            <div className="flex justify-between items-center text-rose-400">
              <span className="flex items-center gap-1">⚡ Phản phệ / Mệt mỏi:</span>
              <span className="font-mono text-rose-400">-{(breakdown.status_penalty * 100).toFixed(1)}%</span>
            </div>
          ) : null}
          {breakdown.completion_penalty < 1.0 && (
            <div className="flex justify-between items-center text-rose-400">
              <span className="flex items-center gap-1">⚠️ Chiết khấu tàn phương:</span>
              <span className="font-mono text-rose-400">x{(breakdown.completion_penalty * 100).toFixed(0)}%</span>
            </div>
          )}
          {!breakdown.combo_accuracy && (
            <div className="text-[10px] text-rose-400 font-bold italic mt-1 leading-tight">
              ⚠️ Cổ trùng nguyên liệu sai công thức (Tỷ lệ về 0%)!
            </div>
          )}
        </div>
        <div className="flex justify-between items-center border-t border-zinc-900 pt-2 font-bold font-serif text-amber-400">
          <span>✨ Tỷ lệ cuối cùng:</span>
          <span className="font-mono text-sm">{(breakdown.final_rate * 100).toFixed(1)}%</span>
        </div>
      </div>
    );
  };

  return (
    <GamePanel variant="default" className="grid grid-cols-1 lg:grid-cols-12 gap-6 w-full mx-auto relative">
      {/* 1. LEFT PANEL: Inventory List */}
      <div className="lg:col-span-5 glass-panel-gold rounded-3xl p-3 sm:p-5 flex flex-col max-h-[580px] overflow-hidden">
        <div className="flex items-center gap-2 border-b border-zinc-800/80 pb-3 mb-4 shrink-0">
          <Sparkles className="w-5 h-5 text-amber-500 animate-pulse" />
          <h3 className="text-md font-bold text-zinc-100 font-serif">
            Linh Khí Túi
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
            (() => {
              // Group by gu_id
              const groupsMap: Record<string, GuInstance[]> = {};
              availableInventoryGus.forEach((gu) => {
                const groupKey = `${gu.gu_id}::${gu.rank}`;
                if (!groupsMap[groupKey]) {
                  groupsMap[groupKey] = [];
                }
                groupsMap[groupKey].push(gu);
              });

              const groups = Object.entries(groupsMap).map(([groupKey, instances]) => {
                const [guId, rankStr] = groupKey.split("::");
                const rank = parseInt(rankStr, 10) || 1;
                const spec = (guSpecFor(guId, rank) || {
                  id: guId,
                  name: guId,
                  rank: rank,
                  path: ["Unknown Path"],
                  gu_type: "Phàm Cổ",
                  base_mana_cost: 0,
                  effects: {},
                  hunger_rate: 0,
                  food_type: "unknown",
                  max_hunger: 100,
                  description: "Cổ trùng chưa được giải nghĩa.",
                }) as GuSpec;
                return { groupKey, guId, spec, instances };
              });

              // Sort groups by rank descending, name ascending
              groups.sort((a, b) => {
                if (b.spec.rank !== a.spec.rank) return b.spec.rank - a.spec.rank;
                return a.spec.name.localeCompare(b.spec.name);
              });

              return groups.map(({ groupKey, guId, spec, instances }) => {
                const isExpanded = !!expandedGroups[groupKey];
                const totalAvailable = instances.length;
                const bestCandidate = getBestCandidateForRefine(guId, spec.rank);

                return (
                  <div
                    key={groupKey}
                    className="border border-zinc-900 bg-zinc-950/40 rounded-xl overflow-hidden"
                  >
                    {/* Group Header Row */}
                    <div
                      draggable={!!bestCandidate}
                      onDragStart={(e) => {
                        if (bestCandidate) {
                          handleDragStart(e, bestCandidate.instance_id);
                        }
                      }}
                      onClick={() => handleInventoryGroupClick(guId, spec.rank)}
                      className={`p-3 flex justify-between items-center cursor-grab active:cursor-grabbing hover:bg-zinc-950/80 transition-all select-none group text-left`}
                    >
                      <div className="space-y-0.5 max-w-[75%]">
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <span className="text-xs font-bold text-zinc-200 group-hover:text-amber-400 transition-colors font-serif">
                            {spec.name} (Bậc {spec.rank})
                          </span>
                          <span className="text-[9px] font-mono px-1.5 py-0.2 rounded-full bg-zinc-900 border border-zinc-850 text-zinc-400">
                            x{totalAvailable}
                          </span>
                          <span className="text-[8px] font-mono border border-amber-500/20 px-1 py-0.2 rounded text-amber-400 uppercase">
                            Bậc {spec.rank}
                          </span>
                        </div>
                        <p className="text-[9px] text-zinc-500 line-clamp-1 leading-normal">
                          {spec.description}
                        </p>
                      </div>

                      <div className="flex items-center gap-2 shrink-0">
                        <button
                          type="button"
                          onClick={(e) => toggleGroupExpand(groupKey, e)}
                          className="text-[9px] py-1 px-2 bg-zinc-900 hover:bg-zinc-850 border border-zinc-800 text-zinc-400 rounded transition-all cursor-pointer font-mono"
                        >
                          {isExpanded ? "Ẩn" : `Hiện (${totalAvailable})`}
                        </button>
                      </div>
                    </div>

                    {/* Expandable detailed candidate list */}
                    {isExpanded && (
                      <div className="bg-zinc-950/85 border-t border-zinc-900 p-2 space-y-1.5">
                        <div className="text-[9px] font-mono text-zinc-550 uppercase tracking-widest block font-bold border-b border-zinc-900 pb-1">
                          Chọn cá thể:
                        </div>
                        <div className="space-y-1 max-h-[140px] overflow-y-auto custom-scrollbar">
                          {instances.map((gu) => {
                            const isSleeping = gu.status === "SLEEPING";
                            const isEquipped = gu.equipped;
                            const isVital = gu.is_vital;

                            return (
                              <div
                                key={gu.instance_id}
                                draggable
                                onDragStart={(e) => {
                                  e.stopPropagation();
                                  handleDragStart(e, gu.instance_id);
                                }}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleInventoryClick(gu);
                                }}
                                className="p-2 rounded-lg bg-zinc-950 border border-zinc-900 hover:border-amber-500/30 flex justify-between items-center text-xs cursor-pointer transition-colors"
                              >
                                <div className="flex items-center gap-2">
                                  <span className="text-[10px] font-mono font-bold text-zinc-405">
                                    #{gu.instance_id.slice(0, 6)}
                                  </span>
                                  {isVital && (
                                    <span className="text-[8px] font-bold text-amber-300 bg-amber-950/60 border border-amber-500/60 px-1 py-0.2 rounded uppercase">
                                      Bản Mệnh
                                    </span>
                                  )}
                                  {isEquipped && (
                                    <span className="text-[8px] font-bold text-emerald-400 bg-emerald-950/20 border border-emerald-500/20 px-1 py-0.2 rounded uppercase">
                                      Xuất Trận
                                    </span>
                                  )}
                                  {isSleeping && (
                                    <span className="text-[8px] font-bold text-amber-550 bg-amber-950/20 border border-amber-500/20 px-1 py-0.2 rounded uppercase">
                                      Ngủ
                                    </span>
                                  )}
                                </div>
                                <div className="flex items-center gap-3">
                                  <span className="text-[10px] font-mono text-zinc-500">
                                    Đo no: {gu.hunger}/100
                                  </span>
                                  <button
                                    type="button"
                                    className="text-[9px] py-0.5 px-1.5 bg-amber-500/10 text-amber-400 border border-amber-500/20 rounded font-bold font-serif hover:bg-amber-500/20 transition-all cursor-pointer"
                                  >
                                    Xếp vào
                                  </button>
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      </div>
                    )}
                  </div>
                );
              });
            })()
          )}
        </div>
      </div>

      {/* 2. RIGHT PANEL: Furnace Crucible */}
      <div
        className={`lg:col-span-7 glass-panel rounded-3xl p-3 sm:p-5 flex flex-col max-h-[600px] overflow-y-auto custom-scrollbar relative transition-all duration-300 ${
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

        <div className="z-10 flex flex-col gap-4">
          {/* Header & Tabs */}
          <div className="shrink-0 space-y-4">
            <div className="flex justify-between items-center border-b border-zinc-900 pb-3">
              <h3 className="text-md font-bold text-zinc-100 font-serif flex items-center gap-1.5">
                <Flame className={`w-5 h-5 text-rose-500 ${isRefining ? "animate-bounce" : ""}`} />
                Lò Luyện Cổ
              </h3>
              <div className="flex items-center gap-3">
                {selectedRecipe?.completion_rate !== undefined && selectedRecipe.completion_rate < 100.0 && (
                  <span className="text-xs text-rose-400 font-mono bg-rose-950/30 border border-rose-900/30 px-2 py-0.5 rounded">
                    Tàn Phương ({selectedRecipe.completion_rate}%)
                  </span>
                )}
                <span className="text-xs text-zinc-500 font-mono">
                  {selectedRecipe?.success_rate ? `Tỷ lệ: ${selectedRecipe.success_rate * 100}%` : ""}
                </span>
              </div>
            </div>

            {/* Player Mental (Niệm Lực) & Quick Use Bar */}
            <div className="flex items-center justify-between bg-zinc-950/70 border border-violet-900/40 px-3.5 py-2 rounded-xl text-xs">
              <div className="flex items-center gap-2 font-mono">
                <Brain className="w-4 h-4 text-violet-400" />
                <span className="text-zinc-400 font-serif">Niệm Lực:</span>
                <span className={`font-bold ${(player.current_thought ?? 0) < 15 ? "text-rose-400" : "text-violet-300"}`}>
                  {(player.current_thought ?? 0).toFixed(0)} / {player.max_thought ?? 100}
                </span>
              </div>
              <button
                type="button"
                onClick={handleQuickUseRestoreMental}
                disabled={restoreMentalGus.length === 0 || isUsingRestoreGu || actionLoading}
                className={`px-2.5 py-1 rounded-lg text-xs font-bold font-serif transition-all flex items-center gap-1.5 cursor-pointer ${
                  restoreMentalGus.length > 0 && !isUsingRestoreGu && !actionLoading
                    ? "bg-gradient-to-r from-violet-600 to-indigo-600 hover:from-violet-500 hover:to-indigo-500 text-zinc-100 shadow-sm active:scale-95"
                    : "bg-zinc-900 border border-zinc-800 text-zinc-600 cursor-not-allowed"
                }`}
                title={
                  restoreMentalGus.length > 0
                    ? `Sử dụng ${guSpecFor(restoreMentalGus[0].gu_id, restoreMentalGus[0].rank)?.name || restoreMentalGus[0].gu_id} để hồi Niệm Lực`
                    : "Không có Cổ trùng hồi Niệm Lực trong túi đồ"
                }
              >
                {isUsingRestoreGu ? (
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                ) : (
                  <Brain className="w-3.5 h-3.5 text-violet-300" />
                )}
                <span>
                  Dùng Cổ Hồi Niệm Lực {restoreMentalGus.length > 0 ? `(${restoreMentalGus.length})` : "(0)"}
                </span>
              </button>
            </div>

            {/* Recipe Selection Combobox */}
            <div className="relative" ref={dropdownRef}>
              <div
                className={`flex items-center justify-between w-full p-3.5 rounded-xl border transition-all cursor-pointer ${
                  isRefining ? "bg-zinc-950/20 border-zinc-800 opacity-50 cursor-not-allowed" : "bg-zinc-950/60 border-zinc-800 hover:border-amber-500/50 hover:shadow-[0_0_10px_rgba(245,158,11,0.1)]"
                }`}
                onClick={() => !isRefining && setIsDropdownOpen(!isDropdownOpen)}
              >
                <div className="flex items-center gap-3">
                  <span className="text-amber-400 font-bold font-serif text-sm">
                    {selectedRecipe ? (() => {
                      const spec = guSpecFor(selectedRecipe.target_gu_id, selectedRecipe.target_rank) || guDictionary[selectedRecipe.target_gu_id];
                      const name = spec?.name || selectedRecipe.target_gu_id;
                      const rank = selectedRecipe.target_rank || spec?.rank || 1;
                      return `${name} (Bậc ${rank})`;
                    })() : "Chọn công thức hợp luyện..."}
                  </span>
                </div>
                <ChevronDown className={`w-4 h-4 text-zinc-500 transition-transform duration-300 ${isDropdownOpen ? "rotate-180" : ""}`} />
              </div>

              {isDropdownOpen && !isRefining && (
                <div className="absolute top-full left-0 right-0 mt-2 bg-zinc-950 border border-zinc-800 rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-72 z-50 animate-in fade-in slide-in-from-top-2 duration-200">
                  <div className="p-3 border-b border-zinc-800/80 flex items-center gap-2 sticky top-0 bg-zinc-950/95 backdrop-blur-sm z-10">
                    <Search className="w-4 h-4 text-amber-500/70" />
                    <input
                      type="text"
                      placeholder="Tìm kiếm tên Cổ trùng..."
                      className="w-full bg-transparent border-none text-sm text-zinc-200 focus:outline-none focus:ring-0 placeholder:text-zinc-600 font-serif"
                      value={searchKeyword}
                      onChange={(e) => setSearchKeyword(e.target.value)}
                      onClick={(e) => e.stopPropagation()}
                      autoFocus
                    />
                  </div>
                  <div className="flex border-b border-zinc-900 bg-zinc-950 text-xs px-2 py-1 gap-2 shrink-0">
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); setStatusTab("all"); }}
                      className={`px-2 py-1 rounded transition-all cursor-pointer font-serif ${statusTab === "all" ? "bg-amber-500/10 text-amber-400 font-bold" : "text-zinc-500 hover:text-zinc-300"}`}
                    >
                      Tất cả
                    </button>
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); setStatusTab("complete"); }}
                      className={`px-2 py-1 rounded transition-all cursor-pointer font-serif ${statusTab === "complete" ? "bg-amber-500/10 text-amber-400 font-bold" : "text-zinc-500 hover:text-zinc-300"}`}
                    >
                      Hoàn chỉnh
                    </button>
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); setStatusTab("incomplete"); }}
                      className={`px-2 py-1 rounded transition-all cursor-pointer font-serif ${statusTab === "incomplete" ? "bg-amber-500/10 text-amber-400 font-bold" : "text-zinc-500 hover:text-zinc-300"}`}
                    >
                      Tàn phương
                    </button>
                  </div>
                  <div className="overflow-y-auto custom-scrollbar flex-1 p-1.5">
                    {filteredRecipes.length === 0 ? (
                      <div className="p-4 text-center text-xs text-zinc-500 italic">
                        Không tìm thấy công thức phù hợp
                      </div>
                    ) : (
                      filteredRecipes.map((recipe) => {
                        const targetGu = guSpecFor(recipe.target_gu_id, recipe.target_rank) || { name: recipe.target_gu_id, rank: recipe.target_rank };
                        const isActive = recipe.id === selectedRecipeId;
                        return (
                          <div
                            key={recipe.id}
                            onClick={() => {
                              setSelectedRecipeId(recipe.id);
                              setIsDropdownOpen(false);
                            }}
                            className={`p-3 rounded-lg text-sm cursor-pointer transition-all flex justify-between items-center ${
                              isActive
                                ? "bg-amber-500/15 text-amber-400 font-bold border border-amber-500/20"
                                : "text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200 border border-transparent"
                            }`}
                          >
                            <span className="font-serif">{targetGu.name} (Bậc {targetGu.rank || 1})</span>
                            <div className="flex items-center gap-2">
                              {recipe.completion_rate !== undefined && recipe.completion_rate < 100.0 && (
                                <span className="text-[9px] text-rose-400 bg-rose-950/20 border border-rose-900/20 px-1.5 py-0.5 rounded font-mono">
                                  Tàn Phương ({recipe.completion_rate}%)
                                </span>
                              )}
                              <span className="text-[10px] font-mono opacity-50">
                                {recipe.success_rate ? `${recipe.success_rate * 100}%` : ""}
                              </span>
                            </div>
                          </div>
                        );
                      })
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Central Furnace Slots Area */}
          <div className="flex flex-col justify-center items-center py-4 relative z-10">
            {selectedRecipe ? (
              <div className="w-full space-y-6">
                {/* Target Gu Header */}
                <div className="text-center space-y-1">
                  <span className="text-[10px] text-amber-500 uppercase tracking-widest font-mono font-bold block">
                    Sản Vật Hợp Thành
                  </span>
                  <h4 className="text-lg font-bold text-zinc-100 font-serif">
                    {(() => {
                      const spec = guSpecFor(selectedRecipe.target_gu_id, selectedRecipe.target_rank) || guDictionary[selectedRecipe.target_gu_id];
                      const name = spec?.name || selectedRecipe.target_gu_id;
                      const rank = selectedRecipe.target_rank || spec?.rank || 1;
                      return `${name} (Bậc ${rank})`;
                    })()}
                  </h4>
                  <p className="text-xs text-zinc-400 max-w-md mx-auto line-clamp-1 italic leading-relaxed">
                    {guSpecFor(selectedRecipe.target_gu_id, selectedRecipe.target_rank)?.description || ""}
                  </p>
                </div>

                {/* Slots Crucible Row */}
                <div className="flex flex-wrap justify-center items-center gap-3 py-2">
                  {requiredSlots.map((slot) => {
                    const reqGuSpec: { name: string; rank?: number } = guSpecFor(slot.guId) || { name: slot.guId };
                    const slottedInstanceId = slottedGus[slot.index];
                    const slottedGu = player.gu_inventory.find(
                      (g) => g.instance_id === slottedInstanceId
                    );
                    const isMaskedSlot = maskedGus.has(slot.guId);

                    return (
                      <div
                        key={slot.index}
                        onDragOver={handleDragOver}
                        onDrop={(e) => handleDrop(e, slot.index)}
                        onClick={() => handleSlotClick(slot)}
                        className={`w-32 h-36 rounded-2xl border transition-all flex flex-col items-center justify-center p-3 relative cursor-pointer group ${
                          slottedGu
                            ? "bg-zinc-950/90 border-amber-500/40 shadow-lg shadow-amber-900/5 hover:border-rose-500/50"
                            : "bg-zinc-950/40 border-dashed border-zinc-800 hover:border-zinc-700"
                        }`}
                      >
                        {slottedGu ? (
                          <div className="text-center space-y-1.5 w-full">
                            <span className="text-[9px] font-mono border border-amber-500/20 px-1 py-0.2 rounded text-amber-400 uppercase inline-block mx-auto">
                              {isMaskedSlot ? "Bậc ???" : `Bậc ${slottedGu.rank}`}
                            </span>
                            <div className="text-xs font-bold text-zinc-200 truncate font-serif">
                              {isMaskedSlot ? "Cổ Trùng Ẩn (Đã đặt)" : `${guSpecFor(slottedGu.gu_id, slottedGu.rank)?.name || slottedGu.gu_id} (Bậc ${slottedGu.rank})`}
                            </div>
                            <div className="text-[9px] text-zinc-505">
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
                              <span className="text-amber-500/80 font-bold">
                                {isMaskedSlot ? "Cổ Trùng Ẩn" : `${reqGuSpec.name}${slot.rank > 0 ? ` (Bậc ${slot.rank})` : ""}`}
                              </span>
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
          <div className="space-y-4 pt-4 border-t border-zinc-900/60 z-10">
            {selectedRecipe && (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {/* Requirements check list */}
                <div className="space-y-2">
                  <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono block">
                    Phụ Gia Vật Phẩm
                  </span>

                  {Object.keys(selectedRecipe.required_materials).length === 0 ? (
                    <div className="text-[11px] text-zinc-500 italic">Không yêu cầu phụ phẩm.</div>
                  ) : (
                    <div className="space-y-1.5">
                      {Object.entries(selectedRecipe.required_materials).map(([matId, qty]) => {
                        const playerQty = player.materials_inventory?.[matId] || 0;
                        const hasEnough = playerQty >= qty;
                        const matDetails = getMaterialDetails(matId);
                        const isMasked = maskedMats.has(matId);

                        return (
                          <div
                            key={matId}
                            className="flex items-center justify-between text-xs bg-zinc-950/50 border border-zinc-900/60 px-3 py-1.5 rounded-lg"
                          >
                            <div className="flex items-center gap-1.5">
                              {isMasked ? (
                                <span className="text-zinc-500 font-mono italic">??? (Phụ phẩm ẩn)</span>
                              ) : (
                                <span className={matDetails.colorClass}>{matDetails.name}</span>
                              )}
                            </div>
                            <div className="flex items-center gap-1.5 font-mono">
                              <span className={isMasked ? "text-zinc-500" : (hasEnough ? "text-zinc-300" : "text-rose-400 font-bold")}>
                                {isMasked ? `?/${qty}` : `${playerQty}/${qty}`}
                              </span>
                              {!isMasked && (hasEnough ? (
                                <Check className="w-3.5 h-3.5 text-emerald-400" />
                              ) : (
                                <X className="w-3.5 h-3.5 text-rose-500" />
                              ))}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>

                {/* Stones cost and Execution button */}
                <div className="flex flex-col justify-end space-y-3">
                  {selectedRecipe.completion_rate !== undefined && selectedRecipe.completion_rate < 100.0 ? (
                    // Incomplete (Tàn Phương) actions
                    <div className="space-y-3">
                      {/* Deduction section */}
                      <div className="border border-zinc-800/80 bg-zinc-950/40 p-3 rounded-xl space-y-2">
                        <div className="flex justify-between items-center text-xs">
                          <span className="text-zinc-400 font-serif font-bold">Thôi Diễn Công Thức (Trí Đạo)</span>
                        </div>
                        <div className="flex justify-between items-center text-[10px] font-mono text-zinc-500">
                          <span>Phí: 15 Niệm Lực & 10 Nguyên Thạch</span>
                        </div>
                        <div className="flex items-center justify-between text-[10px] font-mono">
                          <span className="text-zinc-500">Niệm Lực hiện có:</span>
                            <span className={(player.current_thought ?? 0) >= 15 ? "text-violet-300" : "text-rose-400 font-bold"}>
                              {(player.current_thought ?? 0).toFixed(1)} / {player.max_thought ?? 100}
                            </span>
                        </div>
                        <button
                          type="button"
                          onClick={handleDeduce}
                          disabled={!canDeduce}
                          className={`w-full py-2 px-3 rounded-lg text-xs font-bold font-serif transition-all flex items-center justify-center gap-1.5 cursor-pointer ${
                            canDeduce
                              ? "bg-gradient-to-r from-violet-600 to-indigo-600 hover:from-violet-500 hover:to-indigo-500 text-zinc-100 shadow-md active:scale-[0.98]"
                              : "bg-zinc-900 border border-zinc-800 text-zinc-650 cursor-not-allowed"
                          }`}
                        >
                          {isDeducing ? (
                            <>
                              <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                              <span>Đang thôi diễn...</span>
                            </>
                          ) : (
                            <>
                              <Sparkles className="w-3.5 h-3.5 text-violet-300 animate-pulse" />
                              <span>THÔI DIỄN CỔ PHƯƠNG</span>
                            </>
                          )}
                        </button>
                      </div>

                      {/* Forced Refine section */}
                      <div className="space-y-1">
                        <div className="text-[9px] text-rose-400 font-mono text-center bg-rose-950/20 border border-rose-900/20 py-1.5 px-2 rounded-lg leading-snug">
                          ⚠️ Ép Luyện có phản phệ cực lớn nếu thất bại (Sát thương HP, trừ Max Mana)!
                        </div>
                        <button
                          type="button"
                          onClick={handleRefine}
                          disabled={!canRefine}
                          className={`w-full py-2.5 px-4 rounded-xl text-xs font-bold tracking-wide font-serif transition-all flex items-center justify-center gap-2 cursor-pointer ${
                            canRefine
                              ? "bg-gradient-to-r from-rose-600 to-red-650 hover:from-rose-500 hover:to-red-600 text-zinc-100 shadow-md active:scale-[0.98]"
                              : "bg-zinc-900 border border-zinc-800 text-zinc-650 cursor-not-allowed"
                          }`}
                        >
                          {isRefining ? (
                            <>
                              <RefreshCw className="w-4 h-4 animate-spin text-zinc-100" />
                              <span>{furnaceStage}</span>
                            </>
                          ) : (
                            <>
                              <Flame className="w-4 h-4 text-rose-300 animate-pulse" />
                              <span>ÉP LUYỆN (RỦI RO CAO)</span>
                            </>
                          )}
                        </button>
                      </div>
                    </div>
                  ) : (
                    // Complete (100% or default) actions (Original block)
                    <>
                      <div className="flex justify-between items-center text-xs px-1">
                        <span className="text-zinc-500 uppercase tracking-widest font-mono text-[10px] flex items-center gap-1">
                          <Coins className="w-3.5 h-3.5 text-amber-500" />
                          Nguyên Thạch tiêu hao:
                        </span>
                        <span className={`font-bold font-mono ${hasEnoughStones ? "text-amber-400" : "text-rose-400 font-bold text-sm animate-pulse"}`}>
                          {player.primeval_stones}/{selectedRecipe.primeval_stones_cost} viên
                        </span>
                      </div>

                      {selectedRecipe.immortal_essence_stones_cost > 0 && (
                        <div className="flex justify-between items-center text-xs px-1">
                          <span className="text-zinc-500 uppercase tracking-widest font-mono text-[10px] flex items-center gap-1">
                            <Sparkles className="w-3.5 h-3.5 text-violet-400" />
                            Tiên Tinh Thạch tiêu hao:
                          </span>
                          <span className={`font-bold font-mono ${hasEnoughImmortalStones ? "text-violet-300" : "text-rose-400 font-bold text-sm animate-pulse"}`}>
                            {player.immortal_stones.toFixed(1)}/{selectedRecipe.immortal_essence_stones_cost} viên
                          </span>
                        </div>
                      )}

                      <button
                        type="button"
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
                    </>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* 3. FULL SCREEN RESULT MODAL OVERLAY (GACHA SHOWCASE) */}
        {refineResult && (
          <div className="fixed inset-0 z-[100] flex items-center justify-center p-4 bg-zinc-950/90 backdrop-blur-md animate-fade-in overflow-y-auto select-none">
            {/* Flash Effect on start */}
            <div className="absolute inset-0 bg-white opacity-0 pointer-events-none animate-flash z-50" />

            {refineResult.status === "success" ? (
              <div className="relative w-full max-w-md my-auto space-y-5 animate-gold-glow p-6 rounded-3xl border border-amber-500/50 bg-gradient-to-b from-zinc-900 via-zinc-950 to-zinc-950 shadow-[0_0_60px_rgba(245,158,11,0.25)] overflow-hidden text-center">
                <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-amber-500/15 via-transparent to-transparent pointer-events-none" />

                {/* Close/Exit Button */}
                <button
                  onClick={() => {
                    setRefineResult(null);
                    setSlottedGus({});
                  }}
                  className="absolute top-4 right-4 text-zinc-400 hover:text-amber-400 transition-colors p-1.5 rounded-xl bg-zinc-900/80 border border-zinc-800 hover:border-amber-500/40 cursor-pointer z-10"
                  title="Thoát"
                >
                  <X className="w-4 h-4" />
                </button>

                {/* Celebration header */}
                <div className="space-y-2 pt-2">
                  <div className="relative w-16 h-16 rounded-2xl bg-gradient-to-br from-amber-400/20 to-orange-500/20 border border-amber-400/40 flex items-center justify-center mx-auto shadow-[0_0_25px_rgba(245,158,11,0.35)]">
                    <Sparkles className="w-9 h-9 text-amber-400 animate-pulse" />
                    <div className="absolute -inset-1 rounded-2xl bg-amber-400/10 blur-sm animate-ping -z-10" />
                  </div>
                  <div>
                    <h3 className="text-2xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-amber-300 via-yellow-400 to-amber-500 font-serif tracking-widest uppercase">
                      HỢP LUYỆN THÀNH CÔNG!
                    </h3>
                    <span className="text-[11px] text-amber-400/80 uppercase tracking-widest font-mono font-medium block mt-1">
                      ✨ Khai Sinh Tân Cổ Trùng • Đạo Pháp Thông Thiên ✨
                    </span>
                  </div>
                </div>

                {/* Show created Gu bug specs */}
                {refineResult.new_gu && (() => {
                  const spec = guSpecFor(refineResult.new_gu.gu_id, refineResult.new_gu.rank);
                  const guName = spec?.name || refineResult.new_gu.gu_id;
                  const guType = spec?.gu_type || "Thượng Cổ";
                  const guDesc = spec?.description || "";

                  return (
                    <div className="p-4 rounded-2xl bg-zinc-900/90 border border-amber-500/30 space-y-3 text-left shadow-inner relative overflow-hidden">
                      <div className="absolute -right-6 -bottom-6 w-24 h-24 bg-amber-500/5 rounded-full blur-2xl pointer-events-none" />

                      <div className="flex justify-between items-center border-b border-zinc-800 pb-2.5">
                        <div>
                          <div className="text-[10px] text-amber-400 font-mono uppercase tracking-wider font-semibold">
                            Cổ Trùng Thu Đắc
                          </div>
                          <h4 className="text-base font-bold text-zinc-100 font-serif">
                            {guName}
                          </h4>
                        </div>
                        <div className="flex flex-col items-end gap-1">
                          <span className="text-[10px] font-mono border border-amber-400/40 bg-amber-950/60 px-2 py-0.5 rounded-full text-amber-300 font-bold uppercase shadow-sm">
                            Bậc {refineResult.new_gu.rank}
                          </span>
                          <span className="text-[9px] text-zinc-400 font-mono">
                            Loại: <span className="text-zinc-200">{guType}</span>
                          </span>
                        </div>
                      </div>

                      {guDesc && (
                        <div className="space-y-1.5">
                          <p className={`text-xs text-zinc-300 leading-relaxed italic bg-zinc-950/60 p-3 rounded-xl border border-zinc-800/80 ${!showFullDesc ? "line-clamp-3" : ""}`}>
                            "{guDesc}"
                          </p>
                          {guDesc.length > 80 && (
                            <button
                              onClick={() => setShowFullDesc(!showFullDesc)}
                              className="text-[10px] text-amber-400 hover:text-amber-300 block mx-auto font-mono cursor-pointer select-none border border-amber-500/30 px-2.5 py-0.5 rounded-lg bg-zinc-900/60"
                            >
                              {showFullDesc ? "Rút gọn mô tả ▲" : "Xem đầy đủ mô tả ▼"}
                            </button>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })()}

                {/* Message / Gains Banner */}
                {(() => {
                  const parsed = parseRefineMessage(refineResult.message, true);
                  return (
                    <div className="space-y-2">
                      {parsed.gainsText && (
                        <div className="bg-emerald-950/40 border border-emerald-500/30 p-3 rounded-2xl flex items-center gap-2.5 text-xs text-emerald-300 font-medium justify-center shadow-sm">
                          <TrendingUp className="w-4 h-4 text-emerald-400 shrink-0" />
                          <span>{parsed.gainsText}</span>
                        </div>
                      )}
                    </div>
                  );
                })()}

                {renderRateBreakdown(refineResult.rate_breakdown)}

                <button
                  onClick={() => {
                    setRefineResult(null);
                    setSlottedGus({});
                  }}
                  className="w-full py-3 px-5 rounded-2xl text-xs font-bold bg-gradient-to-r from-amber-400 via-amber-300 to-yellow-400 hover:from-amber-300 hover:to-yellow-300 text-zinc-950 transition-all font-serif cursor-pointer tracking-wider shadow-lg shadow-amber-500/20 active:scale-[0.98] uppercase flex items-center justify-center gap-2"
                >
                  <Award className="w-4 h-4 text-zinc-950" />
                  <span>XÁC NHẬN THU HỒI</span>
                </button>
              </div>
            ) : (
              /* FAILURE / BACKFIRE / FAILSAFE MODAL */
              (() => {
                const parsed = parseRefineMessage(refineResult.message, false);

                let headerColor = "border-rose-500/30 bg-gradient-to-b from-zinc-900 via-zinc-950 to-zinc-950 shadow-[0_0_40px_rgba(244,63,94,0.15)]";
                let iconBg = "bg-rose-500/15 border border-rose-500/30 text-rose-400";
                let mainTitle = "HỢP LUYỆN THẤT BẠI!";
                let mainSubtitle = "Khói đen nghi ngút, hỏa lò tro tàn";
                let IconComponent = AlertCircle;

                if (parsed.isBackfire) {
                  headerColor = "border-rose-500/60 bg-gradient-to-b from-rose-950/40 via-zinc-950 to-zinc-950 shadow-[0_0_50px_rgba(244,63,94,0.3)] animate-pulse";
                  iconBg = "bg-rose-600/20 border border-rose-500/50 text-rose-500 shadow-[0_0_15px_rgba(244,63,94,0.4)]";
                  mainTitle = "HỎA LÒ CHÁY NỔ - PHẢN PHỆ!";
                  mainSubtitle = "Kinh mạch chấn động, thần hồn suy giảm";
                  IconComponent = Flame;
                } else if (parsed.isFailsafe) {
                  headerColor = "border-emerald-500/40 bg-gradient-to-b from-emerald-950/30 via-zinc-950 to-zinc-950 shadow-[0_0_35px_rgba(16,185,129,0.15)]";
                  iconBg = "bg-emerald-500/15 border border-emerald-500/30 text-emerald-400";
                  mainTitle = "BẢO TOÀN NGUYÊN LIỆU!";
                  mainSubtitle = "Cảnh giới thâm hậu, giảm thiểu tổn thất";
                  IconComponent = ShieldCheck;
                }

                return (
                  <div className={`relative w-full max-w-md my-auto space-y-4 p-6 rounded-3xl border ${headerColor} overflow-hidden text-center`}>
                    {/* Close/Exit Button */}
                    <button
                      onClick={() => {
                        setRefineResult(null);
                        setSlottedGus({});
                      }}
                      className="absolute top-4 right-4 text-zinc-500 hover:text-zinc-200 transition-colors p-1.5 rounded-xl bg-zinc-900/80 border border-zinc-800 cursor-pointer z-10"
                      title="Thoát"
                    >
                      <X className="w-4 h-4" />
                    </button>

                    {/* Header */}
                    <div className="space-y-1.5 pt-1">
                      <div className={`w-14 h-14 rounded-2xl ${iconBg} flex items-center justify-center mx-auto mb-1`}>
                        <IconComponent className="w-8 h-8" />
                      </div>
                      <h3 className={`text-xl font-extrabold font-serif tracking-widest uppercase ${parsed.isBackfire ? "text-rose-400" : parsed.isFailsafe ? "text-emerald-400" : "text-rose-500"}`}>
                        {mainTitle}
                      </h3>
                      <span className="text-[10px] text-zinc-400 uppercase tracking-widest font-mono block">
                        {mainSubtitle}
                      </span>
                    </div>

                    {/* Primary Story Text */}
                    <div className="p-3.5 rounded-2xl bg-zinc-900/80 border border-zinc-800/80 text-xs text-zinc-300 leading-relaxed italic shadow-inner">
                      "{parsed.mainText}"
                    </div>

                    {/* Detailed Badges for Backfire, Vital Gu, Status Debuffs */}
                    <div className="space-y-2 text-left text-xs">
                      {parsed.backfireText && (
                        <div className="bg-rose-950/50 border border-rose-800/60 p-3 rounded-2xl flex items-start gap-3 shadow-sm">
                          <Flame className="w-4 h-4 text-rose-400 shrink-0 mt-0.5 animate-pulse" />
                          <div className="space-y-0.5">
                            <span className="text-[10px] uppercase font-mono font-bold text-rose-300 block">
                              Tổn Thất Phản Phệ
                            </span>
                            <p className="text-xs text-rose-200/90 font-sans leading-snug">
                              {parsed.backfireText}
                            </p>
                          </div>
                        </div>
                      )}

                      {parsed.statusDebuffText && (
                        <div className="bg-amber-950/40 border border-amber-800/50 p-3 rounded-2xl flex items-start gap-3 shadow-sm">
                          <Clock className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                          <div className="space-y-0.5">
                            <span className="text-[10px] uppercase font-mono font-bold text-amber-300 block">
                              Hiệu Ứng Mệt Mỏi / Suy Nhược
                            </span>
                            <p className="text-xs text-amber-200/90 font-sans leading-snug">
                              {parsed.statusDebuffText}
                            </p>
                          </div>
                        </div>
                      )}

                      {parsed.vitalSurvivedText && (
                        <div className="bg-emerald-950/40 border border-emerald-800/50 p-3 rounded-2xl flex items-start gap-3 shadow-sm">
                          <ShieldCheck className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                          <div className="space-y-0.5">
                            <span className="text-[10px] uppercase font-mono font-bold text-emerald-300 block">
                              Bảo Hộ Bản Mệnh
                            </span>
                            <p className="text-xs text-emerald-200/90 font-sans leading-snug">
                              {parsed.vitalSurvivedText}
                            </p>
                          </div>
                        </div>
                      )}
                    </div>

                    {renderRateBreakdown(refineResult.rate_breakdown)}

                    <button
                      onClick={() => {
                        setRefineResult(null);
                        setSlottedGus({});
                      }}
                      className="w-full py-3 px-4 rounded-2xl text-xs font-bold bg-zinc-900 hover:bg-zinc-800 text-zinc-200 transition-all font-serif cursor-pointer tracking-wider border border-zinc-700/80 hover:border-zinc-500 shadow-md active:scale-[0.98] uppercase"
                    >
                      RÚT KINH NGHIỆM
                    </button>
                  </div>
                );
              })()
            )}
          </div>
        )}
      </div>
    </GamePanel>
  );
};
