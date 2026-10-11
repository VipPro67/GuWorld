# GuWorld Gameplay Wiki

Wiki này phản ánh các hệ thống đã có trong codebase hiện tại, diễn tả bằng khẩu khí Cổ Chân Nhân.

## 1. Tài khoản và nhân vật

- Cổ sư đăng ký bằng username/email/password hoặc đăng nhập Google nếu cấu hình OAuth.
- **Khóa tài khoản (`players.ban`)**: tài khoản bị khóa (có thời hạn hoặc vĩnh viễn) không thể đăng nhập và bị đẩy khỏi phiên đang chơi; đăng nhập trả về lý do khóa. Admin cũng có thể chỉ **hạn chế chat** (không gửi được chat Bảo Hoàng/private mail) hoặc chặn thiết bị. Hệ thống tự động khóa khi một thiết bị vượt ngưỡng nhiều tài khoản hoặc lạm dụng referral.
- Hồ sơ hỗ trợ bio, avatar, đổi mật khẩu và cơ chế ngụy trang thông tin công khai.
- **Đổi Đạo Hiệu (Hạn chế & Đại Đạo Phản Phệ)**:
  - **Nhất Chuyển & Nhị Chuyển (Rank 1 - 2)**: Tự do đổi tên miễn phí, không tốn tài nguyên hay chịu phản phệ.
  - **Tam Chuyển (Rank 3)**: Tốn **5.000 Nguyên Thạch**.
  - **Tứ Chuyển (Rank 4)**: Tốn **20.000 Nguyên Thạch**.
  - **Ngũ Chuyển (Rank 5)**: Tốn **100.000 Nguyên Thạch**.
  - **Cổ Tiên & Tôn Giả (Rank 6 - 9)**: Nghịch thiên cải mệnh, đổi đạo hiệu gây chấn động Thiên Địa và chịu phản phệ cực lớn (không tụt tiểu cảnh giới):
    - **Tiêu hao Tiên Nguyên Thạch**: Lục Chuyển (**100**), Thất Chuyển (**500**), Bát Chuyển (**2.500**), Cửu Chuyển (**10.000** TNT).
    - **Tụt Đạo Ngân Tiên Khiếu**: Lục Chuyển (**-30%**), Thất Chuyển (**-40%**), Bát Chuyển (**-50%**), Cửu Chuyển (**-70%**).
    - **Tụt Điểm Chân Ý (Cảnh Giới Lưu Phái)**: Lục Chuyển (**-30%**), Thất Chuyển (**-40%**), Bát Chuyển (**-50%**), Cửu Chuyển (**-70%**).
    - **Tổn hại Tiên Khiếu**: Giảm HP Tiên Khiếu (20% / 30% / 40% / 50%) và tăng Áp lực Tiên Khiếu (+30 / +50 / +80 / +100).
    - **Thiên Đạo Biến Động**: Phát thông cáo toàn cõi trên kênh Thế Giới (World Chat).
- Referral có thống kê người được mời và phần thưởng/mốc xử lý ở backend.

### Chủng tộc, Giới tính, Thể chất, Trạng thái

Khi đăng ký, đạo hữu mặc định Nhân Tộc  và chọn **Giới tính**. Trong quá trình tu hành, **Thể chất** và **Trạng thái** có thể thay đổi qua sự kiện/quà hoặc Admin. Mỗi giá trị mang hiệu ứng riêng:

**Chủng tộc:**

| Chủng tộc | Hiệu ứng |
| --- | --- |
| Nhân Tộc (`human`) | Không bonus (mặc định) |
| Người Lông (`fur_people`) | **+10% tỉ lệ Luyện Cổ thành công** |
| Nhân Ngư (`merman`) | Thủy Đạo +10% affinity, +5% sát thương Thủy Đạo |
| Vũ Dân (`feathered`) | +10% tốc độ, Không Gian Đạo +5% affinity |
| Mặc Nhân (`inkman`) | Trí Tuệ Đạo +8%, Hồn Đạo +5% affinity |
| Thạch Nhân (`rockman`) | +15% phòng thủ, -10% tốc độ |
| Tuyết Dân (`snowman`) | Hàn Băng Đạo +15% affinity, -10% sát thương Hỏa nhận |
| Thú Nhân (`variant_beastman`) | +8% tấn công, +5% phòng thủ, -5% tỉ lệ Luyện Cổ |
| Long Nhân (`dragonman`) | +10% tấn công, +10% phòng thủ, -5% tỉ lệ Luyện Cổ |
| Cự Dương Huyết Mạch (`giant_sun_descendant`) | +5 Vận, +5% hiệu quả tu luyện |
| Cương Thi (`cuong_thi`) | **Bất tử tuổi thọ** (thọ nguyên không bao giờ cạn); **Cấm tự hồi phục** HP, Chân Nguyên/Tiên Nguyên, Nguyên Lực, Thần Hồn. Khi ở trạng thái Tiên Cương: gia tốc thời gian khóa ở `1.0`, Tiên Khiếu thành Tiên Khiếu Chết, không còn tiên kiếp, và bị suy thoái `time_rate` & `area_size` theo thời gian |

Danh sách chủng tộc được **data-driven** từ bảng `race_dictionary` (seed từ `data/Race.csv`): admin thêm chủng tộc mới bằng cách sửa CSV (thêm dòng `key,name,effects`) rồi Import bảng "Chủng tộc" trong Admin panel tab **Dữ liệu**. Effect format xem `docs/GAME_DATA.md`.

### Tuổi thọ & Trạng thái Tử Vong

- **Cơ chế Tuổi thọ**: Cổ sư tiêu hao thọ nguyên theo thời gian thực (tốc độ hao mòn = thời gian thực × `time_speed_multiplier`).
- **Rơi vào Cõi Chết**: Khi thọ nguyên giảm xuống $\le 0$, Cổ sư bị chuyển sang trạng thái tử vong. Trạng thái này **không lưu riêng** — được suy ra từ `lifespan` mỗi lần đọc (`gamefx.IsDeadState`). Trong trạng thái này, mọi thao tác tu luyện, trùng kích, chiến đấu, thám hiểm và đào quặng bị phong tỏa hoàn toàn cho tới khi sử dụng **Thọ Nguyên Cổ** khôi phục tuổi thọ hoặc chuyển chủng tộc.
- **Tiên Cương (Chủng tộc Cương Thi)**:
  - **Bất tử tuổi thọ**: Thọ nguyên không bị suy giảm theo thời gian.
  - **Phong tỏa Hồi phục**: Bị cấm tự động hồi phục HP (`DisableHPRegen`), Chân Nguyên/Tiên Nguyên (`DisableManaRegen`), Nguyên Lực (`DisableResourceRegen`) và Thần Hồn (`DisableSoulRegen`).
  - **Tiên Khiếu Chết (Dead Aperture)**: Khi Cổ Tiên chuyển sang Cương Thi, gia tốc thời gian cá nhân `time_speed_multiplier` bị cố định ở `1.0` (không thể tăng tốc), đồng thời Tiên Khiếu ngưng trệ thành Tiên Khiếu Chết, liên tục bị suy thoái giảm dần diện tích (`area_size`) và tốc độ thời gian (`time_rate`) theo thời gian thực (~0.00175 rate/phút & ~416.67 area/phút).

**Giới tính:** Nam (`male`) / Nữ (`female`) — chưa có hiệu ứng (chỉ hiển thị).

**Trạng thái:**

| Trạng thái | Hiệu ứng | Thời hạn |
| --- | --- | --- |
| Trọng Thương (`severely_wounded`) | Khi đổi Bản Mệnh Cổ: công/thủ/tốc và tỉ lệ luyện cổ cùng giảm theo rank/subrank như Áp Chế Giới Vực (10–90%); hồi phục nhận giảm **50%**; không thể thám hiểm | 5–40 giờ theo rank |
| Trúng Độc (`poisoned`) | **-10%** tấn công, **-10%** tốc độ | 30 phút |
| Kiệt Sức (`exhausted`) | **-30%** tốc độ, **-50%** hiệu suất tu luyện; không thể tu luyện | 10 phút |
| Phúc Trạch (`blessed`) | **+5** may mắn, **+10%** hiệu suất tu luyện | 60 phút |
| Hình Phạt Tử Vong (`death_penalty`) | **-50%** tấn công, phòng thủ, tốc độ | 30 phút |
| Tâm Thần Mệt Mỏi (`refine_fatigue`) | **-50%** tỉ lệ Luyện Cổ thành công | 10 phút |
| Luyện Cổ Phản Phệ (`refine_backfire`) | **-95%** tỉ lệ Luyện Cổ thành công | 20 phút |
| Sinh Khí Tàn (`sinh_khi_tan`) | Trúng Sát Sinh Khí Cổ: tầng I **-10%**, II **-25%**, III **-35%**, IV **-50%**, V **-99%** tấn công/phòng thủ/tốc độ. Đánh trúng +1 tầng (tối đa V, refresh 5h; đòn bị khiên hấp thụ vẫn tính trúng, né/mù/miễn nhiễm không tính); thanh tẩy -1 tầng | 5 giờ (refresh mỗi hit; spec gốc 50h đang tạm giảm cân bằng) |

> Gu `FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY` (Ngũ Hành Thiên Chướng Trận, rank 9) có 2 effect khi bắt đầu trận: **gỡ `regional_wall_suppression` khỏi bản thân** (người mang), và **phản chiếu Áp Chế Giới Vực lên địch PvP** (giảm 1 nửa debuff công/thủ/tốc so với Giới Vực Bích gốc, gắn status 15-55 phút theo rank địch) — chỉ PvP thực.

Quy tắc áp dụng trạng thái:

- `stat_mods` là **phần trăm trên tổng chỉ số** (cộng dồn sau toàn bộ nguồn khác: cơ bản, Cổ, buff,	Hộ Sơn…). Nếu tổng chỉ số trước đó bằng 0 thì mod không có tác dụng; chỉ số vật lý (**tấn công / phòng thủ / tốc độ / HP**) luôn **sàn ở 0** — áp chế nặng không thể đẩy chỉ số xuống âm.
- Trạng thái mang **giá trị cường độ âm** (`Value < 0`, VD Áp Chế Giới Vực tính theo rank) sẽ **thay toàn bộ** `%` trong `stat_mods` bằng cường độ đó; cột hiệu ứng trong CSV chỉ là giá trị mặc định hiển thị.

**Thể chất (10 Thập Tuyệt Thể + Thuần Mộng Cầu Chân Thể):**

| Thể chất | Hiệu ứng |
| --- | --- |
| Thái Nhật Dương Mãng Thể (`solar_sun_moon_body`) | Quang Đạo đạo ngân ×150%, +20% sát thương Quang Đạo, +50% Chân Nguyên tối đa, +100% hồi mana |
| Cổ Nguyệt Âm Hoang Thể (`ancient_moon_yin_desolate_body`) | Hắc Ám Đạo đạo ngân ×150%, +50% Chân Nguyên tối đa, +100% hồi mana, +20% kháng hiệu ứng tâm lý |
| Bắc Minh Băng Phách Thể (`northern_dark_ice_soul_body`) | Hàn Băng Đạo đạo ngân ×150%, giảm 10% sát thương Hàn Băng nhận, +50% Chân Nguyên tối đa, +100% hồi mana |
| Sâm Hải Luân Hồi Thể (`forest_sea_reincarnation_body`) | Thủy Đạo đạo ngân ×150%, +50% hồi phục nhận, +50% Chân Nguyên tối đa, +100% hồi mana |
| Vạn Kim Diệu Hoa Thể (`myriad_gold_brilliant_flower_body`) | Kim Đạo đạo ngân ×150%, +30% tấn công, +30% phòng thủ, +50% Chân Nguyên tối đa, +100% hồi mana |
| Viêm Hoàng Lôi Trạch Thể (`blazing_glory_lightning_brilliance_body`) | Lôi Đạo đạo ngân ×150%, +20% tốc độ, +50% Chân Nguyên tối đa, +100% hồi mana |
| Đại Lực Chân Võ Thể (`great_strength_true_martial_body`) | Lực Đạo đạo ngân ×150%, +30% tấn công, +30% phòng thủ, +50% Chân Nguyên tối đa, +100% hồi mana |
| Tiêu Dao Trí Tâm Thể (`carefree_wisdom_heart_body`) | Trí Đạo đạo ngân ×150%, +30% kháng hiệu ứng tâm lý, +50% Chân Nguyên tối đa, +100% hồi mana |
| Hậu Thổ Nguyên Ương Thể (`great_earth_origin_body`) | Thổ Đạo đạo ngân ×150%, +45% phòng thủ, +45% kháng hiệu ứng tâm lý, +50% Chân Nguyên tối đa, +100% hồi mana |
| Vũ Trụ Đại Diễn Thể (`universe_great_evolution_body`) | Không Gian Đạo đạo ngân ×150%, +15% tấn công/phòng thủ/tốc độ, +50% Chân Nguyên tối đa, +100% hồi mana |
| Thuần Mộng Cầu Chân Thể (`pure_dream_reality_seeker_body`) | Mộng Đạo đạo ngân ×140%, +10% sát thương Mộng Đạo, +40% Chân Nguyên tối đa, +100% hồi mana |

Trạng thái có thể tồn tại đồng thời nhiều cái (mảng `statuses`). Một số trạng thái có **thời hạn** (đếm theo giờ thực) và tự biến mất khi hết hạn. Danh sách trạng thái được **data-driven** từ bảng `status_dictionary` (seed từ `data/Status.csv`): admin thêm trạng thái mới bằng cách sửa CSV (thêm dòng `key,name,effects,default_duration_minutes,blocked_actions,description`) rồi Import bảng "Trạng thái" trong Admin panel tab **Dữ liệu**. Effect format xem `docs/GAME_DATA.md`.


## 2. Tu luyện

### Chuyển và tiểu cảnh giới

- Đại cảnh giới tu vi từ Nhất Chuyển đến Cửu Chuyển (1 đến 9).
- `sub_rank` gồm 4 bậc: Sơ giai, Trung giai, Cao giai, Đỉnh phong (chỉ dùng cho Chuyển 1-5).
- Nhất Chuyển đến Ngũ Chuyển là phàm nhân; Lục Chuyển trở lên là Cổ Tiên.
- Lục Chuyển trở lên **không có sub_rank**. Thay vào đó hiển thị tên `Lục Chuyển/Bát Chuyển/Cửu Chuyển Cổ Tiên/Cổ Tôn` kèm số lần vượt qua từng loại kiếp nạn (Địa Tai, Thiên Kiếp, Hạo Kiếp, Vạn Kiếp, Hỗn Độn Đại Nạn).
- **Vĩnh Sinh (Rank 10)** — cảnh giới "Vĩnh Sinh", phẩm chất Mana **Hỗn Độn Tiên Nguyên**. Chỉ được gán thủ công (không có đường đột phá từ Cửu Chuyển). Hai tác dụng: **không còn tai kiếp** và **bất tử tuổi thọ** (thọ nguyên không giảm vì lão hóa, không vào Cõi Chết). Mọi hoạt động Tiên Khiếu khác — sản xuất Tiên Nguyên, hệ sinh thái, dân cư, gia tốc thời gian, hồi phục — vẫn diễn ra bình thường.

### Cảnh Giới Lưu Phái (Path Masteries)

- Trong quá trình tu hành, Cổ sư có thể tích luỹ điểm **Chân Ý** của các lưu phái (ví dụ: Kiếm Đạo, Hỏa Đạo, Trí Đạo...) để thăng tiến cảnh giới lưu phái.
- **Cách nhận điểm Chân Ý:**
  - **Sử dụng Cổ tiêu hao/Cổ trong chiến đấu:** (Bao gồm cả đánh tay và đánh tự động/Simulator như khai thác mỏ, tàn tích). Mỗi lần thi triển thành công nhận Chân Ý cơ bản theo rank: **R1=1, R2=2, R3=5, R4=10, R5=20, R6=50, R7=100, R8=200, R9=300**. Lượng thực nhận còn phụ thuộc hiệu suất cảnh giới và giới hạn của rank Cổ. (Lưu ý: Đấu Trường Ảo không cộng điểm).
  - **Giới hạn mỗi trận:** điểm Chân Ý nhận từ chiến đấu đối với Cổ phàm và Sát chiêu phàm bị chặn ở **Rank × 10 điểm/trận** (đánh tay, đánh tự động và sát chiêu phàm đều tính chung). Riêng **Tiên Cổ và Sát chiêu Tiên đạo (Rank ≥ 6)** không bị giới hạn tối đa trong trận chiến, thỏa sức ngộ đạo không có trần giới hạn. Thưởng phó bản và luyện Cổ không tính vào giới hạn này.
  - Các nguồn điểm khác đã có trong game gồm phần thưởng Phó bản (như Mộng Cảnh) và Luyện Cổ; điểm thực nhận tuân theo công thức của từng hoạt động.
- **Các mốc cảnh giới:** Phàm Thai (< 10.000) -> Đại Sư (10.000) -> Tông Sư (100.000) -> Đại Tông Sư (1.000.000) -> Vô Thượng Đại Tông Sư (10.000.000).
- **Tác dụng:**
  - Là **điều kiện bắt buộc để đột phá tu vi Cổ Tiên** (VD: Đột phá Thất Chuyển cần ít nhất 1 lưu phái đạt Tông Sư; Bát Chuyển cần Đại Tông Sư).
  - Khi không đủ điều kiện, vượt kiếp thành công vẫn được tính số lần tai kiếp nhưng **không được thăng Chuyển**.

### Phàm nhân: phá vách

- Cổ sư tiêu hao Chân Nguyên để tăng `wall_progress`.
- Khi đủ tiến độ, nhân vật đột phá tiểu cảnh giới hoặc lên Chuyển kế tiếp.
- Tư chất ảnh hưởng tốc độ hồi/phá vách: Giáp, Ất, Bính, Đinh.
- Xá Lợi Cổ có thể được dùng qua hệ thống `use-relic-gu` nếu bọc đồ có Cổ phù hợp.
- cẩn thận áp lực không khiếu
### Thăng Tiên và Độ Kiếp

- Ngũ Chuyển Đỉnh phong (Chuyển 5, SubRank 4) có thể thăng tiên khi Nhân Khí đạt từ 100.000 trở lên.
- Nhân Thú Táng Sinh Cổ (`MAN_BEAST_BURIAL`) cho phép người chơi Rank 1–4 ở Đỉnh Phong cưỡng ép lên Rank kế tiếp Sơ Giai; rank Cổ phải bằng rank người dùng. Cổ trùng làm giảm 10 Chân Nguyên tối đa (tối thiểu 20), Chân Nguyên hiện tại cũng bị kẹp theo giới hạn mới.
- **Tỉ lệ thành công phụ thuộc vào Tư chất và Nhân Khí:**
  - **Tỉ lệ gốc (`baseChance`):** Giáp: `20%`, Ất/Bính/Đinh: `5%` (tính từ `GetBreakthroughChance(6, Aptitude)` với phạt Rank 6 là 80%).
  - **Bonus Nhân Khí:** Mỗi 100.000 Nhân Khí cộng thêm `+20%` tỉ lệ (`qiBonus = (NhânKhí / 100.000) * 20%`).
  - **Giới hạn tối đa:** Tổng tỉ lệ thành công tối đa là **90%**.
  - **Ví dụ tại 100.000 Nhân Khí:** Giáp = 40%, Ất/Bính/Đinh = 25%. Để đạt max 90%, Tư chất Giáp cần 350.000 Nhân Khí, các tư chất còn lại cần 425.000 Nhân Khí.
- **Thành công:** lên Lục Chuyển Cổ Tiên, nhận 10 Tiên Nguyên Thạch và 100 Đạo Ngân ở mỗi một trong 2 lưu phái ngẫu nhiên (không gồm Thiên Đạo). Kiếp đầu tiên giáng lâm vào **năm nội bộ thứ 10**, tương đương khoảng **2–10 giờ đời thật** tùy phẩm chất Tiên Khiếu (xem mục Gia tốc thời gian). **Tiên Khiếu được khai mở ngay**, phẩm cấp được định đoạt theo Nhân Khí tại thời điểm thăng tiên (xem mục Tiên khiếu).
- **Thất bại:** Nhân Khí về 0, tu vi phế bỏ rơi về Nhất Chuyển Sơ giai, mất hết tiến độ phá vách.
- Cổ Tiên có tiên khiếu, đạo ngân, tiên nguyên/tài nguyên riêng.
- Tai kiếp có worker kiểm tra tự động mỗi 60 giây. Khi năm nội bộ đạt mốc tiếp theo, worker sinh kiếp và resolve.
- **Lưu phái tai kiếp ưu tiên lưu phái của Cổ bản mệnh** nếu đó không phải Thiên Đạo. Tỉ lệ ưu tiên bắt đầu từ 80% và tăng theo cấp tai kiếp: Địa Tai **80%**, Thiên Kiếp **85%**, Hạo Kiếp **90%**, Vạn Kiếp **95%**, Hỗn Độn Đại Nạn **100%**; lượt không kích hoạt ưu tiên sẽ chọn ngẫu nhiên trong các lưu phái thường còn lại. Thiên Đạo không xuất hiện trong lưu phái chính hoặc phần Đạo Ngân phụ của phần thưởng tai kiếp.
- **Bảo hộ khi vắng mặt:** nếu đến mốc tai kiếp mà đạo hữu **đang offline**, tai kiếp sẽ không sinh quái và không phá hoại — Tiên Khiếu chuyển trạng thái **"Kiếp Lực Tích Lũy"** (`TRIBULATION_PENDING`). Khi đạo hữu online trở lại, tai kiếp mới giáng xuống (trong vòng 60 giây) và có **5 phút chuẩn bị** trước khi quái bắt đầu di chuyển/phá mỏ. Thời gian nội bộ vẫn trôi bình thường nên không thể trốn kiếp vô hạn; nếu giữ kiếp offline quá **24 giờ**, phần thưởng Đạo Ngân sẽ bị **giảm còn một nửa**. Bộ đếm 6 giờ tự kết toán chỉ tính thời gian online.
- Khi đến gần node/vùng, quái đánh vùng 3×3 quanh vị trí của nó. Mỗi ô thuộc node bị trúng cộng sát thương vào HP chung của node; sát thương mỗi ô theo loại kiếp: Địa Tai 1, Thiên Kiếp 2, Hạo Kiếp 4, Vạn Kiếp 8, Hỗn Độn Đại Nạn 16. Đại Yêu gây gấp đôi sát thương yêu tướng.
- **Combat tai kiếp dùng giao diện chiến đấu thông thường:** nhấn **"Xuất Trận Chiến Đấu"** trong bảng điều khiển tai kiếp sẽ mở màn chiến đấu (auto-combat) chống toàn bộ quái tai kiếp còn sống, mang theo đội hình Cổ như đánh mỏ/PvP/PvE. Giáp ảo (`shield_flat` từ Cổ) và **máu của đạo hữu tiêu hao trước**; sát thương tràn qua (vượt quá giáp + máu) mới trừ trực tiếp vào **Sinh lực Tiên Khiếu** (`immortal_apertures.aperture_hp`). Thanh **Máu Tiên Khiếu** hiển thị ngay trong màn chiến đấu và cập nhật theo từng lượt.
- **Cổ sư chỉ được xuất trận khi Sinh lực Tiên Khiếu > 5% Sinh lực tối đa.** Nếu Tiên Khiếu nguy cấp (≤ 5%), đạo hữu không thể ra tay — quái sẽ tiếp tục càn quét mỏ trong sự bất lực; hãy chờ Tiên Khiếu hồi phục hoặc kết toán vượt kiếp.
- **Kết toán tai kiếp:** Nếu kết toán khi còn quái trên bản đồ lưới (chấp nhận rủi ro hoặc hết hạn 6 giờ thực tế tự động kết toán), đạo hữu có nguy cơ thất bại. Khi vượt kiếp thất bại, bộ đếm số lượng tai kiếp (tổng số tai kiếp và số lượng theo từng loại) sẽ không tăng (chỉ tích lũy khi thành công), nhưng thời gian/mốc năm của tai kiếp kế tiếp (`next_tribulation_year`) vẫn sẽ được đẩy lên mốc tiếp theo để người chơi có thời gian hồi phục và không bị dồn dập tai kiếp ngay lập tức.
- **Kiếp nạn theo cấp bậc:**
  - Lục Chuyển: Địa Tai (10 năm), Thiên Kiếp (100 năm)
  - Thất Chuyển: Địa Tai (10 năm), Thiên Kiếp (50 năm), Hạo Kiếp (100 năm)
  - Bát Chuyển: Thiên Kiếp (10 năm), Hạo Kiếp (50 năm), Vạn Kiếp (100 năm)
  - Cửu Chuyển: Hạo Kiếp (10 năm), Vạn Kiếp (50 năm), Hỗn Độn Đại Nạn (100 năm)
