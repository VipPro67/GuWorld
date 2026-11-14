# Sát Chiêu và Tiên Cổ Ốc

## Tính năng làm gì?

Hai tính năng cho phép đóng gói Gu và effect thành sản phẩm có profile riêng:

- **Sát Chiêu:** cấu hình một Gu chính, Gu phụ và các effect được phép composition; sản phẩm được dùng trong combat theo lượt.
- **Tiên Cổ Ốc (Gu House):** chọn core Gu và cấu hình hiệu ứng/khả năng của nhà; có thể trang bị, dùng một số `ON_USE` effect trong world và repair.

Không phải mọi Gu effect đều được phép nối với nhau hoặc dùng trong mọi context.

Effect mới trong `backend/data/Gu.csv`: Xuân Tiễn Tiên Cổ rank 6 có `STATUS SLOW` (20%, 2 lượt) trên mục tiêu; Liệu Quang Giáp Tiên Cổ rank 6 có `CLEANSE` trên bản thân, thay cho trạng thái `BLIND` tự gây trước đây. Tên Xuân Tiễn lấy cảm hứng từ danh sách sát chiêu tiên đạo; trị số và cơ chế là thiết kế của game, không được danh sách đó mô tả. Cả hai là lựa chọn khi tạo Sát Chiêu từ Cổ tương ứng, tuân theo preview và allowlist combat; chúng không phải công thức Sát Chiêu cố định.

### Effect theo lưu phái, đợt 1

Các block dưới đây thuộc Tiên Cổ rank 6, dùng operation combat hiện có. Đây là lựa chọn effect khi có Cổ tương ứng và preview hợp lệ, không phải 25 sát chiêu cố định.

