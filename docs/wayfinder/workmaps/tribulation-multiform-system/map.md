# Wayfinder Map: Triển Khai Hệ Thống Đa Dạng Hóa Tai Kiếp Tiên Khiếu Chuẩn Cổ Chân Nhân (Multi-form Aperture Tribulation System)

## Destination

Đặc tả toàn diện kiến trúc kỹ thuật và lộ trình thực thi để đưa hệ thống Đa Dạng Hóa Tai Kiếp (Multi-form Aperture Tribulation System) vào Tiên Khiếu (`ImmortalApertureView`, `backend/internal/handlers/tribulation_*`, `backend/internal/simulator/`): chuyển đổi từ mô hình đơn điệu spawn 1 boss đấm chay sang hệ sinh thái 4 Archetype (Thiên Tai Sinh Thái, Thú Triều Xâm Lăng, Tâm Ma Khảo Nghiệm, Dị Tượng Sàn Đấu Turn-based), chuẩn hóa vòng đời độ kiếp 3 giai đoạn (Điềm báo chuẩn bị ➔ Nghênh kiếp đa cơ chế giải cứu/thủ hộ/đấu trí/giao tranh ➔ Thu hoạch Kiếp Tài & Xử lý Dư ba ô nhiễm thổ nhưỡng), mang lại trải nghiệm chuẩn nguyên tác Cổ Chân Nhân.

## Notes

- Domain: Immortal Aperture (`immortal_apertures`, `aperture_nodes`, `aperture_resource_points`), Tribulation Engine (`tribulation_events`, `tribulation_enemies`), Turn-based Combat Simulator (`backend/internal/simulator/`), Dilemma Narrative / Soul Trial, Kiếp Tài Item Drops.
- Skills to consult: `domain-modeling`, `codebase-design`, `prototype`, `tdd`.
- Standing preferences:
  - Tuân thủ nghiêm ngặt `AGENTS.md`: Gameplay logic tập trung tại `backend/internal/handlers/` và `backend/internal/simulator/`. Không tự tạo service/repo trừu tượng.
  - Tận dụng cấu trúc lưới ô hiện có (`aperture_nodes`, `aperture_resource_points`) và UI `ImmortalApertureView.tsx`.
  - Giữ vững tính tất định (deterministic) của Turn-based Combat Simulator với hạt giống RNG `Seed`.
  - Cân bằng kinh tế & crafting: Kiếp Tài rơi rớt phải tương thích trực tiếp với dữ liệu Items, Recipe Luyện Cổ và Nâng cấp Tiên Cổ Phòng.
  - Vòng đời tất định và hỗ trợ ngoại tuyến: Có cơ chế xử lý dự phòng nếu người chơi offline trong lúc kiếp nạn diễn ra (Trận pháp tự động chống đỡ tiêu hao tài nguyên).

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

- [[01-tribulation-archetypes-and-lifecycle-state-machine](issues/01-tribulation-archetypes-and-lifecycle-state-machine.md)]: Chuẩn hóa 3 Archetype cốt lõi (Môi Trường, Thú Triều, Dị Tượng Sàn Đấu) và Máy trạng thái Vòng Đời 3 Giai Đoạn (HARBINGER ➔ ORDEAL_ACTIVE ➔ AFTERMATH), mở rộng schema `tribulation_events` với `archetype`, `stage`, `hazard_metadata`; kiểm chứng logic hoàn tất qua prototype HTML.
- [[02-environmental-disaster-and-sat-chieu-mitigation](issues/02-environmental-disaster-and-sat-chieu-mitigation.md)]: Chuẩn hóa mô hình Vùng Hiểm Họa (Hazard Zones) trên lưới ô 20x20 bao quát toàn bộ 10 Nhóm Đại Lưu Phái (40+ Dao Paths: Không-Thời nứt vỡ, Kiếm Khí vũ, Mộng Cảnh bành trướng, Độc/Huyết, Nguyên Tố...), ma trận tương sinh tương khắc Sát Chiêu Tiên Đạo, lồng chắn Trận Pháp 3x3 và cơ chế chuyển hóa ngưng tụ Kiếp Tài; kiểm chứng hoàn tất qua prototype HTML.
- [[03-beast-swarm-siege-and-formation-defense](issues/03-beast-swarm-siege-and-formation-defense.md)]: Chuẩn hóa mô hình Thú Triều Xâm Lăng (BEAST_SWARM) với cơ chế Wave Spawning từ 4 góc bản đồ, AI ưu tiên cắn phá Mỏ Tài Nguyên/Dị Nhân trước Hạch Tâm, hệ thống phòng thủ 3 tầng (Tháp Trận bắn tỉa, Hộ Vệ chặn bước, Cổ Tiên/Cổ Phòng nã pháo), cùng cơ chế Nô Đạo bắt sống thú hoang (HP <= 30%); kiểm chứng hoàn tất qua prototype HTML.
- [[05-combat-anomalies-and-dao-conflict-modifiers](issues/05-combat-anomalies-and-dao-conflict-modifiers.md)]: Chuẩn hóa sàn đấu Dị Tượng Turn-based Simulator (`ANOMALOUS_COMBAT`) với trường biến dị môi trường (Sét giáng, Bão tuyết cản trở mana, Cấm pháp tuyệt vực), công thức ma sát suy giảm sát thương do Đạo Ngân Xung Đột ngoại trừ Chí Tôn Tiên Thể, cơ chế Boss Intent Telegraphing tụ lực 2 lượt và Phá Tụ Lực (Stance Break); kiểm chứng hoàn tất qua prototype HTML.
- [[06-tribulation-spoils-and-ecological-fallout](issues/06-tribulation-spoils-and-ecological-fallout.md)]: Chuẩn hóa kết toán hậu độ kiếp (Phase 3 AFTERMATH) với điểm Kiếp Tài rơi rớt trên lưới ô (tương thích `Item.csv` & công thức luyện Tiên Cổ/Cổ Phòng), mô phỏng hư hại mỏ tài nguyên và ô thổ nhưỡng nhiễm Đạo Ngân, cùng 2 lựa chọn chiến lược tái thiết: Thanh Tẩy Gột Rửa (`CLEANSE`) vs Biến Nguy Thành Cơ (`ECOLOGICAL_PIVOT`) mở mỏ tài nguyên mới; kiểm chứng hoàn tất qua prototype HTML.
- [[07-research-canon-calamity-matrix](issues/07-research-canon-calamity-matrix.md)]: Hoàn tất khảo cứu ma trận tai kiếp chuẩn nguyên tác Cổ Chân Nhân và ánh xạ toàn diện vào codebase: 4 Archetype x 4 Cấp bậc Rank (Lục chuyển Địa Tai đến Cửu chuyển Hỗn Độn Kiếp), danh mục Kiếp Tài tương thích với Item.csv, cùng cơ chế 3 giai đoạn và phục hồi dư ba ô nhiễm thổ nhưỡng.

