# Research Report #474: Aperture Labor, Variant Humans & Variant Gu Immortals — Technical Audit & Lore Synthesis

**Workmap:** Map #473 (`aperture-variant-labor`)  
**Ticket:** #474  
**Scope:** Primary-source technical audit of the GuWorld aperture labor, variant human, and `VARIANT_NPC` combat systems, paired with *Cổ Chân Nhân* (Reverend Insanity) lore precedents for the 9 variant human races, aperture economy, and Variant Gu Immortals (Cổ Tiên Dị Nhân).

---

## Part 1: Technical Audit of Current Codebase

### 1.1 Split-Brain Worker Tables: `aperture_node_workers` vs. `aperture_point_workers`

The database schema and handlers currently maintain two parallel worker-assignment tables with divergent lifecycles:

| Aspect | `aperture_node_workers` | `aperture_point_workers` |
| :--- | :--- | :--- |
| **Schema** | `backend/internal/db/schema_aperture.go:445-466` (`node_id` FK → `aperture_resource_nodes`) | `backend/internal/db/schema_aperture.go:368-383` (`point_id` FK → `aperture_resource_points`) |
| **Parent Table Lifecycle** | `schema_aperture.go:425-431` unconditionally deletes all `aperture_resource_nodes` where `COALESCE(node_category, 'RESOURCE') <> 'DEFENSE'` on every server boot (`aperture_resource_nodes_legacy_strip`). `PlaceApertureNode` (`aperture_nodes.go:312-315`) rejects non-`DEFENSE` nodes. | `aperture_resource_points` owns the active canon resource-point production economy (`aperture_points.go:1-7`). |
| **Assignment Handlers** | `GetApertureNodeWorkers`, `AssignWorkerToNode`, `RemoveWorkerFromNode` in `variant_economy.go:35-228` (checks `assignedElsewhere + req.AssignedPopulation > vhPop` only against `aperture_node_workers`). | `GetAperturePointWorkers`, `AssignWorkerToPoint`, `RemoveWorkerFromPoint` in `aperture_points.go:2498-2653` (checks `assignedElsewhere + req.AssignedPopulation > vhPop` only against `aperture_point_workers`). |
| **Double-Allocation Bug** | Neither table cross-checks the other. A player can assign 100% of a race's population in `aperture_point_workers` and still appear 100% unassigned in `aperture_node_workers` (and vice versa if any `DEFENSE` node is staffed). | Same: `AssignWorkerToPoint` (`aperture_points.go:2588-2601`) ignores `aperture_node_workers`. |
| **Race Affinity** | None enforced. | `AssignWorkerToPoint` (`aperture_points.go:2535-2621`) does **not** enforce `template.WorkerRace` matching; any race can be assigned to any point for `workerMult = 1 + 0.25 * min(1.0, pop/500.0)` (`aperture_points.go:1034-1037`). Meanwhile, `aperturePointYield` (`aperture_points.go:816-830`) computes `popBonus` and `laborMult` **exclusively** from `ctx.variantHumans[template.WorkerRace]`. |
| **Race Key Mismatch** | `aperture_variant_humans.race` uses uppercase enum keys (`HAIRY_MAN`, `ROCKMAN`, `FEATHERMAN`, etc., e.g. `variant_economy.go:554-564`). | `ResourcePointTemplate.csv` uses lowercase `Race.csv` keys (`fur_people`, `rockman`, `feathered`, `inkman`, `merman`, `ghostman`, `variant_beastman`). `normalizeWorkerRace()` (`aperture_points.go:93-107`) bridges `hairy_man → fur_people`, `featherman → feathered`, `rock_man → rockman`, and lowercases the rest (`SNOWMAN → snowman`, `INKMAN → inkman`, `MINIMAN → miniman`, `MERMAN → merman`, `MUSHROOMMAN → mushroomman`, `DRAGONMAN → dragonman`). |

**Key takeaway:** `aperture_node_workers` is effectively orphaned for resource labor because all `RESOURCE` nodes are stripped on boot (`schema_aperture.go:425-427`), leaving `aperture_point_workers` as the sole active production labor table.

