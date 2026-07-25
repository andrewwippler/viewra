-- Pending external IDs for media files, set by plugins before scan runs.
-- During scan, these are promoted to media_external_ids.

CREATE TABLE pending_identifications (
    id BIGSERIAL PRIMARY KEY,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    provider TEXT NOT NULL CHECK(provider IN ('imdb', 'tmdb', 'tvdb', 'musicbrainz')),
    external_id TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(file_path, provider)
);

CREATE INDEX idx_pending_identifications_library_path ON pending_identifications(library_id, file_path);
