import { diagnoseConnectionFailure } from "./ConnectionDiagnostics.ts";
import { BACKEND_URL } from "../../lib/config.ts";
import {
  buildRealtimeWTURL,
  realtimeClient,
  RealtimeUnsupportedError,
  type RealtimeClient,
  type WebTransportDatagramDuplexLike,
  type WebTransportLike,
  type WebTransportCtor,
} from "../../lib/realtime/RealtimeClient.ts";
import type { WebTransportStreamLike } from "../../lib/realtime/UIStream.ts";
import { WorldRealtimeError } from "./WorldRealtimeError.ts";
import type { WorldTransportLike } from "./Transport.ts";
import type {
  BasicAttackPayload,
  CapturePayload,
  CastPayload,
  CompanionStancePayload,
  WorldEntityState,
  WorldEvent,
  WorldJoinPayload,
  WorldProjectileState,
  WorldRejectPayload,
  WorldSnapshotPayload,
  WorldTransferPayload,
  WorldWelcomePayload,
} from "./Protocol.ts";
import type { BufferedWorldState } from "./SnapshotBuffer.ts";
import { SnapshotBuffer } from "./SnapshotBuffer.ts";
import {
  DatagramOp,
  MAX_DATAGRAM_SIZE,
  Op,
  Reader,
  STREAM_WORLD,
  VERSION,
  Writer,
  decodeDatagram,
  decodePongDatagram,
  decodeSnapshotDelta,
  decodeEvents,
  decodeSnapshotFull,
  decodeTransfer,
  decodeWelcome,
  EVENT_TYPE_NAMES,
  dequantize,
  encodeAttack,
  encodeCapture,
  encodeCast,
  encodeFrame,
  encodeInput,
  encodeJoin,
  encodeMoveTarget,
  encodePingDatagram,
  encodeStance,
  readSpawn,
  type DeltaEntity,
  type Projectile,
  type Spawn,
} from "./v2/Codec.ts";

export interface WorldTransportOptions {
  url?: string;
  getToken?: () => string | null;
  getTabId?: () => string | null;
  createTransport?: (url: string) => WebTransportLike;
  client?: RealtimeClient;
}

const JOIN_TIMEOUT_MS = 10_000;
const RESYNC_COOLDOWN_MS = 1_000;
const sharedTextDecoder = new TextDecoder();

/** Protocol v2 client over WebTransport; the only world transport. */
export class WorldTransport implements WorldTransportLike {
  private readonly options: WorldTransportOptions;
  private readonly buffer = new SnapshotBuffer();
  private transport: WebTransportLike | null = null;
  private ownsTransport = false;
  private releaseSharedWorld: (() => void) | null = null;
  private streamWriter: WritableStreamDefaultWriter<Uint8Array> | null = null;
  private streamReader: ReadableStreamDefaultReader<Uint8Array> | null = null;
  private datagramWriter: WritableStreamDefaultWriter<Uint8Array> | null = null;
  private datagramReader: ReadableStreamDefaultReader<Uint8Array> | null = null;
  private incoming: Uint8Array<ArrayBufferLike> = new Uint8Array(0);
  private incomingOffset = 0;
  private welcome: WorldWelcomePayload | null = null;
  private joined = false;
  private explicitClose = false;
  private nextInputSeq = 0;
  private nextRequestId = 1;
  private joinTimeout: ReturnType<typeof setTimeout> | null = null;
  private lastResyncSentAt = 0;

  private netToSpawn = new Map<number, Spawn>();
  private entityByNet = new Map<number, WorldEntityState>();
  private netByEntity = new Map<string, number>();

  private snapshotListeners = new Set<(state: BufferedWorldState) => void>();
  private eventListeners = new Set<(event: WorldEvent) => void>();
  private rejectListeners = new Set<(reject: WorldRejectPayload) => void>();
  private transferListeners = new Set<(transfer: WorldTransferPayload) => void>();
  private resyncListeners = new Set<(reason: string) => void>();
  private errorListeners = new Set<(error: Error) => void>();

  private messageCount = 0;
  private byteCount = 0;
  private messagesSent = 0;
  private bytesSent = 0;
  private snapshotCount = 0;
  private lastMessageAt = 0;
  private lastSnapshotAt = 0;
  private lastCloseCode = 0;
  private lastRttMs = 0;

  constructor(options: WorldTransportOptions = {}) {
    this.options = options;
  }