---

### 1.2 Auto-Refinement Idle Labor Bug (`backend/internal/worker/variant_refinement.go`)

- **Tick Rate & Concurrency:** `StartApertureRefinementWorker` (`variant_refinement.go:421-425`) runs every **10 minutes** via `RunEvery`, processing up to 32 enabled apertures per tick with a semaphore of 4 goroutines and `pg_advisory_lock(hashtext(apertureID))` (`variant_refinement.go:61, 431-450`).
- **Workforce Reservation:** `requiredHairyMen(rank)` (`variant_refinement.go:121-129`) reserves `rank * 100` idle Hairy Men (`HAIRY_MAN`) per queued refinement job per tick (100 for Rank 1, up to 900 for Rank 9).
- **The Bug:** Both `processApertureRefinement` (`variant_refinement.go:68-73`) and `processRefinementJob` (`variant_refinement.go:181`) calculate assigned Hairy Men by querying **`aperture_node_workers`**, NOT `aperture_point_workers`:
  ```sql
  SELECT GREATEST(v.population - COALESCE(SUM(w.assigned_population), 0), 0)
  FROM aperture_variant_humans v
  LEFT JOIN aperture_node_workers w ON w.variant_human_id = v.id
  WHERE v.aperture_id = $1 AND v.race = 'HAIRY_MAN'
  GROUP BY v.population
  ```
  Because players now assign Hairy Men to resource points via `aperture_point_workers` (`AperturePointsPanel.tsx`), `variant_refinement.go` sees `assignedHairy = 0` and lets 100% of Hairy Men double-dip as both point workers and full-time auto-refiners.
- **Variant NPC Refiner Hook:** `processRefinementJob` (`variant_refinement.go:264-282`) queries `non_player_actors` where `actor_kind = 'VARIANT_NPC'`, `lifecycle_state = 'ACTIVE'`, `execution_state = 'WORKING'`, `role = 'REFINER'` and adds up to `+0.25` (`+25%`) to `recipe.successRate` via `namedNPCBonus(refiners, "REFINER", .25)`. Note that `MINER` and `FARMER` roles exist in `AssignNPCRole` (`variant_economy.go:812`) but have no analogous bonus hook in `aperture_points.go`.

---

### 1.3 Society Simulation, Gu Master Spawning & Competing `tribulation_trigger.go` Loop

#### A. `StartSocietySimulationWorker` (`backend/internal/worker/society_simulation.go`, `backend/cmd/server/main.go:206`)
- **Tick Rate:** **5 minutes** (`5 * time.Minute`).
- **Capacity Lookup (`loadSettlementCapacities`, lines 147-175):** Sums `population_capacity` from `aperture_variant_settlements` per `(aperture_id, race)`. Falls back to hardcoded `cap = 1000` (line 105) if no settlement exists for that race.
- **Food & Essence (`simulateRace`, lines 185-202):**
  - `foodNeeded = pop * foodPerCapita` (default `food_per_capita = 1.0`, `food_stored = 0`).
  - If `foodStored < foodNeeded`, it checks `apertureEssence` (`immortal_apertures.immortal_essence`) as a read-only proxy: `loadApertureEssence` reads `immortal_essence`, and `simulateRace` adds `apertureEssence` to `foodAvailable` when `foodStored` is short—**critically, `batchUpdateSociety` (`society_simulation.go:325-373`) never deducts `immortal_essence` from `immortal_apertures`!** Thus `immortal_essence` acts as an unconsumed infinite buffer whenever `immortal_essence >= foodNeeded`.
- **Happiness (`lines 204-233`, clamped `[0, 100]`):**
  - Mean-reversion: `+(50.0 - happiness) * 0.01`
  - Food: `-3.0` if `newFoodStored <= 0`; `+0.5` if `newFoodStored > pop * foodPerCapita * 10`
  - Labor intensity: `LOW` `+1.0`, `NORMAL` `0`, `HIGH` `-2.0`, `EXPLOIT` `-5.0`
  - Oppression: `-oppression * 0.1`
