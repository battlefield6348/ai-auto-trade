package main

import (
    "flag"
    "fmt"
    "log"
    "os"

    "ai-auto-trade/internal/infrastructure/config"
    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
    cfgPath := flag.String("config", "config.yaml", "path to config file")
    migrationsPath := flag.String("dir", "db/migrations", "path to migrations directory")
    flag.Parse()

    cfg, err := config.Load(*cfgPath)
    if err != nil {
        log.Fatalf("讀取組態失敗: %v", err)
    }

    if cfg.DB.URL == "" {
        log.Fatal("config.db.url 未設定，無法執行 migration")
    }

    src := fmt.Sprintf("file://%s", *migrationsPath)
    m, err := migrate.New(src, cfg.DB.URL)
    if err != nil {
        log.Fatalf("初始化 migration 失敗: %v", err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        log.Fatalf("migration 執行失敗: %v", err)
    }

    fmt.Println("Migration 完成")
    os.Exit(0)
}
