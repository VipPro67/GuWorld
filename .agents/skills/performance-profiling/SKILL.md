---
name: performance-profiling
description: Profile frontend render, Go runtime and network under load before optimizing.
---

# Performance profiling
1. Establish baseline environment, representative workload and metrics (FPS, p95 frame time, server tick latency, heap, GC, throughput, CCU).
2. Create a reproducible load scenario with fixed actor/entity counts and duration.
3. Gather traces/profiles: browser Performance/Memory, Go pprof/trace if enabled, server metrics and network counters.
4. Identify the highest-cost bottleneck; optimize one variable and compare before/after under the same scenario.
5. Avoid micro-optimizations without measurable impact; guard against functional regressions.
6. Test degradation at realistic latency and congestion; never load-test production without permission.
## Output
Environment / baseline / profiling evidence / proposed change / after measurement / tradeoffs.