- **Oppression (`lines 235-243`, clamped `[0, 100]`):**
  - Decay: `oppression *= 0.99`; if `laborIntensity == "EXPLOIT"`, `+1.0`.
- **Rebellion Rate (`lines 245-257`, clamped `[0, 100]`):**
  - Decay: `rebellionRate *= 0.98`
  - `+2.0` if `newHappiness < 30`, `+1.0` if `newOppression > 50`, `+1.5` if `laborIntensity == "EXPLOIT"`.
  - **Note:** `society_simulation.go` updates `rebellion_rate` in DB, but **never fires any rebellion event or damage**.
- **Population & Prosperity (`lines 259-303`):**
  - `growthRate = populationGrowth * (1.0 + (prosperity - 50.0)/200.0)` (default `0.001`).
  - `mortality = mortalityRate` (default `0.0005`), `×3.0` if `newFoodStored <= 0`, `×2.0` if `newHappiness < 20`.
  - `newPop = clamp(population + round(pop*growthRate) - round(pop*mortality), 0, capacity)`.
  - `prosperity`: drifts `+(50 - prosperity)*0.005`, `+0.5` if `newFoodStored > pop*foodPerCapita*5`, `-1.0` if `newHappiness < 30`, `-1.5` if `newRebellion > 50`.

#### B. Competing Mutation in `tribulation_trigger.go` (`lines 491-529`)
- **Critical Conflict:** `CheckAndTriggerTribulations` **also** mutates `aperture_variant_humans` (`population`, `happiness`, `oppression`) on its own tick:
  - Uses a completely different cap: `maxPop := 200 * settlementsByAperture[ap.ID]` (line 516) instead of `SUM(population_capacity)` from `aperture_variant_settlements`.
  - Uses a completely different growth/decay formula (`1.05^yearsDiff` when `happiness > 60`, `0.90^yearsDiff` when no settlements, lines 518-525).
  - Actually executes rebellion rolls (`rebellionRisk := (100 - newHappiness + newOppression) / 4.0`, triggers at `happiness < 30 || oppression > 70`, cuts `pop * 0.8`, halves `vitality` on 2 random `aperture_resource_points`, deals `5%` `max_aperture_hp` damage, lines 491-513) — ignoring `society_simulation.go`'s `rebellion_rate` column.

#### C. `StartGuMasterSpawningWorker` (`backend/internal/worker/gu_master_spawning.go`, `backend/cmd/server/main.go:207`)
- **Tick Rate:** **30 minutes** (`30 * time.Minute`).
- **Eligibility & Formula (`lines 24-93`):**
  - Requires `population >= 500`, `gu_master_count < 100`, `aperture.state != 'DESTROYED'`.
  - `chance = 0.05 * (pop / 1000.0) * (1.0 + (prosperity - 50.0) / 200.0)`.
- **Spawned Actor (`lines 108-171`):**
  - Inserts into `non_player_actors` with `actor_kind = 'VARIANT_NPC'`, `role = 'NONE'`, `execution_state = 'STORED'`, `lifecycle_state = 'ACTIVE'`, `path = 'REFINEMENT'` (hardcoded!), `rank` rolled (`1: 70%`, `2: 20%`, `3: 8%`, `4: 2%`), `talent` `20–80`, `loyalty` `50–100`.
  - Increments `aperture_variant_humans.gu_master_count` by 1.
  - **Missing Immortal Ascension:** `aperture_variant_humans.gu_immortal_count` (`schema_aperture.go:356`) is **never incremented** anywhere in `worker/` or `handlers/` (only exposed in `admin_world_extended.go` CRUD). Furthermore, `CultivateNPC` (`variant_npc_gameplay.go:18, 277-280`) explicitly caps NPCs at `maxMortalNPCRank = 5` (`sub_rank = 4`, "NPC đã đạt đỉnh Ngũ Chuyển").
  - **Naming Gap:** `npcPrefixes` (`gu_master_spawning.go:174-178`) only defines prefixes for `HAIRY_MAN`, `ROCKMAN`, and `FEATHERMAN`, falling back to `HAIRY_MAN` for the other 6 races.

