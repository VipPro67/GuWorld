import {
  WorldSocket,
  type WorldSocketOptions,
} from "../network/WorldSocket.ts";
import type { BufferedWorldState } from "../network/SnapshotBuffer.ts";
import type {
  CastPayload,
  WorldEvent,
  WorldJoinPayload,
  WorldProjectileState,
  WorldRejectPayload,
  WorldSkillInfo,
  WorldWelcomePayload,
} from "../network/Protocol.ts";
import { SelfPredictor } from "../network/Prediction.ts";
import { RemoteInterpolator } from "../network/RemoteInterpolator.ts";
import { WorldClock } from "./WorldClock.ts";
import {
  clampMoveTarget,
  normalizeWorldVector,
  type WorldPoint,
} from "../world/WorldUnits.ts";

export type WorldRuntimeStatus =
  | "idle"
  | "connecting"
  | "joined"
  | "error"
  | "closed";

export interface RuntimeEntity {
  id: string;
  kind: "player" | "monster";
  partyId: string;
  x: number;
  y: number;
  hp: number;
  maxHp: number;
  shield: number;
  templateId: string;
  name: string;
  aiState: string;
  aggroTargetId: string;
  isSelf: boolean;
  stealthed?: boolean;
  flying?: boolean;
  statuses?: string[];
}

export interface SelfCombatView {
  mana: number;
  maxMana: number;
  thought?: number;
  maxThought?: number;
  cooldowns: number[];
  castSkillId: string;
  castEndTick: number;
  castWindupTicks: number;
  statuses?: string[];
  stealthed?: boolean;
  flying?: boolean;
}

export interface WorldRuntimeOptions {
  socketOptions?: WorldSocketOptions;
  /** Server speed in world units per second. Mirrors admission defaults for prediction. */
  speed?: number;
}

export interface RealtimeTactic {
  gu_id: string;
  priority: number;
  use_condition?: string;
  condition_value?: number;
}

export interface RealtimeControlSettings {
  leftClickSlot: number;
  rightClickAction: "attack" | "skill";
  rightClickSlot: number;
  autoFarm: boolean;
  farmRange: number;
  farmStopHPPercent: number;
  farmUseSkills: boolean;
  guPriorities: RealtimeTactic[];
}

export const DEFAULT_REALTIME_CONTROL_SETTINGS: RealtimeControlSettings = {
  leftClickSlot: 0,
  rightClickAction: "attack",
  rightClickSlot: 0,
  autoFarm: false,
  farmRange: 14,
  farmStopHPPercent: 25,
  farmUseSkills: false,
  guPriorities: [],
};

const DEFAULT_SPEED = 4;
const INTERPOLATION_DELAY_TICKS = 2;
const INPUT_SEND_INTERVAL = 0.1;
const INPUT_KEEPALIVE_INTERVAL = 0.15;

/**
 * Owns the realtime network session for the Phaser scene. Snapshots are read on
 * the render side only; the scene never mutates authoritative state.
 */
export class WorldRuntime {
  private socket: WorldSocket;
  private readonly socketOptions: WorldSocketOptions | undefined;
  private readonly speed: number;
  private readonly clock = new WorldClock();
  private readonly predictor = new SelfPredictor();
  private readonly interpolator = new RemoteInterpolator();
  private readonly listeners = new Set<() => void>();
  private readonly combatListeners = new Set<(event: WorldEvent) => void>();
  private readonly castRequestListeners = new Set<(slot: number) => void>();
  private readonly unsubscribers: Array<() => void> = [];
  private selectedTargetId: string | null = null;
  private combatNotice = "";
  private uiModal = false;
  private lastCorrection = 0;
  private realtimeSettings: RealtimeControlSettings = { ...DEFAULT_REALTIME_CONTROL_SETTINGS, guPriorities: [] };

