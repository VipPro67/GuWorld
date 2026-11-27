# Tiên Khiếu

## Tính năng làm gì?

Tiên Khiếu là không gian riêng của người chơi tại Rank 6+, có node/resource point, worker, NPC, swarm, portal và hệ thống tribulation.

Danh sách Đàn Cổ lấy tên, loại, lưu phái, mô tả và thức ăn từ `gu_dictionary`; nếu thiếu bản ghi từ điển thì vẫn trả ID Gu để nhận diện đàn.

## Điều kiện mở và quyền truy cập

Player cần đạt Rank 6 và anchor khi không đang trong aperture khác. Chủ sở hữu đặt access policy. Môn Hộ được hiện cho chủ, theo policy mở/đồng môn, hoặc khi điểm dò deterministic đạt điểm che giấu của chủ cộng `concealment_power`. Hai phía dùng STAT và realm/Đạo của scorer hiện có; passive `INVESTIGATION` tăng điểm dò, còn `DISGUISE` tăng điểm che giấu. Passive conditions được chiếu theo race viewer/owner và rank khi scorer kiểm tra portal. Dò thành công hiện tọa độ chính xác nhưng không tiết lộ chủ nhân, mã Tiên Khiếu hay policy cổng. Người đã nhìn thấy Môn Hộ có thể vào trong phạm vi 3 ô; kết quả dò thành công vượt qua giới hạn PARTY/SECT.

## Cách sử dụng

Anchor aperture, quản lý node và worker, đặt policy che giấu/hiển thị, rồi mở portal để di chuyển. Khi khách vào, chủ nhận notification. Chủ có thể trục xuất khách sau khi khách rời realtime combat; yêu cầu trong lúc combat bị từ chối, không xếp hàng. Trục xuất hủy ngay nghi thức di dời mỏ đang do khách thực hiện và mỏ vẫn ở nguyên Tiên Khiếu. Tribulation được khởi động và giải quyết theo lượt; khách có thể tham gia nếu luật session cho phép.

Khi Hạ Khiếu, chủ được chuyển tới tab Bản đồ & Phòng thủ với toàn cảnh lưới 20×20. Khách đã vào qua Môn Hộ chỉ thấy map khám phá quanh vị trí theo tầm nhìn thường; API chỉ trả người chơi, tác nhân quái, chiến lợi phẩm và mỏ trong phạm vi này. Khách có thể thu hoạch điểm tài nguyên khi ở trong phạm vi và tự rời Tiên Khiếu qua nút Rời Tiên Khiếu mà không cần chủ phê duyệt. Khi rời, khách trở về vị trí Môn Hộ bên ngoài. Các thao tác quản trị Tiên Khiếu vẫn theo quyền chủ hiện có.

Thay đổi ô map và Môn Hộ được cập nhật qua `world:change` chỉ trong đúng `map_instance_id`. Snapshot ô map, điểm tài nguyên và actor quái có revision theo scope; client resync sau reconnect hoặc khi phát hiện thiếu delta. Combat Tai Kiếp vẫn theo lượt.

## Tiên Khiếu vô chủ và site cấu hình

Site tạo sẵn lấy tọa độ, rank, grade, diện tích và Đạo Ngân từ `PublicApertureSite.csv`, import qua Admin → Dữ liệu. Các seed hiện tại gồm 23 Động Thiên & Phúc Địa phân bố khắp Ngũ Vực (Nam Cương: Tam Vương, Huyết Hải, Bạch Cốt, Vạn Hoa, Thập Vạn Thú; Trung Châu: Thần Tiêu, Hồ Tiên, Thất Tinh, Lạc Thiên, Vạn Kiếm; Bắc Nguyên: Tuyết Nguyên, Hắc Phàm, Ma Tổ, Vương Đình, Băng Phong; Tây Mạc: Thiên Khanh, Đạo Tặc, Lục Bảo, Xích Nhiễm; Đông Hải: Khí Hải, Bồng Lai, Long Cung, Quy Khư). Tọa độ và khu vực của Môn Hộ tiên khiếu vô chủ được hiển thị trực tiếp trên Admin dashboard (danh sách và chi tiết). Worker tạo state và Môn Hộ một lần theo `site_key`; site chỉ xuất hiện sau khi import cấu hình. Popup cập nhật trong `frontend/src/app/page.tsx` giới thiệu cách scouting, vào khiếu, thu hoạch và hấp thụ. Khi lifespan cạn, worker xử lý cả player offline. Nếu player không ở combat, transaction xóa player, giữ aperture, bỏ liên kết chủ và chuyển Môn Hộ thành công cộng; nếu đang combat thì đánh dấu pending, chặn hành động mới và chờ combat settle. Khách hiện có được giữ lại; khách mới vào theo portal/scouting và không phát notification cho chủ cũ.

