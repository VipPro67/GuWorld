# Gu Combat Skill Profile - Bảng Mẫu & Thư Viện Archetypes

Tài liệu này là cẩm nang tra cứu và thư viện mẫu JSON cho cột `skill_profile` trong `backend/data/Gu.csv`. Khi muốn tạo mới hoặc tinh chỉnh chiêu thức cho bất kỳ Cổ nào, bạn chỉ cần copy mẫu JSON tương ứng, sửa lại các chỉ số và dán trực tiếp vào cột `skill_profile` của Cổ đó.

---

## 1. Bảng Tra Cứu Toàn Bộ Thuộc Tính (Specification)

| Nhóm | Thuộc tính | Kiểu dữ liệu | Mặc định | Ý nghĩa & Mô tả |
| :--- | :--- | :--- | :--- | :--- |
| **Định Danh** | `skill_name` | string | Tên Cổ | Tên chiêu thức hiển thị trên giao diện chiến đấu |
| | `skill_icon` | string | `""` | Đường dẫn icon kỹ năng (VD: `gu/fire`, `gu/water`) |
| | `slot` | int | `0` | Slot gán mặc định (1..6). 0 là tự động sắp xếp vào túi |
| **Tiêu Hao** | `mana_cost` | float | Theo Rank | Chân nguyên tiêu hao khi xuất chiêu |
| | `hp_cost` | float | `0` | Máu tiêu hao (thường dùng cho Ma đạo, Huyết đạo) |
| | `stamina_cost` | float | `0` | Thể lực tiêu hao khi vận chiêu |
| **Định Hướng** | `cast_mode` | enum | `TARGET` | **`TARGET`** (Khóa mục tiêu), **`DIRECTION`** (Định hướng/Skillshot), **`GROUND`** (Chỉ định vùng đất), **`SELF`** (Bản thân) |
| | `shape` | enum | `SINGLE` | **`SINGLE`** (Đơn thể), **`CIRCLE`** (Tròn), **`CONE`** (Phễu hình nón), **`RECTANGLE`** (Đoạn chữ nhật xuyên thấu) |
| **Hình Học** | `cast_range` | float | `9.0` | Tầm thi triển tối đa (đơn vị: mét) |
| | `radius` | float | `0` | Bán kính vụ nổ/vùng ảnh hưởng (`CIRCLE`, `CONE`) |
| | `angle_deg` | float | `0` | Góc quét phễu tính theo độ (`CONE`, VD: 60°, 75°) |
| | `length` | float | `0` | Chiều dài vệt đòn (`RECTANGLE`, VD: 16.0) |
| | `width` | float | `0` | Bề rộng vệt đòn (`RECTANGLE`, VD: 1.8) |
| | `max_targets` | int | `1` | Số lượng mục tiêu tối đa trúng đòn đồng thời |
| **Thời Gian** | `cast_time_ms` | int | Theo Rank | Thời gian niệm / gồng trước khi tung chiêu (ms) |
| | `recovery_ms` | int | `150` | Thời gian khựng sau khi tung chiêu trước khi hành động tiếp (ms) |
| | `cooldown_ms` | int | Theo Rank | Thời gian hồi chiêu (ms). Rank 1: 2000, Rank 5: 9000, Rank 9: 22000 |
| **Đường Đạn** | `projectile_speed` | float | `0` | Tốc độ bay của đường đạn (VD: 18.0 - 28.0) |
| | `pierce` | bool | `false` | Đạn có xuyên qua nhiều mục tiêu hay không |
| | `spread_count` | int | `1` | Số lượng tia đạn phóng ra cùng lúc (chùm) |
| | `spread_angle_deg`| float | `0` | Góc xòe chùm đạn (VD: 25°) |
| **Cơ Động** | `move_distance` | float | `0` | Khoảng cách lướt/đột kích của nhân vật |
| | `knockback_dist` | float | `0` | Khoảng cách đẩy lùi kẻ địch khi trúng đòn |
| **Sát Thương** | `damage_base` | float | Theo Rank | Lượng sát thương cơ bản của chiêu |
| | `element_type` | string | Path Cổ | Hệ nguyên tố: `Fire`, `Water`, `Ice`, `Poison`, `Lightning`, `Wind`, `Earth`, `Sword`, `Light`, `Dark`, `Blood`... |
| | `damage_type` | string | `PATH` | Loại sát thương: `PATH` (Đạo ngân), `PHYSICAL` (Vật lý), `TRUE` (Chuẩn) |
| | `player_atk_ratio`| float | `0.8 - 1.7`| Tỉ lệ cộng hưởng theo Công kích nhân vật (VD: 1.0 = 100% Công) |
| | `gu_power_ratio` | float | `1.0 - 2.3`| Tỉ lệ cộng hưởng theo Cổ Lực (VD: 1.2 = 120% Cổ Lực) |
| **Hỗ Trợ** | `shield_base` | float | `0` | Lượng Giáp/Khiên tạo ra |
| | `heal_base` | float | `0` | Lượng Máu hồi phục trực tiếp |
| | `cleanse_debuff` | bool | `false` | Có giải trừ các hiệu ứng xấu (debuff) hay không |
| **Hiệu Ứng (CC)**| `status_id` | string | `""` | Mã trạng thái: `STUN`, `FREEZE`, `SLOW`, `BLEED`, `POISON`, `SILENCE`, `BURN` |
| | `effect_chance` | float | `1.0` | Tỉ lệ kích hoạt hiệu ứng (0.0 đến 1.0) |
| | `duration_ms` | int | `0` | Thời gian duy trì hiệu ứng khống chế/DoT (ms) |
| | `dot_period_ms` | int | `1000` | Khoảng cách giữa mỗi lần nảy dame DoT (ms) |
| | `dot_tick_value` | float | `0` | Sát thương mỗi lần nảy DoT |
| | `status_move_mult`| float | `1.0` | Hệ số tốc độ chạy khi dính (0.0 = đứng im/đóng băng, 0.5 = làm chậm 50%) |
| **Hình & Âm** | `cast_vfx` | string | `""` | Hiệu ứng hình ảnh lúc gồng niệm |
| | `projectile_vfx` | string | `""` | Hiệu ứng hình ảnh đường đạn đang bay |
| | `impact_vfx` | string | `""` | Hiệu ứng hình ảnh khi trúng đích / phát nổ |
| | `cast_sfx` | string | `""` | Âm thanh xuất chiêu |
| | `hit_sfx` | string | `""` | Âm thanh khi đòn đánh trúng mục tiêu |
| | `camera_shake_intensity` | float | `0.05 - 0.5` | Độ rung màn hình tạo cảm giác lực đòn đánh |

