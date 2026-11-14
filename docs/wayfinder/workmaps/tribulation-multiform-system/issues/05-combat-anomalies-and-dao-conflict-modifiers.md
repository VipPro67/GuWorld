# Ticket 05: Sàn Đấu Dị Tượng & Xung Đột Đạo Ngân Trong Simulator Theo Lượt

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Làm thế nào để nâng cấp trận đánh Boss/Thần Ma trong Turn-based Simulator (`backend/internal/simulator/`, `InteractiveCombatControls.tsx`) thành dạng tai kiếp **Dị Tượng Sàn Đấu (`ANOMALOUS_COMBAT`)** với các quy tắc môi trường độc dị và cơ chế Đạo Ngân Xung Đột thay vì đấm chay?

Cụ thể cần giải quyết:
1. **Trường Môi Trường Chiến Trường (Battlefield Hazard & Weather Modifiers)**:
   - Tích hợp trường `HazardEffect` trên `CombatState` và `CombatantAuto`:
     * *Lôi Đình Chi Địa*: Mỗi đầu turn có 30% tỷ lệ sét đánh ngẫu nhiên vào 1 slot chiến trận gây sát thương chuẩn và trừ mana.
     * *Bão Tuyết Đóng Băng*: Giảm 20% tốc độ hành động và tăng tiêu hao Chân Nguyên/Tiên Nguyên lên 1.5 lần.
     * *Cấm Pháp Tuyệt Vực*: Khóa hoàn toàn một nhánh Cổ trùng (ví dụ: cấm Cổ Trùng hệ Hỏa khi độ Thủy Kiếp).
2. **Cơ Chế Đạo Ngân Xung Đột (Dao Mark Conflict)**:
   - Thiên Kiếp mang Đạo Ngân tương khắc với Lưu Phái Chủ Tu của Cổ Tiên.
   - Sát thương từ các sát chiêu dị hệ bị suy giảm theo mật độ Đạo Ngân đối kháng (`1.0 / (1.0 + conflict_factor * DaoMarks)`).
   - Đòn đánh của Thần Ma mang Đạo Ngân đối kháng sẽ xuyên thấu qua các tầng phòng ngự thông thường nếu Cổ Tiên không sở hữu sát chiêu phòng hộ đặc thù.
3. **Phase Biến Hóa Của Thần Ma Kiếp Nạn**:
   - Boss không còn chỉ dùng đánh thường: Boss sở hữu bộ kỹ năng sát chiêu Thiên Đạo (ví dụ: Tụ Lực Kiếp Lôi 2 lượt, Phân Thân Huyễn Ảnh, Hấp Thu Tiên Nguyên Khiếu Vách).
   - Giao diện HUD chiến đấu thể hiện rõ cảnh báo cơ chế boss (Intent Telegraphing & Weather Forecast).

## Answer

Đã hoàn thành chuẩn hóa thiết kế và kiểm chứng logic cho Archetype **`ANOMALOUS_COMBAT`** nâng cấp Turn-based Combat Simulator:

### 1. Kiến Trúc Mở Rộng State Simulator (`backend/internal/simulator/models.go`)
- **Battlefield Hazard State**:
  ```go
  type CombatHazard struct {
      Type           string          `json:"type"`            // FIELD_LIGHTNING, PERMAFROST, GU_RESTRICTION, DAO_REVERSAL
      Severity       int             `json:"severity"`        // 1 to 5
      ForbiddenPaths []string        `json:"forbidden_paths"` // E.g. ["fire"] in Water tribulation
      Params         json.RawMessage `json:"params"`          // Interval damage, mana drain multiplier
  }
  ```
- **Can thiệp Turn-Loop (`TurnStartHooks` & `ActionFilter`)**:
  - `FIELD_LIGHTNING`: Đầu lượt kiểm tra xác suất nổ sét giáng phạt gây sát thương chuẩn và trừ Tiên Nguyên trực tiếp vào nhục thân.
  - `PERMAFROST`: Nhân hệ số tiêu hao Tiên Nguyên khi xuất chiêu lên $1.5\times$.
  - `GU_RESTRICTION`: Validate action trước khi thi triển; cấm hoàn toàn kích hoạt Cổ trùng thuộc lưu phái bị áp chế.

### 2. Mô Hình Xung Đột Đạo Ngân (Dao Mark Conflict) & Chí Tôn Tiên Thể
- **Tính toán suy giảm sát thương dị hệ**:
  Tích hợp vào `backend/internal/simulator/dao_damage.go`:
  $$\text{DamageModifier} = \frac{1.0}{1.0 + 0.00015 \times (\text{PlayerDaoMarks} + \text{BossDaoMarks})}$$
  Khi Cổ Tiên Hỏa Đạo thi triển Sát chiêu Thủy Đạo vào Thần Ma Kiếp Lôi, sát thương bị triệt tiêu từ 50% đến 75% do Đạo Ngân ma sát kịch liệt.
- **Cộng hưởng Lưu Phái Bản Mệnh**:
  $$\text{BonusDamage} = \text{BaseDamage} \times \left(1.0 + \frac{\text{PlayerDaoMarks}}{1000} \times 0.05\right)$$
- **Ngoại Lệ Tuyệt Đối - Chí Tôn Tiên Thể (`HasNonConflictingDao = true`)**:
  Hoàn toàn bỏ qua công thức xung đột Đạo Ngân, sát thương dị hệ giữ nguyên 100% uy lực tối đa, tái hiện chuẩn xác huyền thoại Cổ Chân Nhân.

### 3. Cơ Chế Boss Intent Telegraphing & Stance Break (Phá Tụ Lực)
- **Boss Intent Telegraphing**:
  Trước mỗi lượt hành động, Boss phát đi tín hiệu ý đồ:
  * `NORMAL_ATTACK`: Đòn đánh ma vật thông thường.
  * `CHARGING_LIGHTNING`: Tụ lực sát chiêu hủy diệt (ví dụ `Cửu Thiên Diệt Thế Lôi` - 1,200 dmg ở lượt kế tiếp), hiển thị thanh Break Threshold (1,500 dmg).
  * `SIPHON_APERTURE`: Hút sinh khí vách khiếu hồi phục HP.
- **Cơ chế Phá Tụ Lực (Stance Break)**:
  Nếu trong lượt tụ lực, Cổ Tiên dồn sát thương vượt mốc `BreakThreshold`, Boss lập tức rơi vào trạng thái `STUNNED`, mất đòn tụ lực và bỏ lượt kế tiếp.
- **Phòng Hộ Tiên Cổ Phòng (`Gu House Form`)**:
  Cổ Tiên có thể chuyển đổi dung nhập vào Tiên Cổ Phòng để phân tán 100% sát thương chí mạng vào kết cấu Cổ Phòng, bảo toàn tính mạng nhục thân.

### 4. Kiểm Chứng Logic Qua Prototype Tương Tác
- Toàn bộ cơ chế (Hazard trigger, Dao conflict, Boss 2-turn intent, Stance break, Gu House switch, Sovereign Body toggle) đã được hiện thực hóa và kiểm chứng hoạt động hoàn hảo tại prototype:
  [05_combat_anomalies_and_dao_conflict_demo.html](file:///home/vippro67/GitHub/GuWorld/docs/wayfinder/workmaps/tribulation-multiform-system/prototypes/05_combat_anomalies_and_dao_conflict_demo.html).
