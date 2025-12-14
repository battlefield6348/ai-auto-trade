package authinfra

import (
	"context"
	"errors"
	"time"

	"ai-auto-trade/internal/domain/auth"

	"github.com/golang-jwt/jwt/v5"
)

// JWTIssuer 實作 TokenIssuer，產生/驗證 JWT access token。
type JWTIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewJWTIssuer 建立 JWT 簽發器。
func NewJWTIssuer(secret string, ttl time.Duration) *JWTIssuer {
	return &JWTIssuer{
		secret: []byte(secret),
		ttl:    ttl,
		now:    time.Now,
	}
}

// Claims 定義 access token 的 payload。
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Issue 產生 access token；refresh token 未實作。
func (j *JWTIssuer) Issue(ctx context.Context, user auth.User) (auth.TokenPair, error) {
	now := j.now()
	exp := now.Add(j.ttl)
	claims := Claims{
		UserID: user.ID,
		Role:   string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secret)
	if err != nil {
		return auth.TokenPair{}, err
	}
	return auth.TokenPair{
		AccessToken:   signed,
		RefreshToken:  "",
		AccessExpiry:  exp,
		RefreshExpiry: exp,
	}, nil
}

// RevokeRefresh 未實作（MVP 不用 refresh token）。
func (j *JWTIssuer) RevokeRefresh(ctx context.Context, token string) error {
	return nil
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
