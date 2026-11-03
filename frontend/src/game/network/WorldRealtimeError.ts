/** Error surfaced by the WebTransport world/UI clients. */
export class WorldRealtimeError extends Error {
  public readonly code: string;

  constructor(message: string, code = "world_realtime_error") {
    super(message);
    this.name = "WorldRealtimeError";
    this.code = code;
  }
}
