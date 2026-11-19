# Wayfinder Map: Hệ Thống Thế Giới Đa Tầng 4 Lớp Chuẩn Cổ Chân Nhân (4-Tier Vertical World System)

## Destination

Đặc tả toàn diện kiến trúc kỹ thuật và lộ trình thực thi để đưa hệ thống **Thế Giới Không Gian 4 Tầng Theo Chiều Dọc (4-Tier Vertical World Map)** vào GuWorld, chuyển đổi từ mô hình mặt đất đơn tầng (Ngũ Vực) sang hệ sinh thái không gian đa tầng chuẩn nguyên tác *Cổ Chân Nhân* (Reverend Insanity):
- **Tầng 0: Địa Uyên (`dia_uyen`)**: Thế giới lòng đất sâu thẳm dưới chân Ngũ Vực (Địa Câu, Địa Mạch, Âm Lưu Cự Hác, mỏ quặng Thổ Đạo, Huyết Trì ngầm, quái thú vực sâu, Rank 5+).
- **Tầng 1: Ngũ Vực (`world_map`)**: Mặt đất hiện hành (Nam Cương, Trung Châu, Bắc Nguyên, Tây Mạc, Đông Hải, Rank 1–9).
- **Tầng 2: Thái Cổ Bạch Thiên (`bai_thien`)**: Cõi trời ánh sáng vĩnh cửu trên Khí Cương Phong Bích (Biển mây, Vân đảo, Quang tinh, Thiết Trảo Phi Hạc, Động Thiên chính đạo, Rank 6–9).
- **Tầng 3: Thái Cổ Hắc Thiên (`hac_thien`)**: Cõi trời bóng đêm vô tận (Màn đêm vĩnh hằng, Tinh thú, Hồn thú Thái Cổ, Mộng Cảnh trôi nổi, cứ địa Dị Nhân, Động Thiên tà phái ẩn thế, Rank 8–9).

Hệ thống bao gồm: mở rộng database schema & cache, cơ chế vượt tầng (Khí Cương Phong Bích & Địa Huyệt Thông Đạo), hệ sinh thái bản đồ/quái vật/tài nguyên tương ứng, cơ chế hạ khiếu cắm rễ Tiên Khiếu trên tầng trời/vực sâu, và giao diện chuyển tầng trên World Atlas.

## Notes

- Domain: World Maps (`world_maps`, `regions`, `region_world_maps`), Exploration Grid (`map_cells`, `monster_spawns`), Movement Pipeline (`handlers/movement.go`), Immortal Aperture Anchoring (`immortal_apertures`, `aperture_portals`), Realtime Map Tracking (`UIRealtimeManager.ts`, `useWorldStore.ts`, `worldPositions.ts`), World Atlas UI (`WorldAtlasModal.tsx`, `ExplorationMap.tsx`).
- Skills to consult: `domain-modeling`, `codebase-design`, `architecture-planning`, `realtime-networking`.
- Standing preferences:
  - Tuân thủ nghiêm ngặt `AGENTS.md`: Logic gameplay tập trung tại `backend/internal/handlers/`, không tự tạo service/repository trừu tượng.
  - Tận dụng hạ tầng đa tầng đã có sẵn trong codebase: bảng `world_maps`, `region_world_maps`, header `X-World-Revision-Map`, và logic cách ly realtime position theo `world_map_id`.
  - Phân tầng tu vi nghiêm ngặt theo đúng canon truyện: Phàm nhân (Rank 1–5) không thể tự ý bay qua Khí Cương Phong Bích lên Bạch Thiên nếu không có Tiên Cổ Ốc hoặc sát chiêu đặc thù (chịu sát thương gió chí mạng).
  - Khí Cương Phong Bích đóng vai trò chướng ngại tự nhiên giữa Tầng 1 và Tầng 2; Địa Câu / Vực Sâu nứt toác đóng vai trò lối vào giữa Tầng 1 và Tầng 0.
  - Cơ chế cắm rễ Tiên Khiếu: Cổ Tiên hạ khiếu ở Tầng 2/3 tăng mạnh sản lượng Thiên Khí (`heaven_qi`), hạ khiếu ở Tầng 0 tăng mạnh sản lượng Địa Khí (`earth_qi`).

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

