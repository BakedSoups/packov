package game

import "math"

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func V(x, y float64) Vec2 { return Vec2{X: x, Y: y} }

func (a Vec2) Add(b Vec2) Vec2    { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2    { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Mul(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) Len() float64       { return math.Hypot(a.X, a.Y) }
func (a Vec2) Len2() float64      { return a.X*a.X + a.Y*a.Y }

func (a Vec2) Normalize() Vec2 {
	l := a.Len()
	if l <= 0.0001 {
		return Vec2{}
	}
	return Vec2{a.X / l, a.Y / l}
}

func (a Vec2) Clamp(max float64) Vec2 {
	if a.Len2() <= max*max {
		return a
	}
	return a.Normalize().Mul(max)
}

func Dist(a, b Vec2) float64 { return a.Sub(b).Len() }

func Angle(v Vec2) float64 { return math.Atan2(v.Y, v.X) }

func FromAngle(rad float64) Vec2 { return Vec2{math.Cos(rad), math.Sin(rad)} }