  private status: WorldRuntimeStatus = "idle";
  private welcome: WorldWelcomePayload | null = null;
  private snapshot: BufferedWorldState | null = null;
  private selfId: string | null = null;
  private selfHp = 0;
  private lastJoin: WorldJoinPayload | null = null;
  private errorMessage = "";

  private inputVector: WorldPoint = { x: 0, y: 0 };
  private lastSentVector: WorldPoint = { x: 0, y: 0 };
  private localTarget: WorldPoint | null = null;
  private inputAccumulator = 0;
  private sendAccumulator = 0;

  constructor(options: WorldRuntimeOptions = {}) {
    this.socketOptions = options.socketOptions;
    this.socket = new WorldSocket(this.socketOptions);
    this.speed = options.speed ?? DEFAULT_SPEED;
    this.predictor.setSpeed(this.speed);
    this.bindSocket();
  }

  public async connect(join: WorldJoinPayload): Promise<WorldWelcomePayload | null> {
    if (this.status === "connecting") return null;
    this.lastJoin = join;
    this.errorMessage = "";
    this.combatNotice = "";
    this.status = "connecting";
    this.emit();
    try {
      const welcome = await this.socket.connect(join);
      this.welcome = welcome;
      this.selfId = welcome.entity_id;
      this.clock.sync(welcome.server_tick, welcome.tick_rate);
      this.applySnapshot(this.socket.getSnapshot());
      this.status = "joined";
      this.emit();
      return welcome;
    } catch (error) {
      this.status = "error";
      this.errorMessage = error instanceof Error ? error.message : "Không thể vào world realtime";
      this.emit();
      return null;
    }
  }

  public async reconnect(): Promise<WorldWelcomePayload | null> {
    const join = this.lastJoin;
    this.disconnect();
    if (!join) return null;
    this.socket = new WorldSocket(this.socketOptions);
    this.bindSocket();
    return this.connect(join);
  }

  public disconnect(): void {
    for (const unsubscribe of this.unsubscribers.splice(0)) unsubscribe();
    this.socket.disconnect();
    this.clock.reset();
    this.predictor.reset();
    this.interpolator.reset();
    this.welcome = null;
    this.snapshot = null;
    this.selfId = null;
    this.selectedTargetId = null;
    this.combatNotice = "";
    this.uiModal = false;
    this.lastCorrection = 0;
    this.status = "closed";
    this.emit();
  }

  public setMoveVector(vector: WorldPoint): void {
    const normalized = normalizeWorldVector(vector);
    if (normalized.x !== 0 || normalized.y !== 0) this.localTarget = null;
    this.inputVector = normalized;
    if (this.status !== "joined") return;
    if (!sameVector(normalized, this.lastSentVector)) {
      this.socket.sendInput(normalized);
      this.lastSentVector = normalized;
      this.sendAccumulator = 0;
    }
  }

  public sendMoveTarget(target: WorldPoint): boolean {
    if (this.status !== "joined") return false;
    const position = this.predictor.getPosition();
    const bounded = clampMoveTarget(position, target);
    const seq = this.socket.sendMoveTarget(bounded);
    if (seq === null) return false;
    this.localTarget = bounded;
    this.inputVector = { x: 0, y: 0 };
    this.lastSentVector = { x: 0, y: 0 };
    this.sendAccumulator = 0;
    return true;
  }

  /** Cancel click-to-move by requesting the current position, which the server resolves instantly. */
  public cancelMoveTarget(): void {
    if (!this.localTarget) return;
    if (this.status === "joined") {
      const position = this.predictor.getPosition();
      this.socket.sendMoveTarget(position);
    }
    this.localTarget = null;
  }