- [[01-vertical-world-schema-and-dia-uyen-layer](issues/01-vertical-world-schema-and-dia-uyen-layer.md)]: Chuẩn hóa schema database `world_maps` bổ sung Tầng 0 (`dia_uyen`), kích hoạt `bai_thien` và `hac_thien` (Rank 5-9, 6-9, 8-9), cập nhật ordering query `GetWorldMaps`, mở route `/world/maps` & `/admin/world-maps` và kích hoạt dropdown gán tầng trên AdminView UI.
- [[02-heavenly-wind-qi-wall-and-layer-transit-mechanics](issues/02-heavenly-wind-qi-wall-and-layer-transit-mechanics.md)]: Thiết kế & triển khai cơ chế vượt Khí Cương Phong Bích (phàm nhân 90% sát thương bị hất văng, Tiên Cổ Ốc tiêu hao 20 độ bền bảo hộ, Cổ Tiên lướt gió an toàn), Cực Điểm Không Gian (Rank 8 hoặc Cổ Ốc), Địa Câu Chướng Khí (Rank 5+) và Rơi Tự Do Skyfall (`POST /api/world/skyfall`) theo phép chiếu tọa độ 1:1.
- [[03-canon-regions-and-biomes-specification](issues/03-canon-regions-and-biomes-specification.md)]: Xây dựng toàn diện hệ sinh thái 9 Region, quái vật, boss, khoáng sản, bàn loot cho Tầng 0 (Địa Uyên), Tầng 2 (Bạch Thiên), Tầng 3 (Hắc Thiên); đồng thời tái cân bằng triệt để dải cấp quái vật trên bề mặt Ngũ Vực (Rank 1-5 phàm cấp).
- [[04-aperture-anchoring-in-upper-heavens-and-abyss](issues/04-aperture-anchoring-in-upper-heavens-and-abyss.md)]: Triển khai phân hóa sản lượng Thiên Địa Nhị Khí theo tầng hạ khiếu (Địa Uyên 20% Thiên / 350% Địa, Bạch Thiên 300% Thiên / 30% Địa, Hắc Thiên tẩm bổ Ám/Tinh Đạo), kéo dài 50% chu kỳ tai kiếp tại Hắc Thiên (Thiên Ý khó cảm ứng), cách ly tầm nhìn Môn Hộ theo `world_map_id`, và bổ sung Địa Khí Đan / Thiên Khí Đan.
- [[05-frontend-vertical-atlas-and-atmospheric-rendering](issues/05-frontend-vertical-atlas-and-atmospheric-rendering.md)]: Thiết kế giao diện bộ chuyển tầng (Layer Switcher HUD) trên World Atlas Modal với cảnh báo tu vi chuẩn canon (chỉ Cổ Tiên Rank 6+ lên Bạch Thiên, Rank 8+ vô Hắc Thiên), hiệu ứng thị giác khí quyển (Atmospheric Overlays) và nút hành động Rơi Tự Do (Skyfall) trên ExplorationMap.

## Open Tickets (Frontier & Blocked)

- **Frontier (Unblocked - Sẵn sàng giải quyết)**:
  *(Toàn bộ 5 ticket cốt lõi của Hệ thống Thế giới Đa tầng 4 Lớp đã được giải quyết trọn vẹn!)*

- **Blocked (Chờ phụ thuộc)**:
  *(Hiện tại không còn ticket nào bị block)*

## Not yet specified

<!-- Fog of War: Những khía cạnh trong phạm vi nhưng cần chờ các quyết định trên Frontier làm rõ trước khi tạo ticket -->

- **Thiên Địa Dị Biến & Triều Tịch Địa Khí (Earth Tide & Heaven Qi Waves)**: Chu kỳ định kỳ nứt vỡ địa mạch sinh ra Địa Câu mới hoặc sóng bão khí trên Bạch Thiên cuốn trôi tài nguyên.
- **Tiên Cổ Ốc Du Hành Xuyên Tầng (Gu House Layer Navigation)**: Cho phép Tiên Cổ Phòng chở đoàn người phàm nhân vượt qua Khí Cương Phong Bích lên Nhị Thiên an toàn.
- **Mộng Cảnh Trôi Nổi Trên Hắc Thiên (Floating Dream Realms in Black Heaven)**: Cơ chế các khối Mộng Cảnh hoang dã tự do di chuyển tọa độ trên Hắc Thiên theo chu kỳ thời gian.

## Out of scope

<!-- Công việc nằm ngoài phạm vi của nỗ lực này -->

- **Tái hiện đầy đủ 7 Tầng Trời đã vỡ thời Thái Cổ (Xích, Chanh, Hoàng, Lục, Thanh, Lam, Tử Thiên)**: Chỉ tập trung vào 2 tầng trời còn nguyên vẹn theo dòng thời gian truyện (Thái Cổ Nhị Thiên: Bạch Thiên & Hắc Thiên) và Địa Uyên.
- **Mô phỏng vật lý trọng lực 3D liên tục (Continuous 3D Flight Physics)**: Thế giới game giữ vững mô hình 2D Grid nhiều tầng (Discrete Multi-layer Grid) chuyển đổi qua cổng/tọa độ, không làm camera 3D góc nhìn thứ nhất.
