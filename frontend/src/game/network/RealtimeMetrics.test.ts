import assert from "node:assert/strict";
import test from "node:test";
import { RealtimeMetricsCollector } from "./RealtimeMetrics.ts";

test("collector derives gap stats, jitter and estimated loss from snapshot arrival", () => {
  const collector = new RealtimeMetricsCollector();
  const base = 1_000;
  collector.onSnapshot({ atMs: base, tick: 100, tickRate: 20, entities: 5, renderDelayMs: 40 });
  collector.onSnapshot({ atMs: base + 50, tick: 101, tickRate: 20, entities: 5, renderDelayMs: 45 });
  collector.onSnapshot({ atMs: base + 150, tick: 103, tickRate: 20, entities: 6, renderDelayMs: 55 });

  const upload = collector.drain({ zone: "Southern_Border", status: "joined" });
  assert.ok(upload);
  const window = upload.windows[0];
  assert.equal(window.snapshots, 3);
  assert.equal(window.tick_rate, 20);
  assert.equal(window.entities, 6);
  assert.ok(window.snapshot_gap_p95_ms >= 50);
  assert.equal(window.estimated_lost, 1, "100ms gap at 20Hz loses one snapshot");
  assert.ok(window.jitter_ms > 0);
  assert.equal(window.render_delay_avg_ms, 46.67);
  assert.equal(upload.zone, "Southern_Border");
});

test("collector separates stalls from packet loss and jitter", () => {
  const collector = new RealtimeMetricsCollector();
  const base = 2_000;
  collector.onSnapshot({ atMs: base, tick: 200, tickRate: 20, entities: 2, renderDelayMs: 10 });
  collector.onSnapshot({ atMs: base + 50, tick: 201, tickRate: 20, entities: 2, renderDelayMs: 10 });
  collector.onSnapshot({ atMs: base + 3_050, tick: 260, tickRate: 20, entities: 2, renderDelayMs: 10 });

  const upload = collector.drain({ zone: "z", status: "joined" });
  assert.ok(upload);
  const window = upload.windows[0];
  assert.equal(window.stalls, 1, "3s gap is a stall, not loss");
  assert.equal(window.estimated_lost, 0, "stall must not be counted as lost snapshots");
  assert.equal(window.snapshot_gap_max_ms, 3000);
  assert.equal(window.jitter_ms, 0, "stall gap is excluded from jitter");
  assert.equal(window.latency_samples, 0);
});

test("collector counts hard snaps and blends without inflating both", () => {
  const collector = new RealtimeMetricsCollector();
  collector.onCorrection(true, 4.2, true);
  collector.onCorrection(false, 0.4, true);
  collector.onCorrection(false, 0.001, true);

  const live = collector.peek();
  assert.equal(live.hardSnaps, 1);
  assert.equal(live.blendCount, 1);
});

test("collector accumulates traffic and resets the window after drain", () => {
  const collector = new RealtimeMetricsCollector();
  collector.onTraffic(3, 900, 120);
  collector.onInput();
  collector.onLatency(30, false);
  collector.onLatency(250, true);
  collector.onResync();
  collector.onReconnect(true);
  collector.mark("join_retry", "realtime_zone_full");

  const first = collector.drain({ zone: "z", status: "joined" });
  assert.ok(first);
  assert.equal(first.windows[0].messages_in, 3);
  assert.equal(first.windows[0].bytes_in, 900);
  assert.equal(first.windows[0].bytes_out, 120);
  assert.equal(first.windows[0].inputs_sent, 1);
  assert.equal(first.windows[0].latency_max_ms, 250);
  assert.equal(first.windows[0].latency_clamped, 1);
  assert.equal(first.windows[0].resyncs, 1);
  assert.equal(first.windows[0].reconnects, 1);
  assert.equal(first.windows[0].stale_reconnects, 1);
  assert.equal(first.events.length, 1);
  assert.equal(first.events[0].kind, "join_retry");

  const second = collector.drain({ zone: "z", status: "joined" });
  assert.equal(second, null, "an empty window is not uploaded");
  assert.equal(collector.peek().latencyAvgMs, 0);
  assert.equal(collector.peek().hardSnaps, 0);
});
