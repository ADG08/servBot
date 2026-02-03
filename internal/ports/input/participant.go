package input

import (
	"context"

	"servbot/internal/domain/entities"
)

type ParticipantUseCase interface {
	JoinEvent(ctx context.Context, eventID uint, userID, username string) (string, error)
	LeaveEvent(ctx context.Context, eventID uint, userID string) (bool, error)
	PromoteParticipant(ctx context.Context, participantID uint, creatorID string) (*entities.Participant, error)
	RemoveParticipant(ctx context.Context, participantID uint, creatorID string) (*entities.Participant, error)
	GetNextWaitlistParticipant(ctx context.Context, eventID uint) (*entities.Participant, error)
}
