package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"ai-auto-trade/internal/infrastructure/config"
	"ai-auto-trade/internal/infrastructure/db"
	httpapi "ai-auto-trade/internal/interface/http"
)

func main() {
	cfg, err := config.LoadFromFile("config.yaml")
	if err != nil {
		log.Fatalf("CRITICAL: load config failed: %v", err)
	}
	log.Printf("configuration loaded (HTTP_ADDR=%s)", cfg.HTTP.Addr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	log.Printf("testing database connection...")
	pool, err := db.Connect(ctx, cfg.DB)
	if err != nil {
		log.Printf("warning: database connection failed, falling back to in-memory store: %v", err)
	} else if pool == nil {
		log.Printf("no DB_DSN provided; running with in-memory store only")
	} else {
		defer pool.Close()
		log.Printf("database connected successfully")
	}

	// 檢查 web 目錄是否存在
	if _, err := os.Stat("web"); os.IsNotExist(err) {
		log.Printf("warning: 'web' directory not found in current directory")
	} else {
		log.Printf("'web' directory found")
	}

	apiServer := httpapi.NewServer(cfg, pool)
	addr := cfg.HTTP.Addr
	log.Printf("starting HTTP server on %s", addr)
	if err := http.ListenAndServe(addr, apiServer.Handler()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
