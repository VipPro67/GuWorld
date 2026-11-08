# Ticket 04: Bùa Lợi Duy Ngã Độc Tôn, Cơ Chế Phản Bội và Hóa Thân Phúc Địa Vô Chủ

Type: prototype
Status: resolved
Assignee: @antigravity
Blocked by: [02-aperture-instance-and-guest-matrix](02-aperture-instance-and-guest-matrix.md)

## Question

Làm thế nào để triển khai cơ chế đối kháng ma đạo sinh tử bên trong Tiên Khiếu, bùa lợi thiên địa của Chủ Khiếu và xử lý hậu quả khi Chủ Khiếu tử trận?

Cụ thể cần giải quyết các vấn đề:
1. **Bùa Lợi Duy Ngã Độc Tôn (Aperture Lord Sovereignty Buff)**:
   - Trong chính Tiên Khiếu của mình, Chủ Khiếu nhận trạng thái tối thượng: giảm 50% tiêu hao Tiên Nguyên cho mọi Cổ trùng và Sát chiêu, tăng 30% tốc độ hồi phục năng lượng, tăng kháng tính hiệu ứng khống chế (CC Resistance) thêm 40%, và tăng 25% uy lực sát thương tổng thể.
   - Cơ chế điều động Trận Pháp Thiên Địa để giam cầm hoặc trừng phạt kẻ địch ngay trên sân nhà.
2. **Cơ Chế Bội Ước & Chuyển Phe Phản Tặc (Traitor Rebellion Protocol)**:
   - Khách có thể chủ động chuyển trạng thái sang `REBEL / TRAITOR`: trở thành kẻ địch của Chủ Khiếu, có thể tấn công Chủ Khiếu và cướp bóc các điểm tài nguyên.
   - Cơ chế khế ước Tín Đạo bảo vệ: Nếu hai bên đã ký "Minh Ước Hộ Đạo" (Tín Đạo Thề Nguyện), hành vi phản bội sẽ ngay lập tức kích phát Phản Phệ Tín Đạo (mất máu, choáng ngợp hoặc tử vong tại chỗ tùy cấp bậc Minh Ước).
3. **Xử Lý Tử Vong Trong Tiên Khiếu (Fatality & Defeat Handling)**:
   - **Trường hợp Khách / Phản Tặc tử trận**: Bị coi như tử vong ngoài chiến trường thông thường; thi thể rơi rớt túi đồ/tiên tài vào kho của Chủ Khiếu; hồn phách hoặc linh thể bị tống khứ ra ngoài cổng thế giới.
   - **Trường hợp Chủ Khiếu tử trận**:
     - Nhục thân Cổ Tiên hoàn toàn tan rã.
     - Tiên Khiếu không biến mất mà lập tức cắm rễ vĩnh viễn tại tọa độ thế giới đó, chuyển trạng thái thành **Phúc Địa Vô Chủ (`UNOWNED_BLESSED_LAND`)**.
     - Sinh ra thực thể **Địa Linh (Land Spirit)** mang theo chấp niệm của tiền nhân, phong tỏa kho báu và đưa ra điều kiện thử thách (Quest di sản) cho bất kỳ Cổ Tiên nào muốn kế thừa toàn bộ phúc địa.

## Answer

Đã chuẩn hóa toàn diện bùa lợi sân nhà của Chủ Khiếu, cơ chế phản phệ Tín Đạo và quy trình hóa thân thành Phúc Địa Vô Chủ có Địa Linh chấp niệm. Kiểm chứng hoạt động hoàn tất qua prototype tương tác tại [04-aperture-sovereignty-and-traitor-pk.html](../prototypes/04-aperture-sovereignty-and-traitor-pk.html).

