import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { test, type TestContext } from "node:test";
import vm from "node:vm";
import ts from "typescript";
import { WorldTransport } from "../../game/network/WorldTransport.ts";
import {
  Mask,
  Op,
  STREAM_UI,
  STREAM_WORLD,
  Vital,
  encodeFrame,
  encodeSnapshotFull,
  encodeWelcome,
  quantize,
  type DeltaEntity,
  type SnapshotFull,
} from "../../game/network/v2/Codec.ts";
import { RealtimeUIManager } from "../realtime/UIRealtimeManager.ts";
import {
  CLOSE_CODE_SESSION_REPLACED,
  RealtimeClient,
  TabIdCoordinator,
  type BroadcastChannelLike,
  type WebTransportLike,
} from "./RealtimeClient.ts";
import type { WebTransportStreamLike } from "./UIStream.ts";

interface FakeStream {
  stream: WebTransportStreamLike;
  written: Uint8Array[];
  push(chunk: Uint8Array): void;
  pushText(text: string): void;
}

function createFakeStream(): FakeStream {
  let controller!: ReadableStreamDefaultController<Uint8Array>;
  const written: Uint8Array[] = [];
  const readable = new ReadableStream<Uint8Array>({
    start(c) {
      controller = c;
    },
  });
  const writable = new WritableStream<Uint8Array>({
    write(chunk) {
      written.push(chunk);
    },
  });
  return {
    stream: { readable, writable },
    written,
    push(chunk: Uint8Array) {
      controller.enqueue(chunk);
    },
    pushText(text: string) {
      controller.enqueue(new TextEncoder().encode(text));
    },
  };
}

function createFakeTransport() {
  const streams: FakeStream[] = [];
  const sentDatagrams: Uint8Array[] = [];
  let resolveClosed!: (info: { closeCode?: number; reason?: string }) => void;
  const closed = new Promise<{ closeCode?: number; reason?: string }>((resolve) => {
    resolveClosed = resolve;
  });
  let closeCalls = 0;

  const transport: WebTransportLike = {
    ready: Promise.resolve(),
    closed,
    datagrams: {
      readable: new ReadableStream<Uint8Array>({ start() {} }),
      writable: new WritableStream<Uint8Array>({
        write(chunk) {
          sentDatagrams.push(chunk);
        },
      }),
      maxDatagramSize: 1200,
    },
    async createBidirectionalStream() {
      const s = createFakeStream();
      streams.push(s);
      return s.stream;
    },
    close(info) {
      closeCalls += 1;
      resolveClosed(info ?? { closeCode: 0, reason: "" });
    },
  };

  return {
    transport,
    streams,
    sentDatagrams,
    getCloseCalls: () => closeCalls,
    serverClose(closeCode: number, reason: string) {
      resolveClosed({ closeCode, reason });
    },
  };
}

function sampleBaseline(): SnapshotFull {
  const state: DeltaEntity = {
    netId: 1, mask: Mask.Position | Mask.Vitals | Mask.Revision, hasExt: false, ext: 0,
    x: quantize(10), y: quantize(20), vitalsSub: Vital.HP | Vital.Mana, hp: 100, shield: 0,
    mana: 50, thought: 0, revision: 1, statuses: [], skillSlot: 0, castEndTick: 0,
    castWindup: 0, flags: 0, formKind: 0, formNetId: 0, modelScaleX100: 0,
    collisionRadiusX10: 0, currentFormHp: 0, formFlags: 0, aiState: 0, aggroNetId: 0,
    ownerNetId: 0, companionKind: "", stance: 0, radiusX10: 0, maxHp: 0, maxMana: 0,
  };
  return {
    tick: 10, baselineId: 1, baseTick: 10, ackSeq: 0, isBaseline: true,
    spawns: [{
      netId: 1, entityId: "player-1", name: "Tester", templateId: "", kind: 0,
      partyId: "", flags: 0, radiusX10: 5, maxHp: 100, maxMana: 50, maxThought: 0,
      ownerNetId: 0, companionKind: "", stance: 0,
    }],
    states: [state], removed: [], projectiles: [],
  };
}

