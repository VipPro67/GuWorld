/**
 * Client-side realtime baseline collector (ticket #308).
 *
 * Aggregates one window of transport health samples in the Phaser runtime and
 * drains it periodically for upload to /api/realtime/metrics. Deliberately
 * allocation-light: it must never perturb the render loop it measures.
 */

export interface RealtimeMetricsWindow {
  window_started_at: number;
  window_duration_ms: number;
  zone: string;
  status: string;
  tick_rate: number;
  entities: number;
  snapshots: number;
  snapshot_rate_hz: number;
  snapshot_gap_avg_ms: number;
  snapshot_gap_p95_ms: number;
  snapshot_gap_p99_ms: number;
  snapshot_gap_max_ms: number;
  snapshot_gap_hist: number[];
  render_delay_avg_ms: number;
  hard_snaps: number;
  blend_count: number;
  latency_avg_ms: number;
  latency_max_ms: number;
  latency_samples: number;
  latency_clamped: number;
  jitter_ms: number;
  estimated_lost: number;
  stalls: number;
  messages_in: number;
  bytes_in: number;
  bytes_out: number;
  inputs_sent: number;
  resyncs: number;
  reconnects: number;
  stale_reconnects: number;
}

export interface RealtimeMetricEvent {
  kind: string;
  code?: string;
  at: number;
}

export interface RealtimeMetricsUpload {
  zone: string;
  status: string;
  network: string;
  user_agent: string;
  windows: RealtimeMetricsWindow[];
  events: RealtimeMetricEvent[];
}

export interface RealtimeMetricsLive {
  snapshots: number;
  rateHz: number;
  gapAvgMs: number;
  gapP95Ms: number;
  gapP99Ms: number;
  gapMaxMs: number;
  renderDelayAvgMs: number;
  latencyAvgMs: number;
  jitterMs: number;
  estimatedLost: number;
  stalls: number;
  hardSnaps: number;
  blendCount: number;
  bytesIn: number;
  bytesOut: number;
}

// Upper bounds in ms. The final bucket collects everything slower.
const GAP_BUCKETS_MS = [20, 40, 60, 80, 100, 150, 250, 500] as const;
const MAX_GAP_SAMPLES = 240;
const MAX_EVENTS = 24;
// Gaps beyond this are treated as stalls (tab suspended, reconnect, GC) and
// excluded from packet-loss/jitter math; a 5s tab pause must not read as 100
// lost snapshots.
const STALL_GAP_MS = 250;

export class RealtimeMetricsCollector {
  private windowStartMs = 0;
  private lastSnapshotAt = 0;
  private lastGapMs: number | null = null;
  private snapshots = 0;
  private gapsMs: number[] = [];
  private hist = new Array<number>(GAP_BUCKETS_MS.length).fill(0);
  private renderDelaySum = 0;
  private hardSnaps = 0;
  private blendCount = 0;
  private latencySum = 0;
  private latencyCount = 0;
  private latencyMax = 0;
  private latencyClamped = 0;
  private jitterSum = 0;
  private jitterCount = 0;
  private estimatedLost = 0;
  private stalls = 0;
  private maxGapMs = 0;
  private messagesIn = 0;
  private bytesIn = 0;
  private bytesOut = 0;
  private inputsSent = 0;
  private resyncs = 0;
  private reconnects = 0;
  private staleReconnects = 0;
  private entities = 0;
  private tickRate = 20;
  private events: RealtimeMetricEvent[] = [];

