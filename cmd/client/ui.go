package main

import (
	"fmt"
	"image/color"
	"math"
	"sort"
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
	screenPlanet
	screenLoadout
	screenInventory
	screenCrafting
	screenMarketplace
	screenProfile
	screenSettings
	screenResult
	screenRun
)

type menuState struct {
	Index        int
	EditIndex    int
	SettingsIdx  int
	LoadoutIdx   int
	PlanetIdx    int
	CraftIdx     int
	MarketIdx    int
	CraftConfirm bool
}

var (
	titleItems    = []string{"Enter Station", "Edit Character", "Settings", "Local Test Run"}
	stationItems  = []string{"Deploy", "Loadout", "Inventory", "Crafting", "Marketplace", "Profile", "Character", "Settings", "Title"}
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
			case "Deploy":
				a.screen = screenPlanet
			case "Loadout":
				a.screen = screenLoadout
			case "Inventory":
				a.screen = screenInventory
			case "Crafting":
				a.screen = screenCrafting
			case "Marketplace":
				a.screen = screenMarketplace
				a.requestMarket()
			case "Profile":
				a.screen = screenProfile
			case "Character":
				a.screen = screenCharacter
			case "Settings":
				a.screen = screenSettings
			case "Title":
				a.screen = screenTitle
			}
		})
	case screenPlanet:
		a.updatePlanetSelect()
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
	case screenProfile:
		if a.justPressed(ebiten.KeyEscape) || a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
			a.screen = screenStation
		}
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
		if a.menu.CraftConfirm {
			a.menu.CraftConfirm = false
			return
		}
		a.screen = screenStation
		return
	}
	if len(a.catalog.Recipes) == 0 {
		return
	}
	if a.justPressed(ebiten.KeyArrowUp) || a.justPressed(ebiten.KeyW) {
		a.menu.CraftIdx = (a.menu.CraftIdx + len(a.catalog.Recipes) - 1) % len(a.catalog.Recipes)
		a.menu.CraftConfirm = false
	}
	if a.justPressed(ebiten.KeyArrowDown) || a.justPressed(ebiten.KeyS) {
		a.menu.CraftIdx = (a.menu.CraftIdx + 1) % len(a.catalog.Recipes)
		a.menu.CraftConfirm = false
	}
	if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
		if !a.menu.CraftConfirm {
			a.menu.CraftConfirm = true
			a.notice("Confirm craft: " + a.catalog.Recipes[a.menu.CraftIdx].Output)
			return
		}
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
		a.notice("Crafting " + recipe.Output)
		a.menu.CraftConfirm = false
		return
	}
	if a.account == nil {
		return
	}
	if err := game.Craft(a.account, recipe); err != nil {
		a.notice("Craft failed: " + err.Error())
		a.menu.CraftConfirm = false
		return
	}
	a.notice("Crafted " + recipe.Output)
	a.menu.CraftConfirm = false
}

func (a *App) notice(text string) {
	a.uiNotice = text
	a.uiNoticeTick = a.seq
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
	planetID := a.selectedPlanetID()
	if a.net != nil && a.net.isOpen() {
		a.queued = true
		a.status = "queueing " + planetID
		a.net.send(protocol.ClientMessage{Type: "queue", PlanetID: planetID, Loadout: a.loadout})
		return
	}
	a.startLocalRun(planetID)
}

func (a *App) selectedPlanetID() string {
	if len(a.catalog.Planets) == 0 {
		return "verdant"
	}
	if a.menu.PlanetIdx < 0 || a.menu.PlanetIdx >= len(a.catalog.Planets) {
		a.menu.PlanetIdx = 0
	}
	return a.catalog.Planets[a.menu.PlanetIdx].ID
}

func (a *App) syncAppearance() {
	if a.account != nil {
		a.account.Appearance = a.look
	}
	if a.net != nil && a.net.isOpen() && a.hello {
		a.net.send(protocol.ClientMessage{Type: "appearance", Appearance: a.look})
	}
}

