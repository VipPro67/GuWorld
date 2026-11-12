import { BACKEND_URL } from "../config.ts";
import {
  claimTabIfConfigured,
  defaultTabCoordinator,
  TabIdCoordinator,
  type BroadcastChannelLike,
} from "../tabCoordinator.ts";
import { UIStream, type WebTransportStreamLike } from "./UIStream.ts";
import { WebSocketTransport } from "./WebSocketTransport.ts";

export { TabIdCoordinator, defaultTabCoordinator, setTabClaimer } from "../tabCoordinator.ts";
export type { BroadcastChannelLike, TabClaimer } from "../tabCoordinator.ts";

export interface WebTransportDatagramDuplexLike {
  readable: ReadableStream<Uint8Array>;
  writable: WritableStream<Uint8Array>;
  maxDatagramSize: number;
}

export interface WebTransportLike {
  ready: Promise<void>;
  closed: Promise<{ closeCode?: number; reason?: string }>;
  datagrams: WebTransportDatagramDuplexLike;
  createBidirectionalStream(): Promise<WebTransportStreamLike>;
  close(info?: { closeCode?: number; reason?: string }): void;
}

export type WebTransportCtor = new (url: string) => WebTransportLike;

/** Terminal capability failure: the browser cannot open a WebTransport session. */
export class RealtimeUnsupportedError extends Error {
  public readonly code = "webtransport_unsupported";

  constructor(message = "Trình duyệt không hỗ trợ WebTransport") {
    super(message);
    this.name = "RealtimeUnsupportedError";
  }
}

export interface RealtimeClientOptions {
  url?: string;
  wsURL?: string;
  graceMs?: number;
  getToken?: () => string | null;
  getTabId?: () => string | null;
  setTabId?: (tabId: string) => void;
  createTransport?: (url: string) => WebTransportLike;
  createWebSocketTransport?: (url: string) => WebTransportLike;
  createBroadcastChannel?: (name: string) => BroadcastChannelLike | null;
  onTabReplaced?: () => void;
  tabCoordinator?: TabIdCoordinator;
}

export interface AcquiredWorldSession {
  session: WebTransportLike;
  stream: WebTransportStreamLike;
  release: () => void;
}

export interface AcquiredUISession {
  session: WebTransportLike;
  uiStream: UIStream;
  release: () => void;
}

export const CLOSE_CODE_SUPERSEDED_SAME_TAB = 4000;
export const CLOSE_CODE_SESSION_REPLACED = 4001;
export const CLOSE_CODE_PROTOCOL_MISMATCH = 4002;
export const CLOSE_CODE_JOIN_TIMEOUT = 4003;
export const CLOSE_CODE_RATE_LIMITED = 4008;
export const CLOSE_CODE_SERVER_DRAINING = 4010;

const DEFAULT_GRACE_MS = 30_000;
const WT_CONNECT_TIMEOUT_MS = 3_000;

/**
 * Shared per-tab WebTransport session manager (WT-311, WT-316 §1).
 * Multiplexes the binary World stream (0x01), the NDJSON UI stream (0x02),
 * and QUIC datagrams over a single WebTransport connection.
 */
export class RealtimeClient {
  private readonly options: RealtimeClientOptions;
  private readonly graceMs: number;
  private readonly tabCoordinator: TabIdCoordinator;

  private session: WebTransportLike | null = null;
  private sessionKind: "wt" | "ws" | null = null;
  private readonly transportKinds = new WeakMap<WebTransportLike, "wt" | "ws">();
  private fallbackPromise: Promise<WebTransportLike> | null = null;
  private connectPromise: Promise<WebTransportLike> | null = null;
  private activeToken: string | null = null;
  private activeTabId: string | null = null;
  private worldRefs = 0;
  private uiRefs = 0;
  private generation = 0;
  private graceTimer: ReturnType<typeof setTimeout> | null = null;
  private lastCloseCode = 0;
  private terminalStop = false;

  private readonly closeListeners = new Set<(info: { closeCode: number; reason: string }) => void>();

