package output

import (
	"context"

	"servbot/internal/domain/entities"
)

type EventRepository interface {
	Create(ctx context.Context, event *entities.Event) error
	FindByMessageID(ctx context.Context, messageID string) (*entities.Event, error)
	FindByID(ctx context.Context, id uint) (*entities.Event, error)
	FindByCreatorID(ctx context.Context, creatorID string) ([]entities.Event, error)
	Update(ctx context.Context, event *entities.Event) error
	Delete(ctx context.Context, id uint) error
}