func (a *App) startLocalRun(planetID ...string) {
	id := a.selectedPlanetID()
	if len(planetID) > 0 && planetID[0] != "" {
		id = planetID[0]
	}
	a.remote = false
	a.queued = false
	a.localSettled = false
	a.run = game.NewRun("local-solo", a.catalog, id, timeSeed())
	a.run.AddPlayer(a.player, a.look.Callsign, a.loadout)
	a.run.SpawnInitial(a.catalog)
	a.screen = screenRun
	a.status = "local fallback"
}

func (a *App) updatePlanetSelect() {
	if a.justPressed(ebiten.KeyEscape) {
		a.screen = screenStation
		return
	}
	if len(a.catalog.Planets) == 0 {
		return
	}
	if a.justPressed(ebiten.KeyArrowUp) || a.justPressed(ebiten.KeyW) {
		a.menu.PlanetIdx = (a.menu.PlanetIdx + len(a.catalog.Planets) - 1) % len(a.catalog.Planets)
	}
	if a.justPressed(ebiten.KeyArrowDown) || a.justPressed(ebiten.KeyS) {
		a.menu.PlanetIdx = (a.menu.PlanetIdx + 1) % len(a.catalog.Planets)
	}
	if a.justPressed(ebiten.KeyEnter) || a.justPressed(ebiten.KeySpace) {
		a.deploy()
	}
}

func (a *App) drawMenu(screen *ebiten.Image) {
	a.drawMenuBackground(screen)
	switch a.screen {
	case screenTitle:
		a.drawTitle(screen)
	case screenStation:
		a.drawStation(screen)
	case screenPlanet:
		a.drawPlanetSelect(screen)
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
	case screenProfile:
		a.drawProfile(screen)
	case screenSettings:
		a.drawSettings(screen)
	case screenResult:
		a.drawRunResult(screen)
	}
}

func (a *App) drawMenuBackground(screen *ebiten.Image) {
	for i := 0; i < 12; i++ {
		x := float32((i*97 + int(a.seq)%97) % screenW)
		y := float32((i*53 + int(a.seq/2)%53) % screenH)
		sides := 3 + i%5
		drawPolygon(screen, game.V(float64(x), float64(y)), float64(10+i%18), sides, float64(a.seq)/80+float64(i), color.RGBA{28, 42, 58, 82})
	}
	DrawStationBackdrop(screen, game.V(905, 350), a.seq)
	vector.DrawFilledRect(screen, 0, 0, screenW, 84, color.RGBA{10, 16, 24, 245}, false)
	vector.DrawFilledRect(screen, 0, screenH-72, screenW, 72, color.RGBA{10, 16, 24, 235}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("PACKOV    Net %s", a.status), 24, 24)
	if a.uiNotice != "" && a.seq-a.uiNoticeTick < 180 {
		ebitenutil.DebugPrintAt(screen, a.uiNotice, 24, screenH-46)
	}
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
	planet := a.selectedPlanet()
	lines := []string{
		fmt.Sprintf("Loadout: %s / %s / %s", a.weaponName(a.loadout.WeaponID), a.abilityName(a.loadout.AbilityID), a.loadout.HullID),
		fmt.Sprintf("Credits: %d", accountCredits(a.account)),
		"Daily: Extract once, mine 3 resources, damage a boss",
		"Weekly: Complete objectives on 3 planet types",
		fmt.Sprintf("Selected: %s / %s / Threat %d", planet.Name, planet.Type, planet.Threat),
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 520, 230)
	a.drawShipPreview(screen, game.V(940, 430), 1.0)
}