### 1. Bùa Lợi "Duy Ngã Độc Tôn" Trong Zone Sim 20Hz
- Khi thực thể người chơi là Chủ Khiếu hiện diện trong Zone `immortal_aperture:<aperture_id>` của mình, Zone Sim tự động kích hoạt bùa lợi tối thượng `sim.StatusApertureLordSovereignty`:
  - **Giảm $50\%$ tiêu hao Tiên Nguyên**: Hệ số mana tax giảm còn $0.5$ cho toàn bộ Cổ trùng và Sát chiêu.
  - **Tăng $30\%$ tốc độ hồi phục**: Gia tốc nhịp thở hồi phục Tiên Nguyên và Chân Nguyên.
  - **Tăng $40\%$ Kháng Khống Chế Cứng (Hard CC Resistance)**: Giảm $40\%$ thời lượng trúng các hiệu ứng Stun, Freeze, Silence, Root.
  - **Tăng $25\%$ Sát thương toàn diện**: Uy lực công kích tăng cường nhờ quy tắc bản nguyên địa phương.
- **Thẩm quyền Thiên Địa**: Chủ Khiếu có quyền thi triển lệnh `FormationSuppression` trói chân và khóa chiêu kẻ phản bội trong 3.0s.

### 2. Giao Thức Phản Bội & Minh Ước Tín Đạo Phản Phệ (Traitor & Alliance Contract)
- **Cơ sở dữ liệu**: Bổ sung bảng `aperture_alliance_contracts` trong `backend/internal/db/schema_aperture.go`:
  ```sql
  CREATE TABLE IF NOT EXISTS aperture_alliance_contracts (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      aperture_id UUID NOT NULL REFERENCES immortal_apertures(id) ON DELETE CASCADE,
      host_id UUID NOT NULL REFERENCES players(id),
      guest_id UUID NOT NULL REFERENCES players(id),
      contract_tier VARCHAR(20) NOT NULL DEFAULT 'MINOR', -- MINOR, MAJOR, ABSOLUTE
      is_active BOOLEAN NOT NULL DEFAULT TRUE,
      created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
  );
  ```
- **Hành vi phản bội**:
  - Khách có thể đổi `combat_stance` sang `TRAITOR`. Khi bật, khách trở thành kẻ địch với Chủ Khiếu và được phép tấn công cướp bóc các Điểm Tài Nguyên (`aperture_resource_points`).
  - **Phản Phệ Tín Đạo Tức Khắc**:
    * Nếu có ký Minh Ước (`MAJOR`): Kẻ bội ước lập tức bị trừ $50\%$ HP tối đa và dính Choáng (Stun) 3.0s.
    * Nếu ký `ABSOLUTE`: Kẻ bội ước mất $90\%$ HP và cạn kiệt toàn bộ Tiên Nguyên.
    * Nếu không ký Minh Ước (chủ khiếu chủ quan): Kẻ phản bội không chịu bất kỳ hình phạt Tín Đạo nào.

### 3. Xử Lý Tử Vong & Khởi Sinh Địa Linh Phúc Địa Vô Chủ
- **Khi Khách / Phản Tặc Tử Trận**:
  - Thi thể tan rã, linh thể bị tống khứ ra ngoài cổng thế giới.
  - Toàn bộ tài nguyên cướp bóc và một phần túi đồ rơi rớt trực tiếp vào kho dự trữ của Chủ Khiếu.
