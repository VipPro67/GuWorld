# Wayfinder Map: Hệ Thống Effect Theo Lưu Phái (Path Effects Framework)

## Destination

Đặc tả toàn diện kiến trúc kỹ thuật và thiết kế gameplay cho Hệ Thống Effect Theo Lưu Phái (Path Effects Framework) trong GuWorld: chuẩn hóa khung cơ chế chung (Universal Combat Baseline), định hình bản sắc cơ chế riêng theo 7 nhóm Lưu Phái chủ đạo (In-Combat Signatures với tương tác Grid 5x5 và Custom Gauges/Stacks), và danh mục năng lực thế giới ngoài giao tranh (World Map Exploration & Tiên Khiếu/Kinh Tế).

## Notes

- Canonical Tracker: [GitHub Issue #447](https://github.com/VipPro67/GuWorld/issues/447).
- Domain: Combat Simulator (`backend/internal/simulator/`), Effect Catalog & Block Manifest (`backend/internal/effectcatalog/`), Dao Combat Profile (`backend/internal/gamefx/`), World Movement & Actions (`handlers/movement.go`, `handlers/aperture_*`), Game Data (`backend/data/Gu.csv`, `Race.csv`, `KillerMoves.csv`).
- Skills to consult: `domain-modeling`, `codebase-design`, `architecture-planning`, `grilling`.
- Standing preferences:
  - Tuân thủ `AGENTS.md`: Giữ logic gameplay trong `handlers/` và simulator tất định trong `simulator/`. Không tùy tiện tạo repository/service trừu tượng.
  - Tận dụng chuẩn Block v2 JSON schema và hệ thống Đạo Ngân (`DaoCombatProfile`).
  - Simulator turn-based phải tuyệt đối deterministic (cùng input + seed cho ra cùng kết quả).
  - Tương tác vị trí trên Grid 5x5 (Va chạm, Đẩy lùi, Xuyên thấu hàng/cột, Thay đổi ô địa hình) là trọng tâm để đa dạng hóa chiến thuật thay vì chỉ cộng trừ chỉ số phẳng (stat sticks).

## Decisions so far

<!-- the index: one line per closed ticket, enough to judge relevance, then zoom the link for the detail the ticket holds -->

Chưa có quyết định nào được đóng. Bản đồ vừa được khởi tạo từ phiên phỏng vấn định hướng.

## Open Tickets (Frontier & Blocked)

- **Frontier (Unblocked - Sẵn sàng giải quyết)**:
  - [[#448] Chốt quy chuẩn Khung Cơ Chế Chung (Universal Combat Baseline) trong combat 5x5](issues/01-universal-combat-baseline.md)
  - [[#452] Chốt hệ thống năng lực Ngoài Combat trên Bản Đồ Thế Giới (World Map & Exploration)](issues/05-world-map-and-exploration.md)
  - [[#453] Chốt hệ thống năng lực Ngoài Combat trong Tiên Khiếu & Kinh Tế (Aperture & Cultivation)](issues/06-aperture-and-cultivation-economy.md)

- **Blocked (Chờ phụ thuộc)**:
  - [[#449] Chốt cơ chế riêng và Custom Gauges cho Nhóm Thể Thuật & Không - Thời Gian](issues/02-strength-and-space-time-signatures.md) (Blocked by [#448](https://github.com/VipPro67/GuWorld/issues/448))
  - [[#450] Chốt cơ chế riêng cho Nhóm Tinh Thần, Bầy Đàn và Tà Đạo](issues/03-mental-swarm-and-unorthodox-signatures.md) (Blocked by [#448](https://github.com/VipPro67/GuWorld/issues/448))
  - [[#451] Chốt cơ chế riêng cho Nhóm Môi Trường & Bổ Trợ](issues/04-environment-and-support-signatures.md) (Blocked by [#448](https://github.com/VipPro67/GuWorld/issues/448))
  - [[#454] Chốt kiến trúc Block v2 Schema và tích hợp Simulator Turn-based](issues/07-block-v2-schema-and-simulator-contract.md) (Blocked by [#448](https://github.com/VipPro67/GuWorld/issues/448), [#449](https://github.com/VipPro67/GuWorld/issues/449), [#450](https://github.com/VipPro67/GuWorld/issues/450), [#451](https://github.com/VipPro67/GuWorld/issues/451))

## Not yet specified

- **Cơ chế Sát Chiêu Kết Hợp Đa Lưu Phái (Cross-Path Killer Moves)**: Quy tắc hòa trộn hiệu ứng khi kết hợp 2-3 Cổ trùng khác nhóm Archetype (ví dụ Lực Đạo + Vũ Đạo = Không Gian Trọng Lực Kích).
- **Thiết kế UI/HUD cho Custom Gauges trên màn hình Turn-based**: Cơ chế hiển thị thanh tích lũy Kiếm Ý, Thú Ảnh, Hồn Phách trên từng chiến binh tại giao diện chiến đấu.
- **Ma trận cân bằng số liệu theo từng Rank 1 -> 9**: Trị số cụ thể về damage, shield, stack cap cho từng phẩm cấp Cổ.

## Out of scope

- Triển khai code runtime hoặc chỉnh sửa database schema trong giai đoạn Wayfinder này (tập trung vào chốt quyết định & đặc tả).
- Thay thế toàn bộ engine đồ họa hoặc animation sprite/VFX của client frontend.
- Cân bằng chỉ số chi tiết cho từng Cổ trùng cụ thể trong `Gu.csv` (sẽ thực hiện theo các gói batch sau khi chốt framework).