  public async connect(join: WorldJoinPayload): Promise<WorldWelcomePayload> {
    if (this.joined) return this.welcome as WorldWelcomePayload;
    const token = this.options.getToken?.() ?? readStorage("localStorage", "gu_auth_token");
    if (!token) throw new WorldRealtimeError("Chưa có token đăng nhập", "missing_auth_token");

    this.explicitClose = false;
    this.nextInputSeq = 0;
    this.incoming = new Uint8Array(0);
    this.incomingOffset = 0;
    this.lastResyncSentAt = 0;
    this.buffer.reset();

    const useSharedClient = Boolean(this.options.client) || !this.options.createTransport;
    if (!useSharedClient) {
      const url = buildRealtimeWTURL(
        this.options.url,
        token,
        this.options.getTabId?.() ?? readStorage("sessionStorage", "gu_gameplay_tab_id"),
      );
      const factory = this.options.createTransport;
      const TransportCtor = defaultTransportFactory();
      if (!factory && !TransportCtor) {
        throw new WorldRealtimeError("Trình duyệt không hỗ trợ WebTransport", "webtransport_unsupported");
      }
      this.transport = factory ? factory(url) : new (TransportCtor as WebTransportCtor)(url);
      this.ownsTransport = true;
    } else {
      this.ownsTransport = false;
    }

    const pending = new Promise<WorldWelcomePayload>((resolve, reject) => {
      this.joinTimeout = setTimeout(() => {
        this.failJoin(reject, new WorldRealtimeError("Máy chủ không phản hồi yêu cầu vào world", "join_timeout"));
        this.disconnect();
      }, JOIN_TIMEOUT_MS);
      if (useSharedClient) {
        void this.openSharedSession(this.options.client ?? realtimeClient, token, join, resolve, reject);
      } else if (this.transport) {
        const transport = this.transport;
        this.attachCloseWatcher(transport, reject);
        void this.openSession(transport, null, join, resolve, reject);
      }
    });
    return pending;
  }

  private attachCloseWatcher(transport: WebTransportLike, reject: (error: Error) => void): void {
    transport.closed
      .then((info: { closeCode?: number; reason?: string }) => {
        if (this.transport !== transport) return;
        this.lastCloseCode = info?.closeCode ?? 0;
        if (!this.joined && this.joinTimeout !== null) {
          this.failJoin(reject, new WorldRealtimeError(`WebTransport đóng (${this.lastCloseCode})`, "socket_closed"));
        }
        this.handleClosed();
      })
      .catch(() => {
        if (this.transport === transport) this.handleClosed();
      });
  }

  private async openSharedSession(
    client: RealtimeClient,
    token: string,
    join: WorldJoinPayload,
    resolve: (welcome: WorldWelcomePayload) => void,
    reject: (error: Error) => void,
  ): Promise<void> {
    try {
      const acquired = await client.openWorldStream(token);
      if (this.explicitClose) {
        acquired.release();
        return;
      }
      this.transport = acquired.session;
      this.releaseSharedWorld = acquired.release;
      this.attachCloseWatcher(acquired.session, reject);
      await this.openSession(acquired.session, acquired.stream, join, resolve, reject);
    } catch (error) {
      const raw = error instanceof Error ? error.message : String(error);
      const isUnsupported = error instanceof RealtimeUnsupportedError;
      const code = isUnsupported ? "webtransport_unsupported" : "transport_connect_failed";
      const pendingJoin = this.joinTimeout;
      if (pendingJoin === null) return;
      clearTimeout(pendingJoin); // Reserve the bounded health probe before reporting the failure.
      const message = isUnsupported ? raw : await diagnoseConnectionFailure(error, client.getEndpointURL());
      if (this.joinTimeout !== pendingJoin) return;
      this.failJoin(reject, new WorldRealtimeError(message, code));
      this.disconnect();
    }
  }

  private async openSession(
    transport: WebTransportLike,
    preopenedStream: WebTransportStreamLike | null,
    join: WorldJoinPayload,
    resolve: (welcome: WorldWelcomePayload) => void,
    reject: (error: Error) => void,
  ): Promise<void> {
    try {
      await transport.ready;
      this.datagramWriter = transport.datagrams.writable.getWriter();
      const stream = preopenedStream ?? await transport.createBidirectionalStream();
      this.streamWriter = stream.writable.getWriter();
      const handshake = new Uint8Array(5);
      handshake[0] = STREAM_WORLD;
      handshake.set([0x47, 0x57, 0x54, VERSION], 1);
      const joinFrame = encodeFrame(
        Op.Join,
        encodeJoin({
          zoneId: join.zone_id,
          mapVersion: join.map_version,
          contentVersion: join.content_version,
          requestId: this.nextRequestId++,
        }),
      );
      await this.writeBytes(concat(handshake, joinFrame));
      this.streamReader = stream.readable.getReader();
      this.datagramReader = transport.datagrams.readable.getReader();
      void this.readFrames(this.streamReader, resolve, reject);
      void this.readDatagrams(this.datagramReader);
    } catch (error) {
      const pendingJoin = this.joinTimeout;
      if (pendingJoin === null) return;
      clearTimeout(pendingJoin); // Reserve the bounded health probe before reporting the failure.
      const message = await diagnoseConnectionFailure(error, this.options.url ?? (process.env.NEXT_PUBLIC_REALTIME_URL || `${BACKEND_URL.replace(/^http/, "https")}/api/wt`));
      if (this.joinTimeout !== pendingJoin) return;
      this.failJoin(reject, new WorldRealtimeError(message, "transport_connect_failed"));
      this.disconnect();
    }
  }

