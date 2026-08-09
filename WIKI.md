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

**Giới tính:** Nam (`male`) / Nữ (`female`) — chưa có hiệu ứng (chỉ hiển thị).

**Trạng thái:**

| Trạng thái | Hiệu ứng |
| --- | --- |
| Trọng Thương (`severely_wounded`) | **-20%** tấn công, phòng thủ, tốc độ |

**Thể chất (10 loại — tất cả đều +10% hồi Chân Nguyên):**

| Thể chất | Hiệu ứng |
| --- | --- |
| Hoang Cổ Nguyệt Thể (`desolate_ancient_moon_body`) | +20% hiệu quả tu luyện, +15% Chân Nguyên tối đa |
| Thanh Liên Đại Nhật Thể (`verdant_great_sun_body`) | Hỏa Đạo +20% affinity, +10% tấn công |
| Bắc Minh Băng Phách Thể (`northern_dark_ice_soul_body`) | Hàn Băng Đạo +20% affinity, +10% phòng thủ |
| Cổ Nguyệt Âm Đức Thể (`ancient_yin_virtue_body`) | Hồn Đạo +15% affinity, +10% hồi phục nhận |
| Hạo Kiếp Viêm Quang Lôi Quang Thể (`blazing_glory_lightning_brilliance_body`) | Lôi Đạo +20% affinity, +10% tốc độ |
| Thuần Mộng Cầu Chân Thể (`pure_dream_reality_seeker_body`) | Mộng Đạo +25% affinity, +15% sát thương Mộng Đạo |
| Vô Cực Thiên Ngân Thể (`unrestrained_heavenly_mark_body`) | +15% Đạo Ngân nhận được |
| Đại Lực Chân Vũ Thể (`great_strength_true_martial_body`) | Lực Đạo +20% affinity, +15% tấn công |
| Tiêu Dao Liên Tâm Thể (`carefree_lotus_heart_body`) | Trí Tuệ Đạo +15% affinity, +20% kháng tâm lý |
| Cổ Hồn Âm Đức Thể (`ancient_soul_virtue_body`) | Hồn Đạo +20% affinity, +15% phòng thủ hồn |

Trạng thái có thể tồn tại đồng thời nhiều cái (mảng `statuses`). Danh sách và con số hiệu ứng được định nghĩa tập trung tại `backend/internal/gamefx/fx.go`.


## 2. Tu luyện

### Chuyển và tiểu cảnh giới

- Đại cảnh giới tu vi từ Nhất Chuyển đến Cửu Chuyển (1 đến 9).
- `sub_rank` gồm 4 bậc: Sơ giai, Trung giai, Cao giai, Đỉnh phong (chỉ dùng cho Chuyển 1-5).
- Nhất Chuyển đến Ngũ Chuyển là phàm nhân; Lục Chuyển trở lên là Cổ Tiên.
- Lục Chuyển trở lên **không có sub_rank**. Thay vào đó hiển thị tên `Lục Chuyển/Bát Chuyển/Cửu Chuyển Cổ Tiên/Cổ Tôn` kèm số lần vượt qua từng loại kiếp nạn (Địa Tai, Thiên Kiếp, Hạo Kiếp, Vạn Kiếp, Hỗn Độn Đại Nạn).

### Phàm nhân: phá vách

- Cổ sư tiêu hao Chân Nguyên để tăng `wall_progress`.
- Khi đủ tiến độ, nhân vật đột phá tiểu cảnh giới hoặc lên Chuyển kế tiếp.
- Tư chất ảnh hưởng tốc độ hồi/phá vách: Giáp, Ất, Bính, Đinh.
- Xá Lợi Cổ có thể được dùng qua hệ thống `use-relic-gu` nếu bọc đồ có Cổ phù hợp.

### Thăng Tiên và Độ Kiếp

