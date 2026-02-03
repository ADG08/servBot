package input

import (
	"context"

	"servbot/internal/domain/entities"
)

type EventUseCase interface {
	CreateEvent(ctx context.Context, event *entities.Event) error
	GetEventByMessageID(ctx context.Context, messageID string) (*entities.Event, error)
	GetEventByID(ctx context.Context, id uint) (*entities.Event, error)
	UpdateEvent(ctx context.Context, event *entities.Event) error
	GetWaitlistParticipants(ctx context.Context, eventID uint) ([]entities.Participant, error)
	GetConfirmedParticipants(ctx context.Context, eventID uint) ([]entities.Participant, error)
	GetEventsByCreatorID(ctx context.Context, creatorID string) ([]entities.Event, error)
}
