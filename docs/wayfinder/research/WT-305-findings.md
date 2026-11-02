# WT-305 — quic-go / webtransport-go + TLS/ACME on :443 (production suitability)

> Ticket: [#305](https://github.com/vippro67/GuWorld/issues/305), part of map [#303](https://github.com/vippro67/GuWorld/issues/303).
> Surveyed: 2026-11-02. Versions checked against the Go module proxy on that date.
> Scope: production suitability, not a tutorial. Anything I could not verify from a primary source is marked **UNVERIFIED**.

## Context assumed true (given, not re-derived)

- Go 1.26.3 (`backend/go.mod`), `quic-go v0.59.0` present as an **indirect** dependency.
- Current production: `gorilla/websocket` + plain `http.Server` on TCP :8080 behind Caddy, Cloudflare proxied in front of `api.cogioi.io.vn`.
- Planned: drop Caddy, Go terminates TLS for REST (TCP/443) and WebTransport (UDP/443).
- Workload: 20 Hz zone sim, ~15 realtime channels, JSON today, binary planned; target ~2000 concurrent sessions per process.

---

## 1. Verdict

**GO — the library stack is production-usable, but the deployment is blocked by Cloudflare and the default tuning is wrong for 2000 sessions.**

1. **Libraries: yes.** `quic-go` is described by its own docs as "an optimized, production-ready implementation of QUIC", and is the QUIC stack inside Caddy, cloudflared, AdGuardHome, Traefik, syncthing, libp2p, etc. `webtransport-go` is the reference Go implementation and is used by Centrifugo, MediaMTX, libp2p, socket.io-go. Neither has a 1.0; both are `v0.x` and make no API-stability promise across minor versions → **pin exact versions**.
2. **One Go process for TCP/443 + UDP/443: supported and normal.** One `net.PacketConn` on UDP/443, wrapped by `webtransport.Server`; that server also dispatches plain HTTP/3 requests to its wrapped `http3.Server`. A separate `net.Listener` on TCP/443 runs `http.Server` for HTTP/1.1 + HTTP/2. Both share one certificate source (`tls.Config.GetCertificate`).
3. **TLS/ACME in-process: yes.** Use `certmagic` + **DNS-01 via Cloudflare**, not `autocert` (`autocert` has no DNS-01 and is explicitly "a work in progress", "makes no API stability promises"). DNS-01 also survives the orange-cloud proxy and works for wildcards.
4. **The hard constraint is not the Go code — it is Cloudflare.** Cloudflare's proxied (orange-cloud) HTTP path terminates HTTP/3 at the edge and **does not speak HTTP/3 to the origin**; Cloudflare also blocks the `CONNECT` method. WebTransport *is* an HTTP/3 Extended `CONNECT`, so it cannot traverse the orange cloud. The realtime hostname must be **DNS-only (grey cloud)** — exposing the origin IP to raw UDP — or put behind **Spectrum** (Enterprise, paid add-on). See §2.5.
5. **At 2000 sessions, the defaults are unsafe.** `MaxConnectionReceiveWindow` defaults to 15 MiB **per QUIC connection**; worst-case buffered inbound at 2000 sessions is ~30 GiB (default) vs ~4 GiB at a 2 MiB cap. Datagram payload is capped at **1024 B** by Chrome and WebKit, and quic-go's DATAGRAM send path is explicitly unoptimized. Tune before load testing.

**Single most important constraint:** WebTransport cannot be served through the Cloudflare orange cloud. Choose now between grey-cloud origin (IP exposed, no WAF/DDoS) or Spectrum (Enterprise), or the plan does not deploy.

---

## 2. Facts

### 2.1 Maturity, versions, maintenance

| Item | Fact | Source |
|---|---|---|
| quic-go status | "optimized, production-ready implementation of the QUIC protocol (RFC 9000/9001/9002)" | https://quic-go.net/docs/ |
| quic-go production users | Caddy, cloudflared, AdGuardHome, Traefik, syncthing, go-libp2p, frp, RoadRunner, Mercure, … | https://github.com/quic-go/quic-go/blob/master/README.md |
| quic-go release policy | "always aims to support the latest two Go releases" | https://github.com/quic-go/quic-go/blob/master/README.md |
| webtransport-go status | Implements **draft-ietf-webtrans-http3-16** | https://quic-go.net/docs/webtransport/ |
| webtransport-go users | any-sync, Centrifugo, go-libp2p, MediaMTX, signalr, socket.io | https://github.com/quic-go/webtransport-go/blob/v0.13.0/README.md |
| Draft status | `draft-ietf-webtrans-http3-16`, 2026-07-06, **WG Last Call**, intended Proposed Standard — not an RFC yet | https://datatracker.ietf.org/doc/draft-ietf-webtrans-http3/ |
| Draft-drift warning | "There is no guarantee that browsers will update in a backwards-compatible way… Support for WebTransport therefore might break for a transition period" | https://quic-go.net/docs/webtransport/ |
| Browser support | Chrome/Edge 97+, Firefox 114+, Safari 26.4+ (Mar 2026); ~91.6 % global | https://caniuse.com/webtransport , https://webstatus.dev/features/webtransport |

**Version timeline (Go module proxy, `.info` timestamps):**

| quic-go | Released | webtransport-go | Released |
|---|---|---|---|
| v0.59.0 | 2026-01-11 | v0.11.0 | 2026-06-15 |
| v0.60.0 | 2026-06-04 | v0.12.0 | 2026-07-27 |
| v0.61.0 | 2026-07-21 | **v0.13.0** | **2026-08-30** |
| **v0.62.0** | **2026-08-30** | — | — |
| v0.63.0 | 2026-09-22 | — | — |
| v0.64.0 | 2026-10-16 | — | — |
| v0.64.1 (latest) | 2026-10-19 | — | — |

`webtransport-go v0.13.0`'s `go.mod` requires `github.com/quic-go/quic-go v0.62.0` and `go 1.26.0`
(https://github.com/quic-go/webtransport-go/blob/v0.13.0/go.mod). It was released the same day as quic-go v0.62.0 — these are the paired, tested versions. The repo's `v0.59.0` is ~10 months stale and predates the draft-16 flow-control work.

**API shape (v0.13.0, verified from source):**

- `webtransport.Server{ H3 *http3.Server, Config *Config, ApplicationProtocols []string, CheckOrigin func(*http.Request) bool, ReorderingTimeout time.Duration }`
  (https://github.com/quic-go/webtransport-go/blob/v0.13.0/server.go)
- `func (s *Server) Serve(net.PacketConn) error` — owns the UDP socket and the QUIC listener; it calls `quic.ListenEarly(conn, s.H3.TLSConfig, quicConf)` and forces `EnableDatagrams` + `EnableStreamResetPartialDelivery` on a clone of `H3.QUICConfig`.
  (same file, `serve()`)
- `func (s *Server) Upgrade(w http.ResponseWriter, r *http.Request) (*Session, error)` — the `CONNECT` handler.
- `Session`: `AcceptStream`, `OpenStream`, `AcceptUniStream`, `OpenUniStream`, `SendDatagram`, `ReceiveDatagram`, `CloseWithError`, `Context`, `SessionState`.
  (https://github.com/quic-go/webtransport-go/blob/v0.13.0/session.go)
- `webtransport.Config{ MaxIncomingStreams, MaxIncomingUniStreams, MaxIncomingData }` — **session-level** draft-16 flow control, added in this version.
  (https://github.com/quic-go/webtransport-go/blob/v0.13.0/config.go)

**Two API details that bite:**

- Default `CheckOrigin` accepts only when `Origin` host == request `Host` (`checkSameOrigin`, server.go:500). A game served from `cogioi.io.vn` connecting to `wt.cogioi.io.vn` is **cross-origin and will be rejected** unless you set `CheckOrigin`.
- `webtransport.Server.Close()` closes all sessions, then calls `s.H3.Close()`. There is **no `Shutdown(ctx)`** on the WebTransport server → graceful WT drain must be implemented at the application level.

### 2.2 Operational limits (defaults, from source)

All constants below are from `quic-go v0.62.0`; the relevant values are identical in v0.64.1 (diffed `internal/protocol/params.go` and `config.go`).

| Knob | Default | Where |
|---|---|---|
| `MaxIdleTimeout` | 30 s | `internal/protocol/params.go` |
| `HandshakeIdleTimeout` | 5 s | same |
| `KeepAlivePeriod` | 0 (disabled) | `config.go` |
| `MaxIncomingStreams` (per QUIC conn) | 100 bidi | `internal/protocol/params.go` |
| `MaxIncomingUniStreams` | 100 | same |
| `InitialStreamReceiveWindow` | 512 KiB | same |
| `MaxStreamReceiveWindow` | 6 MiB | same |
| `InitialConnectionReceiveWindow` | 768 KiB (1.5 × 512 KiB) | same |
| `MaxConnectionReceiveWindow` | **15 MiB per connection** | same |
| `InitialPacketSize` | 1280 B | same |
| Max QUIC packet / MTU ceiling | **1452 B** (`MaxPacketBufferSize`); not configurable | `internal/protocol/protocol.go`, https://quic-go.net/docs/quic/optimizations/ |
| QUIC DATAGRAM frame max | 16383 B advertised (`wire.MaxDatagramSize`), but actually bounded by path MTU | `internal/wire/datagram_frame.go`, `connection.go` `SendDatagram` |
| Browser WebTransport datagram cap | **1024 B** (Chrome constant; WebKit also 1024) | https://chromium.googlesource.com/chromium/src/+/ff66885fb0d785437f82f9ab7aaedd3d3dcc8812 , https://github.com/WebKit/WebKit/pull/70152 |
| Server accept queue | 32 pending | `internal/protocol/params.go` |
| Unprocessed packets | 256/connection, 1024/server | same |
| Congestion control | CUBIC sender, **no public config knob**; pluggable CC is future work (#776) | `internal/ackhandler/sent_packet_handler.go`, https://quic-go.net/docs/quic/congestion-control/ |
| Max congestion window | 10000 packets | `internal/protocol/params.go` |
| Desired UDP buffers | 7 MiB RX/TX; app can only raise up to kernel `rmem_max`/`wmem_max` | https://quic-go.net/docs/quic/optimizations/ |
| HTTP/3 `IdleTimeout` | 0 = no HTTP-layer timeout; QUIC idle timeout still applies | `http3/server.go` |
| Sessions per process | **no cap in quic-go/webtransport-go** — bounded by CPU/RAM; one UDP socket = one fd | architecture (see below) |

**Congestion control correction:** the quic-go docs page still says "RFC 9002 §7", but the shipped code constructs `congestion.NewCubicSender` (CUBIC). There is no supported way to select BBR/NewReno/L4S; the only tunables are the flow-control windows and the internal max cwnd. Treat "configure congestion control" as **not available**.

**Socket/goroutine topology (matters at 2000):**

- One `quic.Transport` per UDP socket; `Transport.listen()` is a **single goroutine** that reads packets (batched `recvmmsg` on Linux), maps connection ID, and enqueues to the per-connection queue. Decryption/frame handling runs on each connection's own goroutine.
  (https://github.com/quic-go/quic-go/blob/v0.62.0/transport.go)
- Per established WebTransport session you get roughly: 2 goroutines for the QUIC connection + 2 goroutines for the WT session capsule reader/writer + your application goroutines. At 2000 sessions that is ~12k goroutines — fine, but not free.
- Because all sessions share one UDP socket, there is no per-connection file descriptor cost. The scaling axis is **packets/s and memory**, not fds. quic-go has no built-in multi-queue/SO_REUSEPORT sharding; scale out with processes (and `SO_REUSEPORT` at the OS level) if one receive goroutine saturates.

**Memory at 2000 sessions (derived, not measured):**

- The only hard bound the library gives you is the flow-control window: a peer may leave up to `MaxConnectionReceiveWindow` bytes buffered per connection if the application does not read. Default 15 MiB × 2000 = **~30 GiB worst case**; at 2 MiB → **~4 GiB**. Realistic steady state is far lower because a 20 Hz snapshot stream is read continuously, but you must budget for the ceiling.
- Measured RSS per session at 2000 sessions: **UNVERIFIED** — no published benchmark for this workload; requires a load test (see §6).

### 2.3 HTTPS (TCP) + WebTransport (UDP) in one process

**Supported, and it is how the library is designed to be used.**

- quic-go docs: "With this package, it is possible to run a Go server that serves HTTP/1.1, HTTP/2 and HTTP/3." (https://quic-go.net/docs/)
- `webtransport.Server.Serve` accepts **all** streams on the QUIC connection: streams whose type is the WebTransport frame go to the session manager, everything else is handed to `http3Conn.HandleRequestStream(str)` / `HandleUnidirectionalStream(str)` — i.e. normal HTTP/3 requests on the **same UDP socket and same QUIC connection**.
  (https://github.com/quic-go/webtransport-go/blob/v0.13.0/server.go, `ServeQUICConn`)
- Therefore: **do not** run `http3.Server.Serve`/`ListenAndServe` separately on the same `:443`. The WebTransport server owns the UDP socket; the H3 server only supplies the handler and TLS/QUIC config.
- TLS: one certificate source, two ALPN profiles:
  - TCP `tls.Config`: `NextProtos = ["h2","http/1.1"]`, `GetCertificate` from certmagic.
  - QUIC `tls.Config`: `http3.ConfigureTLSConfig(magic.TLSConfig())` sets `NextProtos = ["h3"]` and preserves `GetCertificate` (it clones; it wraps `GetConfigForClient`, not `GetCertificate`).
    (https://github.com/quic-go/quic-go/blob/v0.62.0/http3/server.go)
- `http.Server.ListenAndServeTLS("", "")` works with only `GetCertificate` set (stdlib skips `LoadX509KeyPair` when `GetCertificate != nil` and both filenames are empty), and configures HTTP/2 automatically.
  (https://cs.opensource.google/go/go/+/master:src/net/http/server.go)

**Graceful shutdown ordering** (no WT `Shutdown` exists):

1. Stop accepting new `/wt` upgrades (flip a flag; return 503) and broadcast an app-level "draining" message on existing sessions.
2. `tcpServer.Shutdown(ctx)` — drains in-flight REST requests, sends GOAWAY (h2).
3. `wtServer.Close()` — closes QUIC connections and then the wrapped `http3.Server`. Do **not** also call `h3.Shutdown`; `Close` already closes it.
4. Close the `net.PacketConn`.

`http3.Server.Shutdown(ctx)` exists and sends GOAWAY for pure H3 requests, but it does not gracefully end WebTransport sessions; `webtransport.Server.Close` is abrupt by design.

### 2.4 TLS + ACME inside Go

**Recommendation: `certmagic` + DNS-01 (Cloudflare). Not `autocert`.**

| | certmagic v0.25.6 | autocert (x/crypto) |
|---|---|---|
| DNS-01 | Yes, any `libdns` provider | **No** — only `http-01` and `tls-alpn-01` |
| Multiple names | `ManageSync(ctx, names)`; one cert per name, SNI selects | one Manager, `HostPolicy` |
| Storage | pluggable (`FileStorage`, remote backends, `LocalCache`) | `DirCache` / custom `Cache` |
| Stability | used by Caddy; stable API | "work in progress and makes no API stability promises" |
| Source | https://github.com/caddyserver/certmagic | https://pkg.go.dev/golang.org/x/crypto/acme/autocert |

Facts that matter:

- `certmagic.Config.TLSConfig()` returns `{GetCertificate, NextProtos: ["acme-tls/1"]}`; you must **replace/append** `NextProtos` for h2/h1.1 and/or run `http3.ConfigureTLSConfig` for h3.
  (https://github.com/caddyserver/certmagic/blob/v0.25.6/config.go)
- `ACMEIssuer.DNS01Solver` is an `acmez.Solver`; the concrete value is `&certmagic.DNS01Solver{DNSManager: certmagic.DNSManager{DNSProvider: ...}}`.
  (https://github.com/caddyserver/certmagic/blob/v0.25.6/acmeissuer.go , `solvers.go`)
- Cloudflare provider: `github.com/libdns/cloudflare`, field `APIToken` (scoped token, `Zone.DNS:Write`; optional `ZoneToken` for `Zone:Read`) — the provider docs explicitly say not to use the global API key.
  (https://github.com/libdns/cloudflare/blob/v0.2.2/provider.go)
- `ManageSync` iterates the name list and manages **one certificate per name** (`manageOne`), so "multiple hostnames" = pass all names; `GetCertificate` resolves by SNI. It is not a single multi-SAN cert unless you ask for a wildcard.
  (https://github.com/caddyserver/certmagic/blob/v0.25.6/config.go, `manageAll`)
- Hot reload: certificates live in certmagic's in-memory cache; background maintenance renews and swaps them, and `GetCertificate` serves the current cached cert — no restart, no listener reload. certmagic does **not** watch the filesystem; if another process writes the volume, restart or call `CacheManagedCertificate(ctx, name)` to re-read. (Provider-level: certmagic keeps the cert in cache; no fs watcher configured in v0.25.6.)
- Container persistence: default storage is `FileStorage` under a data dir (`storage.go`: `defaultFileStorage = &FileStorage{Path: dataDir()}`), which in a container is ephemeral. **Set `Storage: &certmagic.FileStorage{Path: "/data/certs"}` and mount a volume.** Losing it is not fatal but re-issuance on every restart will hit Let's Encrypt rate limits.
- DNS-01 does not need inbound port 80 or 443, works while the name is orange-clouded, and is the only way to get a wildcard. Disable HTTP-01 and TLS-ALPN-01 explicitly so certmagic never tries to bind port 80/443 challenge listeners.
- Use `certmagic.LetsEncryptStagingCA` while testing.

### 2.5 External WebTransport proxy / terminator

Short answer: **nothing mature exists today.** The reason is structural: WebTransport does not tunnel data inside the Extended `CONNECT` stream; it spawns additional QUIC streams and DATAGRAM frames on the same connection, identified by session ID, so a proxy must demultiplex child streams/datagrams and re-originate them upstream.

| Product | WebTransport proxying | Evidence |
|---|---|---|
| Cloudflare (orange cloud) | **No** | HTTP/3 to origin "not yet supported": https://developers.cloudflare.com/speed/optimization/protocol/http3/ ; `CONNECT` is a restricted method: https://developers.cloudflare.com/fundamentals/concepts/traffic-flow-cloudflare/ ; Cloudflare community answer (LucasCF): "WebTransport won't currently work… use Spectrum for Layer 4": https://community.cloudflare.com/t/unable-to-forward-webtransport-request-via-loadbalancer-to-origin/335218 |
| Cloudflare Spectrum | L4 UDP possible | Enterprise plan + paid add-on: https://developers.cloudflare.com/spectrum/ |
| Envoy | **Alpha, in progress** | Initial WebTransport bridging merged 2026-06-24 (PR #45630), disabled by default behind a runtime flag; server-initiated streams still pending (PR #47270, open). https://github.com/envoyproxy/envoy/pull/45630 , https://github.com/envoyproxy/envoy/issues/41981 |
| NGINX | No | nginx team: "We currently do not have near term plan to support it" https://community.nginx.org/t/http3-webtransport-webtransport-support-in-nginx/5500 |
| HAProxy | No | Feature request open since 2023: https://github.com/haproxy/haproxy/issues/2256 |
| Caddy | Not for upstream WT | Caddy's h3 is quic-go-based, but `reverse_proxy` does not forward WebTransport sessions upstream — **UNVERIFIED** (secondary blog claim; no primary Caddy doc found). Irrelevant here since Caddy is being removed. |

Consequence: terminating WebTransport **in the Go process** is the pragmatic choice; the only alternative is Envoy in alpha, which adds a second QUIC hop and an incomplete feature.

---

## 3. Recommended pin + architecture

### 3.1 Pins

```go
require (
    github.com/quic-go/quic-go          v0.62.0   // exact: paired with webtransport-go v0.13.0
    github.com/quic-go/webtransport-go  v0.13.0   // draft-16, session flow control
    github.com/caddyserver/certmagic    v0.25.6
    github.com/libdns/cloudflare        v0.2.2
)
```

- Do **not** stay on v0.59.0 (indirect): it is from 2026-01-11 and predates the draft-16 flow-control support that webtransport-go v0.13 relies on.
- Do **not** blindly bump quic-go to v0.64.1 on day one. Source inspection shows the APIs webtransport-go uses still exist and the relevant defaults are unchanged (I diffed `config.go` and `params.go`), so v0.64.1 will *probably* compile and work — but that is **UNVERIFIED** (no Go 1.26 toolchain available in this environment to build the pair). Land v0.62.0, then bump in a separate commit with `go test ./...` + an integration handshake.
- Both modules require `go 1.26.0`; repo has `go 1.26.3`. certmagic v0.25.6 also requires `go 1.26.0`.

### 3.2 Topology

```
                    ┌──────────────────── one Go process ─────────────────────┐
 DNS-only (grey)    │  net.ListenUDP(":443")                                   │
 wt.cogioi.io.vn ──►│    webtransport.Server.Serve(udp)                        │
   UDP/443          │      ├── Extended CONNECT /wt ──► *Session (zone sim)     │
                    │      └── all other H3 streams ──► http3.Server.Handler   │
                    │                                                          │
 DNS A (grey or     │  net.Listen("tcp", ":443")                               │
 orange)            │    http.Server.ListenAndServeTLS("", "")                 │
 api.cogioi.io.vn ─►│      Handler = same mux (REST, h1.1/h2)                  │
   TCP/443          │                                                          │
                    │  certmagic cache ── GetCertificate ── both tls.Configs   │
                    └──────────────────────────────────────────────────────────┘
```

- One `mux` serves REST on TCP h1/h2 **and** on H3 (same handler), plus `/wt`.
- One certificate source; two ALPN clones (`h3` vs `h2`/`http/1.1`).
- Optional: advertise H3 to browsers with `http3.Server.SetQUICHeaders` on TCP responses — but behind the Cloudflare proxy this is pointless (browsers speak H3 to the edge, not to origin).

### 3.3 Suggested config for 2000 sessions

| Setting | Value | Why |
|---|---|---|
| `quic.Config.MaxIdleTimeout` | 60–120 s | default 30 s drops mobile/background clients; 20 Hz traffic keeps it alive anyway |
| `KeepAlivePeriod` | 20 s | keeps NAT mappings; redundant while snapshotting at 20 Hz |
| `MaxIncomingStreams` / `MaxIncomingUniStreams` | 8–16 | one bidi stream per session + H3 requests; do not leave 100 |
| `MaxStreamReceiveWindow` | 256 KiB | |
| `MaxConnectionReceiveWindow` | 1–2 MiB | caps worst-case buffered bytes at 2–4 GiB for 2000 sessions |
| `webtransport.Config.MaxIncomingStreams/Uni` | 4–8 | draft-16 session flow control; if left nil, the session falls back to the QUIC-level 100-stream limit |
| `webtransport.Config.MaxIncomingData` | 1 MiB | |
| `EnableDatagrams` | true | required by WebTransport (set automatically by the WT server, but set it explicitly for H3) |
| `Allow0RTT` | **false** | 0-RTT is replayable; a 20 Hz input stream must not replay |
| `DisablePathMTUDiscovery` | false | DPLPMTUD is cheap (<10 probes/connection) |
| kernel `net.core.rmem_max`/`wmem_max` | 7340032 | quic-go asks for 7 MiB; default kernel max is too small |

### 3.4 Deployment checklist

- UDP/443 open inbound (and outbound ICMP for PMTU; QUIC degrades but works without it).
- `sysctl -w net.core.rmem_max=7340032 net.core.wmem_max=7340032` (and persist it in the container/host).
- Certificate volume mounted and writable (`/data/certs`).
- Split DNS: REST host may stay orange-clouded; the WebTransport host must be **DNS-only** (or Spectrum). Prefer a distinct hostname (`wt.…`) so REST keeps WAF/DDoS.
- `CheckOrigin` must explicitly allow the game origin(s), including local dev.
- Container needs no `NET_ADMIN`; needs no privileged port if you bind `:443` as root or grant `CAP_NET_BIND_SERVICE`.

---

## 4. Minimal code sketch

Verified against the APIs of quic-go v0.62.0 / webtransport-go v0.13.0 / certmagic v0.25.6. It is a sketch: error handling is minimized, and it has **not been compiled here** (no Go 1.26 toolchain in this environment).

```go
package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/cloudflare"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---- 1. Certificates: certmagic + ACME DNS-01 (Cloudflare) -------------
	var magic *certmagic.Config
	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(certmagic.Certificate) (*certmagic.Config, error) {
			return magic, nil
		},
	})
	magic = certmagic.New(cache, certmagic.Config{
		Storage: &certmagic.FileStorage{Path: "/data/certs"}, // mounted volume
	})
	issuer := certmagic.NewACMEIssuer(magic, certmagic.ACMEIssuer{
		CA:                      certmagic.LetsEncryptProductionCA,
		Email:                   "ops@cogioi.io.vn",
		Agreed:                  true,
		DisableHTTPChallenge:    true, // DNS-01 only
		DisableTLSALPNChallenge: true,
		DNS01Solver: &certmagic.DNS01Solver{
			DNSManager: certmagic.DNSManager{
				DNSProvider: &cloudflare.Provider{APIToken: os.Getenv("CF_DNS_API_TOKEN")},
			},
		},
	})
	magic.Issuers = []certmagic.Issuer{issuer}

	hosts := []string{"api.cogioi.io.vn", "wt.cogioi.io.vn"}
	bootCtx, cancelBoot := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelBoot()
	if err := magic.ManageSync(bootCtx, hosts); err != nil {
		log.Fatalf("certmagic: %v", err)
	}

	// ---- 2. Same cert source, two ALPN profiles ---------------------------
	tlsTCP := magic.TLSConfig()                            // sets GetCertificate
	tlsTCP.NextProtos = []string{"h2", "http/1.1"}         // no acme-tls/1: DNS-01 only
	tlsQUIC := http3.ConfigureTLSConfig(magic.TLSConfig()) // NextProtos = ["h3"]

	// ---- 3. Shared handler: REST over TCP h1/h2 AND over H3 ---------------
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// ---- 4. QUIC: HTTP/3 + WebTransport on ONE UDP socket -----------------
	h3 := &http3.Server{
		Addr:      ":443",
		TLSConfig: tlsQUIC,
		Handler:   mux,
		QUICConfig: &quic.Config{
			MaxIdleTimeout:             90 * time.Second,
			KeepAlivePeriod:            20 * time.Second,
			MaxIncomingStreams:         16,
			MaxIncomingUniStreams:      16,
			MaxStreamReceiveWindow:     256 << 10,
			MaxConnectionReceiveWindow: 2 << 20,
			EnableDatagrams:            true,
		},
	}
	wt := &webtransport.Server{
		H3: h3,
		Config: &webtransport.Config{
			MaxIncomingStreams:    8,
			MaxIncomingUniStreams: 8,
			MaxIncomingData:       1 << 20,
		},
		ApplicationProtocols: []string{"gu.realtime.v1"},
		CheckOrigin: func(r *http.Request) bool {
			switch r.Header.Get("Origin") {
			case "https://cogioi.io.vn", "http://localhost:3000":
				return true
			}
			return false
		},
	}
	mux.HandleFunc("/wt", func(w http.ResponseWriter, r *http.Request) {
		sess, err := wt.Upgrade(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		go serveSession(sess)
	})

	udp, err := net.ListenUDP("udp", &net.UDPAddr{Port: 443})
	if err != nil {
		log.Fatal(err)
	}

	// ---- 5. TCP: REST over HTTPS (HTTP/1.1 + HTTP/2) ----------------------
	tcp := &http.Server{Addr: ":443", Handler: mux, TLSConfig: tlsTCP}

	errCh := make(chan error, 2)
	go func() { errCh <- wt.Serve(udp) }()               // UDP/443: H3 + WebTransport
	go func() { errCh <- tcp.ListenAndServeTLS("", "") }() // TCP/443: h1/h2, certs from GetCertificate

	select {
	case <-ctx.Done():
	case err := <-errCh:
		log.Printf("server stopped: %v", err)
	}

	// ---- 6. Shutdown: app drain -> TCP -> QUIC ----------------------------
	// 6a. flip an app-level "draining" flag and notify sessions here.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = tcp.Shutdown(shutdownCtx) // graceful h1/h2
	_ = wt.Close()                // closes QUIC conns, then the wrapped http3.Server
	_ = udp.Close()
}

// serveSession is the zone-sim session loop (echo stub).
// One reliable bidi stream carrying length-prefixed frames is the recommended
// shape: 1024 B datagram budget is too small for snapshots; streams are not
// subject to the browser datagram cap.
func serveSession(sess *webtransport.Session) {
	defer sess.CloseWithError(0, "")
	for {
		str, err := sess.AcceptStream(sess.Context())
		if err != nil {
			return
		}
		go func(str *webtransport.Stream) {
			defer str.Close()
			buf := make([]byte, 64<<10)
			for {
				n, err := str.Read(buf)
				if err != nil {
					return
				}
				if _, err := str.Write(buf[:n]); err != nil {
					return
				}
			}
		}(str)
	}
}
```

Notes on the sketch:

- `webtransport.Server.init()` calls `webtransport.ConfigureHTTP3Server(s.H3)` automatically; calling it yourself is optional but harmless if you want H3 settings before `Serve`.
- Do not call `h3.ListenAndServe()` — the WT server already owns UDP/443.
- `tlsQUIC` must not be the same pointer as `tlsTCP`: `http3.ConfigureTLSConfig` clones, but the TCP config needs h2/h1.1 ALPN. Sharing a single base `*tls.Config` with h2+h3 ALPN is possible but conflates session-ticket settings; two clones from the same certmagic source is simpler.

---

## 5. Constraints/limits table (compact)

| # | Constraint | Limit / default | Impact at 2000 sessions | Mitigation |
|---|---|---|---|---|
| 1 | Cloudflare orange cloud | no H3 to origin, `CONNECT` blocked | **WebTransport cannot connect at all** | DNS-only WT host, or Spectrum (Enterprise) |
| 2 | Origin IP exposure when grey-clouded | raw UDP/443 to origin | no WAF/bot/DDoS shield on the realtime host | provider L3/L4 protection, Spectrum, or accept risk |
| 3 | Per-connection receive window | 15 MiB default | ~30 GiB worst-case buffered | set `MaxConnectionReceiveWindow` 1–2 MiB |
| 4 | Per-connection streams | 100 bidi / 100 uni | 200k streams possible; memory + scheduling | set 8–16 at QUIC and WT layers |
| 5 | Datagram payload | 1024 B (Chrome/WebKit); MTU ceiling 1452 B | snapshots >1 KiB need fragmentation/streams | one bidi stream + length-prefixed frames; datagrams only for small inputs |
| 6 | Datagram send path | "not optimized" (#3766); no size query (#4259) | 40k datagrams/s at 20 Hz is risky | measure; prefer streams |
| 7 | Congestion control | CUBIC, not configurable | no BBR/low-latency tuning | accept, or upstream patch |
| 8 | One UDP socket, one receive goroutine | per `quic.Transport` | one core does all packet demux at ~40k pps | horizontal processes + SO_REUSEPORT |
| 9 | Idle timeout | 30 s, keepalive off | backgrounded clients dropped | 60–120 s + keepalive |
| 10 | Accept queue / packet queues | 32 pending conns; 256 pkts/conn | handshake bursts drop | rate-limit reconnects; watch metrics |
| 11 | TLS anti-amplification | server first flight ≤ 3× client Initial (~3600 B) | fat cert chain adds an RTT or fails | ECDSA leaf + short chain |
| 12 | WT graceful shutdown | no `Shutdown`; `Close` is abrupt | deploy kills live sessions | app-level drain + rolling restart |
| 13 | Draft-16 | browsers/servers may drift | breakage window on browser updates | pin, integration-test Chrome/Safari/Firefox |
| 14 | UDP-blocked networks | QUIC needs UDP/443 | some players cannot connect (map accepts this) | none, by map decision |

---

## 6. Open risks

1. **Cloudflare is a hard blocker, not a tuning issue.** The map already lists "DDoS cho host realtime DNS-only (lộ IP origin)" as unspecified; this research confirms it is *unavoidable* unless Spectrum is bought. Decide before implementation.
2. **No measured memory/handshake data at 2000 sessions.** The 30 GiB / 4 GiB figures are flow-control ceilings, not RSS. Required before claiming scale (consistent with `docs/mmo/scaling.md`): a load test with 2000 synthetic WT clients, 20 Hz, measuring RSS, GC, CPU, p99 RTT, and packet loss.
3. **Datagram plan needs to change or be measured.** The map wants datagrams for snapshot/input. With a 1024 B client write cap and an unoptimized quic-go DATAGRAM path, treat datagrams as a small-input channel only; snapshots should stay on a reliable stream with a per-message framing header (or accept application-level loss handling).
4. **Draft drift.** draft-16 is in WG Last Call (not an RFC). quic-go explicitly warns that browsers and servers may update at different times and break for a transition period. Add a browser-matrix integration test (Chrome, Firefox, Safari 26.4+) to CI/nightly; the fallback is "no game" because the map removes WebSocket.
5. **Single receive goroutine / no multi-queue.** At 40k pps, one core is likely enough, but there is no built-in sharding. If CPU-bound, run N processes behind SO_REUSEPORT or accept the ceiling.
6. **No graceful WebTransport shutdown.** Rolling deploys will drop sessions unless you implement app-level drain and a reconnect/takeover path (the map's "reconnect/takeover" ticket must own this).
7. **Certificate handling.** DNS-01 requires a Cloudflare API token in the process; scope it to `Zone.DNS:Edit` on the one zone. Persisting `/data/certs` is mandatory to avoid LE rate limits. A wildcard cert (`*.cogioi.io.vn`) is possible via DNS-01 and would simplify hostnames, at the cost of a wider key scope.
8. **Cert size / handshake.** Keep the chain short (ECDSA, no extra intermediate): QUIC's anti-amplification limit means a >3.6 KB first flight costs an extra RTT.
9. **Go/toolchain drift.** Both libraries require Go 1.26.0; quic-go supports only the latest two Go releases. A Go upgrade cadence must be part of maintenance.
10. **No external escape hatch.** If terminating QUIC in Go fails operationally, the only alternative (Envoy) is alpha and incomplete; there is no mature WT terminator to fall back to.

---

## 7. Explicitly UNVERIFIED

- Compile/run of `webtransport-go v0.13.0` against `quic-go v0.64.1` (no Go 1.26 toolchain here; recommendation is to pin v0.62.0).
- Real memory per session and CPU per packet at 2000 sessions.
- Whether Chrome/Firefox/Safari on the current release all speak draft-16 (the libraries claim draft-16; browser draft level not checked in browser source).
- Caddy `reverse_proxy` WebTransport behaviour (secondary blog claim only; not relevant to the plan).
- Any claim about Cloudflare supporting WebTransport on a non-standard product tier beyond Spectrum (no primary doc found).

---

## 8. Sources (primary)

- quic-go docs: https://quic-go.net/docs/ , https://quic-go.net/docs/quic/server/ , https://quic-go.net/docs/quic/datagrams/ , https://quic-go.net/docs/quic/optimizations/ , https://quic-go.net/docs/quic/congestion-control/ , https://quic-go.net/docs/http3/server/ , https://quic-go.net/docs/webtransport/ , https://quic-go.net/docs/webtransport/server/
- quic-go source v0.62.0: https://github.com/quic-go/quic-go/tree/v0.62.0 (`config.go`, `interface.go`, `transport.go`, `connection.go`, `internal/protocol/params.go`, `internal/protocol/protocol.go`, `internal/ackhandler/sent_packet_handler.go`, `internal/wire/datagram_frame.go`, `http3/server.go`)
- webtransport-go v0.13.0: https://github.com/quic-go/webtransport-go/tree/v0.13.0 (`server.go`, `session.go`, `config.go`, `go.mod`)
- certmagic v0.25.6: https://github.com/caddyserver/certmagic/tree/v0.25.6 (`config.go`, `acmeissuer.go`, `solvers.go`, `filestorage.go`, `storage.go`)
- libdns/cloudflare v0.2.2: https://github.com/libdns/cloudflare/blob/v0.2.2/provider.go
- autocert: https://pkg.go.dev/golang.org/x/crypto/acme/autocert
- Cloudflare: https://developers.cloudflare.com/speed/optimization/protocol/http3/ , https://developers.cloudflare.com/fundamentals/concepts/traffic-flow-cloudflare/ , https://developers.cloudflare.com/spectrum/ , https://community.cloudflare.com/t/unable-to-forward-webtransport-request-via-loadbalancer-to-origin/335218
- Envoy: https://github.com/envoyproxy/envoy/pull/45630 , https://github.com/envoyproxy/envoy/issues/41981 , https://github.com/envoyproxy/envoy/pull/47270
- NGINX: https://community.nginx.org/t/http3-webtransport-webtransport-support-in-nginx/5500
- HAProxy: https://github.com/haproxy/haproxy/issues/2256
- WebTransport draft: https://datatracker.ietf.org/doc/draft-ietf-webtrans-http3/
- Chrome datagram cap: https://chromium.googlesource.com/chromium/src/+/ff66885fb0d785437f82f9ab7aaedd3d3dcc8812
- WebKit datagram cap: https://github.com/WebKit/WebKit/pull/70152
- Browser support: https://caniuse.com/webtransport , https://webstatus.dev/features/webtransport
- Go stdlib TLS behaviour: https://cs.opensource.google/go/go/+/master:src/net/http/server.go
- RFC 9000 anti-amplification: https://datatracker.ietf.org/doc/html/rfc9000#section-8
- RFC 9221 DATAGRAM: https://datatracker.ietf.org/doc/html/rfc9221
