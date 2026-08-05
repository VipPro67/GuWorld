# Thiết kế: Bắt Monster bằng Cổ Trùng (Pet System)

> Trạng thái: **Draft** — chưa code. Plan để review trước khi triển khai.

## 1. Tổng quan

Từ bảng `monster_templates` hiện tại, người chơi có thể **bắt** một số monster trong combat
bằng cách dùng **cổ trùng có effect bắt monster**. Con thú bắt được trở thành **pet**:

- Hỗ trợ chiến đấu như 1 ally (tham gia trận như các `Allies[]` hiện tại).
- **Có thể chết vĩnh viễn** nếu HP về 0 trong trận.
- **Phải cho ăn**: đói như Cổ (`hunger`), đói quá mức → chết vĩnh viễn.

## 2. Quyết định thiết kế (đã xác nhận)

| Vấn đề | Quyết định |
|--------|-----------|
| Cách bắt | Trong combat, khi monster HP ≤ ngưỡng, player kích hoạt effect bắt của cổ trùng |
| Tỉ lệ bắt | Phụ thuộc: cổ dùng (capture bonus), rank cổ ≥ rank monster, HP còn lại, loại (Elite khó hơn) |
| Chết | Chết **vĩnh viễn** khi HP = 0 trong trận (không hồi sinh) |
| Cho ăn | Có hunger decay **theo giờ** (mở rộng worker đói hiện tại); đói = 0 → chết vĩnh viễn |
| Loot/Inventory | Pet **không** nhận loot, **không** mang `gu_inventory` của monster |
| Số pet | Không giới hạn cứng (hàng trăm/ngàn); **gộp cùng loại thành bầy** (§2c), **ra trận theo đội hình**, chia sẻ grid slot |
| Phạm vi | Chỉ bản kế hoạch thiết kế (tài liệu này) |

## 2b. Đội hình pet — triển khai nhiều pet cùng lúc

Để đưa **nhiều pet ra trận cùng lúc** mà vẫn cân bằng, dùng cơ chế **đội hình (formation)**
với số slot giới hạn:

- `player_pets.formation_slot`: `0` = nuôi (KHÔNG ra trận), `1..N` = vị trí trong đội hình.
- **Số slot ra trận** mở khóa theo rank cảnh giới người chơi (config):
  `maxFormation = clamp(1 + player.Rank, 1, MAX_SLOTS)` (vd rank 1 → 1 slot, rank 3 → 2, rank 6 → 3, cap 4).
  Dùng chung bảng config ở server, không hardcode.
- Ở combat start, nạp mọi pet có `formation_slot > 0 AND state='ACTIVE' AND hunger > 0`,
  sort theo `formation_slot`, thêm vào `Allies[]` **sau** player & party members.
- **Ràng buộc grid**: combat có 9 ô ally (3×3, `allySlots` tại `combat_state.go:101`). Đội hình
  pet chỉ điền vào các ô còn trống sau player + party; nếu party đông, pet ưu tiên cao
  (`formation_slot` nhỏ) vào trước, số dư không lên trận (log "pet đứng ngoài").
- API `pets/equip` đổi thành **set formation**: gửi danh sách slot từ frontend, server validate
  lại theo `maxFormation`.

```go
// models/player.go Combatant mới cho pet
petCombatant := models.Combatant{
    ID:          pet.ID.String(),
    Name:        pet.Nickname,            // hoặc tên monster
    Kind:        "Pet",
    Rank:        pet.Rank,
    SubRank:     pet.SubRank,
    HP:          pet.HP, MaxHP: pet.MaxHP,
    BaseAttack:  pet.BaseAttack,
    BaseDefense: pet.BaseDefense,
    Speed:       pet.Speed,
    StatusEffects: []models.StatusEffect{},
}
```

## 2c. Gộp quái cùng loại → bầy (swarm merge)

