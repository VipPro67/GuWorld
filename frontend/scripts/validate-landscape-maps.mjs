import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const frontendDir = path.resolve(scriptDir, "..");
const repoDir = path.resolve(frontendDir, "..");
const clientMapsDir = path.join(frontendDir, "public/game/maps");
const serverMapsDir = path.join(repoDir, "backend/data/mmo/maps");
const manifest = JSON.parse(fs.readFileSync(path.join(frontendDir, "src/game/assets/landscape-assets.json"), "utf8"));
const starterMaps = {
  tan_thu_coc_nam_cuong: "nam_cuong",
  tan_thu_coc_dong_hai: "dong_hai",
  tan_thu_coc_tay_mac: "tay_mac",
  tan_thu_coc_bac_nguyen: "bac_nguyen",
  tan_thu_coc_trung_chau: "trung_chau",
};
const errors = [];

for (const [mapId, theme] of Object.entries(starterMaps)) {
  const filename = `${mapId}.json`;
  const clientPath = path.join(clientMapsDir, filename);
  const serverPath = path.join(serverMapsDir, filename);
  if (!fs.existsSync(clientPath) || !fs.existsSync(serverPath)) {
    errors.push(`${mapId}: missing client or server map copy`);
    continue;
  }
  const clientBytes = fs.readFileSync(clientPath);
  const serverBytes = fs.readFileSync(serverPath);
  if (!clientBytes.equals(serverBytes)) errors.push(`${mapId}: client and server map copies differ`);
  if (clientBytes.byteLength > 8 * 1024 * 1024) errors.push(`${mapId}: map exceeds the 8 MiB limit`);

  let map;
  try {
    map = JSON.parse(clientBytes.toString("utf8"));
  } catch (error) {
    errors.push(`${mapId}: invalid JSON (${error instanceof Error ? error.message : "parse error"})`);
    continue;
  }
  const properties = Object.fromEntries((map.properties ?? []).map((property) => [property.name, property.value]));
  if (map.map_id !== mapId) errors.push(`${mapId}: map_id mismatch`);
  if (map.map_version !== "regions-v2" || map.content_version !== "landscape-v2") {
    errors.push(`${mapId}: expected regions-v2 / landscape-v2, got ${map.map_version} / ${map.content_version}`);
  }
  if (properties.scenery_theme !== theme || !manifest.themes[theme]) errors.push(`${mapId}: scenery theme must be ${theme}`);

  const sceneryLayers = (map.layers ?? []).filter((layer) => layer.type === "objectgroup" && layer.name === "scenery");
  if (sceneryLayers.length !== 1) {
    errors.push(`${mapId}: expected exactly one scenery object layer`);
    continue;
  }
  for (const object of sceneryLayers[0].objects ?? []) {
    const props = Object.fromEntries((object.properties ?? []).map((property) => [property.name, property.value]));
    const asset = manifest.assets[props.asset_id];
    if (!asset) errors.push(`${mapId}: unknown asset_id ${String(props.asset_id)}`);
    else if (asset.theme !== theme) errors.push(`${mapId}/${object.name}: asset theme ${asset.theme} does not match ${theme}`);
    else if (typeof asset.renderScale !== "number" || asset.renderScale < 1 || asset.renderScale > 2) errors.push(`${mapId}/${object.name}: invalid asset renderScale`);
    if (props.collision_mode !== "decor") errors.push(`${mapId}/${object.name}: scenery must declare collision_mode=decor`);
    if (!Number.isFinite(object.x) || !Number.isFinite(object.y) || object.x < 0 || object.y < 0 || object.x >= map.width * map.tilewidth || object.y >= map.height * map.tileheight) {
      errors.push(`${mapId}/${object.name}: scenery position is outside the Tiled pixel bounds`);
    }
    const scale = props.scale ?? 1;
    if (typeof scale !== "number" || scale < 0.5 || scale > 2) errors.push(`${mapId}/${object.name}: invalid scale`);
    if (object.rotation !== undefined && (!Number.isFinite(object.rotation) || Math.abs(object.rotation) > 360)) errors.push(`${mapId}/${object.name}: invalid rotation`);
  }

  const spawnLayer = (map.layers ?? []).find((layer) => layer.type === "objectgroup" && layer.name === "spawns");
  const playerEntry = spawnLayer?.objects?.find((object) => object.type === "player");
  if (!playerEntry) {
    errors.push(`${mapId}: missing player entry spawn`);
    continue;
  }
  for (const object of spawnLayer.objects ?? []) {
    if (object.type !== "monster") continue;
    if (Math.hypot(object.x - playerEntry.x, object.y - playerEntry.y) < 64) {
      errors.push(`${mapId}/${object.name}: monster spawn is inside the 64-unit entry buffer`);
    }
  }
  console.log(`${mapId}: ${sceneryLayers[0].objects?.length ?? 0} scenery objects, theme=${theme}`);
}

if (errors.length > 0) {
  console.error(errors.map((error) => `- ${error}`).join("\n"));
  process.exitCode = 1;
} else {
  console.log("Starter landscape maps and asset references are valid.");
}
