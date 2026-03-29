package sim

import "math"

func clamp(value, minimum, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}

type Vec2 struct {
	X float64
	Y float64
}

func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vec2) Sub(other Vec2) Vec2 {
	return Vec2{X: v.X - other.X, Y: v.Y - other.Y}
}

func (v Vec2) Mul(scalar float64) Vec2 {
	return Vec2{X: v.X * scalar, Y: v.Y * scalar}
}

func (v Vec2) Div(scalar float64) Vec2 {
	return Vec2{X: v.X / scalar, Y: v.Y / scalar}
}

func (v Vec2) Length() float64 {
	return math.Hypot(v.X, v.Y)
}

func (v Vec2) Normalized() Vec2 {
	size := v.Length()
	if size < 1e-6 {
		return Vec2{}
	}
	return v.Div(size)
}

func (v Vec2) Limit(maximum float64) Vec2 {
	size := v.Length()
	if size <= maximum || size < 1e-6 {
		return v
	}
	return v.Mul(maximum / size)
}

func (v Vec2) Dot(other Vec2) float64 {
	return v.X*other.X + v.Y*other.Y
}

func distanceToSegment(point, start, end Vec2) float64 {
	segment := end.Sub(start)
	lengthSquared := segment.Dot(segment)
	if lengthSquared < 1e-6 {
		return point.Sub(start).Length()
	}
	t := clamp(point.Sub(start).Dot(segment)/lengthSquared, 0.0, 1.0)
	closest := start.Add(segment.Mul(t))
	return point.Sub(closest).Length()
}
