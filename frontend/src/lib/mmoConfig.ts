/** Public runtime configuration for the opt-in Phaser MMO slice. */
const value = (raw: string | undefined): string => (raw ?? "").trim();
const enabled = (raw: string | undefined): boolean => value(raw).toLowerCase() === "true";

// These release versions are fixed in the shipped map artifacts. Keep them
// aligned with backend/internal/realtime/config/config.go.
const mapVersion = "regions-v1";
const contentVersion = "region-content-v1";

export const mmoClientConfig = {
  worldEnabled: enabled(process.env.NEXT_PUBLIC_REALTIME_WORLD_ENABLED),
  phaserClientEnabled: enabled(process.env.NEXT_PUBLIC_PHASER_WORLD_CLIENT_ENABLED),
  zoneId: value(process.env.NEXT_PUBLIC_MMO_ZONE_ID),
  mapVersion,
  contentVersion,
  mapUrl: value(process.env.NEXT_PUBLIC_MMO_MAP_URL),
} as const;

/** Feature flags control admission; map/content versions follow the shipped release. */
export const canJoinMMOZone = mmoClientConfig.worldEnabled && mmoClientConfig.phaserClientEnabled;
