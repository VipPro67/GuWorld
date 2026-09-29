# GuWorld Gameplay Wiki

Wiki này phản ánh các hệ thống đã có trong codebase hiện tại, diễn tả bằng khẩu khí Cổ Chân Nhân.

## 1. Tài khoản và nhân vật

- Cổ sư đăng ký bằng username/email/password hoặc đăng nhập Google nếu cấu hình OAuth.
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
| Cương Thi (`cuong_thi`) | **Bất tử tuổi thọ** (thọ nguyên không bao giờ cạn); **Cấm tự hồi phục** HP, Chân Nguyên/Tiên Nguyên, Nguyên Lực, Thần Hồn. Khi ở trạng thái Tiên Cương: gia tốc thời gian khóa ở `1.0`, Tiên Khiếu thành Tiên Khiếu Chết và bị suy thoái `time_rate` & `area_size` theo thời gian |

Danh sách chủng tộc được **data-driven** từ bảng `race_dictionary` (seed từ `data/Race.csv`): admin thêm chủng tộc mới bằng cách sửa CSV (thêm dòng `key,name,effects`) rồi Import bảng "Chủng tộc" trong Admin panel tab **Dữ liệu**. Effect format xem `docs/GAME_DATA.md`.

### Tuổi thọ & Trạng thái Tử Vong (is_dead)

- **Cơ chế Tuổi thọ**: Cổ sư tiêu hao thọ nguyên theo thời gian thực (tốc độ hao mòn = thời gian thực × `time_speed_multiplier`).
- **Rơi vào Cõi Chết (`is_dead`)**: Khi thọ nguyên giảm xuống $\le 0$, Cổ sư bị chuyển sang trạng thái tử vong (`is_dead = true`). Trong trạng thái này, mọi thao tác tu luyện, trùng kích, chiến đấu, thám hiểm và đào quặng bị phong tỏa hoàn toàn cho tới khi sử dụng **Thọ Nguyên Cổ** khôi phục tuổi thọ hoặc chuyển chủng tộc.
- **Tiên Cương (Chủng tộc Cương Thi)**:
  - **Bất tử tuổi thọ**: Thọ nguyên không bị suy giảm theo thời gian.
  - **Phong tỏa Hồi phục**: Bị cấm tự động hồi phục HP (`DisableHPRegen`), Chân Nguyên/Tiên Nguyên (`DisableManaRegen`), Nguyên Lực (`DisableResourceRegen`) và Thần Hồn (`DisableSoulRegen`).
  - **Tiên Khiếu Chết (Dead Aperture)**: Khi Cổ Tiên chuyển sang Cương Thi, gia tốc thời gian cá nhân `time_speed_multiplier` bị cố định ở `1.0` (không thể tăng tốc), đồng thời Tiên Khiếu ngưng trệ thành Tiên Khiếu Chết, liên tục bị suy thoái giảm dần diện tích (`area_size`) và tốc độ thời gian (`time_rate`) theo thời gian thực (~0.00175 rate/phút & ~416.67 area/phút).

**Giới tính:** Nam (`male`) / Nữ (`female`) — chưa có hiệu ứng (chỉ hiển thị).

**Trạng thái:**

| Trạng thái | Hiệu ứng | Thời hạn |
| --- | --- | --- |
| Trọng Thương (`severely_wounded`) | **-20%** tấn công, phòng thủ, tốc độ; **-50%** hồi phục nhận; không thể thám hiểm | 60 phút (hoặc đến khi được chữa trị) |
| Trúng Độc (`poisoned`) | **-10%** tấn công, **-10%** tốc độ | 30 phút |
| Kiệt Sức (`exhausted`) | **-30%** tốc độ, **-50%** hiệu suất tu luyện; không thể tu luyện | 10 phút |
| Phúc Trạch (`blessed`) | **+5** may mắn, **+10%** hiệu suất tu luyện | 60 phút |
| Hình Phạt Tử Vong (`death_penalty`) | **-50%** tấn công, phòng thủ, tốc độ | 30 phút |
| Tâm Thần Mệt Mỏi (`refine_fatigue`) | **-25%** tỉ lệ Luyện Cổ thành công (khi Luyện Tiên Cổ hoặc Cổ $\ge$ Rank bản thân) | 10 phút |
| Luyện Cổ Phản Phệ (`refine_backfire`) | **-50%** tỉ lệ Luyện Cổ thành công (khi cố luyện lúc mệt mỏi hoặc bị nổ lò) | 20 phút |

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

### Cảnh Giới Lưu Phái (Path Masteries)

- Trong quá trình tu hành, Cổ sư có thể tích luỹ điểm **Chân Ý** của các lưu phái (ví dụ: Kiếm Đạo, Hỏa Đạo, Trí Đạo...) để thăng tiến cảnh giới lưu phái.
- **Cách nhận điểm Chân Ý:**
  - **Sử dụng Cổ tiêu hao/Sát chiêu trong chiến đấu:** (Bao gồm cả đánh tay và đánh tự động/Simulator như khai thác mỏ, tàn tích). Mỗi lần thi triển thành công tốn Chân Nguyên: Cổ phàm +1 điểm, Tiên Cổ +10 điểm hệ tương ứng. (Lưu ý: Đánh Đấu Trường Ảo không được cộng điểm).
  - **Giới hạn mỗi trận:** điểm Chân Ý nhận từ chiến đấu đối với Cổ phàm và Sát chiêu phàm bị chặn ở **Rank × 10 điểm/trận** (đánh tay, đánh tự động và sát chiêu phàm đều tính chung). Riêng **Tiên Cổ và Sát chiêu Tiên đạo (Rank ≥ 6)** không bị giới hạn tối đa trong trận chiến, thỏa sức ngộ đạo không có trần giới hạn. Thưởng phó bản và luyện Cổ không tính vào giới hạn này.
  - *(Tương lai)* Sẽ có thêm cơ chế Dung luyện Cổ, Khám phá tàn tích đặc thù hoặc đọc truyền thừa.
- **Các mốc cảnh giới:** Sơ Học (< 100) -> Đại Sư (100) -> Tông Sư (500) -> Đại Tông Sư (2500) -> Vô Thượng Tông Sư (10000).
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
- **Tỉ lệ thành công phụ thuộc vào Tư chất và Nhân Khí:**
  - **Tỉ lệ gốc (`baseChance`):** Giáp: `20%`, Ất/Bính/Đinh: `5%` (tính từ `GetBreakthroughChance(6, Aptitude)` với phạt Rank 6 là 80%).
  - **Bonus Nhân Khí:** Mỗi 100.000 Nhân Khí cộng thêm `+20%` tỉ lệ (`qiBonus = (NhânKhí / 100.000) * 20%`).
  - **Giới hạn tối đa:** Tổng tỉ lệ thành công tối đa là **90%**.
  - **Ví dụ tại 100.000 Nhân Khí:** Giáp = 40%, Ất/Bính/Đinh = 25%. Để đạt max 90%, Tư chất Giáp cần 350.000 Nhân Khí, các tư chất còn lại cần 425.000 Nhân Khí.