| Lưu phái | Cổ và effect mới |
| --- | --- |
| Thiên đạo | Kiếp Giáp: Tăng Thủ; Nhãn Môn: Né Tránh; Kim Hà: Phá Giáp; Chiêu Kiếp: Suy Yếu; Vô Thiên Ý: Câm Lặng |
| Vũ đạo | Giới Vực: Trói Buộc; Phá Nang: phá 20% khiên; Lý Môn: Làm Chậm; Vô Cực Môn: Phá Giáp; Độn Không: Tăng Tốc |
| Trụ đạo | Lưu Tiễn: Làm Chậm; Lai Kính: Né Tránh; Nhân Như Cố: Hồi Phục; Khắc Thủy: Giảm Sát Thương; Thời Sa: Câm Lặng |
| Nhân đạo | Xá Lợi Đoàn Viên: Hồi Phục; Thư Sinh Đoàn Viên: Thanh Tẩy; Binh Binh: Tăng Công; Binh Đoàn Viên: Phản Thương; Nhân Tâm: Tăng Thủ |
| Khí đạo | Bạo Bạo: Phá Giáp; Thần Lưu: Làm Chậm; Lưu Giáp: Tăng Thủ; Khí Bích: Thanh Tẩy; Khí Ba: Suy Yếu |
| Hồn đạo | Cổ Vụ: Hoảng Sợ; Cổ Thương: Hồn Thương; Khấp Đăng: Mê Loạn; Khấp Thương: Choáng; Khấp Bích: Giảm Sát Thương |
| Độc đạo | Cốt Vụ: Trúng Độc; Hủ Cốt Độc Vụ: Tê Cóng; Tuyệt Mệnh Tiễn: Suy Yếu; Độc Huyết: Phá Giáp; Huyết Đỉa: phá 20% khiên |
| Huyết đạo | Huyết Giáp Bích: Phản Thương; Huyết Nhận: Xuất Huyết; Huyết Cốt Thuẫn: Giảm Sát Thương; Huyết Y: Hồi Phục; U Nguyệt Huyết: Suy Yếu |
| Lôi đạo | Thanh Long: Choáng; Sát Xích: Làm Chậm; Chớp Cầu: Mù; Lục Sắc Lôi: Phá Giáp; Thiên Võng: Trói Buộc |
| Phong đạo | Vi Phong Giáp: Né Tránh; Phi Hành: Tăng Tốc; Phong Phong: Xuất Huyết; Thiên Nhận Vụ: Làm Chậm; Thiên Nhận Phong: Suy Yếu |
| Trí đạo | Thái Độ Giáp: Giảm Sát Thương; Kim Cương Niệm: Tạp Niệm; Thanh Huyền: Mê Loạn; Bá Hỗ Trợ: Tăng Công; Giải Mê: Câm Lặng |
| Kiếm đạo | Trảm Long Hạch: Phá Giáp; Phi Kiếm: Kiếm Ngân Tác Mệnh; Phi Kiếm Dực: Tăng Tốc; Hộ Kiếm Thuẫn: Giảm Sát Thương; Phi Kiếm Tiên Cổ: Xuất Huyết |
| Lực đạo | Thỉ Trảo: Cuồng Man Bá Thể; Kim Cương: Tăng Thủ; Trọng Thân: Phản Thương; Trọng Bích: Giảm Sát Thương; Bạt Sơn: Suy Yếu |
| Biến Hóa đạo | Thú Thú: Thích Ứng Đạo Pháp; Thú Thể: Hóa Thân Cự Hùng; Tông Tông: Hóa Thân Kim Điêu; Tông Trảo: Xuất Huyết; Long Tức Châu: Hoảng Sợ |
| Vận đạo | Đoạn Vận: Đoạt Vận; Phúc Vân: Nghịch Chuyển Vận Rủi; Hồng Súc: Né Tránh; Thiên Súc: Suy Yếu; Ẩn Vận: Ẩn Thân |
| Mộng đạo | Mộng Điệp: Mộng Cảnh Giam Cầm; Mộng Cảnh: Mộng Ảnh; Mộng Dực: Né Tránh; Thận Lâu Yên: Mê Loạn; Ảo Giới: Ngủ Say |
| Tinh đạo | Sương Tiễn: Đóng Băng; Sương Kiếm: Tê Cóng; Hoàng Kim Quang: Mù; Nguyệt Thạch: Né Tránh; Dạ Lạc: Ngủ Say |
| Vân đạo | Vân Hải: Hồi Phục; Vân Thuẫn: Giảm Sát Thương; Vụ Thuẫn: Né Tránh; Bình Bộ Thanh Vân: Tăng Tốc; Vũ Tiễn: Làm Chậm |
| Tin Tức đạo | Thư Thư: Loạn Ý; Tâm Kính: Động Tất Sơ Hở; Tam Nhân Thành Hổ: Hoảng Sợ; Chúng Khẩu Thao Kim: Câm Lặng; Hồn Kính rank 7: Lãng Quên Thực Tại |
| Luật đạo | Giả Xích: Phong Ấn Cổ; Giả Tiêu: Xung Đột Đạo Ngân; Kim Thiền Thoát Xác: Hóa Thân Kim Điêu; Pháp Tiêu: Trói Buộc; Lệnh Phán Quyết: Suy Yếu |
| Không Gian đạo | Giới Vực: Trói Buộc; Phá Nang: Làm Chậm; Lý Môn: Né Tránh; Vô Cực Môn: Câm Lặng; Không Kiều: Tăng Tốc |
| Tử Vong đạo | Lãnh Hủ: Trúng Độc; Thi Chú: Khóa Hồi Sinh; Thi Vụ: Hoảng Sợ; Chú Vụ: Suy Yếu; Chú Thủy: Tê Cóng |
| Sáng Tạo đạo | Sáng Tạo Nê: Hồi Phục; Hoa Hạch: Phản Thương; Hoa Mộc: Tăng Thủ; Sinh Chủng: Tăng Tốc; Mệnh Hạch: Giảm Sát Thương |
| Luyện đạo | Dưỡng Đan: Hồi Phục; Linh Hỏa: Thiêu Đốt; Thủy Thạch: Phá Giáp; Vạn Triệt rank 7: phá khiên; Thăng Tuyền rank 8: Tăng Công |
| Quang Minh đạo | Mang Kiếm: Mù; Liệu Quang Giáp: Thanh Tẩy; Liệu Quang Bích: Hồi Phục; Thiên Giáp: Giảm Sát Thương; Thiên Kính: Mê Loạn |
| Tình Cảm đạo | Tâm Hợp: Hồi Phục; Tình Chú: Mê Hoặc; Tình Hợp: Giảm Sát Thương; Tình Hồn rank 7: Hồn Thương; Đoạn Duyên rank 7: Khóa Hồi Sinh |
| Hội Họa đạo | Bút Họa Trung Tiên: Mê Loạn; Thải Họa Trung Tiên: Mù; Mộng Họa Trung Tiên: Né Tránh; Mộng Thuẫn: Giảm Sát Thương; Mặc Thân: Trói Buộc |
| Nô đạo | Ngự Xích: Trói Buộc; Ngự Ấn: Câm Lặng; Bá Lệnh: Hoảng Sợ; Bá Tiên: Suy Yếu; Ưng Tiêu: Tăng Công |
| Đao đạo | Đao Khí: Xuất Huyết; Đao Thuẫn: Phản Thương; Đao Kiếm: Phá Giáp; Thiên Địa Phán Quyết Đao rank 7: Kiếm Ngân Tác Mệnh; Vạn Đao Quy Nhất rank 7: Suy Yếu |
| Thực đạo | Thượng Ngư: Hồi Phục; Thượng Quả: Tăng Công; Thực Đỗ: Giảm Sát Thương; Linh Thực: Tăng Tốc; Thực Thú Nhục: Phản Thương |
| Cốt đạo | Cốt Giáp: Bạch Cốt Giáp; Cốt Thứ Xa: Xuất Huyết; Tử Cốt Thứ: Phá Giáp; Bạch Cốt Sơn rank 7: Phản Thương; Cốt Giáp rank 7: Giảm Sát Thương |
| Ám đạo | Ẩn Y: Ẩn Thân; Ẩn Dạ: Suy Yếu; Thiên Nhai U Minh rank 7: Mù; Thiên Cẩu Thực Nhật rank 7: Làm Chậm; Ẩn Dạ rank 6: Câm Lặng |
| Hủy Diệt đạo | Phần Trảo: Thiêu Đốt; Phần Hỏa: Phá Giáp; Kiếp Ba: Suy Yếu; Phần Trảo rank 7: Hoảng Sợ; Kiếp Ba rank 7: Xuất Huyết |
| Trận đạo | Chướng Phá: Trói Buộc; Liên Hạch: Làm Chậm; Hạch Trụ: Giảm Sát Thương; Chướng Phá rank 7: phá khiên; Hạch Trụ rank 7: Tăng Thủ |
| Sát đạo | Pháp Kích: Câm Lặng; Huyết Sát Thân: Huyết Tế Cuồng Bạo; Thiên Quyền: Tăng Công; Sát Quyền: Choáng; Sát Giới: Khóa Hồi Sinh |
| Ảo Ảnh đạo | Vô Tướng Nhãn: Mù; Tâm Ty: Mê Loạn; Cảnh Thể: Né Tránh; Huyễn Vụ: Câm Lặng; Huyễn Thân: Ẩn Thân |
| Âm đạo | Chung Thuẫn: Giảm Sát Thương; Hồi Âm Tiễn: Câm Lặng; Cổ Ba: Choáng; Sinh Âm Chung: Hồi Phục; Âm Ca: Làm Chậm |
| Đạo Tặc đạo | Thâu Tiền: Suy Yếu; Ẩn Ảnh Đạo Thủ: Câm Lặng; Diệu Thủ Kiếm: Xuất Huyết; Mặc Diệu Thủ: Mù; Đạo Thủ: Đoạt Vận |
| Thủy đạo | Vô Tận Hải: Làm Chậm; Ngư Lân Thủy: Giảm Sát Thương; Thiên Hà: Trói Buộc; Hải Giáp: Phản Thương; Ba Nhận: Xuất Huyết |
| Huyền đạo (XUAN) | Trạch Địa Tụy: Mê Loạn; Trạch Thiên Quải: Né Tránh; Trạch Lôi Tùy: Choáng; Sơn Địa Bác: Tăng Thủ; Thủy Thiên Nhu: Làm Chậm |
| Điều Tra đạo | Thần Tích: +3 Điều Tra; Khuy Thiên Nhãn: +5; Huyền Ảnh: +4; Sắc Ảnh: +4; Hồn Tích: +5. Đây là passive dùng consumer scouting/profile hiện có, không phải effect combat của Sát Chiêu. |

