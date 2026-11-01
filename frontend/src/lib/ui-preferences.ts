"use client";

import { useEffect, useState } from "react";

/**
 * Single local save for UI settings. Everything device-specific that the client
 * persists lives here: quick actions, HUD view modes, changelog snooze, hotbar
 * keybinds and the general UI toggles. Server-side gameplay settings (Cổ/killer
 * move loadout, realtime controls) stay on the player row.
 */

export type HudViewMode = "expanded" | "compact" | "minimized";

export const HOTBAR_SLOT_COUNT = 12;

export type MenuThemePreset =
  | "default"
  | "dark"
  | "jade"
  | "ocean"
  | "crimson"
  | "purple"
  | "custom";

export interface MenuColorConfig {
  preset: MenuThemePreset;
  customBgColor: string;
  opacity: number;
  enableTexture: boolean;
}

export const MENU_THEME_PRESETS: Array<{
  id: MenuThemePreset;
  name: string;
  color: string;
  desc: string;
}> = [
  { id: "default", name: "Hoàng Giấy", color: "#211810", desc: "Cổ điển trầm ấm" },
  { id: "dark", name: "Huyền Mặc", color: "#111116", desc: "U minh sâu thẳm" },
  { id: "jade", name: "Thanh Trúc", color: "#0c1a14", desc: "Ngọc bích tiên môn" },
  { id: "ocean", name: "Hàn Băng", color: "#0b1522", desc: "U lam tĩnh mịch" },
  { id: "crimson", name: "Xích Sa", color: "#1e0b0e", desc: "Huyết ma quyền uy" },
  { id: "purple", name: "Tử Khí", color: "#170e1e", desc: "Tím thẫm tôn quý" },
];

export const DEFAULT_MENU_COLOR: MenuColorConfig = {
  preset: "default",
  customBgColor: "#211810",
  opacity: 96,
  enableTexture: true,
};

export type UiPreferences = {
  quickActions: string[];
  playerViewMode: HudViewMode | "";
  targetViewMode: HudViewMode | "";
  changelogHideUntil: string;
  keybinds: Record<number, string>;
  soundEnabled: boolean;
  showOtherPlayerNames: boolean;
  menuColor: MenuColorConfig;
};

export const UI_PREFERENCES_KEY = "gu_ui_prefs_v1";
export const UI_PREFERENCES_EVENT = "gu_ui_prefs_updated";

const LEGACY_KEYS = [
  "gu_mmo_quick_actions",
  "gu_world_direct_action_hotbar_v1",
  "gu_player_view_mode",
  "gu_target_view_mode",
  "gu_changelog_hide_until",
];

const DEFAULT_QUICK_ACTIONS = ["maps", "inventory", "cultivation", "shop"];
const VIEW_MODES: HudViewMode[] = ["expanded", "compact", "minimized"];

// Base keys owned by movement or fixed controls; binding these would shadow
// gameplay input, so the keybind editor refuses them.
const RESERVED_HOTBAR_CODES = new Set([
  "KeyW", "KeyA", "KeyS", "KeyD",
  "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight",
  "Tab", "Escape", "F3",
]);

export function isReservedHotbarBinding(binding: string): boolean {
  const code = binding.split("+").pop() ?? "";
  return RESERVED_HOTBAR_CODES.has(code);
}

export function defaultHotbarKeybinds(): Record<number, string> {
  const keybinds: Record<number, string> = {};
  for (let slot = 1; slot <= HOTBAR_SLOT_COUNT; slot += 1) {
    keybinds[slot] = slot <= 8 ? `Digit${slot}` : `Alt+Digit${slot - 8}`;
  }
  return keybinds;
}

function isViewMode(value: unknown): value is HudViewMode {
  return typeof value === "string" && VIEW_MODES.includes(value as HudViewMode);
}

function normalizeQuickActions(value: unknown): string[] {
  if (!Array.isArray(value)) return [...DEFAULT_QUICK_ACTIONS];
  const ids = value.filter((id): id is string => typeof id === "string" && id.length > 0);
  return ids.length > 0 ? ids.slice(0, 5) : [...DEFAULT_QUICK_ACTIONS];
}

function normalizeKeybinds(value: unknown): Record<number, string> {
  const defaults = defaultHotbarKeybinds();
  const keybinds: Record<number, string> = {};
  if (value && typeof value === "object") {
    for (const [rawSlot, binding] of Object.entries(value as Record<string, unknown>)) {
      const slot = Number(rawSlot);
      if (!Number.isInteger(slot) || slot < 1 || slot > HOTBAR_SLOT_COUNT) continue;
      if (typeof binding === "string" && binding.length > 0 && binding.length <= 32) {
        keybinds[slot] = binding;
      }
    }
  }
  return { ...defaults, ...keybinds };
}

