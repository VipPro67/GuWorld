# Game Data

## Effect block migration audit

Legacy effects are converted by format, not through a shared best-effort
decoder. Formats are `gu`, `item`, `gamefx`, `killer_move`, and
`aperture_variant_event`. Gu conversion separately audits `active`, `passive`,
`growth`, `other`, `explore`, `support`, `refinement`, and `house`; unknown or
context-dependent semantics are reported rather than guessed.

```bash
cd backend
go run ./cmd/migrate_gu_effects -format gu -source data/Gu.csv > effect-audit.json
go run ./cmd/migrate_gu_effects -format item -source data/Item.csv \
  -write -output /tmp/Item.blocks.csv -report /tmp/item-audit.json
```

The command is a dry-run by default. `-write` requires a distinct `-output`
path and refuses to overwrite its source. Reports expose `converted`,
`validated`, `golden_parity`, `differential_parity`, `exact_parity`,
`shadow_parity`, `unsupported`, `ambiguous`, `duplicate`,
`divergent_explicit_block_legacy`, and `suggested_mapping` fields.

The levels are separate: `converted` only means a candidate was produced;
`validated` means that exact candidate passes the Block v2 contract; golden and
differential parity are evidence from their respective runners; and shadow
parity means runtime comparison found no mismatch. `exact_parity` is populated
only from successful golden/differential evidence, never inferred from
conversion or an empty problem list. A validated empty effect envelope is a
valid migration and does not require a non-zero effect count. Otherwise the
report stays in `audit` until every converted effect has parity evidence and
every problem array is empty. CLI and database backfill use the same gate error
code and reason, and `-write` refuses a closed gate. Aperture variant event
effects currently have no runtime semantic contract, so their converter
deliberately reports them as ambiguous. `aperture_variant_events` is excluded
from the database cutover owner list until it has a native executor and a
parity oracle; an empty envelope must not be used as substitute semantics.

`status_dictionary.blocked_actions` is action-gating metadata, not an effect,
and remains in its dedicated column. Race, constitution, and status modifiers
use `ALWAYS_ON`/`SELF` blocks. Item mappings must choose `ON_CONSUME` or
`ON_USE` from the native execution surface rather than applying one generic
trigger to every legacy key.

Gu candidates contain only `schema_version` and `blocks`; legacy sibling
namespaces are not copied into candidate output.

Backend shadow comparison primitives are opt-in with `EFFECT_BLOCK_SHADOW=1`.
Rollout is intentionally staged in this order: player derived stats,
capabilities, growth projections, item results, world/refinement commands,
combat actions, then deterministic combat outcomes. The legacy result remains
authoritative. Worker/world/item comparisons may only project commands/results;
the shadow callback never receives an effect applicator and therefore cannot
apply a second side effect. Mismatches are normalized for floating-point noise,
map ordering, and nil/empty collections, then emitted as structured
`effect_block_shadow_mismatch` records with definition ID/rank and catalog
revision.

For a non-empty candidate, golden/differential evidence alone never opens the
migration gate. The runtime evidence recorder must attach the exact immutable
snapshot ID, catalog revision, and observed `shadow_mismatch_count` to the
machine-readable `effectmigration.Report`. Missing or unbound evidence keeps the
gate closed; it is never inferred from a default counter.

Nguồn dữ liệu chính nằm trong `backend/data/`. `db.InitSchema()` tạo/patch schema và chỉ seed dữ liệu hệ thống tối thiểu; các dictionary gameplay phải được import/export qua Admin → **Dữ liệu**. Riêng quest hiện vẫn được đồng bộ từ CSV lúc khởi tạo schema.

### Authorization data

Authorization của Admin được chuẩn hóa trong bốn bảng: `admin_roles`,
`admin_permissions`, `admin_role_permissions` và `player_admin_roles`. Schema seed role
`super_admin`, permission `admin.access`, mapping giữa hai giá trị, rồi backfill mọi
player legacy có `is_admin=true` sang `super_admin`. Trong giai đoạn tương thích, nếu
một player vẫn có `is_admin=true` nhưng chưa có assignment trong `player_admin_roles`,
resolver tạm coi player đó là `super_admin` và ghi warning để vận hành phát hiện và
sửa backfill. Assignment trong database là nguồn quyền lâu dài; JWT chỉ giữ identity
và session, không chứa role hoặc permission.

## CSV gameplay files

| File | Nội dung |
| --- | --- |
| `backend/data/Gu.csv` | Dictionary Cổ: rank, path, type, mana cost, effects, hunger, targeting, rarity, role/tags. |
| `backend/data/Item.csv` | Vật phẩm, nguyên liệu, tiền tệ, consumable và effect. |
| `backend/data/Recipe.csv` | Công thức luyện Cổ, nguyên liệu, tỉ lệ và chi phí. |
| `backend/data/Monster.csv`, `LootTable.csv`, `MonsterSpawn.csv` | Template quái, bảng loot và quy tắc spawn. |
| `backend/data/Region.csv`, `MapCell.csv` | Vùng/zone và ô persistent của thế giới. |
| `backend/data/Race.csv`, `Constitution.csv`, `Status.csv` | Chủng tộc, thể chất và trạng thái data-driven. |
| `backend/data/Quest.csv`, `DreamRealm.csv` | Nhiệm vụ và mộng cảnh/phó bản. |
| `backend/data/Sect.csv`, `SectMission.csv` | Môn phái và mẫu nhiệm vụ môn phái. |
| `backend/data/ShopItem.csv`, `ArenaShopItem.csv` | Shop thương đội và shop Võ Đài. |
| `backend/data/PremiumShopPackage.csv`, `TopupPackage.csv` | Gói premium và gói nạp; được handler đọc trực tiếp, không import qua Admin. Kết quả có Redis cache 24 giờ. |
| `backend/data/ApertureNodeTemplate.csv` | Template node Tiên Khiếu: cost, production, HP, defense, attack range, cost Cổ và zone. Đọc trực tiếp lúc khởi động; sửa CSV cần restart. |
| `backend/data/disposable_email_domains.txt` | Danh sách domain email tạm bị chặn khi đăng ký; đọc lúc khởi động. |

Không ghi số dòng cố định trong tài liệu vì dataset thay đổi thường xuyên. Các helper tìm file ở `data/` trước rồi mới thử đường dẫn legacy. Chỉ sửa CSV chưa làm đổi dictionary trong DB: phải import dataset tương ứng. Workflow Admin luôn parse/validate và dry-run trước, hiển thị row count/diff (`inserts`, `updates`, `unchanged`) rồi mới cho xác nhận commit transaction. Preflight có lỗi nghiêm trọng không ghi database. Sau commit server tự reload registry/invalidate cache; UI sẽ cảnh báo rõ nếu dataset hoặc lỗi reload vẫn yêu cầu restart. Với hai file package đọc trực tiếp, cần chờ cache 24 giờ hoặc xóa cache/restart để thấy thay đổi ngay.

## Registry runtime

- `models.LoadGuRegistry(db.DB)` load `gu_dictionary` vào `models.GuRegistry` khi server start.
- `models.LoadItemRegistry(db.DB)` load `item_dictionary` vào `models.ItemRegistry` khi server start.
- Runtime gameplay không đọc trực tiếp các namespace authoring `active`, `passive`,
  `growth`, `other` trong các domain derived stats, scouting/profile/refinement
  và growth. Registry compile chúng thành projection bất biến; item use cũng
  dispatch qua danh sách lệnh typed, có thứ tự ổn định, thay vì để handler đọc
  trực tiếp map JSON.
- Các registry constitution, race, status, monster và zone cũng được load từ database khi server start.
- API `/api/dictionary` trả dữ liệu dictionary cho frontend.
- API `/api/recipes` trả recipe refining cho frontend.

## Cấu trúc cơ sở dữ liệu

### Phân quyền quản trị — Admin RBAC

RBAC quản trị được lưu trong bốn bảng:

- `admin_roles`: định nghĩa vai trò; `key` là định danh ổn định và duy nhất.
- `admin_permissions`: danh mục quyền, khóa chính là `key`.
- `admin_role_permissions`: ánh xạ nhiều-nhiều giữa vai trò và quyền.
- `player_admin_roles`: gán nhiều-nhiều vai trò cho `players`.

Khi khởi tạo schema, hệ thống seed idempotent các role `admin`, `observer`,
`support_admin`, `content_admin`, `game_admin`, `finance_viewer`, `super_admin`,
permission vào dashboard `admin.dashboard.access`, wildcard `admin.*` và các
quan hệ tương ứng. `observer` chỉ đọc dữ liệu vận hành; `support_admin` xử lý
người chơi; `content_admin` quản lý nội dung; `game_admin` vận hành game/world;
`finance_viewer` chỉ đọc billing; `super_admin` nhận wildcard. Role tương thích
`admin` chỉ nhận quyền vào dashboard và có thể được cấp thêm quyền domain. Sau khi tài khoản admin mặc định
được bảo đảm tồn tại, các tài khoản admin cũ được backfill idempotent vào RBAC.

Authorization của dashboard và API quản trị được resolve từ các role/quyền
RBAC. Cột `players.is_admin` được giữ như projection tương thích cho client cũ,
nhưng không phải nguồn quyết định hiển thị entry Admin ở Lobby.

### Bảng `immortal_apertures` (Tiên Khiếu)

Tạo ngay tại thời điểm người chơi thăng tiên lên Rank 6. `grade` được roll theo Nhân Khí lúc thăng tiên (LOW / MID / HIGH / EXTREME). Các cột chính:
- `time_rate` (DOUBLE PRECISION, DEFAULT 10) — gia tốc thời gian nội bộ
- `grade` (VARCHAR) — LOW / MID / HIGH / EXTREME
- `aperture_hp`, `max_aperture_hp` (BIGINT) — HP của Tiên Khiếu (BIGINT để tránh tràn số ở cấp độ cao)
- `area_size` — diện tích grid, quyết định số node tối đa
- `immortal_essence`, `essence_production_per_year` — Tiên Nguyên
- `rank_start_year` — năm bắt đầu rank hiện tại (dùng làm mốc tính lịch giáng lâm tai kiếp)
- `earth_calamity_count`, `heavenly_tribulation_count`, `grand_tribulation_count`, `myriad_tribulation_count`, `chaos_calamity_count` — bộ đếm kiếp nạn
- `tribulation_pending_at` (TIMESTAMPTZ, NULL) — thời điểm tai kiếp bắt đầu bị "giữ" vì người chơi offline; dùng để tính phạt giảm thưởng Đạo Ngân (quá 24 giờ → còn 50%)
- `state` — STABLE / TRIBULATION_PENDING / TRIBULATION_ACTIVE / DAMAGED / DESTROYED

> **Hằng số thời gian nội bộ:** 1 năm ngoại giới = **36.000 giây (10 giờ) đời thật** khi `time_speed_multiplier = 1`; 1 năm nội bộ Tiên Khiếu = **36.000 giây đời thật ÷ (`time_rate` × `time_speed_multiplier`)**. Sản lượng nền của Tiên Khiếu là `essence_production_per_year` sau mỗi năm nội bộ, nên tốc độ nền mỗi giờ đời thật là `essence_production_per_year ÷ 10 × time_rate × time_speed_multiplier`; Cổ, thể chất, tuổi thọ và hiệu ứng khóa hồi phục vẫn được áp dụng sau đó. Kiếp nạn đầu tiên đến vào năm nội bộ thứ 10 → chờ khoảng `10 × 36.000 ÷ time_rate` giây đời thật (khoảng 2–10 giờ, tùy phẩm cấp Tiên Khiếu). Kiếp đang ACTIVE quá **6 giờ đời thật** (chỉ tính thời gian online) tự động kết toán thất bại.
> **Bảo hộ offline:** khi đến mốc kiếp mà người chơi offline, worker không sinh quái, đặt `state='TRIBULATION_PENDING'`. Khi online lại mới spawn và có 5 phút grace trước khi quái di chuyển/phá mỏ.

### Đạo Ngân (Đạo Ấn)

Đạo Ngân chỉ lưu trong cột JSONB `players.dao_marks` (ví dụ `{"strength": 100, "fire": 2500}`). Bảng quan hệ `dao_marks` đã bị xóa — không còn mirror. Path được normalize dạng lowercase (không có hậu tố `" path"`) khi lưu.

### Cảnh Giới Lưu Phái — cột `players.path_masteries`

Cột `path_masteries` (JSONB) lưu **điểm tích lũy** dạng `map[string]int`, giống cột `dao_marks` của `immortal_apertures` — ví dụ `{"fire": 1250, "water": 340}`. Cảnh giới (cấp độ 0-4) được suy ra từ mốc điểm qua `gamefx.PathMasteryLevel(points)`:

| Level | Tên | Hệ số | Mốc điểm |
|-------|-----|-------|----------|
| 0 | Ordinary (Phàm Thai) | ×1.0 | < 10.000 |
| 1 | Master (Đại Sư) | ×1.1 | ≥ 10.000 |
| 2 | Grandmaster (Tông Sư) | ×1.25 | ≥ 100.000 |
| 3 | Great Grandmaster (Đại Tông Sư) | ×1.4 | ≥ 1.000.000 |
| 4 | Supreme Grandmaster (Vô Thượng Đại Tông Sư) | ×1.5 | ≥ 10.000.000 |

Điểm nhận từ: dùng Cổ (theo bảng rank bên dưới), sát chiêu thành công, luyện Cổ thành công (theo rank Cổ đích), và thưởng Chân Ý phó bản (cộng thẳng `amount`). Dữ liệu cũ lưu tên cấp được migration tự động đổi sang mốc điểm khi khởi động.

**Giới hạn điểm Chân Ý mỗi trận:** các nguồn trong trận chiến thường từ Cổ phàm và Sát chiêu phàm (đánh tay, đánh tự động/Simulator, sát chiêu phàm) bị chặn ở `gamefx.PathMasteryBattleGainLimit(rank)` = **Rank × 10 điểm/trận**. Riêng **Tiên Cổ và Sát chiêu Tiên đạo (Rank ≥ 6)** không bị giới hạn tối đa trong trận chiến, thỏa sức ngộ đạo (uncapped). Counter lưu trong `players.active_combat.path_mastery_gained`, tự reset khi trận mới bắt đầu. Thưởng từ phó bản (attainment) và luyện cổ không tính vào giới hạn này.

**Chân Ý khi dùng Cổ:** mỗi lần thôi động Cổ thành công nhận Chân Ý cơ bản theo rank: **R1=1, R2=2, R3=5, R4=10, R5=20, R6=50, R7=100, R8=200, R9=300**. Lượng thực nhận vẫn chịu hiệu suất theo cảnh giới hiện tại và giới hạn cảnh giới của rank Cổ.

### Phó Bản — bảng `instance_templates`

- `type` — `INHERITANCE_RIGHTEOUS`, `INHERITANCE_DEMONIC`, `DREAM_REALM` (Mộng Cảnh), `SECRET_REALM` (Bí Cảnh).
- `cooldown_hours` — số giờ chờ trước khi vào lại phó bản (mặc định 24; `0` = không giới hạn). Cooldown tính từ session gần nhất của player cho instance đó.
- `rewards` JSONB gồm `attainment` (Chân Ý → `players.path_masteries`), `dao_marks` (Đạo Ngân → `immortal_apertures.dao_marks`), `materials`, `primeval_stones`.
- Seed (`SeedInstanceTemplates`): truyền thừa `dungeon_flower_wine`, **mộng cảnh data-driven từ `data/DreamRealm.csv`** (gồm `dream_realm_star_constellation` + **49 mộng cảnh** `dream_realm_<path>`, 1 cho mỗi lưu phái, thưởng +50 Chân Ý cho đúng path đó, vào bằng `dream_path_material`, cooldown 24h) và **3 bí cảnh** (`SECRET_REALM`, thưởng Đạo Ngân, cooldown 48h).
- `DreamRealm.csv` — cột `id, name, path, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours`. Cột `path` là key lưu phái (vd `fire`); nếu `map_data` để trống, stage tree sẽ được sinh tự động theo path (`buildDreamRealmMapData`). Muốn viết cốt truyện riêng cho một mộng cảnh, điền `map_data` (JSON array stage, xem format mộng cảnh thủ công `dream_realm_star_constellation` trong file). Import/export qua Admin tab "Dữ liệu" → bảng "Mộng cảnh".
- `player_instance_sessions` — session đang chạy; index bán phần `player_active_session_idx` đảm bảo 1 phó bản ACTIVE/player.

