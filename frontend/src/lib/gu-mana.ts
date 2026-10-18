// Mirrors backend/internal/gamefx/energy.go (EnergyCost / GetPortalManaCost) so
// the UI shows the same mana a cast actually spends. BaseManaCost is a flat cost
// defined at the Gu's own rank and sub 1; higher player rank/sub-rank lowers it.

const IMMORTAL_RANK = 6;
const MORTAL_RANK_VALUES = [1, 2, 5, 20, 100];
const IMMORTAL_RANK_VALUES = [1, 100, 10000, 1000000];

function rankRatio(values: number[], guIdx: number, userIdx: number): number {
  const clamp = (i: number) => Math.max(0, Math.min(values.length - 1, i));
  return values[clamp(guIdx)] / values[clamp(userIdx)];
}

/** Chân Nguyên / Tiên Nguyên cost of activating a Gu (gamefx.EnergyCost). */
export function guManaCost(
  baseManaCost: number,
  guRank: number,
  playerRank: number,
  playerSubRank: number,
  refineTier = 0,
): number {
  let cost: number;

  if (playerRank >= IMMORTAL_RANK) {
    if (guRank < IMMORTAL_RANK) return 0;
    if (playerRank > guRank) {
      cost = baseManaCost / Math.pow(100, playerRank - guRank);
    } else if (playerRank < guRank) {
      cost = baseManaCost * rankRatio(IMMORTAL_RANK_VALUES, guRank - IMMORTAL_RANK, playerRank - IMMORTAL_RANK);
    } else {
      cost = baseManaCost;
    }
  } else if (playerRank > guRank && guRank < IMMORTAL_RANK) {
    cost = baseManaCost / Math.pow(10, playerRank - guRank);
  } else if (playerRank < guRank) {
    if (guRank >= IMMORTAL_RANK) return Number.POSITIVE_INFINITY;
    cost = baseManaCost * rankRatio(MORTAL_RANK_VALUES, guRank - 1, playerRank - 1);
  } else {
    cost = baseManaCost / Math.pow(2, Math.max(0, playerSubRank - 1));
  }

  return refineTier > 0 ? cost * 0.5 : cost;
}

/** Cost of a portal/teleport Gu (handlers.GetPortalManaCost). */
export function portalManaCost(baseManaCost: number, guRank: number, playerRank: number): number {
  const base = baseManaCost > 0 ? baseManaCost : 10;
  if (guRank === playerRank) return base;
  return base * Math.pow(2, guRank - playerRank);
}

/** Short display form: "10", "2.5", "1.25". */
export function formatManaCost(cost: number): string {
  if (!Number.isFinite(cost)) return "∞";
  if (Number.isInteger(cost)) return String(cost);
  return Number(cost.toFixed(2)).toString();
}
