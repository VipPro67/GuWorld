import { BACKEND_URL } from "../../lib/config.ts";
import { SnapshotBuffer, type BufferedWorldState, type SnapshotApplyResult } from "./SnapshotBuffer.ts";
import {
  WORLD_PROTOCOL_VERSION,
  decodeWorldEnvelope,
  isWorldProtocolMessage,
  parseWorldEventBatch,
  parseWorldSnapshotPayload,
  parseWorldTransferPayload,
  parseWorldWelcomePayload,
  type CastPayload,
  type BasicAttackPayload,
  type MoveTargetPayload,
  type WorldEnvelope,
  type WorldEvent,
  type WorldInputPayload,
  type WorldJoinPayload,
  type WorldRejectPayload,
  type WorldSnapshotPayload,
  type WorldTransferPayload,
  type WorldWelcomePayload,
} from "./Protocol.ts";

interface SocketLike {
  readyState: number;
  onopen: ((event: Event) => void) | null;
  onmessage: ((event: MessageEvent<unknown>) => void) | null;
  onerror: ((event: Event) => void) | null;
  onclose: ((event: CloseEvent) => void) | null;
  send(data: string): void;
  close(code?: number, reason?: string): void;
}

export interface WorldSocketOptions {
  url?: string;
  getToken?: () => string | null;
  getTabId?: () => string | null;
  createSocket?: (url: string) => SocketLike;
}

const WORLD_JOIN_TIMEOUT_MS = 10_000;

export class WorldSocketError extends Error {
  public readonly code: string;

  constructor(message: string, code = "world_socket_error") {
    super(message);
    this.name = "WorldSocketError";
    this.code = code;
  }
}

/** Authenticated protocol-v1 transport for the Phaser world runtime. */
export class WorldSocket {
  private readonly options: WorldSocketOptions;
  private readonly buffer = new SnapshotBuffer();
  private socket: SocketLike | null = null;
  private connecting: Promise<WorldWelcomePayload> | null = null;
  private pendingJoinReject: ((error: Error) => void) | null = null;
  private joinTimeout: ReturnType<typeof setTimeout> | null = null;
  private joinRequestId = "";
  private welcome: WorldWelcomePayload | null = null;
  private nextInputSeq = 0;
  private explicitClose = false;
  private snapshotListeners = new Set<(state: BufferedWorldState) => void>();
  private eventListeners = new Set<(event: WorldEvent) => void>();
  private rejectListeners = new Set<(reject: WorldRejectPayload) => void>();
  private transferListeners = new Set<(transfer: WorldTransferPayload) => void>();
  private resyncListeners = new Set<(reason: string) => void>();
  private errorListeners = new Set<(error: Error) => void>();
  private messageCount = 0;
  private byteCount = 0;
  private snapshotCount = 0;
  private lastMessageAt = 0;
  private lastSnapshotAt = 0;

  constructor(options: WorldSocketOptions = {}) {
    this.options = options;
  }

