# PHASE 2: GU BUG SYSTEM & INVENTORY MANAGEMENT

You are an expert full-stack developer (Golang, Fiber/Gin, PostgreSQL, Next.js, TailwindCSS). 
We are continuing the development of the text-based RPG web game based on the "Reverend Insanity" novel. 
In Phase 2, we will implement the Gu Bug (Equipment/Skill) system and the Hunger/Feeding survival mechanics.

Please implement the following tasks step-by-step:

### 1. Backend: Golang Models & Database Schema
Update the PostgreSQL database and Golang structs.

**A. Gu Dictionary (Master Data)**
Create a struct `GuDictionary` to store fixed stats for each Gu bug. 
- Fields: `ID` (string), `Name` (string), `Rank` (int), `Type` (string: "Active", "Passive", "Support"), `BasePower` (float), `HungerRate` (int: amount of hunger lost per hour), `FoodType` (string), `MaxHunger` (int).
- Hardcode the following sample data in a registry/map:
  1. `GU_MOONLIGHT`: Name: "Nguyệt Quang Cổ", Rank: 1, Type: "Active", BasePower: 50, FoodType: "moon_orchid_petal", HungerRate: 5, MaxHunger: 100.
  2. `GU_JADE_SKIN`: Name: "Ngọc Bì Cổ", Rank: 1, Type: "Passive", BasePower: 500, FoodType: "jade_stone", HungerRate: 2, MaxHunger: 100.
  3. `GU_LIQUOR_WORM`: Name: "Tửu Trùng", Rank: 1, Type: "Support", BasePower: 0, FoodType: "green_bamboo_wine", HungerRate: 3, MaxHunger: 100.

**B. Player Inventory (PostgreSQL JSONB)**
Update the `players` table to include two new JSONB columns:
1. `gu_inventory` (JSONB): Stores the list of Gu bugs the player owns. 
   - Example structure: `[{"instance_id": "uuid", "gu_id": "GU_MOONLIGHT", "hunger": 50, "status": "ACTIVE"}]`. 
   - `status` can be "ACTIVE", "SLEEPING", or "DEAD".
2. `materials_inventory` (JSONB): Stores food and crafting materials.
   - Example structure: `{"moon_orchid_petal": 20, "green_bamboo_wine": 5, "jade_stone": 2}`.

### 2. Backend: Feeding Mechanism & Hunger Worker
**A. Hunger Background Worker**
- Create a Goroutine worker that runs every 1 hour (or mock it to 1 minute for testing).
- It iterates through all players and decreases the `hunger` of each Gu in `gu_inventory` by its `HungerRate`.
- **Lore Mechanic:** If `hunger` drops to 0, there is a 20% chance the Gu's `status` changes to "SLEEPING" (hibernation), and an 80% chance it changes to "DEAD".

**B. Feed API Endpoint**
- Create `POST /api/inventory/feed`
- Payload: `{ "player_id": "uuid", "gu_instance_id": "uuid" }`
- Logic: 
  1. Find the Gu in the player's `gu_inventory`. Check its `FoodType` from the `GuDictionary`.
  2. Check if the player has >= 1 of that `FoodType` in `materials_inventory`.
  3. If not, return 400 "Not enough food materials".
  4. If yes, deduct 1 from the material count, restore the Gu's `hunger` to `MaxHunger`. If the Gu was "SLEEPING", change its status back to "ACTIVE".
  5. Save to DB and return the updated inventories.

### 3. Frontend: Gu Management UI (Next.js)
Build a `GuInventory` component to display and manage the player's Gu bugs and materials.

**A. Materials Section:**
- A simple flex row showing icons/text of current food materials and their quantities (e.g., Moon Orchid Petals: 20, Green Bamboo Wine: 5).

**B. Gu Grid Section:**
- Display the `gu_inventory` as a CSS Grid.
- Each Gu card should display:
  - Gu Name and Rank.
  - A visual `Hunger Bar` (Progress bar: Green if > 50%, Yellow if 20-50%, Red if < 20%).
  - Status text (Active, Sleeping, Dead).
  - A "Feed" button.
- **Interactivity:** 
  - If status is "DEAD", disable the card (grayed out).
  - When "Feed" is clicked, call `POST /api/inventory/feed`. Show a loading spinner on the button. On success, update the state (Zustand/React Query) and show a toast "Fed successfully!". On error, show a toast "Not enough food!".

Please provide the Golang backend code first (Models, Worker, API handler), followed by the Next.js frontend component.



`Cơ chế Ngủ Đông (Hibernation): Trong nguyên tác, tự nhiên vừa tàn nhẫn vừa nhân từ. Cổ trùng hết thức ăn phần lớn sẽ chết, nhưng có xác suất nhỏ tự phong ấn mình (Nguyệt Quang cổ hóa thành đá xám, Tửu trùng hóa thành kén trắng)
. Việc cho AI code tỷ lệ 20% "SLEEPING" và 80% "DEAD" sẽ tạo ra tính rủi ro chân thực, nhưng vẫn chừa lại đường sống cho người chơi nếu lỡ quên đăng nhập.
Quản lý Inventory bằng JSONB: Việc lưu materials (như hoa Nguyệt Lan, rượu Thanh Trúc, ngọc thạch
) và gu_inventory dưới dạng JSONB trong PostgreSQL thay vì tạo nhiều bảng phụ (Joins) sẽ giúp query cực kỳ nhanh, phù hợp cho Web Game có lượng thao tác thay đổi túi đồ liên tục.
Thanh Độ Đói (Hunger Bar): Giao diện Frontend hiển thị thanh màu (Xanh -> Vàng -> Đỏ) sẽ trực quan hóa câu nói "Nuôi cổ như nuôi tình nhân"
. Nó sẽ tạo áp lực sinh tồn ép người chơi phải liên tục đi làm nhiệm vụ (đã làm ở Phase 1) để kiếm nguyên liệu cho cổ ăn.
GuListBasic.txt
ItemListBasic.txt
`
