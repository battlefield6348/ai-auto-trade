-- Add rule_type to strategy_rules to distinguish entry/exit conditions
ALTER TABLE strategy_rules ADD COLUMN IF NOT EXISTS rule_type VARCHAR(10) DEFAULT 'entry';
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS exit_threshold FLOAT NOT NULL DEFAULT 0.0;