test("RealtimeClient shares 1 WebTransport session across WorldTransport (0x01) and UIStream/uiRealtimeManager (0x02)", async () => {
  const createdTransports: ReturnType<typeof createFakeTransport>[] = [];
  const client = new RealtimeClient({
    url: "https://localhost:8443/api/wt",
    getToken: () => "jwt-token",
    getTabId: () => "tab-1",
    createTransport: () => {
      const f = createFakeTransport();
      createdTransports.push(f);
      return f.transport;
    },
  });

  let wsConstructed = 0;
  const originalWS = globalThis.WebSocket;
  globalThis.WebSocket = class {
    constructor() {
      wsConstructed += 1;
    }
  } as unknown as typeof WebSocket;

  try {
    const uiManager = new RealtimeUIManager({ realtimeClient: client });
    const chatEvents: unknown[] = [];
    uiManager.subscribe("chat:world", (data) => chatEvents.push(data));
    await uiManager.connect("jwt-token");

    assert.equal(wsConstructed, 0, "uiRealtimeManager must not construct WebSocket");
    assert.equal(createdTransports.length, 1, "must open 1 shared WebTransport session");
    assert.equal(createdTransports[0].streams.length, 1, "UI stream opened");
    assert.equal(createdTransports[0].streams[0].written[0][0], STREAM_UI);

    // Deliver UI event on stream #0.
    createdTransports[0].streams[0].pushText('{"type":"event","channel":"chat:world","data":{"msg":"chào"}}\n');
    await new Promise((r) => setTimeout(r, 5));
    assert.deepEqual(chatEvents, [{ msg: "chào" }]);

    // Now connect WorldTransport using the SAME RealtimeClient.
    const world = new WorldTransport({
      client,
      getToken: () => "jwt-token",
      getTabId: () => "tab-1",
    });
    const joinPromise = world.connect({ zone_id: "forest", map_version: "m1", content_version: "c1" });
    await new Promise((r) => setTimeout(r, 5));

    assert.equal(createdTransports.length, 1, "still only 1 WebTransport session for both UI and World");
    assert.equal(createdTransports[0].streams.length, 2, "second stream opened on shared session for world");
    assert.equal(createdTransports[0].streams[1].written[0][0], STREAM_WORLD);

    const welcomeBytes = encodeWelcome({
      sessionId: "s1", zoneId: "forest#1", mapVersion: "m1", contentVersion: "c1",
      tickRate: 20, movementSpeedX100: 500, serverTick: 10, entityId: "player-1",
      entityRevision: 1, baselineId: 1, mapHash: "", skillsJson: new Uint8Array(0),
      loadoutRevision: "", snapshot: encodeSnapshotFull(sampleBaseline()),
    });
    createdTransports[0].streams[1].push(encodeFrame(Op.Welcome, welcomeBytes));
    await joinPromise;
    assert.equal(world.isJoined(), true);
    assert.equal(client.getWorldRefCount(), 1);
    assert.equal(client.getUIRefCount(), 1);

    // Unmounting world (disconnecting WorldTransport) must NOT close the shared session or UI stream.
    world.disconnect();
    assert.equal(client.getWorldRefCount(), 0);
    assert.equal(client.getUIRefCount(), 1);
    assert.equal(createdTransports[0].getCloseCalls(), 0, "shared session stays open while UI holds ref");
    assert.equal(uiManager.isConnected(), true);

    uiManager.disconnect();
  } finally {
    globalThis.WebSocket = originalWS;
  }
});

test("RealtimeClient ref-count holds session for 30s grace after last release and cancels if re-acquired", async (t: TestContext) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const fake = createFakeTransport();
  const client = new RealtimeClient({
    url: "https://localhost:8443/api/wt",
    graceMs: 30_000,
    getToken: () => "jwt-token",
    getTabId: () => "tab-1",
    createTransport: () => fake.transport,
  });

  const releaseUI = client.acquireUI("jwt-token");
  const releaseWorld = client.acquireWorld("jwt-token");
  await Promise.resolve();
  assert.equal(client.getTotalRefCount(), 2);

  releaseWorld();
  assert.equal(client.getTotalRefCount(), 1);
  t.mock.timers.tick(60_000);
  assert.equal(fake.getCloseCalls(), 0, "must not close while UI ref > 0");

  releaseUI();
  assert.equal(client.getTotalRefCount(), 0);
  t.mock.timers.tick(20_000);
  assert.equal(fake.getCloseCalls(), 0, "must not close before 30s grace expires");

  // Re-acquiring before 30s cancels the grace close.
  const releaseAgain = client.acquireUI("jwt-token");
  t.mock.timers.tick(20_000);
  assert.equal(fake.getCloseCalls(), 0, "re-acquisition canceled pending grace timer");

  releaseAgain();
  t.mock.timers.tick(30_000);
  assert.equal(fake.getCloseCalls(), 1, "session closed after full 30s grace with 0 refs");
});

