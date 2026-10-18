# Nghiên cứu: Rà soát toàn bộ các endpoint biến động tài nguyên và tình trạng tăng state_revision trong backend

- **Ticket liên quan:** #225 (thuộc Epic #224)
- **Tài liệu phối hợp:** `docs/research/frontend_state_sync_audit.md` (#226)
- **Ngày thực hiện:** 2026-10-18
- **Mục tiêu:** Rà soát toàn bộ các handler/endpoint làm biến động tài nguyên người chơi (vật phẩm, Cổ trùng, linh thạch, sinh lực, chân nguyên, niệm đầu, cống hiến, đạo ngân); kiểm tra responder helper được gọi; phát hiện các trường dữ liệu bị bỏ sót; và phân tích cơ chế tăng `state_revision` trong database.

---

## 1. Cơ chế State Revision trong Database & Backend

### 1.1. Hiện trạng triển khai tại PostgreSQL
Tại `backend/internal/db/schema_players.go`:
1. **Trigger trên bảng `players` (lines 118-131):**
   ```sql
   CREATE OR REPLACE FUNCTION bump_player_row_state_revision()
   RETURNS trigger LANGUAGE plpgsql AS $$
   BEGIN
       NEW.state_revision := OLD.state_revision + 1;
       RETURN NEW;
   END;
   $$;
   CREATE TRIGGER trg_players_state_revision
       BEFORE UPDATE ON players
       FOR EACH ROW EXECUTE FUNCTION bump_player_row_state_revision();
   ```
   -> Bất kỳ câu lệnh `UPDATE players` nào được commit vào DB đều tự động tăng `players.state_revision` lên 1 đơn vị. Không có câu lệnh UPDATE thủ công nào ghi đè trường này.

2. **Trigger trên các bảng con (Collection Triggers, lines 240-325):**
   Mọi bảng có Foreign Key trỏ về `players(id)` (ví dụ: `player_inventory_items`, `player_gu`, `player_recipe_progress`, `player_killer_moves`, v.v.) đều được tự động gắn trigger `bump_player_collection_state_revision()`. Khi có thao tác `INSERT`, `UPDATE`, `DELETE` trên bảng con, trigger sẽ tăng `state_revision` trong bảng `player_collection_revisions`.

3. **Tính toán tổng Revision tại Backend (`backend/internal/handlers/respond.go:19-30`):**
   ```sql
   SELECT p.state_revision + COALESCE((
       SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id
   ), 0), p.primeval_stones, p.immortal_stones, p.human_qi,
   p.hp, p.current_thought, p.soul_hp, p.lifespan
   FROM players p WHERE p.id = $1
   ```
   Hàm `refreshActionPlayerState` tính tổng của revision dòng `players` và toàn bộ shard revision của các bảng con.

### 1.2. Nguồn gốc lỗi lệch State Revision và Mất đồng bộ UI
Mặc dù cơ chế Trigger DB hoạt động chuẩn xác, lỗi nghiêm trọng xảy ra ở tầng Application:
1. **Bất đồng bộ bộ nhớ Go (In-memory Go model):**
   Khi transaction bắt đầu, `evaluateManaTx` tải model `player` với giá trị `state_revision` ban đầu $R_0$. Sau khi câu lệnh `UPDATE players` hoặc `InventorySvc.RemoveItem` chạy, trigger DB tăng revision lên $R_1 = R_0 + 1$. Tuy nhiên, biến struct `player` trong bộ nhớ Go vẫn giữ giá trị cũ $R_0$.
   - Nếu handler gọi `respondActionWithInventory`, `respondActionGuOnly` hoặc `respondActionOnly`: hàm `refreshActionPlayerState(&player)` được gọi để đọc lại $R_1$ từ DB.
   - Nhưng nếu handler gọi trực tiếp `c.JSON` hoặc `responseWithPlayer`: biến `player.StateRevision` trả về client vẫn mang giá trị cũ $R_0$!
2. **Hiện tượng Drop phản hồi do điều kiện `incomingRevision > appliedRevision` ở Frontend:**
   Khi backend trả về HTTP response với `state_revision = R`, nhưng response lại thiếu các trường dữ liệu (ví dụ thiếu `materials_inventory`, `hp`, hoặc `current_mana`):
   - Frontend áp dụng response và gán `playerStateRevision = R`.
   - Khi WebSocket event mang đầy đủ snapshot đến sau với `state_revision = R`, hàm `shouldApplyPlayerRevision` kiểm tra `R > R` $\rightarrow$ `false` $\rightarrow$ **DROP toàn bộ WebSocket event!**
   - Hậu quả: Dữ liệu biến động bị mất vĩnh viễn trên UI cho đến khi F5 trang.

---

## 2. Phân tích các Responder Helper hiện hành (`respond.go`)

### 2.1. `refreshActionPlayerState(player *models.Player)` (lines 13-30)
- **Đang đọc:** `state_revision`, `primeval_stones`, `immortal_stones`, `human_qi`, `hp`, `current_thought`, `soul_hp`, `lifespan`.
- **BỎ SÓT:** 
  - `current_mana`, `max_mana` (Cực kỳ quan trọng: đan dược hồi mana, nạp đá hồi mana, dùng cổ tiêu hao tốn mana).
  - `aptitude_percent`, `aptitude_type` (Cổ tăng tư chất).
  - `rank`, `sub_rank`, `wall_progress` (Đột phá, vượt ải).
  - `dao_marks` (Nằm ở bảng `immortal_apertures`, biến động khi dùng cổ hoặc độ kiếp).

### 2.2. `respondActionWithInventory` (lines 57-83)
- **Trả về:** `success`, `gu_inventory`, `materials_inventory`, `primeval_stones`, `immortal_stones`, `human_qi`, `state_revision`.
- **Khuyết điểm nghiêm trọng:** Mặc dù `refreshActionPlayerState` có đọc `hp`, `current_thought`, `soul_hp`, `lifespan`, nhưng trong JSON trả về `resp := gin.H{...}` lại **HOÀN TOÀN BỎ QUÊN** `hp`, `current_thought`, `soul_hp`, `lifespan` và `current_mana`!
- **Hệ quả:** Dùng đan dược hồi máu/thọ nguyên (`player_use.go`), thôi diễn tốn niệm đầu (`refine_deduce.go`), luyện cổ phản phệ mất máu (`refine_gu.go`), luyện hồn mất soul_hp (`soul.go`) đều không cập nhật được các chỉ số này trên giao diện thông qua direct action response.

### 2.3. `respondActionGuOnly` (lines 87-112)
- **Trả về:** `success`, `gu_inventory`, `primeval_stones`, `immortal_stones`, `human_qi`, `hp`, `current_thought`, `soul_hp`, `lifespan`, `state_revision`.
- **Khuyết điểm:** Không chứa `materials_inventory`.
- **Nguy cơ:** Bị lạm dụng ở các handler có tiêu hao nguyên liệu trong túi đồ.

### 2.4. `respondActionOnly` (lines 117-131)
- **Trả về:** `success`, `state_revision` (kèm extra).
- **Khuyết điểm:** Mặc định không trả về `primeval_stones`, `immortal_stones`, `human_qi`, `materials_inventory`, hay bất kỳ chỉ số nhân vật nào.

---

## 3. Bảng rà soát chi tiết toàn bộ các Endpoint biến động tài nguyên

| STT | File & Dòng | Endpoint / Hàm | Tài nguyên biến động | Responder đang dùng | Tình trạng thiếu trường / Lỗi State Revision | Mức độ |
|:---:|:---|:---|:---|:---|:---|:---:|
| 1 | `player_feed.go:416` | `POST /api/player/feed-gu` (`FeedGuSingle`) | Trừ vật phẩm thức ăn (`materials_inventory`), tăng độ no Cổ | `respondActionGuOnly` | **Sai responder**: Dùng `respondActionGuOnly` khi có trừ nguyên liệu. Thiếu `materials_inventory`. Thức ăn không giảm trên UI. | **CRITICAL** |
| 2 | `player_feed.go:633` | `POST /api/player/feed-group` (`FeedGroup`) | Trừ vật phẩm thức ăn (`materials_inventory`), tăng độ no Cổ | `respondActionGuOnly` | **Sai responder**: Thiếu `materials_inventory`. Thức ăn trong túi không giảm trên UI. | **CRITICAL** |
| 3 | `player_feed.go:881` | `POST /api/player/feed-all` (`FeedAll`) | Trừ nhiều vật phẩm thức ăn (`materials_inventory`), tăng độ no Cổ | `respondActionGuOnly` | **Sai responder**: Thiếu `materials_inventory`. | **CRITICAL** |
| 4 | `player_equip.go:519` | `POST /api/player/gu/status` (`UpdateGuStatus` phong ấn) | Trừ `sealing_stone` trong `materials_inventory`, đổi trạng thái Cổ | `respondActionGuOnly` | **Sai responder**: Tiêu hao Đá Phong Ấn trong kho nhưng gọi `respondActionGuOnly`, thiếu `materials_inventory`. | **HIGH** |
| 5 | `arena_shop.go:242` | `POST /api/arena/shop/buy` (`BuyArenaShopItem`) | Trừ `arena_coin` (vật phẩm), cộng vật phẩm mua hoặc cộng `primeval_stones` | Raw `c.JSON` | **Không dùng responder**: Trả về `{"message":..., "cost":...}`. Thiếu `materials_inventory`, thiếu `primeval_stones`, thiếu `state_revision`. | **CRITICAL** |
| 6 | `daily_missions.go:289` | `POST /api/player/daily-missions/claim` (`ClaimDailyMissionReward`) | Cộng `primeval_stones` hoặc `immortal_stones` | Raw `c.JSON` | **Không dùng responder**: Trả về `{"message":..., "reward_type":...}`. Thiếu `primeval_stones`, thiếu `immortal_stones`, thiếu `state_revision`. | **CRITICAL** |
| 7 | `mine.go:460` | `POST /api/mine/abandon` (`AbandonMineSlot`) | Nhận thưởng tích lũy: cộng đá (`primeval_stones`), nhân khí (`human_qi`), rương vật phẩm (`materials_inventory`) | `respondActionOnly` | **Sai responder**: Thao tác có phát thưởng đá và rương nguyên liệu nhưng gọi `respondActionOnly`. Thiếu `materials_inventory`, thiếu `primeval_stones`. | **CRITICAL** |
| 8 | `player_recharge.go:87, 167` | `POST /api/player/recharge` (`RechargeManaHandler`) | Trừ `primeval_stones` hoặc `immortal_stones`, tăng `current_mana` | `respondActionOnly` | **Sai responder**: Trừ đá và hồi phục chân nguyên nhưng gọi `respondActionOnly`. Thiếu `primeval_stones`, thiếu `immortal_stones`, thiếu `current_mana`. | **HIGH** |
| 9 | `premium.go:678` | `POST /api/shop/exchange` (`ExchangeStones`) | Đổi Tiên Nguyên Thạch $\leftrightarrow$ Nguyên Thạch | `respondActionOnly` | **Thiếu trường**: Trả về `{"wallet": wallet}` lồng nhau, không có `primeval_stones` và `immortal_stones` ở cấp root cho `applyInventoryFromResponse`. | **HIGH** |
| 10 | `sect_salary.go:207` | `POST /api/sects/salary/pay` (`PaySectSalary`) | Cộng `primeval_stones` cho toàn bộ thành viên tông môn (bao gồm caller) | Raw `c.JSON` | **Không dùng responder**: Trả về struct `PaySectSalaryResponse`. Thiếu `primeval_stones` và `state_revision` cho người nhận. | **HIGH** |
| 11 | `instance.go:507` | `POST /api/instances/enter` (`EnterInstance`) | Tiêu hao vật phẩm vé vào phó bản (`materials_inventory`) | Raw `c.JSON` | **Không dùng responder**: Trừ vé phó bản trong DB nhưng response không trả `materials_inventory` hay `state_revision`. | **HIGH** |
| 12 | `aperture_collect.go:253` | `POST /api/aperture/resources/:nodeId/collect` (`CollectApertureResourcePoint`) | Cộng `primeval_stones` hoặc `materials_inventory`, tăng `human_qi` | Raw `c.JSON` | **Thiếu State Revision**: Trả về đá/vật phẩm lẻ tẻ nhưng không gọi responder chuẩn, thiếu `state_revision`. | **HIGH** |
| 13 | `aperture_collect.go:908` | `POST /api/aperture/resources/collect-all` (`CollectAllApertureResources`) | Thu hoạch hàng loạt: cộng `primeval_stones`, `materials_inventory` | Raw `c.JSON` | **Thiếu State Revision**: Không trả về `state_revision`. Client store không cập nhật được revision chuẩn từ DB. | **HIGH** |
| 14 | `aperture_collect.go:1036` | `POST /api/aperture/nodes/repair-all` (`RepairAllApertureNodes`) | Trừ `immortal_stones` sửa toàn bộ công trình | Raw `c.JSON` | **Không dùng responder**: Trừ Tiên Nguyên Thạch nhưng response không trả `immortal_stones` hay `state_revision`. | **HIGH** |
| 15 | `aperture_cleanse.go:178` | `POST /api/aperture/cleanse` (`CleanseImmortalAperture`) | Trừ `immortal_stones`, hoàn trả đạo tài (`materials_inventory`), đổi `dao_marks` | Raw `c.JSON` | **Thiếu State Revision**: Trả về `immortal_stones` và `materials_inventory` nhưng thiếu `state_revision`. | **HIGH** |
| 16 | `aperture_nodes.go:222` | `POST /api/aperture/nodes/bulk` (`BulkApertureNodesHandler`) | Trừ `immortal_stones` nâng cấp/sửa chữa | Raw `c.JSON` | **Không dùng responder**: Thiếu `immortal_stones`, thiếu `state_revision`. | **HIGH** |
| 17 | `aperture_nodes.go:672` | `POST /api/aperture/resources` (`BuildApertureNodeHandler`) | Trừ `immortal_stones`, trừ giống/vật phẩm (`materials_inventory`), tiêu hao Cổ | Raw `c.JSON` | **Không dùng responder**: Trừ nhiều loại tài nguyên nhưng trả về raw struct, thiếu `state_revision`. | **HIGH** |
| 18 | `aperture_nodes.go:745` | `DELETE /api/aperture/resources/:nodeId` (`UninstallApertureNodeHandler`) | Hoàn trả vật phẩm vào túi đồ (`materials_inventory`) | Raw `c.JSON` | **Không dùng responder**: Thiếu `materials_inventory`, thiếu `state_revision`. | **HIGH** |
| 19 | `marketplace.go:354` | `POST /api/shop/bao-hoang/listings` (`CreateListing`) | Rút vật phẩm hoặc Cổ khỏi kho để treo bán | Raw `c.JSON` | **Không dùng responder**: Bán đồ nhưng response chỉ trả về `listing_id`, không trả kho mới và `state_revision`. | **HIGH** |
| 20 | `marketplace.go:623` | `POST /api/shop/bao-hoang/bid` (`PlaceBid`) | Trừ `primeval_stones` hoặc `immortal_stones` đặt giá | Raw `c.JSON` | **Không dùng responder**: Thiếu `primeval_stones`, `immortal_stones`, thiếu `state_revision`. | **HIGH** |
| 21 | `marketplace.go:944` | `POST /api/shop/bao-hoang/cancel` (`CancelListing`) | Hoàn trả vật phẩm hoặc Cổ về kho | Raw `c.JSON` | **Không dùng responder**: Thiếu `gu_inventory`, `materials_inventory`, thiếu `state_revision`. | **HIGH** |
| 22 | `tribulation_handlers.go:590` | `POST /api/aperture/tribulation/:eventId/resolve` (`ResolveTribulation`) | Nhận Đạo Ngân, tăng cấp, trừ đá phạt khi thất bại, tăng Nhân Khí | Raw `c.JSON` | **Không dùng responder**: Thiếu `primeval_stones`, `immortal_stones`, `state_revision`. | **HIGH** |
| 23 | `player_use.go:198, 443` | `POST /api/player/use-item`, `use-items` (`UseItem`, `UseItems`) | Trừ dược liệu, tăng `hp`, `current_mana`, `lifespan`, `soul_hp`, `dao_marks` | `respondActionWithInventory` | **Khuyết trường trong responder**: Dù `respondActionWithInventory` có tải `materials_inventory`, payload lại thiếu `hp`, `current_mana`, `soul_hp`, `lifespan`. | **HIGH** |
| 24 | `player_gu_use.go:219` | `POST /api/player/use-consumable-gu` (`UseConsumableGu`) | Tiêu hao Cổ, trừ `current_mana`, tăng tư chất/thọ nguyên/chỉ số/đạo ngân | `respondActionGuOnly` | **Khuyết trường**: Thiếu `current_mana`, thiếu `aptitude_percent`, thiếu `dao_marks`. Nếu có nhận/mất vật phẩm thì thiếu `materials_inventory`. | **HIGH** |
| 25 | `refine_gu.go:670, 677` | `POST /api/gu/refine` (`RefineGu`) | Tiêu hao Cổ nguyên liệu, đá, trừ `current_mana`, có thể trừ `hp`/`soul_hp` khi phản phệ | `respondActionWithInventory` | **Khuyết trường**: Thiếu `current_mana`, thiếu `hp` và `soul_hp` (khi phản phệ nổ lò). | **MEDIUM** |
| 26 | `refine_deduce.go:251` | `POST /api/gu/refine/deduce` (`DeduceRecipe`) | Trừ `current_thought` (niệm đầu), trừ `primeval_stones` | `respondActionWithInventory` | **Khuyết trường**: `refreshActionPlayerState` có đọc `current_thought` nhưng `respondActionWithInventory` bỏ sót không đưa vào response. | **MEDIUM** |
| 27 | `soul.go:105` | `POST /api/player/soul/refine` (`RefineSoul`) | Trừ 80% `soul_hp`, tăng `soul_tier` | `respondActionWithInventory` | **Khuyết trường**: Giảm mạnh `soul_hp` nhưng `respondActionWithInventory` không trả trường `soul_hp`. | **MEDIUM** |
| 28 | `player_profile.go:287` | `POST /api/player/profile/rename` (`RenamePlayer`) | Trừ đá hoặc thẻ đổi tên | `responseWithPlayer` | **Stale State Revision**: `responseWithPlayer` không gọi `refreshActionPlayerState`, trả về `state_revision` cũ trong Go struct. | **MEDIUM** |
| 29 | `sect_donations.go:101` | `POST /api/sects/donate/stones` (`DonateStones`) | Trừ `primeval_stones`, tăng điểm cống hiến | `respondActionOnly` | Truyền `primeval_stones` qua extra nên đá có trả về, nhưng dùng `respondActionOnly`. | **LOW** |
| 30 | `shop_sell.go:116` | `POST /api/shop/sell`, `sell-bulk` (`runCaravanSell`) | Bán Cổ/vật phẩm, cộng đá | `respondActionWithInventory` | Đầy đủ `materials_inventory`, `gu_inventory`, `primeval_stones`, `state_revision`. | **OK** |
| 31 | `shop_buy.go:280` | `POST /api/shop/buy` (`BuyShopItem`) | Mua Cổ/vật phẩm, trừ đá | `respondActionWithInventory` | Đầy đủ `materials_inventory`, `gu_inventory`, `primeval_stones`, `state_revision`. | **OK** |
| 32 | `sect_vault.go:942` | `POST /api/sects/vault/buy` (`BuyFromSectVault`) | Đổi cống hiến lấy Cổ/vật phẩm | `respondActionWithInventory` | Đầy đủ `materials_inventory`, `gu_inventory`, `member`. | **OK** |
| 33 | `quests.go:455` | `POST /api/quests/:id/complete` (`CompleteQuest`) | Nhận thưởng nhiệm vụ: cộng đá, vật phẩm, Cổ | `respondActionWithInventory` | Đầy đủ `materials_inventory`, `gu_inventory`, `primeval_stones`. | **OK** |
| 34 | `achievements.go:291` | `POST /api/achievements/:id/claim` (`ClaimAchievementReward`) | Nhận thưởng thành tựu: cộng đá, vật phẩm | `respondActionWithInventory` | Đầy đủ `materials_inventory`, `primeval_stones`, `immortal_stones`. | **OK** |
| 35 | `gacha.go:547, 558` | `POST /api/gacha/open` (`OpenGacha`) | Mở thạch: trừ đá, cộng Cổ/vật phẩm/đá | `respondActionWithInventory` | Đầy đủ `materials_inventory`, `gu_inventory`, `primeval_stones`, `immortal_stones`. | **OK** |
| 36 | `exploration_move.go:894` | `POST /api/exploration/move` (`MovePlayer` nhặt tài nguyên) | Nhặt tài nguyên hoang dã vào túi | `respondActionWithInventory` | Đầy đủ `materials_inventory`, `gu_inventory`. | **OK** |
| 37 | `mine.go:374` | `POST /api/mine/claim` (`ClaimMineRewards`) | Nhận thưởng mỏ: cộng đá, nhân khí, rương | `respondActionWithInventory` | Đầy đủ `materials_inventory`, `primeval_stones`. | **OK** |

---

## 4. Phân loại mức độ nghiêm trọng (Severity Matrix)

### Mức CRITICAL (Ảnh hưởng trực tiếp đến tính toàn vẹn kinh tế & trải nghiệm người chơi)
1. **Cho Cổ ăn (`FeedGuSingle`, `FeedGroup`, `FeedAll`):** Tiêu tốn hàng chục đơn vị thức ăn nhưng gọi `respondActionGuOnly`, túi đồ không giảm nguyên liệu khiến người chơi tưởng lỗi hoặc click liên tục.
2. **Shop Đấu Trường (`BuyArenaShopItem`):** Dùng raw `c.JSON`, mất hoàn toàn Xu Đấu trường và vật phẩm/đá vừa mua khỏi giao diện cho tới khi reload trang.
3. **Nhiệm vụ hàng ngày (`ClaimDailyMissionReward`):** Dùng raw `c.JSON`, nhận linh thạch nhưng số dư trên header không nhảy số.
4. **Từ bỏ mỏ tài nguyên (`AbandonMineSlot`):** Nhận thưởng đá + rương nhưng gọi `respondActionOnly`, không trả kho nguyên liệu và số đá mới.

### Mức HIGH (Sai lệch chỉ số sinh tồn, tiền tệ và thiếu State Revision)
1. **Dùng đan dược (`UseItem`, `UseItems`):** Hồi phục máu, chân nguyên, thọ nguyên nhưng `respondActionWithInventory` không trả `hp`, `current_mana`, `lifespan`. Client áp dụng revision rỗng và drop WebSocket packet kế tiếp.
2. **Nạp Chân Nguyên (`RechargeManaHandler`):** Trừ đá hồi chân nguyên nhưng gọi `respondActionOnly`, giao diện không cập nhật chân nguyên và số đá.
3. **Đổi linh thạch (`ExchangeStones`):** Không trả số dư đá mới ở cấp root.
4. **Phong ấn Cổ (`UpdateGuStatus`):** Trừ Đá Phong Ấn nhưng gọi `respondActionGuOnly`.
5. **Toàn bộ hệ thống Tiên Khiếu (`Aperture`):** Thu hoạch, xây dựng mỏ, sửa chữa công trình, tẩy thể đều dùng raw `c.JSON` thiếu `state_revision`.
6. **Bảo Hoàng Thiên (`Marketplace`):** Đăng bán, đặt giá, hủy niêm yết dùng raw `c.JSON`.
7. **Phó bản (`EnterInstance`):** Trừ vé vào cửa nhưng dùng raw `c.JSON`.

### Mức MEDIUM (Ảnh hưởng tính năng thứ cấp hoặc chỉ số phụ)
1. **Thôi diễn công thức (`DeduceRecipe`):** Thiếu `current_thought`.
2. **Luyện Cổ (`RefineGu`):** Thiếu `current_mana`, thiếu sát thương phản phệ `hp`/`soul_hp`.
3. **Luyện Hồn (`RefineSoul`):** Thiếu `soul_hp`.
4. **Đổi tên (`RenamePlayer`):** Dùng `responseWithPlayer` mang `state_revision` cũ.

---

## 5. Giải pháp Kiến trúc Đề xuất

### 5.1. Nâng cấp `refreshActionPlayerState` (`respond.go`)
Mở rộng câu lệnh query trong `refreshActionPlayerState` để đọc đầy đủ các trường thường xuyên biến động:
```go
func refreshActionPlayerState(player *models.Player) {
    if player == nil || player.ID == uuid.Nil {
        return
    }
    _ = db.DB.QueryRow(`
        SELECT p.state_revision + COALESCE((
            SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id
        ), 0), p.primeval_stones, p.immortal_stones, p.human_qi,
        p.hp, p.current_mana, p.current_thought, p.soul_hp, p.lifespan,
        p.aptitude_percent, p.sub_rank, p.rank, p.wall_progress
        FROM players p WHERE p.id = $1
    `, player.ID).Scan(
        &player.StateRevision, &player.PrimevalStones,
        &player.ImmortalStones, &player.HumanQi, &player.HP,
        &player.CurrentMana, &player.CurrentThought, &player.SoulHP, &player.Lifespan,
        &player.AptitudePercent, &player.SubRank, &player.Rank, &player.WallProgress,
    )
}
```

### 5.2. Chuẩn hóa Payload phản hồi `respondActionWithInventory`
Đưa tất cả các trường chỉ số sống đã được `refreshActionPlayerState` truy vấn vào JSON trả về của `respondActionWithInventory`:
```go
resp := gin.H{
    "success":             true,
    "state_revision":      player.StateRevision,
    "gu_inventory":        player.GuInventory,
    "materials_inventory": loadMaterialsInventory(player.ID.String()),
    "primeval_stones":     player.PrimevalStones,
    "immortal_stones":     player.ImmortalStones,
    "human_qi":            player.HumanQi,
    "hp":                  player.HP,
    "current_mana":        player.CurrentMana,
    "current_thought":     player.CurrentThought,
    "soul_hp":             player.SoulHP,
    "lifespan":            player.Lifespan,
    "aptitude_percent":    player.AptitudePercent,
    "rank":                player.Rank,
    "sub_rank":            player.SubRank,
    "wall_progress":       player.WallProgress,
}
```

### 5.3. Chuyển đổi các Endpoint vi phạm sang Responder chuẩn
1. Thay thế `respondActionGuOnly` $\rightarrow$ `respondActionWithInventory` tại:
   - `player_feed.go`: `FeedGuSingle`, `FeedGroup`, `FeedAll`.
   - `player_equip.go`: `UpdateGuStatus` (khi phong ấn Cổ).
2. Thay thế `respondActionOnly` $\rightarrow$ `respondActionWithInventory` tại:
   - `mine.go:460`: `AbandonMineSlot`.
   - `player_recharge.go:87, 167`: `RechargeManaHandler`.
3. Thay thế các lệnh raw `c.JSON` $\rightarrow$ `respondActionWithInventory` tại:
   - `arena_shop.go:242`: `BuyArenaShopItem`.
   - `daily_missions.go:289`: `ClaimDailyMissionReward`.
   - `instance.go:507`: `EnterInstance`.
   - `aperture_collect.go:253, 908, 1036`: `CollectApertureResourcePoint`, `CollectAllApertureResources`, `RepairAllApertureNodes`.
   - `aperture_cleanse.go:178`: `CleanseImmortalAperture`.
   - `aperture_nodes.go:222, 672, 745`: `BulkApertureNodesHandler`, `BuildApertureNodeHandler`, `UninstallApertureNodeHandler`.
   - `marketplace.go:354, 623, 944`: `CreateListing`, `PlaceBid`, `CancelListing`.

### 5.4. Đồng bộ với Frontend (Giải quyết triệt để Issue #224, #226, #227)
- Phía Frontend: Điều chỉnh hàm `shouldApplyPlayerRevision` tại `frontend/src/store/playerRevision.ts` đổi điều kiện thành `>=` thay vì `>` (khi bằng revision vẫn thực hiện merge các trường non-null).
- Đồng bộ mở rộng hàm `applyInventoryFromResponse` (hoặc `applyServerStateUpdate`) tại `useGameStore.ts` để tiếp nhận đầy đủ `current_mana`, `aptitude_percent`, `wall_progress`, `rank`, `sub_rank` từ direct response.
