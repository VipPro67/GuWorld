# Danh mục effect trong game data

Đối chiếu snapshot `backend/data/Item.csv`, `Gu.csv`, `Race.csv`, `Constitution.csv` với runtime adapters/consumers. Nguồn code hiện được kiểm tra trong `backend/internal/handlers/effect_compatibility.go`, `item_adapter.go`, `item_effects.go`, `gu_effects.go`, `backend/internal/gamefx/fx.go` và `frontend/src/lib/wiki/effect-glossary.ts`.

Bốn CSV hiện có 894 item rows, 4713 Gu rows, 18 race rows và 23 constitution rows. Số lần xuất hiện bên dưới là số occurrence trong effect JSON, không phải số lượng nhân vật/item duy nhất.

## Cách đọc trạng thái

- **Hoạt động**: có consumer cụ thể trong code cho data source/context đã nêu. Cũng có thể bị chặn bởi rank, ownership, resource, trigger hoặc điều kiện nghiệp vụ.
- **Một phần/context**: chỉ hoạt động ở consumer/flow chỉ định; không suy rộng sang mọi trigger hay gameplay.
- **Legacy**: tên cũ/compatibility hoặc chỉ còn narrative; xem nội dung từng dòng.
- **Không có tác dụng gameplay**: hiện không tìm thấy consumer thực thi dù code có thể được catalog chấp nhận.
- Đăng ký trong block catalog hoặc pass validation **không chứng minh** effect có tác dụng. Luôn kiểm tra executor cụ thể.

## Quy tắc thực thi theo context

| Data/context | Đường thực thi có hiệu lực | Hạn chế chính |
| --- | --- | --- |
| Item `ON_CONSUME` | `UNIQUE` native hook thuộc `item_use`; `HEAL` HP/MANA có adapter riêng | SELF, không evaluate conditions; một hook chưa map/handler vẫn có thể no-op.
| Gu `ON_CONSUME` / `ON_USE` | Consumable Gu gọi native event handler khớp đúng trigger | Dispatcher chỉ chạy `UNIQUE`, không evaluate generic operations hoặc conditions.
| Gu combat turn-based | Generic operations ở `ON_ACTION`; native combat hook cần đúng domain/trigger | Hook có conditions hiện không được dispatch; từng hook có handler riêng.
| Gu House combat | Một operation `ON_ACTION`: flat DAMAGE vào TARGET hoặc HEAL/SHIELD/BARRIER vào SELF | Không conditions; action ngoài khuôn bị từ chối.
| World `ON_USE` | Self HEAL HP/MANA hoặc STATUS `FLYING`/`HASTE`/`STEALTH` | Không phải native hook bất kỳ nào cũng chạy được trong world.
| Passive `ALWAYS_ON` | SELF + `STAT_MOD` được projection allowlist hoặc UNIQUE `gamefx` có consumer riêng | Không conditions; catalog stat list rộng hơn projection allowlist.
| `ON_TICK` | Năm native hook hồi/tạo tài nguyên được worker đọc | Không conditions; code khác không tự chạy chỉ vì trigger là ON_TICK.
| Race/Constitution | `gamefx.Get` nhập effects của race + constitution vào player effects | Mỗi field cần consumer gameplay; effect decoded/hiện trên stat chưa chắc bị mọi hệ thống sử dụng.

## Item.csv

| Native code | Occurrences | Ví dụ item ID (tối đa 5) | Tác dụng theo handler hiện tại | Trạng thái |
| --- | ---: | --- | --- | --- |
| `ADD_DAO_MARKS` | 44 | `dao_material_blade, dao_material_blood, dao_material_bone, dao_material_cloud, dao_material_creation` | Cộng Đạo Ngân theo path/value của item. | Có handler |
| `HEAL_APERTURE` | 2 | `ITEM_IMMORTAL_RECOVERY_PILL, ITEM_IMMORTAL_SPRING_WATER` | Hồi damage/HP Tiên Khiếu; yêu cầu player Immortal và có aperture. | Có handler |
| `IMMORTAL_ONLY` | 44 | `dao_material_blade, dao_material_blood, dao_material_bone, dao_material_cloud, dao_material_creation` | Điều kiện dùng: chặn người chưa đạt Immortal; không tự cấp chỉ số/reward. | Validation only |
| `INCREASE_APERTURE_AREA` | 1 | `ITEM_APERTURE_EXPANSION_PILL` | Tăng area_size Tiên Khiếu theo value; yêu cầu Immortal/aperture. | Có handler |
| `INCREASE_APERTURE_TIME_RATE` | 1 | `ITEM_APERTURE_TIME_PILL` | Tăng time_rate Tiên Khiếu theo value; yêu cầu Immortal/aperture. | Có handler |
| `INCREASE_HUMAN_QI` | 1 | `HUMAN_QI_PILL` | Cộng Nhân Khí theo handler. | Có handler |
| `OPEN_IMMORTAL_MINE_CHEST` | 1 | `mine_chest_immortal` | Mở rương mỏ tiên và settle reward theo handler. | Có handler |
| `OPEN_MINE_CHEST` | 1 | `mine_chest` | Mở rương mỏ phàm và settle reward theo handler. | Có handler |
| `RANDOM_CONSTITUTION` | 1 | `ITEM_RANDOM_CONSTITUTION` | Đổi sang thể chất ngẫu nhiên khác hiện tại (nếu có hơn một lựa chọn). | Có handler |
| `RANDOM_RACE` | 1 | `ITEM_RANDOM_RACE` | Đổi sang chủng tộc ngẫu nhiên khác hiện tại (nếu có hơn một lựa chọn). | Có handler |
| `RESTORE_HP` | 8 | `ITEM_HEALING_PELLET_SMALL, ITEM_MEDICINAL_PASTE, ITEM_HEALING_PELLET_MEDIUM, ITEM_BLOOD_GINSENG, ITEM_HEALING_PELLET_LARGE` | Hồi HP theo value, không vượt effective max HP. | Có handler |
| `RESTORE_HUNGER` | 8 | `XUAN_RANK1, XUAN_RANK2, XUAN_RANK3, XUAN_RANK4, XUAN_RANK5` | No-op hiện tại: chỉ trả narrative sử dụng item, không sửa Hunger của Gu nào. | No-op |
| `SPRING_EGG_HATCH` | 1 | `spring_egg` | Gọi handler cấy Phàm Tuyền tại vị trí player nếu placement hợp lệ. | Có handler |

Item adapter hiện áp dụng generic `HEAL` và `UNIQUE`; các operation khác rơi vào nhánh không làm gì. `Compiled.Evaluate` đánh giá conditions nếu có, nhưng `effect_compatibility.go` hiện từ chối Item conditions và cả generic `HEAL` trong preflight. Đây là lệch giữa analyzer và runtime cần được đồng bộ; snapshot Item.csv hiện không có block conditions hay generic `HEAL`. `RESTORE_HUNGER` là ví dụ được đăng ký/nhận diện nhưng chỉ thêm thông báo, không thay đổi hunger. `IMMORTAL_ONLY` chỉ validate quyền dùng. Item native handler list và điều kiện nằm trong `backend/internal/handlers/item_effects.go`; codes hiện không xuất hiện trong Item.csv nhưng được adapter hỗ trợ không được tính là data đang dùng.

## Race.csv và Constitution.csv

Các code dưới đây là effect keys thực tế từ hai CSV; cột mẫu hiển thị tối đa 5 race/constitution keys có khai báo code đó. Native `STAT_MODS` và `STAT_MOD` được ghi riêng vì tác dụng phụ thuộc stat/subtype cụ thể.

| Effect key/stat | Race examples | Constitution examples | Ý nghĩa / consumer hiện tại | Trạng thái |
| --- | --- | --- | --- | --- |
| `APERTURE_STRESS` | — | iron_bone_body, blood_fiend_body | Điều chỉnh stress gain khi Break Wall. | Có consumer hiện hành |
| `CC_RESIST` | — | clear_wisdom_heart_body, dream_phantom_body | Tham gia resist check khi áp status trong combat. | Có consumer hiện hành |
| `CULTIVATION_EFFICIENCY` | giant_sun_descendant, ancient_human | — | Tăng wall-progress gain của Break Wall; race/constitution được nhập trực tiếp. | Có consumer hiện hành |
| `DAO_MARK_MODIFIER` | feathered, inkman, merman, snowman, eggman | myriad_gold_brilliant_flower_body, northern_dark_ice_soul_body, pure_dream_reality_seeker_body, solar_sun_moon_body, universe_great_evolution_body | Hệ số Dao Marks theo path ở các flow reward/collection có áp dụng modifier. | Có consumer hiện hành |
| `DISABLE_HP_REGEN` | cuong_thi | — | Chặn scheduled HP regen. | Có consumer hiện hành |
| `DISABLE_MANA_REGEN` | cuong_thi | — | Chặn scheduled mana regen. | Có consumer hiện hành |
| `DISABLE_RESOURCE_REGEN` | cuong_thi | — | Chặn resource regen mà player service gom dưới cờ này. | Có consumer hiện hành |
| `DISABLE_SOUL_REGEN` | cuong_thi | — | Chặn scheduled Soul HP regen. | Có consumer hiện hành |
| `HEALING_RECEIVED` | eggman, ghostman | forest_sea_reincarnation_body, spirit_wood_body, night_shade_body, blood_fiend_body | Điều chỉnh lượng hồi ở player-service/combat healing paths hỗ trợ. | Có consumer hiện hành |
| `HP_REGEN` | mushroomman | spirit_wood_body | Hồi HP định kỳ ngoài combat qua player service. | Có consumer hiện hành |
| `IMMORTAL` | cuong_thi | — | Miễn aging trong các flow kiểm tra aging immunity. | Có consumer hiện hành |
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

