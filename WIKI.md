# GuWorld Gameplay Wiki

Wiki này phản ánh các hệ thống đã có trong codebase hiện tại, diễn tả bằng khẩu khí Cổ Chân Nhân.

## 1. Tài khoản và nhân vật

- Cổ sư đăng ký bằng username/email/password hoặc đăng nhập Google nếu cấu hình OAuth.
- Hồ sơ hỗ trợ bio, avatar, đổi mật khẩu và cơ chế ngụy trang thông tin công khai.
- Referral có thống kê người được mời và phần thưởng/mốc xử lý ở backend.

### Chủng tộc, Giới tính, Thể chất, Trạng thái

Khi đăng ký, đạo hữu chọn **Chủng tộc** (mặc định Nhân Tộc) và **Giới tính**. Trong quá trình tu hành, **Thể chất** và **Trạng thái** có thể thay đổi qua sự kiện/quà hoặc Admin. Mỗi giá trị mang hiệu ứng riêng:

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

Danh sách chủng tộc được **data-driven** từ bảng `race_dictionary` (seed từ `data/Race.csv`): admin thêm chủng tộc mới bằng cách sửa CSV (thêm dòng `key,name,effects`) rồi Import bảng "Chủng tộc" trong Admin panel tab **Dữ liệu**. Effect format xem `docs/GAME_DATA.md`.

**Giới tính:** Nam (`male`) / Nữ (`female`) — chưa có hiệu ứng (chỉ hiển thị).

**Trạng thái:**

| Trạng thái | Hiệu ứng |
| --- | --- |
| Trọng Thương (`severely_wounded`) | **-20%** tấn công, phòng thủ, tốc độ |

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

Trạng thái có thể tồn tại đồng thời nhiều cái (mảng `statuses`). Danh sách thể chất được **data-driven** từ bảng `constitution_dictionary` (seed từ `data/Constitution.csv`): admin thêm thể chất mới bằng cách sửa CSV (thêm dòng `key,name,effects`) rồi Import bảng "Thể chất" trong Admin panel tab **Dữ liệu**. Effect format xem `docs/GAME_DATA.md`.


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
  - **Giới hạn mỗi trận:** điểm Chân Ý nhận từ chiến đấu bị chặn ở **Rank × 10 điểm/trận** (đánh tay, đánh tự động và sát chiêu đều tính chung). Đạt giới hạn thì không ngộ đạo thêm cho tới khi trận chiến kết thúc. Thưởng phó bản và luyện Cổ không tính vào giới hạn này.
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

### Thăng Tiên và Độ Kiếp

