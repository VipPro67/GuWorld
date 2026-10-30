import type { CastPayload, WorldSkillInfo } from "../network/Protocol.ts";
import { buildCastIntent } from "../input/CastIntent.ts";
import type { RuntimeEntity } from "../core/WorldRuntime.ts";
import type { WorldPoint } from "../world/WorldUnits.ts";

export interface CastView {
  drawAim(skill: WorldSkillInfo, self: WorldPoint, pointer: WorldPoint): void;
  drawPreview(skill: WorldSkillInfo, self: WorldPoint): void;
  clear(): void;
}

export interface CastDelegate {
  /** True only when the world server session is live. */
  isJoined(): boolean;
  skillForSlot(slot: number): WorldSkillInfo | null;
  /** Predicted self position used to build wire intents. */
  selfPosition(): WorldPoint;
  /** Rendered self position (may be null before the first snapshot); telegraph anchor. */
  renderedSelfPosition(): WorldPoint | null;
  pointerWorld(): WorldPoint;
  selectedTargetId(): string | null;
  selectTarget(entityId: string | null): void;
  /**
   * Reflect aim state into the input precedence block (InputRouter) and the
   * runtime HUD in one call; null cancels both.
   */
  setAiming(slot: number | null): void;
  /** Attackable entities the player may target (tab/target selection domain). */
  attackableEntities(): RuntimeEntity[];
  entityNear(point: WorldPoint): RuntimeEntity | null;
  notice(message: string): void;
  markPoint(point: WorldPoint): void;
  sendCast(intent: Omit<CastPayload, "seq">): void;
  view: CastView;
}

export interface TriggerOptions {
  /** Pointer commits (click on ground/hotbar-bound slots) bypass the aim phase. */
  fromPointer?: boolean;
}

/**
 * Deep module owning the two-phase skill-cast lifecycle (enter aim → commit at
 * pointer or target), valid target filtering and the telegraph view sync.
 * Hotbar/keyboard/HUD/pointer activation all funnel through `triggerSlot`, so
 * the whole cast flow is testable from intent to wire payload without Phaser.
 */
export class CastController {
  private readonly delegate: CastDelegate;
  private aimingState: number | null = null;
  private hoveredSlot: number | null = null;

  constructor(delegate: CastDelegate) {
    this.delegate = delegate;
  }

  public aimingSlot(): number | null {
    return this.aimingState;
  }

  public isAiming(): boolean {
    return this.aimingState !== null;
  }

  /** Hotbar, keyboard, HUD and pointer activation all funnel through here. */
  public triggerSlot(slot: number, options: TriggerOptions = {}): void {
    const fromPointer = options.fromPointer ?? false;
    if (!this.delegate.isJoined()) {
      this.delegate.notice("Chưa kết nối world server");
      return;
    }
    const skill = this.delegate.skillForSlot(slot);
    if (!skill) {
      this.delegate.notice(`Nút ${slot} chưa có Cổ hỗ trợ realtime`);
      return;
    }
    const self = this.delegate.selfPosition();
    const pointer = this.delegate.pointerWorld();
    if (!fromPointer) {
      if ((skill.mode === "DIRECTION" || skill.mode === "GROUND") && this.aimingState !== slot) {
        this.enterAim(slot, `${skill.name}: chọn hướng/điểm rồi click để cast · Esc hủy`);
        return;
      }
      if (skill.mode === "TARGET" && !this.currentTarget() && this.aimingState !== slot) {
        this.enterAim(slot, `${skill.name}: nhấp vào mục tiêu hoặc khoảng trống để cast · Esc hủy`);
        return;
      }
    }
    this.cancelAim();
    switch (skill.mode) {
      case "SELF": {
        const intent = buildCastIntent(skill, self);
        if (intent) this.delegate.sendCast(intent);
        return;
      }
      case "TARGET": {
        const target = fromPointer
          ? this.delegate.entityNear(pointer)
          : this.currentTarget();
        if (target) {
          this.delegate.selectTarget(target.id);
          this.delegate.markPoint({ x: target.x, y: target.y });
          const intent = buildCastIntent(skill, self, target, target.id);
          if (intent) this.delegate.sendCast(intent);
          return;
        }
        const intent = buildCastIntent(skill, self, pointer);
        if (!intent) {
          this.delegate.notice("Cần hướng ngắm hợp lệ");
          return;
        }
        this.delegate.sendCast(intent);
        this.delegate.markPoint(pointer);
        return;
      }
      case "DIRECTION": {
        const intent = buildCastIntent(skill, self, pointer);
        if (!intent) {
          this.delegate.notice("Cần hướng ngắm hợp lệ");
          return;
        }
        this.delegate.sendCast(intent);
        this.delegate.markPoint(pointer);
        return;
      }
      case "GROUND": {
        const intent = buildCastIntent(skill, self, pointer);
        if (intent) this.delegate.sendCast(intent);
        this.delegate.markPoint(pointer);
        return;
      }
      default:
        return;
    }
  }

  /** Escape / blur / modal shutdown of the aim phase. */
  public cancelAim(): void {
    if (this.aimingState === null) return;
    this.aimingState = null;
    this.delegate.setAiming(null);
    this.delegate.view.clear();
  }

  public setHoveredSlot(slot: number | null): void {
    this.hoveredSlot = slot;
    if (slot === null && this.aimingState === null) {
      this.delegate.view.clear();
    }
  }

  /** Per-frame telegraph sync; the scene calls this once from its update loop. */
  public update(): void {
    const slot = this.aimingState;
    if (slot === null) {
      const hovered = this.hoveredSlot;
      if (hovered === null) {
        this.delegate.view.clear();
        return;
      }
      const preview = this.delegate.skillForSlot(hovered);
      if (!preview) {
        this.delegate.view.clear();
        return;
      }
      const previewSelf = this.delegate.renderedSelfPosition();
      if (!previewSelf) {
        this.delegate.view.clear();
        return;
      }
      this.delegate.view.drawPreview(preview, previewSelf);
      return;
    }
    const skill = this.delegate.skillForSlot(slot);
    if (!skill) {
      this.delegate.view.clear();
      return;
    }
    const aimSelf = this.delegate.renderedSelfPosition();
    if (!aimSelf) {
      this.delegate.view.clear();
      return;
    }
    this.delegate.view.drawAim(skill, aimSelf, this.delegate.pointerWorld());
  }

  /** Auto-farm path: cast directly at a resolved target without the aim phase. */
  public castAtEntity(slot: number, target: RuntimeEntity): void {
    const skill = this.delegate.skillForSlot(slot);
    if (!skill) return;
    const intent = buildCastIntent(skill, this.delegate.selfPosition(), target, target.id);
    if (intent) this.delegate.sendCast(intent);
  }

  private enterAim(slot: number, message: string): void {
    this.aimingState = slot;
    this.delegate.setAiming(slot);
    this.delegate.notice(message);
  }

  private currentTarget(): RuntimeEntity | null {
    const id = this.delegate.selectedTargetId();
    if (!id) return null;
    return this.delegate.attackableEntities().find((entity) => entity.id === id) ?? null;
  }
}
