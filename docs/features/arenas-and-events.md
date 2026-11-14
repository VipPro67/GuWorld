# Đấu trường và sự kiện

## Tính năng làm gì?

Nhóm này gồm arena challenge, Death Arena, Martial Arena, Infinite Tower, gacha và cược linh thú.

## Điều kiện mở và quyền truy cập

Mỗi hoạt động có lịch, rank, phí hoặc registration window riêng. Arena challenge giới hạn 10 lượt/ngày. Death Arena đăng ký theo phase và bracket; Martial Arena ghép 1v1 tự động theo realm, không có route đăng ký thủ công.

## Cách sử dụng

Chọn hoạt động tại UI, kiểm tra quota/phase, đăng ký hoặc bắt đầu challenge. Match được xử lý theo combat engine; nhận reward/history sau khi worker hoặc handler kết thúc sự kiện. Gacha/cược có flow riêng và phải xác nhận chi phí trước khi gửi.

## Luật, chi phí và giới hạn

- Death Arena tối đa 16 đội, mỗi đội tối đa 6 người (96 slots); cần ít nhất 2 đội để tạo bracket. Lịch hiện tại: thứ Bảy 16:00–17:00 giờ Việt Nam, bắt đầu xử lý lúc 17:00.
- Martial Arena chỉ tạo giải khi có ít nhất 8 người chơi thuộc rank group online trong 5 phút gần nhất; snapshot tối đa 128 người. Ghép tự động, không đăng ký thủ công.
- Arena challenge cap: 10/ngày.
- Death Arena phụ thuộc registration phase, bracket và lịch xử lý.
- Martial Arena ghép tự động theo realm và dùng lịch tuần; không cần manual registration.
- Tower có progression/entry rules riêng; gacha và betting dùng currency/odds/state do server trả về.

## Kết quả và trường hợp thất bại

Hết quota, ngoài phase, không đủ rank/currency hoặc session đang bận sẽ chặn entry. Reward/history chỉ được ghi nhận qua settlement server-side.

## Liên hệ với hệ thống khác

Combat xử lý trận đấu; rank quyết định grouping/eligibility; economy cấp/tiêu currency; worker xử lý vòng đời giải đấu.

## Dữ liệu/cấu hình ảnh hưởng

Arena settings, schedule, bracket, reward tables, tower configuration, gacha pool và betting market là nguồn giới hạn runtime.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/arena.go`, `death_arena_*.go`, `martial_arena_*.go`, `tower.go`, `gacha.go`, `beast_betting.go`.
- Frontend: `frontend/src/store/slices/arenaSlice.ts`, `martialArenaSlice.ts`, `gachaSlice.ts`, `bettingSlice.ts`.

## API liên quan

Routes theo từng hoạt động trong `backend/internal/server/server.go`; xem [API guide](../guides/API.md).

## Ví dụ flow

```text
kiểm tra quota/phase → đăng ký hoặc challenge → combat theo lượt
→ settlement ghi bracket/rank/reward → xem history
```
