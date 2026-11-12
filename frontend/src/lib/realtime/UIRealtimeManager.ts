import { realtimeClient, type RealtimeClient } from "./RealtimeClient.ts";
import type { UIStream, UIStreamMessage } from "./UIStream.ts";

export interface RealtimeUIMessage {
  type?: string;
  channel?: string;
  data?: unknown;
  payload?: unknown;
  request_id?: string;
  ok?: boolean;
  status?: number;
  message?: string;
}

export type RealtimeConnectionState = "disconnected" | "connecting" | "connected";

export interface RealtimeOpenEvent {
  reconnected: boolean;
}

export interface RealtimeUIManagerOptions {
  realtimeClient?: RealtimeClient;
}

export class RealtimeRequestError extends Error {
  public readonly status: number;
  constructor(message: string, status = 0) {
    super(message);
    this.status = status;
    this.name = "RealtimeRequestError";
  }
}

const RECONNECT_BASE_DELAY = 1000;
const RECONNECT_MAX_DELAY = 30000;
const MAX_RECONNECT_ATTEMPTS = 10;
const CONNECT_TIMEOUT_MS = 10000;
const IDLE_DISCONNECT_MS = 30000;
const HEARTBEAT_INTERVAL_MS = 30000;
const PONG_TIMEOUT_MS = 10000;
const MAX_MESSAGE_SIZE = 4 * 1024 * 1024;

/**
 * Player-scoped UI event client. It multiplexes the NDJSON UI stream (0x02)
 * over the shared per-tab WebTransport session and never opens a WebSocket.
 */
export class RealtimeUIManager {
  private uiStream: UIStream | null = null;
  private uiStreamRelease: (() => void) | null = null;
  private readonly realtime: RealtimeClient;
  private token: string | null = null;
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private pongTimeout: ReturnType<typeof setTimeout> | null = null;
  private isExplicitClose = false;
  private subscriptions = new Map<string, Set<(data: unknown) => void>>();
  private connectPromise: Promise<void> | null = null;
  private onTokenRefresh: (() => void) | null = null;
  private subscriptionRefCounts = new Map<string, number>();
  private connectionRefCounts = 0;
  private pendingRequests = new Map<string, { resolve: (data: unknown) => void; reject: (error: Error) => void; timer: ReturnType<typeof setTimeout> }>();
  private openListeners = new Set<(event: RealtimeOpenEvent) => void>();
  private hasOpened = false;
  private connectionState: RealtimeConnectionState = "disconnected";
  private connectionListeners = new Set<(state: RealtimeConnectionState) => void>();
  private cancelConnect: ((error: Error) => void) | null = null;
  private connectionGeneration = 0;

  constructor(options: RealtimeUIManagerOptions = {}) {
    this.realtime = options.realtimeClient ?? realtimeClient;
  }

  public isConnected(): boolean {
    return this.uiStream !== null && this.uiStream.isOpen();
  }

  public getConnectionState(): RealtimeConnectionState {
    return this.connectionState;
  }

  public onConnectionChange(callback: (state: RealtimeConnectionState) => void): () => void {
    this.connectionListeners.add(callback);
    return () => this.connectionListeners.delete(callback);
  }

  private setConnectionState(state: RealtimeConnectionState) {
    if (state === this.connectionState) return;
    this.connectionState = state;
    for (const listener of this.connectionListeners) {
      try {
        listener(state);
      } catch (error) {
        console.error("[RealtimeUI] Connection callback failed:", error);
      }
    }
  }

  public setOnTokenRefresh(callback: () => void) {
    this.onTokenRefresh = callback;
  }

  public onOpen(callback: (event: RealtimeOpenEvent) => void): () => void {
    this.openListeners.add(callback);
    return () => this.openListeners.delete(callback);
  }

  public getRefCount(): number {
    return this.connectionRefCounts;
  }

  public acquireConnection(token?: string): () => void {
    const connectionToken = token || this.token;
    this.connectionRefCounts++;
    const generation = this.connectionGeneration;
    let released = false;
    if (this.connectionRefCounts === 1 && connectionToken) {
      // The caller owns the reference until cleanup, including while offline.
      void this.connect(connectionToken).catch((error) => console.warn("[RealtimeUI] Connection unavailable:", error));
    }
    return () => {
      if (released || generation !== this.connectionGeneration) return;
      released = true;
      this.releaseConnection();
    };
  }

