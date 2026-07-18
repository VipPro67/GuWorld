"use client";

import React, { useState } from "react";
import { Player, GuSpec, ItemSpec, GuInstance, useGameStore } from "@/store/useGameStore";
import { Flower2, Wine, Gem, ShieldAlert, Sparkles, HelpCircle, Search, Shield, Zap, ChevronDown, ChevronUp, Lock, Unlock } from "lucide-react";

interface GuInventoryProps {
  player: Player;
  onFeed: (guInstanceId: string) => Promise<void>;
  onFeedGroup?: (guId: string) => Promise<void>;
  actionLoading: boolean;
  guDictionary: Record<string, GuSpec>;
  itemDictionary: Record<string, ItemSpec>;
  onEquipToggle?: (instanceId: string, equip: boolean) => Promise<void>;
  onUseItem?: (itemId: string) => Promise<void>;
  onUpdateStatus?: (instanceId: string, status: string) => Promise<void>;
  onSetVital?: (instanceId: string) => Promise<void>;
  onUseGu?: (instanceId: string) => Promise<void>;
  onDropGu?: (instanceId: string) => Promise<boolean>;
}

const getPathColor = (path: string): string => {
  switch (path) {
    case "Moon Path":
      return "text-indigo-400 bg-indigo-950/30 border-indigo-900/40";
    case "Strength Path":
      return "text-orange-400 bg-orange-950/30 border-orange-900/40";
    case "Earth Path":
      return "text-amber-400 bg-amber-950/30 border-amber-900/40";
    case "Refinement Path":
      return "text-emerald-400 bg-emerald-950/30 border-emerald-900/40";
    case "Rule Path":
      return "text-sky-400 bg-sky-950/30 border-sky-900/40";
    default:
      return "text-zinc-400 bg-zinc-950/30 border-zinc-900/40";
  }
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
  actionLoading,
  guDictionary,
  itemDictionary,
  onEquipToggle,
  onUseItem,
  onUpdateStatus,
  onSetVital,
  onUseGu,
  onDropGu,
}) => {
  const [feedingId, setFeedingId] = useState<string | null>(null);
  const [feedingGroupGuId, setFeedingGroupGuId] = useState<string | null>(null);
  const [expandedGroups, setExpandedGroups] = useState<Record<string, boolean>>({});
  const [equippingId, setEquippingId] = useState<string | null>(null);
  const [usingId, setUsingId] = useState<string | null>(null);
  const [statusUpdatingId, setStatusUpdatingId] = useState<string | null>(null);
  const [usingGuId, setUsingGuId] = useState<string | null>(null);
  const [settingVitalId, setSettingVitalId] = useState<string | null>(null);
  const [droppingId, setDroppingId] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState<Record<string, boolean>>({});
  const [itemDetailOpen, setItemDetailOpen] = useState<Record<string, boolean>>({});
  const [advancedOpen, setAdvancedOpen] = useState<Record<string, boolean>>({});
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedTab, setSelectedTab] = useState<"gu" | "material">("gu");

  const toggleGroupExpand = (guId: string) => {
    setExpandedGroups((prev) => ({ ...prev, [guId]: !prev[guId] }));
  };

  const handleFeedGroupClick = async (guId: string) => {
    if (!onFeedGroup) return;
    setFeedingGroupGuId(guId);
    try {
      await onFeedGroup(guId);
    } finally {
      setFeedingGroupGuId(null);
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


  // Filter materials (only those with qty > 0 and matching searchTerm)
  const filteredMaterials = Object.keys(itemDictionary)
    .filter((key) => {
      const qty = materials[key] || 0;
      if (qty <= 0) return false;

      const spec = itemDictionary[key];
      const matchesSearch =
        spec.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        spec.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (spec.item_type && spec.item_type.toLowerCase().includes(searchTerm.toLowerCase()));
      return matchesSearch;
    })
    .map((key) => ({ key, spec: itemDictionary[key], qty: materials[key] }));

  // Filter Gu bugs matching searchTerm
  const filteredGus = guList.filter((gu) => {
    const spec = guDictionary[gu.gu_id] || {
      name: gu.gu_id,
      description: "Cổ trùng chưa được giải nghĩa.",
      path: [] as string[],
      gu_type: "",
    };
    return (
      spec.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      spec.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (spec.path && spec.path.some(p => p.toLowerCase().includes(searchTerm.toLowerCase()))) ||
      (spec.gu_type && spec.gu_type.toLowerCase().includes(searchTerm.toLowerCase()))
    );
  });

  const showMaterials = selectedTab === "material";
  const showGus = selectedTab === "gu";

  return (
    <div className="glass-panel-gold rounded-3xl p-4 sm:p-6 w-full space-y-4 sm:space-y-6">
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
      </div>

      {/* Materials Inventory section */}
      {showMaterials && (
        <div className="space-y-2 text-left">
          <span className="text-[10px] font-mono text-zinc-505 uppercase tracking-widest block font-bold">
            Nguyên Liệu Sở Hữu
          </span>
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
                        {isConsumable && onUseItem && (
                          <button
                            onClick={() => handleUseItemClick(key)}
                            disabled={actionLoading || isUsing}
                            className="text-[9px] py-1 px-2 bg-emerald-950/30 text-emerald-400 border border-emerald-900/40 hover:bg-emerald-950/50 hover:border-emerald-800 rounded font-mono transition-all cursor-pointer font-bold"
                          >
                            {isUsing ? "..." : "Sử Dụng"}
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
                                  <span key={p} className={getPathColor(p)}>{p}</span>
                                ))
                              ) : (
                                <span className={getPathColor(spec.path as string)}>{spec.path as string}</span>
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
                          <span className="text-zinc-300">{spec.item_type || spec.type || "?"}</span>
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
          <span className="text-[10px] font-mono text-zinc-505 uppercase tracking-widest block font-bold">
            Cổ Trùng Sở Hữu
          </span>

          {filteredGus.length === 0 ? (
            <div className="text-center py-6 border border-dashed border-zinc-900 rounded-2xl text-[10px] text-zinc-600 italic">
              Không tìm thấy Cổ trùng nào khớp.
            </div>
          ) : (
            (() => {
              // Group the filtered Gu instances by gu_id
              const groupsMap: Record<string, GuInstance[]> = {};
              filteredGus.forEach((gu) => {
                if (!groupsMap[gu.gu_id]) {
                  groupsMap[gu.gu_id] = [];
                }
                groupsMap[gu.gu_id].push(gu);
              });

              // Construct list of groups with stats
              const guGroups = Object.entries(groupsMap).map(([guId, instances]) => {
                const spec = guDictionary[guId] || {
                  id: guId,
                  name: guId,
                  rank: 1,
                  path: ["Unknown Path"],
                  gu_type: "Phàm Cổ",
                  base_mana_cost: 0,
                  effects: {},
                  hunger_rate: 0,
                  food_types: ["unknown"],
                  max_hunger: 100,
                  description: "Cổ trùng chưa được giải nghĩa.",
                };

                // Sort instances: Vital first, then Equipped first, then ACTIVE > SLEEPING > SEALED > Hunger (descending)
                const statusPriority = {
                  'ACTIVE': 1,
                  'SLEEPING': 2,
                  'SEALED': 3,
                  'DYING': 4,
                  'DEAD': 5
                };

                instances.sort((a, b) => {
                  if (a.is_vital !== b.is_vital) return a.is_vital ? -1 : 1;
                  if (a.equipped !== b.equipped) return a.equipped ? -1 : 1;
                  const pA = statusPriority[a.status as keyof typeof statusPriority] || 9;
                  const pB = statusPriority[b.status as keyof typeof statusPriority] || 9;
                  if (pA !== pB) return pA - pB;
                  return b.hunger - a.hunger;
                });

                const hasVital = instances.some(inst => inst.is_vital);
                const activeCount = instances.filter(inst => inst.status === "ACTIVE").length;
                const sleepingCount = instances.filter(inst => inst.status === "SLEEPING").length;
                const sealedCount = instances.filter(inst => inst.status === "SEALED").length;
                const dyingCount = instances.filter(inst => inst.status === "DYING").length;
                const deadCount = instances.filter(inst => inst.status === "DEAD").length;
                const equippedCount = instances.filter(inst => inst.equipped).length;

                return {
                  guId,
                  spec,
                  instances,
                  hasVital,
                  activeCount,
                  sleepingCount,
                  sealedCount,
                  dyingCount,
                  deadCount,
                  equippedCount,
                };
              });

              // Sort groups: hasVital first, then equipped first, then Rank descending, then Name ascending
              guGroups.sort((a, b) => {
                if (a.hasVital !== b.hasVital) return a.hasVital ? -1 : 1;
                const aEquipped = a.equippedCount > 0;
                const bEquipped = b.equippedCount > 0;
                if (aEquipped !== bEquipped) return aEquipped ? -1 : 1;
                if (b.spec.rank !== a.spec.rank) return b.spec.rank - a.spec.rank;
                return a.spec.name.localeCompare(b.spec.name);
              });

              return (
                <div className="space-y-4">
                  {guGroups.map(({
                    guId,
                    spec,
                    instances,
                    hasVital,
                    activeCount,
                    sleepingCount,
                    sealedCount,
                    dyingCount,
                    deadCount,
                    equippedCount,
                  }) => {
                    const isExpanded = !!expandedGroups[guId];
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
                    const canFeedGroup = availableFood > 0 && hungryBugsCount > 0 && onFeedGroup;
                    const isCurrentlyFeedingGroup = feedingGroupGuId === guId;

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

                    return (
                      <div
                        key={guId}
                        className={`p-4 sm:p-5 rounded-2xl border transition-all ${cardStyle}`}
                      >
                        {/* Summary Header */}
                        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3">
                          <div className="flex items-center gap-3 flex-wrap">
                            <h4 className="text-sm font-bold text-zinc-100 font-serif flex items-center gap-2">
                              {spec.name}
                              <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-zinc-900 border border-zinc-800 text-zinc-405">
                                x{totalCount}
                              </span>
                            </h4>
                            {hasVital && (
                              <span className="text-[7px] font-bold text-amber-300 bg-amber-950/60 border border-amber-500/60 px-1 py-0.2 rounded uppercase tracking-wider animate-pulse shadow-[0_0_5px_rgba(245,158,11,0.3)]">
                                ⚡ Bản Mệnh
                              </span>
                            )}
                            <span className="text-[8px] font-mono font-bold text-amber-600 dark:text-amber-400 border border-amber-500/20 px-1 py-0.5 rounded uppercase">
                              Rank {spec.rank}
                            </span>
                            {spec.path && spec.path.map(p => (
                              <span key={p} className={`text-[8px] font-mono px-1 rounded border leading-normal ${getPathColor(p)}`}>
                                {p}
                              </span>
                            ))}
                          </div>

                          {/* Group actions */}
                          <div className="flex items-center gap-2 self-stretch sm:self-auto justify-end">
                            {spec.gu_type !== "Consumable" && onFeedGroup && (
                              <button
                                type="button"
                                disabled={actionLoading || isCurrentlyFeedingGroup}
                                onClick={() => {
                                  if (hungryBugsCount === 0) {
                                    useGameStore.getState().addToast("Không có Cổ trùng nào trong nhóm này đang đói.", "warning");
                                    return;
                                  }
                                  if (availableFood === 0) {
                                    useGameStore.getState().addToast(`Đạo hữu không có thức ăn phù hợp (${foodName}) để cho ăn!`, "error");
                                    return;
                                  }
                                  handleFeedGroupClick(guId);
                                }}
                                title={hungryBugsCount === 0 ? "Không có con nào đói" : `Cho ăn tất cả con đang đói. Cần: ${foodName} (Sở hữu: ${availableFood})`}
                                className={`text-[9px] font-bold py-1.5 px-3 rounded-xl flex items-center gap-1 transition-all cursor-pointer font-mono ${
                                  canFeedGroup
                                    ? "bg-emerald-600 text-white hover:bg-emerald-500 active:scale-[0.96] shadow-[0_0_8px_rgba(16,185,129,0.2)]"
                                    : "bg-zinc-900 text-zinc-650 border border-zinc-850"
                                }`}
                              >
                                {isCurrentlyFeedingGroup ? "..." : `Cho Ăn Nhóm (${hungryBugsCount} con)`}
                              </button>
                            )}

                            <button
                              type="button"
                              onClick={() => toggleGroupExpand(guId)}
                              className="text-[9px] py-1.5 px-3 bg-zinc-900 hover:bg-zinc-850 text-zinc-350 hover:text-zinc-150 border border-zinc-800 rounded-xl font-mono transition-all flex items-center gap-1 cursor-pointer"
                            >
                              <span>{isExpanded ? "Thu gọn" : "Chi tiết"}</span>
                              {isExpanded ? <ChevronUp className="w-3.5 h-3.5" /> : <ChevronDown className="w-3.5 h-3.5" />}
                            </button>
                          </div>
                        </div>

                        {/* Expanded detail list */}
                        {isExpanded && (
                          <div className="mt-4 pt-3 border-t border-zinc-900/80 animate-fade-in space-y-2">
                            {instances.map((gu) => {
                              const status = gu.status || "ACTIVE";
                              const isDead = status === "DEAD";
                              const hungerPercent = Math.min(100, Math.max(0, (gu.hunger / spec.max_hunger) * 100));
                              const isHungry = gu.hunger < spec.max_hunger;
                              const canFeedIndividual = availableFood > 0 && isHungry && !isDead && status !== "SEALED";

                              return (
                                <div
                                  key={gu.instance_id}
                                  className="flex flex-col md:flex-row md:items-center justify-between gap-3 p-3 bg-zinc-950/40 rounded-xl border border-zinc-900/60"
                                >
                                  {/* Info */}
                                  <div className="flex items-center gap-3 flex-wrap">
                                    <span className="text-[10px] font-mono text-zinc-400 bg-zinc-900 px-1.5 py-0.5 rounded border border-zinc-850">
                                      #{gu.instance_id.slice(0, 6)}
                                    </span>
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
                                    {spec.gu_type !== "Consumable" && (
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
                                    {spec.gu_type !== "Consumable" && !isDead && status !== "SEALED" && (
                                      <button
                                        type="button"
                                        disabled={actionLoading || feedingId === gu.instance_id}
                                        onClick={() => {
                                          if (!isHungry) {
                                            useGameStore.getState().addToast("Cổ trùng này đã no, không cần ăn thêm.", "warning");
                                            return;
                                          }
                                          if (availableFood === 0) {
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

                                    {/* Detail toggle */}
                                    <button
                                      type="button"
                                      onClick={() => setDetailOpen(prev => ({...prev, [gu.instance_id]: !prev[gu.instance_id]}))}
                                      className="text-[9px] py-1 px-2 bg-zinc-950/50 text-zinc-400 border border-zinc-800 hover:bg-zinc-900 hover:text-zinc-300 rounded font-mono transition-all cursor-pointer"
                                      title="Xem chi tiết"
                                    >
                                      {detailOpen[gu.instance_id] ? "▲" : "▼"}
                                    </button>

                                    {/* Advanced actions toggle */}
                                    {spec.gu_type !== "Consumable" && (
                                      <button
                                        type="button"
                                        onClick={() => setAdvancedOpen(prev => ({...prev, [gu.instance_id]: !prev[gu.instance_id]}))}
                                        className="text-[9px] py-1 px-1.5 bg-zinc-950/50 text-zinc-500 border border-zinc-800 hover:bg-zinc-900 hover:text-zinc-300 rounded font-mono transition-all cursor-pointer"
                                      >
                                        {advancedOpen[gu.instance_id] ? "..." : "···"}
                                      </button>
                                    )}

                                    {/* Advanced actions panel */}
                                    {advancedOpen[gu.instance_id] && (
                                      <div className="w-full flex items-center gap-1.5 justify-end flex-wrap pt-1.5 mt-1.5 border-t border-zinc-900/60">
                                        {/* Drop */}
                                        {onDropGu && !gu.is_vital && !gu.equipped && (
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
                                        {!isDead && !gu.is_vital && onSetVital && (
                                          <button
                                            type="button"
                                            disabled={actionLoading || settingVitalId === gu.instance_id}
                                            onClick={() => handleSetVital(gu.instance_id, spec.name)}
                                            className="text-[8px] py-1 px-1.5 bg-zinc-950/50 text-amber-400 border border-zinc-900 hover:bg-amber-950/20 hover:border-amber-500/50 rounded font-mono font-bold transition-all cursor-pointer"
                                          >
                                            {settingVitalId === gu.instance_id ? "..." : "👑 Bản Mệnh"}
                                          </button>
                                        )}

                                        {/* Quản lý trạng thái (Ngủ / Tỉnh / Phong Ấn / Giải Ấn) */}
                                        {onUpdateStatus && !gu.is_vital && !isDead && (
                                          <>
                                            {status === "ACTIVE" && (
                                              <>
                                                <button
                                                  type="button"
                                                  disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                                  onClick={() => handleStatusUpdate(gu.instance_id, "SLEEPING")}
                                                  className="text-[8px] py-1 px-1.5 bg-zinc-950/50 text-amber-500 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer"
                                                >
                                                  {statusUpdatingId === gu.instance_id ? "..." : "💤 Ngủ"}
                                                </button>
                                                <button
                                                  type="button"
                                                  disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                                  onClick={() => handleStatusUpdate(gu.instance_id, "SEALED")}
                                                  className="text-[8px] py-1 px-1.5 bg-zinc-950/50 text-cyan-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer"
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
                                                  className="text-[8px] py-1 px-1.5 bg-zinc-950/50 text-emerald-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer"
                                                >
                                                  {statusUpdatingId === gu.instance_id ? "..." : "☀️ Tỉnh"}
                                                </button>
                                                <button
                                                  type="button"
                                                  disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                                  onClick={() => handleStatusUpdate(gu.instance_id, "SEALED")}
                                                  className="text-[8px] py-1 px-1.5 bg-zinc-950/50 text-cyan-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer"
                                                >
                                                  {statusUpdatingId === gu.instance_id ? "..." : "🔒 Phong Ấn"}
                                                </button>
                                              </>
                                            )}
                                            {status === "SEALED" && (
                                              <button
                                                type="button"
                                                disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                                onClick={() => handleStatusUpdate(gu.instance_id, "ACTIVE")}
                                                className="text-[8px] py-1 px-1.5 bg-zinc-950/50 text-emerald-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer"
                                              >
                                                {statusUpdatingId === gu.instance_id ? "..." : "🔓 Giải Ấn"}
                                              </button>
                                            )}
                                          </>
                                        )}
                                      </div>
                                    )}
                                  </div>

                                  {/* Detail panel */}
                                  {detailOpen[gu.instance_id] && (
                                    <div className="w-full mt-2 pt-2 border-t border-zinc-900/60 grid grid-cols-2 md:grid-cols-3 gap-2 text-[10px]">
                                      <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                                        <span className="text-zinc-600 font-mono block mb-1">Loại Cổ</span>
                                        <span className="text-zinc-300">{spec.gu_type}</span>
                                      </div>
                                      <div className="bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                                        <span className="text-zinc-600 font-mono block mb-1">Đường</span>
                                        <div className="flex flex-wrap gap-1">
                                          {spec.path && spec.path.map(p => (
                                            <span key={p} className={getPathColor(p)}>{p}</span>
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
                                          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                                            {Object.entries(spec.effects).map(([section, effects]) =>
                                              effects && typeof effects === 'object' && Object.keys(effects).length > 0 && (
                                                <span key={section} className="inline-flex items-center gap-1">
                                                  <span className="text-zinc-500 font-mono text-[9px] uppercase">{section}:</span>
                                                  {Object.entries(effects).map(([k, v]) => (
                                                    <span key={k} className="bg-zinc-900/60 px-1.5 py-0.5 rounded border border-zinc-800/50 whitespace-nowrap">
                                                      {FormatKey(k)}: {String(v)}
                                                    </span>
                                                  ))}
                                                </span>
                                              )
                                            )}
                                          </div>
                                        </div>
                                      )}
                                      {spec.description && (
                                        <div className="col-span-2 md:col-span-3 bg-zinc-950/40 rounded-lg p-2 border border-zinc-900/50">
                                          <span className="text-zinc-600 font-mono block mb-1">Mô Tả</span>
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
                    );
                  })}
                </div>
              );
            })()
          )}
        </div>
      )}
    </div>
  );
};
