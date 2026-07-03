package game

import (
	"math"
	"math/rand"
	"time"
)

const TickRate = 30

type EntityKind string

const (
	EntityPlayer EntityKind = "player"
	EntityEnemy  EntityKind = "enemy"
	EntityBoss   EntityKind = "boss"
	EntityBullet EntityKind = "bullet"
	EntityLoot   EntityKind = "loot"
	EntityDrone  EntityKind = "drone"
	EntityTurret EntityKind = "turret"
)

type Entity struct {
	ID          EntityID   `json:"id"`
	Kind        EntityKind `json:"kind"`
	Owner       PlayerID   `json:"owner,omitempty"`
	DefID       string     `json:"def_id"`
	Position    Vec2       `json:"position"`
	Velocity    Vec2       `json:"velocity"`
	Rotation    float64    `json:"rotation"`
	Radius      float64    `json:"radius"`
	HP          float64    `json:"hp"`
	MaxHP       float64    `json:"max_hp"`
	Damage      float64    `json:"damage"`
	TTL         float64    `json:"ttl"`
	CarriedItem string     `json:"carried_item,omitempty"`
	Phase       int        `json:"phase"`
	Shield      float64    `json:"shield"`
}

type PlayerState struct {
	ID              PlayerID  `json:"id"`
	Name            string    `json:"name"`
	EntityID        EntityID  `json:"entity_id"`
	Loadout         Loadout   `json:"loadout"`
	Carried         Inventory `json:"carried"`
	LastShotAtTick  uint64    `json:"last_shot_at_tick"`
	LastAbilityTick uint64    `json:"last_ability_tick"`
	Extracting      bool      `json:"extracting"`
	ExtractStart    uint64    `json:"extract_start"`
	Extracted       bool      `json:"extracted"`
	Downed          bool      `json:"downed"`
}

type RunPhase string

const (
	PhaseStation    RunPhase = "station"
	PhasePlanet     RunPhase = "planet"
	PhaseExtraction RunPhase = "extraction"
	PhaseComplete   RunPhase = "complete"
	PhaseFailed     RunPhase = "failed"
)

type RunState struct {
	ID            string                    `json:"id"`
	Planet        PlanetDef                 `json:"planet"`
	Map           PlanetMap                 `json:"map"`
	Seed          int64                     `json:"seed"`
	Phase         RunPhase                  `json:"phase"`
	Tick          uint64                    `json:"tick"`
	Entities      map[EntityID]*Entity      `json:"entities"`
	Players       map[PlayerID]*PlayerState `json:"players"`
	PendingInputs map[PlayerID]InputCommand `json:"-"`
	Messages      []string                  `json:"messages"`
	nextID        EntityID
	rng           *rand.Rand
}

type Snapshot struct {
	RunID    string        `json:"run_id"`
	Tick     uint64        `json:"tick"`
	Phase    RunPhase      `json:"phase"`
	Planet   string        `json:"planet"`
	Map      PlanetMap     `json:"map"`
	Entities []Entity      `json:"entities"`
	Players  []PlayerState `json:"players"`
	Messages []string      `json:"messages"`
}

func NewRun(id string, catalog *Catalog, planetID string, seed int64) *RunState {
	p := catalog.PlanetByID[planetID]
	r := &RunState{
		ID:            id,
		Planet:        p,
		Map:           GeneratePlanet(seed, p),
		Seed:          seed,
		Phase:         PhasePlanet,
		Entities:      map[EntityID]*Entity{},
		Players:       map[PlayerID]*PlayerState{},
		PendingInputs: map[PlayerID]InputCommand{},
		nextID:        1,
		rng:           rand.New(rand.NewSource(seed + 99)),
	}
	return r
}

func (r *RunState) next() EntityID {
	r.nextID++
	return r.nextID
}

