"use client";

import React, { useState } from "react";
import { Player, GuSpec, ItemSpec, GuInstance } from "@/store/useGameStore";
import { Flower2, Wine, Gem, ShieldAlert, Sparkles, HelpCircle, Search, Shield, Zap } from "lucide-react";

interface GuInventoryProps {
  player: Player;
  onFeed: (guInstanceId: string) => Promise<void>;
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
  const [equippingId, setEquippingId] = useState<string | null>(null);
  const [usingId, setUsingId] = useState<string | null>(null);
  const [statusUpdatingId, setStatusUpdatingId] = useState<string | null>(null);
  const [usingGuId, setUsingGuId] = useState<string | null>(null);
  const [settingVitalId, setSettingVitalId] = useState<string | null>(null);
  const [droppingId, setDroppingId] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [selectedTab, setSelectedTab] = useState<"gu" | "material">("gu");

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

    if (confirm(confirmMsg)) {
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
      path: "",
      gu_type: "",
    };
    return (
      spec.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      spec.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (spec.path && spec.path.toLowerCase().includes(searchTerm.toLowerCase())) ||
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
            Bách Bảo Nang (Inventory & Gu Bags)
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
            Nguyên Liệu Sở Hữu (Materials Inventory)
          </span>
          {filteredMaterials.length === 0 ? (
            <div className="text-[10px] text-zinc-600 italic">Hộp nguyên liệu trống.</div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3">
              {filteredMaterials.map(({ key, spec, qty }) => {
                const isConsumable = spec.item_type === "Consumable" || spec.type === "Consumable";
                const isUsing = usingId === key;
                return (
                  <div
                    key={key}
                    className="flex justify-between items-center bg-zinc-950/70 border border-zinc-900 px-3.5 py-2.5 rounded-xl"
                    title={spec.description}
                  >
                    <div className="flex items-center gap-2">
                      <div className="text-left">
                        <div className="text-xs font-semibold text-zinc-200">{spec.name}</div>
                        <div className="text-[9px] text-zinc-500 font-mono">x{qty} sở hữu</div>
                      </div>
                    </div>
                    {isConsumable && onUseItem && (
                      <button
                        onClick={() => handleUseItemClick(key)}
                        disabled={actionLoading || isUsing}
                        className="text-[9px] py-1 px-2.5 bg-emerald-950/30 text-emerald-450 border border-emerald-900/40 hover:bg-emerald-950/50 hover:border-emerald-800 rounded font-mono active:scale-[0.98] transition-all cursor-pointer font-bold"
                      >
                        {isUsing ? "..." : "Sử Dụng"}
                      </button>
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
            Cổ Trùng Sở Hữu (My Gu Bugs)
          </span>

          {filteredGus.length === 0 ? (
            <div className="text-center py-6 border border-dashed border-zinc-900 rounded-2xl text-[10px] text-zinc-600 italic">
              Không tìm thấy Cổ trùng nào khớp.
            </div>
          ) : (
            (() => {
              // 1. Phân loại Bản Mệnh Cổ và Cổ trùng thường
              const vitalGus = filteredGus.filter(gu => gu.is_vital);
              const normalGus = filteredGus.filter(gu => !gu.is_vital);

              // Định nghĩa trọng số cho trạng thái để dễ sắp xếp (ACTIVE ưu tiên cao nhất, DEAD thấp nhất)
              const statusPriority = {
                'ACTIVE': 1,
                'SLEEPING': 2,
                'SEALED': 3,
                'DYING': 4,
                'DEAD': 5
              };

              // 2. Sắp xếp danh sách Cổ trùng thường
              normalGus.sort((a, b) => {
                const specA = guDictionary[a.gu_id] || {};
                const specB = guDictionary[b.gu_id] || {};

                // Ưu tiên 1 Equip
                const equipA = a.equipped ? 1 : 0;
                const equipB = b.equipped ? 1 : 0;
                if (equipA !== equipB) return equipB - equipA;

                // Ưu tiên 2: Trạng thái (ACTIVE > SLEEPING > SEALED...)
                const pA = statusPriority[a.status as keyof typeof statusPriority];
                const pB = statusPriority[b.status as keyof typeof statusPriority];
                if (pA !== pB) return pA - pB;

                // Ưu tiên 3: Độ no (Giảm dần từ No nhiều đến Đói)
                return b.hunger - a.hunger;
              });

              // Hàm helper render từng card Cổ Trùng để tránh lặp lại code
              const renderGuCard = (gu: GuInstance) => {
                const spec = guDictionary[gu.gu_id] || {
                  id: gu.gu_id,
                  name: gu.gu_id,
                  rank: 1,
                  path: "Unknown Path",
                  gu_type: "Phàm Cổ",
                  origin_region: "Unknown Region",
                  base_mana_cost: 0,
                  effects: {},
                  hunger_rate: 0,
                  food_types: ["unknown"],
                  max_hunger: 100,
                  description: "Cổ trùng chưa được giải nghĩa.",
                };

                const hunger = gu.hunger;
                const status = gu.status;
                const isDead = status === "DEAD";
                const isSleeping = status === "SLEEPING";
                const isSealed = status === "SEALED";
                const isEquipped = gu.equipped;

                const availableFoodId = spec.food_types.find(
                    foodId => (materials[foodId] || 0) > 0
                );

                const foodName = availableFoodId
                    ? itemDictionary[availableFoodId]?.name || availableFoodId
                    : null;

                const availableFood = availableFoodId
                    ? materials[availableFoodId]
                    : 0;
                const canFeed = availableFood > 0 && !isDead && !isSealed;
                const isCurrentlyFeeding = feedingId === gu.instance_id;
                const isCurrentlyEquipping = equippingId === gu.instance_id;

                let barColor = "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.3)]";
                if (hunger <= 20) {
                  barColor = "bg-rose-500 animate-pulse shadow-[0_0_10px_rgba(239,68,68,0.5)]";
                } else if (hunger <= 50) {
                  barColor = "bg-yellow-500 shadow-[0_0_8px_rgba(234,179,8,0.3)]";
                }

                const isDying = status === "DYING";

                let cardStyle = "bg-zinc-950/60 border-zinc-900 hover:border-zinc-800";
                if (isDead) {
                  cardStyle = "bg-zinc-950/30 border-zinc-950 opacity-40 select-none";
                } else if (isDying) {
                  cardStyle = "border-rose-600 bg-rose-950/10 shadow-[inset_0_0_15px_rgba(220,38,38,0.1)] animate-pulse";
                } else if (gu.is_vital) {
                  cardStyle = "border-amber-400/80 bg-amber-950/5 shadow-[0_0_15px_rgba(245,158,11,0.15)]";
                } else if (hunger <= 20) {
                  cardStyle = "border-rose-500/80 bg-rose-950/5 shadow-[0_0_15px_rgba(239,68,68,0.2)] animate-pulse";
                } else if (isSleeping) {
                  cardStyle = "bg-zinc-950/70 border-amber-600/30 shadow-[inset_0_0_15px_rgba(245,158,11,0.05)]";
                } else if (isSealed) {
                  cardStyle = "bg-zinc-950/70 border-cyan-600/30 shadow-[inset_0_0_15px_rgba(34,211,238,0.05)]";
                } else if (isEquipped) {
                  cardStyle = "bg-emerald-950/5 border-emerald-500/30 shadow-[0_0_12px_rgba(16,185,129,0.05)]";
                }

                return (
                  <div
                    key={gu.instance_id}
                    className={`relative p-4 rounded-2xl border transition-all flex flex-col justify-between min-h-[14rem] h-auto ${cardStyle}`}
                  >
                    {/* Top: Name and Type */}
                    <div>
                      <div className="flex justify-between items-start gap-1">
                        <div className="flex flex-col gap-0.5">
                          <h4 className="text-xs font-bold text-zinc-100 font-serif flex items-center gap-1.5 flex-wrap">
                            {spec.name}
                            {gu.is_vital && (
                              <span className="text-[7px] font-bold text-amber-300 bg-amber-950/60 border border-amber-500/60 px-1 py-0.2 rounded uppercase tracking-wider shadow-[0_0_5px_rgba(245,158,11,0.4)] animate-pulse">
                                ⚡ Bản Mệnh
                              </span>
                            )}
                          </h4>
                          <div className="text-[8px] text-zinc-500 font-mono mt-0.5">{gu.gu_id}</div>
                        </div>
                        <span className="text-[8px] font-mono font-bold text-amber-400 border border-amber-500/20 px-1 py-0.5 rounded uppercase shrink-0">
                          Rank {spec.rank}
                        </span>
                      </div>
                      <div className="flex flex-wrap gap-1 mt-1">
                        <span className="text-[9px] text-zinc-500 font-mono">
                          {spec.gu_type}
                        </span>
                        <span className={`text-[8px] font-mono px-1 rounded border leading-normal ${getPathColor(spec.path)}`}>
                          {spec.path}
                        </span>
                      </div>
                      <p className="text-[10px] text-zinc-400 mt-2 line-clamp-2 leading-relaxed">
                        {spec.description}
                      </p>

                      {/* Effects and Mana Cost */}
                      <div className="mt-2.5 pt-2 border-t border-zinc-900/40 text-[10px] space-y-1">
                        {spec.base_mana_cost > 0 && (
                          <div className="text-indigo-400 font-mono font-medium">
                            Chân Nguyên: {spec.base_mana_cost}%
                          </div>
                        )}
                        {spec.effects && Object.entries(spec.effects).map(([key, val]) => (
                          <div key={key} className="text-zinc-300 font-mono flex justify-between">
                            <span className="text-zinc-500">{FormatKey(key)}:</span>
                            <span className="text-emerald-450 font-bold">+{val}</span>
                          </div>
                        ))}
                      </div>
                    </div>

                    {/* Bottom: Hunger bar and Actions */}
                    <div className="mt-4 space-y-2">
                      {/* Hunger bar */}
                      {!isDead && spec.gu_type !== "Consumable" && (
                        <div className="space-y-1">
                          <div className="flex justify-between text-[9px] font-mono">
                            <span className="text-zinc-500">Độ no:</span>
                            <span
                              className={
                                hunger <= 20
                                  ? "text-rose-450 font-bold"
                                  : hunger <= 50
                                    ? "text-yellow-450"
                                    : "text-emerald-450"
                              }
                            >
                              {hunger}/{spec.max_hunger}
                            </span>
                          </div>
                          <div className="w-full bg-zinc-900 rounded-full h-1.5 overflow-hidden">
                            <div
                              className={`h-full rounded-full transition-all duration-500 ${barColor}`}
                              style={{ width: `${(hunger / spec.max_hunger) * 100}%` }}
                            />
                          </div>
                        </div>
                      )}

                      {/* Status Badge & Button */}
                      <div className="flex items-center justify-between gap-1.5 pt-1.5 border-t border-zinc-900">
                        {/* Status display */}
                        {isDead ? (
                          <div className="flex items-center gap-1 text-[9px] text-rose-500 font-bold uppercase tracking-wider">
                            <ShieldAlert className="w-3 h-3" />
                            Đã Chết
                          </div>
                        ) : isDying ? (
                          <span className="inline-block text-[8px] font-bold text-rose-500 uppercase bg-rose-950/20 border border-rose-500/20 px-1.5 py-0.5 rounded animate-pulse">
                            Hấp Hối
                          </span>
                        ) : isEquipped ? (
                          <span className="inline-block text-[8px] font-bold text-emerald-400 uppercase bg-emerald-950/20 border border-emerald-500/20 px-1.5 py-0.5 rounded">
                            Xuất Trận
                          </span>
                        ) : isSleeping ? (
                          <span className="inline-block text-[8px] font-bold text-amber-500 uppercase bg-amber-950/20 border border-amber-500/20 px-1.5 py-0.5 rounded animate-pulse">
                            Ngủ Đông
                          </span>
                        ) : isSealed ? (
                          <span className="inline-block text-[8px] font-bold text-cyan-400 uppercase bg-cyan-950/20 border border-cyan-500/20 px-1.5 py-0.5 rounded">
                            Phong Ấn
                          </span>
                        ) : spec.gu_type === "Consumable" ? (
                          <span className="inline-block text-[8px] font-bold text-rose-450 uppercase bg-rose-950/20 border border-rose-500/20 px-1.5 py-0.5 rounded animate-pulse">
                            Tiêu Hao
                          </span>
                        ) : (
                          <span className="inline-block text-[8px] font-bold text-zinc-500 uppercase bg-zinc-950/40 border border-zinc-900 px-1.5 py-0.5 rounded">
                            Chờ Lệnh
                          </span>
                        )}

                        {/* Equip & Feed Actions */}
                        {!isDead && (
                          <div className="flex gap-1.5">
                            {spec.gu_type === "Consumable" ? (
                              <button
                                type="button"
                                disabled={actionLoading || usingGuId === gu.instance_id}
                                onClick={() => handleUseGu(gu.instance_id)}
                                className="text-[9px] font-bold py-1 px-2.5 bg-rose-900/20 text-rose-400 border border-rose-900/40 hover:bg-rose-950/50 hover:border-rose-750 rounded font-mono transition-all cursor-pointer active:scale-[0.98]"
                              >
                                {usingGuId === gu.instance_id ? "..." : "🔥 Sử Dụng"}
                              </button>
                            ) : (
                              <>
                                {onEquipToggle && status === "ACTIVE" && !gu.is_vital && (
                                  <button
                                    type="button"
                                    disabled={actionLoading || isCurrentlyEquipping}
                                    onClick={() => handleEquipClick(gu.instance_id, !isEquipped)}
                                    className={`text-[9px] font-bold py-1 px-2.5 rounded-lg transition-all cursor-pointer font-mono ${isEquipped
                                        ? "bg-amber-950/30 text-amber-400 border border-amber-900/40 hover:bg-amber-950/50"
                                        : "bg-indigo-950/30 text-indigo-400 border border-indigo-900/40 hover:bg-indigo-950/50"
                                      }`}
                                  >
                                    {isCurrentlyEquipping ? "..." : isEquipped ? "Tháo" : "Trang Bị"}
                                  </button>
                                )}
                                <button
                                  type="button"
                                  disabled={actionLoading || !canFeed || isCurrentlyFeeding}
                                  onClick={() => handleFeedClick(gu.instance_id)}
                                  title={status === "SEALED" ? "Cổ trùng đang bị phong ấn, không thể cho ăn" : `Ăn: 1x ${foodName}`}
                                  className={`text-[9px] font-bold py-1 px-2.5 rounded-lg flex items-center justify-center gap-1 transition-all cursor-pointer font-mono ${canFeed
                                      ? "bg-emerald-600 text-white hover:bg-emerald-500 active:scale-[0.96]"
                                      : "bg-zinc-900 text-zinc-600 border border-zinc-850 cursor-not-allowed"
                                    }`}
                                >
                                  {isCurrentlyFeeding ? "..." : "Cho Ăn"}
                                </button>
                                {onDropGu && !gu.is_vital && (
                                  <button
                                    type="button"
                                    disabled={actionLoading || droppingId === gu.instance_id}
                                    onClick={async () => {
                                      if (!confirm(`Vứt bỏ ${spec.name}?`)) return;
                                      setDroppingId(gu.instance_id);
                                      try { await onDropGu(gu.instance_id); } finally { setDroppingId(null); }
                                    }}
                                    className="text-[9px] font-bold py-1 px-2.5 rounded-lg transition-all cursor-pointer font-mono bg-rose-950/30 text-rose-400 border border-rose-900/40 hover:bg-rose-950/50"
                                  >
                                    {droppingId === gu.instance_id ? "..." : "Thả"}
                                  </button>
                                )}
                              </>
                            )}
                          </div>
                        )}
                      </div>

                      {/* Status Transition & Vital Setup Buttons */}
                      {!isDead && spec.gu_type !== "Consumable" && (
                        <div className="flex gap-1.5 pt-1.5 border-t border-zinc-900/60 justify-end flex-wrap">
                          {!gu.is_vital && onSetVital && (
                            <button
                              type="button"
                              disabled={actionLoading || settingVitalId === gu.instance_id}
                              onClick={() => handleSetVital(gu.instance_id, spec.name)}
                              className="text-[8px] py-0.8 px-1.5 bg-zinc-950/40 text-amber-400 border border-zinc-900 hover:bg-amber-950/20 hover:border-amber-500/50 rounded font-mono font-bold transition-all cursor-pointer active:scale-[0.98]"
                            >
                              {settingVitalId === gu.instance_id ? "..." : "👑 Hóa Bản Mệnh"}
                            </button>
                          )}

                          {onUpdateStatus && !gu.is_vital && (
                            <>
                              {status === "ACTIVE" && (
                                <>
                                  <button
                                    type="button"
                                    disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                    onClick={() => handleStatusUpdate(gu.instance_id, "SLEEPING")}
                                    className="text-[8px] py-0.8 px-1.5 bg-zinc-950/40 text-amber-500 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer active:scale-[0.98]"
                                  >
                                    {statusUpdatingId === gu.instance_id ? "..." : "💤 Ngủ Đông"}
                                  </button>
                                  <button
                                    type="button"
                                    disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                    onClick={() => handleStatusUpdate(gu.instance_id, "SEALED")}
                                    className="text-[8px] py-0.8 px-1.5 bg-zinc-950/40 text-cyan-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer active:scale-[0.98]"
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
                                    className="text-[8px] py-0.8 px-1.5 bg-zinc-950/40 text-emerald-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer active:scale-[0.98]"
                                  >
                                    {statusUpdatingId === gu.instance_id ? "..." : "☀️ Thức Tỉnh"}
                                  </button>
                                  <button
                                    type="button"
                                    disabled={actionLoading || statusUpdatingId === gu.instance_id}
                                    onClick={() => handleStatusUpdate(gu.instance_id, "SEALED")}
                                    className="text-[8px] py-0.8 px-1.5 bg-zinc-950/40 text-cyan-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer active:scale-[0.98]"
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
                                  className="text-[8px] py-0.8 px-1.5 bg-zinc-950/40 text-emerald-400 border border-zinc-900 hover:bg-zinc-900 rounded font-mono font-medium transition-all cursor-pointer active:scale-[0.98]"
                                >
                                  {statusUpdatingId === gu.instance_id ? "..." : "🔓 Giải Ấn"}
                                </button>
                              )}
                            </>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                );
              };

              return (
                <div className="space-y-6">
                  {/* PHẦN 1: BẢN MỆNH CỔ */}
                  {vitalGus.length > 0 && (
                    <div className="space-y-2">
                      <div className="text-[9px] font-mono font-bold text-amber-400/80 uppercase tracking-wider flex items-center gap-1">
                        <span>✦ Bản Mệnh Cổ</span>
                        <div className="h-px flex-1 bg-gradient-to-r flow from-amber-500/20 to-transparent ml-2"></div>
                      </div>
                      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:gap-4">
                        {vitalGus.map(renderGuCard)}
                      </div>
                    </div>
                  )}

                  {/* PHẦN 2: CỔ TRÙNG KHÁC */}
                  {normalGus.length > 0 && (
                    <div className="space-y-2">
                      {vitalGus.length > 0 && (
                        <div className="text-[9px] font-mono font-bold text-zinc-500 uppercase tracking-wider flex items-center gap-1 pt-2">
                          <span>✦ Cổ Trùng Khác</span>
                          <div className="h-px flex-1 bg-gradient-to-r from-zinc-800 to-transparent ml-2"></div>
                        </div>
                      )}
                      <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-3 sm:gap-4">
                        {normalGus.map(renderGuCard)}
                      </div>
                    </div>
                  )}
                </div>
              );
            })()
          )}
        </div>
      )}
    </div>
  );
};
