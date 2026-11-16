export interface WorldPositionEvent {
  id: string;
  action?: "left";
  username: string;
  avatar_url: string;
  x: number;
  y: number;
  region_id: string;
  world_map_id: string;
  map_instance_id?: string;
  ts: number;
}

export interface WorldMapPlayerPosition {
  id: string;
  username: string;
  avatar_url?: string;
  x: number;
  y: number;
}

export interface WorldMapPlayerSnapshot {
  x: number;
  y: number;
  vision_radius: number;
  world_map_id?: string;
  map_instance_id?: string;
  players_on_map?: WorldMapPlayerPosition[];
}

/** Applies one scoped position delta to a map snapshot without mutating it. */
export function applyWorldPosition<T extends WorldMapPlayerSnapshot>(
  snapshot: T,
  event: WorldPositionEvent,
  localPlayerID: string,
): T {
  if (
    !event.id
    || event.id === localPlayerID
    || !Number.isFinite(event.x)
    || !Number.isFinite(event.y)
    || (snapshot.world_map_id && event.world_map_id !== snapshot.world_map_id)
    || (event.map_instance_id || "") !== (snapshot.map_instance_id || "")
  ) return snapshot;

  const players = (snapshot.players_on_map || []).filter((player) => player.id !== event.id);
  if (event.action === "left") return { ...snapshot, players_on_map: players };

  const isVisible = Math.abs(event.x - snapshot.x) <= snapshot.vision_radius
    && Math.abs(event.y - snapshot.y) <= snapshot.vision_radius;
  if (isVisible) {
    players.push({
      id: event.id,
      username: event.username,
      avatar_url: event.avatar_url,
      x: event.x,
      y: event.y,
    });
  }
  return { ...snapshot, players_on_map: players };
}
