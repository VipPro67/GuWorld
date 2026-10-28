# Ma trận tài nguyên quản trị

Tài liệu này là kết quả đối chiếu thủ công giữa schema, route Admin, handler,
CSV và catalog điều hướng. Đây là **allowlist theo domain**, không phải mô tả để
sinh raw-row editor từ database.

## Quy ước phân loại

| Loại | Ý nghĩa |
| --- | --- |
| **Full CRUD** | Có endpoint chuyên biệt create/read/update/delete cho dữ liệu cấu hình. |
| **Read-only** | Admin chỉ được tra cứu; không cung cấp mutation hoặc raw-row editor. |
| **Domain command only** | Chỉ thao tác bằng command giữ invariant của aggregate (grant, reset, đổi trạng thái, ...), không CRUD dòng thô. |
| **Import/export only** | Chỉ đi qua pipeline CSV có allowlist và validation; không phải CRUD UI. |
| **Internal/not exposed** | Không thuộc bề mặt Admin. |

Một resource mutation chỉ được thêm màn hình sau khi có đủ: permission riêng,
validation domain, audit, chiến lược reload registry/xóa cache và test. Việc có
bảng hoặc CSV **không** tự động làm resource đủ điều kiện. Payment, auth/session,
audit và dữ liệu security không bao giờ được dùng generic editor.

## Bảng database

Phân loại dưới đây mô tả bề mặt Admin hiện tại/an toàn. Một số bảng được gameplay
ghi bằng domain command nhưng vẫn là `Internal/not exposed` nếu không có Admin API.