Trong instance công cộng, UI chỉ mở tab bản đồ và điểm tài nguyên. Môn Hộ chỉ hiện sau khi scouting thành công; người dò thấy có thể vào mà không cần rank, party, sect hay chủ cấp quyền. Các ô sinh thái dùng spawn pool `aperture_interior` trong `MonsterSpawn.csv`; thời gian hồi lấy từ `respawn_seconds`, còn trạng thái ô được scope theo `map_instance_id`, nên hai khiếu có bố cục và chu kỳ hồi độc lập. NPC dị nhân còn sống trong khiếu được giữ lại khi chủ thọ tận, chuyển thành hộ vệ vô chủ có vị trí trong map; bước vào ô hộ vệ sẽ mở combat theo lượt, hộ vệ chỉ có thể bị đánh một trận tại một thời điểm và bị loại khỏi map sau khi bị hạ. Khách có thể di chuyển, giao tranh với người chơi thấy được, thu hoạch stock chung trong phạm vi; stock tích lũy theo template sản lượng và mức Thiên/Địa Khí hiện có. Mỏ không hồi sản lượng/sinh khí khi tàn tích. Người chơi đủ rank và cảnh giới lưu phái chủ đạo có thể bắt đầu hấp thụ khi đang ở đúng instance; một reservation hoạt động trên mỗi mục tiêu/người hấp thụ. Combat PvP làm gián đoạn, hai khiếu vẫn tồn tại và phí/thời lượng do cấu hình server quyết định. Thành công xóa khiếu mục tiêu, cộng dồn diện tích, Đạo Ngân và số lần tai kiếp; mỏ chuyển sang ô trống gần nhất, mỏ không còn ô sẽ bị loại bỏ. Tai kiếp môi trường tự settlement theo lịch tai kiếp hiện có và giảm HP vách theo tỷ lệ cấu hình. Khi HP tường về 0, khiếu chuyển thành tàn tích theo thời hạn cấu hình; hết hạn, hệ thống chờ combat kết thúc, đưa khách online/offline về Môn Hộ rồi xóa state. Cấu hình cần có `APERTURE_ABSORPTION_DURATION`, `APERTURE_ABSORPTION_COST_IMMORTAL_STONES`, `PUBLIC_APERTURE_RUIN_DURATION`, `PUBLIC_APERTURE_TRIBULATION_HP_DAMAGE_FRACTION`.

## Kinh doanh Dị Nhân, Thành Trì & Cổ Tiên Tiên Khiếu