- **Khi Chủ Khiếu Tử Vong**:
  - Nhục thân Cổ Tiên hoàn toàn tan rã, hòa nhập làm dưỡng chất cho tiểu thiên địa.
  - Tiên Khiếu không bị xóa bỏ mà cắm rễ vĩnh viễn tại `(placed_region, placed_x, placed_y)`.
  - Cập nhật DB:
    ```sql
    UPDATE immortal_apertures
    SET state = 'UNOWNED_BLESSED_LAND', is_placed = true, updated_at = NOW()
    WHERE id = $1;
    ```
  - Cổng Môn Hộ ngoài thế giới đổi định danh `Kind: "unowned_blessed_land"`.
  - **Khởi Sinh Địa Linh (Land Spirit Manifestation)**:
    - Bổ sung bảng `aperture_land_spirits`:
      ```sql
      CREATE TABLE IF NOT EXISTS aperture_land_spirits (
          aperture_id UUID PRIMARY KEY REFERENCES immortal_apertures(id) ON DELETE CASCADE,
          spirit_name VARCHAR(100) NOT NULL,
          obsession_type VARCHAR(50) NOT NULL, -- PATH_MASTERY, SACRIFICE_BEASTS, DEFEAT_GUARDIAN
          obsession_desc TEXT NOT NULL,
          target_req_json JSONB NOT NULL,
          is_claimed BOOLEAN NOT NULL DEFAULT FALSE,
          claimed_by UUID REFERENCES players(id)
      );
      ```
    - Địa Linh (như Tiểu Hồ Tiên, Lang Gia Địa Linh) bảo vệ toàn bộ Tiên Cổ và tài nguyên di sản; thiết lập Điều Kiện Nhận Chủ (Quest kế thừa). Cổ Tiên nào hoàn thành điều kiện sẽ trở thành Tân Chủ Nhân của phúc địa.


## Question

Làm thế nào để triển khai cơ chế đối kháng ma đạo sinh tử bên trong Tiên Khiếu, bùa lợi thiên địa của Chủ Khiếu và xử lý hậu quả khi Chủ Khiếu tử trận?

Cụ thể cần giải quyết các vấn đề:
1. **Bùa Lợi Duy Ngã Độc Tôn (Aperture Lord Sovereignty Buff)**:
   - Trong chính Tiên Khiếu của mình, Chủ Khiếu nhận trạng thái tối thượng: giảm 50% tiêu hao Tiên Nguyên cho mọi Cổ trùng và Sát chiêu, tăng 30% tốc độ hồi phục năng lượng, tăng kháng tính hiệu ứng khống chế (CC Resistance) thêm 40%, và tăng 25% uy lực sát thương tổng thể.
   - Cơ chế điều động Trận Pháp Thiên Địa để giam cầm hoặc trừng phạt kẻ địch ngay trên sân nhà.
2. **Cơ Chế Bội Ước & Chuyển Phe Phản Tặc (Traitor Rebellion Protocol)**:
   - Khách có thể chủ động chuyển trạng thái sang `REBEL / TRAITOR`: trở thành kẻ địch của Chủ Khiếu, có thể tấn công Chủ Khiếu và cướp bóc các điểm tài nguyên.
   - Cơ chế khế ước Tín Đạo bảo vệ: Nếu hai bên đã ký "Minh Ước Hộ Đạo" (Tín Đạo Thề Nguyện), hành vi phản bội sẽ ngay lập tức kích phát Phản Phệ Tín Đạo (mất máu, choáng ngợp hoặc tử vong tại chỗ tùy cấp bậc Minh Ước).
3. **Xử Lý Tử Vong Trong Tiên Khiếu (Fatality & Defeat Handling)**:
   - **Trường hợp Khách / Phản Tặc tử trận**: Bị coi như tử vong ngoài chiến trường thông thường; thi thể rơi rớt túi đồ/tiên tài vào kho của Chủ Khiếu; hồn phách hoặc linh thể bị tống khứ ra ngoài cổng thế giới.
   - **Trường hợp Chủ Khiếu tử trận**:
     - Nhục thân Cổ Tiên hoàn toàn tan rã.
     - Tiên Khiếu không biến mất mà lập tức cắm rễ vĩnh viễn tại tọa độ thế giới đó, chuyển trạng thái thành **Phúc Địa Vô Chủ (`UNOWNED_BLESSED_LAND`)**.
     - Sinh ra thực thể **Địa Linh (Land Spirit)** mang theo chấp niệm của tiền nhân, phong tỏa kho báu và đưa ra điều kiện thử thách (Quest di sản) cho bất kỳ Cổ Tiên nào muốn kế thừa toàn bộ phúc địa.
