"use client";

import { useMemo, useEffect } from "react";
import { guSpecFor, useGameStore, type GuInstance, type GuSpec } from "@/store/useGameStore";
import { operationsBy, refinementTier } from "@/lib/effect-blocks";
import {
  COMBAT_QUICKBAR_EFFECT_TYPES,
  classifyGuQuickbarEffect,
  isFlyingStatus,
  type GuQuickbarEffectType,
} from "@/lib/gu-quickbar";
import { formatManaCost, guManaCost, portalManaCost } from "@/lib/gu-mana";
import { useWorldStore } from "@/store/useWorldStore";
import { Sparkles, Heart, Shield, Zap, Flame, Compass, Crosshair, Swords, Feather } from "lucide-react";

interface GuSkillSlot {
  gu: GuInstance;
  spec: GuSpec;
  effectType: GuQuickbarEffectType;
  label: string;
  isTargeted: boolean;
  cost: number;
}

export function GuQuickBar() {
  const player = useGameStore((s) => s.player);
  const guDictionary = useGameStore((s) => s.guDictionary);
  const useGuOnMap = useWorldStore((s) => s.useGuOnMap);
  const targetingGu = useWorldStore((s) => s.targetingGu);
  const selectedTarget = useWorldStore((s) => s.selectedTarget);
  const setTargetingGu = useWorldStore((s) => s.setTargetingGu);

  // Filter equipped Gu that have usable active effects on World Map
  const usableSlots = useMemo<GuSkillSlot[]>(() => {
    if (!player?.gu_inventory) return [];

    // Highest REFINE_MANA_TIER among equipped same-rank active Gu (energy.go).
    let refineManaTier = 0;
    for (const gu of player.gu_inventory) {
      if (!gu.equipped || gu.status !== "ACTIVE" || gu.rank !== player.rank) continue;
      const equippedSpec = guSpecFor(gu.gu_id, gu.rank);
      if (!equippedSpec) continue;
      const tier = refinementTier(equippedSpec.effects);
      if (tier > refineManaTier) refineManaTier = tier;
    }

    const slots: GuSkillSlot[] = [];
    for (const gu of player.gu_inventory) {
      if (!gu.equipped || gu.status !== "ACTIVE") continue;

      const spec = guSpecFor(gu.gu_id, gu.rank);
      if (!spec) continue;

      const effect = classifyGuQuickbarEffect(spec.effects);
      if (!effect) continue;

      const castable = operationsBy(spec.effects, { trigger: ["ON_ACTION", "ON_USE"] }).length > 0;
      if (!COMBAT_QUICKBAR_EFFECT_TYPES.has(effect.type) && (castable || effect.type === "teleport")) {
        const cost =
          effect.type === "teleport"
            ? portalManaCost(spec.base_mana_cost, gu.rank, player.rank)
            : guManaCost(spec.base_mana_cost, gu.rank, player.rank, player.sub_rank, refineManaTier);
        slots.push({
          gu,
          spec,
          effectType: effect.type,
          label: effect.label,
          isTargeted: effect.isTargeted,
          cost,
        });
      }
    }

    // Flying is the most important map utility: keep it visible even when the
    // player has more than five usable Gu equipped.
    return slots
      .sort((a, b) => Number(b.effectType === "fly") - Number(a.effectType === "fly"))
      .slice(0, 5);
  }, [player?.gu_inventory, player?.rank, player?.sub_rank, guDictionary]);

  // World-map flying status (lowercase `phi_hanh`, persisted in player.statuses).
  const playerIsFlying = (() => {
    if (!player?.statuses || !Array.isArray(player.statuses)) return false;
    const now = Date.now();
    return player.statuses.some((st: ApiResponseData) => {
      if (!isFlyingStatus(st.id)) return false;
      return !st.expires_at || new Date(st.expires_at).getTime() > now;
    });
  })();

  // Calculate active cooldowns for Gu
  const getCooldownRemains = (instanceId: string): number => {
    if (!player?.statuses || !Array.isArray(player.statuses)) return 0;
    const cdId = `cd_gu_${instanceId}`;
    const found = player.statuses.find((st: ApiResponseData) => st.id === cdId);
    if (!found || !found.expires_at) return 0;
    const exp = new Date(found.expires_at).getTime();
    const now = Date.now();
    return Math.max(0, Math.ceil((exp - now) / 1000));
  };

  const handleActivateGu = async (slot: GuSkillSlot) => {
    const cd = getCooldownRemains(slot.gu.instance_id);
    if (cd > 0) return;

    if (slot.effectType === "teleport" && player && slot.gu.rank < player.rank) {
      useGameStore.getState().addToast(`Không thể dịch chuyển: Cổ trùng ${slot.spec.name} (Rank ${slot.gu.rank}) thấp hơn tu vi bản thân (Rank ${player.rank})!`, "error");
      return;
    }

    if (slot.isTargeted) {
      if (slot.effectType === "damage" && selectedTarget?.instanceId) {
        // Keep cast mode armed if the moving monster produces a transient
        // conflict/error. useGuOnMap clears it only after a successful cast.
        setTargetingGu({
          guInstanceId: slot.gu.instance_id,
          guId: slot.gu.gu_id,
          name: slot.spec.name,
          mode: "monster",
        });
        await useGuOnMap(slot.gu.instance_id, Math.floor(selectedTarget.x), Math.floor(selectedTarget.y), selectedTarget.instanceId);
        return;
      }
      if (targetingGu?.guInstanceId === slot.gu.instance_id) {
        setTargetingGu(null);
      } else {
        setTargetingGu({
          guInstanceId: slot.gu.instance_id,
          guId: slot.gu.gu_id,
          name: slot.spec.name,
          mode: slot.effectType === "damage" ? "monster" : "position",
        });
        useGameStore.getState().addToast(
          slot.effectType === "damage"
            ? `Đã chọn ${slot.spec.name}! Hãy nhấp trực tiếp vào quái vật để tung chiêu.`
            : `Đã chọn ${slot.spec.name}! Hãy nhấp vị trí trên bản đồ để dịch chuyển.`,
          "info",
        );
      }
    } else {
      setTargetingGu(null);
      await useGuOnMap(slot.gu.instance_id, player?.x || 0, player?.y || 0);
    }
  };

  // Listen to numeric keypress 1..5 for hotkeys
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ignore if user is typing in input/textarea
      if (["INPUT", "TEXTAREA", "SELECT"].includes((e.target as HTMLElement)?.tagName)) {
        return;
      }
      const keyNum = parseInt(e.key, 10);
      if (keyNum >= 1 && keyNum <= 5) {
        const slot = usableSlots[keyNum - 1];
        if (slot) {
          handleActivateGu(slot);
        }
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [usableSlots, player, targetingGu, selectedTarget]);

  if (usableSlots.length === 0) {
    return null;
  }

  const getSlotIcon = (type: GuSkillSlot["effectType"]) => {
    const iconCls = "w-3.5 h-3.5 sm:w-4 sm:h-4";
    switch (type) {
      case "heal":
        return <Heart className={`${iconCls} text-emerald-400`} />;
      case "mana_regen":
        return <Sparkles className={`${iconCls} text-sky-400`} />;
      case "buff_atk":
        return <Flame className={`${iconCls} text-red-400`} />;
      case "buff_def":
        return <Shield className={`${iconCls} text-blue-400`} />;
      case "buff_speed":
        return <Zap className={`${iconCls} text-amber-400`} />;
      case "shield":
        return <Shield className={`${iconCls} text-purple-400`} />;
      case "teleport":
        return <Compass className={`${iconCls} text-cyan-400`} />;
      case "fly":
        return <Feather className={`${iconCls} ${playerIsFlying ? "text-cyan-300" : "text-sky-400"}`} />;
      case "damage":
        return <Swords className={`${iconCls} text-red-400`} />;
      default:
        return <Sparkles className={`${iconCls} text-amber-400`} />;
    }
  };

  return (
    <div className="absolute bottom-2 sm:bottom-4 left-1/2 -translate-x-1/2 z-30 pointer-events-auto flex flex-col items-center gap-1 sm:gap-1.5 select-none max-w-[95vw]">
      {targetingGu && (
        <div className="px-2.5 sm:px-3 py-0.5 sm:py-1 bg-amber-950/90 border border-amber-500/50 rounded-full text-[10px] sm:text-xs text-amber-300 font-serif shadow-lg animate-pulse flex items-center gap-1 sm:gap-1.5 max-w-[90vw] text-center">
          <Crosshair className="w-3 h-3 sm:w-3.5 sm:h-3.5 shrink-0" />
          <span className="truncate">
            {targetingGu.mode === "monster" ? "Nhấp quái vật để tung" : "Đang chọn tọa độ cho"}{" "}
            <strong>{targetingGu.name}</strong> (Nhấn phím lại để hủy)
          </span>
        </div>
      )}

      <div className="flex items-center gap-1 sm:gap-2 p-1 sm:p-1.5 bg-zinc-950/90 backdrop-blur-md rounded-xl sm:rounded-2xl border border-zinc-800 shadow-2xl">
        {usableSlots.map((slot, index) => {
          const cd = getCooldownRemains(slot.gu.instance_id);
          const isActiveTargeting = targetingGu?.guInstanceId === slot.gu.instance_id;
          const flyingActive = slot.effectType === "fly" && playerIsFlying;
          const displayLabel = flyingActive ? "Hạ Cánh" : slot.label;

          return (
            <button
              key={slot.gu.instance_id}
              onClick={() => handleActivateGu(slot)}
              disabled={cd > 0}
              className={`relative group flex flex-col items-center justify-between w-10 h-12.5 sm:w-14 sm:h-16 p-1 sm:p-1.5 rounded-lg sm:rounded-xl border transition-all cursor-pointer ${
                isActiveTargeting
                  ? "bg-amber-900/40 border-amber-400 shadow-[0_0_8px_rgba(245,158,11,0.4)] sm:shadow-[0_0_12px_rgba(245,158,11,0.4)]"
                  : flyingActive
                  ? "bg-cyan-900/40 border-cyan-400 shadow-[0_0_8px_rgba(34,211,238,0.35)] sm:shadow-[0_0_12px_rgba(34,211,238,0.35)]"
                  : cd > 0
                  ? "bg-zinc-900/60 border-zinc-800 opacity-60 cursor-not-allowed"
                  : "bg-zinc-900/90 border-zinc-700/60 hover:border-emerald-500/60 hover:bg-zinc-800/90"
              }`}
              title={`${slot.spec.name} (${displayLabel}) - Bấm [${index + 1}]`}
            >
              {/* Hotkey badge (desktop only) */}
              <span className="hidden sm:flex absolute -top-1.5 -left-1.5 w-4 h-4 rounded-full bg-zinc-800 border border-zinc-600 text-[9px] font-bold text-zinc-300 items-center justify-center shadow">
                {index + 1}
              </span>

              {/* Icon & Rank */}
              <div className="flex items-center justify-center mt-0.5 sm:mt-1">
                {getSlotIcon(slot.effectType)}
              </div>

              {/* Name snippet & cost */}
              <div className="w-full text-center truncate text-[7.5px] sm:text-[9px] font-serif text-zinc-200 mt-0.5 leading-tight">
                {slot.spec.name}
              </div>

              <div className="text-[7px] sm:text-[8px] font-mono text-zinc-400 leading-tight">
                {formatManaCost(slot.cost)} <span className="hidden sm:inline">Mana</span><span className="sm:hidden">M</span>
              </div>

              {/* Cooldown Overlay */}
              {cd > 0 && (
                <div className="absolute inset-0 bg-black/75 rounded-lg sm:rounded-xl flex items-center justify-center font-bold text-amber-400 text-xs sm:text-sm font-mono backdrop-blur-[1px]">
                  {cd}s
                </div>
              )}
            </button>
          );
        })}
      </div>
    </div>
  );
}
