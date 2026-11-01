import Phaser from "phaser";
import {
  ACTOR_ASSETS,
  ANIMATION_STATES,
  VFX_ASSETS,
  animationTextureKey,
  type ActorAnimation,
  type ActorKind,
} from "../assets/AssetManifest.ts";
import { registerActorAnimations } from "../assets/AnimationLibrary.ts";
import { GU_VFX_CATALOG } from "../combat/GuVFXRegistry.ts";

const TILE_SIZE = 256;
const ACTOR_SIZE = 48;

/**
 * Builds the placeholder combat art at runtime so the slice has zero external
 * asset/licensing dependency. Replacing these textures with a packed atlas does
 * not change the animation keys in AssetManifest.
 */
export class PreloadScene extends Phaser.Scene {
  constructor() {
    super({ key: "PreloadScene" });
  }

  preload(): void {
    this.load.image("tile-ground", "/assets/world/ground-moss.webp");
    for (const kind of ["player", "monster", "npc"] as ActorKind[]) {
      const source = kind === "monster" ? "/assets/world/spirit-beast.webp" : "/assets/world/cultivator.webp";
      for (const animation of ANIMATION_STATES) {
        const frames = ACTOR_ASSETS[kind].animations[animation];
        for (let frame = 0; frame < frames; frame += 1) {
          this.load.image(animationTextureKey(kind, animation, frame), source);
        }
      }
    }

    // Preload authored combat & Gu VFX atlases
    const coreAtlases = [
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
    ];
    for (const key of coreAtlases) {
      const def = GU_VFX_CATALOG[key];
      if (def) {
        this.load.atlas(key, def.textureUrl, def.atlasUrl);
      }
    }

    const label = this.add.text(16, 16, "Đang khởi tạo thế giới…", {
      fontFamily: "monospace",
      fontSize: "14px",
      color: "#a7f3d0",
    });
    label.setScrollFactor(0);
  }

  create(): void {
    if (!this.textures.exists("tile-ground")) this.createGroundTexture();
    this.createTokenTexture("token-player", 0x34d399, 0x065f46);
    this.createTokenTexture("token-remote", 0x60a5fa, 0x1e3a8a);
    this.createTokenTexture("token-monster", 0xf87171, 0x7f1d1d);
    this.createTokenTexture("token-npc", 0xfbbf24, 0x78350f);
    this.createRockTexture();

    for (const kind of Object.keys(ACTOR_ASSETS) as ActorKind[]) {
      for (const animation of ANIMATION_STATES) {
        const frames = ACTOR_ASSETS[kind].animations[animation];
        for (let frame = 0; frame < frames; frame += 1) {
          this.createActorFrame(kind, animation, frame, frames);
        }
      }
      registerActorAnimations(this, kind);
    }
    // Keeps the generated VFX keys referenced by the manifest in one place.
    for (const asset of VFX_ASSETS) {
      if (!this.textures.exists(asset.key)) this.createVfxTexture(asset.key);
    }

    this.scene.start("WorldScene");
  }

  private createGroundTexture(): void {
    const graphics = this.add.graphics();
    graphics.fillStyle(0x10271c, 1);
    graphics.fillRect(0, 0, TILE_SIZE, TILE_SIZE);
    // Quiet ground grain gives the procedural world texture without turning
    // every few pixels into a loud editor grid.
    for (let index = 0; index < 120; index += 1) {
      const x = (index * 73 + 19) % TILE_SIZE;
      const y = (index * 137 + 41) % TILE_SIZE;
      const shade = index % 3 === 0 ? 0x285239 : 0x1b3c2a;
      graphics.fillStyle(shade, index % 4 === 0 ? 0.22 : 0.12);
      graphics.fillEllipse(x, y, 10 + (index % 5) * 2, 3 + (index % 3));
    }
    graphics.lineStyle(1, 0x52745b, 0.12);
    graphics.strokeRect(0, 0, TILE_SIZE, TILE_SIZE);
    graphics.lineStyle(1, 0x06160e, 0.18);
    graphics.lineBetween(TILE_SIZE - 1, 0, TILE_SIZE - 1, TILE_SIZE);
    graphics.lineBetween(0, TILE_SIZE - 1, TILE_SIZE, TILE_SIZE - 1);
    graphics.generateTexture("tile-ground", TILE_SIZE, TILE_SIZE);
    graphics.destroy();
  }

  private createTokenTexture(key: string, fill: number, outline: number): void {
    const graphics = this.add.graphics();
    graphics.fillStyle(outline, 1);
    graphics.fillCircle(24, 24, 22);
    graphics.fillStyle(fill, 1);
    graphics.fillCircle(24, 24, 17);
    graphics.lineStyle(2, 0xecfdf5, 0.9);
    graphics.strokeCircle(24, 24, 17);
    graphics.generateTexture(key, ACTOR_SIZE, ACTOR_SIZE);
    graphics.destroy();
  }

  private createRockTexture(): void {
    const graphics = this.add.graphics();
    graphics.fillStyle(0x1f2937, 1);
    graphics.fillRoundedRect(0, 8, 48, 32, 8);
    graphics.fillStyle(0x374151, 1);
    graphics.fillRoundedRect(4, 4, 32, 24, 6);
    graphics.generateTexture("prop-rock", 48, 40);
    graphics.destroy();
  }

  private createVfxTexture(key: string): void {
    const graphics = this.add.graphics();
    graphics.fillStyle(0xffffff, 0.55);
    graphics.fillCircle(32, 32, 26);
    graphics.fillStyle(0xffffff, 0.9);
    graphics.fillCircle(32, 32, 12);
    graphics.generateTexture(key, 64, 64);
    graphics.destroy();
  }