- **Thành công:** lên Lục Chuyển Cổ Tiên, nhận 10 Tiên Nguyên Thạch, Đạo Ngân Thiên/Địa = 100. Kiếp đầu tiên giáng lâm vào **năm nội bộ thứ 10**, tương đương khoảng **2–10 giờ đời thật** tùy phẩm chất Tiên Khiếu (xem mục Gia tốc thời gian). **Tiên Khiếu được khai mở ngay**, phẩm cấp được định đoạt theo Nhân Khí tại thời điểm thăng tiên (xem mục Tiên khiếu).
- **Thất bại:** Nhân Khí về 0, tu vi phế bỏ rơi về Nhất Chuyển Sơ giai, mất hết tiến độ phá vách.
- Cổ Tiên có tiên khiếu, đạo ngân, tiên nguyên/tài nguyên riêng.
- Tai kiếp có worker kiểm tra tự động mỗi 60 giây. Khi năm nội bộ đạt mốc tiếp theo, worker sinh kiếp và resolve.
- **Bảo hộ khi vắng mặt:** nếu đến mốc tai kiếp mà đạo hữu **đang offline**, tai kiếp sẽ không sinh quái và không phá hoại — Tiên Khiếu chuyển trạng thái **"Kiếp Lực Tích Lũy"** (`TRIBULATION_PENDING`). Khi đạo hữu online trở lại, tai kiếp mới giáng xuống (trong vòng 60 giây) và có **5 phút chuẩn bị** trước khi quái bắt đầu di chuyển/phá mỏ. Thời gian nội bộ vẫn trôi bình thường nên không thể trốn kiếp vô hạn; nếu giữ kiếp offline quá **24 giờ**, phần thưởng Đạo Ngân sẽ bị **giảm còn một nửa**. Bộ đếm 6 giờ tự kết toán chỉ tính thời gian online.
- **Quái tai kiếp không phá hủy trực tiếp mỏ tài nguyên.** Khi tiếp cận linh mạch tài nguyên, mỗi yêu tướng mỗi nhát làm giảm `hp` mỏ theo loại kiếp: Địa Tai -1, Thiên Kiếp -2, Hạo Kiếp -4, Vạn Kiếp -8, Hỗn Độn Đại Nạn -16. Đại Yêu di chuyển quanh quẩn và đánh mỏ với sát thương gấp đôi yêu tướng.
- **Combat tai kiếp dùng giao diện chiến đấu thông thường:** nhấn **"Xuất Trận Chiến Đấu"** trong bảng điều khiển tai kiếp sẽ mở màn chiến đấu (auto-combat) chống toàn bộ quái tai kiếp còn sống, mang theo đội hình Cổ như đánh mỏ/PvP/PvE. Giáp ảo (`shield_flat` từ Cổ) và **máu của đạo hữu tiêu hao trước**; sát thương tràn qua (vượt quá giáp + máu) mới trừ trực tiếp vào **Sinh lực Tiên Khiếu** (`immortal_apertures.aperture_hp`). Thanh **Máu Tiên Khiếu** hiển thị ngay trong màn chiến đấu và cập nhật theo từng lượt.
- **Cổ sư chỉ được xuất trận khi Sinh lực Tiên Khiếu > 5% Sinh lực tối đa.** Nếu Tiên Khiếu nguy cấp (≤ 5%), đạo hữu không thể ra tay — quái sẽ tiếp tục càn quét mỏ trong sự bất lực; hãy chờ Tiên Khiếu hồi phục hoặc kết toán vượt kiếp.
- **Kết toán tai kiếp:** Nếu kết toán khi còn quái trên bản đồ lưới (chấp nhận rủi ro hoặc hết hạn 6 giờ thực tế tự động kết toán), đạo hữu có nguy cơ thất bại. Khi vượt kiếp thất bại, bộ đếm số lượng tai kiếp (tổng số tai kiếp và số lượng theo từng loại) sẽ không tăng (chỉ tích lũy khi thành công), nhưng thời gian/mốc năm của tai kiếp kế tiếp (`next_tribulation_year`) vẫn sẽ được đẩy lên mốc tiếp theo để người chơi có thời gian hồi phục và không bị dồn dập tai kiếp ngay lập tức.
- **Kiếp nạn theo cấp bậc:**
  - Lục Chuyển: Địa Tai (10 năm), Thiên Kiếp (100 năm)
  - Thất Chuyển: Địa Tai (10 năm), Thiên Kiếp (50 năm), Hạo Kiếp (100 năm)
  - Bát Chuyển: Thiên Kiếp (10 năm), Hạo Kiếp (50 năm), Vạn Kiếp (100 năm)
  - Cửu Chuyển: Hạo Kiếp (10 năm), Vạn Kiếp (50 năm), Hỗn Độn Đại Nạn (100 năm)
- **Tự động đột phá:** Khi vượt thành công đủ 3 lần kiếp nạn cấp cao nhất của Chuyển hiện tại (Rank 6: 3 Thiên Kiếp; Rank 7: 3 Hạo Kiếp; Rank 8: 3 Vạn Kiếp) và đạt cảnh giới lưu phái yêu cầu, Cổ Tiên tự động lên Chuyển kế tiếp.
- Nhiều kiểu kiếp nạn đến cùng lúc: ưu tiên sinh kiểu hiếm nhất, nhưng đếm tất cả các kiểu đang đến hạn.

## 3. Không khiếu, tiên khiếu và đạo ngân

### Không khiếu/phàm nhân

- UI `Aperture` hiển thị Chân Nguyên, Sinh lực, Chuyển, tư chất, Cổ trang bị, nhiệm vụ và tu luyện.
- Chân Nguyên/Sinh lực được frontend ước tính theo thời gian từ snapshot server.

#### Hồi máu thụ động ngoài combat

- Sinh lực hồi liên tục theo thời gian thực, được cộng dồn cả khi offline (tính từ `last_mana_update`, áp dụng khi đạo hữu tương tác).
- **Tốc độ hồi:** `HP/giờ = (HP tối đa hiệu dụng × 5% + hp_regen từ effect) × (1 + healing_received)`, cap tại HP tối đa hiệu dụng (`base_hp` + `stat_base_hp` thụ động từ Cổ).
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
  - LOW: time rate 10-15x, Sinh lực 10.000, 30 node, 10 tiên nguyên/năm.
  - MID: time rate 20-28x, Sinh lực 20.000, 60 node, 20 tiên nguyên/năm.
  - HIGH: time rate 30-40x, Sinh lực 30.000, 90 node, 30 tiên nguyên/năm.
  - EXTREME: time rate 40-55x, Sinh lực 50.000, 100 node, 50 tiên nguyên/năm.
- **Khi mới khai mở**, Tiên Khiếu được tạo sẵn 2-4 mỏ tài nguyên ngẫu nhiên (tuỳ phẩm cấp):
  - LOW: 2 mỏ, MID: 3 mỏ, HIGH/EXTREME: 4 mỏ.
  - Các mỏ khởi đầu: Nguyệt Lan, Thanh Trúc, Ngọc Thạch, Nguyên Thạch, Kiếm Diệp, Hươu Linh.
- Cổ sư đặt thêm node tài nguyên theo toạ độ, thu hoạch, gỡ/xóa, sửa chữa node.
- **Thu hoạch tất cả** các mỏ đã đủ thời gian tích tụ cùng lúc; sản lượng giảm **20%** và Nhân Khí nhận được giảm còn **+1/mỏ** (so với thu hoạch từng mỏ).
- **Sửa chữa tất cả** các công trình hư tổn/phế tích cùng lúc; phí sửa mỗi công trình tăng **30%**.
- Hệ sinh thái node có kiểm tra xung đột đường tu/path và cân bằng plant/herbivore/carnivore.
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

Đạo Ngân là vết khắc cô đọng của quy luật tự nhiên tích lũy trên thân thể và Tiên Khiếu của Cổ Tiên qua **49 lưu phái** (tích lũy từ Độ Kiếp, Thăng Tiên, Nuốt Tiên Khiếu hoặc Luyện Cổ).

