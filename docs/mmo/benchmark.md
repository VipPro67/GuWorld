# QoS benchmark notes (PR-30)

This file records only **measured synthetic CPU results**. Browser FPS, RTT,
jitter/loss, AOI bytes/client and DB/Redis latency are not measured in this
environment; do not treat these numbers as Gate B/C/D passes. Gate D candidate
thresholds and the capacity decision matrix are in `docs/mmo/slo.md`; the
repeatable automated subset is `scripts/mmo-release-gate.sh`.

## Command

```bash
cd backend
go run ./cmd/mmo_bench -players 5  -mobs 50 -ticks 1000
go run ./cmd/mmo_bench -players 30 -mobs 50 -ticks 1000
go run ./cmd/mmo_bench -players 100 -mobs 50 -ticks 1000
```

The harness runs one in-memory zone, submits movement for every player each
tick and TARGET/DIRECTION casts every 10 ticks, then measures wall-clock `Step`
duration percentiles, final snapshot JSON bytes and allocator deltas.

## Measured on the development machine (2026-10-25)

| Players | Mobs | p50 | p95 | p99 | Final snapshot JSON | Total alloc / 1k ticks |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 5 | 50 | 46µs | 242µs | 317µs | 12 KB | 413 MB |
| 30 | 50 | 79µs | 359µs | 510µs | 24 KB | 614 MB |
| 100 | 50 | 216µs | 967µs | 1.46ms | 56 KB | 1.14 GB |

These are below the 50ms tick budget on this machine, but the allocator churn is
high (~1.1MB/tick at 100 players) because snapshots/events allocate per tick.
That is the next optimization target before claiming 100-player capacity. The
snapshot JSON shown is the whole zone, not per-client AOI bytes.

## Zone lifecycle measurement (2026-10-26)

A local probe loaded all 131 maps in `backend/data/mmo/maps` (34,270 monster
spawns, one instance per map), built the manager, started it, then performed one
cold claim/attach. "Eager" builds every world at startup; "lazy" validates maps
and builds only the claimed instance. Heap/RSS are sampled after `runtime.GC`.

| Mode | Manager build | Heap after build | Peak RSS | Running after start | Cold claim+attach |
| --- | ---: | ---: | ---: | ---: | ---: |
| Eager (old lifecycle) | 107 ms | 251.8 MB | 342.8 MB | 131 | 25 ms |
| Lazy (new lifecycle) | 26 ms | 11.3 MB | 85.8 MB | 0 | 52 ms |

Map parsing/validation itself took ~1.6 s and 11.3 MB in both modes; it still
runs for every configured map at startup. The lazy mode removes the per-world
monster/snapshot memory and 20 Hz runners while idle and pays one world build on
the first join. This is a development-machine measurement, not a production
capacity claim; re-run with the deployed map pack and real traffic before
setting memory alerts.

## Client debug HUD

Press **F3** in the MMO scene to show:
- tick/tick rate, entity/projectile count;
- self position and last prediction correction (world units);
- network messages/snapshots/bytes and age of the last message;
- render FPS.

The HUD polls at 4 Hz and is render-only; it never writes game state.

## Not measured / pending

- Browser FPS p95 frame time, draw calls, texture VRAM, object churn.
- 50/100/150/200ms RTT with jitter and packet loss.
- Per-client AOI snapshot bytes/sec and slow-client resync rate.
- Outbound queue depth, snapshot drops, reconnect rate, RTT estimate.
- PostgreSQL/Redis latency under checkpoint/loot load.
- Server tick p99 under 5/30/100 **real network clients** (current harness is
  synthetic in-process).
- Any automatic optimization: do not change budgets/radii without a measured
  hotspot.

## Optimization backlog suggested by the harness

1. Reuse snapshot/event slices and entity copies to cut per-tick allocations.
2. Avoid serializing the whole zone per telemetry sample; measure AOI bytes.
3. Add per-zone metrics (`zone_tick_duration_ms`, `ws_outbound_bytes`,
   `ws_queue_depth`, `ws_snapshot_drops`, `zone_entity_count`) to an admin/debug
   endpoint or Prometheus; `Manager.Stats()` already exposes tick/entity/player/
   subscriber counts and last step duration for a future endpoint.
4. Profile AI threat map allocation and status slices at high player counts.
5. Re-run browser/network load before any Gate C/D claim.
