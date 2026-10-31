# Thiết kế Tiên Khiếu — Tách Tiên Nguyên & Cải cách Nuôi Trồng

> Trạng thái: PROPOSAL (chưa triển khai). Căn cứ: Cổ Chân Nhân (Reverend Insanity) ch. 568/609/610/616/622/713/795/820/821/1310/1389 + audit code hiện tại (mục nào cũng kèm file:line).

---

## 1. Vấn đề hiện tại

### 1a. Mana đang là một pool kiêm cả hai vai trò

`players.current_mana` hiện **vừa là Chân Nguyên vừa là Tiên Nguyên**:

- Cổ Tiên (rank ≥ 6) có `MaxMana = 99999999999` (`services/player_service.go:124`, `handlers/cultivate.go:199`) — pool "vô hạn", không có khái niệm trữ lượng.
- Regen của Cổ Tiên = `essence_production_per_year / 10` mỗi giờ (`services/player_service.go:150-156`) — tức **sản lượng Tiên Nguyên bị biến thành "tốc độ hồi mana"** thay vì một tài nguyên tích lũy được.
- Cùng một pool bị tiêu cho đủ loại: Cổ phàm, Tiên Cổ, Sát Chiêu, Tiên Cổ Ốc — chi phí được tính khác nhau theo rank người dùng (`gamefx/energy.go` EnergyCost mortal/immortal branches), nhưng nguồn trừ thì chỉ có một.
- Tiên Nguyên Thạch nạp thẳng "mana" (`handlers/player_recharge.go:46-90`) — nửa vời giữa "đá" và "sản lượng sản xuất".
- `immortal_apertures.immortal_essence` hiện chỉ là **bản sao của `players.current_mana`** (4 điểm ghi: `services/player_service.go:505,579`, `aperture_core.go:157,262`, `player_recharge.go:71`) — không phải pool thật; nó chỉ được dùng làm "proxy thức ăn" cho society sim (`worker/society_simulation.go:85-129`).

### 1b. Nuôi trồng rời rạc

- **Bug tĩnh:** worker luyện tự động `processApertureRefinement` (`worker/variant_refinement.go:38`) **không được gọi bởi bất kỳ loop nào** — queue `aperture_refinement_queue` nằm im mãi. Docs hứa 10 phút/tick nhưng main.go chỉ wire Society (5m) + Gu Master Spawning (30m).
- **Phân công lao động vô nghĩa:** `AssignWorkerToNode` (`variant_economy.go:98`) không ảnh hưởng yield — công thức thu hoạch chỉ tính `dân số tổng × cường độ lao động` (`aperture_collect.go:592-608`), nên UI chỉ định công nhân hiện không tạo khác biệt.
- **Công thức yield trôi dạt:** yield RESOURCE hiện dùng `level × ApertureYieldLevelStep (0.5)` (`gamefx/aperture_yield.go:21`), trong khi `production_amount` của template (300/250/210 chất liệu / 10h) **không được dùng**; nhánh `production_type="Stones"` là mã chết (không CSV nào khai).
- **Interval bị ghi đè:** schema boot ép `production_interval_hours = 10` cho mọi node RESOURCE (`schema_aperture.go:165-168`) — dữ liệu CSV trong template không được tôn trọng.
- **Sai vật lực kinh tế:** society sim dùng `immortal_essence` làm **thức ăn của dị nhân** — theo canon, thức ăn của định cư là sản phẩm nông/trại, Tiên Nguyên là nhiên liệu của Cổ Tiên.

### 1c. Cơ chế cốt truyện đang thiếu (theo audit kỳ trước)

- **Địa Linh** chưa tồn tại (ch. 610): Cổ Tiên "rơi vào cõi chết" hiện chỉ là Game Over — để lại Tiên Khiếu mồ côi vô chủ thay vì thành di sản có Địa Linh canh giữ.
- **Tổng Đạo Ngân là thước đo phòng thủ Tiên Khiếu** (ch. 820) chưa được áp vào chỉ số node/tháp/Tiên Khiếu.
- **Biome theo phai chính**, **Phúc Địa công cộng cho tổ chức** (ch. 795), **Động Thiên** (8–9 chuyển) — để dành cho giai đoạn sau.

