const FAST_RETRY_CODES = new Set(["realtime_character_already_owned", "realtime_lease_stale"]);
const SLOW_RETRY_CODES = new Set(["realtime_join_rate_limited", "realtime_zone_full"]);
const FAST_BASE_MS = 300;
const FAST_MAX_MS = 2_000;
const FAST_MAX_ATTEMPTS = 7;
const SLOW_BASE_MS = 5_000;
const SLOW_MAX_MS = 15_000;
const SLOW_MAX_ATTEMPTS = 3;

type RetryMode = "fast" | "slow";

/** Owns the retry budget and backoff policy for world join failures. */
export class WorldJoinRetryPolicy {
  private attempt = 0;
  private mode: RetryMode = "fast";

  public reset(): void {
    this.attempt = 0;
    this.mode = "fast";
  }

  /** Returns the next delay, or null when the failure is not retryable/exhausted. */
  public nextDelay(code: string): number | null {
    const slow = SLOW_RETRY_CODES.has(code);
    if (!slow && !FAST_RETRY_CODES.has(code)) return null;

    const mode: RetryMode = slow ? "slow" : "fast";
    if (this.mode !== mode) {
      this.mode = mode;
      this.attempt = 0;
    }

    const baseMs = slow ? SLOW_BASE_MS : FAST_BASE_MS;
    const maxMs = slow ? SLOW_MAX_MS : FAST_MAX_MS;
    const maxAttempts = slow ? SLOW_MAX_ATTEMPTS : FAST_MAX_ATTEMPTS;
    if (this.attempt >= maxAttempts) return null;

    this.attempt += 1;
    return Math.min(baseMs * Math.pow(1.6, this.attempt - 1), maxMs);
  }
}
