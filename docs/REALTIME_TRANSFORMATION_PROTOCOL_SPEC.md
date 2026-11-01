# Đặc Tả Giao Thức Đồng Bộ Realtime (Wire Protocol) & Hiển Thị Client Cho Biến Thân

## 1. Tổng Quan Kiến Trúc
Tài liệu này đặc tả chi tiết giao thức truyền tải sự kiện nhị phân / JSON giữa Server Zone Sim 20Hz (`backend/internal/realtime/`) và Client Game Canvas/React UI (`frontend/src/game/`).

```
+-----------------------------+           WebSocket           +-----------------------------+
|    Server Zone Sim 20Hz     |  ==========================>  |     Client (WorldScene/UI)  |
|  - TransformationState      |      world.events             |  - Scale Sprite & Hitbox    |
|  - Form HP Layer & Upkeep   |      world.snapshot           |  - Form HP Bar (Purple)     |
|  - Skill Palette Swap       |                               |  - Aura Particle VFX        |
|                             |  <==========================  |  - Action Bar Dynamic Swap  |
|                             |      world.cast               |                             |
+-----------------------------+                               +-----------------------------+
```

---

## 2. Giao Thức Wire Protocol (Server $\rightarrow$ Client)

### 2.1. Danh Mục Sự Kiện Combat Event Vocabulary (`combat.EventType`)
Được phát qua kênh `world.events`:

| Event Type | Mô Tả | Tham Số Đi Kèm |
| :--- | :--- | :--- |
| `TRANSFORMATION_STARTED` | Phát khi Cổ sư bắt đầu hoàn tất biến thân hoặc bắt đầu niệm chú | `source_id`, `form_id`, `form_name`, `form_kind`, `model_scale`, `collision_radius`, `max_form_hp`, `current_form_hp`, `windup_ticks`, `form_skills` |
| `TRANSFORMATION_TICK` | Đồng bộ hao hụt tài nguyên & Form HP theo nhịp | `source_id`, `current_form_hp`, `current_mana`, `upkeep_mana_per_tick` |
| `TRANSFORMATION_ENDED` | Giải trừ hình thái trở lại dạng người phàm | `source_id`, `form_id`, `reason` (`manual_cancel`, `mana_depleted`, `duration_expired`), `restored_skills`, `debuff_applied` |
| `TRANSFORMATION_BROKEN` | Lớp máu Form HP bị đánh sập về 0 | `source_id`, `form_id`, `spillover_damage`, `debuff_applied: "EXHAUSTION"` |

### 2.2. Cấu Trúc Chi Tiết Gói Tin JSON (Payload Schemas)

#### A. Gói tin `TRANSFORMATION_STARTED`
```json
{
  "type": "TRANSFORMATION_STARTED",
  "tick": 1420,
  "source_id": "player_67e9f",
  "form_id": "FORM_FLAME_BEAR",
  "form_name": "Thái Cổ Viêm Hùng",
  "form_kind": "FULL",
  "model_scale": 1.6,
  "collision_radius": 1.4,
  "max_form_hp": 180.0,
  "current_form_hp": 180.0,
  "windup_ticks": 30,
  "form_skills": [
    { "slot": 1, "id": "BEAR_CLAW_SWIPE", "name": "Hùng Trảo Cuồng Liệp", "mode": "DIRECTION", "mana_cost": 5, "cooldown_ticks": 20, "range": 3.5 },
    { "slot": 2, "id": "BEAR_FERAL_CHARGE", "name": "Bạo Nộ Cuồng Húc", "mode": "DIRECTION", "mana_cost": 15, "cooldown_ticks": 80, "range": 8.0 },
    { "slot": 3, "id": "BEAR_EARTH_SHATTER", "name": "Chấn Địa Nham Liệt", "mode": "GROUND", "mana_cost": 25, "cooldown_ticks": 120, "radius_units": 4.5 },
    { "slot": 4, "id": "BEAR_ROAR_INTIMIDATE", "name": "Thao Thiết Thú Hống", "mode": "SELF", "mana_cost": 20, "cooldown_ticks": 160, "radius_units": 6.0 },
    { "slot": 5, "id": "BEAR_FLAME_RAMPAGE", "name": "Vạn Thú Viêm Bạo", "mode": "DIRECTION", "mana_cost": 50, "cooldown_ticks": 300, "range": 10.0 },
    { "slot": 6, "id": "REVERT_METAMORPHOSIS", "name": "Giải Trừ Biến Thân", "mode": "SELF", "mana_cost": 0, "cooldown_ticks": 10 }
  ]
}
```

