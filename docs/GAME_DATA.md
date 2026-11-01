# Game Data

## MMO player location

The `players` row owns durable location. `current_region` is the logical
gameplay region; `world_zone_id`, `world_instance_id`, `world_x/world_y`,
`world_safe_x/world_safe_y`, and `world_location_revision` hold the realtime
map location and its compare-and-swap revision. Integer `players.x/y` remain a
legacy projection for older gameplay handlers. `player_world_state_v2` stores
only HP/shield and map/content versions. Existing databases require the
explicit maintenance script `backend/scripts/migrate_player_world_location.sql`;
it is not run during startup. See `docs/mmo/recovery.md`.

## Realtime skill loadout

`players.realtime_skill_loadout` is an ordered JSONB array (max 4) of
`{"kind":"move"|"house","id":"..."}` entries and
`players.realtime_skill_loadout_configured` distinguishes "never saved" (the
server backfills slots from the equipped pool) from an explicit order. It binds
hotbar slots 9–12, shared by equipped player killer moves
(`player_killer_moves.is_equipped`) and the Trận Kích actions of the equipped
`player_gu_houses` row. Resolution and slot binding live in
`handlers/skill_loadout.go`; entries whose source is gone (move deleted or
unequipped, house unequipped) are dropped and later entries compact upward,
while resolvable-but-unsupported entries keep their slot as an empty gap so the
remaining keys do not shift. `PUT /api/combat/skill-loadout` rejects entries
that are not equipped or not realtime-supported at save time. Trận Kích
consumption drains the house core Gu hunger but does not accrue killer-move
mastery XP.

## Daily missions

Daily mission progress is stored in `player_daily_missions`; `player_daily_mission_events`
deduplicates server-confirmed activities, and `player_daily_mission_rank_snapshots`
locks each player's rank at the 00:00 `Asia/Ho_Chi_Minh` rollover. The rollover
worker expires prior-day rows and snapshots current ranks. Mission definitions
and rank-eligible pools currently live in `handlers/daily_missions.go`.

`player_achievements` is the authoritative ownership table. Schema initialization
backfills legacy `players.earned_achievements`; handlers continue dual-writing the
JSONB field during the compatibility period. Admin revocation clears both records
and any matching equipped title/frame, while leaving previously claimed rewards.

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

`cmd/migrate_csv` only normalizes **already-authored** blocks: for `Gu.csv` it
returns `[]` when the raw value is neither a block array nor an object carrying
`blocks`. Legacy envelopes (`{"active":{...},"passive":{...}}`) therefore lost
their effects during conversion. 191 of the 209 historically-empty rows were
restored using the project's own `effectmigration.ConvertGu` mapping
(`passive.*`→`STAT_MOD ALWAYS_ON`, `active.capture_bonus`/portals→`CAPABILITY`,
`growth.hours_to_max`→`GROWTH_DURATION_HOURS`) plus hand mappings for keys with
no exact mapping: `passive.soul_refinement`→`STAT_MOD SOUL_REFINEMENT`,
`passive.accuracy` (and `legacy_passive_stats` blocks with stat `ACCURACY`)→
`STAT_MOD INVESTIGATION`, `active.cleansing_power`→`STAT_MOD CLEANSING_POWER`,
`active.sleep`→`STATUS SLEEP` (same as legacy `normalize()`),
`passive.kill_path_damage_buff`→`UNIQUE PATH_DAMAGE_MODIFIER`,
`passive.kill_path_dao_marks_mult`→`UNIQUE DAO_MARK_MODIFIER`, and
`active.range`→`UNIQUE RECORD_MESSAGE` (`ON_USE`, `params.range`).
6 rows stay `[]` on purpose: they carry only `description` metadata (Blood Path
materials, Huyết Cửu), which is not an effect. Every row is re-checked with
`effectcatalog.ValidateBlockV2` before import.

> **Lịch sử simulator turn-based:** phần ví dụ `SUMMON_MONSTER` dưới đây ghi lại
> contract cũ. Backend simulator đã bị gỡ; summon từ block này chưa có executor
> realtime và không tạo minion trong chiến đấu hiện hành.

**Triệu hồi monster (`SUMMON_MONSTER`) — contract cũ.** Dữ liệu cũ từng khai báo
hook `ON_BATTLE_START`/domain `combat` như sau:

```json
[{"id": "SUMMON_MONSTER", "target": {"selector": "SELF"}, "trigger": "ON_BATTLE_START",
  "operations": [{"op": "UNIQUE", "code": "SUMMON_MONSTER",
  "params": {"monster_id": "monster_wolf_leader", "hp_scale": 0.5, "damage_scale": 0.33, "position": "frontline"}}]}]
```

Các công thức rank scale, clamp và giới hạn minion từng thuộc simulator đã gỡ;
chúng không phải hành vi runtime hiện tại. Cần profile/executor realtime rõ ràng
trước khi effect này được mở lại.

**Thú triệu hồi scale theo đạo ngân.** Sau clamp, `hp`/`damage`/`defense` của
minion nhân thêm `daoMult` của path nguồn (`actorDaoMultiplier`): với hook
`ON_CAST` là path của action, với hook `ON_BATTLE_START` là `Path` gắn trên
`CombatMechanic` (`combatMechanicsForSpec` stamp từ Cổ gốc) — cùng hệ số mà
damage/shield/heal của Cổ đó dùng. Áp **sau** clamp để clamp anti-abuse không ăn
hết phần khuếch đại khi đạo ngân cao.
`SUMMON_WOLF_FRONTLINE` cũng nhân cùng `daoMult` vào `MaxHP`/`BaseDmg`/`BaseDef`.
Log `SUMMONED` in thêm `Đạo Ngân ×N` khi `daoMult > 1`.

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

### Chỉ mục vận hành

`players` có chỉ mục `(created_at DESC, id DESC)` phục vụ phân trang mặc định
trong danh sách người chơi của Admin và chỉ mục bán phần `google_id IS NOT NULL`
phục vụ tra cứu tài khoản trong callback Google OAuth. `notifications` có chỉ mục riêng cho thông
báo chưa đọc, còn `player_recipe_progress` dùng khóa chính
`(player_id, recipe_id)`. Các truy vấn tương ứng timeout đồng thời thường là dấu
hiệu nghẽn pool, tải database hoặc khóa schema, không nên xử lý bằng cách chỉ tăng
`statement_timeout`.

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
| `backend/data/Monster.csv`, `LootTable.csv`, `MonsterSpawn.csv`, `RegionLootTables.csv` | Template quái, bảng loot, quy tắc spawn và loot theo vùng. |
| `backend/data/Region.csv`, `MapCell.csv` | Vùng/zone và ô persistent của thế giới. |
| `backend/data/Race.csv`, `Constitution.csv`, `Status.csv` | Chủng tộc, thể chất và trạng thái data-driven. |
| `backend/data/Quest.csv`, `DreamRealm.csv` | Nhiệm vụ và mộng cảnh/phó bản. |
| `backend/data/Sect.csv`, `SectMission.csv` | Môn phái và mẫu nhiệm vụ môn phái. |
| `backend/data/ShopItem.csv`, `ArenaShopItem.csv` | Shop thương đội và shop Võ Đài. |
| `backend/data/PremiumShopPackage.csv`, `TopupPackage.csv` | Gói premium và gói nạp; được handler đọc trực tiếp, không import qua Admin. Kết quả có Redis cache 24 giờ. |
| `backend/data/ApertureNodeTemplate.csv` | Template công trình Tiên Khiếu (chỉ DEFENSE hiện được dựng: Tường, Tháp Canh, Tượng Hộ Mệnh, Doanh Trại, Trường Lực): cost, HP, defense, attack range, cost Cổ. Đọc trực tiếp lúc khởi động; sửa CSV cần restart. |
| `backend/data/ResourcePointTemplate.csv` | Template Điểm Tài Nguyên (canon resource points, ch. 1303): `point_type, name, path, produce_key, produce_interval_hours, base_yield, vitality_drain, vitality_regen_hours, is_food, worker_race, rare_rate, gu_rate, point_class`. `worker_race` phải là key trong `Race.csv` (loader chuẩn hoá lowercase + alias cũ `HAIRY_MAN→fur_people`, `FEATHERMAN→feathered`; sai key = mất toàn bộ hệ số lao động). `point_class`: `NORMAL` | `FAMOUS` (núi danh/bí cảnh thiên địa — anchored vào **Boss cell** của vùng rank ≥ 4, interval 24 internal giờ, `gu_rate` 0.30–0.35, vật liệu đặc hữu: `soul_sand/poison_sand/fire_crystal/ice_crystal/enslavement_stone/langya_refinement_shard/blood_crystal`). Hiện có **40 mỏ thường phủ 44/45 đạo** (mỗi đạo một họ nguyên liệu địa chất; họ bộ phận yêu thú vẫn rơi từ quái) + **7 núi danh**. Sản lượng `<produce_key>_rank<N>` (N = level + hệ số grade LOW+0/MID+2/TOP+4, cap 9); `gu_rate` xác suất mỗi lần thu hoạch sinh 1 cổ hoang dã cùng phai (rank cap 6; rank ≥ 7 không sinh tự nhiên). Đọc trực tiếp lúc khởi động, restart để nhận thay đổi. |
| `backend/data/disposable_email_domains.txt` | Danh sách domain email tạm bị chặn khi đăng ký; đọc lúc khởi động. |

Không ghi số dòng cố định trong tài liệu vì dataset thay đổi thường xuyên. Các helper tìm file ở `data/` trước rồi mới thử đường dẫn legacy. Chỉ sửa CSV chưa làm đổi dictionary trong DB: phải import dataset tương ứng. Workflow Admin luôn parse/validate và dry-run trước, hiển thị row count/diff (`inserts`, `updates`, `unchanged`) rồi mới cho xác nhận commit transaction. Preflight có lỗi nghiêm trọng không ghi database. Sau commit server tự reload registry/invalidate cache; UI sẽ cảnh báo rõ nếu dataset hoặc lỗi reload vẫn yêu cầu restart. Với hai file package đọc trực tiếp, cần chờ cache 24 giờ hoặc xóa cache/restart để thấy thay đổi ngay.

Icon vật phẩm ở túi đồ, danh mục và Thương Đội dùng bộ WebP nhỏ tại `frontend/public/assets/items/icons/`. `frontend/src/lib/item-icons.ts` chọn hình theo tên/dáng vật và chỉ dùng `Item.image_url` khi đó là URL hợp lệ. Bộ icon chạy trực tiếp trên frontend; cập nhật riêng icon không cần import lại `Item.csv`. Xem `frontend/public/assets/items/README.md`.

`python3 scripts/adjust_game_prices.py` chuẩn hóa `Item.csv.base_price` và `Gu.csv.base_price` theo bảng giá trong script, dùng bậc hiệu lực `max(rank, rarity)` (rarity `0` tiếp tục định giá theo rank). Vì vậy món có rarity cao hơn rank sẽ nhận giá của bậc rarity đó. Có thể scale phí recipe riêng bằng `--recipe-primeval 1.2` và/hoặc `--recipe-immortal 0.8`; hệ số này giữ nguyên chênh lệch giữa từng recipe. Lệnh mặc định chỉ xem trước; thêm `--apply` để ghi CSV. Sau đó import các dataset tương ứng trong Admin để cập nhật database.

`python3 scripts/rebuild_recipes.py` tái tạo toàn bộ công thức trong `backend/data/Recipe.csv`: chọn nguyên li�ệu Tiên Cổ phụ tá bằng cách **xoay vòng deterministic theo `target_gu_id`** (`crc32` seed) trong bucket path × rank thay vì luôn lấy phần tử đầu tiên — nhờ đó một cổ (vd `GU_NINE_TURNS_NURTURE_PILL_I` slot Refinement Path rank 6 của công thức rank 8) không còn bị dính vào hàng trăm công thức của mọi path. Slot cố định xuyên path (Refinement/Qi Path ở rank 6–9) vẫn được giữ theo thiết kế nên nguyên liệu lệch path ở các slot này là chủ ý. Script cũng **đảm bảo mọi Gu rank 6–9 đều có ít nhất 1 công thức**: cổ có mặt trong Gu.csv nhưng thiếu recipe (thường là dữ liệu mới thêm) được sinh bổ sung với `variant = 1`; rank ≥ 10 cố ý bỏ qua (cổ truyền thuyết không luyện). Sau khi chạy script: import bảng Recipe qua Admin tab "Dữ liệu".

## Registry runtime

- `models.LoadGuRegistry(db.DB)` load `gu_dictionary` vào `models.GuRegistry` khi server start.
- API `/api/dictionary/gus` trả trường `skill_profile` cùng định nghĩa Cổ; trang `/wiki/gu/:id?rank=` dùng profile này để vẽ sơ đồ vùng đánh và hiển thị cách ngắm, tầm, bán kính, niệm chiêu, hồi chiêu, hậu động tác và số mục tiêu.
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

### Khóa / hạn chế người chơi — moderation

Quyền `players.ban` (cấp cho `game_admin`; `super_admin` có wildcard `admin.*`)
điều khiển ba cơ chế độc lập nhưng dùng chung một write path:

- **Khóa tài khoản**: cột `players.is_banned`, `ban_reason`, `banned_until`
  (NULL = vĩnh viễn), `banned_by`, `banned_at`. `AuthRequired` chặn mọi API khi
  ban còn hiệu lực; login bị từ chối. Hết hạn tự hết (so `NOW()`, không cần cron).
  Khi khóa, `active_session_id` được xoay để kick phiên đang chạy.
- **Hạn chế chat**: `players.is_chat_muted`, `chat_muted_until`. Chặn gửi chat
  Bảo Hoàng và private mail.
- **Chặn thiết bị**: bảng `player_access_blocks` (`ip`, `fingerprint`,
  `banned_until`, `reason`, `banned_by`). Kiểm tra lúc đăng nhập/đăng ký.

**Auto-ban hệ thống** (đọc qua `os.Getenv`, 0 = tắt): `AUTO_BAN_FINGERPRINT_LIMIT`
(mặc định 5 tài khoản/thiết bị/24h), `AUTO_BAN_REFERRAL_LIMIT` (mặc định 5
referral đáng ngờ), `AUTO_BAN_DURATION_HOURS` (mặc định 72; ≤0 = vĩnh viễn),
`AUTO_BAN_BLOCK_DEVICE` (mặc định true).

### Bảng `immortal_apertures` (Tiên Khiếu)

Tạo ngay khi hoàn tất khâu Thả Cổ của nghi thức Thăng Tiên (lên Rank 6). `grade` được định theo ngưỡng Nhân Khí tại khâu Dẫn Khí (≥100k LOW / ≥230k MID / ≥370k HIGH / ≥480k EXTREME — xem bảng `ascension_sessions` và `handlers/ascension.go`). Các cột chính:
- `time_rate` (DOUBLE PRECISION, DEFAULT 10) — gia tốc thời gian nội bộ
- `grade` (VARCHAR) — LOW / MID / HIGH / EXTREME
- `aperture_hp`, `max_aperture_hp` (BIGINT) — HP của Tiên Khiếu (BIGINT để tránh tràn số ở cấp độ cao)
- `area_size` — diện tích grid, quyết định số node tối đa
- `immortal_essence`, `essence_production_per_year` — Tiên Nguyên
- `rank_start_year` — năm bắt đầu rank hiện tại (dùng làm mốc tính lịch giáng lâm tai kiếp)
- `earth_calamity_count`, `heavenly_tribulation_count`, `grand_tribulation_count`, `myriad_tribulation_count`, `chaos_calamity_count` — bộ đếm kiếp nạn
- `tribulation_pending_at` (TIMESTAMPTZ, NULL) — thời điểm tai kiếp bắt đầu bị "giữ" vì người chơi offline; dùng để tính phạt giảm thưởng Đạo Ngân (quá 24 giờ → còn 50%)
- `last_cleanse_at` (TIMESTAMPTZ, NULL) — mốc Thanh Đạo Tẩy Thể gần nhất; cooldown 12 giờ giữa các lần bóc tách Đạo Ngân ngoại hệ
- `state` — STABLE / TRIBULATION_PENDING / TRIBULATION_ACTIVE / DAMAGED / DESTROYED

> **Hằng số thời gian nội bộ:** 1 năm ngoại giới = **36.000 giây (10 giờ) đời thật** khi `time_speed_multiplier = 1`; 1 năm nội bộ Tiên Khiếu = **36.000 giây đời thật ÷ (`time_rate` × `time_speed_multiplier`)**. Sản lượng nền của Tiên Khiếu là `essence_production_per_year` sau mỗi năm nội bộ, nên tốc độ nền mỗi giờ đời thật là `essence_production_per_year ÷ 10 × time_rate × time_speed_multiplier`; Cổ, thể chất, tuổi thọ và hiệu ứng khóa hồi phục vẫn được áp dụng sau đó. Kiếp nạn đầu tiên đến vào năm nội bộ thứ 10 → chờ khoảng `10 × 36.000 ÷ time_rate` giây đời thật (khoảng 2–10 giờ, tùy phẩm cấp Tiên Khiếu). Kiếp đang ACTIVE quá **6 giờ đời thật** (chỉ tính thời gian online) tự động kết toán thất bại.
> **Bảo hộ offline:** khi đến mốc kiếp mà người chơi offline, worker không sinh quái, đặt `state='TRIBULATION_PENDING'`. Khi online lại mới spawn và có 5 phút grace trước khi quái di chuyển/phá mỏ.
> **Vĩnh Sinh (Rank 10):** cảnh giới chỉ gán qua Admin (`admin_player_update.go` cap 1–10), không có đường đột phá từ Cửu Chuyển. Hai tác dụng: **không còn tai kiếp** (`tribulation_trigger.go` bỏ qua `checkAndTriggerSingleAperture` + nhánh lên lịch trong `CheckAndTriggerTribulations`) và **bất tử tuổi thọ** (`gamefx.IsDeadState` luôn false + worker không trừ `lifespan` do lão hóa). Sản xuất Tiên Nguyên, hệ sinh thái, dân cư, gia tốc thời gian và hồi phục giữ nguyên. Tên rank `GetRankName(10)="Vĩnh Sinh"`; nhãn Chân Nguyên tối cao là "Hỗn Độn Tiên Nguyên".

### Bảng `ascension_sessions` (Nghi thức Thăng Tiên)

Một phiên nghi thức Thăng Tiên duy nhất mỗi người chơi (UNIQUE `player_id`), điều phối 3 khâu **Nát Khiếu → Dẫn Khí → Thả Cổ** (logic trong `handlers/ascension.go`):
- `step` — `SHATTERING` (đang trùng kích) → `GUID_QI` (khiếu đã nát, dẫn khí) → `CALAMITY` (Địa Tai Bản Yếu chờ nghênh chiến) → `RELEASE` (thả cổ) → `COMPLETED` / `FAILED`
- `attempts` — số lần trùng kích nát khiếu (mỗi lần hụt +15% tỉ lệ vỡ, base 15%, trần 90%)
- `temper_stage`, `temper_choices` (JSONB) — tiến độ khảo nghiệm tâm tính 3 tầng
- `ascension_qi` — Nhân Khí chốt tại lúc hoàn thành Dẫn Khí, quyết định phẩm cấp Phúc Địa (≥100k LOW / ≥230k MID / ≥370k HIGH / ≥480k EXTREME)
- `expires_at` — deadline 2 giờ đời thật tính từ lúc nát khiếu; hết hạn = FAILED (mất 20% Nhân Khí, giữ 5C Đỉnh Phong)
- Địa Tai Bản Yếu chạy trong private realtime session (`mode='ascension'`) trên map ảo; realtime death settlement thắng → `RELEASE`, thua → `FAILED` và áp dụng mất 20% Nhân Khí. Không dùng turn-based `active_combat`.

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

Điểm nhận từ: dùng Cổ (theo bảng rank bên dưới), cast Cổ thường và đánh thường trong combat realtime, sát chiêu thành công (theo rank Cổ hạch tâm), luyện Cổ thành công (theo rank Cổ đích), và thưởng Chân Ý phó bản (cộng thẳng `amount`). Dữ liệu cũ lưu tên cấp được migration tự động đổi sang mốc điểm khi khởi động.

> **Hệ số Đạo Cấp chỉ dùng trong combat.** Uy lực/khiên/hồi máu của Cổ và sát chiêu nhân hệ số trên; còn **chỉ số bị động** (ATK/DEF/HP/SPEED cộng thẳng từ `ALWAYS_ON` / `PASSIVE_STATS`) là **giá trị raw** — không nhân Đạo Cấp lẫn Đạo Ngân (`handlers/player_stats.go`, `gamefx.PassiveHPBonus`).

