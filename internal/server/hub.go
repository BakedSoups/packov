package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
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

	mu          sync.RWMutex
	runs        map[string]*game.RunState
	players     map[game.PlayerID]*Session
	market      map[string]game.MarketplaceListing
	chat        []game.ChatMessage
	settledRuns map[string]bool
	match       game.Matchmaker
	event       game.WorldEvent
	nextListing uint64
	nextChat    uint64
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
	chatWindowStart  time.Time
	chatCount        int
}

func NewHub(c *game.Catalog, store Persistence, log *slog.Logger) *Hub {
	return &Hub{
		catalog:     c,
		store:       store,
		log:         log,
		runs:        map[string]*game.RunState{},
		players:     map[game.PlayerID]*Session{},
		market:      map[string]game.MarketplaceListing{},
		settledRuns: map[string]bool{},
		event:       game.GenerateWorldEvent(c, time.Now()),
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
		chatLog := h.chatHistory()
		missions := game.DailyMissions(time.Now())
		if err := s.write(ctx, protocol.ServerMessage{Type: "hello", PlayerID: id, Account: &account, Catalog: h.catalog, WorldEvent: &ev, Missions: missions, ChatLog: chatLog}); err != nil {
			return err
		}
		if s.runID != "" {
			h.mu.RLock()
			run := h.runs[s.runID]
			h.mu.RUnlock()
			if run != nil {
				snap := run.SnapshotForPlayer(s.id)
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
		if _, ok := h.catalog.PlanetByID[planetID]; !ok {
			return fmt.Errorf("unknown planet %s", planetID)
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
		for _, t := range group {
			if ss := h.session(t.PlayerID); ss != nil {
				snap := run.SnapshotForPlayer(t.PlayerID)
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
	case "market_list":
		return s.write(ctx, protocol.ServerMessage{Type: "market", Listings: h.marketListings()})
	case "market_sell":
		if s.id == "" {
			return fmt.Errorf("authenticate first")
		}
		listing, err := h.createListing(ctx, s, msg.ItemID, msg.Quantity, msg.UnitPrice)
		if err != nil {
			return err
		}
		_ = s.write(ctx, protocol.ServerMessage{Type: "account", Account: &s.account})
		h.broadcast(protocol.ServerMessage{Type: "market", Listings: h.marketListings()})
		h.log.Info("market listing created", "listing", listing.ID, "item", listing.ItemID)
	case "market_buy":
		if s.id == "" {
			return fmt.Errorf("authenticate first")
		}
		if err := h.buyListing(ctx, s, msg.ListingID); err != nil {
			return err
		}
		_ = s.write(ctx, protocol.ServerMessage{Type: "account", Account: &s.account})
		h.broadcast(protocol.ServerMessage{Type: "market", Listings: h.marketListings()})
	case "market_cancel":
		if s.id == "" {
			return fmt.Errorf("authenticate first")
		}
		if err := h.cancelListing(ctx, s, msg.ListingID); err != nil {
			return err
		}
		_ = s.write(ctx, protocol.ServerMessage{Type: "account", Account: &s.account})
		h.broadcast(protocol.ServerMessage{Type: "market", Listings: h.marketListings()})
	case "chat":
		chat, err := h.createChatMessage(s, msg.Channel, msg.Body)
		if err != nil {
			return err
		}
		h.broadcast(protocol.ServerMessage{Type: "chat", Chat: &chat})
	}
	return nil
}

func (h *Hub) marketListings() []game.MarketplaceListing {
	h.mu.RLock()
	defer h.mu.RUnlock()
	listings := make([]game.MarketplaceListing, 0, len(h.market))
	for _, listing := range h.market {
		listings = append(listings, listing)
	}
	return listings
}

func (h *Hub) chatHistory() []game.ChatMessage {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]game.ChatMessage, len(h.chat))
	copy(out, h.chat)
	return out
}

func (h *Hub) createChatMessage(s *Session, channel, body string) (game.ChatMessage, error) {
	if s.id == "" {
		return game.ChatMessage{}, fmt.Errorf("authenticate first")
	}
	now := time.Now()
	if now.Sub(s.chatWindowStart) > 10*time.Second {
		s.chatWindowStart = now
		s.chatCount = 0
	}
	s.chatCount++
	if s.chatCount > 5 {
		return game.ChatMessage{}, fmt.Errorf("chat rate exceeded")
	}
	channel = strings.TrimSpace(channel)
	if channel == "" {
		channel = "global"
	}
	if channel != "global" && channel != "guild" && channel != "party" {
		return game.ChatMessage{}, fmt.Errorf("unsupported chat channel")
	}
	body = strings.Join(strings.Fields(strings.TrimSpace(body)), " ")
	if body == "" {
		return game.ChatMessage{}, fmt.Errorf("empty chat message")
	}
	if len(body) > 180 {
		return game.ChatMessage{}, fmt.Errorf("chat message too long")
	}
	h.mu.Lock()
	h.nextChat++
	chat := game.ChatMessage{
		ID:        fmt.Sprintf("chat-%06d", h.nextChat),
		Channel:   channel,
		SenderID:  s.id,
		Sender:    s.name,
		Body:      body,
		SentAtUTC: now.UTC(),
	}
	h.chat = append(h.chat, chat)
	if len(h.chat) > 50 {
		h.chat = h.chat[len(h.chat)-50:]
	}
	h.mu.Unlock()
	return chat, nil
}

func (h *Hub) createListing(ctx context.Context, s *Session, itemID string, quantity, unitPrice int) (game.MarketplaceListing, error) {
	if itemID == "" || quantity <= 0 || unitPrice <= 0 {
		return game.MarketplaceListing{}, fmt.Errorf("invalid listing")
	}
	if _, ok := h.catalog.LootByID[itemID]; !ok {
		return game.MarketplaceListing{}, fmt.Errorf("unknown market item")
	}
	if !s.account.Inventory.Remove(itemID, quantity) {
		return game.MarketplaceListing{}, fmt.Errorf("not enough inventory")
	}
	h.mu.Lock()
	h.nextListing++
	listing := game.MarketplaceListing{
		ID:        fmt.Sprintf("listing-%06d", h.nextListing),
		SellerID:  s.id,
		ItemID:    itemID,
		Quantity:  quantity,
		UnitPrice: unitPrice,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}
	h.market[listing.ID] = listing
	h.mu.Unlock()
	if err := h.store.SaveAccount(ctx, s.account); err != nil {
		return game.MarketplaceListing{}, err
	}
	return listing, nil
}

func (h *Hub) buyListing(ctx context.Context, buyer *Session, listingID string) error {
	h.mu.Lock()
	listing, ok := h.market[listingID]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("listing not found")
	}
	if listing.SellerID == buyer.id {
		h.mu.Unlock()
		return fmt.Errorf("cannot buy your own listing")
	}
	total := listing.Quantity * listing.UnitPrice
	if buyer.account.Credits < total {
		h.mu.Unlock()
		return fmt.Errorf("not enough credits")
	}
	delete(h.market, listingID)
	h.mu.Unlock()

	seller := h.session(listing.SellerID)
	var sellerAccount game.Account
	var err error
	if seller != nil {
		sellerAccount = seller.account
	} else {
		sellerAccount, err = h.store.LoadAccount(ctx, listing.SellerID, string(listing.SellerID))
		if err != nil {
			return err
		}
	}
	buyer.account.Credits -= total
	buyer.account.Inventory.Add(listing.ItemID, listing.Quantity)
	sellerAccount.Credits += total
	if seller != nil {
		seller.account = sellerAccount
		_ = seller.write(ctx, protocol.ServerMessage{Type: "account", Account: &seller.account})
	}
	if err := h.store.SaveAccount(ctx, buyer.account); err != nil {
		return err
	}
	return h.store.SaveAccount(ctx, sellerAccount)
}

func (h *Hub) cancelListing(ctx context.Context, s *Session, listingID string) error {
	h.mu.Lock()
	listing, ok := h.market[listingID]
	if !ok {
		h.mu.Unlock()
		return fmt.Errorf("listing not found")
	}
	if listing.SellerID != s.id {
		h.mu.Unlock()
		return fmt.Errorf("listing belongs to another seller")
	}
	delete(h.market, listingID)
	h.mu.Unlock()
	s.account.Inventory.Add(listing.ItemID, listing.Quantity)
	return h.store.SaveAccount(ctx, s.account)
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
		if snap.Phase == game.PhaseComplete || snap.Phase == game.PhaseFailed {
			h.settleRun(ctx, r)
		}
		for _, ps := range snap.Players {
			if ss := h.session(ps.ID); ss != nil && ss.runID == r.ID {
				playerSnap := r.SnapshotForPlayer(ps.ID)
				_ = ss.write(ctx, protocol.ServerMessage{Type: "snapshot", Snapshot: &playerSnap})
			}
		}
	}
}

func (h *Hub) settleRun(ctx context.Context, r *game.RunState) {
	h.mu.Lock()
	if h.settledRuns[r.ID] {
		h.mu.Unlock()
		return
	}
	h.settledRuns[r.ID] = true
	h.mu.Unlock()

	for _, ps := range r.Players {
		ss := h.session(ps.ID)
		var account game.Account
		var err error
		if ss != nil {
			account = ss.account
		} else {
			account, err = h.store.LoadAccount(ctx, ps.ID, string(ps.ID))
			if err != nil {
				continue
			}
		}
		if ps.Extracted && r.Phase == game.PhaseComplete {
			for item, count := range ps.Carried.Items {
				account.Inventory.Add(item, count)
			}
			account.Credits += ps.Stats.CreditsEarned
		}
		account.CurrentRun = ""
		if err := h.store.SaveAccount(ctx, account); err != nil {
			continue
		}
		if ss != nil {
			ss.account = account
			ss.runID = ""
			_ = ss.write(ctx, protocol.ServerMessage{Type: "account", Account: &ss.account})
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
