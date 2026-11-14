# Wayfinder Map: Triển Khai Cơ Chế Chiêu Thức & Biến Thân Tiên Cổ Phòng Theo Lượt (Turn-based Immortal Gu House)

## Destination

Đặc tả toàn diện kiến trúc kỹ thuật và lộ trình thực thi để đưa cơ chế Biến Thân (Dung Nhập) và Bộ Chiêu Thức Trận Kích của Tiên Cổ Phòng (Immortal Gu House) vào hệ thống Chiến Đấu Theo Lượt (Turn-based Combat Simulator & Interactive Combat): chuẩn hóa vòng đời Dung Nhập (Auto vào trận vs Interactive chủ động vào/ra), ma trận 6 chiêu thức Trận Kích chuẩn hóa scale theo sức mạnh tổng thể Cổ Phòng và Cổ hạch tâm, cơ chế Form HP hấp thụ 100% không tràn sát thương và phản phệ Vỡ Phòng (Stun, trừ 30% Max HP, truyền suy hao độ bền DB), tiêu hao tài nguyên & độ no theo chiêu, cùng giao diện thanh máu kép Form HP hoàng kim và aura vfx trên CombatScreen.

## Notes

- Domain: Immortal Gu House (Tiên Cổ Phòng / Cổ Ốc), Turn-based Simulator (`backend/internal/simulator/`), Interactive Combat (`InteractiveCombatControls.tsx`), Form HP & Durability Settlement (`player_gu_houses`, `player_gu`).
- Skills to consult: `domain-modeling`, `codebase-design`, `prototype`, `tdd`.
- Standing preferences:
  - Tuân thủ nghiêm ngặt `AGENTS.md`: Gameplay logic tập trung tại `backend/internal/handlers/` và `backend/internal/simulator/`. Không tự tạo thêm tầng service/repository trừu tượng không cần thiết.
  - Tận dụng cấu trúc `GuHouseState` trong `simulator/models.go` và bảng `player_gu_houses`.
  - Giữ vững tính tất định (deterministic) của Turn-based Combat Simulator với hạt giống RNG `Seed`.
  - Khấu trừ tài nguyên và độ no chuẩn xác: Không trừ mana mỗi đầu lượt; trừ Tiên Nguyên / Chân Nguyên và độ no Cổ hạch tâm mỗi khi thi triển chiêu thức.
  - Xử lý Vỡ Phòng chuẩn hóa: No Spillover khi bị vỡ, Cổ Tiên bị văng ra chịu Stun 1–2 lượt + mất 30% Max HP nhục thân, Cổ Phòng về 0% độ bền, Cổ hạch tâm nhận 30% chấn thương vào độ bền DB.

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

- [[01-turnbased-integration-lifecycle](issues/01-turnbased-integration-lifecycle.md)]: Chuẩn hóa vòng đời Dung Nhập / Thoát Cổ Phòng qua trường `GuHouse.Active`: Auto-mode tự động kích hoạt Turn 1, Interactive-mode cho phép tự chọn vào/ra khẩn cấp với Slot 6 `EXIT_GU_HOUSE`, hoán đổi palette chiêu thức Trận Kích và kiểm định điều kiện tiên quyết (Stun/Seal/Broken); kiểm chứng hoàn tất qua prototype HTML.
- [[02-turnbased-full-house-strike-matrix](issues/02-turnbased-full-house-strike-matrix.md)]: Ma trận 6 Trận Kích chuẩn hóa scale theo Rank, Phàm Cổ và Cổ hạch tâm qua hệ số HousePowerScale, tích hợp Turn Pushback (Slot 2) và Charging 2-turn (Slot 5), kèm fallback tự động khi thiếu effect composition; kiểm chứng hoàn tất qua prototype HTML.
- [[03-damage-absorption-and-structural-breach-settlement](issues/03-damage-absorption-and-structural-breach-settlement.md)]: Chuẩn hóa 100% No Spillover trong simulator/engine.go, xử lý phản phệ vỡ phòng (Choáng 2 lượt + trừ 30% Max HP nhục thân), và kết toán DB độ bền Cổ Phòng cùng giảm 30% độ bền Cổ hạch tâm sau trận; kiểm chứng hoàn tất qua unit test.
- [[04-resource-drain-and-hunger-mechanics](issues/04-resource-drain-and-hunger-mechanics.md)]: Bổ sung trường HungerCost trên SimAction, kiểm soát Tiên Nguyên và khóa Trận Kích khi Cổ hạch tâm cạn độ no trong interactive.go và checkActionCondition, đồng bộ độ no tiêu hao vào DB; kiểm chứng hoàn tất qua unit test.
- [[05-combat-screen-hud-and-interactive-controls](issues/05-combat-screen-hud-and-interactive-controls.md)]: Chuẩn hóa giao diện HUD chiến đấu với thanh Form HP kép hoàng kim, hiệu ứng hào quang aura quanh card combatant, bảng 6 nút Trận Kích chuyên biệt có hoán đổi lệnh Vào/Thoát phòng và animation rung chấn vỡ phòng; kiểm chứng qua prototype HTML.

## Open Tickets (Frontier & Blocked)

*(Không còn ticket mở — Toàn bộ bản đồ kiến trúc đã được đặc tả và kiểm chứng hoàn tất)*

## Not yet specified

<!-- Fog of War: Những khía cạnh trong phạm vi nhưng cần chờ các quyết định trên Frontier làm rõ trước khi tạo ticket -->

- **Dao Mark Resonance & Conflict (Đạo Ngân Tương Sinh & Tương Khắc)**: Ảnh hưởng của số lượng Đạo Ngân trên Tiên Cổ Phòng tương tác với Đạo Ngân của Cổ Tiên (tăng uy lực Trận Kích nếu đồng hệ, tăng tiêu hao Tiên Nguyên nếu dị hệ).
- **VFX & Shader Đồ Họa Chuyên Biệt**: Hiệu ứng đồ họa nâng cao (như mô hình pháo đài 2D/3D bao quanh sprite nhân vật khi ở trong Cổ Phòng) sau khi hoàn thiện khung logic thanh Form HP và Aura cơ bản.
- **Tu Sửa Dã Ngoại Khẩn Cấp (Emergency Field Repair)**: Cơ chế tiêu hao trực tiếp Tiên Nguyên Thạch hoặc khoáng thạch trong túi đồ để hồi phục nhanh độ bền Cổ Phòng ngoài map dã ngoại.

## Out of scope

<!-- Công việc nằm ngoài phạm vi của nỗ lực này -->

- **Hệ thống Vật lý Realtime & Va chạm 2D**: Bỏ hoàn toàn việc tính toán hitbox, projectile geometry, va chạm vật lý theo tọa độ liên tục (đã chuyển giao sang Turn-based).
- **Co-op Multi-pilot System**: Nhiều người cùng điều khiển một Tiên Cổ Phòng. Thống nhất Cổ Phòng là phương tiện cá nhân của chủ sở hữu.
- **Giao dịch Tiên Cổ Phòng trên Chợ (Marketplace Trading)**: Cổ Phòng gắn liền với Cổ hạch tâm và bản mệnh của người chơi.
