# Luyện Cổ, Cổ phương và phiên luyện

## Tính năng làm gì?

Hệ thống có ba hoạt động khác nhau: **suy diễn** tăng độ hoàn chỉnh của recipe đã khám phá, **cảm ngộ** tạo tiến trình recipe mới, và **luyện Cổ** tiêu hao nguyên liệu để tạo Gu. Một số recipe dùng quick refine; target khác bắt buộc phải qua timed session có nhiều minigame step.

## Điều kiện mở và quyền truy cập

- `/refine/*` yêu cầu đăng nhập và onboarding.
- Quick refine chỉ chạy khi player còn sống, đã mở recipe đến 100%, đủ mana/currency/materials/Gu requirements và target rank thấp hơn player rank.
- Target rank >= 6 hoặc target rank >= player rank phải dùng phiên luyện.
- Chỉ có một phiên `ACTIVE` trên mỗi player. Một Gu không thể bị giữ đồng thời bởi phiên luyện khác hay active product.
- Deduce cần một partial recipe đã mở; recipe 100% không thể tiếp tục suy diễn.
- Conception không cho mortal tạo Immortal Gu recipe; recipe mục tiêu phải tồn tại trong Gu dictionary.

## Cách sử dụng

### Xem recipe/preview

1. Mở lò luyện và xem recipe đã sở hữu/đã biết.
2. Dùng `GET /refine/preview?recipe_id=...` để xem điều kiện, tài nguyên và rate band; preview chỉ là dự đoán, không khóa tài nguyên.
3. Nếu thiếu recipe, chọn suy diễn hoặc cảm ngộ tương ứng.

### Suy diễn recipe

Gửi recipe ID qua `/refine/deduce`. Server tiêu hao 15 Thought và 10 Primeval Stones cho mỗi attempt, rồi tăng completion dựa trên Wisdom/Refinement/target path mastery, Dao Marks và rank scaling. Gu có native mechanic instant deduction có thể hoàn thiện recipe trong attempt đó; chi phí vẫn đã trừ.

### Cảm ngộ recipe

Gửi target Gu/rank và catalyst tùy chọn qua `/refine/conceive`. Catalyst có thể là Gu sống hoặc material phù hợp. Nếu là Gu: không được vital, equipped, locked, dead, cao rank hơn target và phải cùng path chính. Nếu là item: phải là Material, rank không vượt target và path khớp. Server tính Thought/stone costs theo rank, backlash risk và accuracy từ Wisdom/Refinement mastery, Dao Marks, catalyst và realm.

### Quick refine

1. Chọn recipe completion 100%, selected Gu và materials.
2. Gọi `POST /refine`.
3. Server kiểm tra rank, recipe, currency, materials, Gu ownership/lock và tính kết quả. Response báo Gu tạo thành, nguyên liệu/Gu tiêu hao và outcome.

### Timed refine session

1. Tạo session qua `POST /refine/sessions` với recipe và Gu được chọn.
2. Lưu ID để có thể khôi phục bằng `/sessions/active` hoặc `/sessions/:id`.
3. Theo thời điểm mở step và `nonce` được response trả, chơi minigame rồi gửi `/sessions/:id/steps` đúng action type.
4. Bước lỡ hạn có trạng thái MISSED; bước chưa mở ở trạng thái LOCKED. Khi step cuối hoàn tất hoặc session hết hạn, server resolve kết quả.
5. Đọc result; `/sessions/:id/ack` chỉ đánh dấu UI đã xem, vì reward đã được áp dụng khi resolve.

## Luật, chi phí và giới hạn

### Quick refine: xác suất và mana

Tỉ lệ nền `baseRate` nằm trong recipe DB. Với mục tiêu mortal, các bonus cộng theo điểm phần trăm dạng số thập phân: Dao marks Thực Đạo (`0.00632*sqrt(marks)*(1+targetPathPoints)/RankScale(targetRank)`), realm refinement mastery `[0, .05, .10, .18, .25]`, target path mastery `[0, .02, .05, .08, .10]`, luck `min(.10, luckMarks*.001 + statusLuckBonus/100)`, vùng có path phù hợp `+.05`, mỗi rank vượt yêu cầu của Gu nguyên liệu `+.02`, insight stack `+.01` mỗi stack. Race/constitution refine bonus dương cộng vào; bonus âm là debuff. Tổng bonus dương cộng base được cap ở 1 trước khi trừ debuff; kết quả floor ở 0. Nếu không đủ Gu requirement thì chance bằng 0.

Với target rank >=6, các bonus phẳng (ngoại trừ Dao bonus) chia theo rank penalty: R6 2, R7 5, R8 20, R9 100. Backfire threshold là roll >=.90 với mortal, >=.75 với immortal. Vùng fail-safe có width `(1-finalRate)*.25*refinementRealm/4`; ngoài vùng success/fail-safe, roll vào backfire threshold sẽ backfire. Các vùng kết quả được xét theo thứ tự trong handler; không coi đây là một bảng tỷ lệ độc lập.

Mana cơ sở là 20 mortal hoặc 10 immortal. Cùng nhóm mortal/immortal, nếu player rank cao hơn target thì chia `10^(rankDiff)` cho mortal hoặc `100^(rankDiff)` cho immortal; nếu player thấp hơn target thì nhân `HigherRankManaMultiplier`. Mortal luyện target immortal bị chặn; immortal luyện target mortal có cost 0. Rank scaling và recipe data có thể đổi nên preview/API là số cuối cùng cần dùng.

