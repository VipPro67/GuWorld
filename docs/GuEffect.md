Dưới đây là bản mô tả chi tiết, chuẩn hóa thiết kế kỹ thuật **(Detailed Technical Specification)** cho ý tưởng phân rã hiệu ứng Cổ trùng thành 3 khối: **`active`**, **`passive`**, và **`other`**.

Bản thiết kế này được xây dựng riêng cho kiến trúc dự án Web Game của đạo hữu (Backend **Go/Redis/Postgres** + Frontend **Next.js/TypeScript**), đảm bảo vừa tinh gọn cho cơ sở dữ liệu, vừa tối ưu hóa hiệu năng tính toán thời gian thực của Server.

---

# BẢN THIẾT KẾ CHI TIẾT: HỆ THỐNG HIỆU ỨNG CỔ TRÙNG TRILOGY

*(Active - Passive - Other Architecture)*

## I. TỔNG QUAN KIẾN TRÚC PHÂN LUỒNG

Ý tưởng này chia nhỏ vòng đời và hành vi của Cổ trùng ra làm 3 trạng thái xử lý độc lập trên Server. Sự phân tách này giúp cô lập logic chiến đấu số học ra khỏi các logic phá vỡ quy tắc khái niệm.

```
       [ Cổ Trùng Inventory ]
                 │
      ┌──────────┼──────────┐
      ▼          ▼          ▼
  [Active]   [Passive]   [Other]
      │          │          │
      ▼          ▼          ▼
  Combat     Snapshot     Event
  Tick Loop  Character    Router

```

---

## II. ĐỊNH NGHĨA CHI TIẾT CẤU TRÚC JSON SCHEMA (`effects`)

Trường `effects` trong bảng `Gu` (Postgres) sẽ được cấu hình dưới dạng kiểu dữ liệu `JSONB`. Cấu trúc rỗng của một con Cổ trùng sẽ như sau:

```json
{
  "active": null,
  "passive": null,
  "other": []
}

```

### 1. Khối `active` (Chủ động - Chỉ gọi trong Combat)

Chỉ kích hoạt khi Cổ sư hoặc AI Auto ra lệnh xả chiêu tại một Lượt đấu (Tick).

```json
"active": {
  "damage_flat": 25.0,          // Sát thương phẳng
  "damage_path": "Moon Path",   // Lưu phái sát thương để tính Đạo Ngân tăng phúc
  "heal_flat": 0.0,             // Trị liệu phẳng cho HP
  "shield_flat": 0.0,           // Tạo lớp Chân Nguyên Giáp hộ thể
  "apply_status": "BLEED",      // Áp hiệu ứng bất lợi (nếu có)
  "status_tick_damage": 3.0,    // Dame độc/chảy máu mỗi tick
  "status_duration_ticks": 3    // Thời gian duy trì hiệu ứng bất lợi
}

```

### 2. Khối `passive` (Bị động - Cộng dồn thuộc tính)

Tự động kích hoạt khi Cổ trùng được trang bị vào Không khiếu. Engine sẽ gom toàn bộ các chỉ số này để tạo ra một bản Snapshot trạng thái nhân vật trước trận đấu.

```json
"passive": {
  "stat_base_attack": 0.0,       // Cộng thẳng vào sức mạnh vật lý (như Hắc Thỉ Cổ)
  "stat_base_defense": 20.0,     // Cộng thẳng vào giáp phòng ngự (như Ngọc Bì Cổ)
  "mana_regen_mult": 1.05,       // Tốc độ hồi phục Chân Nguyên (như Thiên Nguyên Bảo Liên)
  "path_conflict_reduction": 2   // Giảm % xung đột Đạo Ngân khi dùng Cổ dị hệ
}

```

### 3. Khối `other` (Mảnh vỡ quy tắc - Hướng sự kiện)

Đây là một **Mảng các Đối tượng (Array of Objects)**. Khối này không tham gia vào vòng lặp tính toán Dame thông thường mà đứng xếp hàng chờ các Hook Sự kiện (`event_trigger`) của hệ thống gọi tên.