- **Tự động đột phá:** Khi vượt thành công đủ 3 lần kiếp nạn cấp cao nhất của Chuyển hiện tại (Rank 6: 3 Thiên Kiếp; Rank 7: 3 Hạo Kiếp; Rank 8: 3 Vạn Kiếp) và đạt cảnh giới lưu phái yêu cầu, Cổ Tiên tự động lên Chuyển kế tiếp.
- **Vĩnh Sinh (Rank 10): không còn tai kiếp giáng xuống** — worker bỏ qua lên lịch kiếp cho cảnh giới này.
- Nhiều kiểu kiếp nạn đến cùng lúc: ưu tiên sinh kiểu hiếm nhất, nhưng đếm tất cả các kiểu đang đến hạn.

## 3. Không khiếu, tiên khiếu và đạo ngân

### Không khiếu/phàm nhân

- UI `Aperture` hiển thị Chân Nguyên, Sinh lực, Chuyển, tư chất, Cổ trang bị, nhiệm vụ và tu luyện.
- Chân Nguyên/Sinh lực được frontend ước tính theo thời gian từ snapshot server.

#### Hồi máu thụ động ngoài combat

- Sinh lực hồi liên tục theo thời gian thực, được cộng dồn cả khi offline (tính từ `last_mana_update`, áp dụng khi đạo hữu tương tác).
- Trong **Túi Đồ → Cổ Trùng**, chọn Cổ loại Hỗ Trợ rồi bấm **Dùng ngoài combat** để kích hoạt hồi máu từ `active.heal_flat`; mỗi lần dùng tốn Chân Nguyên và giảm 1 độ no của Cổ.
- **Tốc độ hồi:** `HP/giờ = (HP tối đa hiệu dụng × 5% + hp_regen từ effect) × (1 + healing_received)`, cap tại HP tối đa hiệu dụng (`base_hp` + `stat_base_hp` thụ động từ Cổ).
- Toàn bộ hồi tự động HP, Chân/Tiên Nguyên, Niệm Lực và Thần Hồn được tổng hợp từ các block của Chủng tộc, Thể chất, Trạng thái và Cổ `ON_TICK`, rồi mới áp dụng hệ số cho phép hồi. Với Cương Thi hệ số cuối bằng `0`, nên cả hồi nền lẫn Cổ hồi thụ động đều không sinh tài nguyên; hồi chủ động bằng vật phẩm/Cổ tiêu hao vẫn hoạt động.
- `hp_regen` (số phẳng HP/giờ) và `healing_received` (% nhân) đến từ thể chất/chủng tộc/trạng thái qua hệ thống effects (xem `docs/GAME_DATA.md`).
- Tốc độ hồi nhân thêm `TimeSpeedMultiplier` (KHÔNG nhân `time_rate` Tiên khiếu — khác với hồi Chân Nguyên).

### Nhân Khí (Human Qi)

- Nhân Khí là tài nguyên phản ánh hoạt động tích cực của đạo hữu trong giới tu tiên, giới hạn tối đa **500.000**.
- Mọi hoạt động lớn đều tích lũy Nhân Khí (tăng theo `human_qi`, lưu trực tiếp vào DB):
  - Chiến đấu: thắng +20, thua +5; tự động chiến +3; rút lui +2, bỏ chạy +1.
  - Mua/bán tại cửa hàng +5; đổi vật phẩm đấu trường +5.
  - Chợ: niêm yết +5, đặt giá +3, mua thành công +5 (cả người mua lẫn người bán).
  - Giao dịch trực tiếp: tạo giao dịch +2, cập nhật đề nghị +1, khóa +1, xác nhận +10 mỗi bên.
  - Khám phá: đi tới ô mới +1, thám hiểm node +3.
  - Cổ trùng: cho ăn +2, dùng Cổ tiêu hao/hỗ trợ +3.
  - Luyện Cổ +10, suy diễn công thức +5.
  - Hồn phách: kiên cố +3, tinh luyện +5.
  - Môn phái: quyên góp nguyên thạch/vật liệu/Cổ +5, mua từ kho +5, gia nhập +3, chiếm mỏ +5.
  - Phó bản: vào phó bản +2, mỗi lựa chọn hành động +3.
  - Tu luyện: phá vách +1, vượt kiếp +5, resolve tai kiếp +10, xả stress +5.
  - Mở đá: mở 1 lần +1, mở 10 lần +10.
  - Thành tựu: nhận thành tựu +5.
  - Nhiệm vụ hoàn thành +5, nhập mã quà +2.
  - Sát chiêu mới +5; hiến tế +10.
  - Party: tạo đội +2, mời vào đội +2.
  - Tiên khiếu: xây node +5, thu hoạch +2.
  - Mua gói quý +10.
  - Chat: mail riêng +1, Bảo Hoàng thị +1.
- Nhân Khí dùng để xét điều kiện Thăng Tiên (Ngũ Chuyển Đỉnh phong) — xem mục Thăng Tiên.

### Tiên khiếu

- Tiên khiếu có phẩm cấp, time rate, Sinh lực, diện tích, tiên nguyên, sản lượng/năm, state.
- **Phẩm cấp được quyết định ngay lúc thăng tiên**, dựa trên Nhân Khí tại thời điểm đó (`q = NhânKhí / 100.000`). Phẩm cấp cố định sau khi khai mở, không thể đổi.
- **Tỉ lệ ra phẩm cấp theo Nhân Khí:**

  | Nhân Khí | EXTREME | HIGH | MID | LOW |
  |---------|---------|------|-----|-----|
  | 0 | 4.8% | 14.3% | 23.8% | 57.1% |
  | 25.000 | 16.7% | 17.9% | 23.8% | 42.9% |
  | 50.000 | 26.2% | 21.4% | 23.8% | 28.6% |
  | 75.000 | 36.9% | 25.0% | 23.8% | 14.3% |
  | 100.000 | 47.6% | 28.6% | 23.8% | 0% |

- **Thuộc tính theo phẩm cấp:**
  - LOW: time rate 10-15x, Sinh lực 10.000, 10 tiên nguyên/năm.
  - MID: time rate 20-28x, Sinh lực 20.000, 20 tiên nguyên/năm.
  - HIGH: time rate 30-40x, Sinh lực 30.000, 30 tiên nguyên/năm.
  - EXTREME: time rate 40-55x, Sinh lực 50.000, 50 tiên nguyên/năm.
- **Khi mới khai mở**, Tiên Khiếu được tạo sẵn 2-4 mỏ tài nguyên ngẫu nhiên (tuỳ phẩm cấp):
  - LOW: 2 mỏ, MID: 3 mỏ, HIGH/EXTREME: 4 mỏ.
  - Các mỏ khởi đầu: Nguyệt Lan, Thanh Trúc, Ngọc Thạch, Nguyên Thạch, Kiếm Diệp, Hươu Linh.
- Bản đồ Tiên Khiếu là lưới 20×20. Mỗi node trồng trọt, chăn nuôi hoặc khai thác có thể chiếm một vùng ô liền nhau; nhiều node cùng loại được phép đặt, nhưng mỗi ô chỉ thuộc một node.
- Sản lượng node tỉ lệ với số ô được giao. Khi lấp kín 400 ô, tổng sản lượng được hiệu chỉnh còn 25% mức tối đa của giới hạn node cũ; các hệ số path, Đạo Ngân, sinh thái, lao động và phẩm vật vẫn áp dụng.
- Xung đột Đạo của sản lượng dùng chung hồ sơ combat theo path, Đạo Ngân, path mastery và hiệu ứng giảm xung đột; vị trí node không tạo luật tương khắc riêng.
- Cổ sư có thể thu hoạch, gỡ/xóa và sửa chữa từng node/vùng.
- Chế độ **Chọn nhiều node** trên bản đồ cho phép sửa chữa, nâng cấp hoặc san phẳng nhiều node cùng lúc; tổng phí sửa/nâng cấp được kiểm tra trước khi áp dụng.
- **Thu hoạch tất cả** các mỏ đã đủ thời gian tích tụ cùng lúc; sản lượng giảm **20%** và Nhân Khí nhận được giảm còn **+1/mỏ** (so với thu hoạch từng mỏ).
- **Sửa chữa tất cả** các công trình còn tồn tại nhưng bị hư tổn cùng lúc; phí sửa mỗi công trình tăng **30%**. Công trình bị phá hủy hoàn toàn sẽ mất vĩnh viễn và phải xây lại.
- Hệ sinh thái vẫn kiểm tra cân bằng trồng trọt, chăn nuôi và thú ăn thịt.
- Dị nhân trong tiên khiếu có population, happiness, oppression, rebellion rate và labor intensity.

### Công Trình Phòng Thủ

Ngoài các mỏ tài nguyên, đạo hữu có thể xây dựng **công trình phòng thủ** để bảo vệ Tiên Khiếu trước thiên kiếp. Mỗi công trình yêu cầu **Tiên Nguyên (Đá)** + **Cổ trùng** (tiêu hao vĩnh viễn).

| Công Trình | Chi Phí | Sinh Lực | Hiệu Ứng |
|---|---|---|---|
| 🧱 **Tường Thành** | 800 Đá + 2 Cổ (Thú Bì, Thạch Bì Nhất Chuyển trở lên) | 3.000 | Chặn kẻ thù di chuyển qua ô. Kẻ thù phải phá hủy tường trước khi đi tiếp. |
| 🗼 **Tháp Canh** | 1.200 Đá + 2 Cổ (Thiết Bì, Bán Thạch Nhị Chuyển trở lên) | 2.000 | Bắn kẻ thù trong phạm vi 3 ô, gây 5 sát thương/kỳ (15 giây). |
| 🗿 **Tượng Hộ Mệnh** | 1.500 Đá + 2 Cổ (Bạch Ngọc, Nguyệt Nghê Nhị Chuyển trở lên) | 2.500 | Tăng +500 Sinh lực cho các công trình trong phạm vi 2 ô. |
| ⚔️ **Doanh Trại** | 2.000 Đá + 2 Cổ (Kim Chung, Tượng Nha Tứ Chuyển trở lên) | 4.000 | Giảm 15% sát thương cho TẤT CẢ công trình trong Tiên Khiếu. |
| 🔮 **Trường Lực** | 2.500 Đá + 2 Cổ (Chí Tôn, Kim Chung Tứ Chuyển trở lên) | 1.500 | Giảm 30% sát thương cho công trình trong phạm vi 2 ô. |

**Lưu ý quan trọng:**
- Cổ trùng bị **tiêu hao vĩnh viễn** khi xây dựng (không thể thu hồi).
- Phép giảm sát thương cộng dồn tối đa **80%**.
- Tường Thành là chướng ngại vật bắt buộc — kẻ thù không thể đi xuyên qua mà phải phá hủy trước.
- Tháp Canh gây sát thương trực tiếp lên Sinh lực kẻ thù mỗi 15 giây.
- Công trình phòng thủ **không sản xuất tài nguyên** — chỉ có hiệu ứng phòng thủ.

### Gia tốc thời gian (Time Rate)

- `time_rate` (1.0x - 100.0x) điều chỉnh tốc độ thời gian bên trong Tiên Khiếu. Phẩm cấp Tiên Khiếu quyết định `time_rate` khởi tạo: **LOW 10–15×, MID 20–28×, HIGH 30–40×, EXTREME 40–55×**.
- **1 năm nội bộ = 36.000 giây đời thật ÷ `time_rate`** (hằng số 36.000 giây/năm trong `tribulation.go` / `aperture.go`).
- **Hồi Tiên Nguyên nền:** mỗi năm nội bộ, Tiên Khiếu sản xuất lượng `essence_production_per_year` theo phẩm cấp. Với `time_speed_multiplier = 1`, tốc độ nền mỗi giờ đời thật là `essence_production_per_year ÷ 10 × time_rate`; không có hệ số tiểu cảnh giới. Cổ, thể chất, tuổi thọ và hiệu ứng khóa hồi phục có thể thay đổi lượng thực nhận.
- **Kiếp nạn đầu tiên** giáng lâm vào năm nội bộ thứ 10 → thời gian đời thật chờ = `10 × 36.000 ÷ time_rate` giây:

| Phẩm cấp Tiên Khiếu | `time_rate` | Kiếp đầu tiên (đời thật) |
|---|---|---|
| LOW | 10–15× | 6h40m – 10 giờ |
| MID | 20–28× | 3h34m – 5 giờ |
| HIGH | 30–40× | 2h30m – 3h20m |
| EXTREME | 40–55× | 1h49m – 2h30m |

- **Ảnh hưởng đến:**
  - Tốc độ thu hoạch tài nguyên (cây, quặng, linh thú)
  - Tăng trưởng dân cư dị nhân
  - Tốc độ hồi Tiên Nguyên (Chân Nguyên của Cổ Tiên)
  - Tích lũy năm nội bộ → Kiếp nạn đến nhanh hơn
  - Đói Cổ trùng (Cổ đói nhanh hơn khi time rate cao)
- **Không ảnh hưởng đến:**
  - Thọ nguyên (luôn tính theo thời gian thực)
  - Cooldown/sự kiện bên ngoài Tiên Khiếu
  - Thời gian ở Ngũ Vực

### Gia tốc thời gian cá nhân (Time Speed Multiplier)

- `time_speed_multiplier` nhân tốc độ dòng thời gian cho mọi Cổ sư. Giá trị thực tế được lưu và dùng trong tính toán là **1.0** (chuẩn hóa trong `player_service.setDefaults`).
- **Ảnh hưởng đến (chỉ nhân `time_speed_multiplier`, không nhân `time_rate`):**
  - Tốc độ hồi Sinh lực (HP)
  - Tiêu hao Thọ nguyên (trôi nhanh hơn)
  - Tốc độ hồi Tinh hồn
  - Tốc độ hồi Ý niệm
- **Nhân cả `time_speed_multiplier` × `time_rate` (Cổ Tiên):**
  - Tốc độ hồi Chân Nguyên (Tiên Nguyên)
  - Tích lũy năm nội bộ / sản xuất tiên khí
- **Cổ ảnh hưởng `time_speed_multiplier`:** một số Cổ Trụ đạo có thể gia tốc vĩnh viễn (Cổ tiêu hao) hoặc khi trang bị (Cổ thụ động) dòng thời gian cá nhân — hồi phục nhanh hơn nhưng tuổi thọ trôi nhanh hơn.

### Đạo ngân, Xung Đột Quy Tắc & Cân Bằng Lưu Phái (Dao Mark Conflict)

Đạo Ngân là vết khắc cô đọng của quy luật tự nhiên tích lũy trên thân thể và Tiên Khiếu của Cổ Tiên qua **49 lưu phái** (tích lũy từ Độ Kiếp, Thăng Tiên, Nuốt Tiên Khiếu hoặc Luyện Cổ). Riêng Đạo Ngân Thiên Đạo không thể nhận từ Thăng Tiên hay Độ Kiếp thông thường và cần một cơ chế riêng biệt.

#### 1. Nhóm Đạo Và Độ Tương Thích

Mỗi lưu phái được chuẩn hóa thành một khóa để tính toán. Các nhóm của hệ thống hiện tại:

- **ELEMENTAL**: Kim, Mộc, Thủy, Hỏa, Thổ, Phong, Lôi, Băng, Vân, Quang, Ám, Tinh, Âm Thanh.
- **SPACE_TIME**: Không Gian, Thời Gian.
- **RULE_ORDER**: Quy Tắc, Cấm Chế, Trận; Phá Hoại cũng được xếp vào nhóm này.
- **KINETIC_WEAPON**: Kiếm, Đao, Binh Khí, Sát Đạo.
- **BODY_LIFE**: Lực, Cốt, Biến Hóa, Thực Đạo.
- **SOUL_MIND**: Hồn, Trí, Tình, Nô, Mộng.
- **INFORMATION_STEALTH**: Thông Tin, Vận, Thâu Đạo, Huyễn, Hội Họa; Điều Tra cũng được xếp vào nhóm này.
- **ORGANIC_CORRUPTION**: Huyết, Độc; Tử Đạo cũng được xếp vào nhóm này.
- **CRAFT_ALCHEMY**: Luyện, Đan; Sáng Tạo cũng được xếp vào nhóm này.
- **HEAVEN_HUMAN**: Thiên, Nhân, Khí.

Cùng đúng một lưu phái có độ tương thích **1,00**. Hai lưu phái khác nhau không mặc định tương thích hoàn toàn, kể cả khi cùng nhóm. Ma trận nhóm hiện cho độ tương thích khoảng **0,80–0,96** trong cùng nhóm; cặp khác nhóm dùng giá trị riêng trong ma trận. Lưu phái chưa có trong ma trận dùng giá trị dự phòng **0,85**.

Một số cặp có độ tương thích ghi đè:

| Cặp lưu phái | Tương thích | Phần xung đột (1 − tương thích) |
| --- | ---: | ---: |
| Hỏa – Thủy | 0,55 | 0,45 |
| Hỏa – Băng | 0,60 | 0,40 |
| Quang – Ám | 0,55 | 0,45 |
| Độc – Thực Đạo | 0,60 | 0,40 |
| Không Gian – Thời Gian | 0,97 | 0,03 |
| Trí – Thông Tin | 0,98 | 0,02 |
| Lực – Cốt | 0,96 | 0,04 |
| Kiếm – Đao | 0,95 | 0,05 |
| Luyện – Đan | 0,97 | 0,03 |

#### 2. Cách Tính Xung Đột Cho Một Hành Động

Hệ thống tính riêng theo lưu phái của từng hành động:

1. **Chọn lưu phái thi triển**: lấy lưu phái đầu tiên được gắn với kỹ năng/Cổ. Nếu hành động không khai báo lưu phái, mặc định là **Strength (Lực Đạo)**. Vì vậy đánh thường thường tính theo Strength; mang Cổ Thủy không tự biến đòn đánh thường thành đòn Thủy.
2. **Tính ngưỡng bỏ qua**: cộng toàn bộ Đạo Ngân dương thành Tổng. Ngưỡng bằng max(1.000, 2% × Tổng). Đạo Ngân ngoại hệ chỉ được đưa vào tính xung đột khi **lớn hơn** ngưỡng; bằng ngưỡng vẫn được bỏ qua.
3. **Tính xung đột thô**: với mỗi phái ngoại hệ đang hoạt động, cộng Đạo Ngân ngoại hệ × (1 − độ tương thích với phái thi triển).
4. **Tính số phái hiệu dụng**: mỗi phái ngoại hệ đóng góp 2 × (1 − độ tương thích). Khi Chân Ý của phái thi triển đạt Tông Sư trở lên, ma sát của cặp tương thích từ 0,90 trở lên giảm một nửa.
5. **Phạt đa tu**: nhiều phái ít tương thích làm xung đột tăng theo F_multi = 1 + (N_eff − 1)^1,3 × (1 − P)^2 × 4, trong đó P = Đạo Ngân của phái thi triển / Tổng Đạo Ngân. Đây là độ thuần của **phái đang dùng**, không phải phần trăm Đạo Ngân chủ đạo hiển thị trên hồ sơ.
6. **Điều hòa bằng Chân Ý**: dùng cảnh giới riêng của lưu phái đang thi triển:

| Chân Ý của phái đang dùng | Mốc điểm | Giảm xung đột |
| --- | ---: | ---: |
| Phàm Thai | 0 | 0% |
| Đại Sư | 10.000 | 10% |
| Tông Sư | 100.000 | 25% |
| Đại Tông Sư | 1.000.000 | 45% |
| Vô Thượng Đại Tông Sư | 10.000.000 | 100% |

7. **Chuẩn hóa và tính hệ số**:
   - Xung Đột Hiệu Dụng = Xung Đột Thô × F_multi × (1 − mức điều hòa)
   - Mẫu Số = Đạo Ngân của phái đang dùng + max(1.000, 10% × Tổng)
   - conflictRatio = Xung Đột Hiệu Dụng / Mẫu Số
   - M = max(0,08, 1 / (1 + conflictRatio × 1,6))

