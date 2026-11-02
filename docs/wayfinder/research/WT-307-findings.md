# WT-307 — Ingress UDP/443: Cloudflare DNS-only, ACME DNS-01, Lightsail

> Ticket: #307 (part of #303). Scope: how to expose a WebTransport endpoint from a single
> AWS Lightsail box once Caddy is removed and Go terminates TLS itself.
> Verified 2026-11-02 against primary sources (AWS docs, Cloudflare docs, quiche issue,
> libdns/certmagic/quic-go source). Claims that could not be pinned to a primary source
> are marked **UNVERIFIED**.

## TL;DR (verdict table)

| # | Question | Verdict |
|---|----------|---------|
| 1 | UDP/443 on Lightsail | **Allowed.** Console → Networking → IPv4 Firewall → *Custom* → UDP → 443. No security groups on Lightsail; the instance firewall is the only L3 filter, allow-only, 60 rules per family. |
| 2 | DNS-only `rt.cogioi.io.vn` | **Works.** Grey cloud = plain A record, client connects straight to the Lightsail IP. Cloudflare edge certs/Universal SSL do **not** apply (and are not needed). |
| 3 | Cloudflare proxy for WebTransport | **Impossible today.** HTTP/3 is edge-only ("HTTP/3 connection to the origin is not yet supported"), and quiche has no WebTransport support with no committed timeline. |
| 4 | Spectrum fronting UDP/443 | **No.** UDP Spectrum requires Enterprise + paid add-on **and** BYOIP (/24). HTTPS Spectrum also cannot do HTTP/3. Not usable here. |
| 5 | ACME DNS-01 in Go | **certmagic + libdns/cloudflare** (same stack Caddy uses). One scoped token (`Zone:DNS:Edit` + `Zone:Read`, zone-limited). Persist `/app/certs` on a named volume. |
| 6 | Origin exposure after grey cloud | **Partially mitigable.** All L7 protections (WAF, rate limit, bot fight, Always-Use-HTTPS) stop applying to `rt`. UDP/443 must stay world-open; only host-level nftables + app-level limits help. Volumetric DDoS is **not** mitigable on Lightsail. |

**Recommended topology in one line:** keep `api.cogioi.io.vn` orange-clouded for L7, add
`rt.cogioi.io.vn` as a DNS-only A record → same box, same Go binary, same cert (DNS-01),
TCP/443 for HTTP + UDP/443 for WebTransport, port 80 closed.

---

## 0. Current state (repo facts used below)

| Fact | Source in repo |
|---|---|
| Backend listens plain HTTP on `:8080`, single `http.Server` (`ListenAndServe`), no TLS | `backend/cmd/server/main.go:930-945` |
| Container runs as non-root `USER app` (cannot bind :443 inside the container) | `backend/Dockerfile:20-36` |
| Compose publishes `127.0.0.1:8080:8080`; Caddy service is commented out but its volumes remain | `docker-compose.yml` |
| Caddy did TLS + SSE flush for `api.cogioi.io.vn` only | `Caddyfile` |
| No WebTransport/QUIC code anywhere yet (greenfield) | `grep -rn "webtransport\|quic-go" backend` → 0 hits |
| Cookies scoped to `.cogioi.io.vn` (`SameSite=None; Secure`) | `docker-compose.yml` backend env |
| No `/api/health` route; `/api/dictionary` is the cheapest public probe | `backend/internal/server/server.go:493` |
| Host has no `ufw`/nftables rules in repo; deploy only does `compose up` | `.github/workflows/deploy-backend.yml` |

---

## 1. AWS Lightsail — opening UDP/443

