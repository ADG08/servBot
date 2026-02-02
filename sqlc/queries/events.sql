-- name: CreateEvent :one
INSERT INTO events (message_id, channel_id, creator_id, title, description, max_slots)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetEventByMessageID :one
SELECT * FROM events WHERE message_id = $1;

-- name: GetEventByID :one
SELECT * FROM events WHERE id = $1;

-- name: GetEventsByCreatorID :many
SELECT * FROM events WHERE creator_id = $1 ORDER BY created_at DESC;

-- name: UpdateEvent :exec
UPDATE events SET
    title = $2,
    description = $3,
    max_slots = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteEvent :exec
DELETE FROM events WHERE id = $1;
