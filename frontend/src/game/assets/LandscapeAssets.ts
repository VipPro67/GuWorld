import Phaser from "phaser";
import landscapeManifest from "./landscape-assets.json";

export type LandscapeTheme = "nam_cuong" | "dong_hai" | "tay_mac" | "bac_nguyen" | "trung_chau";
type LandscapeShape = "tree" | "pine" | "rock" | "water" | "marker" | "palm" | "driftwood" | "dune" | "oasis" | "ruin" | "grass" | "yurt" | "firepit" | "flower" | "pillar" | "banner" | "bamboo";

interface Palette {
  ground: number;
  groundLights: number[];
  deep: number;
  mid: number;
  light: number;
  accent: number;
}

export interface LandscapeAssetDefinition {
  theme: LandscapeTheme;
  shape: LandscapeShape;
  label: string;
  width: number;
  height: number;
  renderScale?: number;
  originY?: number;
}

const PALETTES = landscapeManifest.themes as Record<LandscapeTheme, Palette>;
export const LANDSCAPE_ASSETS = landscapeManifest.assets as Record<string, LandscapeAssetDefinition>;

export function landscapeThemeForMap(mapId: string): LandscapeTheme | null {
  if (mapId === "tan_thu_coc_nam_cuong") return "nam_cuong";
  if (mapId === "tan_thu_coc_dong_hai") return "dong_hai";
  if (mapId === "tan_thu_coc_tay_mac") return "tay_mac";
  if (mapId === "tan_thu_coc_bac_nguyen") return "bac_nguyen";
  if (mapId === "tan_thu_coc_trung_chau") return "trung_chau";
  return null;
}

export function landscapeTextureKey(theme: LandscapeTheme): string {
  return `landscape-ground-${theme}`;
}

export function createLandscapeTextures(scene: Phaser.Scene): void {
  for (const theme of Object.keys(PALETTES) as LandscapeTheme[]) {
    createGroundTexture(scene, theme, PALETTES[theme]);
  }
  for (const [assetId, definition] of Object.entries(LANDSCAPE_ASSETS)) {
    createPropTexture(scene, `landscape-prop-${assetId}`, definition, PALETTES[definition.theme]);
  }
  createFallbackTexture(scene);
}

function createGroundTexture(scene: Phaser.Scene, theme: LandscapeTheme, palette: Palette): void {
  const key = landscapeTextureKey(theme);
  if (scene.textures.exists(key)) return;
  const size = 512;
  const graphics = scene.add.graphics();
  graphics.fillStyle(palette.ground, 1).fillRect(0, 0, size, size);
  // Large, low-contrast patches read as soil and vegetation instead of a field
  // of tiny noise when the camera is at the normal 1x zoom.
  let seed = theme.split("").reduce((value, char) => value * 31 + char.charCodeAt(0), 7) >>> 0;
  const random = (): number => {
    seed = (seed * 1664525 + 1013904223) >>> 0;
    return seed / 0x1_0000_0000;
  };
  for (let index = 0; index < 32; index += 1) {
    const x = 20 + Math.floor(random() * (size - 40));
    const y = 20 + Math.floor(random() * (size - 40));
    const color = palette.groundLights[Math.floor(random() * palette.groundLights.length)];
    const width = 22 + Math.floor(random() * 54);
    const height = 14 + Math.floor(random() * 34);
    graphics.fillStyle(color, 0.18 + random() * 0.10).fillEllipse(x, y, width, height);
  }
  for (let index = 0; index < 18; index += 1) {
    const x = Math.floor(random() * (size / 4)) * 4;
    const y = Math.floor(random() * (size / 4)) * 4;
    const color = index % 3 === 0 ? palette.deep : palette.groundLights[index % palette.groundLights.length];
    graphics.fillStyle(color, 0.24).fillEllipse(x, y, 8 + (index % 3) * 4, 5 + (index % 2) * 3);
  }
  graphics.generateTexture(key, size, size);
  graphics.destroy();
  scene.textures.get(key).setFilter(Phaser.Textures.FilterMode.NEAREST);
}

