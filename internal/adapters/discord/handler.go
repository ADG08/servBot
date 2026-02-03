package discord

import (
	"servbot/internal/ports/input"
)

type Handler struct {
	eventUseCase       input.EventUseCase
	participantUseCase input.ParticipantUseCase
	forumChannelID     string
}

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