Các race/constitution rows đang dùng đều được gamefx decode; không thấy code hiện có nào trong hai CSV bị drop khi load. Tuy nhiên `SOUL_DEFENSE` chưa giảm sát thương. `STAT_MODS` chỉ tác động stat mà `PassiveStatModSupported` cho phép (ví dụ ATTACK, DEFENSE, SPEED, HP, DAO_MARK, PATH_DAMAGE, PATH_RESISTANCE, LUCK, REFINE_SUCCESS, CULTIVATION_EFFICIENCY, DAO_MARK_GAIN, APERTURE_STRESS, PATH_CONFLICT_REDUCTION, TENACITY, CC_RESIST); subtype chỉ nhận `ADD_FLAT` (hoặc để trống), unit chỉ nhận `FLAT` (hoặc để trống); `DAO_MARK`, `PATH_DAMAGE`, `PATH_RESISTANCE` bắt buộc có path.

## Gu.csv: native effect codes

Có 177 native codes phân biệt trong Gu.csv. Danh sách dưới đây lấy count/Gu ID từ CSV và nhãn/mô tả/trạng thái từ `effect-glossary.ts`; code reference của hook có trong `backend/internal/effectcatalog/block_manifest.go` nhưng runtime status phải đọc cùng consumer. `Hoạt động` ở đây không có nghĩa dùng được ở mọi context; xem bảng context ở trên.