---

## 2. Mô hình mục tiêu

### 2a. Hai đồng tiền riêng, mỗi đồng một nguồn

| | **Chân Nguyên** (phàm) | **Tiên Nguyên** (tiên) |
|---|---|---|
| Lưu trữ | `players.current_mana` (giữ nguyên) | `immortal_apertures.immortal_essence` (BIGINT — **nâng thành pool thật**) |
| Trữ lượng | `EffectiveMaxMana` (cap thực — bỏ cap 1e99 cho rank ≥ 6) | `immortal_apertures.essence_capacity` (cột mới, khởi tạo theo grade: ≈ 10/20/30/30/50 năm sản lượng) |
| Dùng cho | Cổ phàm (rank ≤ 5): cast trong + ngoài trận, dịch chuyển, thám tử, luyện Cổ… | **Tiên Cổ (rank ≥ 6)**: cast trong trận + Sát Chiêu + Tiên Cổ Ốc strikes, upkeep status tier immortal, sửa chữa node/áp, tính năng tiên cấp cao |
| Sản xuất | regen theo thiên phú mỗi giờ (mortal table giữ nguyên) | Tiên Khiếu sản ra mỗi internal năm: `essence_production_per_year × time_rate`, nhân hệ số thuần Đạo + dân số, trừ nhược điểm xung đột Đạo |
| Nạp | Nguyên Thạch (như hiện tại) | **Tiên Nguyên Thạch → nạp thẳng Tiên Nguyên** (số hạt, hệ số theo rank) |

Quy tắc chia pool — **tier của cổ/move đang cast, không phải rank người chơi**:

- Cast cổ tier ≤ 5 → trừ Chân Nguyên.
- Cast Tiên Cổ / Sát Chiêu / Ốc strike (tier ≥ 6) → trừ Tiên Nguyên.
- Cổ Tiên dùng cổ phàm vẫn trừ Chân Nguyên; ai chưa có Tiên Khiếu thì không tiêu Tiên Nguyên.

### 2b. Vòng đời nuôi trồng đủ nghĩa

```
Tiên Khiếu mở (grade 1–5: area, time_rate, sản lượng cơ bản)
   → đặt node: Nông Trai / Trại Chăn Nuôi / Mỏ → chất liệu + THỨC ĂN (tồn kho lưu trữ)
   → phân công dị nhân vào node (aperture_node_workers) → +yield từng node (cap 2 công nhân/node)
   → dị nhân ăn từ TỒN KHO thức ăn (không đốt Tiên Nguyên) → dân số / happiness (society sim)
   → happiness + thuần Đạo → hệ số sản xuất Tiên Nguyên (ladder sản xuất)
   → Tiên Nguyên theo thời gian tích lũy (≤ essence_capacity), dùng để:
        - nuôi Tiên Cổ: cast, upkeep, Sát Chiêu, la bài trong trận
        - sửa chữa node/áp, mở tính năng tiên cấp cao
        - giữ "bệ đỡ" trước kiếp nạn kế tiếp (rủi ro "kiếp lún" khi cạn Tiên Nguyên)
   → kiếp nạn thắng: +Đạo Ngân theo ladder canon (250 : 750 : 7250 : 86750 — ch. 1389)
   → thua / quá hạn: phá node, lose Nguyên Thạch + Tiên Nguyên, aperture ×0.9 (giữ cơ chế hiện tại)
```

---

## 3. Thay đổi code theo từng phase

### Phase 0 — Sửa lỗi tĩnh (quick win, khuyến nghị làm ngay)