function createPropTexture(scene: Phaser.Scene, key: string, definition: LandscapeAssetDefinition, palette: Palette): void {
  if (scene.textures.exists(key)) return;
  const width = definition.width;
  const height = definition.height;
  const g = scene.add.graphics();
  const dark = palette.deep;
  const mid = palette.mid;
  const light = palette.light;
  const accent = palette.accent;
  const point = (x: number, y: number): Phaser.Math.Vector2 => new Phaser.Math.Vector2(x * width, y * height);
  const polygon = (points: Array<[number, number]>, color: number, alpha = 1): void => {
    g.fillStyle(color, alpha).fillPoints(points.map(([x, y]) => point(x, y)), true);
  };
  const ellipse = (x: number, y: number, w: number, h: number, color: number, alpha = 1): void => {
    g.fillStyle(color, alpha).fillEllipse(x * width, y * height, w * width, h * height);
  };
  const rect = (x: number, y: number, w: number, h: number, color: number, alpha = 1, radius = 0): void => {
    g.fillStyle(color, alpha);
    if (radius > 0) g.fillRoundedRect(x * width, y * height, w * width, h * height, radius);
    else g.fillRect(x * width, y * height, w * width, h * height);
  };
  const line = (x1: number, y1: number, x2: number, y2: number, color: number, thickness = 2, alpha = 1): void => {
    g.lineStyle(thickness, color, alpha).lineBetween(x1 * width, y1 * height, x2 * width, y2 * height);
  };
  ellipse(0.5, 0.91, 0.78, 0.12, 0x080d0b, 0.38);
  switch (definition.shape) {
    case "pine":
      rect(0.43, 0.59, 0.14, 0.31, dark, 1, 4);
      polygon([[0.5, 0.06], [0.19, 0.45], [0.81, 0.45]], dark);
      polygon([[0.5, 0.22], [0.10, 0.66], [0.90, 0.66]], dark);
      polygon([[0.5, 0.38], [0.03, 0.82], [0.97, 0.82]], dark);
      polygon([[0.5, 0.12], [0.29, 0.42], [0.59, 0.42]], mid);
      polygon([[0.5, 0.29], [0.23, 0.62], [0.72, 0.62]], mid);
      polygon([[0.5, 0.46], [0.18, 0.77], [0.80, 0.77]], mid);
      line(0.42, 0.60, 0.32, 0.74, light, 2, 0.7);
      line(0.58, 0.45, 0.69, 0.58, light, 2, 0.6);
      break;
    case "tree":
      rect(0.43, 0.58, 0.15, 0.33, dark, 1, 5);
      polygon([[0.50, 0.54], [0.31, 0.79], [0.14, 0.90], [0.39, 0.86], [0.51, 0.94], [0.63, 0.84], [0.87, 0.90], [0.68, 0.75]], dark);
      ellipse(0.34, 0.43, 0.47, 0.45, dark);
      ellipse(0.62, 0.39, 0.50, 0.49, dark);
      ellipse(0.50, 0.25, 0.51, 0.48, dark);
      ellipse(0.33, 0.43, 0.31, 0.30, mid);
      ellipse(0.61, 0.38, 0.33, 0.31, mid);
      ellipse(0.49, 0.25, 0.32, 0.30, mid);
      ellipse(0.39, 0.34, 0.12, 0.11, light, 0.75);
      ellipse(0.64, 0.31, 0.11, 0.10, accent, 0.74);
      ellipse(0.50, 0.19, 0.10, 0.09, light, 0.64);
      break;
    case "rock":
      polygon([[0.12, 0.79], [0.20, 0.45], [0.39, 0.19], [0.58, 0.27], [0.76, 0.38], [0.91, 0.76], [0.80, 0.87], [0.28, 0.91]], dark);
      polygon([[0.20, 0.73], [0.29, 0.46], [0.43, 0.30], [0.58, 0.37], [0.73, 0.46], [0.81, 0.73], [0.68, 0.81], [0.32, 0.82]], mid);
      polygon([[0.30, 0.48], [0.43, 0.30], [0.58, 0.37], [0.48, 0.49], [0.39, 0.60]], light, 0.84);
      polygon([[0.60, 0.42], [0.75, 0.50], [0.80, 0.72], [0.69, 0.78]], dark, 0.42);
      line(0.49, 0.52, 0.45, 0.71, dark, 2, 0.6);
      break;
    case "water":
      ellipse(0.50, 0.57, 0.90, 0.61, dark);
      ellipse(0.49, 0.55, 0.80, 0.49, mid);
      ellipse(0.49, 0.55, 0.62, 0.31, 0x356d76, 0.88);
      ellipse(0.40, 0.48, 0.28, 0.05, light, 0.82);
      ellipse(0.65, 0.60, 0.18, 0.035, accent, 0.75);
      ellipse(0.28, 0.69, 0.12, 0.06, dark, 0.78);
      ellipse(0.77, 0.39, 0.10, 0.05, dark, 0.72);
      break;
    case "marker":
    case "pillar":
      polygon([[0.27, 0.89], [0.33, 0.81], [0.67, 0.81], [0.74, 0.89]], dark);
      polygon([[0.37, 0.79], [0.37, 0.24], [0.43, 0.16], [0.60, 0.16], [0.65, 0.24], [0.65, 0.79]], dark);
      polygon([[0.42, 0.76], [0.42, 0.27], [0.47, 0.22], [0.57, 0.22], [0.60, 0.28], [0.60, 0.76]], mid);
      polygon([[0.32, 0.20], [0.39, 0.12], [0.63, 0.12], [0.69, 0.20]], dark);
      line(0.47, 0.34, 0.55, 0.42, accent, 3, 0.9);
      line(0.55, 0.42, 0.47, 0.53, accent, 3, 0.9);
      line(0.47, 0.53, 0.55, 0.63, accent, 3, 0.9);
      line(0.43, 0.72, 0.59, 0.72, light, 2, 0.7);
      break;
    case "palm":
      line(0.50, 0.88, 0.47, 0.42, dark, Math.max(7, width * 0.11));
      line(0.47, 0.44, 0.50, 0.85, light, Math.max(2, width * 0.025), 0.7);
      polygon([[0.47, 0.43], [0.12, 0.22], [0.36, 0.36], [0.44, 0.40]], dark);
      polygon([[0.48, 0.41], [0.28, 0.04], [0.48, 0.34], [0.51, 0.39]], mid);
      polygon([[0.50, 0.40], [0.87, 0.12], [0.60, 0.40], [0.53, 0.43]], dark);
      polygon([[0.52, 0.43], [0.96, 0.45], [0.61, 0.49], [0.54, 0.46]], mid);
      polygon([[0.46, 0.45], [0.09, 0.61], [0.42, 0.49], [0.48, 0.47]], mid);
      ellipse(0.51, 0.43, 0.13, 0.10, accent);
      break;
    case "driftwood":
      polygon([[0.11, 0.61], [0.23, 0.43], [0.70, 0.48], [0.89, 0.34], [0.82, 0.60], [0.70, 0.72], [0.27, 0.70]], dark);
      polygon([[0.18, 0.56], [0.29, 0.47], [0.69, 0.52], [0.82, 0.41], [0.73, 0.59], [0.31, 0.64]], mid);
      line(0.30, 0.54, 0.61, 0.57, light, 3, 0.7);
      ellipse(0.28, 0.54, 0.08, 0.08, dark);
      break;
    case "dune":
      ellipse(0.50, 0.69, 0.91, 0.39, dark);
      ellipse(0.47, 0.61, 0.84, 0.41, mid);
      ellipse(0.40, 0.56, 0.58, 0.24, light, 0.72);
      line(0.19, 0.67, 0.69, 0.68, accent, 2, 0.78);
      break;
    case "oasis":
      ellipse(0.50, 0.59, 0.92, 0.66, dark);
      ellipse(0.50, 0.57, 0.79, 0.54, 0x625c3a);
      ellipse(0.50, 0.57, 0.65, 0.39, 0x265c5c);
      ellipse(0.43, 0.49, 0.30, 0.06, light, 0.85);
      ellipse(0.67, 0.62, 0.18, 0.04, accent, 0.82);
      for (const [x, y] of [[0.18, 0.40], [0.82, 0.56], [0.25, 0.76], [0.75, 0.31]]) ellipse(x, y, 0.15, 0.11, mid);
      break;
    case "ruin":
      polygon([[0.12, 0.78], [0.19, 0.64], [0.35, 0.67], [0.38, 0.29], [0.49, 0.25], [0.53, 0.66], [0.70, 0.65], [0.74, 0.35], [0.84, 0.40], [0.88, 0.78]], dark);
      polygon([[0.17, 0.74], [0.24, 0.68], [0.33, 0.70], [0.36, 0.37], [0.43, 0.32], [0.47, 0.70], [0.68, 0.70], [0.72, 0.43], [0.79, 0.46], [0.82, 0.75]], mid);
      line(0.25, 0.72, 0.32, 0.72, light, 2, 0.65);
      line(0.57, 0.71, 0.65, 0.71, accent, 2, 0.65);
      polygon([[0.38, 0.37], [0.43, 0.32], [0.47, 0.40], [0.44, 0.44]], light, 0.75);
      break;
    case "grass":
      ellipse(0.5, 0.83, 0.82, 0.14, dark, 0.72);
      for (const [x, y, dx] of [[0.22, 0.77, -0.10], [0.34, 0.74, 0.06], [0.46, 0.75, -0.02], [0.59, 0.78, 0.10], [0.73, 0.74, 0.03], [0.84, 0.78, -0.07]]) {
        line(x, y, x + dx, 0.20 + (x % 0.15) * 0.3, (x * 100) % 2 < 1 ? mid : light, 3);
        line(x, y - 0.03, x - dx * 0.7, y - 0.20, accent, 2, 0.8);
      }
      break;
    case "yurt":
      ellipse(0.50, 0.72, 0.74, 0.32, dark);
      ellipse(0.50, 0.52, 0.70, 0.43, mid);
      polygon([[0.18, 0.54], [0.34, 0.30], [0.50, 0.19], [0.67, 0.31], [0.82, 0.55]], dark);
      polygon([[0.26, 0.53], [0.38, 0.34], [0.50, 0.25], [0.63, 0.35], [0.74, 0.53]], accent);
      rect(0.42, 0.57, 0.17, 0.31, dark, 1, 5);
      rect(0.46, 0.62, 0.09, 0.27, 0x171d17, 1, 4);
      line(0.22, 0.55, 0.78, 0.55, light, 2, 0.7);
      polygon([[0.47, 0.25], [0.50, 0.14], [0.53, 0.25]], light);
      break;
    case "firepit":
      ellipse(0.50, 0.72, 0.78, 0.36, dark);
      ellipse(0.50, 0.69, 0.58, 0.22, 0x171816);
      for (const [x, y] of [[0.22, 0.69], [0.34, 0.57], [0.50, 0.55], [0.66, 0.58], [0.78, 0.70], [0.61, 0.82], [0.38, 0.82]]) ellipse(x, y, 0.20, 0.18, x < 0.5 ? mid : light);
      polygon([[0.50, 0.62], [0.37, 0.40], [0.48, 0.11], [0.55, 0.35], [0.66, 0.44], [0.55, 0.63]], accent);
      polygon([[0.51, 0.58], [0.45, 0.41], [0.52, 0.25], [0.59, 0.44]], 0xf0bc70);
      break;
    case "flower":
      ellipse(0.50, 0.84, 0.74, 0.12, dark, 0.72);
      for (const [x, y] of [[0.28, 0.55], [0.45, 0.39], [0.63, 0.57], [0.38, 0.69], [0.72, 0.32]]) {
        line(x, y + 0.05, x - 0.03, 0.83, mid, 2);
        ellipse(x - 0.04, y + 0.06, 0.12, 0.07, light);
        ellipse(x + 0.04, y + 0.11, 0.10, 0.06, mid);
        ellipse(x, y, 0.18, 0.15, accent);
        ellipse(x, y, 0.07, 0.07, 0xc8a96b);
      }
      break;
    case "banner":
      rect(0.47, 0.11, 0.06, 0.78, dark, 1, 3);
      line(0.30, 0.17, 0.76, 0.17, light, 3);
      polygon([[0.50, 0.20], [0.91, 0.27], [0.85, 0.57], [0.50, 0.51]], dark);
      polygon([[0.54, 0.24], [0.85, 0.30], [0.80, 0.51], [0.54, 0.47]], accent);
      polygon([[0.62, 0.29], [0.69, 0.36], [0.62, 0.44], [0.58, 0.36]], light, 0.86);
      rect(0.37, 0.87, 0.27, 0.06, dark, 1, 4);
      break;
    case "bamboo":
      for (const [x, top, bend] of [[0.23, 0.18, -0.06], [0.40, 0.08, 0.03], [0.58, 0.15, 0.05], [0.76, 0.25, -0.02]]) {
        rect(x - 0.035, top, 0.07, 0.70 - top, dark, 1, 5);
        rect(x - 0.009, top + 0.04, 0.018, 0.62 - top, light, 0.72, 2);
        for (const y of [0.40, 0.61, 0.79]) line(x - 0.04, y, x + 0.04, y, mid, 2);
        polygon([[x, top + 0.22], [x + bend - 0.18, top + 0.13], [x + bend, top + 0.12], [x + 0.02, top + 0.25]], mid);
        polygon([[x, top + 0.43], [x + 0.17, top + 0.34], [x + 0.27, top + 0.34], [x + 0.03, top + 0.48]], light, 0.72);
      }
      break;
  }
  g.generateTexture(key, width, height);
  g.destroy();
  scene.textures.get(key).setFilter(Phaser.Textures.FilterMode.NEAREST);
}

function createFallbackTexture(scene: Phaser.Scene): void {
  if (scene.textures.exists("landscape-placeholder")) return;
  const g = scene.add.graphics();
  g.fillStyle(0x101713, 0.95).fillRect(4, 4, 40, 40);
  g.fillStyle(0x9c6b57, 1).fillRect(12, 12, 24, 24);
  g.fillStyle(0xe0b67c, 1).fillRect(19, 19, 10, 10);
  g.generateTexture("landscape-placeholder", 48, 48);
  g.destroy();
  scene.textures.get("landscape-placeholder").setFilter(Phaser.Textures.FilterMode.NEAREST);
}