**Cảnh giới là điều kiện thăng tiên rank:** để đột phá lên Rank 7/8/9 khi vượt Kiếp Nạn, người chơi cần ≥ 1 lưu phái đạt Tông Sư (500)/Đại Tông Sư (2.500)/Vô Thượng (10.000). Kiểm tra trong `tribulation.go` `handleSuccess`.

### Tiêu hao Chân Nguyên / Tiên Nguyên khi dùng Cổ

Công thức tính tiêu hao nằm ở `gamefx/energy.go` (`EnergyCost` / `EnergyCostForCombatant`):

- **Phàm nhân (Rank 1–5):** chi phí = `(BaseManaCost/100) × MaxMana` (mặc định 10% MaxMana). Cổ rank thấp hơn giảm chi phí ÷10/rank; **Cổ rank cao hơn bị nhân hệ số lũy tiến**.
- **Cổ Tiên (Rank ≥ 6):** chi phí = `BaseManaCost` (mặc định 1). Cổ dưới Rank 6 miễn phí; Cổ rank cao hơn nhân hệ số lũy tiến; Cổ rank thấp hơn chia 100^chênh lệch.

**Hệ số rank cao hơn** (`HigherRankManaMultiplier(guRank, userRank)`, theo chênh lệch rank Cổ − rank người dùng):

- **Cổ phàm (rank 1–5):** theo bảng `{1, 2, 5, 25, 125}` cho gap `{0,1,2,3,4}` — ví dụ rank 1 dùng Cổ rank 5 (gap 4) tốn **125×** Chân Nguyên.
- **Cổ Tiên (rank ≥ 6):** mỗi bậc chênh lệch nhân ×100 (`100^gap`) — rank 6 dùng Cổ rank 7 tốn ×100, rank 6 dùng rank 8 tốn ×10.000.

- `refine_mana_tier` (passive Cổ cùng rank đang trang bị) giảm 50% chi phí.
- **Trang bị & Sử dụng Cổ rank cao hơn:** Cho phép **trang bị** và **sử dụng trong chiến đấu** Cổ trùng rank cao hơn trong cùng khoảng cảnh giới (`gamefx.CanEquipGu`):
  - **Khoảng Phàm cổ (Rank < 6):** Phàm Cổ Sư (Rank 1–5) được trang bị và dùng Phàm Cổ rank cao hơn (Rank 1–5). Không thể trang bị/sử dụng Tiên Cổ (Rank ≥ 6).
  - **Khoảng Tiên cổ (Rank ≥ 6):** Cổ Tiên (Rank 6–9) được trang bị và dùng Tiên Cổ rank cao hơn (Rank 6–9) cũng như Phàm Cổ.
  - **Tổn hại chiến đấu:** Khi sử dụng Cổ trùng rank cao hơn trong combat, người dùng phải chịu tiêu hao Chân Nguyên/Tiên Nguyên nhân theo hệ số lũy tiến (`HigherRankManaMultiplier`), đồng thời chịu **phản phệ trừ % máu (15% MaxHP cho mỗi bậc rank chênh lệch)** (`gamefx.HigherRankHPCost`). Mỗi lần equip/unequip, server tự động tháo các Cổ không hợp lệ (Phàm Cổ Sư trang bị Tiên Cổ, loại Consumable, hoặc không có spec); Cổ Bản Mệnh không bao giờ bị tự tháo.

### Nhân vật — bảng `players` (race / gender / constitution / statuses)

Quyền quản trị hiệu lực được resolve qua `player_admin_roles` và
`admin_role_permissions`. Permission `admin.dashboard.access` điều khiển quyền
vào dashboard; `admin.*` là wildcard dành cho `super_admin`.

`players.state_revision` cùng tổng counter trong `player_collection_revisions`
tạo bộ đếm tăng đơn điệu cho trạng thái người chơi. Trigger trên `players` ghi
trực tiếp vào dòng người chơi; trigger trên bảng con tăng một trong 64 counter
phân mảnh theo transaction trong `player_collection_revisions` để không khóa
dòng `players` hay một counter chung đang được nhiều API sử dụng. Snapshot, delta realtime
và mutation response cộng hai giá trị để client loại bỏ dữ liệu đến trễ hoặc
đảo thứ tự.

Các cột mô tả nhân dạng và trạng thái của người chơi, giá trị lưu bằng **key tiếng Anh**, effect được resolve tập trung trong `backend/internal/gamefx/fx.go`:

| Cột | Kiểu | Giá trị | Effect |
| --- | --- | --- | --- |
| `race` | VARCHAR(50) | Xem bảng chủng tộc bên dưới. Mặc định `human` lúc đăng ký. **Key phải tồn tại trong bảng `race_dictionary`** | Hiệu ứng đọc từ `race_dictionary.effects` (per-row) |
| `gender` | VARCHAR(20) | `male`, `female` | Chưa có effect (chỉ hiển thị) |
| `constitution` | VARCHAR(100) | Xem bảng thể chất bên dưới. **Key phải tồn tại trong bảng `constitution_dictionary`** | Hiệu ứng đọc từ `constitution_dictionary.effects` (per-row) |
| `statuses` | JSONB (mảng object) | Xem bảng trạng thái bên dưới | Hiệu ứng đọc từ `status_dictionary.effects` (per-row) |

Chọn lúc đăng ký (`race`, `gender`); `constitution`/`statuses` đổi qua Admin panel (`PUT /api/admin/players/:id`) hoặc event trong game.

**Trạng thái (statuses) — data-driven, seed từ `data/Status.csv` vào bảng `status_dictionary`, nạp registry trong memory (`gamefx.StatusRegistry`) khi khởi động:**

Mỗi phần tử trong `players.statuses` là object `{ "id": string, "expires_at": timestamptz?, "source": string?, "value": number? }`. `expires_at` vắng mặt = vĩnh viễn; có giá trị = tự hết hạn theo **giờ thực** (worker dọn dẹp mỗi phút, effect resolution cũng lọc lazy). Format cũ (mảng string, ví dụ `["severely_wounded"]`) vẫn được unmarshal tương thích.

Bảng `status_dictionary`:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `key` | VARCHAR(100) PK | Key tiếng Anh, lưu vào `players.statuses[].id` |
| `name` | VARCHAR(200) | Tên tiếng Việt hiển thị |
| `effects` | JSONB | Cùng format `effects` như thể chất (stat_mods, luck, ...) |
| `default_duration_minutes` | INTEGER | 0 = vĩnh viễn; > 0 = thời hạn mặc định khi được cấp không kèm duration |
| `blocked_actions` | JSONB (`[]string`) | Chặn hành động: `explore`, `cultivate`, `combat`, `refine`, `trade` |
| `description` | TEXT | Mô tả |

Danh sách hiện tại (7 trạng thái):

| Key | Tên Việt | Effect | Thời hạn | Chặn |
| --- | --- | --- | --- | --- |
| `severely_wounded` | Trọng Thương | -20% attack/defense/speed, -50% hồi phục nhận | Vĩnh viễn | `explore` |
| `poisoned` | Trúng Độc | -10% attack, -10% speed | 30 phút | — |
| `exhausted` | Kiệt Sức | -30% speed, -50% hiệu suất tu luyện | 10 phút | `cultivate` |
| `blessed` | Phúc Trạch | +5 may mắn, +10% hiệu suất tu luyện | 60 phút | — |
| `death_penalty` | Hình Phạt Tử Vong | -50% attack/defense/speed | 30 phút | — |
| `refine_fatigue` | Tâm Thần Mệt Mỏi | -50% tỉ lệ Luyện Cổ | 10 phút | — |
| `refine_backfire` | Luyện Cổ Phản Phệ | -95% tỉ lệ Luyện Cổ | 20 phút | — |
| `refine_insight_1..9` | Luyện Cổ Tâm Đắc R1..R9 | +1%/tầng cho đúng rank đó, không cap (Value = số tầng) | Vĩnh viễn (reset về 0 khi luyện thành công rank đó) | — |
| `sinh_khi_tan` | Sinh Khí Tàn | Tầng I -10%, II -25%, III -35%, IV -50%, V -99% attack/defense/speed (Value âm ghi đè stat_mods) | 300 phút, refresh mỗi hit | — |

Sinh Khí Tàn do **Sát Sinh Khí Cổ** (`GU_KILL_ENERGY`, rank 9 Sát Đạo) gây ra: đánh trúng ở PvP thật thì defender persistent +1 tầng (tối đa V, tầng V vẫn ăn hit refresh, không miễn nhiễm). Công thức luyện: `GU_KILL_HEAVEN_DAO` rank 8 + `heaven_qi_rank8/9`, `kill_path_material`, `primordial_kill_origin_stone` (vật liệu Hỗn Độn trong spec gốc chưa tồn tại dưới dạng item nên tạm thay bằng tiên tài Sát Đạo hiện có). Cơ chế **data-driven trên Cổ**: `Gu.csv` block `ON_HIT` + `UNIQUE PERSIST_STATUS_ON_HIT` (`params.status_key=sinh_khi_tan, pvp_only=true`); engine gắn `CombatLog.SourceGuID/HitLanded` (khiên hấp thụ hết vẫn tính hit, né/mù/miễn nhiễm không tính); hook post-combat `handlers.maybeGrantPersistentHitStatuses` (dùng chung `AutoCombatStart` + `AutoResolveCombat`) grant theo spec, Cổ mới chỉ cần thêm block tương tự. Helper: `gamefx.SinhKhiTanTierFromValue/ValueForTier/TierName`, `handlers.GrantSinhKhiTanHit`, `models.GuPersistentHitStatus`, `handlers.CleanseSinhKhiTanOneTier` (thanh tẩy tụt 1 tầng, hết tầng I thì xóa).

Helper dùng chung: `gamefx.Get(player)` (resolve effects + BlockedActions), `gamefx.HasBlockedAction(player, action)`, `gamefx.BlockingStatuses(player, action)`, `handlers.GrantPlayerStatus/RevokePlayerStatus/CleanExpiredPlayerStatuses` (cấp/gỡ/dọn trạng thái trong transaction), worker `worker.StartStatusExpiryWorker` (dọn hết hạn mỗi phút).

**Bảng chủng tộc (race) — data-driven, seed từ `data/Race.csv` vào bảng `race_dictionary`, nạp registry trong memory (`gamefx.RaceRegistry`) khi khởi động:**

Bảng `race_dictionary`:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `key` | VARCHAR(100) PK | Key tiếng Anh, lưu vào `players.race` |
| `name` | VARCHAR(200) | Tên tiếng Việt hiển thị |
| `effects` | JSONB | Các effect của chủng tộc (cùng format `effects` như thể chất bên dưới) |

Danh sách hiện tại (11 chủng tộc):

| Key | Tên Việt | Hiệu ứng |
| --- | --- | --- |
| `human` | Nhân Tộc | Không bonus đặc biệt (mặc định) |
| `cuong_thi` | Cương Thi | Bất tử tuổi thọ (không giảm thọ); các block `DISABLE_*_REGEN` đặt hệ số hồi tự động HP, Mana, Niệm Lực và Thần Hồn về 0; hồi chủ động có tiêu hao vẫn dùng được |
| `fur_people` | Người Lông | +10% tỉ lệ Luyện Cổ thành công |
| `merman` | Nhân Ngư | Thủy Đạo +10% affinity, +5% sát thương Thủy Đạo |
| `feathered` | Vũ Dân | +10% tốc độ, Không Gian Đạo +5% affinity |
| `inkman` | Mặc Nhân | Trí Tuệ Đạo +8%, Hồn Đạo +5% affinity |
| `rockman` | Thạch Nhân | +15% phòng thủ, -10% tốc độ |
| `snowman` | Tuyết Dân | Hàn Băng Đạo +15% affinity, -10% sát thương Hỏa nhận |
| `variant_beastman` | Thú Nhân | +8% tấn công, +5% phòng thủ, -5% tỉ lệ Luyện Cổ |
| `dragonman` | Long Nhân | +10% tấn công, +10% phòng thủ, -5% tỉ lệ Luyện Cổ |
| `giant_sun_descendant` | Cự Dương Huyết Mạch | +5 Vận, +5% hiệu quả tu luyện |

**Thêm chủng tộc mới:** sửa `backend/cmd/server/data/Race.csv` (thêm dòng `key,name,effects`) → Admin panel tab **Dữ liệu** → Import bảng `Chủng tộc`. Nếu database rỗng, Go có `defaultRaces()` trong `backend/internal/gamefx/races.go` làm fallback để không vỡ hành vi cũ. `players.race` luôn là **1 key duy nhất** (default `human`).

**Bảng thể chất (constitution) — data-driven, seed từ `data/Constitution.csv` vào bảng `constitution_dictionary`, nạp registry trong memory (`gamefx.ConstitutionRegistry`) khi khởi động:**

Bảng `constitution_dictionary`:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `key` | VARCHAR(100) PK | Key tiếng Anh, lưu vào `players.constitution` |
| `name` | VARCHAR(200) | Tên tiếng Việt hiển thị |
| `effects` | JSONB | Các effect của thể chất (format bên dưới) |

Format `effects` (JSON object, toàn bộ field đều optional):

| Key | Ý nghĩa |
| --- | --- |
| `stat_mods` | `{"attack": 0.10, "defense": 0.10, "speed": 0.10}` — % chỉ số |
| `dao_mark` | `{"fire": 0.20}` — cộng affinity Đạo Ngân theo hệ |
| `path_damage` | `{"dream": 0.15}` — +% sát thương hệ đó |
| `path_resistance` | `{"fire": 0.10}` — giảm % sát thương nhận từ hệ đó |
| `mana_regen` | % hồi Chân Nguyên (mỗi thể chất tự quyết, VD `1` = +100%) |
| `max_mana` | % Chân Nguyên tối đa |
| `cultivation_efficiency` | % hiệu quả tu luyện |
| `dao_mark_gain` | % Đạo Ngân nhận được |
| `luck` | Vận (số phẳng) |
| `refine` | % tỉ lệ Luyện Cổ thành công |
| `healing_received` | % healing nhận (và % hồi máu thụ động) |
| `hp_regen` | HP/giờ cộng thẳng vào hồi máu thụ động ngoài combat (số phẳng, VD `50` = +50 HP/giờ) |
| `mental_resistance` | % kháng hiệu ứng tâm lý |
| `soul_defense` | % phòng thủ hồn |
| `soul_regen` | % tăng tốc hồi Soul HP thụ động (dưỡng hồn), VD `0.50` = +50% hồi/giờ |
| `aperture_stress_bonus` | hệ số cộng thêm áp lực Không Khiếu mỗi lần trùng kích (VD `1` = tăng ×2) |

**Thêm thể chất mới:** sửa `backend/cmd/server/data/Constitution.csv` (thêm dòng `key,name,effects`) → Admin panel tab **Dữ liệu** → Import bảng `Thể chất`. Nếu database rỗng, Go có `defaultConstitutions()` trong `backend/internal/gamefx/constitutions.go` làm fallback để không vỡ hành vi cũ.

Danh sách hiện tại (11 thể chất — 10 Thập Tuyệt + Thuần Mộng Cầu Chân Thể):

| Key | Tên Việt | Hiệu ứng |
| --- | --- | --- |
| `solar_sun_moon_body` | Thái Nhật Dương Mãng Thể | Quang Đạo đạo ngân ×150%, +20% sát thương Quang Đạo, +50% max mana, +100% hồi mana |
| `ancient_moon_yin_desolate_body` | Cổ Nguyệt Âm Hoang Thể | Hắc Ám Đạo đạo ngân ×150%, +50% max mana, +100% hồi mana, +20% kháng hiệu ứng tâm lý |
| `northern_dark_ice_soul_body` | Bắc Minh Băng Phách Thể | Hàn Băng Đạo đạo ngân ×150%, giảm 10% sát thương Hàn Băng nhận, +50% max mana, +100% hồi mana |
| `forest_sea_reincarnation_body` | Sâm Hải Luân Hồi Thể | Thủy Đạo đạo ngân ×150%, +50% hồi phục nhận, +50% max mana, +100% hồi mana |
| `myriad_gold_brilliant_flower_body` | Vạn Kim Diệu Hoa Thể | Kim Đạo đạo ngân ×150%, +30% attack, +30% defense, +50% max mana, +100% hồi mana |
| `blazing_glory_lightning_brilliance_body` | Viêm Hoàng Lôi Trạch Thể | Lôi Đạo đạo ngân ×150%, +20% speed, +50% max mana, +100% hồi mana |
| `great_strength_true_martial_body` | Đại Lực Chân Võ Thể | Lực Đạo đạo ngân ×150%, +30% attack, +30% defense, +50% max mana, +100% hồi mana |
| `carefree_wisdom_heart_body` | Tiêu Dao Trí Tâm Thể | Trí Đạo đạo ngân ×150%, +30% kháng hiệu ứng tâm lý, +50% max mana, +100% hồi mana |
| `great_earth_origin_body` | Hậu Thổ Nguyên Ương Thể | Thổ Đạo đạo ngân ×150%, +45% defense, +45% kháng hiệu ứng tâm lý, +50% max mana, +100% hồi mana |
| `universe_great_evolution_body` | Vũ Trụ Đại Diễn Thể | Không Gian Đạo đạo ngân ×150%, +15% attack/defense/speed, +50% max mana, +100% hồi mana |
| `pure_dream_reality_seeker_body` | Thuần Mộng Cầu Chân Thể | Mộng Đạo đạo ngân ×140%, +10% sát thương Mộng Đạo, +40% max mana, +100% hồi mana |