  /**
   * Advance local prediction and throttle held movement input. Called from the
   * Phaser scene update loop, never from React.
   */
  public update(dtSeconds: number): void {
    if (!Number.isFinite(dtSeconds) || dtSeconds <= 0) return;
    const dt = Math.min(dtSeconds, 0.1);

    if (this.localTarget) {
      const position = this.predictor.getPosition();
      const dx = this.localTarget.x - position.x;
      const dy = this.localTarget.y - position.y;
      const distance = Math.hypot(dx, dy);
      if (distance <= 0.35) {
        this.localTarget = null;
      } else {
        this.predictor.advance({ x: dx / distance, y: dy / distance }, dt);
        if (distance <= this.speed * dt + 0.05) this.localTarget = null;
      }
    } else {
      this.predictor.advance(this.inputVector, dt);
    }

    if (this.status !== "joined") return;
    this.inputAccumulator += dt;
    this.sendAccumulator += dt;
    if (this.inputVector.x === 0 && this.inputVector.y === 0) {
      if (!sameVector(this.lastSentVector, { x: 0, y: 0 }) && this.sendAccumulator >= INPUT_SEND_INTERVAL) {
        this.socket.sendInput({ x: 0, y: 0 });
        this.lastSentVector = { x: 0, y: 0 };
        this.sendAccumulator = 0;
      }
      return;
    }
    if (
      this.inputAccumulator >= INPUT_SEND_INTERVAL &&
      this.sendAccumulator >= INPUT_KEEPALIVE_INTERVAL
    ) {
      this.socket.sendInput(this.inputVector);
      this.lastSentVector = this.inputVector;
      this.inputAccumulator = 0;
      this.sendAccumulator = 0;
    }
  }

  public getEntities(): RuntimeEntity[] {
    const snapshot = this.snapshot;
    if (!snapshot) return [];
    const renderTick = Math.max(0, this.clock.estimatedTick() - INTERPOLATION_DELAY_TICKS);
    const entities: RuntimeEntity[] = [];
    for (const entity of snapshot.entities) {
      const maxHp = entity.max_hp ?? 100;
      const kind = entity.kind ?? "player";
      const partyId = entity.party_id ?? "";
      const shield = entity.shield ?? 0;
      const presentation = {
        templateId: entity.template_id ?? "",
        name: entity.name ?? "",
        aiState: entity.ai_state ?? "",
        aggroTargetId: entity.aggro_target_id ?? "",
        stealthed: entity.stealthed,
        flying: entity.flying,
        statuses: entity.statuses,
      };
      if (entity.entity_id === this.selfId) {
        const position = this.predictor.getPosition();
        entities.push({ id: entity.entity_id, kind, partyId, x: position.x, y: position.y, hp: entity.hp, maxHp, shield, isSelf: true, ...presentation });
        continue;
      }
      const position = this.interpolator.positionAt(entity.entity_id, renderTick) ?? { x: entity.x, y: entity.y };
      entities.push({ id: entity.entity_id, kind, partyId, x: position.x, y: position.y, hp: entity.hp, maxHp, shield, isSelf: false, ...presentation });
    }
    return entities;
  }

  public getSelfPosition(): WorldPoint {
    return this.predictor.getPosition();
  }

  public getSelfHp(): number {
    return this.selfHp;
  }

  public getStatus(): WorldRuntimeStatus {
    return this.status;
  }

  public getError(): string {
    return this.errorMessage;
  }

  public getWelcome(): WorldWelcomePayload | null {
    return this.welcome;
  }

  public getSnapshot(): BufferedWorldState | null {
    return this.snapshot;
  }

  public getMoveTarget(): WorldPoint | null {
    return this.localTarget ? { ...this.localTarget } : null;
  }

  public getSkills(): WorldSkillInfo[] {
    return this.welcome?.skills ? this.welcome.skills.map((skill) => ({ ...skill })) : [];
  }

