package entities

import "time"

// Event represents an outing/event in the domain.
type Event struct {
	ID           uint
	MessageID    string
	ChannelID    string
	CreatorID    string
	Title        string
	Description  string
	MaxSlots     int
	ScheduledAt  time.Time // zero = not set (for backward compat)
	Participants []Participant
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
