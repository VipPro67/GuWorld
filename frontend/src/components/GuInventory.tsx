"use client";

import React, { useEffect, useState, useMemo } from "react";
import Link from "next/link";
import { Player, GuSpec, ItemSpec, GuInstance, useGameStore, guSpecFor } from "@/store/useGameStore";
import { Flower2, Wine, Gem, ShieldAlert, Sparkles, HelpCircle, Search, Shield, Zap, ChevronDown, ChevronUp, Lock, Unlock, BookOpen, Copy, Check } from "lucide-react";
import { translateGuType, translateItemType, translatePath } from "@/lib/translate";
import { getPathColor as _getPathColor } from "@/lib/utils";
import { GamePanel } from "@/components/ui";
// UI migrated to shared components

interface GuInventoryProps {
  player: Player;
  onFeed: (guInstanceId: string) => Promise<void>;
  onFeedGroup?: (guId: string, rank?: number) => Promise<void>;
  onFeedAll?: () => Promise<void>;
  onUnequipAll?: () => Promise<void>;
  actionLoading: boolean;
  guDictionary: Record<string, GuSpec>;
  itemDictionary: Record<string, ItemSpec>;
  onEquipToggle?: (instanceId: string, equip: boolean) => Promise<void>;
  onUseItem?: (itemId: string) => Promise<void>;
  onUpdateStatus?: (instanceId: string, status: string) => Promise<void>;
  onSetVital?: (instanceId: string) => Promise<void>;
  onUseGu?: (instanceId: string) => Promise<void>;
  onDropGu?: (instanceId: string) => Promise<boolean>;
  onDropGus?: (instanceIds: string[]) => Promise<boolean>;
  onUseItems?: (itemIds: string[]) => Promise<boolean>;
  onUseGus?: (instanceIds: string[]) => Promise<boolean>;
}

const getPathColor = (path: string): string => {
  const c = _getPathColor(path);
  return `${c.bg} ${c.text} ${c.border}`;
};

const FormatKey = (key: string): string => {
  switch (key) {
    case "damage":
    case "add_base_attack":
      return "Sát Thương";
    case "defense":
    case "add_base_defense":
      return "Phòng Ngự";
    case "refine_mana_tier":
      return "Tinh Luyện Chân Nguyên";
    case "cleansing_power":
      return "Thanh Tẩy";
    default:
      return key.replace(/_/g, " ").replace(/\b\w/g, (c: string) => c.toUpperCase());
  }
};