function normalizeMenuColor(value: unknown): MenuColorConfig {
  if (!value || typeof value !== "object") return { ...DEFAULT_MENU_COLOR };
  const raw = value as Partial<MenuColorConfig>;
  const validPresets: MenuThemePreset[] = [
    "default",
    "dark",
    "jade",
    "ocean",
    "crimson",
    "purple",
    "custom",
  ];
  const preset =
    typeof raw.preset === "string" && validPresets.includes(raw.preset as MenuThemePreset)
      ? (raw.preset as MenuThemePreset)
      : "default";

  let customBgColor = DEFAULT_MENU_COLOR.customBgColor;
  if (
    typeof raw.customBgColor === "string" &&
    /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.test(raw.customBgColor)
  ) {
    customBgColor = raw.customBgColor;
  }

  let opacity = DEFAULT_MENU_COLOR.opacity;
  if (typeof raw.opacity === "number" && !isNaN(raw.opacity)) {
    opacity = Math.max(50, Math.min(100, Math.round(raw.opacity)));
  }

  const enableTexture = raw.enableTexture !== false;

  return { preset, customBgColor, opacity, enableTexture };
}

export function normalizeUiPreferences(value: unknown): UiPreferences {
  const stored = value && typeof value === "object" ? (value as Partial<UiPreferences>) : {};
  return {
    quickActions: normalizeQuickActions(stored.quickActions),
    playerViewMode: isViewMode(stored.playerViewMode) ? stored.playerViewMode : "",
    targetViewMode: isViewMode(stored.targetViewMode) ? stored.targetViewMode : "",
    changelogHideUntil: typeof stored.changelogHideUntil === "string" ? stored.changelogHideUntil : "",
    keybinds: normalizeKeybinds(stored.keybinds),
    soundEnabled: typeof stored.soundEnabled === "boolean" ? stored.soundEnabled : true,
    showOtherPlayerNames: typeof stored.showOtherPlayerNames === "boolean" ? stored.showOtherPlayerNames : true,
    menuColor: normalizeMenuColor(stored.menuColor),
  };
}

/** Resolves the persisted player card mode, falling back to the mobile default. */
export function resolvePlayerViewMode(preferences?: UiPreferences): HudViewMode {
  const prefs = preferences ?? getUiPreferences();
  if (prefs.playerViewMode) return prefs.playerViewMode;
  if (typeof window !== "undefined" && window.innerWidth < 640) return "compact";
  return "expanded";
}

/** Resolves the persisted target card mode, falling back to the mobile default. */
export function resolveTargetViewMode(preferences?: UiPreferences): HudViewMode {
  const prefs = preferences ?? getUiPreferences();
  if (prefs.targetViewMode) return prefs.targetViewMode;
  if (typeof window !== "undefined" && window.matchMedia("(max-width: 767px)").matches) return "compact";
  return "expanded";
}

function readLegacyPreferences(): Partial<UiPreferences> {
  if (typeof window === "undefined") return {};
  const legacy: Partial<UiPreferences> = {};
  try {
    const quick = localStorage.getItem("gu_mmo_quick_actions");
    if (quick) {
      const parsed = JSON.parse(quick);
      if (Array.isArray(parsed)) legacy.quickActions = normalizeQuickActions(parsed);
    }
  } catch { /* Ignore malformed legacy data. */ }
  try {
    const playerMode = localStorage.getItem("gu_player_view_mode");
    if (isViewMode(playerMode)) legacy.playerViewMode = playerMode;
    const targetMode = localStorage.getItem("gu_target_view_mode");
    if (isViewMode(targetMode)) legacy.targetViewMode = targetMode;
    const hiddenUntil = localStorage.getItem("gu_changelog_hide_until");
    if (hiddenUntil) legacy.changelogHideUntil = hiddenUntil;
  } catch { /* Ignore malformed legacy data. */ }
  return legacy;
}

let cache: UiPreferences | null = null;
let storageListenerAttached = false;

function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  const clean = hex.replace("#", "").trim();
  if (clean.length === 3) {
    const r = parseInt(clean[0] + clean[0], 16);
    const g = parseInt(clean[1] + clean[1], 16);
    const b = parseInt(clean[2] + clean[2], 16);
    if (!isNaN(r) && !isNaN(g) && !isNaN(b)) return { r, g, b };
  } else if (clean.length === 6) {
    const r = parseInt(clean.slice(0, 2), 16);
    const g = parseInt(clean.slice(2, 4), 16);
    const b = parseInt(clean.slice(4, 6), 16);
    if (!isNaN(r) && !isNaN(g) && !isNaN(b)) return { r, g, b };
  }
  return null;
}

