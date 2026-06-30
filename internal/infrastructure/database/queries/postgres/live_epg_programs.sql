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
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: GetProgram :one
SELECT * FROM live_epg_programs
WHERE id = $1;

-- name: ListProgramsByChannel :many
SELECT * FROM live_epg_programs
WHERE channel_id = $1
  AND end_time > $2
  AND start_time < $3
ORDER BY start_time ASC;

-- name: ListProgramsByLibrary :many
SELECT p.* FROM live_epg_programs p
JOIN live_channels c ON p.channel_id = c.id
WHERE c.library_id = $1
  AND p.end_time > $2
  AND p.start_time < $3
ORDER BY c.channel_number ASC, p.start_time ASC;

-- name: GetCurrentProgram :one
SELECT * FROM live_epg_programs
WHERE channel_id = $1
  AND start_time <= $2
  AND end_time > $3
LIMIT 1;

-- name: DeleteProgramsByChannel :exec
DELETE FROM live_epg_programs
WHERE channel_id = $1;

-- name: DeleteProgramsByLibrary :exec
DELETE FROM live_epg_programs
WHERE channel_id IN (SELECT id FROM live_channels WHERE library_id = $1);

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
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: DeleteOldPrograms :exec
DELETE FROM live_epg_programs
WHERE end_time < $1;