---

### 1.4 `non_player_actors` (`VARIANT_NPC`) & Combat Integration Hooks

1. **Schema & Constraints (`backend/internal/db/schema_actors.go:60, 78-115`):**
   - `non_player_actors` supports `actor_kind = 'VARIANT_NPC'` with `owner_player_id`, `aperture_id`, `formation_slot` (`1..8` unique per `owner_player_id` across `COMPANION_BEAST` and `VARIANT_NPC` when `lifecycle_state = 'ACTIVE' AND formation_slot > 0`), `path`, `role`, `loyalty`, `talent`, `capabilities` JSONB (`MOVE`, `FOLLOW`, `GUARD`, `COMBAT`, `PRODUCE`, `REFINE`, `TRAIN`, `EQUIP_GU`).
2. **Formation & Equipment (`backend/internal/handlers/variant_npc_gameplay.go`):**
   - `SetNPCFormation` (`lines 79-136`): Assigns `formation_slot` (`1..min(8, 1+playerRank)`), requiring `role = 'NONE'` and `execution_state = 'STORED'`.
   - `EquipNPCGu` / `UnequipNPCGu` (`lines 143-253`): Equips up to `actor.rank` Gu worms from `player_gu` (`gu_rank <= actor.rank`, `location = 'ACTOR'`).
3. **Combat Loading (`backend/internal/handlers/combat_state.go:215-242, 442-491`):**
   - `loadFormationNPCCombatants` loads all `VARIANT_NPC`s with `formation_slot > 0`, `lifecycle_state = 'ACTIVE'`, `execution_state = 'STORED'`, `role = 'NONE'` for both the attacker (`base.Allies`) and PVP defender (`base.Enemies`).
   - Sets `Kind = "VariantNPC"`, `MaxMana = float64(rank * 100)`, `PathMasteries = {path: rank * 100}`, `DaoMarks = {}`, and loads equipped Gu via `loadActorGu`.
4. **Simulation Execution (`backend/internal/handlers/combat_sim_build.go:78-81, 105-108, 293-343`):**
   - `buildSimCombatState` routes `ally.Kind == "VariantNPC"` and `enemy.Kind == "VariantNPC"` to `buildSimVariantNPC(actor)`.
   - `buildSimVariantNPC` **already supports full Gu mechanics**: it resolves `gamefx.Get` for `actor.Race`, `equippedGuRaceModifiers`, active Gu skills (`buildSimActionFromGu`), `Mechanics`, `PassiveDamageModifiers`, `DamageMitigations`, `PassiveOperationEffects`, `TurnStartCounterAdds`, `BeforeHitCounterAdds`, `BeforeHitCounterStatModifiers`, `BeforeHitRecoveries`, and `BattleStartEndCommands`.
   - **Gaps for Variant Gu Immortals & Tribulations:**
     - `loadFormationNPCCombatants` sets `DaoMarks = {}` (`combat_state.go:483`) — Variant Gu Immortals have no personal Dao Marks or Immortal Essence (`immortal_essence`) pool, only `Rank * 100` mortal-style `Mana`.
     - Tribulation defense / aperture-interior combat: `VARIANT_NPC`s with `role != 'NONE'` (e.g., `REFINER`, `MINER`, `FARMER`) are blocked from `formation_slot > 0`, and there is no hook for stationed Variant Gu Immortals to auto-join tribulation defense, cast formation/domain buffs, or mitigate tribulation hazards inside the aperture.

---

### 1.5 Frontend Audit (`ImmortalApertureView.tsx` & `AperturePointsPanel.tsx`)

