# Nhiệm vụ, tiến trình và phần thưởng

## Tính năng làm gì?

Domain này gom các objective phụ giúp người chơi học hệ thống và nhận thưởng: quest, daily missions, achievements, danh hiệu/khung, referral rewards và gift codes. Mỗi nguồn thưởng có record/điều kiện riêng; hoàn thành activity không đồng nghĩa đã nhận thưởng.

## Điều kiện mở và quyền truy cập

- Quest, daily mission và achievement routes yêu cầu đăng nhập, onboarding hoàn tất.
- Daily mission được khởi tạo khi player đọc bảng lần đầu nếu chưa có bộ ngày đó.
- Referral cần mã/link hợp lệ và trạng thái liên kết phù hợp; claim yêu cầu phần thưởng đã đủ điều kiện.
- Gift code phải còn hiệu lực, chưa bị dùng theo policy và account phải đáp ứng giới hạn redemption.

## Cách sử dụng

### Quest

1. Gọi `GET /quest` để đọc quest hiện hành, yêu cầu và rewards.
2. Thực hiện objective trong gameplay. Một số quest kiểm tra trạng thái player trực tiếp; một số kiểm tra inventory hoặc exploration progress.
3. Gọi `POST /quest/complete` với quest ID khi đã đạt điều kiện.
4. Đọc response/player snapshot; reward được áp dụng phía server.

### Daily missions

1. Gọi `GET /daily-missions`; server tạo hoặc trả bộ nhiệm vụ của player trong ngày game.
2. Tiến độ tăng từ các event gameplay đã được server xác nhận.
3. Gọi `POST /daily-missions/claim` cho mission đã hoàn tất.

### Achievements và danh hiệu

1. Đọc `/achievements` để xem trạng thái, tiến độ và phần thưởng.
2. Claim achievement hoàn thành bằng `/achievements/claim`.
3. Chọn title/frame đang sở hữu qua `/achievements/select`; đây là lựa chọn hiển thị, không claim lại thưởng.

### Referral và gift code

Bind referral qua `/player/referrals/bind`, xem số liệu qua `/player/referrals`, nhận thưởng đủ điều kiện qua `/player/referrals/claim`. Gift code được đổi qua `/player/redeem-gift-code`.

## Luật, chi phí và giới hạn

- Quest definition lấy từ registry/data; backend vẫn kiểm tra `Check(player)` khi complete. Không tin tiến độ do client gửi.
- Unknown quest, quest đã hoàn thành hoặc chưa đủ yêu cầu bị từ chối.
- Daily mission pool/rank snapshot khóa theo ngày tại `Asia/Ho_Chi_Minh`; worker rollover xử lý chuyển ngày. Event progress có deduplication để tránh cộng trùng.
- Claim/selection có thể bị giới hạn một lần theo trạng thái ownership.
- Referral và gift code có anti-abuse/rate limits; không ghi giá trị/code cụ thể trong tài liệu vì phụ thuộc dữ liệu và cấu hình runtime.

## Kết quả và trường hợp thất bại

- Quest completion áp dụng reward trong transaction; response trả trạng thái đã được server ghi nhận.
- Daily mission chưa hoàn thành không thể claim; claim trùng bị chặn.
- Achievement reward ownership độc lập với việc chọn title/frame.
- Invalid/expired/already redeemed gift code hoặc referral state không hợp lệ sẽ trả lỗi domain.

## Liên hệ với hệ thống khác

Quest objectives thường gắn cultivation, inventory, rank, exploration và aperture. Daily mission chỉ nhận event từ các handler đã tích hợp tracker. Achievement được cập nhật từ gameplay events. Reward có thể thêm currency, item, Gu hoặc các trạng thái progression, nên phải đối chiếu inventory/player snapshot sau claim.

## Dữ liệu/cấu hình ảnh hưởng

- Quest definitions lấy từ `backend/data/Quest.csv`, bảng `quests` và quest registry.
- Daily mission pools/rank eligibility nằm trong `handlers/daily_missions.go`; rollover timezone là `Asia/Ho_Chi_Minh`.
- Achievement definitions/reward và referral/gift-code records được lưu theo schema domain; đọc code/schema trước khi thay field.
- CSV import/reload được mô tả trong [GAME_DATA.md](../guides/GAME_DATA.md).

## Mã nguồn liên quan

- Frontend: `frontend/src/components/AchievementsPanel.tsx`, `frontend/src/store/slices/achievementsSlice.ts`, `frontend/src/store/slices/referralsSlice.ts`.
- Backend: `backend/internal/handlers/achievements.go`, `daily_missions.go`, `gift_codes.go`; quest/referral route ownership xem `backend/internal/server/server.go`.
- Persistence/workers: `backend/internal/db/schema_daily_missions.go`, achievement/player schema, `backend/internal/worker/daily_missions.go`.

## API liên quan

- `GET /quest`, `POST /quest/complete`.
- `GET /daily-missions`, `POST /daily-missions/claim`.
- `GET /achievements`, `POST /achievements/claim`, `POST /achievements/select`.
- `GET /player/referrals`, `POST /player/referrals/bind`, `POST /player/referrals/claim`, `POST /player/redeem-gift-code`.
- Auth, onboarding và rate limits: [API.md](../guides/API.md).

## Ví dụ flow

```text
GET /daily-missions → làm objective trong gameplay
                   → GET /daily-missions kiểm tra trạng thái mới
                   → POST /daily-missions/claim
                   → dùng response player/inventory làm trạng thái hiện hành
```
