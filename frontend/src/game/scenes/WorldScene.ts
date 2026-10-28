import Phaser from "phaser";
import type { WorldRuntime, RuntimeEntity } from "../core/WorldRuntime.ts";
import { InputRouter } from "../input/InputRouter.ts";
import { ClickToMove } from "../input/ClickToMove.ts";
import { LoadedWorldMap, computeWorldMapHash, loadWorldMap } from "../world/MapLoader.ts";
import { CastTelegraph } from "../combat/CastTelegraph.ts";
import { ProjectileView } from "../combat/ProjectileView.ts";
import { VFXPool } from "../combat/VFXPool.ts";
import { EntityAnimator } from "../entities/EntityAnimator.ts";
import { animationTextureKey, type ActorKind } from "../assets/AssetManifest.ts";
import { getMonsterArt, loadMonsterArtCell } from "../assets/MonsterArt.ts";
import { CameraController } from "../world/CameraController.ts";
import { RealtimeDebugHUD } from "../debug/RealtimeDebugHUD.ts";
import { findPath } from "../world/PathPlanner.ts";
import type { WorldEvent, WorldSkillInfo } from "../network/Protocol.ts";
import { mmoClientConfig } from "@/lib/mmoConfig";
import {
  PIXELS_PER_UNIT,
  pixelsToWorld,
  worldToPixels,
  type WorldPoint,
} from "../world/WorldUnits.ts";
import { PREVIEW_WORLD_HEIGHT, PREVIEW_WORLD_WIDTH, WORLD_RUNTIME_KEY } from "../bootstrap/constants.ts";
import { useGameStore } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";

interface ActorView {
  container: Phaser.GameObjects.Container;
  sprite: Phaser.GameObjects.Sprite;
  animator: EntityAnimator;
  label: Phaser.GameObjects.Text;
  hpBar: Phaser.GameObjects.Graphics;
  lastHp: number;
  lastMaxHp: number;
  lastEngaged: boolean;
  lastSelected: boolean;
  templateId: string;
  lastX: number;
  lastY: number;
  castUntil: number;
  hitUntil: number;
}

const HP_BAR_WIDTH = 42;
const BASIC_ATTACK_RANGE = 2.2;
const AUTO_FARM_ATTACK_INTERVAL_MS = 500;

function formatDamageAmount(amount: number): string {
  if (amount < 0.001) return Number(amount.toPrecision(3)).toString();
  return new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 3 }).format(amount);
}

/**
 * Realtime world scene. When a WorldRuntime is attached it renders authoritative
 * snapshots and forwards WASD/click input; without one it shows a local
 * placeholder so the Phaser lifecycle can still be smoke-tested.
 */
export class WorldScene extends Phaser.Scene {
  private runtime: WorldRuntime | null = null;
  private readonly router = new InputRouter();
  private readonly click = new ClickToMove();
  private readonly actors = new Map<string, ActorView>();
  private readonly requestedMonsterArt = new Set<string>();
  private readonly failedMonsterArtUntil = new Map<string, number>();
  private monsterArtGeneration = 0;
  private marker!: Phaser.GameObjects.Graphics;
  private hud!: Phaser.GameObjects.Text;
  private background!: Phaser.GameObjects.TileSprite;
  private boundary!: Phaser.GameObjects.Graphics;
  private terrain!: Phaser.GameObjects.Graphics;
  private legacyOverlay!: Phaser.GameObjects.Graphics;
  private legacyLabels: Phaser.GameObjects.Text[] = [];
  private dynamicWorldObjects!: Phaser.GameObjects.Graphics;
  private dynamicWorldLabels: Phaser.GameObjects.Text[] = [];
  private lastWorldObjectsSyncAt = 0;
  private map: LoadedWorldMap | null = null;
  private mapHash: string | null = null;
  private mapStatus: "idle" | "loading" | "ready" | "error" = "idle";
  private mapError = "";
  private mapUrl = "";
  private mapNotice = "";
  private mapNoticeUntil = 0;
  private selectedTargetId: string | null = null;
  private targetRing!: Phaser.GameObjects.Graphics;
  private telegraph!: CastTelegraph;
  private projectileView!: ProjectileView;
  private vfx!: VFXPool;
  private cameraController!: CameraController;
  private debugHud: RealtimeDebugHUD | null = null;
  private targetIndicator!: Phaser.GameObjects.Graphics;
  private pathQueue: WorldPoint[] = [];
  private pathGoal: WorldPoint | null = null;
  private aimingSlot: number | null = null;
  private autoFarmTargetId = "";
  private autoFarmMoving = false;
  private autoFarmPauseUntil = 0;
  private nextAutoFarmAttackAt = 0;
  private nextAutoFarmSkillAt = 0;
  private nextAutoFarmMoveAt = 0;
  private autoFarmStopNoticeShown = false;
  private combatUnsubscribe: (() => void) | null = null;
  private welcomeUnsubscribe: (() => void) | null = null;
  private castRequestUnsubscribe: (() => void) | null = null;
  private selfId: string | null = null;
  private lastHudAt = 0;
  private cleanups: Array<() => void> = [];

  private readonly onResize = (): void => this.positionHud();
  private readonly onKeyDown = (event: KeyboardEvent): void => {
    if (event.repeat) return;
    if (event.code === "F3" && this.debugHud) {
      event.preventDefault();
      this.debugHud.toggle();
      this.hud?.setVisible(!this.hud.visible);
      return;
    }
    if (isTextField(event.target)) return;
    if (this.runtime?.isUiModal()) return;
    if (event.code === "Escape") {
      this.click.cancel();
      this.runtime?.cancelMoveTarget();
      this.cancelPath();
      this.cancelAim();
      this.runtime?.setSelectedTarget(null);
      this.selectedTargetId = null;
      return;
    }
    if (event.code === "Tab") {
      event.preventDefault();
      this.cycleTarget();
      return;
    }
    const slot = slotFromCode(event.code);
    if (slot !== null) {
      event.preventDefault();
      this.activateHotbarSlot(slot);
      return;
    }
    if (this.router.setKey(event.code, true)) {
      event.preventDefault();
      this.click.cancel();
    }
  };
  private readonly onKeyUp = (event: KeyboardEvent): void => {
    this.router.setKey(event.code, false);
  };
  private readonly onBlur = (): void => {
    this.router.reset();
    this.runtime?.setMoveVector({ x: 0, y: 0 });
    this.click.cancel();
    this.runtime?.cancelMoveTarget();
    this.cancelPath();
    this.cancelAim();
  };
  private readonly onVisibility = (): void => {
    if (document.visibilityState === "hidden") this.onBlur();
  };
  private readonly onPointerDown = (pointer: Phaser.Input.Pointer): void => {
    if (!this.runtime || (pointer.button !== 0 && pointer.button !== 2)) return;
    if (!this.runtime.getWelcome()) return;
    this.stopAutoFarmMovement();
    this.autoFarmPauseUntil = this.time.now + 3000;
    if (pointer.button === 2) {
      this.activateRightClickAction();
      return;
    }
    if (this.aimingSlot !== null) {
      this.castSlot(this.aimingSlot, true);
      return;
    }
    const leftClickSlot = this.runtime?.getRealtimeSettings().leftClickSlot ?? 0;
    if (leftClickSlot > 0) {
      this.castSlot(leftClickSlot, true);
      return;
    }
    if (!this.router.canClickMove()) return;
    const worldPoint = this.cameras.main.getWorldPoint(pointer.x, pointer.y);
    const target: WorldPoint = pixelsToWorld({ x: worldPoint.x, y: worldPoint.y });
    const clicked = this.findTargetNear(target);
    if (clicked) {
      this.runtime.setSelectedTarget(clicked.id);
      this.selectedTargetId = clicked.id;
      return;
    }
    if (this.map && this.map.isSolid(target.x, target.y)) {
      this.notice("Không thể di chuyển vào vật cản hoặc ngoài bản đồ");
      return;
    }
    if (this.map) {
      const path = findPath(this.map, this.runtime.getSelfPosition(), target);
      if (!path || path.length === 0) {
        this.notice("Không tìm thấy đường đi tới điểm đó");
        return;
      }
      this.startPath(path, target);
      return;
    }
    if (this.click.begin(target)) {
      this.runtime.sendMoveTarget(target);
    }
  };

