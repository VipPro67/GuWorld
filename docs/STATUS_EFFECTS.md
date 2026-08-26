# Status Effects — Hệ thống Hiệu Ứng Chiến Đấu

## Tổng quan

Mỗi hiệu ứng đều có **tier (cấp độ) 1–5**, tier càng cao thì hiệu lực càng mạnh. Giá trị thực tế được tính bởi `TierToPercent()` và `TierToDuration()` trong `simulator/models.go`.

Hiệu ứng được chia thành 2 loại:
- **Debuff** (giảm lực): slow, weak, freeze, silence, stun, bleed, poison
- **Buff** (tăng lực): thorns, regen, atk_up, def_up, taunt

---

## Bảng tier chi tiết

### Debuff (Áp lên kẻ địch)

| Hiệu ứng | T1 | T2 | T3 | T4 | T5 | Cơ chế |
|-----------|-----|-----|-----|-----|-----|--------|
| **Slow** (Chậm) | 15% / 3 lượt | 25% / 3 lượt | 35% / 4 lượt | 45% / 4 lượt | 55% / 5 lượt | Giảm speed → thay đổi thứ tự lượt |
| **Weak** (Yếu) | 10% / 3 lượt | 20% / 3 lượt | 30% / 4 lượt | 40% / 4 lượt | 50% / 5 lượt | Giảm % sát thương gây ra |
| **Freeze** (Đóng băng) | 1 lượt | 1 lượt | 2 lượt | 2 lượt | 3 lượt | Mất lượt, **bị phá khi nhận damage (50%)** |
| **Silence** (Im lặng) | 2 lượt | 2 lượt | 3 lượt | 3 lượt | 4 lượt | Không dùng được skill, chỉ đánh thường |
| **Stun** (Choáng) | 1 lượt | 1 lượt | 2 lượt | 2 lượt | 3 lượt | Mất lượt hoàn toàn, **không bị phá** |
| **Bleed** (Xuất huyết) | — | — | — | — | — | DoT: trừ HP mỗi lượt (giá trị cố định) |
| **Poison** (Độc) | — | — | — | — | — | DoT: trừ HP mỗi lượt (giá trị cố định) |

### Buff (Áp lên bản thân)

| Hiệu ứng | T1 | T2 | T3 | T4 | T5 | Cơ chế |
|-----------|-----|-----|-----|-----|-----|--------|
| **Thorns** (Khiên Gai) | 10% / 3 lượt | 15% / 3 lượt | 20% / 4 lượt | 25% / 4 lượt | 30% / 5 lượt | Phản % sát thương nhận vào cho kẻ tấn công |
| **Regen** (Hồi Sinh) | 3% HP max / 3t | 5% / 4t | 8% / 4t | 12% / 5t | 16% / 5t | Hồi HP mỗi lượt (hoạt động ngược với bleed/poison) |
| **Atk Up** (Tăng Công) | 10% / 3 lượt | 15% / 3 lượt | 20% / 4 lượt | 25% / 4 lượt | 30% / 5 lượt | Tăng % sát thương gây ra |
| **Def Up** (Tăng Thủ) | 10% / 3 lượt | 15% / 3 lượt | 20% / 4 lượt | 25% / 4 lượt | 30% / 5 lượt | Tăng % phòng thủ (giảm damage nhận vào) |
| **Taunt** (Khiêu khích) | 2 lượt | 2 lượt | 3 lượt | 3 lượt | 4 lượt | Buộc kẻ địch phải tấn công mình |

---

## Chi tiết từng hiệu ứng

### 1. Slow (Chẩm)

**Tác động:** Giảm speed → thay đổi thứ tự lượt trong lượt hiện tại.

```
EffectiveSpeed = BaseSpeed × (1 - SlowPercent / 100)
```

- Ví dụ: T3 Slow (35%) áp lên target có Speed 100 → EffectiveSpeed = 65
- Sort lượt: `sort.SliceStable(allActors, func(i, j) { return EffectiveSpeed_i > EffectiveSpeed_j })`
- Slow T5 (55%) có thể đảo ngược hoàn toàn thứ tự lượt

**Triển khai:**
- Auto-battle: `simulator/engine.go` — sort theo `EffectiveSpeed`; mỗi đòn gây sát thương bị nạn nhân đáp trả ngay 1 action (`tryCounter`), speed quyết định ai hành động trước
- Manual combat: `combat_attack.go` — `effectiveCombatSpeed` (tính slow) dùng cho sort toàn cục và quyết định bên nào mở màn chuỗi đánh–đáp với mục tiêu chủ

---

### 2. Weak (Yếu)

**Tác động:** Giảm % sát thương gây ra (tất cả đòn đánh).

```
FinalDamage = BaseDamage × DamageMultiplier
DamageMultiplier = (1 - WeakPercent / 100)
```

