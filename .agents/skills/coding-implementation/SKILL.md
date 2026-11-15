---
name: coding-implementation
description: Implement small, maintainable changes with tests and regression checks. Use for coding tasks and feature delivery.
---

# Coding implementation
1. Inspect the existing relevant files and test/build commands. Check `git status --short` before edits.
2. Reproduce bug or describe expected user-facing behavior and acceptance tests.
3. Make the smallest cohesive change. Preserve public APIs unless migration is required; avoid unrelated formatting.
4. Add input validation, error handling, resource cleanup and focused tests.
5. Run targeted tests, typecheck, lint, then wider checks as appropriate; report exactly which checks ran.
6. Summarize modified files, design decision, residual risks and manual verification.
## Safety
Never overwrite local user changes or commit/push without an explicit request.