```json
"other": [
  {
    "event_trigger": "ON_FATAL_DAMAGE", // Điều kiện kích hoạt: Nhận dame chết
    "action_code": "TIMELINE_ROLLBACK", // Mã hàm Backend sẽ gọi
    "params": {                         // Tham số tùy biến cho hàm đó
      "success_rate_percent": 10.0,
      "rollback_ticks": 120
    }
  }
]

```

---

## III. HIỆN THỰC HÓA BẰNG MÃ NGUỒN (GOLANG & TYPESCRIPT)

Để đạo hữu ném thẳng cho Đội ngũ Phát triển (Developer) hoặc nạp vào AI Agent lập trình, đây là cấu trúc khai báo dữ liệu chuẩn:

### 1. Backend Structs (Golang)

```go
package models

import "encoding/json"

// Cấu trúc Active Effect
type ActiveEffect struct {
	DamageFlat          float64 `json:"damage_flat,omitempty"`
	HealFlat            float64 `json:"heal_flat,omitempty"`
	ShieldFlat          float64 `json:"shield_flat,omitempty"`
	ApplyStatus         string  `json:"apply_status,omitempty"`
	StatusTickDamage    float64 `json:"status_tick_damage,omitempty"`
	StatusDurationTicks int     `json:"status_duration_ticks,omitempty"`
}

// Cấu trúc Passive Effect
type PassiveEffect struct {
	StatBaseAttack        float64 `json:"stat_base_attack,omitempty"`
	StatBaseDefense       float64 `json:"stat_base_defense,omitempty"`
    StatBaseHP            float64 `json:"stat_base_hp,omitempty"`
	ManaRegenMult         float64 `json:"mana_regen_mult,omitempty"`
	PathConflictReduction float64 `json:"path_conflict_reduction,omitempty"`
}

// Cấu trúc Other (Event-Driven Mechanic)
type SpecialMechanic struct {
	EventTrigger string          `json:"event_trigger"`
	ActionCode   string          `json:"action_code"`
	Params       json.RawMessage `json:"params"` // Lưu cấu trúc động không cố định
}

// Cấu trúc tổng thể lưu trong Database
type GuEffects struct {
	Active  *ActiveEffect     `json:"active"`
	Passive *PassiveEffect    `json:"passive"`
	Other   []SpecialMechanic `json:"other"`
}

```

### 2. Frontend Types (TypeScript / Next.js)

```typescript
export interface ActiveEffect {
  damage_flat?: number;
  damage_path?: string;
  heal_flat?: number;
  shield_flat?: number;
  apply_status?: string;
  status_tick_damage?: number;
  status_duration_ticks?: number;
}

export interface PassiveEffect {
  stat_base_attack?: number;
  stat_base_defense?: number;
  mana_regen_mult?: number;
  path_conflict_reduction?: number;
}

export interface SpecialMechanic {
  event_trigger: string;
  action_code: string;
  params: Record<string, any>;
}

export interface GuEffects {
  active: ActiveEffect | null;
  passive: PassiveEffect | null;
  other: SpecialMechanic[];
}

```

---

## IV. VÍ DỤ CẤU HÌNH DATA ĐIỂN HÌNH (CSV TO JSON)

Khi nạp dữ liệu từ file `Gu.csv` vào hệ thống, cột `effects` sẽ bốc bóc tách cực kỳ tường minh:

### 1. Bạch Thỉ Cổ (`GU_WHITE_BOAR`) - Rank 1 (Thuần Thụ Động)

*Chữ hiển thị UI:* Thụ động gia tăng sức mạnh một con Bạch thỉ vĩnh viễn.

```json
{
  "active": null,
  "passive": {
    "stat_base_attack": 25.0
  },
  "other": []
}

```

### 2. Nguyệt Quang Cổ (`GU_MOONLIGHT`) - Rank 1 (Hỗn Hợp Chủ Động + Thụ Động)

*Chữ hiển thị UI:* Kích hoạt bắn ra nguyệt nhận gây 25 dame, kèm hiệu ứng chảy máu. Thụ động tăng nhẹ dame hệ Moon khi mang theo.

```json
{
  "active": {
    "damage_flat": 25.0,
    "damage_path": "Moon Path",
    "apply_status": "BLEED",
    "status_tick_damage": 3.0,
    "status_duration_ticks": 3
  },
  "passive": {
    "path_conflict_reduction": 1.02
  },
  "other": []
}

```

