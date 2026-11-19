# Dữ liệu gameplay

## Nguồn dữ liệu và vòng đời

CSV chuẩn nằm trong `backend/data/`. Admin → **Dữ liệu** đọc danh mục dataset dùng chung từ `backend/internal/db/datasets.go`. Sửa CSV trên disk chưa làm thay đổi DB đang chạy: cần import dataset. Một số registry được nạp vào bộ nhớ khi backend start; sau import, UI/server có thể reload một phần, còn dataset báo `restart_required` cần restart.

Import có bước chuẩn bị và bước commit. Trước commit, xem lỗi kiểm tra, số dòng insert/update/unchanged, thay đổi dự kiến và cờ reload/restart. Commit dùng plan đã xác nhận để tránh áp dụng file thay đổi sau khi review. Không sửa trực tiếp dữ liệu production bằng cách chỉ chép CSV.

`InitSchema()` tạo cấu trúc DB, compatibility DDL/backfill và seed một số dữ liệu nền; không tự import toàn bộ CSV. `Quest.csv` là một ngoại lệ được đồng bộ ở luồng khởi tạo. Các bảng từ điển race/constitution có key rỗng dự phòng để FK hoạt động trước khi import.

## Dataset Admin hiện hỗ trợ

| Key | CSV | Nội dung |
| --- | --- | --- |
| `item` | `Item.csv` | Vật phẩm |
| `gu` | `Gu.csv` | Cổ trùng và effects |
| `constitution` | `Constitution.csv` | Thể chất |
| `race` | `Race.csv` | Chủng tộc |
| `status` | `Status.csv` | Trạng thái |
| `recipe` | `Recipe.csv` | Công thức luyện Cổ |
| `refine_stage` | `RefineStage.csv` | Cấu hình stage luyện Cổ |
| `refine_decision` | `RefineDecision.csv` | Quyết định trong phiên luyện Cổ |
| `shop_item` | `ShopItem.csv` | Cửa hàng |
| `arena_shop_item` | `ArenaShopItem.csv` | Cửa hàng đấu trường |
| `region` | `Region.csv` | Vùng và hub |
| `public_aperture_site` | `PublicApertureSite.csv` | Tọa độ và cấu hình các Tiên Khiếu công cộng tạo sẵn |
| `loot_table` | `LootTable.csv` | Bảng loot |
| `region_loot` | `RegionLootTables.csv` | Quy tắc loot theo vùng |
| `monster` | `Monster.csv` | Quái vật |
| `monster_spawn` | `MonsterSpawn.csv` | Spawn quái |
| `map_cell` | `MapCell.csv` | Ô bản đồ và portal/object |
| `sect_mission` | `SectMission.csv` | Nhiệm vụ môn phái |
| `quest` | `Quest.csv` | Nhiệm vụ |
| `dream_realm` | `DreamRealm.csv` | Kịch bản Mộng Cảnh |

`MonsterSpawn.csv` có các spawn pool của vùng `aperture_interior`, dùng làm quái sinh thái trong map Tiên Khiếu. Import bảng `monster_spawn` sau khi sửa CSV. Mỗi khiếu có bố cục sinh thái ổn định riêng theo `map_instance_id`; thời gian hồi sau khi hạ quái lấy từ `respawn_seconds` của spawn tương ứng. Trạng thái ô bị dọn ở map Tiên Khiếu được Redis scope theo `map_instance_id`; cùng tọa độ trong hai khiếu có chu kỳ hồi độc lập.

Dataset có thể có điều kiện import/export/preflight khác nhau; danh sách runtime ở `/api/admin/data/tables` là nguồn chính xác hơn bảng tóm tắt này.

`PublicApertureSite.csv` dùng `site_key,region_id,x,y,aperture_rank,grade,area_size,dao_marks,is_active`. Mỗi dòng cấu hình một vị trí cố định trong world; `dao_marks` là JSON object. Import tạo cấu hình; worker materialize site, Môn Hộ và số mỏ khởi điểm theo grade (LOW/MID tạo 2, HIGH/EXTREME tạo 3), dùng trọng số Đạo Ngân như khi mở Tiên Khiếu thường. Khi site hết hạn và bị dọn, config được đánh dấu `CONSUMED` để không spawn lại; import lại cùng `site_key` giữ lifecycle state. CSV hiện có 23 seed Động Thiên & Phúc Địa phân bố khắp Ngũ Vực (Nam Cương: Tam Vương, Huyết Hải, Bạch Cốt, Vạn Hoa, Thập Vạn Thú; Trung Châu: Thần Tiêu, Hồ Tiên, Thất Tinh, Lạc Thiên, Vạn Kiếm; Bắc Nguyên: Tuyết Nguyên, Hắc Phàm, Ma Tổ, Vương Đình, Băng Phong; Tây Mạc: Thiên Khanh, Đạo Tặc, Lục Bảo, Xích Nhiễm; Đông Hải: Khí Hải, Bồng Lai, Long Cung, Quy Khư); cần import bảng `public_aperture_site` trong Admin → Dữ liệu để tạo config, sau đó worker mới sinh Môn Hộ.

