# Khảo sát cơ chế tiếp nhận phản hồi và chặn revision trên Zustand store (Frontend)

- **Ticket liên quan:** #226 (thuộc Epic #224)
- **Ngày khảo sát:** 2026-10-18
- **Mục tiêu:** Phân tích luồng đồng bộ trạng thái nhân vật giữa Direct Action HTTP Response và Realtime WebSocket Event; chỉ ra nguyên nhân drop dữ liệu; rà soát các slice gọi ngầm gây race condition; đề xuất giải pháp `applyServerStateUpdate`.

---

## 1. Cơ chế `shouldApplyPlayerRevision` và hiện tượng nuốt (drop) phản hồi

### 1.1. Hiện trạng triển khai
Tại `frontend/src/store/playerRevision.ts:3-9`:
```typescript
export function shouldApplyPlayerRevision(
  appliedRevision: number | null,
  incomingRevision: number | undefined,
): boolean {
  if (incomingRevision === undefined) return true;
  return appliedRevision === null || incomingRevision > appliedRevision;
}
```

Hàm này được dùng để cổng chặn việc áp dụng state tại 3 vị trí then chốt:
1. `frontend/src/store/useGameStore.ts:1352` trong `setPlayerFromServer`.
2. `frontend/src/store/useGameStore.ts:1407` trong `applyInventoryFromResponse`.
3. `frontend/src/store/useGameStore.ts:1796` trong WebSocket listener sự kiện `player`.

### 1.2. Nguồn gốc của Race Condition
Tại Backend (`backend/internal/handlers/respond.go:57-73` và `backend/internal/handlers/player_events.go:110-127`):
Khi một action của người chơi được commit vào Postgres:
- Trigger DB hoặc query tăng `state_revision` lên giá trị mới $R$.
- Backend đồng thời phát sự kiện WebSocket (`PublishPlayerSnapshot` với `StateRevision = R`) và trả về HTTP response của chính request đó (`respondActionWithInventory` với `state_revision = R`).
- Cả hai kênh đều mang cùng một giá trị `state_revision = R`.

### 1.3. Cơ chế nuốt dữ liệu khi gặp Race Condition
Điều kiện kiểm tra của `shouldApplyPlayerRevision` là `incomingRevision > appliedRevision` (lớn hơn tuyệt đối):

- **Kịch bản A (WebSocket event đến trước):**
  WebSocket listener nhận gói tin có `state_revision = R`. Do $R > R_{hiện tại}$, gói tin được áp dụng và store cập nhật `playerStateRevision = R`.
  Ngay sau đó, direct HTTP response từ hành động (chứa Cổ mới nhặt, vật phẩm mua, v.v.) về tới client với `state_revision = R`.
  Hàm `applyInventoryFromResponse` kiểm tra `shouldApplyPlayerRevision(R, R)` $\rightarrow$ trả về `false` vì $R > R$ là sai.
  $\rightarrow$ **Direct HTTP response bị nuốt (drop) hoàn toàn!** Người chơi không nhận được vật phẩm trên UI.

- **Kịch bản B (HTTP response đến trước):**
  HTTP response đến trước, `applyInventoryFromResponse` cập nhật inventory và gán `playerStateRevision = R`.
  WebSocket event đến sau với `state_revision = R` mang thông tin cập nhật `current_mana`, `wall_progress`, `rank`, `sub_rank`.
  WebSocket listener kiểm tra `shouldApplyPlayerRevision(R, R)` $\rightarrow$ trả về `false`.
  $\rightarrow$ **WebSocket event bị nuốt!** Các chỉ số tu vi, chân nguyên, đột phá bị mất cập nhật trên giao diện người dùng.

---

## 2. Phân tích hàm `applyInventoryFromResponse`

Hàm được định nghĩa tại `frontend/src/store/useGameStore.ts:1403-1461`.

### 2.1. Các trường hiện đang xử lý
Chỉ gồm 10 trường: `gu_inventory`, `materials_inventory`, `primeval_stones`, `immortal_stones`, `human_qi`, `hp`, `current_thought`, `soul_hp`, `lifespan`, `recipes`.

### 2.2. Các trường quan trọng bị bỏ sót
- **Tu vi & Chân nguyên:** `current_mana`, `max_mana`, `rank`, `sub_rank`, `wall_progress`, `aperture_stress`, `aptitude_percent`, `aptitude_type`.
- **Hồi phục & Hồn phách:** `max_thought`, `thought_regen_rate`, `mana_regen_rate`, `hp_regen_rate`, `soul_regen_rate`, `max_soul`, `soul_tier`, `needs_soul_refine`.
- **Đạo quy luật & Lưu phái:** `dao_marks`, `path_masteries`, `time_speed_multiplier`.
- **Sinh lực nền & Chiến đấu:** `base_hp`, `active_combat`, `combat_settings`.
- **Trạng thái & Thể chất:** `statuses`, `body_mods`, `race`, `gender`, `constitution`.
- **Khám phá & Bản đồ:** `cleared_cells`, `visited_subzones`, `x`, `y`, `current_region`.
- **Bang hội & Phái:** `party_id`, `arena_rank`.

