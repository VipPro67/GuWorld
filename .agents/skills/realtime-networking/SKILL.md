---
name: realtime-networking
description: Design and diagnose authoritative MMO networking, snapshots, prediction and lag handling.
---

# Realtime networking
1. Establish protocol fields, tick rate, sequence IDs, ownership, time sync and authoritative state source from code; do not invent configured values.
2. Distinguish input commands, server snapshots/events, reliable critical effects and cosmetic events.
3. Validate speed, cooldowns, range, line of sight, hit calculations, inventory and resource costs server-side.
4. Protect against duplicate/reordered packets, reconnect replay and stale snapshots; document idempotency.
5. Measure RTT, jitter, loss, bandwidth, tick duration and entity density before tuning prediction/interpolation.
6. Test 2 clients, disconnect/reconnect, high latency, region entry/exit and packet reordering.
7. Include observability, feature flags and rollback for protocol migrations.
## Never
Do not trust client-supplied damage, coordinates or cooldown expiry as final authority.
