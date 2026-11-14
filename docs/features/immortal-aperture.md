# Tiên Khiếu

## Tính năng làm gì?

Tiên Khiếu là không gian riêng của người chơi tại Rank 6+, có node/resource point, worker, NPC, swarm, portal và hệ thống tribulation.

## Điều kiện mở và quyền truy cập

Player cần đạt Rank 6 và anchor khi không đang trong aperture khác. Chủ sở hữu đặt access policy; khách vào theo quyền OPEN, PARTY, SECT, KEY, APPROVAL hoặc HIDDEN.

## Cách sử dụng

Anchor aperture, quản lý node và worker, phân quyền khách, rồi mở portal để di chuyển. Tribulation được khởi động và giải quyết theo lượt; khách có thể tham gia nếu luật session cho phép.

## Luật, chi phí và giới hạn

- Rank 6+ mới dùng aperture.
- Không thể anchor bên trong aperture khác.
- Portal entry kiểm tra world, combat và policy; chỉ có gateway phù hợp mới cho vào.
- Resource production, worker upkeep, capacity và tribulation dùng state/config server-side.

## Kết quả và trường hợp thất bại

Thiếu rank, vị trí không hợp lệ, đang combat hoặc không đủ quyền sẽ chặn anchor/entry. Tribulation có thể kết thúc bằng kết quả thành công hoặc thất bại và tác động realm/player theo handler.

## Liên hệ với hệ thống khác

Rank 6 mở Tiên Khiếu và currency/economy nhánh tiên. Portal gắn với world movement; resource points và workers chạy qua background workers; combat/tribulation vẫn turn-based.

## Dữ liệu/cấu hình ảnh hưởng

Node, resource, worker, NPC, swarm và tribulation parameters nằm trong schema/config/game data tương ứng.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/aperture_core.go`, `aperture_portal.go`, `legacy_engage_tribulation.go`, `legacy_resolve_tribulation.go`.
- Frontend: `frontend/src/store/slices/apertureSlice.ts` và aperture UI components.

## API liên quan

Nhóm aperture/portal/tribulation routes; tra [API guide](../guides/API.md) và router `backend/internal/server/server.go`.

## Ví dụ flow

```text
đạt Rank 6 → anchor aperture → đặt quyền PARTY → tạo portal
→ thành viên đủ quyền entry → quản lý worker/node → tham gia tribulation theo lượt
```
