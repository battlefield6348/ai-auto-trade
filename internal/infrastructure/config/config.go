package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 儲存 HTTP API 及外部相依的執行設定。
type Config struct {
	HTTP HTTPConfig
	DB   DBConfig
	Auth AuthConfig
}

type HTTPConfig struct {
	Addr string
}

type DBConfig struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
	MaxIdleTime  time.Duration
}

type AuthConfig struct {
	TokenTTL time.Duration
}

// Load 從環境變數載入設定，缺值時使用預設值。
func Load() (Config, error) {
	var cfg Config

	cfg.HTTP.Addr = env("HTTP_ADDR", ":8080")

	cfg.DB.DSN = firstNonEmpty(os.Getenv("DB_DSN"), os.Getenv("DATABASE_URL"))
	var err error
	cfg.DB.MaxOpenConns, err = envInt("DB_MAX_OPEN_CONNS", 5)
	if err != nil {
		return cfg, fmt.Errorf("DB_MAX_OPEN_CONNS: %w", err)
	}
	cfg.DB.MaxIdleConns, err = envInt("DB_MAX_IDLE_CONNS", 2)
	if err != nil {
		return cfg, fmt.Errorf("DB_MAX_IDLE_CONNS: %w", err)
	}
	cfg.DB.MaxIdleTime, err = envDuration("DB_MAX_IDLE_TIME", 15*time.Minute)
	if err != nil {
		return cfg, fmt.Errorf("DB_MAX_IDLE_TIME: %w", err)
	}

	cfg.Auth.TokenTTL, err = envDuration("AUTH_TOKEN_TTL", 30*time.Minute)
	if err != nil {
		return cfg, fmt.Errorf("AUTH_TOKEN_TTL: %w", err)
	}

	return cfg, nil
}

func env(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func envInt(key string, def int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return def, nil
	}
	out, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return out, nil
}

func envDuration(key string, def time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return def, nil
	}
	return time.ParseDuration(val)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
