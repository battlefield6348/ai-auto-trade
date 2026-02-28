package db

import (
	"context"
	"testing"

	"ai-auto-trade/internal/infrastructure/config"
)

func TestConnect_Empty(t *testing.T) {
	ctx := context.Background()
	cfg := config.DBConfig{DSN: ""}
	db, err := Connect(ctx, cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if db != nil {
		t.Error("expected nil db for empty DSN")
	}
}
