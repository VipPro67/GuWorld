# Guild.md — Guide/Prompt xử lý Gu theo Type và Effects

## Mục tiêu

Refactor logic Gu để không xử lý theo `GuID` cố định nữa.

Thay vì:

```go
switch gu.GuID {
case "GU_WHITE_BOAR":
case "GU_MOON_HALO":
case "GU_RELIC":
}
```

hãy xử lý theo:

```go
spec.Type
spec.Effects
```

Ví dụ:

```json
{
  "type": "Active",
  "effects": {
    "damage": 50,
    "bleed": 10,
    "lifesteal": 0.3
  }
}
```

Cổ có nhiều effect thì gom tất cả effect vào một context/action, sau đó resolve theo đúng phase.

---

## Quy tắc tổng quát

Mỗi Gu có:

```go
Type    string
Effects map[string]float64
Cost    float64
Rank    int
Path    string
```

Không viết logic theo tên cổ trùng, trừ vài case cực đặc biệt bắt buộc.

Luồng chuẩn:

```txt
1. Load player
2. Load Gu instance
3. Load Gu spec từ GuRegistry
4. Validate status/rank/mana/type
5. Tạo context/action
6. Chạy effect handlers theo spec.Effects
7. Resolve kết quả
8. Save DB
```

---

# 1. Passive Gu

## Ý nghĩa

Passive Gu là cổ trùng có hiệu ứng thường trực.

Ví dụ:

```txt
GU_WHITE_BOAR        add_base_attack
GU_BLACK_BOAR        add_base_attack
GU_JADE_SKIN         add_base_defense
GU_IRON_BONE         add_base_defense
GU_BITTER_STRENGTH   damage_multiplier_on_hit
GU_WOLF_SOUL         soul_refinement
```

## Khi nào chạy?

Passive chạy khi cần tính stats:

```go
CalculatePlayerStats(player)
```

Điều kiện:

```txt
gu.Equipped == true
gu.Status == "ACTIVE"
spec.Type == "Passive"
```

## Struct đề xuất

```go
type StatEffectHandler func(player *models.Player, spec models.GuDictionaryItem, stats map[string]float64)

var StatEffectRegistry = map[string]StatEffectHandler{
	"add_base_attack":          addAttackStat,
	"add_base_defense":         addDefenseStat,
	"damage_multiplier_on_hit": addDamageOnHitStat,
	"soul_refinement":          addSoulStat,
}
```

## Handler mẫu

```go
func addAttackStat(player *models.Player, spec models.GuDictionaryItem, stats map[string]float64) {
	stats["attack"] += spec.Effects["add_base_attack"] * getPathMasteryMultiplier(player, spec.Path)
}

func addDefenseStat(player *models.Player, spec models.GuDictionaryItem, stats map[string]float64) {
	stats["defense"] += spec.Effects["add_base_defense"] * getPathMasteryMultiplier(player, spec.Path)
}

func addDamageOnHitStat(player *models.Player, spec models.GuDictionaryItem, stats map[string]float64) {
	if player.BaseHP <= 0 {
		return
	}

	missingHpRate := 1 - player.HP/player.BaseHP
	stats["damage_multiplier"] += missingHpRate * spec.Effects["damage_multiplier_on_hit"]
}

func addSoulStat(player *models.Player, spec models.GuDictionaryItem, stats map[string]float64) {
	stats["soul"] += spec.Effects["soul_refinement"]
}
```

## Rule quan trọng

Passive không trừ mana.

Passive không tạo action combat trực tiếp.

Passive chỉ cộng stat hoặc modifier.

---

# 2. Storage Gu

## Ý nghĩa

Storage Gu tăng sức chứa túi đồ.

Ví dụ:

```txt
GU_TUSITA_FLOWER storage: 15000
```

## Khi nào chạy?

Storage nên chạy trong:

```go
CalculatePlayerStats(player)
HasInventorySpace(player, incomingCount)
```

Điều kiện:

```txt
gu.Equipped == true
gu.Status == "ACTIVE"
spec.Type == "Storage"
```

