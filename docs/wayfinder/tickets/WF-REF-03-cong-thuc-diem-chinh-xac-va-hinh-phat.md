# [WF-REF-03] Công thức Điểm Chính Xác → Tỉ Lệ Thành Công, hình phạt chết & thưởng Chân Ý

**Parent Map:** [[WAYFINDER MAP] Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md)
**Label:** `wayfinder:grilling`
**Status:** Closed
**Assignee:** Agent
**Blocked by:** None

---

## Question

Chốt lớp công thức nối phiên luyện với hệ thống tỉ lệ hiện có. Đây là ticket HITL: mỗi con số dưới đây phải được người thiết kế game xác nhận, không tự chốt.

### 1. Quy đổi điểm bước → điểm chính xác
- Mỗi bước cho 0–100 điểm. Trung bình **có trọng số** theo rank bước hay trung bình cộng?
- Bước bỏ lỡ = 0 điểm (đã chốt). Có **phạt thêm** vì bỏ lỡ (ví dụ trừ 5 điểm các bước khác) hay chỉ mất phần dương?
- Điểm chính xác có làm tròn/hiển thị thế nào trước khi resolve?

### 2. Bonus tỉ lệ từ điểm chính xác
Đề xuất dạng tuyến tính có trần, chỉ cộng dương:

```
accuracyBonus = maxBonus(rank) × clamp((accuracy - 50) / 50, 0, 1)
```

| Rank cổ | maxBonus |
|---|---|
| 1–2 | +8% |
| 3–4 | +12% |
| 5 | +15% |
| 6–7 | +18% |
| 8–9 | +20% |

- Ngưỡng 50 điểm có đúng ý "chơi dở không bị phạt, chơi tốt mới được thưởng" không?
- Bonus này cộng **trước** hay **sau** bước chặn trần 100% và trừ debuff `refine_fatigue`/`refine_backfire`? Đề xuất: cộng vào `sumPositiveBonus` trước khi cap 100%, giống các bonus dương hiện tại.
- Có áp dụng `ImmortalPenalty` cho bonus này không? Đề xuất: có, giống các bonus dương khác.

### 3. Hình phạt chết
Người chơi đã chốt: chết khi phiên đang chạy thì **giảm tỉ lệ thành công**, phiên vẫn tiếp tục. Cần chốt:
- Mức giảm: **-15% / -25% / -35%**? Đề xuất **-25%** cho mọi rank.
- Có **stack** khi chết nhiều lần trong cùng phiên không? Đề xuất: không stack, chỉ áp dụng một lần.
- Áp dụng vào `finalSuccessRate` sau cap 100% (giống debuff) và có thể đẩy tỉ lệ xuống 0.
- Có giảm luôn điểm chính xác/khóa bước tiếp theo không? Đề xuất: không, chỉ trừ tỉ lệ cuối.
- Nếu chết đúng lúc đang resolve thì xử lý thế nào (ưu tiên trạng thái nào)?

### 4. Thưởng Chân Ý Luyện Đạo
- Hiện success được `calculateRefinementPathGain(rank, realm)`; đề xuất nhân thêm `(1 + accuracy/100)` (tối đa ×2).
- Có cộng Chân Ý khi thất bại nhưng chơi giỏi không? Đề xuất: không, giữ nguyên luật hiện tại (chỉ success).
- Điểm chính xác có ảnh hưởng pity `insight_<rank>` không? Đề xuất: không, giữ nguyên +1 stack khi fail / reset khi success.

### 5. Preview tỉ lệ
- Có endpoint/response trả **khoảng tỉ lệ dự kiến** (base + bonus tối đa) trước khi start không?
- Nếu có, có lộ chính xác công thức không, hay chỉ hiện dải?

### 6. Tương tác với trạng thái hiện có
- `refine_fatigue` / `refine_backfire` vẫn áp dụng bình thường?
- Pity vẫn tính theo lần resolve, không phải theo bước?
- Có trường hợp nào accuracyBonus bị vô hiệu (ví dụ Tiên Cổ đã tồn tại, nguyên liệu sai công thức)?

---

## Resolution

Đã chốt toàn bộ công thức tại [WF-REF-03 Spec](../specs/WF-REF-03-accuracy-formula.md). Tóm tắt:

- **Accuracy = trung bình cộng đều** điểm 0–100 của $N$ bước; bước bỏ lỡ = 0, không phạt thêm.
- **Bonus chỉ cộng dương**, dạng `MaxBonus(rank) × clamp((acc−50)/50, 0, 1)`; dưới 50 điểm = 0.
- **Trần bonus giảm dần theo rank**: R1–2 +8%, R3–4 +7%, R5 +6%, R6–7 +5%, R8–9 +4% — accuracy là thành phần phụ, càng late-game càng nhỏ.
- Bonus cộng vào `sumPositiveBonus` **trước** cap 100%, chịu `ImmortalPenalty`; `refine_fatigue`/`refine_backfire` vẫn trừ sau cap.
- **Chết**: trừ thẳng **25 điểm phần trăm** vào tỉ lệ cuối, sau cap, trước sàn 0; **không stack**; phiên tiếp tục.
- **Chân Ý Luyện Đạo** khi thành công: `gain × (1 + accuracy/100)`, tối đa ×2; thất bại không cộng.
- **Pity không đổi**: fail hợp lệ +1 stack, success reset.
- **Vô hiệu accuracyBonus** khi sai nguyên liệu hoặc Tiên Cổ mục tiêu đã tồn tại.
- **Preview** `GET /api/refine/preview` trả breakdown hiện có + `rate_min`/`rate_max` + `accuracy_max_bonus`; không lộ seed.
- **Kết quả final hiển thị đủ breakdown**, bổ sung `accuracy`, `accuracy_bonus`, `accuracy_max_bonus`, `death_penalty`, `step_scores`; không đổi trường cũ.

