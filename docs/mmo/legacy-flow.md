# Legacy and realtime world flow inventory

Snapshot: 2026-10-25. This records the checked-out code paths for migration
review; it is not evidence that production cutover or browser flows passed.

## Default legacy flow

1. `frontend/src/app/page.tsx` selects `world-map` by default and mounts the
   existing `frontend/src/components/WorldPage.tsx` business/world view.
2. `WorldPage` delegates map rendering and legacy movement to
   `frontend/src/components/WorldCanvas.tsx` and uses the existing Zustand
   world/game stores for player, inventory, Gu and social state.
3. The frontend's shared authenticated API/WS helpers route legacy movement,
   exploration and turn-combat actions to the existing Gin handlers.
4. Backend handlers persist canonical player/inventory/progression data through
   existing DB/service patterns. Legacy monster/exploration/combat workers
   remain present for rollback and non-MMO gameplay.

## Opt-in Phaser path

1. `frontend/src/lib/featureFlags.ts` gates the `mmo-world` view. The default is
   off; when off, the view resolves back to `world-map`.
2. `frontend/src/app/page.tsx` dynamically imports
   `frontend/src/components/game/PhaserWorldHost.tsx` with SSR disabled.
3. `PhaserWorldHost` dynamically creates the Phaser game and `WorldRuntime`;
   the latter uses `frontend/src/game/network/WorldSocket.ts` and the v1
   protocol. Preview rendering can start without admission configuration, but
   a live join requires configured zone, map version and content version.
4. The backend router authenticates WS messages, checks configured admission
   and delegates joined world commands to the realtime gateway/manager. Zone
   simulation owns movement/combat state while admitted; snapshots are sent to
   the client for prediction/reconciliation and React HUD projection.
5. Business panels continue using existing React/Zustand flows. They are not
   yet all audited or fenced against mutations while the same character is
   realtime-owned; this is a production cutover blocker.

## Historical rollback path

This pre-launch runbook described environment-gated rollout. Current production
behavior keeps MMO enabled for signed-in players; use graceful drain and deploy
the previous known-good application version for rollback. Preserve checkpoints
and loot-grant rows.

The sequence above is the documented rollback design. A staging canary and
timed rollback rehearsal have not been performed; see `cutover.md` and
`release-checklist.md`.

## Inventory still required before cutover

- Enumerate every legacy mutation for inventory, quest, cultivation, trade,
  combat, movement and monster rewards; prove each rejects or safely routes a
  realtime-owned character.
- Validate duplicate tabs, in-flight legacy transactions at join, disconnect,
  reconnect, transfer and process restart against durable ownership fencing.
- Exercise both flag-OFF legacy and flag-ON Phaser paths in a browser, including
  canvas teardown, UI focus, combat, loot and reconnect.
- Attach staging database/media restore evidence and rollback timing.
