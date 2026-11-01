import {
  WorldSocket,
  WorldSocketError,
  type WorldSocketOptions,
} from "../network/WorldSocket.ts";
import type { BufferedWorldState } from "../network/SnapshotBuffer.ts";
import type {
  BasicAttackPayload,
  CastPayload,
  WorldEvent,
  WorldJoinPayload,
  WorldProjectileState,
  WorldRejectPayload,
  WorldSkillInfo,
  WorldTransferPayload,
  WorldWelcomePayload,
  WorldTransformationState,
} from "../network/Protocol.ts";
import { WORLD_COOLDOWN_SLOTS } from "../network/Protocol.ts";
import { SelfPredictor } from "../network/Prediction.ts";
import { RemoteInterpolator } from "../network/RemoteInterpolator.ts";
import { WorldClock } from "./WorldClock.ts";
import {
  clampMoveTarget,
  normalizeWorldVector,
  type WorldPoint,
} from "../world/WorldUnits.ts";
import type { MapPortal } from "../world/MapLoader.ts";

// Active realtime zones publish snapshots at 20Hz. A few seconds without any
// protocol message indicates a half-open socket, not ordinary snapshot jitter.
const STALE_WORLD_CONNECTION_MS = 3_000;
const STALE_WORLD_RECONNECT_COOLDOWN_MS = 5_000;

export type WorldRuntimeStatus =
  | "idle"
  | "connecting"
  | "joined"
  | "error"
  | "closed";

export interface RuntimeEntity {
  id: string;
  kind: "player" | "monster" | "companion";
  partyId: string;
  pvpEnabled: boolean;
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
  formActive?: boolean;
  transformation?: WorldTransformationState;
  statuses?: string[];
  ownerId?: string;
  companionKind?: string;
  stance?: "DEFEND" | "FOCUS" | "FOLLOW";
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
  /** Fallback speed in world units per second for servers without movement_speed. */
  speed?: number;
  nowMs?: () => number;
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
const MAX_SELF_RECONCILE_LEAD_TICKS = 5;
const MAX_SELF_LATENCY_MS = 250;
const INPUT_SEND_INTERVAL = 0.1;
const INPUT_KEEPALIVE_INTERVAL = 0.15;

/**
 * Owns the realtime network session for the Phaser scene. Snapshots are read on
 * the render side only; the scene never mutates authoritative state.
 */
export class WorldRuntime {
  private socket: WorldSocket;
  private readonly socketOptions: WorldSocketOptions | undefined;
  private speed: number;
  private predictionSpeed: number;
  private readonly clock = new WorldClock();
  private readonly predictor = new SelfPredictor();
  private readonly interpolator = new RemoteInterpolator();
  private readonly listeners = new Set<() => void>();
  private readonly combatListeners = new Set<(event: WorldEvent) => void>();
  private readonly castRequestListeners = new Set<(slot: number) => void>();
  private readonly transferListeners = new Set<(transfer: WorldTransferPayload) => void>();
  private readonly unsubscribers: Array<() => void> = [];
  private selectedTargetId: string | null = null;
  private combatNotice = "";
  private uiModal = false;
  private aimingSlot: number | null = null;
  private lastCorrection = 0;
  private realtimeSettings: RealtimeControlSettings = { ...DEFAULT_REALTIME_CONTROL_SETTINGS, guPriorities: [] };

  private status: WorldRuntimeStatus = "idle";
  private welcome: WorldWelcomePayload | null = null;
  private mapPortals: MapPortal[] = [];
  private snapshot: BufferedWorldState | null = null;
  private selfId: string | null = null;
  private selfHp = 0;
  private lastJoin: WorldJoinPayload | null = null;
  private errorMessage = "";
  private errorCode = "";
  private joinZoneResolver: (() => string | null) | null = null;

  private inputVector: WorldPoint = { x: 0, y: 0 };
  private activeDash = false;
  private dashStartTick = 0;
  private lastSentVector: WorldPoint = { x: 0, y: 0 };
  private readonly nowMs: () => number;
  private sentMovementAt = new Map<number, number>();
  private oneWayLatencyMs = 0;
  private stopSeq = 0;
  private localTarget: WorldPoint | null = null;
  private pendingTarget: WorldPoint | null = null;
  private inputAccumulator = 0;
  private sendAccumulator = 0;
  private lastStaleReconnectAt = 0;

