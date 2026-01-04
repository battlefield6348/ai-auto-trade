package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ai-auto-trade/internal/infrastructure/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to config file")
	migrationsPath := flag.String("dir", "db/migrations", "path to migrations directory")
	flag.Parse()

	cfg, err := config.LoadFromFile(*cfgPath)
	if err != nil {
		log.Fatalf("讀取組態失敗: %v", err)
	}

	if cfg.DB.DSN == "" {
		log.Fatal("config.db.dsn 未設定，無法執行 migration")
	}

	absDir, err := filepath.Abs(*migrationsPath)
	if err != nil {
		log.Fatalf("解析 migrations 路徑失敗: %v", err)
	}
	if _, err := os.Stat(absDir); err != nil {
		log.Fatalf("migrations 目錄不存在: %v", err)
	}

	src := fmt.Sprintf("file://%s", absDir)
	m, err := migrate.New(src, cfg.DB.DSN)
	if err != nil {
		log.Fatalf("初始化 migration 失敗: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration 執行失敗: %v", err)
	}

	fmt.Println("Migration 完成")
	os.Exit(0)
}