Hệ số M nằm trong [0,08; 1,00]. Nó nhân vào hệ số Đạo Ngân của đòn: DaoMult = (1 + (DaoAmplification − 1) × RealmEfficiency) × M. Khuếch đại riêng của một phái là 1 + Đạo Ngân phái đó / 100; Đạo Ngân Hỏa không khuếch đại đòn Thủy hay Strength, nhưng có thể gây xung đột với chúng.

#### 3. Hậu Quả Trong Combat

- **Tốn mana hơn** nếu hành động có mana cost: ManaCost × [1 + (1 − M) × 1,5]. Ở mức xung đột nặng nhất (M = 0,08), mana tốn tối đa **2,38 lần**.
- **Kỹ năng tán loạn**: chỉ kỹ năng Cổ (GU_SKILL) và sát chiêu (KILLER_MOVE) bị kiểm tra khi M < 0,70. Xác suất = min(25%, (0,70 − M) / (0,70 − 0,08) × 25%), tối đa **25%** tại M = 0,08. Mana đã trừ trước khi gieo; nếu kỹ năng tán loạn thì lượt đó dừng và không kiểm tra phản phệ.
- **Phản phệ nội thương**: khi M < 0,60, xác suất = min(30%, (0,60 − M) / (0,60 − 0,08) × 30%), tối đa **30%** tại M = 0,08. Kiểm tra này không giới hạn ở kỹ năng Cổ: đánh thường Strength cũng có thể phản phệ nếu hệ số Strength thấp đủ mức.
- **HP mất khi phản phệ**: trừ trực tiếp **8–12% HP tối đa**, không qua phòng thủ hoặc khiên ảo. Đòn đánh phát sinh phản phệ còn bị giảm thêm 50% sát thương.
- **Bạo thể vong mạng**: không có một lần gieo xác suất chết riêng. Nhân vật chết nếu HP còn 0 sau khi trừ máu phản phệ. Một lần phản phệ thường không giết người đang đầy máu, nhưng có thể kết liễu người đã yếu; nhiều lần phản phệ làm nguy cơ cộng dồn. Bất Hoại chặn được sát thương; Bất Tử giữ tối thiểu 1 HP, trừ khi Sát Khí đủ điều kiện xuyên miễn nhiễm.
- **Kháng Đạo Ngân tự nhiên khi phòng thủ**: khi nhận đòn từ phái p, hệ thống cộng Đạo Ngân phòng thủ theo `DefConflict = Σ(ĐạoNgân_q × (1 − TươngThích(p,q)))`, rồi tính `Kháng = 0,30 × DefConflict / (DefConflict + 50.000)`. Kháng tăng dần và tiến gần tối đa 30%; Cổ miễn xung đột cũng bỏ qua lớp kháng này. Cơ chế phòng thủ này riêng với phản phệ tự trừ HP của người ra đòn.
- **Miễn xung đột**: Cổ đang ACTIVE và trang bị có PATH_CONFLICT_REDUCTION từ 100 trở lên miễn hình phạt này. Chân Ý Vô Thượng hóa giải xung đột khi hành động dùng đúng lưu phái đã đạt cảnh giới đó.

#### 4. Ví Dụ Cụ Thể

Các ví dụ giả định không có bonus khác, Chân Ý của phái đang dùng là Phàm Thai, và nhân vật không có Cổ miễn xung đột.

- **Hỏa 1.000, Thủy 0, dùng kỹ năng Thủy**: Tổng là 1.000, ngưỡng cũng là 1.000. Hỏa bằng ngưỡng nên bị bỏ qua (lượng ngoại hệ phải lớn hơn ngưỡng mới tính). M = 1,00: không tăng mana, không tán loạn, không phản phệ. Vì không có Đạo Ngân Thủy, đòn Thủy cũng không được khuếch đại bởi Đạo Ngân Thủy (DaoAmplification = 1,00); sát thương nền và bonus khác của Cổ vẫn áp dụng.
- **Hỏa 1.001, Thủy 0, dùng kỹ năng Thủy**: Hỏa vừa vượt ngưỡng. Với tương thích Hỏa–Thủy 0,55, M ≈ 0,236. Mana tốn khoảng **2,15 lần**; kỹ năng có **18,7%** nguy cơ tán loạn. Nếu qua bước tán loạn, lần kiểm tra phản phệ có khoảng **21,0%** xác suất; nếu phản phệ xảy ra thì mất 8–12% HP tối đa. Có bước nhảy rõ khi vượt ngưỡng 1.000.
- **Hỏa 1.000, Strength 0, đánh thường**: hành động không gắn phái được tính là Strength. Hỏa vẫn bằng ngưỡng nên M = 1,00, không phản phệ. Cổ Thủy đang mang không đổi phái của đòn đánh thường.
- **Hỏa 1.001, Strength 0, đánh thường Strength**: Hỏa vừa vượt ngưỡng, nhưng Strength–Hỏa chỉ xung đột 0,15 cho mỗi Đạo Ngân do tương thích 0,85. M ≈ 0,694; không bị tán loạn vì đây là đánh thường, và cũng chưa phản phệ vì M vẫn trên 0,60.
- **Hỏa 2.000, Strength 0, đánh thường Strength**: Hỏa đã hoạt động. Strength–Hỏa có tương thích mặc định 0,85 nên xung đột nhẹ hơn Hỏa–Thủy. Với các giả định trên, M ≈ 0,532. Đánh thường không bị kiểm tra tán loạn, nhưng có khoảng **4,0%** xác suất phản phệ mỗi đòn. Nếu hành động có mana cost thì mana tốn khoảng **1,70 lần**.
- **Hỏa 2.000, Strength 0, Chân Ý Strength Vô Thượng**: điều hòa 100% xung đột cục bộ nên đòn Strength có M = 1,00. Điều này không miễn xung đột cho kỹ năng Thủy nếu Chân Ý Thủy chưa đạt Vô Thượng.

**Suy Thoái Sinh Thái Tiên Khiếu**: Độ Thuần Khiết $P = \text{DominantMarks} / \text{TotalMarks}$. Khi $P < 0.70$, sản lượng mỏ tài nguyên và Tiên Nguyên tự nhiên giảm theo
$$\text{EcologyPurityFactor} = \max\left(0.25,\ 1.0 - (0.70 - P) \times 1.25\right)$$
(Chí Tôn Tiên Thai / Đạo Ngân Bất Tương Xung miễn nhiễm suy thoái này.)

**Kiếp Nạn Hỗn Loạn (Chaotic Tribulation)**: kích hoạt khi $P < 0.60$ và $N_{\text{eff}} \ge 2.5$. Tai kiếp lai tạp 2–3 lưu phái, quái được khuếch đại `1.0 + (1.0 − P) × 0.75` HP/Công, phá node mạnh hơn, và thưởng Đạo Ngân rải đều theo các hệ thay vì dồn 70% phái chính — tạo **Vòng Xoáy Tha Hóa** nếu không chủ động thanh tẩy.

**Thanh Đạo Tẩy Thể**: dùng Tiên Nguyên Thạch để bóc tách Đạo Ngân ngoại hệ, nâng cao Độ Thuần Khiết:
- Chi phí: **1 Tiên Nguyên Thạch / 100 Đạo Ngân tạp**.
- Hoàn trả **30%** giá trị dưới dạng Đạo Tài của lưu phái tương ứng.
- Bế quan hồi phục **12 giờ** giữa các lần, tối đa **10.000 Đạo Ngân/lần**. Không thể tẩy phái chủ đạo.

## 4. Cổ trùng và bọc đồ

Hiệu ứng chiến đấu đặc thù của Cổ trùng được lấy từ cột `effects` trong danh mục Cổ. ID của Cổ chỉ dùng để nhận diện, hiển thị, cấu hình thứ tự ra chiêu và tính tiêu hao; nó không tự kích hoạt một hiệu ứng ẩn. Vì vậy một Cổ mới có thể tái sử dụng mechanic hiện có chỉ bằng dữ liệu, còn mechanic mới phải được đăng ký và kiểm tra hợp lệ trước khi import.

### Sáng tạo Sát Chiêu

Trong Túi Đồ, mở tab **Sát Chiêu**, chọn một Cổ chính và thêm Cổ phụ tùy ý. Kéo các hiệu ứng được hỗ trợ từ Cổ thành phần vào công thức, xem trước uy lực sau cân bằng, độ hòa hợp và tỉ lệ thành công, rồi đặt tên và lưu. Người chơi lưu tối đa 12 công thức và trang bị tối đa 4 sát chiêu.

Rank Cổ chính quyết định phần lớn ngân sách sức mạnh; Cổ phụ bổ sung ngân sách với mức giảm dần. Số lượng, phạm vi và độ phức tạp effect làm tăng tải công thức. Khi tải vượt ngân sách, hệ thống tự giảm giá trị effect và hiển thị kết quả trước khi lưu. Cấp lưu phái và độ thành thạo ảnh hưởng tỉ lệ thôi động; dùng thành công giúp tăng thành thạo sát chiêu.

Sát chiêu đã trang bị tự tham gia auto-combat. Effect hồi phục `ON_USE` có thể được dùng ngoài combat bằng nút **Dùng**. Mỗi lần thử thi triển đều tốn Chân Nguyên và giảm 1 độ no của từng Cổ thành phần; thất bại không làm mất Cổ. Một instance Cổ có thể tham gia nhiều sát chiêu. Nếu instance đó rời `player_gu`, chết hoặc đổi thành Cổ/rank khác, những công thức tham chiếu nó sẽ tạm thời không dùng được.

Mỗi Cổ instance có:

- `instance_id`, `gu_id`.
- Độ No (giảm theo `HungerRate * TimeSpeedMultiplier` mỗi 3 giờ thực tế, Cổ trong Tiên Khiếu có time rate cao sẽ đói nhanh hơn).
- Status: `ACTIVE`, `SLEEPING`, `DEAD`.
- `equipped` và `is_vital`.

Khi Cổ chết đói, **Ký Sự Tu Hành** ghi bản rõ: độ no ngay trước khi chết, trạng thái lúc đó (`ACTIVE`/`SLEEPING`/`SEALED` — quyết định tốc độ hao hụt), lần cuối được cho ăn và tổng thời gian không được cho ăn của cá thể đó.

Cổ sư có thể:

- Cho ăn từng Cổ hoặc theo nhóm.
- Ngoài vật liệu, một số Cổ đặc thù ăn trực tiếp tài nguyên của chủ nhân: Huyết Thọ Cổ dùng 10 Máu, Nhiếp Tâm Cổ dùng 10 Niệm Lực, Khấp Thương Cổ dùng 10 Linh Hồn và Thâu Mệnh Tiên Cổ dùng 1 Tuổi Thọ cho mỗi khẩu phần 100 Độ No. Chi phí này cố định ở mọi Chuyển; hệ thống không cho Máu, Linh Hồn hoặc Tuổi Thọ bị ăn xuống 0.
- Trang bị/tháo Cổ.
- Đặt Cổ bản mệnh.
- Đổi Bản Mệnh Cổ khi đã có một Cổ khác sẽ gây phản phệ Trọng Thương. Mức giảm công/thủ/tốc và tỉ lệ luyện cổ cùng thời hạn tăng theo rank/subrank như Áp Chế Giới Vực: rank 1 bắt đầu ở 10% trong 5 giờ, rank 9 có thể tới 90% trong 40 giờ. Hồi phục nhận giảm 50% trong thời gian này.
- Chuyển trạng thái active/sleeping.
- Dùng Cổ tiêu hao hoặc Cổ hỗ trợ.
- Vứt bỏ Cổ.

**Tiên Cổ duy nhất:** Cổ từ **Lục Chuyển trở lên (Tiên Cổ)** là độc nhất toàn thiên hạ — mỗi `gu_id` chỉ tồn tại một instance trên toàn server. Các kênh nhận Cổ (Mở đá, Rương Mỏ Linh Thạch, Luyện Cổ, quái rơi đồ) đều loại trừ Tiên Cổ đã tồn tại (trong túi bất kỳ đạo hữu, đấu giá) trước khi roll; nếu hết ứng viên hợp lệ sẽ rơi vật phẩm/thạch bù trừ. Mỗi Tiên Cổ có một mã hiệu (`instance_id`) theo nó suốt đời — mua bán, trao đổi, gửi kho tông môn đều giữ nguyên mã hiệu. Thả, bỏ đói chết hay mất khi bại trận là kết thúc vòng đời mã hiệu đó; sau đó thiên hạ mới có thể xuất hiện lại Cổ cùng loại với mã hiệu mới. Lịch sử đời Cổ xem ở trang cá thể.

**Hiệu ứng Trưởng thành (Growth):** Một số Cổ (Lực đạo/Thổ đạo như Hùng Lực, Hắc Thỉ, Ngọc Bì, Thiết Cốt, Tự Lực...) có hiệu ứng `growth` — khi trang bị ở trạng thái ACTIVE, chỉ số thưởng (Công/Thủ/Máu/Tốc) tăng dần theo thời gian tới mức tối đa (`hours_to_max` giờ). Quy tắc:

- **Chỉ tăng 1 lần cho mỗi loại (GU ID + Chuyển):** dù sở hữu nhiều bản cùng loại, tiến độ và chỉ số chỉ tính 1 lần; không cộng dồn theo số bản sở hữu.
- **Tốc độ trưởng thành nhân `time_speed_multiplier`:** thời gian trưởng thành được rút ngắn theo tốc độ thời gian cá nhân (vd Cổ Tiên tsm 10 → trưởng thành nhanh gấp 10). Không nhân `time_rate` Tiên Khiếu.
- **Tiến độ vĩnh viễn:** tiến độ tích lũy được lưu theo từng instance; gỡ Cổ ra khỏi Không Khiếu vẫn giữ nguyên tiến độ và hiển thị lại ở tab **Trưởng thành** (Không Khiếu). Khi đạt mức tối đa, loại Cổ đó được ghi nhận vĩnh viễn trong danh sách trưởng thành.
- Trang bị lại tiếp tục từ tiến độ đã tích lũy, kể cả khi đổi sang bản khác cùng loại.

## 4.1. Cân bằng: Quái vs Cổ trùng cùng Rank

> **Không có Cổ trùng → người chơi tự động thua quái ở mọi rank.**

Quái và Cổ trùng **cùng rank là cân bằng** (mất ~3–5 hit để hạ nhau). Không có Cổ, người chơi yếu hơn rất nhiều so với quái cùng rank.

### Quái base 100 HP / 10 ATK / 5 DEF (sau ScaleMonsterStats)

| Rank | MaxHP | ATK | DEF |
|------|-------|-----|-----|
| 1 | 100 | 10 | 5 |
| 2 | 200 | 20 | 10 |
| 3 | 500 | 50 | 25 |
| 4 | 2,000 | 200 | 100 |
| 5 | 10,000 | 1,000 | 500 |

### Cổ trùng damage/shield (từ Gu.csv)

| Rank | Damage | Shield |
|------|--------|--------|
| 1 | 20 | 30 |
| 2 | 40 | 60 |
| 3 | 100 | 150 |
| 4 | 400 | 600 |
| 5 | 2,000 | 3,000 |

### Combat cùng rank (người chơi + Cổ vs quái)

| Rank | Cổ DMG + Player ATK | Quái HP | Hit giết | Cổ Shield + Player DEF | Quái ATK | Hit chịu |
|------|---------------------|---------|----------|------------------------|----------|----------|
| 1 | 30 | 100 | 3–4 | 35 | 10 | 3 |
| 2 | 60 | 200 | 3–4 | 70 | 20 | 3 |
| 3 | 130 | 500 | 4 | 165 | 50 | 3 |
| 4 | 440 | 2,000 | 5 | 620 | 200 | 3 |
| 5 | 2,050 | 10,000 | 5 | 3,025 | 1,000 | 3 |

> **Boss (×5 type multiplier):** người chơi cần Cổ rank cao hơn 1 bậc để cân bằng.

## 5. Vật phẩm và nguyên liệu

- Dictionary vật phẩm được Admin import từ `backend/data/Item.csv`.
- Vật phẩm có type, rarity, Chuyển, path, effects, tags và description.
- Item effects hiện có gồm hồi Chân Nguyên/Sinh lực, trị liệu/hồi phục sinh lực Tiên Khiếu (`heal_aperture`), soul boost, đạo ngân, xóa trạng thái xấu và điều kiện chỉ phàm nhân/chỉ Cổ Tiên.

## 6. Luyện Cổ và suy diễn công thức

- Cổ phương được Admin import từ `backend/data/Recipe.csv`.
- Luyện Cổ dùng `/api/refine` với recipe và selected Gu instances.
- Cổ nguyên liệu trong công thức yêu cầu **bằng hoặc cao hơn bậc quy định**. Dùng Cổ nguyên liệu bậc cao hơn sẽ tăng thêm tỉ lệ thành công (Chất lượng nguyên liệu).
- Có thể kéo-thả Cổ bất kỳ vào ô nguyên liệu (kể cả ô "Cổ Trùng Ẩn"). Nếu bộ Cổ không khớp công thức, khi Luyện sẽ **thất bại chắc chắn** — nguyên liệu, tài nguyên vẫn bị tiêu hao.
- Tàn phương (recipe chưa 100%) phải dùng `/api/refine/deduce` để suy diễn lên 100% mới được luyện (không còn Ép Luyện). Tiêu hao 15 Niệm Lực và Nguyên Thạch mỗi lần.
  - **Độ khó suy diễn:** Tiến độ (Improvement %) mỗi lần bấm sẽ bị chia cho hệ số Rank của Cổ (Rank 5 chia 100, Rank 6 chia 1.000, Rank 9 chia 10.000.000). Phàm nhân gần như không thể cày chay Tiên Cổ Phương.
  - **Chìa khóa (Cảnh Giới & Đạo Ngân):** Bù đắp độ khó khổng lồ này bằng Cảnh Giới Trí Đạo (x3), Luyện Đạo (x2), Hệ đích (x1) và Đạo Ngân Trí Đạo (Tăng tiến độ theo hàm Căn bậc hai). Trí Đạo Vô Thượng Đại Tông Sư có thể suy tính nhanh gấp hàng vạn lần kẻ phàm. Mức Cảnh Giới Trí Đạo cũng gia tăng tỷ lệ Linh quang chớp lóe (Inspiration Flash - Lập tức hoàn thiện 100%).
### Tỉ lệ Luyện Cổ và Hệ quả

Tỉ lệ thành công của mỗi lần Luyện Cổ được tính dựa trên các yếu tố sau:
1. **Tỷ lệ gốc:** Từ bí phương.
2. **Độ hoàn thiện:** Bí phương phải thôi diễn lên 100% mới được luyện (không còn Ép Luyện).
3. **Độ chính xác tổ hợp:** Lắp sai Cổ nguyên liệu lập tức tỷ lệ về 0%.
4. **Chất lượng nguyên liệu:** Dùng Cổ nguyên liệu bậc cao hơn bậc yêu cầu (+2% mỗi bậc chênh lệch).
5. **Đạo Ngân Luyện Đạo:** Tích lũy từ Tiên Khiếu. Không giới hạn hiệu quả (unlimit), tăng theo hàm căn bậc hai (càng nhiều càng tăng chậm dần), nhân với hệ số cảnh giới lưu phái đích của Cổ đang luyện (`1.0 + Điểm Chân Ý`) và chia cho hệ số scale bậc Cổ tương ứng. Công thức: `(0.632% * √Đạo Ngân * (1.0 + Điểm Chân Ý)) / Hệ số Rank`.
   - Hệ số Rank P: `[R1: 1, R2: 2, R3: 5, R4: 20, R5: 100, R6: 10000, R7: 1000000, R8: 100000000, R9: 10000000000]`
6. **Cảnh giới Luyện Đạo:** Tăng tỉ lệ thành công (+5%, +10%, +18%, +25% tương ứng với cảnh giới từ Đại Sư đến Vô Thượng Đại Tông Sư).
7. **Cảnh giới Lưu Phái đích:** Cảnh giới lưu phái của Cổ đang luyện (tối đa +10% ở Vô Thượng Đại Tông Sư, cụ thể: +2%, +5%, +8%, +10% ở các cấp cảnh giới tương ứng).
8. **Khí vận:** Dựa trên Đạo Ngân Vận Đạo và hiệu ứng Phúc Trạch (tối đa +10%).
9. **Hỗ trợ:** Chủng tộc/thể chất dương (vd Người Lông +10%). Bonus Cổ phụ trợ (`success_rate_bonus`) đã bị loại bỏ, Cổ trang bị không còn cộng tỉ lệ luyện.
10. **Môi trường:** Luyện Cổ tại Đại Vực có thuộc tính phù hợp (+5%).
11. **Tiên Cổ Giảm Trừ (Immortal Penalty):** Khi luyện Tiên Cổ (Rank 6 - 9), các bonus dương được chỉ định trong công thức sẽ bị chia theo độ khó của rank Tiên Cổ.
12. **Tiêu hao Năng Lượng:** Luyện Cổ tiêu tốn sinh lực khiếu; mức tiêu hao phụ thuộc rank Cổ mục tiêu và chênh lệch cảnh giới.

Hệ quả của quá trình luyện phân thành 4 bậc:
- **Thành Công:** Nhận Cổ trùng mục tiêu và tăng điểm **Chân Ý lưu phái Luyện Đạo** (Luyện Cổ không tăng Chân Ý lưu phái khác; Chân Ý các hệ khác tích lũy qua sát chiêu, chiến đấu, phó bản):
  - **Cổ Rank 1:** +2 điểm (Chỉ tăng khi dưới 1.000 điểm / Tông Sư).
  - **Cổ Rank 2 & 3:** Rank 2 (+5 điểm), Rank 3 (+15 điểm) (Chỉ tăng khi dưới 10.000 điểm / Đại Tông Sư).
  - **Cổ Rank 4 & 5:** Rank 4 (+40 điểm), Rank 5 (+100 điểm) (Tăng lên tới 100.000 điểm / Vô Thượng).
  - **Tiên Cổ (Rank 6+):** Rank 6 (+500 điểm), Rank 7 (+2.500 điểm), Rank 8 (+15.000 điểm), Rank 9 (+50.000 điểm). Điểm gốc sẽ giảm trừ tương ứng khi Cảnh giới Luyện Đạo tăng cao.
