# Tiên Khiếu

## Tính năng làm gì?

Tiên Khiếu là không gian riêng của người chơi tại Rank 6+, có node/resource point, worker, NPC, swarm, portal và hệ thống tribulation.

## Điều kiện mở và quyền truy cập

Player cần đạt Rank 6 và anchor khi không đang trong aperture khác. Chủ sở hữu đặt access policy; khách vào theo quyền OPEN, PARTY, SECT, KEY, APPROVAL hoặc HIDDEN.

## Cách sử dụng

Anchor aperture, quản lý node và worker, phân quyền khách, rồi mở portal để di chuyển. Tribulation được khởi động và giải quyết theo lượt; khách có thể tham gia nếu luật session cho phép.

Khi Hạ Khiếu, chủ được chuyển tới tab Bản đồ & Phòng thủ với toàn cảnh lưới 20×20. Khách đã vào qua Môn Hộ chỉ thấy map khám phá quanh vị trí theo tầm nhìn thường; API chỉ trả người chơi, tác nhân quái, chiến lợi phẩm và mỏ trong phạm vi này. Khách có thể thu hoạch điểm tài nguyên khi ở trong phạm vi và tự rời Tiên Khiếu qua nút Rời Tiên Khiếu mà không cần chủ phê duyệt. Khi rời, khách trở về vị trí Môn Hộ bên ngoài. Các thao tác quản trị Tiên Khiếu vẫn theo quyền chủ hiện có.

## Luật, chi phí và giới hạn

- Rank 6+ mới dùng aperture.
- Không thể anchor bên trong aperture khác.
- Portal entry kiểm tra world, combat và policy; chỉ có gateway phù hợp mới cho vào.
- Grid, người chơi, mỏ, quái và chiến lợi phẩm của Tiên Khiếu được giới hạn theo `map_instance_id`; khách chỉ nhận dữ liệu trong tầm nhìn thường và chỉ đọc mỏ khi còn trong roster, đúng instance.
- Lưới và phạm vi di chuyển của Tiên Khiếu dùng tọa độ 0–19 trên mỗi trục.
- Thu hoạch mỏ Tiên Khiếu ghi tài nguyên vào túi của người thao tác; khách cần ở trong phạm vi 6 ô. Khách có thể tự rời Tiên Khiếu bất cứ lúc nào qua `POST /api/immortal-aperture/leave`; chủ không cần phê duyệt.
- Mỏ ngoài map có nghi thức di dời vào Tiên Khiếu của người chơi qua `POST /api/immortal-aperture/world-points/:id/transplant`; nghi thức tiêu hao mana/tiên nguyên theo cấp mỏ và cần đứng tại mỏ đến khi hoàn tất. Đây là di chuyển mỏ, không phải xóa mỏ. Hiện chưa có combat để tấn công phá mỏ; tai kiếp có thể làm giảm sinh khí mỏ trong Tiên Khiếu.
- Resource production, worker upkeep, capacity và tribulation dùng state/config server-side.

## Kết quả và trường hợp thất bại

Thiếu rank, vị trí không hợp lệ, đang combat hoặc không đủ quyền sẽ chặn anchor/entry. Tribulation có thể kết thúc bằng kết quả thành công hoặc thất bại và tác động realm/player theo handler.

## Liên hệ với hệ thống khác

Rank 6 mở Tiên Khiếu và currency/economy nhánh tiên. Portal gắn với world movement; resource points và workers chạy qua background workers; combat/tribulation vẫn turn-based.

## Dữ liệu/cấu hình ảnh hưởng

Node, resource, worker, NPC, swarm và tribulation parameters nằm trong schema/config/game data tương ứng.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/aperture_core.go`, `aperture_portal.go`, `aperture_points.go`, `exploration_grid.go`, `ground_drops.go`, `tribulation_multiform.go`, `tribulation_handlers.go`, `legacy_engage_tribulation.go`, `legacy_resolve_tribulation.go`.
- Frontend: `frontend/src/store/slices/apertureSlice.ts`, `frontend/src/components/ImmortalApertureView.tsx` và `frontend/src/components/ExplorationMap.tsx`.

## Hệ thống Đa Dạng Hóa Tai Kiếp (Multi-form Tribulation)
Tai kiếp Tiên Khiếu vận hành theo 3 Archetype và vòng đời 3 giai đoạn:
1. **Giai đoạn Điềm Báo (`HARBINGER`)**: Dự báo trước loại kiếp, thời gian chuẩn bị và lưu phái giáng lâm. Cho phép Cổ Tiên củng cố trận pháp hoặc click "Nghênh Kiếp Sớm" (`POST /api/tribulations/:eventId/start`).
2. **Giai đoạn Tác Chiến (`ORDEAL_ACTIVE`)**:
   - `ENVIRONMENTAL`: Thiên tai môi trường sinh thái lan tỏa trên lưới ô, hóa giải bằng Sát Chiêu Tiên Đạo (`POST /api/tribulations/:eventId/mitigate-hazard`).
   - `BEAST_SWARM`: Đợt thú triều càn quét tài nguyên, hỗ trợ Nô Đạo thu phục khi HP quái ≤ 30% (`POST /api/tribulations/:eventId/capture-beast`).
   - `ANOMALOUS_COMBAT`: Chiến đấu boss turn-based với hiệu ứng dị biến môi trường (Lôi Đình, Bão Tuyết, Cấm Pháp) và cơ chế Đạo Ngân Xung Đột.
3. **Giai đoạn Hậu Kiếp & Thu Hoạch (`AFTERMATH`)**:
   - Rơi rớt Kiếp Tài quý hiếm (`aperture_tribulation_spoils`) trên lưới ô, thu hoạch thủ công hoặc nhờ Dị Nhân càn quét (`POST /api/tribulations/:eventId/claim-all-spoils`).
   - Dư ba ô nhiễm Đạo Ngân (`aperture_soil_contaminations`): Cổ Tiên chọn **Thanh Tẩy** gột rửa đất đai hoặc **Biến Nguy Thành Cơ** quy hoạch mở mỏ tài nguyên mới (`aperture_resource_points`).
   - Kết toán hoàn tất (`POST /api/tribulations/:eventId/finalize-aftermath`) đưa Tiên Khiếu trở về `STABLE`.

## API liên quan

Nhóm aperture/portal/tribulation routes; tra [API guide](../guides/API.md) và router `backend/internal/server/server.go`.

## Ví dụ flow

```text
đạt Rank 6 → anchor aperture → đặt quyền PARTY → tạo portal
→ điềm báo tai kiếp (HARBINGER) → nghênh chiến thiên tai/thú triều/boss (ORDEAL_ACTIVE)
→ thu hoạch kiếp tài & thanh tẩy/mở mỏ mới (AFTERMATH) → Tiên Khiếu thăng hoa ổn định (STABLE)
```
