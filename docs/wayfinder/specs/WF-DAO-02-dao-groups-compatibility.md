# [SPECIFICATION] 10 Nhóm Lưu Phái, Ma Trận Tương Thích & Vùng Đệm An Toàn (Dao Groups & Compatibility)

**Ticket:** [WF-DAO-02: Thiết kế 10 Nhóm Lưu Phái (Dao Groups), Ma Trận Tương Thích & Vùng Đệm An Toàn](../tickets/WF-DAO-02-spec-dao-groups-and-compatibility-matrix.md)  
**Tác giả:** Antigravity  
**Ngày hoàn thành:** 2026-09-20  

---

## 1. Phân Loại 49 Lưu Phái Thành 10 Dao Groups

Dựa trên danh sách chuẩn từ `dreamRealmPaths` trong `backend/internal/db/db.go` và `pathToVietnamese` trong `admin_metrics.go`, 49 lưu phái được phân vào 10 nhóm quy tắc:

```go
type DaoGroup string

const (
    DaoGroupElemental          DaoGroup = "ELEMENTAL"
    DaoGroupSpaceTime          DaoGroup = "SPACE_TIME"
    DaoGroupRuleOrder          DaoGroup = "RULE_ORDER"
    DaoGroupKineticWeapon      DaoGroup = "KINETIC_WEAPON"
    DaoGroupBodyLife           DaoGroup = "BODY_LIFE"
    DaoGroupSoulMind           DaoGroup = "SOUL_MIND"
    DaoGroupInformationStealth DaoGroup = "INFORMATION_STEALTH"
    DaoGroupOrganicCorruption  DaoGroup = "ORGANIC_CORRUPTION"
    DaoGroupCraftAlchemy       DaoGroup = "CRAFT_ALCHEMY"
    DaoGroupHeavenHuman        DaoGroup = "HEAVEN_HUMAN"
)
```

### Chi Tiết Phân Nhóm 49 Lưu Phái:

| Dao Group | Canonical Path Keys | Tên Tiếng Việt |
|---|---|---|
| **`ELEMENTAL`** | `metal`, `wood`, `water`, `fire`, `earth`, `wind`, `lightning`, `ice`, `cloud`, `light`, `dark`, `star`, `sound` | Kim, Mộc, Thủy, Hỏa, Thổ, Phong, Lôi, Băng Tuyết, Vân, Quang, Ám, Tinh, Âm |
| **`SPACE_TIME`** | `space`, `time` | Vũ đạo (Không gian), Trụ đạo (Thời gian) |
| **`RULE_ORDER`** | `rule`, `restriction`, `formation` | Luật đạo, Cấm đạo, Trận đạo |
| **`KINETIC_WEAPON`** | `sword`, `blade`, `weapon`, `killing` | Kiếm đạo, Đao đạo, Binh đạo, Sát đạo |
| **`BODY_LIFE`** | `strength`, `bone`, `transformation`, `food` | Lực đạo, Cốt đạo, Biến Hóa đạo, Thực đạo |
| **`SOUL_MIND`** | `soul`, `wisdom`, `emotion`, `enslavement`, `dream` | Hồn đạo, Trí đạo, Tình đạo, Nô đạo, Mộng đạo |
| **`INFORMATION_STEALTH`** | `information`, `luck`, `theft`, `phantom`, `painting` | Tín đạo, Vận đạo, Thâu đạo, Huyễn đạo, Họa đạo |
| **`ORGANIC_CORRUPTION`** | `blood`, `poison` | Huyết đạo, Độc đạo |
| **`CRAFT_ALCHEMY`** | `refinement`, `pill` | Luyện đạo, Đan đạo |
| **`HEAVEN_HUMAN`** | `heaven`, `human`, `qi` | Thiên đạo, Nhân đạo, Khí đạo |

---

## 2. Ma Trận Tương Thích Nhóm (Group Compatibility Matrix $10 \times 10$)

Mức độ tương thích giữa hai nhóm $\text{GroupComp}(G_A, G_B) \in [0.0, 1.0]$. Giá trị càng gần $1.0$ càng ít xung đột.

