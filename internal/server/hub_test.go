package server

import (
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
