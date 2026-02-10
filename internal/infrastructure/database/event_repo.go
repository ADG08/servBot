package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"servbot/internal/domain/entities"
	"servbot/internal/infrastructure/database/sqlc_generated"
	"servbot/internal/ports/output"
)

var _ output.EventRepository = (*EventRepository)(nil)

type EventRepository struct {
	q *sqlc_generated.Queries
}

func NewEventRepository(pool *sqlc_generated.Queries) *EventRepository {
	return &EventRepository{q: pool}
}

func (r *EventRepository) Create(ctx context.Context, event *entities.Event) error {
	var scheduledAt pgtype.Timestamptz
	if !event.ScheduledAt.IsZero() {
		scheduledAt = pgtype.Timestamptz{Time: event.ScheduledAt, Valid: true}
	}
	row, err := r.q.CreateEvent(ctx, sqlc_generated.CreateEventParams{
		MessageID:         event.MessageID,
		ChannelID:         event.ChannelID,
		CreatorID:         event.CreatorID,
		Title:             event.Title,
		Description:       event.Description,
		MaxSlots:          int32(event.MaxSlots),
		ScheduledAt:       scheduledAt,
		PrivateChannelID:  event.PrivateChannelID,
		QuestionsThreadID: event.QuestionsThreadID,
	})
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	event.ID = uint(row.ID)
	event.CreatedAt = pgtypeTimestamptzToTime(row.CreatedAt)
	event.UpdatedAt = pgtypeTimestamptzToTime(row.UpdatedAt)
	return nil
}

func (r *EventRepository) FindByMessageID(ctx context.Context, messageID string) (*entities.Event, error) {
	row, err := r.q.GetEventByMessageID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("get event by message id: %w", err)
	}
	e := eventToDomain(row)
	if err := r.attachParticipants(ctx, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventRepository) FindByID(ctx context.Context, id uint) (*entities.Event, error) {
	row, err := r.q.GetEventByID(ctx, int64(id))
	if err != nil {
		return nil, fmt.Errorf("get event by id: %w", err)
	}
	e := eventToDomain(row)
	if err := r.attachParticipants(ctx, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventRepository) FindByPrivateChannelID(ctx context.Context, privateChannelID string) (*entities.Event, error) {
	row, err := r.q.GetEventByPrivateChannelID(ctx, privateChannelID)
	if err != nil {
		return nil, fmt.Errorf("get event by private channel id: %w", err)
	}
	e := eventToDomain(row)
	if err := r.attachParticipants(ctx, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *EventRepository) attachParticipants(ctx context.Context, e *entities.Event) error {
	participants, err := r.q.GetParticipantsByEventID(ctx, int64(e.ID))
	if err != nil {
		return fmt.Errorf("get participants: %w", err)
	}
	e.Participants = make([]entities.Participant, len(participants))
	for i := range participants {
		e.Participants[i] = participantToDomain(participants[i])
	}
	return nil
}

func (r *EventRepository) FindByCreatorID(ctx context.Context, creatorID string) ([]entities.Event, error) {
	rows, err := r.q.GetEventsByCreatorID(ctx, creatorID)
	if err != nil {
		return nil, fmt.Errorf("get events by creator id: %w", err)
	}
	out := make([]entities.Event, len(rows))
	for i := range rows {
		out[i] = eventToDomain(rows[i])
	}
	return out, nil
}

func (r *EventRepository) FindEventsNeedingH48OrganizerDM(ctx context.Context, now time.Time) ([]entities.Event, error) {
	rows, err := r.q.FindEventsNeedingH48OrganizerDM(ctx, pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("find events needing H48 organizer DM: %w", err)
	}
	out := make([]entities.Event, len(rows))
	for i := range rows {
		out[i] = eventToDomain(rows[i])
	}
	return out, nil
}

func (r *EventRepository) FindStartedNonFinalizedEvents(ctx context.Context, now time.Time) ([]entities.Event, error) {
	rows, err := r.q.FindStartedNonFinalizedEvents(ctx, pgtype.Timestamptz{Time: now, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("find started non-finalized events: %w", err)
	}
	out := make([]entities.Event, len(rows))
	for i := range rows {
		out[i] = eventToDomain(rows[i])
	}
	return out, nil
}

func (r *EventRepository) MarkOrganizerValidationDMSent(ctx context.Context, eventID uint) error {
	if err := r.q.MarkOrganizerValidationDMSent(ctx, int64(eventID)); err != nil {
		return fmt.Errorf("mark organizer validation DM sent: %w", err)
	}
	return nil
}

func (r *EventRepository) MarkOrganizerStep1Finalized(ctx context.Context, eventID uint) error {
	if err := r.q.MarkOrganizerStep1Finalized(ctx, int64(eventID)); err != nil {
		return fmt.Errorf("mark organizer step1 finalized: %w", err)
	}
	return nil
}

func (r *EventRepository) Update(ctx context.Context, event *entities.Event) error {
	var scheduledAt pgtype.Timestamptz
	if !event.ScheduledAt.IsZero() {
		scheduledAt = pgtype.Timestamptz{Time: event.ScheduledAt, Valid: true}
	}
	err := r.q.UpdateEvent(ctx, sqlc_generated.UpdateEventParams{
		ID:          int64(event.ID),
		Title:       event.Title,
		Description: event.Description,
		MaxSlots:    int32(event.MaxSlots),
		ScheduledAt: scheduledAt,
	})
	if err != nil {
		return fmt.Errorf("update event: %w", err)
	}
	return nil
}

func (r *EventRepository) Delete(ctx context.Context, id uint) error {
	if err := r.q.DeleteEvent(ctx, int64(id)); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}
