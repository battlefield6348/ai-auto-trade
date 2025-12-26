package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type BacktestPreset struct {
	ID        string
	UserID    string
	Name      string
	Config    json.RawMessage
	CreatedAt time.Time
	UpdatedAt time.Time
}

type BacktestPresetStore struct {
	db *sql.DB
}

func NewBacktestPresetStore(db *sql.DB) *BacktestPresetStore {
	return &BacktestPresetStore{db: db}
}

func (s *BacktestPresetStore) Save(ctx context.Context, userID string, config []byte) error {
	_, err := s.SaveNamed(ctx, userID, "default", config)
	return err
}

func (s *BacktestPresetStore) SaveNamed(ctx context.Context, userID, name string, config []byte) (string, error) {
	const q = `
INSERT INTO backtest_presets (user_id, name, config, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
ON CONFLICT (user_id, name) DO UPDATE SET config = EXCLUDED.config, updated_at = NOW()
RETURNING id;
`
	var id string
	if err := s.db.QueryRowContext(ctx, q, userID, name, config).Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (s *BacktestPresetStore) Load(ctx context.Context, userID string) ([]byte, error) {
	const q = `
SELECT user_id, config, updated_at FROM backtest_presets WHERE user_id = $1 ORDER BY updated_at DESC LIMIT 1;
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

func (s *BacktestPresetStore) List(ctx context.Context, userID string) ([]BacktestPreset, error) {
	const q = `
SELECT id, user_id, name, config, created_at, updated_at
FROM backtest_presets
WHERE user_id = $1
ORDER BY updated_at DESC;
`
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BacktestPreset
	for rows.Next() {
		var r BacktestPreset
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.Config, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *BacktestPresetStore) Delete(ctx context.Context, userID, id string) error {
	const q = `DELETE FROM backtest_presets WHERE id = $1 AND user_id = $2;`
	res, err := s.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *BacktestPresetStore) SeedDefaults(ctx context.Context) error {
	// no-op
	return nil
}

// NotFound 判斷是否為未找到錯誤。
func (s *BacktestPresetStore) NotFound(err error) bool {
	return err == sql.ErrNoRows
}
