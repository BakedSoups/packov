package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"packov/internal/game"
	"packov/internal/server"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	catalogPath := getenv("PACKOV_CONTENT", "content/game.json")
	catalog, err := game.LoadCatalog(catalogPath)
	if err != nil {
		log.Error("load catalog", "err", err)
		os.Exit(1)
	}

	store := server.NewSpaceTimeDBAdapter(getenv("SPACETIMEDB_ENDPOINT", "http://spacetimedb:3000"), getenv("SPACETIMEDB_DATABASE", "packov"))
	hub := server.NewHub(catalog, store, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go hub.Run(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "time": time.Now().UTC()})
	})
	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(catalog)
	})
	mux.Handle("/", http.FileServer(http.Dir("web")))

	addr := getenv("ADDR", ":8080")
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Info("packov server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server", "err", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
