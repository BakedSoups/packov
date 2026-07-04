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
	catalog      *game.Catalog
	run          *game.RunState
	net          *wsClient
	screen       screenState
	player       game.PlayerID
	account      *game.Account
	loadout      game.Loadout
	listings     []game.MarketplaceListing
	settings     clientSettings
	seq          uint64
	camera       game.Vec2
	trails       []trail
	started      time.Time
	snapshotAt   time.Time
	prevEntities map[game.EntityID]game.Entity
	status       string
	uiNotice     string
	uiNoticeTick uint64
	remote       bool
	hello        bool
	queued       bool
	localSettled bool
	menu         menuState
	keys         map[ebiten.Key]bool
	look         game.Appearance
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
	look := game.DefaultAppearance("Pilot")
	localAccount := game.NewAccount(player, "Pilot")
	app := &App{catalog: c, run: run, player: player, account: &localAccount, loadout: game.DefaultLoadout(), settings: defaultSettings(), started: time.Now(), status: "local fallback", screen: screenTitle, keys: map[ebiten.Key]bool{}, look: look}
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
	defer a.captureKeys()
	a.seq++
	if a.screen != screenRun {
		a.updateMenu()
		a.updateTrails()
		return nil
	}
	if a.justPressed(ebiten.KeyEscape) {
		a.screen = screenStation
		a.remote = false
		return nil
	}
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
		if a.run.Phase == game.PhaseComplete || a.run.Phase == game.PhaseFailed {
			a.settleLocalRun()
			a.screen = screenResult
		}
	}
	if ps := a.run.Players[a.player]; ps != nil {
		if e := a.run.Entities[ps.EntityID]; e != nil {
			a.camera = e.Position
			if e.Velocity.Len2() > 20 {
				a.trails = append(a.trails, trail{Pos: e.Position.Sub(game.FromAngle(e.Rotation).Mul(25)), TTL: 0.35})
			}
		}
	}
	a.updateTrails()
	return nil
}

func (a *App) settleLocalRun() {
	if a.remote || a.localSettled || a.account == nil || a.run == nil {
		return
	}
	a.localSettled = true
	if a.run.Phase != game.PhaseComplete {
		return
	}
	if ps := a.run.Players[a.player]; ps != nil && ps.Extracted {
		for item, count := range ps.Carried.Items {
			a.account.Inventory.Add(item, count)
		}
	}
}

func (a *App) updateTrails() {
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
			if msg.Account != nil {
				a.account = msg.Account
				a.look = msg.Account.Appearance
				if a.look.Callsign == "" {
					a.look = game.DefaultAppearance(msg.Account.Name)
				}
			}
			if msg.Catalog != nil {
				msg.Catalog.BuildIndexes()
				a.catalog = msg.Catalog
			}
		case "match", "snapshot":
			if msg.Snapshot != nil {
				a.applySnapshot(*msg.Snapshot)
				a.remote = true
				if msg.Snapshot.Phase == game.PhaseComplete || msg.Snapshot.Phase == game.PhaseFailed {
					a.screen = screenResult
				} else {
					a.screen = screenRun
				}
				a.queued = true
				a.status = "authoritative server"
			}
		case "world_event":
			if msg.WorldEvent != nil {
				a.run.Messages = append(a.run.Messages, "World event: "+msg.WorldEvent.Name)
			}
		case "market":
			a.listings = msg.Listings
			a.status = fmt.Sprintf("market listings %d", len(a.listings))
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
	a.prevEntities = map[game.EntityID]game.Entity{}
	if a.run.Entities != nil {
		for id, e := range a.run.Entities {
			a.prevEntities[id] = *e
		}
	}
	a.snapshotAt = time.Now()
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
	screen.Fill(color.RGBA{236, 242, 248, 255})
	if a.screen != screenRun {
		a.drawMenu(screen)
		return
	}
	a.drawGrid(screen)
	a.drawMap(screen)
	for _, t := range a.trails {
		p := worldToScreen(a.camera, t.Pos)
		drawOutlinedCircle(screen, p, 9*t.TTL, color.RGBA{58, 202, 255, uint8(135 * t.TTL / 0.35)}, 2)
	}
	for _, e := range a.run.Entities {
		entity := a.renderEntity(e)
		a.drawEntity(screen, &entity)
	}
	a.drawCombatHUD(screen)
	a.drawHUD(screen)
}