| Native code | Trigger trong Gu.csv | Status | Occurrences | Gu ID ví dụ | Mô tả / tình trạng đã ghi nhận |
| --- | --- | --- | ---: | --- | --- |
| `ADD_LIFESPAN_YEARS` | `ON_CONSUME` | Có consumer hiện hành* | 7 | `GU_LIFESPAN_1, GU_LIFESPAN_10, GU_LIFESPAN_100, GU_LIFESPAN_1000, GU_LIFESPAN_5` | Cộng vĩnh viễn số năm thọ nguyên; Cổ tiêu hao biến mất. |
| `ADD_SOUL_MAX` | `ON_CONSUME` | Có consumer hiện hành* | 2 | `GU_ANCIENT_SPEAR_I, GU_GUTS` | Cộng vĩnh viễn Hồn tối đa (nền 10 nếu đang bằng 0). |
| `ADD_THOUGHT_MAX` | `ON_CONSUME` | Có consumer hiện hành* | 2 | `GU_SINGLE_THOUGHT, GU_THOUGHT_MIND` | Cộng vĩnh viễn Niệm Lực tối đa (nền 10 nếu đang bằng 0). |
| `ALL_PEOPLE_UNITED` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_ALL_PEOPLE_UNITED_I` | Nhân Đạo: Hiệu triệu vạn dân đồng lòng, toàn đội nhận Khiên 30% HP và tăng 35% Sát Thương. |
| `ARENA_IDENTITY_MASK` | `CAPABILITY` | Không có consumer gameplay | 1 | `GU_ATTITUDE_GU` | Chưa có hiệu lực: chưa ẩn tên/ẩn danh trong đấu trường. |
| `ARMOR_BREAK_HEAVY` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ANCIENT_BODY_1` | Kim Đạo: Đòn đánh xuyên phá hộ giáp, giảm 40% phòng thủ mục tiêu trong 3 lượt. |
| `BEAST_PHANTOM_STRIKE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BEAR_STRENGTH` | Lực Đạo: Xác suất 45% triệu hồi Cổ Thú Hư Ảnh bồi thêm một đòn bạt sơn bạt hải gây 75% sát thương thêm. |
| `BERSERK_STATE` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_BERSERK` | Lực Đạo: Máu càng thấp chiến ý càng mạnh (tăng tới +80% sát thương và +50% kiên cường), dưới 30% HP miễn nhiễm Choáng và Đóng Băng. |
| `BLEED_BURST` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLOOD_BLADE` | Huyết Đạo: Kích nổ toàn bộ các tầng Xuất Huyết (Bleed) trên người mục tiêu thành sát thương nổ cực mạnh. |
| `BLINDING_MIRAGE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_FORMLESS_EYE_1` | Huyễn Đạo: Hư ảnh trùng trùng điệp điệp, khiến đối thủ rơi vào Mê Loạn và Mù quáng trong 2 lượt. |
| `BLOOD_DEMON_DISINTEGRATION` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLOOD_DEMON` | Huyết Đạo: Tự tổn 20% HP hiện tại giải thể ma huyết, tăng bộc phát 120% sát thương và nhận 50% Hút Máu. |
| `BLOOD_GOD_CHILD` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLOOD_GOD_CHILD` | Huyết Đạo: Huyết phân thân xuất thế trợ chiến, tăng 50% sát thương và nhận 40% Hút Máu. |
| `BLOOD_SACRIFICE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLOOD_ANCESTOR` | Huyết Đạo: Tự trích 15% HP hiến tế để kích phát huyết sát, nhân đôi sát thương chiêu thức (x2.0). |
| `BLOOD_STAINED_ROBE` | `BEFORE_HIT` | Có consumer hiện hành* | 2 | `GU_BLOOD_STAINED_ROBE_I` | Huyết Đạo: Chiến bào đỏ thẫm hấp thụ 40% sát thương nhận vào chuyển hóa thành Huyết Thuẫn hộ thể. |
| `BONE_SPIKE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BONE_ARMOR_1` | Cốt Đạo: Phóng thích gai xương đâm xuyên huyết nhục mục tiêu, gây Xuất Huyết và Phá Giáp 30%. |
| `BONE_THORNS` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_BONE_ARMOR_1` | Cốt Đạo: Ngưng tụ cốt giáp gai nhọn, tăng 20% Phòng thủ, 35% Phản thương và nhận 20% MaxHP Cốt Giáp. |
| `BORROW_PAST_POWER` | `ON_CAST` | Có consumer hiện hành* | 1 | `GU_PRESENT_DAYS_PAST` | Tăng 40% sát thương gây ra trong 3 lượt. |
| `CALCULATED_STRIKE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_WISDOM` | Trí Đạo: Nhìn thấu toàn bộ sơ hở và đường đi của địch, đòn đánh tất trúng không thể né và tăng 35% uy lực. |
| `CAPTURE_BONUS` | `CAPABILITY` | Có consumer hiện hành* | 17 | `GU_BEAST_WHIP_1, GU_CONTROL_CHAIN_1, GU_CONTROL_CHAIN_3, GU_DOG_ENSLAVEMENT, GU_DOMINATE_COMMAND_3` | Cộng thêm tỉ lệ bắt thú; cộng dồn nhiều Cổ, trần 100%, cần Cổ rank không thấp hơn quái. |
| `CC_RESIST` | `ALWAYS_ON` | Có tác dụng một phần/context | 1 | `GU_IMMORTAL_SUPREME_WILL_I` | Giảm tỉ lệ dính debuff/khống chế; trên Cổ phải dùng STAT_MOD CC_RESIST. |
| `CHAIN_LIGHTNING` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLUE_DRAGON_1` | Lôi Đạo: Sấm sét phóng lan truyền dẫn sang mục tiêu lân cận gây thêm sát thương điện giật. |
| `CHANGE_FORM_DISGUISE` | `CAPABILITY` | Có tác dụng một phần/context | 1 | `GU_CHANGE_FORM` | Mở khóa đặt Bản Mệnh Cổ giả; muốn ẩn thân vẫn cần Ngụy Trang Khí Tức. |
| `CHAOS_CAVE_DRILL` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_CHAOS_CAVE_DRILL_I` | Vũ Đạo: Mũi khoan không gian xoắn nát hư không, hủy sạch Hộ Thuẫn, xuyên 100% giáp và tăng 80% sát thương. |
| `CHAOS_GAMBLE` | `ON_CAST` | Có tác dụng một phần/context | 1 | `GU_LOVE_GU` | Ngẫu nhiên một trong ba: ×5 sát thương đòn này, Bất Hoại 1 lượt, hoặc lấy đi 90% HP hiện tại của địch. 'Cái giá' hiện chưa bị trừ. |
| `CHARM_GHOST_COME` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_CHARM_GHOST_I` | Mị Đạo: Tiếng gọi ma mị quyến rũ mê hồn, áp đặt Mê Hoặc khiến đối phương thần hồn điên đảo trong 2 hiệp. |
| `CHAT_CONNECT` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_CONNECTING_HEAVEN` | Cùng Cảm Ứng Thiên Cơ mở kênh chat chung; chỉ cần nằm trong túi. |
| `CHAT_PERCEIVE` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_DIVINE_SENSE` | Cùng Kết Nối Thiên Cơ mở kênh chat chung; thiếu một trong hai thì không gửi/đọc được. |
| `CLEANSE_REGIONAL_WALL` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 1 | `FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY` | Xóa Áp Chế Giới Vực của bản thân ngay đầu trận trước khi tính chỉ số. |
| `CLOUD_MIST` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CLOUD_COVER_SUN_4` | Vân Đạo: Mây mù huyền ảo cuộn quanh, tăng 35% Né tránh và giảm 20% sát thương nhận vào. |
| `CULTIVATION_EFFICIENCY` | `ALWAYS_ON` | Có consumer hiện hành* | 5 | `GU_FATE_BOND, GU_FOUR_FLAVORS_LIQUOR, GU_LIQUOR_WORM, GU_NINE_EYES_LIQUOR, GU_SEVEN_FRAGRANCES_LIQUOR` | Cộng % tiến độ tu luyện. Race/constitution áp dụng qua player effects; nguồn Gu chỉ tính khi Gu ACTIVE, equipped, cùng rank player và chỉ lấy một instance mỗi Gu ID. |
| `DAO_ADAPTIVE_RESIST` | `BEFORE_HIT` | Có consumer hiện hành* | 1 | `GU_BEAST_BEAST_1` | Biến Hóa Đạo: Thích ứng cơ thể với đạo ngân công kích, tăng 25% giảm thương nhận vào. |
| `DAO_MARK_GAIN` | `ALWAYS_ON` | Có tác dụng một phần/context | 5 | `GU_FUTURE_MIRROR_I, GU_HUMAN_FATE, GU_LOVE_SOUL, GU_LOVE_TRIBULATION, GU_TIME_MARK` | Có consumer trong thưởng Đạo Ngân tai kiếp; chỉ áp dụng ở flow phát thưởng đó, không phải mọi nguồn Dao Marks. |
| `DAO_MARK_MODIFIER` | `ALWAYS_ON` | Có consumer hiện hành* | 1 | `GU_KILL_DAO` | Hệ số Đạo Ngân theo lưu phái; được tổng hợp từ race/constitution/status rồi dùng ở các flow Dao Marks có hỗ trợ. |
| `DARK_CURSE_WEAK` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CONCEAL_CLOAK_1` | Ám Đạo: Khí tức u tối nguyền rủa, làm suy yếu 30% sát thương và ăn mòn 15% Chân Nguyên của địch. |
| `DARK_STEALTH` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CONCEAL_CLOAK_1` | Ám Đạo: Hòa tan vào bóng đêm vô ảnh vô hình, chuẩn bị đòn đánh bạo kích (+40% crit). |
| `DECREASE_TIME_SPEED` | `ON_CONSUME` | Có consumer hiện hành* | 2 | `GU_MOON, GU_SUN` | Giảm hệ số tốc độ thời gian, sàn 0.1. Lưu ý: dữ liệu Nhật/Nguyệt Cổ đang khai báo giá trị âm nên thực tế chỉ giảm 0.1. |
| `DEDUCTION_INSTANT_FINISH` | `ALWAYS_ON` | Có consumer hiện hành* | 2 | `GU_WANG_VY, GU_WISDOM` | Mỗi lần thôi diễn có thêm cơ hội hoàn thành ngay; tỉ lệ tăng theo chênh rank Cổ với mục tiêu, vẫn trừ chi phí. |
| `DEVOUR_HEAVEN_EARTH` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_FOOD_DAO_DEVOUR_I` | Thực Đạo: Cắn nuốt thiên địa càn khôn, nuốt 50% Khiên Ảo của địch chuyển hóa thành HP hồi phục bản thân. |
| `DIMENSION_SLASH` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BOUND_LAND_1` | Không Đạo: Đao kiếm xé rách không gian, chém xuyên phòng tuyến giáng thẳng lên mục tiêu tăng 25% uy lực. |
| `DISGUISE` | `CAPABILITY` | Có consumer hiện hành* | 65 | `GU_ATTITUDE_GU, GU_CHANGE_FORM, GU_COIN_COIN_4, GU_COIN_COIN_5, GU_CONCEAL_CLOAK_1` | Bật ngụy trang: chỉ có thể giả rank thấp hơn, tăng khả năng chống trinh sát. |
| `DREAM_UNRAVEL` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_DREAM_UNRAVEL_5` | Mộng Đạo: Thanh tẩy toàn bộ khống chế & mê chướng tâm lý, chuyển hóa thành hồi Chân Nguyên và tạo Mộng Giáp hộ thể. |
| `DRY_WOOD_SPRING` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_DRY_WOOD_SPRING_I` | Mộc Đạo: Cây khô tái sinh nảy lộc, hồi phục 15% MaxHP mỗi hiệp liên tục trong 3 hiệp. |
| `EARTH_GRAVITY` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ANCIENT_WALL_1` | Thổ Đạo: Trọng lực đại địa nghiền ép, làm chậm 40% tốc độ mục tiêu trong 3 lượt. |
| `EARTH_SHIELD` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ANCIENT_WALL_1` | Thổ Đạo: Dẫn động địa khí ngưng tụ 30% MaxHP Thổ Nham Giáp và tăng 25% phòng thủ. |
| `EATING_LOSS_BLESSING` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 3 | `GU_EATING_LOSS_BLESSING_I` | Thực Đạo: Chịu thiệt là phúc, mỗi lần bị đánh tích lũy phúc lực (+5 Công, +5 Thủ, max 10 tầng), khi kích hoạt nổ sát thương và hồi HP. |
| `EXECUTE_STRIKE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLADE_AURA_1` | Đao Đạo: Khi đối thủ suy yếu dưới 35% HP, chém ra Tuyệt Mệnh Nhất Đao nhân đôi x2.0 uy lực trảm sát. |
| `EXTREME_ICE_CRYSTAL_COFFIN` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_ICE_CRYSTAL_COFFIN_I` | Băng Tuyết Đạo: Quan tài băng tinh trấn áp, Đóng Băng và Câm Lặng hoàn toàn mục tiêu trong 2 hiệp. |
| `FAKE_REGIONAL_WALL` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 1 | `FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY` | PvP: giảm nửa Áp Chế Giới Vực tác động lên địch trong trận; sau trận gán Áp Chế lên đối thủ 15 + 5×rank phút. |
| `FAREWELL_FRIEND_WIND` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_FAREWELL_FRIEND_WIND_I` | Phong Đạo: Gió tiễn bạn xuyên thấu phòng ngự tăng 50% sát thương, nguyền rủa Tống Hữu Chi Phong trừ 5% MaxHP mỗi hiệp trong 4 hiệp. |
| `FATE_VANQUISHED` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_FATE_VANQUISHED_I` | Thiên/Thời Đạo: Hồng Liên xé rách số mệnh, phá vỡ hoàn toàn Bất Tử, Bất Hoại và Giáp ảo, giáng đòn x2.2 sát thương. |
| `FIRE_INFERNO` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLAZING_CLAW` | Hỏa Đạo: Ngọn lửa hừng hực thiêu đốt đối phương, mỗi lượt cắn nuốt 8% sinh lực. |
| `FIRST_TURN_EVADE` | `BEFORE_HIT` | Có consumer hiện hành* | 1 | `GU_FIXED_IMMORTAL_TRAVEL` | Né hoàn toàn đòn đánh đầu tiên ở Lượt 1; đòn nhiều hit/AoE sau đó vẫn trúng. Bị Tất Trúng hoặc Cấm Cổ vô hiệu. |
| `FIVE_RESTRICTION_DEEP_LIGHT` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_FIVE_RESTRICTION_LIGHT_I` | Cấm Đạo: Luồng sáng năm màu phong tỏa ngũ hành, phong ấn Cổ trùng (Phong Cổ) của địch trong 3 hiệp. |
| `FORGET_INTENT_INTERRUPT` | `BEFORE_ENEMY_ATTACK` | Legacy/compatibility; xem ghi chú | 1 | `GU_FORGET_INTENT` | Đòn hiện tại vẫn trúng nhưng phong ấn Cổ địch N lượt; vượt số lần thành công tối đa thì tự bị phong ấn. |
| `FORGET_WILL_CONFUSE` | `BEFORE_ENEMY_ATTACK` | Legacy/compatibility; xem ghi chú | 1 | `GU_FORGET_WILL` | Có thể khiến đòn địch đánh lạc sang chính nó hoặc hụt hẳn; có thể phản phệ phong ấn Cổ của mình. |
| `FORMATION_DOMAIN` | `ON_ACTION` | Có consumer hiện hành* | 1 | `FIVE_ELEMENTS_DIVINE_BARRIER_ARRAY` | Trận Đạo: Cổ Trận khởi động bao phủ sàn đấu, tăng 25% phòng thủ và tạo 20% MaxHP Trận Giáp. |
| `FORM_BEAR` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BEAST_BEAST_1` | Biến Hóa Đạo: Hóa thân Viễn Cổ Cự Hùng, tăng 30% Phòng thủ và ngưng tụ 25% MaxHP Hùng Giáp hộ thể. |
| `FULL_SOUL_HEAL` | `ON_CAST` | Có consumer hiện hành* | 1 | `GU_GUTS` | Hồi đầy HP nhưng mất lượt hành động hiện tại. |
| `FUTURE_SELF` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_FUTURE_SELF_I` | Trụ Đạo: Mượn sức mạnh từ tương lai đỉnh phong, tăng vọt 80% Tấn Công và 50% Tốc độ trong 3 hiệp. |
| `GENERATE_IMMORTAL_STONES_TICK` | `ON_TICK` | Có consumer hiện hành* | 7 | `GU_DAO_MARK_WILL_2, GU_GREEN_CLAW_I, GU_HEAVENLY_ESSENCE_LOTUS_I, GU_SHADOW_ARMOR_I` | Cộng Tiên Nguyên Thạch mỗi 3 giờ; không giảm theo rank. |
| `GENERATE_STONES_TICK` | `ON_TICK` | Có consumer hiện hành* | 16 | `GU_CALAMITY_ARMOR_5, GU_COPPER_WALL_4, GU_CREEPING_GATE_4, GU_GIANT_ARMOR_4, GU_GIANT_ARMOR_I` | Cộng Nguyên Thạch mỗi 3 giờ; không giảm theo rank. |
| `GREAT_NIGHT` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_GREAT_NIGHT_I` | Ám Đạo: Màn đêm vĩnh hằng buông xuống, toàn đội Ẩn Thân, +50% Bạo Kích và giảm 40% Né Tránh phe địch. |
| `GROWTH_DURATION_HOURS` | `ALWAYS_ON` | Có consumer hiện hành* | 59 | `GU_ANCIENT_WALL_2, GU_BEAR_STRENGTH, GU_BEAST_BODY_I, GU_BITTER_GATE_1, GU_BITTER_GATE_2` | Đặt số giờ đeo Cổ để đạt chỉ số trưởng thành tối đa; tiến độ tăng tuyến tính. |
| `GUARANTEED_HIT` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_CHAOS_INSECT_KING, GU_MYRIAD_BEAST_I, GU_RED_COLOR_GOD_THUNDER_I` | Đòn không thể bị né hoặc làm hụt, phá cả Né Lượt Đầu và hiệu ứng giảm độ chính xác. |
| `GU_DETONATION` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ASCEND_WELL_1` | Luyện Đạo: Quyết đoán kích nổ Chân Nguyên Cổ trùng, tạo làn sóng xung kích hủy diệt tăng 60% sát thương. |
| `HEAVEN_NET_RESTRICTION` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_PRIMORDIAL_RESTRICTION_CHAINS_I` | Thiên Đạo: Khôi Khôi Thiên Võng phong tỏa toàn trường, giảm 50% né tránh phe địch, khi thi triển khóa cứng né tránh và làm chậm 40%. |
| `HEAVEN_SUPPRESSION` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CALAMITY_ARMOR_1` | Thiên Đạo: Thiên Ý tổn có thừa mà bù không đủ, xóa bỏ 2 buff ngẫu nhiên của địch và tăng 35% uy lực giáng phạt. |
| `HEAVY_LIFESTEAL` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLOOD_BLADE` | Huyết Đạo: Hóa thân huyết hải thao thiết, chuyển hóa 50% sát thương gây ra thành hồi phục sinh mệnh. |
| `HERO_AMONG_MEN` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_HERO_AMONG_MEN_I` | Nhân Đạo: Nghịch cảnh chuyển thương thành dũng, máu hao hụt càng nhiều sát thương bộc phát càng khủng khiếp. |
| `HP_REGEN` | `ALWAYS_ON` | Có consumer hiện hành* | 1 | `GU_HEAVENLY_FATE` | Cộng hồi HP ngoài combat từ race/constitution/status và Gu ACTIVE, equipped; lượng thực tế qua player service và bị giới hạn bởi HP tối đa. |
| `HUMAN_WILL` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_ALL_SPIRITS_PRIMORDIAL_MOTHER_I` | Nhân Đạo: Nghịch cảnh sinh hy vọng, khi dưới 40% HP bùng nổ tăng 50% uy lực chiêu thức. |
| `ICE_DEEP_FREEZE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLUE_BIRD_ARROW_1` | Băng Đạo: Hàn khí ngưng kết đóng băng đối thủ thành tượng tuyết, mất lượt hoàn toàn. |
| `ICE_FROSTBITE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLUE_BIRD_ARROW_1` | Băng Đạo: Băng sương ăn mòn da thịt mục tiêu, vừa cấu rỉa máu vừa làm chậm 35% tốc độ. |
| `IGNITE_BURST` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLAZING_CLAW` | Hỏa Đạo: Nếu mục tiêu đang bị Thiêu Đốt, đòn đánh nổ tung tăng thêm 60% sát thương bạo kích. |
| `IGNORE_MOVEMENT_RANGE` | `CAPABILITY` | Có consumer hiện hành* | 14 | `GU_CELESTIAL_ESCAPE, GU_DISTANT_DOOR_I, GU_FIXED_IMMORTAL_TRAVEL, GU_FORMATION_TRANSFER_I, GU_PORTAL_LAND_5` | Di chuyển không giới hạn số ô khi trang bị; Cổng không khai range cũng thành vô hạn. |
| `IMMEASURABLE_QI_SEA` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_QI_SEA_I` | Khí Đạo: Khí triều mênh mông như biển, tăng 70% sát thương toàn diện, giảm 30% Tấn Công và làm chậm 30% toàn bộ quân địch. |
| `IMMORTAL_BINDING_VINE` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_IMMORTAL_BINDING_VINE_I` | Mộc Đạo: Dây leo tiên đạo trói chặt mục tiêu trong 2 hiệp và hút 20% Chân Nguyên. |
| `IMMUNE_DREAM_STATUS` | `ALWAYS_ON` | Có consumer hiện hành* | 1 | `GU_FORGET_DREAM` | Miễn nhiễm tuyệt đối mọi trạng thái thuộc Mộng đạo. |
| `INCREASE_TIME_SPEED` | `ON_CONSUME` | Có consumer hiện hành* | 4 | `GU_PAST_INCH_3, GU_REBORN_INCH_2, GU_SPRING_ARROW_2, GU_TIME_SPEED_PERM` | Cộng vĩnh viễn vào hệ số tốc độ thời gian của Không Khiếu (mặc định +0.1). |
| `INFINITE_WIND` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_INFINITE_WIND_I` | Phong Đạo: Gió vô cùng vô tận bùng nổ, tăng 75% sát thương, +60% Thần Tốc và +35% Né Tránh trong 3 hiệp. |
| `INSTANT_WIN` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 2 | `GU_XUAN` | Đầu trận hạ toàn bộ phe địch ngay Lượt 0, bỏ qua khiên, miễn nhiễm và Bất Tử. |
| `JUSTICE_VERDICT` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_COMPASS_SQUARE_1` | Quy Tắc Đạo: Thiết diện vô tư, công lý phán xét trừng phạt mục tiêu tăng thêm 35% sát thương. |
| `LANDSCAPE_AS_BEFORE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_LANDSCAPE_AS_BEFORE` | Trụ Đạo: Cảnh cũ người xưa tái hiện, hồi phục 40% MaxHP Giáp và tẩy sạch toàn bộ hiệu ứng suy yếu. |
| `LETHAL_LUCK_WARD` | `BEFORE_HIT` | Có consumer hiện hành* | 1 | `GU_DOG_SHIT_BOOK_1` | Vận Đạo: Khi nhận sát thương kết liễu, có tỉ lệ thoát chết trong gang tấc (HP giữ lại 1 và nhận thêm Chân Nguyên Giáp, 1 lần/trận). |
| `LIGHTNING_STUN_BURST` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLUE_DRAGON_1` | Lôi Đạo: Giáng cuồng lôi tăng 30% uy lực và có 40% xác suất làm Choáng mục tiêu 1 lượt. |
| `LIGHT_BLINDING_FLASH` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BEAM_SWORD_1` | Quang Đạo: Cực quang chói lòa chiếu thẳng vào mắt đối thủ, giảm mạnh 60% độ chính xác. |
| `LIGHT_PURIFY` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BEAM_SWORD_1` | Quang Đạo: Ánh sáng ấm áp bao phủ bản thân, hồi phục 25% HP và thanh tẩy khống chế tinh thần. |
| `LIGHT_TIME_FLYING_BLADE` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_LIGHT_TIME_BLADE_I` | Trụ Đạo: Lưỡi đao thời gian phóng ra từ Trường Hà, tăng 65% sát thương, tước 30% Chân Nguyên và làm chậm 50%. |
| `LOOT_PREVIEW` | `CAPABILITY` | Có consumer hiện hành* | 3 | `GU_HEAVEN_SECRET, GU_WISDOM` | Xem trước bảng rơi đồ (vật phẩm, tỉ lệ, số lượng) khi chạm quái. |
| `LUCK` | `ALWAYS_ON` | Có tác dụng một phần/context | 1 | `GU_HEART_PROMISE` | Cộng Vận Khí từ chủng tộc/trạng thái. Trên Cổ phải dùng STAT_MOD LUCK. |
| `LUCK_ROULETTE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_DOG_SHIT_BOOK_1` | Vận Đạo: Quay thưởng khí vận (Đại Cát ×3.0 dmg + hoàn Chân Nguyên, Trung Cát ×1.5 dmg, Bình Vận, hoặc Đại Hung tự tổn 10% HP và tăng vọt tỉ lệ Đại Cát hiệp sau). |
| `MANA_EAT` | `BEFORE_HIT` | Có consumer hiện hành* | 1 | `GU_ANCIENT_FISH_1` | Thực Đạo: Thôn phệ năng lượng dư thừa, chuyển hóa thành 25% Chân Nguyên tối đa nạp đầy Không Khiếu. |
| `MANA_REGEN_TICK` | `ON_TICK` | Có consumer hiện hành* | 28 | `GU_CALAMITY_ARMOR_5, GU_COPPER_WALL_4, GU_CREATION_ORIGIN, GU_CREEPING_GATE_4, GU_DAO_MARK_WILL_2` | Hồi thêm Chân Nguyên mỗi giờ (mặc định 0.5), giảm theo chênh rank Cổ so với người chơi. |
| `MANA_STEAL` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_COIN_ATTACK_1` | Trộm Đạo: Diệu thủ không không, đánh cắp 25% Chân Nguyên hiện có của mục tiêu chuyển hóa cho bản thân. |
| `MAN_BEAST_BURIAL` | `ON_CONSUME` | Có consumer hiện hành* | 4 | `GU_MAN_BEAST_LIFE_BURIAL` | Rank 1–4 đang Đỉnh Phong: tăng 1 đại cảnh giới, về Sơ Giai và giảm 10 Chân Nguyên tối đa (sàn 20). |
| `MARKETPLACE_ACCESS` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_CONNECTING_HEAVEN` | Mở Bảo Hoàng Thiên: xem, đăng bán và mua vật phẩm/Cổ. |
| `MENTAL_RESISTANCE` | `ALWAYS_ON` | Có consumer hiện hành* | 1 | `GU_HEART_TRIBULATION` | Được tổng hợp từ race/constitution/status và Gu hợp lệ; simulator dùng khi kiểm tra khống chế tinh thần. |
| `METAL_AEGIS` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ANCIENT_BODY_1` | Kim Đạo: Ngưng tụ kim quang hộ thể, tăng 35% Phòng thủ và giảm 15% sát thương nhận vào. |
| `MYRIAD_DISMANTLE` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_MYRIAD_DISMANTLE_I` | Luật Đạo: Vạn vật quy nhất tiêu giải pháp tắc, hủy toàn bộ Hộ Thuẫn, xóa sạch mọi Buff và đánh xuyên 100% phòng ngự. |
| `MYRIAD_STAR_FIREFLY` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_MYRIAD_STAR_FIREFLY_I` | Trí Đạo: Đom đóm ánh sao bùng cháy thiêu rụi ý niệm, tăng 60% sát thương và thiêu rụi 40% Chân Nguyên đối phương. |
| `NIRVANA_FIRE` | `ON_DEATH` | Có consumer hiện hành* | 2 | `GU_NIRVANA_FIRE_I` | Viêm Đạo: Khi nhận đòn chí mạng dục hỏa trùng sinh, hồi sinh 40% HP, tẩy uế toàn bộ và nổ hỏa diễm dư chấn thiêu đốt phe địch. |
| `NO_EXTREME_SPACE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_PHANTOM_ORIGIN_I` | Vũ Đạo: Hòa tan thân thể vào tầng lớp không gian vô tận, tăng 50% Né Tránh trong 3 hiệp. |
| `PACK_FRENZY` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_BEAST_WHIP_1` | Nô Đạo: Mỗi dị thú triệu hồi trên sân tăng thêm 15% uy lực sát thương hiệp đồng tác chiến. |
| `PAINTING_IMPRISONMENT` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BRUSH_CLONE_1` | Họa Đạo: Vung bút điểm mặc, nhốt chặt mục tiêu vào trong bức tranh thủy mặc phong tỏa hành động. |
| `PASSION_OVERLOAD` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_FATE_BOND` | Tình Đạo: Nghịch cảnh sinh tình, khi dưới 35% HP bộc phát 50% uy lực và kháng khống chế. |
| `PATH_DAMAGE_MODIFIER` | `ALWAYS_ON` | Có consumer hiện hành* | 5 | `GU_HEAVEN_FIST_3, GU_HEAVEN_FIST_5, GU_KILL_HEART, GU_KILL_HEAVEN, GU_ROYAL_SWORD_5` | Cộng bonus sát thương theo path trong combat; nguồn có thể là race, constitution hoặc status. |
| `PATH_RESISTANCE_MODIFIER` | `ALWAYS_ON` | Có consumer hiện hành* | 1 | `GU_FATE_TRIBULATION` | Giảm sát thương nhận theo path ở combat; nguồn có thể là race, constitution hoặc status. Gu dùng STAT_MOD PATH_RESISTANCE. |
| `PERFECT_DISGUISE` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_ATTITUDE_GU` | Giả được rank 1–9 (kể cả cao hơn) và giả Bản Mệnh Cổ; hồ sơ hiện dữ liệu giả. |
| `PERM_STEAL_GU` | `ON_BATTLE_START` | Có tác dụng một phần/context | 1 | `GU_FORMLESS_THIEF_9` | PvP thật: cướp vĩnh viễn một Cổ đang trang bị của địch; tỉ lệ ~25% +10%/rank chênh, Cổ Bản Mệnh khó hơn. |
| `PERSIST_STATUS_ON_HIT` | `ON_HIT` | Có tác dụng một phần/context | 1 | `GU_KILL_ENERGY` | Đòn trúng trong PvP thật gán vĩnh viễn một trạng thái lên địch, mỗi trận một lần. |
| `POEM_INTENT` | `ON_TURN_START` | Legacy/compatibility; xem ghi chú | 1 | `GU_POEM_IMMORTAL` | Mỗi lượt +1 Thi Ý và giảm 40% sát thương nhận; 3 tầng xuyên 30% khiên mỗi lượt, 10 tầng phá sạch khiên địch. |
| `POISON_DETONATE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BONE_CLAW_1` | Độc Đạo: Kích nổ toàn bộ các tầng kịch độc trên người đối thủ thành sát thương độc bạo liệt. |
| `POISON_VOW` | `ON_CONSUME` | Legacy/compatibility; xem ghi chú | 5 | `GU_DEADLY_ARROW_I, GU_POISON_BLOOD_1, GU_POISON_FOG_5, GU_POISON_SNAKE_I, GU_POISON_VOW` | Tên cũ của CONTRACT_OATH: chỉ ghi log kết khế ước, chưa có cơ chế ràng buộc thực tế. |
| `PORTAL_CREATE` | `CAPABILITY` | Có consumer hiện hành* | 6 | `GU_FIXED_IMMORTAL_TRAVEL, GU_FORMATION_TRANSFER_I, GU_PORTAL_LAND_5` | Dùng Cổ Tiêu Hao trong Túi Cổ để mở modal nhập tọa độ; máy chủ kiểm tra tầm theo range hoặc rank Cổ rồi mới dịch chuyển và tiêu hao Cổ. |
| `QI_EXPLOSION` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLAST_BLAST_1` | Khí Đạo: Nén ép thiên địa nguyên khí bùng nổ, sóng khí kình cuộn trào tăng 40% sát thương. |
| `QI_JUE_RETURN_LIFE` | `BEFORE_HIT` | Có consumer hiện hành* | 2 | `GU_QI_JUE_LIFE_I` | Khí Đạo: Mượn Hy Vọng Khí chuyển bại thành thắng, khi máu dưới 30% lập tức hồi 40% HP và thanh tẩy toàn bộ debuff. |
| `QI_WALL` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BLAST_BLAST_1` | Khí Đạo: Dựng Nguyên Khí Tường vô hình, cản phá 30% sát thương và tạo 30% MaxHP Khí Giáp. |
| `RAGING_ANGRY_BIRD` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_RAGING_BIRD_I` | Viêm Đạo: Hỏa điểu phẫn nộ phát nổ kinh hoàng, tăng 50% sát thương và gây Thiêu Đốt nặng 3 hiệp. |
| `RANDOM_PORTAL` | `CAPABILITY` | Có consumer hiện hành* | 6 | `GU_RANDOM_TRAVEL_F, GU_RANDOM_TRAVEL_F2, GU_WORLDLY_ESCAPE` | Dùng Cổ Tiêu Hao trong Túi Cổ để xác nhận dịch chuyển đến tọa độ ngẫu nhiên hợp lệ trong tầm range hoặc tầm theo rank Cổ; Cổ chỉ bị tiêu hao khi dịch chuyển thành công. |
| `RANK_PROGRESS` | `ON_CONSUME` | Có consumer hiện hành* | 15 | `GU_BRONZE_RELIC, GU_MIND_HERO_1, GU_MIND_HERO_5, GU_MIND_HERO_I, GU_PURPLE_CRYSTAL_RELIC` | Tăng 1 tiểu cảnh giới khi rank Cổ trùng rank người chơi và chưa Đỉnh Phong; bị chặn khi Không Khiếu hóa đá. |
| `REBORN` | `ON_DEATH` | Có consumer hiện hành* | 5 | `GU_FORGET_DEFEAT, GU_SPRING_AUTUMN_CICADA` | Xác suất sống lại: tua về Lượt 1 hoặc đứng dậy tại chỗ với % HP, xóa debuff và buff thêm. Mỗi trận tối đa 5 lần hồi sinh tua lại. |
| `RECORD_MESSAGE` | `ON_USE` | Legacy/compatibility; xem ghi chú | 8 | `GU_INFO_SHADOW_2, GU_INFO_SHADOW_5, GU_INFO_SHADOW_I, GU_INFO_WORM_1, GU_MIND_SOUND_I` | Tên cũ của RECORD: chỉ ghi log mô tả, không lưu dữ liệu hình ảnh/âm thanh. |
| `REDUCE_STRESS` | `ON_CONSUME` | Có consumer hiện hành* | 5 | `GU_CROWD_RELIC_I, GU_RELIC_UNION_1, GU_SCHOLAR_UNION_1, GU_SCHOLAR_UNION_5, GU_XAM_XAM` | Giảm Áp Lực Không Khiếu (mặc định 5), sàn 0. |
| `REFINE_SELF` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_REFINE_SELF_I` | Luyện Đạo: Đỉnh cao Luyện Đạo lấy thân làm phôi, tăng vĩnh viễn 20% Công và 20% Thủ suốt trận chiến. |
| `REMOTE_TRADE` | `CAPABILITY` | Có consumer hiện hành* | 1 | `GU_CONNECTING_HEAVEN` | Cho phép giao dịch khi hai bên ở xa; cả hai đều phải sở hữu Cổ này. |
| `RESTORE_MENTAL` | `ON_CONSUME` | Có consumer hiện hành* | 3 | `GU_ATTITUDE_ARMOR_5, GU_DIAMOND_THOUGHT_I, GU_THOUGHT_RECOVERY` | Hồi Niệm Lực (mặc định đầy), không vượt mức tối đa. |
| `RESTORE_SOUL_HP` | `ON_CONSUME` | Có consumer hiện hành* | 3 | `GU_CRYING_LAMP_3, GU_CRYING_LAMP_I, GU_SOUL_RESTORE` | Hồi Hồn HP (mặc định 10), không vượt Hồn tối đa. |
| `REVERSE_CALAMITY` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_REVERSE` | Mọi trạng thái xấu, khống chế, Lãng Quên và cướp Cổ nhắm vào chủ đều bị đảo về hư vô; chỉ sát thương đi qua. |
| `RULE_IMPOSITION` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_COMPASS_SQUARE_1` | Quy Tắc Đạo: Vô quy củ bất thành phương viên, thiết lập điều luật chiến trường xóa sạch toàn bộ Buff có lợi của địch. |
| `SACRIFICE_APTITUDE` | `ON_CONSUME` | Có consumer hiện hành* | 1 | `GU_BLOOD_SKULL` | Tăng tư chất Không Khiếu (mặc định 5, trần 99) và đổi loại tư chất; Cổ không mất, chỉ giảm 20 độ no. |
| `SAT_KHI_GUARANTEE` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 1 | `GU_ALLKILL_COLLAR` | Khi Sát Khí trên 10 tầng: xuyên khiên, không thể bị né, phá cả Bất Hoại lẫn Bất Tử. |
| `SEIZE_INITIATIVE` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 1 | `GU_LET_YOU_GO_FIRST` | Lượt 1 được đi trước nếu thọ nguyên địch chính không vượt quá lifespan_ratio lần thọ nguyên của mình (mặc định 3). |
| `SONG_OF_DESTINY` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_DESTINY_SONG_I` | Âm/Vận Đạo: Khúc ca vận mệnh chấn động, buff toàn đội +40% sát thương và hồi 20% chân nguyên, giảm 30% sát thương quân địch. |
| `SONG_OF_SEPARATION` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_SEPARATION_SONG_I` | Âm Đạo: Khúc ca ly tán vạn vật, phá hủy 2 buff, bóc 30% giáp ảo, tăng 65% sát thương và gây Suy Yếu 25%. |
| `SOUL_BEAST_SUMMON` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_SOUL_BEAST_SUMMON_I` | Hồn Đạo: Triệu hoán Thái Cổ Hồn Thú nhập thể hộ vệ, nhận 35% MaxHP Hộ Thuẫn và tăng 30% sát thương. |
| `SOUL_DEFENSE` | `ALWAYS_ON` | Có tác dụng một phần/context | 1 | `GU_NINE_HEAVENS_MIST_CANOPY_I` | Được đọc vào player/combatant stats nhưng hiện chưa có công thức sát thương dùng để giảm damage; chưa tạo giảm thương thực tế. |
| `SOUL_EXPLOSION_DANG_HUN` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_DANG_HUN_EXPLOSION_I` | Hồn Đạo: Chấn động Đãng Hồn Sơn vang vọng, bộc phá sát thương Hồn và Choáng toàn bộ phe địch 1 hiệp. |
| `SOUL_REGEN_TICK` | `ON_TICK` | Có consumer hiện hành* | 4 | `GU_CRYING_SPEAR_3, GU_CRYING_WALL_I, GU_SOUL_NOURISH, GU_SPIRIT_SHIELD_I` | Hồi thêm Hồn HP mỗi giờ (mặc định 1), không vượt Hồn tối đa. |
| `SOUL_SHOCK` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ANCIENT_MIST_1` | Hồn Đạo: Đòn đánh chấn động thẳng vào hồn phách mục tiêu, bỏ qua Khiên Ảo và gán Hồn Thương. |
| `SOUND_DISRUPT_SILENCE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ALERT_BELL` | Âm Đạo: Sóng âm chói tai đâm xuyên màng nhĩ, khóa kỹ năng chỉ có thể đánh thường trong 2 lượt. |
| `SOUND_RESONANCE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ALERT_BELL` | Âm Đạo: Dao động âm ba cộng hưởng nội tạng, bỏ qua Khiên Ảo và gán Hồn Thương trực tiếp. |
| `SOVEREIGN_BODY_TRANSFORM` | `ON_CONSUME` | Có consumer hiện hành* | 1 | `GU_SOVEREIGN_FETUS` | Vĩnh viễn miễn nhiễm xung đột Đạo Ngân; chỉ dùng được một lần. |
| `SPATIAL_SHIFT` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BOUND_LAND_1` | Không Đạo: Bẻ cong không gian bước vào kẽ hở hư không, né tránh 75% đòn tấn công. |
| `SPRING_EGG_HATCH` | `ON_CONSUME` | Có consumer hiện hành* | 3 | `GU_GIANT_ARMOR_I, GU_GIANT_SHIELD_I, GU_SPRING_EGG` | Cấy một Phàm Tuyền ngay dưới chân; hoạt động cả qua vật phẩm Trứng Nguyên Tuyền và Cổ tiêu hao GU_SPRING_EGG. Không bị giới hạn cap 31 Phàm Tuyền hay cap đại vực; không tiêu hao khi ô đã có địa điểm, vùng an toàn hoặc trong bí cảnh. |
| `STAR_FIRE_RAIN` | `ON_ACTION` | Có consumer hiện hành* | 4 | `GU_STAR_FALL_I` | Tinh Đạo: Hàng ngàn ngôi sao sa oanh tạc phe địch (+30% damage) và tạo 15% MaxHP Khiên Tinh Quang. |
| `STAR_THOUGHT_RESTORE` | `ON_ACTION` | Có consumer hiện hành* | 4 | `GU_STAR_FALL_I` | Tinh Đạo: Câu thông vạn dặm tinh hà, niệm đầu thông suốt phục hồi 30% Chân Nguyên. |
| `STAT_MODS` | `ALWAYS_ON` | Có tác dụng một phần/context | 1 | `GU_LOVE_DOMAIN` | Bảng tỉ lệ chỉ số (0.1 = +10%). Được chiếu từ race/constitution/status; gắn trên Gu thì không thay cho op STAT_MOD. |
| `STEAL_LIFE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_THEFT_LIFE_9` | Thâu Đạo: Cướp đoạt thọ mệnh sinh cơ, hút thẳng 15% MaxHP của đối phương chuyển hóa thành máu hồi phục bản thân. |
| `STONE_APERTURE` | `ON_CONSUME` | Legacy/compatibility; xem ghi chú | 4 | `GU_GIANT_POTATO_4, GU_GOLDEN_GATE_I, GU_STONE_APERTURE, GU_STONE_GATE_5` | Tên cũ của APERTURE_PETRIFICATION: đẩy lên Đỉnh Phong nhưng hóa đá 12h × 2^(rank−1); không hồi Chân Nguyên, không đột phá trong thời gian đó. |
| `SUMMON_MONSTER` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 12 | `GU_BOOK_SCHOLAR, GU_CHIVALROUS_HERO, GU_CONTROL_CHAIN_4, GU_CONTROL_STAMP_I, GU_DOMINATE_COMMAND_I` | Triệu hồi quái theo monster_id, mạnh dần theo rank Cổ và Đạo Ngân; giới hạn 2× HP và 3× sát thương của chủ. |
| `SUMMON_WOLF_FRONTLINE` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 1 | `GU_WOLF_ENSLAVEMENT_CORE` | Triệu hồi Địa Lang Vương có 50% MaxHP, 1/3 sát thương và 1/2 phòng thủ của chủ; đẩy chủ về hàng giữa. |
| `SUPPRESS_PRIORITY` | `ON_BATTLE_START` | Legacy/compatibility; xem ghi chú | 1 | `GU_BAN` | Vô hiệu Tiên Thủ/Né Lượt 1 của địch; nếu chặn được hiệu ứng địch thì người mang được đi trước. |
| `SWORD_ARMOR_PIERCE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_DRAGON_SLAYING_CORE_1` | Kiếm Đạo: Kiếm khí ngưng tụ vô kiên bất tồi, xuyên phá 40% phòng ngự và tăng 25% tỉ lệ bạo kích. |
| `SWORD_DRAGON_TRANSFORM` | `ON_CAST` | Có consumer hiện hành* | 1 | `GU_SWORD_DRAGON_TRANSFORM` | Chuyển 1 HP thành Khiên Ảo và tăng 50% sát thương trong 3 lượt. |
| `SWORD_MARK_DEATH` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_SWORD_MARK` | Kiếm Đạo: Khắc sâu kiếm ngân vào hồn phách địch, ăn mòn sinh mệnh mỗi hiệp và trảm sát khi dưới 25% HP. |
| `SWORD_WAVE_TRIPLE` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_TRIPLE_SWORD_WAVE_I` | Kiếm Đạo: Ba đợt sóng kiếm trào dâng, tăng 80% sát thương (3 hit liên kích) và gây Xuất Huyết. |
| `TEMP_STEAL_GU` | `ON_BATTLE_START` | Có tác dụng một phần/context | 1 | `GU_SHADOW_STEAL_8` | PvP thật: đoạt tạm một Cổ để dùng trong trận; tỉ lệ ~40% +5%/rank chênh, hết trận trả lại. |
| `TENACITY` | `ALWAYS_ON` | Có tác dụng một phần/context | 2 | `GU_HEART_DOMAIN, GU_HEART_LOTUS` | Tỉ lệ tự giải debuff đầu lượt và rút ngắn khống chế; trên Cổ phải dùng STAT_MOD TENACITY. |
| `THIRD_WATCH` | `ON_CONSUME` | Legacy/compatibility; xem ghi chú | 2 | `GU_MOMENT_WATER_5, GU_THIRD_WATCH` | Tên cũ của TIME_ACCELERATION: nhân hệ số tốc độ thời gian (mặc định ×3), cộng dồn mỗi lần dùng. |
| `THIRTY_THREE_HEAVEN_LIGHT` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_THIRTY_THREE_HEAVEN_LIGHT_I` | Quang Đạo: 33 tầng thiên quang rọi sáng, thanh tẩy toàn bộ phe ta và làm Mù Lòa toàn bộ quân địch. |
| `THOUGHT_REGEN_TICK` | `ON_TICK` | Có consumer hiện hành* | 8 | `GU_DIAMOND_THOUGHT_5, GU_DIVINE_SENSE, GU_GOLDEN_SUPPORT_I, GU_ROYAL_SUPPORT_4, GU_ROYAL_SUPPORT_I` | Hồi thêm Niệm Lực mỗi giờ (mặc định 1), giảm theo chênh rank Cổ. |
| `THREE_MEN_MAKE_TIGER` | `ON_ACTION` | Có consumer hiện hành* | 3 | `GU_THREE_MEN_TIGER_I` | Tín Đạo: Ba người nói thành hổ, hung hổ ảo ảnh vồ nát tâm trí mục tiêu, tăng 50% sát thương, áp đặt Sợ Hãi và Suy Yếu. |
| `THREE_QI_RETURN` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_THREE_QI_RETURN_I` | Khí Đạo: Thiên Địa Nhân tam khí quy nhất, hồi 30% HP, đầy Chân Nguyên, tăng 85% sát thương và bỏ qua 50% giáp. |
| `TIEP_THIEN_DIA` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CALAMITY_ARMOR_1` | Thiên Đạo: Câu thông ý chí Thiên Đạo, giải trừ toàn bộ khống chế và hồi 25% Chân Nguyên. |
| `TIME_REWIND` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ENDLESS_EONS_CYCLE_I` | Thời Đạo: Nghịch chuyển dòng sông thời gian cục bộ, hồi 35% MaxHP, 40% Chân Nguyên và tiêu trừ khống chế tinh thần. |
| `TOTEM_GOLDEN_BELL` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_GOLDEN_BELL_SHIELD` | Sát Chiêu Đồ Đằng: Chuông vàng viễn cổ buông rủ ngưng tụ hào quang, kích hoạt 35% Phản thương và kháng bạo kích. |
| `TOTEM_KUN_PENG` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_KUN_PENG_TOTEM_I` | Sát Chiêu Đồ Đằng: Côn Bằng tung cánh bạt sơn cửu thiên, tăng 75% sát thương và nhận 50% Thần Tốc. |
| `UNRAVEL_RIDDLE` | `ON_CAST` | Có tác dụng một phần/context | 1 | `GU_UNRAVEL_RIDDLE` | Xóa tối đa 2 buff của mục tiêu đầu tiên; phần 'ngắt chiêu tụ lực' hiện chỉ ghi log. |
| `VAN_DAN_DONG_TAM` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ALL_SPIRITS_PRIMORDIAL_MOTHER_I` | Nhân Đạo: Ngưng tụ hồng trần nhân tâm, tạo 35% MaxHP Khiên Nhân Đạo kiên cố hộ thể. |
| `WATER_CLEANSE` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_CLEANSING_WATER` | Thủy Đạo: Dòng nước tinh khiết bao bọc bản thân, cuốn trôi toàn bộ độc tố, thiêu đốt và xuất huyết. |
| `WATER_MIRROR_SHIELD` | `COMBAT_PASSIVE` | Có consumer hiện hành* | 1 | `GU_CLEANSING_WATER` | Thủy Đạo: Ngưng tụ gương nước phản xạ, tạo 20% MaxHP Khiên Ảo và hồi phục 10% HP. |
| `WIND_HASTE_BURST` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_BREEZE_ARMOR_1` | Phong Đạo: Ngự phong phi hành, thân pháp mau lẹ tăng 40% Tốc độ và 25% Né tránh. |
| `WOOD_BIND` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ALL_SPIRIT_IMMORTAL` | Mộc Đạo: Cổ thụ rễ cây trói chặt mục tiêu trong 2 hiệp, phong tỏa hành động và triệt tiêu né tránh. |
| `WOOD_REGEN_BURST` | `ON_ACTION` | Có consumer hiện hành* | 1 | `GU_ALL_SPIRIT_IMMORTAL` | Mộc Đạo: Vận khởi sinh cơ, hồi 15% Chân Nguyên và hồi 10% MaxHP mỗi lượt trong 4 lượt. |
| `WORDS_ARE_FRIGHTENING` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_WORDS_FRIGHTENING_I` | Tín Đạo: Lời đồn thế gian xé lòng, sát thương tăng theo số lượng debuff trên người mục tiêu (+25%/debuff) và Câm Lặng 2 hiệp. |
| `YAMA_EMPEROR` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_YAMA_EMPEROR_I` | Hồn Đạo: Hóa thân Diêm La Đại Đế, tăng 50% Phòng Ngự và kích hoạt Diêm La Tử cắn nuốt hồn phách địch mỗi hiệp. |
| `YOU_DU_STRENGTH_TIGER` | `ON_ACTION` | Có consumer hiện hành* | 2 | `GU_YOU_DU_TIGER_I` | Sát Chiêu Đồ Đằng: Dực Hổ gầm vang, tăng 85% sát thương, xé toạc sinh mệnh gây Xuất Huyết nặng và Choáng 1 hiệp. |

