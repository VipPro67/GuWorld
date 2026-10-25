/**
 * One Phaser.Game per DOM host.
 *
 * React StrictMode and Next Fast Refresh can run mount -> cleanup -> mount in
 * quick succession. Keeping the registry keyed by the host element guarantees
 * that a leftover game (and its canvas) is destroyed before a new one attaches.
 */
export interface DestroyableGame {
  destroy(removeCanvas?: boolean): void;
}

const attachedGames = new WeakMap<object, DestroyableGame>();

export function attachGame(host: object, game: DestroyableGame): DestroyableGame {
  const existing = attachedGames.get(host);
  if (existing && existing !== game) {
    existing.destroy(true);
  }
  attachedGames.set(host, game);
  return game;
}

export function getAttachedGame(host: object): DestroyableGame | null {
  return attachedGames.get(host) ?? null;
}

/** Destroy the host's game. A stale game never destroys a newer instance. */
export function detachGame(host: object, game?: DestroyableGame): boolean {
  const current = attachedGames.get(host);
  if (!current) return false;
  if (game && current !== game) return false;
  attachedGames.delete(host);
  current.destroy(true);
  return true;
}
