import { PIXELS_PER_UNIT } from "./WorldUnits.ts";

export interface MapSpawn {
  name: string;
  kind: string;
  x: number;
  y: number;
}

export interface MapLayerJSON {
  type: string;
  name: string;
  width?: number;
  height?: number;
  data?: number[];
  objects?: Array<{
    name?: string;
    type?: string;
    x: number;
    y: number;
    width?: number;
    height?: number;
    properties?: Array<{ name: string; value: unknown }>;
  }>;
}

export interface MapPortal {
  name: string;
  targetZoneId: string;
  targetInstanceId: string;
  x: number;
  y: number;
  targetX: number;
  targetY: number;
}

export interface LegacyRegion { name: string; x: number; y: number; width: number; height: number; kind: string }
export interface LegacyPOI { name: string; type: string; x: number; y: number }

export interface WorldMapJSON {
  type?: string;
  orientation: string;
  width: number;
  height: number;
  tilewidth: number;
  tileheight: number;
  map_id: string;
  map_version: string;
  content_version?: string;
  layers: MapLayerJSON[];
  properties?: Array<{ name: string; value: unknown }>;
}

/**
 * Client view of the shared Tiled map artifact. One cell equals one world unit;
 * PIXELS_PER_UNIT is applied only when drawing. The same JSON is validated by
 * the Go server, and a golden test keeps both copies identical.
 */
export class LoadedWorldMap {
  public readonly id: string;
  public readonly version: string;
  public readonly contentVersion: string;
  public readonly width: number;
  public readonly height: number;
  public readonly pixelWidth: number;
  public readonly pixelHeight: number;
  public readonly ground: number[];
  public readonly solid: boolean[];
  public readonly spawns: MapSpawn[];
  public readonly portals: MapPortal[];
  public readonly legacyRegions: LegacyRegion[];
  public readonly legacyPOIs: LegacyPOI[];

  public constructor(raw: WorldMapJSON) {
    this.id = raw.map_id;
    this.version = raw.map_version;
    this.contentVersion = raw.content_version ?? "";
    this.width = raw.width;
    this.height = raw.height;
    this.pixelWidth = raw.width * PIXELS_PER_UNIT;
    this.pixelHeight = raw.height * PIXELS_PER_UNIT;
    this.ground = [];
    this.solid = [];
    this.spawns = [];
    this.portals = [];
    this.legacyRegions = [];
    this.legacyPOIs = [];
    let hasCollision = false;
    for (const layer of raw.layers) {
      if (!layer || typeof layer !== "object" || typeof layer.type !== "string" || typeof layer.name !== "string") {
        throw new Error("Map layer is invalid");
      }
      if (layer.type === "tilelayer" && layer.name === "collision") {
        if (!Array.isArray(layer.data) || (layer.data.length !== 0 && layer.data.length !== raw.width * raw.height)) {
          throw new Error("Map collision layer has the wrong size");
        }
        hasCollision = true;
        if (layer.data.length === raw.width * raw.height) {
          this.solid = layer.data.map((value) => {
            if (value !== 0 && value !== 1) throw new Error("Map collision cells must be 0 or 1");
            return value === 1;
          });
        } else {
          // Sparse / Procedural border collision: empty array avoids allocating 4M booleans in RAM
          this.solid = [];
        }
      } else if (layer.type === "tilelayer" && layer.name === "ground") {
        if (Array.isArray(layer.data) && layer.data.length === raw.width * raw.height) {
          this.ground = [...layer.data];
        } else {
          this.ground = [];
        }
      } else if (layer.type === "objectgroup" && layer.name === "portals" && Array.isArray(layer.objects)) {
        for (const object of layer.objects) {
          const targetZone = propertyString(object.properties, "target_zone_id");
          const targetX = propertyNumber(object.properties, "target_x");
          const targetY = propertyNumber(object.properties, "target_y");
          if (!targetZone || targetX === null || targetY === null || targetX < 0 || targetY < 0) {
            throw new Error("Map portal is missing target metadata");
          }
          if (object.x < 0 || object.y < 0 || object.x >= raw.width || object.y >= raw.height) {
            throw new Error("Map portal is outside the map");
          }
          this.portals.push({
            name: object.name ?? "", targetZoneId: targetZone,
            targetInstanceId: propertyString(object.properties, "target_instance_id") ?? "",
            x: object.x, y: object.y, targetX, targetY,
          });
        }
      } else if (layer.type === "objectgroup" && layer.name === "spawns" && Array.isArray(layer.objects)) {
        for (const object of layer.objects) {
          if (typeof object.type !== "string" || !object.type) continue;
          if (!Number.isFinite(object.x) || !Number.isFinite(object.y)) continue;
          if (object.x < 0 || object.y < 0 || object.x >= raw.width || object.y >= raw.height) {
            throw new Error("Map spawn is outside the map");
          }
          this.spawns.push({ name: object.name ?? "", kind: object.type, x: object.x, y: object.y });
        }
      } else if (layer.type === "objectgroup" && layer.name === "legacy_regions" && Array.isArray(layer.objects)) {
        for (const object of layer.objects) {
          if (object.width && object.height && object.width > 0 && object.height > 0) {
            this.legacyRegions.push({ name: object.name ?? "", x: object.x, y: object.y,
              width: object.width, height: object.height, kind: propertyString(object.properties, "zone_type") ?? "KHU_VUC" });
          }
        }
      } else if (layer.type === "objectgroup" && layer.name === "legacy_pois" && Array.isArray(layer.objects)) {
        for (const object of layer.objects) {
          this.legacyPOIs.push({ name: object.name ?? "", type: object.type ?? "poi", x: object.x, y: object.y });
        }
      }
    }
    if (!hasCollision) throw new Error("Map is missing the collision layer");
  }