  private releaseConnection() {
    this.connectionRefCounts--;
    if (this.connectionRefCounts <= 0) {
      this.connectionRefCounts = 0;
      if (!this.isExplicitClose) {
        const generation = this.connectionGeneration;
        // Keep the UI stream alive briefly so switching views does not reopen a
        // stream; when idle expires, release only the UI stream and preserve
        // registered channel listeners for the next acquireConnection().
        setTimeout(() => {
          if (this.connectionRefCounts === 0 && generation === this.connectionGeneration && !this.isExplicitClose) {
            this.closeIdleStream();
          }
        }, IDLE_DISCONNECT_MS);
      }
    }
  }

  private closeIdleStream() {
    const error = new RealtimeRequestError("Kết nối realtime tạm nghỉ");
    this.cancelConnect?.(error);
    this.rejectPendingRequests(error);
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.stopHeartbeat();
    this.closeUIStream();
    this.connectPromise = null;
    this.setConnectionState("disconnected");
  }

  public retryConnection(): void {
    if (this.isExplicitClose || !this.token || this.connectionRefCounts === 0 || this.isConnected() || this.connectPromise) return;
    this.reconnectAttempts = 0;
    void this.connect(this.token).catch(() => { /* normal reconnect handles failure */ });
  }

  public connect(token: string): Promise<void> {
    if (this.isConnected() && this.token === token) return Promise.resolve();
    if (this.connectPromise && this.token === token) return this.connectPromise;

    this.isExplicitClose = false;
    this.token = token;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.cancelConnect?.(new RealtimeRequestError("Kết nối realtime đã thay đổi"));
    this.stopHeartbeat();
    this.closeUIStream();
    this.rejectPendingRequests(new RealtimeRequestError("Kết nối realtime đã đóng"));

    let resolveConnect!: () => void;
    let rejectConnect!: (error: Error) => void;
    const promise = new Promise<void>((resolve, reject) => {
      resolveConnect = resolve;
      rejectConnect = reject;
    });
    this.connectPromise = promise;
    let timer: ReturnType<typeof setTimeout> | null = null;
    const settle = (error?: Error) => {
      if (this.connectPromise !== promise) return;
      if (timer) clearTimeout(timer);
      this.connectPromise = null;
      this.cancelConnect = null;
      if (error) rejectConnect(error);
      else resolveConnect();
    };
    this.cancelConnect = settle;
    this.setConnectionState("connecting");

    timer = setTimeout(() => {
      this.closeUIStream();
      const err = new RealtimeRequestError("WebTransport UI connect timeout");
      settle(err);
      this.rejectPendingRequests(err);
      this.setConnectionState("disconnected");
      this.scheduleReconnect();
    }, CONNECT_TIMEOUT_MS);

    void this.realtime.openUIStream(token)
      .then((acquired) => {
        if (this.connectPromise !== promise || this.isExplicitClose) {
          acquired.release();
          return;
        }
        const stream = acquired.uiStream;
        this.uiStream = stream;
        this.uiStreamRelease = acquired.release;

        stream.onMessage((msg) => {
          if (this.uiStream === stream) {
            this.handleParsedMessage(msg);
          }
        });
        stream.onClose((err) => {
          if (this.uiStream !== stream) return;
          this.closeUIStream();
          this.stopHeartbeat();
          const closeErr = err ?? new RealtimeRequestError("Kết nối WebTransport UI đã đóng");
          settle(closeErr);
          this.rejectPendingRequests(closeErr);
          this.setConnectionState("disconnected");
          if (!this.realtime.isTerminalStopped()) {
            this.scheduleReconnect();
          }
        });

        const event = { reconnected: this.hasOpened };
        this.hasOpened = true;
        this.reconnectAttempts = 0;
        settle();
        this.startHeartbeat();
        this.resubscribeChannels();
        this.setConnectionState("connected");
        for (const listener of this.openListeners) {
          try {
            listener(event);
          } catch (error) {
            console.error("[RealtimeUI] OPEN lifecycle callback failed:", error);
          }
        }
      })
      .catch((error) => {
        const err = error instanceof Error ? error : new RealtimeRequestError(String(error));
        settle(err);
        this.setConnectionState("disconnected");
        if (!this.realtime.isTerminalStopped()) {
          this.scheduleReconnect();
        }
      });
    return promise;
  }

