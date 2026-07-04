package server

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"packov/internal/game"
)

func TestValidateLoadoutRequiresUnlocks(t *testing.T) {
	account := game.NewAccount("p1", "Pilot")
	if err := validateLoadout(account, game.DefaultLoadout()); err != nil {
		t.Fatalf("default loadout should validate: %v", err)
	}
	if err := validateLoadout(account, game.Loadout{WeaponID: "railgun", AbilityID: "dash", HullID: "hull_scout"}); err == nil {
		t.Fatal("locked railgun should be rejected")
	}
}

func TestAcceptInputRejectsStaleSequence(t *testing.T) {
	s := &Session{}
	if err := s.acceptInput(game.InputCommand{Seq: 10}); err != nil {
		t.Fatalf("first input should validate: %v", err)
	}
	if err := s.acceptInput(game.InputCommand{Seq: 10}); err == nil {
		t.Fatal("duplicate sequence should be rejected")
	}
	if err := s.acceptInput(game.InputCommand{Seq: 9}); err == nil {
		t.Fatal("older sequence should be rejected")
	}
}

func TestMarketplaceCreateBuyCancel(t *testing.T) {
	ctx := context.Background()
	catalog := game.DefaultCatalogForClient()
	store := NewMemoryStore()
	hub := NewHub(catalog, store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	sellerAccount := game.NewAccount("seller", "Seller")
	sellerAccount.Inventory.Add("alien_alloy", 2)
	seller := &Session{id: "seller", name: "Seller", account: sellerAccount}

	listing, err := hub.createListing(ctx, seller, "alien_alloy", 1, 25)
	if err != nil {
		t.Fatalf("create listing: %v", err)
	}
	if seller.account.Inventory.Items["alien_alloy"] != 1 {
		t.Fatalf("listing should escrow item, inventory: %+v", seller.account.Inventory.Items)
	}
	if len(hub.marketListings()) != 1 {
		t.Fatal("listing should be visible")
	}

	if err := hub.cancelListing(ctx, seller, listing.ID); err != nil {
		t.Fatalf("cancel listing: %v", err)
	}
	if seller.account.Inventory.Items["alien_alloy"] != 2 {
		t.Fatalf("cancel should return item, inventory: %+v", seller.account.Inventory.Items)
	}

	listing, err = hub.createListing(ctx, seller, "alien_alloy", 1, 25)
	if err != nil {
		t.Fatalf("recreate listing: %v", err)
	}
	buyerAccount := game.NewAccount("buyer", "Buyer")
	buyer := &Session{id: "buyer", name: "Buyer", account: buyerAccount}
	if err := hub.buyListing(ctx, buyer, listing.ID); err != nil {
		t.Fatalf("buy listing: %v", err)
	}
	if buyer.account.Inventory.Items["alien_alloy"] != 1 || buyer.account.Credits != 475 {
		t.Fatalf("buyer state mismatch: %+v", buyer.account)
	}
	if len(hub.marketListings()) != 0 {
		t.Fatal("bought listing should be removed")
	}
}

func TestSettleRunPersistsExtractedLootOnly(t *testing.T) {
	ctx := context.Background()
	catalog := game.DefaultCatalogForClient()
	store := NewMemoryStore()
	hub := NewHub(catalog, store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	account := game.NewAccount("p1", "Pilot")
	session := &Session{id: "p1", name: "Pilot", account: account, runID: "run"}
	hub.players["p1"] = session

	run := game.NewRun("run", catalog, "verdant", 1)
	run.AddPlayer("p1", "Pilot", game.DefaultLoadout())
	ps := run.Players["p1"]
	ps.Carried.Add("alien_alloy", 2)
	ps.Extracted = true
	ps.Stats.CreditsEarned = 77
	run.Phase = game.PhaseComplete

	hub.settleRun(ctx, run)
	if session.account.Inventory.Items["alien_alloy"] != 2 {
		t.Fatalf("expected extracted loot persisted, got %+v", session.account.Inventory.Items)
	}
	if session.account.Credits != account.Credits+77 {
		t.Fatalf("expected earned credits persisted, got %d", session.account.Credits)
	}

	failed := game.NewRun("failed", catalog, "verdant", 2)
	failed.AddPlayer("p1", "Pilot", game.DefaultLoadout())
	fps := failed.Players["p1"]
	fps.Carried.Add("alien_alloy", 5)
	fps.Downed = true
	failed.Phase = game.PhaseFailed
	hub.settleRun(ctx, failed)
	if session.account.Inventory.Items["alien_alloy"] != 2 {
		t.Fatalf("failed run should not persist carried loot, got %+v", session.account.Inventory.Items)
	}
}
