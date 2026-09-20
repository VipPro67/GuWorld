# [WAYFINDER MAP] Cơ Chế Xung Đột Đạo Ngân & Cân Bằng Lưu Phái (Dao Mark Conflict & Path Balance)

**Status:** In Progress  
**Label:** `wayfinder:map`  
**Tracker:** Local Markdown (`docs/wayfinder/`)

---

## Destination

Thiết kế và tích hợp hoàn chỉnh cơ chế **Xung Đột Đạo Ngân (Dao Mark Conflict)** kết hợp với **Cảnh Giới Lưu Phái (Path Attainment)** và **Lưu Phái (Paths)** hiện có trong codebase GuWorld. Biến Đạo Ngân từ một chỉ số buff đơn thuần thành một hệ thống chiến thuật cốt lõi có chiều sâu:
1. Tạo sự phân hóa rõ ràng và cân bằng giữa các phong cách chơi: **Chuyên Tu (Pure Focus)**, **Song Tu / Kiêm Tu (Dual-Path Harmony)**, và **Đa Tu (Multi-Path)**.
2. Cảnh giới lưu phái (Master -> Supreme Grandmaster) đóng vai trò điều hòa, kiểm soát xung đột và tăng hiệu suất khai thác Đạo Ngân.
3. Cơ chế **Phòng Thủ Tự Nhiên (Natural Dao Resistance)** và trade-off với đặc tính **Đạo Ngân Bất Tương Xung (Non-Conflicting Dao Marks)**.
4. Tối ưu hiệu năng chiến đấu thông qua **Dao Combat Profile Cache** $O(1)$ thay vì duyệt 50 lưu phái mỗi đòn đánh.

---

## Notes

- **Domain:** Hệ thống Combat, Sát Chiêu, Tiên Khiếu, Đạo Ngân, Cảnh Giới Lưu Phái trong thế giới Cổ Chân Nhân.
- **Skills:** `domain-modeling`, `grilling`, `codebase-design`, `tdd`, `code-review`.
- **References:**
  - `docs/dao_mark_conflict_design.md` (Design doc chi tiết)
  - `docs/CANH_GIOI_DAO_NGAN.md` (Đặc tả Cảnh giới & Đạo ngân hiện hành)
  - `backend/internal/gamefx/path_mastery.go` (Logic cảnh giới lưu phái hiện tại)
  - `backend/internal/handlers/player_daongan.go` (Tính toán đạo ngân hiện tại)
  - `backend/internal/simulator/engine.go` (Vòng lặp combat simulator)
- **Standing Preferences:**
  - Gameplay logic nằm trong `backend/internal/handlers/` và `gamefx` / `simulator`.
  - Không hardcode 2.500 cặp tương thích; dùng 10 Dao Groups + Override cặp đặc thù.
  - Phải có vùng đệm an toàn (Grace Threshold) để người chơi không bị "hỏng build" khi nhặt vài đạo ngân rác.
  - Áp dụng diminishing returns cho công thức xung đột và phòng thủ tự nhiên (tránh bất tử hoặc sát thương = 0).

---

## Decisions so far

<!-- Danh sách các ticket đã đóng và giải pháp đã chốt -->

---

## Tickets

| ID | Title | Type | Status | Blocked By |
|---|---|---|---|---|
| [WF-DAO-01](./tickets/WF-DAO-01-research-path-mastery-dao-mark-combat.md) | Khảo sát hiện trạng tích hợp Path, Dao Marks, Attainment trong Simulator & Handlers | `wayfinder:research` | Open | None |
| [WF-DAO-02](./tickets/WF-DAO-02-spec-dao-groups-and-compatibility-matrix.md) | Thiết kế 10 Nhóm Lưu Phái (Dao Groups), Ma Trận Tương Thích & Vùng Đệm An Toàn | `wayfinder:grilling` | Open | WF-DAO-01 |
| [WF-DAO-03](./tickets/WF-DAO-03-formula-realm-control-and-dao-amplification.md) | Thiết kế công thức Điều Hòa Cảnh Giới (Realm Control) & Khuếch Đại Đạo Ngân | `wayfinder:grilling` | Open | WF-DAO-02 |
| [WF-DAO-04](./tickets/WF-DAO-04-spec-natural-dao-resistance-and-traits.md) | Đặc tả Phòng Thủ Tự Nhiên (Natural Resistance) & Trade-off Đạo Ngân Bất Tương Xung | `wayfinder:grilling` | Open | WF-DAO-03 |
| [WF-DAO-05](./tickets/WF-DAO-05-spec-compound-killer-moves-and-dual-path.md) | Đặc tả Sát Chiêu Phức Hợp (Compound Moves) & Cơ Chế Song Đạo Điều Hòa | `wayfinder:grilling` | Open | WF-DAO-03 |
| [WF-DAO-06](./tickets/WF-DAO-06-architecture-combat-profile-cache.md) | Kiến trúc Cache Dao Combat Profile $O(1)$, Data Models & API Contract | `wayfinder:task` | Open | WF-DAO-04, WF-DAO-05 |

---

## Not yet specified (Fog of War)

- **Dị Chủng Đạo Ngân Trong Vết Thương (Residual Dao Marks)**: Sát chiêu Tiên cấp lưu lại đạo ngân ngoại lai khiến việc hồi phục HP đòi hỏi phải thanh tẩy đạo ngân trước.
- **Môi Trường Chiến Đấu Đạo Ngân (Environmental Dao Marks)**: Vùng đất (ví dụ Hỏa Vực, Băng Nguyên) có mật độ đạo ngân tự nhiên hỗ trợ phái tương sinh và áp chế phái tương khắc.
- **Tùy Ý Tương Xung (Controllable Dao Conflict)**: Đặc tính cảnh giới cực cao (Rank 9/Tôn Giả) cho phép công kích không xung đột nhưng phòng thủ vẫn giữ kháng tính tự nhiên.

---

## Out of scope

- Hệ thống Luyện Đạo tự do tùy biến công thức (thuộc nhánh Refinement Gameplay).
- Địa Mạch và Động Thiên xâm chiếm thời gian thực (thuộc roadmap Tiên Khiếu Vực Ngoại).