  public isJoined(): boolean {
    return this.joined;
  }

  public getWelcome(): WorldWelcomePayload | null {
    return this.welcome;
  }

  public getSnapshot(): BufferedWorldState | null {
    return this.buffer.getState();
  }

  public sendInput(move?: { x: number; y: number }): number | null {
    if (!this.joined || !this.datagramWriter) return null;
    if (move && (!Number.isFinite(move.x) || !Number.isFinite(move.y))) return null;
    const seq = ++this.nextInputSeq;
    const normalized = move ? normalizeVector(move) : { x: 0, y: 0 };
    void this.sendDatagram(
      encodeInput({
        seq,
        moveX: Math.max(-127, Math.min(127, Math.round(normalized.x * 127))),
        moveY: Math.max(-127, Math.min(127, Math.round(normalized.y * 127))),
      }),
    );
    return seq;
  }

  public sendMoveTarget(target: { x: number; y: number }): number | null {
    if (!this.joined) return null;
    if (!Number.isFinite(target.x) || !Number.isFinite(target.y)) return null;
    const seq = ++this.nextInputSeq;
    void this.sendFrame(encodeMoveTarget(seq, target.x, target.y));
    return seq;
  }

  public sendCast(cast: Omit<CastPayload, "seq">): number | null {
    if (!this.joined) return null;
    const seq = ++this.nextInputSeq;
    const targetNetId = cast.target_id ? (this.netByEntity.get(cast.target_id) ?? 0) : 0;
    void this.sendFrame(
      encodeCast({
        seq,
        slot: cast.slot,
        mode: castModeCode(cast.mode),
        targetNetId,
        aimX: cast.aim_x,
        aimY: cast.aim_y,
        x: cast.x,
        y: cast.y,
      }),
    );
    return seq;
  }

  public sendBasicAttack(attack: Omit<BasicAttackPayload, "seq">): number | null {
    if (!this.joined) return null;
    const seq = ++this.nextInputSeq;
    const targetNetId = attack.target_id ? (this.netByEntity.get(attack.target_id) ?? 0) : 0;
    void this.sendFrame(encodeAttack({ seq, targetNetId, aimX: attack.aim_x, aimY: attack.aim_y }));
    return seq;
  }

  public sendCapture(targetId: string, guId?: string): number | null {
    if (!this.joined) return null;
    const targetNetId = this.netByEntity.get(targetId) ?? 0;
    if (targetNetId === 0) return null;
    const seq = ++this.nextInputSeq;
    void this.sendFrame(encodeCapture({ seq, targetNetId, guId }));
    return seq;
  }

  public sendCompanionStance(stance: "DEFEND" | "FOCUS" | "FOLLOW", focusTargetId?: string): number | null {
    if (!this.joined) return null;
    const seq = ++this.nextInputSeq;
    const focusNetId = focusTargetId ? (this.netByEntity.get(focusTargetId) ?? 0) : 0;
    void this.sendFrame(encodeStance({ seq, stance: stanceCode(stance), focusNetId }));
    return seq;
  }

  public sendPing(clientMs = Date.now()): void {
    if (!this.joined || !this.datagramWriter) return;
    void this.sendDatagram(encodePingDatagram(clientMs));
  }

  public getLastRttMs(): number {
    return this.lastRttMs;
  }

  public onEvents(callback: (event: WorldEvent) => void): () => void {
    this.eventListeners.add(callback);
    return () => this.eventListeners.delete(callback);
  }

  public onSnapshot(callback: (state: BufferedWorldState) => void): () => void {
    this.snapshotListeners.add(callback);
    return () => this.snapshotListeners.delete(callback);
  }

  public onReject(callback: (reject: WorldRejectPayload) => void): () => void {
    this.rejectListeners.add(callback);
    return () => this.rejectListeners.delete(callback);
  }

  public onTransfer(callback: (transfer: WorldTransferPayload) => void): () => void {
    this.transferListeners.add(callback);
    return () => this.transferListeners.delete(callback);
  }

  public onResyncRequired(callback: (reason: string) => void): () => void {
    this.resyncListeners.add(callback);
    return () => this.resyncListeners.delete(callback);
  }

  public onError(callback: (error: Error) => void): () => void {
    this.errorListeners.add(callback);
    return () => this.errorListeners.delete(callback);
  }

  public disconnect(): void {
    this.explicitClose = true;
    this.joined = false;
    this.welcome = null;
    this.clearJoinTimeout();
    const writer = this.streamWriter;
    this.streamWriter = null;
    try {
      if (writer) {
        void writer.write(encodeFrame(Op.Leave, new Uint8Array(0)))
          .then(() => writer.close())
          .catch(() => {});
      }
    } catch {
      // best effort
    }
    this.releaseStreamLocks();
    if (this.releaseSharedWorld) {
      const release = this.releaseSharedWorld;
      this.releaseSharedWorld = null;
      release();
    }
    if (this.ownsTransport) {
      try {
        this.transport?.close({ closeCode: 0, reason: "client disconnected" });
      } catch {
        // best effort
      }
    }
    this.transport = null;
    this.ownsTransport = false;
    this.buffer.reset();
    this.netToSpawn.clear();
    this.entityByNet.clear();
    this.netByEntity.clear();
  }

