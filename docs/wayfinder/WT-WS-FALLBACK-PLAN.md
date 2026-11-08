# WebSocket Fallback for WebTransport

Status: design agreed; implementation not started (2026-11-05)

## Goal

Keep WebTransport (WT) as the preferred realtime transport and automatically
use WebSocket (WS) when the browser has no WT support or WT cannot establish a
transport connection. Older devices and networks that block UDP should still
receive the full world and UI realtime feature set.

This reopens the WT-only choice recorded in issues #303, #309 and #312. The
current runtime remains WT-only until this plan is implemented.

## Agreed behavior

- Attempt WT first. Fall back after one bounded connection attempt if WT is
  unsupported or fails at the transport layer.
- Do not fall back for rejected world admission, protocol/map/content version
  mismatch, or other application-level rejects. Browser WebTransport handshake
  failures are opaque and may include authentication/origin rejection; the WS
  endpoint rechecks both before upgrade, so such a fallback cannot bypass them.
- WS supports both world gameplay and UI events/commands on one connection per
  tab. Preserve the existing server-authoritative gameplay rules.
- Enable fallback by default. Keep an operational kill switch for emergency
  rollback, defaulting to enabled.
- WS has no QUIC datagrams. Carry input and snapshots as WS frames; coalesce
  input and replace stale queued snapshots to bound latency and memory.

## Current code facts

- The browser's shared `RealtimeClient` only creates WebTransport sessions.
  `WorldTransport` sends high-frequency input and reads snapshot deltas through
  QUIC datagrams; `UIRealtimeManager` uses a separate bidirectional WT stream.
- The backend WT handler owns session multiplexing, world admission and snapshot
  production in `backend/internal/realtime/wttransport`. UI commands and events
  use its `UIOptions` and `internal/uihub`.
- `gateway.Admission` owns player session join, resume/takeover, input and
  disconnect semantics. Both transports must use the same instance so a
  transport switch does not fork gameplay ownership.
- Authenticated WT routing is outside Gin because WebTransport needs the raw
  HTTP/3 response writer. WS can use the normal TLS/TCP HTTP handler path.
- `/api/ws` and `internal/ws` were removed. The Gorilla WebSocket dependency is
  still present for the standalone `cmd/wtspike` probe, not the game runtime.
- `CONTEXT.md` says clients submit intent only and the server owns authoritative
  position, HP, lifecycle and execution state. This plan preserves that rule.

## Target behavior and protocol constraints

Keep protocol v2 world payloads and UI NDJSON payloads unchanged. Add a WS
transport envelope that distinguishes world binary messages from UI text
messages. WT stream identifiers remain a transport detail; gameplay and UI
message semantics must not branch on which transport carried them.

Use one WS connection per tab, authenticated with the same player identity,
active-tab claim, origin policy and tab ID used by WT. A WS world join must pass
through the same `gateway.Admission` instance. If a WT session may have reached
admission before the client observed its failure, reconnect with the same tab
ID so the existing resume/takeover behavior can reclaim the session safely.

Since WS is one ordered byte stream, prioritize bounded queues and freshness:

- Serialize all writes through one writer path per WS connection.
- Coalesce pending movement input to the newest intent and cap its send rate to
  the simulation cadence.
- Keep only the newest unsent world snapshot; do not queue stale snapshots.
- Preserve reliable ordering for join/welcome, combat events, transfer,
  rejects, UI RPC responses and subscription events. If these queues fill,
  close with an observable overload reason and let normal reconnect/resume run.
- Retain the existing tick rate. Do not increase server simulation frequency
  to compensate for TCP behavior.

## Implementation phases

### 0. Reconcile the WT-only decision

- Update the WT rollout/unsupported-device specs so they no longer promise that
  WS will never coexist with WT.
- Record the tradeoff: dual transport increases maintenance and test surface,
  while avoiding UDP-dependent device exclusion; WS has TCP head-of-line delay.
- Update current-state docs only when implementation actually lands.

### 1. Define and test transport mapping

- Specify WS message envelope, size limits, ping/close behavior and ordering.
- Reuse v2 world payload encoders/decoders and UI message schemas. Do not revive
  historical protocol v1.