#### 1. Bảng Phân Loại 10 Nhóm Đạo (Dao Groups)
Các lưu phái trong cùng một nhóm hoàn toàn tương thích **100% (Xung đột = 0.0)**:
- **ELEMENTAL (Nguyên Tố)**: Hỏa, Thủy, Băng, Phong, Lôi, Thổ, Vân, Ám, Quang, Nguyệt, Nhật, Tinh.
- **SPACE_TIME (Thời Không)**: Vũ (Không gian), Trụ (Thời gian), Hư (Hư vô).
- **RULE_ORDER (Quy Tắc & Hạn Chế)**: Luật, Giới (Giới hạn), Phong (Phong ấn), Trận.
- **KINETIC_WEAPON (Vũ Khí & Động Năng)**: Kiếm, Đao, Binh, Thanh (Âm thanh), Chấn (Chấn động).
- **BODY_LIFE (Sinh Mệnh & Thể Phách)**: Lực, Biến Hóa, Cốt, Cương (Thân thể/Cương thi).
- **SOUL_MIND (Linh Hồn & Tâm Thức)**: Hồn, Trí, Nô (Nô dịch), Mộng, Mị (Mị hoặc), Ảo, Tình (Tình cảm).
- **INFORMATION_STEALTH (Ẩn Thao & Tình Báo)**: Tín (Tin tức), Ẩn (Tàng hình), Huyễn (Biến ảo), Đạo (Trộm cắp).
- **ORGANIC_CORRUPTION (Ăn Mòn & Huyết Độc)**: Huyết, Độc, Mộc, Thực (Ăn nuốt).
- **CRAFT_ALCHEMY (Chế Tác & Vật Chất)**: Luyện, Đan, Kim (Kim loại).
- **HEAVEN_HUMAN (Chí Cao Vận Mệnh)**: Thiên, Nhân, Vận, Họa (Tai họa/Vẽ), Hạn (Hạn hán).

#### 2. Cặp Tương Khắc & Bổ Trợ Đặc Thù (Pair Overrides)
- **Tương Khắc Cực Đoan (Xung đột = 1.0 - 1.2)**: Hỏa $\leftrightarrow$ Thủy/Băng; Quang $\leftrightarrow$ Ám; Độc $\leftrightarrow$ Mộc; Kiếm/Đao $\leftrightarrow$ Trận; Trí $\leftrightarrow$ Tình.
- **Cộng Hưởng Bổ Trợ (Xung đột = 0.2 - 0.3)**: Lực $\leftrightarrow$ Biến Hóa; Hồn $\leftrightarrow$ Nô; Hư $\leftrightarrow$ Ẩn; Luyện $\leftrightarrow$ Lực/Hỏa; Vận $\leftrightarrow$ Thiên/Nhân.

#### 3. Công Thức Tính Xung Đột & Điều Hòa Cảnh Giới
- **Ngưỡng An Toàn (Grace Threshold)**: Đạo ngân ngoại hệ $\le \max(1.000, 2\% \text{ tổng Đạo Ngân})$ hoàn toàn không gây xung đột.
- **Hiệu Quả Khuếch Đại Cảnh Giới (Attainment Efficiency)**: Cảnh giới Chân Ý giúp tăng cường uy lực Đạo Ngân đồng hệ:
  - Phàm Thai: $\times 1.00$
  - Đại Sư (Master): $\times 1.10$ (+10% uy lực)
  - Tông Sư (Grandmaster): $\times 1.20$ (+20% uy lực)
  - Đại Tông Sư (Great Grandmaster): $\times 1.30$ (+30% uy lực)
  - Vô Thượng (Supreme Grandmaster): $\times 1.45$ (+45% uy lực)
- **Hệ Số Điều Hòa Cảnh Giới (Realm Control Mitigation)**: Giảm trực tiếp lượng xung đột từ Đạo ngân ngoại hệ:
  - Đại Sư: Giảm **20%** xung đột
  - Tông Sư: Giảm **40%** xung đột
  - Đại Tông Sư: Giảm **60%** xung đột
  - Vô Thượng Đại Tông Sư: Giảm **80%** xung đột
- **Hệ số Xung Đột Khi Tấn Công (`DaoConflictMultiplier`)**: Nằm trong khoảng `[0.10, 1.00]`. Nếu không có Cảnh giới điều hòa, mang lượng lớn Đạo ngân xung đột có thể làm mất tới **90% sát thương**.
- **Màng Phòng Thủ Đạo Ngân Tự Nhiên (`NaturalDaoResistance`)**: Khi bị đòn tấn công dị hệ nhắm vào, Đạo ngân trên người Cổ Tiên phòng thủ tạo ra màng kháng tự nhiên:
  $$\text{Resistance} = \frac{\text{DefenseConflictDao}}{\text{DefenseConflictDao} + 50.000} \times 30\% \quad (\text{Trần tối đa } 30\%)$$
- **Đặc Tính Không Xung Đột (Non-Conflicting Body / Chí Tôn Tiên Thai)**: Bỏ qua 100% hình phạt xung đột khi tấn công (luôn gây 100% sát thương), đổi lại không có màng kháng tự nhiên khi phòng thủ.

#### 4. Ví Dụ Tính Toán Chi Tiết (Step-by-Step)

- **Ví dụ 1: Khuếch Đại Đạo Ngân & Hiệu Suất Cảnh Giới (Attainment Efficiency)**
  - Cổ Tiên Hỏa Đạo có $2.000$ Đạo Ngân Hỏa $\rightarrow \text{daoAmp} = 1.0 + 2000 / 100.0 = 21.0$ ($+2.000\%$).
  - Phàm Thai ($\text{Eff} = 1.00$): $\text{totalDaoMult} = 1.0 + (21.0 - 1.0) \times 1.00 = \mathbf{21.0\times}$.
  - Đại Sư ($\text{Eff} = 1.05$): $\text{totalDaoMult} = 1.0 + (21.0 - 1.0) \times 1.05 = \mathbf{22.0\times}$ ($+100\%$ bonus Chân Ý).
  - Tông Sư ($\text{Eff} = 1.12$): $\text{totalDaoMult} = 1.0 + (21.0 - 1.0) \times 1.12 = \mathbf{23.4\times}$.
  - Vô Thượng ($\text{Eff} = 1.45$): $\text{totalDaoMult} = 1.0 + (21.0 - 1.0) \times 1.45 = \mathbf{30.0\times}$!

- **Ví dụ 2: Xung Đột Đạo Ngân & Khả Năng Hóa Giải Của Cảnh Giới (Attainment Control)**
  - Cổ Tiên Hỏa Đạo có $10.000$ Đạo Ngân Hỏa, nhưng tích lũy $20.000$ Đạo Ngân Thủy ($\text{compatibility} = 0.55 \rightarrow \text{xung đột} = 0.45$).
  - $\text{rawConflict} = 20.000 \times 0.45 = 9.000$.
  - *Trường hợp A: Phàm Thai ($\text{RealmControl} = 0\%$)*:
    - $\text{effConflict} = 9.000 \times (1.0 - 0.0) = 9.000$.
    - $\text{conflictRatio} = 9.000 / (10.000 + 50.000) = 0.15$.
    - $\text{conflictMult} = 1.0 / (1.0 + 0.15 \times 1.6) = \mathbf{0.806}$ ($\rightarrow$ **Mất 19.4% sát thương** do xung đột Thủy Hỏa!).
  - *Trường hợp B: Vô Thượng Đại Tông Sư ($\text{RealmControl} = 80\%$)*:
    - $\text{effConflict} = 9.000 \times (1.0 - 0.80) = 1.800$.
    - $\text{conflictRatio} = 1.800 / 60.000 = 0.03$.
    - $\text{conflictMult} = 1.0 / (1.0 + 0.03 \times 1.6) = \mathbf{0.954}$ ($\rightarrow$ **Chỉ giảm 4.6% sát thương**, Chân Ý hóa giải 76% tổn thất!).

- **Ví dụ 3: Màng Phòng Thủ Đạo Ngân Tự Nhiên (Natural Dao Resistance)**
  - Cổ Tiên Thủy Đạo sở hữu $100.000$ Đạo Ngân Thủy bị tấn công bởi sát chiêu Hỏa Đạo ($\text{compatibility} = 0.55 \rightarrow \text{xung đột thủ} = 0.45$).
  - $\text{defConflict} = 100.000 \times 0.45 = 45.000$.
  - $\text{Resistance} = 0.30 \times [45.000 / (45.000 + 50.000)] = 0.30 \times (45.000 / 95.000) = \mathbf{14.21\%}$.
  - $\rightarrow$ Đòn đánh Lửa của đối phương tự động bị triệt tiêu $14.21\%$ sát thương ngay khi chạm vào lớp màng Thủy Đạo tự nhiên.

