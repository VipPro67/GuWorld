# Kế hoạch tối ưu effect Cổ trùng và viết lại sát chiêu

> Trạng thái: đề xuất kỹ thuật, chưa phải đặc tả gameplay đã chốt.
> Phạm vi khảo sát: mã nguồn và dữ liệu hiện tại tại ngày 2026-09-15.

## 1. Tóm tắt quyết định

Không nên tiếp tục thêm field vào `ActiveEffect`, thêm nhánh `if` trong handler và thêm field vào `SimAction` cho mỗi effect mới. Với quy mô dự kiến “một Cổ có thể có một effect riêng”, cách đó làm chi phí của mỗi effect lan qua loader, handler, simulator, frontend và test.

Hướng đề xuất là xây một **effect runtime data-driven, có kiểu và có phiên bản**:

1. Dữ liệu Cổ và công thức sát chiêu chỉ mô tả một danh sách `EffectSpec` nhỏ, tuần tự và có điều kiện.
2. Một compiler validate dữ liệu rồi biên dịch thành `CompiledAction`; combat engine chỉ thực thi một tập opcode ổn định.
3. Effect thật sự đặc biệt dùng `mechanic_id` trỏ tới implementation Go đăng ký tường minh, thay vì thêm một field boolean mới hoặc nhúng script tùy ý.
4. Sát chiêu trở thành **định nghĩa game toàn cục có công thức chính xác**, còn bản ghi người chơi chỉ giữ trạng thái sở hữu/thông thạo/loadout. Không còn suy ra sát chiêu từ mọi tổ hợp Cổ.
5. Định nghĩa đã biên dịch là immutable, cache theo `(definition_id, version)`; trận đấu chụp snapshot/version để replay vẫn xác định sau khi admin đổi dữ liệu.

Đây là một **module sâu**: interface công khai nhỏ (`Compile`, `Execute`, `Describe`, `ValidateCatalog`) nhưng che giấu validation, targeting, stacking, timing, scaling, RNG và log phía trong. Seam nên nằm giữa “catalog gameplay” và “combat simulator”, không nằm ở database hay handler.

## 2. Hiện trạng đã kiểm tra

### 2.1. Nguồn dữ liệu và vòng đời

- `Gu.csv` có 4.334 dòng dữ liệu. Mỗi dòng luôn mang tám namespace JSON (`active`, `passive`, `growth`, `other`, `explore`, `support`, `refinement`, `house`), nhưng Go chỉ deserialize bốn namespace `active`, `passive`, `growth`, `other` trong `GuEffects`.[^gu-model]
- Admin import ghi JSON vào `gu_dictionary.effects`; bảng dùng khóa `(id, rank)` và JSONB không có version/schema discriminator.[^gu-schema]
- `models.LoadGuRegistry` đọc toàn bảng vào map global `GuRegistry[id][rank]`, thay map nguyên khối và xóa một số Redis dictionary cache. Registry không có revision công khai, snapshot hay cơ chế đồng bộ đọc/ghi.[^gu-registry]
- `GuEffects.Scan` dùng `encoding/json` mặc định: key lạ bị bỏ qua im lặng; bước normalize chỉ quy shorthand status về một `ApplyStatus` duy nhất.[^gu-scan]

### 2.2. Ba hệ effect đang chồng lên nhau

1. **Effect nhân vật** (`gamefx.Effects`) cho race/constitution/status: map stat/path cộng dồn và các cờ khóa hành động.
2. **Effect Cổ** (`models.GuEffects`): struct rộng gồm active/passive/growth và `SpecialMechanic` dạng `action_code + params`.
3. **Effect trong simulator**: `SimAction` là struct phẳng có field riêng cho damage, heal, DoT và từng status tier; trạng thái runtime lại chia thành `Effect`, `DoTEffect` và `statusfx.StatusOp`.[^sim-action]

Các representation này không có một semantic contract chung. Vì vậy cùng một khái niệm có thể dùng key, đơn vị và quy tắc khác nhau ở CSV, model, compiler trong handler và engine.

### 2.3. Luồng combat hiện tại

Luồng chính hiện nay là:

```text
Gu.csv -> Admin import -> gu_dictionary.effects
       -> GuRegistry (global cache)
       -> handler chọn Gu / killer move
       -> CalculateGuEffectForCombatant + buildSimAction...
       -> simulator.SimAction (nhiều field chuyên biệt)
       -> simulator engine (nhiều nhánh effect theo phase)
       -> combat log + persistence kết quả
```