  public getSelfCombat(): SelfCombatView {
    const self = this.snapshot?.entities.find((entity) => entity.entity_id === this.selfId);
    const cooldowns = self?.cooldowns ? [...self.cooldowns] : new Array(7).fill(0);
    while (cooldowns.length < 7) cooldowns.push(0);
    return {
      mana: self?.mana ?? 0,
      maxMana: self?.max_mana ?? 0,
      thought: self?.thought,
      maxThought: self?.max_thought,
      cooldowns,
      castSkillId: self?.cast_skill_id ?? "",
      castEndTick: self?.cast_end_tick ?? 0,
      castWindupTicks: self?.cast_windup_ticks ?? 0,
      statuses: self?.statuses,
      stealthed: self?.stealthed,
      flying: self?.flying,
    };
  }

  public estimatedTick(): number {
    return this.clock.estimatedTick();
  }

  public getSelfId(): string | null {
    return this.selfId;
  }

  public getSelfEntity(): RuntimeEntity | null {
    if (!this.selfId) return null;
    return this.getEntity(this.selfId);
  }

  public getCombatNotice(): string {
    return this.combatNotice;
  }

  /** React business panels set this to block movement/hotkeys in Phaser. */
  public setUiModal(open: boolean): void {
    if (this.uiModal === open) return;
    this.uiModal = open;
    if (open) {
      this.setMoveVector({ x: 0, y: 0 });
      this.cancelMoveTarget();
    }
    this.emit();
  }

  public isUiModal(): boolean {
    return this.uiModal;
  }

  public getDebugStats(): {
    status: WorldRuntimeStatus;
    tick: number;
    tickRate: number;
    entities: number;
    projectiles: number;
    selfX: number;
    selfY: number;
    lastCorrection: number;
    network: { messages: number; bytes: number; snapshots: number; lastMessageAt: number };
  } {
    const self = this.getSelfPosition();
    return {
      status: this.status,
      tick: this.snapshot?.tick ?? 0,
      tickRate: this.clock.getTickRate(),
      entities: this.snapshot?.entities.length ?? 0,
      projectiles: this.snapshot?.projectiles?.length ?? 0,
      selfX: self.x,
      selfY: self.y,
      lastCorrection: this.lastCorrection,
      network: this.socket.getMetrics(),
    };
  }

  public getProjectiles(): WorldProjectileState[] {
    return this.snapshot?.projectiles?.map((projectile) => ({ ...projectile })) ?? [];
  }

  public getSelectedTargetId(): string | null {
    return this.selectedTargetId;
  }

  public setSelectedTarget(entityId: string | null): void {
    if (this.selectedTargetId === entityId) return;
    this.selectedTargetId = entityId;
    this.emit();
  }

  public getEntity(entityId: string): RuntimeEntity | null {
    return this.getEntities().find((entity) => entity.id === entityId) ?? null;
  }

  public onCastRequest(listener: (slot: number) => void): () => void {
    this.castRequestListeners.add(listener);
    return () => this.castRequestListeners.delete(listener);
  }

  /** Called by the React HUD; the scene owns pointer/target aim state. */
  public requestCast(slot: number): void {
    for (const listener of this.castRequestListeners) listener(slot);
  }

  public requestBasicAttack(): void {
    const targetId = this.selectedTargetId;
    if (targetId) this.sendBasicAttack(targetId);
  }

  public getTickRate(): number {
    return this.clock.getTickRate();
  }

  public setRealtimeSettings(settings: Partial<RealtimeControlSettings> | null | undefined): void {
    const value = settings ?? {};
    this.realtimeSettings = {
      leftClickSlot: Number.isInteger(value.leftClickSlot) && value.leftClickSlot! >= 0 && value.leftClickSlot! <= 6
        ? value.leftClickSlot! : DEFAULT_REALTIME_CONTROL_SETTINGS.leftClickSlot,
      rightClickAction: value.rightClickAction === "skill" ? "skill" : "attack",
      rightClickSlot: Number.isInteger(value.rightClickSlot) && value.rightClickSlot! >= 1 && value.rightClickSlot! <= 6
        ? value.rightClickSlot! : 0,
      autoFarm: value.autoFarm === true,
      farmRange: Number.isFinite(value.farmRange) ? Math.max(4, Math.min(32, value.farmRange!)) : DEFAULT_REALTIME_CONTROL_SETTINGS.farmRange,
      farmStopHPPercent: Number.isFinite(value.farmStopHPPercent) ? Math.max(0, Math.min(90, value.farmStopHPPercent!)) : DEFAULT_REALTIME_CONTROL_SETTINGS.farmStopHPPercent,
      farmUseSkills: value.farmUseSkills === true,
      guPriorities: Array.isArray(value.guPriorities) ? value.guPriorities.map((tactic) => ({ ...tactic })) : [],
    };
    this.emit();
  }

