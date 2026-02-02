package database

import (
	"servbot/internal/domain/entities"
	"servbot/internal/infrastructure/database/sqlc_generated"
)

func eventToDomain(e sqlc_generated.Event) entities.Event {
	return entities.Event{
		ID:          uint(e.ID),
		MessageID:   e.MessageID,
		ChannelID:   e.ChannelID,
		CreatorID:   e.CreatorID,
		Title:       e.Title,
		Description: e.Description,
		MaxSlots:    int(e.MaxSlots),
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func participantToDomain(p sqlc_generated.Participant) entities.Participant {
	return entities.Participant{
		ID:        uint(p.ID),
		EventID:   uint(p.EventID),
		UserID:    p.UserID,
		Username:  p.Username,
		Status:    p.Status,
		JoinedAt:  p.JoinedAt,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
