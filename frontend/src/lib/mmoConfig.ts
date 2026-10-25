/** Public runtime configuration for the opt-in Phaser MMO slice. */
const value = (raw: string | undefined): string => (raw ?? "").trim();

export const mmoClientConfig = {
  zoneId: value(process.env.NEXT_PUBLIC_MMO_ZONE_ID),
  mapVersion: value(process.env.NEXT_PUBLIC_MMO_MAP_VERSION),
  contentVersion: value(process.env.NEXT_PUBLIC_MMO_CONTENT_VERSION),
  mapUrl: value(process.env.NEXT_PUBLIC_MMO_MAP_URL),
} as const;

/** Admission needs an exact zone/map/content tuple; otherwise stay in preview. */
export const canJoinMMOZone =
  mmoClientConfig.mapVersion.length > 0 &&
  mmoClientConfig.contentVersion.length > 0;
