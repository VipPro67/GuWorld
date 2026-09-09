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
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private isExplicitClose = false;
  private subscriptions = new Map<string, Set<(data: unknown) => void>>();
  private connectPromise: Promise<void> | null = null;

  constructor(url: string = DEFAULT_WS_URL) {
    this.url = url;
  }

  public setUrl(url: string) {
    this.url = url;
  }

  public isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
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
  }

  public subscribe<T = unknown>(channel: string, callback: (data: T) => void): () => void {
    if (!this.subscriptions.has(channel)) {
      this.subscriptions.set(channel, new Set());
      this.sendSubscription(channel);
    }
    this.subscriptions.get(channel)!.add(callback as (data: unknown) => void);

    return () => {
      const subs = this.subscriptions.get(channel);
      if (subs) {
        subs.delete(callback as (data: unknown) => void);
        if (subs.size === 0) {
          this.subscriptions.delete(channel);
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
    try {
      // Split raw by newlines if server batched multiple frames in one buffer
      const lines = raw.split("\n");
      for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) continue;
        const msg = JSON.parse(trimmed) as WSMessage;

        if (msg.type === "pong") {
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
    this.pingTimer = setInterval(() => {
      if (this.isConnected()) {
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
  }

  private scheduleReconnect() {
    if (this.isExplicitClose || !this.token) return;

    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
    }

    const delay = Math.min(
      this.baseReconnectDelay * Math.pow(1.5, this.reconnectAttempts),
      this.maxReconnectDelay
    );
    this.reconnectAttempts++;

    this.reconnectTimer = setTimeout(() => {
      if (!this.isExplicitClose && this.token) {
        this.connect(this.token).catch(() => {
          // Handled by onclose -> scheduleReconnect
        });
      }
    }, delay);
  }
}

export const wsManager = new WebSocketManager();