  constructor(options: RealtimeClientOptions = {}) {
    this.options = options;
    this.graceMs = options.graceMs ?? DEFAULT_GRACE_MS;
    this.tabCoordinator = options.tabCoordinator ?? (
      options.getTabId || options.setTabId || options.createBroadcastChannel
        ? new TabIdCoordinator({
            getTabId: options.getTabId,
            setTabId: options.setTabId,
            createBroadcastChannel: options.createBroadcastChannel,
          })
        : defaultTabCoordinator
    );
    this.tabCoordinator.onConflictRotated((rotatedTabId) => {
      if (this.session && this.activeTabId && this.activeTabId !== rotatedTabId) {
        this.closeCurrentSession(0, "tab_id rotated");
      }
    });
  }

  public getWorldRefCount(): number {
    return this.worldRefs;
  }

  public getUIRefCount(): number {
    return this.uiRefs;
  }

  public getTotalRefCount(): number {
    return this.worldRefs + this.uiRefs;
  }

  public getLastCloseCode(): number {
    return this.lastCloseCode;
  }

  public isTerminalStopped(): boolean {
    return this.terminalStop;
  }

  public hasActiveSession(): boolean {
    return this.session !== null;
  }

  public onSessionClosed(listener: (info: { closeCode: number; reason: string }) => void): () => void {
    this.closeListeners.add(listener);
    return () => this.closeListeners.delete(listener);
  }

  public acquireWorld(token?: string): () => void {
    this.cancelGraceTimer();
    this.worldRefs += 1;
    const gen = this.generation;
    let released = false;
    const resolvedToken = token ?? this.resolveToken();
    if (resolvedToken) {
      void this.ensureSession(resolvedToken).catch(() => {});
    }
    return () => {
      if (released || gen !== this.generation) return;
      released = true;
      this.worldRefs = Math.max(0, this.worldRefs - 1);
      this.checkGraceRelease();
    };
  }

  public acquireUI(token?: string): () => void {
    this.cancelGraceTimer();
    this.uiRefs += 1;
    const gen = this.generation;
    let released = false;
    const resolvedToken = token ?? this.resolveToken();
    if (resolvedToken) {
      void this.ensureSession(resolvedToken).catch(() => {});
    }
    return () => {
      if (released || gen !== this.generation) return;
      released = true;
      this.uiRefs = Math.max(0, this.uiRefs - 1);
      this.checkGraceRelease();
    };
  }

  public async openWorldStream(token?: string): Promise<AcquiredWorldSession> {
    const release = this.acquireWorld(token);
    try {
      let session = await this.ensureSession(token ?? this.resolveToken());
      let stream: WebTransportStreamLike;
      try {
        stream = await session.createBidirectionalStream();
      } catch (error) {
        if (this.transportKinds.get(session) !== "wt" || this.terminalStop) throw error;
        session = await this.switchToWebSocket(session);
        stream = await session.createBidirectionalStream();
      }
      return { session, stream, release };
    } catch (error) {
      release();
      throw error;
    }
  }

  public async openUIStream(token?: string): Promise<AcquiredUISession> {
    const release = this.acquireUI(token);
    try {
      let session = await this.ensureSession(token ?? this.resolveToken());
      let uiStream: UIStream;
      try {
        const rawStream = await session.createBidirectionalStream();
        uiStream = new UIStream(rawStream);
        await uiStream.open();
      } catch (error) {
        if (this.transportKinds.get(session) !== "wt" || this.terminalStop) throw error;
        session = await this.switchToWebSocket(session);
        const rawStream = await session.createBidirectionalStream();
        uiStream = new UIStream(rawStream);
        await uiStream.open();
      }
      return {
        session,
        uiStream,
        release: () => {
          uiStream.close();
          release();
        },
      };
    } catch (error) {
      release();
      throw error;
    }
  }

  public getEndpointURL(): string {
    try {
      const endpoint = new URL(buildRealtimeWTURL(this.options.url, "", null));
      endpoint.search = "";
      endpoint.hash = "";
      endpoint.username = "";
      endpoint.password = "";
      return endpoint.toString();
    } catch {
      // The diagnostic helper reports invalid configuration without exposing it.
      return "";
    }
  }

