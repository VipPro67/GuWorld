---
name: systematic-debugging
description: Diagnose crashes, errors, incorrect behavior or regressions using hypotheses and evidence.
---

# Systematic debugging
1. Collect exact symptom, environment, reproduction steps, recent changes and logs; scrub tokens and PII.
2. Reproduce first, or state why it could not be reproduced.
3. Trace data flow end-to-end. Form hypotheses ranked by evidence; test one variable at a time.
4. Identify root cause before patching. Add a regression test that fails before and passes after when feasible.
5. Verify fix with the same reproduction plus adjacent edge cases; inspect console/server logs and network messages.
## GuWorld checks
Separate rendering, input, WebSocket transport, tick scheduling, server validation, persistence and client interpolation. Diagnose disconnect vs admission vs state desync distinctly.
## Output
Symptom / root cause / evidence / patch / tests / remaining unknowns.
