# Ticket 01: Chuẩn Hóa 4 Archetype Tai Kiếp & Máy Trạng Thái Vòng Đời Độ Kiếp 3 Giai Đoạn

Type: prototype
Status: resolved
Blocked by: none
Assignee: Antigravity

## Question

Làm thế nào để tái cấu trúc mô hình dữ liệu (`tribulation_events`, `tribulation_enemies`, `immortal_apertures`) và hệ thống kích hoạt (`tribulation_trigger.go`) từ việc chỉ spawn boss đấm chay sang quản lý **4 Archetype Tai Kiếp** cùng **Vòng Đời 3 Giai Đoạn**?

Cụ thể cần giải quyết:
1. **Mô hình hóa 4 Archetype Tai Kiếp**:
   - `ENVIRONMENTAL` (Thiên Tai Sinh Thái: bão tuyết, mưa lửa, độc vụ, địa chấn).
   - `BEAST_SWARM` (Thú Triều Xâm Lăng: dị thú, trùng triều tấn công theo đợt).
   - `MIND_SOUL` (Tâm Ma Khảo Nghiệm: ngã rẽ tâm cảnh, thức hải quyết đấu).
   - `ANOMALOUS_COMBAT` (Dị Tượng Sàn Đấu: chiến đấu với thần ma dưới hiệu ứng sàn đấu đặc thù).
   - Công thức lựa chọn Archetype dựa trên Rank (Lục Chuyển Địa Tai, Thất Chuyển Thiên Kiếp, Bát Chuyển Hạo Kiếp, Cửu Chuyển Vạn Kiếp / Hỗn Độn) kết hợp với Lưu Phái chủ tu / Đạo Ngân Tiên Khiếu.
2. **Vòng đời 3 Giai Đoạn (3-Phase Lifecycle State Machine)**:
   - **Phase 1: Điềm Báo & Chuẩn Bị (`HARBINGER` / `PREPARATION`)**: Cảnh báo trước khi kiếp nạn giáng lâm; cho phép Cổ Tiên gia cố trận pháp, chuyển dịch tài nguyên hoặc chuẩn bị sát chiêu.
   - **Phase 2: Nghênh Kiếp (`ORDEAL_ACTIVE`)**: Giai đoạn thực thi hành vi ứng phó tương ứng với từng Archetype (giải cứu môi trường / phòng thủ thú triều / đàm phán tâm ma / chiến đấu).
   - **Phase 3: Dư Ba & Thu Hoạch (`AFTERMATH`)**: Dọn dẹp chiến trường, thu hoạch Kiếp Tài rơi rớt trên lưới ô, thanh tẩy ô nhiễm Đạo Ngân, kết toán Đạo Ngân thưởng.
3. **Mở rộng Schema Database**:
   - Cột `archetype`, `stage`, `metadata` (JSONB lưu trữ tọa độ ảnh hưởng, danh sách đợt thú triều, tiến trình tâm ma) trên bảng `tribulation_events`.
   - Trạng thái đồng bộ với SSE / WebTransport để client hiển thị đúng overlay tương ứng.

## Answer

Đã chuẩn hóa toàn diện mô hình dữ liệu, ma trận phân bổ Archetype và máy trạng thái (State Machine) vòng đời 3 giai đoạn của hệ thống Tai Kiếp:

### 1. Phân Bổ 4 Archetype & Thuật Toán Tuyển Chọn
- **4 Archetype Chuẩn Hóa**:
  1. `ENVIRONMENTAL`: Thiên tai khí tượng & sinh thái (Bão tuyết, Mưa lửa, Độc vũ, Địa chấn).
  2. `BEAST_SWARM`: Thú triều & Trùng họa tấn công cứ điểm theo đợt (Waves).
  3. `MIND_SOUL`: Tâm ma khảo nghiệm (Ngã rẽ đạo tâm, Quyết đấu Thức Hải chống lại Tâm Ma Bản Thể).
  4. `ANOMALOUS_COMBAT`: Chiến đấu Turn-based với Thần Ma dưới quy tắc sàn đấu biến dị (Cấm pháp, lôi đình giáng phạt, Đạo Ngân Xung Đột).
- **Tỷ Lệ Phân Bổ Theo Rank**:
  - Rank 6 (Địa Tai): 40% `ENVIRONMENTAL`, 30% `BEAST_SWARM`, 15% `MIND_SOUL`, 15% `ANOMALOUS_COMBAT`.
  - Rank 7 (Thiên Kiếp): 25% mỗi loại.
  - Rank 8 (Hạo Kiếp): 20% `ENVIRONMENTAL`, 20% `BEAST_SWARM`, 30% `MIND_SOUL`, 30% `ANOMALOUS_COMBAT`.
  - Rank 9 (Vạn Kiếp / Hỗn Độn Kiếp): 25% mỗi loại (cấp độ Hỗn Độn Phá Diệt).
- **Hệ Số Điều Chỉnh Thiên Ý Thích Ứng (Adaptive Bias)**:
  - Nếu số mỏ tài nguyên `aperture_resource_points >= 6`: Tăng 15% tỷ lệ `BEAST_SWARM` hoặc `ENVIRONMENTAL` (Thiên Ý nhắm vào điểm yếu kinh tế).
  - Nếu `wisdom_attainment < 2` hoặc `soul_hp < 70%`: Tăng 20% tỷ lệ `MIND_SOUL` (Thiên Ý nhắm vào sơ hở tâm cảnh).
  - Nếu có trợ thủ Trợ Trận Độ Kiếp (`helperCount > 0`): Tự động chuyển dịch sang `CHAOTIC_HYBRID` và `ANOMALOUS_COMBAT` để tạo áp lực diện rộng.