export function resolveActiveMenuColor(config?: MenuColorConfig): {
  hex: string;
  rgb: { r: number; g: number; b: number };
  opacity: number;
  enableTexture: boolean;
} {
  const cfg = config ?? getUiPreferences().menuColor;
  let hex = DEFAULT_MENU_COLOR.customBgColor;
  if (cfg.preset === "custom") {
    hex = cfg.customBgColor || DEFAULT_MENU_COLOR.customBgColor;
  } else {
    const found = MENU_THEME_PRESETS.find((p) => p.id === cfg.preset);
    hex = found ? found.color : DEFAULT_MENU_COLOR.customBgColor;
  }
  const rgb = hexToRgb(hex) ?? { r: 33, g: 24, b: 16 };
  const opacity = Math.min(100, Math.max(50, Number(cfg.opacity) || 96));
  const enableTexture = cfg.enableTexture !== false;
  return { hex, rgb, opacity, enableTexture };
}

export function applyMenuThemeToCss(config?: MenuColorConfig) {
  if (typeof document === "undefined") return;
  const { hex, rgb, opacity, enableTexture } = resolveActiveMenuColor(config);
  const root = document.documentElement;

  const alpha1 = (opacity / 100).toFixed(2);
  const alpha2 = Math.min(1, opacity / 100 + 0.02).toFixed(2);

  const r2 = Math.min(255, rgb.r + 6);
  const g2 = Math.min(255, rgb.g + 4);
  const b2 = Math.min(255, rgb.b + 3);

  const hr1 = Math.min(255, Math.round(rgb.r * 1.8 + 20));
  const hg1 = Math.min(255, Math.round(rgb.g * 1.8 + 15));
  const hb1 = Math.min(255, Math.round(rgb.b * 1.8 + 10));
  const hr2 = Math.min(255, Math.round(rgb.r * 1.3 + 10));
  const hg2 = Math.min(255, Math.round(rgb.g * 1.3 + 8));
  const hb2 = Math.min(255, Math.round(rgb.b * 1.3 + 5));

  root.style.setProperty("--mmo-menu-bg", hex);
  root.style.setProperty("--mmo-menu-gradient-1", `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${alpha1})`);
  root.style.setProperty("--mmo-menu-gradient-2", `rgba(${r2}, ${g2}, ${b2}, ${alpha2})`);
  root.style.setProperty(
    "--mmo-menu-heading-bg",
    `linear-gradient(180deg, rgba(${hr1}, ${hg1}, ${hb1}, 0.98), rgba(${hr2}, ${hg2}, ${hb2}, 0.98))`
  );
  root.style.setProperty(
    "--mmo-menu-texture",
    enableTexture ? 'url("/assets/ui/guworld-rice-paper.webp?v=1")' : "none"
  );
}

function attachStorageListener() {
  if (storageListenerAttached || typeof window === "undefined") return;
  storageListenerAttached = true;
  window.addEventListener("storage", (event) => {
    if (event.key !== UI_PREFERENCES_KEY) return;
    cache = null;
    const current = getUiPreferences();
    applyMenuThemeToCss(current.menuColor);
    window.dispatchEvent(new CustomEvent(UI_PREFERENCES_EVENT, { detail: current }));
  });
}

export function getUiPreferences(): UiPreferences {
  if (typeof window === "undefined") return normalizeUiPreferences(null);
  if (cache) return cache;
  attachStorageListener();
  try {
    const raw = localStorage.getItem(UI_PREFERENCES_KEY);
    if (raw) {
      cache = normalizeUiPreferences(JSON.parse(raw));
      return cache;
    }
    // One-time migration from the previous scattered keys.
    cache = normalizeUiPreferences(readLegacyPreferences());
    localStorage.setItem(UI_PREFERENCES_KEY, JSON.stringify(cache));
    for (const key of LEGACY_KEYS) localStorage.removeItem(key);
  } catch {
    cache = normalizeUiPreferences(null);
  }
  return cache;
}

export function updateUiPreferences(patch: Partial<UiPreferences>): UiPreferences {
  const next = normalizeUiPreferences({ ...getUiPreferences(), ...patch });
  cache = next;
  if (typeof window !== "undefined") {
    try {
      localStorage.setItem(UI_PREFERENCES_KEY, JSON.stringify(next));
    } catch { /* Storage can be unavailable in private contexts. */ }
    applyMenuThemeToCss(next.menuColor);
    window.dispatchEvent(new CustomEvent(UI_PREFERENCES_EVENT, { detail: next }));
  }
  return next;
}

export function resetUiPreferences(): UiPreferences {
  return updateUiPreferences({
    quickActions: [...DEFAULT_QUICK_ACTIONS],
    keybinds: defaultHotbarKeybinds(),
    menuColor: { ...DEFAULT_MENU_COLOR },
  });
}

