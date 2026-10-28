# Post-beta roadmap (PR-36)

This roadmap starts **after** the Gate D checklist is honestly satisfied. Dates
are deliberately omitted: throughput and content scope depend on measured load
and art authoring. Priorities are triggered by telemetry and release evidence,
not by wishful capacity claims.

## Guiding constraints

- Turn-based combat entrypoints and backend simulator were retired in #272. Rollback, if needed, is a code release rollback; no turn-based combat feature flag or parallel runtime is maintained.
- Server authority over movement, HP, effects, loot and ownership is non-
  negotiable; clients only send intent.
- Content is data-driven (`Gu.csv`, map JSON, dictionaries); new mechanics need
  an explicit schema/native handler, never a silent fallback.
- 100 players/zone, multi-worker routing and GvG are **unclaimed** until the SLO
  matrix in `docs/mmo/slo.md` is measured.

## Track A — Beta hardening (first)

Exit criteria: canary 5% → 25% → 100% completed, no duplicate economy/ownership,
on-call drill signed.

1. **Cross-process ownership**
   - Redis/DB lease registry with fencing tokens and stale-owner recovery;
   - gateway session directory/routing to the owning process;
   - worker restart/drain and transfer-storm tests.
2. **Durable kill/economy**
   - durable death nonce/outbox so a crash between death and grant does not lose
     or duplicate loot;
   - replay/crash integration tests against PostgreSQL;
   - corpse interaction/F loot UX and client toast.
3. **Observability**
   - Prometheus/OpenTelemetry metrics and dashboards for tick, queues, AOI
     bytes, reconnect, errors, loot/checkpoint;
   - structured redaction audit and alert runbooks.
4. **Safety**
   - 6–24 h soak with reconnect storms, network jitter/loss and DB/Redis failure
     injection;
   - full cross-account quest/party/PvP/loot/debug-endpoint audit.

## Track B — Mobile & input accessibility

Trigger: desktop beta stable; player demand or platform requirement.

- Virtual joystick + tap-to-move with the same server intent protocol.
- Skill wheel/radial or paged hotbar; large-touch target picker and aim assist
  that still sends a server-validated direction/target.
- Responsive HUD/business drawers, safe-area handling, reduced VFX mode.
- Touch device performance budget and battery/thermal measurement.
- Prerequisite: input abstraction seam and browser E2E for touch; no gameplay
  rule may depend on pointer type.

## Track C — GvG, party depth & large world

Trigger: cross-process ownership + measured 30-player zones stable.

- Sect/GvG objectives, scoring, spawn/ownership rules and anti-alt abuse.
- Cross-zone party/raid transfer, reconnect-in-combat and spectator/stream
  safety.
- Interest management beyond a single AOI radius (layers, crowds, visibility
  budgets) driven by measured AOI bytes.
- Multi-worker zone sharding and transfer load balancing; no split-brain leases.
- Large-scale event caps and queueing rather than unbounded zone growth.

## Track D — Content & Gu catalog

Trigger: Alpha retention/usage telemetry; content owner sign-off.

- NPC/dialog/quest/social/drop authoring for the four Alpha maps.
- Expand from 46 compiled profiles toward a curated 75–100 pool, prioritizing
  usage/rarity and `NEEDS_REAUTHOR` blocks with explicit new schema.
- Versioned native handlers for `UNIQUE`/high-concept mechanics with deterministic
  tests and replay fixtures.
- Per-Gu hidden passives and synergy combos; PvE/PvP separate balance modifiers.
- GM/debug replay tools, content lint in CI and asset existence validation.
- Coverage/issue dashboard; explicitly defer unsupported mechanics instead of
  faking behavior.

## Track E — Economy, social & live ops

Trigger: beta economy/loot data stable.

- Auction/marketplace reconciliation and anti-RMT invariants.
- Party/sect chat/moderation tooling, abuse reports and rate limits.
- Seasonal events/content rotation with versioned imports and rollback.
- Telemetry-driven economy sinks/faucets; no silent retroactive nerfs.

## Explicitly deferred

- Deleting remaining non-combat Pixi/legacy compatibility code only after its consumer audit; turn-based combat handlers are already retired under #272.
- Claiming 100 players/zone or multi-worker capacity without a measured run.
- Automatic turn→millisecond conversion for Gu effects.
- Cross-process GvG/large-world routing before lease fencing.
- Mobile storefront/native app packaging.

## Decision inputs

Each roadmap item should cite one or more of:

- `docs/mmo/slo.md` measurement and capacity decision;
- `docs/mmo/benchmark.md` synthetic regressions;
- `docs/mmo/security.md` soak/dupe evidence;
- `docs/mmo/release-checklist.md` canary/QA evidence;
- `docs/mmo/gu-coverage.csv` content gaps and usage telemetry.

If an item cannot state its trigger, prerequisite and rollback, it is not ready
to schedule.
