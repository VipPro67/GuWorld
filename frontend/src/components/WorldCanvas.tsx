"use client";

import { useEffect, useRef, useCallback, useState, useMemo } from "react";
import { Application, Container, Graphics, Text, FederatedPointerEvent, Texture, TilingSprite, Sprite, Assets, Circle } from "pixi.js";
import { CELL_CHUNK_SIZE, useWorldStore, type CellChunk, type NearbyCell, type NearbyPlayer, type WorldObject } from "@/store/useWorldStore";
import { guSpecFor, useGameStore, type Player, type GuSpec } from "@/store/useGameStore";
import { operationsBy } from "@/lib/effect-blocks";

// Tốc chạy map = MOVEMENT (số ô). SPEED chỉ dùng thứ tự đánh combat, không ảnh hưởng di chuyển.
function computeMovementRange(gamePlayer: Player | null, guDictionary: Record<string, GuSpec>): number {
  if (!gamePlayer) return 1;
  let movement = 1.0;
  for (const gu of gamePlayer.gu_inventory) {
    if (!gu.equipped || gu.status !== "ACTIVE") continue;
    const spec = guSpecFor(gu.gu_id, gu.rank);
    movement += operationsBy(spec?.effects, { trigger: "ALWAYS_ON", op: "STAT_MOD", stat: "MOVEMENT" })
      .reduce((sum, operation) => sum + (operation.value || 0), 0);
  }
  return Math.max(1, movement);
}

const BG_COLOR = 0x1d1e2c;

export interface MapLabelSettings {
  monster: boolean;
  bossNearby: boolean;
  hub: boolean;
  khu_vuc: boolean;
  ngu_vuc: boolean;
}

interface WorldCanvasProps {
  mode: "select" | "drag";
  labelSettings: MapLabelSettings;
}

const WORLD_W = 30000;
const WORLD_H = 20000;
const SUBZONE_SIZE = 1000;
// Keep the camera in the level-of-detail range backed by the nearby-cell AOI.
// Rendering a 30k x 20k overview made Pixi shade the whole world while the
// server only provides useful interactive data around the player.
const MIN_CAMERA_SCALE = 4;
const MAX_CAMERA_SCALE = 40;
const LIVE_POSITION_SYNC_MS = 100;
const CELL_AOI_DIAMETER = 254;
// Must match MovePlayerPath's bounded server segment. Prediction only affects
// presentation; every resulting position is still reconciled with the server.
const MOVE_PATH_SEGMENT_CELLS = 5;

function predictedSegmentEnd(x0: number, y0: number, x1: number, y1: number): { x: number; y: number } {
  let x = x0;
  let y = y0;
  const dx = Math.abs(x1 - x0);
  const dy = Math.abs(y1 - y0);
  const sx = x0 < x1 ? 1 : -1;
  const sy = y0 < y1 ? 1 : -1;
  let error = dx - dy;

  for (let step = 0; step < MOVE_PATH_SEGMENT_CELLS && (x !== x1 || y !== y1); step++) {
    const doubledError = 2 * error;
    if (doubledError > -dy) {
      error -= dy;
      x += sx;
    }
    if (doubledError < dx) {
      error += dx;
      y += sy;
    }
  }
  return { x, y };
}

function cameraMinScale(app: Application | null): number {
  if (!app) return MIN_CAMERA_SCALE;
  // The cell endpoint exposes at most a 127-cell radius. Never let the camera
  // show a wider area than that snapshot, otherwise interactive markers appear
  // to vanish at the edge of large/ultrawide displays.
  return Math.max(MIN_CAMERA_SCALE, app.screen.width / CELL_AOI_DIAMETER, app.screen.height / CELL_AOI_DIAMETER);
}

function clearContainer(container: Container) {
  for (const child of container.removeChildren()) {
    child.destroy({ children: true });
  }
}