### Cổ của người chơi — bảng `player_gu`

Cột `players.gu_inventory` (JSONB) đã được **thay thế hoàn toàn** bằng bảng quan hệ `player_gu`. Mỗi row = một Cổ instance của một người chơi:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | VARCHAR(80) PK | = `instance_id` của Cổ. Được tham chiếu bởi `killer_moves.core_player_gu_id` / `killer_move_slots.player_gu_id`. |
| `player_id` | UUID FK | → `players(id)` ON DELETE CASCADE |
| `gu_id` | VARCHAR(50) | FK (cùng `gu_rank`) → `gu_dictionary(id, rank)` |
| `gu_rank` | INTEGER | Cấp Cổ |
| `rank` | INTEGER | Cột legacy, giữ bằng `gu_rank` |
| `level` | INTEGER | Mặc định 1 |
| `hunger` | DOUBLE PRECISION | Độ no |
| `state` | VARCHAR(30) | ACTIVE / SLEEPING / DEAD / ... |
| `location` | VARCHAR(30) | Mặc định `INVENTORY` |
| `is_locked` | BOOLEAN | |
| `equipped` | BOOLEAN | Đang trang bị |
| `is_vital` | BOOLEAN | Bản Mệnh Cổ |
| `integration_time` | DOUBLE PRECISION | |
| `equipped_at` | TIMESTAMPTZ | NULL nếu chưa trang bị |
| `growth_hours` | DOUBLE PRECISION | |
| `last_fed_at` | TIMESTAMPTZ | Lần cuối được cho ăn. Chỉ các feed handler (`gu.feed`/`feed_group`/`feed_all`) ghi trực tiếp; không nằm trong đường sync bulk. Dùng cho audit log chết đói (thời gian không được cho ăn). |
| `created_at` / `updated_at` | TIMESTAMPTZ | |

- Migration `migrate_player_gu_from_inventory` (chạy một lần khi còn cột `gu_inventory`) copy toàn bộ dữ liệu cũ sang `player_gu`, sau đó `DROP COLUMN IF EXISTS gu_inventory` loại bỏ cột. An toàn chạy lặp lại.
- Cổ đang trang bị/Bản Mệnh vẫn được mirror trong cột JSONB `players.equipped_gus` (danh sách `instance_id`) để các truy vấn nhẹ (status) không cần join.
- API `GET /api/player/status/:id` chỉ trả Cổ đang trang bị/vital; toàn bộ túi Cổ lấy qua `GET /api/player/inventory/gus` (helper `db.GuInventoryAggregate` / `db.LoadPlayerGus`).
- `monster_templates.gu_inventory` (JSON trong cột monster) **giữ nguyên**, không đổi.
- Cột `players.temp_loot` đã bị **bỏ hẳn** (migration `ALTER TABLE players DROP COLUMN IF EXISTS temp_loot;`). Chiến lợi phẩm khám phá giờ vào thẳng túi thật (nguyên liệu → `player_inventory_items`, thạch → `players.primeval_stones`/`immortal_stones`, Cổ → `player_gu`).

### Cơ chế rơi đồ khi tử đạo & Thiên Đạo Tiêu Hủy (Death Penalty & Void Destruction)
- **Tỷ lệ phạt cơ bản khi chết:** 40% Nguyên Thạch & Tiên Nguyên Thạch, 40% chủng loại nguyên liệu (tối đa 30 loại), 30% Cổ trùng (ngoại trừ Cổ Bản Mệnh; Cổ trang bị chỉ rơi khi Sát Khí > 5), 30% Bí phương.
- **Hệ số Oán Khí:** Sát Khí tăng tỷ lệ hao hụt tài nguyên (`mult = 1.0 + level * 0.5`).
- **Thiên Đạo Tiêu Hủy (40% Void Destruction):** Trong số tài nguyên bị mất, 40% bị chấn nát tiêu hủy vĩnh viễn khỏi server (chống hành vi tử đạo để tuồn/chuyển tài nguyên cho tài khoản khác). 60% còn lại rơi ra bãi đất `map_ground_drops` tồn tại trong 10 phút.


### Bảng `aperture_resource_nodes`

Node tài nguyên đặt trong Tiên Khiếu:
- `aperture_id` (UUID, FK → immortal_apertures, CASCADE DELETE)
- `node_type`, `name`, `path` — loại node
- `level`, `hp`, `max_hp` — cấp và HP
- `x`, `y` — toạ độ grid (0-19)
- `production_type`, `production_amount`, `production_interval_hours` — sản lượng
- `is_destroyed` — trạng thái hỏng

### Kinh tế và xã hội Dị Nhân trong Tiên Khiếu

- `aperture_node_workers`: gán một phần dân số `aperture_variant_humans` vào node; cặp node/chủng tộc là duy nhất.
- `aperture_refinement_queue`: hàng đợi luyện chế theo recipe, mode, mục tiêu, tiến độ, ưu tiên và trạng thái bật/tắt; `last_error` và `last_processed_at` cho biết lý do dừng/lần xử lý cuối.
- `aperture_variant_settlements`: khu định cư theo chủng tộc, sức chứa, nhà ở, kho lương và an ninh.
- `aperture_variant_events`: sự kiện xã hội (loại, mức độ, effects JSONB, thời điểm giải quyết).
- `aperture_variant_npcs`: cá thể Cổ Sư/Cổ Tiên với rank, lưu phái, vai trò, trung thành, thiên phú và trạng thái.

Các cột xã hội trên quần thể lưu kho lương, lương thực đầu người, tăng trưởng và tử vong. Xây khu định cư đưa 500 cư dân ban đầu và lượng lương thực bằng sức chứa kho; mô phỏng xã hội tăng/giảm dân số nhưng không vượt sức chứa khu định cư. Worker production, society simulation và Gu Master spawning lần lượt chạy theo chu kỳ 10, 5 và 30 phút. Mỗi tick production cũng thử một lần cho các lệnh tự luyện ưu tiên cao. Một lần thử cần `rank × 100` Mao Dân chưa được phân công (Nhất Chuyển 100, Ngũ Chuyển 500, Cửu Chuyển 900); worker giữ chỗ tổng lao động trong tick và bỏ qua lệnh chưa đủ người để lệnh cấp thấp hơn vẫn có thể chạy: cần Mao Dân, bí phương hoàn thiện, nguyên liệu thường và nguyên thạch; thất bại tiêu hao nguyên liệu, thành công tạo một Cổ và tăng tiến độ. Lệnh thiếu tài nguyên tạm chờ và tự thử lại ở tick sau. Bí phương dùng Cổ khác làm nguyên liệu chưa được nhận vào hàng đợi tự động. API cho phép phân công lao động (tổng phân công không vượt dân số), quản lý hàng đợi, nâng khu định cư và đổi vai trò/giải tán NPC; xem route map trong `docs/API.md`.

### Bảng `tribulation_events`

Lưu các sự kiện Kiếp Nạn của Tiên Khiếu:
- `id` (UUID PRIMARY KEY)
- `aperture_id` (UUID FK → `immortal_apertures(id)`)
- `player_id` (UUID FK → `players(id)`)
- `tribulation_type` (VARCHAR) — EARTH_CALAMITY / HEAVENLY_TRIBULATION / GRAND_TRIBULATION / MYRIAD_TRIBULATION / CHAOS_CALAMITY
- `path` (VARCHAR) — Đường hướng kiếp nạn (ví dụ: fire, water, ice)
- `difficulty` (INTEGER)
- `status` (VARCHAR) — ACTIVE / SUCCESS / FAILED
- `aperture_hp_before` (BIGINT) — HP Tiên Khiếu trước kiếp nạn
- `aperture_hp_after` (BIGINT) — HP Tiên Khiếu sau kiếp nạn (NULL nếu chưa kết thúc)
- `reward_dao_marks` (BIGINT) — Số Đạo Ngân thưởng
- `started_at` / `ended_at` (TIMESTAMPTZ)

### Bảng `tribulation_enemies`

Lưu các quái vật sinh ra trong Kiếp Nạn Tiên Khiếu:
- `id` (UUID PRIMARY KEY)
- `event_id` (UUID FK → `tribulation_events(id)`)
- `name` (VARCHAR)
- `enemy_type` (VARCHAR) — BOSS / MINION
- `path` (VARCHAR)
- `hp`, `max_hp` (DOUBLE PRECISION) — HP hiện tại và tối đa của quái vật (dùng DOUBLE PRECISION / float64 chống tràn số ở rank cao)
- `attack`, `defense` (DOUBLE PRECISION) — Tấn công và phòng thủ của quái vật (dùng DOUBLE PRECISION / float64 chống tràn số)
- `target_node_id` (UUID NULL FK → `aperture_resource_nodes(id)`)
- `x`, `y` (INTEGER) — Toạ độ grid (0-19)
- `is_boss` (BOOLEAN)
- `is_dead` (BOOLEAN)

## Nhóm dữ liệu gameplay

### Cổ

Các trường quan trọng trong dictionary:

