package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"packov/internal/game"
	"packov/internal/protocol"
)

const (
	screenW = 1280
	screenH = 720
)

type App struct {
	catalog *game.Catalog
	run     *game.RunState
	net     *wsClient
	player  game.PlayerID
	seq     uint64
	camera  game.Vec2
	trails  []trail
	started time.Time
	status  string
	remote  bool
	hello   bool
	queued  bool
}

type trail struct {
	Pos game.Vec2
	TTL float64
}

func main() {
	ebiten.SetWindowTitle("Packov")
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetTPS(game.TickRate)
	app := newApp()
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}

func newApp() *App {
	c := game.DefaultCatalogForClient()
	run := game.NewRun("local-solo", c, "verdant", 1977)
	player := game.PlayerID("local-pilot")
	run.AddPlayer(player, "Pilot", game.DefaultLoadout())
	run.SpawnInitial(c)
	app := &App{catalog: c, run: run, player: player, started: time.Now(), status: "local fallback"}
	if net, err := newWSClient(); err == nil {
		app.net = net
		app.status = "connecting"
	} else {
		app.status = "offline: " + err.Error()
	}
	return app
}

func (a *App) Update() error {
	a.pollNetwork()
	a.seq++
	move := game.Vec2{}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		move.Y--
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		move.Y++
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		move.X--
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		move.X++
	}
	mx, my := ebiten.CursorPosition()
	aim := screenToWorld(a.camera, game.V(float64(mx), float64(my)))
	cmd := game.InputCommand{
		PlayerID: a.player,
		Seq:      a.seq,
		Move:     move,
		Aim:      aim,
		Fire:     ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
		Ability:  ebiten.IsKeyPressed(ebiten.KeySpace) || ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight),
		Extract:  ebiten.IsKeyPressed(ebiten.KeyE),
	}
	if a.remote && a.net != nil {
		a.net.send(protocol.ClientMessage{Type: "input", Input: cmd})
	} else {
		a.run.ApplyInput(cmd)
		a.run.Step(a.catalog)
	}
	if ps := a.run.Players[a.player]; ps != nil {
		if e := a.run.Entities[ps.EntityID]; e != nil {
			a.camera = e.Position
			if e.Velocity.Len2() > 20 {
				a.trails = append(a.trails, trail{Pos: e.Position.Sub(game.FromAngle(e.Rotation).Mul(25)), TTL: 0.35})
			}
		}
	}
	for i := range a.trails {
		a.trails[i].TTL -= 1.0 / game.TickRate
	}
	dst := a.trails[:0]
	for _, t := range a.trails {
		if t.TTL > 0 {
			dst = append(dst, t)
		}
	}
	a.trails = dst
	return nil
}

func (a *App) pollNetwork() {
	if a.net == nil {
		return
	}
	if a.net.isOpen() && !a.hello {
		a.hello = true
		a.net.send(protocol.ClientMessage{Type: "hello", Token: browserToken(), Name: "Pilot"})
		a.status = "authenticating"
	}
	for {
		msg, ok := a.net.next()
		if !ok {
			break
		}
		switch msg.Type {
		case "hello":
			a.status = "online"
			if msg.PlayerID != "" {
				a.player = msg.PlayerID
			}
			if msg.Catalog != nil {
				msg.Catalog.BuildIndexes()
				a.catalog = msg.Catalog
			}
			if !a.queued {
				a.queued = true
				a.net.send(protocol.ClientMessage{Type: "queue", PlanetID: "verdant", Loadout: game.DefaultLoadout()})
			}
		case "match", "snapshot":
			if msg.Snapshot != nil {
				a.applySnapshot(*msg.Snapshot)
				a.remote = true
				a.status = "authoritative server"
			}
		case "world_event":
			if msg.WorldEvent != nil {
				a.run.Messages = append(a.run.Messages, "World event: "+msg.WorldEvent.Name)
			}
		case "error":
			a.status = "server error: " + msg.Error
		}
	}
	if a.net.isClosed() && a.remote {
		a.status = "disconnected, rendering last snapshot"
	}
}

func (a *App) applySnapshot(s game.Snapshot) {
	if a.run == nil {
		a.run = &game.RunState{}
	}
	a.run.ID = s.RunID
	a.run.Tick = s.Tick
	a.run.Phase = s.Phase
	a.run.Map = s.Map
	a.run.Planet.Name = s.Planet
	a.run.Entities = map[game.EntityID]*game.Entity{}
	for i := range s.Entities {
		e := s.Entities[i]
		a.run.Entities[e.ID] = &e
	}
	a.run.Players = map[game.PlayerID]*game.PlayerState{}
	for i := range s.Players {
		p := s.Players[i]
		a.run.Players[p.ID] = &p
	}
	a.run.Messages = s.Messages
}

