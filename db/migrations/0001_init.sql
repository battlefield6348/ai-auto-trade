-- Initial schema for Taiwan Stock Analyzer (PostgreSQL)
-- Generated from docs/specs/database_schema.md v1.0 (2025-12-03)

-- Helpers
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1) Users & Auth
CREATE TABLE IF NOT EXISTS users (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email             VARCHAR(255) NOT NULL UNIQUE,
    password_hash     VARCHAR(255) NOT NULL,
    display_name      VARCHAR(255) NOT NULL,
    status            VARCHAR(32) NOT NULL DEFAULT 'active',
    last_login_at     TIMESTAMPTZ,
    is_service_account BOOLEAN NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS roles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(64) NOT NULL UNIQUE,
    description     TEXT,
    is_system_role  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_roles (
    id       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id  UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS permissions (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(128) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_id        UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id  UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_id VARCHAR(255) NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    revoked_at       TIMESTAMPTZ,
    user_agent       TEXT,
    ip_address       INET,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (refresh_token_id)
);

-- 2) Core market data
CREATE TABLE IF NOT EXISTS stocks (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stock_code    VARCHAR(16) NOT NULL,
    market_type   VARCHAR(32) NOT NULL,
    name_zh       VARCHAR(255) NOT NULL,
    name_en       VARCHAR(255),
    industry      VARCHAR(255),
    listing_date  DATE,
    delisting_date DATE,
    status        VARCHAR(32) NOT NULL DEFAULT 'active',
    category      VARCHAR(64),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (stock_code, market_type)
);

CREATE INDEX IF NOT EXISTS idx_stocks_code ON stocks(stock_code);

CREATE TABLE IF NOT EXISTS daily_prices (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stock_id       UUID NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    trade_date     DATE NOT NULL,
    open_price     NUMERIC(18,4) NOT NULL CHECK (open_price >= 0),
    high_price     NUMERIC(18,4) NOT NULL CHECK (high_price >= 0),
    low_price      NUMERIC(18,4) NOT NULL CHECK (low_price >= 0),
    close_price    NUMERIC(18,4) NOT NULL CHECK (close_price >= 0),
    volume         BIGINT NOT NULL CHECK (volume >= 0),
    turnover       NUMERIC(20,2) DEFAULT 0 CHECK (turnover >= 0),
    trade_count    BIGINT,
    change         NUMERIC(18,4),
    change_percent NUMERIC(9,4),
    is_limit_up    BOOLEAN,
    is_limit_down  BOOLEAN,
    is_dividend_date BOOLEAN,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (stock_id, trade_date)
);

CREATE INDEX IF NOT EXISTS idx_daily_prices_stock_date ON daily_prices(stock_id, trade_date);
CREATE INDEX IF NOT EXISTS idx_daily_prices_date ON daily_prices(trade_date);

-- 3) Analysis results & jobs
CREATE TABLE IF NOT EXISTS analysis_results (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stock_id           UUID NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    trade_date         DATE NOT NULL,
    analysis_version   VARCHAR(64) NOT NULL,
    close_price        NUMERIC(18,4),
    change             NUMERIC(18,4),
    change_percent     NUMERIC(9,4),
    return_5d          NUMERIC(9,4),
    return_20d         NUMERIC(9,4),
    return_60d         NUMERIC(9,4),
    high_20d           NUMERIC(18,4),
    low_20d            NUMERIC(18,4),
    price_position_20d NUMERIC(9,4),
    ma_5               NUMERIC(18,4),
    ma_10              NUMERIC(18,4),
    ma_20              NUMERIC(18,4),
    ma_60              NUMERIC(18,4),
    ma_trend_flag      VARCHAR(32),
    volume             BIGINT,
    volume_avg_5d      NUMERIC(18,4),
    volume_avg_20d     NUMERIC(18,4),
    volume_ratio       NUMERIC(9,4),
    volatility_20d     NUMERIC(9,4),
    volatility_60d     NUMERIC(9,4),
    score              NUMERIC(9,4),
    tags               JSONB,
    status             VARCHAR(32) NOT NULL DEFAULT 'success',
    error_reason       TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (stock_id, trade_date, analysis_version)
);

CREATE INDEX IF NOT EXISTS idx_analysis_results_date ON analysis_results(trade_date);
CREATE INDEX IF NOT EXISTS idx_analysis_results_stock_date ON analysis_results(stock_id, trade_date);
CREATE INDEX IF NOT EXISTS idx_analysis_results_date_score ON analysis_results(trade_date, score DESC);

