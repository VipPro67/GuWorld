# Inventory item icons

The 70 transparent WebP icons in `icons/` are 96×96 pixels and total about 350 KB. They form a shared visual vocabulary for the 853 entries currently in `backend/data/Item.csv`; they are not 853 individually painted pictures. `frontend/src/lib/item-icons.ts` chooses a silhouette from each item's Vietnamese name, giving physical nouns (pelt, feather, fang, scale, bone, herb, silk, pouch, shard, clay, formation, core, and so on) priority over its elemental theme. The existing rank-colored inventory card indicates rank. All catalog items resolve to an existing file.

`preview.webp` is a contact sheet for visual review; the frontend loads the individual icon files.

The bag, item wiki, Caravan Shop, Bao Hoang Market, Trade modal, Refining furnace, Sect vault, and Exploration map load only displayed thumbnails through browser lazy loading. A valid item `image_url` still overrides this bundle. An empty or malformed value falls back to the bundled icon. The icon bundle does not require a CSV import or backend restart.

The icons were created with the built-in image generation tool, then cropped and downscaled. The prompts requested consistent painterly cultivation RPG inventory icons, transparent backgrounds, isolated objects, clear silhouettes at 64-96 pixels, dark jewel-toned materials, warm gold highlights, and no text or frames. The subjects are:

1. stone, crystal, herb, flower; root, mushroom, feather, pelt; fang, claw, scale, bone; horn, shell, blood, eye.
2. ore, ingot, sand, wood; leaf, seed, fruit, fish; insect, pearl, jade, ember; ice, lightning, shadow, light.
3. pill, potion, salve, chest; egg, scroll, talisman, coin; gourd, food, wine, incense; token, map, key, vial.
4. blade, armor, heart, spirit-flame; cloud, water, wind, hourglass; star, moon, sun, brush; chain, bell, gear, dao-seal.
5. silk, pouch, shard, clay, formation, core.
