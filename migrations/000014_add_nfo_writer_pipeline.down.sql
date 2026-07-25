-- Remove nfo-writer stage from enrichment pipeline
DELETE FROM enrichment_pipelines WHERE plugin_id = 'builtin:nfo-writer';