### 2. Máy Trạng Thái Vòng Đời 3 Giai Đoạn (Lifecycle State Machine)
- **Phase 1: Điềm Báo (`HARBINGER`)**:
  - Kích hoạt khi chu kỳ thời gian Tiên Khiếu chạm mốc (`InternalYearsPassed >= NextTribulationYear`).
  - Gán `stage = 'HARBINGER'`, thiết lập thời gian đếm ngược chuẩn bị 120 giây (`preparation_ends_at = NOW() + 120s`).
  - Gửi thông báo SSE hiển thị dị tượng thời tiết điềm báo trên `ImmortalApertureView.tsx`.
  - Cho phép Cổ Tiên: Dùng Tiên Nguyên Thạch gia cố ô trận pháp (`fortifyFormation`), sơ tán dị nhân, hoặc chuẩn bị sát chiêu tương khắc.
- **Phase 2: Nghênh Kiếp (`ORDEAL_ACTIVE`)**:
  - Chuyển sang khi Cổ Tiên bấm "Tiến Vào Nghênh Kiếp" hoặc khi hết 120 giây chuẩn bị.
  - Vận hành cơ chế tương ứng theo Archetype:
    - `ENVIRONMENTAL`: Cổ Tiên thi triển Sát Chiêu tương khắc để làm tan/dập tắt các ô nguy hiểm (`hazard_cells`).
    - `BEAST_SWARM`: Tháp trận pháp và dị nhân hộ vệ đánh chặn các Wave quái vật tràn vào.
    - `MIND_SOUL`: Vấn tâm đạo tâm qua cây ngã rẽ lựa chọn; nếu ý chí lung lay chuyển sang Quyết Đấu Thức Hải.
    - `ANOMALOUS_COMBAT`: Xuất trận Turn-based tiêu diệt Thần Ma, phá vỡ tụ lực kiếp lôi.
  - Chu kỳ tick dã ngoại nếu không kịp thời hóa giải sẽ trừ sinh lực Tiên Khiếu (`aperture_hp`).
  - **Điều kiện thắng**: Hóa giải 100% mối nguy ➔ Chuyển sang Phase 3 (`AFTERMATH`).
  - **Điều kiện bại**: `aperture_hp <= 5%` hoặc Cổ Tiên tử trận ➔ Chuyển sang `FAILED`, Tiên Khiếu rơi vào trạng thái `DAMAGED`.
- **Phase 3: Dư Ba & Thu Hoạch (`AFTERMATH`)**:
  - Các ô tham chiến kết tinh thành các **Điểm Thu Hoạch Kiếp Tài** (`spoils_cells`) chứa các nguyên liệu quý từ `Item.csv`.
  - Ô đất bị biến dị thành đất ô nhiễm Đạo Ngân (`polluted_cells`).
  - Cổ Tiên nhấp thu gom Kiếp Tài và tiến hành Thanh Tẩy (`cleansePollution`) hoặc giữ lại để đổi hướng sinh thái.
  - Bấm "Hoàn Tất Kết Toán" ➔ Chuyển trạng thái sang `RESOLVED`, cộng thưởng `reward_dao_marks`.

### 3. Mở Rộng Schema Database DDL (`schema_aperture.go`)
```sql
ALTER TABLE tribulation_events
    ADD COLUMN IF NOT EXISTS archetype VARCHAR(30) NOT NULL DEFAULT 'ANOMALOUS_COMBAT',
    ADD COLUMN IF NOT EXISTS stage VARCHAR(30) NOT NULL DEFAULT 'ORDEAL_ACTIVE',
    ADD COLUMN IF NOT EXISTS preparation_ends_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS hazard_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS spoils_collected BOOLEAN NOT NULL DEFAULT FALSE;
```
- Cấu trúc `hazard_metadata`:
```json
{
  "hazard_cells": [0, 1, 5, 6],
  "fortified_cells": [12],
  "waves_remaining": 4,
  "current_wave": 1,
  "mind_step": 0,
  "shadow_hp": 100,
  "boss_anomalies": ["FIELD_LIGHTNING", "GU_SEAL_FIRE"],
  "spoils": [
    {"cell": 0, "item_id": "ice_crystal_rank6", "amount": 2},
    {"cell": 1, "item_id": "ice_cicada_wing_rank6", "amount": 1}
  ],
  "polluted_cells": [0, 1]
}
```

### 4. Kiểm Chứng Trực Quan Qua Prototype HTML
- Đã xây dựng hoàn chỉnh bản thử nghiệm tương tác độc lập tại:
  [`prototypes/01_tribulation_multiform_lifecycle.html`](../prototypes/01_tribulation_multiform_lifecycle.html).
- Chạy thử nghiệm thành công 4 Walkthrough Scenarios:
  1. *Bão Tuyết Địa Tai (Environmental - Rank 6)*.
  2. *Huyết Dơi Xâm Lăng (Beast Swarm - Rank 7)*.
  3. *Ngũ Uẩn Vấn Tâm & Tâm Ma Thức Hải (Mind Soul - Rank 7)*.
  4. *Thiên Kiếp Thần Ma (Anomalous Combat - Rank 7)*.
