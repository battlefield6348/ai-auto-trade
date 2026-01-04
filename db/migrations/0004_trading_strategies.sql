-- Trading strategies, backtests, trades, positions, logs, reports
-- Derived from docs/specs/trade_conditions_backtesting.md v1.0

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS strategies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    base_symbol     VARCHAR(64) NOT NULL DEFAULT 'BTCUSDT',
    timeframe       VARCHAR(32) NOT NULL DEFAULT '1d',
    env             VARCHAR(16) NOT NULL DEFAULT 'both', -- test | prod | both
    status          VARCHAR(32) NOT NULL DEFAULT 'draft', -- draft | active | archived
    version         INTEGER NOT NULL DEFAULT 1,
    buy_conditions  JSONB NOT NULL,
    sell_conditions JSONB NOT NULL,
    risk_settings   JSONB NOT NULL,
    created_by      UUID REFERENCES users(id),
    updated_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_strategy_env CHECK (env IN ('test','prod','both')),
    CONSTRAINT chk_strategy_status CHECK (status IN ('draft','active','archived'))
);

CREATE TABLE IF NOT EXISTS strategy_backtests (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id      UUID REFERENCES strategies(id) ON DELETE CASCADE,
    strategy_version INTEGER NOT NULL,
    start_date       DATE NOT NULL,
    end_date         DATE NOT NULL,
    params           JSONB NOT NULL, -- initial_equity, fees, slippage, price_mode, risk
    stats            JSONB,          -- total_return, win_rate, dd, trade_count, etc.
    equity_curve     JSONB,
    trades           JSONB,
    created_by       UUID REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategy_backtests_strategy ON strategy_backtests(strategy_id);

CREATE TABLE IF NOT EXISTS strategy_trades (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id      UUID REFERENCES strategies(id) ON DELETE CASCADE,
    strategy_version INTEGER NOT NULL,
    env              VARCHAR(16) NOT NULL,
    side             VARCHAR(8) NOT NULL, -- buy | sell
    entry_date       DATE NOT NULL,
    entry_price      NUMERIC(20,8) NOT NULL,
    exit_date        DATE,
    exit_price       NUMERIC(20,8),
    pnl_usdt         NUMERIC(20,8),
    pnl_pct          NUMERIC(10,6),
    hold_days        INTEGER,
    reason           VARCHAR(128),
    params_snapshot  JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_strategy_trades_env CHECK (env IN ('test','prod')),
    CONSTRAINT chk_strategy_trades_side CHECK (side IN ('buy','sell'))
);

CREATE INDEX IF NOT EXISTS idx_strategy_trades_strategy ON strategy_trades(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_trades_env ON strategy_trades(env);

CREATE TABLE IF NOT EXISTS strategy_positions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id UUID REFERENCES strategies(id) ON DELETE CASCADE,
    env         VARCHAR(16) NOT NULL,
    entry_date  DATE NOT NULL,
    entry_price NUMERIC(20,8) NOT NULL,
    size        NUMERIC(20,8) NOT NULL, -- 以 USDT 或張數，由 risk_settings 決定
    stop_loss   NUMERIC(20,8),
    take_profit NUMERIC(20,8),
    status      VARCHAR(16) NOT NULL DEFAULT 'open', -- open | closed
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_strategy_positions_env CHECK (env IN ('test','prod')),
    CONSTRAINT chk_strategy_positions_status CHECK (status IN ('open','closed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_strategy_positions_open
ON strategy_positions(strategy_id, env)
WHERE status = 'open';

CREATE TABLE IF NOT EXISTS strategy_logs (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id      UUID REFERENCES strategies(id) ON DELETE CASCADE,
    strategy_version INTEGER,
    env              VARCHAR(16),
    date             DATE,
    phase            VARCHAR(32),
    message          TEXT,
    payload          JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategy_logs_strategy ON strategy_logs(strategy_id);
CREATE INDEX IF NOT EXISTS idx_strategy_logs_date ON strategy_logs(date);

CREATE TABLE IF NOT EXISTS strategy_reports (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id      UUID REFERENCES strategies(id) ON DELETE CASCADE,
    strategy_version INTEGER,
    env              VARCHAR(16),
    period_start     DATE,
    period_end       DATE,
    summary          JSONB,
    trades_ref       JSONB,
    created_by       UUID REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategy_reports_strategy ON strategy_reports(strategy_id);
