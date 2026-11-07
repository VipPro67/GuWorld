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
  const size = 256;
  const graphics = scene.add.graphics();
  graphics.fillStyle(palette.ground, 1).fillRect(0, 0, size, size);
  // Calm, irregular pixel clusters keep the repeated terrain dark and readable.
  let seed = theme.split("").reduce((value, char) => value * 31 + char.charCodeAt(0), 7) >>> 0;
  const random = (): number => {
    seed = (seed * 1664525 + 1013904223) >>> 0;
    return seed / 0x1_0000_0000;
  };
  for (let index = 0; index < 250; index += 1) {
    const x = Math.floor(random() * (size / 4)) * 4;
    const y = Math.floor(random() * (size / 4)) * 4;
    const color = palette.groundLights[Math.floor(random() * palette.groundLights.length)];
    const width = 4 + Math.floor(random() * 3) * 4;
    const height = 4 + Math.floor(random() * 2) * 4;
    graphics.fillStyle(color, 0.58).fillRect(x, y, width, height);
  }
  for (let index = 0; index < 40; index += 1) {
    const x = Math.floor(random() * (size / 4)) * 4;
    const y = Math.floor(random() * (size / 4)) * 4;
    graphics.fillStyle(palette.deep, 0.5).fillRect(x, y, 4, 4);
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
  const px = (x: number, y: number, w: number, h: number, color: number, alpha = 1): void => {
    g.fillStyle(color, alpha).fillRect(x, y, w, h);
  };
  px(8, height - 12, width - 16, 8, 0x080d0b, 0.34);
  switch (definition.shape) {
    case "pine":
      px(width / 2 - 8, height - 33, 16, 29, dark);
      for (const [x, y, w, h] of [[15, 35, 44, 34], [21, 20, 32, 35], [27, 8, 20, 30]]) px(x, y, w, h, dark);
      px(20, 39, 34, 24, mid); px(27, 25, 22, 25, mid); px(31, 13, 12, 24, light);
      px(23, 44, 10, 8, light, 0.7); px(38, 31, 9, 8, light, 0.6);
      break;
    case "tree":
      px(width / 2 - 8, height - 33, 16, 31, dark);
      px(14, 31, 27, 35, dark); px(32, 23, 30, 38, dark); px(25, 13, 29, 35, dark); px(8, 40, 24, 23, dark);
      px(19, 34, 21, 26, mid); px(36, 27, 21, 28, mid); px(29, 19, 20, 25, light);
      px(25, 38, 9, 7, accent, 0.75); px(42, 31, 8, 6, light, 0.8);
      break;
    case "rock":
      px(14, 27, width - 28, height - 42, dark); px(21, 17, width - 42, height - 36, dark);
      px(19, 25, width - 40, height - 43, mid); px(29, 19, width - 54, 18, light);
      px(25, 40, 13, 5, dark, 0.55); px(width - 31, 32, 10, 5, dark, 0.5);
      break;
    case "water":
      px(12, 17, width - 24, height - 30, dark); px(7, 23, width - 14, height - 34, mid);
      px(17, 25, width - 34, 6, light, 0.72); px(25, 38, width - 50, 4, accent, 0.62);
      px(18, 52, 12, 5, dark); px(width - 35, 21, 10, 8, dark);
      break;
    case "marker":
    case "pillar":
      px(width / 2 - 15, height - 24, 30, 14, dark); px(width / 2 - 11, 13, 22, height - 36, dark);
      px(width / 2 - 8, 19, 16, height - 40, mid); px(width / 2 - 7, 22, 5, height - 45, light);
      px(width / 2 - 18, 10, 36, 12, dark); px(width / 2 - 14, 7, 28, 8, light);
      px(width / 2 - 3, 36, 7, 18, accent, 0.9);
      break;
    case "palm":
      px(width / 2 - 6, 34, 13, height - 46, dark); px(width / 2 - 2, 43, 5, height - 54, light);
      px(12, 20, 28, 9, dark); px(35, 13, 31, 9, dark); px(23, 7, 11, 34, dark); px(46, 25, 27, 9, dark);
      px(17, 22, 25, 5, mid); px(38, 15, 24, 5, mid); px(26, 12, 5, 23, light); px(49, 27, 20, 5, mid);
      break;
    case "driftwood":
      px(12, 19, width - 24, 14, dark); px(21, 15, width - 34, 8, mid); px(17, 28, 24, 7, light);
      px(width - 26, 12, 8, 20, dark); px(24, 32, 9, 6, dark);
      break;
    case "dune":
      px(12, 24, width - 24, 27, dark); px(7, 31, width - 14, 21, mid);
      px(18, 25, width - 36, 8, light); px(28, 39, width - 56, 5, accent, 0.75);
      break;
    case "oasis":
      px(11, 23, width - 22, height - 31, dark); px(17, 28, width - 34, height - 39, 0x172f30);
      px(26, 35, width - 52, 5, light, 0.78); px(34, 48, width - 66, 4, accent, 0.8);
      px(8, 19, 19, 8, mid); px(width - 28, 33, 18, 8, mid);
      break;
    case "ruin":
      px(13, 16, width - 26, height - 29, dark); px(20, 22, 12, height - 36, mid);
      px(width - 32, 25, 12, height - 39, mid); px(27, 20, width - 54, 10, light);
      px(32, 44, 12, 7, accent, 0.65); px(width - 38, 50, 8, 6, dark);
      break;
    case "grass":
      px(12, height - 22, width - 24, 9, dark, 0.8);
      px(18, 22, 7, height - 42, mid); px(29, 12, 6, height - 32, light); px(40, 18, 7, height - 38, mid);
      px(50, 26, 6, height - 46, light); px(width - 24, 20, 6, height - 40, mid);
      px(14, 28, 16, 5, light); px(38, 24, 14, 5, accent, 0.9); px(55, 31, 12, 5, light);
      break;
    case "yurt":
      px(15, 34, width - 30, height - 47, dark); px(22, 25, width - 44, 20, dark);
      px(20, 37, width - 40, height - 52, mid); px(29, 29, width - 58, 13, accent);
      px(width / 2 - 7, 48, 14, height - 62, dark); px(width / 2 - 4, 51, 8, height - 65, light);
      px(width / 2 - 3, 18, 6, 12, light);
      break;
    case "firepit":
      px(9, 21, width - 18, 13, dark); px(15, 15, width - 30, 13, mid);
      px(width / 2 - 5, 7, 10, 20, accent); px(width / 2 - 2, 12, 5, 11, 0xf0bc70);
      px(15, 36, width - 30, 4, light, 0.7);
      break;
    case "flower":
      px(12, height - 17, width - 24, 5, dark, 0.8);
      for (const [x, y] of [[17, 24], [31, 15], [43, 28], [25, 36], [47, 11]]) {
        px(x + 4, y + 7, 4, height - y - 24, mid);
        px(x, y + 2, 12, 5, accent); px(x + 3, y - 1, 6, 11, light);
        px(x + 4, y + 2, 4, 4, 0xc8a96b);
      }
      break;
    case "banner":
      px(width / 2 - 3, 11, 6, height - 22, dark); px(10, 16, width - 16, 32, dark);
      px(15, 19, width - 25, 25, accent); px(20, 23, 10, 15, light);
      px(width / 2 - 2, height - 12, 4, 7, light);
      break;
    case "bamboo":
      for (const x of [16, 31, 47, 59]) {
        px(x, 12 + (x % 3) * 4, 6, height - 29, dark);
        px(x + 2, 17 + (x % 3) * 4, 2, height - 40, light);
        for (let y = 32; y < height - 18; y += 22) px(x - 1, y, 8, 3, mid);
      }
      px(12, 34, 18, 5, mid); px(41, 25, 18, 5, mid); px(34, 52, 18, 5, light);
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
