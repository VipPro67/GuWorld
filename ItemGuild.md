# ItemGuild.md — Plan & Guide xử lý hệ thống Item

## Mục tiêu

Thiết kế hệ thống item theo hướng data-driven.

Không xử lý item bằng `switch item_id`.

Thay vào đó dùng:

```go
item.Type
item.Effects
```

Ví dụ:

```json
{
  "id": "green_bamboo_wine",
  "name": "Rượu Thanh Trúc",
  "type": "Consumable",
  "effects": {
    "restore_mana": 20,
    "mortal_only": 1
  }
}
```

Item chỉ là data. Logic chạy theo effect registry.

---

# 1. Item Types

Nên chia item thành các nhóm sau:

```txt
Currency
Consumable
Food
Material
RefinementMaterial
SoulResource
ImmortalMaterial
QuestItem
DaoMarkResource
EquipmentMaterial
SpecialResource
```

## Currency

Dùng làm tiền tệ hoặc năng lượng giao dịch.

```txt
primeval_stone              Nguyên Thạch
primeval_stone_fragment     Nguyên Thạch Toái Phiến
immortal_essence_stone      Tiên Nguyên Thạch
```

Không dùng trực tiếp bằng `UseItem`, trừ khi có endpoint riêng như `RechargeWithStones`.

---

## Consumable

Dùng trực tiếp một lần, sau đó trừ số lượng.

```txt
green_bamboo_wine           Rượu Thanh Trúc
moon_orchid_petal           Cánh Hoa Nguyệt Lan
jade_stone                  Ngọc Thạch
nine_leaf_vitality_leaf     Sinh Cơ Diệp
guts                        Can Đảm
```

Chạy qua:

```go
UseItem()
```

---

## Food

Thức ăn cho Cổ trùng.

```txt
white_lotus_giant_silkworm  Bạch Liên Cự Tằm
sand_gull_soil              Đất Sa Âu
womans_heart                Trái Tim Phụ Nữ
heavy_gold_ore              Quặng Vàng Nặng
boar_meat                   Thịt Lợn Rừng
moon_orchid_petal           Cánh Hoa Nguyệt Lan
green_bamboo_wine           Rượu Thanh Trúc
```

Chạy qua:

```go
FeedGu()
```

Không nên cho dùng trực tiếp nếu `Type == "Food"`.

---

## Material

Vật liệu thường, dùng để craft, quest, trade.

```txt
iron_ore        Quặng Sắt
jade_stone      Ngọc Thạch
wolf_fang       Lang Nha
wolf_pelt       Da Sói
wild_boar_tusk  Nanh Lợn Rừng
python_scale    Mãng Lân
```

Chạy qua:

```go
CraftItem()
RefineGu()
QuestSubmit()
```

---

## RefinementMaterial

Nguyên liệu luyện Cổ.

```txt
earth_crystal       Địa Tinh
star_metal          Tinh Thần Thiết
cold_jade           Hàn Ngọc
blood_crystal       Huyết Tinh
soul_fragment       Hồn Phách Toái Phiến
```

Chạy qua:

```go
RefineGu()
CombineGu()
UpgradeGu()
```

---

## SoulResource

Tài nguyên hồn đạo.

```txt
guts                    Can Đảm
wolf_soul               Lang Hồn
bear_soul               Hùng Hồn
human_soul_fragment     Nhân Hồn Toái Phiến
desolate_beast_soul     Hoang Thú Hồn
```

Có thể dùng trực tiếp hoặc làm nguyên liệu.

---

## ImmortalMaterial

Tiên tài, dùng cho tiên cổ hoặc luyện cổ cao cấp.

```txt
white_lotus_giant_silkworm  Bạch Liên Cự Tằm
sand_gull_soil              Đất Sa Âu
dragon_fish                 Long Ngư
airsac_fish                 Khí Nang Ngư
cloud_soil                  Vân Thổ
reverse_flow_river_water    Nghịch Lưu Hà Thủy
dang_hun_mountain_rock      Đãng Hồn Sơn Thạch
luo_po_valley_sand          Lạc Phách Cốc Sa
```

