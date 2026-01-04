-- Align strategies table with trading service schema (base_symbol/env/version/buy/sell/risk)

DO $$
BEGIN
    -- 新增必要欄位（若不存在）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'base_symbol'
    ) THEN
        ALTER TABLE strategies ADD COLUMN base_symbol VARCHAR(64) NOT NULL DEFAULT 'BTCUSDT';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'timeframe'
    ) THEN
        ALTER TABLE strategies ADD COLUMN timeframe VARCHAR(32) NOT NULL DEFAULT '1d';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'env'
    ) THEN
        ALTER TABLE strategies ADD COLUMN env VARCHAR(16) NOT NULL DEFAULT 'both';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'status'
    ) THEN
        ALTER TABLE strategies ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'draft';
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'version'
    ) THEN
        ALTER TABLE strategies ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'buy_conditions'
    ) THEN
        ALTER TABLE strategies ADD COLUMN buy_conditions JSONB NOT NULL DEFAULT '{}'::jsonb;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'sell_conditions'
    ) THEN
        ALTER TABLE strategies ADD COLUMN sell_conditions JSONB NOT NULL DEFAULT '{}'::jsonb;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'risk_settings'
    ) THEN
        ALTER TABLE strategies ADD COLUMN risk_settings JSONB NOT NULL DEFAULT '{}'::jsonb;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'created_by'
    ) THEN
        ALTER TABLE strategies ADD COLUMN created_by UUID;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'strategies' AND column_name = 'updated_by'
    ) THEN
        ALTER TABLE strategies ADD COLUMN updated_by UUID;
    END IF;
END $$;

-- 建立唯一索引（active 策略同一 base_symbol + timeframe + env 只能一筆）
CREATE UNIQUE INDEX IF NOT EXISTS idx_strategies_active_env
ON strategies (base_symbol, timeframe, env)
WHERE status = 'active';
