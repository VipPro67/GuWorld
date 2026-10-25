import type { LoadedWorldMap } from "./MapLoader.ts";
import type { WorldPoint } from "./WorldUnits.ts";

interface Node {
  x: number;
  y: number;
  g: number;
  f: number;
  parent: Node | null;
}

const MAX_NODES = 20_000;
const DIRECTIONS: Array<[number, number, number]> = [
  [1, 0, 1], [-1, 0, 1], [0, 1, 1], [0, -1, 1],
  [1, 1, Math.SQRT2], [1, -1, Math.SQRT2], [-1, 1, Math.SQRT2], [-1, -1, Math.SQRT2],
];

/**
 * Client-only A* preview/path executor. The server still validates each
 * waypoint as a normal world.move_target; this never grants authority.
 */
export function findPath(map: LoadedWorldMap, start: WorldPoint, goal: WorldPoint, maxNodes = MAX_NODES): WorldPoint[] | null {
  if (!inside(map, start) || !inside(map, goal)) return null;
  if (map.isSolid(goal.x, goal.y)) return null;
  if (map.isSolid(start.x, start.y)) return null;
  if (lineClear(map, start, goal)) return [{ ...goal }];

  const startX = Math.floor(start.x);
  const startY = Math.floor(start.y);
  const goalX = Math.floor(goal.x);
  const goalY = Math.floor(goal.y);
  if (startX === goalX && startY === goalY) return [{ ...goal }];

  const open: Node[] = [{ x: startX, y: startY, g: 0, f: heuristic(startX, startY, goalX, goalY), parent: null }];
  const best = new Map<string, number>([[`${startX},${startY}`, 0]]);
  const closed = new Set<string>();
  let expanded = 0;

  while (open.length > 0 && expanded < maxNodes) {
    let bestIndex = 0;
    for (let index = 1; index < open.length; index += 1) {
      if (open[index].f < open[bestIndex].f) bestIndex = index;
    }
    const current = open.splice(bestIndex, 1)[0];
    expanded += 1;
    const currentKey = `${current.x},${current.y}`;
    if (closed.has(currentKey)) continue;
    closed.add(currentKey);
    if (current.x === goalX && current.y === goalY) {
      return simplify(map, start, goal, reconstruct(current, start, goal));
    }
    for (const [dx, dy, cost] of DIRECTIONS) {
      const nx = current.x + dx;
      const ny = current.y + dy;
      if (!walkable(map, nx, ny)) continue;
      if (dx !== 0 && dy !== 0) {
        if (!walkable(map, current.x + dx, current.y) || !walkable(map, current.x, current.y + dy)) continue;
      }
      const key = `${nx},${ny}`;
      if (closed.has(key)) continue;
      const g = current.g + cost;
      const known = best.get(key);
      if (known !== undefined && known <= g) continue;
      best.set(key, g);
      open.push({ x: nx, y: ny, g, f: g + heuristic(nx, ny, goalX, goalY), parent: current });
    }
  }
  return null;
}

function reconstruct(goal: Node, start: WorldPoint, exactGoal: WorldPoint): WorldPoint[] {
  const cells: WorldPoint[] = [];
  let node: Node | null = goal;
  while (node) {
    cells.push({ x: node.x + 0.5, y: node.y + 0.5 });
    node = node.parent;
  }
  cells.reverse();
  if (cells.length > 0) cells[0] = { ...start };
  cells[cells.length - 1] = { ...exactGoal };
  return cells;
}

function simplify(map: LoadedWorldMap, start: WorldPoint, goal: WorldPoint, cells: WorldPoint[]): WorldPoint[] {
  const result: WorldPoint[] = [];
  let anchor = start;
  for (let index = 0; index < cells.length; index += 1) {
    const next = cells[index];
    if (index === cells.length - 1 || !lineClear(map, anchor, cells[index + 1])) {
      result.push(next);
      anchor = next;
    }
  }
  if (result.length === 0 || Math.hypot(result[0].x - start.x, result[0].y - start.y) > 1e-6) {
    result.unshift({ ...start });
  }
  if (result.length === 0 || Math.hypot(result[result.length - 1].x - goal.x, result[result.length - 1].y - goal.y) > 1e-9) {
    result[result.length - 1] = { ...goal };
  }
  return result;
}

export function lineClear(map: LoadedWorldMap, from: WorldPoint, to: WorldPoint): boolean {
  const distance = Math.hypot(to.x - from.x, to.y - from.y);
  const steps = Math.max(1, Math.ceil(distance / 0.2));
  for (let step = 1; step <= steps; step += 1) {
    const ratio = step / steps;
    const x = from.x + (to.x - from.x) * ratio;
    const y = from.y + (to.y - from.y) * ratio;
    if (map.isSolid(x, y)) return false;
  }
  return true;
}

function walkable(map: LoadedWorldMap, x: number, y: number): boolean {
  return x >= 0 && y >= 0 && x < map.width && y < map.height && !map.isSolid(x + 0.5, y + 0.5);
}

function inside(map: LoadedWorldMap, point: WorldPoint): boolean {
  return Number.isFinite(point.x) && Number.isFinite(point.y) && point.x >= 0 && point.y >= 0 && point.x <= map.width && point.y <= map.height;
}

function heuristic(x: number, y: number, goalX: number, goalY: number): number {
  return Math.hypot(goalX - x, goalY - y);
}
