---
name: code-review
description: Review diffs and pull requests for correctness, regressions, maintainability and risky changes.
---

# Code review
1. Determine base/target diff and understand intent. Read changed files and callers.
2. Search for correctness bugs, bad state transitions, races, injection, memory leaks, missed cleanup and breaking compatibility.
3. Confirm tests cover meaningful edge cases, not just happy path.
4. Report only substantiated findings sorted by severity, with file:line and concrete failure conditions.
5. Clearly distinguish verified issues, hypotheses and suggestions. If no issues found, state review scope and test limitations.
## GuWorld focus
Check actor ownership, authoritative movement/combat, duplicate events, replay protection, cooldown enforcement and snapshot ordering.
