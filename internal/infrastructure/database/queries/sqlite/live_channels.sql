-- name: CreateChannel :one
INSERT INTO live_channels (
    library_id,
    channel_number,
    name,
    stream_url,
    logo_url,
    channel_group,
    epg_channel_id,
    enabled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetChannel :one
SELECT * FROM live_channels
WHERE id = ?;

-- name: ListChannels :many
SELECT * FROM live_channels
WHERE library_id = ?
ORDER BY channel_number ASC, name ASC;

-- name: UpsertChannel :one
INSERT INTO live_channels (
    library_id,
    channel_number,
    name,
    stream_url,
    logo_url,
    channel_group,
    epg_channel_id,
    enabled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(library_id, name) DO UPDATE SET
    channel_number = excluded.channel_number,
    stream_url = excluded.stream_url,
    logo_url = excluded.logo_url,
    channel_group = excluded.channel_group,
    epg_channel_id = excluded.epg_channel_id,
    enabled = excluded.enabled,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteChannel :exec
DELETE FROM live_channels
WHERE id = ?;

-- name: DeleteChannelsByLibrary :exec
DELETE FROM live_channels
WHERE library_id = ?;

-- name: GetChannelByEPGID :one
SELECT * FROM live_channels
WHERE library_id = ? AND epg_channel_id = ?;