  constructor() {
    super({ key: "WorldScene" });
  }

  create(): void {
    this.input.mouse?.disableContextMenu();
    this.background = this.add
      .tileSprite(0, 0, PREVIEW_WORLD_WIDTH, PREVIEW_WORLD_HEIGHT, "tile-ground")
      .setOrigin(0, 0)
      .setDepth(-10_000);
    this.terrain = this.add.graphics().setDepth(-9_500);
    this.legacyOverlay = this.add.graphics().setDepth(-9_400);
    this.boundary = this.add.graphics().setDepth(-9_000);
    this.drawBoundary(PREVIEW_WORLD_WIDTH, PREVIEW_WORLD_HEIGHT);
    this.dynamicWorldObjects = this.add.graphics().setDepth(-9_100);

    this.marker = this.add.graphics().setDepth(-8_000);

    this.cameras.main.setBackgroundColor("#07110d");
    this.cameraController = new CameraController(this);
    this.cameraController.setBounds(0, 0, PREVIEW_WORLD_WIDTH, PREVIEW_WORLD_HEIGHT);
    this.targetIndicator = this.add.graphics().setScrollFactor(0).setDepth(95_000);

    this.runtime = (this.registry.get(WORLD_RUNTIME_KEY) as WorldRuntime | undefined) ?? null;
    this.targetRing = this.add.graphics().setDepth(-7_000);
    this.telegraph = new CastTelegraph(this);
    this.projectileView = new ProjectileView(this);
    this.vfx = new VFXPool(this);
    this.combatUnsubscribe = this.runtime?.onCombatEvents((event) => this.handleCombatEvent(event)) ?? null;
    this.debugHud = this.runtime ? new RealtimeDebugHUD(this, this.runtime) : null;
    this.castRequestUnsubscribe = this.runtime?.onCastRequest((slot) => this.activateHotbarSlot(slot)) ?? null;
    if (this.runtime) {
      this.welcomeUnsubscribe = this.runtime.subscribe(() => {
        const zoneId = this.runtime?.getWelcome()?.zone_id;
        if (!zoneId) return;
        const mapId = zoneId.includes("#") ? zoneId.split("#")[0] : zoneId;
        const url = `/game/maps/${encodeURIComponent(mapId)}.json`;
        if (url !== this.mapUrl) {
          this.mapUrl = url;
          void this.loadMap(url);
        }
      });
    } else {
      this.mapUrl = mmoClientConfig.mapUrl || `/game/maps/${mmoClientConfig.zoneId || "realtime_preview"}.json`;
      void this.loadMap(this.mapUrl);
    }

    this.hud = this.add
      .text(0, 0, "", {
        fontFamily: "monospace",
        fontSize: "13px",
        color: "#a7f3d0",
        backgroundColor: "#03130ddd",
        padding: { x: 10, y: 8 },
      })
      .setScrollFactor(0)
      .setDepth(100_000)
      .setVisible(false);
    this.positionHud();

    this.scale.on(Phaser.Scale.Events.RESIZE, this.onResize);
    window.addEventListener("keydown", this.onKeyDown, { passive: false });
    window.addEventListener("keyup", this.onKeyUp);
    window.addEventListener("blur", this.onBlur);
    document.addEventListener("visibilitychange", this.onVisibility);
    this.input.mouse?.disableContextMenu();
    this.input.on(Phaser.Input.Events.POINTER_DOWN, this.onPointerDown);

    this.cleanups.push(() => {
      this.scale.off(Phaser.Scale.Events.RESIZE, this.onResize);
      window.removeEventListener("keydown", this.onKeyDown);
      window.removeEventListener("keyup", this.onKeyUp);
      window.removeEventListener("blur", this.onBlur);
      document.removeEventListener("visibilitychange", this.onVisibility);
      this.input.off(Phaser.Input.Events.POINTER_DOWN, this.onPointerDown);
      this.combatUnsubscribe?.();
      this.combatUnsubscribe = null;
      this.welcomeUnsubscribe?.();
      this.welcomeUnsubscribe = null;
      this.castRequestUnsubscribe?.();
      this.castRequestUnsubscribe = null;
      this.projectileView.destroy();
      this.vfx.destroy();
      this.telegraph.clear();
      this.cameraController.destroy();
      this.targetIndicator.destroy();
      this.debugHud?.destroy();
      this.debugHud = null;
    });
    this.events.once(Phaser.Scenes.Events.SHUTDOWN, () => {
      this.monsterArtGeneration++;
      this.requestedMonsterArt.clear();
      this.failedMonsterArtUntil.clear();
      for (const cleanup of this.cleanups.splice(0)) cleanup();
      this.actors.clear();
    });

    if (!this.runtime) this.spawnPreviewActors();
  }

  update(_time: number, delta: number): void {
    this.syncFocus();
    if (!this.runtime) {
      this.updateHud(false);
      return;
    }

    const movement = this.router.movement();
    if (movement.x !== 0 || movement.y !== 0) {
      this.stopAutoFarmMovement();
      this.click.cancel();
      this.cancelPath();
    }
    this.runtime.setMoveVector(movement);
    this.runtime.update(delta / 1000);
    this.advancePath();
    this.selectedTargetId = this.runtime.getSelectedTargetId();

    const entities = this.runtime.getEntities();
    this.updateAutoFarm(_time, entities);
    this.syncActors(entities);
    this.syncCamera();
    this.drawPathMarker(entities);
    this.drawTargetRing();
    this.drawTargetIndicator(entities);
    this.drawAimTelegraph(entities);
    this.projectileView.sync(this.runtime.getProjectiles());
    this.syncDynamicWorldObjects();
    this.debugHud?.update();
    this.updateHud(true);
  }

  private updateAutoFarm(now: number, entities: RuntimeEntity[]): void {
    if (!this.runtime || this.runtime.getStatus() !== "joined") return;
    const controls = this.runtime.getRealtimeSettings();
    if (!controls.autoFarm || this.runtime.isUiModal()) {
      this.autoFarmTargetId = "";
      this.stopAutoFarmMovement();
      return;
    }
    const manualMovement = this.router.movement();
    if (manualMovement.x !== 0 || manualMovement.y !== 0 || now < this.autoFarmPauseUntil) {
      this.autoFarmTargetId = "";
      this.stopAutoFarmMovement();
      return;
    }
    const self = entities.find((entity) => entity.isSelf);
    if (!self || self.maxHp <= 0) return;
    const hpPercent = (self.hp / self.maxHp) * 100;
    if (controls.farmStopHPPercent > 0 && hpPercent <= controls.farmStopHPPercent) {
      this.autoFarmTargetId = "";
      this.stopAutoFarmMovement();
      if (!this.autoFarmStopNoticeShown) {
        this.notice(`Auto farm dừng vì HP ≤ ${controls.farmStopHPPercent}%`);
        this.autoFarmStopNoticeShown = true;
      }
      return;
    }
    this.autoFarmStopNoticeShown = false;

    const target = entities
      .filter((entity) => this.isAttackableEntity(entity))
      .map((entity) => ({ entity, distance: Math.hypot(entity.x - self.x, entity.y - self.y) }))
      .filter(({ distance }) => distance <= controls.farmRange)
      .sort((left, right) => left.distance - right.distance)[0];
    if (!target) {
      this.autoFarmTargetId = "";
      this.stopAutoFarmMovement();
      return;
    }

    const monster = target.entity;
    this.autoFarmTargetId = monster.id;
    this.selectedTargetId = monster.id;
    this.runtime.setSelectedTarget(monster.id);
    if (target.distance > BASIC_ATTACK_RANGE - 0.2) {
      if (now < this.nextAutoFarmMoveAt && this.pathQueue.length > 0) return;
      this.nextAutoFarmMoveAt = now + 900;
      const distance = Math.max(0.01, target.distance);
      const stopDistance = BASIC_ATTACK_RANGE - 0.4;
      const goal = {
        x: monster.x + ((self.x - monster.x) / distance) * stopDistance,
        y: monster.y + ((self.y - monster.y) / distance) * stopDistance,
      };
      if (this.map) {
        const path = findPath(this.map, this.runtime.getSelfPosition(), goal);
        if (path?.length) {
          this.startPath(path, goal);
          this.autoFarmMoving = true;
        }
      } else {
        this.autoFarmMoving = this.runtime.sendMoveTarget(goal);
      }
      return;
    }

    this.cancelPath();
    this.runtime.cancelMoveTarget();
    this.autoFarmMoving = false;
    if (now >= this.nextAutoFarmAttackAt) {
      this.runtime.sendBasicAttack(monster.id);
      this.nextAutoFarmAttackAt = now + AUTO_FARM_ATTACK_INTERVAL_MS;
    }
    if (controls.farmUseSkills && now >= this.nextAutoFarmSkillAt) {
      const skill = this.autoFarmSkill(controls.guPriorities, self, monster);
      this.nextAutoFarmSkillAt = now + 250;
      if (skill) this.castAutoFarmSkill(skill.slot, monster, self);
    }
  }