| # | Việc | File |
|---|---|---|
| P0-1 | **Wire refinement worker**: `worker.RunEvery(ctx, 10*time.Minute, "Aperture Refinement", dispatcher)` trong `main.go` (cạnh Society/GuMaster). Dispatcher chọn tối đa 6 job PENDING mỗi tick rồi gọi `processApertureRefinement`. | `cmd/server/main.go`, `worker/variant_refinement.go` |
| P0-2 | **Chuẩn hóa yield**: `template.production_amount × (1 + (level−1)×0.5) × multipliers` thay cho công thức level-step-only; giữ hệ số dao/ecology/labor/pop như cũ; gỡ nhánh `production_type=="Stones"` nếu CSV không dùng. | `gamefx/aperture_yield.go`, `handlers/aperture_collect.go:568-619` |
| P0-3 | **Interval theo CSV**: bỏ boot-rewrite 10h trong `schema_aperture.go:165-168`; node giữ `production_interval_hours` từ template (thêm compat ALTER nếu cần; cập nhật `GAME_DATA.md`). | `internal/db/schema_aperture.go` |
| P0-4 | **Cap Chân Nguyên cho Cổ Tiên**: thay `99999999999` bằng cap theo rank (đề xuất `1000 × 10^(rank−6)`) trong `normalizeEnergyPool` + `cultivate.go` — dọn đường cho Phase 2. | `services/player_service.go:114-130`, `handlers/cultivate.go:198-200` |
| P0-5 | **Phân công dị nhân có nghĩa**: `+0.25 yield × số công nhân` từng node (cap 2/node; hằng số trước, data-driven sau). | `handlers/aperture_collect.go:592-608`, `variant_economy.go` |
| P0-6 | **Thức ăn của dị nhân**: society sim đọc tồn kho node BREEDING/nông trại (lưu kho `food_storage` của settlement — cột đã có) thay vì tiêu `immortal_essence`; thiếu đồ ăn → đói/khủng loạn như cũ. | `worker/society_simulation.go:85-129` |

Kiểm tra P0: `go test ./internal/handlers/... -run Aperture` + worker tests; cập nhật `docs/GAME_DATA.md` (rule ApertureNodeTemplate.csv) và `WIKI.md` (mảng nuôi trồng).

### Phase 1 — Tách Tiên Nguyên (schema + ngoài trận)

**Schema (sở hữu: `internal/db/schema_aperture.go`):**
- Thêm `essence_capacity BIGINT NOT NULL DEFAULT 0` (+ compat ALTER/backfill theo grade, idempotent).
- `immortal_essence` trở thành **pool thật**: migration một lần `UPDATE immortal_apertures ia SET immortal_essence = LEAST(p.current_mana::bigint, essence_capacity) FROM players p ...`.
- Không cần pre-seed gì thêm (key sẵn là `player_id`).

**Backend:**
- Bỏ dần **4 điểm ghi mirror** (`player_service.go:505,579`, `aperture_core.go:157,262`, `player_recharge.go:71`): chuyển sang ghi pool thật; có thể **double-write** một nhịp trước khi cutover để không mất dữ liệu người chơi.
- `player_service.go`: gỡ nhánh regen immortal = sản lượng/10; regen Chân Nguyên rank ≥ 6 dùng table thiên phú như phàm (giá trị thấp, cố định).
- `player_recharge.go`: Tiên Nguyên Thạch → `immortal_essence += stones` (hệ số theo rank giữ nguyên); không đụng `players.mana` nữa. Mortals không đổi.
- Tiêu ngoài trận theo tier: `EnergyCost` phân nhánh nguồn — tier ≥ 6 trừ `immortal_essence` (UPDATE có guard `>= $cost`, thiếu → lỗi `essence_not_enough`); tier ≤ 5 trừ mana như cũ. Các điểm cần chuyển: `killer_moves.go:939-968`, luyện Cổ `refine_core.go:204-231` (luyện Tiên Cổ / rank ≥ 6 tiêu Tiên Nguyên), một phần `ascension.go` (mở nát khiếu).
- Cấp độ truy xuất: thêm `essence: {current, capacity, production_per_year}` vào response profile/player (field mới, không phá shape cũ).

### Phase 2 — Tách pool trong realtime combat

