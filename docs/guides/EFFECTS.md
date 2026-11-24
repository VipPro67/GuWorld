# Danh mục effect trong game data

Đối chiếu snapshot `backend/data/Item.csv`, `Gu.csv`, `Race.csv`, `Constitution.csv` và `Status.csv` với runtime adapters/consumers. Nguồn code hiện được kiểm tra trong `backend/internal/handlers/effect_compatibility.go`, `item_adapter.go`, `item_effects.go`, `gu_effects.go`, `backend/internal/gamefx/fx.go` và `frontend/src/lib/wiki/effect-glossary.ts`.

Năm CSV hiện có 897 item rows, 4713 Gu rows, 18 race rows, 23 constitution rows và 22 status rows. Tổng còn 0 occurrence `UNIQUE`: Item 0, Gu 0, Race 0, Constitution 0, Status 0. Bảng chi tiết effect bên dưới tập trung vào Item, Gu, Race và Constitution; inventory primitive cũng theo dõi Status. Số lần xuất hiện là occurrence trong effect JSON, không phải số lượng nhân vật/item duy nhất.

## Cách đọc trạng thái

- **Hoạt động**: có consumer cụ thể trong code cho data source/context đã nêu. Cũng có thể bị chặn bởi rank, ownership, resource, trigger hoặc điều kiện nghiệp vụ.
- **Một phần/context**: chỉ hoạt động ở consumer/flow chỉ định; không suy rộng sang mọi trigger hay gameplay.
- **Legacy**: tên cũ/compatibility hoặc chỉ còn narrative; xem nội dung từng dòng.
- **Không có tác dụng gameplay**: hiện không tìm thấy consumer thực thi dù code có thể được catalog chấp nhận.
- Đăng ký trong block catalog hoặc pass validation **không chứng minh** effect có tác dụng. Luôn kiểm tra executor cụ thể.

## Quy tắc thực thi theo context

| Data/context | Đường thực thi có hiệu lực | Hạn chế chính |
| --- | --- | --- |
| Item `ON_CONSUME` | Typed `HEAL`, `REQUIRE`, `GRANT_RESOURCE`, `RESTORE_RESOURCE`, `MODIFY_APERTURE`, `RANDOMIZE_TRAIT`, `REPAIR_APERTURE`, `OPEN_REWARD_CONTAINER`, `PLACE_WORLD_OBJECT`; legacy `UNIQUE` chỉ để đọc định nghĩa cũ | SELF, không evaluate conditions; `Item.csv` hiện không còn `UNIQUE`; operation không có executor bị từ chối.
| Gu `ON_CONSUME` / `ON_USE` | Dispatcher evaluate Block conditions, chạy native hook tương thích và `PLACE_WORLD_OBJECT PRIMEVAL_SPRING` khi tiêu hao | Generic operations chưa có executor bị từ chối; world placement bắt buộc SELF và transaction.
| Gu `ON_TICK` | `PERIODIC_RESOURCE` cho regen theo giờ và sản xuất worker tick; native hooks còn dùng cho định nghĩa cũ | Block có conditions chưa được worker/projection đánh giá; subtype/resource phải khớp catalog.
| Gu combat turn-based | Generic operations ở `ON_ACTION`; native combat hook cần đúng domain/trigger | Hook có conditions hiện không được dispatch; status `stack_policy` áp dụng trong combat với provenance theo Gu instance/block/operation/target.
| Gu House combat | Một operation `ON_ACTION`: flat DAMAGE vào TARGET hoặc HEAL/SHIELD/BARRIER vào SELF | Không conditions; action ngoài khuôn bị từ chối.
| World `ON_USE` | Self HEAL HP/MANA hoặc STATUS `FLYING`/`HASTE`/`STEALTH` | Không phải native hook bất kỳ nào cũng chạy được trong world.
| Passive `ALWAYS_ON` | SELF + `STAT_MOD` được projection allowlist hoặc UNIQUE `gamefx` có consumer riêng | `gamefx.Get` chiếu điều kiện actor-local (race, rank, HP%, mana%); scouting/profile/portal truyền thêm target race. Event-specific conditions không có trong context sẽ không áp dụng; catalog stat list rộng hơn projection allowlist.
| `ON_TICK` | Năm native hook hồi/tạo tài nguyên được worker đọc | Không conditions; code khác không tự chạy chỉ vì trigger là ON_TICK.
| Race/Constitution | `gamefx.Get` nhập effects của race + constitution vào player effects | Mỗi field cần consumer gameplay; effect decoded/hiện trên stat chưa chắc bị mọi hệ thống sử dụng.

## Item.csv

| Operation | Occurrences | Ví dụ item ID (tối đa 5) | Tác dụng theo adapter hiện tại | Trạng thái |
| --- | ---: | --- | --- | --- |
| `GRANT_RESOURCE DAO_MARK` | 44 | `dao_material_blade, dao_material_blood, dao_material_bone, dao_material_cloud, dao_material_creation` | Cộng Đạo Ngân theo path/value của item. | Typed |
| `GRANT_RESOURCE HUMAN_QI` | 1 | `HUMAN_QI_PILL` | Cộng Nhân Khí theo giới hạn hiện có. | Typed |
| `HEAL HP` | 8 | `ITEM_HEALING_PELLET_SMALL, ITEM_MEDICINAL_PASTE, ITEM_HEALING_PELLET_MEDIUM, ITEM_BLOOD_GINSENG, ITEM_HEALING_PELLET_LARGE` | Hồi HP theo value FLAT, không vượt effective max HP. | Typed |
| `MODIFY_APERTURE` | 2 | `ITEM_APERTURE_EXPANSION_PILL, ITEM_APERTURE_TIME_PILL` | Cộng diện tích hoặc tốc độ thời gian Tiên Khiếu. | Typed |
| `OPEN_REWARD_CONTAINER` | 2 | `mine_chest, mine_chest_immortal` | Mở đúng loại rương và settle reward trong transaction. | Typed |
| `PLACE_WORLD_OBJECT` | 1 | `spring_egg` | Cấy Phàm Tuyền tại vị trí player nếu placement hợp lệ. | Typed |
| `RANDOMIZE_TRAIT` | 2 | `ITEM_RANDOM_CONSTITUTION, ITEM_RANDOM_RACE` | Chọn race/constitution khác theo registry hiện hành. | Typed |
| `REPAIR_APERTURE` | 2 | `ITEM_IMMORTAL_RECOVERY_PILL, ITEM_IMMORTAL_SPRING_WATER` | Hồi sinh lực Tiên Khiếu; yêu cầu Immortal và aperture. | Typed |
| `REQUIRE IMMORTAL` | 46 | `dao_material_blade, dao_material_blood, dao_material_bone, dao_material_cloud, dao_material_creation` | Chặn người chưa đạt Immortal trước khi áp dụng effect. | Validation only |
| `RESTORE_RESOURCE` | 2 | `EARTH_QI_PILL, HEAVEN_QI_PILL` | Nạp Địa Khí/Thiên Khí theo giới hạn Tiên Khiếu. | Typed |

Tổng 110 operations; `Item.csv` hiện không còn `UNIQUE`. Item conditions chưa được hỗ trợ. `RESTORE_HUNGER` từng chỉ thêm thông báo nên đã được xóa khỏi Item.csv. `REQUIRE IMMORTAL` chỉ validate quyền dùng. Legacy native handler vẫn được giữ cho item definitions cũ; operation mới thiếu executor bị từ chối trước khi áp dụng effect.

## Race.csv và Constitution.csv

Các code dưới đây là effect keys thực tế từ hai CSV; cột mẫu hiển thị tối đa 5 race/constitution keys có khai báo code đó. Race, Constitution và Status đã chuyển các passive map/stat được projection hỗ trợ sang operation `STAT_MOD`; hai passive Gu scalar sai shape đã bị xóa khỏi data vì projection bỏ qua chúng và chưa đủ căn cứ suy ra stat/path dự định. Tổng 139 passive modifier entries đã chuyển, mỗi stat/path là một operation riêng. Các snapshots dưới đây dùng ký pháp rút gọn cho nhóm operation đó.

