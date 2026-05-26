1. Phân tích Các Tính Năng Cơ Bản (Phase 1)
Tạo nhân vật (Gacha Tư Chất): Khi User đăng ký, hệ thống sẽ random "Tư Chất" (Aptitude). Dưới 40% là Đinh, 40-59% là Bính, 60-79% là Ất, 80-99% là Giáp
.
Chỉ số cơ bản: Nhân vật bắt đầu ở cấp Nhất Chuyển Sơ Giai (Rank 1, Sub-rank 1), sở hữu chân nguyên màu Thanh Đồng
.
Hệ thống thời gian thực (Passive Mana Regen): Chân nguyên tự động hồi phục theo thời gian. Tư chất càng cao hồi càng nhanh (VD: loại Bính hồi 4 phần mỗi giờ, loại Giáp hồi 8 phần mỗi giờ)
.
Hành động cày cuốc cơ bản: Tính năng "Khám phá" hoặc "Đập đá" tiêu hao Chân nguyên để thu về Nguyên Thạch (Primeval Stones) - đơn vị tiền tệ chính của game
.

--------------------------------------------------------------------------------
2. Thiết kế Database Schema (PostgreSQL)
Để tối ưu cho game, chúng ta không lưu Máu (HP) theo kiểu lên cấp được cộng thêm, mà mọi chỉ số dựa vào trạng thái hiện tại. Bảng players là quan trọng nhất:
CREATE TABLE players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    rank INTEGER DEFAULT 1,             -- 1 đến 9 (Nhất chuyển đến Cửu chuyển)
    sub_rank INTEGER DEFAULT 1,         -- 1: Sơ giai, 2: Trung giai, 3: Cao giai, 4: Đỉnh phong
    aptitude_type VARCHAR(10) NOT NULL, -- 'Giáp', 'Ất', 'Bính', 'Đinh'
    aptitude_percent INTEGER NOT NULL,  -- Giá trị từ 20 đến 99
    current_mana FLOAT NOT NULL,        -- Lượng chân nguyên hiện tại
    max_mana FLOAT NOT NULL,            -- Bằng với aptitude_percent (VD: Bính 44% thì max_mana = 44.0)
    primeval_stones INTEGER DEFAULT 0,  -- Số lượng Nguyên Thạch đang có
    last_mana_update TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP, -- Dùng để tính toán lượng mana hồi bù (Lazy Evaluation)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

--------------------------------------------------------------------------------
### 1. Database & Golang Models
Create the PostgreSQL schema and Golang structs for the `Player` model.
- Fields: id (UUID), username, rank (int, default 1), sub_rank (int, default 1), aptitude_type (string), aptitude_percent (int), current_mana (float), max_mana (float), primeval_stones (int), last_mana_update (timestamp).
- Rule: `max_mana` is always equal to `aptitude_percent`.

### 2. Backend Logic (Golang) - Player Creation API
Create an endpoint `POST /api/player/create` with a payload of `{username}`.
- Logic: Randomize the player's aptitude. 
  - 10% chance for 'Giap' (Aptitude A): random 80-99.
  - 20% chance for 'At' (Aptitude B): random 60-79.
  - 30% chance for 'Binh' (Aptitude C): random 40-59.
  - 40% chance for 'Dinh' (Aptitude D): random 20-39.
- Set `max_mana` = aptitude_percent, and `current_mana` = `max_mana` initially. Save to DB.

### 3. Backend Logic (Golang) - Mana Regeneration (Lazy Evaluation)
We need a passive mana regeneration system. Instead of a cron job updating the DB every second, use Lazy Evaluation.
Create an endpoint `GET /api/player/status/{id}`.
- Logic: When called, calculate the time difference (in seconds) between `time.Now()` and `last_mana_update`.
- Regeneration rates per hour based on aptitude_type: 
  - 'Giap': 8.0 mana / hour
  - 'At': 6.0 mana / hour
  - 'Binh': 4.0 mana / hour
  - 'Dinh': 2.0 mana / hour
- Calculate earned mana: `(seconds_passed / 3600.0) * regen_rate`.
- `new_mana` = `current_mana` + earned mana. Cap it at `max_mana`.
- Update `current_mana` and `last_mana_update` in the DB, then return the updated Player object.

### 4. Backend Logic (Golang) - Explore Action
Create an endpoint `POST /api/player/explore/{id}`.
- Logic: This action costs 10.0 mana. 
- First, trigger the Lazy Mana Evaluation (from step 3) to get the most accurate `current_mana`.
- If `current_mana` < 10.0, return a 400 error: "Not enough Primeval Essence".
- If sufficient, deduct 10.0 from `current_mana`.
- Randomize a reward: give the player between 1 to 3 `primeval_stones`.
- Save to DB and return the updated stats and a combat log message (e.g., "Explored the mountain and found 2 Primeval Stones!").

### 5. Frontend (Next.js + TailwindCSS + Zustand)
Build a simple Dashboard UI for the player.
- Use Zustand to manage the player state.
- Create an "Aperture" (Không Khiếu) component: A vertical progress bar representing the Mana Sea. The color of the bar should be "green" (representing Rank 1 Copper Mana). Show text like "Mana: 25/44".
- Create a "Player Stats" card showing Username, Rank, Sub-rank, Aptitude Type, and Primeval Stones.
- Implement a `useInterval` hook or React Query polling to call `GET /api/player/status/{id}` every 5 seconds so the user sees their mana bar slowly filling up in real-time.
- Create an "Explore" button that calls `POST /api/player/explore/{id}`. Add a loading state to prevent spam clicking. Show a toast notification with the reward text.

Please generate the code modularly. Start with the Golang Backend (Models, DB connection, Handlers), then provide the Next.js Frontend code.