# Xã hội và tiện ích tài khoản

## Tính năng làm gì?

Gồm chat, mail/notification, leaderboard, profile, feedback, referral, gift code, audit và các tiện ích tài khoản.

## Điều kiện mở và quyền truy cập

Các thao tác người chơi cần đăng nhập; một số tính năng phụ thuộc onboarding, quyền nhận/quản lý nội dung hoặc giới hạn chống lạm dụng. Admin/audit routes yêu cầu role tương ứng.

## Cách sử dụng

Mở profile/leaderboard để xem dữ liệu công khai; chat/mail/notification để liên lạc và nhận thông báo; referral/gift code nhập mã qua màn hình tương ứng; gửi feedback từ tiện ích. Lịch sử sự kiện có thể nhận realtime qua SSE.

## Luật, chi phí và giới hạn

- Server là nguồn quyết định quyền xem, rate limit và trạng thái nhận thưởng.
- Referral/gift code chỉ settle một lần theo điều kiện của chương trình.
- Admin audit và moderation không phải công cụ người chơi thông thường.
- SSE thông báo thay đổi; thao tác ghi vẫn cần API xác nhận.

## Kết quả và trường hợp thất bại

Mã sai/hết hạn/đã dùng, nội dung bị lọc, thiếu quyền hoặc rate limit sẽ trả lỗi. Client nhận lỗi/toast và giữ state server hiện hành.

## Liên hệ với hệ thống khác

Notification phản ánh các domain như trade, quest, combat, social; profile và leaderboard đọc player state; auth tập trung xử lý session hết hạn.

## Dữ liệu/cấu hình ảnh hưởng

Gift/referral campaign, chat filter, leaderboard calculation và notification settings được cấu hình/lưu theo domain.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/chat.go`, `notifications.go`, `leaderboard.go`, `player_profile.go`, `feedback.go`, `gift_codes.go`, `auditlog.go`.
- Frontend: `frontend/src/store/slices/chatSlice.ts`, `notificationsSlice.ts`, `referralsSlice.ts`, `feedbackSlice.ts`, `profileSlice.ts`.

## API liên quan

Các route social/profile/utility được đăng ký trong `backend/internal/server/server.go`; xem [API guide](../guides/API.md).

## Ví dụ flow

```text
đăng nhập → nhập gift code → server kiểm tra campaign/ownership
→ áp dụng reward → notification/SSE báo kết quả → profile/inventory nhận state mới
```
