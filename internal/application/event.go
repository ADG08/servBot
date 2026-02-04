package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"servbot/internal/domain"
	"servbot/internal/domain/entities"
	"servbot/internal/ports/output"
)

type EventService struct {
	eventRepo       output.EventRepository
	participantRepo output.ParticipantRepository
}

func NewEventService(
	eventRepo output.EventRepository,
	participantRepo output.ParticipantRepository,
) *EventService {
	return &EventService{
		eventRepo:       eventRepo,
		participantRepo: participantRepo,
	}
}

func (s *EventService) CreateEvent(ctx context.Context, event *entities.Event, creatorUsername string) error {
	if err := s.eventRepo.Create(ctx, event); err != nil {
		return err
	}
	username := strings.TrimSpace(creatorUsername)
	if username == "" {
		username = event.CreatorID
	}
	organizer := &entities.Participant{
		EventID:  event.ID,
		UserID:   event.CreatorID,
		Username: username,
		Status:   domain.StatusConfirmed,
		JoinedAt: time.Now(),
	}
	return s.participantRepo.Create(ctx, organizer)
}

func (s *EventService) GetEventByMessageID(ctx context.Context, messageID string) (*entities.Event, error) {
	return s.eventRepo.FindByMessageID(ctx, messageID)
}

func (s *EventService) GetEventByID(ctx context.Context, id uint) (*entities.Event, error) {
	return s.eventRepo.FindByID(ctx, id)
}

func (s *EventService) UpdateEvent(ctx context.Context, event *entities.Event) error {
	confirmedCount, err := s.participantRepo.CountByEventIDAndStatus(ctx, event.ID, domain.StatusConfirmed)
	if err != nil {
		return fmt.Errorf("count confirmed: %w", err)
	}
	if event.MaxSlots > 0 && int(confirmedCount) > event.MaxSlots {
		return domain.ErrCannotReduceSlots
	}
	return s.eventRepo.Update(ctx, event)
}

func (s *EventService) GetWaitlistParticipants(ctx context.Context, eventID uint) ([]entities.Participant, error) {
	return s.participantRepo.FindByEventIDAndStatus(ctx, eventID, domain.StatusWaitlist)
}

func (s *EventService) GetConfirmedParticipants(ctx context.Context, eventID uint) ([]entities.Participant, error) {
	return s.participantRepo.FindByEventIDAndStatus(ctx, eventID, domain.StatusConfirmed)
}

func (s *EventService) GetEventsByCreatorID(ctx context.Context, creatorID string) ([]entities.Event, error) {
	return s.eventRepo.FindByCreatorID(ctx, creatorID)
}
