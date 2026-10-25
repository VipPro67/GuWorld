# Security, fuzz & soak (PR-34)

## Automated tests added

- `FuzzDecodeClientMessage`: arbitrary bytes must never panic; accepted
  envelopes must be v1 with a payload.
- Strict decoder tests:
  - client-supplied `player_id` in `world.join` is rejected (identity comes from
    auth);
  - oversized envelope rejected;
  - movement magnitude > 1.01 rejected.
- `FuzzSubmitInputNeverPanics` / `FuzzMoveTargetNeverPanics`: arbitrary
  entity IDs, sequence numbers, NaN/Inf/huge coordinates must not panic.
- `FuzzParseMapNeverPanics`: arbitrary map JSON cannot crash loading.
- Controller reuse across characters is rejected
  (`realtime_controller_mismatch`); spoofed identity/damage is never trusted.

Run the seed corpus in normal CI with `go test ./...`. Manual fuzzing:

```bash
cd backend
go test ./internal/realtime/protocol -run=^$ -fuzz=FuzzDecodeClientMessage -fuzztime=60s
go test ./internal/realtime/sim -run=^$ -fuzz=FuzzSubmitInputNeverPanics -fuzztime=60s
go test ./internal/realtime/nav -run=^$ -fuzz=FuzzParseMapNeverPanics -fuzztime=60s
```

## Soak / reconnect harness

```bash
cd backend
go run ./cmd/mmo_soak -duration 60s -players 10 -mobs 20 -reconnect-every 5s
```

It runs a wall-clock 20 Hz manager zone, moves/casts for every player,
disconnects/rejoins one player periodically, tracks ownership count, max
entities/players, input/cast errors, event counts and heap delta. It exits
non-zero if reconnect storms create duplicate/lost ownership.

For a longer 6–24h soak use the same command in staging with a realistic
player/mob mix and metrics collection.

## Dupe / economy checks

Already covered:
- loot grant idempotency key + transaction (`realtime_loot_grants`);
- inventory add and Gu insert in the same transaction;
- durable checkpoint version gates;
- stale lease/controller rejection;
- cast/damage never accepted from client values.

Missing:
- PostgreSQL integration test that replays the same grant 100× and verifies one
  row/one item (needs a DB fixture/container).
- Crash injection between grant claim and commit (transaction rollback is
  covered by design, not by a running DB test).
- Cross-account loot/party/PvP authorization audit beyond controller/ownership.

## Threat model / invariants

- Auth identity is bound by middleware; payload IDs are ignored/rejected.
- Movement/cast/range/mana/cooldown/hit/damage are server-derived.
- One process-local controller lease per character; stale tokens rejected.
- Loot rewards require a server death event and idempotency key.
- AOI snapshots only include nearby entities; hidden passive internals are not
  serialized.
- Admin/debug endpoints require existing admin auth; F3 is client-local.

## Known residual risks

- Process-local leases only; cross-process split-brain is not fenced (PR-32).
- A crash before the death worker handles an event loses that loot intent; a
  committed grant is replay-safe. Durable death nonce/outbox pending.
- Bounded death/input queues drop under extreme load without counters yet.
- No automated log-redaction audit for tokens/passwords (ops checklist only).
- No long-running production soak/load data; all capacity claims remain
  unmeasured.
- Release gate remains **HOLD** until `docs/mmo/release-checklist.md` has
  browser/network load, DB crash/dupe replay, staging canary and on-call
  evidence; candidate thresholds are in `docs/mmo/slo.md`.
