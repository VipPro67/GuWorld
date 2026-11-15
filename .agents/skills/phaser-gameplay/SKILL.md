---
name: phaser-gameplay
description: Implement and troubleshoot Phaser top-down 2.5D scenes, input, assets and combat UI.
---

# Phaser gameplay
1. Inspect installed Phaser version and current scenes before coding; do not assume a framework API.
2. Keep rendering/client presentation separate from backend authoritative game state.
3. Ensure WASD + click-to-move behaviors coexist; prevent input propagation when UI is focused.
4. Keep entity spawn/despawn and asset preloading idempotent; cleanup listeners/timers/tweens on scene shutdown.
5. Use object pooling only after profiling; prefer texture atlases and scoped preloading based on real measurements.
6. Check camera transforms, depth order, viewport resizing, effects and device pixel ratio.
7. Add smoke tests and manual steps for low FPS and network jitter.
## Combat
Treat target lock, skillshot and AoE as client intent with server-side hit validation, not trusted client damage.
