# Realtime Beast Enslavement (Nô Đạo Channeling) Specification

## 1. Tổng Quan Quy Trình

Cơ chế thu phục dã thú trong chiến đấu Realtime (20Hz zone sim) thay thế hoàn toàn hệ thống turnbase cũ. Cổ sư phải trực tiếp tiếp cận quái vật trên bản đồ thế giới và tiến hành dẫn kênh (Channeling) kết ấn Nô Đạo. Không cần giảm HP hoặc khống chế quái trước; các điều kiện này chỉ ảnh hưởng xác suất bắt.

```mermaid
sequenceDiagram
    autonumber
    actor Client as Cổ Sư (Client HUD)
    participant Gateway as Realtime Gateway
    participant Zone as Zone Sim (20Hz)
    participant Worker as Capture Worker
    participant DB as PostgreSQL (non_player_actors)

    Client->>Gateway: Envelope { type: "world.capture", target_id, gu_id }
    Gateway->>Zone: Enqueue Capture Intent
    Zone->>Zone: Validate (capture Gu equipped, Range <= 6m, Mana >= 20)
    alt Không đủ điều kiện
        Zone-->>Gateway: Event { type: "CAST_REJECTED", reason: "capture_gu_required / out_of_range / insufficient_mana" }
        Gateway-->>Client: Hiển thị từ chối / Không thể bắt
    else Đủ điều kiện
        Zone->>Zone: Set Caster Pending (3.0s = 60 ticks, Root movement)
        Zone-->>Client: Event { type: "CAPTURE_STARTED", ticks: 60 }
        loop 60 Ticks (Channeling)
            alt Cổ sư bị dính sát thương hoặc dính CC
                Zone->>Zone: Clear Pending & Root
                Zone-->>Client: Event { type: "CAPTURE_INTERRUPTED" }
            end
        end
        Note over Zone: Hết 60 ticks an toàn -> Giải quyết (Resolve)
        Zone->>Zone: Tính toán xác suất P_capture (Công thức Issue #276)
        alt Thất Bại
            Zone-->>Client: Event { type: "CAPTURE_FAILED" }
            opt Thần Hồn yếu hơn Ý Chí Quái
                Zone->>Zone: Apply Soul Backfire (Mất Soul HP & Stun 2s)
                Zone-->>Client: Event { type: "SOUL_BACKFIRE" }
            end
        else Thành Công
            Zone->>Zone: Xóa Entity Quái khỏi sim.World
            Zone-->>Client: Event { type: "CAPTURE_SUCCESS", target_id }
            Zone->>Worker: Enqueue CaptureEvent (asynchronous, non-blocking)
            Worker->>DB: INSERT INTO non_player_actors (COMPANION_BEAST)
            Worker-->>Client: SSE / World Notification: "Đã thu phục thành công!"
        end
    end
```

---

## 2. Giao Thức Wire Protocol (`realtime/protocol`)

### 2.1. Client Intent Message
Thêm loại thông điệp mới vào `backend/internal/realtime/protocol/protocol.go`:
```go
const (
    TypeCapture = "world.capture"
)

type CaptureIntent struct {
    Seq      uint64 `json:"seq"`
    TargetID string `json:"target_id"`
    GuID     string `json:"gu_id"`
}
```

### 2.2. Server Sim Events
Được định nghĩa trong `backend/internal/realtime/combat/events.go`:
```go
const (
    EventCaptureStarted     EventType = "CAPTURE_STARTED"
    EventCaptureInterrupted EventType = "CAPTURE_INTERRUPTED"
    EventCaptureSuccess     EventType = "CAPTURE_SUCCESS"
    EventCaptureFailed      EventType = "CAPTURE_FAILED"
    EventSoulBackfire       EventType = "SOUL_BACKFIRE"
)
```

---

## 3. Điều Kiện Bắt Đầu Dẫn Kênh (Pre-conditions)