## Handler mẫu

```go
var StatEffectRegistry = map[string]StatEffectHandler{
	"storage": addStorageStat,
}

func addStorageStat(player *models.Player, spec models.GuDictionaryItem, stats map[string]float64) {
	stats["storage"] += spec.Effects["storage"]
}
```

## Rule quan trọng

Storage không nên là combat action.

Storage không trừ mana.

Storage chỉ ảnh hưởng giới hạn chứa item/Gu/material.

---

# 3. Active Gu

## Ý nghĩa

Active Gu là cổ dùng trong combat hoặc hành động chủ động.

Ví dụ:

```txt
GU_MOON_HALO       damage
GU_MOON_SILVER     damage + range
GU_MOON_WHIRL      damage + curve_attack
GU_WATER_SHIELD    add_base_defense
GU_CANOPY          add_base_defense
GU_WOOD_CHARM      mana_regen + wood_transformation
GU_BLOOD_ASSET     replicate + blood_drain
```

## Khi nào chạy?

Active chạy trong combat turn:

```go
CombatAttack()
BuildCombatAction()
```

Điều kiện:

```txt
gu.Equipped == true
gu.Status == "ACTIVE"
spec.Type == "Active"
player.CurrentMana >= cost
```

## Action đề xuất

```go
type Action struct {
	Name           string
	Dmg            float64
	Def            float64
	ManaCost       float64
	IsUtility      bool
	LifeStealRate  float64
	StatusEffects  []StatusEffect
	Logs           []string
}

type StatusEffect struct {
	Type  string
	Value float64
	Turns int
}
```

## Registry đề xuất

```go
type CombatEffectHandler func(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error

var ActiveEffectRegistry = map[string]CombatEffectHandler{
	"damage":              handleDamageEffect,
	"add_base_attack":     handleDamageEffect,
	"add_base_defense":    handleDefenseEffect,
	"range":               handleRangeEffect,
	"curve_attack":        handleCurveAttackEffect,
	"mana_regen":          handleManaRegenEffect,
	"wood_transformation": handleWoodTransformationEffect,
	"blood_drain":         handleBloodDrainEffect,
	"replicate":           handleReplicateEffect,
	"bleed":               handleBleedEffect,
	"lifesteal":           handleLifeStealEffect,
}
```

## Build action

```go
func BuildCombatAction(ctx *CombatContext, spec models.GuDictionaryItem) (Action, error) {
	_, _, cost, err := CalculateGuEffect(ctx.Attacker, &spec)
	if err != nil {
		return Action{}, err
	}

	action := Action{
		Name:     spec.Name,
		ManaCost: cost,
	}

	for effectKey := range spec.Effects {
		handler, ok := ActiveEffectRegistry[effectKey]
		if !ok {
			continue
		}

		if err := handler(ctx, &spec, &action); err != nil {
			return Action{}, err
		}
	}

	return action, nil
}
```

## Handler mẫu

```go
func handleDamageEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	dmg, _, _, err := CalculateGuEffect(ctx.Attacker, spec)
	if err != nil {
		return err
	}

	action.Dmg += dmg
	return nil
}

func handleDefenseEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	_, def, _, err := CalculateGuEffect(ctx.Attacker, spec)
	if err != nil {
		return err
	}

	action.Def += def
	return nil
}

func handleBleedEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.StatusEffects = append(action.StatusEffects, StatusEffect{
		Type:  "bleed",
		Value: spec.Effects["bleed"],
		Turns: 3,
	})
	return nil
}

func handleLifeStealEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.LifeStealRate += spec.Effects["lifesteal"]
	return nil
}

func handleManaRegenEffect(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	value := spec.Effects["mana_regen"]

	ctx.Attacker.CurrentMana += value
	if ctx.Attacker.CurrentMana > ctx.Attacker.MaxMana {
		ctx.Attacker.CurrentMana = ctx.Attacker.MaxMana
	}

	action.IsUtility = true
	action.Logs = append(action.Logs, fmt.Sprintf("%s hồi phục %.1f Chân Nguyên.", spec.Name, value))
	return nil
}
```

