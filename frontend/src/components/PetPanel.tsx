"use client";

import React, { useEffect, useState } from "react";
import { GamePanel, GameButton, GameCard, GameBadge } from "@/components/ui";
import { Player, PlayerPet, ItemSpec, MonsterSpec, useGameStore } from "@/store/useGameStore";
import { restoreHunger } from "@/lib/effect-blocks";
import { MonsterSprite } from "@/components/MonsterSprite";
import { getItemIconUrl } from "@/lib/item-icons";
import {
  PawPrint,
  Heart,
  Swords,
  Moon,
  Trash2,
  Merge,
  Users,
  ChevronDown,
  ChevronUp,
  LayoutGrid,
  List,
  X,
  Sparkles,
  Shield,
  Zap,
  Info,
  Check,
} from "lucide-react";
// UI migrated to shared components

interface PetPanelProps {
  player: Player;
  pets: PlayerPet[];
  itemDictionary: Record<string, ItemSpec>;
  monsterDictionary: Record<string, MonsterSpec>;
  actionLoading: boolean;
}

const hungerColor = (hunger: number, max: number) => {
  const pct = hunger / max;
  if (pct > 0.6) return "bg-emerald-500";
  if (pct > 0.3) return "bg-amber-500";
  return "bg-red-500";
};

const hungerLabel = (hunger: number, max: number) => {
  const pct = Math.round((hunger / max) * 100);
  return `${Math.round(hunger)}/${Math.round(max)} (${pct}%)`;
};

const getPetRankBorder = (rank: number) => {
  if (rank >= 6) return "border-purple-500/60 shadow-[0_0_15px_rgba(168,85,247,0.25)] bg-gradient-to-b from-purple-950/30 to-zinc-950";
  if (rank === 5) return "border-rose-500/50 shadow-[0_0_12px_rgba(244,63,94,0.2)] bg-gradient-to-b from-rose-950/30 to-zinc-950";
  if (rank === 4) return "border-amber-500/50 shadow-[0_0_10px_rgba(245,158,11,0.2)] bg-gradient-to-b from-amber-950/30 to-zinc-950";
  if (rank === 3) return "border-cyan-500/50 shadow-[0_0_8px_rgba(6,182,212,0.15)] bg-gradient-to-b from-cyan-950/30 to-zinc-950";
  if (rank === 2) return "border-emerald-500/50 shadow-[0_0_8px_rgba(16,185,129,0.15)] bg-gradient-to-b from-emerald-950/30 to-zinc-950";
  return "border-zinc-800 bg-zinc-900/60";
};