- `id`, `name`, `rank`, `path` (mảng JSON các Đạo học — 49 path có thể có, xem danh sách ở `docs/CANH_GIOI_DAO_NGAN.md`), `gu_type`.
- `base_mana_cost`.
- `effects` JSON: active/passive/growth/other/support tuỳ Cổ. Schema Go nằm ở `models.GuEffects` (`backend/internal/models/gu.go`): `ActiveEffect`, `PassiveEffect`, `GrowthEffect`, `SpecialMechanic`.
  - **Validation khi ghi dữ liệu:** `effectcatalog.NormalizeAndValidate` kiểm tra JSON root, tám namespace chuẩn (`active`, `support`, `passive`, `house`, `explore`, `refinement`, `growth`, `other`), kiểu container, status và cấu trúc `SpecialMechanic`. Admin CRUD trả HTTP 400 kèm `issues[]` có `severity`, `code`, `path`, `message` nếu dữ liệu sai. Import `Gu.csv` preflight toàn bộ effect trước database write đầu tiên.
  - Field đã phân loại nhưng chưa được runtime xử lý đầy đủ vẫn được giữ nguyên và phát warning theo disposition (`effect_migrate`, `effect_partial`, `effect_known_bug`); warning không đồng nghĩa effect đã hoạt động. Cách warning-first này tránh mất dữ liệu trong giai đoạn migrate. Namespace lạ, JSON/kiểu container sai, status hoàn toàn không biết và mechanic thiếu trường bắt buộc là lỗi chặn ghi/import.
  - Manifest `effectcatalog` gán mỗi key/status/cặp `(event_trigger, action_code)` một `owner` và `disposition`: `supported`, `partial`, `metadata`, `migrate` hoặc `known_bug`. Key hoặc mechanic chưa có manifest bị chặn bằng lỗi `unclassified_*`; test coverage toàn `Gu.csv` buộc dữ liệu mới phải được phân loại trước khi merge.
  - Admin có preview read-only `POST /api/admin/gu-effects/validate` và audit database `GET /api/admin/gu-effects/audit`. Audit không thực thi effect; nó cho biết coverage và những effect đang chờ migrate/sửa lỗi.
  - **Khối `active`:** effect chủ động dùng trong combat — `damage_flat`, `damage_mult`, `damage_path`, `heal_flat`, `shield_flat`, `apply_status`, `target_selector`, `summon_*`, `curve_attack`, `lifesteal`, … Danh sách đầy đủ + catalog chuẩn xem mục **"Hiệu ứng Cổ (effects)"** bên dưới.
    - **`apply_status` là khối lắp ráp được:** một action có thể gắn **nhiều status cùng lúc** dưới dạng mảng `[{"id":"BLEED","value":10,"turns":3},{"id":"BIND","turns":2}]`. Ba dạng authoring đều hợp lệ: chuỗi legacy (`"BIND"`), object đơn (`{"id":"BIND","turns":6}`), mảng (khuyến nghị). Entry thiếu `value`/`turns` sẽ kế thừa `status_tick_damage`/`status_duration_ticks` cấp action (chỉ dùng cho dữ liệu cũ; dữ liệu mới khai báo trọn trong từng entry). Entry có thêm **`chance` (0..1, optional)** — tỉ lệ kích hoạt mỗi lần dùng, **0 = luôn trúng** (mọi row cũ không có `chance` giữ nguyên hành vi). Resolve qua `statusfx` registry (`models.ActiveEffect.ResolveStatusApplications`, bản rank-scale `...ForRank` dùng ở simulator build).
    - **Rank-scale cho entry có `chance`:** `chance_thực = min(0.95, chance + 0.05×(rank−1))`, `turns_thực = turns + (rank−1)/3` (rank 1–3: +0, 4–6: +1, 7–9: +2). VD Hoang Mộ Cổ (`GU_DEATH_DESOLATE_TOMB_4`, `chance: 0.35, turns: 1`): rank 4 → 50%/2 lượt; cùng entry ở rank 9 → 75%/3 lượt. Miss log `STATUS_MISS`, roll trước miễn nhiễm/kháng.
    - Key phần trăm trực tiếp (0..1, áp sau sát thương thường): `damage_hp_percent` (% MaxHP mục tiêu, xuyên khiên), `shield_flat_reduce_percent` (% khiên hiện tại bị tước), `disable_gu_next_turn` (bool: `GU_SEAL` mục tiêu — xem enforcement trong simulator), `forget_reality_chance` (cổng proc cho cả gói bonus), `self_evasion_boost_percent` (buff `EVASION` bản thân), `target_damage_reduce_percent` (cường độ `WEAK`/`FORGET_REALITY` lên mục tiêu). `damage_path` ghi đè hệ đòn đánh; `stat_base_attack` trong active cộng flat vào sát thương đòn.
  - **Khối `passive` (Block v2 duy nhất, không còn đọc flat cũ):** chỉ số vĩnh viễn cộng dồn (`STAT_MOD ... SPEED`, `MOVEMENT` = phạm vi di chuyển ô — tốc chạy map, `SPEED` chỉ thứ tự đánh combat), `damage_reduction`, `evasion`, `luck`, `storage_capacity`, `tenacity` (0..1: rút ngắn duration CC + tỉ lệ tự giải 1 debuff đầu lượt), `cc_resist` (0..1: kháng proc mọi debuff, nhân với mental resist, cap tổng 0.75), …
    - Khắc hệ theo Gu path (Block v2): `STAT_MOD PATH_RESISTANCE ADD_FLAT` kèm `path` (`heaven`/`earth`/`dream`, 0..1, merge vào `CombatantAuto.PathResistance`, nhận diện theo `Gu.path` của đòn đánh), `UNIQUE IMMUNE_DREAM_STATUS` (miễn nhiễm status có `spec.Path == "dream"`), `forget_heaven/earth_protection` (cờ metadata).
  - **Shorthand status (đã loại khỏi dữ liệu):** key viết tắt (`freeze`, `slow`, `slow1`, `weak`, `stun`, `silence`, `bleed`, `poison`, `thorns`, `regen`, `atk_up`, `def_up`, `taunt`) không còn xuất hiện trong `Gu.csv` — toàn bộ đã migrate sang `apply_status` dạng mảng. `ActiveEffect.normalize()` vẫn giữ làm fallback cho DB rows cũ chưa re-import; **không dùng cho dữ liệu mới**.
  - **Cơ chế `other` (SpecialMechanic):** mảng `{event_trigger, action_code, params}`. Ngoài các trigger thế giới (`ON_TICK`, `ON_CONSUME`, `ON_USE`), combat hỗ trợ `ON_CAST`, `ON_HIT`, `BEFORE_HIT`, `BEFORE_ENEMY_ATTACK`, `ON_TURN_START`, `ON_DEATH`, `ON_BATTLE_START` và `COMBAT_PASSIVE` (`PASSIVE` cũ vẫn hợp lệ, cùng nghĩa).
    - **Timing kích hoạt (xem `effectcatalog/timing.go`):** `ALWAYS_ON` (race/constitution/status còn hạn/Cổ equipped-ACTIVE, resolve qua `gamefx.Get` mỗi lần đọc) / `ON_ACTION` (`active.*` khi ra đòn, `ON_CONSUME`/`ON_USE` khi dùng Cổ) / `ON_EVENT` (hook combat + status trận) / `SCHEDULED` (`growth`, `ON_TICK`, hết hạn status) / `CAPABILITY` (cổng quyền, không tự chạy) / `META` (`description`).
    - Simulator nhận một snapshot mechanic bất biến được biên dịch khi bắt đầu trận. `gu_id`/`SourceID` chỉ còn dùng cho định danh, log, ưu tiên và trừ độ đói; thay ID nhưng giữ nguyên effect phải cho cùng hành vi.
    - Các mechanic đặc thù hiện đã data-driven gồm: `POEM_INTENT`, `SUMMON_WOLF_FRONTLINE`, `FULL_SOUL_HEAL`, `SWORD_DRAGON_TRANSFORM`, `BORROW_PAST_POWER`, `UNRAVEL_RIDDLE`, `CHAOS_GAMBLE`, `BYPASS_SHIELD`, `FIRST_TURN_EVADE`, `SEIZE_INITIATIVE`, `DOUBLE_DAMAGE_DEALT`, `REBORN`, `FORGET_INTENT_INTERRUPT`, `FORGET_WILL_CONFUSE`.
    - `SEIZE_INITIATIVE` (`event_trigger: ON_BATTLE_START`, Thỉnh Đạo Hữu Tiên Hành Cổ): trên lượt 1, nếu thọ nguyên đối thủ chính (`Enemies[0]`; quái dùng bảng quy đổi theo rank, Cổ sư dùng `players.lifespan`) không hơn `lifespan_ratio` (default 3) lần thọ nguyên chủ Cổ thì chủ Cổ nhảy lên đầu thứ tự hành động, địch vẫn đánh sau theo tốc độ thường (không bỏ lượt).
    - `REBORN` thống nhất (`mode: rewind` reset cả trận / `in_place` đứng dậy tại chỗ): `chance` (0..1, required), `mode`, `max_uses` (1–10, default 1), `turn1_guaranteed`, `revive_hp_percent` (default 0.5), `clear_debuffs` (default true), `empower_attack/speed/shield_percent` + `empower_turns` (default 100; stack `ATK_UP`/`HASTE` + khiên mỗi lần sống dậy). Counter `rebornUses` sống qua rewind; `REVIVE_LOCK` chặn cả 2 mode; mode `rewind` ưu tiên khi cùng lượt có cả 2. **Phe địch chỉ hồi sinh `in_place`** (không rewind); mode bỏ trống ở phe ta = `rewind` như cũ.
    - `BYPASS_SHIELD` (`ON_HIT`) chỉ xuyên Khiên Ảo — không trừ thủ (muốn trừ thủ dùng `ignore_defense*`). `FIRST_TURN_EVADE` chỉ né đòn đánh đầu tiên Lượt 1 (multi-hit/AoE sau đó vẫn trúng, rider `damage_hp_percent`/strip của đòn bị né cũng bị bỏ). Đòn đánh thường fallback chạy qua `runOne` nên chịu `FORGET_INTENT/WILL`, backlash, mana, hunger và ngộ đạo như đòn thường.
    - Alias consume đã chạy (không còn 400): `STONE_APERTURE`→`APERTURE_PETRIFICATION`, `POISON_VOW`→`CONTRACT_OATH`, `THIRD_WATCH`→`TIME_ACCELERATION`, `RECORD_MESSAGE`→`RECORD` (dùng qua endpoint Support, trigger `ON_USE`). `MAN_BEAST_BURIAL`/`SPRING_EGG_HATCH` vẫn chưa hỗ trợ. Endpoint Support persist đủ cột như Consumable.
    - Build phe địch chỉ nhận Cổ **đã trang bị + ACTIVE** (khớp phe ta; hunger vẫn miễn cho quái vì quái đánh ở Hunger 0 theo thiết kế, hunger-gate của engine chỉ áp dụng Kind Player).
    - `GU_SEAL` (`disable_gu_next_turn`, `FORGET_EGO`, `FORGET_INTENT_INTERRUPT`, backfire): engine enforce trong `runOne` — combatant `HasGuSeal` không thể dùng `GU_SKILL` (đánh thường vẫn được, như `SILENCE`); seal N lượt = `Duration N+1` vì duration tick đầu lượt; Tenacity có thể tự giải.
    - `BEFORE_ENEMY_ATTACK` chạy trong `runOne` trước khi đòn đánh resolve: `FORGET_INTENT_INTERRUPT` (`chance`, `disable_target_gu_turns`, `max_successful_activations`, `backfire_after_max`) seal đòn hiện tại vẫn trúng, lượt sau bị cấm; `FORGET_WILL_CONFUSE` (`activation_chance`, `hit_user/self/miss_chance`, `backfire_chance`, `backfire_disable_turns`) override target đã resolve (đánh chủ Cổ / tự đánh mình / hụt hẳn).
    - Status mới: `FORGET_EGO` (behaviour `gu_seal`), `FORGET_REALITY` (behaviour `weak`, cường độ từ `target_damage_reduce_percent`).
    - `REBORN.params.chance` là xác suất chuẩn trong catalog (0..1); runtime không còn dùng tỷ lệ hard-code theo ID.
    - Action code liên quan Hồn Phách:
    - `SOUL_REGEN_TICK` (`event_trigger: ON_TICK`): hồi `soul_hp` theo giờ từ mỗi Cổ trang bị ACTIVE (`params.value` = số điểm/giờ, scale theo rank qua `ManaRegenTickFactor`; xử lý ở `models.SoulRegenTickPerHour`).
    - Ba nguồn `*_REGEN_TICK` là hồi thụ động: chúng được cộng với hồi nền trước khi áp dụng chính sách `DISABLE_HP_REGEN`, `DISABLE_MANA_REGEN`, `DISABLE_RESOURCE_REGEN` (Mana + Niệm Lực) và `DISABLE_SOUL_REGEN`. Item/Cổ `ON_CONSUME` là hồi chủ động nên không bị hệ số hồi tự động này chặn.
    - `RESTORE_SOUL_HP` (`event_trigger: ON_CONSUME`): hồi ngay `soul_hp` khi sử dụng Cổ tiêu hao (`params.value` mặc định 10, cap tại `max_soul`).
    - `ADD_SOUL_MAX` (`ON_CONSUME`): tăng `max_soul`; `SOUL_BOOST` (`ON_CONSUME`): cộng Hồn Đạo đạo ngân.
    - Action code liên quan dòng thời gian:
    - `INCREASE_TIME_SPEED` (`event_trigger: ON_CONSUME`): cộng vĩnh viễn `params.value` vào `players.time_speed_multiplier` → hồi mana/hp/soul/niệm nhanh hơn nhưng tuổi thọ cũng trôi nhanh hơn.
- `hunger_rate`, `food_types`, `max_hunger`. **Quy ước `hunger_rate`:** cổ có `food_types` không rỗng → `6` nếu `effects.passive` không rỗng, ngược lại `3`; cổ không có thức ăn → `0` (không đói). `max_hunger` mặc định `rank × 100`.
- `target_type`: `SINGLE`, `ROW`, `COLUMN`, `ALL`, v.v.
- `rarity`, `is_immortal`, `primary_role`, `combat_roles`, `combat_tags`, `support_tags`.
  - **`rarity` là số nguyên 0–9** (`0` = chưa đặt, fallback theo rank). Dùng để cho Cổ phàm (rank < 6) độ hiếm cao: drop weight tính theo **`max(rank, rarity)`** (`handlers/effectiveDropLevel`), nối vào gacha, rương, và ô WildGu trên map. Item (`item_dictionary`) cũng có cột `rarity` cùng quy tắc. Giá trị cũ dạng chữ/boolean trong CSV đã được dọn về rỗng.
- `base_price`: hỗ trợ tối đa 3 chữ số thập phân (đơn vị nhỏ nhất `0.001`); **fallback đồng đều theo rank** — 200, 500, 1.200, 2.500, 5.000, 100.000, 250.000, 600.000, 1.500.000 cho rank 1–9 (nhảy mạnh ở Tiên Cổ rank ≥ 6). Fallback trong `pkg/pricing/pricing.go`.
- **Phân phối thức ăn:** mọi item trong `food_types` của cổ rank <6 (72 item) được thêm vào (1) loot của quái **Normal/Elite** theo dải rank spawn, (2) `pool_weights.resources` của vùng non-HUB theo dải rank vùng, (3) `vault_materials` khởi đầu của 4 tông (qty 100, khớp dải rank vùng tông). Boss/Tribulation không rơi thêm thức ăn.
- **Tính duy nhất của Tiên Cổ (rank ≥ 6):** cổ trùng rank ≥ 6 là **duy nhất toàn server, tính theo `gu_id`** (cùng id khác rank vẫn bị tính trùng). Áp dụng cho instance **đang sống** ở mọi nguồn: `player_gu`, `auction` ACTIVE (`item_type='Gu'`), `sect_vault_gu` (`quantity>0`). Enforce 2 lớp: app-level `db.CheckImmortalGuExists` (+ advisory lock, trả **409** khi trùng: refine/gacha/deposit) và backstop DB `uq_player_gu_immortal` / `uq_sect_vault_gu_immortal` + `chk_sect_vault_gu_immortal_qty` (kho rank≥6 luôn `quantity<=1`). Destroy (đói chết/thả/chết rơi đồ) = end lifecycle `instance_id` đó, `gu_id` được tái sinh với `instance_id` mới. Nguồn kiểm tra: `db.CheckImmortalGuExists` / `db.GetExistingImmortalGuIDs` / `db.IsImmortalConflict`.
- **`instance_id` bất biến xuyên vòng đời:** trade, đăng bán/đấu giá, ký gửi, nhập/xuất kho tông môn chỉ đổi chủ/vị trí, không đổi `id`. Đăng bán Tiên Cổ mỗi đơn 1 instance; rút kho Tiên Cổ mỗi lần 1 instance, giữ nguyên id. Regen id mới chỉ cho legacy snapshot thiếu id rank < 6.
- **`gu_history_logs` chỉ log Tiên Cổ (rank ≥ 6):** `LogGuHistory` drop mọi event rank < 6. Event terminal (`STARVED_DEATH`, `DEATH_DESTROYED/DROPPED`, `DROPPED_RELEASED` với `metadata.terminal=true`) đóng timeline của `instance_id`.
- **Tiên Cổ không xuất hiện từ loot/region:** mọi entry loot table có `gu_id` là cổ rank ≥ 6 bị **lọc bỏ** khi (1) import CSV LootTable, (2) Admin tạo/sửa bảng loot, (3) `RollLoot` roll thực tế. `pool_weights.wild_gus` của Region cũng được lọc Cổ Tiên khi import CSV (xem `filterImmortalWildGus`) và khi sinh ô WildGu (`GenerateCell`), đảm bảo Cổ Tiên chỉ đến từ gacha, luyện cổ và đấu giá.

### Hiệu ứng Cổ (effects)

## Đặc tả effect thống nhất v2 (authoritative)

> Mục này là hợp đồng đích đã chốt. Struct `ComposableEffectBlock` và trường
> `blocks` hiện tại chỉ là prototype chuyển tiếp; chúng **không** phải v2 và
> không được dùng để suy ra rằng runtime đã thực thi block.

### Biên dữ liệu và inventory legacy

Inventory được quét trên toàn bộ `Gu.csv` (4.223 dòng), `Item.csv` (641 dòng),
`Race.csv`, `Constitution.csv`, `Status.csv`, seed fallback trong
`schema_dictionaries.go`, `killer_moves.effects` và
`aperture_variant_events.effects`. Số lượng chỉ là snapshot kiểm toán; contract
được xác định bởi key, không bởi số dòng.

Các thuộc tính sau **không phải effect** và phải ở ngoài envelope: `name`,
`description`, rank/path/type/rarity/tags và ảnh; `base_mana_cost`, targeting/cast/
cooldown cấu hình; `Status.default_duration_minutes` (policy khi cấp status),
`Status.blocked_actions` (access policy do handler kiểm tra), `Constitution.tier`;
`hunger_rate`, `food_types`, `max_hunger` và quy tắc chết đói; metadata/lifecycle
của aperture event. Duration trong block là lifetime của một runtime effect,
không thay thế duration mặc định của dictionary status.

**Matrix A — namespace/key Cổ legacy.** Key không ghi `[parser-only]` được quan
sát trực tiếp trong `Gu.csv.effects` và persist ở `gu_dictionary.effects`;
`[parser-only]` nghĩa là chỉ còn trong Go model/manifest để đọc DB cũ, không có
trong snapshot CSV hiện tại. UI label là `Gu.name` cộng formatter
operation (legacy chưa có label riêng); definition do `gu_dictionary` sở hữu,
runtime combat nằm trong combat snapshot, còn world/worker command persist bởi
handler/worker tương ứng. `catalog` bên dưới nghĩa là các test
`effectcatalog/{catalog,validate,timing}_test.go`; `characterization` gồm test
handler/simulator/worker hiện có. Mọi key không có executor đầy đủ được giữ trong
inventory nhưng **không được coi là supported**.

| Legacy namespace/key (mỗi key trong ô là một entry độc lập) | Trigger → target | Unit / scaling | Executor hiện tại; persistence owner | Test hiện có |
|---|---|---|---|---|
| `active.damage_flat`; `damage_mult`; `damage_path`; `damage_hp_percent` | action → selected enemy(s) | flat HP; multiplier; path enum; ratio max HP | combat handler + simulator; combat state | catalog + simulator |
| `active.shield_flat`; `shield_flat_reduce_percent`; `heal_flat`; `mana_regen`; `blood_drain`; `convert_damage_to_heal_percent` | action/hit → self or selected target | flat shield/HP/mana; ratios | combat; `players`/combat state; conversion key chưa executor | catalog + handler/simulator (không có conversion) |
| `active.apply_status`; `status_duration_ticks`; `status_tick_damage` | after hit → selected target(s) | status id, turns, flat/tier legacy, optional probability | `statusfx` + simulator; combat state | catalog + status/simulator |
| `active.target_selector`; `range` | selection before action | enum; cells | handler/simulator; transient | catalog |
| `[parser-only] active.ignore_defense`; `ignore_defense_pct`; `armor_pen`; `crit_rate_boost`; `speed_boost_flat`; `all_stats_buff_percent`; `lifesteal_rate` | action/hit | flat or ratio (legacy names không đồng nhất) | typed model/runtime legacy; combat state | model/simulator characterization |
| `active.curve_attack`; `summon_phantom` | action → enemies/summon | flat/count | combat legacy/partial; combat state | catalog + simulator characterization |
| `[parser-only] active.summon_growth_phantoms`; `[parser-only] active.summon_count_per_tick`; `[parser-only] active.phantom_damage_scale` | growth tick → summon | bool/count/ratio | parser/model only; chưa có persistence executor | catalog only, không parity |
| `active.disable_gu_next_turn`; `forget_reality_chance`; `self_evasion_boost_percent`; `target_damage_reduce_percent` | hit → self/target | bool/probability/ratio | simulator/statusfx; combat state | forget-gu + simulator |
| `active.capture_bonus`; `portal_create`; `random_portal` | world use → self/world | ratio/bool | explore/world handlers; player/world tables | handler characterization |
| `active.cleansing_power`; `active.wood_transformation` | action → self | ratio/bool | combat legacy partial; combat state | catalog; parity chưa đầy đủ |
| `[parser-only] active.refine_mana_tier`; `[parser-only] active.replicate`; `[parser-only] active.generate_gu`; `[parser-only] active.invulnerable_ticks`; `[parser-only] active.absorb_damage_flat` | before refine/action → self hoặc generated entity | tier/bool/id/ticks/flat | model/manifest only; chưa có v2 executor/persistence command | catalog only, không parity |
| `active.reveal_all_stealth`; `active.reveal_weakness` | action/world inspect → targets/world | bool | migrate runtime v2; definition JSONB, chưa có persistence command | catalog only, không parity |
| `active.summon_monster_id`; `active.sleep`; `active.decoy_count` | action → summon/targets | entity id/bool/count | migrate runtime v2; combat/world owner chưa chốt | catalog only, không parity |
| `active.stat_base_attack`; `stat_base_defense`; `stat_base_hp`; `evasion` | always-on (đặt nhầm namespace) → self | flat/ratio | legacy player stats; `player_gu` | catalog |
| `active.description` | không trigger | text metadata, **không effect** | content only | catalog |
| `STAT_MOD ATTACK/DEFENSE/HP/SPEED` (`PASSIVE_STATS`, `ALWAYS_ON`) | always-on equipped ACTIVE → self | flat | player stats/combat snapshot; `player_gu` | player-stats + catalog |
| `STAT_MOD MOVEMENT/INVESTIGATION/HEARING/EVASION/LUCK/ACCURACY/DISGUISE/STORAGE` | always-on/capability → self | flat | player/world/inventory readers | catalog + handler characterization |
| `passive.damage_multiplier_on_hit`; `damage_mult`; `[parser-only] damage_reduction`; `tenacity`; `cc_resist` | always-on/combat event → self | ratio | combat; combat snapshot | simulator/status tests; `damage_mult` migrate |
| `STAT_MOD PATH_RESISTANCE` kèm `path` (`heaven`/`earth`/`dream`); `UNIQUE IMMUNE_DREAM_STATUS`; `forget_heaven_protection`; `forget_earth_protection` | combat passive → self | ratio/bool | combat snapshot | simulator characterization |
| `passive.success_rate_bonus`; `refine_mana_tier`; `soul_refinement`; `path_conflict_reduction` | before refinement → self | ratio/tier | refinement handlers; player/loadout; conflict key known bug | refine tests + catalog |
| `passive.time_speed_multiplier`; `dao_mark_multiplier`; `kill_path_damage_buff`; `kill_path_dao_marks_mult` | always-on/worker or reward → self | multiplier/ratio | time/reward paths; latter ba chờ migrate | catalog only |
| `passive.stat_base_luck`; `luck_flat`; `storage_slots` | always-on → self | flat/count | chưa canonical; chờ migrate | catalog only |
| `passive.cleansing_power`; `passive.description` | passive / meta | ratio; text | partial / content only | catalog |
| `growth.max_attack`; `max_defense`; `max_hp`; `max_speed`; `hours_to_max` | growth tick → Gu instance | flat cap; hours | growth worker; `player_gu` | catalog/worker characterization |
| `support`; `house`; `explore`; `refinement` | — | — | hiện rỗng toàn bộ CSV; reserved legacy roots, không được author mới | import/catalog |

