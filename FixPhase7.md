# PHASE 2.5: REFACTOR GU DICTIONARY (SPARSE MATRIX OPTIMIZATION)

You are an expert full-stack developer (Golang, Fiber/Gin, PostgreSQL, Next.js, TailwindCSS). 
In our "Reverend Insanity" RPG project, our current `Gu_Dictionary` database table is suffering from a "Sparse Matrix" problem (too many specific stat columns like `base_damage`, `base_defense` that are mostly 0.0 for non-combat Gu).

We need to refactor the database schema, Golang models, and related logic to use a flexible JSONB `effects` column, add a `path` (element) column for Dao Mark calculations, and add a `base_mana_cost` column.

Please implement the following refactoring steps:

### 1. Database Schema Update (PostgreSQL)
Update the migration script to recreate or alter the `gu_dictionary` table to match this optimized structure:

- Drop columns: `base_power`, `base_damage`, `base_defense`.
- Add column: `path` (VARCHAR). Represents the Gu's path (e.g., 'Moon', 'Strength', 'Water', 'Rule').
- Add column: `base_mana_cost` (FLOAT, default 0.0). Represents the % of mana consumed.
- Add column: `effects` (JSONB, default '{}'). This will dynamically store all stats.
- Update Enum: Ensure `gu_type` enum includes: 'Active', 'Passive', 'Support', 'Consumable'.

### 2. Update Golang Model (GORM)
Refactor the `GuDictionary` struct in Golang:

```go
type GuDictionary struct {
	ID            string             `json:"id" gorm:"primaryKey"`
	Name          string             `json:"name"`
	Rank          int                `json:"rank"`
	Path          string             `json:"path"` 
	GuType        string             `json:"gu_type"`
	OriginRegion  string             `json:"origin_region"`
	BaseManaCost  float64            `json:"base_mana_cost"`
	Effects       map[string]float64 `json:"effects" gorm:"type:jsonb"` // Flexible stats
	FoodID        string             `json:"food_id"`
	HungerRate    int                `json:"hunger_rate"`
	MaxHunger     int                `json:"max_hunger"`
	Description   string             `json:"description"`
}
3. Refactor the Seeder Data (Backend)
Update the master data seeder to match the new JSONB structure. Here are the exact mappings for the initial Gu bugs:
GU_MOONLIGHT: path: "Moon", base_mana_cost: 10.0, effects: {"damage": 50.0}
GU_JADE_SKIN: path: "Earth", base_mana_cost: 0.0, effects: {"add_base_defense": 40.0}
GU_LIQUOR_WORM: path: "Rule", base_mana_cost: 0.0, effects: {"refine_mana_tier": 1.0}
GU_WHITE_BOAR: path: "Strength", base_mana_cost: 0.0, effects: {"add_base_attack": 15.0}
4. Refactor Combat Logic (Golang)
Update the CombatService or CalculateGuEffect function to read from the new Effects map instead of the hardcoded columns.
Example logic update: When an 'Active' Gu attacks, the base damage is now gu.Effects["damage"].
Example logic update: When equipping a 'Passive' Gu like White Boar, add gu.Effects["add_base_attack"] to the Player's base stats.
5. Frontend UI Update (Next.js)
Refactor the GuCard component in the Inventory/Shop UI. Since stats are no longer fixed columns, iterate through the effects object to display stats dynamically.
Example: Object.entries(gu.effects).map(([key, value]) => <p>{FormatKey(key)}: {value}</p>)
Create a helper function FormatKey to turn keys like add_base_attack into readable text (e.g., "Bonus Attack").
Please provide the updated Golang Model, the updated Seeder function, and the updated Next.js Component rendering the stats.

***

**Lợi ích của Prompt này:**
1. **Giảm thiểu lỗi (Bug):** Nó nhắc AI phải xóa các cột cũ (`Drop columns`) và cập nhật lại cả Seeder (hàm tạo dữ liệu mẫu), nếu không code cũ sẽ bị crash khi query Database.
2. **Định hướng Frontend:** Frontend trước đây có thể đang code cứng là `<div>{gu.base_damage}</div>`. Prompt này yêu cầu AI viết một hàm vòng lặp (`Object.entries`) để render linh hoạt bất kỳ chỉ số nào từ JSON (Ví dụ: Máu, Giáp, Độc, Tốc độ đánh...).
3. **Mở đường cho hệ thống Đạo Ngân:** Bằng cách ép AI đưa cột `path` (Lưu phái) vào, bạn đã hoàn tất bước chuẩn bị cho Phase tính toán sát thương kết hợp Đạo Ngân ở các cấp độ Tiên Cổ sau này.


Để hệ thống Cổ Trùng và sức mạnh của nhân vật khớp hoàn toàn với bối cảnh các Lưu phái (Cultivation Paths) và Cảnh giới lưu phái (Mastery Levels) như bạn vừa cung cấp, chúng ta cần mở rộng bản thiết kế Phase 2.5. 

Ngoài việc tối ưu Database theo dạng JSONB, chúng ta sẽ yêu cầu AI thêm luôn hệ thống **Path Mastery (Cảnh giới lưu phái)** vào dữ liệu người chơi, đồng thời dùng nó làm hệ số nhân (Multiplier) trong chiến đấu.

Dưới đây là bản Prompt đã được làm lại, tích hợp hoàn hảo cả việc tối ưu DB lẫn hệ thống Lưu phái chuẩn nguyên tác. Bạn hãy copy toàn bộ khối Markdown dưới đây và giao cho AI Agent:

***

```markdown
# PHASE 2.5: REFACTOR GU DICTIONARY & CULTIVATION PATHS SYSTEM

