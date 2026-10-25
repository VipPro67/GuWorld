"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import dynamic from "next/dynamic";
import type Phaser from "phaser";
import { WorldRuntime, type WorldRuntimeStatus, type RuntimeEntity } from "@/game/core/WorldRuntime";
import { canJoinMMOZone, mmoClientConfig } from "@/lib/mmoConfig";
import { MMOErrorBoundary } from "./MMOErrorBoundary";
import { MMOHud, type HudTarget } from "./MMOHud";
import { MMOBusinessPanels, type MMOPanel } from "./MMOBusinessPanels";
import type { WorldSkillInfo } from "@/game/network/Protocol";
import { useGameStore, guSpecFor, type AutoCombatConfig, type Player } from "@/store/useGameStore";
import { useUIStore } from "@/store/useUIStore";

const GlobalChatOverlay = dynamic(
  () => import("@/components/GlobalChatOverlay").then((m) => m.GlobalChatOverlay),
  { ssr: false },
);

type HostStatus = "loading" | "ready" | "error";
type DestroyGame = (host: HTMLElement, game?: Phaser.Game) => boolean;

interface HudData {
  connected: boolean;
  tickRate: number;
  mana: number;
  maxMana: number;
  hp: number;
  maxHp: number;
  shield: number;
  skills: WorldSkillInfo[];
  guSlots: Array<{ slot: number; guId: string | null; name: string | null; rank: number | null; supported: boolean }>;
  cooldownRemaining: number[];
  castName: string;
  castProgress: number | null;
  target: HudTarget | null;
  statusText: string;
  position: { x: number; y: number };
  nearby: RuntimeEntity[];
}

function buildHud(runtime: WorldRuntime, player: Player | null): HudData | null {
  if (runtime.getStatus() !== "joined") return null;
  const skills = runtime.getSkills();
  const guSlots = Array.from({ length: 6 }, (_, index) => {
    const instanceId = player?.equipped_gus[index];
    const gu = instanceId ? player?.gu_inventory.find((item) => item.instance_id === instanceId && item.equipped && item.status === "ACTIVE") : undefined;
    const spec = gu ? guSpecFor(gu.gu_id, gu.rank) : undefined;
    return {
      slot: index + 1,
      guId: gu?.gu_id ?? null,
      name: gu ? spec?.name ?? gu.gu_id : null,
      rank: gu?.rank ?? null,
      supported: skills.some((skill) => skill.slot === index + 1),
    };
  });
  const combat = runtime.getSelfCombat();
  const self = runtime.getSelfEntity();
  const tick = runtime.estimatedTick();
  const tickRate = runtime.getTickRate();
  const cooldownRemaining = new Array(7).fill(0);
  for (let slot = 1; slot <= 6; slot += 1) {
    cooldownRemaining[slot] = Math.max(0, ((combat.cooldowns[slot] ?? 0) - tick) / tickRate);
  }
  const remainingTicks = combat.castEndTick - tick;
  const castProgress =
    combat.castSkillId && combat.castWindupTicks > 0
      ? Math.max(0, Math.min(1, 1 - remainingTicks / combat.castWindupTicks))
      : null;
  const targetId = runtime.getSelectedTargetId();
  const targetEntity = targetId ? runtime.getEntity(targetId) : null;
  return {
    connected: true,
    tickRate,
    mana: combat.mana,
    maxMana: combat.maxMana,
    hp: self?.hp ?? 0,
    maxHp: self?.maxHp ?? 0,
    shield: self?.shield ?? 0,
    skills,
    guSlots,
    cooldownRemaining,
    castName: skills.find((skill) => skill.id === combat.castSkillId)?.name ?? "",
    castProgress,
    target: targetEntity
      ? { id: targetEntity.id, kind: targetEntity.kind, hp: targetEntity.hp, maxHp: targetEntity.maxHp, shield: targetEntity.shield, name: targetEntity.name, templateId: targetEntity.templateId, aiState: targetEntity.aiState }
      : null,
    statusText: runtime.getCombatNotice(),
    position: runtime.getSelfPosition(),
    nearby: runtime.getEntities(),
  };
}

