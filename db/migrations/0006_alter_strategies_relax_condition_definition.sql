-- Align legacy strategies table with新欄位需求，避免 NULL condition_definition 造成寫入失敗

-- 將舊欄位 condition_definition 設定為可為 NULL，並給預設空 JSON
UPDATE strategies SET condition_definition = '{}'::jsonb WHERE condition_definition IS NULL;
ALTER TABLE strategies
    ALTER COLUMN condition_definition SET DEFAULT '{}'::jsonb,
    ALTER COLUMN condition_definition DROP NOT NULL;