- **Thất Bại (Safe):** Nếu thất bại suýt sao và có Cảnh giới Luyện Đạo cao, đạo hữu được bảo toàn Cổ nguyên liệu chính, chỉ mất 50% nguyên liệu phụ (materials).
- **Thất Bại (Normal):** Mất 100% nguyên liệu. Nếu có Cổ Bản Mệnh tham gia thì không chết mà rơi vào Hấp Hối (Hunger = 1).
- **Cháy Nổ (Backfire):** Nếu vận rủi tột cùng (xúc xắc >= 90%, hoặc >= 75% với Tiên Cổ), Lò Luyện phát nổ! Tổn thất 50% HP tối đa và 30% Hồn Phách tối đa. Cổ nguyên liệu thường hóa thành tro bụi.

### Trạng thái Mệt Mỏi & Phản Phệ khi Luyện Cổ

Để mô phỏng tiêu hao Thần Hồn và hạn chế việc spam Luyện Cổ liên tục:
- **Điều kiện kích hoạt:** Chỉ bị ảnh hưởng khi **Luyện Tiên Cổ (Rank 6 trở lên)** hoặc **Luyện Cổ có Rank $\ge$ Rank Cảnh Giới của bản thân**. Luyện Cổ bậc thấp hơn bản thân ("cổ cùi") không làm tổn hao tâm thần và không làm dính mệt mỏi.
- **[Tâm Thần Mệt Mỏi] (`refine_fatigue`):** Nhận được sau khi Luyện Cổ thỏa điều kiện trên. Giảm **50%** tỉ lệ thành công Luyện Cổ lượt tiếp theo trong **10 phút** (áp dụng trực tiếp sau khi chặn trần 100% các điểm thưởng tích cực).
- **[Luyện Cổ Phản Phệ] (`refine_backfire`):** Xuất hiện nếu đang có `refine_fatigue` và tiếp tục một lần luyện gây mệt mỏi (Tiên Cổ hoặc Cổ có rank bằng/cao hơn rank người chơi), hoặc khi Luyện Cổ bị **Cháy Nổ (Backfire)**. Một lần luyện cấp thấp hơn không tự leo thang trạng thái. Giảm **95%** tỉ lệ thành công Luyện Cổ trong **20 phút**. Luyện tiếp trong lúc Phản Phệ sẽ gia hạn thêm 20 phút.
- **[Tâm Đắc Luyện Cổ] (`refine_insight_1..9`):** Pity theo từng rank. Chỉ lần hợp luyện có nguyên liệu khớp công thức và Cổ mục tiêu chưa tồn tại mới đủ điều kiện tăng pity khi thất bại (`+1%/tầng`, không giới hạn, vĩnh viễn). Sai nguyên liệu hoặc Tiên Cổ mục tiêu đã tồn tại thì không tăng tầng, và tầng pity không được cộng vào tỉ lệ của lần thử đó. Luyện thành công hợp lệ rank R thì reset tầng rank đó về 0; bonus chỉ áp dụng cho đúng rank (chịu Immortal Penalty như các bonus dương khác).

## 7. Sát chiêu

- Cổ sư có thể tạo sát chiêu từ Cổ hạch tâm + tối đa 8 Cổ phụ trợ.
- Preview tính hao Chân Nguyên, stability, damage, defense, cooldown/cast time.
- **Hiệu ứng dung hợp (Compound Effects):** khi Cổ core và support có các hiệu ứng tương đồng (VD: damage + bleed, defense + heal), chúng kết hợp thành hiệu ứng mới mạnh hơn thay vì cộng dồn đơn thuần:
  - `damage + bleed → xuất huyết trọng thương`: sát thương + chảy máu mỗi lượt
  - `defense + heal → khiên sinh mệnh`: khiên tự hồi phục
  - `damage + poison → độc nhận phá thể`: vết thương nhiễm độc
  - `damage + stun → chấn động`: sát thương kèm choáng
  - `movement + stealth → huyễn ảnh bộ`: tốc độ di chuyển
  - `damage + blood_drain → huyết lực`: hút máu theo sát thương
  - `summon + damage → hư ảnh đại lực`: hư ảnh chiến đấu
  - `damage + damage_multiplier → cuồng phong`: đánh liên hoàn
  - `defense + damage → phản thương kích`: phản đòn sát thương
  - `generate/replicate → tạo hóa chi lực`: tăng cường hiệu quả hỗ trợ
- **Cổ trận liên hoàn (Synergy Recipes):** nếu core và support khớp với các công thức đặc biệt, sát chiêu sẽ kích hoạt hiệu ứng cộng hưởng riêng:
  - Đại Lực Thần Quyền (GU_CHARGING_CRASH + GU_TEN_JUN_STRENGTH + GU_JUN_STRENGTH + GU_SELF_STRENGTH): xuyên phá 50% giáp
  - Âm Dương Lưỡng Nghi (GU_MOONLIGHT + GU_LITTLE_LIGHT): sát thương gấp đôi
  - Huyết Sát Trận (GU_BLOOD_MOON + GU_BLOOD_ASSET + GU_BLOOD_SKULL): tiêu hao 10% Sinh lực, tăng 50% sát thương
  - Huyền Ảnh Sát (GU_BLOOD_MOON + GU_SHADOW + GU_SHADOW_STEPS): tấn công từ bóng tối không thể né
  - Ngũ Hành Sơn Hà (GU_CANOPY + GU_IRON_BONE + GU_EARTH_WORM): phòng ngự +200%, miễn khống chế
  - Tứ Hải Tửu Trận (GU_LIQUOR_WORM + GU_FOUR_FLAVORS_LIQUOR + GU_SEVEN_FRAGRANCES_LIQUOR + GU_NINE_EYES_LIQUOR): giảm 50% hao Chân Nguyên
- Sát chiêu đã lưu có thể cast trong combat và có thể bị xóa.

## 8. Khám phá bản đồ

- **Bản đồ thế giới tầng 1** rộng 30.000 × 20.000 ô và được phủ kín bởi 5 Đại Vực: Nam Cương, Trung Châu, Bắc Nguyên nằm thành ba vùng ở phía bắc; Tây Mạc và Đông Hải chia đôi phía nam. Mỗi Đại Vực có HUB, vùng an toàn và các KHU_VUC con; Tây Mạc có thêm 6 KHU_VUC ở phía đông, Đông Hải thêm 3 KHU_VUC ở rìa đông. Bạch Thiên thuộc tầng 2, Hắc Thiên thuộc tầng 3; lối lên hai tầng đang được chuẩn bị và chưa mở. Các địa danh con của Bạch Thiên/Hắc Thiên xuất hiện dưới dạng **Bí Cảnh Bản Đồ dùng chung toàn server**: người chơi cùng vào một bí cảnh sẽ gặp nhau trong cùng một bản đồ riêng, tách khỏi tọa độ thế giới. Mỗi bí cảnh có cổng cố định được đánh dấu trên bản đồ tầng 1; Cổ Sư phải đi tới đúng cổng mới tiến vào được. Mỗi Đại Vực tầng 1 có HUB khởi đầu riêng; khi hoàn tất thiết lập ban đầu, Cổ Sư được đưa vào HUB theo vùng đăng ký.
- Mỗi Đại Vực chia thành **tiểu vực** 1.000 × 1.000. Trong Đại Vực có các **địa danh có tên** (Thanh Mao Sơn, Tam Vương Truyền Thừa, Sa Mạc Thanh Quỷ...) với độ khó cố định theo vùng.
- Cổ sư xem lưới địa đồ xung quanh, di chuyển theo Thân pháp, chạm trán quái, thu thập tài nguyên/Cổ hoang dã; du hành giữa các Đại Vực qua màn hình bản đồ thế giới.
- **Bản đồ lớn "Ngũ Vực"** hiển thị bản đồ thế giới tầng 1 trên một canvas phóng to/thu nhỏ được: vẽ ranh giới các vùng, các điểm dùng chung (Trạm Nghỉ 🏯, Boss ⚔️, Linh Tài 💎) từ `map_cells`, **các chấm quái vật/tài nguyên xung quanh đạo hữu** (resolve theo đúng nội dung từng ô khi di chuyển), và sương mù tối vùng tiểu vực chưa ghé thăm (dựa trên `visited_subzones`). Các ô đã tải được **cache theo chunk 64×64, tách theo khu vực và prefetch dải phía trước hướng đi**; thay đổi ô realtime được apply trực tiếp thay vì tải lại toàn vùng, nên marker hiện dần (không nhấp nháy) và xuất hiện sớm ở khu vực sắp đến. Di chuyển bằng cách **giữ WASD/phím mũi tên trên máy tính hoặc kéo joystick trên thiết bị cảm ứng**; thả phím/joystick để dừng và đổi hướng bất kỳ lúc nào. Nhân vật nội suy liên tục giữa các ô với tốc độ dựa trên chỉ số **MOVEMENT**, trong khi server vẫn xác nhận từng ô và dừng lại nếu phát sinh chiến đấu hoặc sự kiện. Click/tap bản đồ chỉ dùng để chọn mục tiêu hoặc tương tác, không còn ra lệnh di chuyển. Ô trống không tự tạo phục kích procedural, nhưng quái runtime có thể phát hiện, đuổi theo và mở combat khi đạo hữu đến đủ gần mà không cần bước vào ô sinh quái. Vị trí camera + độ zoom của bản đồ được **giữ nguyên sau khi chiến đấu**.
- Trong **Cài Đặt Chung → Hiển thị trên bản đồ**, đạo hữu có thể chọn **Đầy đủ**, **Giảm nhẹ**, **Giảm nhiều** hoặc **Tắt bản đồ**. Các mức giảm bớt nhịp cập nhật dữ liệu nền; Tắt bản đồ tháo giao diện Pixi và ngừng các yêu cầu API nền của bản đồ. Lựa chọn được lưu trên trình duyệt hiện tại.
- **Quái nhỏ, tài nguyên thấp, Cổ cấp thấp** được sinh procedural theo vùng (không lưu DB), **nội dung ổn định theo ô** — quái không tự đổi chỗ/re-randomize theo thời gian; sau khi đạo hữu dọn dẹp một ô, ô đó trống trong 15 phút rồi nội dung cũ quay lại (cooldown `cleared_cells`). Các bãi có tâm và loài quái ổn định; mỗi bãi có 10% cơ hội xuất hiện một Elite ở tâm nếu pool của vùng có Elite. **Boss và tài nguyên cấp cao** là điểm dùng chung toàn thế giới trong `map_cells`, có thời gian hồi sinh sau khi bị đánh bại/hái. Boss trở thành actor dùng chung, phát hiện và đuổi người chơi trong toàn bộ ranh giới `Region` của nó, tự mở combat khi áp sát; khi mất mục tiêu boss quay về ô sinh. Boss không vào `HUB`/`SAFE_ZONE`.
- Khi trạng thái quái thời gian thực đã tải xong, bản đồ hiển thị đúng cá thể đang tồn tại thay cho dấu quái tĩnh: quái thường di chuyển rất chậm (0,25 ô/giây), phát hiện người chơi trong 2–6 ô theo Rank và đuổi trong phạm vi ngắn. Elite ở tâm bãi di chuyển nhanh hơn (0,4 ô/giây), phát hiện trong 8 ô và đuổi tối đa 18 ô quanh bãi; boss săn người chơi trong toàn vùng. Quái không thể sinh ra, chọn mục tiêu hay đi xuyên vào `HUB`/`SAFE_ZONE`; Elite và boss cũng bị giới hạn bởi ranh giới vùng. Đạo hữu còn sống sẽ tự động phản công bằng đòn đánh cơ bản sau mỗi lần bị quái đánh trúng. Bản đồ nội suy mượt giữa các vị trí server xác nhận, hiện thanh máu khi bị thương/được chọn và hiệu ứng khi ra đòn hoặc nhận sát thương.
- **Tung chiêu ngoài bản đồ:** trang bị một Cổ có hiệu ứng Block v2 `ON_ACTION` (sát thương, hồi phục, tạo khiên hoặc buff công/thủ/tốc) rồi bấm ô kỹ năng (hoặc phím `1`–`5`). Nếu đã chọn quái, chiêu sát thương được tung ngay; nếu chưa chọn, con trỏ chuyển sang chế độ nhắm mục tiêu và cần nhấp trực tiếp vào quái runtime. Mục tiêu và chiêu đang ngắm không bị hủy chỉ vì một snapshot tạm thời chưa chứa con quái đang di chuyển; khi đổi Đại Vực hoặc nhận sự kiện quái bị hạ/hủy thì lựa chọn mới được xóa. Nhấp xuống ô đất không tiêu hao mana. Cổ hồi phục, phòng thủ hoặc buff không thể gây sát thương lên quái chỉ vì một mục tiêu cũ vẫn đang được chọn.
- Nội dung mỗi vùng phụ thuộc dữ liệu do Admin import từ `backend/data/Region.csv`, `MonsterSpawn.csv` và `MapCell.csv`.
- Thanh công cụ cho phép chuyển giữa **tương tác/chọn mục tiêu** và **kéo bản đồ**, đưa camera về nhân vật, đồng thời bật/tắt nhãn Quái & Cổ/Boss/HUB/Khu vực/Ngũ Vực. Chọn một người chơi để xem hồ sơ hoặc mời tổ đội; dùng WASD/joystick để tiếp cận và chỉ có thể phát động PvP khi mục tiêu ở trong phạm vi 2 ô.
- Có thể dịch chuyển tức thời đến các vị trí bằng Cổ trùng có khả năng Dịch Chuyển / Tạo Cổng (như Định Tiên Du, Môn Vực Cổ, Thần Du Fake,...) đang trang bị và ở trạng thái ACTIVE. Cổ tạo cổng cho chọn tọa độ hoặc chọn nhanh vùng an toàn khu vực; loại cổng ngẫu nhiên tự chọn đích. Cơ chế dịch chuyển bằng tiền (Linh Thạch / Nguyên Thạch) đã bị hủy bỏ.
- **Chiến lợi phẩm mặt đất/hài cốt:** tài sản rơi khi tử vong được lưu trên ô bản đồ. Trong thời gian khóa, chỉ người hạ gục được nhặt; hết khóa, người khác có thể nhặt. Panel trên bản đồ cho xem nội dung và trạng thái khóa, người liên quan nhận thông báo khi tài sản bị nhặt.

### Hệ Thống Truyền Tống Trận (Teleportation Arrays) & Trận Đạo Lưu Phái

Cổ Sư có thể khắc họa trận văn, câu thông thiên địa để xây dựng các **Truyền Tống Trận** cố định trên bản đồ Thế Giới Ngũ Vực. Hệ thống kết nối các tọa độ không gian, hỗ trợ dịch chuyển nhanh cho cá nhân, tông môn hoặc toàn bộ tu chân giới.

#### 1. Nguyên Lý Hoạt Động & Hướng Kết Nối
- Cần tối thiểu **2 điểm truyền tống** trong tầm phủ sóng để tạo liên kết không gian.
- **Hướng Kết Nối**:
  - **Hai Chiều (`TWO_WAY`)**: Vừa phát tín hiệu truyền tống đi, vừa tiếp nhận Cổ Sư từ nơi khác đến.
  - **Một Chiều Gửi (`ONE_WAY_SEND`)**: Chỉ cho phép truyền tống rời khỏi điểm này (không thể dùng làm điểm đến).
  - **Một Chiều Nhận (`ONE_WAY_RECEIVE`)**: Chỉ tiếp nhận Cổ Sư truyền tống đến (không thể truyền tống đi từ điểm này).

#### 2. Phẩm Cấp Trận Pháp & Yêu Cầu
- **Vi Trận (Tier 1 - Phàm cấp)**:
  - *Tầm phủ sóng:* 1,000 ô (nội bộ tiểu vực/khu vực lân cận).
  - *Yêu cầu:* Tu vi Chuyển 2.
  - *Chi phí gốc:* 3,000 Nguyên Thạch + 20 Tinh Thể Vỡ Không Gian. Độ bền: 50/50.
  - *Tỉ lệ thành công cơ bản:* 70%.
- **Đại Vực Trận (Tier 2 - Linh cấp)**:
  - *Tầm phủ sóng:* 10,000 ô (bao quát toàn bộ 1 Đại Vực).
  - *Yêu cầu:* Tu vi Chuyển 4, Trận Đạo tối thiểu **Đại Sư**.
  - *Chi phí gốc:* 15,000 Nguyên Thạch + 10 Đá Vực Không Gian Vô Tận + 20 Tinh Thể Vỡ Không Gian. Độ bền: 200/200.
  - *Tỉ lệ thành công cơ bản:* 60%.
- **Tiên Trận Xuyên Vực (Tier 3 - Tiên cấp)**:
  - *Tầm phủ sóng:* Vô hạn (Truyền tống xuyên suốt Ngũ Vực).
  - *Yêu cầu:* Tu vi Chuyển 6 (Cổ Tiên), Trận Đạo tối thiểu **Tông Sư**.
  - *Chi phí gốc:* 100,000 Nguyên Thạch + 50 Tiên Nguyên Thạch + 50 Tinh Thể Vỡ Không Gian + 20 Đá Vực Không Gian Vô Tận + 10 Hư Không Thạch. Độ bền: 1,000/1,000.
  - *Tỉ lệ thành công cơ bản:* 45%.

#### 3. Cảnh Giới Trận Đạo, Tỉ Lệ Thành Công & Giảm Giá
Quá trình khắc họa trận pháp phụ thuộc mật thiết vào cảnh giới **Lưu Phái Trận Đạo (`path_masteries["formation"]`)** và **Đạo Ngân Trận Đạo**:
- **Giảm Giá Chi Phí (Cost Discount)**:
  - Nhập Môn (< 100 Chân Ý): `0%`
  - Đại Sư (≥ 100 Chân Ý): `-15%` chi phí nguyên thạch & tiên nguyên
  - Tông Sư (≥ 1,000 Chân Ý): `-30%` chi phí
  - Đại Tông Sư (≥ 10,000 Chân Ý): `-45%` chi phí
  - Vô Thượng Đại Tông Sư (≥ 100,000 Chân Ý): `-60%` chi phí
- **Tỉ Lệ Thành Công Khi Bố Trận**:
  $$\text{Tỉ Lệ} = \text{BaseRate} + (\text{Level} \times 15\%) + \max(0, \text{Rank} - \text{MinRank}) \times 3\% + \min\left(10\%, \lfloor\frac{\text{DaoMarks}}{500}\rfloor\right)$$
- **Sụp Đổ Trận Văn (Khi Bố Trận Thất Bại)**:
  - Tiêu hao 50% tiền tệ và 50% vật liệu tiêu hao do trận văn bị sụp đổ khi khắc họa.
  - Cổ trùng **KHÔNG** bị tổn hại hoặc biến mất.
  - Nhận **50% điểm Chân Ý Trận Đạo** từ kinh nghiệm thất bại.
- **Tiến Hóa Chân Ý**: Bố trận thành công cộng +30 / +100 / +300 Chân Ý Trận Đạo tương ứng Tier 1/2/3.

#### 4. Quyền Truy Cập & Bảo Mật Danh Bạ Không Gian
- **Công Cộng (`PUBLIC`)**: Mọi Cổ Sư đều có thể tra cứu trong tab *Tất Cả* của Danh Bạ và trả phí để dịch chuyển.
- **Tông Môn (`SECT`)**: Chỉ thành viên cùng môn phái mới tra cứu được trong tab *Tông Môn* và sử dụng. Không hiển thị cho người ngoài để bảo mật tuyệt đối căn cứ tông môn.
- **Riêng Tư (`PERSONAL`)**: Chỉ chủ sở hữu mới thấy trong tab *Của Ta* và sử dụng. Tọa độ được ẩn hoàn toàn trước toàn bộ người chơi khác.

#### 5. Vận Hành, Bảo Trì & Thu Hồi
- **Hao Mòn Độ Bền:** Mỗi lượt truyền tống tiêu hao 1 điểm độ bền. Khi độ bền về 0, trận pháp ngưng hoạt động cho tới khi được sửa chữa.
- **Sửa Chữa & Bạo Kích Sửa Chữa:**
  - Chi phí sửa chữa được giảm giá lên đến 50% theo cấp Trận Đạo.
  - Tỉ lệ Bạo Kích Sửa Chữa (10% ~ 40%): Phục hồi thêm +25% độ bền miễn phí và có cơ hội tăng vĩnh viễn giới hạn Max Durability nếu đạt Đại Tông Sư.
- **Thu Phí Dịch Chuyển & Két Sắt:** Chủ trận có thể đặt mức phí (Nguyên Thạch hoặc Tiên Nguyên Thạch) đối với người ngoài; doanh thu tích lũy vào Két Sắt trận pháp và có thể rút về túi đồ bất cứ lúc nào.
- **Thu Hồi & Phá Dỡ:** Chủ sở hữu có thể thu hồi trận pháp để hoàn lại 50% nguyên liệu chế tạo (tỉ lệ theo % độ bền hiện tại) và rút toàn bộ số dư Két Sắt.


## 9. Chiến đấu