  public async ensureSession(token?: string | null): Promise<WebTransportLike> {
    if (this.terminalStop) {
      throw new Error("Phiên realtime đã bị dừng");
    }
    const resolvedToken = token ?? this.resolveToken();
    if (!resolvedToken) {
      throw new Error("Chưa có token đăng nhập");
    }
    if (this.connectPromise && this.activeToken === resolvedToken) {
      return this.connectPromise;
    }
    if (this.session && this.activeToken === resolvedToken) {
      await this.session.ready;
      return this.session;
    }

    if (this.session && this.activeToken !== resolvedToken) {
      this.closeCurrentSession(0, "token changed");
    }

    this.activeToken = resolvedToken;
    const tabId = this.tabCoordinator.ensureTabId();
    this.activeTabId = tabId;
    const generation = this.generation;
    const promise = (async () => {
      try {
        await claimTabIfConfigured(resolvedToken, tabId);
      } catch (err) {
        console.warn("[RealtimeClient] tab claim preflight failed:", err);
      }
      const url = buildRealtimeWTURL(this.options.url, resolvedToken, tabId);
      const factory = this.options.createTransport;
      const TransportCtor = defaultWebTransportCtor();
      const configuredWT = (process.env.NEXT_PUBLIC_REALTIME_URL ?? "").trim();
      const wtAvailable = Boolean(factory || this.options.url || configuredWT || BACKEND_URL.startsWith("https://"));
      let transport: WebTransportLike | null = null;
      let kind: "wt" | "ws" = "wt";
      try {
        if (wtAvailable && (factory || TransportCtor)) {
          transport = factory ? factory(url) : new (TransportCtor as WebTransportCtor)(url);
          this.session = transport;
          try {
            let timeout: ReturnType<typeof setTimeout> | null = null;
            await Promise.race([
              transport.ready,
              new Promise<never>((_, reject) => { timeout = setTimeout(() => reject(new Error("WebTransport connect timeout")), WT_CONNECT_TIMEOUT_MS); }),
            ]).finally(() => { if (timeout !== null) clearTimeout(timeout); });
          } catch {
            transport.close();
            if (this.session === transport) this.session = null;
            transport = null;
          }
        }
      } catch {
        transport = null;
      }
      if (!transport) {
        kind = "ws";
        const wsURL = buildRealtimeWSURL(this.options.wsURL, this.options.url, resolvedToken, tabId);
        transport = this.options.createWebSocketTransport?.(wsURL) ?? new WebSocketTransport(wsURL);
        await transport.ready;
      }
      if (this.generation !== generation || this.activeToken !== resolvedToken || this.activeTabId !== tabId) {
        transport.close();
        throw new Error("Realtime session changed while connecting");
      }
      this.session = transport;
      this.sessionKind = kind;
      this.transportKinds.set(transport, kind);
      transport.closed
        .then((info) => this.handleTransportClosed(transport, info?.closeCode ?? 0, info?.reason ?? ""))
        .catch(() => this.handleTransportClosed(transport, 0, "transport closed with error"));
      return transport;
    })();
    this.connectPromise = promise;
    try { return await promise; }
    catch (error) {
      if (this.connectPromise === promise) this.connectPromise = null;
      throw error;
    }
  }

  public disconnect(closeCode = 0, reason = "client disconnected"): void {
    this.cancelGraceTimer();
    this.worldRefs = 0;
    this.uiRefs = 0;
    this.generation += 1;
    this.terminalStop = false;
    this.closeCurrentSession(closeCode, reason);
  }

  private async switchToWebSocket(failed: WebTransportLike): Promise<WebTransportLike> {
    if (this.session !== failed) {
      if (this.fallbackPromise) return this.fallbackPromise;
      if (this.session && this.sessionKind === "ws") return this.session;
      if (this.session || this.transportKinds.get(failed) !== "wt" || this.terminalStop) {
        throw new Error("Realtime session changed while opening stream");
      }
    }
    if (this.fallbackPromise) return this.fallbackPromise;
    const token = this.activeToken;
    const tabId = this.activeTabId;
    const generation = this.generation;
    if (!token) throw new Error("Chưa có token đăng nhập");
    this.closeCurrentSession(0, "WebTransport stream unavailable");
    const promise = (async () => {
      const wsURL = buildRealtimeWSURL(this.options.wsURL, this.options.url, token, tabId);
      const transport = this.options.createWebSocketTransport?.(wsURL) ?? new WebSocketTransport(wsURL);
      await transport.ready;
      if (this.generation !== generation || this.activeToken !== token || this.activeTabId !== tabId) {
        transport.close();
        throw new Error("Realtime session changed while connecting");
      }
      this.session = transport;
      this.sessionKind = "ws";
      this.transportKinds.set(transport, "ws");
      transport.closed
        .then((info) => this.handleTransportClosed(transport, info?.closeCode ?? 0, info?.reason ?? ""))
        .catch(() => this.handleTransportClosed(transport, 0, "transport closed with error"));
      return transport;
    })();
    this.fallbackPromise = promise;
    this.connectPromise = promise;
    try { return await promise; }
    catch (error) {
      if (this.connectPromise === promise) this.connectPromise = null;
      throw error;
    } finally { if (this.fallbackPromise === promise) this.fallbackPromise = null; }
  }