## Generic operations trong Gu.csv

Operation counts trong Gu.csv:

| Op | Occurrences | Có adapter | Ghi chú thực thi |
| --- | ---: | --- | --- |
| `CLEANSE` | 1 | Có consumer, theo context | Combat adapter; currently very sparse data use. |
| `DAMAGE` | 2271 | Có consumer, theo context | Combat ON_ACTION; số liệu/action còn chịu simulator rules. |
| `DIRECT_HP` | 3 | Có consumer, theo context | Combat only, specialized damage path. |
| `HEAL` | 330 | Có consumer, theo context | Combat or item adapter limited by context; world only self HP/MANA. |
| `SHIELD` | 901 | Có consumer, theo context | Combat ON_ACTION; context-specific target/turn rules. |
| `STATUS` | 707 | Có consumer, theo context | Combat status operation; status ID/trigger/target must be supported. |
| `STAT_MOD` | 2950 | Có consumer, theo context | Combat action buff supported; ALWAYS_ON only through passive projection allowlist. |
| `UNIQUE` | 532 | Có consumer, theo context | Native hook; must match trigger/domain and have a real consumer. |

Stat_MOD stats đang có trong Gu.csv: `ATTACK` (511), `CC_RESIST` (6), `CLEANSING_POWER` (8), `CRIT_RATE` (4), `DEFENSE` (747), `DISGUISE` (65), `HEARING` (1), `HP` (249), `INVESTIGATION` (228), `LUCK` (62), `MANA` (1), `MOVEMENT` (254), `PATH_CONFLICT_REDUCTION` (46), `PATH_RESISTANCE` (5), `REFINE_MANA_TIER` (50), `SOUL_REFINEMENT` (14), `SPEED` (665), `STORAGE` (27), `TENACITY` (6), `TIME_SPEED` (1).

