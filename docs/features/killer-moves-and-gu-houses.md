# Sát Chiêu và Tiên Cổ Ốc

## Tính năng làm gì?

Hai tính năng cho phép đóng gói Gu và effect thành sản phẩm có profile riêng:

- **Sát Chiêu:** cấu hình một Gu chính, Gu phụ và các effect được phép composition; sản phẩm được dùng trong combat theo lượt.
- **Tiên Cổ Ốc (Gu House):** chọn core Gu và cấu hình hiệu ứng/khả năng của nhà; có thể trang bị, dùng một số `ON_USE` effect trong world và repair.

Không phải mọi Gu effect đều được phép nối với nhau hoặc dùng trong mọi context.

Effect mới trong `backend/data/Gu.csv`: Xuân Tiễn Tiên Cổ rank 6 có `STATUS SLOW` (20%, 2 lượt) trên mục tiêu; Liệu Quang Giáp Tiên Cổ rank 6 có `CLEANSE` trên bản thân, thay cho trạng thái `BLIND` tự gây trước đây. Tên Xuân Tiễn lấy cảm hứng từ danh sách sát chiêu tiên đạo; trị số và cơ chế là thiết kế của game, không được danh sách đó mô tả. Cả hai là lựa chọn khi tạo Sát Chiêu từ Cổ tương ứng, tuân theo preview và allowlist combat; chúng không phải công thức Sát Chiêu cố định.

## Điều kiện mở và quyền truy cập

- Routes `/killer-moves/*` và `/gu-houses/*` yêu cầu đăng nhập và onboarding.
- Component Gu phải thuộc player, đang sống, hợp lệ cho product profile/context và không bị product khác giữ.
- Sát Chiêu cần preview hợp lệ; create yêu cầu đúng fingerprint của preview gần nhất. Nếu preview tự scale effect, player phải chấp nhận điều chỉnh trước khi lưu.
- Giới hạn số product đã lưu/equipped, số Gu component và số effect do product profile trả về; xem `/killer-moves/options` thay vì hard-code UI cap.
- Một Gu không thể đồng thời nằm trong active Sát Chiêu, active Gu House hoặc teleport array reservation xung đột.

## Cách sử dụng

### Tạo Sát Chiêu

1. Lấy `/killer-moves/options` để biết Gu instances, effect options, composition templates và rule revision hiện tại.
2. Chọn Gu chính, component Gu và effect output/support được catalog cho phép.
3. Gọi `/killer-moves/preview`; xem power budget/load, scale, harmony, complexity, success chance, mana cost, target profile và warnings.
4. Lưu tên cùng draft và `preview_fingerprint`; chấp nhận điều chỉnh nếu effect bị scale.
5. Trang bị bằng `/:id/equip`. Server tạo reservation cho Gu thành phần; hủy equip hoặc xóa theo route để giải phóng.

### Dùng hoặc repair Sát Chiêu

Sử dụng khi combat/profile cho phép; action combat đi qua combat session plan và simulator. Route use là compatibility/gameplay entry theo handler; UI/server sẽ xác nhận context. Repair khi durability/product state cho phép.

### Tạo và dùng Tiên Cổ Ốc

1. Xem Gu core/effect options trong màn Gu House.
2. Gọi `/gu-houses/preview`, xem warnings, profile và tài nguyên cần thiết.
3. Tạo rồi equip. Active house giữ core Gu bằng owner-typed reservation.
4. Dùng action/ON_USE effect có trong profile; server kiểm tra house đang equipped và core vẫn reserved cho house đó.
5. Unequip hoặc repair qua route tương ứng. Không sửa/xóa core bằng inventory trong lúc còn active reservation.

## Luật, chi phí và giới hạn

### Sát Chiêu: ngân sách và xác suất

Profile hiện tại tính:

```text
powerBudget = 20 + 26*mainGuRank + Σ(13 / sqrt(supportIndex))
complexity = 1 + .14*(componentCount-1) + .12*(effectCount-1)
             + .10*max(0, pathCount-1)
harmony = 1 (một component), ngược lại .6 + .4*(matchingPathPairs/allPairs)
```

Nếu load vượt budget, scalable load nhận scale `(budget-unscalableLoad)/scalableLoad`; cleanse load là unscalable và scale dưới minimum profile hiện .15 bị reject. Xác suất tạo thành công trước clamp là `.55 + .035*pathMasteryLevel + .18*(harmony-.6) + min(.15, masteryXP/1000*.15) - .15*(complexity-1)`, clamp 10%–95%. Profile giới hạn 6 Gu, 8 effect nodes, 12 product đã lưu và 4 product equipped.