### 3. Đâu Suất Hoa (`GU_TUSITA_FLOWER`) - Rank 3 (Thuần Chức Năng / Tiêu Hao ngoài trận)

*Chữ hiển thị UI:* Không khiếu mở rộng không gian chứa đồ cho phàm tài và nguyên thạch.

```json
{
  "active": null,
  "passive": null,
  "other": [
    {
      "event_trigger": "ON_INVENTORY_OPEN",
      "action_code": "EXPAND_STORAGE_SLOTS",
      "params": {
        "max_slots_bonus": 20,
        "allowed_item_tags": ["mortal_material", "primeval_stone"]
      }
    }
  ]
}

```

### 4. Tiên Cổ: Định Tiên Du (`GU_FIXED_IMMORTAL_TRAVEL`) - Rank 6 (Thuần Khái Niệm Tối Cao)

*Chữ hiển thị UI:* Tiên đạo quy tắc, dịch chuyển đến bất kỳ khu vực nào trên thế giới mà không chịu hạn chế.

```json
{
  "active": null,
  "passive": null,
  "other": [
    {
      "event_trigger": "ON_MAP_TELEPORT_CLICK",
      "action_code": "BYPASS_WORLD_BARRIERS",
      "params": {
        "allow_cross_region": true,
        "ignore_fog_of_war": true
      }
    }
  ]
}

```

---

## V. LUỒNG VẬN HÀNH KỸ THUẬT TRÊN SERVER (GO ENGINE)

Thiết kế này giúp phân luồng xử lý cực kỳ thông minh, tránh tối đa việc Server phải chạy vòng lặp lồng nhau (Nested Loops) gây lag game khi số lượng người chơi tăng cao.

### Luồng 1: Giai đoạn Khởi động Trận đấu (Load Snapshot)

Engine quét mảng Cổ trùng đang trang bị của Player $\rightarrow$ Gom tất cả những block `passive` lại $\rightarrow$ Tính toán ra chỉ số Giáp, Sức mạnh, Tốc độ hồi mana cuối cùng của Player đó trước khi bước vào tick đấu đầu tiên.

### Luồng 2: Vòng lặp Lượt đánh (Combat Tick Loop)

Khi đến tick đấu của Player, AI hoặc người chơi chọn dùng **Nguyệt Quang Cổ**:

1. Server chỉ check đúng block `active`.
2. Lấy `damage_flat` (25) + nhân với hệ số buff từ Snapshot (Luồng 1) + nhân hệ số Đạo Ngân $\rightarrow$ Ra con số sát thương cuối cùng trừ vào Máu/Giáp kẻ địch.
3. Nếu có `apply_status`, đẩy Object Status đó vào hàng đợi Debuff của nạn nhân.

### Luồng 3: Bộ định tuyến Sự kiện ngoài trận (Global Event Router)

Mỗi khi hệ thống nổ ra một sự kiện (ví dụ: Player tụt máu về 0), Server sẽ phát ra một tín hiệu toàn cục `ON_FATAL_DAMAGE`.

1. Hệ thống định tuyến sẽ check xem Player này có mang Cổ nào có dòng `other` chứa `event_trigger == "ON_FATAL_DAMAGE"` không.
2. Nếu phát hiện Player đang trang bị **Xuân Thu Thiền**, hàm `TIMELINE_ROLLBACK` lập tức được gọi để xử lý logic đặc biệt.
3. Nếu Player chỉ mang Cổ trùng phàm cấp bình thường (Khối `other` rỗng), bộ định tuyến sẽ lập tức bỏ qua, không tốn một chút CPU nào để kiểm tra thêm chỉ số.

### Kết luận từ Cổ Chân Nhân:

Kiến trúc phân rã làm ba này đạt đến mức **"Đại đạo chí giản"**. Nó giải quyết triệt để bài toán: Phàm cổ thì xử lý siêu tốc bằng toán học phẳng, Tiên cổ thì mở rộng không giới hạn bằng định tuyến hàm đặc biệt. Cơ cấu dữ liệu này đã sẵn sàng để đạo hữu xây dựng một thiên địa Cổ sư hoàn mỹ!