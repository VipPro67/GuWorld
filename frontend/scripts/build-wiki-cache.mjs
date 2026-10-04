import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const BACKEND_DATA_DIR = path.resolve(__dirname, "../../backend/data");
const OUTPUT_FILE = path.resolve(__dirname, "../src/data/wiki-cache.json");

function parseCSV(content) {
  const rows = [];
  let currentRow = [];
  let currentVal = "";
  let insideQuotes = false;

  for (let i = 0; i < content.length; i++) {
    const char = content[i];
    const nextChar = content[i + 1];

    if (char === "\"" && insideQuotes && nextChar === "\"") {
      currentVal += "\"";
      i++;
    } else if (char === "\"") {
      insideQuotes = !insideQuotes;
    } else if (char === "," && !insideQuotes) {
      currentRow.push(currentVal);
      currentVal = "";
    } else if ((char === "\r" || char === "\n") && !insideQuotes) {
      if (char === "\r" && nextChar === "\n") i++;
      currentRow.push(currentVal);
      currentVal = "";
      if (currentRow.length > 1 || (currentRow.length === 1 && currentRow[0] !== "")) {
        rows.push(currentRow);
      }
      currentRow = [];
    } else {
      currentVal += char;
    }
  }
  if (currentVal || currentRow.length > 0) {
    currentRow.push(currentVal);
    rows.push(currentRow);
  }
  return rows;
}

function safeJSON(str, fallback) {
  if (!str) return fallback;
  try {
    return JSON.parse(str);
  } catch {
    return fallback;
  }
}

function normalizeEffectEnvelope(val) {
  if (!val) return { blocks: [] };
  if (Array.isArray(val)) return { schema_version: 2, blocks: val };
  if (typeof val === "object" && Array.isArray(val.blocks)) return val;
  return { schema_version: 2, blocks: [] };
}

function readCSVFile(filename) {
  const fullPath = path.join(BACKEND_DATA_DIR, filename);
  if (!fs.existsSync(fullPath)) {
    console.warn(`[build-wiki-cache] Warning: File not found: ${fullPath}`);
    return [];
  }
  return parseCSV(fs.readFileSync(fullPath, "utf8"));
}

