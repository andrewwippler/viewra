-- Processing queue for background workers.
-- Items are enqueued during scan, processed by background worker (FFprobe + enrichment).

CREATE TABLE processing_queue (
    id BIGSERIAL PRIMARY KEY,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    file_hash TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'processing', 'completed', 'failed')),
    media_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    error_message TEXT,
    attempts INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(library_id, file_path)
);

CREATE INDEX idx_processing_queue_status ON processing_queue(status, created_at);
CREATE INDEX idx_processing_queue_library ON processing_queue(library_id);
CREATE INDEX idx_processing_queue_pending ON processing_queue(status, created_at) WHERE status = 'pending';