## Registry và reload

Các registry được nạp khi startup gồm Gu, item, race, constitution, status, monster, zone, quest, loot, recipe và instance. Import Admin có thể reload registry hoặc invalidate cache theo metadata dataset. Không suy ra rằng mọi thay đổi CSV đều có hiệu lực tức thì; kiểm tra kết quả import và restart khi được yêu cầu.

## Effects

Effects được lưu dạng JSON và runtime adapter hỗ trợ theo ngữ cảnh thực thi. Một effect hợp lệ về cấu trúc chưa chắc được mọi gameplay context hỗ trợ. Dùng [danh mục effect theo Item/Gu/Race/Constitution](EFFECTS.md) để xem code hiện có, status consumer và các no-op đã xác minh; dùng effect catalog/validation của backend và Admin preflight khi sửa data.

`Gu.csv` dùng Block v2 envelope. Ví dụ rút gọn về cấu trúc (operation cụ thể vẫn phải được catalog và consumer hỗ trợ):

```json
{"schema_version":2,"blocks":[{"id":"BASIC_HIT","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10,"unit":"FLAT"}]}]}
```

Đây là ví dụ cấu trúc; operation, trigger, target và conditions phải nằm trong allowlist của consumer tương ứng. Item, race, constitution, status và các product như killer move/Gu House có adapter riêng. Khi thêm trigger, operation hay native hook mới, cập nhật `frontend/src/lib/wiki/effect-glossary.ts` và tài liệu API/gameplay nếu phù hợp. Xuân Tiễn Tiên Cổ rank 6 bổ sung `STATUS SLOW` lên địch (20%, 2 lượt); Liệu Quang Giáp Tiên Cổ rank 6 dùng `CLEANSE` lên bản thân. Hai operation này dùng consumer combat/Sát Chiêu hiện có; cập nhật CSV cần import bộ Cổ qua Admin rồi khởi động lại backend để nạp registry.

World `ON_USE` cho phép `stack_policy` trên `STATUS` được executor hỗ trợ (FLYING, HASTE, STEALTH). Status được ghi trong `players.statuses` với thời hạn theo phút game, nhãn nguồn, source instance, effect definition và context `world`. Cùng source/effect/context sẽ refresh; nguồn khác giữ instance riêng mặc định hoặc với `STACK`. `MAX`/`MIN` giữ một cường độ thắng trong cùng effect/context, còn `REPLACE`/`IGNORE` lần lượt thay thế hoặc giữ status đang hoạt động. Policy trên `HEAL` vẫn không được executor thế giới hỗ trợ.

Condition value phụ thuộc field: các field chỉ số nhận số; `ACTOR_RACE` và `TARGET_RACE` nhận race key dạng chuỗi, chỉ hỗ trợ `EQ`/`NE`, và so khớp chính xác. `ACTOR_RACE` là race của tác nhân gây event; `TARGET_RACE` được đánh giá riêng trên từng mục tiêu đã chọn. Thiếu race trong context thì condition không khớp; block không tác động mục tiêu đó. Preflight vẫn từ chối trigger/context không có evaluator tương ứng theo support matrix.

Passive `ALWAYS_ON` có conditions vẫn là projection thuần: `gamefx.Get` đánh giá race, rank, HP% và mana% actor-local có trên player; consumer có event context bổ sung thì gọi `gamefx.GetWithConditions`. Scouting/profile/portal truyền race tác nhân và race mục tiêu theo flow hiện tại. Field event không có trong context của consumer thì block đó không áp dụng; không dùng projection này để tạo effect instance hoặc ghi state. Scalar Gu projection bỏ qua block có condition để tránh áp dụng ngoài context.

