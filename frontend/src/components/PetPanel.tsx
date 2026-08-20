"use client";

import React, { useEffect, useState } from "react";
import { Player, PlayerPet, ItemSpec, MonsterSpec, useGameStore } from "@/store/useGameStore";
import { PawPrint, Heart, Swords, Moon, Trash2, Merge, Users, ChevronDown, ChevronUp } from "lucide-react";

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
    const effects = item.effects as Record<string, number> | undefined;
    if (effects?.hunger && effects.hunger > 0) return effects.hunger;
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
      <div className="bg-zinc-900/80 border border-zinc-800 rounded-xl p-6 text-center">
        <PawPrint className="w-12 h-12 text-zinc-600 mx-auto mb-3" />
        <p className="text-zinc-400 text-sm">Chưa có cổ thú nào. Hãy bắt monster trong chiến đấu!</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Formation Section */}
      <div className="bg-zinc-900/80 border border-zinc-800 rounded-xl p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-bold text-zinc-300 flex items-center gap-2">
            <Swords className="w-4 h-4 text-amber-400" />
            Đội Hình Ra Trận ({formationPets.length}/{maxSlots})
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
        </div>

        {editingFormation ? (
          <div className="space-y-2">
            <div className="grid grid-cols-4 gap-2">
              {Array.from({ length: maxSlots }).map((_, i) => {
                const petId = formationDraft[i];
                const pet = petId ? pets.find((p) => p.id === petId) : null;
                return (
                  <div
                    key={i}
                    className={`border rounded-lg p-2 text-center text-xs cursor-pointer transition-all ${
                      pet ? "border-amber-700 bg-amber-950/30" : "border-zinc-700 bg-zinc-800/50 border-dashed"
                    }`}
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
                  </div>
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
            <button
              onClick={handleFormationSave}
              disabled={actionLoading}
              className="w-full mt-2 py-2 bg-amber-700 hover:bg-amber-600 text-white text-xs font-bold rounded-lg disabled:opacity-50"
            >
              Lưu Đội Hình
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-4 gap-2">
            {Array.from({ length: maxSlots }).map((_, i) => {
              const pet = formationPets[i];
              return (
                <div key={i} className={`border rounded-lg p-2 text-center text-xs ${pet ? "border-amber-700 bg-amber-950/30" : "border-zinc-700 bg-zinc-800/50 border-dashed"}`}>
                  {pet ? (
                    <>
                      <div className="text-amber-300 font-bold truncate">{pet.nickname}</div>
                      <div className="text-zinc-500">x{pet.count} · HP {Math.round(pet.hp)}</div>
                    </>
                  ) : (
                    <div className="text-zinc-600">Trống</div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Merge Mode Toggle */}
      <div className="flex gap-2">
        <button
          onClick={() => { setMergeMode(!mergeMode); setMergeTarget(null); setMergeSources([]); }}
          className={`text-xs px-3 py-1.5 rounded-lg border transition-all ${mergeMode ? "border-purple-600 bg-purple-950/50 text-purple-300" : "border-zinc-700 bg-zinc-800 text-zinc-400 hover:border-zinc-600"}`}
        >
          <Merge className="w-3 h-3 inline mr-1" />
          Gộp Bầy
        </button>
        {mergeMode && (
          <span className="text-[10px] text-zinc-500 self-center">
            Bầy tối đa <span className="text-purple-400 font-bold">{mergeLimit}</span> cá thể (max_soul/10 = {player.max_soul ?? 0}/10)
          </span>
        )}
        {mergeMode && mergeTarget && mergeSources.length > 0 && (
          <button onClick={handleMerge} disabled={actionLoading || mergeOverLimit} className="text-xs px-3 py-1.5 rounded-lg bg-purple-700 hover:bg-purple-600 text-white font-bold disabled:opacity-50">
            Xác Nhận Gộp (+{mergeSourceTotal} cá thể)
          </button>
        )}
        {mergeOverLimit && (
          <span className="text-[10px] text-red-400 self-center">Vượt giới hạn bầy ({mergeTotalCount}/{mergeLimit})!</span>
        )}
      </div>

      {/* Pet List */}
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
                {/* Pet Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-bold text-zinc-200 truncate">{displayName}</span>
                    {isFormation && <span className="text-[10px] px-1.5 py-0.5 rounded bg-amber-900/50 text-amber-400">Đội hình</span>}
                    {pet.state === "SLEEPING" && <Moon className="w-3 h-3 text-blue-400" />}
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
                      <button onClick={() => setMergeTarget(pet.id)} className="text-[10px] px-2 py-1 rounded bg-purple-700 text-white">Chọn</button>
                    ) : mergeTarget === pet.id ? (
                      <span className="text-[10px] px-2 py-1 rounded bg-purple-900/50 text-purple-300">Target</span>
                    ) : (
                      <button
                        onClick={() => {
                          if (isMergeSelected) setMergeSources(mergeSources.filter((id) => id !== pet.id));
                          else setMergeSources([...mergeSources, pet.id]);
                        }}
                        className={`text-[10px] px-2 py-1 rounded border ${isMergeSelected ? "border-purple-500 bg-purple-900/50 text-purple-300" : "border-zinc-700 text-zinc-400"}`}
                      >
                        {isMergeSelected ? "Bỏ" : "Nguồn"}
                      </button>
                    )
                  ) : (
                    <>
                      <button  onClick={() => setFeedTarget(feedTarget === pet.id ? null : pet.id)} className="hidden text-[10px] px-2 py-1 rounded bg-emerald-800 text-emerald-300 hover:bg-emerald-700">
                        Cho Ăn
                      </button>
                      <button
                        onClick={() => handleAutoFeed(pet.id)}
                        disabled={actionLoading || pet.hunger >= pet.max_hunger}
                        title={pet.hunger >= pet.max_hunger ? "Nô thú đã no" : "Tự chọn thức ăn rank thấp nhất"}
                        className="text-[10px] px-2 py-1 rounded bg-emerald-600 text-white hover:bg-emerald-500 disabled:opacity-40"
                      >
                        Cho Ăn
                      </button>
                      <button onClick={() => setPetSleep(pet.id, pet.state !== "SLEEPING")} className="text-[10px] px-2 py-1 rounded bg-zinc-800 text-zinc-400 hover:bg-zinc-700">
                        {pet.state === "SLEEPING" ? "Thức" : "Ngủ"}
                      </button>
                      <button onClick={() => handleRelease(pet.id, pet.nickname)} className="text-[10px] px-2 py-1 rounded bg-red-900/50 text-red-400 hover:bg-red-800/50">
                        Thả
                      </button>
                    </>
                  )}
                </div>
              </div>

              {/* Feed Dropdown */}
              {feedTarget === pet.id && (
                <div className="mt-2 pt-2 border-t border-zinc-800">
                  <div className="text-[10px] text-zinc-500 mb-1">Chọn thức ăn:</div>
                  <div className="flex flex-wrap gap-1">
                    {foodItems.map(([itemId, item]) => {
                      const owned = player.materials_inventory?.[itemId] || 0;
                      return (
                        <button
                          key={itemId}
                          onClick={() => handleFeed(pet.id, itemId)}
                          disabled={owned <= 0 || actionLoading}
                          className="text-[10px] px-2 py-1 rounded bg-zinc-800 border border-zinc-700 text-zinc-300 hover:border-emerald-600 disabled:opacity-30"
                        >
                          {item.name} +{hungerValue(item)} ({owned})
                        </button>
                      );
                    })}
                    {foodItems.length === 0 && <span className="text-[10px] text-zinc-600">Không có thức ăn</span>}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};
