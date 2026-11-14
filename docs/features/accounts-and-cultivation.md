# Tài khoản, nhân vật và tu luyện

## Tính năng làm gì?

Tài khoản tạo danh tính đăng nhập. Nhân vật giữ rank, chỉ số, vùng, trạng thái và tiến trình tu luyện. Người chơi mới đi qua tutorial Rank 0, hoàn tất onboarding, rồi dùng phá vách/Xá Lợi Cổ để tiến triển qua rank phàm nhân. Thăng Tiên là quy trình riêng để chuyển từ Rank 5 sang Rank 6; nó mở Tiên Khiếu.

## Điều kiện mở và quyền truy cập

- Đăng ký/đăng nhập không cần onboarding. OTP và Google OAuth chỉ hoạt động khi cấu hình server tương ứng bật.
- `/auth/me` và player mutations cần session hợp lệ. Profile public có endpoint riêng.
- `complete-tutorial` và `complete-onboarding` là hai bước tách biệt; middleware cho phép hai route này trước khi onboarding xong.
- Break Wall chỉ dành cho Rank < 6 và bị chặn nếu status hiện tại cấm Cultivate/Breakthrough.
- Shatter yêu cầu Rank 5, SubRank 4, wall progress đủ, Cổ Bản Mệnh còn sống, đủ Nhân Khí và Chân Nguyên.
- Một số hành động cần vị trí/encounter world hoạt động; Thăng Tiên calamity yêu cầu vùng cho phép combat.

## Cách sử dụng

### Tài khoản và phiên

1. Gọi `GET /auth/config` để UI biết chính sách xác thực.
2. Nếu OTP được yêu cầu, gửi mã qua `/auth/send-otp`, rồi đăng ký với OTP.
3. Đăng nhập hoặc hoàn tất Google OAuth; frontend nạp player bằng `/auth/me`.
4. Client gameplay claim tab hiện hành qua `/auth/claim-tab`; logout kết thúc session phía client/server.

### Tutorial và onboarding

1. Player mới xem `LevelZeroView`; hoàn tất để chuyển qua bước khởi tạo nhân vật.
2. Chọn vùng xuất phát, Cổ tấn công, Cổ phòng thủ và vital Gu trong `OnboardingModal`.
3. Server chuẩn hóa vùng về hub, kiểm tra dữ liệu Gu và tạo loadout ban đầu.
4. Frontend tải lại player/inventory rồi vào lobby.

### Phá vách và Xá Lợi Cổ

1. Trong màn hình nhân vật, chọn lượng Chân Nguyên muốn dùng cho `/cultivate/break-wall`.
2. Server trừ mana, tính wall progress từ mana/max mana, aptitude, rank/subrank và cultivation effects.
3. Khi progress đầy, server thử đột phá; tại Rank 5 Đỉnh Phong, phá vách thường không thay thế nghi thức Thăng Tiên.
4. Nếu dùng Relic Gu, `/cultivate/use-relic-gu` tiêu thụ relic hợp lệ và tiến subrank ngay theo logic hiện hành.

### Thăng Tiên

1. Đọc checklist/trạng thái bằng `GET /cultivate/ascension/session`.
2. Gửi `/ascension/shatter` để thử nát khiếu. Mỗi lần tốn phần Chân Nguyên tối đa được cấu hình; xác suất tăng theo số lần thử đến mức trần.
3. Khi mở bước Dẫn Khí, chọn một phương án ở từng tầng. Mỗi lựa chọn trừ Soul HP, Chân Nguyên hoặc Nguyên Thạch theo lựa chọn.
4. Gọi `/ascension/calamity/engage` để tạo encounter tai kiếp tại thế giới. Combat theo lượt; thắng chuyển session sang Thả Cổ.
5. Chọn các Cổ Rank 5 đủ điều kiện để hiến tế qua `/ascension/release`.

## Luật, chi phí và giới hạn

### Công thức tu luyện đang dùng

Trong `cultivate.go`, đặt `manaRatio = manaSpend / max(1, effectiveMaxMana)`. Hệ số tư chất `aptMul` là Giáp 1.4, Ất 1.1, Bính 0.85, Đinh 0.6; giá trị khác dùng 0.8. Tiến độ cộng mỗi lần:

```text
gain = manaRatio * (0.2 + 0.3*aptMul) / (rank + subrank/4)
       * (1 + cultivationEfficiencyBonus)
gain *= 0.25 nếu apertureStress >= 80
wallProgress = min(1, wallProgress + gain)
```

Stress tăng `min(10, manaSpend/10) * (1 + apertureStressBonus)`. Khi stress đạt 100: stress về 50, max mana thành `max(20, round(maxMana*0.95))`, tiến độ về 0 và tụt một subrank (hoặc rank). Khi vách đầy, chance đột phá là `clamp(baseAptitudeChance - rankPenalty, 0.05, 1)`: base Giáp/A 1.0, Ất/B 0.8, Bính/C 0.7, Đinh/D 0.6, mặc định 0.5; penalty rank 1–6 lần lượt `0, .1, .2, .3, .4, .8`. Thất bại trừ thêm 20% mana hiện tại và 0.1 tiến độ.

Thăng Tiên dùng `chance = min(0.90, 0.15 + 0.15*priorAttempts)` và mỗi lần nát khiếu tốn `max(1, 0.30*effectiveMaxMana)`. Grade theo Nhân Khí: LOW từ 100.000, MID từ 230.000, HIGH từ 370.000, EXTREME từ 480.000. Đây là constants hiện tại trong handler, không phải công thức lấy từ CSV.

### Break Wall

