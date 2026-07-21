# GuWorld Gameplay Wiki

Wiki này phản ánh các hệ thống đã có trong codebase hiện tại.

## 1. Tài khoản và nhân vật

- Người chơi đăng ký bằng username/email/password hoặc đăng nhập Google nếu cấu hình OAuth.
- Hồ sơ hỗ trợ bio, avatar, đổi mật khẩu và cơ chế ngụy trang thông tin công khai.
- Referral có thống kê người được mời và phần thưởng/mốc xử lý ở backend.

## 2. Tu luyện

### Rank và tiểu cảnh giới

- Rank chính từ 1 đến 9.
- `sub_rank` gồm 4 bậc: Sơ giai, Trung giai, Cao giai, Đỉnh phong (chỉ dùng cho Rank 1-5).
- Rank 1-5 là phàm nhân; Rank 6+ là Cổ Tiên.
- Rank 6+ **không có sub_rank**. Thay vào đó hiển thị tên `Lục Chuyển/Bát Chuyển/Cửu Chuyển Cổ Tiên/Cổ Tôn` kèm số lần vượt qua từng loại kiếp nạn (Địa Tai, Thiên Kiếp, Hạo Kiếp, Vạn Kiếp, Hỗn Độn Đại Nạn).

### Phàm nhân: phá vách

- Người chơi tiêu hao chân nguyên/mana để tăng `wall_progress`.
- Khi đủ tiến độ, nhân vật đột phá tiểu cảnh giới hoặc lên rank kế tiếp.
- Tư chất ảnh hưởng tốc độ hồi/phá vách: Giáp, Ất, Bính, Đinh.
- Xá Lợi Cổ có thể được dùng qua hệ thống `use-relic-gu` nếu inventory có Cổ phù hợp.

### Thăng Tiên và Độ Kiếp

- Rank 5 Đỉnh phong có thể thăng tiên khi đạt điều kiện backend yêu cầu.
- Cổ Tiên có tiên khiếu, đạo ngân, tiên nguyên/tài nguyên riêng.
- Tai kiếp có worker kiểm tra tự động mỗi 60 giây. Khi năm nội bộ đạt mốc tiếp theo, worker spawn kiếp và resolve.
- **Kiếp nạn theo cấp bậc:**
  - Rank 6: Địa Tai (10 năm), Thiên Kiếp (100 năm)
  - Rank 7: Địa Tai (10 năm), Thiên Kiếp (50 năm), Hạo Kiếp (100 năm)
  - Rank 8: Thiên Kiếp (10 năm), Hạo Kiếp (50 năm), Vạn Kiếp (100 năm)
  - Rank 9: Hạo Kiếp (10 năm), Vạn Kiếp (50 năm), Hỗn Độn Đại Nạn (100 năm)
- **Tự động đột phá:** Khi đủ 3 lần vượt kiếp cấp cao nhất (vd: 3 Thiên Kiếp), kết hợp điều kiện số năm ở rank (300 năm với 6→7, 7→8), Cổ Tiên tự động lên rank kế tiếp.
- Nhiều kiểu kiếp nạn đến cùng lúc: ưu tiên spawn kiểu hiếm nhất, nhưng đếm tất cả các kiểu đang đến hạn.

## 3. Không khiếu, tiên khiếu và đạo ngân

### Không khiếu/phàm nhân

- UI `Aperture` hiển thị mana, HP, rank, tư chất, Cổ trang bị, quest và tu luyện.
- Mana/HP được frontend ước tính theo thời gian từ snapshot server.

### Tiên khiếu

- Tiên khiếu có grade, time rate, HP, diện tích, immortal essence, sản lượng/năm, state.
- **Khi mới khai mở**, Tiên Khiếu được tạo sẵn 2-4 mỏ tài nguyên ngẫu nhiên (tuỳ grade):
  - LOW: 2 mỏ, MID: 3 mỏ, HIGH/EXTREME: 4 mỏ.
  - Các mỏ khởi đầu: Nguyệt Lan, Thanh Trúc, Ngọc Thạch, Nguyên Thạch, Kiếm Diệp, Hươu Linh.
- Người chơi đặt thêm node tài nguyên theo toạ độ, thu hoạch, gỡ/xóa, sửa chữa node.
- Hệ sinh thái node có kiểm tra xung đột đường tu/path và cân bằng plant/herbivore/carnivore.
- Dị nhân trong tiên khiếu có population, happiness, oppression, rebellion rate và labor intensity.

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