  private releaseStreamLocks(): void {
    const dgWriter = this.datagramWriter;
    this.datagramWriter = null;
    try {
      dgWriter?.releaseLock();
    } catch {
      // ignore
    }
    const dgReader = this.datagramReader;
    this.datagramReader = null;
    try {
      dgReader?.releaseLock();
    } catch {
      // ignore
    }
    const stReader = this.streamReader;
    this.streamReader = null;
    try {
      void stReader?.cancel().catch(() => {});
    } catch {
      // ignore
    }
  }

  public getMetrics(): {
    messages: number;
    bytes: number;
    messagesSent: number;
    bytesSent: number;
    snapshots: number;
    lastMessageAt: number;
    lastSnapshotAt: number;
    lastCloseCode: number;
  } {
    return {
      messages: this.messageCount,
      bytes: this.byteCount,
      messagesSent: this.messagesSent,
      bytesSent: this.bytesSent,
      snapshots: this.snapshotCount,
      lastMessageAt: this.lastMessageAt,
      lastSnapshotAt: this.lastSnapshotAt,
      lastCloseCode: this.lastCloseCode,
    };
  }

  // ---- internals ---------------------------------------------------------

  private async readFrames(
    reader: ReadableStreamDefaultReader<Uint8Array>,
    resolve: (welcome: WorldWelcomePayload) => void,
    reject: (error: Error) => void,
  ): Promise<void> {
    try {
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        if (!value) continue;
        this.messageCount += 1;
        this.byteCount += value.byteLength;
        this.lastMessageAt = Date.now();
        this.appendIncoming(value);
        for (;;) {
          const parsed = this.tryParseFrame();
          if (!parsed) break;
          this.handleFrame(parsed.op, parsed.payload, resolve, reject);
        }
        this.compactIncoming();
      }
    } catch (error) {
      this.notifyError(asWorldError(error, "transport_read_failed"));
    }
    if (this.joined) this.handleClosed();
  }

  private appendIncoming(chunk: Uint8Array): void {
    const unread = this.incoming.subarray(this.incomingOffset);
    if (unread.length === 0) {
      this.incoming = chunk;
      this.incomingOffset = 0;
      return;
    }
    this.incoming = concat(unread, chunk);
    this.incomingOffset = 0;
  }

  private compactIncoming(): void {
    if (this.incomingOffset === 0) return;
    if (this.incomingOffset >= this.incoming.length) {
      this.incoming = new Uint8Array(0);
      this.incomingOffset = 0;
    } else if (this.incomingOffset >= 4096) {
      this.incoming = this.incoming.slice(this.incomingOffset);
      this.incomingOffset = 0;
    }
  }

  private tryParseFrame(): { op: number; payload: Uint8Array } | null {
    const available = this.incoming.length - this.incomingOffset;
    if (available < 5) return null;
    const view = new DataView(this.incoming.buffer, this.incoming.byteOffset + this.incomingOffset, available);
    const length = view.getUint32(0, true);
    if (length < 1 || available < 4 + length) return null;
    const base = this.incomingOffset;
    const op = this.incoming[base + 4];
    const payload = this.incoming.subarray(base + 5, base + 4 + length);
    this.incomingOffset = base + 4 + length;
    return { op, payload };
  }

  private handleFrame(
    op: number,
    payload: Uint8Array,
    resolve: (welcome: WorldWelcomePayload) => void,
    reject: (error: Error) => void,
  ): void {
    switch (op) {
      case Op.Welcome: {
        try {
          const welcome = decodeWelcome(payload);
          const full = decodeSnapshotFull(welcome.snapshot);
          this.applyBaseline(full);
          const snapshot = this.toSnapshotPayload(full, true);
          const applied = this.buffer.apply(snapshot);
          if (applied.status !== "applied") throw new Error("Welcome baseline could not be applied");
          const skills = welcome.skillsJson.length > 0
            ? JSON.parse(sharedTextDecoder.decode(welcome.skillsJson))
            : [];
          this.welcome = {
            session_id: welcome.sessionId,
            zone_id: welcome.zoneId,
            map_version: welcome.mapVersion,
            content_version: welcome.contentVersion,
            tick_rate: welcome.tickRate,
            movement_speed: welcome.movementSpeedX100 / 100,
            server_tick: welcome.serverTick,
            entity_id: welcome.entityId,
            entity_revision: welcome.entityRevision,
            baseline_id: welcome.baselineId,
            map_hash: welcome.mapHash || undefined,
            skills,
            loadout_revision: welcome.loadoutRevision || undefined,
            snapshot,
          };
          this.joined = true;
          this.clearJoinTimeout();
          this.snapshotCount += 1;
          this.lastSnapshotAt = Date.now();
          this.emitSnapshot(applied.state);
          resolve(this.welcome);
        } catch (error) {
          this.failJoin(reject, asWorldError(error, "invalid_welcome"));
        }
        break;
      }
      case Op.Snapshot: {
        try {
          const full = decodeSnapshotFull(payload);
          if (full.isBaseline) this.applyBaseline(full);
          const applied = this.buffer.apply(this.toSnapshotPayload(full, full.isBaseline));
          if (applied.status === "resync_required") {
            if (!full.isBaseline && this.requestResync(applied.reason)) {
              break;
            }
            this.resyncListeners.forEach((listener) => listener(applied.reason));
            break;
          }
          if (applied.status !== "applied") break;
          this.snapshotCount += 1;
          this.lastSnapshotAt = Date.now();
          this.emitSnapshot(applied.state);
        } catch (error) {
          this.notifyError(asWorldError(error, "invalid_snapshot"));
        }
        break;
      }
      case Op.Spawn:
        try {
          this.applySpawnPayload(payload);
        } catch (error) {
          this.notifyError(asWorldError(error, "invalid_spawn"));
        }
        break;
      case Op.Events: {
        try {
          const { events } = decodeEvents(payload);
          for (const event of events) {
            const source = this.netToSpawn.get(event.source)?.entityId;
            const target = this.netToSpawn.get(event.target)?.entityId;
            let body: Record<string, unknown> | undefined;
            if (event.payload.length > 0) {
              try {
                body = JSON.parse(sharedTextDecoder.decode(event.payload)) as Record<string, unknown>;
              } catch {
                body = undefined;
              }
            }
            const worldEvent: WorldEvent = {
              event_id: event.eventId,
              tick: event.tick,
              type: EVENT_TYPE_NAMES[event.typeId] ?? "UNKNOWN",
              source_id: source,
              target_id: target,
              payload: body,
            };
            this.eventListeners.forEach((listener) => listener(worldEvent));
          }
        } catch (error) {
          this.notifyError(asWorldError(error, "invalid_events"));
        }
        break;
      }
      case Op.Transfer: {
        try {
          const transfer = decodeTransfer(payload);
          this.netToSpawn.clear();
          this.entityByNet.clear();
          this.netByEntity.clear();
          const transferPayload: WorldTransferPayload = {
            from_zone_id: transfer.fromZoneId || undefined,
            to_zone_id: transfer.toZoneId,
            transfer_id: transfer.transferId || undefined,
            target_x: dequantize(transfer.targetX),
            target_y: dequantize(transfer.targetY),
            legacy_x: transfer.legacyX,
            legacy_y: transfer.legacyY,
            hp: transfer.hp,
          };
          this.transferListeners.forEach((listener) => listener(transferPayload));
        } catch (error) {
          this.notifyError(asWorldError(error, "invalid_transfer"));
        }
        break;
      }
      case Op.Reject: {
        const rejectPayload = decodeReject(payload);
        this.rejectListeners.forEach((listener) => listener(rejectPayload));
        if (!this.joined) {
          this.failJoin(reject, new WorldRealtimeError(rejectPayload.message || rejectPayload.code, rejectPayload.code));
        }
        break;
      }
      default:
        break;
    }
  }

  private async readDatagrams(reader: ReadableStreamDefaultReader<Uint8Array>): Promise<void> {
    try {
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        if (!value) continue;
        this.messageCount += 1;
        this.byteCount += value.byteLength;
        this.lastMessageAt = Date.now();
        const { op, payload } = decodeDatagram(value);
        if (op === DatagramOp.SnapshotDelta) {
          this.applyDelta(decodeSnapshotDelta(payload));
        } else if (op === DatagramOp.Pong) {
          try {
            const { clientMs } = decodePongDatagram(payload);
            const rtt = Date.now() - clientMs;
            if (rtt >= 0 && Number.isFinite(rtt)) {
              this.lastRttMs = rtt;
            }
          } catch {
            // ignore malformed pong
          }
        }
      }
    } catch (error) {
      this.notifyError(asWorldError(error, "transport_datagram_failed"));
    }
  }

  private applyBaseline(full: {
    spawns: Spawn[];
    states: DeltaEntity[];
  }): void {
    this.netToSpawn.clear();
    this.entityByNet.clear();
    this.netByEntity.clear();
    for (const spawn of full.spawns) {
      this.netToSpawn.set(spawn.netId, spawn);
      this.netByEntity.set(spawn.entityId, spawn.netId);
    }
    const states = new Map<number, DeltaEntity>();
    for (const state of full.states) states.set(state.netId, state);
    for (const spawn of full.spawns) {
      const state = states.get(spawn.netId);
      if (state) this.entityByNet.set(spawn.netId, entityFrom(spawn, state, this.netToSpawn));
    }
  }

  private applySpawnPayload(payload: Uint8Array): void {
    const reader = new Reader(payload);
    const count = reader.u8();
    for (let i = 0; i < count; i += 1) {
      const spawn = readSpawn(reader);
      this.netToSpawn.set(spawn.netId, spawn);
      this.netByEntity.set(spawn.entityId, spawn.netId);
    }
  }

  private applyDelta(delta: {
    tick: number;
    baselineId: number;
    ackSeq: number;
    entities: DeltaEntity[];
    removed: number[];
    projectiles: Projectile[];
  }): void {
    for (const state of delta.entities) {
      const spawn = this.netToSpawn.get(state.netId);
      if (!spawn) continue;
      this.entityByNet.set(state.netId, mergeEntity(spawn, this.entityByNet.get(state.netId), state, this.netToSpawn));
    }
    const removedIds: string[] = [];
    const seenRemoved = new Set<string>();
    for (const netId of delta.removed) {
      const spawn = this.netToSpawn.get(netId);
      if (spawn && !seenRemoved.has(spawn.entityId)) {
        seenRemoved.add(spawn.entityId);
        removedIds.push(spawn.entityId);
      }
      this.netToSpawn.delete(netId);
      this.entityByNet.delete(netId);
      if (spawn) this.netByEntity.delete(spawn.entityId);
    }
    const snapshot: WorldSnapshotPayload = {
      tick: delta.tick,
      ack_seq: delta.ackSeq,
      baseline_id: delta.baselineId,
      base_tick: this.buffer.getState()?.baseTick ?? delta.tick,
      is_baseline: false,
      entities: delta.entities
        .map((state) => this.entityByNet.get(state.netId))
        .filter((entity): entity is WorldEntityState => Boolean(entity)),
      removed: removedIds,
      projectiles: this.mapProjectiles(delta.projectiles),
    };
    const applied = this.buffer.apply(snapshot);
    if (applied.status === "resync_required") {
      if (this.requestResync(applied.reason)) {
        return;
      }
      this.resyncListeners.forEach((listener) => listener(applied.reason));
      return;
    }
    if (applied.status !== "applied") return;
    this.snapshotCount += 1;
    this.lastSnapshotAt = Date.now();
    this.emitSnapshot(applied.state);
  }

  private requestResync(reason: string): boolean {
    if (!this.joined || !this.streamWriter) return false;
    const now = Date.now();
    if (now - this.lastResyncSentAt < RESYNC_COOLDOWN_MS) return true;
    this.lastResyncSentAt = now;
    const writer = new Writer(1 + reason.length);
    const reasonBytes = new TextEncoder().encode(reason.slice(0, 120));
    writer.u8(reasonBytes.length).raw(reasonBytes);
    void this.sendFrame(encodeFrame(Op.Resync, writer.bytes()));
    return true;
  }

  private toSnapshotPayload(
    full: {
      tick: number;
      baselineId: number;
      baseTick: number;
      ackSeq: number;
      spawns: Spawn[];
      states: DeltaEntity[];
      removed?: number[];
      projectiles: Projectile[];
    },
    isBaseline: boolean,
  ): WorldSnapshotPayload {
    for (const spawn of full.spawns) {
      this.netToSpawn.set(spawn.netId, spawn);
      this.netByEntity.set(spawn.entityId, spawn.netId);
    }
    const entities: WorldEntityState[] = [];
    if (isBaseline) {
      const states = new Map<number, DeltaEntity>();
      for (const state of full.states) states.set(state.netId, state);
      for (const spawn of full.spawns) {
        const state = states.get(spawn.netId);
        if (state) entities.push(entityFrom(spawn, state, this.netToSpawn));
      }
    } else {
      for (const state of full.states) {
        const spawn = this.netToSpawn.get(state.netId);
        if (!spawn) continue;
        const merged = mergeEntity(spawn, this.entityByNet.get(state.netId), state, this.netToSpawn);
        this.entityByNet.set(state.netId, merged);
        entities.push(merged);
      }
    }
    const removedIds: string[] = [];
    const seenRemoved = new Set<string>();
    for (const netId of full.removed ?? []) {
      const spawn = this.netToSpawn.get(netId);
      if (spawn && !seenRemoved.has(spawn.entityId)) {
        seenRemoved.add(spawn.entityId);
        removedIds.push(spawn.entityId);
      }
      this.netToSpawn.delete(netId);
      this.entityByNet.delete(netId);
      if (spawn) this.netByEntity.delete(spawn.entityId);
    }
    return {
      tick: full.tick,
      ack_seq: full.ackSeq,
      baseline_id: full.baselineId,
      base_tick: isBaseline ? full.tick : (full.baseTick || this.buffer.getState()?.baseTick || full.tick),
      is_baseline: isBaseline,
      entities,
      removed: removedIds.length > 0 ? removedIds : undefined,
      projectiles: this.mapProjectiles(full.projectiles),
    };
  }

  private mapProjectiles(projectiles: Projectile[]): WorldProjectileState[] {
    return projectiles.map((p) => ({
      id: p.id,
      source_id: this.netToSpawn.get(p.sourceNetId)?.entityId ?? "",
      skill_id: p.skillId,
      projectile_vfx: p.vfx || undefined,
      x: dequantize(p.x),
      y: dequantize(p.y),
      radius: p.radiusX10 / 10,
    }));
  }

  private emitSnapshot(state: BufferedWorldState): void {
    this.snapshotListeners.forEach((listener) => listener(state));
  }

  private async sendFrame(frame: Uint8Array): Promise<void> {
    try {
      await this.writeBytes(frame);
    } catch (error) {
      this.notifyError(asWorldError(error, "send_failed"));
    }
  }

  private async writeBytes(bytes: Uint8Array): Promise<void> {
    if (!this.streamWriter) return;
    this.messagesSent += 1;
    this.bytesSent += bytes.byteLength;
    await this.streamWriter.write(bytes);
  }

  private async sendDatagram(datagram: Uint8Array): Promise<void> {
    if (!this.datagramWriter || datagram.byteLength > MAX_DATAGRAM_SIZE) return;
    this.messagesSent += 1;
    this.bytesSent += datagram.byteLength;
    try {
      await this.datagramWriter.write(datagram);
    } catch (error) {
      this.notifyError(asWorldError(error, "send_datagram_failed"));
    }
  }

  private handleClosed(): void {
    if (this.explicitClose) return;
    this.joined = false;
    this.welcome = null;
    this.streamWriter = null;
    this.releaseStreamLocks();
    if (this.releaseSharedWorld) {
      const release = this.releaseSharedWorld;
      this.releaseSharedWorld = null;
      release();
    }
    this.notifyError(new WorldRealtimeError("Kết nối realtime đã đóng", "socket_closed"));
  }

  private failJoin(reject: (error: Error) => void, error: Error): void {
    if (this.joinTimeout === null) return;
    this.clearJoinTimeout();
    reject(error);
    this.notifyError(error);
  }

  private clearJoinTimeout(): void {
    if (this.joinTimeout !== null) {
      clearTimeout(this.joinTimeout);
      this.joinTimeout = null;
    }
  }

  private notifyError(error: Error): void {
    this.errorListeners.forEach((listener) => listener(error));
  }
}

