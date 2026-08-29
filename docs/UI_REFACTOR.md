Refactor the frontend UI architecture of my GuWorld game to create a consistent shared Design System.

Current architecture:
- Next.js + React + TypeScript
- Tailwind CSS
- The project currently has large feature components such as:
  - DungeonView.tsx
  - SectView.tsx
  - GuInventory.tsx
  - RefiningFurniture.tsx
  - ExplorationMap.tsx
  - GachaPanel.tsx
  - PremiumStoreView.tsx
  - TrainingRoom.tsx
  - TradeRoomModal.tsx
  - PetPanel.tsx
  - ImmortalApertureView.tsx
  - LeaderboardView.tsx
  - and others.
- These large components currently contain most of their own Tailwind/CSS styling.
- The result is that different screens look inconsistent even though they belong to the same game.

GOAL

Create a shared UI Design System without rewriting the game logic.

The main principle:

Feature components should contain game-specific layout and logic, while common visual styling should come from reusable UI components and design tokens.

TARGET STRUCTURE

components/
├── ui/
│   ├── GamePage.tsx
│   ├── GamePanel.tsx
│   ├── GameCard.tsx
│   ├── GameButton.tsx
│   ├── GameModal.tsx
│   ├── GameTabs.tsx
│   ├── GameBadge.tsx
│   ├── GameInput.tsx
│   ├── GameProgressBar.tsx
│   ├── GameSection.tsx
│   └── ...
│
├── game/
│   ├── GuCard.tsx
│   ├── ItemCard.tsx
│   ├── ResourceDisplay.tsx
│   ├── RealmDisplay.tsx
│   ├── RankBadge.tsx
│   └── ...
│
├── DungeonView.tsx
├── SectView.tsx
├── GuInventory.tsx
├── RefiningFurniture.tsx
└── ...

Do NOT blindly create components for every small element.

Only extract components when:
1. They are reused.
2. They represent a common visual pattern.
3. They need consistent styling across multiple screens.

DESIGN SYSTEM

First inspect the existing frontend and identify:
- Current colors
- Background colors
- Surface colors
- Border colors
- Text colors
- Accent colors
- Border radius
- Shadows
- Spacing
- Typography
- Button styles
- Card styles
- Modal styles
- Tabs
- Inputs
- Progress bars
- Badges
- Hover/active/disabled states
- Dark/light theme behavior

Then create a centralized Design Token system.

For example:

--game-bg
--game-surface
--game-surface-elevated
--game-border
--game-primary
--game-secondary
--game-success
--game-warning
--game-danger
--game-text
--game-text-muted
--game-radius-sm
--game-radius-md
--game-radius-lg
--game-shadow

Do not blindly use these exact values. Analyze the existing UI first and choose values that preserve the current GuWorld visual identity.

IMPORTANT

Do not replace the existing visual identity with a generic shadcn/ui, Material UI, Bootstrap, or generic dashboard design.

GuWorld is a cultivation/fantasy game.

The UI should feel like ONE coherent cultivation game:
- dark fantasy
- mystical
- premium
- readable
- game-like
- consistent
- not overly flashy
- not like a business dashboard

COMMON COMPONENTS

Create reusable components for common patterns.

GamePanel:
- Main container
- Consistent background
- Consistent border
- Consistent radius
- Consistent padding
- Consistent shadow

GameCard:
- Smaller content container
- Used for items, Gu, characters, rewards, etc.

GameButton:
Variants should include appropriate states such as:
- primary
- secondary
- danger
- ghost
- disabled
- loading

GameModal:
- Consistent overlay
- Consistent panel
- Header
- Content
- Footer
- Close button
- Responsive behavior

GameTabs:
- Consistent active state
- Hover state
- Disabled state

GameBadge:
- Used for status, rank, rarity, realm, etc.

GameProgressBar:
- HP
- Mana
- cultivation progress
- refinement progress
- experience
- other game resources

GameSection:
- Consistent section title
- Optional description
- Optional action area

GamePage:
- Standard page-level spacing
- Page title
- Page description
- Content layout

GAME-SPECIFIC COMPONENTS

Create reusable game components where appropriate:

GuCard
ItemCard
ResourceDisplay
RealmDisplay
RankBadge
DaoMarkDisplay
PlayerCard
RewardCard

These should use the primitive components from components/ui instead of implementing their own visual system.

MIGRATION STRATEGY

Do NOT rewrite everything at once.

1. Inspect the current codebase.
2. Identify repeated UI patterns.
3. Build the Design Tokens.
4. Build the core UI components.
5. Migrate the largest/reused components first.
6. Preserve existing game logic.
7. Preserve API calls.
8. Preserve state management.
9. Preserve event handlers.
10. Preserve routing.
11. Preserve responsive behavior.
12. Preserve dark/light theme functionality.
13. Remove duplicated styling only after the shared component is working.

Prioritize migration roughly in this order:

1. GamePanel
2. GameCard
3. GameButton
4. GameModal
5. GameTabs
6. GameBadge
7. GameProgressBar
8. GameSection
9. GamePage
10. Game-specific components
11. Feature components

IMPORTANT CODE RULES

- Do not modify backend code.
- Do not change API contracts.
- Do not change database logic.
- Do not change game mechanics.
- Do not change business logic unless absolutely necessary for the UI refactor.
- Do not remove existing functionality.
- Do not duplicate components unnecessarily.
- Do not introduce a large UI framework just for this refactor.
- Prefer the existing Tailwind setup.
- Keep TypeScript strongly typed.
- Keep components composable.
- Avoid excessive abstraction.
- Avoid creating a component for a one-off element.
- Avoid hardcoded colors scattered throughout feature components.

THEME

The game currently supports dark/light themes.

The Design System must support both themes through centralized tokens.

Changing the theme should primarily change tokens rather than requiring every feature component to be rewritten.

CONSISTENCY RULE

After the refactor, these screens should clearly look like they belong to the same game:

- Dungeon
- Sect
- Inventory
- Refinement
- Exploration
- Gacha
- Premium Store
- Training
- Trade
- Pet
- Immortal Aperture
- Leaderboard
- Lobby

They can have different layouts because their functionality is different, but they must share:
- visual hierarchy
- colors
- borders
- radius
- typography
- buttons
- cards
- panels
- modals
- tabs
- spacing
- interaction states

QUALITY CHECK

Before finishing:

1. Search for duplicated Tailwind styling patterns.
2. Identify places where feature components still implement common UI patterns themselves.
3. Replace those patterns with shared components where appropriate.
4. Check dark mode.
5. Check light mode.
6. Check mobile/responsive layouts.
7. Check hover/focus/disabled states.
8. Make sure no functionality was lost.
9. Run TypeScript/build/lint/tests if available.
10. Fix any issues caused by the refactor.

Do not perform a massive rewrite.

The final architecture should make it easy to create a new GuWorld screen using:

<GamePage>
    <GamePanel>
        <GameSection>
            <GameCard>
                ...
            </GameCard>
        </GameSection>
    </GamePanel>
</GamePage>

The goal is that future screens automatically inherit the same GuWorld visual language instead of each developer creating a new CSS style from scratch.