- Ngũ Chuyển Đỉnh phong (Chuyển 5, SubRank 4) có thể thăng tiên khi Nhân Khí đạt mức mong muốn.
- **Tỉ lệ thành công tuyến tính theo Nhân Khí:** `chance = NhânKhí / 100.000`. 0 Nhân Khí → 0%, 40.000 → 40%, 100.000 → chắc chắn thành công.
- **Thành công:** lên Lục Chuyển Cổ Tiên, nhận 10 Tiên Nguyên Thạch, Đạo Ngân Thiên/Địa = 100. Kiếp đầu tiên giáng lâm vào **năm nội bộ thứ 10**, tương đương khoảng **2–10 giờ đời thật** tùy phẩm chất Tiên Khiếu (xem mục Gia tốc thời gian). **Tiên Khiếu được khai mở ngay**, phẩm cấp được định đoạt theo Nhân Khí tại thời điểm thăng tiên (xem mục Tiên khiếu).
- **Thất bại:** Nhân Khí về 0, tu vi phế bỏ rơi về Nhất Chuyển Sơ giai, mất hết tiến độ phá vách.
- Cổ Tiên có tiên khiếu, đạo ngân, tiên nguyên/tài nguyên riêng.
- Tai kiếp có worker kiểm tra tự động mỗi 60 giây. Khi năm nội bộ đạt mốc tiếp theo, worker sinh kiếp và resolve.
- **Quái tai kiếp không phá hủy trực tiếp mỏ tài nguyên.** Khi tiếp cận linh mạch tài nguyên, mỗi yêu tướng mỗi nhát làm giảm `hp` mỏ theo loại kiếp: Địa Tai -1, Thiên Kiếp -2, Hạo Kiếp -4, Vạn Kiếp -8, Hỗn Độn Đại Nạn -16. Đại Yêu di chuyển quanh quẩn và đánh mỏ với sát thương gấp đôi yêu tướng.
- **Combat tai kiếp hỗ trợ buff từ Cổ đang trang bị:** hiệu ứng `damage` (sát thương), `damage_multiplier` (nhân sát thương), `defense`/`shield_flat` (tạo Khiên hộ thể hấp thụ phản chấn của quái lên cơ thể hoặc hấp thụ trực tiếp sát thương phản chấn lên Tiên Khiếu đối với Cổ Tiên), `heal`/`heal_flat` (hồi Sinh lực, tối đa bằng HP tối đa hiệu dụng = `base_hp` + passive `stat_base_hp` từ Cổ).
- **Cổ sư chỉ được công kích khi Sinh lực Tiên Khiếu > 20% Sinh lực tối đa.** Nếu Tiên Khiếu nguy cấp (≤ 20%), đạo hữu không thể ra tay — quái sẽ tiếp tục càn quét mỏ trong sự bất lực; hãy chờ Tiên Khiếu hồi phục hoặc kết toán vượt kiếp.
- **Kết toán tai kiếp:** Nếu kết toán khi còn quái trên bản đồ lưới (chấp nhận rủi ro hoặc hết hạn 6 giờ thực tế tự động kết toán), đạo hữu có nguy cơ thất bại. Khi vượt kiếp thất bại, bộ đếm số lượng tai kiếp (tổng số tai kiếp và số lượng theo từng loại) sẽ không tăng (chỉ tích lũy khi thành công), nhưng thời gian/mốc năm của tai kiếp kế tiếp (`next_tribulation_year`) vẫn sẽ được đẩy lên mốc tiếp theo để người chơi có thời gian hồi phục và không bị dồn dập tai kiếp ngay lập tức.
- **Kiếp nạn theo cấp bậc:**
  - Lục Chuyển: Địa Tai (10 năm), Thiên Kiếp (100 năm)
  - Thất Chuyển: Địa Tai (10 năm), Thiên Kiếp (50 năm), Hạo Kiếp (100 năm)
  - Bát Chuyển: Thiên Kiếp (10 năm), Hạo Kiếp (50 năm), Vạn Kiếp (100 năm)
  - Cửu Chuyển: Hạo Kiếp (10 năm), Vạn Kiếp (50 năm), Hỗn Độn Đại Nạn (100 năm)
- **Tự động đột phá:** Khi đủ 3 lần vượt kiếp cấp cao nhất (vd: 3 Thiên Kiếp), kết hợp điều kiện số năm ở Chuyển (300 năm với 6→7, 7→8), Cổ Tiên tự động lên Chuyển kế tiếp.
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

- Nhân Khí là tài nguyên phản ánh hoạt động tích cực của đạo hữu trong giới tu tiên, giới hạn tối đa **100.000**.
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

### Đạo ngân

- Đạo ngân lưu theo path và amount.
- Tai kiếp thành công thưởng đạo ngân.

## 4. Cổ trùng và bọc đồ

Mỗi Cổ instance có:

- `instance_id`, `gu_id`.
- Độ No (giảm theo `HungerRate * TimeSpeedMultiplier` mỗi 3 giờ thực tế, Cổ trong Tiên Khiếu có time rate cao sẽ đói nhanh hơn).
- Status: `ACTIVE`, `SLEEPING`, `DEAD`.
- `equipped` và `is_vital`.

Cổ sư có thể:

- Cho ăn từng Cổ hoặc theo nhóm.
- Trang bị/tháo Cổ.
- Đặt Cổ bản mệnh.
- Chuyển trạng thái active/sleeping.
- Dùng Cổ tiêu hao hoặc Cổ hỗ trợ.
- Vứt bỏ Cổ.

**Tiên Cổ duy nhất:** Cổ từ **Lục Chuyển trở lên (Tiên Cổ)** là độc nhất toàn thiên hạ — mỗi `gu_id` chỉ tồn tại một instance trên toàn server. Các kênh nhận Cổ (Mở đá, Rương Mỏ Linh Thạch, Luyện Cổ, quái rơi đồ) đều loại trừ Tiên Cổ đã tồn tại (trong túi bất kỳ đạo hữu, đấu giá) trước khi roll; nếu hết ứng viên hợp lệ sẽ rơi vật phẩm/thạch bù trừ.