Các shorthand active cũ `freeze`, `slow`, `slow1`, `weak`, `stun`, `silence`,
`bleed`, `poison`, `thorns`, `regen`, `atk_up`, `def_up`, `taunt` không còn ở
CSV nhưng vẫn là legacy keys trong parser/manifest. Chúng map sang
`apply_status`; chỉ adapter import được phép nhận chúng.

**Matrix B — `other`/`UNIQUE` action code (đủ 45 code trong CSV, cộng một hook
runtime-only được ghi rõ).** Mỗi code là
một native hook candidate; `source` là `Gu.effects.other`, UI là tên Cổ + renderer
hook. `supported` chỉ mô tả runtime hiện tại, không miễn yêu cầu registry v2.

| Trigger | Action code | Target / unit & scaling | Executor; persistence owner | Test |
|---|---|---|---|---|
| `ON_CONSUME` | `SACRIFICE_APTITUDE`, `RANK_PROGRESS`, `ADD_SOUL_MAX`, `ADD_LIFESPAN_YEARS`, `RESTORE_SOUL_HP`, `RESTORE_MENTAL`, `INCREASE_TIME_SPEED`, `REDUCE_STRESS`, `ADD_THOUGHT_MAX`, `DECREASE_TIME_SPEED` | self; typed flat/rank/year/multiplier theo params | `gu-consume`; `players`/aperture | `handlers/gu_effects_test.go` + catalog |
| `ALWAYS_ON` | `DEDUCTION_INSTANT_FINISH` | self; `base_chance + (guRank - targetRank) * per_rank_step`, clamp `[min_chance, max_chance]`; 1 roll/lần thôi diễn, Cổ rank cao nhất thắng, cost vẫn trừ | `DeduceRecipe`; `player_recipe_progress` | `gamefx/balance_test.go` + catalog |
| `COMBAT_PASSIVE` | `REVERSE_CALAMITY` | self; đảo nghịch mọi trạng thái bất lợi nguồn địch về hư vô (mất lượt, cướp Cổ, phong ấn, debuff); chỉ sát thương xuyên qua | simulator (`status_apply`, interrupt/seal gates) + `handlePreCombatSteal`; combat state | simulator + catalog |
| `ON_BATTLE_START` | `SUPPRESS_PRIORITY` | self/side; vô hiệu `SEIZE_INITIATIVE` + `FIRST_TURN_EVADE` của địch, so tốc bình thường; mirror triệt tiêu | simulator (`seizeInitiative`, evade gate); combat state | simulator + catalog |
| `ON_CONSUME` | `MAN_BEAST_BURIAL`, `POISON_VOW`, `SPRING_EGG_HATCH`, `STONE_APERTURE`, `THIRD_WATCH` | self/world; params legacy | chưa đầy đủ, migrate v2; không được silent no-op | catalog only |
| `ON_USE` | `RECORD_MESSAGE` | selected message target; text payload | support endpoint alias `RECORD`; persistent support state | handler + catalog |
| `ON_TICK` | `MANA_REGEN_TICK`, `THOUGHT_REGEN_TICK`, `SOUL_REGEN_TICK` | self; resource points/hour | resource regen worker; `players` | worker/model tests + catalog |
| `ON_TICK` | `GENERATE_STONES_TICK`, `GENERATE_IMMORTAL_STONES_TICK` | owner; currency/hour | hunger worker; wallet/player balances and Gu hunger | `worker/hunger_test.go` + catalog |
| `ON_DEATH` | `REBORN` | self; probability, ratio HP, count/turns | simulator; combat state | simulator + catalog |
| `BEFORE_ENEMY_ATTACK` | `FORGET_INTENT_INTERRUPT`, `FORGET_WILL_CONFUSE` | event attacker/targets; probabilities, turns, activation count | simulator; combat state | forget-gu tests + catalog |
| `ON_TURN_START` | `POEM_INTENT` | self/enemies; hook params | simulator; combat state | effect timing + catalog |
| `ON_BATTLE_START` | `SEIZE_INITIATIVE` | self vs primary enemy; lifespan ratio | simulator; initiative snapshot | engine + catalog |
| `ON_CAST` | `SUMMON_WOLF_FRONTLINE`, `FULL_SOUL_HEAL`, `SWORD_DRAGON_TRANSFORM`, `BORROW_PAST_POWER`, `UNRAVEL_RIDDLE`, `CHAOS_GAMBLE` | hook-declared combat targets; typed params required | simulator; combat state | engine/effect timing + catalog |
| `ON_HIT` | `BYPASS_SHIELD` | event targets; bool | simulator; hit event | engine + catalog |
| `BEFORE_HIT` (manifest/runtime-only; không có trong `other` CSV hiện tại) | `FIRST_TURN_EVADE` | self; bool/turn gate | simulator; combat state | engine + catalog |
| `PASSIVE` | `DOUBLE_DAMAGE_DEALT` | self; multiplier | simulator; combat snapshot (`COMBAT_PASSIVE` là tên v2) | engine + catalog |
| `CAPABILITY` | `MARKETPLACE_ACCESS`, `REMOTE_TRADE`, `CHAT_CONNECT`, `CHAT_PERCEIVE`, `IGNORE_MOVEMENT_RANGE`, `LOOT_PREVIEW`, `DISGUISE`, `CHANGE_FORM_DISGUISE`, `PERFECT_DISGUISE`, `ARENA_IDENTITY_MASK`, `SACRIFICE_NPC` | self/capability subject; bool/range/level theo schema hook | marketplace/trade/chat/scouting/world handlers; owner domain tương ứng | catalog + handler/scouting characterization |

**Matrix C — dictionary nhân vật, item và hai surface JSONB còn lại.**

| Nguồn / legacy key | Trigger → target | Unit / scaling | Executor hiện tại; persistence owner | UI label / test |
|---|---|---|---|---|
| Race/Constitution/Status: `stat_mods.{attack,defense,speed}` | always-on → self | ratio, cộng dồn | `gamefx.Get`; player/combat derived snapshot | `effects.ts`; `gamefx/fx_test.go`, player stats |
| Race/Constitution: `dao_mark.<path>`, `path_damage.<path>`, `path_resistance.<path>` | always-on/combat → self | ratio by path | gamefx + combat; dictionary + player selection | formatter; gamefx/combat tests |
| Shared: `mana_regen`, `max_mana`, `cultivation_efficiency`, `dao_mark_gain`, `refine`, `healing_received`, `mental_resistance`, `soul_defense`, `soul_regen`, `hp_regen`, `aperture_stress_bonus`, `tenacity`, `cc_resist` | always-on → self | ratios, trừ `hp_regen` flat HP/hour | gamefx consumers (regen/cultivate/refine/combat/tribulation); players | formatter có một phần; `gamefx/fx_test.go`, status/refine tests |
| Shared: `luck` | always-on → self | flat | gamefx/player stats | formatter + gamefx tests |
| Race: `immortal`, `disable_hp_regen`, `disable_mana_regen`, `disable_soul_regen`, `disable_resource_regen` | always-on → self | boolean | lifespan/regen policy; players.race | thiếu label cho một số flag; gamefx tests một phần |
| Status: `death_drop_mult`, `pvp_immune` | while status active → self | multiplier/bool | **không có field/executor: silent no-op hiện tại, phải fail khi compile v2** | chưa có label/test execution |
| Item: `hunger`, `restore_hp`, `heal_aperture`, `increase_human_qi`, `random_constitution`, `random_race`, `open_mine_chest` | item use/feed → self/pet/aperture | flat hoặc command enum | item/pet handlers; inventory + players/aperture | `Item.name`; `handlers/item_effects_test.go` (coverage từng key chưa đầy đủ) |
| `killer_moves.effects`: `base_damage_bonus|damage`, `base_defense_bonus|defense|add_base_defense`, `flat_damage`, `flat_defense`, `damage_multiplier|damage_boost`, `defense_multiplier|defense_boost|shield_multiplier`, `success_rate_bonus|stability_bonus|stability_penalty`, `channel_time|channeling|channeling_reduction` | move build/cast → selected enemy or self | flat, ratio, points, ticks; dao scaling `1+marks/100` | builder/manual/auto paths không đồng nhất; `killer_moves` owns snapshot | builder raw key; hầu như chưa có mapping tests |
| `killer_moves.effects`: `life_steal|lifesteal`, `multi_hit`, `phantom_attack`, `thorns_damage`, `heal`, `bleed`, `poison`, `stun`, `mana_regen`, `heal_hp|regen`, `portal_create`, `random_portal` | hit/cast/world use | percent-vs-ratio legacy, count, flat, bool | manual, auto và world chỉ xử lý các tập con khác nhau; combat/player/world persistence | raw key; simulator chỉ có characterization hạn chế |
| `aperture_variant_events.effects` | chưa có producer/trigger | chưa xác định | chỉ có JSONB owner event row; **không executor/persistence command** | không UI, không test; mọi payload hiện là no-op |

Seed fallback trong `schema_dictionaries.go` dùng cùng key với CSV và không tạo
namespace mới. Khác biệt seed/CSV phải được kiểm tra parity; đáng chú ý
`death_drop_mult` và `pvp_immune` hiện bị `encoding/json` bỏ qua.

Các field aggregate chỉ có trong `gamefx.Effects` nhưng không xuất hiện trong
snapshot năm CSV/seed hiện tại là `max_mana`, `dao_mark_gain`, `soul_defense`,
`soul_regen`, `hp_regen`, `aperture_stress_bonus`, `tenacity`, `cc_resist`;
chúng là parser/runtime surface cho DB cũ hoặc nội dung tương lai, không được
gán nhầm provenance CSV. Matrix vẫn giữ chúng để migration không bỏ sót.

### Effect Block v2 authoritative (contract phẳng)

Contract đích hiện tại là contract phẳng đang được validator, compiler và Admin triển khai. Contract giàu semantic trước đây (nested `amount`, `SELECTED_TARGET`, condition discriminated union, v.v.) **không phải** format v2 và không được dùng để tạo format thứ ba. Các khả năng mới phải được nâng dần trên contract này.

Vocabulary authoritative nằm tại `backend/internal/effectcatalog/block_manifest.go`. Endpoint `GET /api/admin/effect-catalog/metadata` sinh trực tiếp trigger, selector, condition field/operator, operation/subtype, stat, resource, unit, stack policy và native hook param schema từ catalog đó. Validator và compiler cùng đọc catalog; frontend không duy trì enum riêng.

Root chỉ có `schema_version` và `blocks`:

```json
{"schema_version":2,"blocks":[{"id":"MAIN_HIT","trigger":"ON_CAST","target":{"selector":"TARGET"},"conditions":[{"field":"TARGET_HP_PERCENT","operator":"LTE","value":0.5}],"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":120,"unit":"FLAT","path":"Sword Path"}]}]}
```

- `schema_version` bắt buộc bằng `2`. Block `id` dùng `UPPER_SNAKE_CASE`, khớp `^[A-Z][A-Z0-9_]*$` và duy nhất trong definition.
- Trigger hiện hành: `ALWAYS_ON`, `ON_ACTION`, `ON_HIT`, `BEFORE_HIT`, `ON_TURN_START`, `ON_DEATH`, `ON_TICK`, `CAPABILITY`, `COMBAT_PASSIVE`, `PASSIVE`, `ON_BATTLE_START`, `BEFORE_ENEMY_ATTACK`, `ON_CAST`, `ON_CONSUME`, `ON_USE`. `PASSIVE` được giữ để tương thích; dữ liệu mới nên dùng `ALWAYS_ON` cho stat thụ động hoặc `COMBAT_PASSIVE` cho hook trong trận.
- Target có shape `{selector,bypass_frontline?}`. Selector: `SELF`, `TARGET`, `ROW`, `COL`, `ALL_ENEMIES`, `ALL_ALLIES`, `ALL`. Không dùng alias `SELECTED_TARGET`, `COLUMN` hoặc `SINGLE`.
- Condition có đúng shape `{field,operator,value}` với `value` là number. Danh sách field/operator đóng và được trả qua metadata.
- Operation là object phẳng. `op` thuộc `DAMAGE`, `SHIELD`, `HEAL`, `DIRECT_HP`, `STAT_MOD`, `STATUS`, `BARRIER`, `CLEANSE`, `UNIQUE`; field tùy operation được metadata khai báo qua `allowed_fields`, `required_fields`, `sub_types`, `triggers`, `targets`. Tên field chuẩn là `sub_type` (không phải `subtype`), `value` number; các semantic bổ sung dùng `unit`, `resource`, `stack_policy`, `stat`, `status_id`, `turns`, `chance`, `path`, `code`, `params`.
- `unit`, `resource` và `stack_policy` được giữ xuyên suốt authoring DTO → compiled operation → runtime command. Không được tự suy diễn/chuyển đổi tỷ lệ.
- `UNIQUE` chỉ chấp nhận `code` đã đăng ký. Mỗi native hook khai báo domain, trigger và schema params (type, required, min/max hoặc enum); validator và compiler đều fail closed khi code/trigger/params không hợp lệ.
- Status ID được lấy từ `statusfx.Registry`, không có danh sách status song song.

Catalog metadata là nguồn đúng duy nhất cho editor và tooling; danh sách trên chỉ mô tả snapshot để đọc tài liệu. Mở rộng vocabulary phải sửa catalog cùng test conformance, không thêm map riêng trong compiler hoặc frontend.

### Cổng migration bắt buộc

Một legacy key/code chỉ được xóa khi: có dòng matrix; converter tạo v2 hợp lệ;
executor/native hook đã đăng ký; characterization + golden parity chạy ở manual,
auto và world/worker liên quan; persistence/UI renderer đã chuyển; audit database
không còn payload legacy. `partial`, `migrate`, `known_bug` và
`aperture_variant_events.effects` chưa dùng đều phải fail khi author v2, tuyệt đối
không được quảng bá là effect hoạt động.

