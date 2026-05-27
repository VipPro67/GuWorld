# GAME MECHANICS UPDATE: VITAL GU vs REGULAR GU vs CONSUMABLE GU vs PASSIVE GU

You are an expert full-stack developer. We need to fix a critical logic flaw in our "Reverend Insanity" RPG game. Currently, the system treats all Gu bugs the same. In the lore, there are strict distinctions between different states and types of Gu. 

Please read this design document carefully and update the Backend (Golang) and Frontend (Next.js) accordingly.

## 1. CORE CONCEPTS & DEFINITIONS

### A. Cổ Bản Mệnh (Vital Gu)
- **Definition:** The very first Gu bug a Gu Master refines becomes their Vital Gu, which determines their future cultivation path [1, 2]. 
- **Life-Linked:** Its life is directly tied to the Gu Master. If the Vital Gu dies, the Gu Master suffers severe backlash and heavy injuries [1, 3].
- **The Ultimate Privilege (Crucial for Crafting):** The greatest advantage of a Vital Gu is that **it cannot be destroyed during a failed refinement process** [3, 4]. If crafting fails, regular Gu bugs perish, but the Vital Gu will survive (it only enters a weakened/dying state) [4].
- **Rule:** A player can only have **ONE** Vital Gu at a time. It can be replaced, but destroying the current Vital Gu causes massive HP damage to the player [5].

### B. Cổ Thường (Regular Gu / Mortal Gu)
- **Definition:** Any standard Gu bug (Rank 1 to 5) that the player refines after their first one. 
- **Rule:** If used in refinement (crafting) and the process fails, these Gu bugs are **permanently destroyed and lost forever** [4]. 
- **Rule:** If they starve to death, they just disappear from the inventory without injuring the player.

### C. Cổ Tiêu Hao (Consumable Gu)
- **Definition:** Gu bugs that disappear immediately after one use [6].
- **Examples:** *Xá Lợi cổ* (Relic Gu) used to instantly raise cultivation levels [7], or *Thạch Khiếu cổ* (Stone Aperture Gu) [6].
- **Rule:** Once the "Use" button is clicked, trigger their effect and delete them from the inventory.

# D. PASSIVE GU (Auto-Effect Gu)

### Definition
Passive Gu are Gu bugs that provide continuous or automatically triggered effects while they are active in the player's inventory, aperture, or equipped slots.

Unlike Consumable Gu, they are NOT destroyed after activation.
Unlike active skill Gu, they do NOT require manual button presses every time.

They function similarly to:
- Passive skills
- Auras
- Auto-trigger effects
- Stat modifiers

### Examples
- Strength Enhancement Gu (+10 STR permanently while equipped)
- Regeneration Gu (recover 1% HP every 10 seconds)
- Luck Gu (small bonus to refinement success rate)
- Iron Skin Gu (reduce incoming physical damage)
- Qi Detection Gu (auto-detect nearby enemies/resources)

---

## 4. PASSIVE GU RULES

### A. Persistent Effects
Passive effects remain active as long as:
- The Gu is alive
- The Gu is equipped/activated
- The player has enough essence or conditions to maintain it( like equip enough time require)

### B. No Manual Consumption
Passive Gu are NOT consumed after triggering.
They only disappear if:
- Destroyed during refinement
- Starved to death
- Explicitly removed/deleted
- Killed by special mechanics

### C. Toggleable State
Some Passive Gu may support ON/OFF states.

Example:
```json
{
  "instance_id": "uuid",
  "gu_id": "GU_REGENERATION",
  "type": "PASSIVE",
  "is_active": true
}
---

## 2. BACKEND IMPLEMENTATION PLAN (GOLANG)

### Step 1: Update Player Inventory Schema
In the `players.gu_inventory` JSONB, we must add a boolean flag to track the Vital Gu.
```json
// Example of an item in gu_inventory
{
  "instance_id": "uuid",
  "gu_id": "GU_MOONLIGHT",
  "is_vital": true, // ONLY ONE GU CAN BE TRUE
  "hunger": 100,
  "status": "ACTIVE" // can be ACTIVE, DYING, DEAD
}
Step 2: Refactoring the Refinement (Crafting) API
Update POST /api/refine to respect the Vital Gu privilege.
When math/rand determines a Refinement Failure:
Loop through the consumed selected_gu_instance_ids.
If a Gu is is_vital == false -> DELETE IT from the database
.
If a Gu is is_vital == true -> DO NOT DELETE IT. Instead, set its hunger = 1 and status = "DYING" (representing the weakened state)
.
Return JSON: { status: "failed", message: "Refinement failed. Regular materials perished, but your Vital Gu survived in a weakened state." }
Step 3: Refactoring the Death & Starvation Worker
Update the Background Worker that decreases Gu hunger.
If a Regular Gu (is_vital == false) reaches 0 hunger and dies -> Delete it.
If a Vital Gu (is_vital == true) dies -> Delete it AND deduct 80 HP from the Player. If Player HP <= 0, trigger Player Death sequence
.
Step 4: Endpoint to Set/Change Vital Gu
Create POST /api/inventory/set-vital
Payload: { "player_id": "uuid", "gu_instance_id": "uuid" }
Logic:
If the player already has a Vital Gu, warn them that replacing it will cost 50 HP (Backlash).
Update the chosen Gu's is_vital to true, and set all others to false.

--------------------------------------------------------------------------------
3. FRONTEND IMPLEMENTATION PLAN (NEXT.JS)
Step 1: Visual Distinction in Inventory
In the <GuInventory /> component, if a Gu card has is_vital === true, give it a glowing golden border or a prominent "VITAL" badge.
This helps players instantly know which Gu is protecting their refinement process.
Step 2: Crafting UI Updates
In the <RefiningFurnace />, when a player drags a Vital Gu into a crafting slot, show a tooltip: "Vital Gu Privilege: This Gu will not be destroyed if refinement fails."
Step 3: Consumable Usage
For Gu types marked as Consumable (e.g., Relic Gu), change the primary action button from "Equip" to "Use (Consume)".

***

**Giải thích thêm cho bạn (Game Design Note):**
Sự nhầm lẫn của AI Agent trước đó sẽ tạo ra một game quá "khổ dâm" (hardcore) nếu mọi Cổ trùng đều bị phá hủy khi đập đồ xịt. Trong nguyên tác, **Cổ Bản Mệnh** chính là "phao cứu sinh" lớn nhất của người chơi. Tỷ lệ thất bại khi luyện cổ (nhất là từ Rank 4 lên Rank 5) là cực kỳ độc ác [4], nhưng nhờ tính mạng Cổ bản mệnh được giữ lại, người chơi mới không bị mất trắng thành quả và có động lực cày cuốc lại từ đầu [4]. File prompt này đã giải thích logic đó bằng ngôn ngữ lập trình (if `is_vital == true` thì không delete) để AI có thể code chính xác 100%.