`CalculateGuEffectForCombatant` vừa kiểm tra cấp sử dụng, vừa scale damage/defense theo attainment, vùng, mastery, đạo ngân và effect nhân vật; riêng tra vùng còn query DB trong lúc build action.[^calculate-gu] `BuildCombatAction` lại diễn giải nhiều field active và tạo status/log bằng chuỗi `if` nối tiếp.[^build-action] Auto combat dùng một đường compile khác sang `SimAction`, nên khả năng drift giữa các mode là cao.

### 2.4. Sát chiêu hiện tại

- `killer_moves` là thực thể thuộc người chơi, lưu cả đầu vào (core/slots) lẫn kết quả tính sẵn (mana, success, damage, defense, cooldown, flags và `effects` JSONB). Slot tham chiếu trực tiếp instance Cổ của người chơi; xóa core Cổ sẽ cascade xóa sát chiêu.[^km-schema]
- Cơ chế compound làm phẳng effect Cổ thành `map[string]float64`, rồi dò tập key bằng các rule Go hard-code. Điều đó mất targeting, timing, nguồn effect, kiểu value và thứ tự thực thi.[^flat-effects][^compound]
- Repo đã có vài `synergyRecipes` chỉ định core/support cụ thể, nhưng chúng vẫn hard-code trong Go và chỉ thêm bonus/chuỗi mô tả; đây chưa phải catalog công thức authoritative.[^synergy]
- Sát chiêu được chuyển tiếp lần nữa sang `SimAction`; dữ liệu JSON effect đã materialize và logic engine có nguy cơ không cùng phiên bản.

### 2.5. Khoảng cách giữa dữ liệu và implementation

Một phép kiểm tra toàn bộ `backend/data/Gu.csv` cho thấy:

| Nhóm | Có trong CSV | Có field typed tương ứng | Rủi ro |
|---|---:|---:|---|
| `active` key khác nhau | 36 | khoảng 30 | key lạ bị bỏ qua |
| `passive` key khác nhau | 25 | 20 | key lạ bị bỏ qua |
| `other.action_code` khác nhau | 19 | xử lý phân tán | dễ tồn tại action không có executor |

Ví dụ key có dữ liệu nhưng không hiện diện trong `ActiveEffect` gồm `reveal_all_stealth`, `reveal_weakness`, `invulnerable_ticks`, `summon_monster_id`, `sleep`, `decoy_count`, `convert_damage_to_heal_percent`, `absorb_damage_flat`; phía passive có `stat_base_luck`, `dao_mark_multiplier`, `kill_path_damage_buff`, `luck_flat`, `storage_slots`. Đây là lỗi nguy hiểm kiểu **data hợp lệ về JSON nhưng vô hiệu về gameplay**.

Ngoài ra `ActiveEffect` chỉ có một `ApplyStatus`, nên một kỹ năng không thể biểu diễn tự nhiên nhiều status độc lập. Normalize shorthand cũng gán `StatusTickDamage` bằng tier cho một số status, trong khi simulator mô tả `Value` là tier rồi chuyển tier sang phần trăm; đơn vị đang bị nhập nhằng.[^gu-active][^status-tier]

## 3. Vấn đề ưu tiên

### P0 — Correctness và khả năng kiểm soát dữ liệu

1. **Unknown field bị nuốt im lặng.** Admin có thể import effect trông đúng nhưng runtime không thi hành.
2. **Nhiều compiler/đường combat.** Manual, auto, PvP defender và killer move có thể hiểu cùng effect khác nhau.
3. **Không có version snapshot.** Thay catalog có thể làm trận đang chạy/replay dùng semantics mới.
4. **Đơn vị không có kiểu.** `0.2`, `20`, tier `2`, ticks và phần trăm có thể cùng nằm trong `float64`.
5. **Materialized output không có provenance.** Không biết sát chiêu được compile từ revision công thức/effect nào.

Các lỗi cụ thể nên sửa/characterize trước rewrite:

- `GetFlatGuEffects` đọc `Active.CaptureBonus` bên trong nhánh chỉ đảm bảo `Passive != nil`; Cổ passive-only có thể nil panic.[^flat-effects]
- Flattener xuất `lifesteal`, trong khi auto killer move đọc biến thể `life_steal`; đơn vị 0..1 và phần trăm cũng bị trộn, nên effect có thể mất hoặc sai 100 lần.
- Cast manual hard-code duration bleed/poison/stun và có nhánh xuyên giáp không được kích hoạt; một số `SpecialEffect` synergy hiện chỉ là mô tả chứ không thi hành.
- Auto combat dùng snapshot damage/effect lúc tạo move, manual cast tính lại từ catalog hiện tại, còn world/map là interpreter thứ ba. Cùng một move vì thế có thể cho ba kết quả.
- Simulator chỉ trừ hunger cho `GU_SKILL`, không tương đương rõ ràng với `KILLER_MOVE`.
- Loader có xu hướng warn/skip dữ liệu lỗi rồi vẫn thay registry; một catalog nửa hợp lệ có thể được publish.
- Một số đường như kiểm tra thành công khi cast sát chiêu manual dùng nguồn ngẫu nhiên global thay vì RNG explicit của simulator, làm yếu tính deterministic.
- Reload admin thay global registry/cache không qua immutable snapshot/atomic pointer, cần race test trước khi bật hot reload thường xuyên.
- Store/load sát chiêu không nhất quán quanh `usage_count`, có thể làm mastery không tiến đúng; list slots theo từng move tạo N+1 query.
- Ownership bất đối xứng: core/support đều FK tới instance nhưng transfer/delete có thể khiến binding recipe thiếu hoặc tham chiếu Cổ không còn thuộc người tạo.
- Contract UI/backend dùng tên khác nhau (`success_rate`/`stability`, `channel_time`/`cast_time`, `is_reliable`/`is_stable`), có nguy cơ hiển thị 0/undefined hoặc crash formatter.

Các mục này cần regression test trước khi “sửa cho đúng”, vì một phần có thể đã trở thành hành vi người chơi đang dựa vào.

### P1 — Khả năng mở rộng

1. Mỗi mechanic mới phải sửa nhiều struct và switch/if.
2. Struct phẳng không biểu diễn tốt chuỗi effect, nhiều mục tiêu, trigger, điều kiện, delayed effect, resource khác mana, summon hay mutation battlefield.
3. `map[string]float64` của compound xóa semantic information trước khi ghép.
4. Formula hard-code khiến cân bằng gameplay cần deploy backend, trong khi công thức sát chiêu cần data-driven và audit được.

### P2 — Hiệu năng và vận hành

1. Build action có thể query vùng theo từng action; cần đưa toàn bộ combat context vào snapshot trước compile.
2. JSON được parse/convert nhiều lần và các map/string dispatch nằm trong hot loop.
3. Registry global mutable không an toàn cho hot reload và khó quan sát revision đang phục vụ.
4. Không có lint/report coverage “mọi key/action_code trong catalog đều có implementation”.

## 4. Kiến trúc đích

### 4.0. Các phương án đã cân nhắc

| Phương án | Ưu điểm | Nhược điểm | Kết luận |
|---|---|---|---|
| Typed opcode IR + native hook | Data-driven, strict, deterministic, dùng chung mọi mode; effect dị vẫn có escape hatch | Tốn compiler/migration ban đầu | **Chọn** |
| Một Go strategy cho mỗi Cổ/move | Type-safe, debug dễ, biểu đạt tùy ý | Mỗi content cần deploy, hàng nghìn implementation, khó cho Admin/designer | Chỉ dùng làm native hook nội bộ |
| DSL/VM tổng quát | Content tự do và hot reload mạnh | Interface ngôn ngữ rất lớn, sandbox/debug/version/budget khó | Chưa làm; nếu cần sau này phải compile xuống IR |

Ngưỡng vận hành gợi ý: nếu native hook vượt 15–20% definition, hoặc cùng một pattern custom lặp ít nhất ba lần, nâng pattern đó thành primitive opcode mới. Không tạo opcode chỉ vì một con số khác nhau.

### 4.1. Mô hình hai tầng: primitives + mechanics

Không chọn một trong hai cực “mỗi effect là field Go” hoặc “script engine tự do”. Dùng hai tầng:

**Primitive opcodes** (dùng lại cho phần lớn effect):

- `deal_damage`, `heal`, `grant_shield`, `modify_resource`;
- `apply_status`, `remove_status`, `dispel`;
- `modify_stat`, `modify_damage`, `redirect_damage`;
- `move`, `select_target`, `repeat`, `schedule`;
- `summon`, `despawn`, `emit_event`.

**Named mechanic** (cho effect thật sự dị):

