const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const DEFAULT_WS_URL = BACKEND_URL.replace(/^http/, "ws") + "/api/ws";

export interface WSMessage {
  type?: string;
  channel?: string;
  data?: unknown;
  payload?: unknown;
}

export class WebSocketManager {
  private ws: WebSocket | null = null;
  private url: string;
  private token: string | null = null;
  private reconnectAttempts = 0;
  private maxReconnectDelay = 30000;
  private baseReconnectDelay = 1000;
  private maxReconnectAttempts = 10;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private pongTimeout: ReturnType<typeof setTimeout> | null = null;
  private isExplicitClose = false;
  private subscriptions = new Map<string, Set<(data: unknown) => void>>();
  private connectPromise: Promise<void> | null = null;
  private lastPingTime: number = 0;
  private onTokenRefresh: (() => void) | null = null;
  private subscriptionRefCounts = new Map<string, number>();
  private connectionRefCounts = 0;

  constructor(url: string = DEFAULT_WS_URL) {
    this.url = url;
  }

  public setUrl(url: string) {
    this.url = url;
  }

  public isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }

  public setOnTokenRefresh(callback: () => void) {
    this.onTokenRefresh = callback;
  }

  // Get the current connection reference count
  public getRefCount(): number {
    return this.connectionRefCounts;
  }

  // Acquire a WebSocket connection with reference counting
  public acquireConnection(token?: string): () => void {
    if (token) {
      this.token = token;
    }
    this.connectionRefCounts++;
    if (this.connectionRefCounts === 1) {
      // First subscriber, establish connection
      if (!this.token) {
        console.warn("[WebSocket] Cannot acquire connection without token");
        return () => this.releaseConnection();
      }
      this.connect(this.token).catch((err) => {
        console.error("[WebSocket] Failed to acquire connection:", err);
        this.releaseConnection();
      });
    }
    return () => this.releaseConnection();
  }

  // Release a WebSocket connection reference
  private releaseConnection() {
    this.connectionRefCounts--;
    if (this.connectionRefCounts <= 0) {
      this.connectionRefCounts = 0;
      if (!this.isExplicitClose) {
        // Wait a bit before disconnecting to allow new subscribers
        setTimeout(() => {
          if (this.connectionRefCounts === 0) {
            this.disconnect();
          }
        }, 30000);
      }
    }
  }

  public connect(token: string): Promise<void> {
    if (this.isConnected() && this.token === token) {
      return Promise.resolve();
    }

    this.token = token;
    this.isExplicitClose = false;

    if (this.connectPromise) {
      return this.connectPromise;
    }

    this.connectPromise = new Promise<void>((resolve, reject) => {
      try {
        if (this.ws) {
          try {
            this.ws.close();
          } catch {
            // ignore
          }
          this.ws = null;
        }

        const wsUrl = `${this.url}?token=${encodeURIComponent(token)}`;
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
          this.reconnectAttempts = 0;
          this.startHeartbeat();

          // Re-subscribe any channels already registered in subscriptions map
          this.resubscribeChannels();

          this.connectPromise = null;
          resolve();
        };

        this.ws.onmessage = (event: MessageEvent<string>) => {
          this.handleMessage(event.data);
        };

        this.ws.onerror = (error) => {
          console.warn("[WebSocket] error:", error);
        };

        this.ws.onclose = () => {
          this.stopHeartbeat();
          this.connectPromise = null;
          this.ws = null;

          if (!this.isExplicitClose) {
            this.scheduleReconnect();
          }
        };
      } catch (err) {
        this.connectPromise = null;
        reject(err);
      }
    });

    return this.connectPromise;
  }

  public disconnect() {
    this.isExplicitClose = true;
    this.connectionRefCounts = 0;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.stopHeartbeat();
    if (this.ws) {
      try {
        this.ws.close();
      } catch {
        // ignore
      }
      this.ws = null;
    }
    this.connectPromise = null;
    // Clear all subscriptions
    this.subscriptions.clear();
    this.subscriptionRefCounts.clear();
  }

  public subscribe<T = unknown>(channel: string, callback: (data: T) => void): () => void {
    // Update reference count
    const currentRef = this.subscriptionRefCounts.get(channel) || 0;
    this.subscriptionRefCounts.set(channel, currentRef + 1);

    if (!this.subscriptions.has(channel)) {
      this.subscriptions.set(channel, new Set());
      this.sendSubscription(channel);
    }
    this.subscriptions.get(channel)!.add(callback as (data: unknown) => void);

    return () => {
      const newRef = currentRef - 1;
      this.subscriptionRefCounts.set(channel, newRef);
      
      const subs = this.subscriptions.get(channel);
      if (subs) {
        subs.delete(callback as (data: unknown) => void);
        if (subs.size === 0) {
          this.subscriptions.delete(channel);
          this.subscriptionRefCounts.delete(channel);
          // Notify server to unsubscribe
          this.send("unsubscribe", channel, { channel });
        }
      }
    };
  }

  public send(type: string, channel?: string, payload?: unknown) {
    if (!this.isConnected()) {
      return;
    }
    const msg = {
      type,
      channel,
      payload,
    };
    try {
      this.ws!.send(JSON.stringify(msg));
    } catch (err) {
      console.error("[WebSocket] Failed to send message:", err);
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

  private handleMessage(raw: string) {
    // Message size validation: reject oversized messages
    const MAX_MESSAGE_SIZE = 65536; // 64KB
    if (raw.length > MAX_MESSAGE_SIZE) {
      console.warn(`[WebSocket] Message too large (${raw.length} bytes), dropping`);
      return;
    }

    try {
      // Split raw by newlines if server batched multiple frames in one buffer
      const lines = raw.split("\n");
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;
        const msg = JSON.parse(trimmed) as WSMessage;

        if (msg.type === "pong") {
          // Clear pong timeout since we received a pong
          if (this.pongTimeout) {
            clearTimeout(this.pongTimeout);
            this.pongTimeout = null;
          }
          continue;
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
                console.error(`[WebSocket] Callback error on channel ${channel}:`, e);
              }
            });
          }
        }
      }
    } catch (err) {
      console.warn("[WebSocket] Failed to parse message:", raw, err);
    }
  }

  private startHeartbeat() {
    this.stopHeartbeat();
    this.lastPingTime = Date.now();
    this.pingTimer = setInterval(() => {
      if (this.isConnected()) {
        this.lastPingTime = Date.now();
        // Set pong timeout: if no pong received within 10s, connection is stale
        this.pongTimeout = setTimeout(() => {
          console.warn("[WebSocket] Pong timeout detected, closing stale connection");
          this.reconnectWithStaleToken();
        }, 10000);

        try {
          this.ws!.send(JSON.stringify({ type: "ping" }));
        } catch {
          // ignore
        }
      }
    }, 30000);
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
    this.connectPromise = null;
    this.ws = null;
    if (!this.isExplicitClose && this.token) {
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    if (this.isExplicitClose || !this.token) return;

    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("[WebSocket] Max reconnect attempts reached, giving up");
      return;
    }

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }

    const delay = Math.min(
      this.baseReconnectDelay * Math.pow(1.5, this.reconnectAttempts),
      this.maxReconnectDelay
    );
    // Add jitter: 50-100% of delay to prevent thundering herd
    const jitter = delay * (0.5 + Math.random() * 0.5);
    this.reconnectAttempts++;

    this.reconnectTimer = setTimeout(() => {
      if (!this.isExplicitClose && this.token) {
        // Trigger token refresh if available before reconnecting
        if (this.onTokenRefresh) {
          try {
            this.onTokenRefresh();
          } catch (e) {
            console.warn("[WebSocket] Token refresh callback failed:", e);
          }
        }
        this.connect(this.token).catch(() => {
          // Handled by onclose -> scheduleReconnect
        });
      }
    }, jitter);
  }
}

export const wsManager = new WebSocketManager();