- Combat PvE khởi tạo từ thám hiểm, mỏ hoặc các hệ thống gọi combat.
- Cổ sư dùng Cổ trang bị, có thể charge và chọn mục tiêu.
- Chỉ định mục tiêu hỗ trợ vị trí/hàng/cột/toàn bộ tùy `target_type` của Cổ.
- **Lượt chiến đấu:** hành động được xử lý theo thứ tự lượt và tốc độ. Trong auto-combat, các hiệu ứng hỗ trợ được resolve trước; mỗi nhân vật dùng hành động gây sát thương đầu tiên trong lượt, hoặc đánh thường nếu không có hành động phù hợp. Đòn đánh không tự kích hoạt một đòn phản công tức thời từ mục tiêu.
- **Thỉnh Đạo Hữu Tiên Hành Cổ (tiên thủ lượt 1):** trước khi trận bắt đầu, nếu thọ nguyên đối thủ chính không hơn gấp 3 lần thọ nguyên của đạo hữu, đạo hữu được **ra tay đầu tiên** ở lượt 1 (địch vẫn hành động ngay sau, theo tốc độ thường — không bị bỏ lượt). Quái vật dùng tuổi thọ quy đổi theo Chuyển; Cổ sư đối địch (PvP) dùng thọ nguyên thật.
- Combat state hỗ trợ đội hình địch/ta, status effects, cooldown, Chân Nguyên/Sinh lực, chiến lợi phẩm.
- Thắng trận áp dụng chiến lợi phẩm vào thẳng túi đồ thật; thua trận cập nhật trạng thái theo handler combat.
- **Túi đồ khi khám phá là túi thật:** nguyên liệu/Cổ/tiền nhặt được trong chuyến đi vào thẳng túi đồ (không còn "túi tạm"). Khi về trấn an toàn không có bước "thu hoạch" — đồ đã nằm sẵn trong túi.
- **Chết thì rơi đồ:** khi bại trận dã ngoại, bỏ chạy ngất xỉu hay thua PvP, đạo hữu có thể rơi một phần thạch, nguyên liệu, Cổ không trang bị/không phải Bản Mệnh và Bí phương. Tài sản tạo thành ground drop/hài cốt trên bản đồ thay vì biến mất hoặc chuyển thẳng vào túi người thắng; quyền nhặt tuân theo thời gian khóa mô tả ở mục bản đồ.
- **Công phá Tông Môn cướp kho:** thắng khiêu chiến Tông Môn sẽ cướp một phần Quỹ, linh tài, Cổ trùng và Bí phương trong bảo khố chia cho đội tấn công. Tuy nhiên **Thiên Đạo cũng tiêu hủy vĩnh viễn ~40% chiến lợi phẩm** (giống cơ chế rơi đồ khi tử đạo: mất đúng 1 đơn vị thì 50% cơ hội bị chấn nát, ngược lại luôn giữ lại ít nhất 1 đơn vị cho đội); tông môn thất thủ được Hộ Sơn Đại Trận bảo hộ.
- **Bỏ chạy (Rút Lui) khỏi chiến đấu** khiến đạo hữu chịu **−20% Max HP**; nếu HP không đủ sẽ ngất xỉu tại HUB và chịu hình phạt rơi đồ như trên. Khi HP ≤ 20% Max, hệ thống hiện cảnh báo xác nhận trước khi bỏ chạy.
- Khi hệ thống quá tải (tranh chấp khoá dữ liệu do nhiều thao tác cùng lúc), các thao tác chiến đấu (bỏ chạy/rút lui/bắt sủng…) trả về thông báo **bận và yêu cầu thử lại** thay vì treo vô hạn.

### Kháng hiệu ứng và Tenacity (đọc theo effect)

- Mọi kháng đều là effect cộng dồn từ Cổ đang trang bị + Chủng tộc + Thể chất + Trạng thái — không có Cổ nào "miễn nhiễm đặc quyền" ngoài effect nó mang.
- `mental_resistance` + `cc_resist`: giảm tỉ lệ dính hiệu ứng bất lợi (nhân với nhau, trần 75%).
- `tenacity` (0..1): rút ngắn thời gian hiệu ứng bất lợi phải chịu, đồng thời mỗi đầu lượt roll để tự giải 1 hiệu ứng (ưu tiên khống chế cứng như Choáng/Trói/Đóng băng/Ngủ trước, rồi đến Khóa chiêu, gây rối, cuối cùng là rút máu theo lượt). Buff có lợi không bao giờ bị tự giải nhầm.
- VD: Ngã Ý Cổ mang `tenacity: 0.35, cc_resist: 0.15` — chịu CC ngắn hơn và 35% tự giải 1 hiệu ứng mỗi lượt.

### Quy luật Áp Chế Cảnh Giới 2 Chiều & Tỉ Lệ Kích Hoạt Động (Dynamic Proc Chance)

- **Suy Hao Cảnh Giới Tự Thân (Caster Attenuation)**: Khi Cổ sư sử dụng Cổ trùng có đẳng cấp thấp hơn cảnh giới hiện tại của mình (`GuRank < CasterRank`) để buff cho bản thân hoặc đồng đội, hiệu lực sẽ bị suy giảm mạnh theo tỷ trọng sức mạnh cảnh giới $K_{\text{caster}} = W(\text{GuRank}) / W(\text{CasterRank})$ (Bảng trọng số: Phàm giới $1, 2, 5, 20, 100$; Tiên giới $1.000, 10.000, 100.000, 1.000.000$).
  - Các hiệu ứng đặc biệt như **Bất Hoại (INVINCIBLE)**, **Bất Tử (LAST_STAND)**, **Né Tránh (EVASION)** hay **Khóa Cổ (GU_SEAL)** sẽ bị nhân tỉ lệ kích hoạt với $K_{\text{caster}}$ (ví dụ: Tiên nhân Bát Chuyển/Cửu Chuyển dùng Cổ Bất Hoại Tứ Chuyển thì tỉ lệ kích hoạt rơi về $\approx 0.002\%$).
  - Trị số khiên ảo, hồi máu và các chỉ số cộng thêm cũng bị điều chỉnh tương ứng theo $K_{\text{caster}}$.
- **Áp Chế Cảnh Giới Mục Tiêu (Target Realm Suppression)**: Khi xuất chiêu khống chế hoặc nguyền rủa lên đối thủ có cảnh giới cao hơn (`TargetRank > GuRank`):
  - Mỗi bậc chênh lệch cảnh giới $\Delta R$ làm giảm $20\%$ tỉ lệ dính hiệu ứng: $\text{SuppressionFactor} = \max(0, 1.0 - 0.20 \times \Delta R)$.
  - Nếu chênh lệch $\Delta R \ge 5$ (ví dụ Cổ Tứ Chuyển đánh lên Tiên Tôn Cửu Chuyển), mục tiêu **miễn dịch tuyệt đối** với mọi hiệu ứng khống chế từ Cổ này.
  - Thời lượng khống chế (nếu trúng) bị cắt ngắn $\lfloor \Delta R / 2 \rfloor$ lượt (tối thiểu giữ 1 lượt).
  - Sát thương theo % Máu (`DamageHPPercent`), phá khiên (`ShieldStripPercent`) và tước quyền dùng Cổ (`DisableGuTurns`) cũng chịu chung quy luật suy giảm này.

### Stat_mods trạng thái trong chiến đấu & Chi tiết công thức

- `stat_mods` của Trạng thái (Trọng Thương, Áp Chế Giới Vực…) tác động **trực tiếp tới kết quả chiến đấu**: hệ số trạng thái **nhân thẳng lên toàn bộ sát thương sát chiêu & tuyệt kỹ** (đòn đánh thường bằng chính chỉ số tấn công bị giảm), phòng thủ trừ thẳng sát thương nhận vào (kèm giảm cả Khiên Ảo tạo ra), tốc độ quyết định thứ tự hành động.
- `healing_received` nhân lên **mọi nguồn hồi trong trận**: hồi của sát chiêu, hồi của Cổ hỗ trợ và **hồi bằng dược liệu tự động** (tối đa 3 lần/trận) — trước đây dược liệu bỏ qua hệ số này.
- Panel **"Chi Tiết Công Thức Tính Sát Thương"** trong nhật ký chiến đấu hiện hiển thị tường minh phần trạng thái: bên gây sát thương có dòng **Tấn Công** (gốc → sau trạng thái kèm tên trạng thái và `%`), bên nhận có dòng **Phòng Ngự** tương tự. Các entry hồi phục mở rộng thành panel **"Chi Tiết Công Thức Hồi Phục"** (hồi gốc + Đạo Ngân + hệ số hưởng + trạng thái ảnh hưởng).

### Bộ Cổ Chiến Đấu (Cài đặt chiến thuật tự động)

Trong tab **Cài đặt** của hồ sơ (Aperture), đạo hữu có thể chọn **Bộ Cổ Chiến Đấu** — danh sách Cổ mang theo khi xuất trận:

- **Dùng khi tấn công:** Khi mở màn hình chiến đấu, bộ Cổ đã lưu được **chọn sẵn** (ưu tiên hơn gợi ý tự động). Đạo hữu vẫn có thể bấm gợi ý hoặc chỉnh tay để đổi bộ cho trận đó.
- **Dùng khi bị tấn công:** Khi bị Cổ sư khác tấn công (mỏ, võ đài, bản đồ), người phòng thủ sẽ chiến đấu bằng đúng bộ Cổ đã lưu (theo thứ tự ưu tiên) — không dùng toàn bộ Cổ đang trang bị.
- **Để trống** (không chọn Cổ nào) = dùng tất cả Cổ đang trang bị & ACTIVE.
- Chỉ các Cổ **đang có trong túi và ở trạng thái ACTIVE** mới được giữ lại khi lưu; Cổ không hợp lệ sẽ bị hệ thống tự bỏ qua.
- Ngoài ra vẫn có thể đặt **điều kiện dùng** (Luôn / HP thấp / Chân nguyên cao / Máu địch thấp) và **thứ tự ưu tiên** cho từng Cổ trong bộ.


### Bắt sủng vật (Monster Capture)

Trong chiến đấu, đạo hữu có thể bắt quái vật làm sủng vật:

- **Điều kiện:** Quái vật phải có cờ `is_capturable` trong dictionary quái vật (hiện tại: Dế Mèn, Ong Vàng, Rắn Hổ, Bọ Cạp, Nhện Độc).
- **Yêu cầu Chuyển:** Cổ trùng bắt buộc phải **Chuyển ≥ Chuyển quái** mới có thể bắt.
- **Tỉ lệ bắt:** `clamp(template.capture_chance × hpFactor × typeFactor + captureBonus, 1%, 100%)`.
  ```
  hpFactor = sqrt(clamp(1 - HP hiện tại / HP tối đa, 0.1, 1))
  ```
  - `template.capture_chance`: xác suất gốc trong dữ liệu quái.
  - `typeFactor`: quái thường `1.0`, `ELITE` `0.6`, Boss và quái tai kiếp `0`.
  - `captureBonus`: cộng tổng bonus từ Cổ đang trang bị và ACTIVE.
  - HP đầy vẫn có hệ số sàn `sqrt(0.1)`; quái yếu dễ bắt hơn.
- **Kết quả:** Roll ngẫu nhiên dưới tỉ lệ bắt đã tính thì thành công; quái trở thành Actor sủng vật persistent và bị xóa khỏi combat. Ngược lại, tiếp tục chiến đấu.

### Sủng vật (Pets)

Sủng vật là quái vật đã bắt, hỗ trợ chiến đấu:

- **Đội hình:** Số slot tối đa là `min(1 + rank người chơi, 8)`; Nhất Chuyển có 2 slot, Ngũ Chuyển có 6, Thất Chuyển trở lên có 8. Pet và NPC dị nhân dùng chung slot đội hình.
- **Chiến đấu:** Sủng vật tự động tấn công quái địch mỗi lượt bằng BASIC_ATTACK (ATK × 1.0).
- **Phí Hồn Phách:** mỗi lần ra trận thật, một bầy sủng vật tiêu tốn `soul_hp` = số cá thể trong bầy; nếu Hồn Phách không đủ thì sủng vật đó bị bỏ qua, không xuất trận. Võ Đài (ảo) không tốn Hồn Phách.
- **Độ no:** Giảm theo `pet_hunger_per_hour` (mặc định 2/giờ). Sủng vật đang ngủ giảm 50%. Khi về 0 → **chết vĩnh viễn** (xóa khỏi DB).
- **Cho ăn:** Dùng vật phẩm có effect `hunger` (Thịt Lợn Rừng +40, Rượu Thanh Trúc +30, Ngọc Thạch +25, Nguyệt Lan +20).
- **Gộp:** Cùng `monster_template_id` có thể gộp thành 1 bầy. Chỉ số × `N^0.65` (theo base, không cộng dồn lũy tiến). Bầy tối đa `max_soul/10` cá thể — cần tăng Hồn Phách để gộp thêm.
- **Chết:** Sinh lực = 0 trong combat hoặc hết độ no → xóa vĩnh viễn. Không thể hồi sinh.

### Mô hình Actor thống nhất (nền tảng)

Sủng vật và NPC dị nhân có tên dùng chung mô hình Actor Instance. Mỗi cá thể giữ định danh và tiến trình lâu dài độc lập với việc có đang xuất hiện trên bản đồ hay không; loài/kiểu cung cấp dữ liệu nền, còn capability quyết định hành động được phép. Quái/sủng thú tiếp tục lấy template từ `monster_templates`; NPC không bị ép thành “monster” mà dùng nguồn template phù hợp với loại của mình.

**Cổ Sư dị nhân có tên:** có thể làm giám sát khai khoáng/nông nghiệp, hỗ trợ luyện Cổ, tu luyện tới Ngũ Chuyển, nhận Cổ từ chủ nhân và tham gia đội hình. NPC đang làm việc không thể đồng thời tu luyện hoặc chiến đấu; NPC tử trận được ghi nhận là đã chết. Các cơ chế hiện diện ngoài bản đồ và AI tự ra quyết định chưa được bật.

### Đấu Trường Sinh Tử (Death Arena)

Đại chiến tổ đội 6v6 loại trực tiếp đỉnh cao hàng tuần:

- **Lịch trình:** Diễn ra vào **chiều Thứ 7 hàng tuần**.
  - **Mở đăng ký:** Từ **16:00 đến 17:00 chiều Thứ 7**.
  - **Khai chiến:** Đúng **17:00 (5h chiều) Thứ 7**.
- **Quy mô & Giới hạn:**
  - Mỗi đội gồm **6 Cổ sư** (hệ thống tự động xếp đội cho người chơi đăng ký đơn).
  - Tối đa **16 đội** (tổng cộng **96 người chơi**). Khi đủ 16 đội hoặc 96 người, cổng đăng ký tự động khóa sớm.
- **Thể thức thi đấu & Nhịp độ vòng đấu:**
  - **Loại trực tiếp (Single Elimination):** Gồm 4 vòng đấu (Vòng 1 → Tứ kết → Bán kết → Chung kết).
  - **Nhịp độ:** Cứ mỗi **5 phút chia đôi số đội 1 lần** (16 → 8 → 4 → 2 → 1).
  - **Nghỉ giữa hiệp:** Sau khi các trận trong vòng kết thúc, hệ thống sinh nhánh đấu cho vòng kế tiếp và đếm ngược thời gian nghỉ để các đạo hữu mở xem **Lịch sử trận đấu (Combat Replay)** của đội mình và khán phá đối thủ.
- **Phần thưởng:**
  - Thưởng chiến thắng theo vòng: tăng lũy tiến từ 50 đến 400 Nguyên Thạch/người.
  - Quán Quân, Á Quân & Top 3: Nhận thêm Tiên Nguyên Thạch, danh hiệu độc quyền và phần thưởng vật phẩm quý hiếm.

### Đại Hội Võ Thuật (Martial Arena)

Đại hội 1v1 loại trực tiếp, tự động gom người online theo cảnh giới:

- **Lịch trình:** Khai mạc **12:00 trưa Thứ 3 hàng tuần**.
- **Cách tham gia:** Không cần đăng ký. Tại thời điểm khai mạc, hệ thống tự gom các đạo hữu **đang online** theo **cảnh giới tu vi** (`players.rank`).
- **Quy mô:**
  - Mỗi cảnh giới có **≥ 8 người online** thì mở 1 bảng đấu riêng (tối đa **128 người**/cảnh giới).
  - Cảnh giới nào dưới 8 người online sẽ **không tổ chức** kỳ đó (không gộp cảnh giới).
- **Thể thức:**
  - **1v1 loại trực tiếp (Single Elimination):** seed theo `sub_rank` — người gần cảnh giới nhất gặp nhau từ vòng đầu.
  - Trận đấu diễn ra **tự động** (mô phỏng chiến đấu), không tiêu hao tài nguyên, không cần online lúc thi đấu.
- **Phần thưởng (Top 3):** Nguyên Thạch, Tiên Nguyên Thạch, Cổ Trùng, vật phẩm và danh hiệu độc quyền (nhận qua mục Phần Thưởng).

## 10. Phó bản

- Instance có template, session active và action theo option index.
- Cổ sư vào instance, chọn hành động và nhận kết quả theo stage/map data.
- **Mộng Cảnh (DREAM_REALM):** có 1 mộng cảnh cho mỗi lưu phái (49 path). Hoàn thành mộng cảnh của lưu phái X giúp **tăng cảnh giới lưu phái X** (thưởng +50 Chân Ý cho đúng path đó). Vào bằng `dream_path_material`, cooldown 24 giờ/lần, tiêu hao Soul HP.
- **Bí Cảnh (SECRET_REALM):** phó bản chiến đấu thưởng Đạo Ngân theo path, cooldown 48 giờ.
- **Truyền Thừa (INHERITANCE_*):** phó bản cốt truyện riêng (vd. Truyền Thừa Hoa Tửu Hành Giả).

## 11. Hồn phách

- Player có `soul_hp`, `max_soul`, `soul_tier`, `needs_soul_refine`.
- Giao diện hiện kết nối hành động **luyện hồn**; các thao tác hồn phách khác phụ thuộc vật phẩm/luật gameplay tương ứng, không có endpoint riêng cho “use-guts” hay “calm”.

## 12. Kinh tế

### Cửa hàng NPC

- Danh sách cửa hàng public; mua/bán cần đăng nhập.
- Dữ liệu cửa hàng được import từ `backend/data/ShopItem.csv`. Có lọc theo loại/rank/lưu phái, mua và bán Cổ–nguyên liệu–Bí phương, kể cả bán hàng loạt.

### Bảo Hoàng Thiên chợ trời

- Hỗ trợ niêm yết kiểu chợ, đấu giá và giao dịch/mục tiêu người mua.
- Khi bán Cổ vào đơn thu mua, người bán phải chọn đủ các cá thể Cổ đúng loại và đúng hạng; Cổ bản mệnh, đang trang bị, bị khóa hoặc đã chết không thể giao dịch. Đơn Bí phương chỉ giao dịch từng bản một.
- Có thể bán nhiều Cổ cùng loại và cùng hạng trong một lần. Số lượng và danh sách Cổ được chọn đồng bộ hai chiều; nút **Tối Đa** tự chọn toàn bộ cá thể hợp lệ trong giới hạn của đơn.
- Có đặt giá, mua, hủy và kết toán đấu giá hết hạn.
- Có chat Bảo Hoàng và tin nhắn real-time qua WebSocket.

### Thông báo

- Chuông thông báo (góc trên phải) báo tin mới theo thời gian thực qua WebSocket.
- Thông báo khi: Cổ chết đói (kể cả bản mệnh cổ), tử đạo đánh rơi tài sản (thông báo chi tiết lượng tài nguyên rơi tại chỗ và lượng bị Thiên Đạo tiêu hủy vĩnh viễn), nhặt chiến lợi phẩm, ai mua hàng của bạn, ai đặt giá đấu giá, thắng đấu giá, đấu giá hết hạn.
- Hỗ trợ đánh dấu đã đọc từng mục hoặc tất cả.

#### Cơ chế Phạt Tử Đạo & Chống Lợi Dụng Chết Chuyển Đồ (Thiên Đạo Tiêu Hủy)
- Để ngăn chặn hành vi cố tình chết để tuồn chuyển tài nguyên, đồ đạc:
  - Tỷ lệ hao hụt tài nguyên cơ bản khi tử đạo tăng lên **40%** (Nguyên Thạch, Tiên Nguyên Thạch, Nguyên liệu) và **30%** (Cổ trùng, Bí phương). Khi có Sát Khí, tỷ lệ này nhân thêm hệ số Oán Khí.
  - **Quy tắc Thiên Đạo Tiêu Hủy (Void Destruction):** Trong tổng số tài nguyên hao hụt khi chết, **40% sẽ bị Thiên Đạo chấn nát và tiêu hủy vĩnh viễn khỏi server**, chỉ **60% còn lại** mới rơi ra bãi chiến lợi phẩm tại chỗ (`map_ground_drops`).
  - Bãi rơi chỉ tồn tại trong vòng **10 phút** (3 phút đầu bảo hộ cho người hạ gục, 7 phút sau cho toàn server).
  - Cổ Bản Mệnh (`IsVital`) tuyệt đối không bao giờ bị rơi hay bị tiêu hủy khi chết. Cổ đang trang bị chỉ có nguy cơ rơi/tiêu hủy khi Sát Khí > 5.

### Giao dịch trực tiếp

- Hai Cổ sư có thể khởi tạo, chấp nhận/từ chối, đề nghị, khóa và xác nhận.
- Đề nghị hỗ trợ nguyên thạch, Tiên Nguyên Thạch, Cổ, vật liệu và **Bí phương**.
- Có WebSocket trade events — nhận cập nhật giao dịch real-time.

### Ví, nạp tiền và gói quý

- Ví có nguyên thạch, tiên nguyên thạch, tiên ngọc.
- Đơn nạp dùng package và PayOS webhook nếu cấu hình.
- Cửa hàng gói quý mua bằng ví.
- **Đổi đá** (tab "Đổi Đá" trong Cát Bảo Các): đổi Tiên Nguyên Thạch ↔ Nguyên Thạch theo tỷ giá cố định — 1 Tiên Nguyên Thạch = 100 Nguyên Thạch, 100.000 Nguyên Thạch = 1 Tiên Nguyên Thạch.

### Khu Giải Trí

Khu vực thư giãn cho các đạo hữu đọ vận may và tranh đoạt Nguyên Thạch. Gồm hai phần:
- **Cổ Từ Bảo Khố (Đổ Thạch):** Mở đá huyền bí đọ cơ duyên nhận Nguyên Thạch, Cổ trùng hoặc Thọ Nguyên.
- **Đấu Thú Đặt Cược:**
  - Hai phe **Tả Thanh Long** (Phe Đỏ) và **Hữu Bạch Hổ** (Phe Xanh) xuất quân với 9 linh thú giống nhau, cắn nhau ngẫu nhiên.
  - Đạo hữu có thể đặt cược Nguyên Thạch vào phe tin rằng sẽ chiến thắng.
  - Phiên cược diễn ra mỗi giờ một lần. **Khóa đặt cược vào phút thứ 55 của mỗi giờ**.
  - Kết quả trận đấu được phân định vào đầu giờ tiếp theo bằng cách mô phỏng trận chiến ngẫu nhiên của các linh thú.
  - Tổng số tiền thắng cược (phần thưởng thêm chia từ bên thua) sẽ được khấu trừ 10% làm phí duy trì hệ thống (bảo đảm hệ thống không lỗ). Số Nguyên Thạch còn lại chia đều tỷ lệ thuận theo lượng cược cho bên thắng (cân kèo chia thưởng).

