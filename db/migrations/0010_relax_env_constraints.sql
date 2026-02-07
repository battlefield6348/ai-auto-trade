-- Relax environment constraints on trade and position tables
ALTER TABLE strategy_trades DROP CONSTRAINT IF EXISTS chk_strategy_trades_env;
ALTER TABLE strategy_trades ADD CONSTRAINT chk_strategy_trades_env CHECK (env IN ('test', 'prod', 'real', 'paper', 'both'));

ALTER TABLE strategy_positions DROP CONSTRAINT IF EXISTS chk_strategy_positions_env;
ALTER TABLE strategy_positions ADD CONSTRAINT chk_strategy_positions_env CHECK (env IN ('test', 'prod', 'real', 'paper', 'both'));

ALTER TABLE strategies DROP CONSTRAINT IF EXISTS chk_strategy_env;
ALTER TABLE strategies ADD CONSTRAINT chk_strategy_env CHECK (env IN ('test', 'prod', 'real', 'paper', 'both'));