export const GuInventory: React.FC<GuInventoryProps> = ({
  player,
  onFeed,
  onFeedGroup,
  onFeedAll,
  onUnequipAll,
  actionLoading,
  guDictionary,
  itemDictionary,
  onEquipToggle,
  onUseItem,
  onUseItems,
  onUpdateStatus,
  onSetVital,
  onUseGu,
  onUseGus,
  onDropGu,
  onDropGus,
}) => {
  const [selectedGuIds, setSelectedGuIds] = useState<Record<string, boolean>>({});
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [isBatchMode, setIsBatchMode] = useState<boolean>(false);
  // Single-item quantity use mode (Items tab)
  const [useMultipleItemId, setUseMultipleItemId] = useState<string | null>(null);
  const [useMultipleItemQty, setUseMultipleItemQty] = useState<number>(1);
  // Single-group quantity use mode (Gu tab)
  const [useMultipleGuGroupKey, setUseMultipleGuGroupKey] = useState<string | null>(null);
  const [useMultipleGuQty, setUseMultipleGuQty] = useState<number>(1);

  const [feedingId, setFeedingId] = useState<string | null>(null);
  const [feedingGroupGuId, setFeedingGroupGuId] = useState<string | null>(null);
  const [feedingAll, setFeedingAll] = useState<boolean>(false);
  const [unequippingAll, setUnequippingAll] = useState<boolean>(false);
  const [expandedGroups, setExpandedGroups] = useState<Record<string, boolean>>({});
  const [equippingId, setEquippingId] = useState<string | null>(null);
  const [usingId, setUsingId] = useState<string | null>(null);
  const [statusUpdatingId, setStatusUpdatingId] = useState<string | null>(null);
  const [usingGuId, setUsingGuId] = useState<string | null>(null);
  const [settingVitalId, setSettingVitalId] = useState<string | null>(null);
  const [droppingId, setDroppingId] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState<Record<string, boolean>>({});
  const [itemDetailOpen, setItemDetailOpen] = useState<Record<string, boolean>>({});

  const [searchTerm, setSearchTerm] = useState("");
  const [selectedTab, setSelectedTab] = useState<"gu" | "material">("gu");
  const [guSubTab, setGuSubTab] = useState<"equipped" | "rest">("equipped");
  const [selectedPath, setSelectedPath] = useState<string>("all");
  const [selectedRank, setSelectedRank] = useState<string>("all");

  // Reset batch selection when switching subtab
  const handleSubTabChange = (tab: "equipped" | "rest") => {
    setGuSubTab(tab);
    setIsBatchMode(false);
    setSelectedGuIds({});
  };

  // The player status snapshot only carries equipped/vital Gus; lazy-load the
  // full inventory when the management view is opened.
  const fetchGuInventory = useGameStore((state) => state.fetchGuInventory);
  const guInventoryLoaded = useGameStore((state) => state.guInventoryLoaded);
  useEffect(() => {
    if (!guInventoryLoaded) void fetchGuInventory();
  }, [fetchGuInventory, guInventoryLoaded]);

  const toggleGroupExpand = (guId: string) => {
    setExpandedGroups((prev) => ({ ...prev, [guId]: !prev[guId] }));
  };

  const handleFeedGroupClick = async (guId: string, rank?: number) => {
    if (!onFeedGroup) return;
    const groupKey = rank != null ? `${guId}::${rank}` : guId;
    setFeedingGroupGuId(groupKey);
    try {
      await onFeedGroup(guId, rank);
    } finally {
      setFeedingGroupGuId(null);
    }
  };

  const handleFeedAllClick = async () => {
    setFeedingAll(true);
    try {
      const activeGroups = guSubTab === "equipped" ? equippedGroups : restGroups;
      let fedAny = false;
      for (const group of activeGroups) {
        const hungryCount = group.instances.filter(
          inst => inst.status !== "DEAD" && inst.status !== "SEALED" && inst.hunger < group.spec.max_hunger
        ).length;
        if (hungryCount > 0 && onFeedGroup) {
          await onFeedGroup(group.guId, group.spec.rank);
          fedAny = true;
        }
      }
      if (!fedAny) {
        if (onFeedAll && activeGroups.length === 0) {
          await onFeedAll();
        } else {
          useGameStore.getState().addToast(
            guSubTab === "equipped"
              ? "Không có Cổ trùng đang xuất trận nào cần cho ăn."
              : "Không có Cổ trùng trong túi nào cần cho ăn.",
            "info"
          );
        }
      }
    } finally {
      setFeedingAll(false);
    }
  };

  const handleUnequipAllClick = async () => {
    const unequipFn = onUnequipAll || useGameStore.getState().unequipAllGu;
    setUnequippingAll(true);
    try {
      await unequipFn();
    } finally {
      setUnequippingAll(false);
    }
  };

  const handleStatusUpdate = async (instanceId: string, status: string) => {
    if (!onUpdateStatus) return;
    setStatusUpdatingId(instanceId);
    try {
      await onUpdateStatus(instanceId, status);
    } finally {
      setStatusUpdatingId(null);
    }
  };

  const handleUseGu = async (instanceId: string) => {
    if (!onUseGu) return;
    setUsingGuId(instanceId);
    try {
      await onUseGu(instanceId);
    } finally {
      setUsingGuId(null);
    }
  };

  const handleSetVital = async (instanceId: string, name: string) => {
    if (!onSetVital) return;
    const targetGu = player.gu_inventory?.find((g) => g.instance_id === instanceId);
    if (targetGu && targetGu.rank > player.rank) {
      useGameStore.getState().addToast(
        `Không thể thiết lập Cổ trùng Rank ${targetGu.rank} vượt quá tu vi hiện tại của đạo hữu (Rank ${player.rank}) làm Bản Mệnh Cổ!`,
        "error"
      );
      return;
    }
    const hadVital = player.gu_inventory?.some((g) => g.is_vital) || false;
    const confirmMsg = hadVital
      ? `Cảnh báo: Đạo hữu đang sở hữu một Bản Mệnh Cổ. Thiết lập Cổ trùng "${name}" làm Bản Mệnh Cổ mới sẽ hủy Bản Mệnh cũ và gây phản phệ khiến đạo hữu mất 50 HP!\n\nĐạo hữu chắc chắn muốn tiếp tục?`
      : `Đạo hữu muốn chọn Cổ trùng "${name}" làm Bản Mệnh Cổ chứ?`;

    if (await useGameStore.getState().showConfirm(confirmMsg)) {
      setSettingVitalId(instanceId);
      try {
        await onSetVital(instanceId);
      } finally {
        setSettingVitalId(null);
      }
    }
  };

  const materials = player.materials_inventory || {};
  const guList = player.gu_inventory || [];

  const isConsumableItem = (spec: ItemSpec) => {
    return spec.item_type === "Consumable" || spec.type === "Consumable" || spec.type === "SoulResource";
  };

  const handleFeedClick = async (instanceId: string) => {
    setFeedingId(instanceId);
    try {
      await onFeed(instanceId);
    } finally {
      setFeedingId(null);
    }
  };

  const handleEquipClick = async (instanceId: string, equip: boolean) => {
    if (!onEquipToggle) return;
    setEquippingId(instanceId);
    try {
      await onEquipToggle(instanceId, equip);
    } finally {
      setEquippingId(null);
    }
  };

  const handleUseItemClick = async (itemId: string) => {
    if (!onUseItem) return;
    setUsingId(itemId);
    try {
      await onUseItem(itemId);
    } finally {
      setUsingId(null);
    }
  };

  const handleUseMultipleItem = async () => {
    if (!useMultipleItemId || useMultipleItemQty < 1) return;
    if (useMultipleItemQty > 100) {
      useGameStore.getState().addToast("Chỉ có thể sử dụng tối đa 100 vật phẩm một lúc", "info");
      return;
    }
    if (!onUseItems) return;

    const spec = itemDictionary[useMultipleItemId];
    const itemName = spec?.name || useMultipleItemId;
    const confirmMsg = `Xác nhận sử dụng ${useMultipleItemQty}x ${itemName}?`;
    if (!(await useGameStore.getState().showConfirm(confirmMsg))) return;

    const itemIds = Array(useMultipleItemQty).fill(useMultipleItemId);
    await onUseItems(itemIds);
    setUseMultipleItemId(null);
    setUseMultipleItemQty(1);
  };

  const handleBatchDrop = async () => {
    const selectedIds = Object.keys(selectedGuIds).filter((id) => selectedGuIds[id]);
    if (selectedIds.length === 0) {
      useGameStore.getState().addToast("Đạo hữu chưa chọn Cổ trùng nào để thả.", "info");
      return;
    }
    const confirmMsg = `Xác nhận phóng sinh (thả) ${selectedIds.length} Cổ trùng đã chọn? Hành động này không thể hoàn tác.`;
    if (!(await useGameStore.getState().showConfirm(confirmMsg))) return;

    setDroppingId("batch");
    try {
      if (onDropGus) {
        await onDropGus(selectedIds);
      }
      setSelectedGuIds({});
      setIsBatchMode(false);
    } finally {
      setDroppingId(null);
    }
  };

  const handleUseMultipleGus = async (groupKey: string, instances: GuInstance[]) => {
    if (useMultipleGuQty < 1) return;
    if (!onUseGus) return;

    const eligibleInstances = instances.filter(isEligibleForUse);
    const qty = Math.min(useMultipleGuQty, eligibleInstances.length);
    if (qty === 0) {
      useGameStore.getState().addToast("Không có Cổ trùng nào đủ điều kiện để sử dụng.", "info");
      return;
    }

    const spec = guSpecFor(eligibleInstances[0].gu_id, eligibleInstances[0].rank);
    const guName = spec?.name || eligibleInstances[0].gu_id;
    const confirmMsg = `Xác nhận sử dụng ${qty}x ${guName}?`;
    if (!(await useGameStore.getState().showConfirm(confirmMsg))) return;

    const selectedIds = eligibleInstances.slice(0, qty).map(g => g.instance_id);
    setUsingGuId("batch");
    try {
      await onUseGus(selectedIds);
      setUseMultipleGuGroupKey(null);
      setUseMultipleGuQty(1);
    } finally {
      setUsingGuId(null);
    }
  };

  const isEligibleForDrop = (gu: GuInstance) => {
    return !gu.is_vital && !gu.equipped;
  };

  const isEligibleForUse = (gu: GuInstance) => {
    const spec = guSpecFor(gu.gu_id, gu.rank);
    if (!spec) return false;
    return spec.gu_type === "Consumable" && gu.status !== "DEAD" && gu.status !== "SEALED";
  };

  const isEligibleForCurrentAction = (gu: GuInstance) => {
    return isEligibleForDrop(gu);
  };

  // Filter materials (only those with qty > 0 and matching searchTerm)
  const filteredMaterials = useMemo(() => {
    return Object.keys(itemDictionary)
      .filter((key) => {
        const qty = materials[key] || 0;
        if (qty <= 0) return false;

        const spec = itemDictionary[key];
        if (!spec) return false;
        const matchesSearch =
          spec.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
          spec.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
          (spec.item_type && spec.item_type.toLowerCase().includes(searchTerm.toLowerCase()));
        return matchesSearch;
      })
      .map((key) => ({ key, spec: itemDictionary[key], qty: materials[key] }));
  }, [itemDictionary, materials, searchTerm]);

  const consumableMaterialRows = useMemo(() => {
    return filteredMaterials.filter(({ spec }) => isConsumableItem(spec));
  }, [filteredMaterials]);

  // Dynamic filters calculation
  const allPaths = useMemo(() => {
    return Array.from(
      new Set(
        guList.map((gu) => {
          const spec = guSpecFor(gu.gu_id, gu.rank);
          return spec?.path && spec.path.length > 0 ? spec.path[0] : null;
        }).filter(Boolean)
      )
    ) as string[];
  }, [guList]);

  const allRanks = useMemo(() => {
    return Array.from(
      new Set(
        guList.map((gu) => gu.rank).filter((r): r is number => r !== undefined)
      )
    ).sort((a, b) => a - b);
  }, [guList]);

  // Filter Gu bugs matching searchTerm, path, and rank
  const filteredGus = useMemo(() => {
    return guList.filter((gu) => {
      const spec = guSpecFor(gu.gu_id, gu.rank) || {
        name: gu.gu_id,
        description: "Cổ trùng chưa được giải nghĩa.",
        path: [] as string[],
        gu_type: "",
      };
      const matchesSearch =
        spec.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        spec.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (spec.path && spec.path.some(p => p.toLowerCase().includes(searchTerm.toLowerCase()))) ||
        (spec.gu_type && spec.gu_type.toLowerCase().includes(searchTerm.toLowerCase()));

      const matchesPath = selectedPath === "all" || (spec.path && spec.path.includes(selectedPath));
      const matchesRank = selectedRank === "all" || (gu.rank !== undefined && gu.rank === parseInt(selectedRank, 10));

      return matchesSearch && matchesPath && matchesRank;
    });
  }, [guList, searchTerm, selectedPath, selectedRank]);

  const eligibleGus = useMemo(() => {
    return filteredGus.filter(isEligibleForCurrentAction);
  }, [filteredGus]);

  const selectedEligibleCount = useMemo(() => {
    return eligibleGus.filter(g => selectedGuIds[g.instance_id]).length;
  }, [eligibleGus, selectedGuIds]);

  const isAllEligibleSelected = eligibleGus.length > 0 && selectedEligibleCount === eligibleGus.length;
  // Not used for Gu use mode anymore – only for batch drop

  const guGroups = useMemo(() => {
    const groupsMap: Record<string, GuInstance[]> = {};
    filteredGus.forEach((gu) => {
      const groupKey = `${gu.gu_id}::${gu.rank ?? 0}`;
      if (!groupsMap[groupKey]) {
        groupsMap[groupKey] = [];
      }
      groupsMap[groupKey].push(gu);
    });

    const statusPriority = {
      'ACTIVE': 1,
      'SLEEPING': 2,
      'SEALED': 3,
      'DYING': 4,
      'DEAD': 5
    };

    const groups = Object.entries(groupsMap).map(([groupKey, instances]) => {
      const [guId, rankStr] = groupKey.split("::");
      const groupRank = parseInt(rankStr, 10) || 0;
      const spec = (guSpecFor(guId, groupRank) || {
        id: guId,
        name: guId,
        rank: groupRank || 1,
        path: ["Unknown Path"],
        gu_type: "Phàm Cổ",
        base_mana_cost: 0,
        effects: {},
        hunger_rate: 0,
        food_types: ["unknown"],
        max_hunger: 100,
        description: "Cổ trùng chưa được giải nghĩa.",
      }) as GuSpec;

      const instancesCopy = [...instances];
      instancesCopy.sort((a, b) => {
        if (a.is_vital !== b.is_vital) return a.is_vital ? -1 : 1;
        if (a.equipped !== b.equipped) return a.equipped ? -1 : 1;
        const pA = statusPriority[a.status as keyof typeof statusPriority] || 9;
        const pB = statusPriority[b.status as keyof typeof statusPriority] || 9;
        if (pA !== pB) return pA - pB;
        return b.hunger - a.hunger;
      });

      const hasVital = instancesCopy.some(inst => inst.is_vital);
      const activeCount = instancesCopy.filter(inst => inst.status === "ACTIVE").length;
      const sleepingCount = instancesCopy.filter(inst => inst.status === "SLEEPING").length;
      const sealedCount = instancesCopy.filter(inst => inst.status === "SEALED").length;
      const dyingCount = instancesCopy.filter(inst => inst.status === "DYING").length;
      const deadCount = instancesCopy.filter(inst => inst.status === "DEAD").length;
      const equippedCount = instancesCopy.filter(inst => inst.equipped).length;

      return {
        guId,
        groupKey,
        groupRank,
        spec,
        instances: instancesCopy,
        hasVital,
        activeCount,
        sleepingCount,
        sealedCount,
        dyingCount,
        deadCount,
        equippedCount,
      };
    });

    groups.sort((a, b) => {
      if (a.hasVital !== b.hasVital) return a.hasVital ? -1 : 1;
      const aEquipped = a.equippedCount > 0;
      const bEquipped = b.equippedCount > 0;
      if (aEquipped !== bEquipped) return aEquipped ? -1 : 1;
      if (b.spec.rank !== a.spec.rank) return b.spec.rank - a.spec.rank;
      return a.spec.name.localeCompare(b.spec.name);
    });

    return groups;
  }, [filteredGus]);

  const { equippedGroups, restGroups, equippedBugCount, restBugCount } = useMemo(() => {
    const equipped = guGroups.filter(g => g.equippedCount > 0);
    const rest = guGroups.filter(g => g.equippedCount === 0);
    const eqCount = equipped.reduce((sum, g) => sum + g.equippedCount, 0);
    const rCount = rest.reduce((sum, g) => sum + g.instances.length, 0);
    return {
      equippedGroups: equipped,
      restGroups: rest,
      equippedBugCount: eqCount,
      restBugCount: rCount,
    };
  }, [guGroups]);

  const handleSelectAllToggle = () => {
    const newSelected = { ...selectedGuIds };
    if (isAllEligibleSelected) {
      eligibleGus.forEach(g => {
        newSelected[g.instance_id] = false;
      });
    } else {
      eligibleGus.forEach(g => {
        newSelected[g.instance_id] = true;
      });
    }
    setSelectedGuIds(newSelected);
  };

  const showMaterials = selectedTab === "material";
  const showGus = selectedTab === "gu";

  return (
    <div className="glass-panel-gold rounded-3xl p-3 sm:p-6 w-full space-y-4 sm:space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
        <div className="flex items-center gap-2">
          <Sparkles className="w-5 h-5 text-amber-400 animate-pulse" />
          <h3 className="text-lg font-bold text-zinc-100 font-serif tracking-wide">
            Bách Bảo Nang
          </h3>
        </div>
      </div>

      {/* Filter and Search controls */}
      <div className="space-y-3">
        <div className="flex gap-2 bg-zinc-950/80 p-1.5 rounded-xl border border-zinc-900">
          <button
            onClick={() => setSelectedTab("gu")}
            className={`flex-1 text-xs py-1.5 rounded-lg font-mono font-medium transition-all ${selectedTab === "gu"
                ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                : "text-zinc-500 hover:text-zinc-300"
              }`}
          >
            Cổ Trùng
          </button>
          <button
            onClick={() => setSelectedTab("material")}
            className={`flex-1 text-xs py-1.5 rounded-lg font-mono font-medium transition-all ${selectedTab === "material"
                ? "bg-amber-500/10 text-amber-400 border border-amber-500/20"
                : "text-zinc-500 hover:text-zinc-300"
              }`}
          >
            Vật Phẩm
          </button>
        </div>

        {/* Search bar */}
        <div className="relative">
          <Search className="w-4 h-4 text-zinc-650 absolute left-3.5 top-3" />
          <input
            type="text"
            placeholder="Tìm kiếm theo tên hoặc mô tả..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full bg-zinc-950/40 border border-zinc-900 focus:border-amber-500/50 outline-none text-xs text-zinc-200 placeholder-zinc-650 pl-10 pr-4 py-2.5 rounded-xl transition-colors font-mono"
          />
        </div>

        {/* Dynamic Filters for Gu bugs */}
        {selectedTab === "gu" && (
          <div className="space-y-3">
            <div className="flex flex-col sm:flex-row gap-3">
              <div className="flex-1 relative">
                <select
                  value={selectedPath}
                  onChange={(e) => setSelectedPath(e.target.value)}
                  className="w-full bg-zinc-950/40 border border-zinc-900 focus:border-amber-500/50 outline-none text-xs text-zinc-300 px-3 py-2 rounded-xl transition-colors font-mono cursor-pointer appearance-none"
                  style={{
                    backgroundImage: `url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='24' height='24' viewBox='0 0 24 24' fill='none' stroke='%23a1a1aa' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><polyline points='6 9 12 15 18 9'></polyline></svg>")`,
                    backgroundRepeat: 'no-repeat',
                    backgroundPosition: 'right 12px center',
                    backgroundSize: '14px'
                  }}
                >
                  <option value="all">Tất cả Lưu Phái</option>
                  {allPaths.map(path => (
                    <option key={path} value={path}>{translatePath(path)}</option>
                  ))}
                </select>
              </div>
              <div className="flex-1 relative">
                <select
                  value={selectedRank}
                  onChange={(e) => setSelectedRank(e.target.value)}
                  className="w-full bg-zinc-950/40 border border-zinc-900 focus:border-amber-500/50 outline-none text-xs text-zinc-300 px-3 py-2 rounded-xl transition-colors font-mono cursor-pointer appearance-none"
                  style={{
                    backgroundImage: `url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='24' height='24' viewBox='0 0 24 24' fill='none' stroke='%23a1a1aa' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><polyline points='6 9 12 15 18 9'></polyline></svg>")`,
                    backgroundRepeat: 'no-repeat',
                    backgroundPosition: 'right 12px center',
                    backgroundSize: '14px'
                  }}
                >
                  <option value="all">Tất cả Bậc (Rank)</option>
                  {allRanks.map(rank => (
                    <option key={rank} value={rank}>Bậc {rank}</option>
                  ))}
                </select>
              </div>
            </div>

            {/* Sub-tab selection: Đang Trang Bị vs Còn Lại */}
            <div className="flex gap-2 bg-zinc-950/60 p-1.5 rounded-xl border border-zinc-900">
              <button
                type="button"
                onClick={() => handleSubTabChange("equipped")}
                className={`flex-1 text-xs py-2 rounded-lg font-mono font-medium transition-all flex items-center justify-center gap-2 cursor-pointer ${
                  guSubTab === "equipped"
                    ? "bg-emerald-950/40 text-emerald-400 border border-emerald-500/30 font-bold shadow-sm"
                    : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/40"
                }`}
              >
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                <span>Đang Trang Bị</span>
                <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-emerald-950/80 text-emerald-300 border border-emerald-800/40 font-mono">
                  {equippedBugCount}
                </span>
              </button>

              <button
                type="button"
                onClick={() => handleSubTabChange("rest")}
                className={`flex-1 text-xs py-2 rounded-lg font-mono font-medium transition-all flex items-center justify-center gap-2 cursor-pointer ${
                  guSubTab === "rest"
                    ? "bg-amber-950/40 text-amber-400 border border-amber-500/30 font-bold shadow-sm"
                    : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/40"
                }`}
              >
                <span className="w-2 h-2 rounded-full bg-amber-500" />
                <span>Rảnh Rỗi</span>
                <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-amber-950/80 text-amber-300 border border-amber-800/40 font-mono">
                  {restBugCount}
                </span>
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Materials Inventory section */}
      {showMaterials && (
        <div className="space-y-2 text-left">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="text-[10px] font-mono text-zinc-505 uppercase tracking-widest block font-bold">
              Nguyên Liệu Sở Hữu
            </span>
          </div>
          {filteredMaterials.length === 0 ? (
            <div className="text-[10px] text-zinc-600 italic">Hộp nguyên liệu trống.</div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3">
              {filteredMaterials.map(({ key, spec, qty }) => {
                const isConsumable = spec.item_type === "Consumable" || spec.type === "Consumable";
                const isUsing = usingId === key;
                const isDetailOpen = !!itemDetailOpen[key];
                const itemRarityLabel = (r?: number) => {
                  if (!r) return "";
                  const labels = ["", "Phàm", "Linh", "Bảo", "Tiên", "Thánh"];
                  return labels[r] || `Cấp ${r}`;
                };
                return (
                  <div
                    key={key}
                    className="flex flex-col bg-zinc-950/70 border border-zinc-900 rounded-xl overflow-hidden"
                  >
                    <div className="flex justify-between items-center px-3.5 py-2.5">
                      <div className="flex items-center gap-2 min-w-0">
                        <div className="text-left min-w-0">
                          <div className="text-xs font-semibold text-zinc-200 truncate">{spec.name}</div>
                          <div className="text-[9px] text-zinc-500 font-mono">x{qty} sở hữu</div>
                        </div>
                      </div>
                      <div className="flex items-center gap-1.5 shrink-0">
                        {isConsumable && onUseItem && useMultipleItemId !== key && (
                          <button
                            onClick={() => handleUseItemClick(key)}
                            disabled={actionLoading || isUsing}
                            className="text-[9px] py-1 px-2 bg-emerald-950/30 text-emerald-400 border border-emerald-900/40 hover:bg-emerald-950/50 hover:border-emerald-800 rounded font-mono transition-all cursor-pointer font-bold"
                          >
                            {isUsing ? "..." : "Sử Dụng"}
                          </button>
                        )}
                        {isConsumable && onUseItems && qty > 1 && useMultipleItemId !== key && (
                          <button
                            type="button"
                            onClick={() => {
                              setUseMultipleItemId(key);
                              setUseMultipleItemQty(1);
                            }}
                            className="text-[9px] py-1 px-2 bg-amber-950/30 text-amber-400 border border-amber-900/40 hover:bg-amber-950/50 hover:border-amber-800 rounded font-mono transition-all cursor-pointer font-bold"
                          >
                            ✨ Nhiều
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() => setItemDetailOpen(prev => ({...prev, [key]: !prev[key]}))}
                          className="text-[9px] py-1 px-1.5 bg-zinc-950/50 text-zinc-400 border border-zinc-800 hover:bg-zinc-900 hover:text-zinc-300 rounded font-mono transition-all cursor-pointer"
                          title="Xem chi tiết"
                        >
                          {isDetailOpen ? "▲" : "▼"}
                        </button>
                      </div>
                    </div>

                    {/* Quantity picker for multi-use */}
                    {useMultipleItemId === key && (
                      <div className="flex items-center gap-2 px-3.5 py-2 border-t border-emerald-900/30 bg-emerald-950/10">
                        <span className="text-[10px] text-emerald-400 font-mono font-bold whitespace-nowrap">Số lượng:</span>
                        <input
                          type="number"
                          min={1}
                          max={Math.min(qty, 100)}
                          value={useMultipleItemQty}
                          onChange={(e) => setUseMultipleItemQty(Math.max(1, Math.min(qty, 100, parseInt(e.target.value) || 1)))}
                          className="w-16 bg-zinc-950 border border-emerald-900/50 text-emerald-300 text-xs font-mono text-center py-1 rounded-lg outline-none focus:border-emerald-500/60"
                        />
                        <span className="text-[9px] text-zinc-500 font-mono">/ {qty}</span>
                        <button
                          type="button"
                          onClick={() => setUseMultipleItemQty(Math.min(qty, 100))}
                          className="text-[9px] py-0.5 px-1.5 bg-zinc-900 text-zinc-400 border border-zinc-800 hover:text-zinc-200 rounded font-mono cursor-pointer"
                        >
                          Tất cả
                        </button>
                        <button
                          type="button"
                          onClick={handleUseMultipleItem}
                          disabled={actionLoading}
                          className="text-[9px] font-bold py-1 px-2.5 bg-emerald-600 text-white hover:bg-emerald-500 rounded font-mono transition-all cursor-pointer shadow-[0_0_8px_rgba(16,185,129,0.2)]"
                        >
                          Xác nhận
                        </button>
                        <button
                          type="button"
                          onClick={() => { setUseMultipleItemId(null); setUseMultipleItemQty(1); }}
                          className="text-[9px] py-1 px-2 bg-zinc-900 text-zinc-400 border border-zinc-800 hover:text-zinc-200 rounded font-mono cursor-pointer"
                        >
                          Hủy
                        </button>
                      </div>
                    )}

                    {isDetailOpen && (
                      <div className="px-3.5 pb-3 pt-0 border-t border-zinc-900/60 grid grid-cols-2 gap-2 text-[10px]">
                        {spec.rank && spec.rank > 0 && (
                          <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                            <span className="text-zinc-600 font-mono block mb-0.5">Cấp</span>
                            <span className="text-zinc-300">{itemRarityLabel(spec.rank) || spec.rank}</span>
                          </div>
                        )}
                        {spec.path && (
                          <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                            <span className="text-zinc-600 font-mono block mb-0.5">Đường</span>
                            <div className="flex flex-wrap gap-1">
                              {Array.isArray(spec.path) ? (
                                spec.path.map(p => (
                                  <span key={p} className={getPathColor(p)}>{translatePath(p)}</span>
                                ))
                              ) : (
                                <span className={getPathColor(spec.path as string)}>{translatePath(spec.path as string)}</span>
                              )}
                            </div>
                          </div>
                        )}
                        {spec.rarity && spec.rarity > 0 && (
                          <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                            <span className="text-zinc-600 font-mono block mb-0.5">Độ hiếm</span>
                            <span className="text-zinc-300">{spec.rarity}</span>
                          </div>
                        )}
                        <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                          <span className="text-zinc-600 font-mono block mb-0.5">Loại</span>
                          <span className="text-zinc-300">{translateItemType(spec.item_type || spec.type)}</span>
                        </div>
                        {spec.effects && Object.keys(spec.effects).length > 0 && (
                          <div className="col-span-2 bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                            <span className="text-zinc-600 font-mono block mb-0.5">Hiệu Ứng</span>
                            <div className="flex flex-wrap items-center gap-1">
                              {Object.entries(spec.effects).map(([k, v]) => (
                                <span key={k} className="bg-zinc-900/60 px-1.5 py-0.5 rounded border border-zinc-800/50 text-zinc-300 whitespace-nowrap">
                                  {FormatKey(k)}: {String(v)}
                                </span>
                              ))}
                            </div>
                          </div>
                        )}
                        {spec.tags && spec.tags.length > 0 && (
                          <div className="col-span-2 bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                            <span className="text-zinc-600 font-mono block mb-0.5">Tags</span>
                            <div className="flex flex-wrap gap-1">
                              {spec.tags.map(tag => (
                                <span key={tag} className="bg-zinc-900/60 px-1.5 py-0.5 rounded border border-zinc-800/50 text-zinc-400 text-[9px]">
                                  {tag}
                                </span>
                              ))}
                            </div>
                          </div>
                        )}
                        {spec.description && (
                          <div className="col-span-2 bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                            <span className="text-zinc-600 font-mono block mb-0.5">Mô Tả</span>
                            <span className="text-zinc-400 italic">{spec.description}</span>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}

      {/* Gu Bug list section */}
      {showGus && (
        <div className="space-y-6 text-left">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-zinc-900 pb-3">
            <span className="text-[10px] font-mono text-zinc-500 uppercase tracking-widest block font-bold">
              Cổ Trùng Sở Hữu ({filteredGus.length} / {guList.length} con)
            </span>

            {(onDropGus || onUseGus) && filteredGus.length > 0 && (
              <div className="flex items-center gap-2">
                {isBatchMode ? (
                  <>
                    <label className="flex items-center gap-1.5 text-xs text-zinc-400 cursor-pointer font-mono select-none">
                      <input
                        type="checkbox"
                        checked={isAllEligibleSelected}
                        onChange={handleSelectAllToggle}
                        className="rounded border-zinc-850 bg-zinc-950 text-amber-550 focus:ring-amber-500/50"
                      />
                      Chọn tất cả ({eligibleGus.length})
                    </label>

                    <button
                      type="button"
                      disabled={selectedEligibleCount === 0 || droppingId === "batch"}
                      onClick={handleBatchDrop}
                      className={`text-[10px] font-bold py-1 px-3 rounded-lg font-mono transition-all ${
                        selectedEligibleCount > 0
                          ? "bg-rose-600 text-white hover:bg-rose-500 cursor-pointer shadow-[0_0_8px_rgba(239,68,68,0.2)]"
                          : "bg-zinc-900 text-zinc-600 border border-zinc-850 cursor-not-allowed"
                      }`}
                    >
                      {droppingId === "batch"
                        ? "Đang thả..."
                        : `Thả Đã Chọn (${selectedEligibleCount})`}
                    </button>

                    <button
                      type="button"
                      onClick={() => {
                        setSelectedGuIds({});
                        setIsBatchMode(false);
                      }}
                      className="text-[10px] font-medium py-1 px-2.5 bg-zinc-900 hover:bg-zinc-850 text-zinc-400 hover:text-zinc-200 border border-zinc-800 rounded-lg font-mono transition-all cursor-pointer"
                    >
                      Hủy
                    </button>
                  </>
                ) : (
                  <>
                    {(onFeedAll || onFeedGroup) && (
                      <button
                        type="button"
                        disabled={actionLoading || feedingAll}
                        onClick={handleFeedAllClick}
                        className="text-[10px] font-bold py-1 px-2.5 bg-emerald-950/20 text-emerald-400 border border-emerald-900 hover:bg-emerald-950/40 rounded-lg font-mono transition-all cursor-pointer disabled:opacity-50"
                      >
                        {feedingAll ? "🌱 Đang Cho Ăn..." : "🌱 Cho Ăn Tất Cả"}
                      </button>
                    )}
                    {guSubTab === "equipped" && (
                      <button
                        type="button"
                        disabled={actionLoading || unequippingAll || equippedBugCount === 0}
                        onClick={handleUnequipAllClick}
                        className="text-[10px] font-bold py-1 px-2.5 bg-amber-950/20 text-amber-400 border border-amber-900 hover:bg-amber-950/40 rounded-lg font-mono transition-all cursor-pointer disabled:opacity-50"
                      >
                        {unequippingAll ? "🛡️ Đang Tháo..." : "🛡️ Tháo Tất Cả"}
                      </button>
                    )}
                    {onDropGus && guSubTab === "rest" && (
                      <button
                        type="button"
                        onClick={() => {
                          setIsBatchMode(true);
                        }}
                        className="text-[10px] font-bold py-1 px-2.5 bg-rose-950/20 text-rose-400 border border-rose-900 hover:bg-rose-950/40 rounded-lg font-mono transition-all cursor-pointer"
                      >
                        🗑️ Thả Hàng Loạt
                      </button>
                    )}
                  </>
                )}
              </div>
            )}
          </div>

          {filteredGus.length === 0 ? (
            <div className="text-center py-6 border border-dashed border-zinc-900 rounded-2xl text-[10px] text-zinc-600 italic">
              Không tìm thấy Cổ trùng nào khớp.
            </div>
          ) : (
            (() => {
              const renderGroup = ({
                guId,
                groupKey,
                spec,
                instances,
                hasVital,
                activeCount,
                sleepingCount,
                sealedCount,
                dyingCount,
                deadCount,
                equippedCount,
              }: typeof guGroups[number]) => {
                    const isExpanded = !!expandedGroups[groupKey];
                    const totalCount = instances.length;

                    // Food info
                    const preferredFoodId = spec.food_types[0] || "unknown";
                    const preferredFoodName = itemDictionary[preferredFoodId]?.name || preferredFoodId;

                    const availableFoodId = spec.food_types.find(
                      foodId => (materials[foodId] || 0) > 0
                    );
                    const foodName = availableFoodId
                      ? itemDictionary[availableFoodId]?.name || availableFoodId
                      : preferredFoodName;
                    const availableFood = availableFoodId
                      ? materials[availableFoodId]
                      : 0;

                    // Group feed status: are there hungry alive bugs that aren't sealed?
                    const hungryBugsCount = instances.filter(
                      inst => inst.status !== "DEAD" && inst.status !== "SEALED" && inst.hunger < spec.max_hunger
                    ).length;
                    const canFeedGroup = onFeedGroup && hungryBugsCount > 0 &&
                      (availableFood > 0 || spec.food_types.includes("primeval_stone") || spec.food_types.includes("immortal_stone"));
                    const isCurrentlyFeedingGroup = feedingGroupGuId === groupKey;

                    // Build dynamic styling based on group attributes
                    let cardStyle = "bg-zinc-950/60 border-zinc-900 hover:border-zinc-800";
                    if (hasVital) {
                      cardStyle = "border-amber-400/60 bg-amber-950/5 shadow-[0_0_15px_rgba(245,158,11,0.1)] hover:border-amber-500/80";
                    }

                    // Tóm tắt trạng thái
                    const statusTextParts: string[] = [];
                    if (equippedCount > 0) statusTextParts.push(`${equippedCount} Xuất Trận`);
                    const idleCount = totalCount - equippedCount - sleepingCount - sealedCount - deadCount - dyingCount;
                    if (idleCount > 0) statusTextParts.push(`${idleCount} Chờ Lệnh`);
                    if (sleepingCount > 0) statusTextParts.push(`${sleepingCount} Ngủ Đông`);
                    if (sealedCount > 0) statusTextParts.push(`${sealedCount} Phong Ấn`);
                    if (dyingCount > 0) statusTextParts.push(`${dyingCount} Hấp Hối`);
                    if (deadCount > 0) statusTextParts.push(`${deadCount} Đã Chết`);
                    const statusSummary = statusTextParts.join(", ");

                    const eligibleGrpGus = instances.filter(isEligibleForCurrentAction);
                    const grpSelectedCount = eligibleGrpGus.filter(g => selectedGuIds[g.instance_id]).length;
                    const isAllGrpSelected = eligibleGrpGus.length > 0 && grpSelectedCount === eligibleGrpGus.length;

                    const handleGroupSelectToggle = () => {
                      const newSelected = { ...selectedGuIds };
                      if (isAllGrpSelected) {
                        eligibleGrpGus.forEach(g => {
                          newSelected[g.instance_id] = false;
                        });
                      } else {
                        eligibleGrpGus.forEach(g => {
                          newSelected[g.instance_id] = true;
                        });
                      }
                      setSelectedGuIds(newSelected);
                    };

                    return (
                      <div
                        key={groupKey}
                        className={`p-3 sm:p-5 rounded-2xl border transition-all ${cardStyle}`}
                      >
                        {/* Summary Header */}
                        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3">
                          <div className="flex items-center gap-2.5 flex-wrap">
                            {isBatchMode && eligibleGrpGus.length > 0 && (
                              <input
                                type="checkbox"
                                checked={isAllGrpSelected}
                                onChange={handleGroupSelectToggle}
                                className="rounded border-zinc-800 bg-zinc-950 text-amber-550 focus:ring-amber-500/50 cursor-pointer mr-1"
                                title="Chọn cả nhóm"
                              />
                            )}
                            <h4 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
                              {spec.name}
                              <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-zinc-900 border border-zinc-800 text-zinc-400">
                                x{totalCount}
                              </span>
                            </h4>
                            {hasVital && (
                              <span className="text-[7px] font-bold text-amber-300 bg-amber-950/60 border border-amber-500/60 px-1 py-0.2 rounded uppercase tracking-wider animate-pulse shadow-[0_0_5px_rgba(245,158,11,0.3)]">
                                ⚡ Bản Mệnh
                              </span>
                            )}
                            <span className="text-[8px] font-mono font-bold text-amber-600 dark:text-amber-400 border border-amber-500/20 px-1 py-0.5 rounded uppercase">
                              Bậc {spec.rank}
                            </span>
                            {spec.path && spec.path.map(p => (
                              <span key={p} className={`text-[8px] font-mono px-1 rounded border leading-normal ${getPathColor(p)}`}>
                                {translatePath(p)}
                              </span>
                            ))}

                            {/* Status & Hunger display when totalCount === 1 */}
                            {totalCount === 1 ? (() => {
                              const single = instances[0];
                              const status = single.status || "ACTIVE";
                              const hasHunger = spec.gu_type !== "Consumable" || (spec.max_hunger && spec.max_hunger > 0);
                              const hungerPercent = hasHunger
                                ? Math.min(100, Math.max(0, (single.hunger / (spec.max_hunger || 1)) * 100))
                                : 0;

                              return (
                                <div className="flex items-center gap-2 flex-wrap">
                                  {single.equipped && (
                                    <span className="text-[8px] font-mono font-bold text-emerald-400 bg-emerald-950/40 border border-emerald-800/60 px-1.5 py-0.5 rounded uppercase">
                                      Xuất Trận
                                    </span>
                                  )}
                                  <span className={`text-[9px] font-mono font-medium ${
                                    status === "ACTIVE" ? "text-emerald-400" :
                                    status === "SLEEPING" ? "text-amber-400" :
                                    status === "SEALED" ? "text-cyan-400" :
                                    status === "DYING" ? "text-rose-400 animate-pulse font-bold" :
                                    "text-zinc-500"
                                  }`}>
                                    {status === "ACTIVE" && "☀️ Hoạt Động"}
                                    {status === "SLEEPING" && "💤 Ngủ Đông"}
                                    {status === "SEALED" && "🔒 Phong Ấn"}
                                    {status === "DYING" && "💀 Hấp Hối"}
                                    {status === "DEAD" && "🪦 Đã Chết"}
                                  </span>

                                  {hasHunger && (
                                    <div className="flex items-center gap-1.5 bg-zinc-900/80 px-2.5 py-1 rounded-xl border border-zinc-800/80">
                                      <span className="text-[10px] text-zinc-400 font-mono font-medium">Độ no:</span>
                                      <div className="w-16 sm:w-24 h-2 bg-zinc-950 rounded-full overflow-hidden border border-zinc-800">
                                        <div
                                          className={`h-full rounded-full transition-all duration-300 ${
                                            hungerPercent < 20
                                              ? "bg-rose-500 animate-pulse"
                                              : hungerPercent < 50
                                              ? "bg-amber-500"
                                              : "bg-emerald-500"
                                          }`}
                                          style={{ width: `${hungerPercent}%` }}
                                        />
                                      </div>
                                      <span className="text-[10px] font-mono text-zinc-200 font-bold">
                                        {single.hunger}/{spec.max_hunger}
                                      </span>
                                    </div>
                                  )}
                                </div>
                              );
                            })() : statusSummary ? (
                              <span className="text-[9px] text-zinc-400 font-mono bg-zinc-900/50 px-2 py-0.5 rounded border border-zinc-850">
                                {statusSummary}
                              </span>
                            ) : null}
                          </div>

                          {/* Group actions */}
                          <div className="flex items-center gap-2 self-stretch sm:self-auto justify-end flex-wrap">
                            {(spec.gu_type !== "Consumable" || (spec.max_hunger && spec.max_hunger > 0)) && onFeedGroup && (
                              <button
                                type="button"
                                disabled={actionLoading || isCurrentlyFeedingGroup}
                                onClick={() => {
                                  if (hungryBugsCount === 0) {
                                    useGameStore.getState().addToast("Không có Cổ trùng nào trong nhóm này đang đói.", "info");
                                    return;
                                  }
                                  if (availableFood === 0 && !spec.food_types.includes("primeval_stone") && !spec.food_types.includes("immortal_stone")) {
                                    useGameStore.getState().addToast(`Đạo hữu không có thức ăn phù hợp (${foodName}) để cho ăn!`, "error");
                                    return;
                                  }
                                  handleFeedGroupClick(guId, spec.rank);
                                }}
                                title={hungryBugsCount === 0
                                  ? "Không có con nào đói"
                                  : spec.food_types.includes("primeval_stone")
                                    ? `Cho ăn tất cả con đang đói. Cần: Nguyên Thạch (Sở hữu: ${Math.floor(player.primeval_stones || 0)})`
                                    : spec.food_types.includes("immortal_stone")
                                      ? `Cho ăn tất cả con đang đói. Cần: Tiên Nguyên Thạch (Sở hữu: ${Math.floor(player.immortal_stones || 0)})`
                                      : `Cho ăn tất cả con đang đói. Cần: ${foodName} (Sở hữu: ${availableFood})`}
                                className={`text-[9px] font-bold py-1.5 px-3 rounded-xl flex items-center gap-1 transition-all cursor-pointer font-mono ${
                                  canFeedGroup
                                    ? "bg-emerald-600 text-white hover:bg-emerald-500 active:scale-[0.96] shadow-[0_0_8px_rgba(16,185,129,0.2)]"
                                    : "bg-zinc-900 text-zinc-650 border border-zinc-850"
                                }`}
                              >
                                {isCurrentlyFeedingGroup ? "..." : totalCount === 1 ? "Cho Ăn" : `Cho Ăn Nhóm (${hungryBugsCount} con)`}
                              </button>
                            )}

                            {/* Per-group "Use Multiple" button for Consumable Gu */}
                            {spec.gu_type === "Consumable" && onUseGus && instances.filter(isEligibleForUse).length > 1 && useMultipleGuGroupKey !== groupKey && (
                              <button
                                type="button"
                                onClick={() => {
                                  setUseMultipleGuGroupKey(groupKey);
                                  setUseMultipleGuQty(1);
                                }}
                                className="text-[9px] font-bold py-1.5 px-3 rounded-xl bg-amber-950/20 text-amber-400 border border-amber-900 hover:bg-amber-950/40 font-mono transition-all cursor-pointer"
                              >
                                ✨ Sử dụng nhiều
                              </button>
                            )}

                            <button
                              type="button"
                              onClick={() => toggleGroupExpand(groupKey)}
                              className="text-[9px] py-1.5 px-3 bg-zinc-900 hover:bg-zinc-850 text-zinc-350 hover:text-zinc-150 border border-zinc-800 rounded-xl font-mono transition-all flex items-center gap-1 cursor-pointer"
                            >
                              <span>{isExpanded ? "Thu gọn" : "Chi tiết"}</span>
                              {isExpanded ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
                            </button>
                          </div>
                        </div>

                        {/* Quantity picker for multi-use Gu group */}
                        {useMultipleGuGroupKey === groupKey && (() => {
                          const eligibleCount = instances.filter(isEligibleForUse).length;
                          return (
                            <div className="flex items-center gap-2 mt-3 px-3 py-2 border border-amber-900/30 bg-amber-950/10 rounded-xl flex-wrap">
                              <span className="text-[10px] text-amber-400 font-mono font-bold whitespace-nowrap">Số lượng:</span>
                              <input
                                type="number"
                                min={1}
                                max={eligibleCount}
                                value={useMultipleGuQty}
                                onChange={(e) => setUseMultipleGuQty(Math.max(1, Math.min(eligibleCount, parseInt(e.target.value) || 1)))}
                                className="w-16 bg-zinc-950 border border-amber-900/50 text-amber-300 text-xs font-mono text-center py-1 rounded-lg outline-none focus:border-amber-500/60"
                              />
                              <span className="text-[9px] text-zinc-500 font-mono">/ {eligibleCount} đủ điều kiện</span>
                              <button
                                type="button"
                                onClick={() => setUseMultipleGuQty(eligibleCount)}
                                className="text-[9px] py-0.5 px-1.5 bg-zinc-900 text-zinc-400 border border-zinc-800 hover:text-zinc-200 rounded font-mono cursor-pointer"
                              >
                                Tất cả
                              </button>
                              <button
                                type="button"
                                onClick={() => handleUseMultipleGus(groupKey, instances)}
                                disabled={actionLoading || usingGuId === "batch"}
                                className="text-[9px] font-bold py-1 px-2.5 bg-amber-600 text-white hover:bg-amber-500 rounded font-mono transition-all cursor-pointer shadow-[0_0_8px_rgba(245,158,11,0.2)]"
                              >
                                {usingGuId === "batch" ? "Đang sử dụng..." : "Xác nhận"}
                              </button>
                              <button
                                type="button"
                                onClick={() => { setUseMultipleGuGroupKey(null); setUseMultipleGuQty(1); }}
                                className="text-[9px] py-1 px-2 bg-zinc-900 text-zinc-400 border border-zinc-800 hover:text-zinc-200 rounded font-mono cursor-pointer"
                              >
                                Hủy
                              </button>
                            </div>
                          );
                        })()}

                        {/* Expanded detail list */}
                        {isExpanded && (
                          <div className="mt-4 pt-3 border-t border-zinc-900/80 animate-fade-in space-y-2">
                            {instances.map((gu) => {
                              const status = gu.status || "ACTIVE";
                              const isDead = status === "DEAD";
                              const hungerPercent = Math.min(100, Math.max(0, (gu.hunger / spec.max_hunger) * 100));
                              const isHungry = gu.hunger < spec.max_hunger;
                              const canFeedIndividual = (availableFood > 0 && isHungry && !isDead && status !== "SEALED") || spec.food_types.includes("primeval_stone") || spec.food_types.includes("immortal_stone");

                              return (
                                <div
                                  key={gu.instance_id}
                                  className="flex flex-col md:flex-row md:items-center justify-between gap-3 p-3 bg-zinc-950/40 rounded-xl border border-zinc-900/60"
                                >
                                  {/* Info */}
                                  <div className="flex items-center gap-3 flex-wrap">
                                    {isBatchMode && (
                                      <input
                                        type="checkbox"
                                        disabled={!isEligibleForCurrentAction(gu)}
                                        checked={!!selectedGuIds[gu.instance_id]}
                                        onChange={(e) => {
                                          setSelectedGuIds((prev) => ({
                                            ...prev,
                                            [gu.instance_id]: e.target.checked,
                                          }));
                                        }}
                                        className={`rounded border-zinc-800 bg-zinc-950 text-amber-550 focus:ring-amber-550/50 cursor-pointer ${
                                          !isEligibleForCurrentAction(gu) ? "opacity-30 cursor-not-allowed" : ""
                                        }`}
                                        title={
                                          !isEligibleForCurrentAction(gu)
                                            ? "Không thể thả Bản Mệnh Cổ hoặc Cổ trùng đang xuất trận"
                                            : "Chọn Cổ trùng"
                                        }
                                      />
                                    )}
                                    <button
                                      type="button"
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        navigator.clipboard.writeText(gu.instance_id);
                                        setCopiedId(gu.instance_id);
                                        setTimeout(() => setCopiedId(null), 2000);
                                        useGameStore.getState().addToast(`Đã copy ID Cổ trùng: ${gu.instance_id}`, "success");
                                      }}
                                      className="text-[10px] font-mono text-zinc-400 hover:text-amber-400 bg-zinc-900 hover:bg-amber-950/40 px-1.5 py-0.5 rounded border border-zinc-850 hover:border-amber-500/50 cursor-pointer active:scale-95 transition-all inline-flex items-center gap-1 group/id"
                                      title={`Click để copy full ID: ${gu.instance_id}`}
                                    >
                                      {copiedId === gu.instance_id ? (
                                        <span className="text-emerald-400 font-bold flex items-center gap-0.5">
                                          <Check className="w-3 h-3 text-emerald-400" /> Copied
                                        </span>
                                      ) : (
                                        <>
                                          <Copy className="w-2.5 h-2.5 opacity-50 group-hover/id:opacity-100 transition-opacity" />
                                          #{gu.instance_id.slice(0, 6)}
                                        </>
                                      )}
                                    </button>
                                    {gu.is_vital && (
                                      <span className="text-[8px] font-bold text-amber-400 font-mono">👑 Bản Mệnh</span>
                                    )}
                                    {gu.equipped && (
                                      <span className="text-[8px] font-bold text-emerald-400 font-mono bg-emerald-950/30 px-1 py-0.5 rounded border border-emerald-900/30">Xuất Trận</span>
                                    )}
                                    <span className={`text-[9px] font-mono font-medium ${
                                      status === "ACTIVE" ? "text-emerald-600 dark:text-emerald-400" :
                                      status === "SLEEPING" ? "text-amber-600 dark:text-amber-400" :
                                      status === "SEALED" ? "text-cyan-600 dark:text-cyan-400" :
                                      status === "DYING" ? "text-rose-600 dark:text-rose-400 animate-pulse font-bold" :
                                      "text-zinc-650"
                                    }`}>
                                      {status === "ACTIVE" && "☀️ Hoạt Động"}
                                      {status === "SLEEPING" && "💤 Ngủ Đông"}
                                      {status === "SEALED" && "🔒 Phong Ấn"}
                                      {status === "DYING" && "💀 Hấp Hối"}
                                      {status === "DEAD" && "🪦 Đã Chết"}
                                    </span>
                                    {(spec.gu_type !== "Consumable" || (spec.max_hunger && spec.max_hunger > 0)) && (
                                      <div className="flex items-center gap-1.5">
                                        <span className="text-[9px] text-zinc-500 font-mono">Độ no:</span>
                                        <div className="w-16 h-1.5 bg-zinc-900 rounded-full overflow-hidden border border-zinc-850">
                                          <div
                                            className={`h-full rounded-full transition-all duration-300 ${
                                              hungerPercent < 20
                                                ? "bg-rose-500 animate-pulse"
                                                : hungerPercent < 50
                                                ? "bg-amber-505"
                                                : "bg-emerald-500"
                                            }`}
                                            style={{ width: `${hungerPercent}%` }}
                                          />
                                        </div>
                                        <span className="text-[9px] font-mono text-zinc-400">
                                          {gu.hunger}/{spec.max_hunger}
                                        </span>
                                      </div>
                                    )}
                                  </div>

                                  {/* Actions */}
                                  <div className="flex items-center gap-1.5 justify-end flex-wrap">
                                    {/* Feed Individual */}
                                    {(spec.gu_type !== "Consumable" || (spec.max_hunger && spec.max_hunger > 0)) && !isDead && status !== "SEALED" && (
                                      <button
                                        type="button"
                                        disabled={actionLoading || feedingId === gu.instance_id}
                                        onClick={() => {
                                          if (!isHungry) {
                                            useGameStore.getState().addToast("Cổ trùng này đã no, không cần ăn thêm.", "info");
                                            return;
                                          }
                                          if (availableFood === 0 && !spec.food_types.includes("primeval_stone") && !spec.food_types.includes("immortal_stone")) {
                                            useGameStore.getState().addToast(`Đạo hữu không có thức ăn phù hợp (${foodName}) để cho ăn!`, "error");
                                            return;
                                          }
                                          handleFeedClick(gu.instance_id);
                                        }}
                                        className={`text-[9px] font-bold py-1 px-2.5 rounded font-mono transition-all cursor-pointer ${
                                          canFeedIndividual
                                            ? "bg-emerald-950/40 text-emerald-400 border border-emerald-900/40 hover:bg-emerald-950/60"
                                            : "bg-zinc-900/40 text-zinc-600 border border-zinc-900/60"
                                        }`}
                                      >
                                        {feedingId === gu.instance_id ? "..." : "Cho Ăn"}
                                      </button>
                                    )}

                                    {/* Equip/Unequip */}
                                    {spec.gu_type !== "Consumable" && !isDead && status !== "SEALED" && onEquipToggle && (
                                      <button
                                        type="button"
                                        disabled={actionLoading || equippingId === gu.instance_id}
                                        onClick={() => handleEquipClick(gu.instance_id, !gu.equipped)}
                                        className="text-[9px] font-bold py-1 px-2.5 bg-indigo-950/30 text-indigo-400 border border-indigo-900/40 hover:bg-indigo-950/50 rounded font-mono cursor-pointer"
                                      >
                                        {equippingId === gu.instance_id ? "..." : gu.equipped ? "Tháo" : "Trang Bị"}
                                      </button>
                                    )}

                                    {/* Use consumable */}
                                    {spec.gu_type === "Consumable" && onUseGu && (
                                      <button
                                        type="button"
                                        disabled={actionLoading || usingGuId === gu.instance_id}
                                        onClick={() => handleUseGu(gu.instance_id)}
                                        className="text-[9px] font-bold py-1 px-2.5 bg-amber-950/30 text-amber-400 border border-amber-900/40 hover:bg-amber-950/50 rounded font-mono cursor-pointer"
                                      >
                                        {usingGuId === gu.instance_id ? "..." : "Sử Dụng"}
                                      </button>
                                    )}

                                    {/* Drop */}
                                    {onDropGu && !gu.is_vital && !gu.equipped && !isBatchMode && (
                                      <button
                                        type="button"
                                        disabled={actionLoading || droppingId === gu.instance_id}
                                        onClick={async () => {
                                          if (!await useGameStore.getState().showConfirm(`Xác nhận phóng sinh (thả) Cổ trùng này? Hành động này không thể hoàn tác.`)) return;
                                          setDroppingId(gu.instance_id);
                                          try {
                                            await onDropGu(gu.instance_id);
                                          } finally {
                                            setDroppingId(null);
                                          }
                                        }}
                                        className="text-[9px] font-bold py-1 px-2.5 bg-rose-950/30 text-rose-400 border border-rose-900/40 hover:bg-rose-950/50 rounded font-mono cursor-pointer"
                                      >
                                        {droppingId === gu.instance_id ? "..." : "Thả"}
                                      </button>
                                    )}

                                    {/* Hóa Bản Mệnh */}
                                    {!isDead && !gu.is_vital && spec.gu_type !== "Consumable" && onSetVital && (
                                      <button
                                        type="button"
                                        disabled={actionLoading || settingVitalId === gu.instance_id || gu.rank > player.rank}
                                        onClick={() => handleSetVital(gu.instance_id, spec.name)}
                                        className={`text-[9px] py-1 px-2.5 rounded font-mono font-bold transition-all cursor-pointer ${
                                          gu.rank > player.rank
                                            ? "bg-zinc-950/20 text-zinc-600 border border-zinc-950/40 cursor-not-allowed opacity-50"
                                            : "bg-amber-950/30 text-amber-400 border border-amber-900/40 hover:bg-amber-950/50"
                                        }`}
                                        title={gu.rank > player.rank ? `Yêu cầu tu vi Rank ${gu.rank} (Hiện tại: Rank ${player.rank})` : ""}
                                      >
                                        {settingVitalId === gu.instance_id ? "..." : "👑 Bản Mệnh"}
                                      </button>
                                    )}

                                    {/* Status Management */}
                                    {onUpdateStatus && !gu.is_vital && !isDead && (
                                      <>
                                        {/* {status === "ACTIVE" && (
                                          <>
                                            <button
                                              type="button"
                                              disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                              onClick={() => handleStatusUpdate(gu.instance_id, "SLEEPING")}
                                              className="text-[9px] font-bold py-1 px-2.5 bg-zinc-950/50 text-amber-500 border border-zinc-900 hover:bg-zinc-900 rounded font-mono transition-all cursor-pointer"
                                            >
                                              {statusUpdatingId === gu.instance_id ? "..." : "💤 Ngủ"}
                                            </button>
                                            <button
                                              type="button"
                                              disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                              onClick={() => handleStatusUpdate(gu.instance_id, "SEALED")}
                                              className="text-[9px] font-bold py-1 px-2.5 bg-zinc-950/50 text-cyan-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono transition-all cursor-pointer"
                                            >
                                              {statusUpdatingId === gu.instance_id ? "..." : "🔒 Phong Ấn"}
                                            </button>
                                          </>
                                        )}
                                        {status === "SLEEPING" && (
                                          <>
                                            <button
                                              type="button"
                                              disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                              onClick={() => handleStatusUpdate(gu.instance_id, "ACTIVE")}
                                              className="text-[9px] font-bold py-1 px-2.5 bg-zinc-950/50 text-emerald-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono transition-all cursor-pointer"
                                            >
                                              {statusUpdatingId === gu.instance_id ? "..." : "☀️ Tỉnh"}
                                            </button>
                                            <button
                                              type="button"
                                              disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                              onClick={() => handleStatusUpdate(gu.instance_id, "SEALED")}
                                              className="text-[9px] font-bold py-1 px-2.5 bg-zinc-950/50 text-cyan-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono transition-all cursor-pointer"
                                            >
                                              {statusUpdatingId === gu.instance_id ? "..." : "🔒 Phong Ấn"}
                                            </button>
                                          </>
                                        )} */}
                                      </>
                                    )}

                                    {/* Detail toggle */}
                                    <button
                                      type="button"
                                      onClick={() => setDetailOpen(prev => ({...prev, [gu.instance_id]: !prev[gu.instance_id]}))}
                                      className="text-[9px] font-bold py-1 px-2 bg-zinc-950/50 text-zinc-400 border border-zinc-800 hover:bg-zinc-900 hover:text-zinc-300 rounded font-mono transition-all cursor-pointer"
                                      title="Xem chi tiết"
                                    >
                                      {detailOpen[gu.instance_id] ? "▲" : "▼"}
                                    </button>
                                  </div>

                                  {/* Detail panel */}
                                  {detailOpen[gu.instance_id] && (
                                    <div className="w-full mt-2 pt-2 border-t border-zinc-900/60 grid grid-cols-2 md:grid-cols-3 gap-2 text-[10px]">
                                      <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                                        <span className="text-zinc-600 font-mono block mb-1">Loại Cổ</span>
                                        <span className="text-zinc-300">{translateGuType(spec.gu_type)}</span>
                                      </div>
                                      <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                                        <span className="text-zinc-600 font-mono block mb-1">Đường</span>
                                        <div className="flex flex-wrap gap-1">
                                          {spec.path && spec.path.map(p => (
                                            <span key={p} className={`text-[9px] px-1.5 py-0.5 rounded-full ${getPathColor(p)}`}>
                                              {translatePath(p)}
                                            </span>
                                          ))}
                                        </div>
                                      </div>
                                      <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                                        <span className="text-zinc-600 font-mono block mb-1">Độ hiếm</span>
                                        <span className="text-zinc-300">{spec.rarity || "?"}</span>
                                      </div>
                                      <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                                        <span className="text-zinc-600 font-mono block mb-1">Thức Ăn</span>
                                        <span className="text-zinc-300">{preferredFoodName}</span>
                                      </div>
                                      <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                                        <span className="text-zinc-600 font-mono block mb-1">Độ đói mỗi 3 giờ</span>
                                        <span className="text-zinc-300">{spec.hunger_rate}/h</span>
                                      </div>
                                      {spec.effects && Object.keys(spec.effects).length > 0 && (
                                        <div className="col-span-2 md:col-span-3 bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                                          <span className="text-zinc-600 font-mono block mb-1">Hiệu Ứng</span>
                                          <div className="flex flex-col gap-1.5">
                                            {Object.entries(spec.effects).map(([section, effects]) =>
                                              section === "other" && Array.isArray(effects) && effects.length > 0 ? (
                                                <div key={section} className="flex flex-wrap items-center gap-1">
                                                  <span className="text-zinc-500 font-mono text-[9px] uppercase">{section}:</span>
                                                  {effects.map((effect, mi: number) => (
                                                    <span key={mi} className="bg-zinc-900/60 px-1.5 py-0.5 rounded border border-zinc-800/50 whitespace-nowrap">
                                                      {effect?.event_trigger || "?"} → {effect?.action_code || "?"}
                                                      {effect?.params && Object.keys(effect.params).length > 0 ? ` (${JSON.stringify(effect.params)})` : ""}
                                                    </span>
                                                  ))}
                                                </div>
                                              ) : (
                                                effects && typeof effects === 'object' && Object.keys(effects).length > 0 && (
                                                  <div key={section} className="flex flex-wrap items-center gap-1">
                                                    <span className="text-zinc-500 font-mono text-[9px] uppercase">{section}:</span>
                                                    {Object.entries(effects).map(([k, v]) => (
                                                      <span key={k} className="bg-zinc-900/60 px-1.5 py-0.5 rounded border border-zinc-800/50 whitespace-nowrap">
                                                        {FormatKey(k)}: {String(v)}
                                                      </span>
                                                    ))}
                                                  </div>
                                                )
                                              )
                                            )}
                                          </div>
                                        </div>
                                      )}
                                      <Link
                                        href={`/wiki/gu/${spec.id}`}
                                        className="col-span-2 md:col-span-3 bg-zinc-900/40 hover:bg-zinc-800/60 rounded-lg p-2 border border-zinc-800/50 text-zinc-400 hover:text-zinc-200 transition-colors flex items-center justify-center gap-1.5 font-mono text-[10px]"
                                      >
                                        <BookOpen className="w-3.5 h-3.5" />
                                        Xem Chi Tiết Trên Wiki
                                      </Link>
                                    </div>
                                  )}
                                </div>
                              );
                            })}
                          </div>
                        )}
                      </div>
                    );
                  };

              return (
                <GamePanel variant="default" className="space-y-6">
                  {guSubTab === "equipped" ? (
                    equippedGroups.length > 0 ? (
                      <div>
                        <h3 className="text-xs font-bold text-emerald-400 mb-3 flex items-center gap-2">
                          <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                          Đang Trang Bị ({equippedBugCount})
                        </h3>
                        <div className="space-y-4">
                          {equippedGroups.map(renderGroup)}
                        </div>
                      </div>
                    ) : (
                      <div className="text-center py-8 text-xs text-zinc-500 font-mono italic border border-dashed border-zinc-900 rounded-xl">
                        Chưa có Cổ trùng nào đang xuất trận.
                      </div>
                    )
                  ) : (
                    restGroups.length > 0 ? (
                      <div>
                        <h3 className="text-xs font-bold text-amber-400/90 mb-3 flex items-center gap-2">
                          <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
                          Rảnh Rỗi ({restBugCount})
                        </h3>
                        <div className="space-y-4">
                          {restGroups.map(renderGroup)}
                        </div>
                      </div>
                    ) : (
                      <div className="text-center py-8 text-xs text-zinc-500 font-mono italic border border-dashed border-zinc-900 rounded-xl">
                        Không có Cổ trùng nào khác trong túi đồ.
                      </div>
                    )
                  )}
                </GamePanel>
              );
            })()
          )}
        </div>
      )}
    </div>
  );
};