## 4. Cổ trùng và inventory

Mỗi Cổ instance có:

- `instance_id`, `gu_id`.
- Hunger (giảm theo `HungerRate * TimeSpeedMultiplier` mỗi 3 giờ thực tế, Cổ trong Tiên Khiếu có time rate cao sẽ đói nhanh hơn).
- Status: `ACTIVE`, `SLEEPING`, `DEAD`.
- `equipped` và `is_vital`.

Người chơi có thể:

- Cho ăn từng Cổ hoặc theo nhóm.
- Trang bị/tháo Cổ.
- Đặt Cổ bản mệnh.
- Chuyển trạng thái active/sleeping.
- Dùng Cổ tiêu hao hoặc Cổ hỗ trợ.
- Drop Cổ.

## 5. Vật phẩm và nguyên liệu

- Item dictionary nằm trong `backend/Item.csv`.
- Vật phẩm có type, rarity, rank, path, effects, tags và description.
- Item effects hiện có gồm hồi mana/HP, soul boost, đạo ngân, xóa trạng thái xấu và điều kiện chỉ phàm nhân/chỉ Cổ Tiên.

## 6. Luyện Cổ và suy diễn công thức

- Recipe lấy từ `backend/Recipe.csv`.
- Luyện Cổ dùng `/api/refine` với recipe và selected Gu instances.
- Suy diễn recipe dùng `/api/refine/deduce`, cập nhật tiến độ unlock recipe trong player.

## 7. Sát chiêu

- Người chơi có thể tạo sát chiêu từ Cổ core + tối đa 8 Cổ support.
- Preview tính mana cost, stability, damage, defense, cooldown/cast time.
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
  - Huyết Sát Trận (GU_BLOOD_MOON + GU_BLOOD_ASSET + GU_BLOOD_SKULL): tiêu hao 10% HP, tăng 50% sát thương
  - Huyền Ảnh Sát (GU_BLOOD_MOON + GU_SHADOW + GU_SHADOW_STEPS): tấn công từ bóng tối không thể né
  - Ngũ Hành Sơn Hà (GU_CANOPY + GU_IRON_BONE + GU_EARTH_WORM): phòng ngự +200%, miễn khống chế
  - Tứ Hải Tửu Trận (GU_LIQUOR_WORM + GU_FOUR_FLAVORS_LIQUOR + GU_SEVEN_FRAGRANCES_LIQUOR + GU_NINE_EYES_LIQUOR): giảm 50% mana cost
- Sát chiêu đã lưu có thể cast trong combat và có thể bị xóa.

## 8. Khám phá bản đồ

- Map theo region, x/y, explored cells và cleared cells.
- Người chơi xem grid, di chuyển, explore node và retreat khỏi combat/node.
- Vùng seed hiện tại nằm trong `backend/Region.csv`.

## 9. Chiến đấu

- Combat PvE khởi tạo từ exploration, mine hoặc các hệ thống gọi combat.
- Người chơi dùng Cổ trang bị, có thể charge và chọn target.
- Targeting hỗ trợ vị trí/hàng/cột/toàn bộ tùy `target_type` của Cổ.
- Combat state hỗ trợ ally/enemy đội hình, status effects, cooldown, mana/HP, loot tạm.
- Thắng trận áp dụng loot; thua trận cập nhật trạng thái theo handler combat.

## 10. Phó bản

- Instance có template, session active và action theo option index.
- Người chơi vào instance, chọn hành động và nhận kết quả theo stage/map data.

## 11. Hồn phách

- Player có `soul_hp`, `max_soul`, `soul_tier`, `needs_soul_refine`.
- API hỗ trợ dùng Guts Gu, refine soul và calm soul.

## 12. Kinh tế

### Shop NPC

- Shop list public; mua/bán cần đăng nhập.
- Dữ liệu shop seed từ `backend/ShopItem.csv`.

### Bảo Hoàng Thiên marketplace

- Hỗ trợ listing kiểu market, auction và trade/target buyer.
- Có bid, buy, cancel và settle expired auctions.
- Có chat Bảo Hoàng và SSE messages.

### P2P trade

