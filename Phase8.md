# COMPREHENSIVE IMPLEMENTATION PLAN: STATS, INVENTORY, EXPLORATION & COMBAT

You are an expert full-stack developer (Golang, PostgreSQL, Next.js, TailwindCSS, Zustand, Recharts/Chart.js). 
We are building a hardcore text-based RPG web game based on the "Reverend Insanity" universe. 
Please implement the following 4 core systems step-by-step based on our strict game design rules.

### MODULE 1: DYNAMIC PLAYER STATISTICS SYSTEM
Unlike traditional RPGs, players do not gain HP by leveling up. Power comes strictly from Gu bugs and Body Modifications.
**1. Backend (Golang/PostgreSQL):**
- Ensure `Player` struct has fields: `base_hp` (fixed 100), `body_mods` (JSONB for passive permanent buffs like "Iron Bone"), `path_masteries` (JSONB for multipliers).
**2. Frontend (Next.js):**
- Implement a **Hexagonal Radar Chart** (using Recharts or Chart.js) in the Player Profile.
- The 6 axes are: `Attack`, `Defense`, `Healing/Support`, `Movement`, `Investigation`, `Storage`.
- **Logic:** Calculate these 6 values dynamically by iterating through the player's *currently equipped Gu bugs* and *passive Body Mods*. 

### MODULE 2: HARDCORE INVENTORY MANAGEMENT
Inventory is strictly limited and acts as a survival mechanic.
**1. Backend (Golang):**
- Define a strict `MaxInventorySlots` for players (default: 20). 
- If a player equips a `Storage Gu` (e.g., Tusita Flower) or reaches Rank 6 (Immortal Aperture), dynamically increase `MaxInventorySlots`.
**2. Frontend (Next.js):**
- Build an optimized `InventoryUI` component. Since late-game players have massive inventories, ONLY render items that currently exist in the inventory (quantity > 0).
- **Filters & Search:** Implement a search bar to filter items by `Name`, `Use/Type`, and `Description`.
- **Gu Survival UI:** Every Gu card in the inventory MUST display a visual `Hunger Bar`. If hunger is low, show a warning red pulse effect.

### MODULE 3: GRID-BASED WORLD EXPLORATION
Exploration is a tactical movement system on a 100x100 grid, not just random clicking.
**1. Backend (Golang):**
- Create a `MapGrid` system (100x100) for regions like "Southern Border" or "Northern Plains". Each cell can contain: Nothing, Monster, Resource, or Gu.
**2. Movement & Vision Logic (Crucial):**
- **Fog of War:** By default, a player can only see 8 adjacent locations (1-cell radius).
- **Investigation Gu:** If the player activates an Investigation Gu (e.g., Earth Communication Ear Grass), expand their vision radius to 24+ cells (2 or 3-cell radius).
- **Movement Gu:** Normal movement costs 1 turn/mana per step. If an active Movement Gu is equipped, allow moving 2 or 3 steps per turn.
**3. Frontend (Next.js):**
- Render a minimap (canvas or CSS grid). Apply dark fog over unvisited cells. 

### MODULE 4: TURN-BASED ASYNCHRONOUS PVE/PVP COMBAT
Combat is about "Mind Games" (Predicting the opponent) and Mana Attrition.
**1. Backend Combat Engine:**
- Implement a `ResolveSimultaneousTurn` function. Both Player and Target submit their planned actions hidden from each other, then resolve them simultaneously.
- **Clash Mechanic:** If Player A and Player B both use Ranged Attack Gu, calculate damage subtraction. If A's attack > B's, A's attack pierces through with remaining damage.
- **Mana Attrition:** Every active Gu usage deducts from the Current Mana pool. If Mana drops to 0, abilities fail.
**2. Frontend Combat UI:**
- Build a "Preparation Phase" UI where the player selects their Gu sequence for the turn (Predicting the enemy).
- Build a "Resolution Phase" UI: A scrollable text-based combat log printing out the simultaneous actions (e.g., "You fired a Moonblade, but the enemy dodged using Stealth Scales Gu!").
- Add a "Gouge/Charge" button for Killer Moves that require multiple turns to cast.

Please structure the backend endpoints cleanly and create modular, reusable React components for the UI. Start with the Database & Golang logic for Player Stats and Inventory first.