  constructor(options: WorldRuntimeOptions = {}) {
    this.socketOptions = options.socketOptions;
    this.nowMs = options.nowMs ?? Date.now;
    this.socket = new WorldSocket(this.socketOptions);
    this.speed = options.speed ?? DEFAULT_SPEED;
    this.predictionSpeed = this.speed;
    this.predictor.setSpeed(this.speed);
    this.bindSocket();
  }

  public async connect(join: WorldJoinPayload): Promise<WorldWelcomePayload | null> {
    if (this.status === "connecting") return null;
    this.lastJoin = join;
    this.errorMessage = "";
    this.errorCode = "";
    this.combatNotice = "";
    this.status = "connecting";
    this.emit();
    try {
      const welcome = await this.socket.connect(join);
      this.welcome = welcome;
      this.activeDash = false;
      this.dashStartTick = 0;
      this.speed = welcome.movement_speed ?? this.speed;
      this.predictionSpeed = this.speed;
      this.predictor.setSpeed(this.speed);
      this.selfId = welcome.entity_id;
      this.clock.sync(welcome.server_tick, welcome.tick_rate);
      this.applySnapshot(this.socket.getSnapshot());
      this.status = "joined";
      this.emit();
      return welcome;
    } catch (error) {
      this.status = "error";
      this.errorMessage = error instanceof Error ? error.message : "Không thể vào world realtime";
      this.errorCode = error instanceof WorldSocketError ? error.code : "world_connect_failed";
      this.emit();
      return null;
    }
  }

  public async reconnect(): Promise<WorldWelcomePayload | null> {
    const join = this.lastJoin;
    this.disconnect();
    if (!join) return null;
    let nextJoin = join;
    const resolvedZone = this.joinZoneResolver?.()?.trim() ?? "";
    if (resolvedZone && resolvedZone !== join.zone_id) {
      nextJoin = { ...join, zone_id: resolvedZone };
    }
    this.socket = new WorldSocket(this.socketOptions);
    this.bindSocket();
    return this.connect(nextJoin);
  }

  /**
   * Reconnect into a different region map after a server-owned border handoff.
   * `reconnect()` alone would replay the previous zone_id and be rejected by
   * admission.
   */
  public async reconnectTo(zoneId: string): Promise<WorldWelcomePayload | null> {
    if (!this.lastJoin || !zoneId) return null;
    this.lastJoin = { ...this.lastJoin, zone_id: zoneId };
    return this.reconnect();
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
    this.activeDash = false;
    this.dashStartTick = 0;
    this.selectedTargetId = null;
    this.combatNotice = "";
    this.uiModal = false;
    this.lastCorrection = 0;
    this.sentMovementAt.clear();
    this.oneWayLatencyMs = 0;
    this.stopSeq = 0;
    this.pendingTarget = null;
    this.status = "closed";
    this.emit();
  }

  public setMoveVector(vector: WorldPoint): void {
    const normalized = normalizeWorldVector(vector);
    const priorMagnitude = Math.hypot(this.inputVector.x, this.inputVector.y);
    const nextMagnitude = Math.hypot(normalized.x, normalized.y);
    const directionalCosine = priorMagnitude > 0 && nextMagnitude > 0
      ? (normalized.x * this.inputVector.x + normalized.y * this.inputVector.y) / (priorMagnitude * nextMagnitude)
      : -1;
    if (nextMagnitude > 0 && directionalCosine < 0.95) {
      this.predictionSpeed = this.speed;
      this.predictor.setSpeed(this.speed);
    }
    if (normalized.x !== 0 || normalized.y !== 0) {
      this.localTarget = null;
      this.pendingTarget = null;
    }
    this.inputVector = normalized;
    if (this.status !== "joined") return;
    if (!sameVector(normalized, this.lastSentVector)) {
      const seq = this.socket.sendInput(normalized);
      if (seq !== null) {
        this.recordMovementSend(seq);
        this.stopSeq = normalized.x === 0 && normalized.y === 0 ? seq : 0;
      }
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
    this.pendingTarget = bounded;
    this.predictionSpeed = this.speed;
    this.predictor.setSpeed(this.speed);
    this.recordMovementSend(seq);
    this.stopSeq = 0;
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
      const seq = this.socket.sendMoveTarget(position);
      if (seq !== null) this.recordMovementSend(seq);
    }
    this.localTarget = null;
    this.pendingTarget = null;
  }

