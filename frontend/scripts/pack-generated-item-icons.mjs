import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "sharp";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.resolve(scriptDir, "..");
const repoDir = path.resolve(frontendDir, "..");
const outputDir = path.join(frontendDir, "public/assets/items/generated");
const itemManifest = JSON.parse(fs.readFileSync(path.join(repoDir, "scripts/item_manifest.json"), "utf8"));
const itemIds = new Set(itemManifest.map((item) => item.id));
const sources = process.argv.slice(2).map((argument) => {
  const separator = argument.indexOf("=");
  if (separator < 1 || separator === argument.length - 1) {
    throw new Error(`Expected ID=source.png, received: ${argument}`);
  }
  return { id: argument.slice(0, separator), source: path.resolve(argument.slice(separator + 1)) };
});

if (!sources.length) {
  console.error("Usage: node scripts/pack-generated-item-icons.mjs ITEM_ID=/path/to/image.png [...]");
  process.exit(1);
}

const seen = new Set();
for (const { id, source } of sources) {
  if (!itemIds.has(id)) throw new Error(`Item ID is not in scripts/item_manifest.json: ${id}`);
  if (seen.has(id)) throw new Error(`Duplicate item ID in arguments: ${id}`);
  seen.add(id);
  if (!fs.existsSync(source)) throw new Error(`Generated image not found: ${source}`);

  const output = path.join(outputDir, `${id}.webp`);
  if (fs.existsSync(output)) throw new Error(`Refusing to replace existing item art: ${output}`);

  const input = sharp(source, { failOn: "error" });
  const metadata = await input.metadata();
  if (!metadata.hasAlpha) throw new Error(`Generated image must have alpha transparency: ${source}`);

  fs.mkdirSync(outputDir, { recursive: true });
  await sharp(source, { failOn: "error" })
    .resize(256, 256, {
      fit: "contain",
      background: { r: 0, g: 0, b: 0, alpha: 0 },
    })
    .webp({ quality: 88, alphaQuality: 100, effort: 6 })
    .toFile(output);
  console.log(`${id}: ${source} -> ${output}`);
}

execFileSync(process.execPath, [path.join(scriptDir, "build-generated-item-icons.mjs")], { stdio: "inherit" });