#### B. Gói tin `TRANSFORMATION_ENDED` / `TRANSFORMATION_BROKEN`
```json
{
  "type": "TRANSFORMATION_ENDED",
  "tick": 1680,
  "source_id": "player_67e9f",
  "form_id": "FORM_FLAME_BEAR",
  "reason": "form_broken",
  "spillover_damage": 35.5,
  "debuff_applied": "EXHAUSTION",
  "debuff_duration_ticks": 100,
  "restored_skills": [
    { "slot": 1, "id": "GU_FIRE_BALL", "cooldown_ticks_remaining": 0 },
    { "slot": 2, "id": "GU_WATER_SHIELD", "cooldown_ticks_remaining": 0 },
    { "slot": 3, "id": "GU_WIND_STEP", "cooldown_ticks_remaining": 15 },
    { "slot": 4, "id": "GU_THUNDER_BURST", "cooldown_ticks_remaining": 0 },
    { "slot": 5, "id": "GU_BLADE_AURA", "cooldown_ticks_remaining": 40 },
    { "slot": 6, "id": "GU_HEAL_MEND", "cooldown_ticks_remaining": 0 }
  ]
}
```

### 2.3. Mở Rộng Snapshot Entity (`WorldEntityState` trong `Protocol.ts`)
Mỗi chu kỳ Snapshot (20Hz), Server xuất kèm trường `transformation`:
```typescript
export interface WorldTransformationState {
  active: boolean;
  form_id: string;
  form_name: string;
  form_kind: "FULL" | "PARTIAL" | "GU_HOUSE";
  model_scale: number;
  collision_radius: number;
  max_form_hp: number;
  current_form_hp: number;
  is_windup: boolean;
  windup_progress?: number; // 0.0 -> 1.0
}
```

---

## 3. Kiến Trúc Hiển Thị Client (Client Rendering Pipeline)

### 3.1. Canvas Rendering (`WorldScene.ts` / Entity Sprite)
1. **Scale Model & Sprite**:
   - Khi `entity.transformation.active = true` và `model_scale > 1.0`:
     ```typescript
     ctx.save();
     ctx.translate(renderX, renderY);
     ctx.scale(entity.transformation.model_scale, entity.transformation.model_scale);
     // Render Avatar / Beast Sprite
     drawEntitySprite(ctx, entity);
     ctx.restore();
     ```
2. **Vòng Tròn Va Chạm (Hitbox Circle)**:
   - Hitbox vòng tròn dưới chân thực thể vẽ theo `entity.radius` nhận được từ Snapshot. Thú to thì vòng tròn tự động phình to trực quan.
3. **Hào Quang Biến Thân (Aura Glow VFX)**:
   - Tùy thuộc vào `form_id`:
     - `FORM_FLAME_BEAR`: Render vòng tròn lửa hạt lấp lánh (Radial Gradient cam/đỏ, chu kỳ nhấp nháy 0.8s).
     - `FORM_SIX_ARM_ICE_DEMON`: Hơi lạnh sương giá và gai băng tuyết (Cyan #58a6ff).
     - `FORM_RECKLESS_SAVAGE`: Huyết khí cuồn cuộn đỏ thẫm (#da3633).
     - `FORM_GU_HOUSE_*`: Kết giới hoàng kim kim cang trận lấp lánh (#d29922).

### 3.2. Hiển Thị Thanh Máu Kép (Dual Health Bar)
* **Thanh Form HP (Màu Tím `#bc8cff`)**:
  - Nằm ngay phía trên thanh máu đỏ truyền thống của Cổ Tiên.
  - Tỉ lệ chiều rộng: $\text{Width} = (\text{CurrentFormHP} / \text{MaxFormHP}) \times 100\%$.
  - Khi dính đòn, thanh Tím tụt trước kèm hiệu ứng lóe trắng (Hit Flash). Khi Form HP về 0, hiển thị hiệu ứng nứt vụn vỡ tan (Shatter particles).

### 3.3. Hoán Đổi UI Thanh Kỹ Năng (Action Bar Dynamic Swap)
1. **Chế độ Full Metamorphosis**:
   - Ngay khi Client nhận sự kiện `TRANSFORMATION_STARTED` (`form_kind == "FULL"`):
     - Action Bar đổi giao diện sang chủ đề thú (Khung viền rực lửa hoặc viền gai băng).
     - 6 phím tắt số [1]..[6] được bind tương ứng với 6 chiêu thức thú trong `form_skills`.
     - Phím [6] mang biểu tượng và nhãn `Giải Trừ Biến Thân`.
2. **Chế độ Hoàn Nguyên**:
   - Khi nhận `TRANSFORMATION_ENDED`:
     - Action Bar chuyển về giao diện nhân tộc bình thường.
     - 6 slot Cổ được phục hồi và cập nhật lại Cooldown overlay chính xác theo `restored_skills`.

---

## 4. Xử Lý Luồng Tương Tác & Chống Gian Lận (Authoritative Action Gating)
* Client gửi lệnh thi triển `world.cast` với số `slot` (1..6).
* Server Zone Sim là thực thể duy nhất quyết định lệnh cast trúng vào Cổ nhân hay Chiêu thú:
  - Nếu `Transformation.Active == true`, Server tra cứu trong `TransformedSkills[slot]`.
  - Mọi nỗ lực hack gọi kỹ năng ngoài Form sẽ bị Server trả về `CAST_REJECTED` với lý do `form_skill_locked`.
