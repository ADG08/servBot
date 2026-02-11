package discord

import (
	"servbot/internal/ports/input"
	"servbot/internal/ports/output"
)

type Handler struct {
	eventUseCase       input.EventUseCase
	participantUseCase input.ParticipantUseCase
	translator         output.T
	forumChannelID     string
	guildID            string
	defaultLocale      string
}

func NewHandler(
	eventUseCase input.EventUseCase,
	participantUseCase input.ParticipantUseCase,
	translator output.T,
	forumChannelID string,
	guildID string,
	defaultLocale string,
) *Handler {
	return &Handler{
		eventUseCase:       eventUseCase,
		participantUseCase: participantUseCase,
		translator:         translator,
		forumChannelID:     forumChannelID,
		guildID:            guildID,
		defaultLocale:      defaultLocale,
	}
}
