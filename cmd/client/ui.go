package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"packov/internal/game"
	"packov/internal/protocol"
)

type screenState int

const (
	screenTitle screenState = iota
	screenCharacter
	screenStation
	screenSettings
	screenRun
)

type menuState struct {
	Index       int
	EditIndex   int
	SettingsIdx int
}

var (
	titleItems    = []string{"Enter Station", "Edit Character", "Settings", "Local Test Run"}
	stationItems  = []string{"Deploy Verdant-9", "Loadout", "Inventory", "Crafting", "Marketplace", "Character", "Settings", "Title"}
	characterRows = []string{"Primary", "Secondary", "Trail", "Nose", "Drone Skin", "Badge"}
	settingsRows  = []string{"Mouse Aim", "Controller", "Damage Numbers", "Screen Shake", "Back"}
	colorOptions  = []string{"cyan", "white", "amber", "green", "violet", "red"}
	trailOptions  = []string{"ion", "spark", "comet", "pulse"}
	noseOptions   = []string{"arrow", "split", "needle"}
	droneOptions  = []string{"standard", "orbital", "wing"}
	badgeOptions  = []string{"founder", "boss", "guild", "season"}
)

func (a *App) updateMenu() {
	switch a.screen {
	case screenTitle:
		a.updateList(len(titleItems), func() {
			switch a.menu.Index {
			case 0:
				a.screen = screenStation
			case 1:
				a.screen = screenCharacter
			case 2:
				a.screen = screenSettings
			case 3:
				a.startLocalRun()
			}
		})
	case screenStation:
		a.updateList(len(stationItems), func() {
			switch stationItems[a.menu.Index] {
			case "Deploy Verdant-9":
				a.deploy()
			case "Character":
				a.screen = screenCharacter
			case "Settings":
				a.screen = screenSettings
			case "Title":
				a.screen = screenTitle
			}
		})
	case screenCharacter:
		a.updateCharacterEditor()
	case screenSettings:
		a.updateList(len(settingsRows), func() {
			if settingsRows[a.menu.Index] == "Back" {
				a.screen = screenStation
			}
		})
	}
}

func (a *App) updateList(count int, activate func()) {
	if count == 0 {
		return
	}
	if a.justPressed(ebiten.KeyArrowUp) || a.justPressed(ebiten.KeyW) {
		a.menu.Index = (a.menu.Index + count - 1) % count
	}
	if a.justPressed(ebiten.KeyArrowDown) || a.justPressed(ebiten.KeyS) {
		a.menu.Index = (a.menu.Index + 1) % count
	}
	if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
		activate()
	}
}

func (a *App) updateCharacterEditor() {
	if a.justPressed(ebiten.KeyEscape) {
		a.syncAppearance()
		a.screen = screenStation
		return
	}
	if a.justPressed(ebiten.KeyArrowUp) || a.justPressed(ebiten.KeyW) {
		a.menu.EditIndex = (a.menu.EditIndex + len(characterRows) - 1) % len(characterRows)
	}
	if a.justPressed(ebiten.KeyArrowDown) || a.justPressed(ebiten.KeyS) {
		a.menu.EditIndex = (a.menu.EditIndex + 1) % len(characterRows)
	}
	if a.justPressed(ebiten.KeyArrowLeft) || a.justPressed(ebiten.KeyA) {
		a.cycleAppearance(-1)
	}
	if a.justPressed(ebiten.KeyArrowRight) || a.justPressed(ebiten.KeyD) || a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
		a.cycleAppearance(1)
	}
	a.look.SpinOffset += 0.025
}

func (a *App) cycleAppearance(delta int) {
	switch a.menu.EditIndex {
	case 0:
		a.look.Primary = cycleString(colorOptions, a.look.Primary, delta)
	case 1:
		a.look.Secondary = cycleString(colorOptions, a.look.Secondary, delta)
	case 2:
		a.look.TrailID = cycleString(trailOptions, a.look.TrailID, delta)
	case 3:
		a.look.NoseID = cycleString(noseOptions, a.look.NoseID, delta)
	case 4:
		a.look.DroneSkin = cycleString(droneOptions, a.look.DroneSkin, delta)
	case 5:
		a.look.BadgeID = cycleString(badgeOptions, a.look.BadgeID, delta)
	}
}

