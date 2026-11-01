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
  { id: "purple", name: "Tử Khí", color: "#261338", desc: "Tím thẫm tôn quý" },
  { id: "jade", name: "Thanh Trúc", color: "#0f2b20", desc: "Ngọc bích tiên môn" },
  { id: "ocean", name: "Hàn Băng", color: "#11263d", desc: "U lam tĩnh mịch" },
  { id: "crimson", name: "Xích Sa", color: "#331217", desc: "Huyết ma quyền uy" },
  { id: "dark", name: "Huyền Mặc", color: "#16171c", desc: "U minh sâu thẳm" },
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

interface ThemeTokens {
  bg: string;
  gradient1: string;
  gradient2: string;
  headingBg: string;
  border: string;
  borderSoft: string;
  glow: string;
  zinc950: string;
  zinc900: string;
  zinc800: string;
  surface: string;
  elevated: string;
  accent: string;
  tabActive: string;
}

const PRESET_TOKENS: Record<Exclude<MenuThemePreset, "custom">, ThemeTokens> = {
  default: {
    bg: "#211810",
    gradient1: "rgba(33, 24, 16, 0.94)",
    gradient2: "rgba(42, 30, 20, 0.96)",
    headingBg: "linear-gradient(180deg, rgba(91, 64, 39, 0.98) 0%, rgba(54, 38, 25, 0.98) 100%)",
    border: "#8c6c40",
    borderSoft: "rgba(168, 137, 82, 0.72)",
    glow: "rgba(238, 211, 158, 0.12)",
    zinc950: "#130e0a",
    zinc900: "#211810",
    zinc800: "#302318",
    surface: "#241a12",
    elevated: "#352618",
    accent: "#e4b96c",
    tabActive: "#6b4728",
  },
  purple: {
    bg: "#261338",
    gradient1: "rgba(38, 19, 56, 0.94)",
    gradient2: "rgba(48, 24, 70, 0.96)",
    headingBg: "linear-gradient(180deg, rgba(92, 38, 138, 0.98) 0%, rgba(52, 20, 78, 0.98) 100%)",
    border: "#8748b5",
    borderSoft: "rgba(168, 105, 215, 0.72)",
    glow: "rgba(216, 180, 254, 0.2)",
    zinc950: "#160a22",
    zinc900: "#2c1742",
    zinc800: "#4c2472",
    surface: "#28143b",
    elevated: "#3a1c56",
    accent: "#c084fc",
    tabActive: "#5b21b6",
  },
  jade: {
    bg: "#0f2b20",
    gradient1: "rgba(15, 43, 32, 0.94)",
    gradient2: "rgba(20, 56, 42, 0.96)",
    headingBg: "linear-gradient(180deg, rgba(32, 95, 72, 0.98) 0%, rgba(18, 55, 40, 0.98) 100%)",
    border: "#2d8a63",
    borderSoft: "rgba(64, 180, 130, 0.72)",
    glow: "rgba(110, 231, 183, 0.2)",
    zinc950: "#081812",
    zinc900: "#14392b",
    zinc800: "#225841",
    surface: "#133527",
    elevated: "#1c4a37",
    accent: "#34d399",
    tabActive: "#065f46",
  },
  ocean: {
    bg: "#11263d",
    gradient1: "rgba(17, 38, 61, 0.94)",
    gradient2: "rgba(23, 50, 80, 0.96)",
    headingBg: "linear-gradient(180deg, rgba(36, 85, 138, 0.98) 0%, rgba(20, 48, 80, 0.98) 100%)",
    border: "#367cb4",
    borderSoft: "rgba(75, 155, 220, 0.72)",
    glow: "rgba(147, 197, 253, 0.2)",
    zinc950: "#091523",
    zinc900: "#173250",
    zinc800: "#244e7c",
    surface: "#142d48",
    elevated: "#1e3f65",
    accent: "#60a5fa",
    tabActive: "#1e40af",
  },
  crimson: {
    bg: "#331217",
    gradient1: "rgba(51, 18, 23, 0.94)",
    gradient2: "rgba(65, 23, 30, 0.96)",
    headingBg: "linear-gradient(180deg, rgba(118, 38, 48, 0.98) 0%, rgba(68, 20, 28, 0.98) 100%)",
    border: "#9d3644",
    borderSoft: "rgba(195, 75, 90, 0.72)",
    glow: "rgba(252, 165, 165, 0.2)",
    zinc950: "#1d0a0d",
    zinc900: "#3d171c",
    zinc800: "#5f222c",
    surface: "#38141a",
    elevated: "#4d1c24",
    accent: "#f87171",
    tabActive: "#991b1b",
  },
  dark: {
    bg: "#16171c",
    gradient1: "rgba(22, 23, 28, 0.94)",
    gradient2: "rgba(30, 32, 38, 0.96)",
    headingBg: "linear-gradient(180deg, rgba(62, 66, 78, 0.98) 0%, rgba(35, 37, 45, 0.98) 100%)",
    border: "#525666",
    borderSoft: "rgba(110, 115, 130, 0.72)",
    glow: "rgba(226, 232, 240, 0.15)",
    zinc950: "#0d0e11",
    zinc900: "#1e2026",
    zinc800: "#333641",
    surface: "#1b1c22",
    elevated: "#282a33",
    accent: "#cbd5e1",
    tabActive: "#334155",
  },
};