| Group A \ Group B | ELEM | SP_TI | RULE | WEAP | BODY | SOUL | INFO | CORR | CRAF | HEAV |
|---|---|---|---|---|---|---|---|---|---|---|
| **ELEM** | **0.88** | 0.90 | 0.85 | 0.85 | 0.85 | 0.85 | 0.85 | 0.80 | 0.92 | 0.88 |
| **SP_TI** | 0.90 | **0.95** | 0.95 | 0.88 | 0.85 | 0.90 | 0.92 | 0.82 | 0.90 | 0.92 |
| **RULE** | 0.85 | 0.95 | **0.95** | 0.90 | 0.85 | 0.90 | 0.94 | 0.80 | 0.94 | 0.95 |
| **WEAP** | 0.85 | 0.88 | 0.90 | **0.94** | 0.90 | 0.88 | 0.85 | 0.88 | 0.92 | 0.88 |
| **BODY** | 0.85 | 0.85 | 0.85 | 0.90 | **0.95** | 0.88 | 0.85 | 0.92 | 0.88 | 0.90 |
| **SOUL** | 0.85 | 0.90 | 0.90 | 0.88 | 0.88 | **0.95** | 0.95 | 0.85 | 0.88 | 0.90 |
| **INFO** | 0.85 | 0.92 | 0.94 | 0.85 | 0.85 | 0.95 | **0.95** | 0.82 | 0.90 | 0.92 |
| **CORR** | 0.80 | 0.82 | 0.80 | 0.88 | 0.92 | 0.85 | 0.82 | **0.94** | 0.85 | 0.85 |
| **CRAF** | 0.92 | 0.90 | 0.94 | 0.92 | 0.88 | 0.88 | 0.90 | 0.85 | **0.96** | 0.92 |
| **HEAV** | 0.88 | 0.92 | 0.95 | 0.88 | 0.90 | 0.90 | 0.92 | 0.85 | 0.92 | **0.95** |

*(Ghi chú: Ma trận đối xứng, $G_A \leftrightarrow G_B = G_B \leftrightarrow G_A$)*

---

## 3. Danh Sách Ghi Đè Cặp Ngoại Lệ (Pair Overrides)

Nếu một cặp $(P_A, P_B)$ có mặt trong danh sách này, giá trị này sẽ thay thế hoàn toàn $\text{GroupComp}$:

### 3.1. Cặp Tương Khắc Sâu Sắc (High Conflict: $0.55 - 0.65$)
- `fire` $\leftrightarrow$ `water` = **0.55** (Thủy Hỏa bất dung)
- `fire` $\leftrightarrow$ `ice` = **0.60** (Hỏa Băng khắc chế)
- `light` $\leftrightarrow$ `dark` = **0.55** (Quang Ám đối nghịch)
- `blood` $\leftrightarrow$ `light` = **0.65** (Huyết đạo bị Quang đạo khắc chế)
- `poison` $\leftrightarrow$ `food` = **0.60** (Độc đạo xung khắc Thực đạo)

### 3.2. Cặp Tương Sinh & Hòa Hợp Đặc Thù (High Harmony: $0.94 - 0.98$)
- `space` $\leftrightarrow$ `time` = **0.97** (Vũ Trụ thời không tương dung)
- `wisdom` $\leftrightarrow$ `information` = **0.98** (Trí Tín tương hỗ)
- `wisdom` $\leftrightarrow$ `soul` = **0.95** (Trí Hồn đồng nguyên)
- `strength` $\leftrightarrow$ `bone` = **0.96** (Lực Cốt liền mạch)
- `strength` $\leftrightarrow$ `blood` = **0.94** (Huyết Khí tráng thịnh)
- `sword` $\leftrightarrow$ `blade` = **0.95** (Đao Kiếm song tuyệt)
- `sword` $\leftrightarrow$ `killing` = **0.96** (Kiếm Sát hợp nhất)
- `refinement` $\leftrightarrow$ `pill` = **0.97** (Luyện Đan tương thông)
- **Ngũ Hành Tương Sinh:**
  - `wood` $\leftrightarrow$ `fire` = **0.95** (Mộc sinh Hỏa)
  - `fire` $\leftrightarrow$ `earth` = **0.95** (Hỏa sinh Thổ)
  - `earth` $\leftrightarrow$ `metal` = **0.95** (Thổ sinh Kim)
  - `metal` $\leftrightarrow$ `water` = **0.95** (Kim sinh Thủy)
  - `water` $\leftrightarrow$ `wood` = **0.95** (Thủy sinh Mộc)

---

## 4. Cơ Chế Vùng Đệm An Toàn (Grace Threshold)

### 4.1. Công thức lọc bỏ Đạo Ngân dị chủng nhỏ lẻ:
Khi người chơi thi triển sát chiêu thuộc `mainPath`, với mỗi `otherPath`:
$$\text{Threshold} = \max\left(1000, \text{TotalDaoMarks} \times 0.02\right)$$
$$\text{If } \text{DaoMarks}(\text{otherPath}) \le \text{Threshold} \implies \text{Ignore in RawConflict}$$

### 4.2. Ý nghĩa Gameplay:
- Ngăn chặn triệt để tình trạng người chơi Cổ Tiên Rank 6-8 (có $50.000 - 500.000$ đạo ngân) bị "bẩn build" hoặc giảm hiệu quả chiến đấu khi lỡ farm/nhặt $100 - 500$ đạo ngân từ phụ bản hay di vật quái vật.
- Chỉ khi người chơi chủ ý tu luyện hoặc tích lũy một lượng đạo ngân ngoại hệ đáng kể ($> 1.000$ và $> 2\%$) thì cơ chế xung đột mới bắt đầu có hiệu lực và kích hoạt yêu cầu điều hòa cảnh giới.
