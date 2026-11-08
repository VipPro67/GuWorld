# Wayfinder Map: Triển Khai Tính Năng Tiên Cổ Phòng Trong Game

## Destination

Đặc tả toàn diện kiến trúc kỹ thuật và lộ trình thực thi để Tiên Cổ Phòng (Immortal Gu House) có thể sử dụng hoàn chỉnh trong game: từ cơ chế kích hoạt Dung Nhập (`GU_HOUSE`) trong Zone Sim 20Hz, tự động cấu trúc bộ 6 chiêu thức Trận Kích chuẩn hóa, hấp thu 100% sát thương vào Form HP độc lập, tiêu hao Tiên Nguyên duy trì theo chênh lệch cảnh giới, đồng bộ suy hao độ bền DB & xử lý vỡ phòng, đến giao diện HUD Action Bar, hotkeys và hiệu ứng client.

## Notes

- Domain: Immortal Gu House (Tiên Cổ Phòng / Cổ Ốc), Zone Sim 20Hz Metamorphosis Engine, Full House Strike Set, Form HP & Structural Durability, Realtime Action Bar.
- Skills to consult: `domain-modeling`, `codebase-design`, `prototype`, `research`.
- Standing preferences:
  - Tuân thủ nghiêm ngặt `AGENTS.md`: Gameplay logic tập trung tại `backend/internal/handlers/` và Zone Sim tại `backend/internal/realtime/sim/`. Không tự tiện đẻ thêm tầng service/repository thừa thãi.
  - Tận dụng cấu trúc `TransformationState` (`TransformationGuHouse = "GU_HOUSE"`) và bảng `player_gu_houses` đã có sẵn.
  - Bảo đảm tính toàn vẹn dữ liệu: Form HP trong trận độc lập, khi kết thúc hoặc vỡ phòng mới cập nhật độ bền Cổ Phòng và truyền chấn thương sang Cổ hạch tâm (`player_gu`) theo `ApplyGuHouseDamage`.
  - Bộ 6 chiêu thức Trận Kích chuẩn hóa theo `CONTEXT.md`: Pháo kích, Xung kích, Phòng hộ, Sửa chữa, Thần thông tối hậu, Lệnh rời phòng.
  - Client Phaser & React Action Bar hoán đổi mượt mà khi nhận sự kiện `TRANSFORMATION_STARTED` (`form_kind == "GU_HOUSE"`).

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

- [[01-gu-house-integration-protocol](issues/01-gu-house-integration-protocol.md)]: Chuẩn hóa opcode `world.gu_house_integrate` (ENTER/EXIT), ma trận kiểm định điều kiện Server, Windup tụ lực 2.0s với cơ chế ngắt Hard CC, hoán đổi 6 Trận Kích & phóng to Hitbox 1.8x, và giải trừ an toàn; kiểm chứng hoàn tất qua prototype demo HTML.
- [[02-standardized-house-strike-builder](issues/02-standardized-house-strike-builder.md)]: Chuẩn hóa ma trận 6 vai trò Trận Kích (Slot 1 Pháo kích tầm xa, Slot 2 Xung kích càn quét 10m, Slot 3 Đại trận khiên 20% Form HP, Slot 4 Tu bổ 15% Form HP, Slot 5 Tuyệt diệt AoE 6.5m, Slot 6 Lệnh rời phòng), cơ chế khấu trừ độ no Cổ hạch tâm và tích hợp `SwapSkillPalette`; kiểm chứng qua prototype demo HTML.
- [[03-upkeep-drain-and-realm-scaling](issues/03-upkeep-drain-and-realm-scaling.md)]: Khóa công thức tiêu hao Tiên Nguyên duy trì hợp nhất, triệt tiêu nghịch lý bản vị qua tỷ lệ 100x và hệ số phạt lũy tiến 1.5x khi vượt Rank; giải trừ cưỡng chế kèm debuff EXHAUSTION khi cạn mana.
- [[04-damage-absorption-and-structural-breach](issues/04-damage-absorption-and-structural-breach.md)]: Hấp thu 100% không tràn sát thương (No Spillover), suy hao độ bền DB phân tầng mượt mà (0.5% tối thiểu, tối đa 10.5%), xử lý Vỡ Phòng hất văng Cổ Tiên mất 30% HP + Stun 3s, độ bền về 0% và truyền 30% chấn thương sang Cổ hạch tâm; cập nhật CONTEXT.md.
- [[05-hud-action-bar-and-client-controls](issues/05-hud-action-bar-and-client-controls.md)]: Chuẩn hóa nút HUD nổi `[Alt+G]` Dung Nhập / Thoát Cổ Phòng, thanh Windup 2.0s trên đầu nhân vật, Action Bar theme Hoàng Kim Kim Cang, thanh Form HP kép vàng kim, hào quang aura 0xeab308 và rung màn hình khi Vỡ Phòng; kiểm chứng qua prototype demo HTML.

## Not yet specified

<!-- Fog of War: Những khía cạnh trong phạm vi nhưng cần chờ các quyết định trên Frontier làm rõ trước khi tạo ticket -->

- **Emergency Field Repair (Tu Bổ Dã Ngoại Khẩn Cấp)**: Khả năng tiêu hao trực tiếp Tiên Nguyên Thạch hoặc vật liệu khoáng thạch trong túi đồ để sửa chữa nhanh Cổ Phòng ngay tại map dã ngoại khi chưa về thành.
- **Dao Mark Resonance & Conflict (Đạo Ngân Tương Sinh & Xung Đột)**: Ảnh hưởng của số lượng Đạo Ngân trên Tiên Cổ Phòng tương tác với Đạo Ngân bản thể Cổ Tiên (tăng uy lực Trận Kích cùng hệ, tăng tiêu hao Tiên Nguyên nếu dị hệ).
- **Mega-hitbox Collision & Trample (Va Chạm Cự Đại & Càn Quét)**: Cơ chế vật lý va chạm khi Tiên Cổ Phòng di chuyển đè bẹp quái vật nhỏ hoặc đẩy lùi các thực thể cản đường.

## Out of scope

<!-- Công việc nằm ngoài phạm vi của nỗ lực này -->

- **Co-op Multi-pilot System**: Cơ chế cho phép nhiều Cổ Tiên cùng bước lên một Tiên Cổ Phòng để phân chia điều khiển (phòng ngự, pháo thủ, hoa tiêu). Thống nhất Tiên Cổ Phòng là phương tiện cá nhân của chủ sở hữu.
- **AFK Auto-pilot AI**: Cơ chế tự động vận hành Cổ Phòng khi người chơi rời máy.
- **Marketplace Trading of Gu Houses**: Mua bán Tiên Cổ Phòng trên sàn giao dịch (Cổ Phòng gắn liền với Cổ hạch tâm bản mệnh của người chơi).
