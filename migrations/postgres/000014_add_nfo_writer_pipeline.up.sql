-- Add nfo-writer stage to enrichment pipeline (writes NFO files after enrichment)
INSERT INTO enrichment_pipelines (media_type, plugin_id, stage_name, position, enabled) VALUES
    ('movie', 'builtin:nfo-writer', 'nfo-writer', 1000, 1),
    ('tv', 'builtin:nfo-writer', 'nfo-writer', 1000, 1),
    ('tv_show', 'builtin:nfo-writer', 'nfo-writer', 1000, 1);
