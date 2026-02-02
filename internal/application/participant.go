package application

import (
	"context"
	"fmt"
	"time"

	"servbot/internal/domain"
	"servbot/internal/domain/entities"
	"servbot/internal/ports/output"
)

// ParticipantService implements input.ParticipantUseCase.
type ParticipantService struct {
	participantRepo output.ParticipantRepository
	eventRepo       output.EventRepository
}

// NewParticipantService creates a ParticipantService.
func NewParticipantService(
	participantRepo output.ParticipantRepository,
	eventRepo output.EventRepository,
) *ParticipantService {
	return &ParticipantService{
		participantRepo: participantRepo,
		eventRepo:       eventRepo,
	}
}

func (s *ParticipantService) JoinEvent(ctx context.Context, eventID uint, userID, username string) (string, error) {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return "", domain.ErrEventNotFound
	}
	existing, _ := s.participantRepo.FindByEventIDAndUserID(ctx, eventID, userID)
	if existing != nil {
		msg := "Tu es déjà inscrit !"
		if existing.Status == domain.StatusWaitlist {
			msg = "Tu es en liste d'attente."
		}
		return msg, domain.ErrParticipantExists
	}
	confirmedCount, err := s.participantRepo.CountByEventIDAndStatus(ctx, eventID, domain.StatusConfirmed)
	if err != nil {
		return "", fmt.Errorf("count confirmed: %w", err)
	}
	status := domain.StatusConfirmed
	reply := "✅ Tu es inscrit !"
	if event.MaxSlots > 0 && int(confirmedCount) >= event.MaxSlots {
		status = domain.StatusWaitlist
		reply = "⚠️ Complet ! Tu es en **liste d'attente**."
	}
	participant := &entities.Participant{
		EventID:  eventID,
		UserID:   userID,
		Username: username,
		Status:   status,
		JoinedAt: time.Now(),
	}
	if err := s.participantRepo.Create(ctx, participant); err != nil {
		return "", fmt.Errorf("create participant: %w", err)
	}
	return reply, nil
}

func (s *ParticipantService) LeaveEvent(ctx context.Context, eventID uint, userID string) (bool, error) {
	participant, err := s.participantRepo.FindByEventIDAndUserID(ctx, eventID, userID)
	if err != nil {
		return false, domain.ErrParticipantNotFound
	}
	wasConfirmed := participant.Status == domain.StatusConfirmed
	if err := s.participantRepo.Delete(ctx, participant); err != nil {
		return false, fmt.Errorf("delete participant: %w", err)
	}
	return wasConfirmed, nil
}

func (s *ParticipantService) PromoteParticipant(ctx context.Context, participantID uint, creatorID string) (*entities.Participant, error) {
	participant, err := s.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, domain.ErrParticipantNotFound
	}
	if participant.Status != domain.StatusWaitlist {
		return nil, domain.ErrParticipantNotWaitlist
	}
	event, err := s.eventRepo.FindByID(ctx, participant.EventID)
	if err != nil {
		return nil, domain.ErrEventNotFound
	}
	if event.CreatorID != creatorID {
		return nil, domain.ErrNotOrganizer
	}
	confirmedCount, err := s.participantRepo.CountByEventIDAndStatus(ctx, event.ID, domain.StatusConfirmed)
	if err != nil {
		return nil, fmt.Errorf("count confirmed: %w", err)
	}
	if event.MaxSlots > 0 && int(confirmedCount) >= event.MaxSlots {
		event.MaxSlots = int(confirmedCount) + 1
		if err := s.eventRepo.Update(ctx, event); err != nil {
			return nil, fmt.Errorf("update event: %w", err)
		}
	}
	participant.Status = domain.StatusConfirmed
	if err := s.participantRepo.Update(ctx, participant); err != nil {
		return nil, fmt.Errorf("update participant: %w", err)
	}
	return participant, nil
}

func (s *ParticipantService) RemoveParticipant(ctx context.Context, participantID uint, creatorID string) (*entities.Participant, error) {
	participant, err := s.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, domain.ErrParticipantNotFound
	}
	event, err := s.eventRepo.FindByID(ctx, participant.EventID)
	if err != nil {
		return nil, domain.ErrEventNotFound
	}
	if event.CreatorID != creatorID {
		return nil, domain.ErrNotOrganizer
	}
	if participant.Status != domain.StatusConfirmed {
		return nil, domain.ErrParticipantNotConfirmed
	}
	if err := s.participantRepo.Delete(ctx, participant); err != nil {
		return nil, fmt.Errorf("delete participant: %w", err)
	}
	return participant, nil
}

func (s *ParticipantService) GetNextWaitlistParticipant(ctx context.Context, eventID uint) (*entities.Participant, error) {
	participants, err := s.participantRepo.FindByEventIDAndStatus(ctx, eventID, domain.StatusWaitlist)
	if err != nil {
		return nil, fmt.Errorf("find waitlist: %w", err)
	}
	if len(participants) == 0 {
		return nil, domain.ErrNoWaitlistParticipant
	}
	oldest := participants[0]
	oldest.Status = domain.StatusConfirmed
	if err := s.participantRepo.Update(ctx, &oldest); err != nil {
		return nil, fmt.Errorf("update participant: %w", err)
	}
	return &oldest, nil
}