function defaultTransportFactory(): WebTransportCtor | null {
  const ctor = (globalThis as unknown as { WebTransport?: WebTransportCtor }).WebTransport;
  return typeof ctor === "function" ? ctor : null;
}

function readStorage(storageName: "localStorage" | "sessionStorage", key: string): string | null {
  if (typeof window === "undefined") return null;
  try {
    return window[storageName].getItem(key);
  } catch {
    return null;
  }
}

function normalizeVector(vector: { x: number; y: number }): { x: number; y: number } {
  const magnitude = Math.hypot(vector.x, vector.y);
  if (magnitude === 0) return { x: 0, y: 0 };
  const scale = magnitude > 1 ? 1 / magnitude : 1;
  return { x: vector.x * scale, y: vector.y * scale };
}

function concat(left: Uint8Array, right: Uint8Array): Uint8Array {
  const out = new Uint8Array(left.length + right.length);
  out.set(left);
  out.set(right, left.length);
  return out;
}

function castModeCode(mode: string): number {
  switch (mode) {
    case "DIRECTION":
      return 1;
    case "GROUND":
      return 2;
    case "SELF":
      return 3;
    default:
      return 0;
  }
}

function stanceCode(stance: "DEFEND" | "FOCUS" | "FOLLOW"): number {
  switch (stance) {
    case "FOCUS":
      return 1;
    case "FOLLOW":
      return 2;
    default:
      return 0;
  }
}

