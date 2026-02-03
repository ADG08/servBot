package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"servbot/internal/domain/entities"
	"servbot/internal/infrastructure/database/sqlc_generated"
	"servbot/internal/ports/output"
)

var _ output.ParticipantRepository = (*ParticipantRepository)(nil)

// ParticipantRepository implements output.ParticipantRepository using sqlc + pgx.
type ParticipantRepository struct {
	q *sqlc_generated.Queries
}

// NewParticipantRepository creates a ParticipantRepository.
func NewParticipantRepository(q *sqlc_generated.Queries) *ParticipantRepository {
	return &ParticipantRepository{q: q}
}

func (r *ParticipantRepository) Create(ctx context.Context, participant *entities.Participant) error {
	row, err := r.q.CreateParticipant(ctx, sqlc_generated.CreateParticipantParams{
		EventID:  int64(participant.EventID),
		UserID:   participant.UserID,
		Username: participant.Username,
		Status:   participant.Status,
		JoinedAt: pgtype.Timestamptz{Time: participant.JoinedAt, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("create participant: %w", err)
	}
	participant.ID = uint(row.ID)
	participant.CreatedAt = pgtypeTimestamptzToTime(row.CreatedAt)
	participant.UpdatedAt = pgtypeTimestamptzToTime(row.UpdatedAt)
	return nil
}

func (r *ParticipantRepository) FindByID(ctx context.Context, id uint) (*entities.Participant, error) {
	row, err := r.q.GetParticipantByID(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("get participant by id: %w", err)
	}
	p := participantToDomain(row)
	return &p, nil
}

func (r *ParticipantRepository) FindByEventID(ctx context.Context, eventID uint) ([]entities.Participant, error) {
	rows, err := r.q.GetParticipantsByEventID(ctx, int64(eventID))
	if err != nil {
		return nil, fmt.Errorf("get participants by event id: %w", err)
	}
	out := make([]entities.Participant, len(rows))
	for i := range rows {
		out[i] = participantToDomain(rows[i])
	}
	return out, nil
}

func (r *ParticipantRepository) FindByEventIDAndUserID(ctx context.Context, eventID uint, userID string) (*entities.Participant, error) {
	row, err := r.q.GetParticipantByEventIDAndUserID(ctx, sqlc_generated.GetParticipantByEventIDAndUserIDParams{
		EventID: int64(eventID),
		UserID:  userID,
	})
	if err != nil {
		return nil, fmt.Errorf("get participant by event id and user id: %w", err)
	}
	p := participantToDomain(row)
	return &p, nil
}

func (r *ParticipantRepository) FindByEventIDAndStatus(ctx context.Context, eventID uint, status string) ([]entities.Participant, error) {
	rows, err := r.q.GetParticipantsByEventIDAndStatus(ctx, sqlc_generated.GetParticipantsByEventIDAndStatusParams{
		EventID: int64(eventID),
		Status:  status,
	})
	if err != nil {
		return nil, fmt.Errorf("get participants by event id and status: %w", err)
	}
	out := make([]entities.Participant, len(rows))
	for i := range rows {
		out[i] = participantToDomain(rows[i])
	}
	return out, nil
}

func (r *ParticipantRepository) Update(ctx context.Context, participant *entities.Participant) error {
	err := r.q.UpdateParticipant(ctx, sqlc_generated.UpdateParticipantParams{
		ID:       int64(participant.ID),
		Username: participant.Username,
		Status:   participant.Status,
	})
	if err != nil {
		return fmt.Errorf("update participant: %w", err)
	}
	return nil
}

func (r *ParticipantRepository) Delete(ctx context.Context, participant *entities.Participant) error {
	if err := r.q.DeleteParticipant(ctx, int64(participant.ID)); err != nil {
		return fmt.Errorf("delete participant: %w", err)
	}
	return nil
}

func (r *ParticipantRepository) CountByEventIDAndStatus(ctx context.Context, eventID uint, status string) (int64, error) {
	count, err := r.q.CountParticipantsByEventIDAndStatus(ctx, sqlc_generated.CountParticipantsByEventIDAndStatusParams{
		EventID: int64(eventID),
		Status:  status,
	})
	if err != nil {
		return 0, fmt.Errorf("count participants: %w", err)
	}
	return count, nil
}