Các status mới thêm có chance gốc 75%, kéo dài 2 lượt và cường độ 20% khi status có trị số phần trăm. Hồi Phục từ Nhân Như Cố và Xá Lợi Đoàn Viên là 60.000 HP mỗi lượt. Rank và kháng hiệu ứng có thể thay đổi xác suất thực tế. Thanh Tẩy xóa debuff của bản thân; Phá Nang dùng `DIRECT_HP/SHIELD_STRIP` thay vì gây HP damage trực tiếp.

## Điều kiện mở và quyền truy cập

- Routes `/killer-moves/*` và `/gu-houses/*` yêu cầu đăng nhập và onboarding.
- Component Gu phải thuộc player, đang sống, hợp lệ cho product profile/context và không bị product khác giữ.
- Sát Chiêu cần preview hợp lệ; create yêu cầu đúng fingerprint của preview gần nhất. Nếu preview tự scale effect, player phải chấp nhận điều chỉnh trước khi lưu.
- Giới hạn số product đã lưu/equipped, số Gu component và số effect do product profile trả về; xem `/killer-moves/options` thay vì hard-code UI cap.
- Một Gu không thể đồng thời nằm trong active Sát Chiêu, active Gu House hoặc teleport array reservation xung đột.

## Cách sử dụng

### Tạo Sát Chiêu

