# Audit Log Implementation Plan

## 1. `handlers/admin.go` — Admin audit log endpoint + log all mutations

- Add `"fmt"` and `"github.com/vippro67/GuWorld/backend/pkg/auditlog"` to imports
- Add `AdminGetAuditLogs` handler: `GET /api/admin/audit-logs?page=&limit=&action=&player_id=&success=&entity_type=&from=&to=`
- In each admin mutation handler, change `if _, ok :=` to `adminID, ok :=` and add `auditlog.Log()` after success

Admin actions to log:
- `admin.add_player_items`, `admin.remove_player_items`
- `admin.add_player_gus`, `admin.remove_player_gu`
- `admin.change_player_stones`
- `admin.update_sect`, `admin.set_sect_leader`, `admin.kick_sect_member`
- `admin.add_to_sect`, `admin.delete_sect`
- `admin.stock_sect_vault`, `admin.remove_sect_vault_item`
- `admin.create_sect_mission`, `admin.delete_sect_mission`
- `admin.import_data`, `admin.export_data`

## 2. `server.go`

```go
api.GET("/admin/audit-logs", middleware.AuthRequired(), handlers.AdminGetAuditLogs)
```

## 3. `handlers/auth.go`

- `LoginPlayer`: add `player.login` log with IP
- `RegisterPlayer`: add `player.register` log
- `LogoutPlayer`: add `player.logout` log

## 4. `handlers/sect.go`

- `JoinSect`: add `sect.join`
- `LeaveSect`: add `sect.leave`
- `KickMember`: add `sect.kick`
- `CreateSect`: add `sect.create`

## 5. `handlers/mine*.go`

- `OccupyMineSlot`: add `mine.occupy`
- `ClaimMineRewards`: add `mine.claim`
- `AbandonMineSlot`: add `mine.abandon`
- `AttackMineSlot`: add `mine.attack`

## 6. `handlers/player.go` (inventory)

- In `EquipGu`, log `gu.equip` / `gu.unequip` based on Equipped value change

## 7. `handlers/combat_retreat.go`

Uncomment line 257 (`combat.retreat` audit log)

## 8. `handlers/combat*.go` / exploration

- In combat start: `combat.start`
- In flee: `combat.flee`  
- In combat result handlers: `combat.victory` / `combat.defeat`

## 9. `handlers/quests.go`

In `CompleteQuest`, add `quest.complete` log before returning success.

## 10. `AdminView.tsx`

- Import `ScrollText` icon
- Add "Nhật ký" section tab
- Log viewer with: filter inputs (action, player_id, success), paginated table, expandable rows, color-coded badges
- Fetch from `GET /api/admin/audit-logs`
