Chào bạn, đây là hai vấn đề rất phổ biến khi làm game RPG. 

1. **Lỗi kẹt map:** Thường xảy ra do Frontend chưa quản lý tốt State (trạng thái) của người chơi (đang ở Base hay đang Explore/Combat), hoặc thiếu nút "Bỏ chạy/Trở về".
2. **Hệ thống Shop khởi đầu:** Dựa theo nguyên tác, ở giai đoạn tân thủ (Nhất chuyển, Nhị chuyển trên núi Thanh Mao), nơi mua bán hợp lý nhất chính là **Thương Đội (Caravan)** của Giả gia (Cổ Kim Sinh, Cổ Phú) đi ngang qua thôn trại,. Khi người chơi lên cấp cao hơn, có thể mở khóa **Thành Thương Gia**. Ở shop khởi đầu, chúng ta sẽ bán các vật phẩm cơ bản như thức ăn cho cổ, Cổ trùng cấp 1, và Hình nộm người cỏ (vật phẩm luyện tập giá 3 nguyên thạch).

Dưới đây là kế hoạch chi tiết (Kế hoạch 1.5) để fix map và thêm Shop, được viết dưới dạng Prompt để bạn giao cho AI Agent làm.

---

### BẢN GIAO VIỆC CHO AI AGENT (Copy phần bên dưới)

```markdown
# PHASE 1.5: MAP ESCAPE FIX & BEGINNER SHOP (CARAVAN)

You are an expert full-stack developer (Golang, Fiber/Gin, PostgreSQL, Next.js, TailwindCSS, Zustand). 
We need to fix an issue where the player gets stuck in the Exploration Map, and implement a Beginner Shop system based on the "Caravan" (Thương Đội) lore from Reverend Insanity.

Please implement the following tasks step-by-step:

### TASK 1: Fix Exploration Map Exit / Retreat Mechanism
**1. Frontend State Management (Zustand):**
- Update the game state store to track `currentView` (e.g., 'dashboard', 'exploring', 'combat', 'shop').
- In the `ExplorationMap` and `CombatScreen` components, add a permanent **"Rời khỏi / Trở về" (Flee / Return to Base)** button at the top-left or bottom of the screen.

**2. Backend Logic (Golang):**
- Create an endpoint `POST /api/explore/retreat/{player_id}`.
- Logic: When retreating from a normal node, it's free. If the player is in 'combat' state, retreating costs a small penalty (e.g., deduct 5 HP or 5 Mana for "fleeing in panic").
- Clear the player's active combat/exploration state in Redis/DB and return them to the Base.
- Frontend Action: On success, change `currentView` back to 'dashboard'.

### TASK 2: Beginner Shop System (Thương Đội / Caravan)
**1. Backend: Shop Models & Data (Golang)**
- Create a `ShopItem` struct: `ID`, `Name`, `Description`, `Type` (Gu or Material), `Price` (in primeval_stones), `Stock` (int).
- Create an endpoint `GET /api/shop/list`. Hardcode a starter Caravan catalog based on lore:
  - Item 1: `Hình nộm người cỏ` (Grass Puppet) - Type: Material (for practice), Price: 3.
  - Item 2: `Hoa Nguyệt Lan` (Moon Orchid Petal) - Type: Material (Food for Moonlight Gu), Price: 5.
  - Item 3: `Thịt heo rừng` (Wild Boar Meat) - Type: Material (Food for Black Boar Gu), Price: 10.
  - Item 4: `Cổ Nguyệt Quang` (Moonlight Gu) - Type: Gu, Rank 1, Price: 50.
  - Item 5: `Cổ Hắc Thỉ` (Black Boar Gu) - Type: Gu, Rank 1, Price: 100.
  - Item 6: `Cổ Tịnh Thủy` (Cleansing Water Gu) - Type: Gu, Rank 1, Price: 650.

**2. Backend: Purchase Logic**
- Create `POST /api/shop/buy`. Payload: `{ player_id, item_id, quantity }`.
- Logic & DB Transaction:
  1. Fetch player's current `primeval_stones`.
  2. Calculate `total_cost = item.Price * quantity`.
  3. If `primeval_stones < total_cost`, return 400 "Not enough primeval stones".
  4. If sufficient, deduct stones.
  5. If `Type == Gu`, add the Gu to `players.gu_inventory` JSONB.
  6. If `Type == Material`, increment the quantity in `players.materials_inventory` JSONB.
  7. Commit transaction and return updated player stats and inventory.

**3. Frontend: Caravan Shop UI (Next.js)**
- Create a `CaravanShop` component. 
- Use a Grid layout to display items. Each item card should show an icon, Name, Description, and Price (with a Primeval Stone icon).
- Add a "Mua" (Buy) button. When clicked, show a small modal to select quantity (for materials) or buy 1 (for Gu).
- Call `/api/shop/buy`. On success, show a Toast: "Giao dịch thành công!" and update the player's primeval stones in the UI instantly.
```

### Giải thích thiết kế cho bạn:
1. **Về việc Kẹt Map:** Lỗi này thường do UI không có component "Navigation/Breadcrumb" khi render Component. Việc thêm State `currentView` vào Zustand và nút "Trở về" sẽ giúp người dùng luôn có đường thoát ra màn hình chính (Dashboard/Không Khiếu).
2. **Về Shop (Thương đội):** Tôi đã đưa vào đúng các vật phẩm trong nguyên tác như *Hình nộm người cỏ* (giá 3 nguyên thạch), *Cổ Tịnh Thủy* (giá 650 nguyên thạch). Việc sử dụng API Transaction ở Backend khi mua hàng (như Task 2.2) là bắt buộc để tránh bug người chơi click liên tục 2 lần để mua đồ dù không đủ tiền (Race condition).