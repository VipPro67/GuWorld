# Chiến đấu

## Tính năng làm gì?

Combat là encounter theo lượt với state được lưu thành session. Player chọn action/Gu hoặc dùng auto-plan; simulator giải quyết lượt và handler lưu kết quả, phần thưởng và lịch sử.

## Điều kiện mở và quyền truy cập

Encounter thường bắt đầu từ world trigger, arena hoặc route combat tương ứng. Player phải không có session xung đột, có actor/loadout hợp lệ và đáp ứng region/PvP policy.

## Cách sử dụng

Vào encounter, chọn đội hình rồi bắt đầu điều khiển theo lượt hoặc mô phỏng tự động. Hiệu ứng kết thúc trận ngay lúc bắt đầu sẽ settle encounter và mở báo cáo kết quả mà không yêu cầu command. Trước khi bắt đầu session, có thể rút lui khỏi encounter; PvP không có thao tác rút lui ở màn hình chuẩn bị. Khi session hoạt động, đọc actor đang chờ và deadline, gửi command với UUID cùng expected revision. Chọn action hoặc plan; PvP dùng PLAN/AUTO và lượt chỉ giải quyết khi hai phía đã sẵn sàng. Sau mỗi response cập nhật session revision.

## Luật, chi phí và giới hạn

### Công thức sát thương

Đường damage action phổ thông trong simulator tính mỗi hit theo dạng:

```text
damageBeforeDefense = actionDamage * skillMultiplier * actorDamageMultiplier
                      * daoMultiplier * daoConflictMultiplier
                      * (1 + pathDamageBonus) * targetDaoResonance
```

Named buffs/mechanics có thể thêm multiplier; mỗi hit sau hit đầu nhân tiếp 0.7. Sau đó trừ defense đã áp dụng flat/percent ignore, nhân `(1-naturalResistance)*(1-pathResistance)` và có floor damage tối thiểu 1. Crit dùng actor crit bonus và multiplier 1.5. Evasion, virtual shield, damage reduction, last stand và effect blocks chạy thêm theo target/mechanic riêng. Công thức này chỉ mô tả ordinary action damage; không áp dụng nguyên xi cho mọi effect/direct HP change.

Gu combat power lấy flat damage/shield từ Active effect hoặc `ON_ACTION` flat operations. Trên player action, Gu power multiplier, Dao combat profile và path damage bonus được nhân vào; path mastery đã tích hợp vào DaoCombatProfile ở nhánh này. Hãy dùng combat breakdown/response để giải thích một action cụ thể.

### Bắt quái

Với monster có capture capability và người chơi đang equip Gu có `CAPTURE_BONUS`, handler tính `hpRatio = clamp(1 - enemyHP/enemyMaxHP, .1, 1)`, `hpFactor = sqrt(hpRatio)`. Type factor là 1 thường, .6 ELITE, 0 với BOSS/tribulation boss/monster. `daoFactor = 1 + sqrt(enslavementMarks)/100` nếu có marks dương, còn lại 1. Khi đó:

```text
chance = clamp((templateCaptureChance*hpFactor*typeFactor + Σ(captureBonus))*daoFactor,
               0.01, 1.0)
```

Gu capture bonus cao nhất phải có rank ít nhất bằng monster rank. Dao factor cũng nhân vào HP/attack/defense của pet vừa bắt. `templateCaptureChance` và capability đến từ monster data/equipped Gu; boss có type factor 0 nhưng Gu bonus vẫn được cộng trước khi clamp.

- Một player chỉ có một active combat session; participant cũng được ràng buộc duy nhất.
- Command chỉ được nhận từ actor đang chờ. UUID lặp có tính idempotent; revision cũ bị conflict.
- Session giữ snapshot/loadout cần thiết để giải quyết nhất quán; combat không chạy trên WebTransport/WebSocket.
- PvP plan có timeout 45 giây; timeout có thể kích hoạt auto action.
- Mana, energy, cooldown, Gu status và effect context được kiểm tra tại server/simulator.
- Khi dung nhập Tiên Cổ Phòng (`GuHouse.Active == true`), combatant chỉ có thể thi triển 6 chiêu thức Trận Kích của Cổ Phòng và lệnh `EXIT_GU_HOUSE`; Cổ kỹ và Sát chiêu cá nhân bị khóa hoàn toàn. Khi ở ngoài Cổ Phòng, combatant dùng Cổ kỹ cá nhân và có thể dùng `ENTER_GU_HOUSE` nếu Cổ Phòng chưa vỡ.
- Đòn đánh phá vỡ Form HP của Tiên Cổ Phòng áp dụng quy tắc 100% No Spillover (không tràn sát thương vào nhục thân tại lượt đó). Cổ Tiên bị văng ra chịu phản phệ mất 30% Max HP nhục thân và bị Choáng 2 lượt. Độ bền Cổ Phòng về 0 và tất cả Cổ hạch tâm bị trừ 30% độ bền trong database.

## Kết quả và trường hợp thất bại

Command sai lượt, revision cũ hoặc action không hợp lệ không được áp dụng. Kết thúc encounter sẽ settle kết quả, cập nhật người chơi/Gu, loot và history theo loại encounter.

## Liên hệ với hệ thống khác

Movement, arena, capture, party/pet, Sát Chiêu và Tiên Cổ Ốc có thể cung cấp encounter/actor/action. Simulator là nguồn luật lượt; realtime chỉ đồng bộ thế giới.

## Dữ liệu/cấu hình ảnh hưởng

Combat profiles, Gu effects, monsters, arena rules và rank scaling ảnh hưởng action/result; xem [GAME_DATA.md](../guides/GAME_DATA.md).

## Mã nguồn liên quan

- Engine: `backend/internal/simulator/`.
- Handlers: `backend/internal/handlers/combat_*.go`.
- Backend: session lifecycle `backend/internal/handlers/combat_session.go`, `combat_lifecycle.go`, Gu scaling `combat_effects.go`, simulation adapter `combat_sim_build.go`, settlement `combat_finalize.go`.
- Frontend: `frontend/src/components/CombatScreen.tsx`, `frontend/src/store/slices/combatSlice.ts`.

## API liên quan

Combat/session/action/history/retreat routes được đăng ký tại `backend/internal/server/server.go`; xem [API guide](../guides/API.md).

## Ví dụ flow

```text
world encounter → tạo session/revision 1 → gửi command UUID + revision 1
→ simulator resolve → nhận revision mới → tiếp tục đến khi settle thắng/thua/flee
```