1. **Overview Sub-tab (`ImmortalApertureView.tsx:2927-3107`) — 4 Distinct Bugs:**
   - **3 vs. 9 Races (`lines 2929-2961`):** Hardcodes only `HAIRY_MAN`, `ROCKMAN`, `FEATHERMAN`, even though the `settlements` sub-tab (`lines 3169-3190, 3291-3299`) supports all **9 races** (`HAIRY_MAN`, `ROCKMAN`, `FEATHERMAN`, `SNOWMAN`, `INKMAN`, `MINIMAN`, `MERMAN`, `MUSHROOMMAN`, `DRAGONMAN`). Likewise, the `npcs` sub-tab (`line 3369`) only maps icons for those 3 races.
   - **Broken Capacity Calculation (`lines 2972-2975`):** Computes `settlementCount` by filtering `apertureNodes` (`aperture_resource_nodes`) for `HAIRY_MAN_SETTLEMENT`, `ROCKMAN_TRIBE`, `FEATHERMAN_CANYON` and setting `maxPop = 200 * settlementCount`. Because `schema_aperture.go:425-427` deletes all non-`DEFENSE` nodes on boot, `settlementCount` is **always `0`** and the UI displays `population / 0`! Meanwhile, `aperture_variant_settlements` (`settlements` state) holds the real `population_capacity` (`10,000` to `1,000,000`).
   - **Stale Client-Side Formulas (`lines 2977-3004`):**
     - Computes `rebellionRate = (100.0 - happiness + oppression) / 4.0` instead of reading `record.rebellion_rate` from `society_simulation.go`.
     - Labor confirmation modal (`lines 3001-3003`) warns of `-10/-25 happiness and +10/+25 oppression per year` (the old `tribulation_trigger.go` numbers), whereas `society_simulation.go` ticks every **5 real minutes** with `-2.0/-5.0` happiness and `+1.0` oppression (`EXPLOIT`).
   - **Misleading Buff Labels (`lines 2980-2989`):** Claims Hairy Men produce "nô lệ luyện Cổ", Rockmen give `+min(50, pop*0.1)%` mining, and Feathermen give `-min(50, pop*0.1)%` tribulation HP loss—none of which match `aperturePointYield` (`popBonus = 1 + pop/1000` for `template.WorkerRace`, `workerMult = 1 + 0.25*min(1, assigned/500)`).
2. **Time Units & `AperturePointsPanel.tsx`:**
   - `TribulationRealTimeInfo.tsx:52` converts internal years to real seconds via `(internalYearsLeft * 36000.0) / fallbackRate` (meaning 1 internal year = `36,000` real seconds = `10` real hours at `1.0x`, or `1` real hour at `10.0x`).
   - `AperturePointsPanel.tsx:306-307` displays `Sản lượng {preview.expected}/{point.production_interval_hours || 12} giờ nội bộ · ~{preview.per_hour}/giờ thật`, which matches `accrueAperturePoint` (`elapsed.Hours() * effectiveTimeRate`, `aperture_points.go:542`) and `buildApertureYieldPreview` (`y.Count * effectiveRate / hours`, `aperture_points.go:885`).
   - `AperturePointsPanel.tsx:179-180` accurately states `mỗi 500 lao động +25% sản lượng (tối đa +25%)`, but `PointWorkerEditor` (`lines 145-155`) lets the player pick any race without indicating whether that race matches `point.worker_race` (which controls `popBonus` and `laborMult` in `aperturePointYield`).

---

## Part 2: *Cổ Chân Nhân* (Reverend Insanity) Lore Precedents

### 2.1 The 9 Variant Human Races (Cửu Đại Dị Nhân)

In *Cổ Chân Nhân*, variant humans are not mere reskins of humans—each possesses innate Dao Marks of a specific path, distinct physiology, ecological requirements, and social structures. (Note: Dragonmen are an artificial/later variant created by Duke Long rather than one of the primal Ten/Nine, though GuWorld groups them as the 9th playable/aperture tribe.)