```go
type Mechanic interface {
    ID() string
    Compile(json.RawMessage) (CompiledMechanic, error)
}
```

`CompiledMechanic` giữ payload typed, immutable và execution-cost bound; runtime không parse lại JSON. Registry chỉ nhận mechanic đã đăng ký lúc startup. Dữ liệu gọi `mechanic_id: "blood_sea_rebirth"`; compiler fail nếu thiếu implementation. Không cho admin upload code/script. Như vậy một Cổ có effect riêng chỉ cần **một adapter mechanic cục bộ + test**, không buộc nở `SimAction`.

### 4.2. Interface nhỏ tại seam combat

```go
type DefinitionCompiler interface {
    CompileGu(GuDefinition, CatalogRevision) (CompiledAction, error)
    CompileKillerMove(KillerMoveDefinition, CatalogRevision) (CompiledAction, error)
}

type EffectRuntime interface {
    Execute(CompiledAction, *EffectContext) EffectResult
}
```

`CompiledAction` là immutable và chỉ chứa opcode/value đã validate, selector đã compile, phase/order rõ ràng; không chứa DB handle. `EffectContext` chứa actor, targets, battlefield, deterministic RNG, turn/phase và catalog revision. `EffectResult` trả event/result để engine áp dụng; implementation không tự ghi DB.

Deletion test: nếu xóa module này, validation, target selection, stacking, execution và log sẽ lại tràn về handler/simulator — module đủ sâu và đáng tồn tại. Đây **không phải service/repository layer**; gameplay orchestration vẫn ở handler theo convention repo, còn semantics effect nằm đúng seam engine.

### 4.3. `EffectSpec` đề xuất

```json
{
  "schema_version": 2,
  "effects": [
    {
      "id": "main_hit",
      "trigger": "on_cast",
      "op": "deal_damage",
      "target": "selected_enemies",
      "value": {"base": 120, "scale": [{"stat": "attack", "ratio": 0.8}]},
      "damage_type": "physical",
      "path": ["blood"],
      "order": 100
    },
    {
      "id": "bleed_after_hit",
      "trigger": "after_damage",
      "condition": {"event": "main_hit", "result": "hit"},
      "op": "apply_status",
      "target": "event_targets",
      "status": "bleed",
      "stacks": 1,
      "duration": {"turns": 3},
      "order": 200
    }
  ]
}
```

Các nguyên tắc:

- `id` ổn định trong definition để condition tham chiếu, debug và telemetry.
- `trigger` và `order` bắt buộc; không dựa vào thứ tự map JSON.
- Giá trị có unit/type riêng: `Ratio`, `Percent`, `Flat`, `Turns`, `Count`, không dùng một `float64` chung ở runtime.
- Target selector thuộc catalog hữu hạn, được compile trước trận.
- RNG chỉ qua `EffectContext.RNG`; seed ghi vào combat snapshot.
- Mọi op trả event chuẩn hóa, log/UI description được render từ event/template chứ không lẫn vào tính toán.

### 4.4. Phase và event chuẩn

Chốt trước một state machine tối thiểu:

```text
validate cast -> pay costs -> before_cast -> select targets -> on_cast
-> before_hit -> hit/miss -> before_damage -> damage -> after_damage
-> apply status -> after_hit -> after_cast -> cooldown -> turn_end
```

Mỗi effect khai báo phase hợp lệ. Event queue dùng `(phase, priority, insertion_order)` để ổn định. Đặt giới hạn `max_events_per_action`, `max_trigger_depth`, `max_summons`, `max_repeats` nhằm ngăn loop vô hạn hoặc data làm nổ CPU.

### 4.5. Stacking và ownership

Mọi status/effect lâu dài phải khai báo:

- `stack_key`: theo status, source definition hay source actor;
- `stack_mode`: `replace`, `refresh`, `add`, `max`, `independent`;
- `max_stacks`, duration, dispel category;
- snapshot hay dynamic scaling;
- source actor/action/path và definition version.

Runtime state thống nhất thành một `RuntimeEffect` thay vì đồng thời `Effect`, `DoTEffect`, tier fields và `StatusOp`. Giai đoạn chuyển tiếp có adapter từ compiled op sang model cũ.

### 4.6. Effect ngoài combat

Envelope, versioning và validation được dùng chung, nhưng không ép mọi domain vào combat runtime. Side effect database phải được trả thành command để handler áp trong transaction.