  public connect(join: WorldJoinPayload): Promise<WorldWelcomePayload> {
    if (this.welcome && this.socket?.readyState === socketOpenState) return Promise.resolve(this.welcome);
    if (this.connecting) return this.connecting;
    if (!isValidJoin(join)) return Promise.reject(new WorldSocketError("Thông tin map/version không hợp lệ", "invalid_join"));

    const token = this.options.getToken?.() ?? readStorage("localStorage", "gu_auth_token");
    if (!token) return Promise.reject(new WorldSocketError("Chưa có token đăng nhập", "missing_auth_token"));
    const createSocket = this.options.createSocket ?? defaultSocketFactory;
    const requestId = createRequestId();
    this.joinRequestId = requestId;
    this.explicitClose = false;
    this.buffer.reset();
    this.nextInputSeq = 0;

    let resolveJoin!: (welcome: WorldWelcomePayload) => void;
    let rejectJoin!: (error: Error) => void;
    const pending = new Promise<WorldWelcomePayload>((resolve, reject) => {
      resolveJoin = resolve;
      rejectJoin = reject;
    });
    this.connecting = pending;
    this.pendingJoinReject = rejectJoin;
    this.joinTimeout = setTimeout(() => {
      this.failJoin(
        rejectJoin,
        new WorldSocketError("Máy chủ không phản hồi yêu cầu vào world", "join_timeout"),
        "join_timeout",
      );
      this.socket?.close(4003, "world join timeout");
    }, WORLD_JOIN_TIMEOUT_MS);

    try {
      const socket = createSocket(buildSocketURL(this.options.url, token, this.options.getTabId?.() ?? readStorage("sessionStorage", "gu_gameplay_tab_id")));
      this.socket = socket;
      socket.onopen = () => {
        const envelope: WorldEnvelope<WorldJoinPayload> = {
          v: WORLD_PROTOCOL_VERSION,
          type: "world.join",
          request_id: requestId,
          payload: join,
        };
        try {
          socket.send(JSON.stringify(envelope));
        } catch (error) {
          this.failJoin(rejectJoin, error, "join_send_failed");
        }
      };
      socket.onmessage = (event) => {
        this.handleRawMessage(event.data, resolveJoin, rejectJoin);
      };
      socket.onerror = () => {
        this.notifyError(new WorldSocketError("Kết nối realtime gặp lỗi", "socket_error"));
      };
      socket.onclose = () => {
        if (this.connecting) {
          this.failJoin(rejectJoin, new Error("WebSocket closed before world.join completed"), "socket_closed");
        }
        this.socket = null;
        this.welcome = null;
        this.buffer.reset();
      };
    } catch (error) {
      this.failJoin(rejectJoin, error, "socket_create_failed");
    }

    return pending;
  }

  public isJoined(): boolean {
    return this.welcome !== null && this.socket?.readyState === socketOpenState;
  }

  public getWelcome(): WorldWelcomePayload | null {
    return this.welcome;
  }

  public getSnapshot(): BufferedWorldState | null {
    return this.buffer.getState();
  }

  public sendInput(move?: { x: number; y: number }): number | null {
    if (!this.isJoined() || !this.socket) return null;
    if (move && (!Number.isFinite(move.x) || !Number.isFinite(move.y))) return null;
    const seq = this.nextSequence();
    if (seq === null || !this.socket) return null;
    const vector = move ? normalizeVector(move) : undefined;
    const payload: WorldInputPayload = { seq, move: vector };
    try {
      this.socket.send(JSON.stringify({ v: WORLD_PROTOCOL_VERSION, type: "world.input", payload }));
      return payload.seq;
    } catch (error) {
      this.notifyError(asError(error, "input_send_failed"));
      return null;
    }
  }

  /**
   * Requests server-bounded click-to-move. It shares the monotonic movement
   * sequence with WASD input, so the latest command wins on the zone tick.
   */
  public sendMoveTarget(target: { x: number; y: number }): number | null {
    if (!this.isJoined() || !this.socket) return null;
    if (!Number.isFinite(target.x) || !Number.isFinite(target.y)) return null;
    if (Math.abs(target.x) > 1_000_000 || Math.abs(target.y) > 1_000_000) return null;
    const seq = this.nextSequence();
    if (seq === null || !this.socket) return null;
    const payload: MoveTargetPayload = { seq, x: target.x, y: target.y };
    try {
      this.socket.send(JSON.stringify({ v: WORLD_PROTOCOL_VERSION, type: "world.move_target", payload }));
      return payload.seq;
    } catch (error) {
      this.notifyError(asError(error, "move_target_send_failed"));
      return null;
    }
  }

  private nextSequence(): number | null {
    if (this.nextInputSeq >= Number.MAX_SAFE_INTEGER) {
      this.notifyError(new WorldSocketError("Input sequence đã vượt giới hạn", "input_sequence_exhausted"));
      return null;
    }
    return ++this.nextInputSeq;
  }