**Hiệu ứng Trưởng thành (Growth):** Một số Cổ (Lực đạo/Thổ đạo như Hùng Lực, Hắc Thỉ, Ngọc Bì, Thiết Cốt, Tự Lực...) có hiệu ứng `growth` — khi trang bị ở trạng thái ACTIVE, chỉ số thưởng (Công/Thủ/Máu/Tốc) tăng dần theo thời gian tới mức tối đa (`hours_to_max` giờ). Quy tắc:

- **Chỉ tăng 1 lần cho mỗi loại (GU ID + Chuyển):** dù sở hữu nhiều bản cùng loại, tiến độ và chỉ số chỉ tính 1 lần; không cộng dồn theo số bản sở hữu.
- **Tốc độ trưởng thành nhân `time_speed_multiplier`:** thời gian trưởng thành được rút ngắn theo tốc độ thời gian cá nhân (vd Cổ Tiên tsm 10 → trưởng thành nhanh gấp 10). Không nhân `time_rate` Tiên Khiếu.
- **Tiến độ vĩnh viễn:** tiến độ tích lũy được lưu theo từng instance; gỡ Cổ ra khỏi Không Khiếu vẫn giữ nguyên tiến độ và hiển thị lại ở tab **Trưởng thành** (Không Khiếu). Khi đạt mức tối đa, loại Cổ đó được ghi nhận vĩnh viễn trong danh sách trưởng thành.
- Trang bị lại tiếp tục từ tiến độ đã tích lũy, kể cả khi đổi sang bản khác cùng loại.

## 5. Vật phẩm và nguyên liệu

- Dictionary vật phẩm nằm trong `backend/Item.csv`.
- Vật phẩm có type, rarity, Chuyển, path, effects, tags và description.
- Item effects hiện có gồm hồi Chân Nguyên/Sinh lực, trị liệu/hồi phục sinh lực Tiên Khiếu (`heal_aperture`), soul boost, đạo ngân, xóa trạng thái xấu và điều kiện chỉ phàm nhân/chỉ Cổ Tiên.

## 6. Luyện Cổ và suy diễn công thức

- Cổ phương lấy từ `backend/Recipe.csv`.
- Luyện Cổ dùng `/api/refine` với recipe và selected Gu instances.
- Cổ nguyên liệu trong công thức yêu cầu **đúng bậc** (`required_gus` lưu cả `rank` lẫn `qty`); Cổ khác bậc sẽ không được tính vào công thức.
- Có thể kéo-thả Cổ bất kỳ vào ô nguyên liệu (kể cả ô "Cổ Trùng Ẩn"). Nếu bộ Cổ không khớp công thức, khi Ép Luyện sẽ **thất bại chắc chắn** — nguyên liệu, tài nguyên vẫn bị tiêu hao và (nếu là Ép Luyện) chịu phản phệ như thất bại thường.
- Suy diễn recipe dùng `/api/refine/deduce`, cập nhật tiến độ unlock recipe trong player.

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
- **Bản đồ lớn "Ngũ Vực"** hiển thị toàn bộ Lục Đại Vực trên một canvas phóng to/thu nhỏ được: vẽ ranh giới các vùng, các điểm dùng chung (Trạm Nghỉ 🏯, Boss ⚔️, Linh Tài 💎) từ `map_cells`, **các chấm quái vật/tài nguyên xung quanh đạo hữu** (resolve theo đúng nội dung từng ô khi di chuyển), và sương mù tối vùng tiểu vực chưa ghé thăm (dựa trên `visited_subzones`). Các ô đã tải được **cache phía client và prefetch dải phía trước hướng đi**, nên marker hiện dần (không nhấp nháy) và xuất hiện sớm ở khu vực sắp đến. Click vào bất kỳ vị trí nào trên bản đồ để **di chuyển tự do** — player đi bộ theo đường thẳng qua từng ô, mỗi ô đi qua đều kích hoạt nội dung; **chỉ chạm trán quái khi bước vào ô thật sự có quái/boss** (bước vào ô trống thì đi bình thường, không có phục kích ngẫu nhiên). **Có thể đổi hướng/dừng giữa chừng** bằng cách click vị trí khác (hoặc click lại ô đang đứng để dừng), vị trí server cập nhật mịn từng ~150 ô nên di chuyển của người chơi khác hiển thị gần như real-time. Vị trí camera + độ zoom của bản đồ được **giữ nguyên sau khi chiến đấu**.
- **Quái nhỏ, tài nguyên thấp, Cổ cấp thấp** được sinh procedural theo vùng (không lưu DB), **nội dung ổn định theo ô** — quái không tự đổi chỗ/re-randomize theo thời gian; sau khi đạo hữu dọn dẹp một ô, ô đó trống trong 15 phút rồi nội dung cũ quay lại (cooldown `cleared_cells`). **Boss và tài nguyên cấp cao** là điểm dùng chung toàn thế giới trong `map_cells`, có thời gian hồi sinh sau khi bị đánh bại/hái.
- Nội dung mỗi vùng phụ thuộc `backend/Region.csv` (geometry + pool), `backend/MonsterSpawn.csv` (pool quái) và `backend/MapCell.csv` (boss/tài nguyên cao/điểm an toàn).