### Timed refine: chấm điểm

- Tap: `100*max(0, 1 - abs(position-center)/halfWidth)`.
- Sequence: số ký hiệu đúng / độ dài chuỗi * 100. Chuỗi perfect đạt 100 nếu dùng trong target time; quá target thì `max(80, 100 - 20*(used/target - 1))`.
- Choice: đúng 100, neutral 50, sai 0.
- Order: `100 * số cặp kề đúng / (n-1)`.
- Session accuracy là trung bình điểm các bước được cấu hình; bước không submit tính 0. Kết quả bị clamp 0–100, làm tròn 2 chữ số.
- Thời lượng dùng `1 - min(.05*refinementRealm, .20)`, không thấp hơn .50 nhân với thời lượng cấu hình; stage snapshot giữ cấu hình tại lúc tạo session.

- Recipe definition, required materials, currency costs, target rank, variant và Gu requirements lấy từ DB/CSV.
- Deduction dùng Thought + Primeval Stones. Cảm ngộ dùng Thought, Primeval Stones và có thể Immortal Stones/catalyst theo rank.
- Quick refine tiêu hao mana, currency, materials và Gu theo recipe. Target Gu unique đã tồn tại có thể làm lần luyện Immortal thất bại.
- Session tạo transactionally: khóa selected Gu, escrow/deduct materials và currency, lưu rate snapshot, plan, step results, accuracy/death penalty và end time. Step config rank 1–9, thường gồm 3–12 steps theo RefineStage data.
- Submit step cần session owner, ACTIVE status, đúng index/window/nonce/action. Cùng nonce được xử lý idempotent; response stale hoặc invalid không nên retry với payload mới nếu chưa tải state.
- Abort refund `floor(qty/2)` của mỗi material; không hoàn mana/stone. Selected Gu được mở khóa, unequip, đặt hunger thấp và vào DYING state.
- Timeout/resolve dùng worker/lazy resolve; kết quả không phụ thuộc client tự đoán qua thanh UI.

## Kết quả và trường hợp thất bại

- Deduce không đủ Thought/stone, recipe chưa sở hữu, đã 100% hoặc target data lỗi sẽ bị từ chối.
- Conception có thể thành công, lệch recipe, hoặc backlash. Backlash có thể làm mất Thought còn lại và gây soul-injury status, đặc biệt nguy hiểm ở target Immortal.
- Quick refine trả lỗi `REFINE_SESSION_REQUIRED` khi target cần timed session.
- Session có thể RESOLVED hoặc ABORTED. Hủy giữ lại một nửa material theo floor và làm Gu ingredients suy kiệt; đây không phải rollback hoàn toàn.
- Một số kết quả thành công vẫn có thể tiêu hao Gu ingredient theo recipe; đọc result object trước khi tiếp tục thao tác inventory.

## Liên hệ với hệ thống khác

- Recipe progress là prerequisite của luyện Cổ và quest progression.
- Gu/materials lấy từ Inventory; Gu locked không dùng lại cho combat, trade hoặc sản phẩm khác.
- Dao Marks, Wisdom/Refinement mastery và Gu effects ảnh hưởng deduction/conception.
- Aperture refinement queue là một feature khác, dùng handler/worker riêng; không nhầm với player timed refine session.

## Dữ liệu/cấu hình ảnh hưởng

- `Recipe.csv` → `refining_recipes`.
- `RefineStage.csv` → stage/minigame plan theo target rank.
- `RefineDecision.csv` → decision data được dùng bởi session.
- `Gu.csv`/`Item.csv` định nghĩa target, path, ingredient, effects và catalysts.
- Import qua Admin → Dữ liệu; xem [GAME_DATA.md](../guides/GAME_DATA.md).

## Mã nguồn liên quan

- Frontend: `frontend/src/components/RefiningFurnace.tsx`, `RefineSessionPanel.tsx`, `RefineConceiveTab.tsx`; `frontend/src/store/slices/refineSessionSlice.ts`, `recipeSync.ts`.
- Backend: `backend/internal/handlers/refine_core.go`, `refine_gu.go`, `refine_deduce.go`, `refine_conceive.go`, `refine_session.go`, `refine_session_plan.go`, `refine_session_score.go`, `refine_session_resolve.go`.
- Worker/schema: `backend/internal/handlers/refine_session_resolve.go` worker startup, `backend/internal/db/schema_refine.go`.

## API liên quan

- Public catalog: `GET /recipes`.
- Recipe: `GET /refine/conceive/candidates`, `GET /refine/preview`, `POST /refine/deduce`, `POST /refine/conceive`, `POST /refine`.
- Session: `POST /refine/sessions`, `GET /refine/sessions/active`, `GET /refine/sessions/:id`, `POST /refine/sessions/:id/steps`, `/abort`, `/ack`.

## Ví dụ flow

```text
GET /recipes → GET /refine/preview?recipe_id=...
→ nếu progress < 100: POST /refine/deduce đến khi hoàn chỉnh
→ chọn quick POST /refine hoặc POST /refine/sessions
→ session: GET active → submit từng step theo nonce/window → nhận resolve result
→ POST ack chỉ để đánh dấu đã xem
```
