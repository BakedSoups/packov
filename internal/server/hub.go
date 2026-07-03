package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"packov/internal/game"
)

type Hub struct {
	catalog *game.Catalog
	store   Persistence
	log     *slog.Logger

	mu      sync.RWMutex
	runs    map[string]*game.RunState
	players map[game.PlayerID]*Session
	match   game.Matchmaker
	event   game.WorldEvent
}

type Session struct {
	id      game.PlayerID
	name    string
	account game.Account
	conn    *websocket.Conn
	sendMu  sync.Mutex
	runID   string
}

func NewHub(c *game.Catalog, store Persistence, log *slog.Logger) *Hub {
	return &Hub{
		catalog: c,
		store:   store,
		log:     log,
		runs:    map[string]*game.RunState{},
		players: map[game.PlayerID]*Session{},
		event:   game.GenerateWorldEvent(c, time.Now()),
	}
}

func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second / game.TickRate)
	eventTicker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	defer eventTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.step(ctx)
		case <-eventTicker.C:
			ev := game.GenerateWorldEvent(h.catalog, time.Now())
			h.mu.Lock()
			changed := ev.ID != h.event.ID
			h.event = ev
			h.mu.Unlock()
			if changed {
				_ = h.store.PublishWorldEvent(ctx, ev)
				h.broadcast(ServerMessage{Type: "world_event", WorldEvent: &ev})
			}
		}
	}
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		h.log.Error("websocket accept", "err", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	ctx := r.Context()
	s := &Session{conn: conn}
	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var msg ClientMessage
		if err := json.Unmarshal(b, &msg); err != nil {
			s.write(ctx, ServerMessage{Type: "error", Error: "bad json"})
			continue
		}
		if err := h.handle(ctx, s, msg); err != nil {
			s.write(ctx, ServerMessage{Type: "error", Error: err.Error()})
		}
	}
}

func (h *Hub) handle(ctx context.Context, s *Session, msg ClientMessage) error {
	switch msg.Type {
	case "hello":
		id := game.PlayerID(msg.Token)
		if id == "" {
			id = game.PlayerID(fmt.Sprintf("guest-%d", time.Now().UnixNano()))
		}
		name := msg.Name
		if name == "" {
			name = string(id)
		}
		account, err := h.store.LoadAccount(ctx, id, name)
		if err != nil {
			return err
		}
		s.id, s.name, s.account = id, name, account
		h.mu.Lock()
		h.players[id] = s
		h.mu.Unlock()
		ev := h.event
		missions := game.DailyMissions(time.Now())
		return s.write(ctx, ServerMessage{Type: "hello", PlayerID: id, Account: &account, Catalog: h.catalog, WorldEvent: &ev, Missions: missions})
	case "queue":
		if s.id == "" {
			return fmt.Errorf("authenticate first")
		}
		loadout := msg.Loadout
		if loadout.WeaponID == "" {
			loadout = game.DefaultLoadout()
		}
		planetID := msg.PlanetID
		if planetID == "" {
			planetID = "verdant"
		}
		runID, group := h.match.Enqueue(game.QueueTicket{PlayerID: s.id, Name: s.name, PlanetID: planetID, Loadout: loadout, QueuedAt: time.Now()})
		if runID == "" {
			return nil
		}
		seed := time.Now().UnixNano() ^ int64(rand.Int())
		run := game.NewRun(runID, h.catalog, planetID, seed)
		for _, t := range group {
			run.AddPlayer(t.PlayerID, t.Name, t.Loadout)
		}
		run.SpawnInitial(h.catalog)
		h.mu.Lock()
		h.runs[runID] = run
		for _, t := range group {
			if ss := h.players[t.PlayerID]; ss != nil {
				ss.runID = runID
			}
		}
		h.mu.Unlock()
		snap := run.Snapshot()
		for _, t := range group {
			if ss := h.session(t.PlayerID); ss != nil {
				_ = ss.write(ctx, ServerMessage{Type: "match", Snapshot: &snap})
			}
		}
	case "input":
		if s.id == "" {
			return fmt.Errorf("authenticate first")
		}
		h.mu.RLock()
		run := h.runs[s.runID]
		h.mu.RUnlock()
		if run != nil {
			msg.Input.PlayerID = s.id
			run.ApplyInput(msg.Input)
		}
	case "craft":
		recipe, ok := h.catalog.RecipeByID[msg.RecipeID]
		if !ok {
			return fmt.Errorf("unknown recipe")
		}
		if err := game.Craft(&s.account, recipe); err != nil {
			return err
		}
		return s.write(ctx, ServerMessage{Type: "account", Account: &s.account})
	}
	return nil
}

func (h *Hub) step(ctx context.Context) {
	h.mu.RLock()
	runs := make([]*game.RunState, 0, len(h.runs))
	for _, r := range h.runs {
		runs = append(runs, r)
	}
	h.mu.RUnlock()
	for _, r := range runs {
		r.Step(h.catalog)
		snap := r.Snapshot()
		_ = h.store.RecordRunSnapshot(ctx, snap)
		for _, ps := range snap.Players {
			if ss := h.session(ps.ID); ss != nil && ss.runID == r.ID {
				_ = ss.write(ctx, ServerMessage{Type: "snapshot", Snapshot: &snap})
			}
		}
	}
}

func (h *Hub) session(id game.PlayerID) *Session {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.players[id]
}

func (h *Hub) broadcast(msg ServerMessage) {
	h.mu.RLock()
	sessions := make([]*Session, 0, len(h.players))
	for _, s := range h.players {
		sessions = append(sessions, s)
	}
	h.mu.RUnlock()
	for _, s := range sessions {
		_ = s.write(context.Background(), msg)
	}
}

func (s *Session) write(ctx context.Context, msg ServerMessage) error {
	if s.conn == nil {
		return nil
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return s.conn.Write(ctx, websocket.MessageText, b)
}
