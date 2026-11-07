import assert from "node:assert/strict";
import test from "node:test";
import { STREAM_UI } from "../../game/network/v2/Codec.ts";
import { UIStream, type UIStreamMessage } from "./UIStream.ts";

function createFakeBiDiStream() {
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
    pushText(text: string) {
      controller.enqueue(new TextEncoder().encode(text));
    },
    closeReadable() {
      controller.close();
    },
  };
}

test("UIStream writes 0x02 header and exchanges NDJSON frames across split chunks", async () => {
  const fake = createFakeBiDiStream();
  const ui = new UIStream(fake.stream);
  const received: UIStreamMessage[] = [];
  ui.onMessage((msg) => received.push(msg));

  await ui.open();
  assert.equal(ui.isOpen(), true);
  assert.equal(fake.written.length, 1);
  assert.deepEqual(Array.from(fake.written[0]), [STREAM_UI]);

  ui.subscribe("chat:world");
  ui.ping();
  ui.unsubscribe("chat:world");
  await new Promise((r) => setTimeout(r, 0));

  const decodeWritten = (idx: number) => JSON.parse(new TextDecoder().decode(fake.written[idx]).trim());
  assert.deepEqual(decodeWritten(1), { type: "subscribe", channel: "chat:world", payload: { channel: "chat:world" } });
  assert.deepEqual(decodeWritten(2), { type: "ping" });
  assert.deepEqual(decodeWritten(3), { type: "unsubscribe", channel: "chat:world", payload: { channel: "chat:world" } });

  // Push partial NDJSON chunks to verify line reassembly.
  fake.pushText('{"type":"event","channel":"chat:world",');
  assert.equal(received.length, 0);
  fake.pushText('"data":{"text":"xin chào"}}\n{"type":"pong"}\n');
  await new Promise((r) => setTimeout(r, 5));

  assert.equal(received.length, 2);
  assert.equal(received[0].channel, "chat:world");
  assert.deepEqual(received[0].data, { text: "xin chào" });
  assert.equal(received[1].type, "pong");

  ui.close();
  assert.equal(ui.isOpen(), false);
});
