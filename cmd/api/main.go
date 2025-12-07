package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"ai-auto-trade/internal/infrastructure/config"
	"ai-auto-trade/internal/infrastructure/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DB)
	if err != nil {
		log.Printf("warning: database connection failed, falling back to in-memory store: %v", err)
	} else if pool == nil {
		log.Printf("no DB_DSN provided; running with in-memory store only")
	} else {
		defer pool.Close()
		log.Printf("database connected")
	}

	srv := newServer(cfg, pool)
	addr := cfg.HTTP.Addr
	log.Printf("starting HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, srv.routes()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
