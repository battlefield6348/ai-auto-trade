package db

import (
	"context"
	"database/sql"
	"time"

	"ai-auto-trade/internal/infrastructure/config"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	// Verify responsiveness with a simple query
	var now time.Time
	if err := db.QueryRowContext(pingCtx, "SELECT NOW()").Scan(&now); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// ConnectGORM 建立 GORM PostgreSQL 連線。
func ConnectGORM(ctx context.Context, cfg config.DBConfig) (*gorm.DB, error) {
	if cfg.DSN == "" {
		return nil, nil
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), gormConfig)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(cfg.MaxIdleTime)

	return db, nil
}