> Hai bảng dưới đây đối chiếu **key effect hiện có trong game** (quét `data/Gu.csv`, 596 Cổ) với **catalog chuẩn đề xuất** (nguồn `Turnbase_Combat_Schema_Co_Gioi.xlsx`, sheet `CORE_SCHEMA` + `ALL_EFFECTS`). Cột **Trạng thái trong game** phân loại: `CÓ` (đã có data + code xử lý), `CÓ data / chưa code` (key xuất hiện trong CSV nhưng combat không xử lý), `Đổi tên` (tồn tại nhưng key khác), `Cần thêm` (chưa có). Số trong ngoặc ở cột số Cổ = số Cổ đang dùng key đó.

**Bảng A — Cấu trúc dữ liệu Effect Cổ trùng** (struct chuẩn: `models.GuEffects` bao gồm `Blocks []ComposableEffectBlock`, `Active`, `Passive`, `Growth`, `Other`).

*Khối `blocks` (Composable Effect Blocks — ECA Model, contract phẳng v2):*

`blocks` được parse, validate và compile thành representation typed khi registry tải. Trong giai đoạn migration, nhiều gameplay consumer vẫn còn đọc `active`, `passive`, `growth` và `other`; chỉ xóa namespace cũ sau khi các cổng parity/migration bên dưới đều đạt.

Dữ liệu hiện tại cũng chưa tương đương: bộ chuyển đổi prototype chỉ ánh xạ một phần active/passive/other, không ánh xạ `growth`, và còn bỏ sót các cơ chế như di chuyển, điều tra, cổng dịch chuyển, bắt quái, luyện cổ cùng nhiều hiệu ứng đặc thù. Một số dòng CSV có legacy effect nhưng không có block hữu dụng. Muốn chuyển sang block-only phải hoàn thành đồng thời executor theo trigger/op, ánh xạ toàn bộ key, chuyển đổi mọi dòng CSV, chuyển các consumer frontend/backend và có kiểm thử parity trước khi xóa schema cũ.

Cấu trúc flat v2 của mỗi block:
- `trigger`: thời điểm kích hoạt (`ALWAYS_ON`, `ON_ACTION`, `ON_HIT`, `BEFORE_HIT`, `ON_TURN_START`, `ON_DEATH`, `ON_TICK`, `CAPABILITY`, `COMBAT_PASSIVE`, `PASSIVE`, `ON_BATTLE_START`, `BEFORE_ENEMY_ATTACK`, `ON_CAST`, `ON_CONSUME`, `ON_USE`).
- `target`: mục tiêu thi triển (`selector`: `SELF`, `TARGET`, `ROW`, `COL`, `ALL_ENEMIES`, `ALL_ALLIES`, `ALL`; `bypass_frontline`: boolean).
- `conditions`: điều kiện kích hoạt bổ trợ (mảng `{field, operator, value}`).
- `operations`: mảng các thao tác nguyên tử (Atomic Operations) thuộc 9 nhóm:
  1. `DAMAGE`: sát thương phẳng (`FLAT`), hệ số (`MULTIPLIER`), lưu phái (`path`).
  2. `SHIELD`: khiên ảo Chân Nguyên Giáp (`FLAT`, `MULTIPLIER`).
  3. `HEAL`: hồi phục `FLAT`, `PERCENT`, `LIFESTEAL` hoặc `RESOURCE` (resource chọn bằng field `resource`).
  4. `DIRECT_HP`: sát thương trực tiếp `FLAT`, `MAX_HP_PERCENT` hoặc hủy khiên `SHIELD_STRIP`.
  5. `STAT_MOD`: biến đổi chỉ số nhân vật (`ATTACK`, `DEFENSE`, `SPEED`, `EVASION`, `CRIT_RATE`, `ALL`...) dạng `ADD_FLAT` hoặc `MULTIPLY_PERCENT`.
  6. `STATUS`: gắn hiệu ứng trạng thái quản lý bởi `statusfx` (`status_id`, `value`, `turns`, `chance`).
  7. `BARRIER`: bảo hộ có subtype `FLAT` hoặc `PERCENT` cùng thời lượng.
  8. `CLEANSE`: thanh tẩy `ALL`, `DEBUFF` hoặc một `STATUS` cụ thể.
  9. `UNIQUE`: cơ chế phi tuyến tính đặc thù (`code`, `params`).

*Khối `active` (44 key):*

| Key | Số Cổ | Kiểu | Tác dụng |
|---|---|---:|---|---|
| `damage_flat` | 207 | number | Sát thương cố định (tính sau modifier, trước shield/HP) |
| `damage_path` | 176 | enum | Hệ sát thương (`blood`, `Water Path`, `Sword Path`, `Fire Path`, `Ice Path`, … 20 giá trị) |
| `shield_flat` | 96 | number | Tạo khiên hộ thể (nhận damage trước HP) |
| `shield_mult` | 96 | number | Nhân khiên (×1.075 – ×1.35) |
| `apply_status` | 450 | enum / object / **mảng** | Gắn status — mảng `[{"id","value","turns"}]` cho phép nhiều status mỗi action (phân bố xem mục Status chiến đấu) |
| `status_duration_ticks` | 4 | integer | (Legacy) Số lượt status; dữ liệu mới dùng `turns` trong entry |
| `status_tick_damage` | 1 | number | (Legacy) Damage mỗi tick; dữ liệu mới dùng `value` trong entry |
| `target_selector` | 55 | enum | Quy tắc chọn mục tiêu: `COL`(38), `ALL`(17), `ROW`(1) |
| `heal_flat` | 33 | number | Hồi HP cố định (cap max_hp) |
| `damage_mult` | 18 | number | Nhân sát thương (×1.075 – ×1.35) |
| `mana_regen` | 2 | number | Hồi chân nguyên khi dùng |
| `armor_pen` | 2 | number | Xuyên Defense phẳng |
| `blood_drain` | 2 | bool | Cơ chế hút máu (+50% LifeSteal) |
| `cleansing_power` | 2 | number | Thanh tẩy (cấp độ) |
| `capture_bonus` | 4 | number | +% tỉ lệ bắt monster |
| `summon_monster_id` | 4 | string | Triệu hồi đơn vị (`MONSTER_*`) |
| `summon_phantom` | 4 | bool | Triệu hồi hư ảnh (dmg = BaseAttack) |
| `range` | 7 | number | Tầm tấn công |
| `stat_base_attack` | 10 | number | Attack (chỉ số nền của Cổ — đa phần dùng passive thay thế) |
| `stat_base_defense` | 1 | number | Defense |
| `wood_transformation` | 3 | bool | Hóa thân mộc (+30 Thủ) |
| `create_mud_zone` / `aoe_all` / `curve_attack` / `dash_distance` / `deploy_formation` / `detect_heat` / `enslave_beast_tier` / `ignite_power` / `ignore_defense` / `is_poem_immortal` / `lifesteal` / `refinement_core` / `refinement_material` / `shadow_walk` / `signal_range` / `store_gu` / `store_item` / `store_memory` / `slow` / `slow1` / `freeze` / `sleep` / `stun` | 1–3 | mixed | Cơ chế đặc thù cho 1–3 Cổ (xem struct `ActiveEffect` để biết kiểu) |
| `description` | 4 | string | Ghi chú text (không phải effect) |

*Khối `passive` (32 key):*

| Key | Số Cổ | Kiểu | Tác dụng |
|---|---|---:|---|---|
| `stat_base_attack` | 151 | number | Attack vĩnh viễn (chỉ số nền, không buff tạm) |
| `stat_base_defense` | 151 | number | Defense vĩnh viễn |
| `path_conflict_reduction` | 40 | number | Giảm xung đột Đạo |
| `movement` (STAT_MOD, 240 Cổ) | number | Phạm vi di chuyển (ô, tốc chạy map — không dùng `speed`) |
| `stat_base_hp` | 15 | number | HP vĩnh viễn (Riêng `EffectiveMaxHP` cộng qua `gamefx.PassiveHPBonus`) |
| `stat_base_speed` | 7 | number | Speed vĩnh viễn (không dùng làm Slow tạm) |
| `success_rate_bonus` | 13 | number | +% tỉ lệ thành công |
| `refine_mana_tier` | 9 | number | Bậc tinh luyện Chân Nguyên |
| `investigation` | 7 | number | Điều tra |
| `control_power` | 5 | number | Khống chế |
| `soul_refinement` | 4 | number | Tinh luyện Hồn |
| `time_speed_multiplier` | 1 | number | +`time_speed_multiplier` khi trang bị ACTIVE (cộng vào `players.time_speed_multiplier`, xử lý ở `models.EquippedTimeSpeedBonus`) |
| `luck` + `cleansing_power` + `evasion` + `storage_capacity` + `attack_speed` + `damage_multiplier` + `damage_multiplier_on_hit` + `damage_reduction` + `damage_reflect` | 1–6 | mixed | Minor (xem struct) |
| `healing_multiplier` / `heal_received_multiplier` / `cold_resistance` / `water_resistance` / `dao_mark_multiplier` / `exclude_gu_storage` / `formation_power` / `refinement_power` / `storage_capacity_gu` / `target_selector`(=ROW) | 1 | mixed | Minor (xem struct) |
| `description` | 5 | string | Ghi chú text |

*Khối `growth` (5 key, 18 Cổ):* `max_attack`, `max_defense`, `max_hp`, `hours_to_max`, `max_speed`(4) — chỉ số tăng trưởng khi nuôi.

*Khối `support` đã xóa:* heal ngoài combat dùng chung `active.heal_flat` (endpoint Support áp `active.heal_flat`, cap `EffectiveMaxHP`; combat dùng như đòn heal thường). Envelope `support:{}` rỗng vẫn parse được; key nào trong `support` đều reject `unclassified_effect_key`.

**Bảng B — Catalog chuẩn đề xuất (CORE_SCHEMA 30 key)** — dùng làm key chuẩn khi thêm Cổ mới:

| Nhóm | Key chuẩn | Kiểu | Tác dụng | Trạng thái trong game |
|---|---|---|---|---|
| A - Direct | `damage_flat` | number | Sát thương cố định | CÓ (`damage_flat`) |
| A - Direct | `damage_multiplier` | number | Sát thương theo Attack | Đổi tên (`damage_mult`) |
| A - Direct | `heal_flat` | number | Hồi HP cố định | CÓ |
| A - Direct | `shield_flat` | number | Tạo khiên | CÓ |
| A - Direct | `mana_restore` | number | Hồi chân nguyên | Đổi tên (một phần `mana_regen`) |
| A - Direct | `mana_burn` | number | Đốt chân nguyên mục tiêu | Cần thêm |
| A - Direct | `revive_hp_percent` | number | Hồi sinh | Cần thêm (trong combat; đã có `REBORN` ở `other` ngoài trận) |
| B - Status | `apply_status` | enum | Gắn status | CÓ |
| B - Status | `status_value` | number | Cường độ status | Đổi tên (đang ghép vào `status_tick_damage`) |
| B - Status | `status_duration_ticks` | integer | Số lượt | CÓ |
| B - Status | `status_stack_limit` | integer | Giới hạn cộng tầng | Cần thêm |
| B - Status | `status_tick_damage` | number | Damage mỗi tick (DOT) | CÓ (thiếu BURN) |
| B - Status | `remove_status` | enum | Thanh tẩy chính xác | Cần thêm |
| B - Status | `dispel_count` | integer | Xóa N buff/debuff | Cần thêm |
| C - Persistent | `stat_base_hp/attack/defense/speed` | number | Chỉ số vĩnh viễn | CÓ |
| C - Persistent | `damage_reduction` | decimal | Giảm damage % | CÓ |
| C - Persistent | `crit_chance` / `crit_damage` | decimal | Bạo kích | Cần thêm |
| C - Persistent | `accuracy` | decimal | Tỉ lệ đánh trúng | Cần thêm |
| C - Persistent | `evasion` | decimal | Tỉ lệ né | CÓ (`evasion`) |
| D - Turn | `cooldown_ticks` | integer | Hồi chiêu | Cần thêm |
| D - Turn | `mana_cost` | number | Chi phí dùng Cổ | CÓ (cột `base_mana_cost`) |
| D - Turn | `target_type` | enum | Quy tắc mục tiêu | Cần thêm (hiện dùng `target_selector` giới hạn) |
| D - Turn | `priority_delta` | integer | Ưu tiên lượt | Cần thêm |
| E - Condition | `condition_hp_below` | decimal | Gate theo HP | Cần thêm |
| E - Condition | `trigger` | enum | Sự kiện kích hoạt | Cần thêm (chỉ có `other.event_trigger` ngoài trận) |
| E - Condition | `chance` | decimal | Xác suất | Cần thêm (chỉ `params.chance` của `REBORN`) |

**Bảng C — Trạng thái status thực tế (đối chiếu sheet STATUS đề xuất 18 status):**

| Status | Loại | Có trong data | Code combat |
|---|---|---|---|
| SLOW | Debuff | CÓ (via `slow`) | CÓ |
| SLOW1 | Debuff | CÓ (via `slow1`) | CÓ (không kháng) |
| BLEED | DOT | CÓ (30) | CÓ (registry) |
| POISON | DOT | CÓ (2) | CÓ (registry) |
| STUN | Control | CÓ (via `stun`) | CÓ |
| FREEZE | Control | CÓ (13) | CÓ |
| SILENCE | Control | CÓ (3) | CÓ |
| THORNS / REGEN / ATK_UP / DEF_UP / TAUNT / WEAK / SLOW_DAMAGE | Buff/Debuff | Không | CÓ code (không Cổ nào dùng hiện tại) |
| **BURN** | DOT | CÓ (32) | **CÓ (registry)** |
| **CONFUSE** | Control | CÓ (4) | **CÓ (registry)** |
| **FEAR** | Control | CÓ (5) | **CÓ (registry)** |
| **STEALTH** | Special | CÓ (4) | **CÓ (registry)** |
| **SLEEP** | Control | CÓ (2) | **CÓ (registry)** |
| **BLIND / CHARM / BIND** | Stat/Control | CÓ (1) | **CÓ (registry)** |
| HASTE, ARMOR_BREAK, DAMAGE_REDUCTION_UP, INVINCIBLE, REVIVE_LOCK | Buff/Debuff | Không | **CÓ (registry)** |

**Bảng D — Mapping key cũ → key chuẩn** (dùng key chuẩn khi thêm Cổ mới):

| Key cũ | Key chuẩn đề xuất | Ghi chú |
|---|---|---|
| `damage_mult` | `damage_multiplier` | Damage × Attack |
| `mana_regen` | `mana_restore` | Khác ngữ nghĩa (hồi ngay vs regen/turn) |
| `status_tick_damage` | `status_value` | Cường độ status; giữ `status_tick_damage` chỉ cho DOT |
| `stat_base_attack` | `stat_base_attack` | Chỉ số vĩnh viễn, không dùng làm buff tạm |
| `lifesteal` | `life_steal` | Hồi HP theo damage cuối |
| `damage_reflect` | `reflect_damage` | Phản damage nhận vào |
| `slow`/`freeze`/`sleep`/… | `apply_status: [{id:"SLOW",...}]` | **Đã migrate xong** (Gu.csv + keys chết `invulnerable_ticks`→`INVINCIBLE`, `absorb_damage_flat`→`shield_flat`, `speed_boost_flat`→`HASTE`) |

### Status chiến đấu (combat)

Hệ thống status in-combat xử lý ở **simulator** (`backend/internal/simulator/` — auto-battle/manual đều dùng deterministic simulation). Status là struct `models.StatusEffect{Type, Value, Turns}` hoặc `simulator.Effect{ID,Type,Tier,Value,Duration}`; DoT/HoT là `simulator.DoTEffect`. Thứ tự tick một lượt xem `docs/STATUS_EFFECTS.md` (phần Flow xử lý mỗi lượt).

**14 constant status đã code trong simulator (`simulator/models.go`):**

| Status ID | Loại | Tier → % (TierToPercent) | Duration mặc định | Tick | Kháng |
|---|---|---|---|---|---|
| `slow` | Debuff | 15/25/35/45/55% speed | 3/3/4/4/5 | Không damage | CÓ |
| `slow1` | Debuff | 15/25/35/45/55% speed | 3/3/4/4/5 | Không damage | KHÔNG |
| `slow_damage` | Debuff | 15/25/35/45/55% speed | 3/3/4/4/5 | damage/tick | CÓ |
| `weak` | Debuff | 10/20/30/40/50% dmg | 3/3/4/4/5 | Không damage | CÓ |
| `freeze` | Control | 1/1/2/2/3 lượt | 1/1/2/2/3 | Không | CÓ |
| `silence` | Control | 1/1/2/2/3 lượt | 2/2/3/3/4 | Không | CÓ |
| `stun` | Control | 1/1/2/2/3 lượt | 1/1/2/2/3 | Không | CÓ |
| `thorns` | Buff | 10/15/20/25/30% phản | 3/3/4/4/5 | Tại hit | — |
| `regen` | Buff | 3/5/8/12/16% maxHP | 3/4/4/5/5 | heal cuối lượt | — |
| `atk_up` | Buff | +10/15/20/25/30% dmg | 3/3/4/4/5 | Không | — |
| `def_up` | Buff | +10/15/20/25/30% def | 3/3/4/4/5 | Không | — |
| `taunt` | Buff | 1/2/2/3/3 lượt | 2/2/3/3/4 | Tại chọn target | — |
| `bleed` | DOT | giá trị cố định `/tick` | từ `status_duration_ticks` | cuối lượt | Không |
| `poison` | DOT | giá trị cố định `/tick` | từ `status_duration_ticks` | cuối lượt | Không |