| Domain | Ví dụ | Đích thực thi |
|---|---|---|
| Combat | damage, shield, summon, control | combat snapshot → canonical events |
| Passive | HP, luck, storage, path bonus | loadout snapshot → derived modifiers |
| Exploration/world | portal, reveal, capture | world snapshot → world commands |
| Refinement/cultivation | success, mana tier, stress | operation snapshot → modifiers/commands |
| Periodic/`other` | regen, stones, lifespan, thought | tick snapshot → worker commands |
| House/support | namespace chưa typed | chỉ chốt sau inventory Phase 0 |

Phase 0 phải tạo matrix **mọi key/action code → parser → compiler → executor → persistence owner → tests**. Không xóa legacy namespace nào trước khi có đích migration hoặc xác nhận chỉ là metadata.

## 5. Thiết kế sát chiêu mới

### 5.1. Tách definition khỏi ownership

Đề xuất các bảng:

```text
killer_move_definitions
  id, version, name_key, rank_required, path, effect_spec,
  costs, cooldown, cast_time, enabled, content_hash, created_at

killer_move_recipe_slots
  definition_id, definition_version, slot_key, slot_type,
  required_gu_id, min_rank, max_rank, quantity,
  allowed_variants/tags, optional, consume_policy

player_killer_moves
  player_id, definition_id, definition_version,
  mastery, usage_count, unlocked_at

player_killer_move_bindings
  player_killer_move_id, slot_key, player_gu_id

combat_loadout_killer_moves
  player_id, loadout_id, loadout_slot, player_killer_move_id
```

Không lưu `damage_value`, `defense_value`, `effects` như output mutable của từng người chơi. Nếu cần cache compiled binary/JSON, lưu cùng `content_hash`, compiler version và có thể tái tạo.

Invariant khi implement: definition PK `(id, version)`; recipe slot PK/FK `(definition_id, definition_version, slot_key)`; ownership có ID riêng; binding unique `(ownership_id, slot_key)` và FK tới `player_gu`; thêm CHECK rank/cost/version cùng index theo player/loadout. Policy delete/trade phải vô hiệu hóa binding hoặc từ chối thao tác nguyên tử, không để recipe thiếu. `selected BOOLEAN` không đủ cho nhiều loadout nên không dùng.

Theo quy tắc schema hiện có, khi triển khai phải tạo schema domain phù hợp và mọi cột mới trên bảng cũ phải có `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` ngay sau `CREATE TABLE`.

### 5.2. Công thức chính xác, không ghép bừa

Mỗi recipe khai báo slot định danh:

```json
{
  "killer_move_id": "great_strength_divine_fist",
  "version": 1,
  "slots": [
    {"key":"core", "type":"CORE", "gu_id":"GU_CHARGING_CRASH", "min_rank":4},
    {"key":"ten_jun", "type":"SUPPORT", "gu_id":"GU_TEN_JUN_STRENGTH", "min_rank":4},
    {"key":"self_strength", "type":"SUPPORT", "gu_id":"GU_SELF_STRENGTH", "min_rank":3}
  ]
}
```

- So khớp theo `gu_id + rank constraint`, không dựa vào effect tags tổng quát.
- Slot order chỉ có ý nghĩa khi recipe nói rõ; mặc định canonicalize theo `slot_key`.
- Quy định duplicate, vital Gu, trạng thái ACTIVE/equipped/hunger, một instance dùng cho bao nhiêu move.
- Unlock/craft trong transaction, lock các `player_gu` liên quan (`FOR UPDATE`) nếu operation thay đổi ownership/state.
- Lúc vào trận revalidate availability; compile từ definition version đã sở hữu.
- Không cho tên người chơi quyết định identity; tên hiển thị nằm ở definition/localization.

Cast phải là một aggregate transaction có lock order cố định: player/combat row → ownership/move → bindings và `player_gu` theo ID đã sort. Dùng conditional update cho mana/hunger, action request ID để idempotent, policy retry deadlock rõ ràng; cùng transaction xử lý HP/backlash, cooldown, usage/mastery và combat state. Catalog snapshot/revision phải được capture tại điểm xác định trước khi mutate.

### 5.3. Version và migration

- Definition published là immutable. Cân bằng mới tạo version mới.
- Chính sách theo mùa có thể auto-upgrade ownership sang version mới; replay cũ vẫn giữ version/hash cũ.
- Combat snapshot ghi `definition_id`, `definition_version`, `catalog_revision`, `compiler_version`, RNG seed.
- Sát chiêu legacy được map sang definition nếu khớp recipe; không khớp thì đánh dấu `legacy_disabled` và hoàn trả/cho người chơi chọn chuyển đổi, không âm thầm biến thành move khác.

