/**
 * Centralized Game Balance Configuration & Formulas for Frontend.
 * Mirrors backend/internal/gamefx/balance.go to ensure UI calculations, formulas,
 * and displays match backend game balance rules exactly.
 */

// ========================================================================
// 1. CẢNH GIỚI LƯU PHÁI (PATH MASTERY / ATTAINMENT)
// ========================================================================

export const PATH_MASTERY_MILESTONES: number[] = [10000, 100000, 1000000, 10000000];

export const PATH_MASTERY_LEVEL_KEYS = [
  "Ordinary",
  "Master",
  "Grandmaster",
  "Great Grandmaster",
  "Supreme Grandmaster",
] as const;

export type PathMasteryLevelKey = (typeof PATH_MASTERY_LEVEL_KEYS)[number];

export const PATH_MASTERY_LEVEL_MULTIPLIERS = [1.0, 1.1, 1.25, 1.4, 1.5] as const;

/**
 * Maps accumulated points to realm level index (0-4).
 */
export function masteryLevelForPoints(points: number): number {
  if (points >= PATH_MASTERY_MILESTONES[3]) return 4;
  if (points >= PATH_MASTERY_MILESTONES[2]) return 3;
  if (points >= PATH_MASTERY_MILESTONES[1]) return 2;
  if (points >= PATH_MASTERY_MILESTONES[0]) return 1;
  return 0;
}

/**
 * Returns canonical English level name for path mastery points.
 */
export function masteryKeyForPoints(points: number): PathMasteryLevelKey {
  return PATH_MASTERY_LEVEL_KEYS[masteryLevelForPoints(points)];
}

/**
 * Minimum attainment realm required to advance to a rank (Rank 7=2, Rank 8=3, Rank 9=4).
 */
export function masteryLevelRequiredForRank(rank: number): number {
  if (rank === 7) return 2;
  if (rank === 8) return 3;
  if (rank === 9) return 4;
  return 0;
}

// ========================================================================
// 2. BẬC QUY MÔ / RANK SCALE & MANA TABLE (CHUẨN HÓA 9 CHUYỂN)
// ========================================================================

/**
 * Canonical exponential cost and power scale across ranks 1 to 9.
 * Mortals: R1=1, R2=2, R3=5, R4=20, R5=100.
 * Immortals: R6=10,000, R7=1,000,000, R8=100,000,000, R9=10,000,000,000.
 */
export const RANK_SCALE_TABLE: Record<number, number> = {
  1: 1,
  2: 2,
  3: 5,
  4: 20,
  5: 100,
  6: 10000,
  7: 1000000,
  8: 100000000,
  9: 10000000000,
};

export function getRankScale(rank: number): number {
  if (rank <= 1) return RANK_SCALE_TABLE[1];
  if (rank >= 9) return RANK_SCALE_TABLE[9];
  return RANK_SCALE_TABLE[rank] || 1;
}

/**
 * Quantity of sealing_stone (Đá Phong Ấn) required to seal a Gu of the given rank.
 */
export function getSealingStoneCost(rank: number): number {
  return getRankScale(rank);
}

// ========================================================================
// 2b. GIỐNG TIÊN KHIẾU
// ========================================================================

/** Seed amount to plant one aperture resource zone — same for every rank. */
export const APERTURE_SEED_BASE = 100;

// ========================================================================
// 3. CHI PHÍ ĐỔI ĐẠO HIỆU & HÌNH PHẠT CỔ TIÊN (RENAME & RE-IDENTITY)
// ========================================================================

export interface RenameCostInfo {
  costType: "free" | "primeval" | "immortal";
  amount: number;
  label: string;
}

export function getRenameCost(rank: number): RenameCostInfo {
  if (rank <= 2) {
    return { costType: "free", amount: 0, label: "Đổi Đạo Hiệu (Miễn Phí)" };
  }
  if (rank === 3) {
    return { costType: "primeval", amount: 5000, label: "Phí Đổi Tên: 5.000 Nguyên Thạch" };
  }
  if (rank === 4) {
    return { costType: "primeval", amount: 20000, label: "Phí Đổi Tên: 20.000 Nguyên Thạch" };
  }
  if (rank === 5) {
    return { costType: "primeval", amount: 100000, label: "Phí Đổi Tên: 100.000 Nguyên Thạch" };
  }
  if (rank === 6) {
    return { costType: "immortal", amount: 100, label: "100 Tiên Nguyên Thạch" };
  }
  if (rank === 7) {
    return { costType: "immortal", amount: 500, label: "500 Tiên Nguyên Thạch" };
  }
  if (rank === 8) {
    return { costType: "immortal", amount: 2500, label: "2.500 Tiên Nguyên Thạch" };
  }
  return { costType: "immortal", amount: 10000, label: "10.000 Tiên Nguyên Thạch" };
}

export interface ImmortalPenaltyRateInfo {
  lossPercent: number;     // % Dao marks & Path mastery points lost (30..70)
  hpLossPercent: number;   // % Aperture HP damage (20..50)
  stressGain: number;      // Stress increase (+30..+100)
}

export function getImmortalPenaltyRates(rank: number): ImmortalPenaltyRateInfo {
  if (rank === 6) {
    return { lossPercent: 30, hpLossPercent: 20, stressGain: 30 };
  }
  if (rank === 7) {
    return { lossPercent: 40, hpLossPercent: 30, stressGain: 50 };
  }
  if (rank === 8) {
    return { lossPercent: 50, hpLossPercent: 40, stressGain: 80 };
  }
  return { lossPercent: 70, hpLossPercent: 50, stressGain: 100 };
}
