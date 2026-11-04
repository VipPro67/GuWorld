import { CORE_VFX_ATLAS_KEYS, WORLD_IMAGE_URLS } from "./AssetManifest.ts";
import { GU_VFX_CATALOG } from "../combat/GuVFXRegistry.ts";

const warmedImages = new Map<string, HTMLImageElement>();

/** Start browser image requests while the dictionary and session are loading. */
export function warmupWorldAssets(): void {
  if (typeof window === "undefined" || typeof Image === "undefined") return;

  const urls = new Set<string>([
    ...WORLD_IMAGE_URLS,
    ...CORE_VFX_ATLAS_KEYS.flatMap((key) => {
      const definition = GU_VFX_CATALOG[key];
      return definition ? [definition.textureUrl] : [];
    }),
  ]);

  for (const url of urls) {
    if (warmedImages.has(url)) continue;
    const image = new Image();
    image.decoding = "async";
    image.src = url;
    warmedImages.set(url, image);
  }
}
