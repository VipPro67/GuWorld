# Realtime World Protocol v1

This is the first shared wire contract for the Phaser client and Go world
runtime. It is separate from the existing `match.input` arena protocol and the
legacy `world.move_path` API.

## Envelope

Every message is a JSON object with `v: 1`, a `type`, and an object `payload`.
Client request messages may include a short `request_id` for correlation.
Messages larger than 16 KiB, unsupported versions, unknown message types,
unknown Go fields, malformed JSON, or non-finite numbers are rejected before
they reach a zone inbox.

The realtime runtime shares the `/api/ws` socket with the legacy hub, which
pushes channel traffic (`event`, `pong`, `command` responses, chat/trade/player
snapshots) on every connection. Only frames whose `type` starts with `world.`
belong to this protocol; clients must skip the rest instead of treating them as
a version mismatch.

```json
{"v":1,"type":"world.input","request_id":"c-17","payload":{"seq":17,"move":{"x":0.7071,"y":0.7071}}}
```

## Message types

| Type | Direction | Payload purpose |
| --- | --- | --- |
| `world.join` | client → server | Requested `zone_id`, `map_version`, and `content_version`; identity comes from the authenticated session. |
| `world.welcome` | server → client | Session/entity identity, tick rate, map/content versions, map collision hash, server tick, owner hotbar projection (`skills[]`) and `loadout_revision`, baseline and initial snapshot. |
| `world.input` | client → server | Monotonic `seq` plus normalized movement vector; zero vector stops movement. |
| `world.move_target` | client → server | Monotonic `seq` and a world-unit destination for bounded server pathing. |
| `world.cast` | client → server | Monotonic `seq`, hotbar slot 1–6, `mode` (`TARGET`/`DIRECTION`/`GROUND`/`SELF`), optional target and aim/point coordinates. |
| `world.attack` | client → server | Monotonic `seq` and `target_id` for a basic melee attack; J or the HUD button sends intent only. |
| `world.events` | server → client | Reliable gameplay events with event IDs and server tick (`DAMAGE_APPLIED`, `CAST_ACCEPTED`, `CAST_REJECTED`, `ENTITY_DIED`, ...). |
| `world.snapshot` | server → client | Baseline/delta state, tick, acknowledged sequence and entity revisions; each entity carries `kind` and `max_hp`. Monsters also carry `template_id`, `name`, `ai_state` and `aggro_target_id` for combat presentation. |
| `world.transfer` | server → client | Server-owned border handoff: `from_zone_id`, `to_zone_id`, `target_x/target_y` (map-local cells) and `legacy_x/legacy_y` (players.x/y world projection), plus carried `hp`. The client updates its durable position from the legacy pair and reconnects with the new `zone_id`. Client-initiated transfers are not accepted. |
| `world.reject` | server → client | Stable error code and optional retry guidance. |
| `world.resync` | both | Client requests or server initiates a full baseline after a gap. |

Coordinates use continuous world units. Input sequence, tick and revision values
are server-validated; the client cannot authoritatively set position, HP, hits,
damage or rewards. The authenticated gateway must bind character identity from
the session and ignore any identity fields supplied by a client.

Clients must wait for `world.welcome` before sending `world.input`; input isqueued separately from join/leave lifecycle commands. The server caps that queue
at 64 messages per connection and accepts at most 30 movement messages per
second. Accepted inputs are applied by the fixed-step simulation. `world.input`
carries a held movement vector; `world.move_target` shares the same monotonic
sequence and asks the zone to walk to a destination. The server clamps every
destination to 64 world units from the actor, converts it into the same
per-tick movement intent, and clears it on arrival, on a blocking collision, or
when the player sends non-zero WASD input. Client paths are never trusted and
no database call happens per movement step. The preview gateway streams
snapshots at 20 Hz for a circular AOI of radius 32 world units and includes the
owning character's `ack_seq`. Each snapshot is either a full baseline (first
update and every 40 ticks) or a delta relative to that baseline; `removed` lists
entities that left the AOI. Deltas repeat all baseline-relative changes and
removals, so a newer coalesced delta can replace an older one. Per-session feeds
and the WS snapshot slot each hold at most the newest pending state. Reliable
messages use a separate outbound queue and are written before snapshots. The
client now predicts its own movement, reconciles softly against `ack_seq`, and
interpolates remote actors by a two-tick render delay. Reliable event replay,
automatic resync requests and byte-budget metrics remain pending; admission
stays allowlisted.

