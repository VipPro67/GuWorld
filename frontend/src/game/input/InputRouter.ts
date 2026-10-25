import { normalizeWorldVector, type WorldPoint } from "../world/WorldUnits.ts";

const KEY_VECTORS: Record<string, WorldPoint> = {
  KeyW: { x: 0, y: -1 },
  ArrowUp: { x: 0, y: -1 },
  KeyS: { x: 0, y: 1 },
  ArrowDown: { x: 0, y: 1 },
  KeyA: { x: -1, y: 0 },
  ArrowLeft: { x: -1, y: 0 },
  KeyD: { x: 1, y: 0 },
  ArrowRight: { x: 1, y: 0 },
};

export type InputBlocker = "none" | "ui" | "modal" | "aiming";

const BLOCKER_ORDER: InputBlocker[] = ["none", "aiming", "modal", "ui"];

/**
 * Single arbitration point for gameplay input. Precedence follows ADR-02:
 * UIFocused > Modal > AimingSkill > DirectWASD > ClickPath > Idle.
 */
export class InputRouter {
  private readonly pressed = new Set<string>();
  private joystickVector: WorldPoint = { x: 0, y: 0 };
  private uiFocused = false;
  private modal = false;
  private aiming = false;

  public setKey(code: string, pressed: boolean): boolean {
    if (!(code in KEY_VECTORS)) return false;
    if (pressed) this.pressed.add(code);
    else this.pressed.delete(code);
    return true;
  }

  public setJoystick(vector: WorldPoint): void {
    if (!Number.isFinite(vector.x) || !Number.isFinite(vector.y)) {
      this.joystickVector = { x: 0, y: 0 };
      return;
    }
    this.joystickVector = { x: vector.x, y: vector.y };
  }

  public setUIFocused(focused: boolean): void {
    this.uiFocused = focused;
  }

  public setModal(open: boolean): void {
    this.modal = open;
  }

  public setAiming(aiming: boolean): void {
    this.aiming = aiming;
  }

  public blocker(): InputBlocker {
    if (this.uiFocused) return "ui";
    if (this.modal) return "modal";
    if (this.aiming) return "aiming";
    return "none";
  }

  public canDirectMove(): boolean {
    return this.blocker() === "none";
  }

  public canClickMove(): boolean {
    return this.blocker() === "none";
  }

  public movement(): WorldPoint {
    if (!this.canDirectMove()) return { x: 0, y: 0 };
    if (this.joystickVector.x !== 0 || this.joystickVector.y !== 0) {
      return normalizeWorldVector(this.joystickVector);
    }
    let x = 0;
    let y = 0;
    for (const code of this.pressed) {
      const vector = KEY_VECTORS[code];
      x += vector.x;
      y += vector.y;
    }
    return normalizeWorldVector({ x, y });
  }

  public hasMovementKeys(): boolean {
    if (this.joystickVector.x !== 0 || this.joystickVector.y !== 0) return true;
    for (const code of this.pressed) {
      if (code in KEY_VECTORS) return true;
    }
    return false;
  }

  /** Clear every held key and transient aim state, e.g. on blur. */
  public reset(): void {
    this.pressed.clear();
    this.joystickVector = { x: 0, y: 0 };
    this.aiming = false;
  }

  /** Highest blocker in the precedence chain, useful for debug overlays/tests. */
  public static strongest(blockers: InputBlocker[]): InputBlocker {
    let strongest: InputBlocker = "none";
    for (const blocker of blockers) {
      if (BLOCKER_ORDER.indexOf(blocker) > BLOCKER_ORDER.indexOf(strongest)) strongest = blocker;
    }
    return strongest;
  }
}
