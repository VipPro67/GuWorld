# Hệ Thống Nuôi Dị Nhân Trong Tiên Khiếu

## 1. Mục tiêu thiết kế

Hệ thống dị nhân trong Tiên Khiếu không nên chỉ là:

```text
population -> +production bonus
```

Mỗi chủng tộc cần có **vai trò kinh tế / chiến đấu / luyện cổ / khai thác riêng**, có thể tự hoạt động theo thời gian.

Người chơi đóng vai trò:

- cung cấp môi trường sống;
- xây dựng resource node;
- phân phối lao động;
- đặt chính sách;
- cung cấp nguyên liệu;
- quản lý độ hài lòng / áp bức;
- phát triển quần thể;
- nuôi Cổ Sư và Cổ Tiên dị nhân.

Mục tiêu cuối game:

```text
Population
    ↓
Lao động
    ↓
Sản xuất tài nguyên / luyện Cổ
    ↓
Cổ Sư dị nhân
    ↓
Rank 5
    ↓
Thăng tiên
    ↓
Cổ Tiên dị nhân
    ↓
Truyền thừa chủng tộc
```

---

# 2. Schema hiện tại

Bảng:

```sql
public.aperture_variant_humans
```

| Column | Type | Default |
|---|---|---|
| id | UUID PK | gen_random_uuid() |
| aperture_id | UUID FK | immortal_apertures |
| race | VARCHAR(50) | |
| population | INTEGER | 0 |
| happiness | DOUBLE PRECISION | 70.0 |
| oppression | DOUBLE PRECISION | 0.0 |
| rebellion_rate | DOUBLE PRECISION | 0.0 |
| labor_intensity | VARCHAR(20) | NORMAL |
| created_at | TIMESTAMP | now() |
| updated_at | TIMESTAMP | now() |

Constraint:

```sql
UNIQUE (aperture_id, race)
```

Mỗi Tiên Khiếu hiện chỉ có một aggregate row cho mỗi chủng tộc.

Không nên nhồi toàn bộ logic mới vào bảng này.

Bảng này nên giữ vai trò:

```text
Tổng dân số
Tổng trạng thái xã hội
Tổng lao động
Tổng phát triển chủng tộc
```

---

# 3. Gameplay Core

## 3.1 Population

Mỗi chủng tộc có dân số riêng.

Population ảnh hưởng trực tiếp:

```text
dân số
→ số lao động
→ tốc độ sản xuất
→ khả năng xuất hiện Cổ Sư
→ khả năng xuất hiện thiên tài
→ khả năng sinh ra Cổ Tiên
```

Nhưng dân số cao cũng làm tăng:

```text
tiêu thụ lương thực
nhu cầu đất
nhu cầu tài nguyên
xung đột
rebellion risk
```

Có thể chia:

```text
population
working_population
gu_master_population
military_population
```

Không nhất thiết phải lưu toàn bộ ngay từ đầu.

`working_population` có thể derive:

```text
working_population =
population
* labor_participation
* labor_intensity_modifier
```

---

# 4. Vai trò của từng dị nhân

## 4.1 Mao Dân / Hairy Men

### Vai trò chính

```text
Refinement Path
Luyện Cổ
Chế tạo vật phẩm
Xử lý nguyên liệu
```

Đây nên là chủng tộc có gameplay nổi bật nhất.

### Cơ chế tự luyện Cổ

Người chơi chọn công thức muốn Mao Dân tự luyện.

Ví dụ:

```text
Recipe:
GU_TEN_JUN_STRENGTH

Materials:
IRON_STONE x20
BEAST_BONE x8
STRENGTH_GRASS x15
```

Nếu kho Tiên Khiếu đủ nguyên liệu:

```text
Mao Dân
→ lấy nguyên liệu
→ bắt đầu luyện
→ roll thành công
→ tạo Gu
```

Nếu thất bại:

```text
nguyên liệu bị tiêu hao
Gu không được tạo
```

Tức là:

> Bỏ nguyên liệu đúng công thức vào kho, Mao Dân sẽ tự luyện Cổ.

Không cần player bấm từng lần.

### Flow

```text
Recipe Queue
     ↓
Check materials
     ↓
Reserve materials
     ↓
Hairy Man Refinement Power
     ↓
Refine
     ↓
Success / Failure
```