---

## 2. Thư Viện 10 Mẫu Chiêu Thức Chuẩn (Templates)

### Mẫu 1: Đạn Định Hướng Tầm Xa (Directional Skillshot)
> **Ví dụ điển hình:** Hỏa Cầu Cổ, Thủy Tiễn Cổ, Lôi Điện Cổ.
> **Đặc điểm:** Bắn ra một viên đạn bay theo hướng chỉ định, gây sát thương khi chạm phải mục tiêu đầu tiên.

```json
{
  "cast_mode": "DIRECTION",
  "cast_range": 16.0,
  "projectile_speed": 20.0,
  "radius": 0.35,
  "cast_time_ms": 150,
  "recovery_ms": 150,
  "cooldown_ms": 2500,
  "mana_cost": 10.0,
  "damage_base": 65.0,
  "player_atk_ratio": 1.0,
  "gu_power_ratio": 1.2,
  "element_type": "Fire",
  "damage_type": "PATH",
  "cast_vfx": "cast/fire",
  "projectile_vfx": "projectiles/fireball",
  "impact_vfx": "impact/fire_burst",
  "cast_sfx": "sfx/fire_cast",
  "hit_sfx": "sfx/fire_hit",
  "camera_shake_intensity": 0.1
}
```

---

### Mẫu 2: Bắn Chùm Nhiều Tia (Multi-Shot / Spread Shot)
> **Ví dụ điển hình:** Nhị Chuyển Độc Châm Cổ, Tán Tiễn Cổ, Thất Tinh Quang.
> **Đặc điểm:** Phóng ra nhiều đường đạn cùng lúc hình cánh quạt, áp sát có thể trúng nhiều tia dồn dame.