Gu definition cũng phải immutable/versioned; chỉ version hóa sát chiêu là chưa đủ. Có thể tách `gu_definitions(id, rank, version, effect_spec, content_hash)` và `catalog_revision_entries`, hoặc archive payload theo revision. Retention phải giữ revision lâu hơn trận active và thời hạn replay; restart vẫn phục hồi được snapshot cũ, không chỉ dựa vào pointer RAM.

Trước cutover cần dry-run report (match duy nhất/mơ hồ/không match/binding mất hoặc khác owner), mapping table deterministic, marker backfill idempotent, refund ledger có idempotency key, bảo toàn name/mastery/usage và rollback runbook.

## 6. Validation và công cụ dữ liệu

### 6.1. Pipeline publish

```text
CSV/Admin draft
 -> parse strict (DisallowUnknownFields)
 -> schema + semantic validation
 -> cross-reference validation
 -> compile all definitions
 -> simulation smoke tests / budget checks
 -> publish atomic CatalogSnapshot
 -> invalidate cache by revision
```

Validation bắt buộc:

- schema/version hợp lệ, không unknown field/op/mechanic/status/target/trigger;
- số hữu hạn, range/unit đúng, duration/count dương và có trần;
- reference Gu/status/monster/killer move tồn tại;
- recipe có đúng một core, slot unique, rank hợp lệ;
- graph trigger không cycle nguy hiểm, budget worst-case nằm trong ngưỡng;
- every registered mechanic có test và every used mechanic đã registered;
- generate report key/action coverage cho toàn catalog.

Admin import nên validate trong transaction/staging; chỉ kích hoạt revision khi toàn bộ dataset đạt. Không import từng dòng rồi để registry ở trạng thái nửa cũ nửa mới. `atomic.Pointer` chỉ giải quyết một process: cần `catalog_revisions` persisted (`draft/validated/active`), mỗi process compile đủ rồi báo readiness/checksum và poll revision active. Chốt startup policy (fail hay serve last-known-good), rollback activation và retention snapshot khi process/Redis lỗi.

### 6.2. Quan sát vận hành

Metric/log tối thiểu:

- compile failures theo dataset/key/definition;
- unknown/disabled mechanic (mục tiêu phải bằng 0);
- execution count và latency theo opcode/mechanic;
- event count/depth/action, budget abort;
- damage/heal/resource delta theo definition version;
- catalog revision và checksum đang chạy trên mỗi process.

Thêm endpoint/admin report “catalog health”, không trả implementation internals cho client công khai.

## 7. Tối ưu hiệu năng

1. Parse và validate JSON **một lần lúc publish/startup**, không trong lượt đánh.
2. Compile string IDs/selectors/expressions thành enum/index; hot loop không query DB và hạn chế map lookup.
3. Chụp `CombatContextSnapshot` một lần gồm region, stat, mastery, đạo ngân, character effects và loadout.
4. Cache `CompiledDefinition` theo `(kind,id,version,content_hash,compiler_version)`, không cache kết quả đã scale theo người chơi.
5. Dùng immutable catalog pointer và atomic swap để read không lock; trận đã bắt đầu giữ pointer revision cũ.
6. Preallocate event buffer theo compiled upper bound; enforcement budget bảo vệ CPU/memory.
7. Chỉ benchmark sau khi có correctness baseline; mục tiêu đầu là không DB I/O trong simulator và kết quả deterministic.

## 8. Lộ trình triển khai

### Phase 0 — Đóng băng contract và lập baseline

- Inventory tự động toàn bộ key trong CSV/DB, nơi đọc và nơi thực thi.
- Thêm golden combats cho damage, shield, heal, status, multi-target, killer move hiện tại.
- Ghi rõ units, stacking, targeting và thứ tự hiện hành; đánh dấu bug nào cần preserve tạm thời.
- Chặn thêm field effect mới vào model cũ trừ hotfix.

**Exit:** report coverage bằng máy; replay cùng complete snapshot/action sequence/RNG streams cho cùng canonical events.

### Phase 1 — Strict validation trước khi đổi runtime

