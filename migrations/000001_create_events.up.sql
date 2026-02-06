CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    message_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    creator_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    max_slots INT NOT NULL DEFAULT 0,
    scheduled_at TIMESTAMPTZ,
    private_channel_id TEXT NOT NULL DEFAULT '',
    questions_thread_id TEXT NOT NULL DEFAULT '',
    organizer_validation_dm_sent_at TIMESTAMPTZ,
    organizer_step1_finalized_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_message_id ON events(message_id);
CREATE INDEX IF NOT EXISTS idx_events_creator_id ON events(creator_id);
