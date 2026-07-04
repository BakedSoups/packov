package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"packov/internal/game"
)

func TestSpaceTimeDBAdapterCallsSaveAccountReducer(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotArgs []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotArgs); err != nil {
			t.Fatalf("decode reducer args: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	adapter := NewSpaceTimeDBAdapter(ts.URL, "packov")
	adapter.Token = "test-token"
	account := game.NewAccount("p1", "Pilot")
	account.Credits = 123
	if err := adapter.SaveAccount(context.Background(), account); err != nil {
		t.Fatalf("save account: %v", err)
	}
	if gotPath != "/v1/database/packov/call/save_player_account" {
		t.Fatalf("unexpected reducer path %s", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("missing auth header %q", gotAuth)
	}
	if len(gotArgs) != 1 || !strings.Contains(gotArgs[0], `"credits":123`) {
		t.Fatalf("unexpected reducer args %+v", gotArgs)
	}
	loaded, err := adapter.fallback.LoadAccount(context.Background(), "p1", "Pilot")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Credits != 123 {
		t.Fatalf("fallback should stay updated for local reads, got %+v", loaded)
	}
}

func TestSpaceTimeDBStrictModeReturnsReducerErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "missing reducer", http.StatusNotFound)
	}))
	defer ts.Close()

	adapter := NewSpaceTimeDBAdapter(ts.URL, "packov")
	adapter.Strict = true
	err := adapter.SaveAccount(context.Background(), game.NewAccount("p1", "Pilot"))
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Fatalf("expected strict reducer error, got %v", err)
	}
}