---

## QuestItem

Item chỉ dùng cho nhiệm vụ hoặc event.

```txt
clan_token          Lệnh Bài Gia Tộc
inheritance_key     Chìa Khóa Truyền Thừa
blood_contract      Huyết Khế
map_fragment        Tàn Đồ
```

---

## DaoMarkResource

Tài nguyên tăng đạo ngân.

```txt
strength_dao_mark       Lực Đạo Đạo Ngân
blood_dao_mark          Huyết Đạo Đạo Ngân
soul_dao_mark           Hồn Đạo Đạo Ngân
wisdom_dao_mark         Trí Đạo Đạo Ngân
luck_dao_mark           Vận Đạo Đạo Ngân
time_dao_mark           Thời Gian Đạo Ngân
space_dao_mark          Không Gian Đạo Ngân
```

---

# 2. Data model đề xuất

```go
type ItemSpec struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Type        string             `json:"type"`
	Rarity      int                `json:"rarity"`
	Rank        int                `json:"rank"`
	Path        string             `json:"path"`
	Effects     map[string]float64 `json:"effects"`
	Tags        []string           `json:"tags"`
	Description string             `json:"description"`
}
```

Ví dụ registry:

```go
var ItemRegistry = map[string]ItemSpec{
	"green_bamboo_wine": {
		ID:   "green_bamboo_wine",
		Name: "Rượu Thanh Trúc",
		Type: "Consumable",
		Rank: 1,
		Path: "Wood Path",
		Effects: map[string]float64{
			"restore_mana": 20,
			"mortal_only":  1,
		},
		Description: "Rượu trúc xanh dùng để khôi phục Chân Nguyên.",
	},
}
```

---

# 3. Inventory model

Hiện có thể giữ:

```go
type MaterialsInventory map[string]int
```

Tốt hơn về sau:

```go
type ItemStack struct {
	ItemID   string `json:"item_id"`
	Qty      int    `json:"qty"`
	Bound    bool   `json:"bound"`
	ExpireAt string `json:"expire_at,omitempty"`
}

type ItemInventory []ItemStack
```

Giai đoạn đầu cứ dùng `map[string]int` cho đơn giản.

---

# 4. Item Effects

## Effect nhóm hồi phục

```txt
restore_hp
restore_mana
restore_max_hp
restore_percent_hp
restore_percent_mana
```

## Effect nhóm giới hạn người dùng

```txt
mortal_only
immortal_only
min_rank
max_rank
path_required
```

## Effect nhóm tu luyện

```txt
add_wall_progress
increase_aptitude
soul_boost
dao_mark_gain
rank_progress
sub_rank_progress
```

## Effect nhóm trạng thái

```txt
remove_negative_status
remove_poison
remove_bleed
remove_mana_pollution
apply_buff
apply_debuff
```

## Effect nhóm nguyên liệu

```txt
food_value
refinement_value
craft_value
trade_value
```

---

# 5. Item Effect Registry

```go
type ItemEffectHandler func(player *models.Player, item ItemSpec) (string, error)

var ItemEffectRegistry = map[string]ItemEffectHandler{
	"restore_hp":             restoreHPEffect,
	"restore_mana":           restoreManaEffect,
	"restore_percent_hp":     restorePercentHPEffect,
	"restore_percent_mana":   restorePercentManaEffect,
	"mortal_only":            mortalOnlyEffect,
	"immortal_only":          immortalOnlyEffect,
	"soul_boost":             soulBoostEffect,
	"dao_mark_gain":          daoMarkGainEffect,
	"remove_negative_status": removeNegativeStatusEffect,
}
```

---

# 6. Handler mẫu

