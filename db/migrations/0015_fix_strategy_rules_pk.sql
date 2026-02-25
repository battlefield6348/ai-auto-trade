-- Fix strategy_rules PRIMARY KEY to include rule_type
-- This allows the same condition to be used for both entry and exit with different weights

ALTER TABLE strategy_rules DROP CONSTRAINT IF EXISTS strategy_rules_pkey;
ALTER TABLE strategy_rules ADD PRIMARY KEY (strategy_id, condition_id, rule_type);