Giải pháp cho việc có **trăm/ngàn pet**: gộp các pet **cùng `monster_template_id`**
thành 1 **bầy (pack)** mạnh hơn, và bầy chỉ **chiếm 1 ô combat** (1 `formation_slot`).

- `player_pets.count` (`INT DEFAULT 1`): số cá thể trong bầy.
- **Gộp**: `POST /pets/merge { target_id, source_ids: [...] }`.
  - Điều kiện: tất cả pet cùng `monster_template_id`; nguồn (`source_ids`) phải ở kho
    (`formation_slot = 0`, không ra trận). Target có thể ở kho hoặc trong đội hình.
  - Kết quả: `target.count += len(source_ids)`, xóa các row nguồn, **tính lại stats bầy**
    theo công thức dưới, giữ nguyên 1 `formation_slot`.
- **Công thức stats bầy** (config, sub-linear để không vỡ cân bằng):
  ```
  packScale(N) = N ^ 0.65                       // N = count
  packMaxHP    = round(maxHp_1   * packScale)
  packAttack   = round(atk_1     * packScale)
  packDefense  = round(def_1     * packScale)
  speed        = speed_1                        // không đổi
  ```
  (VD: sói 2 con → ~1.57× sức mạnh; 10 con → ~4.5×; 100 con → ~20× — vẫn chiếm 1 ô.)
- **Đói của bầy**: `pet_hunger_per_hour` nhân theo quy mô
  `hungerDecay = baseRate × (1 + 0.3·(count-1))` → bầy càng đông càng ăn nhanh, đòi feed nhiều hơn.
- **Chết**: 1 bầy chết = mất hết cá thể trong bầy (vĩnh viễn) — rủi ro theo thiết kế.
- Đổi tên hiển thị: pet `count=1` giữ tên monster; `count>1` hiện "Bầy <Tên> (N)" (vd *Bầy Sói (5)*).
- Trăm/ngàn pet giờ nằm gọn trong **một vài bầy** → giảm row bảng `player_pets`, dễ quản lý & cân bằng.

## 3. Data model

### 3.1 Bảng `monster_templates` (thêm cột)

Tái sử dụng bảng hiện tại (không tạo bảng template mới).

```sql
ALTER TABLE monster_templates
    ADD COLUMN IF NOT EXISTS is_capturable   BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS capture_chance  DOUBLE PRECISION NOT NULL DEFAULT 0.20, -- base % (0..1) khi HP thấp nhất
    ADD COLUMN IF NOT EXISTS pet_hunger_rate DOUBLE PRECISION NOT NULL DEFAULT 0.5;  -- hunger giảm / 1 tick worker
```

- `is_capturable = true` chỉ áp cho monster dã ngoại (Normal/Elite). Boss/Tribulation
  boss mặc định `false` (tránh bắt boss).
- CSV seed `backend/cmd/server/data/Monster.csv` + `db.go` insert phải thêm 3 cột.
- Registry `MonsterTemplate` (`backend/internal/models/monster.go`) thêm 3 field tương ứng.

### 3.2 Bảng `player_pets` (mới)

```sql
CREATE TABLE IF NOT EXISTS player_pets (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id          UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    monster_template_id VARCHAR(50) NOT NULL REFERENCES monster_templates(id),
    nickname           VARCHAR(60) NOT NULL DEFAULT '',
    rank               INT NOT NULL DEFAULT 1,
    sub_rank           INT NOT NULL DEFAULT 1,
    hp                 DOUBLE PRECISION NOT NULL,
    max_hp             DOUBLE PRECISION NOT NULL,
    base_attack        DOUBLE PRECISION NOT NULL,
    base_defense       DOUBLE PRECISION NOT NULL,
    speed              DOUBLE PRECISION NOT NULL DEFAULT 10,
    count              INT NOT NULL DEFAULT 1,                -- số cá thể trong bầy (merge)
    hunger             DOUBLE PRECISION NOT NULL DEFAULT 100,
    max_hunger         DOUBLE PRECISION NOT NULL DEFAULT 100,
    state              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE / SLEEPING
    formation_slot      INT NOT NULL DEFAULT 0,                -- 0 = không ra trận; 1..N = vị trí đội hình
    captured_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS player_pets_player_idx ON player_pets (player_id);
CREATE INDEX IF NOT EXISTS player_pets_formation_idx ON player_pets (player_id, formation_slot);
```