| Effect key/stat | Race examples | Constitution examples | Ý nghĩa / consumer hiện tại | Trạng thái |
| --- | --- | --- | --- | --- |
| `APERTURE_STRESS` | — | iron_bone_body, blood_fiend_body | Điều chỉnh stress gain khi Break Wall. | Có consumer hiện hành |
| `CC_RESIST` | — | clear_wisdom_heart_body, dream_phantom_body | Tham gia resist check khi áp status trong combat. | Có consumer hiện hành |
| `CULTIVATION_EFFICIENCY` | giant_sun_descendant, ancient_human | — | Tăng wall-progress gain của Break Wall; race/constitution được nhập trực tiếp. | Có consumer hiện hành |
| `STAT_MOD DAO_MARK` | feathered, inkman, merman, snowman, eggman | myriad_gold_brilliant_flower_body, northern_dark_ice_soul_body, pure_dream_reality_seeker_body, solar_sun_moon_body, universe_great_evolution_body | Cộng hệ số theo path vào cùng projection Dao Marks; native `DAO_MARK_MODIFIER` vẫn được đọc cho định nghĩa cũ. | Typed |
| `SET_RULE DISABLE_HP_REGEN` | cuong_thi | — | Đặt cờ chặn hồi HP định kỳ. | Typed |
| `SET_RULE DISABLE_MANA_REGEN` | cuong_thi | — | Đặt cờ chặn hồi Chân Nguyên định kỳ. | Typed |
| `SET_RULE DISABLE_RESOURCE_REGEN` | cuong_thi | — | Đặt cờ chặn hồi tài nguyên theo luật hiện hành. | Typed |
| `SET_RULE DISABLE_SOUL_REGEN` | cuong_thi | — | Đặt cờ chặn hồi Thần Hồn định kỳ. | Typed |
| `HEALING_RECEIVED` | eggman, ghostman | forest_sea_reincarnation_body, spirit_wood_body, night_shade_body, blood_fiend_body | Điều chỉnh lượng hồi ở player-service/combat healing paths hỗ trợ. | Có consumer hiện hành |
| `HP_REGEN` | mushroomman | spirit_wood_body | Hồi HP định kỳ ngoài combat qua player service. | Có consumer hiện hành |
| `SET_RULE IMMORTAL` | cuong_thi | — | Đặt cờ miễn lão hóa cho các flow kiểm tra tuổi thọ. | Typed |
| `SET_RULE IMMUNE_DREAM_STATUS` | GU_FORGET_DREAM | — | Chặn status thuộc Mộng đạo trong combat. | Typed |
| `LUCK` | giant_sun_descendant, ancient_human | — | Bonus Luck trong refining/loot/roll tương ứng. | Có consumer hiện hành |
| `MANA_REGEN` | — | myriad_gold_brilliant_flower_body, northern_dark_ice_soul_body, pure_dream_reality_seeker_body, solar_sun_moon_body, universe_great_evolution_body | Hồi mana định kỳ qua player service. | Có consumer hiện hành |
| `MAX_MANA` | — | myriad_gold_brilliant_flower_body, northern_dark_ice_soul_body, pure_dream_reality_seeker_body, solar_sun_moon_body, universe_great_evolution_body | Cộng effective max mana, ảnh hưởng flow đọc EffectiveMaxMana. | Có consumer hiện hành |
| `MENTAL_RESISTANCE` | yakshaman | ancient_moon_yin_desolate_body, carefree_wisdom_heart_body, great_earth_origin_body, blazing_fiery_body, heavy_rock_body | Tham gia kiểm tra kháng khống chế tinh thần trong simulator. | Có consumer hiện hành |
| `PATH_DAMAGE_MODIFIER` | merman | pure_dream_reality_seeker_body, solar_sun_moon_body, blazing_fiery_body, illusion_light_body, dream_phantom_body | Bonus damage theo path trong combat. | Có consumer hiện hành |
| `PATH_RESISTANCE_MODIFIER` | snowman | northern_dark_ice_soul_body, frost_chilling_body, night_shade_body | Giảm damage theo path trong combat. | Có consumer hiện hành |
| `REFINE_SUCCESS` | dragonman, fur_people, variant_beastman, miniman, yakshaman | blood_fiend_body | Bonus refining khi công thức/preview đọc RefineBonus. | Có consumer hiện hành |
| `SOUL_DEFENSE` | ghostman | night_shade_body | Được chiếu vào player/combatant stats nhưng chưa được công thức damage dùng để giảm sát thương. | Partial: stat được giữ nhưng không giảm damage |
| `SOUL_REGEN` | ghostman | — | Hồi Soul HP định kỳ qua player service. | Có consumer hiện hành |
| `STAT_MODS` | dragonman, feathered, rockman, variant_beastman, eggman | myriad_gold_brilliant_flower_body, universe_great_evolution_body, blazing_glory_lightning_brilliance_body, great_earth_origin_body, great_strength_true_martial_body | Cộng passive stat theo các key hỗ trợ; projection chỉ nhận allowlist stat/subtype/unit, không phải map tùy ý. | Projection có allowlist/context |
| `TENACITY` | — | iron_bone_body | Giải debuff/rút ngắn khống chế theo combat mechanics. | Có consumer hiện hành |

Các race/constitution rows đang dùng đều được gamefx decode; không thấy code hiện có nào trong hai CSV bị drop khi load. `SOUL_DEFENSE` đã chuyển sang `STAT_MOD` và chiếu vào player/combatant aggregate như trước; simulator chưa áp dụng nó khi resolve damage nên vẫn chưa giảm sát thương Hồn. `STAT_MODS` chỉ tác động stat mà `PassiveStatModSupported` cho phép (ví dụ ATTACK, DEFENSE, SPEED, HP, DAO_MARK, PATH_DAMAGE, PATH_RESISTANCE, HP_REGEN, MENTAL_RESISTANCE, LUCK, REFINE_SUCCESS, CULTIVATION_EFFICIENCY, DAO_MARK_GAIN, APERTURE_STRESS, PATH_CONFLICT_REDUCTION, TENACITY, CC_RESIST); subtype chỉ nhận `ADD_FLAT` (hoặc để trống), unit chỉ nhận `FLAT` (hoặc để trống); `DAO_MARK`, `PATH_DAMAGE`, `PATH_RESISTANCE` bắt buộc có path.

## Gu.csv: effect identifiers

Gu.csv currently has 0 `UNIQUE` operations. All former native hooks have been migrated to typed operations or capability grants. Rows marked legacy/compatibility remain only as readers for stored definitions; the current row of the same name uses its typed replacement. The table below includes native hooks and capability identifiers; capability rows may be authored through `GRANT_CAPABILITY`. Counts/Gu IDs come from CSV, while labels/status come from `effect-glossary.ts`; runtime status must be checked against the consumer. `Hoạt động` does not mean every context supports the effect. See the [effect primitive inventory](../architecture/EFFECT_PRIMITIVE_INVENTORY.md).