function buildWikiData() {
  console.log("[build-wiki-cache] Parsing CSV files from backend/data...");

  // 1. Gus
  const guRows = readCSVFile("Gu.csv");
  const guHeader = guRows[0] || [];
  const guCol = (name) => guHeader.indexOf(name);
  const gusMap = new Map();

  for (let i = 1; i < guRows.length; i++) {
    const r = guRows[i];
    const id = r[guCol("id")];
    if (!id) continue;
    const rank = parseInt(r[guCol("rank")]) || 1;
    const key = `${id}_${rank}`;

    gusMap.set(key, {
      id,
      name: r[guCol("name")] || id,
      rank,
      path: safeJSON(r[guCol("path")], []),
      gu_type: r[guCol("gu_type")] || "Active",
      base_mana_cost: parseFloat(r[guCol("base_mana_cost")]) || 0,
      effects: normalizeEffectEnvelope(safeJSON(r[guCol("effects")], [])),
      hunger_rate: parseFloat(r[guCol("hunger_rate")]) || 1,
      food_types: safeJSON(r[guCol("food_types")], []),
      max_hunger: parseFloat(r[guCol("max_hunger")]) || 100,
      description: r[guCol("description")] || "",
      target_type: r[guCol("target_type")] || "SINGLE",
      rarity: parseInt(r[guCol("rarity")]) || 0,
      is_immortal: (r[guCol("is_immortal")] || "").toLowerCase() === "true",
      primary_role: r[guCol("primary_role")] || "",
      combat_roles: safeJSON(r[guCol("combat_roles")], []),
      combat_tags: safeJSON(r[guCol("combat_tags")], []),
      support_tags: safeJSON(r[guCol("support_tags")], []),
    });
  }
  const gus = Array.from(gusMap.values());

  // 2. Items
  const itemRows = readCSVFile("Item.csv");
  const itemHeader = itemRows[0] || [];
  const itemCol = (name) => itemHeader.indexOf(name);
  const itemsMap = new Map();

  for (let i = 1; i < itemRows.length; i++) {
    const r = itemRows[i];
    const id = r[itemCol("id")];
    if (!id) continue;

    itemsMap.set(id, {
      id,
      name: r[itemCol("name")] || id,
      item_type: r[itemCol("item_type")] || "Material",
      rank: parseInt(r[itemCol("rank")]) || 1,
      effects: normalizeEffectEnvelope(safeJSON(r[itemCol("effects")], [])),
      description: r[itemCol("description")] || "",
      base_price: parseFloat(r[itemCol("base_price")]) || 0,
      image_url: r[itemCol("image_url")] || "",
      rarity: parseInt(r[itemCol("rarity")]) || 0,
      path: safeJSON(r[itemCol("path")], []),
    });
  }
  const items = Array.from(itemsMap.values());

  // 3. Monsters
  const monsterRows = readCSVFile("Monster.csv");
  const monsterHeader = monsterRows[0] || [];
  const monsterCol = (name) => monsterHeader.indexOf(name);
  const monstersMap = new Map();

  for (let i = 1; i < monsterRows.length; i++) {
    const r = monsterRows[i];
    const id = r[monsterCol("id")];
    if (!id) continue;

    monstersMap.set(id, {
      id,
      name: r[monsterCol("name")] || id,
      type: r[monsterCol("type")] || "Normal",
      max_hp: parseFloat(r[monsterCol("max_hp")]) || 0,
      base_attack: parseFloat(r[monsterCol("base_attack")]) || 0,
      base_defense: parseFloat(r[monsterCol("base_defense")]) || 0,
      loot_table_id: r[monsterCol("loot_table_id")] || "",
      description: r[monsterCol("description")] || "",
      is_capturable: (r[monsterCol("is_capturable")] || "").toLowerCase() === "true",
      capture_chance: parseFloat(r[monsterCol("capture_chance")]) || 0,
      pet_hunger_per_hour: parseFloat(r[monsterCol("pet_hunger_per_hour")]) || 0,
    });
  }
  const monsters = Array.from(monstersMap.values());

  // 4. Recipes
  const recipeRows = readCSVFile("Recipe.csv");
  const recipeHeader = recipeRows[0] || [];
  const recipeCol = (name) => recipeHeader.indexOf(name);
  const recipesMap = new Map();

  for (let i = 1; i < recipeRows.length; i++) {
    const r = recipeRows[i];
    const targetGuId = r[recipeCol("target_gu_id")];
    if (!targetGuId) continue;
    const targetRank = parseInt(r[recipeCol("target_rank")]) || 1;
    const variant = parseInt(r[recipeCol("variant")]) || 1;
    const id = `recipe_${targetGuId}_${targetRank}_${variant}`;

    recipesMap.set(id, {
      id,
      target_gu_id: targetGuId,
      target_rank: targetRank,
      variant,
      required_gus: safeJSON(r[recipeCol("required_gus")], {}),
      required_materials: safeJSON(r[recipeCol("required_materials")], {}),
      success_rate: parseFloat(r[recipeCol("success_rate")]) || 0,
      primeval_stones_cost: parseFloat(r[recipeCol("primeval_stones_cost")]) || 0,
      immortal_essence_stones_cost: parseFloat(r[recipeCol("immortal_essence_stones_cost")]) || 0,
    });
  }
  const recipes = Array.from(recipesMap.values());

  // 5. Regions
  const regionRows = readCSVFile("Region.csv");
  const regionHeader = regionRows[0] || [];
  const regionCol = (name) => regionHeader.indexOf(name);
  const regionsMap = new Map();

  for (let i = 1; i < regionRows.length; i++) {
    const r = regionRows[i];
    const id = r[regionCol("id")];
    if (!id) continue;

    regionsMap.set(id, {
      id,
      name: r[regionCol("name")] || id,
      description: r[regionCol("description")] || "",
      min_rank: parseInt(r[regionCol("min_rank")]) || 1,
      max_rank: parseInt(r[regionCol("max_rank")]) || 9,
      parent_id: r[regionCol("parent_id")] || undefined,
      zone_type: r[regionCol("zone_type")] || "NORMAL",
      min_x: parseInt(r[regionCol("min_x")]) || 0,
      min_y: parseInt(r[regionCol("min_y")]) || 0,
      max_x: parseInt(r[regionCol("max_x")]) || 0,
      max_y: parseInt(r[regionCol("max_y")]) || 0,
      entry_x: parseInt(r[regionCol("entry_x")]) || 0,
      entry_y: parseInt(r[regionCol("entry_y")]) || 0,
      pool_weights: safeJSON(r[regionCol("pool_weights")], { resources: [], wild_gus: [] }),
    });
  }
  const regions = Array.from(regionsMap.values());

  // 6. Loot Tables
  const lootRows = readCSVFile("LootTable.csv");
  const lootHeader = lootRows[0] || [];
  const lootCol = (name) => lootHeader.indexOf(name);
  const lootMap = new Map();

  for (let i = 1; i < lootRows.length; i++) {
    const r = lootRows[i];
    const id = r[lootCol("id")];
    if (!id) continue;

    lootMap.set(id, {
      id,
      name: r[lootCol("name")] || id,
      entries: safeJSON(r[lootCol("entries")], []),
    });
  }
  const lootTables = Array.from(lootMap.values());

  // 7. Monster Spawns
  const spawnRows = readCSVFile("MonsterSpawn.csv");
  const spawnHeader = spawnRows[0] || [];
  const spawnCol = (name) => spawnHeader.indexOf(name);
  const spawns = [];

  for (let i = 1; i < spawnRows.length; i++) {
    const r = spawnRows[i];
    const regionId = r[spawnCol("region_id")];
    const monsterId = r[spawnCol("monster_id")];
    if (!regionId || !monsterId) continue;

    spawns.push({
      region_id: regionId,
      monster_id: monsterId,
      chance: parseFloat(r[spawnCol("chance")]) || 0,
      min_rank: parseInt(r[spawnCol("min_rank")]) || 1,
      max_rank: parseInt(r[spawnCol("max_rank")]) || 9,
      is_boss_spawn: (r[spawnCol("is_boss_spawn")] || "").toLowerCase() === "true",
    });
  }

  const result = {
    gus,
    items,
    monsters,
    recipes,
    regions,
    lootTables,
    monsterSpawns: spawns,
    generatedAt: new Date().toISOString(),
  };

  fs.mkdirSync(path.dirname(OUTPUT_FILE), { recursive: true });
  fs.writeFileSync(OUTPUT_FILE, JSON.stringify(result));

  console.log(`[build-wiki-cache] Successfully generated ${OUTPUT_FILE}`);
  console.log(`[build-wiki-cache] Stats: ${gus.length} Gus, ${items.length} Items, ${monsters.length} Monsters, ${recipes.length} Recipes, ${regions.length} Regions, ${lootTables.length} LootTables, ${spawns.length} Spawns.`);
}

buildWikiData();
