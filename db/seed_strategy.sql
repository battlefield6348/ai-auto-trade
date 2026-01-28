-- Seed a sample scoring strategy
INSERT INTO strategies (id, name, slug, threshold)
VALUES 
    ('00000000-0000-0000-0000-000000000001', 'BTC Trend Follower', 'btc_trend_v1', 15.0)
ON CONFLICT (slug) DO NOTHING;

-- Seed conditions
INSERT INTO conditions (id, name, type, params)
VALUES 
    ('10000000-0000-0000-0000-000000000001', 'High Return (5d)', 'PRICE_RETURN', '{"days": 5}'),
    ('10000000-0000-0000-0000-000000000002', 'Volume Surge', 'VOLUME_SURGE', '{}'),
    ('10000000-0000-0000-0000-000000000003', 'RSI Overbought (Sim)', 'PRICE_RETURN', '{"days": 20}')
ON CONFLICT (id) DO NOTHING;

-- Link rules to strategy
-- Condition 1 (Return 5d) has weight 1.0 (so return% * 100 * 1.0)
-- Condition 2 (Volume Surge) has weight 2.0 (so (vol-1)*10 * 2.0)
INSERT INTO strategy_rules (strategy_id, condition_id, weight)
VALUES 
    ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 1.0),
    ('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', 2.0)
ON CONFLICT DO NOTHING;