```json
{
  "cast_mode": "DIRECTION",
  "cast_range": 14.0,
  "projectile_speed": 22.0,
  "radius": 0.3,
  "spread_count": 5,
  "spread_angle_deg": 30.0,
  "cast_time_ms": 150,
  "recovery_ms": 150,
  "cooldown_ms": 3200,
  "mana_cost": 15.0,
  "damage_base": 40.0,
  "player_atk_ratio": 0.8,
  "gu_power_ratio": 1.0,
  "element_type": "Poison",
  "damage_type": "PATH",
  "status_id": "POISON",
  "effect_chance": 0.8,
  "duration_ms": 3000,
  "dot_period_ms": 1000,
  "dot_tick_value": 15.0,
  "cast_vfx": "cast/poison",
  "projectile_vfx": "projectiles/poison_spit",
  "impact_vfx": "impact/poison_cloud",
  "cast_sfx": "sfx/poison_cast",
  "hit_sfx": "sfx/poison_hit"
}
```

---

### Mẫu 3: Đòn Chém Quét Cận Chiến Hình Nón (Frontal Cone Cleave)
> **Ví dụ điển hình:** Đao Phong Cổ, Kiếm Khí Cổ, Hỏa Diễm Phún Cổ.
> **Đặc điểm:** Quét một hình nón diện rộng phía trước mặt, trúng tất cả kẻ địch trong vùng.

```json
{
  "cast_mode": "GROUND",
  "shape": "CONE",
  "cast_range": 4.5,
  "radius": 4.5,
  "angle_deg": 75.0,
  "max_targets": 6,
  "cast_time_ms": 150,
  "recovery_ms": 150,
  "cooldown_ms": 2800,
  "mana_cost": 12.0,
  "damage_base": 70.0,
  "player_atk_ratio": 1.1,
  "gu_power_ratio": 1.1,
  "knockback_dist": 1.5,
  "element_type": "Blade",
  "damage_type": "PATH",
  "cast_vfx": "cast/sword",
  "impact_vfx": "impact/sword_slash",
  "cast_sfx": "sfx/slash_cast",
  "hit_sfx": "sfx/slash_hit",
  "camera_shake_intensity": 0.15
}
```

---

### Mẫu 4: Vệt Kiếm / Tia Sáng Xuyên Thấu (Linear Piercing Beam)
> **Ví dụ điển hình:** Nguyệt Quang Cổ, Nguyệt Ngân Cổ, Bảo Nguyệt Quang Vương Cổ, Lôi Lạc Cổ.
> **Đặc điểm:** Bắn ra một dải sáng hình chữ nhật dài, xuyên thấu toàn bộ mục tiêu trên đường thẳng.

```json
{
  "cast_mode": "DIRECTION",
  "shape": "RECTANGLE",
  "cast_range": 18.0,
  "length": 18.0,
  "width": 1.8,
  "pierce": true,
  "max_targets": 8,
  "cast_time_ms": 150,
  "recovery_ms": 150,
  "cooldown_ms": 3200,
  "mana_cost": 15.0,
  "damage_base": 75.0,
  "player_atk_ratio": 1.0,
  "gu_power_ratio": 1.25,
  "element_type": "Light",
  "damage_type": "PATH",
  "cast_vfx": "cast/light",
  "projectile_vfx": "projectiles/moon_blade",
  "impact_vfx": "impact/light_slash",
  "cast_sfx": "sfx/slash_cast",
  "hit_sfx": "sfx/slash_hit",
  "camera_shake_intensity": 0.12
}
```

---

### Mẫu 5: Ném Búa / Thả Thiên Thạch Vùng Đất Chỉ Định (Target Ground AOE)
> **Ví dụ điển hình:** Nhị Chuyển Loa Toàn Chùy Cổ, Lạc Thạch Cổ, Thiên Lôi Trận.
> **Đặc điểm:** Chọn một vị trí trên mặt đất trong tầm bắn, dội sát thương xuống tâm vùng tròn.