| Code Key | Vietnamese / Lore Name | Primary Dao Marks & Habitat | Labor Specialization | Social & Behavioral Traits |
| :--- | :--- | :--- | :--- | :--- |
| `HAIRY_MAN` | **Mao Dân** (Người Lông) | **Luyện Đạo (Refinement Path)**; hair-covered humanoids; thrive in Lang Ya-style refinement peaks, volcanic/forge valleys, and resource-rich blessed lands. | **Gu Refinement** (mortal & immortal assisted refinement, material processing, maintaining refinement formations). | Obsessive about refining Gu; simple, stubborn, proud of craftsmanship; easily placated with refinement materials and recipes, but resentful if forced to refine trash or starved of materials. |
| `ROCKMAN` | **Thạch Nhân** (Người Đá) | **Thổ Đạo (Earth Path)**; live deep underground in caverns, mineral veins, and earth trenches; eat soil/minerals and sleep for decades. | **Mining, Earth-Vein Excavation, Stone/Ore Extraction, Earth-Path Construction**. | Slow, stoic, deeply hierarchical (tribes led by elder Rockmen); reproduce slowly by splitting/awakening souls in stone; very high endurance against harsh labor, but if pushed to rebellion they collapse mines/earth veins. |
| `FEATHERMAN` | **Vũ Dân** (Người Lông Vũ) | **Vũ Đạo / Phong Đạo / Vân Đạo (Space, Wind, Cloud)**; winged humanoids living on floating islands, sky-peaks, and high canyons. | **Aerial Logistics, Cloud/Wind/Space Resource Harvesting, Aperture Scouting & Sky Defense**. | Fiercely **freedom-loving**; highest aversion to slavery/oppression (`EXPLOIT` causes rapid rebellion or mass suicide/exodus); honor-bound if treated as free allies. |
| `SNOWMAN` | **Tuyết Dân** (Người Tuyết) | **Băng Tuyết Đạo (Ice & Snow Path)**; thrive in Northern Plains-style icy plains, frost peaks, and glaciers; weakened/killed by intense heat. | **Ice/Snow Material Harvesting, Cold-Storage Preservation, Frost-Path Formations**. | Clan-oriented, xenophobic toward outsiders but fiercely loyal once bound by marriage/alliance contracts (e.g., Xue Er's clan); value purity and cold climates. |
| `INKMAN` | **Mặc Nhân** (Người Mực) | **Trí Đạo / Tín Đạo / Họa Đạo (Wisdom, Information, Painting)**; black-skinned, white-haired scholars living in Inkman Cities/libraries. | **Information Gathering, Recipe Deduction Assistance, Painting/Scripture Production, Administration**. | Scholarly, literate, meritocratic; value knowledge, books, and fair governance over brute force; Excel at managing aperture bureaucracy and boosting research/refinement deduction. |
| `MINIMAN` | **Tiểu Nhân** (Người Tí Hon) | **Mộc Đạo (Wood Path)**; tiny humanoids living in flower-forests, giant herb valleys, and Mini-regions (e.g., Mini Southern Border). | **Herb Gardening, Spirit Plant Cultivation, Food/Crop Farming, Pollination/Ecology Care**. | Timid, industrious, peaceful, high birth rate when food and flora flourish; fragile in combat and vulnerable to disasters, but unmatched at tending high-rank spirit herbs and stabilizing aperture ecology. |
| `MERMAN` | **Giao Nhân** (Nhân Ngư) | **Thủy Đạo (Water Path)**; amphibious scaled humanoids living in Eastern Sea-style oceans, spirit springs, and underwater cities. | **Aquatic Farming (Dragonfish, Pearls), Water-Vein Tending, Sea-Beast Husbandry**. | Matriarchal/Saintess-led religious society (Sea God worship), artistic (singing/pearls), adept at large-scale aquaculture; require water-rich aperture zones. |
| `MUSHROOMMAN` | **Cô Nhân** (Người Nấm) | **Độc Đạo / Mộc Đạo / Thổ Đạo (Poison, Wood, Earth)**; mushroom-capped humanoids dwelling in damp caves, swamps, and toxic valleys. | **Poison-Path Material Harvesting, Toxic Herb Cultivation, Decay/Compost Recycling**. | Reclusive, secretive, naturally resistant to miasma and poison; thrive in environments lethal to other races, turning waste/poison zones into productive assets. |
| `DRAGONMAN` | **Long Nhân** (Người Rồng) | **Biến Hóa Đạo / Nô Đạo (Transformation & Enslavement)**; horned, scaled elites created via Longevity/Dragonman lifespan methods. | **Military Garrison, Beast/Dragonfish Taming, High-Rank Combat Vanguard**. | Proud, martial, naturally talented in cultivation and combat; high pride means heavy oppression breeds dangerous, high-rank rebellions led by Dragonman elites. |

---

### 2.2 Aperture Economy Loop (Upkeep, Exploitation vs. Nurturing, Rebellions)

- **Self-Sustaining Food & Ecology Chain:** In canon (especially Fang Yuan's Sovereign Immortal Aperture), an Immortal Aperture is a miniature world. Variant humans consume **mortal food** produced by aperture resource points (`BEAST_PASTURE`, `YEAR_ESSENCE_POOL`, `MINIMAN` farms, `MERMAN` dragonfish), NOT Immortal Essence—Immortal Essence (Tiên Nguyên) is condensed by the aperture's Heaven & Earth Qi and used by the Gu Immortal to activate Immortal Gu, feed formations, or buy/transmute resources in Treasure Yellow Heaven. Using `immortal_essence` as a silent infinite food fallback breaks both canon and economic tension.
- **Exploitation vs. Nurturing (Dưỡng vs. Áp Bức):**
  - *Short-term exploitation* (high labor intensity, heavy taxation, forced overtime refinement) spikes immediate output and drains race vitality/happiness, driving up **oppression** and **rebellion risk** (and mortality/productivity collapse).
  - *Nurturing* (adequate food reserves, habitat matching, lower labor intensity, housing/security) raises **prosperity** and **population growth**, which in turn increases the birth rate of **talented Gu Masters** and potential **Variant Gu Immortals**.
- **Rebellions & Strikes:** In canon, oppressed variant human tribes do not merely tick a number—they launch **work slowdowns/strikes** (halting point bonuses and auto-refinement), **sabotage** resource points and formations, or **openly revolt** under their strongest Gu Masters/Immortals if loyalty collapses.

---

### 2.3 Variant Gu Immortals (Cổ Tiên Dị Nhân)

1. **Origin & Subjugation:**
   - **Native Ascension:** Within an accelerated Immortal Aperture (`time_rate` 10x–60x+), populous and prosperous variant human tribes produce Rank 5 peak Gu Masters who—with the aperture owner's blessing (supplying Immortal Essence, Heaven/Earth Qi balance, and tribulation protection)—ascend to **Rank 6 Variant Gu Immortals** (e.g., Hairyman Elders in Lang Ya Blessed Land, Snowman/Rockman/Miniman immortals in Fang Yuan's Sovereign Immortal Aperture).
   - **External Subjugation / Migration:** Defeated or allied variant tribes from the Five Regions / Two Heavens (e.g., Lang Ya Sect's Hairy Men, Eastern Sea Mermen, Northern Plains Snowmen) can be relocated into an aperture via **alliance contracts** or **enslavement/loyalty bindings**.
2. **Aperture Governance & Sub-Domain Management:**
   - **Sub-Domain Stewards:** Variant Gu Immortals act as **managers of aperture sub-domains** (e.g., Mini Southern Border, Mini Eastern Sea), overseeing resource points, maintaining **Immortal Formations** (Tiên Trận), and leading **collective Immortal Gu refinement** (like Lang Ya's Hairy Man refinement formation, which drastically boosts success rate and enables Rank 6+ refinement).
   - **Economy Scaling:** Unlike mortal Gu Masters who only tend single points, a Variant Gu Immortal can buff an entire path/region of resource points or stabilize **Dao Mark conflict** within their native biome.
3. **Combat & Tribulation Assistance:**
   - **Tribulation Defense:** When Heaven & Earth Calamities/Tribulations strike the aperture, stationed Variant Gu Immortals mobilize their tribes, man **defense structures / Immortal Formations**, and directly engage tribulation beasts and anomalies inside the aperture to protect resource points and aperture HP.
   - **External Combat Support:** Loyal Variant Gu Immortals with Rank 6+ Immortal Gu can be slotted into the Gu Immortal's **combat formation** (or summoned from the aperture) as high-rank allies, bringing their race's Dao Marks, Immortal Essence reserves, and signature killer moves—subject to loyalty and upkeep.