- **Ví dụ 4: Tính Sát Thương Thực Tế Toàn Diện (A $\rightarrow$ Z)**
  - **Bên Công**: Sát thương gốc = $1.000$, DamageMult = $2.0$, Buff = $1.2$. Có $5.000$ Đạo Ngân Lực Đạo ($\text{daoAmp} = 51.0$), Tông Sư Lực Đạo ($\text{Eff} = 1.12, \text{Ctrl} = 25\%$), mang $10.000$ Đạo Ngân Hồn Đạo ngoại hệ ($\text{comp} = 0.88 \rightarrow \text{xung đột} = 0.12$).
    - $\text{totalDaoMult} = 1.0 + (51.0 - 1.0) \times 1.12 = 57.0$.
    - $\text{effConflict} = (10.000 \times 0.12) \times (1.0 - 0.25) = 900$.
    - $\text{conflictRatio} = 900 / (5.000 + 50.000) = 0.01636 \rightarrow \text{conflictMult} = 0.974$.
    - $\text{hitDmg} = 1.000 \times 2.0 \times 1.2 \times 57.0 \times 0.974 = \mathbf{133.243}$ sát thương.
  - **Bên Thủ**: Phòng thủ $\text{BaseDef} = 2.000$, Màng kháng tự nhiên $\text{Resistance} = 10\%$.
    - $\text{netDmg} = (133.243 - 2.000) \times (1.0 - 0.10) = 131.243 \times 0.90 = \mathbf{118.118}$ Sinh Lực mất thực tế!

#### 5. Ba Hướng Xây Dựng Nhân Vật (Build Paths)
1. **Chuyên Tu (Pure Focus)**: Dồn toàn lực vào 1 nhóm Đạo. Không lo xung đột, sát thương đồng hệ cực đại.
2. **Song Tu / Phụ Tu (Dual Synergic)**: 1 phái Chính + 1 phái Phụ tương thích cao (như Lực + Biến Hóa, Hồn + Nô), nâng Chân Ý lên Tông Sư để xóa bỏ xung đột, công thủ toàn diện.
3. **Đa Tu Toàn Năng (Omni Master)**: Luyện nhiều phái đối nghịch, bắt buộc phải cày Mộng Cảnh để đạt Đại Tông Sư / Vô Thượng ở các phái nhằm áp chế xung đột tối đa.

## 4. Cổ trùng và bọc đồ

Hiệu ứng chiến đấu đặc thù của Cổ trùng được lấy từ cột `effects` trong danh mục Cổ. ID của Cổ chỉ dùng để nhận diện, hiển thị, cấu hình thứ tự ra chiêu và tính tiêu hao; nó không tự kích hoạt một hiệu ứng ẩn. Vì vậy một Cổ mới có thể tái sử dụng mechanic hiện có chỉ bằng dữ liệu, còn mechanic mới phải được đăng ký và kiểm tra hợp lệ trước khi import.

Mỗi Cổ instance có:

- `instance_id`, `gu_id`.
- Độ No (giảm theo `HungerRate * TimeSpeedMultiplier` mỗi 3 giờ thực tế, Cổ trong Tiên Khiếu có time rate cao sẽ đói nhanh hơn).
- Status: `ACTIVE`, `SLEEPING`, `DEAD`.
- `equipped` và `is_vital`.

Khi Cổ chết đói, **Ký Sự Tu Hành** ghi bản rõ: độ no ngay trước khi chết, trạng thái lúc đó (`ACTIVE`/`SLEEPING`/`SEALED` — quyết định tốc độ hao hụt), lần cuối được cho ăn và tổng thời gian không được cho ăn của cá thể đó.

Cổ sư có thể:

- Cho ăn từng Cổ hoặc theo nhóm.
- Trang bị/tháo Cổ.
- Đặt Cổ bản mệnh.
- Chuyển trạng thái active/sleeping.
- Dùng Cổ tiêu hao hoặc Cổ hỗ trợ.
- Vứt bỏ Cổ.

**Tiên Cổ duy nhất:** Cổ từ **Lục Chuyển trở lên (Tiên Cổ)** là độc nhất toàn thiên hạ — mỗi `gu_id` chỉ tồn tại một instance trên toàn server. Các kênh nhận Cổ (Mở đá, Rương Mỏ Linh Thạch, Luyện Cổ, quái rơi đồ) đều loại trừ Tiên Cổ đã tồn tại (trong túi bất kỳ đạo hữu, đấu giá) trước khi roll; nếu hết ứng viên hợp lệ sẽ rơi vật phẩm/thạch bù trừ. Mỗi Tiên Cổ có một mã hiệu (`instance_id`) theo nó suốt đời — mua bán, trao đổi, gửi kho tông môn đều giữ nguyên mã hiệu. Thả, bỏ đói chết hay mất khi bại trận là kết thúc vòng đời mã hiệu đó; sau đó thiên hạ mới có thể xuất hiện lại Cổ cùng loại với mã hiệu mới. Lịch sử đời Cổ xem ở trang cá thể.

**Hiệu ứng Trưởng thành (Growth):** Một số Cổ (Lực đạo/Thổ đạo như Hùng Lực, Hắc Thỉ, Ngọc Bì, Thiết Cốt, Tự Lực...) có hiệu ứng `growth` — khi trang bị ở trạng thái ACTIVE, chỉ số thưởng (Công/Thủ/Máu/Tốc) tăng dần theo thời gian tới mức tối đa (`hours_to_max` giờ). Quy tắc:

- **Chỉ tăng 1 lần cho mỗi loại (GU ID + Chuyển):** dù sở hữu nhiều bản cùng loại, tiến độ và chỉ số chỉ tính 1 lần; không cộng dồn theo số bản sở hữu.
- **Tốc độ trưởng thành nhân `time_speed_multiplier`:** thời gian trưởng thành được rút ngắn theo tốc độ thời gian cá nhân (vd Cổ Tiên tsm 10 → trưởng thành nhanh gấp 10). Không nhân `time_rate` Tiên Khiếu.
- **Tiến độ vĩnh viễn:** tiến độ tích lũy được lưu theo từng instance; gỡ Cổ ra khỏi Không Khiếu vẫn giữ nguyên tiến độ và hiển thị lại ở tab **Trưởng thành** (Không Khiếu). Khi đạt mức tối đa, loại Cổ đó được ghi nhận vĩnh viễn trong danh sách trưởng thành.
- Trang bị lại tiếp tục từ tiến độ đã tích lũy, kể cả khi đổi sang bản khác cùng loại.

## 5. Vật phẩm và nguyên liệu

- Dictionary vật phẩm được Admin import từ `backend/data/Item.csv`.
- Vật phẩm có type, rarity, Chuyển, path, effects, tags và description.
- Item effects hiện có gồm hồi Chân Nguyên/Sinh lực, trị liệu/hồi phục sinh lực Tiên Khiếu (`heal_aperture`), soul boost, đạo ngân, xóa trạng thái xấu và điều kiện chỉ phàm nhân/chỉ Cổ Tiên.

## 6. Luyện Cổ và suy diễn công thức

