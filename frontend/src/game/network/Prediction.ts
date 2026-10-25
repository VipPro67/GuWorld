export interface Vector2 {
  x: number;
  y: number;
}

interface PendingInput {
  seq: number;
  move: Vector2;
  dt: number;
}

export interface ReconcileResult {
  position: Vector2;
  correction: number;
  replayed: number;
}

/**
 * Local movement prediction for the owning character.
 *
 * Inputs are applied immediately for responsive controls, kept until the
 * authoritative snapshot acknowledges them, then replayed on top of the server
 * position. The server still owns the final position; this only hides latency.
 */
export class SelfPredictor {
  private position: Vector2 = { x: 0, y: 0 };
  private speed = 0;
  private ackSeq = 0;
  private pending: PendingInput[] = [];

  public reset(): void {
    this.position = { x: 0, y: 0 };
    this.speed = 0;
    this.ackSeq = 0;
    this.pending = [];
  }

  public setSpeed(speed: number): void {
    if (Number.isFinite(speed) && speed >= 0) this.speed = speed;
  }

  public setAuthoritative(position: Vector2, ackSeq: number): void {
    this.position = { ...position };
    this.ackSeq = Math.max(this.ackSeq, ackSeq);
    this.pending = this.pending.filter((input) => input.seq > this.ackSeq);
    this.replay();
  }

  /** Apply a held movement vector for one rendered frame. */
  public advance(move: Vector2, dtSeconds: number): void {
    if (!Number.isFinite(dtSeconds) || dtSeconds <= 0) return;
    this.apply(move, dtSeconds);
  }

  /** Record a discrete input for replay after reconciliation. */
  public recordInput(seq: number, move: Vector2, dtSeconds: number): void {
    if (!Number.isFinite(seq) || seq <= 0 || !Number.isFinite(dtSeconds) || dtSeconds <= 0) return;
    if (move.x === 0 && move.y === 0) return;
    this.pending.push({ seq, move: { ...move }, dt: dtSeconds });
    if (this.pending.length > 64) this.pending.splice(0, this.pending.length - 64);
  }

  public reconcile(position: Vector2, ackSeq: number): ReconcileResult {
    const before = { ...this.position };
    this.setAuthoritative(position, ackSeq);
    return {
      position: { ...this.position },
      correction: Math.hypot(this.position.x - before.x, this.position.y - before.y),
      replayed: this.pending.length,
    };
  }

  /**
   * Reconcile against the newest snapshot without teleporting on small errors.
   * Large errors (blocked movement, respawn, long loss) snap immediately.
   */
  public softReconcile(
    position: Vector2,
    ackSeq: number,
    snapThreshold = 2,
    smoothing = 0.25,
  ): ReconcileResult & { snapped: boolean } {
    const before = { ...this.position };
    const distance = Math.hypot(position.x - before.x, position.y - before.y);
    this.ackSeq = Math.max(this.ackSeq, ackSeq);
    this.pending = this.pending.filter((input) => input.seq > this.ackSeq);
    if (distance > snapThreshold) {
      this.position = { ...position };
    } else {
      const factor = Math.min(1, Math.max(0, smoothing));
      this.position = {
        x: before.x + (position.x - before.x) * factor,
        y: before.y + (position.y - before.y) * factor,
      };
    }
    return {
      position: { ...this.position },
      correction: distance,
      replayed: this.pending.length,
      snapped: distance > snapThreshold,
    };
  }

  public getPosition(): Vector2 {
    return { ...this.position };
  }

  public getAckSeq(): number {
    return this.ackSeq;
  }

  public pendingCount(): number {
    return this.pending.length;
  }

  private replay(): void {
    const position = { ...this.position };
    this.position = position;
    for (const input of this.pending) this.apply(input.move, input.dt);
  }

  private apply(move: Vector2, dtSeconds: number): void {
    const magnitude = Math.hypot(move.x, move.y);
    if (magnitude === 0 || this.speed <= 0) return;
    const scale = magnitude > 1 ? 1 / magnitude : 1;
    const distance = this.speed * dtSeconds;
    this.position = {
      x: this.position.x + move.x * scale * distance,
      y: this.position.y + move.y * scale * distance,
    };
  }
}