| Effect code | Trigger trong Gu.csv | Status | Occurrences | Gu ID ví dụ | Mô tả / tình trạng đã ghi nhận |
| --- | --- | --- | ---: | --- | --- |
| `ALL_PEOPLE_UNITED` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_ALL_PEOPLE_UNITED_I` | Nhân Đạo: Hiệu triệu vạn dân đồng lòng, toàn đội nhận Khiên 30% HP và tăng 35% Sát Thương. |
| `ARENA_IDENTITY_MASK` | `CAPABILITY` | Không được hỗ trợ | 0 | — | Đã gỡ khỏi Gu và catalog authoring vì chưa có consumer ẩn danh đấu trường; validator từ chối khai báo mới. |
| `ARMOR_BREAK_HEAVY` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ANCIENT_BODY_1` | Kim Đạo: Đòn đánh xuyên phá hộ giáp, giảm 40% phòng thủ mục tiêu trong 3 lượt. |
| `BEAST_PHANTOM_STRIKE` | `ON_ACTION` | Legacy/compatibility | 0 | — | GU_BEAR_STRENGTH dùng BONUS_STRIKE (45%, ×0.75, sàn 50) với log cấu hình CSV; hook cũ chỉ giữ để đọc definition đã lưu. |
| `BERSERK_STATE` | `COMBAT_PASSIVE` | Một phần/context | 1 | `GU_BERSERK` | `DAMAGE_MODIFIER MISSING_HP` cấu hình phần tăng sát thương `1 + missing HP ratio × 0.8`; native hook còn điều khiển Kiên Cường và miễn nhiễm Choáng/Đóng Băng dưới 30% HP. |
| `BLEED_BURST` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_BLOOD_BLADE` dùng `DETONATE_STATUS BLEED`: 100 sát thương mỗi stack cộng 8% MaxHP, tiêu thụ mọi stack và đánh trực tiếp tại phase hook cũ. |
| `BLINDING_MIRAGE` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu mới dùng `STATUS CONFUSE` + `STATUS BLIND 40` trong 2 lượt. |
| `BLOOD_DEMON_DISINTEGRATION` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_BLOOD_DEMON` dùng `HP_COST CURRENT_HP_PERCENT 0.20` (tối thiểu 10, giữ lại 1 HP), `DAMAGE MULTIPLIER 2.20` và `HEAL LIFESTEAL 50`; hook cũ còn để tương thích dữ liệu đã lưu. |
| `BLOOD_GOD_CHILD` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLOOD_GOD_CHILD` | Huyết Đạo: Huyết phân thân xuất thế trợ chiến, tăng 50% sát thương và nhận 40% Hút Máu. |
| `BLOOD_SACRIFICE` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_BLOOD_ANCESTOR` dùng `HP_COST CURRENT_HP_PERCENT 0.15` tại `PRE_RIDER`; multiplier x2 cũ tác động lên action không có damage nên đã bỏ. |
| `BLOOD_STAINED_ROBE` | `BEFORE_HIT` | Legacy/compatibility | 0 | — | Gu rank 7–8 dùng `MITIGATE_DAMAGE CONVERT_TO_SHIELD 0.4`; native hook chỉ còn hỗ trợ definition cũ. |
| `BONE_SPIKE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BONE_ARMOR_1` | Cốt Đạo: Phóng thích gai xương đâm xuyên huyết nhục mục tiêu, gây Xuất Huyết và Phá Giáp 30%. |
| `BONE_THORNS` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_BONE_ARMOR_1` | Cốt Đạo: Ngưng tụ cốt giáp gai nhọn, tăng 20% Phòng thủ, 35% Phản thương và nhận 20% MaxHP Cốt Giáp. |
| `BORROW_PAST_POWER` | `ON_CAST` | Legacy/compatibility | 0 | — | `GU_PRESENT_DAYS_PAST` dùng `STATUS ATK_UP 40` trên `ON_CAST`/`SELF`; native hook chỉ đọc dữ liệu cũ. |
| `CALCULATED_STRIKE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_WISDOM` | Trí Đạo: Nhìn thấu toàn bộ sơ hở và đường đi của địch, đòn đánh tất trúng không thể né và tăng 35% uy lực. |
| `CAPTURE_BONUS` | `CAPABILITY` | Có consumer hiện hành* | 17 | `GU_BEAST_WHIP_1, GU_CONTROL_CHAIN_1, GU_CONTROL_CHAIN_3, GU_DOG_ENSLAVEMENT, GU_DOMINATE_COMMAND_3` | Cộng thêm tỉ lệ bắt thú; cộng dồn nhiều Cổ, trần 100%, cần Cổ rank không thấp hơn quái. |
| `CHAIN_LIGHTNING` | `ON_ACTION` | Legacy/compatibility | 0 | — | GU_BLUE_DRAGON_1 dùng `CHAIN_DAMAGE` (50% sát thương action, sàn 30, một mục tiêu phụ); native hook chỉ giữ để đọc definition cũ. |
| `CHANGE_FORM_DISGUISE` | `CAPABILITY` | Có tác dụng một phần/context | 1 | `GU_CHANGE_FORM` | Mở khóa đặt Bản Mệnh Cổ giả; muốn ẩn thân vẫn cần Ngụy Trang Khí Tức. |
| `CHAOS_CAVE_DRILL` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_CHAOS_CAVE_DRILL_I` | Vũ Đạo: Mũi khoan không gian xoắn nát hư không, hủy sạch Hộ Thuẫn, xuyên 100% giáp và tăng 80% sát thương. |
| `CHAOS_GAMBLE` | `ON_CAST` | Legacy/compatibility | 0 | — | `GU_LOVE_GU` dùng `OUTCOME_ROULETTE` `sub_type: "CHAOS"`; native hook chỉ còn để đọc definition cũ. |
| `CHARM_GHOST_COME` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `STATUS` (CHARM in 2 turns) với `application_mode=DIRECT`, giữ khả năng áp dụng chắc chắn của hook cũ. |
| `CHAT_CONNECT` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_CONNECTING_HEAVEN` | Cùng Cảm Ứng Thiên Cơ mở kênh chat chung; chỉ cần nằm trong túi. |
| `CHAT_PERCEIVE` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_DIVINE_SENSE` | Cùng Kết Nối Thiên Cơ mở kênh chat chung; thiếu một trong hai thì không gửi/đọc được. |
| `CLEANSE_REGIONAL_WALL` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 0 | — | Hook cũ tương thích dữ liệu; Gu hiện dùng `REVOKE_STATUS` để xóa Áp Chế Giới Vực trước khi tính chỉ số. |
| `CLOUD_MIST` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CLOUD_COVER_SUN_4` | Vân Đạo: Mây mù huyền ảo cuộn quanh, tăng 35% Né tránh và giảm 20% sát thương nhận vào. |
| `DAO_ADAPTIVE_RESIST` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu mới dùng `STATUS ADAPTIVE_RESIST 25` trong 3 lượt; trigger dữ liệu cũ `BEFORE_HIT` không có consumer. |
| `DAO_MARK_MODIFIER` | `ALWAYS_ON` | Có consumer hiện hành* | 1 | `GU_KILL_DAO` | Hệ số Đạo Ngân theo lưu phái; được tổng hợp từ race/constitution/status rồi dùng ở các flow Dao Marks có hỗ trợ. |
| `DARK_CURSE_WEAK` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CONCEAL_CLOAK_1` | Ám Đạo: Khí tức u tối nguyền rủa, làm suy yếu 30% sát thương và ăn mòn 15% Chân Nguyên của địch. |
| `DARK_STEALTH` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CONCEAL_CLOAK_1` | Ám Đạo: Hòa tan vào bóng đêm vô ảnh vô hình, chuẩn bị đòn đánh bạo kích (+40% crit). |
| `DEDUCTION_INSTANT_FINISH` | `ALWAYS_ON` | Legacy/compatibility | 0 | — | Hook tương thích; dữ liệu mới khai báo `DEDUCTION_FINISH_CHANCE`. |
| `DEVOUR_HEAVEN_EARTH` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_FOOD_DAO_DEVOUR_I` dùng `DAMAGE MULTIPLIER 1.5` và `TRANSFER_RESOURCE HP/PERCENT_OF_TARGET_SHIELD` tại `EARLY_RIDER`. |
| `DIMENSION_SLASH` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BOUND_LAND_1` | Không Đạo: Đao kiếm xé rách không gian, chém xuyên phòng tuyến giáng thẳng lên mục tiêu tăng 25% uy lực. |
| `DISGUISE` | `CAPABILITY` | Có consumer hiện hành* | 65 | `GU_ATTITUDE_GU, GU_CHANGE_FORM, GU_COIN_COIN_4, GU_COIN_COIN_5, GU_CONCEAL_CLOAK_1` | Bật ngụy trang: chỉ có thể giả rank thấp hơn, tăng khả năng chống trinh sát. |
| `DREAM_UNRAVEL` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_DREAM_UNRAVEL_5` dùng `RESOURCE_CONVERSION` `sub_type: "DISPEL_TO_RESOURCES"`; native hook chỉ còn để đọc definition cũ. |
| `DRY_WOOD_SPRING` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_DRY_WOOD_SPRING_I` | Mộc Đạo: Cây khô tái sinh nảy lộc, hồi phục 15% MaxHP mỗi hiệp liên tục trong 3 hiệp. |
| `EARTH_GRAVITY` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu.csv dùng `STATUS SLOW` với `phase=PRE_DAMAGE` và message giữ log cũ; hook cũ còn cho dữ liệu đã lưu trước đây. |
| `EARTH_SHIELD` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ANCIENT_WALL_1` | Thổ Đạo: Dẫn động địa khí ngưng tụ 30% MaxHP Thổ Nham Giáp và tăng 25% phòng thủ. |
| `EATING_LOSS_BLESSING` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 3 | `GU_EATING_LOSS_BLESSING_I` | Thực Đạo: Gu.csv dùng COUNTER_ADD BEFORE_HIT tối đa 10 tầng, DAMAGE_MODIFIER COUNTER_COUNT (1.30 + 0.15/tầng) và HEAL COUNTER_PERCENT (5% MaxHP/tầng) ở FINAL_RIDER; native compatibility rider cộng +5 Công/+5 Thủ và ghi log. |
| `EXECUTE_STRIKE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLADE_AURA_1` | Đao Đạo: Khi đối thủ suy yếu dưới 35% HP, chém ra Tuyệt Mệnh Nhất Đao nhân đôi x2.0 uy lực trảm sát. |
| `EXTREME_ICE_CRYSTAL_COFFIN` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_ICE_CRYSTAL_COFFIN_I` | Băng Tuyết Đạo: Quan tài băng tinh trấn áp, Đóng Băng và Câm Lặng hoàn toàn mục tiêu trong 2 hiệp. |
| `FAKE_REGIONAL_WALL` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 1 | `FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY` | PvP: giảm nửa Áp Chế Giới Vực tác động lên địch trong trận; sau trận gán Áp Chế lên đối thủ 15 + 5×rank phút. |
| `FAREWELL_FRIEND_WIND` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_FAREWELL_FRIEND_WIND_I` | Phong Đạo: Gió tiễn bạn xuyên thấu phòng ngự tăng 50% sát thương, nguyền rủa Tống Hữu Chi Phong trừ 5% MaxHP mỗi hiệp trong 4 hiệp. |
| `FATE_VANQUISHED` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `DAMAGE MULTIPLIER 2.2` và `DISPEL PROTECTION`; native hook chỉ đọc dữ liệu cũ. |
| `FIRE_INFERNO` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLAZING_CLAW` | Hỏa Đạo: Ngọn lửa hừng hực thiêu đốt đối phương, mỗi lượt cắn nuốt 8% sinh lực. |
| `FIRST_TURN_EVADE` | `BEFORE_HIT` | Legacy/compatibility | 0 | — | Gu.csv dùng `HIT_RULE FIRST_TURN_EVADE` với target `SELF`; hook cũ còn cho definition lưu trước đây. |
| `FIVE_RESTRICTION_DEEP_LIGHT` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `STATUS` (GU_SEAL in 3 turns) với `application_mode=DIRECT`, giữ khả năng áp dụng chắc chắn của hook cũ. |
| `FORGET_INTENT_INTERRUPT` | `BEFORE_ENEMY_ATTACK` | Legacy/compatibility; xem ghi chú | 1 | `GU_FORGET_INTENT` | Đòn hiện tại vẫn trúng nhưng phong ấn Cổ địch N lượt; vượt số lần thành công tối đa thì tự bị phong ấn. |
| `FORGET_WILL_CONFUSE` | `BEFORE_ENEMY_ATTACK` | Legacy/compatibility | 0 | — | `GU_FORGET_WILL` dùng `TARGET_REDIRECTION` `sub_type: "CONFUSION"`; native hook chỉ còn để đọc definition cũ. |
| `FORMATION_DOMAIN` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu.csv dùng `SHIELD PERCENT 20` và `STATUS DEF_UP 25` trong 3 lượt, đều trên `SELF`; parity kiểm tra khiên và status. |
| `FORM_BEAR` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu mới dùng `STATUS BEAR_FORM 30` + `SHIELD PERCENT 25` trên bản thân. |
| `FULL_SOUL_HEAL` | `ON_CAST` | Legacy/compatibility | 0 | — | Block no-op đã được gỡ khỏi Gu tiêu hao `GU_GUTS`; luồng tiêu hao chỉ thực thi `ON_CONSUME`. |
| `FUTURE_SELF` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_FUTURE_SELF_I` | Trụ Đạo: Mượn sức mạnh từ tương lai đỉnh phong, tăng vọt 80% Tấn Công và 50% Tốc độ trong 3 hiệp. |
| `GREAT_NIGHT` | `ON_BATTLE_START` | Legacy/compatibility | 0 | — | Consumer cũ chỉ chạy ở đầu trận; hai block `ON_ACTION` không có consumer đã được gỡ khỏi Gu.csv và catalog từ chối trigger sai. |
| `GUARANTEED_HIT` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng operation `HIT_RULE GUARANTEED`, bỏ qua né tránh, mù và né lượt đầu. |
| `GU_DETONATION` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_ASCEND_WELL_1` dùng `HP_COST CURRENT_HP_PERCENT 0.05` tại `EARLY_RIDER`, tối thiểu 1 và giữ lại 1 HP. |
| `HEAVEN_NET_RESTRICTION` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_PRIMORDIAL_RESTRICTION_CHAINS_I` | Thiên Đạo: Khôi Khôi Thiên Võng phong tỏa toàn trường, giảm 50% né tránh phe địch, khi thi triển khóa cứng né tránh và làm chậm 40%. |
| `HEAVEN_SUPPRESSION` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu mới dùng `DAMAGE MULTIPLIER 1.35` + `DISPEL BUFFS 2` trên TARGET. |
| `HEAVY_LIFESTEAL` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `HEAL LIFESTEAL` cùng nhóm `DAMAGE`. |
| `HERO_AMONG_MEN` | `ON_ACTION` | Đã chuyển sang operation typed | 0 | `GU_HERO_AMONG_MEN_I` | `DAMAGE_MODIFIER MISSING_HP`: hệ số `1.30 + (1 - HP/MaxHP) × 1.50`, áp dụng cho damage của action và các operation damage cùng block. |
| `HP_REGEN` | `ALWAYS_ON` | Có consumer hiện hành* | 1 | `GU_HEAVENLY_FATE` | Cộng hồi HP ngoài combat từ race/constitution/status và Gu ACTIVE, equipped; lượng thực tế qua player service và bị giới hạn bởi HP tối đa. |
| `HUMAN_WILL` | `COMBAT_PASSIVE` | Legacy/compatibility | 0 | — | `GU_ALL_SPIRITS_PRIMORDIAL_MOTHER_I` dùng `DAMAGE_MODIFIER MISSING_HP` với điều kiện `HP_PERCENT ≤ 0.40` tại `EARLY_RIDER`. |
| `ICE_DEEP_FREEZE` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `STATUS` (FREEZE 1 lượt, `application_mode=DIRECT`) để giữ hành vi áp dụng chắc chắn; hook cũ vẫn tương thích. |
| `ICE_FROSTBITE` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `STATUS` (SLOW_DAMAGE value 35 trong 3 lượt, `application_mode=DIRECT`) để giữ hành vi áp dụng chắc chắn; hook cũ vẫn tương thích. |
| `IGNITE_BURST` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_BLAZING_CLAW` dùng `DAMAGE_MODIFIER TARGET_HAS_STATUS` với `status_id=BURN`; modifier đọc trạng thái trước khi status mới của cùng đòn được áp dụng. |
| `IGNORE_MOVEMENT_RANGE` | `CAPABILITY` | Có consumer hiện hành* | 14 | `GU_CELESTIAL_ESCAPE, GU_DISTANT_DOOR_I, GU_FIXED_IMMORTAL_TRAVEL, GU_FORMATION_TRANSFER_I, GU_PORTAL_LAND_5` | Di chuyển không giới hạn số ô khi trang bị; Cổng không khai range cũng thành vô hạn. |
| `IMMEASURABLE_QI_SEA` | `ON_ACTION` | Legacy/compatibility | 0 | — | GU_QI_SEA_I dùng `DAMAGE MULTIPLIER 1.7`, `TARGET_DAMAGE_MODIFIER REDUCE 0.3` (sàn 0.2), `STATUS SLOW1 30` trong 3 lượt và `log_action` do CSV cấu hình. |
| `IMMORTAL_BINDING_VINE` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_IMMORTAL_BINDING_VINE_I` | Mộc Đạo: Dây leo tiên đạo trói chặt mục tiêu trong 2 hiệp và hút 20% Chân Nguyên. |
| `IMMUNE_DREAM_STATUS` | `ALWAYS_ON` | Legacy/compatibility | 0 | — | Gu.csv dùng `SET_RULE IMMUNE_DREAM_STATUS`; native alias còn hỗ trợ dữ liệu đã lưu cũ. |
| `INFINITE_WIND` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_INFINITE_WIND_I` | Phong Đạo: Gió vô cùng vô tận bùng nổ, tăng 75% sát thương, +60% Thần Tốc và +35% Né Tránh trong 3 hiệp. |
| `INSTANT_WIN` | `ON_BATTLE_START` | Legacy/compatibility | 0 | — | `GU_XUAN` dùng `END_BATTLE FORCE_VICTORY` trên `ON_BATTLE_START`/`ALL_ENEMIES`; native hook chỉ đọc định nghĩa cũ. |
| `JUSTICE_VERDICT` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_COMPASS_SQUARE_1` | Quy Tắc Đạo: Thiết diện vô tư, công lý phán xét trừng phạt mục tiêu tăng thêm 35% sát thương. |
| `LANDSCAPE_AS_BEFORE` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data dùng `SHIELD PERCENT 40` và `CLEANSE DEBUFF`, đều target `SELF`; hook cũ giữ tương thích dữ liệu cũ. |
| `LETHAL_LUCK_WARD` | `BEFORE_HIT` | Legacy/compatibility | 0 | — | Gu data mới dùng `HIT_RULE LETHAL_LUCK_WARD` với target `SELF`; consumer giữ nguyên cơ chế thoát chết hiện hữu. |
| `LIGHTNING_STUN_BURST` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLUE_DRAGON_1` | Lôi Đạo: Giáng cuồng lôi tăng 30% uy lực và có 40% xác suất làm Choáng mục tiêu 1 lượt. |
| `LIGHT_BLINDING_FLASH` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `STATUS` (BLIND value 60 trong 2 lượt, `application_mode=DIRECT`) để giữ hành vi áp dụng chắc chắn; hook cũ vẫn tương thích. |
| `LIGHT_PURIFY` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BEAM_SWORD_1` | Quang Đạo: Ánh sáng ấm áp bao phủ bản thân, hồi phục 25% HP và thanh tẩy khống chế tinh thần. |
| `LIGHT_TIME_FLYING_BLADE` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_LIGHT_TIME_BLADE_I` | Trụ Đạo: Lưỡi đao thời gian phóng ra từ Trường Hà, tăng 65% sát thương, tước 30% Chân Nguyên và làm chậm 50%. |
| `LOOT_PREVIEW` | `CAPABILITY` | Có consumer hiện hành* | 3 | `GU_HEAVEN_SECRET, GU_WISDOM` | Xem trước bảng rơi đồ (vật phẩm, tỉ lệ, số lượng) khi chạm quái. |
| `LUCK_ROULETTE` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_DOG_SHIT_BOOK_1` dùng `OUTCOME_ROULETTE` `sub_type: "LUCK"`; native hook chỉ còn để đọc definition cũ. |
| `MANA_EAT` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu mới dùng `HEAL RESOURCE MANA PERCENT 25` trên SELF khi ra chiêu; BEFORE_HIT không có consumer native. |
| `MANA_STEAL` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu.csv dùng `TRANSFER_RESOURCE PERCENT_OF_TARGET` với resource MANA; hook cũ vẫn giữ cho definition lưu trữ trước đây. |
| `MAN_BEAST_BURIAL` | `ON_CONSUME` | Có consumer hiện hành* | 4 | `GU_MAN_BEAST_LIFE_BURIAL` | Rank 1–4 đang Đỉnh Phong: tăng 1 đại cảnh giới, về Sơ Giai và giảm 10 Chân Nguyên tối đa (sàn 20). |
| `MARKETPLACE_ACCESS` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_CONNECTING_HEAVEN` | Mở Bảo Hoàng Thiên: xem, đăng bán và mua vật phẩm/Cổ. |
| `MENTAL_RESISTANCE` | `ALWAYS_ON` | Có consumer hiện hành* | 1 | `GU_HEART_TRIBULATION` | Được tổng hợp từ race/constitution/status và Gu hợp lệ; simulator dùng khi kiểm tra khống chế tinh thần. |
| `METAL_AEGIS` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ANCIENT_BODY_1` | Kim Đạo: Ngưng tụ kim quang hộ thể, tăng 35% Phòng thủ và giảm 15% sát thương nhận vào. |
| `MYRIAD_DISMANTLE` | `ON_ACTION` | Legacy/compatibility | 0 | — | GU_MYRIAD_DISMANTLE_I dùng DAMAGE MULTIPLIER 1.6, DIRECT_HP SHIELD_STRIP 100%, DISPEL ALL_BUFFS và DEFENSE_PENETRATION 1; log cấu hình trong CSV. |
| `MYRIAD_STAR_FIREFLY` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_MYRIAD_STAR_FIREFLY_I` | Trí Đạo: Đom đóm ánh sao bùng cháy thiêu rụi ý niệm, tăng 60% sát thương và thiêu rụi 40% Chân Nguyên đối phương. |
| `NIRVANA_FIRE` | `ON_DEATH` | Có consumer hiện hành* | 2 | `GU_NIRVANA_FIRE_I` | Viêm Đạo: Khi nhận đòn chí mạng dục hỏa trùng sinh, hồi sinh 40% HP, tẩy uế toàn bộ và nổ hỏa diễm dư chấn thiêu đốt phe địch. |
| `NO_EXTREME_SPACE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_PHANTOM_ORIGIN_I` | Vũ Đạo: Hòa tan thân thể vào tầng lớp không gian vô tận, tăng 50% Né Tránh trong 3 hiệp. |
| `PACK_FRENZY` | `COMBAT_PASSIVE` | Legacy/compatibility | 0 | — | `GU_BEAST_WHIP_1` dùng `DAMAGE_MODIFIER MINION_COUNT`; mỗi minion đồng minh còn sống cộng 15% sát thương action và child operations tại phase trước rider. |
| `PAINTING_IMPRISONMENT` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BRUSH_CLONE_1` | Họa Đạo: Vung bút điểm mặc, nhốt chặt mục tiêu vào trong bức tranh thủy mặc phong tỏa hành động. |
| `PASSION_OVERLOAD` | `COMBAT_PASSIVE` | Legacy/compatibility | 0 | — | `GU_FATE_BOND` dùng `DAMAGE_MODIFIER MISSING_HP` với điều kiện `HP_PERCENT ≤ 0.35` tại `LATE_RIDER`, cộng `CLEANSE STATUS` riêng cho STUN/FREEZE. |
| `PATH_DAMAGE_MODIFIER` | `ALWAYS_ON` | Legacy cho race/constitution | 0 | — | Gu data mới khai báo bonus cùng semantics bằng `STAT_MOD` `PATH_DAMAGE` và `path`; hook map-valued chỉ còn để tương thích race/constitution. |
| `PERFECT_DISGUISE` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_ATTITUDE_GU` | Giả được rank 1–9 (kể cả cao hơn) và giả Bản Mệnh Cổ; hồ sơ hiện dữ liệu giả. |
| `PERM_STEAL_GU` | `ON_BATTLE_START` | Có tác dụng một phần/context | 1 | `GU_FORMLESS_THIEF_9` | PvP thật: cướp vĩnh viễn một Cổ đang trang bị của địch; tỉ lệ ~25% +10%/rank chênh, Cổ Bản Mệnh khó hơn. |
| `PERSIST_STATUS_ON_HIT` | `ON_HIT` | Có tác dụng một phần/context | 1 | `GU_KILL_ENERGY` | Đòn trúng trong PvP thật gán vĩnh viễn một trạng thái lên địch, mỗi trận một lần. |
| `POISON_DETONATE` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_BONE_CLAW_1` dùng `DETONATE_STATUS POISON` (`per_strength: 2.5`, tối thiểu 60 sát thương) tại phase `LATE_RIDER`; hook cũ còn để tương thích dữ liệu đã lưu. |
| `POISON_VOW` | `ON_CONSUME` | Legacy/compatibility; xem ghi chú | 5 | `GU_DEADLY_ARROW_I, GU_POISON_BLOOD_1, GU_POISON_FOG_5, GU_POISON_SNAKE_I, GU_POISON_VOW` | Tên cũ của CONTRACT_OATH: chỉ ghi log kết khế ước, chưa có cơ chế ràng buộc thực tế. |
| `PORTAL_CREATE` | `CAPABILITY` | Có consumer hiện hành* | 6 | `GU_FIXED_IMMORTAL_TRAVEL, GU_FORMATION_TRANSFER_I, GU_PORTAL_LAND_5` | Dùng Cổ Tiêu Hao trong Túi Cổ để mở modal nhập tọa độ; máy chủ kiểm tra tầm theo range hoặc rank Cổ rồi mới dịch chuyển và tiêu hao Cổ. |
| `QI_EXPLOSION` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLAST_BLAST_1` | Khí Đạo: Nén ép thiên địa nguyên khí bùng nổ, sóng khí kình cuộn trào tăng 40% sát thương. |
| `QI_JUE_RETURN_LIFE` | `BEFORE_HIT` | Có consumer hiện hành* | 2 | `GU_QI_JUE_LIFE_I` | Khí Đạo: Mượn Hy Vọng Khí chuyển bại thành thắng, khi máu dưới 30% lập tức hồi 40% HP và thanh tẩy toàn bộ debuff. |
| `QI_WALL` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLAST_BLAST_1` | Khí Đạo: Dựng Nguyên Khí Tường vô hình, cản phá 30% sát thương và tạo 30% MaxHP Khí Giáp. |
| `RAGING_ANGRY_BIRD` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu.csv dùng DAMAGE MULTIPLIER 1.5 kết hợp STATUS BURN ở phase PRE_DAMAGE; hook cũ còn cho dữ liệu đã lưu trước đây. |
| `RANDOM_PORTAL` | `CAPABILITY` | Có consumer hiện hành* | 6 | `GU_RANDOM_TRAVEL_F, GU_RANDOM_TRAVEL_F2, GU_WORLDLY_ESCAPE` | Dùng Cổ Tiêu Hao trong Túi Cổ để xác nhận dịch chuyển đến tọa độ ngẫu nhiên hợp lệ trong tầm range hoặc tầm theo rank Cổ; Cổ chỉ bị tiêu hao khi dịch chuyển thành công. |
| `REBORN` | `ON_DEATH` | Có consumer hiện hành* | 5 | `GU_FORGET_DEFEAT, GU_SPRING_AUTUMN_CICADA` | Xác suất sống lại: tua về Lượt 1 hoặc đứng dậy tại chỗ với % HP, xóa debuff và buff thêm. Mỗi trận tối đa 5 lần hồi sinh tua lại. |
| `RECORD_MESSAGE` | `ON_USE` | Legacy (đã chuyển CSV) | 0 | — | Tám Gu dùng `NOTIFY` với message trong dữ liệu; placeholder `{gu}` được thay bằng tên Cổ. Native hook vẫn đọc định nghĩa cũ. |
| `REFINE_SELF` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_REFINE_SELF_I` | Luyện Đạo: Đỉnh cao Luyện Đạo lấy thân làm phôi, tăng vĩnh viễn 20% Công và 20% Thủ suốt trận chiến. |
| `REMOTE_TRADE` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_CONNECTING_HEAVEN` | Cho phép giao dịch khi hai bên ở xa; cả hai đều phải sở hữu Cổ này. |
| `REVERSE_CALAMITY` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_REVERSE` | Mọi trạng thái xấu, khống chế, Lãng Quên và cướp Cổ nhắm vào chủ đều bị đảo về hư vô; chỉ sát thương đi qua. |
| `RULE_IMPOSITION` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu mới dùng `DISPEL ALL_BUFFS` trên TARGET; native hook còn tương thích với dữ liệu cũ. |
| `SACRIFICE_APTITUDE` | `ON_CONSUME` | Legacy/compatibility | 0 | — | `GU_BLOOD_SKULL` dùng `MODIFY_APTITUDE ADD_FLAT 5`; vẫn đổi loại tư chất, cập nhật giới hạn Chân Nguyên và giảm 20 độ no. |
| `SAT_KHI_GUARANTEE` | `ON_BATTLE_START` | Legacy/compatibility | 0 | — | Gu data mới dùng `HIT_RULE COUNTER_THRESHOLD`; hook này chỉ đọc dữ liệu cũ. |
| `SEIZE_INITIATIVE` | `ON_BATTLE_START` | Legacy/compatibility | 0 | — | Gu data mới dùng `TURN_ORDER_RULE SEIZE_INITIATIVE`, cấu hình `lifespan_ratio` trong `params`. |
| `SONG_OF_DESTINY` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu rank 8–9 hiện cấu hình bằng `DAMAGE`, `STATUS`, `HEAL RESOURCE MANA` target `ALL_ALLIES` và `NOTIFY`; native hook chỉ đọc định nghĩa cũ. |
| `SONG_OF_SEPARATION` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu rank 7–8 dùng `DAMAGE MULTIPLIER`, `DIRECT_HP SHIELD_STRIP`, `DISPEL BUFFS` và `STATUS WEAK`; hook cũ còn để đọc dữ liệu đã lưu trước đây. |
| `SOUL_BEAST_SUMMON` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_SOUL_BEAST_SUMMON_I` | Hồn Đạo: Triệu hoán Thái Cổ Hồn Thú nhập thể hộ vệ, nhận 35% MaxHP Hộ Thuẫn và tăng 30% sát thương. |
| `SOUL_DEFENSE` | `ALWAYS_ON` | Có tác dụng một phần/context | 1 | `GU_NINE_HEAVENS_MIST_CANOPY_I` | Được đọc vào player/combatant stats nhưng hiện chưa có công thức sát thương dùng để giảm damage; chưa tạo giảm thương thực tế. |
| `SOUL_EXPLOSION_DANG_HUN` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_DANG_HUN_EXPLOSION_I` | Hồn Đạo: Chấn động Đãng Hồn Sơn vang vọng, bộc phá sát thương Hồn và Choáng toàn bộ phe địch 1 hiệp. |
| `SOUL_SHOCK` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ANCIENT_MIST_1` | Hồn Đạo: Đòn đánh chấn động thẳng vào hồn phách mục tiêu, bỏ qua Khiên Ảo và gán Hồn Thương. |
| `SOUND_DISRUPT_SILENCE` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `STATUS` (SILENCE 2 lượt, `application_mode=DIRECT`) để giữ hành vi áp dụng chắc chắn; hook cũ vẫn tương thích. |
| `SOUND_RESONANCE` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `STATUS` (SOUL_WOUND value 25 trong 3 lượt, `application_mode=DIRECT`) để giữ hành vi áp dụng chắc chắn; hook cũ vẫn tương thích. |
| `SOVEREIGN_BODY_TRANSFORM` | `ON_CONSUME` | Legacy/compatibility | 0 | — | `GU_SOVEREIGN_FETUS` dùng `GRANT_BODY_MOD SOVEREIGN_IMMORTAL`; body mod vĩnh viễn và từ chối cấp lặp được giữ nguyên. |
| `SPATIAL_SHIFT` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BOUND_LAND_1` | Không Đạo: Bẻ cong không gian bước vào kẽ hở hư không, né tránh 75% đòn tấn công. |
| `STAR_FIRE_RAIN` | `ON_ACTION` | Có consumer hiện hành* | 4 | `GU_STAR_FALL_I` | Tinh Đạo: tăng 30% damage và cộng trực tiếp khiên bằng 15% MaxHP, bỏ qua hệ số attainment/buff khiên chờ. |
| `STAR_THOUGHT_RESTORE` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `HEAL RESOURCE MANA` với unit `PERCENT`, value 30 và target SELF; hook cũ vẫn được giữ để tương thích. |
| `STEAL_LIFE` | `ON_ACTION` | Legacy/compatibility | 0 | — | `GU_THEFT_LIFE_9` dùng `TRANSFER_RESOURCE HP/PERCENT_OF_MAX_HP` (15%, tối thiểu 30) tại `LATE_RIDER`; hook cũ còn để tương thích dữ liệu đã lưu. |
| `STONE_APERTURE` | `ON_CONSUME` | Legacy/compatibility; xem ghi chú | 4 | `GU_GIANT_POTATO_4, GU_GOLDEN_GATE_I, GU_STONE_APERTURE, GU_STONE_GATE_5` | Tên cũ của APERTURE_PETRIFICATION: đẩy lên Đỉnh Phong nhưng hóa đá 12h × 2^(rank−1); không hồi Chân Nguyên, không đột phá trong thời gian đó. |
| `SUMMON_WOLF_FRONTLINE` | `ON_BATTLE_START` | Legacy/compatibility | 0 | — | Gu data mới ghép `SUMMON` actor-stat scaling với `DISPLACE SET_ROW 3`; hook cũ chỉ đọc dữ liệu lưu trước đây. |
| `SUPPRESS_PRIORITY` | `ON_BATTLE_START` | Legacy/compatibility | 0 | — | Gu data mới dùng `TURN_ORDER_RULE SUPPRESS_ENEMY_PRIORITY`; hook cũ còn cho dữ liệu lưu trước đây. |
| `SWORD_ARMOR_PIERCE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_DRAGON_SLAYING_CORE_1` | Kiếm Đạo: Kiếm khí ngưng tụ vô kiên bất tồi, xuyên phá 40% phòng ngự và tăng 25% tỉ lệ bạo kích. |
| `SWORD_DRAGON_TRANSFORM` | `ON_CAST` | Legacy/compatibility | 0 | — | Gu.csv dùng `RESOURCE_CONVERSION CURRENT_HP_TO_SHIELD` và `STATUS ATK_UP`; native hook chỉ còn để đọc dữ liệu cũ. |
| `SWORD_MARK_DEATH` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_SWORD_MARK` | Kiếm Đạo: Khắc sâu kiếm ngân vào hồn phách địch, ăn mòn sinh mệnh mỗi hiệp và trảm sát khi dưới 25% HP. |
| `SWORD_WAVE_TRIPLE` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu rank 6–8 dùng `DAMAGE MULTIPLIER`, `HIT_RULE MULTI_HIT count 3` và `STATUS BLEED` trực tiếp trên mục tiêu chính; hook cũ còn để đọc dữ liệu đã lưu trước đây. |
| `TEMP_STEAL_GU` | `ON_BATTLE_START` | Có tác dụng một phần/context | 1 | `GU_SHADOW_STEAL_8` | PvP thật: đoạt tạm một Cổ để dùng trong trận; tỉ lệ ~40% +5%/rank chênh, hết trận trả lại. |
| `THIRTY_THREE_HEAVEN_LIGHT` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data dùng `DAMAGE MULTIPLIER 1.45`, `CLEANSE DEBUFF` target `ALL_ALLIES` và direct `STATUS BLIND 35` trong 3 lượt trên `ALL_ENEMIES`. |
| `THREE_MEN_MAKE_TIGER` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_THREE_MEN_TIGER_I` | Tín Đạo: Ba người nói thành hổ, hung hổ ảo ảnh vồ nát tâm trí mục tiêu, tăng 50% sát thương, áp đặt Sợ Hãi và Suy Yếu. |
| `THREE_QI_RETURN` | `ON_ACTION` | Legacy/compatibility | 0 | — | GU_THREE_QI_RETURN_I dùng DAMAGE MULTIPLIER 1.85, DEFENSE_PENETRATION 0.5, HEAL HP 30% và HEAL MANA 100%. |
| `TIEP_THIEN_DIA` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CALAMITY_ARMOR_1` | Thiên Đạo: Câu thông ý chí Thiên Đạo, giải trừ toàn bộ khống chế và hồi 25% Chân Nguyên. |
| `TIME_REWIND` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ENDLESS_EONS_CYCLE_I` | Thời Đạo: Nghịch chuyển dòng sông thời gian cục bộ, hồi 35% MaxHP, 40% Chân Nguyên và tiêu trừ khống chế tinh thần. |
| `TOTEM_GOLDEN_BELL` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_GOLDEN_BELL_SHIELD` | Sát Chiêu Đồ Đằng: Chuông vàng viễn cổ buông rủ ngưng tụ hào quang, kích hoạt 35% Phản thương và kháng bạo kích. |
| `TOTEM_KUN_PENG` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_KUN_PENG_TOTEM_I` | Sát Chiêu Đồ Đằng: Côn Bằng tung cánh bạt sơn cửu thiên, tăng 75% sát thương và nhận 50% Thần Tốc. |
| `UNRAVEL_RIDDLE` | `ON_CAST` | Legacy/compatibility | 0 | — | `GU_UNRAVEL_RIDDLE` dùng `DISPEL BUFFS 2` trên ON_CAST/TARGET; phần ngắt chiêu cũ chỉ ghi log nên đã bỏ. |
| `VAN_DAN_DONG_TAM` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ALL_SPIRITS_PRIMORDIAL_MOTHER_I` | Nhân Đạo: Ngưng tụ hồng trần nhân tâm, tạo 35% MaxHP Khiên Nhân Đạo kiên cố hộ thể. |
| `WATER_CLEANSE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CLEANSING_WATER` | Thủy Đạo: Dòng nước tinh khiết bao bọc bản thân, cuốn trôi toàn bộ độc tố, thiêu đốt và xuất huyết. |
| `WATER_MIRROR_SHIELD` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_CLEANSING_WATER` | Thủy Đạo: Ngưng tụ gương nước phản xạ, tạo 20% MaxHP Khiên Ảo và hồi phục 10% HP. |
| `WIND_HASTE_BURST` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu.csv now composes HASTE 40% and EVASION 25% as direct statuses for 3 turns; the second operation suppresses its separate log. |
| `WOOD_BIND` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data mới dùng `STATUS` (BIND 2 lượt, `application_mode=DIRECT`) để giữ hành vi áp dụng chắc chắn; hook cũ vẫn tương thích. |
| `WOOD_REGEN_BURST` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ALL_SPIRIT_IMMORTAL` | Mộc Đạo: Vận khởi sinh cơ, hồi 15% Chân Nguyên và hồi 10% MaxHP mỗi lượt trong 4 lượt. |
| `WORDS_ARE_FRIGHTENING` | `ON_ACTION` | Legacy/compatibility | 0 | — | Gu data dùng `DAMAGE_MODIFIER TARGET_DEBUFF_COUNT` (`1.40 + 0.25/debuff`) cùng `STATUS SILENCE` direct trong 2 lượt. |
| `YAMA_EMPEROR` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_YAMA_EMPEROR_I` | Hồn Đạo: Hóa thân Diêm La Đại Đế, tăng 50% Phòng Ngự và kích hoạt Diêm La Tử cắn nuốt hồn phách địch mỗi hiệp. |
| `YOU_DU_STRENGTH_TIGER` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_YOU_DU_TIGER_I` | Sát Chiêu Đồ Đằng: Dực Hổ gầm vang, tăng 85% sát thương, xé toạc sinh mệnh gây Xuất Huyết nặng và Choáng 1 hiệp. |
## Generic operations trong Gu.csv

