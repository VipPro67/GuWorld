# WT-306 — Ngữ nghĩa WebTransport phía browser (datagram, migration, auth)

> Ticket: [#306](https://github.com/vippro67/GuWorld/issues/306) · Map: #303 (WebTransport thay thế WebSocket)
> Ngày khảo sát: 2026-11-02. Chrome stable lúc khảo sát: **156.0.8078.80** (`chromiumdash`).
> Mọi kết luận dưới đây đọc từ **spec + source code triển khai + compat data**, không đọc từ blog tóm tắt.
> Chỗ nào không xác minh được thì ghi rõ **UNVERIFIED**.

## 0. TL;DR — 6 điều làm thay đổi thiết kế protocol

| # | Kết luận | Ảnh hưởng |
|---|----------|-----------|
| 1 | Datagram **không bao giờ được fragment** (cả QUIC lẫn browser). Vượt `maxDatagramSize` → **im lặng bị bỏ**, promise vẫn resolve. | Phải tự assert size trước khi write, hoặc mất snapshot mà không có lỗi nào. |
| 2 | `maxDatagramSize` **khác nhau giữa browser**: Safari hard-code **1024**, Chrome trả giá trị negotiated **~1200**. | Ngân sách payload chung phải là **1024**, không phải 1200. |
| 3 | **Cookie KHÔNG được gửi.** Spec đặt `credentials mode = "omit"`; Chrome/WebKit không có cookie jar trong handshake. | Cookie `gu_auth_token` trên `.cogioi.io.vn` **vô dụng** cho WebTransport. JWT trong `?token=` vẫn là đường portable. |
| 4 | Custom request header **đã có** (`WebTransportOptions.headers`) — Chrome 156/157+, Safari 26.4+, **Firefox: không**. | Không dùng header làm auth chính. Chỉ là đường nâng cấp sau này. |
| 5 | **Connection migration: KHÔNG hoạt động thực tế** trên Chrome desktop (feature chỉ bật mặc định trên Android) và WebTransport dùng QUIC client riêng, không có network-change observer. | WiFi → 4G = **mất session**, phải reconnect + resume. Không được thiết kế "session sống mãi". |
| 6 | Safari có **fallback HTTP/2** ⇒ `ready` có thể resolve mà **không có datagram**. | Phải check `transport.reliability` sau `ready`, không được giả định datagram tồn tại. |

---

## 1. Datagram

### 1.1 Kích thước

- Một datagram = **một QUIC DATAGRAM frame = một UDP datagram**. Không có fragmentation ở bất kỳ tầng nào:
  > "DATAGRAM frames cannot be fragmented; therefore, application protocols need to handle cases where the maximum datagram size is limited by other factors." — [RFC 9221 §4](https://www.rfc-editor.org/rfc/rfc9221.html#section-4)
- API công bố giới hạn qua `datagrams.maxDatagramSize` (Chrome 97, Firefox 114, Safari 26.4 — [BCD](https://github.com/mdn/browser-compat-data/blob/main/api/WebTransportDatagramDuplexStream.json)).
- Chrome **plumb giá trị negotiated** từ tầng QUIC: `services/network/web_transport.cc` truyền `transport_->GetMaxDatagramSize()` xuống renderer; quiche tính `GetGuaranteedLargestDatagramPayload() - varint(quarterStreamId)` ([`quic_spdy_stream.cc:1847`](https://github.com/google/quiche/blob/main/quiche/quic/core/http/quic_spdy_stream.cc#L1847)). Đo thực tế trên Chrome 155: **1211 byte** ([nguồn đo](https://www.irurueta.com/web/webrtc-websockets/webtransport-and-media-over-quic.html)).
- **Safari hard-code 1024**: `unsigned WebTransportDatagramDuplexStream::maxDatagramSize() const { … return 1024; }` ([WebKit main](https://github.com/WebKit/WebKit/blob/main/Source/WebCore/Modules/webtransport/WebTransportDatagramDuplexStream.cpp#L51)), kèm FIXME chờ `rdar://182433814`.
- Suy ra ngân sách an toàn: **≤ 1024 byte** payload ứng dụng (đã trừ overhead). Đừng dùng 1200: 1200 là **UDP payload tối thiểu mà QUIC bảo đảm** (`max_udp_payload_size ≥ 1200`, RFC 9000 §18.2), trừ tiếp header QUIC (~21–41B), frame type (1B) và quarter-stream-id (1–8B) mới ra payload.

### 1.2 Vượt ngưỡng thì sao

- Spec: nếu `[[OutgoingMaxDatagramSize]] < byteLength` thì **return a promise resolved with undefined** — tức là **không gửi, không throw, không fragment** ([spec §4.3 step 4](https://w3c.github.io/webtransport/#datagram-writable)).
- Chromium: CL `WebTransport: Implement Datagram writable streams` ghi rõ "plumbs the negotiated maximum and **silently discards oversized writes**". Ở network service, `SendOrQueueDatagram()` trả status `kTooBig` → callback `false`, JS chỉ thấy write resolve.
- Hệ quả: **không có exception nào để bắt**. Nếu quên check size, 100% snapshot oversized biến mất im lặng, và triệu chứng duy nhất là "client đứng hình".

### 1.3 Không có `datagrams.writeable` (tiền đề trong ticket sai)

- Không tồn tại property boolean `writeable` trong WebTransport API (không có trong [IDL Chromium](https://github.com/chromium/chromium/blob/main/third_party/blink/renderer/modules/webtransport/web_transport_datagram_duplex_stream.idl) lẫn [WebKit](https://github.com/WebKit/WebKit/blob/main/Source/WebCore/Modules/webtransport/WebTransportDatagramDuplexStream.idl)).
- Backpressure thật nằm ở `WritableStream`:
  - `writer.ready` / `writer.desiredSize` — **ready nghĩa là "còn chỗ trong queue", KHÔNG phải "đã gửi được"** (spec note, §4.3).
  - `datagrams.incomingMaxBufferedDatagrams` / `outgoingMaxBufferedDatagrams` (Chrome **151+**, Safari "preview", Firefox không).
  - `incomingMaxAge` / `outgoingMaxAge` — **hữu ích trực tiếp cho snapshot 20 Hz**: đặt `outgoingMaxAge ≈ 50–100 ms` để datagram cũ tự bị hủy thay vì gửi state đã lỗi thời.
  - `incomingHighWaterMark` / `outgoingHighWaterMark` đã deprecate (Chromestatus: "Deprecate WebTransport incomingHighWaterMark/outgoingHighWaterMark").

### 1.4 Hành vi drop (drop behaviour)

- Phía gửi, Chrome network service: tối đa **4096 datagram / 16 MiB** pending (`kMaxPendingDatagrams`, `kMaxPendingDatagramBytes`, `services/network/web_transport.h`); quá hạn `outgoingMaxAge` → drop.
- Phía nhận: queue bị cắt từ **head** khi vượt `incomingMaxBufferedDatagrams` (spec §5.4) — tức là **mất frame cũ, giữ frame mới**: đúng ý cho latest-wins snapshots.
- Phía server quic-go: send queue **32**, recv queue **128** datagram (`datagram_queue.go`) — nhỏ hơn Chrome; burst > 32 datagram sẽ bị drop ở server.
- Datagram **có** chịu congestion control của QUIC → ghi nhanh hơn mạng cho phép sẽ queue/drop tại local chứ không làm nghẽn path.

---

## 2. Streams

### 2.1 Uni vs bi

- `createUnidirectionalStream()` → `WritableStream` (một chiều client→server). Muốn server→client phải để **server mở** uni stream, client đọc qua `incomingUnidirectionalStreams`.
- `createBidirectionalStream()` → cặp readable/writable.
- **Không có API "server push datagram"** — datagram là 2 chiều trên cùng session, không cần stream.

### 2.2 Chi phí mở stream

- Mở stream **không tốn RTT**: chỉ cấp 1 stream ID + gửi STREAM frame. Nhưng **phụ thuộc credit `MAX_STREAMS` của peer**; khi hết credit, promise của `createUnidirectionalStream()` **treo** cho tới khi peer cấp thêm.
- Giới hạn mặc định phía server quic-go: **100 uni + 100 bidi** (`DefaultMaxIncomingUniStreams/Streams`, `internal/protocol/params.go`), flow-control stream 512 KB, connection window lớn hơn.
- Browser: `anticipatedConcurrentIncomingUnidirectionalStreams/BidirectionalStreams` (Safari 26.4, Chrome 158) cho phép khai báo trước để khỏi tốn RTT.

### 2.3 `sendOrder` / `sendGroup`

- Chỉ tồn tại trên `WebTransportSendStream` / `WebTransportDatagramsWritable` **mới**:
  - Chrome: `WebTransportSendGroup` + `WebTransportDatagramsWritable` **experimental, milestone 159** → **không có** trong stable 156. `createUnidirectionalStream()` ở Chrome ≤158 trả `WritableStream` thuần (Chromium IDL còn TODO đổi sang `WebTransportSendStream`).
  - Safari 26.4: có `createSendGroup()` và `sendOrder` trong IDL.
  - Firefox: `createSendGroup` 155 nhưng `createWritable` 155, `responseHeaders` không có.
- Kết luận: **v1 không dùng `sendOrder`**. Không portable, và 20 Hz snapshot không cần priority trong 1 session.

### 2.4 Reliable stream làm kênh event low-rate?

**Có — và nên làm, nhưng là MỘT stream sống lâu, không phải một stream mỗi event.**

- Bằng chứng định lượng: WebKit bug [317084](https://bugs.webkit.org/show_bug.cgi?id=317084) (06/2026) — pattern "một incoming uni stream cho mỗi frame" làm Safari **deadlock flow-control ở mốc 16 MiB** (`connection receive window is 0`), session vẫn "connected", `closed`/`draining` treo vĩnh viễn. Trong cùng báo cáo: **một uni stream dài 92 MB chạy tốt**. Đây là rủi ro Safari-specific lớn nhất và tránh được bằng thiết kế.
- Chi phí: 1 stream ID + framing length-prefix; không HOL giữa các stream, nhưng **trong** một stream thì vẫn HOL — chấp nhận được cho event batch (tần suất thấp, không phải snapshot).
- Khuyến nghị: 1 uni stream server→client cho event/notification, 1 uni stream client→server cho input batch (nếu cần), cộng datagram cho snapshot. Tổng ≤ 4 stream/session.

---

## 3. Vòng đời session

| Thành phần | Ngữ nghĩa | Ghi chú |
|---|---|---|
| `ready` | resolve khi session thiết lập; **reject** khi handshake/session fail | reject cả khi server từ chối CONNECT (4xx) |
| `closed` | **fulfill** `{closeCode, reason}` **chỉ khi** peer kết thúc session *có kèm code*; **reject** `WebTransportError{source:"session"}` khi lỗi kết nối hoặc đóng không code | spec §6.5 cleanup step 17/18 |
| `draining` | resolve khi server bắt đầu graceful shutdown (GOAWAY) | Chrome 158, Safari 26.4, FF 155 |
| `close({closeCode, reason})` | client chủ động đóng | `closeCode`: `unsigned long`; `reason` UTF-8 cắt ở **1024 byte** |
| `WebTransportError.source` | `"stream"` \| `"session"` | không có mã lỗi mạng |

### 3.1 Phân biệt "mạng chết" vs "server đóng"

Đây là chỗ dễ hiểu sai nhất, và **không có API riêng**:

```js
transport.closed
  .then(info => { /* server đóng CÓ code → deploy/restart/maintenance */ })
  .catch(err  => { /* mạng chết, TLS/QUIC error, hoặc server đóng KHÔNG code */ });
```

- Server **phải** luôn gửi code (ví dụ `4001 = restart`, `4002 = maintenance`, `4003 = kicked`) nếu muốn client phân biệt. Không code = client thấy giống hệt mất mạng.
- Không có trường nào cho biết "nguyên nhân mạng" (DNS? UDP blocked? idle timeout?). `err.message` là implementation-defined → **không được parse**.

### 3.2 Idle timeout

- Chrome: **30 s** mặc định (`GetIdleConnectionTimeout()`, `net/quic/quic_context.h`), override 300 s bằng field trial `kQuicLongerIdleConnectionTimeout`; **handshake** idle timeout **5 s** (`quic::kInitialIdleTimeoutSecs = 5`).
- quic-go (server dự kiến): `DefaultIdleTimeout = 30 s`, `DefaultHandshakeIdleTimeout = 5 s` (`internal/protocol/params.go`).
- Idle timeout có hiệu lực = **min hai phía**. Bất kỳ packet nào (kể cả datagram) reset đồng hồ.
- Kết luận: server phải bắn keepalive **≤ 15 s**; và trên mobile, tab bị suspend → khi resume gần như chắc chắn `closed` đã reject.

---

## 4. Connection migration — **KHÔNG hoạt động thực tế**

Trả lời thẳng: **đừng thiết kế dựa vào migration. WiFi → 4G sẽ làm mất WebTransport session trong Chrome desktop hiện tại.**

Bằng chứng chuỗi:

1. **Chromium chỉ bật migration trên Android.** `net/base/features.cc`:
   ```cpp
   #if BUILDFLAG(IS_ANDROID)
   kMigrateSessionsOnNetworkChangeV2Default = base::FEATURE_ENABLED_BY_DEFAULT;
   #else
   kMigrateSessionsOnNetworkChangeV2Default = base::FEATURE_DISABLED_BY_DEFAULT;
   ```
   Và `QuicSessionPool::InitializeMigrationOptions()` return sớm nếu `!NetworkChangeNotifier::AreNetworkHandlesSupported()` — API này chỉ có trên Android.
2. **WebTransport không đi qua `QuicSessionPool`.** Nó dùng `DedicatedWebTransportHttp3Client` (`net/quic/dedicated_web_transport_http3_client.cc`) — một QUIC connection riêng cho mỗi session (`allowPooling` mặc định false). File này **không** có `NetworkChangeNotifier`, không observer, không gọi `ProbeNewPath`. Nó chỉ nhận `target_network` (Android handle) lúc tạo socket.
3. Vì vậy khi interface đổi: socket cũ không gửi được nữa → QUIC write error kết thúc connection, hoặc nếu packet chỉ bị blackhole thì chết sau **idle timeout 30 s**. Không có path challenge, không có migration. `closed` reject.
4. Phía server cũng là nửa còn lại của câu chuyện: quic-go **mới** có connection migration (issue [#4960](https://github.com/quic-go/quic-go/issues/4960) merge 03/2025, API [#3990](https://github.com/quic-go/quic-go/issues/3990) 04/2025), và còn bug migration trong 10/2026 ([#5915](https://github.com/quic-go/quic-go/issues/5915) "server keeps sending to the old path after Chrome port migration, connection dies with TOO_MANY_RTOS", [#5963](https://github.com/quic-go/quic-go/issues/5963)). Nếu backend dùng bản cũ hơn, migration fail chắc chắn.
5. Safari/WebKit: WebTransport session chạy trên `nw_parameters_create_webtransport_http` (Network.framework/libquic) — xem `NetworkTransportSessionCocoa.mm`. Không có hook migration/better-path nào ở tầng WebKit; khả năng libquic tự xử lý path change khi OS báo interface đổi là **UNVERIFIED**, và kể cả có thì server Go vẫn phải chấp nhận migration. **Không được tính vào thiết kế.**

**Datagram có sống sót không?** Không — datagram sống trong session; session chết thì mất luôn, và datagram đang bay bị drop. Không có "resume" ở tầng transport.

Hệ quả: handover WiFi↔4G = **reconnect đầy đủ** (QUIC handshake mới, ~1 RTT + TLS 1.3). Companion: QUIC 0-RTT **không** được WebTransport API phơi ra (`sessionTicket` không có trong spec) → mỗi reconnect là handshake đầy đủ.

---

## 5. Auth trong browser

### 5.1 Cookie: **KHÔNG**

- Spec đặt `credentials mode = "omit"` cho request CONNECT ([spec §6.5 step 28](https://w3c.github.io/webtransport/#constructor)):
  > "neither [WEB-TRANSPORT-HTTP3] nor [WEB-TRANSPORT-HTTP2] **send cookies** or support HTTP authentication or caching invalidation mechanisms." — [spec §14.2](https://w3c.github.io/webtransport/#privacy-security)
- Chrome: handshake header được dựng thủ công, **không có** `Cookie`:
  ```cpp
  headers[":method"] = "CONNECT";
  headers[":protocol"] = "webtransport";
  headers["sec-webtransport-http3-draft02"] = "1";
  headers["origin"] = origin_.Serialize();
  // + WT-Available-Protocols, + additional_headers_
  ```
  ([`dedicated_web_transport_http3_client.cc:995`](https://github.com/chromium/chromium/blob/main/net/quic/dedicated_web_transport_http3_client.cc))
- Safari: WebKit chỉ thêm `origin` qua `nw_webtransport_options_add_connect_request_header` ([`NetworkTransportSessionCocoa.mm:213`](https://github.com/WebKit/WebKit/blob/main/Source/WebKit/NetworkProcess/webtransport/cocoa/NetworkTransportSessionCocoa.mm)), không có cookie jar.
- ⇒ Cookie `gu_auth_token` (SameSite=None; Secure trên `.cogioi.io.vn`) **không đi theo WebTransport**. Nó chỉ có tác dụng cho fetch/WS.

### 5.2 Custom header: **CÓ, nhưng không portable**

- Spec hiện tại có `WebTransportOptions.headers` (`HeadersInit`) và `responseHeaders` — đây là **thay đổi so với hiểu biết cũ "headers không hỗ trợ"**.
- Chrome: runtime feature `WebTransportHeaders` **status "stable"** trong [`runtime_enabled_features.json5`](https://github.com/chromium/chromium/blob/main/third_party/blink/renderer/platform/runtime_enabled_features.json5); Chromestatus entry "WebTransport headers and responseHeaders" đặt milestone **156**, còn BCD ghi `responseHeaders` = **157**. Chrome stable hiện tại 156 ⇒ **coi như 157 mới dùng được**. (Độ chính xác version của `headers`: **UNVERIFIED**, chromestatus để status "Proposed".)
- Safari 26.4: có trong IDL `[ImplementedAs=headersInit] HeadersInit headers;` ([WebTransportOptions.idl](https://github.com/WebKit/WebKit/blob/main/Source/WebCore/Modules/webtransport/WebTransportOptions.idl)).
- Firefox: **không** (BCD không có entry; `dom/webtransport/api/WebTransport.h` không thấy plumbing header — mức tin cậy: cao nhưng **UNVERIFIED** vì options dictionary có thể nằm ở file khác).
- Ràng buộc khi dùng: header thuộc forbidden list bị **drop im lặng** (`cors::IsForbiddenRequestHeader`); set `wt-available-protocols` → **TypeError**; `set-cookie` bị lọc khỏi response.
- ⇒ `Authorization: Bearer <jwt>` chạy được trên Chrome 157+ và Safari 26.4, **không** chạy Firefox. Nếu chọn header làm auth chính, Firefox mất tính năng. **Giữ `?token=` làm đường chính.**

### 5.3 `Origin`

- Browser **luôn gửi** `Origin` (Chrome: `headers["origin"]`; Safari: `nw_webtransport_options_add_connect_request_header(..., "origin", ...)`). MDN cũng nói rõ: "the request also includes an `Origin` header, which the server must verify before accepting the session" ([MDN WebTransport API](https://developer.mozilla.org/en-US/docs/Web/API/WebTransport_API)).
- **Không có CORS preflight, không có CORS response check** trong algorithm establish của spec. Browser không hộ tống bạn: server **phải tự** so `Origin` với allowlist và từ chối (403) nếu sai, y như WS hiện tại.
- `Origin` **không phải** authentication: client ngoài browser (Go/curl/native) giả mạo được. Nó chỉ chặn CSRF kiểu "trang web khác mở session bằng cookie" — mà WebTransport còn không gửi cookie, nên giá trị thực tế là chặn cross-origin abuse/scan.

### 5.4 `serverCertificateHashes`

- Chỉ cần khi cert **không** verify được bằng Web PKI: self-signed, LAN/dev, hoặc server ephemeral trong container không có DNS công khai. Với `cogioi.io.vn` + Let's Encrypt (hoặc cert thật bất kỳ) → **không cần**.
- Ràng buộc (spec §"custom certificate requirements", [§14.4](https://w3c.github.io/webtransport/#certificate-hashes)):
  - X.509v3, **không được dùng RSA**; bắt buộc hỗ trợ **ECDSA P-256** (nên phát hành cert EC).
  - **Tổng thời hạn hiệu lực ≤ 2 tuần** → phải rotate cert thường xuyên, không dùng cert 90 ngày.
  - **Không dùng được với `allowPooling: true`**.
  - `newConnection: "no"` + `serverCertificateHashes` non-empty → **NotSupportedError**.
- Chrome hỗ trợ từ **100** (Chromestatus "WebTransport serverCertificateHashes option"); Safari 26.4 có trong IDL; Firefox: `serverCertificateHashes` có trong BCD từ 114? → **UNVERIFIED**, nhưng không cần cho production domain thật.

---

## 6. Ma trận tương thích

| Tính năng | Chrome 156 (stable) | Chrome 157–159 | Firefox 155+ | Safari / iOS 26.4 |
|---|---|---|---|---|
| `WebTransport` cơ bản | ✅ 97 | ✅ | ✅ 114 | ✅ 26.4 |
| Datagram (`datagrams`) | ✅ | ✅ | ✅ 114 | ✅ |
| `maxDatagramSize` | ✅ **~1200 (negotiated)** | ✅ | ✅ | ✅ **hard-code 1024** |
| `datagrams.writable` (legacy) | ✅ | ✅ (deprecated) | ✅ | ❌ **không có** |
| `datagrams.createWritable()` | ❌ | ✅ 159 (experimental) | ✅ 155 | ✅ **bắt buộc dùng cái này** |
| `incoming/outgoingMaxBufferedDatagrams` | ✅ 151 | ✅ | ❌ | ⏳ "preview" |
| `incoming/outgoingHighWaterMark` | ✅ (deprecated) | ✅ | ✅ | ❌ (đã bỏ) |
| `incomingMaxAge` / `outgoingMaxAge` | ✅ | ✅ | ✅ | ✅ |
| `createUnidirectionalStream` trả `WebTransportSendStream` | ❌ (trả `WritableStream`) | ✅ 159 | ⚠️ partial | ✅ |
| `sendOrder` / `sendGroup` | ❌ | ⚠️ 159 experimental | ⚠️ createSendGroup 155 | ✅ |
| `reliability` + `supportsReliableOnly` | ❌ (157) | ✅ 157 | ✅ (supportsReliableOnly 158) | ✅ 26.4 |
| `draining` | ❌ (158) | ✅ 158 | ✅ 155 | ✅ |
| `headers` (request) | ⚠️ 156/157 (UNVERIFIED exact) | ✅ | ❌ | ✅ |
| `responseHeaders` | ❌ | ✅ 157 | ❌ | ⏳ preview |
| `getStats()` | ❌ | ✅ 159 | ❌ (throw) | ⚠️ trả zero-filled |
| `exportKeyingMaterial` | ❌ | ✅ 158 | ✅ 155 | ❌ |
| Cookie trong handshake | ❌ | ❌ | ❌ | ❌ |
| `Origin` header | ✅ | ✅ | ✅ | ✅ |
| `serverCertificateHashes` | ✅ 100 | ✅ | UNVERIFIED | ✅ (IDL) |
| Network-change migration | ❌ (chỉ Android) | ❌ | ❌ | UNVERIFIED, không tin được |

Nguồn bảng: [BCD `api/WebTransport.json`](https://github.com/mdn/browser-compat-data/blob/main/api/WebTransport.json), [BCD datagram](https://github.com/mdn/browser-compat-data/blob/main/api/WebTransportDatagramDuplexStream.json), [Chromium runtime flags](https://github.com/chromium/chromium/blob/main/third_party/blink/renderer/platform/runtime_enabled_features.json5), [Chromestatus](https://chromestatus.com/features), [WebKit IDL](https://github.com/WebKit/WebKit/tree/main/Source/WebCore/Modules/webtransport).

### 6.1 Safari/iOS sẽ vỡ ở đâu trước

Xếp theo xác suất:

1. **Size datagram**: code viết cho Chrome (1200B) bị Safari **drop im lặng** vì cap 1024. Không exception, chỉ là client "không thấy gì".
2. **`datagrams.writable` undefined** ở Safari → `TypeError` ngay dòng đầu. Phải `createWritable ? createWritable() : writable`.
3. **Deadlock flow-control khi mở nhiều incoming stream** (bug 317084) → session treo vĩnh viễn, không có tín hiệu lỗi. Tránh bằng 1 stream dài hạn.
4. **HTTP/2 fallback** → `ready` resolve nhưng `reliability === "reliable-only"`, không datagram. Nếu không check, logic snapshot không bao giờ chạy và không có lỗi.
5. `incoming/outgoingHighWaterMark` **không tồn tại** ở Safari → set vào là no-op (không throw vì là attribute trên prototype? — **UNVERIFIED**, coi như không dùng).
6. iOS 26.4 mới ra 03/2026 → phần lớn thiết bị iOS ngoài kia vẫn < 26.4 ⇒ fallback WS là bắt buộc trong nhiều tháng.

---

## 7. Phát hiện "không hỗ trợ" vs "UDP bị chặn"

- **Không hỗ trợ**: quyết định được, đồng bộ: `typeof WebTransport === 'undefined'` (hoặc `!('WebTransport' in window)`).
- **UDP bị chặn**: **không có tín hiệu đáng tin**. Browser không có API probe UDP; không có error code riêng. Mọi failure đều dồn vào `ready` reject với `WebTransportError` message implementation-defined. Phân biệt được duy nhất: **thời gian** và **hành vi** (xem dưới), và ngay cả vậy cũng không chắc.
- Thời gian failure thực tế:
  - UDP bị **blackhole** (firewall drop): hết **handshake idle timeout ~5 s** (Chrome `kInitialIdleTimeoutSecs = 5`; quic-go `DefaultHandshakeIdleTimeout = 5 s`) rồi `ready` reject.
  - UDP bị **reject bằng ICMP port-unreachable**: fail trong **~1 RTT**.
  - Server sống nhưng **không nói HTTP/3/WebTransport** (chỉ HTTP/1.1–2): fail trong **~1 RTT** sau khi thiếu `SETTINGS_WT_ENABLED` / `SETTINGS_H3_DATAGRAM`.
  - Safari có HTTP/2 fallback: có thể **thành công** (không fail) nhưng không có datagram → phải check `reliability`.
- Khuyến nghị: `Promise.race([ready, timeout(3000)])` → nếu timeout/quá 3 s thì **vẫn** thử tiếp tới 5 s ở background, đồng thời mở WS fallback song song; chốt transport khi cái nào `ready` trước. Đo `Date.now()` quanh `ready` để log phân bố (sẽ lộ ra tỉ lệ mạng chặn UDP).

---

## 8. Design consequences — protocol phải khác gì so với WS hiện tại

### 8.1 Datagram (snapshot 20 Hz, latest-wins)

1. **Budget cứng 1024 B/datagram** (min của Chrome ~1200 và Safari 1024), và **assert `payload.byteLength <= t.datagrams.maxDatagramSize`** trước mỗi write; nếu vượt → **không write**, gửi full-state qua stream. Không có exception để bắt, nên assert là bắt buộc.
2. Snapshot phải **delta hoặc quantize** để nằm dưới 1 KB: hiện protocol là JSON text 20 Hz. JSON 1 KB ≈ ~35 entity nhỏ; nếu vượt, cần binary encoding (varint/bitpack) hoặc tách "entity set" khỏi "transform stream".
3. **Đặt `outgoingMaxAge` ~50–100 ms** để datagram cũ tự hủy khi nghẽn, thay vì tới trễ và ghi đè state mới.
4. **Luôn có `seq` trong datagram** (đã có `ack_seq` cho input; snapshot cần `snap_seq`), client bỏ snapshot cũ. Vì queue drop từ head, thứ tự đến có thể lệch.
5. Không BAO GIỜ dựa vào "write thành công" = "server nhận". Không có ack ở tầng datagram.

### 8.2 Event reliable

6. **Một** uni stream server→client, sống suốt session, length-prefix framing. **Không** mở stream cho từng event (bug Safari 317084). Giữ tổng stream/session ≤ 4.
7. Không dùng `sendOrder`/`sendGroup` ở v1 (Chrome stable chưa có). Nếu cần ưu tiên, dùng 2 stream khác nhau.
8. Server phải tôn trọng `MAX_STREAMS` của client (mặc định 100 uni) và không mở stream mới khi client chưa đọc.

### 8.3 Vòng đời / reconnect

9. **Không có migration** ⇒ phải có state machine reconnect: `connecting → ready → (draining) → closed/failed → backoff → connecting`. Reconnect ~300 ms–2 s, exponential backoff, có jitter.
10. **Resume token**: session mới phải gửi kèm `resume_token` (JWT riêng, TTL ngắn) + `last_snap_seq` + `last_ack_seq` để server biết client đã thấy gì. Vì snapshot latest-wins, chỉ cần full-state ngay sau resume.
11. Server **luôn** đóng với `closeCode` (`4001 restart`, `4002 maintenance`, `4003 kicked`, `4000 normal`). Client: `closed.then()` → reconnect ngay; `closed.catch()` → backoff rồi reconnect; `draining` (nếu có) → reconnect **trước khi** socket chết.
12. Keepalive **≤ 15 s** (idle 30 s). Trên mobile/background, giả định session chết và resume bằng full-state.
13. Đừng map `closeCode` của WS (1000/1001) sang WebTransport: đó là không gian mã HTTP/3 + app code 32-bit. Định nghĩa bảng mã riêng.

### 8.4 Auth

14. **Giữ JWT ở query string `?token=`** làm đường chính — đây là đường duy nhất chạy trên cả 3 engine. Cookie **không** hoạt động (mục 5.1).
15. Cân nhắc `headers: { Authorization: ... }` như **đường nâng cấp** khi Chrome 157+/Safari 26.4 phổ cập; khi đó server chấp nhận **cả hai** (query + header). Nhưng đừng bỏ query cho tới khi Firefox có header.
16. Server **validate `Origin`** (allowlist `https://cogioi.io.vn`) và trả 403 — browser không làm hộ. Origin **không** thay thế token.
17. Token trong URL sẽ nằm trong log/Referer của reverse proxy; với WebTransport không có Referer, nhưng log vẫn thấy path → **không log query string**, và dùng token TTL ngắn (hiện tại đã là JWT).
18. Rate-limit theo session: datagram không cần handshake sau khi mở, nên flood rất rẻ. Bound `pending datagrams/session` và đóng session vượt ngưỡng.

### 8.5 Capability detection (bắt buộc, chạy 1 lần lúc boot)

```js
const supported = typeof WebTransport !== 'undefined';
const wrt = supported ? (t.datagrams.createWritable ? t.datagrams.createWritable() : t.datagrams.writable) : null;
const mode = t.reliability; // "pending" | "reliable-only" | "supports-unreliable"
```
- Nếu `!createWritable && !writable` → không có datagram path.
- Nếu `reliability === "reliable-only"` → snapshot phải đi qua stream (chấp nhận HOL) **hoặc** fallback WS.
- `WebTransport.supportsReliableOnly === true` (static) → UA này có thể kết nối không-UDP; không đồng nghĩa session hiện tại không có datagram.

### 8.6 Fallback WS: **giữ lại vĩnh viễn**

Không phải "di trú xong thì bỏ WS". Các lý do cố định:
- Safari/iOS < 26.4 (đa số người dùng tại 11/2026).
- Mạng chặn UDP (khách sạn, wifi công ty, một số ISP VN).
- Safari reliable-only fallback (không datagram).
- WebView Android cũ, Firefox ESR cũ.
Thiết kế: một `Transport` interface chung (`sendInput`, `onSnapshot`, `onEvent`, `close`), hai implementation (`WsTransport`, `WtTransport`), chọn bằng probe + timeout 3–5 s.

---

## 9. Điều ticket giả định sai (cần sửa trong map)

| Giả định trong #306 | Thực tế |
|---|---|
| "`datagrams.writeable`" | Không tồn tại. Dùng `writer.ready` / `outgoingMaxAge` / `outgoingMaxBufferedDatagrams`. |
| "custom header không hỗ trợ" | Spec + Chrome 156/157 + Safari 26.4 **có** `WebTransportOptions.headers`; chỉ Firefox thiếu. |
| "cookie `gu_auth_token` sẽ đi theo" | Không. `credentials mode: omit`, không cookie jar. |
| "~1200 byte là ngưỡng an toàn" | Safari cap **1024**; 1200 chỉ là giới hạn UDP payload tối thiểu của QUIC. |
| "migration có thể cứu session" | Không, trừ Android Chrome + server quic-go mới. |

## 10. UNVERIFIED / cần đo trên thiết bị thật

1. Chrome **156** có `WebTransportOptions.headers` hay chưa (chromestatus milestone 156 vs BCD `responseHeaders` 157). → Test 1 dòng trên Chrome stable.
2. Safari/iOS có migrate path khi WiFi→4G không (phụ thuộc Network.framework/libquic). → Test thực địa, nhưng thiết kế không được phụ thuộc.
3. `maxDatagramSize` Safari **thực tế** lúc runtime (source hard-code 1024, nhưng có thể Apple đã đổi sau bản main tôi đọc). → In ra trong app, log lại.
4. Giá trị `maxDatagramSize` Chrome trên mạng thật (đo được 1211 trên một máy; có thể thấp hơn nếu path MTU nhỏ / VPN).
5. `reliability` trên Safari khi thực sự fallback HTTP/2 — hành vi fallback mới chỉ được WebKit blog mô tả, chưa thấy WPT.
6. Hành vi của Firefox khi ghi datagram oversized (bug 1821566 từ 2023; trạng thái hiện tại chưa kiểm tra).

## 11. Nguồn

**Spec / RFC**
- WebTransport (Editor's Draft, 2026-10-29) — https://w3c.github.io/webtransport/
- WebTransport (W3C TR, 2026-07-30) — https://www.w3.org/TR/webtransport/
- RFC 9221 (QUIC DATAGRAM) §3, §4, §5.1 — https://www.rfc-editor.org/rfc/rfc9221.html
- RFC 9000 §9 (Connection Migration), §18.2 (`max_udp_payload_size`) — https://www.rfc-editor.org/rfc/rfc9000.html
- draft-ietf-webtrans-http3 — https://datatracker.ietf.org/doc/draft-ietf-webtrans-http3/ (không nhắc migration)

**Chromium**
- `runtime_enabled_features.json5` (WebTransportHeaders → stable) — https://github.com/chromium/chromium/blob/main/third_party/blink/renderer/platform/runtime_enabled_features.json5
- `services/network/web_transport.cc` / `.h` (max datagram size plumbing, 4096/16 MiB queue) — https://github.com/chromium/chromium/tree/main/services/network
- `net/quic/dedicated_web_transport_http3_client.cc` (CONNECT headers, không cookie) — https://github.com/chromium/chromium/blob/main/net/quic/dedicated_web_transport_http3_client.cc
- `net/quic/quic_context.h` (idle 30 s, handshake 5 s) — https://github.com/chromium/chromium/blob/main/net/quic/quic_context.h
- `net/base/features.cc` (`kMigrateSessionsOnNetworkChangeV2` chỉ Android) — https://github.com/chromium/chromium/blob/main/net/base/features.cc
- `net/quic/quic_session_pool.cc` (`InitializeMigrationOptions`) — https://github.com/chromium/chromium/blob/main/net/quic/quic_session_pool.cc
- Blink WebTransport module — https://github.com/chromium/chromium/tree/main/third_party/blink/renderer/modules/webtransport

**quiche**
- `quic_spdy_stream.cc` `GetMaxDatagramSize()` — https://github.com/google/quiche/blob/main/quiche/quic/core/http/quic_spdy_stream.cc
- `quic_packet_creator.cc` `GetGuaranteedLargestDatagramPayload()` — https://github.com/google/quiche/blob/main/quiche/quic/core/quic_packet_creator.cc

**WebKit**
- WebTransport IDL — https://github.com/WebKit/WebKit/tree/main/Source/WebCore/Modules/webtransport
- `WebTransportDatagramDuplexStream.cpp` (`maxDatagramSize = 1024`) — https://github.com/WebKit/WebKit/blob/main/Source/WebCore/Modules/webtransport/WebTransportDatagramDuplexStream.cpp
- `NetworkTransportSessionCocoa.mm` (Origin header) — https://github.com/WebKit/WebKit/blob/main/Source/WebKit/NetworkProcess/webtransport/cocoa/NetworkTransportSessionCocoa.mm
- Bug 317084 (flow-control deadlock, many incoming streams) — https://bugs.webkit.org/show_bug.cgi?id=317084
- "WebKit Features for Safari 26.4" (WebTransport shipped, HTTP/2 fallback) — https://webkit.org/blog/17862/webkit-features-for-safari-26-4/

**quic-go / webtransport-go**
- `internal/protocol/params.go` (idle 30 s, handshake 5 s, 100 streams) — https://github.com/quic-go/quic-go/blob/master/internal/protocol/params.go
- `datagram_queue.go` (send 32 / recv 128) — https://github.com/quic-go/quic-go/blob/master/datagram_queue.go
- Issue #4960 (connection migration added) — https://github.com/quic-go/quic-go/issues/4960
- Issue #5915 / #5963 (migration bugs 10/2026) — https://github.com/quic-go/quic-go/issues/5915

**Compat / đo lường**
- MDN BCD `api/WebTransport.json` — https://github.com/mdn/browser-compat-data/blob/main/api/WebTransport.json
- MDN BCD `api/WebTransportDatagramDuplexStream.json` — https://github.com/mdn/browser-compat-data/blob/main/api/WebTransportDatagramDuplexStream.json
- MDN WebTransport API (Origin phải verify) — https://developer.mozilla.org/en-US/docs/Web/API/WebTransport_API
- Chromestatus (feature list cho WebTransport) — https://chromestatus.com/features
- chromiumdash (Chrome stable 156.0.8078.80, 2026-10-29) — https://chromiumdash.appspot.com/fetch_releases?channel=Stable&platform=Windows
- Đo `maxDatagramSize = 1211` trên Chrome 155 — https://www.irurueta.com/web/webrtc-websockets/webtransport-and-media-over-quic.html
- Bugzilla 1821566 (Firefox datagram size) — https://bugzilla.mozilla.org/show_bug.cgi?id=1821566