## 9. Chiến đấu

- Combat PvE khởi tạo từ thám hiểm, mỏ hoặc các hệ thống gọi combat.
- Cổ sư dùng Cổ trang bị, có thể charge và chọn mục tiêu.
- Chỉ định mục tiêu hỗ trợ vị trí/hàng/cột/toàn bộ tùy `target_type` của Cổ.
- Combat state hỗ trợ đội hình địch/ta, status effects, cooldown, Chân Nguyên/Sinh lực, chiến lợi phẩm.
- Thắng trận áp dụng chiến lợi phẩm vào thẳng túi đồ thật; thua trận cập nhật trạng thái theo handler combat.
- **Túi đồ khi khám phá là túi thật:** nguyên liệu/Cổ/tiền nhặt được trong chuyến đi vào thẳng túi đồ (không còn "túi tạm"). Khi về trấn an toàn không có bước "thu hoạch" — đồ đã nằm sẵn trong túi.
- **Chết thì rơi đồ (hình phạt, tùy chỉnh trong `handlers/death_penalty.go`):** khi bại trận dã ngoại, Bỏ chạy ngất xỉu, hay thua PvP, đạo hữu rơi mất một phần tài nguyên đang giữ theo cấu hình: % Nguyên Thạch / Tiên Nguyên Thạch, một phần nguyên liệu, một số Cổ (luôn trừ Cổ đang trang bị & Bản Mệnh Cổ) và một số Bí phương. Chết vì quái (PvE) thì đồ rơi vào hư không; thua PvP thì kẻ thắng cướp đoạt số đồ rơi đó.
- **Bỏ chạy (Rút Lui) khỏi chiến đấu** khiến đạo hữu chịu **−20% Max HP**; nếu HP không đủ sẽ ngất xỉu tại HUB và chịu hình phạt rơi đồ như trên. Khi HP ≤ 20% Max, hệ thống hiện cảnh báo xác nhận trước khi bỏ chạy.
- Khi hệ thống quá tải (tranh chấp khoá dữ liệu do nhiều thao tác cùng lúc), các thao tác chiến đấu (bỏ chạy/rút lui/bắt sủng…) trả về thông báo **bận và yêu cầu thử lại** thay vì treo vô hạn.

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

## 10. Phó bản

- Instance có template, session active và action theo option index.
- Cổ sư vào instance, chọn hành động và nhận kết quả theo stage/map data.
- **Mộng Cảnh (DREAM_REALM):** có 1 mộng cảnh cho mỗi lưu phái (49 path). Hoàn thành mộng cảnh của lưu phái X giúp **tăng cảnh giới lưu phái X** (thưởng +50 Chân Ý cho đúng path đó). Vào bằng `dream_material`, cooldown 24 giờ/lần, tiêu hao Soul HP.
- **Bí Cảnh (SECRET_REALM):** phó bản chiến đấu thưởng Đạo Ngân theo path, cooldown 48 giờ.
- **Truyền Thừa (INHERITANCE_*):** phó bản cốt truyện riêng (vd. Truyền Thừa Hoa Tửu Hành Giả).