> Hoàn thành ngày 2026-09-15: manifest owner/disposition, catalog-wide coverage report, CI gate cho toàn `Gu.csv`, Admin preview/audit và lỗi theo JSON path đã được triển khai. Các disposition `partial`, `migrate`, `known_bug` là backlog đầu vào cho Phase 2, không được hiểu là effect đã hoạt động đầy đủ.

- `ValidateGuCatalog` phát hiện unknown/ignored keys; field đã phân loại để migrate chỉ warning, field chưa phân loại bị chặn tại write/import boundary.
- Phân loại mọi key CSV bằng manifest: supported, partial, metadata, migrate hoặc known bug, kèm runtime owner.
- CI/test fail với dữ liệu mới chưa phân loại hoặc không hợp lệ; Admin preview/audit trả lỗi theo definition và JSON path.

**Exit:** không còn effect “im lặng”; 100% key có owner và disposition.

### Phase 2 — Runtime v2 song song

- Tạo package module effect runtime với compiler, opcodes, event queue, deterministic RNG.
- Implement primitives cho damage/heal/shield/status/resource/targeting.
- Adapter `CompiledAction -> SimAction` chỉ cho tập opcode v1-compatible liệt kê rõ; compiler reject mọi phép chuyển làm mất semantics. Multi-status/trigger/summon/mechanic phải shadow-execute thành canonical events, không ép phẳng.
- Compile dual-path trong test và diff kết quả v1/v2.

**Exit:** bộ golden tương đương cho primitives; runtime không truy cập DB.

### Phase 3 — Catalog sát chiêu exact recipe

- Thêm schema definition/recipe/ownership có version.
- CRUD/import admin dùng validator/compiler chung.
- Endpoint preview recipe trả thiếu slot, rank và cost rõ ràng.
- Transactional unlock/migration legacy; feature flag UI mới.

**Exit:** không thể tạo arbitrary combo; mọi move tham chiếu recipe/version hợp lệ.

### Phase 4 — Chuyển engine sang event runtime

- Engine consume `CompiledAction` trực tiếp.
- Hợp nhất status runtime và phase semantics.
- Chuyển manual/auto/PvP/arena/tribulation sang cùng một build path.
- Xóa query DB khỏi action compile/hot loop.

**Exit:** một compiler, một executor cho mọi mode; deterministic replay qua snapshot.

### Phase 5 — Migrate effect đặc biệt và dọn legacy (liên tục)

- Mỗi batch 10–20 mechanics: spec, implementation, unit test, golden test, telemetry.
- Xóa `GetFlatGuEffects`, compound rules tùy ý, field `SimAction`/`ActiveEffect` không còn dùng.
- Sau ít nhất một release ổn định mới xóa bảng/cột legacy; trước đó backfill và read fallback.

**Exit:** thêm primitive effect chỉ cần data; thêm effect độc nhất chỉ cần mechanic adapter + registration + test, không sửa handler/simulator core.

## 9. Chiến lược test

### Compiler/catalog

- table tests cho từng opcode và từng lỗi JSON path;
- property/fuzz: JSON bất kỳ không panic, NaN/Inf/overflow bị từ chối;
- catalog test duyệt toàn bộ CSV và DB fixture;
- recipe canonicalization và exact matching.

### Runtime

- golden event sequence theo seed;
- phase ordering, stacking, expiry, dispel, immunity, miss/crit;
- trigger loop budget và summon/repeat caps;
- invariant: HP/resource hữu hạn, không âm ngoài contract, target luôn thuộc snapshot.

### Tích hợp

- cùng compiled action + complete snapshot + explicit action sequence + RNG streams cho manual/auto/skip ra cùng canonical effect events; orchestration riêng của từng mode được test tách biệt;
- PvE/PvP/arena/tribulation, reconnect và combat persistence;
- update catalog giữa trận không đổi kết quả snapshot;
- migration legacy idempotent và rollback-safe.

### Hiệu năng

- benchmark compile per definition (cold/cache hit);
- benchmark turn với 10/100/1.000 events;
- race test cho registry atomic swap và nhiều combat đồng thời;
- đặt budget regression cụ thể sau baseline, ví dụ p95 turn không tăng quá 10% trước khi tối ưu sâu.

## 10. Thứ tự thay đổi file dự kiến