func (a *App) renderEntity(e *game.Entity) game.Entity {
	out := *e
	if !a.remote || a.prevEntities == nil || a.snapshotAt.IsZero() {
		return out
	}
	prev, ok := a.prevEntities[e.ID]
	if !ok {
		return out
	}
	alpha := time.Since(a.snapshotAt).Seconds() * game.TickRate
	if alpha < 0 {
		alpha = 0
	}
	if alpha > 1 {
		alpha = 1
	}
	out.Position = prev.Position.Mul(1 - alpha).Add(e.Position.Mul(alpha))
	out.Rotation = prev.Rotation + (e.Rotation-prev.Rotation)*alpha
	out.HP = prev.HP + (e.HP-prev.HP)*alpha
	return out
}

func (a *App) Layout(_, _ int) (int, int) {
	return screenW, screenH
}

func (a *App) drawGrid(screen *ebiten.Image) {
	grid := 80.0
	offsetX := math.Mod(-a.camera.X+screenW/2, grid)
	offsetY := math.Mod(-a.camera.Y+screenH/2, grid)
	for x := offsetX; x < screenW; x += grid {
		vector.StrokeLine(screen, float32(x), 0, float32(x), screenH, 2, color.RGBA{204, 215, 226, 255}, false)
	}
	for y := offsetY; y < screenH; y += grid {
		vector.StrokeLine(screen, 0, float32(y), screenW, float32(y), 2, color.RGBA{204, 215, 226, 255}, false)
	}
}

func (a *App) drawMap(screen *ebiten.Image) {
	for _, h := range a.run.Map.Hazards {
		p := worldToScreen(a.camera, h.Position)
		vector.DrawFilledCircle(screen, float32(p.X), float32(p.Y), float32(h.Radius), color.RGBA{178, 104, 255, 38}, false)
		vector.StrokeCircle(screen, float32(p.X), float32(p.Y), float32(h.Radius), 5, color.RGBA{46, 58, 74, 180}, false)
		vector.StrokeCircle(screen, float32(p.X), float32(p.Y), float32(h.Radius-7), 3, color.RGBA{178, 104, 255, 170}, false)
	}
	for _, o := range a.run.Map.Objectives {
		p := worldToScreen(a.camera, o.Position)
		drawPolygon(screen, p, 16, 4, 0.7, color.RGBA{247, 205, 92, 235})
	}
	for _, r := range a.run.Map.Resources {
		p := worldToScreen(a.camera, r.Position)
		drawOutlinedCircle(screen, p, 8, color.RGBA{62, 214, 139, 235}, 3)
	}
	ep := worldToScreen(a.camera, a.run.Map.Extraction)
	vector.StrokeCircle(screen, float32(ep.X), float32(ep.Y), 130, 7, color.RGBA{46, 58, 74, 210}, false)
	vector.StrokeCircle(screen, float32(ep.X), float32(ep.Y), 121, 4, color.RGBA{47, 178, 255, 210}, false)
	vector.StrokeCircle(screen, float32(ep.X), float32(ep.Y), 90, 4, color.RGBA{47, 178, 255, 145}, false)
}

