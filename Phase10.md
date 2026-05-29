CULTIVATION & ASCENSION SYSTEM (LEVELING MECHANICS)

You are an expert Full-stack Game Developer (Golang, PostgreSQL, Next.js, Zustand). 
In our "Reverend Insanity" text-based RPG, leveling up is NOT based on EXP points. It is a resource-intensive and hardcore survival process based on "Aperture Wall Breaking" for Mortals, and "Tribulation Surviving" for Immortals.

Please implement the following Cultivation System step-by-step:

### TASK 1: Database & Model Updates (PostgreSQL & Golang)
Update the `Player` model to track detailed cultivation stats:
- `rank` (INT): 1 to 9.
- `sub_rank` (INT): 1 (Initial), 2 (Middle), 3 (Upper), 4 (Peak).
- `aptitude` (INT): 1 to 4 (A, B, C, D grade. Determines Max Mana).
- `human_qi` (FLOAT): Accumulated hidden stat based on achievements/battles (used for Immortal Ascension).
- `dao_marks` (JSONB): e.g., `{"Strength": 1500, "Time": 200}`. Gained after Rank 6.
- `next_tribulation_time` (TIMESTAMP): For Immortals (Rank 6+).
- `tribulation_count` (INT): Tracks how many Earth/Heavenly/Grand tribulations the player has passed.

### TASK 2: Mortal Cultivation API (Rank 1 to 5) - "Trùng Kích Vách Khiếu"
Create endpoints for Mortals to level up:
**1. POST `/api/cultivate/break-wall`**
- **Logic:** The player spends a massive amount of `CurrentMana` and `PrimevalStones` to attempt moving to the next `sub_rank`. 
- **Calculation:** Success rate is heavily dependent on their `aptitude`. If D-grade (4), cost is extremely high and success rate is low without constant stone consumption. 
- **Result:** On success, `sub_rank += 1`. If `sub_rank` > 4, `rank += 1` and `sub_rank = 1`. Update Max Mana and Mana color.

**2. POST `/api/cultivate/use-relic-gu`**
- **Logic:** Player consumes a `Relic Gu` (Cổ Xá Lợi) matching their current `rank` (e.g., Green Copper Relic Gu for Rank 1).
- **Result:** Instantly increases `sub_rank += 1` with 100% success and 0 Mana cost. Consume the item.

### TASK 3: Immortal Ascension & Tribulation API (Rank 6 to 9) - "Độ Kiếp"
Create the backend logic for Immortals:
**1. POST `/api/cultivate/ascension`**
- **Logic:** Only available at Rank 5 Peak. Calculates the balance of Heaven, Earth, and the player's `human_qi`. If `human_qi` is too low, the player dies (Permadeath/Heavy penalty). If successful, `rank = 6`, `sub_rank = 1`, and generate a Blessed Land (Tiên Khiếu) for the player. Convert mana to Immortal Essence.

**2. POST `/api/cultivate/tribulation`**
- **Logic:** Runs when `next_tribulation_time` is reached. Triggers a massive PvE combat event (Earth Calamity / Heavenly Tribulation).
- **Result:** If the player survives, increase `tribulation_count`, grant a massive boost in `dao_marks` based on the tribulation type, and calculate the timestamp for the next tribulation (e.g., 10 days in real-time). If they die, their Immortal Aperture is destroyed.

### TASK 4: Frontend UI/UX Implementation (Next.js)
Update the `PlayerProfile` component:
- **For Mortals (Rank 1-5):** Build an "Aperture Meditation" interface. Show a glowing crystal wall. Add a "Charge Wall (Cost: X Mana)" button. Add a quick-use slot for "Relic Gu".
- **For Immortals (Rank 6+):** The Meditation UI completely changes to a "Blessed Land Management" UI. Display a grim, ticking countdown timer: "Next Earth Calamity in: [DD:HH:MM:SS]". Show their accumulated Dao Marks clearly.

Ensure all API calls use transactions to prevent race conditions during item consumption.