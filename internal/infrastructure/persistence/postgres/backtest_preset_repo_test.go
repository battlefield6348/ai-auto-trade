package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBacktestPresetStore_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	store := NewBacktestPresetStore(db)

	mock.ExpectQuery("INSERT INTO backtest_presets").
		WithArgs("u-1", "default", []byte("{}")).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("p-1"))

	err = store.Save(context.Background(), "u-1", []byte("{}"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestBacktestPresetStore_Load(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	store := NewBacktestPresetStore(db)

	rows := sqlmock.NewRows([]string{"user_id", "config", "updated_at"}).
		AddRow("u-1", []byte("{}"), time.Now())

	mock.ExpectQuery("SELECT (.+) FROM backtest_presets").
		WithArgs("u-1").
		WillReturnRows(rows)

	cfg, err := store.Load(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if string(cfg) != "{}" {
		t.Errorf("expected {}, got %s", cfg)
	}
}
