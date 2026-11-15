---
name: architecture-planning
description: Plan multi-file changes, map codebase boundaries, and design incremental implementations. Use for large features or refactors.
---

# Architecture & planning
1. Read `AGENTS.md`, README, build scripts and relevant modules before proposing a design. Never assume an undocumented architecture.
2. Record goal, current behavior, invariants, dependencies, risks, alternatives and phased implementation steps.
3. For GuWorld: preserve the Go authoritative backend; identify Phaser scene ownership, shared protocol, server simulation and persistence boundaries.
4. Break work into reversible PR-sized slices, each with tests, rollout and acceptance criteria.
5. Prefer a runnable vertical slice before broad rewrites; require explicit approval for destructive migrations.
## Output
Provide component map, file-level change list, interfaces/protocol changes, sequencing, checks, and rollback. No unsupported claims of completion.