## Rule nhiều effect

Không apply ngay từng effect nếu effect cần damage thật.

Sai:

```txt
damage handler gây damage ngay
lifesteal handler hồi máu ngay
bleed handler trừ máu ngay
```

Đúng:

```txt
1. damage cộng vào action.Dmg
2. lifesteal lưu vào action.LifeStealRate
3. bleed lưu vào action.StatusEffects
4. Combat resolver tính net damage
5. Sau khi có realDamage mới hồi máu / gắn bleed
```

---

# 4. Support Gu

## Ý nghĩa

Support Gu là cổ hỗ trợ, buff, cleanse, utility hoặc tác động ngoài damage trực tiếp.

Ví dụ:

```txt
GU_CLEANSING_WATER       cleansing_power
GU_BLOOD_SKULL           increase_aptitude
GU_BONE_FLESH_REUNION    share_cultivation
GU_ALL_OUT_EFFORT        summon_phantom
```

## Khi nào chạy?

Có 2 loại Support:

```txt
Combat Support: dùng trong combat turn
Out-of-combat Support: dùng qua endpoint riêng
```

### Combat Support

Ví dụ:

```txt
GU_ALL_OUT_EFFORT summon_phantom
GU_CLEANSING_WATER cleansing_power
```

Có thể dùng chung `CombatEffectHandler`.

```go
var SupportEffectRegistry = map[string]CombatEffectHandler{
	"cleansing_power": handleCleansingPower,
	"summon_phantom": handleSummonPhantom,
}
```

### Out-of-combat Support

Ví dụ:

```txt
GU_BLOOD_SKULL increase_aptitude
GU_BONE_FLESH_REUNION share_cultivation
```

Nên có endpoint riêng:

```go
UseSupportGu(c *gin.Context)
```

## Handler combat support

```go
func handleCleansingPower(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	action.IsUtility = true
	action.Logs = append(action.Logs, fmt.Sprintf("%s thanh tẩy dị lực và trạng thái xấu.", spec.Name))
	return nil
}

func handleSummonPhantom(ctx *CombatContext, spec *models.GuDictionaryItem, action *Action) error {
	stats := CalculatePlayerStats(ctx.Attacker)
	action.Dmg += stats["attack"]
	action.Logs = append(action.Logs, fmt.Sprintf("%s triệu hồi thú lực hư ảnh, gây %.1f sát thương.", spec.Name, stats["attack"]))
	return nil
}
```

## Handler out-of-combat support

```go
type SupportEffectHandler func(player *models.Player, spec models.GuDictionaryItem) (string, error)

var OutCombatSupportRegistry = map[string]SupportEffectHandler{
	"increase_aptitude": increaseAptitude,
	"share_cultivation": shareCultivation,
}
```

```go
func increaseAptitude(player *models.Player, spec models.GuDictionaryItem) (string, error) {
	value := int(spec.Effects["increase_aptitude"])

	player.AptitudePercent += value
	if player.AptitudePercent > 99 {
		player.AptitudePercent = 99
	}

	player.MaxMana = float64(player.AptitudePercent)
	if player.CurrentMana > player.MaxMana {
		player.CurrentMana = player.MaxMana
	}

	return fmt.Sprintf("%s tăng tư chất lên %d%%.", spec.Name, player.AptitudePercent), nil
}

func shareCultivation(player *models.Player, spec models.GuDictionaryItem) (string, error) {
	return fmt.Sprintf("%s cần target player để chia sẻ tu vi.", spec.Name), nil
}
```

## Rule quan trọng

Support có thể trừ mana nếu `Cost > 0`.

Support không nhất thiết gây damage.

Support nên tách combat và non-combat để tránh logic rối.

---

# 5. Consumable Gu

## Ý nghĩa

Consumable Gu dùng một lần rồi biến mất khỏi inventory.

Ví dụ:

```txt
GU_BRONZE_RELIC          rank_progress
GU_RED_STEEL_RELIC       rank_progress
GU_WHITE_SILVER_RELIC    rank_progress
GU_YELLOW_GOLD_RELIC     rank_progress
GU_PURPLE_CRYSTAL_RELIC  rank_progress
GU_STONE_APERTURE        aperture_petrification
GU_GUTS                  soul_boost
```

## Khi nào chạy?

Consumable chạy qua endpoint:

```go
UseConsumableGu()
```

Điều kiện:

```txt
spec.Type == "Consumable"
gu.Status == "ACTIVE"
```

Sau khi chạy thành công:

```txt
remove Gu khỏi player.GuInventory
rebuild player.EquippedGus
save DB
```

## Registry đề xuất

```go
type ConsumableEffectHandler func(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error)

var ConsumableEffectRegistry = map[string]ConsumableEffectHandler{
	"rank_progress":          handleRankProgress,
	"aperture_petrification": handleAperturePetrification,
	"soul_boost":             handleSoulBoost,
	"force_rank_up":          handleForceRankUp,
	"mana_pollution":         handleManaPollution,
}
```

## Handler mẫu

```go
func handleRankProgress(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error) {
	if spec.Rank != player.Rank {
		return "", fmt.Errorf("%s chỉ dùng cho %s", spec.Name, getRankName(spec.Rank))
	}

	if player.SubRank >= 4 {
		return "", fmt.Errorf("Xá Lợi Cổ không thể đột phá đại cảnh giới")
	}

	player.SubRank++

	stageNames := map[int]string{
		1: "Sơ Giai",
		2: "Trung Giai",
		3: "Cao Giai",
		4: "Đỉnh Phong",
	}

	return fmt.Sprintf("Đột phá! Đạo hữu luyện hóa %s, tu vi nâng cao lên %s %s!", spec.Name, getRankName(player.Rank), stageNames[player.SubRank]), nil
}

func handleAperturePetrification(player *models.Player, gu models.GuInstance, spec models.GuDictionaryItem) (string, error) {
	player.SubRank = 4
	player.MaxMana -= 15

	if player.MaxMana < 20 {
		player.MaxMana = 20
	}

	if player.CurrentMana > player.MaxMana {
		player.CurrentMana = player.MaxMana
	}

	return fmt.Sprintf("%s đẩy tu vi lên Đỉnh Phong nhưng làm Không khiếu hóa đá.", spec.Name), nil
}
```

## Rule nhiều effect

Consumable có thể chạy nhiều effect tuần tự.

Ví dụ:

```json
{
  "force_rank_up": 1,
  "mana_pollution": 1
}
```

Flow:

```txt
1. force_rank_up thay đổi rank/subrank
2. mana_pollution gắn debuff hoặc giảm regen
3. remove Gu
4. save DB
```

---

# 6. Effect phase

Mỗi effect nên có phase rõ ràng.

```txt
Stat phase:
- add_base_attack
- add_base_defense
- storage
- soul_refinement

Build action phase:
- damage
- range
- curve_attack
- add_base_defense
- summon_phantom
- cleansing_power

After damage phase:
- lifesteal
- bleed
- poison
- stun

Out-of-combat phase:
- rank_progress
- increase_aptitude
- share_cultivation
- soul_boost
```

Không nên để tất cả effect chạy cùng một phase.

---

# 7. Validate chung

## Check status

```go
if gu.Status == "DEAD" {
	return error
}
```

## Check rank

```go
if player.Rank < spec.Rank {
	applyBacklash()
	return error
}
```

## Check mana

```go
cost := calculateEnergyCost(player, spec)
if player.CurrentMana < cost {
	return insufficientEnergyMessage(player)
}
```

## Check equipped

```go
if actionRequiresEquipped && !gu.Equipped {
	return error
}
```

---

# 8. Save DB

Sau khi thay đổi player:

```go
UPDATE players SET ...
```

Các field thường cần save:

```txt
hp
current_mana
max_mana
primeval_stones
immortal_stones
gu_inventory
materials_inventory
equipped_gus
body_mods
active_combat
```

