-- Down migration for language variant support.
-- Since SQLite has limited ALTER TABLE support, we can't easily drop columns.
-- This removes the indexes but leaves the columns in place (they'll be unused).

DROP INDEX IF EXISTS idx_media_variant_group;
DROP INDEX IF EXISTS idx_media_language;
