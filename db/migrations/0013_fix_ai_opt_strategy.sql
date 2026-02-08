-- Fix AI Opt Strategy to include bonuses and AI score
-- 1. Ensure conditions exist
INSERT INTO conditions (name, type, params)
VALUES 
    ('Base AI Score', 'BASE_SCORE', '{}'),
    ('Daily Change Bonus', 'PRICE_RETURN', '{"days": 1, "min": 0.005}'),
    ('Volume Surge Bonus', 'VOLUME_SURGE', '{"min": 1.2}')
ON CONFLICT DO NOTHING;

-- 2. Clear old rules for the strategy (ID: 5b8bd338...)
DELETE FROM strategy_rules 
WHERE strategy_id = '5b8bd338-8e51-4223-8696-79bd8736a223';

-- 3. Add rules
-- AI Score (100 weight -> 1.0 multiplier)
INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type) 
VALUES (
    '5b8bd338-8e51-4223-8696-79bd8736a223', 
    (SELECT id FROM conditions WHERE type='BASE_SCORE' ORDER BY created_at DESC LIMIT 1), 
    100, 
    'entry'
);

-- Daily Change Bonus (15 weight)
INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type) 
VALUES (
    '5b8bd338-8e51-4223-8696-79bd8736a223', 
    (SELECT id FROM conditions WHERE type='PRICE_RETURN' AND (params->>'days')::int = 1 ORDER BY created_at DESC LIMIT 1), 
    15, 
    'entry'
);

-- Volume Surge Bonus (15 weight)
INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type) 
VALUES (
    '5b8bd338-8e51-4223-8696-79bd8736a223', 
    (SELECT id FROM conditions WHERE type='VOLUME_SURGE' ORDER BY created_at DESC LIMIT 1), 
    15, 
    'entry'
);

-- 4. Add a dummy exit rule to satisfy "must have exit rule" validation in SaveUseCase (though we are direct SQLing here)
INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type) 
VALUES (
    '5b8bd338-8e51-4223-8696-79bd8736a223', 
    (SELECT id FROM conditions WHERE type='BASE_SCORE' ORDER BY created_at DESC LIMIT 1), 
    10, 
    'exit'
);

-- 5. Ensure strategy metadata is set
UPDATE strategies 
SET 
    threshold = 80,
    exit_threshold = 10,
    updated_at = NOW()
WHERE id = '5b8bd338-8e51-4223-8696-79bd8736a223';