**Không có giới hạn Chân Ý theo trận.** Giới hạn Rank × 10 và counter `players.active_combat.path_mastery_gained` thuộc executor turn-based đã bị gỡ. Realtime cộng trực tiếp vào `players.path_masteries` khi settle checkpoint, atomically cùng hunger/mana; checkpoint lỗi sẽ retry ở lần sau.

**Chân Ý khi dùng Cổ:** mỗi lần thôi động Cổ thành công nhận Chân Ý cơ bản theo rank: **R1=1, R2=2, R3=5, R4=10, R5=20, R6=50, R7=100, R8=200, R9=300**. Lượng thực nhận vẫn chịu hiệu suất theo cảnh giới hiện tại và giới hạn cảnh giới của rank Cổ. Sát chiêu dùng cùng bảng theo rank Cổ hạch tâm; đánh thường trúng mục tiêu cộng 1 điểm vào path `strength`.

### Phó Bản — bảng `instance_templates`

- `type` — `INHERITANCE_RIGHTEOUS`, `INHERITANCE_DEMONIC`, `DREAM_REALM` (Mộng Cảnh theo stage), `SECRET_REALM` (dữ liệu stage cũ), `SPATIAL_REALM` (Bí Cảnh bản đồ thao tác).
- Với `SPATIAL_REALM`, `spatial_region_id` phải trỏ tới region `SECRET_REALM` đang hoạt động; `duration_minutes` là thời lượng của phiên chung (mặc định 120 phút).
- `cooldown_hours` — số giờ chờ trước khi vào lại phó bản (mặc định 24; `0` = không giới hạn). Cooldown tính từ session gần nhất của player cho instance đó.
- `rewards` JSONB gồm `attainment` (Chân Ý → `players.path_masteries`), `dao_marks` (Đạo Ngân → `immortal_apertures.dao_marks`), `materials`, `primeval_stones`.
- Seed (`SeedInstanceTemplates`): truyền thừa `dungeon_flower_wine`, **mộng cảnh data-driven từ `data/DreamRealm.csv`** (gồm `dream_realm_star_constellation` + **49 mộng cảnh** `dream_realm_<path>`, 1 cho mỗi lưu phái, thưởng +50 Chân Ý cho đúng path đó, vào cửa tự do (không tốn vật phẩm), cooldown 24h) và **3 bí cảnh** (`SECRET_REALM`, thưởng Đạo Ngân, cooldown 48h).
- `DreamRealm.csv` — cột `id, name, path, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours`. Cột `path` là key lưu phái (vd `fire`); nếu `map_data` để trống, stage tree sẽ được sinh tự động theo path (`buildDreamRealmMapData`). Muốn viết cốt truyện riêng cho một mộng cảnh, điền `map_data` (JSON array stage, xem format mộng cảnh thủ công `dream_realm_star_constellation` trong file). Import/export qua Admin tab "Dữ liệu" → bảng "Mộng cảnh".
- `player_instance_sessions` — session đang chạy; index bán phần `player_active_session_idx` đảm bảo 1 phó bản ACTIVE/player.

**Cảnh giới là điều kiện thăng tiên rank:** để đột phá lên Rank 7/8/9 khi vượt Kiếp Nạn, người chơi cần ≥ 1 lưu phái đạt Tông Sư (500)/Đại Tông Sư (2.500)/Vô Thượng (10.000). Kiểm tra trong `tribulation.go` `handleSuccess`.

### Cấm chế theo khu vực — `regions.rules`

`regions.rules` là JSONB luật dùng chung cho các khu WorldMap và region `SECRET_REALM` của bí cảnh. Luật được kế thừa theo từng nhóm (`rank`, `paths`, `gu`, `items`, `actions`): nhóm có khai báo ở khu con thay toàn bộ nhóm tương ứng ở khu cha; nhóm bị lược bỏ tiếp tục kế thừa. Ví dụ khu con chỉ khai báo `actions` sẽ nhận rank/path/Cổ/vật phẩm từ cha, nhưng thay trọn danh sách hành động bị cấm. Khai báo nhóm rỗng như `"actions": {}` sẽ xóa riêng nhóm đó khỏi luật kế thừa. `{}` kế thừa toàn bộ và giữ nguyên gameplay nếu không có luật ở cha.

Ví dụ:

```json
{
  "rank": {"min": 3, "max": 5},
  "paths": {"allow": ["fire", "sword"], "deny": []},
  "gu": {"max_rank": 5, "deny_ids": ["GU_EXAMPLE"]},
  "items": {"deny_ids": ["escape_token"]},
  "actions": {"deny": ["pvp", "teleport"]}
}
```

`rank` kiểm tra lúc vào; `paths` / `gu` kiểm tra Cổ đang trang bị và Cổ sắp dùng/trang bị; `items` chặn vật phẩm bị cấm; `actions.deny` nhận `pvp`, `teleport`, `gu_use`, `item_use`, `combat`, `monster_combat`, `resource_gathering`, `gu_capture`. Trong một nhóm đã khai báo, mảng rỗng hoặc `max`/`max_rank` bằng `0` sẽ xóa giới hạn tương ứng kế thừa. API Admin kiểm tra cấu trúc và giá trị action; backend kiểm tra lại tại các endpoint gameplay, nên UI client không thể tự bỏ qua luật. API `/explore/regions` trả cả `rules` trực tiếp và `effective_rules` sau kế thừa để client hiển thị. Khi cập nhật rules qua Admin, zone registry và cache khu vực được nạp lại.

### Tiêu hao Chân Nguyên / Tiên Nguyên khi dùng Cổ

Công thức tính tiêu hao nằm ở `gamefx/energy.go` (`EnergyCost` / `EnergyCostForCombatant`):

- **Phàm nhân (Rank 1–5):** chi phí = `BaseManaCost` (mana tuyệt đối, tính ở đúng rank của Cổ và sub 1). Cổ rank thấp hơn giảm chi phí ÷10/rank; **Cổ rank cao hơn bị nhân hệ số lũy tiến**.
- **Phàm nhân, cùng rank, chênh sub-rank:** mỗi sub-rank trên 1 giảm ÷2 (Cổ không có sub-rank nên tính là sub 1): sub 2 → ÷2, sub 3 → ÷4, sub 4 → ÷8. Chỉ áp dụng khi user và Cổ **cùng rank**; đã chênh rank thì chỉ theo thang rank, sub-rank không cộng dồn.
- **Cổ Tiên (Rank ≥ 6):** chi phí = `BaseManaCost` (mặc định 1). Cổ dưới Rank 6 miễn phí; Cổ rank cao hơn nhân hệ số lũy tiến; Cổ rank thấp hơn chia 100^chênh lệch.

**Hệ số rank cao hơn** (`HigherRankManaMultiplier(guRank, userRank)`, theo chênh lệch rank Cổ − rank người dùng):

- **Cổ phàm (rank 1–5):** theo bảng `{1, 2, 5, 20, 100}` cho gap `{0,1,2,3,4}` — ví dụ rank 1 dùng Cổ rank 5 (gap 4) tốn **100×** Chân Nguyên.
- **Cổ Tiên (rank ≥ 6):** mỗi bậc chênh lệch nhân ×100 (`100^gap`) — rank 6 dùng Cổ rank 7 tốn ×100, rank 6 dùng rank 8 tốn ×10.000.

- `refine_mana_tier` (passive Cổ cùng rank đang trang bị, lấy bậc cao nhất) giảm 50% chi phí dùng Cổ và đổi hệ số hồi Chân Nguyên bằng đá trong auto-battle (`0.4 + tier*0.1`, mặc định `1.0`).
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
| `blocked_actions` | JSONB (`[]string`) | Chặn hành động: `explore`, `cultivate`, `combat`, `refine`, `trade`, `breakthrough` (mọi đường lên rank: trùng kích, Thăng Tiên, Relic Cổ, và các Cổ tăng rank) |
| `description` | TEXT | Mô tả |

Danh sách hiện tại (10 trạng thái):

| Key | Tên Việt | Effect | Thời hạn | Chặn |
| --- | --- | --- | --- | --- |
| `severely_wounded` | Trọng Thương | Mặc định -20% attack/defense/speed, -20% tỉ lệ Luyện Cổ và -50% hồi phục nhận; khi đổi Bản Mệnh Cổ, áp chế attack/defense/speed và tỉ lệ Luyện Cổ cùng bằng `RegionalWallDebuff(rank, sub_rank)` (10–90%) | Khi đổi Bản Mệnh Cổ: 5–40 giờ theo rank | `explore` |
| `poisoned` | Trúng Độc | -10% attack, -10% speed | 30 phút | — |
| `exhausted` | Kiệt Sức | -30% speed, -50% hiệu suất tu luyện | 10 phút | `cultivate` |
| `blessed` | Phúc Trạch | +5 may mắn, +10% hiệu suất tu luyện | 60 phút | — |
| `death_penalty` | Hình Phạt Tử Vong | -50% attack/defense/speed | 30 phút | — |
| `refine_fatigue` | Tâm Thần Mệt Mỏi | -50% tỉ lệ Luyện Cổ | 10 phút | — |
| `refine_backfire` | Luyện Cổ Phản Phệ | -95% tỉ lệ Luyện Cổ | 20 phút | — |
| `refine_insight_1..9` | Luyện Cổ Tâm Đắc R1..R9 | +1%/tầng cho đúng rank đó, không cap (Value = số tầng); chỉ hợp luyện đúng nguyên liệu khi Cổ mục tiêu chưa tồn tại mới được cộng/dùng pity | Vĩnh viễn (reset về 0 khi hợp luyện hợp lệ thành công rank đó) | — |
| `sinh_khi_tan` | Sinh Khí Tàn | Tầng I -10%, II -25%, III -35%, IV -50%, V -99% attack/defense/speed (Value âm ghi đè stat_mods) | 300 phút, refresh mỗi hit | — |
| `aperture_petrified` | Không Khiếu Hóa Đá | Không tự hồi Chân Nguyên (block `DISABLE_MANA_REGEN`) | `12 giờ × 2^(rank-1)` (R1 12h, R2 24h, R3 48h, … R9 3072h); `default_duration_minutes` 720 chỉ là fallback khi grant không truyền duration | `breakthrough` |

Sinh Khí Tàn vẫn có status và helper thanh tẩy, nhưng trigger `PERSIST_STATUS_ON_HIT` chưa được nối vào realtime PvP; hit hiện không tự cộng tầng. Các mô tả cũ dựa vào `CombatLog.SourceGuID/HitLanded` và resolver turn-based không phải runtime contract. Recipe: `GU_KILL_HEAVEN_DAO` rank 8 + `heaven_qi_rank8/9`, `kill_path_material`, `primordial_kill_origin_stone`.

**Ngũ Hành Thiên Chướng Trận** (`FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY`, rank 9 Sát Đạo): hai effect `ON_BATTLE_START` từng phụ thuộc resolver turn-based và chưa có profile realtime. Chúng hiện không chạy trong combat. Status `regional_wall_suppression` vẫn có thể được áp bởi quy tắc di chuyển giữa các vùng. Recipe: `GU_KILL_ENERGY` rank 9 + `heaven_qi_rank9` ×10, `kill_path_material` ×20, `primordial_kill_origin_stone` ×10.

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
| `eggman` | Đản Dân | +5% Vân Đạo/Phong Đạo affinity, +10% tốc độ, -10% phòng thủ, +15% hồi phục nhận |
| `miniman` | Tiểu Nhân | +10% Mộc Đạo affinity, +5% tỉ lệ Luyện Cổ, -12% tấn công |
| `mushroomman` | Nấm Nhân | +10% Thổ Đạo affinity, +10% phòng thủ, -10% tốc độ, +15 HP/giờ hồi máu thụ động |
| `yakshaman` | Dạ Xoa Dân | +12% tấn công, +8% Ám Đạo affinity, -8% tỉ lệ Luyện Cổ, -10% kháng tâm lý |
| `ghostman` | U Hồn Tộc | +12% Hồn Đạo affinity, +20% phòng thủ hồn, +20% tốc độ hồi hồn, -10% HP, -15% hồi phục nhận |
| `ancient_human` | Thái Cổ Nhân Tộc | +10% hiệu quả tu luyện, +3 Vận, +10% Nhân Đạo affinity, -5% tỉ lệ Luyện Cổ |

**Thêm chủng tộc mới:** sửa `backend/data/Race.csv` (thêm dòng `key,name,effects`) → Admin panel tab **Dữ liệu** → Import bảng `Chủng tộc`. Nếu database rỗng, Go có `defaultRaces()` trong `backend/internal/gamefx/races.go` làm fallback để không vỡ hành vi cũ. `players.race` luôn là **1 key duy nhất** (default `human`).

**Bảng thể chất (constitution) — data-driven, seed từ `data/Constitution.csv` vào bảng `constitution_dictionary`, nạp registry trong memory (`gamefx.ConstitutionRegistry`) khi khởi động:**

Bảng `constitution_dictionary`:

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `key` | VARCHAR(100) PK | Key tiếng Anh, lưu vào `players.constitution` |
| `name` | VARCHAR(200) | Tên tiếng Việt hiển thị |
| `tier` | INTEGER | Phân cấp thể chất: `0` = Thập Tuyệt Thể / Tuyệt đỉnh; `1` = Dị Thể / Bán Thể Chất cấp dưới |
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
| `cultivation_efficiency` | % hiệu quả tu luyện (trùng kích). Từ cổ trang bị: chỉ tính khi rank cổ trùng rank player, mỗi Gu ID 1 lần |
| `dao_mark_gain` | % Đạo Ngân nhận được |
| `luck` | Vận (số phẳng) |
| `refine` | % tỉ lệ Luyện Cổ thành công |
| `healing_received` | % healing nhận (và % hồi máu thụ động) |
| `hp_regen` | HP/giờ cộng thẳng vào hồi máu thụ động ngoài combat (số phẳng, VD `50` = +50 HP/giờ) |
| `mental_resistance` | % kháng hiệu ứng tâm lý |
| `soul_defense` | % phòng thủ hồn |
| `soul_regen` | % tăng tốc hồi Soul HP thụ động (dưỡng hồn), VD `0.50` = +50% hồi/giờ |
| `aperture_stress_bonus` | hệ số cộng thêm áp lực Không Khiếu mỗi lần trùng kích (VD `1` = tăng ×2) |

**Thêm thể chất mới:** sửa `backend/data/Constitution.csv` (thêm dòng `key,name,effects,tier`) → Admin panel tab **Dữ liệu** → Import bảng `Thể chất`. Nếu database rỗng, Go có `defaultConstitutions()` trong `backend/internal/gamefx/constitutions.go` làm fallback để không vỡ hành vi cũ.

Danh sách hiện tại:
- **Tier 0 (Thập Tuyệt Thể + Thuần Mộng Cầu Chân Thể):**
  - `solar_sun_moon_body` (Thái Nhật Dương Mãng Thể): Quang Đạo đạo ngân ×150%, +20% sát thương Quang Đạo, +50% max mana, +100% hồi mana
  - `ancient_moon_yin_desolate_body` (Cổ Nguyệt Âm Hoang Thể): Hắc Ám Đạo đạo ngân ×150%, +50% max mana, +100% hồi mana, +20% kháng hiệu ứng tâm lý
  - `northern_dark_ice_soul_body` (Bắc Minh Băng Phách Thể): Hàn Băng Đạo đạo ngân ×150%, giảm 10% sát thương Hàn Băng nhận, +50% max mana, +100% hồi mana
  - `forest_sea_reincarnation_body` (Sâm Hải Luân Hồi Thể): Thủy Đạo đạo ngân ×150%, +50% hồi phục nhận, +50% max mana, +100% hồi mana
  - `myriad_gold_brilliant_flower_body` (Vạn Kim Diệu Hoa Thể): Kim Đạo đạo ngân ×150%, +30% attack, +30% defense, +50% max mana, +100% hồi mana
  - `blazing_glory_lightning_brilliance_body` (Viêm Hoàng Lôi Trạch Thể): Lôi Đạo đạo ngân ×150%, +20% speed, +50% max mana, +100% hồi mana
  - `great_strength_true_martial_body` (Đại Lực Chân Võ Thể): Lực Đạo đạo ngân ×150%, +30% attack, +30% defense, +50% max mana, +100% hồi mana
  - `carefree_wisdom_heart_body` (Tiêu Dao Trí Tâm Thể): Trí Đạo đạo ngân ×150%, +30% kháng hiệu ứng tâm lý, +50% max mana, +100% hồi mana
  - `great_earth_origin_body` (Hậu Thổ Nguyên Ương Thể): Thổ Đạo đạo ngân ×150%, +45% defense, +45% kháng hiệu ứng tâm lý, +50% max mana, +100% hồi mana
  - `universe_great_evolution_body` (Vũ Trụ Đại Diễn Thể): Không Gian Đạo đạo ngân ×150%, +15% attack/defense/speed, +50% max mana, +100% hồi mana
  - `pure_dream_reality_seeker_body` (Thuần Mộng Cầu Chân Thể): Mộng Đạo đạo ngân ×140%, +10% sát thương Mộng Đạo, +40% max mana, +100% hồi mana

- **Tier 1 (Bán Tuyệt Thể / Dị Thể cấp dưới - Trade-off tăng giảm đặc thù):**
  - `frost_chilling_body` (Hàn Sương Thể): Băng Đạo đạo ngân +25%, giảm 10% sát thương Hàn Băng nhận, -10% tốc độ
  - `blazing_fiery_body` (Xích Liệt Hoang Thể): Hỏa Đạo đạo ngân +25%, +10% sát thương Hỏa Đạo, +10% attack, -10% defense, -15% kháng tâm lý
  - `brute_strength_body` (Cự Lực Man Thể): Lực Đạo đạo ngân +25%, +15% attack, +10% defense, -10% speed, -20% hồi mana
  - `heavy_rock_body` (Trọng Nham Huyền Thể): Thổ Đạo đạo ngân +25%, +25% defense, +15% kháng tâm lý, -15% speed
  - `spirit_wood_body` (Linh Mộc Dưỡng Sinh Thể): Mộc Đạo đạo ngân +25%, +30% hồi phục nhận, +20 HP/giờ hồi thụ động, -10% attack
  - `iron_bone_body` (Thiết Cốt Huyền Kim Thể): Kim Đạo đạo ngân +20%, +20% defense, +10% tenacity, -10% speed, +25% áp lực không khiếu
  - `clear_wisdom_heart_body` (Minh Tuệ Linh Tâm Thể): Trí Đạo đạo ngân +25%, +20% kháng tâm lý, +10% kháng CC, -10% attack, -10% HP
  - `swift_wind_body` (Tật Phong Phi Hành Thể): Phong Đạo đạo ngân +25%, +20% speed, -15% defense, -10% max mana
  - `night_shade_body` (U Dạ Ám Linh Thể): Ám Đạo đạo ngân +25%, +25% phòng thủ hồn, -10% kháng Quang Đạo, -15% hồi phục nhận
  - `illusion_light_body` (Huyễn Quang Lưu Chuyển Thể): Quang Đạo đạo ngân +25%, +10% sát thương Quang Đạo, +10% speed, -15% hồi mana
  - `blood_fiend_body` (Huyết Ma Sát Thể): Huyết Đạo đạo ngân +25%, +15% attack, +20% hồi phục nhận, +35% áp lực không khiếu, -10% tỉ lệ luyện cổ
  - `dream_phantom_body` (Huyễn Mộng Linh Thể): Mộng Đạo đạo ngân +20%, +5% sát thương Mộng Đạo, -15% kháng tâm lý, -10% kháng CC

### Hồ sơ chiêu của Cổ

`Gu.csv.skill_profile` là object tùy chọn trên đúng dòng `(id, rank)` của Cổ. Nó
lưu cách ngắm, hình vùng đánh và thông số thời gian để gameplay realtime và wiki
cùng đọc một nguồn. Hiệu ứng DAMAGE/STATUS/SHIELD tiếp tục nằm trong `effects`,
không bị lặp trong hồ sơ chiêu. Để trống trường này nếu Cổ chưa có chiêu realtime.