- Cổ phương được Admin import từ `backend/data/Recipe.csv`.
- Luyện Cổ dùng `/api/refine` với recipe và selected Gu instances.
- Cổ nguyên liệu trong công thức yêu cầu **bằng hoặc cao hơn bậc quy định**. Dùng Cổ nguyên liệu bậc cao hơn sẽ tăng thêm tỉ lệ thành công (Chất lượng nguyên liệu).
- Có thể kéo-thả Cổ bất kỳ vào ô nguyên liệu (kể cả ô "Cổ Trùng Ẩn"). Nếu bộ Cổ không khớp công thức, khi Luyện sẽ **thất bại chắc chắn** — nguyên liệu, tài nguyên vẫn bị tiêu hao.
- Tàn phương (recipe chưa unlock 100%) có thể dùng `/api/refine/deduce` để suy diễn tăng % độ hoàn thiện. Tiêu hao 15 Niệm Lực và Nguyên Thạch mỗi lần.
  - **Độ khó suy diễn:** Tiến độ (Improvement %) mỗi lần bấm sẽ bị chia cho hệ số Rank của Cổ (Rank 5 chia 100, Rank 6 chia 1.000, Rank 9 chia 10.000.000). Phàm nhân gần như không thể cày chay Tiên Cổ Phương.
  - **Chìa khóa (Cảnh Giới & Đạo Ngân):** Bù đắp độ khó khổng lồ này bằng Cảnh Giới Trí Đạo (x3), Luyện Đạo (x2), Hệ đích (x1) và Đạo Ngân Trí Đạo (Tăng tiến độ theo hàm Căn bậc hai). Trí Đạo Vô Thượng Đại Tông Sư có thể suy tính nhanh gấp hàng vạn lần kẻ phàm. Mức Cảnh Giới Trí Đạo cũng gia tăng tỷ lệ Linh quang chớp lóe (Inspiration Flash - Lập tức hoàn thiện 100%).
### Tỉ lệ Luyện Cổ và Hệ quả (MỚI)

Tỉ lệ thành công của mỗi lần Luyện Cổ được tính dựa trên 10 yếu tố:
1. **Tỷ lệ gốc:** Từ bí phương.
2. **Độ hoàn thiện:** Ép luyện khi bí phương chưa 100% sẽ bị giảm tỉ lệ tương ứng.
3. **Độ chính xác tổ hợp:** Lắp sai Cổ nguyên liệu lập tức tỷ lệ về 0%.
4. **Chất lượng nguyên liệu:** Dùng Cổ nguyên liệu bậc cao hơn bậc yêu cầu (+2% mỗi bậc chênh lệch).
5. **Đạo Ngân Luyện Đạo:** Tích lũy từ Tiên Khiếu. Không giới hạn hiệu quả (unlimit), tăng theo hàm căn bậc hai (càng nhiều càng tăng chậm dần), nhân với hệ số cảnh giới lưu phái đích của Cổ đang luyện (`1.0 + Điểm Chân Ý`) và chia cho hệ số scale bậc Cổ tương ứng. Công thức: `(0.632% * √Đạo Ngân * (1.0 + Điểm Chân Ý)) / Hệ số Rank`.
   - Hệ số Rank P: `[R1: 1, R2: 2, R3: 5, R4: 20, R5: 100, R6: 10000, R7: 1000000, R8: 100000000, R9: 10000000000]`
6. **Cảnh giới Luyện Đạo:** Tăng tỉ lệ thành công (+5%, +10%, +18%, +25% tương ứng với cảnh giới từ Đại Sư đến Vô Thượng Đại Tông Sư).
7. **Cảnh giới Lưu Phái đích:** Cảnh giới lưu phái của Cổ đang luyện (tối đa +10% ở Vô Thượng Đại Tông Sư, cụ thể: +2%, +5%, +8%, +10% ở các cấp cảnh giới tương ứng).
8. **Khí vận:** Dựa trên Đạo Ngân Vận Đạo và hiệu ứng Phúc Trạch (tối đa +10%).
9. **Hỗ trợ:** Cổ Trí Tuệ Đạo/Luyện Đạo đang kích hoạt (có dòng `success_rate_bonus`).
10. **Môi trường:** Luyện Cổ tại Đại Vực có thuộc tính phù hợp (+5%).
11. **Tiên Cổ Giảm Trừ (Immortal Penalty):** Khi luyện Tiên Cổ (Rank 6 - 9), toàn bộ các gia thành cộng thẳng ở trên (Mục 4, 6, 7, 8, 9, 10) sẽ bị ép chia nhỏ theo độ khó. Rank 6: chia 2, Rank 7: chia 5, Rank 8: chia 20, Rank 9: chia 100. Việc này khiến ngoại vật trở nên yếu ớt ở cấp độ Tiên Cổ, buộc Cổ Tiên phải dựa hoàn toàn vào Đạo Ngân Luyện Đạo (Mục 5) khổng lồ để chống lại độ khó kinh khủng này.
12. **Tiêu hao Năng Lượng:** Luyện Cổ tiêu tốn sinh lực khiếu. Tiêu hao 20 Chân Nguyên đối với đồng cấp Phàm và 10 Tiên Nguyên đối với đồng cấp Tiên. Số lượng này nhân lên hoặc giảm đi (x10 lần với Phàm, x100 lần với Tiên) cho mỗi bậc chênh lệch. Phàm nhân không thể luyện Tiên Cổ do lập tức cạn kiệt Chân Nguyên.

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
- **[Tâm Thần Mệt Mỏi] (`refine_fatigue`):** Nhận được sau khi Luyện Cổ thỏa điều kiện trên. Giảm **25%** tỉ lệ thành công Luyện Cổ lượt tiếp theo trong **10 phút** (áp dụng trực tiếp sau khi chặn trần 100% các điểm thưởng tích cực).
- **[Luyện Cổ Phản Phệ] (`refine_backfire`):** Xuất hiện khi người chơi **vẫn cố gắng Luyện Cổ** trong lúc đang bị `refine_fatigue` (tự động leo thang) hoặc khi Luyện Cổ bị **Cháy Nổ (Backfire)**. Giảm **50%** tỉ lệ thành công Luyện Cổ lượt tiếp theo trong **20 phút** (áp dụng trực tiếp sau khi chặn trần 100% các điểm thưởng tích cực). Tiếp tục luyện khi đang bị Phản Phệ sẽ gia hạn thêm 20 phút.

## 7. Sát chiêu

- Cổ sư có thể tạo sát chiêu từ Cổ hạch tâm + tối đa 8 Cổ phụ trợ.
- Preview tính hao Chân Nguyên, stability, damage, defense, cooldown/cast time.
- **Hiệu ứng dung hợp (Compound Effects):** khi Cổ core và support có các hiệu ứng tương đồng (VD: damage + bleed, defense + heal), chúng kết hợp thành hiệu ứng mới mạnh hơn thay vì cộng dồn đơn thuần:
  - `damage + bleed → xuất huyết trọng thương`: sát thương + chảy máu mỗi lượt
  - `defense + heal → khiên sinh mệnh`: khiên tự hồi phục
  - `damage + poison → độc nhận phá thể`: vết thương nhiễm độc
  - `damage + stun → chấn động`: sát thương kèm choáng
  - `movement + evasion/stealth → huyễn ảnh bộ`: né tránh + tốc độ
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