  public onSnapshot(input: {
    atMs: number;
    tick: number;
    tickRate: number;
    entities: number;
    renderDelayMs: number;
  }): void {
    const now = input.atMs;
    if (this.windowStartMs === 0) this.windowStartMs = now;
    this.tickRate = input.tickRate > 0 ? input.tickRate : this.tickRate;
    this.entities = Math.max(0, input.entities);
    this.snapshots += 1;
    this.renderDelaySum += Math.max(0, input.renderDelayMs);

    if (this.lastSnapshotAt > 0) {
      const gap = now - this.lastSnapshotAt;
      if (gap >= 0 && gap < 10_000) {
        this.gapsMs.push(gap);
        if (this.gapsMs.length > MAX_GAP_SAMPLES) this.gapsMs.shift();
        this.hist[this.gapBucket(gap)] += 1;
        if (gap > this.maxGapMs) this.maxGapMs = gap;
        if (gap > STALL_GAP_MS) {
          this.stalls += 1;
        } else {
          if (this.lastGapMs !== null) {
            this.jitterSum += Math.abs(gap - this.lastGapMs);
            this.jitterCount += 1;
          }
          this.lastGapMs = gap;
          const expected = 1000 / Math.max(1, this.tickRate);
          if (gap > expected * 1.5) {
            this.estimatedLost += Math.max(0, Math.round(gap / expected) - 1);
          }
        }
      }
    }
    this.lastSnapshotAt = now;
  }

  public onCorrection(snapped: boolean, correctionUnits: number, moving: boolean): void {
    if (snapped) {
      this.hardSnaps += 1;
    } else if (moving && correctionUnits > 0.02) {
      this.blendCount += 1;
    }
  }

  public onLatency(sampleMs: number, clamped: boolean): void {
    const value = Math.max(0, sampleMs);
    this.latencySum += value;
    this.latencyCount += 1;
    if (value > this.latencyMax) this.latencyMax = value;
    if (clamped) this.latencyClamped += 1;
  }

  public onTraffic(messagesIn: number, bytesIn: number, bytesOut: number): void {
    this.messagesIn += Math.max(0, messagesIn);
    this.bytesIn += Math.max(0, bytesIn);
    this.bytesOut += Math.max(0, bytesOut);
  }

  public onInput(): void {
    this.inputsSent += 1;
  }

  public onResync(): void {
    this.resyncs += 1;
  }

  public onReconnect(stale: boolean): void {
    this.reconnects += 1;
    if (stale) this.staleReconnects += 1;
  }

  public mark(kind: string, code?: string): void {
    this.events.push({ kind, code, at: Date.now() });
    if (this.events.length > MAX_EVENTS) this.events.shift();
  }

  /** Drains the current window; returns null when nothing was measured. */
  public drain(meta: { zone: string; status: string }): RealtimeMetricsUpload | null {
    const now = Date.now();
    const duration = this.windowStartMs > 0 ? Math.max(0, now - this.windowStartMs) : 0;
    const measured = this.snapshots > 0 || this.messagesIn > 0 || this.inputsSent > 0 || this.latencyCount > 0 || this.events.length > 0;
    if (!measured && duration < 1_000) return null;

    const gaps = this.gapsMs;
    let gapAvg = 0;
    if (gaps.length > 0) {
      let sum = 0;
      for (const gap of gaps) sum += gap;
      gapAvg = sum / gaps.length;
    }
    const window: RealtimeMetricsWindow = {
      window_started_at: this.windowStartMs,
      window_duration_ms: duration,
      zone: meta.zone,
      status: meta.status,
      tick_rate: this.tickRate,
      entities: this.entities,
      snapshots: this.snapshots,
      snapshot_rate_hz: round2(duration > 0 ? (this.snapshots * 1000) / duration : 0),
      snapshot_gap_avg_ms: round2(gapAvg),
      snapshot_gap_p95_ms: round2(percentile(gaps, 0.95)),
      snapshot_gap_p99_ms: round2(percentile(gaps, 0.99)),
      snapshot_gap_max_ms: round2(this.maxGapMs),
      snapshot_gap_hist: [...this.hist],
      render_delay_avg_ms: round2(this.snapshots > 0 ? this.renderDelaySum / this.snapshots : 0),
      hard_snaps: this.hardSnaps,
      blend_count: this.blendCount,
      latency_avg_ms: round2(this.latencyCount > 0 ? this.latencySum / this.latencyCount : 0),
      latency_max_ms: round2(this.latencyMax),
      latency_samples: this.latencyCount,
      latency_clamped: this.latencyClamped,
      jitter_ms: round2(this.jitterCount > 0 ? this.jitterSum / this.jitterCount : 0),
      estimated_lost: this.estimatedLost,
      stalls: this.stalls,
      messages_in: this.messagesIn,
      bytes_in: this.bytesIn,
      bytes_out: this.bytesOut,
      inputs_sent: this.inputsSent,
      resyncs: this.resyncs,
      reconnects: this.reconnects,
      stale_reconnects: this.staleReconnects,
    };
    const events = this.events;
    this.resetWindow();
    if (!measured) return null;
    return {
      zone: meta.zone,
      status: meta.status,
      network: detectNetworkType(),
      user_agent: typeof navigator !== "undefined" ? navigator.userAgent.slice(0, 96) : "",
      windows: [window],
      events,
    };
  }

