# World map art

These original game art assets were generated for Cổ Giới and converted to WebP for the browser build.

- `ground-moss.webp`: repeating overhead moss and leaf terrain.
- `cultivator.webp`: default player character when the account has no profile avatar.
- `spirit-beast.webp`: shared field-monster sprite, including a light tint for boss variants.
- `spirit-herb.webp`: shared gatherable resource sprite.
- `great-regions-atlas.png`: generated terrain panels in this order: Nam Cương, Trung Châu, Bắc Nguyên, Tây Mạc, Đông Hải, Hắc Thiên, Bạch Thiên. The map renderer crops and places them inside their matching `NGU_VUC` bounds.
- `monster-atlas-common.png`: first generated monster batch, cropped row-major using `COMMON_MONSTER_ATLAS_IDS` in `WorldCanvas.tsx`. The batch covers the 16 most frequently spawned templates in current data.

The map renderer keeps its vector markers as a fallback when a sprite cannot be loaded.
