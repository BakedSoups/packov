package game

import (
	"crypto/sha1"
	"encoding/hex"
	"math/rand"
	"time"
)

type WorldEvent struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	Name      string    `json:"name"`
	StartsUTC time.Time `json:"starts_utc"`
	EndsUTC   time.Time `json:"ends_utc"`
	PlanetID  string    `json:"planet_id"`
	Progress  float64   `json:"progress"`
}

type DailyMission struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Target      int    `json:"target"`
	Reward      int    `json:"reward"`
}

func GenerateWorldEvent(c *Catalog, now time.Time) WorldEvent {
	slot := now.UTC().Unix() / int64((45 * time.Minute).Seconds())
	r := rand.New(rand.NewSource(slot))
	ev := c.Events[r.Intn(len(c.Events))]
	planet := c.Planets[r.Intn(len(c.Planets))]
	sum := sha1.Sum([]byte(ev.ID + planet.ID + now.Format("2006-01-02-15")))
	return WorldEvent{
		ID:        hex.EncodeToString(sum[:6]),
		EventID:   ev.ID,
		Name:      ev.Name,
		StartsUTC: time.Unix(slot*int64((45*time.Minute).Seconds()), 0).UTC(),
		EndsUTC:   time.Unix((slot+1)*int64((45*time.Minute).Seconds()), 0).UTC(),
		PlanetID:  planet.ID,
	}
}

func DailyMissions(now time.Time) []DailyMission {
	day := now.UTC().YearDay()
	return []DailyMission{
		{ID: "daily_extract", Description: "Extract from any planet", Target: 1, Reward: 300 + day%100},
		{ID: "daily_salvage", Description: "Recover resources", Target: 12, Reward: 220},
		{ID: "daily_boss", Description: "Contribute boss damage", Target: 1000, Reward: 500},
	}
}