- Ngũ Chuyển Đỉnh phong (Chuyển 5, SubRank 4) có thể thăng tiên khi Nhân Khí đạt mức mong muốn.
- **Tỉ lệ thành công tuyến tính theo Nhân Khí:** `chance = NhânKhí / 100.000`. 0 Nhân Khí → 0%, 40.000 → 40%, 100.000 → chắc chắn thành công.
- **Thành công:** lên Lục Chuyển Cổ Tiên, nhận 10 Tiên Nguyên Thạch, Đạo Ngân Thiên/Địa = 100, kiếp đầu tiên sau 10 ngày. **Tiên Khiếu được khai mở ngay**, phẩm cấp được định đoạt theo Nhân Khí tại thời điểm thăng tiên (xem mục Tiên khiếu).
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

- `time_rate` (1.0x - 100.0x) điều chỉnh tốc độ thời gian bên trong Tiên Khiếu.
- **Ảnh hưởng đến:**
  - Tốc độ thu hoạch tài nguyên (cây, quặng, linh thú)
  - Tăng trưởng dân cư dị nhân
  - Tốc độ hồi Tiên Nguyên
  - Tích lũy năm nội bộ → Kiếp nạn đến nhanh hơn
  - Đói Cổ trùng (Cổ đói nhanh hơn khi time rate cao)
- **Không ảnh hưởng đến:**
  - Thọ nguyên (luôn tính theo thời gian thực)
  - Cooldown/sự kiện bên ngoài Tiên Khiếu
  - Thời gian ở Ngũ Vực

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

**Tiên Cổ duy nhất:** Cổ từ **Lục Chuyển trở lên (Tiên Cổ)** là độc nhất toàn thiên hạ — mỗi `gu_id` chỉ tồn tại một instance trên toàn server. Các kênh nhận Cổ (Mở đá, Rương Mỏ Linh Thạch, Luyện Cổ, quái rơi đồ) đều loại trừ Tiên Cổ đã tồn tại (trong túi bất kỳ đạo hữu, đấu giá, rương tạm) trước khi roll; nếu hết ứng viên hợp lệ sẽ rơi vật phẩm/thạch bù trừ.

**Hiệu ứng Trưởng thành (Growth):** Một số Cổ (Lực đạo/Thổ đạo như Hùng Lực, Hắc Thỉ, Ngọc Bì, Thiết Cốt, Tự Lực...) có hiệu ứng `growth` — khi trang bị ở trạng thái ACTIVE, chỉ số thưởng (Công/Thủ/Máu/Tốc) tăng dần theo thời gian tới mức tối đa (`hours_to_max` giờ). Quy tắc:

- **Chỉ tăng 1 lần cho mỗi loại (GU ID + Chuyển):** dù sở hữu nhiều bản cùng loại, tiến độ và chỉ số chỉ tính 1 lần; không cộng dồn theo số bản sở hữu.
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

- Bản đồ theo region, x/y, explored cells và cleared cells.
- Cổ sư xem lưới địa đồ, di chuyển, thám hiểm node và rút lui khỏi combat/node.
- Vùng seed hiện tại nằm trong `backend/Region.csv`.

## 9. Chiến đấu

- Combat PvE khởi tạo từ thám hiểm, mỏ hoặc các hệ thống gọi combat.
- Cổ sư dùng Cổ trang bị, có thể charge và chọn mục tiêu.
- Chỉ định mục tiêu hỗ trợ vị trí/hàng/cột/toàn bộ tùy `target_type` của Cổ.
- Combat state hỗ trợ đội hình địch/ta, status effects, cooldown, Chân Nguyên/Sinh lực, chiến lợi phẩm tạm.
- Thắng trận áp dụng chiến lợi phẩm; thua trận cập nhật trạng thái theo handler combat.

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
- **Độ no:** Giảm theo `pet_hunger_per_hour` (mặc định 2/giờ). Sủng vật đang ngủ giảm 50%. Khi về 0 → **chết vĩnh viễn** (xóa khỏi DB).
- **Cho ăn:** Dùng vật phẩm có effect `hunger` (Thịt Lợn Rừng +40, Rượu Thanh Trúc +30, Ngọc Thạch +25, Nguyệt Lan +20).
- **Gộp:** Cùng `monster_template_id` có thể gộp thành 1 gói. Chỉ số × `N^0.65`, độ no × `1 + 0.3*(N-1)`.
- **Chết:** Sinh lực = 0 trong combat hoặc hết độ no → xóa vĩnh viễn. Không thể hồi sinh.