  public disconnect() {
    this.isExplicitClose = true;
    this.connectionRefCounts = 0;
    this.connectionGeneration++;
    const error = new RealtimeRequestError("Kết nối realtime đã đóng");
    this.cancelConnect?.(error);
    this.rejectPendingRequests(error);
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.stopHeartbeat();
    this.closeUIStream();
    if (this.realtime.getWorldRefCount() === 0) {
      this.realtime.disconnect();
    }
    this.connectPromise = null;
    this.hasOpened = false;
    this.setConnectionState("disconnected");
    this.subscriptions.clear();
    this.subscriptionRefCounts.clear();
  }

  private closeUIStream() {
    const release = this.uiStreamRelease;
    const stream = this.uiStream;
    this.uiStreamRelease = null;
    this.uiStream = null;
    try {
      stream?.close();
    } catch {
      // ignore
    }
    try {
      release?.();
    } catch {
      // ignore
    }
  }

  public subscribe<T = unknown>(channel: string, callback: (data: T) => void): () => void {
    const currentRef = this.subscriptionRefCounts.get(channel) || 0;
    this.subscriptionRefCounts.set(channel, currentRef + 1);

    if (!this.subscriptions.has(channel)) {
      this.subscriptions.set(channel, new Set());
      this.sendSubscription(channel);
    }
    this.subscriptions.get(channel)!.add(callback as (data: unknown) => void);

    return () => {
      const newRef = Math.max(0, (this.subscriptionRefCounts.get(channel) || 0) - 1);
      this.subscriptionRefCounts.set(channel, newRef);

      const subs = this.subscriptions.get(channel);
      if (subs) {
        subs.delete(callback as (data: unknown) => void);
        if (subs.size === 0) {
          this.subscriptions.delete(channel);
          this.subscriptionRefCounts.delete(channel);
          this.send("unsubscribe", channel, { channel });
        }
      }
    };
  }

  public send(type: string, channel?: string, payload?: unknown) {
    if (!this.isConnected()) return;
    const msg = { type, channel, payload };
    try {
      void this.uiStream!.sendJSON(msg).catch((err) => {
        console.error("[RealtimeUI] Failed to send message:", err);
      });
    } catch (err) {
      console.error("[RealtimeUI] Failed to send message:", err);
    }
  }

