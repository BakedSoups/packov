package game

import (
	"math"
	"math/rand"
	"time"
)

const TickRate = 30
const InterestRadius = 980.0

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
	AIState     string     `json:"ai_state,omitempty"`
	StateTick   uint64     `json:"state_tick,omitempty"`
	NextAction  uint64     `json:"next_action,omitempty"`
	HitTick     uint64     `json:"hit_tick,omitempty"`
	LastHitBy   PlayerID   `json:"last_hit_by,omitempty"`
}

type RunStats struct {
	Kills               int     `json:"kills"`
	ObjectivesCompleted int     `json:"objectives_completed"`
	ResourcesMined      int     `json:"resources_mined"`
	BossDamage          float64 `json:"boss_damage"`
	LootExtractedValue  int     `json:"loot_extracted_value"`
	LootLostValue       int     `json:"loot_lost_value"`
	CreditsEarned       int     `json:"credits_earned"`
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
	Stats           RunStats  `json:"stats"`
}

type RunPhase string

const (
	PhaseStation    RunPhase = "station"
	PhasePlanet     RunPhase = "planet"
	PhaseExtraction RunPhase = "extraction"
	PhaseComplete   RunPhase = "complete"
	PhaseFailed     RunPhase = "failed"
)

const (
	AIApproach = "approach"
	AIOrbit    = "orbit"
	AILunge    = "lunge"
	AIRecover  = "recover"
	AIRetreat  = "retreat"
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
	NextWaveTick  uint64                    `json:"next_wave_tick,omitempty"`
	nextID        EntityID
	rng           *rand.Rand
}

type Snapshot struct {
	RunID        string        `json:"run_id"`
	Tick         uint64        `json:"tick"`
	Phase        RunPhase      `json:"phase"`
	Planet       string        `json:"planet"`
	Map          PlanetMap     `json:"map"`
	Entities     []Entity      `json:"entities"`
	Players      []PlayerState `json:"players"`
	Messages     []string      `json:"messages"`
	NextWaveTick uint64        `json:"next_wave_tick,omitempty"`
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
	r.resolveEntitySeparation()
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
		if cmd.Extract {
			r.updateObjectiveInteraction(ps, e, dt)
		}
		if cmd.Extract && Dist(e.Position, r.Map.Extraction) < 130 {
			if !ps.Extracting {
				ps.Extracting = true
				ps.ExtractStart = r.Tick
				r.Phase = PhaseExtraction
				if r.NextWaveTick == 0 {
					r.NextWaveTick = r.Tick + uint64(TickRate*7)
				}
				r.Messages = append(r.Messages, "Extraction beacon broadcast. Defend the zone.")
			}
		}
		e.Shield = math.Max(0, e.Shield-dt*8)
	}
}

func (r *RunState) updateObjectiveInteraction(ps *PlayerState, e *Entity, dt float64) {
	if idx, ok := nearestObjective(r.Map.Objectives, e.Position, 82); ok {
		obj := &r.Map.Objectives[idx]
		obj.Progress = math.Min(1, obj.Progress+dt/3.0)
		if obj.Progress >= 1 && !obj.Done {
			obj.Done = true
			ps.Stats.ObjectivesCompleted++
			r.Messages = append(r.Messages, ps.Name+" completed "+obj.Kind+".")
		}
		return
	}
	if idx, ok := nearestObjective(r.Map.Resources, e.Position, 70); ok {
		node := &r.Map.Resources[idx]
		node.Progress = math.Min(1, node.Progress+dt/1.4)
		if node.Progress >= 1 && !node.Done {
			node.Done = true
			ps.Stats.ResourcesMined++
			ps.Carried.Add(node.Kind, 1)
			r.Messages = append(r.Messages, ps.Name+" mined "+node.Kind+".")
		}
	}
}

func nearestObjective(nodes []Objective, position Vec2, radius float64) (int, bool) {
	best := -1
	bestDist := radius
	for i, node := range nodes {
		if node.Done {
			continue
		}
		d := Dist(position, node.Position)
		if d <= bestDist {
			best = i
			bestDist = d
		}
	}
	return best, best >= 0
}