func (a *App) drawPlanetSelect(screen *ebiten.Image) {
	drawLargeText(screen, "PLANETS", 72, 108)
	rows := make([]string, 0, len(a.catalog.Planets))
	for _, planet := range a.catalog.Planets {
		rows = append(rows, fmt.Sprintf("%-16s Threat %d", planet.Name, planet.Threat))
	}
	if len(rows) == 0 {
		rows = append(rows, "No planets loaded")
	}
	drawMenuList(screen, rows, a.menu.PlanetIdx, 76, 210)
	planet := a.selectedPlanet()
	boss := a.catalog.BossByID[planet.Boss]
	lines := []string{
		fmt.Sprintf("Biome: %s", planet.Type),
		fmt.Sprintf("Threat: %d", planet.Threat),
		"Resources: " + strings.Join(planet.Resources, ", "),
		"Boss: " + valueOr(boss.Name, planet.Boss),
		"Hazards: " + strings.Join(planet.Hazards, ", "),
		"Event: rotating world event modifiers apply when active",
		"",
		"Daily missions",
		"- Extract from any planet",
		"- Mine resources before extraction",
		"- Damage a boss or elite",
		"",
		"Weekly",
		"- Clear objectives across multiple biomes",
	}
	vector.DrawFilledRect(screen, 520, 190, 610, 330, color.RGBA{18, 28, 38, 225}, false)
	vector.StrokeRect(screen, 520, 190, 610, 330, 4, outlineColor(), false)
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 548, 218)
	ebitenutil.DebugPrintAt(screen, "Enter deploys    Esc returns", 82, 560)
}

func (a *App) selectedPlanet() game.PlanetDef {
	if len(a.catalog.Planets) == 0 {
		return game.PlanetDef{ID: "verdant", Name: "Verdant-9", Type: "forest", Threat: 1, Boss: "hive_queen"}
	}
	if a.menu.PlanetIdx < 0 || a.menu.PlanetIdx >= len(a.catalog.Planets) {
		a.menu.PlanetIdx = 0
	}
	return a.catalog.Planets[a.menu.PlanetIdx]
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
	start := visibleRecipeStart(a.menu.CraftIdx, len(a.catalog.Recipes), 10)
	end := start + 10
	if end > len(a.catalog.Recipes) {
		end = len(a.catalog.Recipes)
	}
	if start > 0 {
		lines = append(lines, "  ...")
	}
	for i := start; i < end; i++ {
		recipe := a.catalog.Recipes[i]
		prefix := "  "
		if i == a.menu.CraftIdx {
			prefix = "> "
		}
		state := "READY"
		if !a.canCraft(recipe) {
			state = "MISSING"
		}
		lines = append(lines, fmt.Sprintf("%s%-13s %-18s %s", prefix, strings.ToUpper(recipe.Category), recipeName(recipe), state))
	}
	if end < len(a.catalog.Recipes) {
		lines = append(lines, "  ...")
	}
	if len(lines) == 0 {
		lines = append(lines, "No recipes loaded")
	}
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 82, 220)
	a.drawCraftingBranch(screen)
	a.drawRecipeDetail(screen)
	footer := "Enter selects    Esc returns"
	if a.menu.CraftConfirm {
		footer = "Enter confirms craft    Esc cancels"
	}
	ebitenutil.DebugPrintAt(screen, footer, 82, 560)
}

