# Thiên Cơ: trinh sát và divination

## Tính năng làm gì?

Thiên Cơ cung cấp inspect cục bộ, divination từ xa, dấu truy tung và các manh mối/resonance liên quan. Các thao tác tách biệt theo khoảng cách, mục tiêu và chi phí.

## Điều kiện mở và quyền truy cập

Các route scouting cần phiên người chơi hợp lệ. Inspect cùng party/sect có thể dùng quyền quan hệ; mục tiêu khác phải ở cùng vùng và trong phạm vi. Divination từ xa dùng tài nguyên riêng.

## Cách sử dụng

Chọn mục tiêu để inspect nếu ở gần; dùng remote divination khi cần tìm người ở xa. Các màn hình scouting có thể đặt tracking mark hoặc dùng sever clue/resonance nếu endpoint và dữ liệu mục tiêu hỗ trợ.

## Luật, chi phí và giới hạn

### Tầm và chi phí hiện tại

Inspect cục bộ trong cùng vùng dùng khoảng cách Euclidean và phạm vi `15 + 2*playerRank + 2*Σ(INVESTIGATION của Gu ACTIVE, equipped)`. Cùng party hoặc cùng sect được bypass kiểm tra tầm cục bộ. Divination thường tiêu tối thiểu 100 Primeval Stones; resource bonus bằng 10% chi phí. Divination bằng immortal stones yêu cầu Rank 6+, tối thiểu 1 stone và bonus theo lượng gửi nhân 5. Divine Gu dùng hệ số immortal 6. Resonance Scan tốn 50 mana + 200 Primeval Stones; Sever Clues tốn 20 mana. Đây là cost hiện trong handler.

- Phạm vi inspect ngoài party/sect là `15 + rank*2 + 2*sum(INVESTIGATION của Gu đang trang bị)` ô Euclidean.
- Divination tiêu hao Stones; lựa chọn immortal stone yêu cầu Rank 6+ và áp dụng bonus mạnh hơn.
- Không thể inspect mục tiêu đang sở hữu realtime session; server trả conflict.
- Quyền, concealment và trạng thái mục tiêu vẫn được kiểm tra server-side.

## Kết quả và trường hợp thất bại

Ngoài vùng/phạm vi, thiếu currency hoặc target realtime-owned khiến request thất bại. Kết quả chỉ tiết lộ mức thông tin cho phép, không cấp quyền điều khiển mục tiêu.

## Liên hệ với hệ thống khác

Rank, Gu Investigation, concealment, party/sect và world presence ảnh hưởng khả năng phát hiện. Tracking/clue lưu trạng thái riêng, không thay thế movement hoặc combat.

## Dữ liệu/cấu hình ảnh hưởng

Các hệ số và trạng thái scouting nằm trong handlers/config/schema; Gu effects ảnh hưởng chỉ số lấy từ registry game data.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/scouting.go` và các handler scouting liên quan.
- Frontend: `frontend/src/components/DivinationModal.tsx`, `TrackingRadarDrawer.tsx`, `frontend/src/store/slices/scoutingSlice.ts`.

## API liên quan

Nhóm `/scouting/*`; xem routes được đăng ký trong `backend/internal/server/server.go` và [API guide](../guides/API.md).

## Ví dụ flow

```text
chọn người chơi → kiểm tra cùng party/sect hoặc khoảng cách → inspect
→ nếu ngoài phạm vi, chọn divination → trả Stones → nhận kết quả giới hạn
```
