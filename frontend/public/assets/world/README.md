# World map art

These original game art assets were generated for Cổ Giới and converted to WebP for the browser build.

- `ground-moss.webp`: repeating overhead moss and leaf terrain.
- `cultivator.webp`: default player character when the account has no profile avatar.
- `spirit-beast.webp`: shared field-monster sprite, including a light tint for boss variants.
- `spirit-herb.webp`: shared gatherable resource sprite.
- `items/`: representative resource sprites used on both maps: `crystal.svg` for materials/minerals, `food.svg` for food, and `gourd.svg` for wine and consumables. Herb resources reuse `spirit-herb.webp`.
- `great-regions-atlas.png`: generated terrain panels in this order: Nam Cương, Trung Châu, Bắc Nguyên, Tây Mạc, Đông Hải, Hắc Thiên, Bạch Thiên. The map renderer crops and places them inside their matching `NGU_VUC` bounds.
- `monster-atlas-common.png`: first generated 4×4 monster batch, covering the 16 most frequently spawned templates in current data.
- `monsters/monster-atlas-batch-01.webp` through `monster-atlas-batch-12.webp`: generated transparent 4×4 sprite atlases for the remaining monster templates. `monster-atlas-batch-13.webp` contains the final template. Their row-major ID mapping is kept in `frontend/src/components/monsterAtlasManifest.ts`.

The map renderer keeps its vector markers as a fallback when a sprite cannot be loaded.
