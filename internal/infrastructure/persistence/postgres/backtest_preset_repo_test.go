package postgres

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupPresetMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	gormDB, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm: %s", err)
	}
	return gormDB, mock, db
}

func TestBacktestPresetStore_Save(t *testing.T) {
	gormDB, mock, db := setupPresetMock(t)
	defer db.Close()

	store := NewBacktestPresetStore(gormDB)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("p-1"))
	mock.ExpectCommit()

	err := store.Save(context.Background(), "u-1", []byte("{}"))
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
}

func TestBacktestPresetStore_Load(t *testing.T) {
	gormDB, mock, db := setupPresetMock(t)
	defer db.Close()

	store := NewBacktestPresetStore(gormDB)

	rows := sqlmock.NewRows([]string{"id", "config"}).AddRow("p-1", []byte("{}"))
	mock.ExpectQuery("SELECT (.+) FROM (.+) WHERE (.+)").WillReturnRows(rows)

	res, err := store.Load(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(res) == 0 {
		t.Error("expected config back")
	}
}
