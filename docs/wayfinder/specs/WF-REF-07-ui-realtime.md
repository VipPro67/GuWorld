# [SPECIFICATION] UI/UX Phiên Luyện Cổ, HUD Chạy Nền & Realtime

**Ticket:** [WF-REF-07: UI/UX phiên luyện, HUD chạy nền & realtime event](../tickets/WF-REF-07-ui-va-realtime.md)
**Map:** [Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Ngày hoàn thành:** 2026-10-27

---

## 1. Mục tiêu

Thay đoạn giả lập "đang luyện" hiện tại bằng một phiên thật:

- Phân biệt rõ **Luyện nhanh** và **Bắt đầu phiên luyện**.
- Trong phiên: timeline bước + minigame + đồng hồ server.
- Rời màn hình vẫn theo dõi qua HUD nổi.
- Kết quả đến qua notification/toast/badge, không cắt ngang combat.
- Không optimistic update; server là nguồn sự thật.

---

## 2. Hiện trạng cần thay

`frontend/src/components/RefiningFurnace.tsx`:

- `isRefining`, `furnaceStage`, mảng `stages`, `setInterval(600ms)` + `setTimeout(2500ms)` (`:283-348`) là **giả lập**, không có cleanup khi unmount.
- `onRefine` gọi `POST /api/refine` một lần; kết quả render overlay `:1184-1420`.
- Breakdown tỉ lệ chỉ hiện sau kết quả (`:494-580`).
- Nút hiện tại luôn là "Hợp luyện", không phân biệt đường nhanh/phiên.

**Việc phải làm:** xóa toàn bộ đoạn giả lập và state liên quan; thay bằng state phiên thật từ store. Không giữ `setTimeout`/`setInterval` nào không cleanup.

---

## 3. Phân biệt đường nhanh vs phiên

### 3.1 Điều kiện
```ts
const requiresSession =
  recipe.target_rank >= 6 || recipe.target_rank >= player.rank;
```

- `requiresSession = false`: nút **"Luyện nhanh"** → gọi `refineGu` hiện tại.
- `requiresSession = true`: nút **"Bắt đầu phiên luyện"** → mở confirm modal.
- Danh sách recipe hiện badge **"Cần thời gian"** cho recipe `requiresSession`, kèm icon đồng hồ.

### 3.2 Preview
- Khi chọn recipe, gọi `fetchRefinePreview(recipe.id)` (debounce 300ms).
- Hiển thị dải `rate_min% – rate_max%`, `accuracy_max_bonus`, cảnh báo thiếu tài nguyên.
- Nếu preview lỗi: ẩn dải, không chặn thao tác; hiện tooltip lỗi.

### 3.3 Confirm modal
Dùng `GameModal` (không dùng `GameConfirmModal` vì cần nội dung dài):

- Tên Cổ đích + rank.
- Thời lượng, số bước, số minigame.
- Chi phí: mana, Nguyên Thạch, Tiên Nguyên Thạch, nguyên liệu (số lượng).
- Danh sách Cổ nguyên liệu sẽ bị khóa.
- Bonus accuracy tối đa theo rank.
- Cảnh báo: "Chết trong lúc luyện sẽ giảm 25% tỉ lệ thành công."
- Nút **"Bắt đầu"** (disabled nếu thiếu tài nguyên) / **"Hủy"**.
- Focus trap, `Esc` đóng.

---

## 4. Bố cục panel phiên

`RefineSessionPanel` thay nội dung `RefiningFurnace` khi có phiên `ACTIVE` hoặc kết quả chưa `ack`.

```
┌──────────────────────────────────────────────────────────┐
│ [Cổ đích R3]  Trạng thái: Đang luyện      Còn 04:12      │
│ Tỉ lệ hiện tại: 62% → 69% (accuracy 82.5)                │
├───────────────────┬──────────────────────────────────────┤
│ GameStepProgress  │  Bước 2/4 · Dãy Ấn                   │
│ ✓ 1 Chọn Hành Động│  ┌────────────────────────────────┐  │
│ ● 2 Dãy Ấn       │  │      minigame component        │  │
│ ○ 3 Ghép Liệu    │  └────────────────────────────────┘  │
│ ○ 4 Khai Lò      │  Cửa sổ còn: 00:06                    │
│                   │  Điểm tạm: 62.5                       │
├───────────────────┴──────────────────────────────────────┤
│ [Thu nhỏ]                         [Hủy phiên]             │
└──────────────────────────────────────────────────────────┘
```

- Mobile: timeline thu gọn thành hàng ngang hoặc accordion; minigame full width.
- Header đếm ngược `ends_at`.
- Banner đỏ khi `death_penalty > 0`: "Tâm thần chấn động — tỉ lệ −25%."
- Khi không có bước mở:
  - Chưa tới bước kế: "Bước kế mở sau mm:ss".
  - Đã qua bước cuối, chờ resolve: "Đang ủ lò…".
- Nút "Thu nhỏ" chỉ ẩn panel; HUD vẫn hiện.
- Nút "Hủy phiên" mở confirm hiển thị rõ: mất mana/đá, mất 50% nguyên liệu, Cổ về Hấp Hối.

---

## 5. Bốn minigame component

Thư mục: `frontend/src/components/refine/games/`

### 5.1 Props chung

```ts
interface RefineGameProps {
  step: PublicRefineStep;
  serverOffsetMs: number;   // server_now - Date.now()
  disabled: boolean;
  onSubmit: (action: RefineAction, payload: unknown) => void;
}
```

- Mỗi component chỉ gọi `onSubmit` **một lần**; sau đó tự disable.
- Không tự tính điểm; chỉ hiển thị và gửi input.

### 5.2 `RefineTapGame`
- Thanh ngang cao ≥ 48px, vùng ngọt tô sáng + nhãn "Vùng ngọt".
- Kim chạy bằng `requestAnimationFrame`, vị trí tính từ `now = Date.now() + serverOffsetMs`:

```ts
const t = (now - step.window_start_ms) % step.params.marker_period_ms;
const u = t / step.params.marker_period_ms;
const pos = 1 - Math.abs(2 * u - 1);
```

- Tap: `pointerdown` trên thanh hoặc `Space`/`Enter`.
- Sau tap: khóa, hiện vệt dừng; chờ server trả điểm.
- Nếu `now > window_end + grace`: disable, hiện "Đã bỏ lỡ bước".
- Không cho tap khi `disabled`.

### 5.3 `RefineSequenceGame`
- 4 nút ấn (biểu tượng + màu khác nhau, không chỉ màu).
- Pha hiện: lần lượt sáng từng ấn theo `display_ms`; có nút "Xem lại" **chỉ khi** còn thời gian và chưa nhập (tối đa 1 lần).
- Pha nhập: bấm theo thứ tự; hiện các ấn đã chọn; `Backspace` xóa ấn cuối.
- Tự submit khi đủ `L` ấn; hoặc nút "Xác nhận" nếu người chơi muốn nộp sớm.
- Phím 1–4.
- Hết giờ: disable + "Đã bỏ lỡ bước".

### 5.4 `RefineChoiceGame`
- Hiện `situation` + 2–3 nút option do server gửi (đã xáo).
- Bấm hoặc phím 1–3 → submit ngay.
- Không hiện đáp án đúng trước khi chấm.

### 5.5 `RefineOrderGame`
- Danh sách vật phẩm (icon, tên, rank pips).
- Bấm vật phẩm → thêm vào ô thứ tự kế tiếp; bấm lại ô đã xếp → bỏ về danh sách.
- Nút "Hoàn tác" (bỏ món cuối) và "Xếp lại" (xóa hết).
- Đủ `M` món → nút "Xác nhận" sáng; bấm hoặc `Enter` để submit.
- Không kéo-thả.

### 5.6 Sau submit
- Parent hiện chip điểm bước (`+100`, `+50`, `0`) trong ~1 giây.
- Nếu bước kế đã mở → tự chuyển.
- Nếu chưa → hiện "Bước kế mở sau X" và giữ timeline.

---

## 6. Đồng bộ đồng hồ server

- Mỗi response session trả `server_now`; store tính `serverOffsetMs = server_now - Date.now()`.
- Mọi đếm ngược/marker dùng `Date.now() + serverOffsetMs`, **không** dùng đồng hồ máy trực tiếp.
- Tick UI 250ms cho countdown; marker dùng `requestAnimationFrame`.
- Tab bị ẩn: rAF dừng; khi quay lại tính lại từ server time.
- Nếu `now > window_end`: hiện "Đã bỏ lỡ bước", gọi `fetchActiveRefineSession` để đồng bộ.
- Grace phía client: không gửi sau `window_end`; server vẫn là nơi quyết định.

---

## 7. HUD nổi

Component `RefineSessionHud`, mount toàn cục trong `app/page.tsx`.

- Chỉ hiện khi:
  - có phiên `ACTIVE`, hoặc
  - có kết quả `RESOLVED` chưa `ack`.
- Vị trí: góc phải dưới, trên dock mobile, `z-index` thấp hơn modal nhưng cao hơn panel.
- Nội dung:
  - `ACTIVE` + đang có bước mở: "Bước 2/4 · Dãy Ấn" + đếm ngược cửa sổ (pulse).
  - `ACTIVE` + chờ bước kế: "Bước kế sau 03:12".
  - `ACTIVE` + đang ủ lò: "Đang ủ lò…".
  - `RESOLVED` chưa ack: badge đỏ + "Có kết quả luyện cổ".
- Bấm HUD → `setCurrentView("refine")`.
- Mobile: thu gọn thành icon + badge; `safe-area-inset-bottom`.
- `prefers-reduced-motion`: bỏ pulse.

---

## 8. Kết quả & overlay

Tái dùng overlay hiện có trong `RefiningFurnace.tsx:1184-1420`:

- Success: giữ showcase gacha; thêm khối:
  - Điểm chính xác (0–100).
  - Bonus accuracy (+X%).
  - Tỉ lệ cuối.
  - Danh sách điểm từng bước.
- Failure/FailSafe/Backfire: giữ layout hiện tại; thêm lý do:
  - `reason = expired` → "Phiên kết thúc khi vắng mặt".
  - `death_penalty > 0` → "Tâm thần chấn động −25%".
  - Bước bị bỏ lỡ → liệt kê.
- Nút **"Đã hiểu"** → `ackRefineResult(sessionId)`.
- Sau ack: `activeRefineSession = null`, `lastRefineResult = null`; HUD ẩn.
- Không auto mở modal nếu người chơi đang ở view khác.

---

## 9. Store slice

Tạo mới `frontend/src/store/slices/refineSessionSlice.ts`, compose vào `useGameStore`.

### 9.1 State

```ts
interface RefineSessionState {
  activeRefineSession: RefineSession | null;
  lastRefineResult: RefineResult | null;
  refinePreview: RefinePreview | null;
  refineSessionLoading: boolean;
  refineStepLoading: boolean;
  refineServerOffsetMs: number;
}
```

### 9.2 Actions

| Action | Gọi API | Ghi chú |
|---|---|---|
| `fetchRefinePreview(recipeId)` | `GET /api/refine/preview` | debounce ở component |
| `startRefineSession(recipeId, guIds)` | `POST /api/refine/sessions` | `applyInventoryFromResponse`; set session |
| `fetchActiveRefineSession()` | `GET /api/refine/sessions/active` | lazy-resolve; set session/result |
| `submitRefineStep(sessionId, req)` | `POST .../steps` | lưu `lastStep`; nếu result → set |
| `abortRefineSession(sessionId)` | `POST .../abort` | `applyInventoryFromResponse` |
| `ackRefineResult(sessionId)` | `POST .../ack` | clear state |

### 9.3 Quy tắc
- `startRefineSession` dùng `beginAction("player:cultivation")` (tạo phiên là hành động nặng, giống refine hiện tại).
- `submitRefineStep` **không** dùng `beginAction` toàn cục; chỉ set `refineStepLoading` để không chặn combat/di chuyển.
- Không optimistic update.
- Lỗi: đọc `data.code`, map sang thông điệp tiếng Việt, `addToast` + `addLog`.
- 401/409/403 để `authFetch` xử lý; không thêm block `if (status === 401)`.
- Type `RefineRateBreakdown` mở rộng thêm `accuracy`, `accuracy_bonus`, `accuracy_max_bonus`, `death_penalty`, `step_scores`.

---

## 10. Realtime

- `useGameStore.subscribeEvents` thêm:

```ts
const unsubscribeRefine = wsManager.subscribe("refine", (payload) => {
  if (payload?.type !== "refine.session") return;
  // cập nhật activeRefineSession / lastRefineResult
});
```

- Payload server: `{type:"refine.session", revision, session}` (WF-REF-05 §10).
- Không áp dụng event nếu `session.player_id` khác player hiện tại (server đã scope, nhưng kiểm tra vẫn tốt).
- Khi `session.status === "RESOLVED"`:
  - set `lastRefineResult`.
  - `addToast` + `addLog`.
  - Nếu đang ở view `"refine"` → overlay tự hiện.
  - Nếu không → HUD badge.
- Reconnect: sau khi WS mở lại, gọi `fetchActiveRefineSession()`.
- `unsubscribeEvents` hủy subscription.
- Không thêm kênh `"player"` branch; dùng kênh `"refine"` riêng.

---

## 11. Tích hợp trang

- `app/page.tsx`:
  - Mount `<RefineSessionHud />` gần các overlay khác (sau `NotificationBell`/`GlobalChatOverlay`).
  - View `"refine"` vẫn render `RefiningFurnace`.
- `RefiningFurnace`:
  - Nếu `activeRefineSession` hoặc `lastRefineResult` → render `RefineSessionPanel` / overlay thay cho slotting UI.
  - Nếu không → giữ UI chọn recipe/slot hiện tại.
  - Nút chọn chế độ theo `requiresSession`.
  - Xóa `isRefining`, `furnaceStage`, `stages`, interval/timeout.
  - Slot Cổ bị khóa: hiện icon khóa + tooltip "Đang dùng trong phiên luyện".
- `currentView` không đổi; không thêm route mới.

---

## 12. Trạng thái lỗi & biên

| Tình huống | Hành vi |
|---|---|
| Bấm bắt đầu khi đã có phiên | Toast lỗi `REFINE_SESSION_ACTIVE`; `fetchActiveRefineSession()` để mở phiên cũ |
| Submit khi bước đã đóng | Disable input; hiện "Đã bỏ lỡ bước"; refetch |
| Submit lỗi mạng | Giữ payload + nonce; hiện "Thử lại" (gửi lại cùng nonce) |
| Server trả 410 | Chuyển bước, refetch |
| Phiên bị resolve ở tab khác | Event cập nhật; panel chuyển overlay kết quả |
| Phiên bị abort ở tab khác | Event cập nhật; panel về UI chọn recipe |
| Chết trong phiên | Banner −25%; cập nhật tỉ lệ hiển thị |
| Player hết mana/đá khi confirm | Nút "Bắt đầu" disabled + danh sách thiếu |
| Recipe < 100% | Giữ luồng deduce hiện tại; không cho bắt đầu phiên |
| Cổ đã bị khóa bởi phiên khác | Badge khóa trong slot; không cho chọn |
| Tab ẩn quá lâu | Khi quay lại refetch; nếu quá hạn hiện kết quả |

---

## 13. Khả năng tiếp cận & responsive

- Nút ≥ 44×44px trên mobile.
- `aria-label` cho thanh Canh Nhịp, các nút ấn, ô thứ tự.
- Focus ring rõ; modal trap focus; `Esc` đóng confirm.
- Phím tắt: `Space`/`Enter` cho Canh Nhịp, `1–4` cho Ấn/Lựa chọn, `Backspace` xóa, `Enter` xác nhận Ghép Liệu.
- Không chỉ dùng màu: vùng ngọt có nhãn + pattern; ấn có ký hiệu riêng.
- `prefers-reduced-motion`: tắt rung/pulse, giữ countdown dạng số.
- Mobile: panel dọc, timeline thu gọn; HUD không che dock.

---

## 14. File thay đổi

**Mới**
- `frontend/src/store/slices/refineSessionSlice.ts`
- `frontend/src/components/refine/RefineSessionPanel.tsx`
- `frontend/src/components/refine/RefineSessionHud.tsx`
- `frontend/src/components/refine/RefineConfirmModal.tsx`
- `frontend/src/components/refine/RefineResultOverlay.tsx`
- `frontend/src/components/refine/games/RefineTapGame.tsx`
- `frontend/src/components/refine/games/RefineSequenceGame.tsx`
- `frontend/src/components/refine/games/RefineChoiceGame.tsx`
- `frontend/src/components/refine/games/RefineOrderGame.tsx`
- `frontend/src/components/refine/types.ts`

**Sửa**
- `frontend/src/components/RefiningFurnace.tsx` — mode split, xóa giả lập, mount panel/confirm/result
- `frontend/src/app/page.tsx` — mount HUD
- `frontend/src/store/useGameStore.ts` — compose slice, subscribe `"refine"`, mở rộng type breakdown
- `frontend/src/components/NotificationBell.tsx` — icon cho loại `REFINE` (nếu server dùng)
- `frontend/src/app/globals.css` — animation minigame/reduced-motion (nếu cần)
- `frontend/README.md` — bảng component mới
- `WIKI.md` — mô tả phiên luyện cho người chơi

---

## 15. Kiểm thử

### Bắt buộc chạy
```bash
cd frontend && npm run lint
cd frontend && npm run build
```

### Kiểm tra thủ công
- Chọn recipe thấp → chỉ có "Luyện nhanh", kết quả như cũ.
- Chọn recipe ngang cấp → confirm hiện đúng chi phí/thời lượng; bắt đầu tạo phiên; tài nguyên trừ; Cổ khóa.
- Reload trang khi phiên đang chạy → panel khôi phục đúng bước/countdown.
- Mở tab thứ hai → cả hai đồng bộ qua WS; submit ở tab này, tab kia cập nhật.
- Ẩn tab qua cửa sổ bước → quay lại thấy "Đã bỏ lỡ bước", không crash.
- Bỏ lỡ toàn bộ → hết hạn, nhận notification + kết quả accuracy 0.
- Hủy phiên → confirm đúng mất gì; inventory cập nhật; Cổ về Hấp Hối.
- Chết trong phiên → banner −25%; kết quả phản ánh.
- Success → overlay gacha có điểm chính xác + breakdown mới; ack ẩn HUD.
- Mạng yếu: submit lỗi → thử lại cùng nonce không tạo double step.
- Mobile 375px: panel/HUD/minigame không tràn, nút đủ lớn.
- `prefers-reduced-motion`: không rung/pulse.

---

## 16. Ngoài phạm vi v1

- Trang lịch sử phiên & replay từng bước.
- Âm thanh riêng cho minigame.
- Minigame Phaser/canvas.
- Kéo-thả tự do.
- Tự động hóa luyện cổ.
- So sánh trực quan nhiều recipe cạnh nhau.