| Cột | Kiểu | Ý nghĩa |
| --- | --- | --- |
| `cast_mode` | VARCHAR(20) | `TARGET` (chọn unit) · `DIRECTION` (hướng) · `GROUND` (điểm đất) · `SELF` |
| `shape` | VARCHAR(20) | `SINGLE` · `LINE` (dài × rộng) · `CIRCLE` (bán kính) · `CONE` (dài × góc) |
| `radius` | DOUBLE | Bán kính vùng tròn |
| `length` | DOUBLE | Chiều dài tia/hình quạt |
| `width` | DOUBLE | Bề rộng tia |
| `angle_deg` | DOUBLE | Góc mở của hình quạt |
| `cast_range` | DOUBLE | Tầm xa tối đa từ người dùng tới điểm/hướng |
| `cast_time_ms` | INT | Thời gian niệm (đứng yên); 0 = tức thời |
| `recovery_ms` | INT | Khựng sau khi xuất chiêu |
| `cooldown_ms` | INT | Hồi chiêu thời gian thực (khác status cooldown theo lượt) |
| `projectile_speed` | DOUBLE | 0 = đánh tức thời; >0 = đạn bay (ô/giây) |
| `pierce` | BOOL | Đạn/tia xuyên qua nhiều mục tiêu |
| `max_targets` | INT | Số mục tiêu tối đa mỗi lần trúng |
| `move_distance` | DOUBLE | Lướt/dịch chuyển tức thời (ô); 0 = không |
| `knockback_distance` | DOUBLE | Đẩy lùi mục tiêu khi trúng đòn (ô); 0 = không |
| `player_atk_ratio` | DOUBLE | Tỷ lệ hưởng theo công kích nhân vật (% Base Attack, mặc định 0.5) |
| `gu_power_ratio` | DOUBLE | Tỷ lệ hưởng theo uy lực Cổ (% Base Gu Damage, mặc định 1.0) |

Quy ước chọn mode theo hình:

- `TARGET` + `SINGLE` → đòn chọn mục tiêu cổ điển.
- `DIRECTION` + `LINE` → skillshot xuyên (đạn bay hoặc tia tức thời).
- `GROUND` + `CIRCLE` → AoE đặt xuống đất, có `cast_time_ms` thì người chơi né được.
- `DIRECTION` + `CONE` → sóng/quạt.
- `GROUND` + `SINGLE` + `move_distance` > 0 → lướt tới điểm.
- `SELF` → tự buff (khiên/hồi), bắn ngay không cần ngắm.

Cơ chế hiệu ứng thế giới mở (Realtime Status Effects):
- **Câm lặng (`SILENCE`, `GU_SEAL`, `FORGET_EGO`)**: Khóa tung chiêu Cổ, nhân vật vẫn có thể di chuyển và đánh thường.
- **Mù lòa (`BLIND`)**: Đòn đánh thường bị trượt (miss với 0 damage).
- **Phản sát thương (`THORNS`)**: Phản lại 35% sát thương nhận trực tiếp ngược về kẻ tấn công.
- **Vết thương thiêu đốt (`BURN`)**: Giảm 50% lượng máu hồi phục từ các chiêu thức hỗ trợ (`HEAL`).
- **Trói chân (`ROOT`, `BIND`) vs Choáng cứng (`STUN`, `FREEZE`, `PETRIFY`)**: `ROOT` giảm tốc độ di chuyển về 0 nhưng vẫn cho phép quay hướng và tung chiêu; Hard CC khóa cả di chuyển lẫn dùng kỹ năng.
- **Tỉ lệ sát thương lai (Hybrid Scaling)**: Sát thương chiêu realtime được tính kết hợp giữa công kích nhân vật và uy lực Cổ: `(Damage * gu_power_ratio + BaseAttack * player_atk_ratio) * DamageScale`.

`cooldown_ms` dùng thời gian thực. Khi vào map, chi phí dùng Cổ được tính từ
`gamefx.EnergyCost` theo chuyển/cảnh giới và thuế xung đột Đạo Ngân. Zone giữ
pool Chân Nguyên của nhân vật và ghi delta tiêu hao về `players.current_mana`
cùng checkpoint; không dùng pool combat mặc định 100 cho nhân vật thật.

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

Trong combat realtime, mỗi slot hotbar gắn với một `player_gu.id` cụ thể. Một
lần thử cast hợp lệ trừ 1 `hunger` của instance đó trước khi kiểm tra thiếu
Chân Nguyên, Zone chặn cast khi độ no dưới 1; checkpoint
trừ tổng lượt dùng theo delta trong cùng transaction ghi vị trí/HP/mana.

`movement_policy` của profile được áp trong windup và recovery: `SLOW` còn 50%
tốc độ chạy, `ROOT` dừng di chuyển, `FREE`/bỏ trống giữ nguyên tốc độ. `recovery_ms`
khóa cast kế tiếp và đánh thường sau khi chiêu kết thúc, kể cả khi GCD đã hết;
chiêu bị ngắt không tạo recovery. Cooldown, status có thời hạn và bộ đếm hidden
passive được ghi vào `player_world_state_combat` trong cùng transaction checkpoint,
theo số tick còn lại, nên rejoin hoặc chuyển vùng không reset để né debuff.
Cooldown chỉ khôi phục nếu đúng Cổ đó vẫn nằm ở slot cũ; nhân vật chết xóa trạng
thái combat có thể khôi phục.

- Migration `migrate_player_gu_from_inventory` (chạy một lần khi còn cột `gu_inventory`) copy toàn bộ dữ liệu cũ sang `player_gu`, sau đó `DROP COLUMN IF EXISTS gu_inventory` loại bỏ cột. An toàn chạy lặp lại.
- Cổ đang trang bị/Bản Mệnh vẫn được mirror trong cột JSONB `players.equipped_gus` (danh sách `instance_id`) để các truy vấn nhẹ (status) không cần join.
- API `GET /api/player/status/:id` chỉ trả Cổ đang trang bị/vital; toàn bộ túi Cổ lấy qua `GET /api/player/inventory/gus` (helper `db.GuInventoryAggregate` / `db.LoadPlayerGus`).
- `monster_templates.gu_inventory` (JSON trong cột monster) **giữ nguyên**, không đổi.
- Sát chiêu tự tạo được lưu riêng trong `player_killer_moves`: công thức gồm Cổ chính/phụ, tham chiếu instance Cổ, effect đã chọn và effect đã cân bằng, cùng độ thành thạo và trạng thái trang bị. Cổ có thể được dùng trong nhiều sát chiêu; mỗi lần kích hoạt, từng Cổ thành phần giảm 1 độ no.
- Các Cổ thành phần được kiểm tra lại theo `player_gu.id`, `gu_id`, `gu_rank` và trạng thái sống khi trang bị/thi triển. Bán, hợp luyện hoặc làm chết Cổ khiến sát chiêu phụ thuộc vào instance đó tạm mất hiệu lực.
- Cân bằng sát chiêu hiện dùng ngân sách theo rank Cổ chính, đóng góp giảm dần của Cổ phụ, tải effect theo loại/giá trị/phạm vi/thời lượng/xác suất, độ hòa hợp lưu phái và độ phức tạp. Effect được giảm tự động khi vượt ngân sách; `balance_version` định danh luật dùng lúc tạo.
- Người chơi có thể lưu tối đa 12 sát chiêu và trang bị tối đa 4. Tối đa bốn sát chiêu được nạp vào realtime hotbar ở ô 9–12 khi có đúng một effect `ON_ACTION` được hỗ trợ: damage đơn mục tiêu, hồi bản thân, tạo khiên bản thân hoặc thanh tẩy. Windup/cooldown realtime được xác định theo rank và độ phức tạp; không chuyển `warmup_turns`/`cooldown_turns` cũ thành thời gian. Effect có điều kiện, status/backfire chưa hỗ trợ, effect nhiều operation và loại effect khác bị khóa. `ON_USE` hồi phục bản thân vẫn có thể kích hoạt ngoài combat qua `UseKillerMove`.
- Cột `players.temp_loot` đã bị **bỏ hẳn** (migration `ALTER TABLE players DROP COLUMN IF EXISTS temp_loot;`). Chiến lợi phẩm khám phá giờ vào thẳng túi thật (nguyên liệu → `player_inventory_items`, thạch → `players.primeval_stones`/`immortal_stones`, Cổ → `player_gu`).

### Cơ chế rơi đồ khi tử đạo & Thiên Đạo Tiêu Hủy (Death Penalty & Void Destruction)
- **Tỷ lệ phạt cơ bản khi chết:** 40% Nguyên Thạch & Tiên Nguyên Thạch, 40% chủng loại nguyên liệu (tối đa 30 loại), 30% Cổ trùng (ngoại trừ Cổ Bản Mệnh; Cổ trang bị chỉ rơi khi Sát Khí > 5), 30% Bí phương.
- **Hệ số Oán Khí:** Sát Khí tăng tỷ lệ hao hụt tài nguyên (`mult = 1.0 + level * 0.5`).
- **Tích Sát Khí:** Chỉ tăng khi hạ Cổ Sư PvP (kẻ chủ động). Lượng tầng nhận = 1 + chênh lệch rank khi rank kẻ hạ cao hơn nạn nhân (cao hạ thấp tích nhanh hơn). Sát Khí là trạng thái vĩnh viễn — không giảm theo thời gian, chỉ bị trừ 1 tầng khi tử đạo.
- **Thiên Đạo Tiêu Hủy (40% Void Destruction):** Trong số tài nguyên bị mất, 40% bị chấn nát tiêu hủy vĩnh viễn khỏi server (chống hành vi tử đạo để tuồn/chuyển tài nguyên cho tài khoản khác). 60% còn lại rơi ra bãi đất `map_ground_drops` tồn tại trong 30 phút.


### Bảng `aperture_resource_nodes`

Node tài nguyên đặt trong Tiên Khiếu:
- `aperture_id` (UUID, FK → immortal_apertures, CASCADE DELETE)
- `node_type`, `name`, `path` — loại node
- `level`, `hp`, `max_hp` — cấp và HP
- `x`, `y` — toạ độ grid (0-19)
- `production_type`, `production_amount`, `production_interval_hours` — sản lượng; `production_amount` chỉ còn là dữ liệu cũ, sản lượng thu hoạch do `gamefx.ComputeApertureYield` tính từ cấp node, Đạo Ngân, diện tích, sinh thái và lao động — sàn 10 món/lần cho mọi rank (xem `WIKI.md` §21)
- `is_destroyed` — cờ tương thích dữ liệu cũ; công trình bị phá hủy hoàn toàn được xóa khỏi bảng và không thể sửa chữa
- Số khu tài nguyên tối đa mỗi Tiên Khiếu = `min(area_size / 100000, 200)`; lưới 20×20 được lưu trong `aperture_resource_node_cells`.
- Xung đột Đạo khi tính sản lượng dùng chung `gamefx.DaoCombatProfile` với combat, không dựa trên khoảng cách giữa các node.

### Bảng `aperture_resource_node_cells`

Mỗi hàng là một ô thuộc vùng của node; `UNIQUE (aperture_id, x, y)` ngăn các node chiếm cùng ô. Xóa node sẽ xóa các ô vùng theo khóa ngoại. `production_remainder` lưu phần sản lượng lẻ tại ô neo để các khu nhỏ vẫn cộng dồn đúng theo diện tích. Node cũ được cấp một ô neo khi bảng mới được khởi tạo.

### Kinh tế và xã hội Dị Nhân trong Tiên Khiếu

- `aperture_node_workers`: gán một phần dân số `aperture_variant_humans` vào node; cặp node/chủng tộc là duy nhất. Bảng chỉ mang tính tổ chức — sản lượng thu hoạch lấy hệ số dân số/lao động theo chủng tộc của cả Tiên Khiếu, không cộng thêm theo phân công.
- `aperture_refinement_queue`: hàng đợi luyện chế theo recipe, mode, mục tiêu, tiến độ, ưu tiên và trạng thái bật/tắt; `last_error` và `last_processed_at` cho biết lý do dừng/lần xử lý cuối.
- `aperture_variant_settlements`: khu định cư theo chủng tộc, sức chứa, nhà ở, kho lương và an ninh.
- `aperture_variant_events`: sự kiện xã hội (loại, mức độ, effects JSONB, thời điểm giải quyết).
- `non_player_actors` với `actor_kind = VARIANT_NPC`: cá thể Cổ Sư/Cổ Tiên với rank, lưu phái, vai trò, trung thành, thiên phú và trạng thái.

Các cột xã hội trên quần thể lưu kho lương, lương thực đầu người, tăng trưởng và tử vong. Xây khu định cư đưa 500 cư dân ban đầu và lượng lương thực bằng sức chứa kho; mô phỏng xã hội tăng/giảm dân số nhưng không vượt sức chứa khu định cư. Worker production, society simulation và Gu Master spawning lần lượt chạy theo chu kỳ 10, 5 và 30 phút. Gu Master spawn khi `population >= 500`, chance `5% * (pop/1000)` mỗi tick (prosperity buff ±25%), cap 100/aperture/race. Mỗi tick production cũng thử một lần cho các lệnh tự luyện ưu tiên cao. Một lần thử cần `rank × 100` Mao Dân chưa được phân công (Nhất Chuyển 100, Ngũ Chuyển 500, Cửu Chuyển 900); worker giữ chỗ tổng lao động trong tick và bỏ qua lệnh chưa đủ người để lệnh cấp thấp hơn vẫn có thể chạy: cần Mao Dân, bí phương hoàn thiện, nguyên liệu thường và nguyên thạch; thất bại tiêu hao nguyên liệu, thành công tạo một Cổ và tăng tiến độ. Lệnh thiếu tài nguyên tạm chờ và tự thử lại ở tick sau. Bí phương dùng Cổ khác làm nguyên liệu chưa được nhận vào hàng đợi tự động. API cho phép phân công lao động (tổng phân công không vượt dân số), quản lý hàng đợi, nâng khu định cư và đổi vai trò/giải tán NPC; xem route map trong `docs/API.md`.

### Bảng `tribulation_events`

Lưu các sự kiện Kiếp Nạn của Tiên Khiếu:
- `id` (UUID PRIMARY KEY)
- `aperture_id` (UUID FK → `immortal_apertures(id)`)
- `player_id` (UUID FK → `players(id)`)
- `tribulation_type` (VARCHAR) — EARTH_CALAMITY / HEAVENLY_TRIBULATION / GRAND_TRIBULATION / MYRIAD_TRIBULATION / CHAOS_CALAMITY
- `path` (VARCHAR) — Đường hướng kiếp nạn (ví dụ: fire, water, ice)
- `status` (VARCHAR) — ACTIVE / SUCCESS / FAILED
- `aperture_hp_before` (BIGINT) — HP Tiên Khiếu trước kiếp nạn
- `aperture_hp_after` (BIGINT) — HP Tiên Khiếu sau kiếp nạn (NULL nếu chưa kết thúc)
- `reward_dao_marks` (BIGINT) — Số Đạo Ngân thưởng
- `started_at` / `ended_at` (TIMESTAMPTZ)

Khi nhân vật đang ở map realtime, các endpoint `realtime-engage`, `realtime-victory`, `realtime-escape`, `realtime-exit` bàn giao lease zone cũ trước khi ghi `current_region/world_zone_id`: flush vị trí live, tạm dừng checkpoint, ghi đích và nhả lease. Nhờ vậy checkpoint/disconnect của zone cũ không ghi đè region và lần join kế tiếp không bị `realtime_map_mismatch`.

Mỗi event có một instance realtime riêng (`aperture_tribulation#<event_id>`), không dùng chung map với người khác. Khi join, backend dựng encounter từ chính `tribulation_enemies` và `aperture_resource_nodes` (không dùng spawn tĩnh trong map artifact): HP/ATK/DEF lấy từ row DB đã scale theo rank/sub-rank, `tribulation_enemies.is_dead` được cập nhật khi quái chết, checkpoint 10 giây đồng bộ HP quái/node còn sống. Quái của kiếp nạn có `NoRespawn`, nên không hồi sinh; node phòng thủ (`node_category='DEFENSE'`) bị loại khỏi encounter realtime. Worker legacy dừng mô phỏng khi `players.current_region='aperture_tribulation'`; node do quái phá trực tiếp trong combat realtime. Khi quái cuối cùng chết, server tự kết toán victory (idempotent) và đưa nhân vật về vị trí thoát. Khi player hết máu trong combat, server tự đưa về vị trí cũ, hồi HP về 1 và không rơi rớt vật phẩm. Map realtime là 40×40; các vị trí node/quái từ lưới Tiên Khiếu bền vững 20×20 được chiếu theo tỷ lệ 2×, người chơi vào tại (20,20). Backend lưu vị trí region/tọa độ local trước khi vào cùng anchor atlas độc lập với FK region. Khi thoát, nếu region lưu còn active và tọa độ nằm trong bounds thì khôi phục đúng vị trí (hub cũng được); nếu không, chọn `KHU_VUC`/`SAFE_ZONE` active gần anchor nhất và dùng entry point, rồi chọn hub active gần nhất. Nếu không tồn tại đích nào thì trả lỗi rõ ràng, không chuyển ngầm về tọa độ mặc định. Map event riêng được nạp nội bộ, không cần cấu hình zone hay allowlist; backend xác thực tài khoản, vị trí và trạng thái event ACTIVE. Khi cập nhật server hiện hữu, import lại dataset **Region** trong Admin để áp dụng kích thước 40×40; seed startup chỉ thêm region còn thiếu và không ghi đè row đã có.

### Bảng `aperture_return_anchors`

Lưu tọa độ atlas của vị trí trước độ kiếp, độc lập với `return_region` để vẫn chọn được vùng an toàn lân cận khi region cũ bị xóa. Gồm `aperture_id` (PK/FK, cascade), `atlas_x`, `atlas_y` (nullable) và `updated_at`. Bảng tự tạo bằng `CREATE TABLE IF NOT EXISTS` khi khởi động.

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

> Combat turn-based, simulator và các `turn`/`turns` trong mô tả legacy bên dưới
> không còn là runtime. Combat hiện chạy trong realtime zone; chỉ Gu/profile và
> effect có executor realtime được khai báo trong `docs/mmo/gu-realtime-schema.md`
> mới được cast. Catalog/import vẫn có thể giữ effect cũ để tránh mất dữ liệu,
> nhưng được phân loại không đồng nghĩa với có hiệu lực trong combat.

Các trường quan trọng trong dictionary:

