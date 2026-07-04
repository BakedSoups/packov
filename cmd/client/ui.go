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
	screenLoadout
	screenInventory
	screenCrafting
	screenMarketplace
	screenSettings
	screenResult
	screenRun
)

type menuState struct {
	Index       int
	EditIndex   int
	SettingsIdx int
	LoadoutIdx  int
	CraftIdx    int
	MarketIdx   int
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
			case "Loadout":
				a.screen = screenLoadout
			case "Inventory":
				a.screen = screenInventory
			case "Crafting":
				a.screen = screenCrafting
			case "Marketplace":
				a.screen = screenMarketplace
				a.requestMarket()
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
	case screenLoadout:
		a.updateLoadout()
	case screenInventory:
		if a.justPressed(ebiten.KeyEscape) || a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
			a.screen = screenStation
		}
	case screenMarketplace:
		a.updateMarketplace()
	case screenCrafting:
		a.updateCrafting()
	case screenSettings:
		a.updateList(len(settingsRows), func() {
			if settingsRows[a.menu.Index] == "Back" {
				a.screen = screenStation
				return
			}
			a.toggleSetting(a.menu.Index)
		})
	case screenResult:
		if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) || a.justPressed(ebiten.KeyEscape) {
			a.remote = false
			a.queued = false
			a.screen = screenStation
		}
	}
}

func (a *App) toggleSetting(index int) {
	switch index {
	case 0:
		a.settings.MouseAim = !a.settings.MouseAim
	case 1:
		a.settings.Controller = !a.settings.Controller
	case 2:
		a.settings.DamageNumbers = !a.settings.DamageNumbers
	case 3:
		a.settings.ScreenShake = !a.settings.ScreenShake
	}
}

func (a *App) updateMarketplace() {
	if a.justPressed(ebiten.KeyEscape) {
		a.screen = screenStation
		return
	}
	if a.justPressed(ebiten.KeyArrowUp) || a.justPressed(ebiten.KeyW) {
		count := max(1, len(a.listings))
		a.menu.MarketIdx = (a.menu.MarketIdx + count - 1) % count
	}
	if a.justPressed(ebiten.KeyArrowDown) || a.justPressed(ebiten.KeyS) {
		count := max(1, len(a.listings))
		a.menu.MarketIdx = (a.menu.MarketIdx + 1) % count
	}
	if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
		a.buyOrCancelSelected()
	}
	if a.justPressed(ebiten.KeyE) {
		a.sellFirstInventoryItem()
	}
}

func (a *App) requestMarket() {
	if a.net != nil && a.net.isOpen() && a.hello {
		a.net.send(protocol.ClientMessage{Type: "market_list"})
	}
}

func (a *App) buyOrCancelSelected() {
	if len(a.listings) == 0 || a.menu.MarketIdx >= len(a.listings) {
		return
	}
	listing := a.listings[a.menu.MarketIdx]
	if a.net == nil || !a.net.isOpen() || !a.hello {
		a.status = "market requires server"
		return
	}
	msgType := "market_buy"
	if listing.SellerID == a.player {
		msgType = "market_cancel"
	}
	a.net.send(protocol.ClientMessage{Type: msgType, ListingID: listing.ID})
}

func (a *App) sellFirstInventoryItem() {
	if a.account == nil {
		return
	}
	for item, count := range a.account.Inventory.Items {
		if count <= 0 {
			continue
		}
		price := 25
		if loot, ok := a.catalog.LootByID[item]; ok {
			price = loot.BaseValue
		}
		if a.net == nil || !a.net.isOpen() || !a.hello {
			a.status = "market requires server"
			return
		}
		a.net.send(protocol.ClientMessage{Type: "market_sell", ItemID: item, Quantity: 1, UnitPrice: price})
		return
	}
	a.status = "no inventory item to sell"
}

func (a *App) updateCrafting() {
	if a.justPressed(ebiten.KeyEscape) {
		a.screen = screenStation
		return
	}
	if len(a.catalog.Recipes) == 0 {
		return
	}
	if a.justPressed(ebiten.KeyArrowUp) || a.justPressed(ebiten.KeyW) {
		a.menu.CraftIdx = (a.menu.CraftIdx + len(a.catalog.Recipes) - 1) % len(a.catalog.Recipes)
	}
	if a.justPressed(ebiten.KeyArrowDown) || a.justPressed(ebiten.KeyS) {
		a.menu.CraftIdx = (a.menu.CraftIdx + 1) % len(a.catalog.Recipes)
	}
	if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
		a.craftSelected()
	}
}