func (r *RunState) AddPlayer(id PlayerID, name string, loadout Loadout) {
	eid := r.next()
	r.Entities[eid] = &Entity{
		ID:       eid,
		Kind:     EntityPlayer,
		Owner:    id,
		DefID:    "hull_scout",
		Position: r.Map.Spawn.Add(V(r.rng.Float64()*80, r.rng.Float64()*80-40)),
		Radius:   18,
		HP:       100,
		MaxHP:    100,
	}
	r.Players[id] = &PlayerState{
		ID:       id,
		Name:     name,
		EntityID: eid,
		Loadout:  loadout,
		Carried:  NewInventory(),
	}
}

func (r *RunState) SpawnInitial(c *Catalog) {
	for i := 0; i < 22+r.Planet.Threat*8; i++ {
		def := c.Enemies[i%len(c.Enemies)]
		r.spawnEnemy(def, V(300+r.rng.Float64()*(r.Map.Size.X-600), 220+r.rng.Float64()*(r.Map.Size.Y-440)))
	}
	bd := c.BossByID[r.Planet.Boss]
	eid := r.next()
	r.Entities[eid] = &Entity{
		ID:       eid,
		Kind:     EntityBoss,
		DefID:    bd.ID,
		Position: r.Map.BossPosition,
		Radius:   76,
		HP:       bd.HP,
		MaxHP:    bd.HP,
		Damage:   28,
	}
}

func (r *RunState) ApplyInput(cmd InputCommand) {
	r.PendingInputs[cmd.PlayerID] = cmd
}

func (r *RunState) Step(c *Catalog) {
	r.Tick++
	dt := 1.0 / TickRate
	r.updatePlayers(c, dt)
	r.updateAI(c, dt)
	r.integrate(dt)
	r.resolveCombat(c)
	r.updateExtraction(c)
	r.updateRunOutcome()
	r.cleanup(c)
	if r.Phase != PhaseComplete && r.Phase != PhaseFailed && r.Tick%uint64(TickRate*18) == 0 {
		r.spawnWave(c, 4+r.Planet.Threat*2, r.Map.Extraction)
	}
}

func (r *RunState) updatePlayers(c *Catalog, dt float64) {
	for pid, ps := range r.Players {
		e := r.Entities[ps.EntityID]
		if e == nil || ps.Extracted || ps.Downed {
			continue
		}
		cmd := r.PendingInputs[pid]
		e.Velocity = cmd.Move.Clamp(1).Mul(230)
		if cmd.Aim.Len2() > 0 {
			e.Rotation = Angle(cmd.Aim.Sub(e.Position))
		}
		if cmd.Fire {
			r.fireWeapon(c, ps, e)
		}
		if cmd.Ability {
			r.useAbility(c, ps, e)
		}
		if cmd.Extract && Dist(e.Position, r.Map.Extraction) < 130 {
			if !ps.Extracting {
				ps.Extracting = true
				ps.ExtractStart = r.Tick
				r.Phase = PhaseExtraction
				r.Messages = append(r.Messages, "Extraction beacon broadcast. Defend the zone.")
			}
		}
		e.Shield = math.Max(0, e.Shield-dt*8)
	}
}

func (r *RunState) fireWeapon(c *Catalog, ps *PlayerState, e *Entity) {
	w := c.WeaponByID[ps.Loadout.WeaponID]
	cooldownTicks := uint64(math.Ceil(float64(w.CooldownMS) / 1000 * TickRate))
	if r.Tick-ps.LastShotAtTick < cooldownTicks {
		return
	}
	ps.LastShotAtTick = r.Tick
	pellets := w.Pellets
	if pellets < 1 {
		pellets = 1
	}
	for i := 0; i < pellets; i++ {
		spread := (r.rng.Float64()*2 - 1) * w.Spread
		dir := FromAngle(e.Rotation + spread)
		eid := r.next()
		r.Entities[eid] = &Entity{
			ID:       eid,
			Kind:     EntityBullet,
			Owner:    ps.ID,
			DefID:    w.ID,
			Position: e.Position.Add(dir.Mul(e.Radius + 8)),
			Velocity: dir.Mul(w.Speed),
			Rotation: e.Rotation,
			Radius:   4,
			Damage:   w.Damage,
			TTL:      w.Range / w.Speed,
		}
	}
}

