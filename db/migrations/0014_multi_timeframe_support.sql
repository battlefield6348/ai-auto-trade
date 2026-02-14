-- Migration: Multi-Timeframe Support
-- Description: Add timeframe column to price and analysis tables, and update unique constraints.

-- 1. Update daily_prices
ALTER TABLE daily_prices ADD COLUMN IF NOT EXISTS timeframe VARCHAR(16) NOT NULL DEFAULT '1d';
ALTER TABLE daily_prices DROP CONSTRAINT IF EXISTS daily_prices_stock_id_trade_date_key;
ALTER TABLE daily_prices ADD CONSTRAINT daily_prices_stock_timeframe_date_key UNIQUE (stock_id, timeframe, trade_date);

-- 2. Update analysis_results
ALTER TABLE analysis_results ADD COLUMN IF NOT EXISTS timeframe VARCHAR(16) NOT NULL DEFAULT '1d';
ALTER TABLE analysis_results DROP CONSTRAINT IF EXISTS analysis_results_stock_id_trade_date_analysis_version_key;
ALTER TABLE analysis_results ADD CONSTRAINT analysis_results_stock_timeframe_date_ver_key UNIQUE (stock_id, timeframe, trade_date, analysis_version);

-- 3. Update strategies
ALTER TABLE strategies ADD COLUMN IF NOT EXISTS timeframe VARCHAR(16) NOT NULL DEFAULT '1d';

-- 4. Update ingestion_job_items (optional but good for tracking)
ALTER TABLE ingestion_job_items ADD COLUMN IF NOT EXISTS timeframe VARCHAR(16) NOT NULL DEFAULT '1d';