Nếu dùng status effect lâu dài, nên thêm field:

```txt
status_effects jsonb
```

---

# 9. Prompt dùng cho AI refactor code

Dùng prompt này khi yêu cầu AI sửa code:

```txt
Refactor Gu logic in Go Gin backend.

Goal:
Do not hardcode Gu behavior by GuID.
Handle Gu behavior by spec.Type and spec.Effects.

Current Gu types:
- Passive
- Storage
- Active
- Support
- Consumable

Rules:
1. Passive and Storage are calculated inside CalculatePlayerStats.
2. Active and combat Support are converted into Action by BuildCombatAction.
3. Consumable is handled in UseConsumableGu and removed after successful use.
4. Out-of-combat Support should use UseSupportGu.
5. One Gu can have multiple effects.
6. Each effect must be handled by registry maps, not switch GuID.
7. Effects that depend on real damage, such as lifesteal or bleed, must be resolved after net damage is calculated.
8. Effects that only modify stats should only update stats map.
9. Effects that change player state must save related DB fields.
10. Unknown effects should be logged or ignored safely, not crash the game.

Create these registries:
- StatEffectRegistry
- ActiveEffectRegistry
- SupportEffectRegistry
- ConsumableEffectRegistry

Create helper functions:
- getPathMasteryMultiplier
- BuildCombatAction
- ApplyPostDamageEffects
- RebuildEquippedGus
- RemoveGuFromInventory
- ValidateGuUsable

Keep existing lore messages in Vietnamese.
Keep code simple and avoid huge switch/case blocks.
```

---

# 10. Checklist refactor

```txt
[ ] Tạo StatEffectRegistry
[ ] Tạo ActiveEffectRegistry
[ ] Tạo SupportEffectRegistry
[ ] Tạo ConsumableEffectRegistry
[ ] Refactor CalculatePlayerStats
[ ] Refactor CombatAttack build action
[ ] Refactor UseConsumableGu
[ ] Thêm UseSupportGu nếu cần
[ ] Thêm Action.StatusEffects
[ ] Thêm Action.LifeStealRate
[ ] Tách ApplyPostDamageEffects
[ ] Bỏ dần AbilityRegistry theo GuID
[ ] Test cổ có nhiều effects
[ ] Test Passive cộng stats
[ ] Test Storage tăng túi đồ
[ ] Test Consumable dùng xong biến mất
[ ] Test Support combat và non-combat
```

---

# 11. Mapping effect hiện tại

## Passive / Storage

```txt
add_base_attack              -> stats["attack"]
add_base_defense             -> stats["defense"]
damage_multiplier_on_hit     -> stats["damage_multiplier"]
soul_refinement              -> stats["soul"]
storage                      -> stats["storage"]
```

## Active

```txt
damage                       -> action.Dmg
add_base_attack              -> action.Dmg
add_base_defense             -> action.Def
range                        -> action metadata/log
curve_attack                 -> action metadata/log
mana_regen                   -> current_mana
wood_transformation          -> action.Def / utility
blood_drain                  -> action.Dmg or lifesteal style
replicate                    -> generate Gu/counter/log
generate_gu                  -> generate item/Gu after turn
```

## Support

```txt
cleansing_power              -> remove negative effects
increase_aptitude            -> aptitude_percent / max_mana
share_cultivation            -> needs target player
summon_phantom               -> action.Dmg based on stats["attack"]
```

## Consumable

```txt
rank_progress                -> sub_rank + 1
aperture_petrification       -> sub_rank = 4, max_mana penalty
soul_boost                   -> stats/body/soul value
force_rank_up                -> rank + 1
mana_pollution               -> negative status
```

---

# 12. Final design

```txt
GuID = identity only
Type = when/how the Gu is used
Effects = actual behavior
Registry = effect logic
Context/Action = temporary result
Resolver = apply final result
DB update = persist player state
```

Đây là hướng dễ mở rộng nhất khi thêm hàng trăm Cổ trùng mới.
