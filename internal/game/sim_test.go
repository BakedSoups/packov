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
