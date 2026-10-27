# [WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước

**Status:** Charting (Frontier Open)
**Label:** `wayfinder:map`
**Tracker:** Local Markdown (`docs/wayfinder/`)

---

## Destination

Biến **Luyện Cổ** từ một `POST /api/refine` tức thời thành một **phiên luyện chạy nền theo thời gian**, chia nhiều bước, mỗi bước là một minigame hoặc lựa chọn hành động được **server chấm điểm**; kết quả cuối vẫn dùng công thức tỉ lệ hiện có, cộng thêm lớp **điểm chính xác** do người chơi tạo ra.

Đích đến là một **spec đầy đủ** (state machine, schema, API, công thức, bộ minigame, UI/realtime, cấu hình data-driven) đủ để agent implement mà không cần quyết định thiết kế nào nữa, kèm một **prototype** nhỏ để chốt cảm giác chơi trước khi đóng băng số liệu.

---

## Notes

- **Domain:** Luyện Cổ / Refining, phiên chạy nền, minigame, kinh tế Cổ trùng.
- **Skills:** `domain-modeling`, `grilling`, `prototype`, `tdd`, `code-review`.
- **References:**
  - `backend/internal/handlers/refine_core.go`, `refine_gu.go`, `refine_deduce.go`
  - `backend/internal/db/schema_dictionaries.go` (`refining_recipes`)
  - `backend/internal/handlers/instance.go` (state machine nhiều bước mẫu)
  - `backend/internal/handlers/tribulation_worker.go`, `tribulation_trigger.go` (deadline + auto-resolve)
  - `backend/internal/worker/runner.go` (`RunEvery`/`SafeRun`)
  - `frontend/src/components/RefiningFurnace.tsx`
  - `frontend/src/components/ui/GameProgressBar.tsx` (`GameStepProgress`, variant `refinement`)
  - `frontend/src/store/slices/cultivationSlice.ts`, `frontend/src/store/useGameStore.ts`
  - `docs/wayfinder/research/WF-REF-02-findings.md`
- **Standing Preferences:**
  - Logic gameplay nằm trong `backend/internal/handlers/`; không thêm service layer mới nếu không cần.
  - **Server-authoritative tuyệt đối**: client chỉ gửi input rời rạc + timestamp; server giữ seed, deadline và tự tính điểm/tỉ lệ. Không nhận `final_rate`/`success` từ client.
  - **Không giữ lock `players` qua thời gian chờ.** Lock row session trong từng transaction ngắn; chỉ lock player khi finalize.
  - Một người chơi chỉ có **một phiên active** (partial unique index).
  - Worker quét phiên hết hạn qua `worker.RunEvery`; thêm lazy-resolve khi mở UI.
  - Cấu hình nhịp độ **data-driven** (CSV + Admin import), không hardcode trong handler.
  - Schema mới khai báo inline trong `CREATE TABLE` của `schema_*.go`; không dùng `ALTER TABLE` patch.
  - Realtime dùng **WebSocket**, không phải SSE; event mới phải có nhánh xử lý tường minh trong `useGameStore`.
  - Cập nhật `docs/API.md`, `docs/GAME_DATA.md`, `WIKI.md` khi chạm route/schema/gameplay.

---

## Decisions so far