  /** Outside the map is solid so a click cannot request an out-of-bounds path. */
  public isSolid(worldX: number, worldY: number): boolean {
    if (!Number.isFinite(worldX) || !Number.isFinite(worldY)) return true;
    const x = Math.floor(worldX);
    const y = Math.floor(worldY);
    if (x < 0 || y < 0 || x >= this.width || y >= this.height) return true;
    if (this.solid.length === 0) {
      // Procedural border collision: borders are solid, interior is walkable
      return x === 0 || x === this.width - 1 || y === 0 || y === this.height - 1;
    }
    return this.solid[y * this.width + x] === true;
  }

  public isSpawnFree(spawn: MapSpawn): boolean {
    return !this.isSolid(spawn.x, spawn.y);
  }
}

export function parseWorldMap(raw: unknown): LoadedWorldMap {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) throw new Error("Map JSON must be an object");
  const value = raw as Record<string, unknown>;
  if (value.orientation !== "orthogonal") throw new Error("Only orthogonal maps are supported");
  if (!isPositiveInt(value.width) || !isPositiveInt(value.height)) throw new Error("Map dimensions are invalid");
  if (value.width * value.height > 16_777_216) throw new Error("Map is too large");
  if (!isPositiveInt(value.tilewidth) || !isPositiveInt(value.tileheight)) throw new Error("Map tile size is invalid");
  if (!isBoundedString(value.map_id) || !isBoundedString(value.map_version)) throw new Error("Map id/version is missing");
  if (!Array.isArray(value.layers)) throw new Error("Map layers are missing");
  return new LoadedWorldMap({
    type: typeof value.type === "string" ? value.type : undefined,
    orientation: "orthogonal",
    width: value.width,
    height: value.height,
    tilewidth: value.tilewidth,
    tileheight: value.tileheight,
    map_id: value.map_id,
    map_version: value.map_version,
    content_version: typeof value.content_version === "string" ? value.content_version : undefined,
    layers: value.layers as MapLayerJSON[],
    properties: Array.isArray(value.properties) ? value.properties as Array<{ name: string; value: unknown }> : undefined,
  });
}

export async function loadWorldMap(url: string, fetcher: typeof fetch = fetch): Promise<LoadedWorldMap> {
  const response = await fetcher(url, { cache: "no-cache" });
  if (!response.ok) throw new Error(`Map asset ${url} returned ${response.status}`);
  return parseWorldMap(await response.json());
}

/**
 * Computes the same SHA-256 as the Go nav loader: map id/version/dimensions,
 * then one byte per collision cell. Used to validate `world.welcome.map_hash`.
 */
export async function computeWorldMapHash(map: LoadedWorldMap): Promise<string | null> {
  if (typeof crypto === "undefined" || !crypto.subtle) return null;
  const prefix = new TextEncoder().encode(
    `${map.id}\u0000${map.version}\u0000${map.width}\u0000${map.height}\u0000`,
  );
  const total = map.width * map.height;
  const bytes = new Uint8Array(prefix.length + total);
  bytes.set(prefix);
  const offset = prefix.length;
  if (map.solid.length === total) {
    for (let index = 0; index < total; index += 1) {
      bytes[offset + index] = map.solid[index] ? 1 : 0;
    }
  } else {
    // Procedural border collision: borders are 1, interior is 0 (Uint8Array is 0 by default)
    for (let x = 0; x < map.width; x += 1) {
      bytes[offset + x] = 1;
      bytes[offset + (map.height - 1) * map.width + x] = 1;
    }
    for (let y = 0; y < map.height; y += 1) {
      bytes[offset + y * map.width] = 1;
      bytes[offset + y * map.width + map.width - 1] = 1;
    }
  }
  const digest = await crypto.subtle.digest("SHA-256", bytes);
  return [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, "0")).join("");
}

function propertyString(properties: Array<{ name: string; value: unknown }> | undefined, name: string): string | null {
  const property = properties?.find((entry) => entry.name === name);
  return typeof property?.value === "string" && property.value.length > 0 ? property.value : null;
}

function propertyNumber(properties: Array<{ name: string; value: unknown }> | undefined, name: string): number | null {
  const property = properties?.find((entry) => entry.name === name);
  return typeof property?.value === "number" && Number.isFinite(property.value) ? property.value : null;
}

function isPositiveInt(value: unknown): value is number {
  return typeof value === "number" && Number.isSafeInteger(value) && value > 0;
}

function isBoundedString(value: unknown): value is string {
  return typeof value === "string" && value.length > 0 && value.length <= 80 && value.trim() === value;
}
