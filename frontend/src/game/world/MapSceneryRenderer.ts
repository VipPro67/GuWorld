import Phaser from "phaser";
import { LANDSCAPE_ASSETS } from "../assets/LandscapeAssets.ts";
import { PIXELS_PER_UNIT, worldToPixels } from "./WorldUnits.ts";
import type { MapSceneryObject } from "./MapLoader.ts";

const SPATIAL_CELL_SIZE = 48;
const SYNC_INTERVAL_MS = 160;

/** Renders authored scenery props with a one-viewport buffer and a reusable image pool. */
export class MapSceneryRenderer {
  private readonly scene: Phaser.Scene;
  private readonly active = new Map<number, Phaser.GameObjects.Image>();
  private readonly pool: Phaser.GameObjects.Image[] = [];
  private objects: MapSceneryObject[] = [];
  private cells = new Map<string, number[]>();
  private unknownAssetIds = new Set<string>();
  private lastSyncAt = -SYNC_INTERVAL_MS;

  constructor(scene: Phaser.Scene) {
    this.scene = scene;
  }

  setObjects(objects: MapSceneryObject[]): void {
    for (const [index, image] of this.active) {
      image.setVisible(false);
      image.setActive(false);
      this.pool.push(image);
      this.active.delete(index);
    }
    this.objects = objects;
    this.cells = new Map();
    this.unknownAssetIds.clear();
    objects.forEach((object, index) => {
      const key = cellKey(Math.floor(object.x / SPATIAL_CELL_SIZE), Math.floor(object.y / SPATIAL_CELL_SIZE));
      const bucket = this.cells.get(key) ?? [];
      bucket.push(index);
      this.cells.set(key, bucket);
    });
    this.lastSyncAt = -SYNC_INTERVAL_MS;
  }

  sync(view: Phaser.Geom.Rectangle, now: number): void {
    if (now - this.lastSyncAt < SYNC_INTERVAL_MS) return;
    this.lastSyncAt = now;
    const marginX = view.width / PIXELS_PER_UNIT;
    const marginY = view.height / PIXELS_PER_UNIT;
    const minX = Math.max(0, view.x / PIXELS_PER_UNIT - marginX);
    const minY = Math.max(0, view.y / PIXELS_PER_UNIT - marginY);
    const maxX = (view.right + view.width) / PIXELS_PER_UNIT;
    const maxY = (view.bottom + view.height) / PIXELS_PER_UNIT;
    const minCellX = Math.floor(minX / SPATIAL_CELL_SIZE);
    const minCellY = Math.floor(minY / SPATIAL_CELL_SIZE);
    const maxCellX = Math.floor(maxX / SPATIAL_CELL_SIZE);
    const maxCellY = Math.floor(maxY / SPATIAL_CELL_SIZE);
    const wanted = new Set<number>();
    for (let cellY = minCellY; cellY <= maxCellY; cellY += 1) {
      for (let cellX = minCellX; cellX <= maxCellX; cellX += 1) {
        for (const index of this.cells.get(cellKey(cellX, cellY)) ?? []) {
          const object = this.objects[index];
          if (object.x >= minX && object.x <= maxX && object.y >= minY && object.y <= maxY) wanted.add(index);
        }
      }
    }

    for (const [index, image] of this.active) {
      if (wanted.has(index)) continue;
      image.setVisible(false);
      image.setActive(false);
      this.pool.push(image);
      this.active.delete(index);
    }
    for (const index of wanted) {
      if (this.active.has(index)) continue;
      const object = this.objects[index];
      const definition = LANDSCAPE_ASSETS[object.assetId];
      const textureKey = `landscape-prop-${object.assetId}`;
      if ((!definition || !this.scene.textures.exists(textureKey)) && !this.unknownAssetIds.has(object.assetId)) {
        this.unknownAssetIds.add(object.assetId);
        const reason = definition ? "texture is unavailable" : "asset_id is unknown";
        console.warn(`[MapScenery] ${reason} for "${object.assetId}" in map scenery; rendering placeholder.`);
      }
      const texture = definition && this.scene.textures.exists(textureKey) ? textureKey : "landscape-placeholder";
      const pixels = worldToPixels(object);
      const image = this.pool.pop() ?? this.scene.add.image(pixels.x, pixels.y, texture);
      image.setTexture(texture);
      image.setPosition(pixels.x, pixels.y);
      image.setOrigin(0.5, definition?.originY ?? 0.88);
      image.setScale(object.scale * (definition?.renderScale ?? 1));
      image.setRotation(Phaser.Math.DegToRad(object.rotation));
      image.setDepth(pixels.y - 0.1);
      image.setVisible(true);
      image.setActive(true);
      this.active.set(index, image);
    }
  }

  destroy(): void {
    for (const image of [...this.active.values(), ...this.pool]) image.destroy();
    this.active.clear();
    this.pool.length = 0;
    this.objects = [];
    this.cells.clear();
  }
}

function cellKey(x: number, y: number): string {
  return `${x}:${y}`;
}
