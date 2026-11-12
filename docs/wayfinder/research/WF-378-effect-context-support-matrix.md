# WF-378: Ma trận hỗ trợ effect theo context đang hoạt động

**Research scope:** kiểm kê code hiện có trên `turnbase`; không đề xuất refactor và không đánh đồng “Block v2 hợp lệ” với “adapter có thể thực thi”. Realtime combat được xem là retired theo quyết định Wayfinder; general deployed formations ngoài Teleport Array và Gu House không nằm trong scope.

## Kết luận nhanh

Block v2 là hợp đồng dữ liệu/biên dịch chung: nó định nghĩa trigger, target, condition, operation, subtype và native hook. `effectcatalog`/`gameplay/effects.Compiler` kiểm tra tính hợp lệ chung; đó **không** phải ma trận executor cho từng sản phẩm. Mỗi context hiện có adapter và giới hạn riêng, và một số trigger vẫn chỉ là từ vựng schema hoặc projection cho một consumer hẹp.

Cụ thể:

- **Turn-based combat:** vận hành qua `simulator`/handler turn-based. Block operation tổng quát chỉ được adapt thành action cho `ON_ACTION`; trigger khác phần lớn chạy như native `UNIQUE` combat mechanics. Các target/subtype được adapter từ chối nếu chưa hỗ trợ.
- **World ON_USE:** `HEAL` self-target cho HP/mana và các status `FLYING`, `HASTE`, `STEALTH` self-target. Toàn bộ command được preflight trước khi mutate. Trigger `ON_USE` tự nó không làm effect combat như damage có thể chạy được ngoài combat.
- **Teleport Array arrival, Gu House ON_USE và Sát Chiêu ON_USE** dùng cùng world executor subset; đây là nhiều producer/context của một executor, không phải mọi operation đều dùng được ở mọi nơi.
- **ALWAYS_ON/derived:** có projection cho stat/passive, capability, growth và một số native behavior. Projection phải được consumer gọi rõ ràng; `Compiled` không tự động áp toàn bộ block.
- **ON_CONSUME/ON_USE Gu event dispatcher:** hiện dispatch `UNIQUE` native mechanic theo trigger. Consumer không gọi `Evaluate` để thực thi danh sách operation thường trong đường này.
- **ON_TICK:** một số worker/projection đọc danh sách `UNIQUE` native mechanic cụ thể (resource regen, stone generation, hunger/growth). Đây không phải executor tổng quát cho mọi operation có trigger ON_TICK.
- **Refinement/deduction:** deduction đang đọc `ALWAYS_ON UNIQUE DEDUCTION_INSTANT_FINISH`; đây là native hook chuyên biệt. Không tìm thấy generic `REFINE` effect context/executor cho refine recipe trong các handler refinement.
- **Gu House combat / Sát Chiêu:** composition lưu Block v2 snapshot và provenance nhưng executor combat vẫn có giới hạn riêng; Gu House action yêu cầu `ON_ACTION`, đúng một operation, không condition, target SELF/TARGET, subtype rỗng/FLAT. Sát Chiêu có selector hỗ trợ rộng hơn nhưng vẫn dùng combat adapter.
- **Realtime:** không có package `backend/internal/realtime` trên branch này. Kết luận retirement là scope theo Wayfinder; matrix này không tính realtime là context cần hỗ trợ.

## Ma trận context

