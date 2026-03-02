package postgres

import (
	"context"
	"database/sql"
	"testing"

	authDomain "ai-auto-trade/internal/domain/auth"

	"github.com/DATA-DOG/go-sqlmock"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestAuthRepo_FindByEmail(t *testing.T) {
	gormDB, mock, db := setupMock(t)
	defer db.Close()
	repo := NewAuthRepo(gormDB)

	// 1. Fetch user
	rows := sqlmock.NewRows([]string{"id", "email", "display_name", "password_hash", "status"}).
		AddRow("u-1", "test@example.com", "Test User", "hash", "active")

	mock.ExpectQuery("SELECT (.+) FROM (.+) WHERE email = (.+)").
		WillReturnRows(rows)

	// 2. Fetch role
	mock.ExpectQuery("SELECT roles.name FROM \"roles\" JOIN user_roles ur").
		WithArgs("u-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("admin"))

	u, err := repo.FindByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if u.ID != "u-1" || u.Role != authDomain.RoleAdmin {
		t.Errorf("unexpected user: %+v", u)
	}
}

func TestAuthRepo_FindByID(t *testing.T) {
	gormDB, mock, db := setupMock(t)
	defer db.Close()
	repo := NewAuthRepo(gormDB)

	// 1. Fetch user
	rows := sqlmock.NewRows([]string{"id", "email", "display_name", "password_hash", "status"}).
		AddRow("u-1", "test@example.com", "Test User", "hash", "active")

	mock.ExpectQuery("SELECT (.+) FROM (.+) WHERE id = (.+)").
		WillReturnRows(rows)

	// 2. Fetch role
	mock.ExpectQuery("SELECT roles.name FROM \"roles\" JOIN user_roles ur").
		WithArgs("u-1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("admin"))

	u, err := repo.FindByID(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if u.ID != "u-1" || u.Role != authDomain.RoleAdmin {
		t.Errorf("unexpected user: %+v", u)
	}
}

func TestAuthRepo_SaveSession(t *testing.T) {
	gormDB, mock, db := setupMock(t)
	defer db.Close()
	repo := NewAuthRepo(gormDB)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("s-1"))
	mock.ExpectCommit()

	sess := authDomain.Session{UserID: "u1"}
	err := repo.SaveSession(context.Background(), sess)
	if err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}
}

func TestAuthRepo_GetSession(t *testing.T) {
	gormDB, mock, db := setupMock(t)
	defer db.Close()
	repo := NewAuthRepo(gormDB)

	rows := sqlmock.NewRows([]string{"user_id", "refresh_token_id"}).AddRow("u-1", "t-1")
	mock.ExpectQuery("SELECT (.+) FROM (.+) WHERE (.+)").WillReturnRows(rows)

	sess, err := repo.GetSession(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if sess.UserID != "u-1" {
		t.Errorf("unexpected session: %+v", sess)
	}
}

func TestAuthRepo_RevokeSession(t *testing.T) {
	gormDB, mock, db := setupMock(t)
	defer db.Close()
	repo := NewAuthRepo(gormDB)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE (.+)").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.RevokeSession(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("RevokeSession failed: %v", err)
	}
}

func TestAuthRepo_Create(t *testing.T) {
	gormDB, mock, db := setupMock(t)
	defer db.Close()
	repo := NewAuthRepo(gormDB)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("u-99"))
	mock.ExpectQuery("SELECT (.+) FROM (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r-1"))
	mock.ExpectExec("INSERT INTO (.+)").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	u := authDomain.User{Email: "new@example.com"}
	id, err := repo.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if id != "u-99" {
		t.Errorf("expected u-99, got %s", id)
	}
}

func TestAuthRepo_IsPermissionGranted(t *testing.T) {
	gormDB, mock, db := setupMock(t)
	defer db.Close()
	repo := NewAuthRepo(gormDB)

	mock.ExpectQuery("SELECT (.+) FROM (.+)").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ok, err := repo.IsPermissionGranted(context.Background(), "u-1", "perm-1")
	if err != nil {
		t.Fatalf("IsPermissionGranted failed: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}
}

func TestAuthRepo_SeedPermissions(t *testing.T) {
	gormDB, mock, db := setupMock(t)
	defer db.Close()
	repo := NewAuthRepo(gormDB)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("p1"))
	mock.ExpectQuery("SELECT (.+) FROM (.+)").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("r1", "admin"))
	mock.ExpectExec("INSERT INTO (.+)").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := repo.SeedPermissions(context.Background(), []string{"p1"}, map[authDomain.Role][]string{authDomain.RoleAdmin: {"p1"}})
	if err != nil {
		t.Fatalf("SeedPermissions failed: %v", err)
	}
}

func TestAuthRepo_SeedDefaults(t *testing.T) {
	gormDB, mock, db := setupMock(t)
	defer db.Close()
	repo := NewAuthRepo(gormDB)

	mock.ExpectBegin()
	// multiple roles
	mock.ExpectQuery("INSERT INTO (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r1"))
	mock.ExpectQuery("INSERT INTO (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r2"))
	mock.ExpectQuery("INSERT INTO (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r3"))
	mock.ExpectQuery("INSERT INTO (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r4"))

	// multiple users
	for i := 0; i < 3; i++ {
		mock.ExpectQuery("INSERT INTO (.+)").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("u"))
		mock.ExpectExec("INSERT INTO (.+)").WillReturnResult(sqlmock.NewResult(1, 1))
	}

	mock.ExpectCommit()

	err := repo.SeedDefaults(context.Background())
	if err != nil {
		t.Fatalf("SeedDefaults failed: %v", err)
	}
}
