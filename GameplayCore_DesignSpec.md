# Gameplay Core Technical Spec — Tu Tiên / Nuôi Cổ

## 1) Player Creation System

### 1.1 Game Flow (State Machine)
```text
START
  -> CreateCharacter
  -> ChooseStartingRegion
  -> ReceiveFirstGu
  -> VitalGuDecision
      -> (A) SetVitalNow -> ContinueJourney
      -> (B) DeferVital  -> ContinueJourney(VitalMissingConstraints)
```

### 1.2 Deferred Vital Gu Rules
- If player chooses **DeferVital**:
  - `player.vitalGuInstanceId = null`
  - All non-vital Gu are marked `activation_locked = true`
- Core rule:
  - Player must own + set exactly 1 Vital Gu before activating functional effects of other Gu.

### 1.3 Validation Rules
- On character creation completion:
  - Must have chosen starting region.
  - Must have received first Gu.
- Vital constraints:
  - At most one Vital Gu at a time.
  - Vital Gu cannot be sold/dropped/destroyed while active (design-safe rule).

---
## 3) Refining Algorithm (Luyện hóa)

## 3.1 Design Goals
- Every newly acquired Gu starts at `refinePercent = 0`.
- Equip/activation requires `refinePercent == 100`.
- Refining consumes resource over time.
- If Vital Gu level > target Gu level: faster and cheaper.

## 3.2 Variables
- `LvV`: Vital Gu level (if absent, treat as null)
- `LvT`: Target Gu level
- `D = LvV - LvT` (only when Vital exists)
- `baseRate`: base refine % gain per second
- `baseCostP`: base primeval essence cost per second
- `baseCostI`: base immortal essence cost per second (optional for high-rank)

## 3.3 Multipliers
### Speed multiplier `M_speed`
- If no Vital Gu: `M_speed = 0.60`
- If Vital exists:
  - `D <= -3 => 0.70`
  - `-2 <= D <= -1 => 0.85`
  - `D == 0 => 1.00`
  - `1 <= D <= 2 => 1.20`
  - `3 <= D <= 5 => 1.45`
  - `D >= 6 => 1.70` (cap)

### Cost multiplier `M_cost`
- If no Vital Gu: `M_cost = 1.35`
- If Vital exists:
  - `D <= -3 => 1.30`
  - `-2 <= D <= -1 => 1.15`
  - `D == 0 => 1.00`
  - `1 <= D <= 2 => 0.90`
  - `3 <= D <= 5 => 0.78`
  - `D >= 6 => 0.65` (floor)

## 3.4 Per-tick Formula
Given tick duration `dt` (seconds):
- `refineGain = baseRate * M_speed * dt`
- `primevalCost = baseCostP * M_cost * dt`
- `immortalCost = baseCostI * M_cost * dt`

Update:
1. If not enough resource -> stop refine tick (or partial tick depending implementation).
2. Deduct resources.
3. `target.refinePercent = min(100, target.refinePercent + refineGain)`.
4. If reaches 100:
   - `target.isRefined = true`
   - unlock equip state (`activationLocked` still follows Vital rule).


## 4) Mastery / Proficiency System

### 4.1 Unlock Condition
- Only after `refinePercent == 100`.

### 4.2 Gain Rule
- Per valid usage event, award mastery XP:
  - `xpGain = baseXpByClass * encounterModifier * difficultyModifier`

### 4.3 Level Curve
- `xpRequired(level) = 100 * level^1.35`

### 4.4 Effect Scaling Examples
- Damage Gu: `finalDamage = baseDamage * (1 + 0.02 * masteryLevel)`
- Heal Gu: `finalHeal = baseHeal * (1 + 0.018 * masteryLevel)`
- Cooldown reduction cap:
  - `effectiveCd = baseCd * max(0.65, 1 - 0.01 * masteryLevel)`
- Primeval cost reduction cap:
  - `effectiveCost = baseCost * max(0.70, 1 - 0.008 * masteryLevel)`

---

## 5) Gu Class Logic

### 5.1 Active Gu
- Requires command activation.
- Consumes resources + starts cooldown.
- Allowed in combat turns.

### 5.2 Support Gu
- Activation-based buff/heal/control.
- Can target self/ally/enemy based on template rules.

### 5.3 Passive Gu
- No manual activation.
- Effects apply when equipped and activation condition valid.

### 5.4 Consumable Gu
- On successful use, apply burst effect then decrement stack.
- If stack reaches 0 -> remove from inventory.

---

## 6) `CanActivateGu()` Function Spec

## 6.1 Functional Requirements
`CanActivateGu(player, gu, context)` must check:
1. Ownership (`gu` in `player.guInventory`).
2. Refinement complete (`refinePercent >= 100`).
3. Vital dependency:
   - If `gu.isVital == false` and player has no vital -> deny.
4. Locked state (`activationLocked == false` unless this is the selected Vital Gu).
5. Resource affordability (if class requires resource).
6. Cooldown readiness.
7. Context rules (combat/non-combat/class-specific).

## 7) Integration Notes (Practical)
- During deferred Vital phase:
  - UI must show warning banner: "Chưa có Bản mệnh Cổ: các Cổ thường bị khóa kích hoạt".
  - Equip/activate buttons disabled for non-vital Gu.
- Backend should enforce all checks server-side; UI checks are advisory only.
- Use event logs for transparency:
  - `VITAL_SET`, `REFINE_TICK`, `REFINE_COMPLETE`, `GU_ACTIVATION_DENIED`, `GU_USED`.

