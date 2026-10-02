# Canonical non-player actor model

We represent every persistent non-player individual as an **Actor Instance**, resolve static identity from an **Actor Template**, and model map existence separately as a **World Presence**. Gameplay changes must pass through commands and validated state transitions so exactly one execution state owns an actor at a time; capability data selects what the actor may do, while `actor_kind` remains identity/lore and a source of default policy.

## Consequences

- An Actor Instance cannot simultaneously execute incompatible work, deployment, or combat states, and it cannot have more than one World Presence.
- Clients submit intent only; authoritative position, HP, lifecycle, and execution state remain server-owned.
- AI emits one intent and may not write unrelated domain tables directly; movement, combat, and production apply that intent through their owning transition boundary.
- `monster_templates` remains the beast/monster template source and aggregate `aperture_variant_humans` remains population data. Legacy pet/variant-NPC instance tables are migrated into the canonical actor table and retired; authored non-monster archetypes may use `NPC_TEMPLATE` rather than overloading monster data.
- Combat consumes an immutable **Combat Snapshot**; real-combat results are persisted through actor transitions, while virtual combat does not persist actor damage, hunger, Soul HP, or death.
