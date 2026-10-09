export type HotbarPreferences = {
  pinnedIds: string[];
  compact: boolean;
  visible: boolean;
  hotkeys: Record<string, string>;
};

const DEFAULT_PINS = ["inventory", "inventory-gu", "cultivation", "character", "world-atlas"];

export function normalizeHotbarPreferences(value: unknown, actionIds: string[]): HotbarPreferences {
  const stored = value && typeof value === "object" ? value as Partial<HotbarPreferences> : {};
  const validIds = new Set(actionIds);
  const storedPins = Array.isArray(stored.pinnedIds) ? stored.pinnedIds
    .filter((id): id is string => typeof id === "string" && validIds.has(id))
    .map((id) => id === "practice-room" && validIds.has("cultivation") ? "cultivation" : id)
    : DEFAULT_PINS.filter((id) => validIds.has(id));
  const pinnedIds = [...new Set(storedPins)];
  const hotkeys: Record<string, string> = {};
  if (stored.hotkeys && typeof stored.hotkeys === "object") {
    for (const [id, key] of Object.entries(stored.hotkeys)) {
      if (validIds.has(id) && typeof key === "string" && key.length <= 24) hotkeys[id] = key;
    }
  }
  return {
    pinnedIds,
    compact: typeof stored.compact === "boolean" ? stored.compact : false,
    visible: typeof stored.visible === "boolean" ? stored.visible : true,
    hotkeys,
  };
}

export function reorderHotbarIds(ids: string[], fromId: string, toId: string): string[] {
  const from = ids.indexOf(fromId);
  const to = ids.indexOf(toId);
  if (from < 0 || to < 0 || from === to) return ids;
  const next = [...ids];
  next.splice(to, 0, next.splice(from, 1)[0]);
  return next;
}

export type HotbarActionHandlers = {
  setView: (id: string) => void;
  openGeneralSettings: () => void;
  openTeleport: () => void;
  openWorldAtlas: () => void;
  openTeleportDirectory: () => void;
  openTeleportBuild: () => void;
  openDivination: () => void;
  openActivityLog: () => void;
};

export function dispatchHotbarAction(id: string, handlers: HotbarActionHandlers) {
  switch (id) {
    case "general-settings": handlers.openGeneralSettings(); return;
    case "teleport": handlers.openTeleport(); return;
    case "world-atlas": handlers.openWorldAtlas(); return;
    case "teleport-directory": handlers.openTeleportDirectory(); return;
    case "teleport-build": handlers.openTeleportBuild(); return;
    case "divination": handlers.openDivination(); return;
    case "activity-log": handlers.openActivityLog(); return;
    default: handlers.setView(id);
  }
}