### Tốc độ luyện

Ví dụ:

```text
refine_power =
hairy_man_population
* prosperity_modifier
* labor_modifier
* building_modifier
```

Có thể giới hạn lao động thực tế:

```text
1 Hairy Man Refinement Workshop
capacity = 5,000 workers
```

Dù có 500,000 Mao Dân nhưng chỉ có 1 workshop thì không thể dùng hết dân số.

### Success Rate

```text
success =
recipe.base_rate
+ hairy_man_bonus
+ refinement_building_bonus
+ variant_gu_master_bonus
+ variant_gu_immortal_bonus
```

Ví dụ:

```text
Base: 45%
Mao Dân bonus: +8%
Refinement Hall: +5%
Hairy Man Rank 5 Gu Master: +2%

Final = 60%
```

### Auto Craft Mode

Có thể cho người chơi chọn:

```text
[ ] Auto Refine
```

Mode:

```text
REFINE_UNTIL_SUCCESS
REFINE_X_TIMES
REFINE_UNTIL_MATERIAL_EMPTY
MAINTAIN_STOCK
```

Ví dụ:

```text
Maintain GU_STRENGTH_FIST >= 100
```

Nếu kho còn 73:

```text
Mao Dân tự luyện thêm 27
```

Đây sẽ rất hợp late-game.

---

# 5. Thạch Nhân / Rockmen

## Vai trò chính

```text
Mining
Earth Path
Stone resource
Ore extraction
Construction
```

Nếu trong Tiên Khiếu có:

```text
Iron Mine
Primeval Stone Mine
Earth Crystal Node
Metal Ore Node
```

Thạch Nhân được phân công vào node.

Ví dụ:

```text
Iron Mine

Rockmen Workers: 10,000
Base output: 1,000 Iron / year

Rockmen bonus: x1.5
Labor HIGH: x1.5

Final:
2,250 Iron / year
```

### Auto Mining

Không cần player click thu hoạch.

Tick:

```text
mine node
→ calculate workers
→ calculate output
→ add to aperture inventory
```

### Deep Mining

Dân số Thạch Nhân càng lớn có thể unlock:

```text
Normal Mining
Deep Mining
Dangerous Mining
Earth Vein Excavation
```

Deep Mining:

```text
+production
+rare materials
+resource node damage
+casualty chance
```

---

# 6. Vũ Nhân / Feathermen

## Vai trò

```text
Transportation
Cloud resources
Air routes
Exploration
Tribulation defense
```

Hiện tại đã có:

```text
feathermanPop * 0.001
```

giảm damage thiên kiếp.

Có thể mở rộng.

### Logistics

Resource node cách xa settlement:

```text
Production efficiency:
70%
```

Nếu có Vũ Nhân:

```text
Transport efficiency:
95%
```

### Aerial Gathering

Có thể khai thác:

```text
Cloud
Wind Essence
Sky Beast resources
Floating Island nodes
```

### Tribulation

Khi thiên kiếp có:

```text
meteor
storm
flying beast
sky fire
```

Vũ Nhân giảm damage đáng kể.

---

# 7. Người Tuyết / Snowmen

## Vai trò

```text
Ice Path
Snow Path
Cold biomes
Ice resource production
```

Resource:

```text
Ice Crystal
Snow Lotus
Frozen Essence
Cold Qi
```

Có thể chịu môi trường lạnh tốt hơn các race khác.

Nếu Tiên Khiếu có Ice biome:

```text
Snowmen population growth +
production +
happiness +
```

Nếu sống ở Fire biome:

```text
population growth -
happiness -
mortality +
```

---

# 8. Mặc Nhân / Inkmen

## Vai trò

```text
Information Path
Research
Administration
Deduction
Record keeping
```

Có thể liên kết cực tốt với hệ `refine/deduce`.

Ví dụ:

```text
Recipe deduction progress
+ Inkman Scholars
```

Formula:

```text
deduction_speed =
base
* inkman_population_modifier
* scholar_modifier
* information_path_building
```

Có thể cho Inkman tự:

```text
research recipe
discover resource
reduce unknown recipe requirements
improve map intel
```

---

# 9. Tiểu Nhân / Minimen

## Vai trò

```text
Wood Path
Farming
Herbs
Plant cultivation
```

Tự động chăm sóc:

```text
Spirit Grass
Gu food
Herbs
Trees
Wood Path materials
```

Gameplay:

```text
plant node
↓
Minimen workers
↓
growth speed
↓
yield
```

Có thể đặc biệt mạnh với:

```text
Gu food production
```

để nuôi Gu trong Tiên Khiếu.

---

# 10. Giao Nhân / Mermen

## Vai trò

```text
Water Path
Fishing
Sea resources
Aquatic farming
```

Tự động khai thác:

```text
Spirit Fish
Water Essence
Pearls
Sea beasts
Water Path materials
```

Nếu Tiên Khiếu có:

```text
Lake
River
Sea
```

Giao Nhân tăng mạnh hiệu suất.

---

# 11. Cô Nhân / Mushroommen

## Vai trò

```text
Poison Path
Herbal resources
Mushrooms
Medicine
```

Production:

```text
Poison mushrooms
Medicinal mushrooms
Toxic materials
Herbs
```

Có thể có bonus:

```text
Poison Path Gu refinement materials
```

---

# 12. Long Nhân / Dragonmen

## Vai trò

```text
Military
Enslavement Path
Combat
Security
```

Có thể dùng để:

```text
bảo vệ settlement
trấn áp rebellion
đánh quái trong aperture
bảo vệ resource nodes
```

Ví dụ:

```text
rebellion suppression =
dragonman_population
* military_ratio
```

Có thể unlock:

```text
Dragonman Army
```

---

# 13. Hệ phân công lao động

Không nên cho toàn bộ population tự động làm một nghề.

Nên có allocation.

Ví dụ Mao Dân:

```text
Population: 100,000

Refinement: 40%
Farming: 10%
Mining: 5%
Construction: 10%
Civilian: 35%
```

Hoặc đơn giản hơn:

```text
AUTO
BALANCED
SPECIALIZED
```

Mode:

```text
AUTO
REFINEMENT_FOCUS
MINING_FOCUS
FOOD_FOCUS
MILITARY_FOCUS
```

---

# 14. Labor Intensity

Schema hiện có:

```text
LOW
NORMAL
HIGH
EXPLOIT
```

Nên giữ.

Ví dụ modifier:

| Mode | Production | Happiness | Oppression | Mortality |
|---|---:|---:|---:|---:|
| LOW | x0.7 | + | - | - |
| NORMAL | x1.0 | 0 | 0 | 0 |
| HIGH | x1.5 | - | + | + |
| EXPLOIT | x2.0 | -- | ++ | ++ |

`EXPLOIT` không nên là bonus miễn phí.

Tick mỗi năm:

```text
EXPLOIT:

happiness -= 5
oppression += 8
rebellion_rate += 4
```

---

# 15. Cổ Sư Dị Nhân

Population đủ lớn sẽ tự sinh ra Cổ Sư.

Ví dụ tick:

```text
100,000 Mao Dân

→ 300 người khai khiếu
```

Sau đó NPC population:

```text
Rank 1: 210
Rank 2: 70
Rank 3: 17
Rank 4: 3
```

Không cần lưu từng Rank 1 NPC.

Có thể lưu aggregate:

```text
gu_master_rank1
gu_master_rank2
gu_master_rank3
gu_master_rank4
gu_master_rank5
```

Rank 5 quan trọng mới tạo NPC entity riêng.

---

# 16. Cổ Tiên Dị Nhân

Khi có Rank 5 Peak:

```text
Rank 5 Peak
↓
Ascension event
↓
success / failure
```

Nếu thành công:

```text
Variant Human Gu Immortal
```

Ví dụ:

```text
Mao Shi

Race:
Hairy Man

Rank:
6

Path:
Refinement

Effects:
+5% Hairy Man refine success
+15% refine speed
Can refine Rank 6 Immortal Gu
```

Cổ Tiên dị nhân nên là **NPC riêng**, không phải một con số aggregate.

---

# 17. Variant Human NPC

Table đề xuất:

```sql
aperture_variant_npcs
```

Ví dụ:

```sql
id
aperture_id
race

name

rank
sub_rank

path

role

loyalty

talent

status

created_at
updated_at
```

Role:

```text
REFINER
MINER
FARMER
SCHOLAR
GENERAL
GOVERNOR
```

---

# 18. Resource Node Worker Assignment

Có thể tạo:

```sql
aperture_node_workers
```

```sql
id

node_id
variant_human_id

assigned_population

created_at
updated_at
```

Ví dụ:

```text
Iron Mine

Rockmen: 20,000

Production:
4,000 Iron/year
```

Node khác:

```text
Refinement Hall

Hairy Men: 30,000

Refinement Power:
18,000/year
```

---

# 19. Auto Production Tick

Không nên tick từng người.

Không nên tick từng phút.

Nên aggregate theo:

```text
Aperture Internal Year
```

Hoặc batch worker định kỳ.

Pseudo:

```go
for aperture in apertures {
    variants := loadVariantHumans(aperture.ID)
    nodes := loadNodes(aperture.ID)

    simulateMining()
    simulateFarming()
    simulateRefinement()
    simulatePopulation()
    simulateSociety()
}
```

Tất cả tính theo aggregate.

---

# 20. Auto Refinement Worker

Pseudo logic:

```go
func ProcessHairyManRefinement(apertureID uuid.UUID) {
    queue := LoadRefinementQueue(apertureID)

    for _, job := range queue {
        recipe := LoadRecipe(job.RecipeID)

        if !HasMaterials(apertureID, recipe.Materials) {
            continue
        }

        if !HasHairyManCapacity(apertureID, recipe) {
            continue
        }

        ConsumeMaterials(recipe.Materials)

        successRate := CalculateVariantRefineRate(
            apertureID,
            recipe,
        )

        if Roll(successRate) {
            CreateGu(apertureID, recipe.OutputGuID)
        }

        UpdateJob(job)
    }
}
```

---

# 21. Refinement Queue

Table:

```sql
aperture_refinement_queue
```

Fields:

```text
id
aperture_id

recipe_id

mode
target_quantity

priority

enabled

created_at
updated_at
```

Mode:

```text
FIXED_COUNT
UNTIL_SUCCESS
UNTIL_MATERIAL_EMPTY
MAINTAIN_STOCK
```

Ví dụ:

```text
GU_LIQUOR_WORM

Mode:
MAINTAIN_STOCK

Target:
100
```

Mao Dân tự đảm bảo kho luôn có 100 con.

---

# 22. Material Storage

Mao Dân sử dụng trực tiếp inventory của Tiên Khiếu.

Ví dụ:

```text
aperture_inventory
```

Khi auto-refine:

```text
check quantity
↓
reserve
↓
consume
↓
refine
```

Nên có transaction để tránh:

```text
player craft
+
hairy man craft
```

cùng consume một material.

Ví dụ:

```sql
SELECT ...
FOR UPDATE
```

hoặc atomic update:

```sql
UPDATE inventory
SET quantity = quantity - $consume
WHERE
    quantity >= $consume;
```

---

# 23. Settlement

Population không nên tồn tại vô hình.

Có thể gắn với settlement.

```text
Hairy Man Village
Rockman Cave
Snowman Tribe
Inkman City
```

Settlement có:

```text
population_capacity
housing
food_storage
security
prosperity
```

Settlement tăng:

```text
population growth
happiness
worker efficiency
```

---

# 24. Công trình dị nhân

## Mao Dân

```text
Refinement Workshop
Refinement Hall
Gu Material Warehouse
```

## Thạch Nhân

```text
Mining Camp
Deep Mine
Earth Vein Tunnel
```

## Vũ Nhân

```text
Sky Port
Cloud Farm
Watch Tower
```

## Mặc Nhân

```text
Library
Academy
Information Hall
```

## Tiểu Nhân

```text
Spirit Farm
Herb Garden
Wood Path Nursery
```

---

# 25. Population Cost

Dân không nên miễn phí.

Mỗi race tiêu thụ tài nguyên.

Ví dụ:

```text
food_consumption =
population
* food_per_capita
```

Thiếu food:

```text
happiness ↓
mortality ↑
rebellion ↑
population ↓
```

---

# 26. Rebellion

Nếu:

```text
happiness thấp
oppression cao
labor EXPLOIT
food thiếu
```

thì:

```text
rebellion_rate ↑
```

Event:

```text
Mao Dân chiếm Refinement Hall

Effect:

Auto refine disabled
20% materials stolen
1 Rank 5 leader appears
```

Player có thể:

```text
Trấn áp
Đàm phán
Tăng food
Giảm lao động
Giết thủ lĩnh
```

