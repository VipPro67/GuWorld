# Location cutover audit (P0, in progress)

The running MMO still loads its initial region/coordinates from `players` and
checkpoints to `player_world_state_v2`. `player_world_location` is preparation
schema only. Do not use it for admission until the controlled backfill and
fenced writer are complete.

| Purpose | Current owner/call sites | Cutover action |
| --- | --- | --- |
| Join/spawn | `handlers/realtime_join_state.go`, `realtime/gateway/admission.go` read `players.current_region/map_instance_id/x/y` and v2 | Load destination row, validate map and instance, then let zone owner choose live spawn. |
| Checkpoint/recovery | `realtime/persistence/state.go`, periodic worker in `cmd/server/main.go` write v2 | Fenced revision write to destination row; drain/leave must confirm commit. |
| Legacy movement and group follow | `exploration_move.go`, `exploration_skills.go`, `teleport_handlers.go`, `party.go`, `spatial_realms.go` write `players` coordinates | Route through zone transfer/command; while pending, block MMO-owned actor and skip MMO-owned party targets. |
| Admin/worker movement | `admin_player_update.go`, spatial realm expiry worker, and other admin update handlers | Reject edits to MMO-owned targets until owner command exists; audit all admin and worker writers. |
| Distance and rewards | `merchant.go`, `sect_warfare.go`, `scouting.go`, `ground_drops.go`, `spatial_realms.go` read legacy coordinates | Require live zone and instance snapshot; temporarily block affected actor routes. |
| Online/nearby/chat | `world_index_sync.go`, `world_index.go`, `exploration_views.go`, `chat_events.go`, `chat.go` | Replace player-row projection with zone-owner online/nearby and region/instance scope. |
| Profile and other projections | `models/player.go`, `services/player_service.go`, profile/player events handlers | Read location through a dedicated small module; stop returning stale player-row coordinates. |
| Client display and join | `PhaserWorldHost.tsx`, `MMOBusinessPanels.tsx`, `app/page.tsx`, `GlobalChatOverlay.tsx`, merchant/sect/spatial panels; legacy `WorldCanvas.tsx` | Use server-selected zone, region, instance and live coordinates; convert atlas units before showing marker. |

The P0 guards added for party invitations and acceptance (not rejection), merchant, scouting,
spatial realm mutations and sect defense/attack apply only to the process-local
ownership checker. Party follow also skips MMO-owned targets; the admin player
editor rejects coordinate fields for an MMO-owned target. These guards reduce
known same-process conflicts but do not close the P0 gate: in-flight requests,
other workers/admin handlers and cross-process leases still need fencing or a
maintenance drain before cutover. Run `rg -n 'current_region|map_instance_id|player_world_state_v2' backend frontend/src`
and review SQL aliases and `models.Player.X/Y` call sites before declaring the
audit complete.