- **9 chủng tộc dị nhân & Sở trường mở:** Hỗ trợ đủ 9 tộc (`HAIRY_MAN`, `ROCKMAN`, `FEATHERMAN`, `SNOWMAN`, `INKMAN`, `MINIMAN`, `MERMAN`, `MUSHROOMMAN`, `DRAGONMAN`). Mọi tộc đều có thể làm việc tại Điểm Tài Nguyên hoặc Luyện Cổ Tự Động, nhưng gán đúng tộc sở trường (`worker_race` trong `ResourcePointTemplate.csv`) nhận thưởng sản lượng tối đa `+50%` ở 500 lao động (gán sai tộc chịu hệ số `0.8x`), kèm đặc tính riêng (vd: Vũ Dân giảm 20% tiêu hao Thiên/Địa Khí).
- **Thành Trì tuần tự & Chiếm ô lưới:** Xây Thành Trì bắt buộc từ `VILLAGE` (tốn 500 Tiên Nguyên Thạch, 0 dân sẵn, chiếm 1 ô `(pos_x, pos_y)` trên lưới Tiên Khiếu), nâng cấp tuần tự `VILLAGE → TOWN → CITY → FORTRESS` theo Rank (6/7/8+), diện tích tối thiểu và Tiên Nguyên Thạch. Dân số ban đầu chiêu mộ qua `/variant-humans/recruit`; sinh trưởng tự nhiên phụ thuộc vào ngưỡng Đạo Ngân môi trường của tộc đó.
- **Lương thực, Cứu trợ & Nổi loạn:** Điểm Tài Nguyên có `is_food = true` tự động nạp `food_stored` vào kho Thành Trì (hoặc nạp thủ công qua `/variant-humans/feed`). Khi hết lương thực, bộ tộc rơi vào trạng thái Đói kém (`starving`), chỉ trừ `immortal_essence` khi chủ khiếu bật công tắc cứu trợ (`/variant-humans/relief`). Bất ổn kéo dài kích hoạt sự kiện leo thang trong `aperture_variant_events` (`STRIKE` → `SABOTAGE` → `REBELLION`), giải quyết qua `An dân` (`PACIFY`) hoặc `Trấn áp` (`SUPPRESS`).
- **Cổ Sư & Cổ Tiên Dị Nhân (`VARIANT_NPC`):** Cổ Sư spawn theo 9 tộc và lưu phái sở trường, hỗ trợ vai trò `REFINER`, `MINER`, `FARMER`, `STEWARD`. Cổ Sư Rank 5.4 trung thành ≥ 70% có thể được Hộ pháp Thăng Tiên lên Rank 6 Cổ Tiên (`/npcs/:npc_id/ascend`), sở hữu Đạo Ngân riêng, trang bị Tiên Cổ, quản trị ổn định khiếu và tham gia chiến đấu qua đội hình (`formation_slot`) hoặc thủ khiếu (`defense_stationed`).

## Luật, chi phí và giới hạn

- Rank 6+ mới dùng aperture.
- Không thể anchor bên trong aperture khác.
- Portal entry kiểm tra world, combat, khoảng cách 3 ô và policy. Người dò thành công Môn Hộ được vào bất kể policy PARTY/SECT; cổng HIDDEN yêu cầu điểm dò đạt điểm che giấu của chủ cộng `concealment_power`.
- Grid, người chơi, mỏ, quái và chiến lợi phẩm của Tiên Khiếu được giới hạn theo `map_instance_id`; khách chỉ nhận dữ liệu trong tầm nhìn thường và chỉ đọc mỏ khi còn trong roster, đúng instance.
- Lưới và phạm vi di chuyển của Tiên Khiếu dùng tọa độ 0–19 trên mỗi trục.
- Thu hoạch mỏ Tiên Khiếu ghi tài nguyên vào túi của người thao tác; khách cần ở trong phạm vi 6 ô. Khách có thể tự rời Tiên Khiếu bất cứ lúc nào qua `POST /api/immortal-aperture/leave`; chủ không cần phê duyệt.
- Mỏ ngoài map có nghi thức di dời vào Tiên Khiếu của người chơi qua `POST /api/immortal-aperture/world-points/:id/transplant`; nghi thức tiêu hao mana/tiên nguyên theo cấp mỏ và cần đứng tại mỏ đến khi hoàn tất. Đây là di chuyển mỏ, không phải xóa mỏ. Hiện chưa có combat để tấn công phá mỏ; tai kiếp có thể làm giảm sinh khí mỏ trong Tiên Khiếu.
- Hấp thụ khiếu công cộng: `GET /api/immortal-aperture/absorption/:apertureId` xem reservation; `POST /api/immortal-aperture/absorption/:apertureId/start` bắt đầu. Rank mục tiêu 6–9 yêu cầu lần lượt Master, Grandmaster, Great Grandmaster, Supreme Grandmaster ở mọi lưu phái đồng hạng Đạo Ngân chủ đạo; khi không có Đạo Ngân dương thì không thể hấp thụ.
- Resource production, worker upkeep, capacity và tribulation dùng state/config server-side.