Operation counts trong Gu.csv:

| Op | Occurrences | Có adapter | Ghi chú thực thi |
| --- | ---: | --- | --- |
| `CLEANSE` | 15 | Có consumer, theo context | Combat adapter; supports `SELF` and `ALL_ALLIES` in combat. |
| `DISPEL` | 9 | Combat | `BUFFS` removes a configured maximum; `ALL_BUFFS` removes every buff on `TARGET`; `PROTECTION` removes shield and invincibility/last-stand protection. |
| `BONUS_STRIKE` | 1 | Combat | TARGET, xác suất `chance`, sát thương trực tiếp `max(action damage × value, min_damage)` tại phase cũ để giữ thứ tự RNG. |
| `DAMAGE` | 2331 | Có consumer, theo context | Combat ON_ACTION; số liệu/action còn chịu simulator rules. |
| `DAMAGE_MODIFIER` | 12 | Combat | `MISSING_HP`, `TARGET_DEBUFF_COUNT`, `TARGET_HAS_STATUS`, hoặc `MINION_COUNT`; minion-count passive chạy tại pre-rider phase. Action modifiers cần damage trong cùng operation group; passive modifier áp dụng cho mỗi action. |
| `TARGET_DAMAGE_MODIFIER` | 2 | Combat | `REDUCE` trừ value khỏi DamageMultiplier của mục tiêu, tồn tại hết combat và clamp theo `min_multiplier`; chạy trước damage và chỉ log một lần cho cả operation group. |
| `MITIGATE_DAMAGE` | 2 | Combat | `BEFORE_HIT`/`SELF`; giảm incoming damage theo `value` hoặc chuyển phần giảm thành khiên. |
| `DETONATE_STATUS` | 2 | Combat | Tiêu thụ status đã đăng ký rồi gây sát thương trực tiếp theo `max(min_damage, per_stack × stack_count + per_strength × total_strength + max_hp_percent × target.MaxHP)`. `phase` chọn `EARLY_RIDER` hoặc `LATE_RIDER` để giữ thứ tự hook cũ; không hỗ trợ block conditions. |
| `DIRECT_HP` | 13 | Có consumer, theo context | Combat only, specialized damage path. |
| `HEAL` | 353 | Có consumer, theo context | Combat/resource restoration or item adapter limited by context; world only self HP/MANA. |
| `HP_COST` | 3 | Combat | Trừ `max(current HP × value, min_cost)`, nhưng không làm người thi triển thấp hơn `min_remaining`; hỗ trợ `ON_ACTION`/`SELF`, mặc định phase muộn hoặc `EARLY_RIDER`, không hỗ trợ block conditions. |
| `SHIELD` | 915 | Có consumer, theo context | Combat ON_ACTION; context-specific target/turn rules. |
| `STATUS` | 1014 | Có consumer, theo context | Combat status operation; supports `message`, `log_action` and `suppress_log` for direct authored effects. |
| `STAT_MOD` | 2978 | Có consumer, theo context | Combat action buff supported; ALWAYS_ON only through passive projection allowlist, including path-scoped PATH_DAMAGE, HP_REGEN, MENTAL_RESISTANCE and DAO_MARK. |
| `RESOURCE_CONVERSION` | 1 | Combat ON_CAST | Converts a configured fraction of current HP into shield while preserving a minimum HP floor; supports authored transformation logs. |
| `TRANSFER_RESOURCE` | 11 | Combat | `PERCENT_OF_TARGET` chuyển MANA hiện có; `PERCENT_OF_MAX_HP`/HP rút theo MaxHP rồi hồi cho SELF tại `LATE_RIDER`; `PERCENT_OF_TARGET_SHIELD`/HP hấp thụ tỷ lệ khiên hiện tại và hồi lượng đó tại `EARLY_RIDER`. |
| `GRANT_CAPABILITY` | 117 | Có consumer, theo capability | Typed capability declaration for world-facing Gu permissions and bonuses. |
| `ADVANCE_PROGRESSION` | 15 | Gu consume | Advances one subrank while enforcing matching rank, peak-subrank, and petrification guards. |
| `ADVANCE_MAJOR_REALM` | 4 | Gu consume | Forces one major-rank breakthrough for matching Rank 1–4 Gu at peak subrank; resets wall progress and reduces max mana with a floor. |
| `APPLY_STACKING_STATUS` | 1 | Persistent status/PvP | Configures the post-combat hit status, stack cap, duration, cleanse rule and tier scaling. |
| `GRANT_RESOURCE` | 7 | Gu consume | Adds authored lifespan years. |
| `GROWTH_DURATION` | 59 | Growth projection | Sets the hours needed to reach the authored growth caps. |
| `MODIFY_APERTURE` | 13 | Gu consume + item | Gu stress/time-rate subtypes preserve the historical clamps; item allowlist remains separate. |
| `SUMMON` | 13 | Combat | Resolves authored monster template; supports template or caster stat sources, rank/speed overrides, dao scaling and caster clamps. |
| `MODIFY_MAX_RESOURCE` | 4 | Gu consume | Increases maximum SOUL or THOUGHT while preserving legacy initialization defaults. |
| `GRANT_BODY_MOD` | 1 | Gu consume | Grants the permanent SOVEREIGN_IMMORTAL body mod once. |
| `MODIFY_APTITUDE` | 1 | Gu consume | `ADD_FLAT` changes aptitude and recalculates the associated MaxMana delta. |
| `PERIODIC_RESOURCE` | 63 | Growth projection + resource worker | `REGEN_PER_HOUR` uses rank scaling; `GENERATION_WORKER_TICK` follows the 3-hour worker cadence. |
| `DEDUCTION_FINISH_CHANCE` | 2 | Refinement/deduction | The recipe deduction handler chooses the best chance across equipped, fed Gu and makes one random roll after deducting costs. `params` accepts `base_chance`, `per_rank_step`, `min_chance`, and `max_chance`. |
| `RESTORE_RESOURCE` | 6 | Gu consume | Restores SOUL or THOUGHT with the historical resource caps. |
| `HIT_RULE` | 15 | Combat | `GUARANTEED`/`BYPASS_SHIELD` modify outgoing hits; `COUNTER_THRESHOLD` configures battle-start counter-gated hit effects; `FIRST_TURN_EVADE` and `LETHAL_LUCK_WARD` are defensive `BEFORE_HIT`/`SELF` rules. |
| `PETRIFY_APERTURE` | 4 | Gu consume | Petrifies the aperture under the operation's rank and state guards. |
| `NOTIFY` | 15 | Gu ON_USE / ON_ACTION | Trả thông điệp cấu hình từ Gu.csv; `{gu}` chèn tên Cổ, không ghi dữ liệu ngoài message phản hồi. |
| `TURN_ORDER_RULE` | 2 | Combat | `SEIZE_INITIATIVE` accepts `params.lifespan_ratio`; `SUPPRESS_ENEMY_PRIORITY` reuses the priority-suppression consumer. |
| `REVOKE_STATUS` | 1 | Combat build | Removes a validated persistent player status before combat stats are projected. |
| `SET_RULE` | 1 | Passive projection | Projects a boolean rule flag through the `gamefx` allowlist. |
| `UNIQUE` | 23 | Có consumer, theo context | Legacy/native hook; must match trigger/domain and have a real consumer. |
| `PLACE_WORLD_OBJECT` | 3 | Gu consume | Cấy Phàm Tuyền qua transaction-backed world placement. |
| `DISPLACE` | 5 | Combat | Typed displacement with direction, grid target, bounded steps, and battle-start `SET_ROW`. |