func (a *App) craftSelected() {
	if len(a.catalog.Recipes) == 0 || a.menu.CraftIdx >= len(a.catalog.Recipes) {
		return
	}
	recipe := a.catalog.Recipes[a.menu.CraftIdx]
	if a.net != nil && a.net.isOpen() && a.hello {
		a.net.send(protocol.ClientMessage{Type: "craft", RecipeID: recipe.ID})
		a.status = "crafting " + recipe.Output
		return
	}
	if a.account == nil {
		return
	}
	if err := game.Craft(a.account, recipe); err != nil {
		a.status = "craft failed: " + err.Error()
		return
	}
	a.status = "crafted " + recipe.Output
}

func (a *App) updateLoadout() {
	if a.justPressed(ebiten.KeyEscape) {
		a.screen = screenStation
		return
	}
	if a.justPressed(ebiten.KeyArrowUp) || a.justPressed(ebiten.KeyW) {
		a.menu.LoadoutIdx = (a.menu.LoadoutIdx + 2) % 3
	}
	if a.justPressed(ebiten.KeyArrowDown) || a.justPressed(ebiten.KeyS) {
		a.menu.LoadoutIdx = (a.menu.LoadoutIdx + 1) % 3
	}
	if a.justPressed(ebiten.KeyArrowLeft) || a.justPressed(ebiten.KeyA) {
		a.cycleLoadout(-1)
	}
	if a.justPressed(ebiten.KeyArrowRight) || a.justPressed(ebiten.KeyD) || a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
		a.cycleLoadout(1)
	}
}

func (a *App) cycleLoadout(delta int) {
	switch a.menu.LoadoutIdx {
	case 0:
		a.loadout.WeaponID = cycleString(a.weaponIDs(), a.loadout.WeaponID, delta)
	case 1:
		a.loadout.AbilityID = cycleString(a.abilityIDs(), a.loadout.AbilityID, delta)
	case 2:
		a.loadout.HullID = cycleString([]string{"hull_scout"}, a.loadout.HullID, delta)
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
		a.net.send(protocol.ClientMessage{Type: "queue", PlanetID: "verdant", Loadout: a.loadout})
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
	a.localSettled = false
	a.run = game.NewRun("local-solo", a.catalog, "verdant", timeSeed())
	a.run.AddPlayer(a.player, a.look.Callsign, a.loadout)
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
	case screenLoadout:
		a.drawLoadout(screen)
	case screenInventory:
		a.drawInventory(screen)
	case screenCrafting:
		a.drawCrafting(screen)
	case screenMarketplace:
		a.drawMarketplace(screen)
	case screenSettings:
		a.drawSettings(screen)
	case screenResult:
		a.drawRunResult(screen)
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
		fmt.Sprintf("Loadout: %s / %s / %s", a.weaponName(a.loadout.WeaponID), a.abilityName(a.loadout.AbilityID), a.loadout.HullID),
		fmt.Sprintf("Credits: %d", accountCredits(a.account)),
		"Daily: Extract once, recover resources, damage a boss",
		"Planet: Verdant-9 / Forest / Threat 1",
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 520, 230)
	a.drawShipPreview(screen, game.V(940, 430), 1.0)
}

func (a *App) drawLoadout(screen *ebiten.Image) {
	drawLargeText(screen, "LOADOUT", 72, 108)
	rows := []string{
		"Weapon  " + a.weaponName(a.loadout.WeaponID),
		"Ability " + a.abilityName(a.loadout.AbilityID),
		"Hull    " + a.loadout.HullID,
	}
	drawMenuList(screen, rows, a.menu.LoadoutIdx, 82, 230)
	ebitenutil.DebugPrintAt(screen, "A/D or arrows change    Esc returns", 82, 390)
	a.drawShipPreview(screen, game.V(880, 380), 1.2)
}

func (a *App) drawInventory(screen *ebiten.Image) {
	drawLargeText(screen, "INVENTORY", 72, 108)
	lines := []string{fmt.Sprintf("Credits: %d", accountCredits(a.account))}
	if a.account == nil || len(a.account.Inventory.Items) == 0 {
		lines = append(lines, "Station inventory empty")
	} else {
		for item, count := range a.account.Inventory.Items {
			lines = append(lines, fmt.Sprintf("%s x%d", item, count))
		}
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 82, 220)
	ebitenutil.DebugPrintAt(screen, "Enter returns", 82, 560)
}