  private stopAutoFarmMovement(): void {
    if (!this.autoFarmMoving) return;
    this.cancelPath();
    this.runtime?.cancelMoveTarget();
    this.autoFarmMoving = false;
  }

  private autoFarmSkill(
    priorities: ReturnType<WorldRuntime["getRealtimeSettings"]>["guPriorities"],
    self: RuntimeEntity,
    target: RuntimeEntity,
  ): WorldSkillInfo | null {
    if (!this.runtime) return null;
    const skills = this.runtime.getSkills();
    const tactics = priorities.length > 0
      ? [...priorities].sort((left, right) => left.priority - right.priority)
      : skills.map((skill, index) => ({ gu_id: skill.id, priority: index + 1, use_condition: "always", condition_value: 0 }));
    const combat = this.runtime.getSelfCombat();
    const tick = this.runtime.estimatedTick();
    const hpPercent = self.maxHp > 0 ? (self.hp / self.maxHp) * 100 : 0;
    const manaPercent = combat.maxMana > 0 ? (combat.mana / combat.maxMana) * 100 : 0;
    const targetHpPercent = target.maxHp > 0 ? (target.hp / target.maxHp) * 100 : 0;
    for (const tactic of tactics) {
      const skill = skills.find((entry) => entry.id === tactic.gu_id);
      if (!skill || combat.mana < skill.mana_cost || (combat.cooldowns[skill.slot] ?? 0) > tick) continue;
      if (combat.castSkillId && combat.castEndTick > tick) continue;
      const value = tactic.condition_value ?? 30;
      const conditionMet = tactic.use_condition === "hp_below"
        ? hpPercent < value
        : tactic.use_condition === "mana_above"
          ? manaPercent > value
          : tactic.use_condition === "enemy_hp_below"
            ? targetHpPercent < value
            : true;
      if (conditionMet) return skill;
    }
    return null;
  }

  private castAutoFarmSkill(slot: number, target: RuntimeEntity, self: RuntimeEntity): void {
    if (!this.runtime) return;
    const skill = this.skillForSlot(slot);
    if (!skill) return;
    switch (skill.mode) {
      case "SELF":
        this.runtime.sendCast({ slot, mode: "SELF" });
        break;
      case "TARGET":
        this.runtime.sendCast({ slot, mode: "TARGET", target_id: target.id });
        break;
      case "DIRECTION": {
        const dx = target.x - self.x;
        const dy = target.y - self.y;
        const length = Math.hypot(dx, dy) || 1;
        this.runtime.sendCast({ slot, mode: "DIRECTION", aim_x: dx / length, aim_y: dy / length });
        break;
      }
      case "GROUND":
        this.runtime.sendCast({ slot, mode: "GROUND", x: target.x, y: target.y });
        break;
    }
  }

  private syncFocus(): void {
    this.router.setUIFocused(isTextField(document.activeElement));
    this.router.setModal(this.runtime?.isUiModal() ?? false);
  }

  private pointerWorld(): WorldPoint {
    const pointer = this.input.activePointer;
    const worldPoint = this.cameras.main.getWorldPoint(pointer.x, pointer.y);
    return pixelsToWorld({ x: worldPoint.x, y: worldPoint.y });
  }

  private monsters(): RuntimeEntity[] {
    if (!this.runtime) return [];
    return this.runtime.getEntities().filter((entity) => this.isAttackableEntity(entity));
  }

  // Aperture resource nodes are stationary player-faction entities used by the
  // tribulation sim. They are visible but must never become player targets.
  private isAttackableEntity(entity: RuntimeEntity): boolean {
    return entity.kind === "monster" && entity.templateId !== "aperture_node" && entity.hp > 0;
  }

  private findTargetNear(point: WorldPoint): RuntimeEntity | null {
    let nearest: RuntimeEntity | null = null;
    let nearestDistance = 1.35;
    const candidates = this.runtime ? this.runtime.getEntities().filter((e) => !e.isSelf && this.isAttackableEntity(e)) : [];
    for (const candidate of candidates) {
      const distance = Math.hypot(candidate.x - point.x, candidate.y - point.y);
      if (distance <= nearestDistance) {
        nearest = candidate;
        nearestDistance = distance;
      }
    }
    return nearest;
  }

  private cycleTarget(): void {
    if (!this.runtime) return;
    const self = this.runtime.getSelfPosition();
    const monsters = this.monsters().sort((left, right) => {
      const leftDistance = Math.hypot(left.x - self.x, left.y - self.y);
      const rightDistance = Math.hypot(right.x - self.x, right.y - self.y);
      if (leftDistance !== rightDistance) return leftDistance - rightDistance;
      return left.id.localeCompare(right.id);
    });
    if (monsters.length === 0) {
      this.notice("Không có mục tiêu trong vùng");
      return;
    }
    const current = monsters.findIndex((monster) => monster.id === this.selectedTargetId);
    const next = monsters[(current + 1) % monsters.length].id;
    this.selectedTargetId = next;
    this.runtime.setSelectedTarget(next);
  }

  private castSlot(slot: number, fromPointer = false): void {
    if (!this.runtime || this.runtime.getStatus() !== "joined") {
      this.notice("Chưa kết nối world server");
      return;
    }
    const skill = this.skillForSlot(slot);
    if (!skill) {
      this.notice(`Nút ${slot} chưa có Cổ hỗ trợ realtime`);
      return;
    }
    const self = this.runtime.getSelfPosition();
    const pointer = this.pointerWorld();
    if (!fromPointer && (skill.mode === "DIRECTION" || skill.mode === "GROUND")) {
      if (this.aimingSlot !== slot) {
        this.aimingSlot = slot;
        this.router.setAiming(true);
        this.runtime?.setAimingSlot(slot);
        this.notice(`${skill.name}: chọn hướng/điểm rồi click để cast · Esc hủy`);
        return;
      }
    }
    if (!fromPointer && skill.mode === "TARGET") {
      const candidate = this.findTargetNear(pointer) ?? (this.selectedTargetId ? this.monsters().find((m) => m.id === this.selectedTargetId) : null);
      if (!candidate && this.aimingSlot !== slot) {
        this.aimingSlot = slot;
        this.router.setAiming(true);
        this.runtime?.setAimingSlot(slot);
        this.notice(`${skill.name}: nhấp vào mục tiêu để cast · Esc hủy`);
        return;
      }
    }
    this.cancelAim();
    switch (skill.mode) {
      case "SELF":
        this.runtime.sendCast({ slot, mode: "SELF" });
        return;
      case "TARGET": {
        const clickedTarget = fromPointer ? this.findTargetNear(pointer) : null;
        let target = clickedTarget ?? this.monsters().find((monster) => monster.id === this.selectedTargetId) ?? null;
        if (!target) target = this.findTargetNear(pointer);
        if (!target) {
          const nearest = this.monsters().sort(
            (left, right) =>
              Math.hypot(left.x - self.x, left.y - self.y) - Math.hypot(right.x - self.x, right.y - self.y),
          )[0];
          target = nearest ?? null;
        }
        if (!target) {
          this.notice("Không có mục tiêu — click quái hoặc Tab");
          return;
        }
        this.selectedTargetId = target.id;
        this.runtime.setSelectedTarget(target.id);
        this.runtime.sendCast({ slot, mode: "TARGET", target_id: target.id });
        return;
      }
      case "DIRECTION": {
        const dx = pointer.x - self.x;
        const dy = pointer.y - self.y;
        const length = Math.hypot(dx, dy);
        if (length < 0.1) {
          this.notice("Cần hướng ngắm hợp lệ");
          return;
        }
        this.runtime.sendCast({ slot, mode: "DIRECTION", aim_x: dx / length, aim_y: dy / length });
        return;
      }
      case "GROUND":
        this.runtime.sendCast({ slot, mode: "GROUND", x: pointer.x, y: pointer.y });
        return;
      default:
        return;
    }
  }

