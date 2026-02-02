package entities

import "time"

// Participant represents a user's participation in an event.
type Participant struct {
	ID        uint
	EventID   uint
	UserID    string
	Username  string
	Status    string
	JoinedAt  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