- **Compiler (`realtime/effects/profile.go`, `killer_move.go`)**: thêm `resource` ("chuan_nguyen" / "tien_nguyen") vào `CompiledSkill`, chọn theo **rank của cổ đang cast** (không phải rank người chơi).
- **`realtime/combat/skills.go`**: `SkillDef` thêm `ResourceSource`; fallback skill demo (4–9 mana) gán theo tier.
- **`realtime/sim/combat.go`**: `CombatState` thêm `Essence` (+ `MaxEssence`); cast-cost check & deduct (L362-367) chọn pool theo `skill.ResourceSource`; **regen Tiên Nguyên trong zone = 0** (projection tại admission như mana hiện tại, `realtime_join_state.go:167-172`); Form upkeep (`FormManaDrainPerTick`) chọn pool theo tier của form.
- **`realtime/gateway/admission.go`**: join-state seed cả 2 pools từ DB; **`manaSettlement`** (L1250-1322) ghi 2 deltas — `players.current_mana` (như cũ) + `immortal_apertures.immortal_essence` (batch per settle window như hiện tại).
- **`realtime/persistence/state.go`**: bump **`CombatStateVersion`** (stale rows tự bỏ qua theo AGENTS.md).
- **Frontend (`MMOHud.tsx`, `apertureSlice.ts`)**: mana bar → 2 thanh cho rank ≥ 6 (Chân Nguyên / Tiên Nguyên 2 màu); hotbar hiển thị số cost nguồn tương ứng; toast "Tiên Nguyên không đủ" cho lỗi `not_enough_essence`; view Tiên Khiếu thêm block hiển thị `{essence.current / capacity}` và tốc độ sản xuất.

Kiểm tra Phase 2: `go test ./internal/realtime/... -race` (dual-pool cast/settlement tests) + `go test ./internal/handlers -run 'KillerMove|Ascension|Recharge'`; frontend lint + build.

### Phase 2b — Gắn sản xuất với nuôi trồng (glue kinh tế)

- Tick internal năm (đã có ở `tribulation_trigger.go:408-418`) **cộng vào `immortal_apertures.immortal_essence`** (thay vì `players.mana`), cap `essence_capacity`.
- Hệ số sản xuất: `purity` (hiện trạng) + `1 + pop/1000` (dân số — tái dùng công thức node yield) + happiness positive-only; đặt trong `gamefx/aperture_yield.go` để hai loop dùng chung.
- Confirmation ammo: nếu cạn Tiên Nguyên khi vào kiếp → "kiếp lún" (kiến nghị: hệ số phòng thủ góc tệ, sẽ cân bằng ở Phase 3).

### Phase 3 — Truyện (theo plan kỳ trước, độc lập từng cái)

1. **Địa Linh**: Cổ Tiên hết thọ nguyên → Tiên Khiếu hình thành di sản có Địa Linh (obsession) — quest "quà cầu nguyện" → công nhận chủ mới, hoặc thành khu vực khám phá cho người khác.
2. **Biome theo phai chính**: mở rộng `ApertureNodeTemplate.csv` thêm cột per-path terrain/bonus.
3. **Phòng thủ canon (ch. 820)**: `aperture_hp/max_hp`, `defense_power` tháp canh scale theo `dao_marks[phai chính]` — công thức data-driven trong `aperture_yield.go`.
4. **Phúc Địa công cộng cho tông (ch. 795)**: reuse secret_realm loader (`realm:<uuid>` + FACTION_PK).
5. **Núi Danh / Bí Cảnh Thiên Địa (`point_class=FAMOUS`)**: lớp "điểm tài nguyên huyền cấp" — mỗi vùng rank ≥ 4 có 1 núi danh anchored Boss cell (guardian thú hoang tối thượng), vật liệu đặc hữu phòng luyện cao + `gu_rate` ×3 sinh cổ đặc hữu theo phái núi (Đãng Hồn → Hồn Đạo, Ngọc Huyền Xà → Độc Đạo, Phượng Vũ → Hỏa Đạo, Băng Thủy → Băng Đạo, Thi Cương → Nô Đạo). Thắng guardian mới có cửa sổ thu hoạch — cạnh tranh tự nhiên, không cần hệ thống claim. Vẫn giao dịch/di dời được như thường (canon famous-mountain trade).
6. **Động Thiên (8–9 chuyển)**: time-dilation cảm nhận nhịp riêng — cần benchmark trước khi làm.

---

## 4. Rủi ro & lưu ý