Khi dùng ngoài combat, chance được điều chỉnh theo chênh lệch mastery hiện tại/lúc tạo và XP, sau đó clamp 5%–95%. Mana theo component cost; lần dùng thất bại vẫn trừ mana và mỗi component mất 1 hunger. Preview/response là số cuối cùng vì rule profile có revision.

### Tiên Cổ Ốc: profile và repair

Profile combat hiện tại tính `HP = highestCoreRank*500 + coreCount*250 + auxCount*35`, `defense = highestCoreRank*40 + coreCount*25 + auxCount*4`, damage reduction `min(.35, .10 + .05*coreCount)`. Repair tốn `ceil((100-durability)*highestCoreRank*10)` Primeval Stones và phục hồi durability lên 100. Đây là profile của nhà, không phải stat cộng vĩnh viễn vào player.

- `composition_contributions` phải grant capability rõ tới effect block/operation. Primary role, combat role hoặc tag không tự tạo quyền composition.
- Product rule set version/revision pin vào composition draft/active artifact. Preview cũ có thể hết hiệu lực nếu source Gu/rule thay đổi; tạo preview mới trước khi commit.
- Effect context/executor phải nằm trong adapter allowlist của sản phẩm. Một operation hợp lệ về schema vẫn có thể bị từ chối nếu không hỗ trợ trong turn-based combat/world use.
- Active Sát Chiêu/Gu House materialize snapshot effect; thay đổi Gu CSV không tự rewrite product đang active. Ownership, availability, hunger/reservation vẫn là lifecycle checks live.
- Component Gu reserved không dùng làm ingredient, trade/drop hay loot; worker/lifecycle có thể tháo product khi core chết do hunger và giải reservation.
- Số slot, mana, success/backfire/durability costs phải đọc preview/response và product profile.

## Kết quả và trường hợp thất bại

- Preview validation failure không tạo product.
- Fingerprint mismatch trả conflict; tải options/preview mới, xem lại thay đổi rồi commit lại.
- Component đang locked, dead, không owned, không active hoặc thiếu provenance/capability khiến preview/create/equip fail.
- Khi component mất trạng thái hợp lệ (ví dụ chết do hunger), product liên quan có thể bị auto-unequip/deactivate và reservation được giải phóng theo lifecycle code.
- Một lần dùng action có thể tiêu hao resource hoặc tạo backfire/durability consequences theo profile; xem response combat/world result.

## Liên hệ với hệ thống khác

- Cùng một Gu instance không thể chiếm dụng đồng thời bởi các active products.
- Combat loadout/session dùng Sát Chiêu như action theo lượt; không có combat simulation trên WebTransport.
- Gu House world effects dùng world `ON_USE` executor và có rule riêng, không tự áp dụng như passive stat toàn game.
- Effect definitions/catalog nằm trong [GAME_DATA.md](../guides/GAME_DATA.md); gameplay syntax xem [combat.md](combat.md).

## Dữ liệu/cấu hình ảnh hưởng

- `Gu.csv` Block v2 effects và `composition_contributions` xác định effect/source grants.
- Composition rule catalogs/revisions trong `backend/internal/gameplay/composition/` quy định edge/capability.
- Product profiles `killer_move.v1`/`gu_house.v1` quản lý component count, executor, context, target và saved/equipped limits.
- Active artifacts lưu composition graph/provenance/effect snapshot; reservations lưu product owner và Gu IDs.

## Mã nguồn liên quan

- Frontend: `frontend/src/components/KillerMovesPanel.tsx`, `GuHousePanel.tsx`; `frontend/src/store/slices/killerMovesSlice.ts`, `guHouseSlice.ts`.
- Backend: `backend/internal/handlers/killer_moves.go`, `gu_house.go`, `gu_house_use.go`, `gu_reservation.go`.
- Rules/data: `backend/internal/gameplay/composition/`, `backend/internal/effectcatalog/`, `backend/internal/db/schema_player_items.go`.

## API liên quan

- Sát Chiêu: `GET /killer-moves/options`, `GET /killer-moves`, `POST /killer-moves/preview`, `POST /killer-moves`, `POST /killer-moves/:id/equip`, `POST /killer-moves/:id/use`, `POST /killer-moves/:id/repair`, `DELETE /killer-moves/:id`.
- Tiên Cổ Ốc: `GET /gu-houses`, `POST /gu-houses/preview`, `POST /gu-houses`, `POST /gu-houses/:id/equip`, `POST /gu-houses/:id/use`, `POST /gu-houses/:id/repair`.

## Ví dụ flow

```text
GET options → draft component/effects → POST preview
→ inspect warnings/scale/fingerprint → POST create with accepted fingerprint
→ POST equip → use in allowed combat/world context
→ unequip/deactivate before moving or consuming a reserved Gu
```