func cycleString(options []string, current string, delta int) string {
	if len(options) == 0 {
		return current
	}
	idx := 0
	for i, option := range options {
		if option == current {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(options)) % len(options)
	return options[idx]
}

func (a *App) deploy() {
	if a.net != nil && a.net.isOpen() {
		a.queued = true
		a.status = "queueing"
		a.net.send(protocol.ClientMessage{Type: "queue", PlanetID: "verdant", Loadout: game.DefaultLoadout()})
		return
	}
	a.startLocalRun()
}

func (a *App) syncAppearance() {
	if a.account != nil {
		a.account.Appearance = a.look
	}
	if a.net != nil && a.net.isOpen() && a.hello {
		a.net.send(protocol.ClientMessage{Type: "appearance", Appearance: a.look})
	}
}

func (a *App) startLocalRun() {
	a.remote = false
	a.queued = false
	a.run = game.NewRun("local-solo", a.catalog, "verdant", timeSeed())
	a.run.AddPlayer(a.player, a.look.Callsign, game.DefaultLoadout())
	a.run.SpawnInitial(a.catalog)
	a.screen = screenRun
	a.status = "local fallback"
}

func (a *App) drawMenu(screen *ebiten.Image) {
	a.drawMenuBackground(screen)
	switch a.screen {
	case screenTitle:
		a.drawTitle(screen)
	case screenStation:
		a.drawStation(screen)
	case screenCharacter:
		a.drawCharacter(screen)
	case screenSettings:
		a.drawSettings(screen)
	}
}

func (a *App) drawMenuBackground(screen *ebiten.Image) {
	for i := 0; i < 20; i++ {
		x := float32((i*97 + int(a.seq)%97) % screenW)
		y := float32((i*53 + int(a.seq/2)%53) % screenH)
		sides := 3 + i%5
		drawPolygon(screen, game.V(float64(x), float64(y)), float64(10+i%18), sides, float64(a.seq)/80+float64(i), color.RGBA{28, 42, 58, 135})
	}
	vector.DrawFilledRect(screen, 0, 0, screenW, 84, color.RGBA{10, 16, 24, 245}, false)
	vector.DrawFilledRect(screen, 0, screenH-72, screenW, 72, color.RGBA{10, 16, 24, 235}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("PACKOV    Net %s", a.status), 24, 24)
}

func (a *App) drawTitle(screen *ebiten.Image) {
	drawLargeText(screen, "PACKOV", 92, 120)
	ebitenutil.DebugPrintAt(screen, "Extraction shooter command deck", 98, 196)
	drawMenuList(screen, titleItems, a.menu.Index, 96, 286)
	a.drawShipPreview(screen, game.V(860, 360), 1.35)
}

func (a *App) drawStation(screen *ebiten.Image) {
	drawLargeText(screen, "STATION", 70, 108)
	drawMenuList(screen, stationItems, a.menu.Index, 76, 220)
	lines := []string{
		"Loadout: Machine Gun / Dash / Scout Hull",
		fmt.Sprintf("Credits: %d", accountCredits(a.account)),
		"Daily: Extract once, recover resources, damage a boss",
		"Planet: Verdant-9 / Forest / Threat 1",
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 520, 230)
	a.drawShipPreview(screen, game.V(940, 430), 1.0)
}

func (a *App) drawCharacter(screen *ebiten.Image) {
	drawLargeText(screen, "CHARACTER", 64, 96)
	rows := make([]string, len(characterRows))
	values := []string{a.look.Primary, a.look.Secondary, a.look.TrailID, a.look.NoseID, a.look.DroneSkin, a.look.BadgeID}
	for i := range rows {
		rows[i] = fmt.Sprintf("%-11s %s", characterRows[i], values[i])
	}
	drawMenuList(screen, rows, a.menu.EditIndex, 72, 210)
	ebitenutil.DebugPrintAt(screen, "A/D or arrows edit    Esc returns", 72, 510)
	a.drawShipPreview(screen, game.V(850, 360), 1.7)
}

func (a *App) drawSettings(screen *ebiten.Image) {
	drawLargeText(screen, "SETTINGS", 72, 108)
	drawMenuList(screen, settingsRows, a.menu.Index, 82, 230)
	ebitenutil.DebugPrintAt(screen, "Settings are Go/Ebitengine UI state and will persist through account settings.", 520, 250)
}

func (a *App) drawShipPreview(screen *ebiten.Image, center game.Vec2, scale float64) {
	rot := float64(a.seq)/45 + a.look.SpinOffset
	primary := appearanceColor(a.look.Primary)
	secondary := appearanceColor(a.look.Secondary)
	vector.DrawFilledCircle(screen, float32(center.X), float32(center.Y), float32(32*scale), primary, false)
	noseLen := 54.0 * scale
	if a.look.NoseID == "needle" {
		noseLen = 68 * scale
	}
	nose := []game.Vec2{
		center.Add(game.FromAngle(rot).Mul(noseLen)),
		center.Add(game.FromAngle(rot + 2.55).Mul(25 * scale)),
		center.Add(game.FromAngle(rot - 2.55).Mul(25 * scale)),
	}
	if a.look.NoseID == "split" {
		nose[1] = center.Add(game.FromAngle(rot + 2.2).Mul(30 * scale))
		nose[2] = center.Add(game.FromAngle(rot - 2.2).Mul(30 * scale))
	}
	fillTriangle(screen, nose, secondary)
	for i := 0; i < 3; i++ {
		t := center.Sub(game.FromAngle(rot).Mul(float64(48+i*18) * scale))
		vector.DrawFilledCircle(screen, float32(t.X), float32(t.Y), float32((12-float64(i)*2)*scale), trailColor(a.look.TrailID, uint8(120-i*28)), false)
	}
	for i := 0; i < 2; i++ {
		ang := rot + math.Pi + (float64(i)*2-1)*0.85
		p := center.Add(game.FromAngle(ang).Mul(52 * scale))
		drawPolygon(screen, p, 11*scale, 4, -rot, secondary)
	}
	ebitenutil.DebugPrintAt(screen, strings.ToUpper(a.look.Callsign), int(center.X-44), int(center.Y+86*scale))
}

func drawMenuList(screen *ebiten.Image, items []string, selected, x, y int) {
	for i, item := range items {
		yy := y + i*34
		clr := color.RGBA{34, 48, 64, 230}
		if i == selected {
			clr = color.RGBA{36, 98, 128, 245}
		}
		vector.DrawFilledRect(screen, float32(x), float32(yy-6), 330, 28, clr, false)
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		ebitenutil.DebugPrintAt(screen, prefix+item, x+12, yy)
	}
}

func drawLargeText(screen *ebiten.Image, text string, x, y int) {
	ebitenutil.DebugPrintAt(screen, text, x, y)
	vector.StrokeLine(screen, float32(x), float32(y+28), float32(x+320), float32(y+28), 2, color.RGBA{80, 212, 255, 180}, false)
}

func accountCredits(a *game.Account) int {
	if a == nil {
		return 500
	}
	return a.Credits
}

func appearanceColor(name string) color.RGBA {
	switch name {
	case "amber":
		return color.RGBA{244, 178, 65, 255}
	case "green":
		return color.RGBA{62, 214, 139, 255}
	case "violet":
		return color.RGBA{178, 104, 255, 255}
	case "red":
		return color.RGBA{255, 91, 94, 255}
	case "white":
		return color.RGBA{232, 248, 255, 255}
	default:
		return color.RGBA{47, 178, 255, 255}
	}
}

func trailColor(name string, alpha uint8) color.RGBA {
	switch name {
	case "spark":
		return color.RGBA{255, 214, 94, alpha}
	case "comet":
		return color.RGBA{123, 241, 177, alpha}
	case "pulse":
		return color.RGBA{184, 92, 255, alpha}
	default:
		return color.RGBA{58, 202, 255, alpha}
	}
}