  activateHotbarSlot(slot: number): void {
    this.castSlot(slot);
  }

  setJoystickVector(vector: WorldPoint): void {
    this.router.setJoystick(vector);
  }

  zoomIn(): number {
    return this.cameraController ? this.cameraController.zoomIn() : 1;
  }

  zoomOut(): number {
    return this.cameraController ? this.cameraController.zoomOut() : 1;
  }

  getZoom(): number {
    return this.cameraController ? this.cameraController.getZoom() : 1;
  }

  activateRightClickAction(): void {
    if (!this.runtime) return;
    const controls = this.runtime.getRealtimeSettings();
    if (controls.rightClickAction === "skill" && controls.rightClickSlot > 0) {
      this.castSlot(controls.rightClickSlot, true);
      return;
    }
    this.basicAttack();
  }

  activateHudAction(): void {
    if (!this.runtime) return;
    const controls = this.runtime.getRealtimeSettings();
    if (controls.rightClickAction === "skill" && controls.rightClickSlot > 0) {
      this.castSlot(controls.rightClickSlot);
      return;
    }
    this.basicAttack(true);
  }

  private basicAttack(preferSelected = false): void {
    if (!this.runtime || this.runtime.getStatus() !== "joined") {
      this.notice("Chưa kết nối world server");
      return;
    }
    let target = preferSelected ? this.monsters().find((monster) => monster.id === this.selectedTargetId) ?? null : null;
    if (!target) {
      const near = this.findTargetNear(this.pointerWorld());
      if (near?.kind === "monster") target = near;
    }
    if (!target) target = this.monsters().find((monster) => monster.id === this.selectedTargetId) ?? null;
    if (!target) {
      const self = this.runtime.getSelfPosition();
      target = this.monsters().sort(
        (left, right) =>
          Math.hypot(left.x - self.x, left.y - self.y) - Math.hypot(right.x - self.x, right.y - self.y),
      )[0] ?? null;
    }
    if (!target) {
      this.notice("Không có quái để đánh — click quái hoặc Tab");
      return;
    }
    this.selectedTargetId = target.id;
    this.runtime.setSelectedTarget(target.id);
    this.runtime.sendBasicAttack(target.id);
  }

  private skillForSlot(slot: number): WorldSkillInfo | null {
    if (!this.runtime) return null;
    return this.runtime.getSkills().find((skill) => skill.slot === slot) ?? null;
  }

  private hoveredSkillSlot: number | null = null;

  public setHoveredSkillSlot(slot: number | null): void {
    this.hoveredSkillSlot = slot;
    if (slot === null && this.aimingSlot === null) {
      this.telegraph.clear();
    }
  }

  public cancelAim(): void {
    if (this.aimingSlot === null) return;
    this.aimingSlot = null;
    this.router.setAiming(false);
    this.runtime?.setAimingSlot(null);
    this.telegraph.clear();
  }

  private drawAimTelegraph(entities: RuntimeEntity[]): void {
    if (!this.runtime) {
      this.telegraph.clear();
      return;
    }
    const self = entities.find((entity) => entity.isSelf);
    if (!self) {
      this.telegraph.clear();
      return;
    }

    if (this.aimingSlot !== null) {
      const skill = this.skillForSlot(this.aimingSlot);
      if (!skill) {
        this.telegraph.clear();
        return;
      }
      this.telegraph.draw(skill, { x: self.x, y: self.y }, this.pointerWorld());
      return;
    }

    if (this.hoveredSkillSlot !== null) {
      const skill = this.skillForSlot(this.hoveredSkillSlot);
      if (!skill) {
        this.telegraph.clear();
        return;
      }
      this.telegraph.drawPreview(skill, { x: self.x, y: self.y });
      return;
    }

    this.telegraph.clear();
  }

  private handleCombatEvent(event: WorldEvent): void {
    const payload = event.payload ?? {};
    const amount = typeof payload.amount === "number" ? payload.amount : 0;
    const reason = typeof payload.reason === "string" ? payload.reason : "";
    const eventX = typeof payload.x === "number" ? payload.x : null;
    const eventY = typeof payload.y === "number" ? payload.y : null;
    switch (event.type) {
      case "BASIC_ATTACK":
        this.markActorCast(event.source_id, 2);
        return;
      case "CAST_ACCEPTED": {
        const duration = typeof payload.duration_ticks === "number" ? payload.duration_ticks : 0;
        this.markActorCast(event.source_id, duration);
        return;
      }
      case "HIT_CONFIRMED": {
        this.markActorHit(event.target_id);
        if (eventX !== null && eventY !== null) {
          const pixels = worldToPixels({ x: eventX, y: eventY });
          this.vfx.spawnRing(pixels.x, pixels.y - 8, 0xfbbf24);
        }
        return;
      }
      case "CAST_REJECTED":
        this.notice(payload.skill_id === "BASIC_ATTACK" ? `Đánh thường bị từ chối: ${reason || "thử lại"}` : `Không cast được: ${reason || "bị từ chối"}`);
        return;
      case "CAST_INTERRUPTED":
        this.notice("Cast bị ngắt");
        return;
      case "MOVE_TARGET_REJECTED":
        this.cancelPath();
        this.runtime?.cancelMoveTarget();
        this.notice(reason ? `Đường đi bị chặn: ${reason}` : "Đường đi bị chặn");
        return;
      case "DAMAGE_APPLIED":
        this.markActorHit(event.target_id);
        if (event.target_id && event.target_id === this.selfId) this.cameraController.shake(0.003);
        if (reason === "immune") {
          this.floatText(eventX, eventY, "MIỄN NHIỄM!", "#fde047");
          if (eventX !== null && eventY !== null) {
            const pixels = worldToPixels({ x: eventX, y: eventY });
            this.vfx.spawnRing(pixels.x, pixels.y - 8, 0xfde047);
          }
        } else if (reason === "evaded") {
          this.floatText(eventX, eventY, "NÉ TRÁNH!", "#67e8f9");
          if (eventX !== null && eventY !== null) {
            const pixels = worldToPixels({ x: eventX, y: eventY });
            this.vfx.spawnRing(pixels.x, pixels.y - 8, 0x67e8f9);
          }
        } else if (reason === "last_stand") {
          this.floatText(eventX, eventY, `-${formatDamageAmount(amount)} BẤT TỬ!`, "#f43f5e");
          if (eventX !== null && eventY !== null) {
            const pixels = worldToPixels({ x: eventX, y: eventY });
            this.vfx.spawnRing(pixels.x, pixels.y - 8, 0xe11d48);
          }
        } else {
          this.floatText(eventX, eventY, amount > 0 ? `-${formatDamageAmount(amount)}` : "chặn", "#fca5a5");
          if (eventX !== null && eventY !== null) {
            const pixels = worldToPixels({ x: eventX, y: eventY });
            this.vfx.spawnRing(pixels.x, pixels.y - 8, amount > 0 ? 0xf87171 : 0x93c5fd);
          }
        }
        return;
      case "HEAL_APPLIED":
        this.floatText(eventX, eventY, `+${Math.round(amount)}`, "#6ee7b7");
        if (eventX !== null && eventY !== null) {
          const pixels = worldToPixels({ x: eventX, y: eventY });
          this.vfx.spawnRing(pixels.x, pixels.y - 8, 0x34d399);
        }
        return;
      case "SHIELD_APPLIED":
        this.floatText(eventX, eventY, `+${Math.round(amount)} khiên`, "#93c5fd");
        return;
      case "STATUS_APPLIED": {
        const status = typeof payload.reason === "string" ? payload.reason : "STATUS";
        this.floatText(eventX, eventY, status, "#c4b5fd");
        return;
      }
      case "HIDDEN_PASSIVE_TRIGGERED":
        this.notice("Nội tại ẩn kích hoạt!");
        this.floatText(eventX, eventY, `+${Math.round(amount)} nội tại`, "#fbbf24");
        return;
      case "ENTITY_DIED":
        if (event.target_id === this.selectedTargetId) {
          this.selectedTargetId = null;
          this.runtime?.setSelectedTarget(null);
        }
        this.notice("Mục tiêu đã gục ngã");
        if (eventX !== null && eventY !== null) {
          const pixels = worldToPixels({ x: eventX, y: eventY });
          this.vfx.spawnRing(pixels.x, pixels.y - 8, 0xf87171);
        }
        return;
      default:
        return;
    }
  }

