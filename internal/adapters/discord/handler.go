package discord

import (
	"servbot/internal/ports/input"
)

// Handler handles Discord interactions using use cases.
type Handler struct {
	eventUseCase       input.EventUseCase
	participantUseCase input.ParticipantUseCase
	forumChannelID     string
}

// NewHandler creates a Handler.
func NewHandler(
	eventUseCase input.EventUseCase,
	participantUseCase input.ParticipantUseCase,
	forumChannelID string,
) *Handler {
	return &Handler{
		eventUseCase:       eventUseCase,
		participantUseCase: participantUseCase,
		forumChannelID:     forumChannelID,
	}
}
