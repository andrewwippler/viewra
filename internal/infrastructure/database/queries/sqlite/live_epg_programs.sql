-- name: CreateProgram :one
INSERT INTO live_epg_programs (
    channel_id,
    start_time,
    end_time,
    title,
    sub_title,
    description,
    category,
    episode_title,
    episode_num,
    season_num,
    is_new,
    is_movie
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetProgram :one
SELECT * FROM live_epg_programs
WHERE id = ?;

-- name: ListProgramsByChannel :many
SELECT * FROM live_epg_programs
WHERE channel_id = ?
  AND end_time > ?
  AND start_time < ?
ORDER BY start_time ASC;

-- name: ListProgramsByLibrary :many
SELECT p.* FROM live_epg_programs p
JOIN live_channels c ON p.channel_id = c.id
WHERE c.library_id = ?
  AND p.end_time > ?
  AND p.start_time < ?
ORDER BY c.channel_number ASC, p.start_time ASC;

-- name: GetCurrentProgram :one
SELECT * FROM live_epg_programs
WHERE channel_id = ?
  AND start_time <= ?
  AND end_time > ?
LIMIT 1;

-- name: DeleteProgramsByChannel :exec
DELETE FROM live_epg_programs
WHERE channel_id = ?;

-- name: DeleteProgramsByLibrary :exec
DELETE FROM live_epg_programs
WHERE channel_id IN (SELECT id FROM live_channels WHERE library_id = ?);

-- name: BulkCreatePrograms :exec
INSERT INTO live_epg_programs (
    channel_id,
    start_time,
    end_time,
    title,
    sub_title,
    description,
    category,
    episode_title,
    episode_num,
    season_num,
    is_new,
    is_movie
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteOldPrograms :exec
DELETE FROM live_epg_programs
WHERE end_time < ?;
