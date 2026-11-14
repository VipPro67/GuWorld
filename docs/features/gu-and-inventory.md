# Cổ trùng và túi đồ

## Tính năng làm gì?

Inventory lưu các Gu instance, vật phẩm và nguyên liệu của player. Màn Túi Đồ cho phép đọc trạng thái từng instance, cho ăn, trang bị, chọn Bản Mệnh, đổi trạng thái, dùng Gu hỗ trợ/tiêu hao và xử lý durability. Các thay đổi quan trọng do server xác thực và đồng bộ lại qua response/player events.

## Điều kiện mở và quyền truy cập

- Nhóm `/inventory` yêu cầu đăng nhập và onboarding.
- Instance phải thuộc player hiện tại; các thao tác kiểm tra trạng thái Gu trước khi ghi.
- Gu `DEAD` không thể feed/equip. Gu `DAMAGED` hoặc durability bằng 0 không thể equip.
- Vital Gu không thể unequip bằng thao tác thường; chọn vital cần Gu hợp lệ và tương thích rank.
- Gu đang được reserve bởi Sát Chiêu/Gu House, tham gia operation, hoặc bị khóa do trạng thái có thể không dùng được cho một action khác.

## Cách sử dụng

### Cho ăn

- Dùng `/inventory/feed` cho một instance. Chọn thức ăn tương thích với `food_types` của Gu.
- `/feed-group` áp dụng cho Gu cùng loại/rank được gửi; `/feed-all` có thể giới hạn scope `all`, `equipped`, `unequipped` hoặc `rest`.
- Gu đủ hunger không cần thức ăn; batch feed bỏ qua Gu không hợp lệ, `DEAD`/`SEALED` hoặc đã no theo phạm vi xử lý.
- Khi dùng một Gu khác làm thức ăn, Gu nguyên liệu phải thỏa điều kiện không vital, không equipped và trạng thái được phép.

### Equip và vital

1. Chọn Gu instance và gọi `/inventory/equip`.
2. Server kiểm tra status, durability, refinement progress, rank và quyền Gu theo vùng.
3. Chọn vital bằng `/inventory/set-vital`; vital Gu đồng thời được yêu cầu giữ ở trạng thái equipped.
4. Sau thay đổi rank/vùng, một số Gu đã equip có thể bị tự động unequip nếu không còn hợp lệ.

### Status và sử dụng Gu

`/inventory/status` đổi trạng thái giữa `ACTIVE`, `SLEEPING`, `SEALED`. Khi rời ACTIVE, server buộc unequip nếu cần; SEALED tiêu hao `sealing_stone` theo rank. `/use-gu` dùng Gu tiêu hao hoặc capability tương ứng; `/use-support` chạy một số hiệu ứng hỗ trợ ngoài combat. Không phải active effect nào cũng dùng được ngoài battle.

### Subjugation và durability

- `/subjugate-gu` tăng tiến độ luyện hóa ý chí cho Gu dã tính theo điều kiện người chơi/Gu.
- `/nourish-gu` hồi durability cho Gu bị thương nhẹ (durability từ 20 trở lên); mỗi lần hồi 15 điểm, tối đa 100.
- `/repair-gu` phục hồi Gu bị hư nặng (durability dưới 20 hoặc state DAMAGED) về 50 và ACTIVE.
- Gu đủ điều kiện có thể tự hồi durability theo thời gian, nhưng việc này tiêu hao hunger; không được hiểu là repair miễn phí tức thời.

## Luật, chi phí và giới hạn

### Công thức cho ăn và durability

Nutrition cơ bản (code `player_feed.go`): nếu rank thức ăn `>=` rank Gu thì được 100; nếu thấp hơn thì `floor(RankScale(foodRank)/RankScale(guRank)*100)`. Không nhận diện được rank thức ăn thì fallback 100. Bonus là `min(0.50, foodDaoMarks/10000)` cộng 0.15 cho mỗi Gu path Thực đang ACTIVE và equipped; tổng bonus bị cap 1.0. Nutrition cuối là `round(base*(1+bonus))`. Đây là lượng hunger cộng trước khi cap ở `MaxHunger`.

Tài nguyên đặc biệt cho một serving có cost riêng: Primeval Stones `2^guRank`, Immortal Stones `guRank`, blood 10 HP, thought 10, soul 10 Soul HP, lifespan 1; blood/soul/lifespan phải còn nhiều hơn cost để tránh xuống 0 hoặc thấp hơn.

