"use client";

import { useMemo, useEffect } from "react";
import { guSpecFor, useGameStore, type GuInstance, type GuSpec } from "@/store/useGameStore";
import { canCreatePortal, operationsBy, usesRandomPortal } from "@/lib/effect-blocks";
import { useWorldStore } from "@/store/useWorldStore";
import { Sparkles, Heart, Shield, Zap, Flame, Compass, Crosshair, Swords } from "lucide-react";

interface GuSkillSlot {
  gu: GuInstance;
  spec: GuSpec;
  effectType: "damage" | "heal" | "mana_regen" | "buff_atk" | "buff_def" | "buff_speed" | "shield" | "teleport" | "utility";
  label: string;
  isTargeted: boolean;
  cost: number;
}

// Combat-only effects are cast from the combat screen, not the world-map bar.
// Keep only special utility effects (heal, mana, teleport, flying, ...).
const COMBAT_EFFECT_TYPES = new Set<GuSkillSlot["effectType"]>(["damage", "shield", "buff_atk", "buff_def", "buff_speed"]);

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

    const slots: GuSkillSlot[] = [];
    for (const gu of player.gu_inventory) {
      if (!gu.equipped || gu.status !== "ACTIVE") continue;

      const spec = guSpecFor(gu.gu_id, gu.rank);
      if (!spec) continue;

      const operations = operationsBy(spec.effects, { trigger: ["ON_ACTION", "ON_USE", "CAPABILITY"] });
      let type: GuSkillSlot["effectType"] | null = null;
      let label = "Kích hoạt";
      let isTargeted = false;
      const first = operations[0];
      const damage = operations.find((operation) => operation.op === "DAMAGE");
      const status = operations.find((operation) => operation.op === "STATUS");
      if (canCreatePortal(spec.effects) || usesRandomPortal(spec.effects)) {
        type = "teleport";
        isTargeted = canCreatePortal(spec.effects);
        label = isTargeted ? "Dịch Chuyển" : "Dịch Chuyển Ngẫu Nhiên";
      } else if (damage) {
        type = "damage";
        isTargeted = true;
        label = `${damage.value || ""} Dmg`.trim();
      } else if (first?.op === "HEAL") {
        type = first.resource === "MANA" ? "mana_regen" : "heal";
        label = `${first.resource === "MANA" ? "+" : "Hồi "}${first.value || 0} ${first.resource || "HP"}`;
      } else if (status?.status_id) {
        const id = status.status_id.toUpperCase();
        type = id === "ATK_UP" ? "buff_atk" : id === "DEF_UP" ? "buff_def" : ["HASTE", "SPEED_UP"].includes(id) ? "buff_speed" : id === "REGEN" ? "heal" : "utility";
        label = id === "FLYING" || id === "PHI_HANH" ? "Phi Hành" : `${id} ${status.value || ""}`.trim();
      } else if (first?.op === "SHIELD" || first?.op === "BARRIER") {
        type = "shield"; label = `+${first.value || 0} Khiên`;
      } else if (first) {
        type = "utility"; label = first.op === "DAMAGE" ? `${first.value || 0} Dmg` : "Kích hoạt";
      }

      const castable = operationsBy(spec.effects, { trigger: ["ON_ACTION", "ON_USE"] }).length > 0;
      if (type && !COMBAT_EFFECT_TYPES.has(type) && (castable || type === "teleport")) {
        const cost = spec.base_mana_cost || 10;
        slots.push({
          gu,
          spec,
          effectType: type,
          label,
          isTargeted,
          cost,
        });
      }
    }

    return slots.slice(0, 5); // Limit to top 5 hotbar slots
  }, [player?.gu_inventory, guDictionary]);

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
    switch (type) {
      case "heal":
        return <Heart className="w-4 h-4 text-emerald-400" />;
      case "mana_regen":
        return <Sparkles className="w-4 h-4 text-sky-400" />;
      case "buff_atk":
        return <Flame className="w-4 h-4 text-red-400" />;
      case "buff_def":
        return <Shield className="w-4 h-4 text-blue-400" />;
      case "buff_speed":
        return <Zap className="w-4 h-4 text-amber-400" />;
      case "shield":
        return <Shield className="w-4 h-4 text-purple-400" />;
      case "teleport":
        return <Compass className="w-4 h-4 text-cyan-400" />;
      case "damage":
        return <Swords className="w-4 h-4 text-red-400" />;
      default:
        return <Sparkles className="w-4 h-4 text-amber-400" />;
    }
  };

  return (
    <div className="absolute bottom-4 left-1/2 -translate-x-1/2 z-30 pointer-events-auto flex flex-col items-center gap-1.5 select-none">
      {targetingGu && (
        <div className="px-3 py-1 bg-amber-950/90 border border-amber-500/50 rounded-full text-xs text-amber-300 font-serif shadow-lg animate-pulse flex items-center gap-1.5">
          <Crosshair className="w-3.5 h-3.5" />
          <span>
            {targetingGu.mode === "monster" ? "Nhấp quái vật để tung" : "Đang chọn tọa độ cho"}{" "}
            <strong>{targetingGu.name}</strong> (Nhấn phím lại để hủy)
          </span>
        </div>
      )}

      <div className="flex items-center gap-2 p-1.5 bg-zinc-950/90 backdrop-blur-md rounded-2xl border border-zinc-800 shadow-2xl">
        {usableSlots.map((slot, index) => {
          const cd = getCooldownRemains(slot.gu.instance_id);
          const isActiveTargeting = targetingGu?.guInstanceId === slot.gu.instance_id;

          return (
            <button
              key={slot.gu.instance_id}
              onClick={() => handleActivateGu(slot)}
              disabled={cd > 0}
              className={`relative group flex flex-col items-center justify-between w-14 h-16 p-1.5 rounded-xl border transition-all cursor-pointer ${
                isActiveTargeting
                  ? "bg-amber-900/40 border-amber-400 shadow-[0_0_12px_rgba(245,158,11,0.4)]"
                  : cd > 0
                  ? "bg-zinc-900/60 border-zinc-800 opacity-60 cursor-not-allowed"
                  : "bg-zinc-900/90 border-zinc-700/60 hover:border-emerald-500/60 hover:bg-zinc-800/90"
              }`}
              title={`${slot.spec.name} (${slot.label}) - Bấm [${index + 1}]`}
            >
              {/* Hotkey badge */}
              <span className="absolute -top-1.5 -left-1.5 w-4 h-4 rounded-full bg-zinc-800 border border-zinc-600 text-[9px] font-bold text-zinc-300 flex items-center justify-center shadow">
                {index + 1}
              </span>

              {/* Icon & Rank */}
              <div className="flex items-center justify-center mt-1">
                {getSlotIcon(slot.effectType)}
              </div>

              {/* Name snippet & cost */}
              <div className="w-full text-center truncate text-[9px] font-serif text-zinc-200 mt-0.5">
                {slot.spec.name}
              </div>

              <div className="text-[8px] font-mono text-zinc-400">
                {slot.cost} Mana
              </div>

              {/* Cooldown Overlay */}
              {cd > 0 && (
                <div className="absolute inset-0 bg-black/75 rounded-xl flex items-center justify-center font-bold text-amber-400 text-sm font-mono backdrop-blur-[1px]">
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