Stat_MOD stats đang có trong Gu.csv: `ATTACK` (511), `CC_RESIST` (7), `CLEANSING_POWER` (8), `CRIT_RATE` (4), `CULTIVATION_EFFICIENCY` (5), `DAO_MARK` (1), `DAO_MARK_GAIN` (5), `DEFENSE` (747), `DISGUISE` (65), `HEARING` (1), `HP` (249), `HP_REGEN` (1), `INVESTIGATION` (233), `LUCK` (63), `MANA` (1), `MENTAL_RESISTANCE` (1), `MOVEMENT` (254), `PATH_CONFLICT_REDUCTION` (46), `PATH_RESISTANCE` (5), `REFINE_MANA_TIER` (50), `SOUL_REFINEMENT` (14), `SPEED` (665), `STORAGE` (27), `TENACITY` (8), `TIME_SPEED` (1).

## Các điểm cần coi là không hoạt động hoặc giới hạn

- **Gu `ARENA_IDENTITY_MASK`**: đã gỡ khỏi Gu.csv và catalog CAPABILITY vì chưa có consumer ẩn danh đấu trường; validator từ chối khai báo mới.
- **Race `SOUL_DEFENSE`**: hiện đi qua projection nhưng chưa được dùng trong công thức damage; chỉ số hiển thị không đồng nghĩa giảm damage.
- **Gu `NOTIFY`**: chỉ trả narrative/log cấu hình; không lưu bản ghi hình ảnh/âm thanh. Native `RECORD_MESSAGE` chỉ còn phục vụ tương thích dữ liệu cũ.
- **Gu `POISON_VOW`**: compatibility alias; handler hiện chỉ log kết ước, chưa enforce lời thề.
- Một hook `CAPABILITY` chỉ mở quyền khi đúng feature gọi kiểm tra capability. Định nghĩa trong catalog không tự kích hoạt action.
- `effectcatalog/manifest.go` có thể gắn disposition `supported` hoặc `migrate` phục vụ migration/ownership. Đây không thay thế kiểm tra adapter/runtime consumer.