Passive durability recovery được tính lazy khi Gu đủ điều kiện và đã hơn một giờ từ lần cập nhật: `gain = min(100-durability, elapsedHours*2, hunger/2.5)`, đồng thời hunger giảm `2.5*gain`. Nourish tăng 15 durability; repair đặt durability 50 và ACTIVE. Mana cost cùng rank là nourish 10, repair 5, nhân theo `rankWeight[guRank]/rankWeight[playerRank]`, với weight rank 1–9 là `[1,2,5,20,100,1000,10000,100000,1000000]`.

- Food mapping, max hunger, hunger rate, type/rank, effects và mana cost lấy từ Gu/item data.
- Feed kiểm tra cả food types và số lượng. Group/all feed chỉ xử lý mục tiêu đủ điều kiện; không đảm bảo mọi instance trong túi đều thay đổi.
- Equip cần refinement progress 100%, rank Gu không vượt rank player cho các slot/vital bị ràng buộc, và region policy cho phép.
- Phong ấn tiêu hao sealing stone theo rank Gu.
- Chi phí mana của nourish/repair scale theo rank player và rank Gu. Đọc response để biết lượng thực tế.
- Drop Gu là thao tác phá hủy/đưa ra world flow; kiểm tra xác nhận trong UI trước khi gửi.

## Kết quả và trường hợp thất bại

- Server trả player/inventory delta hoặc snapshot sau mutation. Dùng response đó để cập nhật state; không tự giữ optimistic state khi server từ chối.
- Invalid instance/ownership, dead/sealed state, mismatch food, thiếu tài nguyên hoặc region policy có thể trả 4xx.
- Một số invalid equipped Gu có thể bị cleanup/unequip trong mutation khác, chẳng hạn rank giảm.
- Gu có thể bị tiêu hao bởi food, refining, product composition, combat hoặc death/lifecycle rules. Timeline lịch sử áp dụng cho loại Gu đủ điều kiện.

## Liên hệ với hệ thống khác

- Inventory là nguồn Gu component cho combat, Sát Chiêu, Gu House, luyện Cổ, capture và aperture swarm.
- Hunger worker tác động tới survival; combat/đói có thể làm Gu chết hoặc giảm durability.
- Active products có reservation; đọc [Sát Chiêu và Tiên Cổ Ốc](killer-moves-and-gu-houses.md) trước khi đổi/xóa thành phần.
- Item consumption effects khác với Gu `/use-gu`; xem data effect adapter trong [GAME_DATA.md](../guides/GAME_DATA.md).

## Dữ liệu/cấu hình ảnh hưởng

- `backend/data/Gu.csv` định nghĩa food types, hunger, rank, path, effects, target type và metadata Gu.
- `backend/data/Item.csv` định nghĩa vật phẩm thức ăn/tiêu hao.
- Runtime cache: `models.GuRegistry`, `models.ItemRegistry`; import qua Admin → Dữ liệu.
- Durability, reservations và instance inventory lưu ở player Gu schema/domain tables.

## Mã nguồn liên quan

- Frontend: `frontend/src/components/GuInventory.tsx`, `frontend/src/store/slices/inventorySlice.ts`.
- Backend: `backend/internal/handlers/player_feed.go`, `player_equip.go`, `player_gu_use.go`, `gu_subjugate.go`, `gu_durability.go`, `ground_drops.go`.
- Persistence/services: `backend/internal/db/player_gu.go`, `schema_player_items.go`, `backend/internal/services/inventory_service.go`, `backend/internal/playerstate/`.
- Route ownership: `backend/internal/server/server.go` group `/inventory`.

## API liên quan

`POST /inventory/feed`, `/feed-group`, `/feed-all`, `/equip`, `/subjugate-gu`, `/nourish-gu`, `/repair-gu`, `/unequip-all`, `/status`, `/set-vital`, `/use-gu`, `/use-support`, `/drop`, `/drop-batch`. Player read endpoints gồm `/player/inventory/gus` và `/player/status/:id`.

## Ví dụ flow

```text
GET /player/inventory/gus → chọn instance đang ACTIVE
→ POST /inventory/feed (nếu thiếu hunger)
→ POST /inventory/equip hoặc /inventory/set-vital
→ áp dụng player/inventory event từ server
```
