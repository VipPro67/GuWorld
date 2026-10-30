package sim

import (
	"math"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

// Deterministic combat geometry. These helpers are pure functions of their
// arguments: point-to-segment distance and the AoE shape intersection tests
// (isInsideGroundArea). They form the testable "intent in, result out" seam
// of the AoE engine and can be exercised without constructing a zone.

func distance(left, right Vec2) float64 {
	return math.Hypot(left.X-right.X, left.Y-right.Y)
}

// segmentDistance returns the shortest distance from a point to a segment.
func segmentDistance(from, to, point Vec2) float64 {
	dx, dy := to.X-from.X, to.Y-from.Y
	lengthSquared := dx*dx + dy*dy
	if lengthSquared == 0 {
		return math.Hypot(point.X-from.X, point.Y-from.Y)
	}
	t := ((point.X-from.X)*dx + (point.Y-from.Y)*dy) / lengthSquared
	t = math.Max(0, math.Min(1, t))
	closestX, closestY := from.X+t*dx, from.Y+t*dy
	return math.Hypot(point.X-closestX, point.Y-closestY)
}

func isInsideGroundArea(casterPos, groundPoint, aim Vec2, targetPos Vec2, targetRadius float64, skill combat.Skill) bool {
	switch skill.Shape {
	case combat.ShapeCone:
		dir := Vec2{X: groundPoint.X - casterPos.X, Y: groundPoint.Y - casterPos.Y}
		if math.Hypot(dir.X, dir.Y) <= 0.01 {
			dir = aim
		}
		dirLen := math.Hypot(dir.X, dir.Y)
		if dirLen <= 0.01 {
			dir = Vec2{X: 1, Y: 0}
			dirLen = 1
		}
		toTarget := Vec2{X: targetPos.X - casterPos.X, Y: targetPos.Y - casterPos.Y}
		targetDist := math.Hypot(toTarget.X, toTarget.Y)
		maxRange := skill.Radius
		if maxRange <= 0 {
			maxRange = skill.Range
		}
		if targetDist > maxRange+targetRadius {
			return false
		}
		if targetDist <= targetRadius {
			return true
		}
		cosAngle := (dir.X*toTarget.X + dir.Y*toTarget.Y) / (dirLen * targetDist)
		halfAngleDeg := skill.AngleDeg / 2
		if halfAngleDeg <= 0 {
			halfAngleDeg = 30
		}
		minCos := math.Cos(halfAngleDeg * math.Pi / 180)
		return cosAngle >= minCos

	case combat.ShapeRectangle, combat.ShapeWall:
		length := skill.LengthUnits
		if length <= 0 {
			length = skill.Range
		}
		width := skill.WidthUnits
		if width <= 0 {
			width = 2.0
		}
		dir := Vec2{X: groundPoint.X - casterPos.X, Y: groundPoint.Y - casterPos.Y}
		if math.Hypot(dir.X, dir.Y) <= 0.01 {
			dir = aim
		}
		dirLen := math.Hypot(dir.X, dir.Y)
		if dirLen <= 0.01 {
			dir = Vec2{X: 1, Y: 0}
			dirLen = 1
		}
		u := Vec2{X: dir.X / dirLen, Y: dir.Y / dirLen}
		v := Vec2{X: targetPos.X - casterPos.X, Y: targetPos.Y - casterPos.Y}
		dotParallel := v.X*u.X + v.Y*u.Y
		distPerp := math.Abs(v.X*u.Y - v.Y*u.X)
		return dotParallel >= -targetRadius && dotParallel <= length+targetRadius && distPerp <= width/2+targetRadius

	case combat.ShapeDonut:
		d := distance(targetPos, groundPoint)
		outer := skill.Radius
		inner := skill.InnerRadius
		if inner >= outer {
			inner = outer * 0.5
		}
		return d <= outer+targetRadius && d >= inner-targetRadius

	case combat.ShapeCross:
		arm := skill.LengthUnits
		if arm <= 0 {
			arm = skill.Radius
		}
		width := skill.WidthUnits
		if width <= 0 {
			width = 1.5
		}
		dx := math.Abs(targetPos.X - groundPoint.X)
		dy := math.Abs(targetPos.Y - groundPoint.Y)
		inHorizontal := dx <= arm+targetRadius && dy <= width/2+targetRadius
		inVertical := dy <= arm+targetRadius && dx <= width/2+targetRadius
		return inHorizontal || inVertical

	default:
		return distance(targetPos, groundPoint) <= skill.Radius+targetRadius
	}
}