export function subscribeUiPreferences(handler: (preferences: UiPreferences) => void): () => void {
  if (typeof window === "undefined") return () => {};
  attachStorageListener();
  const listener = (event: Event) => {
    handler((event as CustomEvent<UiPreferences>).detail ?? getUiPreferences());
  };
  window.addEventListener(UI_PREFERENCES_EVENT, listener);
  return () => window.removeEventListener(UI_PREFERENCES_EVENT, listener);
}

export function useUiPreferences(): UiPreferences {
  const [preferences, setPreferences] = useState<UiPreferences>(() => normalizeUiPreferences(null));
  useEffect(() => {
    setPreferences(getUiPreferences());
    return subscribeUiPreferences(setPreferences);
  }, []);
  return preferences;
}

export function getHotbarKeybinds(): Record<number, string> {
  return getUiPreferences().keybinds;
}

export function setHotbarKeybinds(keybinds: Record<number, string>): UiPreferences {
  return updateUiPreferences({ keybinds });
}

export function useHotbarKeybinds(): Record<number, string> {
  return useUiPreferences().keybinds;
}

/** Returns the binding for a slot as a readable label, e.g. "1", "Alt+1", "Q", "F2". */
export function formatHotbarBinding(binding: string | undefined): string {
  if (!binding) return "";
  const parts = binding.split("+");
  const code = parts.pop() ?? "";
  let label = code;
  const digit = /^(?:Digit|Numpad)(\d)$/.exec(code);
  if (digit) {
    label = /^Numpad/.test(code) ? `Num${digit[1]}` : digit[1];
  } else if (/^Key[A-Z]$/.test(code)) {
    label = code.slice(3);
  } else if (/^Arrow/.test(code)) {
    label = code.replace("Arrow", "").replace("Left", "←").replace("Right", "→").replace("Up", "↑").replace("Down", "↓");
  } else {
    label = code.replace("Space", "Space");
  }
  return [...parts, label].join("+");
}

export function hotbarKeyLabel(slot: number, keybinds?: Record<number, string>): string {
  const bindings = keybinds ?? getHotbarKeybinds();
  return formatHotbarBinding(bindings[slot]);
}

/** Serializes a keyboard event into the stored binding format (uses event.code). */
export function bindingFromEvent(event: Pick<KeyboardEvent, "code" | "key" | "altKey" | "ctrlKey" | "shiftKey" | "metaKey">): string | null {
  const modifierKeys = ["Control", "Alt", "Shift", "Meta"];
  if (modifierKeys.includes(event.key)) return null;
  const code = event.code || event.key;
  if (!code) return null;
  const modifiers: string[] = [];
  if (event.altKey) modifiers.push("Alt");
  if (event.ctrlKey) modifiers.push("Ctrl");
  if (event.shiftKey) modifiers.push("Shift");
  if (event.metaKey) modifiers.push("Meta");
  return [...modifiers, code].join("+");
}

/** Maps a keyboard event to a hotbar slot using the active keybindings. */
export function matchHotbarSlot(
  event: Pick<KeyboardEvent, "code" | "key" | "altKey" | "ctrlKey" | "shiftKey" | "metaKey">,
  keybinds?: Record<number, string>,
): number | null {
  const binding = bindingFromEvent(event);
  if (!binding) return null;
  const normalized = normalizeDigits(binding);
  const bindings = keybinds ?? getHotbarKeybinds();
  for (const [rawSlot, stored] of Object.entries(bindings)) {
    if (stored === binding || normalizeDigits(stored) === normalized) return Number(rawSlot);
  }
  return null;
}

// The pre-remap hotbar accepted both Digit and Numpad keys for the same slot;
// keep that equivalence unless the player explicitly binds a numpad key.
function normalizeDigits(binding: string): string {
  return binding.replace(/Numpad(\d)/g, "Digit$1");
}

/** Returns every pair of slots that share a binding (Numpad/Digit aliases included). */
export function findKeybindConflicts(keybinds: Record<number, string>): Array<[number, number]> {
  const seen = new Map<string, number>();
  const conflicts: Array<[number, number]> = [];
  for (const [rawSlot, binding] of Object.entries(keybinds)) {
    const slot = Number(rawSlot);
    const normalized = normalizeDigits(binding);
    const other = seen.get(normalized);
    if (other !== undefined) conflicts.push([other, slot]);
    else seen.set(normalized, slot);
  }
  return conflicts;
}

export function bindingsEqual(a: string, b: string): boolean {
  return normalizeDigits(a) === normalizeDigits(b);
}

/** Browser-owned chords cannot be intercepted reliably; reject them up front. */
export function isUnsafeHotbarBinding(binding: string): boolean {
  const parts = binding.split("+");
  const modifiers = parts.slice(0, -1);
  if (modifiers.includes("Ctrl") || modifiers.includes("Meta")) return true;
  const key = parts[parts.length - 1] ?? "";
  return key === "F5" || key === "F11" || key === "F12";
}