## 11. Hồn phách

- Player có `soul_hp`, `max_soul`, `soul_tier`, `needs_soul_refine`.
- API hỗ trợ dùng Guts Gu, refine soul và calm soul.

## 12. Kinh tế

### Cửa hàng NPC

- Danh sách cửa hàng public; mua/bán cần đăng nhập.
- Dữ liệu cửa hàng seed từ `backend/ShopItem.csv`.

### Bảo Hoàng Thiên chợ trời

- Hỗ trợ niêm yết kiểu chợ, đấu giá và giao dịch/mục tiêu người mua.
- Có đặt giá, mua, hủy và kết toán đấu giá hết hạn.
- Có chat Bảo Hoàng và SSE messages.

### Giao dịch trực tiếp

- Hai Cổ sư có thể khởi tạo, chấp nhận/từ chối, đề nghị, khóa và xác nhận.
- Đề nghị hỗ trợ nguyên thạch, Tiên Nguyên Thạch, Cổ, vật liệu và **Bí phương**.
- Có SSE trade events.

### Ví, nạp tiền và gói quý

- Ví có nguyên thạch, tiên nguyên thạch, tiên ngọc.
- Đơn nạp dùng package và PayOS webhook nếu cấu hình.
- Cửa hàng gói quý mua bằng ví.
- **Đổi đá** (tab "Đổi Đá" trong Cát Bảo Các): đổi Tiên Nguyên Thạch ↔ Nguyên Thạch theo tỷ giá cố định — 1 Tiên Nguyên Thạch = 100 Nguyên Thạch, 100.000 Nguyên Thạch = 1 Tiên Nguyên Thạch.

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
- **Chiến đấu tổ đội:** Khi các thành viên trong tổ đội đứng cùng một ô trên bản đồ, họ sẽ cùng tham gia vào các trận chiến (cả Turn-based lẫn Auto Combat/Simulation). Đồng đội sẽ tự động chiến đấu bằng các Cổ trùng đang trang bị, Sát chiêu và Thiết lập tự động của chính họ. Phần thưởng chiến thắng (Nguyên thạch, Nguyên liệu, Cổ trùng) sẽ được chia đều cho tất cả thành viên tham chiến, và tiêu hao (Sinh lực, Chân Nguyên, Dược phẩm) cũng được cập nhật tương ứng.
- Đấu trường: leaderboard và khiêu chiến Cổ sư khác.

## 16. Chat, mail và feedback

- Mail riêng: gửi và xem inbox.
- Bảo Hoàng chat theo channel, fetch message và subscribe SSE. Hệ thống tích hợp bộ lọc chặn spam (giới hạn thời gian gửi 1.5s, cấm trùng lặp tin nhắn trong 10s, cấm lặp từ hoặc chuỗi ký tự liên tiếp) và chặn từ ngữ tục tĩu nhạy cảm.
- Feedback (Thỉnh Thiên Đạo): mỗi sớ trình là một topic. Xem danh sách, gửi góp ý, bình chọn, góp ý (comment), cập nhật trạng thái/developer response.
- **Quyền theo trạng thái topic:**
  - `pending` (chờ duyệt): không bình chọn, không góp ý.
  - `active` (đang mở): bình chọn + góp ý.
  - `in_progress` / `completed`: không bình chọn, chỉ góp ý.
  - `rejected`: khép lại, chỉ xem.
- Cả bình chọn và góp ý đều yêu cầu Cổ sư đạt **Nhị Chuyển (Chuyển ≥ 2)** trở lên. Tác giả hoặc Admin có thể xóa góp ý.
- Thưởng: khi topic được duyệt triển khai (`in_progress`/`completed`), tác giả nhận 5 Tiên Nguyên (Chuyển ≥ 6) hoặc 500 Nguyên Thạch (phàm nhân).

## 17. Hiệu ứng Cổ trùng đặc sắc (Special Gu Effects)

Hệ thống tích hợp 18 Cổ trùng đặc biệt mang lại các hiệu ứng độc đáo cả trong và ngoài chiến đấu:

### Trong chiến đấu (Combat Simulator)
1. **Xuân Thu Thiền (GU_SPRING_AUTUMN_CICADA)**: Khi chết, có 15% cơ hội hồi sinh nghịch chuyển thời không quay về Lượt 1.
2. **Định Tiên Du (GU_FIXED_IMMORTAL_TRAVEL)**: Né tránh hoàn toàn đòn đánh đầu tiên ở Lượt 1.
3. **Ngự Lang Vương Cổ (GU_WOLF_ENSLAVEMENT_CORE)**: Triệu hồi Địa Lang Vương lên tiền tuyến gánh chịu sát thương cho Cổ sư.
4. **Cổ Gan Dạ (GU_GUTS)**: Hồi 100% Sinh lực tâm linh lập tức nhưng tiêu tốn lượt hành động của lượt hiện tại.
5. **Kiếm Long Biến Cổ (GU_SWORD_DRAGON_TRANSFORM)**: Chuyển hóa toàn bộ Sinh lực thành Giáp Ảo (giữ lại 1 Sinh lực) và tăng 50% sát thương cận chiến trong 3 lượt.
6. **Cổ Phá Không (GU_BREAK_SPACE)**: Tấn công xuyên thẳng qua Chân Nguyên Giáp hộ thể và bỏ qua 30 phòng ngự vật lý.
7. **Ngã Ý Cổ (GU_MY_WILL)**: Đề kháng khống chế tinh thần, 35% giải trừ trạng thái Choáng vào đầu mỗi lượt.
8. **Giải Mê Cổ (GU_UNRAVEL_RIDDLE)**: Ngắt đòn tụ lực (channeling) của đối phương và hóa giải 2 buff lợi ích của kẻ địch.
9. **Tự Tại Thiên Địa Tiền (GU_PRESENT_DAYS_PAST)**: Mượn sức mạnh quá khứ đỉnh phong tăng 40% sát thương trong 3 lượt.
10. **Cổ Tình Yêu (GU_LOVE_GU)**: Sát chiêu nhân quả ngẫu nhiên đôi:
    - *Hiệu ứng*: Crit 500%, Bất tử 1 lượt, hoặc giảm Sinh lực địch xuống còn 10%.
    - *Cái giá*: Giảm vĩnh viễn 10% Tư chất, giảm vĩnh viễn 5 năm tuổi thọ, hoặc mất 50% Nguyên Thạch.

### Ngoài chiến đấu (Handlers)
1. **Trí Tuệ Cổ (GU_WISDOM_GU)**: Tỏa ra vầng sáng trí tuệ, giảm 50% chi phí Chân Nguyên/Nguyên Thạch khi thôi diễn và luyện cổ, đồng thời tăng 30% tỷ lệ thành công.
2. **Huyết Sọ Cổ (GU_BLOOD_SKULL)**: Cho phép hiến tế đệ tử môn phái để gia tăng vĩnh viễn tư chất Không khiếu của bản thân (+5% tư chất/lần, tối đa 99% - tư chất Giáp).
3. **Cổ Thiên Cơ (GU_HEAVEN_SECRET)**: Tăng gấp đôi tầm nhìn khám phá bản đồ (Vision Radius) và hiển thị trực quan tỷ lệ rớt đồ của quái vật trên bản đồ.
4. **Cổ Thọ Nguyên 100 Năm (GU_LIFESPAN_100)**: Tiêu hao để gia tăng vĩnh viễn 100 năm tuổi thọ cho Cổ sư.
5. **Tiên Thai Chí Tôn (GU_SOVEREIGN_FETUS)**: Cơ thể Tiên Thai tối cao bỏ qua hoàn toàn bài xích/xung đột Đạo văn trong Tiên khiếu (Node) và khi thiết lập sát chiêu liên hoàn. Tăng 100% sát thương tất cả các hệ.

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

