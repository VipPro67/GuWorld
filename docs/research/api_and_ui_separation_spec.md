# Đặc Tả Kỹ Thuật: Chuẩn Hóa Phân Tách API & Kiến Trúc Giao Diện (Issue #235)

> **Mã vé:** [#235 Decision](https://github.com/VipPro67/GuWorld/issues/235)  
> **Trạng thái:** Resolved (Đã chốt thiết kế)  
> **Tài liệu liên quan:** `docs/research/dream_realm_separation_audit.md`, `docs/research/spatial_realms_hybrid_engine_audit.md`, `docs/research/spatial_realm_rules_and_altar_spec.md`

---

## 1. Chuẩn Hóa Tuyến API Backend

Hệ thống phân tách triệt để thành 2 Route Group độc lập trong Gin Engine (`backend/internal/server/server.go`):

### 1.1. Domain Mộng Cảnh: `/api/dream-realms`
```go
dream := api.Group("/dream-realms")
dream.Use(middleware.AuthRequired())
dream.Use(middleware.RequireOnboarded())
dream.Use(middleware.RateLimitByPlayer(30, time.Minute))
{
    dream.GET("", handlers.ListDreamRealms)
    dream.GET("/active", handlers.GetActiveDreamSession)
    dream.POST("/enter", handlers.EnterDreamRealm)
    dream.POST("/action", handlers.ActionDreamRealm)
}
```

* **`GET /api/dream-realms`**:
  - Trả về: `{ "realms": [DreamRealmItem], "daily_used": int, "daily_max": int }`.
  - Không còn chứa bất kỳ bản ghi bí cảnh nào; chỉ trả 49 Mộng Cảnh lưu phái và thời gian hồi chiêu tương ứng.
* **`GET /api/dream-realms/active`**:
  - Trả về: `{ "active": bool, "session": PlayerDreamSession, "template": DreamRealmTemplate }`.
* **`POST /api/dream-realms/enter`**:
  - Payload: `{ "dream_realm_id": string }`.
  - Kiểm tra Hồn Lực `player.SoulHP > 0`, trừ vật phẩm mộng đạo `entry_item`, tạo phiên `player_dream_sessions`.
* **`POST /api/dream-realms/action`**:
  - Payload: `{ "option_index": int }`.
  - Trừ Hồn Lực theo kịch bản, kích hoạt Cổ Giải Mộng nếu chọn phương án yêu cầu, trả về `respondActionWithInventory`.
  - Khi hoàn thành: Cộng Chân Ý (`Attainment`) vào `player.PathMasteries`.

---

### 1.2. Domain Bí Cảnh & Truyền Thừa Không Gian: `/api/spatial-realms`
```go
spatial := api.Group("/spatial-realms")
spatial.Use(middleware.AuthRequired())
spatial.Use(middleware.RequireOnboarded())
spatial.Use(middleware.RateLimitByPlayer(30, time.Minute))
{
    spatial.GET("", handlers.ListSpatialRealms)
    spatial.GET("/active", handlers.GetActiveSpatialRealm)
    spatial.POST("/enter", handlers.EnterSpatialRealm)
    spatial.POST("/exit", handlers.ExitSpatialRealm)
    spatial.POST("/claim-altar", handlers.ClaimInheritanceAltar)
}
```

* **`GET /api/spatial-realms`**:
  - Trả về danh sách Bí cảnh & Truyền thừa khả dụng, tọa độ cổng vào trên thế giới, số người chơi đang có mặt.
* **`GET /api/spatial-realms/active`**:
  - Trả về thông tin phiên không gian người chơi đang ở (`template_id`, `name`, `rules`, `expires_at`, `player_count`).
* **`POST /api/spatial-realms/enter`**:
  - Payload: `{ "template_id": string, "scope": "SHARED" | "PARTY" | "SOLO" }`.
  - Kiểm tra người chơi đứng đúng ô `ENTRY_PORTAL`, kiểm tra tín vật, kiểm tra áp chế cảnh giới, khởi tạo hoặc tham gia phiên `spatial_instances`.
  - Nếu `scope == "PARTY"`: Đưa toàn bộ thành viên trong tổ đội vào cùng lúc.
* **`POST /api/spatial-realms/exit`**:
  - Rời bí cảnh khi đứng tại ô `EXIT_PORTAL`, trả người chơi về tọa độ thế giới ban đầu.
* **`POST /api/spatial-realms/claim-altar`**:
  - Payload: `{ "altar_id": string }`.
  - Kiểm tra điều kiện: Đứng cạnh ô `INHERITANCE_ALTAR`, Boss thủ hộ đã bị tiêu diệt, đạt yêu cầu Đạo Ngân / tu vi, kiểm tra bảng `player_inheritance_claims`, cấp phát di sản.

---

## 2. Kiến Trúc Điều Hướng Giao Diện Frontend

### 2.1. Sảnh Chính (`LobbyView.tsx`)
Tách thẻ `dungeon` cũ thành 2 thẻ rõ ràng:
1. **Thẻ Mộng Cảnh (Dream Realm)**:
   - ID: `"dream-realm"`
   - Tên: `"Mộng Cảnh"`
   - Mô tả: `"Thám hiểm tiềm thức tiền nhân · Lĩnh ngộ Chân Ý"`
   - Icon: `Sparkles` / `BookOpen`
   - Màu sắc: Tím huyền bí (`from-purple-600 to-indigo-700`)
   - Hành vi: Chuyển view sang `DreamRealmView` (thay thế hoàn toàn `DungeonView`).
2. **Thẻ Bí Cảnh & Truyền Thừa (Secret Realm & Inheritance)**:
   - ID: `"spatial-realms-directory"`
   - Tên: `"Bí Cảnh & Truyền Thừa"`
   - Mô tả: `"Tiểu giới Ngũ Vực · Cửa vào truyền thừa"`
   - Icon: `Compass` / `DoorOpen`
   - Màu sắc: Vàng hổ phách (`from-amber-500 to-yellow-600`)
   - Hành vi: Mở modal danh mục các Cổng Bí Cảnh trên thế giới (vị trí tọa độ, cảnh giới yêu cầu, quy tắc PK, nút "Dẫn đường đến cổng").

---

### 2.2. Giao Diện Mộng Cảnh Chuyên Biệt (`DreamRealmView.tsx`)
* Giao diện đọc kịch bản tình huống Q&A với thiết kế trang nhã, lore chuẩn:
  - Thanh hiển thị **Hồn Lực (Soul HP)** và Trạng thái Thần Hồn.
  - Cảnh báo lore: Tiêu hao Hồn Lực, nguy cơ hôn mê nếu về 0.
  - Hộp kịch bản chương hồi, các nút chọn quyết định.
  - Xóa bỏ hoàn toàn các từ "phó bản", "bí cảnh".

---

### 2.3. Trải Nghiệm Thế Giới & Thanh Bí Cảnh HUD (`SpatialRealmHUD.tsx`)
Khi `player.map_instance_id != null` (đang ở trong Bí cảnh):
* Xuất hiện thanh HUD cố định trên đầu màn hình:
  - Tên Bí cảnh / Truyền thừa + Huy hiệu Scope (`Chung Toàn Server` / `Tổ Đội`).
  - Đồng hồ đếm ngược: `Còn 45:12` (màu vàng khi < 10 phút, đỏ khi < 3 phút).
  - Huy hiệu Quy tắc: `Vùng An Toàn (Peaceful)` hoặc `Hỗn Chiến (Free PK)` / `Áp Chế Rank 3`.
  - Nút "Tìm Cổng Thoát" (nhấp nháy hướng mũi tên về ô `EXIT_PORTAL`).
* Khi đứng cạnh ô `INHERITANCE_ALTAR`:
  - Hiển thị Action Button phát sáng: `[🏛️ Khảo Nghiệm Đài Truyền Thừa]`.
  - Bấm vào mở `InheritanceAltarModal`: Hiển thị thông điệp tiền nhân, trạng thái Thủ Hộ Giả, yêu cầu Đạo Ngân, và nút "Tiếp Nhận Chân Truyền".

---

## 3. Tái Cấu Trúc Zustand State (`dreamRealmSlice.ts` & `spatialRealmSlice.ts`)

### 3.1. `dreamRealmSlice.ts`
```typescript
export interface DreamRealmSlice {
  activeDreamSession: PlayerDreamSession | null;
  activeDreamTemplate: DreamRealmTemplate | null;
  availableDreamRealms: DreamRealmListItem[];
  dreamDailyEntriesUsed: number;
  dreamDailyEntriesMax: number;
  fetchDreamRealms: () => Promise<void>;
  fetchActiveDreamRealm: () => Promise<void>;
  enterDreamRealm: (dreamRealmId: string) => Promise<boolean>;
  actionDreamRealm: (optionIndex: number) => Promise<boolean>;
}
```

### 3.2. `spatialRealmSlice.ts`
```typescript
export interface SpatialRealmSlice {
  activeSpatialRealm: SpatialRealmActiveInfo | null;
  availableSpatialRealms: SpatialRealmListItem[];
  fetchSpatialRealms: () => Promise<void>;
  fetchActiveSpatialRealm: () => Promise<void>;
  enterSpatialRealm: (templateId: string, scope?: "SHARED" | "PARTY" | "SOLO") => Promise<boolean>;
  exitSpatialRealm: () => Promise<boolean>;
  claimInheritanceAltar: (altarId: string) => Promise<boolean>;
}
```

* **Xóa bỏ vĩnh viễn `instanceSlice.ts`**.
* Thay thế toàn bộ các thông báo toast: `"Tiến vào Mộng Cảnh thành công!"`, `"Lĩnh ngộ Chân Ý thành công!"`, `"Đã bước vào Bí Cảnh!"`, `"Đã trở về từ Bí Cảnh!"`.