  /** Sends a server-validated cast intent. Damage is never calculated client-side. */
  public sendCast(cast: Omit<CastPayload, "seq">): number | null {
    if (!this.isJoined() || !this.socket) return null;
    if (cast.slot < 1 || cast.slot > 6 || !Number.isFinite(cast.slot)) return null;
    for (const value of [cast.aim_x, cast.aim_y, cast.x, cast.y]) {
      if (value !== undefined && (!Number.isFinite(value) || Math.abs(value) > 1_000_000)) return null;
    }
    const seq = this.nextSequence();
    if (seq === null || !this.socket) return null;
    const payload: CastPayload = { ...cast, seq };
    try {
      this.socket.send(JSON.stringify({ v: WORLD_PROTOCOL_VERSION, type: "world.cast", payload }));
      return seq;
    } catch (error) {
      this.notifyError(asError(error, "cast_send_failed"));
      return null;
    }
  }

  /** Sends a target-only melee intent; server owns damage and attack cadence. */
  public sendBasicAttack(targetId: string): number | null {
	if (!this.isJoined() || !this.socket || !targetId.trim() || targetId.length > 80) return null;
	const seq = this.nextSequence();
	if (seq === null || !this.socket) return null;
	const payload: BasicAttackPayload = { seq, target_id: targetId };
	try {
		this.socket.send(JSON.stringify({ v: WORLD_PROTOCOL_VERSION, type: "world.attack", payload }));
		return seq;
	} catch (error) {
		this.notifyError(asError(error, "basic_attack_send_failed"));
		return null;
	}
  }

  public onEvents(callback: (event: WorldEvent) => void): () => void {
    this.eventListeners.add(callback);
    return () => this.eventListeners.delete(callback);
  }

  public getMetrics(): { messages: number; bytes: number; snapshots: number; lastMessageAt: number; lastSnapshotAt: number } {
    return {
      messages: this.messageCount,
      bytes: this.byteCount,
      snapshots: this.snapshotCount,
      lastMessageAt: this.lastMessageAt,
      lastSnapshotAt: this.lastSnapshotAt,
    };
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
    const socket = this.socket;
    if (this.pendingJoinReject) {
      this.failJoin(this.pendingJoinReject, new Error("World connection was cancelled"), "join_cancelled");
    }
    this.socket = null;
    this.connecting = null;
    this.welcome = null;
    this.joinRequestId = "";
    this.clearJoinTimeout();
    this.buffer.reset();
    if (socket && socket.readyState < socketClosingState) socket.close(1000, "world client disconnected");
  }

  private handleRawMessage(data: unknown, resolveJoin: (welcome: WorldWelcomePayload) => void, rejectJoin: (error: Error) => void): void {
    if (typeof data !== "string") {
      this.notifyError(new WorldSocketError("Realtime message must be text", "invalid_message_type"));
      return;
    }
    this.messageCount += 1;
    this.byteCount += data.length;
    this.lastMessageAt = Date.now();
    for (const line of data.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      let parsed: unknown;
      try {
        parsed = JSON.parse(trimmed);
      } catch (error) {
        this.notifyError(asError(error, "invalid_envelope"));
        continue;
      }
      // Legacy chat/trade/presence frames share this socket; they are not
      // world-protocol envelopes and must be ignored, not reported as
      // "Unsupported realtime world protocol version".
      if (!isWorldProtocolMessage(parsed)) continue;
      let envelope: WorldEnvelope;
      try {
        envelope = decodeWorldEnvelope(trimmed);
      } catch (error) {
        this.notifyError(asError(error, "invalid_envelope"));
        continue;
      }
      this.handleEnvelope(envelope, resolveJoin, rejectJoin);
    }
  }

