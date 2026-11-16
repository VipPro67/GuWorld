# Thế giới và khám phá

## Tính năng làm gì?

Bản đồ theo vùng là nơi di chuyển, gặp quái, thu thập tài nguyên, dùng cổng và tương tác vật phẩm rơi. Di chuyển được xử lý theo từng ô; world realtime đồng bộ vị trí/presence, còn chiến đấu vẫn theo lượt.

## Điều kiện mở và quyền truy cập

Người chơi cần đăng nhập và hoàn tất onboarding để dùng world. Một số vùng, portal, tài nguyên hoặc điểm dịch chuyển có yêu cầu rank, tọa độ, trạng thái chiến đấu hay cấu hình dữ liệu.

## Cách sử dụng

Chọn đường đi trên bản đồ rồi gửi move. Chú giải lưới phân biệt điểm dịch chuyển, Nguyên Tuyền, chiến lợi phẩm, cổng Bí Cảnh và cổng Tiên Khiếu; các điểm tài nguyên khác có ký hiệu riêng. Backend kiểm tra từng ô trên đường, áp dụng trigger trung gian và dừng tại ô đầu tiên bắt đầu encounter. Chọn người chơi trên lưới để mở popup xem hồ sơ, tiến gần/khai chiến PvP (cần trong tầm đánh) hoặc giao dịch tại chỗ (cùng khu vực, tối đa 10 ô, không cần Cổ liên lạc). Portal/teleport dùng flow movement dùng chung để kiểm tra đích và phát sự kiện cập nhật vị trí. Chọn drop để mở popup xem tiền tệ, vật phẩm, Cổ, bảo hộ và hạn tồn tại; có thể tiến gần theo từng chặng tối đa 5 ô và chỉ nhặt khi đứng đúng tọa độ.

Trong **Lobby → Cài Đặt → Chiến thuật turn-based**, bật **Tự động xử lý chiến đấu khi farm** để encounter quái phát sinh lúc di chuyển được mô phỏng và settle ngay; màn điều khiển cùng báo cáo trận không mở. Tắt tùy chọn để vào encounter theo lượt như thường lệ. Cài đặt này lưu trong `combat_settings.skip_battle` và không áp dụng cho encounter bắt đầu trực tiếp từ màn chiến đấu.

Mở **Khám Phá Dã Ngoại** từ Lobby để dùng bản đồ trong popup. **Auto farm quanh vị trí hiện tại** ưu tiên nhắm quái đang thấy trong phạm vi 2 ô; khi chưa thấy mục tiêu, nhân vật tuần tra theo vòng nhỏ bán kính 2 ô để tìm encounter. Auto farm gửi tuần tự lệnh di chuyển qua API hiện có; nhấn Dừng hoặc đóng popup để ngừng gửi request mới. Cần bật `skip_battle`; tính năng chạy trên tab đang mở và dừng khi request lỗi, không tải được bản đồ hoặc nhân vật không thể tiến thêm. Request đang chạy sẽ hoàn tất trước khi dừng hẳn.

Lưới bản đồ gắn với khu vực và tọa độ đã tải; khi player đổi vùng hoặc vị trí, phản hồi grid cũ bị bỏ qua. Thay đổi ô map, drop, điểm tài nguyên và portal được phát qua `world:change` sau khi ghi dữ liệu thành công; publisher xác định người nhận theo region/map/instance, rồi Redis phân phối event kèm danh sách đó cho các backend. Mỗi backend chỉ gửi tới phiên người chơi cục bộ có trong danh sách. Ô đã xóa được phản ánh ngay; những thay đổi cần dữ liệu snapshot mới sẽ kích hoạt tải lại ngay thay vì chờ chu kỳ polling. Nền của khu vực con lấy địa hình từ Đại Vực cha trong `great-regions-atlas.png`; khu vực không thuộc Đại Vực như Tiên Khiếu dùng nền rừng mặc định.

## Luật, chi phí và giới hạn

### Xác suất ambush