func (r *RunState) useAbility(c *Catalog, ps *PlayerState, e *Entity) {
	a := c.AbilityByID[ps.Loadout.AbilityID]
	cooldownTicks := uint64(math.Ceil(float64(a.CooldownMS) / 1000 * TickRate))
	if r.Tick-ps.LastAbilityTick < cooldownTicks {
		return
	}
	ps.LastAbilityTick = r.Tick
	switch a.ID {
	case "dash":
		e.Position = e.Position.Add(FromAngle(e.Rotation).Mul(a.Power))
	case "shield":
		e.Shield = a.Power
	case "heal_pulse":
		e.HP = math.Min(e.MaxHP, e.HP+a.Power)
	case "emp":
		for _, target := range r.Entities {
			if (target.Kind == EntityEnemy || target.Kind == EntityBoss) && Dist(e.Position, target.Position) < a.Power {
				target.Velocity = target.Velocity.Mul(0.2)
				target.HP -= 18
			}
		}
	case "drone", "turret":
		eid := r.next()
		kind := EntityDrone
		if a.ID == "turret" {
			kind = EntityTurret
		}
		r.Entities[eid] = &Entity{ID: eid, Kind: kind, Owner: ps.ID, DefID: a.ID, Position: e.Position, Radius: 12, HP: 50, MaxHP: 50, Damage: a.Power, TTL: float64(a.DurationMS) / 1000}
	}
}

func (r *RunState) updateAI(c *Catalog, dt float64) {
	for _, e := range r.Entities {
		switch e.Kind {
		case EntityEnemy:
			target := r.closestPlayer(e.Position)
			if target == nil {
				continue
			}
			def := c.EnemyByID[e.DefID]
			if Dist(e.Position, target.Position) < def.Sense {
				dir := target.Position.Sub(e.Position).Normalize()
				e.Velocity = dir.Mul(def.Speed)
				e.Rotation = Angle(dir)
			}
		case EntityBoss:
			target := r.closestPlayer(e.Position)
			if target == nil {
				continue
			}
			bd := c.BossByID[e.DefID]
			for phase, pct := range bd.PhaseHP {
				if e.HP/e.MaxHP < pct && e.Phase <= phase {
					e.Phase = phase + 1
					r.spawnWave(c, 5+phase*3, e.Position)
					r.Messages = append(r.Messages, bd.Name+" shifted tactics.")
				}
			}
			dir := target.Position.Sub(e.Position).Normalize()
			orbit := V(-dir.Y, dir.X).Mul(math.Sin(float64(r.Tick)/30) * 90)
			e.Velocity = dir.Mul(45 + float64(e.Phase)*15).Add(orbit)
			e.Rotation += dt * (0.8 + float64(e.Phase)*0.4)
			if r.Tick%uint64(TickRate*(3-e.Phase/2)) == 0 {
				r.radialBossShot(e, 8+e.Phase*4)
			}
		case EntityDrone, EntityTurret:
			if e.Kind == EntityDrone {
				if owner := r.Entities[r.Players[e.Owner].EntityID]; owner != nil {
					e.Position = e.Position.Add(owner.Position.Sub(e.Position).Mul(0.05))
				}
			}
			if r.Tick%18 == 0 {
				if target := r.closestHostile(e.Position); target != nil {
					dir := target.Position.Sub(e.Position).Normalize()
					id := r.next()
					r.Entities[id] = &Entity{ID: id, Kind: EntityBullet, Owner: e.Owner, DefID: e.DefID, Position: e.Position, Velocity: dir.Mul(620), Rotation: Angle(dir), Radius: 4, Damage: e.Damage, TTL: 0.8}
				}
			}
		}
	}
}

func (r *RunState) integrate(dt float64) {
	for _, e := range r.Entities {
		e.Position = e.Position.Add(e.Velocity.Mul(dt))
		e.Position.X = math.Max(0, math.Min(r.Map.Size.X, e.Position.X))
		e.Position.Y = math.Max(0, math.Min(r.Map.Size.Y, e.Position.Y))
		if e.Kind == EntityBullet || e.Kind == EntityDrone || e.Kind == EntityTurret {
			e.TTL -= dt
		}
	}
}