- `id`, `name`, `rank`, `path` (mảng JSON các Đạo học — 49 path có thể có, xem danh sách ở `docs/CANH_GIOI_DAO_NGAN.md`), `gu_type`.
- `base_mana_cost`.
- `effects` JSON: active/passive/growth/other/support tuỳ Cổ. Schema Go nằm ở `models.GuEffects` (`backend/internal/models/gu.go`): `ActiveEffect`, `PassiveEffect`, `GrowthEffect`, `SpecialMechanic`.
  - **Validation khi ghi dữ liệu:** `effectcatalog.NormalizeAndValidate` kiểm tra JSON root, tám namespace chuẩn (`active`, `support`, `passive`, `house`, `explore`, `refinement`, `growth`, `other`), kiểu container, status và cấu trúc `SpecialMechanic`. Admin CRUD trả HTTP 400 kèm `issues[]` có `severity`, `code`, `path`, `message` nếu dữ liệu sai. Import `Gu.csv` preflight toàn bộ effect trước database write đầu tiên.
  - Field đã phân loại nhưng chưa được runtime xử lý đầy đủ vẫn được giữ nguyên và phát warning theo disposition (`effect_migrate`, `effect_partial`, `effect_known_bug`); warning không đồng nghĩa effect đã hoạt động. Cách warning-first này tránh mất dữ liệu trong giai đoạn migrate. Namespace lạ, JSON/kiểu container sai, status hoàn toàn không biết và mechanic thiếu trường bắt buộc là lỗi chặn ghi/import.
  - Manifest `effectcatalog` gán mỗi key/status/cặp `(event_trigger, action_code)` một `owner` và `disposition`: `supported`, `partial`, `metadata`, `migrate` hoặc `known_bug`. Key hoặc mechanic chưa có manifest bị chặn bằng lỗi `unclassified_*`; test coverage toàn `Gu.csv` buộc dữ liệu mới phải được phân loại trước khi merge.
  - Admin có preview read-only `POST /api/admin/gu-effects/validate` và audit database `GET /api/admin/gu-effects/audit`. Audit không thực thi effect; nó cho biết coverage và những effect đang chờ migrate/sửa lỗi.
  - **Khối `active`:** effect chủ động dùng trong combat — `damage_flat`, `damage_mult`, `damage_path`, `heal_flat`, `shield_flat`, `apply_status`, `target_selector`, `summon_*`, `curve_attack`, `lifesteal`, … Danh sách đầy đủ + catalog chuẩn xem mục **"Hiệu ứng Cổ (effects)"** bên dưới.
    - `apply_status`, `chance`, `turns` và các key combat khác có thể còn trong dữ liệu/import legacy. Những trường này không tự tạo profile realtime; không chuyển `turns` thành giây và không suy ra executor từ parser/catalog.
  - **Khối `passive` (Block v2 duy nhất, không còn đọc flat cũ):** chỉ số vĩnh viễn cộng dồn (`STAT_MOD ... SPEED`, `MOVEMENT` = phạm vi di chuyển ô — tốc chạy map, `SPEED` chỉ thứ tự đánh combat), `damage_reduction`, `luck`, `storage_capacity`, `tenacity` (0..1: rút ngắn duration CC + tỉ lệ tự giải 1 debuff đầu lượt), `cc_resist` (0..1: kháng proc mọi debuff, nhân với mental resist, cap tổng 0.75), …
    - Các key khắc hệ/passive trong legacy block không tự áp dụng vào realtime. Chỉ dùng profile và kháng hệ hiện có trong schema MMO; xem `docs/mmo/gu-realtime-schema.md`.
  - **Shorthand status (đã loại khỏi dữ liệu):** key viết tắt (`freeze`, `slow`, `slow1`, `weak`, `stun`, `silence`, `bleed`, `poison`, `thorns`, `regen`, `atk_up`, `def_up`, `taunt`) không còn xuất hiện trong `Gu.csv` — toàn bộ đã migrate sang `apply_status` dạng mảng. `ActiveEffect.normalize()` vẫn giữ làm fallback cho DB rows cũ chưa re-import; **không dùng cho dữ liệu mới**.
  - **Cơ chế `other` (SpecialMechanic):** mảng `{event_trigger, action_code, params}`. Việc trigger/code được chấp nhận ở catalog hoặc parser không bảo đảm executor combat realtime tồn tại. Combat runtime chỉ dùng effect/profile có hỗ trợ được liệt kê trong `docs/mmo/gu-realtime-schema.md` và phần combat realtime của `WIKI.md`.
    - Các mechanic theo lượt (`ON_TURN_START`, `ON_BATTLE_START`, `turns`, initiative, reborn/rewind, summon theo trận và tương tự) là dữ liệu/logic legacy; simulator đã bị xóa và hiện chưa có executor realtime tương đương.
  - Dịch chuyển bằng Cổ vẫn chạy qua movement/world handlers; phạm vi theo rank và quyền dịch chuyển được mô tả ở phần route movement bên trên. Các mechanic hành động combat theo lượt được giữ trong dữ liệu để migration/đối chiếu, không phải runtime contract hiện hành.
    - Alias consume đã chạy (không còn 400): `STONE_APERTURE`→`APERTURE_PETRIFICATION`, `POISON_VOW`→`CONTRACT_OATH`, `THIRD_WATCH`→`TIME_ACCELERATION`, `RECORD_MESSAGE`→`RECORD` (dùng qua endpoint Support, trigger `ON_USE`). `STONE_APERTURE`/`APERTURE_PETRIFICATION` chỉ áp dụng khi rank Cổ trùng với rank người dùng. `MAN_BEAST_BURIAL` cho phép Cổ Sư Rank 1–4 ở Đỉnh Phong đột phá lên Rank kế tiếp Sơ Giai khi rank Cổ trùng với rank người dùng; giảm 10 Chân Nguyên tối đa (tối thiểu 20) và kẹp Chân Nguyên hiện tại theo giới hạn mới. `SPRING_EGG_HATCH` chạy được qua Cổ tiêu hao `GU_SPRING_EGG` (endpoint Consumable, có `tx` để cấy Phàm Tuyền); trigger khác hoặc thiếu `tx` trả lỗi "chỉ có thể cấy Nguyên Tuyền khi tiêu hao trực tiếp" và **không** tiêu Cổ. Endpoint Support persist đủ cột như Consumable.
    - Action code liên quan Hồn Phách:
    - `SOUL_REGEN_TICK` (`event_trigger: ON_TICK`): hồi `soul_hp` theo giờ từ mỗi Cổ trang bị ACTIVE (`params.value` = số điểm/giờ, scale theo rank qua `ManaRegenTickFactor`; xử lý ở `models.SoulRegenTickPerHour`).
    - Ba nguồn `*_REGEN_TICK` là hồi thụ động: chúng được cộng với hồi nền trước khi áp dụng chính sách `DISABLE_HP_REGEN`, `DISABLE_MANA_REGEN`, `DISABLE_RESOURCE_REGEN` (Mana + Niệm Lực) và `DISABLE_SOUL_REGEN`. Item/Cổ `ON_CONSUME` là hồi chủ động nên không bị hệ số hồi tự động này chặn.
    - `RESTORE_SOUL_HP` (`event_trigger: ON_CONSUME`): hồi ngay `soul_hp` khi sử dụng Cổ tiêu hao (`params.value` mặc định 10, cap tại `max_soul`).
    - `ADD_SOUL_MAX` (`ON_CONSUME`): tăng `max_soul`; `SOUL_BOOST` (`ON_CONSUME`): cộng Hồn Đạo đạo ngân.
    - Action code liên quan dòng thời gian:
    - `INCREASE_TIME_SPEED` (`event_trigger: ON_CONSUME`): cộng vĩnh viễn `params.value` vào `players.time_speed_multiplier` → hồi mana/hp/soul/niệm nhanh hơn nhưng tuổi thọ cũng trôi nhanh hơn.
- `hunger_rate`, `food_types`, `max_hunger`. **Quy ước `hunger_rate`:** cổ có `food_types` không rỗng → `6` nếu `effects.passive` không rỗng, ngược lại `3`; cổ không có thức ăn → `0` (không đói). `max_hunger` mặc định `rank × 100`.
  - `food_types` hỗ trợ các pseudo-food trừ trực tiếp tài nguyên người chơi: `primeval_stone` (`2^rank`), `immortal_stone` (`rank`), `blood` (10 HP), `thought` (10 Niệm Lực), `soul` (10 Linh Hồn) và `lifespan` (1 Tuổi Thọ) cho mỗi khẩu phần 100 Độ No. Bốn chi phí sinh mệnh/tinh thần là cố định, không scale theo rank; HP, Linh Hồn và Tuổi Thọ luôn chừa lại ít nhất 1 điểm.
- `target_type`: `SINGLE`, `ROW`, `COLUMN`, `ALL`, v.v.
- `rarity`, `primary_role`, `combat_roles`, `combat_tags`, `support_tags`.
  - Gu rank ≥ 6 mặc định là Tiên Cổ; trạng thái được suy ra từ `rank`, không lưu thành cột riêng.
  - **`rarity` là số nguyên 0–9** (`0` = chưa đặt, fallback theo rank). Dùng để cho Cổ phàm (rank < 6) độ hiếm cao: drop weight tính theo **`max(rank, rarity)`** (`handlers/effectiveDropLevel`), nối vào gacha, rương, và ô WildGu trên map. Item (`item_dictionary`) cũng có cột `rarity` cùng quy tắc. Giá trị cũ dạng chữ/boolean trong CSV đã được dọn về rỗng.
- `base_price`: hỗ trợ tối đa 3 chữ số thập phân (đơn vị nhỏ nhất `0.001`); **fallback đồng đều theo rank** — 200, 500, 1.200, 2.500, 5.000, 100.000, 250.000, 600.000, 1.500.000 cho rank 1–9 (nhảy mạnh ở Tiên Cổ rank ≥ 6). Fallback trong `pkg/pricing/pricing.go`.
- **Phân phối thức ăn:** mọi item trong `food_types` của cổ rank <6 (72 item) được thêm vào (1) loot của quái **Normal/Elite** theo dải rank spawn, (2) `pool_weights.resources` của vùng non-HUB theo dải rank vùng, (3) `vault_materials` khởi đầu của 4 tông (qty 100, khớp dải rank vùng tông). Boss/Tribulation không rơi thêm thức ăn.
- **Tính duy nhất của Tiên Cổ (rank ≥ 6):** cổ trùng rank ≥ 6 là **duy nhất toàn server, tính theo `gu_id`** (cùng id khác rank vẫn bị tính trùng). Áp dụng cho instance **đang sống** ở mọi nguồn: `player_gu`, `auction` ACTIVE (`item_type='Gu'`), `sect_vault_gu` (`quantity>0`). Enforce 2 lớp: app-level `db.CheckImmortalGuExists` (+ advisory lock, trả **409** khi trùng: refine/gacha/deposit) và backstop DB `uq_player_gu_immortal` / `uq_sect_vault_gu_immortal` + `chk_sect_vault_gu_immortal_qty` (kho rank≥6 luôn `quantity<=1`). Destroy (đói chết/thả/chết rơi đồ) = end lifecycle `instance_id` đó, `gu_id` được tái sinh với `instance_id` mới. Nguồn kiểm tra: `db.CheckImmortalGuExists` / `db.GetExistingImmortalGuIDs` / `db.IsImmortalConflict`.
- **`instance_id` bất biến xuyên vòng đời:** trade, đăng bán/đấu giá, ký gửi, nhập/xuất kho tông môn chỉ đổi chủ/vị trí, không đổi `id`. Đăng bán Tiên Cổ mỗi đơn 1 instance; rút kho Tiên Cổ mỗi lần 1 instance, giữ nguyên id. Regen id mới chỉ cho legacy snapshot thiếu id rank < 6.
- **`gu_history_logs` chỉ log Tiên Cổ (rank ≥ 6):** `LogGuHistory` drop mọi event rank < 6. Event terminal (`STARVED_DEATH`, `DEATH_DESTROYED/DROPPED`, `DROPPED_RELEASED` với `metadata.terminal=true`) đóng timeline của `instance_id`.
- **Tiên Cổ không xuất hiện từ loot/region:** mọi entry loot table có `gu_id` là cổ rank ≥ 6 bị **lọc bỏ** khi (1) import CSV LootTable, (2) Admin tạo/sửa bảng loot, (3) `RollLoot` roll thực tế. `pool_weights.wild_gus` và `pool_weights.monster_gus` của Region cũng được lọc Cổ Tiên khi import CSV (xem `filterImmortalWildGus` / `filterImmortalMonsterGus`) và khi sinh ô WildGu (`GenerateCell`), đảm bảo Cổ Tiên chỉ đến từ gacha, luyện cổ và đấu giá.

### Hiệu ứng Cổ (effects)

> Từ điển hiệu ứng cho người chơi: `/wiki/effects` (nội dung tại
> `frontend/src/lib/wiki/effects.tsx` + dữ liệu tra cứu
> `frontend/src/lib/wiki/effect-glossary.ts`). Khi thêm trigger, operation,
> target, condition hoặc native hook mới, cập nhật `effect-glossary.ts` —
> `frontend/src/lib/wiki/effects.test.ts` sẽ fail nếu `Gu.csv` chứa mã chưa
> được giải thích.

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

**Matrix A — namespace/key Cổ legacy (inventory only, not a runtime contract).** Key không ghi `[parser-only]` được quan
sát trực tiếp trong `Gu.csv.effects` và persist ở `gu_dictionary.effects`;
`[parser-only]` nghĩa là chỉ còn trong Go model/manifest để đọc DB cũ, không có
trong snapshot CSV hiện tại. UI label là `Gu.name` cộng formatter
operation (legacy chưa có label riêng); definition do `gu_dictionary` sở hữu,
runtime combat nằm trong combat snapshot, còn world/worker command persist bởi
handler/worker tương ứng. Các ghi chú về executor/test trong bảng là trạng thái lịch sử trước cutover;
`catalog` từng chỉ các test `effectcatalog/{catalog,validate,timing}_test.go`; `characterization` từng gồm test handler/simulator/worker. Backend turn-based đã bị xóa theo #272; những executor/test simulator bên dưới là ghi chú lịch sử, không mô tả runtime hiện hành. Runtime combat hiện hành nằm trong `backend/internal/realtime/`; chỉ các effect có profile realtime được phép dùng trong chiến đấu.

| Legacy namespace/key (mỗi key trong ô là một entry độc lập) | Trigger → target | Unit / scaling | Executor hiện tại; persistence owner | Test hiện có |
|---|---|---|---|---|
| `active.damage_flat`; `damage_mult`; `damage_path`; `damage_hp_percent` | action → selected enemy(s) | flat HP; multiplier; path enum; ratio max HP | combat handler + simulator; combat state | catalog + simulator |
| `active.shield_flat`; `shield_flat_reduce_percent`; `heal_flat`; `mana_regen`; `blood_drain`; `convert_damage_to_heal_percent` | action/hit → self or selected target | flat shield/HP/mana; ratios | combat; `players`/combat state; conversion key chưa executor | catalog + handler/simulator (không có conversion) |
| `active.apply_status`; `status_duration_ticks`; `status_tick_damage` | after hit → selected target(s) | status id, turns, flat/tier legacy, optional probability | `statusfx` + simulator; combat state | catalog + status/simulator |
| `active.target_selector`; `range` | selection before action | enum; cells | handler/simulator; transient | catalog |
| `[parser-only] active.ignore_defense`; `ignore_defense_pct`; `armor_pen`; `crit_rate_boost`; `speed_boost_flat`; `all_stats_buff_percent`; `lifesteal_rate` | action/hit | flat or ratio (legacy names không đồng nhất) | typed model/runtime legacy; combat state | model/simulator characterization |
| `active.curve_attack`; `summon_phantom` | action → enemies/summon | flat/count | combat legacy/partial; combat state | catalog + simulator characterization |
| `[parser-only] active.summon_growth_phantoms`; `[parser-only] active.summon_count_per_tick`; `[parser-only] active.phantom_damage_scale` | growth tick → summon | bool/count/ratio | parser/model only; chưa có persistence executor | catalog only, không parity |
| `active.disable_gu_next_turn`; `forget_reality_chance`; `target_damage_reduce_percent` | hit → self/target | bool/probability/ratio | simulator/statusfx; combat state | forget-gu + simulator |
| `active.portal_create`; `random_portal` | world use → self/world | bool | explore/world handlers; player/world tables | handler characterization |
| `[parser-only] active.capture_bonus` | world use → self/world | ratio | `handlers/combat_capture.go`; player/world tables | handler characterization. CSV hiện tại chỉ viết dạng block `CAPABILITY` + `UNIQUE CAPTURE_BONUS`; key legacy còn lại để đọc DB/import cũ |
| `active.cleansing_power`; `active.wood_transformation` | action → self | ratio/bool | combat legacy partial; combat state | catalog; parity chưa đầy đủ |
| `[parser-only] active.refine_mana_tier`; `[parser-only] active.replicate`; `[parser-only] active.generate_gu`; `[parser-only] active.invulnerable_ticks`; `[parser-only] active.absorb_damage_flat` | before refine/action → self hoặc generated entity | tier/bool/id/ticks/flat | model/manifest only; chưa có v2 executor/persistence command | catalog only, không parity |
| `active.reveal_all_stealth`; `active.reveal_weakness` | action/world inspect → targets/world | bool | migrate runtime v2; definition JSONB, chưa có persistence command | catalog only, không parity |
| `active.summon_monster_id`; `active.sleep`; `active.decoy_count` | action → summon/targets | entity id/bool/count | migrate runtime v2; combat/world owner chưa chốt | catalog only, không parity |
| `active.stat_base_attack`; `stat_base_defense`; `stat_base_hp` | always-on (đặt nhầm namespace) → self | flat | legacy player stats; `player_gu` | catalog |
| `active.description` | không trigger | text metadata, **không effect** | content only | catalog |
| `STAT_MOD ATTACK/DEFENSE/HP/SPEED` (`PASSIVE_STATS`, `ALWAYS_ON`) | always-on equipped ACTIVE → self | flat | player stats/combat snapshot; `player_gu` | player-stats + catalog |
| `STAT_MOD MOVEMENT/INVESTIGATION/HEARING/LUCK/DISGUISE/STORAGE` | always-on/capability → self | flat | player/world/inventory readers | catalog + handler characterization |
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
| `ON_CONSUME` | `MAN_BEAST_BURIAL` | self; yêu cầu người dùng Rank 1–4, Đỉnh Phong và rank Cổ trùng với rank người dùng; tăng 1 rank, về Sơ Giai, reset wall progress, giảm 10 Chân Nguyên tối đa (tối thiểu 20) | `gu-consume`; `players` | `TestManBeastBurialBreakthrough` + catalog |
| `ALWAYS_ON` | `DEDUCTION_INSTANT_FINISH` | self; `base_chance + (guRank - targetRank) * per_rank_step`, clamp `[min_chance, max_chance]`; 1 roll/lần thôi diễn, Cổ rank cao nhất thắng, cost vẫn trừ | `DeduceRecipe`; `player_recipe_progress` | `gamefx/balance_test.go` + catalog |
| `COMBAT_PASSIVE` | `REVERSE_CALAMITY` | self; đảo nghịch mọi trạng thái bất lợi nguồn địch về hư vô (mất lượt, cướp Cổ, phong ấn, debuff); chỉ sát thương xuyên qua | simulator (`status_apply`, interrupt/seal gates) + `handlePreCombatSteal`; combat state | simulator + catalog |
| `ON_BATTLE_START` | `SUPPRESS_PRIORITY` | self/side; vô hiệu `SEIZE_INITIATIVE` + `FIRST_TURN_EVADE` của địch; trấn áp thành công (ưu tiên địch đủ điều kiện kích hoạt) thì người mang đi trước Lượt 1; song Cấm thì suppressor nhanh hơn đi trước | simulator (`seizeInitiative`, suppress-first, evade gate); combat state | simulator + catalog |
| `ON_CONSUME` | `POISON_VOW`, `STONE_APERTURE`, `THIRD_WATCH` | self/world; params legacy | chưa đầy đủ, migrate v2; không được silent no-op | catalog only |
| `ON_CONSUME` | `SPRING_EGG_HATCH` | self/world; cấy 1 Phàm Tuyền tại ô đang đứng, chặn khi đủ cap/trong bí cảnh/đại vực đầy | `UseConsumableGu` → `springEggHatchEffect` (cần `tx`); `primeval_veins` | `TestGuSpringEggHatchDispatch` + `TestSpringEggHatchRegistered` |
| `ON_USE` | `RECORD_MESSAGE` | selected message target; text payload | support endpoint alias `RECORD`; persistent support state | handler + catalog |
| `ON_TICK` | `MANA_REGEN_TICK`, `THOUGHT_REGEN_TICK`, `SOUL_REGEN_TICK` | self; resource points/hour | resource regen worker; `players` | worker/model tests + catalog |
| `ON_TICK` | `GENERATE_STONES_TICK`, `GENERATE_IMMORTAL_STONES_TICK` | owner; currency/hour | hunger worker; wallet/player balances and Gu hunger | `worker/hunger_test.go` + catalog |
| `ON_DEATH` | `REBORN` | self; probability, ratio HP, count/turns | simulator; combat state | simulator + catalog |
| `BEFORE_ENEMY_ATTACK` | `FORGET_INTENT_INTERRUPT`, `FORGET_WILL_CONFUSE` | event attacker/targets; probabilities, turns, activation count | simulator; combat state | forget-gu tests + catalog |
| `ON_TURN_START` | `POEM_INTENT` | self/enemies; hook params | simulator; combat state | effect timing + catalog |
| `ON_BATTLE_START` | `SEIZE_INITIATIVE`, `SUMMON_WOLF_FRONTLINE`, `SUMMON_MONSTER` | self vs primary enemy / self; lifespan ratio hoặc hook params triệu hồi | simulator; initiative snapshot + triệu hồi một lần ở Lượt 0 | engine + catalog |
| `ON_BATTLE_START` | `INSTANT_WIN` | self/side; không params; hạ toàn bộ phe địch ngay Lượt 0, bỏ qua khiên, miễn nhiễm và Bất Tử | simulator (`applyBattleStartInstantWin`); combat state | `simulator/instant_win_test.go` + catalog |
| `ON_CAST` | `FULL_SOUL_HEAL`, `SWORD_DRAGON_TRANSFORM`, `BORROW_PAST_POWER`, `UNRAVEL_RIDDLE`, `CHAOS_GAMBLE` | hook-declared combat targets; typed params required | simulator; combat state | engine/effect timing + catalog |
| `ON_HIT` | `BYPASS_SHIELD`, `GUARANTEED_HIT` | event targets; bool | simulator; hit event (GUARANTEED_HIT bỏ qua Evasion/Blind/First Turn Evade/Forget Will Miss) | engine + catalog |
| `BEFORE_HIT` | `FIRST_TURN_EVADE` | self; bool/turn gate; Định Tiên Du rank 9 mang (rank 6–8 chỉ `portal_create` + `movement`) | simulator; combat state | engine + catalog |
| `PASSIVE` | `DOUBLE_DAMAGE_DEALT` | self; multiplier | simulator; combat snapshot (`COMBAT_PASSIVE` là tên v2) | engine + catalog |
| `CAPABILITY` | `MARKETPLACE_ACCESS`, `REMOTE_TRADE`, `CHAT_CONNECT`, `CHAT_PERCEIVE`, `IGNORE_MOVEMENT_RANGE`, `IGNORE_MONSTER_ENCOUNTER`, `LOOT_PREVIEW`, `DISGUISE`, `CHANGE_FORM_DISGUISE`, `PERFECT_DISGUISE`, `ARENA_IDENTITY_MASK`, `CONCEAL_GU`, `SEVER_CLUES`, `PORTAL_CREATE`, `RANDOM_PORTAL`, `CAPTURE_BONUS` | self/capability subject; bool/range/level hoặc `value` theo schema hook | marketplace/trade/chat/scouting/world handlers; owner domain tương ứng (`CAPTURE_BONUS` → `handlers/combat_capture.go`) | catalog + handler/scouting characterization |