- Ví dụ: T4 Weak (40%) + đòn đánh 100 dmg → gây 60 dmg
- Weak áp lên **caster**, giảm damage của caster gây ra
- Kết hợp được với Atk Up (cộng dồn multiplier)

**Triển khai:**
- `actor.DamageMultiplier` được tính bởi `ComputeEffectMultipliers()`
- Áp dụng khi tính `dmgMultiplier` trong engine.go và dmgVal trong combat_attack.go

---

### 3. Freeze (Đóng băng)

**Tác động:** Mất lượt (giống stun) nhưng **có thể bị phá khi nhận damage** (50% chance).

```go
// Trong applyDamageToTarget():
if target.IsFrozen && damage > 0 {
    if rand.Float64() < 0.50 {
        // Bỏ effect freeze, target thoát đóng băng
    }
}
```

- Khác với Stun: Stun không bị phá bởi damage
- Khác với Stun: Freeze có thể bị phá ngay trong lượt của attacker
- Freeze break xảy ra **trước khi** damage thực sự đượcapply (freeze break log riêng)

**Triển khai:**
- `target.IsFrozen` flag — reset mỗi đầu lượt bởi `ComputeEffectMultipliers()`
- Break logic trong `applyDamageToTarget()` và sau khi deal damage

---

### 4. Silence (Im lặng)

**Tác động:** Không thể dùng kỹ năng (GU_SKILL, KILLER_MOVE), chỉ được dùng BASIC_ATTACK.

```go
// Trong action loop:
if actor.IsSilenced && a.ActionType != "BASIC_ATTACK" {
    continue // skip skill actions
}
```

- Nếu cả inventory đều là skills → forced basic attack fallback
- Silence **không ngăn** items, stones, charging
- Silence **không ngăn** regen/def_up/atk_up buffs (chỉ ngăn offensive skills)

**Triển khai:**
- `actor.IsSilenced` flag — reset mỗi đầu lượt bởi `ComputeEffectMultipliers()`
- Auto-battle: skip actions trong loop
- Manual combat: `isSilenced` check → force basic attack

---

### 5. Stun (Choáng)

**Tác động:** Mất lượt hoàn toàn. **Không bị phá bởi damage.**

- Stun **không stack**: nếu target đã bị stun, stun mới chỉ refresh duration
- GU_MY_WILL (Ngã Ý Cổ): 35% chance tự giải stun đầu lượt

**Triển khai:**
- Kiểm tra `eff.Type == "stun"` hoặc `eff.ID == EffectStun` trong Execute
- Auto-battle: `continue` (skip actor entirely)

---

### 6. Bleed & Poison (Xuất huyết & Độc)

**Tác động:** DoT — trừ HP mỗi lượt.

```
HP -= StatusTickDamage  (mỗi lượt)
Turns--                (giảm lượt)
```

- Bleed và Poison **không stack** với chính nó (effect mới thay effect cũ)
- Damage được apply **đầu lượt**, trước khi actor hành động

**Triển khai:**
- `DoTEffect{Type, Value, Turns}` trong `ActiveDoT` array
- Tick bởi `tickDoTs()` mỗi đầu lượt

---

### 7. Thorns (Khiên Gai)

**Tác động:** Phản % sát thương nhận vào cho kẻ tấn công.

```
ThornsDamage = DamageDealt × (ThornsPercent / 100)
```

- Ví dụ: T3 Thorns (20%) + nhận 100 dmg → phản 20 dmg lại attacker
- Thorns damage bị trừ bởi defense của attacker
- Thorns **không bypass shield**
- Có thể stack nếu có nhiều nguồn thorns

**Triển khai:**
- `target.ThornsDamage` = sum of all thorns buff values (%)
- `ApplyThornsDamage()` được gọi sau mỗi đòn đánh

---

### 8. Regen (Hồi Sinh)

**Tác động:** Hồi HP mỗi lượt.

```
HealPerTurn = MaxHP × (RegenPercent / 100)
HP += HealPerTurn  (mỗi lượt, capped tại MaxHP)
```

- Regen được xử lý như DoT (trong `ActiveDoT` với type `"regen"`)
- Khác với bleed/poison: regen **cộng HP** thay vì trừ

**Triển khai:**
- `DoTEffect{Type: "regen", Value: healPerTurn, Turns: duration}`
- Tick bởi `tickDoTs()` — branch `case "regen"`

---

### 9. Atk Up (Tăng Công Kích)

**Tác động:** Tăng % sát thương gây ra.

```
DamageMultiplier += (AtkUpPercent / 100)
```

- Kết hợp multiplicative với weak: `TotalMult = (1 - weak%) × (1 + atkUp%)`
- Ví dụ: Atk Up T3 (20%) + Weak T2 (20%) → `0.8 × 1.2 = 0.96` (giảm nhẹ)

