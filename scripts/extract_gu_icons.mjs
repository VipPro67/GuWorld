import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";

const ATLAS_JSON = "frontend/public/cotienlo-reference-assets/har-capture/assets/visual/o_tui_atlas.json";
const ATLAS_PNG = "frontend/public/cotienlo-reference-assets/assets/visual/o_tui_atlas.png";
const OUT_DIR = "frontend/public/assets/gu_icons";

fs.mkdirSync(OUT_DIR, { recursive: true });

const raw = JSON.parse(fs.readFileSync(ATLAS_JSON, "utf-8"));
const frames = raw.frames || {};

const guKeys = Object.keys(frames).filter((k) => k.startsWith("gu_") || k.startsWith("icon_gu_"));
console.log(`Extracting ${guKeys.length} Gu icons from atlas...`);

let count = 0;
const start = Date.now();

for (const key of guKeys) {
  const f = frames[key].frame;
  const outPath = path.join(OUT_DIR, `${key}.png`);
  // Crop via ImageMagick
  execFileSync("magick", [
    ATLAS_PNG,
    "-crop",
    `${f.w}x${f.h}+${f.x}+${f.y}`,
    "+repage",
    outPath,
  ]);
  count++;
}

console.log(`Extracted ${count} Gu icons into ${OUT_DIR} in ${Date.now() - start}ms.`);