function kindName(code: number): "player" | "monster" | "companion" {
  if (code === 1) return "monster";
  if (code === 2) return "companion";
  return "player";
}

function stanceName(code: number): "DEFEND" | "FOCUS" | "FOLLOW" {
  if (code === 1) return "FOCUS";
  if (code === 2) return "FOLLOW";
  return "DEFEND";
}

function aiStateName(code: number): string | undefined {
  switch (code) {
    case 1:
      return "idle";
    case 2:
      return "patrol";
    case 3:
      return "aggro";
    case 4:
      return "chase";
    case 5:
      return "attack";
    case 6:
      return "leash";
    case 7:
      return "dead";
    case 8:
      return "respawn";
    default:
      return undefined;
  }
}

function entityFrom(spawn: Spawn, state: DeltaEntity, netToSpawn: Map<number, Spawn>): WorldEntityState {
  const ownerId = spawn.ownerNetId ? netToSpawn.get(spawn.ownerNetId)?.entityId : undefined;
  const aiState = state.hasExt && (state.ext & 1) ? aiStateName(state.aiState) : undefined;
  const aggroTargetId = state.hasExt && (state.ext & 2) && state.aggroNetId
    ? netToSpawn.get(state.aggroNetId)?.entityId
    : undefined;
  const stance = state.hasExt && (state.ext & 16)
    ? stanceName(state.stance)
    : (spawn.companionKind ? stanceName(spawn.stance) : undefined);
  return {
    entity_id: spawn.entityId,
    entity_revision: state.revision,
    kind: kindName(spawn.kind),
    party_id: spawn.partyId || undefined,
    pvp_enabled: (spawn.flags & 1) !== 0,
    template_id: spawn.templateId || undefined,
    name: spawn.name || undefined,
    ai_state: aiState,
    aggro_target_id: aggroTargetId,
    x: dequantize(state.x),
    y: dequantize(state.y),
    hp: state.hp,
    max_hp: spawn.maxHp || undefined,
    shield: state.shield || undefined,
    mana: state.mana,
    max_mana: spawn.maxMana || undefined,
    thought: state.thought,
    max_thought: spawn.maxThought || undefined,
    cast_end_tick: (state.mask & 16) ? state.castEndTick : undefined,
    cast_windup_ticks: (state.mask & 16) ? state.castWindup : undefined,
    stealthed: (spawn.flags & 2) !== 0 || undefined,
    flying: (spawn.flags & 4) !== 0 || undefined,
    form_active: (spawn.flags & 8) !== 0 || undefined,
    owner_id: ownerId,
    companion_kind: spawn.companionKind || undefined,
    stance,
  };
}

