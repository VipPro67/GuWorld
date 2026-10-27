# [WF-REF-02] Khảo sát luồng refine hiện tại và các pattern session/timer có thể tái dùng

**Parent Map:** [[WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Label:** `wayfinder:research`
**Status:** Closed
**Blocked by:** None

---

## Question

Khảo sát codebase để trả lời:

1. Luồng `POST /api/refine` hiện tại: route, handler, transaction, thời điểm trừ tài nguyên, RNG, publish realtime.
2. Có pattern session/timer/progress nào trong backend có thể tái dùng cho phiên luyện nhiều bước không?
3. Đã có minigame / multi-step / staged interaction nào chưa?
4. Data model công thức luyện (`refining_recipes`) có gì dùng được cho cấu hình bước?
5. Có cơ chế resolve phiên khi login/offline không?
6. Pattern lock hiện tại của refine có rủi ro gì nếu phiên kéo dài?
7. Frontend: luồng `RefiningFurnace`, store, realtime WS, UI pattern cho tiến trình nhiều bước.

---

## Resolution

Đã khảo sát đầy đủ, chi tiết tại [WF-REF-02 Findings](../research/WF-REF-02-findings.md). Tóm tắt:

- `POST /api/refine` (`backend/internal/handlers/refine_gu.go:24+`) là **một transaction duy nhất**: validate → trừ mana/đá → roll `rand.Float64()` → trừ nguyên liệu theo outcome → ghi inventory + player → commit. Giữ `SELECT … FOR UPDATE` trên row `players` suốt request (`evaluateManaTx`).
- **Ba pattern tái dùng được:** state machine nhiều bước của `player_instance_sessions` (`current_stage`, `option_index`); deadline + worker auto-resolve của tribulation (`started_at`, `NOW()`); lock chéo `FOR UPDATE OF … SKIP LOCKED` + advisory lock của `variant_refinement`.
- **Chưa có minigame/timing nào** trong repo. Gần nhất là stage UI giả lập trong `RefiningFurnace.tsx:290-313` (600ms label + `setTimeout` 2.5s rồi gọi API một lần, còn leak interval khi unmount).
- **Không có resolve phiên khi login**; mọi deadline đều do worker quét toàn cục. Phiên bỏ dở cần worker sweep + lazy-resolve.
- **Không được giữ lock `players` qua thời gian chờ** — sẽ chặn login/combat/movement của chính người chơi. Chỉ lock row session trong từng transaction ngắn, lock player khi finalize.
- Frontend đã có `GameStepProgress`, `GameProgressBar variant="refinement"`, `GameCircularProgress`; realtime là **WebSocket** (`wsManager.subscribe`), kênh `"player"` sẽ bỏ qua event lạ không có `id`/`patch`, nên cần nhánh xử lý tường minh hoặc kênh riêng.
- `refining_recipes` chỉ có `success_rate`, chi phí, required Gus/materials — **không có duration/stage config**; cần bảng/cột mới theo WF-REF-06.

## Hệ quả

- WF-REF-05 nên tái dùng mẫu `player_instance_sessions` + tribulation worker, không phát minh state machine mới.
- WF-REF-07 phải chỉ rõ event realtime mới và nhánh xử lý trong `useGameStore`.
