package match

import (
	"math"
	"testing"
)

func TestHitCircle(t *testing.T) {
	center := Vec2{5, 5}
	if !HitCircle(center, Vec2{5, 5.4}, 1) {
		t.Fatal("point inside the radius should hit")
	}
	if HitCircle(center, Vec2{5, 7}, 1) {
		t.Fatal("point outside the radius should miss")
	}
}

func TestHitLineIsACapsule(t *testing.T) {
	origin := Vec2{0, 0}
	dir := Vec2{1, 0}
	const length, width = 10, 2

	if !HitLine(origin, dir, Vec2{5, 0.9}, length, width) {
		t.Fatal("point inside the capsule should hit")
	}
	if HitLine(origin, dir, Vec2{5, 2}, length, width) {
		t.Fatal("point beside the capsule should miss")
	}
	if HitLine(origin, dir, Vec2{12, 0}, length, width) {
		t.Fatal("point beyond the length should miss")
	}
	if HitLine(origin, dir, Vec2{-3, 0}, length, width) {
		t.Fatal("point behind the origin should miss")
	}
}

func TestHitConeSector(t *testing.T) {
	origin := Vec2{0, 0}
	dir := Vec2{1, 0}
	const length, angle = 9, 70

	if !HitCone(origin, dir, Vec2{5, 0}, length, angle) {
		t.Fatal("point on the axis should hit")
	}
	inside := 30 * math.Pi / 180
	if !HitCone(origin, dir, Vec2{5 * math.Cos(inside), 5 * math.Sin(inside)}, length, angle) {
		t.Fatal("point inside the sector should hit")
	}
	outside := 55 * math.Pi / 180
	if HitCone(origin, dir, Vec2{5 * math.Cos(outside), 5 * math.Sin(outside)}, length, angle) {
		t.Fatal("point outside the sector should miss")
	}
	if HitCone(origin, dir, Vec2{12, 0}, length, angle) {
		t.Fatal("point beyond the cone length should miss")
	}
}

func TestSlideMoveBlocksAndSlides(t *testing.T) {
	obstacles := []Obstacle{{Kind: "circle", X: 5, Y: 5, R: 1}}
	bounds := Vec2{20, 20}

	straight := slideMove(Vec2{3, 5}, Vec2{1, 0}, obstacles, bounds)
	if straight.Dist(Vec2{5, 5}) < 1+ActorRadius {
		t.Fatalf("walked into the pillar: %+v", straight)
	}

	diagonal := slideMove(Vec2{3, 5}, Vec2{1, 1}, obstacles, bounds)
	if diagonal == (Vec2{3, 5}) {
		t.Fatal("diagonal move into a pillar should slide on the free axis")
	}
	if blockedByObstacle(diagonal, obstacles) {
		t.Fatalf("slid into the pillar: %+v", diagonal)
	}

	clamped := slideMove(Vec2{19.9, 10}, Vec2{5, 0}, obstacles, bounds)
	if clamped.X > 20-ActorRadius+1e-9 {
		t.Fatalf("escaped the arena bounds: %+v", clamped)
	}
}