CREATE TABLE IF NOT EXISTS ingestion_jobs (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_type         VARCHAR(32) NOT NULL,
    target_start_date DATE,
    target_end_date   DATE,
    options          JSONB,
    status           VARCHAR(32) NOT NULL,
    started_at       TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    success_count    INTEGER DEFAULT 0,
    failure_count    INTEGER DEFAULT 0,
    error_summary    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_type_created ON ingestion_jobs(job_type, created_at);
CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_status_created ON ingestion_jobs(status, created_at);

CREATE TABLE IF NOT EXISTS ingestion_job_items (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ingestion_job_id UUID NOT NULL REFERENCES ingestion_jobs(id) ON DELETE CASCADE,
    stock_id         UUID REFERENCES stocks(id) ON DELETE CASCADE,
    trade_date       DATE,
    status           VARCHAR(32) NOT NULL,
    error_reason     TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingestion_job_items_job ON ingestion_job_items(ingestion_job_id);
CREATE INDEX IF NOT EXISTS idx_ingestion_job_items_stock_date ON ingestion_job_items(stock_id, trade_date);

CREATE TABLE IF NOT EXISTS analysis_jobs (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_type         VARCHAR(32) NOT NULL,
    target_date      DATE NOT NULL,
    options          JSONB,
    analysis_version VARCHAR(64),
    status           VARCHAR(32) NOT NULL,
    started_at       TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    total_stocks     INTEGER,
    success_count    INTEGER,
    failure_count    INTEGER,
    error_summary    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analysis_jobs_target_date ON analysis_jobs(target_date);
CREATE INDEX IF NOT EXISTS idx_analysis_jobs_status_created ON analysis_jobs(status, created_at);

CREATE TABLE IF NOT EXISTS analysis_job_items (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    analysis_job_id  UUID NOT NULL REFERENCES analysis_jobs(id) ON DELETE CASCADE,
    stock_id         UUID NOT NULL REFERENCES stocks(id) ON DELETE CASCADE,
    status           VARCHAR(32) NOT NULL,
    error_reason     TEXT,
    duration_ms      INTEGER,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_analysis_job_items_job ON analysis_job_items(analysis_job_id);
CREATE INDEX IF NOT EXISTS idx_analysis_job_items_stock_job ON analysis_job_items(stock_id, analysis_job_id);

-- 6) Screener
CREATE TABLE IF NOT EXISTS screener_presets (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_user_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    name                 VARCHAR(255) NOT NULL,
    description          TEXT,
    is_public            BOOLEAN NOT NULL DEFAULT FALSE,
    is_system_preset     BOOLEAN NOT NULL DEFAULT FALSE,
    condition_definition JSONB NOT NULL,
    sort_definition      JSONB,
    status               VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_screener_presets_owner ON screener_presets(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_screener_presets_public ON screener_presets(is_public, is_system_preset);

CREATE TABLE IF NOT EXISTS screener_queries (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    screener_preset_id   UUID REFERENCES screener_presets(id) ON DELETE SET NULL,
    condition_definition JSONB NOT NULL,
    sort_definition      JSONB,
    trade_date           DATE,
    result_count         INTEGER,
    executed_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_screener_queries_user_exec ON screener_queries(user_id, executed_at);
CREATE INDEX IF NOT EXISTS idx_screener_queries_trade_date ON screener_queries(trade_date);

-- 7) Alerts & Notifications
CREATE TABLE IF NOT EXISTS subscriptions (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                 VARCHAR(255) NOT NULL,
    subscription_type    VARCHAR(32) NOT NULL,
    condition_definition JSONB NOT NULL,
    min_hit_count        INTEGER DEFAULT 0,
    channels             JSONB NOT NULL,
    webhook_url          TEXT,
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    last_triggered_at    TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_type ON subscriptions(subscription_type);
CREATE INDEX IF NOT EXISTS idx_subscriptions_active ON subscriptions(is_active);

CREATE TABLE IF NOT EXISTS notifications (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subscription_id    UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    user_id            UUID REFERENCES users(id) ON DELETE SET NULL,
    notification_type  VARCHAR(32) NOT NULL,
    trade_date         DATE,
    payload_summary    JSONB,
    channel            VARCHAR(32) NOT NULL,
    sent_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status             VARCHAR(32) NOT NULL,
    error_reason       TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id, sent_at);
CREATE INDEX IF NOT EXISTS idx_notifications_subscription ON notifications(subscription_id, sent_at);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(notification_type, sent_at);

-- 8) Strategies
CREATE TABLE IF NOT EXISTS strategies (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                 VARCHAR(255) NOT NULL,
    description          TEXT,
    strategy_type        VARCHAR(64),
    condition_definition JSONB NOT NULL,
    action_definition    JSONB,
    schedule_definition  JSONB,
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    last_executed_at     TIMESTAMPTZ,
    last_triggered_at    TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategies_user ON strategies(user_id);
CREATE INDEX IF NOT EXISTS idx_strategies_active ON strategies(is_active);

CREATE TABLE IF NOT EXISTS strategy_runs (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_id      UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    run_date         DATE NOT NULL,
    started_at       TIMESTAMPTZ,
    finished_at      TIMESTAMPTZ,
    status           VARCHAR(32) NOT NULL,
    total_candidates INTEGER,
    hit_count        INTEGER,
    error_summary    TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategy_runs_strategy_date ON strategy_runs(strategy_id, run_date);
CREATE INDEX IF NOT EXISTS idx_strategy_runs_status ON strategy_runs(status, started_at);

CREATE TABLE IF NOT EXISTS strategy_hits (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    strategy_run_id  UUID NOT NULL REFERENCES strategy_runs(id) ON DELETE CASCADE,
    stock_id         UUID REFERENCES stocks(id) ON DELETE SET NULL,
    trade_date       DATE,
    hit_detail       JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_strategy_hits_run ON strategy_hits(strategy_run_id);
CREATE INDEX IF NOT EXISTS idx_strategy_hits_stock_date ON strategy_hits(stock_id, trade_date);

-- 9) Reports & Exports
CREATE TABLE IF NOT EXISTS export_jobs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    export_type   VARCHAR(64) NOT NULL,
    parameters    JSONB,
    status        VARCHAR(32) NOT NULL,
    file_path     TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    error_reason  TEXT
);

CREATE INDEX IF NOT EXISTS idx_export_jobs_user ON export_jobs(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_export_jobs_type ON export_jobs(export_type, created_at);
CREATE INDEX IF NOT EXISTS idx_export_jobs_status ON export_jobs(status, created_at);

-- 10) System Events
CREATE TABLE IF NOT EXISTS system_events (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type      VARCHAR(64) NOT NULL,
    severity        VARCHAR(16) NOT NULL,
    message         TEXT NOT NULL,
    details         JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    related_job_id  UUID,
    related_stock_id UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_events_type ON system_events(event_type, occurred_at);
CREATE INDEX IF NOT EXISTS idx_system_events_severity ON system_events(severity, occurred_at);

