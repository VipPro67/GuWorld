---
name: security-audit
description: Audit auth, secrets, dependencies, input validation and game exploit surfaces.
---

# Security audit
1. Establish assets, attack surface and trust boundaries; inspect code and deployment configuration.
2. Check authentication/authorization, session handling, WebSocket upgrade/origin, rate limits, input validation, injection and CSRF where applicable.
3. For MMO: validate action legality, cooldown, spatial constraints, item ownership and trade invariants server-side.
4. Check secrets in source/logs; never print tokens, private keys, credentials or production data.
5. Use approved dependency scanners already present (e.g. `govulncheck`, `npm audit`) and record versions/findings. Avoid auto-fixes without reviewing impact.
6. For each finding report severity, threat scenario, affected location, evidence, remediation and test.
7. Do not perform attacks against systems without authorization.