---

## 3. Rà soát các slice trong Store

### 3.1. Các slice dùng `applyInventoryFromResponse`
- `inventorySlice.ts`: 21 endpoints (ngoại trừ `pickupGroundDrop`).
- `cultivationSlice.ts`: `breakWall`, `refineGu`, `deduceRecipe`, `useRelicGu`, `ascension`, `completeQuest`, `redeemGiftCode`.
- `sectSlice.ts`: 8 endpoints (thăng chức, đóng góp, mua shop, nhận lương).
- `marketplaceSlice.ts`: 10 endpoints giao dịch bảo hoàng.
- `apertureSlice.ts`: 15 endpoints đạo ngân, tẩy thể, tai kiếp.
- `combatSlice.ts`: khám phá, bỏ chạy.
- `mineSlice.ts`, `gachaSlice.ts`, `petsSlice.ts`, `killerMovesSlice.ts`, `achievementsSlice.ts`, `instanceSlice.ts`.

### 3.2. Các slice gọi ngầm `fetchPlayerStatus` / `fetchGuInventory` gây race condition
- `marketplaceSlice.ts:124, 128, 153, 172, 175, 250, 282, 284, 312, 358, 360, 387, 389, 416, 418`: Vừa áp dụng response vừa gọi đồng thời `fetchGuInventory()` và `fetchPlayerStatus(pid, true)`.
- `arenaSlice.ts:110, 314`: Không dùng response data, gọi thẳng `fetchPlayerStatus()`.
- `martialArenaSlice.ts:191`: Gọi `fetchPlayerStatus()`.
- `apertureSlice.ts:253, 1075`: Gọi `fetchPlayerStatus(pid, true)`.
- `inventorySlice.ts:1010`: `pickupGroundDrop` gọi `fetchPlayerStatus()`.
- `instanceSlice.ts:134`: Gọi `refreshPlayerSummary()`.
- `tradeSlice.ts:196-197`: Gọi đồng thời `refreshPlayerSummary()` và `fetchGuInventory()`.

**Tác hại:**
1. `fetchPlayerStatus` và `refreshPlayerSummary` chỉ trả về **sparse Gu inventory** (chỉ Cổ trang bị), có thể ghi đè làm mất Cổ trong kho nếu response trả về chậm hơn các thao tác khác.
2. Tạo ra bão request dư thừa lên cơ sở dữ liệu.
3. Thứ tự xử lý mạng bất đồng bộ không được đảm bảo, dẫn đến nguy cơ ghi đè dữ liệu mới bằng dữ liệu cũ.

---

## 4. Đặc tả kỹ thuật giải pháp: `applyServerStateUpdate`

### 4.1. Điều chỉnh hàm `shouldApplyPlayerRevision`
Tại `frontend/src/store/playerRevision.ts`:
```typescript
export function shouldApplyPlayerRevision(
  appliedRevision: number | null,
  incomingRevision: number | undefined,
  options?: { allowEqual?: boolean }
): boolean {
  if (incomingRevision === undefined) return true;
  if (appliedRevision === null) return true;
  return options?.allowEqual
    ? incomingRevision >= appliedRevision
    : incomingRevision > appliedRevision;
}
```

### 4.2. Thiết kế hàm `applyServerStateUpdate`
Tại `frontend/src/store/useGameStore.ts`:
- Nhận payload linh hoạt: hỗ trợ cả flat payload và nested payload `{ player: ... }`.
- Cho phép `allowEqual: true` đối với direct action responses để không bao giờ bị nuốt phản hồi khi WebSocket event cùng revision đến trước.
- Cập nhật đầy đủ mọi trường trạng thái nhân vật:
  - Inventory: `gu_inventory`, `materials_inventory`, `equipped_gus`.
  - Tài nguyên: `primeval_stones`, `immortal_stones`, `human_qi`.
  - Chỉ số tu vi & sinh lực: `current_mana`, `max_mana`, `rank`, `sub_rank`, `wall_progress`, `aperture_stress`, `hp`, `base_hp`, `soul_hp`, `current_thought`.
  - Đạo ngân & Lưu phái: `dao_marks`, `path_masteries`.
  - Trạng thái & Cơ thể: `statuses`, `body_mods`, `race`, `constitution`.
  - Tiến độ: `unlocked_recipes`, `completed_quests`.
- Đồng bộ hóa các trạng thái tính toán phái sinh (`manaSnapshot`, `guInventoryLoaded`, `playerRevision`).

### 4.3. Loại bỏ các lệnh gọi ngầm trong các slice
- Loại bỏ các lệnh gọi `fetchPlayerStatus` và `fetchGuInventory` thừa thãi sau action response trong các slice `marketplaceSlice`, `apertureSlice`, `arenaSlice`, `inventorySlice`.
- Chỉ giữ lại `fetchPlayerStatus` cho các trường hợp phục hồi lỗi hoặc timeout.
