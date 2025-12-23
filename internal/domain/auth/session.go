package auth

import (
	"context"
	"time"
)

// Session 紀錄 refresh token 與其生命週期。
type Session struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
	UserAgent string
	IPAddress string
	CreatedAt time.Time
}

// Active 檢查 session 是否仍可使用。
func (s Session) Active(now time.Time) bool {
	if s.ExpiresAt.Before(now) {
		return false
	}
	if s.RevokedAt != nil && !s.RevokedAt.IsZero() {
		return false
	}
	return true
}

// SessionStore 提供 refresh token 儲存/查詢/撤銷。
type SessionStore interface {
	SaveSession(ctx context.Context, sess Session) error
	GetSession(ctx context.Context, token string) (Session, error)
	RevokeSession(ctx context.Context, token string) error
}

// TokenMeta 可選的 token 生成附帶資訊。
type TokenMeta struct {
	UserAgent string
	IP        string
}
