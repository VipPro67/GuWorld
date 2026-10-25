# Realtime map artifact (v1)

The realtime preview uses one JSON map artifact for both the Go zone and the
Phaser client. The server copy is canonical:

```text
backend/data/mmo/maps/<map_id>.json          # source of truth (server)
frontend/public/game/maps/<map_id>.json      # exact copy served to the client
```

`backend/internal/realtime/nav/map_test.go` fails when any two copies differ (it
walks the whole map directory), so collision and visuals cannot drift silently.
After editing a map:

```bash
cp backend/data/mmo/maps/*.json frontend/public/game/maps/
cd backend && go test ./internal/realtime/nav/...
```

## Format

The file is a small Tiled-compatible subset:

```json
{
  "type": "map",
  "orientation": "orthogonal",
  "renderorder": "right-down",
  "width": 64,
  "height": 48,
  "tilewidth": 32,
  "tileheight": 32,
  "map_id": "realtime_preview",
  "map_version": "preview-1",
  "content_version": "content-1",
  "layers": [
    { "type": "tilelayer", "name": "ground", "width": 64, "height": 48, "data": [] },
    { "type": "tilelayer", "name": "collision", "width": 64, "height": 48, "data": [] },
    { "type": "objectgroup", "name": "spawns", "objects": [] }
  ]
}
```

- `orientation` must be `orthogonal`.
- **One cell equals one world unit.** `tilewidth`/`tileheight` are render
  metadata only; the client applies `PIXELS_PER_UNIT = 24`.
- `collision.data` can contain either `width * height` values of `0` (walkable)
  or `1` (solid) row-major, OR an empty array `[]` (sparse/procedural mode). In
  sparse mode, outer boundary borders are automatically solid and all interior
  cells are walkable. This allows massive 2000×2000 child region maps (matching
  Region.csv 1:1) to serialize in < 5 KB instead of 23 MB JSON files.
- The server compiles sparse collision to a `BorderCollision` that stores only
  width/height, and authored layers to a packed one-bit-per-tile
  `PackedCollision`. Both preserve the legacy `Blocked` result and the SHA-256
  tile stream exposed as `world.welcome.map_hash`; the client JSON/hash contract
  is unchanged. One immutable map artifact is loaded and validated per map at
  startup, then shared by every instance of that map.
- `ground.data` is optional (empty array `[]` defaults to 0). `1` is the base tile and `2` renders as a path tint.
- Spawn objects live in the `spawns` object group. `type` is required
  (`player`, `monster`, `npc`, `portal`, `boss`, ...); `x`/`y` are world units
  and their full collision radius must fit inside walkable space. Monster spawns must carry
  a `monster_id` property that exists in the server monster registry
  (`monster_templates`). The server resolves HP/attack from the template and
  applies placeholder movement/AI archetype stats based on its type
  (`Normal`/`Elite`/`Boss`).
- The converted `world_map` reserves a 2-unit margin from the outer wall for
  monster spawn centers; the map test checks every spawn using the largest
  current monster radius (0.8), since a walkable center alone is insufficient.
- The sample `realtime_preview` map is 64×48 with border walls, obstacle blocks,
  player/monster/NPC spawns and a vertical path.

## Alpha content pack

PR-35 adds four linked 64×48 maps (`map_version: alpha-1`), each with a
server/client byte-identical copy:

| Map | Ruleset | Content |
| --- | --- | --- |
| `misty_forest` | `pve` | Normal/Elite hunt map; portals to crypt + safe hub |
| `bone_crypt` | `pve` | Bone General/centipede plus `boss_wolf_king`; portals to forest + ravine |
| `crimson_ravine` | `pvp` | Open PvP with lava python and `monster_desert_ghost_boss`; portals to crypt + hub |
| `emerald_grotto` | `safe` | No monsters; social/trade hub; portals to forest + ravine |

Portal metadata points both ways, and the nav test verifies every portal target
exists, is inside the target map and lands on a walkable tile. Monster IDs are
resolved through the normal `monster_templates` registry at startup; an unknown
ID logs a warning and skips that spawn rather than failing the whole zone.

To run the Alpha chain, set one uniform version and the map list:

```bash
MMO_ZONE_IDS=misty_forest,bone_crypt,crimson_ravine,emerald_grotto
MMO_MAP_VERSION=alpha-1
```

The per-zone `crimson_ravine` config is what marks the zone PvP; the client still
loads `/game/maps/<zone>.json` from `NEXT_PUBLIC_MMO_ZONE_ID`.

## Version and admission

- `map_version` should equal the backend `MMO_MAP_VERSION` and the client
  `NEXT_PUBLIC_MMO_MAP_VERSION`. Startup logs a warning on mismatch; the client
  shows `LỆCH VERSION` in the HUD when `world.welcome.map_version` differs from
  the loaded file.
- `MMO_ZONE_IDS` values are `region_world_maps.world_map_id` values. To use the
  sample map in staging, map a test region to `realtime_preview` and configure
  `MMO_ZONE_IDS=realtime_preview`.
- `MMO_MAP_DIR` (default `data/mmo/maps`) selects the server map directory.
  A missing or invalid file logs an error and prevents the realtime manager
  from starting; admission stays disabled.
- `NEXT_PUBLIC_MMO_MAP_URL` overrides the client URL. The default is
  `/game/maps/<NEXT_PUBLIC_MMO_ZONE_ID>.json`, falling back to
  `/game/maps/realtime_preview.json`.

## Safe spawn

Admission validates the database position against the loaded map. A position
inside a wall or outside the map is replaced by an authored spawn (or the first
free tile). This keeps legacy coordinates from creating a stuck entity when a
region is first mapped to a realtime zone. The coordinates are still persisted
in the legacy columns for rollback; the realtime world owns the live position.

## Hash

Every loaded map exposes a SHA-256 over `map_id`, `map_version`, dimensions and
the collision bitmap (`nav.Map.Hash`). The hash is logged at startup and sent
as `world.welcome.map_hash`; the client recomputes it from the downloaded JSON
and shows `LỆCH HASH` in the HUD when it differs. The golden tests pin both
implementations to the same digest and also require the two JSON copies to be
byte-identical. The compact collision types still hash the same byte stream as
the old dense grid, so changing them must not change any existing map hash.

## Out of scope in v1

- polygons/slopes and Tiled object collision shapes (only a 0/1 grid);
- client or server A* navigation (click-to-move is a bounded straight intent);
- streaming/chunked maps and occlusion layers;
- conversion tooling from `Region.csv`/`MapCell.csv`/`MonsterSpawn.csv` (future
  PR-12 work; the current map is authored from the Tiled subset directly).