  public request<T>(channel: string, payload?: unknown, timeoutMs = 30000): Promise<T> {
    if (!this.isConnected()) return Promise.reject(new RealtimeRequestError("Realtime chưa kết nối"));
    const requestId = createRequestId();
    return new Promise<T>((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pendingRequests.delete(requestId);
        reject(new RealtimeRequestError("Realtime request timeout"));
      }, timeoutMs);
      this.pendingRequests.set(requestId, { resolve: resolve as (data: unknown) => void, reject, timer });
      try {
        void this.uiStream!.sendJSON({ type: "command", channel, request_id: requestId, payload, data: payload }).catch((error) => {
          clearTimeout(timer);
          this.pendingRequests.delete(requestId);
          reject(error instanceof Error ? error : new RealtimeRequestError(String(error)));
        });
      } catch (error) {
        clearTimeout(timer);
        this.pendingRequests.delete(requestId);
        reject(error instanceof Error ? error : new RealtimeRequestError(String(error)));
      }
    });
  }

  /**
   * Fire-and-forget command for callers that do not need an individual
   * `command.result` response.
   */
  public command(channel: string, payload?: unknown) {
    if (!this.isConnected()) return;
    const requestId = createRequestId();
    try {
      void this.uiStream!.sendJSON({ type: "command", channel, request_id: requestId, payload, data: payload }).catch((err) => {
        console.error("[RealtimeUI] Failed to send command:", err);
      });
    } catch (err) {
      console.error("[RealtimeUI] Failed to send command:", err);
    }
  }

  private sendSubscription(channel: string) {
    if (this.isConnected()) {
      this.send("subscribe", channel, { channel });
    }
  }

  private resubscribeChannels() {
    for (const channel of this.subscriptions.keys()) {
      this.sendSubscription(channel);
    }
  }

  private handleParsedMessage(msg: RealtimeUIMessage | UIStreamMessage) {
    if (msg.type === "error" && msg.request_id) {
      const pending = this.pendingRequests.get(msg.request_id);
      if (pending) {
        clearTimeout(pending.timer);
        this.pendingRequests.delete(msg.request_id);
        pending.reject(new RealtimeRequestError(msg.message || "use HTTP", msg.status ?? 501));
      }
      return;
    }

    if (msg.type === "response" && msg.request_id) {
      const pending = this.pendingRequests.get(msg.request_id);
      if (pending) {
        clearTimeout(pending.timer);
        this.pendingRequests.delete(msg.request_id);
        pending.resolve(msg.data !== undefined ? msg.data : msg.payload);
      }
      return;
    }

    if (msg.type === "command.result" && msg.request_id) {
      const pending = this.pendingRequests.get(msg.request_id);
      if (pending) {
        clearTimeout(pending.timer);
        this.pendingRequests.delete(msg.request_id);
        if (msg.ok) pending.resolve(msg.data);
        else {
          const body = msg.data as { error?: string } | undefined;
          pending.reject(new RealtimeRequestError(body?.error || "Realtime request failed", msg.status));
        }
      }
      return;
    }

    if (msg.type === "pong") {
      if (this.pongTimeout) {
        clearTimeout(this.pongTimeout);
        this.pongTimeout = null;
      }
      return;
    }

    const channel = msg.channel;
    const data = msg.data !== undefined ? msg.data : msg.payload;

    if (channel && this.subscriptions.has(channel)) {
      const cbs = this.subscriptions.get(channel);
      if (cbs) {
        cbs.forEach((cb) => {
          try {
            cb(data);
          } catch (e) {
            console.error(`[RealtimeUI] Callback error on channel ${channel}:`, e);
          }
        });
      }
    }
  }

  private rejectPendingRequests(error: Error) {
    for (const pending of this.pendingRequests.values()) {
      clearTimeout(pending.timer);
      pending.reject(error);
    }
    this.pendingRequests.clear();
  }

  private startHeartbeat() {
    this.stopHeartbeat();
    this.pingTimer = setInterval(() => {
      if (!this.isConnected()) return;
      this.pongTimeout = setTimeout(() => {
        console.warn("[RealtimeUI] Pong timeout detected, closing stale connection");
        this.reconnectWithStaleToken();
      }, PONG_TIMEOUT_MS);
      try {
        this.uiStream?.ping();
      } catch {
        // ignore
      }
    }, HEARTBEAT_INTERVAL_MS);
  }

  private stopHeartbeat() {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
    if (this.pongTimeout) {
      clearTimeout(this.pongTimeout);
      this.pongTimeout = null;
    }
  }

  private reconnectWithStaleToken() {
    this.stopHeartbeat();
    this.cancelConnect?.(new RealtimeRequestError("Kết nối realtime hết phản hồi"));
    this.rejectPendingRequests(new RealtimeRequestError("Kết nối realtime hết phản hồi"));
    this.closeUIStream();
    this.setConnectionState("disconnected");
    if (!this.isExplicitClose && this.token) {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    if (this.isExplicitClose || !this.token) return;

    if (this.reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      console.error("[RealtimeUI] Max reconnect attempts reached, giving up");
      return;
    }

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }

    const delay = Math.min(
      RECONNECT_BASE_DELAY * Math.pow(1.5, this.reconnectAttempts),
      RECONNECT_MAX_DELAY,
    );
    const jitter = delay * (0.5 + Math.random() * 0.5);
    this.reconnectAttempts++;

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      if (!this.isExplicitClose && this.token) {
        if (this.onTokenRefresh) {
          try {
            this.onTokenRefresh();
          } catch (e) {
            console.warn("[RealtimeUI] Token refresh callback failed:", e);
          }
        }
        this.connect(this.token).catch(() => {
          // Handled by the close handler -> scheduleReconnect
        });
      }
    }, jitter);
  }
}

function createRequestId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export const uiRealtimeManager = new RealtimeUIManager();

if (typeof window !== "undefined") {
  window.addEventListener("game:tab-replaced", () => uiRealtimeManager.disconnect());
  window.addEventListener("online", () => uiRealtimeManager.retryConnection());
}