- **Một bản đồ thế giới liên tục** 30.000 × 20.000 ô, chia làm **6 Đại Vực (Lục Đại Vực)** theo lưới 3×2, mỗi Đại Vực 10.000 × 10.000. Trung tâm là **Trấn Thanh Bản / Thương Gia Thành** (vùng an toàn, nơi Cổ Sư hồi phục). Mỗi Đại Vực có **HUB riêng** (trấn an toàn khởi đầu): Thương Gia Thành (Nam Cương), Trung Châu Thành, Bắc Nguyên Trấn, Tây Mạc Trấn, Đông Hải Cảng. Khi hoàn tất thiết lập ban đầu, Cổ Sư được đưa vào HUB của Đại Vực đã chọn theo vị trí đăng ký.
- Mỗi Đại Vực chia thành **tiểu vực** 1.000 × 1.000. Trong Đại Vực có các **địa danh có tên** (Thanh Mao Sơn, Tam Vương Truyền Thừa, Sa Mạc Thanh Quỷ...) với độ khó cố định theo vùng.
- Cổ sư xem lưới địa đồ xung quanh, di chuyển theo Thân pháp, chạm trán quái, thu thập tài nguyên/Cổ hoang dã; du hành giữa các Đại Vực qua màn hình bản đồ thế giới.
- **Bản đồ lớn "Ngũ Vực"** hiển thị toàn bộ Lục Đại Vực trên một canvas phóng to/thu nhỏ được: vẽ ranh giới các vùng, các điểm dùng chung (Trạm Nghỉ 🏯, Boss ⚔️, Linh Tài 💎) từ `map_cells`, **các chấm quái vật/tài nguyên xung quanh đạo hữu** (resolve theo đúng nội dung từng ô khi di chuyển), và sương mù tối vùng tiểu vực chưa ghé thăm (dựa trên `visited_subzones`). Các ô đã tải được **cache theo chunk 64×64, tách theo khu vực và prefetch dải phía trước hướng đi**; thay đổi ô realtime được apply trực tiếp thay vì tải lại toàn vùng, nên marker hiện dần (không nhấp nháy) và xuất hiện sớm ở khu vực sắp đến. Click vào bất kỳ vị trí nào trên bản đồ để **di chuyển tự do** — player đi bộ mượt theo đường thẳng qua từng ô ngay khi nhận thao tác, trong khi vị trí vẫn được server xác nhận theo từng chặng ngắn; nếu gặp chiến đấu hoặc sự kiện giữa đường, nhân vật tự điều chỉnh về ô server đã xác nhận. Mỗi ô đi qua đều kích hoạt nội dung; **chỉ chạm trán quái khi bước vào ô thật sự có quái/boss** (bước vào ô trống thì đi bình thường, không có phục kích ngẫu nhiên). **Có thể đổi hướng/dừng giữa chừng** bằng cách click vị trí khác (hoặc click lại ô đang đứng để dừng); các thao tác liên tiếp được gom về đích mới nhất để không gửi nhiều lệnh di chuyển chồng nhau. Vị trí camera + độ zoom của bản đồ được **giữ nguyên sau khi chiến đấu**.
- **Quái nhỏ, tài nguyên thấp, Cổ cấp thấp** được sinh procedural theo vùng (không lưu DB), **nội dung ổn định theo ô** — quái không tự đổi chỗ/re-randomize theo thời gian; sau khi đạo hữu dọn dẹp một ô, ô đó trống trong 15 phút rồi nội dung cũ quay lại (cooldown `cleared_cells`). **Boss và tài nguyên cấp cao** là điểm dùng chung toàn thế giới trong `map_cells`, có thời gian hồi sinh sau khi bị đánh bại/hái.
- Nội dung mỗi vùng phụ thuộc dữ liệu do Admin import từ `backend/data/Region.csv`, `MonsterSpawn.csv` và `MapCell.csv`.
- Thanh công cụ cho phép chuyển giữa **click để đi** và **kéo bản đồ**, đưa camera về nhân vật, đồng thời bật/tắt nhãn Quái & Cổ/Boss/HUB/Khu vực/Ngũ Vực. Chọn một người chơi để xem hồ sơ, mời tổ đội hoặc tiếp cận; chỉ có thể phát động PvP khi mục tiêu ở trong phạm vi 2 ô.
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
- **Giao trân "đánh trả từng phát":** các Cổ trong lượt tấn công được kích hoạt **lần lượt từng chiêu**, không còn xả trọn bộ trong một nhịp. Sau mỗi đòn đánh ra, mục tiêu chủ (đối thủ chính) sẽ **đáp trả ngay một đòn** vào đạo hữu. Tốc độ quyết định **ai giao trân trước** (tính cả hiệu ứng Chậm) — bên nhanh hơn mở màn chuỗi đánh–đáp, sau đó hai bên luân phiên từng phát. Dùng vật phẩm/hút Nguyên Thạch/tụ lực không bị đáp trả; Bộc Phá Sát Chiêu và đánh thường vẫn bị đáp trả bình thường. Đối thủ ngoài đáp trả vẫn giữ đúng một hành động AI mỗi lượt như cũ.
- **Thỉnh Đạo Hữu Tiên Hành Cổ (tiên thủ lượt 1):** trước khi trận bắt đầu, nếu thọ nguyên đối thủ chính không hơn gấp 3 lần thọ nguyên của đạo hữu, đạo hữu được **ra tay đầu tiên** ở lượt 1 (địch vẫn hành động ngay sau, theo tốc độ thường — không bị bỏ lượt). Quái vật dùng tuổi thọ quy đổi theo Chuyển; Cổ sư đối địch (PvP) dùng thọ nguyên thật.
- **Auto battle (tự chiến/luyện chiến) cùng cơ chế:** mỗi đòn Cổ gây sát thương đều bị nạn nhân **đáp trả ngay bằng viên Cổ kế tiếp** trong bộ của họ (trừ mana, chịu khống chế như thường); cast hỗ trợ/hồi/khiên không bị đáp trả; đòn đáp trả không bị đáp trả ngược (không dây chuyền); AoE chỉ được đáp trả 1 lần bởi mục tiêu trúng đòn chính. Hành động đã dùng để đáp trả sẽ không còn khi tới lượt speed của nhân vật đó.
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
  - Các hiệu ứng đặc biệt như **Bất Tử (INVINCIBLE)** hay **Khóa Cổ (GU_SEAL)** sẽ bị nhân tỉ lệ kích hoạt với $K_{\text{caster}}$ (ví dụ: Tiên nhân Bát Chuyển/Cửu Chuyển dùng Cổ Bất Tử Tứ Chuyển thì tỉ lệ kích hoạt rơi về $\approx 0.002\%$).
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
- **Công thức bắt:**
  ```
  baseChance × typeFactor × hpFactor × (1 + captureBonus) × luckMultiplier
  ```
  - `baseChance`: 0.25 (Nhất Chuyển) → 0.10 (Ngũ Chuyển) — giảm dần theo Chuyển
  - `typeFactor`: 1.0 (insect) / 0.8 (reptile) / 0.6 (beast) / 0.4 (spirit) / 0.2 (divine)
  - `hpFactor`: `(1 - enemy.hp%)^0.5` — quái càng yếu càng dễ bắt (full HP ≈ 0%)
  - `captureBonus`: tổng từ thuộc tính `capture_bonus` của các Cổ đang trang bị
  - `luckMultiplier`: random 0.9-1.1
- **Kết quả:** Nếu roll < baseChance → bắt thành công, quái biến thành sủng vật trong `player_pets`, xóa khỏi combat. Nếu roll ≥ baseChance → bỏ lỡ, tiếp tục chiến đấu.

### Sủng vật (Pets)

Sủng vật là quái vật đã bắt, hỗ trợ chiến đấu:

- **Đội hình:** Tối đa 4 sủng vật trong đội hình (tăng theo Chuyển, Nhất Chuyển = 2 slot, Ngũ Chuyển = 4 slot).
- **Chiến đấu:** Sủng vật tự động tấn công quái địch mỗi lượt bằng BASIC_ATTACK (ATK × 1.0).
- **Phí Hồn Phách:** mỗi lần ra trận thật, một bầy sủng vật tiêu tốn `soul_hp` = số cá thể trong bầy; nếu Hồn Phách không đủ thì sủng vật đó bị bỏ qua, không xuất trận. Võ Đài (ảo) không tốn Hồn Phách.
- **Độ no:** Giảm theo `pet_hunger_per_hour` (mặc định 2/giờ). Sủng vật đang ngủ giảm 50%. Khi về 0 → **chết vĩnh viễn** (xóa khỏi DB).
- **Cho ăn:** Dùng vật phẩm có effect `hunger` (Thịt Lợn Rừng +40, Rượu Thanh Trúc +30, Ngọc Thạch +25, Nguyệt Lan +20).
- **Gộp:** Cùng `monster_template_id` có thể gộp thành 1 bầy. Chỉ số × `N^0.65` (theo base, không cộng dồn lũy tiến). Bầy tối đa `max_soul/10` cá thể — cần tăng Hồn Phách để gộp thêm.
- **Chết:** Sinh lực = 0 trong combat hoặc hết độ no → xóa vĩnh viễn. Không thể hồi sinh.

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