- [[WF-REF-01] Chốt mô hình phiên luyện cổ: tương tác, thời gian, phạm vi, cam kết tài nguyên](./tickets/WF-REF-01-chot-mo-hinh-phien-luyen-co.md): Chốt **hybrid** (bước bắt buộc + minigame ngắn, tăng theo rank); phiên **chạy nền tính từ thời điểm bắt đầu**; **trừ tài nguyên ngay khi start**; bỏ lỡ bước chỉ mất phần cộng từ chính xác; người chơi vẫn làm mọi thứ, chết thì **giảm tỉ lệ thành công**; server-authoritative; cổ rank ≥ rank người chơi và toàn bộ Tiên Cổ bắt buộc qua phiên, cổ thấp hơn đi đường nhanh.
- [[WF-REF-02] Khảo sát luồng refine hiện tại và các pattern session/timer có thể tái dùng](./tickets/WF-REF-02-khao-sat-refine-va-pattern-session.md): Refine hiện là một transaction roll-immediate, giữ `FOR UPDATE` trên `players`; repo đã có state machine nhiều bước (`instance`), deadline worker (`tribulation`), và queue có lock chéo (`variant_refinement`) để tái dùng. Chưa có minigame nào; UI đã có `GameStepProgress`/`GameProgressBar variant="refinement"`. Chi tiết tại [WF-REF-02 Findings](./research/WF-REF-02-findings.md).
- [[WF-REF-03] Công thức Điểm Chính Xác → Tỉ Lệ Thành Công, hình phạt chết & thưởng Chân Ý](./tickets/WF-REF-03-cong-thuc-diem-chinh-xac-va-hinh-phat.md): Accuracy = trung bình cộng đều, chỉ cộng dương qua `MaxBonus(rank) × clamp((acc−50)/50,0,1)`; trần **giảm dần theo rank** (R1–2 +8% → R8–9 +4%) vì accuracy chỉ là thành phần phụ sau công thức gốc + Cảnh giới Luyện Đạo + Cảnh giới Cổ đích + Đạo ngân. Chết trừ thẳng 25 điểm phần trăm, không stack; mastery ×(1+acc/100); preview trả `rate_min`/`rate_max`; kết quả final vẫn show đủ breakdown. Chi tiết tại [WF-REF-03 Spec](./specs/WF-REF-03-accuracy-formula.md).
- [[WF-REF-04] Đặc tả bộ minigame v1: luật chơi, chấm điểm, seed & chống replay](./tickets/WF-REF-04-spec-minigame-v1.md): Chốt 4 dạng `TAP`/`SEQUENCE`/`CHOICE`/`ORDER`; bước 1 `CHOICE`, bước cuối `TAP`, giữa trộn theo `N`/`K` không lặp liền kề; lịch cố định từ `started_at` (`gap = duration/N`), bỏ lỡ = 0 điểm; công thức chấm riêng từng dạng; bảng độ khó theo rank; `RefineDecision.csv` generic; input rời rạc + nonce idempotent + grace 1s; chấp nhận rủi ro bot vì accuracy chỉ là phần phụ. Chi tiết tại [WF-REF-04 Spec](./specs/WF-REF-04-minigames-v1.md).
- [[WF-REF-05] Schema `refine_sessions`, state machine, API contract & worker resolve](./tickets/WF-REF-05-schema-state-machine-api.md): Bảng `refine_sessions` mới + `schema_refine.go`; state machine `ACTIVE → RESOLVED|ABORTED` (resolve áp dụng hiệu ứng ngay, `ack` chỉ là cờ UI); tạo phiên escrow tài nguyên + lock Cổ bằng `player_gu.is_locked`; submit bước chỉ lock row session; resolve lock player trong tx ngắn, re-check Tiên Cổ, savepoint chống race; hook chết trong `persistDeathDrop`; API 7 endpoint; realtime kênh `refine`; worker 30s + lazy-resolve; quick path giữ nguyên với guard `REFINE_SESSION_REQUIRED`. Chi tiết tại [WF-REF-05 Spec](./specs/WF-REF-05-session-architecture.md).
- [[WF-REF-06] Cấu hình nhịp độ theo rank, dữ liệu CSV & Admin import](./tickets/WF-REF-06-cau-hinh-nhip-do-data-driven.md): Bảng `refine_stage_templates` (rank 1–9) chứa duration/step/window, `accuracy_max_bonus`, `death_penalty`, `mastery_multiplier_max`, độ khó 4 minigame và `minigame_weights` JSONB; baseline seed bằng Go, chỉnh qua `RefineStage.csv` + Admin import (dry-run/diff/export, không cần restart); preflight chặn thiếu rank, cửa sổ chồng bước, JSON sai; snapshot cấu hình vào `rate_snapshot` để import giữa phiên không đổi kết quả. Chi tiết tại [WF-REF-06 Spec](./specs/WF-REF-06-stage-config.md).
- [[WF-REF-07] UI/UX phiên luyện, HUD chạy nền & realtime event](./tickets/WF-REF-07-ui-va-realtime.md): Phân chế độ "Luyện nhanh" / "Bắt đầu phiên luyện" theo rank; confirm modal hiện escrow + dải tỉ lệ; panel timeline + 4 minigame DOM/SVG; HUD nổi toàn cục; overlay kết quả tái dùng gacha hiện có; slice `refineSessionSlice` mới; subscribe kênh WS `"refine"`; xóa toàn bộ stage giả lập trong `RefiningFurnace`. Chi tiết tại [WF-REF-07 Spec](./specs/WF-REF-07-ui-realtime.md).

