package match

import "time"

// TickInterval is the authoritative simulation step. 20Hz: fast enough that
// movement feels continuous, cheap enough to run every live match from a single
// ticker.
const TickInterval = 50 * time.Millisecond

const (
	// Arena dimensions in tiles.
	ArenaWidth  = 40.0
	ArenaHeight = 24.0

	// ActorRadius is used by both hit tests and obstacle sliding.
	ActorRadius = 0.5

	// SpeedToTilesPerTick converts a player Speed stat into tiles per tick.
	// Speed stats sit around 10-20, so a factor of 0.02 puts a rank-1 player at
	// roughly 4 tiles/second (a 40-tile arena takes ~10s to cross). This is the
	// knob that decides how fast a match feels; tune it here, not at the call
	// sites.
	SpeedToTilesPerTick = 0.02

	MaxMana          = 100.0
	ManaRegenPerTick = 0.3 // 6 mana/s at 20Hz

	// MatchTimeoutTicks ends a stalemate; the participant with more HP wins.
	MatchTimeoutTicks = 20 * 180

	// Dummy opposition used by the practice match.
	DummyHP            = 800.0
	DummyAttack        = 40.0
	DummyDefense       = 10.0
	DummySpeed         = 14.0
	DummyThinkTicks    = 12 // ~600ms of reaction delay between casts
	DummyAimErrorTiles = 0.6
)