## Open Tickets (Frontier & Blocked)

- **Frontier (Unblocked - Sẵn sàng giải quyết)**:
  *(Tất cả ticket cốt lõi trên lộ trình đã được hoàn thành và kiểm chứng)*

- **Blocked (Chờ phụ thuộc)**:
  *(Không còn ticket nào bị block)*

## Not yet specified

<!-- Fog of War: Những khía cạnh trong phạm vi nhưng cần chờ các quyết định trên Frontier làm rõ trước khi tạo ticket -->

- **Nhân Họa & Môn Hộ Bộc Lộ (Human Calamity & World Exposure)**: Khi Cổ Tiên hạ khiếu cắm rễ vào địa mạch nạp thiên địa nhị khí, vị trí Môn Hộ xuất hiện trên World Map; cơ chế NPC thù địch hoặc người chơi khác phát hiện dị tượng kéo đến tập kích cướp bóc.
- **Cơ Chế Hộ Pháp Nâng Cao & Phân Chia Kiếp Tài**: Thỏa thuận chia sẻ chiến lợi phẩm Kiếp Tài và phân tán rủi ro Đạo Ngân cho các Cổ Tiên trợ trận trong Trợ Trận Độ Kiếp.
- **Hỗn Độn Kiếp Cấp Tôn Giả (Chaos Tribulation for Rank 9)**: Sương mù xám hỗn độn nuốt chửng cấu trúc Tiên Khiếu và cơ chế phá vỡ quy tắc vạn vật đặc thù của Tôn Giả.

## Out of scope

<!-- Công việc nằm ngoài phạm vi của nỗ lực này -->

- [[04-soul-mind-dilemma-and-mental-duel](issues/04-soul-mind-dilemma-and-mental-duel.md)]: Loại bỏ cơ chế Tâm Ma Vấn Tâm & Quyết Đấu Thức Hải do mang phong cách tu tiên huyền huyễn chung chung (generic xianxia), không phù hợp với bản sắc thực dụng và sinh thái khắc nghiệt của Cổ Giới; kiếp nạn tập trung vào Sinh Thái Tiên Khiếu (Môi trường, Thú triều) và Chiến Đấu Sàn Đấu Theo Lượt.
- **Hệ thống Vật lý Realtime & Action Combat**: Kiếp nạn giải quyết qua cơ chế mô phỏng chiến thuật trên lưới ô Tiên Khiếu và Turn-based Simulator, không làm hitbox hay di chuyển realtime vật lý.
- **Bùa Miễn Kiếp / Skip Độ Kiếp Trả Phí**: Không cung cấp cơ chế pay-to-win bỏ qua tai kiếp; độ kiếp là cốt lõi sinh tử không thể né tránh.
- **Tự Động Hóa Vô Điều Kiện (Free AFK Automation)**: Nếu không có trận pháp hoặc sát chiêu phòng hộ mà bỏ mặc Tiên Khiếu, tai kiếp sẽ tàn phá hủy hoại tài nguyên và vách khiếu.