**Triển khai:**
- `actor.DamageMultiplier` được tính bởi `ComputeEffectMultipliers()`

---

### 10. Def Up (Tăng Phòng Thủ)

**Tác động:** Tăng % phòng thủ (giảm damage nhận vào).

```
EffectiveDefense = BaseDefense × DefenseMultiplier
DefenseMultiplier = 1 + (DefUpPercent / 100)
```

- Ví dụ: Def Up T4 (25%) + BaseDef 40 → EffectiveDef = 50

**Triển khai:**
- `target.DefenseMultiplier` được tính bởi `ComputeEffectMultipliers()`
- Áp dụng khi tính `targetDef` trong damage formula

---

### 11. Taunt (Khiêu khích)

**Tác động:** Buộc kẻ địch phải tấn công mình.

```go
// Target selection:
for _, enemy := range enemies {
    if enemy.HasTaunt && enemy.HP > 0 {
        target = enemy // redirect
        break
    }
}
```

- Taunt **ghi đè** logic target mặc định (front → mid → back)
- Nếu có nhiều taunter → ưu tiên taunter HP cao nhất
- Taunt **không ngăn** AOE skills (vẫn damage tất cả)

**Triển khai:**
- `actor.HasTaunt` flag — reset mỗi đầu lượt bởi `ComputeEffectMultipliers()`
- Redirect target trong `getTarget()` hoặc manual target selection

---

## Flow xử lý mỗi lượt

```
1. Tick DoTs (bleed, poison, regen)        ← tickDoTs()
2. Tick status effect durations             ← tickStatusEffects()
3. Compute derived multipliers             ← ComputeEffectMultipliers()
   - EffectiveSpeed (slow)
   - DamageMultiplier (weak, atk_up)
   - DefenseMultiplier (def_up)
   - ThornsDamage (thorns)
   - IsSilenced, IsFrozen, HasTaunt
   - IsSleeping, IsFeared, IsCharmed, IsStealthed, IsConfused, IsBlinded, IsBound
4. Sort actors theo EffectiveSpeed
5. Với mỗi actor:
   a. Check stun → skip lượt
   b. Check freeze → skip lượt
   c. Check sleep → skip lượt
   d. Check bind → skip lượt
   e. Check silence/fear → force basic attack
   f. Charm/confuse → redirect target sang phe mình
   g. Taunt redirect target
   h. Execute action:
      - Blind → xác suất miss
      - Apply weak/atk_up multiplier to damage
      - Apply def_up to target defense
      - Deal damage (sleep/freeze break on damage)
      - Apply thorns reflect
      - Apply status effects to target (slow, weak, freeze, silence, burn, ...)
      - Apply self-buffs (thorns, regen, atk_up, def_up, taunt, stealth)
```

---

## Cách áp dụng hiệu ứng

### Từ Gu Dictionary (CSV/DB)

Set `apply_status` trong `effects.active`:

```json
{
  "active": {
    "apply_status": "SLOW",
    "status_tick_damage": 3,
    "status_duration_ticks": 4
  }
}
```

**Supported `apply_status` values:**
`BLEED`, `POISON`, `STUN`, `SLOW`, `SLOW1`, `SLOW_DAMAGE`, `WEAK`, `FREEZE`, `SILENCE`, `THORNS`, `REGEN`, `ATK_UP`, `DEF_UP`, `TAUNT`, `BURN`, `FEAR`, `CONFUSE`, `STEALTH`, `SLEEP`, `BLIND`, `CHARM`, `BIND`

- `status_tick_damage`: tier (1-5) cho effects cũ, hoặc giá trị/lượt (DoT) cho status registry
- `status_duration_ticks`: duration override (nếu 0 → dùng default trong registry)

### Registry status (mới — data-driven qua `statusfx`)

Từ đợt này, các status mới được chuẩn hóa về **catalog semantics** và quản lý tập trung trong package `internal/statusfx` (không còn if-chain rải rác):