Hệ thống cung cấp trang Hướng Dẫn Tân Thủ chi tiết từng bước cho các hoạt động tại [Wiki Guide](file:///wiki/guide) bao gồm:

### Thuật ngữ cốt lõi:
- **Không Khiếu & Tiên Khiếu:** Phàm nhân sử dụng Không Khiếu tích lũy Chân Nguyên chiến đấu. Cổ Tiên (Lục Chuyển trở lên) sở hữu **Tiên Khiếu** (thế giới nhỏ độc lập bên trong cơ thể) tự sản sinh Tiên Nguyên với 4 phẩm cấp: Thấp, Trung, Cao, Cực Hạn dựa trên Nhân Khí lúc Thăng Tiên.
- **Đạo Ngân (Dao Marks):** Vết khắc của quy luật tự nhiên, chia thành 49 lưu phái (Lực đạo, Hỏa đạo...). Mỗi điểm Đạo Ngân tăng **+1%** sát thương/phòng ngự cho các kỹ năng/Cổ trùng thuộc lưu phái đó (soft cap ở 10.000 điểm).
- **Cảnh Giới Lưu Phái (Attainment):** Độ ngộ Chân Ý gồm 5 bậc: Phàm Thai ➔ Đại Sư ➔ Tông Sư ➔ Đại Tông Sư ➔ Vô Thượng Đại Tông Sư. Điểm tích lũy theo từng lưu phái (thi triển sát chiêu thành công, luyện Cổ thành công, vượt ải phó bản). Cảnh giới suy ra từ mốc điểm: Đại Sư=100, Tông Sư=500, Đại Tông Sư=2.500, Vô Thượng=10.000. **Để đột phá lên Rank 7/8/9 khi vượt Kiếp Nạn, cần ít nhất 1 lưu phái đạt Tông Sư/Đại Tông Sư/Vô Thượng tương ứng.** Cảnh giới càng cao càng tăng tỉ lệ thành công của sát chiêu và luyện Cổ.
- **Bản Mệnh Cổ (Vital Gu):** Cổ trùng gắn liền sinh mệnh, không bao giờ mất khi thất bại dã ngoại.
- **Độ No (Hunger):** Cổ trùng giảm Độ No mỗi 3 giờ. Cần cho ăn đúng loại thức ăn (thịt thú, khoáng thạch...), nếu về 0 Độ No Cổ sẽ **chết vĩnh viễn**.
- **Sát Chiêu (Killer Moves):** Kỹ năng ghép từ 1 Cổ Hạch Tâm (Core Gu) và 0-8 Cổ Phụ Trợ (Support Gu) tăng sát thương, giảm hao Chân Nguyên, thêm khống chế (choáng, đóng băng, câm lặng) hoặc thời gian tụ lực.
- **Kiếp Nạn (Tribulation):** Cổ Tiên phải chống chọi các kiếp nạn (Địa Tai, Thiên Kiếp...) định kỳ bảo vệ tài nguyên Tiên Khiếu. Vượt qua kiếp nạn nhận được lượng lớn Đạo Ngân.

### Các hoạt động chính:
1. **Tạo nhân vật & Thức tỉnh Linh Căn:** Chọn tên và thức tỉnh Linh Căn ngẫu nhiên.
2. **Trùng kích Vách Khiếu:** Đột phá tu vi phàm nhân từ Nhất Chuyển đến Ngũ Chuyển sử dụng Nguyên Thạch.
3. **Khai thác mỏ & Thám hiểm:** Đào Nguyên Thạch dã ngoại (coi chừng Quái Mỏ) và mở bản đồ sương mù nhặt dược thảo.
4. **Lắp ráp Sát Chiêu & Luyện Cổ:** Chế tạo nâng cấp Cổ trùng tại lò luyện và tự do phối hợp Cổ trùng tạo Sát Chiêu độc bản.
5. **Ải Cốt truyện & Đấu Trường:** Đi ải cốt truyện và khiêu chiến Cổ sư khác kiếm Chiến Công đổi đồ.
6. **Thương hội & Tông môn:** Đấu giá vật phẩm tự do tại Bảo Hoàng Thiên và các hoạt động quyên góp, giao dịch bảo khố Tông Môn.
7. **Thăng Tiên:** Đột phá lên Lục Chuyển Cổ Tiên ở Ngũ Chuyển Đỉnh Phong dựa trên tích lũy Nhân Khí (chance tỷ lệ thuận với Nhân Khí, tối đa 100k = 100% thành công). Thất bại phế bỏ tu vi về Nhất Chuyển.
