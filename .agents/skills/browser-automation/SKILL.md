---
name: browser-automation
description: Inspect web UIs with Playwright, reproduce user journeys, console errors and network issues; use for browser-based debug.
---

# Browser automation
Prerequisite: actual browser tool, e.g. Playwright MCP or Playwright CLI, must be connected. A skill does not install a browser.
1. Confirm the app URL, running server, account/test data, and allowed testing environment.
2. Navigate using semantic selectors where possible; inspect DOM/accessibility snapshot and browser console.
3. Exercise a real journey (load world, move, use skill, reconnect), capture screenshots only when needed.
4. Inspect failed requests, WebSocket connection lifecycle and console errors. Do not treat visual success as server correctness.
5. Prefer repeatable tests with explicit assertions and stable selectors; avoid arbitrary sleeps.
6. Do not submit real payments or destructive actions; use test accounts and test environments.
## Phaser canvas
DOM selectors cannot identify sprites inside canvas. Use game debug overlays/test hooks, screenshots, keyboard/mouse actions, and corroborating server telemetry. Do not assert exact world state from pixels alone.
## Output
URL, actions, observed evidence, failed checks and reproduction steps.