**Matrix C — dictionary nhân vật, item và hai surface JSONB còn lại.**

| Nguồn / legacy key | Trigger → target | Unit / scaling | Executor hiện tại; persistence owner | UI label / test |
|---|---|---|---|---|
| Race/Constitution/Status: `stat_mods.{attack,defense,speed}` | always-on → self | ratio, cộng dồn | `gamefx.Get`; player/combat derived snapshot | `effects.ts`; `gamefx/fx_test.go`, player stats |
| Race/Constitution: `dao_mark.<path>`, `path_damage.<path>`, `path_resistance.<path>` | always-on/combat → self | ratio by path | gamefx + combat; dictionary + player selection | formatter; gamefx/combat tests |
| Shared: `mana_regen`, `max_mana`, `cultivation_efficiency`, `dao_mark_gain`, `refine`, `healing_received`, `mental_resistance`, `soul_defense`, `soul_regen`, `hp_regen`, `aperture_stress_bonus`, `tenacity`, `cc_resist` | always-on → self | ratios, trừ `hp_regen` flat HP/hour | gamefx consumers (regen/cultivate/refine/combat/tribulation); players | formatter có một phần; `gamefx/fx_test.go`, status/refine tests |
| Shared: `luck` | always-on → self | flat | gamefx/player stats | formatter + gamefx tests |
| Race: `immortal`, `disable_hp_regen`, `disable_mana_regen`, `disable_soul_regen`, `disable_resource_regen` | always-on → self | boolean | lifespan/regen policy; players.race | thiếu label cho một số flag; gamefx tests một phần |
| Status: `death_drop_mult`, `pvp_immune` | while status active → self | multiplier/bool | **không có field/executor: silent no-op hiện tại, phải fail khi compile v2** | chưa có label/test execution |
| Item: `hunger`, `restore_hp`, `heal_aperture`, `increase_aperture_area`, `increase_aperture_time_rate`, `add_dao_marks`, `increase_human_qi`, `random_constitution`, `random_race`, `open_mine_chest`, `open_immortal_mine_chest`, `spring_egg_hatch` | item use/feed → self/pet/aperture | flat hoặc command enum | item/pet handlers; inventory + players/aperture | `Item.name`; `handlers/item_effects_test.go` (coverage từng key chưa đầy đủ) |
| `killer_moves.effects` legacy keys | legacy move build/cast | nhiều đơn vị cũ, không đồng nhất | Không phải contract realtime. Combat Sát Chiêu dùng `compiled_effects`; chỉ một `ON_ACTION` đơn effect thuộc tập executor realtime hỗ trợ được đưa vào slot 9–12. `/killer-moves/:id/use` ngoài combat chỉ xử lý hồi phục `ON_USE` bản thân. | realtime loadout compiler; effect không hỗ trợ bị khóa |
| `aperture_variant_events.effects` | chưa có producer/trigger | chưa xác định | chỉ có JSONB owner event row; **không executor/persistence command** | không UI, không test; mọi payload hiện là no-op |

**Nguyên Tuyền (primeval vein):** bảng `primeval_veins` (`region_id`, `x`, `y`, `vein_type` `mortal`/`immortal`, `rarity`, `accumulated_since`, `produced_minutes`, `expires_at`). Mỗi mỏ có bể chung: ai đứng trong phạm vi 3 ô cũng thu hoạch được, nhận hết phần đã tích rồi bể reset. Trần tích lũy **1 ngày sản lượng**; trần vòng đời theo phẩm chất (Thường 7 / Hiếm 10 / Sử Thi 14 / Truyền Thuyết 21 / Siêu Cấp 30 ngày sản lượng); mỏ sống tối đa **30 ngày**. Phàm Tuyền nhả `mine_chest`; Tiên Tuyền nhả `mine_chest_immortal` (tỉ lệ rương decay 50%/2h, sàn 20% base — `veinChestDecay`, `handlers/primeval_vein.go`). Worker spawn tối đa **31 phàm + 5 tiên** (tiên chỉ ở vùng `min_rank ≥ 5`; mỗi đại vực ~1/5 cap), mỗi 30 phút bù 1 mỏ/loại. Boss rơi Cổ trùng `GU_SPRING_EGG` (Tuyền Đản Cổ) với tỉ lệ 1%; dùng Cổ này kích hoạt effect `SPRING_EGG_HATCH` để cấy Phàm Tuyền tại chỗ. Người chơi cấy bằng Cổ/vật phẩm **không** bị giới hạn cap 31 Phàm Tuyền toàn cục hay cap đại vực (cap chỉ áp cho worker tự spawn); vẫn chặn ô đã có địa điểm, vùng an toàn và bí cảnh.

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

Bản đồ chuyển đổi sang realtime MMO nằm ở `docs/mmo/gu-coverage.csv` (sinh bởi
`go run ./cmd/gudata audit-realtime`). Mỗi block có trạng thái
`SUPPORTED`/`NEEDS_REAUTHOR`/`UNSUPPORTED`/`NOT_ACTIVE` và lý do; skill chưa
audit **không** được cast như skill mất hiệu ứng. Sáu Gu MVP và quy tắc phân
loại nằm trong `docs/mmo/gu-coverage.md`; schema profile/compiler realtime nằm
trong `docs/mmo/gu-realtime-schema.md`.

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
  1. `DAMAGE`: sát thương phẳng (`FLAT`), hệ số **cộng dồn** (`MULTIPLIER`), lưu phái (`path`).
  2. `SHIELD`: khiên ảo Chân Nguyên Giáp (`FLAT`, `MULTIPLIER` **cộng dồn**).
  3. `HEAL`: hồi phục `FLAT`, `PERCENT`, `LIFESTEAL` hoặc `RESOURCE` (resource chọn bằng field `resource`).
  4. `DIRECT_HP`: sát thương trực tiếp `FLAT`, `MAX_HP_PERCENT` hoặc hủy khiên `SHIELD_STRIP`.
  5. `STAT_MOD`: biến đổi chỉ số nhân vật (`ATTACK`, `DEFENSE`, `SPEED`, `CRIT_RATE`, `ALL`...) dạng `ADD_FLAT` hoặc `MULTIPLY_PERCENT`.
  6. `STATUS`: gắn hiệu ứng trạng thái quản lý bởi `statusfx` (`status_id`, `value`, `turns`, `chance`). Né tránh dùng `status_id: "EVASION"` — `value` là % né, gộp max, cap 75%.
  7. `BARRIER`: bảo hộ có subtype `FLAT` hoặc `PERCENT` cùng thời lượng.
  8. `CLEANSE`: thanh tẩy `ALL`, `DEBUFF` hoặc một `STATUS` cụ thể.
  9. `UNIQUE`: cơ chế phi tuyến tính đặc thù (`code`, `params`).

*Khối `active` (44 key):*

> **Cộng dồn hệ số (damage_mult / shield_mult / `MULTIPLIER`):** các hệ số sát thương và khiên **cộng dồn** theo `1 + Σ(m−1)`, không nhân với nhau. Ví dụ cảnh giới lưu phái ×1.5 + `damage_mult` ×2.0 → `1 + 0.5 + 1.0 = 2.5×` (không phải `3.0×`). Buff chờ (pending) của Cổ thuần buff cũng vào cùng pool. Đạo Ngân (`daoMult`), weak/atk_up, `DOUBLE_DAMAGE_DEALT` và tăng sát thương theo hệ vẫn nhân như cũ.

| Key | Số Cổ | Kiểu | Tác dụng |
|---|---|---:|---|---|
| `damage_flat` | 207 | number | Sát thương cố định (tính sau modifier, trước shield/HP) |
| `damage_path` | 176 | enum | Hệ sát thương (`blood`, `Water Path`, `Sword Path`, `Fire Path`, `Ice Path`, … 20 giá trị) |
| `shield_flat` | 96 | number | Tạo khiên hộ thể (nhận damage trước HP) |
| `shield_mult` | 96 | number | Cộng dồn khiên (`1 + Σ(m−1)`) |
| `apply_status` | 450 | enum / object / **mảng** | Gắn status — mảng `[{"id","value","turns"}]` cho phép nhiều status mỗi action (phân bố xem mục Status chiến đấu) |
| `status_duration_ticks` | 4 | integer | (Legacy) Số lượt status; dữ liệu mới dùng `turns` trong entry |
| `status_tick_damage` | 1 | number | (Legacy) Damage mỗi tick; dữ liệu mới dùng `value` trong entry |
| `target_selector` | 55 | enum | Quy tắc chọn mục tiêu: `COL`(38), `ALL`(17), `ROW`(1) |
| `heal_flat` | 33 | number | Hồi HP cố định (cap max_hp) |
| `damage_mult` | 18 | number | Cộng dồn sát thương (`1 + Σ(m−1)`) |
| `mana_regen` | 2 | number | Hồi chân nguyên khi dùng |
| `armor_pen` | 2 | number | Xuyên Defense phẳng |
| `blood_drain` | 2 | bool | Cơ chế hút máu (+50% LifeSteal) |
| `cleansing_power` | 2 | number | Thanh tẩy (cấp độ) |
| `capture_bonus` | 15 | number | +% tỉ lệ bắt monster. Sau refactor Block v2 viết lại thành block `trigger: CAPABILITY` + `op: UNIQUE, code: CAPTURE_BONUS, params: {value}`; handler `combat_capture.go` đọc qua `models.GuSpecCapabilityFloat(spec, "CAPTURE_BONUS", "value")`. Đạo ngân Nô đạo nhân thêm `captureDaoFactor` (xem `combat_capture.go`): `1 + sqrt(enslavement_marks)/100`, áp cho cả roll bắt lẫn chỉ số thú bắt được. Mortal (rank < 6) không có `dao_marks` nên factor = 1.0 |
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
| `luck` + `cleansing_power` + `storage_capacity` + `attack_speed` + `damage_multiplier` + `damage_multiplier_on_hit` + `damage_reduction` + `damage_reflect` | 1–6 | mixed | Minor (xem struct) |
| `healing_multiplier` / `heal_received_multiplier` / `cold_resistance` / `water_resistance` / `dao_mark_multiplier` / `exclude_gu_storage` / `formation_power` / `refinement_power` / `storage_capacity_gu` / `target_selector`(=ROW) | 1 | mixed | Minor (xem struct) |
| `description` | 5 | string | Ghi chú text |

*Khối `growth` (5 key, 58 Cổ):* `max_attack`, `max_defense`, `max_hp`, `hours_to_max`, `max_speed` — chỉ số tăng trưởng khi nuôi, cộng dồn vĩnh viễn vào `players.growth_data.boosts` (+`base_hp`) qua growth worker (tick 1h) và flush khi unequip/sleep/seal. Block v2: `STAT_MOD ... GROWTH_CAP` + `UNIQUE GROWTH_DURATION_HOURS` (`{"id":"GROWTH","trigger":"ALWAYS_ON"}`). Cổ growth bị rớt khi migrate Block v2 đã được backfill từ CSV cũ.

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
| C - Persistent | `accuracy` | decimal | Tỉ lệ đánh trúng | **Đã gỡ** (xem `apply_status EVASION`) |
| C - Persistent | `evasion` | decimal | Tỉ lệ né | **Đã gỡ** → `apply_status` `status_id: "EVASION"` |
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
| HASTE, ARMOR_BREAK, DAMAGE_REDUCTION_UP, INVINCIBLE, LAST_STAND, EVASION, REVIVE_LOCK | Buff/Debuff | Không | **CÓ (registry)** |

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
| `passive.evasion` / `STAT_MOD EVASION` / `self_evasion_boost_percent` | `apply_status: [{id:"EVASION",...}]` | **Đã migrate xong** — `EVASION` giờ là status tự buff có `chance`/`turns`/`value`, nguồn né duy nhất trong combat |
| `passive.accuracy` / `STAT_MOD ACCURACY` | — | **Đã gỡ khỏi hệ thống** (chỉ số trừ thẳng vào xác suất né 0..1 nên vô hiệu hoá né tránh) |
| `invulnerable_ticks` (miễn nhiễm cũ) | `apply_status: [{id:"INVINCIBLE",...}]` | **Đã migrate xong** — `INVINCIBLE` nay là Bất Hoại chặn **mọi** đường sát thương (kể cả DoT / %MaxHP), không chỉ đòn đánh thường |
| (chưa có) | `apply_status: [{id:"LAST_STAND",...}]` | **Status mới**: Bất Tử — nhận sát thương nhưng HP sàn 1; `CategoryImmortal` nên bị tẩy trừ được |

### Status chiến đấu (combat)

Status chiến đấu realtime được quản lý theo profile status có thời hạn trong world runtime; server quyết định áp dụng, cộng dồn và hết hạn theo tick. `docs/STATUS_EFFECTS.md` còn giữ lịch sử mô hình theo lượt cũ, không phải contract runtime hiện hành.

**14 status từng được định nghĩa cho combat theo lượt (bảng lịch sử, không phải contract realtime):**

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

Lưu ý: `bleed`/`poison`/`slow_damage` dùng **giá trị số thật** (không qua tier). DoT và duration trong mô hình theo lượt cũ không còn là runtime contract; status realtime dùng duration/tick theo profile trong `statusfx` và world runtime.

**Status có trong data, đã wire qua registry `statusfx`** (xem `docs/STATUS_EFFECTS.md` → "Registry status"): `BURN` (DoT hỏa), `FEAR` (force basic), `CONFUSE` (40% mis-target), `STEALTH` (untargetable), `SLEEP` (skip turn + break on damage), `BLIND` (35% miss), `CHARM` (điều khiển), `BIND` (mất lượt).

**Status catalog đã wire qua registry**: `HASTE` (tăng tốc), `ARMOR_BREAK` (giảm thủ), `DAMAGE_REDUCTION_UP` (giảm sát thương nhận), `INVINCIBLE` (Bất Hoại — miễn nhiễm mọi đường sát thương, không dispel), `LAST_STAND` (Bất Tử — nhận sát thương nhưng HP sàn 1, **bị dispel**), `EVASION` (tự buff né tránh — `value` là % né, gộp max, cap 75%), `REVIVE_LOCK` (chặn hồi sinh). Cả `INVINCIBLE` và `LAST_STAND` đều bị Sát Khí tầng $>10$ xuyên.

**Quy tắc gộp hiệu ứng (Consolidation)**:
- `disable_gu_next_turn` và `FORGET_EGO` được quy về trạng thái chuẩn `GU_SEAL` (`StatusGuSeal`).
- `target_damage_reduce_percent` được quy về trạng thái suy yếu sát thương `WEAK` / `FORGET_REALITY`.
- `SLOW1` được quy về trạng thái làm chậm chuẩn `SLOW` (hỗ trợ điều chỉnh độ mạnh qua value/tier).

**Tỉ lệ áp dụng hiệu ứng (Rank Scaling for Self & Enemy)**:
- **Tỉ lệ theo Rank Gu**: Mỗi cấp Rank Gu trên Rank 1 tăng $+5\%$ xác suất proc: $\text{chance}_{\text{gu}} = \text{baseChance} + 0.05 \times (\text{guRank} - 1)$ (tối đa $95\%$).
- **Hiệu ứng Bản thân (Self-Target)**: Tỉ lệ thành công tỉ lệ thuận với Cảnh giới người dùng (Player Rank): $\text{factor}_{\text{self}} = 1.0 + 0.05 \times (\text{casterPlayerRank} - 1)$.
- **Hiệu ứng Đối phương (Enemy-Target)**: Xác suất chịu ảnh hưởng của chênh lệch Cảnh giới ($\text{casterRank} - \text{targetRank}$): Chênh lệch $\ge +2$ ($\times 1.5$), $+1$ ($\times 1.2$), $0$ ($\times 1.0$), $-1$ ($\times 0.6$), $\le -2$ ($\times 0.1$). Tiên Cổ đánh Phàm Cổ áp dụng hệ số tối thiểu $1.0\times$ (tự động nâng lên $1.5\times$).

Thêm status realtime mới: khai báo trong `statusfx.Registry` và nối behavior vào realtime status runtime.

### Item

Item dùng cho consumable, material, currency và shop. Effects hỗ trợ hồi mana/HP, soul, đạo ngân, xóa trạng thái xấu, điều kiện mortal/immortal hoặc các tác động đặc thù khác. **Quy ước `base_price`:** hỗ trợ tối đa 3 chữ số thập phân (đơn vị nhỏ nhất `0.001`); fallback theo rank — 1, 3, 6, 13, 25, 500, 1.250, 3.000, 7.500 cho rank 1–9 (rank 1 = 1, scale tỉ lệ như Cổ); item rank 0 quy về rank 1.

Các effect key đặc biệt của Consumable (resolve trong `backend/internal/handlers/item_effects.go`, map `ApplyItemEffects`):

- `random_constitution` (value `1`): khi dùng sẽ random **thể chất** mới từ `gamefx.AllConstitutions()`, loại trừ thể chất hiện tại của người chơi. Kết quả được ghi vào `players.constitution`.
- `random_race` (value `1`): khi dùng sẽ random **chủng tộc** mới từ `gamefx.AllRaces()`, loại trừ chủng tộc hiện tại. Kết quả được ghi vào `players.race`.
- `increase_aperture_area` (value N): cộng thẳng N vào `immortal_apertures.area_size` (đơn vị mẫu, cùng thang với `area_size` mặc định 3.000.000–10.000.000). Chỉ Cổ Tiên có Tiên Khiếu mới dùng được.
- `increase_aperture_time_rate` (value N): cộng thẳng N vào `immortal_apertures.time_rate` (hệ số gia tốc, mặc định 10–55). Chỉ Cổ Tiên có Tiên Khiếu mới dùng được.
- `add_dao_marks` (value N): cộng N Đạo Ngân vào `immortal_apertures.dao_marks` của **lưu phái ghi trong cột `path` của item** (path đầu tiên, chuẩn hoá qua `gamefx.CleanPath`). Ghi vào `player.DaoMarks` rồi được `UseItem`/`UseItems` persist xuống `immortal_apertures` (giống `soul_boost`). Cần `immortal_only` + Tiên Khiếu tồn tại.

**Đạo Tài (`dao_material_<path>`, rank 6, Consumable):** 44 item sinh sẵn trong `Item.csv`, mỗi phái một món (`path` = `["Fire Path"]`…). Mỗi món khi dùng bổ sung **1.000 Đạo Ngân** đúng phái đó (`IMMORTAL_ONLY` + `ADD_DAO_MARKS` value 1000). Đây là vật hoàn trả của Thanh Đạo Tẩy Thể (30% Đạo Ngân bóc tách, 1.000 Đạo Ngân/món) — xem WIKI §3. Cần import bảng "Vật phẩm" sau khi thêm phái mới.

Hai item mẫu trong `Item.csv`: `ITEM_RANDOM_CONSTITUTION` (Hoán Thể Đan) và `ITEM_RANDOM_RACE` (Hoán Huyết Đan), cùng rank 7 / base_price 200.

Hai đan dược rơi từ **boss rank 8–9** (xem `LootTable.csv`):

- `ITEM_APERTURE_EXPANSION_PILL` (Thác Khiếu Đan, rank 9): `INCREASE_APERTURE_AREA` +500.000 diện tích Tiên Khiếu.
- `ITEM_APERTURE_TIME_PILL` (Thôi Thời Đan, rank 9): `INCREASE_APERTURE_TIME_RATE` +2 tốc độ thời gian Tiên Khiếu.

Cả hai gắn entry `rank` bằng đúng rank boss (8 hoặc 9), base chance `0.04` (rank 8) / `0.03` (rank 9). Qua hệ số loot (`gamefx.LootChanceMultiplier` / `LootQuantityMultiplier`) tỉ lệ thực tế là ~10%/12%, số lượng 5 viên (rank 8) hoặc 8 viên (rank 9) mỗi lần rơi. Danh sách bảng: 12 bảng loot của boss rank 8 trong `MapCell.csv` (`loot_monster_*_thai_co_r8r9`, `loot_monster_blood_sword_asura_lord`) và 3 bảng boss rank 9 (`loot_monster_immemorial_refinement_golem`, `loot_monster_tide_dragon_emperor`, `loot_monster_white_light_divine_sovereign`).

### Recipe

Recipe yêu cầu Cổ instance và/hoặc material. Một số recipe cần suy diễn bằng `/api/refine/deduce` trước khi luyện ổn định.