---

## Tickets

| ID | Title | Type | Status | Blocked By |
|---|---|---|---|---|
| [WF-REF-01](./tickets/WF-REF-01-chot-mo-hinh-phien-luyen-co.md) | Chốt mô hình phiên luyện cổ: tương tác, thời gian, phạm vi, cam kết tài nguyên | `wayfinder:grilling` | Closed | None |
| [WF-REF-02](./tickets/WF-REF-02-khao-sat-refine-va-pattern-session.md) | Khảo sát luồng refine hiện tại và các pattern session/timer có thể tái dùng | `wayfinder:research` | Closed | None |
| [WF-REF-03](./tickets/WF-REF-03-cong-thuc-diem-chinh-xac-va-hinh-phat.md) | Công thức Điểm Chính Xác → Tỉ Lệ Thành Công, hình phạt chết & thưởng Chân Ý | `wayfinder:grilling` | Closed | None |
| [WF-REF-04](./tickets/WF-REF-04-spec-minigame-v1.md) | Đặc tả bộ minigame v1: luật chơi, chấm điểm, seed & chống replay | `wayfinder:grilling` | Closed | None |
| [WF-REF-05](./tickets/WF-REF-05-schema-state-machine-api.md) | Schema `refine_sessions`, state machine, API contract & worker resolve | `wayfinder:task` | Closed | WF-REF-03, WF-REF-04 |
| [WF-REF-06](./tickets/WF-REF-06-cau-hinh-nhip-do-data-driven.md) | Cấu hình nhịp độ theo rank, dữ liệu CSV & Admin import | `wayfinder:task` | Closed | WF-REF-03 |
| [WF-REF-07](./tickets/WF-REF-07-ui-va-realtime.md) | UI/UX phiên luyện, HUD chạy nền & realtime event | `wayfinder:grilling` | Closed | WF-REF-04, WF-REF-05 |
| [WF-REF-08](./tickets/WF-REF-08-prototype-minigame.md) | Prototype một minigame + vòng bước rank 1 | `wayfinder:prototype` | Open | WF-REF-04 |
| [WF-REF-09](./tickets/WF-REF-09-duong-nhanh-va-tuong-thich.md) | Kinh tế đường nhanh (quick path) & tương thích API cũ | `wayfinder:grilling` | Open | WF-REF-03 |

**Frontier hiện tại:** WF-REF-08 (prototype, HITL) và WF-REF-09 (quick path/kinh tế). Chỉ còn hai ticket trước khi map đạt đích spec.

---

## Not yet specified (Fog of War)

- **Minigame v1.1 — Hỏa Hậu & Phản Xạ:** cần tick liên tục/nhiều event hơn; chỉ spec sau khi đo cảm giác của 4 dạng v1.
- **Can thiệp PvP vào lò luyện:** đối thủ phá lò, cướp nguyên liệu, hoặc tăng rủi ro khi luyện ở vùng nguy hiểm.
- **Hỗ trợ Tông Môn / Tổ Đội:** nhiều người góp sức vào một phiên, chia thưởng và rủi ro.
- **Lịch sử & replay phiên:** xem lại từng bước, điểm số, seed để minh bạch chống gian lận.
- **Chống bot bằng phân tích nhịp input:** phát hiện input quá đều/tự động; chỉ cần khi có dấu hiệu lạm dụng.
- **Ảnh hưởng môi trường lên từng bước:** thời tiết, độ no, thương tích, vùng đạo làm thay đổi cửa sổ/dung sai.
- **Phiên luyện cho NPC dị nhân / hàng đợi Tiên Khiếu:** có thể dùng chung state machine nhưng thuộc hệ thống khác.

---

## Out of scope

- Viết lại toàn bộ công thức tỉ lệ Luyện Cổ hiện có — map này chỉ **cộng thêm lớp điểm chính xác**, không cân bằng lại các bonus cũ.
- Minigame cho **suy diễn công thức** (`/api/refine/deduce`) — chỉ áp dụng cho hành động luyện.
- Minigame cho các nghề khác (khai khoáng, trồng trọt, luyện đan) — có thể tái dùng hạ tầng sau, nhưng không thuộc đích này.
- Minigame 3D/Phaser trong thế giới MMO — v1 chỉ dùng DOM/SVG.
- Tự động hóa luyện cổ (auto-refine) cho người chơi — đi ngược mục tiêu "cần thời gian và thao tác".