| Context đang hoạt động | Trigger/đầu vào | Điều kiện, target, operation thực thi | Native hook / state owner | Schema-valid nhưng không chạy được ở đây |
|---|---|---|---|---|
| Turn-based combat (Gu action) | Block `ON_ACTION` được lấy từ compiled Gu effect và adapt thành simulator operation. Combat native mechanics có các trigger combat riêng như `ON_CAST`, `ON_HIT`, `ON_TURN_START`, `ON_DEATH`, `ON_BATTLE_START`, `BEFORE_HIT`, `BEFORE_ENEMY_ATTACK`, `COMBAT_PASSIVE`. | Adapter combat operation nhận target SELF/TARGET/ROW/COL/ALL_ENEMIES/ALL; ALL_ALLIES bị loại. Nó hỗ trợ các nhánh DAMAGE, SHIELD, HEAL, DIRECT_HP, BARRIER, CLEANSE, STATUS, UNIQUE với subtype/target hạn chế ở adapter. Condition của operation được chuyển sang simulator và lọc target. | Native mechanics đi vào `CombatMechanic` rồi simulator xử lý; combat state do simulator/turn-based handler sở hữu. | `PASSIVE`, `CAPABILITY`, `ON_TICK`, `ON_USE`, `ON_CONSUME` không tự chạy trong combat action adapter. `ON_ACTION` operation có subtype/target chưa được adapter nhận sẽ bị fail khi build action. Condition field phải có dữ liệu runtime mà evaluator tương ứng cung cấp. |
| World ON_USE: dùng consumable Gu | `UseConsumableGu` phát `ON_CONSUME` rồi `ON_USE` qua `HandleGuEventTx`. Dispatcher hiện chỉ quét `Compiled.NativeMechanics()`. | Đường native event dispatcher không gọi generic `Evaluate`; operation thường trong block không được thực thi qua đường này. Generic world executor riêng chỉ nhận self-target HEAL (HP/mana) và STATUS FLYING/HASTE/STEALTH; không stack policy; heal dương; status có giới hạn value/duration/chance. | Native event action mutates player/world through handler transaction. World-use commands mutates player HP/mana/status. | Damage, shield, ally/enemy target, unsupported status, stack policies, general native combat action đều không chạy trong world ON_USE subset. Một block v2 hợp lệ vẫn có thể bị adapter từ chối. |
| Teleport Array arrival | Product composition chọn các effect có travel support rồi snapshot chúng thành `ON_USE`; arrival evaluate và chuyển command cho `applyWorldUseCommands`. | Mỗi node snapshot phải là một block, một operation; chỉ `worldUseOperationAllowed` (self HEAL hoặc status subset) mới qua. Arrival dùng cùng command adapter với world ON_USE. | Teleport Array profile/snapshot sở hữu composition và arrival effect; mutation được thực hiện trên player trong flow spatial transition. | Mọi block không phải ON_USE, nhiều operation/block, target khác SELF, unsupported status/heal đều bị từ chối. `Compiler.Compile` thành công riêng lẻ không đủ để pass validation profile. |
| Gu House world ON_USE | Chọn effect `TravelSupported`, dựng lại một ON_USE block, `Evaluate`, rồi gọi `applyWorldUseCommands`. | Cùng giới hạn của world executor. Effect phải được explicit select; player mana/hunger/status được cập nhật trong transaction. | Gu House record/profile giữ core Gu và effect provenance; handler giữ transaction và world mutation. | Combat-only effect, unsupported ON_USE operation/target bị loại từ option list hoặc bị executor từ chối. |
| Sát Chiêu world ON_USE | `getKillerMoveEffectOptions` chỉ đưa ra ON_USE option qua `worldUseOperationAllowed`; sử dụng dựng lại Block và gọi Evaluate + world executor. | Cùng world ON_USE subset; conditions evaluate từ actor snapshot/rank. | Player killer move/profile giữ compiled block snapshot, source refs và composition graph; handler thực hiện mana/hunger/player mutations. | Các ON_USE operation chưa được world executor hỗ trợ không selectable/executable. |
| Derived/passive player stats | `ALWAYS_ON STAT_MOD` có projection qua `Compiled.StatValue`; `PASSIVE`/`COMBAT_PASSIVE` không nên giả định tương đương `ALWAYS_ON`. `CAPABILITY` và growth có projection riêng. | Hiện consumers đọc các stat cụ thể và một số trigger-specific values; ví dụ `GuStatValue` chỉ đọc ALWAYS_ON subtype; `gamefx.Get` tổng hợp character effects/active Gu; `player_stats` đọc một projection ON_ACTION cụ thể cho ALL/MULTIPLY_PERCENT. | Canonical player snapshot/stats và `gamefx` là owner của derived player view; inventory/equipment chọn active Gu instances. | Một `STAT_MOD` schema-valid với trigger/stat/subtype không được consumer đọc sẽ không tự động làm thay đổi derived stats. Trigger `PASSIVE` tồn tại trong catalog nhưng không có evidence về general evaluator bao phủ mọi passive operation. |
| ON_TICK / resource growth | `Compiled.Growth()` lọc ON_TICK blocks, nhưng production consumers cụ thể chủ yếu đọc `NativeMechanics()`: worker hunger scans stone-generation hooks; model projections scan MANA/THOUGHT/etc regen codes. | Consumer nhận đúng native code cụ thể và value; mỗi projection có rank/factor/default handling riêng. | Worker/database batch update sở hữu stone tick; model stat/resource projection sở hữu tính regen/hour. | Generic ON_TICK DAMAGE/HEAL/STATUS/STAT_MOD không có tick executor tổng quát được tìm thấy. `Growth()` projection chỉ trả data; bản thân nó không tạo side effect. |
| Refinement/deduction | `DEDUCTION_INSTANT_FINISH` được đọc như `ALWAYS_ON UNIQUE` trong deduction handler. | Consumer đọc base/per-rank/min/max chance, tính xác suất và roll riêng. Refinement success bonuses xuất hiện trong gamefx projection qua Gu stats; đây không phải trigger `REFINE` executor. | Deduction handler sở hữu roll và cost; player state được commit trong mutation. | Có native domain label `deduction`, nhưng không có general-purpose `REFINE` trigger hoặc executor để áp arbitrary operation vào refine recipe. Triggers `ON_ACTION`/`ON_TICK` không tự áp vào thao tác refine. |
| Gu event ON_CONSUME / ON_USE | `HandleGuEventTx` lọc native mechanics chính xác theo chuỗi trigger và gọi action-code handler. | Native hook phải được đăng ký cho trigger đó, và `handleActionCodeTx` phải có implementation tương ứng. Consumer-specific switch làm state mutation. | Player mutation transaction; hook có thể tác động world cần transaction theo comment của handler. | Regular Block operations không được Evaluate ở dispatcher. Trigger tồn tại trong schema/catalog nhưng chưa đủ để chứng minh generic executable support. |
| General deployed formations | Không có context/executor catalog rộng hiện tại. Teleport Array là product cụ thể. Gu House là product cụ thể, không phải general formation object. | N/A cho general formation. Party/pet/NPC formation routes thể hiện placement/lineup, không có effect deployment executor theo nghiên cứu trước #377. | Không có general deployed formation lifecycle/owner được tìm thấy. | Không suy diễn rằng dùng chung Block v2 đồng nghĩa có executor cho mọi formation product. |
| Realtime combat | Retired khỏi đích Wayfinder; branch hiện tại không có `backend/internal/realtime/`. | Không nằm trong matrix hoạt động. | N/A. | Không dùng effect hỗ trợ realtime làm tiêu chí chặn spec turn-based. |

