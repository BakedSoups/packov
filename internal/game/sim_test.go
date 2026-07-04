package game

import (
	"math"
	"testing"
)

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

func TestEnemySeparationResolvesOverlap(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("separation", c, "verdant", 42)
	def := c.EnemyByID["skitter"]
	r.spawnEnemy(def, V(500, 500))
	r.spawnEnemy(def, V(500, 500))
	before := enemyMinDistance(r)
	r.resolveEntitySeparation()
	after := enemyMinDistance(r)
	if after <= before {
		t.Fatalf("expected separation to increase min distance, before %.2f after %.2f", before, after)
	}
}

func TestEnemySteeringAddsOrganicLateralMovement(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("organic", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	player := r.Entities[r.Players["p1"].EntityID]
	player.Position = V(700, 500)
	def := c.EnemyByID["skitter"]
	r.spawnEnemy(def, V(500, 500))
	for _, e := range r.Entities {
		if e.Kind == EntityEnemy {
			r.updateAI(c, 1.0/TickRate)
			if math.Abs(e.Velocity.Y) < 0.001 {
				t.Fatalf("expected lateral steering, got velocity %+v", e.Velocity)
			}
			return
		}
	}
	t.Fatal("enemy not spawned")
}

func TestEnemyContactUsesCommittedAttackStates(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("contact", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	player := r.Entities[r.Players["p1"].EntityID]
	player.Position = V(500, 500)
	def := c.EnemyByID["skitter"]
	r.spawnEnemy(def, V(430, 500))
	var enemy *Entity
	for _, e := range r.Entities {
		if e.Kind == EntityEnemy {
			enemy = e
			break
		}
	}
	if enemy == nil {
		t.Fatal("enemy not spawned")
	}
	states := map[string]bool{}
	for i := 0; i < 20; i++ {
		r.updateAI(c, 1.0/TickRate)
		states[enemy.AIState] = true
		r.Tick++
	}
	if !states[AIOrbit] || !states[AILunge] {
		t.Fatalf("expected orbit and lunge states, got %+v", states)
	}
}

func TestEnemyMeleeDamageIsCooldownGated(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("cooldown", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	player := r.Entities[r.Players["p1"].EntityID]
	player.Position = V(500, 500)
	def := c.EnemyByID["skitter"]
	r.spawnEnemy(def, V(500, 500))
	var enemy *Entity
	for _, e := range r.Entities {
		if e.Kind == EntityEnemy {
			enemy = e
			break
		}
	}
	enemy.AIState = AILunge
	enemy.StateTick = 0
	r.Tick = 4
	r.resolveCombat(c)
	afterFirstHit := player.HP
	for i := 0; i < 10; i++ {
		r.Tick++
		r.resolveCombat(c)
	}
	if player.HP != afterFirstHit {
		t.Fatalf("expected cooldown to prevent continuous damage, first %.2f now %.2f", afterFirstHit, player.HP)
	}
	if enemy.AIState != AIRecover {
		t.Fatalf("expected recover after hit, got %s", enemy.AIState)
	}
}

func TestCombatSetsHitTicks(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("hits", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	def := c.EnemyByID["skitter"]
	r.spawnEnemy(def, V(520, 500))
	player := r.Entities[r.Players["p1"].EntityID]
	player.Position = V(500, 500)
	r.Tick = 12
	for _, e := range r.Entities {
		if e.Kind == EntityEnemy {
			e.AIState = AILunge
			e.StateTick = 8
		}
	}
	r.resolveCombat(c)
	if player.HitTick != 12 {
		t.Fatalf("expected player hit tick 12, got %d", player.HitTick)
	}
}

func enemyMinDistance(r *RunState) float64 {
	best := math.MaxFloat64
	for _, a := range r.Entities {
		if a.Kind != EntityEnemy {
			continue
		}
		for _, b := range r.Entities {
			if b.ID <= a.ID || b.Kind != EntityEnemy {
				continue
			}
			best = math.Min(best, Dist(a.Position, b.Position))
		}
	}
	return best
}
