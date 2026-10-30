"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import dynamic from "next/dynamic";
import type Phaser from "phaser";
import { WorldRuntime, type WorldRuntimeStatus, type RuntimeEntity } from "@/game/core/WorldRuntime";
import { canJoinMMOZone, mmoClientConfig, versionsForMMOZone } from "@/lib/mmoConfig";
import { MMOErrorBoundary } from "./MMOErrorBoundary";
import { MMOHud, type HudTarget } from "./MMOHud";
import { MMOBusinessPanels, type MMOPanel } from "./MMOBusinessPanels";
import type { WorldSkillInfo } from "@/game/network/Protocol";
import { useGameStore, guSpecFor, type AutoCombatConfig, type Player } from "@/store/useGameStore";
import { useUIStore } from "@/store/useUIStore";
import type { MapPortal } from "@/game/world/MapLoader";

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
  mapPortals: MapPortal[];
  aimingSlot: number | null;
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
  const cooldownRemaining = new Array(13).fill(0);
  for (let slot = 1; slot <= 12; slot += 1) {
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
    mapPortals: runtime.getMapPortals(),
    aimingSlot: runtime.getAimingSlot(),
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
    // Auto-farm follows the realtime hotbar, including supported killer moves.
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
  const worldReconnectToken = useGameStore((state) => state.worldReconnectToken);
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
  const recoveringMismatchRef = useRef(false);
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
    if (runtimeStatus === "error") setPanel(null);
  }, [runtimeStatus]);

  useEffect(() => {
    if (runtimeStatus === "error" || status === "error") {
      setHud(null);
    }
  }, [runtimeStatus, status]);

  useEffect(() => {
    if (!worldActive) {
      setMenuOpen(false);
      setPanel(null);
    }
  }, [worldActive]);

  const lastEquippedGusRef = useRef<string>(JSON.stringify(player?.equipped_gus ?? []));
  const activeZoneID = player?.world_zone_id?.trim() || player?.current_region || "";
  const lastRegionRef = useRef<string>(activeZoneID);
  const activeInstanceID = player?.world_instance_id?.trim() || "";
  const lastInstanceRef = useRef<string>(activeInstanceID);
  const lastReconnectTokenRef = useRef<number>(worldReconnectToken);
  const pendingGusSyncRef = useRef<boolean>(false);

  useEffect(() => {
    if (panel === null && pendingGusSyncRef.current) {
      pendingGusSyncRef.current = false;
      const runtime = runtimeRef.current;
      if (worldActive && runtime && (runtime.getStatus() === "joined" || runtime.getStatus() === "idle")) {
        void runtime.reconnect();
      }
    }
  }, [panel, worldActive]);

  useEffect(() => {
    if (!worldActive) {
      lastReconnectTokenRef.current = worldReconnectToken;
      return;
    }

    const currentSerialized = JSON.stringify(player?.equipped_gus ?? []);
    const region = player?.world_zone_id?.trim() || player?.current_region || "";
    const instance = player?.world_instance_id?.trim() || "";

    const regionChanged = lastRegionRef.current !== "" && lastRegionRef.current !== region;
    const instanceChanged = lastInstanceRef.current !== instance;
    const gusChanged = lastEquippedGusRef.current !== currentSerialized;
    const reconnectRequested = lastReconnectTokenRef.current !== worldReconnectToken;
    const runtime = runtimeRef.current;
    const runtimeCanReconnect = runtime && (
      runtimeStatus === "joined" || runtimeStatus === "idle" || runtimeStatus === "closed" || runtimeStatus === "error"
    );

    // Keep the old routing refs while a join is in flight. runtimeStatus is a
    // dependency below, so the queued route change is retried as soon as that
    // join settles instead of being lost during `connecting`.
    if ((regionChanged || instanceChanged || reconnectRequested || gusChanged) && !runtimeCanReconnect) {
      lastReconnectTokenRef.current = worldReconnectToken;
      return;
    }

    lastRegionRef.current = region;
    lastInstanceRef.current = instance;
    lastEquippedGusRef.current = currentSerialized;
    lastReconnectTokenRef.current = worldReconnectToken;

    if (regionChanged && runtime) {
      void runtime.reconnectTo(region);
    } else if ((reconnectRequested || instanceChanged) && runtime) {
      // Server-owned private sessions (or a same-map teleport) can change the
      // realtime routing key without changing the map ID.
      void runtime.reconnect();
    } else if (gusChanged && runtime) {
      if (panel !== null) {
        pendingGusSyncRef.current = true;
      } else {
        void runtime.reconnect();
      }
    }
  }, [player?.equipped_gus, player?.current_region, player?.world_zone_id, player?.world_instance_id, worldActive, worldReconnectToken, runtimeStatus, panel]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;

    const runtime = canJoinMMOZone ? new WorldRuntime() : null;
    runtime?.setUiModal(panel !== null || menuOpen || isDivinationModalOpen || !worldActive);
    runtime?.setJoinZoneResolver(() => {
      const current = useGameStore.getState().player;
      return current?.world_zone_id?.trim() || current?.current_region?.trim() || null;
    });
    runtimeRef.current = runtime;
    let cancelled = false;
    let hudTimer: number | null = null;
    let lastHudAt = 0;
    const hudRefreshInterval = window.matchMedia("(max-width: 767px)").matches ? 200 : 100;
    const pushHud = () => {
      if (!runtime) return;
      const commit = () => {
        lastHudAt = performance.now();
        const next = buildHud(runtime, playerRef.current);
        if (next) {
          setHud(next);
        }
      };
      if (performance.now() - lastHudAt >= hudRefreshInterval) {
        commit();
      } else if (hudTimer === null) {
        hudTimer = window.setTimeout(() => {
          hudTimer = null;
          if (!cancelled) commit();
        }, hudRefreshInterval);
      }
    };
    const runtimeUnsubscribe = runtime?.subscribe(() => {
      setRuntimeStatus(runtime.getStatus());
      setZoneId(runtime.getWelcome()?.zone_id ?? "");
      if (runtime.getStatus() === "error") {
        setErrorMessage(runtime.getError());
        // A death/settlement in a private encounter moves the character to a
        // new region. If the automatic reconnect raced the player event, refresh
        // the authoritative region and rejoin the correct zone instead of
        // replaying the old aperture zone forever.
        if (runtime.getErrorCode() === "realtime_map_mismatch" && !recoveringMismatchRef.current) {
          recoveringMismatchRef.current = true;
          void (async () => {
            const state = useGameStore.getState();
            const playerId = state.player?.id;
            if (playerId) {
              await state.fetchPlayerStatus(playerId, true);
            }
            const current = useGameStore.getState().player;
            const resolvedZone = current?.world_zone_id?.trim() || current?.current_region?.trim();
            if (!cancelled && resolvedZone) {
              setErrorMessage("");
              setRuntimeStatus("connecting");
              void runtime.reconnectTo(resolvedZone);
            }
            window.setTimeout(() => {
              recoveringMismatchRef.current = false;
            }, 5000);
          })();
        }
      }
      pushHud();
    });
    const errorUnsubscribe = runtime?.onError((error) => {
      setRuntimeStatus("error");
      setErrorMessage(error.message);
    });
    const rejectUnsubscribe = runtime?.onReject((reject) => {
      if (reject.code === "realtime_transfer_denied") {
        useGameStore.getState().addToast(reject.message || "Không thể chuyển vùng", "error");
      }
    });
    const transferUnsubscribe = runtime?.onTransfer((transfer) => {
      const state = useGameStore.getState();
      // This handler owns the reconnect for server handoffs. Prevent the
      // current_region effect below from starting a second reconnect when the
      // player store update renders.
      lastRegionRef.current = transfer.to_zone_id;
      // legacy_x/legacy_y are the players.x/y projection; target_x/target_y
      // are map cells and must never be written to the player store.
      state.setPlayerFromServer({
        current_region: transfer.to_zone_id,
        world_zone_id: transfer.to_zone_id,
        x: transfer.legacy_x,
        y: transfer.legacy_y,
        hp: transfer.hp,
      });
      state.addToast(`Đã vượt rìa sang ${transfer.to_zone_id.replaceAll("_", " ")}`, "success");
      // The server has already released the source-zone lease. Rejoin using
      // the destination zone or the socket's normal reconnect would replay the
      // old zone_id and leave the character stuck at the border.
      setErrorMessage("");
      setRuntimeStatus("connecting");
      void runtimeRef.current?.reconnectTo(transfer.to_zone_id);
    });

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
          const zoneId = playerRef.current?.world_zone_id?.trim() || playerRef.current?.current_region?.trim() || mmoClientConfig.zoneId;
          if (!zoneId) {
            setRuntimeStatus("error");
            setErrorMessage("Nhân vật chưa có KHU_VUC để vào bản đồ realtime");
            return;
          }
          const versions = versionsForMMOZone(zoneId);
          void runtime.connect({
            zone_id: zoneId,
            map_version: versions.mapVersion,
            content_version: versions.contentVersion,
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
      rejectUnsubscribe?.();
      transferUnsubscribe?.();
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

  const reconnect = useCallback(async () => {
    const runtime = runtimeRef.current;
    if (!runtime) {
      retry();
      return;
    }
    setErrorMessage("");
    setRuntimeStatus("connecting");
    const state = useGameStore.getState();
    const playerId = state.player?.id;
    if (playerId) {
      await state.fetchPlayerStatus(playerId, true);
    }
    const current = useGameStore.getState().player;
    const resolvedZone = current?.world_zone_id?.trim() || current?.current_region?.trim();
    if (resolvedZone) {
      await runtime.reconnectTo(resolvedZone);
    } else {
      await runtime.reconnect();
    }
  }, [retry]);

  const connectionFailed = status === "error" || runtimeStatus === "error" || runtimeStatus === "closed";
  const showConnectionOverlay = !hud && (status !== "ready" || canJoinMMOZone || connectionFailed);
  const connectionLabel = status === "loading"
    ? "Đang chuẩn bị bản đồ"
    : runtimeStatus === "connecting"
      ? "Đang kết nối máy chủ realtime"
      : runtimeStatus === "joined"
        ? "Đang đồng bộ dữ liệu thế giới"
        : runtimeStatus === "closed"
          ? "Kết nối realtime đã đóng"
        : canJoinMMOZone
          ? "Đang khởi tạo kết nối"
          : "Chế độ xem trước";
  const connectionRegion = zoneId || player?.world_zone_id || player?.current_region || mmoClientConfig.zoneId;

  return (
    <MMOErrorBoundary>
      <div className="relative h-full w-full overflow-hidden bg-[#07110d]">
        <div ref={hostRef} id="guworld-phaser-host" className="absolute inset-0" />

        {showConnectionOverlay && (
          <div
            className="absolute inset-0 z-20 grid place-items-center overflow-y-auto bg-[#06100c]/85 px-4 py-8 backdrop-blur-sm"
            role={connectionFailed ? "alert" : "status"}
            aria-live={connectionFailed ? "assertive" : "polite"}
          >
            <div className="pointer-events-none absolute inset-0 overflow-hidden">
              <div className="absolute left-1/2 top-1/2 h-[28rem] w-[28rem] -translate-x-1/2 -translate-y-1/2 rounded-full bg-emerald-500/[0.07] blur-3xl" />
              <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-emerald-300/30 to-transparent" />
            </div>
            <section className="relative w-full max-w-md rounded-2xl border border-emerald-200/15 bg-[#0c1712]/95 p-6 shadow-[0_24px_90px_rgba(0,0,0,0.55)] sm:p-8">
              <div className="mb-6 flex items-center gap-3">
                <div className={`grid h-11 w-11 shrink-0 place-items-center rounded-xl border ${connectionFailed ? "border-red-300/20 bg-red-400/10" : "border-emerald-200/20 bg-emerald-300/[0.08]"}`}>
                  {connectionFailed ? (
                    <span className="text-lg text-red-200">!</span>
                  ) : (
                    <span className="relative grid h-5 w-5 place-items-center">
                      <span className="absolute h-5 w-5 animate-ping rounded-full bg-emerald-300/25" />
                      <span className="h-2.5 w-2.5 rounded-full bg-emerald-300 shadow-[0_0_16px_rgba(110,231,183,0.75)]" />
                    </span>
                  )}
                </div>
                <div className="min-w-0">
                  <p className="text-[10px] font-semibold uppercase tracking-[0.24em] text-emerald-200/55">Cổ Giới · Thế giới realtime</p>
                  <h2 className="mt-1 truncate font-serif text-lg text-emerald-50">
                    {connectionFailed
                      ? status === "error" ? "Không thể tải thế giới" : "Mất kết nối realtime"
                      : connectionLabel}
                  </h2>
                </div>
              </div>

              {connectionFailed ? (
                <>
                  <p className="max-h-28 overflow-auto break-words rounded-lg border border-red-200/10 bg-red-950/25 px-3 py-2.5 text-sm leading-relaxed text-red-100/75">
                    {errorMessage || "Kết nối thế giới chưa sẵn sàng. Hãy thử kết nối lại."}
                  </p>
                  <div className="mt-5 flex flex-wrap gap-2">
                    <button type="button" onClick={status === "error" ? retry : reconnect} className="rounded-lg border border-emerald-200/25 bg-emerald-300/10 px-4 py-2 text-sm font-medium text-emerald-50 transition hover:border-emerald-200/40 hover:bg-emerald-300/15">
                      Kết nối lại
                    </button>
                    {onExit && <button type="button" onClick={() => onExitRef.current?.()} className="rounded-lg border border-white/10 px-4 py-2 text-sm text-zinc-300 transition hover:bg-white/[0.06]">Bảng nhân vật</button>}
                  </div>
                </>
              ) : (
                <>
                  <p className="text-sm leading-relaxed text-zinc-300/75">
                    {runtimeStatus === "joined"
                      ? "Đã vào máy chủ. Đang nhận trạng thái nhân vật và bản đồ…"
                      : canJoinMMOZone
                        ? "Đang xác thực nhân vật và đồng bộ trạng thái với máy chủ."
                        : "Máy chủ realtime chưa được cấu hình cho khu vực này."}
                  </p>
                  <div className="mt-6 flex items-center gap-1.5" aria-hidden="true">
                    {[0, 1, 2].map((step) => (
                      <span key={step} className={`h-1 flex-1 rounded-full ${step === 0 ? "bg-emerald-300/80" : "bg-emerald-300/20"} ${runtimeStatus === "connecting" && step === 0 ? "animate-pulse" : ""}`} />
                    ))}
                  </div>
                  <div className="mt-3 flex items-center justify-between gap-3 text-xs text-zinc-400">
                    <span className="truncate">Khu vực: {connectionRegion ? connectionRegion.replaceAll("_", " ") : "đang xác định…"}</span>
                    <span className="shrink-0 text-emerald-200/70">{runtimeStatus === "connecting" ? "Đang kết nối" : status === "loading" ? "Đang tải" : "Đang chờ"}</span>
                  </div>
                  {onExit && <button type="button" onClick={() => onExitRef.current?.()} className="mt-6 w-full rounded-lg border border-white/10 px-4 py-2 text-sm text-zinc-300 transition hover:bg-white/[0.06]">Về bảng nhân vật</button>}
                </>
              )}
            </section>
          </div>
        )}

        {!hud && !showConnectionOverlay && (
          <div className="pointer-events-none absolute left-3 top-3 z-10 rounded-full border border-white/10 bg-black/55 px-3 py-1.5 text-[11px] uppercase tracking-[.16em] text-emerald-200/80 backdrop-blur">
            MMO Preview · chưa cấu hình zone
          </div>
        )}

        {hud && (
          <MMOHud
            {...hud}
            player={player}
            worldActive={worldActive && panel === null && !menuOpen && !isDivinationModalOpen}
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
            onCancelAim={() => {
              const game = gameRef.current;
              if (!game) return;
              const scene = game.scene.getScene("WorldScene") as unknown as {
                cancelAim?: () => void;
              };
              scene.cancelAim?.();
            }}
            onHoverSkill={(slot) => {
              const game = gameRef.current;
              if (!game) return;
              const scene = game.scene.getScene("WorldScene") as unknown as {
                setHoveredSkillSlot?: (s: number | null) => void;
              };
              scene.setHoveredSkillSlot?.(slot);
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
            onClose={() => {
              setPanel(null);
            }}
            onLoadoutSaved={() => {
              pendingGusSyncRef.current = true;
            }}
          />
        )}

        {hud && panel === null && worldActive && <GlobalChatOverlay placement="mmo" />}

      </div>
    </MMOErrorBoundary>
  );
}
