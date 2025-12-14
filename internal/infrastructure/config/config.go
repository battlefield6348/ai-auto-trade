package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 儲存 HTTP API 及外部相依的執行設定。
type Config struct {
	HTTP      HTTPConfig      `yaml:"http"`
	DB        DBConfig        `yaml:"db"`
	Auth      AuthConfig      `yaml:"auth"`
	Ingestion IngestionConfig `yaml:"ingestion"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

type DBConfig struct {
	DSN          string        `yaml:"dsn"`
	MaxOpenConns int           `yaml:"max_open_conns"`
	MaxIdleConns int           `yaml:"max_idle_conns"`
	MaxIdleTime  time.Duration `yaml:"max_idle_time"`
}

type AuthConfig struct {
	TokenTTL time.Duration `yaml:"token_ttl"`
	Secret   string        `yaml:"secret"`
}

type IngestionConfig struct {
	UseSynthetic bool `yaml:"use_synthetic"`
}

// LoadFromFile 從 YAML 組態檔載入設定。
func LoadFromFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config yaml: %w", err)
	}
	return applyDefaults(cfg), nil
}

func applyDefaults(cfg Config) Config {
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8080"
	}
	if cfg.DB.MaxOpenConns == 0 {
		cfg.DB.MaxOpenConns = 5
	}
	if cfg.DB.MaxIdleConns == 0 {
		cfg.DB.MaxIdleConns = 2
	}
	if cfg.DB.MaxIdleTime == 0 {
		cfg.DB.MaxIdleTime = 15 * time.Minute
	}
	if cfg.Auth.TokenTTL == 0 {
		cfg.Auth.TokenTTL = 30 * time.Minute
	}
	if cfg.Auth.Secret == "" {
		cfg.Auth.Secret = "dev-secret-change-me"
	}
	return cfg
}
