# [WAYFINDER MAP] Hệ Thống Trinh Sát, Suy Diễn Thiên Cơ & Che Giấu Thân Phận

**Status:** Completed (Destination Reached)  
**Label:** `wayfinder:map`  
**Tracker:** Local Markdown (`docs/wayfinder/`)

---

## Destination

Xây dựng hoàn chỉnh hệ thống **Trinh Sát Cục Bộ (Local Inspection)** trên bản đồ Ngũ Vực, **Suy Diễn Thiên Cơ (Remote Divination)** bằng Đạo Ngân & Cảnh Giới Trí Đạo/Tín Đạo, **Ấn Ký Truy Tung (Tracking Marks)** có thời hạn, **Cảm Ứng Cổ Trùng (Gu Resonance)**, và hệ thống **Che Giấu Thiên Cơ & Thiên Cơ Phản Phệ (Concealment & Counter-Backfire)**; loại bỏ hoàn toàn cơ chế xem profile tự do từ xa.

---

## Notes

- **Domain:** Hệ thống trinh sát, suy diễn, che giấu và phản phệ trong thế giới Cổ Chân Nhân.
- **Skills:** `domain-modeling`, `grilling`, `tdd`, `code-review`.
- **References:** `docs/CANH_GIOI_DAO_NGAN.md`, `docs/DaoMarks_Attainment_Design.md`, `docs/WORLDMAP_REALTIME.md`, `CONTEXT.md`, `docs/wayfinder/research/WF-01-findings.md`.
- **Standing Preferences:**
  - Logic gameplay tập trung tại `backend/internal/handlers/`.
  - Không hardcode dữ liệu đã có trong CSV/JSONB; mọi chỉ số trinh sát/ngụy trang đọc động từ effect specs (`Investigation`, `DisguiseLevel`, capabilities).
  - Phân tầng Tiên - Phàm rõ rệt (Rank 1-5 dùng Cổ trùng & khoảng cách vùng; Rank 6+ dùng Đạo ngân, Tiên khiếu, Tiên nguyên, xuyên vực).
  - Schema tuân thủ nguyên tắc mở rộng an toàn bằng `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`.

---

## Decisions so far

- [[WF-01] Research existing Worldmap presence, FOV, and profile access endpoints](./tickets/WF-01-research-worldmap-presence-and-profile.md): Đã khảo sát `worldPosIndex` (bucket 300 cells), endpoint `GET /api/explore/players`, cấu trúc `GET /api/player/profile/:username`, hệ thống `dao_marks`, `path_masteries`, và danh sách các UI consumers cần refactor.
- [[WF-02] Spec Local Inspection vs Remote Divination access rules & UI flow](./tickets/WF-02-spec-local-inspection-vs-remote-divination.md): Khóa endpoint xem profile từ xa, bắt buộc kiểm tra FOV trên map (ngoại trừ Party/Sect). Remote click chuyển thành Context Menu. Modal Suy Diễn hỗ trợ cơ chế Clue Fusion nhập nhiều manh mối để tăng tỷ lệ giải mã.
- [[WF-03] Formula design for Deduction Power vs Concealment Defense & Backfire Thresholds](./tickets/WF-03-formula-deduction-power-vs-concealment.md): Thiết lập hoàn chỉnh công thức toán học $P_{div}$ vs $D_{con}$ dựa trên data-driven effects (không dùng `gu_id`), phân loại 4 ngưỡng kết quả ($\Delta \ge +30$ Đại thành công, $0 \dots +30$ Tiểu thành công, $-30 \dots 0$ Thất bại, $\Delta < -30$ Thiên cơ phản phệ gây sát thương HP/Soul và lộ ngược vị trí).
- [[WF-04] Spec Tracking Marks lifecycle, location tracking, and Sever Clues mechanisms](./tickets/WF-04-spec-tracking-marks-and-sever-clues.md): Quy định 3 nguồn gieo ấn ký (Combat, Divination, Local Proximity), vòng đời các loại ấn ký (Khí tức tàn dư 15 phút, Trí niệm ấn 30p x Rank, Huyết mạch ấn 2h x Rank), cơ chế Chém Đứt Manh Mối (Sever Clues) xóa bỏ theo dõi, và giao diện Radar theo dõi mục tiêu thời gian thực trên Worldmap.
- [[WF-05] Spec Gu Resonance & World Dao Mark Detection across Mortal vs Immortal](./tickets/WF-05-spec-gu-resonance-and-dao-detection.md): Thiết kế cơ chế dò tìm Cổ trùng hoang dã / người chơi mang theo; cơ chế chống dò quét qua effect `CONCEAL_GU` / `PERFECT_DISGUISE`; phân cấp Tiên - Phàm (phạm vi, chi phí, cooldown); và hệ thống sự kiện Xung Kích Đạo Ngân (Dao Pulse) khi có Tiên nhân độ kiếp/đại chiến.
- [[WF-06] Database Schema & API Routes specification for Deduction & Tracking](./tickets/WF-06-db-schema-and-api-routes.md): Thiết kế hoàn chỉnh DDL Schema Database (`player_tracking_marks`, `world_residual_traces`, các cột mở rộng `active_concealment`, `last_sever_clues_at`) và danh sách 6 REST API endpoints cho toàn bộ hệ thống Trinh Sát & Suy Diễn.