## Các điểm cần coi là không hoạt động hoặc giới hạn

- **Item `RESTORE_HUNGER`**: không sửa hunger; handler chỉ ghi message.
- **Gu `ARENA_IDENTITY_MASK`**: xuất hiện trong Gu.csv và catalog CAPABILITY nhưng chưa có consumer ẩn danh đấu trường; không có tác dụng player-facing hiện tại.
- **Race `SOUL_DEFENSE`**: hiện đi qua projection nhưng chưa được dùng trong công thức damage; chỉ số hiển thị không đồng nghĩa giảm damage.
- **Gu `RECORD_MESSAGE`**: chỉ trả narrative/log; không lưu bản ghi hình ảnh/âm thanh.
- **Gu `POISON_VOW`**: compatibility alias; handler hiện chỉ log kết ước, chưa enforce lời thề.
- Một hook `CAPABILITY` chỉ mở quyền khi đúng feature gọi kiểm tra capability. Định nghĩa trong catalog không tự kích hoạt action.
- `effectcatalog/manifest.go` có thể gắn disposition `supported` hoặc `migrate` phục vụ migration/ownership. Đây không thay thế kiểm tra adapter/runtime consumer.

## Cập nhật danh mục

Khi đổi `Item.csv`, `Gu.csv`, `Race.csv` hoặc `Constitution.csv`, đối chiếu trang này với effect glossary và consumer code. Mã mới phải có: data source, trigger, context/executor, behavior, status (active/partial/legacy/no-op) và test/validation coverage phù hợp. Trigger/op/native hook được định nghĩa trong `frontend/src/lib/wiki/effect-glossary.ts`; behavior runtime nằm ở handler, `gamefx` hoặc simulator.