- `recipe_id` (dùng ở `player_recipe_progress`, `sect_vault_recipes`, `aperture_refinement_queue`, shop/trade/marketplace) là khóa dẫn xuất `recipe_<target_gu_id>_<target_rank>_<variant>`. Nguồn chân lý vẫn là bộ ba `(target_gu_id, target_rank, variant)` — khóa chính của `refining_recipes`. Logic gameplay (bậc, rarity, tên, điều kiện tông môn, giá) phải tra `refining_recipes`; không suy ra từ việc parse chuỗi, vì Gu ID có thể kết thúc bằng số. Tiền tố `recipe_` giữ nguyên để phân biệt namespace với Gu/item ID trong các cột ID đa hình.
- Cột `player_recipe_progress.recipe_id` là `VARCHAR(100)` (đồng bộ với `sect_vault_recipes`), để recipe mới có tên dài không làm fail insert.
- Cột `required_gus` là JSON map `gu_id → {rank, qty}`. `rank` là bậc Cổ nguyên liệu bắt buộc; `rank = 0` nghĩa là chấp nhận mọi bậc (chỉ dùng cho dữ liệu legacy — mọi recipe import từ CSV đều được chuẩn hóa sang rank cụ thể).
- Khi import từ CSV (`seedRecipesFromCSV`), `required_gus` được tự chuẩn hóa: value dạng số `N` cũ được chuyển thành `{rank: <base_rank>, qty: N}`, trong đó `base_rank` là bậc nhỏ nhất của Cổ đó trong `gu_dictionary` (fallback `1` nếu Cổ chưa có trong dictionary).
- Backend chỉ chấp nhận Cổ nguyên liệu khớp cả `gu_id` lẫn `rank` (khi `rank != 0`); frontend (Lò Luyện, Wiki, Admin) hiển thị và chọn Cổ theo đúng bậc yêu cầu.
- Ví dụ: `{"GU_WHITE_BOAR": {"qty": 1, "rank": 1}}`.

### RefineStage (nhịp độ phiên luyện cổ)

- Bảng `refine_stage_templates` (khóa `rank` 1–9) cấu hình nhịp độ + độ khó của phiên luyện Cổ theo thời gian (`refine_sessions`, WF-REF-06): `duration_seconds`, `step_count` (N), `minigame_count` (K), `window_seconds`, `accuracy_max_bonus`, `death_penalty`, `mastery_multiplier_max`, độ khó từng minigame (`tap_half_width`, `tap_period_ms`, `sequence_length`, `sequence_display_ms`, `sequence_input_ms`, `order_item_count`, `choice_option_count`) và trọng số `minigame_weights` (`{"TAP":1,"SEQUENCE":1,"ORDER":1}`).
- Nguồn CSV: `RefineStage.csv` (16 cột, đúng 9 dòng rank 1–9; `minigame_weights` phải quote JSON). Import: Admin → "Dữ liệu" → "Nhịp độ luyện cổ" → Dry-run → Import. Không cần restart: handler đọc bảng mỗi lần tạo phiên; phiên đang chạy giữ snapshot `rate_snapshot` tại lúc tạo nên import giữa phiên không đổi kết quả.
- Ngoài bảng theo rank, tổng thời gian còn nhân `RefineSessionTimeScale(realm)` theo cảnh giới Chân Ý Luyện Đạo của player tại lúc tạo (constants `gamefx/balance.go`: `RefineRealmTimeDiscountPerLevel = 0.05`/cấp, cap `0.20`, sàn `0.50`; unit test `refine_session_scale_test.go`). Cửa sổ thao tác không đổi; guard `duration ≥ window × N` chặn cấu hình cực đoan.
- Baseline 9 dòng được seed bằng Go khi `InitSchema` (`ON CONFLICT DO NOTHING`) — khôi phục mặc định bằng cách import lại `RefineStage.csv` baseline. Bảng thiếu rank → `POST /refine/sessions` trả **503** `REFINE_CONFIG_MISSING`.
- Ràng buộc preflight: `minigame_count ≤ step_count − 1`, `window_seconds ≤ duration_seconds / step_count`, weights chỉ chứa TAP/SEQUENCE/ORDER với tổng > 0.

### RefineDecision (kịch bản dẫn hỏa)

- Bảng `refine_decisions`: các dòng CHOICE cho bước "Dẫn Hỏa" trong phiên luyện — `id, situation, option_1..3, correct_index, neutral_index, explanation` (CSV `RefineDecision.csv`).
- Import qua Admin ("Kịch bản dẫn hỏa"); chọn dòng theo seed phiên, xáo thứ tự option. Registry rỗng → các bước CHOICE **fallback thành TAP** cùng độ khó (không chặn người chơi) kèm log. Không seed baseline lúc startup.

### Monster/combat

Monster template chứa chỉ số, Gu inventory (`gu_inventory`) và Gu pool (`gu_pool`), item inventory, dao marks, path masteries, AI profile và loot table. Combat handler dùng dữ liệu này để khởi tạo trận, chọn hành động và roll loot.

#### Bảng scale dùng chung (RankScaleTable)

- `models.RankScaleTable` (`internal/models/rank_scale.go`) là **nguồn duy nhất** cho thang 9 bậc: `1, 2, 5, 20, 100` (phàm) và `10^4, 10^6, 10^8, 10^10` (tiên, ×100 mỗi bậc). `gamefx.RankScale` chỉ là wrapper gọi lại — không định nghĩa bảng thứ hai.
- Gu values và chỉ số quái đều là `<ratio> × RankScale(rank)`. Median Gu (`value / RankScale`) giống hệt nhau ở mọi bậc: DAMAGE 20, HP 10, ATTACK 1, DEFENSE 5. Gu utility dùng ratio 8.
- **`Monster.csv` là rank-independent.** `base_hp/base_attack/base_defense` không được tăng theo bậc; `models.ScaleMonsterStats` nhân `base × RankScale(rank) × typeMultiplier × subRankMultiplier` (rồi `× MONSTER_STAT_MULTIPLIER` nếu env này khác 1.0). Median base mọi bậc bằng median bậc 1 (90 HP / 12 ATK / 5 DEF). `base_defense` được cap ở **10** (≤ 50% ratio sát thương Gu chuẩn 20) để không có quái nào bất khả chiến bại dưới damage trừ trực tiếp. Thêm quái mới: chọn base quanh mức rank-1 (Elite/Boss dựa vào `typeMultiplier` 1.5/3.0), **không** nhân theo rank — nếu không sẽ double-count và quái immortal thành bất khả chiến bại.
- **Cờ tắt scale rank:** `MONSTER_RANK_SCALING_ENABLED=false` (hoặc `0`/`off`/`no`) bỏ hẳn `RankScale(rank)` — và cả bump rank 10 — nhưng vẫn giữ `typeMultiplier` và `subRankMultiplier`. Mọi đường spawn (world monster, preview vùng, linh thú tông môn, boss phi thăng/độ kiếp) đi qua cùng một hàm nên tự khớp nhau.
- **Hệ số chỉnh toàn cục:** `MONSTER_STAT_MULTIPLIER` (mặc định `1.0`) nhân vào cả HP/ATK/DEF **sau** mọi hệ số khác, dùng để nerf/buff quái tạm thời mà không sửa CSV. Giá trị thiếu/không hợp lệ/≤ 0 đều rơi về `1.0`. Cả hai cờ là env nên cần restart backend; quái đang spawn giữ chỉ số cũ tới khi respawn.
- Time-to-kill 1 Gu cast chuẩn (~6 hit) được khoá bằng test `TestMonsterTTKBandAcrossRanks` (`internal/models/rank_scale_test.go`).

#### Loot table (bảng rơi đồ)

- Mỗi monster template trỏ tới một bảng override (`Monster.loot_table_id`) — nơi chứa đồ **đặc thù loài** và bonus riêng. Ngoài ra, mọi trận PvE với quái đều roll thêm **bể chung theo bậc** `loot_monster_common` (2 entry/bậc 1–9: `beast_meat_rankN` 60%, `heaven_qi_rankN` 50%, qty 1–2 × hệ số rank). `RollMonsterLoot` (`handlers/combat_loot.go`) = bể chung + override; entry `gu_id` vẫn roll như cũ.
- Mỗi `LootEntry` có thêm 2 field JSON (Admin sửa được, xem modal "Bảng Loot"):
  - `rank` (0–9): **0 = phổ quát/đặc thù loài** (luôn roll, không phụ thuộc rank quái); **N = chỉ rơi khi `EnemyRank == N`** → đảm bảo quái thưởng tài nguyên đúng bậc. `RollLoot` lọc theo `lootEntryEligible`.
  - `jackpot` (bool): **luôn roll dù lệch rank** với `chance` riêng — dùng cho hiếm phẩm bonus vượt bậc (mặc định `false`; chọn thủ công trong Admin).
- Quy ước dữ liệu `LootTable.csv`: item `*_rankN` (thịt yêu thú, thiên khí, thiếc, nước, thạch âm, bùn đất, hắc du, kiếm thảo… 9 bậc) mang `rank = N`; item đặc thù (tơ nhện, lông thú, kén…) và tiền tệ đặc biệt (`primeval_stone`, `immortal_stone`, `white_light_essence`) mang `rank = 0`. Hiện 672/956 entry là universal (theme), phần còn lại gắn bậc.
- **Nhân Khí Đan (`HUMAN_QI_PILL`, rank 9):** rơi 5% (1 viên) từ loot override của 11 boss rank 5 — 10 boss trong `MapCell.csv` cộng `monster_three_kings_boss` (Tam Vương Phúc Địa, seed trong `db.go`) — entry gắn `rank = 5` nên chỉ rơi đúng bậc.
- **Spawn quái:** toàn bộ 209 monster trừ 6 con sinh bởi hệ kiếp (`monster_tribulation_beast`, 5 `*_calamity_boss`/`*_tribulation_boss`) đều có dòng `MonsterSpawn` — 50 con trước đây thiếu đã được bổ sung vào `MonsterSpawn.csv` (boss → `Fixed` có tọa độ, Elite → `Patrol`, Normal → `Roaming`). Rank spawn của chúng lấy theo bậc vật phẩm trong bảng loot (max `_rankN`/item rank, fallback theo vùng).

#### Loot theo vùng (Region loot — lớp thứ 3)

- Loot của một trận PvE gồm **3 lớp độc lập** (cộng dồn, `RollMonsterLoot` trong `handlers/combat_loot.go`):
  1. **Bể chung theo bậc** `loot_monster_common` (mọi quái).
  2. **Override đặc thù loài** `Monster.loot_table_id`.
  3. **Loot theo vùng** — bảng `region_loot_tables`, áp dụng theo `region_id` + `monster_type` (Normal/Elite/Boss…) + dải `min_rank`/`max_rank`.
- Bảng `region_loot_tables` (CSV `RegionLootTables.csv`, dataset "Loot theo vùng") là mapping `region_id, loot_table_id, monster_type, min_rank, max_rank, chance, is_active`; PK `(region_id, loot_table_id, monster_type)`. Mỗi rule có `chance` kích hoạt riêng (mỗi rule là một lần roll độc lập, không chia sẻ ngân sách xác suất với lớp khác). `monster_type` so khớp **không phân biệt hoa/thường** (`lower(monster_type) = lower($2)`), nên `EnemyType` là `Boss` hay `BOSS` đều khớp.
- CSV chỉ chứa mapping; bảng loot đích (đặt tên `loot_monster_<region_id>_<normal|elite|boss>`) khai báo trong `LootTable.csv` như bảng loot bình thường, `entries` tham chiếu `item_id` có thật trong `Item.csv`.
- **Dải rank là của rule, không lọc lại theo `entry.Rank`:** `RollRegionLoot` gọi `rollRegionLootTable` (bỏ qua cổng `entry.Rank`) vì `min_rank`/`max_rank` của rule đã giới hạn bậc. Nếu áp tiếp luật `entry.Rank` (chỉ đúng cho lớp 1–2) thì một rule phủ nhiều rank mà bảng loot chỉ gắn 1 rank sẽ rớt rỗng ở các rank còn lại.
- **Kế thừa theo phân cấp vùng:** `RollRegionLoot` tra cả `region_id` hiện tại **và tổ tiên** (`parent_id`), nên rule đặt ở đại vực (NGU_VUC) tự áp dụng cho mọi KHU_VUC/SAFE_ZONE con.
- **Region loot realtime:** world runtime gọi `backend/internal/realtime/loot` khi có authoritative monster death; dịch vụ dùng template, vùng, loại quái và rank để roll idempotent, không đi qua CombatState/encounter theo lượt.
- **Khoảng trống dữ liệu đã biết:** 5 đại vực (`dai_vuc_bac_nguyen`, `dai_vuc_dong_hai`, `dai_vuc_nam_cuong`, `dai_vuc_tay_mac`, `dai_vuc_trung_chau`) và các safe-zone `tan_thu_coc_*` có spawn nhưng không có dòng `RegionLootTables` nào → quái spawn trực tiếp trong đó không rơi region loot (~40 spawn). Thêm rule cấp đại vực sẽ phủ luôn safe-zone con, nhưng sẽ **cộng dồn** cùng rule sẵn có của các KHU_VUC con (do kế thừa).

#### Cổ trùng trên quái (monster/region Gu pool)

- **Dữ liệu pool Cổ trên quái** — template lưu `gu_inventory`/`gu_pool`, vùng lưu `pool_weights.monster_gus`. Service loot realtime chọn pool riêng của loài trước; nếu pool này rỗng, nó hợp nhất pool vùng từ vùng hiện tại lên cha (con thắng cha theo `gu_id`). Cổ trong `gu_inventory` được xem là Cổ quái mang theo; Cổ từ pool được chọn theo trọng số và giới hạn slot, rồi roll tỉ lệ rơi theo rank Cổ và loại quái. Các lượt này nằm cùng giao dịch/idempotency với loot khác.
  1. **Pool riêng loài** `monster_templates.gu_pool` (JSONB, cột `gu_pool` trong `Monster.csv`, Admin sửa được).
  2. **Pool theo vùng** `regions.pool_weights.monster_gus` (kế thừa `parent_id`, con thắng cha theo `gu_id`; resolve ở `handlers.regionMonsterGuPool`).
  3. Auto-match từ registry/Đạo Ngân đã bị loại bỏ cùng builder turn-based; chỉ dùng các pool được khai báo.
- Quái **Normal không có pool nào thì không mang cổ** (giữ nguyên cân bằng cũ). Muốn Normal rơi cổ phải khai pool ở `gu_pool` hoặc `monster_gus`.
- `GuPoolEntry` (dùng chung cho cả 2 pool): `gu_id`, `rank`, `weight`, `min_count`, `max_count`, `is_world_boss`. Entry `rank > EnemyRank` bị lọc; entry `is_world_boss` chỉ dùng cho drop world-boss (không trang bị thường).
- **Số slot trang bị theo loại:** Normal = 1; Elite = 3 (2 active + 1 passive); Boss = 5 (3 active + 2 passive). Pool nhiều hơn slot → chọn ngẫu nhiên theo `weight`; `min_count`/`max_count` cho phép nhân bản một cổ.
- **Luật rơi Cổ realtime:** mỗi Cổ được chọn từ pool có một lần roll độc lập theo rank Cổ và tier quái:
  - `boss` > `elite` > `normal`. Normal thấp hơn Elite, ví dụ R1 15% / R2 10% / R3 6% / R4 3% / R5 1.5%; R6+ ~0.005–0.05%.
  - Tier suy ra từ `EnemyType` qua `guDropTierForCombat` (Boss/World_Boss/Mine_PVE → boss; Normal/Monster → normal; còn lại → elite).
- **Tiên Cổ (rank ≥ 6):** luật unique toàn server được kiểm tra trong giao dịch cấp thưởng. Pool vùng bị lọc rank ≥ 6 khi import (`filterImmortalMonsterGus`); pool loài vẫn chịu kiểm tra registry và tính duy nhất khi grant.
- CSV: `Monster.csv` cột `gu_pool` (import/export đầy đủ); `Region.csv` cột `pool_weights` chứa `monster_gus`. Admin sửa qua modal quái (`gu_pool` JSON) và modal vùng (`pool_weights` JSON).
- Map Tiled có thể khai báo property số nguyên `rank` trên spawn quái; giá trị này được chuyển qua death event tới loot realtime. Nếu không khai báo, service dùng `regions.min_rank` của vùng, rồi fallback rank 1 nếu region chưa có dữ liệu.

### Canonical non-player actors

`non_player_actors` là bảng canonical cho **Actor Instance** persistent. Nó tham chiếu template qua cặp đa hình `template_source/template_id` thay vì sao chép toàn bộ `monster_templates`; các nguồn hiện có gồm `MONSTER_TEMPLATE`, `VARIANT_RACE`, `NPC_TEMPLATE`, `CUSTOM`. `actor_kind` (`MONSTER`, `COMPANION_BEAST`, `VARIANT_NPC`, `WORLD_NPC`, `SECT_NPC`, `CONSTRUCT`, `BOSS`) chỉ biểu diễn identity/lore và default policy, không phải quyền thực thi hành vi.

Các nhóm dữ liệu chính gồm ownership (`owner_player_id`, `faction_id`, `aperture_id`), identity/template, progression, combat stats, pack/hunger/upkeep, lifecycle/execution, behavior/AI và ba document JSONB `capabilities`, `traits`, `progression`. Các cột `formation_slot`, `path`, `role` phục vụ trực tiếp các truy vấn gameplay thường xuyên. Bảng legacy `player_pets`/`aperture_variant_npcs` đã xóa hoàn toàn; chỉ còn `non_player_actors`. Dân số aggregate `aperture_variant_humans` không trở thành từng actor.

**Execution invariant:** tại một thời điểm, một actor chỉ có đúng một `execution_state` có quyền thay đổi gameplay. Graph tối thiểu là:

```text
STORED -> DEPLOYING | TRAINING | WORKING | RECOVERING | DEAD | DISMISSED
DEPLOYING -> WORLD_IDLE
WORLD_IDLE -> MOVING | FOLLOWING | GUARDING | PATROLLING | GATHERING | COMBAT | RETURNING | INJURED
COMBAT -> WORLD_IDLE | RETURNING | INJURED | DEAD
```

Mọi transition ngoài graph bị từ chối. `lifecycle_state` là coarse persistence lifecycle (`ACTIVE`, `DEAD`, `DISMISSED`) và bị constraint đồng bộ với terminal execution state; nó không phải execution owner thứ hai. Constraint khác bảo đảm `count >= 1`, `hp >= 0`, `max_hp > 0`, `version >= 1`, actor thuộc nhóm owned có owner, và `hunger` nằm trong `[0,max_hunger]` khi `FEEDABLE=true`. Index bao phủ owner/kind, aperture/kind, execution state, template identity, actor active và legacy identity.

`capabilities` là map boolean authoritative, hỗ trợ: `MOVE`, `FOLLOW`, `PATROL`, `GUARD`, `COMBAT`, `PRODUCE`, `REFINE`, `GATHER`, `TRAIN`, `FEEDABLE`, `CAPTURABLE`, `STACKABLE`, `EQUIP_GU`, `USE_SKILL`, `TRADE`, `QUEST_INTERACTION`, `MORTAL`, `REVIVABLE`. Command/handler phải kiểm tra capability thay vì phân nhánh hành vi theo `actor_kind`.

`non_player_actors` là nguồn đọc/ghi duy nhất cho sủng vật và NPC dị nhân có tên. API `/pets/*` và `/immortal-aperture/npcs` vẫn giữ DTO cũ để không phá client. `monster_templates` tiếp tục là template chuyên biệt cho quái/sủng thú; NPC dị nhân dùng `VARIANT_RACE`. Không dùng `monster_templates` làm template chung cho mọi non-player: khi shop/quest/sect NPC cần archetype tái sử dụng sẽ bổ sung `NPC_TEMPLATE`; cá thể độc nhất dùng `CUSTOM`. `monster_spawns` vẫn là spawn definition, còn `world_actor_presence`, command mutation và assignments thuộc các release tiếp theo.

`actor_gu_equipment` giữ trang bị Cổ của actor. Mỗi instance Cổ chỉ thuộc tối đa một actor, mỗi actor có tối đa một Cổ trên mỗi slot; `player_gu.location='ACTOR'` khiến Cổ không xuất hiện trong inventory sử dụng của người chơi. NPC chỉ dùng Cổ đang sống, còn độ no và không cao hơn rank của NPC.

NPC `MINER` và `FARMER` tăng sản lượng node phù hợp; NPC `REFINER` tăng xác suất hàng đợi luyện Cổ. Mỗi NPC đóng góp `0.005 × rank × talent% × loyalty%`; production cap `+50%`, refinement cap `+25%`. Tu luyện NPC là command thủ công, tiêu `rank × sub_rank × 100` Nguyên Thạch, nhận `20 + talent/2` kinh nghiệm (tối thiểu 10), cần `rank × sub_rank × 100` để tiến một tiểu cảnh giới và tạm giới hạn ở Rank 5 Đỉnh Phong.

