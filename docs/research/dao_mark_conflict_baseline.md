# Báo Cáo Nghiên Cứu: Khảo Sát Dữ Liệu Đạo Ngân & Mô Phỏng Baseline Xung Đột

**Ticket:** #177 (Nằm trong Map #176: *Hệ Thống Vạn Đạo Tương Xung & Trừng Phạt Đa Tu*)  
**Mục tiêu:** Khảo sát phân bố dữ liệu Đạo Ngân thực tế (người chơi, quái vật/boss), giải mã công thức xung đột hiện tại trong codebase, và mô phỏng số học tác động suy giảm sát thương khi hạ sàn xung đột từ 0.60 xuống 0.05–0.10 kết hợp hệ số phạt đa tu.  
**Ngày thực hiện:** Tháng 10, 2026  
**Trạng thái:** Hoàn thành (Findings Consolidate)

---

## 1. Tóm Tắt Thực Thi (Executive Summary)

1. **Phân bố dữ liệu người chơi:**
   - Trong database, Đạo Ngân được lưu duy nhất tại cột `immortal_apertures.dao_marks` (JSONB), chỉ dành cho Cổ Tiên từ Cổ Sư Cửu Chuyển thăng Tiên (Rank $\ge$ 6). Bảng `players` không lưu trữ cột `dao_marks` trực tiếp mà chỉ lưu điểm cảnh giới lưu phái `path_masteries` (JSONB).
   - Cơ chế độ kiếp hiện tại (`tribulation_dao.go`) tự động phân bổ **70% Đạo Ngân vào lưu phái chính** và **30% còn lại rải ngẫu nhiên vào 1–2 lưu phái khác**. Do đó, mọi người chơi sau nhiều lần độ kiếp đều **tự nhiên tích lũy tạp tu (nhiễm từ 2 đến 10+ lưu phái dị chủng)** nếu không có cơ chế tẩy rửa.

2. **Phát hiện quan trọng về công thức hiện tại (`dao_combat_profile.go`):**
   - **Sàn 0.60 hiện tại chưa bao giờ bị chạm đến trong thực tế!**
   - Nguyên nhân cốt lõi: Mẫu số tính toán tỷ lệ xung đột bị cộng thêm một hằng số bảo hộ khổng lồ $+50,000$ (`conflictRatio = effConflict / (amt + 50000)`).
   - Ở Rank 6 (khoảng 10,000 – 40,000 Đạo Ngân), mẫu số bị phóng đại gấp 2 đến 17 lần so với số Đạo Ngân thực tế của chiêu thức.
   - Thêm vào đó, công thức hiện tại **hoàn toàn bỏ qua 2 biến số sống còn**: Số lượng lưu phái (`UniquePathCount`) và Độ thuần khiết (`Purity`).
   - Hậu quả: Một nhân vật **Tạp tu 10 phái chia đều** (Purity chỉ 10%) hiện tại chỉ bị phạt vẻn vẹn **13% sát thương** (`conflictMult = 0.8700`), vi phạm nghiêm trọng nguyên lý "chuyên tu là vương đạo, tạp tu tự tìm đường chết" trong thế giới Cổ Chân Nhân.

