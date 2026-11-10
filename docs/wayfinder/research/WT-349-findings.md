# WT-349: Killer move effect execution support

## Scope

Static trace of authored Gu effects through killer-move options, preview and persistence, combat action construction, and the turn-based simulator. No gameplay code was changed and no tests were run.

## Findings

| Question | Finding |
|---|---|
| Which authored effects can be selected? | `getKillerMoveEffectOptions` offers one option per operation from `ON_ACTION` or `ON_USE` blocks. Every `ON_ACTION` operation is labeled combat-supported without consulting `killerMoveOperationAllowed`; that helper's allowlist is currently unused. `ON_USE` is restricted to self/unspecified flat or percent HEAL. See [killer_moves.go](../../../backend/internal/handlers/killer_moves.go#L177-L234). The frontend filters options by selected Gu only, so it may display an `ON_USE` heal with its “outside combat” reason. [KillerMovesPanel.tsx](../../../frontend/src/components/KillerMovesPanel.tsx#L46-L48).
| Are conditions preserved? | No. Source blocks carry conditions, but each selected operation is rebuilt into a one-operation block with ID/trigger/target/operations only. The source block's `Conditions` are omitted. [killer_moves.go](../../../backend/internal/handlers/killer_moves.go#L183-L204) [killer_moves.go](../../../backend/internal/handlers/killer_moves.go#L374-L381). The generic compiler does evaluate/retain authored conditions, but this draft path does not pass them through. [runtime.go](../../../backend/internal/gameplay/effects/runtime.go#L127-L140).
| What is persisted? | Preview applies a power-budget scale to selected operations and returns compiled blocks. Save checks the preview fingerprint and adjustment acceptance, then stores `compiled_effects` JSONB. [killer_moves.go](../../../backend/internal/handlers/killer_moves.go#L423-L451) [killer_moves.go](../../../backend/internal/handlers/killer_moves.go#L691-L705).
| Does combat consume `CompiledOperations`? | No, in the current turn-based path. `buildSimActionFromGu` assigns `CompileCombatAction().Operations` to `SimAction.CompiledOperations`, but the simulator field is explicitly a temporary adapter and simulator code has no read site for it. Runtime behavior comes from legacy `SimAction` fields populated by the `Effects.Blocks` switch. [combat_auto_resolve.go](../../../backend/internal/handlers/combat_auto_resolve.go#L451-L453) [models.go](../../../backend/internal/simulator/models.go#L212-L214). `CompileCombatAction` itself flattens ON_ACTION operations and keeps only the last block target in its action target. [runtime.go](../../../backend/internal/gameplay/effects/runtime.go#L266-L275).
| Is DAMAGE FLAT dropped? | No. `CalculateGuEffectForCombatant` sums ON_ACTION `DAMAGE` with subtype `FLAT` (or empty subtype) into base damage. [combat_effects.go](../../../backend/internal/handlers/combat_effects.go#L53-L70). The adapter switch handles `DAMAGE` multipliers and max-HP percent; flat has already been accounted for by that base calculation. [combat_auto_resolve.go](../../../backend/internal/handlers/combat_auto_resolve.go#L539-L572). Other unrecognized DAMAGE subtypes have no mapping.
| Are multiple targets/operation targets independent? | No. The adapter maps each block/op target into the single `SimAction.TargetType`, overwriting earlier values; the simulator resolves one shared target list from that field. [combat_auto_resolve.go](../../../backend/internal/handlers/combat_auto_resolve.go#L539-L558) [engine.go](../../../backend/internal/simulator/engine.go#L704-L755). For example, a multi-target block followed by a SELF heal can overwrite target type to SELF. Per-op targeting is therefore not represented faithfully.
| Does component Gu rank affect rank scaling? | Selected effect refs retain each component's source rank, and the preview budget uses the main Gu rank. At combat build, `SimAction.Rank` is the main `spec.Rank`; STATUS chance/duration scaling also uses that rank, not the selected effect's source rank. [killer_moves.go](../../../backend/internal/handlers/killer_moves.go#L374-L380) [combat_auto_resolve.go](../../../backend/internal/handlers/combat_auto_resolve.go#L438-L449) [combat_auto_resolve.go](../../../backend/internal/handlers/combat_auto_resolve.go#L621-L642). Thus selected operations have no per-component rank attribution at execution.

## Operation mapping in the legacy turn-based adapter

All `ON_ACTION` operations can be offered, but runtime support is narrower than selection:

- `DAMAGE`: FLAT/empty through base-damage scan; MULTIPLIER and MAX_HP_PERCENT through adapter.
- `SHIELD`: FLAT/empty through base-defense scan; MULTIPLIER through adapter.
- `HEAL`: FLAT/empty, PERCENT, LIFESTEAL, and mana resource forms.
- `DIRECT_HP`: MAX_HP_PERCENT and SHIELD_STRIP.
- `BARRIER`: PERCENT or treated as flat for every other subtype.
- `CLEANSE`: sets the cleanse flag.
- `STATUS`: resolves status ID; rank scaling uses the main action rank.
- `STAT_MOD`: ATTACK is added to action damage; DEFENSE to defense; other stats ignored.
- `UNIQUE`: only SUMMON_PHANTOM, WOOD_TRANSFORMATION, BYPASS_SHIELD, GUARANTEED_HIT mapped here.
- Other operation types and unhandled subtypes are silently ignored by this adapter.

See [combat_auto_resolve.go](../../../backend/internal/handlers/combat_auto_resolve.go#L559-L662). `ON_USE` execution is separately implemented only for healing outside combat; the handler accepts only HEAL and applies FLAT/PERCENT values. [killer_moves.go](../../../backend/internal/handlers/killer_moves.go#L907-L908) [killer_moves.go](../../../backend/internal/handlers/killer_moves.go#L975-L1028).

## Data example

`Gu.csv` contains flat authored damage and multi-operation blocks. `GU_ALL_HEAVEN_RECORD_3` authors one ON_ACTION DAMAGE FLAT operation (`backend/data/Gu.csv:7`). Rank 9 `GU_ANCIENT_FRUIT_I` authors ALL_ENEMIES DAMAGE plus SELF HEAL in one block (`backend/data/Gu.csv:46`); this illustrates why retaining a separate target per operation matters. These examples are authored dictionary data; the combat conclusions above follow the local execution code.

## Implementation implication

The current killer-move path is a selection-and-flattening adapter over legacy simulator fields, not execution of compiled operations. A future effect redesign should decide supported operation/subtype combinations at option-generation time, preserve relevant conditions and source attribution (including component rank), and define whether targeting is per operation or per composed action before changing damage formulas.