```go
func restoreManaEffect(player *models.Player, item ItemSpec) (string, error) {
	value := item.Effects["restore_mana"]

	player.CurrentMana += value
	if player.CurrentMana > player.MaxMana {
		player.CurrentMana = player.MaxMana
	}

	return fmt.Sprintf("%s phục hồi %.1f Chân Nguyên.", item.Name, value), nil
}

func restoreHPEffect(player *models.Player, item ItemSpec) (string, error) {
	value := item.Effects["restore_hp"]

	player.HP += value
	if player.HP > player.BaseHP {
		player.HP = player.BaseHP
	}

	return fmt.Sprintf("%s hồi phục %.1f HP.", item.Name, value), nil
}

func mortalOnlyEffect(player *models.Player, item ItemSpec) (string, error) {
	if isImmortalCultivator(player) {
		return "", fmt.Errorf("%s chỉ có hiệu quả với Cổ sư phàm cấp", item.Name)
	}

	return "", nil
}

func immortalOnlyEffect(player *models.Player, item ItemSpec) (string, error) {
	if !isImmortalCultivator(player) {
		return "", fmt.Errorf("%s chỉ dành cho Cổ Tiên", item.Name)
	}

	return "", nil
}
```

---

# 7. UseItem flow

```txt
1. Bind request
2. Get auth player
3. Begin transaction
4. evaluateManaTx
5. Check item quantity
6. Get ItemSpec
7. Validate item.Type == Consumable hoặc SoulResource có usable flag
8. Run item effect handlers
9. Trừ quantity
10. Save DB
11. Commit
12. Return player + messages
```

---

# 8. UseItem skeleton

```go
func UseItem(c *gin.Context) {
	var req UseItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	qty := player.MaterialsInventory[req.ItemID]
	if qty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm này trong túi đồ"})
		return
	}

	item, ok := ItemRegistry[req.ItemID]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong ItemRegistry"})
		return
	}

	if item.Type != "Consumable" && item.Type != "SoulResource" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm này không thể sử dụng trực tiếp"})
		return
	}

	messages := []string{}

	for effectKey := range item.Effects {
		handler, ok := ItemEffectRegistry[effectKey]
		if !ok {
			continue
		}

		msg, err := handler(&player, item)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if msg != "" {
			messages = append(messages, msg)
		}
	}

	player.MaterialsInventory[req.ItemID]--
	if player.MaterialsInventory[req.ItemID] <= 0 {
		delete(player.MaterialsInventory, req.ItemID)
	}

	_, err = tx.Exec(`
		UPDATE players 
		SET materials_inventory = $1, current_mana = $2, hp = $3, max_mana = $4
		WHERE id = $5`,
		player.MaterialsInventory,
		player.CurrentMana,
		player.HP,
		player.MaxMana,
		player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":   player,
		"messages": messages,
	})
}
```

---

# 9. FeedGu flow

Food item không chạy qua `UseItem`.

Flow:

```txt
1. Chọn Gu instance
2. Load Gu spec
3. Đọc spec.FoodTypes
4. Tìm item trong inventory
5. Check item.Type == Food hoặc item.Tags contains "food"
6. Trừ item
7. Set hunger = MaxHunger
8. Set status = ACTIVE
9. Save DB
```

Rule:

```txt
Food item chỉ dùng để nuôi Gu.
Consumable item mới dùng trực tiếp.
Một item có thể vừa là Food vừa là Consumable nếu có tag phù hợp.
```

Ví dụ `moon_orchid_petal` có thể vừa là thức ăn vừa là tiêu hao:

```go
Type: "Consumable",
Tags: []string{"food", "moon_path_food"},
Effects: map[string]float64{
	"restore_mana": 10,
	"mortal_only":  1,
}
```

---

# 10. Craft / Refine flow

## Craft

```txt
Input: recipe_id
Check materials
Consume materials
Add output item
Save DB
```

## Refine Gu

```txt
Input: recipe_id
Check Gu material
Check item materials
Check primeval stones
Roll success rate
On success: add new Gu
On fail: consume partial/full materials
Save DB
```

Recipe model:

```go
type RecipeSpec struct {
	ID          string         `json:"id"`
	Type        string         `json:"type"`
	Inputs      map[string]int `json:"inputs"`
	OutputItem  string         `json:"output_item"`
	OutputGu    string         `json:"output_gu"`
	SuccessRate float64        `json:"success_rate"`
	StoneCost   int            `json:"stone_cost"`
}
```