func (r *RunState) fireWeapon(c *Catalog, ps *PlayerState, e *Entity) {
	w := c.WeaponByID[ps.Loadout.WeaponID]
	cooldownTicks := uint64(math.Ceil(float64(w.CooldownMS) / 1000 * TickRate))
	if ps.LastShotAtTick != 0 && r.Tick-ps.LastShotAtTick < cooldownTicks {
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
			offset := target.Position.Sub(e.Position)
			dist := offset.Len()
			if dist < def.Sense {
				r.updateEnemyState(e, def, target, offset, dist)
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

func (r *RunState) updateEnemyState(e *Entity, def EnemyDef, target *Entity, offset Vec2, dist float64) {
	dir := offset.Normalize()
	if e.AIState == "" {
		e.AIState = AIApproach
		e.StateTick = r.Tick
	}
	preferred := enemyPreferredRange(e, target, def)
	attackRange := e.Radius + target.Radius + 12
	lungeTicks := uint64(8)
	recoverTicks := uint64(16)
	if def.Shape == "square" {
		lungeTicks = 11
		recoverTicks = 20
	}
	if def.Shape == "hexagon" {
		attackRange += 46
	}

	switch e.AIState {
	case AILunge:
		e.Velocity = dir.Mul(def.Speed * enemyLungeSpeed(def))
		if r.Tick-e.StateTick >= lungeTicks {
			r.setAIState(e, AIRecover)
			e.NextAction = r.Tick + recoverTicks
		}
	case AIRecover:
		backoff := dir.Mul(-def.Speed * 0.38)
		orbit := enemyOrbit(e, def, dir, r.Tick).Mul(0.45)
		e.Velocity = backoff.Add(orbit).Add(r.enemySeparation(e).Mul(def.Speed)).Clamp(def.Speed)
		if r.Tick >= e.NextAction && dist > attackRange+10 {
			r.setAIState(e, AIApproach)
		} else if r.Tick >= e.NextAction {
			r.setAIState(e, AIOrbit)
		}
	case AIRetreat:
		e.Velocity = dir.Mul(-def.Speed * 0.7).Add(r.enemySeparation(e).Mul(def.Speed)).Clamp(def.Speed * 1.1)
		if dist > preferred-12 {
			r.setAIState(e, AIOrbit)
		}
	case AIOrbit:
		if dist < attackRange {
			r.setAIState(e, AIRetreat)
			e.Velocity = dir.Mul(-def.Speed * 0.7).Add(r.enemySeparation(e).Mul(def.Speed)).Clamp(def.Speed * 1.1)
			break
		}
		if r.Tick >= e.NextAction && dist < preferred+26 {
			r.setAIState(e, AILunge)
			e.Velocity = dir.Mul(def.Speed * enemyLungeSpeed(def))
			break
		}
		hold := dir.Mul((dist - preferred) * 2.2)
		orbit := enemyOrbit(e, def, dir, r.Tick)
		e.Velocity = hold.Add(orbit).Add(r.enemySeparation(e).Mul(def.Speed * 1.2)).Clamp(def.Speed * 0.9)
	default:
		if dist < attackRange {
			r.setAIState(e, AIRetreat)
			e.Velocity = dir.Mul(-def.Speed * 0.7).Add(r.enemySeparation(e).Mul(def.Speed)).Clamp(def.Speed * 1.1)
			break
		}
		if dist < preferred+45 {
			r.setAIState(e, AIOrbit)
			hold := dir.Mul((dist - preferred) * 2.2)
			orbit := enemyOrbit(e, def, dir, r.Tick)
			e.Velocity = hold.Add(orbit).Add(r.enemySeparation(e).Mul(def.Speed * 1.2)).Clamp(def.Speed * 0.9)
			break
		}
		orbit := enemyOrbit(e, def, dir, r.Tick).Mul(0.25)
		e.Velocity = dir.Mul(def.Speed).Add(orbit).Add(r.enemySeparation(e).Mul(def.Speed * 1.2)).Clamp(def.Speed * 1.2)
	}
	if e.Velocity.Len2() > 0.01 {
		e.Rotation = Angle(e.Velocity)
	}
}

func (r *RunState) setAIState(e *Entity, state string) {
	if e.AIState == state {
		return
	}
	e.AIState = state
	e.StateTick = r.Tick
}

func enemyPreferredRange(e, target *Entity, def EnemyDef) float64 {
	base := e.Radius + target.Radius
	switch def.Shape {
	case "triangle":
		return base + 42
	case "square":
		return base + 34
	case "hexagon":
		return base + 96
	case "octagon":
		return base + 62
	default:
		return base + 50
	}
}

func enemyLungeSpeed(def EnemyDef) float64 {
	switch def.Shape {
	case "triangle":
		return 1.85
	case "square":
		return 1.2
	case "hexagon":
		return 0.75
	default:
		return 1.35
	}
}

func enemyOrbit(e *Entity, def EnemyDef, dir Vec2, tick uint64) Vec2 {
	side := 1.0
	if e.ID%2 == 0 {
		side = -1
	}
	wobble := 0.72 + 0.28*math.Sin(float64(tick)*0.035+float64(e.ID))
	speed := def.Speed * 0.55 * wobble
	if def.Shape == "triangle" {
		speed *= 1.35
	}
	if def.Shape == "square" {
		speed *= 0.45
	}
	return V(-dir.Y, dir.X).Mul(speed * side)
}

func (r *RunState) enemySeparation(e *Entity) Vec2 {
	force := Vec2{}
	for _, other := range r.Entities {
		if other.ID == e.ID || other.Kind != EntityEnemy {
			continue
		}
		minDist := e.Radius + other.Radius + 10
		delta := e.Position.Sub(other.Position)
		dist := delta.Len()
		if dist >= minDist {
			continue
		}
		if dist <= 0.001 {
			delta = FromAngle(float64(e.ID%17) * 0.77)
			dist = 0.001
		}
		force = force.Add(delta.Normalize().Mul((minDist - dist) / minDist))
	}
	return force.Clamp(1)
}

func (r *RunState) resolveEntitySeparation() {
	entities := make([]*Entity, 0, len(r.Entities))
	for _, e := range r.Entities {
		if e.Kind == EntityEnemy || e.Kind == EntityBoss {
			entities = append(entities, e)
		}
	}
	for i := 0; i < len(entities); i++ {
		for j := i + 1; j < len(entities); j++ {
			a := entities[i]
			b := entities[j]
			minDist := a.Radius + b.Radius + 2
			delta := a.Position.Sub(b.Position)
			dist := delta.Len()
			if dist >= minDist {
				continue
			}
			if dist <= 0.001 {
				delta = FromAngle(float64((a.ID+b.ID)%23) * 0.61)
				dist = 0.001
			}
			push := delta.Normalize().Mul((minDist - dist) * 0.5)
			if a.Kind == EntityBoss {
				b.Position = b.Position.Sub(push.Mul(2))
			} else if b.Kind == EntityBoss {
				a.Position = a.Position.Add(push.Mul(2))
			} else {
				a.Position = a.Position.Add(push)
				b.Position = b.Position.Sub(push)
			}
			a.Position.X = math.Max(0, math.Min(r.Map.Size.X, a.Position.X))
			a.Position.Y = math.Max(0, math.Min(r.Map.Size.Y, a.Position.Y))
			b.Position.X = math.Max(0, math.Min(r.Map.Size.X, b.Position.X))
			b.Position.Y = math.Max(0, math.Min(r.Map.Size.Y, b.Position.Y))
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
				damage := math.Min(target.HP, b.Damage)
				target.HP -= damage
				target.HitTick = r.Tick
				target.LastHitBy = b.Owner
				if target.Kind == EntityBoss {
					if ps := r.Players[b.Owner]; ps != nil {
						ps.Stats.BossDamage += damage
					}
				}
				b.TTL = -1
				break
			}
			if b.Owner == "" && target.Kind == EntityPlayer && Dist(b.Position, target.Position) < b.Radius+target.Radius {
				target.HP -= math.Max(0, b.Damage-target.Shield*0.2)
				target.HitTick = r.Tick
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
				damage := hostile.Damage / float64(TickRate)
				if hostile.Kind == EntityEnemy {
					if hostile.AIState != AILunge || r.Tick < hostile.StateTick+3 {
						continue
					}
					damage = hostile.Damage
					hostile.AIState = AIRecover
					hostile.StateTick = r.Tick
					hostile.NextAction = r.Tick + uint64(TickRate)
				}
				player.HP -= math.Max(1, damage-player.Shield*0.02)
				player.HitTick = r.Tick
				if player.HP <= 0 {
					ps.Downed = true
					ps.Stats.LootLostValue += inventoryValue(c, ps.Carried)
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
				ps.Stats.LootExtractedValue = inventoryValue(c, ps.Carried)
				ps.Stats.CreditsEarned = creditReward(ps)
				r.Messages = append(r.Messages, ps.Name+" extracted.")
			}
		}
	}
	if !anyExtracting {
		r.NextWaveTick = 0
		return
	}
	if r.NextWaveTick == 0 {
		r.NextWaveTick = r.Tick + uint64(TickRate*7)
	}
	if r.NextWaveTick > r.Tick && r.NextWaveTick-r.Tick == uint64(TickRate*3) {
		r.Messages = append(r.Messages, "Enemy wave inbound at extraction.")
	}
	if r.Tick >= r.NextWaveTick {
		r.spawnWave(c, 6+r.Planet.Threat*2, r.Map.Extraction)
		r.Messages = append(r.Messages, "Extraction wave has arrived.")
		r.NextWaveTick = r.Tick + uint64(TickRate*7)
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
		if e.TTL < 0 || (isDamageable(e.Kind) && e.HP <= 0) {
			if e.Kind == EntityEnemy || e.Kind == EntityBoss {
				if ps := r.Players[e.LastHitBy]; ps != nil {
					ps.Stats.Kills++
				}
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

func isDamageable(kind EntityKind) bool {
	return kind == EntityPlayer || kind == EntityEnemy || kind == EntityBoss || kind == EntityDrone || kind == EntityTurret
}

func (r *RunState) spawnEnemy(def EnemyDef, pos Vec2) {
	eid := r.next()
	r.Entities[eid] = &Entity{ID: eid, Kind: EntityEnemy, DefID: def.ID, Position: pos, Radius: 17, HP: def.HP, MaxHP: def.HP, Damage: def.Damage, AIState: AIApproach, StateTick: r.Tick}
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

func inventoryValue(c *Catalog, inv Inventory) int {
	total := 0
	for item, count := range inv.Items {
		if loot, ok := c.LootByID[item]; ok {
			total += loot.BaseValue * count
		}
	}
	return total
}

func creditReward(ps *PlayerState) int {
	return ps.Stats.Kills*20 + ps.Stats.ObjectivesCompleted*125 + ps.Stats.ResourcesMined*15 + ps.Stats.LootExtractedValue/4
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
	return r.snapshotFor("")
}

func (r *RunState) SnapshotForPlayer(playerID PlayerID) Snapshot {
	return r.snapshotFor(playerID)
}

func (r *RunState) snapshotFor(playerID PlayerID) Snapshot {
	var focus *Entity
	if playerID != "" {
		if ps := r.Players[playerID]; ps != nil {
			focus = r.Entities[ps.EntityID]
		}
	}
	entities := make([]Entity, 0, len(r.Entities))
	for _, e := range r.Entities {
		if focus != nil && !inInterestRange(focus, e) {
			continue
		}
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
	return Snapshot{RunID: r.ID, Tick: r.Tick, Phase: r.Phase, Planet: r.Planet.Name, Map: r.Map, Entities: entities, Players: players, Messages: msg, NextWaveTick: r.NextWaveTick}
}

func inInterestRange(focus, e *Entity) bool {
	if e.ID == focus.ID || e.Kind == EntityPlayer || e.Kind == EntityBoss {
		return true
	}
	return Dist(focus.Position, e.Position) <= InterestRadius
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