  private markActorCast(entityId: string | undefined, durationTicks: number): void {
    if (!entityId) return;
    const actor = this.actors.get(entityId);
    if (!actor) return;
    actor.castUntil = this.time.now + Math.max(100, durationTicks * 50);
    if (actor.hitUntil > this.time.now) actor.castUntil = Math.max(actor.castUntil, actor.hitUntil);
  }

  private markActorHit(entityId: string | undefined): void {
    if (!entityId) return;
    const actor = this.actors.get(entityId);
    if (!actor) return;
    actor.hitUntil = this.time.now + 180;
  }

  private floatText(worldX: number | null, worldY: number | null, text: string, color: string): void {
    if (worldX === null || worldY === null) return;
    const pixels = worldToPixels({ x: worldX, y: worldY });
    this.vfx.spawnText(pixels.x, pixels.y - 40, text, color);
  }

  private drawTargetRing(): void {
    this.targetRing.clear();
    if (!this.selectedTargetId) return;
    const target = this.runtime?.getEntities().find((entity) => entity.id === this.selectedTargetId && entity.hp > 0);
    if (!target) {
      this.selectedTargetId = null;
      this.runtime?.setSelectedTarget(null);
      return;
    }
    const pixels = worldToPixels({ x: target.x, y: target.y });
    this.targetRing.lineStyle(2, 0xfbbf24, 0.95);
    this.targetRing.strokeCircle(pixels.x, pixels.y - 8, 24);
    this.targetRing.lineStyle(1, 0xfbbf24, 0.4);
    this.targetRing.strokeCircle(pixels.x, pixels.y - 8, 30);
  }

  private async loadMap(url: string): Promise<void> {
    this.mapStatus = "loading";
    try {
      const loaded = await loadWorldMap(url);
      if (!this.scene.isActive()) return;
      this.map = loaded;
      this.mapStatus = "ready";
      this.mapError = "";
      this.applyMap(loaded);
      this.runtime?.setMapPortals(loaded.portals);
      void computeWorldMapHash(loaded).then((hash) => {
        if (this.map === loaded) this.mapHash = hash;
      });
    } catch (error) {
      this.mapStatus = "error";
      this.mapError = error instanceof Error ? error.message : "Không tải được map";
    }
  }

  private applyMap(map: LoadedWorldMap): void {
    this.background.setSize(map.pixelWidth, map.pixelHeight);
    this.boundary.clear();
    this.drawBoundary(map.pixelWidth, map.pixelHeight);
    this.terrain.clear();
    this.legacyOverlay.clear();
    for (const label of this.legacyLabels) label.destroy();
    this.legacyLabels = [];
    if (map.width * map.height > 65536) {
      // Optimized for large worlds (e.g. 2000x2000): draw outer boundary walls directly instead of 4 million tile iterations
      this.terrain.fillStyle(0x1f2937, 0.92);
      this.terrain.fillRect(0, 0, map.pixelWidth, PIXELS_PER_UNIT);
      this.terrain.fillRect(0, map.pixelHeight - PIXELS_PER_UNIT, map.pixelWidth, PIXELS_PER_UNIT);
      this.terrain.fillRect(0, 0, PIXELS_PER_UNIT, map.pixelHeight);
      this.terrain.fillRect(map.pixelWidth - PIXELS_PER_UNIT, 0, PIXELS_PER_UNIT, map.pixelHeight);
    } else {
      for (let y = 0; y < map.height; y += 1) {
        for (let x = 0; x < map.width; x += 1) {
          const index = y * map.width + x;
          const left = x * PIXELS_PER_UNIT;
          const top = y * PIXELS_PER_UNIT;
          if (map.solid[index]) {
            this.terrain.fillStyle(0x1f2937, 0.92);
            this.terrain.fillRect(left, top, PIXELS_PER_UNIT, PIXELS_PER_UNIT);
          } else if (map.ground[index] === 2) {
            this.terrain.fillStyle(0x123528, 0.55);
            this.terrain.fillRect(left, top, PIXELS_PER_UNIT, PIXELS_PER_UNIT);
          }
        }
      }
    }
    for (const portal of map.portals) {
      const pixel = worldToPixels({ x: portal.x, y: portal.y });
      this.terrain.fillStyle(0x0284c7, 0.22);
      this.terrain.fillCircle(pixel.x, pixel.y - 6, 26);
      this.terrain.lineStyle(2, 0x38bdf8, 0.9);
      this.terrain.strokeCircle(pixel.x, pixel.y - 6, 26);
      this.terrain.lineStyle(1, 0xbae6fd, 0.6);
      this.terrain.strokeCircle(pixel.x, pixel.y - 6, 30);
      this.terrain.fillStyle(0x38bdf8, 0.95);
      this.terrain.fillCircle(pixel.x, pixel.y - 6, 5);

      const destName = portal.name?.trim() || portal.targetZoneId.replaceAll("_", " ");
      this.legacyLabels.push(this.add.text(pixel.x, pixel.y - 36, `🌀 Rìa → ${destName}`, {
        fontFamily: "sans-serif", fontSize: "11px", fontStyle: "bold",
        color: "#e0f2fe", backgroundColor: "#0369a1dd", padding: { x: 5, y: 3 },
      }).setOrigin(0.5, 0.5).setDepth(-8_950));
    }
    for (const region of map.legacyRegions) {
      const x = region.x * PIXELS_PER_UNIT;
      const y = region.y * PIXELS_PER_UNIT;
      const width = region.width * PIXELS_PER_UNIT;
      const height = region.height * PIXELS_PER_UNIT;
      const parent = region.kind === "NGU_VUC";
      const color = regionColor(region.name, region.kind);
      this.legacyOverlay.fillStyle(color, parent ? 0.20 : 0.28);
      this.legacyOverlay.fillRect(x, y, width, height);
      this.legacyOverlay.lineStyle(parent ? 4 : 2, parent ? 0xfbbf24 : color, parent ? 0.78 : 0.62);
      this.legacyOverlay.strokeRect(x, y, width, height);
      if (width >= 80 && height >= 50) {
        this.legacyLabels.push(this.add.text(x + 8, y + 7, region.name, {
          fontFamily: "sans-serif", fontSize: parent ? "16px" : "12px", fontStyle: "bold",
          color: parent ? "#fde68a" : "#f0fdf4", backgroundColor: "#07150ddd", padding: { x: 5, y: 3 },
        }).setDepth(-9_300));
      }
    }
    for (const poi of map.legacyPOIs) {
      const px = poi.x * PIXELS_PER_UNIT;
      const py = poi.y * PIXELS_PER_UNIT;
      const boss = poi.type.toLowerCase().includes("boss");
      this.legacyOverlay.fillStyle(boss ? 0xef4444 : 0xf59e0b, 0.9);
      this.legacyOverlay.fillCircle(px, py, boss ? 7 : 4);
      this.legacyOverlay.lineStyle(1, 0xffffff, 0.75);
      this.legacyOverlay.strokeCircle(px, py, boss ? 7 : 4);
      if (boss || poi.type.toLowerCase().includes("portal") || poi.type.toLowerCase().includes("altar")) {
        const label = poi.type.toLowerCase().includes("boss") ? "BOSS" : poi.type.replaceAll("_", " ");
        this.legacyLabels.push(this.add.text(px + 11, py - 9, label, {
          fontFamily: "monospace", fontSize: "10px", fontStyle: "bold",
          color: boss ? "#fecaca" : "#fde68a", backgroundColor: "#111827dd", padding: { x: 4, y: 2 },
        }).setDepth(-9_200));
      }
    }
    this.cameraController.setBounds(0, 0, map.pixelWidth, map.pixelHeight);
  }

