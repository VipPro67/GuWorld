/** Public runtime configuration for the Phaser MMO world. */
const value = (raw: string | undefined): string => (raw ?? "").trim();

// These release versions are fixed in the shipped map artifacts. Keep them
// aligned with backend/internal/realtime/config/config.go.
const mapVersion = "regions-v1";
const contentVersion = "region-content-v1";
const starterMapVersion = "regions-v2";
const starterContentVersion = "landscape-v2";

const STARTER_MAP_IDS = new Set([
  "tan_thu_coc_nam_cuong",
  "tan_thu_coc_dong_hai",
  "tan_thu_coc_tay_mac",
  "tan_thu_coc_bac_nguyen",
  "tan_thu_coc_trung_chau",
]);

export function versionsForMMOZone(zoneId: string): { mapVersion: string; contentVersion: string } {
  const mapId = value(zoneId).split("#", 1)[0];
  if (mapId === "world_map") {
    return { mapVersion: "legacy-converted-v2", contentVersion: "legacy-regions-v2" };
  }
  if (STARTER_MAP_IDS.has(mapId)) {
    return { mapVersion: starterMapVersion, contentVersion: starterContentVersion };
  }
  return { mapVersion, contentVersion };
}

export const mmoClientConfig = {
  worldEnabled: true,
  phaserClientEnabled: true,
  zoneId: value(process.env.NEXT_PUBLIC_MMO_ZONE_ID),
  mapVersion,
  contentVersion,
  mapUrl: value(process.env.NEXT_PUBLIC_MMO_MAP_URL),
} as const;

/** The world is enabled for every signed-in player; versions follow the shipped release. */
export const canJoinMMOZone = mmoClientConfig.worldEnabled && mmoClientConfig.phaserClientEnabled;
