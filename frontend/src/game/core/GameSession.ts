export type GameSessionStatus =
  | "IDLE"
  | "CONNECTING"
  | "JOINED"
  | "LEAVING"
  | "ERROR";

const ALLOWED: Record<GameSessionStatus, readonly GameSessionStatus[]> = {
  IDLE: ["CONNECTING"],
  CONNECTING: ["JOINED", "ERROR", "LEAVING"],
  JOINED: ["LEAVING", "ERROR"],
  LEAVING: ["IDLE", "ERROR"],
  ERROR: ["IDLE", "CONNECTING"],
};

const TERMINAL_RETRY: Partial<Record<GameSessionStatus, GameSessionStatus>> = {
  LEAVING: "IDLE",
  ERROR: "IDLE",
};

/**
 * Small, framework-free lifecycle for the Phaser world. Zone transfers and the
 * real socket state machine extend this in PR-27 instead of inventing another
 * source of truth in React.
 */
export class GameSession {
  private status: GameSessionStatus = "IDLE";
  private readonly listeners = new Set<(status: GameSessionStatus) => void>();

  public getStatus(): GameSessionStatus {
    return this.status;
  }

  public isJoined(): boolean {
    return this.status === "JOINED";
  }

  public canTransition(next: GameSessionStatus): boolean {
    return ALLOWED[this.status].includes(next);
  }

  public transition(next: GameSessionStatus): boolean {
    if (!this.canTransition(next)) return false;
    this.status = next;
    for (const listener of this.listeners) listener(this.status);
    return true;
  }

  /** Move back to IDLE after leaving or a recoverable error. */
  public reset(): boolean {
    return this.transition(TERMINAL_RETRY[this.status] ?? "IDLE");
  }

  public subscribe(listener: (status: GameSessionStatus) => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }
}
