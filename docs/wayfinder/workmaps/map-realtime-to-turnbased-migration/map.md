# Wayfinder Map: Chuyển Đổi Bản Đồ Realtime Sang Bản Đồ Chiến Lược / Theo Lượt (Turn-based World Map Migration)

## Destination

Đặc tả và lộ trình quyết định kiến trúc để loại bỏ hoàn toàn hệ thống Map Realtime (Phaser 4.2.1, Go Zone Sim 20Hz, WebTransport/QUIC, Spatial Index) và thay thế bằng Mô hình Bản đồ Chiến lược (Node-Graph / Strategic Grid) kết nối trực tiếp với hệ thống Combat Turn-based (`backend/internal/simulator/`, `CombatScreen.tsx`), đồng thời bảo tồn và nâng cấp các hoạt động tương tác, kết nối giữa người chơi (Presence, Local Stalls, Caravan/Vận Tiêu, Mỏ Nguyên Tuyền, Tranh đoạt Bí Cảnh).

## Notes

- Domain: World Navigation, Turn-based Combat Integration, Social Presence & Encounters, Deprecation of 20Hz Zone Sim & WebTransport.
- Skills to consult: `domain-modeling`, `codebase-design`, `grilling`, `prototype`, `research`.
- Standing preferences:
  - Tuân thủ quy ước `AGENTS.md`: Logic gameplay tập trung tại `backend/internal/handlers/`, không tạo thêm các tầng trừu tượng dư thừa.
  - Tái sử dụng tối đa cấu trúc dữ liệu bền vững sẵn có: `regions`, `map_cells`, `primeval_veins`, `monster_spawns`, `monster_templates`.
  - Giảm thiểu tài nguyên phần cứng: Thay thế WebTransport UDP và 20Hz Tick bằng REST API và WebSocket / SSE hiện có cho đồng bộ trạng thái.
  - Phù hợp với cốt truyện và không khí *Cổ Chân Nhân*: Nhấn mạnh vào yếu tố chiến lược, mưu lược, hiểm địa, quản lý tiêu hao Chân Nguyên / Thể Lực và tương tác nhân - nhân.

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

- [[01-realtime-stack-audit-and-deprecation-boundary](issues/01-realtime-stack-audit-and-deprecation-boundary.md)]: Tháo dỡ hoàn toàn Zone Sim 20Hz, WebTransport QUIC, Spatial Index và Phaser; tái sử dụng các bảng dữ liệu thế giới, loot idempotent và hệ thống Combat Turn-based (`simulator/`).
- [[02-world-navigation-model](issues/02-world-navigation-model.md)]: Thống nhất Mô hình Lai (Hybrid) — Cấp Vực dùng Node-Graph POI kèm Travel Time/Caravan; cấp Địa Danh/Phó Bản mở Tactical Grid từng ô kế thừa từ `map_cells`.
- [[03-world-encounter-and-turnbased-combat-trigger](issues/03-world-encounter-and-turnbased-combat-trigger.md)]: Kích hoạt Turn-based Combat chủ động tại Node/Grid hoặc bị Boss phục kích; bảo tồn cơ chế Skip/Auto-resolve nhanh; quản lý session giao tranh an toàn.
- [[04-social-presence-and-pvp-interactions](issues/04-social-presence-and-pvp-interactions.md)]: Hiển thị danh sách Cổ Sư hiện diện tại Node qua WebSocket/SSE; hỗ trợ Chợ Vỉa Hè P2P miễn thuế; phân tầng PvP: Võ Đài đồng bộ thời gian thực, Dã Ngoại PK dùng Bán đồng bộ (Mirror Clone) chống thoát game.
- [[05-primeval-veins-and-contested-resources-migration](issues/05-primeval-veins-and-contested-resources-migration.md)]: Mỏ Nguyên Tuyền cho phép đả tọa hấp thu định kỳ có tranh đoạt vị trí; xây dựng hoạt động cốt lõi Vận Tiêu (Caravan) & Phục Kích Cướp Tiêu dọc các tuyến đường kết nối Node.
- [[06-frontend-cleanup-and-strategic-map-view](issues/06-frontend-cleanup-and-strategic-map-view.md)]: Gỡ bỏ Phaser 4.2.1 và WebTransport client (>30MB bundle); xây dựng giao diện React `StrategicWorldView` gồm 2 tầng (World Node-Graph Atlas + Tactical Grid) hỗ trợ Tuyên Chiến và Skip Combat.


## Not yet specified

<!-- Fog of War: Các khía cạnh cần làm rõ sau khi các quyết định cốt lõi trên Frontier được chốt -->

- **Bản đồ Giới Bích & Áp Chế Ngũ Vực**: Chi tiết cơ chế debuff áp chế tu vi và tiêu hao chân nguyên khi người chơi vượt biên giới giữa các đại vực trong mô hình di chuyển mới.
- **Hệ thống Khí Tượng & Thiên Tai Dã Ngoại**: Cơ chế bão cát Tây Mạc, đầm lầy độc Nam Cương, bão tuyết Bắc Nguyên tác động lên lộ trình di chuyển của người chơi.
- **Cơ chế Thuê / Cắt Cử Cổ Trùng Hộ Thân (AFK Guarding)**: Cho phép người chơi cắm mốc hoặc để lại linh thú / Cổ trùng trấn thủ cứ điểm khi offline.

## Out of scope

<!-- Những hạng mục nằm ngoài phạm vi của nỗ lực chuyển đổi này -->

- **Hệ thống Vật lý Va Chạm 2D / 2.5D**: Bỏ hoàn toàn việc tính toán hitbox, collision mesh, góc bắn skillshot trên không gian tọa độ liên tục.
- **Chế độ Realtime Combat trên Map**: Giữ vững định hướng chiến đấu theo lượt (Turn-based Combat), không duy trì chế độ chiến đấu thời gian thực song song.
