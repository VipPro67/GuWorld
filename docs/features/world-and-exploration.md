# Thế giới và khám phá

## Tính năng làm gì?

Bản đồ thế giới được tổ chức theo hệ thống 4 tầng thẳng đứng (Tầng 0: Địa Uyên, Tầng 1: Ngũ Vực `world_map`, Tầng 2: Bạch Thiên `bai_thien`, Tầng 3: Hắc Thiên `hac_thien`). Bản đồ theo vùng là nơi di chuyển, gặp quái, thu thập tài nguyên, dùng cổng và tương tác vật phẩm rơi. Di chuyển được xử lý theo từng ô; world realtime đồng bộ vị trí/presence, còn chiến đấu vẫn theo lượt.

## Điều kiện mở và quyền truy cập

Người chơi cần đăng nhập và hoàn tất onboarding để dùng world. Một số vùng, portal, tài nguyên hoặc điểm dịch chuyển có yêu cầu rank, tọa độ, trạng thái chiến đấu hay cấu hình dữ liệu.

## Cách sử dụng

Chọn đường đi trên bản đồ rồi gửi move. Chú giải lưới phân biệt điểm dịch chuyển, Nguyên Tuyền, chiến lợi phẩm, cổng Bí Cảnh và cổng Tiên Khiếu; các điểm tài nguyên khác có ký hiệu riêng. Backend kiểm tra từng ô trên đường, áp dụng trigger trung gian và dừng tại ô đầu tiên bắt đầu encounter. Chọn người chơi trên lưới để mở popup xem hồ sơ, tiến gần/khai chiến PvP (cần trong tầm đánh) hoặc giao dịch tại chỗ (cùng khu vực, tối đa 10 ô, không cần Cổ liên lạc). Portal/teleport dùng flow movement dùng chung để kiểm tra đích và phát sự kiện cập nhật vị trí. Chọn drop để mở popup xem tiền tệ, vật phẩm, Cổ, bảo hộ và hạn tồn tại; có thể tiến gần theo từng chặng tối đa 5 ô và chỉ nhặt khi đứng đúng tọa độ.

Trong **Lobby → Cài Đặt → Chiến thuật turn-based**, bật **Tự động xử lý chiến đấu khi farm** để encounter quái phát sinh lúc di chuyển được mô phỏng và settle ngay; màn điều khiển cùng báo cáo trận không mở. Tắt tùy chọn để vào encounter theo lượt như thường lệ. Cài đặt này lưu trong `combat_settings.skip_battle` và không áp dụng cho encounter bắt đầu trực tiếp từ màn chiến đấu.

Mở **Khám Phá Dã Ngoại** từ Lobby để dùng bản đồ trong popup. **Auto farm quanh vị trí hiện tại** ưu tiên nhắm quái đang thấy trong phạm vi 2 ô; khi chưa thấy mục tiêu, nhân vật tuần tra theo vòng nhỏ bán kính 2 ô để tìm encounter. Auto farm gửi tuần tự lệnh di chuyển qua API hiện có; nhấn Dừng hoặc đóng popup để ngừng gửi request mới. Cần bật `skip_battle`; tính năng chạy trên tab đang mở và dừng khi request lỗi, không tải được bản đồ hoặc nhân vật không thể tiến thêm. Request đang chạy sẽ hoàn tất trước khi dừng hẳn.

Lưới bản đồ gắn với khu vực và tọa độ đã tải; khi player đổi vùng hoặc vị trí, phản hồi grid cũ bị bỏ qua. Thay đổi ô map, drop, điểm tài nguyên và portal được phát qua `world:change` sau khi ghi dữ liệu thành công; publisher xác định người nhận theo region/map/instance, rồi Redis phân phối event kèm danh sách đó cho các backend. Mỗi backend chỉ gửi tới phiên người chơi cục bộ có trong danh sách. Ô đã xóa được phản ánh ngay; những thay đổi cần dữ liệu snapshot mới sẽ kích hoạt tải lại ngay thay vì chờ chu kỳ polling. Nền của khu vực con lấy địa hình từ Đại Vực cha trong `great-regions-atlas.png`; khu vực không thuộc Đại Vực như Tiên Khiếu dùng nền rừng mặc định.

## Luật, chi phí và giới hạn

### Di chuyển xuyên tầng (Layer Transitions & Skyfall)