  public getRealtimeSettings(): RealtimeControlSettings {
    return { ...this.realtimeSettings, guPriorities: this.realtimeSettings.guPriorities.map((tactic) => ({ ...tactic })) };
  }

  public subscribe(listener: () => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  public onReject(listener: (reject: WorldRejectPayload) => void): () => void {
    return this.socket.onReject(listener);
  }

  public onError(listener: (error: Error) => void): () => void {
    return this.socket.onError(listener);
  }

  public onCombatEvents(listener: (event: WorldEvent) => void): () => void {
    this.combatListeners.add(listener);
    return () => this.combatListeners.delete(listener);
  }

  public sendCast(cast: Omit<CastPayload, "seq">): number | null {
    if (this.status !== "joined") return null;
    return this.socket.sendCast(cast);
  }

  public sendBasicAttack(targetId: string): number | null {
    if (this.status !== "joined") return null;
    return this.socket.sendBasicAttack(targetId);
  }

  private bindSocket(): void {
    this.unsubscribers.push(
      this.socket.onSnapshot((state) => this.applySnapshot(state)),
      this.socket.onResyncRequired((reason) => {
        this.status = "error";
        this.errorMessage = `Mất đồng bộ snapshot: ${reason}`;
        this.emit();
      }),
      this.socket.onError((error) => {
        this.errorMessage = error.message;
        this.emit();
      }),
      this.socket.onEvents((event) => {
        if (event.type === "CAST_REJECTED") {
          const reason = typeof event.payload?.reason === "string" ? event.payload.reason : "rejected";
          this.combatNotice = `Cast: ${reason}`;
        } else if (event.type === "STATUS_APPLIED") {
          const status = typeof event.payload?.reason === "string" ? event.payload.reason : "STATUS";
          this.combatNotice = `Status: ${status}`;
        } else if (event.type === "HIDDEN_PASSIVE_TRIGGERED") {
          this.combatNotice = "Nội tại ẩn kích hoạt";
        } else if (event.type === "CAST_ACCEPTED") {
          this.combatNotice = "";
        } else if (event.type === "MOVE_TARGET_REJECTED") {
          this.localTarget = null;
          this.combatNotice = "Đường đi bị chặn";
        }
        for (const listener of this.combatListeners) listener(event);
        this.emit();
      }),
    );
  }

  private applySnapshot(state: BufferedWorldState | null): void {
    if (!state) return;
    this.snapshot = state;
    if (Math.abs(this.clock.estimatedTick() - state.tick) > 4) {
      this.clock.sync(state.tick, this.clock.getTickRate());
    }
    const self = state.entities.find((entity) => entity.entity_id === this.selfId);
    if (self) {
      this.selfHp = self.hp;
      const correction = this.predictor.softReconcile({ x: self.x, y: self.y }, state.ackSeq);
      this.lastCorrection = correction.correction;
      if (correction.snapped) this.localTarget = null;
    }
    this.interpolator.push(
      state.tick,
      state.entities.filter((entity) => entity.entity_id !== this.selfId),
    );
    this.emit();
  }

  private emit(): void {
    for (const listener of this.listeners) listener();
  }
}

function sameVector(left: WorldPoint, right: WorldPoint): boolean {
  return Math.abs(left.x - right.x) < 0.001 && Math.abs(left.y - right.y) < 0.001;
}
