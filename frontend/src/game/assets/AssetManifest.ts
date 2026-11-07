/** Content-addressed manifest for the placeholder combat art set. */
export const ASSET_VERSION = "placeholder-1";

export type ActorKind = "player" | "monster" | "npc";
export type ActorAnimation = "idle" | "walk" | "cast" | "hit" | "death";

export interface ActorAsset {
  kind: ActorKind;
  label: string;
  license: string;
  fallbackTexture: string;
  animations: Record<ActorAnimation, number>;
}

export const ANIMATION_STATES: ActorAnimation[] = ["idle", "walk", "cast", "hit", "death"];

/** Images needed for the first Phaser world load. Shared by preload and boot warmup. */
export const WORLD_IMAGE_URLS = [
  "/assets/world/ground-moss.webp",
  "/assets/world/cultivator.webp",
  "/assets/world/spirit-beast.webp",
] as const;

export const CORE_VFX_ATLAS_KEYS = [
  "fx_hit_chem", "fx_hit_dam", "fx_hit_dien", "fx_khien_anh_kim",
  "fx_thu_giap_da_gpt", "fx_thu_vo_giap_gpt", "fx_hieu_ung_choang",
  "fx_hieu_ung_doc", "fx_hieu_ung_troi", "fx_hoi_sinh", "fx_gu_hoa_du",
  "fx_gu_thiem_quang", "fx_gu_dien_nhan_gpt_dau", "fx_gu_thinh_phong_gpt_kich_hoat",
  "fx_gu_xa_tin_dau_gpt_giu", "fx_thu_set_danh_gpt", "fx_song_gam",
  "fx_dat_nut", "fx_gu_nguyet_quang", "fx_gu_huyet_nguyet", "fx_gu_huyet_bao",
  "fx_gu_bach_cot", "fx_thoi_trang_xuan_thu_quyen",
] as const;

export const ACTOR_ASSETS: Record<ActorKind, ActorAsset> = {
  player: {
    kind: "player",
    label: "Player token",
    license: "procedural placeholder (project-owned)",
    fallbackTexture: "token-player",
    animations: { idle: 2, walk: 4, cast: 2, hit: 1, death: 1 },
  },
  monster: {
    kind: "monster",
    label: "Monster token",
    license: "procedural placeholder (project-owned)",
    fallbackTexture: "token-monster",
    animations: { idle: 2, walk: 4, cast: 2, hit: 1, death: 1 },
  },
  npc: {
    kind: "npc",
    label: "NPC token",
    license: "procedural placeholder (project-owned)",
    fallbackTexture: "token-npc",
    animations: { idle: 2, walk: 2, cast: 1, hit: 1, death: 1 },
  },
};

export interface VfxAsset {
  key: string;
  label: string;
  license: string;
  fallback: "circle";
}

export const VFX_ASSETS: VfxAsset[] = [
  { key: "vfx/impact", label: "Impact burst", license: "procedural placeholder (project-owned)", fallback: "circle" },
  { key: "vfx/cast", label: "Cast ring", license: "procedural placeholder (project-owned)", fallback: "circle" },
  { key: "vfx/status", label: "Status tick", license: "procedural placeholder (project-owned)", fallback: "circle" },
  { key: "vfx/death", label: "Death puff", license: "procedural placeholder (project-owned)", fallback: "circle" },
];

export function animationTextureKey(kind: ActorKind, animation: ActorAnimation, frame: number): string {
  return `${kind}-${animation}-${frame}`;
}

export function animationKey(kind: ActorKind, animation: ActorAnimation): string {
  return `${kind}-${animation}`;
}

export function animationFrameKeys(kind: ActorKind, animation: ActorAnimation): string[] {
  const count = ACTOR_ASSETS[kind].animations[animation];
  return Array.from({ length: count }, (_, index) => animationTextureKey(kind, animation, index));
}

export function animationFrameRate(animation: ActorAnimation): number {
  switch (animation) {
    case "walk":
      return 8;
    case "cast":
      return 6;
    case "hit":
      return 12;
    case "death":
      return 5;
    default:
      return 4;
  }
}

/** Pure validation used by tests and the preload scene. */
export function validateAssetManifest(): string[] {
  const errors: string[] = [];
  for (const kind of Object.keys(ACTOR_ASSETS) as ActorKind[]) {
    const asset = ACTOR_ASSETS[kind];
    if (!asset.license) errors.push(`${kind}: missing license`);
    if (!asset.fallbackTexture) errors.push(`${kind}: missing fallback texture`);
    for (const animation of ANIMATION_STATES) {
      const count = asset.animations[animation];
      if (!Number.isInteger(count) || count < 1 || count > 16) {
        errors.push(`${kind}.${animation}: invalid frame count ${count}`);
      }
    }
  }
  const keys = new Set<string>();
  for (const asset of Object.values(ACTOR_ASSETS)) {
    for (const animation of ANIMATION_STATES) {
      for (const key of animationFrameKeys(asset.kind, animation)) {
        if (keys.has(key)) errors.push(`duplicate texture key ${key}`);
        keys.add(key);
      }
    }
  }
  for (const vfx of VFX_ASSETS) {
    if (!vfx.key || !vfx.license) errors.push(`invalid VFX asset ${vfx.key}`);
  }
  return errors;
}