- Stats snapshot tại thời điểm bắt: dùng `ScaleMonsterStats` (`models/monster.go:87`) với
  rank/subRank lúc bắt → lưu cứng để pet không bị đổi theo template. **Bắt sao, ở vậy** —
  không level/upgrade.
- **Pet không có loot & không mang theo inventory**: monster bắt được **không** nhận loot
  (`LootTableID` bỏ), **không** mang `gu_inventory` sang pet.
- Giới hạn: **không giới hạn cứng** — thiết kế để chứa trăm/ngàn pet (normalized table
  `player_pets`, index theo `player_id`, pagination ở API).
- Schema SQL thêm vào `backend/internal/db/db.go` (no-migration convention).

## 4. Effect bắt trên Cổ trùng

### 4.1 Field mới trong `ActiveEffect` (`backend/internal/models/gu.go`)

```go
CaptureBonus float64 `json:"capture_bonus,omitempty"` // +% tỉ lệ bắt (0..1)
```

Cổ có `ActiveEffect.CaptureBonus > 0` (kèm `CombatTags` chứa `"capture"`) được tính là
"cổ trùng bắt monster". Số liệu thêm vào `gu_dictionary` (qua admin/CSV), ví dụ gợi ý:

| Gu | Rank | CaptureBonus | Ghi chú |
|----|------|--------------|---------|
| `GU_WOLF_ENSLAVEMENT_2` | 1 | 0.15 | Cổ nô dịch — bắt loài sói dễ hơn |
| `GU_BLOOD_MOON` | 1 | 0.10 | Máu huyết trợ bắt huyết thú |
| `GU_GHOST_CRY` | 1 | 0.10 | Dòng effect bắt monster trên Cổ có sẵn |

### 4.2 Công thức bắt (đề xuất, config được)

Ràng buộc rank — **cổ trùng chỉ bắt được monster có rank ≤ rank cổ đó**:

```
rankRule: guCapturing.Rank >= monsterTemplate.Rank   (nếu false → không cho bắt)
```

```
captureChance = base (monster_templates.capture_chance)
              × hpFactor            (HP càng thấp càng cao, ví dụ: (1 - HP/maxHP)^0.5, clamp 0.1..1)
              × typeFactor          (Elite khó hơn: 0.6 · Normal=1.0, Boss=0 → không bắt boss)
              + Σ captureBonus của các cổ trùng EQUIPPED + ACTIVE
```

- `Type` là cổ trong dòng effect "bắt monster" của **một số Cổ có sẵn** — không tạo series
  riêng biệt. Cổ có `ActiveEffect.CaptureBonus > 0` (kèm `CombatTags` chứa `"capture"`) được
  tính là cổ trùng bắt monster.
- Cuộn `rand` mỗi lần bắt. Thành công → thêm pet; thất bại → tốn 1 lượt hành động
  (có thể monster phản công).
- Chỉ áp dụng cho monster `is_capturable`; PVP/Arena/Instance không cho bắt.

## 5. Luồng bắt trong combat

1. Player đang combat với monster capturable, HP enemy ≤ 30% max (ngưỡng config).
2. Player gọi `POST /combat/capture` (không gọi attack lượt này).
3. Handler (`combat_capture.go` mới, cùng package `handlers`):
   - Kiểm tra `ActiveCombat` hợp lệ, monster capturable, HP ≤ ngưỡng.
   - Tìm cổ trùng `EQUIPPED + ACTIVE` có `CaptureBonus`; áp rule rank (§4.2) → tính chance.
   - Không có cổ trùng bắt đủ rank → 400 kèm thông báo.
   - Roll kết quả:
     - **Thành công**: tạo row `player_pets` (snapshot stats, `formation_slot=0`), set
       `ActiveCombat = NULL` (trận kết thúc, **không nhận loot**), ghi audit log.
     - **Thất bại**: trừ mana/action, enemy giữ nguyên; append log vào `CombatHistory`.