## 10. Phó bản

- Instance có template, session active và action theo option index.
- Cổ sư vào instance, chọn hành động và nhận kết quả theo stage/map data.

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
- Đề nghị hỗ trợ nguyên thạch, Tiên Nguyên Thạch, Cổ và vật liệu.
- Có SSE trade events.

### Ví, nạp tiền và gói quý

- Ví có nguyên thạch, tiên nguyên thạch, tiên ngọc.
- Đơn nạp dùng package và PayOS webhook nếu cấu hình.
- Cửa hàng gói quý mua bằng ví.

## 13. Môn phái

Hệ thống môn phái hiện có:

- Xem/tạo/gia nhập/rời môn phái.
- Duyệt đơn gia nhập nếu môn phái bật yêu cầu duyệt.
- Thành viên, ban bậc, trục xuất.
- Xin/thẩm định thăng chức.
- **Cấp bậc:** Ngoại Môn → Nội Môn → Tinh Anh → Chân Truyền → Trưởng Lão → **Hộ Pháp** → Chưởng Môn. Chi phí thăng cấp bằng Điểm cống hiến: 100 / 500 / 2.000 / 10.000 / 30.000.
- **Duyệt lên cấp:** Trưởng Lão/Chưởng Môn có thể bật tùy chọn "Duyệt Lên Cấp" (mặc định tắt). Khi bật, **mọi** thăng cấp của đệ tử đều tạo yêu cầu chờ duyệt thay vì tự động. Khi tắt, đệ tử tự thăng cấp theo tiêu chuẩn (trừ bậc Chân Truyền/Trưởng Lão/Hộ Pháp vẫn cần duyệt).
- Đóng góp nguyên thạch/vật liệu/Cổ.
- Bảo khố môn phái, mua vật phẩm từ kho, stock kho.
- Nhiệm vụ mặc định và nhiệm vụ custom:
  - **Hạn tự xóa:** Nhiệm vụ trên bảng có thời hạn (mặc định 24h, Trưởng Lão/Chưởng Môn tùy chỉnh khi phát hành; `-1` = không hết hạn). Hết hạn nhiệm vụ tự biến mất khỏi bảng, nhiệm vụ của đệ tử đang làm cũng bị hủy theo.
  - **Hủy nhiệm vụ:** Đệ tử có thể hủy bỏ nhiệm vụ đang thực hiện để nhận nhiệm vụ khác. Trưởng Lão/Chưởng Môn có thể xóa nhiệm vụ đã phát hành khỏi bảng.
- Nâng cấp môn phái: chi phí `Cấp² × 5.000` Quỹ.
- Mỏ môn phái: xem, tấn công, nhận thưởng.

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
4. **Cổ Gan Dạ (GU_GUTS_GU)**: Hồi 100% Sinh lực tâm linh lập tức nhưng tiêu tốn lượt hành động của lượt hiện tại.
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
- **Tư chất:** Thập tuyệt thể (100%), Giáp đẳng (80-90%), Ất đẳng (60-70%), Bính đẳng (40-50%), Đinh đẳng (20-30%).

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
- **Cảnh Giới Lưu Phái (Attainment):** Độ ngộ Chân Ý gồm 5 bậc: Phàm Thai ➔ Đại Sư ➔ Tông Sư ➔ Đại Tông Sư ➔ Vô Thượng Đại Tông Sư. Giúp nâng tỉ lệ thành công của sát chiêu và luyện Cổ.
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