func (a *App) drawEntity(screen *ebiten.Image, e *game.Entity) {
	p := worldToScreen(a.camera, e.Position)
	switch e.Kind {
	case game.EntityPlayer:
		fill := color.RGBA{47, 178, 255, 255}
		if a.recentHit(e, 10) {
			fill = color.RGBA{255, 255, 255, 255}
		}
		drawOutlinedCircle(screen, p, e.Radius, fill, 5)
		nose := []game.Vec2{
			p.Add(game.FromAngle(e.Rotation).Mul(28)),
			p.Add(game.FromAngle(e.Rotation + 2.55).Mul(15)),
			p.Add(game.FromAngle(e.Rotation - 2.55).Mul(15)),
		}
		fillTriangle(screen, nose, color.RGBA{232, 248, 255, 255})
		if e.Shield > 0 {
			vector.StrokeCircle(screen, float32(p.X), float32(p.Y), float32(e.Radius+9), 5, color.RGBA{46, 58, 74, 185}, false)
			vector.StrokeCircle(screen, float32(p.X), float32(p.Y), float32(e.Radius+14), 3, color.RGBA{47, 178, 255, 185}, false)
		}
	case game.EntityEnemy:
		def := a.catalog.EnemyByID[e.DefID]
		sides := map[string]int{"triangle": 3, "square": 4, "hexagon": 6, "octagon": 8}[def.Shape]
		if sides == 0 {
			sides = 3
		}
		fill := color.RGBA{255, 91, 94, 235}
		if a.recentHit(e, 8) {
			fill = color.RGBA{255, 238, 238, 255}
		}
		drawPolygon(screen, p, e.Radius, sides, e.Rotation, fill)
		drawHealth(screen, p, e)
	case game.EntityBoss:
		fill := color.RGBA{199, 86, 255, 235}
		if a.recentHit(e, 8) {
			fill = color.RGBA{255, 238, 255, 255}
		}
		drawPolygon(screen, p, e.Radius, 8, e.Rotation, fill)
		vector.StrokeCircle(screen, float32(p.X), float32(p.Y), float32(e.Radius+18+float64(e.Phase)*12), 3, color.RGBA{255, 210, 84, 180}, false)
		for i := 0; i < 4+e.Phase; i++ {
			ang := e.Rotation + float64(i)*math.Pi*2/float64(4+e.Phase)
			module := p.Add(game.FromAngle(ang).Mul(e.Radius + 36))
			drawPolygon(screen, module, 13, 4, ang, color.RGBA{255, 210, 84, 230})
		}
		drawHealth(screen, p.Add(game.V(0, -e.Radius-24)), e)
	case game.EntityBullet:
		drawOutlinedCircle(screen, p, e.Radius+1, color.RGBA{255, 221, 76, 255}, 2.5)
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
	if len(a.run.Messages) > 0 {
		lines = append(lines, a.run.Messages[len(a.run.Messages)-1])
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 18, 18)
	_ = ps
}

func (a *App) drawCombatHUD(screen *ebiten.Image) {
	ps := a.run.Players[a.player]
	if ps == nil {
		return
	}
	player := a.run.Entities[ps.EntityID]
	if player == nil {
		return
	}
	if player.HP/player.MaxHP < 0.28 && a.run.Tick%20 < 10 {
		vector.StrokeRect(screen, 8, 8, screenW-16, screenH-16, 8, color.RGBA{255, 91, 94, 190}, false)
	}
	drawBar(screen, 28, screenH-72, 360, 22, player.HP/player.MaxHP, color.RGBA{80, 205, 104, 255}, "HULL")
	drawBar(screen, 28, screenH-42, 360, 14, math.Min(1, player.Shield/100), color.RGBA{47, 178, 255, 255}, "SHIELD")
	value := a.carriedValue(ps)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("CARRIED %d cr  %v", value, ps.Carried.Items), 410, screenH-68)
	if boss := a.primaryBoss(); boss != nil {
		drawBar(screen, screenW/2-240, 24, 480, 18, boss.HP/boss.MaxHP, color.RGBA{199, 86, 255, 255}, strings.ToUpper(a.catalog.BossByID[boss.DefID].Name))
	}
}