## Cập nhật danh mục

Khi đổi `Item.csv`, `Gu.csv`, `Race.csv` hoặc `Constitution.csv`, đối chiếu trang này với effect glossary và consumer code. Mã mới phải có: data source, trigger, context/executor, behavior, status (active/partial/legacy/no-op) và test/validation coverage phù hợp. Trigger/op/native hook được định nghĩa trong `frontend/src/lib/wiki/effect-glossary.ts`; behavior runtime nằm ở handler, `gamefx` hoặc simulator.


### Snapshot giá trị trong Race.csv

| Race key | Tên | Effect values đang khai báo |
| --- | --- | --- |
| `cuong_thi` | Cương Thi | `DISABLE_HP_REGEN=true; DISABLE_MANA_REGEN=true; DISABLE_RESOURCE_REGEN=true; DISABLE_SOUL_REGEN=true; IMMORTAL=true` |
| `dragonman` | Long Nhân | `STAT_MOD REFINE_SUCCESS ADD_FLAT=-0.1; STAT_MOD ADD_FLAT {attack:0.1, defense:0.1}` |
| `feathered` | Vũ Dân | `STAT_MOD DAO_MARK ADD_FLAT {space:0.05}; STAT_MOD ADD_FLAT {speed:0.1}` |
| `fur_people` | Người Lông | `STAT_MOD REFINE_SUCCESS ADD_FLAT=0.1` |
| `giant_sun_descendant` | Cự Dương Huyết Mạch | `STAT_MOD CULTIVATION_EFFICIENCY ADD_FLAT=0.05; STAT_MOD LUCK ADD_FLAT=5` |
| `human` | Nhân Tộc | `—` |
| `inkman` | Mặc Nhân | `STAT_MOD DAO_MARK ADD_FLAT {soul:0.05, wisdom:0.08}` |
| `merman` | Nhân Ngư | `STAT_MOD DAO_MARK ADD_FLAT {water:0.1}; STAT_MOD PATH_DAMAGE ADD_FLAT {water:0.05}` |
| `rockman` | Thạch Nhân | `STAT_MOD ADD_FLAT {defense:0.15, speed:-0.1}` |
| `snowman` | Tuyết Dân | `STAT_MOD DAO_MARK ADD_FLAT {ice:0.15}; STAT_MOD PATH_RESISTANCE ADD_FLAT {fire:0.1}` |
| `variant_beastman` | Thú Nhân | `STAT_MOD REFINE_SUCCESS ADD_FLAT=-0.05; STAT_MOD ADD_FLAT {attack:0.08, defense:0.05}` |
| `eggman` | Đản Dân | `STAT_MOD DAO_MARK ADD_FLAT {cloud:0.05, wind:0.05}; STAT_MOD ADD_FLAT {defense:-0.1, speed:0.1}; HEALING_RECEIVED=0.15` |
| `miniman` | Tiểu Nhân | `STAT_MOD DAO_MARK ADD_FLAT {wood:0.1}; STAT_MOD REFINE_SUCCESS ADD_FLAT=0.05; STAT_MOD ADD_FLAT {attack:-0.12}` |
| `mushroomman` | Nấm Nhân | `STAT_MOD DAO_MARK ADD_FLAT {earth:0.1}; STAT_MOD ADD_FLAT {defense:0.1, speed:-0.1}; STAT_MOD HP_REGEN ADD_FLAT=15` |
| `yakshaman` | Dạ Xoa Dân | `STAT_MOD ADD_FLAT {attack:0.12}; STAT_MOD DAO_MARK ADD_FLAT {dark:0.08}; STAT_MOD REFINE_SUCCESS ADD_FLAT=-0.08; STAT_MOD MENTAL_RESISTANCE ADD_FLAT=-0.1` |
| `ghostman` | U Hồn Tộc | `STAT_MOD DAO_MARK ADD_FLAT {soul:0.12}; SOUL_DEFENSE=0.2; SOUL_REGEN=0.2; STAT_MOD ADD_FLAT {hp:-0.1}; HEALING_RECEIVED=-0.15` |
| `ancient_human` | Thái Cổ Nhân Tộc | `STAT_MOD CULTIVATION_EFFICIENCY ADD_FLAT=0.1; STAT_MOD LUCK ADD_FLAT=3; STAT_MOD DAO_MARK ADD_FLAT {human:0.1}; STAT_MOD REFINE_SUCCESS ADD_FLAT=-0.05` |
| `xuan_human` | Huyền Nhân | `STAT_MOD DAO_MARK ADD_FLAT {xuan:0.15}; STAT_MOD PATH_CONFLICT_REDUCTION ADD_FLAT=10` |