function mergeEntity(
  spawn: Spawn,
  previous: WorldEntityState | undefined,
  state: DeltaEntity,
  netToSpawn: Map<number, Spawn>,
): WorldEntityState {
  const next: WorldEntityState = { ...(previous ?? entityFrom(spawn, state, netToSpawn)) };
  if (state.mask & 4) {
    next.entity_revision = state.revision;
  }
  if (state.mask & 1) {
    next.x = dequantize(state.x);
    next.y = dequantize(state.y);
  }
  if (state.mask & 2) {
    if (state.vitalsSub & 1) next.hp = state.hp;
    if (state.vitalsSub & 2) next.shield = state.shield;
    if (state.vitalsSub & 4) next.mana = state.mana;
    if (state.vitalsSub & 8) next.thought = state.thought;
  }
  if (state.mask & 16) {
    next.cast_end_tick = state.castEndTick || undefined;
    next.cast_windup_ticks = state.castWindup || undefined;
  }
  if (state.mask & 32) {
    next.pvp_enabled = (state.flags & 1) !== 0;
    next.stealthed = (state.flags & 2) !== 0 || undefined;
    next.flying = (state.flags & 4) !== 0 || undefined;
    next.form_active = (state.flags & 8) !== 0 || undefined;
  }
  if (state.hasExt) {
    if (state.ext & 1) {
      next.ai_state = aiStateName(state.aiState);
    }
    if (state.ext & 2) {
      next.aggro_target_id = state.aggroNetId ? netToSpawn.get(state.aggroNetId)?.entityId : undefined;
    }
    if (state.ext & 16) {
      next.stance = stanceName(state.stance);
    }
  }
  return next;
}