**Verdict: supported, no special request needed.** Lightsail has no security groups. Each
instance has two independent firewalls (IPv4 + IPv6), allow-only: you cannot write deny
rules, and traffic that arrives via the *private* IP bypasses them entirely. Protocol
options include UDP, and the docs' own example opens `DNS (UDP) - UDP - 53`, which proves
UDP ingress is a first-class rule type.
([instance firewalls](https://docs.aws.amazon.com/lightsail/latest/userguide/understanding-firewall-and-port-mappings-in-amazon-lightsail.html),
[rules reference](https://docs.aws.amazon.com/lightsail/latest/userguide/amazon-lightsail-firewall-rules-reference.html))

### Console path
`Lightsail → Instances → <instance> → Networking → IPv4 Firewall → + Add rule`

| Application | Protocol | Port | Source |
|---|---|---|---|
| Custom | UDP | 443 | Any IPv4 (`0.0.0.0/0`) |
| Custom | UDP | 443 | Any IPv6 (`::/0`) — only if you publish AAAA |
| HTTPS | TCP | 443 | Any (see hardening §5) |

### CLI equivalent

```bash
aws lightsail open-instance-public-ports \
  --instance-name <instance> \
  --port-info fromPort=443,toPort=443,protocol=udp,cidrs=0.0.0.0/0

# verify
aws lightsail get-instance-port-states --instance-name <instance> \
  --query 'portStates[?fromPort==`443`]'
```

`protocol` accepts `tcp | udp | all | icmp | icmpv6`, ports `0-65535`; `cidrs` and
`ipv6Cidrs` are separate lists.
([open-instance-public-ports](https://docs.aws.amazon.com/cli/latest/reference/lightsail/open-instance-public-ports.html))

### Hard limits / gotchas

1. **60 rules per family.** "a firewall can have 60 inbound rules for IPv4 traffic and 60
   inbound rules for IPv6 traffic". Console allows 30 source IPs per rule; API/CLI allows
   60. Keep one `0.0.0.0/0` rule per port, not many small CIDRs.
2. **Most-permissive wins.** Adding `443/TCP from 0.0.0.0/0` silently widens a narrower
   rule for the same port. You cannot "narrow" by adding another rule.
3. **The firewall is not a security group.** No referencing other resources, no
   stateful egress rules (all outbound allowed), no VPC SG semantics. Any "deny this IP"
   logic must live in nftables on the instance.
4. **Public IP ≠ instance IP.** The instance only sees its private `172.26.x.x` address;
   the public IPv4 is 1:1 NAT'd in front. Source IPs are preserved, so per-IP nftables
   rate limits still work.
5. **Use a static IP before publishing the A record.** A Lightsail public IP is not
   guaranteed stable across stop/start; a static IP is free while attached to a running
   instance but the address changes when it is first attached.
   ([static IPs](https://docs.aws.amazon.com/lightsail/latest/userguide/understanding-static-ip-addresses-in-amazon-lightsail.html))
6. **No Shield Advanced, no L4 DDoS scrubbing.** Lightsail has baseline DDoS protection
   only. Inbound is not billed, but origin egress during an attack is.
7. **IPv6 is optional and off unless enabled.** If `rt` only has an A record, do not bother
   with the IPv6 firewall — but then IPv6-only clients fall back to IPv4.

---

## 2. Cloudflare — DNS-only `rt.cogioi.io.vn`

**Verdict: correct and required for WebTransport.**

Create in the `cogioi.io.vn` zone:

| Type | Name | Content | Proxy status | TTL |
|---|---|---|---|---|
| A | `rt` | `<Lightsail static IPv4>` | **DNS only** (grey cloud) | Auto (300 s is fine) |

Grey cloud means Cloudflare answers with your real IP and does not sit in the path:
"Cloudflare responds with your server's actual IP address and does not route HTTP/HTTPS
traffic through its network."
([Proxy status](https://developers.cloudflare.com/dns/manage-dns-records/reference/proxied-dns-records/))

### Why the orange cloud cannot work

Two independent blockers, both primary:

1. **Cloudflare does not speak HTTP/3 to origins.** The HTTP/3 setting is explicitly
   "for connection between the user and Cloudflare. HTTP/3 connection to the origin is
   not yet supported."
   ([HTTP/3 docs](https://developers.cloudflare.com/speed/optimization/protocol/http3/))
2. **Cloudflare's QUIC stack has no WebTransport.** quiche issue #1114 is open since
   2021-12-11; a Cloudflare maintainer (LPardue) wrote on 2025-08-26: *"We're big fans of
   WebTransport and would like to get support for it in quiche at some point. However, we
   can't commit to any timeline or prioritization of the effort at this time."* A user in
   the same thread reports `ERR_METHOD_NOT_SUPPORTED` with the proxy enabled.
   ([quiche#1114](https://github.com/cloudflare/quiche/issues/1114))

So the browser would complete QUIC to the edge and then be unable to reach your origin's
WebTransport endpoint. Grey cloud is the only option.

### Cross-hostname gotchas (same zone/proxied siblings)

| Area | Effect on grey-clouded `rt` | Action |
|---|---|---|
| **Universal SSL** | Covers only proxied hostnames ("free TLS certificates automatically issued for all **proxied** hostnames"). No CF cert is issued for `rt` — and none is needed, since clients do a real TLS/QUIC handshake with your Go server. ([Universal SSL](https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/)) | Issue your own cert (DNS-01, §4). |
| **HSTS `includeSubDomains`** | Cloudflare serves the HSTS header only for proxied hostnames, **but** the browser policy from `api`/`www` propagates to `rt` via `includeSubDomains`: "Subdomains are inaccessible if they do not support HTTPS." Once a browser caches it, a broken/expired cert on `rt` is a hard failure with no click-through. ([HSTS](https://developers.cloudflare.com/ssl/edge-certificates/additional-options/http-strict-transport-security/)) | Ship `rt` TLS **before** enabling `includeSubDomains` (or before announcing it). Do monitor expiry. |
| **Always Use HTTPS / redirect rules / WAF / rate limiting / Bot Fight / Under Attack** | None apply. All of these are proxy-layer features; Cloudflare itself lists "apply your Cloudflare product configurations (such as WAF rules, caching, and redirect rules)" as a *benefit of proxying*. | Expect zero L7 protection on `rt`; enforce auth/rate limits in Go. |
| **`alt-svc`** | Injected per-response by the edge only for proxied hostnames. `rt` gets no `alt-svc` from Cloudflare — and must not, because `rt`'s QUIC is your own origin, not the edge. Alt-Svc is keyed per origin, so a cached `api` H3 route does not leak to `rt`. | Serve your own `Alt-Svc` for `rt` (quic-go's `SetQUICHeaders`). |
| **Port 80** | Cloudflare's HTTP→HTTPS redirect does not exist off-proxy, so `http://rt...:80` would hit your origin in plaintext (or connection-refused). | Do **not** publish container port 80. |
| **CAA** | Zone-level CAA applies to *any* CA, proxied or not. If the zone has CAA, it must allow your ACME CA or DNS-01 issuance fails. ([CAA](https://developers.cloudflare.com/ssl/edge-certificates/caa-records/)) | Check `dig CAA cogioi.io.vn`; allow `letsencrypt.org` (and `pki.goog` / `sectigo.com` if you keep multi-CA). |
| **Cookies** | `AUTH_COOKIE_DOMAIN=.cogioi.io.vn` + `SameSite=None; Secure` means auth cookies from `api` are also sent to `rt`. Convenient, but widen the blast radius of any cookie bug. | Plan a short-lived WT ticket instead of relying on the session cookie. **UNVERIFIED**: whether the browser attaches cookies to the HTTP/3 CONNECT — test in Chrome/Safari before depending on it. |
| **DNSSEC** | Signing happens at Cloudflare regardless of proxy status. | No change. |

Also: do not add a `CNAME rt → <proxied host>`; a grey-clouded CNAME to a proxied host
still reveals the target and can produce a certificate mismatch.

---

## 3. Cloudflare Spectrum — can it front UDP/443?

**Verdict: no, not for this deployment.** Three independent blockers:

1. **Plan gating.** UDP Spectrum requires **Enterprise + paid add-on**:

   | | Free | Pro | Business | Enterprise |
   |---|---|---|---|---|
   | UDP | No | No | No | Paid add-on |

   ([Protocols per plan](https://developers.cloudflare.com/spectrum/protocols-per-plan/))
2. **BYOIP required for UDP.** "Spectrum UDP applications are supported with BYOIP" and
   "The smallest prefixes that Cloudflare currently supports is /24 for IPv4 and /48 for
   IPv6 … BYOIP does not come standard with Spectrum."
   ([Limitations](https://developers.cloudflare.com/spectrum/reference/limitations/),
   [BYOIP](https://developers.cloudflare.com/spectrum/about/byoip/))
   You do not own a /24, so this alone is fatal.
3. **Even the HTTPS variant cannot carry WebTransport.** "At the moment, HTTPS applications
   do not support HTTP/3" (same Limitations page). WebTransport is HTTP/3, so no.

Additional limits worth knowing if the situation ever changes: no UDP packet
fragmentation at the edge (fragmented datagrams are dropped — relevant because QUIC
handshake packets are large and rely on PMTU), and Universal SSL is incompatible with
Spectrum (needs Advanced/Custom certs). Custom WAF rules do not apply; only IP Access
Rules.

**Pricing:** Cloudflare does not publish a list price for Spectrum; it is an Enterprise
contract add-on. Community reports cluster around **$1/GB** plus an Enterprise minimum.
**UNVERIFIED** — do not quote this in a plan.

**Conclusion:** Spectrum is not a way to hide the origin while serving WebTransport.

---

## 4. ACME DNS-01 inside Go, via the Cloudflare API

**Verdict: `github.com/caddyserver/certmagic` + `github.com/libdns/cloudflare`.** This is
the same library Caddy used, so behaviour/on-disk layout is familiar, and it supports every
libdns provider. `lego` is the alternative (also solid, provider-CLI oriented); pick one,
not both.

certmagic's README documents exactly this Cloudflare case:

```go
import "github.com/libdns/cloudflare"

certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
    DNSManager: certmagic.DNSManager{
        DNSProvider: &cloudflare.Provider{APIToken: token},
    },
}
```

Enabling DNS-01 disables HTTP-01/TLS-ALPN for that issuer, so **no inbound port 80 and no
public ACME reachability is required**. Certmagic persists account key, certs, OCSP
staples and locks; persistent storage is a hard requirement ("ephemeral storage will likely
lead to rate limiting").
([certmagic README](https://github.com/caddyserver/certmagic#dns-challenge),
[Storage](https://github.com/caddyserver/certmagic#storage))

### Token permissions

Create a **scoped API token** (not the Global API Key):

| Scope | Permission | Why |
|---|---|---|
| Zone → `cogioi.io.vn` | `Zone → DNS → Edit` | create/delete `_acme-challenge` TXT |
| Zone → `cogioi.io.vn` | `Zone → Zone → Read` | resolve zone ID (libdns lists zones) |

Cloudflare's permission names are `DNS Read` / `DNS Write` / `Zone Read` / `Zone Write`
under the **Zone** category ([permissions reference](https://developers.cloudflare.com/fundamentals/api/reference/permissions/)).
libdns/cloudflare's `Provider` mirrors lego's model: `APIToken` needs `Zone.DNS:Write`,
and an **optional** `ZoneToken` (Zone:Read) exists specifically for the case where the DNS
token is scoped to one zone and therefore cannot list zones:

```go
// github.com/libdns/cloudflare/provider.go
APIToken  string // API token with Zone.DNS:Write (can be scoped to single Zone if ZoneToken is also provided)
ZoneToken string // Optional Zone:Read token (global scope)
```

Start with the single zone-scoped token. If certmagic fails with "zone not found" during
challenge setup, add a second Zone:Read token as `ZoneToken` — that is a documented,
supported workaround, not a bug.

Everything else (Account, Workers, Cache Purge, DNS Firewall) must be absent.

### Wiring into the server (`main.go`)

```go
// after PORT handling, before srv.ListenAndServe
magic := certmagic.NewDefault()
magic.Storage = &certmagic.FileStorage{Path: cfg.CertDir} // /app/certs
certmagic.DefaultACME.Email = cfg.ACMEEmail
certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA // staging for first rollout
certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
    DNSManager: certmagic.DNSManager{DNSProvider: &cloudflare.Provider{APIToken: cfg.CFDNSToken}},
}
if err := magic.ManageSync(ctx, []string{"api.cogioi.io.vn", "rt.cogioi.io.vn"}); err != nil { ... }

tlsCfg := magic.TLSConfig()
tlsCfg.NextProtos = append([]string{"h3"}, tlsCfg.NextProtos...) // ← required for QUIC
```

**Gotcha (verified in source):** certmagic itself only prepends `h2`/`http/1.1`; it never
adds `h3` (`certmagic.go`, `HTTPS()`). If `h3` is missing from `NextProtos`, the QUIC
handshake fails ALPN negotiation and WebTransport is unreachable while HTTPS still looks
fine — a silent, confusing failure.

### Persisting certs in a docker volume

- Mount a named volume at a path the **non-root** `app` user can write. Docker initialises
  a fresh named volume from the image directory, so create and `chown` it in the Dockerfile:

  ```dockerfile
  RUN mkdir -p /app/certs && chown -R app:app /app/certs
  ```

- Compose: `certmagic_data:/app/certs` + `volumes: certmagic_data:`
- Do not mount a host path owned by root; certmagic will fail to write and re-issue on every
  restart (Let's Encrypt "too many certificates" after ~5 duplicates/week).
- One cert covering both names is enough (they share a zone and both are A records).

### Keeping the token out of the repo and logs

| Do | Don't |
|---|---|
| Put `CLOUDFLARE_DNS_API_TOKEN=…` in `backend/.env` on the Lightsail host (already `env_file:` and gitignored by `backend/.gitignore:1`; the root `.gitignore` also ignores `/.env`). `git reset --hard` does not delete untracked files, so the deploy keeps it. | Never put a literal token in `docker-compose.yml`. |
| `chmod 600 backend/.env`, `chown` the deploy user. | Never add it to GitHub Actions `script:` (ssh-action interpolates it into a shell command; even with masking, it lands in shell history/`ps`). |
| Optional stronger: Docker `secrets:` from `/etc/guworld/cf-dns-token` (root:root 0400), read into `cfg.CFDNSToken` at startup. `libdns`/lego also support `*_FILE` env vars. | Don't `docker compose config` in CI with the token exported — it renders interpolated values. |
| Log only "cert obtained for N names", never the token/env dump. | Don't enable gin debug-mode request dumps in prod. |

Note: `docker inspect` on the container always shows env vars to anyone with docker
access — that is acceptable on a single-admin Lightsail box, but it is why the token file
should be root-owned if you use the secret-file route.

---

## 5. Origin exposure on a single Lightsail box

Once `rt` is grey-clouded, `<static IP>` is public in DNS forever. Realistic posture:

### Mitigable (worth doing)

| Mitigation | Effort | Effect |
|---|---|---|
| **nftables default-drop** ingress; allow only 22 (from admin IPs), 443/tcp, 443/udp | S | Removes everything else from scanners. Lightsail's firewall can't deny, so this is the only host-level filter. |
| **Restrict TCP/443 to Cloudflare IP ranges** (`api` only; `rt` needs UDP only once `rt`'s HTTPS is unused) | M | Origin's TCP services become invisible to scanners. Needs a refresh job for CF ranges. **Caveat:** breaks direct `https://rt…` debugging and any non-CF client that tries TCP fallback. |
| **Published-port hygiene**: stop publishing 5432/6379 (currently `"5432:5432"`, `"6379:6379"` in compose!) | S | Highest value fix in this whole ticket — see risk note below. |
| **UDP/443 per-source rate limit in nftables** + `net.core.rmem_max=7340032` / `wmem_max=7340032` | S | Blunts single-IP floods and fixes QUIC throughput (kernel buffers are too small by default per quic-go). ([quic-go optimizations](https://quic-go.net/docs/quic/optimizations/)) |
| **QUIC handshake limits** in `quic.Config`: `HandshakeIdleTimeout` ~5 s, `MaxIdleTimeout` ~30 s, small `MaxIncomingStreams`/`MaxIncomingUniStreams` per connection, `KeepAlivePeriod` only if needed | S | Caps per-connection cost; prevents slowloris-style QUIC exhaustion. |
| **Auth before work**: require a short-lived WT ticket (signed token) in the CONNECT URL/handshake; validate Origin; one session per player | M | Cheap, and the only real defence for UDP since WAF can't help. |
| **App-level connection caps**: max concurrent WT sessions per IP, per account, global | M | Protects the process when bandwidth is not yet saturated. |
| **`userland-proxy: false`** in `/etc/docker/daemon.json` + restart docker | S | UDP is published through `docker-proxy` userspace by default, which is a pps bottleneck; iptables DNAT is much faster. Restarting docker restarts all containers — schedule it. **UNVERIFIED** perf numbers; verify with a load test. |
| **Alerts**: UDP/443 pps + conntrack count + egress bytes | M | You cannot mitigate what you cannot see. |

Sketch (tune before applying; `@cf_tcp` is a `flags interval` set refreshed from
`https://www.cloudflare.com/ips-v4`):

```nft
table inet guworld {
  set cf_tcp { type ipv4_addr; flags interval; }

  chain input {
    type filter hook input priority filter; policy drop;
    ct state established,related accept
    iif lo accept
    ip protocol icmp accept
    ip6 nexthdr ipv6-icmp accept

    tcp dport 22 ip saddr { 203.0.113.10 } accept      # admin only
    tcp dport 443 ip saddr @cf_tcp accept              # api via Cloudflare only

    udp dport 443 ct state new \
      meter quic_flood { ip saddr limit rate over 200/second } drop
    udp dport 443 accept
  }
}
```

### NOT realistically mitigable

- **Volumetric DDoS on UDP/443.** A few Gbps of spoofed QUIC Initials fills the Lightsail
  uplink; there is no scrubbing tier to buy on Lightsail. Once saturated, everything on the
  box (including the API) is down. The only real answer is a different front door
  (Spectrum/Shield/Cloudflare Tunnel), and Spectrum is ruled out in §3.
- **IP discovery.** It is in public DNS. No amount of hardening changes that.
- **Reflection/amplification abuse of your host.** QUIC servers are not good reflectors
  (address validation / anti-amplification limits), so you are more likely to be the
  *victim* than the amplifier — but you can't stop someone from aiming at you.
- **fail2ban for QUIC.** Not practical: QUIC is encrypted from the first packet and
  quic-go exposes almost no pre-auth, per-IP failure signal in logs. Fail2ban would see
  nothing actionable. Use nftables meters + app-level limits instead; keep fail2ban for
  SSH. (Judgement based on transport properties; **UNVERIFIED** as a formal statement.)
- **Cloudflare L7 protections for `rt`.** Nothing applies. You can only *choose* to route
  the game's fallback transport (WebSocket) through `api`, where they do apply.

### ⚠️ Flagged for the map issue (not this ticket's scope)

`docker-compose.yml` currently publishes **PostgreSQL 5432 and Redis 6379 on all
interfaces**. If that is live on the Lightsail host, the database is already reachable from
the internet, and every consideration in §5 is moot. This must be checked before the
ingress change (expected fix: `127.0.0.1:5432:5432`, `127.0.0.1:6379:6379`, or drop the
mappings entirely — the backend reaches them over the compose network).

---

## 6. Concrete diff (proposal only — do not apply in this ticket)

### 6.1 `docker-compose.yml`

```diff
   backend:
     image: ${BACKEND_IMAGE:-ghcr.io/vippro67/guworld-backend:latest}
     container_name: guworld-backend
     restart: always
     ports:
-      - "127.0.0.1:8080:8080"
+      # Go serves TLS on :8443 inside the container (non-root cannot bind :443)
+      - "443:8443/tcp"
+      - "443:8443/udp"          # WebTransport / HTTP3
+      # optional: plain HTTP for local debugging only, never public
+      - "127.0.0.1:8080:8080"
+    volumes:
+      - certmagic_data:/app/certs
     env_file:
       - ./backend/.env
     environment:
       - PORT=8080
+      - TLS_ADDR=:8443
+      - TLS_DOMAINS=api.cogioi.io.vn,rt.cogioi.io.vn
+      - ACME_EMAIL=...
+      - CERT_DIR=/app/certs
+      # CLOUDFLARE_DNS_API_TOKEN comes from ./backend/.env — never inline here
       - GIN_MODE=release
...
-  # caddy:
-  #   image: caddy:2-alpine
-  #   ...
+  # caddy service intentionally deleted (Go terminates TLS + serves QUIC)

 volumes:
   redis_data:
   postgres_data:
+  certmagic_data:
-  caddy_data:
-  caddy_config:
```

Also fix in the same PR (not ingress, but adjacent and cheap):
`"5432:5432"` → `"127.0.0.1:5432:5432"`, `"6379:6379"` → `"127.0.0.1:6379:6379"`.

### 6.2 `backend/Dockerfile`

```diff
 RUN apk --no-cache add ca-certificates tzdata \
     && addgroup -S app \
     && adduser -S -G app app \
     && mkdir -p /app/storage/images \
+    && mkdir -p /app/certs \
     && chown -R app:app /app
...
-EXPOSE 8080
+EXPOSE 8080 8443 8443/udp
```

### 6.3 `.github/workflows/deploy-backend.yml`

```diff
     paths:
       - 'backend/**'
       - 'docker-compose.yml'
-      - 'Caddyfile'
       - '.github/workflows/deploy-backend.yml'
```

```diff
             export BACKEND_IMAGE="ghcr.io/vippro67/guworld-backend:${{ github.sha }}"
             docker compose pull backend

-            docker compose up -d --no-deps backend
+            # A leftover Caddy container would still hold :443 and make the
+            # new backend crash-loop on bind. Remove it explicitly, then up.
+            docker rm -f guworld-caddy 2>/dev/null || true
+            docker compose up -d --no-deps --remove-orphans backend
+
+            # Fail the deploy if 443 is not actually ours
+            ss -lntp | grep -q ':443 ' || { echo "TCP/443 not bound"; exit 1; }
+            ss -lnup | grep -q ':443 ' || { echo "UDP/443 not bound"; exit 1; }
+
+            # Smoke test through the new listener (public route, no auth)
+            curl -fsS --resolve api.cogioi.io.vn:443:127.0.0.1 \
+              https://api.cogioi.io.vn/api/dictionary >/dev/null
```

Also required once, on the host (outside the workflow):
`sysctl -w net.core.rmem_max=7340032 net.core.wmem_max=7340032` (persist in
`/etc/sysctl.d/99-quic.conf`), the nftables ruleset, `userland-proxy: false`, and the
Lightsail UDP/443 rule. The workflow cannot do these safely.

### 6.4 App-side (separate implementation ticket)

- Gin behind Cloudflare directly: `r.SetTrustedProxies(<cloudflare CIDRs>)` (or read
  `CF-Connecting-IP`). Without Caddy, the reverse-proxy hop disappears; the default
  "trust all proxies" makes `c.ClientIP()` spoofable, which silently breaks every
  `RateLimitByIP` in `server.go`.
- `webtransport-go` requires `http3.Server.EnableDatagrams = true`.
- Keep WebSocket on `api.cogioi.io.vn` as the fallback transport: Cloudflare supports
  proxied WebSockets, and corporate networks that block UDP need that path.
  ([WebSockets](https://developers.cloudflare.com/network/websockets/))

---

## 7. Recommended topology (exact)

```
Browser (https://www.cogioi.io.vn)
  │
  ├── HTTPS / WSS  ──► Cloudflare proxy (orange) ──► api.cogioi.io.vn
  │                      A → <Lightsail static IP> (proxied)
  │                      → origin TCP/443 ──► Go TLS (cert via DNS-01)
  │
  └── WebTransport ──► rt.cogioi.io.vn  (DNS only, grey)
                         A → <Lightsail static IP>
                         → origin UDP/443 ──► Go HTTP/3 + WebTransport
                                              (same cert, same binary)
```

| Item | Value |
|---|---|
| DNS — `api` | A `<static IP>`, **Proxied** |
| DNS — `rt` | A `<static IP>`, **DNS only**, TTL 300 |
| Lightsail firewall | TCP 443 ← 0.0.0.0/0 (or CF ranges), UDP 443 ← 0.0.0.0/0, TCP 22 ← admin IP; nothing else |
| Container ports | `443:8443/tcp`, `443:8443/udp` (Go binds `:8443` as non-root) |
| Port 80 | **closed** (DNS-01 needs no HTTP challenge) |
| Certs | Let's Encrypt via certmagic + DNS-01, SANs `api.cogioi.io.vn`, `rt.cogioi.io.vn`, stored in `certmagic_data:/app/certs` |
| TLS mode at Cloudflare | Full (strict) — origin presents a publicly-trusted LE cert |
| HSTS | keep `includeSubDomains` **off** until `rt` is verified end-to-end |
| UDP fallback | WebSocket on `api` (through Cloudflare), not on `rt` |

**Do we still need Cloudflare in front of `api` when Go terminates TLS?** Not strictly —
DNS-only would work — but keep it: it is the only WAF/DDoS/caching layer you have, and
Full (strict) with a DNS-01 origin cert is exactly the supported configuration. Just be
clear that it no longer hides the origin, because `rt` publishes the same IP.

---

## 8. Mitigations ranked by effort

| Rank | Change | Effort | Blocks |
|---|---|---|---|
| 1 | Close 5432/6379 to loopback (or remove mappings) | 5 min | catastrophic |
| 2 | Add Lightsail UDP/443 + TCP/443 rules, use a static IP | 5 min | feature |
| 3 | Delete Caddy service; `--remove-orphans`; port/ss smoke test in deploy | 30 min | outage |
| 4 | certmagic DNS-01 + scoped token in `backend/.env`, `certmagic_data` volume | 2–4 h | feature |
| 5 | nftables default-drop + QUIC buffer sysctls + `userland-proxy: false` | 1–2 h | hardening |
| 6 | UDP/443 per-source meter + app-level session/IP caps + WT ticket auth | 1–2 d | abuse |
| 7 | Gin trusted-proxies/CF-Connecting-IP correction | 1 h | rate-limit correctness |
| 8 | Alerts on pps/conntrack/egress; cert-expiry monitoring | 2–4 h | ops |
| 9 | TCP/443 restricted to Cloudflare ranges | 2 h + refresh job | origin stealth |

---

## 9. Open / UNVERIFIED

1. **Spectrum price** — no public list price; ~$1/GB community figure is **UNVERIFIED**.
2. **Cookies on WebTransport CONNECT** — does the browser send the `.cogioi.io.vn` cookie?
   Test in Chrome + Safari 26 before designing around it; prefer ticket auth.
3. **fail2ban-for-QUIC** — assessed as impractical, not tested; no primary source asserts it.
4. **`userland-proxy: false` perf delta for this workload** — plausible bottleneck, measure.
5. **Whether one zone-scoped token is enough for libdns/cloudflare** — expected yes; the
   library documents the `ZoneToken` fallback if zone listing 403s.
6. **Lightsail IPv6** — untested; only relevant if you publish AAAA for `rt`.

---

## 10. Sources

- AWS Lightsail, instance firewalls — https://docs.aws.amazon.com/lightsail/latest/userguide/understanding-firewall-and-port-mappings-in-amazon-lightsail.html
- AWS Lightsail, firewall rules reference (UDP example) — https://docs.aws.amazon.com/lightsail/latest/userguide/amazon-lightsail-firewall-rules-reference.html
- AWS CLI `open-instance-public-ports` — https://docs.aws.amazon.com/cli/latest/reference/lightsail/open-instance-public-ports.html
- AWS Lightsail, static IPs — https://docs.aws.amazon.com/lightsail/latest/userguide/understanding-static-ip-addresses-in-amazon-lightsail.html
- Cloudflare, proxy status — https://developers.cloudflare.com/dns/manage-dns-records/reference/proxied-dns-records/
- Cloudflare, Universal SSL — https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/
- Cloudflare, HSTS (`includeSubDomains`) — https://developers.cloudflare.com/ssl/edge-certificates/additional-options/http-strict-transport-security/
- Cloudflare, CAA records — https://developers.cloudflare.com/ssl/edge-certificates/caa-records/
- Cloudflare, HTTP/3 (edge-only to origin) — https://developers.cloudflare.com/speed/optimization/protocol/http3/
- Cloudflare, WebSockets (proxied support) — https://developers.cloudflare.com/network/websockets/
- Cloudflare API token permissions — https://developers.cloudflare.com/fundamentals/api/reference/permissions/
- Cloudflare Spectrum, protocols per plan — https://developers.cloudflare.com/spectrum/protocols-per-plan/
- Cloudflare Spectrum, limitations (UDP/BYOIP, no HTTP/3 for HTTPS) — https://developers.cloudflare.com/spectrum/reference/limitations/
- Cloudflare Spectrum, BYOIP (/24 minimum) — https://developers.cloudflare.com/spectrum/about/byoip/
- Cloudflare Spectrum, settings by plan — https://developers.cloudflare.com/spectrum/reference/settings-by-plan/
- quiche issue #1114 (WebTransport, no timeline) — https://github.com/cloudflare/quiche/issues/1114
- certmagic README (DNS-01, storage) — https://github.com/caddyserver/certmagic
- certmagic source, `HTTPS()` NextProtos (no `h3`) — https://github.com/caddyserver/certmagic/blob/master/certmagic.go
- libdns/cloudflare provider (`APIToken`/`ZoneToken`) — https://github.com/libdns/cloudflare/blob/master/provider.go
- libdns core — https://github.com/libdns/libdns
- lego Cloudflare provider (`CF_DNS_API_TOKEN`, `CF_ZONE_API_TOKEN`, `*_FILE`) — https://go-acme.github.io/lego/dns/cloudflare/
- quic-go, UDP buffer sysctls — https://quic-go.net/docs/quic/optimizations/