| Schema | Bảng | Phân loại | Bề mặt Admin / ghi chú |
| --- | --- | --- | --- |
| admin RBAC/authorization | `admin_roles` | Internal/not exposed | Security catalog; handler role chưa được đăng ký route, cấm raw editor. |
| admin RBAC/authorization | `admin_permissions` | Internal/not exposed | Security permission allowlist, cấm raw editor. |
| admin RBAC/authorization | `admin_role_permissions` | Internal/not exposed | Security join table, cấm raw editor. |
| admin RBAC/authorization | `player_admin_roles` | Internal/not exposed | Chỉ được thay bằng role-assignment command có kiểm tra thẩm quyền và audit. |
| aperture | `immortal_apertures` | Domain command only | List/detail/update aggregate qua `/admin/apertures`. |
| aperture | `aperture_resource_nodes` | Domain command only | Add/remove node trong aperture; không sửa dòng thô. |
| aperture | `aperture_variant_humans` | Domain command only | Add/remove quần thể qua aperture aggregate. |
| aperture | `aperture_node_workers` | Internal/not exposed | Trạng thái worker/runtime. |
| aperture | `aperture_refinement_queue` | Internal/not exposed | Queue/runtime state. |
| aperture | `aperture_variant_settlements` | Internal/not exposed | Gameplay-owned child state. |
| aperture | `aperture_variant_events` | Internal/not exposed | Event history/runtime. |
| actors | `non_player_actors` | Domain command only | Canonical pet/NPC instances; pet admin commands are exposed, not a generic row editor. |
| aperture | `tribulation_events` | Read-only | Paginated per-aperture history includes event outcome, owner, rewards and spawned enemies. |
| aperture | `tribulation_enemies` | Read-only | Returned nested under its tribulation event; no raw combat-state mutation. |
| beast betting | `beast_betting_rounds` | Internal/not exposed | Betting lifecycle; future admin action must be a domain command. |
| beast betting | `beast_bets` | Internal/not exposed | Financial/gameplay ledger; immutable from Admin. |
| combat | `combat_history` | Read-only | `/admin/combat-history`; PVP history and replay logs, append-only. |
| combat | `players.active_combat` | Internal/not exposed | Active session state on player row, cấm editor. |
| death arena | `death_arena_tournaments` | Internal/not exposed | Tournament aggregate state. |
| death arena | `death_arena_teams` | Internal/not exposed | Tournament aggregate child. |
| death arena | `death_arena_team_members` | Internal/not exposed | Tournament aggregate child. |
| death arena | `death_arena_matches` | Internal/not exposed | Match/runtime state. |
| death arena | `death_arena_champion_rewards` | Internal/not exposed | Reward ledger. |
| dictionaries | `item_dictionary` | Full CRUD | Specialized Item handlers; đồng thời có CSV import/export. |
| dictionaries | `gu_dictionary` | Full CRUD | Specialized Gu handlers + effect validation/registry reload; đồng thời có CSV. |
| dictionaries | `constitution_dictionary` | Full CRUD | Specialized handlers + effect validation/registry reload; đồng thời có CSV. |
| dictionaries | `race_dictionary` | Full CRUD | Specialized handlers + effect validation/registry reload; đồng thời có CSV. |
| dictionaries | `status_dictionary` | Full CRUD | Specialized handlers; cache/registry gap phải đóng trước khi mở rộng UI. |
| dictionaries | `loot_tables` | Full CRUD | Specialized loot-table handlers; đồng thời có CSV. |
| dictionaries | `refining_recipes` | Full CRUD | Specialized recipe handlers + catalog invalidation; đồng thời có CSV. |
| dictionaries | `caravan_shop_items` | Full CRUD | `/admin/shop`; cần bảo đảm invalidation catalog sau mutation. |
| dictionaries | `caravan_shop_daily_stock` | Internal/not exposed | Derived/day-scoped stock state. |
| dictionaries | `arena_shop_items` | Full CRUD | Specialized arena-shop handlers; đồng thời có CSV. |
| effect catalog | `effect_catalog_revisions` | Internal/not exposed | Immutable catalog/version infrastructure. |
| effect catalog | `effect_catalog_snapshots` | Internal/not exposed | Immutable snapshot infrastructure. |
| Gu history | `gu_history_logs` | Read-only | `/admin/gu-history`; không mutation. |
| payment | `topup_orders` | Read-only | `/admin/topup-orders`; provider state, cấm raw editor. |
| payment | `wallet_transactions` | Read-only | `/admin/wallet-transactions`; ledger, cấm mutation. |
| payment | `shop_package_purchases` | Internal/not exposed | Purchase ledger. |
| payment | `gift_codes` | Domain command only | List/create command; redemption/counter invariant không CRUD thô. |
| payment | `gift_code_redemptions` | Internal/not exposed | Redemption ledger, cấm editor. |
| player items | `player_gu` | Domain command only | Grant/update/remove Gu theo player command. |
| player items | `killer_moves` | Internal/not exposed | Player-owned combat data. |
| player items | `killer_move_slots` | Internal/not exposed | Player-owned combat layout. |
| player items | `player_inventory_items` | Domain command only | Grant/remove inventory command. |
| player items | `player_recipe_progress` | Domain command only | Grant/remove recipe qua inventory command. |
| player items | `player_quest_progress` | Internal/not exposed | Gameplay progress. |
| player items | `player_dao_path` | Internal/not exposed | Gameplay progression state. |
| player items | `player_map_progress` | Internal/not exposed | Gameplay progress. |
| player items | `player_instance_sessions` | Internal/not exposed | Active session/runtime state. |
| players | `players` | Domain command only | Redacted read/update và explicit commands; password/auth fields không được lộ. |
| players | `player_collection_revisions` | Internal/not exposed | Cache/coherency revision. |
| players | `referrals` | Internal/not exposed | Có IP/device security metadata. |
| players | `leaderboard_settings` | Internal/not exposed | Runtime/config chưa có Admin API. |
| players | `wallets` | Internal/not exposed | Balance aggregate; chỉ thay đổi qua ledger/domain command. |
| players | `email_otps` | Internal/not exposed | Auth secret; tuyệt đối không catalog/editor. |
| players | `player_achievements` | Internal/not exposed | Gameplay progression. |
| scouting | `player_tracking_marks` | Internal/not exposed | Private gameplay/security-sensitive tracking. |
| scouting | `world_residual_traces` | Internal/not exposed | Runtime trace data. |
| sects | `sects` | Domain command only | Detail/update/delete aggregate; không generic create/editor. |
| sects | `sect_missions` | Domain command only | List/create/delete trong sect; đồng thời có CSV seed. |
| sects | `sect_members` | Domain command only | Add/kick/set leader commands. |
| sects | `sect_join_requests` | Domain command only | Admin xem toàn bộ; chỉ hủy yêu cầu đang chờ, không sửa thô hoặc giả lập duyệt thành viên. |
| sects | `sect_promotion_requests` | Domain command only | Admin xem cấp hiện tại/cấp yêu cầu; chỉ hủy yêu cầu đang chờ. |
| sects | `player_sect_missions` | Internal/not exposed | Player progress. |
| sects | `sect_defenders` | Domain command only | Read full defense roster; remove defender through sect-scoped command, never edit combat power fields directly. |
| sects | `sect_vault_materials` | Domain command only | Stock/remove vault command. |
| sects | `sect_vault_gu` | Domain command only | Stock/remove vault command. |
| sects | `sect_vault_recipes` | Domain command only | Stock/remove vault command. |
| social | `messages` | Internal/not exposed | Private communication data. |
| social | `feedback` | Internal/not exposed | Public/domain moderation routes nằm ngoài Admin catalog. |
| social | `feedback_votes` | Internal/not exposed | User activity. |
| social | `feedback_comments` | Internal/not exposed | User content/activity. |
| social | `audit_logs` | Read-only | `/admin/audit-logs`; append-only, cấm raw editor. |
| social | `notifications` | Internal/not exposed | Player-owned messages. |
| teleport | `teleport_arrays` | Internal/not exposed | Gameplay aggregate; future mutation phải là domain command. |
| teleport | `teleport_array_logs` | Internal/not exposed | Append-only activity log. |
| trade | `parties` | Internal/not exposed | Live aggregate. |
| trade | `auction` | Internal/not exposed | Economy transaction state. |
| trade | `trades` | Internal/not exposed | Economy transaction state. |
| trade | `party_invitations` | Internal/not exposed | Workflow state. |
| trade | `buyback_orders` | Internal/not exposed | Economy order state. |
| trade | `trade_offers` | Internal/not exposed | Economy transaction state. |
| world | `regions` | Full CRUD | Specialized region handlers + Zone registry reload; đồng thời có CSV. |
| world | `monster_templates` | Full CRUD | Specialized monster handlers + registry reload; đồng thời có CSV. |
| world | `monster_spawns` | Full CRUD | Specialized handlers; registry refresh gap phải đóng trước khi mở rộng UI. |
| world | `map_cells` | Full CRUD | Specialized map-cell handlers + Zone registry reload; đồng thời có CSV. |
| world | `quests` | Full CRUD | Specialized handlers; quest registry refresh gap phải đóng trước khi mở rộng UI. |
| world | `instance_templates` | Full CRUD | `/admin/dungeons`; Dream Realm còn có CSV, instance cache gap phải đóng. |
| world | `map_ground_drops` | Domain command | `/admin/map-drops`; browse active/expired drops, edit complete drop fields, or delete with reason and audit. |