## Kết quả và trường hợp thất bại

Thiếu rank, vị trí không hợp lệ, đang combat hoặc không đủ quyền sẽ chặn anchor/entry. Tribulation có thể kết thúc bằng kết quả thành công hoặc thất bại và tác động realm/player theo handler.

## Liên hệ với hệ thống khác

Rank 6 mở Tiên Khiếu và currency/economy nhánh tiên. Portal gắn với world movement; resource points và workers chạy qua background workers; combat/tribulation vẫn turn-based.

## Dữ liệu/cấu hình ảnh hưởng

Node, resource, worker, NPC, swarm và tribulation parameters nằm trong schema/config/game data tương ứng.

## Mã nguồn liên quan

- Backend: `backend/internal/handlers/aperture_core.go`, `aperture_portal.go`, `aperture_points.go`, `exploration_grid.go`, `ground_drops.go`, `backend/internal/gamefx/scouting_engine.go`, `tribulation_multiform.go`, `tribulation_handlers.go`, `legacy_engage_tribulation.go`, `legacy_resolve_tribulation.go`.
- Frontend: `frontend/src/store/slices/apertureSlice.ts`, `frontend/src/components/ImmortalApertureView.tsx` và `frontend/src/components/ExplorationMap.tsx`.

## Hệ thống Đa Dạng Hóa Tai Kiếp (Multi-form Tribulation)
Tai kiếp Tiên Khiếu vận hành theo 3 Archetype và vòng đời 3 giai đoạn:
1. **Giai đoạn Điềm Báo (`HARBINGER`)**: Dự báo trước loại kiếp, thời gian chuẩn bị và lưu phái giáng lâm. Cho phép Cổ Tiên củng cố trận pháp hoặc click "Nghênh Kiếp Sớm" (`POST /api/tribulations/:eventId/start`).
2. **Giai đoạn Tác Chiến (`ORDEAL_ACTIVE`)**:
   - `ENVIRONMENTAL`: Thiên tai môi trường sinh thái lan tỏa trên lưới ô, hóa giải bằng Sát Chiêu Tiên Đạo (`POST /api/tribulations/:eventId/mitigate-hazard`).
   - `BEAST_SWARM`: Đợt thú triều càn quét tài nguyên, hỗ trợ Nô Đạo thu phục khi HP quái ≤ 30% (`POST /api/tribulations/:eventId/capture-beast`).
   - `ANOMALOUS_COMBAT`: Chiến đấu boss turn-based với hiệu ứng dị biến môi trường (Lôi Đình, Bão Tuyết, Cấm Pháp) và cơ chế Đạo Ngân Xung Đột.
3. **Giai đoạn Hậu Kiếp & Thu Hoạch (`AFTERMATH`)**:
   - Rơi rớt Kiếp Tài quý hiếm (`aperture_tribulation_spoils`) trên lưới ô, thu hoạch thủ công hoặc nhờ Dị Nhân càn quét (`POST /api/tribulations/:eventId/claim-all-spoils`).
   - Dư ba ô nhiễm Đạo Ngân (`aperture_soil_contaminations`): Cổ Tiên chọn **Thanh Tẩy** gột rửa đất đai hoặc **Biến Nguy Thành Cơ** quy hoạch mở mỏ tài nguyên mới (`aperture_resource_points`).
   - Kết toán hoàn tất (`POST /api/tribulations/:eventId/finalize-aftermath`) đưa Tiên Khiếu trở về `STABLE`.

## API liên quan

Nhóm aperture/portal/tribulation routes; tra [API guide](../guides/API.md) và router `backend/internal/server/server.go`.

## Ví dụ flow

```text
đạt Rank 6 → anchor aperture → đặt quyền PARTY → tạo portal
→ điềm báo tai kiếp (HARBINGER) → nghênh chiến thiên tai/thú triều/boss (ORDEAL_ACTIVE)
→ thu hoạch kiếp tài & thanh tẩy/mở mỏ mới (AFTERMATH) → Tiên Khiếu thăng hoa ổn định (STABLE)
```
