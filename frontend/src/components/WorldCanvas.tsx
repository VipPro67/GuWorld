"use client";

import { useEffect, useRef, useCallback, useState, useMemo } from "react";
import { Application, Container, Graphics, Text, FederatedPointerEvent } from "pixi.js";
import { useWorldStore, type WorldObject } from "@/store/useWorldStore";
import { useGameStore, type Player, type GuSpec } from "@/store/useGameStore";

function computeMovementSpeed(gamePlayer: Player | null, guDictionary: Record<string, GuSpec>): number {
  if (!gamePlayer) return 1;
  let movement = 1.0;
  for (const gu of gamePlayer.gu_inventory) {
    if (!gu.equipped || gu.status !== "ACTIVE") continue;
    const passive = guDictionary[gu.gu_id]?.effects?.passive;
    if (passive?.movement) movement += passive.movement;
  }
  return Math.max(1, movement);
}

const BG_COLOR = 0x1d1e2c;

interface WorldCanvasProps {
  mode: "select" | "drag";
}

const WORLD_W = 30000;
const WORLD_H = 20000;
const SUBZONE_SIZE = 1000;

export default function WorldCanvas({ mode }: WorldCanvasProps) {
  const canvasRef = useRef<HTMLDivElement>(null);
  const appRef = useRef<Application | null>(null);
  const [appReady, setAppReady] = useState(false);

  const stageRef = useRef<Container>(new Container());
  const regionsContainerRef = useRef<Container>(new Container());
  const objectsContainerRef = useRef<Container>(new Container());
  const monstersContainerRef = useRef<Container>(new Container());
  const fogRef = useRef<Graphics>(new Graphics());
  const playerRef = useRef<Graphics>(new Graphics());
  const rangeRef = useRef<Graphics>(new Graphics());

  const fogDirtyRef = useRef(true);
  const scaleRef = useRef<number>(0.05);
  const modeRef = useRef(mode);
  const moveTargetRef = useRef<{ x: number; y: number } | null>(null);
  const animIdRef = useRef<number>(0);
  const isMovingRef = useRef(false);

  const regions = useGameStore((s) => s.regions);
  const regionsLoaded = useGameStore((s) => s.regionsLoaded);
  const fetchRegions = useGameStore((s) => s.fetchRegions);
  const gamePlayer = useGameStore((s) => s.player);
  const setPlayerFromServer = useGameStore((s) => s.setPlayerFromServer);
  const setExploredSubZones = useWorldStore((s) => s.setExploredSubZones);
  const movementRange = useWorldStore((s) => s.movementRange);
  const objects = useWorldStore((s) => s.objects);
  const nearbyCells = useWorldStore((s) => s.nearbyCells);
  const fetchNearbyCells = useWorldStore((s) => s.fetchNearbyCells);
  const setLivePosition = useWorldStore((s) => s.setLivePosition);
  const setCamera = useWorldStore((s) => s.setCamera);
  const exploredSubZones = useWorldStore((s) => s.exploredSubZones);
  const movePath = useWorldStore((s) => s.movePath);
  const setMoving = useWorldStore((s) => s.setMoving);
  const selectObject = useWorldStore((s) => s.selectObject);
  const cameraCenterRequest = useWorldStore((s) => s.cameraCenterRequest);
  const fetchObjects = useWorldStore((s) => s.fetchObjects);
  const guDictionary = useGameStore((s) => s.guDictionary);

  const movementSpeed = useMemo(() => computeMovementSpeed(gamePlayer, guDictionary), [gamePlayer, guDictionary]);

  // Ensure regions loaded so we can draw them
  useEffect(() => {
    if (!regionsLoaded) fetchRegions();
    fetchObjects();
  }, [regionsLoaded, fetchRegions, fetchObjects]);

  // Sync explored sub-zones from the game player
  useEffect(() => {
    if (gamePlayer?.visited_subzones) {
      setExploredSubZones(gamePlayer.visited_subzones);
    }
  }, [gamePlayer?.visited_subzones, setExploredSubZones]);

  useEffect(() => {
    modeRef.current = mode;
    if (appRef.current) {
      appRef.current.stage.cursor = mode === "drag" ? "grab" : "default";
    }
  }, [mode]);

  const centerOnPlayer = useCallback((app: Application, px: number, py: number) => {
    const w = app.screen.width;
    const h = app.screen.height;
    const scale = scaleRef.current;
    stageRef.current.position.set(w / 2 - px * scale, h / 2 - py * scale);
    setCamera(scale, stageRef.current.position.x, stageRef.current.position.y);
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
        const eased = 1 - Math.pow(1 - t, 3);
        const cx = startX + (targetX - startX) * eased;
        const cy = startY + (targetY - startY) * eased;
        playerRef.current.position.set(cx, cy);
        setLivePosition(cx, cy);
        if (appRef.current && modeRef.current === "select") {
          centerOnPlayer(appRef.current, cx, cy);
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
      const startX = playerRef.current.position.x;
      const startY = playerRef.current.position.y;
      const dist = Math.hypot(targetX - startX, targetY - startY);
      if (dist < 1) {
        if (isMovingRef.current) {
          isMovingRef.current = false;
          setMoving(false);
        }
        return;
      }
      if (isMovingRef.current) return;

      cancelAnimationFrame(animIdRef.current);
      isMovingRef.current = true;
      setMoving(true);
      const origin = { x: startX, y: startY };

      // Walk in segments: each move-path call advances up to maxPathCells.
      // If reached is false, keep walking from the updated position.
      const walk = (tx: number, ty: number) => {
        movePath(tx, ty).then((res) => {
          if (!res) {
            cancelAnimationFrame(animIdRef.current);
            isMovingRef.current = false;
            setMoving(false);
            playerRef.current.position.set(origin.x, origin.y);
            if (appRef.current && modeRef.current === "select") {
              centerOnPlayer(appRef.current, origin.x, origin.y);
            }
            return;
          }
          const np = res.player;
          if (np) {
            setPlayerFromServer(np);
            setExploredSubZones(np.visited_subzones || []);
            const animDist = Math.hypot(np.x - playerRef.current.position.x, np.y - playerRef.current.position.y);
            const duration = Math.max(0.25, animDist / movementSpeed);
            animateMove(np.x, np.y, duration, () => {
              if (!res.reached && !res.combat_triggered) {
                walk(tx, ty);
              } else {
                isMovingRef.current = false;
                setMoving(false);
              }
            });
          } else {
            isMovingRef.current = false;
            setMoving(false);
          }
        });
      };

      walk(targetX, targetY);
    },
    [movementSpeed, animateMove, movePath, setMoving, setPlayerFromServer, setExploredSubZones, centerOnPlayer],
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
      const tx = Math.round(worldX);
      const ty = Math.round(worldY);

      if (tx === (gamePlayer?.x ?? 0) && ty === (gamePlayer?.y ?? 0)) {
        return;
      }
      moveTargetRef.current = { x: tx, y: ty };
      performMove(tx, ty);
    },
    [performMove, gamePlayer],
  );

  const handleObjectClick = useCallback(
    (obj: WorldObject) => {
      selectObject(obj);
      if (obj.x === (gamePlayer?.x ?? 0) && obj.y === (gamePlayer?.y ?? 0)) {
        return;
      }
      moveTargetRef.current = { x: obj.x, y: obj.y };
      performMove(obj.x, obj.y);
    },
    [performMove, selectObject, gamePlayer],
  );

  const drawRegions = useCallback((regs: typeof regions) => {
    const container = regionsContainerRef.current;
    container.removeChildren();

    const largeZones = regs.filter((r) => r.zone_type === "DAI_VUC");
    const namedZones = regs.filter((r) => r.zone_type === "NGU_VUC");
    const hub = regs.filter((r) => r.zone_type === "HUB");

    const baseColors = [0x2d5f3f, 0x3f5f2d, 0x5f4f2d, 0x4f2d5f, 0x2d3f5f, 0x5f2d3f];
    const colorFor = (idx: number) => baseColors[idx % baseColors.length];

    largeZones.forEach((z, i) => {
      if (z.min_x === undefined || z.max_x === undefined || z.min_y === undefined || z.max_y === undefined) return;
      const rx = z.min_x, ry = z.min_y, rw = z.max_x - z.min_x + 1, rh = z.max_y - z.min_y + 1;
      const g = new Graphics();
      const color = colorFor(i);
      g.rect(rx, ry, rw, rh);
      g.fill({ color, alpha: 0.25 });
      g.stroke({ color, width: 3, alpha: 0.7 });
      container.addChild(g);

      const label = new Text({
        text: z.name,
        style: { fontSize: 26, fill: 0xffffff, fontFamily: "serif", stroke: { color: 0x000000, width: 4 } },
      });
      label.anchor.set(0.5, 0.5);
      label.position.set((z.min_x + z.max_x) / 2, (z.min_y + z.max_y) / 2);
      container.addChild(label);
    });

    namedZones.forEach((z) => {
      if (z.min_x === undefined || z.max_x === undefined || z.min_y === undefined || z.max_y === undefined) return;
      const g = new Graphics();
      g.rect(z.min_x, z.min_y, z.max_x - z.min_x + 1, z.max_y - z.min_y + 1);
      g.fill({ color: 0xffffff, alpha: 0.04 });
      g.stroke({ color: 0xffffff, width: 1, alpha: 0.2 });
      container.addChild(g);

      const label = new Text({
        text: z.name,
        style: { fontSize: 13, fill: 0xd4d4d8, fontFamily: "serif", stroke: { color: 0x000000, width: 3 } },
      });
      label.anchor.set(0.5, 0.5);
      label.position.set((z.min_x + z.max_x) / 2, (z.min_y + z.max_y) / 2);
      container.addChild(label);
    });

    hub.forEach((z) => {
      if (z.min_x === undefined || z.max_x === undefined || z.min_y === undefined || z.max_y === undefined) return;
      const g = new Graphics();
      g.rect(z.min_x, z.min_y, z.max_x - z.min_x + 1, z.max_y - z.min_y + 1);
      g.fill({ color: 0xfbbf24, alpha: 0.18 });
      g.stroke({ color: 0xfbbf24, width: 3, alpha: 0.9 });
      container.addChild(g);

      const label = new Text({
        text: z.name,
        style: { fontSize: 22, fill: 0xfde68a, fontFamily: "serif", stroke: { color: 0x000000, width: 4 } },
      });
      label.anchor.set(0.5, 0.5);
      label.position.set((z.min_x + z.max_x) / 2, (z.min_y + z.max_y) / 2);
      container.addChild(label);
    });
  }, []);

  const drawObjects = useCallback(
    (obs: WorldObject[]) => {
      const container = objectsContainerRef.current;
      container.removeChildren();
      for (const o of obs) {
        const isBoss = o.cell_type === "Boss";
        const isResource = o.cell_type === "Resource";
        const g = new Graphics();
        const size = isBoss ? 12 : 8;
        if (isBoss) {
          g.poly([0, -size, size, 0, 0, size, -size, 0]);
          g.fill({ color: 0xef4444 });
          g.stroke({ color: 0x991b1b, width: 1.5 });
        } else if (isResource) {
          g.circle(0, 0, size / 2);
          g.fill({ color: 0x34d399 });
          g.stroke({ color: 0x059669, width: 1.5 });
        } else {
          g.rect(-size / 2, -size / 2, size, size);
          g.fill({ color: 0xfbbf24 });
          g.stroke({ color: 0xd97706, width: 1.5 });
        }
        g.eventMode = "static";
        g.cursor = "pointer";
        g.on("pointerdown", (event) => {
          event.stopPropagation();
          handleObjectClick(o);
        });
        container.addChild(g);

        const labelColor = isBoss ? 0xfecaca : 0xffffff;
        const label = new Text({
          text: o.content_id,
          style: { fontSize: 12, fill: labelColor, fontFamily: "serif", stroke: { color: 0x000000, width: 3 } },
        });
        label.anchor.set(0.5, 1);
        label.y = -size - 4;
        g.addChild(label);
        g.position.set(o.x, o.y);
      }
    },
    [handleObjectClick],
  );

  const drawMonsters = useCallback(
    (cells: typeof nearbyCells) => {
      const container = monstersContainerRef.current;
      container.removeChildren();

      // Batch small markers into per-color Graphics so thousands of cells stay
      // cheap for the renderer.
      const batchMonster = new Graphics();
      const batchResource = new Graphics();
      const batchWild = new Graphics();
      const monsterCount = Math.min(cells.filter((c) => c.cell_type === "Monster").length, 1500);
      const resourceCount = Math.min(cells.filter((c) => c.cell_type === "Resource").length, 800);
      const wildCount = Math.min(cells.filter((c) => c.cell_type === "WildGu").length, 400);

      let mM = 0, mR = 0, mW = 0;
      for (const c of cells) {
        if (c.cell_type === "Monster" && mM < monsterCount) {
          batchMonster.circle(c.x, c.y, 3.5);
          mM++;
        } else if (c.cell_type === "Resource" && mR < resourceCount) {
          batchResource.circle(c.x, c.y, 3);
          mR++;
        } else if (c.cell_type === "WildGu" && mW < wildCount) {
          batchWild.circle(c.x, c.y, 3.5);
          mW++;
        }
      }
      batchMonster.fill({ color: 0xf87171, alpha: 0.9 });
      batchResource.fill({ color: 0x34d399, alpha: 0.9 });
      batchWild.fill({ color: 0xfbbf24, alpha: 0.9 });
      container.addChild(batchMonster);
      container.addChild(batchResource);
      container.addChild(batchWild);

      // Draw bosses separately with labels (few in number).
      let bossCount = 0;
      for (const c of cells) {
        if (c.cell_type !== "Boss") continue;
        if (bossCount >= 40) break;
        bossCount++;
        const g = new Graphics();
        g.poly([0, -7, 7, 0, 0, 7, -7, 0]);
        g.fill({ color: 0xef4444 });
        g.stroke({ color: 0x7f1d1d, width: 1.5 });
        const label = new Text({
          text: c.content,
          style: { fontSize: 10, fill: 0xfecaca, fontFamily: "serif", stroke: { color: 0x000000, width: 2 } },
        });
        label.anchor.set(0.5, 1);
        label.y = -10;
        g.addChild(label);
        g.position.set(c.x, c.y);
        container.addChild(g);
      }
    },
    [],
  );

  const drawFog = useCallback(() => {
    const app = appRef.current;
    const g = fogRef.current;
    g.clear();
    if (!app || !gamePlayer) return;

    const stage = stageRef.current;
    const scale = scaleRef.current;
    const explored = exploredSubZones;

    const left = Math.max(0, (0 - stage.position.x) / scale);
    const top = Math.max(0, (0 - stage.position.y) / scale);
    const right = Math.min(WORLD_W, (app.screen.width - stage.position.x) / scale);
    const bottom = Math.min(WORLD_H, (app.screen.height - stage.position.y) / scale);
    if (right <= left || bottom <= top) return;

    const gx0 = Math.max(0, Math.floor(left / SUBZONE_SIZE));
    const gx1 = Math.min(WORLD_W / SUBZONE_SIZE - 1, Math.floor(right / SUBZONE_SIZE));
    const gy0 = Math.max(0, Math.floor(top / SUBZONE_SIZE));
    const gy1 = Math.min(WORLD_H / SUBZONE_SIZE - 1, Math.floor(bottom / SUBZONE_SIZE));

    for (let gy = gy0; gy <= gy1; gy++) {
      for (let gx = gx0; gx <= gx1; gx++) {
        const large = regions.find((r) => r.zone_type === "DAI_VUC" && gx * SUBZONE_SIZE >= (r.min_x ?? 0) && gx * SUBZONE_SIZE <= (r.max_x ?? -1) && gy * SUBZONE_SIZE >= (r.min_y ?? 0) && gy * SUBZONE_SIZE <= (r.max_y ?? -1));
        if (!large) continue;
        const col = Math.floor((gx * SUBZONE_SIZE - (large.min_x ?? 0)) / SUBZONE_SIZE);
        const row = Math.floor((gy * SUBZONE_SIZE - (large.min_y ?? 0)) / SUBZONE_SIZE);
        const key = `${large.id}:${col}:${row}`;
        if (explored.has(key)) continue;
        g.rect(gx * SUBZONE_SIZE, gy * SUBZONE_SIZE, SUBZONE_SIZE, SUBZONE_SIZE);
      }
    }
    g.fill({ color: 0x000000, alpha: 0.72 });
  }, [gamePlayer, exploredSubZones, regions]);

  const drawPlayer = useCallback(() => {
    const g = playerRef.current;
    g.clear();
    g.circle(0, 0, 6);
    g.fill({ color: 0x3b82f6 });
    g.stroke({ color: 0xffffff, width: 1.5 });
    g.circle(0, 0, 3);
    g.fill({ color: 0x93c5fd });
  }, []);

  const drawRange = useCallback(() => {
    const g = rangeRef.current;
    g.clear();
    if (!gamePlayer) return;
    const r = movementRange;
    g.rect(gamePlayer.x - r, gamePlayer.y - r, r * 2 + 1, r * 2 + 1);
    g.stroke({ color: 0x34d399, width: 1, alpha: 0.4 });
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
      // Zoom-out floor: the whole canvas may shrink until it fits the screen
      // (never smaller). Zoom-in is allowed much further for detail viewing.
      const app = appRef.current;
      const minScale = app ? Math.min(app.screen.width / WORLD_W, app.screen.height / WORLD_H) : 0.005;
      newScale = Math.max(minScale, Math.min(8, newScale));
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
        setCamera(newScale, stageRef.current.position.x, stageRef.current.position.y);
        fogDirtyRef.current = true;
      }
    };

    canvasElement.addEventListener("wheel", handleWheel, { passive: false });

    const initApp = async () => {
      const app = new Application();
      await app.init({
        resizeTo: canvasElement,
        background: BG_COLOR,
        antialias: true,
        resolution: window.devicePixelRatio || 1,
        autoDensity: true,
      });

      if (destroyed) {
        app.destroy();
        return;
      }

      canvasElement.appendChild(app.canvas);
      appRef.current = app;

      stageRef.current = new Container();
      app.stage.addChild(stageRef.current);

      stageRef.current.addChild(regionsContainerRef.current);
      stageRef.current.addChild(objectsContainerRef.current);
      stageRef.current.addChild(monstersContainerRef.current);
      stageRef.current.addChild(rangeRef.current);
      stageRef.current.addChild(fogRef.current);
      stageRef.current.addChild(playerRef.current);

      drawPlayer();
      drawRegions(regions);
      drawObjects(objects);
      drawMonsters(nearbyCells);
      drawFog();
      drawRange();

      app.ticker.add(() => {
        // Keep the player marker a constant screen size regardless of zoom.
        const s = scaleRef.current;
        playerRef.current.scale.set(1 / s);
        if (fogDirtyRef.current) {
          fogDirtyRef.current = false;
          drawFog();
        }
      });

      setAppReady(true);

      // Restore the previous camera (position + zoom) if the canvas was
      // unmounted by the combat screen; otherwise start fully zoomed out so
      // the whole world fits the screen.
      const saved = useWorldStore.getState();
      const savedScale = saved.cameraScale;
      const fitScale = Math.min(app.screen.width / WORLD_W, app.screen.height / WORLD_H);
      if (savedScale > 0) {
        scaleRef.current = savedScale;
      } else {
        scaleRef.current = fitScale;
      }
      stageRef.current.scale.set(scaleRef.current);

      if (gamePlayer) {
        playerRef.current.position.set(gamePlayer.x, gamePlayer.y);
        if (savedScale > 0) {
          stageRef.current.position.set(saved.cameraX, saved.cameraY);
          fogDirtyRef.current = true;
        } else {
          centerOnPlayer(app, gamePlayer.x, gamePlayer.y);
        }
      }

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
      canvasElement.removeEventListener("wheel", handleWheel);
      if (appRef.current) {
        appRef.current.destroy();
        appRef.current = null;
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!appReady || !appRef.current || !gamePlayer) return;
    playerRef.current.position.set(gamePlayer.x, gamePlayer.y);
  }, [appReady, gamePlayer]);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    drawObjects(objects);
  }, [appReady, objects, drawObjects]);

  useEffect(() => {
    if (!appReady || !appRef.current) return;
    drawMonsters(nearbyCells);
  }, [appReady, nearbyCells, drawMonsters]);

  // Poll nearby cells (monsters / resources) around the player while the map
  // is open so markers stay fresh as the player moves.
  useEffect(() => {
    if (!appReady || !gamePlayer) return;
    const poll = () => {
      fetchNearbyCells(gamePlayer.x, gamePlayer.y, 100);
    };
    poll();
    const id = setInterval(poll, 8000);
    return () => clearInterval(id);
  }, [appReady, gamePlayer?.x, gamePlayer?.y, fetchNearbyCells]);

  // Keep the live X/Y position in sync with the game player when idle.
  useEffect(() => {
    if (!gamePlayer) return;
    setLivePosition(gamePlayer.x, gamePlayer.y);
  }, [gamePlayer?.x, gamePlayer?.y, setLivePosition]);

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
    centerOnPlayer(appRef.current, gamePlayer.x, gamePlayer.y);
  }, [appReady, cameraCenterRequest, gamePlayer, centerOnPlayer]);

  return <div ref={canvasRef} className="w-full h-full overflow-hidden select-none" />;
}
