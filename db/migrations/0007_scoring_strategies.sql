-- Add slug and threshold to existing strategies table
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS slug VARCHAR(255) UNIQUE;
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS threshold FLOAT NOT NULL DEFAULT 0.0;

-- Create conditions table (Reusable logic components)
CREATE TABLE IF NOT EXISTS conditions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL, -- Display name
    type VARCHAR(50) NOT NULL,  -- Logic type (e.g., 'MA_CROSS', 'RSI')
    params JSONB NOT NULL,      -- Logic parameters
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create strategy_rules table (Association & Weights)
CREATE TABLE IF NOT EXISTS strategy_rules (
    strategy_id UUID NOT NULL REFERENCES strategies(id) ON DELETE CASCADE,
    condition_id UUID NOT NULL REFERENCES conditions(id) ON DELETE CASCADE,
    weight FLOAT NOT NULL DEFAULT 1.0,
    PRIMARY KEY (strategy_id, condition_id)
);

-- Index for performance
CREATE INDEX IF NOT EXISTS idx_strategies_slug ON strategies(slug);
CREATE INDEX IF NOT EXISTS idx_strategy_rules_strategy_id ON strategy_rules(strategy_id);