function decodeReject(payload: Uint8Array): WorldRejectPayload {
  const reader = new Reader(payload);
  const code = reader.string();
  const message = reader.string();
  const retryable = reader.u8() === 1;
  return { code: code || "realtime_world_rejected", message: message || undefined, retryable };
}

function formatRealtimeErrorMessage(rawMessage: string): string {
  const lower = rawMessage.toLowerCase();
  if (lower.includes("opening handshake failed") || lower.includes("failed to establish a quic connection")) {
    return [
      "Chưa kết nối được WebTransport tới backend: bước bắt tay HTTP/3 thất bại.",
      "Chưa xác định được nguyên nhân; lỗi này không đủ để kết luận backend đã dừng.",
      "Mở /api/health trên địa chỉ backend đang cấu hình để kiểm tra:",
      "• Có cảnh báo chứng chỉ: kiểm tra chứng chỉ TLS dev và độ tin cậy của chứng chỉ.",
      "• Không truy cập được health: kiểm tra backend, địa chỉ và cổng HTTPS. Trình duyệt cũng có thể chặn do TLS hoặc CORS.",
      "• Health trả thành công: HTTPS hoạt động, nhưng chưa chứng minh HTTP/3 hoạt động. Kiểm tra endpoint WebTransport, origin frontend trong CORS_ALLOWED_ORIGINS và cổng UDP của endpoint.",
      "Xem Console/Network và log backend để xác định lỗi TLS, từ chối Origin hoặc HTTP/3.",
      `Chi tiết trình duyệt: ${rawMessage}`,
    ].join("\n");
  }
  return rawMessage;
}

function asWorldError(error: unknown, code: string): WorldRealtimeError {
  const raw = error instanceof Error ? error.message : String(error);
  return new WorldRealtimeError(formatRealtimeErrorMessage(raw), code);
}