- Mana spend phải lớn hơn 0 và không vượt số đang có.
- Progress phụ thuộc aptitude/rank/subrank và cultivation efficiency; stress từ 80 trở lên giảm tốc độ tiến độ.
- Mỗi lần trùng kích tăng stress. Khi stress chạm 100, stress về 50, wall progress reset, max mana giảm 5% (tối thiểu 20), và nhân vật tụt một subrank hoặc rank.
- Nếu lần đột phá khi đầy progress thất bại, player mất thêm 20% mana hiện tại và 10% wall progress; server tăng Human Qi theo flow hiện hành.
- Các mức aptitude và breakthrough chance được tính ở handler. Không dùng lời dẫn tutorial thay cho công thức server.

### Xá Lợi Cổ

Relic phải tồn tại trong inventory/materials ở key được server nhận. Dùng relic bị chặn khi breakthrough status bị khóa; khi thành công, relic bị tiêu thụ, wall progress reset và Human Qi tăng theo handler.

### Thăng Tiên

Các mốc hiện trong handler: tối thiểu 100.000 Human Qi, deadline 2 giờ sau khi phiên đi vào các bước sau Shattering, nát khiếu tốn 30% effective max mana mỗi lần, và mỗi Cổ hợp lệ ở bước Thả Cổ cấp 300 Dao Marks. Temper có ba tầng; mỗi tầng có hai lựa chọn được API trả về. Các giá trị này phải cập nhật tài liệu nếu constants/logic đổi.

| Giai đoạn | Kết quả chính | Thất bại/giới hạn |
| --- | --- | --- |
| Shattering | Thành công chuyển sang Dẫn Khí; attempt thất bại vẫn để phiên chờ thử lại | Mỗi attempt mất mana; yêu cầu rank, tiến độ và vital Gu |
| Dẫn Khí | Chọn phương án đủ khả năng thanh toán ở ba tầng | Không đủ option hoặc human qi cuối kỳ thấp sẽ làm nghi thức thất bại |
| Calamity | Tạo `monster_earth_calamity_boss` gần player | Bị chặn nếu world/combat/Gu theo vùng không hợp lệ |
| Release | Hiến Gu Rank 5 đạt path mastery, tạo aperture Rank 6 ở grade đã khóa | Gu phải thuộc player, không DEAD/vital/reserved; phiên hết hạn áp dụng penalty |

## Kết quả và trường hợp thất bại

- Tutorial/onboarding không hoàn thành sẽ giữ một số mutation bị khóa; các read route vẫn có thể dùng để render bước tiếp theo.
- Validation lỗi trả lỗi HTTP/domain; client nên hiển thị lỗi thay vì tự cộng tiến độ.
- Breakthrough fail và aperture crack là hai hậu quả khác nhau.
- Ascension timeout/failure giảm một phần Human Qi và giữ player ở Rank 5 Đỉnh Phong; kiểm tra response session trước khi khởi tạo lại.
- Combat tai kiếp được settle qua cùng combat lifecycle; không tự gán thành công ở client.

## Liên hệ với hệ thống khác

- Onboarding khởi tạo inventory Gu và vùng; dữ liệu vùng xuất phát/hub lấy từ Region.
- Break Wall/Relic tiến độ tác động tới quest và daily missions.
- Ascension release hiến Gu inventory, cấp Dao Marks và mở Immortal Aperture.
- Profile, disguise và avatar là tính năng tài khoản, không đổi rank/progression.

## Dữ liệu/cấu hình ảnh hưởng

- `backend/data/Race.csv`, `Constitution.csv`, `Region.csv`, `Gu.csv` và DB dictionary ảnh hưởng lựa chọn đầu game/effects.
- `backend/internal/handlers/ascension.go` sở hữu constants/temper options; world monster template quyết định boss calamity.
- `REQUIRE_EMAIL_VERIFICATION`, SMTP và Google OAuth env ảnh hưởng onboarding tài khoản.
- Hướng dẫn import/cache nằm ở [GAME_DATA.md](../guides/GAME_DATA.md); env ở [DEVELOPMENT.md](../guides/DEVELOPMENT.md).

## Mã nguồn liên quan

- Frontend: `frontend/src/components/Registration.tsx`, `LevelZeroView.tsx`, `OnboardingModal.tsx`, `Aperture.tsx`; store slices `authSlice.ts`, `cultivationSlice.ts`, `profileSlice.ts`.
- Backend: `backend/internal/handlers/player_auth_login.go`, `player_auth_register.go`, `player_validation.go`, `cultivate.go` (tiến độ/chance), `ascension.go` (nát khiếu/grade), `legacy_engage_tribulation.go`, `legacy_resolve_tribulation.go`, `player_profile.go`.
- Middleware/schema: `backend/internal/middleware/onboarding.go`, `backend/internal/db/schema_players.go`, `schema_aperture.go`, `backend/internal/playerstate/`.

## API liên quan

- Auth: `/auth/config`, `/auth/send-otp`, `/auth/register`, `/auth/login`, `/auth/logout`, `/auth/me`, `/auth/claim-tab`, `/auth/google/*`.
- Player setup/profile: `/player/complete-tutorial`, `/player/complete-onboarding`, `/player/stats`, `/player/profile/:username`, `/player/profile/settings`, `/player/avatar`.
- Tu luyện: `/cultivate/break-wall`, `/cultivate/use-relic-gu`, `/cultivate/ascension/session`, `/cultivate/ascension/shatter`, `/cultivate/ascension/temper`, `/cultivate/ascension/calamity/engage`, `/cultivate/ascension/release`.
- Route guards và auth tổng quát: [API.md](../guides/API.md).

## Ví dụ flow

```text
register → tutorial Rank 0 → complete-tutorial
         → choose hub + starter Gu → complete-onboarding
         → break wall/use relic → Rank 5 Đỉnh Phong
         → shatter → temper × 3 → calamity combat → release Gu
         → Rank 6 + Immortal Aperture
```
