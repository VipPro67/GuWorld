# MonsterBossGuild.md — Plan triển khai Monster/Boss có Cổ, Item, Loot và Combat tối ưu

## Mục tiêu

Code hiện tại đang hardcode nhiều phần:

- `GenerateCell()` random resource/monster/wildGu theo `region`.
- `MovePlayer()` hardcode stat quái theo tên.
- `CombatAttack()` enemy thường chỉ có `EnemyDamage` / `EnemyDefense`.
- Boss chưa có `GuInventory`, `ItemInventory`, `AIProfile`, `LootTable`.
- Loot sau khi thắng đang hardcode trong combat.

Mục tiêu refactor:

```txt
1. Lưu quái thường, boss, item, cổ trùng của nó vào database
2. Lưu spawn theo region/map cell
3. Combat dùng chung action builder cho Player và Boss
4. Boss có thể equip Cổ và thi triển Cổ như player
5. Loot lấy từ loot_table, không hardcode
```

---

## 1. Kiến trúc mới

```txt
MovePlayer
-> ResolveCell
-> Load spawn từ DB hoặc procedural fallback
-> Load MonsterTemplate
-> Build ActiveCombat snapshot
-> CombatAttack
-> Build player actions
-> Build boss/enemy actions từ GuInventory + AIProfile
-> Resolve combat
-> Nếu thắng: RollLoot
-> Save player
```

---

## 2. Database đề xuất

### regions

```sql
CREATE TABLE regions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    width INT NOT NULL DEFAULT 100,
    height INT NOT NULL DEFAULT 100,
    min_rank INT NOT NULL DEFAULT 1,
    max_rank INT NOT NULL DEFAULT 5,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### monster_templates

```sql
CREATE TABLE monster_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    rank INT NOT NULL DEFAULT 1,
    sub_rank INT NOT NULL DEFAULT 1,
    path TEXT,
    max_hp DOUBLE PRECISION NOT NULL,
    base_attack DOUBLE PRECISION NOT NULL,
    base_defense DOUBLE PRECISION NOT NULL,
    max_mana DOUBLE PRECISION NOT NULL DEFAULT 100,
    current_mana DOUBLE PRECISION NOT NULL DEFAULT 100,
    gu_inventory JSONB NOT NULL DEFAULT '[]',
    item_inventory JSONB NOT NULL DEFAULT '{}',
    dao_marks JSONB NOT NULL DEFAULT '{}',
    path_masteries JSONB NOT NULL DEFAULT '{}',
    ai_profile TEXT NOT NULL DEFAULT 'beast_basic',
    loot_table_id TEXT,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

`type`:

```txt
Normal
Elite
Boss
HumanCultivator
BeastKing
DesolateBeast
```

### monster_spawns