  private notice(message: string): void {
    this.mapNotice = message;
    this.mapNoticeUntil = this.time.now + 2_000;
  }

  private syncDynamicWorldObjects(): void {
    const now = this.time.now;
    if (now - this.lastWorldObjectsSyncAt < 800) return;
    this.lastWorldObjectsSyncAt = now;

    this.dynamicWorldObjects.clear();
    for (const label of this.dynamicWorldLabels) label.destroy();
    this.dynamicWorldLabels = [];

    const zoneId = this.runtime?.getWelcome()?.zone_id?.split("#")[0] ?? "";
    const gs = useGameStore.getState();
    const ws = useWorldStore.getState();

    // 1. Primeval Veins (Nguyên Tuyền / Tiên Tuyền)
    const nearbyVeins = ws.nearbyCells.filter((c) => c.cell_type === "PrimevalVein" || Boolean(c.vein));
    for (const cell of nearbyVeins) {
      const px = cell.x * PIXELS_PER_UNIT;
      const py = cell.y * PIXELS_PER_UNIT;
      const immortal = cell.vein?.vein_type === "immortal";

      this.dynamicWorldObjects.fillStyle(immortal ? 0xf59e0b : 0x10b981, 0.22);
      this.dynamicWorldObjects.fillCircle(px, py - 6, 22);
      this.dynamicWorldObjects.lineStyle(2, immortal ? 0xfde047 : 0x34d399, 0.9);
      this.dynamicWorldObjects.strokeCircle(px, py - 6, 18);
      this.dynamicWorldObjects.fillStyle(immortal ? 0xfef08a : 0xa7f3d0, 0.95);
      this.dynamicWorldObjects.fillCircle(px, py - 6, 5);

      const vName = immortal ? "Tiên Tuyền" : "Nguyên Tuyền";
      const rarity = cell.vein?.rarity || "Thường";
      const stones = cell.vein?.accumulated_stones?.toLocaleString("vi-VN") ?? 0;
      const label = this.add.text(px, py - 32, `💠 ${vName} (${rarity})\n${stones} NT`, {
        fontFamily: "sans-serif", fontSize: "10px", fontStyle: "bold", align: "center",
        color: immortal ? "#fef08a" : "#a7f3d0", backgroundColor: "#064e3bdd", padding: { x: 4, y: 2 },
      }).setOrigin(0.5, 0.5).setDepth(-8_950);
      this.dynamicWorldLabels.push(label);
    }

    // 2. Sect Bases & Territory (Tông Môn)
    const sectsInZone = (gs.sectMap || []).filter((s) => s.region === zoneId && typeof s.map_x === "number" && typeof s.map_y === "number");
    const mySectId = gs.mySectStatus?.sect?.id;
    for (const sect of sectsInZone) {
      const sx = sect.map_x * PIXELS_PER_UNIT;
      const sy = sect.map_y * PIXELS_PER_UNIT;
      const isMySect = sect.id === mySectId;
      const mainHallLvl = (sect.buildings as Record<string, number> | undefined)?.main_hall || sect.level || 1;
      const territoryRadius = (5 + mainHallLvl) * PIXELS_PER_UNIT;
      const isShielded = Boolean(sect.shield_until && new Date(sect.shield_until).getTime() > Date.now());

      // Territory square
      this.dynamicWorldObjects.fillStyle(isMySect ? 0x3b82f6 : 0xd97706, 0.08);
      this.dynamicWorldObjects.fillRect(sx - territoryRadius, sy - territoryRadius, territoryRadius * 2, territoryRadius * 2);
      this.dynamicWorldObjects.lineStyle(2, isMySect ? 0x60a5fa : 0xf59e0b, 0.45);
      this.dynamicWorldObjects.strokeRect(sx - territoryRadius, sy - territoryRadius, territoryRadius * 2, territoryRadius * 2);

      // Pagoda fortress building
      const size = 30;
      this.dynamicWorldObjects.fillStyle(isMySect ? 0x2563eb : 0xd97706, 0.28);
      this.dynamicWorldObjects.fillCircle(sx, sy - 8, size + 8);

      // Base
      this.dynamicWorldObjects.fillStyle(0x18181b, 0.95);
      this.dynamicWorldObjects.fillRect(sx - size / 2, sy - size / 4, size, size * 0.75);
      this.dynamicWorldObjects.lineStyle(2, isMySect ? 0x3b82f6 : 0xf59e0b, 0.9);
      this.dynamicWorldObjects.strokeRect(sx - size / 2, sy - size / 4, size, size * 0.75);

      // Roof 1
      this.dynamicWorldObjects.fillStyle(isMySect ? 0x1d4ed8 : 0xb45309, 0.95);
      this.dynamicWorldObjects.fillTriangle(sx, sy - size * 0.75, sx - size * 0.8, sy - size * 0.25, sx + size * 0.8, sy - size * 0.25);
      // Roof 2
      this.dynamicWorldObjects.fillStyle(isMySect ? 0x2563eb : 0xd97706, 0.95);
      this.dynamicWorldObjects.fillTriangle(sx, sy - size * 1.25, sx - size * 0.55, sy - size * 0.7, sx + size * 0.55, sy - size * 0.7);
      // Spire
      this.dynamicWorldObjects.fillStyle(0xfde047, 1);
      this.dynamicWorldObjects.fillCircle(sx, sy - size * 1.35, 4);

      if (isShielded) {
        this.dynamicWorldObjects.lineStyle(3, 0xf43f5e, 0.85);
        this.dynamicWorldObjects.strokeCircle(sx, sy - 8, size + 14);
      }

      const label = this.add.text(sx, sy - size * 1.6, `🏯 ${sect.name} (Cấp ${sect.level || 1})${isShielded ? " 🛡️" : ""}`, {
        fontFamily: "sans-serif", fontSize: "11px", fontStyle: "bold",
        color: isMySect ? "#bfdbfe" : "#fde68a", backgroundColor: isMySect ? "#1e3a8add" : "#451a03dd", padding: { x: 5, y: 3 },
      }).setOrigin(0.5, 0.5).setDepth(-8_950);
      this.dynamicWorldLabels.push(label);
    }

    // 3. Ground Drops (Chiến Lợi Phẩm)
    for (const drop of gs.groundDrops || []) {
      const dx = drop.x * PIXELS_PER_UNIT;
      const dy = drop.y * PIXELS_PER_UNIT;
      this.dynamicWorldObjects.fillStyle(0xb45309, 0.85);
      this.dynamicWorldObjects.fillCircle(dx, dy - 4, 8);
      this.dynamicWorldObjects.fillStyle(0xfde047, 1);
      this.dynamicWorldObjects.fillCircle(dx, dy - 8, 3);
      this.dynamicWorldObjects.lineStyle(1, 0xfef08a, 0.9);
      this.dynamicWorldObjects.strokeCircle(dx, dy - 4, 8);

      const label = this.add.text(dx, dy - 20, `🎁 ${drop.killer_name || "Rơi đồ"}`, {
        fontFamily: "sans-serif", fontSize: "9px", fontStyle: "bold",
        color: "#fef08a", backgroundColor: "#78350fdd", padding: { x: 3, y: 2 },
      }).setOrigin(0.5, 0.5).setDepth(-8_950);
      this.dynamicWorldLabels.push(label);
    }

    // 4. Player-built Teleport Arrays in this zone
    const myTeleportsInZone = (gs.myTeleportArrays || []).filter((t) => t.region_id === zoneId);
    for (const tp of myTeleportsInZone) {
      const tx = tp.x * PIXELS_PER_UNIT;
      const ty = tp.y * PIXELS_PER_UNIT;
      this.dynamicWorldObjects.fillStyle(0x0284c7, 0.2);
      this.dynamicWorldObjects.fillCircle(tx, ty - 4, 22);
      this.dynamicWorldObjects.lineStyle(2, 0x38bdf8, 0.85);
      this.dynamicWorldObjects.strokeCircle(tx, ty - 4, 18);
      this.dynamicWorldObjects.fillStyle(0xbae6fd, 0.9);
      this.dynamicWorldObjects.fillCircle(tx, ty - 4, 4);

      const label = this.add.text(tx, ty - 28, `🌀 ${tp.name || "Truyền Tống Trận"}`, {
        fontFamily: "sans-serif", fontSize: "10px", fontStyle: "bold",
        color: "#e0f2fe", backgroundColor: "#0369a1dd", padding: { x: 4, y: 2 },
      }).setOrigin(0.5, 0.5).setDepth(-8_950);
      this.dynamicWorldLabels.push(label);
    }
  }