### Snapshot giá trị trong Race.csv

| Race key | Tên | Effect values đang khai báo |
| --- | --- | --- |
| `cuong_thi` | Cương Thi | `DISABLE_HP_REGEN=true; DISABLE_MANA_REGEN=true; DISABLE_RESOURCE_REGEN=true; DISABLE_SOUL_REGEN=true; IMMORTAL=true` |
| `dragonman` | Long Nhân | `REFINE_SUCCESS=-0.1; STAT_MODS={attack:0.1, defense:0.1}` |
| `feathered` | Vũ Dân | `DAO_MARK_MODIFIER={space:0.05}; STAT_MODS={speed:0.1}` |
| `fur_people` | Người Lông | `REFINE_SUCCESS=0.1` |
| `giant_sun_descendant` | Cự Dương Huyết Mạch | `CULTIVATION_EFFICIENCY=0.05; LUCK=5` |
| `human` | Nhân Tộc | `—` |
| `inkman` | Mặc Nhân | `DAO_MARK_MODIFIER={soul:0.05, wisdom:0.08}` |
| `merman` | Nhân Ngư | `DAO_MARK_MODIFIER={water:0.1}; PATH_DAMAGE_MODIFIER={water:0.05}` |
| `rockman` | Thạch Nhân | `STAT_MODS={defense:0.15, speed:-0.1}` |
| `snowman` | Tuyết Dân | `DAO_MARK_MODIFIER={ice:0.15}; PATH_RESISTANCE_MODIFIER={fire:0.1}` |
| `variant_beastman` | Thú Nhân | `REFINE_SUCCESS=-0.05; STAT_MODS={attack:0.08, defense:0.05}` |
| `eggman` | Đản Dân | `DAO_MARK_MODIFIER={cloud:0.05, wind:0.05}; STAT_MODS={defense:-0.1, speed:0.1}; HEALING_RECEIVED=0.15` |
| `miniman` | Tiểu Nhân | `DAO_MARK_MODIFIER={wood:0.1}; REFINE_SUCCESS=0.05; STAT_MODS={attack:-0.12}` |
| `mushroomman` | Nấm Nhân | `DAO_MARK_MODIFIER={earth:0.1}; STAT_MODS={defense:0.1, speed:-0.1}; HP_REGEN=15` |
| `yakshaman` | Dạ Xoa Dân | `STAT_MODS={attack:0.12}; DAO_MARK_MODIFIER={dark:0.08}; REFINE_SUCCESS=-0.08; MENTAL_RESISTANCE=-0.1` |
| `ghostman` | U Hồn Tộc | `DAO_MARK_MODIFIER={soul:0.12}; SOUL_DEFENSE=0.2; SOUL_REGEN=0.2; STAT_MODS={hp:-0.1}; HEALING_RECEIVED=-0.15` |
| `ancient_human` | Thái Cổ Nhân Tộc | `CULTIVATION_EFFICIENCY=0.1; LUCK=3; DAO_MARK_MODIFIER={human:0.1}; REFINE_SUCCESS=-0.05` |
| `xuan_human` | Huyền Nhân | `DAO_MARK_MODIFIER={xuan:0.15}; STAT_MOD PATH_CONFLICT_REDUCTION ADD_FLAT=10` |

