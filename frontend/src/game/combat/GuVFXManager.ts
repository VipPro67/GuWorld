import Phaser from "phaser";
import { GU_VFX_CATALOG, resolveGuVfx, resolveSkillVfxKey, resolveVisualTemplate, type VfxDefinition } from "./GuVFXRegistry.ts";
import type { WorldSkillVisuals } from "../network/Protocol.ts";

const MAX_ACTIVE_VFX = 48;

export interface PlayVfxOptions {
  scale?: number;
  rotation?: number;
  tint?: number;
  depth?: number;
  alpha?: number;
  flipX?: boolean;
  blendMode?: Phaser.BlendModes | string;
  onComplete?: () => void;
}

export class GuVFXManager {
  private readonly scene: Phaser.Scene;
  private readonly spritePool: Phaser.GameObjects.Sprite[] = [];
  private readonly loadedAtlases = new Set<string>();
  private readonly loadingAtlases = new Set<string>();
  private activeCount = 0;

  constructor(scene: Phaser.Scene) {
    this.scene = scene;
  }

  /**
   * Preload the core combat and Gu signature atlases in Phaser.
   */
  public preloadCoreAtlases(): void {
    const coreKeys: string[] = [
      "fx_hit_chem",
      "fx_hit_dam",
      "fx_hit_dien",
      "fx_khien_anh_kim",
      "fx_thu_giap_da_gpt",
      "fx_thu_vo_giap_gpt",
      "fx_hieu_ung_choang",
      "fx_hieu_ung_doc",
      "fx_hieu_ung_troi",
      "fx_hoi_sinh",
      "fx_gu_hoa_du",
      "fx_gu_thiem_quang",
      "fx_gu_dien_nhan_gpt_dau",
      "fx_gu_thinh_phong_gpt_kich_hoat",
      "fx_gu_xa_tin_dau_gpt_giu",
      "fx_thu_set_danh_gpt",
      "fx_song_gam",
      "fx_dat_nut",
      "fx_gu_nguyet_quang",
      "fx_gu_huyet_nguyet",
      "fx_gu_huyet_bao",
      "fx_gu_bach_cot",
      "fx_thoi_trang_xuan_thu_quyen",
    ];

    for (const key of coreKeys) {
      this.loadAtlas(key);
    }
  }

  /**
   * Queues an atlas for loading if not already loaded or loading.
   */
  public loadAtlas(key: string, onReady?: () => void): void {
    if (this.loadedAtlases.has(key) || this.scene.textures.exists(key)) {
      this.loadedAtlases.add(key);
      onReady?.();
      return;
    }
    if (this.loadingAtlases.has(key)) return;

    const def = GU_VFX_CATALOG[key];
    if (!def) return;

    this.loadingAtlases.add(key);
    this.scene.load.atlas(key, def.textureUrl, def.atlasUrl);
    this.scene.load.once(`filecomplete-atlasjson-${key}`, () => {
      this.loadingAtlases.delete(key);
      this.loadedAtlases.add(key);
      this.ensureAnimation(key, def);
      onReady?.();
    });

    if (!this.scene.load.isLoading()) {
      this.scene.load.start();
    }
  }

  /**
   * Spawns an animated VFX sprite from an atlas at (x, y).
   */
  public playVFX(key: string, x: number, y: number, options?: PlayVfxOptions): Phaser.GameObjects.Sprite | null {
    if (this.activeCount >= MAX_ACTIVE_VFX) return null;

    if (!this.loadedAtlases.has(key) && !this.scene.textures.exists(key)) {
      this.loadAtlas(key, () => {
        this.playVFX(key, x, y, options);
      });
      return null;
    }

    const def = GU_VFX_CATALOG[key];
    const animKey = this.ensureAnimation(key, def);
    if (!animKey) return null;

    const sprite = this.acquireSprite();
    if (!sprite) return null;

    this.activeCount += 1;
    const baseScale = def?.scale ?? 1.0;
    const finalScale = (options?.scale ?? 1.0) * baseScale;

    sprite.setActive(true)
      .setVisible(true)
      .setPosition(x, y)
      .setScale(finalScale)
      .setRotation(options?.rotation ?? 0)
      .setAlpha(options?.alpha ?? 1.0)
      .setDepth(options?.depth ?? y + 60)
      .setFlipX(options?.flipX ?? false);

    if (options?.tint !== undefined) {
      sprite.setTint(options.tint);
    } else {
      sprite.clearTint();
    }

    if (options?.blendMode !== undefined) {
      sprite.setBlendMode(options.blendMode);
    } else {
      sprite.setBlendMode(Phaser.BlendModes.NORMAL);
    }

    sprite.off(Phaser.Animations.Events.ANIMATION_COMPLETE);
    sprite.once(Phaser.Animations.Events.ANIMATION_COMPLETE, () => {
      sprite.setActive(false).setVisible(false);
      this.activeCount = Math.max(0, this.activeCount - 1);
      options?.onComplete?.();
    });

    sprite.play(animKey);
    return sprite;
  }

