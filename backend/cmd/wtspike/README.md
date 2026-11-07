# wtspike — WebTransport vs WebSocket probe (#310)

Throwaway prototype for [Prototype: spike WebTransport end-to-end Go ↔ browser](https://github.com/VipPro67/GuWorld/issues/310).
It sends synthetic snapshots over **WebTransport (QUIC/UDP)** and **WebSocket (TCP)** at the same rate/size, and measures:

- snapshot inter-arrival p50/p95/p99/max,
- jitter (mean |Δgap|),
- loss % (sequence gaps),
- RTT p50/p95 via 1 Hz ping/pong,
- bytes/s and event-stream delivery.

Not production code: no auth, self-signed cert, no game state.

## Run

```bash
cd backend
go run ./cmd/wtspike -addr :9443 -rate 20 -size 220
```

Open <https://localhost:9443>. Accept the self-signed certificate.

## Option B (recommended for Chrome): real Let's Encrypt cert

Chrome 132+ only trusts QUIC certificates from public roots or the
`CACertificates` enterprise policy; local CAs are painful. Use a DNS-01 cert for
a subdomain you own (no A record needed):

```bash
# Cloudflare token: Zone:DNS:Edit + Zone:Read for cogioi.io.vn
CF_DNS_API_TOKEN='<token>' go run ./cmd/wtspike \
  -addr :9443 \
  -acme-domain spike.cogioi.io.vn \
  -acme-email ops@cogioi.io.vn
```

Then map the hostname to localhost in Chrome (quit all Chrome first):

```bash
google-chrome-stable \
  --host-resolver-rules="MAP spike.cogioi.io.vn 127.0.0.1" \
  --user-data-dir=/tmp/wtspike-chrome \
  https://spike.cogioi.io.vn:9443
```

The certificate lives in `/tmp/wtspike-acme`; `-acme-staging` exists but its
certs are **not** browser-trusted, so it cannot be used for the spike.

## Option A (local CA, OS-level trust)

> **WebTransport + local certificates (important):** Chrome 119+ no longer
> allows bypassing certificate errors for QUIC/WebTransport, including
> `--ignore-certificate-errors` and `--ignore-certificate-errors-spki-list`.
> The only reliable local setup is a CA trusted by the system:
>
> ```bash
> cd backend/cmd/wtspike
> ./gen-cert.sh /tmp/wtcert
> # Arch/CachyOS OS trust (optional, for curl/openssl):
> sudo trust anchor --store /tmp/wtcert/ca.crt
> # Chrome 132+ trusts QUIC CAs only via this enterprise policy:
> CERT_B64=$(openssl x509 -in /tmp/wtcert/ca.crt -outform der | base64 -w0)
> sudo mkdir -p /etc/opt/chrome/policies/managed
> printf '{"CACertificates":["%s"]}\n' "$CERT_B64" | sudo tee /etc/opt/chrome/policies/managed/wtspike-ca.json >/dev/null
> cd ../.. && go run ./cmd/wtspike -addr :9443 -cert /tmp/wtcert/srv.crt -key /tmp/wtcert/srv.key
> ```
>
> Restart Chrome, check `chrome://policy` shows `CACertificates`, then open
> <https://localhost:9443>. To undo: `sudo rm /etc/opt/chrome/policies/managed/wtspike-ca.json`.
> The page logs the exact failure to the browser console (`[wtspike] ...`) and
> shows it in the status badge.


- `-rate` default 20 (Hz); page can request 20 or 60 per connection.
- `-size` default 220 bytes; page can request 220/1024/1200.
  Safari's `datagrams.maxDatagramSize` is capped at 1024, so use **1000** when
  testing Safari (the page warns if the requested size exceeds the negotiated
  max; oversized datagrams are silently dropped by the browser).

Smoke test without a browser:

```bash
go run ./cmd/wtspike -addr :9443 -smoke
# logs "smoke OK"
```

## Measurement protocol (ticket acceptance)

Run each leg for **≥ 10 minutes**, keep the tab visible (background throttling
skews gaps), then press **Download CSV**.

1. WiFi: 5 min idle + 5 min walking/attacking in a real game tab in parallel.
2. 4G: phone hotspot → laptop (or run the server on the laptop and open the
   page on the same laptop).
3. Same scenarios for `WebTransport` and `WebSocket`.
4. Network switch: while connected, disable WiFi so traffic moves to 4G.
   Record whether the WT session survives (`transport.closed` fires) and how
   long the page takes to reconnect. (Expected per #306: WT does **not**
   migrate; the session dies and must be re-joined.)

Compare CSV rows: WT should show equal or lower gap p95/p99 and lower jitter on
the same network; loss % should reflect sequence gaps (datagrams are
unreliable, so some loss is expected — WebSocket/TCP will hide it as latency).

## Notes / gotchas

- Self-signed cert: WT handshake requires a trusted certificate for `localhost`;
  either accept the browser warning or run Chrome with
  `--ignore-certificate-errors` for the spike profile only.
- One process serves TCP `:9443` (HTTPS + WS) and UDP `:9443` (QUIC + WT),
  mirroring the production topology from #305/#315.
- Datagram payload limit is negotiated per browser (Chrome ~1200 B, Safari
  1024 B). The page shows `maxDatagramSize`.
- WS mode adds TCP/TLS + frame overhead; it is a reference for the current
  transport, not a byte-for-byte equal channel.
