# Wayfinder Map: Triển Khai Hệ Thống Tiên Khiếu Sát Cốt Truyện (Hạ Khiếu, Đón Khách, Trợ Trận Độ Kiếp)

## Destination

Bộ Đặc tả Kiến trúc Kỹ thuật & Quyết định Thiết kế toàn diện (Technical Architecture Spec & Decision Map) đưa toàn bộ hệ thống Tiên Khiếu Cổ Tiên bám sát cốt truyện *Cổ Chân Nhân*: từ cơ chế Hạ Khiếu cắm rễ sinh Môn Hộ thực thể trên bản đồ thế giới Ngũ Vực, kiến trúc Tiên Khiếu Giới Vực đa người chơi cùng ma trận phân quyền khách & trần áp lực Đạo Ngân, hệ thống Trợ Trận Độ Kiếp với Thiên Đạo Thích Ứng Biến Dị và phân bổ chiến lợi phẩm, đặc quyền Duy Ngã Độc Tôn & cơ chế hóa thân Phúc Địa Vô Chủ khi chủ khiếu tử vong, đến trọn bộ giao diện Môn Hộ và HUD tác chiến client.

## Notes

- Domain: Immortal Aperture System (Tiên Khiếu, Phúc Địa, Động Thiên), World Gateway (Môn Hộ Thế Giới), Zone Sim 20Hz Instance, Co-op Tribulation (Trợ Trận Độ Kiếp), Reactive Mutation Engine, Aperture Dao Stress, Aperture Lord Sovereignty & Traitor Rebellion.
- Skills to consult: `domain-modeling`, `codebase-design`, `prototype`, `research`.
- Standing preferences:
  - Tuân thủ nghiêm ngặt `AGENTS.md`: Gameplay logic tập trung tại `backend/internal/handlers/` và Zone Sim tại `backend/internal/realtime/sim/`. Không tự tiện sinh thêm tầng service/repository thừa thãi.
  - Sử dụng cơ chế khóa giao dịch an toàn (`FOR UPDATE`) và mô hình di chuyển nguyên tử trong `handlers/movement.go` khi luân chuyển người chơi qua Môn Hộ.
  - Vòng đời Zone Tiên Khiếu trong Zone Sim phải lazy/on-demand: chỉ spin-up khi có người và hibernate khi rỗng, không giữ bộ nhớ lãng phí.
  - Đảm bảo tính cân bằng và tàn khốc đúng chất *Cổ Chân Nhân*: Không hỗ trợ kéo rank miễn phí; Thiên Đạo luôn nhân hệ số khó tương xứng khi có người trợ trận độ kiếp.

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

- [[01-aperture-anchoring-world-gateway](issues/01-aperture-anchoring-world-gateway.md)]: Chuẩn hóa quy tắc nhất thể Cổ Tiên & Tiên Khiếu (bình thường ngự trong nhục thân, không có trên map thế giới; khi hạ khiếu cắm rễ sinh Môn Hộ và Cổ Tiên bắt buộc ở BÊN TRONG tọa trấn, không thể ra ngoài world; muốn ra ngoài bắt buộc phải Thu Khiếu), DDL bảng `aperture_portals`, tích hợp marker thế giới với cơ chế ẩn thân thiên cơ, ma trận 6 chính sách ra vào và 3 chốt khóa an toàn tuyệt đối khi thu khiếu; kiểm chứng qua prototype demo HTML.
- [[02-aperture-instance-and-guest-matrix](issues/02-aperture-instance-and-guest-matrix.md)]: Chuẩn hóa Zone sim lazy spin-up/hibernate với `sim.ZoneKey("immortal_aperture")`, DDL bảng `aperture_guest_roster`, ma trận phân quyền 3 vai trò (Visitor/Assistant/Prisoner), công thức tính áp lực Đạo Ngân ($\Delta R \times 250$ HP/s) kèm cơ chế Thu Liễm Khí Tức và lệnh Trục Xuất khẩn cấp; kiểm chứng qua prototype demo HTML.
- [[03-coop-tribulation-reactive-mutation](issues/03-coop-tribulation-reactive-mutation.md)]: Chuẩn hóa DDL bảng `tribulation_coop_participants`, công thức tăng độ khó Thiên Đạo theo tổ đội ($\text{DifficultyMultiplier} = 1.0 + \sum [0.4 + 0.3 \times \Delta R]$), cơ chế sinh Boss Biến Dị tương khắc lưu phái trợ thủ, và ma trận phân bổ chiến lợi phẩm / chuyển tiền thù lao tự động; kiểm chứng qua prototype demo HTML.
- [[04-aperture-lord-sovereignty-traitor-pk](issues/04-aperture-lord-sovereignty-traitor-pk.md)]: Chuẩn hóa bùa lợi Duy Ngã Độc Tôn (-50% mana, +25% dmg, +40% kháng CC), DDL bảng `aperture_alliance_contracts` & phản phệ Tín Đạo khi phản bội, quy trình hóa thân thành Phúc Địa Vô Chủ với DDL `aperture_land_spirits` có chấp niệm di sản; kiểm chứng qua prototype demo HTML.
- [[05-client-hud-aperture-gateway-ui](issues/05-client-hud-aperture-gateway-ui.md)]: Chuẩn hóa 4 thành phần UI/UX client gồm Môn Hộ thực thể Phaser 4, Modal phân quyền khách & Minh Ước Tín Đạo, HUD Tam Khí & Duy Ngã Độc Tôn & cảnh báo Áp Lực Đạo Ngân, và Sảnh chuẩn bị độ kiếp với biểu đồ nhiệt Thiên Đạo phẫn nộ & dự báo quái biến dị tương khắc; kiểm chứng qua prototype demo HTML.

## Not yet specified

<!-- Fog of War: Những khía cạnh trong phạm vi nhưng cần chờ các quyết định trên Frontier làm rõ trước khi tạo ticket -->

- **Thôn Phệ Tiên Khiếu (Aperture Annexation / Devouring)**: Quy tắc Cổ Tiên thăng cấp bằng cách nuốt chửng phúc địa vô chủ hoặc phúc địa của địch thủ (cần cùng lưu phái và cảnh giới) để bỏ qua tai kiếp.
- **Sát Chiêu Trận Pháp Phong Tỏa Môn Hộ (Gateway Sealing & Defensive Formations)**: Bố trí Tiên Trận quanh Môn Hộ ngoại giới để bẫy kẻ thù hoặc ẩn giấu hoàn toàn khỏi suy diễn thiên cơ.

## Out of scope

<!-- Công việc nằm ngoài phạm vi của nỗ lực này -->

- **Xuyên Toàn Bộ Không Gian Ngũ Vực Không Giới Hạn (Infinite Dimension Sharding)**: Server không mô phỏng hàng ngàn Tiên Khiếu hoạt động vật lý đồng thời 24/7; chỉ spin-up Zone Sim instance cho Tiên Khiếu khi có người chơi hiện diện (lazy spin-up / hibernate khi không có người).
- **Hệ Thống Tiên Khiếu Phàm Nhân (Mortal Apertures World Hosting)**: Phàm nhân (Rank 1-5) không có Tiên Khiếu độc lập có thể hạ khiếu làm map; chỉ áp dụng cho Rank 6+ (Cổ Tiên).