> **Đơn vị tiền tệ:** Nguyên Thạch, Tiên Nguyên Thạch và giá vật phẩm hỗ trợ tối đa 3 chữ số thập phân; `0.001` là đơn vị nhỏ nhất.

## 13. Môn phái (Tông Môn)

Thay thế cho chế độ gia tộc dựa trên huyết thống cổ xưa, Tông Môn là tổ chức thế lực tuyển chọn đệ tử dựa trên năng lực, phẩm hạnh và lòng trung thành, vận hành bằng hệ thống phân cấp, sư đồ, điểm cống hiến, ngân quỹ và căn cứ lãnh địa trên bản đồ thế giới.

### 1. Khai Tông Lập Phái (Thành lập Tông Môn)

- **Điều kiện thành lập**:
  - Tu vi Cổ sư đạt tối thiểu **Tam Chuyển (Rank 3)**.
  - Chi phí: **30.000 Nguyên Thạch** làm ngân quỹ khai sơn.
  - Phải là Cổ sư tự do (chưa gia nhập tông môn nào, nếu có phải rời/phản môn trước).
  - Tên tông môn từ 3 đến 50 ký tự (hệ thống tự chuẩn hóa thành ID duy nhất, cấm trùng lặp).
- **Vị trí Căn cứ Tông Môn**:
  - **Bắt buộc dã ngoại (`KHU_VUC`)**: Không thể lập phái trong An Toàn Khu / Điểm tập kết / Trấn (`HUB`, `SAFE_ZONE`, `hub_nam_cuong`, `Lobby`). Cổ sư phải di chuyển ra ngoài dã ngoại.
  - Tọa độ thực tế của người sáng lập (`current_region`, `x`, `y`) sẽ trở thành **Tọa độ Căn cứ Tông Môn** (`region`, `map_x`, `map_y`) cố định trên bản đồ thế giới.
- **Đặc quyền Người sáng lập**:
  - Tự động đảm nhiệm vị trí **Chưởng Môn (`CHUONG_MON`)**.
  - Nhận ngay **10.000 Điểm Cống Hiến** ban đầu.

---

### 2. Cấu Trúc Thành Viên & Chức Vụ (Hệ Thống Phân Cấp)

Hệ thống môn phái chia thành 7 cấp bậc rõ rệt:

| Cấp bậc | Mã hiệu | Yêu cầu Tu vi | Phí Thăng cấp (Điểm Cống Hiến) | Đặc quyền & Quyền hạn |
| :--- | :--- | :--- | :--- | :--- |
| **Ngoại Môn** | `NGOAI_MON` | Nhất Chuyển (Rank 1) | Mặc định khi gia nhập | Thành viên sơ cấp, làm nhiệm vụ, quyên góp, đổi kho |
| **Nội Môn** | `NOI_MON` | Nhị Chuyển (Rank 2) | **100** Điểm | Nhận nhiệm vụ cấp cao hơn |
| **Tinh Anh** | `TINH_ANH` | Tam Chuyển (Rank 3) | **500** Điểm | Nhận nhiệm vụ Tinh Anh, tăng phụ cấp bổng lộc |
| **Chân Truyền** | `CHAN_TRUYEN` | Tứ Chuyển (Rank 4) | **2.000** Điểm | Mặc định cần lãnh đạo phê duyệt thăng cấp |
| **Trưởng Lão** | `TRUONG_LAO` | Ngũ Chuyển (Rank 5) | **10.000** Điểm | **Thuộc Lãnh Đạo**: Có quyền quản trị tông môn |
| **Hộ Pháp** | `HO_PHAP` | Lục Chuyển (Rank 6 - Cổ Tiên) | **30.000** Điểm | **Thuộc Lãnh Đạo**: Trụ cột chiến lực tông môn |
| **Chưởng Môn** | `CHUONG_MON` | — | Không thể tự thăng cấp | Lãnh tụ tối cao, định đoạt mọi việc lớn nhỏ |

- **Quyền Quản Lý Tông Môn (`canManageSect`)**: Dành riêng cho **Trưởng Lão**, **Hộ Pháp**, **Chưởng Môn**:
  - Xem và phê duyệt / từ chối đơn xin gia nhập tông môn.
  - Khảo hạch và duyệt yêu cầu thăng chức của môn nhân.
  - Ban hành nhiệm vụ tùy biến (`CreateCustomMission`) và xóa nhiệm vụ trên bảng.
  - Bố trí đệ tử trấn thủ hoặc chi ngân quỹ thuê thần thú hộ sơn.
  - Nhập hàng (stocking) Linh tài và Bí phương vào Bảo khố bằng Quỹ tông môn.
  - Nâng cấp Ngũ Đại Kiến Trúc tông môn.
  - Phát bổng lộc hàng ngày cho toàn thể thành viên.
  - Thay đổi cấp bậc đệ tử và Trục xuất thành viên (chỉ trục xuất người có cấp bậc thấp hơn mình; Chưởng Môn không thể bị trục xuất).
- **Chuyển Giao Quyền Lực (Nhường ngôi Chưởng Môn)**:
  - Chưởng Môn có thể phong một thành viên khác làm Chưởng Môn mới.
  - Khi nhường ngôi, Chưởng Môn tiền nhiệm sẽ tự động thoái vị chuyển thành **Trưởng Lão**.
- **Cơ chế Khảo Hạch & Duyệt Lên Cấp**:
  - Khi bật thiết lập **"Duyệt Lên Cấp"** (`require_promotion_approval`): **Tất cả** thăng cấp từ Ngoại Môn trở lên đều phải tạo đơn chờ lãnh đạo phê duyệt. Điểm cống hiến bị trừ ngay khi nộp đơn.
  - Khi tắt thiết lập: Đệ tử tự động thăng lên Nội Môn và Tinh Anh ngay khi đủ điều kiện. Riêng các bậc **Chân Truyền, Trưởng Lão, Hộ Pháp** luôn bắt buộc phải qua phê duyệt nếu tông môn có Chưởng Môn.
- **Gia Nhập & Rời Tông Môn**:
  - **Gia nhập**: Nếu tông môn bật "Duyệt gia nhập" (`require_approval`, yêu cầu tông môn từ Cấp 10 trở lên mới được bật), người chơi gửi đơn chờ duyệt (Nhân Khí +2). Nếu tắt, người chơi gia nhập ngay lập tức (Nhân Khí +3).
  - **Rời tông môn (`LeaveSect`)**: Xóa toàn bộ tư cách thành viên, hủy toàn bộ nhiệm vụ đang làm và **mất trắng toàn bộ Điểm cống hiến tích lũy**.

---

### 3. Ngân Quỹ, Bổng Lộc & Chi Phí Bảo Trì

- **Quỹ Tông Môn (`funds`)**: Đơn vị ngân sách tính bằng Nguyên Thạch của tông môn, dùng cho: nâng cấp kiến trúc, thuê thần thú hộ sơn, nhập hàng bảo khố, phát lương đệ tử và trả phí duy trì hàng ngày.
- **Nguồn Thu Quỹ Tông Môn**:
  - Đệ tử quyên góp Nguyên Thạch (tỉ lệ 1:1).
  - Sản lượng tự động từ **Linh Mạch** mỗi phút (gia tăng theo số lượng đệ tử và thần thú canh giữ).
  - Cướp phá từ ngân quỹ của tông môn khác khi công sơn thắng lợi.
- **Phát Bổng Lộc Hàng Ngày (`PaySectSalary`)**:
  - Trưởng Lão, Hộ Pháp hoặc Chưởng Môn có thể kích hoạt phát bổng lộc **1 lần mỗi 24 giờ**.
  - Toàn bộ bổng lộc được khấu trừ từ Quỹ tông môn và chuyển trực tiếp thành **Nguyên Thạch** vào túi của từng thành viên:
    $$\text{Bổng Lộc} = (\text{Cấp Tông Môn} \times 100) + \text{Phụ Cấp Chức Vụ}$$
  - **Phụ cấp chức vụ**: Ngoại Môn: `+0` | Nội Môn: `+50` | Tinh Anh: `+150` | Chân Truyền: `+400` | Trưởng Lão: `+1.000` | Hộ Pháp: `+2.000` | Chưởng Môn: `+5.000` Nguyên Thạch.
- **Phí Duy Trì Tông Môn Hàng Ngày (`SectMaintenanceWorker`)**:
  - Hệ thống chạy ngầm tự động mỗi 24 giờ.
  - Chi phí bảo trì:
    $$\text{Phí Duy Trì} = \text{Cấp Tông Môn} \times 5.000 \text{ Quỹ}$$
  - **Hệ quả thiếu quỹ**: Nếu Quỹ tông môn không đủ thanh toán phí bảo trì, tông môn **bị giáng 1 cấp** (`Level - 1`, tối thiểu Cấp 1) và Thiên Đạo lập tức gửi thư thông báo cảnh báo đến Chưởng Môn.

---

### 4. Quyên Góp & Bảo Khố Tông Môn (Vault & Donations)

Bảo khố tông môn gồm 3 khoang riêng biệt: **Cổ Trùng (`sect_vault_gu`)**, **Linh Tài (`sect_vault_materials`)** và **Bí Phương (`sect_vault_recipes`)**.

#### Quy Tắc Quyên Góp (Nhận Điểm Cống Hiến):
- **Nguyên Thạch**: 1 Nguyên Thạch $\to$ 1 Điểm cống hiến + 1 Quỹ tông môn.
- **Linh Tài**: Nhận Điểm cống hiến $= \text{BasePrice} \times 50\%$ (tối thiểu 1 điểm). Linh tài được lưu trữ vào kho nguyên liệu.
- **Cổ Trùng**: Nhận Điểm cống hiến $= \text{EffectiveBasePrice} \times 50\%$.
  - Rank 1: **100** | Rank 2: **250** | Rank 3: **600** | Rank 4: **1.250** | Rank 5: **2.500** Điểm cống hiến.
  - Tiên Cổ: Rank 6: **100** | Rank 7: **250** | Rank 8: **600** | Rank 9: **1.250** Điểm cống hiến.
- **Bí Phương**: Nhận Điểm cống hiến $= \text{RecipeCost} / 1.5$ (Rank 1: 13, Rank 2: 33, Rank 3: 80, Rank 4: 166, Rank 5: 333 Điểm).
- Hỗ trợ tính năng **Quyên Góp Hàng Loạt (Bulk Donate)** nhanh chóng.

#### Quy Tắc Đổi & Mua Từ Bảo Khố:
- **Chi phí bằng Điểm Cống Hiến**: Giá đổi vật phẩm từ bảo khố luôn nhân hệ số Markup $1.5\times$ so với điểm khi quyên góp ($\text{Giá Đổi} = \text{BasePrice} \times 75\%$), đảm bảo triệt tiêu hoàn toàn khả năng cày điểm xoay vòng.
- **Bí Phương Luyện Cổ**: Học trực tiếp từ bảo khố, chi phí tính theo Rank của Cổ mục tiêu ($\text{GuBasePrice} / 10$, tối thiểu 20 Điểm).
- **Tiên Cổ (Rank 6+) Độc Nhất Toàn Thiên Hạ**:
  - Mỗi Tiên Cổ trong bảo khố là độc nhất toàn server, giữ nguyên mã số sinh mệnh `instance_id`. Mỗi lần chỉ được rút 1 instance.
  - Thao tác rút Tiên Cổ được ghi chép vĩnh viễn vào **Lịch Sử Đời Cổ (`gu_history`)** kèm tên Cổ sư, tên tông môn và giá trị điểm tiêu hao.
- **Hao Đói & Rủi Ro Tử Vong Của Cổ Trong Kho**:
  - Cổ trùng nằm trong Bảo khố **vẫn liên tục tiêu hao Độ No mỗi 3 giờ** như trong túi Cổ sư! Nếu để Độ No tụt về 0, Cổ sẽ **chết và vĩnh viễn biến mất khỏi kho**.
  - Khi đệ tử đổi Cổ từ kho ra, nếu là loại Cổ có tiêu hao thức ăn (`hunger_rate > 0`), Cổ nhận về có **Độ No khởi đầu = 1** (cần cho ăn gấp, không được reset đầy).

#### Nhập Hàng Vào Kho (Stocking):
- Lãnh đạo có thể dùng Quỹ tông môn để nhập sẵn Linh tài & Bí phương vào kho cho đệ tử đổi:
  - **Linh tài**: Hạng tối đa $= \lfloor\text{Cấp TM} / 10\rfloor + 1$ (tối đa Rank 5). Chi phí nhập: $\text{Hạng} \times 50$ Quỹ/đơn vị.
  - **Bí phương**: Hạng tối đa $= \lfloor\text{Cấp TM} / 10\rfloor + 1$ (tối đa Rank 5). Chi phí nhập: $\text{Hạng Cổ} \times 500$ Quỹ/bản.
  - **Cấm nhập Tiên phẩm**: Tuyệt đối không thể dùng quỹ nhập Linh tài hoặc Bí phương Tiên cấp (Rank 6+).
  - **Cổ trùng**: **Không thể nhập bằng quỹ tông môn** (chỉ có thể thu nạp qua sự tự nguyện quyên góp của môn nhân).

---

### 5. Nhiệm Vụ Tông Môn (Sect Missions)

- **Bảng Nhiệm Vụ**:
  - Gồm 4 phân loại: Thu thập vật liệu (`GATHER`), Quyên góp Cổ (`DONATE`), Tiêu diệt yêu thú (`COMBAT`), Luyện chế Cổ trùng (`REFINE`).
  - Mỗi nhiệm vụ quy định cấp bậc môn nhân tối thiểu được phép tiếp nhận.
  - Mỗi đệ tử tại một thời điểm chỉ được thực hiện tối đa 1 nhiệm vụ tông môn. Đệ tử có thể chủ động **Hủy bỏ nhiệm vụ** (`AbandonSectMission`) đang làm để nhận nhiệm vụ khác.
- **Nhiệm Vụ Tùy Biến (Custom Mission)**:
  - Lãnh đạo tông môn có thể tự phát hành nhiệm vụ mới: đặt tên, mô tả, chỉ định vật phẩm/mục tiêu, số lượng cần nộp, số Điểm cống hiến ban thưởng và thời hạn.
- **Cơ Chế Hết Hạn Tự Động (`SectMissionWorker`)**:
  - Nhiệm vụ trên bảng có thời hạn (mặc định 24 giờ; Trưởng Lão/Chưởng Môn có thể tùy chỉnh khi phát hành; đặt `-1` là vô thời hạn).
  - Worker định kỳ quét: Nhiệm vụ hết hạn sẽ tự biến mất khỏi bảng, tiến độ của đệ tử đang thực hiện dở dang cũng chuyển thành trạng thái thất bại (`EXPIRED`).

---

### 6. Ngũ Đại Kiến Trúc Tông Môn (Buildings)

Mọi kiến trúc khởi đầu ở cấp 1. Nâng cấp tiêu hao Quỹ tông môn:

| Kiến trúc | Mã hiệu | Chi phí nâng cấp | Tác dụng vận hành |
| :--- | :--- | :--- | :--- |
| **Chính Điện** | `main_hall` | $5.000 \times \text{Cấp Mục Tiêu}^2$ Quỹ | Định đoạt **Cấp bậc Tông Môn** (`sects.level`). Là trần giới hạn: cấp của 4 kiến trúc còn lại không được vượt quá Chính Điện. Mở rộng bán kính lãnh địa ($\text{Radius} = 5 + \text{Cấp}$). |
| **Hộ Sơn Trận Pháp** | `defensive_array` | $2.000 \times \text{Cấp Mục Tiêu}^2$ Quỹ | Tăng cường toàn diện chỉ số HP, Công, Thủ, Mana của **Hộ Sơn Thần Thú**. Khi kỹ năng sẵn sàng, chặn lần công kích đầu và mở 10 phút tập hợp; kỹ năng hồi cố định 12 giờ. |
| **Linh Mạch** | `spirit_vein` | $3.000 \times \text{Cấp Mục Tiêu}^2$ Quỹ | Tăng sản lượng tự sinh Quỹ tông môn: $(10 + \text{Cấp} \times 10)$ Quỹ/phút. Tự động thu thập và bổ sung Linh tài phàm phẩm vào kho mỗi phút. |
| **Tàng Bảo Các** | `treasury` | $1.500 \times \text{Cấp Mục Tiêu}^2$ Quỹ | Giảm tỉ lệ từng tài nguyên/Cổ/Bí phương rơi khi thất thủ: từ 20% xuống tối thiểu 5%. Đồng thời giảm 50% phí kho trong duy trì/di dời ở cấp 1, tăng dần đến tối đa 90%. |
| **Diễn Võ Trường** | `training_ground` | $1.500 \times \text{Cấp Mục Tiêu}^2$ Quỹ | Tăng hiệu suất thu hoạch Điểm cống hiến thụ động theo giờ của toàn bộ môn nhân ($+10\%$ mỗi cấp kiến trúc). |

---

### 7. Lãnh Địa & Hiệu Ứng Trận Pháp Sơn Môn (Territory Mechanics)

- **Tọa độ & Bán Kính Lãnh Địa**:
  - Căn cứ tông môn tọa lạc cố định tại `(map_x, map_y)` dã ngoại.
  - Phạm vi lãnh địa hình vuông: $|x - \text{map\_x}| \le \text{Radius}$ và $|y - \text{map\_y}| \le \text{Radius}$, với:
    $$\text{Radius} = 5 + \text{Cấp Chính Điện}$$
- **Hiệu Ứng Hộ Sơn Trận Pháp (Territory AoE Buff)**:
  - Thành viên đứng trong phạm vi lãnh địa của tông môn mình nhận **$+10\%$ Tấn Công và $+10\%$ Phòng Thủ** cơ bản ($+2\%$ công/thủ cho mỗi cấp Chính Điện).
- **Trận Pháp Che Chở (Chống Đồ Sát Ngoài Dã Ngoại)**:
  - Khi Hộ Sơn Trận Pháp còn hiệu lực (chưa bị công phá), đệ tử đứng trong phạm vi lãnh địa tông môn được đại trận che chở: **người chơi khác ngoài dã ngoại không thể phát động tấn công PvP**.
- **Cấm Tuyệt Đối Đồng Môn Tương Tàn**:
  - Hai Cổ sư cùng thuộc một tông môn không thể tấn công lẫn nhau ngoài dã ngoại ("Đồng môn tương tàn là điều đại kỵ trong Tông Môn!").
- **Thu Hoạch Thụ Động Theo Phút (`SectTerritoryWorker`)**:
  - Hệ thống tự động vận hành định kỳ mỗi phút:
    - **Quỹ tông môn**: Tự động tích lũy vào ngân quỹ dựa theo cấp Linh Mạch và nhân thêm hệ số thủ hộ ($+10\%$ mỗi thủ hộ).
    - **Linh tài**: Tự sinh các loại Linh tài có phẩm cấp phù hợp với cấp tông môn vào kho nguyên liệu.
    - **Điểm cống hiến thụ động cho thành viên**:
      - Đệ tử bình thường: $+2$ điểm/giờ ($2.0 / 60$ mỗi phút).
      - Đệ tử Trấn Thủ: $+20$ điểm/giờ ($20.0 / 60$ mỗi phút).
      - Tỷ lệ trên được nhân thêm hệ số từ **Diễn Võ Trường** ($1.0 + \text{Cấp Diễn Võ Trường} \times 0.1$).
      - **Giới hạn trần ngày**: Mỗi đệ tử tối đa nhận **200 Điểm cống hiến thụ động/ngày** (chống treo máy vô tận), tự động làm mới vào ngày hôm sau.

---

### 8. Thủ Hộ, Trấn Thủ & Công Phá Sơn Môn (Warfare & Siege)

- **Bố Trí Thủ Hộ Tông Môn (`DefendSect`)**:
  - Trưởng Lão, Hộ Pháp, Chưởng Môn có thể phân công đệ tử làm Thủ hộ hoặc thuê Thần Thú canh giữ (chi phí thuê: $\text{Rank} \times 500$ Quỹ).
  - Số lượng thủ hộ tối đa $= \text{Cấp Tông Môn} + 1$.
  - Mỗi thủ hộ đang canh giữ tăng thêm $+10\%$ tổng sản lượng thu hoạch Quỹ và Linh tài của tông môn.
- **Trấn Thủ Sơn Môn Cá Nhân (`ToggleDefendSect`)**:
  - Đệ tử đứng trực tiếp tại tọa độ căn cứ tông môn có thể bật/tắt trạng thái **Trấn Thủ** (`is_defending`).
  - Đệ tử trấn thủ nhận tốc độ tích lũy Điểm cống hiến thụ động gấp 10 lần bình thường (20 điểm/giờ).
  - Khi sơn môn bị tấn công, tối đa **8 đệ tử Trấn Thủ** đang có mặt tại chỗ sẽ trực tiếp xuất trận cùng Hộ Sơn Thần Thú để nghênh chiến.
- **Khiêu Chiến & Công Phá Tông Môn (`AttackSect`)**:
  - **Khoảng cách tiếp cận**: Người tấn công phải di chuyển đến vị trí cách căn cứ mục tiêu **trong vòng 2 ô** trên bản đồ thế giới. Không thể tấn công tông môn của chính mình.
  - **Thời gian bảo hộ**: Nếu tông môn mục tiêu đang có Hộ Sơn Đại Trận bảo vệ (`shield_until > now`), kẻ địch bất khả xâm phạm.
  - **Đội hình Phòng thủ Sơn Môn (Tối đa 9 thực thể)**:
    - 1 **Hộ Sơn Thần Thú (Boss)**: Rank bằng Cấp Tông Môn, SubRank bằng Cấp Hộ Sơn Trận Pháp. Chỉ số HP, Tấn công, Phòng thủ, Mana cực lớn tỷ lệ thuận với cấp trận và cấp tông môn.
    - Tối đa **8 Đệ tử Trấn Thủ** còn sống và đang đứng tại sơn môn.
