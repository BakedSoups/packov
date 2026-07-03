package game

import (
	"math/rand"
)

type Objective struct {
	ID       EntityID `json:"id"`
	Kind     string   `json:"kind"`
	Position Vec2     `json:"position"`
	Done     bool     `json:"done"`
}

type Hazard struct {
	Kind     string  `json:"kind"`
	Position Vec2    `json:"position"`
	Radius   float64 `json:"radius"`
}

type PlanetMap struct {
	Seed         int64       `json:"seed"`
	PlanetID     string      `json:"planet_id"`
	Size         Vec2        `json:"size"`
	Spawn        Vec2        `json:"spawn"`
	Extraction   Vec2        `json:"extraction"`
	Objectives   []Objective `json:"objectives"`
	Hazards      []Hazard    `json:"hazards"`
	Resources    []Objective `json:"resources"`
	BossPosition Vec2        `json:"boss_position"`
}

func GeneratePlanet(seed int64, p PlanetDef) PlanetMap {
	r := rand.New(rand.NewSource(seed))
	size := V(2600+float64(p.Threat)*330, 2200+float64(p.Threat)*290)
	pm := PlanetMap{
		Seed:         seed,
		PlanetID:     p.ID,
		Size:         size,
		Spawn:        V(180, size.Y*0.5),
		Extraction:   V(size.X-240, size.Y*0.5),
		BossPosition: V(size.X*0.72, size.Y*0.5),
	}
	for i := 0; i < 3+p.Threat; i++ {
		pm.Objectives = append(pm.Objectives, Objective{
			ID:       EntityID(1000 + i),
			Kind:     []string{"uplink", "sample", "beacon", "reactor"}[i%4],
			Position: V(360+r.Float64()*(size.X-720), 260+r.Float64()*(size.Y-520)),
		})
	}
	for i, h := range p.Hazards {
		pm.Hazards = append(pm.Hazards, Hazard{
			Kind:     h,
			Position: V(260+r.Float64()*(size.X-520), 220+r.Float64()*(size.Y-440)),
			Radius:   90 + float64(30*i) + r.Float64()*80,
		})
	}
	for i := 0; i < 8+p.Threat*3; i++ {
		pm.Resources = append(pm.Resources, Objective{
			ID:       EntityID(2000 + i),
			Kind:     p.Resources[i%len(p.Resources)],
			Position: V(260+r.Float64()*(size.X-520), 220+r.Float64()*(size.Y-440)),
		})
	}
	return pm
}