You are an expert full-stack developer (Golang, Fiber/Gin, PostgreSQL, Next.js, TailwindCSS). 
In our "Reverend Insanity" RPG project, we need to solve the "Sparse Matrix" problem in our `Gu_Dictionary` table by using a JSONB `effects` column. 

Furthermore, we are introducing the Lore-accurate **Cultivation Paths** and **Mastery Levels** system. A player's mastery in a specific path will dictate their combat multipliers and crafting success rates.

Please implement the following refactoring steps step-by-step:

### 1. Define Constants & Enums (Golang)
Create strictly typed constants for the Paths and Mastery Levels based on the novel's lore:

**Cultivation Paths:**
- `Time Path` (Trụ Đạo), `Space Path` (Vũ Đạo), `Strength Path` (Lực Đạo), `Wisdom Path` (Trí Đạo), `Enslavement Path` (Nô Đạo), `Dream Path` (Mộng Đạo), `Fire Path`, `Water Path`, `Earth Path`, `Sword Path`, `Refinement Path`, `Soul Path`, `Blood Path`, `Luck Path`.

**Mastery Levels:**
- `Ordinary` (0 multiplier bonus)
- `Master` (+10% effect/damage bonus)
- `Grandmaster` (+25% effect/damage bonus)
- `Supreme Grandmaster` (+50% effect/damage bonus)

### 2. Database Schema Update (PostgreSQL)
Update the migration scripts to recreate or alter the tables:
- **`gu_dictionary` table:**
  - Drop specific stat columns: `base_power`, `base_damage`, `base_defense`.
  - Add column: `path` (VARCHAR). Must be one of the Cultivation Paths defined above.
  - Add column: `base_mana_cost` (FLOAT, default 0.0).
  - Add column: `effects` (JSONB, default '{}'). This will dynamically store all stats (e.g., `{"damage": 50}`).
- **`players` table:**
  - Add column: `path_masteries` (JSONB, default '{}'). Example structure: `{"Strength Path": "Master", "Time Path": "Ordinary"}`.

### 3. Update Golang Models (GORM)
Refactor the structs to match the new schema:

```go
type Player struct {
	// ... existing fields (id, hp, mana, etc.)
	PathMasteries map[string]string `json:"path_masteries" gorm:"type:jsonb"`
}

type GuDictionary struct {
	ID            string             `json:"id" gorm:"primaryKey"`
	Name          string             `json:"name"`
	Rank          int                `json:"rank"`
	Path          string             `json:"path"` // e.g., "Strength Path"
	GuType        string             `json:"gu_type"`
	OriginRegion  string             `json:"origin_region"`
	BaseManaCost  float64            `json:"base_mana_cost"`
	Effects       map[string]float64 `json:"effects" gorm:"type:jsonb"` // Flexible stats
	FoodID        string             `json:"food_id"`
	HungerRate    int                `json:"hunger_rate"`
	MaxHunger     int                `json:"max_hunger"`
	Description   string             `json:"description"`
}
```

### 4. Refactor the Seeder Data (Backend)
Update the master data seeder to match the new JSONB structure and the official Paths:
- **GU_MOONLIGHT**: path: "Moon Path", base_mana_cost: 10.0, effects: `{"damage": 50.0}`
- **GU_JADE_SKIN**: path: "Earth Path", base_mana_cost: 0.0, effects: `{"add_base_defense": 40.0}`
- **GU_LIQUOR_WORM**: path: "Refinement Path", base_mana_cost: 0.0, effects: `{"refine_mana_tier": 1.0}`
- **GU_WHITE_BOAR**: path: "Strength Path", base_mana_cost: 0.0, effects: `{"add_base_attack": 15.0}`

### 5. Refactor Combat & Game Logic (Golang)
Update the `CombatService` or `CalculateGuEffect` function to incorporate **Path Mastery Multipliers**:
1. Read the base effect from `gu.Effects["damage"]` or `gu.Effects["add_base_defense"]`.
2. Check the player's mastery for that specific Gu's `Path` inside `player.PathMasteries`.
3. Apply the multiplier: 
   - If "Master", multiply the effect by `1.1`.
   - If "Grandmaster", multiply by `1.25`.
   - If "Supreme Grandmaster", multiply by `1.5`.
   - If none/Ordinary, multiply by `1.0`.

### 6. Frontend UI Update (Next.js)
Refactor the `GuCard` and `PlayerStats` components:
- **GuCard:** Iterate through the `effects` JSON object to display stats dynamically (e.g., `Object.entries(gu.effects).map(...)`). Display the `Path` elegantly with an icon or specific color tag.
- **PlayerStats:** Create a new "Mastery" Tab or Section that renders the `path_masteries` object, showing the player's current proficiency in different paths.
```

***

### Lợi ích của việc làm lại Prompt này:
1. **Chuẩn hóa Logic Nguyên tác:** Mọi con Cổ bây giờ đều thuộc một lưu phái cụ thể (`Strength Path`, `Time Path`...).
2. **Khuyến khích chuyên môn hóa:** Người chơi sẽ có động lực để sưu tầm các Cổ trùng cùng một lưu phái. Nếu người chơi tu luyện lên được `Grandmaster` của `Strength Path`, thì tất cả Cổ Trùng thuộc `Strength Path` họ trang bị đều được cộng thẳng 25% uy lực hoặc chỉ số. Điều này mô phỏng chân thực cảm giác "Lấy một đạo chiếu vạn đạo" trong truyện.
3. **Mở đường cho Mộng Cảnh (Phase sau):** Bằng cách lưu `path_masteries` dưới dạng JSONB, sau này khi bạn làm tính năng Phó Bản Mộng Cảnh (Dream Realm), Backend chỉ cần chọc vào JSONB này để nâng cấp từ `Ordinary` -> `Master` -> `Grandmaster` khi người chơi vượt ải thành công. Dữ liệu sẽ cực kỳ gọn nhẹ!