Ở ô Monster/Boss kề bên đủ điều kiện trigger và combat không bị cấm, chance ambush hiện tính `clamp(50 + 2*cellRank - disguiseLevel, 5, 95)%`. Đây là phần trăm, không phải xác suất 0–1. Comment trong handler ghi minimum 50 nhưng code clamp đáy ở 5; tài liệu theo behavior code. Request movement tối đa 5 ô.

- Mỗi request đi tối đa 5 ô; từng ô được kiểm tra hợp lệ.
- Trigger trên các ô trung gian vẫn chạy. Nếu combat bắt đầu, phần đường còn lại không được thực hiện.
- Instance bí cảnh và Tiên Khiếu chỉ cho di chuyển trong hình chữ nhật của vùng instance hiện tại.
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
- Realtime feed and actor lifecycle: `backend/internal/worldfeed/`, `backend/internal/handlers/world_change_events.go`, `backend/internal/handlers/world_monsters.go`, `backend/internal/handlers/admin_bosses.go`, `backend/internal/uihub/redis_sub.go`; frontend dispatch/resync: `frontend/src/store/useGameStore.ts`, `frontend/src/store/useWorldStore.ts`.
- Frontend: `frontend/src/components/ExplorationMap.tsx`, `frontend/src/store/slices/worldSlice.ts`.
- UI: `frontend/src/app/page.tsx`, `frontend/src/components/ExplorationMap.tsx`; asset nền: `frontend/public/assets/world/great-regions-atlas.png`, `ground-moss.webp`; cài đặt: `frontend/src/components/LobbySettingsPanel.tsx`, `frontend/src/store/slices/combatSlice.ts`; API `GET/PUT /api/player/combat-settings`.

## Đồng bộ realtime

Vị trí người chơi tiếp tục đi qua kênh `world:pos`; map subscribe kênh này và cập nhật marker trong viewport ngay sau delta, không chờ lần tải grid kế tiếp. Event được giới hạn theo `world_map_id`/`map_instance_id`; khi người chơi rời map hoặc instance, backend gửi event rời scope đến người xem ở vị trí cũ. Snapshot grid cung cấp scope tương ứng để client bỏ event ngoài map/instance hiện tại.

World Change Feed phát delta vòng đời quái/boss khi actor được tạo, di chuyển/cập nhật bởi quản trị, bị hạ trong encounter hoặc hồi sinh. Payload mang actor mới để client cập nhật marker trực tiếp; event cũ chỉ có tọa độ sẽ khiến client tải lại snapshot. Snapshot quái cung cấp vector revision cho các region mà AOI đọc cùng lúc. Combat vẫn chạy theo lượt, realtime không mô phỏng lượt đánh.

Mỗi event có `epoch` và `revision` tăng theo scope region/map/instance. Snapshot lưới, ô bản đồ, điểm tài nguyên và quái trả sequence bắt đầu/kết thúc qua các response header `X-World-*`; snapshot quái kèm vector cho mọi region scope trong AOI. Client đọc lại snapshot nếu revision đổi trong lúc truy vấn, bỏ event trùng/cũ, resync khi phát hiện gap và đồng bộ lại sau khi UI stream kết nối lại. Bản đồ kiểm tra cursor qua `GET /api/world/revisions` định kỳ để bắt cả event cuối bị rơi mà không có event kế tiếp. Redis Pub/Sub vẫn best-effort; snapshot là nguồn phục hồi khi event bị lỡ. Nếu Redis mất sequence, epoch mới khiến client đồng bộ lại thay vì nhầm revision vừa reset với event cũ.

## API liên quan

Các route map/movement/portal được đăng ký tại `backend/internal/server/server.go`; route cụ thể xem [API guide](../guides/API.md). Ground-drop routes cũng được khai báo tại server router.

## Ví dụ flow

```text
chọn đường 4 ô → backend kiểm tra từng ô → trigger encounter ở ô thứ 2
→ dừng tại đó → mở combat theo lượt → kết thúc combat rồi chọn đường mới
```
