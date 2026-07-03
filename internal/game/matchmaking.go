package game

import (
	"fmt"
	"sync"
	"time"
)

type QueueTicket struct {
	PlayerID PlayerID  `json:"player_id"`
	Name     string    `json:"name"`
	PlanetID string    `json:"planet_id"`
	Loadout  Loadout   `json:"loadout"`
	QueuedAt time.Time `json:"queued_at"`
}

type Matchmaker struct {
	mu      sync.Mutex
	waiting []QueueTicket
	nextRun int
}

func (m *Matchmaker) Enqueue(t QueueTicket) (string, []QueueTicket) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.waiting = append(m.waiting, t)
	if len(m.waiting) < 1 {
		return "", nil
	}
	group := m.waiting
	if len(group) > 4 {
		group = group[:4]
	}
	m.waiting = m.waiting[len(group):]
	m.nextRun++
	return fmt.Sprintf("run-%06d", m.nextRun), group
}
