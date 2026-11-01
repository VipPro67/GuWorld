import type Phaser from "phaser";
import { PIXELS_PER_UNIT } from "../WorldUnits.ts";
import type { WorldMarker } from "./types.ts";

export interface MarkerLabelStyle {
  size: string;
  color: string;
  background: string;
}

export interface MarkerDrawContext {
  graphics: Phaser.GameObjects.Graphics;
  addLabel: (x: number, y: number, text: string, style: MarkerLabelStyle) => void;
}

function drawVein(ctx: MarkerDrawContext, marker: WorldMarker): void {
  const g = ctx.graphics;
  const px = marker.x * PIXELS_PER_UNIT;
  const py = marker.y * PIXELS_PER_UNIT;
  const immortal = marker.vein?.vein_type === "immortal";

  g.fillStyle(immortal ? 0xf59e0b : 0x10b981, 0.22);
  g.fillCircle(px, py - 6, 22);
  g.lineStyle(2, immortal ? 0xfde047 : 0x34d399, 0.9);
  g.strokeCircle(px, py - 6, 18);
  g.fillStyle(immortal ? 0xfef08a : 0xa7f3d0, 0.95);
  g.fillCircle(px, py - 6, 5);

  const rarity = marker.vein?.rarity || "Thường";
  const stones = marker.vein?.accumulated_stones?.toLocaleString("vi-VN") ?? 0;
  ctx.addLabel(px, py - 32, `💠 ${marker.name} (${rarity})\n${stones} NT`, {
    size: "10px",
    color: immortal ? "#fef08a" : "#a7f3d0",
    background: "#064e3bdd",
  });
}

function drawSect(ctx: MarkerDrawContext, marker: WorldMarker): void {
  const g = ctx.graphics;
  const sx = marker.x * PIXELS_PER_UNIT;
  const sy = marker.y * PIXELS_PER_UNIT;
  const isMine = marker.sect?.isMine ?? false;
  const level = marker.sect?.level ?? 1;
  const shielded = Boolean(marker.sect?.shieldUntil && marker.sect.shieldUntil > Date.now());
  const territoryRadius = (marker.radiusCells ?? 5 + level) * PIXELS_PER_UNIT;

  g.fillStyle(isMine ? 0x3b82f6 : 0xd97706, 0.08);
  g.fillRect(sx - territoryRadius, sy - territoryRadius, territoryRadius * 2, territoryRadius * 2);
  g.lineStyle(2, isMine ? 0x60a5fa : 0xf59e0b, 0.45);
  g.strokeRect(sx - territoryRadius, sy - territoryRadius, territoryRadius * 2, territoryRadius * 2);

  const size = 30;
  g.fillStyle(isMine ? 0x2563eb : 0xd97706, 0.28);
  g.fillCircle(sx, sy - 8, size + 8);

  g.fillStyle(0x18181b, 0.95);
  g.fillRect(sx - size / 2, sy - size / 4, size, size * 0.75);
  g.lineStyle(2, isMine ? 0x3b82f6 : 0xf59e0b, 0.9);
  g.strokeRect(sx - size / 2, sy - size / 4, size, size * 0.75);

  g.fillStyle(isMine ? 0x1d4ed8 : 0xb45309, 0.95);
  g.fillTriangle(sx, sy - size * 0.75, sx - size * 0.8, sy - size * 0.25, sx + size * 0.8, sy - size * 0.25);
  g.fillStyle(isMine ? 0x2563eb : 0xd97706, 0.95);
  g.fillTriangle(sx, sy - size * 1.25, sx - size * 0.55, sy - size * 0.7, sx + size * 0.55, sy - size * 0.7);
  g.fillStyle(0xfde047, 1);
  g.fillCircle(sx, sy - size * 1.35, 4);

  if (shielded) {
    g.lineStyle(3, 0xf43f5e, 0.85);
    g.strokeCircle(sx, sy - 8, size + 14);
  }

  ctx.addLabel(sx, sy - size * 1.6, `🏯 ${marker.name} (Cấp ${level})${shielded ? " 🛡️" : ""}`, {
    size: "11px",
    color: isMine ? "#bfdbfe" : "#fde68a",
    background: isMine ? "#1e3a8add" : "#451a03dd",
  });
}