## CSV và import/export catalog

| CSV | Resource đích | Phân loại | Ghi chú |
| --- | --- | --- | --- |
| `Item.csv` | `item_dictionary` | Full CRUD | Có thêm import/export allowlist. |
| `Gu.csv` | `gu_dictionary` | Full CRUD | Import phải chạy effect preflight trước write. |
| `Constitution.csv` | `constitution_dictionary` | Full CRUD | Effect-bearing registry. |
| `Race.csv` | `race_dictionary` | Full CRUD | Effect-bearing registry. |
| `Status.csv` | `status_dictionary` | Full CRUD | Effect-bearing registry. |
| `Recipe.csv` | `refining_recipes` | Full CRUD | Invalidate recipe catalog sau thay đổi. |
| `ShopItem.csv` | `caravan_shop_items` | Full CRUD | Có thêm import/export allowlist. |
| `ArenaShopItem.csv` | `arena_shop_items` | Full CRUD | Có thêm import/export allowlist. |
| `Region.csv` | `regions` | Full CRUD | Reload Zone registry. |
| `LootTable.csv` | `loot_tables` | Full CRUD | Reload monster/dictionary cache. |
| `Monster.csv` | `monster_templates` | Full CRUD | Reload Monster registry. |
| `MonsterSpawn.csv` | `monster_spawns` | Full CRUD | Reload Zone/Monster registry. |
| `MapCell.csv` | `map_cells` | Full CRUD | Reload Zone registry. |
| `Quest.csv` | `quests` | Full CRUD | Reload Quest registry. |
| `DreamRealm.csv` | dream-realm subset của `instance_templates` | Import/export only | Không suy rộng thành editor mọi instance row. |
| `SectMission.csv` | `sect_missions` | Domain command only | CSV seed + aggregate commands; không raw CRUD. |
| `Sect.csv` | `sects` | Domain command only | Hiện chỉ có export mapping, không thuộc import allowlist. |
| `ApertureNodeTemplate.csv` | file-backed node template | Internal/not exposed | Startup/file cache; chưa có pipeline an toàn. |
| `PremiumShopPackage.csv` | file-backed premium catalog | Internal/not exposed | Commerce data; không generic editor. |
| `TopupPackage.csv` | file-backed top-up catalog | Internal/not exposed | Payment data; không generic editor. |