  /**
   * Advance local prediction and throttle held movement input. Called from the
   * Phaser scene update loop, never from React.
   */
  public update(dtSeconds: number): void {
    if (!Number.isFinite(dtSeconds) || dtSeconds <= 0) return;
    const dt = Math.min(dtSeconds, 0.1);

    if (this.status === "joined") {
      const lastSnapshotAt = this.socket.getMetrics().lastSnapshotAt;
      const now = Date.now();
      if (
        lastSnapshotAt > 0 &&
        now - lastSnapshotAt > STALE_WORLD_CONNECTION_MS &&
        now - this.lastStaleReconnectAt > STALE_WORLD_RECONNECT_COOLDOWN_MS
      ) {
        // WebSockets can remain OPEN after a proxy/server path silently dies.
        // Rejoin with a bounded retry rate; if admission fails, the existing
        // connection error UI lets the player retry rather than leaving
        // movement falsely JOINED.
        this.lastStaleReconnectAt = now;
        void this.reconnect();
      }
    }

    if (this.activeDash && this.clock.estimatedTick() >= this.dashStartTick) {
      // Dash travel is server owned; held input is still sent for movement after it ends.
    } else if (this.localTarget) {
      const position = this.predictor.getPosition();
      const dx = this.localTarget.x - position.x;
      const dy = this.localTarget.y - position.y;
      const distance = Math.hypot(dx, dy);
      if (distance <= 0.35) {
        this.localTarget = null;
      } else {
        this.predictor.advance({ x: dx / distance, y: dy / distance }, dt);
        if (distance <= this.predictionSpeed * dt + 0.05) this.localTarget = null;
      }
    } else {
      this.predictor.advance(this.inputVector, dt);
    }

    if (this.status !== "joined") return;
    this.inputAccumulator += dt;
    this.sendAccumulator += dt;
    if (this.inputVector.x === 0 && this.inputVector.y === 0) {
      if (!sameVector(this.lastSentVector, { x: 0, y: 0 }) && this.sendAccumulator >= INPUT_SEND_INTERVAL) {
        const seq = this.socket.sendInput({ x: 0, y: 0 });
        if (seq !== null) this.recordMovementSend(seq);
        this.lastSentVector = { x: 0, y: 0 };
        this.sendAccumulator = 0;
      }
      return;
    }
    if (
      this.inputAccumulator >= INPUT_SEND_INTERVAL &&
      this.sendAccumulator >= INPUT_KEEPALIVE_INTERVAL
    ) {
      const seq = this.socket.sendInput(this.inputVector);
      if (seq !== null) this.recordMovementSend(seq);
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
      const pvpEnabled = entity.pvp_enabled ?? false;
      const shield = entity.shield ?? 0;
      const presentation = {
        templateId: entity.template_id ?? "",
        name: entity.name ?? "",
        aiState: entity.ai_state ?? "",
        aggroTargetId: entity.aggro_target_id ?? "",
        stealthed: entity.stealthed,
        flying: entity.flying,
        formActive: entity.form_active,
        transformation: entity.transformation,
        statuses: entity.statuses,
        ownerId: entity.owner_id,
        companionKind: entity.companion_kind,
        stance: entity.stance,
      };
      if (entity.entity_id === this.selfId) {
        const position = this.predictor.getPosition();
        entities.push({ id: entity.entity_id, kind, partyId, pvpEnabled, x: position.x, y: position.y, hp: entity.hp, maxHp, shield, isSelf: true, ...presentation });
        continue;
      }
      const position = this.interpolator.positionAt(entity.entity_id, renderTick) ?? { x: entity.x, y: entity.y };
      entities.push({ id: entity.entity_id, kind, partyId, pvpEnabled, x: position.x, y: position.y, hp: entity.hp, maxHp, shield, isSelf: false, ...presentation });
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

  public getErrorCode(): string {
    return this.errorCode;
  }

  /** Resolves the freshest server-owned zone before a reconnect. */
  public setJoinZoneResolver(resolver: (() => string | null) | null): void {
    this.joinZoneResolver = resolver;
  }

  public getWelcome(): WorldWelcomePayload | null {
    return this.welcome;
  }

  public setMapPortals(portals: MapPortal[]): void {
    this.mapPortals = [...portals];
    this.emit();
  }

  public getMapPortals(): MapPortal[] {
    return this.mapPortals;
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
    const cooldowns = self?.cooldowns ? [...self.cooldowns] : new Array(WORLD_COOLDOWN_SLOTS).fill(0);
    while (cooldowns.length < WORLD_COOLDOWN_SLOTS) cooldowns.push(0);
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

  public getAimingSlot(): number | null {
    return this.aimingSlot;
  }

  public setAimingSlot(slot: number | null): void {
    if (this.aimingSlot === slot) return;
    this.aimingSlot = slot;
    this.emit();
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
    network: { messages: number; bytes: number; snapshots: number; lastMessageAt: number; lastSnapshotAt: number };
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
    if (targetId) this.sendBasicAttack({ target_id: targetId });
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

  public onTransfer(listener: (transfer: WorldTransferPayload) => void): () => void {
    this.transferListeners.add(listener);
    return () => this.transferListeners.delete(listener);
  }

  public onCombatEvents(listener: (event: WorldEvent) => void): () => void {
    this.combatListeners.add(listener);
    return () => this.combatListeners.delete(listener);
  }

  public sendCast(cast: Omit<CastPayload, "seq">): number | null {
    if (this.status !== "joined") return null;
    return this.socket.sendCast(cast);
  }

  public sendBasicAttack(attack: Omit<BasicAttackPayload, "seq">): number | null {
    if (this.status !== "joined") return null;
    return this.socket.sendBasicAttack(attack);
  }

  public captureTarget(targetId: string, guId?: string): number | null {
    if (this.status !== "joined") return null;
    return this.socket.sendCapture(targetId, guId);
  }

  public setCompanionStance(stance: "DEFEND" | "FOCUS" | "FOLLOW", focusTargetId?: string): number | null {
    if (this.status !== "joined") return null;
    return this.socket.sendCompanionStance(stance, focusTargetId);
  }

  private bindSocket(): void {
    this.unsubscribers.push(
      this.socket.onSnapshot((state) => this.applySnapshot(state)),
      this.socket.onResyncRequired((reason) => {
        this.status = "error";
        this.errorMessage = `Mất đồng bộ snapshot: ${reason}`;
        this.errorCode = "snapshot_resync_required";
        this.emit();
      }),
      this.socket.onError((error) => {
        this.errorMessage = error.message;
        this.errorCode = error instanceof WorldSocketError ? error.code : "world_socket_error";
        this.emit();
      }),
      this.socket.onTransfer((transfer) => {
        this.combatNotice = `Đang chuyển vùng: ${transfer.to_zone_id}`;
        this.emit();
        for (const listener of this.transferListeners) listener(transfer);
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
          if (event.source_id === this.selfId) {
            const skillId = event.payload?.skill_id;
            if (this.welcome?.skills?.some((skill) => skill.id === skillId && (skill.move_distance ?? 0) > 0)) {
              this.activeDash = true;
              const windup = event.payload?.duration_ticks;
              this.dashStartTick = event.tick + (typeof windup === "number" ? windup : 0);
            }
          }
        } else if (event.type === "DASH_ENDED" && event.source_id === this.selfId) {
          this.activeDash = false;
        } else if (event.type === "CAST_INTERRUPTED" && event.source_id === this.selfId) {
          this.activeDash = false;
        } else if (event.type === "MOVE_TARGET_REJECTED") {
          this.localTarget = null;
          this.pendingTarget = null;
          this.combatNotice = "Đường đi bị chặn";
        }
        for (const listener of this.combatListeners) listener(event);
        this.emit();
      }),
    );
  }

  private applySnapshot(state: BufferedWorldState | null): void {
    if (!state) return;
    const previous = this.snapshot;
    this.updateMovementLatency(state.ackSeq);
    this.snapshot = state;
    if (Math.abs(this.clock.estimatedTick() - state.tick) > 4) {
      this.clock.sync(state.tick, this.clock.getTickRate());
    }
    const self = state.entities.find((entity) => entity.entity_id === this.selfId);
    if (self) {
      this.selfHp = self.hp;
      if (this.pendingTarget && Math.hypot(self.x - this.pendingTarget.x, self.y - this.pendingTarget.y) <= 0.15) {
        this.pendingTarget = null;
      }
      // Project using movement the server actually made between snapshots.
      // This yields zero lead at walls/root and the reduced speed during slow.
      const priorSelf = previous?.entities.find((entity) => entity.entity_id === this.selfId);
      const tickDelta = previous ? state.tick - previous.tick : 0;
      const observedSpeed = priorSelf && tickDelta > 0
        ? Math.hypot(self.x - priorSelf.x, self.y - priorSelf.y) * this.clock.getTickRate() / tickDelta
        : null;
      if (observedSpeed !== null && observedSpeed <= this.speed * 1.5 &&
        (this.inputVector.x !== 0 || this.inputVector.y !== 0 || this.pendingTarget)) {
        this.predictionSpeed = observedSpeed;
        this.predictor.setSpeed(this.predictionSpeed);
      }
      const leadTicks = Math.min(
        MAX_SELF_RECONCILE_LEAD_TICKS,
        Math.max(0, this.clock.estimatedTick() - state.tick, this.oneWayLatencyMs * this.clock.getTickRate() / 1000),
      );
      const position = { x: self.x, y: self.y };
      const stillMoving = this.inputVector.x !== 0 || this.inputVector.y !== 0 || this.pendingTarget !== null || (this.stopSeq > 0 && state.ackSeq < this.stopSeq);
      if (priorSelf && tickDelta > 0 && leadTicks > 0 && stillMoving) {
        const dx = self.x - priorSelf.x;
        const dy = self.y - priorSelf.y;
        if (observedSpeed !== null && observedSpeed <= this.speed * 1.5) {
          let projectedX = dx * leadTicks / tickDelta;
          let projectedY = dy * leadTicks / tickDelta;
          if (this.pendingTarget) {
            const remaining = Math.hypot(this.pendingTarget.x - position.x, this.pendingTarget.y - position.y);
            const projectedDistance = Math.hypot(projectedX, projectedY);
            if (projectedDistance > remaining && projectedDistance > 0) {
              projectedX *= remaining / projectedDistance;
              projectedY *= remaining / projectedDistance;
            }
          }
          position.x += projectedX;
          position.y += projectedY;
        }
      }
      const correction = this.activeDash && state.tick >= this.dashStartTick
        ? this.predictor.softReconcile(position, state.ackSeq, 0, 1)
        : this.predictor.softReconcile(position, state.ackSeq, 2, 0.08);
      this.lastCorrection = correction.correction;
      if (correction.snapped) this.localTarget = null;
    }
    this.interpolator.push(
      state.tick,
      state.entities.filter((entity) => entity.entity_id !== this.selfId),
    );
    this.emit();
  }

  private recordMovementSend(seq: number): void {
    this.sentMovementAt.set(seq, this.nowMs());
    if (this.sentMovementAt.size > 64) this.sentMovementAt.delete(this.sentMovementAt.keys().next().value!);
  }

  private updateMovementLatency(ackSeq: number): void {
    let latestSentAt: number | undefined;
    for (const [seq, sentAt] of this.sentMovementAt) {
      if (seq > ackSeq) break;
      latestSentAt = sentAt;
      this.sentMovementAt.delete(seq);
    }
    if (latestSentAt === undefined) return;
    const sample = Math.min(MAX_SELF_LATENCY_MS, Math.max(0, (this.nowMs() - latestSentAt) / 2));
    this.oneWayLatencyMs = this.oneWayLatencyMs === 0 ? sample : this.oneWayLatencyMs * 0.75 + sample * 0.25;
  }

  private emit(): void {
    for (const listener of this.listeners) listener();
  }
}

function sameVector(left: WorldPoint, right: WorldPoint): boolean {
  return Math.abs(left.x - right.x) < 0.001 && Math.abs(left.y - right.y) < 0.001;
}