  private handleEnvelope(envelope: WorldEnvelope, resolveJoin: (welcome: WorldWelcomePayload) => void, rejectJoin: (error: Error) => void): void {
    if (envelope.type === "world.welcome") {
      if (envelope.request_id !== this.joinRequestId) return;
      try {
        const welcome = parseWorldWelcomePayload(envelope.payload);
        const applied = this.buffer.apply(welcome.snapshot);
        if (applied.status !== "applied") throw new Error("Welcome baseline could not be applied");
        this.lastSnapshotAt = Date.now();
        this.welcome = welcome;
        this.clearJoinTimeout();
        this.connecting = null;
        this.pendingJoinReject = null;
        this.emitSnapshot(applied);
        resolveJoin(welcome);
      } catch (error) {
        this.failJoin(rejectJoin, error, "invalid_welcome");
        this.socket?.close(4002, "invalid world welcome");
      }
      return;
    }
    if (envelope.type === "world.snapshot") {
      let payload: WorldSnapshotPayload;
      try {
        payload = parseWorldSnapshotPayload(envelope.payload);
      } catch (error) {
        this.requestResync(asError(error, "invalid_snapshot").message);
        return;
      }
      this.snapshotCount += 1;
      this.lastSnapshotAt = Date.now();
      this.emitSnapshot(this.buffer.apply(payload));
      return;
    }
    if (envelope.type === "world.transfer") {
      try {
        const transfer = parseWorldTransferPayload(envelope.payload);
        this.transferListeners.forEach((listener) => listener(transfer));
      } catch (error) {
        this.notifyError(asError(error, "invalid_transfer"));
      }
      return;
    }
    if (envelope.type === "world.events") {
      try {
        const batch = parseWorldEventBatch(envelope.payload);
        for (const event of batch.events) this.eventListeners.forEach((listener) => listener(event));
      } catch (error) {
        this.notifyError(asError(error, "invalid_events"));
      }
      return;
    }
    if (envelope.type === "world.reject") {
      const reject = parseReject(envelope.payload);
      this.rejectListeners.forEach((listener) => listener(reject));
      if (this.connecting && envelope.request_id === this.joinRequestId) {
        this.failJoin(rejectJoin, new WorldSocketError(reject.message || reject.code, reject.code), reject.code);
      }
    }
  }

  private emitSnapshot(result: SnapshotApplyResult): void {
    if (result.status === "resync_required") {
      this.requestResync(result.reason);
      return;
    }
    if (result.status === "ignored") return;
    this.snapshotListeners.forEach((listener) => listener(result.state));
  }

  private requestResync(reason: string): void {
    this.resyncListeners.forEach((listener) => listener(reason));
  }

  private failJoin(reject: (error: Error) => void, error: unknown, code: string): void {
    if (!this.connecting) return;
    const failure = asError(error, code);
    this.clearJoinTimeout();
    this.connecting = null;
    this.pendingJoinReject = null;
    reject(failure);
    this.notifyError(failure);
  }

  private clearJoinTimeout(): void {
    if (this.joinTimeout === null) return;
    clearTimeout(this.joinTimeout);
    this.joinTimeout = null;
  }

  private notifyError(error: Error): void {
    this.errorListeners.forEach((listener) => listener(error));
  }
}

const socketOpenState = 1;
const socketClosingState = 2;

function defaultSocketFactory(url: string): SocketLike {
  if (typeof WebSocket === "undefined") throw new Error("WebSocket is not available in this environment");
  return new WebSocket(url);
}

function buildSocketURL(baseURL: string | undefined, token: string, tabId: string | null): string {
  const defaultURL = `${BACKEND_URL.replace(/^http/, "ws")}/api/ws`;
  const url = new URL(baseURL ?? defaultURL);
  url.searchParams.set("token", token);
  url.searchParams.set("purpose", "world");
  if (tabId) url.searchParams.set("tab_id", tabId);
  return url.toString();
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

function isValidJoin(join: WorldJoinPayload): boolean {
  return [join.zone_id, join.map_version, join.content_version].every((value) =>
    typeof value === "string" && value.length > 0 && value.length <= 80 && value.trim() === value,
  );
}

function parseReject(value: unknown): WorldRejectPayload {
  if (typeof value !== "object" || value === null || Array.isArray(value)) return { code: "realtime_world_rejected" };
  const record = value as Record<string, unknown>;
  return {
    code: typeof record.code === "string" ? record.code : "realtime_world_rejected",
    message: typeof record.message === "string" ? record.message : undefined,
    retryable: typeof record.retryable === "boolean" ? record.retryable : undefined,
  };
}

function createRequestId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) return crypto.randomUUID();
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function asError(error: unknown, code: string): WorldSocketError {
  return new WorldSocketError(error instanceof Error ? error.message : String(error), code);
}
