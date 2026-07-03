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
	"packov/internal/protocol"
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
	id               game.PlayerID
	name             string
	account          game.Account
	conn             *websocket.Conn
	sendMu           sync.Mutex
	runID            string
	lastSeq          uint64
	inputWindowStart time.Time
	inputCount       int
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
				h.broadcast(protocol.ServerMessage{Type: "world_event", WorldEvent: &ev})
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
		var msg protocol.ClientMessage
		if err := json.Unmarshal(b, &msg); err != nil {
			s.write(ctx, protocol.ServerMessage{Type: "error", Error: "bad json"})
			continue
		}
		if err := h.handle(ctx, s, msg); err != nil {
			s.write(ctx, protocol.ServerMessage{Type: "error", Error: err.Error()})
		}
	}
}

func (h *Hub) handle(ctx context.Context, s *Session, msg protocol.ClientMessage) error {
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
		if account.Appearance.Callsign == "" {
			account.Appearance = game.DefaultAppearance(account.Name)
		}
		s.id, s.name, s.account = id, name, account
		h.mu.Lock()
		h.players[id] = s
		if account.CurrentRun != "" {
			if run := h.runs[account.CurrentRun]; run != nil {
				s.runID = account.CurrentRun
			}
		}
		h.mu.Unlock()
		ev := h.event
		missions := game.DailyMissions(time.Now())
		if err := s.write(ctx, protocol.ServerMessage{Type: "hello", PlayerID: id, Account: &account, Catalog: h.catalog, WorldEvent: &ev, Missions: missions}); err != nil {
			return err
		}
		if s.runID != "" {
			h.mu.RLock()
			run := h.runs[s.runID]
			h.mu.RUnlock()
			if run != nil {
				snap := run.Snapshot()
				return s.write(ctx, protocol.ServerMessage{Type: "match", Snapshot: &snap})
			}
		}
		return nil
	case "queue":
		if s.id == "" {
			return fmt.Errorf("authenticate first")
		}
		loadout := msg.Loadout
		if loadout.WeaponID == "" {
			loadout = game.DefaultLoadout()
		}
		if err := validateLoadout(s.account, loadout); err != nil {
			return err
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
				ss.account.CurrentRun = runID
				_ = h.store.SaveAccount(ctx, ss.account)
			}
		}
		h.mu.Unlock()
		snap := run.Snapshot()
		for _, t := range group {
			if ss := h.session(t.PlayerID); ss != nil {
				_ = ss.write(ctx, protocol.ServerMessage{Type: "match", Snapshot: &snap})
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
			if err := s.acceptInput(msg.Input); err != nil {
				return err
			}
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
		if err := h.store.SaveAccount(ctx, s.account); err != nil {
			return err
		}
		return s.write(ctx, protocol.ServerMessage{Type: "account", Account: &s.account})
	case "appearance":
		if s.id == "" {
			return fmt.Errorf("authenticate first")
		}
		if err := validateAppearance(s.account, msg.Appearance); err != nil {
			return err
		}
		s.account.Appearance = msg.Appearance
		if s.account.Appearance.Callsign == "" {
			s.account.Appearance.Callsign = s.name
		}
		if err := h.store.SaveAccount(ctx, s.account); err != nil {
			return err
		}
		return s.write(ctx, protocol.ServerMessage{Type: "account", Account: &s.account})
	}
	return nil
}

func validateAppearance(account game.Account, appearance game.Appearance) error {
	for _, cosmetic := range []string{appearance.Primary, appearance.Secondary, appearance.TrailID, appearance.NoseID, appearance.DroneSkin, appearance.BadgeID} {
		if cosmetic == "" {
			continue
		}
		if !ownsCosmetic(account, cosmetic) {
			return fmt.Errorf("cosmetic %s is not owned", cosmetic)
		}
	}
	if appearance.HullID != "" && !account.Unlocks[appearance.HullID] {
		return fmt.Errorf("hull %s is not unlocked", appearance.HullID)
	}
	return nil
}

func validateLoadout(account game.Account, loadout game.Loadout) error {
	for _, unlock := range []string{loadout.WeaponID, loadout.AbilityID, loadout.HullID, loadout.DroneID} {
		if unlock == "" {
			continue
		}
		if !account.Unlocks[unlock] {
			return fmt.Errorf("loadout item %s is not unlocked", unlock)
		}
	}
	return nil
}

func (s *Session) acceptInput(input game.InputCommand) error {
	if input.Seq <= s.lastSeq {
		return fmt.Errorf("stale input sequence")
	}
	now := time.Now()
	if now.Sub(s.inputWindowStart) > time.Second {
		s.inputWindowStart = now
		s.inputCount = 0
	}
	s.inputCount++
	if s.inputCount > game.TickRate*3 {
		return fmt.Errorf("input rate exceeded")
	}
	s.lastSeq = input.Seq
	return nil
}

func ownsCosmetic(account game.Account, cosmetic string) bool {
	for _, owned := range account.Cosmetics {
		if owned == cosmetic {
			return true
		}
	}
	return false
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
				_ = ss.write(ctx, protocol.ServerMessage{Type: "snapshot", Snapshot: &snap})
				if snap.Phase == game.PhaseComplete || snap.Phase == game.PhaseFailed {
					ss.account.CurrentRun = ""
					_ = h.store.SaveAccount(ctx, ss.account)
				}
			}
		}
	}
}

func (h *Hub) session(id game.PlayerID) *Session {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.players[id]
}

func (h *Hub) broadcast(msg protocol.ServerMessage) {
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

func (s *Session) write(ctx context.Context, msg protocol.ServerMessage) error {
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
