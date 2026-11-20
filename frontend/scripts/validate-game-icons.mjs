import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.resolve(scriptDir, "..");
const repoDir = path.resolve(frontendDir, "..");
const manifestPath = path.join(frontendDir, "src/data/game-icon-manifest.json");
const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
const requireFirstBatch = process.argv.includes("--require-first-batch");
const groups = ["gu", "item", "resourcePoint"];
const folderFor = { gu: "gu", item: "item", resourcePoint: "resource-point" };
const issues = [];
const runtimeMasters = new Map();

function parseCSV(content) {
  const rows = [];
  let row = [];
  let value = "";
  let quoted = false;
  for (let i = 0; i < content.length; i++) {
    const char = content[i];
    const next = content[i + 1];
    if (char === '"' && quoted && next === '"') { value += '"'; i++; }
    else if (char === '"') quoted = !quoted;
    else if (char === "," && !quoted) { row.push(value); value = ""; }
    else if ((char === "\n" || char === "\r") && !quoted) {
      if (char === "\r" && next === "\n") i++;
      row.push(value); value = "";
      if (row.some((cell) => cell !== "")) rows.push(row);
      row = [];
    } else value += char;
  }
  if (value || row.length) { row.push(value); rows.push(row); }
  return rows;
}

function csvIds(file, column) {
  const rows = parseCSV(fs.readFileSync(path.join(repoDir, "backend/data", file), "utf8"));
  const index = rows[0]?.indexOf(column) ?? -1;
  return new Set(rows.slice(1).map((row) => row[index]).filter(Boolean));
}

const allowedIds = {
  gu: csvIds("Gu.csv", "id"),
  item: csvIds("Item.csv", "id"),
  resourcePoint: csvIds("ResourcePointTemplate.csv", "point_type"),
};
for (const itemId of [...allowedIds.item]) {
  const canonical = itemId.replace(/_rank\d+$/i, "");
  if (canonical !== itemId) allowedIds.item.add(canonical);
}

function within(root, relativePath) {
  const resolved = path.resolve(root, relativePath);
  return resolved.startsWith(root + path.sep) ? resolved : undefined;
}

function pngInfo(file) {
  const data = fs.readFileSync(file);
  if (data.subarray(0, 8).toString("hex") !== "89504e470d0a1a0a") throw new Error("not a PNG file");
  const width = data.readUInt32BE(16);
  const height = data.readUInt32BE(20);
  const colorType = data[25];
  let alpha = colorType === 4 || colorType === 6;
  for (let offset = 8; offset + 12 <= data.length;) {
    const length = data.readUInt32BE(offset);
    const type = data.toString("ascii", offset + 4, offset + 8);
    if (type === "tRNS") alpha = true;
    offset += length + 12;
    if (type === "IEND") break;
  }
  return { width, height, alpha };
}

function webpInfo(file) {
  const data = fs.readFileSync(file);
  if (data.toString("ascii", 0, 4) !== "RIFF" || data.toString("ascii", 8, 12) !== "WEBP") {
    throw new Error("not a WebP file");
  }
  let width;
  let height;
  let alpha = false;
  for (let offset = 12; offset + 8 <= data.length;) {
    const type = data.toString("ascii", offset, offset + 4);
    const size = data.readUInt32LE(offset + 4);
    const payload = offset + 8;
    if (type === "VP8X" && size >= 10) {
      alpha ||= (data[payload] & 0x10) !== 0;
      width = 1 + data.readUIntLE(payload + 4, 3);
      height = 1 + data.readUIntLE(payload + 7, 3);
    } else if (type === "VP8L" && size >= 5 && data[payload] === 0x2f) {
      const b1 = data[payload + 1], b2 = data[payload + 2], b3 = data[payload + 3], b4 = data[payload + 4];
      width ??= 1 + b1 + ((b2 & 0x3f) << 8);
      height ??= 1 + (b2 >> 6) + (b3 << 2) + ((b4 & 0x0f) << 10);
      alpha ||= (b4 & 0x10) !== 0;
    } else if (type === "VP8 " && size >= 10 && data[payload + 3] === 0x9d && data[payload + 4] === 0x01 && data[payload + 5] === 0x2a) {
      width ??= data.readUInt16LE(payload + 6) & 0x3fff;
      height ??= data.readUInt16LE(payload + 8) & 0x3fff;
    } else if (type === "ALPH") alpha = true;
    offset = payload + size + (size & 1);
  }
  if (!width || !height) throw new Error("could not read WebP dimensions");
  return { width, height, alpha };
}