4. Chú ý: `CombatState` (`models/player.go:133`) thêm field
   `CaptureAttempted bool` để chặn spam nhiều lần bắt trong 1 trận (hoặc giới hạn số lần).

## 6. Pet hỗ trợ chiến đấu

- Ở `InitializeCombatState` (`combat_state.go:14`) & `combat_auto.go` build sim: nạp **toàn bộ
  pet trong đội hình** (`formation_slot > 0` & `state='ACTIVE'` & `hunger > 0`), sort theo
  `formation_slot`, append vào `base.Allies` sau player + party (xem §2b — giới hạn theo
  `maxFormation` và grid 9 ô).
- Mỗi pet là 1 `Combatant{ID: petID, Kind: "Pet", ...}` với stats snapshot từ `player_pets`.
- `combat_auto.go:buildSimCombatant`: build pet combatant (simulator `CombatantAuto` với
  `Actions = [BASIC_ATTACK]`).
- Pet tự động đánh basic attack mỗi lượt (không tốn mana), nhận sát thương từ monster,
  **không** heal trong trận — HP giảm vĩnh viễn.
- Pet chết trong trận: khi `HP <= 0` → sau khi resolve combat, `DELETE FROM player_pets`
  + audit log "pet.died_in_combat".
- Vị trí pet điền vào các ô `allySlots` còn trống sau player/party (`combat_state.go:101`).

## 7. Đói & cho ăn (worker)

- **Hunger decay theo giờ**: `pet_hunger_rate` (đổi tên `pet_hunger_per_hour`) giảm mỗi giờ,
  tính theo thời gian trôi (`updated_at` → `NOW()`), không phụ thuộc tần suất tick worker.
  - Worker hiện tại tick 3 giờ/lần (`main.go:117`); vẫn dùng chung tick nhưng decay =
    `pet_hunger_per_hour × hours_elapsed`.
  - Pet `state='SLEEPING'` giảm 50% tốc độ.
  - `hunger <= 0` → **chết vĩnh viễn**: `DELETE FROM player_pets` + audit log
    "pet.starve_death" (không backlash HP người chơi như vital Gu).
- Feed: `POST /pets/feed` nhận `pet_id` + `item_id`. Tiêu hao item trong
  `player_inventory_items` (qua `InventoryService`, reuse category `consumables`), tăng
  `hunger = min(max_hunger, hunger + value)` dựa `ItemRegistry[item_id].Effects["hunger"]`
  (thêm effect `hunger` cho các món thức ăn như `boar_meat`, `moon_orchid_petal`...).

## 8. API endpoints (mới, thêm vào `backend/internal/server/server.go`)

```text
Group /pets (AuthRequired, RateLimitByPlayer 30/min):
  GET    /pets?page=&limit=  -> danh sách pet phân trang (stats, hunger, state, formation_slot)
  PUT    /pets/formation     -> { slots: [petId,...] } set đội hình (validate maxFormation)
  POST   /pets/feed          -> { pet_id, item_id } cho ăn
  POST   /pets/sleep         -> { pet_id, sleeping: bool } chuyển trạng thái nghỉ
  POST   /pets/merge         -> { target_id, source_ids } gộp bầy cùng loại (§2c)
  POST   /pets/release       -> { pet_id } thả (tùy chọn, xác nhận 2 lần)

Group /combat (thêm route):
  POST   /combat/capture     -> bắt monster đang trong combat (handlers.CaptureMonster)

Admin (nhóm admin sẵn có):
  GET    /admin/players/:id/pets      -> xem pet người chơi
  DELETE /admin/players/:id/pets/:id  -> xóa pet (debug)
```