func (a *App) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{8, 12, 18, 255})
	a.drawGrid(screen)
	a.drawMap(screen)
	for _, t := range a.trails {
		p := worldToScreen(a.camera, t.Pos)
		vector.DrawFilledCircle(screen, float32(p.X), float32(p.Y), float32(8*t.TTL), color.RGBA{58, 202, 255, uint8(90 * t.TTL / 0.35)}, false)
	}
	for _, e := range a.run.Entities {
		a.drawEntity(screen, e)
	}
	a.drawHUD(screen)
}

func (a *App) Layout(_, _ int) (int, int) {
	return screenW, screenH
}

func (a *App) drawGrid(screen *ebiten.Image) {
	grid := 80.0
	offsetX := math.Mod(-a.camera.X+screenW/2, grid)
	offsetY := math.Mod(-a.camera.Y+screenH/2, grid)
	for x := offsetX; x < screenW; x += grid {
		vector.StrokeLine(screen, float32(x), 0, float32(x), screenH, 1, color.RGBA{24, 32, 42, 255}, false)
	}
	for y := offsetY; y < screenH; y += grid {
		vector.StrokeLine(screen, 0, float32(y), screenW, float32(y), 1, color.RGBA{24, 32, 42, 255}, false)
	}
}

func (a *App) drawMap(screen *ebiten.Image) {
	for _, h := range a.run.Map.Hazards {
		p := worldToScreen(a.camera, h.Position)
		vector.DrawFilledCircle(screen, float32(p.X), float32(p.Y), float32(h.Radius), color.RGBA{118, 61, 145, 45}, false)
		vector.StrokeCircle(screen, float32(p.X), float32(p.Y), float32(h.Radius), 2, color.RGBA{176, 82, 214, 150}, false)
	}
	for _, o := range a.run.Map.Objectives {
		p := worldToScreen(a.camera, o.Position)
		drawPolygon(screen, p, 16, 4, 0.7, color.RGBA{247, 205, 92, 235})
	}
	for _, r := range a.run.Map.Resources {
		p := worldToScreen(a.camera, r.Position)
		vector.DrawFilledCircle(screen, float32(p.X), float32(p.Y), 7, color.RGBA{62, 214, 139, 215}, false)
	}
	ep := worldToScreen(a.camera, a.run.Map.Extraction)
	vector.StrokeCircle(screen, float32(ep.X), float32(ep.Y), 130, 4, color.RGBA{80, 212, 255, 180}, false)
	vector.StrokeCircle(screen, float32(ep.X), float32(ep.Y), 95, 2, color.RGBA{80, 212, 255, 110}, false)
}

func (a *App) drawEntity(screen *ebiten.Image, e *game.Entity) {
	p := worldToScreen(a.camera, e.Position)
	switch e.Kind {
	case game.EntityPlayer:
		vector.DrawFilledCircle(screen, float32(p.X), float32(p.Y), float32(e.Radius), color.RGBA{47, 178, 255, 255}, false)
		nose := []game.Vec2{
			p.Add(game.FromAngle(e.Rotation).Mul(28)),
			p.Add(game.FromAngle(e.Rotation + 2.55).Mul(15)),
			p.Add(game.FromAngle(e.Rotation - 2.55).Mul(15)),
		}
		fillTriangle(screen, nose, color.RGBA{232, 248, 255, 255})
		if e.Shield > 0 {
			vector.StrokeCircle(screen, float32(p.X), float32(p.Y), float32(e.Radius+8), 3, color.RGBA{80, 212, 255, 170}, false)
		}
	case game.EntityEnemy:
		def := a.catalog.EnemyByID[e.DefID]
		sides := map[string]int{"triangle": 3, "square": 4, "hexagon": 6, "octagon": 8}[def.Shape]
		if sides == 0 {
			sides = 3
		}
		drawPolygon(screen, p, e.Radius, sides, e.Rotation, color.RGBA{255, 91, 94, 235})
		drawHealth(screen, p, e)
	case game.EntityBoss:
		drawPolygon(screen, p, e.Radius, 8, e.Rotation, color.RGBA{199, 86, 255, 235})
		vector.StrokeCircle(screen, float32(p.X), float32(p.Y), float32(e.Radius+18+float64(e.Phase)*12), 3, color.RGBA{255, 210, 84, 180}, false)
		for i := 0; i < 4+e.Phase; i++ {
			ang := e.Rotation + float64(i)*math.Pi*2/float64(4+e.Phase)
			module := p.Add(game.FromAngle(ang).Mul(e.Radius + 36))
			drawPolygon(screen, module, 13, 4, ang, color.RGBA{255, 210, 84, 230})
		}
		drawHealth(screen, p.Add(game.V(0, -e.Radius-24)), e)
	case game.EntityBullet:
		vector.DrawFilledCircle(screen, float32(p.X), float32(p.Y), float32(e.Radius), color.RGBA{255, 235, 120, 255}, false)
	case game.EntityLoot:
		drawPolygon(screen, p, 10, 6, float64(a.run.Tick)/18, rarityColor(a.catalog.LootByID[e.DefID].Rarity))
	case game.EntityDrone, game.EntityTurret:
		drawPolygon(screen, p, e.Radius, 4, float64(a.run.Tick)/12, color.RGBA{79, 235, 186, 235})
	}
}

