-- Fix AI Opt Strategy to use BASE_SCORE condition
-- 1. Insert the missing condition type for Base AI Score
INSERT INTO conditions (name, type, params)
VALUES ('Base AI Score', 'BASE_SCORE', '{}')
ON CONFLICT DO NOTHING;

-- 2. Clear old rules for the strategy (ID: 5b8bd338...)
DELETE FROM strategy_rules 
WHERE strategy_id = '5b8bd338-8e51-4223-8696-79bd8736a223';

-- 3. Add the Base Score rule with 100 weight
-- Note: We use a subquery to get the condition ID we just inserted/found
INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type) 
VALUES (
    '5b8bd338-8e51-4223-8696-79bd8736a223', 
    (SELECT id FROM conditions WHERE type='BASE_SCORE' ORDER BY created_at DESC LIMIT 1), 
    100, 
    'entry'
);

-- 4. Ensure threshold is 80
UPDATE strategies 
SET threshold = 80 
WHERE id = '5b8bd338-8e51-4223-8696-79bd8736a223';
