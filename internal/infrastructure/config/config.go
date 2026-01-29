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
	Notifier  NotifierConfig  `yaml:"notifier"`
	Binance   BinanceConfig   `yaml:"binance"`
	AutoTrade AutoTradeConfig `yaml:"auto_trade"`
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
	TokenTTL   time.Duration `yaml:"token_ttl"`
	RefreshTTL time.Duration `yaml:"refresh_ttl"`
	Secret     string        `yaml:"secret"`
}

type IngestionConfig struct {
	UseSynthetic      bool          `yaml:"use_synthetic"`
	AutoInterval      time.Duration `yaml:"auto_interval"`
	BackfillStartDate string        `yaml:"backfill_start_date"`
}

type NotifierConfig struct {
	Telegram TelegramConfig `yaml:"telegram"`
}

type TelegramConfig struct {
	Enabled        bool          `yaml:"enabled"`
	Token          string        `yaml:"token"`
	ChatID         int64         `yaml:"chat_id"`
	Interval       time.Duration `yaml:"interval"`
	StrongLimit    int           `yaml:"strong_limit"`
	ScoreMin       float64       `yaml:"score_min"`
	VolumeRatioMin float64       `yaml:"volume_ratio_min"`
}

type BinanceConfig struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	UseTestnet bool   `yaml:"use_testnet"`
}

type AutoTradeConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
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
	if cfg.Auth.RefreshTTL == 0 {
		cfg.Auth.RefreshTTL = 24 * time.Hour * 30
	}
	if cfg.Auth.Secret == "" {
		cfg.Auth.Secret = "dev-secret-change-me"
	}
	if cfg.Ingestion.AutoInterval == 0 {
		cfg.Ingestion.AutoInterval = time.Hour
	}
	if cfg.Notifier.Telegram.Interval == 0 {
		cfg.Notifier.Telegram.Interval = time.Hour
	}
	if cfg.Notifier.Telegram.StrongLimit == 0 {
		cfg.Notifier.Telegram.StrongLimit = 5
	}
	if cfg.Notifier.Telegram.ScoreMin == 0 {
		cfg.Notifier.Telegram.ScoreMin = 70
	}
	if cfg.Notifier.Telegram.VolumeRatioMin == 0 {
		cfg.Notifier.Telegram.VolumeRatioMin = 1.5
	}
	return cfg
}
