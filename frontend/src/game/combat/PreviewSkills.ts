import type { CastPayload } from "../network/Protocol.ts";

export type CastMode = CastPayload["mode"];

export interface PreviewSkill {
  slot: number;
  name: string;
  mode: CastMode;
  range: number;
  radius: number;
  cooldownMs: number;
  manaCost: number;
}

/**
 * UI mirror of the server preview loadout. It intentionally carries no damage
 * numbers: the server owns damage/cooldown/mana validation. PR-18 replaces this
 * with the compiled per-player Gu loadout from world.welcome.
 */
export const PREVIEW_SKILLS: readonly PreviewSkill[] = [
  { slot: 1, name: "Bolt", mode: "TARGET", range: 9, radius: 0, cooldownMs: 1500, manaCost: 4 },
  { slot: 2, name: "Water Arrow", mode: "DIRECTION", range: 16, radius: 0.35, cooldownMs: 2000, manaCost: 5 },
  { slot: 3, name: "Ground Burst", mode: "GROUND", range: 10, radius: 2.5, cooldownMs: 3000, manaCost: 7 },
  { slot: 4, name: "Iron Shell", mode: "SELF", range: 0, radius: 0, cooldownMs: 6000, manaCost: 6 },
  { slot: 5, name: "Mending", mode: "SELF", range: 0, radius: 0, cooldownMs: 7500, manaCost: 8 },
  { slot: 6, name: "Frost Lock", mode: "TARGET", range: 9, radius: 0, cooldownMs: 6000, manaCost: 9 },
];

export function previewSkill(slot: number): PreviewSkill | null {
  return PREVIEW_SKILLS.find((skill) => skill.slot === slot) ?? null;
}
