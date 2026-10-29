/** Public runtime configuration for the Phaser MMO world. */
const value = (raw: string | undefined): string => (raw ?? "").trim();

// These release versions are fixed in the shipped map artifacts. Keep them
// aligned with backend/internal/realtime/config/config.go.
const mapVersion = "regions-v1";
const contentVersion = "region-content-v1";

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
