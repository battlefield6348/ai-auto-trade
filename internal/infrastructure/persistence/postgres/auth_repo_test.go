package postgres

import (
	"context"
	"testing"
	"time"

	authDomain "ai-auto-trade/internal/domain/auth"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAuthRepo_FindByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewAuthRepo(db)

	rows := sqlmock.NewRows([]string{"id", "email", "display_name", "password_hash", "status", "role_name"}).
		AddRow("u-1", "test@example.com", "Test User", "hash", "active", "admin")

	mock.ExpectQuery("SELECT (.+) FROM users").
		WithArgs("test@example.com").
		WillReturnRows(rows)

	u, err := repo.FindByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("FindByEmail failed: %v", err)
	}
	if u.ID != "u-1" || u.Role != authDomain.RoleAdmin {
		t.Errorf("unexpected user: %+v", u)
	}
}

func TestAuthRepo_SaveSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewAuthRepo(db)
	sess := authDomain.Session{
		UserID:    "u-1",
		Token:     "t-1",
		ExpiresAt: time.Now().Add(time.Hour),
		UserAgent: "UA",
		IPAddress: "127.0.0.1",
	}

	mock.ExpectExec("INSERT INTO auth_sessions").
		WithArgs(sess.UserID, sess.Token, sess.ExpiresAt, sess.UserAgent, sess.IPAddress).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SaveSession(context.Background(), sess)
	if err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}
}

func TestAuthRepo_GetSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewAuthRepo(db)

	rows := sqlmock.NewRows([]string{"user_id", "refresh_token_id", "expires_at", "revoked_at", "user_agent", "ip_address", "created_at"}).
		AddRow("u-1", "t-1", time.Now().Add(time.Hour), nil, "UA", "127.0.0.1", time.Now())

	mock.ExpectQuery("SELECT (.+) FROM auth_sessions").
		WithArgs("t-1").
		WillReturnRows(rows)

	sess, err := repo.GetSession(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if sess.UserID != "u-1" || sess.Token != "t-1" {
		t.Errorf("unexpected session: %+v", sess)
	}
}

func TestAuthRepo_RevokeSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewAuthRepo(db)

	mock.ExpectExec("UPDATE auth_sessions").
		WithArgs("t-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.RevokeSession(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("RevokeSession failed: %v", err)
	}
}

func TestAuthRepo_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewAuthRepo(db)
	u := authDomain.User{Email: "new@example.com", Name: "New", Role: authDomain.RoleUser, Password: "pwd"}

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO users").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("u-99"))
	mock.ExpectQuery("SELECT id FROM roles WHERE name = \\$1").WithArgs("user").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r-1"))
	mock.ExpectExec("INSERT INTO user_roles").WithArgs("u-99", "r-1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	id, err := repo.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if id != "u-99" {
		t.Errorf("expected u-99, got %s", id)
	}
}

func TestAuthRepo_IsPermissionGranted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()

	repo := NewAuthRepo(db)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("u-1", "perm-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	ok, err := repo.IsPermissionGranted(context.Background(), "u-1", "perm-1")
	if err != nil {
		t.Fatalf("IsPermissionGranted failed: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}
}
func TestAuthRepo_FindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()
	repo := NewAuthRepo(db)

	rows := sqlmock.NewRows([]string{"id", "email", "display_name", "password_hash", "status", "role_name"}).
		AddRow("u-1", "test@example.com", "Test User", "hash", "active", "admin")

	mock.ExpectQuery("SELECT (.+) FROM users u LEFT JOIN user_roles ur").
		WithArgs("u-1").
		WillReturnRows(rows)

	u, err := repo.FindByID(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if u.ID != "u-1" {
		t.Errorf("expected u-1, got %s", u.ID)
	}
}

func TestAuthRepo_SeedDefaults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()
	repo := NewAuthRepo(db)

	mock.ExpectBegin()
	// roles
	mock.ExpectQuery("INSERT INTO roles").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r1")) // admin
	mock.ExpectQuery("INSERT INTO roles").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r2")) // analyst
	mock.ExpectQuery("INSERT INTO roles").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r3")) // user
	mock.ExpectQuery("INSERT INTO roles").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("r4")) // service

	// 3 users
	for i := 0; i < 3; i++ {
		mock.ExpectQuery("INSERT INTO users").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("u"))
		mock.ExpectExec("INSERT INTO user_roles").WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectCommit()

	err = repo.SeedDefaults(context.Background())
	if err != nil {
		t.Fatalf("SeedDefaults failed: %v", err)
	}
}

func TestAuthRepo_SeedPermissions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %s", err)
	}
	defer db.Close()
	repo := NewAuthRepo(db)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO permissions").WithArgs("p1", "auto seeded perm p1").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("pid1"))
	mock.ExpectQuery("SELECT id, name FROM roles").WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("rid1", "admin"))
	mock.ExpectExec("INSERT INTO role_permissions").WithArgs("rid1", "pid1").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	rolePerms := map[authDomain.Role][]string{
		authDomain.RoleAdmin: {"p1"},
	}
	err = repo.SeedPermissions(context.Background(), []string{"p1"}, rolePerms)
	if err != nil {
		t.Fatalf("SeedPermissions failed: %v", err)
	}
}
