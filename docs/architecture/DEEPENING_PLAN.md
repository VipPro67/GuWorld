# Kế hoạch làm sâu module (Deepening Plan)

Tài liệu này là **kế hoạch triển khai**, không phải mô tả tính năng đã phát hành. Trạng thái từng bước nằm ở mục [Nhật ký tiến độ](#nhật-ký-tiến-độ); chỉ đánh dấu `Xong` khi code, test và docs liên quan đã cập nhật trong cùng thay đổi.

Bối cảnh: rà soát kiến trúc ngày 2026-11-24 trên các vùng có churn cao nhất (`git log`): combat, aperture, exploration/movement, `useGameStore`, admin CRUD. Báo cáo đầy đủ (before/after diagram) sinh ở `/tmp/architecture-review-*.html`; tài liệu này giữ phần quyết định.

## Quy ước

Dùng đúng bộ từ của `codebase-design`: **module**, **interface**, **implementation**, **depth** (deep/shallow), **seam**, **adapter**, **leverage**, **locality**. Tránh "component", "service", "layer", "boundary".

Ràng buộc đã chốt:

1. **Behavior-preserving.** Refactor không đổi số liệu gameplay, thứ tự publish hay payload. Lỗi phát hiện dọc đường ghi thành issue riêng kèm test tái hiện, trừ khi lỗi chặn chính việc refactor.
2. **Chỗ đặt module.** Module thuần (không cần `*gin.Context` / `*sql.Tx`) tách package theo tiền lệ `simulator`, `playerstate`, `admincatalog`, `statusfx`. Module cần transaction hoặc HTTP nằm trong `internal/handlers/`. Không thêm tầng service/repository mới.
3. **Điều kiện tồn tại của seam.** Mỗi module mới phải thay thế được nhiều chỗ trùng (deletion test). Một adapter duy nhất không đủ để biện minh cho seam.
4. **Test gate.** Vùng chưa có test thì viết test đặc tả hành vi hiện tại trước khi di chuyển code; test đi qua interface của module, không test qua implementation.
5. **Hợp đồng docs.** Thay đổi chạm route/hành vi player-visible cập nhật `docs/guides/API.md` / `FEATURES.md` / `WIKI.md`; chạm player state cập nhật `docs/architecture/PLAYER_DOMAIN.md`; thuật ngữ mới thêm vào `CONTEXT.md` **sau khi** code tồn tại.

## Thứ tự thực thi

| # | Bước | Mức | Trạng thái |
|---|------|-----|------------|
| 6 | Xoá bản sao xử lý 401 ở store | Strong | Xong |
| 1 | Hoàn tất Chuỗi Đột Biến Người Chơi | Strong | Seam xong, chờ chuyển call site |
| 5 | Gom mana / sức chứa / quyền sở hữu | Worth exploring | Một phần (sect membership; mana/capacity đã tập trung sẵn) |
| 2 | Đưa walk + Hạ Khiếu vào Chuỗi Di Chuyển | Strong | Một phần (guard chết đã xoá) |
| 3 | Tách CombatSettlement | Strong | Một phần (một cửa + arena adapter, chưa tách pha) |
| 4 | Combat log thành sự kiện có kiểu | Worth exploring | Xong |
| 7 | Gom CRUD Admin thành module bảng | Worth exploring | Chờ |
| 8 | Route theo domain | Speculative | Chờ |

Lý do thứ tự: 6 là quét cơ học không rủi ro thiết kế. 1 rồi 5 vì cùng seam trạng thái player (`MutatePlayer` đã bao gồm bước nạp mana, nên 5 là phần còn lại của cùng một cửa). 2 độc lập. 3 là chỗ rủi ro nhất nhưng biên rõ, và hưởng lợi từ 1. 4 chạm cả backend lẫn frontend. 7 và 8 là dọn cấu trúc, không chặn ai.

---

## Bước 6 — Xoá bản sao xử lý 401

**Files**: `frontend/src/lib/auth.ts`, `frontend/src/store/slices/*.ts` (23 file, 194 khối).

**Problem**: `authFetch` đã gọi `unauthorizedHandler` (đăng ký một lần ở `useGameStore`) khi gặp 401, nhưng 23 slice vẫn giữ nhánh `status === 401` tự `logout()`/throw. Chính sách session bị xé thành 194 bản.

**Interface đích**: một chính sách duy nhất ở `authFetch`; slice không kiểm tra 401.

**Pha**:
1. Phân loại 194 khối: (A) throw để dừng hàm, (B) `return` / `return null`, (C) nhánh khác.
2. Với (A): bảo toàn ngữ nghĩa "dừng trước khi đọc body" bằng một guard dùng chung, không bằng `logout()` rải rác.
3. Với (B): xoá trực tiếp.
4. Xoá `logout` khỏi destructuring nếu không còn dùng trong file.

**Test**: `npm run typecheck`, `npm run lint`, `npm test`. Thêm test cho guard dùng chung.

**Hoàn thành khi**: `grep -rn 'status === 401' frontend/src/store/slices | wc -l` chỉ còn các nhánh hoàn tất trạng thái có chủ đích (đã ghi chú tại chỗ) và không slice nào gọi `logout()` chỉ để xử lý 401.

**Kết quả**: 189/194 khối đã xoá ở 23 slice. 5 nhánh còn lại không phải bản sao logout mà là nhánh hoàn tất trạng thái, giữ lại kèm ghi chú: `cultivationSlice` (recipes/quests/regions đặt cờ loaded để UI không retry sau logout) và `profileSlice` (reset session đang dở, logout có guard revision). Không slice nào còn gọi `logout()` cho 401.

Lưu ý ngữ nghĩa: sau khi xoá, một response 401 vẫn đi tiếp vào nhánh đọc body/`!response.ok`. Rà tiếp cho thấy 48 chỗ là fetch-danh-sách không có guard `!response.ok` — body lỗi sẽ được ghi thẳng vào state (ví dụ `marketplaceListings`, `sectMembers`) trước khi logout reset. Đã thêm guard `if (!response.ok) return;` đúng chỗ; 28 guard bị đặt thừa bên trong nhánh `if (response.ok)` đã bỏ. Còn 20 guard mới, tất cả trước bước `response.json()`.

---

## Bước 1 — Hoàn tất Chuỗi Đột Biến Người Chơi

**Files**: `backend/internal/handlers/player_mutation.go`, `backend/internal/playerstate/snapshot.go`, 84 handler file ghi thẳng `players`.

**Problem**: `CONTEXT.md` định nghĩa Chuỗi Đột Biến Người Chơi (tx → `FOR UPDATE` → snapshot → thụ động → closure → ghi đủ cột + `version`/`state_revision` → commit → publish). `MutatePlayer` làm đúng nhưng **tự mở transaction** và chỉ nhận player đã xác thực, nên chỉ 14 call site dùng được; 229 câu `UPDATE players` viết tay (31 chỗ tự tăng `version`) nằm rải ở handlers, còn `playerstate.Load` chỉ có 9 caller so với 293 câu `FROM players`.

**Interface đích**:
- `MutatePlayer(tx, playerID, opts, fn)` — nhận transaction của caller (nil thì tự mở, để adapter cũ giữ nguyên chữ ký hoặc thêm biến thể).
- Một đường publish sau commit cho **nhiều** player trong cùng transaction (PvP, trade, party), thay vì publish lẻ từng handler.
- `playerstate.Load` là cửa đọc full-row; projection hẹp vẫn hợp lệ khi chỉ đọc vài cột.

**Pha**:
1. Thêm biến thể nhận `*sql.Tx` và biến thể nhiều player; test cho version/revision và publish.
2. Chuyển các call site theo cụm, mỗi cụm một commit nhỏ: cultivate/equip (đã dùng), rồi combat, aperture, sect, trade.
3. Xoá dần `UPDATE players` viết tay; giữ lại khi thật sự là projection hẹp có chủ đích (ghi rõ comment).

**Đã làm (pha 1)**: `player_mutation.go` tách thành `MutatePlayer` (tự mở transaction, publish) / `MutatePlayerInTx` (transaction của caller, không commit, không publish) / `PublishPlayerMutation` (gọi sau commit, nhiều player được). Test: `player_mutation_in_tx_test.go`.

**Đã làm (pha 2, cụm respawn)**: `GrantPvpImmunity` và `GrantPvpInitiatorCooldown` chuyển sang `MutatePlayerInTx` với `statusMutationOptions()` (`Scope: GuEquipped`, bỏ đánh giá mana/độ bền thụ động, không publish). Hai đường này giờ tăng `version`/`state_revision` như các mutation khác. `RemovePvpImmunity` giữ nguyên UPDATE có điều kiện vì nó chỉ ghi khi trạng thái tồn tại; ghi chú tại chỗ. Số câu `UPDATE players` trong handlers: 229 → 227.

**Việc tiếp theo (pha 2)**:
- Rà `EvaluateGuDurabilityHealing` khi gọi trong transaction lớn: hiện `MutatePlayer` chạy bước này trước khi khoá dòng.
- Với luồng hai người chơi (PvP `settlePvpDefender`, trade): gọi `MutatePlayerInTx` hai lần trong cùng tx rồi publish cả hai sau commit; kiểm tra thứ tự khoá để tránh deadlock.

**Hoàn thành khi**: không còn `version = version + 1` viết tay trong handlers; mọi mutation nhiều bảng nằm trong một transaction có publish sau commit.

**Rủi ro**: đổi ownership transaction dễ gây deadlock nếu thứ tự khoá khác hiện tại. Kiểm tra `FOR UPDATE` hiện có trước khi gộp.

---

## Bước 5 — Gom mana / sức chứa / quyền sở hữu

**Files**: `backend/internal/handlers/mana.go`, `backend/internal/db/player_gu.go`, `ascension.go`, `gu_*.go`, `killer_moves.go`, `refine_*.go`, `scouting.go`, `world_effects.go`.

**Problem**: 3 biến thể `evaluateManaTx*`, 17 guard `CurrentMana < …` viết tay, 2 helper sức chứa dùng lệch nhau ở 7 chỗ, 148 predicate `player_id = $1 AND …`.

**Interface đích**: `SpendMana(tx, player, cost)`, `CheckCapacity(tx, playerID, delta)`, `OwnsRow(tx, playerID, table, id)` — một nhà cho luật "được phép tiêu / được phép giữ".

**Pha**: gộp 3 biến thể mana thành một (cờ readonly); thay 17 guard inline; thay predicate sở hữu theo cụm file.

**Hoàn thành khi**: `CurrentMana <` còn 0 ngoài module; capacity chỉ gọi một helper.

**Rà soát trước khi sửa (kết luận)**: phần lớn "trùng lặp" trong mục này đã được tập trung sẵn, nên các seam dự kiến không qua được deletion test:

- **Mana**: `evaluateManaTx*` chỉ là ba tên adapter 1 dòng gọi `PlayerSvc.EvaluateAndPersistMana*`; toán học nằm ở service, khác nhau ở scope (`GuFull`/`GuEquipped`) và readonly. Gộp thành một hàm có cờ sẽ thêm tham số ở ~130 call site mà không xoá luật nào. 17 guard `CurrentMana <` có thông điệp/HTTP code riêng cho từng hành động; một predicate dùng chung sẽ đổi câu chữ người chơi thấy.
- **Sức chứa**: `db.MortalApertureCapacity` + `db.CheckApertureCapacity` đã là nhà duy nhất cho luật "được chứa thêm Cổ trùng". Các chỗ đếm equipped (`player_use.go`, `aperture_swarms.go`) đọc nguồn khác nhau (inventory trong bộ nhớ vs DB) nên không phải cùng một luật.
- **Quyền sở hữu Cổ trùng**: `db.LoadPlayerGuForUpdate`, `db.RemovePlayerGuForPlayer`, `db.TransferGuOwnership` đã tồn tại.

**Đã làm (phần còn trùng thật)**: 30 câu đọc `FROM sect_members WHERE player_id = $1` rải ở 16 file gom vào `internal/handlers/sect_membership.go` (`SectMembership`, `LoadSectMembership`, `LoadSectMembershipContext`, `LoadSectID`). Test: `sect_membership_test.go` (có/không có dòng, rank + defending, bản context). Các câu còn lại chỉ là `DELETE`/`INSERT` (ghi, thuộc handler sở hữu) và một câu join `sects.level` giữ nguyên.

**Hoàn thành khi (điều chỉnh)**: `FROM sect_members WHERE player_id = $1` chỉ còn trong module; mana/capacity/ownership không tạo seam mới vì đã có nhà.

---

## Bước 2 — Đưa walk và Hạ Khiếu vào Chuỗi Di Chuyển

**Files**: `backend/internal/handlers/movement.go`, `exploration_move.go`, `aperture_core.go`, `portal_transfer.go`, `teleport_handlers.go`, `movement_layers.go`, `player_gu_use.go`, `travel_journey.go`.

**Problem**: `ExecuteSpatialTransition` là module sâu đúng hình nhưng chỉ 4 caller. Đường đi bộ `ExecuteMovePlayerPath` (1.350 dòng, trả `(int, gin.H)`) tự ghi toạ độ, tự gọi `ResolveCombat`; `AnchorAperture` ghi `players` + `world_location_revision` bằng tay. Hai guard chết: `validateMovementState(..., _ bool)` và `rejectRealtimeOwned` luôn `false`.

**Interface đích**: một chuỗi cho walk/portal/teleport/Gu teleport/luồng quay về realm/Hạ Khiếu; HTTP adapter render `movementErrorPayload`, module không trả status.

**Pha**: chuyển walk sang `movementSpec` + chuỗi (giữ nguyên thứ tự guard hiện tại); chuyển `AnchorAperture`; quyết định xoá hay nối 2 guard chết (nối thì cần middleware realtime, tách bước).

**Hoàn thành khi**: chỉ còn một nơi ghi `players.x/y` + `world_location_revision` cho mọi đường di chuyển; `ExecuteMovePlayerPath` không còn SQL vị trí.

**Đã làm (guard chết)**: xoá `rejectRealtimeOwned` (luôn `false`) cùng hai khối gọi ở `PortalTransfer`/`SkyfallDescent`, bỏ tham số `_ bool` khỏi `validateMovementState`, bỏ phép tính `ownedByRealtime` trong `checkMovement` và trường `movementSpec.realtimeAuthorized`. Chuỗi giờ chỉ có một bộ guard thật; `middleware.IsRealtimeWorldOwned` vẫn là stub `false` và còn được các route party/scouting/teleport gọi — dọn cả cụm đó là việc riêng, không thuộc bước này.

**Chưa làm (cần quyết định hành vi)**: `savePlayerMovement` (đường đi bộ) không ghi `world_location_revision`, còn `ExecuteSpatialTransition` thì có. Gộp hai chỗ ghi sẽ làm walk tăng revision — đổi hành vi đồng bộ, nên chưa gộp khi chưa có quyết định. `AnchorAperture` gọi thẳng chuỗi sẽ chạy thêm guard (combat/HP/vùng), cũng là đổi hành vi; cần test đặc tả trước.

---

## Bước 3 — Tách CombatSettlement

**Files**: `backend/internal/handlers/combat_auto_resolve.go`, `combat_lifecycle.go`, `combat_outcome.go`, `combat_loot.go`, `combat_result_defeat.go`, `combat_history.go`.

**Problem**: `resolveCombatWithTerminalState` 480 dòng trộn policy vùng, mô phỏng, SQL xếp hạng arena, loot, tử vong, lịch sử; 4 cửa vào (`ResolveCombat`, `ResolveCombatFromSession`, `resolveCombat`, `SettleCombatSession`) đều là wrapper rỗng; file không có test.

**Interface đích**: `Settle(tx, input) (*Resolution, error)`; các pha đặt tên (policy, simulate, ranking, loot, defeat, history); xếp hạng arena ra sau adapter.

**Pha**: viết test đặc tả qua interface mới cho các nhánh hiện có; tách pha; xoá wrapper; chuyển SQL arena thành adapter.

**Hoàn thành khi**: `combat_auto_resolve.go` không còn SQL arena; 4 wrapper còn 1 cửa; có test cho thắng/thua/PvP/rút lui.

**Đã làm**: `ResolveCombat`, `ResolveCombatFromSession`, `SettleCombatSession` cùng gọi một lõi `settleCombat` (bỏ wrapper `resolveCombat` rỗng); pha xếp hạng arena tách ra `combat_arena_ranking.go` (`settleArenaRankSwap`), nên `combat_auto_resolve.go` không còn câu SQL `arena_rankings` nào. Test: `combat_arena_ranking_test.go` (hoán hạng, giữ hạng khi tốt hơn/ngang, thiếu transaction, thiếu active combat) dùng fake driver; các nhánh thắng/thua/PvP/rút lui đã có test pha sẵn (`combat_outcome_test.go`, `combat_mode_policy_test.go`, `combat_session_test.go`).

**Chưa làm**: `settleCombat` vẫn là một hàm dài với state chung (`pendingNotifications`, `immortalAnnouncements`, `publishClearedCell`); tách thành các pha có tên cần một struct input/output trước, nên để bước sau. Chưa có test end-to-end qua DB vì backend không có DB test harness.

---

## Bước 4 — Combat log thành sự kiện có kiểu

**Files**: `frontend/src/components/CombatScreen.tsx`, `CombatReplayView.tsx`, `CombatReportView.tsx`, `backend/internal/handlers/combat_sim_build.go`, `backend/internal/simulator/*`.

**Problem**: 3 view tự suy nghĩa log bằng `log.action === …`; backend suy hình phạt vĩnh viễn bằng `strings.Contains(Detail, "Trừ 10% Tư chất…")` — chuỗi hiển thị tiếng Việt thành protocol.

**Interface đích**: simulator trả hiệu ứng settlement có kiểu; một module `combatLog` (frontend `lib/combatLog.ts`) map log → event union; view render event.

**Pha**: thêm `SettlementEffect` có kiểu cho Chaos Gamble; bỏ parse chuỗi; frontend module + test map.

**Hoàn thành khi**: không còn `strings.Contains(log.Detail` trong handlers; ba view không còn `log.action ===` ngoài module.

**Đã làm (backend)**: `CombatLog` mang `Settlement *SettlementEffect` (`Code`/`Amount`, `json:"-"`). `CHAOS_GAMBLE` ghi cái giá đã chọn dưới dạng có kiểu; bước gộp log chuyển settlement sang entry hành động còn lại (passive được log trước entry "dùng [X]" nên phải gom rồi gắn sau). `applyLoveGuPenalties` đọc `Code`/`Amount`, không còn `strings.Contains(Detail, …)`. Test: `simulator/chaos_gamble_settlement_test.go`, `handlers/combat_settlement_effect_test.go`.

**Lỗi phát hiện dọc đường (đã sửa vì chặn chính refactor)**: trước thay đổi này, `applyLoveGuPenalties` chỉ khớp log `Action == "CHAOS_GAMBLE"`, nhưng bước gộp log đã xoá entry đó và chỉ giữ entry hành động — nên cái giá của Nhiệm Ý Cổ **chưa bao giờ được áp** trên đường `simulator.Simulate`. `GAME_DATA.md` vốn mô tả cái giá được áp sau trận; code giờ khớp tài liệu. Đây là thay đổi gameplay có chủ đích (hình phạt thực sự diễn ra) và cần ghi vào thông báo phát hành.

**Còn lại (frontend)**: module `frontend/src/lib/combatLog.ts` map log → event union, ba view dùng module, test map.

**Đã làm (frontend)**: `frontend/src/lib/combatLog.ts` chuẩn hóa `action` thành `CombatLogKind`, gom nhóm ngữ nghĩa (`isHealKind`, `addsShieldKind`, `reducesTargetHp`, `showsFloatingDamage`, `showsInlineDamage`, `isRewardKind`, `isSkillCastKind`, `isLossLog`) và tách `turnNumberFromLog`, `skillNameFromDetail`, `autoUsageCount`, `fallbackActionLabel`. `CombatReplayView`, `CombatReportView`, `InteractiveCombatPanel` không còn so sánh `log.action ===`; reducer và phần tổng kết render theo kind. Test: `frontend/src/lib/combatLog.test.ts` (11 test, thêm vào `npm test`).

**Hoàn thành khi**: không còn `strings.Contains(log.Detail` trong handlers; ba view không còn `log.action ===` ngoài module → đạt (`grep` trong `src/components` trả 0).

---

## Bước 7 — Gom CRUD Admin thành module bảng

**Files**: `backend/internal/handlers/admin*.go` (56 file, ~16.100 dòng), `backend/internal/admincatalog/catalog.go`, `frontend/src/components/AdminView.tsx` (5.803 dòng).

**Problem**: cùng khuôn List/Create/Update/Delete lặp cho ~20 bảng; quyền đã có trong `admincatalog` nhưng CRUD không dùng mô tả đó.

**Interface đích**: một mô tả bảng (cột, validate, quyền, audit, reload registry) phục vụ cả 4 thao tác; handler thành adapter. Frontend: mỗi section admin là module sau `ADMIN_NAVIGATION`.

**Pha**: chọn 3 bảng đơn giản nhất làm mẫu; thêm mô tả vào `admincatalog`; xoá handler trùng; sau đó mở rộng.

**Hoàn thành khi**: thêm bảng mới không cần 4 handler mới.

---

## Bước 8 — Route theo domain

**Files**: `backend/internal/server/server.go` (1.405 dòng, 389 route), `server_test.go`.

**Problem**: mọi route, middleware và rate-limit nằm một file; không có locality theo domain.

**Interface đích**: mỗi domain `RegisterRoutes(r *gin.RouterGroup)`; `server.go` chỉ compose. Giữ nguyên thứ tự middleware để không đổi hành vi.

**Hoàn thành khi**: route combat/aperture/world/sect/trade/admin/auth nằm cạnh domain; test route inventory đọc theo nhóm.

---

## Nhật ký tiến độ

| Ngày | Bước | Việc đã làm | Kiểm chứng |
|------|------|-------------|------------|
| 2026-11-24 | — | Rà soát kiến trúc, chốt 8 bước và thứ tự | Báo cáo HTML + tài liệu này |
| 2026-11-24 | 6 | Xoá 189/194 nhánh `status === 401` ở 23 slice; dọn `logout` khỏi destructure không dùng; ghi chú 5 nhánh giữ lại | `tsc --noEmit`, `eslint`, `npm test` (95/95) |
| 2026-11-24 | 6 | Thêm 20 guard `!response.ok` trước `response.json()` ở các fetch-danh-sách từng dựa vào nhánh 401 để dừng; bỏ 28 guard đặt thừa trong nhánh `if (response.ok)` | `tsc --noEmit`, `eslint`, `npm test` (95/95) |
| 2026-11-24 | 1 | Tách `MutatePlayer` / `MutatePlayerInTx` / `PublishPlayerMutation`; test seam; cập nhật `CONTEXT.md`, `PLAYER_DOMAIN.md` | `go build ./...`, `go vet`, `go test ./...` |
| 2026-11-24 | 1 | Cụm respawn: `GrantPvpImmunity`, `GrantPvpInitiatorCooldown` qua `MutatePlayerInTx` | `go build ./...`, `go test ./...` |
| 2026-11-24 | 4 | Backend: `SettlementEffect` có kiểu, gộp log giữ settlement, bỏ parse `Detail`; thêm test; cập nhật `CONTEXT.md`, `GAME_DATA.md` | `go build ./...`, `go test ./...` (golden hash không đổi) |
| 2026-11-24 | 2 | Xoá `rejectRealtimeOwned` + hai khối gọi, bỏ tham số chết của `validateMovementState`, bỏ `movementSpec.realtimeAuthorized` | `go build ./...`, `go vet`, `go test ./internal/handlers/` |
| 2026-11-25 | 3 | `settleCombat` là lõi duy nhất sau 3 cửa vào; tách `settleArenaRankSwap` sang `combat_arena_ranking.go`; thêm test adapter xếp hạng + guard | `go build ./...`, `go test ./internal/handlers/ -run 'ArenaRank\|SettleCombat'` |
| 2026-11-25 | 4 | Frontend: `lib/combatLog.ts` + test; `CombatReplayView`/`CombatReportView`/`InteractiveCombatPanel` render theo kind, không còn `log.action ===`; cập nhật `CONTEXT.md` | `tsc --noEmit`, `eslint`, `npm test` (106/106) |
| 2026-11-25 | 5 | Rà soát: mana adapters/capacity/ownership đã tập trung sẵn; gom 30 câu đọc `sect_members` vào `sect_membership.go` + test; cập nhật `CONTEXT.md` | `go build ./...`, `go vet ./internal/handlers/`, `go test ./...` |