  /** Non-destructive view for the F3 debug overlay. */
  public peek(): RealtimeMetricsLive {
    return {
      snapshots: this.snapshots,
      rateHz: round2(this.windowStartMs > 0 && this.snapshots > 0 ? (this.snapshots * 1000) / Math.max(1, Date.now() - this.windowStartMs) : 0),
      gapAvgMs: round2(this.gapsMs.length > 0 ? this.gapsMs.reduce((sum, gap) => sum + gap, 0) / this.gapsMs.length : 0),
      gapP95Ms: round2(percentile(this.gapsMs, 0.95)),
      gapP99Ms: round2(percentile(this.gapsMs, 0.99)),
      gapMaxMs: round2(this.maxGapMs),
      renderDelayAvgMs: round2(this.snapshots > 0 ? this.renderDelaySum / this.snapshots : 0),
      latencyAvgMs: round2(this.latencyCount > 0 ? this.latencySum / this.latencyCount : 0),
      jitterMs: round2(this.jitterCount > 0 ? this.jitterSum / this.jitterCount : 0),
      estimatedLost: this.estimatedLost,
      stalls: this.stalls,
      hardSnaps: this.hardSnaps,
      blendCount: this.blendCount,
      bytesIn: this.bytesIn,
      bytesOut: this.bytesOut,
    };
  }

  private resetWindow(): void {
    this.windowStartMs = 0;
    this.lastSnapshotAt = 0;
    this.lastGapMs = null;
    this.snapshots = 0;
    this.gapsMs = [];
    this.hist = new Array<number>(GAP_BUCKETS_MS.length).fill(0);
    this.renderDelaySum = 0;
    this.hardSnaps = 0;
    this.blendCount = 0;
    this.latencySum = 0;
    this.latencyCount = 0;
    this.latencyMax = 0;
    this.latencyClamped = 0;
    this.jitterSum = 0;
    this.jitterCount = 0;
    this.estimatedLost = 0;
    this.stalls = 0;
    this.maxGapMs = 0;
    this.messagesIn = 0;
    this.bytesIn = 0;
    this.bytesOut = 0;
    this.inputsSent = 0;
    this.resyncs = 0;
    this.reconnects = 0;
    this.staleReconnects = 0;
    this.events = [];
  }

  private gapBucket(gap: number): number {
    for (let i = 0; i < GAP_BUCKETS_MS.length; i += 1) {
      if (gap <= GAP_BUCKETS_MS[i]) return i;
    }
    return GAP_BUCKETS_MS.length - 1;
  }
}

function percentile(values: number[], ratio: number): number {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((left, right) => left - right);
  const index = Math.min(sorted.length - 1, Math.max(0, Math.floor((sorted.length - 1) * ratio)));
  return sorted[index];
}

function round2(value: number): number {
  return Math.round(value * 100) / 100;
}

interface NetworkInformationLike {
  type?: string;
  effectiveType?: string;
}

function detectNetworkType(): string {
  if (typeof navigator === "undefined") return "unknown";
  const connection = (navigator as Navigator & { connection?: NetworkInformationLike }).connection;
  if (!connection) return "unknown";
  return connection.type ?? connection.effectiveType ?? "unknown";
}