- **Kết Quả: Công Phá Thành Công (Kẻ Tấn Công Thắng)**:
  - **Cướp phá Kho Tàng**: Đoạt lấy Quỹ, Linh tài, Cổ trùng và Bí phương từ bảo khố đối phương theo tỉ lệ:
    $$\text{Tỉ lệ cướp} = 20\% - (\text{Cấp Tàng Bảo Các} \times 2\%) \quad (\text{Tối thiểu } 5\%)$$
  - **Thiên Đạo Tiêu Hủy (Void Destruction - 40%)**: Thuận theo ý trời trong tranh đoạt khốc liệt, **40% toàn bộ chiến lợi phẩm cướp được sẽ bị chấn nát, tiêu biến vĩnh viễn vào hư vô**! 60% còn lại được chia cho đội tấn công / chuyển vào Quỹ tông môn phe thắng.
  - **Kích Hoạt Hộ Sơn Đại Trận**: khi kỹ năng sẵn sàng, lần công kích đầu tiên bị hủy và toàn bộ môn nhân nhận cảnh báo tập hợp. Sơn môn bất khả xâm phạm trong **10 phút**; kỹ năng chặn này hồi **12 giờ**, không cộng thêm thời gian theo cấp công trình.
  - **Chiến công**: Người tấn công nhận $+10$ Nhân Khí và $+200$ Điểm cống hiến (nếu có tông môn).
- **Kết Quả: Công Phá Thất Bại (Sơn Môn Phòng Thủ Thắng)**:
  - Sơn môn phòng thủ thành công nhận ngay **4 giờ** Hộ Sơn Đại Trận bảo hộ.
  - Toàn thể đệ tử Trấn Thủ tham chiến nhận thưởng **$+100$ Điểm Cống Hiến**.
  - Phát thông cáo chiến thắng oanh liệt trên kênh chat Tông Môn.

---

## 14. Mỏ cá nhân

- Có danh sách mine slots với rarity và phần thưởng tích lũy.
- Cổ sư chiếm mỏ, nhận thưởng, bỏ mỏ hoặc tấn công mỏ người khác.
- Worker có thể gán quái vào mỏ bỏ hoang.

## 15. Tổ đội và Đấu trường

- Tổ đội: tạo tổ đội, xem trạng thái, mời Cổ sư, rời tổ đội.
- **Chiến đấu tổ đội:** Các thành viên trong tổ đội ở cùng Đại Vực sẽ cùng tham gia vào trận chiến khi Trưởng nhóm khởi xướng (cả Turn-based lẫn Auto Combat/Simulation). Đồng đội sẽ tự động chiến đấu bằng các Cổ trùng đang trang bị, Sát chiêu và Thiết lập tự động của chính họ. Phần thưởng chiến thắng (Nguyên thạch, Nguyên liệu, Cổ trùng) sẽ được chia đều cho tất cả thành viên tham chiến, và tiêu hao (Sinh lực, Chân Nguyên, Dược phẩm) cũng được cập nhật tương ứng. Đồng đội không nhận điểm Chân Ý từ trận chiến — chỉ người chơi chính (Trưởng nhóm) được ngộ đạo.
- **Võ Đài:** tối đa 10 lượt khiêu chiến/ngày; thắng người xếp cao hơn thì đổi vị trí. Thắng nhận 50, thua nhận 10 Đồng Võ Đài; hệ thống chốt thưởng ngày lúc 22:30. Đồng dùng tại tab Bảo Các của Võ Đài.

## 16. Chat, mail và feedback

- Mail riêng: gửi và xem inbox.
- Bảo Hoàng chat theo channel, fetch message và subscribe WebSocket. Hệ thống tích hợp bộ lọc chặn spam (giới hạn thời gian gửi 1.5s, cấm trùng lặp tin nhắn trong 10s, cấm lặp từ hoặc chuỗi ký tự liên tiếp) và chặn từ ngữ tục tĩu nhạy cảm.
- Feedback (Thỉnh Thiên Đạo): mỗi sớ trình là một topic. Xem danh sách, gửi góp ý, bình chọn, góp ý (comment), cập nhật trạng thái/developer response.
- **Quyền theo trạng thái topic:**
  - `pending` (chờ duyệt): không bình chọn, không góp ý.
  - `active` (đang mở): bình chọn + góp ý.
  - `in_progress` / `completed`: không bình chọn, chỉ góp ý.
  - `rejected`: khép lại, chỉ xem.
- Cả bình chọn và góp ý đều yêu cầu Cổ sư đạt **Nhị Chuyển (Chuyển ≥ 2)** trở lên. Tác giả hoặc Admin có thể xóa góp ý.
- Thưởng: khi topic được duyệt triển khai (`in_progress`/`completed`), tác giả nhận 5 Tiên Nguyên (Chuyển ≥ 6) hoặc 500 Nguyên Thạch (phàm nhân).

## 17. Hiệu ứng đặc sắc (Special Effects — đọc theo effect, không theo tên Cổ)

Nguyên tắc: engine chỉ đọc effect (`other.action_code`, `active.*`, `passive.*`), không đọc Gu ID. Cổ nào mang effect đó thì có hành vi đó — Cổ nêu bên dưới chỉ là ví dụ minh họa.

### Cơ chế trong chiến đấu (Combat Simulator)
1. `ON_DEATH / REBORN` — hồi sinh thống nhất 2 mode: `rewind` (reset cả trận về Lượt 1, VD Xuân Thu Thiền) và `in_place` (đứng dậy tại chỗ với `revive_hp_percent` HP, VD Lãng Quên Bại Cổ: lượt 1 auto sống, sau roll 50%, tối đa 5 lần). Params: `chance`, `mode`, `max_uses` (1–10), `turn1_guaranteed`, `revive_hp_percent`, `clear_debuffs`, `empower_attack/speed/shield_percent` (+10% công/tốc/khiên mỗi lần sống dậy, cộng dồn hết trận), `empower_turns`. `REVIVE_LOCK` chặn hoàn toàn. **Phe địch cũng hồi sinh được nhưng chỉ `in_place`.**
2. `ON_CAST / SUMMON_WOLF_FRONTLINE` — triệu hồi Địa Lang Vương (HP = 50% MaxHP chủ, công = 1/3, thủ = 1/2) lên tiền tuyến đỡ đòn, chủ lùi về tuyến giữa. VD: Ngự Lang Vương Cổ.
3. `ON_CAST / FULL_SOUL_HEAL` — hồi đầy HP lập tức, đổi lại mất lượt hành động hiện tại. VD: Can Đảm Cổ.
4. `ON_CAST / SWORD_DRAGON_TRANSFORM` — chuyển toàn bộ HP (giữ lại 1) thành Khiên Ảo, +50% sát thương trong 3 lượt. VD: Kiếm Long Biến Cổ.
5. `ON_CAST / BORROW_PAST_POWER` — +40% công kích trong 3 lượt. VD: Tự Tại Thiên Địa Tiền.
6. `ON_CAST / UNRAVEL_RIDDLE` — hóa giải tối đa 2 buff của mục tiêu (kèm log ngắt tụ lực). VD: Giải Mê Cổ (đồng thời gây `SILENCE` từ `active.apply_status`).
7. `ON_CAST / CHAOS_GAMBLE` — gieo 1 lợi trong 3 (crit ×5 / bất tử 1 lượt / trừ 90% HP hiện tại của địch, giữ lại tối thiểu 1 HP) kèm 1 giá trong 3 (tư chất −10% / tuổi thọ −5 năm / nguyên thạch −50%). VD: Cổ Tình Yêu.
8. `ON_HIT / BYPASS_SHIELD` — đòn đánh xuyên Khiên Ảo, trừ thẳng HP (không trừ thủ). VD: Cổ Phá Không.
9. `BEFORE_HIT / FIRST_TURN_EVADE` — né đòn đánh đầu tiên ở Lượt 1, đòn multi-hit/AoE tiếp theo vẫn trúng. Định Tiên Du rank 9 mang mechanic này (rank 6–8 chỉ có `portal_create` + `movement`).
9b. `apply_status / INVINCIBLE` (**Bất Hoại**) có `chance` — miễn nhiễm theo tỉ lệ mỗi lần dùng, tỉ lệ và số lượt tăng theo rank (`chance + 0.05×(rank−1)`, `turns + (rank−1)/3`). VD: Hoang Mộ Cổ rank 4 (`chance: 0.35, turns: 1`) → 50%/2 lượt + khiên 600. Trượt hiện log `STATUS_MISS`.

9c. **Hai lớp bất tử — đừng nhầm.** `INVINCIBLE` = **Bất Hoại**: không dính sát thương nào, kể cả các đường ghi thẳng HP không đi qua cổng đòn đánh (DoT BURN/BLEED/POISON, `%MaxHP` của `damage_hp_percent`, Hồn Thương `SOUL_WOUND`, Phản Phệ `BACKLASH`, Gai Phản Đòn `THORNS`, burst BLEED khi tẩy trừ). Nhấn mạnh hơn: **không** gỡ được (`CategoryBuff`). `LAST_STAND` = **Bất Tử**: vẫn nhận sát thương và log damage như bình thường, chỉ có HP sàn 1. Nhấn nhẹ hơn nhưng **bị tẩy trừ được** (`CategoryImmortal`) — đổi lấy một lượt sống. Cả hai đều tự buff (không qua cổng kháng CC) và đều bị **Sát Khí** tầng $>10$ xuyên: Bất Hoại ghi log `BLOCKED_BREAK` rồi chết, Bất Tử bị bỏ qua sàn và chết. `GU_BACKLASH` (tự trừ vì dùng Cổ vượt cảnh giới) không được hai effect nào bảo vệ.
10. `COMBAT_PASSIVE / DOUBLE_DAMAGE_DEALT` (`PASSIVE` cũ vẫn dùng được) — nhân đôi mọi sát thương gây ra. VD: Tiên Thai Chí Tôn (kèm `path_conflict_reduction: 100`).
11. `active.cleansing_power` / `passive.tenacity` + `passive.cc_resist` — kháng và tự giải hiệu ứng bất lợi (xem mục Kháng hiệu ứng và Tenacity ở mục 9). VD: Ngã Ý Cổ (`tenacity: 0.35`, `cc_resist: 0.15`).
12. `active.damage_hp_percent` + `shield_flat_reduce_percent` + `disable_gu_next_turn` — trừ thẳng % MaxHP (xuyên khiên), tước % khiên hiện tại, phong ấn Cổ mục tiêu lượt sau (`GU_SEAL`, Tenacity có thể tự giải). VD: Lãng Quên Bản Ngã Cổ (40%/40%/1 lượt + status `FORGET_EGO`).
13. `active.forget_reality_chance` + `target_damage_reduce_percent` + status `EVASION` + `FORGET_REALITY` — 40% proc: bản thân tự buff né tránh 50% (1 lượt), địch dính suy yếu 40% sát thương (2 lượt). VD: Lãng Quên Thực Tại Cổ.
13b. **`EVASION` — nguồn né tránh duy nhất trong combat.** Gắn qua `apply_status` (`{"id":"EVASION","value":X,"turns":N,"chance":C}`) trên block `ON_ACTION`, y hệt `INVINCIBLE`: tự buff nên không qua cổng kháng CC, `CategoryBuff` nên không bị dispel, chỉ chịu tỉ lệ `chance` × Caster Attenuation $K_{\text{caster}}$ và rank-scale (`chance + 0.05×(rank−1)` cap 0.95, `turns + (rank−1)/3`). `value` là **% né**; nhiều stack **gộp max** (không cộng dồn) và cap **75%**. Roll né diễn ra **một lần cho cả đòn** (đa đòn không roll lại), trước cả roll Mù. Đối kháng: **Sát Khí** (`SAT_KHI_GUARANTEE`, `sat_khi_level > 10`) bỏ qua hoàn toàn ⇒ không né được đòn từ đó. Chỉ số `accuracy` cũ đã bị gỡ vì nó trừ thẳng (giá trị tới 1000) vào xác suất 0..1 khiến né tránh không bao giờ xảy ra.
14. `STAT_MOD PATH_RESISTANCE` kèm `path` + `UNIQUE IMMUNE_DREAM_STATUS` — giảm sát thương khi đòn đánh mang Gu path tương ứng (nhận diện theo `Gu.path`); miễn nhiễm mọi status Mộng đạo (`SLEEP`, `CONFUSE`...). VD: Lãng Quên Thiên/Địa/Mộng Cổ (65%/65%/50%).
15. `BEFORE_ENEMY_ATTACK / FORGET_INTENT_INTERRUPT` — khi địch tấn công, 40% phong ấn Cổ địch lượt sau (tối đa 8 lần thành công, sau đó phản phệ phong ấn chính mình). VD: Lãng Quên Ý Niệm Cổ.
16. `BEFORE_ENEMY_ATTACK / FORGET_WILL_CONFUSE` — khi địch tấn công, 35% quên mục tiêu (15% trúng chủ Cổ, 35% tự đánh mình, 50% hụt) + 20% phản phệ phong ấn Cổ mình 2 lượt. VD: Lãng Quên Chí Cổ.
17. `active.damage_path` + `stat_base_attack` trong active — ghi đè hệ đòn đánh + cộng flat vào sát thương (VD Trầm Luân Tiên Cổ: 250M + `SLEEP` + hệ Dream). Lưu ý: `status_duration_ticks` ≤ 5 đi qua pipeline tier legacy.

### Cơ chế ngoài chiến đấu (Handlers)
1. `ON_CONSUME / ADD_LIFESPAN_YEARS (params.value)` — tiêu hao để cộng vĩnh viễn N năm tuổi thọ. VD: Thọ Cổ 100 Năm (`value: 100`).
2. `ON_CONSUME / SACRIFICE_NPC + CAPABILITY / SACRIFICE_APTITUDE` — hiến tế để tăng vĩnh viễn tư chất Không khiếu (mỗi lần +5, trần 99). VD: Huyết Sọ Cổ.
3. `CAPABILITY / LOOT_PREVIEW` (+ `active.range`) — hiển thị tỉ lệ rớt đồ của quái trên bản đồ và mở rộng tầm khám phá. VD: Cổ Thiên Cơ (`range: 1000`).
4. `ON_TICK / THOUGHT_REGEN_TICK (params.value)` — hồi Niệm lực mỗi giờ cho Cổ sư trang bị. VD: Trí Tuệ Cổ.
5. `ALWAYS_ON / DEDUCTION_INSTANT_FINISH` — Cổ trang bị có tỉ lệ đốn ngộ, lập tức hoàn thành thôi diễn bí phương (100%). Tỉ lệ = `base_chance + (rank Cổ - rank đích) * per_rank_step`, chặn trong `[min_chance, max_chance]`; mỗi lần thôi diễn chỉ roll 1 lần. VD: Đốn Ngộ Cổ.
6. `COMBAT_PASSIVE / REVERSE_CALAMITY` — miễn nhiễm tuyệt đối mọi trạng thái bất lợi từ địch (choáng/ngủ/phong ấn/cướp Cổ...), damage vẫn nhận. VD: Nghịch Chuyển Cổ.
7. `ON_BATTLE_START / SUPPRESS_PRIORITY` — trấn áp hiệu ứng ưu tiên hành động của địch (`SEIZE_INITIATIVE`, né đòn đầu Định Tiên Du); trấn áp thành công thì người mang Cấm Cổ đi trước Lượt 1, song Cấm thì ai nhanh hơn đi trước. VD: Cấm Cổ.

## 18. Bối cảnh & Cốt truyện nguyên tác (Cổ Chân Nhân)

Thông tin nguyên tác được tổng hợp từ tiểu thuyết **Cổ Chân Nhân**:

### Tóm tắt cốt truyện
Cổ Nguyệt Phương Nguyên, một đại ma đầu bị ép đến đường cùng sau 500 năm, đã sử dụng Xuân Thu Thiền để trọng sinh. Tại một thế giới tàn khốc "nhược nhục cường thực", nơi con người nuôi Cổ, luyện Cổ và dùng Cổ, hắn bất chấp luân lý, sẵn sàng tính kế thiên hạ để truy cầu con đường vĩnh sinh. Đồng hành cùng Phương Nguyên trong giai đoạn đầu là Bạch Ngưng Băng - một chân ma đa mưu túc trí.

### Thế giới quan (Cổ giới)
- **Thái Cổ Cửu Thiên:** Xích Thiên, Chanh Thiên, Hoàng Thiên, Lục Thiên, Thanh Thiên, Lam Thiên, Tử Thiên, Bạch Thiên, Hắc Thiên. Về sau chỉ còn Hắc Thiên và Bạch Thiên (trải qua U Hồn Ma Tôn hoàn toàn hợp nhất thành U Thiên).
- **Ngũ đại địa vực:** Nam Cương, Đông Hải, Trung Châu, Bắc Nguyên, Tây Mạc.

### Cấp bậc tu luyện & Tư chất
- **Đẳng cấp Cổ Sư:** Chia làm 9 chuyển, mỗi chuyển có 4 tiểu cảnh giới (Sơ giai, Trung giai, Cao giai, Đỉnh phong). Từ ngũ chuyển trở xuống là phàm, lục chuyển thành tiên, cửu chuyển xưng tôn.
- **Tiên Nguyên:** Lục chuyển (Thanh Đề), Thất chuyển (Hồng Tảo), Bát chuyển (Bạch Lệ), Cửu chuyển (Hoàng Hạnh). Vượt qua các cấp độ thiên kiếp, hạo kiếp, vạn kiếp để thăng tiến.
- **Tư chất:** Thập tuyệt thể (100% - nhận ngẫu nhiên 1 Thể Chất đặc biệt khi tạo nhân vật), Giáp đẳng (80-90%), Ất đẳng (60-70%), Bính đẳng (40-50%), Đinh đẳng (20-30%).

### Các thế lực phe phái nổi bật
- **Trung Châu:** Thiên Đình (thế lực mạnh nhất từ viễn cổ, do Nguyên Thủy Tiên Tôn sáng lập, tồn tại gần 4 triệu năm) và Thập Đại Cổ Phái.
- **Bắc Nguyên:** Trường Sinh Thiên (Cự Dương Tiên Tôn sáng lập), các Hoàng kim gia tộc (Dược gia, Cung gia, Lưu gia...), và dị nhân (Lang Gia phái).
- **Nam Cương:** Vũ gia, Hạ gia, Ba gia... và Ảnh Tông (do U Hồn Ma Tôn sáng lập nhằm đối kháng Thiên Đình).
- **Đông Hải:** Lưỡng thiên liên minh, Tống gia, Thẩm gia, Long đình.
- **Tây Mạc:** Phòng gia, Lâm gia, Tiêu gia...

### Tu hành lưu phái & Cảnh giới
- **Lưu phái:** Nhân đạo (Nhân Tổ); Khí đạo, Nô đạo (Nguyên Thủy Tiên Tôn); Trí đạo, Tinh đạo (Tinh Túc Tiên Tôn); Lực đạo, Biến hóa đạo (Cuồng Man Ma Tôn); Luật đạo (Vô Cực Ma Tôn); Hồn đạo (U Hồn Ma Tôn); Thâu đạo (Đạo Thiên Ma Tôn),...
- **Cảnh giới lưu phái trong game:** Phàm Thai ➔ Đại Sư (10.000 Chân Ý) ➔ Tông Sư (100.000) ➔ Đại Tông Sư (1.000.000) ➔ Vô Thượng Đại Tông Sư (10.000.000). Đây là năm bậc được triển khai trong hệ thống hiện tại.

## 19. Hướng Dẫn Tân Thủ (Wiki Guide)

Trang `/wiki` là nơi tra cứu hướng dẫn và bách khoa trong game; nội dung hướng dẫn dùng các tệp trong `frontend/src/lib/wiki/`, còn danh mục Cổ, vật phẩm, quái và khu vực lấy từ dictionary backend. Phần dưới là lộ trình tân thủ ngắn gọn:

### Thuật ngữ cốt lõi:
- **Không Khiếu & Tiên Khiếu:** Phàm nhân sử dụng Không Khiếu tích lũy Chân Nguyên chiến đấu. Cổ Tiên (Lục Chuyển trở lên) sở hữu **Tiên Khiếu** (thế giới nhỏ độc lập bên trong cơ thể) tự sản sinh Tiên Nguyên với 4 phẩm cấp: Thấp, Trung, Cao, Cực Hạn dựa trên Nhân Khí lúc Thăng Tiên.
- **Đạo Ngân (Dao Marks):** Vết khắc của quy luật tự nhiên, chia thành 49 lưu phái. Đạo Ngân cùng phái khuếch đại hành động dùng lưu phái đó; Đạo Ngân ngoại phái có thể tạo xung đột. Hiệu quả được tính theo từng hành động và có thể chịu điều hòa từ Chân Ý hoặc effect miễn/giảm xung đột, không phải mỗi điểm luôn cộng thẳng 1% sát thương và phòng ngự.
- **Cảnh Giới Lưu Phái (Attainment):** Độ ngộ Chân Ý gồm 5 bậc: Phàm Thai ➔ Đại Sư ➔ Tông Sư ➔ Đại Tông Sư ➔ Vô Thượng Đại Tông Sư. Điểm tích lũy riêng theo từng lưu phái; các nguồn gồm sử dụng Cổ, chiến đấu, sát chiêu, Luyện Cổ và phần thưởng một số Phó bản. Mốc hiện tại: Đại Sư=10.000, Tông Sư=100.000, Đại Tông Sư=1.000.000, Vô Thượng=10.000.000. **Để đột phá lên Rank 7/8/9 khi vượt Kiếp Nạn, cần ít nhất 1 lưu phái đạt Tông Sư/Đại Tông Sư/Vô Thượng tương ứng.** Cảnh giới còn điều hòa Xung Đột Đạo Ngân và tăng hiệu quả theo cơ chế của từng hệ thống.
- **Bản Mệnh Cổ (Vital Gu):** Cổ trùng gắn liền sinh mệnh, không bao giờ mất khi thất bại dã ngoại.
- **Độ No (Hunger):** Cổ trùng giảm Độ No mỗi 3 giờ. Cần cho ăn đúng loại thức ăn (thịt thú, khoáng thạch...), nếu về 0 Độ No Cổ sẽ **chết vĩnh viễn**. Quy luật này áp dụng cả với Cổ đang nằm trong **Bảo khố Tông Môn**: Cổ trong kho vẫn hao Độ No mỗi 3 giờ (ngang mức hoạt động), về 0 sẽ chết và bị xóa khỏi kho; khi đổi ra, Cổ nhận đúng Độ No hiện tại chứ không được reset đầy.
- **Sát Chiêu (Killer Moves):** Kỹ năng ghép từ 1 Cổ Hạch Tâm (Core Gu) và 0-8 Cổ Phụ Trợ (Support Gu) tăng sát thương, giảm hao Chân Nguyên, thêm khống chế (choáng, đóng băng, câm lặng) hoặc thời gian tụ lực.
- **Kiếp Nạn (Tribulation):** Cổ Tiên phải chống chọi các kiếp nạn (Địa Tai, Thiên Kiếp...) định kỳ bảo vệ tài nguyên Tiên Khiếu. Vượt qua kiếp nạn nhận được lượng lớn Đạo Ngân.