Lưu ý: `bleed`/`poison`/`slow_damage` dùng **giá trị số thật** (không qua tier). DoT tick **cuối lượt** người mang (`simulator/mechanics.go` `tickDoTs`); duration giảm **đầu lượt** (`tickStatusEffects`).

**Status có trong data, đã wire qua registry `statusfx`** (xem `docs/STATUS_EFFECTS.md` → "Registry status"): `BURN` (DoT hỏa), `FEAR` (force basic), `CONFUSE` (40% mis-target), `STEALTH` (untargetable), `SLEEP` (skip turn + break on damage), `BLIND` (35% miss), `CHARM` (điều khiển), `BIND` (mất lượt).

**Status catalog đã wire qua registry**: `HASTE` (tăng tốc), `ARMOR_BREAK` (giảm thủ), `DAMAGE_REDUCTION_UP` (giảm sát thương nhận), `INVINCIBLE` (miễn nhiễm sát thương), `REVIVE_LOCK` (chặn hồi sinh).

**Quy tắc gộp hiệu ứng (Consolidation)**:
- `disable_gu_next_turn` và `FORGET_EGO` được quy về trạng thái chuẩn `GU_SEAL` (`StatusGuSeal`).
- `target_damage_reduce_percent` được quy về trạng thái suy yếu sát thương `WEAK` / `FORGET_REALITY`.
- `SLOW1` được quy về trạng thái làm chậm chuẩn `SLOW` (hỗ trợ điều chỉnh độ mạnh qua value/tier).

**Tỉ lệ áp dụng hiệu ứng (Rank Scaling for Self & Enemy)**:
- **Tỉ lệ theo Rank Gu**: Mỗi cấp Rank Gu trên Rank 1 tăng $+5\%$ xác suất proc: $\text{chance}_{\text{gu}} = \text{baseChance} + 0.05 \times (\text{guRank} - 1)$ (tối đa $95\%$).
- **Hiệu ứng Bản thân (Self-Target)**: Tỉ lệ thành công tỉ lệ thuận với Cảnh giới người dùng (Player Rank): $\text{factor}_{\text{self}} = 1.0 + 0.05 \times (\text{casterPlayerRank} - 1)$.
- **Hiệu ứng Đối phương (Enemy-Target)**: Xác suất chịu ảnh hưởng của chênh lệch Cảnh giới ($\text{casterRank} - \text{targetRank}$): Chênh lệch $\ge +2$ ($\times 1.5$), $+1$ ($\times 1.2$), $0$ ($\times 1.0$), $-1$ ($\times 0.6$), $\le -2$ ($\times 0.1$). Tiên Cổ đánh Phàm Cổ áp dụng hệ số tối thiểu $1.0\times$ (tự động nâng lên $1.5\times$).

Thêm status mới: 1 entry `statusfx.Registry` + tùy chọn 1 behaviour hook `simulator/status_apply.go`.

### Item

Item dùng cho consumable, material, currency và shop. Effects hỗ trợ hồi mana/HP, soul, đạo ngân, xóa trạng thái xấu, điều kiện mortal/immortal hoặc các tác động đặc thù khác. **Quy ước `base_price`:** hỗ trợ tối đa 3 chữ số thập phân (đơn vị nhỏ nhất `0.001`); fallback theo rank — 1, 3, 6, 13, 25, 500, 1.250, 3.000, 7.500 cho rank 1–9 (rank 1 = 1, scale tỉ lệ như Cổ); item rank 0 quy về rank 1.

Các effect key đặc biệt của Consumable (resolve trong `backend/internal/handlers/item_effects.go`, map `ApplyItemEffects`):

- `random_constitution` (value `1`): khi dùng sẽ random **thể chất** mới từ `gamefx.AllConstitutions()`, loại trừ thể chất hiện tại của người chơi. Kết quả được ghi vào `players.constitution`.
- `random_race` (value `1`): khi dùng sẽ random **chủng tộc** mới từ `gamefx.AllRaces()`, loại trừ chủng tộc hiện tại. Kết quả được ghi vào `players.race`.

Hai item mẫu trong `Item.csv`: `ITEM_RANDOM_CONSTITUTION` (Hoán Thể Đan) và `ITEM_RANDOM_RACE` (Hoán Huyết Đan), cùng rank 7 / base_price 200.

### Recipe

Recipe yêu cầu Cổ instance và/hoặc material. Một số recipe cần suy diễn bằng `/api/refine/deduce` trước khi luyện ổn định.

- Cột `required_gus` là JSON map `gu_id → {rank, qty}`. `rank` là bậc Cổ nguyên liệu bắt buộc; `rank = 0` nghĩa là chấp nhận mọi bậc (chỉ dùng cho dữ liệu legacy — mọi recipe import từ CSV đều được chuẩn hóa sang rank cụ thể).
- Khi import từ CSV (`seedRecipesFromCSV`), `required_gus` được tự chuẩn hóa: value dạng số `N` cũ được chuyển thành `{rank: <base_rank>, qty: N}`, trong đó `base_rank` là bậc nhỏ nhất của Cổ đó trong `gu_dictionary` (fallback `1` nếu Cổ chưa có trong dictionary).
- Backend chỉ chấp nhận Cổ nguyên liệu khớp cả `gu_id` lẫn `rank` (khi `rank != 0`); frontend (Lò Luyện, Wiki, Admin) hiển thị và chọn Cổ theo đúng bậc yêu cầu.
- Ví dụ: `{"GU_WHITE_BOAR": {"qty": 1, "rank": 1}}`.

### Monster/combat

Monster template chứa chỉ số, Gu inventory, item inventory, dao marks, path masteries, AI profile và loot table. Combat handler dùng dữ liệu này để khởi tạo trận, chọn hành động và roll loot.

### Region / Bản đồ thế giới (World Map)

Từ phiên bản map 3×2, thế giới là **một bản đồ liên tục duy nhất** 30.000 × 20.000 ô, chia làm **6 Đại Vực (NGU_VUC)** mỗi cái 10.000 × 10.000 (lưới 3 cột × 2 hàng). Mỗi Đại Vực được chia số học thành **tiểu vực** 1.000 × 1.000 (600 tiểu vực toàn thế giới). Các địa danh có tên (KHU_VUC — giữ nguyên 7 Lưu Vực cũ) nằm lồng/chồng bên trong Đại Vực. Ngoài `Southern_Border` (HUB mặc định), mỗi Đại Vực còn có **HUB khởi đầu riêng** (`hub_thuong_gia_thanh`, `hub_trung_chau`, `hub_bac_nguyen`, `hub_tay_mac`, `hub_dong_hai`). Khi hoàn tất onboarding (`complete-onboarding`), người chơi được đặt vào HUB tương ứng với vị trí đăng ký (xem `startingRegionHubs` trong `player.go`), không sinh quái trong HUB.

**Bảng `regions`** — các cột mới:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `parent_id` | VARCHAR | zone cha (KHU_VUC trỏ tới NGU_VUC của nó). |
| `zone_type` | VARCHAR | `HUB` / `SAFE_ZONE` / `BOSS_ZONE` / `SECRET_REALM` / `DUNGEON` / `PVP_ZONE` / `NGU_VUC` / `KHU_VUC`. |
| `min_x, min_y, max_x, max_y` | INT | hình chữ nhật toàn cầu (inclusive) của zone. |
| `entry_x, entry_y` | INT | tọa độ cổng truyền tống khi du hành tới zone. |
| `safe_zone_x, safe_zone_y` | INT | tọa độ điểm an toàn trong khu vực (được dùng cho dịch chuyển tức thời). |
| `pool_weights` | JSONB | pool sinh nội dung procedural của zone (xem dưới). |

**`pool_weights`** JSONB — ví dụ:
```json
{
  "cell_type_weights": { "Monster": 22, "Resource": 14, "WildGu": 5, "Empty": 59 },
  "resources": ["moon_orchid_petal", "jade_stone"],
  "wild_gus": ["GU_LITTLE_LIGHT", "GU_JADE_SKIN"]
}
```
`cell_type_weights` quyết định tỉ lệ các loại ô; `resources` / `wild_gus` là pool tài nguyên và Cổ cấp thấp. Không khai báo → dùng pool mặc định.

**Thứ tự ưu tiên zone_type** — khi một điểm nằm trong nhiều zone chồng lấn, `FindZoneAt` chọn zone có priority cao nhất (bằng → diện tích nhỏ nhất = innermost):

```
HUB > SAFE_ZONE > BOSS_ZONE > SECRET_REALM > DUNGEON > PVP_ZONE > KHU_VUC > NGU_VUC
```

Zone type không nhận diện fallback về priority của `KHU_VUC`. `FindLargeZoneAt` luôn trả `NGU_VUC` chứa điểm.

**Bảng `map_cells`** — các ô **dùng chung toàn thế giới** (chỉ lưu boss + tài nguyên cấp cao + điểm an toàn). `UNIQUE(x, y)` toàn cầu. Cột mới: `last_cleared_at TIMESTAMPTZ`, `cleared_by UUID`. Khi giết boss / hái tài nguyên cao, `last_cleared_at` được set; ô sẽ trống cho tới khi hết `respawn_seconds`.

**Bảng `monster_spawns`** — giờ là **pool theo zone** (không còn ý nghĩa toạ độ cho loại Roaming/Random). `region_id` trỏ tới Đại Vực hoặc địa danh. Pool được cache trong bộ nhớ lúc start.

**Phân lớp sinh nội dung**:
- Quái nhỏ / tài nguyên thấp / Cổ cấp thấp → sinh **procedural** bằng hash seed theo `(zone, x, y, window 180s)`, không đụng DB.
- Boss / tài nguyên cấp cao / làng → ô **persistent** trong `map_cells`, nạp vào index bộ nhớ lúc start, có respawn dùng chung.

Import/Export qua Admin tab "Dữ liệu": bảng `region` (Region.csv), `map_cell` (MapCell.csv), `monster_spawn` (MonsterSpawn.csv). Sau khi import **restart backend** để nạp ZoneRegistry.

## Quy ước cập nhật dữ liệu

1. Sửa CSV khi thay đổi dictionary/seed gốc.
2. Kiểm tra schema seed tương ứng trong `backend/internal/db/db.go` nếu thêm cột mới.
3. Nếu thêm field Cổ/item mới, cập nhật struct model và frontend type trước khi dùng trong UI.
4. Cập nhật số dòng trong bảng trên nếu thêm/xóa seed data.

### Bảng `leaderboard_settings` (Bảng Xếp Hạng)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `player_id` | UUID PK | FK → `players(id)` ON DELETE CASCADE |
| `show_on_leaderboard` | BOOLEAN | DEFAULT TRUE. Người chơi có thể tắt để rút khỏi BXH |

Không có seed CSV cho bảng này. Row được tạo tự động qua `INSERT ON CONFLICT DO NOTHING` khi người chơi truy cập BXH lần đầu.

### Bảng `sects` (Tông môn)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `funds` | BIGINT | Quỹ tông môn (đổi từ INTEGER → BIGINT để tránh tràn số khi tích lũy lãnh địa). |
| `require_approval` | BOOLEAN | DEFAULT FALSE. Bật = phải duyệt đơn gia nhập. |
| `vault_gu` | JSONB | *(Legacy)* Đã chuyển sang bảng quan hệ `sect_vault_gu`. |
| `vault_materials` | JSONB | *(Legacy)* Đã chuyển sang bảng quan hệ `sect_vault_materials`. |
| `vault_recipes` | JSONB | *(Legacy)* Đã chuyển sang bảng quan hệ `sect_vault_recipes`. |
| `last_vault_decay_at` | TIMESTAMPTZ | Cột lịch sử (không còn dùng do đã bỏ cơ chế tính đói kho tông môn). |

### Bảng `sect_vault_materials` (Kho Linh tài Tông môn)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `sect_id` | VARCHAR(50) PK FK | → `sects(id)` ON DELETE CASCADE |
| `item_id` | VARCHAR(100) PK | ID vật phẩm/linh tài |
| `quantity` | BIGINT | Số lượng lưu trữ, `CHECK (quantity >= 0)` |
| `created_at` | TIMESTAMPTZ | Thời điểm thêm vào |
| `updated_at` | TIMESTAMPTZ | Thời điểm cập nhật cuối |

### Bảng `sect_vault_gu` (Bảo Khố Cổ Trùng Tông môn)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | VARCHAR(80) PK | Định danh duy nhất dòng (`sect_id:gu_id:rank` hoặc `sect_id:instance_id`) |
| `sect_id` | VARCHAR(50) FK | → `sects(id)` ON DELETE CASCADE |
| `gu_id` | VARCHAR(50) | ID Cổ trùng |
| `rank` | INTEGER | Phẩm cấp Cổ (1-9) |
| `quantity` | BIGINT | Số lượng tích lũy |
| `cost` | BIGINT | Giá điểm cống hiến để đổi |
| `hunger` | DOUBLE PRECISION | Độ no lưu trữ |
| `instance_id` | VARCHAR(80) | Instance ID gốc (đối với Tiên Cổ rank ≥ 6) |

### Bảng `sect_vault_recipes` (Tàng Kinh Các / Bí Phương Tông môn)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `sect_id` | VARCHAR(50) PK FK | → `sects(id)` ON DELETE CASCADE |
| `recipe_id` | VARCHAR(100) PK | ID Bí phương luyện Cổ |
| `completion_rate` | FLOAT | Tiến độ nghiên cứu / hoàn thiện (100.0) |
| `unlocked_at` | TIMESTAMPTZ | Thời điểm mở khóa |


Các cột được thêm sau khi tạo bảng bằng `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` trong `db.go`.

### Bảng `sect_missions` (Nhiệm vụ tông môn — Bảng nhiệm vụ)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | UUID PK | |
| `sect_id` | VARCHAR(50) FK | → `sects(id)` ON DELETE CASCADE |
| `title` | VARCHAR(100) | |
| `description` | TEXT | |
| `mission_type` | VARCHAR(30) | `GATHER`, `DONATE`, `COMBAT`, `REFINE` |
| `target_id` | VARCHAR(50) | ID mục tiêu (linh tài / Cổ) |
| `target_amount` | INTEGER | Số lượng yêu cầu |
| `reward_contribution` | INTEGER | Điểm cống hiến thưởng |
| `required_member_rank` | VARCHAR(30) | Cấp bậc tối thiểu để nhận |
| `expires_at` | TIMESTAMP NULL | Thời điểm tự hết hạn. `NULL` = không hết hạn (mặc định cho nhiệm vụ seed từ CSV). Khi tạo qua UI: mặc định `now()+24h`, nhập `-1` = vĩnh viễn, nhập `0` = dùng mặc định 24h. Worker (`sect_mission.go`) tự xóa nhiệm vụ hết hạn mỗi phút; hàng `player_sect_missions` liên quan bị xóa theo CASCADE. |

Seed từ `SectMission.csv` (append-only, `ON CONFLICT DO NOTHING`) luôn để `expires_at = NULL`.

### Bảng `player_sect_missions` (Nhiệm vụ cá nhân đang thực hiện)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | UUID PK | |
| `player_id` | UUID FK | → `players(id)` ON DELETE CASCADE |
| `mission_id` | UUID FK | → `sect_missions(id)` ON DELETE CASCADE |
| `current_progress` | INTEGER | |
| `status` | VARCHAR(30) | `IN_PROGRESS`, `COMPLETED`, `EXPIRED` |
| `assigned_at` | TIMESTAMP | |
| `expires_at` | TIMESTAMP NULL | Sao chép từ bảng nhiệm vụ khi nhận. Worker đổi `IN_PROGRESS` → `EXPIRED` khi quá hạn. |
| UNIQUE | (player_id, mission_id) | Mỗi người chơi chỉ nhận 1 lần mỗi nhiệm vụ |

