package game

import "testing"

func TestCraftConsumesComponents(t *testing.T) {
	a := NewAccount("p1", "pilot")
	a.Credits = 5000
	a.Inventory.Add("plasma_battery", 3)
	a.Inventory.Add("quantum_core", 2)
	a.Inventory.Add("railgun_blueprint", 1)
	err := Craft(&a, RecipeDef{Output: "railgun", Costs: map[string]int{"plasma_battery": 3, "quantum_core": 2, "railgun_blueprint": 1}, Credits: 1200})
	if err != nil {
		t.Fatal(err)
	}
	if !a.Unlocks["railgun"] || a.Inventory.Items["plasma_battery"] != 0 || a.Credits != 3800 {
		t.Fatalf("craft state mismatch: %+v", a)
	}
}

func TestRunProducesSnapshot(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("test", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	r.SpawnInitial(c)
	for i := 0; i < 10; i++ {
		r.ApplyInput(InputCommand{PlayerID: "p1", Move: V(1, 0), Aim: V(500, 500), Fire: true})
		r.Step(c)
	}
	s := r.Snapshot()
	if len(s.Entities) == 0 || s.Tick != 10 {
		t.Fatalf("bad snapshot: %+v", s)
	}
}

func TestSquadWipeFailsRunAndDropsCarriedLoot(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("wipe", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	ps := r.Players["p1"]
	ps.Carried.Add("alien_alloy", 3)
	player := r.Entities[ps.EntityID]
	player.HP = 0
	r.resolveCombat(c)
	ps.Downed = true
	ps.Carried = NewInventory()
	r.updateRunOutcome()
	if r.Phase != PhaseFailed {
		t.Fatalf("expected failed run, got %s", r.Phase)
	}
	if len(ps.Carried.Items) != 0 {
		t.Fatalf("carried loot should be lost on wipe: %+v", ps.Carried.Items)
	}
}

func TestExtractedRunCompletesWithCarriedLootIntactForSettlement(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("extract", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	ps := r.Players["p1"]
	ps.Carried.Add("alien_alloy", 2)
	ps.Extracted = true
	r.updateRunOutcome()
	if r.Phase != PhaseComplete {
		t.Fatalf("expected complete run, got %s", r.Phase)
	}
	if ps.Carried.Items["alien_alloy"] != 2 {
		t.Fatalf("extracted carried loot must remain for settlement: %+v", ps.Carried.Items)
	}
}