func (a *App) drawRecipeDetail(screen *ebiten.Image) {
	if len(a.catalog.Recipes) == 0 || a.menu.CraftIdx >= len(a.catalog.Recipes) {
		return
	}
	recipe := a.catalog.Recipes[a.menu.CraftIdx]
	x := float32(570)
	y := float32(205)
	vector.DrawFilledRect(screen, x-18, y-18, 560, 360, color.RGBA{18, 28, 38, 230}, false)
	vector.StrokeRect(screen, x-18, y-18, 560, 360, 4, outlineColor(), false)
	ebitenutil.DebugPrintAt(screen, strings.ToUpper(recipe.Category)+"  "+recipeName(recipe), int(x), int(y))
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("ROLE    %s", valueOr(recipe.Role, "sidegrade")), int(x), int(y+28))
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("CREDITS %d / %d", accountCredits(a.account), recipe.Credits), int(x), int(y+56))
	ready := a.canCraft(recipe)
	readyText := "READY"
	if !ready {
		readyText = "MISSING COMPONENTS"
	}
	ebitenutil.DebugPrintAt(screen, readyText, int(x), int(y+84))
	rowY := y + 116
	for _, item := range sortedCostKeys(recipe.Costs) {
		need := recipe.Costs[item]
		have := a.inventoryCount(item)
		clr := color.RGBA{80, 205, 104, 255}
		if have < need {
			clr = color.RGBA{255, 91, 94, 255}
		}
		vector.DrawFilledRect(screen, x, rowY+2, 16, 16, clr, false)
		vector.StrokeRect(screen, x, rowY+2, 16, 16, 2, outlineColor(), false)
		row := fmt.Sprintf("%s  %d / %d", a.lootName(item), have, need)
		if have < need {
			row += "  Find: " + compactText(a.lootSource(item), 25)
		}
		ebitenutil.DebugPrintAt(screen, row, int(x+26), int(rowY))
		rowY += 28
	}
	ebitenutil.DebugPrintAt(screen, "SOURCE  "+valueOr(recipe.Source, "gameplay drops"), int(x), int(y+250))
	if recipe.Blueprint != "" {
		ebitenutil.DebugPrintAt(screen, "PRINT   "+a.lootName(recipe.Blueprint)+" / "+valueOr(recipe.BlueprintSource, "drop"), int(x), int(y+275))
	}
	ebitenutil.DebugPrintAt(screen, "RULE    "+valueOr(recipe.TradeRule, "components tradable, unlock bound"), int(x), int(y+300))
	if a.menu.CraftConfirm {
		vector.DrawFilledRect(screen, x+300, y+312, 220, 30, color.RGBA{247, 205, 92, 240}, false)
		vector.StrokeRect(screen, x+300, y+312, 220, 30, 3, outlineColor(), false)
		ebitenutil.DebugPrintAt(screen, "CONFIRM CRAFT", int(x+316), int(y+320))
	}
}

func visibleRecipeStart(selected, total, visible int) int {
	if total <= visible {
		return 0
	}
	start := selected - visible/2
	if start < 0 {
		return 0
	}
	if start+visible > total {
		return total - visible
	}
	return start
}

func (a *App) drawCraftingBranch(screen *ebiten.Image) {
	if len(a.catalog.Recipes) == 0 || a.menu.CraftIdx >= len(a.catalog.Recipes) {
		return
	}
	selected := a.catalog.Recipes[a.menu.CraftIdx]
	nodes := make([]game.RecipeDef, 0, 6)
	for _, recipe := range a.catalog.Recipes {
		if recipe.Category == selected.Category {
			nodes = append(nodes, recipe)
		}
	}
	if len(nodes) == 0 {
		return
	}
	x := float32(82)
	y := float32(480)
	w := float32(410)
	vector.DrawFilledRect(screen, x-14, y-32, w+28, 74, color.RGBA{16, 25, 34, 210}, false)
	vector.StrokeRect(screen, x-14, y-32, w+28, 74, 4, outlineColor(), false)
	ebitenutil.DebugPrintAt(screen, strings.ToUpper(selected.Category)+" TREE", int(x), int(y-24))
	step := w
	if len(nodes) > 1 {
		step = w / float32(len(nodes)-1)
	}
	for i := range nodes {
		cx := x + float32(i)*step
		if i > 0 {
			px := x + float32(i-1)*step
			vector.StrokeLine(screen, px+18, y, cx-18, y, 5, color.RGBA{45, 62, 76, 255}, false)
		}
	}
	for i, recipe := range nodes {
		cx := x + float32(i)*step
		fill := color.RGBA{255, 91, 94, 255}
		if a.account != nil && a.account.Unlocks[recipe.Output] {
			fill = color.RGBA{103, 228, 155, 255}
		} else if a.canCraft(recipe) {
			fill = color.RGBA{247, 205, 92, 255}
		}
		if recipe.ID == selected.ID {
			vector.DrawFilledCircle(screen, cx, y, 22, color.RGBA{245, 249, 255, 255}, false)
		}
		vector.DrawFilledCircle(screen, cx, y, 16, fill, false)
		vector.StrokeCircle(screen, cx, y, 16, 4, outlineColor(), false)
		ebitenutil.DebugPrintAt(screen, compactText(recipeName(recipe), 12), int(cx-42), int(y+24))
	}
}

