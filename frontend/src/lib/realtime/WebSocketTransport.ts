import type { WebTransportLike } from "./RealtimeClient.ts";
import type { WebTransportStreamLike } from "./UIStream.ts";

type SocketLike = Pick<WebSocket, "binaryType" | "readyState" | "bufferedAmount" | "send" | "close" | "onopen" | "onerror" | "onclose" | "onmessage">;
type Channel = 1 | 2 | 3 | 4;
const MAX_BUFFERED_BYTES = 2 * 1024 * 1024;

/** WebSocket framing: one message = channel byte followed by unmodified protocol bytes. */
export class WebSocketTransport implements WebTransportLike {
  readonly ready: Promise<void>;
  readonly closed: Promise<{ closeCode?: number; reason?: string }>;
  readonly datagrams: WebTransportLike["datagrams"];
  private readonly socket: SocketLike;
  private readonly readers = new Map<number, ReadableStreamDefaultController<Uint8Array>>();
  private datagramReader: ReadableStreamDefaultController<Uint8Array> | null = null;
  private readonly pendingDatagrams: Uint8Array[] = [];
  private nextStreamId = 1;
  private closedResolve!: (info: { closeCode?: number; reason?: string }) => void;
  private readyReject!: (error: Error) => void;
  private opened = false;
  private ended = false;

  constructor(url: string, createSocket: (url: string) => SocketLike = (value) => new WebSocket(value)) {
    this.socket = createSocket(url);
    this.socket.binaryType = "arraybuffer";
    this.ready = new Promise<void>((resolve, reject) => {
      this.readyReject = reject;
      this.socket.onopen = () => { this.opened = true; resolve(); };
      this.socket.onerror = () => {
        if (!this.opened) reject(new Error("WebSocket handshake failed"));
      };
    });
    this.closed = new Promise((resolve) => { this.closedResolve = resolve; });
    this.socket.onclose = (event) => {
      this.ended = true;
      if (!this.opened) this.readyReject(new Error("WebSocket closed before open"));
      for (const controller of this.readers.values()) {
        try { controller.close(); } catch { /* already closed */ }
      }
      this.readers.clear();
      try { this.datagramReader?.close(); } catch { /* already closed */ }
      this.datagramReader = null;
      this.closedResolve({ closeCode: event.code, reason: event.reason });
    };
    this.socket.onmessage = (event) => {
      const data = event.data;
      if (!(data instanceof ArrayBuffer)) return;
      const bytes = new Uint8Array(data);
      if (bytes.length < 1) return;
      const channel = bytes[0] as Channel;
      if (channel === 3) {
        if (this.pendingDatagrams.length === 4) this.pendingDatagrams.shift();
        this.pendingDatagrams.push(bytes.slice(1));
        this.drainDatagrams();
      } else if ((channel === 1 || channel === 2) && bytes.length >= 2) {
        try { this.readers.get(bytes[1])?.enqueue(bytes.slice(2)); } catch { /* released */ }
      } else if (channel === 4 && bytes.length >= 3) {
        const controller = this.readers.get(bytes[2]);
        this.readers.delete(bytes[2]);
        try { controller?.close(); } catch { /* released */ }
      }
    };
    this.datagrams = {
      maxDatagramSize: 65535,
      readable: new ReadableStream<Uint8Array>({
        start: (controller) => { this.datagramReader = controller; },
        pull: () => { this.drainDatagrams(); },
        cancel: () => { this.datagramReader = null; this.pendingDatagrams.length = 0; },
      }, { highWaterMark: 0 }),
      writable: new WritableStream<Uint8Array>({ write: (bytes) => this.sendDatagram(bytes) }),
    };
  }

  async createBidirectionalStream(): Promise<WebTransportStreamLike> {
    await this.ready;
    // WT's first stream byte declares the logical stream (world=1, UI=2).
    let channel: 1 | 2 | null = null;
    const streamId = this.nextStreamId++;
    if (streamId > 255) throw new Error("Too many realtime streams");
    const readable = new ReadableStream<Uint8Array>({
      start: (controller) => { this.readers.set(streamId, controller); },
      cancel: () => { this.readers.delete(streamId); },
    });
    const writable = new WritableStream<Uint8Array>({
      write: (bytes) => {
        if (channel === null) {
          channel = bytes[0] as 1 | 2;
          if (channel !== 1 && channel !== 2) throw new Error("Invalid realtime stream type");
        }
        this.sendStream(channel, streamId, bytes);
      },
      close: () => {
        if (channel !== null) this.sendStreamClose(channel, streamId);
        this.readers.delete(streamId);
      },
    });
    return { readable, writable };
  }

  close(info?: { closeCode?: number; reason?: string }): void {
    if (this.ended) return;
    const code = info?.closeCode && info.closeCode >= 1000 ? info.closeCode : 1000;
    this.socket.close(code, info?.reason ?? "");
  }

  private sendDatagram(bytes: Uint8Array): void {
    // Datagram semantics allow dropping stale movement/ping when the WS queue stalls.
    if (this.socket.bufferedAmount > MAX_BUFFERED_BYTES) return;
    this.send(3, bytes);
  }

  private drainDatagrams(): void {
    const reader = this.datagramReader;
    if (!reader || reader.desiredSize === null || reader.desiredSize < 0) return;
    const next = this.pendingDatagrams.shift();
    if (next) {
      try { reader.enqueue(next); } catch { /* released */ }
    }
  }

  private sendStream(channel: 1 | 2, streamId: number, bytes: Uint8Array): void {
    const frame = new Uint8Array(2 + bytes.length);
    frame[0] = channel;
    frame[1] = streamId;
    frame.set(bytes, 2);
    this.sendRaw(frame);
  }

  private sendStreamClose(channel: 1 | 2, streamId: number): void {
    if (this.socket.readyState === 1) this.sendRaw(new Uint8Array([4, channel, streamId]));
  }

  private send(channel: Channel, bytes: Uint8Array): void {
    if (this.socket.readyState !== 1) throw new Error("WebSocket is not open");
    const frame = new Uint8Array(1 + bytes.length);
    frame[0] = channel;
    frame.set(bytes, 1);
    this.sendRaw(frame);
  }

  private sendRaw(frame: Uint8Array): void {
    if (this.socket.readyState !== 1) throw new Error("WebSocket is not open");
    if (this.socket.bufferedAmount + frame.length > MAX_BUFFERED_BYTES) {
      this.socket.close(4008, "realtime queue full");
      throw new Error("WebSocket send queue full");
    }
    this.socket.send(frame);
  }
}