### Snapshot giá trị trong Constitution.csv

| Constitution key | Tên | Effect values đang khai báo |
| --- | --- | --- |
| `myriad_gold_brilliant_flower_body` | Vạn Kim Diệu Hoa Thể | `STAT_MOD DAO_MARK ADD_FLAT {metal:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MOD ADD_FLAT {attack:0.3, defense:0.3}` |
| `northern_dark_ice_soul_body` | Bắc Minh Băng Phách Thể | `STAT_MOD DAO_MARK ADD_FLAT {ice:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MOD PATH_RESISTANCE ADD_FLAT {ice:0.1}` |
| `pure_dream_reality_seeker_body` | Thuần Mộng Cầu Chân Thể | `STAT_MOD DAO_MARK ADD_FLAT {dream:0.4}; MANA_REGEN=1.0; MAX_MANA=0.4; STAT_MOD PATH_DAMAGE ADD_FLAT {dream:0.1}` |
| `solar_sun_moon_body` | Thái Nhật Dương Mãng Thể | `STAT_MOD DAO_MARK ADD_FLAT {light:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MOD PATH_DAMAGE ADD_FLAT {light:0.2}` |
| `universe_great_evolution_body` | Vũ Trụ Đại Diễn Thể | `STAT_MOD DAO_MARK ADD_FLAT {space:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MOD ADD_FLAT {attack:0.15, defense:0.15, speed:0.15}` |
| `ancient_moon_yin_desolate_body` | Cổ Nguyệt Âm Hoang Thể | `STAT_MOD DAO_MARK ADD_FLAT {dark:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MOD MENTAL_RESISTANCE ADD_FLAT=0.2` |
| `blazing_glory_lightning_brilliance_body` | Viêm Hoàng Lôi Trạch Thể | `STAT_MOD DAO_MARK ADD_FLAT {lightning:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MOD ADD_FLAT {speed:0.2}` |
| `carefree_wisdom_heart_body` | Tiêu Dao Trí Tâm Thể | `STAT_MOD DAO_MARK ADD_FLAT {wisdom:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MOD MENTAL_RESISTANCE ADD_FLAT=0.3` |
| `forest_sea_reincarnation_body` | Sâm Hải Luân Hồi Thể | `STAT_MOD DAO_MARK ADD_FLAT {water:0.5}; HEALING_RECEIVED=0.5; MANA_REGEN=1.0; MAX_MANA=0.5` |
| `great_earth_origin_body` | Hậu Thổ Nguyên Ương Thể | `STAT_MOD DAO_MARK ADD_FLAT {earth:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MOD MENTAL_RESISTANCE ADD_FLAT=0.45; STAT_MOD ADD_FLAT {defense:0.45}` |
| `great_strength_true_martial_body` | Đại Lực Chân Võ Thể | `STAT_MOD DAO_MARK ADD_FLAT {strength:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MOD ADD_FLAT {attack:0.3, defense:0.3}` |
| `frost_chilling_body` | Hàn Sương Thể | `STAT_MOD DAO_MARK ADD_FLAT {ice:0.25}; STAT_MOD PATH_RESISTANCE ADD_FLAT {ice:0.1}; STAT_MOD ADD_FLAT {speed:-0.1}` |
| `blazing_fiery_body` | Xích Liệt Hoang Thể | `STAT_MOD DAO_MARK ADD_FLAT {fire:0.25}; STAT_MOD PATH_DAMAGE ADD_FLAT {fire:0.1}; STAT_MOD ADD_FLAT {attack:0.1, defense:-0.1}; STAT_MOD MENTAL_RESISTANCE ADD_FLAT=-0.15` |
| `brute_strength_body` | Cự Lực Man Thể | `STAT_MOD DAO_MARK ADD_FLAT {strength:0.25}; STAT_MOD ADD_FLAT {attack:0.15, defense:0.1, speed:-0.1}; MANA_REGEN=-0.2` |
| `heavy_rock_body` | Trọng Nham Huyền Thể | `STAT_MOD DAO_MARK ADD_FLAT {earth:0.25}; STAT_MOD ADD_FLAT {defense:0.25, speed:-0.15}; STAT_MOD MENTAL_RESISTANCE ADD_FLAT=0.15` |
| `spirit_wood_body` | Linh Mộc Dưỡng Sinh Thể | `STAT_MOD DAO_MARK ADD_FLAT {wood:0.25}; HEALING_RECEIVED=0.3; STAT_MOD HP_REGEN ADD_FLAT=20; STAT_MOD ADD_FLAT {attack:-0.1}` |
| `iron_bone_body` | Thiết Cốt Huyền Kim Thể | `STAT_MOD DAO_MARK ADD_FLAT {metal:0.2}; STAT_MOD ADD_FLAT {defense:0.2, speed:-0.1}; STAT_MOD TENACITY ADD_FLAT=0.1; STAT_MOD APERTURE_STRESS ADD_FLAT=0.25` |
| `clear_wisdom_heart_body` | Minh Tuệ Linh Tâm Thể | `STAT_MOD DAO_MARK ADD_FLAT {wisdom:0.25}; STAT_MOD MENTAL_RESISTANCE ADD_FLAT=0.2; STAT_MOD CC_RESIST ADD_FLAT=0.1; STAT_MOD ADD_FLAT {attack:-0.1, hp:-0.1}` |
| `swift_wind_body` | Tật Phong Phi Hành Thể | `STAT_MOD DAO_MARK ADD_FLAT {wind:0.25}; STAT_MOD ADD_FLAT {defense:-0.15, speed:0.2}; MAX_MANA=-0.1` |
| `night_shade_body` | U Dạ Ám Linh Thể | `STAT_MOD DAO_MARK ADD_FLAT {dark:0.25}; SOUL_DEFENSE=0.25; STAT_MOD PATH_RESISTANCE ADD_FLAT {light:-0.1}; HEALING_RECEIVED=-0.15` |
| `illusion_light_body` | Huyễn Quang Lưu Chuyển Thể | `STAT_MOD DAO_MARK ADD_FLAT {light:0.25}; STAT_MOD PATH_DAMAGE ADD_FLAT {light:0.1}; STAT_MOD ADD_FLAT {speed:0.1}; MANA_REGEN=-0.15` |
| `blood_fiend_body` | Huyết Ma Sát Thể | `STAT_MOD DAO_MARK ADD_FLAT {blood:0.25}; STAT_MOD ADD_FLAT {attack:0.15}; HEALING_RECEIVED=0.2; STAT_MOD APERTURE_STRESS ADD_FLAT=0.35; STAT_MOD REFINE_SUCCESS ADD_FLAT=-0.1` |
| `dream_phantom_body` | Huyễn Mộng Linh Thể | `STAT_MOD DAO_MARK ADD_FLAT {dream:0.2}; STAT_MOD PATH_DAMAGE ADD_FLAT {dream:0.05}; STAT_MOD MENTAL_RESISTANCE ADD_FLAT=-0.15; STAT_MOD CC_RESIST ADD_FLAT=-0.1` |

