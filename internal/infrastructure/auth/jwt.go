package authinfra

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"ai-auto-trade/internal/domain/auth"

	"github.com/golang-jwt/jwt/v5"
)

// JWTIssuer 實作 TokenIssuer，產生/驗證 JWT access token。
type JWTIssuer struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	sessions   auth.SessionStore
	users      UserFinder
	now        func() time.Time
}

// NewJWTIssuer 建立 JWT 簽發器。
func NewJWTIssuer(secret string, accessTTL, refreshTTL time.Duration, sessions auth.SessionStore, users UserFinder) *JWTIssuer {
	return &JWTIssuer{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		sessions:   sessions,
		users:      users,
		now:        time.Now,
	}
}

// Claims 定義 access token 的 payload。
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// UserFinder 簡化 user repo 需求，僅用於 refresh 時查詢使用者。
type UserFinder interface {
	FindByID(ctx context.Context, id string) (auth.User, error)
}

// Issue 產生 access/refresh token 並儲存 session。
func (j *JWTIssuer) Issue(ctx context.Context, user auth.User, meta auth.TokenMeta) (auth.TokenPair, error) {
	return j.issueWithSession(ctx, user, meta)
}

// Refresh 使用 refresh token 重新簽發 token，並會輪替 session。
func (j *JWTIssuer) Refresh(ctx context.Context, token string) (auth.TokenPair, error) {
	if strings.TrimSpace(token) == "" {
		return auth.TokenPair{}, fmt.Errorf("refresh token required")
	}
	if j.sessions == nil {
		return auth.TokenPair{}, fmt.Errorf("session store not configured")
	}

	sess, err := j.sessions.GetSession(ctx, token)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("get session: %w", err)
	}
	now := j.now()
	if !sess.Active(now) {
		return auth.TokenPair{}, fmt.Errorf("session expired or revoked")
	}
	if err := j.sessions.RevokeSession(ctx, token); err != nil {
		return auth.TokenPair{}, fmt.Errorf("revoke session: %w", err)
	}

	user, err := j.users.FindByID(ctx, sess.UserID)
	if err != nil {
		return auth.TokenPair{}, fmt.Errorf("find user: %w", err)
	}
	if !user.IsActive() {
		return auth.TokenPair{}, fmt.Errorf("user disabled")
	}
	return j.issueWithSession(ctx, user, auth.TokenMeta{
		UserAgent: sess.UserAgent,
		IP:        sess.IPAddress,
	})
}

// RevokeRefresh 未實作（MVP 不用 refresh token）。
func (j *JWTIssuer) RevokeRefresh(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" || j.sessions == nil {
		return nil
	}
	return j.sessions.RevokeSession(ctx, token)
}

// ParseAccessToken 驗證並解析 access token，回傳 userID 與 role。
func (j *JWTIssuer) ParseAccessToken(token string) (Claims, error) {
	var claims Claims
	tkn, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return j.secret, nil
	})
	if err != nil {
		return Claims{}, err
	}
	if !tkn.Valid {
		return Claims{}, errors.New("invalid token")
	}
	return claims, nil
}

func (j *JWTIssuer) issueWithSession(ctx context.Context, user auth.User, meta auth.TokenMeta) (auth.TokenPair, error) {
	now := j.now()
	accessExp := now.Add(j.accessTTL)
	refreshExp := now.Add(j.refreshTTL)
	claims := Claims{
		UserID: user.ID,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secret)
	if err != nil {
		return auth.TokenPair{}, err
	}

	refreshToken, err := randomToken()
	if err != nil {
		return auth.TokenPair{}, err
	}
	if j.sessions != nil {
		_ = j.sessions.RevokeSession(ctx, refreshToken) // 保險清理同名 token
		if err := j.sessions.SaveSession(ctx, auth.Session{
			Token:     refreshToken,
			UserID:    user.ID,
			ExpiresAt: refreshExp,
			UserAgent: meta.UserAgent,
			IPAddress: meta.IP,
			CreatedAt: now,
		}); err != nil {
			return auth.TokenPair{}, err
		}
	}

	return auth.TokenPair{
		AccessToken:   signed,
		RefreshToken:  refreshToken,
		AccessExpiry:  accessExp,
		RefreshExpiry: refreshExp,
	}, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
