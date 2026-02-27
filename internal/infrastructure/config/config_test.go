package config

import (
	"os"
	"testing"
)

func TestConfig_ApplyDefaults(t *testing.T) {
	cfg := Config{}
	cfg = applyDefaults(cfg)

	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("expected :8080, got %s", cfg.HTTP.Addr)
	}
	if cfg.Auth.TokenTTL.Minutes() != 30 {
		t.Errorf("expected 30m, got %v", cfg.Auth.TokenTTL)
	}
}

func TestConfig_ApplyEnv(t *testing.T) {
	os.Setenv("HTTP_ADDR", ":9090")
	defer os.Unsetenv("HTTP_ADDR")

	cfg := Config{}
	cfg = applyEnv(cfg)

	if cfg.HTTP.Addr != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.HTTP.Addr)
	}
}
