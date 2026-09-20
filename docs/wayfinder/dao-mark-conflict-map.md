# [WAYFINDER MAP] Cơ Chế Xung Đột Đạo Ngân & Cân Bằng Lưu Phái (Dao Mark Conflict & Path Balance)

**Status:** Completed (Destination Reached)  
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

- [[WF-DAO-01] Khảo sát hiện trạng tích hợp Path, Dao Marks, Attainment trong Simulator & Handlers](./tickets/WF-DAO-01-research-path-mastery-dao-mark-combat.md): Đã xác định toàn bộ vị trí lưu trữ dữ liệu (`immortal_apertures.dao_marks`, `players.path_masteries`), công thức hiện hành trong `gamefx` và `simulator`, đồng thời chốt giải pháp kiến trúc **`DaoCombatProfile`** tiền tính toán để đảm bảo truy xuất $O(1)$ trong vòng lặp combat mà không gây suy giảm hiệu năng. Chi tiết xem tại [WF-DAO-01 Findings](./research/WF-DAO-01-findings.md).
- [[WF-DAO-02] Thiết kế 10 Nhóm Lưu Phái (Dao Groups), Ma Trận Tương Thích & Vùng Đệm An Toàn](./tickets/WF-DAO-02-spec-dao-groups-and-compatibility-matrix.md): Đã hoàn thành phân loại 49 lưu phái chuẩn thành 10 Dao Groups, xây dựng ma trận tương thích nhóm $10 \times 10$, xác lập 14 cặp Override tương sinh/tương khắc, và cơ chế Grace Threshold chống "kẹt build" khi đạo ngân ngoại hệ $\le \max(1000, 2\%)$. Chi tiết xem tại [WF-DAO-02 Spec](./specs/WF-DAO-02-dao-groups-compatibility.md).
- [[WF-DAO-03] Thiết kế công thức Điều Hòa Cảnh Giới (Realm Control) & Khuếch Đại Đạo Ngân](./tickets/WF-DAO-03-formula-realm-control-and-dao-amplification.md): Đã thiết lập hoàn chỉnh hệ thống công thức toán học cho `RawConflict`, `RealmControl` ($0\% \to 80\%$), `ConflictMultiplier` (hàm mượt với sàn bảo hộ $\ge 0.60$), `DaoAmp` 2 đoạn chống lạm phát số mũ, và `RealmEfficiency` ($\times 1.00 \to \times 1.45$). Chi tiết xem tại [WF-DAO-03 Spec](./specs/WF-DAO-03-realm-control-dao-amplification.md).
- [[WF-DAO-04] Đặc tả Phòng Thủ Tự Nhiên (Natural Resistance) & Trade-off Đạo Ngân Bất Tương Xung](./tickets/WF-DAO-04-spec-natural-dao-resistance-and-traits.md): Thiết lập công thức kháng tự nhiên tiệm cận với trần $30\%$ dựa trên xung đột giữa đạo ngân bản thân và đòn đánh ngoại lai; làm rõ trade-off sống còn của đặc tính Bất Tương Xung (Offense $100\%$ không phạt, Defense $0\%$ không kháng tự nhiên). Chi tiết xem tại [WF-DAO-04 Spec](./specs/WF-DAO-04-natural-dao-resistance.md).
- [[WF-DAO-06] Kiến trúc Cache Dao Combat Profile O(1), Data Models & API Contract](./tickets/WF-DAO-06-architecture-combat-profile-cache.md): Xây dựng cấu trúc `DaoCombatProfile` và hàm tiền tính toán `BuildDaoCombatProfile` chạy trong $< 5\mu s$, cung cấp các hàm tra cứu $O(1)$ cho Combat Engine và định hình DTO mở rộng cho endpoint `GET /api/player/daongan`. Chi tiết xem tại [WF-DAO-06 Spec](./specs/WF-DAO-06-combat-profile-architecture.md).

---

## Tickets

| ID | Title | Type | Status | Blocked By |
|---|---|---|---|---|
| [WF-DAO-01](./tickets/WF-DAO-01-research-path-mastery-dao-mark-combat.md) | Khảo sát hiện trạng tích hợp Path, Dao Marks, Attainment trong Simulator & Handlers | `wayfinder:research` | Closed | None |
| [WF-DAO-02](./tickets/WF-DAO-02-spec-dao-groups-and-compatibility-matrix.md) | Thiết kế 10 Nhóm Lưu Phái (Dao Groups), Ma Trận Tương Thích & Vùng Đệm An Toàn | `wayfinder:grilling` | Closed | None |
| [WF-DAO-03](./tickets/WF-DAO-03-formula-realm-control-and-dao-amplification.md) | Thiết kế công thức Điều Hòa Cảnh Giới (Realm Control) & Khuếch Đại Đạo Ngân | `wayfinder:grilling` | Closed | None |
| [WF-DAO-04](./tickets/WF-DAO-04-spec-natural-dao-resistance-and-traits.md) | Đặc tả Phòng Thủ Tự Nhiên (Natural Resistance) & Trade-off Đạo Ngân Bất Tương Xung | `wayfinder:grilling` | Closed | None |
| [WF-DAO-05](./tickets/WF-DAO-05-spec-compound-killer-moves-and-dual-path.md) | Đặc tả Sát Chiêu Phức Hợp (Compound Moves) & Cơ Chế Song Đạo Điều Hòa | `wayfinder:grilling` | Closed (Deferred) | None |
| [WF-DAO-06](./tickets/WF-DAO-06-architecture-combat-profile-cache.md) | Kiến trúc Cache Dao Combat Profile $O(1)$, Data Models & API Contract | `wayfinder:task` | Closed | None |

---

## Not yet specified (Fog of War)

- **Hệ Thống Sát Chiêu Phức Hợp (Compound Killer Moves & Dual-Path Harmony)**: Triển khai chi tiết khi tiến hành refactor toàn diện hệ thống Sát Chiêu (đã note tại [WF-DAO-05 Note](./notes/WF-DAO-05-compound-killer-moves-note.md)).
- **Dị Chủng Đạo Ngân Trong Vết Thương (Residual Dao Marks)**: Sát chiêu Tiên cấp lưu lại đạo ngân ngoại lai khiến việc hồi phục HP đòi hỏi phải thanh tẩy đạo ngân trước.
- **Môi Trường Chiến Đấu Đạo Ngân (Environmental Dao Marks)**: Vùng đất (ví dụ Hỏa Vực, Băng Nguyên) có mật độ đạo ngân tự nhiên hỗ trợ phái tương sinh và áp chế phái tương khắc.
- **Tùy Ý Tương Xung (Controllable Dao Conflict)**: Đặc tính cảnh giới cực cao (Rank 9/Tôn Giả) cho phép công kích không xung đột nhưng phòng thủ vẫn giữ kháng tính tự nhiên.

---

## Out of scope

- Hệ thống Luyện Đạo tự do tùy biến công thức (thuộc nhánh Refinement Gameplay).
- Địa Mạch và Động Thiên xâm chiếm thời gian thực (thuộc roadmap Tiên Khiếu Vực Ngoại).
