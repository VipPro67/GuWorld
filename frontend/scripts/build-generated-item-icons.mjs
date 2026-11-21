import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.resolve(scriptDir, "..");
const iconDir = path.join(frontendDir, "public/assets/items/generated");
const manifestPath = path.join(frontendDir, "src/data/generated-item-icons.json");

const ids = fs.existsSync(iconDir)
  ? fs.readdirSync(iconDir)
    .filter((filename) => /^[\w-]+\.webp$/.test(filename))
    .map((filename) => filename.slice(0, -".webp".length))
    .sort()
  : [];

fs.writeFileSync(manifestPath, `${JSON.stringify(ids, null, 2)}\n`);
console.log(`Generated item icon manifest: ${ids.length} icons`);
