-- Add language variant support for media files (PostgreSQL).
-- Allows grouping multiple media files under one movie entry for multi-language support.

ALTER TABLE media ADD COLUMN IF NOT EXISTS language TEXT;
ALTER TABLE media ADD COLUMN IF NOT EXISTS variant_group_id BIGINT REFERENCES media(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_media_variant_group ON media(variant_group_id);
CREATE INDEX IF NOT EXISTS idx_media_language ON media(language);