if (manifest.schema_version !== 1) issues.push("manifest schema_version must be 1");
let count = 0;
for (const group of groups) {
  const entries = manifest[group];
  if (!entries || typeof entries !== "object" || Array.isArray(entries)) {
    issues.push(`${group}: expected an ID-to-record object`);
    continue;
  }
  for (const [id, record] of Object.entries(entries)) {
    count++;
    const prefix = `${group}.${id}`;
    if (!allowedIds[group].has(id)) issues.push(`${prefix}: ID is absent from its source CSV`);
    if (!record || typeof record !== "object") { issues.push(`${prefix}: missing record`); continue; }
    const folder = folderFor[group];
    const artworkId = record.artwork_id || id;
    if (!/^[a-z0-9_]+$/i.test(artworkId)) issues.push(`${prefix}: invalid artwork_id ${artworkId}`);
    if (record.artwork_id && group !== "item") issues.push(`${prefix}: artwork_id sharing is currently supported only for item archetypes`);
    const expectedRuntime = `/assets/game-icons/${folder}/${artworkId}.webp`;
    const expectedMaster = `art/game-icons/masters/${folder}/${artworkId}.png`;
    if (record.runtime !== expectedRuntime) issues.push(`${prefix}: runtime path must be ${expectedRuntime}`);
    if (record.master !== expectedMaster) issues.push(`${prefix}: master path must be ${expectedMaster}`);
    if (!record.source?.trim()) issues.push(`${prefix}: source is required`);
    if (!record.prompt?.trim()) issues.push(`${prefix}: prompt is required`);
    const previousMaster = runtimeMasters.get(record.runtime);
    if (previousMaster && previousMaster !== record.master) issues.push(`${prefix}: shared runtime path maps to a different PNG master`);
    runtimeMasters.set(record.runtime, record.master);

    for (const [kind, relativePath, root, readInfo, min, max] of [
      ["master", record.master, repoDir, pngInfo, 128, 2048],
      ["runtime", record.runtime?.replace(/^\//, ""), path.join(frontendDir, "public"), webpInfo, 64, 512],
    ]) {
      const file = relativePath && within(root, relativePath);
      if (!file) { issues.push(`${prefix}: invalid ${kind} path`); continue; }
      if (!fs.existsSync(file)) { issues.push(`${prefix}: missing ${kind} file ${relativePath}`); continue; }
      try {
        const { width, height, alpha } = readInfo(file);
        const dimensionsValid = kind === "master"
          ? width >= min && width <= max && height >= min && height <= max
          : width === height && width >= min && width <= max;
        if (!dimensionsValid) issues.push(`${prefix}: invalid ${kind} dimensions ${width}x${height} (limits ${min}–${max}px${kind === "runtime" ? ", square" : ""})`);
        if (!alpha) issues.push(`${prefix}: ${kind} has no transparency`);
      } catch (error) {
        issues.push(`${prefix}: invalid ${kind}: ${error.message}`);
      }
    }
  }
}

if (requireFirstBatch) {
  const planPath = path.join(repoDir, "docs/wayfinder/workmaps/gameplay-icon-library/research/first-rollout-batch-plan.json");
  if (!fs.existsSync(planPath)) {
    issues.push("first rollout batch plan not found");
  } else {
    const plan = JSON.parse(fs.readFileSync(planPath, "utf8"));
    const expectedCounts = { gu: 35, item: 44, resourcePoint: 47 };
    const planIds = [];
    for (const group of groups) {
      const planned = plan.batch?.[group] || [];
      if (planned.length !== expectedCounts[group]) issues.push(`plan.${group}: expected ${expectedCounts[group]} records, found ${planned.length}`);
      for (const record of planned) {
        planIds.push(`${group}:${record.id}`);
        if (!record.prompt_seed?.trim()) issues.push(`plan.${group}.${record.id}: prompt_seed is required`);
        const plannedArtworkId = record.shared_artwork_id;
        const manifestArtworkId = manifest[group]?.[record.id]?.artwork_id;
        if (plannedArtworkId && plannedArtworkId !== manifestArtworkId) {
          issues.push(`plan.${group}.${record.id}: shared_artwork_id does not match the manifest`);
        }
      }
      const missing = planned.map((record) => record.id).filter((id) => !manifest[group]?.[id]);
      if (missing.length) issues.push(`${group}: ${missing.length} first-batch IDs missing from manifest (${missing.join(", ")})`);
    }
    const sampleIds = plan.sample_review_ids || {};
    for (const group of groups) {
      if (!plan.batch?.[group]?.some((record) => record.id === sampleIds[group])) {
        issues.push(`plan.sample_review_ids.${group}: sample ID is not in that group's batch`);
      }
    }
    if (new Set(planIds).size !== planIds.length) issues.push("plan.batch contains duplicate group/ID entries");
  }
}

if (!count) issues.push("manifest has no delivered icons yet");
if (issues.length) {
  console.error(`Game icon validation failed (${count} records):`);
  for (const issue of issues) console.error(`- ${issue}`);
  process.exitCode = 1;
} else {
  console.log(`Validated ${count} game icon records (${runtimeMasters.size} unique artworks) against source IDs, PNG masters, WebP runtime files, dimensions, alpha, and prompt metadata.`);
}
