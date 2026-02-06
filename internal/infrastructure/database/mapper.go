package database

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"servbot/internal/domain/entities"
	"servbot/internal/infrastructure/database/sqlc_generated"
)

// pgtypeTimestamptzToTime returns t.Time when Valid, else zero time.
func pgtypeTimestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func eventToDomain(e sqlc_generated.Event) entities.Event {
	return entities.Event{
		ID:                          uint(e.ID),
		MessageID:                   e.MessageID,
		ChannelID:                   e.ChannelID,
		CreatorID:                   e.CreatorID,
		Title:                       e.Title,
		Description:                 e.Description,
		MaxSlots:                    int(e.MaxSlots),
		ScheduledAt:                 pgtypeTimestamptzToTime(e.ScheduledAt),
		PrivateChannelID:            e.PrivateChannelID,
		QuestionsThreadID:           e.QuestionsThreadID,
		OrganizerValidationDMSentAt: pgtypeTimestamptzToTime(e.OrganizerValidationDmSentAt),
		OrganizerStep1FinalizedAt:   pgtypeTimestamptzToTime(e.OrganizerStep1FinalizedAt),
		CreatedAt:                   pgtypeTimestamptzToTime(e.CreatedAt),
		UpdatedAt:                   pgtypeTimestamptzToTime(e.UpdatedAt),
	}
}

func participantToDomain(p sqlc_generated.Participant) entities.Participant {
	return entities.Participant{
		ID:        uint(p.ID),
		EventID:   uint(p.EventID),
		UserID:    p.UserID,
		Username:  p.Username,
		Status:    p.Status,
		JoinedAt:  pgtypeTimestamptzToTime(p.JoinedAt),
		CreatedAt: pgtypeTimestamptzToTime(p.CreatedAt),
		UpdatedAt: pgtypeTimestamptzToTime(p.UpdatedAt),
	}
}