func sortedCostKeys(costs map[string]int) []string {
	keys := make([]string, 0, len(costs))
	for item := range costs {
		keys = append(keys, item)
	}
	sort.Strings(keys)
	return keys
}

func recipeName(recipe game.RecipeDef) string {
	if recipe.OutputName != "" {
		return recipe.OutputName
	}
	return recipe.Output
}

func valueOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func (a *App) lootName(item string) string {
	if loot, ok := a.catalog.LootByID[item]; ok {
		return loot.Name
	}
	return item
}

func (a *App) lootSource(item string) string {
	if loot, ok := a.catalog.LootByID[item]; ok && loot.Source != "" {
		return loot.Source
	}
	return "gameplay drops"
}

func compactText(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func (a *App) canCraft(recipe game.RecipeDef) bool {
	if a.account == nil || a.account.Credits < recipe.Credits {
		return false
	}
	return a.account.Inventory.Has(recipe.Costs)
}

func (a *App) inventoryCount(item string) int {
	if a.account == nil {
		return 0
	}
	return a.account.Inventory.Items[item]
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

func (a *App) drawProfile(screen *ebiten.Image) {
	drawLargeText(screen, "PROFILE", 72, 108)
	lines := []string{}
	if a.account == nil {
		lines = append(lines, "No account loaded")
	} else {
		unlocks := make([]string, 0, len(a.account.Unlocks))
		for unlock, owned := range a.account.Unlocks {
			if owned {
				unlocks = append(unlocks, unlock)
			}
		}
		sort.Strings(unlocks)
		if len(unlocks) > 10 {
			unlocks = unlocks[:10]
		}
		lines = append(lines,
			"Callsign: "+valueOr(a.look.Callsign, string(a.account.ID)),
			fmt.Sprintf("Credits: %d", a.account.Credits),
			fmt.Sprintf("Level: %d", a.account.Level),
			fmt.Sprintf("Unlocks owned: %d", len(a.account.Unlocks)),
			fmt.Sprintf("Cosmetics owned: %d", len(a.account.Cosmetics)),
			"Equipped weapon: "+a.weaponName(a.loadout.WeaponID),
			"Equipped ability: "+a.abilityName(a.loadout.AbilityID),
			"Equipped hull: "+a.loadout.HullID,
			"",
			"Recent unlocks",
		)
		if len(unlocks) == 0 {
			lines = append(lines, "none")
		} else {
			lines = append(lines, unlocks...)
		}
	}
	vector.DrawFilledRect(screen, 72, 190, 470, 360, color.RGBA{18, 28, 38, 225}, false)
	vector.StrokeRect(screen, 72, 190, 470, 360, 4, outlineColor(), false)
	ebitenutil.DebugPrintAt(screen, strings.Join(lines, "\n"), 100, 218)
	a.drawShipPreview(screen, game.V(870, 360), 1.45)
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
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("Kills: %d", ps.Stats.Kills))
			lines = append(lines, fmt.Sprintf("Objectives: %d", ps.Stats.ObjectivesCompleted))
			lines = append(lines, fmt.Sprintf("Resources mined: %d", ps.Stats.ResourcesMined))
			lines = append(lines, fmt.Sprintf("Boss damage: %.0f", ps.Stats.BossDamage))
			lines = append(lines, fmt.Sprintf("Loot extracted value: %d cr", ps.Stats.LootExtractedValue))
			lines = append(lines, fmt.Sprintf("Loot lost value: %d cr", ps.Stats.LootLostValue))
			lines = append(lines, fmt.Sprintf("Credits earned: %d", ps.Stats.CreditsEarned))
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
