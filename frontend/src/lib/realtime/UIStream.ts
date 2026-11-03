import { STREAM_UI } from "../../game/network/v2/Codec.ts";

export interface WebTransportStreamLike {
  readable: ReadableStream<Uint8Array>;
  writable: WritableStream<Uint8Array>;
}

export interface UIStreamMessage {
  type?: string;
  channel?: string;
  data?: unknown;
  payload?: unknown;
  request_id?: string;
  ok?: boolean;
  status?: number;
  message?: string;
}

const MAX_UI_LINE_LENGTH = 4 * 1024 * 1024;

/**
 * Bidirectional NDJSON stream (StreamType = 0x02) over a WebTransport session.
 * Carries UI pub/sub subscriptions, channel events, ping/pong, and RPC commands.
 */
export class UIStream {
  private readonly stream: WebTransportStreamLike;
  private writer: WritableStreamDefaultWriter<Uint8Array> | null = null;
  private reader: ReadableStreamDefaultReader<Uint8Array> | null = null;
  private readonly encoder = new TextEncoder();
  private readonly decoder = new TextDecoder();
  private buffer = "";
  private closed = false;
  private headerSent = false;

  private readonly messageListeners = new Set<(msg: UIStreamMessage) => void>();
  private readonly closeListeners = new Set<(error?: Error) => void>();

  constructor(stream: WebTransportStreamLike) {
    this.stream = stream;
  }

  public async open(): Promise<void> {
    if (this.closed) throw new Error("UIStream is already closed");
    this.writer = this.stream.writable.getWriter();
    this.reader = this.stream.readable.getReader();
    await this.writer.write(new Uint8Array([STREAM_UI]));
    this.headerSent = true;
    void this.readLoop();
  }

  public isOpen(): boolean {
    return this.headerSent && !this.closed && this.writer !== null;
  }

  public onMessage(listener: (msg: UIStreamMessage) => void): () => void {
    this.messageListeners.add(listener);
    return () => this.messageListeners.delete(listener);
  }

  public onClose(listener: (error?: Error) => void): () => void {
    this.closeListeners.add(listener);
    return () => this.closeListeners.delete(listener);
  }

  public async sendJSON(msg: Record<string, unknown>): Promise<void> {
    if (!this.writer || this.closed) {
      throw new Error("UIStream is not open");
    }
    const line = `${JSON.stringify(msg)}\n`;
    await this.writer.write(this.encoder.encode(line));
  }

  public subscribe(channel: string): void {
    if (!this.isOpen()) return;
    void this.sendJSON({ type: "subscribe", channel, payload: { channel } }).catch(() => {});
  }

  public unsubscribe(channel: string): void {
    if (!this.isOpen()) return;
    void this.sendJSON({ type: "unsubscribe", channel, payload: { channel } }).catch(() => {});
  }

  public ping(): void {
    if (!this.isOpen()) return;
    void this.sendJSON({ type: "ping" }).catch(() => {});
  }

  public close(): void {
    if (this.closed) return;
    this.closed = true;
    const writer = this.writer;
    const reader = this.reader;
    this.writer = null;
    this.reader = null;
    try {
      void writer?.close().catch(() => {});
    } catch {
      // ignore
    }
    try {
      void reader?.cancel().catch(() => {});
    } catch {
      // ignore
    }
  }

  private async readLoop(): Promise<void> {
    const reader = this.reader;
    if (!reader) return;
    let closeError: Error | undefined;
    try {
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        if (!value || value.byteLength === 0) continue;
        this.buffer += this.decoder.decode(value, { stream: true });
        if (this.buffer.length > MAX_UI_LINE_LENGTH) {
          this.buffer = "";
          continue;
        }
        let newlineIdx = this.buffer.indexOf("\n");
        while (newlineIdx !== -1) {
          const line = this.buffer.slice(0, newlineIdx).trim();
          this.buffer = this.buffer.slice(newlineIdx + 1);
          if (line.length > 0) {
            this.dispatchLine(line);
          }
          newlineIdx = this.buffer.indexOf("\n");
        }
      }
      const trailing = (this.buffer + this.decoder.decode()).trim();
      this.buffer = "";
      if (trailing.length > 0) {
        this.dispatchLine(trailing);
      }
    } catch (error) {
      closeError = error instanceof Error ? error : new Error(String(error));
    }
    if (!this.closed) {
      this.closed = true;
      this.writer = null;
      this.reader = null;
      for (const listener of this.closeListeners) {
        try {
          listener(closeError);
        } catch {
          // ignore
        }
      }
    }
  }

  private dispatchLine(line: string): void {
    let parsed: UIStreamMessage;
    try {
      parsed = JSON.parse(line) as UIStreamMessage;
    } catch {
      return;
    }
    for (const listener of this.messageListeners) {
      try {
        listener(parsed);
      } catch {
        // ignore listener failure
      }
    }
  }
}