### Region / Bản đồ thế giới (World Map)

**Mật độ quái realtime:** `scripts/convert_regions_to_mmo.py` sinh 120 đàn/điểm sinh trên 1.000.000 đơn vị diện tích khi trọng số Monster là 30/100 (khoảng 420 quái/map chuẩn, phân bố đa dạng gồm 30% đi lẻ 1–2 con, 40% nhóm 3–4 con và 30% đàn 5–6 con với độ tản bán kính lên tới 10 ô); số điểm sinh tăng/giảm theo tỷ lệ Monster trong `Region.csv`. Chạy lại script để cập nhật đồng thời artifact backend/frontend, sau đó restart backend và tải lại client. Import CSV riêng lẻ không sinh lại artifact realtime. Runtime kích hoạt quái trong bán kính 64 quanh người chơi, thu hồi ngoài 96 sau 10 giây; catalog tối đa 1.200 quái/map.

`WorldMap` đại lục hiển thị trên canvas đại giới 15.000 × 10.000 (Atlas) gồm 5 Đại Vực: hàng trên là Nam Cương (atlas_x=0–4.999), Trung Châu (5.000–9.999), Bắc Nguyên (10.000–14.999); hàng dưới là Tây Mạc (atlas_x=0–7.499) và Đông Hải (7.500–14.999). Tây Mạc và Đông Hải đều có atlas_y từ 5.000 đến 9.999. Toàn bộ các `SAFE_ZONE`, `HUB` và `KHU_VUC` con đều áp dụng hệ tọa độ cục bộ (local coordinates) chuẩn MMO bắt đầu từ (0, 0) với kích thước đồng nhất 1.000 × 1.000 (`min_x=0, min_y=0, max_x=1000, max_y=1000, width=1000, height=1000`), điểm vào trung tâm là (500, 500). Vị trí hiển thị trên bản đồ thế giới vĩ mô (World Atlas) được lưu trữ qua hai cột `atlas_x, atlas_y` trong `Region.csv` và bảng `regions`. Các map MMO và map cells đều được chuẩn hóa theo hệ tọa độ local này. Vùng kỹ thuật cũ `ngoai_vuc_tang_nhat` được giữ inactive để tương thích dữ liệu cũ. Bạch Thiên là tầng 2 (`bai_thien`), Hắc Thiên là tầng 3 (`hac_thien`); hai tầng và region Đại Vực tương ứng hiện được seed ở trạng thái khóa để chờ luồng lên tầng. Mười Khu Vực con cũ của hai Đại Vực được chuyển thành `SECRET_REALM` trên tầng 1; mỗi vùng có một template `SPATIAL_REALM`, một instance đang hoạt động dùng chung cho toàn server, một cổng vào cố định trên bản đồ thế giới và một cổng thoát trong bí cảnh. `Region.csv` import tự seed template và hai loại cổng. Backend chỉ cho vào khi người chơi đứng đúng ô `ENTRY_PORTAL` gắn với template; UI hiển thị vị trí cổng và khóa nút vào ở nơi khác. Tọa độ của mỗi bí cảnh là cục bộ trong bản đồ instance, không chiếm vùng tọa độ của thế giới. Ngoài `Southern_Border` (HUB mặc định), có HUB khởi đầu riêng (`hub_thuong_gia_thanh`, `hub_trung_chau`, `hub_bac_nguyen`, `hub_tay_mac`, `hub_dong_hai`). Khi hoàn tất onboarding (`complete-onboarding`), người chơi được đặt vào HUB theo vùng đăng ký (xem `startingRegionHubs` trong `player.go`), không sinh quái trong HUB.

**Marker công trình tĩnh.** Mọi công trình có vị trí trên map đều dùng chung khoá `(region_id, x, y)` với ô local của zone: `sects` (`region` + `map_x/map_y`), `teleport_arrays`, `primeval_veins`, `merchants`, `map_cells`. `GET /api/world/map-objects` gộp `sects` + `teleport_arrays` vào một contract `MapMarkerView` (`id`, `kind`, `region_id`, `x`, `y`, `name`, `detail`); các thực thể động (Nguyên Tuyền, đồ rơi) vẫn đi qua `/explore/cells/rect` vì cần cửa sổ toạ độ và làm mới nhanh. Client hợp nhất hai nguồn thành một danh sách marker dùng chung cho lớp Phaser, radar và panel tương tác gần — thêm loại công trình mới chỉ cần thêm một nhánh trong collector, không phải sửa từng renderer. `sect_mines` đã bị loại bỏ: tài nguyên ngoài map nay do `primeval_veins` đảm nhiệm.

**`world_maps` và `region_world_maps`** là danh mục tầng và liên kết region với tầng. Ba bản ghi tầng được seed theo thứ tự `world_map` (tầng 1), `bai_thien` (tầng 2), `hac_thien` (tầng 3); năm ID map realtime (`realtime_preview`, `misty_forest`, `bone_crypt`, `crimson_ravine`, `emerald_grotto`) cũng được seed để Admin có thể gán region vào zone MMO. Các map realtime dùng kích thước artifact 64×48 và việc thêm vào catalog **không** tự chuyển region hay bật admission. Hai tầng trên vẫn khóa; khi gán region vào tầng khóa, region phải có `is_active=false`. `Region.csv` export có thêm `world_map_id`; import cho phép cấu hình region inactive trên tầng khóa nhưng từ chối region active. Admin Create/Update cũng áp dụng điều kiện này. Region tự gán map riêng (`world_map_id` trùng `id`, dùng cho `KHU_VUC`, `SAFE_ZONE` và `TRIBULATION`) được đăng ký và bật trong `world_maps` từ artifact `data/mmo/maps/<id>.json` ngay khi import, nên `aperture_tribulation` không bị chặn bởi kiểm tra tầng khóa. Catalog này hiện phục vụ cấu hình và quản trị; client realtime lấy map qua runtime/map artifacts, chưa có luồng người chơi để chuyển tầng.