func (r *RunState) resolveCombat(c *Catalog) {
	for _, b := range r.Entities {
		if b.Kind != EntityBullet {
			continue
		}
		for _, target := range r.Entities {
			if target.ID == b.ID || target.Owner == b.Owner {
				continue
			}
			if b.Owner != "" && (target.Kind == EntityEnemy || target.Kind == EntityBoss) && Dist(b.Position, target.Position) < b.Radius+target.Radius {
				target.HP -= b.Damage
				b.TTL = -1
				break
			}
			if b.Owner == "" && target.Kind == EntityPlayer && Dist(b.Position, target.Position) < b.Radius+target.Radius {
				target.HP -= math.Max(0, b.Damage-target.Shield*0.2)
				b.TTL = -1
				break
			}
		}
	}
	for _, hostile := range r.Entities {
		if hostile.Kind != EntityEnemy && hostile.Kind != EntityBoss {
			continue
		}
		for _, ps := range r.Players {
			player := r.Entities[ps.EntityID]
			if player == nil || ps.Downed || ps.Extracted {
				continue
			}
			if Dist(hostile.Position, player.Position) < hostile.Radius+player.Radius {
				player.HP -= math.Max(1, hostile.Damage/float64(TickRate)-player.Shield*0.02)
				if player.HP <= 0 {
					ps.Downed = true
					ps.Carried = NewInventory()
					r.Messages = append(r.Messages, ps.Name+" lost carried loot.")
				}
			}
		}
	}
}

func (r *RunState) updateExtraction(c *Catalog) {
	anyExtracting := false
	for _, ps := range r.Players {
		if ps.Extracting && !ps.Extracted && !ps.Downed {
			anyExtracting = true
			if r.Tick-ps.ExtractStart > uint64(TickRate*45) {
				ps.Extracted = true
				r.Messages = append(r.Messages, ps.Name+" extracted.")
			}
		}
	}
	if anyExtracting && r.Tick%uint64(TickRate*7) == 0 {
		r.spawnWave(c, 6+r.Planet.Threat*2, r.Map.Extraction)
	}
}

func (r *RunState) updateRunOutcome() {
	if r.Phase == PhaseComplete || r.Phase == PhaseFailed || len(r.Players) == 0 {
		return
	}
	active := 0
	extracted := 0
	downed := 0
	for _, ps := range r.Players {
		switch {
		case ps.Extracted:
			extracted++
		case ps.Downed:
			downed++
		default:
			active++
		}
	}
	if active > 0 {
		return
	}
	if extracted > 0 {
		r.Phase = PhaseComplete
		r.Messages = append(r.Messages, "Run complete. Extracted loot secured.")
		return
	}
	if downed > 0 {
		r.Phase = PhaseFailed
		r.Messages = append(r.Messages, "Squad wiped. Carried loot lost; account progression retained.")
	}
}

func (r *RunState) cleanup(c *Catalog) {
	for id, e := range r.Entities {
		if e.TTL < 0 || e.HP <= 0 {
			if e.Kind == EntityEnemy || e.Kind == EntityBoss {
				r.dropLoot(c, e.Position, e.Kind == EntityBoss)
			}
			delete(r.Entities, id)
		}
	}
	for _, ps := range r.Players {
		player := r.Entities[ps.EntityID]
		if player == nil {
			continue
		}
		for id, e := range r.Entities {
			if e.Kind == EntityLoot && Dist(player.Position, e.Position) < player.Radius+e.Radius+10 {
				ps.Carried.Add(e.CarriedItem, 1)
				delete(r.Entities, id)
			}
		}
	}
}

func (r *RunState) spawnEnemy(def EnemyDef, pos Vec2) {
	eid := r.next()
	r.Entities[eid] = &Entity{ID: eid, Kind: EntityEnemy, DefID: def.ID, Position: pos, Radius: 17, HP: def.HP, MaxHP: def.HP, Damage: def.Damage}
}