Di chuyển giữa các tầng thẳng đứng (Tầng 0: Địa Uyên, Tầng 1: Ngũ Vực `world_map`, Tầng 2: Bạch Thiên `bai_thien`, Tầng 3: Hắc Thiên `hac_thien`) chịu sự kiểm tra nghiêm ngặt của tu vi và Tiên Cổ Ốc:
- **Khí Cương Phong Bích (Ngũ Vực ➔ Bạch Thiên)**:
  - Cổ Tiên (Rank 6+): Vận dụng Tiên Thể lướt qua an toàn, tiêu hao 50 Chân Nguyên/Tiên Nguyên (nếu thiếu sẽ chịu 5% Max HP mài mòn).
  - Phàm nhân (Rank < 6): Không có Tiên Thể, nếu không trang bị Tiên Cổ Ốc sẽ bị phong nhận xé rách nhận sát thương 90% Max HP, dính trạng thái `severely_wounded` và bị cuồng phong hất văng trở lại. Nếu ngồi trong Tiên Cổ Ốc đang trang bị (`player_gu_houses`), Cổ Ốc chịu áp lực thay chủ nhân (tiêu hao 20 độ bền và 10% phàm cổ), cho phép phàm nhân phi thăng an toàn.
- **Cực Điểm Không Gian (Bạch Thiên ➔ Hắc Thiên)**:
  - Yêu cầu tu vi Rank 8 (Bát Chuyển) để bước qua an toàn.
  - Cổ Tiên Thất Chuyển (Rank 7): Được phép tiến vào nếu trang bị Tiên Cổ Ốc (tiêu hao 15 độ bền). Nếu không có Cổ Ốc, chịu 40% Max HP sát thương không gian và dính debuff `spatial_suppression` (giảm 30% tốc độ trong 10 phút).
  - Tu vi dưới Rank 7 bị cấm hoàn toàn, chịu 70% Max HP sát thương chấn động khi cố tình xâm nhập.
- **Địa Câu Chướng Khí (Ngũ Vực ➔ Địa Uyên)**:
  - Yêu cầu tu vi Rank 5+ (Ngũ Chuyển đỉnh phong hoặc Cổ Tiên) hoặc Tiên Cổ Ốc.
  - Phàm nhân cấp thấp (Rank < 5) bị chướng khí ngạt thở chặn lại, nhận 50% Max HP sát thương và dính độc `dia_uyen_miasma`.
- **Rơi Tự Do (Skyfall - `POST /api/world/skyfall`)**:
  - Nhân vật đang ở Bạch Thiên, Hắc Thiên hoặc Địa Uyên có thể lao mình rơi tự do theo phương thẳng đứng.
  - Tọa độ chiếu 1:1: Rơi từ tọa độ `(X, Y)` ở Bạch Thiên sẽ đáp thẳng xuống đúng tọa độ `(X, Y)` của đại vực tương ứng ở Ngũ Vực (hoặc từ Hắc Thiên rơi xuống Bạch Thiên).
  - Cổ Tiên hạ giới an toàn; phàm nhân rơi từ Bạch Thiên không có Tiên Thể hay Cổ Ốc chịu 40% Max HP sát thương chấn động khi chạm đất.

### Hạ Khiếu Cắm Rễ Tiên Khiếu Đa Tầng (Vertical Aperture Anchoring)

Cổ Tiên (Rank 6+) hạ khiếu tại các tầng thế giới khác nhau tạo ra sự phân hóa sinh thái và tài nguyên:
- **Tầng 0 (Địa Uyên)**: Dồi dào Địa Khí (350%), cực kỳ hiếm Thiên Khí (20%). Rất thích hợp cho Cổ Tiên tu luyện Thổ Đạo, Kim Đạo, Cốt Đạo, Luyện Đạo.
- **Tầng 1 (Ngũ Vực)**: Cân bằng tự nhiên chuẩn mực (100% Thiên Khí, 100% Địa Khí).
- **Tầng 2 (Bạch Thiên)**: Dồi dào Thiên Khí (300%), thiếu hụt Địa Khí (30%). Cổ Tiên cần mang theo Địa Khí Đan (`EARTH_QI_PILL`) hoặc định kỳ hạ phàm nạp khí.
- **Tầng 3 (Hắc Thiên)**: Thiên Khí dị biến (100% pha trộn Ám/Tinh Đạo), Địa Khí thấp (50%). Hạ khiếu trên 5 phút tẩm bổ thêm Đạo Ngân Ám Đạo và Tinh Đạo khi Thu Khiếu. Dòng chảy thời gian tai kiếp suy giảm 1.5 lần (chu kỳ tai kiếp kéo dài thêm +50% thời gian thực tế do Thiên Ý khó cảm ứng).
- **Cách ly tầm nhìn Môn Hộ**: Môn Hộ của Tiên Khiếu đặt tại Bạch Thiên hoặc Địa Uyên chỉ hiển thị và do thám được bởi người chơi đang ở cùng tầng không gian (`world_map_id`). Chủ Khiếu luôn nhìn thấy Môn Hộ của bản thân.

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