  private closeCurrentSession(closeCode: number, reason: string): void {
    const current = this.session;
    this.session = null;
    this.sessionKind = null;
    this.connectPromise = null;
    if (current) {
      try {
        current.close({ closeCode, reason });
      } catch {
        // ignore
      }
    }
  }

  private handleTransportClosed(transport: WebTransportLike, closeCode: number, reason: string): void {
    if (this.session !== transport) {
      // Old superseded session closing (e.g. 4000 superseded_same_tab): ignore.
      return;
    }
    this.session = null;
    this.sessionKind = null;
    this.connectPromise = null;
    this.lastCloseCode = closeCode;

    if (closeCode === CLOSE_CODE_SESSION_REPLACED) {
      this.terminalStop = true;
      this.cancelGraceTimer();
      this.worldRefs = 0;
      this.uiRefs = 0;
      this.options.onTabReplaced?.();
      if (typeof window !== "undefined") {
        window.dispatchEvent(new CustomEvent("game:tab-replaced"));
      }
    } else if (closeCode === CLOSE_CODE_PROTOCOL_MISMATCH) {
      this.terminalStop = true;
      this.cancelGraceTimer();
    } else if (closeCode === CLOSE_CODE_SERVER_DRAINING) {
      // WT-311 §6: a deploy/shutdown closed the session. Clients keep their
      // refcounts and reconnect with backoff, so do not mark this terminal.
    }

    for (const listener of this.closeListeners) {
      try {
        listener({ closeCode, reason });
      } catch {
        // ignore
      }
    }
  }

  private checkGraceRelease(): void {
    if (this.getTotalRefCount() > 0) return;
    this.cancelGraceTimer();
    const gen = this.generation;
    this.graceTimer = setTimeout(() => {
      this.graceTimer = null;
      if (this.getTotalRefCount() === 0 && gen === this.generation) {
        this.closeCurrentSession(0, "idle grace expired");
      }
    }, this.graceMs);
  }

  private cancelGraceTimer(): void {
    if (this.graceTimer !== null) {
      clearTimeout(this.graceTimer);
      this.graceTimer = null;
    }
  }

  private resolveToken(): string | null {
    return this.options.getToken?.() ?? readStorage("localStorage", "gu_auth_token");
  }
}

export function buildRealtimeWTURL(baseURL: string | undefined, token: string, tabId: string | null): string {
  const configured = (process.env.NEXT_PUBLIC_REALTIME_URL ?? "").trim();
  const fallback = `${BACKEND_URL.replace(/^http/, "https")}/api/wt`;
  const url = new URL(baseURL ?? (configured || fallback));
  url.searchParams.set("token", token);
  if (tabId) url.searchParams.set("tab_id", tabId);
  return url.toString();
}

export function buildRealtimeWSURL(baseURL: string | undefined, wtURL: string | undefined, token: string, tabId: string | null): string {
  const configured = (process.env.NEXT_PUBLIC_REALTIME_WS_URL ?? "").trim();
  const configuredWT = (process.env.NEXT_PUBLIC_REALTIME_URL ?? "").trim();
  const endpoint = baseURL || configured || (wtURL || configuredWT
    ? buildRealtimeWTURL(wtURL, "", null)
    : `${BACKEND_URL}/api/ws`);
  const url = new URL(endpoint);
  if (url.protocol === "https:") url.protocol = "wss:";
  else if (url.protocol === "http:") url.protocol = "ws:";
  if (url.pathname.endsWith("/api/wt")) url.pathname = url.pathname.slice(0, -2) + "ws";
  url.searchParams.set("token", token);
  if (tabId) url.searchParams.set("tab_id", tabId);
  return url.toString();
}

function defaultWebTransportCtor(): WebTransportCtor | null {
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

export const realtimeClient = new RealtimeClient();