Các bí cảnh bản đồ dùng chung có `regions.zone_type='SECRET_REALM'` và một `instance_templates.type='SPATIAL_REALM'` trỏ qua `spatial_region_id`. Chỉ số liệu về tọa độ region được import từ `Region.csv`; sau import, `seedHacBaiSpatialRealms` tạo template dùng chung, `ENTRY_PORTAL` trên tầng 1 và `EXIT_PORTAL` cho mười bí cảnh Hắc/Bạch. Instance isolation dựa trên `players.map_instance_id`; backend kiểm tra đúng region và tọa độ cổng vào trước khi tham gia phiên chung, còn vị trí quay về được lưu theo từng thành viên.

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
  "wild_gus": ["GU_LITTLE_LIGHT", "GU_JADE_SKIN"],
  "monster_gus": [
    { "gu_id": "mortal_gu_2", "rank": 2, "weight": 3, "min_count": 1, "max_count": 1 }
  ]
}
```
`cell_type_weights` quyết định tỉ lệ các loại ô; `resources` / `wild_gus` là pool tài nguyên và Cổ cấp thấp. `monster_gus` là bộ cổ mà **quái đánh trong vùng mang theo** (và do đó có thể rơi) — xem "Cổ trùng trên quái". `monster_gus` rỗng nghĩa là không gán thêm.

`resources` / `wild_gus` **để trống thì pool được suy ra từ registry theo dải rank của vùng** (`min_rank`–`max_rank`), không còn danh sách hard-code:
- `resources`: item `item_type` là `Material` hoặc `Food` trong dải rank vùng.
- `wild_gus`: biến thể Cổ rank ≤ 5 trong dải rank vùng (`HasImmortalRank` loại bỏ Cổ Tiên — Cổ Tiên chỉ đến từ gacha/luyện cổ/đấu giá).

Weight trong cả hai pool dùng `effectiveDropLevel(rank, rarity)` → **cả rank lẫn rarity** đều ảnh hưởng độ hiếm. Khai báo `resources` / `wild_gus` thì danh sách đó được dùng nguyên (override). Pool suy ra được cache theo dải rank và tự xoá khi import lại dữ liệu Vùng/Cổ/Item qua Admin.

**Thứ tự ưu tiên zone_type** — khi một điểm nằm trong nhiều zone chồng lấn, `FindZoneAt` chọn zone có priority cao nhất (bằng → diện tích nhỏ nhất = innermost):

```
HUB > SAFE_ZONE > BOSS_ZONE > SECRET_REALM > DUNGEON > PVP_ZONE > KHU_VUC > NGU_VUC
```

Zone type không nhận diện fallback về priority của `KHU_VUC`. `FindLargeZoneAt` luôn trả `NGU_VUC` chứa điểm.

**Bảng `map_cells`** — các ô persistent (boss, tài nguyên cấp cao, portal, điểm an toàn). Khóa vị trí là `UNIQUE(region_id, x, y)`, vì cùng tọa độ có thể có nội dung khác nhau ở từng tầng hoặc bí cảnh. Cột `last_cleared_at TIMESTAMPTZ`, `cleared_by UUID` theo dõi respawn; state clear được dùng chung trong cùng region.

**Bảng `monster_spawns`** — giờ là **pool theo zone** (không còn ý nghĩa toạ độ cho loại Roaming/Random). Khóa chính là cặp `(region_id, monster_id)` (không còn cột `id`); import CSV upsert theo khóa này nên sửa file rồi import lại sẽ cập nhật dòng cũ thay vì tạo trùng. `region_id` trỏ tới Đại Vực hoặc địa danh. Pool được cache trong bộ nhớ lúc start. `x`/`y` chỉ bắt buộc cho `spawn_mode=Fixed`; các mode theo zone như `Roaming`, `Random` và `Patrol` có thể để trống. Quái procedural sinh thành bãi ổn định theo vùng: tâm bãi và loài quái được chọn bằng seed, mật độ cao nhất gần tâm rồi giảm dần ra rìa; ngoài bãi vẫn có một lượng nhỏ quái đi lẻ. Quái **Elite** đi chung pool ambient với quái Normal (không còn cơ chế "Elite ở tâm bãi") nhưng bị nhân weight bằng `eliteSpawnWeightFactor` (0.2) nên hiếm hơn; Elite phát hiện trong 18 ô, đuổi tối đa 30 ô quanh điểm sinh và không ra ngoài vùng. **Vùng không có dòng `monster_spawns` nào thì không sinh quái** — không fallback sang template quái của vùng khác. Bãi không sinh trong `HUB`/`SAFE_ZONE`.

**Phân lớp sinh nội dung**:
- Quái nhỏ / tài nguyên thấp / Cổ cấp thấp → sinh **procedural** bằng hash seed theo `(zone, x, y, window 180s)`, không đụng DB. Tâm bãi quái dùng lưới seed riêng theo zone; cả vị trí, loài và cấp của bãi ổn định giữa người chơi và lần khởi động.
- Ô quái procedural trong AOI được materialize thành `WorldMonster` Redis với identity ổn định `procedural:<region>:<x>:<y>`; movement và combat sau đó được phát qua `world:combat`. Marker cell chỉ là fallback cho tới khi runtime replacement tại đúng tọa độ xuất hiện.
- Boss / tài nguyên cấp cao / làng → ô **persistent** trong `map_cells`, nạp vào index bộ nhớ lúc start, có respawn dùng chung. Boss world map chạy thành actor Redis dùng chung, đuổi người chơi trong toàn bộ bounds của region đã gán và không vào `HUB`/`SAFE_ZONE`. Khi chuyển spawn trong `MapCell.csv`, giữ ô cũ dưới dạng `is_active=false` để lần import cập nhật/tắt record cũ; restart backend sau import để nạp ZoneRegistry.

Import/Export qua Admin tab "Dữ liệu": bảng `region` (Region.csv), `map_cell` (MapCell.csv), `monster_spawn` (MonsterSpawn.csv). Admin quản lý region cũng có thể xem tầng hiện gán. Sau khi import **restart backend** để nạp ZoneRegistry. Bí cảnh dạng bản đồ dùng một phiên toàn server cho mỗi template: `shared_spatial_instances` giữ lifecycle/expiry, `shared_spatial_instance_members` giữ người tham gia và điểm quay về từng người. `POST /api/instance/spatial/exit` chỉ hoạt động tại portal persistent (`EXIT_PORTAL`, `PORTAL_EXIT` hoặc metadata action `EXIT_SPATIAL_REALM`); item effect `exit_spatial_realm` dùng cùng dữ liệu điểm quay về.

**Map MMO realtime dùng chung cho các region cũ (dev):** chạy `python3 scripts/convert_legacy_regions_to_mmo.py` sau khi sửa CSV để sinh cùng artifact `world_map.json` cho backend và Phaser. Map giữ tọa độ vùng cũ theo tỉ lệ 25 legacy units = 1 MMO world unit; join đổi tọa độ nhân vật cũ, checkpoint MMO giữ trực tiếp đơn vị mới. Converter xuất outline/tên 134 region và marker MapCell. MonsterSpawn hiện thiếu tọa độ nên converter đặt điểm spawn ổn định theo region; giới hạn zone là 500 quái, phần vượt giới hạn được ghi tại `docs/mmo/legacy-map-conversion-report.json`. CSV không có dữ liệu collision/terrain nên map chỉ chặn mép ngoài; luật PvP realtime hiện mở PvP toàn bộ ngoại trừ các HUB/SAFE_ZONE và Tân Thủ Cốc.

Backend tự tạo dữ liệu mẫu idempotent cho `spatial_realm_qingmu_valley` (Thanh Mộc Cốc) khi khởi động: vùng SECRET_REALM 20×20 tại tọa độ 1000–1019, điểm vào (1000,1000), cổng ra (1001,1000), dã trư, Nhân sâm và Ngọc Thạch. Bí cảnh dùng chung một phiên toàn server, mở khi người đầu tiên vào và hết hạn sau 60 phút. Các bản ghi chỉ được thêm nếu chưa tồn tại để giữ thay đổi của admin; Admin có thể quản lý vùng, map cells, monster spawn/dictionary và template như các dữ liệu khác. MapCell.csv dùng khóa `(region_id,x,y)` để các tầng/vùng có thể trùng tọa độ.

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
| `vault_gu` | JSONB | *(Legacy, đã bỏ)* Đã chuyển sang bảng quan hệ `sect_vault_gu`; cột không còn trong `CREATE TABLE`. |
| `vault_materials` | JSONB | *(Legacy, đã bỏ)* Đã chuyển sang bảng quan hệ `sect_vault_materials`. |
| `vault_recipes` | JSONB | *(Legacy, đã bỏ)* Đã chuyển sang bảng quan hệ `sect_vault_recipes`. |
| ~~`last_vault_decay_at`~~ | — | **Đã xoá** (`InitSchema` chạy `DROP COLUMN IF EXISTS`) — thuộc cơ chế đói kho đã bỏ. |

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
| `instance_id` | VARCHAR(80) | Instance ID gốc (đối với Tiên Cổ rank ≥ 6) |

> Giá điểm cống hiến khi đổi Cổ **không lưu ở bảng** mà suy ra từ `gu_dictionary.base_price` qua `pricing.VaultCost` (= 1.0× giá gốc, tối thiểu 1 điểm; đối với Tiên Cổ rank ≥ 6 áp dụng hệ số quy đổi $\times 1.000$ điểm) — xem `sectVaultGuCost`. Độ no cũng không lưu: Cổ trong kho không đói, rút ra luôn bắt đầu `Độ No = 1` (hoặc max nếu `hunger_rate = 0`).

### Quy chế Nhập kho (Stock), Rút kho (Withdraw) & Quyên góp (Donate) Bảo khố Tông môn

1. **Nhập kho từ Hệ thống (`StockSectVault` / `StockSectVaultBulk`)**:
   - **Chi phí Quỹ Tông Môn**: $1.5\times$ giá base (`pricing.ItemBasePrice` hoặc `pricing.RecipeCostByGuRank`). Đối với vật phẩm Tiên cấp (Rank $\ge 6$), nhân thêm hệ số quy đổi $\times 1.000$.
   - **Mở khóa theo Cấp Tông môn**: Cứ 10 cấp tông môn mở khóa 1 bậc Rank (`maxLevel := sectLevel/10 + 1`, tối đa Rank 9). Tông môn cấp 50 mở Rank 6, cấp 80 mở Rank 9.
   - **Giới hạn số lượng Tiên tài**: Mỗi lần nhập tối đa 5 đơn vị Tiên phẩm cùng loại; tổng lượng tồn kho bảo khố không vượt quá 200 đơn vị cho mỗi loại Tiên phẩm.

2. **Rút/Mua từ Bảo khố (`BuyFromSectVault`)**:
   - **Chi phí Điểm cống hiến**: Đúng $1.0\times$ giá base (`pricing.VaultCost`). Đối với Tiên phẩm / Tiên Cổ / Tiên Bí phương (Rank $\ge 6$), chi phí cống hiến nhân hệ số $\times 1.000$.

3. **Quyên góp vào Bảo khố (`DonateMaterials` / `DonateGu` / `DonateRecipe` / `DonateBulk`)**:
   - **Điều kiện Cấp Tông môn**: Tông môn phải đạt `sectLevel >= (rank - 1) * 10` mới có thể thu nạp vật phẩm/Cổ trùng/Bí phương phẩm cấp đó.
   - **Điểm thưởng & Quỹ Tông Môn**: $1.0\times$ điểm quyên góp (`pricing.DonatePoints`). Đối với Rank $\ge 6$, giá base được nhân $\times 1.000$ trước khi tính điểm cống hiến và cộng vào Quỹ Tông Môn.

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

## Lưu ý khi thay đổi schema

- Schema được tách theo domain ở `backend/internal/db/schema_*.go`, mỗi bảng là `CREATE TABLE IF NOT EXISTS` với toàn bộ cột inline. `InitSchema()` trong `db.go` gọi các schema function theo thứ tự FK. Không có migration riêng; cột mới khai báo trực tiếp trong `CREATE TABLE` (fresh init).
- Thêm/sửa bảng: sửa file `schema_*.go` tương ứng → xóa DB (hoặc `DROP TABLE` thủ công) → restart server.
- FK cho các cột `players.race` / `players.constitution` / `players.current_region`: các dòng sentinel (`''`), vùng `Southern_Border` và region spawn mặc định `tan_thu_coc_nam_cuong` được pre-seed trong `InitSchema` nên đăng ký vẫn hoạt động trước khi import CSV. Region `hub_nam_cuong` cũ đã bị loại bỏ hoàn toàn (seed, dữ liệu, map realtime); các tham chiếu legacy trong dữ liệu cũ được chuyển về `tan_thu_coc_nam_cuong`. Các HUB khởi đầu theo Đại Vực (`hub_*`) nằm trong `Region.csv` và được import qua Admin tab "Dữ liệu"; nếu chưa import, `complete-onboarding` fallback về `Southern_Border`. Các bảng dictionary (`race_dictionary`, `constitution_dictionary`) tự seed default dataset khi DB rỗng; CSV import (Admin tab "Dữ liệu") vẫn là nguồn chính thức.

### Lưu trữ sủng vật

Sủng vật được lưu dưới dạng `non_player_actors.actor_kind = 'COMPANION_BEAST'`, tham chiếu `MONSTER_TEMPLATE`. `formation_slot` là cột first-class; `upkeep_rate` là tốc độ đói và `hunger_updated_at` là đồng hồ riêng để các cập nhật actor không làm sai hunger decay. `RECOVERING` tương ứng trạng thái ngủ trong DTO cũ, còn `STORED` là active/có thể vào đội hình. Pet chỉ load vào combat khi `ACTIVE + STORED + slot>0 + hunger>0 + đủ soul`; NPC dị nhân chỉ load khi `ACTIVE + STORED + role=NONE + slot>0`. Slot formation dùng chung pet+NPC (unique `(owner,slot)`).

### Cột bắt sủng vật trong `monster_templates`

| Cột | Kiểu | Ghi chú |
| --- | --- | --- |
| `is_capturable` | BOOLEAN | DEFAULT FALSE. Có thể bắt làm sủng vật không. `Monster.csv` bật cho mọi quái dã ngoại (`Normal`/`Elite`) là sinh vật sống; khôi lỗi/thạch nhân/tàn niệm/linh thể/cây cỏ, `Boss`, `Tribulation_*` và `HumanCultivator` để FALSE |
| `capture_chance` | DOUBLE PRECISION | Tỷ lệ bắt cơ bản (0.0-1.0). rank 1 = 0.25, rank 5 = 0.10 |
| `pet_hunger_per_hour` | DOUBLE PRECISION | Tốc độ đói mỗi giờ. Mặc định 2 |

### Gộp sủng vật (bầy) và phí Hồn Phách khi ra trận

- **Gộp bầy** qua `/api/pets/merge` (`MergePets`): chỉ gộp pet cùng `monster_template_id`, nguồn phải ở kho (`formation_slot = 0`).
- **Giới hạn bầy** = `max(1, max_soul / 10)` (`mergeSoulDivisor = 10` trong `pets.go`). Nếu `count` sau khi gộp vượt giới hạn → **chặn toàn bộ gộp** (không xóa nguồn) và trả `merge_limit`.
- **Công thức chỉ số** (sub-linear, không cộng dồn compound): lấy base `stat / count^0.65` rồi scale `base × newCount^0.65` cho HP/Atk/Def; `speed` giữ nguyên; gộp hồi đầy HP. Hệ số `packScaleExponent = 0.65`.
- **Phí Hồn Phách khi ra trận:** mỗi pet (bầy) tiêu tốn `soul_hp` = `count` (số cá thể) khi load vào trận thật (`combat_state.go`). Nếu `soul_hp < count` → **bỏ qua pet đó**. Trừ dần theo `formation_slot ASC`.
  - Áp dụng cho cả pet của **đối phương phòng thủ** khi bị tấn công PvP/mine/map (trừ `soul_hp` của họ, thiếu thì bỏ pet của họ).
  - **Võ Đài (arena, virtual) không tốn soul.**
  - Phí được xử lý khi build realtime combatants trong `backend/internal/realtime/gateway/combat_state.go`; không dùng snapshot `active_combat` cũ.

### Đội hình tổ đội (trưởng nhóm xếp)

- Trưởng nhóm xếp thứ tự xuất trận qua `/api/party/formation`: `player` (thành viên), `pet`/`npc` (thú/dị nhân của chính trưởng nhóm), tối đa 8 vị trí sau trưởng nhóm (9 ô grid). Trưởng nhóm luôn `Allies[0]`.
- Mỗi entry có `pos` 1-8 là ô grid combat (`allySlots`: cột trái = tiền tuyến hứng đòn trước); `pos` trùng nhau bị từ chối, thiếu = tự xếp ô trống thấp nhất. UI Tổ đội là lưới 3x3 Tiền/Trung/Hậu nhìn đúng như combat.
- Đội hình rỗng = chế độ tự động cũ (cả đội cùng region + thú/NPC theo `formation_slot`). Đã xếp = chỉ ai trong danh sách mới vào, theo đúng thứ tự (vị trí grid gán theo index như cũ).
- Điều kiện từng vị trí vẫn giữ: member cùng region + còn HP (virtual hồi đầy); pet `STORED + hunger>0 + đủ soul` (lineup thay luôn yêu cầu slot); NPC `STORED + role NONE`. Kẻ không đủ điều kiện tự bỏ qua + log.

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

**Hàng mặc định + xoay vòng ngẫu nhiên theo giờ:** món `is_default = true` luôn hiển thị. Món `is_default = false` được shuffle deterministic theo khung giờ (seed = Unix giây của bucket, `shopRotationPeriod = 1h`, làm tròn UTC) theo **tỷ lệ cân bằng ~40 Cổ / ~60 item** (`maxDailyRandomGus`/`maxDailyRandomItems`; cache Redis key theo bucket giờ).

**Thứ tự chọn:** chọn subset từ pool đầy đủ TRƯỚC, rồi mới loại món sold-out khi hiển thị — nếu lọc trước khi shuffle thì pool co lại sau mỗi giao dịch và subset sẽ đổi giữa khung giờ (rò rỉ item mới), phá vỡ tính "cố định trong khung".

**Pool ngẫu nhiên tự sinh (virtual):** ngoài các món `is_default = false` trong `caravan_shop_items`, `autoShopPool()` (`handlers/shop_catalog.go`) tự bổ sung vào pool ngẫu nhiên mọi **Cổ trùng rank < 6 và rarity < 6** (mỗi variant pinned đúng rank) cùng **thức ăn của chúng** (chỉ các id khớp dòng thật trong `item_dictionary`; các pseudo-food tài nguyên không phải item nên tự bị loại), trừ các id đã có trong `caravan_shop_items`. Các món này **không được persist** vào bảng shop nên không làm bẩn danh sách admin hay CSV export.

**Giá + số lượng (stock) món random:** giá mua = `base_price × 1.5` (`randomPoolMarkup`, làm tròn đến `0.001`, tối thiểu `0.001`). Mỗi món có stock hằng ngày theo rank qua `randomPoolStock(rank)`: **R1 = 100, R2 = 50, R3 = 20, R4 = 5, R5 = 2, R6+ = 0**. Số còn lại trong ngày lưu ở bảng `caravan_shop_daily_stock(day, id, rank, remaining)` — key theo ngày nên tự reset mỗi ngày (stock không reset theo giờ). `/api/shop/buy` trừ stock nguyên tử (`UPDATE ... WHERE remaining >= qty`), hết thì trả lỗi "Hàng này hôm nay đã hết". Món default/admin trong `caravan_shop_items` **không** áp dụng cơ chế này (stock CSV của chúng vẫn chỉ hiển thị). Frontend `CaravanShop` hiển thị đếm ngược tới khung xoay kế tiếp và tự nạp lại khi sang khung mới.

**Shop Võ Đài (arena) — tài nguyên Tiên cấp xoay vòng:** ngoài các dòng tĩnh trong `arena_shop_items`, `GET /api/arena/shop` tự thêm tối đa **30** item `Food`/`Material` rank ≥ 6 từ `item_dictionary` (`arenaRandomResourcePool`, trừ id đã có trong `arena_shop_items`), chọn subset deterministic theo cùng khung giờ bằng `selectArenaRandomResources` → `dailyShopRandomSubset`/`shopRotationBucket`. Giá **đồng giá 99 Xu Đấu Trường**, stock vô hạn, `qty = 1`. Món random **không persist** vào `arena_shop_items` (không hiện ở Admin/CSV export); `POST /api/arena/shop/buy` tra `arenaRandomResourceByID` (toàn pool, không chỉ subset đang hiển thị) nên mua được ngay khi vừa thấy. Cache `arena_shop_items` đổi TTL từ 24h xuống **10 phút** để subset đổi theo khung; admin sửa dữ liệu vẫn `cache.Delete("arena_shop_items")` như cũ.

### Thương nhân (2 loại)

Có **2 loại** thương nhân; cả hai dùng profile data-driven + quota toàn server.

**Loại 1 — cố định tại HUB/SAFE_ZONE (cơ chế cũ).** Chính là Caravan/Thương Hội: bán random theo khung giờ, mua tất cả giá 10% phẳng. Profile `village_general` (`buy_ratio = buy_floor_ratio = 0.10`, `quota_per_item = 0` = vô hạn). `/api/shop/sell` + `/sell-bulk` **delegate** sang profile này theo region (giữ nguyên shape response); instance `village_<region>` tạo lazy (`ensureRegionMerchants`).

**Loại 2 — lang thang ngoài map.** Profile `roaming_trader` (`kind = ROAMING`) làm giá trị nền; **mỗi instance override riêng** (`merchants.buy_tags` / `buy_ratio` / `buy_floor_ratio` / `quota_per_item`) nên **mỗi con có bể mua và bể bán riêng**:
- Bể mua: allow-list ngẫu nhiên ~8–15 id (`buy_tags.item_ids`, gồm Cổ/Item/cổ phương), giá riêng (`buy_ratio` random 0.70–1.00 → `buy_floor_ratio` 0.10–0.30), quota riêng 5–20/giờ (toàn server).
- Bể bán: `merchant_stock` random sinh lúc spawn (Cổ rank<6 & rarity<6 + thức ăn + ~3 cổ phương, giá `base × 1.4`).

Worker `StartRoamingMerchantWorker` (2', package `handlers`) duy trì tối đa **20** con ở region `zone_type NOT IN ('HUB','SAFE_ZONE','SECRET_REALM')`: vòng đời 10–30', hết hạn thì xóa (cascade stock/demand) và spawn con mới. **Ưu tiên vùng đang có người chơi:** mỗi lượt spawn, worker tải danh sách vùng kèm số người chơi hoạt động trong **15' gần nhất** (`players.last_activity`, batch flush 2') và số trader vùng đó đang có; **75%** (`roamingPlayerZoneChance`) rút thăm chỉ trong nhóm vùng có người chơi, 25% còn lại rút trên toàn bản đồ ⇒ vùng vắng vẫn có trader. Trọng số rút thăm: `(1 + players) / (1 + traders)` ở nhóm có người (khuyến khích vùng đông, rải đều khi 1 vùng đã có nhiều trader), `1 / (1 + traders)` ở nhóm toàn bản đồ. Toạ độ vẫn random trong bbox vùng (không nhắm sát người chơi). Mỗi spawn **thông báo ở kênh khu vực (`region`) của vùng spawn** kèm tên vùng + toạ độ (chỉ người trong vùng đó thấy).

**Giao dịch & hiển thị:** merchant mua/bán được cả **Cổ, Item, cổ phương** (bán cổ phương = `RecipeSvc.Delete`, mua = `RecipeSvc.Upsert` unlocked). `GET /api/merchant` chỉ trả roaming **trong bán kính ≤ 2 ô** (đứng cạnh) + cờ `at_village` (region HUB/SAFE_ZONE) để UI tự bật/tắt: **Caravan/Thương Đội chỉ mở tại làng**. Luồng marker toàn vùng đã gỡ cùng legacy map endpoint; Phaser dùng dữ liệu thương nhân trong phạm vi tương tác.

| Bảng | Vai trò |
| --- | --- |
| `merchant_profiles` | `buy_ratio`, `buy_floor_ratio`, `buy_tags` (JSONB: `types`/`min_rank`/`max_rank`/`item_ids`), `quota_per_item`, `quota_period` (HOUR/DAY), `sell_markup`, `kind` (FIXED/ROAMING) |
| `merchants` | Instance; cố định id `village_<region>`, roaming id `roam_<uuid>` + `expires_at`; override `buy_tags`/`buy_ratio`/`buy_floor_ratio`/`quota_per_item` (NULL = dùng profile); FK `region_id` |
| `merchant_stock` | Hàng merchant bán ra (roaming sinh random lúc spawn) |
| `merchant_demand` | `(merchant_id,item_id,period_start) → bought` — PK theo kỳ nên quota tự reset |

- **Giá động:** `ratio(bought) = buy_ratio − (buy_ratio − buy_floor_ratio) × min(bought/quota, 1)`; giá đơn vị = `base_price × ratio`, làm tròn `0.001`.
- **Quota toàn server:** ai bán trước được trước; hết quota loại đó tới kỳ sau.
- Endpoint: `GET /api/merchant`, `POST /api/merchant/{quote,sell,buy}`.
- Seed trong `SeedMerchantDefaults` (code). Upgrade path: CSV + Admin dataset khi cần tinh chỉnh runtime.

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
  - Chi phí: 3.000 Nguyên Thạch + 20x Linh Không Gian Tinh (`space_crystal_rank2`).
  - Phạm vi cự ly: Tối đa 1.000 ô trong cùng Đại Vực.
  - Độ bền tối đa: 50 điểm (mỗi lượt nhảy giảm 1 điểm).
- **`TIER_2` (Đại Vực Truyền Tống Trận):**
  - Tu vi tối thiểu: Chuyển 4 (Rank 4).
  - Chi phí: 15.000 Nguyên Thạch + 30x Địa Không Gian Tinh (`space_crystal_rank4`).
  - Phạm vi cự ly: Tối đa 10.000 ô trong cùng Đại Vực.
  - Độ bền tối đa: 100 điểm.
- **`TIER_3` (Ngũ Vực Tiên Cấp Truyền Tống Trận):**
  - Tu vi tối thiểu: Chuyển 6 (Rank 6 - Tiên Nhân).
  - Chi phí: 100.000 Nguyên Thạch + 50 Tiên Nguyên Thạch + 80x Tiên Không Gian Tinh (`space_crystal_rank6`).
  - Phạm vi: Tự do dịch chuyển xuyên tất cả các Đại Vực trong Ngũ Vực (`is_cross_region = true`).
  - Độ bền tối đa: 200 điểm.
  - Phản phệ Giới Vực Bích: Khi nhảy xuyên Đại Vực, người chơi vẫn phải chịu debuff áp chế thuộc tính theo Rank từ `CheckAndApplyRegionalWallDebuff`.

#### Quy tắc kinh tế & vận hành:
- **Chế độ kết nối:** Hai chiều (`TWO_WAY`), Cổng gửi đi (`ONE_WAY_SEND`), Cổng nhận (`ONE_WAY_RECEIVE`).
- **Quyền truy cập:** `PUBLIC` (Toàn server), `SECT` (Thành viên Tông Môn), `PERSONAL` (Chỉ chủ sở hữu).
- **Phí dịch chuyển & Két sắt:** Chủ sở hữu cấu hình bật/tắt thu phí và mức giá vé. Khi có người sử dụng, 95% phí được tích lũy vào Két Sắt Trận Pháp (`vault_balance`) để chủ sở hữu đến rút, 5% hao tán theo quy luật trời đất.
- **Bảo trì & Thu hồi:** Tiêu hao Nguyên Thạch để sửa chữa hồi phục độ bền. Khi thu hồi (Dismantle), hoàn trả 50% chi phí xây dựng ban đầu (nhân theo tỉ lệ % độ bền còn lại) và tự động chuyển toàn bộ số dư Két Sắt về túi nhân vật.
- **Realtime MMO:** Nhân vật đang được zone realtime điều khiển vẫn truyền tống được. `/api/teleport/travel` tạm dừng checkpoint, flush vị trí/HP live vào DB, chạy toàn bộ kiểm tra phí/độ bền/rank, ghi đích (`current_region`, `x/y`, `world_zone_id`, `world_x/y`), rồi nhả lease. Client nhận `realtime_transfer: true` và reconnect vào zone đích với session mới.

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
trigger-depth, repeat, and summon budgets. Combat operations are consumed only by their current realtime handlers; the removed turn-based `simulator.SimAction` is not a runtime consumer.

## Admin audit log fields

`audit_logs` stores immutable admin mutation context in dedicated columns:
`actor_username`, `effective_roles`, `effective_permission`, `entity_type`, `entity_id`,
`change_data`, `reason`, and `request_id`, in addition to actor `player_id`, IP, outcome,
sanitized error, and `created_at`. Existing databases receive these columns through

Sensitive request fields and multipart bodies are excluded from `change_data`.

## Chỉ số thám hiểm (INVESTIGATION / DISGUISE) trên bản đồ

- **INVESTIGATION** cộng trực tiếp vào bán kính tầm nhìn bản đồ (`/api/explore/grid` →
  `vision_radius` và `stats.investigation`): `radius = clamp(int(stats.investigation), 1, 25)`.
  Trần cứng tối đa là **25 ô** (`MaxMapVisionRadius` trong `exploration_grid.go`), áp dụng cho
  cả grid viewport và client world map để Cổ stack INVESTIGATION không mở toàn bản đồ. UI lưới
  cũ vẫn giới hạn hiển thị 5x5 riêng.
- **DISGUISE** giảm cơ hội bị quái tấn công/phục kích khi di chuyển ngoài bản đồ:
  - Phục kích lân cận (`exploration_move.go`): `ambushChance = clamp(50 + rank*2 - disguise, 5, 95)`.
  - Khám phá ngẫu nhiên (`combat_explore.go`): cửa sổ quái `roll < 70` thu hẹp còn
    `roll < max(70 - disguise, 40)` (cửa sổ tài nguyên `< 40` giữ nguyên).
  - `disguise = PlayerDisguiseLevel(player)` = rank + Σ `STAT_MOD DISGUISE ADD_FLAT` của Cổ
    đang trang bị + thưởng cảnh giới lưu phái Trí/Ám (như profile scouting).
- Cổ mang tag/role trinh sát (INVESTIGATION, MAP_REVEAL, scout...) giờ có `STAT_MOD INVESTIGATION`
  trong `effects`; Cổ che giấu (concealment/stealth...) có `STAT_MOD DISGUISE` + capability
  `DISGUISE` trong `effects`. Giá trị = rank của Cổ.

## Phi hành (flying) — bỏ qua quái khi di chuyển

Người chơi ở trạng thái phi hành sẽ **lướt qua quái thú dưới đất**: không kích combat khi dẫm ô
`Monster`/`Boss`, không bị phục kích lân cận, không bị quái world (AI sống) nhắm/đuổi theo. PVP
và du ngoạn (`/explore`) không bị ảnh hưởng. Ô quái không bị xóa — người chơi khác vẫn gặp.

Hai cơ chế, cùng đọc qua `PlayerIgnoresMonsters` (`handlers/player_stats.go`):

- **Chủ động (đa số)** — kỹ năng Active `ON_ACTION` với `STATUS` `status_id` = `FLYING` (hoặc
  `PHI_HANH`). Kích hoạt → vào trạng thái `phi_hanh` trong N phút; **dùng lại để hạ cánh sớm**.
  Block v2 yêu cầu `turns` >= 1, nên đặt `turns` = rank của Cổ (1–9); handler vẫn fallback về
  rank nếu `turns` không có. Cooldown 2 giây. Ví dụ block rank 4:
  ```json
  {"trigger":"ON_ACTION","target":{"selector":"SELF"},"operations":[
    {"op":"STATUS","status_id":"FLYING","turns":4,"value":0,"chance":1}]}
  ```
- **Bị động (cực hiếm)** — capability `IGNORE_MONSTER_ENCOUNTER` trên Cổ trang bị: phi hành vĩnh
  viễn khi đang đeo.

### Cổ phi hành trong dữ liệu

- **Dòng Phi Hành Cổ (Phong đạo, 9 chuyển):**
  `GU_FLY_SKY_1` (Sơ Giai Phi Hành Cổ) → `GU_FLY_SKY_2` (Lăng Không Cổ) → `GU_FLY_SKY_3`
  (Ngự Phong Cổ) → `GU_FLY_SKY_4` (Đạp Vân Cổ) → `GU_FLY_SKY_5` (Cửu Tiêu Cổ) →
  `GU_FLY_SKY_I` rank 6-9 (Phi Hành / Thiên Ngoại / Hư Không / Càn Khôn Tiên Cổ).
  Mỗi Cổ có `STATUS FLYING` với `turns` = rank (phút), kèm `SPEED`/`MOVEMENT` thụ động theo rank.
- **Cổ bay sẵn có được gắn thêm Fly:** `GU_BONE_WINGS` (R4), `GU_FLYING_SMOKE` (R5),
  `GU_FLYING_WING_5` (R5), `GU_LEVEL_CLOUD` (R6), `GU_DREAM_WINGS` (R6).
- `FLYING` và alias `PHI_HANH` được đăng ký trong `statusfx` với behaviour `world_map` (không có
  tác dụng trong combat) để Block v2 compile/validate hợp lệ.

World-monster worker bỏ qua người đang bay qua registry in-memory `flyingPlayers`
(`handlers/world_index.go`), refresh ở mỗi lần di chuyển và mỗi lần toggle. Khi quái áp sát,
worker gây sát thương theo nhịp realtime; không tạo encounter trong `active_combat`.

## Ordinary world-monster lifecycle

`monster_templates`, `monster_spawns`, and `map_cells` remain PostgreSQL static data. Ordinary non-boss monster HP, target, state, statuses, position, timestamps, and revision are ephemeral Redis data under `world:monster:*`; they are not written to PostgreSQL per hit or movement. A static spawn's stable identity is `spawn:<region_id>:<monster_id>` (the table's composite key); procedural cells, map-cell bosses and guards keep their own prefixed ids. Runtime aggro scales from 6 cells at Rank 1 to 14 cells at Rank 9; the leash scales from 10 to 20 cells. Other defaults are a 1.5-cell attack range, speed 0.15 cells/second, and a 2-second attack interval. Monsters cannot spawn in, acquire targets in, attack into, or cross a `HUB`/`SAFE_ZONE`. Runtime TTL is the spawn respawn duration plus a five-minute replay-safety window.

Ordinary world monsters pursue nearby players and attack through the realtime world-combat worker when their authoritative attack interval is ready; contact alone does not create a separate encounter or turn-based auto-combat session. Player attacks, Gu casts and supported Sát Chiêu resolve in the realtime world zone. When a monster defeats a player, its display name is stored in the ground drop's `killer_name`.

Monster runtime AOI uses **32×32-cell buckets**, separate from the legacy
300×300 player-position broadcast index. The activation/fan-out radius is 25
cells, matching the current hard cap for player investigation range, so one
player normally activates at most a 3×3 neighborhood of small monster buckets.
This separation avoids loading a 300×300 monster area for a 10–25-cell view
without changing party, PvP, or world-position broadcast behavior.
