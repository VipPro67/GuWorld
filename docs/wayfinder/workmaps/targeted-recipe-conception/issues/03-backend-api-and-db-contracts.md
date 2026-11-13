Type: research
Status: resolved
Blocked by: 01, 02

## Question

Thiết kế hợp đồng API backend và tối ưu truy vấn Database:
1. `GET /api/refine/conceive/candidates`: API trả về danh sách các Cổ có thể suy tính (lọc theo Path, Rank, loại bỏ các Cổ người chơi đã sở hữu bí phương 100%, kèm theo thông tin chi phí Niệm Lực và Nguyên Thạch dự kiến).
2. `POST /api/refine/conceive`: API thực thi khai sáng suy tính. Nhận `target_gu_id`, `target_rank`, `catalyst_gu_instance_id` (optional), `catalyst_item_id` (optional).
3. Luồng Database Transaction: Kiểm tra điều kiện (Niệm Lực, Nguyên Thạch, Cổ phôi) -> Thực thi tính toán -> Trừ tài nguyên / Hủy Cổ phôi -> Upsert vào `player_recipe_progress` (hoặc xử lý phản phệ nếu thất bại) -> Ghi `auditlog` và gửi `PublishPlayerSnapshot`.

## Answer

### 1. Endpoint Tra Cứu Cổ Khả Dụng (`GET /api/refine/conceive/candidates`)
- **Đường dẫn**: `GET /api/refine/conceive/candidates`
- **Quyền**: `AuthRequired()`, `RequireOnboarded()`, `RateLimitByPlayer(60, time.Minute)`
- **Query Params**:
  - `path` (string, optional): lọc theo lưu phái (vd: `sword`, `wisdom`, `blood`).
  - `rank` (int, optional): lọc theo rank (1-9).
  - `search` (string, optional): tìm kiếm theo tên hoặc id.
  - `limit` (int, default 50, max 100), `offset` (int, default 0).
- **Tối ưu truy vấn Database**:
  ```sql
  SELECT 
      g.id, g.rank, g.name, g.path, g.image_url, g.description,
      COALESCE(p.completion_rate, 0) as existing_rate,
      COALESCE(p.is_unlocked, false) as is_unlocked
  FROM gu_dictionary g
  LEFT JOIN player_recipe_progress p 
      ON p.player_id = $1 
      AND p.recipe_id = ('recipe_' || g.id || '_' || g.rank::text || '_1')
  WHERE ($2 = '' OR g.path::text ILIKE '%' || $2 || '%')
    AND ($3 = 0 OR g.rank = $3)
    AND ($4 = '' OR g.name ILIKE '%' || $4 || '%')
    AND (p.completion_rate IS NULL OR p.completion_rate < 100.0)
  ORDER BY g.rank ASC, g.id ASC
  LIMIT $5 OFFSET $6;
  ```
- **Xử lý dữ liệu trả về**: Duyệt danh sách và tính toán nhanh `required_thought`, `required_stones`, `estimated_accuracy` và cảnh báo `has_backlash_risk` tương ứng với tu vi người chơi hiện tại để frontend render trực tiếp.

### 2. Endpoint Khai Sáng Bí Phương (`POST /api/refine/conceive`)
- **Đường dẫn**: `POST /api/refine/conceive`
- **Request Body DTO**:
  ```go
  type ConceiveRecipeRequest struct {
      TargetGuID           string `json:"target_gu_id" binding:"required"`
      TargetRank           int    `json:"target_rank" binding:"required,min=1,max=9"`
      CatalystGuInstanceID string `json:"catalyst_gu_instance_id,omitempty"`
      CatalystItemID       string `json:"catalyst_item_id,omitempty"`
  }
  ```
- **Response DTO**:
  ```go
  type ConceiveRecipeResponse struct {
      Status           string          `json:"status"` // "SUCCESS" | "DEVIATED" | "BACKLASH"
      IsDeviated       bool            `json:"is_deviated"`
      RecipeID         string          `json:"recipe_id,omitempty"`
      GuID             string          `json:"gu_id,omitempty"`
      GuName           string          `json:"gu_name,omitempty"`
      Rank             int             `json:"rank,omitempty"`
      CompletionRate   float64         `json:"completion_rate"`
      IsInstant100     bool            `json:"is_instant_100"`
      Message          string          `json:"message"`
      CatalystConsumed string          `json:"catalyst_consumed,omitempty"`
      ThoughtCost      float64         `json:"thought_cost"`
      StonesCost       float64         `json:"stones_cost"`
      Backlash         *BacklashDetail `json:"backlash,omitempty"`
  }
  ```

### 3. Quy Trình Database Transaction & State Mutation
1. **Khởi tạo Transaction**: `tx, err := db.DB.Begin()`, khóa và tải người chơi qua `evaluateManaTx(tx, authID)`.
2. **Kiểm tra Hồi Chiêu & Điều Kiện Sàn**:
   - Kiểm tra trạng thái Chấn Thương Hồn Phách (`wisdom_backlash_soul_injury`).
   - Kiểm tra `player.Rank <= 5` với `targetRank >= 6` $\implies$ chặn (400 Bad Request).
   - Kiểm tra số dư Niệm Lực `player.CurrentThought >= thoughtCost` và Nguyên Thạch.
3. **Xử lý Vật Dẫn (Catalyst)**:
   - Nếu có `CatalystGuInstanceID`: xác thực Cổ phôi trong `player.GuInventory` (chưa trang bị, không bản mệnh, không khóa, chưa chết, cùng hệ, rank <= targetRank), xóa Cổ khỏi Không Khiếu và xóa trong DB `player_gus`.
   - Nếu có `CatalystItemID`: xác thực số lượng nguyên liệu trong `player.MaterialsInventory`, trừ số lượng và cập nhật `player_inventory_items`.
4. **Kiểm tra Phản Phệ (Trí Đạo Phản Phệ)**:
   - Tính toán nguy cơ phản phệ theo vé 1.
   - Nếu nổ phản phệ:
     - `player.CurrentThought = 0`
     - `player.CurrentMana *= 0.5`
     - `player.CurrentHP = math.Max(1.0, player.CurrentHP * 0.5)`
     - Nếu Tiên Cổ: gắn cooldown 20 phút.
     - Cập nhật DB `players`, commit, xuất auditlog, publish snapshot và trả response `BACKLASH`.
5. **Tính toán Tỉ lệ Trúng Đích & Lệch Hướng**:
   - Nếu trúng đích: ghi nhận Cổ đích.
   - Nếu không trúng: chạy câu lệnh tìm Cổ thay thế cùng hệ trong `gu_dictionary` có rank bằng hoặc rank-1 chưa hoàn thiện 100%.
6. **Cập nhật Tiến Độ Tàn Phương (`player_recipe_progress`)**:
   - Tính `initialRate` (15% - 35%) hoặc 100% nếu nổ `Inspiration Flash`.
   - Gọi `RecipeSvc.Upsert(tx, &services.RecipeProgress{...})`.
7. **Commit & Phát Sự Kiện**:
   - Commit transaction.
   - `ClearPendingPlayerQueues(player.ID.String())`.
   - `PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, ItemInventoryDomain, RecipesDomain)`.
   - Ghi nhật ký hệ thống qua `auditlog.Log`.

