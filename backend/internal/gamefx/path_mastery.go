package gamefx

import (
	"math"
	"strings"
)

// CalculatePathMasteryGain calculates the Chân Ý (path mastery) points earned
// when refining or using Gu skills / killer moves of a given rank, scaled down by
// the player's current attainment level (level 0-4) in that path. High attainment
// levels receive reduced efficiency when using lower rank Gu, and 0 points when
// exceeding the rank's cap limit.
func CalculatePathMasteryGain(guRank int, pathLevel int, baseGain int) int {
	if baseGain <= 0 {
		return 0
	}
	if guRank < 1 {
		guRank = 1
	}
	// Rank 1: only gives gain when level < 2 (Ordinary / Master)
	if guRank == 1 && pathLevel >= 2 {
		return 0
	}
	// Rank 2 & 3: only gives gain when level < 3 (Ordinary / Master / Grandmaster)
	if (guRank == 2 || guRank == 3) && pathLevel >= 3 {
		return 0
	}
	// Rank 4+: only gives gain when level < 4 (Supreme Grandmaster is capped)
	if guRank >= 4 && pathLevel >= 4 {
		return 0
	}

	factor := 1.0 - 0.20*float64(pathLevel)
	if factor < 0.0 {
		factor = 0.0
	}

	gain := int(math.Round(float64(baseGain) * factor))
	if gain < 1 && baseGain > 0 {
		gain = 1 // guarantee at least 1 point if not capped by level
	}
	return gain
}

// Path mastery (Cảnh Giới Lưu Phái) is stored like Dao Marks: a JSONB map of
// path -> accumulated points (players.path_masteries). The attained realm
// (level 0-4) is derived purely from point milestones:
//
//	Level 0: Ordinary        (Phàm Thai)            points <  100
//	Level 1: Master          (Đại Sư)               points >= 100
//	Level 2: Grandmaster     (Tông Sư)              points >= 500
//	Level 3: Great Grandmaster (Đại Tông Sư)        points >= 2500
//	Level 4: Supreme Grandmaster (Vô Thượng)        points >= 10000
//
// The realm is a requirement for rank advancement, not gated by rank: to
// advance to Rank 7 a player must own at least one path at Grandmaster
// (Tông Sư), Rank 8 requires one at Great Grandmaster, Rank 9 requires one at
// Supreme Grandmaster (see PathMasteryLevelRequiredForRank / HasPathMastery).
//
// Points are earned by casting path killer moves, refining Gu, instance
// attainment rewards, etc.

const (
	// PathMasteryMasterPoints is the milestone to reach Master (level 1).
	PathMasteryMasterPoints = 100
	// PathMasteryGrandmasterPoints is the milestone to reach Grandmaster (level 2).
	PathMasteryGrandmasterPoints = 1000
	// PathMasteryGreatGrandmasterPoints is the milestone to reach Great Grandmaster (level 3).
	PathMasteryGreatGrandmasterPoints = 10000
	// PathMasterySupremeGrandmasterPoints is the milestone to reach Supreme Grandmaster (level 4).
	PathMasterySupremeGrandmasterPoints = 100000
)

// pathMasteryLevelMultipliers maps an attainment level (0-4) to its bonus
// multiplier used in combat (mirrors the old level-name table).
var pathMasteryLevelMultipliers = [5]float64{1.0, 1.1, 1.25, 1.4, 1.5}

// PathMasteryLevel returns the attained realm level (0-4) for a given point
// count. Level is a pure function of points (rank gates are enforced at
// rank-up time, not here).
func PathMasteryLevel(points int) int {
	switch {
	case points >= PathMasterySupremeGrandmasterPoints:
		return 4
	case points >= PathMasteryGreatGrandmasterPoints:
		return 3
	case points >= PathMasteryGrandmasterPoints:
		return 2
	case points >= PathMasteryMasterPoints:
		return 1
	}
	return 0
}

// PathMasteryLevelName returns the canonical English level name for a level
// index (0-4). Used for logging / display.
func PathMasteryLevelName(level int) string {
	names := [5]string{"Ordinary", "Master", "Grandmaster", "Great Grandmaster", "Supreme Grandmaster"}
	if level < 0 || level > 4 {
		return "Ordinary"
	}
	return names[level]
}

// PathMasteryLevelMultiplier returns the combat multiplier for an attainment
// level (0-4). Level 0 = 1.0, Master = 1.1, ..., Supreme = 1.5.
func PathMasteryLevelMultiplier(level int) float64 {
	if level < 0 || level > 4 {
		return 1.0
	}
	return pathMasteryLevelMultipliers[level]
}

// PathMasteryMultiplier returns the highest path-mastery multiplier across the
// given paths for a point-based path_masteries map. 1.0 means no mastery bonus.
func PathMasteryMultiplier(pathMasteries map[string]int, paths []string) float64 {
	masteryMultiplier := 1.0
	if pathMasteries == nil {
		return masteryMultiplier
	}
	for _, path := range paths {
		cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
		points, ok := pathMasteries[cleanPath]
		if !ok {
			points, ok = pathMasteries[path]
		}
		if !ok {
			continue
		}
		mult := PathMasteryLevelMultiplier(PathMasteryLevel(points))
		if mult > masteryMultiplier {
			masteryMultiplier = mult
		}
	}
	return masteryMultiplier
}

// PathMasteryLevelRequiredForRank returns the minimum path-mastery level
// required to advance to the given rank:
//
//	Rank 7 -> level 2 (Tông Sư / Grandmaster)
//	Rank 8 -> level 3 (Đại Tông Sư / Great Grandmaster)
//	Rank 9 -> level 4 (Vô Thượng Đại Tông Sư / Supreme Grandmaster)
//
// Returns 0 for ranks below 7 (no path mastery requirement).
func PathMasteryLevelRequiredForRank(rank int) int {
	switch rank {
	case 7:
		return 2
	case 8:
		return 3
	case 9:
		return 4
	}
	return 0
}

// HasPathMasteryAtLeast reports whether the player owns at least one path whose
// attained level is >= minLevel. Used as the rank-up requirement.
func HasPathMasteryAtLeast(pathMasteries map[string]int, minLevel int) bool {
	if minLevel <= 0 {
		return true
	}
	if len(pathMasteries) == 0 {
		return false
	}
	for _, points := range pathMasteries {
		if PathMasteryLevel(points) >= minLevel {
			return true
		}
	}
	return false
}

// PathMasteryPointsForLevel returns the minimum points milestone for a given
// level (0-4). Level 0 returns 0.
func PathMasteryPointsForLevel(level int) int {
	switch level {
	case 1:
		return PathMasteryMasterPoints
	case 2:
		return PathMasteryGrandmasterPoints
	case 3:
		return PathMasteryGreatGrandmasterPoints
	case 4:
		return PathMasterySupremeGrandmasterPoints
	}
	return 0
}

// PathMasteryBattleGainLimit returns the maximum number of Chân Ý points a
// player can accumulate from a single battle based on their cultivation rank
// (rank * 10). Once the limit is reached, further combat gains are suppressed.
func PathMasteryBattleGainLimit(rank int) int {
	if rank < 1 {
		rank = 1
	}
	return rank * 10
}
