# PHASE 4: COMBAT ENGINE, KILLER MOVES & REGIONAL SUPPRESSION

You are an expert full-stack developer (Golang, Fiber/Gin, PostgreSQL, Next.js, TailwindCSS). 
We are implementing Phase 4 of the "Reverend Insanity" text-based RPG. This phase introduces the core Combat Engine, "Killer Moves" (combining multiple Gu bugs), and the strict lore rules: Level Penalty and Regional Suppression.

Please implement the following tasks step-by-step:

### 1. Backend: Update Models & Setup Strategy Pattern (Golang)
Update the database schemas and models to support combat.
- **Player Model Update:** Add `HP` (float, default 100), `CurrentRegion` (string, e.g., "Southern_Border"), and `DaoMarks` (JSONB, e.g., `{"strength": 100, "fire": 0}`).
- **GuDictionary Update:** Add `OriginRegion` (string), `BaseDamage` (float), and `BaseDefense` (float).

**Implement the Gu Ability Strategy Pattern:**
1. Create a `CombatContext` struct containing `Attacker *Player`, `Defender *Enemy/Player`, and `Logs []string`.
2. Create an interface `GuAbility` with a method `Execute(ctx *CombatContext) error`.
3. Create a Registry map linking Gu IDs to their specific logic structs (e.g., `MoonlightGu`, `AllOutEffortGu`).

### 2. Backend: Core Combat Math & Lore Rules (Golang)
Create a centralized function `CalculateGuEffect(player *Player, gu *Gu) (effectivePower float64, manaCost float64, err error)` that strictly follows these rules:

- **Rule 1: Level Penalty (Backlash):** If `player.Rank < gu.Rank`, return an error "Backlash: Aperture damaged!" and deduct 50 HP from the player. They cannot use this Gu.
- **Rule 2: Regional Suppression:** If `gu.OriginRegion != player.CurrentRegion`, the Gu's `effectiveRank` is reduced by 1 (minimum Rank 1). However, its `manaCost` remains based on its original Rank.
- **Rule 3: Mana Calculation:** 
  - Base cost is 10% of Max Mana. 
  - If `player.Rank > gu.Rank`, divide the cost by 10 for each rank difference.
  - Apply Sub-rank division: Divide the final cost by 2 for each sub-rank above initial (Mid = /2, High = /4, Peak = /8).

### 3. Backend: Killer Moves Execution API
Create an endpoint `POST /api/combat/attack`.
- **Payload:** `{ "player_id": "uuid", "target_id": "uuid", "equipped_gu_ids": ["GU_ALL_OUT_EFFORT", "GU_BITTER_STRENGTH"] }`
- **Logic:**
  1. Start a DB Transaction.
  2. Iterate through `equipped_gu_ids`. Calculate total mana cost using the function from Step 2. If insufficient mana, rollback and return 400.
  3. Calculate `Total_Base_Damage`. 
  4. Apply **Dao Mark Multiplier**: `Final_Damage = Total_Base_Damage * (1 + (player.DaoMarks["strength"] / 100.0))`. Every 100 Dao marks = +100% damage.
  5. Deduct Target's Defense. Apply damage to Target HP.
  6. Generate a detailed Combat Log string array (e.g., `["Used All-out Effort Gu! Mana -5.", "Summoned Beast Phantom, dealt 150 damage!"]`).
  7. Commit and return updated stats and logs.

### 4. Frontend: Text-based Combat UI (Next.js)
Build a `CombatScreen` component.
- **Top Section (Split Screen):** Left side shows Player Stats (HP, Mana Bar, Dao Marks). Right side shows Enemy/Target Stats (HP).
- **Middle Section (Scrollable Log):** A `div` with `overflow-y-auto` displaying the combat text logs returned from the backend. Style it like a terminal or retro RPG log.
- **Bottom Section (Action Bar):** Show the player's equipped Gu bugs. 
- **Interactivity:** User selects Gu bugs to form a "Killer Move" and clicks "Execute". Use optimistic UI updates via Zustand to instantly drop the Mana bar for responsiveness, then reconcile with the API response. Shake the screen (CSS keyframes) if the API returns a "Backlash" error.

---

# PHASE 5: WORLD EXPLORATION & PVE MAP NODES

### 1. Backend: Map Nodes & Exploration Engine (Golang)
Create an endpoint `POST /api/explore`.
- **Static Map Data:** Define areas like "Qing Mao Mountain" (Southern Border) or "Three Kings Inheritance" (Southern Border).
- **Payload:** `{ "player_id": "uuid", "node_id": "string" }`
- **Logic:**
  1. Deduct 5 Mana from the player.
  2. Use `math/rand` to trigger a random event:
     - **40% chance - Resource:** Found 1-3 Primeval Stones.
     - **30% chance - Combat:** Encountered a Wild Beast (e.g., "Wild Boar"). Returns a flag to redirect frontend to Combat Screen.
     - **20% chance - Materials:** Found Gu Food (e.g., "Moon Orchid Petal").
     - **10% chance - Nothing:** "You wandered the bamboo forest but found nothing."
  3. Update player inventory/stats and return the event result.

### 2. Frontend: Exploration UI (Next.js)
Build an `ExplorationMap` component.
- Display map nodes as simple clickable cards or a skill-tree-like layout.
- When clicking a node, show a loading spinner, call the `/api/explore` endpoint.
- Display a sleek Modal or Toast with the randomized text result. If a beast is encountered, transition the view to the `CombatScreen`.

--------------------------------------------------------------------------------
Lời khuyên thiết kế (Game Design Notes) dành cho bạn:
1. Về Sát Chiêu (Killer Moves) ở Task 4.2: Trong nguyên tác, Phương Nguyên dùng Cổ Toàn Lực Ứng Phó + Cổ Khổ Lực + Cổ Tự Lực Cánh Sinh tạo thành một Sát Chiêu Lực Đạo hoàn chỉnh. Ở Frontend, bạn hãy cho phép user "Click chọn" nhiều con Cổ đang trang bị cùng lúc, sau đó bấm 1 nút "Đánh / Execute". Backend sẽ loop qua mảng các ID Cổ trùng này, tính tổng sát thương và trừ tổng Mana. Đây chính là cảm giác "Lắp ghép chiêu thức" (Deck-building) cực kỳ gây nghiện.
2. Về Áp chế Dị vực (Regional Suppression) ở Task 4.3: Bạn phải nói rõ với AI Agent rằng: Áp chế dị vực làm giảm cấp độ uy lực của Cổ trùng (Rank - 1), nhưng Lượng Mana tiêu hao vẫn giữ nguyên ở mức ban đầu. Điều này khiến việc mang Cổ từ map này sang map khác là một hành động cực kỳ hao tổn tài nguyên, ép người chơi phải đi "Khám phá" (Phase 5) ở map mới để bắt Cổ trùng bản địa. Đây là Core Game Loop giữ chân người chơi lâu dài.
3. Hiệu ứng UX ở Frontend (Task 4.4): Game Text-based thì Text Log là linh hồn. Khung <ScrollableLog /> cần được AI code sao cho mỗi lần có dòng log mới (VD: "Nguyệt nhận chém trúng con Khỉ đá, gây 50 sát thương!"), nó sẽ tự động cuộn (auto-scroll) xuống dưới cùng. Nếu user xài Cổ vượt cấp bị Phản phệ (Backlash), hãy yêu cầu AI thêm class CSS rung lắc màn hình animate-shake và chớp viền đỏ để tạo cảm giác đau đớn chân thực nhất.