- Hai người chơi có thể initiate, accept/decline, offer, lock và confirm.
- Offer hỗ trợ nguyên thạch, tiên thạch, Cổ và materials.
- Có SSE trade events.

### Ví, nạp tiền và premium

- Wallet có primeval stone, immortal essence stone, immortal jade.
- Topup order dùng package và PayOS webhook nếu cấu hình.
- Premium shop package mua bằng ví.

## 13. Môn phái

Hệ thống môn phái hiện có:

- Xem/tạo/join/leave môn phái.
- Duyệt đơn gia nhập nếu môn phái bật yêu cầu duyệt.
- Thành viên, set rank, kick.
- Xin/thẩm định thăng chức.
- Đóng góp stones/materials/Gu.
- Kho môn phái, mua vật phẩm từ kho, stock kho.
- Nhiệm vụ mặc định và nhiệm vụ custom.
- Nâng cấp môn phái.
- Mỏ môn phái: xem, tấn công, claim reward.

## 14. Mỏ cá nhân

- Có danh sách mine slots với rarity và phần thưởng tích lũy.
- Người chơi chiếm mỏ, claim reward, bỏ mỏ hoặc tấn công mỏ người khác.
- Worker có thể gán quái vào mỏ bỏ hoang.

## 15. Party và Arena

- Party: tạo party, xem trạng thái, mời người chơi, rời party.
- Arena: leaderboard và challenge người chơi khác.

## 16. Chat, mail và feedback

- Mail riêng: gửi và xem inbox.
- Bảo Hoàng chat theo channel, fetch message và subscribe SSE.
- Feedback: xem danh sách, gửi góp ý, vote, cập nhật trạng thái/developer response.

## 17. Hiệu ứng Cổ trùng đặc sắc (Special Gu Effects)

Hệ thống tích hợp 18 Cổ trùng đặc biệt mang lại các hiệu ứng độc đáo cả trong và ngoài chiến đấu:

### Trong chiến đấu (Combat Simulator)
1. **Xuân Thu Thiền (GU_SPRING_AUTUMN_CICADA)**: Khi chết, có 15% cơ hội hồi sinh nghịch chuyển thời không quay về Lượt 1.
2. **Định Tiên Du (GU_FIXED_IMMORTAL_TRAVEL)**: Né tránh hoàn toàn đòn đánh đầu tiên ở Lượt 1.
3. **Ngự Lang Vương Cổ (GU_WOLF_ENSLAVEMENT_CORE)**: Triệu hồi Địa Lang Vương lên tiền tuyến gánh chịu sát thương cho Cổ sư.
4. **Cổ Gan Dạ (GU_GUTS_GU)**: Hồi 100% HP tâm linh lập tức nhưng tiêu tốn lượt hành động của lượt hiện tại.
5. **Kiếm Long Biến Cổ (GU_SWORD_DRAGON_TRANSFORM)**: Chuyển hóa toàn bộ HP thành Giáp Ảo (giữ lại 1 HP) và tăng 50% sát thương cận chiến trong 3 lượt.
6. **Cổ Phá Không (GU_BREAK_SPACE)**: Tấn công xuyên thẳng qua Chân Nguyên Giáp hộ thể và bỏ qua 30 phòng ngự vật lý.
7. **Ngã Ý Cổ (GU_MY_WILL)**: Đề kháng khống chế tinh thần, 35% giải trừ trạng thái Choáng vào đầu mỗi lượt.
8. **Giải Mê Cổ (GU_UNRAVEL_RIDDLE)**: Ngắt đòn tụ lực (channeling) của đối phương và hóa giải 2 buff lợi ích của kẻ địch.
9. **Tự Tại Thiên Địa Tiền (GU_PRESENT_DAYS_PAST)**: Mượn sức mạnh quá khứ đỉnh phong tăng 40% sát thương trong 3 lượt.
10. **Cổ Tình Yêu (GU_LOVE_GU)**: Sát chiêu nhân quả ngẫu nhiên đôi: 
   - *Hiệu ứng*: Crit 500%, Bất tử 1 lượt, hoặc giảm HP địch xuống còn 10%.
   - *Cái giá*: Giảm vĩnh viễn 10% Tư chất, giảm vĩnh viễn 5 năm tuổi thọ, hoặc mất 50% Nguyên thạch.

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
