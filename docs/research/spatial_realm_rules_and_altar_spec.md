# Đặc Tả Kỹ Thuật: Rule Engine & Cơ Chế Đài Truyền Thừa (Issue #234)

> **Mã vé:** [#234 Decision](https://github.com/VipPro67/GuWorld/issues/234)  
> **Trạng thái:** Resolved (Đã chốt thiết kế)  
> **Tài liệu liên quan:** `docs/research/spatial_realms_hybrid_engine_audit.md`, `CONTEXT.md`

---

## 1. Cấu Trúc Dữ Liệu Quy Tắc Bí Cảnh (`realm_rules`)

### 1.1. Cột JSONB `rules` trong Cấu Hình Mẫu Bí Cảnh
Thay vì tạo bảng quan hệ cứng nhắc, mỗi Bí Cảnh Không Gian hoặc Vùng Truyền Thừa mang trường `rules JSONB NOT NULL DEFAULT '{}'` với cấu trúc chuẩn:

```json
{
  "duration_minutes": 90,
  "pvp_mode": "SAFE",
  "max_allowed_rank": 4,
  "apply_rank_suppression": true,
  "death_penalty": "EVICT",
  "allow_teleport": false,
  "entry_item": "talisman_wood",
  "entry_item_count": 1,
  "min_members": 1,
  "max_members": 5
}
```

### 1.2. Diễn Giải Các Trường Ràng Buộc & Thực Thi Runtime
| Trường | Kiểu | Mặc định | Ý nghĩa & Thực thi Runtime |
|---|---|---|---|
| `duration_minutes` | `int` | `120` | Thời gian tồn tại tối đa của phiên. Worker `StartSpatialRealmExpiryWorker` tự động trục xuất toàn bộ người chơi khi `expires_at <= NOW()`. |
| `pvp_mode` | `string` | `"SAFE"` | `"SAFE"` (Cấm giao tranh giữa người chơi), `"FREE_PK"` (Cho phép đồ sát tự do, tranh đoạt tài nguyên), `"FACTION_PK"` (Chính - Ma đối đầu). Handler `combat.go` kiểm tra rule này khi nhận lệnh tấn công mục tiêu là Player. |
| `max_allowed_rank` | `int` | `0` | Cảnh giới tối đa được phép vào. Nếu > 0 và `apply_rank_suppression == false`: Cổ sư có rank > `max_allowed_rank` bị chặn ngay tại cổng thế giới. |
| `apply_rank_suppression` | `bool` | `true` | Nếu `true`, Cổ sư cấp cao hơn `max_allowed_rank` vẫn được vào nhưng các chỉ số chiến đấu (`Attack`, `Defense`, `HP`, `Mana`, `Skill Power`) bị tạm thời áp chế về trần của `max_allowed_rank` khi đứng trong bí cảnh. |
| `death_penalty` | `string` | `"EVICT"` | `"EVICT"`: Khi tử vong trong bí cảnh, người chơi hồi sinh và bị trục xuất về tọa độ cổng vào thế giới ban đầu (`return_region_id, return_x, return_y`). `"SAFE_RESPAWN"`: Hồi sinh tại điểm vào của chính bí cảnh (`entry_x, entry_y`). |
| `allow_teleport` | `bool` | `false` | Cấm sử dụng các loại Cổ trùng/Phù truyền tống tức thời để trốn thoát khỏi bí cảnh; lối ra duy nhất là tới ô `EXIT_PORTAL` hoặc chờ hết giờ. |
| `entry_item` | `string` | `""` | ID vật phẩm/tín vật tiêu hao để mở cổng (ví dụ: Chìa khóa đồng, Lệnh bài truyền thừa). |
| `entry_item_count` | `int` | `1` | Số lượng tín vật tiêu hao mỗi lần vào. |

---

## 2. Cơ Chế Đài Truyền Thừa (`INHERITANCE_ALTAR`)

### 2.1. Biểu Diễn Trên Bản Đồ Ô (`map_cells`)
- **`cell_type`**: `'INHERITANCE_ALTAR'`
- **`content_id`**: Định danh của Đài (ví dụ: `altar_flower_wine`, `altar_lightning_emperor_1`)
- **`metadata` (JSONB)**:
  ```json
  {
    "altar_name": "Truyền Thừa Đài: Hoa Tửu Hành Giả",
    "path": "Food Path",
    "tier": "IMMORTAL",
    "guardian_spawn_id": "boss_flower_wine_guardian",
    "guardian_coord": {"x": 1015, "y": 1010},
    "req_rank": 3,
    "req_dao_marks": {
      "Food Path": 50
    },
    "required_item": "wine_pot_token",
    "rewards": {
      "gus": ["GU_FOOD_WINE_POT_4", "GU_SOUL_CLEANSE_3"],
      "recipes": ["recipe_wine_gu_3"],
      "primeval_stones": 50000,
      "materials": {
        "green_bamboo_wine": 10
      },
      "legacy_text": "Ta - Hoa Tửu Hành Giả, một đời phong lưu tiêu sái, để lại truyền thừa này cho hậu bối hữu duyên..."
    }
  }
  ```

### 2.2. Luồng Khảo Nghiệm 3 Bước Thực Tế
1. **Bước 1 - Thủ Hộ Giả (Guardian Boss)**:
   - Đài truyền thừa kiểm tra trạng thái của Boss canh gác (`guardian_spawn_id` hoặc tại `guardian_coord`).
   - Nếu trong phiên `map_instance_id` hiện tại, Boss chưa bị tiêu diệt, đài trả về lỗi: `"Thủ hộ giả của truyền thừa vẫn còn hiện diện, cơ quan truyền thừa chưa thể mở!"`.
2. **Bước 2 - Kiểm Tra Điều Kiện Kế Thừa (Requirements)**:
   - Cảnh giới tối thiểu: `player.Rank >= req_rank`.
   - Đạo Ngân lưu phái: Duyệt qua `player.DaoMarks`, kiểm tra xem người chơi có đủ số lượng Đạo Ngân quy định không (ví dụ: `Food Path >= 50`).
   - Vật phẩm tín vật (nếu có): Kiểm tra túi đồ của người chơi.
3. **Bước 3 - Nhận Chân Truyền & Ghi Nhận Lịch Sử**:
   - Kiểm tra chính sách nhận (Claim Policy):
     - Với `tier == "IMMORTAL"` (Rank 6+): Kiểm tra bảng `player_inheritance_claims`. Nếu đã nhận trong đời -> Từ chối: `"Đạo hữu đã lĩnh nhận toàn bộ chân truyền tại nơi này, không thể nhận thêm!"`.
     - Với `tier == "MORTAL"` (Rank 1-5): Có thể hỗ trợ chu kỳ làm mới nếu cấu hình `cooldown_hours > 0`.
   - Trong cùng 1 Transaction DB:
     - Ghi nhận vào bảng `player_inheritance_claims`.
     - Cấp phát Cổ trùng (`db.AddGuInstanceTx`), công thức luyện Cổ (`RecipeSvc.UnlockRecipeTx`), nguyên liệu, đá.
     - Phát thông báo sự kiện / Toast cho người chơi.

---

## 3. Thiết Kế Schema Bảng `player_inheritance_claims`

```sql
CREATE TABLE IF NOT EXISTS player_inheritance_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    altar_id VARCHAR(50) NOT NULL,
    template_id VARCHAR(50) NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    tier VARCHAR(20) NOT NULL DEFAULT 'MORTAL' CHECK (tier IN ('MORTAL', 'IMMORTAL')),
    rewards_snapshot JSONB NOT NULL DEFAULT '{}'
);

-- Index ngăn nhận đúp truyền thừa độc bản
CREATE UNIQUE INDEX IF NOT EXISTS idx_player_inheritance_unique_claim 
    ON player_inheritance_claims (player_id, altar_id) 
    WHERE tier = 'IMMORTAL';

CREATE INDEX IF NOT EXISTS idx_player_inheritance_history 
    ON player_inheritance_claims (player_id, claimed_at);
```

---

## 4. Chuẩn Hóa CSV Data-Driven

### 4.1. File `backend/data/SpatialRealm.csv`
Cấu trúc cột:
```csv
id,name,type,scope,spatial_region_id,entry_region_id,entry_x,entry_y,duration_minutes,required_rank,entry_item,rules,description
```
* `type`: `"SPATIAL_REALM"` (Bí Cảnh) hoặc `"SPATIAL_INHERITANCE"` (Truyền Thừa).
* `scope`: `"SHARED"` (chung toàn server), `"PARTY"` (tổ đội), `"SOLO"` (cá nhân).
* `rules`: Chuỗi JSON escape chứa cấu hình Rule Engine.

### 4.2. File `backend/data/MapCell.csv`
Bổ sung các dòng ô Đài Truyền Thừa vào các `region_id` bí cảnh:
```csv
region_id,x,y,cell_type,content_id,metadata,respawn_seconds,is_active
realm_qingmu_valley,1015,1015,INHERITANCE_ALTAR,altar_qingmu_elder,"{""altar_name"":""Đài Truyền Thừa Thanh Mộc"",...}",0,TRUE
```