---

# 27. Racial Prosperity

Có thể thêm:

```text
prosperity
```

Range:

```text
0 - 100
```

Tăng khi:

```text
food dư
housing tốt
happiness cao
có settlement
có Gu Masters
```

Giảm khi:

```text
đói
chiến tranh
tribulation
EXPLOIT
```

Prosperity buff:

```text
population growth
talent spawn
production
Gu Master emergence
```

---

# 28. Truyền thừa chủng tộc

Late-game feature.

Ví dụ Mao Dân:

Requirement:

```text
Population >= 1,000,000
Prosperity >= 80
Rank 5 Gu Masters >= 10
Gu Immortals >= 1
```

Unlock:

```text
Hairy Man Racial Inheritance
```

Bonus:

```text
Refinement speed +10%
Refinement success +5%
Rare genius spawn chance +
```

Higher level:

```text
Lv1
Lv2
Lv3
Complete
```

---

# 29. Đề xuất schema mở rộng

## aperture_variant_humans

Giữ aggregate:

```sql
population
happiness
oppression
rebellion_rate
labor_intensity

prosperity

gu_master_count
gu_immortal_count
```

---

## aperture_variant_npcs

```text
Gu Master / Gu Immortal quan trọng
```

---

## aperture_variant_settlements

```text
thành / làng / tribe
```

---

## aperture_node_workers

```text
phân công lao động
```

---

## aperture_refinement_queue

```text
Mao Dân auto refine
```

---

## aperture_variant_events

```text
rebellion
genius birth
immortal ascension
migration
disaster
```

---

# 30. MVP nên làm trước

Không nên implement tất cả cùng lúc.

## Phase 1

Làm gameplay economy.

### Mao Dân

```text
Auto refine
```

Player:

```text
chọn recipe
↓
bỏ nguyên liệu vào kho
↓
Mao Dân tự luyện
```

### Thạch Nhân

```text
Auto mining
```

### Tiểu Nhân

```text
Auto farming
```

### Giao Nhân

```text
Auto aquatic gathering
```

---

# 31. Phase 2

Society simulation.

Thêm:

```text
food
housing
prosperity
birth
death
rebellion
```

---

# 32. Phase 3

Cultivation.

Thêm:

```text
Gu Masters
Rank 5
Gu Immortals
```

---

# 33. Phase 4

End-game.

```text
Racial inheritance
Gu Immortal management
Variant Human factions
Human Tribulations
```

---

# 34. Gameplay loop đề xuất

Ví dụ một player có:

```text
Mao Dân:
120,000

Thạch Nhân:
80,000

Tiểu Nhân:
40,000
```

Player xây:

```text
Refinement Hall
Iron Mine
Spirit Herb Farm
```

Assignment:

```text
50,000 Mao Dân
→ luyện Cổ

30,000 Thạch Nhân
→ khai mỏ

20,000 Tiểu Nhân
→ trồng Gu food
```

Loop:

```text
Thạch Nhân
→ lấy Iron

Tiểu Nhân
→ tạo Herbs

          ↓

       Material

          ↓

Mao Dân
→ tự luyện Cổ

          ↓

       Gu Storage
```

Đây nên là loop kinh tế cốt lõi của hệ dị nhân.

---

# 35. Mục tiêu thiết kế cuối cùng

Player không trực tiếp micro-manage mọi hành động.

Thay vào đó player thiết kế:

```text
Một nền kinh tế bên trong Tiên Khiếu
```

Trong đó:

```text
Thạch Nhân
→ khai khoáng

Tiểu Nhân
→ trồng trọt

Giao Nhân
→ khai thác thủy vực

Vũ Nhân
→ vận chuyển

Mặc Nhân
→ nghiên cứu

Mao Dân
→ luyện Cổ

Long Nhân
→ quân sự
```

Tất cả kết nối với nhau thành:

```text
RESOURCE
   ↓
PROCESSING
   ↓
REFINEMENT
   ↓
GU
   ↓
STRONGER APERTURE
   ↓
MORE POPULATION
   ↓
GU MASTERS
   ↓
GU IMMORTALS
```

Hệ này sẽ khiến việc nuôi dị nhân trở thành một phần gameplay lớn của Tiên Khiếu thay vì chỉ là một con số bonus production.
