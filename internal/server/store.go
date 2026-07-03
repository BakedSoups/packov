package server

import (
	"context"
	"sync"
	"time"

	"packov/internal/game"
)

type Persistence interface {
	LoadAccount(ctx context.Context, id game.PlayerID, name string) (game.Account, error)
	SaveAccount(ctx context.Context, account game.Account) error
	RecordRunSnapshot(ctx context.Context, snapshot game.Snapshot) error
	PublishWorldEvent(ctx context.Context, event game.WorldEvent) error
}

type MemoryStore struct {
	mu        sync.Mutex
	accounts  map[game.PlayerID]game.Account
	snapshots []game.Snapshot
	events    []game.WorldEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{accounts: map[game.PlayerID]game.Account{}}
}

func (m *MemoryStore) LoadAccount(_ context.Context, id game.PlayerID, name string) (game.Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.accounts[id]
	if !ok {
		a = game.NewAccount(id, name)
	}
	a.LastSeenUTC = time.Now().UTC()
	m.accounts[id] = a
	return a, nil
}

func (m *MemoryStore) SaveAccount(_ context.Context, account game.Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	account.LastSeenUTC = time.Now().UTC()
	m.accounts[account.ID] = account
	return nil
}

func (m *MemoryStore) RecordRunSnapshot(_ context.Context, snapshot game.Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if snapshot.Tick%uint64(game.TickRate*5) == 0 {
		m.snapshots = append(m.snapshots, snapshot)
	}
	if len(m.snapshots) > 300 {
		m.snapshots = m.snapshots[len(m.snapshots)-300:]
	}
	return nil
}

func (m *MemoryStore) PublishWorldEvent(_ context.Context, event game.WorldEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	if len(m.events) > 64 {
		m.events = m.events[len(m.events)-64:]
	}
	return nil
}

type SpaceTimeDBAdapter struct {
	Endpoint string
	Database string
	fallback *MemoryStore
}

func NewSpaceTimeDBAdapter(endpoint, database string) *SpaceTimeDBAdapter {
	return &SpaceTimeDBAdapter{Endpoint: endpoint, Database: database, fallback: NewMemoryStore()}
}

func (s *SpaceTimeDBAdapter) LoadAccount(ctx context.Context, id game.PlayerID, name string) (game.Account, error) {
	// The adapter boundary mirrors SpaceTimeDB reducers/tables in spacetime/schema.sql.
	// The in-memory fallback keeps local development and tests deterministic while
	// generated SpaceTimeDB bindings can replace these calls without touching gameplay.
	return s.fallback.LoadAccount(ctx, id, name)
}

func (s *SpaceTimeDBAdapter) SaveAccount(ctx context.Context, account game.Account) error {
	return s.fallback.SaveAccount(ctx, account)
}

func (s *SpaceTimeDBAdapter) RecordRunSnapshot(ctx context.Context, snapshot game.Snapshot) error {
	return s.fallback.RecordRunSnapshot(ctx, snapshot)
}

func (s *SpaceTimeDBAdapter) PublishWorldEvent(ctx context.Context, event game.WorldEvent) error {
	return s.fallback.PublishWorldEvent(ctx, event)
}