func (a *App) drawCrafting(screen *ebiten.Image) {
	drawLargeText(screen, "CRAFTING", 72, 108)
	lines := []string{}
	for i, recipe := range a.catalog.Recipes {
		prefix := "  "
		if i == a.menu.CraftIdx {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s -> %s   %d credits", prefix, recipe.ID, recipe.Output, recipe.Credits))
		for item, count := range recipe.Costs {
			lines = append(lines, fmt.Sprintf("  %s x%d", item, count))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, "No recipes loaded")
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 82, 220)
	ebitenutil.DebugPrintAt(screen, "Enter crafts    Esc returns", 82, 560)
}

func (a *App) drawMarketplace(screen *ebiten.Image) {
	drawLargeText(screen, "MARKET", 72, 108)
	lines := []string{
		"Enter buys selected listing or cancels your listing. E lists one inventory item.",
		"",
	}
	if len(a.listings) == 0 {
		lines = append(lines, "No active listings")
	}
	for i, listing := range a.listings {
		prefix := "  "
		if i == a.menu.MarketIdx {
			prefix = "> "
		}
		owner := ""
		if listing.SellerID == a.player {
			owner = " yours"
		}
		lines = append(lines, fmt.Sprintf("%s%s x%d @ %d%s", prefix, listing.ItemID, listing.Quantity, listing.UnitPrice, owner))
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 82, 220)
	ebitenutil.DebugPrintAt(screen, "Enter returns", 82, 560)
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
	rows := []string{
		"Mouse Aim      " + onOff(a.settings.MouseAim),
		"Controller     " + onOff(a.settings.Controller),
		"Damage Numbers " + onOff(a.settings.DamageNumbers),
		"Screen Shake   " + onOff(a.settings.ScreenShake),
		"Back",
	}
	drawMenuList(screen, rows, a.menu.Index, 82, 230)
	ebitenutil.DebugPrintAt(screen, "Enter toggles selected setting", 520, 250)
}

func (a *App) drawRunResult(screen *ebiten.Image) {
	title := "MISSION RESULT"
	if a.run != nil && a.run.Phase == game.PhaseFailed {
		title = "MISSION FAILED"
	}
	drawLargeText(screen, title, 72, 108)
	lines := []string{}
	if a.run == nil {
		lines = append(lines, "No run data available")
	} else {
		lines = append(lines, "Planet: "+a.run.Planet.Name)
		lines = append(lines, "Phase: "+strings.ToUpper(string(a.run.Phase)))
		if ps := a.run.Players[a.player]; ps != nil {
			if ps.Extracted {
				lines = append(lines, "Extraction: successful")
				lines = append(lines, "Carried loot was transferred to station inventory.")
			}
			if ps.Downed || a.run.Phase == game.PhaseFailed {
				lines = append(lines, "Extraction: failed")
				lines = append(lines, "Carried loot was lost. Unlocks, cosmetics, and account progress remain.")
			}
			lines = append(lines, fmt.Sprintf("Carried at result: %v", ps.Carried.Items))
		}
		if len(a.run.Messages) > 0 {
			lines = append(lines, "")
			lines = append(lines, a.run.Messages[len(a.run.Messages)-1])
		}
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 82, 210)
	ebitenutil.DebugPrintAt(screen, "Enter returns to station", 82, 560)
}

func (a *App) drawShipPreview(screen *ebiten.Image, center game.Vec2, scale float64) {
	rot := float64(a.seq)/45 + a.look.SpinOffset
	primary := appearanceColor(a.look.Primary)
	secondary := appearanceColor(a.look.Secondary)
	drawOutlinedCircle(screen, center, 32*scale, primary, float32(5*scale))
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
		drawOutlinedCircle(screen, t, (12-float64(i)*2)*scale, trailColor(a.look.TrailID, uint8(145-i*30)), float32(2*scale))
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

func onOff(v bool) string {
	if v {
		return "ON"
	}
	return "OFF"
}

func (a *App) weaponIDs() []string {
	ids := make([]string, 0, len(a.catalog.Weapons))
	for _, weapon := range a.catalog.Weapons {
		if a.account == nil || a.account.Unlocks[weapon.ID] {
			ids = append(ids, weapon.ID)
		}
	}
	return ids
}

func (a *App) abilityIDs() []string {
	ids := make([]string, 0, len(a.catalog.Abilities))
	for _, ability := range a.catalog.Abilities {
		if a.account == nil || a.account.Unlocks[ability.ID] {
			ids = append(ids, ability.ID)
		}
	}
	return ids
}

func (a *App) weaponName(id string) string {
	if def, ok := a.catalog.WeaponByID[id]; ok {
		return def.Name
	}
	return id
}

func (a *App) abilityName(id string) string {
	if def, ok := a.catalog.AbilityByID[id]; ok {
		return def.Name
	}
	return id
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
