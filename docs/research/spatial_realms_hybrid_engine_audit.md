# Báo Cáo Nghiên Cứu Chi Tiết: Audit Kiến Trúc Spatial Realms & Đề Xuất Hybrid Engine (Issue #233)

> **Mã vé:** [#233](https://github.com/VipPro67/GuWorld/issues/233)  
> **Người thực hiện:** Research Subagent (Spatial Instance Engine Researcher)  
> **Phạm vi khảo sát:** `spatial_realms.go`, `schema_world.go`, `schema_player_items.go`, `instance.go`, `SharedSpatialRealmControls.tsx`, `useWorldStore.ts`, `WorldCanvas.tsx`, `exploration_move.go`, `map_state.go`.

---

# AUDIT & ARCHITECTURE SPEC: HYBRID SPATIAL REALMS ENGINE (TICKET #233)

## 1. Hiện Trạng Kiến Trúc `shared_spatial_instances` và `spatial_realms.go`

### 1.1. Cấu trúc Dữ liệu Database
1. **Bảng `instance_templates` (`backend/internal/db/schema_world.go:207-222`)**:
   - Quản lý metadata mẫu bí cảnh. Đối với các bí cảnh không gian (`type = 'SPATIAL_REALM'`), template liên kết với một vùng địa lý qua cột `spatial_region_id VARCHAR(50) REFERENCES regions(id)` và có thời hạn tồn tại `duration_minutes INTEGER DEFAULT 120`.
2. **Bảng `shared_spatial_instances` (`backend/internal/db/schema_world.go:223-238`)**:
   - Quản lý phiên bí cảnh chung đang chạy: `id UUID`, `template_id`, `region_id`, `status ('ACTIVE'/'CLOSED')`, `opened_at`, `expires_at`, `closed_at`, `close_reason`.
   - **Ràng buộc duy nhất cốt lõi**:
     ```sql
     CREATE UNIQUE INDEX IF NOT EXISTS idx_shared_spatial_instances_one_active 
     ON shared_spatial_instances(template_id) WHERE status='ACTIVE';
     ```
     *(Dòng 234)*: Toàn bộ server chỉ cho phép **tối đa 1 phiên ACTIVE** cho mỗi template tại một thời điểm. Mọi người chơi vào cùng template đều dùng chung phiên này.
3. **Bảng `shared_spatial_instance_members` (`backend/internal/db/schema_player_items.go:186-202`)**:
   - Quản lý người chơi trong phiên: `instance_id UUID REFERENCES shared_spatial_instances(id)`, `player_id UUID`, `return_region_id`, `return_x`, `return_y`, `joined_at`, `exited_at`, `exit_reason`.
   - Ràng buộc: `CREATE UNIQUE INDEX idx_shared_spatial_members_one_active_player ON shared_spatial_instance_members(player_id) WHERE exited_at IS NULL;` *(Dòng 198)* đảm bảo mỗi người chơi chỉ có thể ở trong 1 phiên bí cảnh tại 1 thời điểm.
4. **Cột `players.map_instance_id` (`backend/internal/db/schema_players.go:34, 163`)**:
   - `players` có cột `map_instance_id UUID` và index `idx_players_map_instance_pos ON players (map_instance_id, current_region, x, y)`.

### 1.2. Luồng Xử Lý Logic Backend (`backend/internal/handlers/spatial_realms.go`)
- **Danh sách bí cảnh (`ListSpatialRealms`, L35-73)**: Query các template `type='SPATIAL_REALM'`, LEFT JOIN với `map_cells gate` (`cell_type='ENTRY_PORTAL'`) và `shared_spatial_instances s` (`status='ACTIVE' AND expires_at>NOW()`), đếm số lượng người chơi đang có mặt.
- **Vào bí cảnh (`EnterSpatialRealm`, L109-271)**:
  - Khóa hàng player `FOR UPDATE`, kiểm tra `map_instance_id IS NULL`, không có `active_combat` (L131-144).
  - Kiểm tra không có phiên text-instance `player_instance_sessions WHERE status='ACTIVE'` (L146-153).
  - Kiểm tra người chơi đang đứng đúng ô cổng vào `ENTRY_PORTAL` trên thế giới (L170-183).
  - Kiểm tra điều kiện cảnh giới (`rank >= required_rank`), tiêu hao vật phẩm `entry_item` nếu có (L185-217).
  - Dọn dẹp phiên cũ hết hạn qua `closeSpatialRealmTx` (L220-238).
  - Tìm instance đang ACTIVE; nếu chưa có thì INSERT tạo mới với thời hạn `duration_minutes` (L240-248).
  - Ghi nhận thành viên vào `shared_spatial_instance_members` lưu lại điểm xuất phát `oldRegion, oldX, oldY` (L249-253).
  - Chuyển vị trí nhân vật: `UPDATE players SET current_region=$1, x=$2, y=$3, map_instance_id=$4 WHERE id=$5` (L254-258).
  - Phát sóng vị trí ban đầu qua `syncSpatialPlayer` -> `broadcastWorldPos` (L266, L444-454).
- **Thoát bí cảnh (`ExitSpatialRealm`, L275-329)**:
  - Yêu cầu người chơi đứng tại ô thoát `cell_type IN ('EXIT_PORTAL', 'PORTAL_EXIT')` (L300-308).
  - Cập nhật trả người chơi về vị trí ban đầu: `UPDATE players SET current_region=$1, x=$2, y=$3, map_instance_id=NULL WHERE id=$4 AND map_instance_id=$5` (L309-318).
  - Cập nhật `shared_spatial_instance_members SET exited_at=NOW(), exit_reason='PORTAL'` (L319-322).
  - Đồng bộ vị trí với `uuid.Nil` (L327).
- **Đóng phiên & Quét quá hạn (`closeSpatialRealmTx` & `StartSpatialRealmExpiryWorker`, L331-442)**:
  - Background worker chạy mỗi 15 giây (`RunEvery`), quét các instance quá hạn `expires_at <= NOW()`.
  - Đưa tất cả thành viên còn lại về vị trí ban đầu và cập nhật `exit_reason='TIMEOUT'`.
  - **Dọn dẹp vật phẩm rơi trên sàn**: `DELETE FROM map_ground_drops WHERE region_id=$1` (L368).
  - Chuyển trạng thái instance sang `status='CLOSED'`.

### 1.3. Phân Tách Với Text-Based Instance (`backend/internal/handlers/instance.go`)
- `instance.go` quản lý các phó bản theo lượt/stage (Stage 0, 1, 2) qua bảng `player_instance_sessions`.
- Trong `InitInstanceCache` (L54): Lọc bỏ `SPATIAL_REALM` khỏi danh sách stage-based.
- Trong `EnterInstance` (L348-351): Chặn rõ ràng: `if template.Type == "SPATIAL_REALM" { c.JSON(400, "Bí cảnh bản đồ dùng API spatial riêng"); return }`.
- Trong `EnterSpatialRealm` (L146): Chặn người chơi vào spatial nếu đang có active `player_instance_sessions`.

### 1.4. Giao Diện Frontend
- **`frontend/src/components/SharedSpatialRealmControls.tsx`**:
  - Gọi `/api/instance/spatial` và `/api/instance/spatial/active` mỗi 30 giây (L36-68).
  - Hiển thị thông tin bí cảnh chung, kiểm tra điều kiện đứng đúng cổng `atGate` để mở nút "Tiến vào / Tham gia" (L143-160).
  - Khi đang ở trong bí cảnh, hiển thị đếm ngược thời gian và nút "Thoát tại cổng" gọi `/api/instance/spatial/exit` (L127-139).
- **`frontend/src/store/useWorldStore.ts`**:
  - `movePath` (L324-370): Gửi lệnh di chuyển qua WebSocket `world.move_path` hoặc HTTP `/api/explore/move-path`.
  - `fetchNearbyPlayers` (L629-660): Gọi `/api/explore/players` lấy danh sách người chơi xung quanh.

---

## 2. Các Giới Hạn Cốt Lõi Khiến Không Thể Dùng Cho Private / Party Instance

1. **Ràng buộc Unique Instance trong Database**:
   - `idx_shared_spatial_instances_one_active` chỉ cho phép **1 instance active duy nhất trên mỗi template_id**. Nếu mở tính năng Truyền Thừa cá nhân/tổ đội (nhiều tổ đội cùng đi 1 template bí cảnh cùng lúc), hệ thống sẽ xung đột ngay lập tức (không thể INSERT instance mới, hoặc tất cả các đội bị dồn chung vào 1 instance).
   - Bảng `shared_spatial_instances` không có cột phân loại chế độ (`scope`: `SHARED`, `PARTY`, `SOLO`), không có `party_id` hay `owner_id`.
2. **Xung đột State Ô Bản Đồ (Map Cells) trong Redis & Memory**:
   - Trong `backend/internal/cache/map_state.go:31-33`:
     ```go
     func cellKey(regionID string, x, y int) string {
         return fmt.Sprintf("map:cell:%s:%d:%d", regionID, x, y)
     }
     ```
     Khóa Redis đánh dấu ô đã bị dọn (`cleared`) chỉ gồm `regionID:x:y`. Nếu nhiều phiên Private/Party dùng chung template `spatial_region_id`, việc Đội A tiêu diệt quái/nhặt rương tại ô (X, Y) sẽ khiến ô đó bị biến mất ở cả phiên của Đội B!
   - Trong `backend/internal/handlers/zone_registry.go:757-778`:
     `MarkPersistentCellCleared` cập nhật trực tiếp vào in-memory cache toàn cục `zoneReg.persistent[key]` và DB `UPDATE map_cells SET last_cleared_at = NOW() WHERE region_id=$2 AND x=$3 AND y=$4`. Đồng thời phát `broadcastWorldCell` kênh `"world:cell"` cho toàn server. Điều này gây ô nhiễm trạng thái giữa các instance độc lập.
3. **Xung đột Vật Phẩm Rơi Trên Sàn (`map_ground_drops`)**:
   - Bảng `map_ground_drops` (`backend/internal/db/schema_world.go:267-283`) chỉ có `region_id, x, y` mà **không có `map_instance_id`**.
   - Trong `exploration_grid.go:190-199`, query vật phẩm rơi chỉ lọc theo `region_id`. Người chơi ở các instance khác nhau nếu cùng tọa độ sẽ thấy đồ của nhau!
   - Khi một phiên kết thúc (`closeSpatialRealmTx`, `spatial_realms.go:368`), lệnh:
     `DELETE FROM map_ground_drops WHERE region_id=$1`
     sẽ **xóa sạch toàn bộ đồ rơi** của tất cả các phiên khác đang chạy trong cùng `region_id`!
4. **Vô Hiệu Hóa Monster Engine Khi Có `map_instance_id`**:
   - Trong `backend/internal/handlers/world_monsters.go:734-737`:
     ```go
     if entry.MapID != "world_map" || entry.InstanceID != "" {
         c.JSON(http.StatusOK, gin.H{"monsters": []WorldMonster{}, "server_time": time.Now().UTC()})
         return
     }
     ```
     Hệ thống quái vật chủ động (quái roam, AI phát hiện và tấn công người chơi) hiện bị hard-code tắt hoàn toàn khi người chơi có `entry.InstanceID != ""`.
5. **Cơ Chế Đồng Bộ Vị Trí Người Chơi (Position Sync & Scope Leak)**:
   - **Mặt tốt**: Bộ nhớ RAM `worldPosIndex` (`backend/internal/handlers/world_index.go:44-47, 84-94`) tổ chức theo `worldBucketKey{mapID, instanceID, bx, by}`. Hàm `NearbyInScope` (L150-204) lọc chính xác theo `instanceID`. Người chơi khác `instanceID` không thể thấy nhau trong radar in-memory.
   - **Lỗ hổng rò rỉ**:
     - Trong `exploration_move.go:865-873, 1188-1196`: Khi di chuyển (`MovePlayer` và `MovePlayerPath`), struct `WorldPosEvent` được phát đi **không gán trường `InstanceID`**, buộc server phải đoán dựa trên `previous.InstanceID` trong `worldPosIndex` (`world_events.go:71-75`). Nếu index chưa kịp nạp, tọa độ bị xem như ở World Map!
     - Trong frontend `WorldCanvas.tsx:2720-2732`: Handler `onWorldPos` khi nhận event `"gu_world_pos"` chỉ kiểm tra `detail.id !== player.id`, hoàn toàn **không kiểm tra `detail.map_instance_id === player.map_instance_id`**. Nếu WebSocket fanout lọt gói tin, client sẽ vẽ bóng người chơi của phiên khác lên màn hình.
   - **Tổ đội di chuyển**: `SyncPartyMembersLocation` (`backend/internal/handlers/party.go:575`) kiểm tra nghiêm ngặt `leaderMapInstance.Valid != m.instance.Valid || leaderMapInstance.String != m.instance.String`. Nếu không có luồng Party Entry đưa cả đội vào cùng lúc thì đội trưởng đi vào bí cảnh sẽ lập tức mất đồng bộ di chuyển với các thành viên.

---

## 3. Kiến Trúc Mở Rộng: Hybrid Spatial Realm Engine

### 3.1. Thiết Kế Mở Rộng Database Schema
Nâng cấp bảng `shared_spatial_instances` thành bảng tổng quát `spatial_instances`:

```sql
-- Đổi tên hoặc tạo bảng spatial_instances
CREATE TABLE IF NOT EXISTS spatial_instances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id VARCHAR(50) NOT NULL REFERENCES instance_templates(id) ON DELETE CASCADE,
    region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE CASCADE,
    scope VARCHAR(20) NOT NULL DEFAULT 'SHARED' CHECK (scope IN ('SHARED', 'PARTY', 'SOLO')),
    owner_id UUID REFERENCES players(id) ON DELETE CASCADE,     -- Nullable, dùng cho SOLO
    party_id UUID REFERENCES parties(id) ON DELETE CASCADE,     -- Nullable, dùng cho PARTY
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'CLOSED')),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    closed_at TIMESTAMPTZ,
    close_reason VARCHAR(30) NOT NULL DEFAULT ''
);

-- Ràng buộc Index tách biệt theo từng Scope:
-- 1. Bí cảnh chung: Duy nhất 1 instance ACTIVE trên mỗi template
CREATE UNIQUE INDEX IF NOT EXISTS idx_spatial_instances_shared_active 
    ON spatial_instances(template_id) WHERE scope='SHARED' AND status='ACTIVE';

-- 2. Bí cảnh tổ đội: Mỗi party chỉ tham gia tối đa 1 instance ACTIVE
CREATE UNIQUE INDEX IF NOT EXISTS idx_spatial_instances_party_active 
    ON spatial_instances(party_id) WHERE scope='PARTY' AND status='ACTIVE';

-- 3. Bí cảnh cá nhân: Mỗi player chỉ sở hữu tối đa 1 instance ACTIVE
CREATE UNIQUE INDEX IF NOT EXISTS idx_spatial_instances_solo_active 
    ON spatial_instances(owner_id) WHERE scope='SOLO' AND status='ACTIVE';

-- Index hỗ trợ worker quét hạn dùng
CREATE INDEX IF NOT EXISTS idx_spatial_instances_expiry 
    ON spatial_instances(expires_at) WHERE status='ACTIVE';
```

Bảng thành viên `spatial_instance_members`:
```sql
CREATE TABLE IF NOT EXISTS spatial_instance_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instance_id UUID NOT NULL REFERENCES spatial_instances(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    return_region_id VARCHAR(50) NOT NULL REFERENCES regions(id) ON DELETE RESTRICT,
    return_x INTEGER NOT NULL,
    return_y INTEGER NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    exited_at TIMESTAMPTZ,
    exit_reason VARCHAR(30) NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_spatial_members_one_active_player 
    ON spatial_instance_members(player_id) WHERE exited_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_spatial_members_active_instance 
    ON spatial_instance_members(instance_id, player_id) WHERE exited_at IS NULL;
```

Bổ sung cột `map_instance_id` vào `map_ground_drops`:
```sql
ALTER TABLE map_ground_drops ADD COLUMN IF NOT EXISTS map_instance_id UUID REFERENCES spatial_instances(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_map_ground_drops_instance ON map_ground_drops(map_instance_id, region_id, x, y);
```

### 3.2. Cách Ly Dữ Liệu Ô Bản Đồ & Trạng Thái Dọn Ô (Cell State Isolation)
Để tránh việc phải nhân bản hàng nghìn bản ghi `map_cells` vật lý trong DB, đề xuất giải pháp **Scoped Cell State**:
1. **Redis Cell Keying**:
   Mở rộng helper key trong `backend/internal/cache/map_state.go`:
   ```go
   func cellKey(regionID string, instanceID string, x, y int) string {
       if instanceID == "" {
           return fmt.Sprintf("map:cell:%s:world:%d:%d", regionID, x, y)
       }
       return fmt.Sprintf("map:cell:%s:%s:%d:%d", regionID, instanceID, x, y)
   }
   ```
2. **Persistent Cell Clearing Trong Bí Cảnh**:
   - Đối với bí cảnh `PARTY` và `SOLO`: Trạng thái clear ô (Boss, hòm báu, thảo dược) chỉ ghi vào Redis với `instanceID` tương ứng, **không gọi `UPDATE map_cells` vào database vật lý**.
   - Khi phiên kết thúc, toàn bộ key Redis của instance tự hủy theo TTL hoặc xóa sạch qua tiền tố `map:cell:*:{instanceID}:*`.
   - Sự kiện WebSocket `world:cell` phải kèm theo `map_instance_id`, và frontend chỉ áp dụng nếu khớp với `map_instance_id` hiện tại của nhân vật.
3. **Cách Ly Ground Drops**:
   - Khi rơi đồ trong bí cảnh, ghi `map_instance_id = player.MapInstanceID`.
   - Query trong `exploration_grid.go` và `ground_drops.go` thêm điều kiện:
     `AND map_instance_id IS NOT DISTINCT FROM $X`.
   - Khi đóng instance (`closeSpatialRealmTx`), chỉ xóa:
     `DELETE FROM map_ground_drops WHERE map_instance_id = $1`.

### 3.3. Hoàn Thiện Cơ Chế Đồng Bộ Vị Trí (Position Sync & Party Entry)
1. **Sửa rò rỉ tại `exploration_move.go`**:
   - Tại dòng 865 và dòng 1188, truyền rõ ràng `InstanceID: player.MapInstanceID.String()`.
2. **Lọc tại Frontend Canvas (`frontend/src/components/WorldCanvas.tsx`)**:
   - Tại dòng 2721, bổ sung so khớp `localPlayer?.map_instance_id === detail.map_instance_id` để ngăn vẽ bóng người chơi của phiên khác.
3. **Cơ chế Party Entry (Đội Trưởng Mở Phiên Cho Cả Đội)**:
   - Endpoint `POST /api/instance/spatial/enter` nhận thêm payload:
     `{ "template_id": string, "scope": "SHARED" | "PARTY" | "SOLO" }`.
   - Nếu `scope == "PARTY"`:
     - Kiểm tra người gọi có `party_id != nil` và là `leader_id` trong bảng `parties`.
     - Kiểm tra toàn bộ thành viên đang online, đứng cùng tọa độ cổng với đội trưởng, đủ rank và vật phẩm.
     - Tạo một `spatial_instances` mới với `scope='PARTY', party_id=leader.PartyID`.
     - Trong cùng một Transaction: INSERT tất cả thành viên vào `spatial_instance_members` và UPDATE `players.map_instance_id = newInstanceID, current_region = regionID, x = entryX, y = entryY`.
     - Broadcast event cho toàn đội để frontend client của các thành viên tự động reload map và hiển thị canvas bí cảnh.
