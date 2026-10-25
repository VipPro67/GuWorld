/**
 * Maps wall-clock time onto server ticks. The server remains the only source of
 * truth; this clock is used to choose interpolation points between snapshots.
 */
const fallbackNow = (): number =>
  typeof performance !== "undefined" ? performance.now() : Date.now();

export class WorldClock {
  private anchorTick = 0;
  private anchorAt = 0;
  private tickRate = 20;
  private synced = false;

  public sync(serverTick: number, tickRate: number, receivedAt = fallbackNow()): void {
    if (!Number.isFinite(serverTick) || serverTick < 0) return;
    if (!Number.isFinite(tickRate) || tickRate <= 0 || tickRate > 120) return;
    this.anchorTick = serverTick;
    this.anchorAt = receivedAt;
    this.tickRate = tickRate;
    this.synced = true;
  }

  public estimatedTick(now = fallbackNow()): number {
    if (!this.synced) return 0;
    const elapsed = Math.max(0, now - this.anchorAt) / 1000;
    return this.anchorTick + elapsed * this.tickRate;
  }

  public getTickRate(): number {
    return this.tickRate;
  }

  public isSynced(): boolean {
    return this.synced;
  }

  public reset(): void {
    this.anchorTick = 0;
    this.anchorAt = 0;
    this.tickRate = 20;
    this.synced = false;
  }
}
