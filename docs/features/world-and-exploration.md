# Thế giới và khám phá

## Tính năng làm gì?

Bản đồ theo vùng là nơi di chuyển, gặp quái, thu thập tài nguyên, dùng cổng và tương tác vật phẩm rơi. Di chuyển được xử lý theo từng ô; world realtime đồng bộ vị trí/presence, còn chiến đấu vẫn theo lượt.

## Điều kiện mở và quyền truy cập

Người chơi cần đăng nhập và hoàn tất onboarding để dùng world. Một số vùng, portal, tài nguyên hoặc điểm dịch chuyển có yêu cầu rank, tọa độ, trạng thái chiến đấu hay cấu hình dữ liệu.

## Cách sử dụng

Chọn đường đi trên bản đồ rồi gửi move. Backend kiểm tra từng ô trên đường, áp dụng trigger trung gian và dừng tại ô đầu tiên bắt đầu encounter. Portal/teleport dùng flow movement dùng chung để kiểm tra đích và phát sự kiện cập nhật vị trí. Drop trên mặt đất được nhặt qua tương tác tại vị trí hợp lệ.

Trong **Lobby → Cài Đặt → Chiến thuật turn-based**, bật **Tự động xử lý chiến đấu khi farm** để encounter quái phát sinh lúc di chuyển được mô phỏng và settle ngay; màn điều khiển cùng báo cáo trận không mở. Tắt tùy chọn để vào encounter theo lượt như thường lệ. Cài đặt này lưu trong `combat_settings.skip_battle` và không áp dụng cho encounter bắt đầu trực tiếp từ màn chiến đấu.

Mở **Khám Phá Dã Ngoại** từ Lobby để dùng bản đồ trong popup. **Auto farm quanh vị trí hiện tại** ưu tiên nhắm quái đang thấy trong phạm vi 2 ô; khi chưa thấy mục tiêu, nhân vật tuần tra theo vòng nhỏ bán kính 2 ô để tìm encounter. Auto farm gửi tuần tự lệnh di chuyển qua API hiện có; nhấn Dừng hoặc đóng popup để ngừng gửi request mới. Cần bật `skip_battle`; tính năng chạy trên tab đang mở và dừng khi request lỗi, không tải được bản đồ hoặc nhân vật không thể tiến thêm. Request đang chạy sẽ hoàn tất trước khi dừng hẳn.

## Luật, chi phí và giới hạn

### Xác suất ambush

Ở ô Monster/Boss kề bên đủ điều kiện trigger và combat không bị cấm, chance ambush hiện tính `clamp(50 + 2*cellRank - disguiseLevel, 5, 95)%`. Đây là phần trăm, không phải xác suất 0–1. Comment trong handler ghi minimum 50 nhưng code clamp đáy ở 5; tài liệu theo behavior code. Request movement tối đa 5 ô.

- Mỗi request đi tối đa 5 ô; từng ô được kiểm tra hợp lệ.
- Trigger trên các ô trung gian vẫn chạy. Nếu combat bắt đầu, phần đường còn lại không được thực hiện.
- Vùng, portal, teleport và resource node lấy tọa độ/quyền truy cập từ map hoặc cấu hình runtime.
- Realtime chỉ hỗ trợ movement/presence và đồng bộ ngoài combat; không mô phỏng combat.

## Kết quả và trường hợp thất bại

Đường sai, đích không hợp lệ, portal bị khóa hoặc trạng thái player không cho phép sẽ bị từ chối. Encounter có thể cắt ngắn đường đi. Client nhận player/world events để đồng bộ vị trí và trạng thái.

## Liên hệ với hệ thống khác

Movement có thể khởi tạo combat, thay vùng, kích hoạt tài nguyên, hoặc đưa người chơi vào realm/aperture. Drop và inventory vẫn chịu ownership/lifecycle rules.

## Dữ liệu/cấu hình ảnh hưởng

Map/region và portal data quyết định địa hình, vùng, ô vào và điểm tương tác. CSV được import qua Admin; registry cần restart backend khi implementation cache dữ liệu đó.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/exploration_move.go`, `movement.go`, `world.go`, `portal_transfer.go`, `ground_drops.go`.
- Frontend: `frontend/src/components/ExplorationMap.tsx`, `frontend/src/store/slices/worldSlice.ts`.
- UI: `frontend/src/app/page.tsx`, `frontend/src/components/ExplorationMap.tsx`; cài đặt: `frontend/src/components/LobbySettingsPanel.tsx`, `frontend/src/store/slices/combatSlice.ts`; API `GET/PUT /api/player/combat-settings`.

## API liên quan

Các route map/movement/portal được đăng ký tại `backend/internal/server/server.go`; route cụ thể xem [API guide](../guides/API.md). Ground-drop routes cũng được khai báo tại server router.

## Ví dụ flow

```text
chọn đường 4 ô → backend kiểm tra từng ô → trigger encounter ở ô thứ 2
→ dừng tại đó → mở combat theo lượt → kết thúc combat rồi chọn đường mới
```