function drawDrop(ctx: MarkerDrawContext, marker: WorldMarker): void {
  const g = ctx.graphics;
  const dx = marker.x * PIXELS_PER_UNIT;
  const dy = marker.y * PIXELS_PER_UNIT;

  g.fillStyle(0xb45309, 0.85);
  g.fillCircle(dx, dy - 4, 8);
  g.fillStyle(0xfde047, 1);
  g.fillCircle(dx, dy - 8, 3);
  g.lineStyle(1, 0xfef08a, 0.9);
  g.strokeCircle(dx, dy - 4, 8);

  ctx.addLabel(dx, dy - 20, `🎁 ${marker.name}`, {
    size: "9px",
    color: "#fef08a",
    background: "#78350fdd",
  });
}

function drawTeleport(ctx: MarkerDrawContext, marker: WorldMarker): void {
  const g = ctx.graphics;
  const tx = marker.x * PIXELS_PER_UNIT;
  const ty = marker.y * PIXELS_PER_UNIT;

  g.fillStyle(0x0284c7, 0.2);
  g.fillCircle(tx, ty - 4, 22);
  g.lineStyle(2, 0x38bdf8, 0.85);
  g.strokeCircle(tx, ty - 4, 18);
  g.fillStyle(0xbae6fd, 0.9);
  g.fillCircle(tx, ty - 4, 4);

  ctx.addLabel(tx, ty - 28, `🌀 ${marker.name}`, {
    size: "10px",
    color: "#e0f2fe",
    background: "#0369a1dd",
  });
}

function drawPortal(ctx: MarkerDrawContext, marker: WorldMarker): void {
  const g = ctx.graphics;
  const px = marker.x * PIXELS_PER_UNIT;
  const py = marker.y * PIXELS_PER_UNIT;

  g.fillStyle(0x0284c7, 0.22);
  g.fillCircle(px, py - 6, 26);
  g.lineStyle(2, 0x38bdf8, 0.9);
  g.strokeCircle(px, py - 6, 26);
  g.lineStyle(1, 0xbae6fd, 0.6);
  g.strokeCircle(px, py - 6, 30);
  g.fillStyle(0x38bdf8, 0.95);
  g.fillCircle(px, py - 6, 5);

  ctx.addLabel(px, py - 36, `🌀 Rìa → ${marker.name}`, {
    size: "11px",
    color: "#e0f2fe",
    background: "#0369a1dd",
  });
}

function drawResource(ctx: MarkerDrawContext, marker: WorldMarker): void {
  const g = ctx.graphics;
  const px = marker.x * PIXELS_PER_UNIT;
  const py = marker.y * PIXELS_PER_UNIT + 10; // offset below monster sprites on the same cell
  const famous = marker.resource?.point_class === "FAMOUS";
  const ready = marker.resource?.state === "ACTIVE" && (marker.resource?.stock_total ?? 0) > 0;
  const tint = famous ? 0xf59e0b : ready ? 0x10b981 : 0x64748b;

  g.fillStyle(tint, ready ? 0.24 : 0.12);
  g.fillCircle(px, py, 14);
  g.lineStyle(2, tint, ready ? 0.95 : 0.6);
  g.beginPath();
  g.moveTo(px, py - 9);
  g.lineTo(px + 9, py);
  g.lineTo(px, py + 9);
  g.lineTo(px - 9, py);
  g.closePath();
  g.strokePath();
  g.fillStyle(tint, ready ? 0.9 : 0.45);
  g.fillCircle(px, py, 3.5);

  const stock = marker.resource?.stock_total ?? 0;
  const vitality = marker.resource?.vitality ?? 100;
  const label = famous ? `💎 ${marker.name} ★` : `💎 ${marker.name}`;
  ctx.addLabel(px, py + 26, `${label}\n${stock} món · ${vitality}%`, {
    size: "9px",
    color: famous ? "#fde68a" : ready ? "#a7f3d0" : "#cbd5e1",
    background: famous ? "#78350fdd" : "#064e3bdd",
  });
}

export function drawWorldMarker(ctx: MarkerDrawContext, marker: WorldMarker): void {
  switch (marker.kind) {
    case "vein":
      drawVein(ctx, marker);
      break;
    case "sect":
      drawSect(ctx, marker);
      break;
    case "drop":
      drawDrop(ctx, marker);
      break;
    case "teleport":
      drawTeleport(ctx, marker);
      break;
    case "portal":
      drawPortal(ctx, marker);
      break;
    case "resource":
      drawResource(ctx, marker);
      break;
  }
}
