-- name: CreateEvent :one
INSERT INTO events (message_id, channel_id, creator_id, title, description, max_slots, scheduled_at, private_channel_id, questions_thread_id, waitlist_auto)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: FindEventsNeedingH48OrganizerDM :many
SELECT * FROM events
WHERE scheduled_at IS NOT NULL
  AND scheduled_at > $1
  AND scheduled_at - interval '48 hours' <= $1
  AND scheduled_at - interval '47 hours' > $1
  AND organizer_validation_dm_sent_at IS NULL;

-- name: MarkOrganizerValidationDMSent :exec
UPDATE events SET organizer_validation_dm_sent_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: MarkOrganizerStep1Finalized :exec
UPDATE events SET organizer_step1_finalized_at = NOW(), updated_at = NOW() WHERE id = $1;

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
    scheduled_at = $5,
    waitlist_auto = $6,
    updated_at = NOW()
WHERE id = $1;

-- name: FindStartedNonFinalizedEvents :many
SELECT * FROM events
WHERE scheduled_at IS NOT NULL
  AND scheduled_at <= $1
  AND scheduled_at > $1 - interval '1 hour'
  AND organizer_step1_finalized_at IS NULL;

-- name: GetEventByPrivateChannelID :one
SELECT * FROM events WHERE private_channel_id = $1;

-- name: DeleteEvent :exec
DELETE FROM events WHERE id = $1;