Với effect passive `ALWAYS_ON` được chiếu vào scorer Thiên Cơ, `STAT_MOD INVESTIGATION` và `STAT_MOD DISGUISE` là giá trị flat cộng vào input cùng tên trước trọng số Gu hiện có. Các modifier này chỉ đổi điểm scorer/che giấu; không ghi knowledge, không cấp quyền Môn Hộ, và không đổi phạm vi inspect hay chi phí.

Combat `STATUS` Block v2 chụp source Gu instance, block/operation, target, context combat, giá trị và duration khi áp dụng. Cùng source/effect áp lại sẽ refresh snapshot; nguồn khác giữ instance riêng nếu không khai báo policy. `STACK` giữ các nguồn riêng, `MAX`/`MIN` chọn value cao/thấp (hòa chọn duration còn lại dài hơn, hòa tiếp giữ instance hiện tại), `REFRESH` refresh identity hiện tại, `REPLACE` thay status cùng ID trên target, `IGNORE` giữ status cùng ID đã có. Các policy này chỉ được thực thi bởi status consumer combat; context khác phải có executor hỗ trợ riêng.

Race damage/resistance passive được author bằng `UNIQUE` code `RACE_DAMAGE_MODIFIER` hoặc `RACE_RESISTANCE_MODIFIER`, với `params.value` là map từ race key chính xác sang tỷ lệ additive. Ví dụ `{"value":{"human":0.2}}` tăng 20% damage lên mục tiêu race `human` hoặc giảm 20% damage từ tác nhân race `human`, tùy code. Contributions cùng race key cộng dồn; race thiếu/không trùng không tạo factor. Combat breakdown chỉ trả race key và factor khi factor thực sự khác 0. Race bucket độc lập với Path và DaoCombat.

Các đợt effect theo lưu phái thêm block `PATH_SIGNATURE_*` trong `Gu.csv`. Đợt 1 gồm Thiên, Vũ (Space Path), Trụ, Nhân, Khí; đợt 2 gồm Thổ, Kim, Mộc, Hỏa, Băng; đợt 3 gồm Hồn, Độc, Huyết, Lôi, Phong; đợt 4 gồm Trí, Kiếm, Lực, Biến Hóa, Vận; đợt 5 gồm Mộng, Tinh, Vân, Tin Tức, Luật; đợt 6 gồm Không Gian, Tử Vong, Sáng Tạo, Luyện, Quang Minh; đợt 7 gồm Tình Cảm, Hội Họa, Nô, Đao, Thực; đợt 8 gồm Cốt, Ám, Hủy Diệt, Trận, Sát; đợt 9 gồm Ảo Ảnh, Âm, Đạo Tặc, Thủy, Huyền (XUAN) và Điều Tra. Điều Tra dùng `STAT_MOD INVESTIGATION` ở `ALWAYS_ON`, consumer hiện có là scouting/profile; các phái còn lại dùng status combat. Đa số dùng Tiên Cổ rank 6; một số effect rank 7 hoặc 8 do thiếu lựa chọn rank 6 phù hợp. Các trị số là thiết kế của game lấy cảm hứng từ danh sách sát chiêu, không phải cơ chế được xác nhận bởi tài liệu gốc. Chi tiết từng Cổ ở [tài liệu Sát Chiêu](../features/killer-moves-and-gu-houses.md).

## Thay đổi schema

1. Tìm schema file domain trong `backend/internal/db/schema_*.go`.
2. Cập nhật DDL tạo mới và compatibility DDL/backfill cho DB đã tồn tại.
3. Xác nhận FK/index/unique constraints và dữ liệu cũ.
4. Nếu cần rollout dữ liệu thủ công, tạo/điều chỉnh script rõ ràng; `InitSchema` không thay cho migration vận hành.
5. Cập nhật tài liệu và kiểm tra package backend liên quan.

Vị trí ô hiện tại của player được lưu ở `players.x/y`, cùng `current_region` và các trường instance/scope. Không thêm cặp `world_x/world_y` riêng cho cùng vị trí; các handler cần tọa độ phải đọc `x/y`. `schemaPlayers` xóa hai cột world tọa độ cũ trên DB hiện có.

`combat_sessions` cũ có `state` và ràng buộc `UNIQUE(player_id)`. Khi khởi động, `schemaCombat` bổ sung các cột cho phiên chiến đấu hiện tại, đánh dấu phiên kiểu cũ là `abandoned` (snapshot cũ không thể phát lại bằng simulator hiện tại), và bỏ ràng buộc unique cũ. Chỉ phiên `active` mới được giới hạn một phiên mỗi người chơi qua partial index.
