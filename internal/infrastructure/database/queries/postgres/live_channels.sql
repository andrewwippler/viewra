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
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetChannel :one
SELECT * FROM live_channels
WHERE id = $1;

-- name: ListChannels :many
SELECT * FROM live_channels
WHERE library_id = $1
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
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT(library_id, name) DO UPDATE SET
    channel_number = EXCLUDED.channel_number,
    stream_url = EXCLUDED.stream_url,
    logo_url = EXCLUDED.logo_url,
    channel_group = EXCLUDED.channel_group,
    epg_channel_id = EXCLUDED.epg_channel_id,
    enabled = EXCLUDED.enabled,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteChannel :exec
DELETE FROM live_channels
WHERE id = $1;

-- name: DeleteChannelsByLibrary :exec
DELETE FROM live_channels
WHERE library_id = $1;

-- name: GetChannelByEPGID :one
SELECT * FROM live_channels
WHERE library_id = $1 AND epg_channel_id = $2;