3. **Mô phỏng suy giảm sát thương khi hạ sàn (0.05–0.10) & phạt Đa Tu:**
   - **Nếu chỉ hạ sàn (xuống 0.10 hoặc 0.05) mà giữ nguyên công thức hiện tại:** Mức suy giảm sát thương là **0%** trên mọi mẫu nhân vật (vì không có build nào rơi xuống dưới 0.70). Sàn bảo hộ mới sẽ hoàn toàn vô hiệu!
   - **Khi áp dụng chuẩn hóa mẫu số và hệ số phạt Đa Tu lũy tiến theo Purity & UniquePathCount:**
     - **Đơn tu thuần chủng (1 phái, 100% Purity):** Giữ nguyên $100\%$ công lực (suy giảm 0%).
     - **Song tu tương sinh (2 phái, 70/30):** Suy giảm **31.3%** sát thương (cần cơ chế Tông Sư giải trừ tại #180).
     - **Song tu tương khắc (2 phái, 50/50 Hỏa - Thủy):** Suy giảm **57.2%** sát thương (từ `0.8575` rớt xuống `0.3671`).
     - **Tứ tu kiếp nạn (4 phái, 70/10/10/10):** Suy giảm **55.1%** sát thương.
     - **Tứ tu chia đều (4 phái x 25%):** Suy giảm **81.2%** sát thương (từ `0.8647` rớt xuống `0.1625`).
     - **Tạp tu 10 phái cực đoan (10 phái x 10%):** Suy giảm **90.8%** sát thương, chạm đáy sàn bảo hộ $0.08$ (tương đương bị phế gần như toàn bộ uy lực Tiên Cổ).

4. **Tác động lên PVE (Boss & Quái vật):**
   - Trong `Monster.csv`, toàn bộ 209 quái vật đều có `dao_marks: {}`.
   - Cơ chế sinh động `ScaleAndRandomizeMonsterDaoMarks`:
     * Quái Normal/Elite: Luôn sinh **1 lưu phái** $\implies$ Đơn tu thuần $100\%$ $\implies$ **Không bị dính bất kỳ xung đột nào**.
     * Boss: Sinh ngẫu nhiên **2 lưu phái** (50/50). Hiện tại nếu Boss roll trúng 2 phái tương khắc (ví dụ Hỏa + Thủy), Boss chỉ bị phạt $14.25\%$. Nhưng với mô hình mới, Boss sẽ bị suy giảm tới $57.2\%$ sát thương do nội tại tự xung đột.
     * Kiến nghị: Cần gán cờ `HasNonConflictingDao = true` cho Boss / Hoang thú, hoặc bắt buộc thuật toán sinh 2 phái của Boss phải chọn cặp **tương sinh** (Synergistic pair).

---

## 2. Khảo Sát Kiến Trúc Dữ Liệu Gốc

### 2.1. Lưu trữ Đạo Ngân Người Chơi
- **Database Schema:**
  - Bảng `players` (`backend/internal/db/schema_player.go`): Không có cột `dao_marks`. Chỉ lưu:
    * `path_masteries` (`JSONB DEFAULT '{}'`): Điểm lĩnh ngộ lưu phái (Mastery points dùng để suy diễn cấp độ Cảnh giới từ Phổ thông đến Vô Thượng Đại Tông Sư).
    * `constitution`: Thể chất nhân vật (ví dụ Thập Tuyệt Thể, Chí Tôn Tiên Thai).
  - Bảng `immortal_apertures` (`backend/internal/db/schema_aperture.go`):
    * `dao_marks` (`JSONB NOT NULL DEFAULT '{"strength": 0}'`): Nơi lưu trữ duy nhất của Đạo Ngân người chơi.
- **In-Memory Models & Data Flow:**
  - `PlayerSvc.ProjectStatus` (`backend/internal/services/player_service.go`):
    * Khi `player.Rank >= 6`: Query `SELECT dao_marks FROM immortal_apertures WHERE player_id = $1` và nạp vào struct field `player.DaoMarks`.
    * Khi `player.Rank < 6`: Mặc định gán `player.DaoMarks = {"strength": 0}` (Phàm nhân không có Đạo Ngân).
  - Endpoint `GET /api/player/daongan` (`backend/internal/handlers/player_daongan.go`):
    * Nạp `immortal_apertures.dao_marks` và `players.path_masteries`.
    * Gọi `gamefx.BuildDaoCombatProfile(daoMap, pathMasteries, charFX.NonConflictingDaoMarks, charFX.DaoNganBonus)`.

### 2.2. Cơ Chế Tích Lũy Đạo Ngân Qua Thiên Kiếp (Nguồn gốc tạp tu tự nhiên)
Trong `backend/internal/handlers/tribulation_dao.go`:
```go
func distributeDaoMarksFromPaths(mainPath string, totalMarks int64, paths []string) map[string]int64 {
    // 1. Lưu phái chính nhận 70% tổng Đạo Ngân
    mainAmount := int64(float64(totalMarks) * 0.7)
    rewards[mainPath] = mainAmount
    
    remaining := totalMarks - mainAmount // 30% còn lại
    // 2. Chia ngẫu nhiên cho 1 hoặc 2 lưu phái ngoại lai khác
    numExtra := rand.Intn(2) + 1 
    if numExtra == 1 {
        rewards[otherPaths[0]] = remaining
    } else {
        rewards[otherPaths[0]] = remaining / 2
        rewards[otherPaths[1]] = remaining - remaining / 2
    }
    return rewards
}
```
**Hệ quả gameplay:**
- Cứ mỗi lần vượt qua Địa Tai / Thiên Kiếp / Hạo Kiếp, người chơi nhận 30% Đạo Ngân ngoại hệ ngẫu nhiên.
- Sau 10 lần độ kiếp, nếu người chơi không có công cụ tẩy rửa ("Thanh Đạo Tẩy Thể" - Ticket #182), Tiên Khiếu sẽ tự nhiên tích lũy từ 4 đến 10+ lưu phái dị chủng!

### 2.3. Dữ Liệu Quái Vật & Boss Mẫu
- **Database / CSV:**
  - `backend/data/Monster.csv`: Có tổng cộng **209 mẫu quái vật**.
  - Tất cả 209 dòng đều có trường `dao_marks: "{}"` (chưa gán Đạo Ngân tĩnh).
- **Cơ Chế Dynamic Scaling (`backend/internal/models/monster.go`):**
  - Hàm `ScaleAndRandomizeMonsterDaoMarks(mon, rank, subRank)`:
    * Nếu quái có `rank < 6`: Không có Đạo Ngân (`DaoMarksMap{}`).
    * Nếu `rank >= 6`:
      + Rank 6: 10,000 – 40,000 Đạo Ngân.
      + Rank 7: 40,000 – 220,000 Đạo Ngân.
      + Rank 8: 220,000 – 1,400,000 Đạo Ngân.
      + Rank 9: 8,500,000 – 580,000,000+ Đạo Ngân.
    * Phân bổ lưu phái:
      + **Normal / Elite:** Chọn ngẫu nhiên $1$ lưu phái (`numPaths = 1`) $\implies$ Đơn tu $100\%$ Purity.
      + **Boss:** Chọn ngẫu nhiên $2$ lưu phái (`numPaths = 2`) với trọng số chia đều $50\% / 50\%$.

---

## 3. Giải Mã Công Thức Xung Đột Hiện Tại (`dao_combat_profile.go`)

### 3.1. Các Bước Tính Toán Hiện Hành
Khi thực hiện đòn đánh thuộc lưu phái $p$ với số Đạo Ngân $amt = \text{cleanDaoMarks}[p]$:
1. **Grace Threshold (Vùng đệm an toàn):**
   $$\text{graceThreshold} = \max\left(1000, \lfloor \text{totalMarks} \times 0.02 \rfloor\right)$$
   Mọi lưu phái ngoại lai có Đạo Ngân $\le \text{graceThreshold}$ đều bị loại khỏi danh sách gây xung đột.
2. **Raw Conflict (Xung đột thô):**
   $$\text{RawConflict} = \sum_{q \neq p, \text{amt}_q > \text{grace}} \text{amt}_q \times (1.0 - \text{Compatibility}(p, q))$$
   Với $\text{Compatibility}(p, q) \in [0.55, 1.00]$ tra từ ma trận nhóm ($10 \times 10$) và 14 cặp Override.
3. **Effective Conflict (Kiểm soát bởi Cảnh Giới Lưu Phái):**
   $$\text{EffConflict} = \text{RawConflict} \times (1.0 - \text{RealmControl})$$
   (Phổ thông: $0\%$, Tông Sư L1: $10\%$, Đại Tông Sư L2: $25\%$, Chuẩn Vô Thượng L3: $45\%$, Vô Thượng L4: $80\%$).
4. **Conflict Ratio (Tỷ lệ xung đột):**
   $$\text{ConflictRatio} = \frac{\text{EffConflict}}{amt + 50000}$$
5. **Conflict Multiplier (Hệ số xung đột đầu ra):**
   $$\text{ConflictMult}_{\text{raw}} = \frac{1}{1.0 + \text{ConflictRatio} \times 0.8 \times 2.0} = \frac{1}{1.0 + 1.6 \times \text{ConflictRatio}}$$
   $$\text{ConflictMult} = \max\left(0.60, \text{ConflictMult}_{\text{raw}}\right)$$

### 3.2. Điểm Nghẽn & Khuyết Tật Của Mô Hình Hiện Tại

| Vấn đề | Biểu hiện toán học | Hậu quả gameplay |
|---|---|---|
| **Lạm phát mẫu số $+50,000$** | $\text{Ratio} = \frac{\text{EffConflict}}{amt + 50000}$ | Ở Rank 6 ($amt \approx 10,000$), mẫu số thành $60,000$. Tỷ lệ xung đột bị thu nhỏ tới 6 lần! |
| **Thiếu vắng biến số $N$ (Unique Paths)** | Công thức chỉ cộng dồn tuyến tính $\sum \text{amt}_q$ | Người tu 2 phái hay tu 10 phái có cùng tổng Đạo Ngân ngoại hệ thì chịu mức phạt y hệt nhau. |
| **Thiếu vắng biến số Purity** | Không quan tâm tỷ lệ phái chính trên tổng thể | Nhân vật có Purity $10\%$ chỉ bị phạt $13\%$, không hề có cảm giác "nghẽn mạch kinh mạch/hỗn loạn đạo ngân". |
| **Sàn $0.60$ trở thành "vô hình"** | $\text{ConflictMult}_{\text{raw}}$ hiếm khi $< 0.70$ | Sàn $0.60$ chưa từng phát huy tác dụng khống chế trần phạt, vì bản thân công thức chưa bao giờ kéo chỉ số xuống tới mức đó. |

---

## 4. Mô Phỏng Số Học: So Sánh Baseline Hiện Tại vs Đề Xuất Mới

### 4.1. Định Nghĩa 12 Profile Nhân Vật Điển Hình

1. **ARC-01 (Đơn Tu Rank 6):** $30,000$ Hỏa Đạo ($100\%$ Purity, $N=1$).
2. **ARC-02 (Đơn Tu Rank 7):** $100,000$ Hỏa Đạo ($100\%$ Purity, $N=1$, Cảnh giới L1 Master).
3. **ARC-03 (Song Tu Tương Sinh 50/50):** $15,000$ Hỏa, $15,000$ Mộc ($\text{comp} = 0.95$, $50\%$ Purity, $N=2$).
4. **ARC-04 (Song Tu Tương Sinh 70/30):** $21,000$ Hỏa, $9,000$ Mộc (Chuẩn kiếp nạn, $70\%$ Purity, $N=2$).
5. **ARC-05 (Song Tu Trung Tính 50/50):** $15,000$ Hỏa, $15,000$ Lực ($\text{comp} = 0.85$, $50\%$ Purity, $N=2$).
6. **ARC-06 (Song Tu Tương Khắc 50/50 L0):** $15,000$ Hỏa, $15,000$ Thủy ($\text{comp} = 0.55$, Cực khắc, L0 Ordinary).
7. **ARC-07 (Song Tu Tương Khắc 50/50 L2):** $15,000$ Hỏa, $15,000$ Thủy ($\text{comp} = 0.55$, Đạt Đại Tông Sư L2).
8. **ARC-08 (Tứ Tu Kiếp Nạn 70/10/10/10):** $21,000$ Hỏa, $3,000$ Phong, $3,000$ Thổ, $3,000$ Thủy ($70\%$ Purity, $N=4$).
9. **ARC-09 (Tứ Tu Chia Đều 25%):** $7,500$ Hỏa, $7,500$ Thủy, $7,500$ Kim, $7,500$ Thổ ($25\%$ Purity, $N=4$).
10. **ARC-10 (Tạp Tu 10 Phái - Kiếp Nạn Lâu Năm):** $12,000$ Hỏa ($40\%$), 9 phái phụ mỗi phái $2,000$ ($N=10$, Total $30,000$).
11. **ARC-11 (Tạp Tu 10 Phái Chia Đều):** 10 phái mỗi phái $3,000$ ($10\%$ Purity, $N=10$, Total $30,000$).
12. **ARC-12 (Tạp Tu 10 Phái Rank 8):** $200,000$ Hỏa ($40\%$), 9 phái phụ mỗi phái $33,333$ ($N=10$, Total $500,000$).

### 4.2. Các Phương Án Đề Xuất Được Đưa Vào Tính Toán
- **Current Baseline:** Công thức hiện tại (Mẫu số $+50,000$, sàn $0.60$).
- **Phương Án 1 (Floor Only):** Giữ nguyên công thức hiện tại, chỉ thay sàn $0.60 \to 0.10$.
- **Phương Án 2 (Chuẩn Hóa Mẫu Số Thực Tế):** Bỏ đệm $+50,000$, thay bằng hằng số bảo vệ tối thiểu $+1,000$:
  $$\text{Ratio} = \frac{\text{EffConflict}}{amt + 1000}, \quad \text{Mult} = \max\left(0.10, \frac{1}{1.0 + 2.0 \times \text{Ratio}}\right)$$
- **Phương Án 3 (Mô Hình Đề Xuất Map #176 - Phạt Đa Tu Lũy Tiến):**
  Kết hợp tỷ lệ xung đột thực tế với hệ số phân rã đa tu ($\text{Penalty}_{\text{multi}}$):
  $$\text{ConflictFactor} = \frac{1}{1.0 + 2.0 \times \frac{\text{EffConflict}}{amt + 1000}}$$
  $$\text{PathPenalty} = \begin{cases} 1.0 & \text{khi } N = 1 \\ (\text{Purity})^{0.30} \times \frac{1}{1.0 + 0.20 \times (N - 1)^{1.20}} & \text{khi } N > 1 \end{cases}$$
  $$\text{OutputMult} = \max\left(0.08, \text{ConflictFactor} \times \text{PathPenalty}\right)$$
  *(Sàn bảo hiểm chọn mức 0.08, nằm trong dải đề xuất 0.05–0.10)*.

---

### 4.3. Bảng Kết Quả Tính Toán Số Học Chi Tiết

| Profile Mẫu | Purity | $N$ | Current Mult | PA 1 (Sàn 0.10) | PA 2 (Bỏ 50k) | PA 3 (Đề Xuất Mới) | Suy Giảm Sát Thương (PA 3 vs Current) |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **ARC-01 (Đơn tu R6)** | $100\%$ | 1 | **1.0000** | 1.0000 | 1.0000 | **1.0000** | **0.0%** |
| **ARC-02 (Đơn tu R7)** | $100\%$ | 1 | **1.0000** | 1.0000 | 1.0000 | **1.0000** | **0.0%** |
| **ARC-03 (Song tu Mộc-Hỏa 50/50)** | $50\%$ | 2 | **0.9475** | 0.9475 | 0.7805 | **0.5283** | **-44.2%** |
| **ARC-04 (Song tu Mộc-Hỏa 70/30)** | $70\%$ | 2 | **0.9705** | 0.9705 | 0.8907 | **0.6669** | **-31.3%** |
| **ARC-05 (Song tu Hỏa-Lực 50/50)** | $50\%$ | 2 | **0.9475** | 0.9475 | 0.7805 | **0.5283** | **-44.2%** |
| **ARC-06 (Song tu Hỏa-Thủy 50/50 L0)** | $50\%$ | 2 | **0.8575** | 0.8575 | 0.5424 | **0.3671** | **-57.2%** |
| **ARC-07 (Song tu Hỏa-Thủy 50/50 L2)** | $50\%$ | 2 | **0.8892** | 0.8892 | 0.6124 | **0.4145** | **-53.4%** |
| **ARC-08 (Tứ tu kiếp nạn 70/10/10/10)** | $70\%$ | 4 | **0.9517** | 0.9517 | 0.8302 | **0.4269** | **-55.1%** |
| **ARC-09 (Tứ tu chia đều 25%)** | $25\%$ | 4 | **0.8647** | 0.8647 | 0.4304 | **0.1625** | **-81.2%** |
| **ARC-10 (Tạp tu 10 phái 40% chính)** | $40\%$ | 10 | **0.9215** | 0.9215 | 0.6633 | **0.1328** | **-85.6%** |
| **ARC-11 (Tạp tu 10 phái chia đều 10%)** | $10\%$ | 10 | **0.8700** | 0.8700 | 0.2878 | **0.0800** *(Chạm sàn)* | **-90.8%** |
| **ARC-12 (Tạp tu 10 phái Rank 8 500k)** | $40\%$ | 10 | **0.7396** | 0.7396 | 0.6463 | **0.1294** | **-82.5%** |

---

### 4.4. Đồ Thị Phân Tích Mức Suy Giảm Sát Thương

```
[Mức Suy Giảm Sát Thương Thực Tế Khi Áp Dụng Đề Xuất Mới]

Đơn tu thuần (Purity 100%, N=1)       : [0.0%] (Không suy giảm - Bảo toàn 100%)
Song tu tương sinh (Purity 70%, N=2)  : [========= 31.3%]
Song tu tương sinh (Purity 50%, N=2)  : [============= 44.2%]
Tứ tu kiếp nạn (Purity 70%, N=4)      : [================ 55.1%]
Song tu tương khắc (Purity 50%, N=2)  : [================= 57.2%]
Tứ tu chia đều (Purity 25%, N=4)      : [========================= 81.2%]
Tạp tu 10 phái (Purity 40%, N=10)     : [========================== 85.6%]
Tạp tu chia đều (Purity 10%, N=10)    : [=========================== 90.8%] (Chạm sàn 0.08)
```

### 4.5. Nhận Xét & Đánh Giá Số Học
1. **Phương án 1 (Chỉ hạ sàn $0.60 \to 0.10$):**
   - Cột `PA 1` cho giá trị giống hệt `Current Mult` trên cả 12 profile.
   - **Kết luận:** Nếu không thay đổi mẫu số hoặc không thêm hệ số phạt đa tu, việc hạ sàn xuống $0.10$ hay $0.05$ là **hoàn toàn vô nghĩa** trên phương diện toán học!
2. **Đơn tu vs Tạp tu theo Đề Xuất Mới:**
   - Sự phân hóa cực kỳ sâu sắc: Đơn tu giữ trọn $1.0000$, trong khi Tạp tu 10 phái rớt thẳng từ $0.8700$ xuống $0.0800$ (giảm **$90.8\%$** sát thương).
   - Một đòn đánh Tiên Cổ cơ bản gây $100,000$ damage giờ chỉ còn gây $8,000$ damage nếu chủ nhân là một Cổ Tiên tạp tu 10 phái. Điều này hoàn toàn khớp với nguyên tác Cổ Chân Nhân.
3. **Bài toán Song Tu Tương Sinh & Cảnh Giới Tông Sư:**
   - Song tu tương sinh (ARC-04: Mộc - Hỏa $70/30$) vẫn bị giảm $31.3\%$ nếu chỉ dựa vào công thức cơ học thuần túy.
   - Do đó, **cần thiết phải có cơ chế đặc thù tại Ticket #180**: Khi người chơi đạt cấp độ Tông Sư trở lên ở các phái tương sinh, hoặc sử dụng sát chiêu phức hợp tương sinh, hệ số phạt đa tu $\text{PathPenalty}$ phải được triệt tiêu về $1.0$.

---

## 5. Đánh Giá Tác Động Lên PVE (Quái Vật & Boss)

### 5.1. Cơ Chế Quái Vật Trong Combat Hiện Tại
Khi chiến đấu với quái vật (`backend/internal/handlers/combat_auto.go` dòng 465):
```go
simEnemy.DaoProfile = gamefx.BuildDaoCombatProfile(
    enemy.DaoMarks,
    enemy.PathMasteries,
    false, // hasNonConflictingDao = false
    simEnemy.DaoBonus,
)
```
- Khi quái vật ra chiêu, sát thương của quái vật cũng đi qua hàm `actorDaoMultiplier(simEnemy, a.Path)`.
- Điều này có nghĩa: **Quái vật và Boss hiện tại ĐANG DÙNG CHUNG CÔNG THỨC VỚI NGƯỜI CHƠI!**

### 5.2. Tác Động Cụ Thể Lên Từng Loại Quái
1. **Quái Thường & Tinh Anh (Normal & Elite):**
   - Sinh ngẫu nhiên $1$ lưu phái duy nhất (`numPaths = 1`).
   - Đạt $100\%$ Purity, $N = 1$.
   - **Tác động:** Không chịu bất kỳ xung đột nào, Output Multiplier luôn là $1.0000$.
2. **Quái Boss (Hoang Thú / Thượng Cổ Hoang Thú):**
   - Sinh ngẫu nhiên $2$ lưu phái (`numPaths = 2`) với tỉ lệ $50\% / 50\%$.
   - **Rủi ro lớn phát hiện được:**
     * Vì 2 lưu phái được bốc ngẫu nhiên qua `rand.Perm(len(availablePaths))`, Boss có thể bốc phải cặp **tương khắc sâu** (ví dụ Hỏa + Thủy, Quang + Ám).
     * Với công thức cũ: Boss chỉ bị phạt nhẹ $14.25\%$ sát thương.
     * Với công thức mới: Boss sẽ bị suy giảm tới **$57.2\%$** sát thương!
     * Hậu quả: Boss ngẫu nhiên bị yếu đi một nửa mà không có lý do thiết kế rõ ràng, làm mất cân bằng độ khó của phó bản và khiêu chiến thế giới.

### 5.3. Khuyến Nghị Xử Lý Cho PVE
Có 2 phương án khả thi để bảo vệ tính nhất quán của PVE:
- **Phương án A (Khuyến nghị - Gán cờ Non-Conflicting):**
  Trong `models.MonsterTemplate` hoặc khi khởi tạo Boss Hoang Thú, đặt `HasNonConflictingDao = true` (Đại diện cho huyết mạch Hoang Thú bẩm sinh dung nạp vạn đạo tự nhiên, tương tự như Dị Thú thượng cổ).
- **Phương án B (Thuật toán ghép cặp tương sinh):**
  Trong hàm `ScaleAndRandomizeMonsterDaoMarks`, nếu `isBoss && numPaths == 2`, thay vì random tự do, thuật toán phải chọn một cặp lưu phái có $\text{Compatibility} \ge 0.95$ (ví dụ: Kim + Thủy, Mộc + Hỏa, Phong + Lôi).

---

## 6. Kết Luận & Khuyến Nghị Cho Các Ticket Tiếp Theo

1. **Cho Ticket #178 (Thiết kế mô hình toán học chi tiết):**
   - Phải loại bỏ hoặc giảm mạnh hằng số đệm $+50,000$ trong mẫu số tính `ConflictRatio` (khuyến nghị thay bằng $+1,000$ hoặc chuẩn hóa theo tỷ lệ $amt / \text{totalMarks}$).
   - Đưa chính thức $2$ biến số vào công thức: `UniquePathCount` ($N$) và `Purity` ($P$).
   - Hạ sàn xung đột chính thức về khoảng **$0.05 – 0.08$** để tạo mức đáy bảo hiểm cho tạp tu cực đoan.
2. **Cho Ticket #179 (Mana Tax & Phản Phệ Nội Thương):**
   - Chỉ nên kích hoạt phản phệ nội thương khi $\text{ConflictMult} < 0.50$ (tức chỉ áp dụng cho người chơi đa tu từ 3 phái trở lên hoặc song tu tương khắc).
3. **Cho Ticket #180 (Vô Thượng Đại Tông Sư & Quy Tắc Tương Sinh):**
   - Bổ sung cơ chế: Cảnh giới Đại Tông Sư / Vô Thượng Đại Tông Sư được quyền miễn trừ hệ số phạt $\text{PathPenalty}$ đối với các cặp tương sinh.
4. **Cho Ticket #182 (Thanh Đạo Tẩy Thể):**
   - Do cơ chế độ kiếp tự động văng $30\%$ Đạo Ngân rác sang các hệ khác, hệ thống bắt buộc phải có tính năng Thanh Đạo Tẩy Thể sớm trước khi áp dụng phạt đa tu, nhằm tránh việc người chơi cũ bị "phế build" ngoài ý muốn.
