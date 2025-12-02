package auth

import "time"

// TokenPair 封裝 access/refresh token。
type TokenPair struct {
	AccessToken   string
	RefreshToken  string
	AccessExpiry  time.Time
	RefreshExpiry time.Time
}
