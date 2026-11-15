---
name: verification-gate
description: Verify completion claims with concrete proof, rerun relevant tests and report limitations.
---

# Verification gate
Before declaring work complete:
1. Map each acceptance criterion to a proof artifact: test, browser action, log, metric or code inspection.
2. Run the relevant command/check in the current environment; do not report hypothetical test outcomes as passed.
3. Validate negative paths and regression risks; inspect `git diff --check`, changed files and unintended edits.
4. If a check cannot run due to missing environment or credentials, call it NOT VERIFIED and provide a reproducible command.
5. Give concise evidence, known failures and remaining work; no false '100% done'.
