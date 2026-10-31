# Combat art & animation (PR-21, placeholder set)

Current content version: `placeholder-1` (`frontend/src/game/assets/AssetManifest.ts`).

The slice intentionally ships **procedural placeholder art** generated at runtime;
there are no copied assets, logos, maps or third-party sprites. Every generated
texture is project-owned and safe to replace with a packed atlas later without
changing runtime animation keys.

## Manifest

- `frontend/src/game/assets/AssetManifest.ts`
  - actor kinds: `player`, `monster`, `npc`;
  - animation states: `idle`, `walk`, `cast`, `hit`, `death`;
  - frame counts, frame-rate, fallback texture, license;
  - VFX descriptors (`vfx/impact`, `vfx/cast`, `vfx/status`, `vfx/death`);
  - pure `validateAssetManifest()` used by tests.

- `frontend/src/game/assets/AnimationLibrary.ts`
  - registers idempotent Phaser animations (`player-walk`, `monster-cast`, ...);
  - textures are named `<kind>-<animation>-<frame>`.

- `frontend/src/game/entities/EntityAnimator.ts`
  - chooses idle/walk/cast/hit/death from server state and combat events;
  - flips X from movement direction (placeholder 2-direction rendering);
  - death wins over cast/hit; server HP controls the death state.

- `frontend/src/game/combat/VFXPool.ts`
  - pools floating damage/heal/status text and impact rings;
  - caps active texts/rings (64/48) to avoid unbounded object churn.

- `frontend/src/game/scenes/PreloadScene.ts`
  - generates the placeholder frames and fallback tokens;
  - registers animations before `WorldScene` starts.
- `frontend/src/game/assets/MonsterArt.ts`
  - resolves monster templates to their atlas cells and loads only atlases visible in the current scene;
  - the generated monster frame remains a fallback while an atlas loads or if it fails.

## Runtime mapping

| Server signal | Visual |
| --- | --- |
| position delta > 0.5 px/frame | `walk` |
| no movement | `idle` |
| `CAST_ACCEPTED` (`duration_ticks`) | `cast` until windup ends |
| `HIT_CONFIRMED` / `DAMAGE_APPLIED` | `hit` 180 ms + impact ring |
| `ENTITY_DIED` / HP 0 | `death` (semi-transparent) |
| monster `ai_state` aggro/chase/attack with a target | name and atlas art on the map; red HP bar while engaged |
| `HEAL_APPLIED` / `SHIELD_APPLIED` | floating text, heal ring |
| `STATUS_APPLIED` | status label floating text |
| `HIDDEN_PASSIVE_TRIGGERED` | notice + floating text |

Projectiles are server snapshot data rendered by `ProjectileView`; the client
never resolves a hit. Cast telegraphs come from server skill metadata
(range/radius/mode), not hard-coded per-skill damage.

## Replacing placeholders with real art

1. Add a packed atlas + JSON frames under `frontend/public/game/atlases/`.
2. Keep the same animation keys (`<kind>-<state>`), or update
   `AnimationLibrary`/manifest together.
3. Record source/license/resolution/content hash in the manifest and this doc.
4. Prefer atlas frames over per-frame generated textures; keep fallback tokens
   for missing assets.
5. Do not copy art/UI from another game.

## Known gaps

- Placeholder is 2-direction (flipX), not the 4/8-direction target in ADR-01.
- No packed atlas, texture compression or animation graph yet.
- No asset existence validation against a CDN/atlas build (PR-21 follow-up).
- VFX are primitive rings/text; particle/shader pooling remains future work.

## Hub UI paper skin

- `frontend/public/assets/ui/guworld-rice-paper.webp` is an original generated
  rice-paper material texture for GuWorld UI. It is a repeating 1254×1254 WebP
  (about 84 KB), with no copied game artwork, text, or third-party marks.
- `frontend/src/app/globals.css` loads it as a CSS background only when a paper
  surface is rendered. Browser caching is used; bump the `?v=` query in
  `.mmo-paper-surface` when replacing the file so clients fetch the new texture.
- `mmo-paper-theme`, `mmo-paper-surface`, `mmo-paper-heading`, and
  `mmo-bamboo-action` provide reusable dark lacquer panels, warm headers, and
  bamboo-like controls with ivory text. Small helper labels are raised to 13px,
  and the Aperture panel uses bright semantic accents for dark-surface
  contrast. Text, values, and actions remain live UI elements.
- Applied to the shared MMO feature-view shell, in-world business panels, the
  daily quest notice, and the world atlas frame. New asset files must include
  their source/license notes here and should retain the existing low-size,
  lazy-by-surface loading approach.