func (r *RunState) spawnWave(c *Catalog, count int, around Vec2) {
	for i := 0; i < count; i++ {
		def := c.Enemies[(int(r.Tick)+i)%len(c.Enemies)]
		angle := r.rng.Float64() * math.Pi * 2
		dist := 240 + r.rng.Float64()*360
		r.spawnEnemy(def, around.Add(FromAngle(angle).Mul(dist)))
	}
}

func (r *RunState) radialBossShot(e *Entity, count int) {
	for i := 0; i < count; i++ {
		dir := FromAngle(float64(i)/float64(count)*math.Pi*2 + e.Rotation)
		id := r.next()
		r.Entities[id] = &Entity{ID: id, Kind: EntityBullet, Position: e.Position.Add(dir.Mul(e.Radius)), Velocity: dir.Mul(360), Radius: 5, Damage: 15 + float64(e.Phase*5), TTL: 2.4}
	}
}

func (r *RunState) dropLoot(c *Catalog, pos Vec2, boss bool) {
	var item string
	if boss {
		bossDef := c.BossByID[r.Planet.Boss]
		item = bossDef.ExclusiveLoot[r.rng.Intn(len(bossDef.ExclusiveLoot))]
	} else {
		item = r.Planet.Resources[r.rng.Intn(len(r.Planet.Resources))]
	}
	id := r.next()
	r.Entities[id] = &Entity{ID: id, Kind: EntityLoot, DefID: item, Position: pos, Radius: 10, CarriedItem: item, TTL: 120}
}

func (r *RunState) closestPlayer(pos Vec2) *Entity {
	var best *Entity
	bestDist := math.MaxFloat64
	for _, ps := range r.Players {
		if ps.Downed || ps.Extracted {
			continue
		}
		if e := r.Entities[ps.EntityID]; e != nil {
			if d := Dist(pos, e.Position); d < bestDist {
				bestDist = d
				best = e
			}
		}
	}
	return best
}

func (r *RunState) closestHostile(pos Vec2) *Entity {
	var best *Entity
	bestDist := math.MaxFloat64
	for _, e := range r.Entities {
		if e.Kind != EntityEnemy && e.Kind != EntityBoss {
			continue
		}
		if d := Dist(pos, e.Position); d < bestDist {
			bestDist = d
			best = e
		}
	}
	return best
}

func (r *RunState) Snapshot() Snapshot {
	entities := make([]Entity, 0, len(r.Entities))
	for _, e := range r.Entities {
		entities = append(entities, *e)
	}
	players := make([]PlayerState, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, *p)
	}
	msg := r.Messages
	if len(msg) > 8 {
		msg = msg[len(msg)-8:]
	}
	return Snapshot{RunID: r.ID, Tick: r.Tick, Phase: r.Phase, Planet: r.Planet.Name, Map: r.Map, Entities: entities, Players: players, Messages: msg}
}

func DefaultLoadout() Loadout {
	return Loadout{WeaponID: "machine_gun", AbilityID: "dash", HullID: "hull_scout"}
}

func DefaultAppearance(name string) Appearance {
	if name == "" {
		name = "Pilot"
	}
	return Appearance{
		Callsign:  name,
		HullID:    "hull_scout",
		Primary:   "cyan",
		Secondary: "white",
		TrailID:   "ion",
		NoseID:    "arrow",
		DroneSkin: "standard",
		BadgeID:   "founder",
	}
}

func NewAccount(id PlayerID, name string) Account {
	return Account{
		ID:          id,
		Name:        name,
		Credits:     500,
		Level:       1,
		Unlocks:     map[string]bool{"machine_gun": true, "dash": true, "hull_scout": true},
		Inventory:   NewInventory(),
		Cosmetics:   []string{"cyan", "white", "amber", "green", "violet", "red", "ion", "spark", "comet", "pulse", "arrow", "split", "needle", "standard", "orbital", "wing", "founder", "boss", "guild", "season"},
		Appearance:  DefaultAppearance(name),
		LastSeenUTC: time.Now().UTC(),
	}
}
