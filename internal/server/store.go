package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	Token    string
	Strict   bool
	client   *http.Client
	fallback *MemoryStore
}

func NewSpaceTimeDBAdapter(endpoint, database string) *SpaceTimeDBAdapter {
	return &SpaceTimeDBAdapter{
		Endpoint: strings.TrimRight(endpoint, "/"),
		Database: database,
		Token:    os.Getenv("SPACETIMEDB_TOKEN"),
		Strict:   strings.EqualFold(os.Getenv("SPACETIMEDB_STRICT"), "true"),
		client:   &http.Client{Timeout: 3 * time.Second},
		fallback: NewMemoryStore(),
	}
}

func (s *SpaceTimeDBAdapter) LoadAccount(ctx context.Context, id game.PlayerID, name string) (game.Account, error) {
	if err := s.callReducer(ctx, "upsert_player_account", []any{string(id), name}); err != nil && s.Strict {
		return game.Account{}, err
	}
	return s.fallback.LoadAccount(ctx, id, name)
}

func (s *SpaceTimeDBAdapter) SaveAccount(ctx context.Context, account game.Account) error {
	payload, err := json.Marshal(account)
	if err != nil {
		return err
	}
	if err := s.callReducer(ctx, "save_player_account", []any{string(payload)}); err != nil {
		if s.Strict {
			return err
		}
	}
	return s.fallback.SaveAccount(ctx, account)
}

func (s *SpaceTimeDBAdapter) RecordRunSnapshot(ctx context.Context, snapshot game.Snapshot) error {
	if snapshot.Tick%uint64(game.TickRate*5) == 0 {
		payload, err := json.Marshal(snapshot.Entities)
		if err != nil {
			return err
		}
		if err := s.callReducer(ctx, "record_entity_snapshot", []any{snapshot.RunID, int64(snapshot.Tick), string(payload)}); err != nil && s.Strict {
			return err
		}
	}
	return s.fallback.RecordRunSnapshot(ctx, snapshot)
}

func (s *SpaceTimeDBAdapter) PublishWorldEvent(ctx context.Context, event game.WorldEvent) error {
	if err := s.callReducer(ctx, "publish_world_event", []any{event.ID, event.EventID, event.PlanetID, event.StartsUTC.Format(time.RFC3339), event.EndsUTC.Format(time.RFC3339)}); err != nil && s.Strict {
		return err
	}
	return s.fallback.PublishWorldEvent(ctx, event)
}

func (s *SpaceTimeDBAdapter) callReducer(ctx context.Context, reducer string, args []any) error {
	if s.Endpoint == "" || s.Database == "" || s.client == nil {
		return fmt.Errorf("spacetimedb adapter is not configured")
	}
	body, err := json.Marshal(args)
	if err != nil {
		return err
	}
	endpoint := s.Endpoint + "/v1/database/" + url.PathEscape(s.Database) + "/call/" + url.PathEscape(reducer)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.Token != "" {
		req.Header.Set("Authorization", "Bearer "+s.Token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	return fmt.Errorf("spacetimedb reducer %s failed: status %d %s", reducer, resp.StatusCode, strings.TrimSpace(string(msg)))
}