## 9. Frontend (sau khi backend ổn định)

- `useGameStore.ts`: thêm slice pet (list, formation, feed, sleep, capture) + SSE cập nhật hunger.
- UI:
  - Tab/Panel "Cổ Thú" trong profile: danh sách pet, thanh hunger, nút cho ăn/nghỉ/thả,
    **UI kéo-thả đội hình** (số slot hiện theo rank).
  - CombatScreen: nút "Bắt" hiện khi enemy capturable + HP ≤ ngưỡng + có cổ trùng bắt;
    hiển thị pet trong đội hình như ally; log chết pet.
- Dictionary/Admin: quản lý cột `is_capturable`, `capture_chance`, capture bonus của Gu,
  `maxFormation` theo rank.

## 10. Cập nhật tài liệu (theo AGENTS.md)

| Thay đổi | Tài liệu |
|----------|----------|
| Routes mới `/pets`, `/combat/capture`, admin pets | `docs/API.md` |
| Schema `player_pets`, cột `monster_templates`, CSV, effect `CaptureBonus` | `docs/GAME_DATA.md` |
| Luật gameplay pet (bắt/chết/đói) | `WIKI.md` |
| Phạm vi dự án | `PROJECT.md` |

## 11. Milestone triển khai

- **M1 Backend data**: schema + registry + CSV (`monster.go`, `db.go`, `Monster.csv`).
- **M2 Bắt monster**: field `CaptureBonus`, `combat_capture.go`, route, log/audit.
- **M3 Pet trong combat**: load pet vào `Allies[]`, death vĩnh viễn, build sim.
- **M4 Đói/cho ăn**: worker pet hunger, `pets/feed`, item effect `hunger`.
- **M5 Admin + tests**: `*_test.go` cho capture formula & feed; endpoints admin.
- **M6 Frontend**: store + UI pet panel + nút capture.
- **M7 Docs**: cập nhật theo §10.

## 12. Cân bằng & quyết định đã chốt

### Quyết định đã chốt (từ review)

- **Rank bắt**: cổ trùng rank nào bắt được monster rank đó (rank cổ ≥ rank monster). Elite khó hơn (typeFactor).
- **Quy mô pet**: thiết kế để chứa **trăm/ngàn pet** — no hard cap, normalized table + pagination;
  **gộp cùng loại → bầy** (`count`, chiếm 1 ô) + **ra trận theo đội hình** (`formation_slot`),
  số slot mở theo rank người chơi (§2b, §2c).
- **Không upgrade**: bắt sao thì ở vậy — stats snapshot cố định tại lúc bắt, không level/power-up.
- **Cổ bắt**: không tạo series riêng; nằm trong dòng effect của **một số Cổ có sẵn** (`CaptureBonus`).
- **Không loot / không inventory**: pet bắt được không nhận loot, không mang `gu_inventory`.
- **Đói theo giờ**: `pet_hunger_per_hour` tính theo thời gian trôi, không theo số tick worker.
- **Gộp bầy**: cùng `monster_template_id` gộp thành bầy `count` cá thể, stats `N^0.65`,
  chiếm 1 ô combat, đói tăng theo quy mô (§2c).

### Câu hỏi mở (vẫn cần quyết)

- Số slot đội hình tối đa (`MAX_SLOTS`) và hệ số mở theo rank là bao nhiêu cho cân bằng?
- Hệ số `N^0.65` và đói `1 + 0.3·(count-1)` có hợp lý? có giới hạn `count` tối đa 1 bầy không?
- `pet_hunger_per_hour` = ? (vd 2/h → đói 50 giờ mới chết). SLEEPING có giảm đói 50% không?
- Thức ăn pet dùng item nào, mỗi item +bao nhiêu hunger?