---

# 11. Item data mẫu

```go
var ItemRegistry = map[string]ItemSpec{
	"primeval_stone": {
		ID:   "primeval_stone",
		Name: "Nguyên Thạch",
		Type: "Currency",
		Rank: 1,
		Description: "Tinh thể chứa nguyên khí thiên nhiên, dùng để tu luyện và giao dịch.",
	},
	"green_bamboo_wine": {
		ID:   "green_bamboo_wine",
		Name: "Rượu Thanh Trúc",
		Type: "Consumable",
		Rank: 1,
		Effects: map[string]float64{
			"restore_mana": 20,
			"mortal_only":  1,
		},
		Tags: []string{"food", "wine", "liquor_worm_food"},
		Description: "Rượu trúc xanh, có thể khôi phục Chân Nguyên hoặc nuôi Tửu Trùng.",
	},
	"moon_orchid_petal": {
		ID:   "moon_orchid_petal",
		Name: "Cánh Hoa Nguyệt Lan",
		Type: "Consumable",
		Rank: 1,
		Effects: map[string]float64{
			"restore_mana": 10,
			"mortal_only":  1,
		},
		Tags: []string{"food", "moon_path_food"},
		Description: "Cánh hoa chứa nguyệt hoa, dùng nuôi Nguyệt đạo Cổ hoặc luyện hóa Chân Nguyên.",
	},
	"jade_stone": {
		ID:   "jade_stone",
		Name: "Ngọc Thạch",
		Type: "Consumable",
		Rank: 1,
		Effects: map[string]float64{
			"restore_hp": 20,
		},
		Tags: []string{"material", "earth_path"},
		Description: "Đá ngọc có linh khí, dùng dưỡng thương hoặc luyện Cổ phòng ngự.",
	},
	"white_lotus_giant_silkworm": {
		ID:   "white_lotus_giant_silkworm",
		Name: "Bạch Liên Cự Tằm",
		Type: "Food",
		Rank: 6,
		Tags: []string{"immortal_gu_food", "soul_path_food"},
		Description: "Thức ăn chuyên dụng để nuôi tiên cổ Tịnh Hồn.",
	},
	"sand_gull_soil": {
		ID:   "sand_gull_soil",
		Name: "Đất Sa Âu",
		Type: "Food",
		Rank: 6,
		Tags: []string{"immortal_gu_food", "luck_path_food"},
		Description: "Đất cát đặc biệt dùng nuôi tiên cổ Liên Vận.",
	},
	"womans_heart": {
		ID:   "womans_heart",
		Name: "Trái Tim Phụ Nữ",
		Type: "Food",
		Rank: 6,
		Tags: []string{"immortal_gu_food", "poison_path_food"},
		Description: "Thức ăn của tiên cổ Độc đạo Phụ Nhân Tâm.",
	},
	"heavy_gold_ore": {
		ID:   "heavy_gold_ore",
		Name: "Quặng Vàng Nặng",
		Type: "Food",
		Rank: 6,
		Tags: []string{"immortal_gu_food", "strength_path_food"},
		Description: "Quặng nặng chứa kim khí, là thức ăn của tiên cổ Khí Lực.",
	},
}
```

---

# 12. Rule xử lý nhiều effect

Một item có thể có nhiều effect.

Ví dụ:

```json
{
  "restore_hp": 30,
  "remove_poison": 1,
  "apply_buff": 1
}
```

Không nên apply lung tung. Nên chia phase:

```txt
Validate phase:
- mortal_only
- immortal_only
- min_rank
- max_rank

Apply phase:
- restore_hp
- restore_mana
- soul_boost
- dao_mark_gain

Cleanse phase:
- remove_poison
- remove_bleed
- remove_negative_status

After phase:
- apply_buff
- add_log
```

Đơn giản nhất: chạy validate trước, apply sau.

