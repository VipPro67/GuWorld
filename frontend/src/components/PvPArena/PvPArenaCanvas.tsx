"use client";

import { Application, Assets, Container, Graphics, Sprite, Text, Texture } from "pixi.js";
import { useEffect, useRef } from "react";
import { useGameStore } from "@/store/useGameStore";
import type { PvPControlState, PvPFrame, PvPParticipantState, PvPSetup, PvPShape } from "@/store/slices/pvpSlice";

interface PvPArenaCanvasProps {
  setup: PvPSetup;
  localPlayerId: string;
  controlRef: React.RefObject<PvPControlState>;
  /** Player avatar URL. Falls back to the shared cultivator sprite. */
  playerAvatarUrl?: string;
}

const DEFAULT_CULTIVATOR = "/assets/world/cultivator.webp";

const COLORS = {
  background: 0x070d0b,
  floor: 0x12241a,
  grid: 0x1b3325,
  obstacleFill: 0x1f3b2c,
  obstacleStroke: 0x3f7a58,
  local: 0x34d399,
  remote: 0xf87171,
  projectile: 0xfbbf24,
  boltCore: 0xfff7db,
  indicator: 0x60a5fa,
  rangeRing: 0x60a5fa,
  hpBack: 0x111827,
  hpFill: 0x22c55e,
  shield: 0x38bdf8,
  cast: 0xfacc15,
  woodPost: 0x8b5a2b,
  woodBar: 0x6b4423,
  woodHead: 0xa9713f,
};

/** Re-sync the predicted local position past this error (tiles). */
const SNAP_TILES = 0.75;
/** How fast the prediction eases toward the authoritative position. */
const RECONCILE_RATE = 0.25;
/** Screen-space size of the name/HP readout above each actor. */
const BAR_WIDTH = 46;
const BAR_HEIGHT = 5;

interface FloatingText {
  node: Text;
  life: number;
  /** Arena-tile position the text rises from. */
  x: number;
  y: number;
  rise: number;
}

/** Expanding ring used for impacts, cast pops and blinks. */
interface Burst {
  node: Graphics;
  life: number;
  x: number;
  y: number;
  radius: number;
  color: number;
}

/**
 * Arena renderer. Frames arrive at 20Hz on the match channel; this component
 * subscribes to the store imperatively and renders at display rate, so per-tick
 * updates never trigger a React re-render.
 *
 * World geometry lives in a scaled container. Names, HP bars and floating
 * combat text live in an unscaled overlay so they stay readable at any zoom.
 *
 * The local player is predicted from the same movement math the server runs;
 * the constants come from the setup payload, so the two cannot drift.
 */
