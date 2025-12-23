package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "ai-auto-trade/internal/domain/auth"
)

type fakeUserRepo struct {
	user domain.User
	err  error
}

func (f fakeUserRepo) FindByEmail(_ context.Context, _ string) (domain.User, error) {
	if f.err != nil {
		return domain.User{}, f.err
	}
	return f.user, nil
}

func (f fakeUserRepo) FindByID(_ context.Context, _ string) (domain.User, error) {
	if f.err != nil {
		return domain.User{}, f.err
	}
	return f.user, nil
}

type fakeHasher struct {
	match bool
}

func (f fakeHasher) Compare(_, _ string) bool { return f.match }

type fakeTokens struct {
	pair    domain.TokenPair
	err     error
	revoked string
}

func (f *fakeTokens) Issue(_ context.Context, _ domain.User, _ domain.TokenMeta) (domain.TokenPair, error) {
	if f.err != nil {
		return domain.TokenPair{}, f.err
	}
	return f.pair, nil
}

func (f *fakeTokens) Refresh(_ context.Context, _ string) (domain.TokenPair, error) {
	if f.err != nil {
		return domain.TokenPair{}, f.err
	}
	return f.pair, nil
}

func (f *fakeTokens) RevokeRefresh(_ context.Context, token string) error {
	f.revoked = token
	return f.err
}

func TestLoginSuccess(t *testing.T) {
	user := domain.User{
		ID:       "u1",
		Email:    "user@example.com",
		Role:     domain.RoleAnalyst,
		Status:   domain.StatusActive,
		Password: "hashed",
	}
	tokens := &fakeTokens{pair: domain.TokenPair{
		AccessToken:   "access",
		RefreshToken:  "refresh",
		AccessExpiry:  time.Now().Add(time.Minute),
		RefreshExpiry: time.Now().Add(time.Hour),
	}}
	uc := NewLoginUseCase(fakeUserRepo{user: user}, fakeHasher{match: true}, tokens)
	res, err := uc.Execute(context.Background(), LoginInput{
		Email:    "user@example.com",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Token.AccessToken != "access" || res.Token.RefreshToken != "refresh" {
		t.Fatalf("unexpected token: %+v", res.Token)
	}
}

func TestLoginFailsOnStatusOrPassword(t *testing.T) {
	user := domain.User{
		ID:       "u1",
		Email:    "user@example.com",
		Role:     domain.RoleUser,
		Status:   domain.StatusDisabled,
		Password: "hashed",
	}
	uc := NewLoginUseCase(fakeUserRepo{user: user}, fakeHasher{match: false}, &fakeTokens{})

	if _, err := uc.Execute(context.Background(), LoginInput{Email: "user@example.com", Password: "x"}); err == nil {
		t.Fatalf("expected error for disabled user")
	}
	user.Status = domain.StatusActive
	uc = NewLoginUseCase(fakeUserRepo{user: user}, fakeHasher{match: false}, &fakeTokens{})
	if _, err := uc.Execute(context.Background(), LoginInput{Email: "user@example.com", Password: "x"}); err == nil {
		t.Fatalf("expected error for wrong password")
	}
}

func TestLogoutRevokesRefresh(t *testing.T) {
	tokens := &fakeTokens{}
	uc := NewLogoutUseCase(tokens)
	if err := uc.Execute(context.Background(), "refresh-token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.revoked != "refresh-token" {
		t.Fatalf("expected token revoked")
	}
}

func TestAuthorizeRolePermission(t *testing.T) {
	authz := NewAuthorizer(fakeUserRepo{user: domain.User{ID: "u1", Role: domain.RoleAdmin, Status: domain.StatusActive}}, nil)
	if !authz.HasPermission(domain.RoleAdmin, PermUserManage) {
		t.Fatalf("admin should have user manage")
	}
	if authz.HasPermission(domain.RoleUser, PermUserManage) {
		t.Fatalf("user should not have user manage")
	}

	res, err := authz.Authorize(context.Background(), AuthorizeInput{
		UserID:   "u1",
		Required: []Permission{PermUserManage},
	})
	if err != nil || !res.Allowed {
		t.Fatalf("expected authorize success, got %+v err=%v", res, err)
	}
}

type fakeOwner struct {
	owned bool
}

func (f fakeOwner) IsOwner(_ context.Context, _ string, _ string) bool { return f.owned }

func TestAuthorizeOwnerFallback(t *testing.T) {
	authz := NewAuthorizer(
		fakeUserRepo{user: domain.User{ID: "u1", Role: domain.RoleUser, Status: domain.StatusActive}},
		fakeOwner{owned: true},
	)
	res, err := authz.Authorize(context.Background(), AuthorizeInput{
		UserID:     "u1",
		Required:   []Permission{PermUserManage},
		ResourceID: "res1",
		OwnerPerm:  PermUserManage,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Allowed {
		t.Fatalf("expected allowed due to owner fallback")
	}
}

func TestLoginErrorFromRepo(t *testing.T) {
	uc := NewLoginUseCase(fakeUserRepo{err: errors.New("db down")}, fakeHasher{}, &fakeTokens{})
	if _, err := uc.Execute(context.Background(), LoginInput{Email: "a", Password: "b"}); err == nil {
		t.Fatalf("expected error from repo")
	}
}
