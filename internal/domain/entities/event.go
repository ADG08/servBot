package entities

import "time"

func (e *Event) IsFinalized() bool {
	return !e.OrganizerStep1FinalizedAt.IsZero()
}

type Event struct {
	ID                          uint
	MessageID                   string
	ChannelID                   string
	CreatorID                   string
	Title                       string
	Description                 string
	MaxSlots                    int
	ScheduledAt                 time.Time // zero = not set (for backward compat)
	PrivateChannelID            string    // salon privé organisateur seul (+ bot)
	QuestionsThreadID           string    // thread "Questions" dans ce salon
	OrganizerValidationDMSentAt time.Time
	OrganizerStep1FinalizedAt   time.Time
	Participants                []Participant
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}