export const PvPArenaCanvas: React.FC<PvPArenaCanvasProps> = ({ setup, localPlayerId, controlRef, playerAvatarUrl }) => {
  const hostRef = useRef<HTMLDivElement>(null);
  const bufferRef = useRef<{ prev: PvPFrame | null; curr: PvPFrame | null; receivedAt: number }>({
    prev: null,
    curr: null,
    receivedAt: 0,
  });
  const localPosRef = useRef<{ x: number; y: number } | null>(null);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;

    let destroyed = false;
    let app: Application | null = null;

    const actors = new Map<string, { body: Graphics; sprite: Sprite | null; label: Text; bar: Graphics }>();
    const projectiles = new Map<number, Graphics>();
    const floats: FloatingText[] = [];
    const bursts: Burst[] = [];
    let indicator: Graphics | null = null;
    let actorLayer: Container | null = null;
    let projectileLayer: Container | null = null;
    let overlayLayer: Container | null = null;

    // Tiles/second per point of the Speed stat, derived from the server's own
    // constants so the prediction cannot drift.
    const tilesPerSecondPerSpeed = setup.speed_to_tiles_per_tick * (1000 / setup.tick_ms);

    const blocked = (x: number, y: number) => {
      for (const obstacle of setup.obstacles) {
        if (obstacle.kind === "circle") {
          if (Math.hypot(x - obstacle.x, y - obstacle.y) < obstacle.r + setup.actor_radius) return true;
        } else if (
          x > obstacle.x - setup.actor_radius &&
          x < obstacle.x + obstacle.w + setup.actor_radius &&
          y > obstacle.y - setup.actor_radius &&
          y < obstacle.y + obstacle.h + setup.actor_radius
        ) {
          return true;
        }
      }
      return false;
    };

    const clamp = (x: number, y: number) => ({
      x: Math.max(setup.actor_radius, Math.min(setup.width - setup.actor_radius, x)),
      y: Math.max(setup.actor_radius, Math.min(setup.height - setup.actor_radius, y)),
    });

    // Mirrors match.slideMove: full step, then each axis alone, then hold.
    const slide = (x: number, y: number, dx: number, dy: number) => {
      const attempts: Array<[number, number]> = [
        [dx, dy],
        [dx, 0],
        [0, dy],
      ];
      for (const [stepX, stepY] of attempts) {
        const next = clamp(x + stepX, y + stepY);
        if (!blocked(next.x, next.y)) return next;
      }
      return { x, y };
    };

    // Draws a cast footprint in world units. Shared by the local aim preview
    // and by the telegraphs opponents see, so both render the exact geometry
    // the server resolves against.
    const drawFootprint = (
      g: Graphics,
      footprint: { shape: PvPShape; radius: number; length: number; width: number; angleDeg: number; moveDistance: number },
      origin: { x: number; y: number },
      aim: { x: number; y: number },
      color: number,
      fillAlpha: number,
      strokeAlpha: number,
    ) => {
      const stroke = { color, width: 0.1, alpha: strokeAlpha };
      const dir = Math.atan2(aim.y - origin.y, aim.x - origin.x);
      if (footprint.shape === "CIRCLE") {
        g.circle(aim.x, aim.y, footprint.radius).fill({ color, alpha: fillAlpha });
        g.circle(aim.x, aim.y, footprint.radius).stroke(stroke);
      } else if (footprint.shape === "CONE") {
        const half = (footprint.angleDeg * Math.PI) / 360;
        g.moveTo(origin.x, origin.y)
          .lineTo(origin.x + Math.cos(dir - half) * footprint.length, origin.y + Math.sin(dir - half) * footprint.length)
          .arc(origin.x, origin.y, footprint.length, dir - half, dir + half)
          .lineTo(origin.x, origin.y)
          .fill({ color, alpha: fillAlpha })
          .stroke(stroke);
      } else if (footprint.shape === "LINE") {
        const half = footprint.width / 2;
        const nx = -Math.sin(dir) * half;
        const ny = Math.cos(dir) * half;
        const ex = origin.x + Math.cos(dir) * footprint.length;
        const ey = origin.y + Math.sin(dir) * footprint.length;
        g.moveTo(origin.x + nx, origin.y + ny)
          .lineTo(ex + nx, ey + ny)
          .lineTo(ex - nx, ey - ny)
          .lineTo(origin.x - nx, origin.y - ny)
          .closePath()
          .fill({ color, alpha: fillAlpha })
          .stroke(stroke);
      } else if (footprint.moveDistance > 0) {
        g.moveTo(origin.x, origin.y)
          .lineTo(origin.x + Math.cos(dir) * footprint.moveDistance, origin.y + Math.sin(dir) * footprint.moveDistance)
          .stroke({ color, width: 0.2, alpha: strokeAlpha });
        g.circle(origin.x + Math.cos(dir) * footprint.moveDistance, origin.y + Math.sin(dir) * footprint.moveDistance, 0.45).stroke(stroke);
      } else {
        g.circle(aim.x, aim.y, 0.75).fill({ color, alpha: fillAlpha });
        g.circle(aim.x, aim.y, 0.75).stroke(stroke);
      }
    };

    const spawnBurst = (x: number, y: number, radius: number, color: number) => {
      if (!actorLayer) return;
      const node = new Graphics();
      actorLayer.addChild(node);
      bursts.push({ node, life: 1, x, y, radius, color });
    };

    function spawnEvents(frame: PvPFrame) {
      if (!overlayLayer) return;
      for (const event of frame.events) {
        const label =
          event.type === "hit" && event.damage
            ? `-${Math.round(event.damage)}`
            : event.type === "shield_absorb" && event.amount
              ? `🛡 ${Math.round(event.amount)}`
              : event.type === "shield" && event.amount
                ? `+${Math.round(event.amount)}`
                : null;
        if (label) {
          const node = new Text({
            text: label,
            style: { fontSize: 15, fill: event.type === "hit" ? 0xfca5a5 : 0x93c5fd, fontWeight: "700" },
          });
          node.anchor.set(0.5);
          overlayLayer.addChild(node);
          floats.push({ node, life: 1, x: event.x, y: event.y, rise: 0 });
        }

        // Impact / pop rings, so a landed hit reads as a hit even without art.
        switch (event.type) {
          case "hit":
            spawnBurst(event.x, event.y, 1.1, COLORS.remote);
            break;
          case "shield":
          case "shield_absorb":
            spawnBurst(event.x, event.y, 1.3, COLORS.shield);
            break;
          case "blink":
            spawnBurst(event.x, event.y, 1.4, COLORS.indicator);
            break;
          default:
            break;
        }
      }
    }

    const unsubscribeStore = useGameStore.subscribe((state) => {
      const frame = state.pvpFrame;
      if (!frame) return;
      const buffer = bufferRef.current;
      buffer.prev = buffer.curr;
      buffer.curr = frame;
      buffer.receivedAt = performance.now();

      // Reconcile the local prediction with the authoritative position.
      const authoritative = frame.participants.find((p) => p.id === localPlayerId);
      if (authoritative) {
        const local = localPosRef.current;
        if (!local) {
          localPosRef.current = { x: authoritative.x, y: authoritative.y };
        } else {
          const dx = authoritative.x - local.x;
          const dy = authoritative.y - local.y;
          if (Math.hypot(dx, dy) > SNAP_TILES) {
            local.x = authoritative.x;
            local.y = authoritative.y;
          } else {
            local.x += dx * RECONCILE_RATE;
            local.y += dy * RECONCILE_RATE;
          }
        }
      }

      spawnEvents(frame);
    });

    const init = async () => {
      const instance = new Application();
      await instance.init({
        resizeTo: host,
        background: COLORS.background,
        antialias: true,
        resolution: Math.min(window.devicePixelRatio || 1, 2),
        autoDensity: true,
      });
      if (destroyed) {
        instance.destroy();
        return;
      }
      app = instance;
      instance.canvas.style.display = "block";
      host.appendChild(instance.canvas);

      const root = new Container();
      instance.stage.addChild(root);

      // Static floor and geometry, drawn once in arena coordinates.
      const floor = new Graphics();
      floor.rect(0, 0, setup.width, setup.height).fill({ color: COLORS.floor });
      for (let x = 1; x < setup.width; x += 2) {
        floor.moveTo(x, 0).lineTo(x, setup.height);
      }
      for (let y = 1; y < setup.height; y += 2) {
        floor.moveTo(0, y).lineTo(setup.width, y);
      }
      floor.stroke({ color: COLORS.grid, width: 0.04, alpha: 0.6 });
      root.addChild(floor);

      const geometry = new Graphics();
      for (const obstacle of setup.obstacles) {
        if (obstacle.kind === "circle") {
          geometry.circle(obstacle.x, obstacle.y, obstacle.r);
        } else {
          geometry.rect(obstacle.x, obstacle.y, obstacle.w, obstacle.h);
        }
      }
      geometry.fill({ color: COLORS.obstacleFill }).stroke({ color: COLORS.obstacleStroke, width: 0.12 });
      root.addChild(geometry);

      projectileLayer = new Container();
      actorLayer = new Container();
      const indicatorLayer = new Container();
      root.addChild(projectileLayer, actorLayer, indicatorLayer);

      // Unscaled: names, HP bars and combat text stay a constant pixel size.
      overlayLayer = new Container();
      instance.stage.addChild(overlayLayer);

      indicator = new Graphics();
      indicatorLayer.addChild(indicator);

      instance.ticker.add(() => {
        const buffer = bufferRef.current;
        const current = buffer.curr;
        if (!current) return;

        // Fit the arena into the canvas, centred.
        const scale = Math.min(instance.screen.width / setup.width, instance.screen.height / setup.height);
        const offsetX = (instance.screen.width - setup.width * scale) / 2;
        const offsetY = (instance.screen.height - setup.height * scale) / 2;
        root.scale.set(scale);
        root.position.set(offsetX, offsetY);
        if (controlRef.current) {
          controlRef.current.view = { scale, offsetX, offsetY };
        }
        const toScreenX = (x: number) => offsetX + x * scale;
        const toScreenY = (y: number) => offsetY + y * scale;

        const deltaMs = instance.ticker.deltaMS;
        const alpha = Math.max(0, Math.min(1, (performance.now() - buffer.receivedAt) / setup.tick_ms));
        const control = controlRef.current;

        // Advance the local prediction.
        const local = localPosRef.current;
        if (local && control && (control.move.dx !== 0 || control.move.dy !== 0)) {
          const localState = current.participants.find((p) => p.id === localPlayerId);
          const speed = localState ? localState.speed * tilesPerSecondPerSpeed : 0;
          const dt = deltaMs / 1000;
          const next = slide(local.x, local.y, control.move.dx * speed * dt, control.move.dy * speed * dt);
          local.x = next.x;
          local.y = next.y;
        }

        const positions = new Map<string, { x: number; y: number; state: PvPParticipantState }>();
        for (const participant of current.participants) {
          let x = participant.x;
          let y = participant.y;
          if (participant.id === localPlayerId && local) {
            x = local.x;
            y = local.y;
          } else {
            const previous = buffer.prev?.participants.find((p) => p.id === participant.id);
            if (previous) {
              x = previous.x + (participant.x - previous.x) * alpha;
              y = previous.y + (participant.y - previous.y) * alpha;
            }
          }
          positions.set(participant.id, { x, y, state: participant });
        }

        // Actors: body/sprite in world space, readout in screen space.
        const seen = new Set<string>();
        for (const [id, entry] of positions) {
          seen.add(id);
          let actor = actors.get(id);
          if (!actor) {
            const body = new Graphics();
            const label = new Text({
              text: entry.state.name,
              style: { fontSize: 12, fill: 0xe5e7eb, fontWeight: "600" },
            });
            label.anchor.set(0.5);
            const bar = new Graphics();
            actorLayer!.addChild(body);
            overlayLayer!.addChild(label, bar);

            // Fighters get a portrait sprite; the training dummy is drawn, so
            // the practice arena needs no additional artwork.
            let sprite: Sprite | null = null;
            if (!entry.state.is_dummy) {
              sprite = new Sprite();
              sprite.anchor.set(0.5, 0.78);
              actorLayer!.addChild(sprite);
              const url = id === localPlayerId && playerAvatarUrl ? playerAvatarUrl : DEFAULT_CULTIVATOR;
              Assets.load<Texture>(url)
                .then((texture) => {
                  if (sprite!.destroyed) return;
                  sprite!.texture = texture;
                  sprite!.width = 1.5;
                  sprite!.height = 1.95;
                })
                .catch(() => {
                  // Keep the ring-only marker when the sprite is unavailable.
                });
            }
            actor = { body, sprite, label, bar };
            actors.set(id, actor);
          }

          const radius = 0.6;
          const color = id === localPlayerId ? COLORS.local : COLORS.remote;
          actor.body.clear();

          if (entry.state.is_dummy) {
            // Wooden training dummy: post, crossbar, head.
            actor.body.rect(entry.x - 0.16, entry.y - 0.75, 0.32, 1.5).fill({ color: COLORS.woodPost });
            actor.body.rect(entry.x - 0.62, entry.y - 0.42, 1.24, 0.24).fill({ color: COLORS.woodBar });
            actor.body.circle(entry.x, entry.y - 0.95, 0.28).fill({ color: COLORS.woodHead });
            actor.body.circle(entry.x, entry.y, radius).stroke({ color, width: 0.1, alpha: 0.75 });
          } else {
            // Team ring under the portrait plus a facing pip.
            actor.body.circle(entry.x, entry.y, radius).fill({ color, alpha: 0.22 });
            actor.body.circle(entry.x, entry.y, radius).stroke({ color, width: 0.1 });
            actor.body
              .circle(entry.x + Math.cos(entry.state.facing) * radius, entry.y + Math.sin(entry.state.facing) * radius, 0.16)
              .fill({ color: 0x0b1410, alpha: 0.9 });
          }
          if (actor.sprite) actor.sprite.position.set(entry.x, entry.y);

          const screenX = toScreenX(entry.x);
          const screenY = toScreenY(entry.y);
          const bodyRadiusPx = radius * scale;
          actor.label.position.set(screenX, screenY - bodyRadiusPx - 9);

          const barX = screenX - BAR_WIDTH / 2;
          const barY = screenY + bodyRadiusPx + 3;
          const hpRatio = entry.state.max_hp > 0 ? Math.max(0, entry.state.hp / entry.state.max_hp) : 0;
          actor.bar.clear();
          actor.bar.rect(barX, barY, BAR_WIDTH, BAR_HEIGHT).fill({ color: COLORS.hpBack });
          actor.bar.rect(barX, barY, BAR_WIDTH * hpRatio, BAR_HEIGHT).fill({ color: COLORS.hpFill });
          if (entry.state.shield > 0) {
            actor.bar.rect(barX, barY + BAR_HEIGHT + 1, BAR_WIDTH, 3).fill({ color: COLORS.shield, alpha: 0.85 });
          }
          if (entry.state.cast) {
            actor.bar.rect(barX, barY - 7, BAR_WIDTH, 3).fill({ color: COLORS.cast });
          }
        }
        for (const [id, actor] of actors) {
          if (seen.has(id)) continue;
          actor.body.destroy();
          actor.sprite?.destroy({ texture: false, textureSource: false });
          actor.label.destroy();
          actor.bar.destroy();
          actors.delete(id);
        }

        // Projectiles: an oriented bolt with a fading tail.
        const seenProjectiles = new Set<number>();
        for (const projectile of current.projectiles) {
          seenProjectiles.add(projectile.id);
          let node = projectiles.get(projectile.id);
          if (!node) {
            node = new Graphics();
            node.roundRect(-0.95, -0.1, 1.15, 0.2, 0.1).fill({ color: COLORS.projectile, alpha: 0.35 });
            node.roundRect(-0.45, -0.17, 0.95, 0.34, 0.17).fill({ color: COLORS.projectile });
            node.circle(0.2, 0, 0.24).fill({ color: COLORS.boltCore });
            projectileLayer!.addChild(node);
            projectiles.set(projectile.id, node);
          }
          node.position.set(projectile.x, projectile.y);
          node.rotation = Math.atan2(projectile.dy, projectile.dx);
        }
        for (const [id, node] of projectiles) {
          if (seenProjectiles.has(id)) continue;
          node.destroy();
          projectiles.delete(id);
        }

        // Telegraphs. Every participant's winding-up cast comes from the frame,
        // so both sides see the same footprint before it lands. The fill grows
        // with the wind-up progress so the window to step out is readable.
        indicator!.clear();
        const approxTick = current.tick + (performance.now() - buffer.receivedAt) / setup.tick_ms;

        for (const [id, entry] of positions) {
          const cast = entry.state.cast;
          if (!cast) continue;
          const isLocal = id === localPlayerId;
          const color = isLocal ? COLORS.indicator : COLORS.remote;
          const span = Math.max(1, cast.ends_tick - cast.start_tick);
          const progress = Math.max(0, Math.min(1, (approxTick - cast.start_tick) / span));
          const fillAlpha = (isLocal ? 0.16 : 0.12) + progress * 0.3;
          const strokeAlpha = (isLocal ? 0.55 : 0.5) + progress * 0.45;

          // Charging ring on the caster: makes the wind-up visible on the
          // actor itself, not just on the ground.
          const pulse = 0.6 + 0.4 * Math.sin(progress * Math.PI * 6);
          indicator!.circle(entry.x, entry.y, 0.82 + progress * 0.35).stroke({
            color,
            width: 0.13,
            alpha: 0.25 + 0.6 * pulse,
          });

          if (cast.target_id) {
            // Point-and-click cannot be dodged, so mark the victim instead of
            // pretending there is an area to step out of.
            const victim = positions.get(cast.target_id);
            if (victim) {
              indicator!.circle(victim.x, victim.y, 0.95).stroke({ color, width: 0.14, alpha: strokeAlpha });
            }
            continue;
          }
          drawFootprint(indicator!, {
            shape: cast.shape,
            radius: cast.radius ?? 0,
            length: cast.length ?? 0,
            width: cast.width ?? 0,
            angleDeg: cast.angle_deg ?? 0,
            moveDistance: cast.move_distance ?? 0,
          }, entry, { x: cast.aim_x, y: cast.aim_y }, color, fillAlpha, strokeAlpha);
        }

        // Aim preview for the selected Gu, so reach and shape are readable
        // before committing. Hidden while one of our own casts is winding up.
        const origin = positions.get(localPlayerId);
        const skill = control?.skill;
        if (origin && skill && !origin.state.cast) {
          const aiming = Boolean(control?.aiming);
          const isCancel = Boolean(control?.cancelCast);
          const aimColor = isCancel ? COLORS.remote : COLORS.indicator;
          const fillAlpha = aiming ? (isCancel ? 0.35 : 0.3) : 0.14;
          const strokeAlpha = aiming ? 0.95 : 0.55;

          let aim = control!.aim;
          if (skill.cast_mode !== "DIRECTION") {
            const dx = aim.x - origin.x;
            const dy = aim.y - origin.y;
            const distance = Math.hypot(dx, dy);
            if (distance > skill.cast_range) {
              aim = { x: origin.x + (dx / distance) * skill.cast_range, y: origin.y + (dy / distance) * skill.cast_range };
            }
          }
          // Reach ring: shows how far the cast can go.
          if (skill.cast_mode === "TARGET" || skill.cast_mode === "GROUND") {
            indicator!.circle(origin.x, origin.y, skill.cast_range).stroke({
              color: isCancel ? COLORS.remote : COLORS.rangeRing,
              width: 0.07,
              alpha: isCancel ? 0.6 : 0.35,
            });
          }
          drawFootprint(indicator!, {
            shape: skill.shape,
            radius: skill.radius,
            length: skill.length,
            width: skill.width,
            angleDeg: skill.angle_deg,
            moveDistance: skill.move_distance,
          }, origin, aim, aimColor, fillAlpha, strokeAlpha);
        }

        // Impact and pop rings.
        for (let index = bursts.length - 1; index >= 0; index--) {
          const burst = bursts[index];
          burst.life -= deltaMs / 420;
          if (burst.life <= 0) {
            burst.node.destroy();
            bursts.splice(index, 1);
            continue;
          }
          const expansion = 1 - burst.life;
          burst.node.clear();
          burst.node
            .circle(burst.x, burst.y, burst.radius * (0.35 + expansion * 0.85))
            .stroke({ color: burst.color, width: 0.16, alpha: burst.life });
        }

        // Floating combat text, converted from tiles to pixels.
        for (let index = floats.length - 1; index >= 0; index--) {
          const float = floats[index];
          float.life -= deltaMs / 900;
          if (float.life <= 0) {
            float.node.destroy();
            floats.splice(index, 1);
            continue;
          }
          float.rise += (deltaMs / 900) * 1.6;
          float.node.position.set(toScreenX(float.x), toScreenY(float.y - float.rise));
          float.node.alpha = Math.min(1, float.life * 2);
        }
      });
    };

    void init();

    return () => {
      destroyed = true;
      unsubscribeStore();
      app?.destroy(true, { children: true });
      app = null;
    };
  }, [setup, localPlayerId, controlRef, playerAvatarUrl]);

  return <div ref={hostRef} className="h-full w-full" />;
};
