package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"ai-auto-trade/internal/infrastructure/config"
	_ "github.com/lib/pq"
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

	files, err := filepath.Glob(filepath.Join(absDir, "*.sql"))
	if err != nil {
		log.Fatalf("讀取 migrations 失敗: %v", err)
	}
	if len(files) == 0 {
		log.Fatal("找不到任何 .sql migration 檔案")
	}
	sort.Strings(files)

	db, err := sql.Open("postgres", cfg.DB.DSN)
	if err != nil {
		log.Fatalf("連線資料庫失敗: %v", err)
	}
	defer db.Close()

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("讀取檔案 %s 失敗: %v", f, err)
		}
		log.Printf("執行 migration: %s", filepath.Base(f))
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			log.Fatalf("執行 %s 失敗: %v", filepath.Base(f), err)
		}
	}

	fmt.Println("Migration 完成")
	os.Exit(0)
}
