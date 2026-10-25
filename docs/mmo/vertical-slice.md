# Vertical-slice gate (PR-29)

This is the current Gate B checklist for the opt-in realtime MMO slice. It
separates **automated headless evidence** from **manual browser evidence** and
does not claim a gate that was not measured.

## One-command headless slice

No PostgreSQL/Redis required:

```bash
cd backend
go run ./cmd/mmo_slice -players 5 -mobs 10 -ticks 400
```

The harness:
- starts one manager-owned zone with 5 player entities and 10 AI monsters;
- joins with the server preview loadout;
- moves every 4 ticks, casts TARGET/DIRECTION every 10 ticks;
- counts cast/hit/damage/death events;
- prints elapsed time, per-tick average, and a final snapshot checksum;
- exits non-zero if no monster dies or no player survives.

Loot/checkpoint are reported `NOT MEASURED` because they require PostgreSQL.

## Gate B checklist

| Gate B item | Status | Evidence / gap |
| --- | --- | --- |
| One shared Tiled-subset map with collision | **PASS** | `docs/mmo/map-format.md`, Go/TS parity test |
| 4 linked alpha maps (PvE/PvP/safe) | **PASS** | `misty_forest`/`bone_crypt`/`crimson_ravine`/`emerald_grotto`, portal-link test |
| 2–5 players online same zone | **PARTIAL** | 5 players in headless manager; browser 2-client run not automated |
| WASD + click-to-move | **PARTIAL** | Server sim + client A* planner tests; browser E2E not automated |
| Camera 2.5D / depth / resize | **PARTIAL** | CameraController + depth test by inspection; no browser perf measurement |
| 3 monster types with AI | **PASS** | alpha maps include Normal/Elite/Boss templates; AI FSM/leash/threat tests |
| Server hit/HP authority | **PASS** | cast/projectile/AoE/status/interrupt tests |
| Six MVP Gu + 40 Alpha profiles | **PARTIAL** | 46 compiled profiles; preview runtime bar for MVP, catalog-only Alpha awaits owned-loadout status/passive execution |
| Hidden passive | **PARTIAL** | preview `PREVIEW_LAST_STAND`; no authored/per-Gu hidden passive yet |
| Login with existing account + owned loadout | **PARTIAL** | join-time `players.equipped_gus` adapter; no browser login run |
| Leave/reconnect/checkpoint | **PARTIAL** | fake-store gateway tests + 10s checkpoint; no DB/browser reconnect test |
| Loot on death | **PARTIAL** | idempotent auto-grant transaction; no corpse/F/client toast |
| VFX/telegraph/feedback | **PARTIAL** | procedural frames/VFX pool/telegraph; no browser FPS/VRAM profile |
| Legacy world unaffected when flag OFF | **PASS** | default flags OFF, gated route, legacy tests pass |
| No `world.move_path`/turn encounter for MMO characters | **PASS** | lease guards + MMO client sends v1 world commands only |
| Five-player/30-player load | **NOT MEASURED** | only headless event loop; no browser/network load test |
| Economy exactly-once under retry/crash | **PARTIAL** | DB transaction + unique key tests logic; DB integration/crash replay not measured |

## Manual browser checklist

1. Set backend + frontend flags and an allowlisted account; import the map via
   Admin if needed.
2. Login with an existing account that owns at least one compiled Gu.
3. Open **MMO Beta**, confirm `world.welcome`, map/pulse and 2+ players.
4. Move with WASD and click (path marker/repath); press WASD to cancel a path.
5. Select a monster, cast slots 1–6, verify cast bar/cooldown/HP/shield/status.
6. Kill a monster, open **Túi**, confirm the granted item/Gu appears once.
7. Reload the page; confirm checkpoint resume near the last position, no
   duplicate entity/Gu.
8. Open business panels during combat; confirm hotkeys block but server combat
   continues. Close and confirm controls resume.
9. Disable the frontend/backend flags and confirm the legacy world remains
   playable; no MMO route/bundle loads.

## Staging activation

```bash
# backend (staging only)
REALTIME_WORLD_ENABLED=true \
PHASER_WORLD_CLIENT_ENABLED=true \
MMO_ZONE_IDS=realtime_preview \
MMO_ALLOWLIST=<player-uuid> \
MMO_MAP_VERSION=preview-1 \
MMO_CONTENT_VERSION=content-1 \
MMO_MAP_DIR=data/mmo/maps \
MMO_GU_PROFILES=data/mmo/gu/transpiled_profiles.json \
go run ./cmd/server

# frontend
NEXT_PUBLIC_REALTIME_WORLD_ENABLED=true \
NEXT_PUBLIC_PHASER_WORLD_CLIENT_ENABLED=true \
NEXT_PUBLIC_MMO_ZONE_ID=realtime_preview \
NEXT_PUBLIC_MMO_MAP_VERSION=preview-1 \
NEXT_PUBLIC_MMO_CONTENT_VERSION=content-1 \
npm run dev
```

## Known blockers before calling Gate B complete

- No automated browser E2E (Playwright/Puppeteer not installed) and no measured
  frame-time/VRAM report.
- Six MVP Gu plus 40 Alpha catalog profiles are compiled, but the live runtime
  bar is still the MVP preview set; per-owned-Gu status/passive execution needs
  the full adapter and Alpha balance pass.
- Hidden passive is a preview built-in, not authored content.
- Loot corpse/F interaction and client loot toast are absent.
- Durable death nonce/outbox and reconnect-under-combat tests are incomplete.
- No five-browser/30-player benchmark; do not claim Gate C/D.
