-- Add language variant support for media files.
-- Allows grouping multiple media files (e.g., different language dubs) under one movie entry.
-- See also: migrations/000011_language_variants.down.sql

ALTER TABLE media ADD COLUMN language TEXT;
ALTER TABLE media ADD COLUMN variant_group_id INTEGER REFERENCES media(id) ON DELETE SET NULL;

CREATE INDEX idx_media_variant_group ON media(variant_group_id);
CREATE INDEX idx_media_language ON media(language);