Khi nhận `CaptureIntent`, `Zone Sim` kiểm tra 4 nhóm điều kiện trong tick:
1. **Trạng thái Cổ sư**: `HP > 0`, không bị Choáng (`isStunned == false`), không bị Câm lặng (`isSilenced == false`), không đang trong trạng thái niệm chiêu khác (`Combat.Pending == nil`).
2. **Trạng thái Mục tiêu**: Tồn tại trong zone, `HP > 0`, `Kind == EntityMonster`, đang hoạt động.
3. **Cự Ly**: $\text{Distance}(\text{Caster}, \text{Target}) \le 6.0\text{m}$.
4. **Cổ Trùng & Chân Nguyên**: `gu_id` phải thuộc Cổ Nô Đạo đang trang bị, trạng thái `ACTIVE`, độ no tối thiểu 5; Chân nguyên đủ 20 Mana. Server tra cứu capability từ dữ liệu Cổ, không tin bonus do client tự khai báo.

Quái có thể bị thu phục ở mọi mức HP khi còn sống. Hạ HP hoặc khống chế cứng (`STUN`, `FREEZE`, `PETRIFY`, `BIND`, `ROOT`) làm tăng xác suất bắt.

Nếu thỏa mãn:
- Gán `caster.Combat.Pending = &PendingCast{ Mode: ModeCapture, TargetID: target.ID, ResolveTick: w.tick + 60, MovementPolicy: "ROOT" }`.
- Phát sự kiện `CAPTURE_STARTED` (thời lượng 60 ticks = 3.0 giây).
- Client hiển thị thông báo bắt đầu, thành công, thất bại hoặc bị gián đoạn từ các event tương ứng.

---

## 4. Cơ Chế Ngắt Niệm (Interrupt Handling)

Trong mỗi chu kỳ 20Hz:
- Nếu Cổ sư di chuyển (nhận `Input.Move` mới) &rarr; Ngắt niệm, lý do `"moved"`.
- Nếu Cổ sư nhận bất kỳ sát thương nào (`w.applyDamageLocked`) &rarr; Ngắt niệm, lý do `"damaged"`.
- Nếu Cổ sư trúng hiệu ứng khống chế (`STUN`, `SILENCE`, `FREEZE`) &rarr; Ngắt niệm, lý do `"crowd_control"`.
- Khi ngắt niệm:
  - Hủy `caster.Combat.Pending = nil`.
  - Giải trừ trạng thái `ROOT`.
  - Phát sự kiện `CAPTURE_INTERRUPTED`.

---

## 5. Giải Quyết Hoàn Tất (Completion & Async Persistence)

Tại `w.tick == ResolveTick`:
1. **Tính toán xác suất thành công**:
   - Sử dụng hàm `CalculateCaptureChance` (kết quả vé #276).
2. **Nếu Thất Bại**:
   - Phát sự kiện `CAPTURE_FAILED`.
   - Nếu có phản phệ: Trừ `SoulHP` và gắn hiệu ứng `STUN` 2s lên Cổ sư.
3. **Nếu Thành Công**:
   - Xóa thực thể quái vật khỏi `sim.World` (`w.removeEntityLocked(target)`).
   - Phát sự kiện `CAPTURE_SUCCESS`.
   - **Xử lý lưu trữ bất đồng bộ (Non-blocking DB Persistence)**:
     - Đưa sự kiện vào `m.captureQueue <- CaptureEvent{ ... }` trong `manager.go` (tương tự như `m.deathQueue`).
     - `captureWorker` nhận sự kiện trong background goroutine và mở Transaction lưu trữ:
       ```sql
       INSERT INTO non_player_actors (
           id, actor_kind, owner_player_id, template_source, template_id,
           species, name, rank, hp, max_hp, base_attack, base_defense, speed,
           count, hunger, max_hunger, loyalty, lifecycle_state, execution_state,
           behavior_profile, formation_slot
       ) VALUES (
           gen_random_uuid(), 'COMPANION_BEAST', $1, 'MONSTER_TEMPLATE', $2,
           $3, $4, $5, $6, $7, $8, $9, $10,
           1, 0, 100, $11, 'ACTIVE', 'STORED',
           'PASSIVE', 0
       );
       ```
     - Zone tick không bao giờ bị nghẽn (zero tick stall).