## 13. Môn phái

Hệ thống môn phái hiện có:

- Xem/tạo/gia nhập/rời môn phái.
- Duyệt đơn gia nhập nếu môn phái bật yêu cầu duyệt.
- Thành viên, ban bậc, trục xuất.
- Xin/thẩm định thăng chức.
- **Cấp bậc:** Ngoại Môn → Nội Môn → Tinh Anh → Chân Truyền → Trưởng Lão → **Hộ Pháp** → Chưởng Môn. Chi phí thăng cấp bằng Điểm cống hiến: 100 / 500 / 2.000 / 10.000 / 30.000.
- **Duyệt lên cấp:** Trưởng Lão/Chưởng Môn có thể bật tùy chọn "Duyệt Lên Cấp" (mặc định tắt). Khi bật, **mọi** thăng cấp của đệ tử đều tạo yêu cầu chờ duyệt thay vì tự động. Khi tắt, đệ tử tự thăng cấp theo tiêu chuẩn (trừ bậc Chân Truyền/Trưởng Lão/Hộ Pháp vẫn cần duyệt).
- Đóng góp nguyên thạch/vật liệu/Cổ/**Bí phương**.
- Bảo khố môn phái, mua vật phẩm từ kho, stock kho (hỗ trợ cả Bí phương).
- Nhiệm vụ mặc định và nhiệm vụ custom:
  - **Hạn tự xóa:** Nhiệm vụ trên bảng có thời hạn (mặc định 24h, Trưởng Lão/Chưởng Môn tùy chỉnh khi phát hành; `-1` = không hết hạn). Hết hạn nhiệm vụ tự biến mất khỏi bảng, nhiệm vụ của đệ tử đang làm cũng bị hủy theo.
  - **Hủy nhiệm vụ:** Đệ tử có thể hủy bỏ nhiệm vụ đang thực hiện để nhận nhiệm vụ khác. Trưởng Lão/Chưởng Môn có thể xóa nhiệm vụ đã phát hành khỏi bảng.
- Nâng cấp môn phái: chi phí `Cấp² × 5.000` Quỹ.
- Mỏ môn phái: xem, tấn công, nhận thưởng.
- **Lãnh địa (mỏ tông môn):**
  - **Phòng thủ:** Trưởng Lão/Hộ Pháp/Chưởng Môn bố trí đệ tử hoặc thuê thần thú canh giữ (phí `hạng × 500` Quỹ). Mỗi đệ tử chỉ giữ 1 mỏ. Slot theo rarity: Thường 1 · Hiếm 2 · Sử Thi 3 · Truyền Thuyết 3 · Siêu Cấp 4.
  - **Sinh tài nguyên theo giờ:** mỏ đang chiếm giữ tự sinh **Quỹ** và **Linh tài** vào bảo khố, tỉ lệ theo rarity × cấp TM × số phòng thủ.
  - **Điểm cống hiến theo giờ:** đệ tử phòng thủ +20 pts/h, các đệ tử khác +2 pts/h, cap 200 pts/ngày (chống AFK).
  - **Công kích có chiến đấu:** tấn công mỏ đang bị chiếm sẽ so sánh sức mạnh của bạn vs tổng phòng thủ (được củng cố bởi người giữ). Thắng → chiếm + bảo hộ 4h; thua → trận pháp hộ sơn 15 phút và người phòng thủ được thưởng +20 cống hiến.
  - **Chỉ lãnh đạo phát động tấn công:** Trưởng Lão/Hộ Pháp/Chưởng Môn chọn đệ tử ra trận (Ngoại Môn không tham gia); sức mạnh của người được chọn sẽ được so sánh với phòng thủ, và Nhân Khí khi thắng thuộc về đệ tử ra trận đó.

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
9. `BEFORE_HIT / FIRST_TURN_EVADE` — né đòn đánh đầu tiên ở Lượt 1, đòn multi-hit/AoE tiếp theo vẫn trúng (chỉ Cổ nào mang mechanic này mới có; Định Tiên Du hiện tại mang `portal_create` + `movement`, không có evade).
9b. `apply_status / INVINCIBLE` có `chance` — miễn nhiễm theo tỉ lệ mỗi lần dùng, tỉ lệ và số lượt tăng theo rank (`chance + 0.05×(rank−1)`, `turns + (rank−1)/3`). VD: Hoang Mộ Cổ rank 4 (`chance: 0.35, turns: 1`) → 50%/2 lượt + khiên 600. Trượt hiện log `STATUS_MISS`.
10. `COMBAT_PASSIVE / DOUBLE_DAMAGE_DEALT` (`PASSIVE` cũ vẫn dùng được) — nhân đôi mọi sát thương gây ra. VD: Tiên Thai Chí Tôn (kèm `path_conflict_reduction: 100`).
11. `active.cleansing_power` / `passive.tenacity` + `passive.cc_resist` — kháng và tự giải hiệu ứng bất lợi (xem mục Kháng hiệu ứng và Tenacity ở mục 9). VD: Ngã Ý Cổ (`tenacity: 0.35`, `cc_resist: 0.15`).
12. `active.damage_hp_percent` + `shield_flat_reduce_percent` + `disable_gu_next_turn` — trừ thẳng % MaxHP (xuyên khiên), tước % khiên hiện tại, phong ấn Cổ mục tiêu lượt sau (`GU_SEAL`, Tenacity có thể tự giải). VD: Lãng Quên Bản Ngã Cổ (40%/40%/1 lượt + status `FORGET_EGO`).
13. `active.forget_reality_chance` + `self_evasion_boost_percent` + `target_damage_reduce_percent` + status `FORGET_REALITY` — 40% proc: bản thân +50% né tránh, địch dính suy yếu 40% sát thương (2 lượt). VD: Lãng Quên Thực Tại Cổ.
14. `passive.anti_heaven/earth/dream_damage_reduction_percent` (+ `immune_dream_status`) — giảm sát thương khi đòn đánh mang Gu path tương ứng (nhận diện theo `Gu.path`); miễn nhiễm mọi status Mộng đạo (`SLEEP`, `CONFUSE`...). VD: Lãng Quên Thiên/Địa/Mộng Cổ (65%/65%/50%).
15. `BEFORE_ENEMY_ATTACK / FORGET_INTENT_INTERRUPT` — khi địch tấn công, 40% phong ấn Cổ địch lượt sau (tối đa 8 lần thành công, sau đó phản phệ phong ấn chính mình). VD: Lãng Quên Ý Niệm Cổ.
16. `BEFORE_ENEMY_ATTACK / FORGET_WILL_CONFUSE` — khi địch tấn công, 35% quên mục tiêu (15% trúng chủ Cổ, 35% tự đánh mình, 50% hụt) + 20% phản phệ phong ấn Cổ mình 2 lượt. VD: Lãng Quên Chí Cổ.
17. `active.damage_path` + `stat_base_attack` trong active — ghi đè hệ đòn đánh + cộng flat vào sát thương (VD Trầm Luân Tiên Cổ: 250M + `SLEEP` + hệ Dream). Lưu ý: `status_duration_ticks` ≤ 5 đi qua pipeline tier legacy.

### Cơ chế ngoài chiến đấu (Handlers)
1. `ON_CONSUME / ADD_LIFESPAN_YEARS (params.value)` — tiêu hao để cộng vĩnh viễn N năm tuổi thọ. VD: Thọ Cổ 100 Năm (`value: 100`).
2. `ON_CONSUME / SACRIFICE_NPC + CAPABILITY / SACRIFICE_APTITUDE` — hiến tế để tăng vĩnh viễn tư chất Không khiếu (mỗi lần +5, trần 99). VD: Huyết Sọ Cổ.
3. `CAPABILITY / LOOT_PREVIEW` (+ `active.range`) — hiển thị tỉ lệ rớt đồ của quái trên bản đồ và mở rộng tầm khám phá. VD: Cổ Thiên Cơ (`range: 1000`).
4. `ON_TICK / THOUGHT_REGEN_TICK (params.value)` — hồi Niệm lực mỗi giờ cho Cổ sư trang bị. VD: Trí Tuệ Cổ.

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
- **Cảnh giới lưu phái:** Phổ thông ➔ Chuẩn đại sư ➔ Đại sư ➔ Tông sư ➔ Đại tông sư ➔ Vô thượng đại tông sư ➔ Đạo chủ. (Đạo chủ là cảnh giới chỉ có Tôn Giả mới đạt tới, có thể thao túng đạo ngân trong thiên nhiên).

## 19. Hướng Dẫn Tân Thủ (Wiki Guide)

Hệ thống cung cấp wiki trong game tại `/wiki`; phần dưới là lộ trình tân thủ ngắn gọn:

### Thuật ngữ cốt lõi:
- **Không Khiếu & Tiên Khiếu:** Phàm nhân sử dụng Không Khiếu tích lũy Chân Nguyên chiến đấu. Cổ Tiên (Lục Chuyển trở lên) sở hữu **Tiên Khiếu** (thế giới nhỏ độc lập bên trong cơ thể) tự sản sinh Tiên Nguyên với 4 phẩm cấp: Thấp, Trung, Cao, Cực Hạn dựa trên Nhân Khí lúc Thăng Tiên.
- **Đạo Ngân (Dao Marks):** Vết khắc của quy luật tự nhiên, chia thành 49 lưu phái (Lực đạo, Hỏa đạo...). Mỗi điểm Đạo Ngân tăng **+1%** sát thương/phòng ngự cho các kỹ năng/Cổ trùng thuộc lưu phái đó.
- **Cảnh Giới Lưu Phái (Attainment):** Độ ngộ Chân Ý gồm 5 bậc: Phàm Thai ➔ Đại Sư ➔ Tông Sư ➔ Đại Tông Sư ➔ Vô Thượng Đại Tông Sư. Điểm tích lũy theo từng lưu phái (thi triển sát chiêu thành công, luyện Cổ thành công, vượt ải phó bản). Cảnh giới suy ra từ mốc điểm: Đại Sư=100, Tông Sư=1.000, Đại Tông Sư=10.000, Vô Thượng=100.000. **Để đột phá lên Rank 7/8/9 khi vượt Kiếp Nạn, cần ít nhất 1 lưu phái đạt Tông Sư/Đại Tông Sư/Vô Thượng tương ứng.** Cảnh giới càng cao càng tăng tỉ lệ thành công của sát chiêu và luyện Cổ.
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
- **Thành tựu:** nhận thưởng cho thành tựu đã mở khóa. Danh hiệu/khung đã mở có thể được chọn để hiển thị trên hồ sơ.
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

- **Lịch Sử Chiến Đấu** liệt kê trận gần đây, kết quả/đối thủ và cho mở báo cáo hoặc replay log chi tiết.
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

### Ba trụ điểm (áp cho cả hai phe)
1. **Cổ trùng:** phe công lấy chỉ số Trinh Sát (Investigation) của Cổ đang trang bị; phe thủ lấy Ngụy Trang (DisguiseLevel) + năng lực đặc biệt (`PERFECT_DISGUISE` như Thái Độ Cổ, `CONCEAL_GU`).
2. **Cảnh giới lưu phái + Đạo ngân:**
   - **Trí đạo:** lực suy tính chính của phe công; đồng thời là lá chắn đối kháng trực tiếp của phe thủ.
   - **Ám đạo (thủ):** che bản thân, vị trí, khí tức; tăng rào cản khoảng cách.
   - **Tín đạo:** phe công khuếch đại giá trị manh mối; phe thủ phong tỏa, bóp méo manh mối (Tông Sư Tín đạo trừ tới 36%, tối đa một nửa).
   - **Vận đạo:** gieo xúc xắc vận số mỗi lần suy diễn — phe công tu Vận cao thì kết quả ổn định, phe thủ Vận cao thì kết quả lệch xấu cho kẻ dò.
   - **Thiên đạo (Tiên nhân):** che / bóp méo thiên cơ (màn Thiên Cơ), chỉ Trí/Thiên phe công mới xuyên được.
   - **Biến hóa đạo (thủ):** đổi dấu vết, thân phận, đạo ngân biểu hiện — khiến kết quả trả về là hàng giả.
3. **Manh mối (clue):** tên đúng +20, đúng vùng +15, đúng tông môn +10, Cổ nghi vấn đúng +10 (đều phải **đúng thật** mới tính), khí tức Cổ mục tiêu +30, dò mù không tên −40. Manh mối bị chặn trần, không thể thay tu vi.

### Chi phí suy diễn

- Mỗi lần suy diễn tốn **Nguyên Thạch** (tối thiểu 100, càng nhiều điểm cộng càng cao) hoặc **Tiên Nguyên Thạch** (chỉ Tiên nhân). Không tốn Chân Nguyên.

### Bốn kết quả (chỉ so **điểm tổng cuối**)
| Chênh lệch | Kết quả |
| --- | --- |
| ≥ +30 (người) / +20 (Cổ) | **Đại thành công:** tọa độ thật, tu vi thật, toàn bộ Cổ trang bị; được gieo ấn ký |
| ≥ 0 / ≥ −10 (Cổ) | **Tiểu thành công:** vùng + tu vi ước lượng + 2 Cổ (tọa độ bị giấu) |
| ≥ −30 / ≥ −40 (Cổ) | **Màn sương:** mất tài nguyên, trắng tay |
| Thấp hơn | **Thiên cơ phản phệ:** trọng thương HP + Hồn, trừ 20 Chân Nguyên (cố định, không theo %), **lộ ngược tọa độ cho mục tiêu** |

Sau mỗi lần suy diễn, bảng **điểm của chính đạo hữu** (Cổ / cảnh giới / manh mối / vận / tổng) hiện đầy đủ để biết mình thiếu ở đâu — điểm phòng ngự của địch không bao giờ lộ.

### Suy tính Cổ trùng (ai giữ, ở đâu, luyện sao)
- Phàm Cổ dễ dò; điểm suy tính bị **chia theo bậc Cổ** (bậc 1-5 chia 1/2/3/4/5, Tiên Cổ bậc 6-9 chia 10/18/30/50) — nên Cổ càng cao càng vượt tầm, cá voi cũng phải dốc sức mới dò được Tiên Cổ. Phàm nhân dò Tiên Cổ gần như chắc chắn phản phệ.
- **Cổ trong tay người mang khí tức của chủ:** Cổ trang bị được chủ che full; Cổ nằm kho/ngủ được che một nửa. Muốn dò Cổ của Cổ Tiên ẩn danh, trước hết phải thắng được chính hắn.
- Cổ trong bảo khố tông môn khó theo cấp tông môn; Cổ treo chợ công khai dễ dò nhất.
- Tự sở hữu Cổ đó thì suy tính cộng hưởng trực tiếp, gần như luôn thấu triệt.

### Ấn ký truy tung & chém đứt manh mối
- Đại thành công suy diễn người cho phép **gieo ấn ký**: Trí Niệm Ấn (ngắn) hoặc Huyết Mạch Ấn (lâu). Giới hạn 3 mục tiêu (phàm) / 10 (tiên). Ấn ký cho tọa độ realtime trong radar.
- **Đoạn Tuyệt Thiên Cơ:** tốn 20 Chân Nguyên (cố định), chém toàn bộ ấn ký người khác gieo lên mình + xóa tàn dư khí tức tại ô đang đứng. Bị theo dõi mà không biết chém — chết lúc nào không hay.