func (a *App) drawHUD(screen *ebiten.Image) {
	ps := a.run.Players[a.player]
	lines := []string{
		"PACKOV  " + strings.ToUpper(string(a.run.Phase)) + "  " + a.run.Planet.Name,
		"WASD move  Mouse aim/fire  Space ability  E extract",
		fmt.Sprintf("Tick %d  Entities %d  Runtime %s  Net %s", a.run.Tick, len(a.run.Entities), time.Since(a.started).Truncate(time.Second), a.status),
	}
	if ps != nil {
		if e := a.run.Entities[ps.EntityID]; e != nil {
			lines = append(lines, fmt.Sprintf("Hull %.0f/%.0f  Shield %.0f  Carried %v", e.HP, e.MaxHP, e.Shield, ps.Carried.Items))
		}
	}
	if len(a.run.Messages) > 0 {
		lines = append(lines, a.run.Messages[len(a.run.Messages)-1])
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 18, 18)
}

func drawHealth(screen *ebiten.Image, p game.Vec2, e *game.Entity) {
	w := float32(e.Radius * 2)
	x := float32(p.X) - w/2
	y := float32(p.Y) - float32(e.Radius) - 12
	vector.DrawFilledRect(screen, x, y, w, 4, color.RGBA{45, 18, 24, 220}, false)
	vector.DrawFilledRect(screen, x, y, w*float32(math.Max(0, e.HP/e.MaxHP)), 4, color.RGBA{90, 235, 136, 230}, false)
}

func drawPolygon(screen *ebiten.Image, center game.Vec2, radius float64, sides int, rotation float64, clr color.Color) {
	var path vector.Path
	for i := 0; i < sides; i++ {
		p := center.Add(game.FromAngle(rotation + float64(i)*math.Pi*2/float64(sides)).Mul(radius))
		if i == 0 {
			path.MoveTo(float32(p.X), float32(p.Y))
		} else {
			path.LineTo(float32(p.X), float32(p.Y))
		}
	}
	path.Close()
	var vs []ebiten.Vertex
	var is []uint16
	vs, is = path.AppendVerticesAndIndicesForFilling(vs, is)
	for i := range vs {
		r, g, b, a := clr.RGBA()
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(g) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(a) / 0xffff
	}
	screen.DrawTriangles(vs, is, whiteImage(), nil)
}

func fillTriangle(screen *ebiten.Image, pts []game.Vec2, clr color.Color) {
	var path vector.Path
	path.MoveTo(float32(pts[0].X), float32(pts[0].Y))
	path.LineTo(float32(pts[1].X), float32(pts[1].Y))
	path.LineTo(float32(pts[2].X), float32(pts[2].Y))
	path.Close()
	var vs []ebiten.Vertex
	var is []uint16
	vs, is = path.AppendVerticesAndIndicesForFilling(vs, is)
	r, g, b, a := clr.RGBA()
	for i := range vs {
		vs[i].ColorR = float32(r) / 0xffff
		vs[i].ColorG = float32(g) / 0xffff
		vs[i].ColorB = float32(b) / 0xffff
		vs[i].ColorA = float32(a) / 0xffff
	}
	screen.DrawTriangles(vs, is, whiteImage(), nil)
}

var solid *ebiten.Image

func whiteImage() *ebiten.Image {
	if solid == nil {
		solid = ebiten.NewImage(1, 1)
		solid.Fill(color.White)
	}
	return solid
}

func rarityColor(r game.Rarity) color.Color {
	switch r {
	case game.Uncommon:
		return color.RGBA{66, 221, 126, 240}
	case game.Rare:
		return color.RGBA{77, 166, 255, 240}
	case game.Epic:
		return color.RGBA{184, 92, 255, 240}
	case game.Legendary:
		return color.RGBA{255, 181, 57, 240}
	case game.Relic:
		return color.RGBA{255, 72, 158, 240}
	default:
		return color.RGBA{220, 228, 238, 240}
	}
}

func worldToScreen(camera, world game.Vec2) game.Vec2 {
	return world.Sub(camera).Add(game.V(screenW/2, screenH/2))
}

func screenToWorld(camera, screen game.Vec2) game.Vec2 {
	return screen.Add(camera).Sub(game.V(screenW/2, screenH/2))
}
