# [WF-REF-07] UI/UX phiên luyện, HUD chạy nền & realtime event

**Parent Map:** [[WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Label:** `wayfinder:grilling`
**Status:** Closed
**Assignee:** Agent
**Blocked by:** WF-REF-04, WF-REF-05

---

## Question

Chốt trải nghiệm người chơi từ lúc chọn công thức tới lúc nhận Cổ, gồm cả lúc rời màn hình.

### 1. Luồng vào phiên
- Trong `RefiningFurnace` hiện tại: chọn recipe → slot Cổ → bấm "Bắt đầu luyện" (thay nút "Hợp luyện").
- Có màn hình xác nhận cam kết tài nguyên (mana/đá/nguyên liệu/Cổ bị khóa) trước khi start không?
- Có hiện dải tỉ lệ dự kiến + bonus tối đa từ accuracy trước khi start không?
- Cổ rank thấp vẫn có nút "Luyện nhanh" tức thời; UI phân biệt thế nào để người chơi không nhầm?

### 2. Màn hình phiên
- Layout: full-screen overlay trong panel (như kết quả hiện tại) hay modal riêng?
- Dùng `GameStepProgress` cho timeline, `GameCircularProgress` cho đếm ngược bước, `GameProgressBar` cho tiến độ tổng.
- Hiển thị: bước hiện tại, loại minigame, cửa sổ còn lại, điểm từng bước, điểm chính xác tạm, tỉ lệ dự kiến cập nhật.
- Có hiện "đáp án đúng" sau mỗi bước không?
- Bước bỏ lỡ hiển thị thế nào (mờ, gạch, ghi 0 điểm)?

### 3. Minigame component
- Mỗi dạng là một component DOM/SVG riêng, nhận config từ server + callback submit.
- Có animation thắng/thua ngắn (0.3–0.8s) rồi tự chuyển bước, hay chờ người chơi bấm tiếp?
- Xử lý mất mạng khi đang trong bước: khóa input, hiện "đang kết nối lại", cho retry cùng nonce?
- Mobile/touch: kích thước nút, tránh double-tap, xoay ngang/dọc.

### 4. Rời màn hình & chạy nền
- Khi đóng panel: hiện HUD nhỏ (góc màn hình) với đếm ngược + số bước còn lại, hay chỉ dựa vào notification?
- Có cảnh báo khi sắp hết cửa sổ bước (âm thanh/toast)?
- Quay lại UI: tự mở lại phiên đang chạy, hay hiện nút "Tiếp tục luyện"?
- Nếu phiên đã resolve trong lúc vắng: hiện modal kết quả ngay khi mở game, hay chỉ notification + badge?

### 5. Kết quả & claim
- Success: tái dùng overlay gacha hiện tại (`RefiningFurnace.tsx:1189-1297`) nhưng thêm điểm chính xác, bonus, thời gian hoàn thành.
- Failure/failsafe/backfire: tái dùng overlay hiện tại, thêm lý do (bỏ lỡ bước, chết, điểm thấp).
- Claim thủ công hay tự động khi mở kết quả? Nếu thủ công, nút "Nhận Cổ" rõ ràng.
- Có log lịch sử phiên (thời điểm, điểm từng bước, kết quả) trong UI không?

### 6. Realtime
- Chốt event type + channel (phụ thuộc WF-REF-05). Phải thêm nhánh xử lý trong `useGameStore.subscribeEvents` trước fallback snapshot.
- Payload cập nhật bước có được phép làm thay đổi state cục bộ (optimistic) không? Đề xuất: không, chỉ áp dụng khi server trả.
- Reconnect: sau khi mất kết nối, gọi `GET /sessions/active` để đồng bộ.
- Notification hoàn tất dùng `notify` sẵn có (`notif:<uuid>`) — có cần loại notification riêng trong `NotificationBell` không?

### 7. Trạng thái lỗi
- Phiên đã bị resolve/abort bởi worker khi client đang mở.
- Submit bước trả 409 (sai bước/nonce) hoặc 410 (hết hạn).
- Chết trong lúc phiên chạy: hiện banner "Tâm thần chấn động, tỉ lệ giảm X%".
- Không đủ tài nguyên khi start (đã bị trừ bởi hành động khác).

### 8. Phong cách
- Có dùng pixel-art/furnace animation hiện có không? Có thêm âm thanh không?
- Có chế độ giảm chuyển động (prefers-reduced-motion) không?

---

## Resolution

Đã chốt đặc tả tại [WF-REF-07 Spec](../specs/WF-REF-07-ui-realtime.md). Tóm tắt:

- **Phân chế độ:** recipe có `target_rank >= 6 || target_rank >= player.rank` → "Bắt đầu phiên luyện"; còn lại → "Luyện nhanh" tức thời. Badge "Cần thời gian" trong danh sách.
- **Confirm modal:** hiện mana/đá/nguyên liệu, Cổ bị khóa, thời lượng, số bước, bonus accuracy tối đa, cảnh báo chết −25%, dải tỉ lệ từ preview.
- **Panel phiên:** timeline `GameStepProgress` + minigame + đếm ngược server; mobile thu gọn; nút "Thu nhỏ" và "Hủy phiên".
- **4 minigame DOM/SVG:** Canh Nhịp (rAF + Space), Dãy Ấn (hiện/nhập + phím 1–4), Chọn Hành Động (phím 1–3), Ghép Liệu (click xếp thứ tự, không kéo-thả).
- **Đồng hồ server:** `serverOffsetMs` từ mỗi response; countdown 250ms; marker rAF; refetch khi tab quay lại.
- **HUD nổi:** mount toàn cục, hiện bước/đếm ngược/trạng thái/kết quả chưa ack; bấm mở view `refine`.
- **Kết quả:** tái dùng overlay gacha hiện có, thêm điểm chính xác + breakdown mới; không auto-popup khi đang ở view khác; notification + toast + badge.
- **Store:** slice mới `refineSessionSlice` (không nhồi `cultivationSlice`), 6 action; `submitRefineStep` không dùng `beginAction` toàn cục; không optimistic.
- **Realtime:** subscribe kênh `"refine"`, event `refine.session`; reconnect gọi `GET /sessions/active`.
- **Xóa giả lập:** bỏ `isRefining`/`furnaceStage`/`stages`/`setInterval`/`setTimeout` trong `RefiningFurnace.tsx`.
- **v1 không làm:** lịch sử/replay, âm thanh, Phaser, kéo-thả, auto-refine.