```json
{
  "cast_mode": "GROUND",
  "shape": "CIRCLE",
  "cast_range": 10.0,
  "radius": 2.5,
  "max_targets": 6,
  "cast_time_ms": 250,
  "recovery_ms": 150,
  "cooldown_ms": 3500,
  "mana_cost": 15.0,
  "damage_base": 85.0,
  "player_atk_ratio": 1.1,
  "gu_power_ratio": 1.3,
  "knockback_dist": 1.8,
  "element_type": "Earth",
  "damage_type": "PATH",
  "cast_vfx": "cast/earth",
  "impact_vfx": "impact/heavy_strike",
  "cast_sfx": "sfx/cast_generic",
  "hit_sfx": "sfx/hit_generic",
  "camera_shake_intensity": 0.25
}
```

---

### Mẫu 6: Hộ Thuẫn Tự Thân (Self-Barrier / Shield)
> **Ví dụ điển hình:** Thủy Thuẫn Cổ, Băng Thuẫn Cổ, Thổ Bích Cổ, Thú Thể Tiên Cổ.
> **Đặc điểm:** Tác dụng lên bản thân, tạo lớp giáp hấp thụ sát thương duy trì trong thời gian ngắn.

```json
{
  "cast_mode": "SELF",
  "shape": "SINGLE",
  "cast_time_ms": 100,
  "recovery_ms": 150,
  "cooldown_ms": 5000,
  "mana_cost": 12.0,
  "shield_base": 80.0,
  "shield_ratio": 1.0,
  "element_type": "Water",
  "damage_type": "PATH",
  "cast_vfx": "cast/water",
  "impact_vfx": "impact/water",
  "cast_sfx": "sfx/water_cast",
  "hit_sfx": "sfx/water_impact"
}
```

---

### Mẫu 7: Trị Liệu & Thanh Tẩy (Healing & Cleanse)
> **Ví dụ điển hình:** Huyết Liệu Cổ, Tuyệt Tẩy Cổ, Sinh Âm Chung Cổ, Mộc Linh Cổ.
> **Đặc điểm:** Tự hồi phục lượng lớn sinh lực bản thân hoặc đồng đội, đồng thời giải trừ toàn bộ debuff/đóng băng/trúng độc.

```json
{
  "cast_mode": "SELF",
  "shape": "SINGLE",
  "cast_time_ms": 150,
  "recovery_ms": 150,
  "cooldown_ms": 6500,
  "mana_cost": 20.0,
  "heal_base": 120.0,
  "heal_ratio": 1.0,
  "cleanse_debuff": true,
  "element_type": "Water",
  "damage_type": "PATH",
  "cast_vfx": "cast/water",
  "impact_vfx": "impact/water",
  "cast_sfx": "sfx/water_cast"
}
```

---

### Mẫu 8: Lướt / Đột Kích / Húc Bay (Charge / Gap Closer)
> **Ví dụ điển hình:** Hắc Thỉ Cổ, Bôn Lôi Cổ, Thiết Châm Đột.
> **Đặc điểm:** Lao vọt về phía mục tiêu hoặc hướng chỉ định, gây sát thương và hất văng địch.

```json
{
  "cast_mode": "TARGET",
  "cast_range": 7.0,
  "move_distance": 4.0,
  "knockback_dist": 2.5,
  "cast_time_ms": 150,
  "recovery_ms": 150,
  "cooldown_ms": 3200,
  "mana_cost": 10.0,
  "damage_base": 55.0,
  "player_atk_ratio": 1.2,
  "gu_power_ratio": 1.0,
  "element_type": "Strength",
  "damage_type": "PHYSICAL",
  "cast_vfx": "cast/earth",
  "impact_vfx": "impact/heavy_strike",
  "cast_sfx": "sfx/cast_generic",
  "hit_sfx": "sfx/hit_generic",
  "camera_shake_intensity": 0.2
}
```

---

### Mẫu 9: Khống Chế Cứng (Crowd Control - Freeze / Stun / Silence)
> **Ví dụ điển hình:** Bạch Sương Cổ (Freeze), Nhị Chuyển Cấm Chỉ Cổ (Silence), Toàn Phong Cổ (Stun).
> **Đặc điểm:** Khóa mục tiêu đơn thể hoặc vùng, khiến địch bị bất động hoàn toàn hoặc cấm tung chiêu.