## Combat preview

Basic attack uses the same authoritative zone tick as skills. The server checks
that the player is alive and not stunned, can harm the selected target, and is
within 2.2 world units; a successful hit deals 12 preview damage and starts a
10-tick cooldown. These are temporary preview balance values, not values read
from Gu data. `BASIC_ATTACK` and `HIT_CONFIRMED` events drive the client
animation; damage remains server-authored. Press **J** or click **ĐÁNH** in the
combat HUD. If no target is selected, the client selects the nearest visible
monster before sending the intent.

`world.cast` currently uses six server-side preview skills (slots 1–6) that map
to `TARGET`, `DIRECTION`, `GROUND` and `SELF`. They are placeholders until the
Gu compiler/loadout adapter lands; the client only mirrors their mode/range for
UI and never sends damage. The zone validates ownership, slot, mode, mana,
GCD/cooldown, range, target faction and ground walkability, then resolves windup,
projectile/AoE hit and shield/HP changes on the tick. Incoming damage interrupts
a player's windup.

Damage is always server-authored. `world.events` carries the resulting
`CAST_ACCEPTED`, `CAST_REJECTED`, `CAST_INTERRUPTED`, `PROJECTILE_SPAWNED`,
`HIT_CONFIRMED`, `DAMAGE_APPLIED`, `HEAL_APPLIED`, `SHIELD_APPLIED`,
`STATUS_APPLIED`, `STATUS_EXPIRED`, `HIDDEN_PASSIVE_TRIGGERED`, `MOVE_TARGET_REJECTED` and
`ENTITY_DIED` events with a monotonic event ID. For `DAMAGE_APPLIED`, `amount`
is actual HP lost after shield absorption; zero means the shield blocked the
full hit. Statuses are tick-scheduled (DoT/HoT, stacks, refresh/ICD); a zero move multiplier blocks movement and
casting. Preview limitation: event batches ride alongside snapshots, so a
slow/coalesced snapshot feed can drop an event; reconnect replay/`last_event_id`
is still future work (PR-26/PR-30).

The hotbar is built at join from the character's equipped, ACTIVE Gu rows
(`player_gu.equipped` + `players.equipped_gus` ordering) matched against the
compiled realtime profile catalog. Only slots with a compiled profile are sent;
unsupported/missing Gu are skipped, and a character with no compiled owned Gu
falls back to the server preview hotbar (no partial skill set). `loadout_revision`
is a deterministic hash of the equipped instance order. Live loadout refresh
while inside a zone is still future work; rejoin rebuilds it from the database.

The owner entity in each snapshot also carries coarse combat state for the HUD:`mana`, `max_mana`, `cooldowns` (absolute tick deadlines, index 1–6),
`cast_skill_id`, `cast_end_tick` and `cast_windup_ticks`. Projectiles are sent as
`projectiles[]` with id/source/skill/position/radius; the client renders them
but never resolves hits. React HUD state is derived from these snapshots and is
throttled to 10 Hz; per-frame positions stay in Phaser.

The initial validation limits

- Envelope maximum: 16 KiB; request ID maximum: 64 bytes.
- Entity ID maximum: 80 characters; world coordinate absolute limit: 1,000,000.
- Movement vector must be finite and have magnitude at most 1.01; server normalizes
  the small tolerance used for diagonal keyboard input.
- Cast slot must be 1–6; `seq` must be positive and monotonic per controller.
- Version mismatch, malformed payload, replayed sequence, stale map/content
  version, and invalid target are rejected or sent through resync policy.

The Go strict client-command decoder and TypeScript envelope decoder are in
place. The authenticated WebSocket routes join/leave/input/move_target and
streams AOI baseline/delta snapshots. The Phaser host connects with the
configured zone/map/content tuple, renders the owner baseline, predicts WASD and
click movement locally, applies soft/hard reconciliation from `ack_seq`, and
interpolates remote entities. Other message families, shared golden fixtures,
reliable gameplay events and automatic resync negotiation remain follow-up
work.

The isolated server simulation can be exercised with
`cd backend && go run ./cmd/mmo_headless -ticks 10000`. It spawns two players and
ten monsters, runs fixed 20 Hz steps without sleeping, and prints a repeatable
snapshot checksum. Movement intents expire after five ticks without an update.
