package sim

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// These tests exercise the AoE shape engine through its pure seam: intent in
// (caster point, ground point, aim, skill) → result out, with no zone needed.

func shapeCone(angleDeg float64) combat.Skill {
	return combat.Skill{Shape: combat.ShapeCone, AngleDeg: angleDeg, Range: 5, Radius: 5}
}

func TestIsInsideGroundAreaConeChecksAngleAndRange(t *testing.T) {
	caster := Vec2{X: 0, Y: 0}
	point := Vec2{X: 5, Y: 0} // ground point east of the caster
	aim := Vec2{X: 1, Y: 0}
	targetOnAxis := Vec2{X: 3, Y: 0}
	targetOutsideAngle := Vec2{X: 3, Y: 4} // ~53° off-axis from the ground point

	if !isInsideGroundArea(caster, point, aim, targetOnAxis, 0.2, shapeCone(60)) {
		t.Fatalf("target on the cone axis should be inside")
	}
	skill := shapeCone(20)
	// 20° cone (±10°) cannot reach a target 37° off the ground-point axis.
	if isInsideGroundArea(caster, point, aim, targetOutsideAngle, 0.2, skill) {
		t.Fatalf("fast target far outside a narrow cone should be outside")
	}
	farTarget := Vec2{X: 12, Y: 0}
	if isInsideGroundArea(caster, point, aim, farTarget, 0.2, shapeCone(60)) {
		t.Fatalf("target beyond max range should be outside")
	}
	// A target centered on the caster is always in radius regardless of angle.
	if !isInsideGroundArea(caster, point, aim, caster, 0.2, shapeCone(5)) {
		t.Fatalf("target within the caster radius should always be inside")
	}
}

func TestIsInsideGroundAreaConeFallsBackToAimWhenPointOverlapsCaster(t *testing.T) {
	caster := Vec2{X: 0, Y: 0}
	point := Vec2{X: 0, Y: 0}
	aim := Vec2{X: 0, Y: 1}
	targetAlongAim := Vec2{X: 0, Y: 2}
	if !isInsideGroundArea(caster, point, aim, targetAlongAim, 0.2, shapeCone(60)) {
		t.Fatalf("aim fallback should orient the cone toward the target")
	}
	targetOpposite := Vec2{X: 0, Y: -2}
	if isInsideGroundArea(caster, point, aim, targetOpposite, 0.2, shapeCone(60)) {
		t.Fatalf("cone must not flip to the opposite direction via the aim fallback")
	}
}

func TestIsInsideGroundAreaRectangleChecksLengthAndWidth(t *testing.T) {
	// The rectangle body extends from the caster along the direction toward
	// the ground point; LengthUnits is measured from the caster position.
	caster := Vec2{X: 0, Y: 0}
	point := Vec2{X: 4, Y: 0}
	aim := Vec2{X: 1, Y: 0}
	skill := combat.Skill{Shape: combat.ShapeRectangle, Range: 6, LengthUnits: 6, WidthUnits: 2}

	if !isInsideGroundArea(caster, point, aim, Vec2{X: 3, Y: 0.5}, 0.1, skill) {
		t.Fatalf("target inside the rectangle body should be inside")
	}
	if isInsideGroundArea(caster, point, aim, Vec2{X: 6.8, Y: 0}, 0.1, skill) {
		t.Fatalf("target beyond length plus one target radius should be outside")
	}
	if isInsideGroundArea(caster, point, aim, Vec2{X: 8.5, Y: 0}, 0.1, skill) {
		t.Fatalf("target far beyond the rectangle end should be outside")
	}
	if isInsideGroundArea(caster, point, aim, Vec2{X: 3, Y: 1.6}, 0.1, skill) {
		t.Fatalf("target beyond half width plus radius should be outside")
	}
	// Direction falls back to the aim vector when the point sits on the caster.
	if !isInsideGroundArea(caster, Vec2{X: 0, Y: 0}, aim, Vec2{X: 3, Y: 0.5}, 0.1, skill) {
		t.Fatalf("aim fallback should orient the rectangle along the aim")
	}
}

func TestIsInsideGroundAreaDonutChecksInnerAndOuterRings(t *testing.T) {
	caster := Vec2{X: 0, Y: 0}
	point := Vec2{X: 3, Y: 0}
	skill := combat.Skill{Shape: combat.ShapeDonut, Radius: 3, InnerRadius: 1}

	if !isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 1.6, Y: 0}, 0.1, skill) {
		t.Fatalf("target in the ring band should be inside")
	}
	if isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 3.05, Y: 0}, 0.1, skill) {
		t.Fatalf("target in the donut hole should be outside")
	}
	if isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 6.5, Y: 0}, 0.1, skill) {
		t.Fatalf("target beyond the outer ring should be outside")
	}
	// inner >= outer degenerates to a half-radius inner hole.
	degenerate := skill
	degenerate.InnerRadius = 5
	if isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 3.05, Y: 0}, 0.1, degenerate) {
		t.Fatalf("degenerate donut should clamp the inner radius to half")
	}
}

func TestIsInsideGroundAreaCrossChecksBothArms(t *testing.T) {
	caster := Vec2{X: 0, Y: 0}
	point := Vec2{X: 3, Y: 2}
	skill := combat.Skill{Shape: combat.ShapeCross, Radius: 3, LengthUnits: 3, WidthUnits: 1}

	if !isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 4, Y: 2}, 0.1, skill) {
		t.Fatalf("target on the horizontal arm should be inside")
	}
	if !isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 3, Y: 3.5}, 0.1, skill) {
		t.Fatalf("target on the vertical arm should be inside")
	}
	if isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 4, Y: 3}, 0.1, skill) {
		t.Fatalf("target outside both arms should be outside")
	}
	// LengthUnits falling back to Radius keeps the arm symmetric.
	fallback := skill
	fallback.LengthUnits = 0
	if !isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 4, Y: 2}, 0.1, fallback) {
		t.Fatalf("missing length should fall back to the skill radius")
	}
}

func TestIsInsideGroundAreaDefaultShapeIsCircle(t *testing.T) {
	caster := Vec2{X: 0, Y: 0}
	point := Vec2{X: 2, Y: 0}
	skill := combat.Skill{Shape: combat.ShapeCircle, Radius: 2}

	if !isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 3, Y: 0}, 0.5, skill) {
		t.Fatalf("target within radius plus its own radius should be inside")
	}
	if isInsideGroundArea(caster, point, Vec2{}, Vec2{X: 6, Y: 0}, 0.1, skill) {
		t.Fatalf("target beyond the circle should be outside")
	}
}

func TestSegmentDistance(t *testing.T) {
	from, to := Vec2{X: 0, Y: 0}, Vec2{X: 4, Y: 0}
	if got := segmentDistance(from, to, Vec2{X: 2, Y: 1}); math.Abs(got-1) > 1e-9 {
		t.Fatalf("perpendicular distance = %f, want 1", got)
	}
	if got := segmentDistance(from, to, Vec2{X: 6, Y: 0}); math.Abs(got-2) > 1e-9 {
		t.Fatalf("beyond-end distance = %f, want 2 (clamped to the endpoint)", got)
	}
	if got := segmentDistance(Vec2{X: 1, Y: 1}, Vec2{X: 1, Y: 1}, Vec2{X: 4, Y: 5}); math.Abs(got-5) > 1e-9 {
		t.Fatalf("degenerate segment distance = %f, want 5", got)
	}
}