1. Lấy `/killer-moves/options` để biết Gu instances, effect options, composition templates và rule revision hiện tại.
2. Chọn Gu chính, component Gu và effect output/support được catalog cho phép.
3. Gọi `/killer-moves/preview`; xem power budget/load, scale, harmony, complexity, success chance, mana cost, target profile và warnings.
4. Lưu tên cùng draft và `preview_fingerprint`; chấp nhận điều chỉnh nếu effect bị scale.
5. Trang bị bằng `/:id/equip`. Server tạo reservation cho Gu thành phần; hủy equip hoặc xóa theo route để giải phóng.

### Dùng hoặc repair Sát Chiêu

Sử dụng khi combat/profile cho phép; action combat đi qua combat session plan và simulator. Route use là compatibility/gameplay entry theo handler; UI/server sẽ xác nhận context. Repair khi durability/product state cho phép.

### Tạo và dùng Tiên Cổ Ốc

1. Xem Gu core/effect options trong màn Gu House.
2. Gọi `/gu-houses/preview`, xem warnings, profile và tài nguyên cần thiết.
3. Tạo rồi equip. Active house giữ core Gu bằng owner-typed reservation.
4. Dùng action/ON_USE effect có trong profile; server kiểm tra house đang equipped và core vẫn reserved cho house đó.
5. Unequip hoặc repair qua route tương ứng. Không sửa/xóa core bằng inventory trong lúc còn active reservation.

## Luật, chi phí và giới hạn

### Sát Chiêu: ngân sách và xác suất

Profile hiện tại tính:

```text
powerBudget = 20 + 26*mainGuRank + Σ(13 / sqrt(supportIndex))
complexity = 1 + .14*(componentCount-1) + .12*(effectCount-1)
             + .10*max(0, pathCount-1)
harmony = 1 (một component), ngược lại .6 + .4*(matchingPathPairs/allPairs)
```

Nếu load vượt budget, scalable load nhận scale `(budget-unscalableLoad)/scalableLoad`; cleanse load là unscalable và scale dưới minimum profile hiện .15 bị reject. Xác suất tạo thành công trước clamp là `.55 + .035*pathMasteryLevel + .18*(harmony-.6) + min(.15, masteryXP/1000*.15) - .15*(complexity-1)`, clamp 10%–95%. Profile giới hạn 6 Gu, 8 effect nodes, 12 product đã lưu và 4 product equipped.

Khi dùng ngoài combat, chance được điều chỉnh theo chênh lệch mastery hiện tại/lúc tạo và XP, sau đó clamp 5%–95%. Mana theo component cost; lần dùng thất bại vẫn trừ mana và mỗi component mất 1 hunger. Preview/response là số cuối cùng vì rule profile có revision.

### Tiên Cổ Ốc: profile và repair

Profile combat hiện tại tính `HP = highestCoreRank*500 + coreCount*250 + auxCount*35`, `defense = highestCoreRank*40 + coreCount*25 + auxCount*4`, damage reduction `min(.35, .10 + .05*coreCount)`. Repair tốn `ceil((100-durability)*highestCoreRank*10)` Primeval Stones và phục hồi durability lên 100. Đây là profile của nhà, không phải stat cộng vĩnh viễn vào player.

Trong combat theo lượt, khi Tiên Cổ Ốc đang dung nhập (`GuHouse.Active = true`), Form HP hấp thu 100% sát thương thay cho nhục thân theo quy tắc Không Tràn Sát Thương (No Spillover). Khi Form HP cạn kiệt, Cổ Ốc bị vỡ vụn (`current_durability = 0`, tự gỡ trang bị), kích phát phản phệ trận pháp khiến Cổ Tiên bị văng ra chịu Choáng 2 lượt và mất 30% HP tối đa, đồng thời truyền 30% xung kích làm giảm độ bền của toàn bộ Tiên Cổ hạch tâm. Sau trận đánh thực tế (non-virtual), độ bền còn lại của Cổ Ốc và Cổ hạch tâm được kết toán bền vững vào database (`player_gu_houses`, `player_gu`).

