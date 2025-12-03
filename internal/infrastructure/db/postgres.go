package db

import (
	"context"
	"database/sql"
	"time"

	"ai-auto-trade/internal/infrastructure/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connect 建立 PostgreSQL 連線池；若未設定 DSN 則回傳 nil。
func Connect(ctx context.Context, cfg config.DBConfig) (*sql.DB, error) {
	if cfg.DSN == "" {
		return nil, nil
	}

	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.MaxIdleTime)

	pingCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		pingCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
