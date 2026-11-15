---
name: deployment-verification
description: Deploy and validate dev/staging releases safely with CI, Docker and rollback procedures.
---

# Deployment verification
1. Identify intended environment and deploy mechanism from repo. Treat production deploy as a separate explicitly authorized action.
2. Check clean build, tests, migrations, env names (never values), image tags and compatibility of frontend/backend protocol.
3. For Docker Compose: inspect `docker compose config` and `docker compose config --images` without exposing secrets.
4. Use immutable image identifiers where possible; run health checks, smoke tests, logs and WebSocket admission test after rollout.
5. Plan rollback for schema/protocol changes, monitor error rate/latency/tick lag and record release identifier.
6. Never assume SSH access or overwrite host keys; verify host fingerprint out of band if mismatch.
7. Report actual deployed target, executed commands, check results and rollback readiness. Never claim deployment without evidence.
