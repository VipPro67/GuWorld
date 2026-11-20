import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "sharp";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.resolve(scriptDir, "..");
const repoDir = path.resolve(frontendDir, "..");
const manifest = JSON.parse(fs.readFileSync(path.join(frontendDir, "src/data/game-icon-manifest.json"), "utf8"));
const groups = ["gu", "item", "resourcePoint"];
const records = groups.flatMap((group) => Object.entries(manifest[group] || {}).map(([id, record]) => ({ group, id, record })));

if (!records.length) {
  console.error("No game icon records to build. Add reviewed PNG masters and manifest entries first.");
  process.exit(1);
}

const builtOutputs = new Map();
for (const { group, id, record } of records) {
  const master = path.resolve(repoDir, record.master);
  const output = path.resolve(frontendDir, "public", record.runtime.replace(/^\//, ""));
  if (!master.startsWith(repoDir + path.sep) || !output.startsWith(path.join(frontendDir, "public") + path.sep)) {
    throw new Error(`${group}.${id}: asset path escapes the repository asset directories`);
  }
  const previousMaster = builtOutputs.get(output);
  if (previousMaster) {
    if (previousMaster !== master) throw new Error(`${group}.${id}: shared runtime path maps to different masters`);
    console.log(`${group}.${id}: shares ${record.runtime}`);
    continue;
  }
  if (!fs.existsSync(master)) throw new Error(`${group}.${id}: PNG master not found at ${record.master}`);

  const input = sharp(master, { failOn: "error" });
  const metadata = await input.metadata();
  if (!metadata.hasAlpha) throw new Error(`${group}.${id}: PNG master must include alpha transparency`);

  fs.mkdirSync(path.dirname(output), { recursive: true });
  await sharp(master, { failOn: "error" })
    .trim({ background: { r: 0, g: 0, b: 0, alpha: 0 }, threshold: 0 })
    .resize(256, 256, {
      fit: "contain",
      position: "centre",
      kernel: sharp.kernel.nearest,
      background: { r: 0, g: 0, b: 0, alpha: 0 },
    })
    .webp({ lossless: true, effort: 6, alphaQuality: 100 })
    .toFile(output);
  builtOutputs.set(output, master);
  console.log(`${group}.${id}: ${record.master} -> ${record.runtime}`);
}