| Status | Behaviour | Cơ chế | Dur mặc định | Resist |
|--------|-----------|--------|-------------|--------|
| **BURN** (Thiêu Đốt) | `dot` | DoT hỏa: trừ `status_tick_damage` HP mỗi lượt | 3 | ❌ |
| **FEAR** (Khiếp Sợ) | `force_basic` | Chỉ tấn công cơ bản (như silence) | 2 | ✅ |
| **CONFUSE** (Mê Loạn) | `confuse` | 40% tấn công nhầm đồng minh | 2 | ✅ |
| **STEALTH** (Ẩn Thân) | `stealth` | Không bị chọn mục tiêu (single-target) | 2 | ❌ (self) |
| **SLEEP** (Ngủ) | `sleep` | Mất lượt, **thức dậy khi bị tấn công** | 2 | ✅ |
| **BLIND** (Mù) | `blind` | 35% đòn đánh trượt | 2 | ✅ |
| **CHARM** (Mê Hoặc) | `charm` | Bị điều khiển, tấn công phe mình | 3 | ✅ |
| **BIND** (Trói) | `bind` | Mất lượt (không hành động) | 3 | ✅ |
| **HASTE** (Tăng Tốc) | `haste` | Tăng % tốc độ (self) | 3 | ❌ (self) |
| **ARMOR_BREAK** (Phá Giáp) | `armor_break` | Giảm % phòng thủ | 3 | ✅ |
| **DAMAGE_REDUCTION_UP** (Giảm Sát Thương) | `damage_reduction` | Giảm % sát thương nhận vào (self) | 3 | ❌ (self) |
| **INVINCIBLE** (Bất Tử) | `invincible` | Miễn nhiễm mọi sát thương (self) | 1 | ❌ (self) |
| **REVIVE_LOCK** (Khóa Hồi Sinh) | `revive_lock` | Chặn hồi sinh (Xuân Thu Thiền) | 3 | ✅ |

**Thêm status mới = 1 entry trong `statusfx.Registry` + tùy chọn 1 behaviour hook trong `simulator/status_apply.go`.** Không cần sửa switch nữa.

### Chênh lệch dữ liệu hiện tại (quét `data/Gu.csv`)

Bảng dưới đối chiếu status **có trong Gu dictionary** với **code combat đã xử lý**. Xem thêm `docs/GAME_DATA.md` → "Status chiến đấu (combat)".

| Status | Số Cổ trong data | Code combat | Ghi chú |
|---|---:|---|---|
| BURN | 32 | ✅ (registry) | DoT hỏa qua `statusfx` |
| BLEED | 30 | ✅ | — |
| FREEZE | 13 | ✅ | — |
| FEAR | 5 | ✅ (registry) | force basic, resist-gated |
| STEALTH | 4 | ✅ (registry) | untargetable single-target |
| CONFUSE | 4 | ✅ (registry) | 40% mis-target |
| SILENCE | 3 | ✅ | — |
| STUN | 3 | ✅ | qua shorthand `stun` + `apply_status` |
| SLEEP | 2 | ✅ (registry) | skip turn + break on damage |
| POISON | 2 | ✅ | — |
| CHARM / BIND / BLIND / SLOW / SLOW1 | 1 mỗi loại | ✅ (registry cho CHARM/BIND/BLIND; SLOW/SLOW1 tier) | — |

Lưu ý: `THORNS`, `REGEN`, `ATK_UP`, `DEF_UP`, `TAUNT`, `WEAK`, `SLOW_DAMAGE` đã có code (constant + flow) nhưng **hiện không Cổ nào trong `Gu.csv` sử dụng** — chỉ phát sinh qua KillerMove hoặc sinh ra bởi code.

Catalog đã đăng ký trong `statusfx.Registry` nhưng **chưa wire combat**: `HASTE`, `ARMOR_BREAK`, `DAMAGE_REDUCTION_UP`, `INVINCIBLE`, `REVIVE_LOCK`.

### Từ KillerMove (JSON effects)

```json
{
  "slow": 3,
  "weak": 2,
  "freeze": 1,
  "silence": 2,
  "thorns_buff": 2,
  "regen": 1,
  "atk_up": 2,
  "def_up": 1,
  "taunt": 1,
  "bleed": 15.0,
  "stun": 2
}
```

---

## Code references

| File | Mục đích |
|------|----------|
| `statusfx/status.go` | Canonical status IDs + behaviour tags |
| `statusfx/spec.go` | `StatusSpec` + `Registry` + `ResolveStatus()` |
| `simulator/status_apply.go` | `applyStatusOps()` — registry-driven apply/tick |
| `simulator/models.go` | Constants, `TierToPercent()`, `TierToDuration()`, `SimAction` fields |
| `simulator/mechanics.go` | `tickDoTs()`, `tickStatusEffects()`, `ComputeEffectMultipliers()`, `ApplyThornsDamage()` |
| `simulator/engine.go` | Auto-battle: sort, freeze/stun/silence/taunt checks, damage calc, effect application |
| `handlers/combat_attack.go` | Manual combat: tick loop, slow sort, weak damage, silence/freeze checks |
| `handlers/combat_auto.go` | Map Gu/KillerMove effects → SimAction fields (BLEED/POISON + registry statuses → StatusOps) |
| `handlers/gu_effects.go` | `BuildCombatAction()`, `GetFlatGuEffects()`, `tickStatusEffects()` |
| `models/gu.go` | `GetFlatGuEffects()` — flat effect mapping |
