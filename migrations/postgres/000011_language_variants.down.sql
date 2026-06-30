-- Down migration for language variant support (PostgreSQL).

DROP INDEX IF EXISTS idx_media_variant_group;
DROP INDEX IF EXISTS idx_media_language;

ALTER TABLE media DROP COLUMN IF EXISTS variant_group_id;
ALTER TABLE media DROP COLUMN IF EXISTS language;