function clampByte(val: number): number {
  return Math.max(0, Math.min(255, Math.round(val)));
}

function adjustColor(rgb: { r: number; g: number; b: number }, factor: number, add = 0) {
  return {
    r: clampByte(rgb.r * factor + add),
    g: clampByte(rgb.g * factor + add),
    b: clampByte(rgb.b * factor + add),
  };
}

function toRgbString(rgb: { r: number; g: number; b: number }): string {
  return `${rgb.r}, ${rgb.g}, ${rgb.b}`;
}

function toHexString(rgb: { r: number; g: number; b: number }): string {
  const pad = (n: number) => n.toString(16).padStart(2, "0");
  return `#${pad(rgb.r)}${pad(rgb.g)}${pad(rgb.b)}`;
}

export function resolveActiveMenuTokens(config?: MenuColorConfig): {
  tokens: ThemeTokens;
  enableTexture: boolean;
  opacity: number;
} {
  const cfg = config ?? getUiPreferences().menuColor;
  const opacity = Math.min(100, Math.max(50, Number(cfg.opacity) || 96));
  const enableTexture = cfg.enableTexture !== false;

  if (cfg.preset !== "custom" && PRESET_TOKENS[cfg.preset]) {
    const base = PRESET_TOKENS[cfg.preset];
    const alpha1 = (opacity / 100).toFixed(2);
    const alpha2 = Math.min(1, opacity / 100 + 0.02).toFixed(2);
    const rgb = hexToRgb(base.bg) ?? { r: 33, g: 24, b: 16 };
    const r2 = clampByte(rgb.r + 8);
    const g2 = clampByte(rgb.g + 6);
    const b2 = clampByte(rgb.b + 4);

    return {
      tokens: {
        ...base,
        gradient1: `rgba(${toRgbString(rgb)}, ${alpha1})`,
        gradient2: `rgba(${r2}, ${g2}, ${b2}, ${alpha2})`,
      },
      enableTexture,
      opacity,
    };
  }

  // Custom Hex Theme Calculation
  const customRgb = hexToRgb(cfg.customBgColor || "#211810") ?? { r: 33, g: 24, b: 16 };
  const alpha1 = (opacity / 100).toFixed(2);
  const alpha2 = Math.min(1, opacity / 100 + 0.02).toFixed(2);

  const z950 = adjustColor(customRgb, 0.6);
  const z900 = adjustColor(customRgb, 1.0);
  const z800 = adjustColor(customRgb, 1.5, 15);
  const surf = adjustColor(customRgb, 1.1, 5);
  const elev = adjustColor(customRgb, 1.35, 10);
  const bord = adjustColor(customRgb, 2.0, 30);
  const bordSoft = adjustColor(customRgb, 1.8, 20);
  const head1 = adjustColor(customRgb, 1.9, 25);
  const head2 = adjustColor(customRgb, 1.3, 10);
  const acc = adjustColor(customRgb, 2.5, 50);
  const tab = adjustColor(customRgb, 1.7, 20);

  const tokens: ThemeTokens = {
    bg: toHexString(customRgb),
    gradient1: `rgba(${toRgbString(customRgb)}, ${alpha1})`,
    gradient2: `rgba(${toRgbString(adjustColor(customRgb, 1.15, 6))}, ${alpha2})`,
    headingBg: `linear-gradient(180deg, rgba(${toRgbString(head1)}, 0.98) 0%, rgba(${toRgbString(head2)}, 0.98) 100%)`,
    border: toHexString(bord),
    borderSoft: `rgba(${toRgbString(bordSoft)}, 0.72)`,
    glow: `rgba(${toRgbString(acc)}, 0.2)`,
    zinc950: toHexString(z950),
    zinc900: toHexString(z900),
    zinc800: toHexString(z800),
    surface: toHexString(surf),
    elevated: toHexString(elev),
    accent: toHexString(acc),
    tabActive: toHexString(tab),
  };

  return { tokens, enableTexture, opacity };
}

export function applyMenuThemeToCss(config?: MenuColorConfig) {
  if (typeof document === "undefined") return;
  const { tokens, enableTexture } = resolveActiveMenuTokens(config);
  const root = document.documentElement;

  root.style.setProperty("--mmo-menu-bg", tokens.bg);
  root.style.setProperty("--mmo-menu-gradient-1", tokens.gradient1);
  root.style.setProperty("--mmo-menu-gradient-2", tokens.gradient2);
  root.style.setProperty("--mmo-menu-heading-bg", tokens.headingBg);
  root.style.setProperty("--mmo-menu-border", tokens.border);
  root.style.setProperty("--mmo-menu-border-soft", tokens.borderSoft);
  root.style.setProperty("--mmo-menu-glow", tokens.glow);
  root.style.setProperty("--mmo-theme-zinc-950", tokens.zinc950);
  root.style.setProperty("--mmo-theme-zinc-900", tokens.zinc900);
  root.style.setProperty("--mmo-theme-zinc-800", tokens.zinc800);
  root.style.setProperty("--mmo-theme-surface", tokens.surface);
  root.style.setProperty("--mmo-theme-elevated", tokens.elevated);
  root.style.setProperty("--mmo-menu-accent", tokens.accent);
  root.style.setProperty("--mmo-menu-tab-active", tokens.tabActive);
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
