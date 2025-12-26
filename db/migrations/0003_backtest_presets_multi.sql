-- 允許每位使用者儲存多組回測條件並加入名稱與時間戳
ALTER TABLE IF EXISTS backtest_presets
    DROP CONSTRAINT IF EXISTS backtest_presets_user_id_key;

ALTER TABLE IF EXISTS backtest_presets
    ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT 'default',
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

CREATE UNIQUE INDEX IF NOT EXISTS idx_backtest_presets_user_name ON backtest_presets(user_id, name);
CREATE INDEX IF NOT EXISTS idx_backtest_presets_user ON backtest_presets(user_id);
