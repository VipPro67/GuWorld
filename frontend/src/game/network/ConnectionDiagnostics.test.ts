import test from "node:test";
import assert from "node:assert/strict";
import { diagnoseConnectionFailure } from "./ConnectionDiagnostics.ts";

test("connection diagnosis distinguishes health, HTTP rejection and opaque browser failure", async () => {
  const original = globalThis.fetch;
  try {
    globalThis.fetch = async (input, options) => {
      assert.equal(String(input), "https://backend.example:8443/api/health");
      assert.equal(options?.credentials, "omit");
      return Response.json({ status: "ok" });
    };
    const result = await diagnoseConnectionFailure(new Error("Opening handshake failed"), "https://backend.example:8443/api/wt?token=secret&tab_id=private");
    assert.match(result, /health OK/);
    assert.match(result, /UDP 8443/);
    assert.ok(!result.includes("secret"));
    assert.ok(!result.includes("private"));
    globalThis.fetch = async () => new Response("denied", { status: 403 });
    assert.match(await diagnoseConnectionFailure(new Error("failed"), "https://backend.example:8443/api/wt"), /HTTP 403/);
    globalThis.fetch = async () => { throw new TypeError("Failed to fetch"); };
    const opaque = await diagnoseConnectionFailure(new Error("failed"), "https://backend.example:8443/api/wt");
    assert.match(opaque, /Chưa thể xác định/);
    globalThis.fetch = async () => new Response("<html>fallback</html>");
    assert.match(await diagnoseConnectionFailure(new Error("failed"), "https://backend.example/api/wt"), /không có status=ok/);
  } finally {
    globalThis.fetch = original;
  }
});
