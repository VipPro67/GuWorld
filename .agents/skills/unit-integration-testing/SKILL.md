---
name: unit-integration-testing
description: Build and run unit, integration and contract tests for Go, TypeScript and network protocols.
---

# Unit, integration & contract tests
1. Discover actual test framework from the repository; do not assume paths or scripts.
2. Write deterministic tests with isolated fixtures and controlled time/randomness; clean resources.
3. Cover success, invalid input, boundary values, concurrency, timeout and disconnect.
4. For Go use `go test ./...` only when in a Go module; consider `go test -race ./...` when relevant.
5. For TypeScript discover package manager then run the project's actual test scripts.
6. For protocol tests assert schema, backward compatibility, tick/sequence ordering and forbidden client-supplied authority fields.
7. Report test command, counts/results and what was not run.
