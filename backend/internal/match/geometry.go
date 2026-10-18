package match

import "math"

// HitCircle reports whether point falls inside a circle of the given radius,
// expanded by the actor radius so point-blank casts still connect.
func HitCircle(center, point Vec2, radius float64) bool {
	return center.Dist(point) <= radius+ActorRadius
}

// HitLine reports whether point falls inside a capsule of the given length and
// width, starting at origin and pointing along dir (expected normalized). This
// is the skillshot sweep: a projectile reuses it with length = distance
// travelled so far.
func HitLine(origin, dir, point Vec2, length, width float64) bool {
	rel := point.Sub(origin)
	along := rel.X*dir.X + rel.Y*dir.Y
	along = math.Max(0, math.Min(length, along))
	closest := origin.Add(dir.Scale(along))
	return closest.Dist(point) <= width/2+ActorRadius
}

// HitCone reports whether point falls inside a sector of the given length and
// opening angle (degrees), starting at origin and pointing along dir.
func HitCone(origin, dir, point Vec2, length, angleDeg float64) bool {
	rel := point.Sub(origin)
	dist := rel.Len()
	if dist > length+ActorRadius {
		return false
	}
	if dist == 0 {
		return true
	}
	cosine := (rel.X*dir.X + rel.Y*dir.Y) / dist
	cosine = math.Max(-1, math.Min(1, cosine))
	half := angleDeg * math.Pi / 360
	// Widen with distance so a target standing inside the actor radius at
	// point-blank range is not missed by the sector test.
	half += math.Atan2(ActorRadius, dist)
	return math.Acos(cosine) <= half
}

// blockedByObstacle reports whether an actor at point would overlap geometry.
func blockedByObstacle(point Vec2, obstacles []Obstacle) bool {
	for _, obstacle := range obstacles {
		switch obstacle.Kind {
		case "circle":
			if point.Dist(Vec2{obstacle.X, obstacle.Y}) < obstacle.R+ActorRadius {
				return true
			}
		case "rect":
			if point.X > obstacle.X-ActorRadius && point.X < obstacle.X+obstacle.W+ActorRadius &&
				point.Y > obstacle.Y-ActorRadius && point.Y < obstacle.Y+obstacle.H+ActorRadius {
				return true
			}
		}
	}
	return false
}

// clampToBounds keeps an actor inside the arena.
func clampToBounds(point Vec2, bounds Vec2) Vec2 {
	return Vec2{
		X: math.Max(ActorRadius, math.Min(bounds.X-ActorRadius, point.X)),
		Y: math.Max(ActorRadius, math.Min(bounds.Y-ActorRadius, point.Y)),
	}
}

// slideMove applies delta from pos, sliding along obstacles: the full move is
// tried first, then each axis alone, then the actor stays put. Cheap and
// predictable, which matters because the client predicts this exact function.
func slideMove(pos, delta Vec2, obstacles []Obstacle, bounds Vec2) Vec2 {
	try := func(step Vec2) (Vec2, bool) {
		next := clampToBounds(pos.Add(step), bounds)
		if blockedByObstacle(next, obstacles) {
			return Vec2{}, false
		}
		return next, true
	}
	if next, ok := try(delta); ok {
		return next
	}
	if next, ok := try(Vec2{X: delta.X}); ok {
		return next
	}
	if next, ok := try(Vec2{Y: delta.Y}); ok {
		return next
	}
	return pos
}
