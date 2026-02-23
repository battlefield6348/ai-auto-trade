-- Fix AI Opt Strategy to include bonuses and AI score
-- 0. Ensure the strategy exists (using idempotent logic)
DO $$
DECLARE
    admin_id UUID;
BEGIN
    -- Try to find an admin or valid user to own this strategy
    SELECT id INTO admin_id FROM users WHERE email = 'admin@example.com' LIMIT 1;
    IF admin_id IS NULL THEN
        SELECT id INTO admin_id FROM users LIMIT 1;
    END IF;

    IF admin_id IS NOT NULL THEN
        INSERT INTO strategies (
            id, user_id, name, slug, timeframe, base_symbol, 
            threshold, exit_threshold, status, env, 
            buy_conditions, sell_conditions, risk_settings, 
            is_active, updated_at
        )
        VALUES (
            '5b8bd338-8e51-4223-8696-79bd8736a223', 
            admin_id, 
            'AI Optimized Strategy', 
            'ai-optimized', 
            '1d', 
            'BTCUSDT', 
            80, 
            10, 
            'active', 
            'both', 
            '[]', '[]', '{}', 
            true, 
            NOW()
        )
        ON CONFLICT (id) DO NOTHING;
    END IF;
END $$;

-- 1. Ensure conditions exist (using idempotent logic)
INSERT INTO conditions (name, type, params)
SELECT 'Base AI Score', 'BASE_SCORE', '{}'
WHERE NOT EXISTS (SELECT 1 FROM conditions WHERE type = 'BASE_SCORE');

INSERT INTO conditions (name, type, params)
SELECT 'Daily Change Bonus', 'PRICE_RETURN', '{"days": 1, "min": 0.005}'
WHERE NOT EXISTS (SELECT 1 FROM conditions WHERE type = 'PRICE_RETURN' AND (params->>'days')::int = 1);

INSERT INTO conditions (name, type, params)
SELECT 'Volume Surge Bonus', 'VOLUME_SURGE', '{"min": 1.2}'
WHERE NOT EXISTS (SELECT 1 FROM conditions WHERE type = 'VOLUME_SURGE');

-- 2. Clear old rules for the strategy (ID: 5b8bd338...)
DELETE FROM strategy_rules 
WHERE strategy_id = '5b8bd338-8e51-4223-8696-79bd8736a223';

-- 3. Add rules (Ensuring unique condition_id per strategy)
-- AI Score (100 weight)
INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type) 
SELECT 
    '5b8bd338-8e51-4223-8696-79bd8736a223', 
    id, 
    100, 
    'entry'
FROM conditions WHERE type='BASE_SCORE' ORDER BY created_at DESC LIMIT 1;

-- Daily Change Bonus (15 weight)
INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type) 
SELECT 
    '5b8bd338-8e51-4223-8696-79bd8736a223', 
    id, 
    15, 
    'entry'
FROM conditions WHERE type='PRICE_RETURN' AND (params->>'days')::int = 1 ORDER BY created_at DESC LIMIT 1;

-- Volume Surge Bonus (15 weight)
INSERT INTO strategy_rules (strategy_id, condition_id, weight, rule_type) 
SELECT 
    '5b8bd338-8e51-4223-8696-79bd8736a223', 
    id, 
    15, 
    'entry'
FROM conditions WHERE type='VOLUME_SURGE' ORDER BY created_at DESC LIMIT 1;

-- 4. Ensure strategy metadata is set
UPDATE strategies 
SET 
    threshold = 80,
    exit_threshold = 10,
    updated_at = NOW()
WHERE id = '5b8bd338-8e51-4223-8696-79bd8736a223';