test("uiRealtimeManager rejects request when server returns error use HTTP so caller falls back to HTTP", async () => {
  const fake = createFakeTransport();
  const client = new RealtimeClient({
    url: "https://localhost:8443/api/wt",
    getToken: () => "jwt-token",
    getTabId: () => "tab-1",
    createTransport: () => fake.transport,
  });
  const uiManager = new RealtimeUIManager({ realtimeClient: client });
  await uiManager.connect("jwt-token");

  const uiStream = fake.streams[0];
  const reqPromise = uiManager.request<{ drops: string[] }>("world.drops", { x: 1, y: 2, radius: 3 });
  await new Promise((r) => setTimeout(r, 0));

  const sentCmd = JSON.parse(new TextDecoder().decode(uiStream.written.at(-1)!).trim()) as {
    type: string;
    channel: string;
    request_id: string;
  };
  assert.equal(sentCmd.type, "command");
  assert.equal(sentCmd.channel, "world.drops");
  assert.ok(sentCmd.request_id);

  uiStream.pushText(`${JSON.stringify({ type: "error", request_id: sentCmd.request_id, message: "use HTTP" })}\n`);

  // Caller pattern from useWorldStore / inventorySlice: catch and fallback to HTTP.
  let usedHttpFallback = false;
  const data = await reqPromise.catch(async (err: Error) => {
    assert.match(err.message, /use HTTP/);
    usedHttpFallback = true;
    return { drops: ["drop-from-http"] };
  });
  assert.equal(usedHttpFallback, true);
  assert.deepEqual(data, { drops: ["drop-from-http"] });

  uiManager.disconnect();
});

test("close code 4001 (session_replaced) marks client terminalStopped and fires onTabReplaced", async () => {
  const fake = createFakeTransport();
  let replacedCount = 0;
  const client = new RealtimeClient({
    url: "https://localhost:8443/api/wt",
    getToken: () => "jwt-token",
    getTabId: () => "tab-1",
    createTransport: () => fake.transport,
    onTabReplaced: () => {
      replacedCount += 1;
    },
  });

  await client.ensureSession("jwt-token");
  fake.serverClose(CLOSE_CODE_SESSION_REPLACED, "session_replaced");
  await new Promise((r) => setTimeout(r, 5));

  assert.equal(replacedCount, 1);
  assert.equal(client.isTerminalStopped(), true);
  await assert.rejects(client.ensureSession("jwt-token"), /đã bị dừng/);
});

test("TabIdCoordinator rotates duplicated tab_id via BroadcastChannel", () => {
  const peers = new Set<{ onmessage: ((event: { data: unknown }) => void) | null }>();
  const createChannel = (): BroadcastChannelLike => {
    const self = {
      onmessage: null as ((event: { data: unknown }) => void) | null,
      postMessage(data: unknown) {
        for (const peer of peers) {
          if (peer !== self) {
            peer.onmessage?.({ data });
          }
        }
      },
      close() {
        peers.delete(self);
      },
    };
    peers.add(self);
    return self;
  };

  let tab1Storage = "shared-tab-id";
  let tab2Storage = "shared-tab-id"; // Simulates Chrome "Duplicate tab" copying sessionStorage

  const tab1 = new TabIdCoordinator({
    instanceId: "instance-1",
    getTabId: () => tab1Storage,
    setTabId: (id) => { tab1Storage = id; },
    generateId: () => "rotated-1",
    createBroadcastChannel: createChannel,
  });
  assert.equal(tab1.ensureTabId(), "shared-tab-id");

  const tab2 = new TabIdCoordinator({
    instanceId: "instance-2",
    getTabId: () => tab2Storage,
    setTabId: (id) => { tab2Storage = id; },
    generateId: () => "rotated-tab-2",
    createBroadcastChannel: createChannel,
  });
  assert.equal(tab2.ensureTabId(), "rotated-tab-2", "duplicated tab must rotate its tab_id");
  assert.equal(tab1Storage, "shared-tab-id", "original tab keeps its tab_id");
  assert.equal(tab2Storage, "rotated-tab-2");

  tab1.close();
  tab2.close();
});

