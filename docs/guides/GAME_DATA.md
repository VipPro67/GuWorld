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
| `loot_table` | `LootTable.csv` | Bảng loot |
| `region_loot` | `RegionLootTables.csv` | Quy tắc loot theo vùng |
| `monster` | `Monster.csv` | Quái vật |
| `monster_spawn` | `MonsterSpawn.csv` | Spawn quái |
| `map_cell` | `MapCell.csv` | Ô bản đồ và portal/object |
| `sect_mission` | `SectMission.csv` | Nhiệm vụ môn phái |
| `quest` | `Quest.csv` | Nhiệm vụ |
| `dream_realm` | `DreamRealm.csv` | Kịch bản Mộng Cảnh |

Dataset có thể có điều kiện import/export/preflight khác nhau; danh sách runtime ở `/api/admin/data/tables` là nguồn chính xác hơn bảng tóm tắt này.

## Registry và reload

Các registry được nạp khi startup gồm Gu, item, race, constitution, status, monster, zone, quest, loot, recipe và instance. Import Admin có thể reload registry hoặc invalidate cache theo metadata dataset. Không suy ra rằng mọi thay đổi CSV đều có hiệu lực tức thì; kiểm tra kết quả import và restart khi được yêu cầu.

## Effects

Effects được lưu dạng JSON và runtime adapter hỗ trợ theo ngữ cảnh thực thi. Một effect hợp lệ về cấu trúc chưa chắc được mọi gameplay context hỗ trợ. Dùng effect catalog/validation của backend và Admin preflight; không tự thêm op/trigger rồi giả định simulator sẽ thực thi.

`Gu.csv` dùng Block v2 envelope. Ví dụ rút gọn về cấu trúc (operation cụ thể vẫn phải được catalog và consumer hỗ trợ):

```json
{"schema_version":2,"blocks":[{"id":"BASIC_HIT","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10,"unit":"FLAT"}]}]}
```

Đây là ví dụ cấu trúc; operation, trigger, target và conditions phải nằm trong allowlist của consumer tương ứng. Item, race, constitution, status và các product như killer move/Gu House có adapter riêng. Khi thêm trigger, operation hay native hook mới, cập nhật `frontend/src/lib/wiki/effect-glossary.ts` và tài liệu API/gameplay nếu phù hợp.

## Thay đổi schema

1. Tìm schema file domain trong `backend/internal/db/schema_*.go`.
2. Cập nhật DDL tạo mới và compatibility DDL/backfill cho DB đã tồn tại.
3. Xác nhận FK/index/unique constraints và dữ liệu cũ.
4. Nếu cần rollout dữ liệu thủ công, tạo/điều chỉnh script rõ ràng; `InitSchema` không thay cho migration vận hành.
5. Cập nhật tài liệu và kiểm tra package backend liên quan.