  private syncActors(entities: ReturnType<WorldRuntime["getEntities"]>): void {
    const seen = new Set<string>();
    const selfParty = entities.find((entity) => entity.isSelf)?.partyId ?? "";
    for (const entity of entities) {
      seen.add(entity.id);
      const pixels = worldToPixels({ x: entity.x, y: entity.y });
      let actor = this.actors.get(entity.id);
      if (!actor) {
        const partyMember = !entity.isSelf && selfParty !== "" && entity.partyId === selfParty;
        actor = this.createActor(entity.id, entity.isSelf, entity.kind, partyMember, pixels.x, pixels.y);
        this.actors.set(entity.id, actor);
        if (entity.isSelf) {
          this.selfId = entity.id;
          this.cameraController.follow(actor.container);
        }
      }
      const dx = pixels.x - actor.lastX;
      const moved = Math.hypot(dx, pixels.y - actor.lastY) > 0.5;
      actor.animator.update(this.time.now, moved, entity.hp <= 0, actor.castUntil, actor.hitUntil, dx);
      actor.lastX = pixels.x;
      actor.lastY = pixels.y;
      let renderY = pixels.y;
      if (entity.flying) {
        renderY -= 12 + Math.sin(this.time.now / 300) * 3;
      }
      actor.container.setPosition(pixels.x, renderY);
      actor.container.setDepth(pixels.y);
      if (entity.kind === "monster" && entity.hp <= 0) {
        if (actor.container.visible && actor.container.alpha > 0.4) {
          this.tweens.add({
            targets: actor.container,
            alpha: 0,
            duration: 400,
            onComplete: () => {
              actor.container.setVisible(false);
            },
          });
        } else if (!this.tweens.isTweening(actor.container)) {
          actor.container.setVisible(false);
        }
        if (this.selectedTargetId === entity.id) {
          this.events.emit("select-target", null);
        }
      } else {
        if (!this.tweens.isTweening(actor.container)) {
          actor.container.setVisible(true);
          const targetAlpha = entity.stealthed ? 0.45 : 1;
          actor.container.setAlpha(targetAlpha);
        }
      }
      if (entity.kind === "monster") {
        actor.label.setText(entity.name || entity.templateId.replaceAll("_", " ") || "Quái");
        actor.templateId = entity.templateId;
        this.applyMonsterArt(actor, entity.templateId);
      } else if (entity.kind === "player") {
        actor.label.setText(entity.isSelf ? "Bạn" : entity.name || entity.id.slice(0, 8));
      }
      const engaged = entity.kind === "monster" && !!entity.aggroTargetId && ["aggro", "chase", "attack"].includes(entity.aiState);
      const selected = this.selectedTargetId === entity.id;
      actor.hpBar.setVisible(entity.kind !== "monster" || engaged || selected);
      if (entity.hp !== actor.lastHp || entity.maxHp !== actor.lastMaxHp || engaged !== actor.lastEngaged || selected !== actor.lastSelected) {
        this.drawHpBar(actor.hpBar, entity.maxHp > 0 ? entity.hp / entity.maxHp : 0, entity.kind === "monster" ? 0xd05348 : undefined);
        actor.lastHp = entity.hp;
        actor.lastMaxHp = entity.maxHp;
        actor.lastEngaged = engaged;
        actor.lastSelected = selected;
      }
    }
    for (const [id, actor] of this.actors) {
      if (seen.has(id)) continue;
      actor.container.destroy(true);
      this.actors.delete(id);
      if (id === this.selfId) {
        this.selfId = null;
        this.cameraController.stopFollow();
      }
    }
  }

  private createActor(id: string, isSelf: boolean, kind: ActorKind, partyMember: boolean, x: number, y: number): ActorView {
    const sprite = this.add.sprite(0, 0, animationTextureKey(kind, "idle", 0))
      .setOrigin(0.5, 1)
      .setDisplaySize(kind === "monster" ? 58 : 52, kind === "player" ? 66 : 58);
    if (partyMember) sprite.setTint(0x86efac);
    const animator = new EntityAnimator(sprite, kind);
    const label = this.add
      .text(0, -78, isSelf ? "Bạn" : kind === "monster" ? "Quái" : kind === "npc" ? "NPC" : partyMember ? "Tổ đội" : id.slice(0, 8), {
        fontFamily: "monospace",
        fontSize: "11px",
        color: isSelf || partyMember ? "#6ee7b7" : kind === "monster" ? "#fca5a5" : kind === "npc" ? "#fcd34d" : "#93c5fd",
      })
      .setOrigin(0.5, 1);
    const hpBar = this.add.graphics();
    const container = this.add.container(x, y, [sprite, label, hpBar]);
    const actor: ActorView = {
      container, sprite, animator, label, hpBar, lastHp: -1, lastMaxHp: -1, lastEngaged: false, lastSelected: false, templateId: "",
      lastX: x, lastY: y, castUntil: 0, hitUntil: 0,
    };
    this.drawHpBar(hpBar, 1);
    return actor;
  }

  private applyMonsterArt(actor: ActorView, templateId: string): void {
    if (!templateId) return;
    const key = `monster-art-${templateId}`;
    if (this.textures.exists(key)) {
      actor.animator.setStaticTexture(key);
      return;
    }
    if (this.requestedMonsterArt.has(key) || this.time.now < (this.failedMonsterArtUntil.get(key) ?? 0)) return;
    const cell = getMonsterArt(templateId);
    if (!cell) return;
    this.requestedMonsterArt.add(key);
    const generation = this.monsterArtGeneration;
    void loadMonsterArtCell(cell).then((canvas) => {
      if (!this.sys.isActive() || generation !== this.monsterArtGeneration) return;
      if (!this.textures.exists(key)) this.textures.addCanvas(key, canvas);
      for (const view of this.actors.values()) {
        if (view.templateId === templateId && view.sprite.active) view.animator.setStaticTexture(key);
      }
    }).catch(() => {
      if (generation !== this.monsterArtGeneration) return;
      this.requestedMonsterArt.delete(key);
      this.failedMonsterArtUntil.set(key, this.time.now + 10_000);
    });
  }

  private drawHpBar(graphics: Phaser.GameObjects.Graphics, ratio: number, color?: number): void {
    const value = Math.max(0, Math.min(1, ratio));
    graphics.clear();
    graphics.fillStyle(0x000000, 0.65);
    graphics.fillRect(-HP_BAR_WIDTH / 2 - 1, -66, HP_BAR_WIDTH + 2, 6);
    graphics.fillStyle(color ?? (value > 0.3 ? 0x34d399 : 0xf87171), 1);
    graphics.fillRect(-HP_BAR_WIDTH / 2, -65, HP_BAR_WIDTH * value, 4);
  }

  private syncCamera(): void {
    if (this.map) return;
    if (!this.selfId) return;
    const actor = this.actors.get(this.selfId);
    if (!actor) return;
    const { x, y } = actor.container;
    const width = Math.max(PREVIEW_WORLD_WIDTH, Math.ceil(x + this.scale.width));
    const height = Math.max(PREVIEW_WORLD_HEIGHT, Math.ceil(y + this.scale.height));
    const bounds = this.cameras.main.getBounds();
    if (bounds.width !== width || bounds.height !== height) {
      this.cameraController.setBounds(0, 0, width, height);
    }
  }