```json
{
  "cast_mode": "TARGET",
  "cast_range": 9.0,
  "cast_time_ms": 200,
  "recovery_ms": 150,
  "cooldown_ms": 5000,
  "mana_cost": 15.0,
  "damage_base": 30.0,
  "player_atk_ratio": 0.5,
  "gu_power_ratio": 1.0,
  "element_type": "Ice",
  "damage_type": "PATH",
  "status_id": "FREEZE",
  "effect_chance": 1.0,
  "duration_ms": 1500,
  "status_move_mult": 0.0,
  "pvp_dr_category": "STUN",
  "cast_vfx": "cast/frost",
  "impact_vfx": "impact/frost",
  "cast_sfx": "sfx/ice_cast",
  "hit_sfx": "sfx/ice_hit",
  "camera_shake_intensity": 0.15
}
```

---

### Mẫu 10: Trảm Kích Đơn Thể & Chảy Máu (Single Heavy Strike + DoT Bleed)
> **Ví dụ điển hình:** Trảm Long Tiên Cổ, Tuyệt Mệnh Kiếm Cổ, Huyết Trảm Cổ.
> **Đặc điểm:** Dồn sát thương cực lớn vào 1 mục tiêu chỉ định, đồng thời gắn hiệu ứng Xuất Huyết rút máu liên tục.

```json
{
  "cast_mode": "TARGET",
  "cast_range": 10.0,
  "cast_time_ms": 250,
  "recovery_ms": 200,
  "cooldown_ms": 8000,
  "mana_cost": 25.0,
  "damage_base": 180.0,
  "player_atk_ratio": 1.5,
  "gu_power_ratio": 1.5,
  "element_type": "Sword",
  "damage_type": "PATH",
  "status_id": "BLEED",
  "effect_chance": 1.0,
  "duration_ms": 4000,
  "dot_period_ms": 1000,
  "dot_tick_value": 35.0,
  "cast_vfx": "cast/sword",
  "impact_vfx": "impact/sword_slash",
  "cast_sfx": "sfx/slash_cast",
  "hit_sfx": "sfx/slash_hit",
  "camera_shake_intensity": 0.3
}
```

---

## 3. Bảng Chuẩn Hồi Chiêu & Sát Thương Tham Khảo Theo Rank (Pacing Guide Chuẩn Mới)

| Rank | Thời gian niệm (`cast_time_ms`) | Hồi chiêu (`cooldown_ms`) | Tiêu hao Mana (`mana_cost`) | Sát thương cơ bản (`damage_base`) |
| :---: | :---: | :---: | :---: | :---: |
| **Rank 1** | $100\,\text{ms}$ | $1.000\,\text{ms}$ ($1.0\,\text{s}$) | $10$ (Chân nguyên) | $30$ |
| **Rank 2** | $100\,\text{ms}$ | $1.100\,\text{ms}$ ($1.1\,\text{s}$) | $10$ (Chân nguyên) | $60$ |
| **Rank 3** | $150\,\text{ms}$ | $1.200\,\text{ms}$ ($1.2\,\text{s}$) | $12$ (Chân nguyên) | $150$ |
| **Rank 4** | $150\,\text{ms}$ | $1.400\,\text{ms}$ ($1.4\,\text{s}$) | $15$ (Chân nguyên) | $600$ |
| **Rank 5** | $150\,\text{ms}$ | $1.500\,\text{ms}$ ($1.5\,\text{s}$) | $15$ (Chân nguyên) | $3.000$ |
| **Rank 6 (Tiên Cổ)** | $200\,\text{ms}$ | $1.600\,\text{ms}$ ($1.6\,\text{s}$) | **$1$** (Tiên nguyên) | $300.000$ |
| **Rank 7 (Tiên Cổ)** | $200\,\text{ms}$ | $1.800\,\text{ms}$ ($1.8\,\text{s}$) | **$2$** (Tiên nguyên) | $30.000.000$ |
| **Rank 8 (Tiên Cổ)** | $200\,\text{ms}$ | $1.900\,\text{ms}$ ($1.9\,\text{s}$) | **$3$** (Tiên nguyên) | $3.000.000.000$ |
| **Rank 9 (Tôn Giả)** | $200\,\text{ms}$ | $2.000\,\text{ms}$ ($2.0\,\text{s}$) | **$5$** (Tiên nguyên) | $300.000.000.000$ |