---

## Tickets

| ID | Title | Type | Status | Blocked By |
|---|---|---|---|---|
| [WF-01](./tickets/WF-01-research-worldmap-presence-and-profile.md) | Research existing Worldmap presence, FOV, and profile access endpoints | `wayfinder:research` | Closed | None |
| [WF-02](./tickets/WF-02-spec-local-inspection-vs-remote-divination.md) | Spec Local Inspection vs Remote Divination access rules & UI flow | `wayfinder:grilling` | Closed | None |
| [WF-03](./tickets/WF-03-formula-deduction-power-vs-concealment.md) | Formula design for Deduction Power vs Concealment Defense & Backfire Thresholds | `wayfinder:grilling` | Closed | None |
| [WF-04](./tickets/WF-04-spec-tracking-marks-and-sever-clues.md) | Spec Tracking Marks lifecycle, location tracking, and Sever Clues mechanisms | `wayfinder:grilling` | Closed | None |
| [WF-05](./tickets/WF-05-spec-gu-resonance-and-dao-detection.md) | Spec Gu Resonance & World Dao Mark Detection across Mortal vs Immortal | `wayfinder:grilling` | Closed | None |
| [WF-06](./tickets/WF-06-db-schema-and-api-routes.md) | Database Schema & API Routes specification for Deduction & Tracking | `wayfinder:task` | Closed | None |

---

## Not yet specified (Fog of War)

- **Cơ chế Hỏa Mù / Ngụy Tạo Thiên Cơ Chủ Động (Active Misdirection)**: Cho phép cao thủ Trí Đạo tạo thông tin sai lệch để khi kẻ khác suy diễn sẽ nhận được tọa độ giả hoặc danh tính giả thay vì chỉ che giấu thụ động.
- **Sát Chiêu Trí Đạo Kết Hợp (Killer Moves for Divination & Severing)**: Định hình danh sách các Sát Chiêu Tiên cấp mẫu (ví dụ: *Vạn Tinh Suy Diễn*, *Thiên Cơ Nhất Niệm*, *Tuệ Kiếm Đoán Tình/Đoạn Niệm*).
- **Cơ chế Tổ Đội & Tông Môn Bảo Hộ (Sect & Party Concealment Arrays)**: Các đại trận bảo hộ tông môn hoặc sát chiêu phòng hộ tổ đội giúp che chắn toàn bộ thành viên khỏi suy diễn ngoại lai.

---

## Out of scope

- Hệ thống PvP Arena tự động (đã có cơ chế Arena riêng biệt không phụ thuộc vị trí bản đồ).
- Đột nhập Tiên Khiếu người chơi khác (nằm trong roadmap Tiên Khiếu Động Thiên riêng).
