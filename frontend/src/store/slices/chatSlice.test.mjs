import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";
import ts from "typescript";

// Execute the real slice with its HTTP and realtime dependencies replaced,
// without importing the application-wide Zustand store or browser runtime.
const source = await readFile(new URL("./chatSlice.ts", import.meta.url), "utf8");
const compiled = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
}).outputText;

function createHarness(request) {
  const requests = [];
  const events = [];
  const exports = {};
  vm.runInNewContext(compiled, {
    exports,
    console,
    require(name) {
      if (name === "@/lib/config") return { BACKEND_URL: "https://api.example" };
      if (name === "@/lib/auth") return {
        authFetch: async (...args) => {
          requests.push(args);
          return request(...args);
        },
      };
      if (name === "@/lib/realtime/UIRealtimeManager") return {
        uiRealtimeManager: {
          request: () => assert.fail("chat mutations must not depend on or retry over WS"),
        },
      };
      throw new Error(`Unexpected dependency: ${name}`);
    },
  });
  let state = {
    token: "token",
    beginAction: (key) => { events.push(["begin", key]); return true; },
    endAction: (key) => { events.push(["end", key]); },
    addLog: (message) => { events.push(["log", message]); },
    addToast: (message) => { events.push(["toast", message]); },
  };
  const set = (patch) => { state = { ...state, ...(typeof patch === "function" ? patch(state) : patch) }; };
  state = { ...state, ...exports.createChatSlice(set, () => state) };
  state.fetchBaoHoangMessages = async (channel) => { events.push(["history", channel]); };
  state.reconcilePlayer = async () => { events.push(["player"]); };
  return { state, requests, events };
}

test("chat sends once over HTTP and refreshes history and player without realtime", async () => {
  const { state, requests, events } = createHarness(async () => new Response(JSON.stringify({ message: "Sent" })));

  assert.equal(await state.sendBaoHoangChat("hello", "world"), true);
  assert.equal(requests.length, 1);
  assert.equal(requests[0][0], "https://api.example/api/chat/bao-hoang/send");
  assert.equal(requests[0][1].method, "POST");
  assert.equal(requests[0][1].headers["Content-Type"], "application/json");
  assert.deepEqual(JSON.parse(requests[0][1].body), { content: "hello", channel: "world" });
  assert.deepEqual(events, [["begin", "chat"], ["log", "Sent"], ["history", "world"], ["player"], ["end", "chat"]]);
});

test("a rejected chat reports server error without refresh or retry", async () => {
  const { state, requests, events } = createHarness(async () => new Response(JSON.stringify({ error: "Not enough stones" }), { status: 400 }));

  assert.equal(await state.sendBaoHoangChat("hello", "world"), false);
  assert.equal(requests.length, 1);
  assert.deepEqual(events, [["begin", "chat"], ["toast", "Not enough stones"], ["end", "chat"]]);
});

test("an ambiguous HTTP failure is not replayed as a second mutation", async () => {
  const { state, requests, events } = createHarness(async () => { throw new Error("connection lost"); });

  assert.equal(await state.sendBaoHoangChat("hello", "region"), false);
  assert.equal(requests.length, 1);
  assert.deepEqual(events.at(-1), ["end", "chat"]);
  assert.equal(events.some(([event]) => event === "history"), false);
});

test("refresh failures do not turn a confirmed send into a failed mutation", async () => {
  const { state, requests, events } = createHarness(async () => new Response("{}"));
  state.fetchBaoHoangMessages = async () => { throw new Error("history unavailable"); };
  state.reconcilePlayer = async () => { throw new Error("status unavailable"); };

  assert.equal(await state.sendBaoHoangChat("hello", "region"), true);
  assert.equal(requests.length, 1);
  assert.equal(events.some(([event]) => event === "toast"), false);
  assert.deepEqual(events.at(-1), ["end", "chat"]);
});

test("logged-out and already-running chat actions do not send requests", async () => {
  const { state, requests } = createHarness(async () => assert.fail("must not send"));
  state.token = null;
  assert.equal(await state.sendBaoHoangChat("hello", "world"), false);
  state.token = "token";
  state.beginAction = () => false;
  assert.equal(await state.sendBaoHoangChat("hello", "world"), false);
  assert.equal(requests.length, 0);
});
