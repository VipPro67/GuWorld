# Kế hoạch chuyển combat turn-based sang realtime

Ngày lập: 2026-10-27. Trạng thái: đang thực hiện.

Cập nhật 2026-10-27 (lần 1): chặng 1 (bất biến cast, checkpoint, độ bền
cooldown/status qua rejoin) và mục 2.1 (`movement_policy`, `recovery_ms`) đã
hoàn tất kèm test. Compiler giờ từ chối `STAT_MOD` active thay vì nhận rồi bỏ
qua (một phần 2.2); `PREVIEW_LAST_STAND` đã bị loại khỏi nhân vật thật (một
phần 2.3). Phần còn lại của chặng 2–4 vẫn mở.

## Mục tiêu và nguyên tắc

Mỗi Cổ, sát chiêu, passive, tài nguyên và hậu quả combat đã có trong turn-based phải có hành vi realtime tương đương hoặc được khai báo rõ là chưa thể cast. Không tự quy đổi `turns` thành giây, không thay `UNIQUE` bằng sát thương giả. Server zone quyết định kết quả; profile và dữ liệu Cổ là nguồn cấu hình. Các hiệu ứng ngoài combat tiếp tục đi qua handler hiện có.

`backend/internal/simulator/engine.go` là chuẩn cho luật gameplay hiện hữu, nhưng không phải chuẩn cho lịch thời gian: turn-based không thực thi `cooldown_ms`, `cast_time_ms`, `cast_range`. Realtime dùng tick 20 Hz và các thời lượng được tác giả profile khai báo. Khi luật cũ bất nhất, cần test nêu rõ luật được chọn trước khi chuyển.

## Hiện trạng được kiểm chứng

- Turn-based trừ độ no theo instance Cổ, dùng sát chiêu với các Cổ thành phần, tính mastery, điều kiện chiến thuật, passive/native effect, mana, phản phệ, đạo ngân, kết quả và thưởng.
- Realtime có hotbar Cổ từ catalog, kiểm tra cast trên zone, damage/defense, shield/heal, một status, projectile/AoE và loot idempotent.
- Catalog hiện có 4.427 profile, 2.423 active, 1.294 có passive, 710 chưa cast được; 871 entry cần reauthor theo audit. Con số `SUPPORTED` trong audit chỉ nói về từ vựng effect, không chứng minh cast parity.
- Working tree lúc lập kế hoạch có thay đổi chưa commit ở realtime/tai kiếp và các tài liệu khác. Giữ nguyên phần việc đó.

## Các chặng thực hiện

### 1. Bất biến cast và tài nguyên

- [x] Gắn instance ID và độ no vào slot; Cổ 0 độ no không cast được. Mỗi lần thử dùng hợp lệ tiêu hao đúng 1 độ no theo luật turn-based, kể cả khi phản phệ hoặc thiếu mana nếu đó là thứ tự của luật cũ. Không trừ cho lệnh sai slot/target/range/GCD.
- [x] Checkpoint delta độ no cùng HP, mana và vị trí trong một transaction, có điều kiện ownership/revision; lỗi DB phải giữ session để thử lại. Đồng bộ inventory snapshot sau commit.
- [x] Khóa hoặc định nghĩa rõ đánh thường, di chuyển, cast mới trong windup/recovery; không cho bypass bằng nhiều command hoặc reconnect.
- [x] Persist cooldown, status, hidden proc và các tài nguyên combat cần qua rejoin/transfer với mốc thời gian có thể khôi phục, không reset để né debuff.

### 2. Thực thi profile đúng với dữ liệu

- [x] Đưa `movement_policy`, `recovery_ms` vào runtime. Hiện mọi active profile khai báo `SLOW` và recovery 150–350 ms nhưng zone đang root lúc windup và bỏ recovery.
- [ ] Chạy từng operation trong active theo thứ tự và target đã khai báo. Không chấp nhận `STAT_MOD` rồi bỏ qua; giữ nhiều status cùng lúc khi profile cần.
- [ ] Gắn passive/hidden passive của Cổ sở hữu vào zone; bỏ `PREVIEW_LAST_STAND` khỏi nhân vật thật. Tránh cộng đôi chỉ số đã được `CalculatePlayerStats`/`gamefx` đưa vào snapshot.
- [ ] Đối chiếu bảng `Gu.csv`/Block v2 và `docs/GAME_DATA.md` cho từng loại: damage, heal, shield, mana restore, direct HP, strip/bypass, multihit, lifesteal, crit, thorns, control, cleanse, status/drain, summon và native hook. Viết runtime handler xác định cho `UNIQUE` có gameplay; các entry chưa author vẫn bị khóa cast.
- [ ] Tích hợp sát chiêu: xác thực từng instance thành phần, mana/độ no, xác suất, mastery, effect và dữ liệu recipe; không giả làm một slot Cổ thông thường.

### 3. Kết quả trận và độ bền

- [ ] Chuyển thưởng Nhân Khí, tiến triển lưu phái/đạo ngân, death/drop, status sau combat và hành vi PvP/party phù hợp realtime; so với các nhánh finalize turn-based.
- [ ] Death/reward intent phải bền vững hoặc có retry có giới hạn; queue đầy hay DB lỗi không được làm mất thưởng. Giữ idempotency key.
- [ ] Rà soát tính nhất quán của HP tối đa, passive, loadout khi đổi trang bị hoặc độ no trong session.

### 4. Chứng minh parity và tài liệu

- [ ] Bảng đối chiếu toàn bộ operation/trigger Block v2: nguồn turn-based, profile realtime, handler realtime, test và trạng thái coverage. Đo coverage theo **cast thực tế**, không dùng audit từ vựng làm thước đo.
- [ ] Test deterministic cho cast/cancel/rejoin/checkpoint đồng thời, từng status/native hook, sát chiêu, kill reward và lỗi DB/queue. Test với dữ liệu Cổ thật và nhiều instance cùng Gu ID.
- [ ] Chạy `go test ./...`, `go test -race ./...` cho thay đổi zone/concurrency, `npm run lint` và `npm run build` khi sửa frontend.
- [ ] Cập nhật `WIKI.md`, `docs/GAME_DATA.md`, `docs/mmo/gu-realtime-schema.md` và tài liệu API/schema tương ứng với các luật thực tế.

## Thứ tự triển khai

Làm chặng 1 trước để không tạo thêm đường cast miễn phí hoặc reset trạng thái. Sau đó triển khai engine effect theo nhóm nhỏ có test dữ liệu thật, cuối cùng mới mở cast cho entry đã đạt parity. Mỗi chặng phải chạy được độc lập; chưa hoàn tất chặng 4 thì mục tiêu chuyển toàn bộ cơ chế vẫn là **chưa đạt**.
