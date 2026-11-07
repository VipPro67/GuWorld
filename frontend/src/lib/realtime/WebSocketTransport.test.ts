import assert from "node:assert/strict";
import test from "node:test";
import { WebSocketTransport } from "./WebSocketTransport.ts";

test("WebSocket fallback keeps world, UI and datagrams on one framed socket", async () => {
  const sent: Uint8Array[] = [];
  const socket = {
    binaryType: "blob" as BinaryType,
    readyState: 1,
    bufferedAmount: 0,
    onopen: null as ((event: Event) => void) | null,
    onerror: null as ((event: Event) => void) | null,
    onclose: null as ((event: CloseEvent) => void) | null,
    onmessage: null as ((event: MessageEvent) => void) | null,
    send(data: Uint8Array) { sent.push(data); },
    close() { this.onclose?.({ code: 1000, reason: "" } as CloseEvent); },
  };
  const transport = new WebSocketTransport("wss://example.test/api/ws", () => socket);
  socket.onopen?.({} as Event);
  await transport.ready;
  const ui = await transport.createBidirectionalStream();
  const world = await transport.createBidirectionalStream();
  const uiWriter = ui.writable.getWriter();
  const worldWriter = world.writable.getWriter();
  await uiWriter.write(new Uint8Array([2]));
  await worldWriter.write(new Uint8Array([1, 71, 87, 84, 2]));
  await transport.datagrams.writable.getWriter().write(new Uint8Array([9]));
  assert.deepEqual(sent.map((frame) => [...frame]), [[2, 1, 2], [1, 2, 1, 71, 87, 84, 2], [3, 9]]);
  socket.onmessage?.({ data: new Uint8Array([2, 1, 123]).buffer } as MessageEvent);
  socket.onmessage?.({ data: new Uint8Array([1, 2, 42]).buffer } as MessageEvent);
  socket.onmessage?.({ data: new Uint8Array([3, 7]).buffer } as MessageEvent);
  assert.deepEqual([...(await ui.readable.getReader().read()).value!], [123]);
  assert.deepEqual([...(await world.readable.getReader().read()).value!], [42]);
  const datagramReader = transport.datagrams.readable.getReader();
  assert.deepEqual([...(await datagramReader.read()).value!], [7]);
  for (let value = 0; value < 10; value++) {
    socket.onmessage?.({ data: new Uint8Array([3, value]).buffer } as MessageEvent);
  }
  const retained: number[] = [];
  for (let i = 0; i < 5; i++) retained.push((await datagramReader.read()).value![0]);
  assert.deepEqual(retained, [0, 6, 7, 8, 9]);
  await uiWriter.close();
  assert.deepEqual([...sent.at(-1)!], [4, 2, 1]);
  transport.close();
  assert.deepEqual(await transport.closed, { closeCode: 1000, reason: "" });
});
