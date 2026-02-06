package discord

import (
	"servbot/internal/ports/input"
)

type Handler struct {
	eventUseCase       input.EventUseCase
	participantUseCase input.ParticipantUseCase
	forumChannelID     string
	guildID            string
}

func NewHandler(
	eventUseCase input.EventUseCase,
	participantUseCase input.ParticipantUseCase,
	forumChannelID string,
	guildID string,
) *Handler {
	return &Handler{
		eventUseCase:       eventUseCase,
		participantUseCase: participantUseCase,
		forumChannelID:     forumChannelID,
		guildID:            guildID,
	}
}