export const PetPanel: React.FC<PetPanelProps> = ({
  player,
  pets,
  itemDictionary,
  monsterDictionary,
  actionLoading,
}) => {
  const fetchPlayerPets = useGameStore((s) => s.fetchPlayerPets);
  const feedPet = useGameStore((s) => s.feedPet);
  const setPetFormation = useGameStore((s) => s.setPetFormation);
  const setPetSleep = useGameStore((s) => s.setPetSleep);
  const releasePet = useGameStore((s) => s.releasePet);
  const mergePets = useGameStore((s) => s.mergePets);
  const addToast = useGameStore((s) => s.addToast);
  const showConfirm = useGameStore((s) => s.showConfirm);

  const [feedTarget, setFeedTarget] = useState<string | null>(null);
  const [mergeMode, setMergeMode] = useState(false);
  const [mergeTarget, setMergeTarget] = useState<string | null>(null);
  const [mergeSources, setMergeSources] = useState<string[]>([]);
  const [formationDraft, setFormationDraft] = useState<string[]>([]);
  const [editingFormation, setEditingFormation] = useState(false);
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const [inspectedPetId, setInspectedPetId] = useState<string | null>(null);

  const inspectedPet = pets.find((p) => p.id === inspectedPetId) || null;

  useEffect(() => { void fetchPlayerPets(); }, [fetchPlayerPets]);

  const formationPets = pets.filter((p) => p.formation_slot > 0).sort((a, b) => a.formation_slot - b.formation_slot);
  const storagePets = pets.filter((p) => p.formation_slot === 0);

  const maxSlots = Math.min(1 + player.rank, 8);
  const mergeLimit = Math.max(1, Math.floor((player.max_soul || 0) / 10));
  const formationSoulCost = formationPets.reduce((sum, p) => sum + (p.count || 1), 0);
  const mergeTargetCount = mergeTarget ? pets.find((p) => p.id === mergeTarget)?.count || 1 : 0;
  const mergeSourceTotal = mergeSources.reduce((sum, id) => sum + (pets.find((p) => p.id === id)?.count || 0), 0);
  const mergeTotalCount = mergeTarget ? mergeTargetCount + mergeSourceTotal : 0;
  const mergeOverLimit = !!(mergeMode && mergeTarget && mergeSources.length > 0 && mergeTotalCount > mergeLimit);

  const hungerValue = (item: ItemSpec): number => {
    const restored = restoreHunger(item);
    if (restored > 0) return restored;
    return Math.max(item.rank ?? 1, 1) * 10;
  };

  const foodItems = Object.entries(itemDictionary).filter(([, item]) => {
    return item.item_type === "Food";
  });

  const handleFeed = async (petId: string, itemId: string) => {
    await feedPet(petId, itemId);
    setFeedTarget(null);
  };

  const handleAutoFeed = async (petId: string) => {
    const ownedFoods = foodItems
      .map(([itemId, item]) => ({ itemId, item }))
      .filter(({ itemId }) => (player.materials_inventory?.[itemId] || 0) > 0)
      .sort((a, b) => (a.item.rank ?? 1) - (b.item.rank ?? 1));
    if (ownedFoods.length === 0) {
      addToast("Đạo hữu không có thức ăn để cho nô thú ăn.", "error");
      return;
    }
    await handleFeed(petId, ownedFoods[0].itemId);
  };

  const handleFormationSave = async () => {
    await setPetFormation(formationDraft);
    setEditingFormation(false);
  };

  const handleMerge = async () => {
    if (!mergeTarget || mergeSources.length === 0) return;
    const confirmed = await showConfirm(`Gộp ${mergeSourceTotal} cá thể cùng loại vào một bầy?`, { confirmText: "Gộp" });
    if (confirmed) {
      await mergePets(mergeTarget, mergeSources);
      setMergeMode(false);
      setMergeTarget(null);
      setMergeSources([]);
    }
  };

  const handleRelease = async (petId: string, name: string) => {
    const confirmed = await showConfirm(`Thả ${name}? Hành động này không thể hoàn tác.`, { confirmText: "Thả" });
    if (confirmed) await releasePet(petId);
  };

  if (pets.length === 0) {
    return (
      <GamePanel variant="default" className="text-center">
        <PawPrint className="w-12 h-12 text-zinc-600 mx-auto mb-3" />
        <p className="text-zinc-400 text-sm">Chưa có cổ thú nào. Hãy bắt monster trong chiến đấu!</p>
      </GamePanel>
    );
  }

  return (
    <div className="space-y-4">
      {/* Formation Section */}
      <div className="bg-zinc-900/80 border border-zinc-800 rounded-xl p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-bold text-zinc-300 flex items-center gap-2">
            <Swords className="w-4 h-4 text-amber-400" />
            Đội Hình Thú ({formationPets.length}/{maxSlots})
          </h3>
          <button
            onClick={() => {
              if (editingFormation) {
                setEditingFormation(false);
                setFormationDraft(formationPets.map((p) => p.id));
              } else {
                setFormationDraft(formationPets.map((p) => p.id));
                setEditingFormation(true);
              }
            }}
            className="text-xs text-amber-400 hover:text-amber-300"
          >
            {editingFormation ? "Hủy" : "Sửa"}
          </button>
        </div>

        <div className="text-[10px] text-zinc-500 mb-3">
          Phí Hồn Phách khi ra trận: <span className="text-purple-400 font-bold">{formationSoulCost}</span> / {player.soul_hp ?? 0} soul (mỗi cá thể = 1 soul; Võ Đài miễn phí).
          Slot dùng chung với Dị Nhân (pet + NPC share 1-{maxSlots}).
        </div>

        {editingFormation ? (
          <div className="space-y-2">
<div className="grid grid-cols-4 gap-2">
                {Array.from({ length: maxSlots }).map((_, i) => {
                  const petId = formationDraft[i];
                  const pet = petId ? pets.find((p) => p.id === petId) : null;
                  return (
                    <GameCard
                      key={i}
                      variant={pet ? "selected" : "default"}
                      className="p-2 text-center text-xs cursor-pointer"
                      onClick={() => {
                        if (pet) {
                          setFormationDraft(formationDraft.filter((_, j) => j !== i));
                        }
                      }}
                    >
                      {pet ? (
                        <>
                          <div className="text-amber-300 font-bold truncate">{pet.nickname}</div>
                          <div className="text-zinc-500">x{pet.count}</div>
                        </>
                      ) : (
                        <div className="text-zinc-600">Slot {i + 1}</div>
                      )}
                    </GameCard>
                  );
                })}
              </div>
            {/* Available pets to add */}
            {storagePets.length > 0 && formationDraft.length < maxSlots && (
              <div className="flex flex-wrap gap-1 mt-2">
                {storagePets.map((pet) => (
                  <button
                    key={pet.id}
                    onClick={() => setFormationDraft([...formationDraft, pet.id])}
                    className="text-[10px] px-2 py-1 rounded bg-zinc-800 border border-zinc-700 text-zinc-300 hover:border-amber-600"
                  >
                    {pet.nickname} x{pet.count}
                  </button>
                ))}
              </div>
            )}
            <GameButton
              onClick={handleFormationSave}
              variant="warning"
              fullWidth
              loading={actionLoading}
            >
              Lưu Đội Hình
            </GameButton>
          </div>
        ) : (
<div className="grid grid-cols-4 gap-2">
              {Array.from({ length: maxSlots }).map((_, i) => {
                const pet = formationPets[i];
                return (
                  <GameCard
                    key={i}
                    variant={pet ? "selected" : "default"}
                    className="p-2 text-center text-xs"
                  >
                    {pet ? (
                      <>
                        <div className="text-amber-300 font-bold truncate">{pet.nickname}</div>
                        <div className="text-zinc-500">x{pet.count} · HP {Math.round(pet.hp)}</div>
                      </>
                    ) : (
                      <div className="text-zinc-600">Trống</div>
                    )}
                  </GameCard>
                );
              })}
            </div>
        )}
      </div>

      {/* Controls Bar */}
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <div className="flex items-center gap-2">
          <button
            onClick={() => { setMergeMode(!mergeMode); setMergeTarget(null); setMergeSources([]); }}
            className={`text-xs px-3 py-1.5 rounded-lg border transition-all cursor-pointer ${mergeMode ? "border-purple-600 bg-purple-950/50 text-purple-300" : "border-zinc-700 bg-zinc-800 text-zinc-400 hover:border-zinc-600"}`}
          >
            <Merge className="w-3 h-3 inline mr-1" />
            Gộp Bầy
          </button>
          {mergeMode && (
            <span className="text-[10px] text-zinc-500 self-center">
              Bầy tối đa <span className="text-purple-400 font-bold">{mergeLimit}</span> cá thể
            </span>
          )}
          {mergeMode && mergeTarget && mergeSources.length > 0 && (
            <button onClick={handleMerge} disabled={actionLoading || mergeOverLimit} className="text-xs px-3 py-1.5 rounded-lg bg-purple-700 hover:bg-purple-600 text-white font-bold disabled:opacity-50 cursor-pointer">
              Xác Nhận Gộp (+{mergeSourceTotal})
            </button>
          )}
          {mergeOverLimit && (
            <span className="text-[10px] text-red-400 self-center">Vượt giới hạn ({mergeTotalCount}/{mergeLimit})!</span>
          )}
        </div>

        {/* View mode toggle */}
        <div className="flex items-center gap-1 bg-zinc-950/80 p-1 rounded-xl border border-zinc-800">
          <button
            type="button"
            onClick={() => setViewMode("grid")}
            className={`p-1.5 rounded-lg text-xs font-mono transition-all flex items-center gap-1 cursor-pointer ${
              viewMode === "grid"
                ? "bg-amber-500/20 text-amber-300 border border-amber-500/40 shadow-sm"
                : "text-zinc-500 hover:text-zinc-300"
            }`}
            title="Dạng Thẻ Bài RPG"
          >
            <LayoutGrid className="w-3.5 h-3.5" />
            <span className="text-[10px] hidden sm:inline">Lưới RPG</span>
          </button>
          <button
            type="button"
            onClick={() => setViewMode("list")}
            className={`p-1.5 rounded-lg text-xs font-mono transition-all flex items-center gap-1 cursor-pointer ${
              viewMode === "list"
                ? "bg-amber-500/20 text-amber-300 border border-amber-500/40 shadow-sm"
                : "text-zinc-500 hover:text-zinc-300"
            }`}
            title="Dạng Danh Sách"
          >
            <List className="w-3.5 h-3.5" />
            <span className="text-[10px] hidden sm:inline">Danh Sách</span>
          </button>
        </div>
      </div>

      {/* Pet Display: Grid Mode */}
      {viewMode === "grid" ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3">
          {pets.map((pet) => {
            const monSpec = monsterDictionary[pet.monster_template_id];
            const isFormation = pet.formation_slot > 0;
            const isMergeSelected = mergeSources.includes(pet.id);
            const isMergeTargetMode = mergeMode && !mergeTarget;
            const isTarget = mergeTarget === pet.id;
            const rankBorder = getPetRankBorder(pet.rank);

            return (
              <div
                key={pet.id}
                onClick={() => {
                  if (mergeMode) {
                    if (isMergeTargetMode) {
                      setMergeTarget(pet.id);
                    } else if (isTarget) {
                      setMergeTarget(null);
                    } else {
                      if (isMergeSelected) setMergeSources(mergeSources.filter((id) => id !== pet.id));
                      else setMergeSources([...mergeSources, pet.id]);
                    }
                  } else {
                    setInspectedPetId(pet.id);
                  }
                }}
                className={`relative rounded-2xl border p-3 flex flex-col justify-between transition-all duration-200 cursor-pointer select-none group hover:-translate-y-0.5 hover:shadow-xl ${
                  isMergeSelected
                    ? "border-purple-500 bg-purple-950/30 ring-2 ring-purple-500/50"
                    : isTarget
                    ? "border-amber-400 bg-amber-950/40 ring-2 ring-amber-400/60"
                    : isFormation
                    ? "border-amber-500/50 bg-amber-950/15"
                    : rankBorder
                }`}
              >
                {/* Top badges */}
                <div className="flex items-center justify-between gap-1 mb-1">
                  <span className="text-[9px] font-mono font-bold px-1.5 py-0.5 rounded bg-zinc-900/80 text-amber-400 border border-zinc-800">
                    R{pet.rank}{pet.sub_rank > 0 ? `.${pet.sub_rank}` : ""}
                  </span>
                  <div className="flex items-center gap-1">
                    {isFormation && (
                      <span className="text-[9px] px-1 py-0.5 rounded bg-amber-950/60 text-amber-300 border border-amber-800/40 font-mono">
                        Đội hình
                      </span>
                    )}
                    {pet.state === "SLEEPING" && (
                      <span className="p-0.5 rounded bg-blue-950 text-blue-300" title="Đang ngủ">
                        <Moon className="w-3 h-3" />
                      </span>
                    )}
                    {pet.state === "INJURED" && (
                      <span className="text-[9px] px-1 py-0.5 rounded bg-rose-950 text-rose-300 font-mono">
                        Bị thương
                      </span>
                    )}
                  </div>
                </div>

                {/* Monster Sprite Visual */}
                <div className="w-20 h-20 mx-auto my-1 flex items-center justify-center relative">
                  <MonsterSprite
                    templateId={pet.monster_template_id}
                    alt={pet.nickname}
                    className="w-18 h-18 object-contain drop-shadow-[0_4px_10px_rgba(0,0,0,0.6)] group-hover:scale-105 transition-transform"
                  />
                  {pet.count > 1 && (
                    <span className="absolute bottom-0 right-0 text-[10px] font-mono font-bold px-1.5 py-0.2 rounded-full bg-zinc-950/90 text-amber-300 border border-amber-500/40 shadow-sm">
                      x{pet.count}
                    </span>
                  )}
                </div>

                {/* Name & Subtitle */}
                <div className="text-center mt-1">
                  <div className="text-xs font-serif font-bold text-zinc-100 truncate group-hover:text-amber-300 transition-colors">
                    {pet.nickname}
                  </div>
                  <div className="text-[10px] text-zinc-500 font-mono truncate">
                    {monSpec?.name || pet.monster_template_id}
                  </div>
                </div>

                {/* Bottom Bars */}
                <div className="mt-2.5 pt-2 border-t border-zinc-800/60 space-y-1">
                  <div className="flex items-center justify-between text-[9px] font-mono text-zinc-400">
                    <span>HP</span>
                    <span className="text-zinc-300">{Math.round(pet.hp)}/{Math.round(pet.max_hp)}</span>
                  </div>
                  <div className="h-1 bg-zinc-800 rounded-full overflow-hidden">
                    <div
                      className="h-full bg-rose-500 rounded-full transition-all"
                      style={{ width: `${Math.min(100, (pet.hp / Math.max(1, pet.max_hp)) * 100)}%` }}
                    />
                  </div>

                  <div className="flex items-center justify-between text-[9px] font-mono text-zinc-400 mt-1">
                    <span>No</span>
                    <span className="text-zinc-300">{Math.round((pet.hunger / pet.max_hunger) * 100)}%</span>
                  </div>
                  <div className="h-1 bg-zinc-800 rounded-full overflow-hidden">
                    <div
                      className={`h-full ${hungerColor(pet.hunger, pet.max_hunger)} rounded-full transition-all`}
                      style={{ width: `${(pet.hunger / pet.max_hunger) * 100}%` }}
                    />
                  </div>
                </div>

                {/* Merge Mode Indicator */}
                {mergeMode && (
                  <div className="mt-2 pt-1 border-t border-zinc-800 text-center">
                    {isTarget ? (
                      <span className="text-[10px] text-amber-400 font-bold font-mono">BẦY ĐÍCH</span>
                    ) : isMergeSelected ? (
                      <span className="text-[10px] text-purple-300 font-bold font-mono">ĐÃ CHỌN GỘP</span>
                    ) : (
                      <span className="text-[9px] text-zinc-500 font-mono">Bấm để chọn</span>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      ) : (
        /* Pet Display: List Mode */
        <div className="space-y-2">
          {pets.map((pet) => {
            const monSpec = monsterDictionary[pet.monster_template_id];
            const displayName = pet.count > 1 ? `Bầy ${pet.nickname} (${pet.count})` : pet.nickname;
            const isFormation = pet.formation_slot > 0;
            const isMergeSelected = mergeSources.includes(pet.id);
            const isMergeTargetMode = mergeMode && !mergeTarget;

            return (
              <div
                key={pet.id}
                className={`bg-zinc-900/80 border rounded-xl p-3 transition-all ${
                  isMergeSelected ? "border-purple-500 bg-purple-950/20" : isFormation ? "border-amber-700/50" : "border-zinc-800"
                }`}
              >
                <div className="flex items-start gap-3">
                  {/* Sprite */}
                  <div className="w-14 h-14 rounded-xl bg-zinc-950/60 border border-zinc-800 flex items-center justify-center shrink-0">
                    <MonsterSprite templateId={pet.monster_template_id} alt={pet.nickname} className="w-12 h-12 object-contain" />
                  </div>

                  {/* Pet Info */}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-bold text-zinc-200 truncate">{displayName}</span>
                      {isFormation && <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-900/50 text-amber-400">Đội hình</span>}
                      {pet.state === "SLEEPING" && <Moon className="w-3 h-3 text-blue-400" />}
                      {pet.state === "INJURED" && <span className="text-[10px] px-1.5 py-0.5 rounded bg-rose-950/60 text-rose-300">Bị thương</span>}
                    </div>
                    <div className="text-[10px] text-zinc-500 mt-0.5">
                      {monSpec?.name || pet.monster_template_id} · Rank {pet.rank}
                      {pet.sub_rank > 0 ? `.${pet.sub_rank}` : ""}
                    </div>
                    {/* Stats */}
                    <div className="flex gap-3 mt-1 text-[10px] text-zinc-400">
                      <span>HP: {Math.round(pet.hp)}/{Math.round(pet.max_hp)}</span>
                      <span>ATK: {Math.round(pet.base_attack)}</span>
                      <span>DEF: {Math.round(pet.base_defense)}</span>
                    </div>
                    {/* Hunger Bar */}
                    <div className="mt-1.5 flex items-center gap-2">
                      <Heart className="w-3 h-3 text-red-400 flex-shrink-0" />
                      <div className="flex-1 h-1.5 bg-zinc-800 rounded-full overflow-hidden">
                        <div className={`h-full ${hungerColor(pet.hunger, pet.max_hunger)} rounded-full transition-all`} style={{ width: `${(pet.hunger / pet.max_hunger) * 100}%` }} />
                      </div>
                      <span className="text-[10px] text-zinc-500 w-20 text-right">{hungerLabel(pet.hunger, pet.max_hunger)}</span>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex flex-col gap-1 flex-shrink-0">
                    {mergeMode ? (
                      isMergeTargetMode ? (
                        <button onClick={() => setMergeTarget(pet.id)} className="text-[10px] px-2 py-1 rounded bg-purple-700 text-white cursor-pointer">Chọn</button>
                      ) : mergeTarget === pet.id ? (
                        <span className="text-[10px] px-2 py-1 rounded bg-purple-900/50 text-purple-300">Target</span>
                      ) : (
                        <button
                          onClick={() => {
                            if (isMergeSelected) setMergeSources(mergeSources.filter((id) => id !== pet.id));
                            else setMergeSources([...mergeSources, pet.id]);
                          }}
                          className={`text-[10px] px-2 py-1 rounded border cursor-pointer ${isMergeSelected ? "border-purple-500 bg-purple-900/50 text-purple-300" : "border-zinc-700 text-zinc-400"}`}
                        >
                          {isMergeSelected ? "Bỏ" : "Nguồn"}
                        </button>
                      )
                    ) : (
                      <>
                        <button
                          onClick={() => setInspectedPetId(pet.id)}
                          className="text-[10px] px-2.5 py-1 rounded bg-amber-500/20 text-amber-300 border border-amber-500/30 hover:bg-amber-500/30 cursor-pointer"
                        >
                          Chi Tiết
                        </button>
                        <button
                          onClick={() => handleAutoFeed(pet.id)}
                          disabled={actionLoading || pet.hunger >= pet.max_hunger}
                          title={pet.hunger >= pet.max_hunger ? "Nô thú đã no" : "Tự chọn thức ăn rank thấp nhất"}
                          className="text-[10px] px-2 py-1 rounded bg-emerald-600 text-white hover:bg-emerald-500 disabled:opacity-40 cursor-pointer"
                        >
                          Cho Ăn
                        </button>
                        <button onClick={() => setPetSleep(pet.id, pet.state === "ACTIVE")} className="text-[10px] px-2 py-1 rounded bg-zinc-800 text-zinc-400 hover:bg-zinc-700 cursor-pointer">
                          {pet.state === "SLEEPING" ? "Thức" : pet.state === "INJURED" ? "Đưa về kho" : "Ngủ"}
                        </button>
                        <button onClick={() => handleRelease(pet.id, pet.nickname)} className="text-[10px] px-2 py-1 rounded bg-red-900/50 text-red-400 hover:bg-red-800/50 cursor-pointer">
                          Thả
                        </button>
                      </>
                    )}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Pet Inspector Modal */}
      {inspectedPet && (() => {
        const monSpec = monsterDictionary[inspectedPet.monster_template_id];
        const isFormation = inspectedPet.formation_slot > 0;
        const hungerPct = Math.round((inspectedPet.hunger / inspectedPet.max_hunger) * 100);
        const hpPct = Math.min(100, Math.round((inspectedPet.hp / Math.max(1, inspectedPet.max_hp)) * 100));

        return (
          <div
            className="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-4 bg-black/80 backdrop-blur-sm animate-fade-in"
            onClick={() => setInspectedPetId(null)}
          >
            <div
              className="relative w-full max-w-md max-h-[90vh] overflow-y-auto bg-zinc-950 border border-amber-500/35 rounded-3xl p-5 sm:p-6 shadow-[0_0_40px_rgba(245,158,11,0.2)] text-left space-y-4"
              onClick={(e) => e.stopPropagation()}
            >
              {/* Close Button */}
              <button
                type="button"
                onClick={() => setInspectedPetId(null)}
                className="absolute top-4 right-4 text-zinc-500 hover:text-zinc-200 transition-colors p-1.5 rounded-xl hover:bg-zinc-900 cursor-pointer"
                title="Đóng"
              >
                <X className="w-5 h-5" />
              </button>

              {/* Header Visual */}
              <div className="flex flex-col items-center text-center pt-2">
                <div className="w-28 h-28 rounded-2xl bg-gradient-to-br from-amber-500/10 via-zinc-900 to-zinc-950 border-2 border-amber-500/40 flex items-center justify-center p-2 shadow-inner relative">
                  <MonsterSprite
                    templateId={inspectedPet.monster_template_id}
                    alt={inspectedPet.nickname}
                    className="w-24 h-24 object-contain drop-shadow-[0_6px_12px_rgba(0,0,0,0.7)]"
                  />
                  {inspectedPet.count > 1 && (
                    <span className="absolute -bottom-2 -right-2 text-xs font-mono font-bold px-2 py-0.5 rounded-full bg-amber-500 text-zinc-950 shadow-md">
                      Bầy x{inspectedPet.count}
                    </span>
                  )}
                </div>

                <h3 className="text-lg font-serif font-bold text-zinc-100 mt-3">
                  {inspectedPet.nickname}
                </h3>
                <p className="text-xs font-mono text-zinc-400">
                  {monSpec?.name || inspectedPet.monster_template_id}
                </p>

                {/* Badges */}
                <div className="flex items-center gap-1.5 mt-2 flex-wrap justify-center">
                  <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded-md bg-amber-500/20 text-amber-300 border border-amber-500/30">
                    Bậc {inspectedPet.rank}{inspectedPet.sub_rank > 0 ? `.${inspectedPet.sub_rank}` : ""}
                  </span>
                  {isFormation ? (
                    <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded-md bg-emerald-500/20 text-emerald-300 border border-emerald-500/30">
                      Xuất Trận (Slot {inspectedPet.formation_slot})
                    </span>
                  ) : (
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded-md bg-zinc-900 text-zinc-400 border border-zinc-800">
                      Trong Kho
                    </span>
                  )}
                  {inspectedPet.state === "SLEEPING" ? (
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded-md bg-blue-500/20 text-blue-300 border border-blue-500/30 flex items-center gap-1">
                      <Moon className="w-3 h-3" /> Đang Ngủ
                    </span>
                  ) : inspectedPet.state === "INJURED" ? (
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded-md bg-rose-500/20 text-rose-300 border border-rose-500/30">
                      Bị Thương
                    </span>
                  ) : (
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded-md bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                      Sẵn Sàng
                    </span>
                  )}
                </div>
              </div>

              {/* Stats Card */}
              <div className="bg-zinc-900/60 border border-zinc-850 rounded-2xl p-4 space-y-3 font-mono text-xs">
                {/* HP */}
                <div>
                  <div className="flex justify-between items-center text-[11px] mb-1">
                    <span className="text-zinc-400 flex items-center gap-1">
                      <Heart className="w-3.5 h-3.5 text-rose-400" /> Sinh Mệnh (HP)
                    </span>
                    <span className="text-zinc-200 font-bold">
                      {Math.round(inspectedPet.hp)} / {Math.round(inspectedPet.max_hp)} ({hpPct}%)
                    </span>
                  </div>
                  <div className="h-2 bg-zinc-800 rounded-full overflow-hidden">
                    <div className="h-full bg-rose-500 rounded-full transition-all" style={{ width: `${hpPct}%` }} />
                  </div>
                </div>

                {/* Hunger */}
                <div>
                  <div className="flex justify-between items-center text-[11px] mb-1">
                    <span className="text-zinc-400 flex items-center gap-1">
                      <PawPrint className="w-3.5 h-3.5 text-amber-400" /> Độ No
                    </span>
                    <span className="text-zinc-200 font-bold">
                      {hungerLabel(inspectedPet.hunger, inspectedPet.max_hunger)}
                    </span>
                  </div>
                  <div className="h-2 bg-zinc-800 rounded-full overflow-hidden">
                    <div
                      className={`h-full ${hungerColor(inspectedPet.hunger, inspectedPet.max_hunger)} rounded-full transition-all`}
                      style={{ width: `${hungerPct}%` }}
                    />
                  </div>
                </div>

                {/* Attributes */}
                <div className="grid grid-cols-3 gap-2 pt-2 border-t border-zinc-800 text-center">
                  <div className="bg-zinc-950/60 p-2 rounded-xl border border-zinc-850">
                    <div className="text-[10px] text-zinc-500">Công Kích</div>
                    <div className="text-sm font-bold text-amber-300 mt-0.5">
                      {Math.round(inspectedPet.base_attack)}
                    </div>
                  </div>
                  <div className="bg-zinc-950/60 p-2 rounded-xl border border-zinc-855">
                    <div className="text-[10px] text-zinc-500">Phòng Ngự</div>
                    <div className="text-sm font-bold text-cyan-300 mt-0.5">
                      {Math.round(inspectedPet.base_defense)}
                    </div>
                  </div>
                  <div className="bg-zinc-950/60 p-2 rounded-xl border border-zinc-855">
                    <div className="text-[10px] text-zinc-500">Phí Hồn Phách</div>
                    <div className="text-sm font-bold text-purple-300 mt-0.5">
                      {inspectedPet.count} soul
                    </div>
                  </div>
                </div>
              </div>

              {/* Feed Drawer */}
              {feedTarget === inspectedPet.id && (
                <div className="bg-zinc-900/90 border border-emerald-500/30 rounded-2xl p-3.5 space-y-2 animate-in fade-in">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-serif font-bold text-emerald-300">Chọn Thức Ăn Cho Nô Thú</span>
                    <button
                      type="button"
                      onClick={() => setFeedTarget(null)}
                      className="text-[10px] text-zinc-400 hover:text-zinc-200"
                    >
                      Đóng
                    </button>
                  </div>
                  <div className="grid grid-cols-2 gap-1.5 max-h-40 overflow-y-auto pr-1">
                    {foodItems.map(([itemId, item]) => {
                      const owned = player.materials_inventory?.[itemId] || 0;
                      return (
                        <button
                          key={itemId}
                          onClick={() => handleFeed(inspectedPet.id, itemId)}
                          disabled={owned <= 0 || actionLoading}
                          className="flex items-center gap-2 p-2 rounded-xl bg-zinc-950/80 border border-zinc-800 hover:border-emerald-500/50 text-left transition-colors disabled:opacity-30 cursor-pointer"
                        >
                          <img
                            src={getItemIconUrl(item)}
                            alt=""
                            className="w-7 h-7 rounded-lg object-contain bg-zinc-900 border border-zinc-800 shrink-0"
                          />
                          <div className="min-w-0 flex-1">
                            <div className="text-xs font-mono font-bold text-zinc-200 truncate">{item.name}</div>
                            <div className="text-[10px] font-mono text-emerald-400">+{hungerValue(item)} no (x{owned})</div>
                          </div>
                        </button>
                      );
                    })}
                  </div>
                  {foodItems.length === 0 && (
                    <div className="text-xs text-zinc-500 font-mono text-center py-2">
                      Đạo hữu không có thức ăn nào trong túi đồ.
                    </div>
                  )}
                </div>
              )}

              {/* Actions Grid */}
              <div className="grid grid-cols-2 gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => handleAutoFeed(inspectedPet.id)}
                  disabled={actionLoading || inspectedPet.hunger >= inspectedPet.max_hunger}
                  className="py-2.5 px-3 rounded-xl bg-emerald-600 hover:bg-emerald-500 disabled:bg-zinc-900 disabled:border-zinc-800 disabled:text-zinc-600 text-white font-mono font-bold text-xs transition-all shadow-md active:scale-98 cursor-pointer disabled:cursor-not-allowed"
                >
                  Cho Ăn Nhanh
                </button>

                <button
                  type="button"
                  onClick={() => setFeedTarget(feedTarget === inspectedPet.id ? null : inspectedPet.id)}
                  className="py-2.5 px-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 text-zinc-200 font-mono font-bold text-xs transition-all cursor-pointer"
                >
                  {feedTarget === inspectedPet.id ? "Đóng Thức Ăn" : "Chọn Thức Ăn"}
                </button>

                <button
                  type="button"
                  onClick={() => setPetSleep(inspectedPet.id, inspectedPet.state === "ACTIVE")}
                  disabled={actionLoading}
                  className="py-2.5 px-3 rounded-xl bg-zinc-900 hover:bg-zinc-800 border border-zinc-800 text-zinc-300 font-mono font-bold text-xs transition-all cursor-pointer"
                >
                  {inspectedPet.state === "SLEEPING" ? "Đánh Thức" : inspectedPet.state === "INJURED" ? "Đưa Về Kho" : "Cho Ngủ (Hồi Phục)"}
                </button>

                <button
                  type="button"
                  onClick={() => {
                    handleRelease(inspectedPet.id, inspectedPet.nickname);
                    setInspectedPetId(null);
                  }}
                  disabled={actionLoading}
                  className="py-2.5 px-3 rounded-xl bg-red-950/40 hover:bg-red-900/60 border border-red-800/40 text-red-300 font-mono font-bold text-xs transition-all cursor-pointer"
                >
                  Thả Về Tự Nhiên
                </button>
              </div>
            </div>
          </div>
        );
      })()}
    </div>
  );
};
