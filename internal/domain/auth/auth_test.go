package auth

import (
	"testing"
	"time"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr bool
	}{
		{
			name: "Valid User",
			user: User{
				ID:     "u-1",
				Email:  "test@example.com",
				Role:   RoleUser,
				Status: StatusActive,
			},
			wantErr: false,
		},
		{
			name: "Missing Email",
			user: User{
				ID:     "u-1",
				Role:   RoleUser,
				Status: StatusActive,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.user.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUser_IsActive(t *testing.T) {
	u := User{Status: StatusActive}
	if !u.IsActive() {
		t.Error("expected active")
	}
	u.Status = StatusDisabled
	if u.IsActive() {
		t.Error("expected not active")
	}
}

func TestSession_Active(t *testing.T) {
	now := time.Now()
	s := Session{
		ExpiresAt: now.Add(time.Hour),
	}
	if !s.Active(now) {
		t.Error("expected active")
	}

	s.ExpiresAt = now.Add(-time.Hour)
	if s.Active(now) {
		t.Error("expected inactive due to expiry")
	}

	revoked := now.Add(-time.Minute)
	s.ExpiresAt = now.Add(time.Hour)
	s.RevokedAt = &revoked
	if s.Active(now) {
		t.Error("expected inactive due to revocation")
	}
}