```go
var ValidateItemEffects = map[string]ItemEffectHandler{
	"mortal_only":   mortalOnlyEffect,
	"immortal_only": immortalOnlyEffect,
	"min_rank":      minRankEffect,
	"max_rank":      maxRankEffect,
}

var ApplyItemEffects = map[string]ItemEffectHandler{
	"restore_hp":             restoreHPEffect,
	"restore_mana":           restoreManaEffect,
	"soul_boost":             soulBoostEffect,
	"dao_mark_gain":          daoMarkGainEffect,
	"remove_negative_status": removeNegativeStatusEffect,
}
```

---

# 13. DB fields cần lưu

Tùy effect, cần update các field:

```txt
materials_inventory
current_mana
max_mana
hp
base_hp
aptitude_percent
aptitude_grade
dao_marks
body_mods
active_combat
```

Nếu có status effect:

```txt
status_effects jsonb
```

---

# 14. API đề xuất

```txt
POST /api/items/use
POST /api/items/feed-gu
POST /api/items/craft
POST /api/items/refine-gu
GET  /api/items/registry
GET  /api/items/inventory
```

Request:

```json
{
  "item_id": "green_bamboo_wine",
  "quantity": 1
}
```

Response:

```json
{
  "player": {},
  "messages": [
    "Rượu Thanh Trúc phục hồi 20 Chân Nguyên."
  ]
}
```

---

# 15. Plan triển khai

## Phase 1: ItemRegistry

```txt
[ ] Tạo ItemSpec
[ ] Tạo ItemRegistry
[ ] Convert item hardcode hiện tại sang ItemRegistry
[ ] Thêm Type, Effects, Tags
```

## Phase 2: UseItem registry

```txt
[ ] Tạo ItemEffectHandler
[ ] Tạo ItemEffectRegistry
[ ] Refactor UseItem bỏ switch
[ ] Test restore_mana
[ ] Test restore_hp
[ ] Test immortal không dùng mortal item
```

## Phase 3: Food / FeedGu

```txt
[ ] Validate item tồn tại trong ItemRegistry
[ ] Check item.Type == Food hoặc tag food
[ ] FeedGu dùng item data thay vì chỉ string
[ ] Test thức ăn thường
[ ] Test tiên cổ food
```

## Phase 4: Craft / Refine

```txt
[ ] Tạo RecipeSpec
[ ] Tạo RecipeRegistry
[ ] Tạo CraftItem
[ ] Tạo RefineGu
[ ] Consume materials
[ ] Roll success rate
[ ] Add output item/Gu
```

## Phase 5: Advanced resources

```txt
[ ] SoulResource
[ ] ImmortalMaterial
[ ] DaoMarkResource
[ ] QuestItem
[ ] Status effects
[ ] Buff duration
```

---

# 16. Prompt dùng cho AI refactor

```txt
Refactor item system in Go Gin backend.

Goal:
Remove switch item_id logic from UseItem.
Handle items by ItemRegistry, item.Type, and item.Effects.

Current inventory:
player.MaterialsInventory map[string]int

Create:
- ItemSpec struct
- ItemRegistry
- ItemEffectHandler
- ItemEffectRegistry
- ValidateItemEffects
- ApplyItemEffects

Rules:
1. Consumable items can be used by UseItem.
2. Food items are used by FeedGu, not UseItem.
3. Material items are used by CraftItem or RefineGu.
4. Currency items are handled by specific endpoints.
5. One item can have many effects.
6. Validate effects must run before apply effects.
7. Unknown effects should be ignored safely or logged.
8. After using item successfully, decrease quantity by 1.
9. Delete item key if quantity <= 0.
10. Save changed player fields to DB.

Implement handlers:
- restore_mana
- restore_hp
- mortal_only
- immortal_only
- soul_boost
- dao_mark_gain
- remove_negative_status

Keep Vietnamese response messages.
Keep code simple.
Do not use huge switch/case by item ID.
```

---

# 17. Final rule

```txt
ItemID = identity
Type = item category
Tags = flexible usage
Effects = actual behavior
Registry = logic
Endpoint = use context
DB update = persistence
```

Thiết kế này giúp thêm item mới bằng data, không cần sửa logic core.
