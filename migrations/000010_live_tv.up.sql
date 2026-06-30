-- Live TV Library Support (idempotent)
-- Only creates the live TV tables. The libraries_v2 table swap was already handled by 000009.

-- Live TV Channels
CREATE TABLE IF NOT EXISTS live_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    channel_number INTEGER NOT NULL DEFAULT 0,
    name TEXT NOT NULL,
    stream_url TEXT NOT NULL,
    logo_url TEXT,
    channel_group TEXT,
    epg_channel_id TEXT,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(library_id, name)
);

CREATE INDEX IF NOT EXISTS idx_live_channels_library ON live_channels(library_id);
CREATE INDEX IF NOT EXISTS idx_live_channels_group ON live_channels(channel_group);

-- Live TV EPG Programs
CREATE TABLE IF NOT EXISTS live_epg_programs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER NOT NULL REFERENCES live_channels(id) ON DELETE CASCADE,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    title TEXT NOT NULL,
    sub_title TEXT,
    description TEXT,
    category TEXT,
    episode_title TEXT,
    episode_num INTEGER,
    season_num INTEGER,
    is_new INTEGER DEFAULT 0,
    is_movie INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_epg_channel_time ON live_epg_programs(channel_id, start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_epg_time_range ON live_epg_programs(start_time, end_time);

-- Live TV Channel EPG Mapping
CREATE TABLE IF NOT EXISTS live_channel_epg_mapping (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    library_id INTEGER NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    xmltv_channel_id TEXT NOT NULL,
    channel_id INTEGER NOT NULL REFERENCES live_channels(id) ON DELETE CASCADE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(library_id, xmltv_channel_id)
);

CREATE INDEX IF NOT EXISTS idx_live_channel_epg_mapping_library ON live_channel_epg_mapping(library_id);
CREATE INDEX IF NOT EXISTS idx_live_channel_epg_mapping_channel ON live_channel_epg_mapping(channel_id);
