-- name: CreateParticipant :one
INSERT INTO participants (event_id, user_id, username, status, joined_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetParticipantByID :one
SELECT * FROM participants WHERE id = $1;

-- name: GetParticipantsByEventID :many
SELECT * FROM participants WHERE event_id = $1 ORDER BY created_at ASC;

-- name: GetParticipantByEventIDAndUserID :one
SELECT * FROM participants WHERE event_id = $1 AND user_id = $2;

-- name: GetParticipantsByEventIDAndStatus :many
SELECT * FROM participants WHERE event_id = $1 AND status = $2 ORDER BY created_at ASC;

-- name: UpdateParticipant :exec
UPDATE participants SET
    username = $2,
    status = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteParticipant :exec
DELETE FROM participants WHERE id = $1;

-- name: CountParticipantsByEventIDAndStatus :one
SELECT COUNT(*) FROM participants WHERE event_id = $1 AND status = $2;