1. **Bốn góc dính chặt khi tách pool: (a) regen** (`player_service.go:150-156`), (b) **EnergyCost branches** (`gamefx/energy.go` theo IsImmortalPlayer), (c) **mirror writes** (4 điểm), (d) **nạp đá** (`player_recharge.go`). Chiến lược: schema-first, double-write, rồi cutover.
2. **Realtime settlement là điểm nhạy cảm nhất**: `manaSettlement` hiện chỉ theo 1 delta; cần tests chống lost-update cho cả 2 pools (dùng `GREATEST(0, ...)` như hiện trạng).
3. **Không làm mất giá Tiên Nguyên đã tích lũy**: backfill phải chạy đúng một lần, idempotent, theo pattern initSchema của repo.
4. **Giao thoa UI cũ/mới**: giữ response field cũ (`max_mana` cho rank ≥ 6 trả cap logic mới) cho tới khi dual-bar sống; arena/death-arena copy (`schema_death_arena.go:63-64`) giữ nguyên vì arena offline không dùng Tiên Nguyên.
5. **P0 phải đi trước** vì economies thay đổi: refinement worker chưa wire nghĩa là player đang mở "luyện tự động" mà không thực thi — cần change note.
6. **Tài liệu cần cập nhật**: `WIKI.md` (§ mana, § Tiên Khiếu/nuôi trồng, § Thạch nạp), `docs/API.md` (response shapes), `docs/GAME_DATA.md` (CSV columns + cap per grade); `PROJECT.md` khi scope mở sang Địa Linh 공공.
7. **Thứ tự bắt buộc**: P0 → P1 → P2 → P2b → P3 (P2 trước P1 sẽ đụng settlement với migration đang chạy).

## 5. Thứ tự triển khai đề xuất

```
P0 (≈1 ngày) → P1 (2–3 ngày) → P2 (3–4 ngày, nhạy cảm nhất) → P2b (1–2 ngày) → P3 (từng tính năng độc lập)
```

Mỗi phase là một PR độc lập + tests riêng; trước khi kết phase chạy `go test ./...`, `npm run lint/build`, và `-race` cho realtime.

---

## 6. Điểm Tài Nguyên — xây lại vòng nuôi trồng sát canon (thi công)

> Quyết định: **bỏ cơ chế node-RESOURCE** (đặt từng ô, gieo giống, chọn vật phẩm sản xuất) và thay bằng mô hình Điểm Tài Nguyên. Node chỉ còn là **công trình phòng thủ** (WALL/ WATCHTOWER/…). Destructive migration được chấp nhận.

### 6.1 Canon được áp vào thiết kế

| Canon | Áp dụng |
|---|---|
| Điểm tài nguyên là địa hình cô đặc theo phai, tự sinh từ Đạo Ngân (Moon Bowl — ch. 1303; blessed land page) | Điểm được **system-generated** theo trọng số `dao_marks`; người chơi không đặt từng ruộng |
| Cấp bậc điểm: nhỏ / trung / top (Treasure Moon Oasis, Profound Jade Snake Cave) | `grade` LOW/MID/TOP quyết định rate + vật liệu hiếm |
| Sinh khí — khai thác lố làm yield rớt (Frosty Peak — ch. 2152) | `vitality` 0–100: mỗi lần thu hoạch trừ; hồi theo thời gian (×time_rate); ≤30 → yield ½; =0 → phải phục dựng |
| Phục dựng là khoản đầu tư (Beehive Island) | Hành động "Bồi Dưỡng" tốn vật liệu của phai điểm |
| Kiếp nạn phá tài nguyên | Realtime arena: quái phá vào **thân thể điểm tài nguyên** (vitality là HP của điểm trong trận); thua 30% điểm vitality→0 |
| Phúc Địa nào lớn thì trước đó điểm nhiều | Số/max điểm + cấp bậc scale theo grade Tiên Khiếu (LOW→EXTREME) và đượt phá tăng thêm điểm |

### 6.2 Data model

- `data/ResourcePointTemplate.csv`: `point_type, name, path, produce_key, produce_interval_hours, base_yield, vitality_drain, vitality_regen_hours, rare_key, rare_grade_mid, rare_grade_top, is_food, worker_race`.
  - 5 archetype canon: `HERB_VALLEY` (Thung Lũng Linh Thảo — Wood), `BEAST_PASTURE` (Đồng Cỏ Thú — Food, **is_food**), `ORE_VEIN` (Mạch Quặng — Earth), `SPIRIT_SPRING` (Ao Linh Tuyền — Water), `GOLD_SAND_SHOAL` (Bãi Cát Vàng — Metal).
