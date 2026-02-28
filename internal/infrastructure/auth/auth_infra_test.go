package authinfra

import (
	"context"
	"fmt"
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

func TestJWTIssuer_Refresh(t *testing.T) {
	store := &mockSessionStore{}
	finder := &mockUserFinder{}
	issuer := NewJWTIssuer("secret", time.Hour, time.Hour*24, store, finder)
	user := auth.User{ID: "u-1", Role: auth.RoleAdmin, Status: auth.StatusActive}

	pair, _ := issuer.Issue(context.Background(), user, auth.TokenMeta{})

	// Success refresh
	newPair, err := issuer.Refresh(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if newPair.AccessToken == "" || newPair.RefreshToken == "" {
		t.Error("new tokens should not be empty")
	}

	// Error: empty token
	if _, err := issuer.Refresh(context.Background(), ""); err == nil {
		t.Error("expected error for empty token")
	}

	// Error: expired session
	store.sess.ExpiresAt = time.Now().Add(-time.Hour)
	if _, err := issuer.Refresh(context.Background(), pair.RefreshToken); err == nil {
		t.Error("expected error for expired session")
	}
}

func TestJWTIssuer_RevokeRefresh(t *testing.T) {
	store := &mockSessionStore{}
	issuer := NewJWTIssuer("secret", time.Hour, time.Hour*24, store, &mockUserFinder{})
	_ = issuer.RevokeRefresh(context.Background(), "token-1")
}

func TestJWTIssuer_RefreshErrors(t *testing.T) {
	ctx := context.Background()
	
	// Session store nil
	issuerNoStore := NewJWTIssuer("secret", time.Hour, time.Hour, nil, nil)
	if _, err := issuerNoStore.Refresh(ctx, "t"); err == nil {
		t.Error("expected error for nil store")
	}

	// RevokeRefresh with nil store or empty token
	_ = issuerNoStore.RevokeRefresh(ctx, "")
	_ = issuerNoStore.RevokeRefresh(ctx, "t")

	// User disabled or not found
	store := &mockSessionStore{sess: auth.Session{UserID: "bad-user", ExpiresAt: time.Now().Add(time.Hour)}}
	finder := &mockUserFinderErr{}
	issuer := NewJWTIssuer("secret", time.Hour, time.Hour, store, finder)
	
	if _, err := issuer.Refresh(ctx, "t"); err == nil {
		t.Error("expected error for bad user")
	}
}

type mockUserFinderErr struct{}

func (m *mockUserFinderErr) FindByID(ctx context.Context, id string) (auth.User, error) {
	if id == "disabled" {
		return auth.User{ID: id, Role: auth.RoleAdmin, Status: auth.StatusDisabled}, nil
	}
	return auth.User{}, fmt.Errorf("user not found")
}

func TestJWTIssuer_RefreshDisabled(t *testing.T) {
	store := &mockSessionStore{sess: auth.Session{UserID: "disabled", ExpiresAt: time.Now().Add(time.Hour)}}
	issuer := NewJWTIssuer("secret", time.Hour, time.Hour, store, &mockUserFinderErr{})
	if _, err := issuer.Refresh(context.Background(), "t"); err == nil {
		t.Error("expected error for disabled user")
	}
}

func TestJWTIssuer_ParseErrors(t *testing.T) {
	issuer := NewJWTIssuer("secret", time.Hour, time.Hour*24, nil, nil)
	
	// Invalid signing method (simulated by some random junk or an invalid token)
	if _, err := issuer.ParseAccessToken("not.a.token"); err == nil {
		t.Error("expected error for invalid token format")
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
