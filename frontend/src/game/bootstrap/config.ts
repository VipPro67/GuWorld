import Phaser from "phaser";
import { BootScene } from "../scenes/BootScene.ts";
import { PreloadScene } from "../scenes/PreloadScene.ts";
import { WorldScene } from "../scenes/WorldScene.ts";
import type { WorldRuntime } from "../core/WorldRuntime.ts";
import {
  PHASER_PARENT_ID,
  PREVIEW_WORLD_HEIGHT,
  PREVIEW_WORLD_WIDTH,
  WORLD_RUNTIME_KEY,
} from "./constants.ts";

export { PHASER_PARENT_ID, PREVIEW_WORLD_HEIGHT, PREVIEW_WORLD_WIDTH, WORLD_RUNTIME_KEY };

export function createWorldGameConfig(
  parent: string | HTMLElement = PHASER_PARENT_ID,
  runtime?: WorldRuntime | null,
): Phaser.Types.Core.GameConfig {
  return {
    // AUTO picks WebGL when the browser supports it and falls back to Canvas.
    type: Phaser.AUTO,
    parent,
    title: "GuWorld Realtime Preview",
    version: "mmo-preview.1",
    backgroundColor: "#07110d",
    transparent: false,
    antialias: true,
    roundPixels: true,
    banner: false,
    disableContextMenu: true,
    // Rendering only: collision, hits and rewards stay authoritative on the Go
    // world server. No Phaser physics system is enabled on purpose.
    physics: undefined,
    callbacks: runtime
      ? { preBoot: (game) => game.registry.set(WORLD_RUNTIME_KEY, runtime) }
      : undefined,
    scale: {
      mode: Phaser.Scale.RESIZE,
      autoCenter: Phaser.Scale.CENTER_BOTH,
      width: "100%",
      height: "100%",
      parent,
    },
    fps: {
      target: 60,
      forceSetTimeOut: false,
    },
    render: {
      antialias: true,
      powerPreference: "high-performance",
      batchSize: 4096,
    },
    scene: [BootScene, PreloadScene, WorldScene],
  };
}