### Snapshot giá trị trong Constitution.csv

| Constitution key | Tên | Effect values đang khai báo |
| --- | --- | --- |
| `myriad_gold_brilliant_flower_body` | Vạn Kim Diệu Hoa Thể | `DAO_MARK_MODIFIER={metal:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MODS={attack:0.3, defense:0.3}` |
| `northern_dark_ice_soul_body` | Bắc Minh Băng Phách Thể | `DAO_MARK_MODIFIER={ice:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; PATH_RESISTANCE_MODIFIER={ice:0.1}` |
| `pure_dream_reality_seeker_body` | Thuần Mộng Cầu Chân Thể | `DAO_MARK_MODIFIER={dream:0.4}; MANA_REGEN=1.0; MAX_MANA=0.4; PATH_DAMAGE_MODIFIER={dream:0.1}` |
| `solar_sun_moon_body` | Thái Nhật Dương Mãng Thể | `DAO_MARK_MODIFIER={light:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; PATH_DAMAGE_MODIFIER={light:0.2}` |
| `universe_great_evolution_body` | Vũ Trụ Đại Diễn Thể | `DAO_MARK_MODIFIER={space:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MODS={attack:0.15, defense:0.15, speed:0.15}` |
| `ancient_moon_yin_desolate_body` | Cổ Nguyệt Âm Hoang Thể | `DAO_MARK_MODIFIER={dark:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; MENTAL_RESISTANCE=0.2` |
| `blazing_glory_lightning_brilliance_body` | Viêm Hoàng Lôi Trạch Thể | `DAO_MARK_MODIFIER={lightning:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MODS={speed:0.2}` |
| `carefree_wisdom_heart_body` | Tiêu Dao Trí Tâm Thể | `DAO_MARK_MODIFIER={wisdom:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; MENTAL_RESISTANCE=0.3` |
| `forest_sea_reincarnation_body` | Sâm Hải Luân Hồi Thể | `DAO_MARK_MODIFIER={water:0.5}; HEALING_RECEIVED=0.5; MANA_REGEN=1.0; MAX_MANA=0.5` |
| `great_earth_origin_body` | Hậu Thổ Nguyên Ương Thể | `DAO_MARK_MODIFIER={earth:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; MENTAL_RESISTANCE=0.45; STAT_MODS={defense:0.45}` |
| `great_strength_true_martial_body` | Đại Lực Chân Võ Thể | `DAO_MARK_MODIFIER={strength:0.5}; MANA_REGEN=1.0; MAX_MANA=0.5; STAT_MODS={attack:0.3, defense:0.3}` |
| `frost_chilling_body` | Hàn Sương Thể | `DAO_MARK_MODIFIER={ice:0.25}; PATH_RESISTANCE_MODIFIER={ice:0.1}; STAT_MODS={speed:-0.1}` |
| `blazing_fiery_body` | Xích Liệt Hoang Thể | `DAO_MARK_MODIFIER={fire:0.25}; PATH_DAMAGE_MODIFIER={fire:0.1}; STAT_MODS={attack:0.1, defense:-0.1}; MENTAL_RESISTANCE=-0.15` |
| `brute_strength_body` | Cự Lực Man Thể | `DAO_MARK_MODIFIER={strength:0.25}; STAT_MODS={attack:0.15, defense:0.1, speed:-0.1}; MANA_REGEN=-0.2` |
| `heavy_rock_body` | Trọng Nham Huyền Thể | `DAO_MARK_MODIFIER={earth:0.25}; STAT_MODS={defense:0.25, speed:-0.15}; MENTAL_RESISTANCE=0.15` |
| `spirit_wood_body` | Linh Mộc Dưỡng Sinh Thể | `DAO_MARK_MODIFIER={wood:0.25}; HEALING_RECEIVED=0.3; HP_REGEN=20; STAT_MODS={attack:-0.1}` |
| `iron_bone_body` | Thiết Cốt Huyền Kim Thể | `DAO_MARK_MODIFIER={metal:0.2}; STAT_MODS={defense:0.2, speed:-0.1}; TENACITY=0.1; APERTURE_STRESS=0.25` |
| `clear_wisdom_heart_body` | Minh Tuệ Linh Tâm Thể | `DAO_MARK_MODIFIER={wisdom:0.25}; MENTAL_RESISTANCE=0.2; CC_RESIST=0.1; STAT_MODS={attack:-0.1, hp:-0.1}` |
| `swift_wind_body` | Tật Phong Phi Hành Thể | `DAO_MARK_MODIFIER={wind:0.25}; STAT_MODS={defense:-0.15, speed:0.2}; MAX_MANA=-0.1` |
| `night_shade_body` | U Dạ Ám Linh Thể | `DAO_MARK_MODIFIER={dark:0.25}; SOUL_DEFENSE=0.25; PATH_RESISTANCE_MODIFIER={light:-0.1}; HEALING_RECEIVED=-0.15` |
| `illusion_light_body` | Huyễn Quang Lưu Chuyển Thể | `DAO_MARK_MODIFIER={light:0.25}; PATH_DAMAGE_MODIFIER={light:0.1}; STAT_MODS={speed:0.1}; MANA_REGEN=-0.15` |
| `blood_fiend_body` | Huyết Ma Sát Thể | `DAO_MARK_MODIFIER={blood:0.25}; STAT_MODS={attack:0.15}; HEALING_RECEIVED=0.2; APERTURE_STRESS=0.35; REFINE_SUCCESS=-0.1` |
| `dream_phantom_body` | Huyễn Mộng Linh Thể | `DAO_MARK_MODIFIER={dream:0.2}; PATH_DAMAGE_MODIFIER={dream:0.05}; MENTAL_RESISTANCE=-0.15; CC_RESIST=-0.1` |

