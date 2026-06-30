-- Revert Live TV Library Support

DROP TABLE IF EXISTS live_channel_epg_mapping;
DROP TABLE IF EXISTS live_epg_programs;
DROP TABLE IF EXISTS live_channels;

-- Restore libraries CHECK constraint
ALTER TABLE libraries DROP CONSTRAINT IF EXISTS libraries_type_check;
ALTER TABLE libraries ADD CONSTRAINT libraries_type_check CHECK(type IN ('movies', 'tv', 'music'));
