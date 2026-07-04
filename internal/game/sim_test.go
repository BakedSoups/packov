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

func TestCatalogIncludesCraftingProgressionTree(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	categories := map[string]int{}
	for _, recipe := range c.Recipes {
		categories[recipe.Category]++
		if recipe.Blueprint == "" {
			t.Fatalf("recipe %s should declare blueprint gate", recipe.ID)
		}
		if recipe.Source == "" || recipe.BlueprintSource == "" || recipe.TradeRule == "" || recipe.StatHint == "" {
			t.Fatalf("recipe %s missing progression metadata: %+v", recipe.ID, recipe)
		}
	}
	for _, category := range []string{"weapon", "armor", "hull", "drone", "module", "ability", "cosmetic", "planet_access"} {
		if categories[category] == 0 {
			t.Fatalf("missing recipe category %s", category)
		}
	}
	for _, item := range []string{"hive_egg", "queen_carapace", "rail_actuator", "void_crystal", "phase_membrane", "seismic_tooth"} {
		loot, ok := c.LootByID[item]
		if !ok {
			t.Fatalf("missing boss component %s", item)
		}
		if !hasTag(loot.Tags, "boss_component") {
			t.Fatalf("%s should be tagged as boss_component: %+v", item, loot.Tags)
		}
		if loot.Source == "" {
			t.Fatalf("%s should describe where to find it", item)
		}
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

func TestFireWeaponCreatesBulletImmediately(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("fire", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	player := r.Entities[r.Players["p1"].EntityID]
	r.Tick = 1
	r.ApplyInput(InputCommand{PlayerID: "p1", Aim: player.Position.Add(V(300, 0)), Fire: true})
	r.Step(c)
	if bulletCount(r) == 0 {
		t.Fatal("expected first fire input to spawn a bullet immediately")
	}
}

func TestCleanupKeepsZeroHPLootUntilTTL(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("loot-cleanup", c, "verdant", 42)
	id := r.next()
	r.Entities[id] = &Entity{ID: id, Kind: EntityLoot, DefID: "alien_alloy", Position: V(100, 100), Radius: 10, CarriedItem: "alien_alloy", TTL: 120}
	r.cleanup(c)
	if r.Entities[id] == nil {
		t.Fatal("loot with zero HP should persist until picked up or TTL expires")
	}
}

func TestSnapshotForPlayerFiltersDistantEntities(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("interest", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	player := r.Entities[r.Players["p1"].EntityID]
	player.Position = V(100, 100)
	nearID := r.next()
	r.Entities[nearID] = &Entity{ID: nearID, Kind: EntityEnemy, Position: player.Position.Add(V(120, 0)), Radius: 12, HP: 10, MaxHP: 10}
	farID := r.next()
	r.Entities[farID] = &Entity{ID: farID, Kind: EntityEnemy, Position: player.Position.Add(V(InterestRadius+500, 0)), Radius: 12, HP: 10, MaxHP: 10}
	bossID := r.next()
	r.Entities[bossID] = &Entity{ID: bossID, Kind: EntityBoss, Position: player.Position.Add(V(InterestRadius+900, 0)), Radius: 70, HP: 100, MaxHP: 100}
	snap := r.SnapshotForPlayer("p1")
	if !snapshotHasEntity(snap, nearID) {
		t.Fatal("near enemy should be included")
	}
	if snapshotHasEntity(snap, farID) {
		t.Fatal("far enemy should be filtered")
	}
	if !snapshotHasEntity(snap, bossID) {
		t.Fatal("boss should stay visible for encounter awareness")
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

func TestExtractionSchedulesTelegraphedWaves(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("extract-waves", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	player := r.Entities[r.Players["p1"].EntityID]
	player.Position = r.Map.Extraction
	r.ApplyInput(InputCommand{PlayerID: "p1", Aim: player.Position.Add(V(1, 0)), Extract: true})
	r.Step(c)
	firstWave := r.NextWaveTick
	if firstWave <= r.Tick {
		t.Fatalf("expected future wave tick, tick %d wave %d", r.Tick, firstWave)
	}
	telegraphTick := firstWave - uint64(TickRate*3)
	for r.Tick < telegraphTick-1 {
		r.ApplyInput(InputCommand{PlayerID: "p1", Aim: player.Position.Add(V(1, 0))})
		r.Step(c)
	}
	r.Step(c)
	if len(r.Messages) == 0 || r.Messages[len(r.Messages)-1] != "Enemy wave inbound at extraction." {
		t.Fatalf("expected wave telegraph message, got %+v", r.Messages)
	}
	before := enemyCount(r)
	for r.Tick < firstWave {
		r.Step(c)
	}
	r.Step(c)
	if enemyCount(r) <= before {
		t.Fatalf("expected extraction wave enemies, before %d after %d", before, enemyCount(r))
	}
	if r.NextWaveTick <= r.Tick {
		t.Fatalf("expected next wave rescheduled, tick %d wave %d", r.Tick, r.NextWaveTick)
	}
}

func TestObjectiveInteractionCompletesObjective(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("objective", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	ps := r.Players["p1"]
	player := r.Entities[ps.EntityID]
	player.Position = r.Map.Objectives[0].Position
	for i := 0; i < TickRate*4; i++ {
		r.ApplyInput(InputCommand{PlayerID: "p1", Aim: player.Position.Add(V(1, 0)), Extract: true})
		r.Step(c)
	}
	if !r.Map.Objectives[0].Done {
		t.Fatalf("expected objective complete, progress %.2f", r.Map.Objectives[0].Progress)
	}
	if ps.Stats.ObjectivesCompleted != 1 {
		t.Fatalf("expected objective stat, got %+v", ps.Stats)
	}
}

func TestMiningResourceAddsCarriedLoot(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("mine", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	ps := r.Players["p1"]
	player := r.Entities[ps.EntityID]
	resource := r.Map.Resources[0]
	player.Position = resource.Position
	for i := 0; i < TickRate*2; i++ {
		r.ApplyInput(InputCommand{PlayerID: "p1", Aim: player.Position.Add(V(1, 0)), Extract: true})
		r.Step(c)
	}
	if !r.Map.Resources[0].Done {
		t.Fatalf("expected resource mined, progress %.2f", r.Map.Resources[0].Progress)
	}
	if ps.Carried.Items[resource.Kind] != 1 {
		t.Fatalf("expected carried %s, got %+v", resource.Kind, ps.Carried.Items)
	}
	if ps.Stats.ResourcesMined != 1 {
		t.Fatalf("expected mining stat, got %+v", ps.Stats)
	}
}

func TestExtractionStatsValueAndCredits(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("extract-stats", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	ps := r.Players["p1"]
	ps.Carried.Add("alien_alloy", 4)
	ps.Stats.Kills = 2
	ps.Stats.ObjectivesCompleted = 1
	ps.Stats.ResourcesMined = 1
	ps.Extracting = true
	ps.ExtractStart = r.Tick
	for i := 0; i < TickRate*46; i++ {
		r.Step(c)
	}
	if !ps.Extracted {
		t.Fatal("expected player extracted")
	}
	if ps.Stats.LootExtractedValue != 72 {
		t.Fatalf("expected loot value 72, got %+v", ps.Stats)
	}
	if ps.Stats.CreditsEarned != 198 {
		t.Fatalf("expected credits 198, got %+v", ps.Stats)
	}
}

func TestDownedStatsRecordLostLootValue(t *testing.T) {
	c, err := LoadCatalog("../../content/game.json")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRun("lost-value", c, "verdant", 42)
	r.AddPlayer("p1", "pilot", DefaultLoadout())
	ps := r.Players["p1"]
	ps.Carried.Add("alien_alloy", 3)
	player := r.Entities[ps.EntityID]
	player.Position = V(500, 500)
	def := c.EnemyByID["skitter"]
	r.spawnEnemy(def, V(500, 500))
	for _, e := range r.Entities {
		if e.Kind == EntityEnemy {
			e.AIState = AILunge
			e.StateTick = 0
			e.Damage = 200
		}
	}
	r.Tick = 5
	r.resolveCombat(c)
	if ps.Stats.LootLostValue != 54 {
		t.Fatalf("expected lost loot value 54, got %+v", ps.Stats)
	}
	if len(ps.Carried.Items) != 0 {
		t.Fatalf("expected carried loot cleared, got %+v", ps.Carried.Items)
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

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func enemyCount(r *RunState) int {
	count := 0
	for _, e := range r.Entities {
		if e.Kind == EntityEnemy {
			count++
		}
	}
	return count
}

func bulletCount(r *RunState) int {
	count := 0
	for _, e := range r.Entities {
		if e.Kind == EntityBullet {
			count++
		}
	}
	return count
}

func snapshotHasEntity(s Snapshot, id EntityID) bool {
	for _, e := range s.Entities {
		if e.ID == id {
			return true
		}
	}
	return false
}