### Bảng `sect_mine_defenders` (Phòng thủ Lãnh địa)

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | UUID PK | |
| `mine_id` | INTEGER FK | → `sect_mines(id)` ON DELETE CASCADE |
| `sect_id` | VARCHAR(50) FK | → `sects(id)` ON DELETE CASCADE |
| `defender_type` | VARCHAR(20) | `member` (đệ tử) hoặc `beast` (thần thú thuê) |
| `player_id` | UUID NULL | Đệ tử phòng thủ. UNIQUE (partial) — mỗi đệ tử chỉ giữ 1 mỏ |
| `name` | VARCHAR(100) | Tên hiển thị (snapshot) |
| `monster_id` / `monster_rank` / `monster_sub_rank` | VARCHAR(80)/INT | Dữ liệu thần thú |
| `power` | DOUBLE PRECISION | Điểm phòng thủ (tính lúc gán) |
| `assigned_at` | TIMESTAMP | |

`power` đệ tử = `(atk + def + hp/20) × (1 + 0.15×(rank−1))`; thần thú = `atk + def + hp/20` (qua `ScaleMonsterStats`).

### Bảng `sect_mines` — cột tích lũy

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `pending_funds` | DOUBLE PRECISION | Bộ tích lũy Quỹ (worker cộng theo phút, đổi sang số nguyên khi ≥ 1) |
| `pending_materials` | DOUBLE PRECISION | Bộ tích lũy Linh tài |

### Bảng `sect_members` — cột cống hiến lãnh địa

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `territory_pending` | DOUBLE PRECISION | Bộ tích lũy điểm cống hiến theo giờ |
| `territory_credited_at` | DATE | Ngày đã cộng (để reset cap) |
| `territory_credited_today` | INTEGER | Số điểm đã cộng hôm nay (cap 200/ngày) |

## Lưu ý khi thay đổi schema

- Schema được tách theo domain ở `backend/internal/db/schema_*.go`, mỗi bảng là `CREATE TABLE IF NOT EXISTS` với toàn bộ cột inline. `InitSchema()` trong `db.go` gọi các schema function theo thứ tự FK. Không có migration riêng; cột mới phải có `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` liền sau schema để patch database hiện có.
- Thêm/sửa bảng: sửa file `schema_*.go` tương ứng → xóa DB (hoặc `DROP TABLE` thủ công) → restart server.
- FK cho các cột `players.race` / `players.constitution` / `players.current_region`: các dòng sentinel (`''`) và vùng `Southern_Border` được pre-seed trong `InitSchema` nên đăng ký vẫn hoạt động trước khi import CSV. Các HUB khởi đầu theo Đại Vực (`hub_*`) nằm trong `Region.csv` và được import qua Admin tab "Dữ liệu"; nếu chưa import, `complete-onboarding` fallback về `Southern_Border`. Các bảng dictionary (`race_dictionary`, `constitution_dictionary`) tự seed default dataset khi DB rỗng; CSV import (Admin tab "Dữ liệu") vẫn là nguồn chính thức.

### Bảng `player_pets` (Sủng vật)

Bảng lưu sủng vật bắt được từ quái vật trong combat.

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `id` | UUID PK | |
| `player_id` | UUID FK | → `players(id)` ON DELETE CASCADE |
| `monster_template_id` | VARCHAR(80) FK | → `monster_templates(id)` |
| `monster_name` | VARCHAR(100) | Tên quái tại thời điểm bắt |
| `rank` | INTEGER | Rank của quái vật |
| `current_hp` | DOUBLE PRECISION | HP hiện tại |
| `max_hp` | DOUBLE PRECISION | HP tối đa |
| `attack` | DOUBLE PRECISION | Tấn công |
| `defense` | DOUBLE PRECISION | Phòng thủ |
| `hunger` | DOUBLE PRECISION | Độ no (0-100) |
| `status` | VARCHAR(20) | `active` hoặc `sleeping` |
| `formation_slot` | INTEGER | 0 = storage, 1..N = formation slot |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | Dùng cho hunger decay theo elapsed time |

Indexes: `idx_player_pets_player_id`, `idx_player_pets_player_id_template`.

### Cột bắt sủng vật trong `monster_templates`

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `is_capturable` | BOOLEAN | DEFAULT FALSE. Có thể bắt làm sủng vật không |
| `capture_chance` | DOUBLE PRECISION | Tỷ lệ bắt cơ bản (0.0-1.0). rank 1 = 0.25, rank 5 = 0.10 |
| `pet_hunger_per_hour` | DOUBLE PRECISION | Tốc độ đói mỗi giờ. Mặc định 2 |

### Gộp sủng vật (bầy) và phí Hồn Phách khi ra trận

- **Gộp bầy** qua `/api/pets/merge` (`MergePets`): chỉ gộp pet cùng `monster_template_id`, nguồn phải ở kho (`formation_slot = 0`).
- **Giới hạn bầy** = `max(1, max_soul / 10)` (`mergeSoulDivisor = 10` trong `pets.go`). Nếu `count` sau khi gộp vượt giới hạn → **chặn toàn bộ gộp** (không xóa nguồn) và trả `merge_limit`.
- **Công thức chỉ số** (sub-linear, không cộng dồn compound): lấy base `stat / count^0.65` rồi scale `base × newCount^0.65` cho HP/Atk/Def; `speed` giữ nguyên; gộp hồi đầy HP. Hệ số `packScaleExponent = 0.65`.
- **Phí Hồn Phách khi ra trận:** mỗi pet (bầy) tiêu tốn `soul_hp` = `count` (số cá thể) khi load vào trận thật (`combat_state.go`). Nếu `soul_hp < count` → **bỏ qua pet đó**. Trừ dần theo `formation_slot ASC`.
  - Áp dụng cho cả pet của **đối phương phòng thủ** khi bị tấn công PvP/mine/map (trừ `soul_hp` của họ, thiếu thì bỏ pet của họ).
  - **Võ Đài (arena, virtual) không tốn soul.**
  - Cờ `active_combat.pet_soul_charged` đảm bảo mỗi trận chỉ tính phí một lần (tránh re-init giữa trận tính lại).

### Cho sủng vật ăn (thức ăn loại Food)
- Sủng vật được cho ăn qua `/api/pets/feed` (handler `FeedPet`).
- Độ no cộng thêm lấy từ `item_dictionary.effects.hunger` (data-driven).
- **Quy ước dữ liệu:** mọi item loại `Food` có `hunger = rank × 10` (rank 1 = +10 ... rank 9 = +90), cập nhật qua `backend/cmd/server/data/Item.csv` → Import bảng "Vật phẩm". Backend fallback cùng công thức `rank × 10` nếu effects chưa có `hunger`.
- Item được tiêu hao từ bucket `consumables` hoặc `materials` trong `player_inventory_items`.

### Giá tiền theo rank (Thương Đội)

Bảng `caravan_shop_items` lưu `price` (số lượng, không lưu loại tiền) và có khóa `(id, rank)`. Cột `rank` quy ước:
- **`rank = 0`** (mặc định) → tự suy rank: item dùng `item_dictionary.rank`, Cổ trùng dùng variant rank **thấp nhất** (`gu_dictionary.rank`, đồng nhất `models.GetGuAny`), bí phương dùng `refining_recipes.target_rank`.
- **`rank > 0`** → bán **đúng rank đó** của Cổ trùng (cho phép 1 Cổ đa rank xuất hiện nhiều lần trong shop với giá khác nhau), đồng thời giữ hành vi cũ cho mọi dòng `rank = 0`.

Loại tiền được suy ra từ **rank hiệu dụng** qua `pkg/pricing.StoneTierByRank`:

- **rank < 6** → Nguyên Thạch (`primeval_stones`)
- **rank ≥ 6** → Tiên Nguyên Thạch (`immortal_stones`)

- `/api/shop/list` trả thêm `rank` + `currency` mỗi món.
- `/api/shop/buy` nhận thêm `rank` (0 = auto); trừ đúng balance theo `currency`.
- `/api/shop/sell` và `/api/shop/sell-bulk` nhận thêm `rank` và trả về đúng loại tiền theo rank món bán (bí phương dùng `target_rank`).

**Hàng mặc định + xoay vòng ngẫu nhiên theo ngày:** món `is_default = true` luôn hiển thị. Món `is_default = false` được shuffle deterministic theo ngày (seed = `YYYYMMDD` UTC) theo **tỷ lệ cân bằng ~40 Cổ / ~60 item** (`maxDailyRandomGus`/`maxDailyRandomItems`; cache Redis key theo ngày).

**Pool ngẫu nhiên tự sinh (virtual):** ngoài các món `is_default = false` trong `caravan_shop_items`, `autoShopPool()` (handlers/shop.go) tự bổ sung vào pool ngẫu nhiên mọi **Cổ trùng rank < 5** (mỗi variant pinned đúng rank) cùng **thức ăn của chúng** (các item trong `gu_dictionary.food_types`, trừ `primeval_stone`/`immortal_stone`), trừ các id đã có trong `caravan_shop_items`. Các món này **không được persist** vào bảng shop nên không làm bẩn danh sách admin hay CSV export.

**Giá + số lượng (stock) món random:** giá mua = `base_price × 1.5` (`randomPoolMarkup`, làm tròn đến `0.001`, tối thiểu `0.001`). Mỗi món có stock hằng ngày theo rank qua `randomPoolStock(rank)`: **R1 = 100, R2 = 50, R3 = 20, R4 = 5, R5 = 2, R6+ = 1**. Số còn lại trong ngày lưu ở bảng `caravan_shop_daily_stock(day, id, rank, remaining)` — key theo ngày nên tự reset mỗi ngày. `/api/shop/buy` trừ stock nguyên tử (`UPDATE ... WHERE remaining >= qty`), hết thì trả lỗi "Hàng này hôm nay đã hết". Món default/admin trong `caravan_shop_items` **không** áp dụng cơ chế này (stock CSV của chúng vẫn chỉ hiển thị).

**`base_price` rank ≥ 6 là Tiên Nguyên Thạch** — bảng giá mặc định (fallback trong `pkg/pricing/pricing.go`, dùng khi CSV không ghi `base_price`):

| Rank | 6 | 7 | 8 | 9 |
| --- | ---: | ---: | ---: | ---: |
| Cổ trùng | 200 | 500 | 1.200 | 2.500 |
| Vật phẩm | 10 | 25 | 60 | 150 |

Ví dụ bán-lại cho Thương Đội (10% `base_price`): Cổ rank 9 = 250 Tiên Nguyên Thạch; item rank 6 = 1 Tiên Nguyên Thạch.

### Đổi đá (exchange)

Đổi giữa hai loại đá qua `POST /api/shop/exchange` (xem `ExchangeStones` trong `handlers/premium.go`, tỷ giá hằng số trong file):

- **1 Tiên Nguyên Thạch = 100 Nguyên Thạch** — đổi chiều `IMMORTAL_ESSENCE_STONE` → `PRIMEVAL_STONE` (`amount * 100`).
- **100.000 Nguyên Thạch = 1 Tiên Nguyên Thạch** — đổi chiều `PRIMEVAL_STONE` → `IMMORTAL_ESSENCE_STONE` (`amount` phải là bội số của 100.000, nhận `amount / 100000`).

Giao dịch đồng bộ cả `players.primeval_stones/immortal_stones` lẫn `wallets.primeval_stone/immortal_essence_stone`, ghi 2 dòng sổ cái `wallet_transactions` với `type = 'EXCHANGE'` (thêm vào CHECK constraint của bảng `wallet_transactions`; `InitSchema` tự `ALTER TABLE ... DROP/ADD CONSTRAINT` để áp dụng lên DB cũ).

### Truyền Tống Trận Ngũ Vực (Teleportation Arrays)

Bảng `teleport_arrays` lưu các thực thể trận pháp do người chơi và Tông Môn xây dựng trên bản đồ thế giới Ngũ Vực.

#### Phân cấp trận pháp:
- **`TIER_1` (Tiểu Hình Truyền Tống Trận / Vi Trận):**
  - Tu vi tối thiểu: Chuyển 2 (Rank 2).
  - Chi phí: 2.000 Nguyên Thạch + 2x Không Gian Thạch (`khong_gian_thach`).
  - Phạm vi cự ly: Tối đa 1.000 ô trong cùng Đại Vực.
  - Độ bền tối đa: 50 điểm (mỗi lượt nhảy giảm 1 điểm).
- **`TIER_2` (Đại Vực Truyền Tống Trận):**
  - Tu vi tối thiểu: Chuyển 4 (Rank 4).
  - Chi phí: 10.000 Nguyên Thạch + 10x Không Gian Thạch (`khong_gian_thach`).
  - Phạm vi cự ly: Tối đa 10.000 ô trong cùng Đại Vực.
  - Độ bền tối đa: 100 điểm.
- **`TIER_3` (Ngũ Vực Tiên Cấp Truyền Tống Trận):**
  - Tu vi tối thiểu: Chuyển 6 (Rank 6 - Tiên Nhân).
  - Chi phí: 50.000 Nguyên Thạch + 20 Tiên Nguyên Thạch + 50x Không Gian Thạch.
  - Phạm vi: Tự do dịch chuyển xuyên tất cả các Đại Vực trong Ngũ Vực (`is_cross_region = true`).
  - Độ bền tối đa: 200 điểm.
  - Phản phệ Giới Vực Bích: Khi nhảy xuyên Đại Vực, người chơi vẫn phải chịu debuff áp chế thuộc tính theo Rank từ `CheckAndApplyRegionalWallDebuff`.

#### Quy tắc kinh tế & vận hành:
- **Chế độ kết nối:** Hai chiều (`TWO_WAY`), Cổng gửi đi (`ONE_WAY_SEND`), Cổng nhận (`ONE_WAY_RECEIVE`).
- **Quyền truy cập:** `PUBLIC` (Toàn server), `SECT` (Thành viên Tông Môn), `PERSONAL` (Chỉ chủ sở hữu).
- **Phí dịch chuyển & Két sắt:** Chủ sở hữu cấu hình bật/tắt thu phí và mức giá vé. Khi có người sử dụng, 95% phí được tích lũy vào Két Sắt Trận Pháp (`vault_balance`) để chủ sở hữu đến rút, 5% hao tán theo quy luật trời đất.
- **Bảo trì & Thu hồi:** Tiêu hao Nguyên Thạch để sửa chữa hồi phục độ bền. Khi thu hồi (Dismantle), hoàn trả 50% chi phí xây dựng ban đầu (nhân theo tỉ lệ % độ bền còn lại) và tự động chuyển toàn bộ số dư Két Sắt về túi nhân vật.

### Trạng thái công trình Tông Môn

`sects.defensive_array_ready_at` lưu thời điểm Hộ Sơn Đại Trận có thể chặn đợt
công kích đầu tiên tiếp theo. Khi kích hoạt, `shield_until` chỉ mở cửa sổ tập hợp
10 phút; thời gian hồi chặn 12 giờ được theo dõi độc lập và không tăng theo cấp.
`sects.buildings.treasury` tiếp tục là nguồn dữ liệu cấp Bảo Khố dùng để tính tỉ lệ
rơi tài sản và chiết khấu phí lưu kho khi duy trì/di dời.

### Compiled effect runtime

`effects.blocks` is authored as JSON but is compiled once when a Gu definition is
loaded. The immutable runtime representation lives in
`backend/internal/gameplay/effects`; gameplay loops must query that representation
instead of parsing JSON or dispatching unregistered strings. The compiler validates
the authoritative trigger, selector, condition, operation, stat/resource, and native
mechanic vocabularies. Status IDs and semantics remain authoritative in `statusfx`;
a `STATUS` operation becomes a `statusfx.StatusOp` with an explicit value unit,
chance, and turn duration.

Effect evaluation is pure and returns domain commands. Handlers/workers remain the
orchestrators and must apply any database-changing command through their existing
transaction. Runtime execution contexts carry deterministic RNG, actor/target
snapshots, rank, mastery, Dao marks, paths, and catalog revision, and enforce event,
trigger-depth, repeat, and summon budgets. During migration, compiled combat
operations are attached to `simulator.SimAction` alongside its legacy fields.

## Admin audit log fields

`audit_logs` stores immutable admin mutation context in dedicated columns:
`actor_username`, `effective_roles`, `effective_permission`, `entity_type`, `entity_id`,
`change_data`, `reason`, and `request_id`, in addition to actor `player_id`, IP, outcome,
sanitized error, and `created_at`. Existing databases receive these columns through
idempotent `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` statements during schema startup.
Sensitive request fields and multipart bodies are excluded from `change_data`.
