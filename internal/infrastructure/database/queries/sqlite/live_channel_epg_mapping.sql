-- name: CreateChannelEPGMapping :one
INSERT INTO live_channel_epg_mapping (
    library_id,
    xmltv_channel_id,
    channel_id
) VALUES (?, ?, ?)
RETURNING *;

-- name: GetChannelEPGMapping :one
SELECT * FROM live_channel_epg_mapping
WHERE library_id = ? AND xmltv_channel_id = ?;

-- name: ListChannelEPGMappings :many
SELECT m.*, c.name as channel_name, c.stream_url
FROM live_channel_epg_mapping m
JOIN live_channels c ON c.id = m.channel_id
WHERE m.library_id = ?
ORDER BY m.xmltv_channel_id;

-- name: UpdateChannelEPGMapping :one
UPDATE live_channel_epg_mapping
SET channel_id = ?
WHERE library_id = ? AND xmltv_channel_id = ?
RETURNING *;

-- name: DeleteChannelEPGMapping :exec
DELETE FROM live_channel_epg_mapping
WHERE library_id = ? AND xmltv_channel_id = ?;

-- name: DeleteChannelEPGMappingsByLibrary :exec
DELETE FROM live_channel_epg_mapping
WHERE library_id = ?;

-- name: GetEPGMappingForLibrary :many
SELECT xmltv_channel_id, channel_id
FROM live_channel_epg_mapping
WHERE library_id = ?;