ALTER TABLE release ADD COLUMN created_at timestamp with time zone;
UPDATE release SET created_at = now();
ALTER TABLE release ALTER COLUMN created_at SET NOT NULL;
