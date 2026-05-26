# PHASE 3: GU REFINING (CRAFTING & GACHA SYSTEM)

You are an expert full-stack developer (Golang, Fiber/Gin, PostgreSQL, Next.js, TailwindCSS). 
We are continuing the development of the text-based RPG web game based on the "Reverend Insanity" novel. 
In Phase 3, we will implement the Gu Refining system (Crafting), which heavily relies on RNG (Gacha mechanics) and strict resource management.

Please implement the following tasks step-by-step:

### 1. Backend: Database Schema & Golang Models
Create the `RefiningRecipe` model to store crafting formulas.

**A. RefiningRecipe Struct:**
- `ID` (string)
- `TargetGuID` (string): The ID of the Gu that will be created on success.
- `RequiredGus` (JSONB): e.g., `{"GU_MOONLIGHT": 1, "GU_JADE_SKIN": 1}`
- `RequiredMaterials` (JSONB): e.g., `{"wild_boar_tusk": 1}` (Optional auxiliary items).
- `SuccessRate` (float64): Example 0.8 (80% success rate).
- `PrimevalStonesCost` (int): Cost of refining.

**B. Seed Sample Lore-Accurate Recipes:**
Insert these into the database or keep them in a static registry:
1. **Moon Raiment Gu (Nguyệt Nghê Thường):** Requires `GU_MOONLIGHT` (1) + `GU_JADE_SKIN` (1). Success Rate: 0.85.
2. **White Jade Gu (Bạch Ngọc Cổ):** Requires `GU_WHITE_BOAR` (1) + `GU_JADE_SKIN` (1) + Material `wild_boar_tusk` (1). Success Rate: 0.80.
3. **Four Flavors Liquor Worm (Tứ Vị Tửu Trùng):** Requires `GU_LIQUOR_WORM` (2) + Materials `sour_wine` (1), `sweet_wine` (1), `bitter_wine` (1), `spicy_wine` (1). Success Rate: 0.60.

### 2. Backend: Refining Algorithm (Golang)
Create a highly secure API endpoint `POST /api/refine` to handle the crafting logic.
- **Payload:** `{ "player_id": "uuid", "recipe_id": "string", "selected_gu_instance_ids": ["uuid1", "uuid2"] }`
- **Logic & DB Transaction (Crucial to prevent duplication exploits):**
  1. Start a PostgreSQL Transaction.
  2. Fetch the Recipe.
  3. Verify the player has the required Primeval Stones, Materials, and the specific Gu instances (`selected_gu_instance_ids`) in their inventory.
  4. Deduct the Primeval Stones and Materials.
  5. Delete/Remove the used Gu instances from `gu_inventory` (They are consumed regardless of success or failure).
  6. **RNG Calculation:** Generate a random float between 0.0 and 1.0 using `math/rand` (ensure seed is initialized). 
  7. If `rand <= SuccessRate`:
     - Create a new Gu instance based on `TargetGuID` and insert it into the player's inventory.
     - Commit transaction. Return `{ status: "success", message: "Refining successful!", new_gu: {...} }`
  8. If `rand > SuccessRate`:
     - Commit transaction (The materials and old Gus are lost forever). 
     - Return `{ status: "failed", message: "Refining failed. The Gu bugs perished in the fire." }`

### 3. Frontend: Refining Furnace UI (Next.js)
Build a `RefiningFurnace` component. This is a crucial UI that needs a "Gacha" feel.

**A. Drag and Drop Interface:**
- Use `react-dnd` or `@dnd-kit/react`.
- **Left Panel:** Display the user's available Gu inventory.
- **Right Panel (Furnace):** Display slots based on a selected Recipe. Users can drag their Gu from the left panel and drop them into the required slots.

**B. The Refining Action & Animation (Gacha Suspense):**
- Create a "Refine" button that becomes active only when all required slots are filled.
- When clicked:
  1. Call the `POST /api/refine` endpoint.
  2. While waiting for the response (and artificially delaying for 2-3 seconds for suspense), show a **Furnace Animation**: Flames burning intensely, screen shaking slightly (use `framer-motion` or simple CSS keyframes).
  3. Once the 3 seconds pass, resolve the API response.
- **On Success:** Show a bright flash animation, play a success sound, and display a Toast: *"Success! You obtained [Gu Name]!"*.
- **On Failure:** Show an ash/smoke animation, and display a red error Toast: *"Refining Failed! The Gu bugs have perished."*

Please provide the Golang backend code first (Models, RNG logic, Transaction Handler), followed by the Next.js frontend code (DND setup and Animation logic).
Lời khuyên thiết kế (Game Design Notes) cho Giai đoạn 3:
Xử lý Transaction (Rất quan trọng): Trong game, các API liên quan đến túi đồ và chế tạo là nơi dễ bị hack/bug nhất (gọi là Item Duplication Glitch). Yêu cầu AI dùng Transaction trong Golang là để đảm bảo: Game sẽ trừ/xóa Cổ trùng cũ của user TRƯỚC khi thông báo kết quả. Nếu server lag giữa chừng, transaction sẽ tự động rollback, người chơi không bị mất oan vật phẩm mà cũng không thể nhân bản Cổ trùng.
Hiệu ứng Gacha (Delay): Ở frontend, tôi đã yêu cầu AI thêm một khoảng delay giả (artificial delay) khoảng 2-3 giây trước khi hiện kết quả. Dù API Go trả về kết quả trong 0.1 giây, Next.js vẫn sẽ hiện hoạt ảnh ngọn lửa bùng cháy trong 3 giây. Điều này đánh mạnh vào tâm lý hồi hộp, mong chờ của người chơi — cảm giác "nghiện" cốt lõi của mọi game Gacha/Đập đồ.
Cơ chế nanh lợn rừng: Trong nguyên tác, Phương Nguyên dùng thêm nanh lợn rừng (wild_boar_tusk) để tăng xác suất luyện Bạch Ngọc Cổ
,
. Bằng cách thiết kế trường RequiredMaterials dạng JSONB, bạn có thể dễ dàng chèn thêm các vật phẩm phụ này vào công thức mà không làm hỏng cấu trúc code.