`db.AvailableTables()` hiện quảng bá `node_output` nhưng `TableImport`/export không
có implementation tương ứng. Frontend cũ cũng từng hard-code `sect` nhưng import
không hỗ trợ, đồng thời bỏ sót `status`. Đây là catalog drift; không được dùng các
entry lệch này để tự động sinh màn hình.

## Catalog module Admin

| Module | Phân loại | Permission để nhìn thấy | Phạm vi |
| --- | --- | --- | --- |
| `players` | Domain command only | `players.read` | Redacted player + inventory/Gu/pet commands. |
| `sects` | Domain command only | `sects.read` | Sect aggregate commands. |
| `apertures` | Domain command only | `apertures.read` | Aperture aggregate commands. |
| `gu_history` | Read-only | `gus.read` | History only. |
| `data` | Import/export only | `game_data.read` | CSV operations; nested dictionary CRUD vẫn cần permission riêng. |
| `shop` | Full CRUD | `shop.read` | Caravan shop; arena shop dùng permission riêng cho mutation. |
| `gift_codes` | Domain command only | `gift_codes.read` | List/create only. |
| `topup_wallet` | Read-only | `billing.read` | Payment/order ledger; không mutation. |
| `logs` | Read-only | `audit.read` | Audit log, append-only. |
| `monitoring` | Domain command only | `monitoring.read` | Metrics/traces + explicit reset command. |

Catalog backend chỉ chứa metadata ứng dụng và được lọc theo effective permission;
không trả tên bảng, cột, schema hoặc SQL metadata. Permission nhìn thấy module
không thay thế permission mutation trên từng route; mảng `capabilities` cũng
được lọc theo permission mutation của actor. `data` ở bảng này phân loại
resource vận chuyển CSV; các editor chuyên biệt nằm cùng trang được phân
loại riêng theo bảng/resource ở hai ma trận phía trên.

## Gate trước khi thêm màn hình

Checklist bắt buộc cho từng mutation resource:

1. **Permission:** route khai báo permission cụ thể trong `adminRouteSpecs`; UI ẩn
   action nếu thiếu đúng permission đó, backend vẫn là enforcement authority.
2. **Validation:** request DTO/allowlist/FK/effect schema kiểm tra trước write;
   không nhận arbitrary column/value.
3. **Audit:** mutation thành công/thất bại quan trọng ghi actor, action, entity và
   context; audit record không được sửa từ Admin.
4. **Cache/registry:** có reload/invalidation xác định và xử lý lỗi; không yêu cầu
   operator đoán rằng phải restart nếu catalog tuyên bố hot reload.
5. **Test:** có test permission, validation, audit và cache/registry behavior cho
   mutation trước khi expose UI.

Các gap đã thấy khi đối chiếu (status/spawn/quest/dungeon/shop invalidation, pet
validation/audit, mutation-level UI gating và thiếu handler tests) là backlog an
toàn; không được dùng để mở thêm editor cho đến khi đóng đủ năm gate.