  private startPath(path: WorldPoint[], goal: WorldPoint): void {
    this.pathQueue = path.slice(1);
    this.pathGoal = goal;
    this.click.begin(goal);
    this.sendNextPath();
  }

  private sendNextPath(): void {
    if (!this.runtime) return;
    const self = this.runtime.getSelfPosition();
    while (this.pathQueue.length > 0 && Math.hypot(this.pathQueue[0].x - self.x, this.pathQueue[0].y - self.y) <= 0.4) {
      this.pathQueue.shift();
    }
    if (this.pathQueue.length === 0) {
      this.cancelPath();
      return;
    }
    this.runtime.sendMoveTarget(this.pathQueue[0]);
  }

  private advancePath(): void {
    if (!this.runtime || this.pathQueue.length === 0) return;
    if (this.runtime.getMoveTarget() !== null) return;
    this.sendNextPath();
  }

  private cancelPath(): void {
    this.pathQueue = [];
    this.pathGoal = null;
    this.click.cancel();
  }

  private drawTargetIndicator(entities: RuntimeEntity[]): void {
    this.targetIndicator.clear();
    if (!this.selectedTargetId) return;
    const target = entities.find((entity) => entity.id === this.selectedTargetId);
    const self = entities.find((entity) => entity.isSelf);
    if (!target || !self) return;
    const targetPx = worldToPixels({ x: target.x, y: target.y });
    const direction = this.cameraController.offscreenDirection(targetPx);
    if (!direction) return;
    const centerX = this.scale.width / 2;
    const centerY = this.scale.height / 2;
    const radius = Math.min(this.scale.width, this.scale.height) * 0.38;
    const x = centerX + direction.x * radius;
    const y = centerY + direction.y * radius;
    this.targetIndicator.lineStyle(3, 0xfbbf24, 0.9);
    this.targetIndicator.beginPath();
    this.targetIndicator.moveTo(x, y - 10);
    this.targetIndicator.lineTo(x + 7, y + 6);
    this.targetIndicator.lineTo(x - 7, y + 6);
    this.targetIndicator.closePath();
    this.targetIndicator.strokePath();
    this.targetIndicator.fillStyle(0xfbbf24, 0.35);
    this.targetIndicator.fillCircle(centerX, centerY, 2);
  }

  private drawPathMarker(entities: ReturnType<WorldRuntime["getEntities"]>): void {
    this.marker.clear();
    if (!this.runtime) return;
    const self = entities.find((entity) => entity.isSelf);
    if (!self) return;
    if (this.pathQueue.length > 0) {
      const points = [self, ...this.pathQueue];
      if (this.pathGoal) points.push(this.pathGoal);
      this.marker.lineStyle(2, 0x34d399, 0.55);
      for (let index = 1; index < points.length; index += 1) {
        const from = worldToPixels(points[index - 1]);
        const to = worldToPixels(points[index]);
        this.marker.beginPath();
        this.marker.moveTo(from.x, from.y);
        this.marker.lineTo(to.x, to.y);
        this.marker.strokePath();
      }
      for (const waypoint of this.pathQueue) {
        const pixel = worldToPixels(waypoint);
        this.marker.fillStyle(0x6ee7b7, 0.45);
        this.marker.fillCircle(pixel.x, pixel.y, 4);
      }
      if (this.pathGoal) {
        const goal = worldToPixels(this.pathGoal);
        this.marker.lineStyle(2, 0x6ee7b7, 0.9);
        this.marker.strokeCircle(goal.x, goal.y, 10);
      }
      return;
    }
    const target = this.click.getTarget() ?? this.runtime.getMoveTarget();
    if (!target) return;
    this.click.completeIfNear(self);
    const from = worldToPixels({ x: self.x, y: self.y });
    const to = worldToPixels(target);
    this.marker.lineStyle(2, 0x34d399, 0.55);
    this.marker.beginPath();
    this.marker.moveTo(from.x, from.y);
    this.marker.lineTo(to.x, to.y);
    this.marker.strokePath();
    this.marker.lineStyle(2, 0x6ee7b7, 0.9);
    this.marker.strokeCircle(to.x, to.y, 10);
    this.marker.fillStyle(0x34d399, 0.25);
    this.marker.fillCircle(to.x, to.y, 10);
  }

  private updateHud(realtime: boolean): void {
    const now = this.time.now;
    if (now - this.lastHudAt < 100) return;
    this.lastHudAt = now;
    const fps = Math.round(this.game.loop.actualFps);
    if (!realtime || !this.runtime) {
      const mapLine = this.map
        ? `Map ${this.map.id}@${this.map.version}`
        : this.mapStatus === "error"
          ? `Map lỗi: ${this.mapError}`
          : "Đang tải map…";
      this.hud.setText([
        "GUWORLD · MMO PREVIEW",
        `Phaser ${Phaser.VERSION} · ${fps} FPS`,
        mapLine,
        "Chưa kết nối world server — WASD/click hoạt động khi có zone.",
      ].join("\n"));
      return;
    }
    const status = this.runtime.getStatus();
    const welcome = this.runtime.getWelcome();
    const self = this.runtime.getSelfPosition();
    const target = this.runtime.getMoveTarget();
    const mapLine = this.map
      ? `Map ${this.map.id}@${this.map.version}${welcome && welcome.map_version !== this.map.version ? " · LỆCH VERSION" : ""}${welcome?.map_hash && this.mapHash && welcome.map_hash !== this.mapHash ? " · LỆCH HASH" : ""}`
      : this.mapStatus === "error"
        ? `Map lỗi: ${this.mapError}`
        : "Đang tải map…";
    const lines = [
      `GUWORLD · MMO ${status.toUpperCase()}`,
      `Zone ${welcome?.zone_id ?? "—"} · tick ${this.runtime.getSnapshot()?.tick ?? 0} · ${fps} FPS`,
      mapLine,
      `HP ${Math.round(this.runtime.getSelfHp())} · ${self.x.toFixed(1)}, ${self.y.toFixed(1)}`,
      target ? `Đang tới ${target.x.toFixed(1)}, ${target.y.toFixed(1)} · WASD để hủy` : "WASD hoặc click để di chuyển",
    ];
    if (status === "error") lines.push(`Lỗi: ${this.runtime.getError()}`);
    if (this.mapNotice && now < this.mapNoticeUntil) lines.push(this.mapNotice);
    this.hud.setText(lines.join("\n"));
  }

  private spawnPreviewActors(): void {
    this.createActor("preview-self", true, "player", false, 620, 640);
    this.createActor("preview-npc", false, "npc", false, 500, 520);
    this.createActor("preview-mob-1", false, "monster", false, 760, 430);
    this.createActor("preview-mob-2", false, "monster", false, 320, 780);
    this.selfId = "preview-self";
    const self = this.actors.get("preview-self");
    if (self) this.cameraController.follow(self.container);
  }

  private positionHud(): void {
    if (!this.hud) return;
    this.hud.setPosition(16, 52);
  }

  private drawBoundary(width: number, height: number): void {
    this.boundary.lineStyle(6, 0x10b981, 0.55);
    this.boundary.strokeRect(0, 0, width, height);
  }
}

function regionColor(name: string, kind: string): number {
  if (kind === "SAFE_ZONE") return 0x39a878;
  if (kind === "SECRET_REALM") return 0x9a70c8;
  if (kind === "NGU_VUC") return 0x8c793c;
  let hash = 2166136261;
  for (let index = 0; index < name.length; index += 1) {
    hash = Math.imul(hash ^ name.charCodeAt(index), 16777619);
  }
  const palette = [0x3f7653, 0x637d45, 0x777047, 0x3c6f70, 0x755b45];
  return palette[(hash >>> 0) % palette.length];
}

function isTextField(target: EventTarget | Element | null): boolean {
  if (!target || !(target instanceof HTMLElement)) return false;
  const tag = target.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || target.isContentEditable;
}

function slotFromCode(code: string): number | null {
  const match = /^(?:Digit|Numpad)([1-6])$/.exec(code);
  if (!match) return null;
  return Number(match[1]);
}