function realtimeControls(settings: AutoCombatConfig | null | undefined) {
  return {
    leftClickSlot: settings?.realtime_left_click_slot ?? 0,
    rightClickAction: settings?.realtime_right_click_action ?? "attack",
    rightClickSlot: settings?.realtime_right_click_slot ?? 0,
    autoFarm: settings?.realtime_auto_farm ?? false,
    farmRange: settings?.realtime_farm_range ?? 14,
    farmStopHPPercent: settings?.realtime_farm_stop_hp_percent ?? 25,
    farmUseSkills: settings?.realtime_farm_use_skills ?? false,
    // Turn-based Gu tactic loadout was retired; auto-farm now follows hotbar order.
    guPriorities: [],
  };
}

interface PhaserWorldHostProps {
  onExit?: () => void;
  worldActive?: boolean;
}

/**
 * Owns the single Phaser canvas and realtime session. Phaser is imported lazily
 * so server rendering never touches `window`.
 */
export function PhaserWorldHost({ onExit, worldActive = true }: PhaserWorldHostProps) {
  const setView = useUIStore((state) => state.setView);
  const player = useGameStore((state) => state.player);
  const isDivinationModalOpen = useGameStore((state) => state.isDivinationModalOpen);
  const savedCombatSettings = useGameStore((state) => state.combatSettings);
  const currentCombatSettings = savedCombatSettings ?? player?.combat_settings ?? null;
  const playerRef = useRef(player);
  useEffect(() => {
    playerRef.current = player;
  }, [player]);
  const hostRef = useRef<HTMLDivElement | null>(null);
  const gameRef = useRef<Phaser.Game | null>(null);
  const destroyRef = useRef<DestroyGame | null>(null);
  const runtimeRef = useRef<WorldRuntime | null>(null);
  const onExitRef = useRef(onExit);
  useEffect(() => {
    onExitRef.current = onExit;
  }, [onExit]);

  const [status, setStatus] = useState<HostStatus>("loading");
  const [runtimeStatus, setRuntimeStatus] = useState<WorldRuntimeStatus>("idle");
  const [zoneId, setZoneId] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [hud, setHud] = useState<HudData | null>(null);
  const [panel, setPanel] = useState<MMOPanel>(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const [attempt, setAttempt] = useState(0);

  const retry = useCallback(() => {
    setStatus("loading");
    setErrorMessage("");
    setPanel(null);
    setMenuOpen(false);
    setAttempt((value) => value + 1);
  }, []);

  useEffect(() => {
    runtimeRef.current?.setUiModal(panel !== null || menuOpen || isDivinationModalOpen || !worldActive);
    return () => {
      runtimeRef.current?.setUiModal(false);
    };
  }, [panel, menuOpen, isDivinationModalOpen, worldActive]);

  useEffect(() => {
    if (!menuOpen) return;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setMenuOpen(false);
    };
    window.addEventListener("keydown", closeOnEscape);
    return () => window.removeEventListener("keydown", closeOnEscape);
  }, [menuOpen]);

  useEffect(() => {
    if (runtimeStatus === "error" || runtimeStatus === "closed") setPanel(null);
  }, [runtimeStatus]);

  useEffect(() => {
    if (!worldActive) setMenuOpen(false);
  }, [worldActive]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;

    const runtime = canJoinMMOZone ? new WorldRuntime() : null;
    runtime?.setUiModal(panel !== null || menuOpen || isDivinationModalOpen || !worldActive);
    runtimeRef.current = runtime;
    let hudTimer: number | null = null;
    let lastHudAt = 0;
    const pushHud = () => {
      if (!runtime) return;
      const commit = () => {
        lastHudAt = performance.now();
        setHud(buildHud(runtime, playerRef.current));
      };
      if (performance.now() - lastHudAt >= 100) {
        commit();
      } else if (hudTimer === null) {
        hudTimer = window.setTimeout(() => {
          hudTimer = null;
          if (!cancelled) commit();
        }, 100);
      }
    };
    const runtimeUnsubscribe = runtime?.subscribe(() => {
      setRuntimeStatus(runtime.getStatus());
      setZoneId(runtime.getWelcome()?.zone_id ?? "");
      if (runtime.getStatus() === "error") setErrorMessage(runtime.getError());
      pushHud();
    });
    const errorUnsubscribe = runtime?.onError((error) => {
      setRuntimeStatus("error");
      setErrorMessage(error.message);
    });

    let cancelled = false;
    pushHud();

    void (async () => {
      try {
        const { createPhaserGame, destroyPhaserGame } = await import(
          "@/game/bootstrap/createPhaserGame"
        );
        if (cancelled || !hostRef.current) return;

        destroyRef.current = destroyPhaserGame;
        const game = createPhaserGame(hostRef.current, runtime);
        gameRef.current = game;
        setStatus("ready");
        if (runtime && canJoinMMOZone) {
          const zoneId = playerRef.current?.current_region?.trim() || mmoClientConfig.zoneId;
          if (!zoneId) {
            setRuntimeStatus("error");
            setErrorMessage("Nhân vật chưa có KHU_VUC để vào bản đồ realtime");
            return;
          }
          void runtime.connect({
            zone_id: zoneId,
            map_version: mmoClientConfig.mapVersion,
            content_version: mmoClientConfig.contentVersion,
          });
        }
      } catch (error) {
        if (cancelled) return;
        setErrorMessage(error instanceof Error ? error.message : "Không rõ lỗi khởi tạo Phaser");
        setStatus("error");
      }
    })();

    return () => {
      cancelled = true;
      if (hudTimer !== null) window.clearTimeout(hudTimer);
      hudTimer = null;
      setHud(null);
      runtimeUnsubscribe?.();
      errorUnsubscribe?.();
      runtime?.disconnect();
      runtimeRef.current = null;
      const game = gameRef.current;
      gameRef.current = null;
      const destroy = destroyRef.current;
      destroyRef.current = null;
      if (destroy) {
        destroy(host, game ?? undefined);
      } else if (game) {
        game.destroy(true);
      }
      host.replaceChildren();
    };
  }, [attempt]);

  useEffect(() => {
    runtimeRef.current?.setRealtimeSettings(realtimeControls(currentCombatSettings));
  }, [currentCombatSettings]);

  const reconnect = useCallback(() => {
    const runtime = runtimeRef.current;
    if (!runtime) {
      retry();
      return;
    }
    setErrorMessage("");
    setRuntimeStatus("connecting");
    void runtime.reconnect();
  }, [retry]);

  const showRuntimeError =
    runtimeStatus === "error" && status === "ready" && !!errorMessage;

  return (
    <MMOErrorBoundary>
      <div className="relative h-full w-full overflow-hidden bg-[#07110d]">
        <div ref={hostRef} id="guworld-phaser-host" className="absolute inset-0" />

        {status === "loading" && (
          <div className="pointer-events-none absolute inset-0 grid place-items-center bg-[#07110d]">
            <div className="text-center">
              <div className="mx-auto mb-3 h-10 w-10 animate-spin rounded-full border-2 border-emerald-400/30 border-t-emerald-300" />
              <div className="font-serif text-sm text-emerald-100">
                Đang tải thế giới Phaser…
              </div>
            </div>
          </div>
        )}

        {status === "error" && (
          <div className="absolute inset-0 grid place-items-center bg-[#07110d] p-6">
            <div className="max-w-md text-center">
              <div className="font-serif text-lg text-red-200">
                Không thể khởi động thế giới realtime
              </div>
              <p className="mt-2 text-sm text-zinc-400">{errorMessage}</p>
              <button
                type="button"
                onClick={retry}
                className="mt-4 rounded-lg border border-emerald-400/40 bg-emerald-400/10 px-4 py-2 text-sm text-emerald-100 transition hover:bg-emerald-400/20"
              >
                Thử lại
              </button>
            </div>
          </div>
        )}

        {hud && (
          <MMOHud
            {...hud}
            player={player}
            zoneId={zoneId}
            activePanel={panel}
            menuOpen={menuOpen}
            onToggleMenu={() => setMenuOpen((open) => !open)}
            onCloseMenu={() => setMenuOpen(false)}
            onOpenPanel={(next) => { setMenuOpen(false); setPanel((current) => current === next ? null : next); }}
            onOpenView={(view) => { setMenuOpen(false); setPanel(null); setView(view); }}
            onCast={(slot) => {
              const game = gameRef.current;
              if (!game) return;
              const scene = game.scene.getScene("WorldScene") as unknown as {
                activateHotbarSlot?: (selectedSlot: number) => void;
              };
              scene.activateHotbarSlot?.(slot);
            }}
            onAttack={() => {
              const game = gameRef.current;
              const scene = game?.scene.getScene("WorldScene") as unknown as { activateHudAction?: () => void } | undefined;
              if (scene?.activateHudAction) scene.activateHudAction();
              else runtimeRef.current?.requestBasicAttack();
            }}
            rightClickAction={currentCombatSettings?.realtime_right_click_action === "skill" ? "skill" : "attack"}
            rightClickSlot={currentCombatSettings?.realtime_right_click_slot ?? 0}
            onClearTarget={() => runtimeRef.current?.setSelectedTarget(null)}
            onSelectTarget={(id) => runtimeRef.current?.setSelectedTarget(id)}
            onMoveJoystick={(vector) => {
              const game = gameRef.current;
              if (!game) return;
              const scene = game.scene.getScene("WorldScene") as unknown as {
                setJoystickVector?: (v: { x: number; y: number }) => void;
              } | undefined;
              scene?.setJoystickVector?.(vector);
            }}
            onZoomIn={() => {
              const game = gameRef.current;
              if (!game) return 1;
              const scene = game.scene.getScene("WorldScene") as unknown as {
                zoomIn?: () => number;
              } | undefined;
              return scene?.zoomIn?.() ?? 1;
            }}
            onZoomOut={() => {
              const game = gameRef.current;
              if (!game) return 1;
              const scene = game.scene.getScene("WorldScene") as unknown as {
                zoomOut?: () => number;
              } | undefined;
              return scene?.zoomOut?.() ?? 1;
            }}
          />
        )}

        {panel && (
          <MMOBusinessPanels
            panel={panel}
            onClose={() => setPanel(null)}
            onLoadoutSaved={() => {
              setPanel(null);
              void runtimeRef.current?.reconnect();
            }}
          />
        )}

        {hud && panel === null && worldActive && <GlobalChatOverlay placement="mmo" />}

        {!hud && <div className="pointer-events-none absolute left-3 top-3 z-10 flex items-center gap-2 rounded-full border border-white/10 bg-black/55 px-3 py-1.5 text-[11px] uppercase tracking-[.18em] text-emerald-200/80 backdrop-blur">
          <span
            className={`h-2 w-2 rounded-full ${
              runtimeStatus === "joined"
                ? "bg-emerald-400"
                : runtimeStatus === "error"
                  ? "bg-red-400"
                  : "animate-pulse bg-amber-300"
            }`}
          />
          {runtimeStatus === "joined"
            ? `MMO · ${zoneId || "joined"}`
            : canJoinMMOZone
              ? `MMO · ${runtimeStatus}`
              : "MMO Preview · chưa cấu hình zone"}
        </div>}

        {showRuntimeError && (
          <div className="absolute left-1/2 top-16 z-20 w-[min(92vw,520px)] -translate-x-1/2 rounded-xl border border-red-300/25 bg-red-950/85 p-4 text-sm text-red-100 shadow-xl backdrop-blur">
            <div className="font-semibold">Không thể vào world realtime</div>
            <p className="mt-1 break-words text-red-200/80">{errorMessage}</p>
            <div className="mt-3 flex gap-2">
              <button
                type="button"
                onClick={reconnect}
                className="rounded-lg border border-red-200/30 bg-red-200/10 px-3 py-1.5 text-xs text-red-50 transition hover:bg-red-200/20"
              >
                Kết nối lại
              </button>
            </div>
          </div>
        )}

        {onExit && !hud && (
          <button
            type="button"
            onClick={() => onExitRef.current?.()}
            className="absolute right-3 top-3 z-10 rounded-lg border border-white/15 bg-black/55 px-3 py-1.5 text-xs text-emerald-100 backdrop-blur transition hover:bg-black/75"
          >
            Bảng nhân vật
          </button>
        )}
      </div>
    </MMOErrorBoundary>
  );
}
