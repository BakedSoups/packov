package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"packov/internal/game"
)

type styleTokens struct {
	Outline       color.Color
	OutlineStroke float32
	Player        color.RGBA
	PlayerHit     color.RGBA
	PlayerNose    color.RGBA
	Enemy         color.RGBA
	EnemyHit      color.RGBA
	Boss          color.RGBA
	BossHit       color.RGBA
	BossAccent    color.RGBA
	Bullet        color.RGBA
	Drone         color.RGBA
	Resource      color.RGBA
	Objective     color.RGBA
}

var primitiveStyle = styleTokens{
	Outline:       color.RGBA{37, 47, 60, 255},
	OutlineStroke: 6,
	Player:        color.RGBA{47, 178, 255, 255},
	PlayerHit:     color.RGBA{255, 255, 255, 255},
	PlayerNose:    color.RGBA{232, 248, 255, 255},
	Enemy:         color.RGBA{255, 91, 94, 235},
	EnemyHit:      color.RGBA{255, 238, 238, 255},
	Boss:          color.RGBA{199, 86, 255, 235},
	BossHit:       color.RGBA{255, 238, 255, 255},
	BossAccent:    color.RGBA{255, 210, 84, 230},
	Bullet:        color.RGBA{255, 221, 76, 255},
	Drone:         color.RGBA{79, 235, 186, 235},
	Resource:      color.RGBA{62, 214, 139, 235},
	Objective:     color.RGBA{247, 205, 92, 235},
}

func DrawPlayerShip(screen *ebiten.Image, center game.Vec2, radius, rotation float64, shield, hit bool) {
	fill := primitiveStyle.Player
	if hit {
		fill = primitiveStyle.PlayerHit
	}
	drawOutlinedCircle(screen, center, radius, fill, primitiveStyle.OutlineStroke)
	nose := []game.Vec2{
		center.Add(game.FromAngle(rotation).Mul(radius + 12)),
		center.Add(game.FromAngle(rotation + 2.55).Mul(radius * 0.68)),
		center.Add(game.FromAngle(rotation - 2.55).Mul(radius * 0.68)),
	}
	fillTriangle(screen, nose, primitiveStyle.PlayerNose)
	engine := center.Sub(game.FromAngle(rotation).Mul(radius + 9))
	drawOutlinedCircle(screen, engine, radius*0.24, color.RGBA{80, 212, 255, 180}, 2)
	if shield {
		vector.StrokeCircle(screen, float32(center.X), float32(center.Y), float32(radius+9), 5, color.RGBA{46, 58, 74, 185}, false)
		vector.StrokeCircle(screen, float32(center.X), float32(center.Y), float32(radius+14), 3, color.RGBA{47, 178, 255, 185}, false)
	}
}

func DrawEnemyShape(screen *ebiten.Image, center game.Vec2, radius float64, sides int, rotation float64, hit bool) {
	fill := primitiveStyle.Enemy
	if hit {
		fill = primitiveStyle.EnemyHit
	}
	drawPolygonWithOutline(screen, center, radius, sides, rotation, fill, primitiveStyle.OutlineStroke)
}

func DrawBossModule(screen *ebiten.Image, center game.Vec2, radius float64, phase int, rotation float64, hit bool) {
	fill := primitiveStyle.Boss
	if hit {
		fill = primitiveStyle.BossHit
	}
	drawPolygonWithOutline(screen, center, radius, 8, rotation, fill, primitiveStyle.OutlineStroke+1)
	vector.StrokeCircle(screen, float32(center.X), float32(center.Y), float32(radius+18+float64(phase)*12), 4, primitiveStyle.BossAccent, false)
	for i := 0; i < 4+phase; i++ {
		ang := rotation + float64(i)*math.Pi*2/float64(4+phase)
		module := center.Add(game.FromAngle(ang).Mul(radius + 36))
		drawPolygonWithOutline(screen, module, 13, 4, ang, primitiveStyle.BossAccent, primitiveStyle.OutlineStroke-1)
	}
}

