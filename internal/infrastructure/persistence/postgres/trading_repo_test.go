package postgres

import (
	"context"
	"database/sql"
	"testing"

	"ai-auto-trade/internal/application/trading"
	tradingDomain "ai-auto-trade/internal/domain/trading"

	"github.com/DATA-DOG/go-sqlmock"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTradingMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestCreateStrategy_UseCreatedBy(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO \"strategies\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("s1"))
	mock.ExpectCommit()

	s := tradingDomain.Strategy{Slug: "s1", CreatedBy: "u1"}
	id, err := repo.CreateStrategy(context.Background(), s)
	if err != nil || id != "s1" {
		t.Fatalf("failed: %v", err)
	}
}

func TestUpdateStrategy(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE \"strategies\"").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.UpdateStrategy(context.Background(), tradingDomain.Strategy{ID: "s1"})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
}

func TestGetStrategy(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	rows := sqlmock.NewRows([]string{"id", "slug", "name"}).AddRow("s1", "slug1", "name1")
	mock.ExpectQuery("SELECT (.+) FROM \"strategies\" WHERE id = (.+)").WillReturnRows(rows)

	s, err := repo.GetStrategy(context.Background(), "s1")
	if err != nil || s.ID != "s1" {
		t.Fatalf("failed: %v", err)
	}
}

func TestDeleteStrategy(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM \"strategies\"").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.DeleteStrategy(context.Background(), "s1")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
}

func TestListStrategies(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	mock.ExpectQuery("SELECT (.+) FROM \"strategies\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("s1"))

	res, err := repo.ListStrategies(context.Background(), trading.StrategyFilter{})
	if err != nil || len(res) != 1 {
		t.Fatalf("failed: %v", err)
	}
}

func TestSaveTrade(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO \"strategy_trades\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("t1"))
	mock.ExpectCommit()

	err := repo.SaveTrade(context.Background(), tradingDomain.TradeRecord{ID: "t1"})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
}

func TestGetOpenPosition(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	mock.ExpectQuery("SELECT (.+) FROM \"strategy_positions\" WHERE (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("p1"))

	p, err := repo.GetOpenPosition(context.Background(), "s1", "paper")
	if err != nil || p.ID != "p1" {
		t.Fatalf("failed: %v", err)
	}
}

func TestUpsertPosition(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO \"strategy_positions\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("p1"))
	mock.ExpectCommit()

	err := repo.UpsertPosition(context.Background(), tradingDomain.Position{ID: "p1"})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
}

func TestListTrades(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	mock.ExpectQuery("SELECT (.+) FROM \"strategy_trades\"").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("t1"))

	res, err := repo.ListTrades(context.Background(), tradingDomain.TradeFilter{})
	if err != nil || len(res) != 1 {
		t.Fatalf("failed: %v", err)
	}
}

func TestScoringStrategyMethods(t *testing.T) {
	gormDB, mock, db := setupTradingMock(t)
	defer db.Close()
	repo := NewTradingRepo(gormDB)

	// 1. Get slugs
	mock.ExpectQuery("SELECT \"slug\" FROM \"strategies\" WHERE is_active = (.+)").WillReturnRows(sqlmock.NewRows([]string{"slug"}).AddRow("slug1"))

	// 2. LoadScoringStrategyBySlugGORM
	mock.ExpectQuery("SELECT (.+) FROM \"strategies\" WHERE slug = (.+)").
		WillReturnRows(sqlmock.NewRows([]string{"id", "slug"}).AddRow("s1", "slug1"))
	
	// 3. Fetch Rules
	mock.ExpectQuery("SELECT (.+) FROM strategy_rules (.+) WHERE sr.strategy_id = (.+)").
		WillReturnRows(sqlmock.NewRows([]string{"strategy_id"}).AddRow("s1"))

	res, err := repo.ListActiveScoringStrategies(context.Background())
	if err != nil || len(res) != 1 {
		t.Fatalf("failed: %v", err)
	}
}
