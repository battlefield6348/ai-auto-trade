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
func TestConfig_ApplyEnvFull(t *testing.T) {
	os.Setenv("PORT", "3000")
	os.Setenv("DB_DSN", "postgres://...")
	os.Setenv("TELEGRAM_CHAT_ID", "12345")
	os.Setenv("TELEGRAM_ENABLED", "true")
	os.Setenv("BINANCE_USE_TESTNET", "true")
	os.Setenv("AUTO_INTERVAL", "5m")
	
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DB_DSN")
		os.Unsetenv("TELEGRAM_CHAT_ID")
		os.Unsetenv("TELEGRAM_ENABLED")
		os.Unsetenv("BINANCE_USE_TESTNET")
		os.Unsetenv("AUTO_INTERVAL")
	}()

	cfg := Config{}
	cfg = applyEnv(cfg)

	if cfg.HTTP.Addr != ":3000" {
		t.Errorf("expected :3000, got %s", cfg.HTTP.Addr)
	}
	if cfg.DB.DSN != "postgres://..." {
		t.Error("DSN mismatch")
	}
	if cfg.Notifier.Telegram.ChatID != 12345 {
		t.Error("ChatID mismatch")
	}
	if !cfg.Notifier.Telegram.Enabled {
		t.Error("Enabled mismatch")
	}
	if !cfg.Binance.UseTestnet {
		t.Error("Testnet mismatch")
	}
	if cfg.Ingestion.AutoInterval.Minutes() != 5 {
		t.Error("Interval mismatch")
	}
}

func TestLoadFromFile(t *testing.T) {
	// 1. Non-existent file (should use defaults)
	cfg, err := LoadFromFile("no-such-file.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Error("default failed")
	}

	// 2. YAML file
	tmp := "test_config.yaml"
	os.WriteFile(tmp, []byte("http:\n  addr: \":7777\"\n"), 0644)
	defer os.Remove(tmp)

	cfg, err = LoadFromFile(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Addr != ":7777" {
		t.Errorf("got %s", cfg.HTTP.Addr)
	}
}