- Add shared golden vectors proving WT and WS carry identical world payloads.
- Decide explicit maximum message size and per-connection queue budgets before
  coding the writer.

### 2. Add backend WS transport

- Add an authenticated TLS/TCP WS route, using the existing origin allowlist,
  active-tab claim and token policy. Redact tokens from logs.
- Add a transport-specific session runner that maps WS frames to the current
  world and UI handlers and shares the existing admission, hub, command,
  initial-event and activity-touch dependencies.
- Do not copy combat, movement, admission, snapshot projection or UI command
  rules into a second implementation.
- Register live WS sessions for graceful drain and close them on shutdown.
- Enforce frame size, read deadlines, ping/pong, a single writer, bounded
  queues and cleanup of player leases/subscriptions on disconnect.
- Add an emergency backend setting for disabling WS fallback; default it to
  enabled in deployed environments.

### 3. Add browser WS adapter and bounded fallback

- Extend the shared per-tab realtime client to select WT first and acquire WS
  only for unsupported WT or bounded transport connection failures.
- Keep application-level WT rejects terminal; do not retry them through WS.
- Make the failed WT attempt release streams/session references before WS
  acquisition. Preserve the same tab ID and auth identity across the switch.
- Adapt `WorldTransport` so reliable WS frames feed the same protocol-v2
  decoder, projection, buffer and event flow. Replace datagram send/receive
  with bounded WS frame queues when WS is active.
- Adapt `UIRealtimeManager` to use the same WS session while retaining
  subscriptions, request IDs, timeouts and existing HTTP fallback behavior.
- Keep public world and UI caller behavior transport-independent.

### 4. Validate parity, security and resource limits

- Unit test transport selection for: no WT API, WT constructor error, WT ready
  timeout, close before welcome, auth reject, join reject, version mismatch,
  server draining, and successful WT with no WS attempt.
- Integration test both transports against the same backend runtime: join,
  baseline, movement, attack/cast, snapshot updates, combat events, transfer,
  UI subscribe/unsubscribe, UI command/response, chat, disconnect and resume.
- Verify slow-client behavior: snapshot coalescing, input bounds, reliable-event
  queue overflow, bounded memory and clean disconnect.
- Verify auth/origin rejection, oversized frames, duplicate-tab rotation,
  same-tab WT-to-WS takeover, token redaction and graceful server drain.
- Run backend tests and race checks for realtime packages; run frontend tests,
  typecheck, lint and build. Add browser smoke coverage for a WT-capable browser
  and an environment where UDP/443 is blocked.
- Compare WT and WS latency, bytes per player, queue depth, reconnect rate and
  server CPU at 5/30/100-player workloads. Confirm TCP snapshots do not starve
  input or reliable combat/UI events.

### 5. Enable by default and observe

- Ship fallback enabled by default; no player-facing transport setting.
- Keep a server-side emergency disable switch and document its effect: clients
  without usable WT return to the existing HTTP-only behavior.
- Add per-transport connection/fallback counters and fallback reason labels;
  never label metrics with player IDs or auth tokens.
- Track WS/WT join success, fallback rate, p95 join time, p95 input-to-snapshot
  latency, queue overflow, reconnect/resume, bytes and CPU. Alert on sustained
  WS overload or a rise in failed joins.
- Update `docs/architecture/ARCHITECTURE.md`, `docs/guides/API.md`, deployment docs and the Wayfinder
  implementation progress note when code is released.

## Completion criteria

- A browser without WT and a browser whose UDP path is blocked can enter and
  play in the realtime world and use UI realtime features over WS.
- WT-capable clients continue using WT by default and never open WS unless WT
  transport establishment fails.
- Authentication and admission rejects do not trigger fallback.
- Gameplay state has one owner in `gateway.Admission`; switching transports
  with the same tab does not duplicate a player entity or lose durable state.
- WS queues remain bounded under slow-reader load, with fresh snapshots and
  reliable event ordering preserved within configured limits.
- Fallback is enabled by default, observable, and can be disabled operationally.