  /**
   * High-impact melee slash visual for basic attack.
   */
  public playHitSlash(x: number, y: number, directionX: number, directionY: number): void {
    const angle = Math.atan2(directionY, directionX);
    this.playVFX("fx_hit_chem", x, y, {
      rotation: angle,
      scale: 1.25,
      blendMode: Phaser.BlendModes.ADD,
    });
  }

  /**
   * Hit impact spark/burst when damage is dealt.
   */
  public playHitImpact(x: number, y: number, kind: "slash" | "blunt" | "shock" = "slash"): void {
    const key = kind === "shock" ? "fx_hit_dien" : kind === "blunt" ? "fx_hit_dam" : "fx_hit_chem";
    this.playVFX(key, x, y, {
      scale: 1.1,
      blendMode: Phaser.BlendModes.ADD,
    });
  }

  /**
   * Shield absorption sparkle when a shield takes damage.
   */
  public playShieldImpact(x: number, y: number): void {
    this.playVFX("fx_khien_anh_kim", x, y, {
      scale: 1.05,
      blendMode: Phaser.BlendModes.ADD,
    });
  }

  /**
   * Dramatic shield shatter explosion when depleted.
   */
  public playShieldBreak(x: number, y: number): void {
    this.playVFX("fx_thu_vo_giap_gpt", x, y, {
      scale: 1.2,
      depth: y + 90,
    });
  }

  /**
   * Signature Gu skill animation based on gu_id, rank, skill ID or name.
   */
  public playSkillVFX(
    skillId: string,
    skillName: string | undefined,
    x: number,
    y: number,
    guId?: string | null,
    rank?: number | null,
    visuals?: WorldSkillVisuals
  ): void {
    const resolved = resolveGuVfx(guId, rank, skillId, skillName);
    const key = resolveVisualTemplate(visuals?.cast_vfx) ?? resolved.vfxKey;
    this.playVFX(key, x, y, {
      scale: resolved.scale,
      depth: y + 70,
    });
  }

  public playImpactVFX(
    template: string | undefined,
    x: number,
    y: number,
    skillId = "",
    skillName?: string
  ): void {
    const key = resolveVisualTemplate(template) ?? resolveSkillVfxKey(skillId, skillName);
    this.playVFX(key, x, y, { scale: 1.05, depth: y + 72 });
  }

  public destroy(): void {
    for (const sprite of this.spritePool) {
      sprite.destroy();
    }
    this.spritePool.length = 0;
    this.loadedAtlases.clear();
    this.loadingAtlases.clear();
    this.activeCount = 0;
  }

  private acquireSprite(): Phaser.GameObjects.Sprite | null {
    const existing = this.spritePool.find((s) => !s.active);
    if (existing) return existing;

    if (this.spritePool.length >= MAX_ACTIVE_VFX) return null;

    const sprite = this.scene.add.sprite(0, 0, "__DEFAULT").setActive(false).setVisible(false);
    this.spritePool.push(sprite);
    return sprite;
  }

  private ensureAnimation(key: string, def?: VfxDefinition): string | null {
    const animKey = `${key}_anim`;
    if (this.scene.anims.exists(animKey)) return animKey;

    if (!this.scene.textures.exists(key)) return null;

    const texture = this.scene.textures.get(key);
    const frameNames = texture.getFrameNames().filter((f) => f !== "__BASE");
    if (frameNames.length === 0) return null;

    frameNames.sort((a, b) => {
      const numA = parseInt(a, 10);
      const numB = parseInt(b, 10);
      if (!Number.isNaN(numA) && !Number.isNaN(numB)) return numA - numB;
      return a.localeCompare(b);
    });

    const frames = frameNames.map((frame) => ({ key, frame }));
    this.scene.anims.create({
      key: animKey,
      frames,
      frameRate: def?.frameRate ?? 14,
      repeat: 0,
    });

    return animKey;
  }
}