- Bảng `aperture_resource_points`: `id, aperture_id, point_type, name, path, grade (LOW/MID/TOP), level (1–5), vitality INT (0–100), stock JSONB {item: int}, stock_cap INT, x, y (điểm phóng chiếu trên lưới 20×20 cho arena kiếp), last_produced_at, state ACTIVE/EXHAUSTED, created_at, updated_at` + index `(aperture_id, state)`.
- Bảng `aperture_point_workers`: bản sao `aperture_node_workers` cho điểm (`point_id, variant_human_id, assigned_population`).
- Migration: `DELETE FROM aperture_resource_nodes WHERE node_category='RESOURCE'` (bỏ phần cell lương) + bỏ boot-rewrite 10h; `aperture_resource_nodes` còn DDL chỉ để DEFENSE.

### 6.3 Luồng gameplay mới

- **Sinh điểm:** lúc khai mở Tiên Khiếu + **mỗi lần đột phá độ kiếp thêm 1 điểm** (canon Blessed Land Consolidation); trọng số loại/phai điểm sinh theo `dao_marks` (phai chính 70%/phái phụ 30%, già resemble distributeDaoMarks).
- **Tích lũy lười:** mỗi đọc/ghi điểm tính `elapsed × time_rate` → credited vào `stock` (≤ `stock_cap` = base × levelM × 3 interval ≈ 3 nhịp); tràn thì mất (canon: phải về thu hoạch). Hetick cho click-nhiều-node; vẫn them giữ "về nhà thu hoạch" vì stock fills.
- **Thu hoạch:** theo điểm (không theo ô); collect-all gom mọi stock sẵn có — **không còn trừ 20%** vì phát bù waste khi tràn đã là anti-idle.
- **Độ sinh khí:** mỗi thu hoạch trừ `vitality_drain` (~10, nhân theo cường độ lao động EXPLOIT trừ gấp đôi); hồi `2/môi internal giờ`; vitality ≤ 30 → yield ×0.5; = 0 → trạng thái EXHAUSTED (yield 0) cần Bồi Dưỡng.
- **Dị nhân:`aperture_point_workers`: +0.25 yield × số công nhân (cap 2 slot × dân 500), tăng hồi sinh khí; đủ nghĩa phân công dị nhân (giải quyết chỉ trích P0-5).**
- **Arena kiếp nạn:** loader realtime chọn **points** thay node: mỗi điểm 1 entity n: (vitality = HP), vị trí theo (x,y) đã seed; snapshot persist chuyển về vitality; thua = 30% điểm vitality→0 (thay DELETE node).
- **Luyện queue/society:** ăn từ tồn kho thức ăn trồng các điểm `is_food` (BREED/PASTURE) — sửa tick yearly + society_simulation bỏ đốt immortal_essence; overflow không cần quyết định nữa.
- **Worker refinement unwired bug:** wire `processApertureRefinement` 10 phút/tick (P0-1).

### 6.4 Phạm vi cắt (files)

B: `schema_aperture.go` (DDL + migrate), `models/aperture.go` (+types), `gamefx/aperture_yield.go` (+vitality/stock), NEW `handlers/aperture_points.go`, `handlers/aperture_collect.go` (cut & glue), `handlers/aperture_nodes.go` (defense-only), `handlers/aperture_core.go` (GET nodes → defense; new GET points; seed gen), `handlers/aperture_settings.go` (defense-only), `handlers/tribulation_trigger.go` (yearly sim trên điểm), `handlers/realtime_tribulation.go` + `handlers/tribulation_realtime.go` (arena swap), `worker/variant_refinement.go` wire, `server.go` routes, admin_world_extended (giữ list DEFENSE, điểm admin list thêm).
F: `apertureSlice.ts` + `ImmortalApertureView.tsx` (tab Điểm Tài Nguyên thay tab khu trội; giờ vẫn giữ grid phòng thu); `docs/GAME_DATA.md` CSV rule; `WIKI.md` § Tiên đạo rewrite; `API.md` mới route.
