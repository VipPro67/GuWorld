# Kế hoạch Refactor Player (GuWorld)

## Mục tiêu

-   Giảm kích thước `Player`.
-   Tách dữ liệu theo domain.
-   Dễ mở rộng lên Rank 6--9.
-   Không phá vỡ API hiện tại.

------------------------------------------------------------------------

# Phase 1 -- Tách dữ liệu lớn (High Priority)

## 1. Recipe Progress

Tách `UnlockedRecipes` sang bảng:

``` sql
player_recipe_progress
- player_id
- recipe_id
- completion_rate
- is_unlocked
- deduce_count
- last_deduce_at
- favorite
- created_at
- updated_at
```

Lợi ích: - Player nhỏ hơn. - Query recipe nhanh. - Dễ thống kê.

------------------------------------------------------------------------

## 2. Map Progress

Tách:

-   ExploredCells
-   ClearedCells

sang

``` sql
player_map_progress
- player_id
- region_id
- cell_x
- cell_y
- explored
- cleared
- updated_at
```

------------------------------------------------------------------------

## 3. Combat Session

Không lưu ActiveCombat trong Player.

Đưa sang:

``` sql
combat_sessions
```

hoặc Redis.

------------------------------------------------------------------------

# Phase 2 -- Refactor Object

## Player

``` text
Player
├── Account
├── Cultivation
├── Resources
├── Body
├── Soul
├── Dao
├── World
├── Social
└── CombatConfig
```

## CultivationState

-   Rank
-   SubRank
-   Aptitude
-   WallProgress
-   HumanQi
-   Lifespan
-   ApertureStress
-   LastCultivateTime
-   TimeSpeedMultiplier

## Aptitude

-   Type
-   Grade
-   Percent

## ResourceState

-   CurrentMana
-   MaxMana
-   PrimevalStones
-   ImmortalStones

## SoulState

-   SoulHP
-   MaxSoul
-   SoulTier
-   NeedsSoulRefine

## BodyState

-   HP
-   BaseHP
-   BodyMods
-   DaoMarkDamage
-   ApertureDamage

## WorldState

-   X
-   Y
-   CurrentRegion

## SocialState

-   PartyID
-   ArenaRank
-   Bio
-   AvatarURL
-   RefCode

## CombatConfig

-   EquippedGus
-   CombatSettings

------------------------------------------------------------------------

# Phase 3 -- Lore Refactor

## Loại bỏ Wisdom stat Đã làm

Thay bằng hệ Thought. cần làm

``` text
ThoughtPool
- CurrentThought
- MaxThought
- RegenRate
```

Deduce tiêu hao Thought thay vì Mana.

## Dao

Chuẩn hóa:

``` text
PlayerDaoPath
- player_id
- path
- mastery
- dao_marks
```

------------------------------------------------------------------------

# Phase 4 -- Inventory

``` text
Inventory
├── Gu
├── Materials
├── Consumables
├── Treasures
└── ImmortalMaterials
```

------------------------------------------------------------------------

# Phase 5 -- Immortal Expansion

Thêm:

``` text
ImmortalState
- ImmortalAperture
- BlessedLand
- HeavenSpirit
- Tribulations
- ImmortalEssence
```

------------------------------------------------------------------------

# Phase 6 -- Data Driven

Player chỉ giữ trạng thái.

Các hệ riêng:

-   PlayerInventory
-   PlayerRecipeProgress
-   PlayerMapProgress
-   PlayerCombat
-   PlayerQuest
-   PlayerAchievement
-   PlayerGuild
-   PlayerAuction
-   PlayerImmortal

------------------------------------------------------------------------

# Thứ tự triển khai

  Phase   Ưu tiên   Độ khó
  ------- --------- ------------
  1       ★★★★★     Thấp
  2       ★★★★★     Trung bình
  3       ★★★★☆     Trung bình
  4       ★★★☆☆     Trung bình
  5       ★★☆☆☆     Cao
  6       ★★☆☆☆     Rất cao

# Kết quả mong muốn

-   Player nhỏ, rõ trách nhiệm.
-   Dễ mở rộng gameplay.
-   Chuẩn bị cho Rank 6--9.
-   Giảm tải DB.
-   Bám sát lore Cổ Chân Nhân.
