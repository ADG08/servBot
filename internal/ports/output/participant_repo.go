package output

import (
	"context"

	"servbot/internal/domain/entities"
)

type ParticipantRepository interface {
	Create(ctx context.Context, participant *entities.Participant) error
	FindByID(ctx context.Context, id uint) (*entities.Participant, error)
	FindByEventID(ctx context.Context, eventID uint) ([]entities.Participant, error)
	FindByEventIDAndUserID(ctx context.Context, eventID uint, userID string) (*entities.Participant, error)
	FindByEventIDAndStatus(ctx context.Context, eventID uint, status string) ([]entities.Participant, error)
	Update(ctx context.Context, participant *entities.Participant) error
	Delete(ctx context.Context, participant *entities.Participant) error
	CountByEventIDAndStatus(ctx context.Context, eventID uint, status string) (int64, error)
}