export default function WorldCanvas({ mode, labelSettings }: WorldCanvasProps) {
  const canvasRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<Application | null>(null);
  const [appReady, setAppReady] = useState(false);

  const stageRef = useRef<Container>(new Container());
  const regionsContainerRef = useRef<Container>(new Container());
  const objectsContainerRef = useRef<Container>(new Container());
  const sectsContainerRef = useRef<Container>(new Container());
  const dropsContainerRef = useRef<Container>(new Container());
  const monstersContainerRef = useRef<Container>(new Container());
  const monsterLabelsContainerRef = useRef<Container>(new Container());
  const playersContainerRef = useRef<Container>(new Container());
  const fogRef = useRef<Graphics>(new Graphics());
  const playerRef = useRef<Graphics>(new Graphics());
  const rangeRef = useRef<Graphics>(new Graphics());
  const targetRingRef = useRef<Graphics>(new Graphics());
  const skillFxContainerRef = useRef<Container>(new Container());
  const processedFxRef = useRef<Set<string>>(new Set());

  const playerAvatarRef = useRef<Sprite | null>(null);
  const fogDirtyRef = useRef(true);
  const scaleRef = useRef<number>(0.05);
  const modeRef = useRef(mode);
  const labelSettingsRef = useRef(labelSettings);
  const labelRenderStateRef = useRef("");

  useEffect(() => {
    labelSettingsRef.current = labelSettings;
  }, [labelSettings]);
  const moveTargetRef = useRef<{ x: number; y: number } | null>(null);
  const animIdRef = useRef<number>(0);
  const lastLivePositionSyncRef = useRef(0);
  const lastCameraSyncRef = useRef(0);
  const isMovingRef = useRef(false);
  const walkVersionRef = useRef(0);
  const confirmedPositionRef = useRef<{ x: number; y: number } | null>(null);
  const playerMarkersRef = useRef<Map<string, { container: Container; player: NearbyPlayer; fromX: number; fromY: number; toX: number; toY: number; t0: number; duration: number; active: boolean }>>(new Map());
  const monsterChunksRef = useRef<Map<string, { container: Container; source: CellChunk }>>(new Map());
  const monsterLabelRenderKeyRef = useRef("");
  const renderObjectIdsRef = useRef<WeakMap<object, number>>(new WeakMap());
  const nextRenderObjectIdRef = useRef(1);
  const scheduleMonsterChunkSyncRef = useRef<() => void>(() => {});
  const loadedTilesRef = useRef<Set<string>>(new Set());

  const regions = useGameStore((s) => s.regions);
  const regionsLoaded = useGameStore((s) => s.regionsLoaded);
  const fetchRegions = useGameStore((s) => s.fetchRegions);
  const gamePlayer = useGameStore((s) => s.player);
  const playerAvatarUrl = gamePlayer?.avatar_url;
  const setPlayerFromServer = useGameStore((s) => s.setPlayerFromServer);
  const setExploredSubZones = useWorldStore((s) => s.setExploredSubZones);
  const movementRange = useWorldStore((s) => s.movementRange);
  const investigationRange = useWorldStore((s) => s.investigationRange);
  const objects = useWorldStore((s) => s.objects);
  const nearbyCells = useWorldStore((s) => s.nearbyCells);
  const cellChunks = useWorldStore((s) => s.cellChunks);
  const cellRegion = useWorldStore((s) => s.cellRegion);
  const nearbyWindow = useWorldStore((s) => s.nearbyWindow);
  const nearbyPlayers = useWorldStore((s) => s.nearbyPlayers);
  const fetchNearbyCells = useWorldStore((s) => s.fetchNearbyCells);
  const fetchNearbyPlayers = useWorldStore((s) => s.fetchNearbyPlayers);
  const fetchCellsRect = useWorldStore((s) => s.fetchCellsRect);
  const invalidateCell = useWorldStore((s) => s.invalidateCell);
  const mergeWSCells = useWorldStore((s) => s.mergeWSCells);
  const upsertPlayerPosition = useWorldStore((s) => s.upsertPlayerPosition);
  const pruneStalePlayers = useWorldStore((s) => s.pruneStalePlayers);
  const setLivePosition = useWorldStore((s) => s.setLivePosition);
  const monsterDictionary = useGameStore((s) => s.monsterDictionary);
  const setCamera = useWorldStore((s) => s.setCamera);
  const exploredSubZones = useWorldStore((s) => s.exploredSubZones);
  const movePath = useWorldStore((s) => s.movePath);
  const addWorldLog = useWorldStore((s) => s.addWorldLog);
  const setMoving = useWorldStore((s) => s.setMoving);
  const selectObject = useWorldStore((s) => s.selectObject);
  const cameraCenterRequest = useWorldStore((s) => s.cameraCenterRequest);
  const fetchObjects = useWorldStore((s) => s.fetchObjects);
  const guDictionary = useGameStore((s) => s.guDictionary);
  const itemDictionary = useGameStore((s) => s.itemDictionary);
  const groundDrops = useGameStore((s) => s.groundDrops);
  const sectMap = useGameStore((s) => s.sectMap);
  const fetchSectsMap = useGameStore((s) => s.fetchSectsMap);
  const myTeleportArrays = useGameStore((s) => s.myTeleportArrays);
  const skillFxEvents = useWorldStore((s) => s.skillFxEvents);
  const clearSkillFx = useWorldStore((s) => s.clearSkillFx);
  const selectedTarget = useWorldStore((s) => s.selectedTarget);

  const drawTargetRing = useCallback(() => {
    const ring = targetRingRef.current;
    ring.clear();
    if (!selectedTarget) return;

    ring.position.set(selectedTarget.x + 0.5, selectedTarget.y + 0.5);

    // Outer red pulsing circle
    ring.circle(0, 0, 0.65);
    ring.stroke({ width: 0.08, color: 0xef4444, alpha: 0.9 });

    // Inner reticle notches
    ring.rect(-0.7, -0.05, 0.2, 0.1).fill(0xef4444);
    ring.rect(0.5, -0.05, 0.2, 0.1).fill(0xef4444);
    ring.rect(-0.05, -0.7, 0.1, 0.2).fill(0xef4444);
    ring.rect(-0.05, 0.5, 0.1, 0.2).fill(0xef4444);
  }, [selectedTarget]);

  useEffect(() => {
    drawTargetRing();
  }, [drawTargetRing]);

  // Render Pixi.js skill animations (Aura ring & floating text popups)
  useEffect(() => {
    if (!appReady || !skillFxContainerRef.current) return;

    for (const event of skillFxEvents) {
      if (processedFxRef.current.has(event.id)) continue;
      processedFxRef.current.add(event.id);

      const colorMap: Record<string, number> = {
        damage: 0xef4444,
        heal: 0x10b981,
        buff_atk: 0xef4444,
        buff_def: 0x3b82f6,
        buff_speed: 0xeab308,
        mana_regen: 0x06b6d4,
        shield: 0xa855f7,
        teleport: 0x06b6d4,
      };
      const color = colorMap[event.type] || 0xf59e0b;

      const fxGroup = new Container();
      fxGroup.position.set(event.x + 0.5, event.y + 0.5);

      // Aura Ring
      const ring = new Graphics();
      ring.circle(0, 0, 0.4);
      ring.stroke({ width: 0.1, color, alpha: 0.9 });
      fxGroup.addChild(ring);

      // Floating Text Label
      let textStr = event.text;
      if (event.value && event.value > 0) {
        if (event.type === "heal") textStr = `+${Math.round(event.value)} HP`;
        else if (event.type === "buff_atk") textStr = `+${Math.round(event.value)} ATK`;
        else if (event.type === "buff_def") textStr = `+${Math.round(event.value)} DEF`;
        else if (event.type === "buff_speed") textStr = `+${Math.round(event.value)} SPEED`;
        else if (event.type === "mana_regen") textStr = `+${Math.round(event.value)} Mana`;
        else if (event.type === "shield") textStr = `+${Math.round(event.value)} Khiên`;
      }

      const txt = new Text({
        text: textStr,
        style: {
          fontSize: 16,
          fill: color,
          fontWeight: "bold",
          stroke: { color: 0x000000, width: 3 },
        },
      });
      txt.anchor.set(0.5, 1);
      txt.position.set(0, -0.6);
      const targetScale = scaleRef.current > 0 ? 1.0 / scaleRef.current : 20;
      txt.scale.set(targetScale);
      fxGroup.addChild(txt);

      skillFxContainerRef.current.addChild(fxGroup);

      const startTime = performance.now();
      const duration = 1600;

      const animateFx = () => {
        const elapsed = performance.now() - startTime;
        const progress = elapsed / duration;

        if (progress >= 1.0) {
          if (fxGroup.parent) fxGroup.parent.removeChild(fxGroup);
          fxGroup.destroy({ children: true });
          clearSkillFx(event.id);
        } else {
          const scaleVal = 0.4 + progress * 1.2;
          ring.clear();
          ring.circle(0, 0, scaleVal);
          ring.stroke({ width: Math.max(0.01, 0.15 * (1 - progress)), color, alpha: Math.max(0, 1 - progress) });

          txt.position.y = -0.6 - progress * 1.2;
          txt.alpha = Math.max(0, 1 - progress * 0.8);
          requestAnimationFrame(animateFx);
        }
      };

      requestAnimationFrame(animateFx);
    }
  }, [skillFxEvents, appReady, clearSkillFx]);

  const movementStat = useMemo(() => computeMovementRange(gamePlayer, guDictionary), [gamePlayer, guDictionary]);

  const displayObjects = useMemo(() => {
    const byKey = new Map<string, WorldObject>();
    for (const obj of objects) {
      byKey.set(`${obj.x},${obj.y}`, obj);
    }
    for (const cell of nearbyCells) {
      if (cell.cell_type !== "Boss" && cell.cell_type !== "Resource" && cell.cell_type !== "SafePoint" && cell.cell_type !== "Village" && cell.cell_type !== "TeleportArray") continue;
      const key = `${cell.x},${cell.y}`;
      if (!byKey.has(key)) {
        byKey.set(key, {
          region_id: gamePlayer?.current_region || "",
          x: cell.x,
          y: cell.y,
          cell_type: cell.cell_type,
          content_id: cell.content,
          rank: cell.rank,
        });
      }
    }
    if (myTeleportArrays) {
      for (const t of myTeleportArrays) {
        if (gamePlayer?.current_region && t.region_id !== gamePlayer.current_region) continue;
        const key = `${t.x},${t.y}`;
        byKey.set(key, {
          id: t.id,
          region_id: t.region_id,
          x: t.x,
          y: t.y,
          cell_type: "TeleportArray",
          content_id: t.name,
        });
      }
    }
    return Array.from(byKey.values());
  }, [objects, nearbyCells, myTeleportArrays, gamePlayer]);

  // Ensure regions loaded so we can draw them
  useEffect(() => {
    if (!regionsLoaded) fetchRegions();
    fetchObjects();
  }, [regionsLoaded, fetchRegions, fetchObjects]);

  // Sync explored sub-zones from the game player
  useEffect(() => {
    if (gamePlayer?.visited_subzones) {
      setExploredSubZones(gamePlayer.visited_subzones);
    } else if (gamePlayer) {
      setExploredSubZones([]);
    }
  }, [gamePlayer?.visited_subzones, setExploredSubZones, gamePlayer]);

  useEffect(() => {
    modeRef.current = mode;
    if (appRef.current) {
      appRef.current.stage.cursor = mode === "drag" ? "grab" : "default";
    }
  }, [mode]);

  const centerOnPlayer = useCallback((app: Application, px: number, py: number, forceSync = true) => {
    const w = app.screen.width;
    const h = app.screen.height;
    const scale = scaleRef.current;
    stageRef.current.position.set(w / 2 - px * scale, h / 2 - py * scale);
    scheduleMonsterChunkSyncRef.current();
    const now = performance.now();
    if (forceSync || now - lastCameraSyncRef.current >= LIVE_POSITION_SYNC_MS) {
      lastCameraSyncRef.current = now;
      setCamera(scale, stageRef.current.position.x, stageRef.current.position.y);
    }
    fogDirtyRef.current = true;
  }, [setCamera]);

  const animateMove = useCallback(
    (targetX: number, targetY: number, duration: number, onComplete?: () => void) => {
      cancelAnimationFrame(animIdRef.current);
      const start = performance.now();
      const startX = playerRef.current.position.x;
      const startY = playerRef.current.position.y;

      const step = (now: number) => {
        const t = Math.min((now - start) / (duration * 1000), 1);
        // Constant velocity prevents a visible slow-down at every five-cell
        // server segment and makes adjacent segments read as one continuous walk.
        const cx = startX + (targetX - startX) * t;
        const cy = startY + (targetY - startY) * t;
        playerRef.current.position.set(cx, cy);
        // Pixi owns the frame-by-frame position. Publishing to Zustand on every
        // animation frame caused WorldPage and every coordinate subscriber to
        // re-render at 60 FPS, so expose a lower-frequency UI snapshot instead.
        if (now - lastLivePositionSyncRef.current >= LIVE_POSITION_SYNC_MS || t >= 1) {
          lastLivePositionSyncRef.current = now;
          setLivePosition(cx, cy);
        }
        if (appRef.current && modeRef.current === "select") {
          centerOnPlayer(appRef.current, cx, cy, t >= 1);
        }
        if (t < 1) {
          animIdRef.current = requestAnimationFrame(step);
        } else {
          onComplete?.();
        }
      };
      animIdRef.current = requestAnimationFrame(step);
    },
    [centerOnPlayer, setLivePosition],
  );

  const performMove = useCallback(
    (targetX: number, targetY: number) => {
      const currentParty = useGameStore.getState().party;
      const curPlayer = useGameStore.getState().player;
      const isMemberNotLeader = Boolean(curPlayer?.party_id) && (currentParty === null || currentParty.leader_id !== curPlayer?.id);
      if (isMemberNotLeader) {
        useGameStore.getState().addToast("Chỉ có Trưởng nhóm mới có quyền di chuyển trên bản đồ!", "error");
        return;
      }

      const startX = playerRef.current.position.x;
      const startY = playerRef.current.position.y;
      const dist = Math.hypot(targetX + 0.5 - startX, targetY + 0.5 - startY);
      const confirmed = confirmedPositionRef.current;
      const confirmedDist = confirmed
        ? Math.hypot(targetX + 0.5 - confirmed.x, targetY + 0.5 - confirmed.y)
        : dist;

      // A server mutation cannot be cancelled once sent. While one is in
      // flight, coalesce input into the latest desired destination instead of
      // issuing overlapping requests and discarding an authoritative response.
      if (isMovingRef.current) {
        moveTargetRef.current = dist < 1 || confirmedDist < 1 ? null : { x: targetX, y: targetY };
        return;
      }

      // Clicking the cell the character is already on while idle is a no-op.
      if (dist < 1) {
        return;
      }

      moveTargetRef.current = { x: targetX, y: targetY };
      walkVersionRef.current += 1;
      const myVersion = walkVersionRef.current;
      cancelAnimationFrame(animIdRef.current);
      if (!isMovingRef.current) {
        isMovingRef.current = true;
        setMoving(true);
      }
      const player = useGameStore.getState().player;
      confirmedPositionRef.current = player
        ? { x: player.x + 0.5, y: player.y + 0.5 }
        : { x: startX, y: startY };

      const walk = (tx: number, ty: number) => {
        if (walkVersionRef.current !== myVersion) return;
        const confirmed = confirmedPositionRef.current ?? {
          x: playerRef.current.position.x,
          y: playerRef.current.position.y,
        };
        const predicted = predictedSegmentEnd(
          Math.round(confirmed.x - 0.5),
          Math.round(confirmed.y - 0.5),
          tx,
          ty,
        );
        const predictedX = predicted.x + 0.5;
        const predictedY = predicted.y + 0.5;
        const predictedDistance = Math.hypot(
          predictedX - playerRef.current.position.x,
          predictedY - playerRef.current.position.y,
        );
        let response: Awaited<ReturnType<typeof movePath>> | undefined;
        let predictionFinished = false;

        // Start walking immediately along the same bounded Bresenham segment
        // as the backend. Network latency is hidden behind this visual tween;
        // the response below remains authoritative if an encounter stops early.
        animateMove(predictedX, predictedY, Math.max(0.08, predictedDistance / movementStat), () => {
          predictionFinished = true;
          if (response !== undefined && response !== null) reconcile(response, predictedX, predictedY, tx, ty);
        });

        movePath(tx, ty).then((res) => {
          if (walkVersionRef.current !== myVersion) return;
          response = res;
          if (!res) {
            cancelAnimationFrame(animIdRef.current);
            isMovingRef.current = false;
            setMoving(false);
            moveTargetRef.current = null;
            const lastConfirmed = confirmedPositionRef.current;
            if (lastConfirmed) {
              animateMove(lastConfirmed.x, lastConfirmed.y, 0.15);
            }
            return;
          }
          const responseMatchedPrediction = res.player.x + 0.5 === predictedX && res.player.y + 0.5 === predictedY;
          if (predictionFinished || !responseMatchedPrediction) {
            reconcile(res, predictedX, predictedY, tx, ty);
          }
        });

        const reconcile = (
          res: NonNullable<Awaited<ReturnType<typeof movePath>>>,
          expectedX: number,
          expectedY: number,
          requestedX: number,
          requestedY: number,
        ) => {
          if (walkVersionRef.current !== myVersion) return;
          const np = res.player;
          if (np) {
            const authoritativeX = np.x + 0.5;
            const authoritativeY = np.y + 0.5;
            confirmedPositionRef.current = { x: authoritativeX, y: authoritativeY };
            const correctionDistance = Math.hypot(
              authoritativeX - playerRef.current.position.x,
              authoritativeY - playerRef.current.position.y,
            );
            const predictionMatched = authoritativeX === expectedX && authoritativeY === expectedY;
            const duration = predictionMatched
              ? 0
              : Math.min(0.2, Math.max(0.08, correctionDistance / movementStat));

            if (Array.isArray(res.synced_party_members)) {
              for (const sm of res.synced_party_members) {
                upsertPlayerPosition({
                  id: sm.id,
                  username: sm.username,
                  avatar_url: sm.avatar_url,
                  region_id: sm.region,
                  x: sm.x,
                  y: sm.y,
                });
              }
            }

            const finishSegment = () => {
              if (walkVersionRef.current !== myVersion) return;
              setPlayerFromServer(np);
              setExploredSubZones(np.visited_subzones || []);
              if (res.log) addWorldLog(res.log);

              // Handle skip battle result
              if (res.skip_result) {
                const skipResult = res.skip_result as { winner?: string; enemy_name?: string; mine_occupied?: string };
                const addToast = useGameStore.getState().addToast;
                if (skipResult.winner === "Phe Ta") {
                  addToast(`Tự động chiến thắng ${skipResult.enemy_name || "kẻ thù"}!`, "success");
                  if (skipResult.mine_occupied) {
                    addToast(`Đã chiếm mỏ ${skipResult.mine_occupied}!`, "info");
                  }
                } else {
                  addToast(`Tự động chiến đấu thất bại! ${skipResult.enemy_name || "Kẻ thù"} quá mạnh.`, "error");
                }
              }

              const desired = moveTargetRef.current;
              const hasAnotherDestination = Boolean(
                desired && (desired.x !== np.x || desired.y !== np.y),
              );
              if (!res.combat_triggered && hasAnotherDestination) {
                const skipRes = res.skip_result as { winner?: string } | undefined;
                if (skipRes && skipRes.winner !== "Phe Ta") {
                  isMovingRef.current = false;
                  setMoving(false);
                  selectObject(null);
                  return;
                }
                if (desired) walk(desired.x, desired.y);
                else {
                  isMovingRef.current = false;
                  setMoving(false);
                }
              } else {
                isMovingRef.current = false;
                setMoving(false);
                const currentDrops = useGameStore.getState().groundDrops;
                const dropAtTarget = currentDrops.find((d) => d.x === requestedX && d.y === requestedY);
                if (dropAtTarget) {
                  useGameStore.getState().lootGroundDrop(dropAtTarget.id);
                } else {
                  selectObject(null);
                }
              }
            };

            if (duration === 0) finishSegment();
            else animateMove(authoritativeX, authoritativeY, duration, finishSegment);
          } else {
            isMovingRef.current = false;
            setMoving(false);
          }
        };
      };

      walk(targetX, targetY);
    },
    [movementStat, animateMove, movePath, setMoving, setPlayerFromServer, setExploredSubZones, addWorldLog, selectObject, upsertPlayerPosition],
  );

  const handleObjectClick = useCallback(
    (obj: WorldObject) => {
      const s = useWorldStore.getState();
      if (s.selectedObject && s.selectedObject.x === obj.x && s.selectedObject.y === obj.y) {
        if (obj.x === (gamePlayer?.x ?? 0) && obj.y === (gamePlayer?.y ?? 0)) {
          if (obj.cell_type === "GroundDrop") {
            const dropId = obj.dropData?.id || obj.id;
            if (dropId) useGameStore.getState().lootGroundDrop(dropId);
          }
          return;
        }
        moveTargetRef.current = { x: obj.x, y: obj.y };
        performMove(obj.x, obj.y);
      } else {
        selectObject(obj);
      }
    },
    [performMove, selectObject, gamePlayer],
  );

  const handleBackgroundClick = useCallback(
    (app: Application, event: FederatedPointerEvent) => {
      const pos = event.global;
      const stage = stageRef.current;
      const scale = scaleRef.current;
      let worldX = (pos.x - stage.position.x) / scale;
      let worldY = (pos.y - stage.position.y) / scale;
      worldX = Math.max(0, Math.min(WORLD_W - 1, worldX));
      worldY = Math.max(0, Math.min(WORLD_H - 1, worldY));
      const tx = Math.floor(worldX);
      const ty = Math.floor(worldY);

      const s = useWorldStore.getState();
      if (s.targetingGu) {
        s.useGuOnMap(s.targetingGu.guInstanceId, tx, ty);
        return;
      }

      const nearby = s.getCachedCell(tx, ty);
      const persistent = s.objects.find((c) => c.x === tx && c.y === ty);
      const nearbyPl = s.nearbyPlayers.find((p) => p.x === tx && p.y === ty);
      const dropOnTile = useGameStore.getState().groundDrops.find((d) => d.x === tx && d.y === ty);
      const currentRegion = gamePlayer?.current_region || "";
      const sectOnTile = useGameStore.getState().sectMap.find(
        (sm) => sm.region === currentRegion && sm.map_x === tx && sm.map_y === ty
      );

      if (nearby || persistent || nearbyPl || dropOnTile || sectOnTile) {
        if (sectOnTile) {
          handleObjectClick({
            id: sectOnTile.id,
            region_id: sectOnTile.region,
            x: sectOnTile.map_x,
            y: sectOnTile.map_y,
            cell_type: "Sect",
            content_id: sectOnTile.name,
            rank: sectOnTile.level,
          });
          return;
        }

        const cellType = nearby ? nearby.cell_type : persistent ? persistent.cell_type : dropOnTile ? "GroundDrop" : "Player";
        const contentId = nearby ? nearby.content : persistent ? persistent.content_id : dropOnTile ? dropOnTile.id : nearbyPl?.username || "";
        const rank = nearby?.rank || persistent?.rank || 1;

        if (cellType === "Monster" || cellType === "Boss" || cellType === "WildGu") {
          const name = monsterDictionary[contentId]?.name || guSpecFor(contentId, rank)?.name || contentId;
          const baseHp = monsterDictionary[contentId]?.max_hp || rank * 500;
          s.selectTarget({
            x: tx,
            y: ty,
            cellType,
            contentId,
            name,
            rank,
            hp: baseHp,
            maxHp: baseHp,
          });
        }

        if (s.selectedObject && s.selectedObject.x === tx && s.selectedObject.y === ty) {
          // Already selected, proceed to move
          moveTargetRef.current = { x: tx, y: ty };
          performMove(tx, ty);
        } else if (dropOnTile) {
          s.selectObject({
            id: dropOnTile.id,
            region_id: dropOnTile.region_id,
            x: tx,
            y: ty,
            cell_type: "GroundDrop",
            content_id: dropOnTile.id,
            dropData: dropOnTile,
          });
        } else if (nearbyPl) {
          s.selectObject({
            id: nearbyPl.id,
            region_id: "",
            x: tx,
            y: ty,
            cell_type: "Player",
            content_id: nearbyPl.username,
            rank: undefined,
          });
        } else if (nearby) {
          s.selectObject({
            region_id: gamePlayer?.current_region || "",
            x: tx,
            y: ty,
            cell_type: nearby.cell_type,
            content_id: nearby.content,
            rank: nearby.rank,
          });
        } else if (persistent) {
          s.selectObject(persistent);
        }
        return;
      }

      s.selectObject(null);

      // Ignore clicks outside the investigation (scout) range
      const px = gamePlayer?.x ?? 0;
      const py = gamePlayer?.y ?? 0;
      const dist = Math.hypot(tx - px, ty - py);
      if (dist > s.investigationRange) return;

      moveTargetRef.current = { x: tx, y: ty };
      performMove(tx, ty);
    },
    [performMove, gamePlayer, handleObjectClick, monsterDictionary, guDictionary],
  );

  useEffect(() => {
    const handleCustomMove = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (detail && detail.x !== undefined && detail.y !== undefined) {
        moveTargetRef.current = { x: detail.x, y: detail.y };
        performMove(detail.x, detail.y);
      }
    };

    const handleKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      if (target && (["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName) || target.isContentEditable)) {
        return;
      }

      let dx = 0;
      let dy = 0;
      const key = e.key.toLowerCase();

      if (key === "w" || key === "arrowup") {
        dy = -1;
      } else if (key === "s" || key === "arrowdown") {
        dy = 1;
      } else if (key === "a" || key === "arrowleft") {
        dx = -1;
      } else if (key === "d" || key === "arrowright") {
        dx = 1;
      } else {
        return;
      }

      e.preventDefault();

      const curPlayer = useGameStore.getState().player;
      if (!curPlayer) return;

      const queuedTarget = moveTargetRef.current;
      const confirmed = confirmedPositionRef.current;
      const currentX = queuedTarget?.x ?? (confirmed ? Math.round(confirmed.x - 0.5) : curPlayer.x);
      const currentY = queuedTarget?.y ?? (confirmed ? Math.round(confirmed.y - 0.5) : curPlayer.y);

      const targetX = Math.max(0, Math.min(WORLD_W - 1, currentX + dx));
      const targetY = Math.max(0, Math.min(WORLD_H - 1, currentY + dy));

      performMove(targetX, targetY);
    };

    window.addEventListener("gu_world_move", handleCustomMove);
    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("gu_world_move", handleCustomMove);
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [performMove]);

  const drawRegions = useCallback((regs: typeof regions) => {
    const container = regionsContainerRef.current;
    clearContainer(container);
    labelRenderStateRef.current = "";

    const largeZones = regs.filter((r) => r.zone_type === "NGU_VUC");
    const namedZones = regs.filter((r) => r.zone_type === "KHU_VUC");
    const hub = regs.filter((r) => r.zone_type === "HUB" || r.zone_type === "SAFE_ZONE");
    const pvpZones = regs.filter((r) => r.zone_type === "PVP_ZONE");

    const baseColors = [0x2d5f3f, 0x3f5f2d, 0x5f4f2d, 0x4f2d5f, 0x2d3f5f, 0x5f2d3f];
    const colorFor = (idx: number) => baseColors[idx % baseColors.length];

    const addRegionLabel = (z: { name: string }, rx: number, ry: number, rw: number, rh: number, color: number, fontSize: number, type: string) => {
      const label = new Text({
        text: z.name,
        style: { fontSize, fill: color, fontFamily: "serif", stroke: { color: 0x000000, width: 4 }, fontWeight: "bold" },
      });
      label.anchor.set(0.5, 0.5);

      const labelContainer = new Container();
      labelContainer.name = `region-label-${type}`;
      labelContainer.position.set(rx + rw / 2, ry + rh / 2);
      labelContainer.addChild(label);
      container.addChild(labelContainer);
    };

    largeZones.forEach((z, i) => {
      if (z.min_x === undefined || z.max_x === undefined || z.min_y === undefined || z.max_y === undefined) return;
      const rx = z.min_x, ry = z.min_y, rw = z.max_x - z.min_x + 1, rh = z.max_y - z.min_y + 1;
      const g = new Graphics();
      const color = colorFor(i);
      g.rect(rx, ry, rw, rh);
      g.fill({ color, alpha: 0.05 });
      g.stroke({ color, width: 3, alpha: 0.8 });
      container.addChild(g);
      addRegionLabel(z, rx, ry, rw, rh, 0x86efac, 48, "NGU_VUC"); // Large zone label
    });

    namedZones.forEach((z) => {
      if (z.min_x === undefined || z.max_x === undefined || z.min_y === undefined || z.max_y === undefined) return;
      const rx = z.min_x, ry = z.min_y, rw = z.max_x - z.min_x + 1, rh = z.max_y - z.min_y + 1;
      const g = new Graphics();
      g.rect(rx, ry, rw, rh);
      g.fill({ color: 0xffffff, alpha: 0.04 });
      g.stroke({ color: 0xffffff, width: 1, alpha: 0.2 });
      container.addChild(g);
      addRegionLabel(z, rx, ry, rw, rh, 0xd8b4fe, 24, "KHU_VUC"); // Named zone label
    });

    hub.forEach((z) => {
      if (z.min_x === undefined || z.max_x === undefined || z.min_y === undefined || z.max_y === undefined) return;
      const rx = z.min_x, ry = z.min_y, rw = z.max_x - z.min_x + 1, rh = z.max_y - z.min_y + 1;
      const g = new Graphics();
      g.rect(rx, ry, rw, rh);
      g.fill({ color: 0x10b981, alpha: 0.15 });
      g.stroke({ color: 0x34d399, width: 3, alpha: 0.9 });
      container.addChild(g);
      addRegionLabel(z, rx, ry, rw, rh, 0x6ee7b7, 36, "HUB"); // Hub / Safe label
    });

    pvpZones.forEach((z) => {
      if (z.min_x === undefined || z.max_x === undefined || z.min_y === undefined || z.max_y === undefined) return;
      const rx = z.min_x, ry = z.min_y, rw = z.max_x - z.min_x + 1, rh = z.max_y - z.min_y + 1;
      const g = new Graphics();
      g.rect(rx, ry, rw, rh);
      g.fill({ color: 0xef4444, alpha: 0.20 });
      g.stroke({ color: 0xf87171, width: 3, alpha: 0.9 });
      container.addChild(g);
      addRegionLabel(z, rx, ry, rw, rh, 0xfca5a5, 36, "PVP_ZONE"); // PVP zone label
    });
  }, []);

  const drawObjects = useCallback(
    (obs: WorldObject[]) => {
      const container = objectsContainerRef.current;
      clearContainer(container);
      labelRenderStateRef.current = "";
      for (const o of obs) {
        const isBoss = o.cell_type === "Boss";
        const isResource = o.cell_type === "Resource";
        const isTeleport = o.cell_type === "TeleportArray";
        const g = new Graphics();
        const size = isBoss ? 0.8 : isTeleport ? 0.75 : 0.6;
        if (isBoss) {
          // Boss: Star/Diamond shape with glowing aura
          g.circle(0, 0, size + 0.2);
          g.fill({ color: 0xef4444, alpha: 0.3 }); // Aura
          g.poly([0, -size, size, 0, 0, size, -size, 0]);
          g.fill({ color: 0xef4444 });
          g.stroke({ color: 0x7f1d1d, width: 0.1 });
          g.circle(0, 0, size / 3);
          g.fill({ color: 0xfca5a5 }); // Inner highlight
        } else if (isResource) {
          // Resource / Tree: Layered green circles
          g.circle(0, 0.15, size * 0.8);
          g.fill({ color: 0x064e3b, alpha: 0.5 }); // Drop shadow
          g.circle(0, 0, size * 0.8);
          g.fill({ color: 0x065f46 }); // Dark base
          g.circle(0, -0.15, size * 0.6);
          g.fill({ color: 0x10b981 }); // Lighter top
          g.circle(-0.15, -0.3, size * 0.3);
          g.fill({ color: 0x34d399 }); // Highlight
        } else if (isTeleport) {
          // Teleport Array: Glowing cyan rune circle and magic vortex
          g.circle(0, 0, size + 0.25);
          g.fill({ color: 0x06b6d4, alpha: 0.25 }); // Outer glow
          g.circle(0, 0, size);
          g.fill({ color: 0x083344, alpha: 0.85 }); // Dark cyan base
          g.stroke({ color: 0x22d3ee, width: 0.1 });
          // Inner rune ring
          g.circle(0, 0, size * 0.6);
          g.stroke({ color: 0x67e8f9, width: 0.08, alpha: 0.7 });
          // Portal diamond rune
          g.poly([0, -size * 0.5, size * 0.35, 0, 0, size * 0.5, -size * 0.35, 0]);
          g.fill({ color: 0xa5f3fc });
          g.circle(0, 0, size * 0.2);
          g.fill({ color: 0xffffff });
        } else {
          // Empty / Location: Small shrine or stone marker
          g.rect(-size / 1.5, -size / 2, size * 1.33, size);
          g.fill({ color: 0x52525b }); // Stone base
          g.poly([0, -size, size / 1.5 + 0.1, -size / 2, -size / 1.5 - 0.1, -size / 2]);
          g.fill({ color: 0x0ea5e9 }); // Blue roof
          g.stroke({ color: 0x0284c7, width: 0.05 });
        }
        g.eventMode = "static";
        g.cursor = "pointer";
        g.on("pointerdown", (event) => {
          event.stopPropagation();
          handleObjectClick(o);
        });
        container.addChild(g);

        const labelColor = isBoss ? 0xfecaca : isTeleport ? 0x67e8f9 : 0xffffff;
        let text = o.content_id;
        if (isBoss || o.cell_type === "Monster") {
          text = monsterDictionary[o.content_id]?.name || text;
        } else if (isResource) {
          text = itemDictionary[o.content_id]?.name || text;
        } else if (o.cell_type === "WildGu") {
          text = guSpecFor(o.content_id, o.rank)?.name || text;
        } else if (isTeleport) {
          text = `🌀 ${o.content_id || "Trận Pháp"}`;
        }

        const showBossLabel = isBoss && !!labelSettingsRef.current.bossNearby && !!gamePlayer && Math.hypot(o.x - gamePlayer.x, o.y - gamePlayer.y) <= useWorldStore.getState().investigationRange;
        const label = new Text({
          text: text,
          style: { fontSize: 12, fill: labelColor, fontFamily: "sans-serif", stroke: { color: 0x000000, width: 3 } },
        });
        label.anchor.set(0.5, 1);
        label.y = -size - 0.8;
        label.name = "object-label-text";
        label.visible = !isBoss || showBossLabel;
        g.addChild(label);
        g.position.set(o.x + 0.5, o.y + 0.5);
      }
    },
    [handleObjectClick, monsterDictionary, itemDictionary, guDictionary],
  );

  const drawSects = useCallback(
    (sects: typeof sectMap) => {
      const container = sectsContainerRef.current;
      clearContainer(container);
      labelRenderStateRef.current = "";
      const currentRegion = gamePlayer?.current_region;
      if (!sects || !currentRegion) return;

      const mySectId = useGameStore.getState().mySectStatus?.sect?.id;
      for (const s of sects) {
        if (s.region !== currentRegion) continue;

        const sectGroup = new Container();

        // 1. Territory AoE Square - NON-INTERACTIVE
        const territoryG = new Graphics();
        const isMySect = s.id === mySectId;
        const mainHallLvl = s.buildings?.main_hall || s.level || 1;
        const territoryRadius = 5 + mainHallLvl;
        const halfSize = territoryRadius + 0.5;

        territoryG.rect(-halfSize, -halfSize, halfSize * 2, halfSize * 2);
        territoryG.fill({ color: isMySect ? 0x3b82f6 : 0xf59e0b, alpha: 0.08 });
        territoryG.stroke({ color: isMySect ? 0x60a5fa : 0xfbbf24, width: 0.15, alpha: 0.4 });
        territoryG.eventMode = "none";
        sectGroup.addChild(territoryG);

        // 2. Pagoda / Sect fortress building graphic - INTERACTIVE
        const g = new Graphics();
        const size = 1.2;
        // Outer glowing aura
        g.circle(0, 0, size + 0.3);
        g.fill({ color: isMySect ? 0x2563eb : 0xd97706, alpha: 0.25 });

        // Fortress Base
        g.rect(-size / 2, -size / 4, size, size * 0.75);
        g.fill({ color: 0x27272a });
        g.stroke({ color: isMySect ? 0x3b82f6 : 0xf59e0b, width: 0.1 });

        // Multi-tiered Pagoda Roofs
        g.poly([0, -size * 0.6, size * 0.8, -size / 4, -size * 0.8, -size / 4]);
        g.fill({ color: isMySect ? 0x1d4ed8 : 0xb45309 });
        g.stroke({ color: isMySect ? 0x93c5fd : 0xfde047, width: 0.08 });

        g.poly([0, -size * 1.1, size * 0.5, -size * 0.6, -size * 0.5, -size * 0.6]);
        g.fill({ color: isMySect ? 0x2563eb : 0xd97706 });
        g.stroke({ color: isMySect ? 0xbfdbfe : 0xfef08a, width: 0.08 });

        // Top spire
        g.circle(0, -size * 1.15, 0.15);
        g.fill({ color: 0xfef08a });

        g.hitArea = new Circle(0, 0, 1.2);
        g.eventMode = "static";
        g.cursor = "pointer";

        const sectWorldObj: WorldObject = {
          id: s.id,
          region_id: s.region,
          x: s.map_x,
          y: s.map_y,
          cell_type: "Sect",
          content_id: s.name,
          rank: s.level,
        };

        g.on("pointerdown", (event) => {
          event.stopPropagation();
          handleObjectClick(sectWorldObj);
        });
        sectGroup.addChild(g);

        const labelText = `🏯 ${s.name} (Cấp ${s.level || 1})`;
        const label = new Text({
          text: labelText,
          style: {
            fontSize: 13,
            fontWeight: "bold",
            fill: isMySect ? 0x93c5fd : 0xfde047,
            fontFamily: "sans-serif",
            stroke: { color: 0x000000, width: 3.5 },
          },
        });
        label.anchor.set(0.5, 1);
        label.y = -size * 1.3;
        label.name = "sect-label-text";
        label.eventMode = "none";
        sectGroup.addChild(label);

        sectGroup.position.set(s.map_x + 0.5, s.map_y + 0.5);
        container.addChild(sectGroup);
      }
    },
    [gamePlayer?.current_region, handleObjectClick],
  );

  const drawGroundDrops = useCallback(
    (drops: typeof groundDrops) => {
      const container = dropsContainerRef.current;
      clearContainer(container);
      labelRenderStateRef.current = "";

      for (const drop of drops) {
        const g = new Graphics();
        const size = 0.65;

        // Outer pulsing glow aura
        g.circle(0, 0, size + 0.2);
        g.fill({ color: 0xf59e0b, alpha: 0.35 });

        // Golden pouch / loot chest shape
        g.roundRect(-size / 2, -size / 2, size, size * 0.8, 0.15);
        g.fill({ color: 0xd97706 });
        g.stroke({ color: 0xfef08a, width: 0.08 });

        // Shiny gem accent in center
        g.poly([0, -size / 4, size / 4, 0, 0, size / 4, -size / 4, 0]);
        g.fill({ color: 0xfde047 });

        g.eventMode = "static";
        g.cursor = "pointer";
        g.on("pointerdown", (event) => {
          event.stopPropagation();
          handleObjectClick({
            id: drop.id,
            region_id: "",
            x: drop.x,
            y: drop.y,
            cell_type: "GroundDrop",
            content_id: drop.id,
            dropData: drop,
          });
        });

        g.position.set(drop.x + 0.5, drop.y + 0.5);
        container.addChild(g);

        // Label above loot bag (scaled dynamically with camera targetScale)
        const label = new Text({
          text: `🎁 Hài cốt: ${drop.loser_name}`,
          style: {
            fontSize: 11,
            fill: 0xfcd34d,
            fontFamily: "sans-serif",
            stroke: { color: 0x000000, width: 2 },
          },
        });
        label.anchor.set(0.5, 1);
        label.y = -0.6;

        const labelContainer = new Container();
        labelContainer.name = "drop-label";
        labelContainer.eventMode = "static";
        labelContainer.cursor = "pointer";
        labelContainer.on("pointerdown", (event) => {
          event.stopPropagation();
          handleObjectClick({
            id: drop.id,
            region_id: "",
            x: drop.x,
            y: drop.y,
            cell_type: "GroundDrop",
            content_id: drop.id,
            dropData: drop,
          });
        });
        labelContainer.position.set(drop.x + 0.5, drop.y + 0.5);
        labelContainer.addChild(label);
        container.addChild(labelContainer);
      }
    },
    [handleObjectClick, groundDrops],
  );

  const drawPlayers = useCallback(
    (players: typeof nearbyPlayers) => {
      const container = playersContainerRef.current;
      const markers = playerMarkersRef.current;
      const now = performance.now();
      // Prefer markers already on screen, then the closest newcomers. Limiting
      // before the loop prevents a full marker set from alternating between
      // removal and recreation on successive position updates.
      const px = useWorldStore.getState().liveX;
      const py = useWorldStore.getState().liveY;
      const selectedPlayers = players.length <= 300
        ? players
        : [...players]
            .sort((a, b) => Number(markers.has(b.id)) - Number(markers.has(a.id)) ||
              Math.hypot(a.x - px, a.y - py) - Math.hypot(b.x - px, b.y - py))
            .slice(0, 300);
      const alive = new Set(selectedPlayers.map((p) => p.id));
      for (const p of selectedPlayers) {
        alive.add(p.id);
        const nx = p.x + 0.5;
        const ny = p.y + 0.5;
        const existing = markers.get(p.id);
        if (!existing) {
          // Fresh marker: drop it at the reported position (no tween needed).
          const charContainer = new Container();
          charContainer.eventMode = "static";
          charContainer.cursor = "pointer";
          charContainer.on("pointerdown", (event) => {
            event.stopPropagation();
            const current = playerMarkersRef.current.get(p.id)?.player ?? p;
            handleObjectClick({
              id: current.id,
              region_id: "",
              x: current.x,
              y: current.y,
              cell_type: "Player",
              content_id: current.username,
              rank: undefined,
            });
          });

          if (p.avatar_url) {
            const sprite = new Sprite();
            sprite.anchor.set(0.5);
            sprite.width = 0.8;
            sprite.height = 0.8;
            Assets.load(p.avatar_url)
              .then((texture) => {
                if (!sprite.destroyed) sprite.texture = texture;
              })
              .catch(() => {
                if (charContainer.destroyed) return;
                const fallbackG = new Graphics();
                fallbackG.poly([0, -0.6, 0.6, 0, 0, 0.6, -0.6, 0]);
                fallbackG.fill({ color: 0x9333ea });
                fallbackG.stroke({ color: 0x6b21a8, width: 0.1 });
                charContainer.addChild(fallbackG);
              });
            charContainer.addChild(sprite);
          } else {
            const g = new Graphics();
            g.poly([0, -0.6, 0.6, 0, 0, 0.6, -0.6, 0]);
            g.fill({ color: 0x9333ea });
            g.stroke({ color: 0x6b21a8, width: 0.1 });
            charContainer.addChild(g);
          }

          charContainer.position.set(nx, ny);
          container.addChild(charContainer);
          markers.set(p.id, { container: charContainer, player: p, fromX: nx, fromY: ny, toX: nx, toY: ny, t0: now, duration: 250, active: false });
        } else {
          existing.player = p;
          // Known marker: if the server-reported position differs, glide from
          // the current (possibly interpolated) spot to the new spot.
          if (existing.toX !== nx || existing.toY !== ny) {
            const currentX = existing.container.position.x;
            const currentY = existing.container.position.y;
            const animDist = Math.hypot(nx - currentX, ny - currentY);
            const durMs = Math.max(200, Math.min(2000, (animDist / Math.max(1, movementStat)) * 1000));
            existing.fromX = currentX;
            existing.fromY = currentY;
            existing.toX = nx;
            existing.toY = ny;
            existing.t0 = now;
            existing.duration = durMs;
            existing.active = true;
            // Point this marker array no longer needs re-render; ticker tweens it.
          }
        }
      }

      // Remove markers for players that left the reported set.
      for (const id of Array.from(markers.keys())) {
        if (!alive.has(id)) {
          const leaving = markers.get(id);
          if (leaving) {
            container.removeChild(leaving.container);
            leaving.container.destroy({ children: true });
          }
          markers.delete(id);
        }
      }
    },
    [handleObjectClick, movementStat]
  );

  const drawMonsterChunk = useCallback((container: Container, cells: NearbyCell[]) => {
    clearContainer(container);

    // Each visible 64x64 chunk owns only its marker geometry. A cell delta
    // replaces one chunk object in Zustand, so every other Pixi batch survives.
    const batchMonster = new Graphics();
    const batchResource = new Graphics();
    const batchWild = new Graphics();
    let monsterCount = 0;
    let resourceCount = 0;
    let wildCount = 0;

    for (const c of cells) {
      if (c.cell_type === "Monster") {
        batchMonster.poly([c.x + 0.5, c.y + 0.1, c.x + 0.9, c.y + 0.8, c.x + 0.1, c.y + 0.8]);
        monsterCount++;
      } else if (c.cell_type === "Resource") {
        batchResource.circle(c.x + 0.5, c.y + 0.6, 0.3);
        batchResource.circle(c.x + 0.5, c.y + 0.4, 0.2);
        resourceCount++;
      } else if (c.cell_type === "WildGu") {
        batchWild.poly([c.x + 0.5, c.y + 0.1, c.x + 0.8, c.y + 0.5, c.x + 0.5, c.y + 0.9, c.x + 0.2, c.y + 0.5]);
        wildCount++;
      } else if (c.cell_type === "Boss") {
        const boss = new Graphics();
        boss.poly([0, -0.6, 0.6, 0, 0, 0.6, -0.6, 0]);
        boss.fill({ color: 0xef4444 });
        boss.stroke({ color: 0x7f1d1d, width: 0.1 });
        boss.position.set(c.x + 0.5, c.y + 0.5);
        container.addChild(boss);
      }
    }

    if (monsterCount) {
      batchMonster.fill({ color: 0xf87171, alpha: 0.9 });
      batchMonster.stroke({ color: 0x991b1b, width: 0.05 });
      container.addChild(batchMonster);
    } else batchMonster.destroy();
    if (resourceCount) {
      batchResource.fill({ color: 0x10b981, alpha: 0.9 });
      batchResource.stroke({ color: 0x065f46, width: 0.05 });
      container.addChild(batchResource);
    } else batchResource.destroy();
    if (wildCount) {
      batchWild.fill({ color: 0xfbbf24, alpha: 0.9 });
      batchWild.stroke({ color: 0xb45309, width: 0.05 });
      container.addChild(batchWild);
    } else batchWild.destroy();
  }, []);

  const drawMonsterLabels = useCallback((cells: NearbyCell[]) => {
    const container = monsterLabelsContainerRef.current;
    clearContainer(container);
    const settings = labelSettingsRef.current;
    const state = useGameStore.getState();
    const player = state.player;
    if (!player) return;

    if (settings.bossNearby) {
      let bossCount = 0;
      const investigationRange = useWorldStore.getState().investigationRange;
      for (const c of cells) {
        if (c.cell_type !== "Boss" || Math.hypot(c.x - player.x, c.y - player.y) > investigationRange) continue;
        if (bossCount++ >= 40) break;
        const label = new Text({
          text: state.monsterDictionary[c.content]?.name || c.content,
          style: { fontSize: 12, fill: 0xfecaca, fontFamily: "sans-serif", stroke: { color: 0x000000, width: 2 } },
        });
        label.anchor.set(0.5, 1);
        label.position.set(c.x + 0.5, c.y - 0.3);
        label.name = "boss-label";
        container.addChild(label);
      }
    }

    if (!settings.monster) return;
    const labelCells = cells
      .filter((c) => c.cell_type === "Monster" || c.cell_type === "Resource" || c.cell_type === "WildGu")
      .sort((a, b) => Math.hypot(a.x - player.x, a.y - player.y) - Math.hypot(b.x - player.x, b.y - player.y))
      .slice(0, 250);

    for (const c of labelCells) {
      let text = "";
      let fill = 0xffffff;
      if (c.cell_type === "Monster") {
        text = `${state.monsterDictionary[c.content]?.name || c.content}${c.rank ? ` R${c.rank}` : ""}`;
        fill = 0xfecaca;
      } else if (c.cell_type === "WildGu") {
        text = `${guSpecFor(c.content, c.rank)?.name || c.content}${c.rank ? ` R${c.rank}` : ""}`;
        fill = 0xf5d0fe;
      } else if (c.cell_type === "Resource") {
        text = state.itemDictionary[c.content]?.name || c.content;
        fill = 0xd1fae5;
      }
      if (!text) continue;
      const label = new Text({
        text,
        style: { fontSize: 11, fill, fontFamily: "sans-serif", stroke: { color: 0x000000, width: 2 } },
      });
      label.anchor.set(0.5, 1);
      label.position.set(c.x + 0.5, c.y - 0.1);
      label.name = "monster-label";
      container.addChild(label);
    }
  }, []);

  const syncMonsterChunks = useCallback((chunks: Record<string, CellChunk>, region: string, window: typeof nearbyWindow) => {
    const rendered = monsterChunksRef.current;
    if (!window || !region) {
      for (const entry of rendered.values()) entry.container.destroy({ children: true });
      rendered.clear();
      monstersContainerRef.current.removeChildren();
      clearContainer(monsterLabelsContainerRef.current);
      monsterLabelRenderKeyRef.current = "";
      return;
    }

    const app = appRef.current;
    const stage = stageRef.current;
    const scale = scaleRef.current;
    const overscan = CELL_CHUNK_SIZE;
    const minWorldX = app ? Math.max(0, (-stage.position.x / scale) - overscan) : window.x - window.r;
    const maxWorldX = app ? Math.min(WORLD_W - 1, ((app.screen.width - stage.position.x) / scale) + overscan) : window.x + window.r;
    const minWorldY = app ? Math.max(0, (-stage.position.y / scale) - overscan) : window.y - window.r;
    const maxWorldY = app ? Math.min(WORLD_H - 1, ((app.screen.height - stage.position.y) / scale) + overscan) : window.y + window.r;
    const minChunkX = Math.floor(minWorldX / CELL_CHUNK_SIZE);
    const maxChunkX = Math.floor(maxWorldX / CELL_CHUNK_SIZE);
    const minChunkY = Math.floor(minWorldY / CELL_CHUNK_SIZE);
    const maxChunkY = Math.floor(maxWorldY / CELL_CHUNK_SIZE);
    const wanted = new Set<string>();
    const visibleCells: NearbyCell[] = [];
    const visibleChunkVersions: string[] = [];

    for (let cx = minChunkX; cx <= maxChunkX; cx++) {
      for (let cy = minChunkY; cy <= maxChunkY; cy++) {
        const key = `${region}:${cx},${cy}`;
        const chunk = chunks[key];
        if (!chunk) continue;
        wanted.add(key);
        visibleCells.push(...Object.values(chunk.cells));
        let objectId = renderObjectIdsRef.current.get(chunk);
        if (!objectId) {
          objectId = nextRenderObjectIdRef.current++;
          renderObjectIdsRef.current.set(chunk, objectId);
        }
        visibleChunkVersions.push(`${key}@${objectId}`);
        const existing = rendered.get(key);
        if (existing?.source === chunk) continue;

        const chunkContainer = existing?.container ?? new Container();
        if (!existing) monstersContainerRef.current.addChild(chunkContainer);
        drawMonsterChunk(chunkContainer, Object.values(chunk.cells));
        rendered.set(key, { container: chunkContainer, source: chunk });
      }
    }

    for (const [key, entry] of rendered) {
      if (wanted.has(key)) continue;
      monstersContainerRef.current.removeChild(entry.container);
      entry.container.destroy({ children: true });
      rendered.delete(key);
    }
    const state = useGameStore.getState();
    const settings = labelSettingsRef.current;
    const labelInputs = [state.monsterDictionary, state.itemDictionary, state.guDictionary];
    const dictionaryIds = labelInputs.map((dictionary) => {
      let objectId = renderObjectIdsRef.current.get(dictionary);
      if (!objectId) {
        objectId = nextRenderObjectIdRef.current++;
        renderObjectIdsRef.current.set(dictionary, objectId);
      }
      return objectId;
    });
    const labelRenderKey = `${visibleChunkVersions.join("|")}:${Number(settings.monster)}${Number(settings.bossNearby)}:${state.player?.x ?? 0},${state.player?.y ?? 0}:${useWorldStore.getState().investigationRange}:${dictionaryIds.join(",")}`;
    if (labelRenderKey !== monsterLabelRenderKeyRef.current) {
      monsterLabelRenderKeyRef.current = labelRenderKey;
      drawMonsterLabels(visibleCells);
      labelRenderStateRef.current = "";
    }
  }, [drawMonsterChunk, drawMonsterLabels]);

  useEffect(() => {
    let frame = 0;
    scheduleMonsterChunkSyncRef.current = () => {
      if (frame) return;
      frame = requestAnimationFrame(() => {
        frame = 0;
        const state = useWorldStore.getState();
        syncMonsterChunks(state.cellChunks, state.cellRegion, state.nearbyWindow);
      });
    };
    return () => {
      scheduleMonsterChunkSyncRef.current = () => {};
      cancelAnimationFrame(frame);
    };
  }, [syncMonsterChunks]);

  const drawFog = useCallback(() => {
    const app = appRef.current;
    const g = fogRef.current;
    g.clear();
    // Fog disabled per user request
    return;
  }, []);

  const drawPlayer = useCallback(() => {
    const g = playerRef.current;
    g.clear();
    // Shadow
    g.circle(0, 0.2, 0.4);
    g.fill({ color: 0x000000, alpha: 0.4 });
    // Aura
    g.circle(0, 0, 0.5);
    g.fill({ color: 0x60a5fa, alpha: 0.3 });

    if (playerAvatarUrl) {
      if (!playerAvatarRef.current) {
        const sprite = new Sprite();
        sprite.anchor.set(0.5);
        sprite.width = 0.8;
        sprite.height = 0.8;

        Assets.load(playerAvatarUrl).then((texture) => {
          if (playerAvatarRef.current) playerAvatarRef.current.texture = texture;
        }).catch(console.error);

        const mask = new Graphics();
        mask.circle(0, 0, 0.4);
        mask.fill({ color: 0xffffff });
        sprite.mask = mask;

        g.addChild(mask);
        g.addChild(sprite);
        playerAvatarRef.current = sprite;
      }
      // Avatar border
      g.circle(0, 0, 0.4);
      g.stroke({ color: 0xffffff, width: 0.05 });
    } else {
      // Character base
      g.circle(0, 0, 0.3);
      g.fill({ color: 0x3b82f6 });
      g.stroke({ color: 0xffffff, width: 0.1 });
      // Center point
      g.circle(0, 0, 0.1);
      g.fill({ color: 0xffffff });
    }
  }, [playerAvatarUrl]);

  const drawRange = useCallback(() => {
    const g = rangeRef.current;
    g.clear();
    if (!gamePlayer) return;

    // Investigation range glow
    const invR = useWorldStore.getState().investigationRange; // It is fine to use getState here as investigationRange is tracked elsewhere
    for (let i = 0; i < 4; i++) {
      g.circle(gamePlayer.x + 0.5, gamePlayer.y + 0.5, invR * (1 - i * 0.2));
      g.fill({ color: 0xffffff, alpha: 0.02 });
    }
    g.circle(gamePlayer.x + 0.5, gamePlayer.y + 0.5, invR);
    g.stroke({ color: 0xffffff, width: 0.5, alpha: 0.1 });

    // Movement range box
    const r = movementRange;
    g.rect(gamePlayer.x - r, gamePlayer.y - r, r * 2 + 1, r * 2 + 1);
    g.stroke({ color: 0x34d399, width: 0.1, alpha: 0.8 });
  }, [gamePlayer, movementRange]);

  const clickHandlerRef = useRef(handleBackgroundClick);
  useEffect(() => {
    clickHandlerRef.current = handleBackgroundClick;
  }, [handleBackgroundClick]);

  useEffect(() => {
    if (!canvasRef.current) return;
    let destroyed = false;
    const canvasElement = canvasRef.current;

    const handleWheel = (e: WheelEvent) => {
      e.preventDefault();
      const zoomFactor = 1.15;
      const oldScale = scaleRef.current;
      let newScale = oldScale;
      if (e.deltaY < 0) {
        newScale *= zoomFactor;
      } else {
        newScale /= zoomFactor;
      }
      newScale = Math.max(cameraMinScale(appRef.current), Math.min(MAX_CAMERA_SCALE, newScale));
      scaleRef.current = newScale;

      if (appRef.current && stageRef.current) {
        const mouseX = e.offsetX;
        const mouseY = e.offsetY;
        const stageX = stageRef.current.position.x;
        const stageY = stageRef.current.position.y;
        const worldX = (mouseX - stageX) / oldScale;
        const worldY = (mouseY - stageY) / oldScale;
        stageRef.current.scale.set(newScale);
        stageRef.current.position.set(mouseX - worldX * newScale, mouseY - worldY * newScale);
        scheduleMonsterChunkSyncRef.current();
        setCamera(newScale, stageRef.current.position.x, stageRef.current.position.y);
        fogDirtyRef.current = true;
      }
    };

    let initialTouchDistance = 0;
    let initialScale = 0;
    let initialTouchCenter = { x: 0, y: 0 };

    const getTouchDistance = (t1: Touch, t2: Touch) => {
      return Math.hypot(t1.clientX - t2.clientX, t1.clientY - t2.clientY);
    };

    const getTouchCenter = (t1: Touch, t2: Touch) => {
      return {
        x: (t1.clientX + t2.clientX) / 2,
        y: (t1.clientY + t2.clientY) / 2,
      };
    };

    const handleTouchStart = (e: TouchEvent) => {
      if (e.touches.length === 2) {
        initialTouchDistance = getTouchDistance(e.touches[0], e.touches[1]);
        initialScale = scaleRef.current;
        initialTouchCenter = getTouchCenter(e.touches[0], e.touches[1]);
      }
    };

    const handleTouchMove = (e: TouchEvent) => {
      if (e.touches.length === 2 && initialTouchDistance > 0) {
        e.preventDefault();
        const currentDistance = getTouchDistance(e.touches[0], e.touches[1]);
        const ratio = currentDistance / initialTouchDistance;
        let newScale = initialScale * ratio;

        newScale = Math.max(cameraMinScale(appRef.current), Math.min(MAX_CAMERA_SCALE, newScale));

        const oldScale = scaleRef.current;
        scaleRef.current = newScale;

        if (appRef.current && stageRef.current) {
          const rect = canvasElement.getBoundingClientRect();
          const centerX = initialTouchCenter.x - rect.left;
          const centerY = initialTouchCenter.y - rect.top;

          const stageX = stageRef.current.position.x;
          const stageY = stageRef.current.position.y;
          const worldX = (centerX - stageX) / oldScale;
          const worldY = (centerY - stageY) / oldScale;

          stageRef.current.scale.set(newScale);
          stageRef.current.position.set(centerX - worldX * newScale, centerY - worldY * newScale);
          scheduleMonsterChunkSyncRef.current();
          setCamera(newScale, stageRef.current.position.x, stageRef.current.position.y);
          fogDirtyRef.current = true;
        }
      }
    };

    const handleTouchEnd = () => {
      initialTouchDistance = 0;
    };

    const handleResize = () => scheduleMonsterChunkSyncRef.current();

    canvasElement.addEventListener("wheel", handleWheel, { passive: false });
    canvasElement.addEventListener("touchstart", handleTouchStart, { passive: false });
    canvasElement.addEventListener("touchmove", handleTouchMove, { passive: false });
    canvasElement.addEventListener("touchend", handleTouchEnd);
    window.addEventListener("resize", handleResize);

    const resizeObserver = new ResizeObserver(() => {
      if (appRef.current && !destroyed) {
        appRef.current.resize();
        scheduleMonsterChunkSyncRef.current();
        fogDirtyRef.current = true;
      }
    });
    resizeObserver.observe(canvasElement);

    const initApp = async () => {
      const isMobile = window.matchMedia("(pointer: coarse)").matches || window.innerWidth < 768;
      const app = new Application();
      await app.init({
        resizeTo: canvasElement,
        background: BG_COLOR,
        // A 3x DPR phone otherwise shades nine times as many pixels as a 1x
        // display. Map geometry is already simple enough not to need MSAA on
        // small/coarse-pointer screens.
        antialias: !isMobile,
        resolution: Math.min(window.devicePixelRatio || 1, isMobile ? 1 : 2),
        autoDensity: true,
      });

      if (destroyed) {
        app.destroy();
        return;
      }

      app.canvas.style.display = "block";
      app.canvas.style.width = "100%";
      app.canvas.style.height = "100%";
      canvasElement.appendChild(app.canvas);
      appRef.current = app;
      // Keep animations smooth without forcing mobile GPUs to redraw this
      // very large scene at 60/120Hz.
      app.ticker.maxFPS = isMobile ? 30 : 60;

      stageRef.current = new Container();
      app.stage.addChild(stageRef.current);

      const bgGraphics = new Graphics();
      const tileSize = 1;
      bgGraphics.rect(0, 0, tileSize, tileSize);
      bgGraphics.fill({ color: 0x1a4226 });
      bgGraphics.stroke({ color: 0x14361e, width: 0.02, alignment: 0 }); // alignment 0 draws stroke completely inside

      const bgTexture = app.renderer.generateTexture(bgGraphics);
      const bgSprite = new TilingSprite({
        texture: bgTexture,
        width: WORLD_W,
        height: WORLD_H
      });
      stageRef.current.addChild(bgSprite);

      regionsContainerRef.current.eventMode = "none";
      fogRef.current.eventMode = "none";
      rangeRef.current.eventMode = "none";
      playerRef.current.eventMode = "none";
      targetRingRef.current.eventMode = "none";
      skillFxContainerRef.current.eventMode = "none";
      monsterLabelsContainerRef.current.eventMode = "none";

      stageRef.current.addChild(regionsContainerRef.current);
      stageRef.current.addChild(fogRef.current);
      stageRef.current.addChild(objectsContainerRef.current);
      stageRef.current.addChild(sectsContainerRef.current);
      stageRef.current.addChild(dropsContainerRef.current);
      stageRef.current.addChild(monstersContainerRef.current);
      stageRef.current.addChild(monsterLabelsContainerRef.current);
      stageRef.current.addChild(playersContainerRef.current);
      stageRef.current.addChild(rangeRef.current);
      stageRef.current.addChild(playerRef.current);
      stageRef.current.addChild(targetRingRef.current);
      stageRef.current.addChild(skillFxContainerRef.current);

      drawPlayer();
      drawRegions(regions);
      drawObjects(displayObjects);
      drawGroundDrops(groundDrops);
      drawPlayers(nearbyPlayers);
      drawFog();
      drawRange();

      app.ticker.add(() => {
        if (fogDirtyRef.current) {
          fogDirtyRef.current = false;
          drawFog();
        }

        // Interpolate remote player markers between server-reported positions
        // so movement reads smooth instead of teleporting.
        const markers = playerMarkersRef.current;
        for (const m of markers.values()) {
          if (!m.active) continue;
          const duration = m.duration || 300;
          const t = Math.min((performance.now() - m.t0) / duration, 1);
          const eased = 1 - Math.pow(1 - t, 3);
          m.container.position.set(m.fromX + (m.toX - m.fromX) * eased, m.fromY + (m.toY - m.fromY) * eased);
          if (t >= 1) {
            m.container.position.set(m.toX, m.toY);
            m.active = false;
          }
        }

        // Labels only need recalculating when zoom or label preferences change.
        // Previously every animation frame walked every map child, which was
        // especially expensive on mobile even while the camera was idle.
        const currentScale = scaleRef.current;
        if (currentScale > 0) {
          const settings = labelSettingsRef.current;
          const labelRenderState = `${currentScale}|${Number(settings.monster)}${Number(settings.bossNearby)}${Number(settings.hub)}${Number(settings.khu_vuc)}${Number(settings.ngu_vuc)}`;
          if (labelRenderState === labelRenderStateRef.current) return;
          labelRenderStateRef.current = labelRenderState;
          const showMonsters = settings.monster && currentScale >= 12;
          const targetScale = 1.0 / currentScale; // 1:1 crisp scale

          for (const child of monsterLabelsContainerRef.current.children) {
            if (!(child instanceof Text)) continue;
            child.visible = child.name !== "monster-label" || showMonsters;
            if (child.visible) child.scale.set(targetScale);
          }

          // Ground drop labels (scaled identically to monster labels)
          for (const child of dropsContainerRef.current.children) {
            if (child.name === "drop-label") {
              for (const sub of child.children) {
                if (sub instanceof Text) {
                  sub.scale.set(targetScale);
                }
              }
            }
          }

          // Sect labels
          for (const child of sectsContainerRef.current.children) {
            if (child instanceof Container) {
              for (const sub of child.children) {
                if (sub.name === "sect-label-text" && sub instanceof Text) {
                  sub.scale.set(targetScale);
                  sub.y = -1.8 - (6 * targetScale);
                }
              }
            }
          }

          // Persistent object labels
          for (const child of objectsContainerRef.current.children) {
            if (child instanceof Graphics) {
              for (const sub of child.children) {
                if (sub.name === "object-label-text" && sub instanceof Text) {
                  sub.visible = showMonsters;
                  if (showMonsters) {
                    sub.scale.set(targetScale);
                    // The icon radius is ~12 in world units, plus some padding in screen pixels
                    sub.y = -12 - (4 * targetScale);
                  }
                }
              }
            }
          }

          // Adjust region labels
          for (const child of regionsContainerRef.current.children) {
            let labelType: "hub" | "khu_vuc" | "ngu_vuc" | null = null;
            if (child.name === "region-label-HUB") {
              labelType = "hub";
            } else if (child.name === "region-label-KHU_VUC") {
              labelType = "khu_vuc";
            } else if (child.name === "region-label-NGU_VUC") {
              labelType = "ngu_vuc";
            }

            if (labelType) {
              const isVisible = settings[labelType];
              child.visible = isVisible;
              if (isVisible) {
                for (const sub of child.children) {
                  if (sub instanceof Text) {
                    sub.scale.set(targetScale);
                  }
                }
              }
            }
          }
        }
      });

      setAppReady(true);

      // Restore a compatible camera after combat; legacy whole-world camera
      // values are recentered because their translation is invalid at the new
      // detail-only zoom range.
      const saved = useWorldStore.getState();
      const savedScale = saved.cameraScale;
      const minScale = cameraMinScale(app);
      const canRestoreCamera = savedScale >= minScale && savedScale <= MAX_CAMERA_SCALE;
      if (canRestoreCamera) {
        scaleRef.current = savedScale;
      } else {
        scaleRef.current = MAX_CAMERA_SCALE;
      }
      stageRef.current.scale.set(scaleRef.current);

      if (gamePlayer) {
        playerRef.current.position.set(gamePlayer.x + 0.5, gamePlayer.y + 0.5);
        setLivePosition(gamePlayer.x + 0.5, gamePlayer.y + 0.5);
        if (canRestoreCamera) {
          stageRef.current.position.set(saved.cameraX, saved.cameraY);
          fogDirtyRef.current = true;
        } else {
          centerOnPlayer(app, gamePlayer.x + 0.5, gamePlayer.y + 0.5);
        }
      }
      scheduleMonsterChunkSyncRef.current();

      let isDragging = false;
      const dragStart = { x: 0, y: 0 };
      const stageStart = { x: 0, y: 0 };

      const onPointerDown = (e: FederatedPointerEvent) => {
        if (modeRef.current !== "drag") return;
        isDragging = true;
        dragStart.x = e.global.x;
        dragStart.y = e.global.y;
        stageStart.x = stageRef.current.position.x;
        stageStart.y = stageRef.current.position.y;
        app.stage.cursor = "grabbing";
      };

      const onPointerMove = (e: FederatedPointerEvent) => {
        if (!isDragging || modeRef.current !== "drag") return;
        const dx = e.global.x - dragStart.x;
        const dy = e.global.y - dragStart.y;
        stageRef.current.position.set(stageStart.x + dx, stageStart.y + dy);
        scheduleMonsterChunkSyncRef.current();
        setCamera(scaleRef.current, stageRef.current.position.x, stageRef.current.position.y);
        fogDirtyRef.current = true;
      };

      const onPointerUp = () => {
        isDragging = false;
        if (appRef.current) {
          appRef.current.stage.cursor = modeRef.current === "drag" ? "grab" : "default";
        }
      };

      app.stage.eventMode = "static";
      app.stage.cursor = modeRef.current === "drag" ? "grab" : "default";

      app.stage.on("pointerdown", (event) => {
        if (modeRef.current === "select") {
          clickHandlerRef.current(app, event);
        } else {
          onPointerDown(event);
        }
      });
      app.stage.on("pointermove", onPointerMove);
      app.stage.on("pointerup", onPointerUp);
      app.stage.on("pointerupoutside", onPointerUp);
    };

    initApp();

    return () => {
      destroyed = true;
      setAppReady(false);
      cancelAnimationFrame(animIdRef.current);
      resizeObserver.disconnect();
      canvasElement.removeEventListener("wheel", handleWheel);
      canvasElement.removeEventListener("touchstart", handleTouchStart);
      canvasElement.removeEventListener("touchmove", handleTouchMove);
      canvasElement.removeEventListener("touchend", handleTouchEnd);
      window.removeEventListener("resize", handleResize);
      if (appRef.current) {
        appRef.current.destroy();
        appRef.current = null;
      }
      monsterChunksRef.current.clear();
    };

  }, []);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    drawObjects(displayObjects);
  }, [appReady, displayObjects, drawObjects]);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    syncMonsterChunks(cellChunks, cellRegion, nearbyWindow);
  }, [appReady, cellChunks, cellRegion, nearbyWindow, labelSettings.monster, labelSettings.bossNearby, gamePlayer?.x, gamePlayer?.y, investigationRange, monsterDictionary, itemDictionary, guDictionary, syncMonsterChunks]);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    drawPlayers(nearbyPlayers);
  }, [appReady, nearbyPlayers, drawPlayers]);

  useEffect(() => {
    fetchSectsMap();
  }, [fetchSectsMap]);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    drawSects(sectMap);
  }, [appReady, sectMap, drawSects]);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    drawGroundDrops(groundDrops);
  }, [appReady, groundDrops, drawGroundDrops]);

  // Poll nearby cells (monsters / resources) around the player while the map
  // is open so markers stay fresh as the player moves. A separate, faster poll
  // keeps remote players fresh, and a background prefetch warms cells ahead of
  // the movement so markers appear progressively instead of popping in.
  useEffect(() => {
    if (!appReady || !useGameStore.getState().player) return;
    const pollCells = () => {
      const player = useGameStore.getState().player;
      if (player) fetchNearbyCells(player.x, player.y, useWorldStore.getState().investigationRange);
    };
    const pollPlayers = () => {
      const player = useGameStore.getState().player;
      if (player) fetchNearbyPlayers(player.x, player.y, useWorldStore.getState().investigationRange);
      pruneStalePlayers();
    };
    const prefetch = () => {
      const store = useWorldStore.getState();
      const px = store.liveX;
      const py = store.liveY;
      const tile = Math.max(150, investigationRange);
      const region = useGameStore.getState().player?.current_region ?? "";
      const want = new Set<string>();
      const addTile = (wx: number, wy: number) => {
        const tx = Math.floor(wx / tile);
        const ty = Math.floor(wy / tile);
        if (tx >= 0 && ty >= 0) want.add(`${region}|${tx},${ty}`);
      };
      addTile(px, py);
      // Band ahead of the current walking direction.
      const goal = moveTargetRef.current;
      const gx = goal ? goal.x + 0.5 : px;
      const gy = goal ? goal.y + 0.5 : py;
      const dist = Math.hypot(gx - px, gy - py);
      if (dist > tile * 0.5) {
        const ux = (gx - px) / dist;
        const uy = (gy - py) / dist;
        for (let k = 1; k <= 3; k++) addTile(px + ux * k * tile, py + uy * k * tile);
      }
      // Drop the "loaded" watermark for tiles far away so returning players
      // re-fetch (their cells may have been pruned from the client cache).
      const pt = Math.floor(px / tile);
      const qy = Math.floor(py / tile);
      for (const key of Array.from(loadedTilesRef.current)) {
        const [keyRegion, coords] = key.split("|");
        const [tx, ty] = coords.split(",").map(Number);
        if (keyRegion !== region || Math.abs(tx - pt) > 7 || Math.abs(ty - qy) > 7) loadedTilesRef.current.delete(key);
      }
      let scheduled = 0;
      for (const key of want) {
        if (loadedTilesRef.current.has(key)) continue;
        if (scheduled >= 2) break;
        scheduled++;
        const [, coords] = key.split("|");
        const [tx, ty] = coords.split(",").map(Number);
        // Reserve while in flight, but only retain the watermark on success so
        // transient websocket/HTTP failures can be retried by the next tick.
        loadedTilesRef.current.add(key);
        void fetchCellsRect(tx * tile, ty * tile, Math.min(WORLD_W - 1, (tx + 1) * tile - 1), Math.min(WORLD_H - 1, (ty + 1) * tile - 1))
          .then((loaded) => {
            if (!loaded) loadedTilesRef.current.delete(key);
          });
      }
    };
    const onWorldPos = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      const currentRegion = useGameStore.getState().player?.current_region;
      if (
        detail &&
        detail.id &&
        detail.x !== undefined &&
        detail.y !== undefined &&
        (!detail.region_id || detail.region_id === currentRegion) &&
        detail.id !== useGameStore.getState().player?.id
      ) {
        upsertPlayerPosition(detail);
        // Prune stale players whenever we get a fresh position update from the server.
        pruneStalePlayers();
      }
    };

    const onWorldCell = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      if (!detail || detail.x === undefined || detail.y === undefined) return;
      // Coordinates are global and a visible rectangle can cross a named-region
      // boundary, so apply cell deltas even when their derived region differs
      // from the player's current region.
      // When a cell is cleared, invalidate it immediately from cache so markers disappear.
      if (detail.type === "cell_cleared" || detail.type === "monster_killed") {
        invalidateCell(detail.x, detail.y);
      }
      // Newer servers may include the replacement cell in the delta. Applying
      // it directly avoids a full nearby-cell and persistent-object refetch.
      if (detail.cell) mergeWSCells([detail.cell]);
    };

    const onWorldCells = (e: Event) => {
      const detail = (e as CustomEvent).detail;
      const currentRegion = useGameStore.getState().player?.current_region;
      if ((!detail?.region_id || detail.region_id === currentRegion) && detail?.cells && Array.isArray(detail.cells)) {
        mergeWSCells(detail.cells);
      }
    };

    pollCells();
    pollPlayers();
    prefetch();
    window.addEventListener("gu_world_pos", onWorldPos);
    window.addEventListener("gu_world_cell", onWorldCell);
    window.addEventListener("gu_world_cells", onWorldCells);
    let lastCellsPoll = Date.now();
    let lastPlayersPoll = Date.now();
    let lastPrefetchPoll = Date.now();
    const unifiedTick = () => {
      const now = Date.now();
      // WebSocket tile commands make frequent cell polling redundant; keep 60s as fallback.
      if (now - lastCellsPoll >= 60000) {
        lastCellsPoll = now;
        pollCells();
      }
      if (now - lastPlayersPoll >= 30000) {
        lastPlayersPoll = now;
        pollPlayers();
      }
      if (now - lastPrefetchPoll >= 3000) {
        lastPrefetchPoll = now;
        prefetch();
      }
    };
    const intervalId = setInterval(unifiedTick, 1000);
    return () => {
      window.removeEventListener("gu_world_pos", onWorldPos);
      window.removeEventListener("gu_world_cell", onWorldCell);
      window.removeEventListener("gu_world_cells", onWorldCells);
      clearInterval(intervalId);
    };
    // Read current coordinates inside each tick: position changes must not tear
    // down this effect and immediately launch another group of map requests.
  }, [appReady, investigationRange, fetchNearbyCells, fetchNearbyPlayers, fetchCellsRect, upsertPlayerPosition, pruneStalePlayers, mergeWSCells, invalidateCell]);

  // Keep the live X/Y position in sync with the game player when idle or moved by party leader.
  useEffect(() => {
    if (!appReady || !gamePlayer || isMovingRef.current) return;
    const targetX = gamePlayer.x + 0.5;
    const targetY = gamePlayer.y + 0.5;
    const curX = playerRef.current.position.x;
    const curY = playerRef.current.position.y;
    const dist = Math.hypot(targetX - curX, targetY - curY);

    if (dist > 0.05) {
      if (dist <= 25) {
        const duration = Math.max(0.25, dist / Math.max(1, movementStat));
        animateMove(targetX, targetY, duration, () => {
          fetchNearbyCells(gamePlayer.x, gamePlayer.y, useWorldStore.getState().investigationRange);
          fetchNearbyPlayers(gamePlayer.x, gamePlayer.y, useWorldStore.getState().investigationRange);
        });
      } else {
        playerRef.current.position.set(targetX, targetY);
        setLivePosition(targetX, targetY);
        if (modeRef.current === "select" && appRef.current) {
          centerOnPlayer(appRef.current, targetX, targetY);
        }
        fetchNearbyCells(gamePlayer.x, gamePlayer.y, useWorldStore.getState().investigationRange);
        fetchNearbyPlayers(gamePlayer.x, gamePlayer.y, useWorldStore.getState().investigationRange);
      }
    }
  }, [appReady, gamePlayer?.x, gamePlayer?.y, movementStat, animateMove, centerOnPlayer, setLivePosition, fetchNearbyCells, fetchNearbyPlayers]);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    drawFog();
  }, [appReady, exploredSubZones, drawFog]);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    drawRegions(regions);
  }, [appReady, regions, drawRegions]);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    drawRange();
  }, [appReady, movementRange, gamePlayer, drawRange]);

  useEffect(() => {
    if (!appReady || !appRef.current || !gamePlayer || cameraCenterRequest === 0) return;
    scaleRef.current = MAX_CAMERA_SCALE;
    stageRef.current.scale.set(MAX_CAMERA_SCALE);
    centerOnPlayer(appRef.current, gamePlayer.x + 0.5, gamePlayer.y + 0.5);
  }, [appReady, cameraCenterRequest, gamePlayer, centerOnPlayer]);

  return <div ref={canvasRef} className="w-full h-full overflow-hidden select-none" />;
}