Các snapshot trên chỉ ghi payload CSV (value/path/stat), chưa thay thế quy tắc resolve của `gamefx` hay downstream consumer. Ví dụ, một giá trị được decode và lưu vào stats chưa chắc được công thức combat dùng; xem cột trạng thái phía trên.

### Snapshot Item.csv có khai báo effect

| Item ID | Effect code và value hiện tại |
| --- | --- |
| `ITEM_HEALING_PELLET_SMALL` | `RESTORE_HP=30` |
| `ITEM_MEDICINAL_PASTE` | `RESTORE_HP=50` |
| `mine_chest` | `OPEN_MINE_CHEST=1` |
| `mine_chest_immortal` | `OPEN_IMMORTAL_MINE_CHEST=1` |
| `spring_egg` | `SPRING_EGG_HATCH=None` |
| `ITEM_HEALING_PELLET_MEDIUM` | `RESTORE_HP=80` |
| `ITEM_BLOOD_GINSENG` | `RESTORE_HP=150` |
| `ITEM_HEALING_PELLET_LARGE` | `RESTORE_HP=200` |
| `ITEM_LIFE_SAVING_PILL` | `RESTORE_HP=500` |
| `ITEM_IMMORTAL_RECOVERY_PILL` | `HEAL_APERTURE=10000, RESTORE_HP=1000` |
| `ITEM_IMMORTAL_SPRING_WATER` | `HEAL_APERTURE=500, RESTORE_HP=500` |
| `ITEM_RANDOM_CONSTITUTION` | `RANDOM_CONSTITUTION=1` |
| `HUMAN_QI_PILL` | `INCREASE_HUMAN_QI=50000` |
| `ITEM_RANDOM_RACE` | `RANDOM_RACE=1` |
| `ITEM_APERTURE_EXPANSION_PILL` | `INCREASE_APERTURE_AREA=500000` |
| `ITEM_APERTURE_TIME_PILL` | `INCREASE_APERTURE_TIME_RATE=2` |
| `dao_material_blade` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_blood` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_bone` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_cloud` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_creation` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_dark` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_death` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_destruction` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_dream` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_earth` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_emotion` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_enslavement` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_fire` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_food` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_formation` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_heaven` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_human` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_ice` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_information` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_investigation` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_killing` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_light` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_lightning` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_luck` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_metal` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_painting` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_phantom` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_poison` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_qi` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_refinement` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_rule` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_soul` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_sound` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_space` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_star` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_strength` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_sword` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_theft` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_time` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_transformation` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_water` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_wind` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_wisdom` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `dao_material_wood` | `IMMORTAL_ONLY=1, ADD_DAO_MARKS=1000` |
| `XUAN_RANK1` | `RESTORE_HUNGER=10` |
| `XUAN_RANK2` | `RESTORE_HUNGER=20` |
| `XUAN_RANK3` | `RESTORE_HUNGER=30` |
| `XUAN_RANK4` | `RESTORE_HUNGER=40` |
| `XUAN_RANK5` | `RESTORE_HUNGER=50` |
| `XUAN_RANK6` | `RESTORE_HUNGER=60` |
| `XUAN_RANK7` | `RESTORE_HUNGER=70` |
| `XUAN_RANK8` | `RESTORE_HUNGER=80` |

Payload snapshot không tự khẳng định effect hoạt động; status runtime ở bảng Item.csv phía trên.
