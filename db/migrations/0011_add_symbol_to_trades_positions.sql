-- Add symbol column to trades and positions for manual or multi-symbol support
ALTER TABLE strategy_trades ADD COLUMN IF NOT EXISTS symbol VARCHAR(64);
ALTER TABLE strategy_positions ADD COLUMN IF NOT EXISTS symbol VARCHAR(64);

-- Update existing records if possible (though base_symbol is on strategy)
UPDATE strategy_trades t SET symbol = s.base_symbol FROM strategies s WHERE t.strategy_id = s.id AND t.symbol IS NULL;
UPDATE strategy_positions p SET symbol = s.base_symbol FROM strategies s WHERE p.strategy_id = s.id AND p.symbol IS NULL;

-- Make symbol NOT NULL for future records if desired, but we'll leave it optional for now to avoid migration issues with legacy code that doesn't provide it.
-- Actually, better to make it DEFAULT 'BTCUSDT' or similar.
ALTER TABLE strategy_trades ALTER COLUMN symbol SET DEFAULT 'BTCUSDT';
ALTER TABLE strategy_positions ALTER COLUMN symbol SET DEFAULT 'BTCUSDT';

-- Also drop the foreign key constraint on strategy_id for manual trades if needed,
-- or just allow NULL strategy_id.
ALTER TABLE strategy_trades ALTER COLUMN strategy_id DROP NOT NULL;
ALTER TABLE strategy_positions ALTER COLUMN strategy_id DROP NOT NULL;