1. `backend/internal/effectruntime/`: types, compiler, validator, executor, registry.
2. `backend/internal/simulator/`: nhận compiled actions/events; bỏ dần field effect chuyên biệt.
3. `backend/internal/handlers/combat_sim_build.go` và các builder liên quan: dựng snapshot rồi gọi compiler duy nhất.
4. `backend/internal/db/schema_*.go`: definition/recipe/ownership/version và các `ALTER ... IF NOT EXISTS` cần thiết.
5. `backend/internal/handlers/admin_*`: staging validation, publish revision, health report.
6. `backend/data/`: migrate effect schema v2 và catalog killer moves theo batch.
7. `frontend/src/store/slices/` + UI sát chiêu: đọc definition, recipe requirement và validation errors từ server; frontend không tự tính effect authoritative.
8. `docs/GAME_DATA.md`, `docs/API.md`, `WIKI.md`, `PROJECT.md`: cập nhật cùng phase tương ứng.

## 11. Những điều không nên làm

- Không thêm một boolean/float field vào `ActiveEffect` và `SimAction` cho mỗi Cổ mới.
- Không dùng reflection trên tên key trong hot loop.
- Không cho phép script tùy ý trong DB; khó sandbox, version, debug và đảm bảo deterministic.
- Không để frontend tính damage/công thức authoritative.
- Không hot reload bằng mutate map global tại chỗ.
- Không migrate tất cả 4.334 dòng trong một big-bang release.
- Không xóa legacy trước khi có dual-run/golden/migration audit.
- Không dùng tags/effect coincidence để tự phát minh sát chiêu; tags chỉ dùng tìm kiếm, gợi ý hoặc validation phụ.

## 12. Checklist chốt thiết kế trước khi code

- [ ] Chốt phase/event ordering và semantics hit/crit/defense.
- [ ] Chốt unit conventions và rounding/clamp.
- [ ] Chốt targeting grammar, team/grid semantics.
- [ ] Chốt stacking/refresh/source ownership.
- [ ] Chốt deterministic RNG và replay snapshot.
- [ ] Chốt lifecycle catalog draft/publish/rollback/version.
- [ ] Chốt policy upgrade killer move definition version.
- [ ] Chốt migration/refund cho sát chiêu legacy không match.
- [ ] Chốt execution budgets và failure mode (skip effect hay abort action/trận).
- [ ] Chốt localization/log event contract.
- [ ] Chốt security permissions và audit log cho admin publish.
- [ ] Chốt SLO benchmark và telemetry.

## 13. Tiêu chí thành công

- 100% effect key và mechanic trong catalog được validate; không unknown key bị bỏ qua.
- Một semantics duy nhất dùng cho mọi combat mode.
- Replay cùng complete snapshot, action sequence, catalog/compiler/runtime versions và RNG streams cho cùng canonical event hash/result.
- Không có DB/network I/O trong compile đã cache hoặc execution hot loop.
- Recipe sát chiêu exact và versioned; không tạo được tổ hợp tùy ý.
- Thêm effect primitive không sửa engine; thêm effect độc nhất chỉ sửa một mechanic module cục bộ và test của nó.
- Catalog publish atomic, rollback được, có checksum/health report.
- Legacy migration đo đếm được, idempotent và không làm mất Cổ/sát chiêu của người chơi.

---

[^gu-model]: `backend/internal/models/gu.go`, `GuEffects` và `GuDictionaryItem`.
[^gu-schema]: `backend/internal/db/schema_dictionaries.go`, định nghĩa `gu_dictionary`.
[^gu-registry]: `backend/internal/models/gu.go`, `LoadGuRegistry`.
[^gu-scan]: `backend/internal/models/gu.go`, `GuEffects.Scan` và `ActiveEffect.normalize`.
[^sim-action]: `backend/internal/simulator/models.go`, `SimAction`, `Effect` và `DoTEffect`.
[^calculate-gu]: `backend/internal/handlers/combat_effects.go`, `CalculateGuEffectForCombatant`.
[^build-action]: `backend/internal/handlers/gu_effects.go`, `BuildCombatAction`.
[^km-schema]: `backend/internal/db/schema_player_items.go`, `killer_moves` và `killer_move_slots`.
[^flat-effects]: `backend/internal/models/gu.go`, `GetFlatGuEffects`.
[^compound]: `backend/internal/handlers/killer_move_compounding.go`, `compoundRules` và `detectCompounds`.
[^synergy]: `backend/internal/handlers/killer_move_compounding.go`, `synergyRecipes`.
[^gu-active]: `backend/internal/models/gu.go`, `ActiveEffect`.
[^status-tier]: `backend/internal/simulator/models.go`, `TierToPercent`, `TierToDuration`.
