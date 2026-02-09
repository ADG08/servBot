package output

import (
	"context"
	"time"

	"servbot/internal/domain/entities"
)

type EventRepository interface {
	Create(ctx context.Context, event *entities.Event) error
	FindByMessageID(ctx context.Context, messageID string) (*entities.Event, error)
	FindByID(ctx context.Context, id uint) (*entities.Event, error)
	FindByPrivateChannelID(ctx context.Context, privateChannelID string) (*entities.Event, error)
	FindByCreatorID(ctx context.Context, creatorID string) ([]entities.Event, error)
	FindEventsNeedingH48OrganizerDM(ctx context.Context, now time.Time) ([]entities.Event, error)
	Update(ctx context.Context, event *entities.Event) error
	MarkOrganizerValidationDMSent(ctx context.Context, eventID uint) error
	MarkOrganizerStep1Finalized(ctx context.Context, eventID uint) error
	Delete(ctx context.Context, id uint) error
}
