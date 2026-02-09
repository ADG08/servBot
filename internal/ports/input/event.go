package input

import (
	"context"
	"time"

	"servbot/internal/domain/entities"
)

type EventUseCase interface {
	CreateEvent(ctx context.Context, event *entities.Event, creatorUsername string) error
	GetEventByMessageID(ctx context.Context, messageID string) (*entities.Event, error)
	GetEventByID(ctx context.Context, id uint) (*entities.Event, error)
	GetEventByPrivateChannelID(ctx context.Context, privateChannelID string) (*entities.Event, error)
	UpdateEvent(ctx context.Context, event *entities.Event) error
	GetWaitlistParticipants(ctx context.Context, eventID uint) ([]entities.Participant, error)
	GetConfirmedParticipants(ctx context.Context, eventID uint) ([]entities.Participant, error)
	GetEventsByCreatorID(ctx context.Context, creatorID string) ([]entities.Event, error)
	EventsNeedingH48OrganizerDM(ctx context.Context, now time.Time) ([]entities.Event, error)
	MarkOrganizerValidationDMSent(ctx context.Context, eventID uint) error
	FinalizeOrganizerStep1(ctx context.Context, eventID uint, creatorID string) (*entities.Event, error)
}