Các snapshot trên chỉ ghi payload CSV (value/path/stat), chưa thay thế quy tắc resolve của `gamefx` hay downstream consumer. Ví dụ, một giá trị được decode và lưu vào stats chưa chắc được công thức combat dùng; xem cột trạng thái phía trên.

### Snapshot Item.csv có khai báo effect

| Item ID | Effect code và value hiện tại |
| --- | --- |
| `ITEM_HEALING_PELLET_SMALL` | `HEAL HP=30` |
| `ITEM_MEDICINAL_PASTE` | `HEAL HP=50` |
| `mine_chest` | `OPEN_MINE_CHEST=1` |
| `mine_chest_immortal` | `OPEN_IMMORTAL_MINE_CHEST=1` |
| `spring_egg` | `PLACE_WORLD_OBJECT PRIMEVAL_SPRING` |
| `ITEM_HEALING_PELLET_MEDIUM` | `HEAL HP=80` |
| `ITEM_BLOOD_GINSENG` | `HEAL HP=150` |
| `ITEM_HEALING_PELLET_LARGE` | `HEAL HP=200` |
| `ITEM_LIFE_SAVING_PILL` | `HEAL HP=500` |
| `ITEM_IMMORTAL_RECOVERY_PILL` | `REPAIR_APERTURE=10000, HEAL HP=1000` |
| `ITEM_IMMORTAL_SPRING_WATER` | `REPAIR_APERTURE=500, HEAL HP=500` |
| `ITEM_RANDOM_CONSTITUTION` | `RANDOMIZE_TRAIT CONSTITUTION` |
| `HUMAN_QI_PILL` | `GRANT_RESOURCE HUMAN_QI=50000` |
| `EARTH_QI_PILL` | `REQUIRE IMMORTAL, RESTORE_RESOURCE EARTH_QI=30000` |
| `HEAVEN_QI_PILL` | `REQUIRE IMMORTAL, RESTORE_RESOURCE HEAVEN_QI=30000` |
| `ITEM_RANDOM_RACE` | `RANDOMIZE_TRAIT RACE` |
| `ITEM_APERTURE_EXPANSION_PILL` | `MODIFY_APERTURE AREA_SIZE=500000` |
| `ITEM_APERTURE_TIME_PILL` | `MODIFY_APERTURE TIME_RATE=2` |
| `dao_material_blade` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_blood` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_bone` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_cloud` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_creation` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_dark` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_death` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_destruction` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_dream` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_earth` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_emotion` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_enslavement` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_fire` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_food` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_formation` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_heaven` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_human` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_ice` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_information` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_investigation` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_killing` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_light` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_lightning` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_luck` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_metal` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_painting` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_phantom` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_poison` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_qi` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_refinement` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_rule` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_soul` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_sound` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_space` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_star` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_strength` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_sword` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_theft` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_time` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_transformation` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_water` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_wind` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_wisdom` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `dao_material_wood` | `REQUIRE IMMORTAL, GRANT_RESOURCE DAO_MARK=1000` |
| `XUAN_RANK1`–`XUAN_RANK8` | Không có item-use effect; đây là thức ăn cho Cổ, xử lý ở luồng cho ăn. |

Payload snapshot không tự khẳng định effect hoạt động; status runtime ở bảng Item.csv phía trên.


Combat `NOTIFY` (`ON_ACTION`/`SELF`) writes one CSV-authored entry to the battle log and can preserve a combined effect message while component operations suppress their individual entries. `HEAL RESOURCE MANA` supports `ALL_ALLIES` and computes percentage values from each recipient’s own maximum mana. A self-target status can also be applied to the selector’s resolved party by authoring `STATUS` with `ALL_ALLIES`; this lets CSV configure the full `SONG_OF_DESTINY` effect without its native UNIQUE hook.
