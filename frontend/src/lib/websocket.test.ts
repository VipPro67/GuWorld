import assert from "node:assert/strict";
import { test, type TestContext } from "node:test";
import { WebSocketManager, type WebSocketConnectionState } from "./websocket.ts";

class FakeSocket {
  static OPEN = 1;
  static sockets: FakeSocket[] = [];
  readyState = 0;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  sent: string[] = [];
  url: string;
  constructor(url: string) { this.url = url; FakeSocket.sockets.push(this); }
  open() { this.readyState = 1; this.onopen?.(); }
  close() { this.readyState = 3; this.onclose?.(); }
  send(value: string) { this.sent.push(value); }
  receive(value: unknown) { this.onmessage?.({ data: JSON.stringify(value) }); }
}

function setup(t: TestContext) {
  FakeSocket.sockets = [];
  t.mock.timers.enable({ apis: ["setTimeout", "setInterval"] });
  const original = globalThis.WebSocket;
  globalThis.WebSocket = FakeSocket as unknown as typeof WebSocket;
  const manager = new WebSocketManager("wss://example.test/api/ws");
  t.after(() => { manager.disconnect(); globalThis.WebSocket = original; });
  return manager;
}
function lastSocket() { return FakeSocket.sockets.at(-1)!; }

test("close before open rejects and retries with registered channels", async (t) => {
  const manager = setup(t);
  const states: WebSocketConnectionState[] = [];
  manager.onConnectionChange((state) => states.push(state));
  const messages: unknown[] = [];
  manager.subscribe("player", (data) => messages.push(data));
  const failed = assert.rejects(manager.connect("token"), /đã đóng/);
  lastSocket().close();
  await failed;
  t.mock.timers.tick(1000);
  assert.equal(FakeSocket.sockets.length, 2);
  lastSocket().open();
  lastSocket().receive({ channel: "player", data: { hp: 42 } });
  assert.deepEqual(messages, [{ hp: 42 }]);
  assert.equal(JSON.parse(lastSocket().sent[0]).channel, "player");
  assert.deepEqual(states, ["connecting", "disconnected", "connecting", "connected"]);
  assert.equal(manager.isConnected(), true);
});

test("handshake timeout permits recovery and ignores obsolete socket events", async (t) => {
  const manager = setup(t);
  const failed = assert.rejects(manager.connect("token"), /connect timeout/);
  const oldSocket = lastSocket();
  t.mock.timers.tick(10000);
  await failed;
  assert.equal(oldSocket.readyState, 3);
  const connected = manager.connect("token");
  oldSocket.open();
  assert.equal(manager.getConnectionState(), "connecting");
  lastSocket().open();
  await connected;
  t.mock.timers.tick(1000);
  assert.equal(FakeSocket.sockets.length, 2);
});

test("socket error rejects without requiring a close event", async (t) => {
  const manager = setup(t);
  const failed = assert.rejects(manager.connect("token"), /Không thể kết nối/);
  lastSocket().onerror?.();
  await failed;
  assert.equal(manager.getConnectionState(), "disconnected");
});

test("pending commands reject immediately and subscriptions survive recovery", async (t) => {
  const manager = setup(t);
  const opened: boolean[] = [];
  manager.onOpen(({ reconnected }) => opened.push(reconnected));
  manager.subscribe("trade", () => {});
  const connected = manager.connect("token");
  lastSocket().open();
  await connected;
  const failed = assert.rejects(manager.request("world.cells_rect"), /đã đóng/);
  lastSocket().close();
  await failed;
  t.mock.timers.tick(1000);
  lastSocket().open();
  assert.deepEqual(opened, [false, true]);
  assert.equal(JSON.parse(lastSocket().sent[0]).channel, "trade");
});

test("manual disconnect settles handshake and commands and prevents retry", async (t) => {
  const manager = setup(t);
  const failedConnect = assert.rejects(manager.connect("token"), /đã đóng/);
  manager.disconnect();
  await failedConnect;
  t.mock.timers.tick(60000);
  assert.equal(FakeSocket.sockets.length, 1);
  const connected = manager.connect("token");
  lastSocket().open();
  await connected;
  const failedCommand = assert.rejects(manager.request("world.drops"), /đã đóng/);
  manager.disconnect();
  await failedCommand;
  t.mock.timers.tick(60000);
  assert.equal(FakeSocket.sockets.length, 2);
  assert.equal(manager.getConnectionState(), "disconnected");
});

test("failed acquisition retains ownership and release is idempotent across sessions", async (t) => {
  const manager = setup(t);
  t.mock.method(console, "warn", () => {});
  const release = manager.acquireConnection("token");
  lastSocket().close();
  await Promise.resolve();
  assert.equal(manager.getRefCount(), 1);
  release();
  release();
  assert.equal(manager.getRefCount(), 0);
  const oldRelease = manager.acquireConnection("token");
  manager.disconnect();
  const newRelease = manager.acquireConnection("new-token");
  oldRelease();
  assert.equal(manager.getRefCount(), 1);
  newRelease();
});

test("automatic retries are bounded and explicit recovery remains possible", async (t) => {
  const manager = setup(t);
  t.mock.method(console, "error", () => {});
  const failed = assert.rejects(manager.connect("token"));
  lastSocket().close();
  await failed;
  for (let attempt = 0; attempt < 10; attempt++) {
    t.mock.timers.tick(30000);
    lastSocket().close();
    await Promise.resolve();
  }
  assert.equal(FakeSocket.sockets.length, 11);
  t.mock.timers.tick(60000);
  assert.equal(FakeSocket.sockets.length, 11);
  const connected = manager.connect("token");
  lastSocket().open();
  await connected;
  assert.equal(manager.isConnected(), true);
});

test("online retry restarts an active exhausted connection but never revives a logged out session", async (t) => {
  const manager = setup(t);
  t.mock.method(console, "warn", () => {});
  t.mock.method(console, "error", () => {});
  manager.acquireConnection("token");
  lastSocket().close();
  await Promise.resolve();
  for (let attempt = 0; attempt < 10; attempt++) {
    t.mock.timers.tick(30000);
    lastSocket().close();
    await Promise.resolve();
  }
  assert.equal(FakeSocket.sockets.length, 11);
  manager.retryConnection();
  assert.equal(FakeSocket.sockets.length, 12);
  lastSocket().close();
  await Promise.resolve();
  t.mock.timers.tick(1000);
  assert.equal(FakeSocket.sockets.length, 13, "online retry renews the reconnect budget");
  manager.disconnect();
  manager.retryConnection();
  t.mock.timers.tick(60000);
  assert.equal(FakeSocket.sockets.length, 13);
});

test("heartbeat loss rejects pending commands and transitions to degraded state", async (t) => {
  const manager = setup(t);
  t.mock.method(console, "warn", () => {});
  const observedConnectivity: boolean[] = [];
  manager.onConnectionChange((state) => {
    if (state === "disconnected") observedConnectivity.push(manager.isConnected());
  });
  const connected = manager.connect("token");
  lastSocket().open();
  await connected;
  t.mock.timers.tick(30000);
  const failed = assert.rejects(manager.request("world.cells_rect"), /hết phản hồi/);
  t.mock.timers.tick(10000);
  await failed;
  assert.equal(manager.getConnectionState(), "disconnected");
  assert.equal(manager.isConnected(), false);
  assert.deepEqual(observedConnectivity, [false], "degraded listeners must see the socket as disconnected immediately");
});
