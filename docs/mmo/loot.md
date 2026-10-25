# Realtime loot grants (PR-25)

Kill rewards are **server-authored and exactly-once** at the database level. The
zone tick never calls the database; the world manager extracts `ENTITY_DIED`
events into a bounded death queue and a worker calls the loot service.

## Flow

1. `sim.applyDamageLocked` attributes the killing source and emits
   `ENTITY_DIED` with `SourceID` (killer), `TargetID`, and monster `TemplateID`.
2. `Manager.broadcastSnapshots` enqueues a `DeathEvent` (queue 256, non-blocking).
3. `loot.Service.GrantKill`:
   - validates zone/death/recipient/template/version;
   - builds `idempotency_key = zone:death:recipient:rewardVersion`;
   - `INSERT ... ON CONFLICT DO NOTHING` into `realtime_loot_grants`;
   - if the key already existed → return `AlreadyGranted`, no mutation;
   - resolves `monster_templates.loot_table_id` → `loot_tables.entries`;
   - rolls item/Gu rewards on a server RNG;
   - inserts items through `InventoryService.AddItemsBatch` and Gu through
     `player_gu` in the **same transaction**;
   - writes the reward payload to the grant row and commits.
4. A crash/failure rolls back both the idempotency claim and rewards, so retry is
   safe. A committed grant can be replayed without duplicating anything.

`rewardVersion` is `1`; bump it only when a conscious economy migration requires
re-rolling existing death keys.

## Table

`realtime_loot_grants` (see `backend/internal/db/schema_realtime.go`):

- `idempotency_key` primary key;
- `zone_id`, `death_id`, `recipient_id` (FK players), `template_id`,
  `reward_version`, `payload`, `created_at`;
- indexes by `(zone_id, death_id)` and `(recipient_id, created_at)`.

## Current limitations

- Rewards are **auto-granted** to the killer on death. Corpse/ground-drop
  entities, TTL, F interaction, distance/LOS pickup checks and party loot rules
  are not implemented yet.
- No client loot toast/snapshot yet; open the bag to see granted items/Gu.
- `death_id` is derived from the process-local zone event ID
  (`eventID:targetID`). It is idempotent within a running zone/redeploy replay but
  is not yet a durable death nonce across a full process crash (PR-26).
- Gu drops use `uuid.NewString` DB instances, skip unknown/invalid templates and
  skip a second immortal-rank (`rank >= 6`) instance of the same Gu.
- Item/Gu roll uses the server RNG; no audit replay seed is stored yet.
- Death worker drops the DeathEvent when the bounded queue is full; telemetry is
  future work. Zone ticks are never blocked by DB latency.

## Tests

- `GrantKey` validation/idempotency-key uniqueness.
- Loot roll: chance, missing item/Gu catalog entries, quantity bounds.
- Manager death queue: a player-cast kill emits exactly one `DeathEvent` with the
  correct source/template.
- DB transaction/idempotency integration requires PostgreSQL and is not covered
  by the unit suite yet.