test("useWorldStore.fetchNearbyCells and inventorySlice.fetchGroundDrops fall back to HTTP when uiRealtimeManager is active", async (t: TestContext) => {
  const fake = createFakeTransport();
  const client = new RealtimeClient({
    url: "https://localhost:8443/api/wt",
    getToken: () => "jwt-token",
    getTabId: () => "tab-1",
    createTransport: () => fake.transport,
  });
  const uiManager = new RealtimeUIManager({ realtimeClient: client });
  t.after(() => {
    uiManager.disconnect();
    client.disconnect();
  });
  await uiManager.connect("jwt-token");
  const uiStream = fake.streams[0];

  // Auto-reply "error: use HTTP" whenever a command frame is written to the UI stream.
  const origRequest = uiManager.request.bind(uiManager);
  uiManager.request = (<T>(channel: string, payload?: unknown, timeoutMs?: number): Promise<T> => {
    const p = origRequest<T>(channel, payload, timeoutMs);
    setTimeout(() => {
      const lastRaw = uiStream.written.at(-1);
      if (!lastRaw) return;
      const parsed = JSON.parse(new TextDecoder().decode(lastRaw).trim()) as { type?: string; request_id?: string };
      if (parsed.type === "command" && parsed.request_id) {
        uiStream.pushText(`${JSON.stringify({ type: "error", request_id: parsed.request_id, message: "use HTTP" })}\n`);
      }
    }, 0);
    return p;
  }) as typeof uiManager.request;

  const httpUrls: string[] = [];
  const fakeAuthFetch = async (url: string) => {
    httpUrls.push(url);
    if (url.includes("/api/explore/cells/rect")) {
      return new Response(JSON.stringify([{ x: 5, y: 6, cell_type: "Forest", content: "Rừng" }]), { status: 200 });
    }
    if (url.includes("/api/world/drops")) {
      return new Response(JSON.stringify({ drops: [{ id: "drop-1", x: 5, y: 6 }] }), { status: 200 });
    }
    return new Response("{}", { status: 404 });
  };

  // 1. Test useWorldStore.fetchNearbyCells
  const worldStoreSource = await readFile(new URL("../../store/useWorldStore.ts", import.meta.url), "utf8");
  const worldStoreCompiled = ts.transpileModule(worldStoreSource, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText;
  let worldState: Record<string, unknown> = {};
  const worldExports: Record<string, unknown> = {};
  vm.runInNewContext(worldStoreCompiled, {
    exports: worldExports,
    console,
    encodeURIComponent,
    require(name: string) {
      if (name === "zustand") {
        return {
          create: (initializer: (set: (patch: Record<string, unknown>) => void, get: () => Record<string, unknown>) => Record<string, unknown>) => {
            const set = (patch: Record<string, unknown>) => { worldState = { ...worldState, ...patch }; };
            worldState = initializer(set, () => worldState);
            return worldState;
          },
        };
      }
      if (name === "@/lib/config") return { BACKEND_URL: "https://api.example" };
      if (name === "@/lib/auth") return { authFetch: fakeAuthFetch };
      if (name === "@/store/useGameStore") return { useGameStore: { getState: () => ({}) } };
      if (name === "@/lib/realtime/UIRealtimeManager") return { uiRealtimeManager: uiManager };
      throw new Error(`Unexpected dependency: ${name}`);
    },
  });

  const ok = await (worldState.fetchNearbyCells as (x: number, y: number, r: number) => Promise<boolean>)(5, 6, 2);
  assert.equal(ok, true);
  assert.equal((worldState.nearbyCells as unknown[]).length, 1);
  assert.ok(httpUrls.some((u) => u.includes("/api/explore/cells/rect")));

  // 2. Test inventorySlice.fetchGroundDrops
  const invSource = await readFile(new URL("../../store/slices/inventorySlice.ts", import.meta.url), "utf8");
  const invCompiled = ts.transpileModule(invSource, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  }).outputText;
  let invState: Record<string, unknown> = { groundDrops: [] };
  const invExports: Record<string, unknown> = {};
  vm.runInNewContext(invCompiled, {
    exports: invExports,
    console,
    require(name: string) {
      if (name === "@/lib/config") return { BACKEND_URL: "https://api.example" };
      if (name === "@/lib/auth") return { authFetch: fakeAuthFetch };
      if (name === "@/lib/realtime/UIRealtimeManager") return { uiRealtimeManager: uiManager };
      if (name === "@/lib/gu-index") return { guAt: () => null };
      if (name === "@/lib/player-reconciliation") return { LatestRequestGate: class {}, shouldApplyPlayerRevision: () => true };
      return {};
    },
  });
  const createInventorySlice = invExports.createInventorySlice as (
    set: (patch: Record<string, unknown>) => void,
    get: () => Record<string, unknown>,
  ) => Record<string, unknown>;
  const setInv = (patch: Record<string, unknown>) => { invState = { ...invState, ...patch }; };
  invState = { ...invState, ...createInventorySlice(setInv, () => invState) };

  await (invState.fetchGroundDrops as (vp: { x: number; y: number; radius: number }) => Promise<void>)({ x: 5, y: 6, radius: 3 });
  assert.deepEqual(invState.groundDrops, [{ id: "drop-1", x: 5, y: 6 }]);
  assert.ok(httpUrls.some((u) => u.includes("/api/world/drops?x=5&y=6&radius=3")));

  uiManager.disconnect();
});