- `composition_contributions` phải grant capability rõ tới effect block/operation. Primary role, combat role hoặc tag không tự tạo quyền composition.
- Product rule set version/revision pin vào composition draft/active artifact. Preview cũ có thể hết hiệu lực nếu source Gu/rule thay đổi; tạo preview mới trước khi commit.
- Effect context/executor phải nằm trong adapter allowlist của sản phẩm. Một operation hợp lệ về schema vẫn có thể bị từ chối nếu không hỗ trợ trong turn-based combat/world use.
- Active Sát Chiêu/Gu House materialize snapshot effect; thay đổi Gu CSV không tự rewrite product đang active. Ownership, availability, hunger/reservation vẫn là lifecycle checks live.
- Component Gu reserved không dùng làm ingredient, trade/drop hay loot; worker/lifecycle có thể tháo product khi core chết do hunger và giải reservation.
- Số slot, mana, success/backfire/durability costs phải đọc preview/response và product profile.

## Kết quả và trường hợp thất bại

- Preview validation failure không tạo product.
- Fingerprint mismatch trả conflict; tải options/preview mới, xem lại thay đổi rồi commit lại.
- Component đang locked, dead, không owned, không active hoặc thiếu provenance/capability khiến preview/create/equip fail.
- Khi component mất trạng thái hợp lệ (ví dụ chết do hunger), product liên quan có thể bị auto-unequip/deactivate và reservation được giải phóng theo lifecycle code.
- Một lần dùng action có thể tiêu hao resource hoặc tạo backfire/durability consequences theo profile; xem response combat/world result.

## Liên hệ với hệ thống khác

- Cùng một Gu instance không thể chiếm dụng đồng thời bởi các active products.
- Combat loadout/session dùng Sát Chiêu như action theo lượt; không có combat simulation trên WebTransport.
- Khi Cổ Tiên dung nhập vào Tiên Cổ Phòng (`GuHouse.Active == true`), nhục thân ẩn vào trung tâm trận nhãn. Bảng chiêu thức trong lượt chiến đấu chỉ hiển thị và cho phép thôi động các chiêu thức Trận Kích của Tiên Cổ Phòng cùng lệnh `EXIT_GU_HOUSE` (Lệnh Rời Phòng). Toàn bộ Cổ kỹ và Sát chiêu cá nhân tạm thời bị khóa cho đến khi người chơi chủ động rời phòng hoặc Cổ Phòng bị công phá (Vỡ Phòng). Khi ở ngoài Cổ Phòng, người chơi sử dụng bộ Cổ kỹ cá nhân như bình thường và có thể chọn `ENTER_GU_HOUSE` (Dung Nhập) nếu Cổ Phòng chưa bị phá hủy.
- Gu House world effects dùng world `ON_USE` executor và có rule riêng, không tự áp dụng như passive stat toàn game.
- Effect definitions/catalog nằm trong [GAME_DATA.md](../guides/GAME_DATA.md); gameplay syntax xem [combat.md](combat.md).

## Dữ liệu/cấu hình ảnh hưởng

- `Gu.csv` Block v2 effects và `composition_contributions` xác định effect/source grants.
- Composition rule catalogs/revisions trong `backend/internal/gameplay/composition/` quy định edge/capability.
- Product profiles `killer_move.v1`/`gu_house.v1` quản lý component count, executor, context, target và saved/equipped limits.
- Active artifacts lưu composition graph/provenance/effect snapshot; reservations lưu product owner và Gu IDs.

## Mã nguồn liên quan

- Frontend: `frontend/src/components/KillerMovesPanel.tsx`, `GuHousePanel.tsx`; `frontend/src/store/slices/killerMovesSlice.ts`, `guHouseSlice.ts`.
- Backend: `backend/internal/handlers/killer_moves.go`, `gu_house.go`, `gu_house_use.go`, `gu_reservation.go`.
- Rules/data: `backend/internal/gameplay/composition/`, `backend/internal/effectcatalog/`, `backend/internal/db/schema_player_items.go`.

## API liên quan

- Sát Chiêu: `GET /killer-moves/options`, `GET /killer-moves`, `POST /killer-moves/preview`, `POST /killer-moves`, `POST /killer-moves/:id/equip`, `POST /killer-moves/:id/use`, `POST /killer-moves/:id/repair`, `DELETE /killer-moves/:id`.
- Tiên Cổ Ốc: `GET /gu-houses`, `POST /gu-houses/preview`, `POST /gu-houses`, `POST /gu-houses/:id/equip`, `POST /gu-houses/:id/use`, `POST /gu-houses/:id/repair`.

## Ví dụ flow

```text
GET options → draft component/effects → POST preview
→ inspect warnings/scale/fingerprint → POST create with accepted fingerprint
→ POST equip → use in allowed combat/world context
→ unequip/deactivate before moving or consuming a reserved Gu
```