  private createActorFrame(kind: ActorKind, animation: ActorAnimation, frame: number, total: number): void {
    if (this.textures.exists(animationTextureKey(kind, animation, frame))) return;
    const colors = actorColors(kind);
    const graphics = this.add.graphics();
    const bob = animation === "walk" ? (frame % 2 === 0 ? 2 : -2) : animation === "idle" ? (frame % 2 === 0 ? 0 : -1) : 0;
    const centerX = ACTOR_SIZE / 2;
    const centerY = ACTOR_SIZE / 2 + bob;
    const radius = kind === "monster" ? 15 : kind === "npc" ? 13 : 14;

    graphics.fillStyle(0x020617, 0.35);
    graphics.fillEllipse(centerX, ACTOR_SIZE - 7, radius * 2.1, 7);

    if (animation === "death") {
      graphics.fillStyle(colors.outline, 1);
      graphics.fillEllipse(centerX, centerY + 8, radius * 2, radius * 0.8);
      graphics.fillStyle(colors.body, 0.8);
      graphics.fillEllipse(centerX, centerY + 8, radius * 1.5, radius * 0.5);
      graphics.lineStyle(2, 0x0f172a, 0.8);
      graphics.lineBetween(centerX - 6, centerY + 4, centerX + 6, centerY + 12);
      graphics.lineBetween(centerX + 6, centerY + 4, centerX - 6, centerY + 12);
      graphics.generateTexture(animationTextureKey(kind, animation, frame), ACTOR_SIZE, ACTOR_SIZE);
      graphics.destroy();
      return;
    }

    if (animation === "cast") {
      graphics.lineStyle(3, 0xfbbf24, 0.85);
      graphics.strokeCircle(centerX, centerY, radius + 6 + frame * 2);
    }
    if (animation === "walk") {
      graphics.fillStyle(colors.body, 0.9);
      const step = frame % 2 === 0 ? -4 : 4;
      graphics.fillEllipse(centerX + step, ACTOR_SIZE - 10, 8, 5);
    }

    const bodyColor = animation === "hit" ? 0xffffff : colors.body;
    if (kind === "player") {
      graphics.fillStyle(colors.outline, 1);
      graphics.fillEllipse(centerX, centerY + 8, 32, 25);
      graphics.fillStyle(bodyColor, 1);
      graphics.fillEllipse(centerX, centerY + 7, 26, 21);
      graphics.fillStyle(0xf3c99d, 1);
      graphics.fillCircle(centerX, centerY - 9, 9);
      graphics.fillStyle(0x172b25, 1);
      graphics.fillRoundedRect(centerX - 10, centerY - 19, 20, 7, 3);
      graphics.fillStyle(0xfef3c7, 1);
      graphics.fillRect(centerX - 1, centerY - 2, 2, 10);
      graphics.lineStyle(2, 0xecfdf5, 0.95);
      graphics.strokeEllipse(centerX, centerY + 8, 32, 25);
    } else if (kind === "monster") {
      graphics.fillStyle(colors.outline, 1);
      graphics.fillEllipse(centerX, centerY + 2, 37, 31);
      graphics.fillStyle(bodyColor, 1);
      graphics.fillEllipse(centerX, centerY + 2, 29, 24);
      graphics.fillStyle(colors.outline, 1);
      graphics.fillTriangle(centerX - 11, centerY - 6, centerX - 17, centerY - 21, centerX - 3, centerY - 12);
      graphics.fillTriangle(centerX + 11, centerY - 6, centerX + 17, centerY - 21, centerX + 3, centerY - 12);
      graphics.lineStyle(2, 0xfecaca, 0.95);
      graphics.strokeEllipse(centerX, centerY + 2, 37, 31);
      graphics.fillStyle(0x0f172a, 0.9);
      graphics.fillRect(centerX - 7, centerY - 3, 14, 4);
      graphics.fillStyle(0xfef2f2, 0.9);
      graphics.fillRect(centerX - 5, centerY - 2, 3, 2);
      graphics.fillRect(centerX + 2, centerY - 2, 3, 2);
    } else {
      graphics.fillStyle(colors.outline, 1);
      graphics.fillCircle(centerX, centerY, radius + 3);
      graphics.fillStyle(bodyColor, 1);
      graphics.fillCircle(centerX, centerY, radius);
      graphics.lineStyle(2, 0xecfdf5, 0.9);
      graphics.strokeCircle(centerX, centerY, radius);
      graphics.fillStyle(0x0f172a, 0.9);
      graphics.fillCircle(centerX - 4, centerY - 3, 1.8);
      graphics.fillCircle(centerX + 4, centerY - 3, 1.8);
    }
    if (animation === "hit" && total > 1) {
      graphics.lineStyle(2, 0xfca5a5, 1);
      graphics.lineBetween(centerX - 10, centerY - 10, centerX + 10, centerY + 10);
      graphics.lineBetween(centerX + 10, centerY - 10, centerX - 10, centerY + 10);
    }
    if (animation === "cast") {
      graphics.fillStyle(0xfde68a, 0.8);
      graphics.fillCircle(centerX, centerY - radius - 6, 3 + frame);
    }
    graphics.generateTexture(animationTextureKey(kind, animation, frame), ACTOR_SIZE, ACTOR_SIZE);
    graphics.destroy();
  }
}

function actorColors(kind: ActorKind): { body: number; outline: number } {
  switch (kind) {
    case "monster":
      return { body: 0xf87171, outline: 0x7f1d1d };
    case "npc":
      return { body: 0xfbbf24, outline: 0x78350f };
    default:
      return { body: 0x34d399, outline: 0x065f46 };
  }
}
