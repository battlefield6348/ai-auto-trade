-- Remove duplicate conditions before creating unique index
DELETE FROM conditions a USING conditions b 
WHERE a.id < b.id AND a.name = b.name AND a.type = b.type AND a.params = b.params;

-- Add unique constraint to conditions to prevent duplicates and enable clean seeding
CREATE UNIQUE INDEX IF NOT EXISTS idx_conditions_unique_identity ON conditions (name, type, params);