### Các hoạt động chính:
1. **Tạo nhân vật & Thức tỉnh Linh Căn:** Chọn tên và thức tỉnh Linh Căn ngẫu nhiên.
2. **Trùng kích Vách Khiếu:** Đột phá tu vi phàm nhân từ Nhất Chuyển đến Ngũ Chuyển sử dụng Nguyên Thạch.
3. **Khai thác mỏ & Thám hiểm:** Đào Nguyên Thạch dã ngoại (coi chừng Quái Mỏ) và mở bản đồ sương mù nhặt dược thảo.
4. **Lắp ráp Sát Chiêu & Luyện Cổ:** Chế tạo nâng cấp Cổ trùng tại lò luyện và tự do phối hợp Cổ trùng tạo Sát Chiêu độc bản.
5. **Ải Cốt truyện & Đấu Trường:** Đi ải cốt truyện và khiêu chiến Cổ sư khác kiếm Chiến Công đổi đồ.
6. **Thương hội & Tông môn:** Đấu giá vật phẩm tự do tại Bảo Hoàng Thiên và các hoạt động quyên góp, giao dịch bảo khố Tông Môn.
7. **Thăng Tiên:** Đột phá lên Lục Chuyển Cổ Tiên ở Ngũ Chuyển Đỉnh Phong dựa trên Tư chất và tích lũy Nhân Khí (yêu cầu tối thiểu 100k Nhân Khí, xác suất gốc theo tư chất + bonus Nhân Khí, tối đa 90%). Thất bại phế bỏ tu vi về Nhất Chuyển.

### Giai đoạn Phàm Nhân (lv0)

Khi tạo tài khoản, người chơi bắt đầu ở **Phàm Nhân (rank 0)**, `needs_onboarding = true`. Trước khi chọn Cổ trùng và khu vực, màn hình "Lộ Trình Tân Thủ — Phàm Nhân" hiện ra để hướng dẫn cơ bản về Không Khiếu, Độ No, Thú/Cổ Trùng, khai thác, luyện Cổ và Cổ Chân Nhân tổng quan. Hoàn tất gọi `POST /api/player/complete-tutorial` → nhân vật lên **Nhất Chuyển (rank 1)**. Sau đó luồng `needs_onboarding` kích hoạt OnboardingModal (chọn Đại Vực khởi đầu + Cổ trùng Tấn công/Phòng thủ + Bản Mệnh), rồi gọi `POST /api/player/complete-onboarding` để chốt region/tọa độ và giải phóng mọi cơ chế chơi. Trong suốt giai đoạn `needs_onboarding = true`, mọi yêu cầu `POST/PUT/DELETE` ở phía server đều bị chặn (403) bởi middleware `RequireOnboarded`; chỉ có `complete-tutorial` và `complete-onboarding` được miễn.

## 20. Điều hướng giao diện và lộ trình chơi

Sau khi đăng nhập, đại sảnh chia tính năng thành các nhóm:

- **Phát triển nhân vật:** Nhân vật, Túi Cổ, Luyện Cổ, Nhiệm vụ & Thành tựu, hồ sơ/referral.
- **Tiên đạo:** Tiên Khiếu mở từ Lục Chuyển; tại đây quản lý node, dị nhân, khu định cư và kiếp nạn.
- **Kinh tế:** Cát Bảo Các, Thương Đội/Cửa hàng, Bảo Hoàng Thiên và giao dịch trực tiếp. Tab nạp trong Cát Bảo Các có thể bị tắt; trang nạp riêng nằm tại `/topup` khi hệ thống thanh toán được vận hành.
- **Chiến đấu/hoạt động:** Nô Thú, Phòng Tập, Võ Đài, Đấu Trường Sinh Tử, Mỏ, Bí–Mộng Cảnh, Giải Trí và Ngũ Vực.
- **Theo dõi:** Bảng Xếp Hạng, Lịch Sử Chiến Đấu, Ký Sự Tu Hành, thông báo và Feedback.

Ô **Sát Chiêu** trên đại sảnh hiện được đánh dấu đang phát triển/khóa dù backend và màn hình chiến đấu đã có dữ liệu sát chiêu. Hãy coi trạng thái hiển thị trên đại sảnh là nguồn chính xác về khả năng truy cập ở bản build đang chơi.

## 21. Tiên Khiếu: kinh tế và xã hội Dị Nhân

Sau khi thành Tiên, Tiên Khiếu không chỉ là lưới đặt công trình:

1. Đặt node tài nguyên/phòng thủ, thu hoạch riêng hoặc tất cả, sửa tất cả node hỏng và nâng từng node. Node tối đa cấp 5; mỗi cấp tăng 500 HP tối đa. Node tài nguyên tăng 50% sản lượng/cấp, node phòng thủ tăng 25% chỉ số/cấp.
2. Phân công dân số dị nhân còn khả dụng vào node tài nguyên. Sản lượng phụ thuộc dân số, cấp node, hiệu suất chủng tộc, mức lao động và trạng thái node; node bị phá hủy không sản xuất.
3. Chọn cường độ **LOW / NORMAL / HIGH / EXPLOIT**. Mức cao cho sản lượng nhanh hơn nhưng làm giảm hạnh phúc, tăng áp bức/nguy cơ phản loạn; mức thấp giúp xã hội hồi phục.
4. Xây khu định cư sẽ đưa 500 cư dân ban đầu cùng kho lương; dân số sinh trưởng mỗi 5 phút và bị giới hạn bởi sức chứa. Có thể tạo hàng đợi luyện chế cho Mao Dân xử lý mỗi 10 phút, tối đa một lần thử/lệnh/tick. Cổ càng cao cấp càng cần nhiều lao động nhàn rỗi: **số Mao Dân = cấp Cổ × 100** (Nhất Chuyển 100, Ngũ Chuyển 500, Lục Chuyển 600, Cửu Chuyển 900). Lao động được giữ chỗ cho toàn tick nên không bị dùng trùng giữa các lệnh. Lần thử tiêu hao nguyên liệu và nguyên thạch dù thất bại; thành công tạo Cổ và tăng tiến độ. Lệnh thiếu tài nguyên sẽ chờ và tự thử lại; hiện chưa nhận bí phương cần Cổ khác làm nguyên liệu.
5. Xem và nâng cấp khu định cư theo từng chủng tộc. Theo dõi dân số, sức chứa, lương thực, nhà ở, an ninh, hạnh phúc, thịnh vượng và phản loạn; thiếu lương thực hoặc hạnh phúc thấp làm tăng tử vong.
6. Khi một chủng tộc đủ đông, xã hội có cơ hội sinh Cổ Sư dị nhân. Có thể giao vai trò hoặc giải tán từng NPC; thiên phú, trung thành, rank và lưu phái của họ khác nhau.

Mô phỏng xã hội cập nhật nền sau mỗi vài phút, vì vậy số liệu có thể thay đổi dù người chơi không bấm hành động. Đừng dùng `EXPLOIT` dài hạn nếu kho lương, hạnh phúc và an ninh chưa ổn định.

## 22. Nhiệm vụ, thành tựu và phần thưởng tài khoản

- **Nhiệm vụ:** xem yêu cầu/tiến độ/phần thưởng rồi hoàn thành khi đủ điều kiện. Một số tiến độ được ghi tự động từ hoạt động gameplay.
- **Thành tựu:** nhận thưởng khi đạt điều kiện; danh hiệu và khung chỉ chọn được sau khi đã nhận thành tựu tương ứng. Danh hiệu đang trang bị tăng 2% sát thương PvE trước một loại quái cố định và tăng 5% sản lượng Tiên Khiếu. Hiệu ứng được chụp khi bắt đầu trận, áp dụng mọi PvE phù hợp và không áp dụng PvP. Admin gỡ thành tựu sẽ tháo danh hiệu/hiệu ứng, không thu hồi thưởng đã nhận.
- **Nhiệm vụ hằng ngày:** server tạo 5 nhiệm vụ theo Rank lúc 00:00 giờ Việt Nam, chọn từ hoạt động khám phá, PvE, tu luyện, luyện Cổ và thu hoạch. Tiến độ chỉ ghi từ hành động được server xác nhận; nhiệm vụ, tiến độ và thưởng chưa nhận hết hạn khi sang ngày mới. Bấm **Nhận** để nhận thưởng.
- **Referral:** xem mã và thống kê, liên kết mã giới thiệu hợp lệ, sau đó nhận thưởng khi đạt điều kiện.
- **Gift code:** nhập trong giao diện tài khoản/tiện ích; mỗi code tuân theo thời hạn, giới hạn và phần thưởng do Admin cấu hình.
- **Thông báo:** chuông hiển thị số chưa đọc, mở chi tiết từng mục, đánh dấu một nhóm hoặc tất cả là đã đọc.

## 23. Phong Ấn Cổ

Trong **Tiện Ích → Phong Ấn Cổ**, có thể tìm kiếm/lọc túi Cổ rồi:

- phong ấn một Cổ bằng số **Đá Phong Ấn** tương ứng Chuyển của nó (tối thiểu 1);
- giải phong miễn phí;
- Cổ `SEALED` vẫn đói và vẫn có thể chết, nhưng tốc độ giảm độ no chậm hơn trạng thái hoạt động;
- Cổ phong ấn không được dùng như Cổ ACTIVE trong chiến đấu.

Phong ấn thích hợp cho Cổ ít dùng nhưng quý; vẫn phải kiểm tra và cho ăn định kỳ.

## 24. Lịch sử, replay và Ký Sự

- **Lịch Sử Chiến Đấu** liệt kê trận gần đây (PVP và các trận PVE thua), kết quả/đối thủ và cho mở báo cáo hoặc replay log chi tiết. Mỗi lượt lưu lại HP, chân nguyên, khiên, công/thủ/tốc độ hiện tại và các trạng thái còn hiệu lực để tiện xem lại diễn biến.
- **Ký Sự Tu Hành** ghi các hành động quan trọng như tu luyện, luyện Cổ, giao dịch, chăm Cổ và thay đổi tài sản. Có tìm kiếm tự do, lọc loại hành động/trạng thái và phân trang.
- Đây là hai nơi đầu tiên nên kiểm tra khi cần hiểu vì sao tài nguyên, Cổ hoặc kết quả trận đấu thay đổi.

## 25. Bảng Xếp Hạng

Các hạng mục hiện có gồm **Tu vi**, **Nguyên Thạch**, **Cổ mạnh nhất**, **Tiên Nguyên Thạch** và **Sát Khí**. Người chơi có thể ẩn/hiện mình trên các bảng thông thường; bảng Sát Khí bắt buộc công khai với người đang có Sát Khí. Dữ liệu xếp hạng được cập nhật theo chu kỳ ngày, nên không nhất thiết đổi ngay sau một hành động.

## 26. Bảng Quy Đổi Đơn Vị Số Lượng (Vạn, Ức, Kinh...)

Hệ thống rút gọn hiển thị số lượng lớn (sát thương, khiên, HP, tài nguyên) sử dụng thang đơn vị truyền thống Á Đông (mỗi cấp bậc tăng $10^4 = 10.000$ lần):

| Đơn vị | Lũy thừa ($10^n$) | Giá trị số | Diễn giải |
| --- | --- | --- | --- |
| **Vạn** | $10^4$ | 10.000 | 10 nghìn |
| **Ức** | $10^8$ | 100.000.000 | 100 triệu (1 vạn vạn) |
| **Triệu** | $10^{12}$ | 1.000.000.000.000 | 1 nghìn tỷ (1 triệu triệu / 1 vạn ức) |
| **Kinh** | $10^{16}$ | $10^{16}$ | 10 triệu tỷ (10.000 Triệu) |
| **Cai** | $10^{20}$ | $10^{20}$ | 100 nghìn tỷ tỷ |
| **Tử** | $10^{24}$ | $10^{24}$ | 1 triệu tỷ tỷ |
| **Nhưỡng** | $10^{28}$ | $10^{28}$ | 10 tỷ tỷ tỷ |
| **Câu** | $10^{32}$ | $10^{32}$ | 100 triệu tỷ tỷ tỷ |
| **Giản** | $10^{36}$ | $10^{36}$ | 1 trạch / 1 tỷ tỷ tỷ tỷ |
| **Chính** | $10^{40}$ | $10^{40}$ | 10.000 Giản |
| **Tải** | $10^{44}$ | $10^{44}$ | 10.000 Chính |
| **Cực** | $10^{48}$ | $10^{48}$ | 10.000 Tải |
| **Hằng Hà Sa** | $10^{52}$ | $10^{52}$ | Nhiều như số cát sông Hằng |
| **A Tăng Kỳ** | $10^{56}$ | $10^{56}$ | Vô số lượng lớn (Asamkhyeya) |
| **Na Do Tha** | $10^{60}$ | $10^{60}$ | Vô lượng số (Nayuta) |
| **Bất Khả Tư Nghị** | $10^{64}$ | $10^{64}$ | Vượt xa suy tư tư duy |
| **Vô Lượng** | $10^{68}$ | $10^{68}$ | Cực hạn vô lượng |

## 27. Trinh Sát & Phản Trinh Sát (Thiên Cơ)

Không ai được xem trộm hồ sơ từ xa. Muốn biết địch ở đâu, mạnh cỡ nào, giữ Cổ gì — phải **trinh sát** (gần) hoặc **suy diễn thiên cơ** (xa). Và kẻ bị dò luôn có đường **phản trinh sát**.

### Tác dụng ngoài bản đồ

- **Trinh Sát (INVESTIGATION):** mỗi điểm cộng 1 ô tầm nhìn trên bản đồ (`/api/explore/grid`),
  tối đa **25 ô** — Cổ trinh sát càng cao nhìn càng xa nhưng bị trần cứng.
- **Ngụy Trang (DISGUISE):** tổng từ rank + Cổ đang trang bị + cảnh giới Trí/Ám.
  Mỗi điểm giảm 1% cơ hội bị quái phục kích lân cận (`50 + rank*2 - disguise`, sàn 5%, trần 95%)
  và thu hẹp cửa sổ gặp quái ngẫu nhiên khi khám phá (`70 - disguise`, sàn 40%).

### Ba trụ điểm (áp cho cả hai phe) — cảnh giới là trụ số một

Sức dò chia làm ba trụ, **càng đầu bảng càng nặng**:

1. **Cảnh giới lưu phái (Trí đạo) — trụ số một.** Mỗi bậc cộng thẳng vào điểm và **không phụ thuộc tu vi**, nên người tu vi thấp vẫn có thể vượt kẻ tu vi cao nếu cảnh giới đủ sâu:

   | Bậc | Phàm Thai | Đại Sư | Tông Sư | Đại Tông Sư | Vô Thượng |
   | --- | --- | --- | --- | --- | --- |
   | Cộng phẳng | +0 | **+50** | **+140** | **+280** | **+470** |
   | Nhân thêm | ×1.00 | ×1.05 | ×1.12 | ×1.20 | ×1.30 |

   Mốc cảnh giới: 10.000 / 100.000 / 1.000.000 / 10.000.000 Chân Ý. Cảnh giới tích từ **4 nguồn**: dùng Cổ, chiến đấu, thi triển sát chiêu, và luyện Cổ.

2. **Cổ trùng.** Mỗi Cổ đang trang bị cộng theo phẩm bậc (Cổ rank *n* → `+5n`): Cổ 3 chuyển +15, Cổ 7 chuyển +35, Cổ 9 chuyển +45. Phe thủ đối xứng bằng Ngụy Trang (`DISGUISE ×5`) + `PERFECT_DISGUISE` (+80, Thái Độ Cổ) + `CONCEAL_GU` (+30), nên **cùng lượng Cổ thì hai phe cân bằng**.

3. **Đạo ngân — yếu nhất, bão hòa rất sớm.** Đạo ngân Trí/Tín (phe công) hoặc Che/Ám/Biến Hóa (phe thủ) chỉ tăng tối đa **+20%** điểm nền, đạt trần quanh 3.000 điểm đạo ngân và chỉ hoạt động từ Tiên nhân (bậc 6+). **Một bậc cảnh giới luôn đáng hơn toàn bộ đạo ngân** — đừng dồn đạo ngân thay vì luyện cảnh giới.

**Thứ tự ưu tiên đo được** (mức tăng tối đa mỗi trụ): cảnh giới **+578** · cổ trùng **+360** · đạo ngân **+68**.

Các cơ chế phụ:
   - **Ám đạo (thủ):** che bản thân, vị trí, khí tức; tăng rào cản khoảng cách.
   - **Tín đạo:** phe công khuếch đại giá trị manh mối; phe thủ phong tỏa, bóp méo manh mối (Tông Sư Tín đạo trừ tới 36%, tối đa một nửa).
   - **Vận đạo:** gieo xúc xắc vận số mỗi lần suy diễn — phe công tu Vận cao thì kết quả ổn định, phe thủ Vận cao thì kết quả lệch xấu cho kẻ dò.
   - **Thiên đạo (Tiên nhân):** che / bóp méo thiên cơ (màn Thiên Cơ), chỉ Trí/Thiên phe công mới xuyên được.
   - **Biến hóa đạo (thủ):** đổi dấu vết, thân phận, đạo ngân biểu hiện — khiến kết quả trả về là hàng giả.
   - **Manh mối (clue):** tên đúng +20, đúng vùng +15, đúng tông môn +10, Cổ nghi vấn đúng +10 (đều phải **đúng thật** mới tính), khí tức Cổ mục tiêu +30, dò mù không tên −40. Manh mối bị chặn trần ±40, không thể thay tu vi.

### Chênh lệch tu vi được tính dịu

Điểm chênh lệch cuối được nhân với hệ số `(10 / bậc tu vi mục tiêu)^0.6`, nên **lệch 2 bậc chỉ khiến màn sương chứ chưa chắc chắn phản phệ**. Dồn đạo ngân cũng không còn đủ sức bù nổi khoảng cách tu vi.

### Chi phí suy diễn

- Mỗi lần suy diễn tốn **Nguyên Thạch** (tối thiểu 100, càng nhiều điểm cộng càng cao) hoặc **Tiên Nguyên Thạch** (chỉ Tiên nhân). Không tốn Chân Nguyên.

### Bốn kết quả (chỉ so **điểm tổng cuối**)
| Chênh lệch | Kết quả |
| --- | --- |
| ≥ +30 (người) / +20 (Cổ) | **Đại thành công:** tọa độ thật, tu vi thật, toàn bộ Cổ trang bị; được gieo ấn ký |
| ≥ 0 / ≥ −10 (Cổ) | **Tiểu thành công:** vùng + tu vi ước lượng + 2 Cổ (tọa độ bị giấu) |
| ≥ −30 / ≥ −40 (Cổ) | **Màn sương:** mất tài nguyên, trắng tay |
| Thấp hơn | **Thiên cơ phản phệ:** mất **15% HP tối đa + 15% Hồn tối đa** và 20 Chân Nguyên (cố định), **lộ ngược tọa độ cho mục tiêu**. Sát thương tính theo tỉ lệ nên không bao giờ một lần dò khiến bạn tử vong, dù đối thủ có yếu hay mạnh |

Sau mỗi lần suy diễn, bảng **điểm của chính đạo hữu** (Cổ / cảnh giới / manh mối / vận / tổng) hiện đầy đủ để biết mình thiếu ở đâu — điểm phòng ngự của địch không bao giờ lộ.

### Suy tính Cổ trùng (ai giữ, ở đâu, luyện sao)
- Phàm Cổ dễ dò; điểm suy tính bị **chia theo bậc Cổ** (bậc 1-5 chia 1/2/3/4/5, Tiên Cổ bậc 6-9 chia 10/18/30/50) — nên Cổ càng cao càng vượt tầm, cá voi cũng phải dốc sức mới dò được Tiên Cổ. Phàm nhân dò Tiên Cổ gần như chắc chắn phản phệ.
- **Cổ trong tay người mang khí tức của chủ:** Cổ trang bị được chủ che full; Cổ nằm kho/ngủ được che một nửa. Muốn dò Cổ của Cổ Tiên ẩn danh, trước hết phải thắng được chính hắn.
- Cổ trong bảo khố tông môn khó theo cấp tông môn; Cổ treo chợ công khai dễ dò nhất.
- Tự sở hữu Cổ đó thì suy tính cộng hưởng trực tiếp, gần như luôn thấu triệt.

### Ấn ký truy tung & chém đứt manh mối
- Đại thành công suy diễn người cho phép **gieo ấn ký**: Trí Niệm Ấn (ngắn) hoặc Huyết Mạch Ấn (lâu). Giới hạn 3 mục tiêu (phàm) / 10 (tiên). Ấn ký cho tọa độ realtime trong radar.
- **Đoạn Tuyệt Thiên Cơ:** tốn 20 Chân Nguyên (cố định), chém toàn bộ ấn ký người khác gieo lên mình + xóa tàn dư khí tức tại ô đang đứng. Bị theo dõi mà không biết chém — chết lúc nào không hay.

## Quái thường trên World Map

Quái thường nay là thực thể sống riêng biệt: mỗi lần sinh có ID, HP, vị trí và trạng thái riêng. Sát thương không chí mạng được giữ lại và đồng bộ cho người chơi gần đó. Quái có thể phát hiện, đuổi, vào tầm đánh và quay lại điểm sinh khi mất mục tiêu hoặc vượt dây cương; quái không chọn mục tiêu ở khu an toàn hay vùng khác. Sau khi bị hạ, quái sinh lại bằng instance ID mới sau thời gian hồi sinh của điểm spawn; nếu `respawn_seconds` nhỏ hơn 1, thời gian mặc định là 300 giây. Client chỉ dự đoán hiệu ứng hình ảnh; HP, tử vong và phần thưởng luôn do server quyết định.

Người chơi bị hạ gục trong combat được hồi sinh tại HUB an toàn gần nhất thay vì xuất hiện ngẫu nhiên trong vùng quái. Sau khi phần thưởng combat đã lưu, máy chủ vẫn hoàn tất việc đánh dấu quái bị hạ với context riêng có timeout ngắn, kể cả khi kết nối HTTP của người chơi đã ngắt.