func DrawProjectile(screen *ebiten.Image, center game.Vec2, radius float64, weaponID string, rotation float64) {
	switch weaponID {
	case "machine_gun":
		drawOutlinedCircle(screen, center, radius+1, color.RGBA{255, 221, 76, 255}, 3)
	case "shotgun":
		drawOutlinedCircle(screen, center, radius, color.RGBA{255, 236, 141, 245}, 2.5)
	case "railgun":
		dir := game.FromAngle(rotation)
		a := center.Sub(dir.Mul(radius * 2.2))
		b := center.Add(dir.Mul(radius * 3.6))
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 11, outlineColor(), false)
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 5, color.RGBA{238, 247, 255, 255}, false)
	case "laser":
		dir := game.FromAngle(rotation)
		a := center.Sub(dir.Mul(radius * 1.2))
		b := center.Add(dir.Mul(radius * 5.5))
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 8, outlineColor(), false)
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 4, color.RGBA{255, 72, 158, 255}, false)
	case "flamethrower":
		drawOutlinedCircle(screen, center, radius+6, color.RGBA{255, 112, 48, 200}, 3)
		vector.StrokeCircle(screen, float32(center.X), float32(center.Y), float32(radius+12), 2, color.RGBA{255, 181, 57, 110}, false)
	case "rocket_launcher":
		back := center.Sub(game.FromAngle(rotation).Mul(radius + 8))
		drawOutlinedCircle(screen, back, radius*0.8, color.RGBA{255, 112, 48, 160}, 2)
		drawPolygonWithOutline(screen, center, radius+5, 3, rotation, color.RGBA{255, 146, 62, 255}, primitiveStyle.OutlineStroke-2)
	case "plasma_cannon":
		drawOutlinedCircle(screen, center, radius+5, color.RGBA{140, 104, 255, 255}, primitiveStyle.OutlineStroke-2)
		vector.StrokeCircle(screen, float32(center.X), float32(center.Y), float32(radius+12), 3, color.RGBA{184, 92, 255, 130}, false)
	default:
		drawOutlinedCircle(screen, center, radius+1, primitiveStyle.Bullet, 3)
	}
}

func DrawLootNode(screen *ebiten.Image, center game.Vec2, rarity game.Rarity, tick uint64) {
	drawPolygonWithOutline(screen, center, 10, 6, float64(tick)/18, rarityColor(rarity), primitiveStyle.OutlineStroke-1)
}

func DrawAbilityEffect(screen *ebiten.Image, center game.Vec2, radius float64, tick uint64) {
	drawPolygonWithOutline(screen, center, radius, 4, float64(tick)/12, primitiveStyle.Drone, primitiveStyle.OutlineStroke-1)
}

func DrawStationBackdrop(screen *ebiten.Image, center game.Vec2, tick uint64) {
	t := float64(tick) / 60
	for ring := 0; ring < 4; ring++ {
		radius := 72 + float64(ring)*42
		count := 9 + ring*4
		spin := t*(0.35+float64(ring)*0.13) + float64(ring)*0.8
		vector.StrokeCircle(screen, float32(center.X), float32(center.Y), float32(radius), 2, color.RGBA{55, 78, 98, 72}, false)
		for i := 0; i < count; i++ {
			ang := spin + float64(i)*math.Pi*2/float64(count)
			breathe := 1 + math.Sin(t*2+float64(i))*0.18
			p := center.Add(game.FromAngle(ang).Mul(radius))
			alpha := uint8(105 + ring*26)
			drawOutlinedCircle(screen, p, (4+float64(ring))*breathe, color.RGBA{103, 228, 155, alpha}, 2)
		}
	}
	drawOutlinedCircle(screen, center, 10+math.Sin(t*3)*2, color.RGBA{247, 205, 92, 210}, 3)
}
