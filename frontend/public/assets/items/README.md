# Inventory item icons

The 64 transparent WebP icons in `icons/` are 96×96 pixels and total about 290 KB. They form a shared visual vocabulary for the 853 entries currently in `backend/data/Item.csv`; they are not 853 individually painted pictures. `frontend/src/lib/item-icons.ts` chooses a silhouette from each item's Vietnamese name, giving physical nouns (pelt, feather, fang, scale, bone, herb, and so on) priority over its elemental theme. The existing rank-colored inventory card indicates rank. All catalog items resolve to an existing file.

`preview.webp` is a contact sheet for visual review; the frontend loads the individual icon files.

The bag, item wiki, and Caravan Shop load only displayed thumbnails through browser lazy loading. A valid item `image_url` still overrides this bundle. An empty or malformed value such as the current `200.0` entries falls back to the bundled icon. The icon bundle does not require a CSV import or backend restart.

The icons were created with the built-in image generation tool, then cropped and downscaled. The four prompts requested consistent painterly cultivation RPG inventory icons, transparent backgrounds, isolated objects in regular 4×4 grids, clear silhouettes at 64 pixels, dark jewel-toned materials, warm gold highlights, and no text or frames. The row-major subjects were:

1. stone, crystal, herb, flower; root, mushroom, feather, pelt; fang, claw, scale, bone; horn, shell, blood, eye.
2. ore, ingot, sand, wood; leaf, seed, fruit, fish; insect, pearl, jade, ember; ice, lightning, shadow, light.
3. pill, potion, salve, chest; egg, scroll, talisman, coin; gourd, food, wine, incense; token, map, key, vial.
4. blade, armor, heart, spirit-flame; cloud, water, wind, hourglass; star, moon, sun, brush; chain, bell, gear, dao-seal.
