package authinfra

import (
	"context"
	"testing"
	"time"

	"ai-auto-trade/internal/domain/auth"
)

type mockSessionStore struct {
	sess auth.Session
}

func (m *mockSessionStore) SaveSession(ctx context.Context, sess auth.Session) error {
	m.sess = sess
	return nil
}
func (m *mockSessionStore) GetSession(ctx context.Context, token string) (auth.Session, error) {
	return m.sess, nil
}
func (m *mockSessionStore) RevokeSession(ctx context.Context, token string) error {
	return nil
}

type mockUserFinder struct{}

func (m *mockUserFinder) FindByID(ctx context.Context, id string) (auth.User, error) {
	return auth.User{ID: id, Role: auth.RoleAdmin, Status: auth.StatusActive}, nil
}

func TestJWTIssuer_IssueAndParse(t *testing.T) {
	issuer := NewJWTIssuer("secret", time.Hour, time.Hour*24, &mockSessionStore{}, &mockUserFinder{})
	user := auth.User{ID: "u-1", Role: auth.RoleAdmin}

	pair, err := issuer.Issue(context.Background(), user, auth.TokenMeta{})
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	claims, err := issuer.ParseAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ParseAccessToken failed: %v", err)
	}

	if claims.UserID != "u-1" || claims.Role != "admin" {
		t.Errorf("unexpected claims: %+v", claims)
	}
}

func TestBcryptHasher(t *testing.T) {
	h := BcryptHasher{}
	pwd := "password123"
	hashed, err := h.Hash(pwd)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if !h.Compare(hashed, pwd) {
		t.Error("Compare failed")
	}

	if h.Compare(hashed, "wrong") {
		t.Error("Compare should have failed")
	}
}