```sql
CREATE TABLE monster_spawns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_id TEXT NOT NULL REFERENCES regions(id),
    monster_id TEXT NOT NULL REFERENCES monster_templates(id),
    x INT,
    y INT,
    spawn_mode TEXT NOT NULL DEFAULT 'Random',
    chance DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    respawn_seconds INT NOT NULL DEFAULT 300,
    min_rank INT NOT NULL DEFAULT 1,
    max_rank INT NOT NULL DEFAULT 9,
    is_boss_spawn BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

`spawn_mode`:

```txt
Fixed
Random
Event
BossRoom
```

### loot_tables

```sql
CREATE TABLE loot_tables (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    entries JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

`entries`:

```json
[
  {
    "item_id": "boar_meat",
    "min_qty": 1,
    "max_qty": 2,
    "chance": 0.6
  },
  {
    "gu_id": "GU_WHITE_BOAR",
    "chance": 0.03
  }
]
```

### map_cells optional

Chỉ lưu cell đặc biệt, không cần lưu toàn bộ map.

```sql
CREATE TABLE map_cells (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_id TEXT NOT NULL REFERENCES regions(id),
    x INT NOT NULL,
    y INT NOT NULL,
    cell_type TEXT NOT NULL,
    content_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    respawn_seconds INT NOT NULL DEFAULT 300,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE(region_id, x, y)
);
```

Dùng cho:

```txt
Boss room
NPC
Inheritance
Shop
Teleport
Quest location
Resource đặc biệt
```

---

## 3. Go models đề xuất

### MonsterTemplate

```go
type MonsterTemplate struct {
	ID             string                    `json:"id"`
	Name           string                    `json:"name"`
	Type           string                    `json:"type"`
	Rank           int                       `json:"rank"`
	SubRank        int                       `json:"sub_rank"`
	Path           string                    `json:"path"`
	MaxHP          float64                   `json:"max_hp"`
	BaseAttack     float64                   `json:"base_attack"`
	BaseDefense    float64                   `json:"base_defense"`
	MaxMana        float64                   `json:"max_mana"`
	CurrentMana    float64                   `json:"current_mana"`
	GuInventory    models.GuInventory        `json:"gu_inventory"`
	ItemInventory  models.MaterialsInventory `json:"item_inventory"`
	DaoMarks       map[string]int            `json:"dao_marks"`
	PathMasteries  map[string]string         `json:"path_masteries"`
	AIProfile      string                    `json:"ai_profile"`
	LootTableID    string                    `json:"loot_table_id"`
	Description    string                    `json:"description"`
}
```

### Combatant

Tạo struct trung gian để Player/Boss dùng chung combat.

```go
type Combatant struct {
	ID            string
	Name          string
	Kind          string
	Rank          int
	SubRank       int
	HP            float64
	MaxHP         float64
	CurrentMana   float64
	MaxMana       float64
	BaseAttack    float64
	BaseDefense   float64
	GuInventory   models.GuInventory
	DaoMarks      map[string]int
	PathMasteries map[string]string
	AIProfile     string
	LootTableID   string
}
```

---

## 4. Mở rộng CombatState

Hiện `CombatState` nên thêm các field này:

```go
type CombatState struct {
	EnemyTemplateID    string             `json:"enemy_template_id"`
	EnemyName          string             `json:"enemy_name"`
	EnemyType          string             `json:"enemy_type"`
	EnemyHP            float64            `json:"enemy_hp"`
	EnemyMaxHP         float64            `json:"enemy_max_hp"`
	EnemyMana          float64            `json:"enemy_mana"`
	EnemyMaxMana       float64            `json:"enemy_max_mana"`
	EnemyDamage        float64            `json:"enemy_damage"`
	EnemyDefense       float64            `json:"enemy_defense"`
	EnemyRank          int                `json:"enemy_rank"`
	EnemySubRank       int                `json:"enemy_sub_rank"`
	EnemyGuInventory   models.GuInventory `json:"enemy_gu_inventory"`
	EnemyDaoMarks      map[string]int     `json:"enemy_dao_marks"`
	EnemyPathMasteries map[string]string  `json:"enemy_path_masteries"`
	AIProfile          string             `json:"ai_profile"`
	LootTableID         string             `json:"loot_table_id"`
	ChargePoints       int                `json:"charge_points"`
	TargetPlayerID     string             `json:"target_player_id,omitempty"`
}
```

---

## 5. Build ActiveCombat từ MonsterTemplate

```go
func BuildCombatStateFromMonster(mon MonsterTemplate) models.CombatState {
	return models.CombatState{
		EnemyTemplateID:    mon.ID,
		EnemyName:          mon.Name,
		EnemyType:          mon.Type,
		EnemyHP:            mon.MaxHP,
		EnemyMaxHP:         mon.MaxHP,
		EnemyMana:          mon.CurrentMana,
		EnemyMaxMana:       mon.MaxMana,
		EnemyDamage:        mon.BaseAttack,
		EnemyDefense:       mon.BaseDefense,
		EnemyRank:          mon.Rank,
		EnemySubRank:       mon.SubRank,
		EnemyGuInventory:   mon.GuInventory,
		EnemyDaoMarks:      mon.DaoMarks,
		EnemyPathMasteries: mon.PathMasteries,
		AIProfile:          mon.AIProfile,
		LootTableID:        mon.LootTableID,
	}
}
```

---

## 6. Convert Player/Boss thành Combatant

```go
func CombatantFromPlayer(player *models.Player) Combatant {
	return Combatant{
		ID:            player.ID.String(),
		Name:          player.Username,
		Kind:          "Player",
		Rank:          player.Rank,
		SubRank:       player.SubRank,
		HP:            player.HP,
		MaxHP:         player.BaseHP,
		CurrentMana:   player.CurrentMana,
		MaxMana:       player.MaxMana,
		GuInventory:   player.GuInventory,
		DaoMarks:      player.DaoMarks,
		PathMasteries: player.PathMasteries,
	}
}
```

```go
func CombatantFromActiveCombat(cs *models.CombatState) Combatant {
	return Combatant{
		ID:            cs.EnemyTemplateID,
		Name:          cs.EnemyName,
		Kind:          cs.EnemyType,
		Rank:          cs.EnemyRank,
		SubRank:       cs.EnemySubRank,
		HP:            cs.EnemyHP,
		MaxHP:         cs.EnemyMaxHP,
		CurrentMana:   cs.EnemyMana,
		MaxMana:       cs.EnemyMaxMana,
		BaseAttack:    cs.EnemyDamage,
		BaseDefense:   cs.EnemyDefense,
		GuInventory:   cs.EnemyGuInventory,
		DaoMarks:      cs.EnemyDaoMarks,
		PathMasteries: cs.EnemyPathMasteries,
		AIProfile:     cs.AIProfile,
		LootTableID:   cs.LootTableID,
	}
}
```

---

## 7. Tối ưu combat cho Boss có Cổ

### Rule mới

```txt
Nếu enemy có EnemyGuInventory:
  build action từ Cổ đang equipped + ACTIVE
Nếu không có Gu:
  dùng basic enemy action như hiện tại
```

### BuildEnemyActions

```go
func BuildEnemyActions(ctx *CombatContext, enemy Combatant) []Action {
	actions := []Action{}

	for _, gu := range enemy.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}

		spec, ok := models.GuRegistry[gu.GuID]
		if !ok {
			continue
		}

		action, err := BuildCombatAction(ctx, spec)
		if err == nil {
			actions = append(actions, action)
		}
	}

	if len(actions) > 0 {
		return SelectActionsByAI(enemy, actions)
	}

	return BuildBasicEnemyAction(enemy)
}
```

---

## 8. Boss AI Profile

```txt
beast_basic
defensive
aggressive
boss_balanced
boss_burst
boss_summoner
boss_poison
boss_strength
```

```go
func SelectActionsByAI(enemy Combatant, actions []Action) []Action {
	switch enemy.AIProfile {
	case "boss_burst":
		return selectTopDamageActions(actions, 2)
	case "defensive":
		return selectDefenseFirst(actions, 2)
	case "boss_poison":
		return selectStatusEffectActions(actions, 2)
	case "boss_summoner":
		return selectUtilityFirst(actions, 2)
	default:
		return selectBalancedActions(actions, 2)
	}
}
```

---

## 9. CalculateGuEffect cho Player và Boss

Hiện function nhận `*models.Player`.

Nên tạo bản chung:

```go
func CalculateGuEffectForCombatant(actor Combatant, guSpec *models.GuDictionaryItem) (float64, float64, float64, error) {
	if actor.Rank < guSpec.Rank {
		return 0, 0, 0, fmt.Errorf("%s không đủ cảnh giới dùng %s", actor.Name, guSpec.Name)
	}

	powerScale := 1.0
	masteryMultiplier := getActorPathMasteryMultiplier(actor, guSpec.Path)

	baseDmg := guSpec.Effects["damage"] + guSpec.Effects["add_base_attack"]
	baseDef := guSpec.Effects["defense"] + guSpec.Effects["add_base_defense"]

	damage := baseDmg * powerScale * masteryMultiplier
	defense := baseDef * powerScale * masteryMultiplier
	cost := calculateEnergyCostForRank(actor.Rank, guSpec)

	return damage, defense, cost, nil
}
```

Giữ adapter cũ:

```go
func CalculateGuEffect(player *models.Player, guSpec *models.GuDictionaryItem) (float64, float64, float64, error) {
	actor := CombatantFromPlayer(player)
	return CalculateGuEffectForCombatant(actor, guSpec)
}
```

---

## 10. ResolveCell thay GenerateCell trực tiếp

```go
func ResolveCell(tx *sql.Tx, region string, x int, y int) (Cell, error) {
	special, ok := FindSpecialCell(tx, region, x, y)
	if ok {
		return special, nil
	}

	spawn, ok := FindFixedSpawn(tx, region, x, y)
	if ok {
		return Cell{X: x, Y: y, Type: "Monster", Content: spawn.MonsterID}, nil
	}

	cell := GenerateCell(x, y, region)

	if cell.Type == "Monster" {
		monsterID := PickRandomMonsterFromRegion(tx, region)
		cell.Content = monsterID
	}

	return cell, nil
}
```

`GenerateCell()` chỉ còn fallback procedural.

---

## 11. MovePlayer refactor

Thay đoạn hardcode stat theo name bằng:

```go
cell, err := ResolveCell(tx, player.CurrentRegion, player.X, player.Y)
if err != nil {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve cell"})
	return
}

if cell.Type == "Monster" || cell.Type == "Boss" {
	monster, err := LoadMonsterTemplate(tx, cell.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load monster"})
		return
	}

	combat := BuildCombatStateFromMonster(monster)
	player.ActiveCombat = &combat

	logMsg = fmt.Sprintf("Nguy hiểm! Đạo hữu chạm trán %s!", monster.Name)
}
```

Không còn:

```go
if cell.Content == "Mãng Xà Dung Nham (Lava Python)" {
}
```

---

## 12. CombatAttack refactor

### Player actions

Giữ flow:

```txt
req.EquippedGuIDs
-> selectedSpecs
-> BuildCombatAction
```

### Enemy actions

Thay random enemy action bằng:

```go
enemy := CombatantFromActiveCombat(player.ActiveCombat)

enemyCtx := &CombatContext{
	Attacker: enemy,
	Logs: []string{},
}

enemyActions := BuildEnemyActions(enemyCtx, enemy)
```

Nếu chưa đổi `CombatContext.Attacker` từ `*models.Player` sang `Combatant`, tạo `CombatContextV2` hoặc đổi dần.

Khuyến nghị cuối:

```go
type CombatContext struct {
	Attacker Combatant
	Defender Combatant
	Logs []string
	DamageBoost float64
}
```

---

## 13. Enemy mana

Boss có Cổ phải tốn mana.

```go
if enemy.CurrentMana >= action.ManaCost {
	enemy.CurrentMana -= action.ManaCost
} else {
	action = BuildBasicEnemyAction(enemy)
}
```

Sau turn phải sync lại:

```go
player.ActiveCombat.EnemyMana = enemy.CurrentMana
player.ActiveCombat.EnemyHP = enemy.HP
```

---

## 14. Loot resolver

### Loot structs

```go
type LootEntry struct {
	ItemID string  `json:"item_id"`
	GuID   string  `json:"gu_id"`
	MinQty int     `json:"min_qty"`
	MaxQty int     `json:"max_qty"`
	Chance float64 `json:"chance"`
}

type LootTable struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Entries []LootEntry `json:"entries"`
}

type LootResult struct {
	Items map[string]int
	Gus   []models.GuInstance
}
```

### RollLoot

```go
func RollLoot(table LootTable) LootResult {
	result := LootResult{
		Items: map[string]int{},
		Gus: []models.GuInstance{},
	}

	for _, entry := range table.Entries {
		if rand.Float64() > entry.Chance {
			continue
		}

		if entry.ItemID != "" {
			qty := entry.MinQty
			if entry.MaxQty > entry.MinQty {
				qty += rand.Intn(entry.MaxQty - entry.MinQty + 1)
			}
			result.Items[entry.ItemID] += qty
		}

		if entry.GuID != "" {
			result.Gus = append(result.Gus, models.GuInstance{
				InstanceID: uuid.New().String(),
				GuID: entry.GuID,
				Hunger: 100,
				Status: "ACTIVE",
				Equipped: false,
			})
		}
	}

	return result
}
```

### ApplyLoot

```go
func ApplyLoot(player *models.Player, loot LootResult) []string {
	logs := []string{}

	if player.MaterialsInventory == nil {
		player.MaterialsInventory = make(models.MaterialsInventory)
	}

	for itemID, qty := range loot.Items {
		player.MaterialsInventory[itemID] += qty
		logs = append(logs, fmt.Sprintf("Thu hoạch %dx %s.", qty, itemID))
	}

	for _, gu := range loot.Gus {
		player.GuInventory = append(player.GuInventory, gu)
		logs = append(logs, fmt.Sprintf("Thu phục Cổ trùng: %s.", gu.GuID))
	}

	return logs
}
```

---

## 15. Victory flow mới

Trong `CombatAttack`, khi thắng:

```go
if victory {
	if player.ActiveCombat.LootTableID != "" {
		table, err := LoadLootTable(tx, player.ActiveCombat.LootTableID)
		if err == nil {
			loot := RollLoot(table)
			lootLogs := ApplyLoot(&player, loot)
			ctx.Logs = append(ctx.Logs, lootLogs...)
		}
	}

	player.ActiveCombat = nil
}
```

Xóa hardcode:

```go
player.MaterialsInventory["boar_meat"]++
player.MaterialsInventory["wild_boar_tusk"]++
```

---

## 16. Seed boss mẫu

### Loot table

```sql
INSERT INTO loot_tables (id, name, entries)
VALUES (
  'loot_boss_wolf_king',
  'Loot Độc Lang Vương',
  '[
    {"item_id": "wolf_fang", "min_qty": 2, "max_qty": 5, "chance": 0.8},
    {"item_id": "wolf_pelt", "min_qty": 1, "max_qty": 2, "chance": 0.5},
    {"gu_id": "GU_WOLF_ENSLAVEMENT_3", "chance": 0.05}
  ]'::jsonb
);
```

### Boss template

```sql
INSERT INTO monster_templates (
    id, name, type, rank, sub_rank, path,
    max_hp, base_attack, base_defense,
    max_mana, current_mana,
    gu_inventory,
    path_masteries,
    ai_profile,
    loot_table_id,
    description
)
VALUES (
    'boss_wolf_king',
    'Độc Lang Vương',
    'Boss',
    3,
    4,
    'Enslavement Path',
    500,
    35,
    18,
    120,
    120,
    '[
      {
        "instance_id": "boss-wolf-enslave",
        "gu_id": "GU_WOLF_ENSLAVEMENT_3",
        "hunger": 100,
        "status": "ACTIVE",
        "equipped": true
      },
      {
        "instance_id": "boss-moon-whirl",
        "gu_id": "GU_MOON_WHIRL",
        "hunger": 100,
        "status": "ACTIVE",
        "equipped": true
      }
    ]'::jsonb,
    '{"Enslavement Path": "Master", "Moon Path": "Ordinary"}'::jsonb,
    'boss_balanced',
    'loot_boss_wolf_king',
    'Lang vương thống lĩnh đàn sói, biết dùng Cổ nô đạo.'
);
```

### Boss spawn

```sql
INSERT INTO monster_spawns (
    region_id,
    monster_id,
    x,
    y,
    spawn_mode,
    chance,
    respawn_seconds,
    is_boss_spawn
)
VALUES (
    'gu_luan_slope',
    'boss_wolf_king',
    80,
    80,
    'BossRoom',
    1.0,
    3600,
    TRUE
);
```

---

## 17. Repository cần tạo

```go
func LoadMonsterTemplate(tx *sql.Tx, id string) (MonsterTemplate, error)

func FindFixedSpawn(tx *sql.Tx, regionID string, x int, y int) (MonsterSpawn, bool, error)

func PickRandomMonsterFromRegion(tx *sql.Tx, regionID string, playerRank int) (string, error)

func LoadLootTable(tx *sql.Tx, id string) (LootTable, error)

func FindSpecialCell(tx *sql.Tx, regionID string, x int, y int) (Cell, bool, error)
```

---

## 18. Phase triển khai

### Phase 1 — Tách monster template

```txt
[ ] Tạo bảng monster_templates
[ ] Tạo MonsterTemplate model
[ ] Tạo LoadMonsterTemplate
[ ] Seed quái hiện tại vào DB
[ ] MovePlayer dùng LoadMonsterTemplate
[ ] Xóa if/else monster name trong MovePlayer
```

Seed quái hiện tại:

```txt
Dã Trĩ
Thạch Hầu
Độc Lang Vương
Dã Trĩ Vương
Mãng Xà Dung Nham
Thiết Giáp Hùng
Cổ Sư Lang Thang
```

### Phase 2 — Tách loot table

```txt
[ ] Tạo bảng loot_tables
[ ] Tạo LootTable model
[ ] Tạo RollLoot
[ ] Tạo ApplyLoot
[ ] CombatAttack gọi loot table khi victory
[ ] Xóa hardcode boar_meat/wild_boar_tusk
```

### Phase 3 — Boss có Cổ

```txt
[ ] Mở rộng CombatState thêm EnemyGuInventory
[ ] Mở rộng CombatState thêm EnemyMana/EnemyMaxMana
[ ] Tạo Combatant struct
[ ] Tạo CombatantFromPlayer
[ ] Tạo CombatantFromActiveCombat
[ ] Refactor CalculateGuEffectForCombatant
[ ] BuildEnemyActions từ EnemyGuInventory
[ ] Boss trừ mana khi dùng Cổ
[ ] Save EnemyMana trong active_combat
```

### Phase 4 — AI profile

```txt
[ ] Tạo SelectActionsByAI
[ ] Tạo boss_burst
[ ] Tạo defensive
[ ] Tạo boss_poison
[ ] Tạo boss_summoner
[ ] Test boss chọn Cổ hợp lý
```

### Phase 5 — Spawn DB

```txt
[ ] Tạo regions
[ ] Tạo monster_spawns
[ ] Tạo map_cells nếu cần
[ ] Tạo ResolveCell
[ ] GenerateCell chỉ còn fallback procedural
[ ] Boss room dùng DB fixed spawn
```

### Phase 6 — Admin/CMS sau này

```txt
[ ] CRUD monster_templates
[ ] CRUD loot_tables
[ ] CRUD monster_spawns
[ ] Upload image_url cho boss/quái
[ ] Admin chỉnh boss equip Cổ
```

---

## 19. Lỗi cần tránh

### Không lưu boss vào bảng players

Boss không nên là player giả.

Lý do:

```txt
Player có auth/password/session
Boss cần template/spawn/loot/AI
Dữ liệu sẽ rối
```

### Không hardcode boss theo name

Sai:

```go
if enemyName == "Độc Lang Vương" {
}
```

Đúng:

```go
monster.ID == "boss_wolf_king"
monster.AIProfile == "boss_balanced"
monster.GuInventory
```

### Không cho boss dùng Cổ miễn phí

Boss có Cổ phải có mana:

```txt
EnemyMana >= action.ManaCost
Không đủ mana -> fallback basic attack
```

### Không hardcode loot trong CombatAttack

Sai:

```go
player.MaterialsInventory["boar_meat"]++
```

Đúng:

```go
loot := RollLoot(lootTable)
ApplyLoot(player, loot)
```

---

## 20. Prompt cho AI refactor

```txt
Refactor the current Go Gin combat and exploration system.

Current problems:
- GenerateCell hardcodes monsters/resources by region.
- MovePlayer hardcodes monster stats by monster name.
- CombatAttack enemies only use EnemyDamage/EnemyDefense.
- Bosses cannot equip or use Gu.
- Loot is hardcoded after victory.

Goal:
Create a data-driven monster/boss system.

Implement:
1. regions table
2. monster_templates table
3. monster_spawns table
4. loot_tables table
5. MonsterTemplate Go model
6. LootTable Go model
7. LoadMonsterTemplate
8. RollLoot and ApplyLoot
9. BuildCombatStateFromMonster
10. Combatant struct
11. CalculateGuEffectForCombatant
12. BuildEnemyActions
13. SelectActionsByAI
14. ResolveCell

Rules:
- Player and Boss should both use BuildCombatAction.
- Boss can have GuInventory and equipped Gu.
- Boss Gu consumes EnemyMana.
- Boss action selection depends on AIProfile.
- Loot should come from loot_table_id.
- Keep GenerateCell only as procedural fallback.
- Remove hardcoded monster stat if/else from MovePlayer.
- Remove hardcoded loot from CombatAttack.
- Keep Vietnamese combat logs.
- Keep changes incremental and compatible with current player.ActiveCombat JSONB.
```

---

## 21. Final architecture

```txt
Region DB
-> Spawn DB
-> MonsterTemplate DB
-> GuInventory JSONB
-> ItemInventory JSONB
-> LootTable DB
-> CombatState snapshot
-> Combatant
-> BuildCombatAction
-> Combat resolver
-> RollLoot
-> Save player
```

Kết quả:

```txt
Boss có Cổ
Boss có item
Boss có loot riêng
Boss có AI riêng
Map có spawn riêng
Event/truyền thừa/NPC dễ mở rộng
```
