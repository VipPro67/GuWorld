# Báo cáo Nghiên Cứu Chi Tiết: Rà Soát Tồn Đọng Mộng Cảnh & Chuỗi Văn Bản Nhầm Lẫn Bí Cảnh (Issue #232)

> **Mã vé:** [#232](https://github.com/VipPro67/GuWorld/issues/232)  
> **Người thực hiện:** Research Subagent (Dream Realm Separation Researcher)  
> **Phạm vi khảo sát:** Backend (`handlers`, `db`, `server`), Frontend (`components`, `store`, `app`), Data CSV (`DreamRealm.csv`), Database Schema (`schema_world.go`, `schema_player_items.go`).

---

## 1. Tổng Quan & Hiện Trạng Gộp Domain (Executive Summary)

Hiện tại, hệ thống tồn tại sự **lai tạp và lẫn lộn nghiêm trọng** giữa ba khái niệm hoàn toàn khác biệt theo chuẩn lore *Cổ Chân Nhân*:
1. **Mộng Cảnh (Dream Realm):** Trải nghiệm cốt truyện lựa chọn Q&A, không có tọa độ vật lý, tiêu hao **Hồn Lực (Soul HP)**, sử dụng Cổ Giải Mộng, nhận **Chân Ý Lưu Phái (Path Attainment)**.
2. **Bí Cảnh Bản Đồ Không Gian (Spatial Secret Realm):** Không gian thế giới riêng biệt có lối vào/ra vật lý (`ENTRY_PORTAL`/`EXIT_PORTAL`), di chuyển bằng tọa độ (X, Y), chiến đấu quái vật thời gian thực, nhận **Đạo Ngân (Dao Marks)**.
3. **Phó Bản / Truyền Thừa Dạng Stage Cũ (Legacy Instances / Demonic Inheritance):** Ví dụ `dungeon_flower_wine` hay 3 bí cảnh text cũ (`secret_realm_sword_mountain`, `secret_realm_fire_cave`, `secret_realm_water_palace`).

Cả ba hệ thống này đang bị **gộp chung** vào cùng một bảng DB (`instance_templates`), cùng bảng session (`player_instance_sessions`), cùng file xử lý (`instance.go`), cùng tiền tố endpoint (`/api/instance/*`), và cùng một component UI frontend mang tên ngoại lai là `DungeonView.tsx`.

---

## 2. Rà Soát Chi Tiết Backend (Trích Dẫn File & Dòng Cụ Thể)

### 2.1. `backend/internal/handlers/instance.go`
* **Dòng 22:** `const MaxDailyInstanceEntries = 5` — Giới hạn lượt đi gộp chung cho cả Mộng Cảnh lẫn Bí Cảnh stage cũ.
* **Dòng 105:** `Type string // "INHERITANCE_RIGHTEOUS", "INHERITANCE_DEMONIC", "DREAM_REALM", "SECRET_REALM"` — Khai báo struct `InstanceTemplate` gom tất cả vào làm một.
* **Dòng 156–177 (`parseHPEffects`):**
  - Tách Soul HP và HP bằng cách parse chuỗi Regex thô sơ: `(\d+)\s+Soul\s+HP` và `(\d+)\s+HP`.
* **Dòng 250–333 (`ListInstances`):**
  - Dòng 268: `WHERE type <> 'SPATIAL_REALM'` — Loại trừ Bí Cảnh Không Gian nhưng vẫn gom Mộng Cảnh (`DREAM_REALM`) và Bí Cảnh cũ (`SECRET_REALM`, `INHERITANCE_*`) vào một danh sách trả về qua `/api/instance/list`.
  - Dòng 320–326: Đếm chung giới hạn hàng ngày: `AND it.type IN ('DREAM_REALM', 'SECRET_REALM')`.
* **Dòng 336–512 (`EnterInstance`):**
  - Dòng 348–350: Đã phải viết rào chắn né Spatial: `if template.Type == "SPATIAL_REALM" { c.JSON(http.StatusBadRequest, gin.H{"error": "Bí cảnh bản đồ dùng API spatial riêng"}) }`.
  - Dòng 368: Dùng sai thuật ngữ: `"Đạo hữu đang thám hiểm một phó bản khác!"`.
  - Dòng 386: Dùng sai thuật ngữ: `"Không tìm thấy phó bản yêu cầu!"`.
  - Dòng 410: Dùng sai thuật ngữ: `"Đạo hữu đã hết lượt thám hiểm phó bản hôm nay (tối đa %d lượt/ngày)!"`.
  - Dòng 427: `"vào lại phó bản này!"`.
  - Dòng 445: `"vào phó bản này!"`.
  - Dòng 449–452: Logic nhánh riêng của Mộng Cảnh bị nhồi vào handler chung:
    ```go
    if template.Type == "DREAM_REALM" && player.SoulHP <= 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Linh hồn đạo hữu đang ở trạng thái ngủ say / hôn mê, không thể thám hiểm mộng cảnh!"})
    }
    ```
  - Dòng 496: Audit log nhầm lẫn: `RequestSummary: fmt.Sprintf("Tiến vào bí cảnh %s | Tiêu hao vé vào: 1x %s", template.Name, template.EntryItem)` — Khi vào Mộng Cảnh lại ghi là "Tiến vào bí cảnh".
  - Dòng 508: Dùng từ "phó bản": `"Tiến vào phó bản \"" + template.Name + "\" thành công!"`.
* **Dòng 515–921 (`ActionInstance`):**
  - Dòng 548: `"Không có phó bản nào đang hoạt động!"`.
  - Dòng 595: `"Không tìm thấy giai đoạn hiện tại của phó bản!"`.
  - Dòng 700: `"Đạo hữu bị thương nặng dẫn đến thám hiểm phó bản thất bại!"` (Dành cho HP thông thường).
  - Dòng 708: `"Hồn phách của đạo hữu kiệt quệ hoàn toàn rơi vào trạng thái hôn mê thảm hại! Thám hiểm mộng cảnh thất bại."` (Dành riêng cho Mộng Cảnh).
  - Dòng 742: `"Phó bản thất bại! " + logResult`.
  - Dòng 788: Audit log nhầm lẫn: `RequestSummary: fmt.Sprintf("Kỳ ngộ bí cảnh %s: %s | Tiêu hao: %s", template.Name, option.Text, string(reqBytes))` — Gọi Mộng Cảnh là "Kỳ ngộ bí cảnh".
  - Dòng 823–846: Xử lý phần thưởng gộp:
    - Nhánh 1 (Mộng Cảnh): `if rewards.Attainment != nil && rewards.Attainment.Path != ""` -> Cộng Chân Ý vào `player.PathMasteries`.
    - Nhánh 2 (Bí Cảnh cũ): `else if len(rewards.DaoMarks) > 0` -> Gọi `addMultipleDaoMarksTx` cộng Đạo Ngân.
    - Nhánh 3: Fallback "chinh phục thành công phó bản này!".
  - Dòng 880: `"Phó bản hoàn thành! " + logResult`.

### 2.2. `backend/internal/handlers/spatial_realms.go`
* **Dòng 33–35:** Comment chính thức chỉ ra điểm bất cập:
  `// ListSpatialRealms exposes walkable shared realms independently from`
  `// choice-based DREAM_REALM and legacy stage-based SECRET_REALM templates.`
* **Dòng 41, 89, 157:** Bản thân hệ thống Bí Cảnh Không Gian (`SPATIAL_REALM`) vẫn phải dựa dẫm vào bảng `instance_templates` chung chạ (`FROM instance_templates t WHERE t.type='SPATIAL_REALM'`).
* **Dòng 146–153:** Xung đột session giữa hai domain:
  ```go
  if err := tx.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM player_instance_sessions WHERE player_id=$1 AND status='ACTIVE')`, playerID).Scan(&hasStageSession); ...
  if hasStageSession {
      c.JSON(http.StatusConflict, gin.H{"error": "Đạo hữu đang ở một phó bản khác"})
      return
  }
  ```
  -> Khóa chéo: Người chơi đang đọc cốt truyện Mộng Cảnh thì bị chặn không vào được Bí Cảnh Bản Đồ và nhận thông báo "đang ở một phó bản khác".

### 2.3. `backend/internal/handlers/admin_player_instance_sessions.go`
* **Dòng 34, 100, 137:** Quản lý bảng `player_instance_sessions`.
* **Dòng 36, 48, 77, 83, 103, 110, 115, 119, 139, 144, 148:** Toàn bộ thông báo lỗi và log của Admin đều gọi các session stage này là "phiên bí cảnh":
  - `"Không thể tải phiên bí cảnh"`, `"Không thể đọc phiên bí cảnh"`, `"Dữ liệu phiên bí cảnh không hợp lệ"`, `"Không tìm thấy phiên bí cảnh"`, `"Đã cập nhật phiên bí cảnh"`, `"Đã xóa phiên bí cảnh"`.
  - Trong thực tế, các phiên này gần như 100% là phiên Mộng Cảnh của người chơi!

### 2.4. `backend/internal/handlers/admin_world_extended.go`
* **Dòng 320–450:** Tên hàm API Admin: `AdminListDungeons`, `AdminCreateDungeon`, `AdminUpdateDungeon`, `AdminDeleteDungeon`.
* Các câu thông báo dùng từ "phụ bản" / "dungeon": `"Không thể đọc danh sách phụ bản"`, `"ID và Tên phụ bản là bắt buộc"`, `"Loại, rank hoặc thời gian hồi phụ bản không hợp lệ"`, `"Đã tạo phụ bản mới"`, `"Đã cập nhật phụ bản"`, `"Đã xóa phụ bản"`.

### 2.5. `backend/internal/server/server.go`
* **Dòng 1032–1046:**
  Tất cả được gộp chung dưới router group `/api/instance`:
  - `/api/instance/active` (dùng cho Mộng Cảnh & stage instance)
  - `/api/instance/list` (dùng cho Mộng Cảnh & stage instance)
  - `/api/instance/enter` (tiến vào Mộng Cảnh & stage instance)
  - `/api/instance/action` (chọn bước Mộng Cảnh & stage instance)
  - `/api/instance/spatial` (Bí Cảnh Không Gian)
  - `/api/instance/spatial/active` (Bí Cảnh Không Gian đang vào)
  - `/api/instance/spatial/enter` (vào Bí Cảnh Không Gian)
  - `/api/instance/spatial/exit` (thoát Bí Cảnh Không Gian)
* **Dòng 86:** Admin CRUD: `/api/admin/dungeons` (tác động trực tiếp vào `instance_templates`).
* **Dòng 92–93:** Admin session: `/api/players/:id/instance-sessions/:session_id`.

### 2.6. `backend/internal/db/db.go`
* **Dòng 2830–2863 (`buildDreamRealmMapData`):**
  - **Lỗi ngữ nghĩa nghiêm trọng:** Khi sinh stage mặc định cho Mộng Cảnh nếu cột `map_data` rỗng, backend tạo nội dung:
    - `"Đạo hữu lần theo cổ địa đồ tìm vào bí cảnh của %s. Không khí âm u nặng nề, vô số cấm chế cổ xưa đang ẩn nấp trong bóng tối:"`
    - `"Rút lui khỏi bí cảnh ngay lập tức (Thất bại)"`
    - `"Cổ điện cuối cùng hiện ra, cơ duyên của %s nằm trong bảo rương cổ xưa giữa đại điện: Mở bảo rương nhận cơ duyên và rời bí cảnh"`
    -> Hàm sinh Mộng Cảnh nhưng nội dung 100% là text đi "bí cảnh", "mở bảo rương", "cổ địa đồ"!
* **Dòng 2865–2935 (`seedDreamRealmsFromCSV`):**
  - Đọc `DreamRealm.csv` và `INSERT INTO instance_templates`.
* **Dòng 2950–3006:** Seed hardcoded `dungeon_flower_wine` vào `instance_templates`.
* **Dòng 3016–3072:** Seed 3 bí cảnh text `secret_realm_sword_mountain`, `secret_realm_fire_cave`, `secret_realm_water_palace` (`type='SECRET_REALM'`) vào `instance_templates`.
* **Dòng 3520:** Export Mộng Cảnh: `SELECT ... FROM instance_templates WHERE type='DREAM_REALM' ORDER BY id`.

### 2.7. `backend/internal/db/import_workflow.go`
* **Dòng 86:**
  `{"dream_realm", "DreamRealm.csv", "instance_templates", true, false, []string{"id"}}`
  -> Giao diện Admin hiển thị bảng "Mộng cảnh", nhưng đích đến lưu trữ ngầm lại là `instance_templates`.

### 2.8. `backend/internal/db/schema_world.go` & `schema_player_items.go`
* **`schema_world.go:208–220` (`instance_templates`):**
  - Cột `type`: chứa cả `'DREAM_REALM'`, `'SECRET_REALM'`, `'SPATIAL_REALM'`, `'INHERITANCE_DEMONIC'`, `'INHERITANCE_RIGHTEOUS'`.
  - Cột `map_data JSONB`: chứa cấu trúc cây stage Q&A của Mộng Cảnh.
  - Cột `spatial_region_id` và `duration_minutes`: dùng riêng cho Bí Cảnh Không Gian.
* **`schema_world.go:224–238` (`shared_spatial_instances`):**
  - `template_id VARCHAR(50) NOT NULL REFERENCES instance_templates(id)` — Buộc Bí Cảnh Không Gian phụ thuộc khóa ngoại vào bảng chung `instance_templates`.
* **`schema_player_items.go:167–184` (`player_instance_sessions`):**
  - Lưu trữ trạng thái bước stage (`current_stage`, `temp_inventory`) của Mộng Cảnh và stage instance cũ.

---

## 3. Rà Soát Chi Tiết Frontend (Trích Dẫn File & Dòng Cụ Thể)

### 3.1. `frontend/src/components/DungeonView.tsx`
* **Tên file:** `DungeonView.tsx` — Sử dụng thuật ngữ RPG Tây phương ("Dungeon").
* **Dòng 20–25:**
  ```typescript
  const TYPE_LABELS: Record<string, { label: string; dream: boolean }> = {
    DREAM_REALM: { label: "Mộng Cảnh", dream: true },
    SECRET_REALM: { label: "Bí Cảnh", dream: false },
    INHERITANCE_DEMONIC: { label: "Truyền Thừa Ma Đạo", dream: false },
    INHERITANCE_RIGHTEOUS: { label: "Truyền Thừa Chính Đạo", dream: false },
  };
  ```
* **Dòng 83:** `const activeTag = activeType ? activeType.label : "Phó Bản";` — Fallback là "Phó Bản".
* **Dòng 92–93 (Header tiêu đề):**
  ```tsx
  <h1 className="text-lg font-serif font-bold text-zinc-100">Bí Cảnh & Mộng Cảnh</h1>
  <p className="text-xs text-zinc-500 font-mono">Thám hiểm nhận Chân Ý tăng cảnh giới</p>
  ```
  -> Gộp chung tiêu đề "Bí Cảnh & Mộng Cảnh", nhưng dòng mô tả lại ghi "nhận Chân Ý tăng cảnh giới" (chỉ đúng với Mộng Cảnh; Bí Cảnh cho Đạo Ngân).
* **Dòng 98:** `Lượt hôm nay: {instanceDailyEntriesUsed ?? 0}/{instanceDailyEntriesMax ?? 2}` — Giới hạn lượt đi gộp chung.
* **Dòng 190:** `"Đang tải phó bản..."`
* **Dòng 196:** `"⚠️ Đạo hữu đã sử dụng hết lượt thám hiểm phó bản hôm nay..."`
* **Dòng 206:** `const isInstanceDailyLimited = (dg.type === "DREAM_REALM" || dg.type === "SECRET_REALM") && isDailyLimitReached;`
* **Dòng 249–277:** Khung tóm tắt phần thưởng hiển thị hỗn hợp cả Chân Ý (`attainment`), Đạo Ngân (`dao_marks`), Nguyên Liệu, Nguyên Thạch trong cùng danh sách.
* **Dòng 291:** Nút `Bắt Đầu Thám Hiểm` (gọi `enterInstance`).

### 3.2. `frontend/src/components/LobbyView.tsx`
* **Dòng 290–298:**
  ```typescript
  {
    id: "dungeon",
    label: "Mộng Cảnh & Bí Cảnh",
    icon: Gem,
    gradient: "from-purple-500 to-fuchsia-600",
    glow: "group-hover:shadow-[0_0_25px_rgba(168,85,247,0.45)]",
    borderColor: "border-purple-400/30",
    iconColor: "text-purple-500",
  }
  ```
  -> Tại sảnh chính (Lobby), card điều hướng gộp "Mộng Cảnh & Bí Cảnh" dưới ID `dungeon`.

### 3.3. `frontend/src/components/WorldPage.tsx`
* **Dòng 41:** Dynamic import `DungeonView`.
* **Dòng 59:** Khai báo Cell Type trên World Map:
  `ENTRY_PORTAL: { icon: "🚪", label: "Cổng Bí Cảnh", color: "text-amber-300", ... }` — Dành cho Bí Cảnh Không Gian thực tế.
* **Dòng 214:** Menu khám phá:
  `{ id: "dungeon", label: "Bí Cảnh", icon: Compass, color: "text-indigo-400" },`
  -> **Sai lệch nghiêm trọng:** Đặt nhãn nút là "Bí Cảnh", nhưng khi bấm vào lại mở modal `DungeonView` (vốn chứa toàn bộ 49 Mộng Cảnh text Q&A!).
* **Dòng 245:** `itemIds: ["dungeon", "mine", ...]`
* **Dòng 436:** `{activeModal === "dungeon" && <DungeonView />}`.

### 3.4. `frontend/src/app/page.tsx`
* **Dòng 41:** Import dynamic `DungeonView`.
* **Dòng 325:** Map tiêu đề header: `aperture: "Tiên khiếu", dungeon: "Bí cảnh", sect: "Tông môn",` — Gán nhãn `dungeon` là "Bí cảnh".
* **Dòng 562–565:** Render `currentView === "dungeon" ? <DungeonView /> : ...`.

### 3.5. `frontend/src/store/slices/instanceSlice.ts`
* Tên slice: `InstanceSlice` quản lý các hàm: `fetchInstances`, `fetchActiveInstance`, `enterInstance`, `actionInstance`.
* **Dòng 33, 57, 84, 113:** Gọi các API `/api/instance/*`.
* **Dòng 75, 91, 96, 97, 120, 133:** Toàn bộ thông báo toast và log đều dùng từ "phó bản" ("Lỗi tải phó bản.", "Không thể tiến vào phó bản.", "Tiến vào phó bản thành công!", "Hành động phó bản thất bại.", "Đã hành động trong phó bản.").

### 3.6. `frontend/src/store/useGameStore.ts`
* **Dòng 812–856:** Các interface mang tên chung `InstanceOption`, `InstanceStage`, `InstanceTemplate`, `InstanceListItem`, `PlayerInstanceSession`.
* **Dòng 1695:** Tích hợp `createInstanceSlice` vào GameStore gốc.

### 3.7. `frontend/src/components/admin/WorldExtendedAdminTab.tsx`
* **Dòng 10, 28, 62, 106, 111, 131:** Tab quản trị dùng từ `"dungeons"`, tiêu đề `"Phụ Bản / Dungeons"`, trỏ tới `/api/admin/dungeons`.

### 3.8. Các tài liệu Wiki trong frontend (`roadmap.tsx`, `equipment-jobs.tsx`, `character.tsx`)
* `roadmap.tsx:174`: `"Đã có phó bản Mộng Cảnh..."` -> Thuật ngữ game lai tạp.
* `equipment-jobs.tsx:118`: `keywords: [..., "phó bản", "mộng cảnh", "dungeon", "dream realm"]`.

---

## 4. Dữ Liệu `DreamRealm.csv` Hiện Tại

* Vị trí: `backend/data/DreamRealm.csv`.
* Số lượng: 49 hàng dữ liệu, bao gồm `dream_realm_star_constellation` (Mộng Cảnh Tinh Túc Tiên Tôn) và 48 Mộng Cảnh cho 48 Lưu phái (`dream_realm_<path>_path` hoặc `dream_realm_<path>`).
* Cột hiện tại: `id, name, path, type, required_rank, entry_item, description, rewards, map_data, cooldown_hours`.
* **Đặc tính dữ liệu:**
  1. Cột `rewards`: Chứa `attainment` (`{"path": "Fire Path", "amount": 4000}`), `guts_gu` (Can Đảm Cổ), `dream_path_material`, `primeval_stones`, Cổ Mộng Đạo (`GU_DREAM_UNRAVEL_5`).
  2. Cột `map_data`: Chứa mảng JSON 50 tầng/giai đoạn (từ Khởi Đầu, Nghịch Cảnh, Lĩnh Ngộ, Đại Chiến & Thiên Kiếp, Đốn Ngộ Chân Ý đến Hoàn Thành Viên Mãn).
  3. Cơ chế tương tác trong stage: Tiêu hao `Soul HP` (15, 20, 25, 30, 35 Soul HP), yêu cầu `min_soul_hp`, kích hoạt Giải Mộng Cổ (`GU_DREAM_UNRAVEL_5`).
  4. Thuộc tính `type`: Đang mang giá trị `"DREAM_REALM"`, nhưng đang bị ép nhập vào bảng `instance_templates`.

---

## 5. Đề Xuất Kế Hoạch Chuyển Giao Sang Domain Mộng Cảnh Chuyên Biệt

### 5.1. Bảng Dữ Liệu Chuyên Biệt (Database Schema)
1. **Tách bảng mẫu Mộng Cảnh:** `dream_realm_templates` (thay vì để chung trong `instance_templates`):
   ```sql
   CREATE TABLE IF NOT EXISTS dream_realm_templates (
       id VARCHAR(50) PRIMARY KEY,
       name VARCHAR(100) NOT NULL,
       path VARCHAR(50) NOT NULL,
       required_rank INTEGER NOT NULL DEFAULT 1,
       entry_item VARCHAR(50) NOT NULL DEFAULT 'dream_path_material',
       description TEXT NOT NULL,
       rewards JSONB NOT NULL DEFAULT '{}',
       stages JSONB NOT NULL DEFAULT '[]', -- Thay thế tên cột map_data thành stages
       cooldown_hours INTEGER NOT NULL DEFAULT 24
   );
   ```
2. **Tách bảng phiên thám hiểm Mộng Cảnh:** `player_dream_sessions` (thay vì để chung trong `player_instance_sessions`):
   ```sql
   CREATE TABLE IF NOT EXISTS player_dream_sessions (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
       dream_realm_id VARCHAR(50) NOT NULL REFERENCES dream_realm_templates(id) ON DELETE CASCADE,
       current_stage INTEGER NOT NULL DEFAULT 0,
       status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'COMPLETED', 'FAILED')),
       created_at TIMESTAMP NOT NULL DEFAULT NOW()
   );
   CREATE UNIQUE INDEX IF NOT EXISTS idx_player_active_dream_session ON player_dream_sessions (player_id) WHERE status = 'ACTIVE';
   ```
3. **Giải phóng bảng `instance_templates`:**
   - Chỉ giữ lại phục vụ cho **Bí Cảnh Không Gian (`SPATIAL_REALM`)** và **Truyền Thừa Không Gian (`SPATIAL_INHERITANCE`)** có tọa độ và bản đồ thật. Loại bỏ toàn bộ `type='DREAM_REALM'`.

### 5.2. Chuẩn Hóa API Endpoints
* **Domain Mộng Cảnh (Chuyên biệt):**
  - `GET  /api/dream-realms` — Lấy danh sách Mộng Cảnh khả dụng và thời gian hồi chiêu từng mộng cảnh.
  - `GET  /api/dream-realms/active` — Lấy trạng thái phiên Mộng Cảnh đang khám phá.
  - `POST /api/dream-realms/enter` — Tiến vào Mộng Cảnh (tiêu hao vật phẩm mộng đạo, kiểm tra Hồn Lực).
  - `POST /api/dream-realms/action` — Lựa chọn phương án trong Mộng Cảnh (tiêu hao Hồn Lực, dùng Giải Mộng Cổ, hoàn thành nhận Chân Ý).
* **Domain Bí Cảnh Không Gian (Giữ riêng biệt):**
  - Giữ và nâng cấp nhóm `/api/instance/spatial/*` (hoặc chuyển thành `/api/spatial-realms/*`).

### 5.3. Chuẩn Hóa Giao Diện & Điều Hướng Frontend
1. **Tại `LobbyView.tsx`:**
   - Card 1: **Mộng Cảnh (Dream Realm)** — Icon `Sparkles`/`BookOpen`, màu tím/fuchsia đặc trưng, bấm vào mở giao diện thám hiểm Mộng Cảnh & hấp thu Chân Ý.
   - Card 2: **Bí Cảnh Không Gian (Secret Realm)** — Icon `Compass`/`DoorOpen`, màu vàng hổ phách, hiển thị thông tin các Cổng Bí Cảnh trên thế giới.
2. **Tại `WorldPage.tsx`:**
   - Nút menu Khám phá: Sửa nhãn `"Bí Cảnh"` (mở DungeonView cũ) thành `"Mộng Cảnh"` (hoặc chuyển Mộng Cảnh vào danh mục Thức Hải / Tu Luyện của nhân vật vì Mộng Cảnh không phụ thuộc vào vị trí vật lý trên bản đồ).
   - Ô `ENTRY_PORTAL` trên World Map: Giữ nguyên là "Cổng Bí Cảnh", dẫn vào Bí Cảnh Không Gian chung/riêng.
3. **Component UI mới:**
   - Đổi tên `DungeonView.tsx` thành `DreamRealmView.tsx` (loại bỏ hoàn toàn các nhãn "Phó bản", "Bí cảnh", "Dungeon").
   - Giao diện Mộng Cảnh hiển thị rõ: Thanh Hồn Lực (Soul HP), Sát chiêu Giải Mộng, Cảnh Giới Lưu Phái & Chân Ý thu hoạch.

### 5.4. Sửa Đổi Chuỗi Văn Bản Nhầm Lẫn (Terminology Cleanup)
* Xóa bỏ hoàn toàn các từ: `"phó bản"`, `"phụ bản"`, `"dungeon"`.
* Trong `backend/internal/db/db.go` (`buildDreamRealmMapData`): Sửa ngay các câu dẫn `"tìm vào bí cảnh"`, `"rời bí cảnh"`, `"rút lui khỏi bí cảnh"` thành `"tiến vào Mộng Cảnh"`, `"rút lui khỏi Mộng Cảnh"`, `"thức tỉnh khỏi Mộng Cảnh"`.
* Audit log: Đổi `instance.enter` thành `dream_realm.enter` với text `"Thám hiểm Mộng Cảnh %s"`.