func drawBar(screen *ebiten.Image, x, y, w, h float32, pct float64, fill color.RGBA, label string) {
	pct = math.Max(0, math.Min(1, pct))
	vector.DrawFilledRect(screen, x-4, y-4, w+8, h+8, outlineColor(), false)
	vector.DrawFilledRect(screen, x, y, w, h, color.RGBA{255, 255, 255, 245}, false)
	vector.DrawFilledRect(screen, x, y, w*float32(pct), h, fill, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s %.0f%%", label, pct*100), int(x+8), int(y+2))
}

func (a *App) carriedValue(ps *game.PlayerState) int {
	total := 0
	for item, count := range ps.Carried.Items {
		if loot, ok := a.catalog.LootByID[item]; ok {
			total += loot.BaseValue * count
		}
	}
	return total
}

func (a *App) primaryBoss() *game.Entity {
	for _, e := range a.run.Entities {
		if e.Kind == game.EntityBoss {
			return e
		}
	}
	return nil
}

func (a *App) recentHit(e *game.Entity, ticks uint64) bool {
	return e.HitTick > 0 && a.run.Tick >= e.HitTick && a.run.Tick-e.HitTick <= ticks
}

func drawHealth(screen *ebiten.Image, p game.Vec2, e *game.Entity) {
	w := float32(e.Radius * 2)
	x := float32(p.X) - w/2
	y := float32(p.Y) - float32(e.Radius) - 12
	vector.DrawFilledRect(screen, x-2, y-2, w+4, 8, color.RGBA{46, 58, 74, 235}, false)
	vector.DrawFilledRect(screen, x, y, w, 4, color.RGBA{255, 255, 255, 240}, false)
	vector.DrawFilledRect(screen, x, y, w*float32(math.Max(0, e.HP/e.MaxHP)), 4, color.RGBA{80, 205, 104, 255}, false)
}

func drawPolygon(screen *ebiten.Image, center game.Vec2, radius float64, sides int, rotation float64, clr color.Color) {
	drawPolygonWithOutline(screen, center, radius, sides, rotation, clr, 5)
}

func drawPolygonWithOutline(screen *ebiten.Image, center game.Vec2, radius float64, sides int, rotation float64, clr color.Color, stroke float32) {
	var path vector.Path
	points := make([]game.Vec2, 0, sides)
	for i := 0; i < sides; i++ {
		p := center.Add(game.FromAngle(rotation + float64(i)*math.Pi*2/float64(sides)).Mul(radius))
		points = append(points, p)
		if i == 0 {
			path.MoveTo(float32(p.X), float32(p.Y))
		} else {
			path.LineTo(float32(p.X), float32(p.Y))
		}
	}
	path.Close()
	fillPath(screen, path, clr)
	for i := 0; i < len(points); i++ {
		a := points[i]
		b := points[(i+1)%len(points)]
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), stroke, outlineColor(), false)
	}
}

func drawOutlinedCircle(screen *ebiten.Image, center game.Vec2, radius float64, clr color.Color, stroke float32) {
	vector.DrawFilledCircle(screen, float32(center.X), float32(center.Y), float32(radius)+stroke, outlineColor(), false)
	vector.DrawFilledCircle(screen, float32(center.X), float32(center.Y), float32(radius), clr, false)
}

func fillPath(screen *ebiten.Image, path vector.Path, clr color.Color) {
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
	for i := 0; i < len(pts); i++ {
		a := pts[i]
		b := pts[(i+1)%len(pts)]
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 9, outlineColor(), false)
	}
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
	for i := 0; i < len(pts); i++ {
		a := pts[i]
		b := pts[(i+1)%len(pts)]
		vector.StrokeLine(screen, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), 4, outlineColor(), false)
	}
}

func outlineColor() color.Color {
	return color.RGBA{46, 58, 74, 255}
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
