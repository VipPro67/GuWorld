import Phaser from "phaser";
import type { WorldRuntime } from "../core/WorldRuntime.ts";
import { createWorldGameConfig } from "./config.ts";
import { attachGame, detachGame } from "./gameRegistry.ts";

/**
 * Create the realtime Phaser game inside a React-owned host element.
 *
 * This module must only be imported from client code (the React host imports
 * it lazily in an effect) because Phaser touches `window` at import time.
 */
export function createPhaserGame(host: HTMLElement, runtime?: WorldRuntime | null): Phaser.Game {
  // Never stack a second canvas on the same host, even if a previous cleanup
  // was skipped by Fast Refresh.
  detachGame(host);
  const game = new Phaser.Game(createWorldGameConfig(host, runtime));
  attachGame(host, game);
  return game;
}

export function destroyPhaserGame(host: HTMLElement, game?: Phaser.Game): boolean {
  return detachGame(host, game);
}
