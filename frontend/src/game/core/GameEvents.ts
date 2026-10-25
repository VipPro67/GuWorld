import type { GameSessionStatus } from "./GameSession.ts";

/** Typed events forwarded from the Phaser client to the React HUD. */
export interface GameEventMap {
  "session:status": { status: GameSessionStatus };
  "world:error": { code: string; message: string };
}

type EventName = keyof GameEventMap;
type Handler<K extends EventName> = (payload: GameEventMap[K]) => void;

/**
 * Typed event bus for the React bridge. Payloads stay coarse (status, errors);
 * per-frame transforms never travel through React state.
 */
export class GameEventBus {
  private readonly handlers = new Map<EventName, Set<Handler<EventName>>>();

  public on<K extends EventName>(event: K, handler: Handler<K>): () => void {
    const set = this.handlers.get(event) ?? new Set<Handler<EventName>>();
    set.add(handler as Handler<EventName>);
    this.handlers.set(event, set);
    return () => this.off(event, handler);
  }

  public off<K extends EventName>(event: K, handler: Handler<K>): void {
    this.handlers.get(event)?.delete(handler as Handler<EventName>);
  }

  public emit<K extends EventName>(event: K, payload: GameEventMap[K]): void {
    const set = this.handlers.get(event);
    if (!set) return;
    for (const handler of [...set]) {
      (handler as Handler<K>)(payload);
    }
  }

  public clear(): void {
    this.handlers.clear();
  }
}