## Các lớp “hỗ trợ” cần phân biệt

1. **Envelope/schema validity:** Block v2 canonical validator kiểm tra field, trigger, target, condition, operation vocabulary, trigger-operation compatibility, enum/range và native params. Điều này trả lời “dữ liệu có cấu trúc/cú pháp hợp đồng không?”.
2. **Generic compilation:** `gameplay/effects.Compiler` compile thành immutable typed representation và kiểm tra vocabulary, trigger-operation/target, status, native params. Nó không biết product profile hay transaction owner.
3. **Context support:** Adapter/context-specific selection quyết định operation nào được nhận, như `killerMoveOperationAllowed`, `worldUseOperationAllowed`, `adaptBlockCombatOperations`, hoặc profile validator của Teleport Array/Gu House.
4. **Execution:** handler/simulator/worker thực sự áp command hoặc native mechanic vào owner state. Chỉ tầng này mới chứng minh effect đang “chạy”.

## Source map

### Contract và validation

- Trigger, target, operation, block DTO: [`contract.go`](../../../backend/internal/gameplay/effects/contract.go#L7).
- Generic compiler checks vocabulary, operation-trigger compatibility, target/subtype, status/native mechanics: [`runtime.go`](../../../backend/internal/gameplay/effects/runtime.go#L121).
- Generic `Evaluate` emits commands for matching trigger/conditions; it does not mutate domain state: [`runtime.go`](../../../backend/internal/gameplay/effects/runtime.go#L355).
- Condition runtime fields and comparisons: [`runtime.go`](../../../backend/internal/gameplay/effects/runtime.go#L396).
- Schema catalog trigger list and trigger families: [`block_manifest.go`](../../../backend/internal/effectcatalog/block_manifest.go#L26).
- Schema operation trigger matrices: [`block_manifest.go`](../../../backend/internal/effectcatalog/block_manifest.go#L37).
- Canonical Block v2 validator checks schema constraints: [`block_v2.go`](../../../backend/internal/effectcatalog/block_v2.go#L167).

### Turn-based combat

- Build action compiles and adapts only `ON_ACTION` operations: [`combat_auto_resolve.go`](../../../backend/internal/handlers/combat_auto_resolve.go#L451).
- Combat operation target/subtype adapter and errors: [`combat_block_adapter.go`](../../../backend/internal/handlers/combat_block_adapter.go#L16).
- Trigger routing in combat mechanics has context-specific trigger cases: [`combat_auto_resolve.go`](../../../backend/internal/handlers/combat_auto_resolve.go#L703).
- Native mechanic conversion to simulator carrier: [`models.go`](../../../backend/internal/simulator/models.go#L253).
- Operation conditions filter simulator combat targets: [`effect_operation.go`](../../../backend/internal/simulator/effect_operation.go#L20).

### World ON_USE and teleport

- Exact world-use authored operation allowlist: [`world_effects.go`](../../../backend/internal/handlers/world_effects.go#L15).
- Command preflight and mutation semantics: [`world_effects.go`](../../../backend/internal/handlers/world_effects.go#L52).
- Consumable Gu dispatches both `ON_CONSUME` and `ON_USE`: [`player_gu_use.go`](../../../backend/internal/handlers/player_gu_use.go#L109).
- Event dispatcher iterates native mechanics only: [`gu_effects.go`](../../../backend/internal/handlers/gu_effects.go#L144).
- Teleport Array emits ON_USE snapshots only after travel executor support check: [`teleport_effects.go`](../../../backend/internal/handlers/teleport_effects.go#L103).
- Teleport profile requires ON_USE, one allowed operation, then compiles: [`teleport_effects.go`](../../../backend/internal/handlers/teleport_effects.go#L329).
- Arrival evaluates ON_USE and applies shared world commands: [`teleport_effects.go`](../../../backend/internal/handlers/teleport_effects.go#L398).

### Derived/passive, tick and refinement

- Compiled projections for ALWAYS_ON passives, CAPABILITY, Growth and ON_ACTION: [`runtime.go`](../../../backend/internal/gameplay/effects/runtime.go#L216).
- Gu stat projection explicitly reads ALWAYS_ON subtype: [`gu.go`](../../../backend/internal/models/gu.go#L430).
- Character/Gu derived effects aggregation: [`fx.go`](../../../backend/internal/gamefx/fx.go#L492).
- Explicit ON_ACTION derived player stat projection: [`player_stats.go`](../../../backend/internal/handlers/player_stats.go#L303).
- ON_TICK stone worker reads native hooks by trigger/code and batch updates: [`hunger.go`](../../../backend/internal/worker/hunger.go#L93).
- ON_TICK regen projection reads only requested native codes: [`gu.go`](../../../backend/internal/models/gu.go#L1320).
- Deduction reads ALWAYS_ON `DEDUCTION_INSTANT_FINISH` and performs a local chance roll: [`refine_deduce.go`](../../../backend/internal/handlers/refine_deduce.go#L95).
- Native hook declares its own trigger/domain/params: [`block_manifest.go`](../../../backend/internal/effectcatalog/block_manifest.go#L194).

### Composition products

- Sát Chiêu options explicitly classify combat vs outside-combat support: [`killer_moves.go`](../../../backend/internal/handlers/killer_moves.go#L193).
- Gu House combat action requires ON_ACTION/one op/no conditions/target and subtype restrictions: [`gu_house.go`](../../../backend/internal/handlers/gu_house.go#L288).
- Gu House world ON_USE uses the world adapter: [`gu_house_use.go`](../../../backend/internal/handlers/gu_house_use.go#L105).
- Prior scope finding for deployed formations is recorded by Wayfinder ticket [“Chốt phạm vi Trận Pháp triển khai trong đặc tả effect”](https://github.com/VipPro67/GuWorld/issues/377): existing Teleport Array and Gu products only; general deployed formations deferred.

## Limitations

This is a source-code inventory, not a claim that every data row has been tested against each context. Catalog tests and existing handler/simulator tests provide examples, but this report did not run tests. `TriggerCatalog` registration is treated as schema vocabulary only; runtime support is claimed only where a concrete consumer/adapter was found. The matrix records explicit gaps instead of inferring that a generic compiler projection is an executor.
