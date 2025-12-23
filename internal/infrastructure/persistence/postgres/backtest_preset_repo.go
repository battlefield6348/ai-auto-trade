package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type BacktestPreset struct {
	UserID string
	Config json.RawMessage
}

type BacktestPresetStore struct {
	db *sql.DB
}

func NewBacktestPresetStore(db *sql.DB) *BacktestPresetStore {
	return &BacktestPresetStore{db: db}
}

func (s *BacktestPresetStore) Save(ctx context.Context, userID string, config []byte) error {
	const q = `
INSERT INTO backtest_presets (user_id, config, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (user_id) DO UPDATE SET config = EXCLUDED.config, updated_at = NOW();
`
	_, err := s.db.ExecContext(ctx, q, userID, config)
	return err
}

func (s *BacktestPresetStore) Load(ctx context.Context, userID string) ([]byte, error) {
	const q = `
SELECT user_id, config, updated_at FROM backtest_presets WHERE user_id = $1 LIMIT 1;
`
	var (
		uID string
		cfg []byte
	)
	if err := s.db.QueryRowContext(ctx, q, userID).Scan(&uID, &cfg, new(time.Time)); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *BacktestPresetStore) SeedDefaults(ctx context.Context) error {
	// no-op
	return nil
}

// NotFound 判斷是否為未找到錯誤。
func (s *BacktestPresetStore) NotFound(err error) bool {
	return err == sql.ErrNoRows
}
